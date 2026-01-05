package standings

import (
	"time"

	"github.com/google/uuid"
)

// Standing represents a player's position in the standings
type Standing struct {
	Rank             int       `json:"rank"`
	ParticipantID    uuid.UUID `json:"participantId"`
	PlayerID         uuid.UUID `json:"playerId"`
	PlayerName       string    `json:"playerName"`
	PlayerFideID     *string   `json:"playerFideId,omitempty"`
	PlayerTitle      *string   `json:"playerTitle,omitempty"`
	ClubName         *string   `json:"clubName,omitempty"`
	Rating           int       `json:"rating"`
	Points           float64   `json:"points"`
	GamesPlayed      int       `json:"gamesPlayed"`
	Wins             int       `json:"wins"`
	Draws            int       `json:"draws"`
	Losses           int       `json:"losses"`
	Tiebreaks        map[string]float64 `json:"tiebreaks"`
	IsActive         bool      `json:"isActive"`
	PerformanceRating *int     `json:"performanceRating,omitempty"`
}

// StandingsResponse represents the full standings for a tournament
type StandingsResponse struct {
	TournamentID     uuid.UUID  `json:"tournamentId"`
	TournamentName   string     `json:"tournamentName"`
	CurrentRound     int        `json:"currentRound"`
	TotalRounds      int        `json:"totalRounds"`
	LastUpdated      time.Time  `json:"lastUpdated"`
	Standings        []Standing `json:"standings"`
	TiebreakOrder    []string   `json:"tiebreakOrder"`
}

// CrosstableEntry represents a single cell in the crosstable
type CrosstableEntry struct {
	OpponentRank int     `json:"opponentRank"`
	Result       *string `json:"result,omitempty"` // "1", "0", "=", "+", "-", null
	Color        string  `json:"color"`            // "w", "b", "-"
}

// CrosstableRow represents one player's row in the crosstable
type CrosstableRow struct {
	Rank       int               `json:"rank"`
	PlayerName string            `json:"playerName"`
	Rating     int               `json:"rating"`
	Results    []CrosstableEntry `json:"results"` // Indexed by round (0 = round 1)
	Points     float64           `json:"points"`
}

// CrosstableResponse represents a tournament crosstable
type CrosstableResponse struct {
	TournamentID   uuid.UUID       `json:"tournamentId"`
	TournamentName string          `json:"tournamentName"`
	CurrentRound   int             `json:"currentRound"`
	TotalRounds    int             `json:"totalRounds"`
	Rows           []CrosstableRow `json:"rows"`
}

// RoundStanding represents standings after a specific round
type RoundStanding struct {
	RoundNumber int        `json:"roundNumber"`
	Standings   []Standing `json:"standings"`
	Timestamp   time.Time  `json:"timestamp"`
}

// StandingsHistory represents historical standings progression
type StandingsHistory struct {
	TournamentID uuid.UUID       `json:"tournamentId"`
	Rounds       []RoundStanding `json:"rounds"`
}

// PlayerProgress represents a player's progress through the tournament
type PlayerProgress struct {
	ParticipantID uuid.UUID `json:"participantId"`
	PlayerName    string    `json:"playerName"`
	RoundPoints   []float64 `json:"roundPoints"`      // Cumulative points after each round
	RoundRanks    []int     `json:"roundRanks"`       // Rank after each round
	Opponents     []struct {
		RoundNumber int     `json:"roundNumber"`
		Name        string  `json:"name"`
		Rating      int     `json:"rating"`
		Result      string  `json:"result"`
		Color       string  `json:"color"`
	} `json:"opponents"`
}

// TiebreakInfo contains information about a tiebreak system
type TiebreakInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PerformanceData contains data for performance rating calculation
type PerformanceData struct {
	TotalOpponentRating int
	OpponentCount       int
	Score               float64
	GamesPlayed         int
}
