package export

import (
	"archive/zip"
	"bytes"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// createZipArchive creates a ZIP archive containing multiple PDF files
func createZipArchive(certificates []PDFExportResponse) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for _, cert := range certificates {
		f, err := w.Create(cert.Filename)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(cert.Content)
		if err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

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

// ExportStandingsPDF exports tournament standings in PDF format
// @Summary Export standings as PDF
// @Description Export tournament standings in PDF format
// @Tags export
// @Produce application/pdf
// @Param id path string true "Tournament ID"
// @Success 200 {file} file "PDF file"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/export/standings/pdf [get]
func (h *Handler) ExportStandingsPDF(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	export, err := h.service.ExportStandingsPDF(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, export.Content)
}

// ExportCrosstablePDF exports tournament crosstable in PDF format
// @Summary Export crosstable as PDF
// @Description Export tournament crosstable in PDF format
// @Tags export
// @Produce application/pdf
// @Param id path string true "Tournament ID"
// @Success 200 {file} file "PDF file"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/export/crosstable/pdf [get]
func (h *Handler) ExportCrosstablePDF(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	export, err := h.service.ExportCrosstablePDF(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, export.Content)
}

// ExportParticipantCertificate exports a certificate for a specific participant
// @Summary Export participant certificate as PDF
// @Description Generate a participation/achievement certificate for a tournament participant
// @Tags export
// @Produce application/pdf
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Success 200 {file} file "PDF certificate"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/participants/{participantId}/certificate [get]
func (h *Handler) ExportParticipantCertificate(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	export, err := h.service.GenerateCertificate(c.Request.Context(), tournamentID, participantID)
	if err != nil {
		if err.Error() == "participant not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+export.Filename)
	c.Data(http.StatusOK, export.MimeType, export.Content)
}

// ExportTopCertificates exports certificates for top N participants
// @Summary Export certificates for top players
// @Description Generate certificates for top N participants (default top 3)
// @Tags export
// @Produce application/zip
// @Param id path string true "Tournament ID"
// @Param top query int false "Number of top players" default(3)
// @Success 200 {file} file "ZIP file with certificates"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/certificates [get]
func (h *Handler) ExportTopCertificates(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	topN := 3
	if topStr := c.Query("top"); topStr != "" {
		if n, err := strconv.Atoi(topStr); err == nil && n > 0 {
			topN = n
		}
	}

	certificates, err := h.service.GenerateAllCertificates(c.Request.Context(), tournamentID, topN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(certificates) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no participants found"})
		return
	}

	// If only one certificate, return it directly
	if len(certificates) == 1 {
		c.Header("Content-Disposition", "attachment; filename="+certificates[0].Filename)
		c.Data(http.StatusOK, certificates[0].MimeType, certificates[0].Content)
		return
	}

	// For multiple certificates, create a ZIP archive
	zipContent, err := createZipArchive(certificates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create archive"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=certificates.zip")
	c.Data(http.StatusOK, "application/zip", zipContent)
}
