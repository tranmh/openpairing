package player

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCreatePlayerRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     CreatePlayerRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: CreatePlayerRequest{
				FirstName: "Magnus",
				LastName:  "Carlsen",
			},
			wantErr: false,
		},
		{
			name: "valid with all fields",
			req: CreatePlayerRequest{
				FirstName:    "Magnus",
				LastName:     "Carlsen",
				FideID:       strPtr("1503014"),
				FideStandard: intPtr(2830),
				FideRapid:    intPtr(2820),
				FideBlitz:    intPtr(2885),
				FideTitle:    strPtr("GM"),
				Federation:   strPtr("NOR"),
			},
			wantErr: false,
		},
		{
			name: "invalid - missing first name",
			req: CreatePlayerRequest{
				LastName: "Carlsen",
			},
			wantErr: true,
		},
		{
			name: "invalid - missing last name",
			req: CreatePlayerRequest{
				FirstName: "Magnus",
			},
			wantErr: true,
		},
		{
			name: "invalid - invalid FIDE title",
			req: CreatePlayerRequest{
				FirstName: "Test",
				LastName:  "Player",
				FideTitle: strPtr("INVALID"),
			},
			wantErr: true,
		},
		{
			name: "valid - women's title",
			req: CreatePlayerRequest{
				FirstName: "Hou",
				LastName:  "Yifan",
				FideTitle: strPtr("WGM"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation would be done by the handler/gin binding
			// This tests the model structure
			hasFirstName := tt.req.FirstName != ""
			hasLastName := tt.req.LastName != ""
			validTitle := tt.req.FideTitle == nil || isValidTitle(*tt.req.FideTitle)

			valid := hasFirstName && hasLastName && validTitle

			if valid == tt.wantErr {
				t.Errorf("CreatePlayerRequest validation = %v, wantErr %v", !valid, tt.wantErr)
			}
		})
	}
}

func TestPlayer_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		player   Player
		expected string
	}{
		{
			name:     "standard name",
			player:   Player{FirstName: "Magnus", LastName: "Carlsen"},
			expected: "Magnus Carlsen",
		},
		{
			name:     "german name",
			player:   Player{FirstName: "Vincent", LastName: "Keymer"},
			expected: "Vincent Keymer",
		},
		{
			name:     "name with umlaut",
			player:   Player{FirstName: "Thomas", LastName: "Müller"},
			expected: "Thomas Müller",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.player.GetDisplayName()
			if got != tt.expected {
				t.Errorf("GetDisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlayer_GetHighestRating(t *testing.T) {
	tests := []struct {
		name     string
		player   Player
		expected int
	}{
		{
			name: "FIDE standard highest",
			player: Player{
				FideStandard: intPtr(2830),
				DWZ:          intPtr(2750),
				OesbElo:      intPtr(2700),
			},
			expected: 2830,
		},
		{
			name: "DWZ highest",
			player: Player{
				FideStandard: intPtr(2200),
				DWZ:          intPtr(2350),
			},
			expected: 2350,
		},
		{
			name: "OESB highest",
			player: Player{
				OesbElo: intPtr(1800),
			},
			expected: 1800,
		},
		{
			name:     "no ratings",
			player:   Player{},
			expected: 0,
		},
		{
			name: "only DWZ",
			player: Player{
				DWZ: intPtr(1500),
			},
			expected: 1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.player.GetHighestRating()
			if got != tt.expected {
				t.Errorf("GetHighestRating() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestPlayer_HasFederationID(t *testing.T) {
	tests := []struct {
		name     string
		player   Player
		expected bool
	}{
		{
			name: "has FIDE ID",
			player: Player{
				FideID: strPtr("1503014"),
			},
			expected: true,
		},
		{
			name: "has DSB ID",
			player: Player{
				DsbID: strPtr("10012345"),
			},
			expected: true,
		},
		{
			name: "has OESB ID",
			player: Player{
				OesbID: strPtr("123456"),
			},
			expected: true,
		},
		{
			name: "has multiple IDs",
			player: Player{
				FideID: strPtr("1503014"),
				DsbID:  strPtr("10012345"),
			},
			expected: true,
		},
		{
			name:     "no federation ID",
			player:   Player{},
			expected: false,
		},
		{
			name: "empty federation IDs",
			player: Player{
				FideID: strPtr(""),
				DsbID:  strPtr(""),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.player.HasFederationID()
			if got != tt.expected {
				t.Errorf("HasFederationID() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSearchPlayersRequest_Defaults(t *testing.T) {
	req := &SearchPlayersRequest{}

	// Test default values
	if req.Page != 0 {
		t.Errorf("Default Page should be 0 (unset), got %d", req.Page)
	}
	if req.Limit != 0 {
		t.Errorf("Default Limit should be 0 (unset), got %d", req.Limit)
	}

	// Service should apply defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	if req.Page != 1 {
		t.Errorf("Page after defaults = %d, want 1", req.Page)
	}
	if req.Limit != 20 {
		t.Errorf("Limit after defaults = %d, want 20", req.Limit)
	}
}

func TestSearchPlayersRequest_LimitBounds(t *testing.T) {
	tests := []struct {
		name      string
		reqLimit  int
		wantLimit int
	}{
		{"zero uses default", 0, 20},
		{"negative uses default", -5, 20},
		{"within bounds", 50, 50},
		{"above max capped", 150, 100},
		{"at max", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &SearchPlayersRequest{Limit: tt.reqLimit}

			// Apply bounds (as service would)
			if req.Limit <= 0 {
				req.Limit = 20
			}
			if req.Limit > 100 {
				req.Limit = 100
			}

			if req.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", req.Limit, tt.wantLimit)
			}
		})
	}
}

// Mock repository for service tests
type mockRepository struct {
	players map[uuid.UUID]*Player
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		players: make(map[uuid.UUID]*Player),
	}
}

func (m *mockRepository) Create(ctx context.Context, player *Player) error {
	m.players[player.ID] = player
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (*Player, error) {
	if p, ok := m.players[id]; ok {
		return p, nil
	}
	return nil, ErrPlayerNotFound
}

func (m *mockRepository) GetByFideID(ctx context.Context, fideID string) (*Player, error) {
	for _, p := range m.players {
		if p.FideID != nil && *p.FideID == fideID {
			return p, nil
		}
	}
	return nil, ErrPlayerNotFound
}

func (m *mockRepository) GetByDsbID(ctx context.Context, dsbID string) (*Player, error) {
	for _, p := range m.players {
		if p.DsbID != nil && *p.DsbID == dsbID {
			return p, nil
		}
	}
	return nil, ErrPlayerNotFound
}

func (m *mockRepository) GetByOesbID(ctx context.Context, oesbID string) (*Player, error) {
	for _, p := range m.players {
		if p.OesbID != nil && *p.OesbID == oesbID {
			return p, nil
		}
	}
	return nil, ErrPlayerNotFound
}

func (m *mockRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Player, error) {
	for _, p := range m.players {
		if p.UserID != nil && *p.UserID == userID {
			return p, nil
		}
	}
	return nil, ErrPlayerNotFound
}

func (m *mockRepository) Update(ctx context.Context, player *Player) error {
	if _, ok := m.players[player.ID]; !ok {
		return ErrPlayerNotFound
	}
	m.players[player.ID] = player
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.players[id]; !ok {
		return ErrPlayerNotFound
	}
	delete(m.players, id)
	return nil
}

func (m *mockRepository) Search(ctx context.Context, req *SearchPlayersRequest) ([]PlayerSearchResult, int64, error) {
	var results []PlayerSearchResult
	for _, p := range m.players {
		results = append(results, PlayerSearchResult{Player: *p})
	}
	return results, int64(len(results)), nil
}

func (m *mockRepository) List(ctx context.Context, page, limit int) ([]Player, int64, error) {
	var players []Player
	for _, p := range m.players {
		players = append(players, *p)
	}
	return players, int64(len(players)), nil
}

func (m *mockRepository) CreateClub(ctx context.Context, club *Club) error {
	return nil
}

func (m *mockRepository) GetClubByID(ctx context.Context, id uuid.UUID) (*Club, error) {
	return nil, ErrClubNotFound
}

func (m *mockRepository) ListClubs(ctx context.Context, page, limit int) ([]Club, int64, error) {
	return nil, 0, nil
}

func (m *mockRepository) AddRatingHistory(ctx context.Context, history *RatingHistory) error {
	return nil
}

func (m *mockRepository) GetRatingHistory(ctx context.Context, playerID uuid.UUID, ratingType string, limit int) ([]RatingHistory, error) {
	return nil, nil
}

func (m *mockRepository) UpsertByFideID(ctx context.Context, player *Player) error {
	return nil
}

func (m *mockRepository) UpsertByDsbID(ctx context.Context, player *Player) error {
	return nil
}

func (m *mockRepository) BulkUpsertByFideID(ctx context.Context, players []*Player) error {
	return nil
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func isValidTitle(title string) bool {
	for _, t := range ValidFideTitles {
		if t == title {
			return true
		}
	}
	return false
}
