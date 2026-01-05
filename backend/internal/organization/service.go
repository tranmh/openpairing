package organization

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles organization business logic
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new organization service
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("service", "organization").Logger(),
	}
}

// CreateOrganization creates a new organization
func (s *Service) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, creatorID uuid.UUID) (*Organization, error) {
	// Generate slug if not provided
	if req.Slug == nil || *req.Slug == "" {
		slug := generateSlug(req.Name)
		req.Slug = &slug
	}

	// Validate slug format
	if !isValidSlug(*req.Slug) {
		return nil, fmt.Errorf("invalid slug format: must be lowercase alphanumeric with hyphens")
	}

	// Check if slug already exists
	existing, err := s.repo.GetBySlug(ctx, *req.Slug)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("slug already exists")
	}

	org := &Organization{
		Name:    req.Name,
		Type:    req.Type,
		Slug:    req.Slug,
		LogoURL: req.LogoURL,
		Website: req.Website,
	}

	if err := s.repo.Create(ctx, org); err != nil {
		s.logger.Error().Err(err).Msg("Failed to create organization")
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Add creator as owner
	if err := s.repo.AddMember(ctx, org.ID, creatorID, RoleOwner); err != nil {
		s.logger.Error().Err(err).Msg("Failed to add creator as owner")
		// Delete the organization since we couldn't add the owner
		_ = s.repo.Delete(ctx, org.ID)
		return nil, fmt.Errorf("failed to add creator as owner: %w", err)
	}

	s.logger.Info().
		Str("org_id", org.ID.String()).
		Str("name", org.Name).
		Str("creator_id", creatorID.String()).
		Msg("Organization created")

	return org, nil
}

// GetOrganization retrieves an organization by ID
func (s *Service) GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error) {
	return s.repo.GetByID(ctx, id)
}

// GetOrganizationWithMembers retrieves an organization with its members
func (s *Service) GetOrganizationWithMembers(ctx context.Context, id uuid.UUID) (*OrganizationWithMembers, error) {
	org, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	members, err := s.repo.GetMembers(ctx, id)
	if err != nil {
		return nil, err
	}

	return &OrganizationWithMembers{
		Organization: *org,
		Members:      members,
	}, nil
}

// UpdateOrganization updates an organization
func (s *Service) UpdateOrganization(ctx context.Context, id uuid.UUID, req *UpdateOrganizationRequest, userID uuid.UUID) (*Organization, error) {
	// Check if user is admin or owner
	isAdmin, err := s.repo.IsAdminOrOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, fmt.Errorf("unauthorized: must be admin or owner to update organization")
	}

	// Validate slug if being updated
	if req.Slug != nil && *req.Slug != "" {
		if !isValidSlug(*req.Slug) {
			return nil, fmt.Errorf("invalid slug format: must be lowercase alphanumeric with hyphens")
		}
		// Check if new slug already exists
		existing, err := s.repo.GetBySlug(ctx, *req.Slug)
		if err == nil && existing != nil && existing.ID != id {
			return nil, fmt.Errorf("slug already exists")
		}
	}

	org, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("org_id", id.String()).
		Str("updated_by", userID.String()).
		Msg("Organization updated")

	return org, nil
}

// DeleteOrganization deletes an organization
func (s *Service) DeleteOrganization(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// Only owners can delete organizations
	isOwner, err := s.repo.IsOwner(ctx, id, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return fmt.Errorf("unauthorized: only owners can delete organizations")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.logger.Info().
		Str("org_id", id.String()).
		Str("deleted_by", userID.String()).
		Msg("Organization deleted")

	return nil
}

// ListOrganizations lists organizations with optional filtering
func (s *Service) ListOrganizations(ctx context.Context, req *ListOrganizationsRequest) (*OrganizationListResponse, error) {
	orgs, total, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	totalPages := (total + req.Limit - 1) / req.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &OrganizationListResponse{
		Organizations: orgs,
		Total:         total,
		Page:          req.Page,
		Limit:         req.Limit,
		TotalPages:    totalPages,
	}, nil
}

// AddMember adds a member to an organization
func (s *Service) AddMember(ctx context.Context, orgID uuid.UUID, req *AddMemberRequest, addedByID uuid.UUID) error {
	// Check if current user is admin or owner
	isAdmin, err := s.repo.IsAdminOrOwner(ctx, orgID, addedByID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("unauthorized: must be admin or owner to add members")
	}

	// Only owners can add other owners
	if req.Role == RoleOwner {
		isOwner, err := s.repo.IsOwner(ctx, orgID, addedByID)
		if err != nil {
			return err
		}
		if !isOwner {
			return fmt.Errorf("unauthorized: only owners can add other owners")
		}
	}

	if err := s.repo.AddMember(ctx, orgID, req.UserID, req.Role); err != nil {
		return err
	}

	s.logger.Info().
		Str("org_id", orgID.String()).
		Str("user_id", req.UserID.String()).
		Str("role", string(req.Role)).
		Str("added_by", addedByID.String()).
		Msg("Member added to organization")

	return nil
}

// RemoveMember removes a member from an organization
func (s *Service) RemoveMember(ctx context.Context, orgID, userID uuid.UUID, removedByID uuid.UUID) error {
	// Check if current user is admin or owner
	isAdmin, err := s.repo.IsAdminOrOwner(ctx, orgID, removedByID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("unauthorized: must be admin or owner to remove members")
	}

	// Get the member to check their role
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return err
	}

	// Only owners can remove other owners
	if member.Role == RoleOwner {
		isOwner, err := s.repo.IsOwner(ctx, orgID, removedByID)
		if err != nil {
			return err
		}
		if !isOwner {
			return fmt.Errorf("unauthorized: only owners can remove other owners")
		}
	}

	// Prevent removing the last owner
	if member.Role == RoleOwner {
		members, err := s.repo.GetMembers(ctx, orgID)
		if err != nil {
			return err
		}
		ownerCount := 0
		for _, m := range members {
			if m.Role == RoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return fmt.Errorf("cannot remove the last owner of an organization")
		}
	}

	if err := s.repo.RemoveMember(ctx, orgID, userID); err != nil {
		return err
	}

	s.logger.Info().
		Str("org_id", orgID.String()).
		Str("user_id", userID.String()).
		Str("removed_by", removedByID.String()).
		Msg("Member removed from organization")

	return nil
}

// UpdateMemberRole updates a member's role in an organization
func (s *Service) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, req *UpdateMemberRequest, updatedByID uuid.UUID) error {
	// Get current member info
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return err
	}

	// Only owners can modify roles
	isOwner, err := s.repo.IsOwner(ctx, orgID, updatedByID)
	if err != nil {
		return err
	}
	if !isOwner {
		return fmt.Errorf("unauthorized: only owners can modify member roles")
	}

	// Prevent demoting the last owner
	if member.Role == RoleOwner && req.Role != RoleOwner {
		members, err := s.repo.GetMembers(ctx, orgID)
		if err != nil {
			return err
		}
		ownerCount := 0
		for _, m := range members {
			if m.Role == RoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return fmt.Errorf("cannot demote the last owner of an organization")
		}
	}

	if err := s.repo.UpdateMemberRole(ctx, orgID, userID, req.Role); err != nil {
		return err
	}

	s.logger.Info().
		Str("org_id", orgID.String()).
		Str("user_id", userID.String()).
		Str("new_role", string(req.Role)).
		Str("updated_by", updatedByID.String()).
		Msg("Member role updated")

	return nil
}

// GetMembers retrieves all members of an organization
func (s *Service) GetMembers(ctx context.Context, orgID uuid.UUID) ([]OrganizationMember, error) {
	return s.repo.GetMembers(ctx, orgID)
}

// GetOrganizationTournaments retrieves tournaments belonging to an organization
func (s *Service) GetOrganizationTournaments(ctx context.Context, orgID uuid.UUID, page, limit int) (*OrganizationTournamentsResponse, error) {
	// Verify organization exists
	_, err := s.repo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	tournaments, total, err := s.repo.GetOrganizationTournaments(ctx, orgID, page, limit)
	if err != nil {
		return nil, err
	}

	return &OrganizationTournamentsResponse{
		Tournaments: tournaments,
		Total:       total,
		Page:        page,
		Limit:       limit,
	}, nil
}

// GetUserOrganizations retrieves all organizations a user belongs to
func (s *Service) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]Organization, error) {
	return s.repo.GetUserOrganizations(ctx, userID)
}

// CheckAccess checks if a user has the required access level
func (s *Service) CheckAccess(ctx context.Context, orgID, userID uuid.UUID, requiredRole MemberRole) (bool, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return false, nil // Not a member
	}

	switch requiredRole {
	case RoleArbiter:
		return true, nil // Any member has at least arbiter-level access
	case RoleAdmin:
		return member.Role == RoleAdmin || member.Role == RoleOwner, nil
	case RoleOwner:
		return member.Role == RoleOwner, nil
	}

	return false, nil
}

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]")
	slug = reg.ReplaceAllString(slug, "")
	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")
	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")
	return slug
}

// isValidSlug validates a slug format
func isValidSlug(slug string) bool {
	if len(slug) < 2 || len(slug) > 100 {
		return false
	}
	matched, _ := regexp.MatchString("^[a-z0-9]+(-[a-z0-9]+)*$", slug)
	return matched
}
