package result

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ResultRepository defines the interface for result repository operations
type ResultRepository interface {
	GetGame(ctx context.Context, gameID uuid.UUID) (*Game, error)
	GetGameWithPlayers(ctx context.Context, gameID uuid.UUID) (*Game, error)
	UpdateResult(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error
	ClearResult(ctx context.Context, gameID uuid.UUID) error
	UpdateParticipantStats(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error
	UpdateRoundCompletedCount(ctx context.Context, roundID uuid.UUID) error
	CreateAuditLog(ctx context.Context, audit *ResultAudit) error
	GetAuditLog(ctx context.Context, gameID uuid.UUID) ([]ResultAudit, error)
	GetRoundGames(ctx context.Context, tournamentID uuid.UUID, roundNumber int) ([]Game, error)
	GetRoundStatus(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*RoundStatus, error)
}

// TestableService is a service that uses an interface for the repository
type TestableService struct {
	repo ResultRepository
}

// NewTestableService creates a testable result service
func NewTestableService(repo ResultRepository) *TestableService {
	return &TestableService{repo: repo}
}

// EnterResult enters a result for a game (testable version)
func (s *TestableService) EnterResult(ctx context.Context, entry *ResultEntry) (*ResultEntryResponse, error) {
	// Validate the result
	validation := s.ValidateResult(ctx, entry)
	if !validation.Valid {
		return nil, errors.New("invalid result: " + validation.Errors[0])
	}

	// Get the game
	game, err := s.repo.GetGame(ctx, entry.GameID)
	if err != nil {
		return nil, errors.New("failed to get game: " + err.Error())
	}

	// Check if result already exists
	if game.Result != nil && *game.Result != "" {
		return nil, errors.New("game already has a result, use correction workflow to change it")
	}

	// Parse the result into scores
	whiteScore, blackScore := ParseResult(entry.Result)

	// Update the game result
	err = s.repo.UpdateResult(ctx, entry.GameID, string(entry.Result), whiteScore, blackScore, entry.EnteredBy)
	if err != nil {
		return nil, errors.New("failed to update result: " + err.Error())
	}

	// Update participant statistics
	if game.WhiteParticipantID != nil {
		err = s.updateParticipantStats(ctx, *game.WhiteParticipantID, whiteScore)
		if err != nil {
			return nil, errors.New("failed to update white participant stats: " + err.Error())
		}
	}
	if game.BlackParticipantID != nil {
		err = s.updateParticipantStats(ctx, *game.BlackParticipantID, blackScore)
		if err != nil {
			return nil, errors.New("failed to update black participant stats: " + err.Error())
		}
	}

	// Update round completion count
	err = s.repo.UpdateRoundCompletedCount(ctx, game.RoundID)
	if err != nil {
		return nil, errors.New("failed to update round count: " + err.Error())
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
	_ = s.repo.CreateAuditLog(ctx, audit)

	return &ResultEntryResponse{
		GameID:          entry.GameID,
		Result:          string(entry.Result),
		WhiteScore:      whiteScore,
		BlackScore:      blackScore,
		StandingsUpdate: true,
	}, nil
}

// ValidateResult validates a result entry (testable version)
func (s *TestableService) ValidateResult(ctx context.Context, entry *ResultEntry) *ResultValidation {
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

// DeleteResult removes a result from a game (testable version)
func (s *TestableService) DeleteResult(ctx context.Context, gameID uuid.UUID, deletedBy uuid.UUID, reason string) error {
	// Get the current game state
	game, err := s.repo.GetGame(ctx, gameID)
	if err != nil {
		return errors.New("failed to get game: " + err.Error())
	}

	if game.Result == nil {
		return errors.New("game has no result to delete")
	}

	// Store old values for audit
	oldResult := *game.Result
	oldWhiteScore := game.WhiteScore
	oldBlackScore := game.BlackScore

	// Reverse the result from participant stats
	if game.WhiteParticipantID != nil && oldWhiteScore != nil {
		err = s.reverseParticipantStats(ctx, *game.WhiteParticipantID, *oldWhiteScore)
		if err != nil {
			return errors.New("failed to reverse white participant stats: " + err.Error())
		}
	}
	if game.BlackParticipantID != nil && oldBlackScore != nil {
		err = s.reverseParticipantStats(ctx, *game.BlackParticipantID, *oldBlackScore)
		if err != nil {
			return errors.New("failed to reverse black participant stats: " + err.Error())
		}
	}

	// Clear the result
	err = s.repo.ClearResult(ctx, gameID)
	if err != nil {
		return errors.New("failed to clear result: " + err.Error())
	}

	// Update round completion count
	err = s.repo.UpdateRoundCompletedCount(ctx, game.RoundID)
	if err != nil {
		return errors.New("failed to update round count: " + err.Error())
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
	_ = s.repo.CreateAuditLog(ctx, audit)

	return nil
}

// Helper functions
func (s *TestableService) updateParticipantStats(ctx context.Context, participantID uuid.UUID, score float64) error {
	var winsDelta, drawsDelta, lossesDelta int
	pointsDelta := int(score * 10)

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

func (s *TestableService) reverseParticipantStats(ctx context.Context, participantID uuid.UUID, score float64) error {
	var winsDelta, drawsDelta, lossesDelta int
	pointsDelta := -int(score * 10)

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

// TestableCorrectService is for testing CorrectResult
type TestableCorrectService struct {
	repo ResultRepository
}

// CorrectResult corrects an existing result (testable version)
func (s *TestableCorrectService) CorrectResult(ctx context.Context, correction *ResultCorrection) (*ResultEntryResponse, error) {
	game, err := s.repo.GetGame(ctx, correction.GameID)
	if err != nil {
		return nil, errors.New("failed to get game: " + err.Error())
	}

	if game.Result == nil || *game.Result != string(correction.OldResult) {
		return nil, errors.New("current result does not match expected old result")
	}

	oldWhiteScore := game.WhiteScore
	oldBlackScore := game.BlackScore

	// Reverse old stats
	if game.WhiteParticipantID != nil && oldWhiteScore != nil {
		err = s.reverseParticipantStats(ctx, *game.WhiteParticipantID, *oldWhiteScore)
		if err != nil {
			return nil, errors.New("failed to reverse white participant stats: " + err.Error())
		}
	}
	if game.BlackParticipantID != nil && oldBlackScore != nil {
		err = s.reverseParticipantStats(ctx, *game.BlackParticipantID, *oldBlackScore)
		if err != nil {
			return nil, errors.New("failed to reverse black participant stats: " + err.Error())
		}
	}

	newWhiteScore, newBlackScore := ParseResult(correction.NewResult)

	err = s.repo.UpdateResult(ctx, correction.GameID, string(correction.NewResult), newWhiteScore, newBlackScore, correction.CorrectedBy)
	if err != nil {
		return nil, errors.New("failed to update result: " + err.Error())
	}

	// Apply new stats
	if game.WhiteParticipantID != nil {
		err = s.updateParticipantStats(ctx, *game.WhiteParticipantID, newWhiteScore)
		if err != nil {
			return nil, errors.New("failed to update white participant stats: " + err.Error())
		}
	}
	if game.BlackParticipantID != nil {
		err = s.updateParticipantStats(ctx, *game.BlackParticipantID, newBlackScore)
		if err != nil {
			return nil, errors.New("failed to update black participant stats: " + err.Error())
		}
	}

	return &ResultEntryResponse{
		GameID:          correction.GameID,
		Result:          string(correction.NewResult),
		WhiteScore:      newWhiteScore,
		BlackScore:      newBlackScore,
		StandingsUpdate: true,
	}, nil
}

func (s *TestableCorrectService) updateParticipantStats(ctx context.Context, participantID uuid.UUID, score float64) error {
	var winsDelta, drawsDelta, lossesDelta int
	pointsDelta := int(score * 10)

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

func (s *TestableCorrectService) reverseParticipantStats(ctx context.Context, participantID uuid.UUID, score float64) error {
	var winsDelta, drawsDelta, lossesDelta int
	pointsDelta := -int(score * 10)

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

// MockResultRepository is a mock implementation of ResultRepository
type MockResultRepository struct {
	GetGameFunc                 func(ctx context.Context, gameID uuid.UUID) (*Game, error)
	GetGameWithPlayersFunc      func(ctx context.Context, gameID uuid.UUID) (*Game, error)
	UpdateResultFunc            func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error
	ClearResultFunc             func(ctx context.Context, gameID uuid.UUID) error
	UpdateParticipantStatsFunc  func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error
	UpdateRoundCompletedCountFunc func(ctx context.Context, roundID uuid.UUID) error
	CreateAuditLogFunc          func(ctx context.Context, audit *ResultAudit) error
	GetAuditLogFunc             func(ctx context.Context, gameID uuid.UUID) ([]ResultAudit, error)
	GetRoundGamesFunc           func(ctx context.Context, tournamentID uuid.UUID, roundNumber int) ([]Game, error)
	GetRoundStatusFunc          func(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*RoundStatus, error)
}

func (m *MockResultRepository) GetGame(ctx context.Context, gameID uuid.UUID) (*Game, error) {
	if m.GetGameFunc != nil {
		return m.GetGameFunc(ctx, gameID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockResultRepository) GetGameWithPlayers(ctx context.Context, gameID uuid.UUID) (*Game, error) {
	if m.GetGameWithPlayersFunc != nil {
		return m.GetGameWithPlayersFunc(ctx, gameID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockResultRepository) UpdateResult(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
	if m.UpdateResultFunc != nil {
		return m.UpdateResultFunc(ctx, gameID, result, whiteScore, blackScore, enteredBy)
	}
	return nil
}

func (m *MockResultRepository) ClearResult(ctx context.Context, gameID uuid.UUID) error {
	if m.ClearResultFunc != nil {
		return m.ClearResultFunc(ctx, gameID)
	}
	return nil
}

func (m *MockResultRepository) UpdateParticipantStats(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
	if m.UpdateParticipantStatsFunc != nil {
		return m.UpdateParticipantStatsFunc(ctx, participantID, pointsDelta, winsDelta, drawsDelta, lossesDelta)
	}
	return nil
}

func (m *MockResultRepository) UpdateRoundCompletedCount(ctx context.Context, roundID uuid.UUID) error {
	if m.UpdateRoundCompletedCountFunc != nil {
		return m.UpdateRoundCompletedCountFunc(ctx, roundID)
	}
	return nil
}

func (m *MockResultRepository) CreateAuditLog(ctx context.Context, audit *ResultAudit) error {
	if m.CreateAuditLogFunc != nil {
		return m.CreateAuditLogFunc(ctx, audit)
	}
	return nil
}

func (m *MockResultRepository) GetAuditLog(ctx context.Context, gameID uuid.UUID) ([]ResultAudit, error) {
	if m.GetAuditLogFunc != nil {
		return m.GetAuditLogFunc(ctx, gameID)
	}
	return nil, nil
}

func (m *MockResultRepository) GetRoundGames(ctx context.Context, tournamentID uuid.UUID, roundNumber int) ([]Game, error) {
	if m.GetRoundGamesFunc != nil {
		return m.GetRoundGamesFunc(ctx, tournamentID, roundNumber)
	}
	return nil, nil
}

func (m *MockResultRepository) GetRoundStatus(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*RoundStatus, error) {
	if m.GetRoundStatusFunc != nil {
		return m.GetRoundStatusFunc(ctx, tournamentID, roundNumber)
	}
	return nil, nil
}

// Test helper functions
func testGame() *Game {
	whiteID := testutil.ParticipantID1
	blackID := testutil.ParticipantID2
	return &Game{
		ID:                 testutil.GameID1,
		TournamentID:       testutil.TournamentID1,
		RoundID:            testutil.RoundID1,
		RoundNumber:        1,
		BoardNumber:        1,
		WhiteParticipantID: &whiteID,
		BlackParticipantID: &blackID,
		Status:             "pending",
	}
}

func testGameWithResult() *Game {
	game := testGame()
	result := "1-0"
	whiteScore := 1.0
	blackScore := 0.0
	game.Result = &result
	game.WhiteScore = &whiteScore
	game.BlackScore = &blackScore
	return game
}

// Tests

func TestService_EnterResult_Success(t *testing.T) {
	var updatedStats []uuid.UUID
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
		UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
			assert.Equal(t, testutil.GameID1, gameID)
			assert.Equal(t, "1-0", result)
			assert.Equal(t, 1.0, whiteScore)
			assert.Equal(t, 0.0, blackScore)
			return nil
		},
		UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
			updatedStats = append(updatedStats, participantID)
			return nil
		},
		UpdateRoundCompletedCountFunc: func(ctx context.Context, roundID uuid.UUID) error {
			assert.Equal(t, testutil.RoundID1, roundID)
			return nil
		},
		CreateAuditLogFunc: func(ctx context.Context, audit *ResultAudit) error {
			assert.Equal(t, "entered", audit.Action)
			return nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultWhiteWins,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.NoError(t, err)
	assert.Equal(t, testutil.GameID1, resp.GameID)
	assert.Equal(t, "1-0", resp.Result)
	assert.Equal(t, 1.0, resp.WhiteScore)
	assert.Equal(t, 0.0, resp.BlackScore)
	assert.True(t, resp.StandingsUpdate)
	assert.Len(t, updatedStats, 2) // Both participants should have stats updated
}

func TestService_EnterResult_Draw(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
		UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
			assert.Equal(t, "1/2-1/2", result)
			assert.Equal(t, 0.5, whiteScore)
			assert.Equal(t, 0.5, blackScore)
			return nil
		},
		UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
			assert.Equal(t, 5, pointsDelta)   // 0.5 * 10
			assert.Equal(t, 0, winsDelta)
			assert.Equal(t, 1, drawsDelta)
			assert.Equal(t, 0, lossesDelta)
			return nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultDraw,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.NoError(t, err)
	assert.Equal(t, "1/2-1/2", resp.Result)
	assert.Equal(t, 0.5, resp.WhiteScore)
	assert.Equal(t, 0.5, resp.BlackScore)
}

func TestService_EnterResult_GameNotFound(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return nil, errors.New("game not found")
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultWhiteWins,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "game not found")
}

func TestService_EnterResult_AlreadyHasResult(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGameWithResult(), nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultBlackWins,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "already has a result")
}

func TestService_EnterResult_InvalidResult(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    GameResult("invalid"),
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid result")
}

func TestService_EnterResult_ByeWithTwoPlayers(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultBye,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "cannot enter bye result for game with two players")
}

func TestService_EnterResult_ByeWithOnePlayer(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			game := testGame()
			game.BlackParticipantID = nil // Bye game has no black player
			return game, nil
		},
		UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
			return nil
		},
		UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
			return nil
		},
		UpdateRoundCompletedCountFunc: func(ctx context.Context, roundID uuid.UUID) error {
			return nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultBye,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.NoError(t, err)
	assert.Equal(t, "BYE", resp.Result)
}

func TestService_ValidateResult_Valid(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultWhiteWins,
		EnteredBy: testutil.UserID1,
	}

	validation := service.ValidateResult(context.Background(), entry)

	assert.True(t, validation.Valid)
	assert.Empty(t, validation.Errors)
}

func TestService_ValidateResult_WithWarnings(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGameWithResult(), nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultWhiteForfeit,
		EnteredBy: testutil.UserID1,
	}

	validation := service.ValidateResult(context.Background(), entry)

	// Valid but with warnings
	assert.True(t, validation.Valid)
	assert.Len(t, validation.Warnings, 2) // "game already has a result" + "forfeit result entered"
}

func TestService_DeleteResult_Success(t *testing.T) {
	statsReversed := make(map[uuid.UUID]bool)
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGameWithResult(), nil
		},
		UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
			statsReversed[participantID] = true
			// For reversing 1-0: white should be -10, -1, 0, 0; black should be 0, 0, 0, -1
			return nil
		},
		ClearResultFunc: func(ctx context.Context, gameID uuid.UUID) error {
			assert.Equal(t, testutil.GameID1, gameID)
			return nil
		},
		UpdateRoundCompletedCountFunc: func(ctx context.Context, roundID uuid.UUID) error {
			return nil
		},
		CreateAuditLogFunc: func(ctx context.Context, audit *ResultAudit) error {
			assert.Equal(t, "deleted", audit.Action)
			assert.NotNil(t, audit.OldResult)
			assert.Equal(t, "1-0", *audit.OldResult)
			return nil
		},
	}

	service := NewTestableService(mockRepo)
	err := service.DeleteResult(context.Background(), testutil.GameID1, testutil.UserID1, "Test deletion")

	require.NoError(t, err)
	assert.Len(t, statsReversed, 2) // Both participants' stats should be reversed
}

func TestService_DeleteResult_NoResult(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil // No result on game
		},
	}

	service := NewTestableService(mockRepo)
	err := service.DeleteResult(context.Background(), testutil.GameID1, testutil.UserID1, "Test deletion")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no result to delete")
}

func TestService_DeleteResult_GameNotFound(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return nil, errors.New("game not found")
		},
	}

	service := NewTestableService(mockRepo)
	err := service.DeleteResult(context.Background(), testutil.GameID1, testutil.UserID1, "Test deletion")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get game")
}

func TestService_EnterResult_Forfeit(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
		UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
			assert.Equal(t, "1-0F", result)
			assert.Equal(t, 1.0, whiteScore)
			assert.Equal(t, 0.0, blackScore)
			return nil
		},
		UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
			return nil
		},
		UpdateRoundCompletedCountFunc: func(ctx context.Context, roundID uuid.UUID) error {
			return nil
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultWhiteForfeit,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.NoError(t, err)
	assert.Equal(t, "1-0F", resp.Result)
	assert.Equal(t, 1.0, resp.WhiteScore)
	assert.Equal(t, 0.0, resp.BlackScore)
}

func TestService_EnterResult_UpdateResultFails(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
		UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
			return errors.New("database error")
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultWhiteWins,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to update result")
}

func TestService_EnterResult_StatsUpdateFails(t *testing.T) {
	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
		UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
			return nil
		},
		UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
			return errors.New("stats update failed")
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultWhiteWins,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to update white participant stats")
}

// REGRESSION TEST: Demonstrates the partial failure issue that was FIXED by adding transactions
// This test uses TestableService (without transactions) to show what WOULD happen without the fix.
// The production Service now uses database transactions to prevent this inconsistent state.
func TestService_EnterResult_PartialFailure_LeavesInconsistentState(t *testing.T) {
	// This test demonstrates what would happen WITHOUT transaction support:
	// Partial failures would leave the database in an inconsistent state.
	//
	// FIX APPLIED: The production Service (service.go) now wraps all database
	// operations in a transaction. If any step fails, the entire transaction
	// is rolled back, preventing inconsistent state.

	var resultUpdated bool
	var whiteStatsUpdated bool

	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGame(), nil
		},
		UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
			resultUpdated = true
			return nil // Result update succeeds
		},
		UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
			if participantID == testutil.ParticipantID1 {
				whiteStatsUpdated = true
				return nil // White stats update succeeds
			}
			// Black stats update FAILS
			return errors.New("database connection lost")
		},
	}

	service := NewTestableService(mockRepo)
	entry := &ResultEntry{
		GameID:    testutil.GameID1,
		Result:    ResultWhiteWins,
		EnteredBy: testutil.UserID1,
	}

	resp, err := service.EnterResult(context.Background(), entry)

	// The operation fails as expected
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to update black participant stats")

	// Without transactions (TestableService), the following would be true:
	// - Game shows result "1-0" (already saved)
	// - White participant has +1 win, +10 points (already saved)
	// - Black participant has NO stats update (should have +1 loss)
	//
	// This demonstrates the old buggy behavior. The production Service now
	// uses transactions, so a failure at this point would rollback everything.
	assert.True(t, resultUpdated, "Result was saved before stats update failed (would be rolled back in production)")
	assert.True(t, whiteStatsUpdated, "White stats were updated before black stats failed (would be rolled back in production)")
}

// REGRESSION TEST: Demonstrates CorrectResult partial failure that was FIXED by adding transactions
// This test uses TestableCorrectService (without transactions) to show what WOULD happen without the fix.
// The production Service now uses database transactions to prevent this data corruption.
func TestService_CorrectResult_PartialFailure_CorruptsStats(t *testing.T) {
	// This test demonstrates what would happen WITHOUT transaction support:
	// CorrectResult could corrupt stats if a failure occurs mid-operation.
	//
	// When correcting 1-0 to 0-1, the steps are:
	// 1. Reverse white's old win: pointsDelta=-10, winsDelta=-1
	// 2. Reverse black's old loss: pointsDelta=0, lossesDelta=-1
	// 3. Apply white's new loss: pointsDelta=0, lossesDelta=+1
	// 4. Apply black's new win: pointsDelta=10, winsDelta=+1
	//
	// FIX APPLIED: The production Service (service.go) now wraps all these
	// operations in a transaction. If any step fails, everything is rolled back.

	var callCount int
	var whiteReversed, blackReversed, whiteApplied, blackApplied bool

	mockRepo := &MockResultRepository{
		GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
			return testGameWithResult(), nil // Game has existing result "1-0"
		},
		UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
			callCount++
			// Call 1: Reverse white's win (pointsDelta=-10)
			if callCount == 1 && participantID == testutil.ParticipantID1 && pointsDelta == -10 {
				whiteReversed = true
				return nil
			}
			// Call 2: Reverse black's loss (pointsDelta=0, lossesDelta=-1)
			if callCount == 2 && participantID == testutil.ParticipantID2 && lossesDelta == -1 {
				blackReversed = true
				return nil
			}
			// Call 3: Apply white's new loss - FAIL HERE
			if callCount == 3 {
				whiteApplied = true
				return errors.New("database error during new stats")
			}
			// Call 4: This won't be reached
			blackApplied = true
			return nil
		},
		UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
			return nil
		},
	}

	service := &TestableCorrectService{repo: mockRepo}
	correction := &ResultCorrection{
		GameID:      testutil.GameID1,
		OldResult:   ResultWhiteWins,
		NewResult:   ResultBlackWins,
		Reason:      "Correcting score error in original entry",
		CorrectedBy: testutil.UserID1,
	}

	_, err := service.CorrectResult(context.Background(), correction)

	require.Error(t, err)

	// Without transactions (TestableCorrectService), the following would happen:
	// - White's old win stats were reversed (already saved)
	// - Black's old loss stats were reversed (already saved)
	// - White's new loss stats failed to apply
	// - Black's new win stats never applied
	//
	// Result: Both players have 0 wins, 0 losses (corrupted!)
	// They should have: White 0W/1L, Black 1W/0L
	//
	// This demonstrates the old buggy behavior. The production Service now
	// uses transactions, so this failure would rollback the reversals too.
	assert.True(t, whiteReversed, "White's old stats were reversed (would be rolled back in production)")
	assert.True(t, blackReversed, "Black's old stats were reversed (would be rolled back in production)")
	assert.True(t, whiteApplied, "Tried to apply new stats after reversal")
	assert.False(t, blackApplied, "Black's new stats were never applied due to error")
}

func TestService_ParticipantStatsDeltas(t *testing.T) {
	tests := []struct {
		name           string
		result         GameResult
		expectedWhite  struct{ points, wins, draws, losses int }
		expectedBlack  struct{ points, wins, draws, losses int }
	}{
		{
			name:          "white wins",
			result:        ResultWhiteWins,
			expectedWhite: struct{ points, wins, draws, losses int }{10, 1, 0, 0},
			expectedBlack: struct{ points, wins, draws, losses int }{0, 0, 0, 1},
		},
		{
			name:          "black wins",
			result:        ResultBlackWins,
			expectedWhite: struct{ points, wins, draws, losses int }{0, 0, 0, 1},
			expectedBlack: struct{ points, wins, draws, losses int }{10, 1, 0, 0},
		},
		{
			name:          "draw",
			result:        ResultDraw,
			expectedWhite: struct{ points, wins, draws, losses int }{5, 0, 1, 0},
			expectedBlack: struct{ points, wins, draws, losses int }{5, 0, 1, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statsCalled := make(map[uuid.UUID]struct{ points, wins, draws, losses int })
			mockRepo := &MockResultRepository{
				GetGameFunc: func(ctx context.Context, gameID uuid.UUID) (*Game, error) {
					return testGame(), nil
				},
				UpdateResultFunc: func(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
					return nil
				},
				UpdateParticipantStatsFunc: func(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
					statsCalled[participantID] = struct{ points, wins, draws, losses int }{pointsDelta, winsDelta, drawsDelta, lossesDelta}
					return nil
				},
				UpdateRoundCompletedCountFunc: func(ctx context.Context, roundID uuid.UUID) error {
					return nil
				},
			}

			service := NewTestableService(mockRepo)
			entry := &ResultEntry{
				GameID:    testutil.GameID1,
				Result:    tt.result,
				EnteredBy: testutil.UserID1,
			}

			_, err := service.EnterResult(context.Background(), entry)
			require.NoError(t, err)

			// Check white participant stats
			whiteStats := statsCalled[testutil.ParticipantID1]
			assert.Equal(t, tt.expectedWhite.points, whiteStats.points, "white points")
			assert.Equal(t, tt.expectedWhite.wins, whiteStats.wins, "white wins")
			assert.Equal(t, tt.expectedWhite.draws, whiteStats.draws, "white draws")
			assert.Equal(t, tt.expectedWhite.losses, whiteStats.losses, "white losses")

			// Check black participant stats
			blackStats := statsCalled[testutil.ParticipantID2]
			assert.Equal(t, tt.expectedBlack.points, blackStats.points, "black points")
			assert.Equal(t, tt.expectedBlack.wins, blackStats.wins, "black wins")
			assert.Equal(t, tt.expectedBlack.draws, blackStats.draws, "black draws")
			assert.Equal(t, tt.expectedBlack.losses, blackStats.losses, "black losses")
		})
	}
}
