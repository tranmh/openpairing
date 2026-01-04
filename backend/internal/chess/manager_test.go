package chess

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGameManager() *GameManager {
	logger := zerolog.Nop()
	return NewGameManager(logger)
}

func TestNewGameManager(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	assert.NotNil(t, gm)
	assert.NotNil(t, gm.games)
	assert.NotNil(t, gm.userGames)
	assert.Equal(t, 0, gm.GameCount())
}

func TestGameManager_CreateGame(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 3000)

	assert.NoError(t, err)
	assert.NotNil(t, game)
	assert.Equal(t, 1, gm.GameCount())
	assert.Equal(t, whiteID, game.WhiteUserID)
	assert.Equal(t, blackID, game.BlackUserID)
}

func TestGameManager_CreateGame_UserAlreadyInGame(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	_, err := gm.CreateGame(user1, user2, 300000, 0)
	require.NoError(t, err)

	// User1 tries to create another game
	_, err = gm.CreateGame(user1, user3, 300000, 0)
	assert.Error(t, err)
	assert.Equal(t, ErrUserAlreadyInGame, err)

	// User2 tries to create another game
	_, err = gm.CreateGame(user3, user2, 300000, 0)
	assert.Error(t, err)
	assert.Equal(t, ErrUserAlreadyInGame, err)
}

func TestGameManager_GetGame(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	retrieved := gm.GetGame(game.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, game.ID, retrieved.ID)
}

func TestGameManager_GetGame_NotFound(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	retrieved := gm.GetGame(uuid.New())
	assert.Nil(t, retrieved)
}

func TestGameManager_GetUserGame(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	// Get game by white user
	whiteGame := gm.GetUserGame(whiteID)
	assert.NotNil(t, whiteGame)
	assert.Equal(t, game.ID, whiteGame.ID)

	// Get game by black user
	blackGame := gm.GetUserGame(blackID)
	assert.NotNil(t, blackGame)
	assert.Equal(t, game.ID, blackGame.ID)

	// Random user has no game
	randomGame := gm.GetUserGame(uuid.New())
	assert.Nil(t, randomGame)
}

func TestGameManager_StartGame(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	assert.NoError(t, err)

	retrieved := gm.GetGame(game.ID)
	assert.Equal(t, StatusOngoing, retrieved.Status)
}

func TestGameManager_StartGame_NotFound(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	err := gm.StartGame(uuid.New())
	assert.Error(t, err)
	assert.Equal(t, ErrGameNotFound, err)
}

func TestGameManager_StartGame_AlreadyStarted(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrGameAlreadyStarted, err)
}

func TestGameManager_MakeMove(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	// Make a move
	updatedGame, err := gm.MakeMove(game.ID, whiteID, "e2e4")
	assert.NoError(t, err)
	assert.Equal(t, 1, updatedGame.MoveCount)
}

func TestGameManager_MakeMove_NotFound(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	_, err := gm.MakeMove(uuid.New(), uuid.New(), "e2e4")
	assert.Error(t, err)
	assert.Equal(t, ErrGameNotFound, err)
}

func TestGameManager_Resign(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	err = gm.Resign(game.ID, whiteID)
	assert.NoError(t, err)

	// Check game is completed
	retrieved := gm.GetGame(game.ID)
	assert.Equal(t, StatusCompleted, retrieved.Status)
	assert.Equal(t, "0-1", retrieved.Result)

	// User should be removed from userGames
	userGame := gm.GetUserGame(whiteID)
	assert.Nil(t, userGame)
}

func TestGameManager_OfferDraw(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	err = gm.OfferDraw(game.ID, whiteID)
	assert.NoError(t, err)

	retrieved := gm.GetGame(game.ID)
	assert.NotNil(t, retrieved.DrawOfferedBy)
}

func TestGameManager_AcceptDraw(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	err = gm.OfferDraw(game.ID, whiteID)
	require.NoError(t, err)

	err = gm.AcceptDraw(game.ID, blackID)
	assert.NoError(t, err)

	retrieved := gm.GetGame(game.ID)
	assert.Equal(t, StatusCompleted, retrieved.Status)
	assert.Equal(t, "1/2-1/2", retrieved.Result)
}

func TestGameManager_DeclineDraw(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	err = gm.OfferDraw(game.ID, whiteID)
	require.NoError(t, err)

	err = gm.DeclineDraw(game.ID, blackID)
	assert.NoError(t, err)

	retrieved := gm.GetGame(game.ID)
	assert.Nil(t, retrieved.DrawOfferedBy)
	assert.Equal(t, StatusOngoing, retrieved.Status)
}

func TestGameManager_Abort(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	err = gm.Abort(game.ID, whiteID)
	assert.NoError(t, err)

	retrieved := gm.GetGame(game.ID)
	assert.Equal(t, StatusAborted, retrieved.Status)
}

func TestGameManager_GetActiveGames(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	// Create multiple games
	game1, _ := gm.CreateGame(uuid.New(), uuid.New(), 300000, 0)
	game2, _ := gm.CreateGame(uuid.New(), uuid.New(), 300000, 0)
	_, _ = gm.CreateGame(uuid.New(), uuid.New(), 300000, 0) // game3 remains in waiting status

	// Start some games
	gm.StartGame(game1.ID)
	gm.StartGame(game2.ID)

	activeGames := gm.GetActiveGames()
	assert.Len(t, activeGames, 3) // All waiting and ongoing games
}

func TestGameManager_RemoveGame(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	game, _ := gm.CreateGame(uuid.New(), uuid.New(), 300000, 0)

	assert.Equal(t, 1, gm.GameCount())

	gm.RemoveGame(game.ID)

	assert.Equal(t, 0, gm.GameCount())
	assert.Nil(t, gm.GetGame(game.ID))
}

func TestGameManager_ClockLoop(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	// Create game with very short time
	game, err := gm.CreateGame(whiteID, blackID, 500, 0) // 500ms
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	// Wait for clock to tick and game to end
	time.Sleep(700 * time.Millisecond)

	retrieved := gm.GetGame(game.ID)
	assert.Equal(t, StatusCompleted, retrieved.Status)
	assert.Equal(t, TerminationTimeout, retrieved.Termination)
}

func TestGameManager_CleanupAfterCompletion(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	// Users should be in userGames
	assert.NotNil(t, gm.GetUserGame(whiteID))
	assert.NotNil(t, gm.GetUserGame(blackID))

	// Resign to complete the game
	err = gm.Resign(game.ID, whiteID)
	require.NoError(t, err)

	// Users should be removed from userGames
	assert.Nil(t, gm.GetUserGame(whiteID))
	assert.Nil(t, gm.GetUserGame(blackID))

	// But game should still be retrievable (for history)
	assert.NotNil(t, gm.GetGame(game.ID))
}

func TestGameManager_ConcurrentAccess(t *testing.T) {
	gm := newTestGameManager()
	defer gm.Stop()

	whiteID := uuid.New()
	blackID := uuid.New()

	game, err := gm.CreateGame(whiteID, blackID, 300000, 0)
	require.NoError(t, err)

	err = gm.StartGame(game.ID)
	require.NoError(t, err)

	// Simulate concurrent access
	done := make(chan bool)

	// Multiple goroutines reading game state
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = gm.GetGame(game.ID)
				_ = gm.GetUserGame(whiteID)
				_ = gm.GameCount()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
