package team_tournament

import (
	"time"

	"github.com/google/uuid"
)

// TeamTournamentSettings contains team tournament specific configuration
type TeamTournamentSettings struct {
	TournamentID                   uuid.UUID `json:"tournamentId" db:"tournament_id"`
	BoardsPerMatch                 int       `json:"boardsPerMatch" db:"boards_per_match"`
	LineupSubmissionDeadlineMinutes int       `json:"lineupSubmissionDeadlineMinutes" db:"lineup_submission_deadline_minutes"`
	AllowLateLineup                bool      `json:"allowLateLineup" db:"allow_late_lineup"`
	EnforceRatingOrder             bool      `json:"enforceRatingOrder" db:"enforce_rating_order"`
	RatingOrderTolerance           int       `json:"ratingOrderTolerance" db:"rating_order_tolerance"`
	MatchWinPoints                 float64   `json:"matchWinPoints" db:"match_win_points"`
	MatchDrawPoints                float64   `json:"matchDrawPoints" db:"match_draw_points"`
	MatchLossPoints                float64   `json:"matchLossPoints" db:"match_loss_points"`
	BoardWinPoints                 float64   `json:"boardWinPoints" db:"board_win_points"`
	BoardDrawPoints                float64   `json:"boardDrawPoints" db:"board_draw_points"`
	BoardLossPoints                float64   `json:"boardLossPoints" db:"board_loss_points"`
	DefaultBoardResult             string    `json:"defaultBoardResult" db:"default_board_result"` // loss, draw, none
	TeamTiebreaks                  []string  `json:"teamTiebreaks" db:"team_tiebreaks"`
}

// DefaultTeamTournamentSettings returns default settings for team tournaments
func DefaultTeamTournamentSettings() *TeamTournamentSettings {
	return &TeamTournamentSettings{
		BoardsPerMatch:                 8,
		LineupSubmissionDeadlineMinutes: 30,
		AllowLateLineup:                false,
		EnforceRatingOrder:             true,
		RatingOrderTolerance:           0,
		MatchWinPoints:                 2.0,
		MatchDrawPoints:                1.0,
		MatchLossPoints:                0.0,
		BoardWinPoints:                 1.0,
		BoardDrawPoints:                0.5,
		BoardLossPoints:                0.0,
		DefaultBoardResult:             "loss",
		TeamTiebreaks:                  []string{"board_points", "buchholz", "direct_encounter"},
	}
}

// TeamParticipant represents a team registered for a team tournament
type TeamParticipant struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	TournamentID       uuid.UUID `json:"tournamentId" db:"tournament_id"`
	TeamID             uuid.UUID `json:"teamId" db:"team_id"`
	RegistrationStatus string    `json:"registrationStatus" db:"registration_status"`
	RegisteredAt       time.Time `json:"registeredAt" db:"registered_at"`
	StartRank          int       `json:"startRank" db:"start_rank"`
	SeedRating         int       `json:"seedRating" db:"seed_rating"`
	MatchPoints        float64   `json:"matchPoints" db:"match_points"`
	BoardPoints        float64   `json:"boardPoints" db:"board_points"`
	Buchholz           float64   `json:"buchholz" db:"buchholz"`
	SonnebornBerger    float64   `json:"sonnebornBerger" db:"sonneborn_berger"`
	LineupLocked       bool      `json:"lineupLocked" db:"lineup_locked"`

	// Joined data
	TeamName      string `json:"teamName,omitempty" db:"team_name"`
	TeamShortName string `json:"teamShortName,omitempty" db:"team_short_name"`
	ClubName      string `json:"clubName,omitempty" db:"club_name"`
}

// LineupEntry represents a player assignment in a team lineup
type LineupEntry struct {
	Board    int       `json:"board"`
	PlayerID uuid.UUID `json:"playerId"`
	Rating   int       `json:"rating"`
	Name     string    `json:"name,omitempty"`
}

// TeamLineup represents a team's lineup for a specific round
type TeamLineup struct {
	ID                uuid.UUID     `json:"id" db:"id"`
	TournamentID      uuid.UUID     `json:"tournamentId" db:"tournament_id"`
	RoundID           uuid.UUID     `json:"roundId" db:"round_id"`
	TeamParticipantID uuid.UUID     `json:"teamParticipantId" db:"team_participant_id"`
	Lineup            []LineupEntry `json:"lineup" db:"lineup"`
	IsValid           bool          `json:"isValid" db:"is_valid"`
	ValidationErrors  []string      `json:"validationErrors,omitempty" db:"validation_errors"`
	SubmittedAt       *time.Time    `json:"submittedAt,omitempty" db:"submitted_at"`
	SubmittedBy       *uuid.UUID    `json:"submittedBy,omitempty" db:"submitted_by"`
	IsLocked          bool          `json:"isLocked" db:"is_locked"`
}

// TeamMatch represents a match between two teams in a round
type TeamMatch struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TournamentID    uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	RoundID         uuid.UUID  `json:"roundId" db:"round_id"`
	MatchNumber     int        `json:"matchNumber" db:"match_number"`
	HomeTeamID      *uuid.UUID `json:"homeTeamId,omitempty" db:"home_team_id"`
	AwayTeamID      *uuid.UUID `json:"awayTeamId,omitempty" db:"away_team_id"`
	HomeMatchPoints *float64   `json:"homeMatchPoints,omitempty" db:"home_match_points"`
	AwayMatchPoints *float64   `json:"awayMatchPoints,omitempty" db:"away_match_points"`
	HomeBoardPoints *float64   `json:"homeBoardPoints,omitempty" db:"home_board_points"`
	AwayBoardPoints *float64   `json:"awayBoardPoints,omitempty" db:"away_board_points"`
	Status          string     `json:"status" db:"status"` // scheduled, ongoing, completed
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`

	// Joined data
	HomeTeamName string `json:"homeTeamName,omitempty" db:"home_team_name"`
	AwayTeamName string `json:"awayTeamName,omitempty" db:"away_team_name"`
}

// MatchStatus constants
const (
	MatchStatusScheduled = "scheduled"
	MatchStatusOngoing   = "ongoing"
	MatchStatusCompleted = "completed"
)

// TeamMatchBoard represents an individual board game within a team match
type TeamMatchBoard struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TeamMatchID   uuid.UUID  `json:"teamMatchId" db:"team_match_id"`
	GameID        *uuid.UUID `json:"gameId,omitempty" db:"game_id"`
	BoardNumber   int        `json:"boardNumber" db:"board_number"`
	HomePlayerID  *uuid.UUID `json:"homePlayerId,omitempty" db:"home_player_id"`
	AwayPlayerID  *uuid.UUID `json:"awayPlayerId,omitempty" db:"away_player_id"`
	HomeRating    int        `json:"homeRating" db:"home_rating"`
	AwayRating    int        `json:"awayRating" db:"away_rating"`
	HomeScore     *float64   `json:"homeScore,omitempty" db:"home_score"`
	AwayScore     *float64   `json:"awayScore,omitempty" db:"away_score"`
	Result        *string    `json:"result,omitempty" db:"result"`

	// Joined data
	HomePlayerName string `json:"homePlayerName,omitempty" db:"home_player_name"`
	AwayPlayerName string `json:"awayPlayerName,omitempty" db:"away_player_name"`
}

// TeamStanding represents a team's position in the standings
type TeamStanding struct {
	Rank            int       `json:"rank"`
	TeamParticipant TeamParticipant `json:"teamParticipant"`
	MatchesPlayed   int       `json:"matchesPlayed"`
	MatchWins       int       `json:"matchWins"`
	MatchDraws      int       `json:"matchDraws"`
	MatchLosses     int       `json:"matchLosses"`
	BoardsPlayed    int       `json:"boardsPlayed"`
	BoardWins       int       `json:"boardWins"`
	BoardDraws      int       `json:"boardDraws"`
	BoardLosses     int       `json:"boardLosses"`
}

// Request/Response types

// RegisterTeamRequest is the request to register a team for a tournament
type RegisterTeamRequest struct {
	TournamentID uuid.UUID `json:"tournamentId"`
	TeamID       uuid.UUID `json:"teamId"`
}

// SubmitLineupRequest is the request to submit a team lineup
type SubmitLineupRequest struct {
	TournamentID      uuid.UUID     `json:"tournamentId"`
	RoundNumber       int           `json:"roundNumber"`
	TeamParticipantID uuid.UUID     `json:"teamParticipantId"`
	Lineup            []LineupEntry `json:"lineup"`
}

// EnterBoardResultRequest is the request to enter a board result
type EnterBoardResultRequest struct {
	TeamMatchID uuid.UUID `json:"teamMatchId"`
	BoardNumber int       `json:"boardNumber"`
	Result      string    `json:"result"` // 1-0, 0-1, 1/2-1/2, 1-0F, 0-1F
}

// TeamTournamentStandingsResponse is the response for team standings
type TeamTournamentStandingsResponse struct {
	TournamentID uuid.UUID        `json:"tournamentId"`
	Standings    []TeamStanding   `json:"standings"`
	Tiebreaks    []string         `json:"tiebreaks"`
}

// TeamMatchResponse wraps a team match with board details
type TeamMatchResponse struct {
	Match  TeamMatch        `json:"match"`
	Boards []TeamMatchBoard `json:"boards"`
}

// LineupValidationError represents a lineup validation error
type LineupValidationError struct {
	Board   int    `json:"board,omitempty"`
	Message string `json:"message"`
}
