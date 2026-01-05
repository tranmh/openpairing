package gameaction

import (
	"time"

	"github.com/google/uuid"
)

// GameStatus represents the status of an online chess game
type GameStatus string

const (
	StatusWaiting   GameStatus = "waiting"
	StatusOngoing   GameStatus = "ongoing"
	StatusCompleted GameStatus = "completed"
	StatusAborted   GameStatus = "aborted"
)

// Termination represents how a game ended
type Termination string

const (
	TermCheckmate      Termination = "checkmate"
	TermResignation    Termination = "resignation"
	TermTimeout        Termination = "timeout"
	TermDrawAgreement  Termination = "draw_agreement"
	TermStalemate      Termination = "stalemate"
	TermAborted        Termination = "aborted"
	TermArbiterDecision Termination = "arbiter_decision"
)

// ChessGame represents an online chess game
type ChessGame struct {
	ID           uuid.UUID   `json:"id" db:"id"`
	WhiteUserID  *uuid.UUID  `json:"whiteUserId,omitempty" db:"white_user_id"`
	BlackUserID  *uuid.UUID  `json:"blackUserId,omitempty" db:"black_user_id"`
	FEN          string      `json:"fen" db:"fen"`
	Moves        string      `json:"moves" db:"moves"`
	MoveCount    int         `json:"moveCount" db:"move_count"`
	WhiteClockMs *int        `json:"whiteClockMs,omitempty" db:"white_clock_ms"`
	BlackClockMs *int        `json:"blackClockMs,omitempty" db:"black_clock_ms"`
	InitialTimeMs *int       `json:"initialTimeMs,omitempty" db:"initial_time_ms"`
	IncrementMs  *int        `json:"incrementMs,omitempty" db:"increment_ms"`
	Status       GameStatus  `json:"status" db:"status"`
	Result       *string     `json:"result,omitempty" db:"result"`
	Termination  *Termination `json:"termination,omitempty" db:"termination"`
	StartedAt    *time.Time  `json:"startedAt,omitempty" db:"started_at"`
	LastMoveAt   *time.Time  `json:"lastMoveAt,omitempty" db:"last_move_at"`
	CompletedAt  *time.Time  `json:"completedAt,omitempty" db:"completed_at"`
	CreatedAt    time.Time   `json:"createdAt" db:"created_at"`
}

// GameWithPlayers includes player information
type GameWithPlayers struct {
	ChessGame
	WhitePlayerName string `json:"whitePlayerName,omitempty" db:"white_player_name"`
	BlackPlayerName string `json:"blackPlayerName,omitempty" db:"black_player_name"`
	WhiteRating     *int   `json:"whiteRating,omitempty" db:"white_rating"`
	BlackRating     *int   `json:"blackRating,omitempty" db:"black_rating"`
}

// DrawOffer represents a draw offer state
type DrawOffer struct {
	GameID     uuid.UUID `json:"gameId"`
	OfferedBy  uuid.UUID `json:"offeredBy"`
	OfferedAt  time.Time `json:"offeredAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// ArbiterRequest represents a request for arbiter intervention
type ArbiterRequest struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	GameID      uuid.UUID  `json:"gameId" db:"game_id"`
	RequestedBy uuid.UUID  `json:"requestedBy" db:"requested_by"`
	Reason      string     `json:"reason" db:"reason"`
	Status      string     `json:"status" db:"status"` // pending, resolved, dismissed
	ResolvedBy  *uuid.UUID `json:"resolvedBy,omitempty" db:"resolved_by"`
	Resolution  *string    `json:"resolution,omitempty" db:"resolution"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty" db:"resolved_at"`
}

// ResignRequest is the request to resign from a game
type ResignRequest struct {
	GameID uuid.UUID `json:"gameId" binding:"required"`
}

// DrawOfferRequest is the request to offer or respond to a draw
type DrawOfferRequest struct {
	GameID uuid.UUID `json:"gameId" binding:"required"`
	Accept *bool     `json:"accept,omitempty"` // If responding to an offer
}

// AbortRequest is the request to abort a game
type AbortRequest struct {
	GameID uuid.UUID `json:"gameId" binding:"required"`
	Reason string    `json:"reason,omitempty" binding:"max=500"`
}

// CallArbiterRequest is the request to call an arbiter
type CallArbiterRequest struct {
	GameID uuid.UUID `json:"gameId" binding:"required"`
	Reason string    `json:"reason" binding:"required,min=10,max=500"`
}

// GameActionResponse is the response for game actions
type GameActionResponse struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Game       *ChessGame  `json:"game,omitempty"`
	NewStatus  *GameStatus `json:"newStatus,omitempty"`
	Result     *string     `json:"result,omitempty"`
}
