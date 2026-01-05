package club

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService() (*Service, *MockRepository) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	service := NewService(repo, logger)
	return service, repo
}

func TestService_Create(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	req := &CreateClubRequest{
		Name:    "Test Club",
		Country: "GER",
	}

	club, err := service.Create(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, club)
	assert.Equal(t, "Test Club", club.Name)
	assert.Equal(t, "GER", club.Country)
	assert.NotEqual(t, uuid.Nil, club.ID)
}

func TestService_Create_WithOptionalFields(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	shortName := "TCB"
	federation := "GER"
	dsbID := "12345"
	city := "Berlin"
	website := "https://test-club.de"
	email := "info@test-club.de"

	req := &CreateClubRequest{
		Name:       "Test Club Berlin",
		ShortName:  &shortName,
		Federation: &federation,
		DsbClubID:  &dsbID,
		City:       &city,
		Country:    "GER",
		Website:    &website,
		Email:      &email,
	}

	club, err := service.Create(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "TCB", *club.ShortName)
	assert.Equal(t, "GER", *club.Federation)
	assert.Equal(t, "12345", *club.DsbClubID)
	assert.Equal(t, "Berlin", *club.City)
}

func TestService_Create_Duplicate(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	dsbID := "12345"
	req := &CreateClubRequest{
		Name:      "Test Club 1",
		Country:   "GER",
		DsbClubID: &dsbID,
	}

	_, err := service.Create(ctx, req)
	require.NoError(t, err)

	req2 := &CreateClubRequest{
		Name:      "Test Club 2",
		Country:   "GER",
		DsbClubID: &dsbID,
	}

	_, err = service.Create(ctx, req2)
	assert.ErrorIs(t, err, ErrDuplicateClub)
}

func TestService_GetByID(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	req := &CreateClubRequest{
		Name:    "Test Club",
		Country: "GER",
	}

	created, err := service.Create(ctx, req)
	require.NoError(t, err)

	found, err := service.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.Name, found.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	_, err := service.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestService_GetByDsbID(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	dsbID := "12345"
	req := &CreateClubRequest{
		Name:      "Test Club",
		Country:   "GER",
		DsbClubID: &dsbID,
	}

	created, err := service.Create(ctx, req)
	require.NoError(t, err)

	found, err := service.GetByDsbID(ctx, dsbID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
}

func TestService_GetByOesbID(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	oesbID := "AT-123"
	req := &CreateClubRequest{
		Name:       "Test Club",
		Country:    "AUT",
		OesbClubID: &oesbID,
	}

	created, err := service.Create(ctx, req)
	require.NoError(t, err)

	found, err := service.GetByOesbID(ctx, oesbID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
}

func TestService_Update(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	req := &CreateClubRequest{
		Name:    "Test Club",
		Country: "GER",
	}

	created, err := service.Create(ctx, req)
	require.NoError(t, err)

	newName := "Updated Club"
	newCity := "Munich"
	updateReq := &UpdateClubRequest{
		Name: &newName,
		City: &newCity,
	}

	updated, err := service.Update(ctx, created.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Club", updated.Name)
	assert.Equal(t, "Munich", *updated.City)
}

func TestService_Update_NotFound(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	newName := "Updated Club"
	updateReq := &UpdateClubRequest{
		Name: &newName,
	}

	_, err := service.Update(ctx, uuid.New(), updateReq)
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestService_Update_PartialUpdate(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	city := "Berlin"
	req := &CreateClubRequest{
		Name:    "Test Club",
		City:    &city,
		Country: "GER",
	}

	created, err := service.Create(ctx, req)
	require.NoError(t, err)

	// Only update name, city should remain
	newName := "Updated Club"
	updateReq := &UpdateClubRequest{
		Name: &newName,
	}

	updated, err := service.Update(ctx, created.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Club", updated.Name)
	assert.Equal(t, "Berlin", *updated.City) // City should be unchanged
}

func TestService_Delete(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	req := &CreateClubRequest{
		Name:    "Test Club",
		Country: "GER",
	}

	created, err := service.Create(ctx, req)
	require.NoError(t, err)

	err = service.Delete(ctx, created.ID)
	require.NoError(t, err)

	_, err = service.GetByID(ctx, created.ID)
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	err := service.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestService_List(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	// Create multiple clubs
	for i := 0; i < 15; i++ {
		req := &CreateClubRequest{
			Name:    "Test Club",
			Country: "GER",
		}
		_, err := service.Create(ctx, req)
		require.NoError(t, err)
	}

	result, err := service.List(ctx, ClubListParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 15, result.TotalCount)
	assert.Equal(t, 10, len(result.Clubs))
	assert.Equal(t, 2, result.TotalPages)
}

func TestService_GetMembers(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	req := &CreateClubRequest{
		Name:    "Test Club",
		Country: "GER",
	}

	club, err := service.Create(ctx, req)
	require.NoError(t, err)

	// Add members to mock
	repo.AddMember(club.ID, ClubMember{
		PlayerID:   uuid.New(),
		PlayerName: "Player 1",
		Rating:     1800,
	})

	members, err := service.GetMembers(ctx, club.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "Player 1", members[0].PlayerName)
}

func TestService_GetMembers_ClubNotFound(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	_, err := service.GetMembers(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestService_GetStats(t *testing.T) {
	service, repo := newTestService()
	ctx := context.Background()

	req := &CreateClubRequest{
		Name:    "Test Club",
		Country: "GER",
	}

	club, err := service.Create(ctx, req)
	require.NoError(t, err)

	// Set stats in mock
	repo.SetStats(club.ID, &ClubStats{
		MemberCount:   10,
		AverageRating: 1850.5,
		HighestRating: 2200,
		TitledPlayers: 2,
		ActiveTeams:   3,
	})

	stats, err := service.GetStats(ctx, club.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, stats.MemberCount)
	assert.Equal(t, 2200, stats.HighestRating)
}

func TestService_GetStats_ClubNotFound(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	_, err := service.GetStats(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrClubNotFound)
}

func TestService_Search(t *testing.T) {
	service, _ := newTestService()
	ctx := context.Background()

	req := &CreateClubRequest{
		Name:    "Schachclub Berlin",
		Country: "GER",
	}
	_, err := service.Create(ctx, req)
	require.NoError(t, err)

	clubs, err := service.Search(ctx, "Berlin", 10)
	require.NoError(t, err)
	assert.Len(t, clubs, 1)
}
