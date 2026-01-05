package registration

import (
	"time"

	"github.com/google/uuid"
)

// Registration status constants
const (
	StatusPreRegistered   = "pre_registered"   // Initial state after self-registration
	StatusPendingApproval = "pending_approval" // Awaiting organizer approval
	StatusApproved        = "approved"         // Approved, not yet checked in
	StatusCheckedIn       = "checked_in"       // Confirmed presence at venue
	StatusConverted       = "converted"        // Guest converted to participant
	StatusRejected        = "rejected"         // Rejected by organizer
	StatusExpired         = "expired"          // Registration expired
	StatusNoShow          = "no_show"          // Did not check in
)

// Registration source constants
const (
	SourceOrganizer      = "organizer"       // Added by organizer
	SourceSelfService    = "self_service"    // Self-registered
	SourceGuestConverted = "guest_converted" // Converted from guest registration
)

// GuestRegistration represents a guest pre-registration (for users without accounts)
type GuestRegistration struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TournamentID uuid.UUID `json:"tournamentId" db:"tournament_id"`

	// Guest identification
	Email     string `json:"email" db:"email"`
	FirstName string `json:"firstName" db:"first_name"`
	LastName  string `json:"lastName" db:"last_name"`

	// Optional federation IDs for lookup
	FideID *string `json:"fideId,omitempty" db:"fide_id"`
	DsbID  *string `json:"dsbId,omitempty" db:"dsb_id"`
	OesbID *string `json:"oesbId,omitempty" db:"oesb_id"`
	SsbID  *string `json:"ssbId,omitempty" db:"ssb_id"`

	// Resolved player (after federation lookup or manual approval)
	PlayerID *uuid.UUID `json:"playerId,omitempty" db:"player_id"`

	// Registration flow
	RegistrationStatus string `json:"registrationStatus" db:"registration_status"`

	// Check-in tracking
	CheckInCode   *string    `json:"checkInCode,omitempty" db:"check_in_code"`
	CheckedInAt   *time.Time `json:"checkedInAt,omitempty" db:"checked_in_at"`
	CheckedInBy   *uuid.UUID `json:"checkedInBy,omitempty" db:"checked_in_by"`
	CheckInMethod *string    `json:"checkInMethod,omitempty" db:"check_in_method"`

	// Approval tracking
	ApprovedAt      *time.Time `json:"approvedAt,omitempty" db:"approved_at"`
	ApprovedBy      *uuid.UUID `json:"approvedBy,omitempty" db:"approved_by"`
	RejectionReason *string    `json:"rejectionReason,omitempty" db:"rejection_reason"`

	// Metadata
	Notes *string `json:"notes,omitempty" db:"notes"`

	// Timestamps
	PreRegisteredAt time.Time `json:"preRegisteredAt" db:"pre_registered_at"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

// InviteToken represents a tournament invite token for private tournaments
type InviteToken struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TournamentID uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	TokenHash    string     `json:"-" db:"token_hash"` // Never expose hash
	Name         *string    `json:"name,omitempty" db:"name"`
	MaxUses      *int       `json:"maxUses,omitempty" db:"max_uses"`
	UsesCount    int        `json:"usesCount" db:"uses_count"`
	ValidUntil   *time.Time `json:"validUntil,omitempty" db:"valid_until"`
	CreatedBy    uuid.UUID  `json:"createdBy" db:"created_by"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`

	// Computed field for API response (the actual token, not hash)
	Token string `json:"token,omitempty" db:"-"`
}

// TournamentPublicInfo contains public tournament information for the registration page
type TournamentPublicInfo struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Description          *string    `json:"description,omitempty"`
	Slug                 string     `json:"slug"`
	StartDate            time.Time  `json:"startDate"`
	EndDate              *time.Time `json:"endDate,omitempty"`
	VenueName            *string    `json:"venueName,omitempty"`
	VenueCity            *string    `json:"venueCity,omitempty"`
	VenueCountry         *string    `json:"venueCountry,omitempty"`
	MaxPlayers           *int       `json:"maxPlayers,omitempty"`
	ParticipantCount     int        `json:"participantCount"`
	Status               string     `json:"status"`
	RegistrationOpens    *time.Time `json:"registrationOpens,omitempty"`
	RegistrationCloses   *time.Time `json:"registrationCloses,omitempty"`
	EntryFeeCents        int        `json:"entryFeeCents"`
	Currency             string     `json:"currency"`
	RegistrationSettings interface{} `json:"registrationSettings,omitempty"`
}

// CheckInStatus represents the check-in status for a tournament
type CheckInStatus struct {
	TotalRegistrations int  `json:"totalRegistrations"`
	CheckedIn          int  `json:"checkedIn"`
	NotCheckedIn       int  `json:"notCheckedIn"`
	PendingApproval    int  `json:"pendingApproval"`
	NoShows            int  `json:"noShows"`
	IsWindowOpen       bool `json:"isWindowOpen"`
}

// RegistrationListItem represents a combined view of all registrations (guests and participants)
type RegistrationListItem struct {
	ID                 uuid.UUID  `json:"id"`
	Type               string     `json:"type"` // "guest" or "participant"
	Email              string     `json:"email,omitempty"`
	FirstName          string     `json:"firstName"`
	LastName           string     `json:"lastName"`
	FideID             *string    `json:"fideId,omitempty"`
	RegistrationStatus string     `json:"registrationStatus"`
	PreRegisteredAt    *time.Time `json:"preRegisteredAt,omitempty"`
	ApprovedAt         *time.Time `json:"approvedAt,omitempty"`
	CheckedInAt        *time.Time `json:"checkedInAt,omitempty"`
	CheckInMethod      *string    `json:"checkInMethod,omitempty"`
	RegistrationSource string     `json:"registrationSource,omitempty"`
}

// --- Request Types ---

// PreRegisterRequest is the request body for pre-registration
type PreRegisterRequest struct {
	// For authenticated users with linked player
	PlayerID *uuid.UUID `json:"playerId,omitempty"`

	// For guests (required if no PlayerID)
	Email     string `json:"email" binding:"required_without=PlayerId,omitempty,email"`
	FirstName string `json:"firstName" binding:"required_without=PlayerId,omitempty"`
	LastName  string `json:"lastName" binding:"required_without=PlayerId,omitempty"`

	// Optional federation ID for lookup
	FederationID   string `json:"federationId,omitempty"`
	FederationType string `json:"federationType,omitempty"` // dsb, oesb, ssb, fide

	// For private tournaments
	InviteToken string `json:"inviteToken,omitempty"`
}

// CheckInRequest is the request body for check-in
type CheckInRequest struct {
	Code   string `json:"code,omitempty"`   // Check-in code (for QR or manual)
	Method string `json:"method,omitempty"` // self_service, qr_code
}

// ApproveRegistrationRequest is the request body for approving/rejecting registration
type ApproveRegistrationRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"` // Required if rejecting
}

// BulkCheckInRequest is the request body for bulk check-in
type BulkCheckInRequest struct {
	RegistrationIDs []uuid.UUID `json:"registrationIds" binding:"required"`
}

// CreateInviteTokenRequest is the request body for creating invite tokens
type CreateInviteTokenRequest struct {
	Name       string     `json:"name,omitempty"`
	MaxUses    *int       `json:"maxUses,omitempty"`
	ValidUntil *time.Time `json:"validUntil,omitempty"`
}

// UpdateRegistrationSettingsRequest wraps registration settings update
type UpdateRegistrationSettingsRequest struct {
	PreRegistration *PreRegistrationSettingsUpdate `json:"preRegistration,omitempty"`
	CheckIn         *CheckInSettingsUpdate         `json:"checkIn,omitempty"`
	Access          *AccessSettingsUpdate          `json:"access,omitempty"`
}

// PreRegistrationSettingsUpdate for partial updates
type PreRegistrationSettingsUpdate struct {
	Enabled          *bool `json:"enabled,omitempty"`
	RequiresApproval *bool `json:"requiresApproval,omitempty"`
	AllowGuests      *bool `json:"allowGuests,omitempty"`
	NotifyOrganizer  *bool `json:"notifyOrganizer,omitempty"`
}

// CheckInSettingsUpdate for partial updates
type CheckInSettingsUpdate struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	Methods            []string `json:"methods,omitempty"`
	WindowOpenMinutes  *int     `json:"windowOpenMinutes,omitempty"`
	WindowCloseMinutes *int     `json:"windowCloseMinutes,omitempty"`
	AutoNoShow         *bool    `json:"autoNoShow,omitempty"`
}

// AccessSettingsUpdate for partial updates
type AccessSettingsUpdate struct {
	Mode *string `json:"mode,omitempty"`
}

// --- Response Types ---

// RegistrationListResponse wraps the list of registrations
type RegistrationListResponse struct {
	Registrations []RegistrationListItem `json:"registrations"`
	Total         int                    `json:"total"`
	Tournament    *TournamentPublicInfo  `json:"tournament,omitempty"`
}

// PreRegisterResponse is the response after successful pre-registration
type PreRegisterResponse struct {
	ID                 uuid.UUID `json:"id"`
	Status             string    `json:"status"`
	CheckInCode        string    `json:"checkInCode,omitempty"`
	RequiresApproval   bool      `json:"requiresApproval"`
	Message            string    `json:"message"`
	ConfirmationNeeded bool      `json:"confirmationNeeded,omitempty"`
}

// CheckInResponse is the response after successful check-in
type CheckInResponse struct {
	Success   bool      `json:"success"`
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	CheckedIn time.Time `json:"checkedIn"`
	Message   string    `json:"message"`
}

// MyRegistrationResponse is the response for user's own registration status
type MyRegistrationResponse struct {
	Found              bool       `json:"found"`
	Type               string     `json:"type,omitempty"` // "guest" or "participant"
	ID                 *uuid.UUID `json:"id,omitempty"`
	Status             string     `json:"status,omitempty"`
	PreRegisteredAt    *time.Time `json:"preRegisteredAt,omitempty"`
	ApprovedAt         *time.Time `json:"approvedAt,omitempty"`
	CheckedInAt        *time.Time `json:"checkedInAt,omitempty"`
	CheckInCode        *string    `json:"checkInCode,omitempty"`
	IsCheckInOpen      bool       `json:"isCheckInOpen"`
	CheckInWindowStart *time.Time `json:"checkInWindowStart,omitempty"`
	CheckInWindowEnd   *time.Time `json:"checkInWindowEnd,omitempty"`
}
