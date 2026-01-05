package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service provides audit logging business logic
type Service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new audit service
func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("component", "audit_service").Logger(),
	}
}

// Log creates a new audit log entry
func (s *Service) Log(ctx context.Context, req *CreateAuditLogRequest) error {
	log := &AuditLog{
		UserID:      req.UserID,
		Action:      req.Action,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		OldValue:    req.OldValue,
		NewValue:    req.NewValue,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		Description: req.Description,
	}

	if err := s.repo.Create(ctx, log); err != nil {
		s.logger.Error().Err(err).
			Str("action", string(req.Action)).
			Str("entityType", req.EntityType).
			Msg("Failed to create audit log")
		return err
	}

	s.logger.Debug().
		Str("action", string(req.Action)).
		Str("entityType", req.EntityType).
		Msg("Audit log created")

	return nil
}

// LogAction is a convenience method for logging actions without the full request
func (s *Service) LogAction(ctx context.Context, userID *uuid.UUID, action AuditAction, entityType string, entityID *uuid.UUID, description string, ipAddress, userAgent string) error {
	return s.Log(ctx, &CreateAuditLogRequest{
		UserID:      userID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Description: description,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})
}

// LogWithValues logs an action with old and new values for change tracking
func (s *Service) LogWithValues(ctx context.Context, userID *uuid.UUID, action AuditAction, entityType string, entityID *uuid.UUID, oldValue, newValue map[string]interface{}, ipAddress, userAgent string) error {
	return s.Log(ctx, &CreateAuditLogRequest{
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		OldValue:   oldValue,
		NewValue:   newValue,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

// GetByID retrieves an audit log by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*AuditLog, error) {
	return s.repo.GetByID(ctx, id)
}

// List lists audit logs with pagination and filtering
func (s *Service) List(ctx context.Context, params AuditLogListParams) (*AuditLogListResponse, error) {
	return s.repo.List(ctx, params)
}

// GetEntityHistory retrieves the audit history for a specific entity
func (s *Service) GetEntityHistory(ctx context.Context, entityType string, entityID uuid.UUID, limit int) ([]AuditLog, error) {
	return s.repo.GetByEntityID(ctx, entityType, entityID, limit)
}

// GetUserActivity retrieves the activity log for a specific user
func (s *Service) GetUserActivity(ctx context.Context, userID uuid.UUID, limit int) ([]AuditLog, error) {
	return s.repo.GetByUserID(ctx, userID, limit)
}

// Cleanup removes old audit logs based on retention policy
func (s *Service) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	before := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := s.repo.DeleteOlderThan(ctx, before)
	if err != nil {
		return 0, err
	}

	s.logger.Info().
		Int64("deleted", deleted).
		Int("retentionDays", retentionDays).
		Msg("Cleaned up old audit logs")

	return deleted, nil
}

// Helper functions for common audit actions

// LogUserLogin logs a user login event
func (s *Service) LogUserLogin(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string) error {
	return s.LogAction(ctx, &userID, ActionUserLogin, "user", &userID, "User logged in", ipAddress, userAgent)
}

// LogUserLogout logs a user logout event
func (s *Service) LogUserLogout(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string) error {
	return s.LogAction(ctx, &userID, ActionUserLogout, "user", &userID, "User logged out", ipAddress, userAgent)
}

// LogUserRegistered logs a user registration event
func (s *Service) LogUserRegistered(ctx context.Context, userID uuid.UUID, email, ipAddress, userAgent string) error {
	return s.Log(ctx, &CreateAuditLogRequest{
		UserID:      &userID,
		Action:      ActionUserRegistered,
		EntityType:  "user",
		EntityID:    &userID,
		Description: "User registered",
		NewValue:    map[string]interface{}{"email": email},
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})
}

// LogPasswordReset logs a password reset request
func (s *Service) LogPasswordReset(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string) error {
	return s.LogAction(ctx, &userID, ActionPasswordReset, "user", &userID, "Password reset requested", ipAddress, userAgent)
}

// LogPasswordChanged logs a password change
func (s *Service) LogPasswordChanged(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string) error {
	return s.LogAction(ctx, &userID, ActionPasswordChanged, "user", &userID, "Password changed", ipAddress, userAgent)
}

// LogTournamentCreated logs tournament creation
func (s *Service) LogTournamentCreated(ctx context.Context, userID, tournamentID uuid.UUID, tournamentName, ipAddress, userAgent string) error {
	return s.Log(ctx, &CreateAuditLogRequest{
		UserID:      &userID,
		Action:      ActionTournamentCreated,
		EntityType:  "tournament",
		EntityID:    &tournamentID,
		Description: "Tournament created: " + tournamentName,
		NewValue:    map[string]interface{}{"name": tournamentName},
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})
}

// LogTournamentUpdated logs tournament update
func (s *Service) LogTournamentUpdated(ctx context.Context, userID, tournamentID uuid.UUID, changes map[string]interface{}, ipAddress, userAgent string) error {
	return s.Log(ctx, &CreateAuditLogRequest{
		UserID:      &userID,
		Action:      ActionTournamentUpdated,
		EntityType:  "tournament",
		EntityID:    &tournamentID,
		Description: "Tournament updated",
		NewValue:    changes,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})
}

// LogGameResultEntered logs game result entry
func (s *Service) LogGameResultEntered(ctx context.Context, userID, gameID uuid.UUID, result, ipAddress, userAgent string) error {
	return s.Log(ctx, &CreateAuditLogRequest{
		UserID:      &userID,
		Action:      ActionGameResultEntered,
		EntityType:  "game",
		EntityID:    &gameID,
		Description: "Game result entered: " + result,
		NewValue:    map[string]interface{}{"result": result},
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})
}

// LogGameResultModified logs game result modification
func (s *Service) LogGameResultModified(ctx context.Context, userID, gameID uuid.UUID, oldResult, newResult, ipAddress, userAgent string) error {
	return s.Log(ctx, &CreateAuditLogRequest{
		UserID:      &userID,
		Action:      ActionGameResultModified,
		EntityType:  "game",
		EntityID:    &gameID,
		Description: "Game result modified",
		OldValue:    map[string]interface{}{"result": oldResult},
		NewValue:    map[string]interface{}{"result": newResult},
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})
}

// LogAdminExport logs admin export action
func (s *Service) LogAdminExport(ctx context.Context, userID uuid.UUID, exportType, format, ipAddress, userAgent string) error {
	return s.Log(ctx, &CreateAuditLogRequest{
		UserID:      &userID,
		Action:      ActionAdminExport,
		EntityType:  "export",
		Description: "Exported " + exportType + " in " + format + " format",
		NewValue:    map[string]interface{}{"type": exportType, "format": format},
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})
}
