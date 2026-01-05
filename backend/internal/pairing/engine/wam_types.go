package engine

import (
	"time"

	"github.com/google/uuid"
)

// TournamentTypeWAM is the WAM tournament type
const TournamentTypeWAM TournamentType = "wam"

// WAMGroupAssignment represents how players are assigned to groups
type WAMGroupAssignment string

const (
	WAMAssignmentSequential WAMGroupAssignment = "sequential" // Best N in A, next N in B, etc.
	WAMAssignmentSerpentine WAMGroupAssignment = "serpentine" // Snake draft: 1→A, 2→B, 3→C, 4→D, 5→D, 6→C, etc.
	WAMAssignmentRandom     WAMGroupAssignment = "random"     // Random distribution
	WAMAssignmentManual     WAMGroupAssignment = "manual"     // Manual only
)

// WAMPhase represents the current phase of a WAM tournament
type WAMPhase string

const (
	WAMPhaseSetup    WAMPhase = "setup"    // Groups not yet assigned
	WAMPhaseGroup    WAMPhase = "group"    // Group round-robin phase
	WAMPhasePlayoff  WAMPhase = "playoff"  // Knockout playoffs
	WAMPhaseComplete WAMPhase = "complete" // Tournament finished
)

// WAMConfig holds configuration for a WAM tournament
type WAMConfig struct {
	TournamentID     uuid.UUID          `json:"tournamentId"`
	GroupSize        int                `json:"groupSize"`        // User-chosen: 3, 4, 5, 6, etc.
	AssignmentMethod WAMGroupAssignment `json:"assignmentMethod"` // sequential, serpentine, random, manual
	EnablePlayoffs   bool               `json:"enablePlayoffs"`
	PlayoffsTopN     int                `json:"playoffsTopN"`     // Top N from each group advance
	ThirdPlaceMatch  bool               `json:"thirdPlaceMatch"`
	YouthAgeLimit    int                `json:"youthAgeLimit"`    // For soft warning (default 12)
	CurrentPhase     WAMPhase           `json:"currentPhase"`
	Tiebreaks        []string           `json:"tiebreaks"` // ["direct_encounter", "sonneborn_berger", "buchholz"]
	Season           string             `json:"season"`    // e.g., "2025/2026"
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
}

// WAMGroup represents a single group in the WAM tournament
type WAMGroup struct {
	ID           uuid.UUID        `json:"id"`
	TournamentID uuid.UUID        `json:"tournamentId"`
	Name         string           `json:"name"`        // "A", "B", "C", etc.
	GroupNumber  int              `json:"groupNumber"` // 1, 2, 3, etc.
	Players      []WAMGroupPlayer `json:"players"`
	RoundsTotal  int              `json:"roundsTotal"`
	RoundsPlayed int              `json:"roundsPlayed"`
	IsComplete   bool             `json:"isComplete"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// WAMGroupPlayer represents a player's status within a group
type WAMGroupPlayer struct {
	ID             uuid.UUID  `json:"id"`
	GroupID        uuid.UUID  `json:"groupId"`
	ParticipantID  uuid.UUID  `json:"participantId"`
	PlayerID       uuid.UUID  `json:"playerId,omitempty"`
	Name           string     `json:"name"`
	Rating         int        `json:"rating"`
	ClubID         *uuid.UUID `json:"clubId,omitempty"`
	ClubName       *string    `json:"clubName,omitempty"`
	BirthDate      *time.Time `json:"birthDate,omitempty"`
	SortOrder      int        `json:"sortOrder"` // Position within group (1-indexed)
	Points         float64    `json:"points"`
	Wins           int        `json:"wins"`
	Draws          int        `json:"draws"`
	Losses         int        `json:"losses"`
	IsGroupWinner  bool       `json:"isGroupWinner"`
	QualifiedFinal bool       `json:"qualifiedFinal"` // Qualified for annual finals
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// WAMGroupStanding represents standings within a group
type WAMGroupStanding struct {
	Rank            int       `json:"rank"`
	ParticipantID   uuid.UUID `json:"participantId"`
	Name            string    `json:"name"`
	Rating          int       `json:"rating"`
	ClubName        *string   `json:"clubName,omitempty"`
	Points          float64   `json:"points"`
	Wins            int       `json:"wins"`
	Draws           int       `json:"draws"`
	Losses          int       `json:"losses"`
	DirectScore     float64   `json:"directScore"`     // Head-to-head for ties
	SonnebornBerger float64   `json:"sonnebornBerger"`
	Buchholz        float64   `json:"buchholz"`
	QualifiesNext   bool      `json:"qualifiesNext"` // Advances to playoffs
}

// WAMPlayoffMatch represents a single playoff match
type WAMPlayoffMatch struct {
	ID            uuid.UUID  `json:"id"`
	TournamentID  uuid.UUID  `json:"tournamentId"`
	RoundName     string     `json:"roundName"`    // "quarterfinal", "semifinal", "final", "third_place"
	MatchNumber   int        `json:"matchNumber"`
	Player1ID     *uuid.UUID `json:"player1Id,omitempty"`
	Player2ID     *uuid.UUID `json:"player2Id,omitempty"`
	Player1Source string     `json:"player1Source,omitempty"` // e.g., "group_A_1" or "semifinal_1_winner"
	Player2Source string     `json:"player2Source,omitempty"`
	WinnerID      *uuid.UUID `json:"winnerId,omitempty"`
	LoserID       *uuid.UUID `json:"loserId,omitempty"`
	GameID        *uuid.UUID `json:"gameId,omitempty"`
	Player1Name   string     `json:"player1Name,omitempty"`
	Player2Name   string     `json:"player2Name,omitempty"`
	WinnerName    string     `json:"winnerName,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// WAMPlayoffBracket represents the complete knockout playoff structure
type WAMPlayoffBracket struct {
	TournamentID uuid.UUID         `json:"tournamentId"`
	Matches      []WAMPlayoffMatch `json:"matches"`
	Champion     *uuid.UUID        `json:"champion,omitempty"`
	RunnerUp     *uuid.UUID        `json:"runnerUp,omitempty"`
	ThirdPlace   *uuid.UUID        `json:"thirdPlace,omitempty"`
	IsComplete   bool              `json:"isComplete"`
}

// WAMQualification tracks players qualified for annual finals
type WAMQualification struct {
	ID            uuid.UUID `json:"id"`
	TournamentID  uuid.UUID `json:"tournamentId"`
	ParticipantID uuid.UUID `json:"participantId"`
	PlayerID      uuid.UUID `json:"playerId"`
	PlayerName    string    `json:"playerName"`
	GroupName     string    `json:"groupName"`
	GroupRank     int       `json:"groupRank"`
	Season        string    `json:"season"` // e.g., "2025/2026"
	QualifiedAt   time.Time `json:"qualifiedAt"`
}

// WAMPairingRequest extends GeneralPairingRequest for WAM-specific needs
type WAMPairingRequest struct {
	TournamentID uuid.UUID  `json:"tournamentId"`
	Config       *WAMConfig `json:"config"`
	Groups       []WAMGroup `json:"groups"`
	Phase        WAMPhase   `json:"phase"`              // "group" or "playoff"
	GroupID      *uuid.UUID `json:"groupId,omitempty"`  // Specific group for pairing
	RoundNumber  int        `json:"roundNumber"`
}

// WAMClubConflict represents same-club players in the same group
type WAMClubConflict struct {
	GroupID   uuid.UUID        `json:"groupId"`
	GroupName string           `json:"groupName"`
	ClubID    uuid.UUID        `json:"clubId"`
	ClubName  string           `json:"clubName"`
	Players   []WAMGroupPlayer `json:"players"`
}

// WAMAgeWarning represents a warning about age restrictions
type WAMAgeWarning struct {
	GroupID    uuid.UUID `json:"groupId"`
	GroupName  string    `json:"groupName"`
	GroupSize  int       `json:"groupSize"`
	Player     string    `json:"player"`
	Age        int       `json:"age"`
	AgeLimit   int       `json:"ageLimit"`
	Message    string    `json:"message"`
}

// WAMGroupGame links a game to a specific group
type WAMGroupGame struct {
	ID           uuid.UUID `json:"id"`
	GroupID      uuid.UUID `json:"groupId"`
	GameID       uuid.UUID `json:"gameId"`
	RoundInGroup int       `json:"roundInGroup"`
	CreatedAt    time.Time `json:"createdAt"`
}

// GetTotalGroupRounds returns total rounds needed for a round-robin within a group
func GetTotalGroupRounds(groupSize int) int {
	if groupSize <= 1 {
		return 0
	}
	if groupSize%2 == 0 {
		return groupSize - 1
	}
	return groupSize // Odd number needs n rounds (with byes)
}

// GroupNameFromNumber converts group number to letter (1 -> "A", 2 -> "B", etc.)
func GroupNameFromNumber(n int) string {
	if n < 1 || n > 26 {
		return ""
	}
	return string(rune('A' + n - 1))
}

// GroupNumberFromName converts group letter to number ("A" -> 1, "B" -> 2, etc.)
func GroupNumberFromName(name string) int {
	if len(name) != 1 {
		return 0
	}
	ch := rune(name[0])
	if ch >= 'A' && ch <= 'Z' {
		return int(ch - 'A' + 1)
	}
	if ch >= 'a' && ch <= 'z' {
		return int(ch - 'a' + 1)
	}
	return 0
}
