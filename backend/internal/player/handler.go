package player

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tranmh/openpairing/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the player routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	players := r.Group("/players")
	{
		// Public routes
		players.GET("", h.SearchPlayers)
		players.GET("/:id", h.GetPlayer)
		players.GET("/:id/history", h.GetRatingHistory)

		// Protected routes (require authentication)
		protected := players.Group("")
		protected.Use(authMiddleware)
		{
			// Admin only routes
			admin := protected.Group("")
			admin.Use(adminMiddleware)
			{
				admin.POST("", h.CreatePlayer)
				admin.PUT("/:id", h.UpdatePlayer)
				admin.DELETE("/:id", h.DeletePlayer)
			}
		}
	}

	// Club routes
	clubs := r.Group("/clubs")
	{
		clubs.GET("", h.ListClubs)
		clubs.GET("/:id", h.GetClub)

		protected := clubs.Group("")
		protected.Use(authMiddleware)
		{
			admin := protected.Group("")
			admin.Use(adminMiddleware)
			{
				admin.POST("", h.CreateClub)
			}
		}
	}
}

// SearchPlayers godoc
// @Summary Search players
// @Description Search for players by name, federation ID, or filter by rating
// @Tags players
// @Accept json
// @Produce json
// @Param q query string false "Search query (name, FIDE ID, DSB ID)"
// @Param federation query string false "Filter by federation (fide, dsb, oesb)"
// @Param minRating query int false "Minimum rating"
// @Param maxRating query int false "Maximum rating"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(20)
// @Success 200 {object} PlayerListResponse
// @Failure 400 {object} response.ErrorResponse
// @Router /players [get]
func (h *Handler) SearchPlayers(c *gin.Context) {
	var req SearchPlayersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	result, err := h.service.SearchPlayers(c.Request.Context(), &req)
	if err != nil {
		response.InternalServerError(c, "Failed to search players")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetPlayer godoc
// @Summary Get player by ID
// @Description Get a player's details by their ID
// @Tags players
// @Accept json
// @Produce json
// @Param id path string true "Player ID" format(uuid)
// @Success 200 {object} Player
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /players/{id} [get]
func (h *Handler) GetPlayer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid player ID: must be a valid UUID")
		return
	}

	player, err := h.service.GetPlayer(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPlayerNotFound) {
			response.NotFound(c, "Player not found")
			return
		}
		response.InternalServerError(c, "Failed to get player")
		return
	}

	c.JSON(http.StatusOK, player)
}

// CreatePlayer godoc
// @Summary Create a new player
// @Description Create a new player (admin only)
// @Tags players
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param player body CreatePlayerRequest true "Player data"
// @Success 201 {object} Player
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Router /players [post]
func (h *Handler) CreatePlayer(c *gin.Context) {
	var req CreatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	player, err := h.service.CreatePlayer(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateFideID):
			response.Conflict(c, "Player with this FIDE ID already exists")
		case errors.Is(err, ErrDuplicateDsbID):
			response.Conflict(c, "Player with this DSB ID already exists")
		case errors.Is(err, ErrDuplicateOesbID):
			response.Conflict(c, "Player with this ÖSB ID already exists")
		default:
			response.InternalServerError(c, "Failed to create player")
		}
		return
	}

	c.JSON(http.StatusCreated, player)
}

// UpdatePlayer godoc
// @Summary Update a player
// @Description Update an existing player (admin only)
// @Tags players
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Player ID" format(uuid)
// @Param player body UpdatePlayerRequest true "Player data"
// @Success 200 {object} Player
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Router /players/{id} [put]
func (h *Handler) UpdatePlayer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid player ID: must be a valid UUID")
		return
	}

	var req UpdatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	player, err := h.service.UpdatePlayer(c.Request.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrPlayerNotFound):
			response.NotFound(c, "Player not found")
		case errors.Is(err, ErrDuplicateFideID):
			response.Conflict(c, "Player with this FIDE ID already exists")
		case errors.Is(err, ErrDuplicateDsbID):
			response.Conflict(c, "Player with this DSB ID already exists")
		case errors.Is(err, ErrDuplicateOesbID):
			response.Conflict(c, "Player with this ÖSB ID already exists")
		default:
			response.InternalServerError(c, "Failed to update player")
		}
		return
	}

	c.JSON(http.StatusOK, player)
}

// DeletePlayer godoc
// @Summary Delete a player
// @Description Delete a player (admin only)
// @Tags players
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Player ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /players/{id} [delete]
func (h *Handler) DeletePlayer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid player ID: must be a valid UUID")
		return
	}

	if err := h.service.DeletePlayer(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrPlayerNotFound) {
			response.NotFound(c, "Player not found")
			return
		}
		response.InternalServerError(c, "Failed to delete player")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRatingHistory godoc
// @Summary Get player rating history
// @Description Get the rating history for a player
// @Tags players
// @Accept json
// @Produce json
// @Param id path string true "Player ID" format(uuid)
// @Param type query string false "Rating type (fide_standard, fide_rapid, dwz, oesb)"
// @Param limit query int false "Maximum entries to return" default(50)
// @Success 200 {array} RatingHistory
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /players/{id}/history [get]
func (h *Handler) GetRatingHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid player ID: must be a valid UUID")
		return
	}

	ratingType := c.DefaultQuery("type", "")
	limit := 50 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if _, err := c.GetQuery("limit"); err {
			// Parse limit from query
		}
	}

	history, err := h.service.GetRatingHistory(c.Request.Context(), id, ratingType, limit)
	if err != nil {
		response.InternalServerError(c, "Failed to get rating history")
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// Club handlers

// ListClubs godoc
// @Summary List clubs
// @Description Get a paginated list of clubs
// @Tags clubs
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /clubs [get]
func (h *Handler) ListClubs(c *gin.Context) {
	page := 1
	limit := 20
	// Parse pagination params
	if p := c.Query("page"); p != "" {
		// parse page
	}
	if l := c.Query("limit"); l != "" {
		// parse limit
	}

	clubs, total, err := h.service.ListClubs(c.Request.Context(), page, limit)
	if err != nil {
		response.InternalServerError(c, "Failed to list clubs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clubs": clubs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetClub godoc
// @Summary Get club by ID
// @Description Get a club's details by their ID
// @Tags clubs
// @Accept json
// @Produce json
// @Param id path string true "Club ID" format(uuid)
// @Success 200 {object} Club
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /clubs/{id} [get]
func (h *Handler) GetClub(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid club ID: must be a valid UUID")
		return
	}

	club, err := h.service.GetClub(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrClubNotFound) {
			response.NotFound(c, "Club not found")
			return
		}
		response.InternalServerError(c, "Failed to get club")
		return
	}

	c.JSON(http.StatusOK, club)
}

// CreateClub godoc
// @Summary Create a new club
// @Description Create a new club (admin only)
// @Tags clubs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param club body Club true "Club data"
// @Success 201 {object} Club
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /clubs [post]
func (h *Handler) CreateClub(c *gin.Context) {
	var club Club
	if err := c.ShouldBindJSON(&club); err != nil {
		response.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if err := h.service.CreateClub(c.Request.Context(), &club); err != nil {
		response.InternalServerError(c, "Failed to create club")
		return
	}

	c.JSON(http.StatusCreated, club)
}
