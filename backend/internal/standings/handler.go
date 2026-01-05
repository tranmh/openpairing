package standings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles standings HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new standings handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetStandings returns tournament standings
// @Summary Get tournament standings
// @Description Get current standings for a tournament with all tiebreaks
// @Tags standings
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} StandingsResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/standings [get]
func (h *Handler) GetStandings(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	standings, err := h.service.GetStandings(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, standings)
}

// GetCrosstable returns tournament crosstable
// @Summary Get tournament crosstable
// @Description Get crosstable showing all games for a tournament
// @Tags standings
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} CrosstableResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/crosstable [get]
func (h *Handler) GetCrosstable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	crosstable, err := h.service.GetCrosstable(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, crosstable)
}

// RecalculateStandings recalculates and updates standings
// @Summary Recalculate standings
// @Description Recalculate all tiebreaks and update standings
// @Tags standings
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/standings/recalculate [post]
func (h *Handler) RecalculateStandings(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	err = h.service.RecalculateStandings(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "standings recalculated successfully"})
}

// GetPlayerProgress returns a player's tournament progress
// @Summary Get player progress
// @Description Get a player's progress through the tournament
// @Tags standings
// @Produce json
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Success 200 {object} PlayerProgress
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/participants/{participantId}/progress [get]
func (h *Handler) GetPlayerProgress(c *gin.Context) {
	tournamentIDStr := c.Param("id")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	participantIDStr := c.Param("participantId")
	participantID, err := uuid.Parse(participantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	progress, err := h.service.GetPlayerProgress(c.Request.Context(), tournamentID, participantID)
	if err != nil {
		if err.Error() == "participant not found" || err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetAvailableTiebreaks returns all available tiebreak systems
// @Summary Get available tiebreaks
// @Description Get list of all available tiebreak systems with descriptions
// @Tags standings
// @Produce json
// @Success 200 {array} TiebreakInfo
// @Router /tiebreaks [get]
func (h *Handler) GetAvailableTiebreaks(c *gin.Context) {
	tiebreaks := h.service.GetAvailableTiebreaks()
	c.JSON(http.StatusOK, tiebreaks)
}
