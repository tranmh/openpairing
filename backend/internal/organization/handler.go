package organization

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tranmh/openpairing/pkg/response"
)

// Handler handles organization HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new organization handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "organization").Logger(),
	}
}

// RegisterRoutes registers the organization routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	orgs := r.Group("/organizations")
	{
		// Public routes
		orgs.GET("", h.ListOrganizations)
		orgs.GET("/:id", h.GetOrganization)
		orgs.GET("/:id/tournaments", h.GetOrganizationTournaments)

		// Protected routes (require authentication)
		protected := orgs.Group("")
		protected.Use(authMiddleware)
		{
			protected.POST("", h.CreateOrganization)
			protected.PUT("/:id", h.UpdateOrganization)
			protected.DELETE("/:id", h.DeleteOrganization)

			// Member management
			protected.GET("/:id/members", h.GetMembers)
			protected.POST("/:id/members", h.AddMember)
			protected.PUT("/:id/members/:userId", h.UpdateMemberRole)
			protected.DELETE("/:id/members/:userId", h.RemoveMember)
		}
	}

	// User's organizations route
	r.GET("/users/me/organizations", authMiddleware, h.GetMyOrganizations)
}

// ListOrganizations lists organizations with optional filtering
// @Summary List organizations
// @Description List organizations with optional filtering by name or type
// @Tags organizations
// @Produce json
// @Param q query string false "Search query"
// @Param type query string false "Filter by type (federation, club, private)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(20)
// @Success 200 {object} OrganizationListResponse
// @Failure 400 {object} response.ErrorResponse
// @Router /organizations [get]
func (h *Handler) ListOrganizations(c *gin.Context) {
	var req ListOrganizationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	result, err := h.service.ListOrganizations(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list organizations")
		response.InternalServerError(c, "Failed to list organizations")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetOrganization retrieves an organization by ID
// @Summary Get organization
// @Description Get an organization by ID
// @Tags organizations
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Success 200 {object} OrganizationWithMembers
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /organizations/{id} [get]
func (h *Handler) GetOrganization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid organization ID")
		return
	}

	org, err := h.service.GetOrganizationWithMembers(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "organization not found" {
			response.NotFound(c, "Organization not found")
			return
		}
		h.logger.Error().Err(err).Str("id", idStr).Msg("Failed to get organization")
		response.InternalServerError(c, "Failed to get organization")
		return
	}

	c.JSON(http.StatusOK, org)
}

// CreateOrganization creates a new organization
// @Summary Create organization
// @Description Create a new organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param body body CreateOrganizationRequest true "Organization data"
// @Success 201 {object} Organization
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /organizations [post]
func (h *Handler) CreateOrganization(c *gin.Context) {
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	org, err := h.service.CreateOrganization(c.Request.Context(), &req, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "slug already exists" {
			response.BadRequest(c, "Slug already exists")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to create organization")
		response.InternalServerError(c, "Failed to create organization")
		return
	}

	c.JSON(http.StatusCreated, org)
}

// UpdateOrganization updates an organization
// @Summary Update organization
// @Description Update an existing organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Param body body UpdateOrganizationRequest true "Organization data"
// @Success 200 {object} Organization
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /organizations/{id} [put]
func (h *Handler) UpdateOrganization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid organization ID")
		return
	}

	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	org, err := h.service.UpdateOrganization(c.Request.Context(), id, &req, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "organization not found" {
			response.NotFound(c, "Organization not found")
			return
		}
		if err.Error() == "unauthorized: must be admin or owner to update organization" {
			response.Forbidden(c, err.Error())
			return
		}
		if err.Error() == "slug already exists" {
			response.BadRequest(c, "Slug already exists")
			return
		}
		h.logger.Error().Err(err).Str("id", idStr).Msg("Failed to update organization")
		response.InternalServerError(c, "Failed to update organization")
		return
	}

	c.JSON(http.StatusOK, org)
}

// DeleteOrganization deletes an organization
// @Summary Delete organization
// @Description Delete an organization (owner only)
// @Tags organizations
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /organizations/{id} [delete]
func (h *Handler) DeleteOrganization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid organization ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err = h.service.DeleteOrganization(c.Request.Context(), id, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "organization not found" {
			response.NotFound(c, "Organization not found")
			return
		}
		if err.Error() == "unauthorized: only owners can delete organizations" {
			response.Forbidden(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Str("id", idStr).Msg("Failed to delete organization")
		response.InternalServerError(c, "Failed to delete organization")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted successfully"})
}

// GetMembers retrieves all members of an organization
// @Summary Get organization members
// @Description Get all members of an organization
// @Tags organizations
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Success 200 {array} OrganizationMember
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /organizations/{id}/members [get]
func (h *Handler) GetMembers(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid organization ID")
		return
	}

	members, err := h.service.GetMembers(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("id", idStr).Msg("Failed to get members")
		response.InternalServerError(c, "Failed to get members")
		return
	}

	c.JSON(http.StatusOK, members)
}

// AddMember adds a member to an organization
// @Summary Add member
// @Description Add a member to an organization (admin/owner only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Param body body AddMemberRequest true "Member data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /organizations/{id}/members [post]
func (h *Handler) AddMember(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid organization ID")
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err = h.service.AddMember(c.Request.Context(), id, &req, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "unauthorized: must be admin or owner to add members" ||
			err.Error() == "unauthorized: only owners can add other owners" {
			response.Forbidden(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Str("org_id", idStr).Msg("Failed to add member")
		response.InternalServerError(c, "Failed to add member")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added successfully"})
}

// UpdateMemberRole updates a member's role
// @Summary Update member role
// @Description Update a member's role in an organization (owner only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Param userId path string true "User ID" format(uuid)
// @Param body body UpdateMemberRequest true "New role"
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /organizations/{id}/members/{userId} [put]
func (h *Handler) UpdateMemberRole(c *gin.Context) {
	orgIDStr := c.Param("id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid organization ID")
		return
	}

	memberIDStr := c.Param("userId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err = h.service.UpdateMemberRole(c.Request.Context(), orgID, memberID, &req, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "member not found" {
			response.NotFound(c, "Member not found")
			return
		}
		if err.Error() == "unauthorized: only owners can modify member roles" ||
			err.Error() == "cannot demote the last owner of an organization" {
			response.Forbidden(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Str("org_id", orgIDStr).Msg("Failed to update member role")
		response.InternalServerError(c, "Failed to update member role")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member role updated successfully"})
}

// RemoveMember removes a member from an organization
// @Summary Remove member
// @Description Remove a member from an organization (admin/owner only)
// @Tags organizations
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Param userId path string true "User ID" format(uuid)
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /organizations/{id}/members/{userId} [delete]
func (h *Handler) RemoveMember(c *gin.Context) {
	orgIDStr := c.Param("id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid organization ID")
		return
	}

	memberIDStr := c.Param("userId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err = h.service.RemoveMember(c.Request.Context(), orgID, memberID, userID.(uuid.UUID))
	if err != nil {
		if err.Error() == "member not found" {
			response.NotFound(c, "Member not found")
			return
		}
		if err.Error() == "unauthorized: must be admin or owner to remove members" ||
			err.Error() == "unauthorized: only owners can remove other owners" ||
			err.Error() == "cannot remove the last owner of an organization" {
			response.Forbidden(c, err.Error())
			return
		}
		h.logger.Error().Err(err).Str("org_id", orgIDStr).Msg("Failed to remove member")
		response.InternalServerError(c, "Failed to remove member")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// GetOrganizationTournaments retrieves tournaments belonging to an organization
// @Summary Get organization tournaments
// @Description Get tournaments belonging to an organization
// @Tags organizations
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(20)
// @Success 200 {object} OrganizationTournamentsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /organizations/{id}/tournaments [get]
func (h *Handler) GetOrganizationTournaments(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid organization ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.service.GetOrganizationTournaments(c.Request.Context(), id, page, limit)
	if err != nil {
		if err.Error() == "organization not found" {
			response.NotFound(c, "Organization not found")
			return
		}
		h.logger.Error().Err(err).Str("id", idStr).Msg("Failed to get organization tournaments")
		response.InternalServerError(c, "Failed to get organization tournaments")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetMyOrganizations retrieves the current user's organizations
// @Summary Get my organizations
// @Description Get organizations the current user belongs to
// @Tags organizations
// @Produce json
// @Success 200 {array} Organization
// @Failure 401 {object} response.ErrorResponse
// @Router /users/me/organizations [get]
func (h *Handler) GetMyOrganizations(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orgs, err := h.service.GetUserOrganizations(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get user organizations")
		response.InternalServerError(c, "Failed to get user organizations")
		return
	}

	c.JSON(http.StatusOK, orgs)
}
