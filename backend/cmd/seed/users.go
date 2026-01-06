package main

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// UserSeeder handles user seeding
type UserSeeder struct {
	pool *pgxpool.Pool
}

// NewUserSeeder creates a new user seeder
func NewUserSeeder(pool *pgxpool.Pool) *UserSeeder {
	return &UserSeeder{pool: pool}
}

// AdminConfig defines the admin user configuration
type AdminConfig struct {
	Email    string
	Password string
}

// DefaultAdminConfig returns the default admin configuration
func DefaultAdminConfig() AdminConfig {
	return AdminConfig{
		Email:    "admin@openpairing.org",
		Password: "admin123",
	}
}

// SeedAdmin creates the default admin user
func (s *UserSeeder) SeedAdmin(ctx context.Context, config AdminConfig) (uuid.UUID, error) {
	// Check if admin already exists
	var existingID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM users WHERE email = $1
	`, config.Email).Scan(&existingID)

	if err == nil {
		log.Info().Str("email", config.Email).Msg("Admin user already exists, skipping")
		return existingID, nil
	}

	// Hash password with bcrypt cost 12 (same as auth service)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(config.Password), 12)
	if err != nil {
		return uuid.Nil, err
	}

	// Create admin user
	id := uuid.New()
	now := time.Now()

	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, preferred_language, email_verified, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, config.Email, string(passwordHash), "admin", "de", true, now, now, now)

	if err != nil {
		return uuid.Nil, err
	}

	log.Info().
		Str("email", config.Email).
		Str("id", id.String()).
		Msg("Admin user created")

	return id, nil
}

// SeedTestUsers creates additional test users for different roles
func (s *UserSeeder) SeedTestUsers(ctx context.Context) (map[string]uuid.UUID, error) {
	users := map[string]uuid.UUID{}

	testUsers := []struct {
		Email    string
		Password string
		Role     string
	}{
		{"organizer@openpairing.org", "organizer123", "organizer"},
		{"arbiter@openpairing.org", "arbiter123", "arbiter"},
		{"player@openpairing.org", "player123", "player"},
	}

	for _, u := range testUsers {
		// Check if user already exists
		var existingID uuid.UUID
		err := s.pool.QueryRow(ctx, `
			SELECT id FROM users WHERE email = $1
		`, u.Email).Scan(&existingID)

		if err == nil {
			log.Debug().Str("email", u.Email).Msg("Test user already exists, skipping")
			users[u.Role] = existingID
			continue
		}

		// Hash password
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(u.Password), 12)
		if err != nil {
			return nil, err
		}

		// Create user
		id := uuid.New()
		now := time.Now()

		_, err = s.pool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, role, preferred_language, email_verified, email_verified_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, id, u.Email, string(passwordHash), u.Role, "de", true, now, now, now)

		if err != nil {
			return nil, err
		}

		users[u.Role] = id
		log.Debug().Str("email", u.Email).Str("role", u.Role).Msg("Test user created")
	}

	return users, nil
}

// ClearUsers removes all seeded users (for force cleanup)
func (s *UserSeeder) ClearUsers(ctx context.Context) error {
	// Delete seeded users by email pattern
	_, err := s.pool.Exec(ctx, `
		DELETE FROM users WHERE email LIKE '%@openpairing.org'
	`)
	return err
}
