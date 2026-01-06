package auth

import (
	"context"
	"testing"
	"time"

	"github.com/tranmh/openpairing/internal/config"
)

func TestJWTService_GenerateAndValidateAccessToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret-key-min-32-characters!",
		RefreshSecret: "test-refresh-secret-key-min-32!",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
		Issuer:        "openpairing-test",
	}

	jwtService := NewJWTService(cfg)

	t.Run("GenerateAccessToken_Success", func(t *testing.T) {
		token, expiresAt, err := jwtService.GenerateAccessToken("user-123", "test@example.com", "player")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if token == "" {
			t.Error("Expected non-empty token")
		}

		if expiresAt.Before(time.Now()) {
			t.Error("Expected expiration in the future")
		}
	})

	t.Run("ValidateAccessToken_ValidToken", func(t *testing.T) {
		token, _, err := jwtService.GenerateAccessToken("user-123", "test@example.com", "player")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		claims, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if claims.UserID != "user-123" {
			t.Errorf("Expected userID 'user-123', got %s", claims.UserID)
		}

		if claims.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %s", claims.Email)
		}

		if claims.Role != "player" {
			t.Errorf("Expected role 'player', got %s", claims.Role)
		}

		if claims.Type != AccessToken {
			t.Errorf("Expected type 'access', got %s", claims.Type)
		}
	})

	t.Run("ValidateAccessToken_InvalidToken", func(t *testing.T) {
		_, err := jwtService.ValidateAccessToken("invalid-token")
		if err == nil {
			t.Error("Expected error for invalid token")
		}

		if err != ErrInvalidToken {
			t.Errorf("Expected ErrInvalidToken, got %v", err)
		}
	})

	t.Run("ValidateAccessToken_WrongSecret", func(t *testing.T) {
		// Generate token with different secret
		otherService := NewJWTService(&config.JWTConfig{
			Secret:        "different-secret-key-32-chars!!",
			RefreshSecret: "different-refresh-secret-32!!!!",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
			Issuer:        "other",
		})

		token, _, _ := otherService.GenerateAccessToken("user-123", "test@example.com", "player")

		_, err := jwtService.ValidateAccessToken(token)
		if err == nil {
			t.Error("Expected error for token with wrong secret")
		}
	})

	t.Run("ValidateAccessToken_RefreshTokenAsAccess", func(t *testing.T) {
		// Generate refresh token
		refreshToken, _, _ := jwtService.GenerateRefreshToken("user-123")

		// Try to validate as access token
		_, err := jwtService.ValidateAccessToken(refreshToken)
		if err == nil {
			t.Error("Expected error when validating refresh token as access token")
		}
	})
}

func TestJWTService_RefreshToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret-key-min-32-characters!",
		RefreshSecret: "test-refresh-secret-key-min-32!",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
		Issuer:        "openpairing-test",
	}

	jwtService := NewJWTService(cfg)

	t.Run("GenerateRefreshToken_Success", func(t *testing.T) {
		token, expiresAt, err := jwtService.GenerateRefreshToken("user-123")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if token == "" {
			t.Error("Expected non-empty token")
		}

		if expiresAt.Before(time.Now().Add(6 * 24 * time.Hour)) {
			t.Error("Expected expiration to be about 7 days in the future")
		}
	})

	t.Run("ValidateRefreshToken_ValidToken", func(t *testing.T) {
		token, _, err := jwtService.GenerateRefreshToken("user-123")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		claims, err := jwtService.ValidateRefreshToken(token)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if claims.UserID != "user-123" {
			t.Errorf("Expected userID 'user-123', got %s", claims.UserID)
		}

		if claims.Type != RefreshToken {
			t.Errorf("Expected type 'refresh', got %s", claims.Type)
		}
	})

	t.Run("ValidateRefreshToken_AccessTokenAsRefresh", func(t *testing.T) {
		// Generate access token
		accessToken, _, _ := jwtService.GenerateAccessToken("user-123", "test@example.com", "player")

		// Try to validate as refresh token
		_, err := jwtService.ValidateRefreshToken(accessToken)
		if err == nil {
			t.Error("Expected error when validating access token as refresh token")
		}
	})
}

func TestGenerateRandomToken(t *testing.T) {
	t.Run("GenerateRandomToken_UniqueTokens", func(t *testing.T) {
		token1, err := GenerateRandomToken(32)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		token2, err := GenerateRandomToken(32)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if token1 == token2 {
			t.Error("Expected unique tokens")
		}

		// 32 bytes = 64 hex characters
		if len(token1) != 64 {
			t.Errorf("Expected token length 64, got %d", len(token1))
		}
	})

	t.Run("GenerateRandomToken_DifferentLengths", func(t *testing.T) {
		token16, _ := GenerateRandomToken(16)
		token32, _ := GenerateRandomToken(32)

		if len(token16) != 32 { // 16 bytes = 32 hex chars
			t.Errorf("Expected token length 32, got %d", len(token16))
		}

		if len(token32) != 64 { // 32 bytes = 64 hex chars
			t.Errorf("Expected token length 64, got %d", len(token32))
		}
	})
}

func TestHashToken(t *testing.T) {
	t.Run("HashToken_Deterministic", func(t *testing.T) {
		token := "test-token-123"
		hash1 := hashToken(token)
		hash2 := hashToken(token)

		if hash1 != hash2 {
			t.Error("Expected same hash for same token")
		}
	})

	t.Run("HashToken_DifferentTokens", func(t *testing.T) {
		hash1 := hashToken("token-1")
		hash2 := hashToken("token-2")

		if hash1 == hash2 {
			t.Error("Expected different hashes for different tokens")
		}
	})

	t.Run("HashToken_Length", func(t *testing.T) {
		hash := hashToken("any-token")
		// SHA256 produces 64 hex characters
		if len(hash) != 64 {
			t.Errorf("Expected hash length 64, got %d", len(hash))
		}
	})
}

// Integration test placeholder (requires database)
func TestService_Register(t *testing.T) {
	t.Skip("Requires database connection - run with integration tests")

	// This would test:
	// - Successful registration
	// - Duplicate email error
	// - Player profile creation
}

func TestService_Login(t *testing.T) {
	t.Skip("Requires database connection - run with integration tests")

	// This would test:
	// - Successful login
	// - Invalid credentials
	// - Token generation
}

// Context helper for tests
func testContext() context.Context {
	return context.Background()
}
