package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AuthService defines the interface for auth operations used by the handler.
// This interface allows for easier testing by enabling mock implementations.
type AuthService interface {
	Register(ctx context.Context, req *RegisterRequest) (*User, error)
	Login(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshTokens(ctx context.Context, refreshToken string) (*User, *TokenPair, error)
	RequestPasswordReset(ctx context.Context, email string) (*User, error)
	ResetPassword(ctx context.Context, token, newPassword string) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	VerifyEmail(ctx context.Context, token string) error
	ResendVerificationEmail(ctx context.Context, userID string) error
}

// MockAuthService is a mock implementation of AuthService for testing.
type MockAuthService struct {
	RegisterFunc              func(ctx context.Context, req *RegisterRequest) (*User, error)
	LoginFunc                 func(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error)
	LogoutFunc                func(ctx context.Context, refreshToken string) error
	RefreshTokensFunc         func(ctx context.Context, refreshToken string) (*User, *TokenPair, error)
	RequestPasswordResetFunc  func(ctx context.Context, email string) (*User, error)
	ResetPasswordFunc         func(ctx context.Context, token, newPassword string) error
	GetUserByIDFunc           func(ctx context.Context, id string) (*User, error)
	VerifyEmailFunc           func(ctx context.Context, token string) error
	ResendVerificationFunc    func(ctx context.Context, userID string) error
}

func (m *MockAuthService) Register(ctx context.Context, req *RegisterRequest) (*User, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockAuthService) Login(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, req)
	}
	return nil, nil, nil
}

func (m *MockAuthService) Logout(ctx context.Context, refreshToken string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, refreshToken)
	}
	return nil
}

func (m *MockAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*User, *TokenPair, error) {
	if m.RefreshTokensFunc != nil {
		return m.RefreshTokensFunc(ctx, refreshToken)
	}
	return nil, nil, nil
}

func (m *MockAuthService) RequestPasswordReset(ctx context.Context, email string) (*User, error) {
	if m.RequestPasswordResetFunc != nil {
		return m.RequestPasswordResetFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockAuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if m.ResetPasswordFunc != nil {
		return m.ResetPasswordFunc(ctx, token, newPassword)
	}
	return nil
}

func (m *MockAuthService) GetUserByID(ctx context.Context, id string) (*User, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockAuthService) VerifyEmail(ctx context.Context, token string) error {
	if m.VerifyEmailFunc != nil {
		return m.VerifyEmailFunc(ctx, token)
	}
	return nil
}

func (m *MockAuthService) ResendVerificationEmail(ctx context.Context, userID string) error {
	if m.ResendVerificationFunc != nil {
		return m.ResendVerificationFunc(ctx, userID)
	}
	return nil
}

// TestableHandler wraps Handler but uses an interface for testing.
type TestableHandler struct {
	service AuthService
}

func NewTestableHandler(service AuthService) *TestableHandler {
	return &TestableHandler{service: service}
}

// Register handles user registration.
func (h *TestableHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	user, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		switch err {
		case ErrUserExists:
			c.JSON(http.StatusConflict, gin.H{"error": "A user with this email already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// Login handles user login.
func (h *TestableHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	user, tokens, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		}
		return
	}

	c.SetCookie("refresh_token", tokens.RefreshToken, 60*60*24*7, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"user":        user,
		"accessToken": tokens.AccessToken,
		"expiresAt":   tokens.ExpiresAt,
	})
}

// Logout handles user logout.
func (h *TestableHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie("refresh_token")
	if refreshToken != "" {
		_ = h.service.Logout(c.Request.Context(), refreshToken)
	}
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.Status(http.StatusNoContent)
}

// RefreshToken handles token refresh.
func (h *TestableHandler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No refresh token provided"})
		return
	}

	user, tokens, err := h.service.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		switch err {
		case ErrInvalidToken, ErrExpiredToken, ErrTokenRevoked:
			c.SetCookie("refresh_token", "", -1, "/", "", false, true)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token"})
		}
		return
	}

	c.SetCookie("refresh_token", tokens.RefreshToken, 60*60*24*7, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"user":        user,
		"accessToken": tokens.AccessToken,
		"expiresAt":   tokens.ExpiresAt,
	})
}

// GetCurrentUser handles getting the current user.
func (h *TestableHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		switch err {
		case ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func setupTestRouter(handler *TestableHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/auth/register", handler.Register)
	r.POST("/auth/login", handler.Login)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/refresh", handler.RefreshToken)
	r.GET("/auth/me", handler.GetCurrentUser)

	return r
}

// Helper to create test user
func testUser() *User {
	return &User{
		ID:                "user-123",
		Email:             "test@example.com",
		Role:              "player",
		PreferredLanguage: "de",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// Helper to create test tokens
func testTokens() *TokenPair {
	return &TokenPair{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}
}

func TestHandler_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		mockService := &MockAuthService{
			RegisterFunc: func(ctx context.Context, req *RegisterRequest) (*User, error) {
				return testUser(), nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		body := `{"email":"test@example.com","password":"Password123!","firstName":"Test","lastName":"User"}`
		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "user")
	})

	t.Run("user already exists", func(t *testing.T) {
		mockService := &MockAuthService{
			RegisterFunc: func(ctx context.Context, req *RegisterRequest) (*User, error) {
				return nil, ErrUserExists
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		body := `{"email":"existing@example.com","password":"Password123!","firstName":"Test","lastName":"User"}`
		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_Login(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		mockService := &MockAuthService{
			LoginFunc: func(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error) {
				return testUser(), testTokens(), nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		body := `{"email":"test@example.com","password":"Password123!"}`
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "user")
		assert.Contains(t, response, "accessToken")
		assert.Contains(t, response, "expiresAt")

		// Check cookie is set
		cookies := w.Result().Cookies()
		var refreshCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "refresh_token" {
				refreshCookie = c
				break
			}
		}
		require.NotNil(t, refreshCookie, "refresh_token cookie should be set")
		assert.Equal(t, "test-refresh-token", refreshCookie.Value)
		assert.True(t, refreshCookie.HttpOnly)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockService := &MockAuthService{
			LoginFunc: func(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error) {
				return nil, nil, ErrInvalidCredentials
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		body := `{"email":"test@example.com","password":"wrong"}`
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString("invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_Logout(t *testing.T) {
	t.Run("successful logout", func(t *testing.T) {
		logoutCalled := false
		mockService := &MockAuthService{
			LogoutFunc: func(ctx context.Context, refreshToken string) error {
				logoutCalled = true
				assert.Equal(t, "test-refresh-token", refreshToken)
				return nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "test-refresh-token"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.True(t, logoutCalled)

		// Check cookie is cleared
		cookies := w.Result().Cookies()
		var refreshCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "refresh_token" {
				refreshCookie = c
				break
			}
		}
		require.NotNil(t, refreshCookie)
		assert.Equal(t, "", refreshCookie.Value)
		assert.True(t, refreshCookie.MaxAge < 0)
	})

	t.Run("logout without cookie", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/logout", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// The API server listens on plain HTTP behind a TLS-terminating proxy, so the
// refresh_token cookie must be Secure when the proxy reports HTTPS.
func TestRefreshCookieIsSecureBehindTLSProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(nil)
	router := gin.New()
	router.POST("/auth/logout", handler.Logout)

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var refreshCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}
	require.NotNil(t, refreshCookie)
	assert.True(t, refreshCookie.HttpOnly)
	assert.True(t, refreshCookie.Secure, "refresh_token cookie must be Secure behind an HTTPS proxy")
}

func TestHandler_RefreshToken(t *testing.T) {
	t.Run("successful token refresh", func(t *testing.T) {
		mockService := &MockAuthService{
			RefreshTokensFunc: func(ctx context.Context, refreshToken string) (*User, *TokenPair, error) {
				assert.Equal(t, "old-refresh-token", refreshToken)
				return testUser(), testTokens(), nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "old-refresh-token"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "accessToken")
	})

	t.Run("no refresh token provided", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/refresh", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		mockService := &MockAuthService{
			RefreshTokensFunc: func(ctx context.Context, refreshToken string) (*User, *TokenPair, error) {
				return nil, nil, ErrInvalidToken
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "invalid-token"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Cookie should be cleared
		cookies := w.Result().Cookies()
		var refreshCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "refresh_token" {
				refreshCookie = c
				break
			}
		}
		require.NotNil(t, refreshCookie)
		assert.True(t, refreshCookie.MaxAge < 0)
	})

	t.Run("expired refresh token", func(t *testing.T) {
		mockService := &MockAuthService{
			RefreshTokensFunc: func(ctx context.Context, refreshToken string) (*User, *TokenPair, error) {
				return nil, nil, ErrExpiredToken
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired-token"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("revoked refresh token", func(t *testing.T) {
		mockService := &MockAuthService{
			RefreshTokensFunc: func(ctx context.Context, refreshToken string) (*User, *TokenPair, error) {
				return nil, nil, ErrTokenRevoked
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("POST", "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "revoked-token"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandler_GetCurrentUser(t *testing.T) {
	t.Run("successful get current user", func(t *testing.T) {
		mockService := &MockAuthService{
			GetUserByIDFunc: func(ctx context.Context, id string) (*User, error) {
				assert.Equal(t, "user-123", id)
				return testUser(), nil
			},
		}
		handler := NewTestableHandler(mockService)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/auth/me", func(c *gin.Context) {
			c.Set("user_id", "user-123")
			handler.GetCurrentUser(c)
		})

		req := httptest.NewRequest("GET", "/auth/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "user")
	})

	t.Run("not authenticated", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewTestableHandler(mockService)
		router := setupTestRouter(handler)

		req := httptest.NewRequest("GET", "/auth/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		mockService := &MockAuthService{
			GetUserByIDFunc: func(ctx context.Context, id string) (*User, error) {
				return nil, ErrUserNotFound
			},
		}
		handler := NewTestableHandler(mockService)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/auth/me", func(c *gin.Context) {
			c.Set("user_id", "deleted-user")
			handler.GetCurrentUser(c)
		})

		req := httptest.NewRequest("GET", "/auth/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
