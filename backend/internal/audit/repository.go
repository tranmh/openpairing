package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for audit log data operations
type Repository interface {
	Create(ctx context.Context, log *AuditLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*AuditLog, error)
	List(ctx context.Context, params AuditLogListParams) (*AuditLogListResponse, error)
	GetByEntityID(ctx context.Context, entityType string, entityID uuid.UUID, limit int) ([]AuditLog, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]AuditLog, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL audit log repository
func NewRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create creates a new audit log entry
func (r *PostgresRepository) Create(ctx context.Context, log *AuditLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()

	oldValueJSON, err := json.Marshal(log.OldValue)
	if err != nil {
		return fmt.Errorf("failed to marshal old value: %w", err)
	}

	newValueJSON, err := json.Marshal(log.NewValue)
	if err != nil {
		return fmt.Errorf("failed to marshal new value: %w", err)
	}

	query := `
		INSERT INTO audit_logs (id, user_id, action, entity_type, entity_id, old_value, new_value, ip_address, user_agent, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.db.Exec(ctx, query,
		log.ID, log.UserID, log.Action, log.EntityType, log.EntityID,
		oldValueJSON, newValueJSON, log.IPAddress, log.UserAgent, log.Description, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetByID retrieves an audit log by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*AuditLog, error) {
	query := `
		SELECT id, user_id, action, entity_type, entity_id, old_value, new_value, ip_address, user_agent, description, created_at
		FROM audit_logs WHERE id = $1
	`

	var log AuditLog
	var oldValueJSON, newValueJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&log.ID, &log.UserID, &log.Action, &log.EntityType, &log.EntityID,
		&oldValueJSON, &newValueJSON, &log.IPAddress, &log.UserAgent, &log.Description, &log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	if len(oldValueJSON) > 0 {
		json.Unmarshal(oldValueJSON, &log.OldValue)
	}
	if len(newValueJSON) > 0 {
		json.Unmarshal(newValueJSON, &log.NewValue)
	}

	return &log, nil
}

// List lists audit logs with pagination and filtering
func (r *PostgresRepository) List(ctx context.Context, params AuditLogListParams) (*AuditLogListResponse, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if params.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argNum))
		args = append(args, *params.UserID)
		argNum++
	}

	if params.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argNum))
		args = append(args, params.Action)
		argNum++
	}

	if params.EntityType != "" {
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argNum))
		args = append(args, params.EntityType)
		argNum++
	}

	if params.EntityID != nil {
		conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argNum))
		args = append(args, *params.EntityID)
		argNum++
	}

	if params.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argNum))
		args = append(args, *params.StartDate)
		argNum++
	}

	if params.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argNum))
		args = append(args, *params.EndDate)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Apply pagination
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 50
	}

	offset := (params.Page - 1) * params.PageSize
	args = append(args, params.PageSize, offset)

	query := fmt.Sprintf(`
		SELECT id, user_id, action, entity_type, entity_id, old_value, new_value, ip_address, user_agent, description, created_at
		FROM audit_logs %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var oldValueJSON, newValueJSON []byte
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.EntityType, &log.EntityID,
			&oldValueJSON, &newValueJSON, &log.IPAddress, &log.UserAgent, &log.Description, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(oldValueJSON) > 0 {
			json.Unmarshal(oldValueJSON, &log.OldValue)
		}
		if len(newValueJSON) > 0 {
			json.Unmarshal(newValueJSON, &log.NewValue)
		}

		logs = append(logs, log)
	}

	totalPages := (totalCount + params.PageSize - 1) / params.PageSize

	return &AuditLogListResponse{
		Logs:       logs,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByEntityID retrieves audit logs for a specific entity
func (r *PostgresRepository) GetByEntityID(ctx context.Context, entityType string, entityID uuid.UUID, limit int) ([]AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, user_id, action, entity_type, entity_id, old_value, new_value, ip_address, user_agent, description, created_at
		FROM audit_logs
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, entityType, entityID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by entity: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var oldValueJSON, newValueJSON []byte
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.EntityType, &log.EntityID,
			&oldValueJSON, &newValueJSON, &log.IPAddress, &log.UserAgent, &log.Description, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(oldValueJSON) > 0 {
			json.Unmarshal(oldValueJSON, &log.OldValue)
		}
		if len(newValueJSON) > 0 {
			json.Unmarshal(newValueJSON, &log.NewValue)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetByUserID retrieves audit logs for a specific user
func (r *PostgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, user_id, action, entity_type, entity_id, old_value, new_value, ip_address, user_agent, description, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by user: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var oldValueJSON, newValueJSON []byte
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.EntityType, &log.EntityID,
			&oldValueJSON, &newValueJSON, &log.IPAddress, &log.UserAgent, &log.Description, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(oldValueJSON) > 0 {
			json.Unmarshal(oldValueJSON, &log.OldValue)
		}
		if len(newValueJSON) > 0 {
			json.Unmarshal(newValueJSON, &log.NewValue)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// DeleteOlderThan deletes audit logs older than the specified time
func (r *PostgresRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	result, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}
	return result.RowsAffected(), nil
}
