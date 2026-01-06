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
	ErrUserNotFound              = errors.New("user not found")
	ErrUserExists                = errors.New("user already exists")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrTokenRevoked              = errors.New("token has been revoked")
	ErrInvalidResetToken         = errors.New("invalid or expired reset token")
	ErrInvalidVerificationToken  = errors.New("invalid or expired verification token")
	ErrEmailAlreadyVerified      = errors.New("email already verified")
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

// EmailSender interface for sending emails
type EmailSender interface {
	SendPasswordReset(ctx context.Context, to, recipientName, resetLink string) error
	SendEmailVerification(ctx context.Context, to, recipientName, verificationLink string) error
}

// Service handles authentication operations
type Service struct {
	db           *pgxpool.Pool
	jwt          *JWTService
	bcryptCost   int
	emailService EmailSender
	appBaseURL   string
}

// NewService creates a new auth service
func NewService(db *pgxpool.Pool, cfg *config.JWTConfig, emailService EmailSender, appBaseURL string) *Service {
	return &Service{
		db:           db,
		jwt:          NewJWTService(cfg),
		bcryptCost:   12,
		emailService: emailService,
		appBaseURL:   appBaseURL,
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

	// Send verification email
	if s.emailService != nil {
		go func() {
			ctx := context.Background()
			_ = s.sendVerificationEmail(ctx, userID, req.Email, req.FirstName)
		}()
	}

	log.Info().Str("user_id", userID).Str("email", req.Email).Msg("User registered")
	return user, nil
}

// sendVerificationEmail creates a verification token and sends email
func (s *Service) sendVerificationEmail(ctx context.Context, userID, email, firstName string) error {
	// Generate verification token
	token, err := GenerateRandomToken(32)
	if err != nil {
		return err
	}

	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err = s.db.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to create verification token")
		return err
	}

	// Send verification email
	recipientName := firstName
	if recipientName == "" {
		recipientName = "User"
	}
	verificationLink := s.appBaseURL + "/verify-email?token=" + token

	if err := s.emailService.SendEmailVerification(ctx, email, recipientName, verificationLink); err != nil {
		log.Error().Err(err).Str("email", email).Msg("Failed to send verification email")
		return err
	}

	log.Info().Str("email", email).Msg("Verification email sent")
	return nil
}

// ResendVerificationEmail resends verification email for a user
func (s *Service) ResendVerificationEmail(ctx context.Context, userID string) error {
	// Get user info
	var email, firstName string
	var emailVerified bool
	err := s.db.QueryRow(ctx, `
		SELECT u.email, u.email_verified, COALESCE(p.first_name, '')
		FROM users u
		LEFT JOIN players p ON p.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&email, &emailVerified, &firstName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	if emailVerified {
		return ErrEmailAlreadyVerified
	}

	// Invalidate existing tokens
	_, _ = s.db.Exec(ctx, `
		UPDATE email_verification_tokens SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)

	return s.sendVerificationEmail(ctx, userID, email, firstName)
}

// VerifyEmail verifies a user's email using a verification token
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	tokenHash := hashToken(token)

	// Get and validate token
	var userID string
	var expiresAt time.Time
	var usedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT user_id, expires_at, used_at FROM email_verification_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&userID, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidVerificationToken
		}
		return err
	}

	if usedAt != nil || time.Now().After(expiresAt) {
		return ErrInvalidVerificationToken
	}

	// Check if already verified
	var emailVerified bool
	err = s.db.QueryRow(ctx, "SELECT email_verified FROM users WHERE id = $1", userID).Scan(&emailVerified)
	if err != nil {
		return err
	}
	if emailVerified {
		return ErrEmailAlreadyVerified
	}

	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Update user as verified
	_, err = tx.Exec(ctx, `
		UPDATE users SET email_verified = TRUE, email_verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return err
	}

	// Mark token as used
	_, err = tx.Exec(ctx, "UPDATE email_verification_tokens SET used_at = NOW() WHERE token_hash = $1", tokenHash)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	log.Info().Str("user_id", userID).Msg("Email verified successfully")
	return nil
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

	tokenHash := hashToken(refreshToken)

	// Start transaction to prevent race conditions
	// This ensures check-and-revoke is atomic
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	// Check if token is revoked AND lock the row to prevent concurrent refreshes
	// FOR UPDATE ensures no other transaction can read or modify this row until we commit
	var revoked bool
	err = tx.QueryRow(ctx, `
		SELECT revoked_at IS NOT NULL FROM refresh_tokens
		WHERE token_hash = $1 AND user_id = $2
		FOR UPDATE
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

	// Immediately revoke the token within the same transaction
	// This happens before generating new tokens to ensure atomicity
	_, err = tx.Exec(ctx, "UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1", tokenHash)
	if err != nil {
		return nil, nil, err
	}

	// Get user (still within transaction for consistency)
	var user User
	err = tx.QueryRow(ctx, `
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

	// Commit the transaction (revokes the old token)
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	// Generate new tokens (outside transaction - this creates a new token)
	tokens, err := s.generateTokenPair(ctx, &user)
	if err != nil {
		// Note: The old token is already revoked at this point.
		// If token generation fails, user will need to re-login.
		// This is safer than leaving the old token valid.
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

// RequestPasswordReset creates a password reset token and sends email
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	// Check if user exists and get user info
	var userID string
	var firstName, lastName string
	err := s.db.QueryRow(ctx, `
		SELECT u.id, COALESCE(p.first_name, ''), COALESCE(p.last_name, '')
		FROM users u
		LEFT JOIN players p ON p.user_id = u.id
		WHERE u.email = $1
	`, email).Scan(&userID, &firstName, &lastName)
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

	// Send password reset email
	if s.emailService != nil {
		recipientName := firstName
		if recipientName == "" {
			recipientName = "User"
		}
		resetLink := s.appBaseURL + "/reset-password?token=" + token
		if err := s.emailService.SendPasswordReset(ctx, email, recipientName, resetLink); err != nil {
			log.Error().Err(err).Str("email", email).Msg("Failed to send password reset email")
			// Don't return error to user - token is still created
		} else {
			log.Info().Str("email", email).Msg("Password reset email sent")
		}
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
