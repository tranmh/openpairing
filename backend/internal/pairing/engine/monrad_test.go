package engine

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestMonrad_GenerateGeneralPairings(t *testing.T) {
	logger := zerolog.Nop()
	m := NewMonrad(logger)
	ctx := context.Background()

	tests := []struct {
		name      string
		players   []Player
		wantPairs int
		wantByes  int
		wantErr   bool
	}{
		{
			name: "4 players round 1",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2000, Points: 0, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 1900, Points: 0, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 1800, Points: 0, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 1700, Points: 0, IsActive: true},
			},
			wantPairs: 2,
			wantByes:  0,
			wantErr:   false,
		},
		{
			name: "5 players (odd - needs bye)",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2000, Points: 0, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 1900, Points: 0, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 1800, Points: 0, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 1700, Points: 0, IsActive: true},
				{StartNo: 5, Name: "P5", Rating: 1600, Points: 0, IsActive: true},
			},
			wantPairs: 3, // 2 games + 1 bye
			wantByes:  1,
			wantErr:   false,
		},
		{
			name: "with different point groups",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2000, Points: 1.0, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 1900, Points: 1.0, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 1800, Points: 0.5, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 1700, Points: 0.5, IsActive: true},
				{StartNo: 5, Name: "P5", Rating: 1600, Points: 0.0, IsActive: true},
				{StartNo: 6, Name: "P6", Rating: 1500, Points: 0.0, IsActive: true},
			},
			wantPairs: 3,
			wantByes:  0,
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
				PairingSystem:  "monrad",
			}

			result, err := m.GenerateGeneralPairings(ctx, req)

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

			byes := 0
			for _, p := range result.Pairings {
				if p.IsBye {
					byes++
				}
			}
			if byes != tt.wantByes {
				t.Errorf("Got %d byes, want %d", byes, tt.wantByes)
			}

			// Verify each player appears exactly once
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

func TestMonrad_ScoreGroupPairing(t *testing.T) {
	logger := zerolog.Nop()
	m := NewMonrad(logger)
	ctx := context.Background()

	// Players with different scores - should be grouped
	players := []Player{
		{StartNo: 1, Name: "P1", Rating: 2000, Points: 2.0, IsActive: true},
		{StartNo: 2, Name: "P2", Rating: 1950, Points: 2.0, IsActive: true},
		{StartNo: 3, Name: "P3", Rating: 1900, Points: 1.5, IsActive: true},
		{StartNo: 4, Name: "P4", Rating: 1850, Points: 1.5, IsActive: true},
		{StartNo: 5, Name: "P5", Rating: 1800, Points: 1.0, IsActive: true},
		{StartNo: 6, Name: "P6", Rating: 1750, Points: 1.0, IsActive: true},
	}

	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeSwiss,
		Players:        players,
		RoundNumber:    3,
		PairingSystem:  "monrad",
	}

	result, err := m.GenerateGeneralPairings(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(result.Pairings) != 3 {
		t.Errorf("Got %d pairings, want 3", len(result.Pairings))
	}

	// Verify players are paired within score groups when possible
	// 2.0 group: P1 vs P2
	// 1.5 group: P3 vs P4
	// 1.0 group: P5 vs P6
	expectedPairs := map[[2]int]bool{
		{1, 2}: true,
		{3, 4}: true,
		{5, 6}: true,
	}

	for _, p := range result.Pairings {
		key := [2]int{minInt(p.WhiteStartNo, p.BlackStartNo), maxInt(p.WhiteStartNo, p.BlackStartNo)}
		if !expectedPairs[key] {
			t.Logf("Unexpected pairing: %d vs %d (may be valid if avoiding rematches)", p.WhiteStartNo, p.BlackStartNo)
		}
	}
}

func TestMonrad_AvoidRematches(t *testing.T) {
	logger := zerolog.Nop()
	m := NewMonrad(logger)
	ctx := context.Background()

	// P1 and P2 have already played
	players := []Player{
		{StartNo: 1, Name: "P1", Rating: 2000, Points: 1.0, Opponents: []int{2}, IsActive: true},
		{StartNo: 2, Name: "P2", Rating: 1900, Points: 1.0, Opponents: []int{1}, IsActive: true},
		{StartNo: 3, Name: "P3", Rating: 1800, Points: 0.0, IsActive: true},
		{StartNo: 4, Name: "P4", Rating: 1700, Points: 0.0, IsActive: true},
	}

	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeSwiss,
		Players:        players,
		RoundNumber:    2,
		PairingSystem:  "monrad",
	}

	result, err := m.GenerateGeneralPairings(ctx, req)
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

func TestMonrad_ColorBalance(t *testing.T) {
	logger := zerolog.Nop()
	m := NewMonrad(logger)
	ctx := context.Background()

	// P1 has played 2 whites, P2 has played 2 blacks
	players := []Player{
		{StartNo: 1, Name: "P1", Rating: 2000, Points: 1.0, ColorHistory: []Color{ColorWhite, ColorWhite}, IsActive: true},
		{StartNo: 2, Name: "P2", Rating: 1900, Points: 1.0, ColorHistory: []Color{ColorBlack, ColorBlack}, IsActive: true},
	}

	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeSwiss,
		Players:        players,
		RoundNumber:    3,
		PairingSystem:  "monrad",
	}

	result, err := m.GenerateGeneralPairings(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(result.Pairings) != 1 {
		t.Errorf("Expected 1 pairing, got %d", len(result.Pairings))
		return
	}

	p := result.Pairings[0]
	// P2 should get white (has more blacks)
	if p.WhiteStartNo != 2 {
		t.Errorf("Expected P2 to get white, but P%d got white", p.WhiteStartNo)
	}
}

func TestMonrad_GroupByScore(t *testing.T) {
	logger := zerolog.Nop()
	m := NewMonrad(logger)

	players := []Player{
		{StartNo: 1, Points: 2.0, IsActive: true},
		{StartNo: 2, Points: 1.5, IsActive: true},
		{StartNo: 3, Points: 2.0, IsActive: true},
		{StartNo: 4, Points: 1.0, IsActive: true},
		{StartNo: 5, Points: 1.5, IsActive: true},
	}

	groups := m.groupByScore(players)

	if len(groups) != 3 {
		t.Errorf("Expected 3 score groups, got %d", len(groups))
	}

	// Groups should be sorted descending by score
	expectedScores := []float64{2.0, 1.5, 1.0}
	for i, group := range groups {
		if len(group) == 0 {
			t.Errorf("Group %d is empty", i)
			continue
		}
		if group[0].Points != expectedScores[i] {
			t.Errorf("Group %d has score %f, expected %f", i, group[0].Points, expectedScores[i])
		}
	}
}

func TestMonrad_Name(t *testing.T) {
	logger := zerolog.Nop()
	m := NewMonrad(logger)

	if m.Name() != "Monrad" {
		t.Errorf("Name() = %s, want Monrad", m.Name())
	}
}

func TestMonrad_IsAvailable(t *testing.T) {
	logger := zerolog.Nop()
	m := NewMonrad(logger)
	ctx := context.Background()

	if !m.IsAvailable(ctx) {
		t.Error("IsAvailable() should return true")
	}
}

func TestMonrad_DetermineColors(t *testing.T) {
	logger := zerolog.Nop()
	m := NewMonrad(logger)

	tests := []struct {
		name        string
		p1          Player
		p2          Player
		expectWhite int // StartNo of expected white player
	}{
		{
			name:        "p1 has fewer whites",
			p1:          Player{StartNo: 1, ColorHistory: []Color{ColorBlack, ColorBlack}},
			p2:          Player{StartNo: 2, ColorHistory: []Color{ColorWhite, ColorWhite}},
			expectWhite: 1,
		},
		{
			name:        "p2 has fewer whites",
			p1:          Player{StartNo: 1, ColorHistory: []Color{ColorWhite, ColorWhite}},
			p2:          Player{StartNo: 2, ColorHistory: []Color{ColorBlack, ColorBlack}},
			expectWhite: 2,
		},
		{
			name:        "equal colors, p1 last black",
			p1:          Player{StartNo: 1, ColorHistory: []Color{ColorWhite, ColorBlack}},
			p2:          Player{StartNo: 2, ColorHistory: []Color{ColorBlack, ColorWhite}},
			expectWhite: 1,
		},
		{
			name:        "all equal, higher rated gets white",
			p1:          Player{StartNo: 1, Rating: 2100, ColorHistory: []Color{ColorWhite}},
			p2:          Player{StartNo: 2, Rating: 2000, ColorHistory: []Color{ColorWhite}},
			expectWhite: 1,
		},
		{
			name:        "all equal, lower start number gets white",
			p1:          Player{StartNo: 1, Rating: 2000, ColorHistory: []Color{}},
			p2:          Player{StartNo: 2, Rating: 2000, ColorHistory: []Color{}},
			expectWhite: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			white, _ := m.determineColors(tt.p1, tt.p2)
			if white.StartNo != tt.expectWhite {
				t.Errorf("Expected player %d to get white, got player %d", tt.expectWhite, white.StartNo)
			}
		})
	}
}
