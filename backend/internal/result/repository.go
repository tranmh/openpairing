package result

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier interface for database operations - satisfied by both *pgxpool.Pool and pgx.Tx
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Repository handles database operations for results
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new result repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// BeginTx starts a new transaction
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// GetGame retrieves a game by ID
func (r *Repository) GetGame(ctx context.Context, gameID uuid.UUID) (*Game, error) {
	var game Game
	err := r.db.QueryRow(ctx, `
		SELECT
			g.id,
			g.tournament_id,
			g.round_id,
			tr.round_number,
			g.board_number,
			g.white_participant_id,
			g.black_participant_id,
			g.result,
			g.white_score,
			g.black_score,
			g.status,
			g.result_entered_by,
			g.result_entered_at,
			g.pgn,
			g.created_at
		FROM games g
		JOIN tournament_rounds tr ON tr.id = g.round_id
		WHERE g.id = $1
	`, gameID).Scan(
		&game.ID,
		&game.TournamentID,
		&game.RoundID,
		&game.RoundNumber,
		&game.BoardNumber,
		&game.WhiteParticipantID,
		&game.BlackParticipantID,
		&game.Result,
		&game.WhiteScore,
		&game.BlackScore,
		&game.Status,
		&game.ResultEnteredBy,
		&game.ResultEnteredAt,
		&game.PGN,
		&game.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	return &game, nil
}

// GetGameWithPlayers retrieves a game with player information
func (r *Repository) GetGameWithPlayers(ctx context.Context, gameID uuid.UUID) (*Game, error) {
	var game Game
	var whiteFirstName, whiteLastName, blackFirstName, blackLastName *string
	var whiteRating, blackRating *int

	err := r.db.QueryRow(ctx, `
		SELECT
			g.id,
			g.tournament_id,
			g.round_id,
			tr.round_number,
			g.board_number,
			g.white_participant_id,
			g.black_participant_id,
			wp.first_name,
			wp.last_name,
			bp.first_name,
			bp.last_name,
			wtp.starting_rating,
			btp.starting_rating,
			g.result,
			g.white_score,
			g.black_score,
			g.status,
			g.result_entered_by,
			g.result_entered_at,
			g.pgn,
			g.created_at
		FROM games g
		JOIN tournament_rounds tr ON tr.id = g.round_id
		LEFT JOIN tournament_participants wtp ON wtp.id = g.white_participant_id
		LEFT JOIN tournament_participants btp ON btp.id = g.black_participant_id
		LEFT JOIN players wp ON wp.id = wtp.player_id
		LEFT JOIN players bp ON bp.id = btp.player_id
		WHERE g.id = $1
	`, gameID).Scan(
		&game.ID,
		&game.TournamentID,
		&game.RoundID,
		&game.RoundNumber,
		&game.BoardNumber,
		&game.WhiteParticipantID,
		&game.BlackParticipantID,
		&whiteFirstName,
		&whiteLastName,
		&blackFirstName,
		&blackLastName,
		&whiteRating,
		&blackRating,
		&game.Result,
		&game.WhiteScore,
		&game.BlackScore,
		&game.Status,
		&game.ResultEnteredBy,
		&game.ResultEnteredAt,
		&game.PGN,
		&game.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Build player names
	if whiteFirstName != nil && whiteLastName != nil {
		name := *whiteFirstName + " " + *whiteLastName
		game.WhitePlayerName = &name
	}
	if blackFirstName != nil && blackLastName != nil {
		name := *blackFirstName + " " + *blackLastName
		game.BlackPlayerName = &name
	}
	game.WhiteRating = whiteRating
	game.BlackRating = blackRating

	return &game, nil
}

// UpdateResult updates a game's result
func (r *Repository) UpdateResult(ctx context.Context, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
	return r.UpdateResultWithTx(ctx, r.db, gameID, result, whiteScore, blackScore, enteredBy)
}

// UpdateResultWithTx updates a game's result within a transaction
func (r *Repository) UpdateResultWithTx(ctx context.Context, q Querier, gameID uuid.UUID, result string, whiteScore, blackScore float64, enteredBy uuid.UUID) error {
	now := time.Now()
	_, err := q.Exec(ctx, `
		UPDATE games
		SET result = $2, white_score = $3, black_score = $4,
		    result_entered_by = $5, result_entered_at = $6,
		    status = 'completed'
		WHERE id = $1
	`, gameID, result, whiteScore, blackScore, enteredBy, now)
	if err != nil {
		return fmt.Errorf("failed to update result: %w", err)
	}
	return nil
}

// ClearResult removes a game's result
func (r *Repository) ClearResult(ctx context.Context, gameID uuid.UUID) error {
	return r.ClearResultWithTx(ctx, r.db, gameID)
}

// ClearResultWithTx removes a game's result within a transaction
func (r *Repository) ClearResultWithTx(ctx context.Context, q Querier, gameID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		UPDATE games
		SET result = NULL, white_score = NULL, black_score = NULL,
		    result_entered_by = NULL, result_entered_at = NULL,
		    status = 'scheduled'
		WHERE id = $1
	`, gameID)
	if err != nil {
		return fmt.Errorf("failed to clear result: %w", err)
	}
	return nil
}

// UpdateParticipantStats updates participant statistics after a result change
func (r *Repository) UpdateParticipantStats(ctx context.Context, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
	return r.UpdateParticipantStatsWithTx(ctx, r.db, participantID, pointsDelta, winsDelta, drawsDelta, lossesDelta)
}

// UpdateParticipantStatsWithTx updates participant statistics within a transaction
func (r *Repository) UpdateParticipantStatsWithTx(ctx context.Context, q Querier, participantID uuid.UUID, pointsDelta, winsDelta, drawsDelta, lossesDelta int) error {
	_, err := q.Exec(ctx, `
		UPDATE tournament_participants
		SET points = points + $2,
		    wins = wins + $3,
		    draws = draws + $4,
		    losses = losses + $5
		WHERE id = $1
	`, participantID, float64(pointsDelta)/10.0, winsDelta, drawsDelta, lossesDelta)
	if err != nil {
		return fmt.Errorf("failed to update participant stats: %w", err)
	}
	return nil
}

// RecalculateParticipantStats recalculates all stats for a participant from games
func (r *Repository) RecalculateParticipantStats(ctx context.Context, participantID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tournament_participants tp
		SET points = COALESCE(stats.total_points, 0),
		    wins = COALESCE(stats.wins, 0),
		    draws = COALESCE(stats.draws, 0),
		    losses = COALESCE(stats.losses, 0)
		FROM (
			SELECT
				$1::uuid as participant_id,
				SUM(CASE
					WHEN white_participant_id = $1 THEN COALESCE(white_score, 0)
					WHEN black_participant_id = $1 THEN COALESCE(black_score, 0)
					ELSE 0
				END) as total_points,
				SUM(CASE
					WHEN (white_participant_id = $1 AND white_score = 1) OR
					     (black_participant_id = $1 AND black_score = 1) THEN 1
					ELSE 0
				END) as wins,
				SUM(CASE
					WHEN (white_participant_id = $1 AND white_score = 0.5) OR
					     (black_participant_id = $1 AND black_score = 0.5) THEN 1
					ELSE 0
				END) as draws,
				SUM(CASE
					WHEN (white_participant_id = $1 AND white_score = 0) OR
					     (black_participant_id = $1 AND black_score = 0) THEN 1
					ELSE 0
				END) as losses
			FROM games
			WHERE (white_participant_id = $1 OR black_participant_id = $1)
			  AND result IS NOT NULL
		) stats
		WHERE tp.id = $1
	`, participantID)
	if err != nil {
		return fmt.Errorf("failed to recalculate participant stats: %w", err)
	}
	return nil
}

// GetRoundStatus retrieves result status for a round
func (r *Repository) GetRoundStatus(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*RoundStatus, error) {
	var status RoundStatus
	status.RoundNumber = roundNumber

	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN g.status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN g.status = 'scheduled' THEN 1 END) as pending,
			COUNT(CASE WHEN g.status = 'ongoing' THEN 1 END) as ongoing
		FROM games g
		JOIN tournament_rounds tr ON tr.id = g.round_id
		WHERE g.tournament_id = $1 AND tr.round_number = $2
	`, tournamentID, roundNumber).Scan(
		&status.TotalGames,
		&status.CompletedGames,
		&status.PendingGames,
		&status.OngoingGames,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get round status: %w", err)
	}

	status.IsComplete = status.CompletedGames == status.TotalGames && status.TotalGames > 0

	return &status, nil
}

// GetRoundGames retrieves all games for a round
func (r *Repository) GetRoundGames(ctx context.Context, tournamentID uuid.UUID, roundNumber int) ([]Game, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			g.id,
			g.tournament_id,
			g.round_id,
			tr.round_number,
			g.board_number,
			g.white_participant_id,
			g.black_participant_id,
			COALESCE(wp.first_name, '') || ' ' || COALESCE(wp.last_name, ''),
			COALESCE(bp.first_name, '') || ' ' || COALESCE(bp.last_name, ''),
			wtp.starting_rating,
			btp.starting_rating,
			g.result,
			g.white_score,
			g.black_score,
			g.status,
			g.result_entered_by,
			g.result_entered_at,
			g.created_at
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
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		var whiteName, blackName *string
		err := rows.Scan(
			&g.ID,
			&g.TournamentID,
			&g.RoundID,
			&g.RoundNumber,
			&g.BoardNumber,
			&g.WhiteParticipantID,
			&g.BlackParticipantID,
			&whiteName,
			&blackName,
			&g.WhiteRating,
			&g.BlackRating,
			&g.Result,
			&g.WhiteScore,
			&g.BlackScore,
			&g.Status,
			&g.ResultEnteredBy,
			&g.ResultEnteredAt,
			&g.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}
		g.WhitePlayerName = whiteName
		g.BlackPlayerName = blackName
		games = append(games, g)
	}

	return games, rows.Err()
}

// CreateAuditLog creates an audit log entry for a result change
func (r *Repository) CreateAuditLog(ctx context.Context, audit *ResultAudit) error {
	audit.ID = uuid.New()
	audit.PerformedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_log (id, user_id, action, entity_type, entity_id, old_value, new_value, ip_address, created_at)
		VALUES ($1, $2, $3, 'game', $4, $5, $6, $7, $8)
	`, audit.ID, audit.PerformedBy, audit.Action, audit.GameID,
		formatAuditOldValue(audit), formatAuditNewValue(audit), audit.IPAddress, audit.PerformedAt)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// GetAuditLog retrieves audit logs for a game
func (r *Repository) GetAuditLog(ctx context.Context, gameID uuid.UUID) ([]ResultAudit, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, entity_id, action, old_value, new_value, user_id, created_at, ip_address
		FROM audit_log
		WHERE entity_type = 'game' AND entity_id = $1
		ORDER BY created_at DESC
	`, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var audits []ResultAudit
	for rows.Next() {
		var a ResultAudit
		var oldValue, newValue []byte
		err := rows.Scan(
			&a.ID,
			&a.GameID,
			&a.Action,
			&oldValue,
			&newValue,
			&a.PerformedBy,
			&a.PerformedAt,
			&a.IPAddress,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit: %w", err)
		}
		// Parse JSON values if needed
		audits = append(audits, a)
	}

	return audits, rows.Err()
}

// UpdateRoundCompletedCount updates the completed game count for a round
func (r *Repository) UpdateRoundCompletedCount(ctx context.Context, roundID uuid.UUID) error {
	return r.UpdateRoundCompletedCountWithTx(ctx, r.db, roundID)
}

// UpdateRoundCompletedCountWithTx updates the completed game count within a transaction
func (r *Repository) UpdateRoundCompletedCountWithTx(ctx context.Context, q Querier, roundID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		UPDATE tournament_rounds tr
		SET completed_count = (
			SELECT COUNT(*) FROM games g
			WHERE g.round_id = tr.id AND g.status = 'completed'
		)
		WHERE tr.id = $1
	`, roundID)
	if err != nil {
		return fmt.Errorf("failed to update round completed count: %w", err)
	}
	return nil
}

// Helper functions

func formatAuditOldValue(audit *ResultAudit) interface{} {
	if audit.OldResult == nil {
		return nil
	}
	return fmt.Sprintf(`{"result":"%s","whiteScore":%v,"blackScore":%v}`,
		*audit.OldResult,
		floatPtrToString(audit.OldWhiteScore),
		floatPtrToString(audit.OldBlackScore))
}

func formatAuditNewValue(audit *ResultAudit) interface{} {
	if audit.NewResult == nil {
		return nil
	}
	return fmt.Sprintf(`{"result":"%s","whiteScore":%v,"blackScore":%v,"reason":"%s"}`,
		*audit.NewResult,
		floatPtrToString(audit.NewWhiteScore),
		floatPtrToString(audit.NewBlackScore),
		stringPtrToValue(audit.Reason))
}

func floatPtrToString(f *float64) string {
	if f == nil {
		return "null"
	}
	return fmt.Sprintf("%.1f", *f)
}

func stringPtrToValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
