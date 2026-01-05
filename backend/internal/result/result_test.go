package result

import (
	"testing"
)

func TestParseResult(t *testing.T) {
	tests := []struct {
		name           string
		result         GameResult
		wantWhiteScore float64
		wantBlackScore float64
	}{
		{
			name:           "white wins",
			result:         ResultWhiteWins,
			wantWhiteScore: 1.0,
			wantBlackScore: 0.0,
		},
		{
			name:           "black wins",
			result:         ResultBlackWins,
			wantWhiteScore: 0.0,
			wantBlackScore: 1.0,
		},
		{
			name:           "draw",
			result:         ResultDraw,
			wantWhiteScore: 0.5,
			wantBlackScore: 0.5,
		},
		{
			name:           "white wins by forfeit",
			result:         ResultWhiteForfeit,
			wantWhiteScore: 1.0,
			wantBlackScore: 0.0,
		},
		{
			name:           "black wins by forfeit",
			result:         ResultBlackForfeit,
			wantWhiteScore: 0.0,
			wantBlackScore: 1.0,
		},
		{
			name:           "double forfeit",
			result:         ResultDoubleForfeit,
			wantWhiteScore: 0.0,
			wantBlackScore: 0.0,
		},
		{
			name:           "bye",
			result:         ResultBye,
			wantWhiteScore: 1.0,
			wantBlackScore: 0.0,
		},
		{
			name:           "empty/ongoing",
			result:         ResultOngoing,
			wantWhiteScore: 0.0,
			wantBlackScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whiteScore, blackScore := ParseResult(tt.result)
			if whiteScore != tt.wantWhiteScore {
				t.Errorf("ParseResult(%q) whiteScore = %v, want %v", tt.result, whiteScore, tt.wantWhiteScore)
			}
			if blackScore != tt.wantBlackScore {
				t.Errorf("ParseResult(%q) blackScore = %v, want %v", tt.result, blackScore, tt.wantBlackScore)
			}
		})
	}
}

func TestIsValidResult(t *testing.T) {
	tests := []struct {
		result string
		valid  bool
	}{
		{"1-0", true},
		{"0-1", true},
		{"1/2-1/2", true},
		{"1-0F", true},
		{"0-1F", true},
		{"0-0F", true},
		{"BYE", true},
		{"invalid", false},
		{"1/2", false},
		{"", false},
		{"2-0", false},
	}

	for _, tt := range tests {
		t.Run(tt.result, func(t *testing.T) {
			if got := IsValidResult(tt.result); got != tt.valid {
				t.Errorf("IsValidResult(%q) = %v, want %v", tt.result, got, tt.valid)
			}
		})
	}
}

func TestValidResults(t *testing.T) {
	validResults := ValidResults()

	// Should have exactly 7 valid results
	expectedCount := 7
	if len(validResults) != expectedCount {
		t.Errorf("ValidResults() returned %d results, want %d", len(validResults), expectedCount)
	}

	// All should be valid when checked
	for _, r := range validResults {
		if !IsValidResult(string(r)) {
			t.Errorf("ValidResults() contains %q which IsValidResult says is invalid", r)
		}
	}
}

func TestResultDescription(t *testing.T) {
	tests := []struct {
		result      GameResult
		description string
	}{
		{ResultWhiteWins, "White wins"},
		{ResultBlackWins, "Black wins"},
		{ResultDraw, "Draw"},
		{ResultWhiteForfeit, "White wins by forfeit"},
		{ResultBlackForfeit, "Black wins by forfeit"},
		{ResultDoubleForfeit, "Double forfeit"},
		{ResultBye, "Bye"},
		{GameResult("invalid"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.result), func(t *testing.T) {
			got := ResultDescription(tt.result)
			if got != tt.description {
				t.Errorf("ResultDescription(%q) = %q, want %q", tt.result, got, tt.description)
			}
		})
	}
}

func TestResultEntry_ScoreCalculation(t *testing.T) {
	// Test that result entries produce correct scores
	tests := []struct {
		result     GameResult
		whiteScore float64
		blackScore float64
	}{
		{ResultWhiteWins, 1.0, 0.0},
		{ResultBlackWins, 0.0, 1.0},
		{ResultDraw, 0.5, 0.5},
	}

	for _, tt := range tests {
		t.Run(string(tt.result), func(t *testing.T) {
			ws, bs := ParseResult(tt.result)
			if ws != tt.whiteScore || bs != tt.blackScore {
				t.Errorf("Scores for %s: got (%.1f, %.1f), want (%.1f, %.1f)",
					tt.result, ws, bs, tt.whiteScore, tt.blackScore)
			}
		})
	}
}

func TestScoreSumTo1(t *testing.T) {
	// For most results, scores should sum to 1.0 (except double forfeit)
	results := []GameResult{
		ResultWhiteWins,
		ResultBlackWins,
		ResultDraw,
		ResultWhiteForfeit,
		ResultBlackForfeit,
		ResultBye,
	}

	for _, r := range results {
		ws, bs := ParseResult(r)
		sum := ws + bs
		if sum != 1.0 {
			t.Errorf("Scores for %s sum to %.1f, expected 1.0", r, sum)
		}
	}

	// Double forfeit should sum to 0
	ws, bs := ParseResult(ResultDoubleForfeit)
	if ws+bs != 0.0 {
		t.Errorf("Scores for double forfeit sum to %.1f, expected 0.0", ws+bs)
	}
}
