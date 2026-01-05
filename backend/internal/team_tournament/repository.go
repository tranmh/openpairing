package team_tournament

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides access to team tournament data
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new team tournament repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetSettings retrieves team tournament settings
func (r *Repository) GetSettings(ctx context.Context, tournamentID uuid.UUID) (*TeamTournamentSettings, error) {
	query := `
		SELECT tournament_id, boards_per_match, lineup_submission_deadline_minutes,
			   allow_late_lineup, enforce_rating_order, rating_order_tolerance,
			   match_win_points, match_draw_points, match_loss_points,
			   board_win_points, board_draw_points, board_loss_points,
			   default_board_result, team_tiebreaks
		FROM team_tournament_settings
		WHERE tournament_id = $1`

	var settings TeamTournamentSettings
	var tiebreaksJSON []byte

	err := r.pool.QueryRow(ctx, query, tournamentID).Scan(
		&settings.TournamentID, &settings.BoardsPerMatch, &settings.LineupSubmissionDeadlineMinutes,
		&settings.AllowLateLineup, &settings.EnforceRatingOrder, &settings.RatingOrderTolerance,
		&settings.MatchWinPoints, &settings.MatchDrawPoints, &settings.MatchLossPoints,
		&settings.BoardWinPoints, &settings.BoardDrawPoints, &settings.BoardLossPoints,
		&settings.DefaultBoardResult, &tiebreaksJSON,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("team tournament settings not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(tiebreaksJSON, &settings.TeamTiebreaks); err != nil {
		settings.TeamTiebreaks = []string{"board_points", "buchholz"}
	}

	return &settings, nil
}

// CreateSettings creates team tournament settings
func (r *Repository) CreateSettings(ctx context.Context, settings *TeamTournamentSettings) error {
	tiebreaksJSON, _ := json.Marshal(settings.TeamTiebreaks)

	query := `
		INSERT INTO team_tournament_settings (
			tournament_id, boards_per_match, lineup_submission_deadline_minutes,
			allow_late_lineup, enforce_rating_order, rating_order_tolerance,
			match_win_points, match_draw_points, match_loss_points,
			board_win_points, board_draw_points, board_loss_points,
			default_board_result, team_tiebreaks
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tournament_id) DO UPDATE SET
			boards_per_match = EXCLUDED.boards_per_match,
			lineup_submission_deadline_minutes = EXCLUDED.lineup_submission_deadline_minutes,
			allow_late_lineup = EXCLUDED.allow_late_lineup,
			enforce_rating_order = EXCLUDED.enforce_rating_order,
			rating_order_tolerance = EXCLUDED.rating_order_tolerance,
			match_win_points = EXCLUDED.match_win_points,
			match_draw_points = EXCLUDED.match_draw_points,
			match_loss_points = EXCLUDED.match_loss_points,
			board_win_points = EXCLUDED.board_win_points,
			board_draw_points = EXCLUDED.board_draw_points,
			board_loss_points = EXCLUDED.board_loss_points,
			default_board_result = EXCLUDED.default_board_result,
			team_tiebreaks = EXCLUDED.team_tiebreaks`

	_, err := r.pool.Exec(ctx, query,
		settings.TournamentID, settings.BoardsPerMatch, settings.LineupSubmissionDeadlineMinutes,
		settings.AllowLateLineup, settings.EnforceRatingOrder, settings.RatingOrderTolerance,
		settings.MatchWinPoints, settings.MatchDrawPoints, settings.MatchLossPoints,
		settings.BoardWinPoints, settings.BoardDrawPoints, settings.BoardLossPoints,
		settings.DefaultBoardResult, tiebreaksJSON,
	)
	return err
}

// RegisterTeam registers a team for a tournament
func (r *Repository) RegisterTeam(ctx context.Context, participant *TeamParticipant) error {
	query := `
		INSERT INTO team_tournament_participants (
			id, tournament_id, team_id, registration_status, registered_at,
			start_rank, seed_rating, match_points, board_points, buchholz,
			sonneborn_berger, lineup_locked
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		participant.ID, participant.TournamentID, participant.TeamID,
		participant.RegistrationStatus, participant.RegisteredAt,
		participant.StartRank, participant.SeedRating, participant.MatchPoints,
		participant.BoardPoints, participant.Buchholz, participant.SonnebornBerger,
		participant.LineupLocked,
	)
	return err
}

// GetTeamParticipant retrieves a team participant by ID
func (r *Repository) GetTeamParticipant(ctx context.Context, id uuid.UUID) (*TeamParticipant, error) {
	query := `
		SELECT tp.id, tp.tournament_id, tp.team_id, tp.registration_status, tp.registered_at,
			   tp.start_rank, tp.seed_rating, tp.match_points, tp.board_points, tp.buchholz,
			   tp.sonneborn_berger, tp.lineup_locked,
			   t.name as team_name, t.short_name as team_short_name,
			   COALESCE(c.name, '') as club_name
		FROM team_tournament_participants tp
		JOIN teams t ON t.id = tp.team_id
		LEFT JOIN clubs c ON c.id = t.club_id
		WHERE tp.id = $1`

	var p TeamParticipant
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.TournamentID, &p.TeamID, &p.RegistrationStatus, &p.RegisteredAt,
		&p.StartRank, &p.SeedRating, &p.MatchPoints, &p.BoardPoints, &p.Buchholz,
		&p.SonnebornBerger, &p.LineupLocked,
		&p.TeamName, &p.TeamShortName, &p.ClubName,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("team participant not found")
	}
	return &p, err
}

// ListTeamParticipants lists all team participants for a tournament
func (r *Repository) ListTeamParticipants(ctx context.Context, tournamentID uuid.UUID) ([]TeamParticipant, error) {
	query := `
		SELECT tp.id, tp.tournament_id, tp.team_id, tp.registration_status, tp.registered_at,
			   tp.start_rank, tp.seed_rating, tp.match_points, tp.board_points, tp.buchholz,
			   tp.sonneborn_berger, tp.lineup_locked,
			   t.name as team_name, t.short_name as team_short_name,
			   COALESCE(c.name, '') as club_name
		FROM team_tournament_participants tp
		JOIN teams t ON t.id = tp.team_id
		LEFT JOIN clubs c ON c.id = t.club_id
		WHERE tp.tournament_id = $1
		ORDER BY tp.match_points DESC, tp.board_points DESC, tp.start_rank ASC`

	rows, err := r.pool.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []TeamParticipant
	for rows.Next() {
		var p TeamParticipant
		if err := rows.Scan(
			&p.ID, &p.TournamentID, &p.TeamID, &p.RegistrationStatus, &p.RegisteredAt,
			&p.StartRank, &p.SeedRating, &p.MatchPoints, &p.BoardPoints, &p.Buchholz,
			&p.SonnebornBerger, &p.LineupLocked,
			&p.TeamName, &p.TeamShortName, &p.ClubName,
		); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

// UpdateTeamParticipant updates a team participant
func (r *Repository) UpdateTeamParticipant(ctx context.Context, p *TeamParticipant) error {
	query := `
		UPDATE team_tournament_participants SET
			registration_status = $2, start_rank = $3, seed_rating = $4,
			match_points = $5, board_points = $6, buchholz = $7,
			sonneborn_berger = $8, lineup_locked = $9
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query,
		p.ID, p.RegistrationStatus, p.StartRank, p.SeedRating,
		p.MatchPoints, p.BoardPoints, p.Buchholz, p.SonnebornBerger, p.LineupLocked,
	)
	return err
}

// WithdrawTeam withdraws a team from a tournament
func (r *Repository) WithdrawTeam(ctx context.Context, participantID uuid.UUID) error {
	query := `UPDATE team_tournament_participants SET registration_status = 'withdrawn' WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, participantID)
	return err
}

// SaveLineup saves or updates a team lineup
func (r *Repository) SaveLineup(ctx context.Context, lineup *TeamLineup) error {
	lineupJSON, err := json.Marshal(lineup.Lineup)
	if err != nil {
		return err
	}

	validationJSON, _ := json.Marshal(lineup.ValidationErrors)

	query := `
		INSERT INTO team_lineups (
			id, tournament_id, round_id, team_participant_id, lineup,
			is_valid, validation_errors, submitted_at, submitted_by, is_locked
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (round_id, team_participant_id) DO UPDATE SET
			lineup = EXCLUDED.lineup,
			is_valid = EXCLUDED.is_valid,
			validation_errors = EXCLUDED.validation_errors,
			submitted_at = EXCLUDED.submitted_at,
			submitted_by = EXCLUDED.submitted_by,
			is_locked = EXCLUDED.is_locked`

	_, err = r.pool.Exec(ctx, query,
		lineup.ID, lineup.TournamentID, lineup.RoundID, lineup.TeamParticipantID,
		lineupJSON, lineup.IsValid, validationJSON, lineup.SubmittedAt, lineup.SubmittedBy,
		lineup.IsLocked,
	)
	return err
}

// GetLineup retrieves a team lineup for a round
func (r *Repository) GetLineup(ctx context.Context, roundID, teamParticipantID uuid.UUID) (*TeamLineup, error) {
	query := `
		SELECT id, tournament_id, round_id, team_participant_id, lineup,
			   is_valid, validation_errors, submitted_at, submitted_by, is_locked
		FROM team_lineups
		WHERE round_id = $1 AND team_participant_id = $2`

	var lineup TeamLineup
	var lineupJSON, validationJSON []byte

	err := r.pool.QueryRow(ctx, query, roundID, teamParticipantID).Scan(
		&lineup.ID, &lineup.TournamentID, &lineup.RoundID, &lineup.TeamParticipantID,
		&lineupJSON, &lineup.IsValid, &validationJSON, &lineup.SubmittedAt,
		&lineup.SubmittedBy, &lineup.IsLocked,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("lineup not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(lineupJSON, &lineup.Lineup)
	json.Unmarshal(validationJSON, &lineup.ValidationErrors)

	return &lineup, nil
}

// ListLineupsForRound lists all lineups for a round
func (r *Repository) ListLineupsForRound(ctx context.Context, roundID uuid.UUID) ([]TeamLineup, error) {
	query := `
		SELECT id, tournament_id, round_id, team_participant_id, lineup,
			   is_valid, validation_errors, submitted_at, submitted_by, is_locked
		FROM team_lineups
		WHERE round_id = $1`

	rows, err := r.pool.Query(ctx, query, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineups []TeamLineup
	for rows.Next() {
		var lineup TeamLineup
		var lineupJSON, validationJSON []byte

		if err := rows.Scan(
			&lineup.ID, &lineup.TournamentID, &lineup.RoundID, &lineup.TeamParticipantID,
			&lineupJSON, &lineup.IsValid, &validationJSON, &lineup.SubmittedAt,
			&lineup.SubmittedBy, &lineup.IsLocked,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(lineupJSON, &lineup.Lineup)
		json.Unmarshal(validationJSON, &lineup.ValidationErrors)
		lineups = append(lineups, lineup)
	}
	return lineups, rows.Err()
}

// LockLineup locks a team's lineup
func (r *Repository) LockLineup(ctx context.Context, roundID, teamParticipantID uuid.UUID) error {
	query := `UPDATE team_lineups SET is_locked = true WHERE round_id = $1 AND team_participant_id = $2`
	_, err := r.pool.Exec(ctx, query, roundID, teamParticipantID)
	return err
}

// CreateTeamMatch creates a team match
func (r *Repository) CreateTeamMatch(ctx context.Context, match *TeamMatch) error {
	query := `
		INSERT INTO team_matches (
			id, tournament_id, round_id, match_number, home_team_id, away_team_id,
			home_match_points, away_match_points, home_board_points, away_board_points,
			status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		match.ID, match.TournamentID, match.RoundID, match.MatchNumber,
		match.HomeTeamID, match.AwayTeamID, match.HomeMatchPoints, match.AwayMatchPoints,
		match.HomeBoardPoints, match.AwayBoardPoints, match.Status, match.CreatedAt,
	)
	return err
}

// GetTeamMatch retrieves a team match by ID
func (r *Repository) GetTeamMatch(ctx context.Context, id uuid.UUID) (*TeamMatch, error) {
	query := `
		SELECT tm.id, tm.tournament_id, tm.round_id, tm.match_number,
			   tm.home_team_id, tm.away_team_id, tm.home_match_points, tm.away_match_points,
			   tm.home_board_points, tm.away_board_points, tm.status, tm.created_at,
			   COALESCE(ht.name, '') as home_team_name,
			   COALESCE(at.name, '') as away_team_name
		FROM team_matches tm
		LEFT JOIN team_tournament_participants htp ON htp.id = tm.home_team_id
		LEFT JOIN teams ht ON ht.id = htp.team_id
		LEFT JOIN team_tournament_participants atp ON atp.id = tm.away_team_id
		LEFT JOIN teams at ON at.id = atp.team_id
		WHERE tm.id = $1`

	var m TeamMatch
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.TournamentID, &m.RoundID, &m.MatchNumber,
		&m.HomeTeamID, &m.AwayTeamID, &m.HomeMatchPoints, &m.AwayMatchPoints,
		&m.HomeBoardPoints, &m.AwayBoardPoints, &m.Status, &m.CreatedAt,
		&m.HomeTeamName, &m.AwayTeamName,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("team match not found")
	}
	return &m, err
}

// ListTeamMatches lists all team matches for a tournament
func (r *Repository) ListTeamMatches(ctx context.Context, tournamentID uuid.UUID) ([]TeamMatch, error) {
	query := `
		SELECT tm.id, tm.tournament_id, tm.round_id, tm.match_number,
			   tm.home_team_id, tm.away_team_id, tm.home_match_points, tm.away_match_points,
			   tm.home_board_points, tm.away_board_points, tm.status, tm.created_at,
			   COALESCE(ht.name, '') as home_team_name,
			   COALESCE(at.name, '') as away_team_name
		FROM team_matches tm
		LEFT JOIN team_tournament_participants htp ON htp.id = tm.home_team_id
		LEFT JOIN teams ht ON ht.id = htp.team_id
		LEFT JOIN team_tournament_participants atp ON atp.id = tm.away_team_id
		LEFT JOIN teams at ON at.id = atp.team_id
		WHERE tm.tournament_id = $1
		ORDER BY tm.round_id, tm.match_number`

	rows, err := r.pool.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []TeamMatch
	for rows.Next() {
		var m TeamMatch
		if err := rows.Scan(
			&m.ID, &m.TournamentID, &m.RoundID, &m.MatchNumber,
			&m.HomeTeamID, &m.AwayTeamID, &m.HomeMatchPoints, &m.AwayMatchPoints,
			&m.HomeBoardPoints, &m.AwayBoardPoints, &m.Status, &m.CreatedAt,
			&m.HomeTeamName, &m.AwayTeamName,
		); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// ListTeamMatchesByRound lists team matches for a specific round
func (r *Repository) ListTeamMatchesByRound(ctx context.Context, roundID uuid.UUID) ([]TeamMatch, error) {
	query := `
		SELECT tm.id, tm.tournament_id, tm.round_id, tm.match_number,
			   tm.home_team_id, tm.away_team_id, tm.home_match_points, tm.away_match_points,
			   tm.home_board_points, tm.away_board_points, tm.status, tm.created_at,
			   COALESCE(ht.name, '') as home_team_name,
			   COALESCE(at.name, '') as away_team_name
		FROM team_matches tm
		LEFT JOIN team_tournament_participants htp ON htp.id = tm.home_team_id
		LEFT JOIN teams ht ON ht.id = htp.team_id
		LEFT JOIN team_tournament_participants atp ON atp.id = tm.away_team_id
		LEFT JOIN teams at ON at.id = atp.team_id
		WHERE tm.round_id = $1
		ORDER BY tm.match_number`

	rows, err := r.pool.Query(ctx, query, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []TeamMatch
	for rows.Next() {
		var m TeamMatch
		if err := rows.Scan(
			&m.ID, &m.TournamentID, &m.RoundID, &m.MatchNumber,
			&m.HomeTeamID, &m.AwayTeamID, &m.HomeMatchPoints, &m.AwayMatchPoints,
			&m.HomeBoardPoints, &m.AwayBoardPoints, &m.Status, &m.CreatedAt,
			&m.HomeTeamName, &m.AwayTeamName,
		); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// UpdateTeamMatch updates a team match
func (r *Repository) UpdateTeamMatch(ctx context.Context, match *TeamMatch) error {
	query := `
		UPDATE team_matches SET
			home_match_points = $2, away_match_points = $3,
			home_board_points = $4, away_board_points = $5, status = $6
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query,
		match.ID, match.HomeMatchPoints, match.AwayMatchPoints,
		match.HomeBoardPoints, match.AwayBoardPoints, match.Status,
	)
	return err
}

// CreateTeamMatchBoard creates a board entry for a team match
func (r *Repository) CreateTeamMatchBoard(ctx context.Context, board *TeamMatchBoard) error {
	query := `
		INSERT INTO team_match_boards (
			id, team_match_id, game_id, board_number,
			home_player_id, away_player_id, home_rating, away_rating,
			home_score, away_score, result
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.pool.Exec(ctx, query,
		board.ID, board.TeamMatchID, board.GameID, board.BoardNumber,
		board.HomePlayerID, board.AwayPlayerID, board.HomeRating, board.AwayRating,
		board.HomeScore, board.AwayScore, board.Result,
	)
	return err
}

// GetTeamMatchBoards retrieves all boards for a team match
func (r *Repository) GetTeamMatchBoards(ctx context.Context, teamMatchID uuid.UUID) ([]TeamMatchBoard, error) {
	query := `
		SELECT tmb.id, tmb.team_match_id, tmb.game_id, tmb.board_number,
			   tmb.home_player_id, tmb.away_player_id, tmb.home_rating, tmb.away_rating,
			   tmb.home_score, tmb.away_score, tmb.result,
			   COALESCE(hp.first_name || ' ' || hp.last_name, '') as home_player_name,
			   COALESCE(ap.first_name || ' ' || ap.last_name, '') as away_player_name
		FROM team_match_boards tmb
		LEFT JOIN players hp ON hp.id = tmb.home_player_id
		LEFT JOIN players ap ON ap.id = tmb.away_player_id
		WHERE tmb.team_match_id = $1
		ORDER BY tmb.board_number`

	rows, err := r.pool.Query(ctx, query, teamMatchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []TeamMatchBoard
	for rows.Next() {
		var b TeamMatchBoard
		if err := rows.Scan(
			&b.ID, &b.TeamMatchID, &b.GameID, &b.BoardNumber,
			&b.HomePlayerID, &b.AwayPlayerID, &b.HomeRating, &b.AwayRating,
			&b.HomeScore, &b.AwayScore, &b.Result,
			&b.HomePlayerName, &b.AwayPlayerName,
		); err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	return boards, rows.Err()
}

// UpdateTeamMatchBoard updates a board result
func (r *Repository) UpdateTeamMatchBoard(ctx context.Context, board *TeamMatchBoard) error {
	query := `
		UPDATE team_match_boards SET
			home_score = $2, away_score = $3, result = $4
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query,
		board.ID, board.HomeScore, board.AwayScore, board.Result,
	)
	return err
}

// GetBoardByMatchAndNumber retrieves a specific board from a match
func (r *Repository) GetBoardByMatchAndNumber(ctx context.Context, teamMatchID uuid.UUID, boardNumber int) (*TeamMatchBoard, error) {
	query := `
		SELECT id, team_match_id, game_id, board_number,
			   home_player_id, away_player_id, home_rating, away_rating,
			   home_score, away_score, result
		FROM team_match_boards
		WHERE team_match_id = $1 AND board_number = $2`

	var b TeamMatchBoard
	err := r.pool.QueryRow(ctx, query, teamMatchID, boardNumber).Scan(
		&b.ID, &b.TeamMatchID, &b.GameID, &b.BoardNumber,
		&b.HomePlayerID, &b.AwayPlayerID, &b.HomeRating, &b.AwayRating,
		&b.HomeScore, &b.AwayScore, &b.Result,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("board not found")
	}
	return &b, err
}
