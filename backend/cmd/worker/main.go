package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openpairing/openpairing/internal/config"
	"github.com/openpairing/openpairing/internal/database"
	"github.com/openpairing/openpairing/internal/worker"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENVIRONMENT") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Msg("Starting OpenPairing Worker Service")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	log.Info().Msg("Connected to database")

	// Create scheduler
	scheduler := worker.NewScheduler(log.Logger)

	// Register jobs
	jobRegistry := worker.NewJobRegistry(db.Pool, log.Logger)
	jobRegistry.RegisterAllJobs(scheduler)

	// Register additional custom jobs
	registerCustomJobs(scheduler, db.Pool, log.Logger)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start scheduler
	scheduler.Start(ctx)

	// Log registered jobs
	jobs := scheduler.GetJobs()
	for _, job := range jobs {
		log.Info().
			Str("id", job.ID).
			Str("name", job.Name).
			Dur("interval", job.Interval).
			Msg("Job registered")
	}

	log.Info().Int("job_count", len(jobs)).Msg("Worker service started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Worker shutting down...")

	// Stop scheduler gracefully
	scheduler.Stop()

	// Give running jobs time to complete
	time.Sleep(2 * time.Second)

	log.Info().Msg("Worker shutdown complete")
}

func registerCustomJobs(scheduler *worker.Scheduler, db interface{}, logger zerolog.Logger) {
	// Heartbeat job for monitoring
	scheduler.RegisterJob(
		"heartbeat",
		"Worker heartbeat for monitoring",
		30*time.Second,
		func(ctx context.Context) error {
			logger.Debug().Msg("Worker heartbeat")
			return nil
		},
	)

	// Tournament reminder job
	scheduler.RegisterJob(
		"tournament_reminders",
		"Send reminders for upcoming tournament rounds",
		15*time.Minute,
		func(ctx context.Context) error {
			// This would query for upcoming rounds and send reminders
			logger.Debug().Msg("Checking for tournament reminders")
			return nil
		},
	)

	// Lineup deadline reminder job (for team tournaments)
	scheduler.RegisterJob(
		"lineup_deadline_reminders",
		"Send reminders for upcoming lineup submission deadlines",
		15*time.Minute,
		func(ctx context.Context) error {
			// This would query for upcoming lineup deadlines and send reminders
			logger.Debug().Msg("Checking for lineup deadline reminders")
			return nil
		},
	)

	// Game starting notifications job
	scheduler.RegisterJob(
		"game_starting_notifications",
		"Send notifications for games starting soon",
		5*time.Minute,
		func(ctx context.Context) error {
			// This would query for games starting in the next 15 minutes
			logger.Debug().Msg("Checking for games starting soon")
			return nil
		},
	)
}
