package analytics

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tranmh/openpairing/pkg/response"
	"github.com/rs/zerolog"
)

// Handler handles analytics-related HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new analytics handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("component", "analytics_handler").Logger(),
	}
}

// RegisterRoutes registers the analytics routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	analytics := r.Group("/analytics")
	{
		analytics.GET("/dashboard", h.GetDashboardStats)
		analytics.GET("/tournaments/:id", h.GetTournamentStats)
		analytics.GET("/players/:id", h.GetPlayerStats)
		analytics.GET("/compare/players", h.ComparePlayers)
		analytics.GET("/leaderboard", h.GetLeaderboard)
	}
}

// GetDashboardStats returns overall platform statistics
// @Summary Get dashboard statistics
// @Tags Analytics
// @Success 200 {object} DashboardStats
// @Router /analytics/dashboard [get]
func (h *Handler) GetDashboardStats(c *gin.Context) {
	stats, err := h.service.GetDashboardStats(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get dashboard stats")
		response.InternalServerError(c, "Failed to get statistics")
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// GetTournamentStats returns statistics for a specific tournament
// @Summary Get tournament statistics
// @Tags Analytics
// @Param id path string true "Tournament ID"
// @Success 200 {object} TournamentStats
// @Router /analytics/tournaments/{id} [get]
func (h *Handler) GetTournamentStats(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	stats, err := h.service.GetTournamentStats(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to get tournament stats")
		response.InternalServerError(c, "Failed to get statistics")
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// GetPlayerStats returns statistics for a specific player
// @Summary Get player statistics
// @Tags Analytics
// @Param id path string true "Player ID"
// @Success 200 {object} PlayerStats
// @Router /analytics/players/{id} [get]
func (h *Handler) GetPlayerStats(c *gin.Context) {
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid player ID")
		return
	}

	stats, err := h.service.GetPlayerStats(c.Request.Context(), playerID)
	if err != nil {
		h.logger.Error().Err(err).Str("playerId", playerID.String()).Msg("Failed to get player stats")
		response.InternalServerError(c, "Failed to get statistics")
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// ComparePlayers compares statistics between multiple players
// @Summary Compare player statistics
// @Tags Analytics
// @Param ids query []string true "Player IDs to compare"
// @Success 200 {object} map[string]PlayerStats
// @Router /analytics/compare/players [get]
func (h *Handler) ComparePlayers(c *gin.Context) {
	idStrings := c.QueryArray("ids")
	if len(idStrings) < 2 {
		response.BadRequest(c, "At least 2 player IDs are required")
		return
	}
	if len(idStrings) > 5 {
		response.BadRequest(c, "Maximum 5 players can be compared")
		return
	}

	comparison := make(map[string]*PlayerStats)
	for _, idStr := range idStrings {
		playerID, err := uuid.Parse(idStr)
		if err != nil {
			response.BadRequest(c, "Invalid player ID: "+idStr)
			return
		}

		stats, err := h.service.GetPlayerStats(c.Request.Context(), playerID)
		if err != nil {
			h.logger.Warn().Err(err).Str("playerId", playerID.String()).Msg("Failed to get player stats for comparison")
			continue
		}
		comparison[playerID.String()] = stats
	}

	if len(comparison) < 2 {
		response.BadRequest(c, "Could not find stats for enough players")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"players": comparison,
		"count":   len(comparison),
	})
}

// LeaderboardRequest contains filters for the leaderboard
type LeaderboardRequest struct {
	RatingType  string `form:"ratingType"`  // fide, dwz, oesb, ssb
	Category    string `form:"category"`    // open, women, junior, senior
	Federation  string `form:"federation"`  // Filter by federation
	ClubID      string `form:"clubId"`      // Filter by club
	Limit       int    `form:"limit"`       // Number of results (default 100)
	Offset      int    `form:"offset"`      // Offset for pagination
}

// GetLeaderboard returns a rating leaderboard
// @Summary Get player leaderboard
// @Tags Analytics
// @Param ratingType query string false "Rating type (fide, dwz)"
// @Param category query string false "Category (open, women, junior, senior)"
// @Param federation query string false "Federation filter"
// @Param clubId query string false "Club ID filter"
// @Param limit query int false "Limit (default 100)"
// @Param offset query int false "Offset"
// @Success 200 {array} PlayerSummary
// @Router /analytics/leaderboard [get]
func (h *Handler) GetLeaderboard(c *gin.Context) {
	var req LeaderboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid parameters")
		return
	}

	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 100
	}

	// Build query based on filters
	ctx := c.Request.Context()

	// For now, use the dashboard top players as a simple leaderboard
	// In production, this would use a more sophisticated query
	stats, err := h.service.GetDashboardStats(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get leaderboard")
		response.InternalServerError(c, "Failed to get leaderboard")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"players": stats.TopPlayers,
		"total":   len(stats.TopPlayers),
		"limit":   req.Limit,
		"offset":  req.Offset,
	})
}
