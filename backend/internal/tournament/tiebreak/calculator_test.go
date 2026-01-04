package tiebreak

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuchholz_Standard(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBBuchholz}, 5)

	// Player 1: played against players with 3, 2.5, 3.5, 4, 2 points
	player1 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5,
		Results: []PlayerResult{
			{OpponentPts: 3},
			{OpponentPts: 2.5},
			{OpponentPts: 3.5},
			{OpponentPts: 4},
			{OpponentPts: 2},
		},
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	// Expected Buchholz: 3 + 2.5 + 3.5 + 4 + 2 = 15
	expected := 15.0
	if results[player1.ID][TBBuchholz] != expected {
		t.Errorf("Buchholz = %v, want %v", results[player1.ID][TBBuchholz], expected)
	}
}

func TestBuchholz_Cut1(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBBuchholzCut1}, 5)

	player1 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5,
		Results: []PlayerResult{
			{OpponentPts: 3},
			{OpponentPts: 2.5}, // This is the lowest, should be cut
			{OpponentPts: 3.5},
			{OpponentPts: 4},
			{OpponentPts: 2}, // This is lower but 2.5 comes first in sort
		},
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	// Expected Buchholz Cut 1: 15 - 2 (lowest) = 13
	expected := 13.0
	if results[player1.ID][TBBuchholzCut1] != expected {
		t.Errorf("Buchholz Cut 1 = %v, want %v", results[player1.ID][TBBuchholzCut1], expected)
	}
}

func TestBuchholz_Cut2(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBBuchholzCut2}, 5)

	player1 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5,
		Results: []PlayerResult{
			{OpponentPts: 3},
			{OpponentPts: 2.5},
			{OpponentPts: 3.5},
			{OpponentPts: 4},
			{OpponentPts: 2},
		},
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	// Expected Buchholz Cut 2: 15 - 2 - 2.5 = 10.5
	expected := 10.5
	if results[player1.ID][TBBuchholzCut2] != expected {
		t.Errorf("Buchholz Cut 2 = %v, want %v", results[player1.ID][TBBuchholzCut2], expected)
	}
}

func TestBuchholz_Median(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBBuchholzMedian}, 5)

	player1 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5,
		Results: []PlayerResult{
			{OpponentPts: 3},
			{OpponentPts: 2.5},
			{OpponentPts: 3.5},
			{OpponentPts: 4}, // Highest, should be cut
			{OpponentPts: 2}, // Lowest, should be cut
		},
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	// Expected Median: sorted [2, 2.5, 3, 3.5, 4] -> cut first and last -> 2.5 + 3 + 3.5 = 9
	expected := 9.0
	if results[player1.ID][TBBuchholzMedian] != expected {
		t.Errorf("Buchholz Median = %v, want %v", results[player1.ID][TBBuchholzMedian], expected)
	}
}

func TestSonnebornBerger(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBSonnebornBerger}, 5)

	player1 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5,
		Results: []PlayerResult{
			{Score: 1, OpponentPts: 3},    // Win: +3
			{Score: 0.5, OpponentPts: 2.5}, // Draw: +1.25
			{Score: 1, OpponentPts: 3.5},   // Win: +3.5
			{Score: 0, OpponentPts: 4},     // Loss: +0
			{Score: 1, OpponentPts: 2},     // Win: +2
		},
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	// Expected SB: 3 + 1.25 + 3.5 + 0 + 2 = 9.75
	expected := 9.75
	if results[player1.ID][TBSonnebornBerger] != expected {
		t.Errorf("Sonneborn-Berger = %v, want %v", results[player1.ID][TBSonnebornBerger], expected)
	}
}

func TestWins(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBWins}, 5)

	player1 := &PlayerData{
		ID:   uuid.New(),
		Wins: 4,
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	if results[player1.ID][TBWins] != 4 {
		t.Errorf("Wins = %v, want 4", results[player1.ID][TBWins])
	}
}

func TestGamesBlack(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBGamesBlack}, 5)

	player1 := &PlayerData{
		ID:           uuid.New(),
		GamesAsBlack: 3,
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	if results[player1.ID][TBGamesBlack] != 3 {
		t.Errorf("GamesBlack = %v, want 3", results[player1.ID][TBGamesBlack])
	}
}

func TestWinsBlack(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBWinsBlack}, 5)

	player1 := &PlayerData{
		ID:          uuid.New(),
		WinsAsBlack: 2,
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	if results[player1.ID][TBWinsBlack] != 2 {
		t.Errorf("WinsBlack = %v, want 2", results[player1.ID][TBWinsBlack])
	}
}

func TestProgress(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBProgress}, 5)

	// Progressive score: sum of cumulative points after each round
	// Round 1: 1 -> cumulative 1
	// Round 2: 0.5 -> cumulative 1.5
	// Round 3: 1 -> cumulative 2.5
	// Round 4: 0 -> cumulative 2.5
	// Round 5: 1 -> cumulative 3.5
	// Progress = 1 + 1.5 + 2.5 + 2.5 + 3.5 = 11
	player1 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5,
		Results: []PlayerResult{
			{Score: 1, RoundNumber: 1},
			{Score: 0.5, RoundNumber: 2},
			{Score: 1, RoundNumber: 3},
			{Score: 0, RoundNumber: 4},
			{Score: 1, RoundNumber: 5},
		},
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	// Progress calculation from results: 1 + (1+0.5) + (1+0.5+1) + (1+0.5+1+0) + (1+0.5+1+0+1)
	// = 1 + 1.5 + 2.5 + 2.5 + 3.5 = 11
	expected := 11.0
	if results[player1.ID][TBProgress] != expected {
		t.Errorf("Progress = %v, want %v", results[player1.ID][TBProgress], expected)
	}
}

func TestKoya(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBKoya}, 5)

	// Koya: points scored against opponents with >= 50% score (2.5 in 5-round)
	player1 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5,
		Results: []PlayerResult{
			{Score: 1, OpponentPts: 3},    // >= 2.5, counts: 1
			{Score: 0.5, OpponentPts: 2},  // < 2.5, doesn't count
			{Score: 1, OpponentPts: 2.5},  // >= 2.5, counts: 1
			{Score: 0, OpponentPts: 4},    // >= 2.5, counts: 0
			{Score: 1, OpponentPts: 1.5},  // < 2.5, doesn't count
		},
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	// Expected Koya: 1 + 1 + 0 = 2
	expected := 2.0
	if results[player1.ID][TBKoya] != expected {
		t.Errorf("Koya = %v, want %v", results[player1.ID][TBKoya], expected)
	}
}

func TestARCO(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBAROC}, 5)

	player1ID := uuid.New()
	player2ID := uuid.New()
	player3ID := uuid.New()
	player4ID := uuid.New()

	// Player 1 played against players with ratings: 2000, 1800, 2200, 1900
	player1 := &PlayerData{
		ID:     player1ID,
		Rating: 2100,
		Results: []PlayerResult{
			{OpponentID: player2ID},
			{OpponentID: player3ID},
			{OpponentID: player4ID},
		},
	}

	player2 := &PlayerData{
		ID:     player2ID,
		Rating: 2000,
	}
	player3 := &PlayerData{
		ID:     player3ID,
		Rating: 1800,
	}
	player4 := &PlayerData{
		ID:     player4ID,
		Rating: 2200,
	}

	calc.AddPlayer(player1)
	calc.AddPlayer(player2)
	calc.AddPlayer(player3)
	calc.AddPlayer(player4)

	results := calc.Calculate()

	// AROC: avg of opponents' ratings with lowest cut
	// Ratings: [1800, 2000, 2200] -> cut lowest (1800) -> avg(2000, 2200) = 2100
	expected := 2100.0
	if results[player1.ID][TBAROC] != expected {
		t.Errorf("AROC = %v, want %v", results[player1.ID][TBAROC], expected)
	}
}

func TestAvgOpponentRating(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBAvgOppRating}, 5)

	player1ID := uuid.New()
	player2ID := uuid.New()
	player3ID := uuid.New()

	player1 := &PlayerData{
		ID:     player1ID,
		Rating: 2100,
		Results: []PlayerResult{
			{OpponentID: player2ID},
			{OpponentID: player3ID},
		},
	}

	player2 := &PlayerData{
		ID:     player2ID,
		Rating: 2000,
	}
	player3 := &PlayerData{
		ID:     player3ID,
		Rating: 1800,
	}

	calc.AddPlayer(player1)
	calc.AddPlayer(player2)
	calc.AddPlayer(player3)

	results := calc.Calculate()

	// Avg opponent rating: (2000 + 1800) / 2 = 1900
	expected := 1900.0
	if results[player1.ID][TBAvgOppRating] != expected {
		t.Errorf("AvgOppRating = %v, want %v", results[player1.ID][TBAvgOppRating], expected)
	}
}

func TestGetStandings(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBBuchholz, TBSonnebornBerger}, 5)

	// Create three players with different points and tiebreaks
	player1 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5,
		Results: []PlayerResult{
			{Score: 1, OpponentPts: 3},
			{Score: 1, OpponentPts: 2},
		},
	}
	player2 := &PlayerData{
		ID:     uuid.New(),
		Points: 3.5, // Same points as player1
		Results: []PlayerResult{
			{Score: 1, OpponentPts: 2.5},
			{Score: 1, OpponentPts: 2},
		},
	}
	player3 := &PlayerData{
		ID:     uuid.New(),
		Points: 4, // Higher points
		Results: []PlayerResult{
			{Score: 1, OpponentPts: 3},
			{Score: 1, OpponentPts: 3},
		},
	}

	calc.AddPlayer(player1)
	calc.AddPlayer(player2)
	calc.AddPlayer(player3)

	standings := calc.GetStandings()

	// Player 3 should be first (4 points)
	if standings[0].PlayerID != player3.ID {
		t.Errorf("First place should be player3")
	}
	if standings[0].Rank != 1 {
		t.Errorf("First place rank = %d, want 1", standings[0].Rank)
	}

	// Player 1 should be second (3.5 points, higher Buchholz: 5 vs 4.5)
	if standings[1].PlayerID != player1.ID {
		t.Errorf("Second place should be player1 (higher Buchholz)")
	}

	// Player 2 should be third
	if standings[2].PlayerID != player2.ID {
		t.Errorf("Third place should be player2")
	}
}

func TestTiebreakName(t *testing.T) {
	tests := []struct {
		code TiebreakCode
		name string
	}{
		{TBBuchholz, "Buchholz"},
		{TBBuchholzCut1, "Buchholz Cut 1"},
		{TBSonnebornBerger, "Sonneborn-Berger"},
		{TBKoya, "Koya-System"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			name := TiebreakName(tt.code)
			if name != tt.name {
				t.Errorf("TiebreakName(%s) = %s, want %s", tt.code, name, tt.name)
			}
		})
	}
}

func TestAvailableTiebreaks(t *testing.T) {
	tiebreaks := AvailableTiebreaks()

	if len(tiebreaks) < 15 {
		t.Errorf("Expected at least 15 tiebreaks, got %d", len(tiebreaks))
	}

	// Check some required tiebreaks exist
	required := []TiebreakCode{TBBuchholz, TBSonnebornBerger, TBWins, TBProgress}
	for _, req := range required {
		found := false
		for _, tb := range tiebreaks {
			if tb == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required tiebreak %s not found", req)
		}
	}
}

func TestEmptyResults(t *testing.T) {
	calc := NewCalculator([]TiebreakCode{TBBuchholz, TBSonnebornBerger, TBProgress}, 5)

	player1 := &PlayerData{
		ID:      uuid.New(),
		Points:  0,
		Results: []PlayerResult{},
	}
	calc.AddPlayer(player1)

	results := calc.Calculate()

	// All tiebreaks should be 0 for player with no games
	if results[player1.ID][TBBuchholz] != 0 {
		t.Errorf("Buchholz for no games = %v, want 0", results[player1.ID][TBBuchholz])
	}
	if results[player1.ID][TBSonnebornBerger] != 0 {
		t.Errorf("SB for no games = %v, want 0", results[player1.ID][TBSonnebornBerger])
	}
	if results[player1.ID][TBProgress] != 0 {
		t.Errorf("Progress for no games = %v, want 0", results[player1.ID][TBProgress])
	}
}
