package wam

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrConfigNotFound    = errors.New("WAM config not found")
	ErrGroupNotFound     = errors.New("WAM group not found")
	ErrMemberNotFound    = errors.New("group member not found")
	ErrPlayerNotInGroup  = errors.New("player not in any group")
	ErrGroupFull         = errors.New("group is full")
	ErrDuplicateMember   = errors.New("player already in this group")
)

// Repository defines WAM-specific data operations
type Repository interface {
	// Config
	CreateConfig(ctx context.Context, config *WAMConfig) error
	GetConfig(ctx context.Context, tournamentID uuid.UUID) (*WAMConfig, error)
	UpdateConfig(ctx context.Context, config *WAMConfig) error
	DeleteConfig(ctx context.Context, tournamentID uuid.UUID) error

	// Groups
	CreateGroup(ctx context.Context, group *WAMGroup) error
	GetGroup(ctx context.Context, groupID uuid.UUID) (*WAMGroup, error)
	GetGroupByNumber(ctx context.Context, tournamentID uuid.UUID, groupNumber int) (*WAMGroup, error)
	UpdateGroup(ctx context.Context, group *WAMGroup) error
	ListGroups(ctx context.Context, tournamentID uuid.UUID) ([]WAMGroup, error)
	DeleteGroup(ctx context.Context, groupID uuid.UUID) error
	DeleteAllGroups(ctx context.Context, tournamentID uuid.UUID) error

	// Group Members
	AddGroupMember(ctx context.Context, member *WAMGroupPlayer) error
	UpdateGroupMember(ctx context.Context, member *WAMGroupPlayer) error
	RemoveGroupMember(ctx context.Context, groupID, participantID uuid.UUID) error
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]WAMGroupPlayer, error)
	GetPlayerGroup(ctx context.Context, tournamentID, participantID uuid.UUID) (*WAMGroup, error)
	MovePlayerToGroup(ctx context.Context, participantID, fromGroupID, toGroupID uuid.UUID, newSortOrder int) error
	GetMaxSortOrder(ctx context.Context, groupID uuid.UUID) (int, error)

	// Group Games
	LinkGameToGroup(ctx context.Context, groupGame *WAMGroupGame) error
	GetGroupGames(ctx context.Context, groupID uuid.UUID) ([]Game, error)

	// Playoffs
	CreatePlayoffMatch(ctx context.Context, match *WAMPlayoffMatch) error
	GetPlayoffMatches(ctx context.Context, tournamentID uuid.UUID) ([]WAMPlayoffMatch, error)
	UpdatePlayoffMatch(ctx context.Context, match *WAMPlayoffMatch) error
	DeletePlayoffMatches(ctx context.Context, tournamentID uuid.UUID) error

	// Qualifications
	AddQualification(ctx context.Context, qual *WAMQualification) error
	GetQualifications(ctx context.Context, tournamentID uuid.UUID) ([]WAMQualification, error)
	GetQualificationsBySeason(ctx context.Context, season string) ([]WAMQualification, error)
	GetPlayerQualifications(ctx context.Context, playerID uuid.UUID) ([]WAMQualification, error)
	DeleteQualifications(ctx context.Context, tournamentID uuid.UUID) error

	// Tournament Participants (for assignment)
	GetTournamentParticipants(ctx context.Context, tournamentID uuid.UUID) ([]TournamentParticipant, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new WAM repository
func NewRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{db: db}
}

// ===== Config Operations =====

func (r *PostgresRepository) CreateConfig(ctx context.Context, config *WAMConfig) error {
	tiebreaksJSON, err := json.Marshal(config.Tiebreaks)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO wam_tournament_config (
			tournament_id, group_size, assignment_method, enable_playoffs, playoffs_top_n,
			third_place_match, youth_age_limit, current_phase, tiebreaks, season
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, config.TournamentID, config.GroupSize, config.AssignmentMethod, config.EnablePlayoffs,
		config.PlayoffsTopN, config.ThirdPlaceMatch, config.YouthAgeLimit, config.CurrentPhase,
		tiebreaksJSON, config.Season)

	return err
}

func (r *PostgresRepository) GetConfig(ctx context.Context, tournamentID uuid.UUID) (*WAMConfig, error) {
	var config WAMConfig
	var tiebreaksJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT tournament_id, group_size, assignment_method, enable_playoffs, playoffs_top_n,
		       third_place_match, youth_age_limit, current_phase, tiebreaks, season,
		       created_at, updated_at
		FROM wam_tournament_config WHERE tournament_id = $1
	`, tournamentID).Scan(
		&config.TournamentID, &config.GroupSize, &config.AssignmentMethod, &config.EnablePlayoffs,
		&config.PlayoffsTopN, &config.ThirdPlaceMatch, &config.YouthAgeLimit, &config.CurrentPhase,
		&tiebreaksJSON, &config.Season, &config.CreatedAt, &config.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(tiebreaksJSON, &config.Tiebreaks); err != nil {
		return nil, err
	}

	return &config, nil
}

func (r *PostgresRepository) UpdateConfig(ctx context.Context, config *WAMConfig) error {
	tiebreaksJSON, err := json.Marshal(config.Tiebreaks)
	if err != nil {
		return err
	}

	result, err := r.db.Exec(ctx, `
		UPDATE wam_tournament_config SET
			group_size = $2, assignment_method = $3, enable_playoffs = $4, playoffs_top_n = $5,
			third_place_match = $6, youth_age_limit = $7, current_phase = $8, tiebreaks = $9, season = $10
		WHERE tournament_id = $1
	`, config.TournamentID, config.GroupSize, config.AssignmentMethod, config.EnablePlayoffs,
		config.PlayoffsTopN, config.ThirdPlaceMatch, config.YouthAgeLimit, config.CurrentPhase,
		tiebreaksJSON, config.Season)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrConfigNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteConfig(ctx context.Context, tournamentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM wam_tournament_config WHERE tournament_id = $1", tournamentID)
	return err
}

// ===== Group Operations =====

func (r *PostgresRepository) CreateGroup(ctx context.Context, group *WAMGroup) error {
	if group.ID == uuid.Nil {
		group.ID = uuid.New()
	}
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO wam_groups (id, tournament_id, name, group_number, rounds_total, rounds_played, is_complete, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, group.ID, group.TournamentID, group.Name, group.GroupNumber, group.RoundsTotal, group.RoundsPlayed, group.IsComplete, group.CreatedAt, group.UpdatedAt)

	return err
}

func (r *PostgresRepository) GetGroup(ctx context.Context, groupID uuid.UUID) (*WAMGroup, error) {
	var group WAMGroup

	err := r.db.QueryRow(ctx, `
		SELECT id, tournament_id, name, group_number, rounds_total, rounds_played, is_complete, created_at, updated_at
		FROM wam_groups WHERE id = $1
	`, groupID).Scan(
		&group.ID, &group.TournamentID, &group.Name, &group.GroupNumber,
		&group.RoundsTotal, &group.RoundsPlayed, &group.IsComplete,
		&group.CreatedAt, &group.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}

	// Load members
	members, err := r.GetGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	group.Players = members

	return &group, nil
}

func (r *PostgresRepository) GetGroupByNumber(ctx context.Context, tournamentID uuid.UUID, groupNumber int) (*WAMGroup, error) {
	var groupID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id FROM wam_groups WHERE tournament_id = $1 AND group_number = $2
	`, tournamentID, groupNumber).Scan(&groupID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}

	return r.GetGroup(ctx, groupID)
}

func (r *PostgresRepository) UpdateGroup(ctx context.Context, group *WAMGroup) error {
	result, err := r.db.Exec(ctx, `
		UPDATE wam_groups SET
			name = $2, rounds_total = $3, rounds_played = $4, is_complete = $5
		WHERE id = $1
	`, group.ID, group.Name, group.RoundsTotal, group.RoundsPlayed, group.IsComplete)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func (r *PostgresRepository) ListGroups(ctx context.Context, tournamentID uuid.UUID) ([]WAMGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tournament_id, name, group_number, rounds_total, rounds_played, is_complete, created_at, updated_at
		FROM wam_groups WHERE tournament_id = $1 ORDER BY group_number
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []WAMGroup
	for rows.Next() {
		var group WAMGroup
		if err := rows.Scan(
			&group.ID, &group.TournamentID, &group.Name, &group.GroupNumber,
			&group.RoundsTotal, &group.RoundsPlayed, &group.IsComplete,
			&group.CreatedAt, &group.UpdatedAt,
		); err != nil {
			return nil, err
		}

		// Load members for each group
		members, err := r.GetGroupMembers(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		group.Players = members

		groups = append(groups, group)
	}

	return groups, nil
}

func (r *PostgresRepository) DeleteGroup(ctx context.Context, groupID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM wam_groups WHERE id = $1", groupID)
	return err
}

func (r *PostgresRepository) DeleteAllGroups(ctx context.Context, tournamentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM wam_groups WHERE tournament_id = $1", tournamentID)
	return err
}

// ===== Group Member Operations =====

func (r *PostgresRepository) AddGroupMember(ctx context.Context, member *WAMGroupPlayer) error {
	if member.ID == uuid.Nil {
		member.ID = uuid.New()
	}
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO wam_group_members (id, group_id, participant_id, sort_order, points, wins, draws, losses, is_group_winner, qualified_final, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, member.ID, member.GroupID, member.ParticipantID, member.SortOrder, member.Points,
		member.Wins, member.Draws, member.Losses, member.IsGroupWinner, member.QualifiedFinal,
		member.CreatedAt, member.UpdatedAt)

	return err
}

func (r *PostgresRepository) UpdateGroupMember(ctx context.Context, member *WAMGroupPlayer) error {
	result, err := r.db.Exec(ctx, `
		UPDATE wam_group_members SET
			sort_order = $3, points = $4, wins = $5, draws = $6, losses = $7,
			is_group_winner = $8, qualified_final = $9
		WHERE group_id = $1 AND participant_id = $2
	`, member.GroupID, member.ParticipantID, member.SortOrder, member.Points,
		member.Wins, member.Draws, member.Losses, member.IsGroupWinner, member.QualifiedFinal)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *PostgresRepository) RemoveGroupMember(ctx context.Context, groupID, participantID uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM wam_group_members WHERE group_id = $1 AND participant_id = $2
	`, groupID, participantID)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *PostgresRepository) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]WAMGroupPlayer, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.group_id, m.participant_id, m.sort_order, m.points, m.wins, m.draws, m.losses,
		       m.is_group_winner, m.qualified_final, m.created_at, m.updated_at,
		       tp.player_id, COALESCE(p.first_name || ' ' || p.last_name, '') as name,
		       COALESCE(p.rating, 0) as rating, p.club_id, c.name as club_name, p.birth_date
		FROM wam_group_members m
		JOIN tournament_participants tp ON m.participant_id = tp.id
		JOIN players p ON tp.player_id = p.id
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE m.group_id = $1
		ORDER BY m.sort_order
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []WAMGroupPlayer
	for rows.Next() {
		var m WAMGroupPlayer
		if err := rows.Scan(
			&m.ID, &m.GroupID, &m.ParticipantID, &m.SortOrder, &m.Points,
			&m.Wins, &m.Draws, &m.Losses, &m.IsGroupWinner, &m.QualifiedFinal,
			&m.CreatedAt, &m.UpdatedAt,
			&m.PlayerID, &m.Name, &m.Rating, &m.ClubID, &m.ClubName, &m.BirthDate,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

func (r *PostgresRepository) GetPlayerGroup(ctx context.Context, tournamentID, participantID uuid.UUID) (*WAMGroup, error) {
	var groupID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT g.id FROM wam_groups g
		JOIN wam_group_members m ON g.id = m.group_id
		WHERE g.tournament_id = $1 AND m.participant_id = $2
	`, tournamentID, participantID).Scan(&groupID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayerNotInGroup
	}
	if err != nil {
		return nil, err
	}

	return r.GetGroup(ctx, groupID)
}

func (r *PostgresRepository) MovePlayerToGroup(ctx context.Context, participantID, fromGroupID, toGroupID uuid.UUID, newSortOrder int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get current sort order
	var oldSortOrder int
	err = tx.QueryRow(ctx, `
		SELECT sort_order FROM wam_group_members WHERE group_id = $1 AND participant_id = $2
	`, fromGroupID, participantID).Scan(&oldSortOrder)
	if err != nil {
		return err
	}

	// Remove from old group
	_, err = tx.Exec(ctx, `
		DELETE FROM wam_group_members WHERE group_id = $1 AND participant_id = $2
	`, fromGroupID, participantID)
	if err != nil {
		return err
	}

	// Reorder remaining members in old group
	_, err = tx.Exec(ctx, `
		UPDATE wam_group_members SET sort_order = sort_order - 1
		WHERE group_id = $1 AND sort_order > $2
	`, fromGroupID, oldSortOrder)
	if err != nil {
		return err
	}

	// If no sort order specified, put at end
	if newSortOrder <= 0 {
		var maxOrder int
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(sort_order), 0) FROM wam_group_members WHERE group_id = $1
		`, toGroupID).Scan(&maxOrder)
		if err != nil {
			return err
		}
		newSortOrder = maxOrder + 1
	} else {
		// Shift existing members to make room
		_, err = tx.Exec(ctx, `
			UPDATE wam_group_members SET sort_order = sort_order + 1
			WHERE group_id = $1 AND sort_order >= $2
		`, toGroupID, newSortOrder)
		if err != nil {
			return err
		}
	}

	// Add to new group
	_, err = tx.Exec(ctx, `
		INSERT INTO wam_group_members (id, group_id, participant_id, sort_order, points, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, NOW(), NOW())
	`, uuid.New(), toGroupID, participantID, newSortOrder)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetMaxSortOrder(ctx context.Context, groupID uuid.UUID) (int, error) {
	var maxOrder int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order), 0) FROM wam_group_members WHERE group_id = $1
	`, groupID).Scan(&maxOrder)
	return maxOrder, err
}

// ===== Group Game Operations =====

func (r *PostgresRepository) LinkGameToGroup(ctx context.Context, groupGame *WAMGroupGame) error {
	if groupGame.ID == uuid.Nil {
		groupGame.ID = uuid.New()
	}
	groupGame.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO wam_group_games (id, group_id, game_id, round_in_group, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, groupGame.ID, groupGame.GroupID, groupGame.GameID, groupGame.RoundInGroup, groupGame.CreatedAt)

	return err
}

func (r *PostgresRepository) GetGroupGames(ctx context.Context, groupID uuid.UUID) ([]Game, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.round_id, g.board_number, g.white_participant_id, g.black_participant_id,
		       g.white_score, g.black_score, g.result, g.status
		FROM games g
		JOIN wam_group_games wgg ON g.id = wgg.game_id
		WHERE wgg.group_id = $1
		ORDER BY wgg.round_in_group, g.board_number
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		if err := rows.Scan(
			&game.ID, &game.RoundID, &game.BoardNumber,
			&game.WhiteParticipantID, &game.BlackParticipantID,
			&game.WhiteScore, &game.BlackScore, &game.Result, &game.Status,
		); err != nil {
			return nil, err
		}
		games = append(games, game)
	}

	return games, nil
}

// ===== Playoff Operations =====

func (r *PostgresRepository) CreatePlayoffMatch(ctx context.Context, match *WAMPlayoffMatch) error {
	if match.ID == uuid.Nil {
		match.ID = uuid.New()
	}
	match.CreatedAt = time.Now()
	match.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO wam_playoff_matches (id, tournament_id, round_name, match_number,
			player1_participant_id, player2_participant_id, player1_source, player2_source,
			winner_participant_id, loser_participant_id, game_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, match.ID, match.TournamentID, match.RoundName, match.MatchNumber,
		match.Player1ID, match.Player2ID, match.Player1Source, match.Player2Source,
		match.WinnerID, match.LoserID, match.GameID, match.CreatedAt)

	return err
}

func (r *PostgresRepository) GetPlayoffMatches(ctx context.Context, tournamentID uuid.UUID) ([]WAMPlayoffMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.tournament_id, m.round_name, m.match_number,
		       m.player1_participant_id, m.player2_participant_id,
		       m.player1_source, m.player2_source,
		       m.winner_participant_id, m.loser_participant_id, m.game_id,
		       m.created_at, m.updated_at,
		       COALESCE(p1.first_name || ' ' || p1.last_name, '') as player1_name,
		       COALESCE(p2.first_name || ' ' || p2.last_name, '') as player2_name,
		       COALESCE(pw.first_name || ' ' || pw.last_name, '') as winner_name
		FROM wam_playoff_matches m
		LEFT JOIN tournament_participants tp1 ON m.player1_participant_id = tp1.id
		LEFT JOIN players p1 ON tp1.player_id = p1.id
		LEFT JOIN tournament_participants tp2 ON m.player2_participant_id = tp2.id
		LEFT JOIN players p2 ON tp2.player_id = p2.id
		LEFT JOIN tournament_participants tpw ON m.winner_participant_id = tpw.id
		LEFT JOIN players pw ON tpw.player_id = pw.id
		WHERE m.tournament_id = $1
		ORDER BY
			CASE m.round_name
				WHEN 'quarterfinal' THEN 1
				WHEN 'semifinal' THEN 2
				WHEN 'third_place' THEN 3
				WHEN 'final' THEN 4
				ELSE 0
			END,
			m.match_number
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []WAMPlayoffMatch
	for rows.Next() {
		var m WAMPlayoffMatch
		if err := rows.Scan(
			&m.ID, &m.TournamentID, &m.RoundName, &m.MatchNumber,
			&m.Player1ID, &m.Player2ID, &m.Player1Source, &m.Player2Source,
			&m.WinnerID, &m.LoserID, &m.GameID, &m.CreatedAt, &m.UpdatedAt,
			&m.Player1Name, &m.Player2Name, &m.WinnerName,
		); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}

	return matches, nil
}

func (r *PostgresRepository) UpdatePlayoffMatch(ctx context.Context, match *WAMPlayoffMatch) error {
	result, err := r.db.Exec(ctx, `
		UPDATE wam_playoff_matches SET
			player1_participant_id = $2, player2_participant_id = $3,
			winner_participant_id = $4, loser_participant_id = $5, game_id = $6
		WHERE id = $1
	`, match.ID, match.Player1ID, match.Player2ID, match.WinnerID, match.LoserID, match.GameID)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("playoff match not found")
	}
	return nil
}

func (r *PostgresRepository) DeletePlayoffMatches(ctx context.Context, tournamentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM wam_playoff_matches WHERE tournament_id = $1", tournamentID)
	return err
}

// ===== Qualification Operations =====

func (r *PostgresRepository) AddQualification(ctx context.Context, qual *WAMQualification) error {
	if qual.ID == uuid.Nil {
		qual.ID = uuid.New()
	}
	qual.QualifiedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO wam_qualifications (id, tournament_id, participant_id, player_id, group_name, group_rank, season, qualified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, qual.ID, qual.TournamentID, qual.ParticipantID, qual.PlayerID, qual.GroupName, qual.GroupRank, qual.Season, qual.QualifiedAt)

	return err
}

func (r *PostgresRepository) GetQualifications(ctx context.Context, tournamentID uuid.UUID) ([]WAMQualification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT q.id, q.tournament_id, q.participant_id, q.player_id, q.group_name, q.group_rank, q.season, q.qualified_at,
		       COALESCE(p.first_name || ' ' || p.last_name, '') as player_name
		FROM wam_qualifications q
		JOIN players p ON q.player_id = p.id
		WHERE q.tournament_id = $1
		ORDER BY q.group_name, q.group_rank
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var qualifications []WAMQualification
	for rows.Next() {
		var q WAMQualification
		if err := rows.Scan(
			&q.ID, &q.TournamentID, &q.ParticipantID, &q.PlayerID,
			&q.GroupName, &q.GroupRank, &q.Season, &q.QualifiedAt,
			&q.PlayerName,
		); err != nil {
			return nil, err
		}
		qualifications = append(qualifications, q)
	}

	return qualifications, nil
}

func (r *PostgresRepository) GetQualificationsBySeason(ctx context.Context, season string) ([]WAMQualification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT q.id, q.tournament_id, q.participant_id, q.player_id, q.group_name, q.group_rank, q.season, q.qualified_at,
		       COALESCE(p.first_name || ' ' || p.last_name, '') as player_name
		FROM wam_qualifications q
		JOIN players p ON q.player_id = p.id
		WHERE q.season = $1
		ORDER BY q.qualified_at
	`, season)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var qualifications []WAMQualification
	for rows.Next() {
		var q WAMQualification
		if err := rows.Scan(
			&q.ID, &q.TournamentID, &q.ParticipantID, &q.PlayerID,
			&q.GroupName, &q.GroupRank, &q.Season, &q.QualifiedAt,
			&q.PlayerName,
		); err != nil {
			return nil, err
		}
		qualifications = append(qualifications, q)
	}

	return qualifications, nil
}

func (r *PostgresRepository) GetPlayerQualifications(ctx context.Context, playerID uuid.UUID) ([]WAMQualification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT q.id, q.tournament_id, q.participant_id, q.player_id, q.group_name, q.group_rank, q.season, q.qualified_at,
		       COALESCE(p.first_name || ' ' || p.last_name, '') as player_name
		FROM wam_qualifications q
		JOIN players p ON q.player_id = p.id
		WHERE q.player_id = $1
		ORDER BY q.qualified_at DESC
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var qualifications []WAMQualification
	for rows.Next() {
		var q WAMQualification
		if err := rows.Scan(
			&q.ID, &q.TournamentID, &q.ParticipantID, &q.PlayerID,
			&q.GroupName, &q.GroupRank, &q.Season, &q.QualifiedAt,
			&q.PlayerName,
		); err != nil {
			return nil, err
		}
		qualifications = append(qualifications, q)
	}

	return qualifications, nil
}

func (r *PostgresRepository) DeleteQualifications(ctx context.Context, tournamentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM wam_qualifications WHERE tournament_id = $1", tournamentID)
	return err
}

// ===== Tournament Participants =====

func (r *PostgresRepository) GetTournamentParticipants(ctx context.Context, tournamentID uuid.UUID) ([]TournamentParticipant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tp.id, tp.player_id,
		       COALESCE(p.first_name || ' ' || p.last_name, '') as name,
		       COALESCE(p.rating, 0) as rating, p.club_id, c.name as club_name, p.birth_date
		FROM tournament_participants tp
		JOIN players p ON tp.player_id = p.id
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE tp.tournament_id = $1 AND tp.status = 'active'
		ORDER BY COALESCE(p.rating, 0) DESC
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []TournamentParticipant
	for rows.Next() {
		var p TournamentParticipant
		if err := rows.Scan(
			&p.ParticipantID, &p.PlayerID, &p.Name, &p.Rating,
			&p.ClubID, &p.ClubName, &p.BirthDate,
		); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, nil
}
