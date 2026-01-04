package trf

import (
	"strings"
	"testing"
)

func TestParseTRF_ValidFile(t *testing.T) {
	// Use generator to create properly formatted TRF content
	gen := NewGenerator()
	gen.SetTournamentInfo("Test Tournament 2024", "Berlin", "GER", 7)
	gen.trf.StartDate = "2024/01/15"
	gen.trf.EndDate = "2024/01/20"
	gen.SetTournamentType("Swiss")
	gen.SetArbiters("IA John Smith", nil)
	gen.SetTimeControl("90+30")

	gen.AddPlayer(TRFPlayer{
		StartNo:    1,
		Sex:        "m",
		Title:      "GM",
		Name:       "Carlsen, Magnus",
		Rating:     2830,
		Federation: "NOR",
		FideID:     "1503014",
		BirthDate:  "1990/11/30",
		Points:     7.0,
		Rank:       1,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "w", Result: "1", OpponentNo: 2},
			{Round: 2, Color: "b", Result: "1", OpponentNo: 15},
			{Round: 3, Color: "w", Result: "1", OpponentNo: 3},
		},
	})
	gen.AddPlayer(TRFPlayer{
		StartNo:    2,
		Sex:        "m",
		Title:      "GM",
		Name:       "Caruana, Fabiano",
		Rating:     2820,
		Federation: "USA",
		FideID:     "2020009",
		BirthDate:  "1992/07/30",
		Points:     6.5,
		Rank:       2,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "b", Result: "0", OpponentNo: 1},
			{Round: 2, Color: "w", Result: "1", OpponentNo: 8},
			{Round: 3, Color: "b", Result: "1", OpponentNo: 5},
		},
	})
	gen.AddPlayer(TRFPlayer{
		StartNo:    3,
		Sex:        "m",
		Title:      "GM",
		Name:       "Nakamura, Hikaru",
		Rating:     2780,
		Federation: "USA",
		FideID:     "2016192",
		BirthDate:  "1987/12/09",
		Points:     5.5,
		Rank:       3,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "w", Result: "=", OpponentNo: 15},
			{Round: 2, Color: "b", Result: "1", OpponentNo: 1},
			{Round: 3, Color: "w", Result: "0", OpponentNo: 7},
		},
	})

	trfContent := gen.Generate()

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
	// Note: TournamentType and ChiefArbiter codes (062, 112) are no longer generated
	// for JaVaFo compatibility, so we don't test for them here

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
	// Generate a properly formatted player line
	gen := NewGenerator()
	player := TRFPlayer{
		StartNo:    1,
		Sex:        "m",
		Title:      "GM",
		Name:       "Carlsen, Magnus",
		Rating:     2830,
		Federation: "NOR",
		FideID:     "1503014",
		BirthDate:  "1990/11/30",
		Points:     7.0,
		Rank:       1,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "w", Result: "1", OpponentNo: 2},
			{Round: 2, Color: "b", Result: "0", OpponentNo: 3},
		},
	}
	line := gen.generatePlayerLine(player)

	parser := NewParser()
	parsed, err := parser.parsePlayerLine(line)
	if err != nil {
		t.Fatalf("parsePlayerLine failed: %v", err)
	}

	if parsed.StartNo != 1 {
		t.Errorf("StartNo = %d, want 1", parsed.StartNo)
	}
	if parsed.Points != 7.0 {
		t.Errorf("Points = %v, want 7.0", parsed.Points)
	}

	// Check round results
	if len(parsed.RoundResults) < 2 {
		t.Fatalf("RoundResults count = %d, want >= 2", len(parsed.RoundResults))
	}

	// Round 1: white, win against 2
	r1 := parsed.RoundResults[0]
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
	// Use generator to create properly formatted round results
	gen := NewGenerator()

	tests := []struct {
		name      string
		input     TRFRoundResult
		expColor  string
		expResult string
		expOpp    int
	}{
		{"win white", TRFRoundResult{Round: 1, Color: "w", Result: "1", OpponentNo: 2}, "w", "1", 2},
		{"loss black", TRFRoundResult{Round: 1, Color: "b", Result: "0", OpponentNo: 5}, "b", "0", 5},
		{"draw white", TRFRoundResult{Round: 1, Color: "w", Result: "=", OpponentNo: 10}, "w", "=", 10},
		{"bye", TRFRoundResult{Round: 1, Color: "-", Result: "H", OpponentNo: 0}, "-", "H", 0},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate the round result string using the generator
			data := gen.generateRoundResult(tt.input)
			result := parser.parseRoundResult(1, data)

			if result.Color != tt.expColor {
				t.Errorf("Color = %q, want %q (data=%q)", result.Color, tt.expColor, data)
			}
			if result.Result != tt.expResult {
				t.Errorf("Result = %q, want %q (data=%q)", result.Result, tt.expResult, data)
			}
			if result.OpponentNo != tt.expOpp {
				t.Errorf("OpponentNo = %d, want %d (data=%q)", result.OpponentNo, tt.expOpp, data)
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
	// Use generator to create properly formatted TRF content
	gen := NewGenerator()
	gen.SetTournamentInfo("Test Tournament", "", "", 5)

	gen.AddPlayer(TRFPlayer{
		StartNo:    1,
		Name:       "Player One",
		Rating:     2000,
		BirthDate:  "1990/01/01",
		Points:     1.0,
		Rank:       1,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "w", Result: "1", OpponentNo: 2},
		},
	})
	gen.AddPlayer(TRFPlayer{
		StartNo:    2,
		Name:       "Player Two",
		Rating:     1900,
		BirthDate:  "1990/01/01",
		Points:     0.0,
		Rank:       2,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "b", Result: "0", OpponentNo: 1},
		},
	})
	gen.AddPlayer(TRFPlayer{
		StartNo:    3,
		Name:       "Player Three",
		Rating:     1800,
		BirthDate:  "1990/01/01",
		Points:     0.5,
		Rank:       3,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "-", Result: "H", OpponentNo: 0},
		},
	})

	trfContent := gen.Generate()

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
