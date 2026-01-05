package waitlist

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// EmailSender interface for sending emails
type EmailSender interface {
	SendWaitlistNotification(ctx context.Context, to, recipientName string, data map[string]interface{}) error
}

// Service provides waitlist business logic
type Service struct {
	repo         Repository
	emailService EmailSender
	appBaseURL   string
	logger       zerolog.Logger
}

// NewService creates a new waitlist service
func NewService(repo Repository, emailService EmailSender, appBaseURL string, logger zerolog.Logger) *Service {
	return &Service{
		repo:         repo,
		emailService: emailService,
		appBaseURL:   appBaseURL,
		logger:       logger.With().Str("service", "waitlist").Logger(),
	}
}

// AddToWaitlist adds a player to the tournament waitlist
func (s *Service) AddToWaitlist(ctx context.Context, tournamentID uuid.UUID, req *AddToWaitlistRequest) (*WaitlistEntry, error) {
	// Check if player is already registered
	registered, err := s.repo.IsPlayerRegistered(ctx, tournamentID, req.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check registration: %w", err)
	}
	if registered {
		return nil, ErrAlreadyRegistered
	}

	// Check if already on waitlist
	existing, err := s.repo.GetByTournamentAndPlayer(ctx, tournamentID, req.PlayerID)
	if err == nil && existing != nil {
		return nil, ErrAlreadyOnWaitlist
	}

	entry := &WaitlistEntry{
		TournamentID:   tournamentID,
		PlayerID:       req.PlayerID,
		StartingRating: req.StartingRating,
		Notes:          req.Notes,
	}

	if err := s.repo.Add(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to add to waitlist: %w", err)
	}

	// Fetch the complete entry with player details
	entry, err = s.repo.GetByID(ctx, entry.ID)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("player_id", req.PlayerID.String()).
		Int("position", entry.Position).
		Msg("Player added to waitlist")

	return entry, nil
}

// GetWaitlist retrieves the waitlist for a tournament
func (s *Service) GetWaitlist(ctx context.Context, tournamentID uuid.UUID) (*WaitlistListResponse, error) {
	entries, err := s.repo.List(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get waitlist: %w", err)
	}

	info, err := s.repo.GetTournamentInfo(ctx, tournamentID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get tournament info")
	}

	return &WaitlistListResponse{
		Entries:    entries,
		Total:      len(entries),
		Tournament: info,
	}, nil
}

// GetEntry retrieves a waitlist entry by ID
func (s *Service) GetEntry(ctx context.Context, id uuid.UUID) (*WaitlistEntry, error) {
	return s.repo.GetByID(ctx, id)
}

// RemoveFromWaitlist removes a player from the waitlist
func (s *Service) RemoveFromWaitlist(ctx context.Context, id uuid.UUID) error {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Remove(ctx, id); err != nil {
		return fmt.Errorf("failed to remove from waitlist: %w", err)
	}

	s.logger.Info().
		Str("entry_id", id.String()).
		Str("tournament_id", entry.TournamentID.String()).
		Str("player_id", entry.PlayerID.String()).
		Msg("Player removed from waitlist")

	return nil
}

// PromoteNextPlayer promotes the next player in the queue
func (s *Service) PromoteNextPlayer(ctx context.Context, tournamentID uuid.UUID, req *PromoteFromWaitlistRequest) (*WaitlistEntry, error) {
	// Get next in queue
	entry, err := s.repo.GetNextInQueue(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next in queue: %w", err)
	}
	if entry == nil {
		return nil, nil // No one in queue
	}

	if req.NotifyPlayer && s.emailService != nil {
		// Set expiry and notify player
		expiryHours := req.ExpiryHours
		if expiryHours == 0 {
			expiryHours = 48 // Default 48 hours
		}
		expiresAt := time.Now().Add(time.Duration(expiryHours) * time.Hour)

		if err := s.repo.SetNotified(ctx, entry.ID, expiresAt); err != nil {
			return nil, fmt.Errorf("failed to set notification status: %w", err)
		}

		// Get tournament info for email
		info, _ := s.repo.GetTournamentInfo(ctx, tournamentID)
		tournamentName := "Tournament"
		if info != nil {
			tournamentName = info.Name
		}

		// Get player email (would need to join with users table)
		// For now, we'll log and continue
		s.logger.Info().
			Str("entry_id", entry.ID.String()).
			Str("player_name", entry.PlayerFirstName+" "+entry.PlayerLastName).
			Int("expiry_hours", expiryHours).
			Msg("Player notified about available spot")

		// TODO: Send email notification
		// This would require getting the user's email from the players table
		_ = tournamentName
	}

	if req.AutoPromote {
		// Automatically register the player (would need to call tournament service)
		s.logger.Info().
			Str("entry_id", entry.ID.String()).
			Msg("Auto-promoting player from waitlist")
		// This would require dependency on tournament service
	}

	return entry, nil
}

// GetPlayerPosition gets a player's position on the waitlist
func (s *Service) GetPlayerPosition(ctx context.Context, tournamentID, playerID uuid.UUID) (*WaitlistEntry, error) {
	return s.repo.GetByTournamentAndPlayer(ctx, tournamentID, playerID)
}

// ProcessExpiredEntries removes expired entries and notifies next in queue
func (s *Service) ProcessExpiredEntries(ctx context.Context) (int, error) {
	entries, err := s.repo.GetExpiredEntries(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get expired entries: %w", err)
	}

	processed := 0
	for _, entry := range entries {
		if err := s.repo.Remove(ctx, entry.ID); err != nil {
			s.logger.Error().
				Err(err).
				Str("entry_id", entry.ID.String()).
				Msg("Failed to remove expired entry")
			continue
		}

		s.logger.Info().
			Str("entry_id", entry.ID.String()).
			Str("tournament_id", entry.TournamentID.String()).
			Msg("Removed expired waitlist entry")

		// Notify next person in queue
		_, _ = s.PromoteNextPlayer(ctx, entry.TournamentID, &PromoteFromWaitlistRequest{
			NotifyPlayer: true,
			ExpiryHours:  48,
		})

		processed++
	}

	return processed, nil
}
