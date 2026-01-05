package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// Service handles user business logic
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new user service
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("service", "user").Logger(),
	}
}

// GetProfile retrieves the current user's profile
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	profile, err := s.repo.GetUserProfile(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get user profile")
		return nil, err
	}
	return profile, nil
}

// UpdateProfile updates the current user's profile
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) (*UserProfile, error) {
	// Update user fields
	if err := s.repo.UpdateUser(ctx, userID, req); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to update user")
		return nil, err
	}

	// Update player fields if any
	if req.FirstName != nil || req.LastName != nil {
		if err := s.repo.UpdatePlayerProfile(ctx, userID, req); err != nil {
			s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to update player profile")
			return nil, err
		}
	}

	s.logger.Info().Str("user_id", userID.String()).Msg("Profile updated")

	return s.repo.GetUserProfile(ctx, userID)
}

// ChangePassword changes the user's password
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error {
	// Verify current password
	currentHash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Validate new password is different
	if req.CurrentPassword == req.NewPassword {
		return fmt.Errorf("new password must be different from current password")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to hash password")
		return fmt.Errorf("failed to process password")
	}

	// Update password
	if err := s.repo.UpdatePassword(ctx, userID, string(newHash)); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to update password")
		return err
	}

	// Revoke all sessions except current (for security)
	if err := s.repo.RevokeAllSessions(ctx, userID); err != nil {
		s.logger.Warn().Err(err).Str("user_id", userID.String()).Msg("Failed to revoke sessions after password change")
	}

	s.logger.Info().Str("user_id", userID.String()).Msg("Password changed")
	return nil
}

// LinkPlayer links a player profile to the user account
func (s *Service) LinkPlayer(ctx context.Context, userID uuid.UUID, req *LinkPlayerRequest) error {
	// Check if user already has a linked player
	profile, err := s.repo.GetUserProfile(ctx, userID)
	if err != nil {
		return err
	}

	if profile.PlayerID != nil {
		return fmt.Errorf("user already has a linked player profile")
	}

	if err := s.repo.LinkPlayer(ctx, userID, req.PlayerID); err != nil {
		s.logger.Error().Err(err).
			Str("user_id", userID.String()).
			Str("player_id", req.PlayerID.String()).
			Msg("Failed to link player")
		return err
	}

	s.logger.Info().
		Str("user_id", userID.String()).
		Str("player_id", req.PlayerID.String()).
		Msg("Player linked to user")
	return nil
}

// UnlinkPlayer removes the link between user and player profile
func (s *Service) UnlinkPlayer(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.UnlinkPlayer(ctx, userID); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to unlink player")
		return err
	}

	s.logger.Info().Str("user_id", userID.String()).Msg("Player unlinked from user")
	return nil
}

// GetSessions retrieves active sessions for the user
func (s *Service) GetSessions(ctx context.Context, userID uuid.UUID) ([]UserSession, error) {
	return s.repo.GetUserSessions(ctx, userID)
}

// RevokeSession revokes a specific session
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	if err := s.repo.RevokeSession(ctx, userID, sessionID); err != nil {
		s.logger.Error().Err(err).
			Str("user_id", userID.String()).
			Str("session_id", sessionID.String()).
			Msg("Failed to revoke session")
		return err
	}

	s.logger.Info().
		Str("user_id", userID.String()).
		Str("session_id", sessionID.String()).
		Msg("Session revoked")
	return nil
}

// RevokeAllSessions revokes all sessions for the user
func (s *Service) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.RevokeAllSessions(ctx, userID); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to revoke all sessions")
		return err
	}

	s.logger.Info().Str("user_id", userID.String()).Msg("All sessions revoked")
	return nil
}

// GetStats retrieves user statistics
func (s *Service) GetStats(ctx context.Context, userID uuid.UUID) (*UserStats, error) {
	return s.repo.GetUserStats(ctx, userID)
}
