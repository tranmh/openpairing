package tournament

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/pairing/engine"
	"github.com/openpairing/openpairing/internal/pairing/trf"
	"github.com/openpairing/openpairing/internal/tournament/tiebreak"
	"github.com/rs/zerolog"
)

// Service provides tournament business logic
type Service struct {
	repo           Repository
	pairingEngine  engine.PairingEngine // Legacy: single engine for backwards compatibility
	engineRegistry *engine.Registry     // New: registry of all engines
	logger         zerolog.Logger
}

// NewService creates a new tournament service
func NewService(repo Repository, pairingEngine engine.PairingEngine, logger zerolog.Logger) *Service {
	return &Service{
		repo:          repo,
		pairingEngine: pairingEngine,
		logger:        logger.With().Str("service", "tournament").Logger(),
	}
}

// NewServiceWithRegistry creates a new tournament service with engine registry
func NewServiceWithRegistry(repo Repository, registry *engine.Registry, logger zerolog.Logger) *Service {
	// Get default Swiss engine for backwards compatibility
	var defaultEngine engine.PairingEngine
	if registry != nil {
		defaultEngine, _ = registry.GetSwissEngine()
	}

	return &Service{
		repo:           repo,
		pairingEngine:  defaultEngine,
		engineRegistry: registry,
		logger:         logger.With().Str("service", "tournament").Logger(),
	}
}

// CreateTournament creates a new tournament
func (s *Service) CreateTournament(ctx context.Context, req *CreateTournamentRequest, organizerID uuid.UUID) (*Tournament, error) {
	// Parse start date from string
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		// Try date-only format
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start date format: %w", err)
		}
	}

	// Parse end date if provided
	var endDate *time.Time
	if req.EndDate != nil {
		ed, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			ed, err = time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				return nil, fmt.Errorf("invalid end date format: %w", err)
			}
		}
		endDate = &ed
	}

	tournament := &Tournament{
		ID:                uuid.New(),
		OrganizationID:   req.OrganizationID,
		Name:             req.Name,
		Description:      req.Description,
		Slug:             req.Slug,
		Type:             req.Type,
		Mode:             req.Mode,
		TimeControl:      req.TimeControl,
		TimeControlType:  req.TimeControlType,
		PairingEngine:    req.PairingEngine,
		PairingSystem:    req.PairingSystem,
		AcceleratedPairing: req.AcceleratedPairing,
		StartDate:        startDate,
		EndDate:          endDate,
		RegistrationOpens:  req.RegistrationOpens,
		RegistrationCloses: req.RegistrationCloses,
		MaxPlayers:       req.MaxPlayers,
		MinPlayers:       req.MinPlayers,
		Rounds:           req.Rounds,
		VenueName:        req.VenueName,
		VenueAddress:     req.VenueAddress,
		VenueCity:        req.VenueCity,
		VenueCountry:     req.VenueCountry,
		FideRated:        req.FIDERated,
		DwzRated:         req.DWZRated,
		OesbRated:        req.OESBRated,
		EntryFeeCents:    req.EntryFeeCents,
		Currency:         req.Currency,
		Status:           StatusDraft,
		Tiebreaks:        req.Tiebreaks,
		Settings:         req.Settings,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Set defaults
	if tournament.PairingEngine == "" {
		tournament.PairingEngine = EngineJaVaFo
	}
	if tournament.PairingSystem == "" {
		tournament.PairingSystem = "dutch"
	}
	if tournament.MinPlayers == 0 {
		tournament.MinPlayers = 4
	}
	if len(tournament.Tiebreaks) == 0 {
		tournament.Tiebreaks = []string{"buchholz_cut1", "buchholz", "sonneborn_berger"}
	}
	if tournament.Currency == "" {
		tournament.Currency = "EUR"
	}

	if err := s.repo.Create(ctx, tournament); err != nil {
		return nil, fmt.Errorf("failed to create tournament: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournament.ID.String()).
		Str("name", tournament.Name).
		Msg("Tournament created")

	return tournament, nil
}

// CloneTournament creates a copy of an existing tournament with a new name
func (s *Service) CloneTournament(ctx context.Context, id uuid.UUID, req *CloneTournamentRequest) (*Tournament, error) {
	// Get the source tournament
	source, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get source tournament: %w", err)
	}

	// Create a new tournament based on the source
	clone := &Tournament{
		ID:                 uuid.New(),
		OrganizationID:     source.OrganizationID,
		Name:               req.Name,
		Description:        source.Description,
		Type:               source.Type,
		Mode:               source.Mode,
		TimeControl:        source.TimeControl,
		TimeControlType:    source.TimeControlType,
		PairingEngine:      source.PairingEngine,
		PairingSystem:      source.PairingSystem,
		AcceleratedPairing: source.AcceleratedPairing,
		MaxPlayers:         source.MaxPlayers,
		MinPlayers:         source.MinPlayers,
		Rounds:             source.Rounds,
		VenueName:          source.VenueName,
		VenueAddress:       source.VenueAddress,
		VenueCity:          source.VenueCity,
		VenueCountry:       source.VenueCountry,
		FideRated:          source.FideRated,
		DwzRated:           source.DwzRated,
		OesbRated:          source.OesbRated,
		EntryFeeCents:      source.EntryFeeCents,
		Currency:           source.Currency,
		Status:             StatusDraft, // Always start as draft
		Tiebreaks:          source.Tiebreaks,
		Settings:           source.Settings,
		CurrentRound:       0,
		ParticipantCount:   0,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Apply override values if provided
	if req.StartDate != nil {
		startDate, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			startDate, err = time.Parse("2006-01-02", *req.StartDate)
			if err != nil {
				return nil, fmt.Errorf("invalid start date format: %w", err)
			}
		}
		clone.StartDate = startDate
	} else {
		// Default to source start date + 1 year
		clone.StartDate = source.StartDate.AddDate(1, 0, 0)
	}

	if req.EndDate != nil {
		endDate, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			endDate, err = time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				return nil, fmt.Errorf("invalid end date format: %w", err)
			}
		}
		clone.EndDate = &endDate
	} else if source.EndDate != nil {
		// Adjust end date by the same offset
		diff := source.EndDate.Sub(source.StartDate)
		newEnd := clone.StartDate.Add(diff)
		clone.EndDate = &newEnd
	}

	if req.RegistrationOpens != nil {
		clone.RegistrationOpens = req.RegistrationOpens
	}
	if req.RegistrationCloses != nil {
		clone.RegistrationCloses = req.RegistrationCloses
	}
	if req.Description != nil {
		clone.Description = req.Description
	}

	if err := s.repo.Create(ctx, clone); err != nil {
		return nil, fmt.Errorf("failed to create cloned tournament: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", clone.ID.String()).
		Str("source_id", id.String()).
		Str("name", clone.Name).
		Msg("Tournament cloned")

	return clone, nil
}

// GetTournament retrieves a tournament by ID
func (s *Service) GetTournament(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	return s.repo.GetByID(ctx, id)
}

// GetTournamentBySlug retrieves a tournament by slug
func (s *Service) GetTournamentBySlug(ctx context.Context, slug string) (*Tournament, error) {
	return s.repo.GetBySlug(ctx, slug)
}

// UpdateTournament updates a tournament
func (s *Service) UpdateTournament(ctx context.Context, id uuid.UUID, req *UpdateTournamentRequest) (*Tournament, error) {
	tournament, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Only allow updates to draft or published tournaments
	if tournament.Status != StatusDraft && tournament.Status != StatusPublished {
		return nil, fmt.Errorf("cannot update tournament in %s status", tournament.Status)
	}

	// Apply updates
	if req.Name != nil {
		tournament.Name = *req.Name
	}
	if req.Description != nil {
		tournament.Description = req.Description
	}
	if req.StartDate != nil {
		startDate, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			startDate, err = time.Parse("2006-01-02", *req.StartDate)
			if err != nil {
				return nil, fmt.Errorf("invalid start date format: %w", err)
			}
		}
		tournament.StartDate = startDate
	}
	if req.EndDate != nil {
		endDate, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			endDate, err = time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				return nil, fmt.Errorf("invalid end date format: %w", err)
			}
		}
		tournament.EndDate = &endDate
	}
	if req.Rounds != nil {
		tournament.Rounds = *req.Rounds
	}
	if req.MaxPlayers != nil {
		tournament.MaxPlayers = req.MaxPlayers
	}
	if req.Tiebreaks != nil {
		tournament.Tiebreaks = req.Tiebreaks
	}
	if req.ScoringSettings != nil {
		// Validate scoring settings
		if errs := req.ScoringSettings.Validate(); len(errs) > 0 {
			return nil, fmt.Errorf("invalid scoring settings: %s", errs[0].Message)
		}
		tournament.ScoringSettings = req.ScoringSettings
	}

	tournament.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf("failed to update tournament: %w", err)
	}

	return tournament, nil
}

// DeleteTournament deletes a tournament
func (s *Service) DeleteTournament(ctx context.Context, id uuid.UUID) error {
	tournament, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Only allow deleting draft tournaments
	if tournament.Status != StatusDraft {
		return fmt.Errorf("cannot delete tournament in %s status", tournament.Status)
	}

	return s.repo.Delete(ctx, id)
}

// ListTournaments lists tournaments with filters
func (s *Service) ListTournaments(ctx context.Context, filter *ListFilters) ([]Tournament, int64, error) {
	return s.repo.List(ctx, filter)
}

// PublishTournament publishes a tournament for registration
func (s *Service) PublishTournament(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	tournament, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if tournament.Status != StatusDraft {
		return nil, fmt.Errorf("can only publish draft tournaments")
	}

	// Validate tournament is ready
	if err := s.validateForPublish(tournament); err != nil {
		return nil, fmt.Errorf("tournament not ready: %w", err)
	}

	tournament.Status = StatusPublished
	tournament.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tournament); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("tournament_id", id.String()).
		Msg("Tournament published")

	return tournament, nil
}

func (s *Service) validateForPublish(t *Tournament) error {
	if t.Name == "" {
		return fmt.Errorf("name is required")
	}
	if t.Rounds == 0 {
		return fmt.Errorf("rounds must be specified")
	}
	if t.StartDate.IsZero() {
		return fmt.Errorf("start date is required")
	}
	return nil
}

// StartRegistration opens registration for a tournament
func (s *Service) StartRegistration(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	tournament, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if tournament.Status != StatusPublished {
		return nil, fmt.Errorf("can only open registration for published tournaments")
	}

	tournament.Status = StatusRegistration
	tournament.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tournament); err != nil {
		return nil, err
	}

	return tournament, nil
}

// StartTournament starts a tournament
func (s *Service) StartTournament(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	tournament, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if tournament.Status != StatusRegistration {
		return nil, fmt.Errorf("can only start tournaments in registration status")
	}

	// Check minimum players
	participants, err := s.repo.ListParticipants(ctx, id)
	if err != nil {
		return nil, err
	}

	activeCount := 0
	for _, p := range participants {
		if p.IsActive {
			activeCount++
		}
	}

	if activeCount < tournament.MinPlayers {
		return nil, fmt.Errorf("minimum %d players required, only %d registered", tournament.MinPlayers, activeCount)
	}

	tournament.Status = StatusOngoing
	tournament.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tournament); err != nil {
		return nil, err
	}

	// Auto-lock ratings if setting is enabled (default: true)
	autoLockRatings := true // default
	if tournament.Settings != nil {
		if val, ok := tournament.Settings["auto_lock_ratings"].(bool); ok {
			autoLockRatings = val
		}
	}

	if autoLockRatings && tournament.RatingsLockedAt == nil {
		if err := s.LockRatings(ctx, id); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to auto-lock ratings at tournament start")
		}
	}

	// Create first round
	round := &Round{
		ID:           uuid.New(),
		TournamentID: id,
		RoundNumber:  1,
		Status:       RoundStatusPending,
	}

	if err := s.repo.CreateRound(ctx, round); err != nil {
		return nil, fmt.Errorf("failed to create first round: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", id.String()).
		Int("participants", activeCount).
		Msg("Tournament started")

	return tournament, nil
}

// RegisterParticipant registers a player for a tournament
func (s *Service) RegisterParticipant(ctx context.Context, tournamentID uuid.UUID, req *RegisterParticipantRequest) (*Participant, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Check registration is open
	if tournament.Status != StatusPublished && tournament.Status != StatusRegistration {
		return nil, fmt.Errorf("registration is not open")
	}

	// Check registration dates
	now := time.Now()
	if tournament.RegistrationOpens != nil && now.Before(*tournament.RegistrationOpens) {
		return nil, fmt.Errorf("registration has not opened yet")
	}
	if tournament.RegistrationCloses != nil && now.After(*tournament.RegistrationCloses) {
		return nil, fmt.Errorf("registration has closed")
	}

	// Check max players
	if tournament.MaxPlayers != nil {
		participants, err := s.repo.ListParticipants(ctx, tournamentID)
		if err != nil {
			return nil, err
		}

		activeCount := 0
		for _, p := range participants {
			if p.IsActive && p.RegistrationStatus != "withdrawn" {
				activeCount++
			}
		}

		if activeCount >= *tournament.MaxPlayers {
			return nil, fmt.Errorf("tournament is full")
		}
	}

	// Check if player already registered
	existing, _ := s.repo.GetParticipantByPlayerID(ctx, tournamentID, req.PlayerID)
	if existing != nil {
		return nil, fmt.Errorf("player already registered")
	}

	// Auto-fetch player rating if not provided
	startingRating := req.StartingRating
	if startingRating == 0 {
		// Determine rating type based on tournament time control
		ratingType := "standard"
		if tournament.TimeControlType == TimeControlRapid {
			ratingType = "rapid"
		} else if tournament.TimeControlType == TimeControlBlitz {
			ratingType = "blitz"
		}
		fetchedRating, err := s.repo.GetPlayerRating(ctx, req.PlayerID, ratingType)
		if err == nil && fetchedRating > 0 {
			startingRating = fetchedRating
		}
	}

	participant := &Participant{
		ID:                 uuid.New(),
		TournamentID:       tournamentID,
		SectionID:          req.SectionID,
		PlayerID:           req.PlayerID,
		RegistrationStatus: "registered",
		RegisteredAt:       time.Now(),
		StartingRating:     startingRating,
		IsActive:           true,
		PaymentStatus:      "pending",
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to register participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("player_id", req.PlayerID.String()).
		Msg("Player registered")

	return participant, nil
}

// WithdrawParticipant withdraws a player from a tournament
func (s *Service) WithdrawParticipant(ctx context.Context, tournamentID, participantID uuid.UUID) error {
	participant, err := s.repo.GetParticipant(ctx, participantID)
	if err != nil {
		return err
	}

	if participant.TournamentID != tournamentID {
		return fmt.Errorf("participant not in this tournament")
	}

	participant.RegistrationStatus = "withdrawn"
	participant.IsActive = false

	if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to withdraw participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("participant_id", participantID.String()).
		Msg("Participant withdrawn")

	return nil
}

// GetParticipants gets participants with optional filters
func (s *Service) GetParticipants(ctx context.Context, tournamentID uuid.UUID, filter *ParticipantFilter) ([]Participant, error) {
	participants, err := s.repo.ListParticipants(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	// Apply filter in memory if needed
	if filter != nil && filter.ActiveOnly {
		filtered := make([]Participant, 0)
		for _, p := range participants {
			if p.IsActive {
				filtered = append(filtered, p)
			}
		}
		return filtered, nil
	}
	return participants, nil
}

// AssignStartRanks assigns start ranks based on effective rating (ManualOverride or StartingRating)
func (s *Service) AssignStartRanks(ctx context.Context, tournamentID uuid.UUID) error {
	participants, err := s.GetParticipants(ctx, tournamentID, &ParticipantFilter{
		ActiveOnly: true,
	})
	if err != nil {
		return err
	}

	// Sort by effective rating descending
	for i := 0; i < len(participants); i++ {
		for j := i + 1; j < len(participants); j++ {
			if participants[j].EffectiveRating() > participants[i].EffectiveRating() {
				participants[i], participants[j] = participants[j], participants[i]
			}
		}
	}

	// Assign ranks
	for i := range participants {
		participants[i].StartRank = i + 1
		if err := s.repo.UpdateParticipant(ctx, &participants[i]); err != nil {
			return err
		}
	}

	return nil
}

// GeneratePairings generates pairings for a round
func (s *Service) GeneratePairings(ctx context.Context, tournamentID uuid.UUID, roundNumber int, userID uuid.UUID) ([]Game, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if tournament.Status != StatusOngoing {
		return nil, fmt.Errorf("tournament is not ongoing")
	}

	// Set pairing lock to prevent result entry during pairing generation
	previousStatus := tournament.Status
	tournament.Status = StatusPairing
	if err := s.repo.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf("failed to set pairing lock: %w", err)
	}

	// Ensure we unlock on exit (both success and error)
	defer func() {
		tournament.Status = previousStatus
		if updateErr := s.repo.Update(ctx, tournament); updateErr != nil {
			s.logger.Error().Err(updateErr).Msg("Failed to release pairing lock")
		}
	}()

	// Get the round
	round, err := s.repo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err != nil {
		return nil, err
	}

	if round.Status != RoundStatusPending {
		return nil, fmt.Errorf("round is not in pending status")
	}

	// Auto-assign start ranks if not already assigned (required for round 1)
	participants, err := s.GetParticipants(ctx, tournamentID, &ParticipantFilter{
		ActiveOnly: true,
	})
	if err != nil {
		return nil, err
	}

	needsRanks := false
	for _, p := range participants {
		if p.StartRank == 0 {
			needsRanks = true
			break
		}
	}

	if needsRanks {
		if err := s.AssignStartRanks(ctx, tournamentID); err != nil {
			return nil, fmt.Errorf("failed to assign start ranks: %w", err)
		}
		// Re-fetch participants with updated ranks
		participants, err = s.GetParticipants(ctx, tournamentID, &ParticipantFilter{
			ActiveOnly: true,
		})
		if err != nil {
			return nil, err
		}
	}

	// Generate pairings based on tournament type
	var result *engine.PairingResult
	var pairingErr error

	switch tournament.Type {
	case TypeRoundRobin, TypeKnockout, TypeScheveningen, TypeArena:
		// Use built-in engines via general pairing request
		result, pairingErr = s.generateBuiltInPairings(ctx, tournament, participants, roundNumber)
	case TypeTeamSwiss:
		// Team Swiss needs special handling
		result, pairingErr = s.generateTeamSwissPairings(ctx, tournament, roundNumber)
	default:
		// Swiss tournaments use TRF-based engines (JaVaFo, BBP, Monrad, Keizer)
		result, pairingErr = s.generateSwissPairings(ctx, tournament, roundNumber)
	}

	if pairingErr != nil {
		return nil, fmt.Errorf("pairing generation failed: %w", pairingErr)
	}

	// Convert pairings to games (re-fetch participants to get updated ranks if assigned)
	participants, err = s.GetParticipants(ctx, tournamentID, &ParticipantFilter{
		ActiveOnly: true,
	})
	if err != nil {
		return nil, err
	}

	// Build start rank to participant map
	rankToParticipant := make(map[int]*Participant)
	for i := range participants {
		rankToParticipant[participants[i].StartRank] = &participants[i]
	}

	var games []Game
	for _, p := range result.Pairings {
		game := Game{
			ID:           uuid.New(),
			TournamentID: tournamentID,
			RoundID:      round.ID,
			BoardNumber:  p.Board,
			Status:       GameStatusScheduled,
			CreatedAt:    time.Now(),
		}

		if p.IsBye {
			if wp, ok := rankToParticipant[p.WhiteStartNo]; ok {
				game.WhiteParticipantID = &wp.ID
			}
			game.Result = stringPtr("BYE")
			game.WhiteScore = floatPtr(1.0) // Standard FIDE bye is 1 point
			game.BlackScore = floatPtr(0)
			game.Status = GameStatusCompleted
		} else {
			if wp, ok := rankToParticipant[p.WhiteStartNo]; ok {
				game.WhiteParticipantID = &wp.ID
			}
			if bp, ok := rankToParticipant[p.BlackStartNo]; ok {
				game.BlackParticipantID = &bp.ID
			}
		}

		games = append(games, game)
	}

	// Save games
	if err := s.repo.CreateGames(ctx, games); err != nil {
		return nil, fmt.Errorf("failed to save pairings: %w", err)
	}

	// Update participant scores for BYE games
	scoring := tournament.ScoringSettings
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}
	for _, game := range games {
		if game.Result != nil && *game.Result == "BYE" {
			if err := s.updateParticipantScores(ctx, &game, scoring); err != nil {
				s.logger.Warn().Err(err).Msg("Failed to update BYE participant score")
			}
		}
	}

	// Update round status
	now := time.Now()
	round.Status = RoundStatusPaired
	round.PairedAt = &now
	round.PairedBy = &userID
	engineStr := string(tournament.PairingEngine)
	round.PairingEngineUsed = &engineStr

	if err := s.repo.UpdateRound(ctx, round); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("round", roundNumber).
		Int("games", len(games)).
		Msg("Pairings generated")

	return games, nil
}

func floatPtr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}

func (s *Service) generateTRFForPairing(ctx context.Context, tournament *Tournament, roundNumber int) (string, error) {
	participants, err := s.GetParticipants(ctx, tournament.ID, &ParticipantFilter{
		ActiveOnly: true,
	})
	if err != nil {
		return "", err
	}

	generator := trf.NewGenerator()
	city := ""
	if tournament.VenueCity != nil {
		city = *tournament.VenueCity
	}
	generator.SetTournamentInfo(tournament.Name, city, "", tournament.Rounds)
	generator.SetDates(tournament.StartDate, time.Time{})

	for _, p := range participants {
		// Build player name for TRF
		playerName := ""
		if p.PlayerLastName != nil && p.PlayerFirstName != nil {
			playerName = *p.PlayerLastName + ", " + *p.PlayerFirstName
		} else if p.PlayerLastName != nil {
			playerName = *p.PlayerLastName
		} else if p.PlayerFirstName != nil {
			playerName = *p.PlayerFirstName
		} else {
			playerName = fmt.Sprintf("Player %d", p.StartRank)
		}

		fideID := ""
		if p.PlayerFideID != nil {
			fideID = *p.PlayerFideID
		}

		player := trf.TRFPlayer{
			StartNo:    p.StartRank,
			Name:       playerName,
			Rating:     p.EffectiveRating(),
			FideID:     fideID,
			Points:     p.Points,
			Rank:       p.StartRank,
		}

		// Get previous round results
		games, err := s.repo.GetPlayerGames(ctx, tournament.ID, p.ID)
		if err != nil {
			return "", err
		}

		for _, g := range games {
			if g.Result == nil || *g.Result == "" {
				continue
			}

			result := trf.TRFRoundResult{
				Round: g.BoardNumber, // This should be round number, needs fixing
			}

			// Handle BYE games
			if *g.Result == "BYE" {
				result.Color = "-"
				if g.WhiteScore != nil && *g.WhiteScore == 1 {
					result.Result = "F" // Full-point bye
				} else {
					result.Result = "H" // Half-point bye
				}
				result.OpponentNo = 0
				player.RoundResults = append(player.RoundResults, result)
				continue
			}

			// Determine color and opponent
			if g.WhiteParticipantID != nil && *g.WhiteParticipantID == p.ID {
				result.Color = "w"
				if g.BlackParticipantID != nil {
					// Find opponent's start rank
					for _, op := range participants {
						if op.ID == *g.BlackParticipantID {
							result.OpponentNo = op.StartRank
							break
						}
					}
				}
				if g.WhiteScore != nil {
					if *g.WhiteScore == 1 {
						result.Result = "1"
					} else if *g.WhiteScore == 0.5 {
						result.Result = "="
					} else {
						result.Result = "0"
					}
				}
			} else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID {
				result.Color = "b"
				if g.WhiteParticipantID != nil {
					for _, op := range participants {
						if op.ID == *g.WhiteParticipantID {
							result.OpponentNo = op.StartRank
							break
						}
					}
				}
				if g.BlackScore != nil {
					if *g.BlackScore == 1 {
						result.Result = "1"
					} else if *g.BlackScore == 0.5 {
						result.Result = "="
					} else {
						result.Result = "0"
					}
				}
			}

			player.RoundResults = append(player.RoundResults, result)
		}

		generator.AddPlayer(player)
	}

	return generator.Generate(), nil
}

// GetRound gets a specific round
func (s *Service) GetRound(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*Round, error) {
	return s.repo.GetRoundByNumber(ctx, tournamentID, roundNumber)
}

// GetRounds gets all rounds for a tournament
func (s *Service) GetRounds(ctx context.Context, tournamentID uuid.UUID) ([]Round, error) {
	return s.repo.ListRounds(ctx, tournamentID)
}

// GetPairings gets pairings for a round
func (s *Service) GetPairings(ctx context.Context, tournamentID uuid.UUID, roundNumber int) ([]Game, error) {
	round, err := s.repo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err != nil {
		return nil, err
	}

	return s.repo.ListGamesByRound(ctx, round.ID)
}

// StartRound starts a round
func (s *Service) StartRound(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*Round, error) {
	round, err := s.repo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err != nil {
		return nil, err
	}

	if round.Status != RoundStatusPaired {
		return nil, fmt.Errorf("round must be paired before starting")
	}

	now := time.Now()
	round.Status = RoundStatusOngoing
	round.ActualStart = &now

	if err := s.repo.UpdateRound(ctx, round); err != nil {
		return nil, err
	}

	return round, nil
}

// EnterResult enters a game result
func (s *Service) EnterResult(ctx context.Context, gameID uuid.UUID, req *EnterResultRequest, userID uuid.UUID) (*Game, error) {
	game, err := s.repo.GetGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	// Check if tournament is locked for pairing
	tournament, err := s.repo.GetByID(ctx, game.TournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament: %w", err)
	}
	if tournament.Status == StatusPairing {
		return nil, fmt.Errorf("tournament is locked for pairing - results cannot be entered")
	}

	if game.Status == GameStatusCompleted && game.Result != nil && *game.Result != "" {
		return nil, fmt.Errorf("result already entered")
	}

	// Get scoring settings
	scoring := tournament.ScoringSettings
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}

	// Parse and validate result using scoring settings
	whiteScore, blackScore := ParseResultWithScoring(req.Result, scoring)
	// Note: ParseResultWithScoring doesn't return error, invalid results default to loss-loss

	now := time.Now()
	game.Result = &req.Result
	game.WhiteScore = &whiteScore
	game.BlackScore = &blackScore
	game.Status = GameStatusCompleted
	game.ResultEnteredBy = &userID
	game.ResultEnteredAt = &now

	if req.PGN != "" {
		game.PGN = &req.PGN
	}

	if err := s.repo.UpdateGame(ctx, game); err != nil {
		return nil, err
	}

	// Update participant scores
	if err := s.updateParticipantScores(ctx, game, scoring); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("game_id", gameID.String()).
		Str("result", req.Result).
		Msg("Result entered")

	return game, nil
}

func (s *Service) updateParticipantScores(ctx context.Context, game *Game, scoring *ScoringSettings) error {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}

	if game.WhiteParticipantID != nil && game.WhiteScore != nil {
		participant, err := s.repo.GetParticipant(ctx, *game.WhiteParticipantID)
		if err != nil {
			return err
		}
		participant.Points += *game.WhiteScore

		// Determine result type using scoring settings
		if IsWin(*game.WhiteScore, scoring) {
			participant.Wins++
		} else if IsDraw(*game.WhiteScore, scoring) {
			participant.Draws++
		} else {
			participant.Losses++
		}
		if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
			return err
		}
	}

	if game.BlackParticipantID != nil && game.BlackScore != nil {
		participant, err := s.repo.GetParticipant(ctx, *game.BlackParticipantID)
		if err != nil {
			return err
		}
		participant.Points += *game.BlackScore

		// Determine result type using scoring settings
		if IsWin(*game.BlackScore, scoring) {
			participant.Wins++
		} else if IsDraw(*game.BlackScore, scoring) {
			participant.Draws++
		} else {
			participant.Losses++
		}
		if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
			return err
		}
	}

	return nil
}

// CompleteRound completes a round and optionally creates the next one
func (s *Service) CompleteRound(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*Round, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	round, err := s.repo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err != nil {
		return nil, err
	}

	if round.Status != RoundStatusOngoing {
		return nil, fmt.Errorf("round is not ongoing")
	}

	// Check all games have results
	games, err := s.repo.ListGamesByRound(ctx, round.ID)
	if err != nil {
		return nil, err
	}

	for _, g := range games {
		if g.Result == nil || *g.Result == "" {
			return nil, fmt.Errorf("game on board %d has no result", g.BoardNumber)
		}
	}

	now := time.Now()
	round.Status = RoundStatusCompleted
	round.CompletedAt = &now

	if err := s.repo.UpdateRound(ctx, round); err != nil {
		return nil, err
	}

	// Recalculate tiebreaks
	if err := s.RecalculateTiebreaks(ctx, tournamentID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to recalculate tiebreaks")
	}

	// Create next round if not last round
	if roundNumber < tournament.Rounds {
		nextRound := &Round{
			ID:           uuid.New(),
			TournamentID: tournamentID,
			RoundNumber:  roundNumber + 1,
			Status:       RoundStatusPending,
		}

		if err := s.repo.CreateRound(ctx, nextRound); err != nil {
			return nil, fmt.Errorf("failed to create next round: %w", err)
		}
	} else {
		// Tournament complete
		tournament.Status = StatusCompleted
		tournament.UpdatedAt = time.Now()
		if err := s.repo.Update(ctx, tournament); err != nil {
			return nil, err
		}

		s.logger.Info().
			Str("tournament_id", tournamentID.String()).
			Msg("Tournament completed")
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("round", roundNumber).
		Msg("Round completed")

	return round, nil
}

// GetStandings gets current tournament standings
func (s *Service) GetStandings(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error) {
	return s.repo.GetStandings(ctx, tournamentID)
}

// RecalculateTiebreaks recalculates all tiebreaks for a tournament
func (s *Service) RecalculateTiebreaks(ctx context.Context, tournamentID uuid.UUID) error {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	participants, err := s.GetParticipants(ctx, tournamentID, &ParticipantFilter{
		ActiveOnly: true,
	})
	if err != nil {
		return err
	}

	// Convert tiebreak codes
	var tiebreakCodes []tiebreak.TiebreakCode
	for _, tb := range tournament.Tiebreaks {
		tiebreakCodes = append(tiebreakCodes, tiebreak.TiebreakCode(tb))
	}

	calc := tiebreak.NewCalculator(tiebreakCodes, tournament.Rounds)

	// Build player data
	for _, p := range participants {
		games, err := s.repo.GetPlayerGames(ctx, tournamentID, p.ID)
		if err != nil {
			continue
		}

		playerData := &tiebreak.PlayerData{
			ID:           p.ID,
			StartRank:    p.StartRank,
			Rating:       p.EffectiveRating(),
			Points:       p.Points,
			Wins:         p.Wins,
			Draws:        p.Draws,
			Losses:       p.Losses,
		}

		// Build results
		for _, g := range games {
			if g.Result == nil || *g.Result == "" {
				continue
			}

			result := tiebreak.PlayerResult{}

			if g.WhiteParticipantID != nil && *g.WhiteParticipantID == p.ID {
				if g.WhiteScore != nil {
					result.Score = *g.WhiteScore
				}
				result.PlayedWhite = true
				if g.BlackParticipantID != nil {
					result.OpponentID = *g.BlackParticipantID
					// Find opponent's points
					for _, op := range participants {
						if op.ID == *g.BlackParticipantID {
							result.OpponentPts = op.Points
							break
						}
					}
				}
			} else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID {
				if g.BlackScore != nil {
					result.Score = *g.BlackScore
				}
				result.PlayedWhite = false
				playerData.GamesAsBlack++
				if result.Score == 1 {
					playerData.WinsAsBlack++
				}
				if g.WhiteParticipantID != nil {
					result.OpponentID = *g.WhiteParticipantID
					for _, op := range participants {
						if op.ID == *g.WhiteParticipantID {
							result.OpponentPts = op.Points
							break
						}
					}
				}
			}

			playerData.Results = append(playerData.Results, result)
			playerData.GamesPlayed++
		}

		calc.AddPlayer(playerData)
	}

	// Calculate tiebreaks
	results := calc.Calculate()

	// Update participants with tiebreak scores
	for i := range participants {
		if tbs, ok := results[participants[i].ID]; ok {
			if buchholz, ok := tbs[tiebreak.TBBuchholz]; ok {
				participants[i].Buchholz = buchholz
			}
			if buchholzCut1, ok := tbs[tiebreak.TBBuchholzCut1]; ok {
				participants[i].BuchholzCut1 = buchholzCut1
			}
			if sb, ok := tbs[tiebreak.TBSonnebornBerger]; ok {
				participants[i].SonnebornBerger = sb
			}
		}

		if err := s.repo.UpdateParticipant(ctx, &participants[i]); err != nil {
			return err
		}
	}

	return nil
}

// GetCrosstable generates a crosstable for the tournament
func (s *Service) GetCrosstable(ctx context.Context, tournamentID uuid.UUID) (*Crosstable, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	participants, err := s.repo.GetStandings(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	crosstable := &Crosstable{
		TournamentID:   tournamentID,
		TournamentName: tournament.Name,
		Rounds:         tournament.Rounds,
		Players:        make([]CrosstablePlayer, len(participants)),
	}

	// Build participant ID to index map
	idToIndex := make(map[uuid.UUID]int)
	for i, p := range participants {
		idToIndex[p.ID] = i
	}

	for i, p := range participants {
		// Build player name from joined participant data
		var name string
		if p.PlayerLastName != nil && p.PlayerFirstName != nil {
			name = *p.PlayerLastName + ", " + *p.PlayerFirstName
		} else if p.PlayerLastName != nil {
			name = *p.PlayerLastName
		} else if p.PlayerFirstName != nil {
			name = *p.PlayerFirstName
		}

		player := CrosstablePlayer{
			Rank:           i + 1,
			ParticipantID:  p.ID,
			PlayerID:       p.PlayerID,
			Name:           name,
			Rating:         p.EffectiveRating(),
			Points:         p.Points,
			Buchholz:       p.Buchholz,
			SonnebornBerger: p.SonnebornBerger,
			Results:        make([]CrosstableResult, tournament.Rounds),
		}

		// Get all games for this participant
		games, err := s.repo.GetPlayerGames(ctx, tournamentID, p.ID)
		if err != nil {
			continue
		}

		for _, g := range games {
			// Determine round number from game
			round, _ := s.repo.GetRound(ctx, g.RoundID)
			if round == nil {
				continue
			}

			result := CrosstableResult{
				Round: round.RoundNumber,
			}

			if g.WhiteParticipantID != nil && *g.WhiteParticipantID == p.ID {
				result.Color = "w"
				if g.BlackParticipantID != nil {
					if idx, ok := idToIndex[*g.BlackParticipantID]; ok {
						result.OpponentRank = idx + 1
					}
				}
				if g.WhiteScore != nil {
					result.Score = *g.WhiteScore
				}
			} else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID {
				result.Color = "b"
				if g.WhiteParticipantID != nil {
					if idx, ok := idToIndex[*g.WhiteParticipantID]; ok {
						result.OpponentRank = idx + 1
					}
				}
				if g.BlackScore != nil {
					result.Score = *g.BlackScore
				}
			}

			if round.RoundNumber > 0 && round.RoundNumber <= tournament.Rounds {
				player.Results[round.RoundNumber-1] = result
			}
		}

		crosstable.Players[i] = player
	}

	return crosstable, nil
}

// Crosstable represents a tournament crosstable
type Crosstable struct {
	TournamentID   uuid.UUID          `json:"tournamentId"`
	TournamentName string             `json:"tournamentName"`
	Rounds         int                `json:"rounds"`
	Players        []CrosstablePlayer `json:"players"`
}

// CrosstablePlayer represents a player in the crosstable
type CrosstablePlayer struct {
	Rank            int               `json:"rank"`
	ParticipantID   uuid.UUID         `json:"participantId"`
	PlayerID        uuid.UUID         `json:"playerId"`
	Name            string            `json:"name"`
	Rating          int               `json:"rating"`
	Points          float64           `json:"points"`
	Buchholz        float64           `json:"buchholz"`
	SonnebornBerger float64           `json:"sonnebornBerger"`
	Results         []CrosstableResult `json:"results"`
}

// CrosstableResult represents a single result in the crosstable
type CrosstableResult struct {
	Round        int     `json:"round"`
	OpponentRank int     `json:"opponentRank"`
	Color        string  `json:"color"`
	Score        float64 `json:"score"`
}

// ExportTRF exports the tournament to TRF format
func (s *Service) ExportTRF(ctx context.Context, tournamentID uuid.UUID) (string, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return "", err
	}

	participants, err := s.GetParticipants(ctx, tournamentID, &ParticipantFilter{})
	if err != nil {
		return "", err
	}

	generator := trf.NewGenerator()
	city := ""
	if tournament.VenueCity != nil {
		city = *tournament.VenueCity
	}
	generator.SetTournamentInfo(tournament.Name, city, "", tournament.Rounds)
	var endDate time.Time
	if tournament.EndDate != nil {
		endDate = *tournament.EndDate
	}
	generator.SetDates(tournament.StartDate, endDate)
	generator.SetTimeControl(tournament.TimeControl)

	switch tournament.Type {
	case TypeSwiss:
		generator.SetTournamentType("Swiss")
	case TypeRoundRobin:
		generator.SetTournamentType("Round Robin")
	}

	for _, p := range participants {
		// Build player name
		playerName := ""
		if p.PlayerLastName != nil && p.PlayerFirstName != nil {
			playerName = *p.PlayerLastName + ", " + *p.PlayerFirstName
		} else if p.PlayerLastName != nil {
			playerName = *p.PlayerLastName
		}

		player := trf.TRFPlayer{
			StartNo: p.StartRank,
			Name:    playerName,
			Rating:  p.StartingRating,
			Points:  p.Points,
			Rank:    p.StartRank, // This should be final rank
		}

		// Get all games
		games, err := s.repo.GetPlayerGames(ctx, tournamentID, p.ID)
		if err != nil {
			continue
		}

		for _, g := range games {
			round, _ := s.repo.GetRound(ctx, g.RoundID)
			if round == nil {
				continue
			}

			result := trf.TRFRoundResult{
				Round: round.RoundNumber,
			}

			// Handle BYE games
			if g.Result != nil && *g.Result == "BYE" {
				result.Color = "-"
				if g.WhiteScore != nil && *g.WhiteScore == 1 {
					result.Result = "F" // Full-point bye
				} else {
					result.Result = "H" // Half-point bye
				}
				result.OpponentNo = 0
				player.RoundResults = append(player.RoundResults, result)
				continue
			}

			if g.WhiteParticipantID != nil && *g.WhiteParticipantID == p.ID {
				result.Color = "w"
				if g.BlackParticipantID != nil {
					for _, op := range participants {
						if op.ID == *g.BlackParticipantID {
							result.OpponentNo = op.StartRank
							break
						}
					}
				}
				if g.WhiteScore != nil {
					if *g.WhiteScore == 1 {
						result.Result = "1"
					} else if *g.WhiteScore == 0.5 {
						result.Result = "="
					} else {
						result.Result = "0"
					}
				}
			} else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID {
				result.Color = "b"
				if g.WhiteParticipantID != nil {
					for _, op := range participants {
						if op.ID == *g.WhiteParticipantID {
							result.OpponentNo = op.StartRank
							break
						}
					}
				}
				if g.BlackScore != nil {
					if *g.BlackScore == 1 {
						result.Result = "1"
					} else if *g.BlackScore == 0.5 {
						result.Result = "="
					} else {
						result.Result = "0"
					}
				}
			}

			player.RoundResults = append(player.RoundResults, result)
		}

		generator.AddPlayer(player)
	}

	return generator.Generate(), nil
}

// AddForbiddenPair adds a forbidden pair
func (s *Service) AddForbiddenPair(ctx context.Context, tournamentID uuid.UUID, player1ID, player2ID uuid.UUID, reason string) error {
	pair := &ForbiddenPair{
		ID:           uuid.New(),
		TournamentID: tournamentID,
		Player1ID:    player1ID,
		Player2ID:    player2ID,
		Reason:       reason,
		CreatedAt:    time.Now(),
	}

	return s.repo.AddForbiddenPair(ctx, pair)
}

// RemoveForbiddenPair removes a forbidden pair
func (s *Service) RemoveForbiddenPair(ctx context.Context, id uuid.UUID) error {
	return s.repo.RemoveForbiddenPair(ctx, id)
}

// GetForbiddenPairs gets all forbidden pairs for a tournament
func (s *Service) GetForbiddenPairs(ctx context.Context, tournamentID uuid.UUID) ([]ForbiddenPair, error) {
	return s.repo.ListForbiddenPairs(ctx, tournamentID)
}

// generateBuiltInPairings generates pairings using built-in engines (RoundRobin, Knockout, Scheveningen, Arena)
func (s *Service) generateBuiltInPairings(ctx context.Context, tournament *Tournament, participants []Participant, roundNumber int) (*engine.PairingResult, error) {
	if s.engineRegistry == nil {
		return nil, fmt.Errorf("engine registry not configured")
	}

	// Map tournament type to engine type
	var engineType engine.TournamentType
	switch tournament.Type {
	case TypeRoundRobin:
		engineType = engine.TournamentTypeRoundRobin
	case TypeKnockout:
		engineType = engine.TournamentTypeKnockout
	case TypeScheveningen:
		engineType = engine.TournamentTypeScheveningen
	case TypeArena:
		engineType = engine.TournamentTypeArena
	default:
		return nil, fmt.Errorf("unsupported tournament type for built-in engine: %s", tournament.Type)
	}

	// Get the general pairing engine
	eng, err := s.engineRegistry.GetGeneralEngineForTournament(engineType)
	if err != nil {
		return nil, fmt.Errorf("failed to get engine: %w", err)
	}

	// Convert participants to engine.Player format
	players := make([]engine.Player, 0, len(participants))
	for _, p := range participants {
		if !p.IsActive {
			continue
		}

		// Build player name
		name := ""
		if p.PlayerLastName != nil && p.PlayerFirstName != nil {
			name = *p.PlayerLastName + ", " + *p.PlayerFirstName
		} else if p.PlayerLastName != nil {
			name = *p.PlayerLastName
		} else if p.PlayerFirstName != nil {
			name = *p.PlayerFirstName
		}

		player := engine.Player{
			StartNo:  p.StartRank,
			Name:     name,
			Rating:   p.EffectiveRating(),
			Points:   p.Points,
			IsActive: p.IsActive,
		}

		// Fetch previous opponents for rematch avoidance
		games, _ := s.repo.GetPlayerGames(ctx, tournament.ID, p.ID)
		for _, g := range games {
			if g.Result == nil || *g.Result == "" {
				continue
			}
			var opponentID uuid.UUID
			if g.WhiteParticipantID != nil && *g.WhiteParticipantID == p.ID {
				if g.BlackParticipantID != nil {
					opponentID = *g.BlackParticipantID
				}
			} else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID {
				if g.WhiteParticipantID != nil {
					opponentID = *g.WhiteParticipantID
				}
			}
			// Find opponent's start rank
			for _, op := range participants {
				if op.ID == opponentID {
					player.Opponents = append(player.Opponents, op.StartRank)
					break
				}
			}
		}

		players = append(players, player)
	}

	// Build the request
	req := &engine.GeneralPairingRequest{
		TournamentType: engineType,
		Players:        players,
		RoundNumber:    roundNumber,
		TotalRounds:    tournament.Rounds,
		PairingSystem:  string(tournament.PairingSystem),
	}

	// Generate pairings
	return eng.GenerateGeneralPairings(ctx, req)
}

// generateTeamSwissPairings generates pairings for team Swiss tournaments
func (s *Service) generateTeamSwissPairings(ctx context.Context, tournament *Tournament, roundNumber int) (*engine.PairingResult, error) {
	if s.engineRegistry == nil {
		return nil, fmt.Errorf("engine registry not configured")
	}

	// Get the team Swiss engine
	eng, err := s.engineRegistry.GetGeneralEngineForTournament(engine.TournamentTypeTeamSwiss)
	if err != nil {
		return nil, fmt.Errorf("failed to get team Swiss engine: %w", err)
	}

	// Fetch teams from repository (this requires the team data to be stored)
	teams, err := s.repo.GetTeams(ctx, tournament.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}

	if len(teams) < 2 {
		return nil, fmt.Errorf("need at least 2 teams for team Swiss")
	}

	// Convert to engine.Team format
	engineTeams := make([]engine.Team, 0, len(teams))
	for _, t := range teams {
		team := engine.Team{
			ID:     t.ID,
			Name:   t.Name,
			Points: t.Points,
		}

		// Get team players
		for _, p := range t.Players {
			player := engine.Player{
				StartNo:     p.StartRank,
				Name:        p.Name,
				Rating:      p.Rating,
				BoardNumber: p.BoardNumber,
				IsActive:    p.IsActive,
			}
			team.Players = append(team.Players, player)
		}

		engineTeams = append(engineTeams, team)
	}

	// Build the request
	req := &engine.GeneralPairingRequest{
		TournamentType: engine.TournamentTypeTeamSwiss,
		Teams:          engineTeams,
		RoundNumber:    roundNumber,
		TotalRounds:    tournament.Rounds,
	}

	return eng.GenerateGeneralPairings(ctx, req)
}

// generateSwissPairings generates pairings for Swiss tournaments using TRF-based engines
func (s *Service) generateSwissPairings(ctx context.Context, tournament *Tournament, roundNumber int) (*engine.PairingResult, error) {
	// Generate TRF for the pairing engine
	trfData, err := s.generateTRFForPairing(ctx, tournament, roundNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TRF: %w", err)
	}

	// Try engine registry first if available
	if s.engineRegistry != nil {
		// Get engine based on configuration or use default Swiss engine
		var eng engine.PairingEngine
		var engineErr error

		switch tournament.PairingEngine {
		case EngineMonrad:
			eng, engineErr = s.engineRegistry.GetEngine("monrad")
		case EngineKeizer:
			eng, engineErr = s.engineRegistry.GetEngine("keizer")
		case EngineBBP:
			eng, engineErr = s.engineRegistry.GetEngine("bbp")
		case EngineJaVaFo:
			eng, engineErr = s.engineRegistry.GetEngine("javafo")
		default:
			// Use the best available Swiss engine
			eng, engineErr = s.engineRegistry.GetSwissEngine()
		}

		if engineErr == nil && eng != nil {
			// Create pairing request
			pairingReq := &engine.PairingRequest{
				TRFData:       trfData,
				RoundNumber:   roundNumber,
				PairingSystem: string(tournament.PairingSystem),
			}

			return eng.GeneratePairings(ctx, pairingReq)
		}
	}

	// Fall back to legacy single engine
	if s.pairingEngine == nil {
		return nil, fmt.Errorf("no pairing engine available")
	}

	pairingReq := &engine.PairingRequest{
		TRFData:       trfData,
		RoundNumber:   roundNumber,
		PairingSystem: string(tournament.PairingSystem),
	}

	return s.pairingEngine.GeneratePairings(ctx, pairingReq)
}

// LockRatings locks the ratings for a tournament by refreshing all participant ratings
// from the current player profile and marking the tournament as ratings locked.
func (s *Service) LockRatings(ctx context.Context, tournamentID uuid.UUID) error {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	// Only allow locking for non-completed tournaments
	if tournament.Status == StatusCompleted || tournament.Status == StatusCancelled {
		return fmt.Errorf("cannot lock ratings for %s tournament", tournament.Status)
	}

	// Already locked?
	if tournament.RatingsLockedAt != nil {
		s.logger.Info().
			Str("tournament_id", tournamentID.String()).
			Time("locked_at", *tournament.RatingsLockedAt).
			Msg("Ratings already locked")
		return nil
	}

	participants, err := s.repo.ListParticipants(ctx, tournamentID)
	if err != nil {
		return fmt.Errorf("failed to list participants: %w", err)
	}

	// Determine rating type based on time control
	ratingType := s.getRatingType(tournament.TimeControlType)

	// Refresh each participant's starting rating from current player profile
	for i := range participants {
		currentRating, err := s.repo.GetPlayerRating(ctx, participants[i].PlayerID, ratingType)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Str("player_id", participants[i].PlayerID.String()).
				Msg("Failed to fetch current rating, keeping existing")
			continue
		}

		if currentRating > 0 {
			participants[i].StartingRating = currentRating
			if err := s.repo.UpdateParticipant(ctx, &participants[i]); err != nil {
				return fmt.Errorf("failed to update participant rating: %w", err)
			}
		}
	}

	// Mark tournament as ratings locked
	now := time.Now()
	tournament.RatingsLockedAt = &now
	if err := s.repo.Update(ctx, tournament); err != nil {
		return fmt.Errorf("failed to update tournament: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("participants", len(participants)).
		Msg("Ratings locked")

	return nil
}

// getRatingType returns the rating type based on time control
func (s *Service) getRatingType(timeControlType TimeControlType) string {
	switch timeControlType {
	case TimeControlRapid:
		return "rapid"
	case TimeControlBlitz:
		return "blitz"
	default:
		return "standard"
	}
}

// SetParticipantRatingOverride sets or clears a manual rating override for a participant
func (s *Service) SetParticipantRatingOverride(ctx context.Context, participantID uuid.UUID, rating *int) error {
	participant, err := s.repo.GetParticipant(ctx, participantID)
	if err != nil {
		return err
	}

	participant.ManualRatingOverride = rating
	if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	if rating != nil {
		s.logger.Info().
			Str("participant_id", participantID.String()).
			Int("rating_override", *rating).
			Msg("Rating override set")
	} else {
		s.logger.Info().
			Str("participant_id", participantID.String()).
			Msg("Rating override cleared")
	}

	return nil
}

// UpdateByeSettings updates the bye settings for a tournament
func (s *Service) UpdateByeSettings(ctx context.Context, tournamentID uuid.UUID, settings *ByeSettings) (*Tournament, []FIDEViolation, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, nil, err
	}

	// Only allow updates before tournament completes
	if tournament.Status == StatusCompleted || tournament.Status == StatusCancelled {
		return nil, nil, fmt.Errorf("cannot update bye settings for %s tournament", tournament.Status)
	}

	tournament.ByeSettings = settings
	tournament.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tournament); err != nil {
		return nil, nil, fmt.Errorf("failed to update tournament: %w", err)
	}

	// Check for FIDE violations and return warnings
	var violations []FIDEViolation
	if settings != nil {
		violations = settings.ValidateFIDECompliance(tournament.Rounds)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("violations", len(violations)).
		Msg("Bye settings updated")

	return tournament, violations, nil
}

// GetByeSettings returns the bye settings for a tournament (with defaults if not set)
func (s *Service) GetByeSettings(ctx context.Context, tournamentID uuid.UUID) (*ByeSettings, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if tournament.ByeSettings != nil {
		return tournament.ByeSettings, nil
	}

	// Return default settings
	return DefaultByeSettings(), nil
}

// UpdateScoringSettings updates the scoring settings for a tournament
func (s *Service) UpdateScoringSettings(ctx context.Context, tournamentID uuid.UUID, settings *ScoringSettings) (*Tournament, []FIDEViolation, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, nil, err
	}

	// Only allow updates before tournament starts (ongoing or later)
	if tournament.Status == StatusOngoing || tournament.Status == StatusCompleted || tournament.Status == StatusCancelled {
		return nil, nil, fmt.Errorf("cannot update scoring settings for %s tournament", tournament.Status)
	}

	// Validate scoring settings
	if settings != nil {
		if errs := settings.Validate(); len(errs) > 0 {
			return nil, nil, fmt.Errorf("invalid scoring settings: %s", errs[0].Message)
		}
	}

	tournament.ScoringSettings = settings
	tournament.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tournament); err != nil {
		return nil, nil, fmt.Errorf("failed to update tournament: %w", err)
	}

	// Check for FIDE violations and return warnings
	var violations []FIDEViolation
	if settings != nil {
		violations = settings.ValidateFIDECompliance()
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Interface("scoring_settings", settings).
		Int("fide_violations", len(violations)).
		Msg("Scoring settings updated")

	return tournament, violations, nil
}

// GetScoringSettings returns the scoring settings for a tournament (with defaults if not set)
func (s *Service) GetScoringSettings(ctx context.Context, tournamentID uuid.UUID) (*ScoringSettings, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if tournament.ScoringSettings != nil {
		return tournament.ScoringSettings, nil
	}

	// Return default settings
	return DefaultScoringSettings(), nil
}

// RequestHalfPointBye requests a half-point bye for a participant in a specific round
func (s *Service) RequestHalfPointBye(ctx context.Context, tournamentID, participantID uuid.UUID, roundNumber int) error {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	participant, err := s.repo.GetParticipant(ctx, participantID)
	if err != nil {
		return err
	}

	if participant.TournamentID != tournamentID {
		return fmt.Errorf("participant not in this tournament")
	}

	settings := tournament.ByeSettings
	if settings == nil {
		settings = DefaultByeSettings()
	}

	// Check if half-point byes are allowed
	if !settings.HalfPointBye.Allowed {
		return fmt.Errorf("half-point byes are not allowed in this tournament")
	}

	// Validate round number is within valid range
	if roundNumber < 1 {
		return fmt.Errorf("invalid round number: must be at least 1")
	}
	if roundNumber > tournament.Rounds {
		return fmt.Errorf("invalid round number: tournament only has %d rounds", tournament.Rounds)
	}

	// Check maximum per player limit
	if settings.HalfPointBye.MaxPerPlayer > 0 && participant.HalfPointByesUsed >= settings.HalfPointBye.MaxPerPlayer {
		return fmt.Errorf("player has reached maximum half-point byes (%d)", settings.HalfPointBye.MaxPerPlayer)
	}

	// Check if round is within allowed range based on AllowedRounds setting
	if settings.HalfPointBye.AllowedRounds == "first_half" {
		halfRounds := tournament.Rounds / 2
		if roundNumber > halfRounds {
			return fmt.Errorf("half-point byes only allowed in first half (rounds 1-%d)", halfRounds)
		}
	} else if settings.HalfPointBye.AllowedRounds == "specific" {
		allowed := false
		for _, r := range settings.HalfPointBye.SpecificRounds {
			if r == roundNumber {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("half-point bye not allowed in round %d", roundNumber)
		}
	} else if settings.HalfPointBye.AllowedRounds == "none" {
		return fmt.Errorf("half-point byes are not allowed in any round")
	}

	// Check if already requested for this round
	for _, r := range participant.ByeRequests {
		if r == roundNumber {
			return fmt.Errorf("bye already requested for round %d", roundNumber)
		}
	}

	// Add to bye requests
	participant.ByeRequests = append(participant.ByeRequests, roundNumber)

	if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("participant_id", participantID.String()).
		Int("round", roundNumber).
		Msg("Half-point bye requested")

	return nil
}

// CancelByeRequest cancels a bye request for a participant
func (s *Service) CancelByeRequest(ctx context.Context, tournamentID, participantID uuid.UUID, roundNumber int) error {
	participant, err := s.repo.GetParticipant(ctx, participantID)
	if err != nil {
		return err
	}

	if participant.TournamentID != tournamentID {
		return fmt.Errorf("participant not in this tournament")
	}

	// Check if request exists and find its index
	foundIndex := -1
	for i, r := range participant.ByeRequests {
		if r == roundNumber {
			foundIndex = i
			break
		}
	}
	if foundIndex == -1 {
		return fmt.Errorf("no bye request found for round %d", roundNumber)
	}

	// Check if round is already paired
	round, err := s.repo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err == nil && round != nil && round.Status != RoundStatusPending {
		return fmt.Errorf("cannot cancel bye request - round %d is already paired", roundNumber)
	}

	// Remove from slice
	participant.ByeRequests = append(participant.ByeRequests[:foundIndex], participant.ByeRequests[foundIndex+1:]...)

	if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("participant_id", participantID.String()).
		Int("round", roundNumber).
		Msg("Bye request cancelled")

	return nil
}

// RecordSystemBye records that a participant received a system bye
func (s *Service) RecordSystemBye(ctx context.Context, tournamentID, participantID uuid.UUID, roundNumber int) error {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	participant, err := s.repo.GetParticipant(ctx, participantID)
	if err != nil {
		return err
	}

	if participant.TournamentID != tournamentID {
		return fmt.Errorf("participant not in this tournament")
	}

	settings := tournament.ByeSettings
	if settings == nil {
		settings = DefaultByeSettings()
	}

	// Check if player already received system bye (FIDE: only once per tournament)
	// MaxPerPlayer of 1 means only one system bye allowed
	if participant.SystemByeRound != nil && settings.SystemBye.MaxPerPlayer == 1 {
		return fmt.Errorf("player already received system bye in round %d", *participant.SystemByeRound)
	}

	participant.SystemByeRound = &roundNumber

	if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("participant_id", participantID.String()).
		Int("round", roundNumber).
		Msg("System bye recorded")

	return nil
}

// RecordForfeit records a forfeit for a participant and checks auto-withdrawal threshold
func (s *Service) RecordForfeit(ctx context.Context, tournamentID, participantID uuid.UUID, isConsecutive bool) (*Participant, bool, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, false, err
	}

	participant, err := s.repo.GetParticipant(ctx, participantID)
	if err != nil {
		return nil, false, err
	}

	if participant.TournamentID != tournamentID {
		return nil, false, fmt.Errorf("participant not in this tournament")
	}

	settings := tournament.ByeSettings
	if settings == nil {
		settings = DefaultByeSettings()
	}

	participant.ForfeitCount++
	if isConsecutive {
		participant.ConsecutiveForfeits++
	} else {
		participant.ConsecutiveForfeits = 1
	}

	autoWithdrawn := false

	// Check auto-withdrawal thresholds
	// AutoWithdrawAfter of 0 means disabled
	if settings.Forfeit.AutoWithdrawAfter > 0 {
		if settings.Forfeit.ConsecutiveOnly {
			// Only count consecutive forfeits
			if participant.ConsecutiveForfeits >= settings.Forfeit.AutoWithdrawAfter {
				autoWithdrawn = true
			}
		} else {
			// Count total forfeits
			if participant.ForfeitCount >= settings.Forfeit.AutoWithdrawAfter {
				autoWithdrawn = true
			}
		}
	}

	if autoWithdrawn {
		currentRound := tournament.CurrentRound
		participant.WithdrawnRound = &currentRound
		var reason string
		if settings.Forfeit.ConsecutiveOnly {
			reason = fmt.Sprintf("Auto-withdrawn after %d consecutive forfeits", participant.ConsecutiveForfeits)
		} else {
			reason = fmt.Sprintf("Auto-withdrawn after %d total forfeits", participant.ForfeitCount)
		}
		participant.WithdrawalReason = &reason
		participant.IsActive = false
		participant.RegistrationStatus = "withdrawn"

		s.logger.Info().
			Str("tournament_id", tournamentID.String()).
			Str("participant_id", participantID.String()).
			Int("forfeit_count", participant.ForfeitCount).
			Int("consecutive_forfeits", participant.ConsecutiveForfeits).
			Msg("Participant auto-withdrawn due to forfeits")
	}

	if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
		return nil, false, fmt.Errorf("failed to update participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("participant_id", participantID.String()).
		Int("forfeit_count", participant.ForfeitCount).
		Int("consecutive", participant.ConsecutiveForfeits).
		Msg("Forfeit recorded")

	return participant, autoWithdrawn, nil
}

// WithdrawParticipantWithDetails withdraws a participant with reason and round tracking
func (s *Service) WithdrawParticipantWithDetails(ctx context.Context, tournamentID, participantID uuid.UUID, reason string) error {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	participant, err := s.repo.GetParticipant(ctx, participantID)
	if err != nil {
		return err
	}

	if participant.TournamentID != tournamentID {
		return fmt.Errorf("participant not in this tournament")
	}

	currentRound := tournament.CurrentRound
	if currentRound == 0 {
		currentRound = 1 // Before first round
	}

	participant.WithdrawnRound = &currentRound
	if reason != "" {
		participant.WithdrawalReason = &reason
	}
	participant.IsActive = false
	participant.RegistrationStatus = "withdrawn"

	if err := s.repo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("participant_id", participantID.String()).
		Int("round", currentRound).
		Str("reason", reason).
		Msg("Participant withdrawn with details")

	return nil
}

// AddLateEntry adds a late entry participant with proper missed round handling
func (s *Service) AddLateEntry(ctx context.Context, tournamentID uuid.UUID, req *RegisterParticipantRequest, entryRound int) (*Participant, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Tournament must be ongoing for late entry
	if tournament.Status != StatusOngoing {
		return nil, fmt.Errorf("late entry only allowed for ongoing tournaments")
	}

	settings := tournament.ByeSettings
	if settings == nil {
		settings = DefaultByeSettings()
	}

	// Check if late entry is allowed
	if !settings.LateEntry.Allowed {
		return nil, fmt.Errorf("late entry is not allowed in this tournament")
	}

	// Validate entry round is within valid range
	if entryRound < 1 {
		return nil, fmt.Errorf("invalid entry round: must be at least 1")
	}
	if entryRound > tournament.Rounds {
		return nil, fmt.Errorf("invalid entry round: tournament only has %d rounds", tournament.Rounds)
	}

	// Check maximum entry round
	if settings.LateEntry.MaxRound > 0 && entryRound > settings.LateEntry.MaxRound {
		return nil, fmt.Errorf("late entry not allowed after round %d", settings.LateEntry.MaxRound)
	}

	// Check if player already registered
	existing, _ := s.repo.GetParticipantByPlayerID(ctx, tournamentID, req.PlayerID)
	if existing != nil {
		return nil, fmt.Errorf("player already registered")
	}

	// Auto-fetch player rating if not provided
	startingRating := req.StartingRating
	if startingRating == 0 {
		ratingType := s.getRatingType(tournament.TimeControlType)
		fetchedRating, err := s.repo.GetPlayerRating(ctx, req.PlayerID, ratingType)
		if err == nil && fetchedRating > 0 {
			startingRating = fetchedRating
		}
	}

	// Check rating limits for late entry
	if settings.LateEntry.MinRating != nil && startingRating < *settings.LateEntry.MinRating {
		return nil, fmt.Errorf("player rating %d is below minimum required rating %d", startingRating, *settings.LateEntry.MinRating)
	}
	if settings.LateEntry.MaxRating != nil && startingRating > *settings.LateEntry.MaxRating {
		return nil, fmt.Errorf("player rating %d exceeds maximum allowed rating %d", startingRating, *settings.LateEntry.MaxRating)
	}

	// Calculate initial points for missed rounds
	missedRounds := entryRound - 1
	initialPoints := float64(missedRounds) * settings.LateEntry.MissedRoundPoints

	participant := &Participant{
		ID:                 uuid.New(),
		TournamentID:       tournamentID,
		SectionID:          req.SectionID,
		PlayerID:           req.PlayerID,
		RegistrationStatus: "registered",
		RegisteredAt:       time.Now(),
		StartingRating:     startingRating,
		IsActive:           true,
		PaymentStatus:      "pending",
		LateEntryRound:     &entryRound,
		Points:             initialPoints,
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to add late entry: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("player_id", req.PlayerID.String()).
		Int("entry_round", entryRound).
		Int("missed_rounds", missedRounds).
		Float64("initial_points", initialPoints).
		Msg("Late entry added")

	return participant, nil
}

// GetFIDEViolations returns any FIDE rule violations for the current tournament settings
func (s *Service) GetFIDEViolations(ctx context.Context, tournamentID uuid.UUID) ([]FIDEViolation, error) {
	tournament, err := s.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	var violations []FIDEViolation

	// Check bye settings violations
	if tournament.ByeSettings != nil {
		violations = append(violations, tournament.ByeSettings.ValidateFIDECompliance(tournament.Rounds)...)
	}

	// Check scoring settings violations
	if tournament.ScoringSettings != nil {
		violations = append(violations, tournament.ScoringSettings.ValidateFIDECompliance()...)
	}

	return violations, nil
}

// GetParticipantByeStatus returns detailed bye/withdrawal status for a participant
func (s *Service) GetParticipantByeStatus(ctx context.Context, participantID uuid.UUID) (*ParticipantByeStatus, error) {
	participant, err := s.repo.GetParticipant(ctx, participantID)
	if err != nil {
		return nil, err
	}

	tournament, err := s.repo.GetByID(ctx, participant.TournamentID)
	if err != nil {
		return nil, err
	}

	settings := tournament.ByeSettings
	if settings == nil {
		settings = DefaultByeSettings()
	}

	status := &ParticipantByeStatus{
		ParticipantID:        participant.ID,
		HalfPointByesUsed:    participant.HalfPointByesUsed,
		ZeroPointByesUsed:    participant.ZeroPointByesUsed,
		SystemByeRound:       participant.SystemByeRound,
		ForfeitCount:         participant.ForfeitCount,
		ConsecutiveForfeits:  participant.ConsecutiveForfeits,
		LateEntryRound:       participant.LateEntryRound,
		WithdrawnRound:       participant.WithdrawnRound,
		WithdrawalReason:     participant.WithdrawalReason,
		PendingByeRequests:   participant.ByeRequests,
		MaxHalfPointByesLeft: settings.HalfPointBye.MaxPerPlayer - participant.HalfPointByesUsed,
		CanRequestBye:        settings.HalfPointBye.Allowed && (settings.HalfPointBye.MaxPerPlayer == 0 || participant.HalfPointByesUsed < settings.HalfPointBye.MaxPerPlayer),
	}

	return status, nil
}

// ParticipantByeStatus contains detailed bye/withdrawal status for a participant
type ParticipantByeStatus struct {
	ParticipantID        uuid.UUID `json:"participantId"`
	HalfPointByesUsed    int       `json:"halfPointByesUsed"`
	ZeroPointByesUsed    int       `json:"zeroPointByesUsed"`
	SystemByeRound       *int      `json:"systemByeRound,omitempty"`
	ForfeitCount         int       `json:"forfeitCount"`
	ConsecutiveForfeits  int       `json:"consecutiveForfeits"`
	LateEntryRound       *int      `json:"lateEntryRound,omitempty"`
	WithdrawnRound       *int      `json:"withdrawnRound,omitempty"`
	WithdrawalReason     *string   `json:"withdrawalReason,omitempty"`
	PendingByeRequests   []int     `json:"pendingByeRequests,omitempty"` // Rounds where bye requested
	MaxHalfPointByesLeft int       `json:"maxHalfPointByesLeft"`
	CanRequestBye        bool      `json:"canRequestBye"`
}
