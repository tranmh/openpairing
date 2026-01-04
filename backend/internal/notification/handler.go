package notification

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpairing/openpairing/pkg/response"
	"github.com/rs/zerolog"
)

// Handler handles notification-related HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new notification handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("component", "notification_handler").Logger(),
	}
}

// RegisterRoutes registers the notification routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	notifications := r.Group("/notifications")
	{
		notifications.GET("", h.GetNotifications)
		notifications.GET("/unread-count", h.GetUnreadCount)
		notifications.PUT("/:id/read", h.MarkAsRead)
		notifications.PUT("/read-all", h.MarkAllAsRead)
		notifications.DELETE("/:id", h.DeleteNotification)
		notifications.GET("/stream", h.StreamNotifications)
	}
}

// GetNotifications returns notifications for the authenticated user
// @Summary Get user notifications
// @Tags Notifications
// @Param unreadOnly query bool false "Only return unread notifications"
// @Param limit query int false "Limit (default 20)"
// @Param offset query int false "Offset"
// @Success 200 {array} Notification
// @Router /notifications [get]
func (h *Handler) GetNotifications(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	unreadOnly := c.Query("unreadOnly") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	notifications, err := h.service.GetUserNotifications(c.Request.Context(), userID, unreadOnly, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("userId", userID.String()).Msg("Failed to get notifications")
		response.InternalServerError(c, "Failed to get notifications")
		return
	}

	unreadCount, _ := h.service.GetUnreadCount(c.Request.Context(), userID)

	response.Success(c, http.StatusOK, gin.H{
		"notifications": notifications,
		"unreadCount":   unreadCount,
		"limit":         limit,
		"offset":        offset,
	})
}

// GetUnreadCount returns the count of unread notifications
// @Summary Get unread notification count
// @Tags Notifications
// @Success 200 {object} map[string]int
// @Router /notifications/unread-count [get]
func (h *Handler) GetUnreadCount(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	count, err := h.service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("userId", userID.String()).Msg("Failed to get unread count")
		response.InternalServerError(c, "Failed to get unread count")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"count": count,
	})
}

// MarkAsRead marks a notification as read
// @Summary Mark notification as read
// @Tags Notifications
// @Param id path string true "Notification ID"
// @Success 204
// @Router /notifications/{id}/read [put]
func (h *Handler) MarkAsRead(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	// Verify ownership before marking as read
	if err := h.service.MarkAsReadForUser(c.Request.Context(), notificationID, userID); err != nil {
		h.logger.Error().Err(err).Str("notificationId", notificationID.String()).Msg("Failed to mark as read")
		response.InternalServerError(c, "Failed to mark notification as read")
		return
	}

	response.NoContent(c)
}

// MarkAllAsRead marks all notifications as read for the user
// @Summary Mark all notifications as read
// @Tags Notifications
// @Success 204
// @Router /notifications/read-all [put]
func (h *Handler) MarkAllAsRead(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.service.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		h.logger.Error().Err(err).Str("userId", userID.String()).Msg("Failed to mark all as read")
		response.InternalServerError(c, "Failed to mark notifications as read")
		return
	}

	response.NoContent(c)
}

// DeleteNotification deletes a notification
// @Summary Delete notification
// @Tags Notifications
// @Param id path string true "Notification ID"
// @Success 204
// @Router /notifications/{id} [delete]
func (h *Handler) DeleteNotification(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	// Verify ownership before deleting
	if err := h.service.DeleteForUser(c.Request.Context(), notificationID, userID); err != nil {
		h.logger.Error().Err(err).Str("notificationId", notificationID.String()).Msg("Failed to delete notification")
		response.InternalServerError(c, "Failed to delete notification")
		return
	}

	response.NoContent(c)
}

// StreamNotifications streams notifications via Server-Sent Events (SSE)
// @Summary Stream notifications (SSE)
// @Tags Notifications
// @Produce text/event-stream
// @Success 200 {object} Notification
// @Router /notifications/stream [get]
func (h *Handler) StreamNotifications(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Subscribe to notifications
	ch := h.service.Subscribe(userID)
	defer h.service.Unsubscribe(userID, ch)

	// Client gone detection
	clientGone := c.Request.Context().Done()

	// Stream notifications
	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case notification, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("notification", notification)
			return true
		}
	})
}

// SendNotificationRequest represents a request to send a notification
type SendNotificationRequest struct {
	UserIDs []uuid.UUID            `json:"userIds" binding:"required"`
	Type    NotificationType       `json:"type" binding:"required"`
	Title   string                 `json:"title" binding:"required"`
	Message string                 `json:"message" binding:"required"`
	Data    map[string]interface{} `json:"data"`
}

// SendNotification sends a notification to specified users (admin only)
// @Summary Send notification (admin)
// @Tags Notifications
// @Param request body SendNotificationRequest true "Notification request"
// @Success 204
// @Router /admin/notifications/send [post]
func (h *Handler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	if err := h.service.SendToMany(c.Request.Context(), req.UserIDs, req.Type, req.Title, req.Message, req.Data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to send notifications")
		response.InternalServerError(c, "Failed to send notifications")
		return
	}

	response.NoContent(c)
}

// getUserIDFromContext extracts user ID from the Gin context
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	// Try to get from context (set by auth middleware)
	if userID, exists := c.Get("userID"); exists {
		if id, ok := userID.(uuid.UUID); ok {
			return id, nil
		}
		if idStr, ok := userID.(string); ok {
			return uuid.Parse(idStr)
		}
	}

	// Fallback: try to get from header (for testing)
	if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
		return uuid.Parse(userIDStr)
	}

	return uuid.Nil, http.ErrNoCookie
}
