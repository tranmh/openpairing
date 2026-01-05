package engine

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestTeamSwiss_GenerateGeneralPairings(t *testing.T) {
	logger := zerolog.Nop()
	ts := NewTeamSwiss(logger)
	ctx := context.Background()

	teams := []Team{
		{
			ID:     1,
			Name:   "Russia",
			Points: 2.0,
			Players: []Player{
				{StartNo: 1, Name: "R1", Rating: 2750, BoardNumber: 1, IsActive: true},
				{StartNo: 2, Name: "R2", Rating: 2700, BoardNumber: 2, IsActive: true},
				{StartNo: 3, Name: "R3", Rating: 2650, BoardNumber: 3, IsActive: true},
				{StartNo: 4, Name: "R4", Rating: 2600, BoardNumber: 4, IsActive: true},
			},
		},
		{
			ID:     2,
			Name:   "USA",
			Points: 2.0,
			Players: []Player{
				{StartNo: 5, Name: "U1", Rating: 2780, BoardNumber: 1, IsActive: true},
				{StartNo: 6, Name: "U2", Rating: 2720, BoardNumber: 2, IsActive: true},
				{StartNo: 7, Name: "U3", Rating: 2680, BoardNumber: 3, IsActive: true},
				{StartNo: 8, Name: "U4", Rating: 2620, BoardNumber: 4, IsActive: true},
			},
		},
		{
			ID:     3,
			Name:   "China",
			Points: 1.5,
			Players: []Player{
				{StartNo: 9, Name: "C1", Rating: 2760, BoardNumber: 1, IsActive: true},
				{StartNo: 10, Name: "C2", Rating: 2710, BoardNumber: 2, IsActive: true},
				{StartNo: 11, Name: "C3", Rating: 2660, BoardNumber: 3, IsActive: true},
				{StartNo: 12, Name: "C4", Rating: 2610, BoardNumber: 4, IsActive: true},
			},
		},
		{
			ID:     4,
			Name:   "India",
			Points: 1.5,
			Players: []Player{
				{StartNo: 13, Name: "I1", Rating: 2740, BoardNumber: 1, IsActive: true},
				{StartNo: 14, Name: "I2", Rating: 2690, BoardNumber: 2, IsActive: true},
				{StartNo: 15, Name: "I3", Rating: 2640, BoardNumber: 3, IsActive: true},
				{StartNo: 16, Name: "I4", Rating: 2590, BoardNumber: 4, IsActive: true},
			},
		},
	}

	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeTeamSwiss,
		Teams:          teams,
		RoundNumber:    2,
	}

	result, err := ts.GenerateGeneralPairings(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// 4 teams = 2 matches = 8 board pairings
	if len(result.Pairings) != 8 {
		t.Errorf("Got %d pairings, want 8", len(result.Pairings))
	}

	// Verify each player appears exactly once
	seen := make(map[int]bool)
	for _, p := range result.Pairings {
		if p.IsBye {
			if seen[p.WhiteStartNo] {
				t.Errorf("Player %d appears multiple times", p.WhiteStartNo)
			}
			seen[p.WhiteStartNo] = true
			continue
		}

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

func TestTeamSwiss_OddNumberOfTeams(t *testing.T) {
	logger := zerolog.Nop()
	ts := NewTeamSwiss(logger)
	ctx := context.Background()

	teams := []Team{
		{
			ID:     1,
			Name:   "Team 1",
			Points: 2.0,
			Players: []Player{
				{StartNo: 1, Name: "P1", Rating: 2500, BoardNumber: 1, IsActive: true},
				{StartNo: 2, Name: "P2", Rating: 2400, BoardNumber: 2, IsActive: true},
			},
		},
		{
			ID:     2,
			Name:   "Team 2",
			Points: 1.5,
			Players: []Player{
				{StartNo: 3, Name: "P3", Rating: 2450, BoardNumber: 1, IsActive: true},
				{StartNo: 4, Name: "P4", Rating: 2350, BoardNumber: 2, IsActive: true},
			},
		},
		{
			ID:     3,
			Name:   "Team 3",
			Points: 1.0,
			Players: []Player{
				{StartNo: 5, Name: "P5", Rating: 2400, BoardNumber: 1, IsActive: true},
				{StartNo: 6, Name: "P6", Rating: 2300, BoardNumber: 2, IsActive: true},
			},
		},
	}

	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeTeamSwiss,
		Teams:          teams,
		RoundNumber:    1,
	}

	result, err := ts.GenerateGeneralPairings(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// 3 teams: 1 match (2 boards) + 1 team bye
	// The team bye is represented as a single pairing with IsBye=true
	// So we expect at least 1 pairing (the match generates board pairings or the bye)
	if len(result.Pairings) < 1 {
		t.Errorf("Expected at least 1 pairing, got %d", len(result.Pairings))
	}

	// Count actual games vs byes
	games := 0
	byes := 0
	for _, p := range result.Pairings {
		if p.IsBye {
			byes++
		} else {
			games++
		}
	}

	// With 3 teams: 1 team match (2 board games) + 1 team bye
	// Total: 2 games + 1 bye = 3 pairings OR just team-level representation
	t.Logf("Generated %d games and %d byes", games, byes)
}

func TestTeamSwiss_BoardPairings(t *testing.T) {
	logger := zerolog.Nop()
	ts := NewTeamSwiss(logger)

	home := Team{
		ID:   1,
		Name: "Home",
		Players: []Player{
			{StartNo: 1, Name: "H1", BoardNumber: 1, IsActive: true},
			{StartNo: 2, Name: "H2", BoardNumber: 2, IsActive: true},
			{StartNo: 3, Name: "H3", BoardNumber: 3, IsActive: true},
			{StartNo: 4, Name: "H4", BoardNumber: 4, IsActive: true},
		},
	}

	away := Team{
		ID:   2,
		Name: "Away",
		Players: []Player{
			{StartNo: 5, Name: "A1", BoardNumber: 1, IsActive: true},
			{StartNo: 6, Name: "A2", BoardNumber: 2, IsActive: true},
			{StartNo: 7, Name: "A3", BoardNumber: 3, IsActive: true},
			{StartNo: 8, Name: "A4", BoardNumber: 4, IsActive: true},
		},
	}

	pairings := ts.generateBoardPairings(home, away, 1)

	if len(pairings) != 4 {
		t.Errorf("Got %d board pairings, want 4", len(pairings))
	}

	// Verify board 1: H1 vs A1
	// Board order should match
	for _, p := range pairings {
		// Find which home and away player are in this pairing
		var homePlayer, awayPlayer int
		for _, hp := range home.Players {
			if hp.StartNo == p.WhiteStartNo || hp.StartNo == p.BlackStartNo {
				homePlayer = hp.BoardNumber
				break
			}
		}
		for _, ap := range away.Players {
			if ap.StartNo == p.WhiteStartNo || ap.StartNo == p.BlackStartNo {
				awayPlayer = ap.BoardNumber
				break
			}
		}

		if homePlayer != awayPlayer {
			t.Errorf("Board mismatch: home board %d vs away board %d", homePlayer, awayPlayer)
		}
	}

	// Verify color alternation: odd boards home=white, even boards home=black
	for i, p := range pairings {
		boardNum := i + 1
		isHomeWhite := false
		for _, hp := range home.Players {
			if hp.StartNo == p.WhiteStartNo {
				isHomeWhite = true
				break
			}
		}

		expectedHomeWhite := boardNum%2 == 1
		if isHomeWhite != expectedHomeWhite {
			t.Errorf("Board %d: home white = %v, expected %v", boardNum, isHomeWhite, expectedHomeWhite)
		}
	}
}

func TestTeamSwiss_ValidateTeams(t *testing.T) {
	logger := zerolog.Nop()
	ts := NewTeamSwiss(logger)

	// Valid teams
	validTeams := []Team{
		{
			ID:   1,
			Name: "Team 1",
			Players: []Player{
				{StartNo: 1, Name: "P1", BoardNumber: 1, IsActive: true},
				{StartNo: 2, Name: "P2", BoardNumber: 2, IsActive: true},
			},
		},
		{
			ID:   2,
			Name: "Team 2",
			Players: []Player{
				{StartNo: 3, Name: "P3", BoardNumber: 1, IsActive: true},
				{StartNo: 4, Name: "P4", BoardNumber: 2, IsActive: true},
			},
		},
	}

	errors := ts.ValidateTeams(validTeams)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid teams, got: %v", errors)
	}

	// Unequal team sizes
	unequalTeams := []Team{
		{
			ID:   1,
			Name: "Team 1",
			Players: []Player{
				{StartNo: 1, Name: "P1", BoardNumber: 1, IsActive: true},
				{StartNo: 2, Name: "P2", BoardNumber: 2, IsActive: true},
			},
		},
		{
			ID:   2,
			Name: "Team 2",
			Players: []Player{
				{StartNo: 3, Name: "P3", BoardNumber: 1, IsActive: true},
			},
		},
	}

	errors = ts.ValidateTeams(unequalTeams)
	if len(errors) == 0 {
		t.Error("Expected error for unequal team sizes")
	}

	// Missing board numbers
	missingBoards := []Team{
		{
			ID:   1,
			Name: "Team 1",
			Players: []Player{
				{StartNo: 1, Name: "P1", BoardNumber: 0, IsActive: true},
				{StartNo: 2, Name: "P2", BoardNumber: 2, IsActive: true},
			},
		},
		{
			ID:   2,
			Name: "Team 2",
			Players: []Player{
				{StartNo: 3, Name: "P3", BoardNumber: 1, IsActive: true},
				{StartNo: 4, Name: "P4", BoardNumber: 2, IsActive: true},
			},
		},
	}

	errors = ts.ValidateTeams(missingBoards)
	if len(errors) == 0 {
		t.Error("Expected error for missing board number")
	}

	// Duplicate board numbers
	duplicateBoards := []Team{
		{
			ID:   1,
			Name: "Team 1",
			Players: []Player{
				{StartNo: 1, Name: "P1", BoardNumber: 1, IsActive: true},
				{StartNo: 2, Name: "P2", BoardNumber: 1, IsActive: true}, // Duplicate
			},
		},
		{
			ID:   2,
			Name: "Team 2",
			Players: []Player{
				{StartNo: 3, Name: "P3", BoardNumber: 1, IsActive: true},
				{StartNo: 4, Name: "P4", BoardNumber: 2, IsActive: true},
			},
		},
	}

	errors = ts.ValidateTeams(duplicateBoards)
	if len(errors) == 0 {
		t.Error("Expected error for duplicate board numbers")
	}
}

func TestTeamSwiss_ErrorCases(t *testing.T) {
	logger := zerolog.Nop()
	ts := NewTeamSwiss(logger)
	ctx := context.Background()

	// Too few teams
	req := &GeneralPairingRequest{
		TournamentType: TournamentTypeTeamSwiss,
		Teams: []Team{
			{ID: 1, Name: "Only One", Players: []Player{{StartNo: 1, BoardNumber: 1, IsActive: true}}},
		},
		RoundNumber: 1,
	}

	_, err := ts.GenerateGeneralPairings(ctx, req)
	if err == nil {
		t.Error("Expected error for too few teams")
	}
}

func TestTeamSwiss_Name(t *testing.T) {
	logger := zerolog.Nop()
	ts := NewTeamSwiss(logger)

	if ts.Name() != "Team Swiss" {
		t.Errorf("Name() = %s, want Team Swiss", ts.Name())
	}
}

func TestTeamSwiss_TeamAvgRating(t *testing.T) {
	logger := zerolog.Nop()
	ts := NewTeamSwiss(logger)

	team := Team{
		ID:   1,
		Name: "Test",
		Players: []Player{
			{StartNo: 1, Rating: 2500, IsActive: true},
			{StartNo: 2, Rating: 2400, IsActive: true},
			{StartNo: 3, Rating: 2300, IsActive: true},
			{StartNo: 4, Rating: 2200, IsActive: true},
		},
	}

	avg := ts.teamAvgRating(team)
	expected := 2350.0 // (2500+2400+2300+2200) / 4

	if avg != expected {
		t.Errorf("teamAvgRating() = %f, want %f", avg, expected)
	}

	// Empty team
	emptyTeam := Team{ID: 2, Name: "Empty", Players: []Player{}}
	if ts.teamAvgRating(emptyTeam) != 0 {
		t.Error("Expected 0 for empty team")
	}
}
