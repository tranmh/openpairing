package tournament

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TournamentType represents the type of tournament
type TournamentType string

const (
	TypeSwiss      TournamentType = "swiss"
	TypeRoundRobin TournamentType = "round_robin"
	TypeKnockout   TournamentType = "knockout"
	TypeTeamSwiss  TournamentType = "team_swiss"
)

// TournamentMode represents how games are played
type TournamentMode string

const (
	ModeOTB    TournamentMode = "otb"
	ModeOnline TournamentMode = "online"
	ModeHybrid TournamentMode = "hybrid"
)

// TournamentStatus represents the current status
type TournamentStatus string

const (
	StatusDraft        TournamentStatus = "draft"
	StatusPublished    TournamentStatus = "published"
	StatusRegistration TournamentStatus = "registration"
	StatusOngoing      TournamentStatus = "ongoing"
	StatusCompleted    TournamentStatus = "completed"
	StatusCancelled    TournamentStatus = "cancelled"
)

// TimeControlType represents the time control category
type TimeControlType string

const (
	TimeControlStandard TimeControlType = "standard"
	TimeControlRapid    TimeControlType = "rapid"
	TimeControlBlitz    TimeControlType = "blitz"
)

// PairingEngine represents the pairing engine to use
type PairingEngine string

const (
	EngineJaVaFo PairingEngine = "javafo"
	EngineBBP    PairingEngine = "bbp"
)

// PairingSystem represents the pairing system
type PairingSystem string

const (
	PairingDutch    PairingSystem = "dutch"
	PairingBurstein PairingSystem = "burstein"
	PairingDubov    PairingSystem = "dubov"
)

// Tournament represents a chess tournament
type Tournament struct {
	ID             uuid.UUID        `json:"id" db:"id"`
	OrganizationID *uuid.UUID       `json:"organizationId,omitempty" db:"organization_id"`

	// Basic Info
	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`
	Slug        string  `json:"slug" db:"slug"`

	// Type & Format
	Type            TournamentType  `json:"type" db:"type"`
	Mode            TournamentMode  `json:"mode" db:"mode"`
	TimeControl     string          `json:"timeControl" db:"time_control"`
	TimeControlType TimeControlType `json:"timeControlType" db:"time_control_type"`

	// Pairing Configuration
	PairingEngine      PairingEngine `json:"pairingEngine" db:"pairing_engine"`
	PairingSystem      PairingSystem `json:"pairingSystem" db:"pairing_system"`
	AcceleratedPairing bool          `json:"acceleratedPairing" db:"accelerated_pairing"`

	// Dates
	StartDate          string     `json:"startDate" db:"start_date"`
	EndDate            *string    `json:"endDate,omitempty" db:"end_date"`
	RegistrationOpens  *time.Time `json:"registrationOpens,omitempty" db:"registration_opens"`
	RegistrationCloses *time.Time `json:"registrationCloses,omitempty" db:"registration_closes"`

	// Limits
	MaxPlayers *int `json:"maxPlayers,omitempty" db:"max_players"`
	MinPlayers int  `json:"minPlayers" db:"min_players"`
	Rounds     int  `json:"rounds" db:"rounds"`

	// Location
	VenueName    *string `json:"venueName,omitempty" db:"venue_name"`
	VenueAddress *string `json:"venueAddress,omitempty" db:"venue_address"`
	VenueCity    *string `json:"venueCity,omitempty" db:"venue_city"`
	VenueCountry *string `json:"venueCountry,omitempty" db:"venue_country"`

	// Rating & Reporting
	FideRated bool `json:"fideRated" db:"fide_rated"`
	DwzRated  bool `json:"dwzRated" db:"dwz_rated"`
	OesbRated bool `json:"oesbRated" db:"oesb_rated"`

	// Fees
	EntryFeeCents int    `json:"entryFeeCents" db:"entry_fee_cents"`
	Currency      string `json:"currency" db:"currency"`

	// Status
	Status TournamentStatus `json:"status" db:"status"`

	// Tiebreaks (JSON array)
	Tiebreaks []string `json:"tiebreaks" db:"tiebreaks"`

	// Settings (JSON)
	Settings map[string]interface{} `json:"settings,omitempty" db:"settings"`

	// Denormalized counts
	CurrentRound      int `json:"currentRound" db:"current_round"`
	ParticipantCount  int `json:"participantCount" db:"participant_count"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// TournamentSection represents a section within a tournament
type TournamentSection struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TournamentID uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	Name         string     `json:"name" db:"name"`
	RatingMin    *int       `json:"ratingMin,omitempty" db:"rating_min"`
	RatingMax    *int       `json:"ratingMax,omitempty" db:"rating_max"`
	AgeMin       *int       `json:"ageMin,omitempty" db:"age_min"`
	AgeMax       *int       `json:"ageMax,omitempty" db:"age_max"`
	Gender       *string    `json:"gender,omitempty" db:"gender"`
	SortOrder    int        `json:"sortOrder" db:"sort_order"`
}

// Participant represents a player registered for a tournament
type Participant struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TournamentID uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	SectionID    *uuid.UUID `json:"sectionId,omitempty" db:"section_id"`
	PlayerID     uuid.UUID  `json:"playerId" db:"player_id"`

	// Registration
	RegistrationStatus string    `json:"registrationStatus" db:"registration_status"`
	RegisteredAt       time.Time `json:"registeredAt" db:"registered_at"`

	// Starting Info
	StartRank      int `json:"startRank" db:"start_rank"`
	StartingRating int `json:"startingRating" db:"starting_rating"`

	// Results (denormalized)
	Points          float64 `json:"points" db:"points"`
	Buchholz        float64 `json:"buchholz" db:"buchholz"`
	BuchholzCut1    float64 `json:"buchholzCut1" db:"buchholz_cut1"`
	SonnebornBerger float64 `json:"sonnebornBerger" db:"sonneborn_berger"`
	Wins            int     `json:"wins" db:"wins"`
	Draws           int     `json:"draws" db:"draws"`
	Losses          int     `json:"losses" db:"losses"`

	// Status
	IsActive    bool     `json:"isActive" db:"is_active"`
	ByeRequests []int    `json:"byeRequests" db:"bye_requests"`

	// Payment
	PaymentStatus    string  `json:"paymentStatus" db:"payment_status"`
	PaymentReference *string `json:"paymentReference,omitempty" db:"payment_reference"`

	// Player info (joined)
	PlayerFirstName *string `json:"playerFirstName,omitempty" db:"player_first_name"`
	PlayerLastName  *string `json:"playerLastName,omitempty" db:"player_last_name"`
	PlayerFideID    *string `json:"playerFideId,omitempty" db:"player_fide_id"`
	PlayerClubName  *string `json:"playerClubName,omitempty" db:"player_club_name"`
}

// Round represents a tournament round
type Round struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TournamentID uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	RoundNumber  int        `json:"roundNumber" db:"round_number"`

	// Timing
	ScheduledStart *time.Time `json:"scheduledStart,omitempty" db:"scheduled_start"`
	ActualStart    *time.Time `json:"actualStart,omitempty" db:"actual_start"`
	CompletedAt    *time.Time `json:"completedAt,omitempty" db:"completed_at"`

	// Status
	Status string `json:"status" db:"status"`

	// Pairing metadata
	PairedAt          *time.Time `json:"pairedAt,omitempty" db:"paired_at"`
	PairedBy          *uuid.UUID `json:"pairedBy,omitempty" db:"paired_by"`
	PairingEngineUsed *string    `json:"pairingEngineUsed,omitempty" db:"pairing_engine_used"`

	// Game count (denormalized)
	GameCount     int `json:"gameCount" db:"game_count"`
	CompletedCount int `json:"completedCount" db:"completed_count"`
}

// RoundStatus constants
const (
	RoundStatusPending   = "pending"
	RoundStatusPairing   = "pairing"
	RoundStatusPaired    = "paired"
	RoundStatusOngoing   = "ongoing"
	RoundStatusCompleted = "completed"
)

// Game represents a single game/pairing
type Game struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TournamentID uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	RoundID      uuid.UUID  `json:"roundId" db:"round_id"`
	SectionID    *uuid.UUID `json:"sectionId,omitempty" db:"section_id"`

	// Pairing
	BoardNumber        int        `json:"boardNumber" db:"board_number"`
	WhiteParticipantID *uuid.UUID `json:"whiteParticipantId,omitempty" db:"white_participant_id"`
	BlackParticipantID *uuid.UUID `json:"blackParticipantId,omitempty" db:"black_participant_id"`

	// Result
	Result     *string  `json:"result,omitempty" db:"result"`
	WhiteScore *float64 `json:"whiteScore,omitempty" db:"white_score"`
	BlackScore *float64 `json:"blackScore,omitempty" db:"black_score"`

	// Online game reference
	OnlineGameID *uuid.UUID `json:"onlineGameId,omitempty" db:"online_game_id"`

	// Status
	Status string `json:"status" db:"status"`

	// Metadata
	ResultEnteredBy *uuid.UUID `json:"resultEnteredBy,omitempty" db:"result_entered_by"`
	ResultEnteredAt *time.Time `json:"resultEnteredAt,omitempty" db:"result_entered_at"`
	PGN             *string    `json:"pgn,omitempty" db:"pgn"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// Joined player info
	WhitePlayerName *string `json:"whitePlayerName,omitempty"`
	BlackPlayerName *string `json:"blackPlayerName,omitempty"`
	WhiteRating     *int    `json:"whiteRating,omitempty"`
	BlackRating     *int    `json:"blackRating,omitempty"`
}

// GameResult constants
const (
	ResultWhiteWins   = "1-0"
	ResultBlackWins   = "0-1"
	ResultDraw        = "1/2-1/2"
	ResultWhiteForfeit = "1-0F"
	ResultBlackForfeit = "0-1F"
	ResultDoubleForfeit = "0-0F"
	ResultBye         = "BYE"
)

// GameStatus constants
const (
	GameStatusScheduled = "scheduled"
	GameStatusOngoing   = "ongoing"
	GameStatusCompleted = "completed"
	GameStatusAdjourned = "adjourned"
)

// ForbiddenPair represents players who shouldn't be paired
type ForbiddenPair struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TournamentID uuid.UUID `json:"tournamentId" db:"tournament_id"`
	Player1ID    uuid.UUID `json:"player1Id" db:"player1_id"`
	Player2ID    uuid.UUID `json:"player2Id" db:"player2_id"`
	Reason       string    `json:"reason" db:"reason"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

// Request/Response types

// CreateTournamentRequest is the request body for creating a tournament
type CreateTournamentRequest struct {
	OrganizationID     *uuid.UUID      `json:"organizationId,omitempty"`
	Name               string          `json:"name" binding:"required,min=1,max=255"`
	Description        *string         `json:"description,omitempty"`
	Slug               string          `json:"slug,omitempty"`
	Type               TournamentType  `json:"type" binding:"required,oneof=swiss round_robin knockout team_swiss"`
	Mode               TournamentMode  `json:"mode" binding:"required,oneof=otb online hybrid"`
	TimeControl        string          `json:"timeControl" binding:"required"`
	TimeControlType    TimeControlType `json:"timeControlType" binding:"required,oneof=standard rapid blitz"`
	PairingEngine      PairingEngine   `json:"pairingEngine" binding:"omitempty,oneof=javafo bbp"`
	PairingSystem      PairingSystem   `json:"pairingSystem,omitempty"`
	AcceleratedPairing bool            `json:"acceleratedPairing"`
	Rounds             int             `json:"rounds" binding:"required,min=1,max=15"`
	StartDate          string          `json:"startDate" binding:"required"`
	EndDate            *string         `json:"endDate,omitempty"`
	RegistrationOpens  *time.Time      `json:"registrationOpens,omitempty"`
	RegistrationCloses *time.Time      `json:"registrationCloses,omitempty"`
	MaxPlayers         *int            `json:"maxPlayers,omitempty"`
	MinPlayers         int             `json:"minPlayers,omitempty"`
	VenueName          *string         `json:"venueName,omitempty"`
	VenueAddress       *string         `json:"venueAddress,omitempty"`
	VenueCity          *string         `json:"venueCity,omitempty"`
	VenueCountry       *string         `json:"venueCountry,omitempty"`
	FIDERated          bool            `json:"fideRated"`
	DWZRated           bool            `json:"dwzRated"`
	OESBRated          bool            `json:"oesbRated"`
	EntryFeeCents      int             `json:"entryFeeCents"`
	Currency           string          `json:"currency,omitempty"`
	Tiebreaks          []string        `json:"tiebreaks,omitempty"`
	Settings           map[string]interface{} `json:"settings,omitempty"`
}

// UpdateTournamentRequest is the request body for updating a tournament
type UpdateTournamentRequest struct {
	Name               *string          `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description        *string          `json:"description,omitempty"`
	TimeControl        *string          `json:"timeControl,omitempty"`
	Rounds             *int             `json:"rounds,omitempty" binding:"omitempty,min=1,max=15"`
	StartDate          *string          `json:"startDate,omitempty"`
	EndDate            *string          `json:"endDate,omitempty"`
	MaxPlayers         *int             `json:"maxPlayers,omitempty"`
	RegistrationOpens  *time.Time       `json:"registrationOpens,omitempty"`
	RegistrationCloses *time.Time       `json:"registrationCloses,omitempty"`
	FideRated          *bool            `json:"fideRated,omitempty"`
	DwzRated           *bool            `json:"dwzRated,omitempty"`
	AcceleratedPairing *bool            `json:"acceleratedPairing,omitempty"`
	Tiebreaks          []string         `json:"tiebreaks,omitempty"`
	VenueName          *string          `json:"venueName,omitempty"`
	VenueAddress       *string          `json:"venueAddress,omitempty"`
	VenueCity          *string          `json:"venueCity,omitempty"`
	Status             *TournamentStatus `json:"status,omitempty"`
}

// RegisterParticipantRequest is the request for registering a participant
type RegisterParticipantRequest struct {
	PlayerID       uuid.UUID  `json:"playerId" binding:"required"`
	SectionID      *uuid.UUID `json:"sectionId,omitempty"`
	StartingRating int        `json:"startingRating,omitempty"`
}

// DefaultTiebreaks returns the default tiebreak order
func DefaultTiebreaks() []string {
	return []string{
		"buchholz_cut1",
		"buchholz",
		"sonneborn_berger",
		"direct_encounter",
		"wins",
		"games_black",
	}
}

// ParseResult converts a result string to scores
func ParseResult(result string) (whiteScore, blackScore float64) {
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

// MarshalJSON for Tiebreaks to handle JSON array
func (t *Tournament) MarshalTiebreaks() ([]byte, error) {
	return json.Marshal(t.Tiebreaks)
}

// UnmarshalTiebreaks from JSON
func (t *Tournament) UnmarshalTiebreaks(data []byte) error {
	return json.Unmarshal(data, &t.Tiebreaks)
}
