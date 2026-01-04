package reports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDataProvider implements TournamentDataProvider using PostgreSQL
type PostgresDataProvider struct {
	db *pgxpool.Pool
}

// NewPostgresDataProvider creates a new PostgreSQL data provider
func NewPostgresDataProvider(db *pgxpool.Pool) *PostgresDataProvider {
	return &PostgresDataProvider{db: db}
}

// GetTournament fetches tournament data
func (p *PostgresDataProvider) GetTournament(ctx context.Context, id uuid.UUID) (*TournamentData, error) {
	query := `
		SELECT id, name, start_date, COALESCE(end_date, start_date),
		       COALESCE(venue_city, ''),
		       time_control, rounds, pairing_system
		FROM tournaments
		WHERE id = $1`

	var t TournamentData
	var startDate, endDate time.Time
	err := p.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &startDate, &endDate,
		&t.City,
		&t.TimeControl, &t.Rounds, &t.PairingSystem,
	)
	if err != nil {
		return nil, err
	}

	t.StartDate = startDate
	t.EndDate = endDate
	// ChiefArbiter and Organizer are not stored in tournaments table yet
	t.ChiefArbiter = ""
	t.Organizer = ""

	return &t, nil
}

// GetParticipants fetches participant data for a tournament
func (p *PostgresDataProvider) GetParticipants(ctx context.Context, tournamentID uuid.UUID) ([]ParticipantData, error) {
	query := `
		SELECT
			tp.start_rank,
			tp.player_id,
			COALESCE(pl.first_name, ''),
			COALESCE(pl.last_name, ''),
			COALESCE(pl.dsb_id, ''),
			COALESCE(pl.club_id::text, ''),
			COALESCE(c.name, ''),
			COALESCE(pl.dwz, 0),
			COALESCE(pl.dwz_index, 0),
			COALESCE(pl.fide_id, ''),
			COALESCE(pl.fide_standard, 0),
			COALESCE(EXTRACT(YEAR FROM pl.birth_date)::int, 0),
			COALESCE(pl.gender, 'M'),
			tp.points,
			tp.start_rank
		FROM tournament_participants tp
		JOIN players pl ON tp.player_id = pl.id
		LEFT JOIN clubs c ON pl.club_id = c.id
		WHERE tp.tournament_id = $1 AND tp.is_active = true
		ORDER BY tp.start_rank`

	rows, err := p.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []ParticipantData
	for rows.Next() {
		var pd ParticipantData
		err := rows.Scan(
			&pd.StartNumber,
			&pd.PlayerID,
			&pd.FirstName,
			&pd.LastName,
			&pd.DsbID,
			&pd.ClubID,
			&pd.ClubName,
			&pd.DWZ,
			&pd.DWZIndex,
			&pd.FideID,
			&pd.FideRating,
			&pd.BirthYear,
			&pd.Gender,
			&pd.Score,
			&pd.Rank,
		)
		if err != nil {
			return nil, err
		}
		participants = append(participants, pd)
	}

	return participants, rows.Err()
}

// GetGames fetches game data for a tournament
func (p *PostgresDataProvider) GetGames(ctx context.Context, tournamentID uuid.UUID) ([]GameData, error) {
	query := `
		SELECT
			r.round_number,
			COALESCE(g.white_participant_id, '00000000-0000-0000-0000-000000000000'),
			COALESCE(g.black_participant_id, '00000000-0000-0000-0000-000000000000'),
			COALESCE(g.white_score, 0),
			COALESCE(g.black_score, 0)
		FROM games g
		JOIN tournament_rounds r ON g.round_id = r.id
		WHERE r.tournament_id = $1 AND g.result IS NOT NULL
		ORDER BY r.round_number, g.board_number`

	rows, err := p.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []GameData
	for rows.Next() {
		var gd GameData
		var whiteID, blackID string
		err := rows.Scan(
			&gd.Round,
			&whiteID,
			&blackID,
			&gd.WhiteScore,
			&gd.BlackScore,
		)
		if err != nil {
			return nil, err
		}

		// Parse UUIDs, handle null participant IDs
		if whiteID != "00000000-0000-0000-0000-000000000000" {
			gd.WhiteID, _ = uuid.Parse(whiteID)
		}
		if blackID != "00000000-0000-0000-0000-000000000000" {
			gd.BlackID, _ = uuid.Parse(blackID)
		}

		games = append(games, gd)
	}

	return games, rows.Err()
}
