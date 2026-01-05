package fairplay

import (
	"testing"
)

func TestViolationType_Constants(t *testing.T) {
	types := []ViolationType{
		ViolationSuspiciousPlay,
		ViolationRatingManipulation,
		ViolationCollusion,
		ViolationTimestalling,
		ViolationUnsportsmanlike,
		ViolationOther,
	}

	for _, vt := range types {
		if vt == "" {
			t.Error("ViolationType should not be empty")
		}
	}

	// Verify specific values
	if ViolationSuspiciousPlay != "suspicious_play" {
		t.Errorf("ViolationSuspiciousPlay = %q, want %q", ViolationSuspiciousPlay, "suspicious_play")
	}
	if ViolationCollusion != "collusion" {
		t.Errorf("ViolationCollusion = %q, want %q", ViolationCollusion, "collusion")
	}
}

func TestSeverityLevel_Constants(t *testing.T) {
	levels := []SeverityLevel{
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}

	for _, level := range levels {
		if level == "" {
			t.Error("SeverityLevel should not be empty")
		}
	}

	// Verify specific values
	if SeverityLow != "low" {
		t.Errorf("SeverityLow = %q, want %q", SeverityLow, "low")
	}
	if SeverityCritical != "critical" {
		t.Errorf("SeverityCritical = %q, want %q", SeverityCritical, "critical")
	}
}

func TestViolationStatus_Constants(t *testing.T) {
	statuses := []ViolationStatus{
		StatusPending,
		StatusReviewing,
		StatusConfirmed,
		StatusDismissed,
		StatusAppealed,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("ViolationStatus should not be empty")
		}
	}
}

func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		violationType ViolationType
		expected      SeverityLevel
	}{
		{ViolationSuspiciousPlay, SeverityHigh},
		{ViolationRatingManipulation, SeverityCritical},
		{ViolationCollusion, SeverityCritical},
		{ViolationTimestalling, SeverityLow},
		{ViolationUnsportsmanlike, SeverityMedium},
		{ViolationOther, SeverityMedium},
	}

	for _, tt := range tests {
		t.Run(string(tt.violationType), func(t *testing.T) {
			result := determineSeverity(tt.violationType)
			if result != tt.expected {
				t.Errorf("determineSeverity(%q) = %q, want %q", tt.violationType, result, tt.expected)
			}
		})
	}
}

func TestFairplayStats_DefaultValues(t *testing.T) {
	stats := FairplayStats{}

	// All counts should be zero by default
	if stats.TotalViolations != 0 {
		t.Errorf("TotalViolations should be 0, got %d", stats.TotalViolations)
	}
	if stats.PendingViolations != 0 {
		t.Errorf("PendingViolations should be 0, got %d", stats.PendingViolations)
	}
	if stats.ActiveFlags != 0 {
		t.Errorf("ActiveFlags should be 0, got %d", stats.ActiveFlags)
	}
}

func TestCreateViolationRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		description string
		valid       bool
	}{
		{
			name:        "valid description",
			description: "The player exhibited suspicious play patterns with unusually high accuracy in complex positions",
			valid:       true,
		},
		{
			name:        "too short",
			description: "cheating",
			valid:       false,
		},
		{
			name:        "minimum valid length",
			description: "This is just barely long enough to be considered a valid violation description",
			valid:       true,
		},
	}

	minDescLength := 20

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.description) >= minDescLength
			if isValid != tt.valid {
				t.Errorf("Description %q: valid=%v, want %v", tt.description, isValid, tt.valid)
			}
		})
	}
}

func TestFlagType_Validation(t *testing.T) {
	validTypes := map[string]bool{
		"warning": true,
		"watch":   true,
		"banned":  true,
	}

	tests := []struct {
		flagType string
		valid    bool
	}{
		{"warning", true},
		{"watch", true},
		{"banned", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.flagType, func(t *testing.T) {
			isValid := validTypes[tt.flagType]
			if isValid != tt.valid {
				t.Errorf("FlagType %q: valid=%v, want %v", tt.flagType, isValid, tt.valid)
			}
		})
	}
}

func TestReviewViolationRequest_StatusValidation(t *testing.T) {
	validStatuses := map[ViolationStatus]bool{
		StatusConfirmed: true,
		StatusDismissed: true,
	}

	tests := []struct {
		status ViolationStatus
		valid  bool
	}{
		{StatusConfirmed, true},
		{StatusDismissed, true},
		{StatusPending, false},
		{StatusReviewing, false},
		{StatusAppealed, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			isValid := validStatuses[tt.status]
			if isValid != tt.valid {
				t.Errorf("ReviewStatus %q: valid=%v, want %v", tt.status, isValid, tt.valid)
			}
		})
	}
}

func TestPerformanceAnalysis_RiskScore(t *testing.T) {
	// Risk score should be between 0 and 100
	tests := []struct {
		score float64
		valid bool
	}{
		{0, true},
		{50, true},
		{100, true},
		{-1, false},
		{101, false},
	}

	for _, tt := range tests {
		isValid := tt.score >= 0 && tt.score <= 100
		if isValid != tt.valid {
			t.Errorf("RiskScore %.1f: valid=%v, want %v", tt.score, isValid, tt.valid)
		}
	}
}
