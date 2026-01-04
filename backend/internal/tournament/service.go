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
	repo          Repository
	pairingEngine engine.PairingEngine
	logger        zerolog.Logger
}

// NewService creates a new tournament service
func NewService(repo Repository, pairingEngine engine.PairingEngine, logger zerolog.Logger) *Service {
	return &Service{
		repo:          repo,
		pairingEngine: pairingEngine,
		logger:        logger.With().Str("service", "tournament").Logger(),
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

	participant := &Participant{
		ID:                 uuid.New(),
		TournamentID:       tournamentID,
		SectionID:          req.SectionID,
		PlayerID:           req.PlayerID,
		RegistrationStatus: "registered",
		RegisteredAt:       time.Now(),
		StartingRating:     req.StartingRating,
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

// AssignStartRanks assigns start ranks based on rating
func (s *Service) AssignStartRanks(ctx context.Context, tournamentID uuid.UUID) error {
	participants, err := s.GetParticipants(ctx, tournamentID, &ParticipantFilter{
		ActiveOnly: true,
	})
	if err != nil {
		return err
	}

	// Sort by rating descending
	for i := 0; i < len(participants); i++ {
		for j := i + 1; j < len(participants); j++ {
			if participants[j].StartingRating > participants[i].StartingRating {
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

	// Get the round
	round, err := s.repo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err != nil {
		return nil, err
	}

	if round.Status != RoundStatusPending {
		return nil, fmt.Errorf("round is not in pending status")
	}

	// Check if pairing engine is available
	if s.pairingEngine == nil {
		return nil, fmt.Errorf("pairing engine not configured - set JAVAFO_PATH environment variable")
	}

	// Generate TRF for pairing engine
	trfData, err := s.generateTRFForPairing(ctx, tournament, roundNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TRF: %w", err)
	}

	// Call pairing engine
	req := &engine.PairingRequest{
		TRFData:       trfData,
		RoundNumber:   roundNumber,
		PairingSystem: string(tournament.PairingSystem),
	}

	result, err := s.pairingEngine.GeneratePairings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("pairing engine failed: %w", err)
	}

	// Convert pairings to games
	participants, err := s.GetParticipants(ctx, tournamentID, &ParticipantFilter{
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
	for _, game := range games {
		if game.Result != nil && *game.Result == "BYE" {
			if err := s.updateParticipantScores(ctx, &game); err != nil {
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
			Rating:     p.StartingRating,
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

	if game.Status == GameStatusCompleted && game.Result != nil && *game.Result != "" {
		return nil, fmt.Errorf("result already entered")
	}

	// Parse and validate result
	whiteScore, blackScore, err := parseResult(req.Result)
	if err != nil {
		return nil, err
	}

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
	if err := s.updateParticipantScores(ctx, game); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("game_id", gameID.String()).
		Str("result", req.Result).
		Msg("Result entered")

	return game, nil
}

func parseResult(result string) (float64, float64, error) {
	switch result {
	case "1-0", "1-0F":
		return 1, 0, nil
	case "0-1", "0-1F":
		return 0, 1, nil
	case "1/2-1/2":
		return 0.5, 0.5, nil
	case "0-0F":
		return 0, 0, nil
	case "BYE":
		return 0.5, 0, nil
	default:
		return 0, 0, fmt.Errorf("invalid result: %s", result)
	}
}

func (s *Service) updateParticipantScores(ctx context.Context, game *Game) error {
	if game.WhiteParticipantID != nil && game.WhiteScore != nil {
		participant, err := s.repo.GetParticipant(ctx, *game.WhiteParticipantID)
		if err != nil {
			return err
		}
		participant.Points += *game.WhiteScore
		if *game.WhiteScore == 1 {
			participant.Wins++
		} else if *game.WhiteScore == 0.5 {
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
		if *game.BlackScore == 1 {
			participant.Wins++
		} else if *game.BlackScore == 0.5 {
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
			Rating:       p.StartingRating,
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
		player := CrosstablePlayer{
			Rank:           i + 1,
			ParticipantID:  p.ID,
			PlayerID:       p.PlayerID,
			Name:           "", // Would need to join with players table
			Rating:         p.StartingRating,
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
