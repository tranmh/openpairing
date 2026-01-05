package export

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles export HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new export handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ExportTournamentPGN exports a tournament in PGN format
// @Summary Export tournament games as PGN
// @Description Export all games from a tournament in PGN format
// @Tags export
// @Produce text/plain
// @Param id path string true "Tournament ID"
// @Success 200 {string} string "PGN file content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/export/pgn [get]
func (h *Handler) ExportTournamentPGN(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	export, err := h.service.ExportTournamentPGN(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, []byte(export.Content))
}

// ExportRoundPGN exports a specific round in PGN format
// @Summary Export round games as PGN
// @Description Export all games from a specific round in PGN format
// @Tags export
// @Produce text/plain
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {string} string "PGN file content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/rounds/{round}/export/pgn [get]
func (h *Handler) ExportRoundPGN(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	roundStr := c.Param("round")
	roundNum, err := strconv.Atoi(roundStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
		return
	}

	export, err := h.service.ExportRoundPGN(c.Request.Context(), id, roundNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, []byte(export.Content))
}

// ExportGamePGN exports a single game in PGN format
// @Summary Export single game as PGN
// @Description Export a single game in PGN format
// @Tags export
// @Produce text/plain
// @Param id path string true "Game ID"
// @Success 200 {string} string "PGN file content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /games/{id}/export/pgn [get]
func (h *Handler) ExportGamePGN(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	export, err := h.service.ExportGamePGN(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "game not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, []byte(export.Content))
}

// ExportTournamentJSON exports a tournament in JSON format
// @Summary Export tournament data as JSON
// @Description Export tournament data including games and participants in JSON format
// @Tags export
// @Produce application/json
// @Param id path string true "Tournament ID"
// @Success 200 {object} TournamentExport
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/export/json [get]
func (h *Handler) ExportTournamentJSON(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	export, err := h.service.ExportTournamentJSON(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, []byte(export.Content))
}

// ExportStandingsCSV exports tournament standings in CSV format
// @Summary Export standings as CSV
// @Description Export tournament standings in CSV format
// @Tags export
// @Produce text/csv
// @Param id path string true "Tournament ID"
// @Success 200 {string} string "CSV file content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/export/standings [get]
func (h *Handler) ExportStandingsCSV(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	export, err := h.service.ExportStandingsCSV(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, []byte(export.Content))
}

// ExportCrosstableCSV exports tournament crosstable in CSV format
// @Summary Export crosstable as CSV
// @Description Export tournament crosstable in CSV format
// @Tags export
// @Produce text/csv
// @Param id path string true "Tournament ID"
// @Success 200 {string} string "CSV file content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/export/crosstable [get]
func (h *Handler) ExportCrosstableCSV(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	export, err := h.service.ExportCrosstableCSV(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, []byte(export.Content))
}
