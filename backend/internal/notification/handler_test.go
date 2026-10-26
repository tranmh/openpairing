package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// recordingRepo is an in-memory Repository that records which users were queried
type recordingRepo struct {
	mu      sync.Mutex
	queried []uuid.UUID
}

func (r *recordingRepo) record(userID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queried = append(r.queried, userID)
}

func (r *recordingRepo) users() []uuid.UUID {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]uuid.UUID(nil), r.queried...)
}

func (r *recordingRepo) Create(ctx context.Context, n *Notification) error { return nil }
func (r *recordingRepo) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	return &Notification{ID: id}, nil
}
func (r *recordingRepo) GetByUserID(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]Notification, error) {
	r.record(userID)
	return []Notification{}, nil
}
func (r *recordingRepo) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	r.record(userID)
	return 0, nil
}
func (r *recordingRepo) MarkAsRead(ctx context.Context, id uuid.UUID) error { return nil }
func (r *recordingRepo) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	r.record(userID)
	return nil
}
func (r *recordingRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *recordingRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

// streamRecorder adds the CloseNotifier gin's SSE streaming needs to httptest.ResponseRecorder
type streamRecorder struct {
	*httptest.ResponseRecorder
}

func (r *streamRecorder) CloseNotify() <-chan bool {
	return make(chan bool)
}

func newNotificationTestRouter(repo *recordingRepo, authenticatedUser string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(repo, zerolog.Nop()), zerolog.Nop())
	r := gin.New()
	if authenticatedUser != "" {
		// Same context key the auth middleware sets after validating a token
		r.Use(func(c *gin.Context) {
			c.Set("user_id", authenticatedUser)
			c.Next()
		})
	}
	r.GET("/notifications", h.GetNotifications)
	r.GET("/notifications/unread-count", h.GetUnreadCount)
	r.PUT("/notifications/read-all", h.MarkAllAsRead)
	r.GET("/notifications/stream", h.StreamNotifications)
	return r
}

func TestNotificationsRejectUnauthenticatedUserIDHeader(t *testing.T) {
	victim := uuid.New()

	requests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/notifications"},
		{http.MethodGet, "/notifications/unread-count"},
		{http.MethodPut, "/notifications/read-all"},
		{http.MethodGet, "/notifications/stream"},
	}

	for _, tc := range requests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			repo := &recordingRepo{}
			router := newNotificationTestRouter(repo, "")

			// A cancelled request context ends the SSE stream instead of blocking
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			req := httptest.NewRequest(tc.method, tc.path, nil).WithContext(ctx)
			req.Header.Set("X-User-ID", victim.String())
			w := &streamRecorder{httptest.NewRecorder()}

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Empty(t, repo.users(), "no notification data may be accessed without authentication")
		})
	}
}

func TestNotificationsUseAuthenticatedUserNotHeader(t *testing.T) {
	attacker := uuid.New()
	victim := uuid.New()

	repo := &recordingRepo{}
	router := newNotificationTestRouter(repo, attacker.String())

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.Header.Set("X-User-ID", victim.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, repo.users(), victim, "the X-User-ID header must not override the authenticated user")
	assert.Contains(t, repo.users(), attacker)
}
