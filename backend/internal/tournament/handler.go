package tournament

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Handler handles tournament HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new tournament handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "tournament").Logger(),
	}
}

// RegisterRoutes registers tournament routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc, organizerMiddleware gin.HandlerFunc) {
	tournaments := r.Group("/tournaments")
	{
		// Public routes
		tournaments.GET("", h.ListTournaments)
		tournaments.GET("/:id", h.GetTournament)
		tournaments.GET("/slug/:slug", h.GetTournamentBySlug)
		tournaments.GET("/:id/standings", h.GetStandings)
		tournaments.GET("/:id/crosstable", h.GetCrosstable)
		tournaments.GET("/:id/rounds", h.GetRounds)
		tournaments.GET("/:id/rounds/:round/pairings", h.GetPairings)
		tournaments.GET("/:id/export/trf", h.ExportTRF)

		// Protected routes (authenticated users)
		protected := tournaments.Group("")
		protected.Use(authMiddleware)
		{
			// Participants - any authenticated user can register
			protected.GET("/:id/participants", h.GetParticipants)
			protected.POST("/:id/participants", h.RegisterParticipant)
		}

		// Organizer-only routes (organizer and admin roles)
		organizer := tournaments.Group("")
		organizer.Use(authMiddleware, organizerMiddleware)
		{
			organizer.POST("", h.CreateTournament)
			organizer.POST("/:id/clone", h.CloneTournament)
			organizer.PUT("/:id", h.UpdateTournament)
			organizer.DELETE("/:id", h.DeleteTournament)
			organizer.POST("/:id/publish", h.PublishTournament)
			organizer.POST("/:id/start-registration", h.StartRegistration)
			organizer.POST("/:id/start", h.StartTournament)

			// Participant management (organizer only)
			organizer.DELETE("/:id/participants/:participantId", h.WithdrawParticipant)
			organizer.POST("/:id/assign-ranks", h.AssignStartRanks)

			// Rating management (organizer only)
			organizer.POST("/:id/lock-ratings", h.LockRatings)
			organizer.PUT("/:id/participants/:participantId/rating-override", h.SetRatingOverride)
			organizer.DELETE("/:id/participants/:participantId/rating-override", h.ClearRatingOverride)

			// Bye settings management (organizer only)
			organizer.GET("/:id/bye-settings", h.GetByeSettings)
			organizer.PUT("/:id/bye-settings", h.UpdateByeSettings)
			organizer.GET("/:id/fide-violations", h.GetFIDEViolations)

			// Bye request management (organizer only)
			organizer.POST("/:id/participants/:participantId/bye-request", h.RequestHalfPointBye)
			organizer.DELETE("/:id/participants/:participantId/bye-request/:round", h.CancelByeRequest)
			organizer.GET("/:id/participants/:participantId/bye-status", h.GetParticipantByeStatus)

			// Late entry and withdrawal (organizer only)
			organizer.POST("/:id/late-entry", h.AddLateEntry)
			organizer.POST("/:id/participants/:participantId/withdraw", h.WithdrawWithReason)

			// Rounds & Pairings (organizer only)
			organizer.POST("/:id/rounds/:round/pair", h.GeneratePairings)
			organizer.POST("/:id/rounds/:round/start", h.StartRound)
			organizer.POST("/:id/rounds/:round/complete", h.CompleteRound)

			// Games & Results (organizer only)
			organizer.GET("/:id/games/:gameId", h.GetGame)
			organizer.PUT("/:id/games/:gameId/result", h.EnterResult)

			// Forbidden Pairs (organizer only)
			organizer.GET("/:id/forbidden-pairs", h.GetForbiddenPairs)
			organizer.POST("/:id/forbidden-pairs", h.AddForbiddenPair)
			organizer.DELETE("/:id/forbidden-pairs/:pairId", h.RemoveForbiddenPair)
		}
	}
}

// CreateTournament creates a new tournament
// @Summary Create tournament
// @Tags tournaments
// @Accept json
// @Produce json
// @Param request body CreateTournamentRequest true "Tournament data"
// @Success 201 {object} TournamentResponse
// @Router /tournaments [post]
func (h *Handler) CreateTournament(c *gin.Context) {
	var req CreateTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	organizerID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	tournament, err := h.service.CreateTournament(c.Request.Context(), &req, organizerID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create tournament")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, TournamentResponse{Tournament: tournament})
}

// CloneTournament clones an existing tournament
// @Summary Clone tournament
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body CloneTournamentRequest true "Clone data"
// @Success 201 {object} TournamentResponse
// @Router /tournaments/{id}/clone [post]
func (h *Handler) CloneTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req CloneTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tournament, err := h.service.CloneTournament(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to clone tournament")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, TournamentResponse{Tournament: tournament})
}

// GetTournament gets a tournament by ID
// @Summary Get tournament
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} TournamentResponse
// @Router /tournaments/{id} [get]
func (h *Handler) GetTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	tournament, err := h.service.GetTournament(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tournament not found"})
		return
	}

	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

// GetTournamentBySlug gets a tournament by slug
// @Summary Get tournament by slug
// @Tags tournaments
// @Produce json
// @Param slug path string true "Tournament slug"
// @Success 200 {object} TournamentResponse
// @Router /tournaments/slug/{slug} [get]
func (h *Handler) GetTournamentBySlug(c *gin.Context) {
	slug := c.Param("slug")

	tournament, err := h.service.GetTournamentBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tournament not found"})
		return
	}

	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

// UpdateTournament updates a tournament
// @Summary Update tournament
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body UpdateTournamentRequest true "Tournament data"
// @Success 200 {object} TournamentResponse
// @Router /tournaments/{id} [put]
func (h *Handler) UpdateTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req UpdateTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tournament, err := h.service.UpdateTournament(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

// DeleteTournament deletes a tournament
// @Summary Delete tournament
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Success 204
// @Router /tournaments/{id} [delete]
func (h *Handler) DeleteTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	if err := h.service.DeleteTournament(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListTournaments lists tournaments
// @Summary List tournaments
// @Tags tournaments
// @Produce json
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset results"
// @Success 200 {object} TournamentListResponse
// @Router /tournaments [get]
func (h *Handler) ListTournaments(c *gin.Context) {
	filter := &ListFilters{
		Limit: 20,
		Page:  0,
	}

	if status := c.Query("status"); status != "" {
		s := TournamentStatus(status)
		filter.Status = &s
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Page = o / filter.Limit // Convert offset to page
		}
	}

	tournaments, total, err := h.service.ListTournaments(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TournamentListResponse{
		Tournaments: tournaments,
		Total:       int(total),
		Limit:       filter.Limit,
		Offset:      filter.Page * filter.Limit,
	})
}

// PublishTournament publishes a tournament
// @Summary Publish tournament
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Success 200 {object} TournamentResponse
// @Router /tournaments/{id}/publish [post]
func (h *Handler) PublishTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	tournament, err := h.service.PublishTournament(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

// StartRegistration opens registration
// @Summary Start registration
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Success 200 {object} TournamentResponse
// @Router /tournaments/{id}/start-registration [post]
func (h *Handler) StartRegistration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	tournament, err := h.service.StartRegistration(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

// StartTournament starts a tournament
// @Summary Start tournament
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Success 200 {object} TournamentResponse
// @Router /tournaments/{id}/start [post]
func (h *Handler) StartTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	tournament, err := h.service.StartTournament(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

// GetParticipants gets tournament participants
// @Summary Get participants
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} ParticipantListResponse
// @Router /tournaments/{id}/participants [get]
func (h *Handler) GetParticipants(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	filter := &ParticipantFilter{}
	if active := c.Query("active"); active == "true" {
		filter.ActiveOnly = true
	}

	participants, err := h.service.GetParticipants(c.Request.Context(), id, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ParticipantListResponse{
		Participants: participants,
		Total:        len(participants),
	})
}

// RegisterParticipant registers a player
// @Summary Register participant
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body RegisterParticipantRequest true "Registration data"
// @Success 201 {object} ParticipantResponse
// @Router /tournaments/{id}/participants [post]
func (h *Handler) RegisterParticipant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req RegisterParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	participant, err := h.service.RegisterParticipant(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ParticipantResponse{Participant: participant})
}

// WithdrawParticipant withdraws a participant
// @Summary Withdraw participant
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Success 204
// @Router /tournaments/{id}/participants/{participantId} [delete]
func (h *Handler) WithdrawParticipant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	if err := h.service.WithdrawParticipant(c.Request.Context(), id, participantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// AssignStartRanks assigns start ranks
// @Summary Assign start ranks
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Success 204
// @Router /tournaments/{id}/assign-ranks [post]
func (h *Handler) AssignStartRanks(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	if err := h.service.AssignStartRanks(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRounds gets tournament rounds
// @Summary Get rounds
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} RoundListResponse
// @Router /tournaments/{id}/rounds [get]
func (h *Handler) GetRounds(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	rounds, err := h.service.GetRounds(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RoundListResponse{Rounds: rounds})
}

// GeneratePairings generates pairings for a round
// @Summary Generate pairings
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {object} PairingsResponse
// @Router /tournaments/{id}/rounds/{round}/pair [post]
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

	games, err := h.service.GeneratePairings(c.Request.Context(), id, round, arbiterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, PairingsResponse{Games: games})
}

// GetPairings gets pairings for a round
// @Summary Get pairings
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {object} PairingsResponse
// @Router /tournaments/{id}/rounds/{round}/pairings [get]
func (h *Handler) GetPairings(c *gin.Context) {
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

	games, err := h.service.GetPairings(c.Request.Context(), id, round)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, PairingsResponse{Games: games})
}

// StartRound starts a round
// @Summary Start round
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {object} RoundResponse
// @Router /tournaments/{id}/rounds/{round}/start [post]
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

	r, err := h.service.StartRound(c.Request.Context(), id, round)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RoundResponse{Round: r})
}

// CompleteRound completes a round
// @Summary Complete round
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Param round path int true "Round number"
// @Success 200 {object} RoundResponse
// @Router /tournaments/{id}/rounds/{round}/complete [post]
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

	r, err := h.service.CompleteRound(c.Request.Context(), id, round)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RoundResponse{Round: r})
}

// GetGame gets a game by ID
// @Summary Get game
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Param gameId path string true "Game ID"
// @Success 200 {object} GameResponse
// @Router /tournaments/{id}/games/{gameId} [get]
func (h *Handler) GetGame(c *gin.Context) {
	gameID, err := uuid.Parse(c.Param("gameId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	game, err := h.service.repo.GetGame(c.Request.Context(), gameID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
		return
	}

	c.JSON(http.StatusOK, GameResponse{Game: game})
}

// EnterResult enters a game result
// @Summary Enter result
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param gameId path string true "Game ID"
// @Param request body EnterResultRequest true "Result data"
// @Success 200 {object} GameResponse
// @Router /tournaments/{id}/games/{gameId}/result [put]
func (h *Handler) EnterResult(c *gin.Context) {
	gameID, err := uuid.Parse(c.Param("gameId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game ID"})
		return
	}

	var req EnterResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, _ := c.Get("user_id")
	arbiterID, _ := uuid.Parse(userIDStr.(string))

	game, err := h.service.EnterResult(c.Request.Context(), gameID, &req, arbiterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, GameResponse{Game: game})
}

// GetStandings gets tournament standings
// @Summary Get standings
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} StandingsResponse
// @Router /tournaments/{id}/standings [get]
func (h *Handler) GetStandings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	standings, err := h.service.GetStandings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, StandingsResponse{Standings: standings})
}

// GetCrosstable gets tournament crosstable
// @Summary Get crosstable
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} CrosstableResponse
// @Router /tournaments/{id}/crosstable [get]
func (h *Handler) GetCrosstable(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	crosstable, err := h.service.GetCrosstable(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, CrosstableResponse{Crosstable: crosstable})
}

// ExportTRF exports tournament to TRF format
// @Summary Export TRF
// @Tags tournaments
// @Produce text/plain
// @Param id path string true "Tournament ID"
// @Success 200 {string} string "TRF file content"
// @Router /tournaments/{id}/export/trf [get]
func (h *Handler) ExportTRF(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	trf, err := h.service.ExportTRF(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.Header("Content-Disposition", "attachment; filename=tournament.trf")
	c.String(http.StatusOK, trf)
}

// GetForbiddenPairs gets forbidden pairs
// @Summary Get forbidden pairs
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} ForbiddenPairsResponse
// @Router /tournaments/{id}/forbidden-pairs [get]
func (h *Handler) GetForbiddenPairs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	pairs, err := h.service.GetForbiddenPairs(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ForbiddenPairsResponse{ForbiddenPairs: pairs})
}

// AddForbiddenPair adds a forbidden pair
// @Summary Add forbidden pair
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body AddForbiddenPairRequest true "Forbidden pair data"
// @Success 201
// @Router /tournaments/{id}/forbidden-pairs [post]
func (h *Handler) AddForbiddenPair(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req AddForbiddenPairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddForbiddenPair(c.Request.Context(), id, req.Player1ID, req.Player2ID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// RemoveForbiddenPair removes a forbidden pair
// @Summary Remove forbidden pair
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Param pairId path string true "Pair ID"
// @Success 204
// @Router /tournaments/{id}/forbidden-pairs/{pairId} [delete]
func (h *Handler) RemoveForbiddenPair(c *gin.Context) {
	pairID, err := uuid.Parse(c.Param("pairId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pair ID"})
		return
	}

	if err := h.service.RemoveForbiddenPair(c.Request.Context(), pairID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// LockRatings locks ratings for a tournament
// @Summary Lock ratings
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Success 204
// @Router /tournaments/{id}/lock-ratings [post]
func (h *Handler) LockRatings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	if err := h.service.LockRatings(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// SetRatingOverride sets a manual rating override for a participant
// @Summary Set rating override
// @Tags tournaments
// @Accept json
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Param request body SetRatingOverrideRequest true "Rating override"
// @Success 204
// @Router /tournaments/{id}/participants/{participantId}/rating-override [put]
func (h *Handler) SetRatingOverride(c *gin.Context) {
	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	var req SetRatingOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.SetParticipantRatingOverride(c.Request.Context(), participantID, &req.Rating); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ClearRatingOverride clears the manual rating override for a participant
// @Summary Clear rating override
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Success 204
// @Router /tournaments/{id}/participants/{participantId}/rating-override [delete]
func (h *Handler) ClearRatingOverride(c *gin.Context) {
	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	if err := h.service.SetParticipantRatingOverride(c.Request.Context(), participantID, nil); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetByeSettings gets bye settings for a tournament
// @Summary Get bye settings
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} ByeSettingsResponse
// @Router /tournaments/{id}/bye-settings [get]
func (h *Handler) GetByeSettings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	settings, err := h.service.GetByeSettings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ByeSettingsResponse{ByeSettings: settings})
}

// UpdateByeSettings updates bye settings for a tournament
// @Summary Update bye settings
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body ByeSettings true "Bye settings"
// @Success 200 {object} ByeSettingsUpdateResponse
// @Router /tournaments/{id}/bye-settings [put]
func (h *Handler) UpdateByeSettings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var settings ByeSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tournament, violations, err := h.service.UpdateByeSettings(c.Request.Context(), id, &settings)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ByeSettingsUpdateResponse{
		Tournament:     tournament,
		FIDEViolations: violations,
	})
}

// GetFIDEViolations gets FIDE rule violations for current bye settings
// @Summary Get FIDE violations
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} FIDEViolationsResponse
// @Router /tournaments/{id}/fide-violations [get]
func (h *Handler) GetFIDEViolations(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	violations, err := h.service.GetFIDEViolations(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, FIDEViolationsResponse{Violations: violations})
}

// RequestHalfPointBye requests a half-point bye for a participant
// @Summary Request half-point bye
// @Tags tournaments
// @Accept json
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Param request body ByeRequestRequest true "Bye request"
// @Success 204
// @Router /tournaments/{id}/participants/{participantId}/bye-request [post]
func (h *Handler) RequestHalfPointBye(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	var req ByeRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RequestHalfPointBye(c.Request.Context(), id, participantID, req.Round); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CancelByeRequest cancels a bye request for a participant
// @Summary Cancel bye request
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Param round path int true "Round number"
// @Success 204
// @Router /tournaments/{id}/participants/{participantId}/bye-request/{round} [delete]
func (h *Handler) CancelByeRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	round, err := strconv.Atoi(c.Param("round"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
		return
	}

	if err := h.service.CancelByeRequest(c.Request.Context(), id, participantID, round); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetParticipantByeStatus gets detailed bye/withdrawal status for a participant
// @Summary Get participant bye status
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Success 200 {object} ParticipantByeStatusResponse
// @Router /tournaments/{id}/participants/{participantId}/bye-status [get]
func (h *Handler) GetParticipantByeStatus(c *gin.Context) {
	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	status, err := h.service.GetParticipantByeStatus(c.Request.Context(), participantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ParticipantByeStatusResponse{Status: status})
}

// AddLateEntry adds a late entry participant
// @Summary Add late entry
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body LateEntryRequest true "Late entry data"
// @Success 201 {object} ParticipantResponse
// @Router /tournaments/{id}/late-entry [post]
func (h *Handler) AddLateEntry(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req LateEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	regReq := &RegisterParticipantRequest{
		PlayerID:       req.PlayerID,
		SectionID:      req.SectionID,
		StartingRating: req.StartingRating,
	}

	participant, err := h.service.AddLateEntry(c.Request.Context(), id, regReq, req.EntryRound)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ParticipantResponse{Participant: participant})
}

// WithdrawWithReason withdraws a participant with reason tracking
// @Summary Withdraw participant with reason
// @Tags tournaments
// @Accept json
// @Param id path string true "Tournament ID"
// @Param participantId path string true "Participant ID"
// @Param request body WithdrawRequest true "Withdrawal data"
// @Success 204
// @Router /tournaments/{id}/participants/{participantId}/withdraw [post]
func (h *Handler) WithdrawWithReason(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for simple withdrawal
		req = WithdrawRequest{}
	}

	if err := h.service.WithdrawParticipantWithDetails(c.Request.Context(), id, participantID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Response types

// TournamentResponse wraps a tournament
type TournamentResponse struct {
	Tournament *Tournament `json:"tournament"`
}

// TournamentListResponse wraps a list of tournaments
type TournamentListResponse struct {
	Tournaments []Tournament `json:"tournaments"`
	Total       int          `json:"total"`
	Limit       int          `json:"limit"`
	Offset      int          `json:"offset"`
}

// ParticipantResponse wraps a participant
type ParticipantResponse struct {
	Participant *Participant `json:"participant"`
}

// ParticipantListResponse wraps a list of participants
type ParticipantListResponse struct {
	Participants []Participant `json:"participants"`
	Total        int           `json:"total"`
}

// RoundResponse wraps a round
type RoundResponse struct {
	Round *Round `json:"round"`
}

// RoundListResponse wraps a list of rounds
type RoundListResponse struct {
	Rounds []Round `json:"rounds"`
}

// GameResponse wraps a game
type GameResponse struct {
	Game *Game `json:"game"`
}

// PairingsResponse wraps pairings
type PairingsResponse struct {
	Games []Game `json:"games"`
}

// StandingsResponse wraps standings
type StandingsResponse struct {
	Standings []Participant `json:"standings"`
}

// CrosstableResponse wraps a crosstable
type CrosstableResponse struct {
	Crosstable *Crosstable `json:"crosstable"`
}

// ForbiddenPairsResponse wraps forbidden pairs
type ForbiddenPairsResponse struct {
	ForbiddenPairs []ForbiddenPair `json:"forbiddenPairs"`
}

// Request types

// AddForbiddenPairRequest for adding a forbidden pair
type AddForbiddenPairRequest struct {
	Player1ID uuid.UUID `json:"player1Id" binding:"required"`
	Player2ID uuid.UUID `json:"player2Id" binding:"required"`
	Reason    string    `json:"reason"`
}

// EnterResultRequest for entering a game result
type EnterResultRequest struct {
	Result string `json:"result" binding:"required"`
	PGN    string `json:"pgn"`
}

// SetRatingOverrideRequest for setting a manual rating override
type SetRatingOverrideRequest struct {
	Rating int `json:"rating" binding:"required,min=0"`
}

// ByeSettingsResponse wraps bye settings
type ByeSettingsResponse struct {
	ByeSettings *ByeSettings `json:"byeSettings"`
}

// ByeSettingsUpdateResponse wraps updated tournament with FIDE violations
type ByeSettingsUpdateResponse struct {
	Tournament     *Tournament     `json:"tournament"`
	FIDEViolations []FIDEViolation `json:"fideViolations,omitempty"`
}

// FIDEViolationsResponse wraps FIDE violations
type FIDEViolationsResponse struct {
	Violations []FIDEViolation `json:"violations"`
}

// ByeRequestRequest for requesting a bye
type ByeRequestRequest struct {
	Round int `json:"round" binding:"required,min=1"`
}

// LateEntryRequest for adding a late entry
type LateEntryRequest struct {
	PlayerID       uuid.UUID  `json:"playerId" binding:"required"`
	SectionID      *uuid.UUID `json:"sectionId"`
	StartingRating int        `json:"startingRating"`
	EntryRound     int        `json:"entryRound" binding:"required,min=2"`
}

// WithdrawRequest for withdrawing with reason
type WithdrawRequest struct {
	Reason string `json:"reason"`
}

// ParticipantByeStatusResponse wraps participant bye status
type ParticipantByeStatusResponse struct {
	Status *ParticipantByeStatus `json:"status"`
}

// TournamentFilter helper for StartDate/EndDate
type TournamentFilter struct {
	Status         *TournamentStatus
	OrganizationID *uuid.UUID
	StartDateFrom  *time.Time
	StartDateTo    *time.Time
	Limit          int
	Offset         int
}

// ParticipantFilter helper
type ParticipantFilter struct {
	ActiveOnly bool
	SectionID  *uuid.UUID
	Limit      int
	Offset     int
}
