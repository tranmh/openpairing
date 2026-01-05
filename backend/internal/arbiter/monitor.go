package arbiter

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// MonitorHub manages WebSocket connections for live monitoring
type MonitorHub struct {
	mu           sync.RWMutex
	connections  map[uuid.UUID]map[*MonitorClient]bool // tournamentID -> clients
	service      *Service
	logger       zerolog.Logger
	updateTicker *time.Ticker
	done         chan struct{}
}

// MonitorClient represents a connected arbiter client
type MonitorClient struct {
	conn         *websocket.Conn
	arbiterID    uuid.UUID
	tournamentID uuid.UUID
	send         chan []byte
	hub          *MonitorHub
}

// MonitorUpdate represents a real-time update sent to clients
type MonitorUpdate struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// GameUpdate represents a game status update
type GameUpdate struct {
	GameID           uuid.UUID  `json:"gameId"`
	WhitePlayer      string     `json:"whitePlayer"`
	BlackPlayer      string     `json:"blackPlayer"`
	BoardNumber      int        `json:"boardNumber"`
	Status           string     `json:"status"`
	Result           *string    `json:"result,omitempty"`
	MoveCount        int        `json:"moveCount,omitempty"`
	WhiteTimeSeconds int        `json:"whiteTimeSeconds,omitempty"`
	BlackTimeSeconds int        `json:"blackTimeSeconds,omitempty"`
	LastMove         string     `json:"lastMove,omitempty"`
	LastMoveAt       *time.Time `json:"lastMoveAt,omitempty"`
}

// RoundUpdate represents a round status update
type RoundUpdate struct {
	RoundNumber    int    `json:"roundNumber"`
	Status         string `json:"status"`
	TotalGames     int    `json:"totalGames"`
	CompletedGames int    `json:"completedGames"`
	OngoingGames   int    `json:"ongoingGames"`
	PendingResults int    `json:"pendingResults"`
}

// NewMonitorHub creates a new monitor hub
func NewMonitorHub(service *Service, logger zerolog.Logger) *MonitorHub {
	hub := &MonitorHub{
		connections: make(map[uuid.UUID]map[*MonitorClient]bool),
		service:     service,
		logger:      logger.With().Str("component", "monitor-hub").Logger(),
		done:        make(chan struct{}),
	}

	return hub
}

// Start starts the monitor hub's update loop
func (h *MonitorHub) Start(ctx context.Context) {
	h.updateTicker = time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.done:
				return
			case <-h.updateTicker.C:
				h.broadcastUpdates()
			}
		}
	}()

	h.logger.Info().Msg("Monitor hub started")
}

// Stop stops the monitor hub
func (h *MonitorHub) Stop() {
	if h.updateTicker != nil {
		h.updateTicker.Stop()
	}
	close(h.done)
	h.logger.Info().Msg("Monitor hub stopped")
}

// broadcastUpdates sends tournament updates to all connected clients
func (h *MonitorHub) broadcastUpdates() {
	h.mu.RLock()
	tournaments := make([]uuid.UUID, 0, len(h.connections))
	for tournamentID := range h.connections {
		tournaments = append(tournaments, tournamentID)
	}
	h.mu.RUnlock()

	ctx := context.Background()

	for _, tournamentID := range tournaments {
		h.mu.RLock()
		clients := h.connections[tournamentID]
		if len(clients) == 0 {
			h.mu.RUnlock()
			continue
		}

		// Get status for any connected arbiter
		var arbiterID uuid.UUID
		for client := range clients {
			arbiterID = client.arbiterID
			break
		}
		h.mu.RUnlock()

		// Get tournament status
		status, err := h.service.GetTournamentStatus(ctx, tournamentID, arbiterID)
		if err != nil {
			h.logger.Error().Err(err).Str("tournament_id", tournamentID.String()).Msg("Failed to get status for broadcast")
			continue
		}

		// Create round update
		var roundUpdate *RoundUpdate
		if status.CurrentRound != nil {
			roundUpdate = &RoundUpdate{
				RoundNumber:    status.CurrentRound.RoundNumber,
				Status:         status.CurrentRound.Status,
				TotalGames:     len(status.ActiveGames) + status.PendingResults,
				CompletedGames: status.Statistics.CompletedGames,
				OngoingGames:   len(status.ActiveGames),
				PendingResults: status.PendingResults,
			}
		}

		// Create game updates
		gameUpdates := make([]GameUpdate, 0, len(status.ActiveGames))
		for _, game := range status.ActiveGames {
			gu := GameUpdate{
				GameID:      game.Game.ID,
				BoardNumber: game.Game.BoardNumber,
				Status:      string(game.Game.Status),
				Result:      game.Game.Result,
			}

			if game.WhitePlayer != nil {
				gu.WhitePlayer = game.WhitePlayer.Name
			}
			if game.BlackPlayer != nil {
				gu.BlackPlayer = game.BlackPlayer.Name
			}

			gameUpdates = append(gameUpdates, gu)
		}

		// Build update message
		update := MonitorUpdate{
			Type:      "status_update",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"round":      roundUpdate,
				"games":      gameUpdates,
				"statistics": status.Statistics,
			},
		}

		data, err := json.Marshal(update)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to marshal update")
			continue
		}

		// Send to all clients for this tournament
		h.mu.RLock()
		for client := range clients {
			select {
			case client.send <- data:
			default:
				// Client not ready, skip
			}
		}
		h.mu.RUnlock()
	}
}

// Register adds a client to the hub
func (h *MonitorHub) Register(client *MonitorClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[client.tournamentID] == nil {
		h.connections[client.tournamentID] = make(map[*MonitorClient]bool)
	}
	h.connections[client.tournamentID][client] = true

	h.logger.Info().
		Str("tournament_id", client.tournamentID.String()).
		Str("arbiter_id", client.arbiterID.String()).
		Msg("Client registered for monitoring")
}

// Unregister removes a client from the hub
func (h *MonitorHub) Unregister(client *MonitorClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.connections[client.tournamentID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.connections, client.tournamentID)
		}
	}

	close(client.send)

	h.logger.Info().
		Str("tournament_id", client.tournamentID.String()).
		Str("arbiter_id", client.arbiterID.String()).
		Msg("Client unregistered from monitoring")
}

// BroadcastGameUpdate broadcasts a game update to all clients monitoring a tournament
func (h *MonitorHub) BroadcastGameUpdate(tournamentID uuid.UUID, update GameUpdate) {
	msg := MonitorUpdate{
		Type:      "game_update",
		Timestamp: time.Now(),
		Data:      update,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.connections[tournamentID] {
		select {
		case client.send <- data:
		default:
		}
	}
}

// BroadcastAlert broadcasts an urgent alert to all clients monitoring a tournament
func (h *MonitorHub) BroadcastAlert(tournamentID uuid.UUID, alertType, message string) {
	msg := MonitorUpdate{
		Type:      "alert",
		Timestamp: time.Now(),
		Data: map[string]string{
			"alertType": alertType,
			"message":   message,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.connections[tournamentID] {
		select {
		case client.send <- data:
		default:
		}
	}
}

// HandleWebSocket handles WebSocket connections for live monitoring
func (h *MonitorHub) HandleWebSocket(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	// Get arbiter ID from context
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	arbiterID, ok := userIDRaw.(uuid.UUID)
	if !ok {
		// Try parsing as string
		if strID, ok := userIDRaw.(string); ok {
			arbiterID, err = uuid.Parse(strID)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID type"})
			return
		}
	}

	// Verify arbiter has permission
	ctx := c.Request.Context()
	if err := h.service.canManageTournament(ctx, tournamentID, arbiterID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized for this tournament"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to upgrade WebSocket")
		return
	}

	client := &MonitorClient{
		conn:         conn,
		arbiterID:    arbiterID,
		tournamentID: tournamentID,
		send:         make(chan []byte, 256),
		hub:          h,
	}

	h.Register(client)

	// Send initial status
	status, err := h.service.GetTournamentStatus(ctx, tournamentID, arbiterID)
	if err == nil {
		initial := MonitorUpdate{
			Type:      "initial_status",
			Timestamp: time.Now(),
			Data:      status,
		}
		if data, err := json.Marshal(initial); err == nil {
			client.send <- data
		}
	}

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// readPump reads messages from the WebSocket connection
func (c *MonitorClient) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}

		// Handle incoming messages (e.g., command requests)
		var cmd struct {
			Command string          `json:"command"`
			Data    json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(message, &cmd); err == nil {
			c.handleCommand(cmd.Command, cmd.Data)
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *MonitorClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleCommand handles incoming commands from the client
func (c *MonitorClient) handleCommand(command string, data json.RawMessage) {
	switch command {
	case "refresh":
		// Request immediate status refresh
		ctx := context.Background()
		status, err := c.hub.service.GetTournamentStatus(ctx, c.tournamentID, c.arbiterID)
		if err == nil {
			update := MonitorUpdate{
				Type:      "status_update",
				Timestamp: time.Now(),
				Data:      status,
			}
			if d, err := json.Marshal(update); err == nil {
				c.send <- d
			}
		}
	case "focus_game":
		// Client wants to focus on a specific game
		// Could trigger more frequent updates for that game
		c.hub.logger.Debug().
			Str("command", command).
			RawJSON("data", data).
			Msg("Received command")
	}
}
