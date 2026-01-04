package team

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Handler handles team HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new team handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "team").Logger(),
	}
}

// RegisterRoutes registers team routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	teams := r.Group("/teams")
	teams.Use(authMiddleware)
	{
		teams.POST("", h.CreateTeam)
		teams.GET("/:id", h.GetTeam)
		teams.PUT("/:id", h.UpdateTeam)
		teams.DELETE("/:id", h.DeleteTeam)

		// Roster management
		teams.GET("/:id/members", h.GetRoster)
		teams.POST("/:id/members", h.AddRosterMember)
		teams.PUT("/:id/members/:playerId", h.UpdateRosterMember)
		teams.DELETE("/:id/members/:playerId", h.RemoveRosterMember)
	}

	// Club teams
	clubs := r.Group("/clubs")
	clubs.Use(authMiddleware)
	{
		clubs.GET("/:id/teams", h.GetClubTeams)
	}

	// Tournament team routes
	tournaments := r.Group("/tournaments")
	tournaments.Use(authMiddleware)
	{
		// Team registration
		tournaments.GET("/:id/teams", h.GetTournamentTeams)
		tournaments.POST("/:id/teams", h.RegisterTeam)
		tournaments.DELETE("/:id/teams/:teamParticipantId", h.WithdrawTeam)

		// Lineups
		tournaments.GET("/:id/rounds/:roundId/lineups", h.GetLineupsForRound)
		tournaments.GET("/:id/rounds/:roundId/lineups/:teamParticipantId", h.GetLineup)
		tournaments.PUT("/:id/rounds/:roundId/lineups/:teamParticipantId", h.SubmitLineup)

		// Team matches
		tournaments.GET("/:id/matches", h.GetTournamentMatches)
		tournaments.GET("/:id/rounds/:roundId/matches", h.GetRoundMatches)
		tournaments.GET("/:id/matches/:matchId", h.GetMatch)
		tournaments.GET("/:id/matches/:matchId/boards", h.GetMatchBoards)
		tournaments.PUT("/:id/matches/:matchId/boards/:boardId", h.EnterBoardResult)
		tournaments.POST("/:id/matches/:matchId/calculate", h.CalculateMatchResult)

		// Standings
		tournaments.GET("/:id/team-standings", h.GetTeamStandings)

		// Settings
		tournaments.GET("/:id/team-settings", h.GetTeamSettings)
		tournaments.PUT("/:id/team-settings", h.UpdateTeamSettings)
	}
}

// Team CRUD handlers

// CreateTeam creates a new team
func (h *Handler) CreateTeam(c *gin.Context) {
	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team, err := h.service.CreateTeam(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create team")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"team": team})
}

// GetTeam retrieves a team by ID
func (h *Handler) GetTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team ID"})
		return
	}

	team, err := h.service.GetTeam(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if team == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"team": team})
}

// UpdateTeam updates a team
func (h *Handler) UpdateTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team ID"})
		return
	}

	var req UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team, err := h.service.UpdateTeam(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"team": team})
}

// DeleteTeam deletes a team
func (h *Handler) DeleteTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team ID"})
		return
	}

	if err := h.service.DeleteTeam(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "team deleted"})
}

// Roster handlers

// GetRoster retrieves a team's roster
func (h *Handler) GetRoster(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team ID"})
		return
	}

	members, err := h.service.GetRoster(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddRosterMember adds a player to a team's roster
func (h *Handler) AddRosterMember(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team ID"})
		return
	}

	var req AddRosterMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := h.service.AddPlayerToRoster(c.Request.Context(), teamID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"member": member})
}

// UpdateRosterMember updates a roster member
func (h *Handler) UpdateRosterMember(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team ID"})
		return
	}

	var req UpdateRosterMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateRosterMember(c.Request.Context(), teamID, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "roster member updated"})
}

// RemoveRosterMember removes a player from a team's roster
func (h *Handler) RemoveRosterMember(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team ID"})
		return
	}

	playerID, err := uuid.Parse(c.Param("playerId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player ID"})
		return
	}

	if err := h.service.RemovePlayerFromRoster(c.Request.Context(), teamID, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "player removed from roster"})
}

// Club handlers

// GetClubTeams retrieves all teams for a club
func (h *Handler) GetClubTeams(c *gin.Context) {
	clubID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid club ID"})
		return
	}

	teams, err := h.service.GetTeamsByClub(c.Request.Context(), clubID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

// Tournament team handlers

// GetTournamentTeams retrieves all teams registered for a tournament
func (h *Handler) GetTournamentTeams(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	teams, err := h.service.GetTournamentTeams(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

// RegisterTeam registers a team for a tournament
func (h *Handler) RegisterTeam(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req struct {
		TeamID     uuid.UUID `json:"teamId" binding:"required"`
		SeedRating *int      `json:"seedRating"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	participant, err := h.service.RegisterTeamForTournament(c.Request.Context(), tournamentID, req.TeamID, req.SeedRating)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"participant": participant})
}

// WithdrawTeam withdraws a team from a tournament
func (h *Handler) WithdrawTeam(c *gin.Context) {
	participantID, err := uuid.Parse(c.Param("teamParticipantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team participant ID"})
		return
	}

	if err := h.service.WithdrawTeamFromTournament(c.Request.Context(), participantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "team withdrawn"})
}

// Lineup handlers

// GetLineupsForRound retrieves all lineups for a round
func (h *Handler) GetLineupsForRound(c *gin.Context) {
	roundID, err := uuid.Parse(c.Param("roundId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round ID"})
		return
	}

	lineups, err := h.service.GetLineupsForRound(c.Request.Context(), roundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"lineups": lineups})
}

// GetLineup retrieves a team's lineup for a round
func (h *Handler) GetLineup(c *gin.Context) {
	roundID, err := uuid.Parse(c.Param("roundId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round ID"})
		return
	}

	teamParticipantID, err := uuid.Parse(c.Param("teamParticipantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team participant ID"})
		return
	}

	lineup, err := h.service.GetLineup(c.Request.Context(), roundID, teamParticipantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if lineup == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lineup not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"lineup": lineup})
}

// SubmitLineup submits or updates a team's lineup
func (h *Handler) SubmitLineup(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	roundID, err := uuid.Parse(c.Param("roundId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round ID"})
		return
	}

	teamParticipantID, err := uuid.Parse(c.Param("teamParticipantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team participant ID"})
		return
	}

	var req struct {
		Lineup []LineupEntry `json:"lineup" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	submittedBy, _ := userID.(uuid.UUID)

	submitReq := &SubmitLineupRequest{
		TournamentID:      tournamentID,
		RoundID:           roundID,
		TeamParticipantID: teamParticipantID,
		Lineup:            req.Lineup,
	}

	lineup, err := h.service.SubmitLineup(c.Request.Context(), submitReq, submittedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"lineup": lineup})
}

// Match handlers

// GetTournamentMatches retrieves all team matches for a tournament
func (h *Handler) GetTournamentMatches(c *gin.Context) {
	// This would need to aggregate matches across all rounds
	// For now, redirect to use round-specific endpoint
	c.JSON(http.StatusNotImplemented, gin.H{"error": "use /rounds/:roundId/matches instead"})
}

// GetRoundMatches retrieves all team matches for a round
func (h *Handler) GetRoundMatches(c *gin.Context) {
	roundID, err := uuid.Parse(c.Param("roundId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round ID"})
		return
	}

	matches, err := h.service.GetTeamMatchesForRound(c.Request.Context(), roundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"matches": matches})
}

// GetMatch retrieves a team match by ID
func (h *Handler) GetMatch(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("matchId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match ID"})
		return
	}

	match, err := h.service.GetTeamMatch(c.Request.Context(), matchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if match == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "match not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"match": match})
}

// GetMatchBoards retrieves all boards for a team match
func (h *Handler) GetMatchBoards(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("matchId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match ID"})
		return
	}

	boards, err := h.service.GetMatchBoards(c.Request.Context(), matchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"boards": boards})
}

// EnterBoardResult enters the result for a board
func (h *Handler) EnterBoardResult(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	var req struct {
		Result string `json:"result" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.EnterBoardResult(c.Request.Context(), boardID, req.Result); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "result entered"})
}

// CalculateMatchResult calculates the match result from board results
func (h *Handler) CalculateMatchResult(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("matchId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match ID"})
		return
	}

	if err := h.service.CalculateMatchResult(c.Request.Context(), matchID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "match result calculated"})
}

// Standings handler

// GetTeamStandings retrieves team standings for a tournament
func (h *Handler) GetTeamStandings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	standings, err := h.service.GetTeamStandings(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"standings": standings})
}

// Settings handlers

// GetTeamSettings retrieves team tournament settings
func (h *Handler) GetTeamSettings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	settings, err := h.service.GetTeamTournamentSettings(c.Request.Context(), tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

// UpdateTeamSettings updates team tournament settings
func (h *Handler) UpdateTeamSettings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var settings TeamTournamentSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings.TournamentID = tournamentID

	if err := h.service.SaveTeamTournamentSettings(c.Request.Context(), &settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}
