package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/openpairing/openpairing/internal/chess"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, check origin properly
		return true
	},
}

// Client represents a WebSocket client
type Client struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Conn       *websocket.Conn
	Send       chan []byte
	Hub        *Hub
	GameID     *uuid.UUID // Current game being watched/played
	IsSpectator bool
	mu         sync.Mutex
}

// Hub maintains the set of active clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Clients by user ID
	userClients map[uuid.UUID]*Client

	// Clients by game ID (including spectators)
	gameClients map[uuid.UUID]map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Game manager
	gameManager *chess.GameManager

	logger zerolog.Logger

	mutex sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	Type   string          `json:"type"`
	Data   json.RawMessage `json:"data,omitempty"`
	GameID *uuid.UUID      `json:"gameId,omitempty"`
	Client *Client         `json:"-"`
}

// NewHub creates a new WebSocket hub
func NewHub(gameManager *chess.GameManager, logger zerolog.Logger) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		userClients: make(map[uuid.UUID]*Client),
		gameClients: make(map[uuid.UUID]map[*Client]bool),
		broadcast:   make(chan *Message, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		gameManager: gameManager,
		logger:      logger.With().Str("component", "websocket_hub").Logger(),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.userClients[client.UserID] = client
			h.mutex.Unlock()

			h.logger.Debug().
				Str("client_id", client.ID.String()).
				Str("user_id", client.UserID.String()).
				Msg("Client registered")

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.userClients, client.UserID)

				// Remove from game clients
				if client.GameID != nil {
					if gameClients, ok := h.gameClients[*client.GameID]; ok {
						delete(gameClients, client)
						if len(gameClients) == 0 {
							delete(h.gameClients, *client.GameID)
						}
					}
				}

				close(client.Send)
			}
			h.mutex.Unlock()

			h.logger.Debug().
				Str("client_id", client.ID.String()).
				Msg("Client unregistered")

		case message := <-h.broadcast:
			h.handleMessage(message)
		}
	}
}

func (h *Hub) handleMessage(msg *Message) {
	h.logger.Debug().
		Str("type", msg.Type).
		Msg("Handling message")

	switch msg.Type {
	case "join_game":
		h.handleJoinGame(msg)
	case "leave_game":
		h.handleLeaveGame(msg)
	case "make_move":
		h.handleMakeMove(msg)
	case "offer_draw":
		h.handleOfferDraw(msg)
	case "accept_draw":
		h.handleAcceptDraw(msg)
	case "decline_draw":
		h.handleDeclineDraw(msg)
	case "resign":
		h.handleResign(msg)
	case "abort":
		h.handleAbort(msg)
	case "spectate":
		h.handleSpectate(msg)
	}
}

// JoinGameData represents the data for joining a game
type JoinGameData struct {
	GameID string `json:"gameId"`
}

func (h *Hub) handleJoinGame(msg *Message) {
	var data JoinGameData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		h.sendError(msg.Client, "invalid join data")
		return
	}

	gameID, err := uuid.Parse(data.GameID)
	if err != nil {
		h.sendError(msg.Client, "invalid game ID")
		return
	}

	game := h.gameManager.GetGame(gameID)
	if game == nil {
		h.sendError(msg.Client, "game not found")
		return
	}

	h.mutex.Lock()
	msg.Client.GameID = &gameID
	msg.Client.IsSpectator = !game.IsUserInGame(msg.Client.UserID)

	if _, ok := h.gameClients[gameID]; !ok {
		h.gameClients[gameID] = make(map[*Client]bool)
	}
	h.gameClients[gameID][msg.Client] = true
	h.mutex.Unlock()

	// Send current game state
	state := game.GetState()
	h.sendToClient(msg.Client, "game_state", state)

	// Notify spectator count if spectator
	if msg.Client.IsSpectator {
		h.broadcastToGame(gameID, "spectator_count", map[string]int{
			"count": h.getSpectatorCount(gameID),
		})
	}
}

func (h *Hub) handleLeaveGame(msg *Message) {
	if msg.Client.GameID == nil {
		return
	}

	gameID := *msg.Client.GameID

	h.mutex.Lock()
	if gameClients, ok := h.gameClients[gameID]; ok {
		delete(gameClients, msg.Client)
		if len(gameClients) == 0 {
			delete(h.gameClients, gameID)
		}
	}
	msg.Client.GameID = nil
	msg.Client.IsSpectator = false
	h.mutex.Unlock()

	// Notify spectator count
	h.broadcastToGame(gameID, "spectator_count", map[string]int{
		"count": h.getSpectatorCount(gameID),
	})
}

// MakeMoveData represents the data for making a move
type MakeMoveData struct {
	GameID string `json:"gameId"`
	Move   string `json:"move"`
}

func (h *Hub) handleMakeMove(msg *Message) {
	var data MakeMoveData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		h.sendError(msg.Client, "invalid move data")
		return
	}

	gameID, err := uuid.Parse(data.GameID)
	if err != nil {
		h.sendError(msg.Client, "invalid game ID")
		return
	}

	game, err := h.gameManager.MakeMove(gameID, msg.Client.UserID, data.Move)
	if err != nil {
		h.sendError(msg.Client, err.Error())
		return
	}

	// Broadcast move to all game clients
	state := game.GetState()
	h.broadcastToGame(gameID, "move_made", map[string]interface{}{
		"move":  data.Move,
		"state": state,
	})

	// If game ended, broadcast game over
	if game.Status == chess.StatusCompleted {
		h.broadcastToGame(gameID, "game_over", map[string]interface{}{
			"result":      game.Result,
			"termination": game.Termination,
			"state":       state,
		})
	}
}

func (h *Hub) handleOfferDraw(msg *Message) {
	if msg.Client.GameID == nil {
		h.sendError(msg.Client, "not in a game")
		return
	}

	gameID := *msg.Client.GameID
	if err := h.gameManager.OfferDraw(gameID, msg.Client.UserID); err != nil {
		h.sendError(msg.Client, err.Error())
		return
	}

	h.broadcastToGame(gameID, "draw_offered", map[string]string{
		"by": msg.Client.UserID.String(),
	})
}

func (h *Hub) handleAcceptDraw(msg *Message) {
	if msg.Client.GameID == nil {
		h.sendError(msg.Client, "not in a game")
		return
	}

	gameID := *msg.Client.GameID
	if err := h.gameManager.AcceptDraw(gameID, msg.Client.UserID); err != nil {
		h.sendError(msg.Client, err.Error())
		return
	}

	game := h.gameManager.GetGame(gameID)
	if game == nil {
		// Game was cleaned up between AcceptDraw and GetGame - notify draw accepted but can't broadcast state
		h.logger.Warn().Str("game_id", gameID.String()).Msg("Game not found after accepting draw")
		h.broadcastToGame(gameID, "game_over", map[string]interface{}{
			"result":      "1/2-1/2",
			"termination": "agreement",
		})
		return
	}
	state := game.GetState()

	h.broadcastToGame(gameID, "game_over", map[string]interface{}{
		"result":      game.Result,
		"termination": game.Termination,
		"state":       state,
	})
}

func (h *Hub) handleDeclineDraw(msg *Message) {
	if msg.Client.GameID == nil {
		h.sendError(msg.Client, "not in a game")
		return
	}

	gameID := *msg.Client.GameID
	if err := h.gameManager.DeclineDraw(gameID, msg.Client.UserID); err != nil {
		h.sendError(msg.Client, err.Error())
		return
	}

	h.broadcastToGame(gameID, "draw_declined", nil)
}

func (h *Hub) handleResign(msg *Message) {
	if msg.Client.GameID == nil {
		h.sendError(msg.Client, "not in a game")
		return
	}

	gameID := *msg.Client.GameID
	if err := h.gameManager.Resign(gameID, msg.Client.UserID); err != nil {
		h.sendError(msg.Client, err.Error())
		return
	}

	game := h.gameManager.GetGame(gameID)
	if game == nil {
		// Game was cleaned up between Resign and GetGame - notify resignation but can't broadcast state
		h.logger.Warn().Str("game_id", gameID.String()).Msg("Game not found after resignation")
		h.broadcastToGame(gameID, "game_over", map[string]interface{}{
			"termination": "resignation",
		})
		return
	}
	state := game.GetState()

	h.broadcastToGame(gameID, "game_over", map[string]interface{}{
		"result":      game.Result,
		"termination": game.Termination,
		"state":       state,
	})
}

func (h *Hub) handleAbort(msg *Message) {
	if msg.Client.GameID == nil {
		h.sendError(msg.Client, "not in a game")
		return
	}

	gameID := *msg.Client.GameID
	if err := h.gameManager.Abort(gameID, msg.Client.UserID); err != nil {
		h.sendError(msg.Client, err.Error())
		return
	}

	h.broadcastToGame(gameID, "game_aborted", nil)
}

func (h *Hub) handleSpectate(msg *Message) {
	var data JoinGameData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		h.sendError(msg.Client, "invalid spectate data")
		return
	}

	gameID, err := uuid.Parse(data.GameID)
	if err != nil {
		h.sendError(msg.Client, "invalid game ID")
		return
	}

	game := h.gameManager.GetGame(gameID)
	if game == nil {
		h.sendError(msg.Client, "game not found")
		return
	}

	h.mutex.Lock()
	msg.Client.GameID = &gameID
	msg.Client.IsSpectator = true

	if _, ok := h.gameClients[gameID]; !ok {
		h.gameClients[gameID] = make(map[*Client]bool)
	}
	h.gameClients[gameID][msg.Client] = true
	h.mutex.Unlock()

	// Send current game state
	state := game.GetState()
	h.sendToClient(msg.Client, "game_state", state)

	// Notify spectator count
	h.broadcastToGame(gameID, "spectator_count", map[string]int{
		"count": h.getSpectatorCount(gameID),
	})
}

func (h *Hub) sendError(client *Client, message string) {
	h.sendToClient(client, "error", map[string]string{"message": message})
}

func (h *Hub) sendToClient(client *Client, msgType string, data interface{}) {
	msg := map[string]interface{}{
		"type": msgType,
		"data": data,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case client.Send <- bytes:
	default:
		// Client buffer full, skip
	}
}

func (h *Hub) broadcastToGame(gameID uuid.UUID, msgType string, data interface{}) {
	msg := map[string]interface{}{
		"type": msgType,
		"data": data,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mutex.RLock()
	clients := h.gameClients[gameID]
	h.mutex.RUnlock()

	for client := range clients {
		select {
		case client.Send <- bytes:
		default:
			// Client buffer full, skip
		}
	}
}

func (h *Hub) getSpectatorCount(gameID uuid.UUID) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	count := 0
	if clients, ok := h.gameClients[gameID]; ok {
		for client := range clients {
			if client.IsSpectator {
				count++
			}
		}
	}
	return count
}

// BroadcastClocks broadcasts clock updates to all game clients
func (h *Hub) BroadcastClocks() {
	h.mutex.RLock()
	gameIDs := make([]uuid.UUID, 0, len(h.gameClients))
	for id := range h.gameClients {
		gameIDs = append(gameIDs, id)
	}
	h.mutex.RUnlock()

	for _, gameID := range gameIDs {
		game := h.gameManager.GetGame(gameID)
		if game == nil || game.Status != chess.StatusOngoing {
			continue
		}

		state := game.GetState()
		h.broadcastToGame(gameID, "clock_update", map[string]interface{}{
			"whiteClockMs": state.WhiteClockMs,
			"blackClockMs": state.BlackClockMs,
		})
	}
}

// HandleWebSocket handles WebSocket upgrade and communication
func (h *Hub) HandleWebSocket(c *gin.Context) {
	// Get user ID from auth context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to upgrade to WebSocket")
		return
	}

	client := &Client{
		ID:     uuid.New(),
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h,
	}

	h.register <- client

	// Start read/write goroutines
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(65536)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			continue
		}

		msg.Client = c
		c.Hub.broadcast <- &msg
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain the queue
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// RegisterClient registers a client with the hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client from the hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	c.readPump()
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	c.writePump()
}
