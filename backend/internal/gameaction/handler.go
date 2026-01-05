package gameaction

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpairing/openpairing/pkg/response"
)

// Handler handles game action HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new game action handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "gameaction").Logger(),
	}
}

// Resign handles a player resigning from a game
// @Summary Resign from game
// @Description Resign from an ongoing chess game
// @Tags games
// @Accept json
// @Produce json
// @Param id path string true "Game ID" format(uuid)
// @Success 200 {object} GameActionResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /games/{id}/resign [post]
func (h *Handler) Resign(c *gin.Context) {
	gameIDStr := c.Param("id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid game ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.service.Resign(c.Request.Context(), gameID, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "game not found" {
			response.NotFound(c, "Game not found")
			return
		}
		if err.Error() == "you are not a player in this game" ||
			err.Error() == "game is not ongoing" {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Msg("Failed to process resignation")
		response.InternalServerError(c, "Failed to process resignation")
		return
	}

	c.JSON(http.StatusOK, result)
}

// DrawOffer handles draw offers and responses
// @Summary Offer or respond to draw
// @Description Offer a draw or accept/decline a draw offer
// @Tags games
// @Accept json
// @Produce json
// @Param id path string true "Game ID" format(uuid)
// @Param body body DrawOfferRequest false "Draw offer response"
// @Success 200 {object} GameActionResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /games/{id}/draw-offer [post]
func (h *Handler) DrawOffer(c *gin.Context) {
	gameIDStr := c.Param("id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid game ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Check for accept/decline in body (optional)
	var req DrawOfferRequest
	_ = c.ShouldBindJSON(&req) // Ignore errors - body is optional

	result, err := h.service.OfferDraw(c.Request.Context(), gameID, userID.(uuid.UUID), req.Accept)
	if err != nil {
		if err.Error() == "game not found" {
			response.NotFound(c, "Game not found")
			return
		}
		if err.Error() == "you are not a player in this game" ||
			err.Error() == "game is not ongoing" ||
			err.Error() == "no draw offer to respond to" ||
			err.Error() == "you cannot respond to your own draw offer" ||
			err.Error() == "you already have a pending draw offer" {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Msg("Failed to process draw offer")
		response.InternalServerError(c, "Failed to process draw offer")
		return
	}

	c.JSON(http.StatusOK, result)
}

// Abort handles aborting a game
// @Summary Abort game
// @Description Abort a chess game (only allowed in early moves)
// @Tags games
// @Accept json
// @Produce json
// @Param id path string true "Game ID" format(uuid)
// @Param body body AbortRequest false "Abort reason"
// @Success 200 {object} GameActionResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /games/{id}/abort [post]
func (h *Handler) Abort(c *gin.Context) {
	gameIDStr := c.Param("id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid game ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req AbortRequest
	_ = c.ShouldBindJSON(&req) // Body is optional

	result, err := h.service.AbortGame(c.Request.Context(), gameID, userID.(uuid.UUID), req.Reason)
	if err != nil {
		if err.Error() == "game not found" {
			response.NotFound(c, "Game not found")
			return
		}
		if err.Error() == "you are not a player in this game" ||
			err.Error() == "game is already finished" ||
			err.Error() == "game has progressed too far to abort, you may resign instead" {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Msg("Failed to abort game")
		response.InternalServerError(c, "Failed to abort game")
		return
	}

	c.JSON(http.StatusOK, result)
}

// CallArbiter handles requests for arbiter intervention
// @Summary Call arbiter
// @Description Request arbiter intervention for an ongoing game
// @Tags games
// @Accept json
// @Produce json
// @Param id path string true "Game ID" format(uuid)
// @Param body body CallArbiterRequest true "Arbiter request"
// @Success 200 {object} GameActionResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /games/{id}/call-arbiter [post]
func (h *Handler) CallArbiter(c *gin.Context) {
	gameIDStr := c.Param("id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid game ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CallArbiterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: reason is required (10-500 characters)")
		return
	}

	result, err := h.service.CallArbiter(c.Request.Context(), gameID, userID.(uuid.UUID), req.Reason)
	if err != nil {
		if err.Error() == "game not found" {
			response.NotFound(c, "Game not found")
			return
		}
		if err.Error() == "you are not a player in this game" ||
			err.Error() == "cannot call arbiter for a finished game" ||
			err.Error() == "there is already a pending arbiter request for this game" {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Msg("Failed to call arbiter")
		response.InternalServerError(c, "Failed to submit arbiter request")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDrawOfferStatus checks if there's a pending draw offer
// @Summary Get draw offer status
// @Description Check if there's a pending draw offer for a game
// @Tags games
// @Produce json
// @Param id path string true "Game ID" format(uuid)
// @Success 200 {object} DrawOffer
// @Failure 400 {object} response.ErrorResponse
// @Router /games/{id}/draw-offer [get]
func (h *Handler) GetDrawOfferStatus(c *gin.Context) {
	gameIDStr := c.Param("id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid game ID")
		return
	}

	offer := h.service.GetDrawOffer(c.Request.Context(), gameID)
	if offer == nil {
		c.JSON(http.StatusOK, gin.H{"pending": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending":   true,
		"offeredBy": offer.OfferedBy,
		"expiresAt": offer.ExpiresAt,
	})
}
