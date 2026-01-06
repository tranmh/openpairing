package arbiter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/player"
	"github.com/openpairing/openpairing/internal/tournament"
	"github.com/rs/zerolog"
)

// ArbiterAssignment represents an arbiter assigned to a tournament
type ArbiterAssignment struct {
	TournamentID uuid.UUID `json:"tournamentId" db:"tournament_id"`
	UserID       uuid.UUID `json:"userId" db:"user_id"`
	Role         string    `json:"role" db:"role"` // "chief", "assistant"
}

// Service provides arbiter-specific business logic
type Service struct {
	tournamentRepo tournament.Repository
	playerRepo     player.Repository
	repo           *Repository
	logger         zerolog.Logger
}

// NewService creates a new arbiter service
func NewService(tournamentRepo tournament.Repository, playerRepo player.Repository, repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		tournamentRepo: tournamentRepo,
		playerRepo:     playerRepo,
		repo:           repo,
		logger:         logger.With().Str("service", "arbiter").Logger(),
	}
}

// canManageTournament checks if the arbiter has permission to manage the tournament
func (s *Service) canManageTournament(ctx context.Context, tournamentID, arbiterID uuid.UUID) error {
	return s.CanManageTournamentWithRole(ctx, tournamentID, arbiterID, "")
}

// CanManageTournamentWithRole checks if the user has permission to manage the tournament
// considering their role
func (s *Service) CanManageTournamentWithRole(ctx context.Context, tournamentID, arbiterID uuid.UUID, userRole string) error {
	// Users with organizer or admin role can manage any tournament
	if userRole == "organizer" || userRole == "admin" {
		// Verify the tournament exists
		_, err := s.tournamentRepo.GetByID(ctx, tournamentID)
		if err != nil {
			return fmt.Errorf("tournament not found: %w", err)
		}
		return nil
	}

	tourney, err := s.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return fmt.Errorf("tournament not found: %w", err)
	}

	// Check if arbiter has a specific assignment for this tournament
	if s.repo != nil {
		assignment, err := s.repo.GetArbiterAssignment(ctx, tournamentID, arbiterID)
		if err == nil && assignment != nil {
			return nil // Arbiter is assigned to this tournament
		}

		// Check if arbiter is a member of the tournament's organization
		if tourney.OrganizationID != nil {
			isMember, err := s.repo.IsOrganizationMember(ctx, *tourney.OrganizationID, arbiterID)
			if err == nil && isMember {
				return nil // Arbiter is a member of the organization
			}
		}
	}

	// If no repo or checks failed, deny access
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
	return s.GetTournamentStatusWithRole(ctx, tournamentID, arbiterID, "")
}

// GetTournamentStatusWithRole gets the current tournament status with role-based authorization
func (s *Service) GetTournamentStatusWithRole(ctx context.Context, tournamentID uuid.UUID, arbiterID uuid.UUID, userRole string) (*TournamentStatus, error) {
	// Verify user has permission for this tournament (considering role)
	if err := s.CanManageTournamentWithRole(ctx, tournamentID, arbiterID, userRole); err != nil {
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

	// Build player name map from participants
	playerNameMap := make(map[uuid.UUID]string)
	if s.playerRepo != nil {
		// Collect unique player IDs
		playerIDs := make([]uuid.UUID, 0, len(participants))
		for i := range participants {
			playerIDs = append(playerIDs, participants[i].PlayerID)
		}

		// Fetch players
		if len(playerIDs) > 0 {
			players, err := s.playerRepo.GetByIDs(ctx, playerIDs)
			if err == nil {
				for i := range players {
					playerNameMap[players[i].ID] = players[i].FirstName + " " + players[i].LastName
				}
			}
		}
	}

	// Build participant -> player name map
	participantNameMap := make(map[uuid.UUID]string)
	for i := range participants {
		if name, ok := playerNameMap[participants[i].PlayerID]; ok {
			participantNameMap[participants[i].ID] = name
		}
	}

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
								Name:          participantNameMap[p.ID],
								Rating:        p.StartingRating,
								Points:        p.Points,
							}
						}
					}

					if games[i].BlackParticipantID != nil {
						if p, ok := participantMap[*games[i].BlackParticipantID]; ok {
							gs.BlackPlayer = &PlayerInfo{
								ParticipantID: p.ID,
								Name:          participantNameMap[p.ID],
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

// ReactivatePlayerRequest contains data for reactivating a withdrawn player
type ReactivatePlayerRequest struct {
	TournamentID      uuid.UUID `json:"tournamentId"`
	ParticipantID     uuid.UUID `json:"participantId"`
	MissedRoundPoints float64   `json:"missedRoundPoints"` // Points per missed round (0 = keep existing score)
	Reason            string    `json:"reason"`
}

// ReactivatePlayer reactivates a withdrawn player
func (s *Service) ReactivatePlayer(ctx context.Context, req *ReactivatePlayerRequest, arbiterID uuid.UUID) error {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, req.TournamentID, arbiterID); err != nil {
		return err
	}

	// Get tournament to check settings
	tourney, err := s.tournamentRepo.GetByID(ctx, req.TournamentID)
	if err != nil {
		return fmt.Errorf("tournament not found: %w", err)
	}

	// Check if re-entry is allowed
	if tourney.ByeSettings != nil && !tourney.ByeSettings.Withdrawal.ReentryAllowed {
		return fmt.Errorf("re-entry is not allowed for this tournament")
	}

	// Get participant
	participant, err := s.tournamentRepo.GetParticipant(ctx, req.ParticipantID)
	if err != nil {
		return fmt.Errorf("participant not found: %w", err)
	}

	if participant.TournamentID != req.TournamentID {
		return fmt.Errorf("participant not in this tournament")
	}

	if participant.IsActive {
		return fmt.Errorf("participant is already active")
	}

	// Get current round
	rounds, err := s.tournamentRepo.ListRounds(ctx, req.TournamentID)
	if err != nil {
		return fmt.Errorf("failed to get rounds: %w", err)
	}

	currentRound := 0
	for _, r := range rounds {
		if r.RoundNumber > currentRound {
			currentRound = r.RoundNumber
		}
	}

	// Check max rounds missed (if configured)
	if tourney.ByeSettings != nil && tourney.ByeSettings.Withdrawal.ReentryMaxRoundsMissed > 0 {
		withdrawnRound := 0
		if participant.WithdrawnRound != nil {
			withdrawnRound = *participant.WithdrawnRound
		}
		roundsMissed := currentRound - withdrawnRound
		if roundsMissed > tourney.ByeSettings.Withdrawal.ReentryMaxRoundsMissed {
			return fmt.Errorf("player has missed too many rounds (%d) - maximum allowed is %d",
				roundsMissed, tourney.ByeSettings.Withdrawal.ReentryMaxRoundsMissed)
		}
	}

	// Calculate points for missed rounds if requested
	if req.MissedRoundPoints > 0 && participant.WithdrawnRound != nil {
		roundsMissed := currentRound - *participant.WithdrawnRound
		if roundsMissed > 0 {
			participant.Points += req.MissedRoundPoints * float64(roundsMissed)
		}
	}

	// Reactivate the participant
	participant.IsActive = true
	participant.RegistrationStatus = "registered"
	// Keep WithdrawnRound and WithdrawalReason for audit history

	if err := s.tournamentRepo.UpdateParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Str("participant_id", req.ParticipantID.String()).
		Str("reason", req.Reason).
		Str("arbiter_id", arbiterID.String()).
		Msg("Player reactivated by arbiter")

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		ArbiterID:    arbiterID,
		Action:       "reactivate_player",
		EntityType:   "participant",
		EntityID:     req.ParticipantID,
		OldValue:     "withdrawn",
		NewValue:     "active",
		Reason:       req.Reason,
		CreatedAt:    time.Now(),
	})

	return nil
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

// DeletePairingRequest contains data for deleting a pairing
type DeletePairingRequest struct {
	TournamentID uuid.UUID `json:"tournamentId"`
	GameID       uuid.UUID `json:"gameId"`
	Reason       string    `json:"reason"`
}

// DeletePairing deletes a pairing from a round
func (s *Service) DeletePairing(ctx context.Context, req *DeletePairingRequest, arbiterID uuid.UUID) error {
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

	// Get round to verify status
	round, err := s.tournamentRepo.GetRound(ctx, game.RoundID)
	if err != nil {
		return fmt.Errorf("round not found: %w", err)
	}

	// Only allow deleting pairings in paired rounds (not ongoing or completed)
	if round.Status != tournament.RoundStatusPaired && round.Status != tournament.RoundStatusPending {
		return fmt.Errorf("cannot delete pairing: round is %s", round.Status)
	}

	// If it was a bye game, reverse the points (bye games have result "BYE")
	isByeGame := game.BlackParticipantID == nil && game.WhiteParticipantID != nil
	if isByeGame && game.WhiteScore != nil {
		white, err := s.tournamentRepo.GetParticipant(ctx, *game.WhiteParticipantID)
		if err == nil && white != nil {
			white.Points -= *game.WhiteScore
			s.tournamentRepo.UpdateParticipant(ctx, white)
		}
	} else if game.Result != nil && *game.Result != "" {
		// For non-bye games, don't allow deletion if result is already entered
		return fmt.Errorf("cannot delete pairing with result already entered")
	}

	// Delete the game
	if err := s.tournamentRepo.DeleteGame(ctx, req.GameID); err != nil {
		return fmt.Errorf("failed to delete game: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Str("game_id", req.GameID.String()).
		Str("reason", req.Reason).
		Str("arbiter_id", arbiterID.String()).
		Msg("Pairing deleted by arbiter")

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		ArbiterID:    arbiterID,
		Action:       "delete_pairing",
		EntityType:   "game",
		EntityID:     req.GameID,
		Reason:       req.Reason,
		CreatedAt:    time.Now(),
	})

	return nil
}

// CreateManualPairingRequest contains data for creating a manual pairing
type CreateManualPairingRequest struct {
	TournamentID       uuid.UUID  `json:"tournamentId"`
	RoundNumber        int        `json:"roundNumber"`
	WhiteParticipantID uuid.UUID  `json:"whiteParticipantId"`
	BlackParticipantID *uuid.UUID `json:"blackParticipantId"` // nil for bye
	BoardNumber        *int       `json:"boardNumber"`        // auto-assign if nil
}

// ManualPairingWarning represents a warning about a manual pairing
type ManualPairingWarning struct {
	Type    string `json:"type"`    // "forbidden_pair", "previous_encounter"
	Message string `json:"message"`
}

// CreateManualPairingResponse contains the result of creating a manual pairing
type CreateManualPairingResponse struct {
	GameID   uuid.UUID              `json:"gameId"`
	Warnings []ManualPairingWarning `json:"warnings"`
}

// CreateManualPairing creates a manual pairing in a round
func (s *Service) CreateManualPairing(ctx context.Context, req *CreateManualPairingRequest, arbiterID uuid.UUID) (*CreateManualPairingResponse, error) {
	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, req.TournamentID, arbiterID); err != nil {
		return nil, err
	}

	// Get tournament for scoring settings
	tourney, err := s.tournamentRepo.GetByID(ctx, req.TournamentID)
	if err != nil {
		return nil, fmt.Errorf("tournament not found: %w", err)
	}

	// Get round
	round, err := s.tournamentRepo.GetRoundByNumber(ctx, req.TournamentID, req.RoundNumber)
	if err != nil {
		return nil, fmt.Errorf("round not found: %w", err)
	}

	// Only allow creating pairings in pending or paired rounds
	if round.Status != tournament.RoundStatusPaired && round.Status != tournament.RoundStatusPending {
		return nil, fmt.Errorf("cannot create pairing: round is %s", round.Status)
	}

	// Verify white participant exists
	white, err := s.tournamentRepo.GetParticipant(ctx, req.WhiteParticipantID)
	if err != nil {
		return nil, fmt.Errorf("white participant not found: %w", err)
	}
	if white.TournamentID != req.TournamentID {
		return nil, fmt.Errorf("white participant not in this tournament")
	}

	var warnings []ManualPairingWarning

	// If not a bye, verify black participant and check for warnings
	if req.BlackParticipantID != nil {
		black, err := s.tournamentRepo.GetParticipant(ctx, *req.BlackParticipantID)
		if err != nil {
			return nil, fmt.Errorf("black participant not found: %w", err)
		}
		if black.TournamentID != req.TournamentID {
			return nil, fmt.Errorf("black participant not in this tournament")
		}

		// Check for forbidden pairs
		forbiddenPairs, err := s.tournamentRepo.ListForbiddenPairs(ctx, req.TournamentID)
		if err == nil {
			for _, fp := range forbiddenPairs {
				if (fp.Player1ID == req.WhiteParticipantID && fp.Player2ID == *req.BlackParticipantID) ||
					(fp.Player1ID == *req.BlackParticipantID && fp.Player2ID == req.WhiteParticipantID) {
					warnings = append(warnings, ManualPairingWarning{
						Type:    "forbidden_pair",
						Message: fmt.Sprintf("These players are in a forbidden pair (reason: %s)", fp.Reason),
					})
					break
				}
			}
		}

		// Check for previous encounters
		games, err := s.tournamentRepo.GetPlayerGames(ctx, req.TournamentID, req.WhiteParticipantID)
		if err == nil {
			for _, g := range games {
				if (g.WhiteParticipantID != nil && *g.WhiteParticipantID == *req.BlackParticipantID) ||
					(g.BlackParticipantID != nil && *g.BlackParticipantID == *req.BlackParticipantID) {
					warnings = append(warnings, ManualPairingWarning{
						Type:    "previous_encounter",
						Message: "These players have already played each other in this tournament",
					})
				}
			}
		}
	}

	// Calculate board number if not specified
	boardNumber := 0
	if req.BoardNumber != nil {
		boardNumber = *req.BoardNumber
	} else {
		// Get existing games in the round to find the next board number
		games, err := s.tournamentRepo.ListGamesByRound(ctx, round.ID)
		if err == nil {
			for _, g := range games {
				if g.BoardNumber >= boardNumber {
					boardNumber = g.BoardNumber + 1
				}
			}
		}
		if boardNumber == 0 {
			boardNumber = 1
		}
	}

	// Create the game
	game := tournament.Game{
		ID:                 uuid.New(),
		TournamentID:       req.TournamentID,
		RoundID:            round.ID,
		BoardNumber:        boardNumber,
		WhiteParticipantID: &req.WhiteParticipantID,
		BlackParticipantID: req.BlackParticipantID,
		Status:             tournament.GameStatusScheduled,
		CreatedAt:          time.Now(),
	}

	// If it's a bye, set result immediately
	if req.BlackParticipantID == nil {
		byeResult := "BYE"
		// Use tournament scoring settings for bye points
		scoring := tourney.ScoringSettings
		if scoring == nil {
			scoring = tournament.DefaultScoringSettings()
		}
		byeScore := scoring.WinPoints
		zeroScore := 0.0
		game.Result = &byeResult
		game.WhiteScore = &byeScore
		game.BlackScore = &zeroScore
		game.Status = tournament.GameStatusCompleted
		now := time.Now()
		game.ResultEnteredAt = &now
		game.ResultEnteredBy = &arbiterID

		// Update white participant points
		white.Points += byeScore
		s.tournamentRepo.UpdateParticipant(ctx, white)
	}

	if err := s.tournamentRepo.CreateGames(ctx, []tournament.Game{game}); err != nil {
		return nil, fmt.Errorf("failed to create game: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Str("game_id", game.ID.String()).
		Int("round", req.RoundNumber).
		Int("board", boardNumber).
		Str("arbiter_id", arbiterID.String()).
		Msg("Manual pairing created by arbiter")

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: req.TournamentID,
		ArbiterID:    arbiterID,
		Action:       "create_manual_pairing",
		EntityType:   "game",
		EntityID:     game.ID,
		NewValue:     fmt.Sprintf("board %d", boardNumber),
		CreatedAt:    time.Now(),
	})

	return &CreateManualPairingResponse{
		GameID:   game.ID,
		Warnings: warnings,
	}, nil
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

	if s.repo != nil {
		return s.repo.GetAuditLog(ctx, tournamentID, limit)
	}

	return []AuditAction{}, nil
}

func (s *Service) logAudit(ctx context.Context, action *AuditAction) {
	// Persist audit action to database
	if s.repo != nil {
		if err := s.repo.CreateAuditLog(ctx, action); err != nil {
			s.logger.Error().Err(err).Msg("Failed to persist audit log")
		}
	}

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

	// Persist to database
	if s.repo != nil {
		if err := s.repo.CreateAnnouncement(ctx, announcement); err != nil {
			return nil, fmt.Errorf("failed to create announcement: %w", err)
		}
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Str("title", req.Title).
		Str("priority", req.Priority).
		Msg("Announcement created")

	return announcement, nil
}

// ListAnnouncements lists announcements for a tournament
func (s *Service) ListAnnouncements(ctx context.Context, tournamentID uuid.UUID, activeOnly bool) ([]Announcement, error) {
	if s.repo == nil {
		return []Announcement{}, nil
	}
	return s.repo.ListAnnouncements(ctx, tournamentID, activeOnly)
}

// DeactivateAnnouncement deactivates an announcement
func (s *Service) DeactivateAnnouncement(ctx context.Context, announcementID uuid.UUID, arbiterID uuid.UUID) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}

	announcement, err := s.repo.GetAnnouncement(ctx, announcementID)
	if err != nil || announcement == nil {
		return fmt.Errorf("announcement not found")
	}

	// Verify arbiter has permission for this tournament
	if err := s.canManageTournament(ctx, announcement.TournamentID, arbiterID); err != nil {
		return err
	}

	return s.repo.DeactivateAnnouncement(ctx, announcementID)
}

// AssignArbiter assigns an arbiter to a tournament
func (s *Service) AssignArbiter(ctx context.Context, tournamentID, userID uuid.UUID, role string, assignedBy uuid.UUID) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}

	// Verify the assigner has permission
	if err := s.canManageTournament(ctx, tournamentID, assignedBy); err != nil {
		return err
	}

	assignment := &ArbiterAssignment{
		TournamentID: tournamentID,
		UserID:       userID,
		Role:         role,
	}

	if err := s.repo.AssignArbiter(ctx, assignment, assignedBy); err != nil {
		return fmt.Errorf("failed to assign arbiter: %w", err)
	}

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: tournamentID,
		ArbiterID:    assignedBy,
		Action:       "assign_arbiter",
		EntityType:   "user",
		EntityID:     userID,
		NewValue:     role,
		CreatedAt:    time.Now(),
	})

	return nil
}

// RemoveArbiter removes an arbiter from a tournament
func (s *Service) RemoveArbiter(ctx context.Context, tournamentID, userID, removedBy uuid.UUID) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}

	// Verify the remover has permission
	if err := s.canManageTournament(ctx, tournamentID, removedBy); err != nil {
		return err
	}

	if err := s.repo.RemoveArbiter(ctx, tournamentID, userID); err != nil {
		return fmt.Errorf("failed to remove arbiter: %w", err)
	}

	s.logAudit(ctx, &AuditAction{
		ID:           uuid.New(),
		TournamentID: tournamentID,
		ArbiterID:    removedBy,
		Action:       "remove_arbiter",
		EntityType:   "user",
		EntityID:     userID,
		CreatedAt:    time.Now(),
	})

	return nil
}

// ListTournamentArbiters lists all arbiters for a tournament
func (s *Service) ListTournamentArbiters(ctx context.Context, tournamentID uuid.UUID) ([]ArbiterAssignment, error) {
	if s.repo == nil {
		return []ArbiterAssignment{}, nil
	}
	return s.repo.ListTournamentArbiters(ctx, tournamentID)
}
