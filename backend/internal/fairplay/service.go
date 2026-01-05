package fairplay

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles fairplay business logic
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new fairplay service
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("service", "fairplay").Logger(),
	}
}

// Initialize ensures database tables exist
func (s *Service) Initialize(ctx context.Context) error {
	return s.repo.EnsureTables(ctx)
}

// ReportViolation creates a new violation report
func (s *Service) ReportViolation(ctx context.Context, req *CreateViolationRequest, reportedBy uuid.UUID) (*FairplayViolation, error) {
	// Determine severity based on type
	severity := determineSeverity(req.Type)

	violation := &FairplayViolation{
		TournamentID: req.TournamentID,
		GameID:       req.GameID,
		PlayerID:     req.PlayerID,
		ReportedBy:   &reportedBy,
		Type:         req.Type,
		Severity:     severity,
		Status:       StatusPending,
		Description:  req.Description,
		Evidence:     req.Evidence,
	}

	if err := s.repo.CreateViolation(ctx, violation); err != nil {
		s.logger.Error().Err(err).
			Str("player_id", req.PlayerID.String()).
			Str("type", string(req.Type)).
			Msg("Failed to create violation report")
		return nil, err
	}

	s.logger.Info().
		Str("violation_id", violation.ID.String()).
		Str("player_id", req.PlayerID.String()).
		Str("type", string(req.Type)).
		Str("reported_by", reportedBy.String()).
		Msg("Violation reported")

	return violation, nil
}

// GetViolation retrieves a violation by ID
func (s *Service) GetViolation(ctx context.Context, id uuid.UUID) (*FairplayViolation, error) {
	return s.repo.GetViolation(ctx, id)
}

// ReviewViolation reviews a violation report
func (s *Service) ReviewViolation(ctx context.Context, id uuid.UUID, req *ReviewViolationRequest, reviewedBy uuid.UUID) (*FairplayViolation, error) {
	// Get existing violation
	violation, err := s.repo.GetViolation(ctx, id)
	if err != nil {
		return nil, err
	}

	if violation.Status != StatusPending && violation.Status != StatusReviewing {
		return nil, fmt.Errorf("violation has already been reviewed")
	}

	if err := s.repo.UpdateViolation(ctx, id, req.Status, reviewedBy, req.ReviewNotes, req.ActionTaken); err != nil {
		s.logger.Error().Err(err).
			Str("violation_id", id.String()).
			Msg("Failed to review violation")
		return nil, err
	}

	s.logger.Info().
		Str("violation_id", id.String()).
		Str("status", string(req.Status)).
		Str("reviewed_by", reviewedBy.String()).
		Msg("Violation reviewed")

	// Auto-flag player if confirmed high severity
	if req.Status == StatusConfirmed && violation.Severity == SeverityHigh || violation.Severity == SeverityCritical {
		flag := &PlayerFlag{
			PlayerID:  violation.PlayerID,
			FlaggedBy: reviewedBy,
			FlagType:  "watch",
			Reason:    fmt.Sprintf("Confirmed fairplay violation: %s", violation.Type),
		}
		if err := s.repo.CreateFlag(ctx, flag); err != nil {
			s.logger.Warn().Err(err).
				Str("player_id", violation.PlayerID.String()).
				Msg("Failed to auto-flag player after violation")
		}
	}

	return s.repo.GetViolation(ctx, id)
}

// ListViolations lists violations with optional filtering
func (s *Service) ListViolations(ctx context.Context, req *ViolationListRequest) (*ViolationListResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	violations, total, err := s.repo.ListViolations(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ViolationListResponse{
		Violations: violations,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
	}, nil
}

// FlagPlayer creates a flag on a player
func (s *Service) FlagPlayer(ctx context.Context, req *CreateFlagRequest, flaggedBy uuid.UUID) (*PlayerFlag, error) {
	flag := &PlayerFlag{
		PlayerID:  req.PlayerID,
		FlaggedBy: flaggedBy,
		FlagType:  req.FlagType,
		Reason:    req.Reason,
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.repo.CreateFlag(ctx, flag); err != nil {
		s.logger.Error().Err(err).
			Str("player_id", req.PlayerID.String()).
			Str("flag_type", req.FlagType).
			Msg("Failed to create player flag")
		return nil, err
	}

	s.logger.Info().
		Str("flag_id", flag.ID.String()).
		Str("player_id", req.PlayerID.String()).
		Str("flag_type", req.FlagType).
		Str("flagged_by", flaggedBy.String()).
		Msg("Player flagged")

	return flag, nil
}

// GetPlayerFlags retrieves active flags for a player
func (s *Service) GetPlayerFlags(ctx context.Context, playerID uuid.UUID) (*FlagListResponse, error) {
	flags, err := s.repo.GetPlayerFlags(ctx, playerID)
	if err != nil {
		return nil, err
	}

	return &FlagListResponse{
		Flags: flags,
		Total: len(flags),
	}, nil
}

// RemoveFlag removes a player flag
func (s *Service) RemoveFlag(ctx context.Context, flagID, removedBy uuid.UUID) error {
	if err := s.repo.RemoveFlag(ctx, flagID, removedBy); err != nil {
		s.logger.Error().Err(err).
			Str("flag_id", flagID.String()).
			Msg("Failed to remove flag")
		return err
	}

	s.logger.Info().
		Str("flag_id", flagID.String()).
		Str("removed_by", removedBy.String()).
		Msg("Flag removed")

	return nil
}

// GetStats retrieves fair play statistics
func (s *Service) GetStats(ctx context.Context, tournamentID *uuid.UUID) (*FairplayStats, error) {
	return s.repo.GetStats(ctx, tournamentID)
}

// GetPlayerViolationHistory retrieves a player's violation history
func (s *Service) GetPlayerViolationHistory(ctx context.Context, playerID uuid.UUID) ([]FairplayViolation, error) {
	return s.repo.GetPlayerViolationHistory(ctx, playerID)
}

// CheckPlayerEligibility checks if a player is eligible to participate
func (s *Service) CheckPlayerEligibility(ctx context.Context, playerID uuid.UUID) (bool, string, error) {
	// Check for banned flag
	isBanned, err := s.repo.HasActiveFlag(ctx, playerID, "banned")
	if err != nil {
		return false, "", err
	}
	if isBanned {
		return false, "Player is currently banned from participation", nil
	}

	return true, "", nil
}

// IsPlayerWatched checks if a player is on watch list
func (s *Service) IsPlayerWatched(ctx context.Context, playerID uuid.UUID) (bool, error) {
	return s.repo.HasActiveFlag(ctx, playerID, "watch")
}

// determineSeverity determines the severity based on violation type
func determineSeverity(vType ViolationType) SeverityLevel {
	switch vType {
	case ViolationSuspiciousPlay:
		return SeverityHigh
	case ViolationRatingManipulation:
		return SeverityCritical
	case ViolationCollusion:
		return SeverityCritical
	case ViolationTimestalling:
		return SeverityLow
	case ViolationUnsportsmanlike:
		return SeverityMedium
	default:
		return SeverityMedium
	}
}
