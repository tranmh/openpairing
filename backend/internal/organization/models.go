package organization

import (
	"time"

	"github.com/google/uuid"
)

// OrganizationType represents the type of organization
type OrganizationType string

const (
	OrgTypeFederation OrganizationType = "federation"
	OrgTypeClub       OrganizationType = "club"
	OrgTypePrivate    OrganizationType = "private"
)

// MemberRole represents a member's role in an organization
type MemberRole string

const (
	RoleOwner   MemberRole = "owner"
	RoleAdmin   MemberRole = "admin"
	RoleArbiter MemberRole = "arbiter"
)

// Organization represents an organization entity
type Organization struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	Name      string           `json:"name" db:"name"`
	Type      OrganizationType `json:"type" db:"type"`
	Slug      *string          `json:"slug,omitempty" db:"slug"`
	LogoURL   *string          `json:"logoUrl,omitempty" db:"logo_url"`
	Website   *string          `json:"website,omitempty" db:"website"`
	CreatedAt time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time        `json:"updatedAt" db:"updated_at"`
}

// OrganizationMember represents a member of an organization
type OrganizationMember struct {
	OrganizationID uuid.UUID  `json:"organizationId" db:"organization_id"`
	UserID         uuid.UUID  `json:"userId" db:"user_id"`
	Role           MemberRole `json:"role" db:"role"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	// Populated from join
	UserEmail string `json:"userEmail,omitempty" db:"user_email"`
	UserName  string `json:"userName,omitempty" db:"user_name"`
}

// OrganizationWithMembers includes organization details with its members
type OrganizationWithMembers struct {
	Organization
	Members []OrganizationMember `json:"members"`
}

// CreateOrganizationRequest is the request to create an organization
type CreateOrganizationRequest struct {
	Name    string           `json:"name" binding:"required,min=2,max=255"`
	Type    OrganizationType `json:"type" binding:"required,oneof=federation club private"`
	Slug    *string          `json:"slug,omitempty" binding:"omitempty,min=2,max=100"`
	LogoURL *string          `json:"logoUrl,omitempty" binding:"omitempty,url,max=500"`
	Website *string          `json:"website,omitempty" binding:"omitempty,url,max=500"`
}

// UpdateOrganizationRequest is the request to update an organization
type UpdateOrganizationRequest struct {
	Name    *string          `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	Type    OrganizationType `json:"type,omitempty" binding:"omitempty,oneof=federation club private"`
	Slug    *string          `json:"slug,omitempty" binding:"omitempty,min=2,max=100"`
	LogoURL *string          `json:"logoUrl,omitempty" binding:"omitempty,max=500"`
	Website *string          `json:"website,omitempty" binding:"omitempty,max=500"`
}

// AddMemberRequest is the request to add a member to an organization
type AddMemberRequest struct {
	UserID uuid.UUID  `json:"userId" binding:"required"`
	Role   MemberRole `json:"role" binding:"required,oneof=owner admin arbiter"`
}

// UpdateMemberRequest is the request to update a member's role
type UpdateMemberRequest struct {
	Role MemberRole `json:"role" binding:"required,oneof=owner admin arbiter"`
}

// ListOrganizationsRequest is the request to list organizations
type ListOrganizationsRequest struct {
	Query  string           `form:"q"`
	Type   OrganizationType `form:"type"`
	Page   int              `form:"page,default=1" binding:"min=1"`
	Limit  int              `form:"limit,default=20" binding:"min=1,max=100"`
}

// OrganizationListResponse is the paginated list of organizations
type OrganizationListResponse struct {
	Organizations []Organization `json:"organizations"`
	Total         int            `json:"total"`
	Page          int            `json:"page"`
	Limit         int            `json:"limit"`
	TotalPages    int            `json:"totalPages"`
}

// TournamentSummary is a brief summary of a tournament for listing
type TournamentSummary struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	StartDate time.Time `json:"startDate" db:"start_date"`
	EndDate   time.Time `json:"endDate" db:"end_date"`
	Status    string    `json:"status" db:"status"`
	Type      string    `json:"type" db:"type"`
	Players   int       `json:"players" db:"player_count"`
}

// OrganizationTournamentsResponse is the response for organization tournaments
type OrganizationTournamentsResponse struct {
	Tournaments []TournamentSummary `json:"tournaments"`
	Total       int                 `json:"total"`
	Page        int                 `json:"page"`
	Limit       int                 `json:"limit"`
}
