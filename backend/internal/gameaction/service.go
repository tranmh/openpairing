package gameaction

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles game action business logic
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new game action service
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("service", "gameaction").Logger(),
	}
}

// Resign handles a player resigning from a game
func (s *Service) Resign(ctx context.Context, gameID, userID uuid.UUID) (*GameActionResponse, error) {
	// Check if user is a player in this game
	isPlayer, color, err := s.repo.IsPlayerInGame(ctx, gameID, userID)
	if err != nil {
		return nil, err
	}
	if !isPlayer {
		return nil, fmt.Errorf("you are not a player in this game")
	}

	// Get game to check status
	game, err := s.repo.GetGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if game.Status != StatusOngoing {
		return nil, fmt.Errorf("game is not ongoing")
	}

	// Determine result based on who resigned
	var result string
	if color == "white" {
		result = "0-1" // Black wins
	} else {
		result = "1-0" // White wins
	}

	// Update the game
	if err := s.repo.UpdateGameResult(ctx, gameID, result, TermResignation); err != nil {
		s.logger.Error().Err(err).Str("game_id", gameID.String()).Msg("Failed to update game result")
		return nil, err
	}

	// Clear any pending draw offers
	s.repo.ClearDrawOffer(gameID)

	s.logger.Info().
		Str("game_id", gameID.String()).
		Str("resigned_by", userID.String()).
		Str("color", color).
		Msg("Player resigned")

	status := StatusCompleted
	return &GameActionResponse{
		Success:   true,
		Message:   fmt.Sprintf("%s resigned, game over", color),
		NewStatus: &status,
		Result:    &result,
	}, nil
}

// OfferDraw handles a draw offer or acceptance
func (s *Service) OfferDraw(ctx context.Context, gameID, userID uuid.UUID, accept *bool) (*GameActionResponse, error) {
	// Check if user is a player
	isPlayer, color, err := s.repo.IsPlayerInGame(ctx, gameID, userID)
	if err != nil {
		return nil, err
	}
	if !isPlayer {
		return nil, fmt.Errorf("you are not a player in this game")
	}

	// Get game to check status
	game, err := s.repo.GetGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if game.Status != StatusOngoing {
		return nil, fmt.Errorf("game is not ongoing")
	}

	// Check for existing draw offer
	existingOffer := s.repo.GetDrawOffer(gameID)

	if accept != nil {
		// Responding to an offer
		if existingOffer == nil {
			return nil, fmt.Errorf("no draw offer to respond to")
		}
		if existingOffer.OfferedBy == userID {
			return nil, fmt.Errorf("you cannot respond to your own draw offer")
		}

		if *accept {
			// Accept the draw
			result := "1/2-1/2"
			if err := s.repo.UpdateGameResult(ctx, gameID, result, TermDrawAgreement); err != nil {
				s.logger.Error().Err(err).Str("game_id", gameID.String()).Msg("Failed to update game for draw")
				return nil, err
			}

			s.repo.ClearDrawOffer(gameID)

			s.logger.Info().
				Str("game_id", gameID.String()).
				Str("accepted_by", userID.String()).
				Msg("Draw accepted")

			status := StatusCompleted
			return &GameActionResponse{
				Success:   true,
				Message:   "Draw accepted",
				NewStatus: &status,
				Result:    &result,
			}, nil
		} else {
			// Decline the draw
			s.repo.ClearDrawOffer(gameID)

			s.logger.Info().
				Str("game_id", gameID.String()).
				Str("declined_by", userID.String()).
				Msg("Draw declined")

			return &GameActionResponse{
				Success: true,
				Message: "Draw offer declined",
			}, nil
		}
	}

	// Making a new offer
	if existingOffer != nil && existingOffer.OfferedBy == userID {
		return nil, fmt.Errorf("you already have a pending draw offer")
	}

	s.repo.SetDrawOffer(gameID, userID)

	s.logger.Info().
		Str("game_id", gameID.String()).
		Str("offered_by", userID.String()).
		Str("color", color).
		Msg("Draw offered")

	return &GameActionResponse{
		Success: true,
		Message: fmt.Sprintf("Draw offered by %s", color),
	}, nil
}

// AbortGame handles aborting a game
func (s *Service) AbortGame(ctx context.Context, gameID, userID uuid.UUID, reason string) (*GameActionResponse, error) {
	// Check if user is a player
	isPlayer, _, err := s.repo.IsPlayerInGame(ctx, gameID, userID)
	if err != nil {
		return nil, err
	}
	if !isPlayer {
		return nil, fmt.Errorf("you are not a player in this game")
	}

	// Get game to check status
	game, err := s.repo.GetGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	// Can only abort games that haven't progressed much
	if game.Status == StatusCompleted || game.Status == StatusAborted {
		return nil, fmt.Errorf("game is already finished")
	}

	// Allow abort only in first few moves (typically first 2 full moves)
	if game.MoveCount > 4 && game.Status == StatusOngoing {
		return nil, fmt.Errorf("game has progressed too far to abort, you may resign instead")
	}

	// Abort the game
	if err := s.repo.AbortGame(ctx, gameID); err != nil {
		s.logger.Error().Err(err).Str("game_id", gameID.String()).Msg("Failed to abort game")
		return nil, err
	}

	s.repo.ClearDrawOffer(gameID)

	s.logger.Info().
		Str("game_id", gameID.String()).
		Str("aborted_by", userID.String()).
		Str("reason", reason).
		Msg("Game aborted")

	status := StatusAborted
	return &GameActionResponse{
		Success:   true,
		Message:   "Game aborted",
		NewStatus: &status,
	}, nil
}

// CallArbiter handles requesting arbiter intervention
func (s *Service) CallArbiter(ctx context.Context, gameID, userID uuid.UUID, reason string) (*GameActionResponse, error) {
	// Check if user is a player
	isPlayer, color, err := s.repo.IsPlayerInGame(ctx, gameID, userID)
	if err != nil {
		return nil, err
	}
	if !isPlayer {
		return nil, fmt.Errorf("you are not a player in this game")
	}

	// Get game to check status
	game, err := s.repo.GetGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if game.Status != StatusOngoing && game.Status != StatusWaiting {
		return nil, fmt.Errorf("cannot call arbiter for a finished game")
	}

	// Check if there's already a pending request
	existing, err := s.repo.GetActiveArbiterRequest(ctx, gameID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to check existing arbiter request")
	}
	if existing != nil {
		return nil, fmt.Errorf("there is already a pending arbiter request for this game")
	}

	// Create arbiter request
	request, err := s.repo.CreateArbiterRequest(ctx, gameID, userID, reason)
	if err != nil {
		s.logger.Error().Err(err).Str("game_id", gameID.String()).Msg("Failed to create arbiter request")
		return nil, fmt.Errorf("failed to submit arbiter request")
	}

	s.logger.Info().
		Str("game_id", gameID.String()).
		Str("requested_by", userID.String()).
		Str("color", color).
		Str("reason", reason).
		Msg("Arbiter requested")

	return &GameActionResponse{
		Success: true,
		Message: fmt.Sprintf("Arbiter request submitted (ID: %s)", request.ID.String()),
	}, nil
}

// GetGame retrieves a game with player information
func (s *Service) GetGame(ctx context.Context, gameID uuid.UUID) (*GameWithPlayers, error) {
	return s.repo.GetGameWithPlayers(ctx, gameID)
}

// GetDrawOffer checks if there's a pending draw offer
func (s *Service) GetDrawOffer(ctx context.Context, gameID uuid.UUID) *DrawOffer {
	return s.repo.GetDrawOffer(gameID)
}

// GetUserActiveGames gets a user's active games
func (s *Service) GetUserActiveGames(ctx context.Context, userID uuid.UUID) ([]ChessGame, error) {
	return s.repo.GetUserActiveGames(ctx, userID)
}
