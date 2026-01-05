package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	logs map[uuid.UUID]*AuditLog
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		logs: make(map[uuid.UUID]*AuditLog),
	}
}

func (m *MockRepository) Create(ctx context.Context, log *AuditLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	m.logs[log.ID] = log
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*AuditLog, error) {
	log, ok := m.logs[id]
	if !ok {
		return nil, nil
	}
	return log, nil
}

func (m *MockRepository) List(ctx context.Context, params AuditLogListParams) (*AuditLogListResponse, error) {
	logs := make([]AuditLog, 0)
	for _, log := range m.logs {
		// Apply basic filtering
		if params.Action != "" && log.Action != params.Action {
			continue
		}
		if params.EntityType != "" && log.EntityType != params.EntityType {
			continue
		}
		logs = append(logs, *log)
	}

	// Apply pagination
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 50
	}

	totalCount := len(logs)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > totalCount {
		logs = []AuditLog{}
	} else if end > totalCount {
		logs = logs[start:]
	} else {
		logs = logs[start:end]
	}

	return &AuditLogListResponse{
		Logs:       logs,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (totalCount + pageSize - 1) / pageSize,
	}, nil
}

func (m *MockRepository) GetByEntityID(ctx context.Context, entityType string, entityID uuid.UUID, limit int) ([]AuditLog, error) {
	logs := make([]AuditLog, 0)
	for _, log := range m.logs {
		if log.EntityType == entityType && log.EntityID != nil && *log.EntityID == entityID {
			logs = append(logs, *log)
		}
	}
	if limit > 0 && len(logs) > limit {
		logs = logs[:limit]
	}
	return logs, nil
}

func (m *MockRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]AuditLog, error) {
	logs := make([]AuditLog, 0)
	for _, log := range m.logs {
		if log.UserID != nil && *log.UserID == userID {
			logs = append(logs, *log)
		}
	}
	if limit > 0 && len(logs) > limit {
		logs = logs[:limit]
	}
	return logs, nil
}

func (m *MockRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	deleted := int64(0)
	for id, log := range m.logs {
		if log.CreatedAt.Before(before) {
			delete(m.logs, id)
			deleted++
		}
	}
	return deleted, nil
}

func newTestService() (*Service, *MockRepository) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	service := NewService(repo, logger)
	return service, repo
}

func TestService_Log(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	entityID := uuid.New()

	req := &CreateAuditLogRequest{
		UserID:      &userID,
		Action:      ActionUserLogin,
		EntityType:  "user",
		EntityID:    &entityID,
		Description: "Test login",
		IPAddress:   "192.168.1.1",
		UserAgent:   "Mozilla/5.0",
	}

	err := service.Log(ctx, req)
	require.NoError(t, err)

	// Verify log was created
	assert.Len(t, repo.logs, 1)
	for _, log := range repo.logs {
		assert.Equal(t, ActionUserLogin, log.Action)
		assert.Equal(t, "user", log.EntityType)
		assert.Equal(t, "Test login", log.Description)
	}
}

func TestService_LogAction(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	entityID := uuid.New()

	err := service.LogAction(ctx, &userID, ActionTournamentCreated, "tournament", &entityID, "Created tournament", "10.0.0.1", "Test Agent")
	require.NoError(t, err)
}

func TestService_LogWithValues(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	entityID := uuid.New()
	oldValue := map[string]interface{}{"name": "Old Name"}
	newValue := map[string]interface{}{"name": "New Name"}

	err := service.LogWithValues(ctx, &userID, ActionTournamentUpdated, "tournament", &entityID, oldValue, newValue, "10.0.0.1", "Test Agent")
	require.NoError(t, err)

	// Verify values were stored
	for _, log := range repo.logs {
		assert.Equal(t, "Old Name", log.OldValue["name"])
		assert.Equal(t, "New Name", log.NewValue["name"])
	}
}

func TestService_List(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	// Create multiple logs
	for i := 0; i < 15; i++ {
		userID := uuid.New()
		service.LogAction(ctx, &userID, ActionUserLogin, "user", &userID, "Login", "10.0.0.1", "Agent")
	}

	result, err := service.List(ctx, AuditLogListParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 15, result.TotalCount)
	assert.Equal(t, 10, len(result.Logs))
	assert.Equal(t, 2, result.TotalPages)
}

func TestService_GetEntityHistory(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	tournamentID := uuid.New()

	// Create logs for the same tournament
	service.LogAction(ctx, &userID, ActionTournamentCreated, "tournament", &tournamentID, "Created", "10.0.0.1", "Agent")
	service.LogAction(ctx, &userID, ActionTournamentUpdated, "tournament", &tournamentID, "Updated", "10.0.0.1", "Agent")
	service.LogAction(ctx, &userID, ActionTournamentStarted, "tournament", &tournamentID, "Started", "10.0.0.1", "Agent")

	// Create log for different entity
	otherID := uuid.New()
	service.LogAction(ctx, &userID, ActionTournamentCreated, "tournament", &otherID, "Other", "10.0.0.1", "Agent")

	logs, err := service.GetEntityHistory(ctx, "tournament", tournamentID, 10)
	require.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestService_GetUserActivity(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	otherUserID := uuid.New()

	// Create logs for the user
	entityID := uuid.New()
	service.LogAction(ctx, &userID, ActionUserLogin, "user", &userID, "Login", "10.0.0.1", "Agent")
	service.LogAction(ctx, &userID, ActionTournamentCreated, "tournament", &entityID, "Created", "10.0.0.1", "Agent")

	// Create log for different user
	service.LogAction(ctx, &otherUserID, ActionUserLogin, "user", &otherUserID, "Other login", "10.0.0.1", "Agent")

	logs, err := service.GetUserActivity(ctx, userID, 10)
	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestService_LogUserLogin(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	err := service.LogUserLogin(ctx, userID, "192.168.1.1", "Chrome")
	require.NoError(t, err)

	assert.Len(t, repo.logs, 1)
	for _, log := range repo.logs {
		assert.Equal(t, ActionUserLogin, log.Action)
	}
}

func TestService_LogUserLogout(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	err := service.LogUserLogout(ctx, userID, "192.168.1.1", "Chrome")
	require.NoError(t, err)

	assert.Len(t, repo.logs, 1)
	for _, log := range repo.logs {
		assert.Equal(t, ActionUserLogout, log.Action)
	}
}

func TestService_LogTournamentCreated(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	tournamentID := uuid.New()
	err := service.LogTournamentCreated(ctx, userID, tournamentID, "Test Tournament", "192.168.1.1", "Chrome")
	require.NoError(t, err)

	assert.Len(t, repo.logs, 1)
	for _, log := range repo.logs {
		assert.Equal(t, ActionTournamentCreated, log.Action)
		assert.Contains(t, log.Description, "Test Tournament")
	}
}

func TestService_LogGameResultEntered(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	gameID := uuid.New()
	err := service.LogGameResultEntered(ctx, userID, gameID, "1-0", "192.168.1.1", "Chrome")
	require.NoError(t, err)

	assert.Len(t, repo.logs, 1)
	for _, log := range repo.logs {
		assert.Equal(t, ActionGameResultEntered, log.Action)
		assert.Equal(t, "1-0", log.NewValue["result"])
	}
}

func TestService_LogGameResultModified(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	gameID := uuid.New()
	err := service.LogGameResultModified(ctx, userID, gameID, "1-0", "0-1", "192.168.1.1", "Chrome")
	require.NoError(t, err)

	assert.Len(t, repo.logs, 1)
	for _, log := range repo.logs {
		assert.Equal(t, ActionGameResultModified, log.Action)
		assert.Equal(t, "1-0", log.OldValue["result"])
		assert.Equal(t, "0-1", log.NewValue["result"])
	}
}
