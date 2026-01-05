package wam

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Handler handles WAM HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new WAM handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "wam").Logger(),
	}
}

// RegisterRoutes registers WAM routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	wam := r.Group("/tournaments/:id/wam")
	wam.Use(authMiddleware)
	{
		// Configuration
		wam.POST("/config", h.ConfigureWAM)
		wam.GET("/config", h.GetConfig)
		wam.GET("/status", h.GetStatus)

		// Groups
		wam.POST("/groups/assign", h.AssignGroups)
		wam.GET("/groups", h.ListGroups)
		wam.GET("/groups/:groupId", h.GetGroup)
		wam.POST("/groups/:groupId/members", h.AddGroupMember)
		wam.DELETE("/groups/:groupId/members/:participantId", h.RemoveGroupMember)
		wam.GET("/groups/:groupId/standings", h.GetGroupStandings)
		wam.POST("/groups/:groupId/pair", h.GenerateGroupPairings)

		// Player movement
		wam.PUT("/move-player", h.MovePlayer)
		wam.GET("/club-conflicts", h.GetClubConflicts)

		// Phase transitions
		wam.POST("/start-groups", h.StartGroupPhase)
		wam.POST("/groups/:groupId/complete-round", h.CompleteGroupRound)

		// Playoffs
		wam.POST("/playoffs/initialize", h.InitializePlayoffs)
		wam.GET("/playoffs/bracket", h.GetPlayoffBracket)

		// Qualifications
		wam.GET("/qualifications", h.GetQualifications)
		wam.GET("/qualifications/export", h.ExportQualifications)
	}
}

// ConfigureWAM configures WAM settings for a tournament
// @Summary Configure WAM tournament
// @Tags wam
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body ConfigureWAMRequest true "WAM configuration"
// @Success 200 {object} WAMConfig
// @Router /tournaments/{id}/wam/config [post]
func (h *Handler) ConfigureWAM(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req ConfigureWAMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.ConfigureWAM(c.Request.Context(), tournamentID, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to configure WAM")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetConfig gets WAM configuration
// @Summary Get WAM configuration
// @Tags wam
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} WAMConfig
// @Router /tournaments/{id}/wam/config [get]
func (h *Handler) GetConfig(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	config, err := h.service.GetConfig(c.Request.Context(), tournamentID)
	if err != nil {
		if err == ErrConfigNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "WAM not configured for this tournament"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetStatus gets overall WAM tournament status
// @Summary Get WAM status
// @Tags wam
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} WAMTournamentStatus
// @Router /tournaments/{id}/wam/status [get]
func (h *Handler) GetStatus(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	status, err := h.service.GetStatus(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// AssignGroups automatically assigns players to groups
// @Summary Assign players to groups
// @Tags wam
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body AssignGroupsRequest true "Assignment options"
// @Success 200 {object} AssignGroupsResponse
// @Router /tournaments/{id}/wam/groups/assign [post]
func (h *Handler) AssignGroups(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req AssignGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.AssignGroups(c.Request.Context(), tournamentID, &req)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrInvalidPhase {
			status = http.StatusConflict
		} else if err == ErrNoParticipants {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListGroups lists all groups for a tournament
// @Summary List groups
// @Tags wam
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {array} WAMGroup
// @Router /tournaments/{id}/wam/groups [get]
func (h *Handler) ListGroups(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	groups, err := h.service.ListGroups(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// GetGroup gets a single group with its members
// @Summary Get group
// @Tags wam
// @Produce json
// @Param id path string true "Tournament ID"
// @Param groupId path string true "Group ID"
// @Success 200 {object} WAMGroup
// @Router /tournaments/{id}/wam/groups/{groupId} [get]
func (h *Handler) GetGroup(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	group, err := h.service.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		if err == ErrGroupNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

// AddGroupMember adds a player to a group
// @Summary Add group member
// @Tags wam
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param groupId path string true "Group ID"
// @Param request body AddGroupMemberRequest true "Member data"
// @Success 200
// @Router /tournaments/{id}/wam/groups/{groupId}/members [post]
func (h *Handler) AddGroupMember(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req AddGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddPlayerToGroup(c.Request.Context(), groupID, &req); err != nil {
		status := http.StatusInternalServerError
		if err == ErrInvalidPhase {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member added"})
}

// RemoveGroupMember removes a player from a group
// @Summary Remove group member
// @Tags wam
// @Param id path string true "Tournament ID"
// @Param groupId path string true "Group ID"
// @Param participantId path string true "Participant ID"
// @Success 200
// @Router /tournaments/{id}/wam/groups/{groupId}/members/{participantId} [delete]
func (h *Handler) RemoveGroupMember(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	if err := h.service.RemovePlayerFromGroup(c.Request.Context(), groupID, participantID); err != nil {
		status := http.StatusInternalServerError
		if err == ErrMemberNotFound {
			status = http.StatusNotFound
		} else if err == ErrInvalidPhase {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}

// MovePlayer moves a player between groups
// @Summary Move player
// @Tags wam
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body MovePlayerRequest true "Move data"
// @Success 200
// @Router /tournaments/{id}/wam/move-player [put]
func (h *Handler) MovePlayer(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req MovePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.MovePlayer(c.Request.Context(), tournamentID, &req); err != nil {
		status := http.StatusInternalServerError
		if err == ErrPlayerNotInGroup {
			status = http.StatusNotFound
		} else if err == ErrInvalidPhase {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "player moved"})
}

// GetGroupStandings gets standings for a group
// @Summary Get group standings
// @Tags wam
// @Produce json
// @Param id path string true "Tournament ID"
// @Param groupId path string true "Group ID"
// @Success 200 {object} GroupStandingsResponse
// @Router /tournaments/{id}/wam/groups/{groupId}/standings [get]
func (h *Handler) GetGroupStandings(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	standings, err := h.service.GetGroupStandings(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, standings)
}

// GetClubConflicts gets all club conflicts
// @Summary Get club conflicts
// @Tags wam
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {array} WAMClubConflict
// @Router /tournaments/{id}/wam/club-conflicts [get]
func (h *Handler) GetClubConflicts(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	conflicts, err := h.service.GetClubConflicts(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conflicts)
}

// StartGroupPhase starts the group phase
// @Summary Start group phase
// @Tags wam
// @Param id path string true "Tournament ID"
// @Success 200
// @Router /tournaments/{id}/wam/start-groups [post]
func (h *Handler) StartGroupPhase(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	if err := h.service.StartGroupPhase(c.Request.Context(), tournamentID); err != nil {
		status := http.StatusInternalServerError
		if err == ErrInvalidPhase {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group phase started"})
}

// GenerateGroupPairings generates pairings for a group
// @Summary Generate group pairings
// @Tags wam
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param groupId path string true "Group ID"
// @Param request body GeneratePairingsRequest false "Pairing options"
// @Success 200 {array} engine.Pairing
// @Router /tournaments/{id}/wam/groups/{groupId}/pair [post]
func (h *Handler) GenerateGroupPairings(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req GeneratePairingsRequest
	_ = c.ShouldBindJSON(&req) // Optional

	pairings, err := h.service.GenerateGroupPairings(c.Request.Context(), groupID, req.RoundNumber)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrInvalidPhase {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pairings)
}

// CompleteGroupRound marks a group round as complete
// @Summary Complete group round
// @Tags wam
// @Param id path string true "Tournament ID"
// @Param groupId path string true "Group ID"
// @Success 200
// @Router /tournaments/{id}/wam/groups/{groupId}/complete-round [post]
func (h *Handler) CompleteGroupRound(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	if err := h.service.CompleteGroupRound(c.Request.Context(), groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "round completed"})
}

// InitializePlayoffs initializes the playoff bracket
// @Summary Initialize playoffs
// @Tags wam
// @Param id path string true "Tournament ID"
// @Success 200 {object} WAMPlayoffBracket
// @Router /tournaments/{id}/wam/playoffs/initialize [post]
func (h *Handler) InitializePlayoffs(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	bracket, err := h.service.InitializePlayoffs(c.Request.Context(), tournamentID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrPlayoffsNotEnabled || err == ErrGroupsNotComplete {
			status = http.StatusBadRequest
		} else if err == ErrInvalidPhase {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bracket)
}

// GetPlayoffBracket gets the playoff bracket
// @Summary Get playoff bracket
// @Tags wam
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} PlayoffBracketResponse
// @Router /tournaments/{id}/wam/playoffs/bracket [get]
func (h *Handler) GetPlayoffBracket(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	bracket, err := h.service.GetPlayoffBracket(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bracket)
}

// GetQualifications gets qualified players
// @Summary Get qualifications
// @Tags wam
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} QualificationsResponse
// @Router /tournaments/{id}/wam/qualifications [get]
func (h *Handler) GetQualifications(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	qualifications, err := h.service.GetQualifications(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, qualifications)
}

// ExportQualifications exports qualifications in various formats
// @Summary Export qualifications
// @Tags wam
// @Produce json,text/csv
// @Param id path string true "Tournament ID"
// @Param format query string false "Export format (csv, json)" default(json)
// @Success 200
// @Router /tournaments/{id}/wam/qualifications/export [get]
func (h *Handler) ExportQualifications(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	format := c.DefaultQuery("format", "json")

	qualifications, err := h.service.GetQualifications(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=qualifications_%s.csv", qualifications.Season))

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Header
		writer.Write([]string{"Player Name", "Group", "Rank", "Season", "Qualified At"})

		// Data
		for _, q := range qualifications.Qualifications {
			writer.Write([]string{
				q.PlayerName,
				q.GroupName,
				fmt.Sprintf("%d", q.GroupRank),
				q.Season,
				q.QualifiedAt.Format("2006-01-02"),
			})
		}

	default: // json
		data, err := json.MarshalIndent(qualifications, "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=qualifications_%s.json", qualifications.Season))
		c.Writer.Write(data)
	}
}
