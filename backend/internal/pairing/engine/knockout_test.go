package engine

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestKnockout_GenerateFirstRoundPairings(t *testing.T) {
	logger := zerolog.Nop()
	ko := NewKnockout(logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		players     []Player
		wantPairs   int
		wantByes    int
	}{
		{
			name: "4 players (power of 2)",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2400, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 2300, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 2200, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 2100, IsActive: true},
			},
			wantPairs: 2,
			wantByes:  0,
		},
		{
			name: "8 players (power of 2)",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2700, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 2650, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 2600, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 2550, IsActive: true},
				{StartNo: 5, Name: "P5", Rating: 2500, IsActive: true},
				{StartNo: 6, Name: "P6", Rating: 2450, IsActive: true},
				{StartNo: 7, Name: "P7", Rating: 2400, IsActive: true},
				{StartNo: 8, Name: "P8", Rating: 2350, IsActive: true},
			},
			wantPairs: 4,
			wantByes:  0,
		},
		{
			name: "3 players (needs bye)",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2500, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 2400, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 2300, IsActive: true},
			},
			wantPairs: 2, // 1 game + 1 bye
			wantByes:  1,
		},
		{
			name: "5 players (needs 3 byes)",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2500, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 2400, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 2300, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 2200, IsActive: true},
				{StartNo: 5, Name: "P5", Rating: 2100, IsActive: true},
			},
			wantPairs: 4, // includes byes
			wantByes:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GeneralPairingRequest{
				TournamentType: TournamentTypeKnockout,
				Players:        tt.players,
				RoundNumber:    1,
				KnockoutFormat: KnockoutSingleElimination,
			}

			result, err := ko.GenerateGeneralPairings(ctx, req)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result.Pairings) != tt.wantPairs {
				t.Errorf("Got %d pairings, want %d", len(result.Pairings), tt.wantPairs)
			}

			byes := 0
			for _, p := range result.Pairings {
				if p.IsBye {
					byes++
				}
			}

			if byes != tt.wantByes {
				t.Errorf("Got %d byes, want %d", byes, tt.wantByes)
			}
		})
	}
}

func TestKnockout_SeededPositions(t *testing.T) {
	logger := zerolog.Nop()
	ko := NewKnockout(logger)

	tests := []struct {
		size int
		want []int
	}{
		{2, []int{1, 2}},
		{4, []int{1, 4, 2, 3}},
		{8, []int{1, 8, 4, 5, 2, 7, 3, 6}},
	}

	for _, tt := range tests {
		seeds := ko.generateSeededPositions(tt.size)

		if len(seeds) != len(tt.want) {
			t.Errorf("generateSeededPositions(%d) length = %d, want %d", tt.size, len(seeds), len(tt.want))
			continue
		}

		for i, s := range seeds {
			if s != tt.want[i] {
				t.Errorf("generateSeededPositions(%d)[%d] = %d, want %d", tt.size, i, s, tt.want[i])
			}
		}
	}
}

func TestKnockout_SubsequentRounds(t *testing.T) {
	logger := zerolog.Nop()
	ko := NewKnockout(logger)
	ctx := context.Background()

	// Simulate semifinal with 4 players
	players := []Player{
		{StartNo: 1, Name: "P1", Rating: 2500, IsActive: true},
		{StartNo: 3, Name: "P3", Rating: 2400, IsActive: true},
		{StartNo: 5, Name: "P5", Rating: 2300, IsActive: true},
		{StartNo: 7, Name: "P7", Rating: 2200, IsActive: true},
	}

	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeKnockout,
		Players:        players,
		RoundNumber:    2,
		KnockoutFormat: KnockoutSingleElimination,
	}

	result, err := ko.GenerateGeneralPairings(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(result.Pairings) != 2 {
		t.Errorf("Got %d pairings, want 2", len(result.Pairings))
	}

	// Verify highest seed plays lowest seed
	// Players sorted by rating: 1, 3, 5, 7
	// Expected: 1 vs 7, 3 vs 5
	found := make(map[[2]int]bool)
	for _, p := range result.Pairings {
		key := [2]int{minInt(p.WhiteStartNo, p.BlackStartNo), maxInt(p.WhiteStartNo, p.BlackStartNo)}
		found[key] = true
	}

	if !found[[2]int{1, 7}] {
		t.Error("Expected pairing 1 vs 7 not found")
	}
	if !found[[2]int{3, 5}] {
		t.Error("Expected pairing 3 vs 5 not found")
	}
}

func TestGetKnockoutTotalRounds(t *testing.T) {
	tests := []struct {
		numPlayers int
		want       int
	}{
		{2, 1},
		{3, 2},
		{4, 2},
		{5, 3},
		{8, 3},
		{16, 4},
		{32, 5},
	}

	for _, tt := range tests {
		got := GetKnockoutTotalRounds(tt.numPlayers)
		if got != tt.want {
			t.Errorf("GetKnockoutTotalRounds(%d) = %d, want %d", tt.numPlayers, got, tt.want)
		}
	}
}

func TestCalculateByesForKnockout(t *testing.T) {
	tests := []struct {
		numPlayers int
		want       int
	}{
		{2, 0},
		{3, 1},
		{4, 0},
		{5, 3},
		{6, 2},
		{7, 1},
		{8, 0},
		{9, 7},
	}

	for _, tt := range tests {
		got := CalculateByesForKnockout(tt.numPlayers)
		if got != tt.want {
			t.Errorf("CalculateByesForKnockout(%d) = %d, want %d", tt.numPlayers, got, tt.want)
		}
	}
}

func TestIsPowerOfTwo(t *testing.T) {
	tests := []struct {
		n    int
		want bool
	}{
		{1, true},
		{2, true},
		{3, false},
		{4, true},
		{5, false},
		{8, true},
		{16, true},
		{17, false},
		{0, false},
		{-1, false},
	}

	for _, tt := range tests {
		got := IsPowerOfTwo(tt.n)
		if got != tt.want {
			t.Errorf("IsPowerOfTwo(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

func TestKnockout_GenerateBracket(t *testing.T) {
	logger := zerolog.Nop()
	ko := NewKnockout(logger)

	players := []Player{
		{StartNo: 1, Name: "P1", Rating: 2700, IsActive: true},
		{StartNo: 2, Name: "P2", Rating: 2650, IsActive: true},
		{StartNo: 3, Name: "P3", Rating: 2600, IsActive: true},
		{StartNo: 4, Name: "P4", Rating: 2550, IsActive: true},
		{StartNo: 5, Name: "P5", Rating: 2500, IsActive: true},
		{StartNo: 6, Name: "P6", Rating: 2450, IsActive: true},
		{StartNo: 7, Name: "P7", Rating: 2400, IsActive: true},
		{StartNo: 8, Name: "P8", Rating: 2350, IsActive: true},
	}

	bracket := ko.GenerateBracket(players)

	// 8 players = 3 rounds
	if len(bracket) != 3 {
		t.Errorf("Got %d rounds in bracket, want 3", len(bracket))
	}

	// First round should have 8 entries
	if len(bracket[0]) != 8 {
		t.Errorf("First round has %d entries, want 8", len(bracket[0]))
	}

	// Second round should have 4 entries
	if len(bracket[1]) != 4 {
		t.Errorf("Second round has %d entries, want 4", len(bracket[1]))
	}

	// Final should have 2 entries
	if len(bracket[2]) != 2 {
		t.Errorf("Final has %d entries, want 2", len(bracket[2]))
	}
}

func TestKnockout_Name(t *testing.T) {
	logger := zerolog.Nop()
	ko := NewKnockout(logger)

	if ko.Name() != "Knockout" {
		t.Errorf("Name() = %s, want Knockout", ko.Name())
	}
}

func TestKnockout_IsAvailable(t *testing.T) {
	logger := zerolog.Nop()
	ko := NewKnockout(logger)
	ctx := context.Background()

	if !ko.IsAvailable(ctx) {
		t.Error("IsAvailable() should return true")
	}
}

func TestKnockout_ErrorCases(t *testing.T) {
	logger := zerolog.Nop()
	ko := NewKnockout(logger)
	ctx := context.Background()

	// Too few players
	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeKnockout,
		Players: []Player{
			{StartNo: 1, Name: "P1", Rating: 2500, IsActive: true},
		},
		RoundNumber: 1,
	}

	_, err := ko.GenerateGeneralPairings(ctx, req)
	if err == nil {
		t.Error("Expected error for too few players")
	}

	// No active players
	req = &GeneralPairingRequest{
		TournamentType: TournamentTypeKnockout,
		Players: []Player{
			{StartNo: 1, Name: "P1", Rating: 2500, IsActive: false},
			{StartNo: 2, Name: "P2", Rating: 2400, IsActive: false},
		},
		RoundNumber: 1,
	}

	_, err = ko.GenerateGeneralPairings(ctx, req)
	if err == nil {
		t.Error("Expected error for no active players")
	}
}
