package audit

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of action being logged
type AuditAction string

const (
	// User actions
	ActionUserLogin          AuditAction = "user.login"
	ActionUserLogout         AuditAction = "user.logout"
	ActionUserRegistered     AuditAction = "user.registered"
	ActionUserUpdated        AuditAction = "user.updated"
	ActionUserDeleted        AuditAction = "user.deleted"
	ActionUserRoleChanged    AuditAction = "user.role_changed"
	ActionPasswordReset      AuditAction = "user.password_reset"
	ActionPasswordChanged    AuditAction = "user.password_changed"

	// Tournament actions
	ActionTournamentCreated   AuditAction = "tournament.created"
	ActionTournamentUpdated   AuditAction = "tournament.updated"
	ActionTournamentDeleted   AuditAction = "tournament.deleted"
	ActionTournamentStarted   AuditAction = "tournament.started"
	ActionTournamentFinished  AuditAction = "tournament.finished"
	ActionTournamentCancelled AuditAction = "tournament.cancelled"

	// Round actions
	ActionRoundCreated    AuditAction = "round.created"
	ActionRoundPaired     AuditAction = "round.paired"
	ActionRoundStarted    AuditAction = "round.started"
	ActionRoundCompleted  AuditAction = "round.completed"

	// Game actions
	ActionGameResultEntered   AuditAction = "game.result_entered"
	ActionGameResultModified  AuditAction = "game.result_modified"
	ActionGameForfeited       AuditAction = "game.forfeited"

	// Player actions
	ActionPlayerCreated       AuditAction = "player.created"
	ActionPlayerUpdated       AuditAction = "player.updated"
	ActionPlayerRegistered    AuditAction = "player.registered"
	ActionPlayerWithdrawn     AuditAction = "player.withdrawn"

	// Team actions
	ActionTeamCreated       AuditAction = "team.created"
	ActionTeamUpdated       AuditAction = "team.updated"
	ActionLineupSubmitted   AuditAction = "team.lineup_submitted"
	ActionLineupModified    AuditAction = "team.lineup_modified"

	// Club actions
	ActionClubCreated AuditAction = "club.created"
	ActionClubUpdated AuditAction = "club.updated"
	ActionClubDeleted AuditAction = "club.deleted"

	// Admin actions
	ActionAdminSettingsChanged AuditAction = "admin.settings_changed"
	ActionAdminExport          AuditAction = "admin.export"
	ActionAdminImport          AuditAction = "admin.import"
)

// AuditLog represents a single audit log entry
type AuditLog struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	UserID       *uuid.UUID             `json:"userId,omitempty" db:"user_id"`
	Action       AuditAction            `json:"action" db:"action"`
	EntityType   string                 `json:"entityType" db:"entity_type"`
	EntityID     *uuid.UUID             `json:"entityId,omitempty" db:"entity_id"`
	OldValue     map[string]interface{} `json:"oldValue,omitempty" db:"old_value"`
	NewValue     map[string]interface{} `json:"newValue,omitempty" db:"new_value"`
	IPAddress    string                 `json:"ipAddress,omitempty" db:"ip_address"`
	UserAgent    string                 `json:"userAgent,omitempty" db:"user_agent"`
	Description  string                 `json:"description,omitempty" db:"description"`
	CreatedAt    time.Time              `json:"createdAt" db:"created_at"`
}

// AuditLogListParams contains parameters for listing audit logs
type AuditLogListParams struct {
	UserID     *uuid.UUID  `form:"userId"`
	Action     AuditAction `form:"action"`
	EntityType string      `form:"entityType"`
	EntityID   *uuid.UUID  `form:"entityId"`
	StartDate  *time.Time  `form:"startDate"`
	EndDate    *time.Time  `form:"endDate"`
	Page       int         `form:"page,default=1"`
	PageSize   int         `form:"pageSize,default=50"`
}

// AuditLogListResponse contains a paginated list of audit logs
type AuditLogListResponse struct {
	Logs       []AuditLog `json:"logs"`
	TotalCount int        `json:"totalCount"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalPages int        `json:"totalPages"`
}

// CreateAuditLogRequest represents a request to create an audit log
type CreateAuditLogRequest struct {
	UserID      *uuid.UUID             `json:"userId"`
	Action      AuditAction            `json:"action"`
	EntityType  string                 `json:"entityType"`
	EntityID    *uuid.UUID             `json:"entityId"`
	OldValue    map[string]interface{} `json:"oldValue,omitempty"`
	NewValue    map[string]interface{} `json:"newValue,omitempty"`
	IPAddress   string                 `json:"ipAddress"`
	UserAgent   string                 `json:"userAgent"`
	Description string                 `json:"description"`
}
