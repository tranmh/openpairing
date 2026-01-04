package team

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles team database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new team repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Team CRUD

// CreateTeam creates a new team
func (r *Repository) CreateTeam(ctx context.Context, team *Team) error {
	query := `
		INSERT INTO teams (id, club_id, name, short_name, division, region, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`
	_, err := r.db.Exec(ctx, query, team.ID, team.ClubID, team.Name, team.ShortName, team.Division, team.Region)
	return err
}

// GetTeamByID retrieves a team by ID
func (r *Repository) GetTeamByID(ctx context.Context, id uuid.UUID) (*Team, error) {
	query := `
		SELECT id, club_id, name, short_name, division, region, created_at, updated_at
		FROM teams WHERE id = $1
	`
	var team Team
	err := r.db.QueryRow(ctx, query, id).Scan(
		&team.ID, &team.ClubID, &team.Name, &team.ShortName,
		&team.Division, &team.Region, &team.CreatedAt, &team.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &team, err
}

// GetTeamsByClubID retrieves all teams for a club
func (r *Repository) GetTeamsByClubID(ctx context.Context, clubID uuid.UUID) ([]Team, error) {
	query := `
		SELECT id, club_id, name, short_name, division, region, created_at, updated_at
		FROM teams WHERE club_id = $1
		ORDER BY name
	`
	rows, err := r.db.Query(ctx, query, clubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(
			&team.ID, &team.ClubID, &team.Name, &team.ShortName,
			&team.Division, &team.Region, &team.CreatedAt, &team.UpdatedAt,
		); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, nil
}

// UpdateTeam updates a team
func (r *Repository) UpdateTeam(ctx context.Context, team *Team) error {
	query := `
		UPDATE teams
		SET name = $2, short_name = $3, division = $4, region = $5, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, team.ID, team.Name, team.ShortName, team.Division, team.Region)
	return err
}

// DeleteTeam deletes a team
func (r *Repository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM teams WHERE id = $1", id)
	return err
}

// Team Members

// AddTeamMember adds a player to a team roster
func (r *Repository) AddTeamMember(ctx context.Context, member *TeamMember) error {
	query := `
		INSERT INTO team_members (id, team_id, player_id, board_number, is_captain, is_reserve, valid_from, valid_until, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`
	_, err := r.db.Exec(ctx, query,
		member.ID, member.TeamID, member.PlayerID, member.BoardNumber,
		member.IsCaptain, member.IsReserve, member.ValidFrom, member.ValidUntil,
	)
	return err
}

// GetTeamMembers retrieves all members of a team
func (r *Repository) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]TeamMemberWithPlayer, error) {
	query := `
		SELECT tm.id, tm.team_id, tm.player_id, tm.board_number, tm.is_captain, tm.is_reserve,
		       tm.valid_from, tm.valid_until, tm.created_at,
		       COALESCE(p.first_name || ' ' || p.last_name, '') as player_name,
		       COALESCE(p.dwz, p.fide_standard, 0) as player_rating,
		       COALESCE(p.fide_title, '') as player_title
		FROM team_members tm
		LEFT JOIN players p ON tm.player_id = p.id
		WHERE tm.team_id = $1
		ORDER BY tm.board_number NULLS LAST, player_rating DESC
	`
	rows, err := r.db.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []TeamMemberWithPlayer
	for rows.Next() {
		var m TeamMemberWithPlayer
		if err := rows.Scan(
			&m.ID, &m.TeamID, &m.PlayerID, &m.BoardNumber, &m.IsCaptain, &m.IsReserve,
			&m.ValidFrom, &m.ValidUntil, &m.CreatedAt,
			&m.PlayerName, &m.PlayerRating, &m.PlayerTitle,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// RemoveTeamMember removes a player from a team roster
func (r *Repository) RemoveTeamMember(ctx context.Context, teamID, playerID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM team_members WHERE team_id = $1 AND player_id = $2", teamID, playerID)
	return err
}

// UpdateTeamMember updates a team member
func (r *Repository) UpdateTeamMember(ctx context.Context, member *TeamMember) error {
	query := `
		UPDATE team_members
		SET board_number = $3, is_captain = $4, is_reserve = $5, valid_from = $6, valid_until = $7
		WHERE team_id = $1 AND player_id = $2
	`
	_, err := r.db.Exec(ctx, query,
		member.TeamID, member.PlayerID, member.BoardNumber,
		member.IsCaptain, member.IsReserve, member.ValidFrom, member.ValidUntil,
	)
	return err
}

// Team Tournament Participants

// RegisterTeamForTournament registers a team for a tournament
func (r *Repository) RegisterTeamForTournament(ctx context.Context, participant *TeamTournamentParticipant) error {
	query := `
		INSERT INTO team_tournament_participants (id, tournament_id, team_id, registration_status, registered_at, start_rank, seed_rating)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		participant.ID, participant.TournamentID, participant.TeamID,
		participant.RegistrationStatus, participant.StartRank, participant.SeedRating,
	)
	return err
}

// GetTeamParticipants retrieves all team participants for a tournament
func (r *Repository) GetTeamParticipants(ctx context.Context, tournamentID uuid.UUID) ([]TeamTournamentParticipantWithTeam, error) {
	query := `
		SELECT ttp.id, ttp.tournament_id, ttp.team_id, ttp.registration_status, ttp.registered_at,
		       ttp.start_rank, ttp.seed_rating, ttp.match_points, ttp.board_points, ttp.buchholz, ttp.lineup_locked,
		       t.name as team_name, COALESCE(t.short_name, '') as team_short_name
		FROM team_tournament_participants ttp
		JOIN teams t ON ttp.team_id = t.id
		WHERE ttp.tournament_id = $1
		ORDER BY ttp.match_points DESC, ttp.board_points DESC
	`
	rows, err := r.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []TeamTournamentParticipantWithTeam
	for rows.Next() {
		var p TeamTournamentParticipantWithTeam
		if err := rows.Scan(
			&p.ID, &p.TournamentID, &p.TeamID, &p.RegistrationStatus, &p.RegisteredAt,
			&p.StartRank, &p.SeedRating, &p.MatchPoints, &p.BoardPoints, &p.Buchholz, &p.LineupLocked,
			&p.TeamName, &p.TeamShortName,
		); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, nil
}

// GetTeamParticipantByID retrieves a team participant by ID
func (r *Repository) GetTeamParticipantByID(ctx context.Context, id uuid.UUID) (*TeamTournamentParticipantWithTeam, error) {
	query := `
		SELECT ttp.id, ttp.tournament_id, ttp.team_id, ttp.registration_status, ttp.registered_at,
		       ttp.start_rank, ttp.seed_rating, ttp.match_points, ttp.board_points, ttp.buchholz, ttp.lineup_locked,
		       t.name as team_name, COALESCE(t.short_name, '') as team_short_name
		FROM team_tournament_participants ttp
		JOIN teams t ON ttp.team_id = t.id
		WHERE ttp.id = $1
	`
	var p TeamTournamentParticipantWithTeam
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.TournamentID, &p.TeamID, &p.RegistrationStatus, &p.RegisteredAt,
		&p.StartRank, &p.SeedRating, &p.MatchPoints, &p.BoardPoints, &p.Buchholz, &p.LineupLocked,
		&p.TeamName, &p.TeamShortName,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

// UpdateTeamParticipantStandings updates a team's standings
func (r *Repository) UpdateTeamParticipantStandings(ctx context.Context, id uuid.UUID, matchPoints, boardPoints, buchholz float64) error {
	query := `
		UPDATE team_tournament_participants
		SET match_points = $2, board_points = $3, buchholz = $4
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, matchPoints, boardPoints, buchholz)
	return err
}

// WithdrawTeamFromTournament withdraws a team from a tournament
func (r *Repository) WithdrawTeamFromTournament(ctx context.Context, participantID uuid.UUID) error {
	query := `
		UPDATE team_tournament_participants
		SET registration_status = $2
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, participantID, TeamRegistrationStatusWithdrawn)
	return err
}

// Team Lineups

// SaveLineup saves or updates a team lineup for a round
func (r *Repository) SaveLineup(ctx context.Context, lineup *TeamLineup) error {
	lineupJSON, err := json.Marshal(lineup.Lineup)
	if err != nil {
		return fmt.Errorf("failed to marshal lineup: %w", err)
	}
	errorsJSON, _ := json.Marshal(lineup.ValidationErrors)

	query := `
		INSERT INTO team_lineups (id, tournament_id, round_id, team_participant_id, lineup, is_valid, validation_errors, submitted_at, submitted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (round_id, team_participant_id)
		DO UPDATE SET lineup = $5, is_valid = $6, validation_errors = $7, submitted_at = $8, submitted_by = $9
	`
	_, err = r.db.Exec(ctx, query,
		lineup.ID, lineup.TournamentID, lineup.RoundID, lineup.TeamParticipantID,
		lineupJSON, lineup.IsValid, errorsJSON, lineup.SubmittedAt, lineup.SubmittedBy,
	)
	return err
}

// GetLineup retrieves a team's lineup for a round
func (r *Repository) GetLineup(ctx context.Context, roundID, teamParticipantID uuid.UUID) (*TeamLineup, error) {
	query := `
		SELECT id, tournament_id, round_id, team_participant_id, lineup, is_valid, validation_errors, submitted_at, submitted_by
		FROM team_lineups
		WHERE round_id = $1 AND team_participant_id = $2
	`
	var lineup TeamLineup
	var lineupJSON, errorsJSON []byte
	err := r.db.QueryRow(ctx, query, roundID, teamParticipantID).Scan(
		&lineup.ID, &lineup.TournamentID, &lineup.RoundID, &lineup.TeamParticipantID,
		&lineupJSON, &lineup.IsValid, &errorsJSON, &lineup.SubmittedAt, &lineup.SubmittedBy,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(lineupJSON, &lineup.Lineup)
	json.Unmarshal(errorsJSON, &lineup.ValidationErrors)
	return &lineup, nil
}

// GetLineupsForRound retrieves all lineups for a round
func (r *Repository) GetLineupsForRound(ctx context.Context, roundID uuid.UUID) ([]TeamLineup, error) {
	query := `
		SELECT id, tournament_id, round_id, team_participant_id, lineup, is_valid, validation_errors, submitted_at, submitted_by
		FROM team_lineups
		WHERE round_id = $1
	`
	rows, err := r.db.Query(ctx, query, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineups []TeamLineup
	for rows.Next() {
		var lineup TeamLineup
		var lineupJSON, errorsJSON []byte
		if err := rows.Scan(
			&lineup.ID, &lineup.TournamentID, &lineup.RoundID, &lineup.TeamParticipantID,
			&lineupJSON, &lineup.IsValid, &errorsJSON, &lineup.SubmittedAt, &lineup.SubmittedBy,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(lineupJSON, &lineup.Lineup)
		json.Unmarshal(errorsJSON, &lineup.ValidationErrors)
		lineups = append(lineups, lineup)
	}
	return lineups, nil
}

// Team Matches

// CreateTeamMatch creates a new team match
func (r *Repository) CreateTeamMatch(ctx context.Context, match *TeamMatch) error {
	query := `
		INSERT INTO team_matches (id, tournament_id, round_id, match_number, home_team_id, away_team_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	_, err := r.db.Exec(ctx, query,
		match.ID, match.TournamentID, match.RoundID, match.MatchNumber,
		match.HomeTeamID, match.AwayTeamID, match.Status,
	)
	return err
}

// GetTeamMatch retrieves a team match by ID
func (r *Repository) GetTeamMatch(ctx context.Context, id uuid.UUID) (*TeamMatchWithTeams, error) {
	query := `
		SELECT tm.id, tm.tournament_id, tm.round_id, tm.match_number,
		       tm.home_team_id, tm.away_team_id,
		       tm.home_match_points, tm.away_match_points,
		       tm.home_board_points, tm.away_board_points,
		       tm.status, tm.created_at,
		       COALESCE(ht.name, '') as home_team_name,
		       COALESCE(at.name, '') as away_team_name
		FROM team_matches tm
		LEFT JOIN team_tournament_participants htp ON tm.home_team_id = htp.id
		LEFT JOIN teams ht ON htp.team_id = ht.id
		LEFT JOIN team_tournament_participants atp ON tm.away_team_id = atp.id
		LEFT JOIN teams at ON atp.team_id = at.id
		WHERE tm.id = $1
	`
	var m TeamMatchWithTeams
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.TournamentID, &m.RoundID, &m.MatchNumber,
		&m.HomeTeamID, &m.AwayTeamID,
		&m.HomeMatchPoints, &m.AwayMatchPoints,
		&m.HomeBoardPoints, &m.AwayBoardPoints,
		&m.Status, &m.CreatedAt,
		&m.HomeTeamName, &m.AwayTeamName,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &m, err
}

// GetTeamMatchesForRound retrieves all team matches for a round
func (r *Repository) GetTeamMatchesForRound(ctx context.Context, roundID uuid.UUID) ([]TeamMatchWithTeams, error) {
	query := `
		SELECT tm.id, tm.tournament_id, tm.round_id, tm.match_number,
		       tm.home_team_id, tm.away_team_id,
		       tm.home_match_points, tm.away_match_points,
		       tm.home_board_points, tm.away_board_points,
		       tm.status, tm.created_at,
		       COALESCE(ht.name, '') as home_team_name,
		       COALESCE(at.name, '') as away_team_name
		FROM team_matches tm
		LEFT JOIN team_tournament_participants htp ON tm.home_team_id = htp.id
		LEFT JOIN teams ht ON htp.team_id = ht.id
		LEFT JOIN team_tournament_participants atp ON tm.away_team_id = atp.id
		LEFT JOIN teams at ON atp.team_id = at.id
		WHERE tm.round_id = $1
		ORDER BY tm.match_number
	`
	rows, err := r.db.Query(ctx, query, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []TeamMatchWithTeams
	for rows.Next() {
		var m TeamMatchWithTeams
		if err := rows.Scan(
			&m.ID, &m.TournamentID, &m.RoundID, &m.MatchNumber,
			&m.HomeTeamID, &m.AwayTeamID,
			&m.HomeMatchPoints, &m.AwayMatchPoints,
			&m.HomeBoardPoints, &m.AwayBoardPoints,
			&m.Status, &m.CreatedAt,
			&m.HomeTeamName, &m.AwayTeamName,
		); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// UpdateTeamMatchResult updates the result of a team match
func (r *Repository) UpdateTeamMatchResult(ctx context.Context, id uuid.UUID, homeMatchPoints, awayMatchPoints, homeBoardPoints, awayBoardPoints float64) error {
	query := `
		UPDATE team_matches
		SET home_match_points = $2, away_match_points = $3,
		    home_board_points = $4, away_board_points = $5,
		    status = $6
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, homeMatchPoints, awayMatchPoints, homeBoardPoints, awayBoardPoints, TeamMatchStatusCompleted)
	return err
}

// Team Match Boards

// CreateTeamMatchBoard creates a board pairing within a team match
func (r *Repository) CreateTeamMatchBoard(ctx context.Context, board *TeamMatchBoard) error {
	query := `
		INSERT INTO team_match_boards (id, team_match_id, game_id, board_number, home_player_id, away_player_id, home_rating, away_rating)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		board.ID, board.TeamMatchID, board.GameID, board.BoardNumber,
		board.HomePlayerID, board.AwayPlayerID, board.HomeRating, board.AwayRating,
	)
	return err
}

// GetTeamMatchBoards retrieves all boards for a team match
func (r *Repository) GetTeamMatchBoards(ctx context.Context, matchID uuid.UUID) ([]TeamMatchBoardWithPlayers, error) {
	query := `
		SELECT tmb.id, tmb.team_match_id, tmb.game_id, tmb.board_number,
		       tmb.home_player_id, tmb.away_player_id,
		       tmb.home_rating, tmb.away_rating,
		       tmb.home_score, tmb.away_score,
		       COALESCE(hp.first_name || ' ' || hp.last_name, '') as home_player_name,
		       COALESCE(ap.first_name || ' ' || ap.last_name, '') as away_player_name
		FROM team_match_boards tmb
		LEFT JOIN players hp ON tmb.home_player_id = hp.id
		LEFT JOIN players ap ON tmb.away_player_id = ap.id
		WHERE tmb.team_match_id = $1
		ORDER BY tmb.board_number
	`
	rows, err := r.db.Query(ctx, query, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []TeamMatchBoardWithPlayers
	for rows.Next() {
		var b TeamMatchBoardWithPlayers
		if err := rows.Scan(
			&b.ID, &b.TeamMatchID, &b.GameID, &b.BoardNumber,
			&b.HomePlayerID, &b.AwayPlayerID,
			&b.HomeRating, &b.AwayRating,
			&b.HomeScore, &b.AwayScore,
			&b.HomePlayerName, &b.AwayPlayerName,
		); err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	return boards, nil
}

// UpdateBoardResult updates the result of a board
func (r *Repository) UpdateBoardResult(ctx context.Context, boardID uuid.UUID, homeScore, awayScore float64) error {
	query := `
		UPDATE team_match_boards
		SET home_score = $2, away_score = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, boardID, homeScore, awayScore)
	return err
}

// Team Tournament Settings

// SaveTeamTournamentSettings saves team tournament settings
func (r *Repository) SaveTeamTournamentSettings(ctx context.Context, settings *TeamTournamentSettings) error {
	tiebreaksJSON, _ := json.Marshal(settings.TeamTiebreaks)
	query := `
		INSERT INTO team_tournament_settings (
			tournament_id, boards_per_match, lineup_submission_deadline_minutes,
			allow_late_lineup, enforce_rating_order, rating_order_tolerance,
			match_win_points, match_draw_points, match_loss_points,
			team_tiebreaks, board_win_points, board_draw_points, board_loss_points,
			default_board_result
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tournament_id)
		DO UPDATE SET
			boards_per_match = $2, lineup_submission_deadline_minutes = $3,
			allow_late_lineup = $4, enforce_rating_order = $5, rating_order_tolerance = $6,
			match_win_points = $7, match_draw_points = $8, match_loss_points = $9,
			team_tiebreaks = $10, board_win_points = $11, board_draw_points = $12, board_loss_points = $13,
			default_board_result = $14
	`
	_, err := r.db.Exec(ctx, query,
		settings.TournamentID, settings.BoardsPerMatch, settings.LineupSubmissionDeadlineMinutes,
		settings.AllowLateLineup, settings.EnforceRatingOrder, settings.RatingOrderTolerance,
		settings.MatchWinPoints, settings.MatchDrawPoints, settings.MatchLossPoints,
		tiebreaksJSON, settings.BoardWinPoints, settings.BoardDrawPoints, settings.BoardLossPoints,
		settings.DefaultBoardResult,
	)
	return err
}

// GetTeamTournamentSettings retrieves team tournament settings
func (r *Repository) GetTeamTournamentSettings(ctx context.Context, tournamentID uuid.UUID) (*TeamTournamentSettings, error) {
	query := `
		SELECT tournament_id, boards_per_match, lineup_submission_deadline_minutes,
		       allow_late_lineup, enforce_rating_order, rating_order_tolerance,
		       match_win_points, match_draw_points, match_loss_points,
		       team_tiebreaks, board_win_points, board_draw_points, board_loss_points,
		       default_board_result
		FROM team_tournament_settings
		WHERE tournament_id = $1
	`
	var settings TeamTournamentSettings
	var tiebreaksJSON []byte
	err := r.db.QueryRow(ctx, query, tournamentID).Scan(
		&settings.TournamentID, &settings.BoardsPerMatch, &settings.LineupSubmissionDeadlineMinutes,
		&settings.AllowLateLineup, &settings.EnforceRatingOrder, &settings.RatingOrderTolerance,
		&settings.MatchWinPoints, &settings.MatchDrawPoints, &settings.MatchLossPoints,
		&tiebreaksJSON, &settings.BoardWinPoints, &settings.BoardDrawPoints, &settings.BoardLossPoints,
		&settings.DefaultBoardResult,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tiebreaksJSON, &settings.TeamTiebreaks)
	return &settings, nil
}
