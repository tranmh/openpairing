package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENVIRONMENT") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

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
	// TODO: Implement WebSocket upgrade and game handling
	// 1. Authenticate user from JWT
	// 2. Upgrade to WebSocket
	// 3. Register connection with game hub
	// 4. Handle game messages (moves, draw offers, etc.)
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "WebSocket game handler not yet implemented",
	})
}

func handleSpectateWebSocket(c *gin.Context) {
	// TODO: Implement WebSocket upgrade for spectating
	// 1. Upgrade to WebSocket
	// 2. Subscribe to game events
	// 3. Broadcast moves to spectators
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "WebSocket spectate handler not yet implemented",
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
