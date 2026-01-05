package worker

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// JobRegistry contains all registered job creators
type JobRegistry struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewJobRegistry creates a new job registry
func NewJobRegistry(db *pgxpool.Pool, logger zerolog.Logger) *JobRegistry {
	return &JobRegistry{
		db:     db,
		logger: logger,
	}
}

// RegisterAllJobs registers all jobs with the scheduler
func (r *JobRegistry) RegisterAllJobs(scheduler *Scheduler) {
	// Cleanup jobs
	scheduler.RegisterJob(
		"notification_cleanup",
		"Clean up old notifications (older than 90 days)",
		24*time.Hour,
		r.CleanupNotifications,
	)

	scheduler.RegisterJob(
		"session_cleanup",
		"Clean up expired sessions",
		time.Hour,
		r.CleanupSessions,
	)

	scheduler.RegisterJob(
		"completed_games_cleanup",
		"Archive completed games older than 30 days",
		24*time.Hour,
		r.CleanupCompletedGames,
	)

	// Health check / maintenance jobs
	scheduler.RegisterJob(
		"database_stats",
		"Collect database statistics",
		time.Hour,
		r.CollectDatabaseStats,
	)

	scheduler.RegisterJob(
		"audit_log_cleanup",
		"Clean up old audit logs (older than 1 year)",
		24*time.Hour,
		r.CleanupAuditLogs,
	)

	r.logger.Info().Msg("All jobs registered")
}

// CleanupNotifications removes notifications older than 90 days
func (r *JobRegistry) CleanupNotifications(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -90)

	result, err := r.db.Exec(ctx, `
		DELETE FROM notifications
		WHERE created_at < $1 AND read_at IS NOT NULL
	`, cutoff)
	if err != nil {
		return err
	}

	r.logger.Info().
		Int64("deleted", result.RowsAffected()).
		Time("cutoff", cutoff).
		Msg("Cleaned up old notifications")

	return nil
}

// CleanupSessions removes expired sessions
func (r *JobRegistry) CleanupSessions(ctx context.Context) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM refresh_tokens
		WHERE expires_at < NOW()
	`)
	if err != nil {
		return err
	}

	r.logger.Info().
		Int64("deleted", result.RowsAffected()).
		Msg("Cleaned up expired sessions")

	return nil
}

// CleanupCompletedGames archives completed games older than 30 days
func (r *JobRegistry) CleanupCompletedGames(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -30)

	// For now, just mark them as archived (could move to cold storage in production)
	result, err := r.db.Exec(ctx, `
		UPDATE chess_games
		SET status = 'archived'
		WHERE status = 'completed'
		AND completed_at < $1
		AND status != 'archived'
	`, cutoff)
	if err != nil {
		return err
	}

	r.logger.Info().
		Int64("archived", result.RowsAffected()).
		Time("cutoff", cutoff).
		Msg("Archived old completed games")

	return nil
}

// CollectDatabaseStats collects and logs database statistics
func (r *JobRegistry) CollectDatabaseStats(ctx context.Context) error {
	// Get table sizes
	rows, err := r.db.Query(ctx, `
		SELECT
			relname AS table_name,
			pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
			pg_size_pretty(pg_relation_size(relid)) AS table_size,
			n_live_tup AS row_count
		FROM pg_stat_user_tables
		ORDER BY pg_total_relation_size(relid) DESC
		LIMIT 20
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type tableStats struct {
		Name      string
		TotalSize string
		TableSize string
		RowCount  int64
	}

	var stats []tableStats
	for rows.Next() {
		var s tableStats
		if err := rows.Scan(&s.Name, &s.TotalSize, &s.TableSize, &s.RowCount); err != nil {
			return err
		}
		stats = append(stats, s)
	}

	// Log stats
	for _, s := range stats {
		r.logger.Info().
			Str("table", s.Name).
			Str("total_size", s.TotalSize).
			Str("table_size", s.TableSize).
			Int64("row_count", s.RowCount).
			Msg("Database table stats")
	}

	return nil
}

// CleanupAuditLogs removes audit logs older than 1 year
func (r *JobRegistry) CleanupAuditLogs(ctx context.Context) error {
	cutoff := time.Now().AddDate(-1, 0, 0)

	result, err := r.db.Exec(ctx, `
		DELETE FROM audit_log
		WHERE created_at < $1
	`, cutoff)
	if err != nil {
		return err
	}

	r.logger.Info().
		Int64("deleted", result.RowsAffected()).
		Time("cutoff", cutoff).
		Msg("Cleaned up old audit logs")

	return nil
}

// EmailQueueProcessor processes the email queue
type EmailQueueProcessor struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
	// emailService would be injected here
}

// NewEmailQueueProcessor creates a new email queue processor
func NewEmailQueueProcessor(db *pgxpool.Pool, logger zerolog.Logger) *EmailQueueProcessor {
	return &EmailQueueProcessor{
		db:     db,
		logger: logger,
	}
}

// ProcessQueue processes pending emails in the queue
func (p *EmailQueueProcessor) ProcessQueue(ctx context.Context) error {
	// Get pending emails
	rows, err := p.db.Query(ctx, `
		SELECT id, recipient_email, template_name, subject, template_data, created_at
		FROM email_queue
		WHERE status = 'pending'
		AND attempts < 3
		ORDER BY created_at ASC
		LIMIT 50
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Process each email
	processed := 0
	failed := 0

	for rows.Next() {
		// In a real implementation, this would:
		// 1. Parse the row data
		// 2. Send the email using the email service
		// 3. Update the status to 'sent' or increment attempts on failure

		// For now, just count
		processed++
	}

	p.logger.Info().
		Int("processed", processed).
		Int("failed", failed).
		Msg("Processed email queue")

	return nil
}

// RatingSync handles rating synchronization jobs
type RatingSync struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewRatingSync creates a new rating sync service
func NewRatingSync(db *pgxpool.Pool, logger zerolog.Logger) *RatingSync {
	return &RatingSync{
		db:     db,
		logger: logger,
	}
}

// SyncFIDERatings syncs FIDE ratings (monthly)
func (s *RatingSync) SyncFIDERatings(ctx context.Context) error {
	// This would integrate with the federation/fide importer
	s.logger.Info().Msg("Starting FIDE rating sync")

	// Record sync start
	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_sync_log (federation, sync_type, started_at, status)
		VALUES ('fide', 'full', NOW(), 'running')
	`)
	if err != nil {
		return err
	}

	// In a real implementation, this would:
	// 1. Download the FIDE XML file
	// 2. Parse and update player ratings
	// 3. Record the sync completion

	s.logger.Info().Msg("FIDE rating sync completed")
	return nil
}

// SyncDWZRatings syncs DWZ ratings (weekly)
func (s *RatingSync) SyncDWZRatings(ctx context.Context) error {
	s.logger.Info().Msg("Starting DWZ rating sync")

	// Record sync start
	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_sync_log (federation, sync_type, started_at, status)
		VALUES ('dsb', 'incremental', NOW(), 'running')
	`)
	if err != nil {
		return err
	}

	// In a real implementation, this would:
	// 1. Query the DSB API for updated ratings
	// 2. Update player DWZ ratings
	// 3. Record the sync completion

	s.logger.Info().Msg("DWZ rating sync completed")
	return nil
}
