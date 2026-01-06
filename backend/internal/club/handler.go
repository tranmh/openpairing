package club

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tranmh/openpairing/pkg/response"
)

// Handler handles HTTP requests for clubs
type Handler struct {
	service *Service
}

// NewHandler creates a new club handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers club routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	clubs := r.Group("/clubs")
	{
		// Public routes
		clubs.GET("", h.List)
		clubs.GET("/search", h.Search)
		clubs.GET("/:id", h.GetByID)
		clubs.GET("/:id/members", h.GetMembers)
		clubs.GET("/:id/stats", h.GetStats)

		// Protected routes
		protected := clubs.Group("")
		protected.Use(authMiddleware)
		{
			protected.POST("", h.Create)
			protected.PUT("/:id", h.Update)
			protected.DELETE("/:id", h.Delete)
		}
	}
}

// Create handles POST /clubs
// @Summary Create a new club
// @Tags clubs
// @Accept json
// @Produce json
// @Param club body CreateClubRequest true "Club data"
// @Success 201 {object} Club
// @Failure 400 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /clubs [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateClubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	club, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrDuplicateClub) {
			response.Conflict(c, "Club with this identifier already exists")
			return
		}
		response.InternalServerError(c, "Failed to create club")
		return
	}

	response.Created(c, club)
}

// GetByID handles GET /clubs/:id
// @Summary Get a club by ID
// @Tags clubs
// @Produce json
// @Param id path string true "Club ID"
// @Success 200 {object} Club
// @Failure 404 {object} response.Error
// @Router /clubs/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid club ID")
		return
	}

	club, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrClubNotFound) {
			response.NotFound(c, "Club not found")
			return
		}
		response.InternalServerError(c, "Failed to get club")
		return
	}

	response.Success(c, http.StatusOK, club)
}

// Update handles PUT /clubs/:id
// @Summary Update a club
// @Tags clubs
// @Accept json
// @Produce json
// @Param id path string true "Club ID"
// @Param club body UpdateClubRequest true "Club data"
// @Success 200 {object} Club
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /clubs/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid club ID")
		return
	}

	var req UpdateClubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	club, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ErrClubNotFound) {
			response.NotFound(c, "Club not found")
			return
		}
		response.InternalServerError(c, "Failed to update club")
		return
	}

	response.Success(c, http.StatusOK, club)
}

// Delete handles DELETE /clubs/:id
// @Summary Delete a club
// @Tags clubs
// @Param id path string true "Club ID"
// @Success 204
// @Failure 404 {object} response.Error
// @Router /clubs/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid club ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrClubNotFound) {
			response.NotFound(c, "Club not found")
			return
		}
		response.InternalServerError(c, "Failed to delete club")
		return
	}

	response.NoContent(c)
}

// List handles GET /clubs
// @Summary List clubs
// @Tags clubs
// @Produce json
// @Param search query string false "Search query"
// @Param federation query string false "Federation filter"
// @Param city query string false "City filter"
// @Param country query string false "Country filter"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} ClubListResponse
// @Router /clubs [get]
func (h *Handler) List(c *gin.Context) {
	var params ClubListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "Invalid query parameters")
		return
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.InternalServerError(c, "Failed to list clubs")
		return
	}

	response.Success(c, http.StatusOK, result)
}

// GetMembers handles GET /clubs/:id/members
// @Summary Get club members
// @Tags clubs
// @Produce json
// @Param id path string true "Club ID"
// @Success 200 {array} ClubMember
// @Failure 404 {object} response.Error
// @Router /clubs/{id}/members [get]
func (h *Handler) GetMembers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid club ID")
		return
	}

	members, err := h.service.GetMembers(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrClubNotFound) {
			response.NotFound(c, "Club not found")
			return
		}
		response.InternalServerError(c, "Failed to get club members")
		return
	}

	response.Success(c, http.StatusOK, members)
}

// GetStats handles GET /clubs/:id/stats
// @Summary Get club statistics
// @Tags clubs
// @Produce json
// @Param id path string true "Club ID"
// @Success 200 {object} ClubStats
// @Failure 404 {object} response.Error
// @Router /clubs/{id}/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid club ID")
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrClubNotFound) {
			response.NotFound(c, "Club not found")
			return
		}
		response.InternalServerError(c, "Failed to get club stats")
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// Search handles GET /clubs/search
// @Summary Search clubs
// @Tags clubs
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Limit" default(20)
// @Success 200 {array} Club
// @Router /clubs/search [get]
func (h *Handler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.BadRequest(c, "Search query is required")
		return
	}

	limit := 20

	clubs, err := h.service.Search(c.Request.Context(), query, limit)
	if err != nil {
		response.InternalServerError(c, "Failed to search clubs")
		return
	}

	response.Success(c, http.StatusOK, clubs)
}
