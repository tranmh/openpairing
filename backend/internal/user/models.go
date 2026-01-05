package user

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the user's role in the system
type UserRole string

const (
	RolePlayer    UserRole = "player"
	RoleArbiter   UserRole = "arbiter"
	RoleOrganizer UserRole = "organizer"
	RoleAdmin     UserRole = "admin"
)

// User represents a user account
type User struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	Email             string     `json:"email" db:"email"`
	Role              UserRole   `json:"role" db:"role"`
	PreferredLanguage string     `json:"preferredLanguage" db:"preferred_language"`
	EmailVerified     bool       `json:"emailVerified" db:"email_verified"`
	EmailVerifiedAt   *time.Time `json:"emailVerifiedAt,omitempty" db:"email_verified_at"`
	LastLoginAt       *time.Time `json:"lastLoginAt,omitempty" db:"last_login_at"`
	DisplayName       *string    `json:"displayName,omitempty" db:"display_name"`
	CreatedAt         time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time  `json:"updatedAt" db:"updated_at"`
}

// UserProfile combines user account with linked player profile
type UserProfile struct {
	User
	PlayerID    *uuid.UUID `json:"playerId,omitempty" db:"player_id"`
	FirstName   *string    `json:"firstName,omitempty" db:"first_name"`
	LastName    *string    `json:"lastName,omitempty" db:"last_name"`
	FideID      *string    `json:"fideId,omitempty" db:"fide_id"`
	DsbID       *string    `json:"dsbId,omitempty" db:"dsb_id"`
	OesbID      *string    `json:"oesbId,omitempty" db:"oesb_id"`
	SsbID       *string    `json:"ssbId,omitempty" db:"ssb_id"`
	FideRating  *int       `json:"fideRating,omitempty" db:"fide_standard"`
	DwzRating   *int       `json:"dwzRating,omitempty" db:"dwz"`
	FideTitle   *string    `json:"fideTitle,omitempty" db:"fide_title"`
	ClubID      *uuid.UUID `json:"clubId,omitempty" db:"club_id"`
	ClubName    *string    `json:"clubName,omitempty" db:"club_name"`
	Federation  *string    `json:"federation,omitempty" db:"federation"`
}

// UpdateProfileRequest is the request to update user profile
type UpdateProfileRequest struct {
	DisplayName       *string `json:"displayName,omitempty" binding:"omitempty,max=100"`
	PreferredLanguage *string `json:"preferredLanguage,omitempty" binding:"omitempty,oneof=de en"`
	FirstName         *string `json:"firstName,omitempty" binding:"omitempty,max=100"`
	LastName          *string `json:"lastName,omitempty" binding:"omitempty,max=100"`
}

// ChangePasswordRequest is the request to change password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,min=8"`
	NewPassword     string `json:"newPassword" binding:"required,min=8"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,eqfield=NewPassword"`
}

// LinkPlayerRequest is the request to link a player profile to the user
type LinkPlayerRequest struct {
	PlayerID uuid.UUID `json:"playerId" binding:"required"`
}

// UserSession represents an active user session
type UserSession struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"`
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`
	RevokedAt *time.Time `json:"revokedAt,omitempty" db:"revoked_at"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UserAgent *string   `json:"userAgent,omitempty" db:"user_agent"`
	IPAddress *string   `json:"ipAddress,omitempty" db:"ip_address"`
}

// UserPreferences represents user preferences
type UserPreferences struct {
	Language         string `json:"language"`
	Theme            string `json:"theme"`
	NotifyByEmail    bool   `json:"notifyByEmail"`
	NotifyTournament bool   `json:"notifyTournament"`
	NotifyResults    bool   `json:"notifyResults"`
}

// UpdatePreferencesRequest is the request to update preferences
type UpdatePreferencesRequest struct {
	Language         *string `json:"language,omitempty" binding:"omitempty,oneof=de en"`
	Theme            *string `json:"theme,omitempty" binding:"omitempty,oneof=light dark system"`
	NotifyByEmail    *bool   `json:"notifyByEmail,omitempty"`
	NotifyTournament *bool   `json:"notifyTournament,omitempty"`
	NotifyResults    *bool   `json:"notifyResults,omitempty"`
}

// UserStats represents user statistics
type UserStats struct {
	TournamentsPlayed  int     `json:"tournamentsPlayed"`
	GamesPlayed        int     `json:"gamesPlayed"`
	Wins               int     `json:"wins"`
	Draws              int     `json:"draws"`
	Losses             int     `json:"losses"`
	WinRate            float64 `json:"winRate"`
	AveragePerformance *int    `json:"averagePerformance,omitempty"`
}
