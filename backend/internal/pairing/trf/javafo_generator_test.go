package trf

import (
	"strings"
	"testing"
	"time"
)

func TestJaVaFoGenerator_EngineName(t *testing.T) {
	g := NewJaVaFoGenerator()
	if g.EngineName() != "JaVaFo" {
		t.Errorf("EngineName() = %q, want %q", g.EngineName(), "JaVaFo")
	}
}

func TestJaVaFoGenerator_Generate_BasicHeader(t *testing.T) {
	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Test Tournament 2024", "Berlin", "GER", 7)
	g.SetDates(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC))
	g.AddPlayer(TRFPlayer{
		StartNo: 1,
		Name:    "Carlsen, Magnus",
		Rating:  2830,
		Points:  4.5,
	})

	output := g.Generate()

	// Should contain basic header codes
	if !strings.Contains(output, "012 Test Tournament 2024") {
		t.Error("Expected 012 tournament name")
	}
	if !strings.Contains(output, "022 Berlin") {
		t.Error("Expected 022 city")
	}
	if !strings.Contains(output, "032 GER") {
		t.Error("Expected 032 federation")
	}
	if !strings.Contains(output, "042 2024/01/15") {
		t.Error("Expected 042 start date")
	}
	if !strings.Contains(output, "052 2024/01/20") {
		t.Error("Expected 052 end date")
	}

	// Should contain XXR for rounds
	if !strings.Contains(output, "XXR 7") {
		t.Error("Expected XXR record for rounds")
	}

	// Should contain blank line marker
	if !strings.Contains(output, "122") {
		t.Error("Expected 122 blank line marker")
	}
}

func TestJaVaFoGenerator_NoForbiddenHeaderCodes(t *testing.T) {
	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Test", "City", "FED", 5)
	g.SetTimeControl("90+30")
	g.SetTournamentType("Swiss")
	g.SetArbiters("Chief Arbiter", []string{"Deputy 1"})
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Test Player"})

	output := g.Generate()

	// These codes cause JaVaFo to fail - must NOT be present
	forbiddenCodes := []string{"062 ", "072 ", "082 ", "092 ", "102 ", "112 "}
	for _, code := range forbiddenCodes {
		if strings.Contains(output, code) {
			t.Errorf("Output should NOT contain %q (JaVaFo incompatible)", code)
		}
	}
}

func TestJaVaFoGenerator_XXCRecordsOptional(t *testing.T) {
	// Without XXC
	g1 := NewJaVaFoGenerator()
	g1.SetTournamentInfo("Test", "", "", 5)
	g1.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player"})

	output1 := g1.Generate()
	if strings.Contains(output1, "XXC ") {
		t.Error("XXC should not be present when not set")
	}

	// With XXC
	g2 := NewJaVaFoGenerator()
	g2.SetTournamentInfo("Test", "", "", 5)
	g2.SetInitialColors("white1")
	g2.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player"})

	output2 := g2.Generate()
	if !strings.Contains(output2, "XXC white1") {
		t.Error("XXC should be present when set")
	}
}

func TestJaVaFoGenerator_PlayerLine(t *testing.T) {
	g := NewJaVaFoGenerator()
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
		t.Error("Expected 001 player line")
	}
	if !strings.Contains(output, "GM") {
		t.Error("Expected title in output")
	}
	if !strings.Contains(output, "Carlsen, Magnus") {
		t.Error("Expected player name in output")
	}
	if !strings.Contains(output, "2830") {
		t.Error("Expected rating in output")
	}
	if !strings.Contains(output, "NOR") {
		t.Error("Expected federation in output")
	}
}

func TestJaVaFoGenerator_Validate_Valid(t *testing.T) {
	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Test Tournament", "", "", 5)
	g.AddPlayer(TRFPlayer{
		StartNo: 1,
		Name:    "Player One",
	})

	errors := g.Validate()

	for _, e := range errors {
		if e.Fatal {
			t.Errorf("Unexpected fatal error: %s", e.Message)
		}
	}
}

func TestJaVaFoGenerator_Validate_MissingName(t *testing.T) {
	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("", "", "", 5)
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player"})

	errors := g.Validate()

	hasFatalNameError := false
	for _, e := range errors {
		if e.Fatal && strings.Contains(e.Message, "name") {
			hasFatalNameError = true
		}
	}
	if !hasFatalNameError {
		t.Error("Expected fatal error for missing tournament name")
	}
}

func TestJaVaFoGenerator_Validate_NoPlayers(t *testing.T) {
	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Test", "", "", 5)
	// No players added

	errors := g.Validate()

	hasFatalPlayerError := false
	for _, e := range errors {
		if e.Fatal && strings.Contains(e.Message, "player") {
			hasFatalPlayerError = true
		}
	}
	if !hasFatalPlayerError {
		t.Error("Expected fatal error for no players")
	}
}

func TestJaVaFoGenerator_Validate_MissingRounds(t *testing.T) {
	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Test", "", "", 0) // 0 rounds
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player"})

	errors := g.Validate()

	hasFatalRoundsError := false
	for _, e := range errors {
		if e.Fatal && strings.Contains(e.Message, "round") {
			hasFatalRoundsError = true
		}
	}
	if !hasFatalRoundsError {
		t.Error("Expected fatal error for missing rounds")
	}
}

func TestJaVaFoGenerator_RoundResults(t *testing.T) {
	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Test", "", "", 5)

	player := TRFPlayer{
		StartNo: 1,
		Name:    "Player One",
		RoundResults: []TRFRoundResult{
			{Round: 1, Color: "w", Result: "1", OpponentNo: 2},
			{Round: 2, Color: "b", Result: "0", OpponentNo: 3},
			{Round: 3, Color: "w", Result: "=", OpponentNo: 4},
			{Round: 4, Color: "-", Result: "H", OpponentNo: 0},
			{Round: 5, Color: "-", Result: "Z", OpponentNo: 0},
		},
	}
	g.AddPlayer(player)

	output := g.Generate()

	// Check round results are present
	if !strings.Contains(output, "w 1") {
		t.Error("Expected win with white")
	}
	if !strings.Contains(output, "b 0") {
		t.Error("Expected loss with black")
	}
	if !strings.Contains(output, "w =") {
		t.Error("Expected draw with white")
	}
	if !strings.Contains(output, "- H") {
		t.Error("Expected half-point bye")
	}
	if !strings.Contains(output, "- Z") {
		t.Error("Expected zero-point bye")
	}
}

func TestJaVaFoGenerator_TeamLine(t *testing.T) {
	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Team Tournament", "", "", 9)

	team := TRFTeam{
		TeamNo:    1,
		TeamName:  "Berlin Chess Club",
		PlayerNos: []int{1, 2, 3, 4},
	}
	g.AddTeam(team)
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player 1"})

	output := g.Generate()

	if !strings.Contains(output, "013") {
		t.Error("Expected 013 team line")
	}
	if !strings.Contains(output, "Berlin Chess Club") {
		t.Error("Expected team name in output")
	}
}

func TestJaVaFoGenerator_ImplementsInterface(t *testing.T) {
	var _ TRFGenerator = (*JaVaFoGenerator)(nil)
}
