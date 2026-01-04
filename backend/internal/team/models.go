package team

import (
	"time"

	"github.com/google/uuid"
)

// Team represents a chess team (e.g., club team for Bundesliga)
type Team struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	ClubID    *uuid.UUID `json:"clubId,omitempty" db:"club_id"`
	Name      string     `json:"name" db:"name"`
	ShortName string     `json:"shortName,omitempty" db:"short_name"`
	Division  string     `json:"division,omitempty" db:"division"`
	Region    string     `json:"region,omitempty" db:"region"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
}

// TeamMember represents a player assigned to a team roster
type TeamMember struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TeamID      uuid.UUID  `json:"teamId" db:"team_id"`
	PlayerID    uuid.UUID  `json:"playerId" db:"player_id"`
	BoardNumber *int       `json:"boardNumber,omitempty" db:"board_number"`
	IsCaptain   bool       `json:"isCaptain" db:"is_captain"`
	IsReserve   bool       `json:"isReserve" db:"is_reserve"`
	ValidFrom   *time.Time `json:"validFrom,omitempty" db:"valid_from"`
	ValidUntil  *time.Time `json:"validUntil,omitempty" db:"valid_until"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
}

// TeamMemberWithPlayer includes player details
type TeamMemberWithPlayer struct {
	TeamMember
	PlayerName   string `json:"playerName"`
	PlayerRating int    `json:"playerRating"`
	PlayerTitle  string `json:"playerTitle,omitempty"`
}

// TeamTournamentParticipant represents a team registered for a tournament
type TeamTournamentParticipant struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	TournamentID       uuid.UUID `json:"tournamentId" db:"tournament_id"`
	TeamID             uuid.UUID `json:"teamId" db:"team_id"`
	RegistrationStatus string    `json:"registrationStatus" db:"registration_status"`
	RegisteredAt       time.Time `json:"registeredAt" db:"registered_at"`
	StartRank          *int      `json:"startRank,omitempty" db:"start_rank"`
	SeedRating         *int      `json:"seedRating,omitempty" db:"seed_rating"`
	MatchPoints        float64   `json:"matchPoints" db:"match_points"`
	BoardPoints        float64   `json:"boardPoints" db:"board_points"`
	Buchholz           float64   `json:"buchholz" db:"buchholz"`
	LineupLocked       bool      `json:"lineupLocked" db:"lineup_locked"`
}

// TeamTournamentParticipantWithTeam includes team details
type TeamTournamentParticipantWithTeam struct {
	TeamTournamentParticipant
	TeamName      string `json:"teamName"`
	TeamShortName string `json:"teamShortName,omitempty"`
}

// TeamLineup represents a team's lineup for a specific round
type TeamLineup struct {
	ID                uuid.UUID `json:"id" db:"id"`
	TournamentID      uuid.UUID `json:"tournamentId" db:"tournament_id"`
	RoundID           uuid.UUID `json:"roundId" db:"round_id"`
	TeamParticipantID uuid.UUID `json:"teamParticipantId" db:"team_participant_id"`
	Lineup            []LineupEntry `json:"lineup" db:"lineup"`
	IsValid           bool      `json:"isValid" db:"is_valid"`
	ValidationErrors  []string  `json:"validationErrors,omitempty" db:"validation_errors"`
	SubmittedAt       *time.Time `json:"submittedAt,omitempty" db:"submitted_at"`
	SubmittedBy       *uuid.UUID `json:"submittedBy,omitempty" db:"submitted_by"`
}

// LineupEntry represents a single board assignment in a lineup
type LineupEntry struct {
	Board    int       `json:"board"`
	PlayerID uuid.UUID `json:"playerId"`
	Rating   int       `json:"rating"`
}

// TeamMatch represents a match between two teams
type TeamMatch struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TournamentID     uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	RoundID          uuid.UUID  `json:"roundId" db:"round_id"`
	MatchNumber      int        `json:"matchNumber" db:"match_number"`
	HomeTeamID       *uuid.UUID `json:"homeTeamId,omitempty" db:"home_team_id"`
	AwayTeamID       *uuid.UUID `json:"awayTeamId,omitempty" db:"away_team_id"`
	HomeMatchPoints  *float64   `json:"homeMatchPoints,omitempty" db:"home_match_points"`
	AwayMatchPoints  *float64   `json:"awayMatchPoints,omitempty" db:"away_match_points"`
	HomeBoardPoints  *float64   `json:"homeBoardPoints,omitempty" db:"home_board_points"`
	AwayBoardPoints  *float64   `json:"awayBoardPoints,omitempty" db:"away_board_points"`
	Status           string     `json:"status" db:"status"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
}

// TeamMatchWithTeams includes team details
type TeamMatchWithTeams struct {
	TeamMatch
	HomeTeamName string `json:"homeTeamName,omitempty"`
	AwayTeamName string `json:"awayTeamName,omitempty"`
}

// TeamMatchBoard represents an individual board pairing within a team match
type TeamMatchBoard struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TeamMatchID  uuid.UUID  `json:"teamMatchId" db:"team_match_id"`
	GameID       *uuid.UUID `json:"gameId,omitempty" db:"game_id"`
	BoardNumber  int        `json:"boardNumber" db:"board_number"`
	HomePlayerID *uuid.UUID `json:"homePlayerId,omitempty" db:"home_player_id"`
	AwayPlayerID *uuid.UUID `json:"awayPlayerId,omitempty" db:"away_player_id"`
	HomeRating   *int       `json:"homeRating,omitempty" db:"home_rating"`
	AwayRating   *int       `json:"awayRating,omitempty" db:"away_rating"`
	HomeScore    *float64   `json:"homeScore,omitempty" db:"home_score"`
	AwayScore    *float64   `json:"awayScore,omitempty" db:"away_score"`
}

// TeamMatchBoardWithPlayers includes player details
type TeamMatchBoardWithPlayers struct {
	TeamMatchBoard
	HomePlayerName string `json:"homePlayerName,omitempty"`
	AwayPlayerName string `json:"awayPlayerName,omitempty"`
}

// TeamTournamentSettings contains settings specific to team tournaments
type TeamTournamentSettings struct {
	TournamentID                    uuid.UUID `json:"tournamentId" db:"tournament_id"`
	BoardsPerMatch                  int       `json:"boardsPerMatch" db:"boards_per_match"`
	LineupSubmissionDeadlineMinutes int       `json:"lineupSubmissionDeadlineMinutes" db:"lineup_submission_deadline_minutes"`
	AllowLateLineup                 bool      `json:"allowLateLineup" db:"allow_late_lineup"`
	EnforceRatingOrder              bool      `json:"enforceRatingOrder" db:"enforce_rating_order"`
	RatingOrderTolerance            int       `json:"ratingOrderTolerance" db:"rating_order_tolerance"`
	MatchWinPoints                  float64   `json:"matchWinPoints" db:"match_win_points"`
	MatchDrawPoints                 float64   `json:"matchDrawPoints" db:"match_draw_points"`
	MatchLossPoints                 float64   `json:"matchLossPoints" db:"match_loss_points"`
	TeamTiebreaks                   []string  `json:"teamTiebreaks" db:"team_tiebreaks"`
	BoardWinPoints                  float64   `json:"boardWinPoints" db:"board_win_points"`
	BoardDrawPoints                 float64   `json:"boardDrawPoints" db:"board_draw_points"`
	BoardLossPoints                 float64   `json:"boardLossPoints" db:"board_loss_points"`
	DefaultBoardResult              string    `json:"defaultBoardResult" db:"default_board_result"`
}

// DefaultTeamTournamentSettings returns default settings for team tournaments
func DefaultTeamTournamentSettings(tournamentID uuid.UUID) *TeamTournamentSettings {
	return &TeamTournamentSettings{
		TournamentID:                    tournamentID,
		BoardsPerMatch:                  8,
		LineupSubmissionDeadlineMinutes: 30,
		AllowLateLineup:                 false,
		EnforceRatingOrder:              true,
		RatingOrderTolerance:            0,
		MatchWinPoints:                  2.0,
		MatchDrawPoints:                 1.0,
		MatchLossPoints:                 0.0,
		TeamTiebreaks:                   []string{"board_points", "buchholz", "direct_encounter"},
		BoardWinPoints:                  1.0,
		BoardDrawPoints:                 0.5,
		BoardLossPoints:                 0.0,
		DefaultBoardResult:              "loss",
	}
}

// TeamStanding represents a team's standing in the tournament
type TeamStanding struct {
	Rank              int       `json:"rank"`
	TeamParticipantID uuid.UUID `json:"teamParticipantId"`
	TeamID            uuid.UUID `json:"teamId"`
	TeamName          string    `json:"teamName"`
	MatchPoints       float64   `json:"matchPoints"`
	BoardPoints       float64   `json:"boardPoints"`
	Buchholz          float64   `json:"buchholz"`
	Wins              int       `json:"wins"`
	Draws             int       `json:"draws"`
	Losses            int       `json:"losses"`
	MatchesPlayed     int       `json:"matchesPlayed"`
}

// Registration status constants
const (
	TeamRegistrationStatusRegistered = "registered"
	TeamRegistrationStatusConfirmed  = "confirmed"
	TeamRegistrationStatusWithdrawn  = "withdrawn"
)

// Match status constants
const (
	TeamMatchStatusScheduled = "scheduled"
	TeamMatchStatusOngoing   = "ongoing"
	TeamMatchStatusCompleted = "completed"
)
