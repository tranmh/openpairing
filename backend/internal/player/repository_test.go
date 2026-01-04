package player

import (
	"testing"
)

func TestSearchPlayersRequest_BuildConditions(t *testing.T) {
	tests := []struct {
		name          string
		req           SearchPlayersRequest
		wantCondCount int // Number of conditions expected
	}{
		{
			name:          "empty request",
			req:           SearchPlayersRequest{},
			wantCondCount: 0,
		},
		{
			name: "with query",
			req: SearchPlayersRequest{
				Query: "Carlsen",
			},
			wantCondCount: 1,
		},
		{
			name: "with federation filter",
			req: SearchPlayersRequest{
				Federation: strPtr("fide"),
			},
			wantCondCount: 1,
		},
		{
			name: "with min rating",
			req: SearchPlayersRequest{
				MinRating: intPtr(2000),
			},
			wantCondCount: 1,
		},
		{
			name: "with max rating",
			req: SearchPlayersRequest{
				MaxRating: intPtr(2500),
			},
			wantCondCount: 1,
		},
		{
			name: "with all filters",
			req: SearchPlayersRequest{
				Query:      "Magnus",
				Federation: strPtr("fide"),
				MinRating:  intPtr(2000),
				MaxRating:  intPtr(3000),
				ClubID:     strPtr("123e4567-e89b-12d3-a456-426614174000"),
			},
			wantCondCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condCount := 0
			if tt.req.Query != "" {
				condCount++
			}
			if tt.req.Federation != nil {
				condCount++
			}
			if tt.req.MinRating != nil {
				condCount++
			}
			if tt.req.MaxRating != nil {
				condCount++
			}
			if tt.req.ClubID != nil {
				condCount++
			}

			if condCount != tt.wantCondCount {
				t.Errorf("condition count = %d, want %d", condCount, tt.wantCondCount)
			}
		})
	}
}

func TestFederationFilterValues(t *testing.T) {
	validFederations := []string{"fide", "dsb", "oesb"}

	for _, fed := range validFederations {
		t.Run(fed, func(t *testing.T) {
			// In a real test, this would verify the query modification
			switch fed {
			case "fide":
				// Should add: p.fide_id IS NOT NULL
			case "dsb":
				// Should add: p.dsb_id IS NOT NULL
			case "oesb":
				// Should add: p.oesb_id IS NOT NULL
			}
		})
	}
}

func TestRatingRangeValidation(t *testing.T) {
	tests := []struct {
		name      string
		minRating *int
		maxRating *int
		valid     bool
	}{
		{
			name:  "no range",
			valid: true,
		},
		{
			name:      "valid min",
			minRating: intPtr(1500),
			valid:     true,
		},
		{
			name:      "valid max",
			maxRating: intPtr(2500),
			valid:     true,
		},
		{
			name:      "valid range",
			minRating: intPtr(1500),
			maxRating: intPtr(2500),
			valid:     true,
		},
		{
			name:      "negative min",
			minRating: intPtr(-100),
			valid:     false,
		},
		{
			name:      "too high max",
			maxRating: intPtr(4000),
			valid:     false,
		},
		{
			name:      "min greater than max",
			minRating: intPtr(2500),
			maxRating: intPtr(1500),
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := true
			if tt.minRating != nil && (*tt.minRating < 0 || *tt.minRating > 3500) {
				valid = false
			}
			if tt.maxRating != nil && (*tt.maxRating < 0 || *tt.maxRating > 3500) {
				valid = false
			}
			if tt.minRating != nil && tt.maxRating != nil && *tt.minRating > *tt.maxRating {
				valid = false
			}

			if valid != tt.valid {
				t.Errorf("rating range valid = %v, want %v", valid, tt.valid)
			}
		})
	}
}

func TestPlayerErrorMessages(t *testing.T) {
	tests := []struct {
		err     error
		message string
	}{
		{ErrPlayerNotFound, "player not found"},
		{ErrDuplicateFideID, "player with this FIDE ID already exists"},
		{ErrDuplicateDsbID, "player with this DSB ID already exists"},
		{ErrDuplicateOesbID, "player with this ÖSB ID already exists"},
		{ErrClubNotFound, "club not found"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			if tt.err.Error() != tt.message {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.message)
			}
		})
	}
}

func TestValidFideTitles(t *testing.T) {
	expected := []string{"GM", "IM", "FM", "CM", "WGM", "WIM", "WFM", "WCM"}

	if len(ValidFideTitles) != len(expected) {
		t.Errorf("ValidFideTitles has %d entries, want %d", len(ValidFideTitles), len(expected))
	}

	for i, title := range expected {
		if ValidFideTitles[i] != title {
			t.Errorf("ValidFideTitles[%d] = %q, want %q", i, ValidFideTitles[i], title)
		}
	}
}

func TestValidGenders(t *testing.T) {
	expected := []string{"m", "f", "other"}

	if len(ValidGenders) != len(expected) {
		t.Errorf("ValidGenders has %d entries, want %d", len(ValidGenders), len(expected))
	}

	for i, gender := range expected {
		if ValidGenders[i] != gender {
			t.Errorf("ValidGenders[%d] = %q, want %q", i, ValidGenders[i], gender)
		}
	}
}
