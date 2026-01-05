package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpairing/openpairing/pkg/response"
)

// Handler handles user HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new user handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "user").Logger(),
	}
}

// RegisterRoutes registers the user routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// All user routes require authentication
	users := r.Group("/users")
	users.Use(authMiddleware)
	{
		// Profile management
		users.GET("/me", h.GetProfile)
		users.PUT("/me", h.UpdateProfile)
		users.POST("/me/change-password", h.ChangePassword)

		// Player linking
		users.POST("/me/link-player", h.LinkPlayer)
		users.DELETE("/me/link-player", h.UnlinkPlayer)

		// Session management
		users.GET("/me/sessions", h.GetSessions)
		users.DELETE("/me/sessions/:sessionId", h.RevokeSession)
		users.DELETE("/me/sessions", h.RevokeAllSessions)

		// Statistics
		users.GET("/me/stats", h.GetStats)
	}
}

// GetProfile retrieves the current user's profile
// @Summary Get current user profile
// @Description Get the profile of the currently authenticated user
// @Tags users
// @Produce json
// @Success 200 {object} UserProfile
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me [get]
func (h *Handler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "user not found" {
			response.NotFound(c, "User not found")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get profile")
		response.InternalServerError(c, "Failed to get profile")
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile updates the current user's profile
// @Summary Update current user profile
// @Description Update the profile of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Param body body UpdateProfileRequest true "Profile update data"
// @Success 200 {object} UserProfile
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	profile, err := h.service.UpdateProfile(c.Request.Context(), userID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to update profile")
		response.InternalServerError(c, "Failed to update profile")
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ChangePassword changes the current user's password
// @Summary Change password
// @Description Change the password of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Param body body ChangePasswordRequest true "Password change data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me/change-password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	err := h.service.ChangePassword(c.Request.Context(), userID.(uuid.UUID), &req)
	if err != nil {
		if err.Error() == "current password is incorrect" {
			response.BadRequest(c, err.Error())
			return
		}
		if err.Error() == "new password must be different from current password" {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Msg("Failed to change password")
		response.InternalServerError(c, "Failed to change password")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// LinkPlayer links a player profile to the user
// @Summary Link player profile
// @Description Link a player profile to the current user's account
// @Tags users
// @Accept json
// @Produce json
// @Param body body LinkPlayerRequest true "Player link data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me/link-player [post]
func (h *Handler) LinkPlayer(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req LinkPlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	err := h.service.LinkPlayer(c.Request.Context(), userID.(uuid.UUID), &req)
	if err != nil {
		if err.Error() == "player not found" {
			response.NotFound(c, "Player not found")
			return
		}
		if err.Error() == "player already linked to another account" ||
			err.Error() == "user already has a linked player profile" {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Msg("Failed to link player")
		response.InternalServerError(c, "Failed to link player")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Player linked successfully"})
}

// UnlinkPlayer unlinks the player profile from the user
// @Summary Unlink player profile
// @Description Remove the link between the current user and their player profile
// @Tags users
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me/link-player [delete]
func (h *Handler) UnlinkPlayer(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err := h.service.UnlinkPlayer(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to unlink player")
		response.InternalServerError(c, "Failed to unlink player")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Player unlinked successfully"})
}

// GetSessions retrieves the user's active sessions
// @Summary Get active sessions
// @Description Get all active sessions for the current user
// @Tags users
// @Produce json
// @Success 200 {array} UserSession
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me/sessions [get]
func (h *Handler) GetSessions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	sessions, err := h.service.GetSessions(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get sessions")
		response.InternalServerError(c, "Failed to get sessions")
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// RevokeSession revokes a specific session
// @Summary Revoke session
// @Description Revoke a specific session by ID
// @Tags users
// @Produce json
// @Param sessionId path string true "Session ID" format(uuid)
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /users/me/sessions/{sessionId} [delete]
func (h *Handler) RevokeSession(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	sessionIDStr := c.Param("sessionId")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid session ID")
		return
	}

	err = h.service.RevokeSession(c.Request.Context(), userID.(uuid.UUID), sessionID)
	if err != nil {
		if err.Error() == "session not found" {
			response.NotFound(c, "Session not found")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to revoke session")
		response.InternalServerError(c, "Failed to revoke session")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked successfully"})
}

// RevokeAllSessions revokes all sessions for the user
// @Summary Revoke all sessions
// @Description Revoke all active sessions for the current user
// @Tags users
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me/sessions [delete]
func (h *Handler) RevokeAllSessions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err := h.service.RevokeAllSessions(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to revoke all sessions")
		response.InternalServerError(c, "Failed to revoke all sessions")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All sessions revoked successfully"})
}

// GetStats retrieves the user's statistics
// @Summary Get user statistics
// @Description Get tournament and game statistics for the current user
// @Tags users
// @Produce json
// @Success 200 {object} UserStats
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get stats")
		response.InternalServerError(c, "Failed to get statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}
