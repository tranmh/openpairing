package tournament

import (
	"testing"
)

func TestDefaultRegistrationSettings(t *testing.T) {
	settings := DefaultRegistrationSettings()

	// Pre-registration defaults
	if !settings.PreRegistration.Enabled {
		t.Error("PreRegistration.Enabled should be true by default")
	}
	if settings.PreRegistration.RequiresApproval {
		t.Error("PreRegistration.RequiresApproval should be false by default")
	}
	if !settings.PreRegistration.AllowGuests {
		t.Error("PreRegistration.AllowGuests should be true by default")
	}
	if !settings.PreRegistration.NotifyOrganizer {
		t.Error("PreRegistration.NotifyOrganizer should be true by default")
	}

	// Check-in defaults
	if !settings.CheckIn.Enabled {
		t.Error("CheckIn.Enabled should be true by default")
	}
	if settings.CheckIn.WindowOpenMinutes != 120 {
		t.Errorf("CheckIn.WindowOpenMinutes should be 120, got %d", settings.CheckIn.WindowOpenMinutes)
	}
	if settings.CheckIn.WindowCloseMinutes != 15 {
		t.Errorf("CheckIn.WindowCloseMinutes should be 15, got %d", settings.CheckIn.WindowCloseMinutes)
	}
	if settings.CheckIn.AutoNoShow {
		t.Error("CheckIn.AutoNoShow should be false by default")
	}
	if len(settings.CheckIn.Methods) == 0 {
		t.Error("CheckIn.Methods should not be empty")
	}

	// Access defaults
	if settings.Access.Mode != "public" {
		t.Errorf("Access.Mode should be 'public', got %s", settings.Access.Mode)
	}
}

func TestRegistrationSettings_Validate(t *testing.T) {
	tests := []struct {
		name       string
		settings   *RegistrationSettings
		wantErrors int
	}{
		{
			name:       "default settings are valid",
			settings:   DefaultRegistrationSettings(),
			wantErrors: 0,
		},
		{
			name: "negative window open minutes",
			settings: &RegistrationSettings{
				CheckIn: CheckInSettings{
					Enabled:           true,
					WindowOpenMinutes: -10,
					Methods:           []string{"self_service"},
				},
				Access: AccessSettings{Mode: "public"},
			},
			wantErrors: 1,
		},
		{
			name: "too early window close",
			settings: &RegistrationSettings{
				CheckIn: CheckInSettings{
					Enabled:            true,
					WindowOpenMinutes:  120,
					WindowCloseMinutes: -120, // 2 hours before start
					Methods:            []string{"self_service"},
				},
				Access: AccessSettings{Mode: "public"},
			},
			wantErrors: 1,
		},
		{
			name: "check-in enabled without methods",
			settings: &RegistrationSettings{
				CheckIn: CheckInSettings{
					Enabled:           true,
					WindowOpenMinutes: 120,
					Methods:           []string{}, // Empty methods
				},
				Access: AccessSettings{Mode: "public"},
			},
			wantErrors: 1,
		},
		{
			name: "invalid check-in method",
			settings: &RegistrationSettings{
				CheckIn: CheckInSettings{
					Enabled:           true,
					WindowOpenMinutes: 120,
					Methods:           []string{"invalid_method"},
				},
				Access: AccessSettings{Mode: "public"},
			},
			wantErrors: 1,
		},
		{
			name: "invalid access mode",
			settings: &RegistrationSettings{
				CheckIn: CheckInSettings{
					Enabled: false,
				},
				Access: AccessSettings{Mode: "invalid"},
			},
			wantErrors: 1,
		},
		{
			name: "valid invite_only mode",
			settings: &RegistrationSettings{
				CheckIn: CheckInSettings{
					Enabled:           true,
					WindowOpenMinutes: 60,
					Methods:           []string{"organizer"},
				},
				Access: AccessSettings{Mode: "invite_only"},
			},
			wantErrors: 0,
		},
		{
			name: "multiple valid methods",
			settings: &RegistrationSettings{
				CheckIn: CheckInSettings{
					Enabled:           true,
					WindowOpenMinutes: 120,
					Methods:           []string{"self_service", "qr_code", "organizer"},
				},
				Access: AccessSettings{Mode: "public"},
			},
			wantErrors: 0,
		},
		{
			name: "check-in disabled - methods not validated",
			settings: &RegistrationSettings{
				CheckIn: CheckInSettings{
					Enabled:           false,
					WindowOpenMinutes: 120,
					Methods:           []string{}, // Empty but OK since disabled
				},
				Access: AccessSettings{Mode: "public"},
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.settings.Validate()
			if len(errors) != tt.wantErrors {
				t.Errorf("Validate() returned %d errors, want %d", len(errors), tt.wantErrors)
				for _, err := range errors {
					t.Logf("  Error: %s - %s", err.Field, err.Message)
				}
			}
		})
	}
}

func TestRegistrationSettings_IsValid(t *testing.T) {
	validSettings := DefaultRegistrationSettings()
	if !validSettings.IsValid() {
		t.Error("Default settings should be valid")
	}

	invalidSettings := &RegistrationSettings{
		CheckIn: CheckInSettings{
			Enabled:           true,
			WindowOpenMinutes: -100,
		},
		Access: AccessSettings{Mode: "invalid"},
	}
	if invalidSettings.IsValid() {
		t.Error("Invalid settings should return false")
	}
}

func TestRegistrationSettings_IsPreRegistrationOpen(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "enabled",
			enabled:  true,
			expected: true,
		},
		{
			name:     "disabled",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &RegistrationSettings{
				PreRegistration: PreRegistrationSettings{
					Enabled: tt.enabled,
				},
			}

			if settings.IsPreRegistrationOpen() != tt.expected {
				t.Errorf("IsPreRegistrationOpen() = %v, want %v", settings.IsPreRegistrationOpen(), tt.expected)
			}
		})
	}
}

func TestRegistrationSettings_HasCheckInMethod(t *testing.T) {
	settings := &RegistrationSettings{
		CheckIn: CheckInSettings{
			Methods: []string{"self_service", "organizer"},
		},
	}

	tests := []struct {
		method   string
		expected bool
	}{
		{"self_service", true},
		{"organizer", true},
		{"qr_code", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if settings.HasCheckInMethod(tt.method) != tt.expected {
				t.Errorf("HasCheckInMethod(%q) = %v, want %v", tt.method, settings.HasCheckInMethod(tt.method), tt.expected)
			}
		})
	}
}

func TestCheckInMethod_Constants(t *testing.T) {
	// Verify method constants match expected values
	if CheckInMethodSelfService != "self_service" {
		t.Errorf("CheckInMethodSelfService = %q, want 'self_service'", CheckInMethodSelfService)
	}
	if CheckInMethodQRCode != "qr_code" {
		t.Errorf("CheckInMethodQRCode = %q, want 'qr_code'", CheckInMethodQRCode)
	}
	if CheckInMethodOrganizer != "organizer" {
		t.Errorf("CheckInMethodOrganizer = %q, want 'organizer'", CheckInMethodOrganizer)
	}
}

func TestAccessMode_Constants(t *testing.T) {
	// Verify access mode constants
	if AccessModePublic != "public" {
		t.Errorf("AccessModePublic = %q, want 'public'", AccessModePublic)
	}
	if AccessModeInviteOnly != "invite_only" {
		t.Errorf("AccessModeInviteOnly = %q, want 'invite_only'", AccessModeInviteOnly)
	}
}
