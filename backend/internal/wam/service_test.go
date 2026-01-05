package wam

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/pairing/engine"
	"github.com/rs/zerolog"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	config       *WAMConfig
	groups       []WAMGroup
	members      map[uuid.UUID][]WAMGroupPlayer
	participants []TournamentParticipant
	games        map[uuid.UUID][]Game
	playoffs     []WAMPlayoffMatch
	quals        []WAMQualification
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		members: make(map[uuid.UUID][]WAMGroupPlayer),
		games:   make(map[uuid.UUID][]Game),
	}
}

func (m *MockRepository) GetConfig(ctx context.Context, tournamentID uuid.UUID) (*WAMConfig, error) {
	if m.config == nil {
		return nil, ErrConfigNotFound
	}
	return m.config, nil
}

func (m *MockRepository) CreateConfig(ctx context.Context, config *WAMConfig) error {
	m.config = config
	return nil
}

func (m *MockRepository) UpdateConfig(ctx context.Context, config *WAMConfig) error {
	m.config = config
	return nil
}

func (m *MockRepository) DeleteConfig(ctx context.Context, tournamentID uuid.UUID) error {
	m.config = nil
	return nil
}

func (m *MockRepository) GetTournamentParticipants(ctx context.Context, tournamentID uuid.UUID) ([]TournamentParticipant, error) {
	return m.participants, nil
}

func (m *MockRepository) CreateGroup(ctx context.Context, group *WAMGroup) error {
	m.groups = append(m.groups, *group)
	return nil
}

func (m *MockRepository) GetGroup(ctx context.Context, groupID uuid.UUID) (*WAMGroup, error) {
	for i, g := range m.groups {
		if g.ID == groupID {
			// Add members
			g.Players = m.members[groupID]
			m.groups[i] = g
			return &m.groups[i], nil
		}
	}
	return nil, ErrGroupNotFound
}

func (m *MockRepository) ListGroups(ctx context.Context, tournamentID uuid.UUID) ([]WAMGroup, error) {
	result := make([]WAMGroup, len(m.groups))
	for i, g := range m.groups {
		g.Players = m.members[g.ID]
		result[i] = g
	}
	return result, nil
}

func (m *MockRepository) GetGroupByNumber(ctx context.Context, tournamentID uuid.UUID, groupNumber int) (*WAMGroup, error) {
	for i, g := range m.groups {
		if g.TournamentID == tournamentID && g.GroupNumber == groupNumber {
			g.Players = m.members[g.ID]
			m.groups[i] = g
			return &m.groups[i], nil
		}
	}
	return nil, ErrGroupNotFound
}

func (m *MockRepository) UpdateGroup(ctx context.Context, group *WAMGroup) error {
	for i, g := range m.groups {
		if g.ID == group.ID {
			m.groups[i] = *group
			return nil
		}
	}
	return nil
}

func (m *MockRepository) DeleteGroup(ctx context.Context, groupID uuid.UUID) error {
	for i, g := range m.groups {
		if g.ID == groupID {
			m.groups = append(m.groups[:i], m.groups[i+1:]...)
			delete(m.members, groupID)
			return nil
		}
	}
	return nil
}

func (m *MockRepository) DeleteAllGroups(ctx context.Context, tournamentID uuid.UUID) error {
	m.groups = nil
	m.members = make(map[uuid.UUID][]WAMGroupPlayer)
	return nil
}

func (m *MockRepository) AddGroupMember(ctx context.Context, member *WAMGroupPlayer) error {
	m.members[member.GroupID] = append(m.members[member.GroupID], *member)
	return nil
}

func (m *MockRepository) UpdateGroupMember(ctx context.Context, member *WAMGroupPlayer) error {
	members := m.members[member.GroupID]
	for i, mem := range members {
		if mem.ParticipantID == member.ParticipantID {
			m.members[member.GroupID][i] = *member
			return nil
		}
	}
	return nil
}

func (m *MockRepository) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]WAMGroupPlayer, error) {
	return m.members[groupID], nil
}

func (m *MockRepository) RemoveGroupMember(ctx context.Context, groupID, participantID uuid.UUID) error {
	members := m.members[groupID]
	for i, mem := range members {
		if mem.ParticipantID == participantID {
			m.members[groupID] = append(members[:i], members[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MockRepository) GetMaxSortOrder(ctx context.Context, groupID uuid.UUID) (int, error) {
	maxOrder := 0
	for _, mem := range m.members[groupID] {
		if mem.SortOrder > maxOrder {
			maxOrder = mem.SortOrder
		}
	}
	return maxOrder, nil
}

func (m *MockRepository) GetPlayerGroup(ctx context.Context, tournamentID, participantID uuid.UUID) (*WAMGroup, error) {
	for _, g := range m.groups {
		for _, mem := range m.members[g.ID] {
			if mem.ParticipantID == participantID {
				g.Players = m.members[g.ID]
				return &g, nil
			}
		}
	}
	return nil, ErrGroupNotFound
}

func (m *MockRepository) MovePlayerToGroup(ctx context.Context, participantID, fromGroupID, toGroupID uuid.UUID, sortOrder int) error {
	// Remove from old group
	members := m.members[fromGroupID]
	var movedMember WAMGroupPlayer
	for i, mem := range members {
		if mem.ParticipantID == participantID {
			movedMember = mem
			m.members[fromGroupID] = append(members[:i], members[i+1:]...)
			break
		}
	}
	// Add to new group
	movedMember.GroupID = toGroupID
	movedMember.SortOrder = sortOrder
	m.members[toGroupID] = append(m.members[toGroupID], movedMember)
	return nil
}

func (m *MockRepository) GetGroupGames(ctx context.Context, groupID uuid.UUID) ([]Game, error) {
	return m.games[groupID], nil
}

func (m *MockRepository) LinkGameToGroup(ctx context.Context, groupGame *WAMGroupGame) error {
	// No-op for mock
	return nil
}

func (m *MockRepository) GetPlayoffMatches(ctx context.Context, tournamentID uuid.UUID) ([]WAMPlayoffMatch, error) {
	return m.playoffs, nil
}

func (m *MockRepository) CreatePlayoffMatch(ctx context.Context, match *WAMPlayoffMatch) error {
	m.playoffs = append(m.playoffs, *match)
	return nil
}

func (m *MockRepository) UpdatePlayoffMatch(ctx context.Context, match *WAMPlayoffMatch) error {
	for i, pm := range m.playoffs {
		if pm.ID == match.ID {
			m.playoffs[i] = *match
			return nil
		}
	}
	return nil
}

func (m *MockRepository) DeletePlayoffMatches(ctx context.Context, tournamentID uuid.UUID) error {
	m.playoffs = nil
	return nil
}

func (m *MockRepository) GetQualifications(ctx context.Context, tournamentID uuid.UUID) ([]WAMQualification, error) {
	return m.quals, nil
}

func (m *MockRepository) AddQualification(ctx context.Context, qual *WAMQualification) error {
	m.quals = append(m.quals, *qual)
	return nil
}

func (m *MockRepository) GetQualificationsBySeason(ctx context.Context, season string) ([]WAMQualification, error) {
	var result []WAMQualification
	for _, q := range m.quals {
		if q.Season == season {
			result = append(result, q)
		}
	}
	return result, nil
}

func (m *MockRepository) GetPlayerQualifications(ctx context.Context, playerID uuid.UUID) ([]WAMQualification, error) {
	var result []WAMQualification
	for _, q := range m.quals {
		if q.PlayerID == playerID {
			result = append(result, q)
		}
	}
	return result, nil
}

func (m *MockRepository) DeleteQualifications(ctx context.Context, tournamentID uuid.UUID) error {
	var remaining []WAMQualification
	for _, q := range m.quals {
		if q.TournamentID != tournamentID {
			remaining = append(remaining, q)
		}
	}
	m.quals = remaining
	return nil
}

func (m *MockRepository) SetJokerPoints(ctx context.Context, tournamentID, participantID uuid.UUID, jokerPoints int) error {
	for i := range m.participants {
		if m.participants[i].ParticipantID == participantID {
			m.participants[i].JokerPoints = jokerPoints
			return nil
		}
	}
	return ErrMemberNotFound
}

func (m *MockRepository) GetJokerPoints(ctx context.Context, tournamentID, participantID uuid.UUID) (int, error) {
	for _, p := range m.participants {
		if p.ParticipantID == participantID {
			return p.JokerPoints, nil
		}
	}
	return 0, ErrMemberNotFound
}

// Test cases

func TestServiceAssignGroupsSequential(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	// Create config
	repo.config = &WAMConfig{
		TournamentID:     tournamentID,
		GroupSize:        4,
		AssignmentMethod: engine.WAMAssignmentSequential,
		CurrentPhase:     engine.WAMPhaseSetup,
	}

	// Create 12 participants
	for i := 0; i < 12; i++ {
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: uuid.New(),
			Name:          "Player " + string(rune('A'+i)),
			Rating:        2000 - i*10,
		})
	}

	// Assign groups
	req := &AssignGroupsRequest{AssignmentMethod: "sequential"}
	resp, err := svc.AssignGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("AssignGroups failed: %v", err)
	}

	if len(resp.Groups) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(resp.Groups))
	}

	// Verify all participants were assigned
	totalAssigned := 0
	for _, g := range resp.Groups {
		totalAssigned += len(g.Players)
	}
	if totalAssigned != 12 {
		t.Errorf("Expected 12 players assigned, got %d", totalAssigned)
	}
}

func TestServiceAssignGroupsSequentialWithUnevenLastGroup(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	// Create config with group size 4
	repo.config = &WAMConfig{
		TournamentID:     tournamentID,
		GroupSize:        4,
		AssignmentMethod: engine.WAMAssignmentSequential,
		CurrentPhase:     engine.WAMPhaseSetup,
	}

	// Create 10 participants (not divisible by 4)
	// Expected: Group A (4), Group B (4), Group C (2)
	for i := 0; i < 10; i++ {
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: uuid.New(),
			Name:          "Player " + string(rune('A'+i)),
			Rating:        2000 - i*10,
		})
	}

	req := &AssignGroupsRequest{AssignmentMethod: "sequential"}
	resp, err := svc.AssignGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("AssignGroups failed: %v", err)
	}

	// Verify all players are assigned (this is where the bug might be!)
	totalAssigned := 0
	assignedParticipants := make(map[uuid.UUID]bool)
	for _, g := range resp.Groups {
		for _, p := range g.Players {
			if p.ParticipantID == uuid.Nil {
				t.Error("Found player with nil participant ID - mapping bug!")
			}
			if assignedParticipants[p.ParticipantID] {
				t.Errorf("Player %s assigned to multiple groups", p.ParticipantID)
			}
			assignedParticipants[p.ParticipantID] = true
			totalAssigned++
		}
	}

	if totalAssigned != 10 {
		t.Errorf("Expected 10 players assigned, got %d - some players missing!", totalAssigned)
	}
}

func TestServiceAssignGroupsSerpentine(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:     tournamentID,
		GroupSize:        4,
		AssignmentMethod: engine.WAMAssignmentSerpentine,
		CurrentPhase:     engine.WAMPhaseSetup,
	}

	// Create 16 participants with unique names and ratings
	for i := 0; i < 16; i++ {
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: uuid.New(),
			Name:          "Player " + string(rune('A'+i)),
			Rating:        2000 - i*10, // All unique ratings
		})
	}

	req := &AssignGroupsRequest{AssignmentMethod: "serpentine"}
	resp, err := svc.AssignGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("AssignGroups failed: %v", err)
	}

	// Verify all participants were assigned
	totalAssigned := 0
	assignedParticipants := make(map[uuid.UUID]bool)
	for _, g := range resp.Groups {
		for _, p := range g.Players {
			if p.ParticipantID == uuid.Nil {
				t.Error("Found player with nil participant ID")
			}
			if assignedParticipants[p.ParticipantID] {
				t.Errorf("Player %s assigned to multiple groups", p.ParticipantID)
			}
			assignedParticipants[p.ParticipantID] = true
			totalAssigned++
		}
	}

	if totalAssigned != 16 {
		t.Errorf("Expected 16 players assigned, got %d", totalAssigned)
	}
}

func TestServiceAssignGroupsSerpentineDuplicateNameRating(t *testing.T) {
	// This test exposes a bug: serpentine matching relies on name+rating
	// If two players have the same name and rating, the wrong one might be matched
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:     tournamentID,
		GroupSize:        4,
		AssignmentMethod: engine.WAMAssignmentSerpentine,
		CurrentPhase:     engine.WAMPhaseSetup,
	}

	// Create participants with duplicate names and ratings
	for i := 0; i < 8; i++ {
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: uuid.New(),
			Name:          "Player A", // All same name!
			Rating:        2000,       // All same rating!
		})
	}

	req := &AssignGroupsRequest{AssignmentMethod: "serpentine"}
	resp, err := svc.AssignGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("AssignGroups failed: %v", err)
	}

	// Check for duplicate participant assignments (the bug!)
	assignedParticipants := make(map[uuid.UUID]int)
	for _, g := range resp.Groups {
		for _, p := range g.Players {
			if p.ParticipantID == uuid.Nil {
				continue // Skip nil
			}
			assignedParticipants[p.ParticipantID]++
		}
	}

	// All 8 participants should be assigned exactly once
	for pid, count := range assignedParticipants {
		if count > 1 {
			t.Errorf("Participant %s assigned %d times - duplicate mapping bug!", pid, count)
		}
	}

	totalAssigned := len(assignedParticipants)
	if totalAssigned != 8 {
		t.Errorf("Expected 8 unique participants, got %d - some participants assigned to same player or missing", totalAssigned)
	}
}

func TestServiceAssignGroupsRandom(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:     tournamentID,
		GroupSize:        3,
		AssignmentMethod: engine.WAMAssignmentRandom,
		CurrentPhase:     engine.WAMPhaseSetup,
	}

	// Create 9 participants
	for i := 0; i < 9; i++ {
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: uuid.New(),
			Name:          "Player " + string(rune('A'+i)),
			Rating:        2000 - i*10,
		})
	}

	req := &AssignGroupsRequest{AssignmentMethod: "random"}
	resp, err := svc.AssignGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("AssignGroups failed: %v", err)
	}

	// The bug: random mapping uses sequential index calculation
	// but the engine shuffles players, so indices won't match
	totalAssigned := 0
	assignedParticipants := make(map[uuid.UUID]bool)
	for _, g := range resp.Groups {
		for _, p := range g.Players {
			if p.ParticipantID == uuid.Nil {
				t.Error("Found player with nil participant ID - random mapping bug!")
			}
			if assignedParticipants[p.ParticipantID] {
				t.Errorf("Player %s assigned to multiple groups", p.ParticipantID)
			}
			assignedParticipants[p.ParticipantID] = true
			totalAssigned++
		}
	}

	if totalAssigned != 9 {
		t.Errorf("Expected 9 players assigned, got %d", totalAssigned)
	}
}

func TestServiceAssignGroupsManualReturnsEmptyGroups(t *testing.T) {
	// Manual assignment should return empty groups (no automatic assignment)
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:     tournamentID,
		GroupSize:        4,
		AssignmentMethod: engine.WAMAssignmentManual,
		CurrentPhase:     engine.WAMPhaseSetup,
	}

	// Create participants
	for i := 0; i < 8; i++ {
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: uuid.New(),
			Name:          "Player " + string(rune('A'+i)),
			Rating:        2000 - i*10,
		})
	}

	req := &AssignGroupsRequest{AssignmentMethod: "manual"}
	resp, err := svc.AssignGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("AssignGroups failed: %v", err)
	}

	// Manual should create groups but not assign players
	for _, g := range resp.Groups {
		if len(g.Players) > 0 {
			// Check if any player has valid participant ID
			for _, p := range g.Players {
				if p.ParticipantID != uuid.Nil {
					t.Logf("Warning: Manual assignment assigned player %s to group - expected empty groups", p.Name)
				}
			}
		}
	}
}

func TestServiceStartGroupPhaseNoGroups(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseSetup,
	}

	// Try to start without groups
	err := svc.StartGroupPhase(ctx, tournamentID)
	if err == nil {
		t.Error("Expected error when starting group phase with no groups")
	}
}

func TestServiceStartGroupPhaseGroupWithOnePlayer(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseSetup,
	}

	// Create group with only 1 player
	groupID := uuid.New()
	repo.groups = []WAMGroup{{
		ID:           groupID,
		TournamentID: tournamentID,
		Name:         "Group A",
	}}
	repo.members[groupID] = []WAMGroupPlayer{{
		GroupID:       groupID,
		ParticipantID: uuid.New(),
		SortOrder:     1,
	}}

	err := svc.StartGroupPhase(ctx, tournamentID)
	if err == nil {
		t.Error("Expected error when group has fewer than 2 players")
	}
}

func TestServiceGenerateGroupPairingsWrongPhase(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()
	groupID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseSetup, // Wrong phase!
	}

	repo.groups = []WAMGroup{{
		ID:           groupID,
		TournamentID: tournamentID,
		Name:         "Group A",
		RoundsTotal:  3,
	}}

	// Try to generate pairings in wrong phase
	_, err := svc.GenerateGroupPairings(ctx, groupID, 1)
	if err != ErrInvalidPhase {
		t.Errorf("Expected ErrInvalidPhase, got %v", err)
	}
}

func TestServiceInitializePlayoffsGroupsNotComplete(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:   tournamentID,
		EnablePlayoffs: true,
		CurrentPhase:   engine.WAMPhaseGroup,
	}

	// Create incomplete group
	groupID := uuid.New()
	repo.groups = []WAMGroup{{
		ID:           groupID,
		TournamentID: tournamentID,
		Name:         "Group A",
		IsComplete:   false, // Not complete!
	}}

	// Try to initialize playoffs
	_, err := svc.InitializePlayoffs(ctx, tournamentID)
	if err != ErrGroupsNotComplete {
		t.Errorf("Expected ErrGroupsNotComplete, got %v", err)
	}
}

func TestServiceInitializePlayoffsDisabled(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:   tournamentID,
		EnablePlayoffs: false, // Playoffs disabled
		CurrentPhase:   engine.WAMPhaseGroup,
	}

	_, err := svc.InitializePlayoffs(ctx, tournamentID)
	if err != ErrPlayoffsNotEnabled {
		t.Errorf("Expected ErrPlayoffsNotEnabled, got %v", err)
	}
}

func TestServiceMovePlayerWrongPhase(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseGroup, // Can't move during group phase
	}

	req := &MovePlayerRequest{
		ParticipantID: uuid.New(),
		TargetGroupID: uuid.New(),
	}

	err := svc.MovePlayer(ctx, tournamentID, req)
	if err != ErrInvalidPhase {
		t.Errorf("Expected ErrInvalidPhase, got %v", err)
	}
}

func TestServiceConfigureWAMDefaults(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	req := &ConfigureWAMRequest{
		GroupSize:        4,
		AssignmentMethod: "sequential",
		EnablePlayoffs:   false,
		// PlayoffsTopN not set - should default to 1
		// YouthAgeLimit not set - should default to 12
		// Tiebreaks not set - should get defaults
	}

	config, err := svc.ConfigureWAM(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("ConfigureWAM failed: %v", err)
	}

	if config.PlayoffsTopN != 1 {
		t.Errorf("Expected PlayoffsTopN default 1, got %d", config.PlayoffsTopN)
	}

	if config.YouthAgeLimit != 12 {
		t.Errorf("Expected YouthAgeLimit default 12, got %d", config.YouthAgeLimit)
	}

	if len(config.Tiebreaks) != 3 {
		t.Errorf("Expected 3 default tiebreaks, got %d", len(config.Tiebreaks))
	}
}

func TestServiceGetStatusNotConfigured(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	// No config exists
	repo.config = nil

	status, err := svc.GetStatus(ctx, tournamentID)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status.Phase != engine.WAMPhaseSetup {
		t.Errorf("Expected Setup phase for unconfigured tournament, got %s", status.Phase)
	}
}

// ==================== NEW TESTS FOR JOKER, VARIABLE GROUPS, U12 ====================

func TestServiceSetJokerPointsValid(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()
	participantID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseSetup,
	}

	repo.participants = []TournamentParticipant{
		{ParticipantID: participantID, Name: "Test Player", Rating: 1500},
	}

	// Test valid joker values
	validJokers := []int{0, 50, 100, 150}
	for _, joker := range validJokers {
		err := svc.SetJokerPoints(ctx, tournamentID, participantID, joker)
		if err != nil {
			t.Errorf("SetJokerPoints(%d) failed: %v", joker, err)
		}

		// Verify it was set
		points, err := repo.GetJokerPoints(ctx, tournamentID, participantID)
		if err != nil {
			t.Fatalf("GetJokerPoints failed: %v", err)
		}
		if points != joker {
			t.Errorf("Expected joker points %d, got %d", joker, points)
		}
	}
}

func TestServiceSetJokerPointsInvalidValue(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()
	participantID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseSetup,
	}

	repo.participants = []TournamentParticipant{
		{ParticipantID: participantID, Name: "Test Player", Rating: 1500},
	}

	// Test invalid joker values
	invalidJokers := []int{25, 75, 125, 200, -50}
	for _, joker := range invalidJokers {
		err := svc.SetJokerPoints(ctx, tournamentID, participantID, joker)
		if err == nil {
			t.Errorf("SetJokerPoints(%d) should fail for invalid value", joker)
		}
	}
}

func TestServiceSetJokerPointsWrongPhase(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()
	participantID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseGroup, // Wrong phase!
	}

	repo.participants = []TournamentParticipant{
		{ParticipantID: participantID, Name: "Test Player", Rating: 1500},
	}

	err := svc.SetJokerPoints(ctx, tournamentID, participantID, 50)
	if err != ErrInvalidPhase {
		t.Errorf("Expected ErrInvalidPhase, got %v", err)
	}
}

func TestServiceProposeGroupsWithJokerAffectsSorting(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:      tournamentID,
		DefaultGroupSize:  4,
		AllowedGroupSizes: []int{4, 6},
		CurrentPhase:      engine.WAMPhaseSetup,
		YouthAgeLimit:     12,
	}

	// Create 8 participants
	// Player B has lower rating but with joker should be higher
	repo.participants = []TournamentParticipant{
		{ParticipantID: uuid.New(), Name: "Player A", Rating: 1500, JokerPoints: 0},    // Effective: 1500
		{ParticipantID: uuid.New(), Name: "Player B", Rating: 1400, JokerPoints: 150},  // Effective: 1550 <- should be first!
		{ParticipantID: uuid.New(), Name: "Player C", Rating: 1480, JokerPoints: 0},    // Effective: 1480
		{ParticipantID: uuid.New(), Name: "Player D", Rating: 1350, JokerPoints: 100},  // Effective: 1450
		{ParticipantID: uuid.New(), Name: "Player E", Rating: 1440, JokerPoints: 0},    // Effective: 1440
		{ParticipantID: uuid.New(), Name: "Player F", Rating: 1280, JokerPoints: 150},  // Effective: 1430
		{ParticipantID: uuid.New(), Name: "Player G", Rating: 1420, JokerPoints: 0},    // Effective: 1420
		{ParticipantID: uuid.New(), Name: "Player H", Rating: 1300, JokerPoints: 50},   // Effective: 1350
	}

	proposal, err := svc.ProposeGroups(ctx, tournamentID)
	if err != nil {
		t.Fatalf("ProposeGroups failed: %v", err)
	}

	if len(proposal.Groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(proposal.Groups))
	}

	// BUG TEST: With serpentine, Player B (effective 1550) should be in Group A first
	// Expected order by effective rating: B(1550), A(1500), C(1480), D(1450), E(1440), F(1430), G(1420), H(1350)
	// Serpentine for 2 groups: B->A, A->B, C->B, D->A, E->A, F->B, G->B, H->A
	// Group A: B, D, E, H
	// Group B: A, C, F, G

	groupAPlayers := proposal.Groups[0].Players
	if len(groupAPlayers) == 0 {
		t.Fatal("Group A has no players")
	}

	// The first player in Group A should be Player B (with effective rating 1550)
	if groupAPlayers[0].Name != "Player B" {
		t.Errorf("BUG: Joker points not affecting sort order! Expected Player B first in Group A, got %s",
			groupAPlayers[0].Name)
		t.Logf("Group A players: %v", func() []string {
			names := make([]string, len(groupAPlayers))
			for i, p := range groupAPlayers {
				names[i] = p.Name
			}
			return names
		}())
	}
}

func TestServiceProposeGroupsU12Identification(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:      tournamentID,
		DefaultGroupSize:  4,
		AllowedGroupSizes: []int{4, 6},
		CurrentPhase:      engine.WAMPhaseSetup,
		YouthAgeLimit:     12, // Under 12
	}

	now := time.Now()
	age10 := now.AddDate(-10, 0, 0)
	age11 := now.AddDate(-11, 0, 0)
	age12 := now.AddDate(-12, -1, 0) // Just turned 12
	age15 := now.AddDate(-15, 0, 0)

	repo.participants = []TournamentParticipant{
		{ParticipantID: uuid.New(), Name: "Young 10", Rating: 1200, BirthDate: &age10},
		{ParticipantID: uuid.New(), Name: "Young 11", Rating: 1180, BirthDate: &age11},
		{ParticipantID: uuid.New(), Name: "Just 12", Rating: 1300, BirthDate: &age12},   // NOT U12 anymore
		{ParticipantID: uuid.New(), Name: "Teen 15", Rating: 1400, BirthDate: &age15},
		{ParticipantID: uuid.New(), Name: "No DOB", Rating: 1350},                         // No birthdate
	}

	proposal, err := svc.ProposeGroups(ctx, tournamentID)
	if err != nil {
		t.Fatalf("ProposeGroups failed: %v", err)
	}

	// Count U12 players in proposal
	u12Count := 0
	for _, group := range proposal.Groups {
		for _, player := range group.Players {
			if player.IsU12 {
				u12Count++
				// Verify age
				if player.Age != nil && *player.Age >= 12 {
					t.Errorf("BUG: Player %s marked as U12 but age is %d", player.Name, *player.Age)
				}
			}
		}
	}

	// Should have 2 U12 players (Young 10 and Young 11)
	if u12Count != 2 {
		t.Errorf("Expected 2 U12 players, got %d", u12Count)
	}
}

func TestServiceProposeGroupsWithVariableGroupSizes(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:      tournamentID,
		DefaultGroupSize:  4,
		AllowedGroupSizes: []int{4, 6},
		CurrentPhase:      engine.WAMPhaseSetup,
	}

	// 10 players: can be 2x4 + 1x2, or should propose 1x4 + 1x6
	for i := 0; i < 10; i++ {
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: uuid.New(),
			Name:          "Player " + string(rune('A'+i)),
			Rating:        2000 - i*10,
		})
	}

	proposal, err := svc.ProposeGroups(ctx, tournamentID)
	if err != nil {
		t.Fatalf("ProposeGroups failed: %v", err)
	}

	// Count total players
	totalPlayers := 0
	for _, g := range proposal.Groups {
		totalPlayers += len(g.Players)
	}

	if totalPlayers != 10 {
		t.Errorf("Expected 10 total players in groups, got %d", totalPlayers)
	}

	// Check that all players were assigned exactly once
	assignedIDs := make(map[uuid.UUID]int)
	for _, g := range proposal.Groups {
		for _, p := range g.Players {
			if p.ParticipantID == uuid.Nil {
				t.Errorf("BUG: Player %s has nil ParticipantID in proposal", p.Name)
			}
			assignedIDs[p.ParticipantID]++
		}
	}

	for pid, count := range assignedIDs {
		if count > 1 {
			t.Errorf("BUG: Participant %s assigned %d times", pid, count)
		}
	}

	if len(assignedIDs) != 10 {
		t.Errorf("BUG: Expected 10 unique participants assigned, got %d", len(assignedIDs))
	}
}

func TestServiceProposeGroupsU12ProposalWithDuplicateNames(t *testing.T) {
	// This test exposes BUG: U12 proposal matching uses name only,
	// which fails when multiple players have the same name
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	now := time.Now()
	age10 := now.AddDate(-10, 0, 0)

	repo.config = &WAMConfig{
		TournamentID:      tournamentID,
		DefaultGroupSize:  4,
		AllowedGroupSizes: []int{4, 6},
		CurrentPhase:      engine.WAMPhaseSetup,
		YouthAgeLimit:     12,
	}

	// Create 14 participants with 7 U12 players
	// Many have the same name "Mueller" which is common in Germany
	for i := 0; i < 14; i++ {
		participant := TournamentParticipant{
			ParticipantID: uuid.New(),
			Name:          "Mueller", // All same name!
			Rating:        2000 - i*10,
		}
		// First 7 are U12
		if i < 7 {
			participant.BirthDate = &age10
		}
		repo.participants = append(repo.participants, participant)
	}

	proposal, err := svc.ProposeGroups(ctx, tournamentID)
	if err != nil {
		t.Fatalf("ProposeGroups failed: %v", err)
	}

	// If there's a U12 proposal, check for duplicate participants
	if len(proposal.U12Proposals) > 0 {
		u12Proposal := proposal.U12Proposals[0]

		// Check that all U12 proposal players have unique participant IDs
		seenIDs := make(map[uuid.UUID]bool)
		for _, player := range u12Proposal.Players {
			if player.ParticipantID == uuid.Nil {
				t.Errorf("BUG: U12 proposal player has nil ParticipantID (name: %s)", player.Name)
				continue
			}
			if seenIDs[player.ParticipantID] {
				t.Errorf("BUG: Duplicate ParticipantID in U12 proposal due to name matching: %s", player.ParticipantID)
			}
			seenIDs[player.ParticipantID] = true
		}
	}
}

func TestServiceConfirmGroupsCreatesCorrectGroups(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:      tournamentID,
		DefaultGroupSize:  4,
		AllowedGroupSizes: []int{4, 6},
		CurrentPhase:      engine.WAMPhaseSetup,
	}

	// Create 8 participants
	participantIDs := make([]uuid.UUID, 8)
	for i := 0; i < 8; i++ {
		participantIDs[i] = uuid.New()
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: participantIDs[i],
			Name:          "Player " + string(rune('A'+i)),
			Rating:        2000 - i*10,
			JokerPoints:   50 * (i % 3), // 0, 50, 100, 0, 50, 100, 0, 50
		})
	}

	req := &ConfirmGroupsRequest{
		Proposals: []ConfirmedGroup{
			{
				GroupName:      "A",
				TargetSize:     4,
				ParticipantIDs: participantIDs[:4],
			},
			{
				GroupName:      "B",
				TargetSize:     4,
				ParticipantIDs: participantIDs[4:],
			},
		},
	}

	resp, err := svc.ConfirmGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("ConfirmGroups failed: %v", err)
	}

	if len(resp.Groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(resp.Groups))
	}

	// Verify joker points were stored correctly in group members
	for _, g := range resp.Groups {
		for _, p := range g.Players {
			// Find original participant
			var originalJoker int
			for _, orig := range repo.participants {
				if orig.ParticipantID == p.ParticipantID {
					originalJoker = orig.JokerPoints
					break
				}
			}

			if p.JokerPoints != originalJoker {
				t.Errorf("BUG: Player %s joker points not preserved: expected %d, got %d",
					p.Name, originalJoker, p.JokerPoints)
			}

			expectedEffective := p.Rating + p.JokerPoints
			if p.EffectiveRating != expectedEffective {
				t.Errorf("BUG: Player %s effective rating wrong: expected %d (rating %d + joker %d), got %d",
					p.Name, expectedEffective, p.Rating, p.JokerPoints, p.EffectiveRating)
			}
		}
	}
}

func TestServiceConfirmGroupsValidatesParticipants(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseSetup,
	}

	// Create only 2 participants
	participantIDs := make([]uuid.UUID, 2)
	for i := 0; i < 2; i++ {
		participantIDs[i] = uuid.New()
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: participantIDs[i],
			Name:          "Player " + string(rune('A'+i)),
		})
	}

	// Try to confirm with a non-existent participant
	fakeID := uuid.New()
	req := &ConfirmGroupsRequest{
		Proposals: []ConfirmedGroup{
			{
				GroupName:      "A",
				TargetSize:     4,
				ParticipantIDs: []uuid.UUID{participantIDs[0], participantIDs[1], fakeID},
			},
		},
	}

	resp, err := svc.ConfirmGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("ConfirmGroups failed: %v", err)
	}

	// Should have a warning about missing participant
	foundWarning := false
	for _, w := range resp.Warnings {
		if containsSubstr(w, "not found") {
			foundWarning = true
			break
		}
	}

	if !foundWarning {
		t.Error("Expected warning about non-existent participant")
	}
}

func TestServiceConfirmGroupsSetsCorrectRounds(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID:      tournamentID,
		DefaultGroupSize:  4,
		AllowedGroupSizes: []int{4, 6},
		CurrentPhase:      engine.WAMPhaseSetup,
	}

	// Create participants
	var participantIDs4 []uuid.UUID
	var participantIDs6 []uuid.UUID

	for i := 0; i < 10; i++ {
		pid := uuid.New()
		repo.participants = append(repo.participants, TournamentParticipant{
			ParticipantID: pid,
			Name:          "Player " + string(rune('A'+i)),
			Rating:        2000 - i*10,
		})
		if i < 4 {
			participantIDs4 = append(participantIDs4, pid)
		} else {
			participantIDs6 = append(participantIDs6, pid)
		}
	}

	req := &ConfirmGroupsRequest{
		Proposals: []ConfirmedGroup{
			{GroupName: "A", TargetSize: 4, ParticipantIDs: participantIDs4}, // 4 players = 3 rounds
			{GroupName: "B", TargetSize: 6, ParticipantIDs: participantIDs6}, // 6 players = 5 rounds
		},
	}

	resp, err := svc.ConfirmGroups(ctx, tournamentID, req)
	if err != nil {
		t.Fatalf("ConfirmGroups failed: %v", err)
	}

	if len(resp.Groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(resp.Groups))
	}

	// Check round counts
	for _, g := range resp.Groups {
		expectedRounds := engine.GetTotalGroupRounds(len(g.Players))
		if g.RoundsTotal != expectedRounds {
			t.Errorf("Group %s with %d players: expected %d rounds, got %d",
				g.Name, len(g.Players), expectedRounds, g.RoundsTotal)
		}
	}
}

func TestEngineProposeGroupsSerpentineCorrectness(t *testing.T) {
	// Test the engine directly to verify serpentine assignment
	wam := engine.NewWAM(zerolog.Nop())

	players := make([]engine.Player, 12)
	for i := 0; i < 12; i++ {
		players[i] = engine.Player{
			StartNo: i + 1,
			Name:    "Player " + string(rune('A'+i)),
			Rating:  1200 - i*10, // A=1200, B=1190, ..., L=1090
		}
	}

	config := &engine.WAMConfig{
		DefaultGroupSize:  4,
		AllowedGroupSizes: []int{4, 6},
		YouthAgeLimit:     12,
	}

	proposal, err := wam.ProposeGroups(players, config, nil, time.Now())
	if err != nil {
		t.Fatalf("ProposeGroups failed: %v", err)
	}

	// With serpentine for 3 groups of 4:
	// Row 1: A(1200)->G1, B(1190)->G2, C(1180)->G3
	// Row 2: D(1170)->G3, E(1160)->G2, F(1150)->G1
	// Row 3: G(1140)->G1, H(1130)->G2, I(1120)->G3
	// Row 4: J(1110)->G3, K(1100)->G2, L(1090)->G1
	// Group 1: A, F, G, L (1200+1150+1140+1090 = 4580, avg = 1145)
	// Group 2: B, E, H, K (1190+1160+1130+1100 = 4580, avg = 1145)
	// Group 3: C, D, I, J (1180+1170+1120+1110 = 4580, avg = 1145)

	if len(proposal.Groups) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(proposal.Groups))
	}

	// Calculate average ratings for each group
	avgRatings := make([]float64, 3)
	for i, g := range proposal.Groups {
		sum := 0
		for _, p := range g.Players {
			sum += p.Rating
		}
		avgRatings[i] = float64(sum) / float64(len(g.Players))
	}

	// All groups should have similar averages (within 20 points difference)
	for i := 0; i < len(avgRatings); i++ {
		for j := i + 1; j < len(avgRatings); j++ {
			diff := avgRatings[i] - avgRatings[j]
			if diff < 0 {
				diff = -diff
			}
			if diff > 20 {
				t.Errorf("Serpentine not balanced: Group %d avg=%.1f, Group %d avg=%.1f, diff=%.1f",
					i+1, avgRatings[i], j+1, avgRatings[j], diff)
			}
		}
	}
}

func TestEngineIsU12Player(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		birthDate time.Time
		ageLimit  int
		expected  bool
	}{
		{"10 year old, limit 12", now.AddDate(-10, 0, 0), 12, true},
		{"11 year old, limit 12", now.AddDate(-11, 0, 0), 12, true},
		{"Just turned 12, limit 12", now.AddDate(-12, 0, 0), 12, false},
		{"12 year old, limit 12", now.AddDate(-12, -1, 0), 12, false},
		{"15 year old, limit 12", now.AddDate(-15, 0, 0), 12, false},
		{"10 year old, limit 10", now.AddDate(-10, 0, 0), 10, false},
		{"9 year old, limit 10", now.AddDate(-9, 0, 0), 10, true},
		{"Birthday tomorrow", now.AddDate(-12, 0, 1), 12, true}, // Still 11
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.IsU12Player(&tt.birthDate, now, tt.ageLimit)
			if result != tt.expected {
				t.Errorf("IsU12Player() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestServiceProposeGroupsNoParticipants(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseSetup,
	}

	// No participants
	repo.participants = []TournamentParticipant{}

	_, err := svc.ProposeGroups(ctx, tournamentID)
	if err != ErrNoParticipants {
		t.Errorf("Expected ErrNoParticipants, got %v", err)
	}
}

func TestServiceProposeGroupsWrongPhase(t *testing.T) {
	repo := NewMockRepository()
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()
	tournamentID := uuid.New()

	repo.config = &WAMConfig{
		TournamentID: tournamentID,
		CurrentPhase: engine.WAMPhaseGroup, // Wrong phase
	}

	repo.participants = []TournamentParticipant{
		{ParticipantID: uuid.New(), Name: "Player A", Rating: 1500},
	}

	_, err := svc.ProposeGroups(ctx, tournamentID)
	if err != ErrInvalidPhase {
		t.Errorf("Expected ErrInvalidPhase, got %v", err)
	}
}

// Helper function
func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
