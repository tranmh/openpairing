package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// CLI flags
	dbURL := flag.String("db-url", "", "Database connection URL (required)")
	force := flag.Bool("force", false, "Clear existing seed data before seeding")
	verbose := flag.Bool("verbose", false, "Show detailed progress")
	skipProgression := flag.Bool("skip-progression", false, "Skip generating game results (faster)")
	flag.Parse()

	// Validate required flags
	if *dbURL == "" {
		// Try environment variable
		*dbURL = os.Getenv("DATABASE_URL")
		if *dbURL == "" {
			fmt.Fprintln(os.Stderr, "Error: --db-url is required or set DATABASE_URL environment variable")
			flag.Usage()
			os.Exit(1)
		}
	}

	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	if !*verbose {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}
	log.Info().Msg("Connected to database")

	// Create seeder
	seeder := NewSeeder(pool, *verbose)

	// Force cleanup if requested
	if *force {
		log.Warn().Msg("Force mode: clearing existing seed data...")
		if err := seeder.ClearSeedData(ctx); err != nil {
			log.Fatal().Err(err).Msg("Failed to clear seed data")
		}
	}

	// Run seeding
	startTime := time.Now()

	if err := seeder.Run(ctx, *skipProgression); err != nil {
		log.Fatal().Err(err).Msg("Seeding failed")
	}

	duration := time.Since(startTime)
	log.Info().
		Dur("duration", duration).
		Msg("Seeding completed successfully")
}
