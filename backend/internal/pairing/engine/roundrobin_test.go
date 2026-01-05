package engine

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestGenerateBergerTable(t *testing.T) {
	tests := []struct {
		name       string
		numPlayers int
		wantRounds int
		wantPairs  int // pairs per round
	}{
		{
			name:       "4 players",
			numPlayers: 4,
			wantRounds: 3,
			wantPairs:  2,
		},
		{
			name:       "5 players (odd)",
			numPlayers: 5,
			wantRounds: 5, // With bye
			wantPairs:  3,
		},
		{
			name:       "6 players",
			numPlayers: 6,
			wantRounds: 5,
			wantPairs:  3,
		},
		{
			name:       "8 players",
			numPlayers: 8,
			wantRounds: 7,
			wantPairs:  4,
		},
		{
			name:       "10 players",
			numPlayers: 10,
			wantRounds: 9,
			wantPairs:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := GenerateBergerTable(tt.numPlayers)

			if len(table) != tt.wantRounds {
				t.Errorf("GenerateBergerTable(%d) = %d rounds, want %d", tt.numPlayers, len(table), tt.wantRounds)
			}

			for round, pairs := range table {
				if len(pairs) != tt.wantPairs {
					t.Errorf("Round %d has %d pairs, want %d", round+1, len(pairs), tt.wantPairs)
				}

				// Check each player appears exactly once per round
				seen := make(map[int]bool)
				for _, pair := range pairs {
					if seen[pair[0]] {
						t.Errorf("Round %d: player %d appears twice", round+1, pair[0])
					}
					if seen[pair[1]] {
						t.Errorf("Round %d: player %d appears twice", round+1, pair[1])
					}
					seen[pair[0]] = true
					seen[pair[1]] = true
				}
			}

			// Verify each pair of players meets exactly once
			matchups := make(map[[2]int]int)
			numPlayers := tt.numPlayers
			if numPlayers%2 == 1 {
				numPlayers++ // Include bye position
			}

			for _, pairs := range table {
				for _, pair := range pairs {
					key := [2]int{minInt(pair[0], pair[1]), maxInt(pair[0], pair[1])}
					matchups[key]++
				}
			}

			for i := 1; i <= numPlayers; i++ {
				for j := i + 1; j <= numPlayers; j++ {
					key := [2]int{i, j}
					if matchups[key] != 1 {
						t.Errorf("Players %d and %d meet %d times, want 1", i, j, matchups[key])
					}
				}
			}
		})
	}
}

func TestRoundRobin_GenerateGeneralPairings(t *testing.T) {
	logger := zerolog.Nop()
	rr := NewRoundRobin(logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		players     []Player
		round       int
		doubleRR    bool
		wantPairs   int
		wantErr     bool
	}{
		{
			name: "4 players round 1",
			players: []Player{
				{StartNo: 1, Name: "Player 1", Rating: 2000, IsActive: true},
				{StartNo: 2, Name: "Player 2", Rating: 1900, IsActive: true},
				{StartNo: 3, Name: "Player 3", Rating: 1800, IsActive: true},
				{StartNo: 4, Name: "Player 4", Rating: 1700, IsActive: true},
			},
			round:     1,
			wantPairs: 2,
			wantErr:   false,
		},
		{
			name: "4 players round 3",
			players: []Player{
				{StartNo: 1, Name: "Player 1", Rating: 2000, IsActive: true},
				{StartNo: 2, Name: "Player 2", Rating: 1900, IsActive: true},
				{StartNo: 3, Name: "Player 3", Rating: 1800, IsActive: true},
				{StartNo: 4, Name: "Player 4", Rating: 1700, IsActive: true},
			},
			round:     3,
			wantPairs: 2,
			wantErr:   false,
		},
		{
			name: "5 players round 1 (odd - has bye)",
			players: []Player{
				{StartNo: 1, Name: "Player 1", Rating: 2000, IsActive: true},
				{StartNo: 2, Name: "Player 2", Rating: 1900, IsActive: true},
				{StartNo: 3, Name: "Player 3", Rating: 1800, IsActive: true},
				{StartNo: 4, Name: "Player 4", Rating: 1700, IsActive: true},
				{StartNo: 5, Name: "Player 5", Rating: 1600, IsActive: true},
			},
			round:     1,
			wantPairs: 3, // 2 games + 1 bye
			wantErr:   false,
		},
		{
			name: "double round robin round 4",
			players: []Player{
				{StartNo: 1, Name: "Player 1", Rating: 2000, IsActive: true},
				{StartNo: 2, Name: "Player 2", Rating: 1900, IsActive: true},
				{StartNo: 3, Name: "Player 3", Rating: 1800, IsActive: true},
				{StartNo: 4, Name: "Player 4", Rating: 1700, IsActive: true},
			},
			round:     4, // First round of second half
			doubleRR:  true,
			wantPairs: 2,
			wantErr:   false,
		},
		{
			name: "invalid round",
			players: []Player{
				{StartNo: 1, Name: "Player 1", Rating: 2000, IsActive: true},
				{StartNo: 2, Name: "Player 2", Rating: 1900, IsActive: true},
			},
			round:   10,
			wantErr: true,
		},
		{
			name: "too few players",
			players: []Player{
				{StartNo: 1, Name: "Player 1", Rating: 2000, IsActive: true},
			},
			round:   1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GeneralPairingRequest{
				TournamentType:   TournamentTypeRoundRobin,
				Players:          tt.players,
				RoundNumber:      tt.round,
				DoubleRoundRobin: tt.doubleRR,
			}

			result, err := rr.GenerateGeneralPairings(ctx, req)

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

func TestRoundRobin_ValidatePairings(t *testing.T) {
	logger := zerolog.Nop()
	rr := NewRoundRobin(logger)

	players := []Player{
		{StartNo: 1, Name: "Player 1", IsActive: true},
		{StartNo: 2, Name: "Player 2", IsActive: true},
		{StartNo: 3, Name: "Player 3", IsActive: true},
		{StartNo: 4, Name: "Player 4", IsActive: true},
	}

	// Valid pairings
	validPairings := []Pairing{
		{Board: 1, WhiteStartNo: 1, BlackStartNo: 4, IsBye: false},
		{Board: 2, WhiteStartNo: 2, BlackStartNo: 3, IsBye: false},
	}

	errors := rr.ValidatePairings(players, validPairings, 1)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid pairings, got: %v", errors)
	}

	// Invalid: player appears twice
	invalidPairings := []Pairing{
		{Board: 1, WhiteStartNo: 1, BlackStartNo: 4, IsBye: false},
		{Board: 2, WhiteStartNo: 1, BlackStartNo: 3, IsBye: false},
	}

	errors = rr.ValidatePairings(players, invalidPairings, 1)
	if len(errors) == 0 {
		t.Error("Expected errors for invalid pairings")
	}
}

func TestGetRoundRobinTotalRounds(t *testing.T) {
	tests := []struct {
		numPlayers int
		doubleRR   bool
		want       int
	}{
		{4, false, 3},
		{4, true, 6},
		{5, false, 5},
		{5, true, 10},
		{6, false, 5},
		{6, true, 10},
		{10, false, 9},
		{10, true, 18},
	}

	for _, tt := range tests {
		got := GetRoundRobinTotalRounds(tt.numPlayers, tt.doubleRR)
		if got != tt.want {
			t.Errorf("GetRoundRobinTotalRounds(%d, %v) = %d, want %d",
				tt.numPlayers, tt.doubleRR, got, tt.want)
		}
	}
}

func TestRoundRobin_Name(t *testing.T) {
	logger := zerolog.Nop()
	rr := NewRoundRobin(logger)

	if rr.Name() != "Round Robin" {
		t.Errorf("Name() = %s, want Round Robin", rr.Name())
	}
}

func TestRoundRobin_IsAvailable(t *testing.T) {
	logger := zerolog.Nop()
	rr := NewRoundRobin(logger)
	ctx := context.Background()

	if !rr.IsAvailable(ctx) {
		t.Error("IsAvailable() should return true")
	}
}

func TestRoundRobin_SupportedTournamentTypes(t *testing.T) {
	logger := zerolog.Nop()
	rr := NewRoundRobin(logger)

	types := rr.SupportedTournamentTypes()
	if len(types) != 1 || types[0] != TournamentTypeRoundRobin {
		t.Errorf("SupportedTournamentTypes() = %v, want [round_robin]", types)
	}
}
