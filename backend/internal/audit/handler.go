package audit

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/openpairing/openpairing/pkg/response"
)

// Handler handles audit log HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new audit handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers audit routes (admin only)
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware, adminMiddleware gin.HandlerFunc) {
	audit := r.Group("/audit")
	audit.Use(authMiddleware, adminMiddleware)
	{
		audit.GET("", h.List)
		audit.GET("/:id", h.GetByID)
		audit.GET("/entity/:type/:id", h.GetEntityHistory)
		audit.GET("/user/:userId", h.GetUserActivity)
	}
}

// List handles GET /audit
// @Summary List audit logs
// @Tags audit
// @Security BearerAuth
// @Produce json
// @Param userId query string false "Filter by user ID"
// @Param action query string false "Filter by action"
// @Param entityType query string false "Filter by entity type"
// @Param entityId query string false "Filter by entity ID"
// @Param startDate query string false "Filter by start date"
// @Param endDate query string false "Filter by end date"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(50)
// @Success 200 {object} AuditLogListResponse
// @Router /audit [get]
func (h *Handler) List(c *gin.Context) {
	var params AuditLogListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "Invalid query parameters")
		return
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.InternalServerError(c, "Failed to list audit logs")
		return
	}

	response.Success(c, http.StatusOK, result)
}

// GetByID handles GET /audit/:id
// @Summary Get audit log by ID
// @Tags audit
// @Security BearerAuth
// @Produce json
// @Param id path string true "Audit log ID"
// @Success 200 {object} AuditLog
// @Failure 404 {object} response.ErrorResponse
// @Router /audit/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid audit log ID")
		return
	}

	log, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Audit log not found")
		return
	}

	response.Success(c, http.StatusOK, log)
}

// GetEntityHistory handles GET /audit/entity/:type/:id
// @Summary Get audit history for an entity
// @Tags audit
// @Security BearerAuth
// @Produce json
// @Param type path string true "Entity type"
// @Param id path string true "Entity ID"
// @Param limit query int false "Limit" default(50)
// @Success 200 {array} AuditLog
// @Router /audit/entity/{type}/{id} [get]
func (h *Handler) GetEntityHistory(c *gin.Context) {
	entityType := c.Param("type")
	if entityType == "" {
		response.BadRequest(c, "Entity type is required")
		return
	}

	entityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid entity ID")
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		// Parse limit if provided
	}

	logs, err := h.service.GetEntityHistory(c.Request.Context(), entityType, entityID, limit)
	if err != nil {
		response.InternalServerError(c, "Failed to get entity history")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"logs":       logs,
		"entityType": entityType,
		"entityId":   entityID,
	})
}

// GetUserActivity handles GET /audit/user/:userId
// @Summary Get audit activity for a user
// @Tags audit
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID"
// @Param limit query int false "Limit" default(50)
// @Success 200 {array} AuditLog
// @Router /audit/user/{userId} [get]
func (h *Handler) GetUserActivity(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		// Parse limit if provided
	}

	logs, err := h.service.GetUserActivity(c.Request.Context(), userID, limit)
	if err != nil {
		response.InternalServerError(c, "Failed to get user activity")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"logs":   logs,
		"userId": userID,
	})
}
