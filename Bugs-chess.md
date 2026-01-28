# Bug Report: chess

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/backend/internal/chess`
**Files Analyzed:** 4 files (game.go, manager.go, game_test.go, manager_test.go)
**Bugs Found:** 10 bugs

---

## Summary

The chess module handles real-time chess game logic and WebSocket game management. Analysis revealed **10 functional bugs** spanning multiple categories:

- **Critical Issues**: Memory leak in game cleanup (Bug #1), missing field initialization (Bug #2), authorization bypass (Bug #3)
- **High Priority**: Race condition potential (Bug #5), state machine issues (Bug #4, #7)
- **Medium Priority**: Validation gaps (Bug #6, #8, #9, #10)

The most severe issue is the memory leak in `cleanupCompletedGames()` which prevents users from creating new games after their old games are cleaned up. The authorization bypass in `Abort()` is a security vulnerability allowing any user to abort any game.

---

## Bugs

## Bug #1: Memory Leak - userGames Not Cleaned Up in cleanupCompletedGames

**Severity:** CRITICAL

**Description:**
The `cleanupCompletedGames()` function removes old games from the `games` map but fails to clean up corresponding entries in the `userGames` map. This causes a critical memory leak and prevents users from ever creating new games after their old games are automatically cleaned up.

When a game is completed and removed after 10 minutes (line 81 in manager.go), the `userGames` map still contains entries mapping both player UUIDs to the now-deleted game ID. Subsequently:
1. `GetUserGame()` returns `nil` because the game no longer exists in `gm.games`
2. `CreateGame()` fails with `ErrUserAlreadyInGame` because the stale entry exists in `userGames`
3. Users are permanently locked out of creating new games

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/manager.go`
- Function: `cleanupCompletedGames`
- Lines: 72-89

**Steps to Reproduce:**
1. Create a game between two users
2. Complete the game (resign, checkmate, etc.)
3. Wait 10+ minutes for automatic cleanup
4. Try to create a new game with either user
5. Expected: New game created successfully
6. Actual: Returns `ErrUserAlreadyInGame` even though `GetUserGame()` returns nil

**Fix:**
Clean up `userGames` entries before removing games from the `games` map:

```diff
func (gm *GameManager) cleanupCompletedGames() {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	removed := 0

	for id, game := range gm.games {
		if (game.Status == StatusCompleted || game.Status == StatusAborted) && game.CompletedAt != nil && game.CompletedAt.Before(cutoff) {
+			// Clean up userGames entries first
+			delete(gm.userGames, game.WhiteUserID)
+			delete(gm.userGames, game.BlackUserID)
			delete(gm.games, id)
			removed++
		}
	}

	if removed > 0 {
		gm.logger.Info().Int("removed", removed).Int("remaining", len(gm.games)).Msg("Cleaned up old games")
	}
}
```

---

## Bug #2: LoadGame Missing Critical Field Initialization

**Severity:** CRITICAL

**Description:**
The `LoadGame()` function only initializes a minimal subset of `GameState` fields (ID, FEN, Moves, MoveCount, game). Critical fields required for gameplay are left uninitialized with zero values:
- `WhiteUserID` and `BlackUserID` are zero UUIDs
- `WhiteClockMs`, `BlackClockMs`, `InitialTimeMs`, `IncrementMs` are all 0
- `Status` defaults to empty string (not a valid GameStatus)
- `lastClockUpdate` is zero time

If a loaded game is used for actual gameplay (started, moves made), it will cause panics, incorrect clock behavior, and authorization failures.

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/game.go`
- Function: `LoadGame`
- Lines: 96-117

**Steps to Reproduce:**
1. Load a game using `LoadGame()` with valid FEN and moves
2. Try to start the game with `Start()`
3. Try to make a move with a user ID
4. Expected: Game works normally with loaded state
5. Actual: Authorization check fails (userID never equals zero UUID), clock operations fail

**Fix:**
Add required parameters and initialize all fields:

```diff
-func LoadGame(id uuid.UUID, fen string, moves []string) (*GameState, error) {
+func LoadGame(id uuid.UUID, fen string, moves []string, whiteUserID, blackUserID uuid.UUID,
+	initialTimeMs, incrementMs, whiteClockMs, blackClockMs int64, status GameStatus) (*GameState, error) {
	fenOpt, err := chess.FEN(fen)
	if err != nil {
		return nil, fmt.Errorf("invalid FEN: %w", err)
	}
	g := chess.NewGame(fenOpt)

	// Replay moves
	for _, move := range moves {
		if err := g.MoveStr(move); err != nil {
			return nil, fmt.Errorf("invalid move %s: %w", move, err)
		}
	}

	return &GameState{
		ID:        id,
+		WhiteUserID: whiteUserID,
+		BlackUserID: blackUserID,
		FEN:       g.FEN(),
		Moves:     moves,
		MoveCount: len(moves),
+		WhiteClockMs: whiteClockMs,
+		BlackClockMs: blackClockMs,
+		InitialTimeMs: initialTimeMs,
+		IncrementMs: incrementMs,
+		Status: status,
		game:      g,
+		lastClockUpdate: time.Now(),
	}, nil
}
```

---

## Bug #3: Authorization Bypass - Abort Doesn't Verify User Is In Game

**Severity:** CRITICAL (Security Vulnerability)

**Description:**
The `Abort()` function accepts a `userID` parameter but never validates that the user is actually a player in the game. Any user can abort any game by calling this function with any user ID. This is a critical authorization vulnerability that allows unauthorized game manipulation.

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/game.go`
- Function: `Abort`
- Lines: 385-402

**Steps to Reproduce:**
1. User A and User B are playing a game
2. User C (not in the game) calls `Abort(gameID, randomUserID)`
3. Expected: Error "user not in game"
4. Actual: Game is aborted successfully

**Fix:**
Add authorization check similar to other game methods:

```diff
func (gs *GameState) Abort(userID uuid.UUID) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.Status != StatusOngoing && gs.Status != StatusWaiting {
		return fmt.Errorf("game cannot be aborted")
	}

+	if userID != gs.WhiteUserID && userID != gs.BlackUserID {
+		return fmt.Errorf("user not in game")
+	}

	if gs.MoveCount >= 2 {
		return fmt.Errorf("cannot abort after move 2")
	}

	gs.Status = StatusAborted
	gs.Termination = TerminationAborted
	now := time.Now()
	gs.CompletedAt = &now
	return nil
}
```

---

## Bug #4: Draw Offer State Machine Bug - Offers Can Be Lost

**Severity:** HIGH

**Description:**
The draw offer mechanism has a flawed state machine. The `DrawOfferedBy` field can only hold one player's offer at a time. If White offers a draw, then Black also offers a draw (before accepting), Black's offer overwrites White's offer. This creates confusion:
- White's original offer is lost
- White could now "accept" Black's offer (which was really White's own idea)
- The draw offer state doesn't represent mutual agreement properly

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/game.go`
- Function: `OfferDraw`
- Lines: 325-343

**Steps to Reproduce:**
1. White offers draw → `DrawOfferedBy = White`
2. Black also offers draw (instead of accepting) → Line 337 check passes (White != Black)
3. `DrawOfferedBy` is now set to Black → White's offer is lost
4. White can now call `AcceptDraw()` and it succeeds (line 358 check passes)
5. Expected: Both offers should be tracked or auto-accept on mutual offer
6. Actual: Offers overwrite each other, creating confusing state

**Fix:**
Auto-accept when the opponent also offers a draw (mutual agreement):

```diff
func (gs *GameState) OfferDraw(userID uuid.UUID) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.Status != StatusOngoing {
		return fmt.Errorf("game is not ongoing")
	}

	if userID != gs.WhiteUserID && userID != gs.BlackUserID {
		return fmt.Errorf("user not in game")
	}

	if gs.DrawOfferedBy != nil && *gs.DrawOfferedBy == userID {
		return fmt.Errorf("draw already offered")
	}

+	// If opponent has already offered draw, accept it (mutual agreement)
+	if gs.DrawOfferedBy != nil && *gs.DrawOfferedBy != userID {
+		gs.game.Draw(chess.DrawOffer)
+		gs.endGame("1/2-1/2", TerminationDrawAgreement)
+		return nil
+	}

	gs.DrawOfferedBy = &userID
	return nil
}
```

---

## Bug #5: Stop() Doesn't Wait for Goroutines to Finish

**Severity:** HIGH (Race Condition)

**Description:**
The `Stop()` method closes the stop channel and stops tickers but doesn't wait for the background goroutines (`clockLoop` and `cleanupLoop`) to actually terminate. If code accesses the GameManager immediately after calling `Stop()`, there are potential race conditions with goroutines that are still executing cleanup or clock update operations.

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/manager.go`
- Function: `Stop`
- Lines: 43-47

**Steps to Reproduce:**
1. Create GameManager with active games
2. Call `Stop()`
3. Immediately call `GetGame()` or manipulate games
4. Expected: Clean shutdown with no races
5. Actual: Potential race between cleanup goroutine and main thread access

**Fix:**
Use a WaitGroup to ensure goroutines have finished:

```diff
type GameManager struct {
	games         map[uuid.UUID]*GameState
	userGames     map[uuid.UUID]uuid.UUID
	mutex         sync.RWMutex
	logger        zerolog.Logger
	clockTicker   *time.Ticker
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
+	wg            sync.WaitGroup
}

func NewGameManager(logger zerolog.Logger) *GameManager {
	gm := &GameManager{
		games:     make(map[uuid.UUID]*GameState),
		userGames: make(map[uuid.UUID]uuid.UUID),
		logger:    logger.With().Str("component", "game_manager").Logger(),
		stopChan:  make(chan struct{}),
	}

	// Start clock update goroutine
	gm.clockTicker = time.NewTicker(100 * time.Millisecond)
+	gm.wg.Add(1)
	go gm.clockLoop()

	// Start cleanup goroutine to remove completed games after a delay
	gm.cleanupTicker = time.NewTicker(5 * time.Minute)
+	gm.wg.Add(1)
	go gm.cleanupLoop()

	return gm
}

func (gm *GameManager) Stop() {
	close(gm.stopChan)
	gm.clockTicker.Stop()
	gm.cleanupTicker.Stop()
+	gm.wg.Wait()
}

func (gm *GameManager) clockLoop() {
+	defer gm.wg.Done()
	for {
		select {
		case <-gm.clockTicker.C:
			gm.updateClocks()
		case <-gm.stopChan:
			return
		}
	}
}

func (gm *GameManager) cleanupLoop() {
+	defer gm.wg.Done()
	for {
		select {
		case <-gm.cleanupTicker.C:
			gm.cleanupCompletedGames()
		case <-gm.stopChan:
			return
		}
	}
}
```

---

## Bug #6: GetState Returns Mutable Slice Reference

**Severity:** HIGH (Data Integrity)

**Description:**
The `GetState()` method returns a `GameStateSnapshot` containing the `Moves` slice directly from the internal game state (line 447). This slice is not copied, meaning callers can modify the returned slice and corrupt the internal game state. This violates the snapshot pattern and can lead to subtle data corruption bugs.

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/game.go`
- Function: `GetState`
- Lines: 438-458

**Steps to Reproduce:**
1. Create and start a game, make some moves
2. Call `snapshot := game.GetState()`
3. Modify `snapshot.Moves`: `snapshot.Moves[0] = "corrupted"`
4. Call `game.GetState()` again
5. Expected: Original moves preserved
6. Actual: Moves slice is corrupted in the internal game state

**Fix:**
Return a copy of the Moves slice:

```diff
func (gs *GameState) GetState() GameStateSnapshot {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

+	// Create a copy of the moves slice to prevent external modification
+	movesCopy := make([]string, len(gs.Moves))
+	copy(movesCopy, gs.Moves)

	return GameStateSnapshot{
		ID:            gs.ID,
		WhiteUserID:   gs.WhiteUserID,
		BlackUserID:   gs.BlackUserID,
		FEN:           gs.FEN,
-		Moves:         gs.Moves,
+		Moves:         movesCopy,
		MoveCount:     gs.MoveCount,
		WhiteClockMs:  gs.WhiteClockMs,
		BlackClockMs:  gs.BlackClockMs,
		Status:        gs.Status,
		Result:        gs.Result,
		Termination:   gs.Termination,
		Turn:          gs.getTurn(),
		InCheck:       gs.isInCheck(),
		DrawOfferedBy: gs.DrawOfferedBy,
	}
}
```

---

## Bug #7: DeclineDraw Doesn't Check If Game Is Ongoing

**Severity:** MEDIUM

**Description:**
The `DeclineDraw()` function doesn't validate that the game is still ongoing. A draw offer can be declined even after the game has ended (completed or aborted). This allows meaningless state changes on completed games and could lead to confusion or edge case bugs.

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/game.go`
- Function: `DeclineDraw`
- Lines: 368-382

**Steps to Reproduce:**
1. Start a game, White offers draw
2. Black resigns (game ends)
3. Call `DeclineDraw()` with Black's user ID
4. Expected: Error "game is not ongoing"
5. Actual: Draw offer is cleared successfully on a completed game

**Fix:**
Add status check like other game methods:

```diff
func (gs *GameState) DeclineDraw(userID uuid.UUID) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

+	if gs.Status != StatusOngoing {
+		return fmt.Errorf("game is not ongoing")
+	}

	if gs.DrawOfferedBy == nil {
		return fmt.Errorf("no draw offer pending")
	}

	if *gs.DrawOfferedBy == userID {
		return fmt.Errorf("cannot decline your own draw offer")
	}

	gs.DrawOfferedBy = nil
	return nil
}
```

---

## Bug #8: No Validation That White And Black Users Are Different

**Severity:** MEDIUM

**Description:**
The `CreateGame()` function doesn't validate that `whiteUserID` and `blackUserID` are different. A user can create a game where they play against themselves. While this might be intentional for training purposes, it's more likely an oversight that could lead to confusion, billing issues, or rating calculation problems.

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/manager.go`
- Function: `CreateGame`
- Lines: 117-143

**Steps to Reproduce:**
1. Call `CreateGame(userID, userID, 300000, 0)` with the same user ID
2. Expected: Error "cannot play against yourself"
3. Actual: Game is created successfully with same user as both players

**Fix:**
Add validation check:

```diff
func (gm *GameManager) CreateGame(whiteUserID, blackUserID uuid.UUID, initialTimeMs, incrementMs int64) (*GameState, error) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

+	// Validate that white and black are different users
+	if whiteUserID == blackUserID {
+		return nil, &GameError{"cannot create game with same user as both players"}
+	}

	// Check if either user is already in a game
	if _, exists := gm.userGames[whiteUserID]; exists {
		return nil, ErrUserAlreadyInGame
	}
	if _, exists := gm.userGames[blackUserID]; exists {
		return nil, ErrUserAlreadyInGame
	}

	game := NewGame(whiteUserID, blackUserID, initialTimeMs, incrementMs)
	gm.games[game.ID] = game
	gm.userGames[whiteUserID] = game.ID
	gm.userGames[blackUserID] = game.ID

	gm.logger.Info().
		Str("game_id", game.ID.String()).
		Str("white", whiteUserID.String()).
		Str("black", blackUserID.String()).
		Int64("time_ms", initialTimeMs).
		Int64("increment_ms", incrementMs).
		Msg("Game created")

	return game, nil
}
```

---

## Bug #9: No Validation For Negative Time Controls

**Severity:** MEDIUM

**Description:**
The `NewGame()` and `CreateGame()` functions don't validate that `initialTimeMs` and `incrementMs` are non-negative. Negative time values could cause undefined behavior in clock calculations, potentially causing games to end immediately or exhibit strange timeout behavior.

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/game.go`
- Function: `NewGame`
- Lines: 75-93
- File: `/home/user/openpairing/backend/internal/chess/manager.go`
- Function: `CreateGame`
- Lines: 117-143

**Steps to Reproduce:**
1. Call `NewGame(white, black, -1000, -500)` with negative time values
2. Start the game
3. Expected: Validation error or panic
4. Actual: Game created with negative clock values, may cause immediate timeout or overflow

**Fix:**
Add validation in both functions:

```diff
func NewGame(whiteUserID, blackUserID uuid.UUID, initialTimeMs, incrementMs int64) *GameState {
+	if initialTimeMs < 0 {
+		initialTimeMs = 0
+	}
+	if incrementMs < 0 {
+		incrementMs = 0
+	}

	g := chess.NewGame()

	return &GameState{
		ID:            uuid.New(),
		WhiteUserID:   whiteUserID,
		BlackUserID:   blackUserID,
		FEN:           g.FEN(),
		Moves:         []string{},
		MoveCount:     0,
		WhiteClockMs:  initialTimeMs,
		BlackClockMs:  initialTimeMs,
		InitialTimeMs: initialTimeMs,
		IncrementMs:   incrementMs,
		Status:        StatusWaiting,
		game:          g,
		lastClockUpdate: time.Now(),
	}
}
```

And in CreateGame:

```diff
func (gm *GameManager) CreateGame(whiteUserID, blackUserID uuid.UUID, initialTimeMs, incrementMs int64) (*GameState, error) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

+	// Validate time controls
+	if initialTimeMs < 0 {
+		return nil, &GameError{"initial time must be non-negative"}
+	}
+	if incrementMs < 0 {
+		return nil, &GameError{"increment must be non-negative"}
+	}

	// Check if either user is already in a game
	if _, exists := gm.userGames[whiteUserID]; exists {
		return nil, ErrUserAlreadyInGame
	}
	if _, exists := gm.userGames[blackUserID]; exists {
		return nil, ErrUserAlreadyInGame
	}

	game := NewGame(whiteUserID, blackUserID, initialTimeMs, incrementMs)
	// ... rest of function
}
```

---

## Bug #10: OfferDraw Allows Offering Draw In Non-Ongoing Game States

**Severity:** LOW

**Description:**
While `OfferDraw()` checks if the game is ongoing (line 329), a draw could theoretically be offered in a `StatusWaiting` game (before it starts). While this is caught by the status check, the error message could be more specific, and the function should explicitly reject offers in waiting, completed, or aborted states.

**Location:**
- File: `/home/user/openpairing/backend/internal/chess/game.go`
- Function: `OfferDraw`
- Lines: 325-343

**Steps to Reproduce:**
1. Create a game (status = waiting)
2. Try to offer draw before starting
3. Expected: Clear error about game not started
4. Actual: Generic "game is not ongoing" error

**Fix:**
More explicit state validation:

```diff
func (gs *GameState) OfferDraw(userID uuid.UUID) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.Status != StatusOngoing {
-		return fmt.Errorf("game is not ongoing")
+		switch gs.Status {
+		case StatusWaiting:
+			return fmt.Errorf("game has not started yet")
+		case StatusCompleted:
+			return fmt.Errorf("game has already ended")
+		case StatusAborted:
+			return fmt.Errorf("game was aborted")
+		default:
+			return fmt.Errorf("game is not ongoing")
+		}
	}

	if userID != gs.WhiteUserID && userID != gs.BlackUserID {
		return fmt.Errorf("user not in game")
	}

	if gs.DrawOfferedBy != nil && *gs.DrawOfferedBy == userID {
		return fmt.Errorf("draw already offered")
	}

	gs.DrawOfferedBy = &userID
	return nil
}
```

---

## Statistics

- **Critical:** 3 bugs (#1, #2, #3)
- **High:** 3 bugs (#4, #5, #6)
- **Medium:** 3 bugs (#7, #8, #9)
- **Low:** 1 bug (#10)

---

## Recommendations

### Immediate Actions Required:
1. **Fix Bug #1 immediately** - The memory leak will cause production issues within hours as users become unable to create new games
2. **Fix Bug #3** - Security vulnerability allowing unauthorized game manipulation
3. **Fix Bug #2** - LoadGame is currently unusable for any real-world scenario

### Code Quality Improvements:
1. **Add comprehensive input validation** - Validate all user inputs, time controls, and user IDs at entry points
2. **Improve state machine clarity** - Document valid state transitions and add validation
3. **Add integration tests** - Test concurrent access patterns, cleanup behaviors, and edge cases
4. **Consider using sync.WaitGroup pattern** - Ensure clean shutdown of background goroutines
5. **Implement defensive copying** - Return copies of mutable data structures in read operations

### Architecture Considerations:
1. Consider separating concerns between GameState (chess logic) and GameSession (user/clock management)
2. Add telemetry for cleanup operations to detect memory leaks in production
3. Consider adding a "game archive" system separate from active game management
4. Implement rate limiting on game creation to prevent abuse
5. Add audit logging for security-sensitive operations (abort, resign)

### Testing Gaps:
1. No tests for LoadGame with actual gameplay
2. No tests for concurrent cleanup and game creation
3. No tests for memory leak scenarios
4. No tests for authorization checks in Abort
5. Missing negative test cases for time control validation
