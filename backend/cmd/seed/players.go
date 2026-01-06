package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// PlayerSeeder handles player seeding
type PlayerSeeder struct {
	pool          *pgxpool.Pool
	nameGen       *NameGenerator
	ratingGen     *RatingGenerator
	createdIDs    []uuid.UUID
	playersByRating map[int][]uuid.UUID // Rating bucket -> player IDs
}

// NewPlayerSeeder creates a new player seeder
func NewPlayerSeeder(pool *pgxpool.Pool, seed int64) *PlayerSeeder {
	return &PlayerSeeder{
		pool:          pool,
		nameGen:       NewNameGenerator(seed),
		ratingGen:     NewRatingGenerator(seed + 1),
		createdIDs:    make([]uuid.UUID, 0),
		playersByRating: make(map[int][]uuid.UUID),
	}
}

// SeededPlayer represents a seeded player with their data
type SeededPlayer struct {
	ID           uuid.UUID
	FirstName    string
	LastName     string
	Gender       string
	Federation   string
	FideRating   int
	DWZ          int
	FideTitle    *string
	BirthYear    int
	FideID       string
	FederationID string
}

// SeedPlayers creates the player pool
func (s *PlayerSeeder) SeedPlayers(ctx context.Context, count int) ([]SeededPlayer, error) {
	log.Info().Int("count", count).Msg("Seeding players...")

	players := make([]SeededPlayer, 0, count)
	batchSize := 50
	batch := make([]SeededPlayer, 0, batchSize)

	for i := 0; i < count; i++ {
		player := s.generatePlayer(i)
		batch = append(batch, player)

		if len(batch) >= batchSize || i == count-1 {
			if err := s.insertPlayerBatch(ctx, batch); err != nil {
				return nil, fmt.Errorf("failed to insert player batch: %w", err)
			}
			players = append(players, batch...)
			batch = batch[:0]

			if (i+1)%100 == 0 || i == count-1 {
				log.Debug().Int("progress", i+1).Int("total", count).Msg("Players seeded")
			}
		}
	}

	log.Info().Int("count", len(players)).Msg("Players seeded successfully")
	return players, nil
}

func (s *PlayerSeeder) generatePlayer(index int) SeededPlayer {
	federation := s.nameGen.GenerateFederation()
	gender := s.nameGen.GenerateGender()
	firstName, lastName := s.nameGen.GenerateName(federation, gender)

	fideRating := s.ratingGen.GenerateRating()
	dwz := s.ratingGen.GenerateDWZ(fideRating)
	title := s.ratingGen.AssignWomenTitle(fideRating, gender)
	birthYear := s.nameGen.GenerateBirthYear()

	fideID := s.ratingGen.GenerateFideID(index)
	federationID := s.ratingGen.GenerateFederationID(federation, index)

	id := uuid.New()
	s.createdIDs = append(s.createdIDs, id)

	// Track by rating bucket (every 100 points)
	bucket := (fideRating / 100) * 100
	s.playersByRating[bucket] = append(s.playersByRating[bucket], id)

	return SeededPlayer{
		ID:           id,
		FirstName:    firstName,
		LastName:     lastName,
		Gender:       gender,
		Federation:   federation,
		FideRating:   fideRating,
		DWZ:          dwz,
		FideTitle:    title,
		BirthYear:    birthYear,
		FideID:       fideID,
		FederationID: federationID,
	}
}

func (s *PlayerSeeder) insertPlayerBatch(ctx context.Context, players []SeededPlayer) error {
	now := time.Now()

	for _, p := range players {
		birthDate := fmt.Sprintf("%d-06-15", p.BirthYear) // Mid-year as placeholder

		var dsbID, oesbID, ssbID *string
		switch p.Federation {
		case "GER":
			dsbID = &p.FederationID
		case "AUT":
			oesbID = &p.FederationID
		case "SUI":
			ssbID = &p.FederationID
		}

		_, err := s.pool.Exec(ctx, `
			INSERT INTO players (
				id, first_name, last_name, birth_date, gender,
				fide_id, dsb_id, oesb_id, ssb_id,
				fide_standard, fide_rapid, fide_blitz, dwz,
				fide_title, federation,
				ratings_updated_at, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9,
				$10, $11, $12, $13,
				$14, $15,
				$16, $17, $18
			)
			ON CONFLICT (fide_id) DO NOTHING
		`, p.ID, p.FirstName, p.LastName, birthDate, p.Gender,
			p.FideID, dsbID, oesbID, ssbID,
			p.FideRating, p.FideRating-50, p.FideRating-100, p.DWZ,
			p.FideTitle, p.Federation,
			now, now, now)

		if err != nil {
			return err
		}
	}

	return nil
}

// GetPlayersForTournament returns player IDs suitable for a tournament of given size
// Optionally filters by rating range for varied tournament compositions
func (s *PlayerSeeder) GetPlayersForTournament(count int, minRating, maxRating int) []uuid.UUID {
	var eligible []uuid.UUID

	for bucket, ids := range s.playersByRating {
		if bucket >= minRating && bucket <= maxRating {
			eligible = append(eligible, ids...)
		}
	}

	// If not enough in rating range, use all
	if len(eligible) < count {
		eligible = s.createdIDs
	}

	if count > len(eligible) {
		count = len(eligible)
	}

	// Shuffle and return first N
	shuffled := make([]uuid.UUID, len(eligible))
	copy(shuffled, eligible)
	s.nameGen.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

// GetCreatedIDs returns all created player IDs
func (s *PlayerSeeder) GetCreatedIDs() []uuid.UUID {
	return s.createdIDs
}

// ClearPlayers removes all seeded players (for force cleanup)
func (s *PlayerSeeder) ClearPlayers(ctx context.Context) error {
	// Delete players with our FIDE ID pattern (46XXXXXX)
	_, err := s.pool.Exec(ctx, `
		DELETE FROM players WHERE fide_id LIKE '46%'
	`)
	return err
}
