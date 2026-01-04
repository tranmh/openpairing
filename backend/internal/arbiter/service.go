package arbiter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/tournament"
	"github.com/rs/zerolog"
)

// ArbiterAssignment represents an arbiter assigned to a tournament
type ArbiterAssignment struct {
	TournamentID uuid.UUID `json:"tournamentId" db:"tournament_id"`
	UserID       uuid.UUID `json:"userId" db:"user_id"`
	Role         string    `json:"role" db:"role"` // "chief", "assistant"
}

// ArbiterRepository provides access to arbiter assignments
type ArbiterRepository interface {
	GetArbiterAssignment(ctx context.Context, tournamentID, userID uuid.UUID) (*ArbiterAssignment, error)
	IsOrganizationMember(ctx context.Context, organizationID, userID uuid.UUID) (bool, error)
}

// Service provides arbiter-specific business logic
type Service struct {
	tournamentRepo tournament.Repository
	arbiterRepo    ArbiterRepository
	logger         zerolog.Logger
}

// NewService creates a new arbiter service
func NewService(tournamentRepo tournament.Repository, arbiterRepo ArbiterRepository, logger zerolog.Logger) *Service {
	return &Service{
		tournamentRepo: tournamentRepo,
		arbiterRepo:    arbiterRepo,
		logger:         logger.With().Str("service", "arbiter").Logger(),
	}
}

// canManageTournament checks if the arbiter has permission to manage the tournament
func (s *Service) canManageTournament(ctx context.Context, tournamentID, arbiterID uuid.UUID) error {
	tourney, err := s.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return fmt.Errorf("tournament not found: %w", err)
	}

	// Check if arbiter has a specific assignment for this tournament
	if s.arbiterRepo != nil {
		assignment, err := s.arbiterRepo.GetArbiterAssignment(ctx, tournamentID, arbiterID)
		if err == nil && assignment != nil {
			return nil // Arbiter is assigned to this tournament
		}

		// Check if arbiter is a member of the tournament's organization
		if tourney.OrganizationID != nil {
			isMember, err := s.arbiterRepo.IsOrganizationMember(ctx, *tourney.OrganizationID, arbiterID)
			if err == nil && isMember {
				return nil // Arbiter is a member of the organization
			}
		}
	}

	// If no arbiter repo or checks failed, deny access
	// In a production system, you might want to allow site admins here
	return fmt.Errorf("arbiter not authorized for this tournament")
}

// AuditAction represents an auditable arbiter action
type AuditAction struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TournamentID uuid.UUID  `json:"tournamentId" db:"tournament_id"`
	ArbiterID    uuid.UUID  `json:"arbiterId" db:"arbiter_id"`
	Action       string     `json:"action" db:"action"`
	EntityType   string     `json:"entityType" db:"entity_type"`
	EntityID     uuid.UUID  `json:"entityId" db:"entity_id"`
	OldValue     string     `json:"oldValue" db:"old_value"`
	NewValue     string     `json:"newValue" db:"new_value"`
	Reason       string     `json:"reason" db:"reason"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
}

// TournamentStatus contains current tournament status for the arbiter console
type TournamentStatus struct {
	Tournament     *tournament.Tournament        `json:"tournament"`
	CurrentRound   *tournament.Round             `json:"currentRound"`
	Rounds         []tournament.Round            `json:"rounds"`
	Participants   []tournament.Participant      `json:"participants"`
	ActiveGames    []GameStatus                  `json:"activeGames"`
	PendingResults int                           `json:"pendingResults"`
	Statistics     TournamentStatistics          `json:"statistics"`
}

// GameStatus represents a game's current status for monitoring
type GameStatus struct {
	Game           *tournament.Game   `json:"game"`
	WhitePlayer    *PlayerInfo        `json:"whitePlayer"`
	BlackPlayer    *PlayerInfo        `json:"blackPlayer"`
	ElapsedMinutes int                `json:"elapsedMinutes"`
}

// PlayerInfo contains player information for display
type PlayerInfo struct {
	ParticipantID uuid.UUID `json:"participantId"`
	Name          string    `json:"name"`
	Rating        int       `json:"rating"`
	Points        float64   `json:"points"`
}

// TournamentStatistics contains statistics for the arbiter console
type TournamentStatistics struct {
	TotalParticipants  int     `json:"totalParticipants"`
	ActiveParticipants int     `json:"activeParticipants"`
	WithdrawnCount     int     `json:"withdrawnCount"`
	CompletedRounds    int     `json:"completedRounds"`
	TotalGames         int     `json:"totalGames"`
	CompletedGames     int     `json:"completedGames"`
	PendingGames       int     `json:"pendingGames"`
	AverageRating      float64 `json:"averageRating"`
}

// GetTournamentStatus gets the current tournament status for the arbiter console
func (s *Service) GetTournamentStatus(ctx context.Context, tournamentID uuid.UUID, arbiterID uuid.UUID) (*TournamentStatus, error) {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, tournamentID, arbiterID); err != nil {
		return nil, err
	}

	tourney, err := s.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("tournament not found: %w", err)
	}

	participants, err := s.tournamentRepo.ListParticipants(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	rounds, err := s.tournamentRepo.ListRounds(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Find current round
	var currentRound *tournament.Round
	for i := range rounds {
		if rounds[i].Status == tournament.RoundStatusOngoing || rounds[i].Status == tournament.RoundStatusPaired {
			currentRound = &rounds[i]
			break
		}
	}

	// Calculate statistics
	stats := s.calculateStatistics(participants, rounds)

	// Get active games if round is ongoing
	var activeGames []GameStatus
	var pendingResults int

	if currentRound != nil {
		games, err := s.tournamentRepo.ListGamesByRound(ctx, currentRound.ID)
		if err == nil {
			participantMap := make(map[uuid.UUID]*tournament.Participant)
			for i := range participants {
				participantMap[participants[i].ID] = &participants[i]
			}

			for i := range games {
				if games[i].Status == tournament.GameStatusScheduled || games[i].Status == tournament.GameStatusOngoing {
					gs := GameStatus{
						Game: &games[i],
					}

					if games[i].WhiteParticipantID != nil {
						if p, ok := participantMap[*games[i].WhiteParticipantID]; ok {
							gs.WhitePlayer = &PlayerInfo{
								ParticipantID: p.ID,
								Rating:        p.StartingRating,
								Points:        p.Points,
							}
						}
					}

					if games[i].BlackParticipantID != nil {
						if p, ok := participantMap[*games[i].BlackParticipantID]; ok {
							gs.BlackPlayer = &PlayerInfo{
								ParticipantID: p.ID,
								Rating:        p.StartingRating,
								Points:        p.Points,
							}
						}
					}

					if currentRound.ActualStart != nil {
						gs.ElapsedMinutes = int(time.Since(*currentRound.ActualStart).Minutes())
					}

					activeGames = append(activeGames, gs)

					if games[i].Result == nil || *games[i].Result == "" {
						pendingResults++
					}
				}
			}
		}
	}

	return &TournamentStatus{
		Tournament:     tourney,
		CurrentRound:   currentRound,
		Rounds:         rounds,
		Participants:   participants,
		ActiveGames:    activeGames,
		PendingResults: pendingResults,
		Statistics:     stats,
	}, nil
}

func (s *Service) calculateStatistics(participants []tournament.Participant, rounds []tournament.Round) TournamentStatistics {
	stats := TournamentStatistics{
		TotalParticipants: len(participants),
	}

	var totalRating int
	for _, p := range participants {
		if p.IsActive {
			stats.ActiveParticipants++
			totalRating += p.StartingRating
		} else {
			stats.WithdrawnCount++
		}
	}

	if stats.ActiveParticipants > 0 {
		stats.AverageRating = float64(totalRating) / float64(stats.ActiveParticipants)
	}

	for _, r := range rounds {
		if r.Status == tournament.RoundStatusCompleted {
			stats.CompletedRounds++
		}
	}

	return stats
}

// WithdrawPlayerRequest contains data for withdrawing a player
type WithdrawPlayerRequest struct {
	TournamentID  uuid.UUID `json:"tournamentId"`
	ParticipantID uuid.UUID `json:"participantId"`
	Reason        string    `json:"reason"`
	EffectiveFrom int       `json:"effectiveFrom"` // Round number, 0 = immediate
}

// WithdrawPlayer withdraws a player from the tournament
func (s *Service) WithdrawPlayer(ctx context.Context, req *WithdrawPlayerRequest, arbiterID uuid.UUID) error {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, req.TournamentID, arbiterID); err != nil {
		return err
	}

	participant, err := s.tournamentRepo.GetParticipant(ctx, req.ParticipantID)
	if err != nil {
		return fmt.Errorf("participant not found: %w", err)
	}

	if participant.TournamentID != req.TournamentID {
		return fmt.Errorf("participant not in this tournament")
	}

	if !participant.IsActive {
		return fmt.Errorf("participant already withdrawn")
	}

	// Check if there's an ongoing game
	rounds, err := s.tournamentRepo.ListRounds(ctx, req.TournamentID)
	if err != nil {
		return err
	}

	// Find current round
	for _, r := range rounds {
		if r.Status == tournament.RoundStatusOngoing {
			// Check if player has an ongoing game
			games, err := s.tournamentRepo.ListGamesByRound(ctx, r.ID)
			if err != nil {
				continue
			}

			for _, g := range games {
				if (g.WhiteParticipantID != nil && *g.WhiteParticipantID == req.ParticipantID) ||
					(g.BlackParticipantID != nil && *g.BlackParticipantID == req.ParticipantID) {
					if g.Result == nil || *g.Result == "" {
						// Enter forfeit result
						if g.WhiteParticipantID != nil && *g.WhiteParticipantID == req.ParticipantID {
							result := "0-1F"
							g.Result = &result
							g.WhiteScore = floatPtr(0)
							g.BlackScore = floatPtr(1)
						} else {
							result := "1-0F"
							g.Result = &result
							g.WhiteScore = floatPtr(1)
							g.BlackScore = floatPtr(0)
						}
						g.Status = tournament.GameStatusCompleted
						now := time.Now()
						g.ResultEnteredAt = &now
						g.ResultEnteredBy = &arbiterID

						if err := s.tournamentRepo.UpdateGame(ctx, &g); err != nil {
							return fmt.Errorf("failed to update game: %w", err)
						}

						s.logger.Info().
							Str("game_id", g.ID.String()).
							Str("result", *g.Result).
							Msg("Forfeit result entered for withdrawn player")
					}
				}
			}
		}
	}

	// Update participant status
	participant.IsActive = false
	participant.RegistrationStatus = "withdrawn"

	if err := s.tournamentRepo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Str("participant_id", req.ParticipantID.String()).
		Str("reason", req.Reason).
		Str("arbiter_id", arbiterID.String()).
		Msg("Player withdrawn by arbiter")

	// Log audit action
	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		ArbiterID:    arbiterID,
		Action:       "withdraw_player",
		EntityType:   "participant",
		EntityID:     req.ParticipantID,
		OldValue:     "active",
		NewValue:     "withdrawn",
		Reason:       req.Reason,
		CreatedAt:    time.Now(),
	})

	return nil
}

func floatPtr(f float64) *float64 {
	return &f
}

// AssignByeRequest contains data for assigning a bye
type AssignByeRequest struct {
	TournamentID  uuid.UUID `json:"tournamentId"`
	ParticipantID uuid.UUID `json:"participantId"`
	RoundNumber   int       `json:"roundNumber"`
	ByeType       string    `json:"byeType"` // "half" (0.5 points), "full" (1 point), "zero" (0 points)
}

// AssignBye assigns a bye to a player
func (s *Service) AssignBye(ctx context.Context, req *AssignByeRequest, arbiterID uuid.UUID) error {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, req.TournamentID, arbiterID); err != nil {
		return err
	}

	participant, err := s.tournamentRepo.GetParticipant(ctx, req.ParticipantID)
	if err != nil {
		return fmt.Errorf("participant not found: %w", err)
	}

	round, err := s.tournamentRepo.GetRoundByNumber(ctx, req.TournamentID, req.RoundNumber)
	if err != nil {
		return fmt.Errorf("round not found: %w", err)
	}

	if round.Status != tournament.RoundStatusPending && round.Status != tournament.RoundStatusPaired {
		return fmt.Errorf("cannot assign bye: round is %s", round.Status)
	}

	// Determine score based on bye type
	var score float64
	var resultCode string
	switch req.ByeType {
	case "full":
		score = 1.0
		resultCode = "F" // Forfeit win / Full point bye
	case "zero":
		score = 0.0
		resultCode = "Z" // Zero point bye
	default: // half
		score = 0.5
		resultCode = "H" // Half point bye
	}

	// Create bye game
	byeResult := "BYE"
	game := tournament.Game{
		ID:                 uuid.New(),
		TournamentID:       req.TournamentID,
		RoundID:            round.ID,
		BoardNumber:        0, // Byes are typically board 0 or last
		WhiteParticipantID: &req.ParticipantID,
		BlackParticipantID: nil,
		Result:             &byeResult,
		WhiteScore:         &score,
		BlackScore:         floatPtr(0),
		Status:             tournament.GameStatusCompleted,
		ResultEnteredBy:    &arbiterID,
		CreatedAt:          time.Now(),
	}

	now := time.Now()
	game.ResultEnteredAt = &now

	if err := s.tournamentRepo.CreateGames(ctx, []tournament.Game{game}); err != nil {
		return fmt.Errorf("failed to create bye: %w", err)
	}

	// Update participant points
	participant.Points += score
	if err := s.tournamentRepo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant points: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Str("participant_id", req.ParticipantID.String()).
		Int("round", req.RoundNumber).
		Str("bye_type", req.ByeType).
		Float64("score", score).
		Msg("Bye assigned by arbiter")

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		ArbiterID:    arbiterID,
		Action:       "assign_bye",
		EntityType:   "game",
		EntityID:     game.ID,
		NewValue:     fmt.Sprintf("%s (%s)", resultCode, req.ByeType),
		CreatedAt:    time.Now(),
	})

	return nil
}

// CorrectResultRequest contains data for correcting a game result
type CorrectResultRequest struct {
	TournamentID uuid.UUID `json:"tournamentId"`
	GameID       uuid.UUID `json:"gameId"`
	NewResult    string    `json:"newResult"`
	Reason       string    `json:"reason"`
}

// CorrectResult corrects a game result
func (s *Service) CorrectResult(ctx context.Context, req *CorrectResultRequest, arbiterID uuid.UUID) error {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, req.TournamentID, arbiterID); err != nil {
		return err
	}

	game, err := s.tournamentRepo.GetGame(ctx, req.GameID)
	if err != nil {
		return fmt.Errorf("game not found: %w", err)
	}

	if game.TournamentID != req.TournamentID {
		return fmt.Errorf("game not in this tournament")
	}

	oldResult := ""
	if game.Result != nil {
		oldResult = *game.Result
	}
	oldWhiteScore := game.WhiteScore
	oldBlackScore := game.BlackScore

	// Parse new result
	whiteScore, blackScore, err := parseResult(req.NewResult)
	if err != nil {
		return err
	}

	// Update game
	game.Result = &req.NewResult
	game.WhiteScore = &whiteScore
	game.BlackScore = &blackScore
	now := time.Now()
	game.ResultEnteredAt = &now
	game.ResultEnteredBy = &arbiterID

	if err := s.tournamentRepo.UpdateGame(ctx, game); err != nil {
		return fmt.Errorf("failed to update game: %w", err)
	}

	// Update participant scores
	if game.WhiteParticipantID != nil {
		white, err := s.tournamentRepo.GetParticipant(ctx, *game.WhiteParticipantID)
		if err == nil {
			// Reverse old score
			if oldWhiteScore != nil {
				white.Points -= *oldWhiteScore
			}
			// Add new score
			white.Points += whiteScore
			s.tournamentRepo.UpdateParticipant(ctx, white)
		}
	}

	if game.BlackParticipantID != nil {
		black, err := s.tournamentRepo.GetParticipant(ctx, *game.BlackParticipantID)
		if err == nil {
			// Reverse old score
			if oldBlackScore != nil {
				black.Points -= *oldBlackScore
			}
			// Add new score
			black.Points += blackScore
			s.tournamentRepo.UpdateParticipant(ctx, black)
		}
	}

	s.logger.Info().
		Str("game_id", req.GameID.String()).
		Str("old_result", oldResult).
		Str("new_result", req.NewResult).
		Str("reason", req.Reason).
		Str("arbiter_id", arbiterID.String()).
		Msg("Result corrected by arbiter")

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		ArbiterID:    arbiterID,
		Action:       "correct_result",
		EntityType:   "game",
		EntityID:     req.GameID,
		OldValue:     oldResult,
		NewValue:     req.NewResult,
		Reason:       req.Reason,
		CreatedAt:    time.Now(),
	})

	return nil
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

// SwapPairingRequest contains data for swapping players in pairings
type SwapPairingRequest struct {
	TournamentID uuid.UUID `json:"tournamentId"`
	RoundNumber  int       `json:"roundNumber"`
	Game1ID      uuid.UUID `json:"game1Id"`
	Player1ID    uuid.UUID `json:"player1Id"` // Player to swap from game1
	Game2ID      uuid.UUID `json:"game2Id"`
	Player2ID    uuid.UUID `json:"player2Id"` // Player to swap from game2
	Reason       string    `json:"reason"`
}

// SwapPairing swaps two players between pairings
func (s *Service) SwapPairing(ctx context.Context, req *SwapPairingRequest, arbiterID uuid.UUID) error {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, req.TournamentID, arbiterID); err != nil {
		return err
	}

	round, err := s.tournamentRepo.GetRoundByNumber(ctx, req.TournamentID, req.RoundNumber)
	if err != nil {
		return fmt.Errorf("round not found: %w", err)
	}

	if round.Status != tournament.RoundStatusPaired {
		return fmt.Errorf("can only swap pairings in paired rounds")
	}

	game1, err := s.tournamentRepo.GetGame(ctx, req.Game1ID)
	if err != nil {
		return fmt.Errorf("game 1 not found: %w", err)
	}

	game2, err := s.tournamentRepo.GetGame(ctx, req.Game2ID)
	if err != nil {
		return fmt.Errorf("game 2 not found: %w", err)
	}

	// Determine which position each player is in and swap
	var player1IsWhite, player2IsWhite bool

	if game1.WhiteParticipantID != nil && *game1.WhiteParticipantID == req.Player1ID {
		player1IsWhite = true
	} else if game1.BlackParticipantID != nil && *game1.BlackParticipantID == req.Player1ID {
		player1IsWhite = false
	} else {
		return fmt.Errorf("player 1 not found in game 1")
	}

	if game2.WhiteParticipantID != nil && *game2.WhiteParticipantID == req.Player2ID {
		player2IsWhite = true
	} else if game2.BlackParticipantID != nil && *game2.BlackParticipantID == req.Player2ID {
		player2IsWhite = false
	} else {
		return fmt.Errorf("player 2 not found in game 2")
	}

	// Perform swap
	if player1IsWhite {
		game1.WhiteParticipantID = &req.Player2ID
	} else {
		game1.BlackParticipantID = &req.Player2ID
	}

	if player2IsWhite {
		game2.WhiteParticipantID = &req.Player1ID
	} else {
		game2.BlackParticipantID = &req.Player1ID
	}

	if err := s.tournamentRepo.UpdateGame(ctx, game1); err != nil {
		return fmt.Errorf("failed to update game 1: %w", err)
	}

	if err := s.tournamentRepo.UpdateGame(ctx, game2); err != nil {
		return fmt.Errorf("failed to update game 2: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Int("round", req.RoundNumber).
		Str("player1", req.Player1ID.String()).
		Str("player2", req.Player2ID.String()).
		Str("reason", req.Reason).
		Msg("Pairing swapped by arbiter")

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		ArbiterID:    arbiterID,
		Action:       "swap_pairing",
		EntityType:   "round",
		EntityID:     round.ID,
		OldValue:     fmt.Sprintf("%s vs %s", req.Player1ID, req.Player2ID),
		NewValue:     fmt.Sprintf("swapped"),
		Reason:       req.Reason,
		CreatedAt:    time.Now(),
	})

	return nil
}

// ChangeColorRequest contains data for swapping colors in a game
type ChangeColorRequest struct {
	TournamentID uuid.UUID `json:"tournamentId"`
	GameID       uuid.UUID `json:"gameId"`
	Reason       string    `json:"reason"`
}

// ChangeColor swaps the colors in a game
func (s *Service) ChangeColor(ctx context.Context, req *ChangeColorRequest, arbiterID uuid.UUID) error {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, req.TournamentID, arbiterID); err != nil {
		return err
	}

	game, err := s.tournamentRepo.GetGame(ctx, req.GameID)
	if err != nil {
		return fmt.Errorf("game not found: %w", err)
	}

	if game.TournamentID != req.TournamentID {
		return fmt.Errorf("game not in this tournament")
	}

	if game.Result != nil && *game.Result != "" {
		return fmt.Errorf("cannot change colors after result is entered")
	}

	// Swap colors
	game.WhiteParticipantID, game.BlackParticipantID = game.BlackParticipantID, game.WhiteParticipantID

	if err := s.tournamentRepo.UpdateGame(ctx, game); err != nil {
		return fmt.Errorf("failed to update game: %w", err)
	}

	s.logger.Info().
		Str("game_id", req.GameID.String()).
		Str("reason", req.Reason).
		Msg("Colors swapped by arbiter")

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		ArbiterID:    arbiterID,
		Action:       "change_color",
		EntityType:   "game",
		EntityID:     req.GameID,
		Reason:       req.Reason,
		CreatedAt:    time.Now(),
	})

	return nil
}

// GetAuditLog retrieves the audit log for a tournament
func (s *Service) GetAuditLog(ctx context.Context, tournamentID uuid.UUID, arbiterID uuid.UUID, limit int) ([]AuditAction, error) {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, tournamentID, arbiterID); err != nil {
		return nil, err
	}

	// This would be implemented with a proper audit log repository
	// For now, return empty slice
	return []AuditAction{}, nil
}

func (s *Service) logAudit(ctx context.Context, action *AuditAction) {
	// This would be implemented to persist audit actions to database
	s.logger.Debug().
		Str("action", action.Action).
		Str("entity_type", action.EntityType).
		Str("entity_id", action.EntityID.String()).
		Str("arbiter_id", action.ArbiterID.String()).
		Msg("Audit log entry")
}

// Announcement represents a tournament announcement
type Announcement struct {
	ID           uuid.UUID `json:"id"`
	TournamentID uuid.UUID `json:"tournamentId"`
	Title        string    `json:"title"`
	Message      string    `json:"message"`
	Priority     string    `json:"priority"` // info, warning, urgent
	CreatedBy    uuid.UUID `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
}

// CreateAnnouncementRequest contains data for creating an announcement
type CreateAnnouncementRequest struct {
	TournamentID uuid.UUID `json:"tournamentId"`
	Title        string    `json:"title"`
	Message      string    `json:"message"`
	Priority     string    `json:"priority"`
}

// CreateAnnouncement creates a tournament announcement
func (s *Service) CreateAnnouncement(ctx context.Context, req *CreateAnnouncementRequest, arbiterID uuid.UUID) (*Announcement, error) {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, req.TournamentID, arbiterID); err != nil {
		return nil, err
	}

	announcement := &Announcement{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		Title:        req.Title,
		Message:      req.Message,
		Priority:     req.Priority,
		CreatedBy:    arbiterID,
		CreatedAt:    time.Now(),
	}

	// This would be persisted to database
	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Str("title", req.Title).
		Str("priority", req.Priority).
		Msg("Announcement created")

	return announcement, nil
}
