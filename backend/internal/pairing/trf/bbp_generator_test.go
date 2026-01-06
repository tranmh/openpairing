package trf

import (
	"strings"
	"testing"
	"time"
)

func TestBBPGenerator_EngineName(t *testing.T) {
	g := NewBBPGenerator()
	if g.EngineName() != "bbpPairings" {
		t.Errorf("EngineName() = %q, want %q", g.EngineName(), "bbpPairings")
	}
}

func TestBBPGenerator_Generate_RequiresXXC(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test Tournament", "", "", 5)
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player One", Points: 0})

	output := g.Generate()

	// BBP REQUIRES XXC record
	if !strings.Contains(output, "XXC ") {
		t.Error("BBP output must contain XXC record")
	}
}

func TestBBPGenerator_DefaultInitialColor(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test", "", "", 5)
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player", Points: 0})

	output := g.Generate()

	// Default should be "white1"
	if !strings.Contains(output, "XXC white1") {
		t.Errorf("Expected default XXC white1, got output:\n%s", output)
	}
}

func TestBBPGenerator_CustomInitialColor(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test", "", "", 5)
	g.SetInitialColorConfig(BBPColorBlack1)
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player", Points: 0})

	output := g.Generate()

	if !strings.Contains(output, "XXC black1") {
		t.Errorf("Expected XXC black1, got output:\n%s", output)
	}
}

func TestBBPGenerator_SetInitialColors_OverridesDefault(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test", "", "", 5)
	g.SetInitialColors("rank")
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player", Points: 0})

	output := g.Generate()

	if !strings.Contains(output, "XXC rank") {
		t.Errorf("Expected XXC rank, got output:\n%s", output)
	}
	// Should not have default white1
	if strings.Contains(output, "XXC white1") {
		t.Error("Should not have default XXC white1 when SetInitialColors is used")
	}
}

func TestBBPGenerator_Generate_BasicHeader(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test Tournament 2024", "Berlin", "GER", 7)
	g.SetDates(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC))
	g.AddPlayer(TRFPlayer{
		StartNo: 1,
		Name:    "Carlsen, Magnus",
		Rating:  2830,
		Points:  0,
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

	// Should contain XXR for rounds
	if !strings.Contains(output, "XXR 7") {
		t.Error("Expected XXR record for rounds")
	}
}

func TestBBPGenerator_Validate_ScoreMismatch(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test", "", "", 5)

	// Player claims 3.0 points but results show 2.0
	g.AddPlayer(TRFPlayer{
		StartNo: 1,
		Name:    "Player One",
		Points:  3.0,
		RoundResults: []TRFRoundResult{
			{Round: 1, Result: "1"},  // 1.0
			{Round: 2, Result: "="},  // 0.5
			{Round: 3, Result: "H"},  // 0.5
			// Total: 2.0, but declared 3.0
		},
	})

	errors := g.Validate()

	// Should have a fatal error for score mismatch
	hasFatalScoreError := false
	for _, e := range errors {
		if e.Fatal && strings.Contains(e.Message, "points") {
			hasFatalScoreError = true
			break
		}
	}

	if !hasFatalScoreError {
		t.Error("Expected fatal validation error for score mismatch")
	}
}

func TestBBPGenerator_Validate_CorrectScores(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test", "", "", 5)

	// Correct: 1 + 0.5 + 0.5 = 2.0
	g.AddPlayer(TRFPlayer{
		StartNo: 1,
		Name:    "Player One",
		Points:  2.0,
		RoundResults: []TRFRoundResult{
			{Round: 1, Result: "1"},
			{Round: 2, Result: "="},
			{Round: 3, Result: "H"},
		},
	})

	errors := g.Validate()

	for _, e := range errors {
		if e.Fatal {
			t.Errorf("Unexpected fatal error: %s", e.Message)
		}
	}
}

func TestBBPGenerator_Validate_NoResultsNoValidation(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test", "", "", 5)

	// Player with points but no results (round 1 - no games played yet)
	g.AddPlayer(TRFPlayer{
		StartNo:      1,
		Name:         "Player One",
		Points:       0,
		RoundResults: []TRFRoundResult{}, // Empty - first round
	})

	errors := g.Validate()

	// Should not have fatal errors for score when no results
	for _, e := range errors {
		if e.Fatal && strings.Contains(e.Message, "points") {
			t.Errorf("Should not validate scores when no results: %s", e.Message)
		}
	}
}

func TestBBPGenerator_CalculatePoints(t *testing.T) {
	g := NewBBPGenerator()

	tests := []struct {
		name    string
		results []TRFRoundResult
		want    float64
	}{
		{
			name: "all wins",
			results: []TRFRoundResult{
				{Result: "1"},
				{Result: "+"},
				{Result: "W"},
				{Result: "F"},
			},
			want: 4.0,
		},
		{
			name: "all draws",
			results: []TRFRoundResult{
				{Result: "="},
				{Result: "D"},
				{Result: "H"},
			},
			want: 1.5,
		},
		{
			name: "all losses",
			results: []TRFRoundResult{
				{Result: "0"},
				{Result: "-"},
				{Result: "L"},
				{Result: "Z"},
				{Result: "U"},
			},
			want: 0.0,
		},
		{
			name: "mixed results",
			results: []TRFRoundResult{
				{Result: "1"},  // 1.0
				{Result: "="},  // 0.5
				{Result: "0"},  // 0.0
				{Result: "H"},  // 0.5
				{Result: "F"},  // 1.0
			},
			want: 3.0,
		},
		{
			name:    "empty results",
			results: []TRFRoundResult{},
			want:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.calculatePoints(tt.results)
			if got != tt.want {
				t.Errorf("calculatePoints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBBPGenerator_Validate_MissingName(t *testing.T) {
	g := NewBBPGenerator()
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

func TestBBPGenerator_Validate_NoPlayers(t *testing.T) {
	g := NewBBPGenerator()
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

func TestBBPGenerator_XXCBeforeXXR(t *testing.T) {
	g := NewBBPGenerator()
	g.SetTournamentInfo("Test", "", "", 5)
	g.AddPlayer(TRFPlayer{StartNo: 1, Name: "Player", Points: 0})

	output := g.Generate()

	xxcIndex := strings.Index(output, "XXC")
	xxrIndex := strings.Index(output, "XXR")

	if xxcIndex == -1 {
		t.Fatal("XXC not found in output")
	}
	if xxrIndex == -1 {
		t.Fatal("XXR not found in output")
	}

	if xxcIndex > xxrIndex {
		t.Error("XXC should appear before XXR in BBP output")
	}
}

func TestBBPGenerator_ImplementsInterface(t *testing.T) {
	var _ TRFGenerator = (*BBPGenerator)(nil)
}
