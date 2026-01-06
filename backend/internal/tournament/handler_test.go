package tournament

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tranmh/openpairing/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TournamentService defines the interface for tournament operations used by the handler.
type TournamentService interface {
	CreateTournament(ctx context.Context, req *CreateTournamentRequest, organizerID uuid.UUID) (*Tournament, error)
	GetTournament(ctx context.Context, id uuid.UUID) (*Tournament, error)
	GetTournamentBySlug(ctx context.Context, slug string) (*Tournament, error)
	UpdateTournament(ctx context.Context, id uuid.UUID, req *UpdateTournamentRequest) (*Tournament, error)
	DeleteTournament(ctx context.Context, id uuid.UUID) error
	ListTournaments(ctx context.Context, filter *ListFilters) ([]Tournament, int64, error)
	PublishTournament(ctx context.Context, id uuid.UUID) (*Tournament, error)
	StartTournament(ctx context.Context, id uuid.UUID) (*Tournament, error)
	GetParticipants(ctx context.Context, tournamentID uuid.UUID, filter *ParticipantFilter) ([]Participant, error)
	GetRounds(ctx context.Context, tournamentID uuid.UUID) ([]Round, error)
	GetPairings(ctx context.Context, tournamentID uuid.UUID, round int) ([]Game, error)
}

// MockTournamentService is a mock implementation of TournamentService.
type MockTournamentService struct {
	CreateTournamentFunc   func(ctx context.Context, req *CreateTournamentRequest, organizerID uuid.UUID) (*Tournament, error)
	GetTournamentFunc      func(ctx context.Context, id uuid.UUID) (*Tournament, error)
	GetTournamentBySlugFunc func(ctx context.Context, slug string) (*Tournament, error)
	UpdateTournamentFunc   func(ctx context.Context, id uuid.UUID, req *UpdateTournamentRequest) (*Tournament, error)
	DeleteTournamentFunc   func(ctx context.Context, id uuid.UUID) error
	ListTournamentsFunc    func(ctx context.Context, filter *ListFilters) ([]Tournament, int64, error)
	PublishTournamentFunc  func(ctx context.Context, id uuid.UUID) (*Tournament, error)
	StartTournamentFunc    func(ctx context.Context, id uuid.UUID) (*Tournament, error)
	GetParticipantsFunc    func(ctx context.Context, tournamentID uuid.UUID, filter *ParticipantFilter) ([]Participant, error)
	GetRoundsFunc          func(ctx context.Context, tournamentID uuid.UUID) ([]Round, error)
	GetPairingsFunc        func(ctx context.Context, tournamentID uuid.UUID, round int) ([]Game, error)
}

func (m *MockTournamentService) CreateTournament(ctx context.Context, req *CreateTournamentRequest, organizerID uuid.UUID) (*Tournament, error) {
	if m.CreateTournamentFunc != nil {
		return m.CreateTournamentFunc(ctx, req, organizerID)
	}
	return nil, nil
}

func (m *MockTournamentService) GetTournament(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	if m.GetTournamentFunc != nil {
		return m.GetTournamentFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTournamentService) GetTournamentBySlug(ctx context.Context, slug string) (*Tournament, error) {
	if m.GetTournamentBySlugFunc != nil {
		return m.GetTournamentBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *MockTournamentService) UpdateTournament(ctx context.Context, id uuid.UUID, req *UpdateTournamentRequest) (*Tournament, error) {
	if m.UpdateTournamentFunc != nil {
		return m.UpdateTournamentFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *MockTournamentService) DeleteTournament(ctx context.Context, id uuid.UUID) error {
	if m.DeleteTournamentFunc != nil {
		return m.DeleteTournamentFunc(ctx, id)
	}
	return nil
}

func (m *MockTournamentService) ListTournaments(ctx context.Context, filter *ListFilters) ([]Tournament, int64, error) {
	if m.ListTournamentsFunc != nil {
		return m.ListTournamentsFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *MockTournamentService) PublishTournament(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	if m.PublishTournamentFunc != nil {
		return m.PublishTournamentFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTournamentService) StartTournament(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	if m.StartTournamentFunc != nil {
		return m.StartTournamentFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTournamentService) GetParticipants(ctx context.Context, tournamentID uuid.UUID, filter *ParticipantFilter) ([]Participant, error) {
	if m.GetParticipantsFunc != nil {
		return m.GetParticipantsFunc(ctx, tournamentID, filter)
	}
	return nil, nil
}

func (m *MockTournamentService) GetRounds(ctx context.Context, tournamentID uuid.UUID) ([]Round, error) {
	if m.GetRoundsFunc != nil {
		return m.GetRoundsFunc(ctx, tournamentID)
	}
	return nil, nil
}

func (m *MockTournamentService) GetPairings(ctx context.Context, tournamentID uuid.UUID, round int) ([]Game, error) {
	if m.GetPairingsFunc != nil {
		return m.GetPairingsFunc(ctx, tournamentID, round)
	}
	return nil, nil
}

// TestableHandler is a testable version of the handler using interfaces.
type TestableHandler struct {
	service TournamentService
}

func NewTestableHandler(service TournamentService) *TestableHandler {
	return &TestableHandler{service: service}
}

func (h *TestableHandler) GetTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	tournament, err := h.service.GetTournament(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tournament not found"})
		return
	}

	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

func (h *TestableHandler) GetTournamentBySlug(c *gin.Context) {
	slug := c.Param("slug")
	tournament, err := h.service.GetTournamentBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tournament not found"})
		return
	}
	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

func (h *TestableHandler) CreateTournament(c *gin.Context) {
	var req CreateTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	organizerID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	tournament, err := h.service.CreateTournament(c.Request.Context(), &req, organizerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, TournamentResponse{Tournament: tournament})
}

func (h *TestableHandler) ListTournaments(c *gin.Context) {
	filter := &ListFilters{Limit: 20, Page: 0}

	tournaments, total, err := h.service.ListTournaments(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TournamentListResponse{
		Tournaments: tournaments,
		Total:       int(total),
		Limit:       filter.Limit,
		Offset:      filter.Page * filter.Limit,
	})
}

func (h *TestableHandler) DeleteTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	if err := h.service.DeleteTournament(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *TestableHandler) PublishTournament(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	tournament, err := h.service.PublishTournament(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TournamentResponse{Tournament: tournament})
}

func (h *TestableHandler) GetRounds(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	rounds, err := h.service.GetRounds(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RoundListResponse{Rounds: rounds})
}

func (h *TestableHandler) GetParticipants(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
		return
	}

	filter := &ParticipantFilter{}
	participants, err := h.service.GetParticipants(c.Request.Context(), id, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ParticipantListResponse{
		Participants: participants,
		Total:        len(participants),
	})
}

func setupTournamentTestRouter(handler *TestableHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/tournaments", handler.ListTournaments)
	r.GET("/tournaments/:id", handler.GetTournament)
	r.GET("/tournaments/slug/:slug", handler.GetTournamentBySlug)
	r.POST("/tournaments", handler.CreateTournament)
	r.DELETE("/tournaments/:id", handler.DeleteTournament)
	r.POST("/tournaments/:id/publish", handler.PublishTournament)
	r.GET("/tournaments/:id/rounds", handler.GetRounds)
	r.GET("/tournaments/:id/participants", handler.GetParticipants)

	return r
}

func testTournament() *Tournament {
	return &Tournament{
		ID:        testutil.TournamentID1,
		Name:      "Test Open",
		Slug:      "test-open",
		Type:      TypeSwiss,
		Mode:      ModeOTB,
		Status:    StatusDraft,
		Rounds:    7,
		StartDate: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestTournamentHandler_GetTournament(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		mockService := &MockTournamentService{
			GetTournamentFunc: func(ctx context.Context, id uuid.UUID) (*Tournament, error) {
				assert.Equal(t, testutil.TournamentID1, id)
				return testTournament(), nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments/"+testutil.TournamentID1.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response TournamentResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Test Open", response.Tournament.Name)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		mockService := &MockTournamentService{}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockService := &MockTournamentService{
			GetTournamentFunc: func(ctx context.Context, id uuid.UUID) (*Tournament, error) {
				return nil, ErrTournamentNotFound
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments/"+testutil.TournamentID1.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTournamentHandler_GetTournamentBySlug(t *testing.T) {
	t.Run("successful get by slug", func(t *testing.T) {
		mockService := &MockTournamentService{
			GetTournamentBySlugFunc: func(ctx context.Context, slug string) (*Tournament, error) {
				assert.Equal(t, "test-open", slug)
				return testTournament(), nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments/slug/test-open", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("slug not found", func(t *testing.T) {
		mockService := &MockTournamentService{
			GetTournamentBySlugFunc: func(ctx context.Context, slug string) (*Tournament, error) {
				return nil, ErrTournamentNotFound
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments/slug/non-existent", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTournamentHandler_CreateTournament(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		mockService := &MockTournamentService{
			CreateTournamentFunc: func(ctx context.Context, req *CreateTournamentRequest, organizerID uuid.UUID) (*Tournament, error) {
				assert.Equal(t, "New Tournament", req.Name)
				return testTournament(), nil
			},
		}
		handler := NewTestableHandler(mockService)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.POST("/tournaments", func(c *gin.Context) {
			c.Set("user_id", testutil.UserID1.String())
			handler.CreateTournament(c)
		})

		body := `{"name":"New Tournament","type":"swiss","mode":"otb","rounds":7,"startDate":"2024-07-01","timeControl":"90+30","timeControlType":"standard"}`
		req := httptest.NewRequest("POST", "/tournaments", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("unauthorized - no user_id", func(t *testing.T) {
		mockService := &MockTournamentService{}
		handler := NewTestableHandler(mockService)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		// Don't set user_id to simulate unauthorized request
		router.POST("/tournaments", handler.CreateTournament)

		body := `{"name":"New Tournament","type":"swiss","mode":"otb","rounds":7,"startDate":"2024-07-01","timeControl":"90+30","timeControlType":"standard"}`
		req := httptest.NewRequest("POST", "/tournaments", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := &MockTournamentService{}
		handler := NewTestableHandler(mockService)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.POST("/tournaments", func(c *gin.Context) {
			c.Set("user_id", testutil.UserID1.String())
			handler.CreateTournament(c)
		})

		req := httptest.NewRequest("POST", "/tournaments", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTournamentHandler_ListTournaments(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		mockService := &MockTournamentService{
			ListTournamentsFunc: func(ctx context.Context, filter *ListFilters) ([]Tournament, int64, error) {
				return []Tournament{*testTournament()}, 1, nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response TournamentListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Tournaments, 1)
		assert.Equal(t, 1, response.Total)
	})

	t.Run("empty list", func(t *testing.T) {
		mockService := &MockTournamentService{
			ListTournamentsFunc: func(ctx context.Context, filter *ListFilters) ([]Tournament, int64, error) {
				return []Tournament{}, 0, nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response TournamentListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Tournaments, 0)
	})
}

func TestTournamentHandler_DeleteTournament(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		deleteCalled := false
		mockService := &MockTournamentService{
			DeleteTournamentFunc: func(ctx context.Context, id uuid.UUID) error {
				deleteCalled = true
				assert.Equal(t, testutil.TournamentID1, id)
				return nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("DELETE", "/tournaments/"+testutil.TournamentID1.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.True(t, deleteCalled)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		mockService := &MockTournamentService{}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("DELETE", "/tournaments/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTournamentHandler_PublishTournament(t *testing.T) {
	t.Run("successful publish", func(t *testing.T) {
		mockService := &MockTournamentService{
			PublishTournamentFunc: func(ctx context.Context, id uuid.UUID) (*Tournament, error) {
				tournament := testTournament()
				tournament.Status = StatusPublished
				return tournament, nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("POST", "/tournaments/"+testutil.TournamentID1.String()+"/publish", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response TournamentResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, StatusPublished, response.Tournament.Status)
	})

	t.Run("publish failure", func(t *testing.T) {
		mockService := &MockTournamentService{
			PublishTournamentFunc: func(ctx context.Context, id uuid.UUID) (*Tournament, error) {
				return nil, ErrInvalidStatus
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("POST", "/tournaments/"+testutil.TournamentID1.String()+"/publish", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTournamentHandler_GetRounds(t *testing.T) {
	t.Run("successful get rounds", func(t *testing.T) {
		mockService := &MockTournamentService{
			GetRoundsFunc: func(ctx context.Context, tournamentID uuid.UUID) ([]Round, error) {
				return []Round{
					{ID: testutil.RoundID1, TournamentID: tournamentID, RoundNumber: 1, Status: RoundStatusPending},
					{ID: testutil.RoundID2, TournamentID: tournamentID, RoundNumber: 2, Status: RoundStatusPending},
				}, nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments/"+testutil.TournamentID1.String()+"/rounds", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RoundListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Rounds, 2)
	})
}

func TestTournamentHandler_GetParticipants(t *testing.T) {
	t.Run("successful get participants", func(t *testing.T) {
		mockService := &MockTournamentService{
			GetParticipantsFunc: func(ctx context.Context, tournamentID uuid.UUID, filter *ParticipantFilter) ([]Participant, error) {
				return []Participant{
					{ID: testutil.PlayerID1, TournamentID: tournamentID, Points: 3.5},
					{ID: testutil.PlayerID2, TournamentID: tournamentID, Points: 3.0},
				}, nil
			},
		}
		handler := NewTestableHandler(mockService)
		router := setupTournamentTestRouter(handler)

		req := httptest.NewRequest("GET", "/tournaments/"+testutil.TournamentID1.String()+"/participants", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ParticipantListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Participants, 2)
		assert.Equal(t, 2, response.Total)
	})
}
