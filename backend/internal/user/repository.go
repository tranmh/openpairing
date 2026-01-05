package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for users
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new user repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, role, preferred_language, email_verified,
		       email_verified_at, last_login_at, display_name, created_at, updated_at
		FROM users
		WHERE id = $1`

	var u User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.Role, &u.PreferredLanguage, &u.EmailVerified,
		&u.EmailVerifiedAt, &u.LastLoginAt, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &u, nil
}

// GetUserProfile retrieves a user's full profile including player data
func (r *Repository) GetUserProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	query := `
		SELECT u.id, u.email, u.role, u.preferred_language, u.email_verified,
		       u.email_verified_at, u.last_login_at, u.display_name, u.created_at, u.updated_at,
		       p.id as player_id, p.first_name, p.last_name, p.fide_id, p.dsb_id,
		       p.oesb_id, p.ssb_id, p.fide_standard, p.dwz, p.fide_title,
		       p.club_id, c.name as club_name, p.federation
		FROM users u
		LEFT JOIN players p ON p.user_id = u.id
		LEFT JOIN clubs c ON c.id = p.club_id
		WHERE u.id = $1`

	var profile UserProfile
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&profile.ID, &profile.Email, &profile.Role, &profile.PreferredLanguage, &profile.EmailVerified,
		&profile.EmailVerifiedAt, &profile.LastLoginAt, &profile.DisplayName, &profile.CreatedAt, &profile.UpdatedAt,
		&profile.PlayerID, &profile.FirstName, &profile.LastName, &profile.FideID, &profile.DsbID,
		&profile.OesbID, &profile.SsbID, &profile.FideRating, &profile.DwzRating, &profile.FideTitle,
		&profile.ClubID, &profile.ClubName, &profile.Federation,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &profile, nil
}

// UpdateUser updates a user's basic information
func (r *Repository) UpdateUser(ctx context.Context, id uuid.UUID, req *UpdateProfileRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	argNum := 1

	if req.DisplayName != nil {
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argNum))
		args = append(args, *req.DisplayName)
		argNum++
	}
	if req.PreferredLanguage != nil {
		setClauses = append(setClauses, fmt.Sprintf("preferred_language = $%d", argNum))
		args = append(args, *req.PreferredLanguage)
		argNum++
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE users
		SET %s, updated_at = NOW()
		WHERE id = $%d`,
		strings.Join(setClauses, ", "), argNum)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdatePlayerProfile updates the linked player profile
func (r *Repository) UpdatePlayerProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) error {
	// First check if user has a linked player
	var playerID *uuid.UUID
	err := r.pool.QueryRow(ctx, "SELECT id FROM players WHERE user_id = $1", userID).Scan(&playerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil // No linked player, skip
		}
		return err
	}

	setClauses := []string{}
	args := []interface{}{}
	argNum := 1

	if req.FirstName != nil {
		setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argNum))
		args = append(args, *req.FirstName)
		argNum++
	}
	if req.LastName != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argNum))
		args = append(args, *req.LastName)
		argNum++
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, playerID)
	query := fmt.Sprintf(`
		UPDATE players
		SET %s, updated_at = NOW()
		WHERE id = $%d`,
		strings.Join(setClauses, ", "), argNum)

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

// GetPasswordHash retrieves the password hash for a user
func (r *Repository) GetPasswordHash(ctx context.Context, userID uuid.UUID) (string, error) {
	var hash string
	err := r.pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("user not found")
		}
		return "", err
	}
	return hash, nil
}

// UpdatePassword updates a user's password
func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.pool.Exec(ctx, query, passwordHash, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// LinkPlayer links a player profile to a user account
func (r *Repository) LinkPlayer(ctx context.Context, userID, playerID uuid.UUID) error {
	// Check if player is already linked
	var existingUserID *uuid.UUID
	err := r.pool.QueryRow(ctx, "SELECT user_id FROM players WHERE id = $1", playerID).Scan(&existingUserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("player not found")
		}
		return err
	}

	if existingUserID != nil {
		return fmt.Errorf("player already linked to another account")
	}

	// Link the player
	query := `UPDATE players SET user_id = $1, updated_at = NOW() WHERE id = $2`
	_, err = r.pool.Exec(ctx, query, userID, playerID)
	return err
}

// UnlinkPlayer removes the link between a user and player profile
func (r *Repository) UnlinkPlayer(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE players SET user_id = NULL, updated_at = NOW() WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// GetUserSessions retrieves active sessions for a user
func (r *Repository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]UserSession, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []UserSession
	for rows.Next() {
		var s UserSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// RevokeSession revokes a specific session
func (r *Repository) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, sessionID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found")
	}
	return nil
}

// RevokeAllSessions revokes all sessions for a user
func (r *Repository) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// GetUserStats retrieves statistics for a user
func (r *Repository) GetUserStats(ctx context.Context, userID uuid.UUID) (*UserStats, error) {
	// Get player ID
	var playerID *uuid.UUID
	err := r.pool.QueryRow(ctx, "SELECT id FROM players WHERE user_id = $1", userID).Scan(&playerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &UserStats{}, nil // No linked player
		}
		return nil, err
	}

	// Get tournament participation stats
	query := `
		SELECT
			COUNT(DISTINCT tp.tournament_id) as tournaments,
			COALESCE(SUM(tp.wins + tp.draws + tp.losses), 0) as games,
			COALESCE(SUM(tp.wins), 0) as wins,
			COALESCE(SUM(tp.draws), 0) as draws,
			COALESCE(SUM(tp.losses), 0) as losses
		FROM tournament_participants tp
		WHERE tp.player_id = $1`

	var stats UserStats
	err = r.pool.QueryRow(ctx, query, playerID).Scan(
		&stats.TournamentsPlayed, &stats.GamesPlayed,
		&stats.Wins, &stats.Draws, &stats.Losses,
	)
	if err != nil {
		return nil, err
	}

	if stats.GamesPlayed > 0 {
		stats.WinRate = float64(stats.Wins) / float64(stats.GamesPlayed) * 100
	}

	return &stats, nil
}
