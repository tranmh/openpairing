package organization

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for organizations
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new organization repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new organization
func (r *Repository) Create(ctx context.Context, org *Organization) error {
	query := `
		INSERT INTO organizations (name, type, slug, logo_url, website)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		org.Name, org.Type, org.Slug, org.LogoURL, org.Website,
	).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
}

// GetByID retrieves an organization by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	query := `
		SELECT id, name, type, slug, logo_url, website, created_at, updated_at
		FROM organizations
		WHERE id = $1`

	var org Organization
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Type, &org.Slug,
		&org.LogoURL, &org.Website, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, err
	}
	return &org, nil
}

// GetBySlug retrieves an organization by slug
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	query := `
		SELECT id, name, type, slug, logo_url, website, created_at, updated_at
		FROM organizations
		WHERE slug = $1`

	var org Organization
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Type, &org.Slug,
		&org.LogoURL, &org.Website, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, err
	}
	return &org, nil
}

// Update updates an organization
func (r *Repository) Update(ctx context.Context, id uuid.UUID, req *UpdateOrganizationRequest) (*Organization, error) {
	// Build dynamic update query
	setClauses := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *req.Name)
		argNum++
	}
	if req.Type != "" {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", argNum))
		args = append(args, req.Type)
		argNum++
	}
	if req.Slug != nil {
		setClauses = append(setClauses, fmt.Sprintf("slug = $%d", argNum))
		args = append(args, *req.Slug)
		argNum++
	}
	if req.LogoURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("logo_url = $%d", argNum))
		args = append(args, *req.LogoURL)
		argNum++
	}
	if req.Website != nil {
		setClauses = append(setClauses, fmt.Sprintf("website = $%d", argNum))
		args = append(args, *req.Website)
		argNum++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE organizations
		SET %s, updated_at = NOW()
		WHERE id = $%d
		RETURNING id, name, type, slug, logo_url, website, created_at, updated_at`,
		strings.Join(setClauses, ", "), argNum)

	var org Organization
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&org.ID, &org.Name, &org.Type, &org.Slug,
		&org.LogoURL, &org.Website, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, err
	}
	return &org, nil
}

// Delete deletes an organization
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM organizations WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("organization not found")
	}
	return nil
}

// List lists organizations with optional filtering
func (r *Repository) List(ctx context.Context, req *ListOrganizationsRequest) ([]Organization, int, error) {
	// Build query with filters
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if req.Query != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR slug ILIKE $%d)", argNum, argNum)
		args = append(args, "%"+req.Query+"%")
		argNum++
	}
	if req.Type != "" {
		whereClause += fmt.Sprintf(" AND type = $%d", argNum)
		args = append(args, req.Type)
		argNum++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (req.Page - 1) * req.Limit
	args = append(args, req.Limit, offset)
	query := fmt.Sprintf(`
		SELECT id, name, type, slug, logo_url, website, created_at, updated_at
		FROM organizations
		%s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var org Organization
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Type, &org.Slug,
			&org.LogoURL, &org.Website, &org.CreatedAt, &org.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		orgs = append(orgs, org)
	}

	return orgs, total, nil
}

// AddMember adds a member to an organization
func (r *Repository) AddMember(ctx context.Context, orgID, userID uuid.UUID, role MemberRole) error {
	query := `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = $3`

	_, err := r.pool.Exec(ctx, query, orgID, userID, role)
	return err
}

// RemoveMember removes a member from an organization
func (r *Repository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	query := `DELETE FROM organization_members WHERE organization_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, orgID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

// UpdateMemberRole updates a member's role
func (r *Repository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role MemberRole) error {
	query := `
		UPDATE organization_members
		SET role = $3
		WHERE organization_id = $1 AND user_id = $2`

	result, err := r.pool.Exec(ctx, query, orgID, userID, role)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

// GetMembers retrieves all members of an organization
func (r *Repository) GetMembers(ctx context.Context, orgID uuid.UUID) ([]OrganizationMember, error) {
	query := `
		SELECT om.organization_id, om.user_id, om.role, om.created_at,
		       u.email as user_email, COALESCE(u.display_name, u.email) as user_name
		FROM organization_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.organization_id = $1
		ORDER BY om.role, u.email`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []OrganizationMember
	for rows.Next() {
		var m OrganizationMember
		if err := rows.Scan(
			&m.OrganizationID, &m.UserID, &m.Role, &m.CreatedAt,
			&m.UserEmail, &m.UserName,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

// GetMember retrieves a specific member of an organization
func (r *Repository) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*OrganizationMember, error) {
	query := `
		SELECT om.organization_id, om.user_id, om.role, om.created_at,
		       u.email as user_email, COALESCE(u.display_name, u.email) as user_name
		FROM organization_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.organization_id = $1 AND om.user_id = $2`

	var m OrganizationMember
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(
		&m.OrganizationID, &m.UserID, &m.Role, &m.CreatedAt,
		&m.UserEmail, &m.UserName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("member not found")
		}
		return nil, err
	}
	return &m, nil
}

// GetUserOrganizations retrieves all organizations a user is a member of
func (r *Repository) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]Organization, error) {
	query := `
		SELECT o.id, o.name, o.type, o.slug, o.logo_url, o.website, o.created_at, o.updated_at
		FROM organizations o
		JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = $1
		ORDER BY o.name`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var org Organization
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Type, &org.Slug,
			&org.LogoURL, &org.Website, &org.CreatedAt, &org.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}

// GetOrganizationTournaments retrieves tournaments belonging to an organization
func (r *Repository) GetOrganizationTournaments(ctx context.Context, orgID uuid.UUID, page, limit int) ([]TournamentSummary, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM tournaments WHERE organization_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, orgID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * limit
	query := `
		SELECT t.id, t.name, t.start_date, COALESCE(t.end_date, t.start_date) as end_date,
		       t.status, t.type,
		       (SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = t.id) as player_count
		FROM tournaments t
		WHERE t.organization_id = $1
		ORDER BY t.start_date DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tournaments []TournamentSummary
	for rows.Next() {
		var t TournamentSummary
		if err := rows.Scan(&t.ID, &t.Name, &t.StartDate, &t.EndDate, &t.Status, &t.Type, &t.Players); err != nil {
			return nil, 0, err
		}
		tournaments = append(tournaments, t)
	}

	return tournaments, total, nil
}

// IsOwner checks if a user is an owner of an organization
func (r *Repository) IsOwner(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM organization_members
			WHERE organization_id = $1 AND user_id = $2 AND role = 'owner'
		)`
	var isOwner bool
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(&isOwner)
	return isOwner, err
}

// IsAdminOrOwner checks if a user is an admin or owner of an organization
func (r *Repository) IsAdminOrOwner(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM organization_members
			WHERE organization_id = $1 AND user_id = $2 AND role IN ('owner', 'admin')
		)`
	var isAdmin bool
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(&isAdmin)
	return isAdmin, err
}

// IsMember checks if a user is a member of an organization
func (r *Repository) IsMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM organization_members
			WHERE organization_id = $1 AND user_id = $2
		)`
	var isMember bool
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(&isMember)
	return isMember, err
}
