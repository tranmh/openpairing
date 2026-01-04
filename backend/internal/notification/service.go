package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// NotificationType defines the type of notification
type NotificationType string

const (
	// Tournament notifications
	TypeTournamentCreated     NotificationType = "tournament_created"
	TypeTournamentUpdated     NotificationType = "tournament_updated"
	TypeTournamentCancelled   NotificationType = "tournament_cancelled"
	TypeRegistrationOpen      NotificationType = "registration_open"
	TypeRegistrationClosed    NotificationType = "registration_closed"

	// Pairing notifications
	TypePairingPublished      NotificationType = "pairing_published"
	TypeRoundStarting         NotificationType = "round_starting"
	TypeRoundCompleted        NotificationType = "round_completed"

	// Game notifications
	TypeGameStarting          NotificationType = "game_starting"
	TypeGameResult            NotificationType = "game_result"
	TypeOpponentAssigned      NotificationType = "opponent_assigned"

	// Team notifications
	TypeLineupDeadline        NotificationType = "lineup_deadline"
	TypeLineupSubmitted       NotificationType = "lineup_submitted"
	TypeTeamMatchScheduled    NotificationType = "team_match_scheduled"

	// Administrative notifications
	TypeArbiterMessage        NotificationType = "arbiter_message"
	TypeSystemAnnouncement    NotificationType = "system_announcement"
)

// Notification represents a notification to a user
type Notification struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	UserID      uuid.UUID              `json:"userId" db:"user_id"`
	Type        NotificationType       `json:"type" db:"type"`
	Title       string                 `json:"title" db:"title"`
	Message     string                 `json:"message" db:"message"`
	Data        map[string]interface{} `json:"data,omitempty" db:"data"`
	ReadAt      *time.Time             `json:"readAt,omitempty" db:"read_at"`
	CreatedAt   time.Time              `json:"createdAt" db:"created_at"`
}

// Repository handles notification persistence
type Repository interface {
	Create(ctx context.Context, notification *Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]Notification, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new notification repository
func NewRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, n *Notification) error {
	n.ID = uuid.New()
	n.CreatedAt = time.Now()

	dataJSON, err := json.Marshal(n.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
		INSERT INTO notifications (id, user_id, type, title, message, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = r.db.Exec(ctx, query,
		n.ID, n.UserID, n.Type, n.Title, n.Message, dataJSON, n.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, data, read_at, created_at
		FROM notifications
		WHERE id = $1`

	var n Notification
	var dataJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &dataJSON, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &n.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return &n, nil
}

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, data, read_at, created_at
		FROM notifications
		WHERE user_id = $1`

	if unreadOnly {
		query += " AND read_at IS NULL"
	}

	query += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		var dataJSON []byte
		err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &dataJSON, &n.ReadAt, &n.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		if len(dataJSON) > 0 {
			json.Unmarshal(dataJSON, &n.Data)
		}

		notifications = append(notifications, n)
	}

	return notifications, nil
}

func (r *PostgresRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread: %w", err)
	}

	return count, nil
}

func (r *PostgresRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE notifications SET read_at = $1 WHERE id = $2 AND read_at IS NULL`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE notifications SET read_at = $1 WHERE user_id = $2 AND read_at IS NULL`
	_, err := r.db.Exec(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to mark all as read: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM notifications WHERE created_at < $1`
	result, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old notifications: %w", err)
	}
	return result.RowsAffected(), nil
}

// Service handles notification business logic
type Service struct {
	repo   Repository
	logger zerolog.Logger

	// Channels for real-time delivery
	subscribers map[uuid.UUID][]chan Notification
	subMu       sync.RWMutex
}

// NewService creates a new notification service
func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:        repo,
		logger:      logger.With().Str("component", "notification_service").Logger(),
		subscribers: make(map[uuid.UUID][]chan Notification),
	}
}

// Send creates and delivers a notification
func (s *Service) Send(ctx context.Context, userID uuid.UUID, notifType NotificationType, title, message string, data map[string]interface{}) error {
	notification := &Notification{
		UserID:  userID,
		Type:    notifType,
		Title:   title,
		Message: message,
		Data:    data,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Deliver to real-time subscribers
	s.deliverToSubscribers(userID, *notification)

	s.logger.Debug().
		Str("userId", userID.String()).
		Str("type", string(notifType)).
		Msg("Notification sent")

	return nil
}

// SendToMany sends a notification to multiple users
func (s *Service) SendToMany(ctx context.Context, userIDs []uuid.UUID, notifType NotificationType, title, message string, data map[string]interface{}) error {
	for _, userID := range userIDs {
		if err := s.Send(ctx, userID, notifType, title, message, data); err != nil {
			s.logger.Warn().Err(err).Str("userId", userID.String()).Msg("Failed to send notification")
			// Continue sending to others
		}
	}
	return nil
}

// GetUserNotifications retrieves notifications for a user
func (s *Service) GetUserNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]Notification, error) {
	return s.repo.GetByUserID(ctx, userID, unreadOnly, limit, offset)
}

// GetUnreadCount returns the count of unread notifications
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	return s.repo.MarkAsRead(ctx, notificationID)
}

// MarkAsReadForUser marks a notification as read with ownership verification
func (s *Service) MarkAsReadForUser(ctx context.Context, notificationID, userID uuid.UUID) error {
	// Verify ownership
	notification, err := s.repo.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("notification not found: %w", err)
	}
	if notification.UserID != userID {
		return fmt.Errorf("notification does not belong to user")
	}
	return s.repo.MarkAsRead(ctx, notificationID)
}

// DeleteForUser deletes a notification with ownership verification
func (s *Service) DeleteForUser(ctx context.Context, notificationID, userID uuid.UUID) error {
	// Verify ownership
	notification, err := s.repo.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("notification not found: %w", err)
	}
	if notification.UserID != userID {
		return fmt.Errorf("notification does not belong to user")
	}
	return s.repo.Delete(ctx, notificationID)
}

// MarkAllAsRead marks all notifications as read for a user
func (s *Service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

// Subscribe allows a client to receive real-time notifications
func (s *Service) Subscribe(userID uuid.UUID) <-chan Notification {
	ch := make(chan Notification, 100)
	s.subMu.Lock()
	s.subscribers[userID] = append(s.subscribers[userID], ch)
	s.subMu.Unlock()
	return ch
}

// Unsubscribe removes a subscription
func (s *Service) Unsubscribe(userID uuid.UUID, ch <-chan Notification) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	subs := s.subscribers[userID]
	for i, sub := range subs {
		if sub == ch {
			// Remove from slice
			s.subscribers[userID] = append(subs[:i], subs[i+1:]...)
			// Close the channel (safe because we created it as bidirectional internally)
			close(sub)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(s.subscribers[userID]) == 0 {
		delete(s.subscribers, userID)
	}
}

// deliverToSubscribers sends notification to real-time subscribers
func (s *Service) deliverToSubscribers(userID uuid.UUID, n Notification) {
	s.subMu.RLock()
	subs := s.subscribers[userID]
	s.subMu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- n:
		default:
			// Channel full, skip
			s.logger.Warn().Str("userId", userID.String()).Msg("Notification channel full, dropping notification")
		}
	}
}

// Cleanup removes old notifications
func (s *Service) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	before := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := s.repo.DeleteOlderThan(ctx, before)
	if err != nil {
		return 0, err
	}

	s.logger.Info().Int64("deleted", deleted).Int("retentionDays", retentionDays).Msg("Cleaned up old notifications")
	return deleted, nil
}

// Tournament notification helpers

// NotifyPairingPublished sends pairing notifications to participants
func (s *Service) NotifyPairingPublished(ctx context.Context, tournamentID uuid.UUID, tournamentName string, round int, participantUserIDs []uuid.UUID) error {
	data := map[string]interface{}{
		"tournamentId":   tournamentID.String(),
		"tournamentName": tournamentName,
		"round":          round,
	}

	return s.SendToMany(ctx, participantUserIDs, TypePairingPublished,
		fmt.Sprintf("Round %d Pairings Published", round),
		fmt.Sprintf("Pairings for round %d of %s have been published.", round, tournamentName),
		data)
}

// NotifyGameStarting sends game start notifications
func (s *Service) NotifyGameStarting(ctx context.Context, gameID, tournamentID uuid.UUID, round int, whiteUserID, blackUserID uuid.UUID, startTime time.Time) error {
	data := map[string]interface{}{
		"gameId":       gameID.String(),
		"tournamentId": tournamentID.String(),
		"round":        round,
		"startTime":    startTime.Format(time.RFC3339),
	}

	userIDs := []uuid.UUID{}
	if whiteUserID != uuid.Nil {
		userIDs = append(userIDs, whiteUserID)
	}
	if blackUserID != uuid.Nil {
		userIDs = append(userIDs, blackUserID)
	}

	return s.SendToMany(ctx, userIDs, TypeGameStarting,
		"Your Game is Starting",
		fmt.Sprintf("Your game in round %d is starting at %s.", round, startTime.Format("15:04")),
		data)
}

// NotifyGameResult sends result notifications
func (s *Service) NotifyGameResult(ctx context.Context, gameID, tournamentID uuid.UUID, round int, userID uuid.UUID, result string) error {
	data := map[string]interface{}{
		"gameId":       gameID.String(),
		"tournamentId": tournamentID.String(),
		"round":        round,
		"result":       result,
	}

	return s.Send(ctx, userID, TypeGameResult,
		"Game Result Entered",
		fmt.Sprintf("Result for your game in round %d: %s", round, result),
		data)
}

// NotifyLineupDeadline sends lineup deadline reminders
func (s *Service) NotifyLineupDeadline(ctx context.Context, tournamentID uuid.UUID, tournamentName string, round int, deadline time.Time, captainUserID uuid.UUID) error {
	data := map[string]interface{}{
		"tournamentId":   tournamentID.String(),
		"tournamentName": tournamentName,
		"round":          round,
		"deadline":       deadline.Format(time.RFC3339),
	}

	return s.Send(ctx, captainUserID, TypeLineupDeadline,
		"Lineup Submission Deadline",
		fmt.Sprintf("Lineup for round %d of %s must be submitted by %s.", round, tournamentName, deadline.Format("15:04 on Jan 2")),
		data)
}
