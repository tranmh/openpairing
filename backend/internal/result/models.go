package result

import (
	"time"

	"github.com/google/uuid"
)

// GameResult represents a game result
type GameResult string

const (
	ResultWhiteWins     GameResult = "1-0"
	ResultBlackWins     GameResult = "0-1"
	ResultDraw          GameResult = "1/2-1/2"
	ResultWhiteForfeit  GameResult = "1-0F"
	ResultBlackForfeit  GameResult = "0-1F"
	ResultDoubleForfeit GameResult = "0-0F"
	ResultBye           GameResult = "BYE"
	ResultOngoing       GameResult = ""
)

// ResultEntry represents a result entry request
type ResultEntry struct {
	GameID    uuid.UUID  `json:"gameId" binding:"required"`
	Result    GameResult `json:"result" binding:"required,oneof=1-0 0-1 1/2-1/2 1-0F 0-1F 0-0F BYE"`
	EnteredBy uuid.UUID  `json:"enteredBy"`
	Notes     *string    `json:"notes,omitempty"`
}

// ResultCorrection represents a request to correct a result
type ResultCorrection struct {
	GameID      uuid.UUID  `json:"gameId" binding:"required"`
	OldResult   GameResult `json:"oldResult"`
	NewResult   GameResult `json:"newResult" binding:"required,oneof=1-0 0-1 1/2-1/2 1-0F 0-1F 0-0F BYE"`
	Reason      string     `json:"reason" binding:"required,min=10,max=500"`
	CorrectedBy uuid.UUID  `json:"correctedBy"`
}

// ResultAudit represents an audit log entry for result changes
type ResultAudit struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	GameID      uuid.UUID  `json:"gameId" db:"game_id"`
	Action      string     `json:"action" db:"action"` // "entered", "corrected", "deleted"
	OldResult   *string    `json:"oldResult,omitempty" db:"old_result"`
	NewResult   *string    `json:"newResult,omitempty" db:"new_result"`
	OldWhiteScore *float64 `json:"oldWhiteScore,omitempty" db:"old_white_score"`
	OldBlackScore *float64 `json:"oldBlackScore,omitempty" db:"old_black_score"`
	NewWhiteScore *float64 `json:"newWhiteScore,omitempty" db:"new_white_score"`
	NewBlackScore *float64 `json:"newBlackScore,omitempty" db:"new_black_score"`
	Reason      *string    `json:"reason,omitempty" db:"reason"`
	PerformedBy uuid.UUID  `json:"performedBy" db:"performed_by"`
	PerformedAt time.Time  `json:"performedAt" db:"performed_at"`
	IPAddress   *string    `json:"ipAddress,omitempty" db:"ip_address"`
}

// Game represents a game with full details
type Game struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	TournamentID       uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	RoundID            uuid.UUID  `json:"roundId" db:"round_id"`
	RoundNumber        int        `json:"roundNumber" db:"round_number"`
	BoardNumber        int        `json:"boardNumber" db:"board_number"`
	WhiteParticipantID *uuid.UUID `json:"whiteParticipantId,omitempty" db:"white_participant_id"`
	BlackParticipantID *uuid.UUID `json:"blackParticipantId,omitempty" db:"black_participant_id"`
	WhitePlayerName    *string    `json:"whitePlayerName,omitempty"`
	BlackPlayerName    *string    `json:"blackPlayerName,omitempty"`
	WhiteRating        *int       `json:"whiteRating,omitempty"`
	BlackRating        *int       `json:"blackRating,omitempty"`
	Result             *string    `json:"result,omitempty" db:"result"`
	WhiteScore         *float64   `json:"whiteScore,omitempty" db:"white_score"`
	BlackScore         *float64   `json:"blackScore,omitempty" db:"black_score"`
	Status             string     `json:"status" db:"status"`
	ResultEnteredBy    *uuid.UUID `json:"resultEnteredBy,omitempty" db:"result_entered_by"`
	ResultEnteredAt    *time.Time `json:"resultEnteredAt,omitempty" db:"result_entered_at"`
	PGN                *string    `json:"pgn,omitempty" db:"pgn"`
	CreatedAt          time.Time  `json:"createdAt" db:"created_at"`
}

// BulkResultEntry represents multiple results to enter at once
type BulkResultEntry struct {
	TournamentID uuid.UUID     `json:"tournamentId" binding:"required"`
	RoundNumber  int           `json:"roundNumber" binding:"required,min=1"`
	Results      []ResultEntry `json:"results" binding:"required,min=1,dive"`
}

// ResultEntryResponse represents the response after entering a result
type ResultEntryResponse struct {
	GameID          uuid.UUID `json:"gameId"`
	Result          string    `json:"result"`
	WhiteScore      float64   `json:"whiteScore"`
	BlackScore      float64   `json:"blackScore"`
	StandingsUpdate bool      `json:"standingsUpdate"`
}

// ResultValidation contains validation errors for a result
type ResultValidation struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// RoundStatus contains the result status for a round
type RoundStatus struct {
	RoundNumber     int   `json:"roundNumber"`
	TotalGames      int   `json:"totalGames"`
	CompletedGames  int   `json:"completedGames"`
	PendingGames    int   `json:"pendingGames"`
	OngoingGames    int   `json:"ongoingGames"`
	IsComplete      bool  `json:"isComplete"`
}

// ParseResult converts a result string to white and black scores
func ParseResult(result GameResult) (whiteScore, blackScore float64) {
	switch result {
	case ResultWhiteWins, ResultWhiteForfeit:
		return 1.0, 0.0
	case ResultBlackWins, ResultBlackForfeit:
		return 0.0, 1.0
	case ResultDraw:
		return 0.5, 0.5
	case ResultDoubleForfeit:
		return 0.0, 0.0
	case ResultBye:
		return 1.0, 0.0 // Full point bye by default
	default:
		return 0.0, 0.0
	}
}

// ValidResults returns all valid result values
func ValidResults() []GameResult {
	return []GameResult{
		ResultWhiteWins,
		ResultBlackWins,
		ResultDraw,
		ResultWhiteForfeit,
		ResultBlackForfeit,
		ResultDoubleForfeit,
		ResultBye,
	}
}

// IsValidResult checks if a result string is valid
func IsValidResult(result string) bool {
	for _, r := range ValidResults() {
		if string(r) == result {
			return true
		}
	}
	return false
}

// ResultDescription returns a human-readable description of a result
func ResultDescription(result GameResult) string {
	descriptions := map[GameResult]string{
		ResultWhiteWins:     "White wins",
		ResultBlackWins:     "Black wins",
		ResultDraw:          "Draw",
		ResultWhiteForfeit:  "White wins by forfeit",
		ResultBlackForfeit:  "Black wins by forfeit",
		ResultDoubleForfeit: "Double forfeit",
		ResultBye:           "Bye",
	}

	if desc, ok := descriptions[result]; ok {
		return desc
	}
	return "Unknown"
}
