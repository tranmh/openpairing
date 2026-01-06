package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// TournamentSeeder handles tournament seeding
type TournamentSeeder struct {
	pool *pgxpool.Pool
}

// NewTournamentSeeder creates a new tournament seeder
func NewTournamentSeeder(pool *pgxpool.Pool) *TournamentSeeder {
	return &TournamentSeeder{pool: pool}
}

// TournamentScenario defines a tournament to seed
type TournamentScenario struct {
	Name            string
	Type            string // swiss, round_robin, knockout, team_swiss, arena
	Size            int    // Number of participants
	Status          string // registration, ongoing, completed
	Rounds          int
	CurrentRound    int
	Mode            string // otb, online
	PairingEngine   string // javafo, bbp, round_robin, knockout, monrad, arena
	TimeControl     string
	TimeControlType string // standard, rapid, blitz
	VenueCity       string
	MinRating       int // For filtering players
	MaxRating       int
}

// GetTournamentScenarios returns all tournament scenarios to seed
func GetTournamentScenarios() []TournamentScenario {
	return []TournamentScenario{
		// Small tournaments (8 players)
		{
			Name:            "Berliner Schnellschach Open 2025",
			Type:            "swiss",
			Size:            8,
			Status:          "registration",
			Rounds:          5,
			CurrentRound:    0,
			Mode:            "otb",
			PairingEngine:   "javafo",
			TimeControl:     "15+10",
			TimeControlType: "rapid",
			VenueCity:       "Berlin",
			MinRating:       1400,
			MaxRating:       2200,
		},
		{
			Name:            "Wiener Blitz-Meisterschaft 2025",
			Type:            "swiss",
			Size:            8,
			Status:          "ongoing",
			Rounds:          7,
			CurrentRound:    4,
			Mode:            "otb",
			PairingEngine:   "javafo",
			TimeControl:     "3+2",
			TimeControlType: "blitz",
			VenueCity:       "Wien",
			MinRating:       1600,
			MaxRating:       2400,
		},
		{
			Name:            "Zürich Vereinsturnier 2024",
			Type:            "round_robin",
			Size:            8,
			Status:          "completed",
			Rounds:          7,
			CurrentRound:    7,
			Mode:            "otb",
			PairingEngine:   "round_robin",
			TimeControl:     "90+30",
			TimeControlType: "standard",
			VenueCity:       "Zürich",
			MinRating:       1800,
			MaxRating:       2500,
		},

		// Medium tournaments (64 players)
		{
			Name:            "Deutsche Jugend-Einzelmeisterschaft U18 2025",
			Type:            "swiss",
			Size:            64,
			Status:          "registration",
			Rounds:          9,
			CurrentRound:    0,
			Mode:            "otb",
			PairingEngine:   "javafo",
			TimeControl:     "90+30",
			TimeControlType: "standard",
			VenueCity:       "Hamburg",
			MinRating:       1200,
			MaxRating:       2200,
		},
		{
			Name:            "Österreichische Staatsmeisterschaft Rapid 2025",
			Type:            "swiss",
			Size:            64,
			Status:          "ongoing",
			Rounds:          9,
			CurrentRound:    5,
			Mode:            "otb",
			PairingEngine:   "bbp",
			TimeControl:     "15+10",
			TimeControlType: "rapid",
			VenueCity:       "Graz",
			MinRating:       1600,
			MaxRating:       2600,
		},
		{
			Name:            "Schweizer Mannschaftsmeisterschaft 2024",
			Type:            "team_swiss",
			Size:            64,
			Status:          "completed",
			Rounds:          7,
			CurrentRound:    7,
			Mode:            "otb",
			PairingEngine:   "team_swiss",
			TimeControl:     "90+30",
			TimeControlType: "standard",
			VenueCity:       "Bern",
			MinRating:       1500,
			MaxRating:       2400,
		},

		// Large tournaments (320+ players)
		{
			Name:            "Grenke Chess Open 2025",
			Type:            "swiss",
			Size:            320,
			Status:          "registration",
			Rounds:          9,
			CurrentRound:    0,
			Mode:            "otb",
			PairingEngine:   "javafo",
			TimeControl:     "90+30",
			TimeControlType: "standard",
			VenueCity:       "Karlsruhe",
			MinRating:       800,
			MaxRating:       2600,
		},
		{
			Name:            "Wien Open 2025",
			Type:            "swiss",
			Size:            350,
			Status:          "ongoing",
			Rounds:          9,
			CurrentRound:    6,
			Mode:            "otb",
			PairingEngine:   "javafo",
			TimeControl:     "90+30",
			TimeControlType: "standard",
			VenueCity:       "Wien",
			MinRating:       800,
			MaxRating:       2600,
		},
		{
			Name:            "Biel Chess Festival 2024",
			Type:            "swiss",
			Size:            400,
			Status:          "completed",
			Rounds:          9,
			CurrentRound:    9,
			Mode:            "otb",
			PairingEngine:   "bbp",
			TimeControl:     "90+30",
			TimeControlType: "standard",
			VenueCity:       "Biel",
			MinRating:       800,
			MaxRating:       2600,
		},

		// Special formats
		{
			Name:            "Hamburg Arena Online 2025",
			Type:            "arena",
			Size:            32,
			Status:          "completed",
			Rounds:          0,
			CurrentRound:    0,
			Mode:            "online",
			PairingEngine:   "arena",
			TimeControl:     "3+0",
			TimeControlType: "blitz",
			VenueCity:       "Hamburg",
			MinRating:       1200,
			MaxRating:       2200,
		},
		{
			Name:            "Münchner KO-Turnier 2025",
			Type:            "knockout",
			Size:            16,
			Status:          "ongoing",
			Rounds:          4,
			CurrentRound:    2,
			Mode:            "otb",
			PairingEngine:   "knockout",
			TimeControl:     "25+10",
			TimeControlType: "rapid",
			VenueCity:       "München",
			MinRating:       1800,
			MaxRating:       2400,
		},
	}
}

// SeededTournament represents a seeded tournament
type SeededTournament struct {
	ID             uuid.UUID
	Scenario       TournamentScenario
	ParticipantIDs []uuid.UUID
}

// SeedTournament creates a single tournament with participants
func (s *TournamentSeeder) SeedTournament(ctx context.Context, scenario TournamentScenario, playerIDs []uuid.UUID) (*SeededTournament, error) {
	// Create tournament
	id := uuid.New()
	slug := strings.ToLower(strings.ReplaceAll(scenario.Name, " ", "-"))
	now := time.Now()

	// Calculate dates based on status
	var startDate, endDate time.Time
	switch scenario.Status {
	case "registration":
		startDate = now.AddDate(0, 1, 0) // 1 month from now
		endDate = startDate.AddDate(0, 0, 3)
	case "ongoing":
		startDate = now.AddDate(0, 0, -3) // Started 3 days ago
		endDate = now.AddDate(0, 0, 4)
	case "completed":
		startDate = now.AddDate(0, -1, 0) // 1 month ago
		endDate = startDate.AddDate(0, 0, 5)
	}

	// Default tiebreaks
	tiebreaks := []string{"buchholz_cut1", "buchholz", "sonneborn_berger", "direct_encounter", "wins"}
	tiebreaksJSON, _ := json.Marshal(tiebreaks)

	// Insert tournament
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tournaments (
			id, name, description, slug,
			type, mode, time_control, time_control_type,
			pairing_engine, pairing_system, accelerated_pairing,
			start_date, end_date, rounds, min_players, max_players,
			venue_name, venue_city, venue_country,
			fide_rated, dwz_rated,
			entry_fee_cents, currency,
			status, current_round, participant_count,
			tiebreaks,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19,
			$20, $21,
			$22, $23,
			$24, $25, $26,
			$27,
			$28, $29
		)
	`, id, scenario.Name, fmt.Sprintf("Seeded tournament: %s", scenario.Name), slug,
		scenario.Type, scenario.Mode, scenario.TimeControl, scenario.TimeControlType,
		scenario.PairingEngine, "dutch", false,
		startDate, endDate, scenario.Rounds, scenario.Size/2, scenario.Size,
		scenario.VenueCity+" Chess Club", scenario.VenueCity, "DE",
		scenario.TimeControlType == "standard", scenario.TimeControlType != "blitz",
		0, "EUR",
		scenario.Status, scenario.CurrentRound, len(playerIDs),
		tiebreaksJSON,
		now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create tournament: %w", err)
	}

	// Add participants
	participantIDs := make([]uuid.UUID, 0, len(playerIDs))
	for i, playerID := range playerIDs {
		participantID, err := s.addParticipant(ctx, id, playerID, i+1)
		if err != nil {
			return nil, fmt.Errorf("failed to add participant: %w", err)
		}
		participantIDs = append(participantIDs, participantID)
	}

	log.Debug().
		Str("name", scenario.Name).
		Str("status", scenario.Status).
		Int("participants", len(participantIDs)).
		Msg("Tournament created")

	return &SeededTournament{
		ID:             id,
		Scenario:       scenario,
		ParticipantIDs: participantIDs,
	}, nil
}

func (s *TournamentSeeder) addParticipant(ctx context.Context, tournamentID, playerID uuid.UUID, startRank int) (uuid.UUID, error) {
	// Get player's rating
	var fideRating *int
	err := s.pool.QueryRow(ctx, `
		SELECT fide_standard FROM players WHERE id = $1
	`, playerID).Scan(&fideRating)
	if err != nil {
		return uuid.Nil, err
	}

	rating := 1500
	if fideRating != nil {
		rating = *fideRating
	}

	id := uuid.New()
	now := time.Now()

	_, err = s.pool.Exec(ctx, `
		INSERT INTO tournament_participants (
			id, tournament_id, player_id,
			registration_status, registered_at,
			start_rank, starting_rating,
			points, buchholz, buchholz_cut1, sonneborn_berger,
			wins, draws, losses,
			is_active, payment_status, registration_source
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7,
			$8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17
		)
	`, id, tournamentID, playerID,
		"confirmed", now,
		startRank, rating,
		0, 0, 0, 0,
		0, 0, 0,
		true, "not_required", "seed")

	return id, err
}

// GetParticipantMap returns a map of start_rank to participant_id for a tournament
func (s *TournamentSeeder) GetParticipantMap(ctx context.Context, tournamentID uuid.UUID) (map[int]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, start_rank FROM tournament_participants
		WHERE tournament_id = $1
		ORDER BY start_rank
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var rank int
		if err := rows.Scan(&id, &rank); err != nil {
			return nil, err
		}
		result[rank] = id
	}

	return result, nil
}

// ClearTournaments removes all seeded tournaments
func (s *TournamentSeeder) ClearTournaments(ctx context.Context) error {
	// Delete tournaments by description pattern
	_, err := s.pool.Exec(ctx, `
		DELETE FROM tournaments WHERE description LIKE 'Seeded tournament:%'
	`)
	return err
}
