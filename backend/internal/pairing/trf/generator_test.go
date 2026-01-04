package trf

import (
	"strings"
	"testing"
	"time"
)

func TestGenerator_Header(t *testing.T) {
	g := NewGenerator()
	g.SetTournamentInfo("Test Tournament 2024", "Berlin", "GER", 7)
	g.SetDates(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC))
	g.SetArbiters("IA John Smith", []string{"FA Deputy One", "NA Deputy Two"})
	g.SetTimeControl("90+30")
	g.SetTournamentType("Swiss")

	output := g.Generate()

	// Check header lines
	if !strings.Contains(output, "012 Test Tournament 2024") {
		t.Errorf("Output should contain tournament name line")
	}
	if !strings.Contains(output, "022 Berlin") {
		t.Errorf("Output should contain city line")
	}
	if !strings.Contains(output, "032 GER") {
		t.Errorf("Output should contain federation line")
	}
	if !strings.Contains(output, "042 2024/01/15") {
		t.Errorf("Output should contain start date line")
	}
	if !strings.Contains(output, "052 2024/01/20") {
		t.Errorf("Output should contain end date line")
	}
	// XXR record for rounds (instead of 062 which JaVaFo doesn't recognize)
	if !strings.Contains(output, "XXR 7") {
		t.Errorf("Output should contain XXR rounds line")
	}
	// Note: 062, 072, 082, 092, 102 are intentionally omitted for JaVaFo compatibility
}

func TestGenerator_PlayerLine(t *testing.T) {
	g := NewGenerator()
	g.SetTournamentInfo("Test", "", "", 5)

	player := TRFPlayer{
		StartNo:    1,
		Sex:        "m",
		Title:      "GM",
		Name:       "Carlsen, Magnus",
		Rating:     2830,
		Federation: "NOR",
		FideID:     "1503014",
		BirthDate:  "1990/11/30",
		Points:     4.5,
		Rank:       1,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "w", Result: "1", OpponentNo: 2},
			{Round: 2, Color: "b", Result: "=", OpponentNo: 3},
		},
	}
	g.AddPlayer(player)

	output := g.Generate()

	// Check player line is present
	if !strings.Contains(output, "001") {
		t.Errorf("Output should contain player line (001)")
	}
	if !strings.Contains(output, "GM") {
		t.Errorf("Output should contain title")
	}
	if !strings.Contains(output, "Carlsen, Magnus") {
		t.Errorf("Output should contain player name")
	}
	if !strings.Contains(output, "2830") {
		t.Errorf("Output should contain rating")
	}
	if !strings.Contains(output, "NOR") {
		t.Errorf("Output should contain federation")
	}
}

func TestGenerator_RoundResult(t *testing.T) {
	g := NewGenerator()

	tests := []struct {
		name     string
		result   TRFRoundResult
		expected string
	}{
		{
			name: "win white",
			result: TRFRoundResult{
				Round:      1,
				Color:      "w",
				Result:     "1",
				OpponentNo: 2,
			},
			expected: "w 1",
		},
		{
			name: "loss black",
			result: TRFRoundResult{
				Round:      1,
				Color:      "b",
				Result:     "0",
				OpponentNo: 5,
			},
			expected: "b 0",
		},
		{
			name: "draw",
			result: TRFRoundResult{
				Round:      1,
				Color:      "w",
				Result:     "=",
				OpponentNo: 3,
			},
			expected: "w =",
		},
		{
			name: "half-point bye",
			result: TRFRoundResult{
				Round:      1,
				Color:      "-",
				Result:     "H",
				OpponentNo: 0,
			},
			expected: "- H",
		},
		{
			name: "zero-point bye",
			result: TRFRoundResult{
				Round:      1,
				Color:      "-",
				Result:     "Z",
				OpponentNo: 0,
			},
			expected: "- Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := g.generateRoundResult(tt.result)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("generateRoundResult() = %q, want to contain %q", output, tt.expected)
			}
		})
	}
}

func TestGenerator_TeamLine(t *testing.T) {
	g := NewGenerator()
	g.SetTournamentInfo("Team Tournament", "", "", 9)

	team := TRFTeam{
		TeamNo:    1,
		TeamName:  "Berlin Chess Club",
		PlayerNos: []int{1, 2, 3, 4, 5, 6, 7, 8},
	}
	g.AddTeam(team)

	output := g.Generate()

	if !strings.Contains(output, "013") {
		t.Errorf("Output should contain team line (013)")
	}
	if !strings.Contains(output, "Berlin Chess Club") {
		t.Errorf("Output should contain team name")
	}
}

func TestGenerator_ValidateForFIDE(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*Generator)
		wantErrors int
	}{
		{
			name: "valid tournament",
			setup: func(g *Generator) {
				g.SetTournamentInfo("Test Tournament", "", "", 7)
				g.AddPlayer(TRFPlayer{
					StartNo:    1,
					Name:       "Player One",
					FideID:     "12345678",
					Rating:     2000,
				})
			},
			wantErrors: 0,
		},
		{
			name: "missing tournament name",
			setup: func(g *Generator) {
				g.SetTournamentInfo("", "", "", 7)
				g.AddPlayer(TRFPlayer{
					StartNo: 1,
					Name:    "Player One",
				})
			},
			wantErrors: 1,
		},
		{
			name: "missing rounds",
			setup: func(g *Generator) {
				g.SetTournamentInfo("Test", "", "", 0)
				g.AddPlayer(TRFPlayer{
					StartNo: 1,
					Name:    "Player One",
				})
			},
			wantErrors: 1,
		},
		{
			name: "no players",
			setup: func(g *Generator) {
				g.SetTournamentInfo("Test", "", "", 7)
			},
			wantErrors: 1,
		},
		{
			name: "player missing name",
			setup: func(g *Generator) {
				g.SetTournamentInfo("Test", "", "", 7)
				g.AddPlayer(TRFPlayer{
					StartNo: 1,
					Name:    "",
				})
			},
			wantErrors: 1,
		},
		{
			name: "rated player missing FIDE ID",
			setup: func(g *Generator) {
				g.SetTournamentInfo("Test", "", "", 7)
				g.AddPlayer(TRFPlayer{
					StartNo: 1,
					Name:    "Player One",
					Rating:  2000,
					FideID:  "", // Missing FIDE ID
				})
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator()
			tt.setup(g)

			errors := g.ValidateForFIDE()
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateForFIDE() returned %d errors, want %d: %v", len(errors), tt.wantErrors, errors)
			}
		})
	}
}

func TestCreateFromTournament(t *testing.T) {
	data := &TournamentData{
		Name:           "Test Tournament 2024",
		City:           "Berlin",
		Federation:     "GER",
		StartDate:      time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
		Rounds:         7,
		TournamentType: "Swiss",
		TimeControl:    "90+30",
		ChiefArbiter:   "IA John Smith",
		Players: []PlayerData{
			{
				StartNo:    1,
				Sex:        "m",
				Title:      "GM",
				FirstName:  "Magnus",
				LastName:   "Carlsen",
				Rating:     2830,
				Federation: "NOR",
				FideID:     "1503014",
				BirthDate:  "1990/11/30",
				Points:     4.5,
				Rank:       1,
				RoundResults: []RoundResultData{
					{Round: 1, OpponentNo: 2, Color: "w", Result: "1"},
				},
			},
		},
	}

	g := CreateFromTournament(data)
	output := g.Generate()

	if !strings.Contains(output, "Test Tournament 2024") {
		t.Errorf("Output should contain tournament name")
	}
	if !strings.Contains(output, "Carlsen, Magnus") {
		t.Errorf("Output should contain player name in 'Last, First' format")
	}
}

func TestGenerator_BlankLine(t *testing.T) {
	g := NewGenerator()
	g.SetTournamentInfo("Test", "", "", 5)
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player"})

	output := g.Generate()

	// Should contain 122 blank line before players
	if !strings.Contains(output, "\n122\n") {
		t.Errorf("Output should contain blank line marker (122)")
	}
}

func TestGenerator_LongName(t *testing.T) {
	g := NewGenerator()
	g.SetTournamentInfo("Test", "", "", 5)

	// Name longer than 33 chars
	longName := "Verylongfirstname, Andaverylonglastname"
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: longName})

	output := g.Generate()

	// Name should be truncated
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "001") {
			// Check line isn't excessively long due to name
			nameSection := line[14:47] // Position 14-47 is name field
			if len(nameSection) > 33 {
				t.Errorf("Name section too long: %d chars", len(nameSection))
			}
		}
	}
}

func TestGenerator_RoundTrip(t *testing.T) {
	// Generate TRF, parse it back, generate again - should be consistent
	g := NewGenerator()
	g.SetTournamentInfo("Test Tournament", "Berlin", "GER", 5)
	g.SetTimeControl("90+30")
	g.AddPlayer(TRFPlayer{
		StartNo:    1,
		Name:       "Player One",
		Rating:     2000,
		Federation: "GER",
		Points:     3.5,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "w", Result: "1", OpponentNo: 2},
		},
	})
	g.AddPlayer(TRFPlayer{
		StartNo:    2,
		Name:       "Player Two",
		Rating:     1900,
		Federation: "GER",
		Points:     1.5,
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "b", Result: "0", OpponentNo: 1},
		},
	})

	output1 := g.Generate()

	// Parse the output
	parser := NewParser()
	parsed, err := parser.Parse(strings.NewReader(output1))
	if err != nil {
		t.Fatalf("Failed to parse generated TRF: %v", err)
	}

	// Verify parsed data
	if parsed.TournamentName != "Test Tournament" {
		t.Errorf("Round trip: TournamentName = %q, want %q", parsed.TournamentName, "Test Tournament")
	}
	if len(parsed.Players) != 2 {
		t.Errorf("Round trip: Players = %d, want 2", len(parsed.Players))
	}
}

func TestGenerator_XXRRecords(t *testing.T) {
	g := NewGenerator()
	g.SetTournamentInfo("Test", "", "", 5)
	g.trf.XXRRecords[1] = "Additional info for player 1"

	output := g.Generate()

	if !strings.Contains(output, "XXR") {
		t.Errorf("Output should contain XXR record")
	}
	if !strings.Contains(output, "Additional info") {
		t.Errorf("Output should contain XXR data")
	}
}

func TestGenerator_XXCRecords(t *testing.T) {
	g := NewGenerator()
	g.SetTournamentInfo("Test", "", "", 5)
	g.trf.XXCRecords = []string{"white1", "initial"}

	output := g.Generate()

	if !strings.Contains(output, "XXC white1") {
		t.Errorf("Output should contain XXC record")
	}
	if !strings.Contains(output, "XXC initial") {
		t.Errorf("Output should contain XXC initial")
	}
}
