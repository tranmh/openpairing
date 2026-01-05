package fairplay

import (
	"time"

	"github.com/google/uuid"
)

// ViolationType represents the type of fair play violation
type ViolationType string

const (
	ViolationSuspiciousPlay   ViolationType = "suspicious_play"
	ViolationRatingManipulation ViolationType = "rating_manipulation"
	ViolationCollusion        ViolationType = "collusion"
	ViolationTimestalling     ViolationType = "time_stalling"
	ViolationUnsportsmanlike  ViolationType = "unsportsmanlike"
	ViolationOther            ViolationType = "other"
)

// SeverityLevel represents the severity of a violation
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// ViolationStatus represents the status of a violation report
type ViolationStatus string

const (
	StatusPending    ViolationStatus = "pending"
	StatusReviewing  ViolationStatus = "reviewing"
	StatusConfirmed  ViolationStatus = "confirmed"
	StatusDismissed  ViolationStatus = "dismissed"
	StatusAppealed   ViolationStatus = "appealed"
)

// FairplayViolation represents a fair play violation report
type FairplayViolation struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	TournamentID  *uuid.UUID      `json:"tournamentId,omitempty" db:"tournament_id"`
	GameID        *uuid.UUID      `json:"gameId,omitempty" db:"game_id"`
	PlayerID      uuid.UUID       `json:"playerId" db:"player_id"`
	ReportedBy    *uuid.UUID      `json:"reportedBy,omitempty" db:"reported_by"`
	Type          ViolationType   `json:"type" db:"violation_type"`
	Severity      SeverityLevel   `json:"severity" db:"severity"`
	Status        ViolationStatus `json:"status" db:"status"`
	Description   string          `json:"description" db:"description"`
	Evidence      *string         `json:"evidence,omitempty" db:"evidence"`
	ReviewedBy    *uuid.UUID      `json:"reviewedBy,omitempty" db:"reviewed_by"`
	ReviewNotes   *string         `json:"reviewNotes,omitempty" db:"review_notes"`
	ActionTaken   *string         `json:"actionTaken,omitempty" db:"action_taken"`
	CreatedAt     time.Time       `json:"createdAt" db:"created_at"`
	ReviewedAt    *time.Time      `json:"reviewedAt,omitempty" db:"reviewed_at"`
}

// PlayerFlag represents a flag/note on a player for arbiters
type PlayerFlag struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	PlayerID    uuid.UUID   `json:"playerId" db:"player_id"`
	FlaggedBy   uuid.UUID   `json:"flaggedBy" db:"flagged_by"`
	FlagType    string      `json:"flagType" db:"flag_type"` // warning, watch, banned
	Reason      string      `json:"reason" db:"reason"`
	ExpiresAt   *time.Time  `json:"expiresAt,omitempty" db:"expires_at"`
	CreatedAt   time.Time   `json:"createdAt" db:"created_at"`
	RemovedAt   *time.Time  `json:"removedAt,omitempty" db:"removed_at"`
	RemovedBy   *uuid.UUID  `json:"removedBy,omitempty" db:"removed_by"`
}

// PerformanceAnalysis represents analysis of a player's performance
type PerformanceAnalysis struct {
	PlayerID           uuid.UUID `json:"playerId"`
	GamesAnalyzed      int       `json:"gamesAnalyzed"`
	AverageAccuracy    *float64  `json:"averageAccuracy,omitempty"`
	PerformanceRating  *int      `json:"performanceRating,omitempty"`
	RatingDeviation    float64   `json:"ratingDeviation"`
	SuspiciousPatterns []string  `json:"suspiciousPatterns,omitempty"`
	RiskScore          float64   `json:"riskScore"` // 0-100
}

// CreateViolationRequest is the request to create a violation report
type CreateViolationRequest struct {
	TournamentID *uuid.UUID    `json:"tournamentId,omitempty"`
	GameID       *uuid.UUID    `json:"gameId,omitempty"`
	PlayerID     uuid.UUID     `json:"playerId" binding:"required"`
	Type         ViolationType `json:"type" binding:"required"`
	Description  string        `json:"description" binding:"required,min=20,max=2000"`
	Evidence     *string       `json:"evidence,omitempty" binding:"omitempty,max=5000"`
}

// ReviewViolationRequest is the request to review a violation
type ReviewViolationRequest struct {
	Status      ViolationStatus `json:"status" binding:"required,oneof=confirmed dismissed"`
	ReviewNotes string          `json:"reviewNotes" binding:"required,min=10,max=2000"`
	ActionTaken *string         `json:"actionTaken,omitempty" binding:"omitempty,max=1000"`
}

// CreateFlagRequest is the request to flag a player
type CreateFlagRequest struct {
	PlayerID  uuid.UUID  `json:"playerId" binding:"required"`
	FlagType  string     `json:"flagType" binding:"required,oneof=warning watch banned"`
	Reason    string     `json:"reason" binding:"required,min=10,max=1000"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// ViolationListRequest is the request to list violations
type ViolationListRequest struct {
	TournamentID *uuid.UUID      `form:"tournamentId"`
	PlayerID     *uuid.UUID      `form:"playerId"`
	Status       ViolationStatus `form:"status"`
	Type         ViolationType   `form:"type"`
	Page         int             `form:"page,default=1" binding:"min=1"`
	Limit        int             `form:"limit,default=20" binding:"min=1,max=100"`
}

// ViolationListResponse is the paginated list of violations
type ViolationListResponse struct {
	Violations []FairplayViolation `json:"violations"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
}

// FlagListResponse is the list of player flags
type FlagListResponse struct {
	Flags []PlayerFlag `json:"flags"`
	Total int          `json:"total"`
}

// FairplayStats represents fair play statistics
type FairplayStats struct {
	TotalViolations     int `json:"totalViolations"`
	PendingViolations   int `json:"pendingViolations"`
	ConfirmedViolations int `json:"confirmedViolations"`
	ActiveFlags         int `json:"activeFlags"`
	PlayersWatched      int `json:"playersWatched"`
}
