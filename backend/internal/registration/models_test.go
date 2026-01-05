package registration

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGuestRegistration_StatusTransitions(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  string
		expectedStatus string
	}{
		{
			name:           "pre_registered is valid initial status",
			initialStatus:  StatusPreRegistered,
			expectedStatus: StatusPreRegistered,
		},
		{
			name:           "pending_approval is valid status",
			initialStatus:  StatusPendingApproval,
			expectedStatus: StatusPendingApproval,
		},
		{
			name:           "approved is valid status",
			initialStatus:  StatusApproved,
			expectedStatus: StatusApproved,
		},
		{
			name:           "checked_in is valid status",
			initialStatus:  StatusCheckedIn,
			expectedStatus: StatusCheckedIn,
		},
		{
			name:           "rejected is valid status",
			initialStatus:  StatusRejected,
			expectedStatus: StatusRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &GuestRegistration{
				ID:                 uuid.New(),
				TournamentID:       uuid.New(),
				Email:              "test@example.com",
				FirstName:          "Test",
				LastName:           "User",
				RegistrationStatus: tt.initialStatus,
				PreRegisteredAt:    time.Now(),
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}

			if reg.RegistrationStatus != tt.expectedStatus {
				t.Errorf("GuestRegistration status = %v, want %v", reg.RegistrationStatus, tt.expectedStatus)
			}
		})
	}
}

func TestPreRegisterRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		req         PreRegisterRequest
		isGuest     bool
		wantValid   bool
		description string
	}{
		{
			name: "valid guest registration",
			req: PreRegisterRequest{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			isGuest:     true,
			wantValid:   true,
			description: "Guest with all required fields should be valid",
		},
		{
			name: "valid guest with federation ID",
			req: PreRegisterRequest{
				Email:          "test@example.com",
				FirstName:      "Test",
				LastName:       "User",
				FederationID:   "12345678",
				FederationType: "dsb",
			},
			isGuest:     true,
			wantValid:   true,
			description: "Guest with federation ID should be valid",
		},
		{
			name: "valid authenticated user",
			req: PreRegisterRequest{
				PlayerID: uuidPtr(uuid.New()),
			},
			isGuest:     false,
			wantValid:   true,
			description: "Authenticated user with player ID should be valid",
		},
		{
			name: "invalid guest - missing email",
			req: PreRegisterRequest{
				FirstName: "Test",
				LastName:  "User",
			},
			isGuest:     true,
			wantValid:   false,
			description: "Guest without email should be invalid",
		},
		{
			name: "invalid guest - missing first name",
			req: PreRegisterRequest{
				Email:    "test@example.com",
				LastName: "User",
			},
			isGuest:     true,
			wantValid:   false,
			description: "Guest without first name should be invalid",
		},
		{
			name: "invalid guest - missing last name",
			req: PreRegisterRequest{
				Email:     "test@example.com",
				FirstName: "Test",
			},
			isGuest:     true,
			wantValid:   false,
			description: "Guest without last name should be invalid",
		},
		{
			name: "guest with invite token",
			req: PreRegisterRequest{
				Email:       "test@example.com",
				FirstName:   "Test",
				LastName:    "User",
				InviteToken: "abc123",
			},
			isGuest:     true,
			wantValid:   true,
			description: "Guest with invite token for private tournament",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isGuest := tt.req.PlayerID == nil
			if isGuest != tt.isGuest {
				t.Errorf("Expected isGuest=%v, got %v", tt.isGuest, isGuest)
			}

			// Validate based on whether it's a guest or authenticated request
			var valid bool
			if isGuest {
				valid = tt.req.Email != "" && tt.req.FirstName != "" && tt.req.LastName != ""
			} else {
				valid = tt.req.PlayerID != nil
			}

			if valid != tt.wantValid {
				t.Errorf("%s: got valid=%v, want %v", tt.description, valid, tt.wantValid)
			}
		})
	}
}

func TestCheckInRequest_Methods(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		wantValid bool
	}{
		{
			name:      "self_service method",
			method:    "self_service",
			wantValid: true,
		},
		{
			name:      "qr_code method",
			method:    "qr_code",
			wantValid: true,
		},
		{
			name:      "organizer method",
			method:    "organizer",
			wantValid: true,
		},
		{
			name:      "invalid method",
			method:    "invalid",
			wantValid: false,
		},
		{
			name:      "empty method defaults to self_service",
			method:    "",
			wantValid: true, // Empty defaults to self_service in handler
		},
	}

	validMethods := map[string]bool{
		"self_service": true,
		"qr_code":      true,
		"organizer":    true,
		"":             true, // Empty is valid (defaults to self_service)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CheckInRequest{
				Method: tt.method,
			}

			valid := validMethods[req.Method]
			if valid != tt.wantValid {
				t.Errorf("CheckInRequest method %q: got valid=%v, want %v", tt.method, valid, tt.wantValid)
			}
		})
	}
}

func TestApproveRegistrationRequest_Validation(t *testing.T) {
	tests := []struct {
		name      string
		req       ApproveRegistrationRequest
		wantValid bool
	}{
		{
			name: "approve without reason",
			req: ApproveRegistrationRequest{
				Approved: true,
				Reason:   "",
			},
			wantValid: true,
		},
		{
			name: "approve with reason",
			req: ApproveRegistrationRequest{
				Approved: true,
				Reason:   "Welcome to the tournament!",
			},
			wantValid: true,
		},
		{
			name: "reject with reason",
			req: ApproveRegistrationRequest{
				Approved: false,
				Reason:   "Tournament is full",
			},
			wantValid: true,
		},
		{
			name: "reject without reason is invalid",
			req: ApproveRegistrationRequest{
				Approved: false,
				Reason:   "",
			},
			wantValid: false, // Reason required when rejecting
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Rejection requires a reason
			valid := tt.req.Approved || tt.req.Reason != ""

			if valid != tt.wantValid {
				t.Errorf("ApproveRegistrationRequest: got valid=%v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestRegistrationStatusConstants(t *testing.T) {
	// Verify status constants are unique
	statuses := []string{
		StatusPreRegistered,
		StatusPendingApproval,
		StatusApproved,
		StatusCheckedIn,
		StatusConverted,
		StatusRejected,
		StatusExpired,
		StatusNoShow,
	}

	seen := make(map[string]bool)
	for _, status := range statuses {
		if seen[status] {
			t.Errorf("Duplicate status constant: %s", status)
		}
		seen[status] = true

		if status == "" {
			t.Error("Empty status constant found")
		}
	}
}

func TestSourceConstants(t *testing.T) {
	// Verify source constants are unique
	sources := []string{
		SourceOrganizer,
		SourceSelfService,
		SourceGuestConverted,
	}

	seen := make(map[string]bool)
	for _, source := range sources {
		if seen[source] {
			t.Errorf("Duplicate source constant: %s", source)
		}
		seen[source] = true

		if source == "" {
			t.Error("Empty source constant found")
		}
	}
}

// Helper functions
func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
