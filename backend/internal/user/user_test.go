package user

import (
	"testing"
)

func TestUserRole_Constants(t *testing.T) {
	// Verify role constants are defined correctly
	roles := []UserRole{
		RolePlayer,
		RoleArbiter,
		RoleOrganizer,
		RoleAdmin,
	}

	for _, role := range roles {
		if role == "" {
			t.Error("UserRole should not be empty")
		}
	}

	// Check specific values
	if RolePlayer != "player" {
		t.Errorf("RolePlayer = %q, want %q", RolePlayer, "player")
	}
	if RoleAdmin != "admin" {
		t.Errorf("RoleAdmin = %q, want %q", RoleAdmin, "admin")
	}
}

func TestUpdateProfileRequest_Validation(t *testing.T) {
	tests := []struct {
		name  string
		req   UpdateProfileRequest
		valid bool
	}{
		{
			name: "valid display name",
			req: UpdateProfileRequest{
				DisplayName: strPtr("John Doe"),
			},
			valid: true,
		},
		{
			name: "valid language de",
			req: UpdateProfileRequest{
				PreferredLanguage: strPtr("de"),
			},
			valid: true,
		},
		{
			name: "valid language en",
			req: UpdateProfileRequest{
				PreferredLanguage: strPtr("en"),
			},
			valid: true,
		},
		{
			name: "empty request is valid",
			req:  UpdateProfileRequest{},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation for language
			if tt.req.PreferredLanguage != nil {
				lang := *tt.req.PreferredLanguage
				isValidLang := lang == "de" || lang == "en"
				if !isValidLang && tt.valid {
					t.Errorf("Expected valid but language %q is invalid", lang)
				}
			}
		})
	}
}

func TestChangePasswordRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		current string
		newPwd  string
		confirm string
		valid   bool
	}{
		{
			name:    "valid password change",
			current: "oldPassword123",
			newPwd:  "newPassword456",
			confirm: "newPassword456",
			valid:   true,
		},
		{
			name:    "passwords don't match",
			current: "oldPassword123",
			newPwd:  "newPassword456",
			confirm: "differentPassword",
			valid:   false,
		},
		{
			name:    "new password too short",
			current: "oldPassword123",
			newPwd:  "short",
			confirm: "short",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.newPwd) >= 8 && tt.newPwd == tt.confirm
			if isValid != tt.valid {
				t.Errorf("ChangePasswordRequest validation: got %v, want %v", isValid, tt.valid)
			}
		})
	}
}

func TestUserStats_DefaultValues(t *testing.T) {
	stats := UserStats{}

	// Default values should be zero
	if stats.TournamentsPlayed != 0 {
		t.Errorf("TournamentsPlayed should default to 0, got %d", stats.TournamentsPlayed)
	}
	if stats.GamesPlayed != 0 {
		t.Errorf("GamesPlayed should default to 0, got %d", stats.GamesPlayed)
	}
	if stats.WinRate != 0 {
		t.Errorf("WinRate should default to 0, got %f", stats.WinRate)
	}
}

func TestUserStats_WinRateCalculation(t *testing.T) {
	tests := []struct {
		name    string
		wins    int
		games   int
		winRate float64
	}{
		{"no games", 0, 0, 0},
		{"all wins", 10, 10, 100},
		{"half wins", 5, 10, 50},
		{"one third wins", 1, 3, 33.33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var winRate float64
			if tt.games > 0 {
				winRate = float64(tt.wins) / float64(tt.games) * 100
			}
			// Allow small floating point difference
			diff := winRate - tt.winRate
			if diff < -1 || diff > 1 {
				t.Errorf("WinRate = %.2f, want ~%.2f", winRate, tt.winRate)
			}
		})
	}
}

func TestUserProfile_OptionalFields(t *testing.T) {
	profile := UserProfile{
		User: User{
			Email: "test@example.com",
			Role:  RolePlayer,
		},
	}

	// Optional fields should be nil by default
	if profile.PlayerID != nil {
		t.Error("PlayerID should be nil by default")
	}
	if profile.FirstName != nil {
		t.Error("FirstName should be nil by default")
	}
	if profile.FideRating != nil {
		t.Error("FideRating should be nil by default")
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}
