package tournament

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/federation/portal64"
	"github.com/openpairing/openpairing/internal/player"
	"github.com/rs/zerolog"
)

// Handler handles tournament HTTP requests
type Handler struct {
	service         *Service
	portal64Service *portal64.Service
	playerRepo      player.Repository
	logger          zerolog.Logger
}

// NewHandler creates a new tournament handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "tournament").Logger(),
	}
}

// NewHandlerWithImporter creates a tournament handler with player import support
func NewHandlerWithImporter(service *Service, portal64Service *portal64.Service, playerRepo player.Repository, logger zerolog.Logger) *Handler {
	return &Handler{
		service:         service,
		portal64Service: portal64Service,
		playerRepo:      playerRepo,
		logger:          logger.With().Str("handler", "tournament").Logger(),
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
		// Note: standings and crosstable are registered via standingsHandler in main.go
		// tournaments.GET("/:id/standings", h.GetStandings)
		// tournaments.GET("/:id/crosstable", h.GetCrosstable)
		tournaments.GET("/:id/rounds", h.GetRounds)
		tournaments.GET("/:id/rounds/:round/pairings", h.GetPairings)
		tournaments.GET("/:id/export/trf", h.ExportTRF)

		// Round Robin Seating Chart (public - visible to all)
		tournaments.GET("/:id/seating-chart", h.GetSeatingChart)
		tournaments.GET("/:id/rotation/:from/:to", h.GetRotationData)

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

			// Scoring settings management (organizer only)
			organizer.GET("/:id/scoring-settings", h.GetScoringSettings)
			organizer.PUT("/:id/scoring-settings", h.UpdateScoringSettings)

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

			// Special Rankings (organizer only)
			organizer.GET("/:id/special-rankings", h.ListSpecialRankings)
			organizer.POST("/:id/special-rankings", h.CreateSpecialRanking)
			organizer.PUT("/:id/special-rankings/:rankingId", h.UpdateSpecialRanking)
			organizer.DELETE("/:id/special-rankings/:rankingId", h.DeleteSpecialRanking)

			// Round Robin Settings (organizer only)
			organizer.GET("/:id/round-robin-settings", h.GetRoundRobinSettings)
			organizer.PUT("/:id/round-robin-settings", h.UpdateRoundRobinSettings)
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

	ctx := c.Request.Context()

	// If PlayerID is not provided, try to import or create player
	if req.PlayerID == uuid.Nil {
		playerID, importErr := h.resolveOrImportPlayer(ctx, &req)
		if importErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": importErr.Error()})
			return
		}
		req.PlayerID = playerID
	}

	participant, err := h.service.RegisterParticipant(ctx, id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ParticipantResponse{Participant: participant})
}

// resolveOrImportPlayer resolves an existing player or imports/creates one
func (h *Handler) resolveOrImportPlayer(ctx context.Context, req *RegisterParticipantRequest) (uuid.UUID, error) {
	// Try to import from DSB/portal64
	if req.DsbID != "" && h.portal64Service != nil {
		importedPlayer, err := h.portal64Service.ImportPlayer(ctx, req.DsbID)
		if err == nil && importedPlayer != nil {
			h.logger.Info().
				Str("dsbId", req.DsbID).
				Str("playerId", importedPlayer.ID.String()).
				Msg("Player imported from portal64")
			return importedPlayer.ID, nil
		}
		h.logger.Warn().Err(err).Str("dsbId", req.DsbID).Msg("Portal64 import failed")

		// If import failed and manual entry not allowed, return error
		if !req.ManualEntry {
			return uuid.Nil, fmt.Errorf("failed to import player from federation: %w", err)
		}
	}

	// Manual entry fallback
	if req.ManualEntry {
		if req.ManualFirstName == "" || req.ManualLastName == "" {
			return uuid.Nil, fmt.Errorf("manual entry requires firstName and lastName")
		}

		if h.playerRepo == nil {
			return uuid.Nil, fmt.Errorf("player repository not available for manual entry")
		}

		// Create manual player entry
		manualPlayer := &player.Player{
			ID:        uuid.New(),
			FirstName: req.ManualFirstName,
			LastName:  req.ManualLastName,
		}

		if req.ManualDWZ != nil {
			manualPlayer.DWZ = req.ManualDWZ
		}
		if req.ManualFideID != "" {
			manualPlayer.FideID = &req.ManualFideID
		}
		if req.ManualClub != "" {
			manualPlayer.ClubName = &req.ManualClub
		}
		if req.ManualBirthYear != nil {
			birthDate := fmt.Sprintf("%d-01-01", *req.ManualBirthYear)
			manualPlayer.BirthDate = &birthDate
		}
		if req.ManualGender != "" {
			manualPlayer.Gender = &req.ManualGender
		}
		if req.DsbID != "" {
			manualPlayer.DsbID = &req.DsbID
		}

		if err := h.playerRepo.Create(ctx, manualPlayer); err != nil {
			return uuid.Nil, fmt.Errorf("failed to create manual player: %w", err)
		}

		h.logger.Info().
			Str("playerId", manualPlayer.ID.String()).
			Str("name", manualPlayer.FirstName+" "+manualPlayer.LastName).
			Msg("Manual player entry created")

		return manualPlayer.ID, nil
	}

	return uuid.Nil, fmt.Errorf("playerId required, or provide dsbId for import, or enable manualEntry")
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

// ListSpecialRankings lists special rankings for a tournament
// @Summary List special rankings
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} SpecialRankingsResponse
// @Router /tournaments/{id}/special-rankings [get]
func (h *Handler) ListSpecialRankings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	rankings, err := h.service.ListSpecialRankings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SpecialRankingsResponse{Rankings: rankings})
}

// CreateSpecialRanking creates a new special ranking
// @Summary Create special ranking
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body CreateSpecialRankingRequest true "Special ranking data"
// @Success 201 {object} SpecialRankingResponse
// @Router /tournaments/{id}/special-rankings [post]
func (h *Handler) CreateSpecialRanking(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var req CreateSpecialRankingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sr := &SpecialRanking{
		TournamentID: tournamentID,
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		MinBirthYear: req.MinBirthYear,
		MaxBirthYear: req.MaxBirthYear,
		MinRating:    req.MinRating,
		MaxRating:    req.MaxRating,
		Gender:       req.Gender,
		SortOrder:    req.SortOrder,
		IsActive:     true,
	}

	if err := h.service.CreateSpecialRanking(c.Request.Context(), sr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, SpecialRankingResponse{Ranking: sr})
}

// UpdateSpecialRanking updates a special ranking
// @Summary Update special ranking
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param rankingId path string true "Ranking ID"
// @Param request body UpdateSpecialRankingRequest true "Special ranking data"
// @Success 200 {object} SpecialRankingResponse
// @Router /tournaments/{id}/special-rankings/{rankingId} [put]
func (h *Handler) UpdateSpecialRanking(c *gin.Context) {
	rankingID, err := uuid.Parse(c.Param("rankingId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ranking ID"})
		return
	}

	var req UpdateSpecialRankingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sr, err := h.service.GetSpecialRanking(c.Request.Context(), rankingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "special ranking not found"})
		return
	}

	if req.Name != nil {
		sr.Name = *req.Name
	}
	if req.DisplayName != nil {
		sr.DisplayName = *req.DisplayName
	}
	sr.MinBirthYear = req.MinBirthYear
	sr.MaxBirthYear = req.MaxBirthYear
	sr.MinRating = req.MinRating
	sr.MaxRating = req.MaxRating
	sr.Gender = req.Gender
	if req.SortOrder != nil {
		sr.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		sr.IsActive = *req.IsActive
	}

	if err := h.service.UpdateSpecialRanking(c.Request.Context(), sr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SpecialRankingResponse{Ranking: sr})
}

// DeleteSpecialRanking deletes a special ranking
// @Summary Delete special ranking
// @Tags tournaments
// @Param id path string true "Tournament ID"
// @Param rankingId path string true "Ranking ID"
// @Success 204
// @Router /tournaments/{id}/special-rankings/{rankingId} [delete]
func (h *Handler) DeleteSpecialRanking(c *gin.Context) {
	rankingID, err := uuid.Parse(c.Param("rankingId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ranking ID"})
		return
	}

	if err := h.service.DeleteSpecialRanking(c.Request.Context(), rankingID); err != nil {
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

// GetScoringSettings gets scoring settings for a tournament
// @Summary Get scoring settings
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} ScoringSettingsResponse
// @Router /tournaments/{id}/scoring-settings [get]
func (h *Handler) GetScoringSettings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	settings, err := h.service.GetScoringSettings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ScoringSettingsResponse{ScoringSettings: settings})
}

// UpdateScoringSettings updates scoring settings for a tournament
// @Summary Update scoring settings
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body ScoringSettings true "Scoring settings"
// @Success 200 {object} ScoringSettingsUpdateResponse
// @Router /tournaments/{id}/scoring-settings [put]
func (h *Handler) UpdateScoringSettings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	var settings ScoringSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tournament, violations, err := h.service.UpdateScoringSettings(c.Request.Context(), id, &settings)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ScoringSettingsUpdateResponse{
		Tournament:     tournament,
		FIDEViolations: violations,
	})
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

// SpecialRankingsResponse wraps special rankings
type SpecialRankingsResponse struct {
	Rankings []SpecialRanking `json:"rankings"`
}

// SpecialRankingResponse wraps a single special ranking
type SpecialRankingResponse struct {
	Ranking *SpecialRanking `json:"ranking"`
}

// Request types

// AddForbiddenPairRequest for adding a forbidden pair
type AddForbiddenPairRequest struct {
	Player1ID uuid.UUID `json:"player1Id" binding:"required"`
	Player2ID uuid.UUID `json:"player2Id" binding:"required"`
	Reason    string    `json:"reason"`
}

// CreateSpecialRankingRequest for creating a special ranking
type CreateSpecialRankingRequest struct {
	Name         string  `json:"name" binding:"required"`
	DisplayName  string  `json:"displayName" binding:"required"`
	MinBirthYear *int    `json:"minBirthYear,omitempty"`
	MaxBirthYear *int    `json:"maxBirthYear,omitempty"`
	MinRating    *int    `json:"minRating,omitempty"`
	MaxRating    *int    `json:"maxRating,omitempty"`
	Gender       *string `json:"gender,omitempty"`
	SortOrder    int     `json:"sortOrder"`
}

// UpdateSpecialRankingRequest for updating a special ranking
type UpdateSpecialRankingRequest struct {
	Name         *string `json:"name,omitempty"`
	DisplayName  *string `json:"displayName,omitempty"`
	MinBirthYear *int    `json:"minBirthYear,omitempty"`
	MaxBirthYear *int    `json:"maxBirthYear,omitempty"`
	MinRating    *int    `json:"minRating,omitempty"`
	MaxRating    *int    `json:"maxRating,omitempty"`
	Gender       *string `json:"gender,omitempty"`
	SortOrder    *int    `json:"sortOrder,omitempty"`
	IsActive     *bool   `json:"isActive,omitempty"`
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

// ScoringSettingsResponse wraps scoring settings
type ScoringSettingsResponse struct {
	ScoringSettings *ScoringSettings `json:"scoringSettings"`
}

// ScoringSettingsUpdateResponse wraps updated tournament with FIDE violations
type ScoringSettingsUpdateResponse struct {
	Tournament     *Tournament     `json:"tournament"`
	FIDEViolations []FIDEViolation `json:"fideViolations,omitempty"`
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

// ========== Round Robin Seating Chart Handlers ==========

// GetSeatingChart returns seating arrangement for a round
// @Summary Get seating chart for round robin tournament
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Param round query int false "Round number (defaults to current round)"
// @Success 200 {object} SeatingChartResponse
// @Router /tournaments/{id}/seating-chart [get]
func (h *Handler) GetSeatingChart(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament id"})
		return
	}

	roundNumber := 0
	if roundStr := c.Query("round"); roundStr != "" {
		roundNumber, err = strconv.Atoi(roundStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
			return
		}
	}

	chart, err := h.service.GetSeatingChart(c.Request.Context(), tournamentID, roundNumber)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to get seating chart")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chart)
}

// GetRotationData returns movement data between two rounds
// @Summary Get rotation data between rounds
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Param from path int true "From round number"
// @Param to path int true "To round number"
// @Success 200 {object} RotationDataResponse
// @Router /tournaments/{id}/rotation/{from}/{to} [get]
func (h *Handler) GetRotationData(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament id"})
		return
	}

	fromRound, err := strconv.Atoi(c.Param("from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from round"})
		return
	}

	toRound, err := strconv.Atoi(c.Param("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to round"})
		return
	}

	data, err := h.service.GetRotationData(c.Request.Context(), tournamentID, fromRound, toRound)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to get rotation data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetRoundRobinSettings returns round robin settings for a tournament
// @Summary Get round robin settings
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} RoundRobinSettingsResponse
// @Router /tournaments/{id}/round-robin-settings [get]
func (h *Handler) GetRoundRobinSettings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament id"})
		return
	}

	tournament, err := h.service.GetTournament(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to get tournament")
		c.JSON(http.StatusNotFound, gin.H{"error": "tournament not found"})
		return
	}

	settings := tournament.RoundRobinSettings
	if settings == nil {
		settings = DefaultRoundRobinSettings()
	}

	c.JSON(http.StatusOK, RoundRobinSettingsResponse{Settings: settings})
}

// UpdateRoundRobinSettings updates round robin settings for a tournament
// @Summary Update round robin settings
// @Tags tournaments
// @Accept json
// @Produce json
// @Param id path string true "Tournament ID"
// @Param request body UpdateRoundRobinSettingsRequest true "Settings"
// @Success 200 {object} RoundRobinSettingsResponse
// @Router /tournaments/{id}/round-robin-settings [put]
func (h *Handler) UpdateRoundRobinSettings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament id"})
		return
	}

	var req UpdateRoundRobinSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing settings or create new
	tournament, err := h.service.GetTournament(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to get tournament")
		c.JSON(http.StatusNotFound, gin.H{"error": "tournament not found"})
		return
	}

	settings := tournament.RoundRobinSettings
	if settings == nil {
		settings = DefaultRoundRobinSettings()
	}

	// Apply updates
	if req.ByePosition != nil {
		settings.ByePosition = *req.ByePosition
	}
	if req.FixedPlayer != nil {
		settings.FixedPlayer = *req.FixedPlayer
	}
	if req.ShowSeatingChart != nil {
		settings.ShowSeatingChart = *req.ShowSeatingChart
	}

	if err := h.service.UpdateRoundRobinSettings(c.Request.Context(), tournamentID, settings); err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to update round robin settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RoundRobinSettingsResponse{Settings: settings})
}

// RoundRobinSettingsResponse wraps round robin settings
type RoundRobinSettingsResponse struct {
	Settings *RoundRobinSettings `json:"settings"`
}

// UpdateRoundRobinSettingsRequest for updating round robin settings
type UpdateRoundRobinSettingsRequest struct {
	ByePosition      *int  `json:"byePosition,omitempty"`
	FixedPlayer      *int  `json:"fixedPlayer,omitempty"`
	ShowSeatingChart *bool `json:"showSeatingChart,omitempty"`
}
