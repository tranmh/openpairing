package export

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for exports
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new export repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetTournamentExportData retrieves all data needed to export a tournament
func (r *Repository) GetTournamentExportData(ctx context.Context, tournamentID uuid.UUID) (*TournamentExport, error) {
	// Get tournament details
	var tournament TournamentExport
	var endDate *time.Time
	var site string

	err := r.db.QueryRow(ctx, `
		SELECT
			t.id,
			t.name,
			COALESCE(t.venue_city, '') || ', ' || COALESCE(t.venue_country, ''),
			t.start_date,
			t.end_date,
			t.rounds,
			t.type,
			t.time_control,
			t.time_control_type,
			t.pairing_system,
			COALESCE((t.settings->>'chief_arbiter')::text, ''),
			COALESCE((t.settings->>'deputy_arbiter')::text, ''),
			COALESCE(t.venue_country, ''),
			t.participant_count
		FROM tournaments t
		WHERE t.id = $1
	`, tournamentID).Scan(
		&tournament.ID,
		&tournament.Name,
		&site,
		&tournament.StartDate,
		&endDate,
		&tournament.Rounds,
		&tournament.Type,
		&tournament.TimeControl,
		&tournament.TimeControlType,
		&tournament.PairingSystem,
		&tournament.Chief,
		&tournament.Deputy,
		&tournament.Federation,
		&tournament.ParticipantCount,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tournament not found")
		}
		return nil, fmt.Errorf("failed to get tournament: %w", err)
	}

	tournament.Site = site

	// Get participants with their results
	participants, err := r.GetParticipantsExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	tournament.Participants = participants

	// Get games
	games, err := r.GetGamesExportData(ctx, tournamentID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}
	tournament.Games = games

	return &tournament, nil
}

// GetParticipantsExportData retrieves all participants for export
func (r *Repository) GetParticipantsExportData(ctx context.Context, tournamentID uuid.UUID) ([]ParticipantExport, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			tp.id,
			tp.start_rank,
			COALESCE(p.first_name, '') || ' ' || COALESCE(p.last_name, ''),
			p.fide_id,
			COALESCE(tp.starting_rating, 0),
			p.fide_title,
			p.federation,
			c.name,
			tp.points,
			tp.wins,
			tp.draws,
			tp.losses,
			tp.buchholz,
			tp.buchholz_cut1,
			tp.sonneborn_berger
		FROM tournament_participants tp
		JOIN players p ON p.id = tp.player_id
		LEFT JOIN clubs c ON c.id = p.club_id
		WHERE tp.tournament_id = $1
		ORDER BY tp.points DESC, tp.buchholz_cut1 DESC, tp.buchholz DESC, tp.start_rank ASC
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	var participants []ParticipantExport
	rank := 1
	for rows.Next() {
		var p ParticipantExport
		err := rows.Scan(
			&p.ID,
			&p.StartRank,
			&p.Name,
			&p.FideID,
			&p.Rating,
			&p.Title,
			&p.Federation,
			&p.Club,
			&p.Points,
			&p.Wins,
			&p.Draws,
			&p.Losses,
			&p.Buchholz,
			&p.BuchholzCut1,
			&p.SonnebornBerger,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		p.Rank = rank
		rank++
		participants = append(participants, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating participants: %w", err)
	}

	// Get round results for each participant
	for i := range participants {
		results, err := r.GetParticipantRoundResults(ctx, tournamentID, participants[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get results for participant %s: %w", participants[i].ID, err)
		}
		participants[i].Results = results
	}

	return participants, nil
}

// GetParticipantRoundResults retrieves round-by-round results for a participant
func (r *Repository) GetParticipantRoundResults(ctx context.Context, tournamentID, participantID uuid.UUID) ([]RoundResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			tr.round_number,
			CASE
				WHEN g.white_participant_id = $2 THEN opp_tp.start_rank
				WHEN g.black_participant_id = $2 THEN opp_tp.start_rank
				ELSE NULL
			END as opponent_rank,
			CASE
				WHEN g.white_participant_id = $2 THEN 'w'
				WHEN g.black_participant_id = $2 THEN 'b'
				ELSE '-'
			END as color,
			CASE
				WHEN g.result = 'BYE' THEN '+'
				WHEN g.white_participant_id = $2 AND g.result = '1-0' THEN '1'
				WHEN g.white_participant_id = $2 AND g.result = '0-1' THEN '0'
				WHEN g.white_participant_id = $2 AND g.result = '1/2-1/2' THEN '='
				WHEN g.white_participant_id = $2 AND g.result = '1-0F' THEN '+'
				WHEN g.white_participant_id = $2 AND g.result = '0-1F' THEN '-'
				WHEN g.black_participant_id = $2 AND g.result = '0-1' THEN '1'
				WHEN g.black_participant_id = $2 AND g.result = '1-0' THEN '0'
				WHEN g.black_participant_id = $2 AND g.result = '1/2-1/2' THEN '='
				WHEN g.black_participant_id = $2 AND g.result = '0-1F' THEN '+'
				WHEN g.black_participant_id = $2 AND g.result = '1-0F' THEN '-'
				ELSE ''
			END as score
		FROM tournament_rounds tr
		LEFT JOIN games g ON g.round_id = tr.id
			AND (g.white_participant_id = $2 OR g.black_participant_id = $2)
		LEFT JOIN tournament_participants opp_tp ON
			(g.white_participant_id = $2 AND opp_tp.id = g.black_participant_id) OR
			(g.black_participant_id = $2 AND opp_tp.id = g.white_participant_id)
		WHERE tr.tournament_id = $1
		ORDER BY tr.round_number
	`, tournamentID, participantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}
	defer rows.Close()

	var results []RoundResult
	for rows.Next() {
		var r RoundResult
		err := rows.Scan(
			&r.RoundNumber,
			&r.OpponentRank,
			&r.Color,
			&r.Score,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// GetGamesExportData retrieves all games for export
func (r *Repository) GetGamesExportData(ctx context.Context, tournamentID uuid.UUID, roundNumber *int) ([]GameExport, error) {
	query := `
		SELECT
			g.id,
			t.id,
			tr.round_number,
			g.board_number,
			t.name,
			COALESCE(t.venue_city, '') || ', ' || COALESCE(t.venue_country, ''),
			t.start_date,
			COALESCE(wp.first_name, '') || ', ' || COALESCE(wp.last_name, ''),
			COALESCE(bp.first_name, '') || ', ' || COALESCE(bp.last_name, ''),
			g.result,
			wtp.starting_rating,
			btp.starting_rating,
			wp.fide_title,
			bp.fide_title,
			wp.fide_id,
			bp.fide_id,
			t.time_control,
			COALESCE(g.pgn, ''),
			g.created_at
		FROM games g
		JOIN tournament_rounds tr ON tr.id = g.round_id
		JOIN tournaments t ON t.id = g.tournament_id
		LEFT JOIN tournament_participants wtp ON wtp.id = g.white_participant_id
		LEFT JOIN tournament_participants btp ON btp.id = g.black_participant_id
		LEFT JOIN players wp ON wp.id = wtp.player_id
		LEFT JOIN players bp ON bp.id = btp.player_id
		WHERE g.tournament_id = $1
	`

	args := []interface{}{tournamentID}

	if roundNumber != nil {
		query += " AND tr.round_number = $2"
		args = append(args, *roundNumber)
	}

	query += " ORDER BY tr.round_number, g.board_number"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	var games []GameExport
	for rows.Next() {
		var g GameExport
		var tID uuid.UUID
		var roundNum, boardNum int
		var event, site string
		var startDate time.Time
		var white, black string
		var result *string
		var whiteElo, blackElo *int
		var whiteTitle, blackTitle *string
		var whiteFideID, blackFideID *string
		var timeControl string
		var pgn string

		err := rows.Scan(
			&g.ID,
			&tID,
			&roundNum,
			&boardNum,
			&event,
			&site,
			&startDate,
			&white,
			&black,
			&result,
			&whiteElo,
			&blackElo,
			&whiteTitle,
			&blackTitle,
			&whiteFideID,
			&blackFideID,
			&timeControl,
			&pgn,
			&g.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}

		g.TournamentID = &tID
		g.RoundNumber = &roundNum
		g.BoardNumber = &boardNum
		g.Header = PGNHeader{
			Event:       event,
			Site:        site,
			Date:        startDate.Format("2006.01.02"),
			Round:       fmt.Sprintf("%d", roundNum),
			White:       white,
			Black:       black,
			Result:      ResultToPGN(getStringValue(result)),
			WhiteElo:    whiteElo,
			BlackElo:    blackElo,
			WhiteTitle:  whiteTitle,
			BlackTitle:  blackTitle,
			WhiteFideID: whiteFideID,
			BlackFideID: blackFideID,
			TimeControl: timeControl,
		}
		g.Moves = pgn

		games = append(games, g)
	}

	return games, rows.Err()
}

// GetSingleGameExportData retrieves a single game for export
func (r *Repository) GetSingleGameExportData(ctx context.Context, gameID uuid.UUID) (*GameExport, error) {
	var g GameExport
	var tID *uuid.UUID
	var roundNum, boardNum *int
	var event, site string
	var startDate string
	var white, black string
	var result *string
	var whiteElo, blackElo *int
	var whiteTitle, blackTitle *string
	var whiteFideID, blackFideID *string
	var timeControl string
	var pgn string

	err := r.db.QueryRow(ctx, `
		SELECT
			g.id,
			t.id,
			tr.round_number,
			g.board_number,
			COALESCE(t.name, 'Chess Game'),
			COALESCE(t.venue_city, '') || ', ' || COALESCE(t.venue_country, ''),
			COALESCE(t.start_date::text, to_char(g.created_at, 'YYYY.MM.DD')),
			COALESCE(wp.first_name, 'White') || ', ' || COALESCE(wp.last_name, 'Player'),
			COALESCE(bp.first_name, 'Black') || ', ' || COALESCE(bp.last_name, 'Player'),
			g.result,
			wtp.starting_rating,
			btp.starting_rating,
			wp.fide_title,
			bp.fide_title,
			wp.fide_id,
			bp.fide_id,
			COALESCE(t.time_control, ''),
			COALESCE(g.pgn, ''),
			g.created_at
		FROM games g
		LEFT JOIN tournament_rounds tr ON tr.id = g.round_id
		LEFT JOIN tournaments t ON t.id = g.tournament_id
		LEFT JOIN tournament_participants wtp ON wtp.id = g.white_participant_id
		LEFT JOIN tournament_participants btp ON btp.id = g.black_participant_id
		LEFT JOIN players wp ON wp.id = wtp.player_id
		LEFT JOIN players bp ON bp.id = btp.player_id
		WHERE g.id = $1
	`, gameID).Scan(
		&g.ID,
		&tID,
		&roundNum,
		&boardNum,
		&event,
		&site,
		&startDate,
		&white,
		&black,
		&result,
		&whiteElo,
		&blackElo,
		&whiteTitle,
		&blackTitle,
		&whiteFideID,
		&blackFideID,
		&timeControl,
		&pgn,
		&g.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	g.TournamentID = tID
	g.RoundNumber = roundNum
	g.BoardNumber = boardNum
	g.Header = PGNHeader{
		Event:       event,
		Site:        site,
		Date:        startDate,
		Round:       formatRound(roundNum),
		White:       white,
		Black:       black,
		Result:      ResultToPGN(getStringValue(result)),
		WhiteElo:    whiteElo,
		BlackElo:    blackElo,
		WhiteTitle:  whiteTitle,
		BlackTitle:  blackTitle,
		WhiteFideID: whiteFideID,
		BlackFideID: blackFideID,
		TimeControl: timeControl,
	}
	g.Moves = pgn

	return &g, nil
}

// GetRoundPairings retrieves pairings for a specific round
func (r *Repository) GetRoundPairings(ctx context.Context, tournamentID uuid.UUID, roundNumber int) ([]PairingGame, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			g.board_number,
			COALESCE(wp.first_name, '') || ' ' || COALESCE(wp.last_name, ''),
			COALESCE(wtp.starting_rating, 0),
			COALESCE(bp.first_name, '') || ' ' || COALESCE(bp.last_name, ''),
			COALESCE(btp.starting_rating, 0),
			COALESCE(g.result, '')
		FROM games g
		JOIN tournament_rounds tr ON tr.id = g.round_id
		LEFT JOIN tournament_participants wtp ON wtp.id = g.white_participant_id
		LEFT JOIN tournament_participants btp ON btp.id = g.black_participant_id
		LEFT JOIN players wp ON wp.id = wtp.player_id
		LEFT JOIN players bp ON bp.id = btp.player_id
		WHERE g.tournament_id = $1 AND tr.round_number = $2
		ORDER BY g.board_number
	`, tournamentID, roundNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query pairings: %w", err)
	}
	defer rows.Close()

	var pairings []PairingGame
	for rows.Next() {
		var p PairingGame
		err := rows.Scan(
			&p.BoardNumber,
			&p.WhiteName,
			&p.WhiteRating,
			&p.BlackName,
			&p.BlackRating,
			&p.Result,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pairing: %w", err)
		}
		pairings = append(pairings, p)
	}

	return pairings, rows.Err()
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatRound(r *int) string {
	if r == nil {
		return "?"
	}
	return fmt.Sprintf("%d", *r)
}
