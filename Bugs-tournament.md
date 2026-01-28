# Bug Report: tournament

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/backend/internal/tournament`
**Files Analyzed:** 13 files
**Bugs Found:** 11 bugs

---

## Summary

The tournament module contains critical race conditions in participant registration that could allow tournaments to exceed maximum capacity. There are also several data integrity issues with ignored JSON unmarshal errors, missing error propagation in score updates, and incomplete validation of game completion. Security concerns include potential division-by-zero in offset-to-page conversion and ignored authentication errors. Most issues center around concurrent access control and error handling gaps that could lead to data inconsistency.

---

## Bugs

## Bug #1: Critical Race Condition in Participant Registration (Max Capacity Bypass)

**Description:**
The `RegisterParticipant` function has a Time-of-Check-Time-of-Use (TOCTOU) race condition when validating tournament capacity. The code checks if the tournament is full by querying participants (lines 484-498), then adds the new participant (line 536). Between these operations, another concurrent request could add a participant, causing the tournament to exceed its maximum capacity limit.

**Location:**
- File: `backend/internal/tournament/service.go`
- Function: `RegisterParticipant`
- Lines: 484-498 (check), 536 (insert)

**Steps to Reproduce:**
1. Create a tournament with `max_players = 10`
2. Register 9 players
3. Send 2 concurrent registration requests (e.g., via parallel HTTP requests)
4. Expected: One request succeeds, one fails with "tournament is full"
5. Actual: Both requests can succeed, resulting in 11 players (exceeding limit)

**Fix:**
Move the capacity check into a database transaction with row-level locking, or use the atomic check already implemented in `AddParticipant` in the repository layer. The repository's `AddParticipant` already has proper locking (lines 442-461 in repository.go), but the service layer does a redundant check outside the transaction.

```diff
func (s *Service) RegisterParticipant(ctx context.Context, tournamentID uuid.UUID, req *RegisterParticipantRequest) (*Participant, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Check registration is open
	if tournament.Status != StatusPublished && tournament.Status != StatusRegistration {
		return nil, fmt.Errorf("registration is not open")
	}

-	// Check max players
-	if tournament.MaxPlayers != nil {
-		participants, err := s.repo.ListParticipants(ctx, tournamentID)
-		if err != nil {
-			return nil, err
-		}
-
-		activeCount := 0
-		for _, p := range participants {
-			if p.IsActive && p.RegistrationStatus != "withdrawn" {
-				activeCount++
-			}
-		}
-
-		if activeCount >= *tournament.MaxPlayers {
-			return nil, fmt.Errorf("tournament is full")
-		}
-	}
+	// Capacity check is handled atomically in AddParticipant with row-level locking

	// ... rest of function

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
+		if errors.Is(err, ErrTournamentFull) {
+			return nil, fmt.Errorf("tournament is full")
+		}
		return nil, fmt.Errorf("failed to register participant: %w", err)
	}
```

The repository layer already handles this correctly with `FOR UPDATE` locking, so the service layer should rely on it and handle the `ErrTournamentFull` error appropriately.

---

## Bug #2: Silent JSON Unmarshal Failures in Repository GetByID

**Description:**
In the `GetByID` function, JSON unmarshal errors for tournament settings (tiebreaks, settings, bye_settings, scoring_settings, round_robin_settings) are silently ignored (lines 222-232). If the database contains malformed JSON, the function will return a tournament with nil/empty settings without any indication that data was corrupted or invalid. This violates data integrity and makes debugging extremely difficult.

**Location:**
- File: `backend/internal/tournament/repository.go`
- Function: `GetByID`
- Lines: 222-232

**Steps to Reproduce:**
1. Manually insert a tournament with invalid JSON in the `bye_settings` column (e.g., `{"invalid": }`)
2. Call `GetByID` on that tournament
3. Expected: Function returns an error indicating corrupted data
4. Actual: Function succeeds, returning tournament with nil ByeSettings

**Fix:**
Check and propagate JSON unmarshal errors:

```diff
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	// ... query and scan code ...

-	json.Unmarshal(tiebreaksJSON, &t.Tiebreaks)
-	json.Unmarshal(settingsJSON, &t.Settings)
+	if err := json.Unmarshal(tiebreaksJSON, &t.Tiebreaks); err != nil {
+		return nil, fmt.Errorf("failed to unmarshal tiebreaks: %w", err)
+	}
+	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
+		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
+	}
	if byeSettingsJSON != nil {
-		json.Unmarshal(byeSettingsJSON, &t.ByeSettings)
+		if err := json.Unmarshal(byeSettingsJSON, &t.ByeSettings); err != nil {
+			return nil, fmt.Errorf("failed to unmarshal bye settings: %w", err)
+		}
	}
	if scoringSettingsJSON != nil {
-		json.Unmarshal(scoringSettingsJSON, &t.ScoringSettings)
+		if err := json.Unmarshal(scoringSettingsJSON, &t.ScoringSettings); err != nil {
+			return nil, fmt.Errorf("failed to unmarshal scoring settings: %w", err)
+		}
	}
	if roundRobinSettingsJSON != nil {
-		json.Unmarshal(roundRobinSettingsJSON, &t.RoundRobinSettings)
+		if err := json.Unmarshal(roundRobinSettingsJSON, &t.RoundRobinSettings); err != nil {
+			return nil, fmt.Errorf("failed to unmarshal round robin settings: %w", err)
+		}
	}

	return &t, nil
}
```

---

## Bug #3: Silent JSON Unmarshal Failures in Repository List

**Description:**
Similar to Bug #2, the `List` function ignores JSON unmarshal errors for all tournament settings (lines 405-412). This affects batch operations and tournament listings, potentially returning partial or corrupted data without error indication.

**Location:**
- File: `backend/internal/tournament/repository.go`
- Function: `List`
- Lines: 405-412

**Steps to Reproduce:**
1. Create multiple tournaments, with one having corrupted JSON settings
2. Call `List` to retrieve all tournaments
3. Expected: Function returns error or excludes corrupted tournament
4. Actual: Returns all tournaments, with corrupted one having nil settings

**Fix:**
Similar to Bug #2, add error checking for all JSON unmarshal operations:

```diff
func (r *PostgresRepository) List(ctx context.Context, filters *ListFilters) ([]Tournament, int64, error) {
	// ... query code ...

	for rows.Next() {
		var t Tournament
		var tiebreaksJSON, settingsJSON, byeSettingsJSON, scoringSettingsJSON []byte

		err := rows.Scan(/* ... */)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan tournament: %w", err)
		}

-		json.Unmarshal(tiebreaksJSON, &t.Tiebreaks)
-		json.Unmarshal(settingsJSON, &t.Settings)
+		if err := json.Unmarshal(tiebreaksJSON, &t.Tiebreaks); err != nil {
+			return nil, 0, fmt.Errorf("failed to unmarshal tiebreaks for tournament %s: %w", t.ID, err)
+		}
+		if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
+			return nil, 0, fmt.Errorf("failed to unmarshal settings for tournament %s: %w", t.ID, err)
+		}
		if byeSettingsJSON != nil {
-			json.Unmarshal(byeSettingsJSON, &t.ByeSettings)
+			if err := json.Unmarshal(byeSettingsJSON, &t.ByeSettings); err != nil {
+				return nil, 0, fmt.Errorf("failed to unmarshal bye settings for tournament %s: %w", t.ID, err)
+			}
		}
		if scoringSettingsJSON != nil {
-			json.Unmarshal(scoringSettingsJSON, &t.ScoringSettings)
+			if err := json.Unmarshal(scoringSettingsJSON, &t.ScoringSettings); err != nil {
+				return nil, 0, fmt.Errorf("failed to unmarshal scoring settings for tournament %s: %w", t.ID, err)
+			}
		}

		tournaments = append(tournaments, t)
	}

	return tournaments, total, nil
}
```

---

## Bug #4: Ignored Error in GetParticipantByPlayerID Check

**Description:**
In `RegisterParticipant` function (line 503), the error from `GetParticipantByPlayerID` is discarded with `_`. While the intent is to check if a player is already registered, this discards ALL errors including database connection failures, query timeouts, or other critical issues. Only the "not found" case should be ignored, while other errors should halt registration.

**Location:**
- File: `backend/internal/tournament/service.go`
- Function: `RegisterParticipant`
- Lines: 503-506

**Steps to Reproduce:**
1. Simulate a database connection failure or timeout
2. Attempt to register a participant
3. Expected: Registration fails with database error
4. Actual: Registration proceeds, potentially creating duplicate participants

**Fix:**
Distinguish between "not found" (acceptable) and other errors (critical):

```diff
func (s *Service) RegisterParticipant(ctx context.Context, tournamentID uuid.UUID, req *RegisterParticipantRequest) (*Participant, error) {
	// ... previous code ...

	// Check if player already registered
-	existing, _ := s.repo.GetParticipantByPlayerID(ctx, tournamentID, req.PlayerID)
-	if existing != nil {
-		return nil, fmt.Errorf("player already registered")
+	existing, err := s.repo.GetParticipantByPlayerID(ctx, tournamentID, req.PlayerID)
+	if err != nil && !errors.Is(err, ErrParticipantNotFound) {
+		return nil, fmt.Errorf("failed to check existing registration: %w", err)
+	}
+	if existing != nil {
+		return nil, ErrDuplicateRegistration
	}

	// ... rest of function ...
}
```

---

## Bug #5: BYE Score Update Failures Are Only Logged as Warnings

**Description:**
In `GeneratePairings` (lines 764-767), when updating participant scores for BYE games, errors are only logged as warnings and not propagated. This means if a database update fails, the participant will not receive their BYE points, causing incorrect standings, but the pairing generation will appear to succeed. This is a critical data integrity issue.

**Location:**
- File: `backend/internal/tournament/service.go`
- Function: `GeneratePairings`
- Lines: 764-767

**Steps to Reproduce:**
1. Generate pairings for a round with an odd number of players (creating a BYE)
2. Simulate a database failure during the `updateParticipantScores` call
3. Expected: Pairing generation fails with error
4. Actual: Pairing succeeds, but BYE recipient has incorrect points total

**Fix:**
Propagate the error instead of just logging it:

```diff
func (s *Service) GeneratePairings(ctx context.Context, tournamentID uuid.UUID, roundNumber int, userID uuid.UUID) ([]Game, error) {
	// ... pairing generation code ...

	// Update participant scores for BYE games
	for _, game := range games {
		if game.Result != nil && *game.Result == "BYE" {
			if err := s.updateParticipantScores(ctx, &game, scoring); err != nil {
-				s.logger.Warn().Err(err).Msg("Failed to update BYE participant score")
+				return nil, fmt.Errorf("failed to update BYE participant score: %w", err)
			}
		}
	}

	// ... rest of function ...
}
```

---

## Bug #6: Incomplete Game Completion Check in CompleteRound

**Description:**
The `CompleteRound` function checks if all games have results (lines 1096-1100), but does not verify that the game status is `GameStatusCompleted`. A game could have a result string but still be in `GameStatusOngoing` or `GameStatusScheduled` state due to a data entry error or race condition. This could allow a round to be marked complete with games still in progress.

**Location:**
- File: `backend/internal/tournament/service.go`
- Function: `CompleteRound`
- Lines: 1096-1100

**Steps to Reproduce:**
1. Start a round with games in progress
2. Manually set a result on a game without changing its status to "completed"
3. Call CompleteRound
4. Expected: Function rejects completion due to game not being completed
5. Actual: Round is marked complete despite game status being "ongoing"

**Fix:**
Check both result and status:

```diff
func (s *Service) CompleteRound(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*Round, error) {
	// ... previous code ...

	// Check all games have results
	games, err := s.repo.ListGamesByRound(ctx, round.ID)
	if err != nil {
		return nil, err
	}

	for _, g := range games {
-		if g.Result == nil || *g.Result == "" {
-			return nil, fmt.Errorf("game on board %d has no result", g.BoardNumber)
+		if g.Status != GameStatusCompleted {
+			return nil, fmt.Errorf("game on board %d is not completed (status: %s)", g.BoardNumber, g.Status)
+		}
+		if g.Result == nil || *g.Result == "" {
+			return nil, fmt.Errorf("game on board %d has no result despite being marked completed", g.BoardNumber)
		}
	}

	// ... rest of function ...
}
```

---

## Bug #7: Division by Zero Risk in Offset-to-Page Conversion

**Description:**
In the `ListTournaments` handler (line 326), the code converts offset to page number using `offset / filter.Limit`. While `filter.Limit` is initialized to 20 (line 311) and validated in the repository (lines 353-355), there's no explicit validation in the handler. If a malicious client passes `limit=0` in the query parameters (line 319-322), the division by zero will occur before repository validation, causing a runtime panic.

**Location:**
- File: `backend/internal/tournament/handler.go`
- Function: `ListTournaments`
- Lines: 326

**Steps to Reproduce:**
1. Send GET request to `/tournaments?limit=0&offset=10`
2. Expected: Returns validation error
3. Actual: Server panics with division by zero error

**Fix:**
Add explicit validation before division:

```diff
func (h *Handler) ListTournaments(c *gin.Context) {
	filter := &ListFilters{
		Limit: 20,
		Page:  0,
	}

	if status := c.Query("status"); status != "" {
		s := TournamentStatus(status)
		filter.Status = &s
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
+			if l <= 0 {
+				c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be greater than 0"})
+				return
+			}
			filter.Limit = l
		}
	}
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
+			if o < 0 {
+				c.JSON(http.StatusBadRequest, gin.H{"error": "offset cannot be negative"})
+				return
+			}
			filter.Page = o / filter.Limit // Convert offset to page
		}
	}

	// ... rest of function ...
}
```

---

## Bug #8: Ignored User ID Parse Errors in Handlers

**Description:**
In multiple handler functions (`GeneratePairings` line 649-650, `EnterResult` line 796-797), the user ID is parsed from the authentication context but parse errors are silently ignored using `_`. If the user ID is invalid or missing, `uuid.Nil` is used, which could lead to audit trail corruption or authorization bypass issues. Operations should fail if the authenticated user cannot be identified.

**Location:**
- File: `backend/internal/tournament/handler.go`
- Function: `GeneratePairings`, `EnterResult`
- Lines: 649-650, 796-797

**Steps to Reproduce:**
1. Bypass auth middleware or provide malformed user_id in context
2. Call GeneratePairings or EnterResult
3. Expected: Request fails with authentication error
4. Actual: Request proceeds with nil UUID, corrupting audit trails

**Fix:**
Properly handle parse errors:

```diff
func (h *Handler) GeneratePairings(c *gin.Context) {
	// ... previous code ...

	userIDStr, _ := c.Get("user_id")
-	arbiterID, _ := uuid.Parse(userIDStr.(string))
+	arbiterID, err := uuid.Parse(userIDStr.(string))
+	if err != nil {
+		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user authentication"})
+		return
+	}

	games, err := h.service.GeneratePairings(c.Request.Context(), id, round, arbiterID)
	// ... rest of function ...
}

func (h *Handler) EnterResult(c *gin.Context) {
	// ... previous code ...

	userIDStr, _ := c.Get("user_id")
-	arbiterID, _ := uuid.Parse(userIDStr.(string))
+	arbiterID, err := uuid.Parse(userIDStr.(string))
+	if err != nil {
+		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user authentication"})
+		return
+	}

	game, err := h.service.EnterResult(c.Request.Context(), gameID, &req, arbiterID)
	// ... rest of function ...
}
```

---

## Bug #9: Integer Division Precision Loss in FIDE Compliance Check

**Description:**
In `ValidateFIDECompliance` for `ByeSettings` (line 264), the code uses integer division `totalRounds / 2` to check if late entry max round exceeds half the tournament. For tournaments with odd rounds (e.g., 9 rounds), `9 / 2 = 4` in integer division, but FIDE actually means "first half" as rounds 1-5 (ceiling division). This causes incorrect FIDE violation warnings.

**Location:**
- File: `backend/internal/tournament/models.go`
- Function: `ValidateFIDECompliance` (on ByeSettings)
- Lines: 264-273

**Steps to Reproduce:**
1. Create tournament with 9 rounds
2. Set lateEntry.maxRound = 5 (which should be allowed in first half)
3. Call ValidateFIDECompliance
4. Expected: No violation (5 <= 5 ceiling)
5. Actual: Violation warning (5 > 4 floor)

**Fix:**
Use ceiling division for proper "first half" calculation:

```diff
func (s *ByeSettings) ValidateFIDECompliance(totalRounds int) []FIDEViolation {
	var violations []FIDEViolation

	// Late Entry violations
-	if s.LateEntry.MaxRound > totalRounds/2 {
+	firstHalf := (totalRounds + 1) / 2  // Ceiling division: 9 rounds -> 5, 10 rounds -> 5
+	if s.LateEntry.MaxRound > firstHalf {
		violations = append(violations, FIDEViolation{
			Setting:    "lateEntry.maxRound",
			Value:      s.LateEntry.MaxRound,
-			FIDEValue:  totalRounds / 2,
+			FIDEValue:  firstHalf,
			Severity:   "warning",
			Message:    "FIDE recommends late entry only in first half of tournament",
			Regulation: "FIDE Handbook C.04.2.B",
		})
	}
	// ... rest of function ...
}
```

---

## Bug #10: Floating Point Comparison for Win Detection

**Description:**
In `RecalculateTiebreaks` (line 1239), the code compares floating point scores directly using `==` operator: `*g.BlackScore == scoring.WinPoints`. Floating point arithmetic can introduce precision errors, causing this comparison to fail even when the values should be equal (e.g., `3.0` computed as `2.9999999999999996`). This can cause wins to not be counted correctly, affecting tiebreaker calculations.

**Location:**
- File: `backend/internal/tournament/service.go`
- Function: `RecalculateTiebreaks`
- Lines: 1239-1241

**Steps to Reproduce:**
1. Use football scoring (3-1-0 system)
2. Enter results that involve floating point arithmetic
3. Recalculate tiebreaks
4. Expected: Wins counted correctly
5. Actual: Some wins not detected due to floating point precision mismatch

**Fix:**
Use epsilon-based comparison or the existing helper functions:

```diff
func (s *Service) RecalculateTiebreaks(ctx context.Context, tournamentID uuid.UUID) error {
	// ... previous code ...

	for _, g := range games {
		// ... build result ...

		} else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID {
			if g.BlackScore != nil {
				result.Score = *g.BlackScore
			}
			result.PlayedWhite = false
			// Only count GamesAsBlack when there's an actual opponent
			if g.WhiteParticipantID != nil {
				playerData.GamesAsBlack++
-				if g.BlackScore != nil && *g.BlackScore == scoring.WinPoints {
+				if g.BlackScore != nil && IsWin(*g.BlackScore, scoring) {
					playerData.WinsAsBlack++
				}
				result.OpponentID = *g.WhiteParticipantID
				// ... rest ...
			}
		}
	}
	// ... rest of function ...
}
```

The `IsWin()` helper function already exists in `scoring.go` and should be used instead of direct comparison.

---

## Bug #11: Negative Participant Count Vulnerability

**Description:**
In `RemoveParticipant` (line 637-639), the participant count is decremented with `participant_count = participant_count - 1`. If due to a bug or database corruption the count is already 0 or somehow becomes negative, the decrement operation will create an invalid negative count. While PostgreSQL columns can be constrained, there's no validation in the code to prevent or detect this condition.

**Location:**
- File: `backend/internal/tournament/repository.go`
- Function: `RemoveParticipant`
- Lines: 637-639

**Steps to Reproduce:**
1. Manually set a tournament's participant_count to 0 in database
2. Remove a participant from that tournament
3. Expected: Error or validation failure
4. Actual: Participant count becomes -1

**Fix:**
Add validation to prevent negative counts:

```diff
func (r *PostgresRepository) RemoveParticipant(ctx context.Context, id uuid.UUID) error {
	// Start transaction to ensure participant delete and count update are atomic
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get tournament ID first (within transaction for consistency)
	var tournamentID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT tournament_id FROM tournament_participants WHERE id = $1", id).Scan(&tournamentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrParticipantNotFound
	}
	if err != nil {
		return err
	}

	result, err := tx.Exec(ctx, "DELETE FROM tournament_participants WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	// Update count within the same transaction
-	_, err = tx.Exec(ctx, `
-		UPDATE tournaments SET participant_count = participant_count - 1 WHERE id = $1
-	`, tournamentID)
+	result, err = tx.Exec(ctx, `
+		UPDATE tournaments
+		SET participant_count = GREATEST(0, participant_count - 1)
+		WHERE id = $1 AND participant_count > 0
+	`, tournamentID)
	if err != nil {
		return fmt.Errorf("failed to update participant count: %w", err)
	}
+	if result.RowsAffected() == 0 {
+		return fmt.Errorf("participant count already at 0 or tournament not found")
+	}

	return tx.Commit(ctx)
}
```

---

## Statistics

- **Critical:** 2 bugs (Race condition in registration, BYE score update failures)
- **High:** 4 bugs (JSON unmarshal failures, ignored error checks, incomplete game validation, floating point comparison)
- **Medium:** 4 bugs (Division by zero, auth errors ignored, FIDE compliance check, negative count)
- **Low:** 1 bug (Integer division precision)

---

## Recommendations

### Immediate Actions (Critical/High Priority)

1. **Fix Race Condition**: Remove the redundant capacity check in the service layer and rely on the repository's atomic transaction. This prevents tournament capacity overflow.

2. **Propagate All Errors**: Review all instances where errors are ignored (`_`, missing checks) and ensure proper error handling. Particularly critical for:
   - JSON unmarshal operations (data integrity)
   - Database query errors (reliability)
   - Authentication/authorization (security)

3. **Enforce Data Validation**: Add comprehensive validation at transaction boundaries:
   - Check game status before allowing round completion
   - Validate participant counts before operations
   - Use epsilon-based comparison for floating point values

### Medium-Term Improvements

4. **Add Integration Tests**: Create tests that simulate concurrent operations (parallel registrations, score updates during pairing) to catch race conditions.

5. **Implement Database Constraints**: Add CHECK constraints in PostgreSQL to enforce data integrity rules:
   - `participant_count >= 0`
   - Game result presence implies completed status
   - Tournament capacity constraints

6. **Audit Trail Enhancement**: Ensure all critical operations (pairings, results, withdrawals) properly record the authenticated user performing the action.

7. **Add Monitoring**: Implement alerts for:
   - JSON unmarshal failures
   - Participant count anomalies
   - Failed score updates

### Code Quality Improvements

8. **Consolidate Float Comparison**: Always use helper functions (`IsWin`, `IsDraw`, `IsLoss`) instead of direct equality checks for floating point values.

9. **Transaction Consistency**: Review all database operations that modify multiple tables and ensure they're wrapped in transactions with proper rollback handling.

10. **API Input Validation**: Add validation middleware to reject invalid query parameters (negative offsets, zero limits) before they reach handler logic.
