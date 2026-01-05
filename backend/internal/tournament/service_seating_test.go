package tournament

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MockSeatingRepository extends the mock for seating chart tests
type MockSeatingRepository struct {
	tournaments  map[uuid.UUID]*Tournament
	participants map[uuid.UUID]*Participant
}

func NewMockSeatingRepository() *MockSeatingRepository {
	return &MockSeatingRepository{
		tournaments:  make(map[uuid.UUID]*Tournament),
		participants: make(map[uuid.UUID]*Participant),
	}
}

// Implement Repository interface methods

func (m *MockSeatingRepository) Create(ctx context.Context, t *Tournament) error {
	m.tournaments[t.ID] = t
	return nil
}

func (m *MockSeatingRepository) GetByID(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	t, ok := m.tournaments[id]
	if !ok {
		return nil, ErrTournamentNotFound
	}
	return t, nil
}

func (m *MockSeatingRepository) GetBySlug(ctx context.Context, slug string) (*Tournament, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) Update(ctx context.Context, t *Tournament) error {
	m.tournaments[t.ID] = t
	return nil
}

func (m *MockSeatingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) List(ctx context.Context, filters *ListFilters) ([]Tournament, int64, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) AddParticipant(ctx context.Context, p *Participant) error {
	m.participants[p.ID] = p
	return nil
}

func (m *MockSeatingRepository) GetParticipant(ctx context.Context, id uuid.UUID) (*Participant, error) {
	p, ok := m.participants[id]
	if !ok {
		return nil, ErrParticipantNotFound
	}
	return p, nil
}

func (m *MockSeatingRepository) GetParticipantByPlayerID(ctx context.Context, tournamentID, playerID uuid.UUID) (*Participant, error) {
	for _, p := range m.participants {
		if p.TournamentID == tournamentID && p.PlayerID == playerID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *MockSeatingRepository) UpdateParticipant(ctx context.Context, p *Participant) error {
	m.participants[p.ID] = p
	return nil
}

func (m *MockSeatingRepository) RemoveParticipant(ctx context.Context, id uuid.UUID) error {
	delete(m.participants, id)
	return nil
}

func (m *MockSeatingRepository) ListParticipants(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error) {
	var result []Participant
	for _, p := range m.participants {
		if p.TournamentID == tournamentID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *MockSeatingRepository) GetStandings(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) CreateRound(ctx context.Context, r *Round) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) GetRound(ctx context.Context, id uuid.UUID) (*Round, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) GetRoundByNumber(ctx context.Context, tournamentID uuid.UUID, roundNum int) (*Round, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) UpdateRound(ctx context.Context, r *Round) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) ListRounds(ctx context.Context, tournamentID uuid.UUID) ([]Round, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) CreateGame(ctx context.Context, g *Game) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) CreateGames(ctx context.Context, games []Game) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) GetGame(ctx context.Context, id uuid.UUID) (*Game, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) UpdateGame(ctx context.Context, g *Game) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) DeleteGame(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) ListGamesByRound(ctx context.Context, roundID uuid.UUID) ([]Game, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) ListGamesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Game, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) GetPlayerGames(ctx context.Context, tournamentID, participantID uuid.UUID) ([]Game, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) AddForbiddenPair(ctx context.Context, fp *ForbiddenPair) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) ListForbiddenPairs(ctx context.Context, tournamentID uuid.UUID) ([]ForbiddenPair, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) RemoveForbiddenPair(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) GetPlayerRating(ctx context.Context, playerID uuid.UUID, ratingType string) (int, error) {
	return 1500, nil
}

func (m *MockSeatingRepository) GetTeams(ctx context.Context, tournamentID uuid.UUID) ([]Team, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) CreateSpecialRanking(ctx context.Context, sr *SpecialRanking) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) GetSpecialRanking(ctx context.Context, id uuid.UUID) (*SpecialRanking, error) {
	panic("not implemented")
}

func (m *MockSeatingRepository) UpdateSpecialRanking(ctx context.Context, sr *SpecialRanking) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) DeleteSpecialRanking(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (m *MockSeatingRepository) ListSpecialRankings(ctx context.Context, tournamentID uuid.UUID) ([]SpecialRanking, error) {
	panic("not implemented")
}

func createTestSeatingService(mock *MockSeatingRepository) *Service {
	logger := zerolog.New(zerolog.NewTestWriter(nil)).Level(zerolog.Disabled)
	return &Service{
		repo:   mock,
		logger: logger,
	}
}

// Helper to create a tournament with participants
func setupRoundRobinTournament(mock *MockSeatingRepository, numPlayers int) uuid.UUID {
	tournamentID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:           tournamentID,
		Name:         "Test Round Robin",
		Type:         TypeRoundRobin,
		Rounds:       numPlayers - 1,
		CurrentRound: 1,
		RoundRobinSettings: &RoundRobinSettings{
			ByePosition:      0,
			FixedPlayer:      1,
			ShowSeatingChart: true,
		},
	}

	for i := 1; i <= numPlayers; i++ {
		p := &Participant{
			ID:           uuid.New(),
			TournamentID: tournamentID,
			StartRank:    i,
		}
		firstName := "Player"
		lastName := string(rune('A' + i - 1))
		p.PlayerFirstName = &firstName
		p.PlayerLastName = &lastName
		mock.participants[p.ID] = p
	}

	return tournamentID
}

// Tests

func TestGetSeatingChart_EvenPlayers(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)
	svc := createTestSeatingService(mock)

	chart, err := svc.GetSeatingChart(context.Background(), tournamentID, 1)
	if err != nil {
		t.Fatalf("GetSeatingChart failed: %v", err)
	}

	if chart.NumPlayers != 6 {
		t.Errorf("Expected 6 players, got %d", chart.NumPlayers)
	}
	if chart.NumSeats != 6 {
		t.Errorf("Expected 6 seats for 6 players, got %d", chart.NumSeats)
	}
	if chart.ByeSeat != 0 {
		t.Errorf("Expected no bye seat for even players, got %d", chart.ByeSeat)
	}
	if len(chart.Seats) != 6 {
		t.Errorf("Expected 6 seats, got %d", len(chart.Seats))
	}
	if len(chart.Pairings) != 3 {
		t.Errorf("Expected 3 pairings, got %d", len(chart.Pairings))
	}
}

func TestGetSeatingChart_OddPlayers(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 5)
	svc := createTestSeatingService(mock)

	chart, err := svc.GetSeatingChart(context.Background(), tournamentID, 1)
	if err != nil {
		t.Fatalf("GetSeatingChart failed: %v", err)
	}

	if chart.NumPlayers != 5 {
		t.Errorf("Expected 5 players, got %d", chart.NumPlayers)
	}
	if chart.NumSeats != 6 {
		t.Errorf("Expected 6 seats for 5 players (1 bye), got %d", chart.NumSeats)
	}
	if chart.ByeSeat <= 0 {
		t.Error("Expected bye seat for odd players")
	}

	// Check that one pairing is a bye
	byePairings := 0
	for _, p := range chart.Pairings {
		if p.IsBye {
			byePairings++
		}
	}
	if byePairings != 1 {
		t.Errorf("Expected 1 bye pairing, got %d", byePairings)
	}
}

func TestGetSeatingChart_NotRoundRobin(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:   tournamentID,
		Type: TypeSwiss,
	}

	svc := createTestSeatingService(mock)

	_, err := svc.GetSeatingChart(context.Background(), tournamentID, 1)
	if err == nil {
		t.Error("Expected error for non-round-robin tournament")
	}
}

func TestGetSeatingChart_TooFewParticipants(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:   tournamentID,
		Type: TypeRoundRobin,
	}

	// Add only 1 participant
	p := &Participant{
		ID:           uuid.New(),
		TournamentID: tournamentID,
		StartRank:    1,
	}
	mock.participants[p.ID] = p

	svc := createTestSeatingService(mock)

	_, err := svc.GetSeatingChart(context.Background(), tournamentID, 1)
	if err == nil {
		t.Error("Expected error for too few participants")
	}
}

func TestGetSeatingChart_FixedPlayer(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)
	svc := createTestSeatingService(mock)

	// Check round 1 and round 2 - fixed player should stay at seat 1
	for round := 1; round <= 2; round++ {
		chart, err := svc.GetSeatingChart(context.Background(), tournamentID, round)
		if err != nil {
			t.Fatalf("GetSeatingChart round %d failed: %v", round, err)
		}

		// Find the fixed player seat
		var fixedPlayerSeat *SeatingPosition
		for _, seat := range chart.Seats {
			if seat.IsFixed {
				fixedPlayerSeat = &seat
				break
			}
		}

		if fixedPlayerSeat == nil {
			t.Errorf("Round %d: No fixed player found", round)
			continue
		}

		if fixedPlayerSeat.SeatNumber != 1 {
			t.Errorf("Round %d: Fixed player should be at seat 1, got seat %d", round, fixedPlayerSeat.SeatNumber)
		}
	}
}

func TestGetRotationData_Basic(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)
	svc := createTestSeatingService(mock)

	data, err := svc.GetRotationData(context.Background(), tournamentID, 1, 2)
	if err != nil {
		t.Fatalf("GetRotationData failed: %v", err)
	}

	if data.FromRound != 1 || data.ToRound != 2 {
		t.Errorf("Expected rounds 1->2, got %d->%d", data.FromRound, data.ToRound)
	}

	if len(data.BeforeSeats) != 6 {
		t.Errorf("Expected 6 before seats, got %d", len(data.BeforeSeats))
	}
	if len(data.AfterSeats) != 6 {
		t.Errorf("Expected 6 after seats, got %d", len(data.AfterSeats))
	}
	if len(data.Movements) != 6 {
		t.Errorf("Expected 6 movements, got %d", len(data.Movements))
	}

	// Fixed player should not move
	for _, m := range data.Movements {
		if m.IsFixed {
			if m.FromSeat != m.ToSeat {
				t.Errorf("Fixed player should not move: seat %d -> %d", m.FromSeat, m.ToSeat)
			}
		}
	}
}

func TestGetRotationData_NotRoundRobin(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:   tournamentID,
		Type: TypeSwiss,
	}

	svc := createTestSeatingService(mock)

	_, err := svc.GetRotationData(context.Background(), tournamentID, 1, 2)
	if err == nil {
		t.Error("Expected error for non-round-robin tournament")
	}
}

func TestUpdateRoundRobinSettings_Success(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)
	svc := createTestSeatingService(mock)

	// For even players, byePosition must be 0 (no bye needed)
	newSettings := &RoundRobinSettings{
		ByePosition:      0,
		FixedPlayer:      2,
		ShowSeatingChart: false,
	}

	err := svc.UpdateRoundRobinSettings(context.Background(), tournamentID, newSettings)
	if err != nil {
		t.Fatalf("UpdateRoundRobinSettings failed: %v", err)
	}

	tournament := mock.tournaments[tournamentID]
	if tournament.RoundRobinSettings.ByePosition != 0 {
		t.Errorf("Expected ByePosition=0, got %d", tournament.RoundRobinSettings.ByePosition)
	}
	if tournament.RoundRobinSettings.FixedPlayer != 2 {
		t.Errorf("Expected FixedPlayer=2, got %d", tournament.RoundRobinSettings.FixedPlayer)
	}
	if tournament.RoundRobinSettings.ShowSeatingChart {
		t.Error("Expected ShowSeatingChart=false")
	}
}

func TestUpdateRoundRobinSettings_SuccessWithBye(t *testing.T) {
	// Test with odd players where byePosition is valid
	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 5) // Odd - has bye

	svc := createTestSeatingService(mock)

	newSettings := &RoundRobinSettings{
		ByePosition:      3, // Valid for odd players
		FixedPlayer:      2,
		ShowSeatingChart: true,
	}

	err := svc.UpdateRoundRobinSettings(context.Background(), tournamentID, newSettings)
	if err != nil {
		t.Fatalf("UpdateRoundRobinSettings failed: %v", err)
	}

	tournament := mock.tournaments[tournamentID]
	if tournament.RoundRobinSettings.ByePosition != 3 {
		t.Errorf("Expected ByePosition=3, got %d", tournament.RoundRobinSettings.ByePosition)
	}
}

func TestUpdateRoundRobinSettings_NotRoundRobin(t *testing.T) {
	mock := NewMockSeatingRepository()
	tournamentID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:   tournamentID,
		Type: TypeSwiss,
	}

	svc := createTestSeatingService(mock)

	err := svc.UpdateRoundRobinSettings(context.Background(), tournamentID, &RoundRobinSettings{})
	if err == nil {
		t.Error("Expected error for non-round-robin tournament")
	}
}

func TestCalculateSeatingForRound_Round1(t *testing.T) {
	// Round 1 should have players in starting positions
	result := calculateSeatingForRound(6, 1, 1)

	for seat := 1; seat <= 6; seat++ {
		if result[seat] != seat {
			t.Errorf("Round 1, seat %d should have player %d, got %d", seat, seat, result[seat])
		}
	}
}

func TestCalculateSeatingForRound_Rotation(t *testing.T) {
	// After rotation, player 1 should stay fixed, others rotate
	resultR1 := calculateSeatingForRound(6, 1, 1)
	resultR2 := calculateSeatingForRound(6, 2, 1)

	// Player 1 should stay at seat 1
	if resultR1[1] != 1 || resultR2[1] != 1 {
		t.Error("Player 1 should always be at seat 1")
	}

	// Others should have rotated
	sameCount := 0
	for seat := 2; seat <= 6; seat++ {
		if resultR1[seat] == resultR2[seat] {
			sameCount++
		}
	}
	if sameCount == 5 {
		t.Error("Non-fixed players should rotate between rounds")
	}
}

func TestCalculateSeatingForRound_FullCycle(t *testing.T) {
	// After numSeats-1 rounds, should return to starting position
	numSeats := 6
	numRounds := numSeats - 1

	resultR1 := calculateSeatingForRound(numSeats, 1, 1)
	resultRN := calculateSeatingForRound(numSeats, numRounds+1, 1)

	for seat := 1; seat <= numSeats; seat++ {
		if resultR1[seat] != resultRN[seat] {
			t.Errorf("After full cycle, seat %d should have player %d, got %d", seat, resultR1[seat], resultRN[seat])
		}
	}
}

func TestGetEffectiveByePosition(t *testing.T) {
	tests := []struct {
		name       string
		settings   *RoundRobinSettings
		numPlayers int
		expected   int
	}{
		{
			name: "Auto bye position (5 players)",
			settings: &RoundRobinSettings{
				ByePosition: 0,
				FixedPlayer: 1,
			},
			numPlayers: 5,
			expected:   4, // Opposite to player 1 in 6-seat layout
		},
		{
			name: "Explicit bye position",
			settings: &RoundRobinSettings{
				ByePosition: 3,
				FixedPlayer: 1,
			},
			numPlayers: 5,
			expected:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.settings.GetEffectiveByePosition(tt.numPlayers)
			if result != tt.expected {
				t.Errorf("Expected bye position %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestDefaultRoundRobinSettings(t *testing.T) {
	settings := DefaultRoundRobinSettings()

	if settings.ByePosition != 0 {
		t.Errorf("Default ByePosition should be 0 (auto), got %d", settings.ByePosition)
	}
	if settings.FixedPlayer != 1 {
		t.Errorf("Default FixedPlayer should be 1, got %d", settings.FixedPlayer)
	}
	if !settings.ShowSeatingChart {
		t.Error("Default ShowSeatingChart should be true")
	}
}

// Tests for fixed player behavior

func TestCalculateSeatingForRound_FixedPlayerNotOne(t *testing.T) {
	// Verify that specifying a non-default fixed player works correctly

	numSeats := 6
	fixedPlayer := 3 // We want player 3 to stay fixed

	resultR1 := calculateSeatingForRound(numSeats, 1, fixedPlayer)
	resultR2 := calculateSeatingForRound(numSeats, 2, fixedPlayer)

	// Find where player 3 sits in round 1 and round 2
	var player3SeatR1, player3SeatR2 int
	for seat, startNo := range resultR1 {
		if startNo == 3 {
			player3SeatR1 = seat
		}
	}
	for seat, startNo := range resultR2 {
		if startNo == 3 {
			player3SeatR2 = seat
		}
	}

	// Player 3 should stay at the same seat between rounds (it's the fixed player)
	if player3SeatR1 != player3SeatR2 {
		t.Errorf("Fixed player 3 should stay at same seat. Round 1 seat: %d, Round 2 seat: %d",
			player3SeatR1, player3SeatR2)
	}

	// Player 3 should be at seat 1 (by design, fixed player always at seat 1)
	if player3SeatR1 != 1 {
		t.Errorf("Fixed player should be at seat 1, but player 3 is at seat %d", player3SeatR1)
	}
}

func TestGetSeatingChart_RoundExceedsTournament(t *testing.T) {
	// Validates that roundNumber is checked against tournament.Rounds
	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)
	mock.tournaments[tournamentID].Rounds = 5 // 5 rounds total

	svc := createTestSeatingService(mock)

	// Request round 99 - should error
	_, err := svc.GetSeatingChart(context.Background(), tournamentID, 99)
	if err == nil {
		t.Error("Expected error for round 99 in a 5-round tournament")
	}
}

func TestCalculateSeatingForRound_TwoPlayers(t *testing.T) {
	// With exactly 2 players (numSeats=2), player 1 stays fixed, player 2 stays fixed too
	// (no meaningful rotation with only 2 seats where 1 is fixed)
	// Color alternation is handled by the pairing logic, not seat rotation

	numSeats := 2
	resultR1 := calculateSeatingForRound(numSeats, 1, 1)
	resultR2 := calculateSeatingForRound(numSeats, 2, 1)

	// Player 1 should always be at seat 1 (fixed)
	if resultR1[1] != 1 {
		t.Errorf("Round 1: Player 1 should be at seat 1, got player %d", resultR1[1])
	}
	if resultR2[1] != 1 {
		t.Errorf("Round 2: Player 1 should be at seat 1, got player %d", resultR2[1])
	}

	// Player 2 should always be at seat 2 (only other option)
	if resultR1[2] != 2 {
		t.Errorf("Round 1: Player 2 should be at seat 2, got player %d", resultR1[2])
	}
	if resultR2[2] != 2 {
		t.Errorf("Round 2: Player 2 should be at seat 2, got player %d", resultR2[2])
	}
}

func TestGetSeatingChart_ColorAlternation(t *testing.T) {
	// Verify that the fixed player alternates colors between rounds

	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)

	// Set fixed player to 3 instead of 1
	mock.tournaments[tournamentID].RoundRobinSettings.FixedPlayer = 3

	svc := createTestSeatingService(mock)

	chart1, err := svc.GetSeatingChart(context.Background(), tournamentID, 1)
	if err != nil {
		t.Fatalf("Round 1: %v", err)
	}
	chart2, err := svc.GetSeatingChart(context.Background(), tournamentID, 2)
	if err != nil {
		t.Fatalf("Round 2: %v", err)
	}

	// Find player 3's games in both rounds
	var player3ColorR1, player3ColorR2 string
	for _, p := range chart1.Pairings {
		if p.WhiteStartNo == 3 {
			player3ColorR1 = "white"
		} else if p.BlackStartNo == 3 {
			player3ColorR1 = "black"
		}
	}
	for _, p := range chart2.Pairings {
		if p.WhiteStartNo == 3 {
			player3ColorR2 = "white"
		} else if p.BlackStartNo == 3 {
			player3ColorR2 = "black"
		}
	}

	// The fixed player should alternate colors between rounds
	if player3ColorR1 == player3ColorR2 && player3ColorR1 != "" {
		t.Errorf("Fixed player 3 should alternate colors: R1=%s, R2=%s", player3ColorR1, player3ColorR2)
	}
}

func TestGetSeatingChart_FixedPlayerSeat(t *testing.T) {
	// Verify that FixedSeat correctly reports where the fixed player sits

	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)
	mock.tournaments[tournamentID].RoundRobinSettings.FixedPlayer = 3

	svc := createTestSeatingService(mock)
	chart, err := svc.GetSeatingChart(context.Background(), tournamentID, 1)
	if err != nil {
		t.Fatalf("GetSeatingChart failed: %v", err)
	}

	// Find which seat has player 3
	var player3Seat int
	for _, seat := range chart.Seats {
		if seat.StartNumber == 3 {
			player3Seat = seat.SeatNumber
			break
		}
	}

	// FixedSeat should be the seat where the fixed player sits
	if chart.FixedSeat != player3Seat {
		t.Errorf("FixedSeat is %d but player 3 (the fixed player) is at seat %d",
			chart.FixedSeat, player3Seat)
	}

	// The fixed player should always be at seat 1 (by design)
	if player3Seat != 1 {
		t.Errorf("Fixed player should always be at seat 1, but player 3 is at seat %d", player3Seat)
	}
}

// ============ Validation Tests ============

func TestGetRotationData_Validation(t *testing.T) {
	// Validates that GetRotationData properly validates fromRound/toRound

	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)
	mock.tournaments[tournamentID].Rounds = 5

	svc := createTestSeatingService(mock)

	t.Run("Negative fromRound should error", func(t *testing.T) {
		_, err := svc.GetRotationData(context.Background(), tournamentID, -1, 2)
		if err == nil {
			t.Error("Expected error for negative fromRound")
		}
	})

	t.Run("Zero fromRound should error", func(t *testing.T) {
		_, err := svc.GetRotationData(context.Background(), tournamentID, 0, 2)
		if err == nil {
			t.Error("Expected error for zero fromRound")
		}
	})

	t.Run("fromRound > toRound should error", func(t *testing.T) {
		_, err := svc.GetRotationData(context.Background(), tournamentID, 5, 2)
		if err == nil {
			t.Error("Expected error for fromRound > toRound")
		}
	})

	t.Run("Round exceeding tournament rounds should error", func(t *testing.T) {
		_, err := svc.GetRotationData(context.Background(), tournamentID, 1, 99)
		if err == nil {
			t.Error("Expected error for round exceeding tournament rounds")
		}
	})
}

func TestUpdateRoundRobinSettings_Validation(t *testing.T) {
	// Validates that UpdateRoundRobinSettings properly validates settings

	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 6)

	svc := createTestSeatingService(mock)

	t.Run("FixedPlayer > numPlayers should error", func(t *testing.T) {
		settings := &RoundRobinSettings{
			FixedPlayer:      99, // Invalid - only 6 players
			ByePosition:      0,
			ShowSeatingChart: true,
		}
		err := svc.UpdateRoundRobinSettings(context.Background(), tournamentID, settings)
		if err == nil {
			t.Error("Expected error for fixedPlayer > numPlayers")
		}
	})

	t.Run("FixedPlayer = 0 should error", func(t *testing.T) {
		settings := &RoundRobinSettings{
			FixedPlayer:      0, // Invalid
			ByePosition:      0,
			ShowSeatingChart: true,
		}
		err := svc.UpdateRoundRobinSettings(context.Background(), tournamentID, settings)
		if err == nil {
			t.Error("Expected error for fixedPlayer = 0")
		}
	})

	t.Run("Negative ByePosition should error", func(t *testing.T) {
		settings := &RoundRobinSettings{
			FixedPlayer:      1,
			ByePosition:      -5, // Invalid
			ShowSeatingChart: true,
		}
		err := svc.UpdateRoundRobinSettings(context.Background(), tournamentID, settings)
		if err == nil {
			t.Error("Expected error for negative byePosition")
		}
	})
}

func TestUpdateRoundRobinSettings_WrongTournamentType(t *testing.T) {
	// Validates that round robin settings can only be set on round robin tournaments

	mock := NewMockSeatingRepository()
	tournamentID := uuid.New()

	mock.tournaments[tournamentID] = &Tournament{
		ID:     tournamentID,
		Type:   TypeSwiss, // NOT round robin!
		Rounds: 9,
	}

	svc := createTestSeatingService(mock)

	// Updating settings on a Swiss tournament should fail
	settings := &RoundRobinSettings{
		FixedPlayer:      1,
		ByePosition:      0,
		ShowSeatingChart: true,
	}
	err := svc.UpdateRoundRobinSettings(context.Background(), tournamentID, settings)
	if err == nil {
		t.Error("Expected error when setting round robin settings on Swiss tournament")
	}
}

func TestGetSeatingChart_ByeSeatPerRound(t *testing.T) {
	// Validates that byeSeat is calculated per round (bye moves with rotation)

	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 5) // Odd - has bye
	mock.tournaments[tournamentID].Rounds = 4

	svc := createTestSeatingService(mock)

	chart1, err := svc.GetSeatingChart(context.Background(), tournamentID, 1)
	if err != nil {
		t.Fatalf("Round 1 failed: %v", err)
	}

	chart2, err := svc.GetSeatingChart(context.Background(), tournamentID, 2)
	if err != nil {
		t.Fatalf("Round 2 failed: %v", err)
	}

	// Find actual bye seat (where startNumber > numPlayers) in each round
	var actualByeSeatR1, actualByeSeatR2 int
	for _, seat := range chart1.Seats {
		if seat.IsBye {
			actualByeSeatR1 = seat.SeatNumber
		}
	}
	for _, seat := range chart2.Seats {
		if seat.IsBye {
			actualByeSeatR2 = seat.SeatNumber
		}
	}

	// The bye should move between rounds (it rotates with others)
	if actualByeSeatR1 == actualByeSeatR2 {
		t.Logf("Note: Bye is at seat %d in both rounds (may be intentional if bye is fixed)", actualByeSeatR1)
	}

	// byeSeat in response should match where the bye actually IS in each round
	if chart1.ByeSeat != actualByeSeatR1 {
		t.Errorf("Round 1: ByeSeat=%d but actual bye is at seat %d", chart1.ByeSeat, actualByeSeatR1)
	}
	if chart2.ByeSeat != actualByeSeatR2 {
		t.Errorf("Round 2: ByeSeat=%d but actual bye is at seat %d", chart2.ByeSeat, actualByeSeatR2)
	}
}

func TestGetSeatingChart_ByePairingCorrectness(t *testing.T) {
	// Validates that bye pairings correctly identify the player with the bye

	mock := NewMockSeatingRepository()
	tournamentID := setupRoundRobinTournament(mock, 5)

	svc := createTestSeatingService(mock)

	// Check multiple rounds to find one where bye is on white side
	for round := 1; round <= 4; round++ {
		chart, err := svc.GetSeatingChart(context.Background(), tournamentID, round)
		if err != nil {
			t.Fatalf("Round %d failed: %v", round, err)
		}

		for _, p := range chart.Pairings {
			if p.IsBye {
				// For a bye pairing, exactly one of white/black should be > numPlayers
				whiteIsBye := p.WhiteStartNo > chart.NumPlayers
				blackIsBye := p.BlackStartNo > chart.NumPlayers

				if whiteIsBye && blackIsBye {
					t.Errorf("Round %d Board %d: Both players are bye?!", round, p.Board)
				}
				if !whiteIsBye && !blackIsBye {
					t.Errorf("Round %d Board %d: Marked as bye but neither player is bye", round, p.Board)
				}

				// The non-bye player should have a valid name or start number
				var realPlayerStartNo int
				if whiteIsBye {
					realPlayerStartNo = p.BlackStartNo
				} else {
					realPlayerStartNo = p.WhiteStartNo
				}

				if realPlayerStartNo < 1 || realPlayerStartNo > chart.NumPlayers {
					t.Errorf("Round %d Board %d: Real player has invalid startNo=%d",
						round, p.Board, realPlayerStartNo)
				}
			}
		}
	}
}
