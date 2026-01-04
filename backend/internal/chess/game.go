package chess

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/notnil/chess"
)

// GameState represents the current state of a chess game
type GameState struct {
	ID          uuid.UUID  `json:"id"`
	WhiteUserID uuid.UUID  `json:"whiteUserId"`
	BlackUserID uuid.UUID  `json:"blackUserId"`

	// Game state
	FEN       string   `json:"fen"`
	Moves     []string `json:"moves"` // UCI format
	MoveCount int      `json:"moveCount"`

	// Clock state (milliseconds)
	WhiteClockMs   int64 `json:"whiteClockMs"`
	BlackClockMs   int64 `json:"blackClockMs"`
	InitialTimeMs  int64 `json:"initialTimeMs"`
	IncrementMs    int64 `json:"incrementMs"`

	// Status
	Status      GameStatus  `json:"status"`
	Result      string      `json:"result"`
	Termination Termination `json:"termination"`

	// Timestamps
	StartedAt   *time.Time `json:"startedAt"`
	LastMoveAt  *time.Time `json:"lastMoveAt"`
	CompletedAt *time.Time `json:"completedAt"`

	// Draw offer
	DrawOfferedBy *uuid.UUID `json:"drawOfferedBy"`

	// Internal
	game        *chess.Game
	mutex       sync.RWMutex
	lastClockUpdate time.Time
}

// GameStatus represents the game status
type GameStatus string

const (
	StatusWaiting   GameStatus = "waiting"
	StatusOngoing   GameStatus = "ongoing"
	StatusCompleted GameStatus = "completed"
	StatusAborted   GameStatus = "aborted"
)

// Termination represents how the game ended
type Termination string

const (
	TerminationCheckmate         Termination = "checkmate"
	TerminationResignation       Termination = "resignation"
	TerminationTimeout           Termination = "timeout"
	TerminationDrawAgreement     Termination = "draw_agreement"
	TerminationStalemate         Termination = "stalemate"
	TerminationInsufficientMaterial Termination = "insufficient_material"
	TerminationThreefoldRepetition Termination = "threefold_repetition"
	TerminationFiftyMoveRule     Termination = "fifty_move_rule"
	TerminationAborted           Termination = "aborted"
)

// NewGame creates a new chess game
func NewGame(whiteUserID, blackUserID uuid.UUID, initialTimeMs, incrementMs int64) *GameState {
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

// LoadGame loads a game from a FEN position and moves
func LoadGame(id uuid.UUID, fen string, moves []string) (*GameState, error) {
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
		FEN:       g.FEN(),
		Moves:     moves,
		MoveCount: len(moves),
		game:      g,
	}, nil
}

// Start starts the game
func (gs *GameState) Start() {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	now := time.Now()
	gs.Status = StatusOngoing
	gs.StartedAt = &now
	gs.lastClockUpdate = now
}

// MakeMove makes a move in UCI format (e.g., "e2e4", "e7e8q")
func (gs *GameState) MakeMove(userID uuid.UUID, moveUCI string) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.Status != StatusOngoing {
		return fmt.Errorf("game is not ongoing")
	}

	// Check if it's the user's turn
	turn := gs.game.Position().Turn()
	if turn == chess.White && userID != gs.WhiteUserID {
		return fmt.Errorf("not white's turn")
	}
	if turn == chess.Black && userID != gs.BlackUserID {
		return fmt.Errorf("not black's turn")
	}

	// Update clock for the player who just moved
	now := time.Now()
	elapsed := now.Sub(gs.lastClockUpdate).Milliseconds()

	if turn == chess.White {
		gs.WhiteClockMs -= elapsed
		if gs.WhiteClockMs <= 0 {
			gs.WhiteClockMs = 0
			gs.endGame("0-1", TerminationTimeout)
			return nil
		}
		gs.WhiteClockMs += gs.IncrementMs
	} else {
		gs.BlackClockMs -= elapsed
		if gs.BlackClockMs <= 0 {
			gs.BlackClockMs = 0
			gs.endGame("1-0", TerminationTimeout)
			return nil
		}
		gs.BlackClockMs += gs.IncrementMs
	}

	// Parse and validate move
	move, err := parseUCIMove(gs.game, moveUCI)
	if err != nil {
		return fmt.Errorf("invalid move: %w", err)
	}

	// Make the move
	if err := gs.game.Move(move); err != nil {
		return fmt.Errorf("illegal move: %w", err)
	}

	gs.Moves = append(gs.Moves, moveUCI)
	gs.MoveCount++
	gs.FEN = gs.game.FEN()
	gs.LastMoveAt = &now
	gs.lastClockUpdate = now
	gs.DrawOfferedBy = nil // Clear draw offer after move

	// Check for game end
	gs.checkGameEnd()

	return nil
}

// squareFromString parses a square string like "e2" to a chess.Square
func squareFromString(s string) chess.Square {
	if len(s) != 2 {
		return chess.NoSquare
	}
	file := chess.File(s[0] - 'a')
	rank := chess.Rank(s[1] - '1')
	if file < chess.FileA || file > chess.FileH || rank < chess.Rank1 || rank > chess.Rank8 {
		return chess.NoSquare
	}
	return chess.NewSquare(file, rank)
}

func parseUCIMove(g *chess.Game, uci string) (*chess.Move, error) {
	if len(uci) < 4 || len(uci) > 5 {
		return nil, fmt.Errorf("invalid UCI format")
	}

	from := squareFromString(uci[0:2])
	to := squareFromString(uci[2:4])

	var promo chess.PieceType = chess.NoPieceType
	if len(uci) == 5 {
		switch strings.ToLower(string(uci[4])) {
		case "q":
			promo = chess.Queen
		case "r":
			promo = chess.Rook
		case "b":
			promo = chess.Bishop
		case "n":
			promo = chess.Knight
		default:
			return nil, fmt.Errorf("invalid promotion piece")
		}
	}

	// Find the matching legal move
	for _, move := range g.ValidMoves() {
		if move.S1() == from && move.S2() == to {
			if promo != chess.NoPieceType {
				if move.Promo() == promo {
					return move, nil
				}
			} else if move.Promo() == chess.NoPieceType {
				return move, nil
			}
		}
	}

	return nil, fmt.Errorf("move not legal")
}

func (gs *GameState) checkGameEnd() {
	outcome := gs.game.Outcome()

	switch outcome {
	case chess.WhiteWon:
		method := gs.game.Method()
		termination := TerminationCheckmate
		if method == chess.Resignation {
			termination = TerminationResignation
		}
		gs.endGame("1-0", termination)

	case chess.BlackWon:
		method := gs.game.Method()
		termination := TerminationCheckmate
		if method == chess.Resignation {
			termination = TerminationResignation
		}
		gs.endGame("0-1", termination)

	case chess.Draw:
		method := gs.game.Method()
		var termination Termination
		switch method {
		case chess.Stalemate:
			termination = TerminationStalemate
		case chess.InsufficientMaterial:
			termination = TerminationInsufficientMaterial
		case chess.ThreefoldRepetition:
			termination = TerminationThreefoldRepetition
		case chess.FiftyMoveRule:
			termination = TerminationFiftyMoveRule
		default:
			termination = TerminationDrawAgreement
		}
		gs.endGame("1/2-1/2", termination)
	}
}

func (gs *GameState) endGame(result string, termination Termination) {
	now := time.Now()
	gs.Status = StatusCompleted
	gs.Result = result
	gs.Termination = termination
	gs.CompletedAt = &now
}

// Resign resigns the game
func (gs *GameState) Resign(userID uuid.UUID) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.Status != StatusOngoing {
		return fmt.Errorf("game is not ongoing")
	}

	var result string
	if userID == gs.WhiteUserID {
		result = "0-1"
	} else if userID == gs.BlackUserID {
		result = "1-0"
	} else {
		return fmt.Errorf("user not in game")
	}

	gs.game.Resign(gs.colorForUser(userID))
	gs.endGame(result, TerminationResignation)
	return nil
}

func (gs *GameState) colorForUser(userID uuid.UUID) chess.Color {
	if userID == gs.WhiteUserID {
		return chess.White
	}
	return chess.Black
}

// OfferDraw offers a draw
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

	gs.DrawOfferedBy = &userID
	return nil
}

// AcceptDraw accepts a draw offer
func (gs *GameState) AcceptDraw(userID uuid.UUID) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.Status != StatusOngoing {
		return fmt.Errorf("game is not ongoing")
	}

	if gs.DrawOfferedBy == nil {
		return fmt.Errorf("no draw offer pending")
	}

	if *gs.DrawOfferedBy == userID {
		return fmt.Errorf("cannot accept your own draw offer")
	}

	gs.game.Draw(chess.DrawOffer)
	gs.endGame("1/2-1/2", TerminationDrawAgreement)
	return nil
}

// DeclineDraw declines a draw offer
func (gs *GameState) DeclineDraw(userID uuid.UUID) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.DrawOfferedBy == nil {
		return fmt.Errorf("no draw offer pending")
	}

	if *gs.DrawOfferedBy == userID {
		return fmt.Errorf("cannot decline your own draw offer")
	}

	gs.DrawOfferedBy = nil
	return nil
}

// Abort aborts the game (only before move 2)
func (gs *GameState) Abort(userID uuid.UUID) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.Status != StatusOngoing && gs.Status != StatusWaiting {
		return fmt.Errorf("game cannot be aborted")
	}

	if gs.MoveCount >= 2 {
		return fmt.Errorf("cannot abort after move 2")
	}

	gs.Status = StatusAborted
	gs.Termination = TerminationAborted
	now := time.Now()
	gs.CompletedAt = &now
	return nil
}

// UpdateClock updates the clock for the current player
func (gs *GameState) UpdateClock() (whiteMs, blackMs int64, expired bool) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.Status != StatusOngoing {
		return gs.WhiteClockMs, gs.BlackClockMs, false
	}

	now := time.Now()
	elapsed := now.Sub(gs.lastClockUpdate).Milliseconds()

	turn := gs.game.Position().Turn()
	if turn == chess.White {
		gs.WhiteClockMs -= elapsed
		if gs.WhiteClockMs <= 0 {
			gs.WhiteClockMs = 0
			gs.endGame("0-1", TerminationTimeout)
			return gs.WhiteClockMs, gs.BlackClockMs, true
		}
	} else {
		gs.BlackClockMs -= elapsed
		if gs.BlackClockMs <= 0 {
			gs.BlackClockMs = 0
			gs.endGame("1-0", TerminationTimeout)
			return gs.WhiteClockMs, gs.BlackClockMs, true
		}
	}

	gs.lastClockUpdate = now
	return gs.WhiteClockMs, gs.BlackClockMs, false
}

// GetState returns a snapshot of the current game state
func (gs *GameState) GetState() GameStateSnapshot {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	return GameStateSnapshot{
		ID:            gs.ID,
		WhiteUserID:   gs.WhiteUserID,
		BlackUserID:   gs.BlackUserID,
		FEN:           gs.FEN,
		Moves:         gs.Moves,
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

// isInCheck determines if the current side to move is in check
// by checking if there are no valid moves that don't result from check evasion
func (gs *GameState) isInCheck() bool {
	// If we have moves, check if the last move has the Check tag
	moves := gs.game.Moves()
	if len(moves) > 0 {
		lastMove := moves[len(moves)-1]
		// Check if the last move was a check
		for _, tag := range []chess.MoveTag{chess.Check} {
			if lastMove.HasTag(tag) {
				return true
			}
		}
	}
	return false
}

func (gs *GameState) getTurn() string {
	if gs.game.Position().Turn() == chess.White {
		return "white"
	}
	return "black"
}

// GameStateSnapshot is a read-only snapshot of the game state
type GameStateSnapshot struct {
	ID            uuid.UUID   `json:"id"`
	WhiteUserID   uuid.UUID   `json:"whiteUserId"`
	BlackUserID   uuid.UUID   `json:"blackUserId"`
	FEN           string      `json:"fen"`
	Moves         []string    `json:"moves"`
	MoveCount     int         `json:"moveCount"`
	WhiteClockMs  int64       `json:"whiteClockMs"`
	BlackClockMs  int64       `json:"blackClockMs"`
	Status        GameStatus  `json:"status"`
	Result        string      `json:"result"`
	Termination   Termination `json:"termination"`
	Turn          string      `json:"turn"`
	InCheck       bool        `json:"inCheck"`
	DrawOfferedBy *uuid.UUID  `json:"drawOfferedBy"`
}

// GetPGN returns the game in PGN format
func (gs *GameState) GetPGN() string {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	return gs.game.String()
}

// IsUserInGame checks if a user is a player in the game
func (gs *GameState) IsUserInGame(userID uuid.UUID) bool {
	return userID == gs.WhiteUserID || userID == gs.BlackUserID
}

// GetOpponent returns the opponent's user ID
func (gs *GameState) GetOpponent(userID uuid.UUID) (uuid.UUID, error) {
	if userID == gs.WhiteUserID {
		return gs.BlackUserID, nil
	}
	if userID == gs.BlackUserID {
		return gs.WhiteUserID, nil
	}
	return uuid.Nil, fmt.Errorf("user not in game")
}

// ValidMoves returns all valid moves for the current position
func (gs *GameState) ValidMoves() []string {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	moves := gs.game.ValidMoves()
	uciMoves := make([]string, len(moves))
	for i, m := range moves {
		uci := m.S1().String() + m.S2().String()
		if m.Promo() != chess.NoPieceType {
			uci += strings.ToLower(m.Promo().String())
		}
		uciMoves[i] = uci
	}
	return uciMoves
}
