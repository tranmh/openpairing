package fairplay

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for fair play
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new fairplay repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// EnsureTables ensures the required tables exist
func (r *Repository) EnsureTables(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS fairplay_violations (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tournament_id UUID REFERENCES tournaments(id) ON DELETE SET NULL,
			game_id UUID,
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			reported_by UUID REFERENCES users(id) ON DELETE SET NULL,
			violation_type VARCHAR(50) NOT NULL,
			severity VARCHAR(20) NOT NULL DEFAULT 'medium',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			description TEXT NOT NULL,
			evidence TEXT,
			reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
			review_notes TEXT,
			action_taken TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			reviewed_at TIMESTAMPTZ
		);

		CREATE INDEX IF NOT EXISTS idx_fp_violations_player ON fairplay_violations(player_id);
		CREATE INDEX IF NOT EXISTS idx_fp_violations_tournament ON fairplay_violations(tournament_id);
		CREATE INDEX IF NOT EXISTS idx_fp_violations_status ON fairplay_violations(status);

		CREATE TABLE IF NOT EXISTS player_flags (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			flagged_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			flag_type VARCHAR(20) NOT NULL,
			reason TEXT NOT NULL,
			expires_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			removed_at TIMESTAMPTZ,
			removed_by UUID REFERENCES users(id) ON DELETE SET NULL
		);

		CREATE INDEX IF NOT EXISTS idx_player_flags_player ON player_flags(player_id);
		CREATE INDEX IF NOT EXISTS idx_player_flags_type ON player_flags(flag_type);
	`
	_, err := r.pool.Exec(ctx, query)
	return err
}

// CreateViolation creates a new violation report
func (r *Repository) CreateViolation(ctx context.Context, v *FairplayViolation) error {
	query := `
		INSERT INTO fairplay_violations (tournament_id, game_id, player_id, reported_by,
		                                  violation_type, severity, description, evidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, status, created_at`

	return r.pool.QueryRow(ctx, query,
		v.TournamentID, v.GameID, v.PlayerID, v.ReportedBy,
		v.Type, v.Severity, v.Description, v.Evidence,
	).Scan(&v.ID, &v.Status, &v.CreatedAt)
}

// GetViolation retrieves a violation by ID
func (r *Repository) GetViolation(ctx context.Context, id uuid.UUID) (*FairplayViolation, error) {
	query := `
		SELECT id, tournament_id, game_id, player_id, reported_by, violation_type,
		       severity, status, description, evidence, reviewed_by, review_notes,
		       action_taken, created_at, reviewed_at
		FROM fairplay_violations
		WHERE id = $1`

	var v FairplayViolation
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.TournamentID, &v.GameID, &v.PlayerID, &v.ReportedBy, &v.Type,
		&v.Severity, &v.Status, &v.Description, &v.Evidence, &v.ReviewedBy, &v.ReviewNotes,
		&v.ActionTaken, &v.CreatedAt, &v.ReviewedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("violation not found")
		}
		return nil, err
	}
	return &v, nil
}

// UpdateViolation updates a violation after review
func (r *Repository) UpdateViolation(ctx context.Context, id uuid.UUID, status ViolationStatus, reviewedBy uuid.UUID, reviewNotes string, actionTaken *string) error {
	query := `
		UPDATE fairplay_violations
		SET status = $1, reviewed_by = $2, review_notes = $3, action_taken = $4, reviewed_at = NOW()
		WHERE id = $5`

	result, err := r.pool.Exec(ctx, query, status, reviewedBy, reviewNotes, actionTaken, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("violation not found")
	}
	return nil
}

// ListViolations lists violations with optional filtering
func (r *Repository) ListViolations(ctx context.Context, req *ViolationListRequest) ([]FairplayViolation, int, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if req.TournamentID != nil {
		whereClause += fmt.Sprintf(" AND tournament_id = $%d", argNum)
		args = append(args, *req.TournamentID)
		argNum++
	}
	if req.PlayerID != nil {
		whereClause += fmt.Sprintf(" AND player_id = $%d", argNum)
		args = append(args, *req.PlayerID)
		argNum++
	}
	if req.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, req.Status)
		argNum++
	}
	if req.Type != "" {
		whereClause += fmt.Sprintf(" AND violation_type = $%d", argNum)
		args = append(args, req.Type)
		argNum++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM fairplay_violations %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (req.Page - 1) * req.Limit
	args = append(args, req.Limit, offset)
	query := fmt.Sprintf(`
		SELECT id, tournament_id, game_id, player_id, reported_by, violation_type,
		       severity, status, description, evidence, reviewed_by, review_notes,
		       action_taken, created_at, reviewed_at
		FROM fairplay_violations
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var violations []FairplayViolation
	for rows.Next() {
		var v FairplayViolation
		if err := rows.Scan(
			&v.ID, &v.TournamentID, &v.GameID, &v.PlayerID, &v.ReportedBy, &v.Type,
			&v.Severity, &v.Status, &v.Description, &v.Evidence, &v.ReviewedBy, &v.ReviewNotes,
			&v.ActionTaken, &v.CreatedAt, &v.ReviewedAt,
		); err != nil {
			return nil, 0, err
		}
		violations = append(violations, v)
	}

	return violations, total, nil
}

// CreateFlag creates a player flag
func (r *Repository) CreateFlag(ctx context.Context, f *PlayerFlag) error {
	query := `
		INSERT INTO player_flags (player_id, flagged_by, flag_type, reason, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		f.PlayerID, f.FlaggedBy, f.FlagType, f.Reason, f.ExpiresAt,
	).Scan(&f.ID, &f.CreatedAt)
}

// GetPlayerFlags retrieves active flags for a player
func (r *Repository) GetPlayerFlags(ctx context.Context, playerID uuid.UUID) ([]PlayerFlag, error) {
	query := `
		SELECT id, player_id, flagged_by, flag_type, reason, expires_at, created_at, removed_at, removed_by
		FROM player_flags
		WHERE player_id = $1 AND removed_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []PlayerFlag
	for rows.Next() {
		var f PlayerFlag
		if err := rows.Scan(
			&f.ID, &f.PlayerID, &f.FlaggedBy, &f.FlagType, &f.Reason,
			&f.ExpiresAt, &f.CreatedAt, &f.RemovedAt, &f.RemovedBy,
		); err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}

	return flags, nil
}

// RemoveFlag removes a player flag
func (r *Repository) RemoveFlag(ctx context.Context, flagID, removedBy uuid.UUID) error {
	query := `UPDATE player_flags SET removed_at = NOW(), removed_by = $1 WHERE id = $2`
	result, err := r.pool.Exec(ctx, query, removedBy, flagID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("flag not found")
	}
	return nil
}

// GetStats retrieves fair play statistics
func (r *Repository) GetStats(ctx context.Context, tournamentID *uuid.UUID) (*FairplayStats, error) {
	var stats FairplayStats

	// Base where clause
	whereClause := ""
	args := []interface{}{}
	if tournamentID != nil {
		whereClause = " WHERE tournament_id = $1"
		args = append(args, *tournamentID)
	}

	// Count violations
	query := fmt.Sprintf("SELECT COUNT(*) FROM fairplay_violations%s", whereClause)
	r.pool.QueryRow(ctx, query, args...).Scan(&stats.TotalViolations)

	query = fmt.Sprintf("SELECT COUNT(*) FROM fairplay_violations%s", strings.Replace(whereClause, "WHERE", "WHERE status = 'pending' AND", 1))
	if whereClause == "" {
		query = "SELECT COUNT(*) FROM fairplay_violations WHERE status = 'pending'"
	}
	r.pool.QueryRow(ctx, query, args...).Scan(&stats.PendingViolations)

	query = fmt.Sprintf("SELECT COUNT(*) FROM fairplay_violations%s", strings.Replace(whereClause, "WHERE", "WHERE status = 'confirmed' AND", 1))
	if whereClause == "" {
		query = "SELECT COUNT(*) FROM fairplay_violations WHERE status = 'confirmed'"
	}
	r.pool.QueryRow(ctx, query, args...).Scan(&stats.ConfirmedViolations)

	// Count active flags
	r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM player_flags WHERE removed_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())").Scan(&stats.ActiveFlags)

	// Count watched players
	r.pool.QueryRow(ctx, "SELECT COUNT(DISTINCT player_id) FROM player_flags WHERE flag_type = 'watch' AND removed_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())").Scan(&stats.PlayersWatched)

	return &stats, nil
}

// GetPlayerViolationHistory retrieves violation history for a player
func (r *Repository) GetPlayerViolationHistory(ctx context.Context, playerID uuid.UUID) ([]FairplayViolation, error) {
	query := `
		SELECT id, tournament_id, game_id, player_id, reported_by, violation_type,
		       severity, status, description, evidence, reviewed_by, review_notes,
		       action_taken, created_at, reviewed_at
		FROM fairplay_violations
		WHERE player_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []FairplayViolation
	for rows.Next() {
		var v FairplayViolation
		if err := rows.Scan(
			&v.ID, &v.TournamentID, &v.GameID, &v.PlayerID, &v.ReportedBy, &v.Type,
			&v.Severity, &v.Status, &v.Description, &v.Evidence, &v.ReviewedBy, &v.ReviewNotes,
			&v.ActionTaken, &v.CreatedAt, &v.ReviewedAt,
		); err != nil {
			return nil, err
		}
		violations = append(violations, v)
	}

	return violations, nil
}

// HasActiveFlag checks if a player has an active flag of a specific type
func (r *Repository) HasActiveFlag(ctx context.Context, playerID uuid.UUID, flagType string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM player_flags
			WHERE player_id = $1 AND flag_type = $2 AND removed_at IS NULL
			  AND (expires_at IS NULL OR expires_at > NOW())
		)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, playerID, flagType).Scan(&exists)
	return exists, err
}
