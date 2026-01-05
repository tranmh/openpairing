package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openpairing/openpairing/internal/analytics"
	"github.com/openpairing/openpairing/internal/auth"
	"github.com/openpairing/openpairing/internal/config"
	"github.com/openpairing/openpairing/internal/database"
	"github.com/openpairing/openpairing/internal/federation"
	"github.com/openpairing/openpairing/internal/federation/dsb"
	"github.com/openpairing/openpairing/internal/federation/fide"
	"github.com/openpairing/openpairing/internal/federation/oesb"
	"github.com/openpairing/openpairing/internal/federation/ssb"
	"github.com/openpairing/openpairing/internal/middleware"
	"github.com/openpairing/openpairing/internal/arbiter"
	"github.com/openpairing/openpairing/internal/notification"
	"github.com/openpairing/openpairing/internal/pairing/engine"
	"github.com/openpairing/openpairing/internal/player"
	"github.com/openpairing/openpairing/internal/reports"
	"github.com/openpairing/openpairing/internal/tournament"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if cfg.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Connect to database
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize services
	authService := auth.NewService(db.Pool, &cfg.JWT)
	authHandler := auth.NewHandler(authService)
	authMiddleware := auth.NewMiddleware(auth.NewJWTService(&cfg.JWT))

	// Player service
	playerRepo := player.NewRepository(db.Pool)
	playerService := player.NewService(playerRepo, log.Logger)
	playerHandler := player.NewHandler(playerService)

	// Federation services
	fideImporter := fide.NewImporter(playerRepo, &fide.ImporterConfig{
		CacheDir:   cfg.Federation.CacheDir,
		BatchSize:  500,
		Workers:    4,
		FilterDACH: false,
		MinRating:  0,
	}, log.Logger)
	fideScheduler := fide.NewScheduler(fideImporter, cfg.Federation.FideSyncInterval, log.Logger)

	dsbImporter := dsb.NewImporter(playerRepo, &dsb.ImporterConfig{
		BatchSize: 100,
	}, log.Logger)
	dsbScheduler := dsb.NewScheduler(dsbImporter, cfg.Federation.DSBSyncInterval, log.Logger)

	oesbImporter := oesb.NewImporter(playerRepo, &oesb.ImporterConfig{
		BatchSize: 100,
		Workers:   4,
	}, log.Logger)
	oesbScheduler := oesb.NewScheduler(oesbImporter, cfg.Federation.OESBSyncInterval, log.Logger)

	ssbImporter := ssb.NewImporter(playerRepo, &ssb.ImporterConfig{
		BatchSize: 100,
		Workers:   4,
	}, log.Logger)
	ssbScheduler := ssb.NewScheduler(ssbImporter, cfg.Federation.SSBSyncInterval, log.Logger)

	federationHandler := federation.NewHandler(&federation.HandlerConfig{
		FideImporter:  fideImporter,
		FideScheduler: fideScheduler,
		DSBImporter:   dsbImporter,
		DSBScheduler:  dsbScheduler,
		OESBImporter:  oesbImporter,
		OESBScheduler: oesbScheduler,
		SSBImporter:   ssbImporter,
		SSBScheduler:  ssbScheduler,
		PlayerRepo:    playerRepo,
	}, log.Logger)

	// Analytics service
	analyticsService := analytics.NewService(db.Pool, log.Logger)
	analyticsHandler := analytics.NewHandler(analyticsService, log.Logger)

	// Notification service
	notificationRepo := notification.NewRepository(db.Pool)
	notificationService := notification.NewService(notificationRepo, log.Logger)
	notificationHandler := notification.NewHandler(notificationService, log.Logger)

	// Tournament service
	tournamentRepo := tournament.NewRepository(db.Pool)
	// Initialize pairing engine - try BBPPairings first, then JaVaFo
	var pairingEngine engine.PairingEngine
	if cfg.Pairing.BBPPath != "" {
		bbp, err := engine.NewBBPPairings(&engine.BBPConfig{
			ExecPath: cfg.Pairing.BBPPath,
			Timeout:  cfg.Pairing.Timeout,
		}, log.Logger)
		if err != nil {
			log.Warn().Err(err).Msg("bbpPairings not available")
		} else {
			pairingEngine = bbp
			log.Info().Msg("Using bbpPairings engine")
		}
	}
	if pairingEngine == nil && cfg.Pairing.JaVaFoPath != "" {
		javafo, err := engine.NewJaVaFo(&engine.JaVaFoConfig{
			JarPath: cfg.Pairing.JaVaFoPath,
			Timeout: cfg.Pairing.Timeout,
		}, log.Logger)
		if err != nil {
			log.Warn().Err(err).Msg("JaVaFo not available")
		} else {
			pairingEngine = javafo
			log.Info().Msg("Using JaVaFo engine")
		}
	}
	if pairingEngine == nil {
		log.Warn().Msg("No pairing engine available - pairing generation will be disabled")
	}
	tournamentService := tournament.NewService(tournamentRepo, pairingEngine, log.Logger)
	tournamentHandler := tournament.NewHandler(tournamentService, log.Logger)

	// Arbiter service
	arbiterService := arbiter.NewService(tournamentRepo, nil, log.Logger)
	arbiterHandler := arbiter.NewHandler(arbiterService, log.Logger)

	// Reports service
	reportsDataProvider := reports.NewPostgresDataProvider(db.Pool)
	dwzReportService := reports.NewDWZReportService(reportsDataProvider)
	trfReportService := reports.NewTRFReportService(reportsDataProvider)
	normReportService := reports.NewNormReportService(reportsDataProvider)
	reportsHandler := reports.NewHandler(dwzReportService, normReportService, trfReportService, log.Logger)

	// Configure rate limiting
	middleware.RateLimitEnabled = cfg.RateLimit.Enabled
	if !cfg.RateLimit.Enabled {
		log.Warn().Msg("Rate limiting is DISABLED")
	}

	// Create Gin router
	router := setupRouter(cfg, authHandler, authMiddleware, playerHandler, federationHandler, analyticsHandler, notificationHandler, tournamentHandler, arbiterHandler, reportsHandler)

	// Create server
	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start federation schedulers if enabled
	ctx, cancelSchedulers := context.WithCancel(context.Background())
	if cfg.Federation.EnableFIDESync {
		go fideScheduler.Start(ctx)
	}
	if cfg.Federation.EnableDSBSync {
		go dsbScheduler.Start(ctx)
	}
	if cfg.Federation.EnableOESBSync {
		go oesbScheduler.Start(ctx)
	}
	if cfg.Federation.EnableSSBSync {
		go ssbScheduler.Start(ctx)
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("Starting API server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	// Cancel federation schedulers
	cancelSchedulers()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

func setupRouter(cfg *config.Config, authHandler *auth.Handler, authMiddleware *auth.Middleware, playerHandler *player.Handler, federationHandler *federation.Handler, analyticsHandler *analytics.Handler, notificationHandler *notification.Handler, tournamentHandler *tournament.Handler, arbiterHandler *arbiter.Handler, reportsHandler *reports.Handler) *gin.Engine {
	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(&cfg.CORS))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "openpairing-api",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes - with rate limiting
		authRoutes := v1.Group("/auth")
		authRoutes.Use(middleware.AuthRateLimit())
		{
			authRoutes.POST("/register", authHandler.Register)
			authRoutes.POST("/login", authHandler.Login)
			authRoutes.POST("/logout", authHandler.Logout)
			authRoutes.POST("/refresh", authHandler.RefreshToken)
			authRoutes.POST("/forgot-password", authHandler.ForgotPassword)
			authRoutes.POST("/reset-password", authHandler.ResetPassword)
			authRoutes.GET("/me", authMiddleware.Authenticate(), authHandler.GetCurrentUser)
		}

		// Player routes
		playerHandler.RegisterRoutes(v1, authMiddleware.Authenticate(), authMiddleware.RequireAdmin())

		// Federation routes
		federationHandler.RegisterRoutes(v1, authMiddleware.Authenticate(), authMiddleware.RequireAdmin())

		// Analytics routes
		analyticsHandler.RegisterRoutes(v1)

		// Notification routes
		notificationHandler.RegisterRoutes(v1)

		// Tournament routes - use the tournament handler's RegisterRoutes
		tournamentHandler.RegisterRoutes(v1, authMiddleware.Authenticate())

		// Arbiter routes
		arbiterHandler.RegisterRoutes(v1, authMiddleware.Authenticate(), authMiddleware.RequireArbiter())

		// Reports routes
		reportsHandler.RegisterRoutes(v1)

		// Organization routes
		orgs := v1.Group("/organizations")
		orgs.Use(middleware.APIRateLimit())
		{
			orgs.GET("", notImplemented)
			orgs.POST("", authMiddleware.Authenticate(), notImplemented)
			orgs.GET("/:id", notImplemented)
			orgs.PUT("/:id", authMiddleware.Authenticate(), notImplemented)
			orgs.GET("/:id/tournaments", notImplemented)
		}

		// Team routes
		teams := v1.Group("/teams")
		teams.Use(middleware.APIRateLimit())
		{
			teams.GET("", notImplemented)
			teams.POST("", authMiddleware.Authenticate(), notImplemented)
			teams.GET("/:id", notImplemented)
			teams.PUT("/:id", authMiddleware.Authenticate(), notImplemented)
			teams.GET("/:id/members", notImplemented)
			teams.POST("/:id/members", authMiddleware.Authenticate(), notImplemented)
			teams.DELETE("/:id/members/:pid", authMiddleware.Authenticate(), notImplemented)
		}

		// Club routes
		clubs := v1.Group("/clubs")
		clubs.Use(middleware.APIRateLimit())
		{
			clubs.GET("/:id/teams", notImplemented)
		}

		// Online games
		games := v1.Group("/games")
		games.Use(middleware.APIRateLimit())
		{
			games.GET("/:id", notImplemented)
			games.POST("/:id/resign", authMiddleware.Authenticate(), notImplemented)
			games.POST("/:id/draw-offer", authMiddleware.Authenticate(), notImplemented)
			games.POST("/:id/abort", authMiddleware.Authenticate(), notImplemented)
			games.POST("/:id/call-arbiter", authMiddleware.Authenticate(), notImplemented)
		}
	}

	return router
}

func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not_implemented",
		"message": "This endpoint is not yet implemented",
	})
}
