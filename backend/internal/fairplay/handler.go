package fairplay

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpairing/openpairing/pkg/response"
)

// Handler handles fairplay HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new fairplay handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "fairplay").Logger(),
	}
}

// RegisterRoutes registers the fairplay routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc, arbiterMiddleware gin.HandlerFunc) {
	fp := r.Group("/fairplay")
	fp.Use(authMiddleware)
	{
		// Violation routes
		fp.POST("/violations", h.ReportViolation)
		fp.GET("/violations", arbiterMiddleware, h.ListViolations)
		fp.GET("/violations/:id", arbiterMiddleware, h.GetViolation)
		fp.PUT("/violations/:id/review", arbiterMiddleware, h.ReviewViolation)

		// Player flags (arbiter only)
		fp.POST("/flags", arbiterMiddleware, h.FlagPlayer)
		fp.GET("/players/:playerId/flags", arbiterMiddleware, h.GetPlayerFlags)
		fp.DELETE("/flags/:flagId", arbiterMiddleware, h.RemoveFlag)

		// Player history and eligibility
		fp.GET("/players/:playerId/history", arbiterMiddleware, h.GetPlayerHistory)
		fp.GET("/players/:playerId/eligibility", h.CheckPlayerEligibility)

		// Statistics
		fp.GET("/stats", arbiterMiddleware, h.GetStats)
	}
}

// ReportViolation creates a new violation report
// @Summary Report fair play violation
// @Description Report a potential fair play violation
// @Tags fairplay
// @Accept json
// @Produce json
// @Param body body CreateViolationRequest true "Violation report"
// @Success 201 {object} FairplayViolation
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /fairplay/violations [post]
func (h *Handler) ReportViolation(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CreateViolationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	violation, err := h.service.ReportViolation(c.Request.Context(), &req, userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to report violation")
		response.InternalServerError(c, "Failed to report violation")
		return
	}

	c.JSON(http.StatusCreated, violation)
}

// GetViolation retrieves a violation by ID
// @Summary Get violation
// @Description Get a violation report by ID
// @Tags fairplay
// @Produce json
// @Param id path string true "Violation ID" format(uuid)
// @Success 200 {object} FairplayViolation
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /fairplay/violations/{id} [get]
func (h *Handler) GetViolation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid violation ID")
		return
	}

	violation, err := h.service.GetViolation(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "violation not found" {
			response.NotFound(c, "Violation not found")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get violation")
		response.InternalServerError(c, "Failed to get violation")
		return
	}

	c.JSON(http.StatusOK, violation)
}

// ReviewViolation reviews a violation report
// @Summary Review violation
// @Description Review and resolve a violation report
// @Tags fairplay
// @Accept json
// @Produce json
// @Param id path string true "Violation ID" format(uuid)
// @Param body body ReviewViolationRequest true "Review data"
// @Success 200 {object} FairplayViolation
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /fairplay/violations/{id}/review [put]
func (h *Handler) ReviewViolation(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid violation ID")
		return
	}

	var req ReviewViolationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	violation, err := h.service.ReviewViolation(c.Request.Context(), id, &req, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "violation not found" {
			response.NotFound(c, "Violation not found")
			return
		}
		if err.Error() == "violation has already been reviewed" {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Msg("Failed to review violation")
		response.InternalServerError(c, "Failed to review violation")
		return
	}

	c.JSON(http.StatusOK, violation)
}

// ListViolations lists violations with optional filtering
// @Summary List violations
// @Description List fair play violations with optional filtering
// @Tags fairplay
// @Produce json
// @Param tournamentId query string false "Filter by tournament ID"
// @Param playerId query string false "Filter by player ID"
// @Param status query string false "Filter by status"
// @Param type query string false "Filter by type"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(20)
// @Success 200 {object} ViolationListResponse
// @Failure 400 {object} response.ErrorResponse
// @Router /fairplay/violations [get]
func (h *Handler) ListViolations(c *gin.Context) {
	var req ViolationListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	result, err := h.service.ListViolations(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list violations")
		response.InternalServerError(c, "Failed to list violations")
		return
	}

	c.JSON(http.StatusOK, result)
}

// FlagPlayer flags a player
// @Summary Flag player
// @Description Create a flag on a player for monitoring
// @Tags fairplay
// @Accept json
// @Produce json
// @Param body body CreateFlagRequest true "Flag data"
// @Success 201 {object} PlayerFlag
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /fairplay/flags [post]
func (h *Handler) FlagPlayer(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CreateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	flag, err := h.service.FlagPlayer(c.Request.Context(), &req, userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to flag player")
		response.InternalServerError(c, "Failed to flag player")
		return
	}

	c.JSON(http.StatusCreated, flag)
}

// GetPlayerFlags retrieves flags for a player
// @Summary Get player flags
// @Description Get active flags for a player
// @Tags fairplay
// @Produce json
// @Param playerId path string true "Player ID" format(uuid)
// @Success 200 {object} FlagListResponse
// @Failure 400 {object} response.ErrorResponse
// @Router /fairplay/players/{playerId}/flags [get]
func (h *Handler) GetPlayerFlags(c *gin.Context) {
	playerIDStr := c.Param("playerId")
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid player ID")
		return
	}

	result, err := h.service.GetPlayerFlags(c.Request.Context(), playerID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get player flags")
		response.InternalServerError(c, "Failed to get player flags")
		return
	}

	c.JSON(http.StatusOK, result)
}

// RemoveFlag removes a player flag
// @Summary Remove flag
// @Description Remove a flag from a player
// @Tags fairplay
// @Produce json
// @Param flagId path string true "Flag ID" format(uuid)
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /fairplay/flags/{flagId} [delete]
func (h *Handler) RemoveFlag(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	flagIDStr := c.Param("flagId")
	flagID, err := uuid.Parse(flagIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid flag ID")
		return
	}

	err = h.service.RemoveFlag(c.Request.Context(), flagID, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "flag not found" {
			response.NotFound(c, "Flag not found")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to remove flag")
		response.InternalServerError(c, "Failed to remove flag")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flag removed successfully"})
}

// GetPlayerHistory retrieves a player's violation history
// @Summary Get player violation history
// @Description Get the fair play violation history for a player
// @Tags fairplay
// @Produce json
// @Param playerId path string true "Player ID" format(uuid)
// @Success 200 {array} FairplayViolation
// @Failure 400 {object} response.ErrorResponse
// @Router /fairplay/players/{playerId}/history [get]
func (h *Handler) GetPlayerHistory(c *gin.Context) {
	playerIDStr := c.Param("playerId")
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid player ID")
		return
	}

	violations, err := h.service.GetPlayerViolationHistory(c.Request.Context(), playerID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get player history")
		response.InternalServerError(c, "Failed to get player history")
		return
	}

	c.JSON(http.StatusOK, violations)
}

// CheckPlayerEligibility checks if a player is eligible to participate
// @Summary Check player eligibility
// @Description Check if a player is eligible to participate in tournaments
// @Tags fairplay
// @Produce json
// @Param playerId path string true "Player ID" format(uuid)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} response.ErrorResponse
// @Router /fairplay/players/{playerId}/eligibility [get]
func (h *Handler) CheckPlayerEligibility(c *gin.Context) {
	playerIDStr := c.Param("playerId")
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid player ID")
		return
	}

	eligible, reason, err := h.service.CheckPlayerEligibility(c.Request.Context(), playerID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to check eligibility")
		response.InternalServerError(c, "Failed to check eligibility")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"eligible": eligible,
		"reason":   reason,
	})
}

// GetStats retrieves fair play statistics
// @Summary Get fair play statistics
// @Description Get overall or tournament-specific fair play statistics
// @Tags fairplay
// @Produce json
// @Param tournamentId query string false "Tournament ID for specific stats"
// @Success 200 {object} FairplayStats
// @Failure 400 {object} response.ErrorResponse
// @Router /fairplay/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	var tournamentID *uuid.UUID
	if tidStr := c.Query("tournamentId"); tidStr != "" {
		tid, err := uuid.Parse(tidStr)
		if err != nil {
			response.BadRequest(c, "Invalid tournament ID")
			return
		}
		tournamentID = &tid
	}

	stats, err := h.service.GetStats(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get stats")
		response.InternalServerError(c, "Failed to get statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}
