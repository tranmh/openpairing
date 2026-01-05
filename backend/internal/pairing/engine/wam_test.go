package engine

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestGroupNameConversion(t *testing.T) {
	tests := []struct {
		number int
		name   string
	}{
		{1, "A"},
		{2, "B"},
		{3, "C"},
		{26, "Z"},
		{0, ""},  // Invalid
		{27, ""}, // Out of range
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupNameFromNumber(tt.number)
			if got != tt.name {
				t.Errorf("GroupNameFromNumber(%d) = %q, want %q", tt.number, got, tt.name)
			}
		})
	}
}

func TestGroupNumberFromName(t *testing.T) {
	tests := []struct {
		name   string
		number int
	}{
		{"A", 1},
		{"B", 2},
		{"Z", 26},
		{"a", 1}, // Lowercase
		{"b", 2},
		{"", 0},   // Empty
		{"AB", 0}, // Multi-char
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupNumberFromName(tt.name)
			if got != tt.number {
				t.Errorf("GroupNumberFromName(%q) = %d, want %d", tt.name, got, tt.number)
			}
		})
	}
}

func TestGetTotalGroupRounds(t *testing.T) {
	tests := []struct {
		groupSize int
		expected  int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{3, 3}, // Odd: n rounds with byes
		{4, 3},
		{5, 5}, // Odd: n rounds with byes
		{6, 5},
		{7, 7},
		{8, 7},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := GetTotalGroupRounds(tt.groupSize)
			if got != tt.expected {
				t.Errorf("GetTotalGroupRounds(%d) = %d, want %d", tt.groupSize, got, tt.expected)
			}
		})
	}
}

func TestWAMAssignGroupsSequential(t *testing.T) {
	wam := NewWAM(testLogger())

	// 16 players, groups of 4 -> 4 groups
	players := createTestPlayers(16)
	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSequential,
	}

	groups, warnings := wam.AssignGroups(players, config)

	if len(groups) != 4 {
		t.Errorf("Expected 4 groups, got %d", len(groups))
	}

	// Check group names
	expectedNames := []string{"A", "B", "C", "D"}
	for i, g := range groups {
		if g.Name != expectedNames[i] {
			t.Errorf("Group %d: expected name %q, got %q", i, expectedNames[i], g.Name)
		}
		if len(g.Players) != 4 {
			t.Errorf("Group %s: expected 4 players, got %d", g.Name, len(g.Players))
		}
	}

	// Sequential: Group A should have highest rated players (1600, 1590, 1580, 1570)
	groupA := groups[0]
	for i, p := range groupA.Players {
		expectedRating := 1600 - (i * 10)
		if p.Rating != expectedRating {
			t.Errorf("Group A player %d: expected rating %d, got %d", i, expectedRating, p.Rating)
		}
	}

	if len(warnings) != 0 {
		t.Errorf("Expected no warnings for even distribution, got %v", warnings)
	}
}

func TestWAMAssignGroupsSerpentine(t *testing.T) {
	wam := NewWAM(testLogger())

	// 16 players, groups of 4 -> 4 groups
	players := createTestPlayers(16)
	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSerpentine,
	}

	groups, _ := wam.AssignGroups(players, config)

	if len(groups) != 4 {
		t.Errorf("Expected 4 groups, got %d", len(groups))
	}

	// Serpentine pattern: 1→A, 2→B, 3→C, 4→D, 5→D, 6→C, 7→B, 8→A
	// Group A should have players with ratings: 1600, 1530, 1520, 1450
	// (1st, 8th, 9th, 16th by rating)
	groupA := groups[0]
	expectedRatingsA := []int{1600, 1530, 1520, 1450}
	for i, p := range groupA.Players {
		if p.Rating != expectedRatingsA[i] {
			t.Errorf("Serpentine Group A player %d: expected rating %d, got %d", i, expectedRatingsA[i], p.Rating)
		}
	}

	// Group D should have: 1570, 1560, 1490, 1480
	groupD := groups[3]
	expectedRatingsD := []int{1570, 1560, 1490, 1480}
	for i, p := range groupD.Players {
		if p.Rating != expectedRatingsD[i] {
			t.Errorf("Serpentine Group D player %d: expected rating %d, got %d", i, expectedRatingsD[i], p.Rating)
		}
	}
}

func TestWAMAssignGroupsUnevenDistribution(t *testing.T) {
	wam := NewWAM(testLogger())

	// 14 players, groups of 4 -> 3 full groups + 1 group with 2 players
	players := createTestPlayers(14)
	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSequential,
		YouthAgeLimit:    12,
	}

	groups, warnings := wam.AssignGroups(players, config)

	if len(groups) != 4 {
		t.Errorf("Expected 4 groups, got %d", len(groups))
	}

	// Check last group has 2 players
	lastGroup := groups[len(groups)-1]
	if len(lastGroup.Players) != 2 {
		t.Errorf("Last group should have 2 players, got %d", len(lastGroup.Players))
	}

	// Should have a warning about uneven group
	foundWarning := false
	for _, w := range warnings {
		if containsString(w, "2 players instead of 4") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("Expected warning about uneven group size, got %v", warnings)
	}
}

func TestWAMAssignGroupsLargeLastGroup(t *testing.T) {
	wam := NewWAM(testLogger())

	// 18 players, groups of 4 -> 4 groups of 4 + 1 group of 2, or adjusted
	// Actually: 18 / 4 = 4 remainder 2, so 5 groups with last having 2
	// But if we have 17 players: 4 groups of 4 + 1 of 1? No, should be 4+4+4+5
	// Let's test 22 players: 5 groups of 4 + 1 of 2 = 22
	players := createTestPlayers(22)
	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSequential,
		YouthAgeLimit:    12,
	}

	groups, warnings := wam.AssignGroups(players, config)

	// 22 / 4 = 5 remainder 2 -> 6 groups, last with 2
	if len(groups) != 6 {
		t.Errorf("Expected 6 groups for 22 players with group size 4, got %d", len(groups))
	}

	// Check total players assigned
	totalAssigned := 0
	for _, g := range groups {
		totalAssigned += len(g.Players)
	}
	if totalAssigned != 22 {
		t.Errorf("Expected 22 total players assigned, got %d", totalAssigned)
	}

	// Last group should have 2 players
	lastGroup := groups[len(groups)-1]
	if len(lastGroup.Players) != 2 {
		t.Errorf("Last group should have 2 players, got %d", len(lastGroup.Players))
	}

	// Should have warning about small last group
	if len(warnings) == 0 {
		t.Error("Expected warning about uneven last group")
	}
}

func TestWAMAssignGroupsYouthWarning(t *testing.T) {
	wam := NewWAM(testLogger())

	// 22 players with group of 6 in last group should trigger youth warning
	players := createTestPlayers(22)
	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSequential,
		YouthAgeLimit:    12,
	}

	// Actually let's test with explicit 6-player group size
	// 30 players with group size 6 = 5 groups exactly
	_ = players // Unused in this test branch
	_ = config  // Unused in this test branch

	players6 := createTestPlayers(30)
	config6 := &WAMConfig{
		GroupSize:        6,
		AssignmentMethod: WAMAssignmentSequential,
		YouthAgeLimit:    12,
	}

	groups6, _ := wam.AssignGroups(players6, config6)

	// 30 / 6 = 5 groups exactly
	if len(groups6) != 5 {
		t.Errorf("Expected 5 groups for 30 players with size 6, got %d", len(groups6))
	}

	for _, g := range groups6 {
		if len(g.Players) != 6 {
			t.Errorf("Group %s should have 6 players, got %d", g.Name, len(g.Players))
		}
	}
}

func TestWAMGenerateGroupPairings(t *testing.T) {
	wam := NewWAM(testLogger())

	group := &WAMGroup{
		ID:          uuid.New(),
		Name:        "A",
		GroupNumber: 1,
		Players: []WAMGroupPlayer{
			{ParticipantID: uuid.New(), Name: "Player 1", SortOrder: 1},
			{ParticipantID: uuid.New(), Name: "Player 2", SortOrder: 2},
			{ParticipantID: uuid.New(), Name: "Player 3", SortOrder: 3},
			{ParticipantID: uuid.New(), Name: "Player 4", SortOrder: 4},
		},
		RoundsTotal: 3,
	}

	// Generate pairings for each round
	allPairings := make([][]Pairing, 3)
	for round := 1; round <= 3; round++ {
		pairings, err := wam.GenerateGroupPairings(group, round)
		if err != nil {
			t.Fatalf("Round %d: unexpected error: %v", round, err)
		}
		allPairings[round-1] = pairings

		// Should have 2 pairings per round (4 players / 2)
		if len(pairings) != 2 {
			t.Errorf("Round %d: expected 2 pairings, got %d", round, len(pairings))
		}
	}

	// Verify each player plays exactly 3 games (against each other player once)
	playerGames := make(map[int]int)
	opponents := make(map[int]map[int]bool)
	for i := 1; i <= 4; i++ {
		opponents[i] = make(map[int]bool)
	}

	for _, roundPairings := range allPairings {
		for _, p := range roundPairings {
			if !p.IsBye {
				playerGames[p.WhiteStartNo]++
				playerGames[p.BlackStartNo]++
				opponents[p.WhiteStartNo][p.BlackStartNo] = true
				opponents[p.BlackStartNo][p.WhiteStartNo] = true
			}
		}
	}

	// Each player should have exactly 3 games
	for player := 1; player <= 4; player++ {
		if playerGames[player] != 3 {
			t.Errorf("Player %d: expected 3 games, got %d", player, playerGames[player])
		}
		// Should have played against 3 different opponents
		if len(opponents[player]) != 3 {
			t.Errorf("Player %d: expected 3 opponents, got %d", player, len(opponents[player]))
		}
	}
}

func TestWAMGenerateGroupPairingsOddPlayers(t *testing.T) {
	wam := NewWAM(testLogger())

	group := &WAMGroup{
		ID:          uuid.New(),
		Name:        "A",
		GroupNumber: 1,
		Players: []WAMGroupPlayer{
			{ParticipantID: uuid.New(), Name: "Player 1", SortOrder: 1},
			{ParticipantID: uuid.New(), Name: "Player 2", SortOrder: 2},
			{ParticipantID: uuid.New(), Name: "Player 3", SortOrder: 3},
		},
		RoundsTotal: 3, // 3 players need 3 rounds with byes
	}

	byeCount := make(map[int]int)
	totalGames := make(map[int]int)

	for round := 1; round <= 3; round++ {
		pairings, err := wam.GenerateGroupPairings(group, round)
		if err != nil {
			t.Fatalf("Round %d: unexpected error: %v", round, err)
		}

		// With 3 players: 1 game + 1 bye per round
		gameCount := 0
		byeRound := 0
		for _, p := range pairings {
			if p.IsBye {
				byeRound++
				byeCount[p.WhiteStartNo]++
			} else {
				gameCount++
				totalGames[p.WhiteStartNo]++
				totalGames[p.BlackStartNo]++
			}
		}

		if gameCount != 1 {
			t.Errorf("Round %d: expected 1 game, got %d", round, gameCount)
		}
		if byeRound != 1 {
			t.Errorf("Round %d: expected 1 bye, got %d", round, byeRound)
		}
	}

	// Each player should have exactly 1 bye
	for player := 1; player <= 3; player++ {
		if byeCount[player] != 1 {
			t.Errorf("Player %d: expected 1 bye, got %d", player, byeCount[player])
		}
		// Each player should have 2 games
		if totalGames[player] != 2 {
			t.Errorf("Player %d: expected 2 games, got %d", player, totalGames[player])
		}
	}
}

func TestWAMGenerateGroupPairingsInvalidRound(t *testing.T) {
	wam := NewWAM(testLogger())

	group := &WAMGroup{
		ID:          uuid.New(),
		Name:        "A",
		Players:     make([]WAMGroupPlayer, 4),
		RoundsTotal: 3,
	}

	// Round 0 should fail
	_, err := wam.GenerateGroupPairings(group, 0)
	if err == nil {
		t.Error("Expected error for round 0")
	}

	// Round 4 should fail (only 3 rounds)
	_, err = wam.GenerateGroupPairings(group, 4)
	if err == nil {
		t.Error("Expected error for round exceeding total")
	}
}

func TestWAMGenerateGroupPairingsTooFewPlayers(t *testing.T) {
	wam := NewWAM(testLogger())

	group := &WAMGroup{
		ID:          uuid.New(),
		Name:        "A",
		Players:     make([]WAMGroupPlayer, 1), // Only 1 player
		RoundsTotal: 0,
	}

	_, err := wam.GenerateGroupPairings(group, 1)
	if err == nil {
		t.Error("Expected error for group with fewer than 2 players")
	}
}

func TestWAMCalculateGroupStandings(t *testing.T) {
	wam := NewWAM(testLogger())

	group := &WAMGroup{
		ID:   uuid.New(),
		Name: "A",
		Players: []WAMGroupPlayer{
			{ParticipantID: uuid.New(), Name: "Player 1", Rating: 1600, SortOrder: 1, Points: 2.5},
			{ParticipantID: uuid.New(), Name: "Player 2", Rating: 1550, SortOrder: 2, Points: 2.0},
			{ParticipantID: uuid.New(), Name: "Player 3", Rating: 1500, SortOrder: 3, Points: 1.5},
			{ParticipantID: uuid.New(), Name: "Player 4", Rating: 1450, SortOrder: 4, Points: 0.0},
		},
	}

	// Simulate game results
	games := []GameResult{
		{WhiteStartNo: 1, BlackStartNo: 2, WhiteScore: 1.0, BlackScore: 0.0},
		{WhiteStartNo: 3, BlackStartNo: 4, WhiteScore: 1.0, BlackScore: 0.0},
		{WhiteStartNo: 1, BlackStartNo: 3, WhiteScore: 0.5, BlackScore: 0.5},
		{WhiteStartNo: 2, BlackStartNo: 4, WhiteScore: 1.0, BlackScore: 0.0},
		{WhiteStartNo: 1, BlackStartNo: 4, WhiteScore: 1.0, BlackScore: 0.0},
		{WhiteStartNo: 2, BlackStartNo: 3, WhiteScore: 1.0, BlackScore: 0.0},
	}

	tiebreaks := []string{"direct_encounter", "sonneborn_berger"}
	standings := wam.CalculateGroupStandings(group, games, tiebreaks)

	if len(standings) != 4 {
		t.Fatalf("Expected 4 standings, got %d", len(standings))
	}

	// Player 1 should be first (2.5 points)
	if standings[0].Name != "Player 1" {
		t.Errorf("Expected Player 1 first, got %s", standings[0].Name)
	}
	if standings[0].Rank != 1 {
		t.Errorf("Expected rank 1, got %d", standings[0].Rank)
	}

	// Player 4 should be last (0 points)
	if standings[3].Name != "Player 4" {
		t.Errorf("Expected Player 4 last, got %s", standings[3].Name)
	}
}

func TestWAMDetectClubConflicts(t *testing.T) {
	wam := NewWAM(testLogger())

	clubA := uuid.New()
	clubB := uuid.New()
	clubNameA := "Chess Club A"
	clubNameB := "Chess Club B"

	groups := []WAMGroup{
		{
			ID:   uuid.New(),
			Name: "A",
			Players: []WAMGroupPlayer{
				{Name: "Player 1", ClubID: &clubA, ClubName: &clubNameA},
				{Name: "Player 2", ClubID: &clubA, ClubName: &clubNameA}, // Same club!
				{Name: "Player 3", ClubID: &clubB, ClubName: &clubNameB},
				{Name: "Player 4", ClubID: nil}, // No club
			},
		},
		{
			ID:   uuid.New(),
			Name: "B",
			Players: []WAMGroupPlayer{
				{Name: "Player 5", ClubID: &clubB, ClubName: &clubNameB},
				{Name: "Player 6", ClubID: &clubB, ClubName: &clubNameB}, // Same club!
				{Name: "Player 7", ClubID: &clubB, ClubName: &clubNameB}, // 3 from same club!
				{Name: "Player 8", ClubID: nil},
			},
		},
	}

	conflicts := wam.DetectClubConflicts(groups)

	if len(conflicts) != 2 {
		t.Fatalf("Expected 2 club conflicts, got %d", len(conflicts))
	}

	// Group A: 2 players from Club A
	foundGroupA := false
	for _, c := range conflicts {
		if c.GroupName == "A" && c.ClubID == clubA {
			foundGroupA = true
			if len(c.Players) != 2 {
				t.Errorf("Group A conflict: expected 2 players, got %d", len(c.Players))
			}
		}
	}
	if !foundGroupA {
		t.Error("Expected conflict in Group A for Club A")
	}

	// Group B: 3 players from Club B
	foundGroupB := false
	for _, c := range conflicts {
		if c.GroupName == "B" && c.ClubID == clubB {
			foundGroupB = true
			if len(c.Players) != 3 {
				t.Errorf("Group B conflict: expected 3 players, got %d", len(c.Players))
			}
		}
	}
	if !foundGroupB {
		t.Error("Expected conflict in Group B for Club B")
	}
}

func TestWAMCheckAgeWarnings(t *testing.T) {
	wam := NewWAM(testLogger())

	now := time.Now()
	age10 := now.AddDate(-10, 0, 0)
	age13 := now.AddDate(-13, 0, 0)
	age15 := now.AddDate(-15, 0, 0)

	groups := []WAMGroup{
		{
			ID:   uuid.New(),
			Name: "A",
			Players: []WAMGroupPlayer{
				{Name: "Young 1", BirthDate: &age10},
				{Name: "Young 2", BirthDate: &age10},
				{Name: "Teen 1", BirthDate: &age13}, // Over 12!
				{Name: "Teen 2", BirthDate: &age15}, // Over 12!
				{Name: "No DOB", BirthDate: nil},
				{Name: "Young 3", BirthDate: &age10},
			},
		},
		{
			ID:   uuid.New(),
			Name: "B",
			Players: []WAMGroupPlayer{
				{Name: "Player 1", BirthDate: &age10},
				{Name: "Player 2", BirthDate: &age10},
				{Name: "Player 3", BirthDate: &age10},
				{Name: "Player 4", BirthDate: &age10},
			},
		},
	}

	config := &WAMConfig{YouthAgeLimit: 12}
	warnings := wam.CheckAgeWarnings(groups, config, now)

	// Group A has 6 players, 2 over age 12 -> 2 warnings
	if len(warnings) != 2 {
		t.Errorf("Expected 2 age warnings, got %d", len(warnings))
	}

	// Group B has only 4 players, should not trigger warnings
	for _, w := range warnings {
		if w.GroupName == "B" {
			t.Error("Group B should not have age warnings (only 4 players)")
		}
	}
}

func TestWAMGeneratePlayoffBracket(t *testing.T) {
	wam := NewWAM(testLogger())

	// 4 groups, top 1 from each = 4 qualifiers -> semifinals + final
	groups := make([]WAMGroup, 4)
	for i := 0; i < 4; i++ {
		groups[i] = WAMGroup{
			ID:         uuid.New(),
			Name:       GroupNameFromNumber(i + 1),
			IsComplete: true,
			Players: []WAMGroupPlayer{
				{ParticipantID: uuid.New(), Name: "Winner " + GroupNameFromNumber(i+1), Points: 3.0},
				{ParticipantID: uuid.New(), Name: "Second " + GroupNameFromNumber(i+1), Points: 2.0},
				{ParticipantID: uuid.New(), Name: "Third " + GroupNameFromNumber(i+1), Points: 1.0},
				{ParticipantID: uuid.New(), Name: "Fourth " + GroupNameFromNumber(i+1), Points: 0.0},
			},
		}
	}

	config := &WAMConfig{
		TournamentID:    uuid.New(),
		EnablePlayoffs:  true,
		PlayoffsTopN:    1,
		ThirdPlaceMatch: true,
	}

	bracket, err := wam.GeneratePlayoffBracket(groups, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have: 2 semifinals + 1 final + 1 third place = 4 matches
	if len(bracket.Matches) != 4 {
		t.Errorf("Expected 4 matches, got %d", len(bracket.Matches))
	}

	// Count match types
	semis := 0
	finals := 0
	thirdPlace := 0
	for _, m := range bracket.Matches {
		switch m.RoundName {
		case "semifinal":
			semis++
		case "final":
			finals++
		case "third_place":
			thirdPlace++
		}
	}

	if semis != 2 {
		t.Errorf("Expected 2 semifinals, got %d", semis)
	}
	if finals != 1 {
		t.Errorf("Expected 1 final, got %d", finals)
	}
	if thirdPlace != 1 {
		t.Errorf("Expected 1 third place match, got %d", thirdPlace)
	}
}

func TestWAMGeneratePlayoffBracketNoThirdPlace(t *testing.T) {
	wam := NewWAM(testLogger())

	groups := make([]WAMGroup, 4)
	for i := 0; i < 4; i++ {
		groups[i] = WAMGroup{
			ID:         uuid.New(),
			Name:       GroupNameFromNumber(i + 1),
			IsComplete: true,
			Players: []WAMGroupPlayer{
				{ParticipantID: uuid.New(), Name: "Winner", Points: 3.0},
			},
		}
	}

	config := &WAMConfig{
		TournamentID:    uuid.New(),
		EnablePlayoffs:  true,
		PlayoffsTopN:    1,
		ThirdPlaceMatch: false, // No third place match
	}

	bracket, err := wam.GeneratePlayoffBracket(groups, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have: 2 semifinals + 1 final = 3 matches (no third place)
	if len(bracket.Matches) != 3 {
		t.Errorf("Expected 3 matches without third place, got %d", len(bracket.Matches))
	}
}

func TestWAMGeneratePlayoffBracketIncompleteGroups(t *testing.T) {
	wam := NewWAM(testLogger())

	groups := []WAMGroup{
		{ID: uuid.New(), Name: "A", IsComplete: true},
		{ID: uuid.New(), Name: "B", IsComplete: false}, // Not complete!
	}

	config := &WAMConfig{
		EnablePlayoffs: true,
		PlayoffsTopN:   1,
	}

	_, err := wam.GeneratePlayoffBracket(groups, config)
	if err == nil {
		t.Error("Expected error for incomplete groups")
	}
}

func TestWAMGeneratePlayoffBracketPlayoffsDisabled(t *testing.T) {
	wam := NewWAM(testLogger())

	groups := []WAMGroup{{ID: uuid.New(), Name: "A", IsComplete: true}}

	config := &WAMConfig{
		EnablePlayoffs: false,
	}

	_, err := wam.GeneratePlayoffBracket(groups, config)
	if err == nil {
		t.Error("Expected error when playoffs disabled")
	}
}

func TestWAMAssignGroupsManual(t *testing.T) {
	wam := NewWAM(testLogger())

	players := createTestPlayers(16)
	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentManual,
	}

	groups, _ := wam.AssignGroups(players, config)

	// Should create empty groups for manual assignment
	if len(groups) != 4 {
		t.Errorf("Expected 4 empty groups, got %d", len(groups))
	}

	// All groups should be empty
	for _, g := range groups {
		if len(g.Players) != 0 {
			t.Errorf("Group %s should be empty for manual assignment, got %d players", g.Name, len(g.Players))
		}
	}
}

func TestWAMAssignGroupsEmptyPlayers(t *testing.T) {
	wam := NewWAM(testLogger())

	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSequential,
	}

	groups, warnings := wam.AssignGroups([]Player{}, config)

	if groups != nil {
		t.Errorf("Expected nil groups for empty players, got %v", groups)
	}

	if len(warnings) == 0 {
		t.Error("Expected warning for no players")
	}
}

func TestWAMAssignGroupsDefaultGroupSize(t *testing.T) {
	wam := NewWAM(testLogger())

	players := createTestPlayers(8)
	config := &WAMConfig{
		GroupSize:        0, // Not set, should default to 4
		AssignmentMethod: WAMAssignmentSequential,
	}

	groups, _ := wam.AssignGroups(players, config)

	// 8 players with group size 4 = 2 groups
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups with default size 4, got %d", len(groups))
	}

	for _, g := range groups {
		if len(g.Players) != 4 {
			t.Errorf("Group %s should have 4 players, got %d", g.Name, len(g.Players))
		}
	}
}

// Helper functions

func createTestPlayers(n int) []Player {
	players := make([]Player, n)
	for i := 0; i < n; i++ {
		players[i] = Player{
			StartNo: i + 1,
			Name:    "Player " + string(rune('A'+i)),
			Rating:  1600 - (i * 10), // Descending ratings
		}
	}
	return players
}

func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Additional edge case tests

func TestWAMAssignGroupsSerpentineBalanced(t *testing.T) {
	wam := NewWAM(testLogger())

	// Test that serpentine creates balanced groups by rating
	players := createTestPlayers(12) // 3 groups of 4
	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSerpentine,
	}

	groups, _ := wam.AssignGroups(players, config)

	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(groups))
	}

	// Calculate average rating per group - should be similar
	avgRatings := make([]float64, 3)
	for i, g := range groups {
		sum := 0
		for _, p := range g.Players {
			sum += p.Rating
		}
		avgRatings[i] = float64(sum) / float64(len(g.Players))
	}

	// All average ratings should be within 50 points of each other
	for i := 0; i < len(avgRatings); i++ {
		for j := i + 1; j < len(avgRatings); j++ {
			diff := avgRatings[i] - avgRatings[j]
			if diff < 0 {
				diff = -diff
			}
			if diff > 50 {
				t.Errorf("Serpentine groups not balanced: Group %d avg=%.1f, Group %d avg=%.1f, diff=%.1f",
					i, avgRatings[i], j, avgRatings[j], diff)
			}
		}
	}
}

func TestWAMAssignGroupsSerpentineWithRemainder(t *testing.T) {
	wam := NewWAM(testLogger())

	// 10 players, groups of 4 -> 2 full groups + 1 with 2
	players := createTestPlayers(10)
	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSerpentine,
	}

	groups, _ := wam.AssignGroups(players, config)

	// Should have 3 groups
	if len(groups) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(groups))
	}

	// Verify all players are assigned exactly once
	assigned := make(map[string]bool)
	for _, g := range groups {
		for _, p := range g.Players {
			if assigned[p.Name] {
				t.Errorf("Player %s assigned to multiple groups", p.Name)
			}
			assigned[p.Name] = true
		}
	}

	if len(assigned) != 10 {
		t.Errorf("Expected 10 players assigned, got %d", len(assigned))
	}
}

func TestWAMGenerateGroupPairingsAllRounds(t *testing.T) {
	wam := NewWAM(testLogger())

	// Test various group sizes
	testCases := []struct {
		size   int
		rounds int
	}{
		{2, 1},
		{3, 3},
		{4, 3},
		{5, 5},
		{6, 5},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			players := make([]WAMGroupPlayer, tc.size)
			for i := 0; i < tc.size; i++ {
				players[i] = WAMGroupPlayer{
					ParticipantID: uuid.New(),
					Name:          "Player " + string(rune('A'+i)),
					SortOrder:     i + 1,
				}
			}

			group := &WAMGroup{
				ID:          uuid.New(),
				Name:        "A",
				Players:     players,
				RoundsTotal: tc.rounds,
			}

			// Track all matchups
			matchups := make(map[string]int)

			for round := 1; round <= tc.rounds; round++ {
				pairings, err := wam.GenerateGroupPairings(group, round)
				if err != nil {
					t.Fatalf("Size %d, Round %d: error: %v", tc.size, round, err)
				}

				for _, p := range pairings {
					if !p.IsBye {
						// Create normalized key
						w, b := p.WhiteStartNo, p.BlackStartNo
						if w > b {
							w, b = b, w
						}
						key := string(rune('A'+w-1)) + "-" + string(rune('A'+b-1))
						matchups[key]++
					}
				}
			}

			// Each pair should play exactly once
			expectedMatchups := (tc.size * (tc.size - 1)) / 2
			if len(matchups) != expectedMatchups {
				t.Errorf("Size %d: expected %d matchups, got %d", tc.size, expectedMatchups, len(matchups))
			}

			for key, count := range matchups {
				if count != 1 {
					t.Errorf("Size %d: matchup %s played %d times, expected 1", tc.size, key, count)
				}
			}
		})
	}
}

func TestWAMCalculateGroupStandingsTiebreaks(t *testing.T) {
	wam := NewWAM(testLogger())

	// Create a scenario with tied players
	p1ID := uuid.New()
	p2ID := uuid.New()
	p3ID := uuid.New()

	group := &WAMGroup{
		ID:   uuid.New(),
		Name: "A",
		Players: []WAMGroupPlayer{
			{ParticipantID: p1ID, Name: "Player 1", Rating: 1600, SortOrder: 1, Points: 1.5},
			{ParticipantID: p2ID, Name: "Player 2", Rating: 1550, SortOrder: 2, Points: 1.5}, // Tied!
			{ParticipantID: p3ID, Name: "Player 3", Rating: 1500, SortOrder: 3, Points: 0.0},
		},
	}

	// Player 1 beat Player 2 head-to-head
	games := []GameResult{
		{WhiteStartNo: 1, BlackStartNo: 2, WhiteScore: 1.0, BlackScore: 0.0}, // P1 beats P2
		{WhiteStartNo: 1, BlackStartNo: 3, WhiteScore: 0.5, BlackScore: 0.5}, // Draw
		{WhiteStartNo: 2, BlackStartNo: 3, WhiteScore: 1.0, BlackScore: 0.0}, // P2 beats P3
	}

	// With direct encounter tiebreak, P1 should be first
	tiebreaks := []string{"direct_encounter"}
	standings := wam.CalculateGroupStandings(group, games, tiebreaks)

	if standings[0].ParticipantID != p1ID {
		t.Errorf("Player 1 should be first (won direct encounter), got %s", standings[0].Name)
	}
	if standings[1].ParticipantID != p2ID {
		t.Errorf("Player 2 should be second, got %s", standings[1].Name)
	}
}

func TestWAMGroupsWithMinimumPlayers(t *testing.T) {
	wam := NewWAM(testLogger())

	// Test with exactly 3 players (minimum for a group)
	players := createTestPlayers(3)
	config := &WAMConfig{
		GroupSize:        3,
		AssignmentMethod: WAMAssignmentSequential,
	}

	groups, _ := wam.AssignGroups(players, config)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}

	if len(groups[0].Players) != 3 {
		t.Errorf("Expected 3 players in group, got %d", len(groups[0].Players))
	}

	// Rounds should be 3 for 3 players
	if groups[0].RoundsTotal != 3 {
		t.Errorf("Expected 3 rounds for 3 players, got %d", groups[0].RoundsTotal)
	}
}

func TestWAMAssignGroupsPreservesPlayerData(t *testing.T) {
	wam := NewWAM(testLogger())

	players := []Player{
		{StartNo: 1, Name: "Alpha", Rating: 2000},
		{StartNo: 2, Name: "Beta", Rating: 1900},
		{StartNo: 3, Name: "Gamma", Rating: 1800},
		{StartNo: 4, Name: "Delta", Rating: 1700},
	}

	config := &WAMConfig{
		GroupSize:        4,
		AssignmentMethod: WAMAssignmentSequential,
	}

	groups, _ := wam.AssignGroups(players, config)

	// Verify player data is preserved
	for _, g := range groups {
		for _, p := range g.Players {
			found := false
			for _, orig := range players {
				if orig.Name == p.Name && orig.Rating == p.Rating {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Player data not preserved: %s (rating %d)", p.Name, p.Rating)
			}
		}
	}
}

func TestWAMPlayoffBracketTwoGroups(t *testing.T) {
	wam := NewWAM(testLogger())

	// Only 2 groups -> direct final
	groups := make([]WAMGroup, 2)
	for i := 0; i < 2; i++ {
		groups[i] = WAMGroup{
			ID:         uuid.New(),
			Name:       GroupNameFromNumber(i + 1),
			IsComplete: true,
			Players: []WAMGroupPlayer{
				{ParticipantID: uuid.New(), Name: "Winner", Points: 2.0},
				{ParticipantID: uuid.New(), Name: "Second", Points: 1.0},
			},
		}
	}

	config := &WAMConfig{
		TournamentID:   uuid.New(),
		EnablePlayoffs: true,
		PlayoffsTopN:   1,
	}

	bracket, err := wam.GeneratePlayoffBracket(groups, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have just 1 final match
	if len(bracket.Matches) != 1 {
		t.Errorf("Expected 1 match for 2 groups, got %d", len(bracket.Matches))
	}

	if bracket.Matches[0].RoundName != "final" {
		t.Errorf("Expected final, got %s", bracket.Matches[0].RoundName)
	}
}

func TestWAMPlayoffBracketInsufficientQualifiers(t *testing.T) {
	wam := NewWAM(testLogger())

	// Only 1 group with 1 qualifier -> not enough for playoffs
	groups := []WAMGroup{
		{
			ID:         uuid.New(),
			Name:       "A",
			IsComplete: true,
			Players: []WAMGroupPlayer{
				{ParticipantID: uuid.New(), Name: "Winner", Points: 2.0},
			},
		},
	}

	config := &WAMConfig{
		TournamentID:   uuid.New(),
		EnablePlayoffs: true,
		PlayoffsTopN:   1,
	}

	_, err := wam.GeneratePlayoffBracket(groups, config)
	if err == nil {
		t.Error("Expected error for insufficient qualifiers")
	}
}

func TestWAMRoundRobinColorBalance(t *testing.T) {
	wam := NewWAM(testLogger())

	group := &WAMGroup{
		ID:   uuid.New(),
		Name: "A",
		Players: []WAMGroupPlayer{
			{ParticipantID: uuid.New(), Name: "P1", SortOrder: 1},
			{ParticipantID: uuid.New(), Name: "P2", SortOrder: 2},
			{ParticipantID: uuid.New(), Name: "P3", SortOrder: 3},
			{ParticipantID: uuid.New(), Name: "P4", SortOrder: 4},
		},
		RoundsTotal: 3,
	}

	whiteCount := make(map[int]int)
	blackCount := make(map[int]int)

	for round := 1; round <= 3; round++ {
		pairings, err := wam.GenerateGroupPairings(group, round)
		if err != nil {
			t.Fatalf("Round %d error: %v", round, err)
		}

		for _, p := range pairings {
			if !p.IsBye {
				whiteCount[p.WhiteStartNo]++
				blackCount[p.BlackStartNo]++
			}
		}
	}

	// Each player should have roughly balanced colors (in 3 games: 1-2 white, 1-2 black)
	for player := 1; player <= 4; player++ {
		w := whiteCount[player]
		b := blackCount[player]
		total := w + b
		if total != 3 {
			t.Errorf("Player %d: expected 3 games, got %d", player, total)
		}
		// Accept 1-2 or 2-1 color distribution
		if w < 1 || w > 2 || b < 1 || b > 2 {
			t.Errorf("Player %d: poor color balance (W=%d, B=%d)", player, w, b)
		}
	}
}

func TestWAMAssignGroupsExactFit(t *testing.T) {
	wam := NewWAM(testLogger())

	// Test cases where players fit exactly into groups
	testCases := []struct {
		players   int
		groupSize int
		groups    int
	}{
		{8, 4, 2},
		{12, 4, 3},
		{16, 4, 4},
		{18, 6, 3},
		{9, 3, 3},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			players := createTestPlayers(tc.players)
			config := &WAMConfig{
				GroupSize:        tc.groupSize,
				AssignmentMethod: WAMAssignmentSequential,
			}

			groups, warnings := wam.AssignGroups(players, config)

			if len(groups) != tc.groups {
				t.Errorf("%d players, size %d: expected %d groups, got %d",
					tc.players, tc.groupSize, tc.groups, len(groups))
			}

			// No warnings for exact fit
			for _, w := range warnings {
				if containsSubstring(w, "instead of") {
					t.Errorf("Unexpected warning for exact fit: %s", w)
				}
			}

			// All groups should have exact size
			for _, g := range groups {
				if len(g.Players) != tc.groupSize {
					t.Errorf("Group %s: expected %d players, got %d",
						g.Name, tc.groupSize, len(g.Players))
				}
			}
		})
	}
}
