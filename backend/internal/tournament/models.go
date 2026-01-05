package tournament

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TournamentType represents the type of tournament
type TournamentType string

const (
	TypeSwiss        TournamentType = "swiss"
	TypeRoundRobin   TournamentType = "round_robin"
	TypeKnockout     TournamentType = "knockout"
	TypeTeamSwiss    TournamentType = "team_swiss"
	TypeScheveningen TournamentType = "scheveningen"
	TypeArena        TournamentType = "arena"
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
	StatusPairing      TournamentStatus = "pairing" // Locked during pairing generation
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
	EngineJaVaFo     PairingEngine = "javafo"
	EngineBBP        PairingEngine = "bbp"
	EngineRoundRobin PairingEngine = "round_robin"
	EngineKnockout   PairingEngine = "knockout"
	EngineTeamSwiss  PairingEngine = "team_swiss"
	EngineMonrad     PairingEngine = "monrad"
	EngineKeizer     PairingEngine = "keizer"
	EngineArena      PairingEngine = "arena"
)

// PairingSystem represents the pairing system
type PairingSystem string

const (
	PairingDutch       PairingSystem = "dutch"
	PairingBurstein    PairingSystem = "burstein"
	PairingDubov       PairingSystem = "dubov"
	PairingMonrad      PairingSystem = "monrad"
	PairingKeizer      PairingSystem = "keizer"
	PairingBerger      PairingSystem = "berger"       // Round robin
	PairingScheveningen PairingSystem = "scheveningen"
)

// ByeSettings contains all configurable settings for byes, late entry, forfeits, and withdrawals
type ByeSettings struct {
	LateEntry          LateEntrySettings          `json:"lateEntry"`
	HalfPointBye       HalfPointByeSettings       `json:"halfPointBye"`
	SystemBye          SystemByeSettings          `json:"systemBye"`
	ZeroPointBye       ZeroPointByeSettings       `json:"zeroPointBye"`
	Forfeit            ForfeitSettings            `json:"forfeit"`
	Withdrawal         WithdrawalSettings         `json:"withdrawal"`
	PairingAdjustments PairingAdjustmentSettings  `json:"pairingAdjustments"`
}

// LateEntrySettings configures late entry rules
type LateEntrySettings struct {
	Allowed           bool    `json:"allowed"`           // FIDE: true
	MaxRound          int     `json:"maxRound"`          // FIDE: 2
	MissedRoundPoints float64 `json:"missedRoundPoints"` // FIDE: 0
	RequiresApproval  bool    `json:"requiresApproval"`  // FIDE: true
	MinRating         *int    `json:"minRating"`         // FIDE: null
	MaxRating         *int    `json:"maxRating"`         // FIDE: null
	PairingPriority   string  `json:"pairingPriority"`   // FIDE: "bottom" (options: bottom, by_rating, random)
}

// HalfPointByeSettings configures requested half-point bye rules
type HalfPointByeSettings struct {
	Allowed         bool    `json:"allowed"`         // FIDE: true
	MaxPerPlayer    int     `json:"maxPerPlayer"`    // FIDE: 1
	Points          float64 `json:"points"`          // FIDE: 0.5
	AllowedRounds   string  `json:"allowedRounds"`   // FIDE: "first_half" (options: first_half, all, specific, none)
	SpecificRounds  []int   `json:"specificRounds"`  // Used when AllowedRounds = "specific"
	RequestDeadline string  `json:"requestDeadline"` // FIDE: "before_round1" (options: before_round1, before_each_round, anytime)
	DeadlineHours   int     `json:"deadlineHours"`   // Hours before round if deadline = "before_each_round"
	CountsAsGame    bool    `json:"countsAsGame"`    // For color alternation purposes
}

// SystemByeSettings configures system bye (odd player count) rules
type SystemByeSettings struct {
	Points                float64 `json:"points"`                // FIDE: 1.0
	Selection             string  `json:"selection"`             // FIDE: "lowest_score_then_rating" (options below)
	MaxPerPlayer          int     `json:"maxPerPlayer"`          // FIDE: 1
	CountsForBuchholz     bool    `json:"countsForBuchholz"`     // FIDE: true
	VirtualOpponentRating *int    `json:"virtualOpponentRating"` // FIDE: null (use own rating)
	ColorAssigned         string  `json:"colorAssigned"`         // FIDE: "none" (options: none, due_color, white, black)
}

// SystemBye selection options
const (
	SystemByeLowestScoreThenRating = "lowest_score_then_rating" // FIDE default
	SystemByeLowestRating          = "lowest_rating"
	SystemByeHighestRating         = "highest_rating"
	SystemByeRandom                = "random"
	SystemByeRotation              = "rotation"
	SystemByeVolunteer             = "volunteer"
)

// ZeroPointByeSettings configures zero-point bye rules
type ZeroPointByeSettings struct {
	Allowed         bool   `json:"allowed"`         // FIDE: true
	MaxPerPlayer    int    `json:"maxPerPlayer"`    // FIDE: -1 (unlimited)
	AllowedRounds   string `json:"allowedRounds"`   // FIDE: "all" (options: all, first_half, specific, none)
	SpecificRounds  []int  `json:"specificRounds"`  // Used when AllowedRounds = "specific"
	RequestDeadline string `json:"requestDeadline"` // FIDE: "before_each_round"
}

// ForfeitSettings configures forfeit handling rules
type ForfeitSettings struct {
	WinnerPoints         float64 `json:"winnerPoints"`         // FIDE: 1.0
	LoserPoints          float64 `json:"loserPoints"`          // FIDE: 0
	DoubleForfeitPoints  float64 `json:"doubleForfeitPoints"`  // FIDE: 0
	AutoWithdrawAfter    int     `json:"autoWithdrawAfter"`    // FIDE: 2 (0 = disabled)
	ConsecutiveOnly      bool    `json:"consecutiveOnly"`      // FIDE: true
	CountsForBuchholz    string  `json:"countsForBuchholz"`    // FIDE: "as_loss" (options: as_loss, excluded, as_draw)
	NotificationRequired bool    `json:"notificationRequired"` // FIDE: false
	GracePeriodMinutes   int     `json:"gracePeriodMinutes"`   // FIDE: 60
}

// WithdrawalSettings configures withdrawal handling rules
type WithdrawalSettings struct {
	Allowed                bool    `json:"allowed"`                // FIDE: true
	RequiresApproval       bool    `json:"requiresApproval"`       // FIDE: false
	BuchholzMethod         string  `json:"buchholzMethod"`         // FIDE: "virtual_50_percent" (options below)
	VirtualScorePercent    float64 `json:"virtualScorePercent"`    // FIDE: 50 (used with virtual methods)
	ReentryAllowed         bool    `json:"reentryAllowed"`         // FIDE: false
	ReentryMaxRoundsMissed int     `json:"reentryMaxRoundsMissed"` // Max rounds missed for reentry
	OpponentResults        string  `json:"opponentResults"`        // FIDE: "keep" (options: keep, annul, convert_to_bye)
}

// Buchholz calculation methods for withdrawn players
const (
	BuchholzVirtual50Percent     = "virtual_50_percent"     // FIDE default
	BuchholzVirtualCustomPercent = "virtual_custom_percent" // Use VirtualScorePercent
	BuchholzActualOnly           = "actual_only"            // Only count actual games
	BuchholzMinimumOpponent      = "minimum_opponent"       // Use lowest opponent's score
	BuchholzExcluded             = "excluded"               // Don't count in Buchholz
)

// PairingAdjustmentSettings configures pairing behavior for byes
type PairingAdjustmentSettings struct {
	ByeRecipientColorPref     string `json:"byeRecipientColorPref"`     // FIDE: "due_color" (options: due_color, none, white, black)
	ByeRecipientFloatPriority string `json:"byeRecipientFloatPriority"` // FIDE: "normal" (options: normal, down, up)
	LateEntryColorPref        string `json:"lateEntryColorPref"`        // FIDE: "none" (options: none, white, black, random)
	MultipleByeSameRound      string `json:"multipleByeSameRound"`      // FIDE: "sequential" (options: sequential, parallel, error)
}

// DefaultByeSettings returns FIDE-compliant default settings
func DefaultByeSettings() *ByeSettings {
	return &ByeSettings{
		LateEntry: LateEntrySettings{
			Allowed:           true,
			MaxRound:          2,
			MissedRoundPoints: 0,
			RequiresApproval:  true,
			MinRating:         nil,
			MaxRating:         nil,
			PairingPriority:   "bottom",
		},
		HalfPointBye: HalfPointByeSettings{
			Allowed:         true,
			MaxPerPlayer:    1,
			Points:          0.5,
			AllowedRounds:   "first_half",
			SpecificRounds:  nil,
			RequestDeadline: "before_round1",
			DeadlineHours:   24,
			CountsAsGame:    false,
		},
		SystemBye: SystemByeSettings{
			Points:                1.0,
			Selection:             SystemByeLowestScoreThenRating,
			MaxPerPlayer:          1,
			CountsForBuchholz:     true,
			VirtualOpponentRating: nil,
			ColorAssigned:         "none",
		},
		ZeroPointBye: ZeroPointByeSettings{
			Allowed:         true,
			MaxPerPlayer:    -1, // unlimited
			AllowedRounds:   "all",
			SpecificRounds:  nil,
			RequestDeadline: "before_each_round",
		},
		Forfeit: ForfeitSettings{
			WinnerPoints:         1.0,
			LoserPoints:          0,
			DoubleForfeitPoints:  0,
			AutoWithdrawAfter:    2,
			ConsecutiveOnly:      true,
			CountsForBuchholz:    "as_loss",
			NotificationRequired: false,
			GracePeriodMinutes:   60,
		},
		Withdrawal: WithdrawalSettings{
			Allowed:                true,
			RequiresApproval:       false,
			BuchholzMethod:         BuchholzVirtual50Percent,
			VirtualScorePercent:    50,
			ReentryAllowed:         false,
			ReentryMaxRoundsMissed: 1,
			OpponentResults:        "keep",
		},
		PairingAdjustments: PairingAdjustmentSettings{
			ByeRecipientColorPref:     "due_color",
			ByeRecipientFloatPriority: "normal",
			LateEntryColorPref:        "none",
			MultipleByeSameRound:      "sequential",
		},
	}
}

// FIDEViolation represents a deviation from FIDE rules
type FIDEViolation struct {
	Setting    string `json:"setting"`
	Value      any    `json:"value"`
	FIDEValue  any    `json:"fideValue"`
	Severity   string `json:"severity"` // "warning", "info"
	Message    string `json:"message"`
	Regulation string `json:"regulation"` // FIDE handbook reference
}

// ValidateFIDECompliance checks settings against FIDE rules and returns violations
func (s *ByeSettings) ValidateFIDECompliance(totalRounds int) []FIDEViolation {
	var violations []FIDEViolation

	// Late Entry violations
	if s.LateEntry.MaxRound > totalRounds/2 {
		violations = append(violations, FIDEViolation{
			Setting:    "lateEntry.maxRound",
			Value:      s.LateEntry.MaxRound,
			FIDEValue:  totalRounds / 2,
			Severity:   "warning",
			Message:    "FIDE recommends late entry only in first half of tournament",
			Regulation: "FIDE Handbook C.04.2.B",
		})
	}
	if s.LateEntry.MissedRoundPoints > 0 {
		violations = append(violations, FIDEViolation{
			Setting:    "lateEntry.missedRoundPoints",
			Value:      s.LateEntry.MissedRoundPoints,
			FIDEValue:  0,
			Severity:   "warning",
			Message:    "FIDE awards 0 points for missed rounds due to late entry",
			Regulation: "FIDE Handbook C.04.2.B",
		})
	}

	// Half-Point Bye violations
	if s.HalfPointBye.MaxPerPlayer > 1 {
		violations = append(violations, FIDEViolation{
			Setting:    "halfPointBye.maxPerPlayer",
			Value:      s.HalfPointBye.MaxPerPlayer,
			FIDEValue:  1,
			Severity:   "warning",
			Message:    "FIDE allows maximum 1 half-point bye per player",
			Regulation: "FIDE Handbook C.04.1.f",
		})
	}
	if s.HalfPointBye.Points != 0.5 {
		violations = append(violations, FIDEViolation{
			Setting:    "halfPointBye.points",
			Value:      s.HalfPointBye.Points,
			FIDEValue:  0.5,
			Severity:   "warning",
			Message:    "FIDE awards exactly 0.5 points for requested bye",
			Regulation: "FIDE Handbook C.04.1.f",
		})
	}
	if s.HalfPointBye.AllowedRounds == "all" {
		violations = append(violations, FIDEViolation{
			Setting:    "halfPointBye.allowedRounds",
			Value:      s.HalfPointBye.AllowedRounds,
			FIDEValue:  "first_half",
			Severity:   "warning",
			Message:    "FIDE restricts half-point byes to first half of tournament",
			Regulation: "FIDE Handbook C.04.1.f",
		})
	}

	// System Bye violations
	if s.SystemBye.Points != 1.0 {
		violations = append(violations, FIDEViolation{
			Setting:    "systemBye.points",
			Value:      s.SystemBye.Points,
			FIDEValue:  1.0,
			Severity:   "warning",
			Message:    "FIDE awards 1 point for system bye (odd player)",
			Regulation: "FIDE Handbook C.04.1.e",
		})
	}
	if s.SystemBye.MaxPerPlayer > 1 {
		violations = append(violations, FIDEViolation{
			Setting:    "systemBye.maxPerPlayer",
			Value:      s.SystemBye.MaxPerPlayer,
			FIDEValue:  1,
			Severity:   "warning",
			Message:    "FIDE: a player should not receive more than 1 system bye",
			Regulation: "FIDE Handbook C.04.1.e",
		})
	}

	// Forfeit violations
	if s.Forfeit.GracePeriodMinutes > 60 {
		violations = append(violations, FIDEViolation{
			Setting:    "forfeit.gracePeriodMinutes",
			Value:      s.Forfeit.GracePeriodMinutes,
			FIDEValue:  60,
			Severity:   "info",
			Message:    "FIDE default forfeit time is 60 minutes",
			Regulation: "FIDE Laws of Chess 6.7",
		})
	}
	if s.Forfeit.AutoWithdrawAfter == 0 {
		violations = append(violations, FIDEViolation{
			Setting:    "forfeit.autoWithdrawAfter",
			Value:      0,
			FIDEValue:  2,
			Severity:   "info",
			Message:    "FIDE recommends auto-withdrawal after 2 consecutive forfeits",
			Regulation: "FIDE Handbook C.04.3",
		})
	}
	if !s.Forfeit.ConsecutiveOnly {
		violations = append(violations, FIDEViolation{
			Setting:    "forfeit.consecutiveOnly",
			Value:      false,
			FIDEValue:  true,
			Severity:   "warning",
			Message:    "FIDE counts only consecutive forfeits for auto-withdrawal",
			Regulation: "FIDE Handbook C.04.3",
		})
	}

	// Withdrawal violations
	if s.Withdrawal.BuchholzMethod != BuchholzVirtual50Percent {
		violations = append(violations, FIDEViolation{
			Setting:    "withdrawal.buchholzMethod",
			Value:      s.Withdrawal.BuchholzMethod,
			FIDEValue:  BuchholzVirtual50Percent,
			Severity:   "warning",
			Message:    "FIDE uses virtual 50% continuation for withdrawn players in Buchholz",
			Regulation: "FIDE Handbook C.02.13.4",
		})
	}
	if s.Withdrawal.OpponentResults != "keep" {
		violations = append(violations, FIDEViolation{
			Setting:    "withdrawal.opponentResults",
			Value:      s.Withdrawal.OpponentResults,
			FIDEValue:  "keep",
			Severity:   "warning",
			Message:    "FIDE keeps all results against withdrawn players",
			Regulation: "FIDE Handbook C.04.3",
		})
	}

	return violations
}

// ScoringPreset represents predefined scoring configurations
type ScoringPreset string

const (
	ScoringPresetStandard ScoringPreset = "standard" // 1-0.5-0 (FIDE default)
	ScoringPresetFootball ScoringPreset = "football" // 3-1-0 (encourages decisive games)
	ScoringPresetCustom   ScoringPreset = "custom"   // User-defined values
)

// SonnebornBergerMode defines how Sonneborn-Berger tiebreaker is calculated
type SonnebornBergerMode string

const (
	// SBModeProportional calculates SB as: sum of (normalized_score * opponent_points)
	// With 3-1-0 scoring, a draw contributes 1/3 of opponent's score (mathematically consistent)
	SBModeProportional SonnebornBergerMode = "proportional"

	// SBModeTraditional calculates SB as: full opponent points for win, half for draw
	// Draw always contributes 1/2 of opponent's score regardless of point system
	SBModeTraditional SonnebornBergerMode = "traditional"
)

// ScoringSettings contains configurable point values for game results
// Bye and forfeit points are configured separately in ByeSettings
type ScoringSettings struct {
	Preset              ScoringPreset       `json:"preset"`              // Preset name for UI
	WinPoints           float64             `json:"winPoints"`           // Points for a win (default: 1.0)
	DrawPoints          float64             `json:"drawPoints"`          // Points for a draw (default: 0.5)
	LossPoints          float64             `json:"lossPoints"`          // Points for a loss (default: 0.0)
	SonnebornBergerMode SonnebornBergerMode `json:"sonnebornBergerMode"` // SB calculation mode
}

// DefaultScoringSettings returns FIDE-compliant standard scoring (1-0.5-0)
func DefaultScoringSettings() *ScoringSettings {
	return &ScoringSettings{
		Preset:              ScoringPresetStandard,
		WinPoints:           1.0,
		DrawPoints:          0.5,
		LossPoints:          0.0,
		SonnebornBergerMode: SBModeProportional,
	}
}

// FootballScoringSettings returns 3-1-0 scoring (Bilbao system)
func FootballScoringSettings() *ScoringSettings {
	return &ScoringSettings{
		Preset:              ScoringPresetFootball,
		WinPoints:           3.0,
		DrawPoints:          1.0,
		LossPoints:          0.0,
		SonnebornBergerMode: SBModeProportional,
	}
}

// GetScoringPreset returns settings for a named preset
func GetScoringPreset(preset ScoringPreset) *ScoringSettings {
	switch preset {
	case ScoringPresetFootball:
		return FootballScoringSettings()
	default:
		return DefaultScoringSettings()
	}
}

// ScoringValidationError represents a validation error for scoring settings
type ScoringValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validate checks if the scoring settings are valid
func (s *ScoringSettings) Validate() []ScoringValidationError {
	var errors []ScoringValidationError

	// Win points must be positive
	if s.WinPoints <= 0 {
		errors = append(errors, ScoringValidationError{
			Field:   "winPoints",
			Message: "win points must be greater than 0",
		})
	}

	// Draw points must be non-negative
	if s.DrawPoints < 0 {
		errors = append(errors, ScoringValidationError{
			Field:   "drawPoints",
			Message: "draw points cannot be negative",
		})
	}

	// Loss points must be non-negative
	if s.LossPoints < 0 {
		errors = append(errors, ScoringValidationError{
			Field:   "lossPoints",
			Message: "loss points cannot be negative",
		})
	}

	// Win > Draw > Loss ordering
	if s.WinPoints <= s.DrawPoints {
		errors = append(errors, ScoringValidationError{
			Field:   "winPoints",
			Message: "win points must be greater than draw points",
		})
	}

	if s.DrawPoints <= s.LossPoints {
		errors = append(errors, ScoringValidationError{
			Field:   "drawPoints",
			Message: "draw points must be greater than loss points",
		})
	}

	// Validate Sonneborn-Berger mode
	if s.SonnebornBergerMode != SBModeProportional && s.SonnebornBergerMode != SBModeTraditional {
		errors = append(errors, ScoringValidationError{
			Field:   "sonnebornBergerMode",
			Message: "invalid Sonneborn-Berger mode, must be 'proportional' or 'traditional'",
		})
	}

	// Validate preset matches values
	if s.Preset == ScoringPresetStandard {
		if s.WinPoints != 1.0 || s.DrawPoints != 0.5 || s.LossPoints != 0.0 {
			errors = append(errors, ScoringValidationError{
				Field:   "preset",
				Message: "standard preset must use 1-0.5-0 scoring",
			})
		}
	} else if s.Preset == ScoringPresetFootball {
		if s.WinPoints != 3.0 || s.DrawPoints != 1.0 || s.LossPoints != 0.0 {
			errors = append(errors, ScoringValidationError{
				Field:   "preset",
				Message: "football preset must use 3-1-0 scoring",
			})
		}
	}

	return errors
}

// IsValid returns true if the scoring settings pass validation
func (s *ScoringSettings) IsValid() bool {
	return len(s.Validate()) == 0
}

// ValidateFIDECompliance checks settings against FIDE rules and returns violations
func (s *ScoringSettings) ValidateFIDECompliance() []FIDEViolation {
	var violations []FIDEViolation

	// FIDE uses standard 1-0.5-0 scoring
	if s.WinPoints != 1.0 {
		violations = append(violations, FIDEViolation{
			Setting:    "winPoints",
			Value:      s.WinPoints,
			FIDEValue:  1.0,
			Severity:   "warning",
			Message:    "FIDE uses 1 point for a win",
			Regulation: "FIDE Laws of Chess 11.1",
		})
	}

	if s.DrawPoints != 0.5 {
		violations = append(violations, FIDEViolation{
			Setting:    "drawPoints",
			Value:      s.DrawPoints,
			FIDEValue:  0.5,
			Severity:   "warning",
			Message:    "FIDE uses 0.5 points for a draw",
			Regulation: "FIDE Laws of Chess 11.1",
		})
	}

	if s.LossPoints != 0.0 {
		violations = append(violations, FIDEViolation{
			Setting:    "lossPoints",
			Value:      s.LossPoints,
			FIDEValue:  0.0,
			Severity:   "warning",
			Message:    "FIDE uses 0 points for a loss",
			Regulation: "FIDE Laws of Chess 11.1",
		})
	}

	return violations
}

// IsStandardScoring returns true if using FIDE standard 1-0.5-0 scoring
func (s *ScoringSettings) IsStandardScoring() bool {
	return s.WinPoints == 1.0 && s.DrawPoints == 0.5 && s.LossPoints == 0.0
}

// RegistrationAccessMode defines how players can access tournament registration
type RegistrationAccessMode string

const (
	AccessModePublic     RegistrationAccessMode = "public"      // Anyone with slug can access
	AccessModeInviteOnly RegistrationAccessMode = "invite_only" // Requires invite token
)

// CheckInMethod defines how players can check in
type CheckInMethod string

const (
	CheckInMethodSelfService CheckInMethod = "self_service" // Player clicks check-in link
	CheckInMethodQRCode      CheckInMethod = "qr_code"      // Player scans QR at venue
	CheckInMethodOrganizer   CheckInMethod = "organizer"    // Organizer confirms manually
)

// RegistrationSettings configures self-service pre-registration and check-in
type RegistrationSettings struct {
	PreRegistration PreRegistrationSettings `json:"preRegistration"`
	CheckIn         CheckInSettings         `json:"checkIn"`
	Access          AccessSettings          `json:"access"`
}

// PreRegistrationSettings configures player self-registration behavior
type PreRegistrationSettings struct {
	Enabled          bool `json:"enabled"`          // Allow players to pre-register via link
	RequiresApproval bool `json:"requiresApproval"` // Organizer must approve each registration
	AllowGuests      bool `json:"allowGuests"`      // Allow users without accounts to register
	NotifyOrganizer  bool `json:"notifyOrganizer"`  // Email organizer on new registration
}

// CheckInSettings configures player check-in at the venue
type CheckInSettings struct {
	Enabled            bool     `json:"enabled"`            // Enable check-in functionality
	Methods            []string `json:"methods"`            // Allowed check-in methods: self_service, qr_code, organizer
	WindowOpenMinutes  int      `json:"windowOpenMinutes"`  // Minutes before tournament start to open check-in
	WindowCloseMinutes int      `json:"windowCloseMinutes"` // Minutes after tournament start to close check-in
	AutoNoShow         bool     `json:"autoNoShow"`         // Automatically mark non-checked-in as no-show
}

// AccessSettings configures how players can access the tournament
type AccessSettings struct {
	Mode string `json:"mode"` // "public" or "invite_only"
}

// DefaultRegistrationSettings returns sensible defaults for self-registration
func DefaultRegistrationSettings() *RegistrationSettings {
	return &RegistrationSettings{
		PreRegistration: PreRegistrationSettings{
			Enabled:          true,
			RequiresApproval: false,
			AllowGuests:      true,
			NotifyOrganizer:  true,
		},
		CheckIn: CheckInSettings{
			Enabled:            true,
			Methods:            []string{"self_service", "organizer"},
			WindowOpenMinutes:  120, // 2 hours before
			WindowCloseMinutes: 15,  // 15 minutes after start
			AutoNoShow:         false,
		},
		Access: AccessSettings{
			Mode: "public",
		},
	}
}

// RegistrationValidationError represents a validation error for registration settings
type RegistrationValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validate checks if the registration settings are valid
func (s *RegistrationSettings) Validate() []RegistrationValidationError {
	var errors []RegistrationValidationError

	// Validate check-in window
	if s.CheckIn.WindowOpenMinutes < 0 {
		errors = append(errors, RegistrationValidationError{
			Field:   "checkIn.windowOpenMinutes",
			Message: "window open minutes cannot be negative",
		})
	}

	if s.CheckIn.WindowCloseMinutes < -60 {
		errors = append(errors, RegistrationValidationError{
			Field:   "checkIn.windowCloseMinutes",
			Message: "window close cannot be more than 60 minutes before start",
		})
	}

	// Validate check-in methods if check-in is enabled
	if s.CheckIn.Enabled && len(s.CheckIn.Methods) == 0 {
		errors = append(errors, RegistrationValidationError{
			Field:   "checkIn.methods",
			Message: "at least one check-in method must be enabled",
		})
	}

	// Validate method values
	validMethods := map[string]bool{
		"self_service": true,
		"qr_code":      true,
		"organizer":    true,
	}
	for _, method := range s.CheckIn.Methods {
		if !validMethods[method] {
			errors = append(errors, RegistrationValidationError{
				Field:   "checkIn.methods",
				Message: "invalid check-in method: " + method,
			})
		}
	}

	// Validate access mode
	if s.Access.Mode != "public" && s.Access.Mode != "invite_only" {
		errors = append(errors, RegistrationValidationError{
			Field:   "access.mode",
			Message: "access mode must be 'public' or 'invite_only'",
		})
	}

	return errors
}

// IsValid returns true if the registration settings pass validation
func (s *RegistrationSettings) IsValid() bool {
	return len(s.Validate()) == 0
}

// IsPreRegistrationOpen checks if pre-registration is currently available
func (s *RegistrationSettings) IsPreRegistrationOpen() bool {
	return s.PreRegistration.Enabled
}

// HasCheckInMethod returns true if the specified check-in method is allowed
func (s *RegistrationSettings) HasCheckInMethod(method string) bool {
	for _, m := range s.CheckIn.Methods {
		if m == method {
			return true
		}
	}
	return false
}

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
	StartDate          time.Time  `json:"startDate" db:"start_date"`
	EndDate            *time.Time `json:"endDate,omitempty" db:"end_date"`
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

	// Rating Lock
	RatingsLockedAt *time.Time `json:"ratingsLockedAt,omitempty" db:"ratings_locked_at"`

	// Tiebreaks (JSON array)
	Tiebreaks []string `json:"tiebreaks" db:"tiebreaks"`

	// Bye Settings (JSON) - configurable bye, forfeit, withdrawal rules
	ByeSettings *ByeSettings `json:"byeSettings,omitempty" db:"bye_settings"`

	// Scoring Settings (JSON) - configurable point values and SB mode
	ScoringSettings *ScoringSettings `json:"scoringSettings,omitempty" db:"scoring_settings"`

	// Registration Settings (JSON) - self-registration and check-in configuration
	RegistrationSettings *RegistrationSettings `json:"registrationSettings,omitempty" db:"registration_settings"`

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
	StartRank            int  `json:"startRank" db:"start_rank"`
	StartingRating       int  `json:"startingRating" db:"starting_rating"`
	ManualRatingOverride *int `json:"manualRatingOverride,omitempty" db:"manual_rating_override"`

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
	ByeRequests []int    `json:"byeRequests" db:"bye_requests"` // Rounds where half-point bye requested

	// Bye Tracking
	HalfPointByesUsed  int   `json:"halfPointByesUsed" db:"half_point_byes_used"`   // Count of half-point byes taken
	ZeroPointByesUsed  int   `json:"zeroPointByesUsed" db:"zero_point_byes_used"`   // Count of zero-point byes taken
	SystemByeRound     *int  `json:"systemByeRound,omitempty" db:"system_bye_round"` // Round where system bye received (null if none)
	ForfeitCount       int   `json:"forfeitCount" db:"forfeit_count"`               // Total forfeits
	ConsecutiveForfeits int  `json:"consecutiveForfeits" db:"consecutive_forfeits"` // Current consecutive forfeit streak
	LateEntryRound     *int  `json:"lateEntryRound,omitempty" db:"late_entry_round"` // Round joined (null if joined before start)
	WithdrawnRound     *int  `json:"withdrawnRound,omitempty" db:"withdrawn_round"`  // Round withdrawn (null if still active)
	WithdrawalReason   *string `json:"withdrawalReason,omitempty" db:"withdrawal_reason"` // Reason for withdrawal

	// Payment
	PaymentStatus    string  `json:"paymentStatus" db:"payment_status"`
	PaymentReference *string `json:"paymentReference,omitempty" db:"payment_reference"`

	// Self-registration and Check-in
	PreRegisteredAt    *time.Time `json:"preRegisteredAt,omitempty" db:"pre_registered_at"`
	ApprovedAt         *time.Time `json:"approvedAt,omitempty" db:"approved_at"`
	ApprovedBy         *uuid.UUID `json:"approvedBy,omitempty" db:"approved_by"`
	CheckedInAt        *time.Time `json:"checkedInAt,omitempty" db:"checked_in_at"`
	CheckedInBy        *uuid.UUID `json:"checkedInBy,omitempty" db:"checked_in_by"`
	CheckInMethod      *string    `json:"checkInMethod,omitempty" db:"check_in_method"`
	CheckInCode        *string    `json:"checkInCode,omitempty" db:"check_in_code"`
	RegistrationSource string     `json:"registrationSource" db:"registration_source"` // organizer, self_service, guest_converted

	// Player info (joined)
	PlayerFirstName *string `json:"playerFirstName,omitempty" db:"player_first_name"`
	PlayerLastName  *string `json:"playerLastName,omitempty" db:"player_last_name"`
	PlayerFideID    *string `json:"playerFideId,omitempty" db:"player_fide_id"`
	PlayerClubName  *string `json:"playerClubName,omitempty" db:"player_club_name"`
}

// EffectiveRating returns the rating to use for pairing.
// Returns ManualRatingOverride if set, otherwise StartingRating.
func (p *Participant) EffectiveRating() int {
	if p.ManualRatingOverride != nil {
		return *p.ManualRatingOverride
	}
	return p.StartingRating
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

// SpecialRanking represents a special ranking category (Sonderwertung) for prize eligibility
type SpecialRanking struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TournamentID uuid.UUID `json:"tournamentId" db:"tournament_id"`
	Name         string    `json:"name" db:"name"`               // Internal code: "u12", "seniors", "women"
	DisplayName  string    `json:"displayName" db:"display_name"` // "U12 Wertung", "Seniorenwertung"

	// Filter criteria (nil = no filter on this criterion)
	MinBirthYear *int    `json:"minBirthYear,omitempty" db:"min_birth_year"`
	MaxBirthYear *int    `json:"maxBirthYear,omitempty" db:"max_birth_year"`
	MinRating    *int    `json:"minRating,omitempty" db:"min_rating"`
	MaxRating    *int    `json:"maxRating,omitempty" db:"max_rating"`
	Gender       *string `json:"gender,omitempty" db:"gender"`

	SortOrder int  `json:"sortOrder" db:"sort_order"`
	IsActive  bool `json:"isActive" db:"is_active"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// IsEligible checks if a player qualifies for this special ranking
func (sr *SpecialRanking) IsEligible(birthYear int, rating int, gender string) bool {
	if sr.MinBirthYear != nil && birthYear < *sr.MinBirthYear {
		return false
	}
	if sr.MaxBirthYear != nil && birthYear > *sr.MaxBirthYear {
		return false
	}
	if sr.MinRating != nil && rating < *sr.MinRating {
		return false
	}
	if sr.MaxRating != nil && rating > *sr.MaxRating {
		return false
	}
	if sr.Gender != nil && *sr.Gender != "" && gender != *sr.Gender {
		return false
	}
	return true
}

// Team represents a team in a team tournament
type Team struct {
	ID           int       `json:"id" db:"id"`
	TournamentID uuid.UUID `json:"tournamentId" db:"tournament_id"`
	Name         string    `json:"name" db:"name"`
	Points       float64   `json:"points" db:"points"`
	Players      []TeamPlayer `json:"players"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

// TeamPlayer represents a player on a team
type TeamPlayer struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TeamID      int       `json:"teamId" db:"team_id"`
	PlayerID    uuid.UUID `json:"playerId" db:"player_id"`
	StartRank   int       `json:"startRank" db:"start_rank"`
	Name        string    `json:"name" db:"name"`
	Rating      int       `json:"rating" db:"rating"`
	BoardNumber int       `json:"boardNumber" db:"board_number"`
	IsActive    bool      `json:"isActive" db:"is_active"`
}

// Request/Response types

// CreateTournamentRequest is the request body for creating a tournament
type CreateTournamentRequest struct {
	OrganizationID     *uuid.UUID      `json:"organizationId,omitempty"`
	Name               string          `json:"name" binding:"required,min=1,max=255"`
	Description        *string         `json:"description,omitempty"`
	Slug               string          `json:"slug,omitempty"`
	Type               TournamentType  `json:"type" binding:"required,oneof=swiss round_robin knockout team_swiss scheveningen arena"`
	Mode               TournamentMode  `json:"mode" binding:"required,oneof=otb online hybrid"`
	TimeControl        string          `json:"timeControl" binding:"required"`
	TimeControlType    TimeControlType `json:"timeControlType" binding:"required,oneof=standard rapid blitz"`
	PairingEngine      PairingEngine   `json:"pairingEngine" binding:"omitempty,oneof=javafo bbp round_robin knockout team_swiss monrad keizer arena"`
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

// CloneTournamentRequest is the request body for cloning a tournament
type CloneTournamentRequest struct {
	Name               string     `json:"name" binding:"required,min=1,max=255"`
	Description        *string    `json:"description,omitempty"`
	StartDate          *string    `json:"startDate,omitempty"`
	EndDate            *string    `json:"endDate,omitempty"`
	RegistrationOpens  *time.Time `json:"registrationOpens,omitempty"`
	RegistrationCloses *time.Time `json:"registrationCloses,omitempty"`
}

// UpdateTournamentRequest is the request body for updating a tournament
type UpdateTournamentRequest struct {
	Name               *string           `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description        *string           `json:"description,omitempty"`
	TimeControl        *string           `json:"timeControl,omitempty"`
	Rounds             *int              `json:"rounds,omitempty" binding:"omitempty,min=1,max=15"`
	StartDate          *string           `json:"startDate,omitempty"`
	EndDate            *string           `json:"endDate,omitempty"`
	MaxPlayers         *int              `json:"maxPlayers,omitempty"`
	RegistrationOpens  *time.Time        `json:"registrationOpens,omitempty"`
	RegistrationCloses *time.Time        `json:"registrationCloses,omitempty"`
	FideRated          *bool             `json:"fideRated,omitempty"`
	DwzRated           *bool             `json:"dwzRated,omitempty"`
	AcceleratedPairing *bool             `json:"acceleratedPairing,omitempty"`
	Tiebreaks          []string          `json:"tiebreaks,omitempty"`
	VenueName          *string           `json:"venueName,omitempty"`
	VenueAddress       *string           `json:"venueAddress,omitempty"`
	VenueCity          *string           `json:"venueCity,omitempty"`
	Status             *TournamentStatus `json:"status,omitempty"`
	ScoringSettings    *ScoringSettings  `json:"scoringSettings,omitempty"`
}

// RegisterParticipantRequest is the request for registering a participant
type RegisterParticipantRequest struct {
	PlayerID       uuid.UUID  `json:"playerId"`
	SectionID      *uuid.UUID `json:"sectionId,omitempty"`
	StartingRating int        `json:"startingRating,omitempty"`

	// Federation import fields - used when player doesn't exist locally
	DsbID  string `json:"dsbId,omitempty"`  // PKZ for importing from DSB/portal64
	OesbID string `json:"oesbId,omitempty"` // OESB ID for importing from Austrian federation
	SsbID  string `json:"ssbId,omitempty"`  // SSB ID for importing from Swiss federation

	// Manual entry fields - fallback when federation APIs unavailable
	ManualEntry     bool    `json:"manualEntry,omitempty"`
	ManualFirstName string  `json:"manualFirstName,omitempty"`
	ManualLastName  string  `json:"manualLastName,omitempty"`
	ManualDWZ       *int    `json:"manualDwz,omitempty"`
	ManualFideID    string  `json:"manualFideId,omitempty"`
	ManualClub      string  `json:"manualClub,omitempty"`
	ManualBirthYear *int    `json:"manualBirthYear,omitempty"`
	ManualGender    string  `json:"manualGender,omitempty"`
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
