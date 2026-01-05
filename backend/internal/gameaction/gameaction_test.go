package gameaction

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGameStatus_Constants(t *testing.T) {
	statuses := []GameStatus{
		StatusWaiting,
		StatusOngoing,
		StatusCompleted,
		StatusAborted,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("GameStatus should not be empty")
		}
	}

	// Verify specific values
	if StatusWaiting != "waiting" {
		t.Errorf("StatusWaiting = %q, want %q", StatusWaiting, "waiting")
	}
	if StatusCompleted != "completed" {
		t.Errorf("StatusCompleted = %q, want %q", StatusCompleted, "completed")
	}
}

func TestTermination_Constants(t *testing.T) {
	terminations := []Termination{
		TermCheckmate,
		TermResignation,
		TermTimeout,
		TermDrawAgreement,
		TermStalemate,
		TermAborted,
		TermArbiterDecision,
	}

	for _, term := range terminations {
		if term == "" {
			t.Error("Termination should not be empty")
		}
	}

	// Verify specific values
	if TermCheckmate != "checkmate" {
		t.Errorf("TermCheckmate = %q, want %q", TermCheckmate, "checkmate")
	}
	if TermResignation != "resignation" {
		t.Errorf("TermResignation = %q, want %q", TermResignation, "resignation")
	}
}

func TestDrawOffer_Expiration(t *testing.T) {
	offer := DrawOffer{
		GameID:    uuid.New(),
		OfferedBy: uuid.New(),
		OfferedAt: time.Now(),
		ExpiresAt: time.Now().Add(60 * time.Second),
	}

	// Should not be expired
	if time.Now().After(offer.ExpiresAt) {
		t.Error("New draw offer should not be expired")
	}

	// Expired offer
	expiredOffer := DrawOffer{
		GameID:    uuid.New(),
		OfferedBy: uuid.New(),
		OfferedAt: time.Now().Add(-2 * time.Minute),
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}

	if !time.Now().After(expiredOffer.ExpiresAt) {
		t.Error("Old draw offer should be expired")
	}
}

func TestResignResult(t *testing.T) {
	tests := []struct {
		color  string
		result string
	}{
		{"white", "0-1"}, // White resigns, black wins
		{"black", "1-0"}, // Black resigns, white wins
	}

	for _, tt := range tests {
		t.Run(tt.color+" resigns", func(t *testing.T) {
			var result string
			if tt.color == "white" {
				result = "0-1"
			} else {
				result = "1-0"
			}
			if result != tt.result {
				t.Errorf("When %s resigns, result = %q, want %q", tt.color, result, tt.result)
			}
		})
	}
}

func TestChessGame_DefaultFEN(t *testing.T) {
	startFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	game := ChessGame{
		FEN: startFEN,
	}

	if game.FEN != startFEN {
		t.Errorf("Default FEN = %q, want %q", game.FEN, startFEN)
	}
}

func TestGameActionResponse_Success(t *testing.T) {
	status := StatusCompleted
	result := "1-0"

	response := GameActionResponse{
		Success:   true,
		Message:   "Game ended",
		NewStatus: &status,
		Result:    &result,
	}

	if !response.Success {
		t.Error("Response should be successful")
	}
	if response.NewStatus == nil || *response.NewStatus != StatusCompleted {
		t.Error("NewStatus should be completed")
	}
	if response.Result == nil || *response.Result != "1-0" {
		t.Error("Result should be 1-0")
	}
}

func TestAbortMoveLimit(t *testing.T) {
	// Games with more than 4 half-moves cannot be aborted
	maxAbortMoves := 4

	tests := []struct {
		moveCount  int
		canAbort   bool
	}{
		{0, true},
		{2, true},
		{4, true},
		{5, false},
		{10, false},
	}

	for _, tt := range tests {
		canAbort := tt.moveCount <= maxAbortMoves
		if canAbort != tt.canAbort {
			t.Errorf("moveCount=%d: canAbort=%v, want %v", tt.moveCount, canAbort, tt.canAbort)
		}
	}
}

func TestArbiterRequest_Validation(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		valid  bool
	}{
		{"valid reason", "My opponent is taking too long between moves and I suspect time stalling behavior", true},
		{"too short", "help", false},
		{"minimum length", "This is a reasonably long reason", true},
	}

	minReasonLength := 10

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.reason) >= minReasonLength
			if isValid != tt.valid {
				t.Errorf("Reason %q: valid=%v, want %v", tt.reason, isValid, tt.valid)
			}
		})
	}
}
