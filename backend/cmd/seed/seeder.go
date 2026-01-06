package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Seeder orchestrates the seeding process
type Seeder struct {
	pool        *pgxpool.Pool
	verbose     bool
	userSeeder  *UserSeeder
	playerSeeder *PlayerSeeder
	tournamentSeeder *TournamentSeeder
	progressionSeeder *ProgressionSeeder
}

// NewSeeder creates a new seeder
func NewSeeder(pool *pgxpool.Pool, verbose bool) *Seeder {
	seed := time.Now().UnixNano()

	return &Seeder{
		pool:        pool,
		verbose:     verbose,
		userSeeder:  NewUserSeeder(pool),
		playerSeeder: NewPlayerSeeder(pool, seed),
		tournamentSeeder: NewTournamentSeeder(pool),
		progressionSeeder: NewProgressionSeeder(pool, seed+100),
	}
}

// Run executes the complete seeding process
func (s *Seeder) Run(ctx context.Context, skipProgression bool) error {
	var stats SeedingStats

	// 1. Seed admin user
	log.Info().Msg("Step 1/4: Seeding admin user...")
	adminConfig := DefaultAdminConfig()
	adminID, err := s.userSeeder.SeedAdmin(ctx, adminConfig)
	if err != nil {
		return err
	}
	stats.AdminID = adminID.String()
	stats.AdminEmail = adminConfig.Email

	// 2. Seed test users
	log.Info().Msg("Step 2/4: Seeding test users...")
	testUsers, err := s.userSeeder.SeedTestUsers(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to seed test users (continuing)")
	}
	stats.TestUsers = len(testUsers)

	// 3. Seed players
	log.Info().Msg("Step 3/4: Seeding player pool...")
	playerCount := 550 // Enough for all tournaments with some overlap
	players, err := s.playerSeeder.SeedPlayers(ctx, playerCount)
	if err != nil {
		return err
	}
	stats.Players = len(players)

	// 4. Seed tournaments
	log.Info().Msg("Step 4/4: Seeding tournaments...")
	scenarios := GetTournamentScenarios()

	for _, scenario := range scenarios {
		// Get players for this tournament
		playerIDs := s.playerSeeder.GetPlayersForTournament(
			scenario.Size,
			scenario.MinRating,
			scenario.MaxRating,
		)

		if len(playerIDs) < scenario.Size {
			log.Warn().
				Str("tournament", scenario.Name).
				Int("requested", scenario.Size).
				Int("available", len(playerIDs)).
				Msg("Not enough players, using available")
		}

		// Create tournament
		tournament, err := s.tournamentSeeder.SeedTournament(ctx, scenario, playerIDs)
		if err != nil {
			log.Error().Err(err).Str("tournament", scenario.Name).Msg("Failed to create tournament")
			continue
		}

		stats.Tournaments++
		switch scenario.Status {
		case "registration":
			stats.TournamentsRegistration++
		case "ongoing":
			stats.TournamentsOngoing++
		case "completed":
			stats.TournamentsCompleted++
		}

		// Generate progression (rounds/games) if not skipped
		if !skipProgression && scenario.Status != "registration" {
			if err := s.progressionSeeder.SeedProgression(ctx, tournament); err != nil {
				log.Error().Err(err).Str("tournament", scenario.Name).Msg("Failed to generate progression")
				continue
			}
		}
	}

	// Print summary
	stats.Print()

	return nil
}

// ClearSeedData removes all seeded data for force cleanup
func (s *Seeder) ClearSeedData(ctx context.Context) error {
	log.Info().Msg("Clearing seeded data...")

	// Order matters due to foreign keys

	// 1. Clear games (FK to rounds and participants)
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM games WHERE tournament_id IN (
			SELECT id FROM tournaments WHERE description LIKE 'Seeded tournament:%'
		)
	`); err != nil {
		log.Warn().Err(err).Msg("Failed to clear games")
	}

	// 2. Clear tournament rounds
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM tournament_rounds WHERE tournament_id IN (
			SELECT id FROM tournaments WHERE description LIKE 'Seeded tournament:%'
		)
	`); err != nil {
		log.Warn().Err(err).Msg("Failed to clear tournament rounds")
	}

	// 3. Clear tournament participants
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM tournament_participants WHERE tournament_id IN (
			SELECT id FROM tournaments WHERE description LIKE 'Seeded tournament:%'
		)
	`); err != nil {
		log.Warn().Err(err).Msg("Failed to clear tournament participants")
	}

	// 4. Clear tournaments
	if err := s.tournamentSeeder.ClearTournaments(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to clear tournaments")
	}

	// 5. Clear players
	if err := s.playerSeeder.ClearPlayers(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to clear players")
	}

	// 6. Clear users
	if err := s.userSeeder.ClearUsers(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to clear users")
	}

	log.Info().Msg("Seeded data cleared")
	return nil
}

// SeedingStats holds statistics about the seeding process
type SeedingStats struct {
	AdminID                string
	AdminEmail             string
	TestUsers              int
	Players                int
	Tournaments            int
	TournamentsRegistration int
	TournamentsOngoing     int
	TournamentsCompleted   int
}

// Print outputs the seeding statistics
func (s *SeedingStats) Print() {
	log.Info().Msg("")
	log.Info().Msg("========================================")
	log.Info().Msg("       Seeding Completed!")
	log.Info().Msg("========================================")
	log.Info().Msg("")
	log.Info().Str("email", s.AdminEmail).Str("password", "admin123").Msg("Admin user")
	log.Info().Int("count", s.TestUsers).Msg("Test users (organizer, arbiter, player)")
	log.Info().Int("count", s.Players).Msg("Players seeded")
	log.Info().Msg("")
	log.Info().Int("total", s.Tournaments).Msg("Tournaments")
	log.Info().Int("count", s.TournamentsRegistration).Msg("  - Registration")
	log.Info().Int("count", s.TournamentsOngoing).Msg("  - Ongoing")
	log.Info().Int("count", s.TournamentsCompleted).Msg("  - Completed")
	log.Info().Msg("")
}
