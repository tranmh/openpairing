package engine

import (
	"testing"
)

func TestGenerateBergerTable_Correctness(t *testing.T) {
	// Comprehensive test for Berger table generation
	for n := 2; n <= 12; n++ {
		t.Run("", func(t *testing.T) {
			table := GenerateBergerTable(n)

			numPlayers := n
			if n%2 == 1 {
				numPlayers = n + 1
			}
			expectedRounds := numPlayers - 1

			if len(table) != expectedRounds {
				t.Errorf("n=%d: got %d rounds, want %d", n, len(table), expectedRounds)
			}

			// Verify each pair meets exactly once
			matchups := make(map[[2]int]int)
			for _, round := range table {
				for _, pair := range round {
					key := [2]int{minInt(pair[0], pair[1]), maxInt(pair[0], pair[1])}
					matchups[key]++
				}
			}

			for i := 1; i <= numPlayers; i++ {
				for j := i + 1; j <= numPlayers; j++ {
					key := [2]int{i, j}
					if matchups[key] != 1 {
						t.Errorf("n=%d: players %d and %d meet %d times", n, i, j, matchups[key])
					}
				}
			}
		})
	}
}

func TestCalculateByesForKnockout_Correctness(t *testing.T) {
	tests := []struct {
		players  int
		wantByes int
	}{
		{1, 0},
		{2, 0},
		{3, 1},
		{4, 0},
		{5, 3},
		{6, 2},
		{7, 1},
		{8, 0},
		{9, 7},
		{10, 6},
		{15, 1},
		{16, 0},
		{17, 15},
	}

	for _, tt := range tests {
		got := CalculateByesForKnockout(tt.players)
		if got != tt.wantByes {
			t.Errorf("CalculateByesForKnockout(%d) = %d, want %d", tt.players, got, tt.wantByes)
		}
	}
}

func TestIsPowerOfTwo_Correctness(t *testing.T) {
	// Powers of 2
	powersOf2 := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	for _, n := range powersOf2 {
		if !IsPowerOfTwo(n) {
			t.Errorf("IsPowerOfTwo(%d) = false, want true", n)
		}
	}

	// Not powers of 2
	notPowersOf2 := []int{0, 3, 5, 6, 7, 9, 10, 12, 15, 17, 100}
	for _, n := range notPowersOf2 {
		if IsPowerOfTwo(n) {
			t.Errorf("IsPowerOfTwo(%d) = true, want false", n)
		}
	}
}

func TestColor_Constants(t *testing.T) {
	if ColorWhite != "W" {
		t.Errorf("ColorWhite = %s, want W", ColorWhite)
	}
	if ColorBlack != "B" {
		t.Errorf("ColorBlack = %s, want B", ColorBlack)
	}
	if ColorNone != "" {
		t.Errorf("ColorNone = %s, want empty", ColorNone)
	}
}

func TestTournamentType_Constants(t *testing.T) {
	expectedTypes := map[TournamentType]string{
		TournamentTypeSwiss:        "swiss",
		TournamentTypeRoundRobin:   "round_robin",
		TournamentTypeKnockout:     "knockout",
		TournamentTypeTeamSwiss:    "team_swiss",
		TournamentTypeScheveningen: "scheveningen",
		TournamentTypeArena:        "arena",
	}

	for tt, expected := range expectedTypes {
		if string(tt) != expected {
			t.Errorf("TournamentType %v = %s, want %s", tt, string(tt), expected)
		}
	}
}

func TestKnockoutFormat_Constants(t *testing.T) {
	if KnockoutSingleElimination != "single" {
		t.Errorf("KnockoutSingleElimination = %s, want single", KnockoutSingleElimination)
	}
	if KnockoutDoubleElimination != "double" {
		t.Errorf("KnockoutDoubleElimination = %s, want double", KnockoutDoubleElimination)
	}
}

func TestFloatStatus_Constants(t *testing.T) {
	if FloatNone != "" {
		t.Errorf("FloatNone = %s, want empty", FloatNone)
	}
	if FloatUp != "U" {
		t.Errorf("FloatUp = %s, want U", FloatUp)
	}
	if FloatDown != "D" {
		t.Errorf("FloatDown = %s, want D", FloatDown)
	}
}

func TestPlayer_Struct(t *testing.T) {
	player := Player{
		StartNo:      1,
		Name:         "Test Player",
		Rating:       2000,
		Title:        "GM",
		FideID:       "12345678",
		Federation:   "USA",
		Points:       5.5,
		ColorHistory: []Color{ColorWhite, ColorBlack, ColorWhite},
		Opponents:    []int{2, 3, 4},
		IsActive:     true,
		ByeCount:     0,
		FloatStatus:  FloatNone,
		TeamID:       1,
		BoardNumber:  1,
	}

	if player.StartNo != 1 {
		t.Error("Player StartNo not set correctly")
	}
	if player.Name != "Test Player" {
		t.Error("Player Name not set correctly")
	}
	if player.Rating != 2000 {
		t.Error("Player Rating not set correctly")
	}
	if len(player.ColorHistory) != 3 {
		t.Error("Player ColorHistory length incorrect")
	}
	if len(player.Opponents) != 3 {
		t.Error("Player Opponents length incorrect")
	}
}

func TestTeam_Struct(t *testing.T) {
	team := Team{
		ID:   1,
		Name: "Test Team",
		Players: []Player{
			{StartNo: 1, Name: "P1"},
			{StartNo: 2, Name: "P2"},
		},
		Points: 3.5,
	}

	if team.ID != 1 {
		t.Error("Team ID not set correctly")
	}
	if team.Name != "Test Team" {
		t.Error("Team Name not set correctly")
	}
	if len(team.Players) != 2 {
		t.Error("Team Players length incorrect")
	}
	if team.Points != 3.5 {
		t.Error("Team Points not set correctly")
	}
}

func TestPairing_Struct(t *testing.T) {
	pairing := Pairing{
		Board:        1,
		WhiteStartNo: 5,
		BlackStartNo: 12,
		IsBye:        false,
	}

	if pairing.Board != 1 {
		t.Error("Pairing Board not set correctly")
	}
	if pairing.WhiteStartNo != 5 {
		t.Error("Pairing WhiteStartNo not set correctly")
	}
	if pairing.BlackStartNo != 12 {
		t.Error("Pairing BlackStartNo not set correctly")
	}
	if pairing.IsBye {
		t.Error("Pairing IsBye should be false")
	}

	// Test bye pairing
	byePairing := Pairing{
		Board:        2,
		WhiteStartNo: 7,
		BlackStartNo: 0,
		IsBye:        true,
	}

	if !byePairing.IsBye {
		t.Error("Bye pairing IsBye should be true")
	}
}
