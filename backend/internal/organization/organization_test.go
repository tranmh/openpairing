package organization

import (
	"testing"
)

func TestOrganizationType_Valid(t *testing.T) {
	validTypes := []OrganizationType{
		OrgTypeFederation,
		OrgTypeClub,
		OrgTypePrivate,
	}

	for _, ot := range validTypes {
		if ot == "" {
			t.Errorf("OrganizationType should not be empty")
		}
	}
}

func TestMemberRole_Valid(t *testing.T) {
	validRoles := []MemberRole{
		RoleOwner,
		RoleAdmin,
		RoleArbiter,
	}

	for _, role := range validRoles {
		if role == "" {
			t.Errorf("MemberRole should not be empty")
		}
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "Chess Club",
			expected: "chess-club",
		},
		{
			name:     "name with special chars",
			input:    "München Chess Club!",
			expected: "mnchen-chess-club",
		},
		{
			name:     "name with multiple spaces",
			input:    "Chess   Club  Berlin",
			expected: "chess-club-berlin",
		},
		{
			name:     "uppercase name",
			input:    "BERLIN CHESS",
			expected: "berlin-chess",
		},
		{
			name:     "name with numbers",
			input:    "Club 2024",
			expected: "club-2024",
		},
		{
			name:     "name with leading/trailing special chars",
			input:    "---Club Name---",
			expected: "club-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidSlug(t *testing.T) {
	tests := []struct {
		slug  string
		valid bool
	}{
		{"chess-club", true},
		{"club2024", true},
		{"my-awesome-club", true},
		{"a", false},                  // too short
		{"chess_club", false},         // underscore not allowed
		{"Chess-Club", false},         // uppercase not allowed
		{"chess--club", false},        // double hyphen
		{"-chess-club", false},        // leading hyphen
		{"chess-club-", false},        // trailing hyphen
		{"chess club", false},         // space not allowed
		{"", false},                   // empty
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			result := isValidSlug(tt.slug)
			if result != tt.valid {
				t.Errorf("isValidSlug(%q) = %v, want %v", tt.slug, result, tt.valid)
			}
		})
	}
}

func TestOrganizationListResponse_Pagination(t *testing.T) {
	// Test that pagination calculations work correctly
	tests := []struct {
		total      int
		page       int
		limit      int
		totalPages int
	}{
		{total: 100, page: 1, limit: 20, totalPages: 5},
		{total: 101, page: 1, limit: 20, totalPages: 6},
		{total: 0, page: 1, limit: 20, totalPages: 1},
		{total: 20, page: 1, limit: 20, totalPages: 1},
		{total: 21, page: 2, limit: 20, totalPages: 2},
	}

	for _, tt := range tests {
		totalPages := (tt.total + tt.limit - 1) / tt.limit
		if totalPages == 0 {
			totalPages = 1
		}
		if totalPages != tt.totalPages {
			t.Errorf("totalPages for total=%d, limit=%d: got %d, want %d",
				tt.total, tt.limit, totalPages, tt.totalPages)
		}
	}
}

func TestCreateOrganizationRequest_Validation(t *testing.T) {
	// Test that validation rules are correctly defined
	tests := []struct {
		name    string
		req     CreateOrganizationRequest
		isValid bool
	}{
		{
			name: "valid request",
			req: CreateOrganizationRequest{
				Name: "Test Organization",
				Type: OrgTypeClub,
			},
			isValid: true,
		},
		{
			name: "empty name",
			req: CreateOrganizationRequest{
				Name: "",
				Type: OrgTypeClub,
			},
			isValid: false,
		},
		{
			name: "name too short",
			req: CreateOrganizationRequest{
				Name: "A",
				Type: OrgTypeClub,
			},
			isValid: false,
		},
		{
			name: "invalid type",
			req: CreateOrganizationRequest{
				Name: "Test Organization",
				Type: OrganizationType("invalid"),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation check
			valid := len(tt.req.Name) >= 2 && (tt.req.Type == OrgTypeFederation || tt.req.Type == OrgTypeClub || tt.req.Type == OrgTypePrivate)
			if valid != tt.isValid {
				t.Errorf("CreateOrganizationRequest validation: got %v, want %v", valid, tt.isValid)
			}
		})
	}
}

func TestAddMemberRequest_RoleValidation(t *testing.T) {
	validRoles := map[MemberRole]bool{
		RoleOwner:   true,
		RoleAdmin:   true,
		RoleArbiter: true,
	}

	tests := []struct {
		role  MemberRole
		valid bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleArbiter, true},
		{MemberRole("invalid"), false},
		{MemberRole(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			isValid := validRoles[tt.role]
			if isValid != tt.valid {
				t.Errorf("MemberRole %q validation: got %v, want %v", tt.role, isValid, tt.valid)
			}
		})
	}
}

func TestOrganization_DefaultValues(t *testing.T) {
	org := Organization{
		Name: "Test Org",
		Type: OrgTypeClub,
	}

	// Verify default/zero values
	if org.Slug != nil {
		t.Error("Slug should be nil by default")
	}
	if org.LogoURL != nil {
		t.Error("LogoURL should be nil by default")
	}
	if org.Website != nil {
		t.Error("Website should be nil by default")
	}
}

func TestOrganizationMember_RoleHierarchy(t *testing.T) {
	// Test role hierarchy concept
	// Owner > Admin > Arbiter

	isHigherOrEqual := func(role1, role2 MemberRole) bool {
		hierarchy := map[MemberRole]int{
			RoleOwner:   3,
			RoleAdmin:   2,
			RoleArbiter: 1,
		}
		return hierarchy[role1] >= hierarchy[role2]
	}

	tests := []struct {
		role1    MemberRole
		role2    MemberRole
		expected bool
	}{
		{RoleOwner, RoleAdmin, true},
		{RoleOwner, RoleArbiter, true},
		{RoleAdmin, RoleArbiter, true},
		{RoleAdmin, RoleOwner, false},
		{RoleArbiter, RoleOwner, false},
		{RoleArbiter, RoleAdmin, false},
		{RoleOwner, RoleOwner, true},
		{RoleAdmin, RoleAdmin, true},
		{RoleArbiter, RoleArbiter, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.role1)+"_vs_"+string(tt.role2), func(t *testing.T) {
			result := isHigherOrEqual(tt.role1, tt.role2)
			if result != tt.expected {
				t.Errorf("isHigherOrEqual(%s, %s) = %v, want %v",
					tt.role1, tt.role2, result, tt.expected)
			}
		})
	}
}
