package club

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	clubs   map[uuid.UUID]*Club
	members map[uuid.UUID][]ClubMember
	stats   map[uuid.UUID]*ClubStats
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		clubs:   make(map[uuid.UUID]*Club),
		members: make(map[uuid.UUID][]ClubMember),
		stats:   make(map[uuid.UUID]*ClubStats),
	}
}

func (m *MockRepository) Create(ctx context.Context, club *Club) error {
	// Check for duplicates
	for _, c := range m.clubs {
		if club.DsbClubID != nil && c.DsbClubID != nil && *club.DsbClubID == *c.DsbClubID {
			return ErrDuplicateClub
		}
		if club.OesbClubID != nil && c.OesbClubID != nil && *club.OesbClubID == *c.OesbClubID {
			return ErrDuplicateClub
		}
	}

	club.ID = uuid.New()
	m.clubs[club.ID] = club
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*Club, error) {
	club, ok := m.clubs[id]
	if !ok {
		return nil, ErrClubNotFound
	}
	return club, nil
}

func (m *MockRepository) GetByDsbID(ctx context.Context, dsbID string) (*Club, error) {
	for _, club := range m.clubs {
		if club.DsbClubID != nil && *club.DsbClubID == dsbID {
			return club, nil
		}
	}
	return nil, ErrClubNotFound
}

func (m *MockRepository) GetByOesbID(ctx context.Context, oesbID string) (*Club, error) {
	for _, club := range m.clubs {
		if club.OesbClubID != nil && *club.OesbClubID == oesbID {
			return club, nil
		}
	}
	return nil, ErrClubNotFound
}

func (m *MockRepository) Update(ctx context.Context, club *Club) error {
	if _, ok := m.clubs[club.ID]; !ok {
		return ErrClubNotFound
	}
	m.clubs[club.ID] = club
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.clubs[id]; !ok {
		return ErrClubNotFound
	}
	delete(m.clubs, id)
	return nil
}

func (m *MockRepository) List(ctx context.Context, params ClubListParams) (*ClubListResponse, error) {
	clubs := make([]Club, 0)
	for _, club := range m.clubs {
		clubs = append(clubs, *club)
	}

	// Apply pagination
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	totalCount := len(clubs)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > totalCount {
		clubs = []Club{}
	} else if end > totalCount {
		clubs = clubs[start:]
	} else {
		clubs = clubs[start:end]
	}

	return &ClubListResponse{
		Clubs:      clubs,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (totalCount + pageSize - 1) / pageSize,
	}, nil
}

func (m *MockRepository) GetMembers(ctx context.Context, clubID uuid.UUID) ([]ClubMember, error) {
	if _, ok := m.clubs[clubID]; !ok {
		return nil, ErrClubNotFound
	}
	members, ok := m.members[clubID]
	if !ok {
		return []ClubMember{}, nil
	}
	return members, nil
}

func (m *MockRepository) GetStats(ctx context.Context, clubID uuid.UUID) (*ClubStats, error) {
	if _, ok := m.clubs[clubID]; !ok {
		return nil, ErrClubNotFound
	}
	stats, ok := m.stats[clubID]
	if !ok {
		return &ClubStats{}, nil
	}
	return stats, nil
}

func (m *MockRepository) Search(ctx context.Context, query string, limit int) ([]Club, error) {
	clubs := make([]Club, 0)
	for _, club := range m.clubs {
		clubs = append(clubs, *club)
	}
	if limit > 0 && len(clubs) > limit {
		clubs = clubs[:limit]
	}
	return clubs, nil
}

// Helper to add test data
func (m *MockRepository) AddMember(clubID uuid.UUID, member ClubMember) {
	m.members[clubID] = append(m.members[clubID], member)
}

func (m *MockRepository) SetStats(clubID uuid.UUID, stats *ClubStats) {
	m.stats[clubID] = stats
}

func TestMockRepository_Create(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	club := &Club{
		Name:    "Test Club",
		Country: "GER",
	}

	err := repo.Create(ctx, club)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, club.ID)
}

func TestMockRepository_Create_Duplicate(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	dsbID := "12345"
	club1 := &Club{
		Name:      "Club 1",
		Country:   "GER",
		DsbClubID: &dsbID,
	}

	err := repo.Create(ctx, club1)
	require.NoError(t, err)

	club2 := &Club{
		Name:      "Club 2",
		Country:   "GER",
		DsbClubID: &dsbID,
	}

	err = repo.Create(ctx, club2)
	assert.ErrorIs(t, err, ErrDuplicateClub)
}

func TestMockRepository_GetByID(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	club := &Club{
		Name:    "Test Club",
		Country: "GER",
	}

	err := repo.Create(ctx, club)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, club.ID)
	require.NoError(t, err)
	assert.Equal(t, club.Name, found.Name)
}

func TestMockRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestMockRepository_GetByDsbID(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	dsbID := "12345"
	club := &Club{
		Name:      "Test Club",
		Country:   "GER",
		DsbClubID: &dsbID,
	}

	err := repo.Create(ctx, club)
	require.NoError(t, err)

	found, err := repo.GetByDsbID(ctx, dsbID)
	require.NoError(t, err)
	assert.Equal(t, club.Name, found.Name)
}

func TestMockRepository_GetByOesbID(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	oesbID := "AT-123"
	club := &Club{
		Name:       "Test Club",
		Country:    "AUT",
		OesbClubID: &oesbID,
	}

	err := repo.Create(ctx, club)
	require.NoError(t, err)

	found, err := repo.GetByOesbID(ctx, oesbID)
	require.NoError(t, err)
	assert.Equal(t, club.Name, found.Name)
}

func TestMockRepository_Update(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	club := &Club{
		Name:    "Test Club",
		Country: "GER",
	}

	err := repo.Create(ctx, club)
	require.NoError(t, err)

	club.Name = "Updated Club"
	err = repo.Update(ctx, club)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, club.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Club", found.Name)
}

func TestMockRepository_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	club := &Club{
		ID:      uuid.New(),
		Name:    "Test Club",
		Country: "GER",
	}

	err := repo.Update(ctx, club)
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestMockRepository_Delete(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	club := &Club{
		Name:    "Test Club",
		Country: "GER",
	}

	err := repo.Create(ctx, club)
	require.NoError(t, err)

	err = repo.Delete(ctx, club.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, club.ID)
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestMockRepository_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestMockRepository_List(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Create multiple clubs
	for i := 0; i < 25; i++ {
		club := &Club{
			Name:    "Test Club",
			Country: "GER",
		}
		err := repo.Create(ctx, club)
		require.NoError(t, err)
	}

	// Test first page
	result, err := repo.List(ctx, ClubListParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 25, result.TotalCount)
	assert.Equal(t, 10, len(result.Clubs))
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 3, result.TotalPages)

	// Test last page
	result, err = repo.List(ctx, ClubListParams{Page: 3, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 5, len(result.Clubs))
}

func TestMockRepository_GetMembers(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	club := &Club{
		Name:    "Test Club",
		Country: "GER",
	}
	err := repo.Create(ctx, club)
	require.NoError(t, err)

	// Add members
	repo.AddMember(club.ID, ClubMember{
		PlayerID:   uuid.New(),
		PlayerName: "Player 1",
		Rating:     1800,
	})
	repo.AddMember(club.ID, ClubMember{
		PlayerID:   uuid.New(),
		PlayerName: "Player 2",
		Rating:     1900,
	})

	members, err := repo.GetMembers(ctx, club.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestMockRepository_GetStats(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	club := &Club{
		Name:    "Test Club",
		Country: "GER",
	}
	err := repo.Create(ctx, club)
	require.NoError(t, err)

	repo.SetStats(club.ID, &ClubStats{
		MemberCount:   10,
		AverageRating: 1850.5,
		HighestRating: 2200,
		TitledPlayers: 2,
		ActiveTeams:   3,
	})

	stats, err := repo.GetStats(ctx, club.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, stats.MemberCount)
	assert.Equal(t, 1850.5, stats.AverageRating)
	assert.Equal(t, 2200, stats.HighestRating)
	assert.Equal(t, 2, stats.TitledPlayers)
	assert.Equal(t, 3, stats.ActiveTeams)
}

func TestMockRepository_Search(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	club := &Club{
		Name:    "Schachclub Berlin",
		Country: "GER",
	}
	err := repo.Create(ctx, club)
	require.NoError(t, err)

	clubs, err := repo.Search(ctx, "Berlin", 10)
	require.NoError(t, err)
	assert.Len(t, clubs, 1)
}
