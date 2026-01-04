package arbiter

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Handler handles arbiter HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new arbiter handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "arbiter").Logger(),
	}
}

// RegisterRoutes registers arbiter routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc, arbiterMiddleware gin.HandlerFunc) {
	arbiter := r.Group("/arbiter")
	arbiter.Use(authMiddleware)
	arbiter.Use(arbiterMiddleware)
	{
		// Console & Status
		arbiter.GET("/tournaments/:id/status", h.GetTournamentStatus)
		arbiter.GET("/tournaments/:id/audit-log", h.GetAuditLog)

		// Player Management
		arbiter.POST("/tournaments/:id/withdraw-player", h.WithdrawPlayer)
		arbiter.POST("/tournaments/:id/assign-bye", h.AssignBye)

		// Result Management
		arbiter.POST("/tournaments/:id/correct-result", h.CorrectResult)

		// Pairing Management
		arbiter.POST("/tournaments/:id/swap-pairing", h.SwapPairing)
		arbiter.POST("/tournaments/:id/change-color", h.ChangeColor)

		// Announcements
		arbiter.POST("/tournaments/:id/announcements", h.CreateAnnouncement)
	}
}

// GetTournamentStatus gets the tournament status for the arbiter console
// @Summary Get tournament status
// @Tags arbiter
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} TournamentStatusResponse
// @Router /arbiter/tournaments/{id}/status [get]
func (h *Handler) GetTournamentStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	status, err := h.service.GetTournamentStatus(c.Request.Context(), id, arbiterID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get tournament status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TournamentStatusResponse{Status: status})
}

// GetAuditLog gets the audit log for a tournament
// @Summary Get audit log
// @Tags arbiter
// @Produce json
// @Param id path string true "Tournament ID"
// @Param limit query int false "Limit results"
// @Success 200 {object} AuditLogResponse
// @Router /arbiter/tournaments/{id}/audit-log [get]
func (h *Handler) GetAuditLog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	log, err := h.service.GetAuditLog(c.Request.Context(), id, arbiterID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, AuditLogResponse{AuditLog: log})
}

// WithdrawPlayer withdraws a player from the tournament
// @Summary Withdraw player
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body WithdrawPlayerRequest true "Withdraw data"
// @Success 200
// @Router /arbiter/tournaments/{id}/withdraw-player [post]
func (h *Handler) WithdrawPlayer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req WithdrawPlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TournamentID = id

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	if err := h.service.WithdrawPlayer(c.Request.Context(), &req, arbiterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "player withdrawn"})
}

// AssignBye assigns a bye to a player
// @Summary Assign bye
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body AssignByeRequest true "Bye data"
// @Success 200
// @Router /arbiter/tournaments/{id}/assign-bye [post]
func (h *Handler) AssignBye(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req AssignByeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TournamentID = id

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	if err := h.service.AssignBye(c.Request.Context(), &req, arbiterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bye assigned"})
}

// CorrectResult corrects a game result
// @Summary Correct result
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body CorrectResultRequest true "Correction data"
// @Success 200
// @Router /arbiter/tournaments/{id}/correct-result [post]
func (h *Handler) CorrectResult(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req CorrectResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TournamentID = id

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	if err := h.service.CorrectResult(c.Request.Context(), &req, arbiterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "result corrected"})
}

// SwapPairing swaps two players between pairings
// @Summary Swap pairing
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body SwapPairingRequest true "Swap data"
// @Success 200
// @Router /arbiter/tournaments/{id}/swap-pairing [post]
func (h *Handler) SwapPairing(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req SwapPairingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TournamentID = id

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	if err := h.service.SwapPairing(c.Request.Context(), &req, arbiterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pairing swapped"})
}

// ChangeColor swaps the colors in a game
// @Summary Change color
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body ChangeColorRequest true "Change data"
// @Success 200
// @Router /arbiter/tournaments/{id}/change-color [post]
func (h *Handler) ChangeColor(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req ChangeColorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TournamentID = id

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	if err := h.service.ChangeColor(c.Request.Context(), &req, arbiterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "colors changed"})
}

// CreateAnnouncement creates a tournament announcement
// @Summary Create announcement
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body CreateAnnouncementRequest true "Announcement data"
// @Success 201 {object} AnnouncementResponse
// @Router /arbiter/tournaments/{id}/announcements [post]
func (h *Handler) CreateAnnouncement(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TournamentID = id

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	announcement, err := h.service.CreateAnnouncement(c.Request.Context(), &req, arbiterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, AnnouncementResponse{Announcement: announcement})
}

// Response types

// TournamentStatusResponse wraps tournament status
type TournamentStatusResponse struct {
	Status *TournamentStatus `json:"status"`
}

// AuditLogResponse wraps audit log
type AuditLogResponse struct {
	AuditLog []AuditAction `json:"auditLog"`
}

// AnnouncementResponse wraps an announcement
type AnnouncementResponse struct {
	Announcement *Announcement `json:"announcement"`
}
