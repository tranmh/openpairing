package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ProgressionSeeder handles tournament progression (rounds/games)
type ProgressionSeeder struct {
	pool *pgxpool.Pool
	rand *rand.Rand
}

// NewProgressionSeeder creates a new progression seeder
func NewProgressionSeeder(pool *pgxpool.Pool, seed int64) *ProgressionSeeder {
	return &ProgressionSeeder{
		pool: pool,
		rand: rand.New(rand.NewSource(seed)),
	}
}

// ParticipantState tracks a participant's state during progression
type ParticipantState struct {
	ID           uuid.UUID
	StartRank    int
	Rating       int
	Points       float64
	Buchholz     float64
	Wins         int
	Draws        int
	Losses       int
	ColorHistory []string // "W" or "B"
	Opponents    []int    // Start ranks played against
}

// SeedProgression creates rounds and games for a tournament
func (s *ProgressionSeeder) SeedProgression(ctx context.Context, tournament *SeededTournament) error {
	scenario := tournament.Scenario

	// Skip if tournament is in registration (no rounds played)
	if scenario.Status == "registration" || scenario.CurrentRound == 0 {
		return nil
	}

	log.Debug().
		Str("name", scenario.Name).
		Int("rounds", scenario.CurrentRound).
		Msg("Generating tournament progression")

	// Load participants with ratings
	participants, err := s.loadParticipants(ctx, tournament.ID)
	if err != nil {
		return fmt.Errorf("failed to load participants: %w", err)
	}

	// Generate rounds up to current round
	roundsToComplete := scenario.CurrentRound
	if scenario.Status == "ongoing" {
		roundsToComplete-- // Current round is in progress
	}

	for round := 1; round <= roundsToComplete; round++ {
		roundID, err := s.createRound(ctx, tournament.ID, round, true)
		if err != nil {
			return fmt.Errorf("failed to create round %d: %w", round, err)
		}

		// Generate pairings using simple Swiss simulation
		pairings := s.generateSwissPairings(participants, round)

		// Create games with results
		if err := s.createGamesWithResults(ctx, tournament.ID, roundID, pairings, participants); err != nil {
			return fmt.Errorf("failed to create games for round %d: %w", round, err)
		}
	}

	// If ongoing, create current round with pairings but no results
	if scenario.Status == "ongoing" {
		roundID, err := s.createRound(ctx, tournament.ID, scenario.CurrentRound, false)
		if err != nil {
			return fmt.Errorf("failed to create current round: %w", err)
		}

		pairings := s.generateSwissPairings(participants, scenario.CurrentRound)
		if err := s.createGamesWithoutResults(ctx, tournament.ID, roundID, pairings); err != nil {
			return fmt.Errorf("failed to create current round games: %w", err)
		}
	}

	// Update participant standings
	if err := s.updateParticipantStandings(ctx, tournament.ID, participants); err != nil {
		return fmt.Errorf("failed to update standings: %w", err)
	}

	return nil
}

func (s *ProgressionSeeder) loadParticipants(ctx context.Context, tournamentID uuid.UUID) (map[int]*ParticipantState, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tp.id, tp.start_rank, tp.starting_rating
		FROM tournament_participants tp
		WHERE tp.tournament_id = $1
		ORDER BY tp.starting_rating DESC, tp.start_rank
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := make(map[int]*ParticipantState)
	for rows.Next() {
		var p ParticipantState
		if err := rows.Scan(&p.ID, &p.StartRank, &p.Rating); err != nil {
			return nil, err
		}
		p.ColorHistory = make([]string, 0)
		p.Opponents = make([]int, 0)
		participants[p.StartRank] = &p
	}

	return participants, nil
}

func (s *ProgressionSeeder) createRound(ctx context.Context, tournamentID uuid.UUID, roundNum int, completed bool) (uuid.UUID, error) {
	id := uuid.New()
	now := time.Now()

	status := "completed"
	var completedAt *time.Time
	if completed {
		t := now.Add(-time.Duration(24*roundNum) * time.Hour)
		completedAt = &t
	} else {
		status = "ongoing"
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO tournament_rounds (
			id, tournament_id, round_number,
			status, paired_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, id, tournamentID, roundNum, status, now, completedAt)

	return id, err
}

// Pairing represents a single game pairing
type Pairing struct {
	Board      int
	WhiteRank  int
	BlackRank  int
	IsBye      bool
}

func (s *ProgressionSeeder) generateSwissPairings(participants map[int]*ParticipantState, round int) []Pairing {
	// Simple Swiss pairing simulation:
	// 1. Sort by points, then rating
	// 2. Pair top half with bottom half of each score group
	// 3. Avoid repeat pairings

	// Get all active participants sorted by points then rating
	type rankedPlayer struct {
		rank   int
		points float64
		rating int
	}

	players := make([]rankedPlayer, 0, len(participants))
	for rank, p := range participants {
		players = append(players, rankedPlayer{rank: rank, points: p.Points, rating: p.Rating})
	}

	// Sort by points (desc), then rating (desc)
	for i := 0; i < len(players)-1; i++ {
		for j := i + 1; j < len(players); j++ {
			if players[i].points < players[j].points ||
				(players[i].points == players[j].points && players[i].rating < players[j].rating) {
				players[i], players[j] = players[j], players[i]
			}
		}
	}

	pairings := make([]Pairing, 0)
	paired := make(map[int]bool)
	board := 1

	// Simple pairing: pair consecutive unpaired players
	for i := 0; i < len(players); i++ {
		if paired[players[i].rank] {
			continue
		}

		// Find opponent
		for j := i + 1; j < len(players); j++ {
			if paired[players[j].rank] {
				continue
			}

			// Check if they've played before
			p1 := participants[players[i].rank]
			alreadyPlayed := false
			for _, opp := range p1.Opponents {
				if opp == players[j].rank {
					alreadyPlayed = true
					break
				}
			}

			if !alreadyPlayed {
				// Determine colors
				whiteRank, blackRank := s.assignColors(
					participants[players[i].rank],
					participants[players[j].rank],
				)

				pairings = append(pairings, Pairing{
					Board:     board,
					WhiteRank: whiteRank,
					BlackRank: blackRank,
				})
				board++

				paired[players[i].rank] = true
				paired[players[j].rank] = true
				break
			}
		}
	}

	// Handle odd player (bye)
	for _, p := range players {
		if !paired[p.rank] {
			pairings = append(pairings, Pairing{
				Board:     board,
				WhiteRank: p.rank,
				BlackRank: 0,
				IsBye:     true,
			})
			break
		}
	}

	return pairings
}

func (s *ProgressionSeeder) assignColors(p1, p2 *ParticipantState) (white, black int) {
	// Count color imbalance
	p1Whites := 0
	p2Whites := 0
	for _, c := range p1.ColorHistory {
		if c == "W" {
			p1Whites++
		}
	}
	for _, c := range p2.ColorHistory {
		if c == "W" {
			p2Whites++
		}
	}

	p1Balance := p1Whites - (len(p1.ColorHistory) - p1Whites)
	p2Balance := p2Whites - (len(p2.ColorHistory) - p2Whites)

	// Give white to player with fewer whites (or random if equal)
	if p1Balance < p2Balance {
		return p1.StartRank, p2.StartRank
	} else if p1Balance > p2Balance {
		return p2.StartRank, p1.StartRank
	}

	// Equal balance - alternate based on last color or random
	if len(p1.ColorHistory) > 0 && p1.ColorHistory[len(p1.ColorHistory)-1] == "B" {
		return p1.StartRank, p2.StartRank
	} else if len(p2.ColorHistory) > 0 && p2.ColorHistory[len(p2.ColorHistory)-1] == "B" {
		return p2.StartRank, p1.StartRank
	}

	// Random
	if s.rand.Float64() < 0.5 {
		return p1.StartRank, p2.StartRank
	}
	return p2.StartRank, p1.StartRank
}

func (s *ProgressionSeeder) createGamesWithResults(ctx context.Context, tournamentID, roundID uuid.UUID, pairings []Pairing, participants map[int]*ParticipantState) error {
	now := time.Now()

	for _, p := range pairings {
		id := uuid.New()

		var result string
		var whiteScore, blackScore float64

		if p.IsBye {
			result = "BYE"
			whiteScore = 1.0
			blackScore = 0.0

			// Update participant state
			if white, ok := participants[p.WhiteRank]; ok {
				white.Points += 1.0
				white.Wins++
			}
		} else {
			// Generate realistic result based on rating difference
			white := participants[p.WhiteRank]
			black := participants[p.BlackRank]

			result, whiteScore, blackScore = s.generateResult(white.Rating, black.Rating)

			// Update participant states
			white.Points += whiteScore
			black.Points += blackScore
			white.ColorHistory = append(white.ColorHistory, "W")
			black.ColorHistory = append(black.ColorHistory, "B")
			white.Opponents = append(white.Opponents, black.StartRank)
			black.Opponents = append(black.Opponents, white.StartRank)

			if whiteScore == 1.0 {
				white.Wins++
				black.Losses++
			} else if whiteScore == 0.0 {
				white.Losses++
				black.Wins++
			} else {
				white.Draws++
				black.Draws++
			}
		}

		var whiteID, blackID *uuid.UUID
		if white, ok := participants[p.WhiteRank]; ok {
			whiteID = &white.ID
		}
		if !p.IsBye {
			if black, ok := participants[p.BlackRank]; ok {
				blackID = &black.ID
			}
		}

		_, err := s.pool.Exec(ctx, `
			INSERT INTO games (
				id, tournament_id, round_id,
				board_number, white_participant_id, black_participant_id,
				result, white_score, black_score,
				status, result_entered_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, id, tournamentID, roundID,
			p.Board, whiteID, blackID,
			result, whiteScore, blackScore,
			"completed", now, now)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ProgressionSeeder) createGamesWithoutResults(ctx context.Context, tournamentID, roundID uuid.UUID, pairings []Pairing) error {
	now := time.Now()

	// Get participant map
	rows, err := s.pool.Query(ctx, `
		SELECT id, start_rank FROM tournament_participants WHERE tournament_id = $1
	`, tournamentID)
	if err != nil {
		return err
	}
	defer rows.Close()

	rankToID := make(map[int]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var rank int
		if err := rows.Scan(&id, &rank); err != nil {
			return err
		}
		rankToID[rank] = id
	}

	for _, p := range pairings {
		id := uuid.New()

		var whiteID, blackID *uuid.UUID
		if wid, ok := rankToID[p.WhiteRank]; ok {
			whiteID = &wid
		}
		if !p.IsBye {
			if bid, ok := rankToID[p.BlackRank]; ok {
				blackID = &bid
			}
		}

		_, err := s.pool.Exec(ctx, `
			INSERT INTO games (
				id, tournament_id, round_id,
				board_number, white_participant_id, black_participant_id,
				status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id, tournamentID, roundID,
			p.Board, whiteID, blackID,
			"scheduled", now)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ProgressionSeeder) generateResult(whiteRating, blackRating int) (result string, whiteScore, blackScore float64) {
	// Expected score formula (simplified ELO)
	diff := float64(whiteRating - blackRating)
	expectedWhite := 1.0 / (1.0 + math.Pow(10, -diff/400.0))

	// Add some randomness and white advantage
	expectedWhite = expectedWhite*0.85 + 0.075 // Scale to ~15% draws

	r := s.rand.Float64()

	// Determine result
	if r < expectedWhite*0.7 { // White wins
		return "1-0", 1.0, 0.0
	} else if r < expectedWhite*0.7+0.2 { // Draw
		return "1/2-1/2", 0.5, 0.5
	} else { // Black wins
		return "0-1", 0.0, 1.0
	}
}

func (s *ProgressionSeeder) updateParticipantStandings(ctx context.Context, tournamentID uuid.UUID, participants map[int]*ParticipantState) error {
	// Calculate Buchholz scores
	for _, p := range participants {
		var buchholz float64
		for _, oppRank := range p.Opponents {
			if opp, ok := participants[oppRank]; ok {
				buchholz += opp.Points
			}
		}
		p.Buchholz = buchholz
	}

	// Update database
	for _, p := range participants {
		_, err := s.pool.Exec(ctx, `
			UPDATE tournament_participants
			SET points = $1, buchholz = $2, wins = $3, draws = $4, losses = $5
			WHERE id = $6
		`, p.Points, p.Buchholz, p.Wins, p.Draws, p.Losses, p.ID)

		if err != nil {
			return err
		}
	}

	return nil
}
