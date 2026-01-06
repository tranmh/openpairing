package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tranmh/openpairing/pkg/response"
	"github.com/rs/zerolog"
)

// WebPushHandler handles web push notification HTTP requests
type WebPushHandler struct {
	service *WebPushService
	logger  zerolog.Logger
}

// NewWebPushHandler creates a new web push handler
func NewWebPushHandler(service *WebPushService, logger zerolog.Logger) *WebPushHandler {
	return &WebPushHandler{
		service: service,
		logger:  logger.With().Str("component", "webpush_handler").Logger(),
	}
}

// RegisterRoutes registers web push routes
func (h *WebPushHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	push := r.Group("/push")
	{
		// Public route for VAPID key
		push.GET("/vapid-key", h.GetVAPIDKey)

		// Protected routes
		protected := push.Group("")
		protected.Use(authMiddleware)
		{
			protected.POST("/subscribe", h.Subscribe)
			protected.POST("/unsubscribe", h.Unsubscribe)
			protected.POST("/test", h.SendTestNotification)
		}
	}
}

// GetVAPIDKey returns the VAPID public key
// @Summary Get VAPID public key
// @Tags Push Notifications
// @Produce json
// @Success 200 {object} map[string]string
// @Router /push/vapid-key [get]
func (h *WebPushHandler) GetVAPIDKey(c *gin.Context) {
	publicKey := h.service.GetVAPIDPublicKey()
	if publicKey == "" {
		response.InternalServerError(c, "Web push not configured")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"publicKey": publicKey,
	})
}

// SubscribeRequest represents a push subscription request
type SubscribeRequest struct {
	Endpoint  string `json:"endpoint" binding:"required"`
	P256dh    string `json:"p256dh" binding:"required"`
	Auth      string `json:"auth" binding:"required"`
	UserAgent string `json:"userAgent"`
}

// Subscribe registers a push subscription for the authenticated user
// @Summary Subscribe to push notifications
// @Tags Push Notifications
// @Accept json
// @Produce json
// @Param request body SubscribeRequest true "Subscription data"
// @Success 200 {object} map[string]string
// @Router /push/subscribe [post]
func (h *WebPushHandler) Subscribe(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	if err := h.service.Subscribe(c.Request.Context(), userID, req.Endpoint, req.P256dh, req.Auth, req.UserAgent); err != nil {
		h.logger.Error().Err(err).Str("userId", userID.String()).Msg("Failed to subscribe")
		response.InternalServerError(c, "Failed to subscribe to push notifications")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Successfully subscribed to push notifications",
	})
}

// UnsubscribeRequest represents a push unsubscription request
type UnsubscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
}

// Unsubscribe removes a push subscription
// @Summary Unsubscribe from push notifications
// @Tags Push Notifications
// @Accept json
// @Produce json
// @Param request body UnsubscribeRequest true "Unsubscription data"
// @Success 200 {object} map[string]string
// @Router /push/unsubscribe [post]
func (h *WebPushHandler) Unsubscribe(c *gin.Context) {
	_, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UnsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	if err := h.service.Unsubscribe(c.Request.Context(), req.Endpoint); err != nil {
		h.logger.Error().Err(err).Msg("Failed to unsubscribe")
		response.InternalServerError(c, "Failed to unsubscribe from push notifications")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Successfully unsubscribed from push notifications",
	})
}

// SendTestNotification sends a test push notification to the user
// @Summary Send test push notification
// @Tags Push Notifications
// @Success 200 {object} map[string]string
// @Router /push/test [post]
func (h *WebPushHandler) SendTestNotification(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	message := &PushMessage{
		Title: "Test Notification",
		Body:  "This is a test push notification from OpenPairing.",
		Tag:   "test",
		URL:   "/",
		Data: map[string]interface{}{
			"type": "test",
		},
	}

	if err := h.service.SendToUser(c.Request.Context(), userID, message); err != nil {
		h.logger.Error().Err(err).Str("userId", userID.String()).Msg("Failed to send test notification")
		response.InternalServerError(c, "Failed to send test notification")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Test notification sent",
	})
}

// AdminSendPushRequest represents an admin request to send push notifications
type AdminSendPushRequest struct {
	UserIDs []uuid.UUID `json:"userIds" binding:"required"`
	Title   string      `json:"title" binding:"required"`
	Body    string      `json:"body" binding:"required"`
	URL     string      `json:"url"`
	Tag     string      `json:"tag"`
}

// AdminSendPush sends push notifications to specified users (admin only)
// @Summary Send push notification to users (admin)
// @Tags Push Notifications
// @Accept json
// @Produce json
// @Param request body AdminSendPushRequest true "Push notification data"
// @Success 200 {object} map[string]string
// @Router /admin/push/send [post]
func (h *WebPushHandler) AdminSendPush(c *gin.Context) {
	var req AdminSendPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	message := &PushMessage{
		Title: req.Title,
		Body:  req.Body,
		URL:   req.URL,
		Tag:   req.Tag,
	}

	if err := h.service.SendToMany(c.Request.Context(), req.UserIDs, message); err != nil {
		h.logger.Error().Err(err).Msg("Failed to send push notifications")
		response.InternalServerError(c, "Failed to send push notifications")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Push notifications sent",
	})
}
