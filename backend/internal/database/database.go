package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tranmh/openpairing/internal/config"
	"github.com/rs/zerolog/log"
)

// DB wraps the connection pool
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new database connection pool
func New(cfg *config.DatabaseConfig) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns / 2)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	log.Info().Str("host", cfg.Host).Str("database", cfg.Name).Msg("Connected to database")

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
}

// Health checks if the database connection is healthy
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
