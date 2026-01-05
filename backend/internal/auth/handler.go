package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/openpairing/openpairing/pkg/response"
	"github.com/openpairing/openpairing/pkg/validator"
)

// Handler handles authentication HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register godoc
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 201 {object} User
// @Failure 400 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.Email = validator.SanitizeString(req.Email)
	req.FirstName = validator.SanitizeString(req.FirstName)
	req.LastName = validator.SanitizeString(req.LastName)

	// Validate
	errors := validator.ValidationErrors{}

	if !validator.ValidateEmail(req.Email) {
		errors.Add("email", "Invalid email address")
	}

	if valid, msg := validator.ValidatePassword(req.Password); !valid {
		errors.Add("password", msg)
	}

	if !validator.ValidateName(req.FirstName) {
		errors.Add("firstName", "First name is required and must be less than 100 characters")
	}

	if !validator.ValidateName(req.LastName) {
		errors.Add("lastName", "Last name is required and must be less than 100 characters")
	}

	if errors.HasErrors() {
		response.UnprocessableEntity(c, "Validation failed", errors)
		return
	}

	user, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		switch err {
		case ErrUserExists:
			response.Conflict(c, "A user with this email already exists")
		default:
			response.InternalServerError(c, "Failed to register user")
		}
		return
	}

	response.Created(c, gin.H{"user": user})
}

// Login godoc
// @Summary Login a user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} TokenPair
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	req.Email = validator.SanitizeString(req.Email)

	user, tokens, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			response.Unauthorized(c, "Invalid email or password")
		default:
			response.InternalServerError(c, "Failed to login")
		}
		return
	}

	// Set refresh token as HTTP-only cookie
	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		60*60*24*7, // 7 days
		"/",
		"",
		c.Request.TLS != nil, // Secure flag
		true,                 // HTTP-only
	)

	response.Success(c, http.StatusOK, gin.H{
		"user":        user,
		"accessToken": tokens.AccessToken,
		"expiresAt":   tokens.ExpiresAt,
	})
}

// Logout godoc
// @Summary Logout a user
// @Tags auth
// @Produce json
// @Success 204
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		_ = h.service.Logout(c.Request.Context(), refreshToken)
	}

	// Clear the refresh token cookie
	c.SetCookie("refresh_token", "", -1, "/", "", c.Request.TLS != nil, true)
	response.NoContent(c)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Tags auth
// @Produce json
// @Success 200 {object} TokenPair
// @Failure 401 {object} response.Error
// @Router /auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		response.Unauthorized(c, "No refresh token provided")
		return
	}

	user, tokens, err := h.service.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		switch err {
		case ErrInvalidToken, ErrExpiredToken, ErrTokenRevoked:
			// Clear invalid cookie
			c.SetCookie("refresh_token", "", -1, "/", "", c.Request.TLS != nil, true)
			response.Unauthorized(c, "Invalid or expired refresh token")
		default:
			response.InternalServerError(c, "Failed to refresh token")
		}
		return
	}

	// Update refresh token cookie
	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		60*60*24*7,
		"/",
		"",
		c.Request.TLS != nil,
		true,
	)

	response.Success(c, http.StatusOK, gin.H{
		"user":        user,
		"accessToken": tokens.AccessToken,
		"expiresAt":   tokens.ExpiresAt,
	})
}

// ForgotPassword godoc
// @Summary Request password reset
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{email=string} true "Email"
// @Success 200
// @Router /auth/forgot-password [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	req.Email = validator.SanitizeString(req.Email)

	if !validator.ValidateEmail(req.Email) {
		response.BadRequest(c, "Invalid email address")
		return
	}

	// Always return success to prevent email enumeration
	// Email is sent within RequestPasswordReset if user exists
	_, _ = h.service.RequestPasswordReset(c.Request.Context(), req.Email)

	response.Success(c, http.StatusOK, gin.H{
		"message": "If an account with that email exists, a password reset link has been sent",
	})
}

// ResetPassword godoc
// @Summary Reset password with token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{token=string,password=string} true "Reset data"
// @Success 200
// @Failure 400 {object} response.Error
// @Router /auth/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	if req.Token == "" {
		response.BadRequest(c, "Reset token is required")
		return
	}

	if valid, msg := validator.ValidatePassword(req.Password); !valid {
		response.UnprocessableEntity(c, "Validation failed", validator.ValidationErrors{"password": msg})
		return
	}

	err := h.service.ResetPassword(c.Request.Context(), req.Token, req.Password)
	if err != nil {
		switch err {
		case ErrInvalidResetToken:
			response.BadRequest(c, "Invalid or expired reset token")
		default:
			response.InternalServerError(c, "Failed to reset password")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Password has been reset successfully",
	})
}

// GetCurrentUser godoc
// @Summary Get current user info
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} User
// @Failure 401 {object} response.Error
// @Router /auth/me [get]
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Not authenticated")
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		switch err {
		case ErrUserNotFound:
			response.NotFound(c, "User not found")
		default:
			response.InternalServerError(c, "Failed to get user")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"user": user})
}

// VerifyEmail godoc
// @Summary Verify email address
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{token=string} true "Verification token"
// @Success 200
// @Failure 400 {object} response.Error
// @Router /auth/verify-email [post]
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	if req.Token == "" {
		response.BadRequest(c, "Verification token is required")
		return
	}

	err := h.service.VerifyEmail(c.Request.Context(), req.Token)
	if err != nil {
		switch err {
		case ErrInvalidVerificationToken:
			response.BadRequest(c, "Invalid or expired verification token")
		case ErrEmailAlreadyVerified:
			response.BadRequest(c, "Email is already verified")
		default:
			response.InternalServerError(c, "Failed to verify email")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Email verified successfully",
	})
}

// ResendVerificationEmail godoc
// @Summary Resend verification email
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200
// @Failure 400 {object} response.Error
// @Router /auth/resend-verification [post]
func (h *Handler) ResendVerificationEmail(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Not authenticated")
		return
	}

	err := h.service.ResendVerificationEmail(c.Request.Context(), userID.(string))
	if err != nil {
		switch err {
		case ErrUserNotFound:
			response.NotFound(c, "User not found")
		case ErrEmailAlreadyVerified:
			response.BadRequest(c, "Email is already verified")
		default:
			response.InternalServerError(c, "Failed to send verification email")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Verification email sent",
	})
}
