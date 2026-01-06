package registration

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tranmh/openpairing/internal/tournament"
)

// Service provides registration business logic
type Service struct {
	repo       Repository
	appBaseURL string
	logger     zerolog.Logger
}

// NewService creates a new registration service
func NewService(repo Repository, appBaseURL string, logger zerolog.Logger) *Service {
	return &Service{
		repo:       repo,
		appBaseURL: appBaseURL,
		logger:     logger.With().Str("service", "registration").Logger(),
	}
}

// GetPublicTournament retrieves public tournament info by slug
func (s *Service) GetPublicTournament(ctx context.Context, slug string) (*TournamentPublicInfo, *tournament.RegistrationSettings, error) {
	info, settings, err := s.repo.GetTournamentBySlug(ctx, slug)
	if err != nil {
		return nil, nil, err
	}

	// Add settings to info for API response
	info.RegistrationSettings = settings

	return info, settings, nil
}

// PreRegister handles player pre-registration
func (s *Service) PreRegister(ctx context.Context, tournamentID uuid.UUID, req *PreRegisterRequest, userID *uuid.UUID) (*PreRegisterResponse, error) {
	// Get tournament and settings
	info, settings, err := s.repo.GetTournamentByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Validate registration is open
	if err := s.validateRegistrationOpen(info, settings); err != nil {
		return nil, err
	}

	// Validate invite token for private tournaments
	if settings.Access.Mode == "invite_only" {
		if req.InviteToken == "" {
			return nil, ErrInvalidInviteToken
		}
		// Use atomic validate-and-use to prevent TOCTOU race conditions
		// This validates the token AND increments uses in a single transaction
		_, err := s.repo.ValidateAndUseInviteToken(ctx, tournamentID, req.InviteToken)
		if err != nil {
			return nil, err
		}
	}

	// Note: Capacity check is done atomically in CreateGuestRegistrationAtomic
	// to prevent race conditions with concurrent registrations

	// Determine if this is a guest or authenticated registration
	isGuest := req.PlayerID == nil

	if isGuest {
		// Validate guest registration is allowed
		if !settings.PreRegistration.AllowGuests {
			return nil, ErrGuestsNotAllowed
		}

		// Create registration atomically (handles duplicates and capacity via DB constraints)
		return s.createGuestRegistration(ctx, tournamentID, req, settings, info.MaxPlayers)
	}

	// Authenticated user registration
	// Check if already registered as participant
	registered, err := s.repo.IsParticipantRegistered(ctx, tournamentID, *req.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check registration: %w", err)
	}
	if registered {
		return nil, ErrAlreadyRegistered
	}

	// For authenticated users, we would create a participant record
	// This requires integration with the tournament service
	// For now, we'll create a "pre-registered" participant entry
	return s.createAuthenticatedPreRegistration(ctx, tournamentID, req, userID, settings)
}

func (s *Service) createGuestRegistration(ctx context.Context, tournamentID uuid.UUID, req *PreRegisterRequest, settings *tournament.RegistrationSettings, maxPlayers *int) (*PreRegisterResponse, error) {
	status := StatusPreRegistered
	if settings.PreRegistration.RequiresApproval {
		status = StatusPendingApproval
	}

	// Generate check-in code BEFORE creating registration to avoid partial state
	code := generateCheckInCode()

	reg := &GuestRegistration{
		TournamentID:       tournamentID,
		Email:              req.Email,
		FirstName:          req.FirstName,
		LastName:           req.LastName,
		RegistrationStatus: status,
		CheckInCode:        &code,
	}

	// Set federation IDs if provided
	if req.FederationID != "" {
		switch req.FederationType {
		case "fide":
			reg.FideID = &req.FederationID
		case "dsb":
			reg.DsbID = &req.FederationID
		case "oesb":
			reg.OesbID = &req.FederationID
		case "ssb":
			reg.SsbID = &req.FederationID
		}
	}

	// Create registration atomically with capacity check
	// This handles race conditions for both duplicates and capacity
	if err := s.repo.CreateGuestRegistrationAtomic(ctx, reg, maxPlayers); err != nil {
		if err == ErrAlreadyRegistered {
			return nil, ErrAlreadyRegistered
		}
		if err == ErrTournamentFull {
			return nil, ErrTournamentFull
		}
		return nil, fmt.Errorf("failed to create registration: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("email", req.Email).
		Str("status", status).
		Msg("Guest pre-registered")

	message := "Voranmeldung erfolgreich. "
	if settings.PreRegistration.RequiresApproval {
		message += "Deine Anmeldung muss noch vom Veranstalter bestätigt werden."
	} else {
		message += "Du kannst dich am Turniertag einchecken."
	}

	return &PreRegisterResponse{
		ID:               reg.ID,
		Status:           status,
		CheckInCode:      code,
		RequiresApproval: settings.PreRegistration.RequiresApproval,
		Message:          message,
	}, nil
}

func (s *Service) createAuthenticatedPreRegistration(ctx context.Context, tournamentID uuid.UUID, req *PreRegisterRequest, userID *uuid.UUID, settings *tournament.RegistrationSettings) (*PreRegisterResponse, error) {
	// TODO: Implement authenticated user pre-registration
	// This requires integration with the tournament service to create a participant entry
	// For now, return an error to prevent users thinking they're registered when they're not
	return nil, fmt.Errorf("authenticated pre-registration not yet implemented - please use guest registration or contact the organizer")
}

// generateCheckInCode generates a unique check-in code using crypto/rand
func generateCheckInCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Exclude confusing characters (0, O, 1, I, L)
	code := make([]byte, 8)
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	for i := range code {
		code[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(code)
}

// CheckIn handles player check-in
func (s *Service) CheckIn(ctx context.Context, tournamentID uuid.UUID, req *CheckInRequest, userID *uuid.UUID) (*CheckInResponse, error) {
	// Get tournament and settings
	info, settings, err := s.repo.GetTournamentByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Validate check-in is enabled and window is open
	if !settings.CheckIn.Enabled {
		return nil, ErrCheckInWindowClosed
	}

	if !s.IsCheckInWindowOpen(info, settings) {
		return nil, ErrCheckInWindowClosed
	}

	// Determine method
	method := req.Method
	if method == "" {
		method = "self_service"
	}

	// Validate method is allowed
	if !settings.HasCheckInMethod(method) {
		return nil, fmt.Errorf("check-in method '%s' is not allowed", method)
	}

	// Find registration by code
	if req.Code != "" {
		return s.checkInByCode(ctx, req.Code, method, userID)
	}

	// If authenticated user, check in their registration
	if userID != nil {
		// TODO: Find their registration and check in
		return nil, fmt.Errorf("check-in by user ID not yet implemented")
	}

	return nil, fmt.Errorf("check-in code is required")
}

func (s *Service) checkInByCode(ctx context.Context, code string, method string, checkedInBy *uuid.UUID) (*CheckInResponse, error) {
	// Try to find registration by code
	regID, isGuest, err := s.repo.GetRegistrationByCheckInCode(ctx, code)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if isGuest {
		// Atomically check in the guest - the repository handles status validation
		if err := s.repo.SetGuestCheckIn(ctx, *regID, method, checkedInBy); err != nil {
			if err == ErrAlreadyCheckedIn {
				// Return success for idempotency - they're already checked in
				reg, getErr := s.repo.GetGuestRegistration(ctx, *regID)
				if getErr == nil && reg.CheckedInAt != nil {
					return &CheckInResponse{
						Success:   true,
						ID:        reg.ID,
						Status:    StatusCheckedIn,
						CheckedIn: *reg.CheckedInAt,
						Message:   "Du bist bereits eingecheckt.",
					}, nil
				}
				return nil, err
			}
			if err == ErrInvalidCheckInStatus {
				// Get status to provide specific error message
				reg, getErr := s.repo.GetGuestRegistration(ctx, *regID)
				if getErr == nil {
					switch reg.RegistrationStatus {
					case StatusRejected:
						return nil, fmt.Errorf("registration was rejected")
					case StatusPendingApproval:
						return nil, fmt.Errorf("registration is pending organizer approval")
					}
				}
				return nil, fmt.Errorf("registration status does not allow check-in")
			}
			return nil, fmt.Errorf("failed to check in: %w", err)
		}

		s.logger.Info().
			Str("guest_id", regID.String()).
			Str("method", method).
			Msg("Guest checked in")

		return &CheckInResponse{
			Success:   true,
			ID:        *regID,
			Status:    StatusCheckedIn,
			CheckedIn: now,
			Message:   "Erfolgreich eingecheckt!",
		}, nil
	}

	// Participant check-in - repository handles idempotency check
	if err := s.repo.SetParticipantCheckIn(ctx, *regID, method, checkedInBy); err != nil {
		if err == ErrAlreadyCheckedIn {
			// Return success for idempotency
			return &CheckInResponse{
				Success:   true,
				ID:        *regID,
				Status:    StatusCheckedIn,
				CheckedIn: now,
				Message:   "Du bist bereits eingecheckt.",
			}, nil
		}
		return nil, fmt.Errorf("failed to check in: %w", err)
	}

	s.logger.Info().
		Str("participant_id", regID.String()).
		Str("method", method).
		Msg("Participant checked in")

	return &CheckInResponse{
		Success:   true,
		ID:        *regID,
		Status:    StatusCheckedIn,
		CheckedIn: now,
		Message:   "Erfolgreich eingecheckt!",
	}, nil
}

// ApproveRegistration approves or rejects a guest registration
func (s *Service) ApproveRegistration(ctx context.Context, registrationID uuid.UUID, req *ApproveRegistrationRequest, approverID uuid.UUID) error {
	if !req.Approved && req.Reason == "" {
		return fmt.Errorf("reason is required when rejecting")
	}

	if err := s.repo.SetGuestApproval(ctx, registrationID, req.Approved, approverID, req.Reason); err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	action := "approved"
	if !req.Approved {
		action = "rejected"
	}

	s.logger.Info().
		Str("registration_id", registrationID.String()).
		Str("action", action).
		Str("approver_id", approverID.String()).
		Msg("Registration approval updated")

	return nil
}

// GetRegistrations returns all registrations for a tournament
func (s *Service) GetRegistrations(ctx context.Context, tournamentID uuid.UUID) (*RegistrationListResponse, error) {
	items, err := s.repo.GetAllRegistrations(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registrations: %w", err)
	}

	info, _, err := s.repo.GetTournamentByID(ctx, tournamentID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get tournament info")
	}

	return &RegistrationListResponse{
		Registrations: items,
		Total:         len(items),
		Tournament:    info,
	}, nil
}

// GetCheckInStatus returns check-in statistics
func (s *Service) GetCheckInStatus(ctx context.Context, tournamentID uuid.UUID) (*CheckInStatus, error) {
	status, err := s.repo.GetCheckInStatus(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get check-in status: %w", err)
	}

	info, settings, err := s.repo.GetTournamentByID(ctx, tournamentID)
	if err == nil {
		status.IsWindowOpen = s.IsCheckInWindowOpen(info, settings)
	}

	return status, nil
}

// GetRegistrationSettings returns registration settings for a tournament
func (s *Service) GetRegistrationSettings(ctx context.Context, tournamentID uuid.UUID) (*tournament.RegistrationSettings, error) {
	return s.repo.GetRegistrationSettings(ctx, tournamentID)
}

// UpdateRegistrationSettings updates registration settings
func (s *Service) UpdateRegistrationSettings(ctx context.Context, tournamentID uuid.UUID, req *UpdateRegistrationSettingsRequest) error {
	// Get current settings
	settings, err := s.repo.GetRegistrationSettings(ctx, tournamentID)
	if err != nil {
		return err
	}

	// Apply updates
	if req.PreRegistration != nil {
		if req.PreRegistration.Enabled != nil {
			settings.PreRegistration.Enabled = *req.PreRegistration.Enabled
		}
		if req.PreRegistration.RequiresApproval != nil {
			settings.PreRegistration.RequiresApproval = *req.PreRegistration.RequiresApproval
		}
		if req.PreRegistration.AllowGuests != nil {
			settings.PreRegistration.AllowGuests = *req.PreRegistration.AllowGuests
		}
		if req.PreRegistration.NotifyOrganizer != nil {
			settings.PreRegistration.NotifyOrganizer = *req.PreRegistration.NotifyOrganizer
		}
	}

	if req.CheckIn != nil {
		if req.CheckIn.Enabled != nil {
			settings.CheckIn.Enabled = *req.CheckIn.Enabled
		}
		if req.CheckIn.Methods != nil {
			settings.CheckIn.Methods = req.CheckIn.Methods
		}
		if req.CheckIn.WindowOpenMinutes != nil {
			settings.CheckIn.WindowOpenMinutes = *req.CheckIn.WindowOpenMinutes
		}
		if req.CheckIn.WindowCloseMinutes != nil {
			settings.CheckIn.WindowCloseMinutes = *req.CheckIn.WindowCloseMinutes
		}
		if req.CheckIn.AutoNoShow != nil {
			settings.CheckIn.AutoNoShow = *req.CheckIn.AutoNoShow
		}
	}

	if req.Access != nil && req.Access.Mode != nil {
		settings.Access.Mode = *req.Access.Mode
	}

	// Validate
	if errs := settings.Validate(); len(errs) > 0 {
		return fmt.Errorf("invalid settings: %s", errs[0].Message)
	}

	return s.repo.UpdateRegistrationSettings(ctx, tournamentID, settings)
}

// CreateInviteToken creates a new invite token
func (s *Service) CreateInviteToken(ctx context.Context, tournamentID uuid.UUID, req *CreateInviteTokenRequest, creatorID uuid.UUID) (*InviteToken, string, error) {
	rawToken := GenerateInviteToken()

	token := &InviteToken{
		TournamentID: tournamentID,
		Name:         &req.Name,
		MaxUses:      req.MaxUses,
		ValidUntil:   req.ValidUntil,
		CreatedBy:    creatorID,
	}

	if err := s.repo.CreateInviteToken(ctx, token, rawToken); err != nil {
		return nil, "", fmt.Errorf("failed to create invite token: %w", err)
	}

	token.Token = rawToken

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("token_id", token.ID.String()).
		Msg("Invite token created")

	return token, rawToken, nil
}

// ListInviteTokens lists all invite tokens for a tournament
func (s *Service) ListInviteTokens(ctx context.Context, tournamentID uuid.UUID) ([]InviteToken, error) {
	return s.repo.ListInviteTokens(ctx, tournamentID)
}

// DeleteInviteToken deletes an invite token
func (s *Service) DeleteInviteToken(ctx context.Context, tokenID uuid.UUID) error {
	return s.repo.DeleteInviteToken(ctx, tokenID)
}

// GetMyRegistration returns the current user's registration status
func (s *Service) GetMyRegistration(ctx context.Context, tournamentID uuid.UUID, userID *uuid.UUID, email *string) (*MyRegistrationResponse, error) {
	info, settings, err := s.repo.GetTournamentByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	resp := &MyRegistrationResponse{
		Found:         false,
		IsCheckInOpen: s.IsCheckInWindowOpen(info, settings),
	}

	// Calculate check-in window times
	if settings.CheckIn.Enabled {
		windowStart := info.StartDate.Add(-time.Duration(settings.CheckIn.WindowOpenMinutes) * time.Minute)
		windowEnd := info.StartDate.Add(time.Duration(settings.CheckIn.WindowCloseMinutes) * time.Minute)
		resp.CheckInWindowStart = &windowStart
		resp.CheckInWindowEnd = &windowEnd
	}

	// Try to find by email (guest registration)
	if email != nil {
		guest, err := s.repo.GetGuestRegistrationByEmail(ctx, tournamentID, *email)
		if err == nil {
			resp.Found = true
			resp.Type = "guest"
			resp.ID = &guest.ID
			resp.Status = guest.RegistrationStatus
			resp.PreRegisteredAt = &guest.PreRegisteredAt
			resp.ApprovedAt = guest.ApprovedAt
			resp.CheckedInAt = guest.CheckedInAt
			resp.CheckInCode = guest.CheckInCode
			return resp, nil
		}
	}

	// Try to find by player ID (participant)
	// This would require looking up the player associated with the user
	// For now, we return not found for authenticated users without guest registration

	return resp, nil
}

// IsCheckInWindowOpen checks if the check-in window is currently open
func (s *Service) IsCheckInWindowOpen(info *TournamentPublicInfo, settings *tournament.RegistrationSettings) bool {
	if settings == nil || !settings.CheckIn.Enabled {
		return false
	}

	now := time.Now()
	windowOpen := info.StartDate.Add(-time.Duration(settings.CheckIn.WindowOpenMinutes) * time.Minute)
	windowClose := info.StartDate.Add(time.Duration(settings.CheckIn.WindowCloseMinutes) * time.Minute)

	return now.After(windowOpen) && now.Before(windowClose)
}

// validateRegistrationOpen checks if registration is currently open
func (s *Service) validateRegistrationOpen(info *TournamentPublicInfo, settings *tournament.RegistrationSettings) error {
	// Check if pre-registration is enabled
	if !settings.PreRegistration.Enabled {
		return ErrPreRegistrationDisabled
	}

	// Check tournament status
	if info.Status != "published" && info.Status != "registration" {
		return ErrRegistrationClosed
	}

	now := time.Now()

	// Check registration window
	if info.RegistrationOpens != nil && now.Before(*info.RegistrationOpens) {
		return ErrRegistrationClosed
	}
	if info.RegistrationCloses != nil && now.After(*info.RegistrationCloses) {
		return ErrRegistrationClosed
	}

	return nil
}
