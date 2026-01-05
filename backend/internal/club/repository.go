package club

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for club data operations
type Repository interface {
	Create(ctx context.Context, club *Club) error
	GetByID(ctx context.Context, id uuid.UUID) (*Club, error)
	GetByDsbID(ctx context.Context, dsbID string) (*Club, error)
	GetByOesbID(ctx context.Context, oesbID string) (*Club, error)
	Update(ctx context.Context, club *Club) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ClubListParams) (*ClubListResponse, error)
	GetMembers(ctx context.Context, clubID uuid.UUID) ([]ClubMember, error)
	GetStats(ctx context.Context, clubID uuid.UUID) (*ClubStats, error)
	Search(ctx context.Context, query string, limit int) ([]Club, error)
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL club repository
func NewRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create creates a new club in the database
func (r *PostgresRepository) Create(ctx context.Context, club *Club) error {
	club.ID = uuid.New()
	club.CreatedAt = time.Now()
	club.UpdatedAt = time.Now()

	query := `
		INSERT INTO clubs (id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, website, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.Exec(ctx, query,
		club.ID, club.Name, club.ShortName, club.Federation,
		club.DsbClubID, club.OesbClubID, club.City, club.Country,
		club.Website, club.Email, club.CreatedAt, club.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrDuplicateClub
		}
		return fmt.Errorf("failed to create club: %w", err)
	}

	return nil
}

// GetByID retrieves a club by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Club, error) {
	query := `
		SELECT id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, website, email, created_at, updated_at
		FROM clubs WHERE id = $1
	`

	var club Club
	err := r.db.QueryRow(ctx, query, id).Scan(
		&club.ID, &club.Name, &club.ShortName, &club.Federation,
		&club.DsbClubID, &club.OesbClubID, &club.City, &club.Country,
		&club.Website, &club.Email, &club.CreatedAt, &club.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClubNotFound
		}
		return nil, fmt.Errorf("failed to get club: %w", err)
	}

	return &club, nil
}

// GetByDsbID retrieves a club by DSB club ID
func (r *PostgresRepository) GetByDsbID(ctx context.Context, dsbID string) (*Club, error) {
	query := `
		SELECT id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, website, email, created_at, updated_at
		FROM clubs WHERE dsb_club_id = $1
	`

	var club Club
	err := r.db.QueryRow(ctx, query, dsbID).Scan(
		&club.ID, &club.Name, &club.ShortName, &club.Federation,
		&club.DsbClubID, &club.OesbClubID, &club.City, &club.Country,
		&club.Website, &club.Email, &club.CreatedAt, &club.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClubNotFound
		}
		return nil, fmt.Errorf("failed to get club by DSB ID: %w", err)
	}

	return &club, nil
}

// GetByOesbID retrieves a club by ÖSB club ID
func (r *PostgresRepository) GetByOesbID(ctx context.Context, oesbID string) (*Club, error) {
	query := `
		SELECT id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, website, email, created_at, updated_at
		FROM clubs WHERE oesb_club_id = $1
	`

	var club Club
	err := r.db.QueryRow(ctx, query, oesbID).Scan(
		&club.ID, &club.Name, &club.ShortName, &club.Federation,
		&club.DsbClubID, &club.OesbClubID, &club.City, &club.Country,
		&club.Website, &club.Email, &club.CreatedAt, &club.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClubNotFound
		}
		return nil, fmt.Errorf("failed to get club by ÖSB ID: %w", err)
	}

	return &club, nil
}

// Update updates an existing club
func (r *PostgresRepository) Update(ctx context.Context, club *Club) error {
	club.UpdatedAt = time.Now()

	query := `
		UPDATE clubs SET
			name = $1, short_name = $2, federation = $3,
			dsb_club_id = $4, oesb_club_id = $5, city = $6, country = $7,
			website = $8, email = $9, updated_at = $10
		WHERE id = $11
	`

	result, err := r.db.Exec(ctx, query,
		club.Name, club.ShortName, club.Federation,
		club.DsbClubID, club.OesbClubID, club.City, club.Country,
		club.Website, club.Email, club.UpdatedAt, club.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update club: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrClubNotFound
	}

	return nil
}

// Delete deletes a club by ID
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM clubs WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete club: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrClubNotFound
	}

	return nil
}

// List lists clubs with pagination and filtering
func (r *PostgresRepository) List(ctx context.Context, params ClubListParams) (*ClubListResponse, error) {
	// Build query
	var conditions []string
	var args []interface{}
	argNum := 1

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR short_name ILIKE $%d OR city ILIKE $%d)", argNum, argNum, argNum))
		args = append(args, "%"+params.Search+"%")
		argNum++
	}

	if params.Federation != "" {
		conditions = append(conditions, fmt.Sprintf("federation = $%d", argNum))
		args = append(args, params.Federation)
		argNum++
	}

	if params.City != "" {
		conditions = append(conditions, fmt.Sprintf("city ILIKE $%d", argNum))
		args = append(args, "%"+params.City+"%")
		argNum++
	}

	if params.Country != "" {
		conditions = append(conditions, fmt.Sprintf("country = $%d", argNum))
		args = append(args, params.Country)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM clubs %s", whereClause)
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count clubs: %w", err)
	}

	// Apply pagination
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize
	args = append(args, params.PageSize, offset)

	query := fmt.Sprintf(`
		SELECT id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, website, email, created_at, updated_at
		FROM clubs %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list clubs: %w", err)
	}
	defer rows.Close()

	var clubs []Club
	for rows.Next() {
		var club Club
		if err := rows.Scan(
			&club.ID, &club.Name, &club.ShortName, &club.Federation,
			&club.DsbClubID, &club.OesbClubID, &club.City, &club.Country,
			&club.Website, &club.Email, &club.CreatedAt, &club.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan club: %w", err)
		}
		clubs = append(clubs, club)
	}

	totalPages := (totalCount + params.PageSize - 1) / params.PageSize

	return &ClubListResponse{
		Clubs:      clubs,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetMembers retrieves all members of a club
func (r *PostgresRepository) GetMembers(ctx context.Context, clubID uuid.UUID) ([]ClubMember, error) {
	query := `
		SELECT p.id, p.first_name || ' ' || p.last_name as name,
			COALESCE(p.fide_standard, p.dwz, 0) as rating,
			p.fide_title, p.created_at
		FROM players p
		WHERE p.club_id = $1
		ORDER BY COALESCE(p.fide_standard, p.dwz, 0) DESC
	`

	rows, err := r.db.Query(ctx, query, clubID)
	if err != nil {
		return nil, fmt.Errorf("failed to get club members: %w", err)
	}
	defer rows.Close()

	var members []ClubMember
	for rows.Next() {
		var m ClubMember
		if err := rows.Scan(&m.PlayerID, &m.PlayerName, &m.Rating, &m.Title, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, m)
	}

	return members, nil
}

// GetStats retrieves statistics for a club
func (r *PostgresRepository) GetStats(ctx context.Context, clubID uuid.UUID) (*ClubStats, error) {
	query := `
		SELECT
			COUNT(*) as member_count,
			COALESCE(AVG(NULLIF(COALESCE(fide_standard, dwz, 0), 0)), 0) as avg_rating,
			COALESCE(MAX(COALESCE(fide_standard, dwz, 0)), 0) as max_rating,
			COUNT(CASE WHEN fide_title IS NOT NULL AND fide_title != '' THEN 1 END) as titled_players
		FROM players
		WHERE club_id = $1
	`

	var stats ClubStats
	err := r.db.QueryRow(ctx, query, clubID).Scan(
		&stats.MemberCount, &stats.AverageRating,
		&stats.HighestRating, &stats.TitledPlayers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get club stats: %w", err)
	}

	// Get active teams count
	teamsQuery := `SELECT COUNT(*) FROM teams WHERE club_id = $1`
	r.db.QueryRow(ctx, teamsQuery, clubID).Scan(&stats.ActiveTeams)

	return &stats, nil
}

// Search searches for clubs by name
func (r *PostgresRepository) Search(ctx context.Context, query string, limit int) ([]Club, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	sqlQuery := `
		SELECT id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, website, email, created_at, updated_at
		FROM clubs
		WHERE name ILIKE $1 OR short_name ILIKE $1
		ORDER BY name ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, sqlQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search clubs: %w", err)
	}
	defer rows.Close()

	var clubs []Club
	for rows.Next() {
		var club Club
		if err := rows.Scan(
			&club.ID, &club.Name, &club.ShortName, &club.Federation,
			&club.DsbClubID, &club.OesbClubID, &club.City, &club.Country,
			&club.Website, &club.Email, &club.CreatedAt, &club.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan club: %w", err)
		}
		clubs = append(clubs, club)
	}

	return clubs, nil
}
