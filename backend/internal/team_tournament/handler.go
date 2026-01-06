package team_tournament

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tranmh/openpairing/pkg/response"
)

// Handler handles team tournament HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new team tournament handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "team_tournament").Logger(),
	}
}

// RegisterRoutes registers the team tournament routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc, organizerMiddleware gin.HandlerFunc) {
	// Team tournament settings
	r.GET("/tournaments/:id/team-settings", authMiddleware, h.GetSettings)
	r.PUT("/tournaments/:id/team-settings", authMiddleware, organizerMiddleware, h.UpdateSettings)

	// Team registration
	r.GET("/tournaments/:id/teams", h.ListTeamParticipants)
	r.POST("/tournaments/:id/teams", authMiddleware, h.RegisterTeam)
	r.DELETE("/tournaments/:id/teams/:teamId", authMiddleware, h.WithdrawTeam)

	// Team lineups
	r.GET("/tournaments/:id/rounds/:round/lineups", authMiddleware, h.ListLineups)
	r.GET("/tournaments/:id/rounds/:round/lineups/:teamId", authMiddleware, h.GetLineup)
	r.PUT("/tournaments/:id/rounds/:round/lineups/:teamId", authMiddleware, h.SubmitLineup)
	r.POST("/tournaments/:id/rounds/:round/lineups/:teamId/lock", authMiddleware, organizerMiddleware, h.LockLineup)

	// Team matches
	r.GET("/tournaments/:id/matches", h.ListMatches)
	r.GET("/tournaments/:id/rounds/:round/matches", h.ListMatchesByRound)
	r.GET("/tournaments/:id/matches/:matchId", h.GetMatch)
	r.GET("/tournaments/:id/matches/:matchId/boards", h.GetMatchBoards)
	r.PUT("/tournaments/:id/matches/:matchId/boards/:board", authMiddleware, h.EnterBoardResult)

	// Team standings
	r.GET("/tournaments/:id/team-standings", h.GetTeamStandings)
}

// GetSettings retrieves team tournament settings
// @Summary Get team tournament settings
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} TeamTournamentSettings
// @Router /tournaments/{id}/team-settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	settings, err := h.service.GetSettings(c.Request.Context(), tournamentID)
	if err != nil {
		response.InternalServerError(c, "Failed to get settings")
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates team tournament settings
// @Summary Update team tournament settings
// @Tags team-tournament
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param settings body TeamTournamentSettings true "Settings"
// @Success 200 {object} TeamTournamentSettings
// @Router /tournaments/{id}/team-settings [put]
func (h *Handler) UpdateSettings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	var settings TeamTournamentSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	settings.TournamentID = tournamentID

	if err := h.service.UpdateSettings(c.Request.Context(), &settings); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update settings")
		response.InternalServerError(c, "Failed to update settings")
		return
	}

	c.JSON(http.StatusOK, settings)
}

// ListTeamParticipants lists all teams registered for a tournament
// @Summary List tournament teams
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {array} TeamParticipant
// @Router /tournaments/{id}/teams [get]
func (h *Handler) ListTeamParticipants(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	participants, err := h.service.ListTeamParticipants(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list teams")
		response.InternalServerError(c, "Failed to list teams")
		return
	}

	c.JSON(http.StatusOK, participants)
}

// RegisterTeam registers a team for a tournament
// @Summary Register team for tournament
// @Tags team-tournament
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body RegisterTeamRequest true "Registration request"
// @Success 201 {object} TeamParticipant
// @Router /tournaments/{id}/teams [post]
func (h *Handler) RegisterTeam(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req RegisterTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	req.TournamentID = tournamentID

	participant, err := h.service.RegisterTeam(c.Request.Context(), &req, userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to register team")
		response.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, participant)
}

// WithdrawTeam withdraws a team from a tournament
// @Summary Withdraw team from tournament
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Param teamId path string true "Team Participant ID"
// @Success 200 {object} map[string]string
// @Router /tournaments/{id}/teams/{teamId} [delete]
func (h *Handler) WithdrawTeam(c *gin.Context) {
	participantID, err := uuid.Parse(c.Param("teamId"))
	if err != nil {
		response.BadRequest(c, "Invalid team participant ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.service.WithdrawTeam(c.Request.Context(), participantID, userID.(uuid.UUID)); err != nil {
		h.logger.Error().Err(err).Msg("Failed to withdraw team")
		response.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team withdrawn successfully"})
}

// ListLineups lists all lineups for a round
// @Summary List round lineups
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {array} TeamLineup
// @Router /tournaments/{id}/rounds/{round}/lineups [get]
func (h *Handler) ListLineups(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	roundNumber, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		response.BadRequest(c, "Invalid round number")
		return
	}

	// Get all team participants
	participants, err := h.service.ListTeamParticipants(c.Request.Context(), tournamentID)
	if err != nil {
		response.InternalServerError(c, "Failed to get participants")
		return
	}

	// Get lineup for each team
	lineups := make([]TeamLineup, 0)
	for _, p := range participants {
		lineup, err := h.service.GetLineup(c.Request.Context(), tournamentID, roundNumber, p.ID)
		if err == nil {
			lineups = append(lineups, *lineup)
		}
	}

	c.JSON(http.StatusOK, lineups)
}

// GetLineup retrieves a team's lineup for a round
// @Summary Get team lineup
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Param teamId path string true "Team Participant ID"
// @Success 200 {object} TeamLineup
// @Router /tournaments/{id}/rounds/{round}/lineups/{teamId} [get]
func (h *Handler) GetLineup(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	roundNumber, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		response.BadRequest(c, "Invalid round number")
		return
	}

	teamParticipantID, err := uuid.Parse(c.Param("teamId"))
	if err != nil {
		response.BadRequest(c, "Invalid team participant ID")
		return
	}

	lineup, err := h.service.GetLineup(c.Request.Context(), tournamentID, roundNumber, teamParticipantID)
	if err != nil {
		response.NotFound(c, "Lineup not found")
		return
	}

	c.JSON(http.StatusOK, lineup)
}

// SubmitLineup submits or updates a team's lineup
// @Summary Submit team lineup
// @Tags team-tournament
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Param teamId path string true "Team Participant ID"
// @Param lineup body SubmitLineupRequest true "Lineup"
// @Success 200 {object} TeamLineup
// @Router /tournaments/{id}/rounds/{round}/lineups/{teamId} [put]
func (h *Handler) SubmitLineup(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	roundNumber, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		response.BadRequest(c, "Invalid round number")
		return
	}

	teamParticipantID, err := uuid.Parse(c.Param("teamId"))
	if err != nil {
		response.BadRequest(c, "Invalid team participant ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req SubmitLineupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	req.TournamentID = tournamentID
	req.RoundNumber = roundNumber
	req.TeamParticipantID = teamParticipantID

	lineup, err := h.service.SubmitLineup(c.Request.Context(), &req, userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to submit lineup")
		response.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, lineup)
}

// LockLineup locks a team's lineup
// @Summary Lock team lineup
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Param teamId path string true "Team Participant ID"
// @Success 200 {object} map[string]string
// @Router /tournaments/{id}/rounds/{round}/lineups/{teamId}/lock [post]
func (h *Handler) LockLineup(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	roundNumber, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		response.BadRequest(c, "Invalid round number")
		return
	}

	teamParticipantID, err := uuid.Parse(c.Param("teamId"))
	if err != nil {
		response.BadRequest(c, "Invalid team participant ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.service.LockLineup(c.Request.Context(), tournamentID, roundNumber, teamParticipantID, userID.(uuid.UUID)); err != nil {
		h.logger.Error().Err(err).Msg("Failed to lock lineup")
		response.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lineup locked successfully"})
}

// ListMatches lists all team matches for a tournament
// @Summary List all tournament matches
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {array} TeamMatch
// @Router /tournaments/{id}/matches [get]
func (h *Handler) ListMatches(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	matches, err := h.service.ListTeamMatches(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list matches")
		response.InternalServerError(c, "Failed to list matches")
		return
	}

	c.JSON(http.StatusOK, matches)
}

// ListMatchesByRound lists team matches for a specific round
// @Summary List round matches
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {array} TeamMatch
// @Router /tournaments/{id}/rounds/{round}/matches [get]
func (h *Handler) ListMatchesByRound(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	roundNumber, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		response.BadRequest(c, "Invalid round number")
		return
	}

	matches, err := h.service.ListTeamMatchesByRound(c.Request.Context(), tournamentID, roundNumber)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list matches")
		response.InternalServerError(c, "Failed to list matches")
		return
	}

	c.JSON(http.StatusOK, matches)
}

// GetMatch retrieves a team match with boards
// @Summary Get team match
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Param matchId path string true "Match ID"
// @Success 200 {object} TeamMatchResponse
// @Router /tournaments/{id}/matches/{matchId} [get]
func (h *Handler) GetMatch(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("matchId"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	match, err := h.service.GetTeamMatch(c.Request.Context(), matchID)
	if err != nil {
		response.NotFound(c, "Match not found")
		return
	}

	c.JSON(http.StatusOK, match)
}

// GetMatchBoards retrieves boards for a team match
// @Summary Get match boards
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Param matchId path string true "Match ID"
// @Success 200 {array} TeamMatchBoard
// @Router /tournaments/{id}/matches/{matchId}/boards [get]
func (h *Handler) GetMatchBoards(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("matchId"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	match, err := h.service.GetTeamMatch(c.Request.Context(), matchID)
	if err != nil {
		response.NotFound(c, "Match not found")
		return
	}

	c.JSON(http.StatusOK, match.Boards)
}

// EnterBoardResult enters a board result
// @Summary Enter board result
// @Tags team-tournament
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param matchId path string true "Match ID"
// @Param board path int true "Board number"
// @Param request body EnterBoardResultRequest true "Result"
// @Success 200 {object} map[string]string
// @Router /tournaments/{id}/matches/{matchId}/boards/{board} [put]
func (h *Handler) EnterBoardResult(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("matchId"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	boardNumber, err := strconv.Atoi(c.Param("board"))
	if err != nil {
		response.BadRequest(c, "Invalid board number")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req EnterBoardResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	req.TeamMatchID = matchID
	req.BoardNumber = boardNumber

	if err := h.service.EnterBoardResult(c.Request.Context(), &req, userID.(uuid.UUID)); err != nil {
		h.logger.Error().Err(err).Msg("Failed to enter board result")
		response.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Board result entered"})
}

// GetTeamStandings retrieves team standings
// @Summary Get team standings
// @Tags team-tournament
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} TeamTournamentStandingsResponse
// @Router /tournaments/{id}/team-standings [get]
func (h *Handler) GetTeamStandings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	standings, err := h.service.GetTeamStandings(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get team standings")
		response.InternalServerError(c, "Failed to get standings")
		return
	}

	c.JSON(http.StatusOK, standings)
}
