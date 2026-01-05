package result

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service provides result management functionality
type Service struct {
	repo *Repository
}

// NewService creates a new result service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// EnterResult enters a result for a game
func (s *Service) EnterResult(ctx context.Context, entry *ResultEntry) (*ResultEntryResponse, error) {
	// Validate the result
	validation := s.ValidateResult(ctx, entry)
	if !validation.Valid {
		return nil, fmt.Errorf("invalid result: %v", validation.Errors)
	}

	// Get the game
	game, err := s.repo.GetGame(ctx, entry.GameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Check if result already exists
	if game.Result != nil && *game.Result != "" {
		return nil, fmt.Errorf("game already has a result, use correction workflow to change it")
	}

	// Parse the result into scores
	whiteScore, blackScore := ParseResult(entry.Result)

	// Update the game result
	err = s.repo.UpdateResult(ctx, entry.GameID, string(entry.Result), whiteScore, blackScore, entry.EnteredBy)
	if err != nil {
		return nil, fmt.Errorf("failed to update result: %w", err)
	}

	// Update participant statistics
	if game.WhiteParticipantID != nil {
		err = s.updateParticipantStats(ctx, *game.WhiteParticipantID, whiteScore)
		if err != nil {
			return nil, fmt.Errorf("failed to update white participant stats: %w", err)
		}
	}
	if game.BlackParticipantID != nil {
		err = s.updateParticipantStats(ctx, *game.BlackParticipantID, blackScore)
		if err != nil {
			return nil, fmt.Errorf("failed to update black participant stats: %w", err)
		}
	}

	// Update round completion count
	err = s.repo.UpdateRoundCompletedCount(ctx, game.RoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to update round count: %w", err)
	}

	// Create audit log
	newResult := string(entry.Result)
	audit := &ResultAudit{
		GameID:        entry.GameID,
		Action:        "entered",
		NewResult:     &newResult,
		NewWhiteScore: &whiteScore,
		NewBlackScore: &blackScore,
		Reason:        entry.Notes,
		PerformedBy:   entry.EnteredBy,
	}
	err = s.repo.CreateAuditLog(ctx, audit)
	if err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to create audit log: %v\n", err)
	}

	return &ResultEntryResponse{
		GameID:          entry.GameID,
		Result:          string(entry.Result),
		WhiteScore:      whiteScore,
		BlackScore:      blackScore,
		StandingsUpdate: true,
	}, nil
}

// CorrectResult corrects an existing result
func (s *Service) CorrectResult(ctx context.Context, correction *ResultCorrection) (*ResultEntryResponse, error) {
	// Get the current game state
	game, err := s.repo.GetGame(ctx, correction.GameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Verify the old result matches
	if game.Result == nil || *game.Result != string(correction.OldResult) {
		return nil, fmt.Errorf("current result does not match expected old result")
	}

	// Store old values for audit
	oldResult := *game.Result
	oldWhiteScore := game.WhiteScore
	oldBlackScore := game.BlackScore

	// Reverse the old result from participant stats
	if game.WhiteParticipantID != nil && oldWhiteScore != nil {
		err = s.reverseParticipantStats(ctx, *game.WhiteParticipantID, *oldWhiteScore)
		if err != nil {
			return nil, fmt.Errorf("failed to reverse white participant stats: %w", err)
		}
	}
	if game.BlackParticipantID != nil && oldBlackScore != nil {
		err = s.reverseParticipantStats(ctx, *game.BlackParticipantID, *oldBlackScore)
		if err != nil {
			return nil, fmt.Errorf("failed to reverse black participant stats: %w", err)
		}
	}

	// Parse the new result
	newWhiteScore, newBlackScore := ParseResult(correction.NewResult)

	// Update the game result
	err = s.repo.UpdateResult(ctx, correction.GameID, string(correction.NewResult), newWhiteScore, newBlackScore, correction.CorrectedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to update result: %w", err)
	}

	// Apply new result to participant stats
	if game.WhiteParticipantID != nil {
		err = s.updateParticipantStats(ctx, *game.WhiteParticipantID, newWhiteScore)
		if err != nil {
			return nil, fmt.Errorf("failed to update white participant stats: %w", err)
		}
	}
	if game.BlackParticipantID != nil {
		err = s.updateParticipantStats(ctx, *game.BlackParticipantID, newBlackScore)
		if err != nil {
			return nil, fmt.Errorf("failed to update black participant stats: %w", err)
		}
	}

	// Create audit log
	newResult := string(correction.NewResult)
	audit := &ResultAudit{
		GameID:        correction.GameID,
		Action:        "corrected",
		OldResult:     &oldResult,
		OldWhiteScore: oldWhiteScore,
		OldBlackScore: oldBlackScore,
		NewResult:     &newResult,
		NewWhiteScore: &newWhiteScore,
		NewBlackScore: &newBlackScore,
		Reason:        &correction.Reason,
		PerformedBy:   correction.CorrectedBy,
	}
	err = s.repo.CreateAuditLog(ctx, audit)
	if err != nil {
		fmt.Printf("warning: failed to create audit log: %v\n", err)
	}

	return &ResultEntryResponse{
		GameID:          correction.GameID,
		Result:          string(correction.NewResult),
		WhiteScore:      newWhiteScore,
		BlackScore:      newBlackScore,
		StandingsUpdate: true,
	}, nil
}

// DeleteResult removes a result from a game
func (s *Service) DeleteResult(ctx context.Context, gameID uuid.UUID, deletedBy uuid.UUID, reason string) error {
	// Get the current game state
	game, err := s.repo.GetGame(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game: %w", err)
	}

	if game.Result == nil {
		return fmt.Errorf("game has no result to delete")
	}

	// Store old values for audit
	oldResult := *game.Result
	oldWhiteScore := game.WhiteScore
	oldBlackScore := game.BlackScore

	// Reverse the result from participant stats
	if game.WhiteParticipantID != nil && oldWhiteScore != nil {
		err = s.reverseParticipantStats(ctx, *game.WhiteParticipantID, *oldWhiteScore)
		if err != nil {
			return fmt.Errorf("failed to reverse white participant stats: %w", err)
		}
	}
	if game.BlackParticipantID != nil && oldBlackScore != nil {
		err = s.reverseParticipantStats(ctx, *game.BlackParticipantID, *oldBlackScore)
		if err != nil {
			return fmt.Errorf("failed to reverse black participant stats: %w", err)
		}
	}

	// Clear the result
	err = s.repo.ClearResult(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to clear result: %w", err)
	}

	// Update round completion count
	err = s.repo.UpdateRoundCompletedCount(ctx, game.RoundID)
	if err != nil {
		return fmt.Errorf("failed to update round count: %w", err)
	}

	// Create audit log
	audit := &ResultAudit{
		GameID:        gameID,
		Action:        "deleted",
		OldResult:     &oldResult,
		OldWhiteScore: oldWhiteScore,
		OldBlackScore: oldBlackScore,
		Reason:        &reason,
		PerformedBy:   deletedBy,
	}
	err = s.repo.CreateAuditLog(ctx, audit)
	if err != nil {
		fmt.Printf("warning: failed to create audit log: %v\n", err)
	}

	return nil
}

// ValidateResult validates a result entry
func (s *Service) ValidateResult(ctx context.Context, entry *ResultEntry) *ResultValidation {
	validation := &ResultValidation{Valid: true}

	// Check if game exists
	game, err := s.repo.GetGame(ctx, entry.GameID)
	if err != nil {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "game not found")
		return validation
	}

	// Check if result is valid
	if !IsValidResult(string(entry.Result)) {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "invalid result value")
	}

	// Warn if game already has a result
	if game.Result != nil && *game.Result != "" {
		validation.Warnings = append(validation.Warnings, "game already has a result")
	}

	// Check for bye result with two players
	if entry.Result == ResultBye {
		if game.WhiteParticipantID != nil && game.BlackParticipantID != nil {
			validation.Valid = false
			validation.Errors = append(validation.Errors, "cannot enter bye result for game with two players")
		}
	}

	// Check for forfeit results
	if entry.Result == ResultWhiteForfeit || entry.Result == ResultBlackForfeit {
		validation.Warnings = append(validation.Warnings, "forfeit result entered")
	}

	return validation
}

// GetGame retrieves a game with full details
func (s *Service) GetGame(ctx context.Context, gameID uuid.UUID) (*Game, error) {
	return s.repo.GetGameWithPlayers(ctx, gameID)
}

// GetRoundGames retrieves all games for a round
func (s *Service) GetRoundGames(ctx context.Context, tournamentID uuid.UUID, roundNumber int) ([]Game, error) {
	return s.repo.GetRoundGames(ctx, tournamentID, roundNumber)
}

// GetRoundStatus retrieves the status of a round
func (s *Service) GetRoundStatus(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*RoundStatus, error) {
	return s.repo.GetRoundStatus(ctx, tournamentID, roundNumber)
}

// GetResultAuditLog retrieves the audit log for a game
func (s *Service) GetResultAuditLog(ctx context.Context, gameID uuid.UUID) ([]ResultAudit, error) {
	return s.repo.GetAuditLog(ctx, gameID)
}

// EnterBulkResults enters multiple results at once
func (s *Service) EnterBulkResults(ctx context.Context, bulk *BulkResultEntry) ([]ResultEntryResponse, []error) {
	var responses []ResultEntryResponse
	var errors []error

	for _, entry := range bulk.Results {
		resp, err := s.EnterResult(ctx, &entry)
		if err != nil {
			errors = append(errors, fmt.Errorf("game %s: %w", entry.GameID, err))
		} else {
			responses = append(responses, *resp)
		}
	}

	return responses, errors
}

// RecalculateAllStats recalculates all participant statistics from games
func (s *Service) RecalculateAllStats(ctx context.Context, tournamentID uuid.UUID) error {
	// This would need to iterate through all participants
	// For now, return not implemented
	return fmt.Errorf("not implemented - use standings service to recalculate")
}

// Helper functions

func (s *Service) updateParticipantStats(ctx context.Context, participantID uuid.UUID, score float64) error {
	var winsDelta, drawsDelta, lossesDelta int
	pointsDelta := int(score * 10) // Convert to tenths for integer arithmetic

	switch score {
	case 1.0:
		winsDelta = 1
	case 0.5:
		drawsDelta = 1
	case 0.0:
		lossesDelta = 1
	}

	return s.repo.UpdateParticipantStats(ctx, participantID, pointsDelta, winsDelta, drawsDelta, lossesDelta)
}

func (s *Service) reverseParticipantStats(ctx context.Context, participantID uuid.UUID, score float64) error {
	var winsDelta, drawsDelta, lossesDelta int
	pointsDelta := -int(score * 10) // Negative to reverse

	switch score {
	case 1.0:
		winsDelta = -1
	case 0.5:
		drawsDelta = -1
	case 0.0:
		lossesDelta = -1
	}

	return s.repo.UpdateParticipantStats(ctx, participantID, pointsDelta, winsDelta, drawsDelta, lossesDelta)
}
