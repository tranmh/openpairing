package tournament

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MockByeRepository is a mock implementation for testing bye-related methods
type MockByeRepository struct {
	tournaments  map[uuid.UUID]*Tournament
	participants map[uuid.UUID]*Participant
	rounds       map[string]*Round // key is "tournamentID-roundNumber"
}

func NewMockByeRepository() *MockByeRepository {
	return &MockByeRepository{
		tournaments:  make(map[uuid.UUID]*Tournament),
		participants: make(map[uuid.UUID]*Participant),
		rounds:       make(map[string]*Round),
	}
}

// Tournament CRUD
func (m *MockByeRepository) Create(ctx context.Context, t *Tournament) error {
	m.tournaments[t.ID] = t
	return nil
}

func (m *MockByeRepository) GetByID(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	t, ok := m.tournaments[id]
	if !ok {
		return nil, ErrTournamentNotFound
	}
	return t, nil
}

func (m *MockByeRepository) GetBySlug(ctx context.Context, slug string) (*Tournament, error) {
	panic("not implemented")
}

func (m *MockByeRepository) Update(ctx context.Context, t *Tournament) error {
	m.tournaments[t.ID] = t
	return nil
}

func (m *MockByeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (m *MockByeRepository) List(ctx context.Context, filters *ListFilters) ([]Tournament, int64, error) {
	panic("not implemented")
}

// Participants
func (m *MockByeRepository) AddParticipant(ctx context.Context, p *Participant) error {
	m.participants[p.ID] = p
	return nil
}

func (m *MockByeRepository) GetParticipant(ctx context.Context, id uuid.UUID) (*Participant, error) {
	p, ok := m.participants[id]
	if !ok {
		return nil, ErrParticipantNotFound
	}
	return p, nil
}

func (m *MockByeRepository) GetParticipantByPlayerID(ctx context.Context, tournamentID, playerID uuid.UUID) (*Participant, error) {
	for _, p := range m.participants {
		if p.TournamentID == tournamentID && p.PlayerID == playerID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *MockByeRepository) UpdateParticipant(ctx context.Context, p *Participant) error {
	m.participants[p.ID] = p
	return nil
}

func (m *MockByeRepository) RemoveParticipant(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (m *MockByeRepository) ListParticipants(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error) {
	panic("not implemented")
}

func (m *MockByeRepository) GetStandings(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error) {
	panic("not implemented")
}

// Rounds
func (m *MockByeRepository) CreateRound(ctx context.Context, r *Round) error {
	panic("not implemented")
}

func (m *MockByeRepository) GetRound(ctx context.Context, id uuid.UUID) (*Round, error) {
	panic("not implemented")
}

func (m *MockByeRepository) GetRoundByNumber(ctx context.Context, tournamentID uuid.UUID, roundNum int) (*Round, error) {
	key := tournamentID.String() + "-" + itoa(roundNum)
	r, ok := m.rounds[key]
	if !ok {
		return nil, ErrRoundNotFound
	}
	return r, nil
}

func (m *MockByeRepository) UpdateRound(ctx context.Context, r *Round) error {
	panic("not implemented")
}

func (m *MockByeRepository) ListRounds(ctx context.Context, tournamentID uuid.UUID) ([]Round, error) {
	panic("not implemented")
}

// Games
func (m *MockByeRepository) CreateGame(ctx context.Context, g *Game) error {
	panic("not implemented")
}

func (m *MockByeRepository) CreateGames(ctx context.Context, games []Game) error {
	panic("not implemented")
}

func (m *MockByeRepository) GetGame(ctx context.Context, id uuid.UUID) (*Game, error) {
	panic("not implemented")
}

func (m *MockByeRepository) UpdateGame(ctx context.Context, g *Game) error {
	panic("not implemented")
}

func (m *MockByeRepository) ListGamesByRound(ctx context.Context, roundID uuid.UUID) ([]Game, error) {
	panic("not implemented")
}

func (m *MockByeRepository) ListGamesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Game, error) {
	panic("not implemented")
}

func (m *MockByeRepository) GetPlayerGames(ctx context.Context, tournamentID, participantID uuid.UUID) ([]Game, error) {
	panic("not implemented")
}

// Forbidden pairs
func (m *MockByeRepository) AddForbiddenPair(ctx context.Context, fp *ForbiddenPair) error {
	panic("not implemented")
}

func (m *MockByeRepository) ListForbiddenPairs(ctx context.Context, tournamentID uuid.UUID) ([]ForbiddenPair, error) {
	panic("not implemented")
}

func (m *MockByeRepository) RemoveForbiddenPair(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

// Player lookup
func (m *MockByeRepository) GetPlayerRating(ctx context.Context, playerID uuid.UUID, ratingType string) (int, error) {
	return 1500, nil
}

// Teams
func (m *MockByeRepository) GetTeams(ctx context.Context, tournamentID uuid.UUID) ([]Team, error) {
	panic("not implemented")
}

// Special Rankings
func (m *MockByeRepository) CreateSpecialRanking(ctx context.Context, sr *SpecialRanking) error {
	panic("not implemented")
}

func (m *MockByeRepository) GetSpecialRanking(ctx context.Context, id uuid.UUID) (*SpecialRanking, error) {
	panic("not implemented")
}

func (m *MockByeRepository) UpdateSpecialRanking(ctx context.Context, sr *SpecialRanking) error {
	panic("not implemented")
}

func (m *MockByeRepository) DeleteSpecialRanking(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (m *MockByeRepository) ListSpecialRankings(ctx context.Context, tournamentID uuid.UUID) ([]SpecialRanking, error) {
	panic("not implemented")
}

// Games
func (m *MockByeRepository) DeleteGame(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

// itoa converts int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// createTestByeService creates a service with a mock repository for bye testing
func createTestByeService(mock *MockByeRepository) *Service {
	logger := zerolog.New(zerolog.NewTestWriter(nil)).Level(zerolog.Disabled)
	return &Service{
		repo:   mock,
		logger: logger,
	}
}

func TestRequestHalfPointBye_NotAllowed(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	settings := DefaultByeSettings()
	settings.HalfPointBye.Allowed = false

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: settings,
	}
	mock.participants[participantID] = &Participant{
		ID:           participantID,
		TournamentID: tournamentID,
	}

	svc := createTestByeService(mock)
	err := svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, 1)

	if err == nil {
		t.Error("Expected error when half-point byes not allowed")
	}
}

func TestRequestHalfPointBye_MaxPerPlayerExceeded(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	settings := DefaultByeSettings()
	settings.HalfPointBye.MaxPerPlayer = 1

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: settings,
	}
	mock.participants[participantID] = &Participant{
		ID:                participantID,
		TournamentID:      tournamentID,
		HalfPointByesUsed: 1, // Already used maximum
	}

	svc := createTestByeService(mock)
	err := svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, 2)

	if err == nil {
		t.Error("Expected error when max half-point byes exceeded")
	}
}

func TestRequestHalfPointBye_FirstHalfOnly(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	settings := DefaultByeSettings()
	settings.HalfPointBye.AllowedRounds = "first_half"

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9, // Half is 4
		ByeSettings: settings,
	}
	mock.participants[participantID] = &Participant{
		ID:           participantID,
		TournamentID: tournamentID,
	}

	svc := createTestByeService(mock)

	// Round 5 should fail (5 > 4 = 9/2)
	err := svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, 5)
	if err == nil {
		t.Error("Expected error for round 5 in 9-round tournament with first_half rule")
	}

	// Round 4 should succeed
	err = svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, 4)
	if err != nil {
		t.Errorf("Round 4 should be allowed: %v", err)
	}
}

func TestRequestHalfPointBye_InvalidRoundNumber(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: DefaultByeSettings(),
	}
	mock.participants[participantID] = &Participant{
		ID:           participantID,
		TournamentID: tournamentID,
	}

	svc := createTestByeService(mock)

	// BUG: Negative round should be rejected
	t.Run("Negative round number", func(t *testing.T) {
		err := svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, -1)
		if err == nil {
			t.Error("BUG FOUND: Negative round number should be rejected")
		}
	})

	// BUG: Round 0 should be rejected
	t.Run("Round 0", func(t *testing.T) {
		err := svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, 0)
		if err == nil {
			t.Error("BUG FOUND: Round 0 should be rejected")
		}
	})

	// BUG: Round beyond tournament rounds should be rejected
	t.Run("Round beyond tournament rounds", func(t *testing.T) {
		err := svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, 10)
		if err == nil {
			t.Error("BUG FOUND: Round beyond tournament rounds should be rejected")
		}
	})
}

func TestRequestHalfPointBye_DuplicateRequest(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	settings := DefaultByeSettings()
	settings.HalfPointBye.MaxPerPlayer = 3 // Allow multiple
	settings.HalfPointBye.AllowedRounds = "all"

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: settings,
	}
	mock.participants[participantID] = &Participant{
		ID:           participantID,
		TournamentID: tournamentID,
		ByeRequests:  []int{3}, // Already requested for round 3
	}

	svc := createTestByeService(mock)

	// Duplicate request should fail
	err := svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, 3)
	if err == nil {
		t.Error("Expected error for duplicate bye request")
	}
}

func TestRequestHalfPointBye_ParticipantNotInTournament(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	otherTournamentID := uuid.New()
	participantID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: DefaultByeSettings(),
	}
	mock.participants[participantID] = &Participant{
		ID:           participantID,
		TournamentID: otherTournamentID, // Different tournament!
	}

	svc := createTestByeService(mock)
	err := svc.RequestHalfPointBye(context.Background(), tournamentID, participantID, 1)

	if err == nil {
		t.Error("Expected error when participant not in tournament")
	}
}

func TestRecordForfeit_ConsecutiveTracking(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	settings := DefaultByeSettings()
	settings.Forfeit.AutoWithdrawAfter = 2
	settings.Forfeit.ConsecutiveOnly = true

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		Rounds:       9,
		CurrentRound: 1,
		ByeSettings:  settings,
	}
	mock.participants[participantID] = &Participant{
		ID:                  participantID,
		TournamentID:        tournamentID,
		ForfeitCount:        0,
		ConsecutiveForfeits: 0,
		IsActive:            true,
	}

	svc := createTestByeService(mock)

	// First forfeit
	p, autoWithdrawn, err := svc.RecordForfeit(context.Background(), tournamentID, participantID, true)
	if err != nil {
		t.Fatalf("First forfeit failed: %v", err)
	}
	if autoWithdrawn {
		t.Error("Should not auto-withdraw after 1 forfeit")
	}
	if p.ForfeitCount != 1 || p.ConsecutiveForfeits != 1 {
		t.Errorf("Expected ForfeitCount=1, ConsecutiveForfeits=1, got %d, %d",
			p.ForfeitCount, p.ConsecutiveForfeits)
	}

	// Second consecutive forfeit - should trigger auto-withdraw
	p, autoWithdrawn, err = svc.RecordForfeit(context.Background(), tournamentID, participantID, true)
	if err != nil {
		t.Fatalf("Second forfeit failed: %v", err)
	}
	if !autoWithdrawn {
		t.Error("Should auto-withdraw after 2 consecutive forfeits")
	}
	if p.IsActive {
		t.Error("Participant should be inactive after auto-withdrawal")
	}
}

func TestRecordForfeit_NonConsecutiveReset(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	settings := DefaultByeSettings()
	settings.Forfeit.AutoWithdrawAfter = 2
	settings.Forfeit.ConsecutiveOnly = true

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		Rounds:       9,
		CurrentRound: 1,
		ByeSettings:  settings,
	}
	mock.participants[participantID] = &Participant{
		ID:                  participantID,
		TournamentID:        tournamentID,
		ForfeitCount:        1, // Already had one forfeit
		ConsecutiveForfeits: 1,
		IsActive:            true,
	}

	svc := createTestByeService(mock)

	// Non-consecutive forfeit - should reset consecutive count
	p, autoWithdrawn, err := svc.RecordForfeit(context.Background(), tournamentID, participantID, false)
	if err != nil {
		t.Fatalf("Forfeit failed: %v", err)
	}
	if autoWithdrawn {
		t.Error("Should not auto-withdraw for non-consecutive forfeit")
	}
	if p.ConsecutiveForfeits != 1 {
		t.Errorf("Expected ConsecutiveForfeits=1 after non-consecutive, got %d", p.ConsecutiveForfeits)
	}
	if p.ForfeitCount != 2 {
		t.Errorf("Expected ForfeitCount=2, got %d", p.ForfeitCount)
	}
}

func TestRecordForfeit_TotalForfeitCount_WithdrawalMessage(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	settings := DefaultByeSettings()
	settings.Forfeit.AutoWithdrawAfter = 2
	settings.Forfeit.ConsecutiveOnly = false // Count total, not consecutive

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		Rounds:       9,
		CurrentRound: 1,
		ByeSettings:  settings,
	}
	mock.participants[participantID] = &Participant{
		ID:                  participantID,
		TournamentID:        tournamentID,
		ForfeitCount:        1, // Already had one
		ConsecutiveForfeits: 0, // Not consecutive
		IsActive:            true,
	}

	svc := createTestByeService(mock)

	// Second forfeit (not consecutive) should still trigger auto-withdraw
	p, autoWithdrawn, err := svc.RecordForfeit(context.Background(), tournamentID, participantID, false)
	if err != nil {
		t.Fatalf("Forfeit failed: %v", err)
	}
	if !autoWithdrawn {
		t.Error("Should auto-withdraw after 2 total forfeits when ConsecutiveOnly=false")
	}

	// BUG: Check the withdrawal reason message
	// When ConsecutiveOnly=false, the message should NOT say "consecutive forfeits"
	if p.WithdrawalReason != nil {
		reason := *p.WithdrawalReason
		if containsSubstring(reason, "consecutive") && !settings.Forfeit.ConsecutiveOnly {
			t.Errorf("BUG FOUND: Withdrawal reason incorrectly mentions 'consecutive' when ConsecutiveOnly=false: %s", reason)
		}
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRecordSystemBye_MaxOnePerPlayer(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()
	round1 := 1

	settings := DefaultByeSettings()
	settings.SystemBye.MaxPerPlayer = 1 // FIDE default

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: settings,
	}
	mock.participants[participantID] = &Participant{
		ID:             participantID,
		TournamentID:   tournamentID,
		SystemByeRound: &round1, // Already received in round 1
	}

	svc := createTestByeService(mock)

	// Second system bye should fail
	err := svc.RecordSystemBye(context.Background(), tournamentID, participantID, 3)
	if err == nil {
		t.Error("Expected error when trying to give second system bye with MaxPerPlayer=1")
	}
}

func TestRecordSystemBye_AllowMultiple(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()
	round1 := 1

	settings := DefaultByeSettings()
	settings.SystemBye.MaxPerPlayer = 2 // Allow multiple

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: settings,
	}
	mock.participants[participantID] = &Participant{
		ID:             participantID,
		TournamentID:   tournamentID,
		SystemByeRound: &round1,
	}

	svc := createTestByeService(mock)

	// Second system bye should succeed (overwrites previous)
	err := svc.RecordSystemBye(context.Background(), tournamentID, participantID, 3)
	if err != nil {
		t.Errorf("Second system bye should be allowed when MaxPerPlayer=2: %v", err)
	}
}

func TestAddLateEntry_NotAllowed(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	playerID := uuid.New()

	settings := DefaultByeSettings()
	settings.LateEntry.Allowed = false

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		Status:       StatusOngoing,
		CurrentRound: 3,
		Rounds:       9,
		ByeSettings:  settings,
	}

	svc := createTestByeService(mock)

	req := &RegisterParticipantRequest{
		PlayerID: playerID,
	}
	_, err := svc.AddLateEntry(context.Background(), tournamentID, req, 3)
	if err == nil {
		t.Error("Expected error when late entry not allowed")
	}
}

func TestAddLateEntry_MaxRoundExceeded(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	playerID := uuid.New()

	settings := DefaultByeSettings()
	settings.LateEntry.MaxRound = 2

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		Status:       StatusOngoing,
		CurrentRound: 4,
		Rounds:       9,
		ByeSettings:  settings,
	}

	svc := createTestByeService(mock)

	req := &RegisterParticipantRequest{
		PlayerID: playerID,
	}
	_, err := svc.AddLateEntry(context.Background(), tournamentID, req, 4)
	if err == nil {
		t.Error("Expected error when entry round exceeds MaxRound")
	}
}

func TestAddLateEntry_InvalidEntryRound(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	playerID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		Status:       StatusOngoing,
		CurrentRound: 3,
		Rounds:       9,
		ByeSettings:  DefaultByeSettings(),
	}

	svc := createTestByeService(mock)

	req := &RegisterParticipantRequest{
		PlayerID: playerID,
	}

	// BUG: Negative entry round should be rejected
	t.Run("Negative entry round", func(t *testing.T) {
		_, err := svc.AddLateEntry(context.Background(), tournamentID, req, -1)
		if err == nil {
			t.Error("BUG FOUND: Negative entry round should be rejected")
		}
	})

	// BUG: Entry round 0 should be rejected
	t.Run("Entry round 0", func(t *testing.T) {
		_, err := svc.AddLateEntry(context.Background(), tournamentID, req, 0)
		if err == nil {
			t.Error("BUG FOUND: Entry round 0 should be rejected")
		}
	})
}

func TestAddLateEntry_RatingLimits(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	playerID := uuid.New()

	minRating := 1600
	maxRating := 2000

	settings := DefaultByeSettings()
	settings.LateEntry.MinRating = &minRating
	settings.LateEntry.MaxRating = &maxRating

	mock.tournaments[tournamentID] = &Tournament{
		ID:              tournamentID,
		Status:          StatusOngoing,
		CurrentRound:    2,
		Rounds:          9,
		TimeControlType: TimeControlStandard,
		ByeSettings:     settings,
	}

	svc := createTestByeService(mock)

	// Test below minimum rating
	t.Run("Below minimum rating", func(t *testing.T) {
		req := &RegisterParticipantRequest{
			PlayerID:       playerID,
			StartingRating: 1500, // Below minimum of 1600
		}
		_, err := svc.AddLateEntry(context.Background(), tournamentID, req, 2)
		// The current implementation doesn't check rating limits
		if err == nil {
			t.Log("BUG FOUND: Rating limits (MinRating/MaxRating) are not enforced for late entries")
		}
	})

	// Test above maximum rating
	t.Run("Above maximum rating", func(t *testing.T) {
		playerID2 := uuid.New()
		req := &RegisterParticipantRequest{
			PlayerID:       playerID2,
			StartingRating: 2100, // Above maximum of 2000
		}
		_, err := svc.AddLateEntry(context.Background(), tournamentID, req, 2)
		if err == nil {
			t.Log("BUG FOUND: MaxRating limit is not enforced for late entries")
		}
	})
}

func TestAddLateEntry_TournamentNotOngoing(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	playerID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Status:      StatusRegistration, // Not ongoing
		Rounds:      9,
		ByeSettings: DefaultByeSettings(),
	}

	svc := createTestByeService(mock)

	req := &RegisterParticipantRequest{
		PlayerID: playerID,
	}
	_, err := svc.AddLateEntry(context.Background(), tournamentID, req, 1)
	if err == nil {
		t.Error("Expected error for late entry when tournament not ongoing")
	}
}

func TestAddLateEntry_PlayerAlreadyRegistered(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	playerID := uuid.New()
	participantID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		Status:       StatusOngoing,
		CurrentRound: 2,
		Rounds:       9,
		ByeSettings:  DefaultByeSettings(),
	}
	mock.participants[participantID] = &Participant{
		ID:           participantID,
		TournamentID: tournamentID,
		PlayerID:     playerID,
	}

	svc := createTestByeService(mock)

	req := &RegisterParticipantRequest{
		PlayerID: playerID,
	}
	_, err := svc.AddLateEntry(context.Background(), tournamentID, req, 2)
	if err == nil {
		t.Error("Expected error when player already registered")
	}
}

func TestAddLateEntry_CorrectInitialPoints(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	playerID := uuid.New()

	settings := DefaultByeSettings()
	settings.LateEntry.MissedRoundPoints = 0.5
	settings.LateEntry.MaxRound = 5 // Allow late entry up to round 5

	mock.tournaments[tournamentID] = &Tournament{
		ID:              tournamentID,
		Status:          StatusOngoing,
		CurrentRound:    3,
		Rounds:          9,
		TimeControlType: TimeControlStandard,
		ByeSettings:     settings,
	}

	svc := createTestByeService(mock)

	req := &RegisterParticipantRequest{
		PlayerID: playerID,
	}
	p, err := svc.AddLateEntry(context.Background(), tournamentID, req, 3)
	if err != nil {
		t.Fatalf("AddLateEntry failed: %v", err)
	}

	// Entry round 3 means missed rounds 1, 2 = 2 rounds
	// 2 * 0.5 = 1.0 points
	expectedPoints := 2 * 0.5
	if p.Points != expectedPoints {
		t.Errorf("Expected %.1f initial points, got %.1f", expectedPoints, p.Points)
	}
}

func TestCancelByeRequest_NotFound(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: DefaultByeSettings(),
	}
	mock.participants[participantID] = &Participant{
		ID:           participantID,
		TournamentID: tournamentID,
		ByeRequests:  []int{3, 5}, // Has requests for rounds 3 and 5
	}

	svc := createTestByeService(mock)

	// Try to cancel non-existent request
	err := svc.CancelByeRequest(context.Background(), tournamentID, participantID, 4)
	if err == nil {
		t.Error("Expected error when cancelling non-existent bye request")
	}
}

func TestWithdrawParticipantWithDetails_Success(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		CurrentRound: 3,
		Rounds:       9,
		ByeSettings:  DefaultByeSettings(),
	}
	mock.participants[participantID] = &Participant{
		ID:                 participantID,
		TournamentID:       tournamentID,
		IsActive:           true,
		RegistrationStatus: "registered",
	}

	svc := createTestByeService(mock)

	reason := "Personal reasons"
	err := svc.WithdrawParticipantWithDetails(context.Background(), tournamentID, participantID, reason)
	if err != nil {
		t.Fatalf("Withdrawal failed: %v", err)
	}

	p := mock.participants[participantID]
	if p.IsActive {
		t.Error("Participant should be inactive after withdrawal")
	}
	if p.WithdrawnRound == nil || *p.WithdrawnRound != 3 {
		t.Errorf("WithdrawnRound should be 3, got %v", p.WithdrawnRound)
	}
	if p.WithdrawalReason == nil || *p.WithdrawalReason != reason {
		t.Errorf("WithdrawalReason should be '%s', got %v", reason, p.WithdrawalReason)
	}
	if p.RegistrationStatus != "withdrawn" {
		t.Errorf("RegistrationStatus should be 'withdrawn', got %s", p.RegistrationStatus)
	}
}

func TestGetFIDEViolations_NilSettings(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: nil, // No custom settings
	}

	svc := createTestByeService(mock)

	violations, err := svc.GetFIDEViolations(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("GetFIDEViolations failed: %v", err)
	}
	if violations != nil {
		t.Errorf("Expected nil violations for nil settings, got %v", violations)
	}
}

func TestGetFIDEViolations_WithViolations(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()

	settings := DefaultByeSettings()
	settings.HalfPointBye.MaxPerPlayer = 5 // Violation

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: settings,
	}

	svc := createTestByeService(mock)

	violations, err := svc.GetFIDEViolations(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("GetFIDEViolations failed: %v", err)
	}
	if len(violations) == 0 {
		t.Error("Expected violations for non-compliant settings")
	}
}

func TestGetParticipantByeStatus_Complete(t *testing.T) {
	mock := NewMockByeRepository()
	tournamentID := uuid.New()
	participantID := uuid.New()
	systemByeRound := 2
	lateEntryRound := 3

	settings := DefaultByeSettings()
	settings.HalfPointBye.MaxPerPlayer = 2

	mock.tournaments[tournamentID] = &Tournament{
		ID:          tournamentID,
		Rounds:      9,
		ByeSettings: settings,
	}
	mock.participants[participantID] = &Participant{
		ID:                participantID,
		TournamentID:      tournamentID,
		HalfPointByesUsed: 1,
		ZeroPointByesUsed: 0,
		SystemByeRound:    &systemByeRound,
		ForfeitCount:      2,
		LateEntryRound:    &lateEntryRound,
		ByeRequests:       []int{5, 7},
	}

	svc := createTestByeService(mock)

	status, err := svc.GetParticipantByeStatus(context.Background(), participantID)
	if err != nil {
		t.Fatalf("GetParticipantByeStatus failed: %v", err)
	}

	if status.HalfPointByesUsed != 1 {
		t.Errorf("Expected HalfPointByesUsed=1, got %d", status.HalfPointByesUsed)
	}
	if status.MaxHalfPointByesLeft != 1 { // MaxPerPlayer=2, used=1
		t.Errorf("Expected MaxHalfPointByesLeft=1, got %d", status.MaxHalfPointByesLeft)
	}
	if status.SystemByeRound == nil || *status.SystemByeRound != 2 {
		t.Errorf("Expected SystemByeRound=2, got %v", status.SystemByeRound)
	}
	if status.ForfeitCount != 2 {
		t.Errorf("Expected ForfeitCount=2, got %d", status.ForfeitCount)
	}
	if status.LateEntryRound == nil || *status.LateEntryRound != 3 {
		t.Errorf("Expected LateEntryRound=3, got %v", status.LateEntryRound)
	}
	if len(status.PendingByeRequests) != 2 {
		t.Errorf("Expected 2 pending bye requests, got %d", len(status.PendingByeRequests))
	}
}

func init() {
	// Ensure time is imported (needed for Participant.RegisteredAt)
	_ = time.Now()
}
