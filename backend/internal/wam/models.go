package wam

import (
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/pairing/engine"
)

// Re-export types from engine package for convenience
type (
	WAMConfig         = engine.WAMConfig
	WAMGroup          = engine.WAMGroup
	WAMGroupPlayer    = engine.WAMGroupPlayer
	WAMGroupStanding  = engine.WAMGroupStanding
	WAMPlayoffMatch   = engine.WAMPlayoffMatch
	WAMPlayoffBracket = engine.WAMPlayoffBracket
	WAMQualification  = engine.WAMQualification
	WAMClubConflict   = engine.WAMClubConflict
	WAMAgeWarning     = engine.WAMAgeWarning
	WAMGroupGame      = engine.WAMGroupGame
	WAMPhase          = engine.WAMPhase
	WAMGroupAssignment = engine.WAMGroupAssignment
)

// Phase constants
const (
	WAMPhaseSetup    = engine.WAMPhaseSetup
	WAMPhaseGroup    = engine.WAMPhaseGroup
	WAMPhasePlayoff  = engine.WAMPhasePlayoff
	WAMPhaseComplete = engine.WAMPhaseComplete
)

// Assignment method constants
const (
	WAMAssignmentSequential = engine.WAMAssignmentSequential
	WAMAssignmentSerpentine = engine.WAMAssignmentSerpentine
	WAMAssignmentRandom     = engine.WAMAssignmentRandom
	WAMAssignmentManual     = engine.WAMAssignmentManual
)

// ConfigureWAMRequest is the request to configure WAM tournament settings
type ConfigureWAMRequest struct {
	GroupSize         int      `json:"groupSize" binding:"required,min=3,max=10"`
	DefaultGroupSize  int      `json:"defaultGroupSize" binding:"omitempty,min=3,max=10"`
	AllowedGroupSizes []int    `json:"allowedGroupSizes,omitempty"` // e.g., [4, 6]
	AssignmentMethod  string   `json:"assignmentMethod" binding:"required,oneof=sequential serpentine random manual"`
	EnablePlayoffs    bool     `json:"enablePlayoffs"`
	PlayoffsTopN      int      `json:"playoffsTopN" binding:"omitempty,min=1,max=4"`
	ThirdPlaceMatch   bool     `json:"thirdPlaceMatch"`
	YouthAgeLimit     int      `json:"youthAgeLimit" binding:"omitempty,min=0,max=100"`
	Tiebreaks         []string `json:"tiebreaks,omitempty"`
	Season            string   `json:"season,omitempty"`
}

// AssignGroupsRequest is the request for automatic group assignment
type AssignGroupsRequest struct {
	AssignmentMethod string `json:"assignmentMethod" binding:"required,oneof=sequential serpentine random"`
}

// AssignGroupsResponse is the response after assigning groups
type AssignGroupsResponse struct {
	Groups   []WAMGroup `json:"groups"`
	Warnings []string   `json:"warnings,omitempty"`
}

// MovePlayerRequest is the request to move a player between groups
type MovePlayerRequest struct {
	ParticipantID uuid.UUID `json:"participantId" binding:"required"`
	TargetGroupID uuid.UUID `json:"targetGroupId" binding:"required"`
	NewSortOrder  int       `json:"newSortOrder,omitempty"` // Position in new group (optional)
}

// AddGroupMemberRequest is the request to add a player to a group
type AddGroupMemberRequest struct {
	ParticipantID uuid.UUID `json:"participantId" binding:"required"`
	SortOrder     int       `json:"sortOrder,omitempty"`
}

// GroupStandingsResponse is the response containing group standings
type GroupStandingsResponse struct {
	GroupID       uuid.UUID          `json:"groupId"`
	GroupName     string             `json:"groupName"`
	Standings     []WAMGroupStanding `json:"standings"`
	IsComplete    bool               `json:"isComplete"`
	ClubConflicts []WAMClubConflict  `json:"clubConflicts,omitempty"`
}

// PlayoffBracketResponse is the response containing the playoff bracket
type PlayoffBracketResponse struct {
	Bracket       *WAMPlayoffBracket `json:"bracket"`
	CanInitialize bool               `json:"canInitialize"`
	IsComplete    bool               `json:"isComplete"`
}

// QualificationsResponse is the response containing qualified players
type QualificationsResponse struct {
	Season         string             `json:"season"`
	Qualifications []WAMQualification `json:"qualifications"`
	TotalCount     int                `json:"totalCount"`
}

// QualificationsExportRequest is the request for exporting qualifications
type QualificationsExportRequest struct {
	Format string `json:"format" binding:"required,oneof=csv json"`
	Season string `json:"season,omitempty"`
}

// GeneratePairingsRequest is the request for generating group pairings
type GeneratePairingsRequest struct {
	RoundNumber int `json:"roundNumber,omitempty"` // If not specified, uses next round
}

// PairingsResponse is the response after generating pairings
type PairingsResponse struct {
	GroupID      uuid.UUID `json:"groupId"`
	GroupName    string    `json:"groupName"`
	RoundNumber  int       `json:"roundNumber"`
	GamesCreated int       `json:"gamesCreated"`
	GameIDs      []uuid.UUID `json:"gameIds"`
}

// WAMTournamentStatus represents the overall WAM tournament status
type WAMTournamentStatus struct {
	TournamentID    uuid.UUID        `json:"tournamentId"`
	Config          *WAMConfig       `json:"config"`
	Phase           WAMPhase         `json:"phase"`
	Groups          []WAMGroupStatus `json:"groups"`
	TotalPlayers    int              `json:"totalPlayers"`
	PlayersAssigned int              `json:"playersAssigned"`
	CanStartGroups  bool             `json:"canStartGroups"`
	CanStartPlayoffs bool            `json:"canStartPlayoffs"`
	Warnings        []string         `json:"warnings,omitempty"`
}

// WAMGroupStatus is a summary of a group's status
type WAMGroupStatus struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	PlayerCount   int       `json:"playerCount"`
	RoundsTotal   int       `json:"roundsTotal"`
	RoundsPlayed  int       `json:"roundsPlayed"`
	GamesComplete int       `json:"gamesComplete"`
	GamesTotal    int       `json:"gamesTotal"`
	IsComplete    bool      `json:"isComplete"`
}

// TournamentParticipant represents a participant for WAM assignment
type TournamentParticipant struct {
	ParticipantID uuid.UUID  `json:"participantId"`
	PlayerID      uuid.UUID  `json:"playerId"`
	Name          string     `json:"name"`
	Rating        int        `json:"rating"`
	JokerPoints   int        `json:"jokerPoints"`   // 0, 50, 100, or 150
	IsU12         bool       `json:"isU12"`         // Player is under youth age limit
	Age           *int       `json:"age,omitempty"` // Calculated age
	ClubID        *uuid.UUID `json:"clubId,omitempty"`
	ClubName      *string    `json:"clubName,omitempty"`
	BirthDate     *time.Time `json:"birthDate,omitempty"`
}

// Game represents a game for WAM purposes
type Game struct {
	ID                 uuid.UUID  `json:"id"`
	RoundID            uuid.UUID  `json:"roundId"`
	BoardNumber        int        `json:"boardNumber"`
	WhiteParticipantID *uuid.UUID `json:"whiteParticipantId,omitempty"`
	BlackParticipantID *uuid.UUID `json:"blackParticipantId,omitempty"`
	WhiteScore         *float64   `json:"whiteScore,omitempty"`
	BlackScore         *float64   `json:"blackScore,omitempty"`
	Result             *string    `json:"result,omitempty"`
	Status             string     `json:"status"`
}

// SetJokerRequest is the request to set joker points for a participant
type SetJokerRequest struct {
	JokerPoints int `json:"jokerPoints" binding:"oneof=0 50 100 150"`
}

// GroupProposal represents a proposed group configuration
type GroupProposal struct {
	Groups       []ProposedGroup    `json:"groups"`
	TotalPlayers int                `json:"totalPlayers"`
	Warnings     []string           `json:"warnings,omitempty"`
	U12Proposals []U12GroupProposal `json:"u12Proposals,omitempty"`
}

// ProposedGroup represents a single proposed group
type ProposedGroup struct {
	Name       string                  `json:"name"`       // "A", "B", etc.
	TargetSize int                     `json:"targetSize"` // 4 or 6
	Players    []TournamentParticipant `json:"players"`
	IsU12Group bool                    `json:"isU12Group"`
}

// U12GroupProposal represents a proposal for a U12-only group
type U12GroupProposal struct {
	GroupName        string                  `json:"groupName"`
	Players          []TournamentParticipant `json:"players"`
	Reason           string                  `json:"reason"`
	RequiresApproval bool                    `json:"requiresApproval"`
}

// ConfirmGroupsRequest is the request to confirm proposed groups
type ConfirmGroupsRequest struct {
	Proposals []ConfirmedGroup `json:"proposals" binding:"required"`
}

// ConfirmedGroup represents a confirmed group configuration
type ConfirmedGroup struct {
	GroupName      string      `json:"groupName"`
	TargetSize     int         `json:"targetSize"`
	ParticipantIDs []uuid.UUID `json:"participantIds"`
}
