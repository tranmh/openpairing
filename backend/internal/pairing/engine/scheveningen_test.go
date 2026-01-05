package engine

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestScheveningen_GenerateGeneralPairings(t *testing.T) {
	logger := zerolog.Nop()
	sch := NewScheveningen(logger)
	ctx := context.Background()

	team1 := Team{
		ID:   1,
		Name: "Team Alpha",
		Players: []Player{
			{StartNo: 1, Name: "A1", Rating: 2500, BoardNumber: 1, IsActive: true},
			{StartNo: 2, Name: "A2", Rating: 2400, BoardNumber: 2, IsActive: true},
			{StartNo: 3, Name: "A3", Rating: 2300, BoardNumber: 3, IsActive: true},
			{StartNo: 4, Name: "A4", Rating: 2200, BoardNumber: 4, IsActive: true},
		},
	}

	team2 := Team{
		ID:   2,
		Name: "Team Beta",
		Players: []Player{
			{StartNo: 5, Name: "B1", Rating: 2450, BoardNumber: 1, IsActive: true},
			{StartNo: 6, Name: "B2", Rating: 2350, BoardNumber: 2, IsActive: true},
			{StartNo: 7, Name: "B3", Rating: 2250, BoardNumber: 3, IsActive: true},
			{StartNo: 8, Name: "B4", Rating: 2150, BoardNumber: 4, IsActive: true},
		},
	}

	tests := []struct {
		name       string
		round      int
		wantPairs  int
		wantErr    bool
	}{
		{
			name:      "round 1",
			round:     1,
			wantPairs: 4,
			wantErr:   false,
		},
		{
			name:      "round 2",
			round:     2,
			wantPairs: 4,
			wantErr:   false,
		},
		{
			name:      "round 4 (last round)",
			round:     4,
			wantPairs: 4,
			wantErr:   false,
		},
		{
			name:    "invalid round 5",
			round:   5,
			wantErr: true,
		},
		{
			name:    "invalid round 0",
			round:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GeneralPairingRequest{
				TournamentType: TournamentTypeScheveningen,
				Teams:          []Team{team1, team2},
				RoundNumber:    tt.round,
			}

			result, err := sch.GenerateGeneralPairings(ctx, req)

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

			// Verify each player from smaller team appears exactly once
			seen := make(map[int]bool)
			for _, p := range result.Pairings {
				if seen[p.WhiteStartNo] {
					t.Errorf("Player %d appears multiple times as white", p.WhiteStartNo)
				}
				if seen[p.BlackStartNo] {
					t.Errorf("Player %d appears multiple times as black", p.BlackStartNo)
				}
				seen[p.WhiteStartNo] = true
				seen[p.BlackStartNo] = true
			}
		})
	}
}

func TestScheveningen_CompleteSchedule(t *testing.T) {
	logger := zerolog.Nop()
	sch := NewScheveningen(logger)
	ctx := context.Background()

	team1 := Team{
		ID:   1,
		Name: "Team Alpha",
		Players: []Player{
			{StartNo: 1, Name: "A1", BoardNumber: 1, IsActive: true},
			{StartNo: 2, Name: "A2", BoardNumber: 2, IsActive: true},
			{StartNo: 3, Name: "A3", BoardNumber: 3, IsActive: true},
		},
	}

	team2 := Team{
		ID:   2,
		Name: "Team Beta",
		Players: []Player{
			{StartNo: 4, Name: "B1", BoardNumber: 1, IsActive: true},
			{StartNo: 5, Name: "B2", BoardNumber: 2, IsActive: true},
			{StartNo: 6, Name: "B3", BoardNumber: 3, IsActive: true},
		},
	}

	// Track all matchups across all rounds
	matchups := make(map[[2]int]int)

	totalRounds := GetScheveningenTotalRounds(len(team1.Players), len(team2.Players))

	for round := 1; round <= totalRounds; round++ {
		req := &GeneralPairingRequest{
			TournamentType: TournamentTypeScheveningen,
			Teams:          []Team{team1, team2},
			RoundNumber:    round,
		}

		result, err := sch.GenerateGeneralPairings(ctx, req)
		if err != nil {
			t.Errorf("Round %d error: %v", round, err)
			continue
		}

		for _, p := range result.Pairings {
			key := [2]int{minInt(p.WhiteStartNo, p.BlackStartNo), maxInt(p.WhiteStartNo, p.BlackStartNo)}
			matchups[key]++
		}
	}

	// Verify each player from team1 plays each player from team2 exactly once
	for _, p1 := range team1.Players {
		for _, p2 := range team2.Players {
			key := [2]int{minInt(p1.StartNo, p2.StartNo), maxInt(p1.StartNo, p2.StartNo)}
			if matchups[key] != 1 {
				t.Errorf("Players %d and %d met %d times, want 1", p1.StartNo, p2.StartNo, matchups[key])
			}
		}
	}
}

func TestScheveningen_UnequalTeams(t *testing.T) {
	logger := zerolog.Nop()
	sch := NewScheveningen(logger)
	ctx := context.Background()

	// Team 1 has 3 players, Team 2 has 4 players
	team1 := Team{
		ID:   1,
		Name: "Team Small",
		Players: []Player{
			{StartNo: 1, Name: "S1", BoardNumber: 1, IsActive: true},
			{StartNo: 2, Name: "S2", BoardNumber: 2, IsActive: true},
			{StartNo: 3, Name: "S3", BoardNumber: 3, IsActive: true},
		},
	}

	team2 := Team{
		ID:   2,
		Name: "Team Large",
		Players: []Player{
			{StartNo: 4, Name: "L1", BoardNumber: 1, IsActive: true},
			{StartNo: 5, Name: "L2", BoardNumber: 2, IsActive: true},
			{StartNo: 6, Name: "L3", BoardNumber: 3, IsActive: true},
			{StartNo: 7, Name: "L4", BoardNumber: 4, IsActive: true},
		},
	}

	totalRounds := GetScheveningenTotalRounds(len(team1.Players), len(team2.Players))
	if totalRounds != 4 {
		t.Errorf("Expected 4 rounds for 3v4, got %d", totalRounds)
	}

	// Each round should have 3 pairings (smaller team size)
	for round := 1; round <= totalRounds; round++ {
		req := &GeneralPairingRequest{
			TournamentType: TournamentTypeScheveningen,
			Teams:          []Team{team1, team2},
			RoundNumber:    round,
		}

		result, err := sch.GenerateGeneralPairings(ctx, req)
		if err != nil {
			t.Errorf("Round %d error: %v", round, err)
			continue
		}

		if len(result.Pairings) != 3 {
			t.Errorf("Round %d: got %d pairings, want 3", round, len(result.Pairings))
		}
	}
}

func TestScheveningen_ErrorCases(t *testing.T) {
	logger := zerolog.Nop()
	sch := NewScheveningen(logger)
	ctx := context.Background()

	// Wrong number of teams
	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeScheveningen,
		Teams: []Team{
			{ID: 1, Name: "Only One", Players: []Player{{StartNo: 1, IsActive: true}}},
		},
		RoundNumber: 1,
	}

	_, err := sch.GenerateGeneralPairings(ctx, req)
	if err == nil {
		t.Error("Expected error for wrong number of teams")
	}

	// Empty team
	req = &GeneralPairingRequest{
		TournamentType: TournamentTypeScheveningen,
		Teams: []Team{
			{ID: 1, Name: "Empty", Players: []Player{}},
			{ID: 2, Name: "NotEmpty", Players: []Player{{StartNo: 1, IsActive: true}}},
		},
		RoundNumber: 1,
	}

	_, err = sch.GenerateGeneralPairings(ctx, req)
	if err == nil {
		t.Error("Expected error for empty team")
	}

	// Too many teams
	req = &GeneralPairingRequest{
		TournamentType: TournamentTypeScheveningen,
		Teams: []Team{
			{ID: 1, Name: "T1", Players: []Player{{StartNo: 1, IsActive: true}}},
			{ID: 2, Name: "T2", Players: []Player{{StartNo: 2, IsActive: true}}},
			{ID: 3, Name: "T3", Players: []Player{{StartNo: 3, IsActive: true}}},
		},
		RoundNumber: 1,
	}

	_, err = sch.GenerateGeneralPairings(ctx, req)
	if err == nil {
		t.Error("Expected error for too many teams")
	}
}

func TestScheveningen_ValidateTeams(t *testing.T) {
	logger := zerolog.Nop()
	sch := NewScheveningen(logger)

	// Valid teams
	team1 := Team{
		ID:   1,
		Name: "Team 1",
		Players: []Player{
			{StartNo: 1, Name: "P1", IsActive: true},
			{StartNo: 2, Name: "P2", IsActive: true},
		},
	}
	team2 := Team{
		ID:   2,
		Name: "Team 2",
		Players: []Player{
			{StartNo: 3, Name: "P3", IsActive: true},
			{StartNo: 4, Name: "P4", IsActive: true},
		},
	}

	errors := sch.ValidateTeams(team1, team2)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid teams, got: %v", errors)
	}

	// Duplicate start number across teams
	team2Dup := Team{
		ID:   2,
		Name: "Team 2",
		Players: []Player{
			{StartNo: 1, Name: "P3", IsActive: true}, // Same as team1
			{StartNo: 4, Name: "P4", IsActive: true},
		},
	}

	errors = sch.ValidateTeams(team1, team2Dup)
	if len(errors) == 0 {
		t.Error("Expected error for duplicate start number")
	}
}

func TestGetScheveningenTotalRounds(t *testing.T) {
	tests := []struct {
		team1Size int
		team2Size int
		want      int
	}{
		{4, 4, 4},
		{3, 4, 4},
		{4, 3, 4},
		{2, 5, 5},
		{6, 6, 6},
	}

	for _, tt := range tests {
		got := GetScheveningenTotalRounds(tt.team1Size, tt.team2Size)
		if got != tt.want {
			t.Errorf("GetScheveningenTotalRounds(%d, %d) = %d, want %d",
				tt.team1Size, tt.team2Size, got, tt.want)
		}
	}
}

func TestScheveningen_Name(t *testing.T) {
	logger := zerolog.Nop()
	sch := NewScheveningen(logger)

	if sch.Name() != "Scheveningen" {
		t.Errorf("Name() = %s, want Scheveningen", sch.Name())
	}
}
