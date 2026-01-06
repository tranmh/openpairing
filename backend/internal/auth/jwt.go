package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tranmh/openpairing/internal/config"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenType represents the type of JWT token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents the JWT claims
type Claims struct {
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Type     TokenType `json:"type"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewJWTService creates a new JWT service
func NewJWTService(cfg *config.JWTConfig) *JWTService {
	return &JWTService{
		accessSecret:  []byte(cfg.Secret),
		refreshSecret: []byte(cfg.RefreshSecret),
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
		issuer:        cfg.Issuer,
	}
}

// GenerateAccessToken generates a new access token
func (s *JWTService) GenerateAccessToken(userID, email, role string) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.accessExpiry)

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.accessSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// GenerateRefreshToken generates a new refresh token
func (s *JWTService) GenerateRefreshToken(userID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.refreshExpiry)

	claims := Claims{
		UserID: userID,
		Type:   RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.refreshSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// ValidateAccessToken validates an access token and returns the claims
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, s.accessSecret, AccessToken)
}

// ValidateRefreshToken validates a refresh token and returns the claims
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, s.refreshSecret, RefreshToken)
}

func (s *JWTService) validateToken(tokenString string, secret []byte, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Type != expectedType {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateRandomToken generates a random token for password reset, etc.
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
