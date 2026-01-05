package standings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for standings
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new standings repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ParticipantData contains data needed for standings calculation
type ParticipantData struct {
	ID             uuid.UUID
	PlayerID       uuid.UUID
	PlayerName     string
	PlayerFideID   *string
	PlayerTitle    *string
	ClubName       *string
	StartRank      int
	Rating         int
	Points         float64
	Wins           int
	Draws          int
	Losses         int
	Buchholz       float64
	BuchholzCut1   float64
	SonnebornBerger float64
	IsActive       bool
}

// GameData contains data about a game for tiebreak calculation
type GameData struct {
	ID                 uuid.UUID
	RoundNumber        int
	WhiteParticipantID *uuid.UUID
	BlackParticipantID *uuid.UUID
	Result             *string
	WhiteScore         *float64
	BlackScore         *float64
}

// ScoringConfig holds scoring settings for standings calculations
type ScoringConfig struct {
	WinPoints           float64
	DrawPoints          float64
	LossPoints          float64
	SonnebornBergerMode string
}

// TournamentInfo contains tournament information for standings
type TournamentInfo struct {
	ID           uuid.UUID
	Name         string
	CurrentRound int
	TotalRounds  int
	Tiebreaks    []string
	Scoring      *ScoringConfig
}

// GetTournamentInfo retrieves tournament information
func (r *Repository) GetTournamentInfo(ctx context.Context, tournamentID uuid.UUID) (*TournamentInfo, error) {
	var info TournamentInfo
	var tiebreaksJSON []byte
	var scoringJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT id, name, current_round, rounds, tiebreaks, scoring_settings
		FROM tournaments
		WHERE id = $1
	`, tournamentID).Scan(
		&info.ID,
		&info.Name,
		&info.CurrentRound,
		&info.TotalRounds,
		&tiebreaksJSON,
		&scoringJSON,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tournament not found")
		}
		return nil, fmt.Errorf("failed to get tournament: %w", err)
	}

	// Parse tiebreaks JSON
	if len(tiebreaksJSON) > 0 {
		// Simple JSON array parsing
		info.Tiebreaks = parseTiebreaks(tiebreaksJSON)
	}

	// Parse scoring settings JSON
	info.Scoring = parseScoringSettings(scoringJSON)

	return &info, nil
}

// GetParticipants retrieves all participants for a tournament
func (r *Repository) GetParticipants(ctx context.Context, tournamentID uuid.UUID) ([]ParticipantData, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			tp.id,
			tp.player_id,
			COALESCE(p.first_name, '') || ' ' || COALESCE(p.last_name, '') as player_name,
			p.fide_id,
			p.fide_title,
			c.name as club_name,
			tp.start_rank,
			COALESCE(tp.starting_rating, 0) as rating,
			tp.points,
			tp.wins,
			tp.draws,
			tp.losses,
			tp.buchholz,
			tp.buchholz_cut1,
			tp.sonneborn_berger,
			tp.is_active
		FROM tournament_participants tp
		JOIN players p ON p.id = tp.player_id
		LEFT JOIN clubs c ON c.id = p.club_id
		WHERE tp.tournament_id = $1
		ORDER BY tp.points DESC, tp.buchholz_cut1 DESC, tp.start_rank ASC
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	var participants []ParticipantData
	for rows.Next() {
		var p ParticipantData
		err := rows.Scan(
			&p.ID,
			&p.PlayerID,
			&p.PlayerName,
			&p.PlayerFideID,
			&p.PlayerTitle,
			&p.ClubName,
			&p.StartRank,
			&p.Rating,
			&p.Points,
			&p.Wins,
			&p.Draws,
			&p.Losses,
			&p.Buchholz,
			&p.BuchholzCut1,
			&p.SonnebornBerger,
			&p.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, p)
	}

	return participants, rows.Err()
}

// GetGames retrieves all games for a tournament
func (r *Repository) GetGames(ctx context.Context, tournamentID uuid.UUID) ([]GameData, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			g.id,
			tr.round_number,
			g.white_participant_id,
			g.black_participant_id,
			g.result,
			g.white_score,
			g.black_score
		FROM games g
		JOIN tournament_rounds tr ON tr.id = g.round_id
		WHERE g.tournament_id = $1
		ORDER BY tr.round_number, g.board_number
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	var games []GameData
	for rows.Next() {
		var g GameData
		err := rows.Scan(
			&g.ID,
			&g.RoundNumber,
			&g.WhiteParticipantID,
			&g.BlackParticipantID,
			&g.Result,
			&g.WhiteScore,
			&g.BlackScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}
		games = append(games, g)
	}

	return games, rows.Err()
}

// GetOpponentRatings retrieves opponent ratings for performance calculation
func (r *Repository) GetOpponentRatings(ctx context.Context, tournamentID, participantID uuid.UUID) ([]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(opp_tp.starting_rating, 0)
		FROM games g
		JOIN tournament_participants opp_tp ON
			(g.white_participant_id = $2 AND opp_tp.id = g.black_participant_id) OR
			(g.black_participant_id = $2 AND opp_tp.id = g.white_participant_id)
		WHERE g.tournament_id = $1 AND g.result IS NOT NULL
	`, tournamentID, participantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query opponent ratings: %w", err)
	}
	defer rows.Close()

	var ratings []int
	for rows.Next() {
		var rating int
		if err := rows.Scan(&rating); err != nil {
			return nil, fmt.Errorf("failed to scan rating: %w", err)
		}
		ratings = append(ratings, rating)
	}

	return ratings, rows.Err()
}

// ParticipantEligibilityData contains data needed for special ranking eligibility
type ParticipantEligibilityData struct {
	ParticipantID uuid.UUID
	BirthYear     int
	Rating        int
	Gender        string
}

// SpecialRankingData contains special ranking definition for standings
type SpecialRankingData struct {
	ID           uuid.UUID
	Name         string
	DisplayName  string
	MinBirthYear *int
	MaxBirthYear *int
	MinRating    *int
	MaxRating    *int
	Gender       *string
}

// GetSpecialRankings retrieves active special rankings for a tournament
func (r *Repository) GetSpecialRankings(ctx context.Context, tournamentID uuid.UUID) ([]SpecialRankingData, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, display_name, min_birth_year, max_birth_year, min_rating, max_rating, gender
		FROM special_rankings
		WHERE tournament_id = $1 AND is_active = true
		ORDER BY sort_order, name
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query special rankings: %w", err)
	}
	defer rows.Close()

	var rankings []SpecialRankingData
	for rows.Next() {
		var sr SpecialRankingData
		if err := rows.Scan(&sr.ID, &sr.Name, &sr.DisplayName, &sr.MinBirthYear, &sr.MaxBirthYear, &sr.MinRating, &sr.MaxRating, &sr.Gender); err != nil {
			return nil, fmt.Errorf("failed to scan special ranking: %w", err)
		}
		rankings = append(rankings, sr)
	}

	return rankings, rows.Err()
}

// GetParticipantEligibilityData retrieves eligibility data for all participants
func (r *Repository) GetParticipantEligibilityData(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]ParticipantEligibilityData, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			tp.id,
			COALESCE(EXTRACT(YEAR FROM p.birth_date::date), 0)::int as birth_year,
			COALESCE(tp.starting_rating, 0) as rating,
			COALESCE(p.gender, '') as gender
		FROM tournament_participants tp
		JOIN players p ON p.id = tp.player_id
		WHERE tp.tournament_id = $1
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query eligibility data: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]ParticipantEligibilityData)
	for rows.Next() {
		var data ParticipantEligibilityData
		if err := rows.Scan(&data.ParticipantID, &data.BirthYear, &data.Rating, &data.Gender); err != nil {
			return nil, fmt.Errorf("failed to scan eligibility data: %w", err)
		}
		result[data.ParticipantID] = data
	}

	return result, rows.Err()
}

// UpdateParticipantTiebreaks updates the denormalized tiebreak values for a participant
func (r *Repository) UpdateParticipantTiebreaks(ctx context.Context, participantID uuid.UUID, buchholz, buchholzCut1, sonnebornBerger float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tournament_participants
		SET buchholz = $2, buchholz_cut1 = $3, sonneborn_berger = $4
		WHERE id = $1
	`, participantID, buchholz, buchholzCut1, sonnebornBerger)
	if err != nil {
		return fmt.Errorf("failed to update tiebreaks: %w", err)
	}
	return nil
}

// GetCrosstableData retrieves data for crosstable generation
func (r *Repository) GetCrosstableData(ctx context.Context, tournamentID uuid.UUID) ([]CrosstableRow, int, error) {
	// Get tournament rounds
	var totalRounds int
	err := r.db.QueryRow(ctx, `SELECT rounds FROM tournaments WHERE id = $1`, tournamentID).Scan(&totalRounds)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get rounds: %w", err)
	}

	// Get participants ordered by points
	rows, err := r.db.Query(ctx, `
		SELECT
			tp.id,
			COALESCE(p.first_name, '') || ' ' || COALESCE(p.last_name, '') as player_name,
			COALESCE(tp.starting_rating, 0) as rating,
			tp.points,
			tp.start_rank
		FROM tournament_participants tp
		JOIN players p ON p.id = tp.player_id
		WHERE tp.tournament_id = $1
		ORDER BY tp.points DESC, tp.buchholz_cut1 DESC, tp.start_rank ASC
	`, tournamentID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	type participant struct {
		ID        uuid.UUID
		Name      string
		Rating    int
		Points    float64
		StartRank int
	}

	var participants []participant
	participantRanks := make(map[uuid.UUID]int)
	rank := 1
	for rows.Next() {
		var p participant
		if err := rows.Scan(&p.ID, &p.Name, &p.Rating, &p.Points, &p.StartRank); err != nil {
			return nil, 0, fmt.Errorf("failed to scan participant: %w", err)
		}
		participantRanks[p.ID] = rank
		participants = append(participants, p)
		rank++
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	// Get all games
	gameRows, err := r.db.Query(ctx, `
		SELECT
			tr.round_number,
			g.white_participant_id,
			g.black_participant_id,
			g.result
		FROM games g
		JOIN tournament_rounds tr ON tr.id = g.round_id
		WHERE g.tournament_id = $1
		ORDER BY tr.round_number
	`, tournamentID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query games: %w", err)
	}
	defer gameRows.Close()

	// Build game lookup: participantID -> roundNumber -> entry
	gameLookup := make(map[uuid.UUID]map[int]CrosstableEntry)
	for gameRows.Next() {
		var roundNum int
		var whiteID, blackID *uuid.UUID
		var result *string

		if err := gameRows.Scan(&roundNum, &whiteID, &blackID, &result); err != nil {
			return nil, 0, fmt.Errorf("failed to scan game: %w", err)
		}

		if whiteID != nil {
			if gameLookup[*whiteID] == nil {
				gameLookup[*whiteID] = make(map[int]CrosstableEntry)
			}
			entry := CrosstableEntry{Color: "w"}
			if blackID != nil {
				entry.OpponentRank = participantRanks[*blackID]
			}
			entry.Result = convertResultForWhite(result)
			gameLookup[*whiteID][roundNum] = entry
		}

		if blackID != nil {
			if gameLookup[*blackID] == nil {
				gameLookup[*blackID] = make(map[int]CrosstableEntry)
			}
			entry := CrosstableEntry{Color: "b"}
			if whiteID != nil {
				entry.OpponentRank = participantRanks[*whiteID]
			}
			entry.Result = convertResultForBlack(result)
			gameLookup[*blackID][roundNum] = entry
		}
	}

	// Build crosstable rows
	var crosstable []CrosstableRow
	for i, p := range participants {
		row := CrosstableRow{
			Rank:       i + 1,
			PlayerName: p.Name,
			Rating:     p.Rating,
			Points:     p.Points,
			Results:    make([]CrosstableEntry, totalRounds),
		}

		// Fill in results for each round
		for round := 1; round <= totalRounds; round++ {
			if entry, ok := gameLookup[p.ID][round]; ok {
				row.Results[round-1] = entry
			} else {
				row.Results[round-1] = CrosstableEntry{Color: "-", Result: nil}
			}
		}

		crosstable = append(crosstable, row)
	}

	return crosstable, totalRounds, nil
}

// Helper functions

func parseTiebreaks(data []byte) []string {
	// Simple parsing - data is like ["buchholz_cut1","buchholz","sonneborn_berger"]
	s := string(data)
	if len(s) < 2 {
		return nil
	}
	s = s[1 : len(s)-1] // Remove [ and ]
	if s == "" {
		return nil
	}

	var result []string
	current := ""
	inQuote := false
	for _, c := range s {
		switch c {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote && current != "" {
				result = append(result, current)
				current = ""
			}
		default:
			if inQuote {
				current += string(c)
			}
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func parseScoringSettings(data []byte) *ScoringConfig {
	if len(data) == 0 || string(data) == "null" {
		// Return default FIDE 1-0.5-0 scoring
		return &ScoringConfig{
			WinPoints:           1.0,
			DrawPoints:          0.5,
			LossPoints:          0.0,
			SonnebornBergerMode: "proportional",
		}
	}

	// Parse JSON manually (simple approach)
	var scoring ScoringConfig
	if err := json.Unmarshal(data, &scoring); err != nil {
		// Return defaults on parse error
		return &ScoringConfig{
			WinPoints:           1.0,
			DrawPoints:          0.5,
			LossPoints:          0.0,
			SonnebornBergerMode: "proportional",
		}
	}

	// Ensure SB mode has a default
	if scoring.SonnebornBergerMode == "" {
		scoring.SonnebornBergerMode = "proportional"
	}

	return &scoring
}

func convertResultForWhite(result *string) *string {
	if result == nil {
		return nil
	}
	switch *result {
	case "1-0", "1-0F":
		s := "1"
		return &s
	case "0-1", "0-1F":
		s := "0"
		return &s
	case "1/2-1/2":
		s := "="
		return &s
	case "BYE":
		s := "+"
		return &s
	default:
		return nil
	}
}

func convertResultForBlack(result *string) *string {
	if result == nil {
		return nil
	}
	switch *result {
	case "0-1", "0-1F":
		s := "1"
		return &s
	case "1-0", "1-0F":
		s := "0"
		return &s
	case "1/2-1/2":
		s := "="
		return &s
	default:
		return nil
	}
}
