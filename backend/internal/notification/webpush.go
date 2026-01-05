package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// WebPushConfig holds Web Push configuration
type WebPushConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	Subscriber      string // Contact email for VAPID
}

// PushSubscription represents a user's push subscription
type PushSubscription struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	Endpoint  string    `json:"endpoint" db:"endpoint"`
	P256dh    string    `json:"p256dh" db:"p256dh"`
	Auth      string    `json:"auth" db:"auth"`
	UserAgent string    `json:"userAgent,omitempty" db:"user_agent"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// PushMessage represents a push notification message
type PushMessage struct {
	Title   string                 `json:"title"`
	Body    string                 `json:"body"`
	Icon    string                 `json:"icon,omitempty"`
	Badge   string                 `json:"badge,omitempty"`
	Tag     string                 `json:"tag,omitempty"`
	URL     string                 `json:"url,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Actions []PushAction           `json:"actions,omitempty"`
}

// PushAction represents an action button in a push notification
type PushAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// PushSubscriptionRepository handles push subscription persistence
type PushSubscriptionRepository interface {
	Create(ctx context.Context, sub *PushSubscription) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]PushSubscription, error)
	GetByEndpoint(ctx context.Context, endpoint string) (*PushSubscription, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByEndpoint(ctx context.Context, endpoint string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

// PostgresPushSubscriptionRepository implements PushSubscriptionRepository
type PostgresPushSubscriptionRepository struct {
	db *pgxpool.Pool
}

// NewPushSubscriptionRepository creates a new push subscription repository
func NewPushSubscriptionRepository(db *pgxpool.Pool) PushSubscriptionRepository {
	return &PostgresPushSubscriptionRepository{db: db}
}

func (r *PostgresPushSubscriptionRepository) Create(ctx context.Context, sub *PushSubscription) error {
	sub.ID = uuid.New()
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()

	query := `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, user_agent, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth,
			user_agent = EXCLUDED.user_agent,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		sub.ID, sub.UserID, sub.Endpoint, sub.P256dh, sub.Auth, sub.UserAgent, sub.CreatedAt, sub.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create push subscription: %w", err)
	}

	return nil
}

func (r *PostgresPushSubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]PushSubscription, error) {
	query := `
		SELECT id, user_id, endpoint, p256dh, auth, user_agent, created_at, updated_at
		FROM push_subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query push subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var sub PushSubscription
		err := rows.Scan(&sub.ID, &sub.UserID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.UserAgent, &sub.CreatedAt, &sub.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan push subscription: %w", err)
		}
		subs = append(subs, sub)
	}

	return subs, nil
}

func (r *PostgresPushSubscriptionRepository) GetByEndpoint(ctx context.Context, endpoint string) (*PushSubscription, error) {
	query := `
		SELECT id, user_id, endpoint, p256dh, auth, user_agent, created_at, updated_at
		FROM push_subscriptions
		WHERE endpoint = $1`

	var sub PushSubscription
	err := r.db.QueryRow(ctx, query, endpoint).Scan(
		&sub.ID, &sub.UserID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.UserAgent, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("push subscription not found")
		}
		return nil, fmt.Errorf("failed to get push subscription: %w", err)
	}

	return &sub, nil
}

func (r *PostgresPushSubscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM push_subscriptions WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete push subscription: %w", err)
	}
	return nil
}

func (r *PostgresPushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	query := `DELETE FROM push_subscriptions WHERE endpoint = $1`
	_, err := r.db.Exec(ctx, query, endpoint)
	if err != nil {
		return fmt.Errorf("failed to delete push subscription by endpoint: %w", err)
	}
	return nil
}

func (r *PostgresPushSubscriptionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM push_subscriptions WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete push subscriptions by user: %w", err)
	}
	return nil
}

// WebPushService handles web push notifications
type WebPushService struct {
	config     *WebPushConfig
	repo       PushSubscriptionRepository
	logger     zerolog.Logger
	appBaseURL string
}

// NewWebPushService creates a new web push service
func NewWebPushService(config *WebPushConfig, repo PushSubscriptionRepository, logger zerolog.Logger, appBaseURL string) *WebPushService {
	return &WebPushService{
		config:     config,
		repo:       repo,
		logger:     logger.With().Str("component", "webpush_service").Logger(),
		appBaseURL: appBaseURL,
	}
}

// GetVAPIDPublicKey returns the VAPID public key for clients
func (s *WebPushService) GetVAPIDPublicKey() string {
	return s.config.VAPIDPublicKey
}

// Subscribe registers a new push subscription
func (s *WebPushService) Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth, userAgent string) error {
	sub := &PushSubscription{
		UserID:    userID,
		Endpoint:  endpoint,
		P256dh:    p256dh,
		Auth:      auth,
		UserAgent: userAgent,
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	s.logger.Info().
		Str("userId", userID.String()).
		Str("endpoint", endpoint[:min(50, len(endpoint))]+"...").
		Msg("Push subscription created")

	return nil
}

// Unsubscribe removes a push subscription
func (s *WebPushService) Unsubscribe(ctx context.Context, endpoint string) error {
	if err := s.repo.DeleteByEndpoint(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	s.logger.Info().
		Str("endpoint", endpoint[:min(50, len(endpoint))]+"...").
		Msg("Push subscription removed")

	return nil
}

// SendToUser sends a push notification to all of a user's subscribed devices
func (s *WebPushService) SendToUser(ctx context.Context, userID uuid.UUID, message *PushMessage) error {
	subs, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if len(subs) == 0 {
		s.logger.Debug().Str("userId", userID.String()).Msg("No push subscriptions for user")
		return nil
	}

	// Set default icon if not provided
	if message.Icon == "" {
		message.Icon = s.appBaseURL + "/icons/icon-192.png"
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	var sendErrors []error
	for _, sub := range subs {
		if err := s.sendToSubscription(ctx, &sub, payload); err != nil {
			sendErrors = append(sendErrors, err)
			// If subscription is gone (410), remove it
			if isSubscriptionExpired(err) {
				s.repo.DeleteByEndpoint(ctx, sub.Endpoint)
				s.logger.Info().Str("endpoint", sub.Endpoint[:min(50, len(sub.Endpoint))]+"...").Msg("Removed expired subscription")
			}
		}
	}

	if len(sendErrors) > 0 && len(sendErrors) == len(subs) {
		return fmt.Errorf("all push notifications failed")
	}

	return nil
}

// SendToMany sends a push notification to multiple users
func (s *WebPushService) SendToMany(ctx context.Context, userIDs []uuid.UUID, message *PushMessage) error {
	for _, userID := range userIDs {
		if err := s.SendToUser(ctx, userID, message); err != nil {
			s.logger.Warn().Err(err).Str("userId", userID.String()).Msg("Failed to send push to user")
			// Continue sending to others
		}
	}
	return nil
}

func (s *WebPushService) sendToSubscription(ctx context.Context, sub *PushSubscription, payload []byte) error {
	subscription := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	options := &webpush.Options{
		Subscriber:      s.config.Subscriber,
		VAPIDPublicKey:  s.config.VAPIDPublicKey,
		VAPIDPrivateKey: s.config.VAPIDPrivateKey,
		TTL:             3600, // 1 hour
	}

	resp, err := webpush.SendNotification(payload, subscription, options)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &PushError{StatusCode: resp.StatusCode, Endpoint: sub.Endpoint}
	}

	s.logger.Debug().
		Str("endpoint", sub.Endpoint[:min(50, len(sub.Endpoint))]+"...").
		Int("status", resp.StatusCode).
		Msg("Push notification sent")

	return nil
}

// PushError represents a push notification error
type PushError struct {
	StatusCode int
	Endpoint   string
}

func (e *PushError) Error() string {
	return fmt.Sprintf("push failed with status %d for endpoint %s", e.StatusCode, e.Endpoint[:min(50, len(e.Endpoint))]+"...")
}

func isSubscriptionExpired(err error) bool {
	if pushErr, ok := err.(*PushError); ok {
		return pushErr.StatusCode == 404 || pushErr.StatusCode == 410
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
