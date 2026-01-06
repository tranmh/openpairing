package arbiter

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/tournament"
	"github.com/rs/zerolog"
)

// Handler handles arbiter HTTP requests
type Handler struct {
	service           *Service
	tournamentService *tournament.Service
	monitorHub        *MonitorHub
	logger            zerolog.Logger
}

// NewHandler creates a new arbiter handler
func NewHandler(service *Service, tournamentService *tournament.Service, logger zerolog.Logger) *Handler {
	hub := NewMonitorHub(service, logger)
	return &Handler{
		service:           service,
		tournamentService: tournamentService,
		monitorHub:        hub,
		logger:            logger.With().Str("handler", "arbiter").Logger(),
	}
}

// GetMonitorHub returns the monitor hub for external use
func (h *Handler) GetMonitorHub() *MonitorHub {
	return h.monitorHub
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

		// Live Monitoring WebSocket
		arbiter.GET("/tournaments/:id/monitor", h.monitorHub.HandleWebSocket)

		// Player Management
		arbiter.POST("/tournaments/:id/withdraw-player", h.WithdrawPlayer)
		arbiter.POST("/tournaments/:id/reactivate-player", h.ReactivatePlayer)
		arbiter.POST("/tournaments/:id/assign-bye", h.AssignBye)

		// Result Management
		arbiter.POST("/tournaments/:id/correct-result", h.CorrectResult)

		// Pairing Management
		arbiter.POST("/tournaments/:id/swap-pairing", h.SwapPairing)
		arbiter.POST("/tournaments/:id/change-color", h.ChangeColor)
		arbiter.POST("/tournaments/:id/create-pairing", h.CreateManualPairing)
		arbiter.DELETE("/tournaments/:id/pairings/:gameId", h.DeletePairing)

		// Round Control
		arbiter.POST("/tournaments/:id/rounds/:round/pair", h.GeneratePairings)
		arbiter.POST("/tournaments/:id/rounds/:round/start", h.StartRound)
		arbiter.POST("/tournaments/:id/rounds/:round/complete", h.CompleteRound)

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

	userIDStr, _ := c.Get("user_id")
	arbiterID, _ := uuid.Parse(userIDStr.(string))
	userRole, _ := c.Get("user_role")
	role, _ := userRole.(string)

	status, err := h.service.GetTournamentStatusWithRole(c.Request.Context(), id, arbiterID, role)
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

// ReactivatePlayer reactivates a withdrawn player
// @Summary Reactivate player
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body ReactivatePlayerRequest true "Reactivation data"
// @Success 200
// @Router /arbiter/tournaments/{id}/reactivate-player [post]
func (h *Handler) ReactivatePlayer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req ReactivatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TournamentID = id

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	if err := h.service.ReactivatePlayer(c.Request.Context(), &req, arbiterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "player reactivated"})
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

// DeletePairing deletes a pairing from a round
// @Summary Delete pairing
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param gameId path string true "Game ID"
// @Param request body DeletePairingRequest true "Delete data"
// @Success 200
// @Router /arbiter/tournaments/{id}/pairings/{gameId} [delete]
func (h *Handler) DeletePairing(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	gameID, err := uuid.Parse(c.Param("gameId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	var req DeletePairingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body with just reason in query param
		req.Reason = c.Query("reason")
	}
	req.TournamentID = tournamentID
	req.GameID = gameID

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	if err := h.service.DeletePairing(c.Request.Context(), &req, arbiterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pairing deleted"})
}

// CreateManualPairing creates a manual pairing in a round
// @Summary Create manual pairing
// @Tags arbiter
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body CreateManualPairingRequest true "Pairing data"
// @Success 201 {object} CreateManualPairingResponse
// @Router /arbiter/tournaments/{id}/create-pairing [post]
func (h *Handler) CreateManualPairing(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req CreateManualPairingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TournamentID = tournamentID

	userID, _ := c.Get("user_id")
	arbiterID, _ := userID.(uuid.UUID)

	response, err := h.service.CreateManualPairing(c.Request.Context(), &req, arbiterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
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

// GeneratePairings generates pairings for a round
// @Summary Generate pairings
// @Tags arbiter
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {object} PairingsResponse
// @Router /arbiter/tournaments/{id}/rounds/{round}/pair [post]
func (h *Handler) GeneratePairings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	round, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	arbiterID, _ := uuid.Parse(userIDStr.(string))

	// Verify arbiter has permission
	if err := h.service.canManageTournament(c.Request.Context(), id, arbiterID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	games, err := h.tournamentService.GeneratePairings(c.Request.Context(), id, round, arbiterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, PairingsResponse{Games: games})
}

// StartRound starts a round
// @Summary Start round
// @Tags arbiter
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {object} RoundResponse
// @Router /arbiter/tournaments/{id}/rounds/{round}/start [post]
func (h *Handler) StartRound(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	round, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	arbiterID, _ := uuid.Parse(userIDStr.(string))

	// Verify arbiter has permission
	if err := h.service.canManageTournament(c.Request.Context(), id, arbiterID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	roundData, err := h.tournamentService.StartRound(c.Request.Context(), id, round)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RoundResponse{Round: roundData})
}

// CompleteRound completes a round
// @Summary Complete round
// @Tags arbiter
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {object} RoundResponse
// @Router /arbiter/tournaments/{id}/rounds/{round}/complete [post]
func (h *Handler) CompleteRound(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	round, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	arbiterID, _ := uuid.Parse(userIDStr.(string))

	// Verify arbiter has permission
	if err := h.service.canManageTournament(c.Request.Context(), id, arbiterID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	roundData, err := h.tournamentService.CompleteRound(c.Request.Context(), id, round)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RoundResponse{Round: roundData})
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

// PairingsResponse wraps pairings
type PairingsResponse struct {
	Games []tournament.Game `json:"games"`
}

// RoundResponse wraps a round
type RoundResponse struct {
	Round *tournament.Round `json:"round"`
}
