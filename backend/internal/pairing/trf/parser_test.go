package trf

import (
	"strings"
	"testing"
)

func TestParseTRF_ValidFile(t *testing.T) {
	trfContent := `012 Test Tournament 2024
022 Berlin
032 GER
042 2024/01/15
052 2024/01/20
062 7
072 Swiss
082 IA John Smith
102 90+30
001    1 m  GM Carlsen, Magnus               2830 NOR 1503014    1990/11/30  7.0    1  w 1   2  b 1  15  w 1   3
001    2 m  GM Caruana, Fabiano              2820 USA 2020009    1992/07/30  6.5    2  b 0   1  w 1   8  b 1   5
001    3 m  GM Nakamura, Hikaru              2780 USA 2016192    1987/12/09  5.5    3  w =  15  b 1   1  w 0   7
`

	parser := NewParser()
	trf, err := parser.Parse(strings.NewReader(trfContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check header
	if trf.TournamentName != "Test Tournament 2024" {
		t.Errorf("TournamentName = %q, want %q", trf.TournamentName, "Test Tournament 2024")
	}
	if trf.City != "Berlin" {
		t.Errorf("City = %q, want %q", trf.City, "Berlin")
	}
	if trf.Federation != "GER" {
		t.Errorf("Federation = %q, want %q", trf.Federation, "GER")
	}
	if trf.Rounds != 7 {
		t.Errorf("Rounds = %d, want %d", trf.Rounds, 7)
	}
	if trf.TournamentType != "Swiss" {
		t.Errorf("TournamentType = %q, want %q", trf.TournamentType, "Swiss")
	}
	if trf.ChiefArbiter != "IA John Smith" {
		t.Errorf("ChiefArbiter = %q, want %q", trf.ChiefArbiter, "IA John Smith")
	}

	// Check players
	if len(trf.Players) != 3 {
		t.Fatalf("Players count = %d, want %d", len(trf.Players), 3)
	}

	// Check first player
	p1 := trf.Players[0]
	if p1.StartNo != 1 {
		t.Errorf("Player 1 StartNo = %d, want %d", p1.StartNo, 1)
	}
	if p1.Sex != "m" {
		t.Errorf("Player 1 Sex = %q, want %q", p1.Sex, "m")
	}
	if p1.Title != "GM" {
		t.Errorf("Player 1 Title = %q, want %q", p1.Title, "GM")
	}
	if !strings.Contains(p1.Name, "Carlsen") {
		t.Errorf("Player 1 Name = %q, should contain 'Carlsen'", p1.Name)
	}
	if p1.Rating != 2830 {
		t.Errorf("Player 1 Rating = %d, want %d", p1.Rating, 2830)
	}
	if p1.Federation != "NOR" {
		t.Errorf("Player 1 Federation = %q, want %q", p1.Federation, "NOR")
	}
}

func TestParseTRF_PlayerLine(t *testing.T) {
	// Test parsing of player line with round results
	line := "001    1 m  GM Carlsen, Magnus               2830 NOR 1503014    1990/11/30  7.0    1  w 1   2  b 0   3"

	parser := NewParser()
	player, err := parser.parsePlayerLine(line)
	if err != nil {
		t.Fatalf("parsePlayerLine failed: %v", err)
	}

	if player.StartNo != 1 {
		t.Errorf("StartNo = %d, want 1", player.StartNo)
	}
	if player.Points != 7.0 {
		t.Errorf("Points = %v, want 7.0", player.Points)
	}

	// Check round results
	if len(player.RoundResults) < 2 {
		t.Fatalf("RoundResults count = %d, want >= 2", len(player.RoundResults))
	}

	// Round 1: white, win against 2
	r1 := player.RoundResults[0]
	if r1.Color != "w" {
		t.Errorf("Round 1 Color = %q, want %q", r1.Color, "w")
	}
	if r1.Result != "1" {
		t.Errorf("Round 1 Result = %q, want %q", r1.Result, "1")
	}
	if r1.OpponentNo != 2 {
		t.Errorf("Round 1 OpponentNo = %d, want 2", r1.OpponentNo)
	}
}

func TestParseTRF_RoundResult(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expColor string
		expResult string
		expOpp   int
	}{
		{"win white", "  w 1   2", "w", "1", 2},
		{"loss black", "  b 0   5", "b", "0", 5},
		{"draw white", "  w =  10", "w", "=", 10},
		{"bye", "  - H    ", "-", "H", 0},
		{"forfeit win", "  w +   3", "w", "1", 3}, // + maps to 1
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseRoundResult(1, tt.data)

			if result.Color != tt.expColor {
				t.Errorf("Color = %q, want %q", result.Color, tt.expColor)
			}
			if result.Result != tt.expResult {
				t.Errorf("Result = %q, want %q", result.Result, tt.expResult)
			}
			if result.OpponentNo != tt.expOpp {
				t.Errorf("OpponentNo = %d, want %d", result.OpponentNo, tt.expOpp)
			}
		})
	}
}

func TestParseTRF_TeamLine(t *testing.T) {
	line := "013    1 Berlin Chess Club 1 2 3 4"

	parser := NewParser()
	team, err := parser.parseTeamLine(line)
	if err != nil {
		t.Fatalf("parseTeamLine failed: %v", err)
	}

	if team.TeamNo != 1 {
		t.Errorf("TeamNo = %d, want 1", team.TeamNo)
	}
	if !strings.Contains(team.TeamName, "Berlin") {
		t.Errorf("TeamName = %q, should contain 'Berlin'", team.TeamName)
	}
	if len(team.PlayerNos) != 4 {
		t.Errorf("PlayerNos count = %d, want 4", len(team.PlayerNos))
	}
}

func TestParseTRF_XXR(t *testing.T) {
	line := "XXR    1 Additional info"

	parser := NewParser()
	playerNo, data := parser.parseXXRLine(line)

	if playerNo != 1 {
		t.Errorf("PlayerNo = %d, want 1", playerNo)
	}
	if data != "Additional info" {
		t.Errorf("Data = %q, want %q", data, "Additional info")
	}
}

func TestGetPairings(t *testing.T) {
	trfContent := `012 Test Tournament
062 5
001    1    Player One                       2000     1990/01/01  1.0    1  w 1   2
001    2    Player Two                       1900     1990/01/01  0.0    2  b 0   1
001    3    Player Three                     1800     1990/01/01  0.5    1  - H
`

	parser := NewParser()
	trf, err := parser.Parse(strings.NewReader(trfContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	pairings := trf.GetPairings(1)

	// Should have 2 pairings: 1 vs 2 and bye for 3
	if len(pairings) != 2 {
		t.Fatalf("Pairings count = %d, want 2", len(pairings))
	}

	// First pairing: 1 (white) vs 2 (black)
	p1 := pairings[0]
	if p1.White != 1 || p1.Black != 2 {
		t.Errorf("First pairing: White=%d, Black=%d, want White=1, Black=2", p1.White, p1.Black)
	}
	if p1.IsBye {
		t.Errorf("First pairing should not be a bye")
	}

	// Second pairing: bye for 3
	p2 := pairings[1]
	if p2.White != 3 {
		t.Errorf("Second pairing White = %d, want 3", p2.White)
	}
	if !p2.IsBye {
		t.Errorf("Second pairing should be a bye")
	}
}

func TestParseTRF_EmptyFile(t *testing.T) {
	parser := NewParser()
	trf, err := parser.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse failed on empty file: %v", err)
	}

	if len(trf.Players) != 0 {
		t.Errorf("Players count = %d, want 0", len(trf.Players))
	}
}

func TestParseTRF_ShortLines(t *testing.T) {
	trfContent := `012 T
01
0
`

	parser := NewParser()
	trf, err := parser.Parse(strings.NewReader(trfContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if trf.TournamentName != "T" {
		t.Errorf("TournamentName = %q, want %q", trf.TournamentName, "T")
	}
}

func TestParseTRF_DeputyArbiters(t *testing.T) {
	trfContent := `012 Test Tournament
082 IA Chief Arbiter
092 FA Deputy One
092 NA Deputy Two
`

	parser := NewParser()
	trf, err := parser.Parse(strings.NewReader(trfContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if trf.ChiefArbiter != "IA Chief Arbiter" {
		t.Errorf("ChiefArbiter = %q", trf.ChiefArbiter)
	}
	if len(trf.DeputyArbiters) != 2 {
		t.Errorf("DeputyArbiters count = %d, want 2", len(trf.DeputyArbiters))
	}
}

func TestParseTRF_RoundDates(t *testing.T) {
	trfContent := `012 Test Tournament
062 3
112 2024/01/15 2024/01/16 2024/01/17
`

	parser := NewParser()
	trf, err := parser.Parse(strings.NewReader(trfContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(trf.RoundDates) != 3 {
		t.Errorf("RoundDates count = %d, want 3", len(trf.RoundDates))
	}
}

func TestParseTRF_XXCRecords(t *testing.T) {
	trfContent := `012 Test Tournament
XXC white1
XXC initial
`

	parser := NewParser()
	trf, err := parser.Parse(strings.NewReader(trfContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(trf.XXCRecords) != 2 {
		t.Errorf("XXCRecords count = %d, want 2", len(trf.XXCRecords))
	}
}
