package chess

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// GameManager manages active chess games
type GameManager struct {
	games         map[uuid.UUID]*GameState
	userGames     map[uuid.UUID]uuid.UUID // user ID -> game ID
	mutex         sync.RWMutex
	logger        zerolog.Logger
	clockTicker   *time.Ticker
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
}

// NewGameManager creates a new game manager
func NewGameManager(logger zerolog.Logger) *GameManager {
	gm := &GameManager{
		games:     make(map[uuid.UUID]*GameState),
		userGames: make(map[uuid.UUID]uuid.UUID),
		logger:    logger.With().Str("component", "game_manager").Logger(),
		stopChan:  make(chan struct{}),
	}

	// Start clock update goroutine
	gm.clockTicker = time.NewTicker(100 * time.Millisecond)
	go gm.clockLoop()

	// Start cleanup goroutine to remove completed games after a delay
	gm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go gm.cleanupLoop()

	return gm
}

// Stop stops the game manager
func (gm *GameManager) Stop() {
	close(gm.stopChan)
	gm.clockTicker.Stop()
	gm.cleanupTicker.Stop()
}

func (gm *GameManager) clockLoop() {
	for {
		select {
		case <-gm.clockTicker.C:
			gm.updateClocks()
		case <-gm.stopChan:
			return
		}
	}
}

func (gm *GameManager) cleanupLoop() {
	for {
		select {
		case <-gm.cleanupTicker.C:
			gm.cleanupCompletedGames()
		case <-gm.stopChan:
			return
		}
	}
}

// cleanupCompletedGames removes completed/aborted games that ended more than 10 minutes ago
func (gm *GameManager) cleanupCompletedGames() {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	removed := 0

	for id, game := range gm.games {
		if (game.Status == StatusCompleted || game.Status == StatusAborted) && game.CompletedAt != nil && game.CompletedAt.Before(cutoff) {
			delete(gm.games, id)
			removed++
		}
	}

	if removed > 0 {
		gm.logger.Info().Int("removed", removed).Int("remaining", len(gm.games)).Msg("Cleaned up old games")
	}
}

func (gm *GameManager) updateClocks() {
	gm.mutex.RLock()
	gameIDs := make([]uuid.UUID, 0, len(gm.games))
	for id := range gm.games {
		gameIDs = append(gameIDs, id)
	}
	gm.mutex.RUnlock()

	for _, id := range gameIDs {
		game := gm.GetGame(id)
		if game == nil || game.Status != StatusOngoing {
			continue
		}

		_, _, expired := game.UpdateClock()
		if expired {
			gm.logger.Info().
				Str("game_id", id.String()).
				Str("result", game.Result).
				Msg("Game ended by timeout")
		}
	}
}

// CreateGame creates a new game
func (gm *GameManager) CreateGame(whiteUserID, blackUserID uuid.UUID, initialTimeMs, incrementMs int64) (*GameState, error) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	// Check if either user is already in a game
	if _, exists := gm.userGames[whiteUserID]; exists {
		return nil, ErrUserAlreadyInGame
	}
	if _, exists := gm.userGames[blackUserID]; exists {
		return nil, ErrUserAlreadyInGame
	}

	game := NewGame(whiteUserID, blackUserID, initialTimeMs, incrementMs)
	gm.games[game.ID] = game
	gm.userGames[whiteUserID] = game.ID
	gm.userGames[blackUserID] = game.ID

	gm.logger.Info().
		Str("game_id", game.ID.String()).
		Str("white", whiteUserID.String()).
		Str("black", blackUserID.String()).
		Int64("time_ms", initialTimeMs).
		Int64("increment_ms", incrementMs).
		Msg("Game created")

	return game, nil
}

// GetGame retrieves a game by ID
func (gm *GameManager) GetGame(gameID uuid.UUID) *GameState {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	return gm.games[gameID]
}

// GetUserGame retrieves the game a user is currently in
func (gm *GameManager) GetUserGame(userID uuid.UUID) *GameState {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	gameID, exists := gm.userGames[userID]
	if !exists {
		return nil
	}
	return gm.games[gameID]
}

// StartGame starts a game
func (gm *GameManager) StartGame(gameID uuid.UUID) error {
	game := gm.GetGame(gameID)
	if game == nil {
		return ErrGameNotFound
	}

	if game.Status != StatusWaiting {
		return ErrGameAlreadyStarted
	}

	game.Start()

	gm.logger.Info().
		Str("game_id", gameID.String()).
		Msg("Game started")

	return nil
}

// MakeMove makes a move in a game
func (gm *GameManager) MakeMove(gameID uuid.UUID, userID uuid.UUID, move string) (*GameState, error) {
	game := gm.GetGame(gameID)
	if game == nil {
		return nil, ErrGameNotFound
	}

	if err := game.MakeMove(userID, move); err != nil {
		return nil, err
	}

	gm.logger.Debug().
		Str("game_id", gameID.String()).
		Str("user", userID.String()).
		Str("move", move).
		Int("move_count", game.MoveCount).
		Msg("Move made")

	// If game ended, clean up
	if game.Status == StatusCompleted || game.Status == StatusAborted {
		gm.cleanupGame(gameID)
	}

	return game, nil
}

// Resign resigns a game
func (gm *GameManager) Resign(gameID uuid.UUID, userID uuid.UUID) error {
	game := gm.GetGame(gameID)
	if game == nil {
		return ErrGameNotFound
	}

	if err := game.Resign(userID); err != nil {
		return err
	}

	gm.logger.Info().
		Str("game_id", gameID.String()).
		Str("user", userID.String()).
		Str("result", game.Result).
		Msg("Player resigned")

	gm.cleanupGame(gameID)
	return nil
}

// OfferDraw offers a draw
func (gm *GameManager) OfferDraw(gameID uuid.UUID, userID uuid.UUID) error {
	game := gm.GetGame(gameID)
	if game == nil {
		return ErrGameNotFound
	}

	if err := game.OfferDraw(userID); err != nil {
		return err
	}

	gm.logger.Debug().
		Str("game_id", gameID.String()).
		Str("user", userID.String()).
		Msg("Draw offered")

	return nil
}

// AcceptDraw accepts a draw offer
func (gm *GameManager) AcceptDraw(gameID uuid.UUID, userID uuid.UUID) error {
	game := gm.GetGame(gameID)
	if game == nil {
		return ErrGameNotFound
	}

	if err := game.AcceptDraw(userID); err != nil {
		return err
	}

	gm.logger.Info().
		Str("game_id", gameID.String()).
		Msg("Draw accepted")

	gm.cleanupGame(gameID)
	return nil
}

// DeclineDraw declines a draw offer
func (gm *GameManager) DeclineDraw(gameID uuid.UUID, userID uuid.UUID) error {
	game := gm.GetGame(gameID)
	if game == nil {
		return ErrGameNotFound
	}

	return game.DeclineDraw(userID)
}

// Abort aborts a game
func (gm *GameManager) Abort(gameID uuid.UUID, userID uuid.UUID) error {
	game := gm.GetGame(gameID)
	if game == nil {
		return ErrGameNotFound
	}

	if err := game.Abort(userID); err != nil {
		return err
	}

	gm.logger.Info().
		Str("game_id", gameID.String()).
		Msg("Game aborted")

	gm.cleanupGame(gameID)
	return nil
}

func (gm *GameManager) cleanupGame(gameID uuid.UUID) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	game, exists := gm.games[gameID]
	if !exists {
		return
	}

	delete(gm.userGames, game.WhiteUserID)
	delete(gm.userGames, game.BlackUserID)
	// Don't delete the game immediately to allow retrieving final state
	// A separate cleanup routine could remove old completed games
}

// GetActiveGames returns all active games
func (gm *GameManager) GetActiveGames() []*GameState {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	games := make([]*GameState, 0)
	for _, game := range gm.games {
		if game.Status == StatusOngoing || game.Status == StatusWaiting {
			games = append(games, game)
		}
	}
	return games
}

// RemoveGame removes a completed game from memory
func (gm *GameManager) RemoveGame(gameID uuid.UUID) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	delete(gm.games, gameID)
}

// GameCount returns the number of games
func (gm *GameManager) GameCount() int {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	return len(gm.games)
}

// Errors
var (
	ErrGameNotFound      = &GameError{"game not found"}
	ErrUserAlreadyInGame = &GameError{"user already in a game"}
	ErrGameAlreadyStarted = &GameError{"game already started"}
)

// GameError represents a game-related error
type GameError struct {
	Message string
}

func (e *GameError) Error() string {
	return e.Message
}

// GameEventType represents the type of game event
type GameEventType string

const (
	EventMove        GameEventType = "move"
	EventGameStart   GameEventType = "game_start"
	EventGameEnd     GameEventType = "game_end"
	EventDrawOffer   GameEventType = "draw_offer"
	EventDrawDecline GameEventType = "draw_decline"
	EventClockUpdate GameEventType = "clock_update"
	EventSpectatorJoin GameEventType = "spectator_join"
	EventSpectatorLeave GameEventType = "spectator_leave"
)

// GameEvent represents an event in a game
type GameEvent struct {
	Type    GameEventType     `json:"type"`
	GameID  uuid.UUID         `json:"gameId"`
	UserID  *uuid.UUID        `json:"userId,omitempty"`
	Move    string            `json:"move,omitempty"`
	State   *GameStateSnapshot `json:"state,omitempty"`
}
