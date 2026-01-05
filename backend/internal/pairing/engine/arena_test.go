package engine

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestArena_GenerateGeneralPairings(t *testing.T) {
	logger := zerolog.Nop()
	arena := NewArena(logger, nil)
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
			name: "6 players with scores",
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
			name: "odd players",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2000, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 1900, IsActive: true},
				{StartNo: 3, Name: "P3", Rating: 1800, IsActive: true},
			},
			wantPairs: 1, // One pair, one waiting
			wantErr:   false,
		},
		{
			name: "1 player (not enough)",
			players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2000, IsActive: true},
			},
			wantPairs: 0,
			wantErr:   false, // Returns empty result, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GeneralPairingRequest{
				TournamentType: TournamentTypeArena,
				Players:        tt.players,
				RoundNumber:    1,
			}

			result, err := arena.GenerateGeneralPairings(ctx, req)

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
				if seen[p.WhiteStartNo] {
					t.Errorf("Player %d appears multiple times", p.WhiteStartNo)
				}
				if seen[p.BlackStartNo] {
					t.Errorf("Player %d appears multiple times", p.BlackStartNo)
				}
				seen[p.WhiteStartNo] = true
				seen[p.BlackStartNo] = true
			}
		})
	}
}

func TestArena_MatchQuality(t *testing.T) {
	logger := zerolog.Nop()
	config := &ArenaConfig{
		MaxRatingDiff:    300,
		PreventRematches: 3,
	}
	arena := NewArena(logger, config)

	now := time.Now()

	p1 := ArenaPlayer{
		Player:          Player{StartNo: 1, Rating: 2000},
		ArenaScore:      10.0,
		SearchStartedAt: now.Add(-5 * time.Second),
	}

	p2Close := ArenaPlayer{
		Player:          Player{StartNo: 2, Rating: 2050},
		ArenaScore:      9.0,
		SearchStartedAt: now.Add(-3 * time.Second),
	}

	p3Far := ArenaPlayer{
		Player:          Player{StartNo: 3, Rating: 2500},
		ArenaScore:      15.0,
		SearchStartedAt: now.Add(-1 * time.Second),
	}

	scoreClose := arena.calculateMatchScore(p1, p2Close)
	scoreFar := arena.calculateMatchScore(p1, p3Far)

	// Close rating should score higher
	if scoreClose <= scoreFar {
		t.Errorf("Close rating score (%f) should be > far rating score (%f)", scoreClose, scoreFar)
	}
}

func TestArena_RecentRematchPrevention(t *testing.T) {
	logger := zerolog.Nop()
	config := &ArenaConfig{
		PreventRematches: 3,
	}
	arena := NewArena(logger, config)

	p1 := ArenaPlayer{
		Player:          Player{StartNo: 1},
		RecentOpponents: []int{2, 3, 4},
	}

	p2Recent := ArenaPlayer{
		Player: Player{StartNo: 2},
	}

	p5NotRecent := ArenaPlayer{
		Player: Player{StartNo: 5},
	}

	if !arena.isRecentRematch(p1, p2Recent) {
		t.Error("Expected P1 vs P2 to be detected as recent rematch")
	}

	if arena.isRecentRematch(p1, p5NotRecent) {
		t.Error("P1 vs P5 should not be a recent rematch")
	}
}

func TestArena_CalculateArenaScore(t *testing.T) {
	logger := zerolog.Nop()
	arena := NewArena(logger, &ArenaConfig{StreakBonusEnabled: true})

	tests := []struct {
		name      string
		result    float64
		winStreak int
		berserk   bool
		want      float64
	}{
		{"win no streak", 1.0, 1, false, 2.0},
		{"win 2 streak", 1.0, 2, false, 2.5},
		{"win 3 streak", 1.0, 3, false, 3.0},
		{"win max streak", 1.0, 10, false, 4.0}, // Max bonus is 2.0
		{"draw", 0.5, 0, false, 1.0},
		{"loss", 0.0, 0, false, 0.0},
		{"win berserk", 1.0, 1, true, 3.0},
		{"draw berserk", 0.5, 0, true, 2.0},
		{"loss berserk", 0.0, 0, true, 0.0}, // No berserk bonus for loss
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := arena.CalculateArenaScore(tt.result, tt.winStreak, tt.berserk)
			if got != tt.want {
				t.Errorf("CalculateArenaScore(%f, %d, %v) = %f, want %f",
					tt.result, tt.winStreak, tt.berserk, got, tt.want)
			}
		})
	}
}

func TestArena_PlayerPool(t *testing.T) {
	logger := zerolog.Nop()
	arena := NewArena(logger, nil)

	p1 := Player{StartNo: 1, Name: "P1", Rating: 2000}
	p2 := Player{StartNo: 2, Name: "P2", Rating: 1900}

	// Add players
	arena.AddPlayer(p1)
	arena.AddPlayer(p2)

	if arena.GetWaitingCount() != 2 {
		t.Errorf("Expected 2 waiting players, got %d", arena.GetWaitingCount())
	}

	// Remove player
	arena.RemovePlayer(1)

	if arena.GetWaitingCount() != 1 {
		t.Errorf("Expected 1 waiting player after removal, got %d", arena.GetWaitingCount())
	}

	// Remove non-existent
	arena.RemovePlayer(999)

	if arena.GetWaitingCount() != 1 {
		t.Errorf("Count should still be 1 after removing non-existent, got %d", arena.GetWaitingCount())
	}
}

func TestArena_GameCompleted(t *testing.T) {
	logger := zerolog.Nop()
	arena := NewArena(logger, nil)

	// Simulate active pairing
	arena.activePairings[1] = 2
	arena.activePairings[2] = 1

	arena.GameCompleted(1, 2, 1) // Player 1 wins

	// Active pairings should be cleared
	if _, exists := arena.activePairings[1]; exists {
		t.Error("Player 1 should be removed from active pairings")
	}
	if _, exists := arena.activePairings[2]; exists {
		t.Error("Player 2 should be removed from active pairings")
	}
}

func TestArena_Name(t *testing.T) {
	logger := zerolog.Nop()
	arena := NewArena(logger, nil)

	if arena.Name() != "Arena" {
		t.Errorf("Name() = %s, want Arena", arena.Name())
	}
}

func TestArena_IsAvailable(t *testing.T) {
	logger := zerolog.Nop()
	arena := NewArena(logger, nil)
	ctx := context.Background()

	if !arena.IsAvailable(ctx) {
		t.Error("IsAvailable() should return true")
	}
}

func TestArena_SupportedTournamentTypes(t *testing.T) {
	logger := zerolog.Nop()
	arena := NewArena(logger, nil)

	types := arena.SupportedTournamentTypes()
	if len(types) != 1 || types[0] != TournamentTypeArena {
		t.Errorf("SupportedTournamentTypes() = %v, want [arena]", types)
	}
}

func TestArena_DefaultConfig(t *testing.T) {
	logger := zerolog.Nop()
	arena := NewArena(logger, nil)

	// Check defaults were applied
	if arena.config.MaxRatingDiff != 300 {
		t.Errorf("Default MaxRatingDiff = %d, want 300", arena.config.MaxRatingDiff)
	}

	if arena.config.MaxWaitTime != 30*time.Second {
		t.Errorf("Default MaxWaitTime = %v, want 30s", arena.config.MaxWaitTime)
	}

	if arena.config.PreventRematches != 3 {
		t.Errorf("Default PreventRematches = %d, want 3", arena.config.PreventRematches)
	}
}

func TestArena_CustomConfig(t *testing.T) {
	logger := zerolog.Nop()
	config := &ArenaConfig{
		MaxRatingDiff:    500,
		MaxWaitTime:      60 * time.Second,
		PreventRematches: 5,
	}
	arena := NewArena(logger, config)

	if arena.config.MaxRatingDiff != 500 {
		t.Errorf("Custom MaxRatingDiff = %d, want 500", arena.config.MaxRatingDiff)
	}

	if arena.config.MaxWaitTime != 60*time.Second {
		t.Errorf("Custom MaxWaitTime = %v, want 60s", arena.config.MaxWaitTime)
	}

	if arena.config.PreventRematches != 5 {
		t.Errorf("Custom PreventRematches = %d, want 5", arena.config.PreventRematches)
	}
}
