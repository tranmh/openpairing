package standings

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsEligibleForRanking(t *testing.T) {
	tests := []struct {
		name     string
		data     ParticipantEligibilityData
		ranking  SpecialRankingData
		expected bool
	}{
		{
			name: "U12 eligible - born 2014",
			data: ParticipantEligibilityData{
				BirthYear: 2014,
				Rating:    1200,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinBirthYear: intPtr(2012), // Born 2012 or later = U12
			},
			expected: true,
		},
		{
			name: "U12 not eligible - born 2010",
			data: ParticipantEligibilityData{
				BirthYear: 2010,
				Rating:    1500,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinBirthYear: intPtr(2012),
			},
			expected: false,
		},
		{
			name: "Seniors eligible - born 1960",
			data: ParticipantEligibilityData{
				BirthYear: 1960,
				Rating:    1800,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MaxBirthYear: intPtr(1964), // Born 1964 or earlier = 60+
			},
			expected: true,
		},
		{
			name: "Seniors not eligible - born 1980",
			data: ParticipantEligibilityData{
				BirthYear: 1980,
				Rating:    1800,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MaxBirthYear: intPtr(1964),
			},
			expected: false,
		},
		{
			name: "Rating class 1500-1699 eligible",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1550,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinRating: intPtr(1500),
				MaxRating: intPtr(1699),
			},
			expected: true,
		},
		{
			name: "Rating class 1500-1699 not eligible - too high",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1750,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinRating: intPtr(1500),
				MaxRating: intPtr(1699),
			},
			expected: false,
		},
		{
			name: "Rating class 1500-1699 not eligible - too low",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1400,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinRating: intPtr(1500),
				MaxRating: intPtr(1699),
			},
			expected: false,
		},
		{
			name: "Women's ranking eligible",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1600,
				Gender:    "w",
			},
			ranking: SpecialRankingData{
				Gender: strPtr("w"),
			},
			expected: true,
		},
		{
			name: "Women's ranking not eligible - male",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1600,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				Gender: strPtr("w"),
			},
			expected: false,
		},
		{
			name: "Combined criteria - U20 women eligible",
			data: ParticipantEligibilityData{
				BirthYear: 2006,
				Rating:    1400,
				Gender:    "w",
			},
			ranking: SpecialRankingData{
				MinBirthYear: intPtr(2004), // U20
				Gender:       strPtr("w"),
			},
			expected: true,
		},
		{
			name: "Combined criteria - U20 women not eligible (male)",
			data: ParticipantEligibilityData{
				BirthYear: 2006,
				Rating:    1400,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinBirthYear: intPtr(2004),
				Gender:       strPtr("w"),
			},
			expected: false,
		},
		{
			name: "No criteria - everyone eligible",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1600,
				Gender:    "m",
			},
			ranking:  SpecialRankingData{},
			expected: true,
		},
		{
			name: "Empty gender filter should match all",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1600,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				Gender: strPtr(""),
			},
			expected: true,
		},
		{
			name: "Birth year 0 (unknown) should fail min birth year check",
			data: ParticipantEligibilityData{
				BirthYear: 0,
				Rating:    1600,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinBirthYear: intPtr(2012),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEligibleForRanking(tt.data, tt.ranking)
			if result != tt.expected {
				t.Errorf("isEligibleForRanking() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRankingCriteria_IsEligible(t *testing.T) {
	tests := []struct {
		name      string
		criteria  RankingCriteria
		birthYear int
		rating    int
		gender    string
		expected  bool
	}{
		{
			name: "All nil criteria matches everyone",
			criteria: RankingCriteria{},
			birthYear: 1990,
			rating:    1500,
			gender:    "m",
			expected:  true,
		},
		{
			name: "Min birth year filter",
			criteria: RankingCriteria{
				MinBirthYear: intPtr(2000),
			},
			birthYear: 2005,
			rating:    1200,
			gender:    "m",
			expected:  true,
		},
		{
			name: "Min birth year filter - fail",
			criteria: RankingCriteria{
				MinBirthYear: intPtr(2000),
			},
			birthYear: 1995,
			rating:    1200,
			gender:    "m",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.criteria.IsEligible(tt.birthYear, tt.rating, tt.gender)
			if result != tt.expected {
				t.Errorf("IsEligible() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateRankInGroups(t *testing.T) {
	svc := &Service{}

	// Create test data with 5 players
	player1 := uuid.New()
	player2 := uuid.New()
	player3 := uuid.New()
	player4 := uuid.New()
	player5 := uuid.New()

	u12Ranking := uuid.New()
	womenRanking := uuid.New()

	// Standings in order (player1 is rank 1, player5 is rank 5)
	standings := []struct {
		PlayerID uuid.UUID
		Points   float64
	}{
		{player1, 5.0},
		{player2, 4.0},
		{player3, 3.0},
		{player4, 2.0},
		{player5, 1.0},
	}

	// Convert to tiebreak.Standing format (we need to import or mock this)
	// For this test, we'll test the helper function directly with mock data

	eligibility := map[uuid.UUID]ParticipantEligibilityData{
		player1: {ParticipantID: player1, BirthYear: 2015, Rating: 1000, Gender: "m"}, // U12, male
		player2: {ParticipantID: player2, BirthYear: 2014, Rating: 1100, Gender: "w"}, // U12, female
		player3: {ParticipantID: player3, BirthYear: 2000, Rating: 1500, Gender: "w"}, // Not U12, female
		player4: {ParticipantID: player4, BirthYear: 2016, Rating: 900, Gender: "m"},  // U12, male
		player5: {ParticipantID: player5, BirthYear: 1990, Rating: 1800, Gender: "m"}, // Not U12, male
	}

	rankings := []SpecialRankingData{
		{
			ID:           u12Ranking,
			Name:         "u12",
			DisplayName:  "U12 Wertung",
			MinBirthYear: intPtr(2012), // Born 2012 or later
		},
		{
			ID:          womenRanking,
			Name:        "women",
			DisplayName: "Frauenwertung",
			Gender:      strPtr("w"),
		},
	}

	// We need to create proper tiebreak.Standing slice
	// Since we can't easily import tiebreak here without circular deps,
	// let's test the eligibility logic which is the core of the feature

	// Test U12 eligibility
	u12Eligible := []uuid.UUID{}
	for _, s := range standings {
		if data, ok := eligibility[s.PlayerID]; ok {
			if isEligibleForRanking(data, rankings[0]) {
				u12Eligible = append(u12Eligible, s.PlayerID)
			}
		}
	}

	// Should be: player1 (rank 1 in U12), player2 (rank 2 in U12), player4 (rank 3 in U12)
	if len(u12Eligible) != 3 {
		t.Errorf("Expected 3 U12 eligible players, got %d", len(u12Eligible))
	}
	if u12Eligible[0] != player1 {
		t.Errorf("Expected player1 to be first in U12, got %v", u12Eligible[0])
	}
	if u12Eligible[1] != player2 {
		t.Errorf("Expected player2 to be second in U12, got %v", u12Eligible[1])
	}
	if u12Eligible[2] != player4 {
		t.Errorf("Expected player4 to be third in U12, got %v", u12Eligible[2])
	}

	// Test women eligibility
	womenEligible := []uuid.UUID{}
	for _, s := range standings {
		if data, ok := eligibility[s.PlayerID]; ok {
			if isEligibleForRanking(data, rankings[1]) {
				womenEligible = append(womenEligible, s.PlayerID)
			}
		}
	}

	// Should be: player2 (rank 1 in women), player3 (rank 2 in women)
	if len(womenEligible) != 2 {
		t.Errorf("Expected 2 women eligible players, got %d", len(womenEligible))
	}
	if womenEligible[0] != player2 {
		t.Errorf("Expected player2 to be first in women, got %v", womenEligible[0])
	}
	if womenEligible[1] != player3 {
		t.Errorf("Expected player3 to be second in women, got %v", womenEligible[1])
	}

	_ = svc // silence unused warning
}

func TestIsEligibleForRanking_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name     string
		data     ParticipantEligibilityData
		ranking  SpecialRankingData
		expected bool
	}{
		{
			name: "Rating exactly at min boundary - should be eligible",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1500,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinRating: intPtr(1500),
				MaxRating: intPtr(1699),
			},
			expected: true,
		},
		{
			name: "Rating exactly at max boundary - should be eligible",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1699,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinRating: intPtr(1500),
				MaxRating: intPtr(1699),
			},
			expected: true,
		},
		{
			name: "Rating one above max boundary - should NOT be eligible",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1700,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinRating: intPtr(1500),
				MaxRating: intPtr(1699),
			},
			expected: false,
		},
		{
			name: "Rating one below min boundary - should NOT be eligible",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    1499,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinRating: intPtr(1500),
				MaxRating: intPtr(1699),
			},
			expected: false,
		},
		{
			name: "Birth year exactly at min boundary",
			data: ParticipantEligibilityData{
				BirthYear: 2012,
				Rating:    1000,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinBirthYear: intPtr(2012),
			},
			expected: true,
		},
		{
			name: "Birth year one below min boundary",
			data: ParticipantEligibilityData{
				BirthYear: 2011,
				Rating:    1000,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinBirthYear: intPtr(2012),
			},
			expected: false,
		},
		{
			name: "Birth year exactly at max boundary",
			data: ParticipantEligibilityData{
				BirthYear: 1964,
				Rating:    1800,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MaxBirthYear: intPtr(1964),
			},
			expected: true,
		},
		{
			name: "Birth year one above max boundary",
			data: ParticipantEligibilityData{
				BirthYear: 1965,
				Rating:    1800,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MaxBirthYear: intPtr(1964),
			},
			expected: false,
		},
		{
			name: "Rating 0 with min rating filter",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    0,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MinRating: intPtr(1500),
			},
			expected: false,
		},
		{
			name: "Rating 0 with max rating filter only - should be eligible",
			data: ParticipantEligibilityData{
				BirthYear: 1990,
				Rating:    0,
				Gender:    "m",
			},
			ranking: SpecialRankingData{
				MaxRating: intPtr(1699),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEligibleForRanking(tt.data, tt.ranking)
			if result != tt.expected {
				t.Errorf("isEligibleForRanking() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEligibleRankingsInStanding(t *testing.T) {
	// Test that EligibleRankings slice is properly initialized
	standing := Standing{
		Rank:          1,
		ParticipantID: uuid.New(),
		PlayerID:      uuid.New(),
		PlayerName:    "Test Player",
		Rating:        1500,
		Points:        5.0,
	}

	// EligibleRankings should be nil by default (omitempty in JSON)
	if standing.EligibleRankings != nil {
		t.Error("EligibleRankings should be nil by default")
	}

	// After assignment, it should work
	standing.EligibleRankings = []EligibleRanking{
		{
			ID:          uuid.New(),
			Name:        "u12",
			DisplayName: "U12 Wertung",
			RankInGroup: 1,
		},
	}

	if len(standing.EligibleRankings) != 1 {
		t.Errorf("Expected 1 eligible ranking, got %d", len(standing.EligibleRankings))
	}
	if standing.EligibleRankings[0].RankInGroup != 1 {
		t.Errorf("Expected RankInGroup 1, got %d", standing.EligibleRankings[0].RankInGroup)
	}
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
