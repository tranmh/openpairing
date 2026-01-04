package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpairing/openpairing/internal/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrInvalidResetToken  = errors.New("invalid or expired reset token")
)

// User represents a user in the system
type User struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	PasswordHash      string     `json:"-"`
	Role              string     `json:"role"`
	PreferredLanguage string     `json:"preferredLanguage"`
	EmailVerified     bool       `json:"emailVerified"`
	EmailVerifiedAt   *time.Time `json:"emailVerifiedAt,omitempty"`
	LastLoginAt       *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// Service handles authentication operations
type Service struct {
	db         *pgxpool.Pool
	jwt        *JWTService
	bcryptCost int
}

// NewService creates a new auth service
func NewService(db *pgxpool.Pool, cfg *config.JWTConfig) *Service {
	return &Service{
		db:         db,
		jwt:        NewJWTService(cfg),
		bcryptCost: 12,
	}
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*User, error) {
	// Check if user already exists
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, err
	}

	// Create user
	userID := uuid.New().String()
	user := &User{
		ID:                userID,
		Email:             req.Email,
		Role:              "player",
		PreferredLanguage: "de",
		EmailVerified:     false,
	}

	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Insert user
	err = tx.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, role, preferred_language)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`, userID, req.Email, string(hashedPassword), "player", "de").Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Create player profile
	_, err = tx.Exec(ctx, `
		INSERT INTO players (id, user_id, first_name, last_name)
		VALUES ($1, $2, $3, $4)
	`, uuid.New().String(), userID, req.FirstName, req.LastName)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	log.Info().Str("user_id", userID).Str("email", req.Email).Msg("User registered")
	return user, nil
}

// Login authenticates a user and returns tokens
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error) {
	// Get user
	var user User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, preferred_language, email_verified,
		       email_verified_at, last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`, req.Email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.PreferredLanguage,
		&user.EmailVerified, &user.EmailVerifiedAt, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, &user)
	if err != nil {
		return nil, nil, err
	}

	// Update last login
	_, err = s.db.Exec(ctx, "UPDATE users SET last_login_at = NOW() WHERE id = $1", user.ID)
	if err != nil {
		log.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to update last login")
	}

	log.Info().Str("user_id", user.ID).Msg("User logged in")
	return &user, tokens, nil
}

// RefreshTokens refreshes the token pair using a refresh token
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*User, *TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, nil, err
	}

	// Check if token is revoked
	tokenHash := hashToken(refreshToken)
	var revoked bool
	err = s.db.QueryRow(ctx, `
		SELECT revoked_at IS NOT NULL FROM refresh_tokens
		WHERE token_hash = $1 AND user_id = $2
	`, tokenHash, claims.UserID).Scan(&revoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrInvalidToken
		}
		return nil, nil, err
	}
	if revoked {
		return nil, nil, ErrTokenRevoked
	}

	// Get user
	var user User
	err = s.db.QueryRow(ctx, `
		SELECT id, email, role, preferred_language, email_verified, created_at, updated_at
		FROM users WHERE id = $1
	`, claims.UserID).Scan(
		&user.ID, &user.Email, &user.Role, &user.PreferredLanguage,
		&user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}

	// Revoke old refresh token
	_, err = s.db.Exec(ctx, "UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1", tokenHash)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to revoke old refresh token")
	}

	// Generate new tokens
	tokens, err := s.generateTokenPair(ctx, &user)
	if err != nil {
		return nil, nil, err
	}

	return &user, tokens, nil
}

// Logout revokes the refresh token
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	_, err := s.db.Exec(ctx, "UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1", tokenHash)
	return err
}

// GetUserByID gets a user by ID
func (s *Service) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var user User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, role, preferred_language, email_verified,
		       email_verified_at, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Email, &user.Role, &user.PreferredLanguage,
		&user.EmailVerified, &user.EmailVerifiedAt, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// RequestPasswordReset creates a password reset token
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	// Check if user exists
	var userID string
	err := s.db.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Don't reveal if email exists
			return "", nil
		}
		return "", err
	}

	// Generate reset token
	token, err := GenerateRandomToken(32)
	if err != nil {
		return "", err
	}

	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = s.db.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ResetPassword resets the password using a reset token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	tokenHash := hashToken(token)

	// Get and validate token
	var userID string
	var expiresAt time.Time
	var usedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT user_id, expires_at, used_at FROM password_reset_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&userID, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidResetToken
		}
		return err
	}

	if usedAt != nil || time.Now().After(expiresAt) {
		return ErrInvalidResetToken
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Update password
	_, err = tx.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hashedPassword), userID)
	if err != nil {
		return err
	}

	// Mark token as used
	_, err = tx.Exec(ctx, "UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1", tokenHash)
	if err != nil {
		return err
	}

	// Revoke all refresh tokens for user
	_, err = tx.Exec(ctx, "UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// generateTokenPair generates a new access and refresh token pair
func (s *Service) generateTokenPair(ctx context.Context, user *User) (*TokenPair, error) {
	accessToken, expiresAt, err := s.jwt.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshExpiresAt, err := s.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// Store refresh token hash
	tokenHash := hashToken(refreshToken)
	_, err = s.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, user.ID, tokenHash, refreshExpiresAt)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// hashToken creates a SHA256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
