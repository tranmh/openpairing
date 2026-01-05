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
	"github.com/openpairing/openpairing/internal/arbiter"
	"github.com/openpairing/openpairing/internal/auth"
	"github.com/openpairing/openpairing/internal/config"
	"github.com/openpairing/openpairing/internal/database"
	"github.com/openpairing/openpairing/internal/email"
	"github.com/openpairing/openpairing/internal/export"
	"github.com/openpairing/openpairing/internal/federation"
	"github.com/openpairing/openpairing/internal/federation/dsb"
	"github.com/openpairing/openpairing/internal/federation/fide"
	"github.com/openpairing/openpairing/internal/federation/oesb"
	"github.com/openpairing/openpairing/internal/federation/portal64"
	"github.com/openpairing/openpairing/internal/federation/ssb"
	"github.com/openpairing/openpairing/internal/fairplay"
	"github.com/openpairing/openpairing/internal/gameaction"
	"github.com/openpairing/openpairing/internal/middleware"
	"github.com/openpairing/openpairing/internal/notification"
	"github.com/openpairing/openpairing/internal/organization"
	"github.com/openpairing/openpairing/internal/pairing/engine"
	"github.com/openpairing/openpairing/internal/player"
	"github.com/openpairing/openpairing/internal/registration"
	"github.com/openpairing/openpairing/internal/reports"
	"github.com/openpairing/openpairing/internal/result"
	"github.com/openpairing/openpairing/internal/standings"
	"github.com/openpairing/openpairing/internal/team"
	"github.com/openpairing/openpairing/internal/team_tournament"
	"github.com/openpairing/openpairing/internal/tournament"
	"github.com/openpairing/openpairing/internal/user"
	"github.com/openpairing/openpairing/internal/waitlist"
	"github.com/openpairing/openpairing/internal/wam"
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

	// Initialize email service
	emailConfig := &email.Config{
		SMTPHost:     cfg.Email.SMTPHost,
		SMTPPort:     cfg.Email.SMTPPort,
		SMTPUsername: cfg.Email.SMTPUsername,
		SMTPPassword: cfg.Email.SMTPPassword,
		FromEmail:    cfg.Email.FromEmail,
		FromName:     cfg.Email.FromName,
		UseTLS:       cfg.Email.UseTLS,
		SkipVerify:   cfg.Email.SkipVerify,
	}
	emailService := email.NewService(emailConfig, log.Logger)

	// Initialize services
	authService := auth.NewService(db.Pool, &cfg.JWT, emailService, cfg.App.BaseURL)
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

	// Portal64 service (alternative DSB data source)
	portal64Service := portal64.NewService(playerRepo, &portal64.ServiceConfig{
		BaseURL:        cfg.Federation.Portal64BaseURL,
		Timeout:        cfg.Federation.Portal64Timeout,
		PlayerCacheTTL: cfg.Federation.Portal64CacheTTL,
		SearchCacheTTL: cfg.Federation.Portal64SearchTTL,
	}, log.Logger)

	// Log which DSB provider is active
	log.Info().
		Str("dsb_provider", cfg.Federation.DSBProvider).
		Str("portal64_url", cfg.Federation.Portal64BaseURL).
		Msg("DSB data source configured")

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
		FideImporter:    fideImporter,
		FideScheduler:   fideScheduler,
		DSBImporter:     dsbImporter,
		DSBScheduler:    dsbScheduler,
		Portal64Service: portal64Service,
		DSBProvider:     cfg.Federation.DSBProvider,
		OESBImporter:    oesbImporter,
		OESBScheduler:   oesbScheduler,
		SSBImporter:     ssbImporter,
		SSBScheduler:    ssbScheduler,
		PlayerRepo:      playerRepo,
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

	// Initialize pairing engine registry with all available engines
	engineRegistry := engine.NewRegistry(&engine.RegistryConfig{
		JaVaFoPath: cfg.Pairing.JaVaFoPath,
		BBPPath:    cfg.Pairing.BBPPath,
	}, log.Logger)

	// Log available engines
	availableEngines := engineRegistry.ListAvailableEngines()
	log.Info().
		Strs("engines", availableEngines).
		Bool("has_swiss", engineRegistry.HasSwissEngine()).
		Msg("Pairing engine registry initialized")

	if !engineRegistry.HasSwissEngine() {
		log.Warn().Msg("No Swiss pairing engine (JaVaFo/BBP) available - Swiss tournaments will use Monrad fallback")
	}

	tournamentService := tournament.NewServiceWithRegistry(tournamentRepo, engineRegistry, log.Logger)
	tournamentHandler := tournament.NewHandlerWithImporter(tournamentService, portal64Service, playerRepo, log.Logger)

	// Arbiter service
	arbiterRepo := arbiter.NewRepository(db.Pool)
	arbiterService := arbiter.NewService(tournamentRepo, arbiterRepo, log.Logger)
	arbiterHandler := arbiter.NewHandler(arbiterService, log.Logger)

	// Start arbiter monitor hub for live tournament monitoring
	go arbiterHandler.GetMonitorHub().Start(context.Background())
	defer arbiterHandler.GetMonitorHub().Stop()

	// Reports service
	reportsDataProvider := reports.NewPostgresDataProvider(db.Pool)
	dwzReportService := reports.NewDWZReportService(reportsDataProvider)
	trfReportService := reports.NewTRFReportService(reportsDataProvider)
	normReportService := reports.NewNormReportService(reportsDataProvider)
	reportsHandler := reports.NewHandler(dwzReportService, normReportService, trfReportService, log.Logger)

	// Export service
	exportRepo := export.NewRepository(db.Pool)
	exportService := export.NewService(exportRepo)
	exportHandler := export.NewHandler(exportService)

	// Standings service
	standingsRepo := standings.NewRepository(db.Pool)
	standingsService := standings.NewService(standingsRepo)
	standingsHandler := standings.NewHandler(standingsService)

	// Result service
	resultRepo := result.NewRepository(db.Pool)
	resultService := result.NewService(resultRepo)
	resultHandler := result.NewHandler(resultService)

	// Organization service
	organizationRepo := organization.NewRepository(db.Pool)
	organizationService := organization.NewService(organizationRepo, log.Logger)
	organizationHandler := organization.NewHandler(organizationService, log.Logger)

	// Team service
	teamRepo := team.NewRepository(db.Pool)
	teamService := team.NewService(teamRepo, log.Logger)
	teamHandler := team.NewHandler(teamService, log.Logger)

	// Team Tournament service
	teamTournamentRepo := team_tournament.NewRepository(db.Pool)
	teamTournamentService := team_tournament.NewService(teamTournamentRepo, tournamentRepo, teamRepo, log.Logger)
	teamTournamentHandler := team_tournament.NewHandler(teamTournamentService, log.Logger)

	// User service
	userRepo := user.NewRepository(db.Pool)
	userService := user.NewService(userRepo, log.Logger)
	userHandler := user.NewHandler(userService, log.Logger)

	// Game action service (resign, draw, abort, arbiter call)
	gameActionRepo := gameaction.NewRepository(db.Pool)
	gameActionService := gameaction.NewService(gameActionRepo, log.Logger)
	gameActionHandler := gameaction.NewHandler(gameActionService, log.Logger)

	// Fairplay service
	fairplayRepo := fairplay.NewRepository(db.Pool)
	fairplayService := fairplay.NewService(fairplayRepo, log.Logger)
	fairplayHandler := fairplay.NewHandler(fairplayService, log.Logger)

	// Waitlist service
	waitlistRepo := waitlist.NewRepository(db.Pool)
	waitlistService := waitlist.NewService(waitlistRepo, emailService, cfg.App.BaseURL, log.Logger)
	waitlistHandler := waitlist.NewHandler(waitlistService, log.Logger)

	// Registration service (self-service pre-registration and check-in)
	registrationRepo := registration.NewRepository(db.Pool)
	registrationService := registration.NewService(registrationRepo, cfg.App.BaseURL, log.Logger)
	registrationHandler := registration.NewHandler(registrationService, log.Logger)

	// WAM (Württembergische Amateurmeisterschaft) service
	wamRepo := wam.NewRepository(db.Pool)
	wamService := wam.NewService(wamRepo, log.Logger)
	wamHandler := wam.NewHandler(wamService, log.Logger)

	// Configure rate limiting
	middleware.RateLimitEnabled = cfg.RateLimit.Enabled
	if !cfg.RateLimit.Enabled {
		log.Warn().Msg("Rate limiting is DISABLED")
	}

	// Create Gin router
	router := setupRouter(cfg, authHandler, authMiddleware, playerHandler, federationHandler, analyticsHandler, notificationHandler, tournamentHandler, arbiterHandler, reportsHandler, exportHandler, standingsHandler, resultHandler, organizationHandler, teamHandler, userHandler, gameActionHandler, fairplayHandler, teamTournamentHandler, waitlistHandler, registrationHandler, wamHandler)

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

func setupRouter(cfg *config.Config, authHandler *auth.Handler, authMiddleware *auth.Middleware, playerHandler *player.Handler, federationHandler *federation.Handler, analyticsHandler *analytics.Handler, notificationHandler *notification.Handler, tournamentHandler *tournament.Handler, arbiterHandler *arbiter.Handler, reportsHandler *reports.Handler, exportHandler *export.Handler, standingsHandler *standings.Handler, resultHandler *result.Handler, organizationHandler *organization.Handler, teamHandler *team.Handler, userHandler *user.Handler, gameActionHandler *gameaction.Handler, fairplayHandler *fairplay.Handler, teamTournamentHandler *team_tournament.Handler, waitlistHandler *waitlist.Handler, registrationHandler *registration.Handler, wamHandler *wam.Handler) *gin.Engine {
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
			authRoutes.POST("/verify-email", authHandler.VerifyEmail)
			authRoutes.GET("/me", authMiddleware.Authenticate(), authHandler.GetCurrentUser)
			authRoutes.POST("/resend-verification", authMiddleware.Authenticate(), authHandler.ResendVerificationEmail)
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
		tournamentHandler.RegisterRoutes(v1, authMiddleware.Authenticate(), authMiddleware.RequireOrganizer())

		// Arbiter routes
		arbiterHandler.RegisterRoutes(v1, authMiddleware.Authenticate(), authMiddleware.RequireArbiter())

		// Reports routes
		reportsHandler.RegisterRoutes(v1)

		// Organization routes
		organizationHandler.RegisterRoutes(v1, authMiddleware.Authenticate())

		// Team routes (includes club teams)
		teamHandler.RegisterRoutes(v1, authMiddleware.Authenticate())

		// User profile routes
		userHandler.RegisterRoutes(v1, authMiddleware.Authenticate())

		// Fairplay routes
		fairplayHandler.RegisterRoutes(v1, authMiddleware.Authenticate(), authMiddleware.RequireArbiter())

		// Team Tournament routes
		teamTournamentHandler.RegisterRoutes(v1, authMiddleware.Authenticate(), authMiddleware.RequireOrganizer())

		// Waitlist routes
		waitlistHandler.RegisterRoutes(v1, authMiddleware.Authenticate(), authMiddleware.RequireOrganizer())

		// WAM (Württembergische Amateurmeisterschaft) routes
		wamHandler.RegisterRoutes(v1, authMiddleware.Authenticate())

		// Registration routes (self-service pre-registration and check-in)
		registrationHandler.RegisterRoutes(router, authMiddleware.Authenticate(), authMiddleware.OptionalAuth(), authMiddleware.RequireOrganizer())

		// Online games
		games := v1.Group("/games")
		games.Use(middleware.APIRateLimit())
		{
			games.GET("/:id", resultHandler.GetGame)
			games.GET("/:id/audit", resultHandler.GetResultAuditLog)
			games.GET("/:id/export/pgn", exportHandler.ExportGamePGN)
			games.GET("/:id/draw-offer", gameActionHandler.GetDrawOfferStatus)
			games.POST("/:id/resign", authMiddleware.Authenticate(), gameActionHandler.Resign)
			games.POST("/:id/draw-offer", authMiddleware.Authenticate(), gameActionHandler.DrawOffer)
			games.POST("/:id/abort", authMiddleware.Authenticate(), gameActionHandler.Abort)
			games.POST("/:id/call-arbiter", authMiddleware.Authenticate(), gameActionHandler.CallArbiter)
		}

		// Result management routes
		results := v1.Group("/results")
		results.Use(middleware.APIRateLimit())
		{
			results.POST("", authMiddleware.Authenticate(), resultHandler.EnterResult)
			results.POST("/correct", authMiddleware.Authenticate(), resultHandler.CorrectResult)
			results.POST("/bulk", authMiddleware.Authenticate(), resultHandler.EnterBulkResults)
			results.POST("/validate", resultHandler.ValidateResult)
			results.DELETE("/:id", authMiddleware.Authenticate(), resultHandler.DeleteResult)
		}

		// Export routes (additional tournament exports)
		exports := v1.Group("/tournaments/:id/export")
		exports.Use(middleware.APIRateLimit())
		{
			exports.GET("/pgn", exportHandler.ExportTournamentPGN)
			exports.GET("/json", exportHandler.ExportTournamentJSON)
			exports.GET("/standings", exportHandler.ExportStandingsCSV)
			exports.GET("/crosstable", exportHandler.ExportCrosstableCSV)
		}

		// Certificate routes
		v1.GET("/tournaments/:id/certificates", middleware.APIRateLimit(), exportHandler.ExportTopCertificates)
		v1.GET("/tournaments/:id/participants/:participantId/certificate", middleware.APIRateLimit(), exportHandler.ExportParticipantCertificate)

		// Round exports
		v1.GET("/tournaments/:id/rounds/:round/export/pgn", middleware.APIRateLimit(), exportHandler.ExportRoundPGN)

		// Standings routes
		v1.GET("/tournaments/:id/standings", middleware.APIRateLimit(), standingsHandler.GetStandings)
		v1.GET("/tournaments/:id/crosstable", middleware.APIRateLimit(), standingsHandler.GetCrosstable)
		v1.POST("/tournaments/:id/standings/recalculate", middleware.APIRateLimit(), authMiddleware.Authenticate(), standingsHandler.RecalculateStandings)
		v1.GET("/tournaments/:id/participants/:participantId/progress", middleware.APIRateLimit(), standingsHandler.GetPlayerProgress)

		// Round games and status
		v1.GET("/tournaments/:id/rounds/:round/games", middleware.APIRateLimit(), resultHandler.GetRoundGames)
		v1.GET("/tournaments/:id/rounds/:round/status", middleware.APIRateLimit(), resultHandler.GetRoundStatus)

		// Tiebreaks reference
		v1.GET("/tiebreaks", middleware.APIRateLimit(), standingsHandler.GetAvailableTiebreaks)
	}

	return router
}

func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not_implemented",
		"message": "This endpoint is not yet implemented",
	})
}
