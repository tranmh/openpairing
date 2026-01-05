package waitlist

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpairing/openpairing/pkg/response"
)

// Handler handles waitlist HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new waitlist handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "waitlist").Logger(),
	}
}

// RegisterRoutes registers waitlist routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware, organizerMiddleware gin.HandlerFunc) {
	waitlist := r.Group("/tournaments/:id/waitlist")
	{
		// Public route - view waitlist
		waitlist.GET("", h.GetWaitlist)

		// Authenticated routes
		protected := waitlist.Group("")
		protected.Use(authMiddleware)
		{
			// Players can add themselves or check position
			protected.POST("", h.AddToWaitlist)
			protected.GET("/position", h.GetPosition)
		}

		// Organizer routes
		organizer := waitlist.Group("")
		organizer.Use(authMiddleware, organizerMiddleware)
		{
			organizer.DELETE("/:entryId", h.RemoveFromWaitlist)
			organizer.POST("/promote", h.PromoteNext)
		}
	}
}

// GetWaitlist godoc
// @Summary Get tournament waitlist
// @Tags waitlist
// @Produce json
// @Param tournamentId path string true "Tournament ID"
// @Success 200 {object} WaitlistListResponse
// @Router /tournaments/{tournamentId}/waitlist [get]
func (h *Handler) GetWaitlist(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	result, err := h.service.GetWaitlist(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournament_id", tournamentID.String()).Msg("Failed to get waitlist")
		response.InternalServerError(c, "Failed to get waitlist")
		return
	}

	response.Success(c, http.StatusOK, result)
}

// AddToWaitlist godoc
// @Summary Add to waitlist
// @Tags waitlist
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param tournamentId path string true "Tournament ID"
// @Param request body AddToWaitlistRequest true "Waitlist entry data"
// @Success 201 {object} WaitlistEntry
// @Failure 400 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /tournaments/{tournamentId}/waitlist [post]
func (h *Handler) AddToWaitlist(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	var req AddToWaitlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	entry, err := h.service.AddToWaitlist(c.Request.Context(), tournamentID, &req)
	if err != nil {
		switch err {
		case ErrAlreadyOnWaitlist:
			response.Conflict(c, "Player is already on the waitlist")
		case ErrAlreadyRegistered:
			response.Conflict(c, "Player is already registered for this tournament")
		default:
			h.logger.Error().Err(err).Msg("Failed to add to waitlist")
			response.InternalServerError(c, "Failed to add to waitlist")
		}
		return
	}

	response.Created(c, entry)
}

// GetPosition godoc
// @Summary Get player's position on waitlist
// @Tags waitlist
// @Security BearerAuth
// @Produce json
// @Param tournamentId path string true "Tournament ID"
// @Param playerId query string true "Player ID"
// @Success 200 {object} WaitlistEntry
// @Failure 404 {object} response.Error
// @Router /tournaments/{tournamentId}/waitlist/position [get]
func (h *Handler) GetPosition(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	playerIDStr := c.Query("playerId")
	if playerIDStr == "" {
		response.BadRequest(c, "Player ID is required")
		return
	}

	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid player ID")
		return
	}

	entry, err := h.service.GetPlayerPosition(c.Request.Context(), tournamentID, playerID)
	if err != nil {
		if err == ErrEntryNotFound {
			response.NotFound(c, "Player is not on the waitlist")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get waitlist position")
		response.InternalServerError(c, "Failed to get waitlist position")
		return
	}

	response.Success(c, http.StatusOK, entry)
}

// RemoveFromWaitlist godoc
// @Summary Remove from waitlist
// @Tags waitlist
// @Security BearerAuth
// @Param tournamentId path string true "Tournament ID"
// @Param entryId path string true "Waitlist entry ID"
// @Success 204
// @Failure 404 {object} response.Error
// @Router /tournaments/{tournamentId}/waitlist/{entryId} [delete]
func (h *Handler) RemoveFromWaitlist(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("entryId"))
	if err != nil {
		response.BadRequest(c, "Invalid entry ID")
		return
	}

	if err := h.service.RemoveFromWaitlist(c.Request.Context(), entryID); err != nil {
		if err == ErrEntryNotFound {
			response.NotFound(c, "Waitlist entry not found")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to remove from waitlist")
		response.InternalServerError(c, "Failed to remove from waitlist")
		return
	}

	response.NoContent(c)
}

// PromoteNext godoc
// @Summary Promote next player from waitlist
// @Tags waitlist
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param tournamentId path string true "Tournament ID"
// @Param request body PromoteFromWaitlistRequest true "Promotion options"
// @Success 200 {object} WaitlistEntry
// @Router /tournaments/{tournamentId}/waitlist/promote [post]
func (h *Handler) PromoteNext(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	var req PromoteFromWaitlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		req = PromoteFromWaitlistRequest{
			NotifyPlayer: true,
			ExpiryHours:  48,
		}
	}

	entry, err := h.service.PromoteNextPlayer(c.Request.Context(), tournamentID, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to promote player")
		response.InternalServerError(c, "Failed to promote player")
		return
	}

	if entry == nil {
		response.Success(c, http.StatusOK, gin.H{"message": "No players in queue"})
		return
	}

	response.Success(c, http.StatusOK, entry)
}
