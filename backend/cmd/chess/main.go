package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/openpairing/openpairing/internal/chess"
	"github.com/openpairing/openpairing/internal/config"
	"github.com/openpairing/openpairing/internal/database"
	"github.com/openpairing/openpairing/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	hub      *server.Hub
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// In production, check origin properly
			return true
		},
	}
)

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENVIRONMENT") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Load configuration
	cfg := config.Load()

	// Connect to database (for game persistence)
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Warn().Err(err).Msg("Database not available - games will not be persisted")
	}
	if db != nil {
		defer db.Close()
	}

	// Initialize game manager
	gameManager := chess.NewGameManager(log.Logger)

	// Initialize WebSocket hub
	hub = server.NewHub(gameManager, log.Logger)

	// Start hub in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Create router
	router := setupRouter()

	// Create server
	srv := &http.Server{
		Addr:         ":" + getEnv("CHESS_PORT", "8081"),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("Starting Chess WebSocket server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start chess server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Chess server shutting down...")
	cancel() // Stop the hub

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func setupRouter() *gin.Engine {
	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "openpairing-chess",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// WebSocket endpoint for games
	router.GET("/ws/game/:id", handleGameWebSocket)

	// WebSocket endpoint for spectating
	router.GET("/ws/spectate/:id", handleSpectateWebSocket)

	return router
}

func handleGameWebSocket(c *gin.Context) {
	gameIDStr := c.Param("id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	// Get user ID from query param or header (in production, extract from JWT)
	userIDStr := c.Query("userId")
	if userIDStr == "" {
		userIDStr = c.GetHeader("X-User-ID")
	}
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
		return
	}

	// Create client
	client := &server.Client{
		ID:          uuid.New(),
		UserID:      userID,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		Hub:         hub,
		GameID:      &gameID,
		IsSpectator: false,
	}

	// Register client with hub
	hub.RegisterClient(client)

	// Start read and write pumps
	go client.WritePump()
	go client.ReadPump()

	log.Info().
		Str("game_id", gameID.String()).
		Str("user_id", userID.String()).
		Msg("Player connected to game")
}

func handleSpectateWebSocket(c *gin.Context) {
	gameIDStr := c.Param("id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
		return
	}

	// Create spectator client (no user ID required)
	client := &server.Client{
		ID:          uuid.New(),
		UserID:      uuid.Nil,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		Hub:         hub,
		GameID:      &gameID,
		IsSpectator: true,
	}

	// Register client with hub
	hub.RegisterClient(client)

	// Start read and write pumps
	go client.WritePump()
	go client.ReadPump()

	log.Info().
		Str("game_id", gameID.String()).
		Msg("Spectator connected to game")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
