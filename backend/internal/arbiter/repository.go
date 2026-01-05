package arbiter

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides access to arbiter-related data
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new arbiter repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// EnsureTables creates the required tables if they don't exist
func (r *Repository) EnsureTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS arbiter_assignments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tournament_id UUID NOT NULL,
			user_id UUID NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'assistant',
			assigned_at TIMESTAMPTZ DEFAULT NOW(),
			assigned_by UUID,
			UNIQUE(tournament_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_arbiter_assignments_tournament ON arbiter_assignments(tournament_id)`,
		`CREATE INDEX IF NOT EXISTS idx_arbiter_assignments_user ON arbiter_assignments(user_id)`,

		`CREATE TABLE IF NOT EXISTS arbiter_audit_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tournament_id UUID NOT NULL,
			arbiter_id UUID NOT NULL,
			action VARCHAR(100) NOT NULL,
			entity_type VARCHAR(50),
			entity_id UUID,
			old_value TEXT,
			new_value TEXT,
			reason TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_tournament ON arbiter_audit_log(tournament_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_arbiter ON arbiter_audit_log(arbiter_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_created ON arbiter_audit_log(created_at DESC)`,

		`CREATE TABLE IF NOT EXISTS tournament_announcements (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tournament_id UUID NOT NULL,
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			priority VARCHAR(50) DEFAULT 'info',
			created_by UUID NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			expires_at TIMESTAMPTZ,
			is_active BOOLEAN DEFAULT TRUE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_announcements_tournament ON tournament_announcements(tournament_id)`,
		`CREATE INDEX IF NOT EXISTS idx_announcements_active ON tournament_announcements(tournament_id, is_active)`,
	}

	for _, query := range queries {
		if _, err := r.pool.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

// GetArbiterAssignment retrieves an arbiter assignment
func (r *Repository) GetArbiterAssignment(ctx context.Context, tournamentID, userID uuid.UUID) (*ArbiterAssignment, error) {
	query := `
		SELECT tournament_id, user_id, role
		FROM arbiter_assignments
		WHERE tournament_id = $1 AND user_id = $2`

	var assignment ArbiterAssignment
	err := r.pool.QueryRow(ctx, query, tournamentID, userID).Scan(
		&assignment.TournamentID, &assignment.UserID, &assignment.Role,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

// AssignArbiter assigns an arbiter to a tournament
func (r *Repository) AssignArbiter(ctx context.Context, assignment *ArbiterAssignment, assignedBy uuid.UUID) error {
	query := `
		INSERT INTO arbiter_assignments (tournament_id, user_id, role, assigned_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tournament_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			assigned_by = EXCLUDED.assigned_by`

	_, err := r.pool.Exec(ctx, query,
		assignment.TournamentID, assignment.UserID, assignment.Role, assignedBy,
	)
	return err
}

// RemoveArbiter removes an arbiter from a tournament
func (r *Repository) RemoveArbiter(ctx context.Context, tournamentID, userID uuid.UUID) error {
	query := `DELETE FROM arbiter_assignments WHERE tournament_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, tournamentID, userID)
	return err
}

// ListTournamentArbiters lists all arbiters for a tournament
func (r *Repository) ListTournamentArbiters(ctx context.Context, tournamentID uuid.UUID) ([]ArbiterAssignment, error) {
	query := `
		SELECT tournament_id, user_id, role
		FROM arbiter_assignments
		WHERE tournament_id = $1
		ORDER BY role, user_id`

	rows, err := r.pool.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []ArbiterAssignment
	for rows.Next() {
		var a ArbiterAssignment
		if err := rows.Scan(&a.TournamentID, &a.UserID, &a.Role); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

// IsOrganizationMember checks if a user is a member of an organization
func (r *Repository) IsOrganizationMember(ctx context.Context, organizationID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM organization_members
			WHERE organization_id = $1 AND user_id = $2
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, organizationID, userID).Scan(&exists)
	return exists, err
}

// CreateAuditLog creates an audit log entry
func (r *Repository) CreateAuditLog(ctx context.Context, action *AuditAction) error {
	query := `
		INSERT INTO arbiter_audit_log (
			id, tournament_id, arbiter_id, action, entity_type, entity_id,
			old_value, new_value, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		action.ID, action.TournamentID, action.ArbiterID, action.Action,
		action.EntityType, action.EntityID, action.OldValue, action.NewValue,
		action.Reason, action.CreatedAt,
	)
	return err
}

// GetAuditLog retrieves audit log entries for a tournament
func (r *Repository) GetAuditLog(ctx context.Context, tournamentID uuid.UUID, limit int) ([]AuditAction, error) {
	query := `
		SELECT id, tournament_id, arbiter_id, action, entity_type, entity_id,
			   COALESCE(old_value, '') as old_value,
			   COALESCE(new_value, '') as new_value,
			   COALESCE(reason, '') as reason,
			   created_at
		FROM arbiter_audit_log
		WHERE tournament_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, tournamentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []AuditAction
	for rows.Next() {
		var a AuditAction
		if err := rows.Scan(
			&a.ID, &a.TournamentID, &a.ArbiterID, &a.Action, &a.EntityType, &a.EntityID,
			&a.OldValue, &a.NewValue, &a.Reason, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// GetAuditLogByEntity retrieves audit log entries for a specific entity
func (r *Repository) GetAuditLogByEntity(ctx context.Context, entityType string, entityID uuid.UUID, limit int) ([]AuditAction, error) {
	query := `
		SELECT id, tournament_id, arbiter_id, action, entity_type, entity_id,
			   COALESCE(old_value, '') as old_value,
			   COALESCE(new_value, '') as new_value,
			   COALESCE(reason, '') as reason,
			   created_at
		FROM arbiter_audit_log
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := r.pool.Query(ctx, query, entityType, entityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []AuditAction
	for rows.Next() {
		var a AuditAction
		if err := rows.Scan(
			&a.ID, &a.TournamentID, &a.ArbiterID, &a.Action, &a.EntityType, &a.EntityID,
			&a.OldValue, &a.NewValue, &a.Reason, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// CreateAnnouncement creates a tournament announcement
func (r *Repository) CreateAnnouncement(ctx context.Context, announcement *Announcement) error {
	query := `
		INSERT INTO tournament_announcements (
			id, tournament_id, title, message, priority, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		announcement.ID, announcement.TournamentID, announcement.Title,
		announcement.Message, announcement.Priority, announcement.CreatedBy,
		announcement.CreatedAt,
	)
	return err
}

// GetAnnouncement retrieves an announcement by ID
func (r *Repository) GetAnnouncement(ctx context.Context, id uuid.UUID) (*Announcement, error) {
	query := `
		SELECT id, tournament_id, title, message, priority, created_by, created_at
		FROM tournament_announcements
		WHERE id = $1`

	var a Announcement
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.TournamentID, &a.Title, &a.Message, &a.Priority, &a.CreatedBy, &a.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

// ListAnnouncements lists announcements for a tournament
func (r *Repository) ListAnnouncements(ctx context.Context, tournamentID uuid.UUID, activeOnly bool) ([]Announcement, error) {
	query := `
		SELECT id, tournament_id, title, message, priority, created_by, created_at
		FROM tournament_announcements
		WHERE tournament_id = $1`

	if activeOnly {
		query += ` AND is_active = true`
	}

	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var announcements []Announcement
	for rows.Next() {
		var a Announcement
		if err := rows.Scan(
			&a.ID, &a.TournamentID, &a.Title, &a.Message, &a.Priority, &a.CreatedBy, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		announcements = append(announcements, a)
	}
	return announcements, rows.Err()
}

// DeactivateAnnouncement marks an announcement as inactive
func (r *Repository) DeactivateAnnouncement(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tournament_announcements SET is_active = false WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// DeleteAnnouncement deletes an announcement
func (r *Repository) DeleteAnnouncement(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tournament_announcements WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// GetRecentAuditCount gets count of recent audit actions (for rate limiting)
func (r *Repository) GetRecentAuditCount(ctx context.Context, arbiterID uuid.UUID, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*) FROM arbiter_audit_log
		WHERE arbiter_id = $1 AND created_at > $2`

	var count int
	err := r.pool.QueryRow(ctx, query, arbiterID, since).Scan(&count)
	return count, err
}
