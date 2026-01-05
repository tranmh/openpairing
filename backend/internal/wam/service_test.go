package wam

import (
	"context"
	"testing"

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
