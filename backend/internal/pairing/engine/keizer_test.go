package engine

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestKeizer_GenerateGeneralPairings(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)
	ctx := context.Background()

	tests := []struct {
		name      string
		players   []Player
		wantPairs int
		wantErr   bool
	}{
		{
			name: "4 players",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2000, Points: 0, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 1900, Points: 0, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 1800, Points: 0, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 1700, Points: 0, IsActive: true},
			},
			wantPairs: 2,
			wantErr:   false,
		},
		{
			name: "6 players with points",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2000, Points: 10.0, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 1900, Points: 8.0, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 1800, Points: 6.0, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 1700, Points: 4.0, IsActive: true},
				{StartNo: 5, Name: "P5", Rating: 1600, Points: 2.0, IsActive: true},
				{StartNo: 6, Name: "P6", Rating: 1500, Points: 0.0, IsActive: true},
			},
			wantPairs: 3,
			wantErr:   false,
		},
		{
			name: "too few players",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2000, IsActive: true},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GeneralPairingRequest{
				TournamentType: TournamentTypeSwiss,
				Players:        tt.players,
				RoundNumber:    1,
				PairingSystem:  "keizer",
			}

			result, err := k.GenerateGeneralPairings(ctx, req)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result.Pairings) != tt.wantPairs {
				t.Errorf("Got %d pairings, want %d", len(result.Pairings), tt.wantPairs)
			}

			// Verify each player appears at most once
			seen := make(map[int]bool)
			for _, p := range result.Pairings {
				if p.IsBye {
					if seen[p.WhiteStartNo] {
						t.Errorf("Player %d appears multiple times", p.WhiteStartNo)
					}
					seen[p.WhiteStartNo] = true
				} else {
					if seen[p.WhiteStartNo] {
						t.Errorf("Player %d appears multiple times", p.WhiteStartNo)
					}
					if seen[p.BlackStartNo] {
						t.Errorf("Player %d appears multiple times", p.BlackStartNo)
					}
					seen[p.WhiteStartNo] = true
					seen[p.BlackStartNo] = true
				}
			}
		})
	}
}

func TestKeizer_CalculateRanking(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)

	players := []KeizerPlayer{
		{Player: Player{StartNo: 1, Rating: 2000}, KeizerPoints: 10.0, GamesPlayed: 5},
		{Player: Player{StartNo: 2, Rating: 1900}, KeizerPoints: 12.0, GamesPlayed: 6},
		{Player: Player{StartNo: 3, Rating: 1800}, KeizerPoints: 10.0, GamesPlayed: 6},
		{Player: Player{StartNo: 4, Rating: 1700}, KeizerPoints: 8.0, GamesPlayed: 5},
	}

	k.calculateRanking(players)

	// Expected order: P2 (12pts), P3 (10pts, 6 games), P1 (10pts, 5 games), P4 (8pts)
	expectedRanks := map[int]int{
		2: 1,
		3: 2,
		1: 3,
		4: 4,
	}

	for _, p := range players {
		if p.KeizerRank != expectedRanks[p.StartNo] {
			t.Errorf("Player %d has rank %d, expected %d", p.StartNo, p.KeizerRank, expectedRanks[p.StartNo])
		}
	}
}

func TestKeizer_CalculateKeizerPoints(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)

	// Initialize weights for 10 players
	k.initWeights(10)

	tests := []struct {
		oppRank int
		result  float64
		want    float64
	}{
		{1, 1.0, 10.0},   // Win vs rank 1 = 10 points
		{1, 0.5, 5.0},    // Draw vs rank 1 = 5 points
		{1, 0.0, 0.0},    // Loss vs rank 1 = 0 points
		{5, 1.0, 6.0},    // Win vs rank 5 = 6 points
		{10, 1.0, 1.0},   // Win vs rank 10 = 1 point
		{10, 0.5, 0.5},   // Draw vs rank 10 = 0.5 points
	}

	for _, tt := range tests {
		got := k.CalculateKeizerPoints(tt.oppRank, tt.result)
		if got != tt.want {
			t.Errorf("CalculateKeizerPoints(%d, %f) = %f, want %f", tt.oppRank, tt.result, got, tt.want)
		}
	}
}

func TestKeizer_InitWeights(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)

	k.initWeights(5)

	// Weights should be: rank 1 = 5, rank 2 = 4, ..., rank 5 = 1
	expected := []float64{0, 5, 4, 3, 2, 1}

	if len(k.keizerWeights) != len(expected) {
		t.Errorf("Weights length = %d, want %d", len(k.keizerWeights), len(expected))
		return
	}

	for i, w := range expected {
		if k.keizerWeights[i] != w {
			t.Errorf("Weight[%d] = %f, want %f", i, k.keizerWeights[i], w)
		}
	}
}

func TestKeizer_AvoidRematches(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)
	ctx := context.Background()

	// P1 and P2 have already played
	players := []Player{
		{StartNo: 1, Name: "P1", Rating: 2000, Points: 5.0, Opponents: []int{2}, IsActive: true},
		{StartNo: 2, Name: "P2", Rating: 1900, Points: 5.0, Opponents: []int{1}, IsActive: true},
		{StartNo: 3, Name: "P3", Rating: 1800, Points: 3.0, IsActive: true},
		{StartNo: 4, Name: "P4", Rating: 1700, Points: 3.0, IsActive: true},
	}

	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeSwiss,
		Players:        players,
		RoundNumber:    2,
		PairingSystem:  "keizer",
	}

	result, err := k.GenerateGeneralPairings(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// P1 should not be paired with P2 again
	for _, p := range result.Pairings {
		if p.IsBye {
			continue
		}
		if (p.WhiteStartNo == 1 && p.BlackStartNo == 2) ||
			(p.WhiteStartNo == 2 && p.BlackStartNo == 1) {
			t.Error("P1 and P2 were paired again (rematch)")
		}
	}
}

func TestKeizer_GetKeizerStandings(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)

	players := []KeizerPlayer{
		{Player: Player{StartNo: 1, Rating: 2000}, KeizerPoints: 8.0, GamesPlayed: 4},
		{Player: Player{StartNo: 2, Rating: 1900}, KeizerPoints: 12.0, GamesPlayed: 5},
		{Player: Player{StartNo: 3, Rating: 1800}, KeizerPoints: 10.0, GamesPlayed: 5},
		{Player: Player{StartNo: 4, Rating: 1700}, KeizerPoints: 10.0, GamesPlayed: 4},
	}

	standings := k.GetKeizerStandings(players)

	// Expected order: P2 (12pts), P3 (10pts, 5 games), P4 (10pts, 4 games), P1 (8pts)
	expectedOrder := []int{2, 3, 4, 1}

	for i, p := range standings {
		if p.StartNo != expectedOrder[i] {
			t.Errorf("Position %d: got player %d, expected %d", i+1, p.StartNo, expectedOrder[i])
		}
	}
}

func TestKeizer_PairingScore(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)
	k.initWeights(10)

	p1 := KeizerPlayer{
		Player:      Player{StartNo: 1},
		KeizerRank:  1,
		GamesPlayed: 5,
	}

	p2Close := KeizerPlayer{
		Player:      Player{StartNo: 2},
		KeizerRank:  2,
		GamesPlayed: 5,
	}

	p3Far := KeizerPlayer{
		Player:      Player{StartNo: 3},
		KeizerRank:  10,
		GamesPlayed: 3,
	}

	scoreClose := k.pairingScore(p1, p2Close, 1)
	scoreFar := k.pairingScore(p1, p3Far, 9)

	// Close rank should score higher than far rank
	if scoreClose <= scoreFar {
		t.Errorf("Close rank score (%f) should be > far rank score (%f)", scoreClose, scoreFar)
	}
}

func TestKeizer_Name(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)

	if k.Name() != "Keizer" {
		t.Errorf("Name() = %s, want Keizer", k.Name())
	}
}

func TestKeizer_IsAvailable(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)
	ctx := context.Background()

	if !k.IsAvailable(ctx) {
		t.Error("IsAvailable() should return true")
	}
}

func TestKeizer_SupportedTournamentTypes(t *testing.T) {
	logger := zerolog.Nop()
	k := NewKeizer(logger)

	types := k.SupportedTournamentTypes()
	if len(types) != 1 || types[0] != TournamentTypeSwiss {
		t.Errorf("SupportedTournamentTypes() = %v, want [swiss]", types)
	}
}
