package result

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles result HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new result handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// EnterResult enters a result for a game
// @Summary Enter game result
// @Description Enter the result for a game
// @Tags results
// @Accept json
// @Produce json
// @Param body body ResultEntry true "Result entry"
// @Success 200 {object} ResultEntryResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /results [post]
func (h *Handler) EnterResult(c *gin.Context) {
	var entry ResultEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if exists {
		entry.EnteredBy = userID.(uuid.UUID)
	}

	response, err := h.service.EnterResult(c.Request.Context(), &entry)
	if err != nil {
		if err.Error() == "game not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CorrectResult corrects an existing result
// @Summary Correct game result
// @Description Correct an existing game result
// @Tags results
// @Accept json
// @Produce json
// @Param body body ResultCorrection true "Result correction"
// @Success 200 {object} ResultEntryResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /results/correct [post]
func (h *Handler) CorrectResult(c *gin.Context) {
	var correction ResultCorrection
	if err := c.ShouldBindJSON(&correction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if exists {
		correction.CorrectedBy = userID.(uuid.UUID)
	}

	response, err := h.service.CorrectResult(c.Request.Context(), &correction)
	if err != nil {
		if err.Error() == "game not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteResult removes a result from a game
// @Summary Delete game result
// @Description Remove the result from a game
// @Tags results
// @Accept json
// @Produce json
// @Param id path string true "Game ID"
// @Param body body object{reason string} true "Reason for deletion"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /results/{id} [delete]
func (h *Handler) DeleteResult(c *gin.Context) {
	idStr := c.Param("id")
	gameID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	var body struct {
		Reason string `json:"reason" binding:"required,min=10"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	var deletedBy uuid.UUID
	userID, exists := c.Get("userID")
	if exists {
		deletedBy = userID.(uuid.UUID)
	}

	err = h.service.DeleteResult(c.Request.Context(), gameID, deletedBy, body.Reason)
	if err != nil {
		if err.Error() == "game not found" || err.Error() == "game has no result to delete" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "result deleted successfully"})
}

// GetGame retrieves a game with full details
// @Summary Get game details
// @Description Get a game with player information and result
// @Tags results
// @Produce json
// @Param id path string true "Game ID"
// @Success 200 {object} Game
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /games/{id} [get]
func (h *Handler) GetGame(c *gin.Context) {
	idStr := c.Param("id")
	gameID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	game, err := h.service.GetGame(c.Request.Context(), gameID)
	if err != nil {
		if err.Error() == "game not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, game)
}

// GetRoundGames retrieves all games for a round
// @Summary Get round games
// @Description Get all games for a specific round
// @Tags results
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {array} Game
// @Failure 400 {object} map[string]string
// @Router /tournaments/{id}/rounds/{round}/games [get]
func (h *Handler) GetRoundGames(c *gin.Context) {
	tournamentIDStr := c.Param("id")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	roundStr := c.Param("round")
	roundNumber, err := strconv.Atoi(roundStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
		return
	}

	games, err := h.service.GetRoundGames(c.Request.Context(), tournamentID, roundNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, games)
}

// GetRoundStatus retrieves the result status for a round
// @Summary Get round status
// @Description Get the status of results for a round
// @Tags results
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {object} RoundStatus
// @Failure 400 {object} map[string]string
// @Router /tournaments/{id}/rounds/{round}/status [get]
func (h *Handler) GetRoundStatus(c *gin.Context) {
	tournamentIDStr := c.Param("id")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	roundStr := c.Param("round")
	roundNumber, err := strconv.Atoi(roundStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
		return
	}

	status, err := h.service.GetRoundStatus(c.Request.Context(), tournamentID, roundNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetResultAuditLog retrieves the audit log for a game
// @Summary Get result audit log
// @Description Get the audit history for a game's results
// @Tags results
// @Produce json
// @Param id path string true "Game ID"
// @Success 200 {array} ResultAudit
// @Failure 400 {object} map[string]string
// @Router /games/{id}/audit [get]
func (h *Handler) GetResultAuditLog(c *gin.Context) {
	idStr := c.Param("id")
	gameID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	audits, err := h.service.GetResultAuditLog(c.Request.Context(), gameID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, audits)
}

// EnterBulkResults enters multiple results at once
// @Summary Enter bulk results
// @Description Enter results for multiple games at once
// @Tags results
// @Accept json
// @Produce json
// @Param body body BulkResultEntry true "Bulk result entry"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /results/bulk [post]
func (h *Handler) EnterBulkResults(c *gin.Context) {
	var bulk BulkResultEntry
	if err := c.ShouldBindJSON(&bulk); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if exists {
		for i := range bulk.Results {
			bulk.Results[i].EnteredBy = userID.(uuid.UUID)
		}
	}

	responses, errors := h.service.EnterBulkResults(c.Request.Context(), &bulk)

	// Build response with both successes and failures
	errorStrings := make([]string, len(errors))
	for i, err := range errors {
		errorStrings[i] = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"successful": responses,
		"errors":     errorStrings,
		"totalCount": len(bulk.Results),
		"successCount": len(responses),
		"errorCount":   len(errors),
	})
}

// ValidateResult validates a result entry
// @Summary Validate result
// @Description Validate a result entry without saving
// @Tags results
// @Accept json
// @Produce json
// @Param body body ResultEntry true "Result entry"
// @Success 200 {object} ResultValidation
// @Failure 400 {object} map[string]string
// @Router /results/validate [post]
func (h *Handler) ValidateResult(c *gin.Context) {
	var entry ResultEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validation := h.service.ValidateResult(c.Request.Context(), &entry)
	c.JSON(http.StatusOK, validation)
}
