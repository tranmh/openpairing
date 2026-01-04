package player

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

var (
	ErrPlayerNotFound     = errors.New("player not found")
	ErrDuplicateFideID    = errors.New("player with this FIDE ID already exists")
	ErrDuplicateDsbID     = errors.New("player with this DSB ID already exists")
	ErrDuplicateOesbID    = errors.New("player with this ÖSB ID already exists")
	ErrClubNotFound       = errors.New("club not found")
)

type Repository interface {
	Create(ctx context.Context, player *Player) error
	GetByID(ctx context.Context, id uuid.UUID) (*Player, error)
	GetByFideID(ctx context.Context, fideID string) (*Player, error)
	GetByDsbID(ctx context.Context, dsbID string) (*Player, error)
	GetByOesbID(ctx context.Context, oesbID string) (*Player, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Player, error)
	Update(ctx context.Context, player *Player) error
	Delete(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, req *SearchPlayersRequest) ([]PlayerSearchResult, int64, error)
	List(ctx context.Context, page, limit int) ([]Player, int64, error)

	// Club operations
	CreateClub(ctx context.Context, club *Club) error
	GetClubByID(ctx context.Context, id uuid.UUID) (*Club, error)
	ListClubs(ctx context.Context, page, limit int) ([]Club, int64, error)

	// Rating history
	AddRatingHistory(ctx context.Context, history *RatingHistory) error
	GetRatingHistory(ctx context.Context, playerID uuid.UUID, ratingType string, limit int) ([]RatingHistory, error)

	// Bulk operations for federation sync
	UpsertByFideID(ctx context.Context, player *Player) error
	UpsertByDsbID(ctx context.Context, player *Player) error
	UpsertByOesbID(ctx context.Context, player *Player) error
	UpsertBySsbID(ctx context.Context, player *Player) error
	BulkUpsertByFideID(ctx context.Context, players []*Player) error
	BulkUpsertByOesbID(ctx context.Context, players []*Player) error
	BulkUpsertBySsbID(ctx context.Context, players []*Player) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, player *Player) error {
	player.ID = uuid.New()
	player.CreatedAt = time.Now()
	player.UpdatedAt = time.Now()

	query := `
		INSERT INTO players (
			id, user_id, first_name, last_name, birth_date, gender,
			fide_id, dsb_id, oesb_id, ssb_id,
			fide_standard, fide_rapid, fide_blitz, dwz, dwz_index, oesb_elo,
			fide_title, national_title, club_id, federation,
			ratings_updated_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20,
			$21, $22, $23
		)`

	_, err := r.db.Exec(ctx, query,
		player.ID, player.UserID, player.FirstName, player.LastName, player.BirthDate, player.Gender,
		player.FideID, player.DsbID, player.OesbID, player.SsbID,
		player.FideStandard, player.FideRapid, player.FideBlitz, player.DWZ, player.DWZIndex, player.OesbElo,
		player.FideTitle, player.NationalTitle, player.ClubID, player.Federation,
		player.RatingsUpdatedAt, player.CreatedAt, player.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "players_fide_id_key") {
			return ErrDuplicateFideID
		}
		if strings.Contains(err.Error(), "players_dsb_id_key") {
			return ErrDuplicateDsbID
		}
		if strings.Contains(err.Error(), "players_oesb_id_key") {
			return ErrDuplicateOesbID
		}
		return fmt.Errorf("failed to create player: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Player, error) {
	query := `
		SELECT
			p.id, p.user_id, p.first_name, p.last_name, p.birth_date, p.gender,
			p.fide_id, p.dsb_id, p.oesb_id, p.ssb_id,
			p.fide_standard, p.fide_rapid, p.fide_blitz, p.dwz, p.dwz_index, p.oesb_elo,
			p.fide_title, p.national_title, p.club_id, p.federation,
			p.ratings_updated_at, p.created_at, p.updated_at,
			c.name as club_name
		FROM players p
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE p.id = $1`

	var player Player
	err := r.db.QueryRow(ctx, query, id).Scan(
		&player.ID, &player.UserID, &player.FirstName, &player.LastName, &player.BirthDate, &player.Gender,
		&player.FideID, &player.DsbID, &player.OesbID, &player.SsbID,
		&player.FideStandard, &player.FideRapid, &player.FideBlitz, &player.DWZ, &player.DWZIndex, &player.OesbElo,
		&player.FideTitle, &player.NationalTitle, &player.ClubID, &player.Federation,
		&player.RatingsUpdatedAt, &player.CreatedAt, &player.UpdatedAt,
		&player.ClubName,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	return &player, nil
}

func (r *PostgresRepository) GetByFideID(ctx context.Context, fideID string) (*Player, error) {
	query := `
		SELECT
			id, user_id, first_name, last_name, birth_date, gender,
			fide_id, dsb_id, oesb_id, ssb_id,
			fide_standard, fide_rapid, fide_blitz, dwz, dwz_index, oesb_elo,
			fide_title, national_title, club_id, federation,
			ratings_updated_at, created_at, updated_at
		FROM players
		WHERE fide_id = $1`

	var player Player
	err := r.db.QueryRow(ctx, query, fideID).Scan(
		&player.ID, &player.UserID, &player.FirstName, &player.LastName, &player.BirthDate, &player.Gender,
		&player.FideID, &player.DsbID, &player.OesbID, &player.SsbID,
		&player.FideStandard, &player.FideRapid, &player.FideBlitz, &player.DWZ, &player.DWZIndex, &player.OesbElo,
		&player.FideTitle, &player.NationalTitle, &player.ClubID, &player.Federation,
		&player.RatingsUpdatedAt, &player.CreatedAt, &player.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player by FIDE ID: %w", err)
	}

	return &player, nil
}

func (r *PostgresRepository) GetByDsbID(ctx context.Context, dsbID string) (*Player, error) {
	query := `
		SELECT
			id, user_id, first_name, last_name, birth_date, gender,
			fide_id, dsb_id, oesb_id, ssb_id,
			fide_standard, fide_rapid, fide_blitz, dwz, dwz_index, oesb_elo,
			fide_title, national_title, club_id, federation,
			ratings_updated_at, created_at, updated_at
		FROM players
		WHERE dsb_id = $1`

	var player Player
	err := r.db.QueryRow(ctx, query, dsbID).Scan(
		&player.ID, &player.UserID, &player.FirstName, &player.LastName, &player.BirthDate, &player.Gender,
		&player.FideID, &player.DsbID, &player.OesbID, &player.SsbID,
		&player.FideStandard, &player.FideRapid, &player.FideBlitz, &player.DWZ, &player.DWZIndex, &player.OesbElo,
		&player.FideTitle, &player.NationalTitle, &player.ClubID, &player.Federation,
		&player.RatingsUpdatedAt, &player.CreatedAt, &player.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player by DSB ID: %w", err)
	}

	return &player, nil
}

func (r *PostgresRepository) GetByOesbID(ctx context.Context, oesbID string) (*Player, error) {
	query := `
		SELECT
			id, user_id, first_name, last_name, birth_date, gender,
			fide_id, dsb_id, oesb_id, ssb_id,
			fide_standard, fide_rapid, fide_blitz, dwz, dwz_index, oesb_elo,
			fide_title, national_title, club_id, federation,
			ratings_updated_at, created_at, updated_at
		FROM players
		WHERE oesb_id = $1`

	var player Player
	err := r.db.QueryRow(ctx, query, oesbID).Scan(
		&player.ID, &player.UserID, &player.FirstName, &player.LastName, &player.BirthDate, &player.Gender,
		&player.FideID, &player.DsbID, &player.OesbID, &player.SsbID,
		&player.FideStandard, &player.FideRapid, &player.FideBlitz, &player.DWZ, &player.DWZIndex, &player.OesbElo,
		&player.FideTitle, &player.NationalTitle, &player.ClubID, &player.Federation,
		&player.RatingsUpdatedAt, &player.CreatedAt, &player.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player by ÖSB ID: %w", err)
	}

	return &player, nil
}

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Player, error) {
	query := `
		SELECT
			id, user_id, first_name, last_name, birth_date, gender,
			fide_id, dsb_id, oesb_id, ssb_id,
			fide_standard, fide_rapid, fide_blitz, dwz, dwz_index, oesb_elo,
			fide_title, national_title, club_id, federation,
			ratings_updated_at, created_at, updated_at
		FROM players
		WHERE user_id = $1`

	var player Player
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&player.ID, &player.UserID, &player.FirstName, &player.LastName, &player.BirthDate, &player.Gender,
		&player.FideID, &player.DsbID, &player.OesbID, &player.SsbID,
		&player.FideStandard, &player.FideRapid, &player.FideBlitz, &player.DWZ, &player.DWZIndex, &player.OesbElo,
		&player.FideTitle, &player.NationalTitle, &player.ClubID, &player.Federation,
		&player.RatingsUpdatedAt, &player.CreatedAt, &player.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player by user ID: %w", err)
	}

	return &player, nil
}

func (r *PostgresRepository) Update(ctx context.Context, player *Player) error {
	player.UpdatedAt = time.Now()

	query := `
		UPDATE players SET
			first_name = $2, last_name = $3, birth_date = $4, gender = $5,
			fide_id = $6, dsb_id = $7, oesb_id = $8, ssb_id = $9,
			fide_standard = $10, fide_rapid = $11, fide_blitz = $12, dwz = $13, dwz_index = $14, oesb_elo = $15,
			fide_title = $16, national_title = $17, club_id = $18, federation = $19,
			ratings_updated_at = $20, updated_at = $21
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query,
		player.ID,
		player.FirstName, player.LastName, player.BirthDate, player.Gender,
		player.FideID, player.DsbID, player.OesbID, player.SsbID,
		player.FideStandard, player.FideRapid, player.FideBlitz, player.DWZ, player.DWZIndex, player.OesbElo,
		player.FideTitle, player.NationalTitle, player.ClubID, player.Federation,
		player.RatingsUpdatedAt, player.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "players_fide_id_key") {
			return ErrDuplicateFideID
		}
		if strings.Contains(err.Error(), "players_dsb_id_key") {
			return ErrDuplicateDsbID
		}
		if strings.Contains(err.Error(), "players_oesb_id_key") {
			return ErrDuplicateOesbID
		}
		return fmt.Errorf("failed to update player: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPlayerNotFound
	}

	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM players WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete player: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrPlayerNotFound
	}
	return nil
}

func (r *PostgresRepository) Search(ctx context.Context, req *SearchPlayersRequest) ([]PlayerSearchResult, int64, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build search query with full-text search
	baseQuery := `
		SELECT
			p.id, p.user_id, p.first_name, p.last_name, p.birth_date, p.gender,
			p.fide_id, p.dsb_id, p.oesb_id, p.ssb_id,
			p.fide_standard, p.fide_rapid, p.fide_blitz, p.dwz, p.dwz_index, p.oesb_elo,
			p.fide_title, p.national_title, p.club_id, p.federation,
			p.ratings_updated_at, p.created_at, p.updated_at,
			c.name as club_name`

	var rankSelect string
	if req.Query != "" {
		// Add full-text search rank
		rankSelect = `,
			ts_rank(
				to_tsvector('simple', coalesce(p.first_name, '') || ' ' || coalesce(p.last_name, '')),
				plainto_tsquery('simple', $` + fmt.Sprintf("%d", argIndex) + `)
			) as rank`
		args = append(args, req.Query)
		argIndex++
	} else {
		rankSelect = `, 0 as rank`
	}

	fromClause := `
		FROM players p
		LEFT JOIN clubs c ON p.club_id = c.id`

	// Add search condition
	if req.Query != "" {
		// Search by name (full-text), FIDE ID, DSB ID, or ÖSB ID
		conditions = append(conditions, fmt.Sprintf(`(
			to_tsvector('simple', coalesce(p.first_name, '') || ' ' || coalesce(p.last_name, '')) @@ plainto_tsquery('simple', $%d)
			OR p.fide_id = $%d
			OR p.dsb_id = $%d
			OR p.oesb_id = $%d
			OR LOWER(p.first_name || ' ' || p.last_name) LIKE LOWER($%d)
		)`, argIndex, argIndex+1, argIndex+2, argIndex+3, argIndex+4))
		args = append(args, req.Query, req.Query, req.Query, req.Query, "%"+req.Query+"%")
		argIndex += 5
	}

	// Add federation filter
	if req.Federation != nil {
		switch *req.Federation {
		case "fide":
			conditions = append(conditions, fmt.Sprintf("p.fide_id IS NOT NULL"))
		case "dsb":
			conditions = append(conditions, fmt.Sprintf("p.dsb_id IS NOT NULL"))
		case "oesb":
			conditions = append(conditions, fmt.Sprintf("p.oesb_id IS NOT NULL"))
		}
	}

	// Add rating filters
	if req.MinRating != nil {
		conditions = append(conditions, fmt.Sprintf(`(
			COALESCE(p.fide_standard, 0) >= $%d OR
			COALESCE(p.dwz, 0) >= $%d OR
			COALESCE(p.oesb_elo, 0) >= $%d
		)`, argIndex, argIndex, argIndex))
		args = append(args, *req.MinRating)
		argIndex++
	}

	if req.MaxRating != nil {
		conditions = append(conditions, fmt.Sprintf(`(
			(p.fide_standard IS NOT NULL AND p.fide_standard <= $%d) OR
			(p.dwz IS NOT NULL AND p.dwz <= $%d) OR
			(p.oesb_elo IS NOT NULL AND p.oesb_elo <= $%d)
		)`, argIndex, argIndex, argIndex))
		args = append(args, *req.MaxRating)
		argIndex++
	}

	// Add club filter
	if req.ClubID != nil {
		clubUUID, err := uuid.Parse(*req.ClubID)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("p.club_id = $%d", argIndex))
			args = append(args, clubUUID)
			argIndex++
		}
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total results
	countQuery := "SELECT COUNT(*) " + fromClause + whereClause
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count players: %w", err)
	}

	// Add ordering and pagination
	orderClause := " ORDER BY "
	if req.Query != "" {
		orderClause += "rank DESC, "
	}
	orderClause += "p.last_name ASC, p.first_name ASC"

	offset := (req.Page - 1) * req.Limit
	paginationClause := fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, req.Limit, offset)

	// Execute main query
	fullQuery := baseQuery + rankSelect + fromClause + whereClause + orderClause + paginationClause
	rows, err := r.db.Query(ctx, fullQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search players: %w", err)
	}
	defer rows.Close()

	var results []PlayerSearchResult
	for rows.Next() {
		var result PlayerSearchResult
		err := rows.Scan(
			&result.ID, &result.UserID, &result.FirstName, &result.LastName, &result.BirthDate, &result.Gender,
			&result.FideID, &result.DsbID, &result.OesbID, &result.SsbID,
			&result.FideStandard, &result.FideRapid, &result.FideBlitz, &result.DWZ, &result.DWZIndex, &result.OesbElo,
			&result.FideTitle, &result.NationalTitle, &result.ClubID, &result.Federation,
			&result.RatingsUpdatedAt, &result.CreatedAt, &result.UpdatedAt,
			&result.ClubName, &result.Rank,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan player: %w", err)
		}
		results = append(results, result)
	}

	return results, total, nil
}

func (r *PostgresRepository) List(ctx context.Context, page, limit int) ([]Player, int64, error) {
	// Count total
	var total int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM players").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count players: %w", err)
	}

	offset := (page - 1) * limit
	query := `
		SELECT
			p.id, p.user_id, p.first_name, p.last_name, p.birth_date, p.gender,
			p.fide_id, p.dsb_id, p.oesb_id, p.ssb_id,
			p.fide_standard, p.fide_rapid, p.fide_blitz, p.dwz, p.dwz_index, p.oesb_elo,
			p.fide_title, p.national_title, p.club_id, p.federation,
			p.ratings_updated_at, p.created_at, p.updated_at,
			c.name as club_name
		FROM players p
		LEFT JOIN clubs c ON p.club_id = c.id
		ORDER BY p.last_name, p.first_name
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list players: %w", err)
	}
	defer rows.Close()

	var players []Player
	for rows.Next() {
		var player Player
		err := rows.Scan(
			&player.ID, &player.UserID, &player.FirstName, &player.LastName, &player.BirthDate, &player.Gender,
			&player.FideID, &player.DsbID, &player.OesbID, &player.SsbID,
			&player.FideStandard, &player.FideRapid, &player.FideBlitz, &player.DWZ, &player.DWZIndex, &player.OesbElo,
			&player.FideTitle, &player.NationalTitle, &player.ClubID, &player.Federation,
			&player.RatingsUpdatedAt, &player.CreatedAt, &player.UpdatedAt,
			&player.ClubName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan player: %w", err)
		}
		players = append(players, player)
	}

	return players, total, nil
}

func (r *PostgresRepository) CreateClub(ctx context.Context, club *Club) error {
	club.ID = uuid.New()
	club.CreatedAt = time.Now()

	query := `
		INSERT INTO clubs (id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.Exec(ctx, query,
		club.ID, club.Name, club.ShortName, club.Federation,
		club.DsbClubID, club.OesbClubID, club.City, club.Country, club.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create club: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetClubByID(ctx context.Context, id uuid.UUID) (*Club, error) {
	query := `
		SELECT id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, created_at
		FROM clubs WHERE id = $1`

	var club Club
	err := r.db.QueryRow(ctx, query, id).Scan(
		&club.ID, &club.Name, &club.ShortName, &club.Federation,
		&club.DsbClubID, &club.OesbClubID, &club.City, &club.Country, &club.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrClubNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get club: %w", err)
	}

	return &club, nil
}

func (r *PostgresRepository) ListClubs(ctx context.Context, page, limit int) ([]Club, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM clubs").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count clubs: %w", err)
	}

	offset := (page - 1) * limit
	query := `
		SELECT id, name, short_name, federation, dsb_club_id, oesb_club_id, city, country, created_at
		FROM clubs
		ORDER BY name
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list clubs: %w", err)
	}
	defer rows.Close()

	var clubs []Club
	for rows.Next() {
		var club Club
		err := rows.Scan(
			&club.ID, &club.Name, &club.ShortName, &club.Federation,
			&club.DsbClubID, &club.OesbClubID, &club.City, &club.Country, &club.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan club: %w", err)
		}
		clubs = append(clubs, club)
	}

	return clubs, total, nil
}

func (r *PostgresRepository) AddRatingHistory(ctx context.Context, history *RatingHistory) error {
	history.ID = uuid.New()
	history.CreatedAt = time.Now()

	query := `
		INSERT INTO rating_history (id, player_id, rating_type, old_rating, new_rating, tournament_id, change_date, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.Exec(ctx, query,
		history.ID, history.PlayerID, history.RatingType, history.OldRating, history.NewRating,
		history.TournamentID, history.ChangeDate, history.Source, history.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add rating history: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetRatingHistory(ctx context.Context, playerID uuid.UUID, ratingType string, limit int) ([]RatingHistory, error) {
	query := `
		SELECT id, player_id, rating_type, old_rating, new_rating, tournament_id, change_date, source, created_at
		FROM rating_history
		WHERE player_id = $1 AND ($2 = '' OR rating_type = $2)
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := r.db.Query(ctx, query, playerID, ratingType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating history: %w", err)
	}
	defer rows.Close()

	var history []RatingHistory
	for rows.Next() {
		var h RatingHistory
		err := rows.Scan(
			&h.ID, &h.PlayerID, &h.RatingType, &h.OldRating, &h.NewRating,
			&h.TournamentID, &h.ChangeDate, &h.Source, &h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rating history: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}

func (r *PostgresRepository) UpsertByFideID(ctx context.Context, player *Player) error {
	now := time.Now()
	player.UpdatedAt = now
	if player.ID == uuid.Nil {
		player.ID = uuid.New()
		player.CreatedAt = now
	}

	query := `
		INSERT INTO players (
			id, first_name, last_name, fide_id, fide_standard, fide_rapid, fide_blitz,
			fide_title, federation, ratings_updated_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (fide_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			fide_standard = EXCLUDED.fide_standard,
			fide_rapid = EXCLUDED.fide_rapid,
			fide_blitz = EXCLUDED.fide_blitz,
			fide_title = EXCLUDED.fide_title,
			federation = EXCLUDED.federation,
			ratings_updated_at = EXCLUDED.ratings_updated_at,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		player.ID, player.FirstName, player.LastName, player.FideID,
		player.FideStandard, player.FideRapid, player.FideBlitz,
		player.FideTitle, player.Federation, player.RatingsUpdatedAt,
		player.CreatedAt, player.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert player by FIDE ID: %w", err)
	}

	return nil
}

func (r *PostgresRepository) UpsertByDsbID(ctx context.Context, player *Player) error {
	now := time.Now()
	player.UpdatedAt = now
	if player.ID == uuid.Nil {
		player.ID = uuid.New()
		player.CreatedAt = now
	}

	query := `
		INSERT INTO players (
			id, first_name, last_name, dsb_id, dwz, dwz_index,
			federation, ratings_updated_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (dsb_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			dwz = EXCLUDED.dwz,
			dwz_index = EXCLUDED.dwz_index,
			federation = EXCLUDED.federation,
			ratings_updated_at = EXCLUDED.ratings_updated_at,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		player.ID, player.FirstName, player.LastName, player.DsbID,
		player.DWZ, player.DWZIndex, player.Federation,
		player.RatingsUpdatedAt, player.CreatedAt, player.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert player by DSB ID: %w", err)
	}

	return nil
}

func (r *PostgresRepository) BulkUpsertByFideID(ctx context.Context, players []*Player) error {
	if len(players) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	now := time.Now()

	for _, player := range players {
		player.UpdatedAt = now
		if player.ID == uuid.Nil {
			player.ID = uuid.New()
			player.CreatedAt = now
		}

		query := `
			INSERT INTO players (
				id, first_name, last_name, fide_id, fide_standard, fide_rapid, fide_blitz,
				fide_title, federation, ratings_updated_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (fide_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				fide_standard = EXCLUDED.fide_standard,
				fide_rapid = EXCLUDED.fide_rapid,
				fide_blitz = EXCLUDED.fide_blitz,
				fide_title = EXCLUDED.fide_title,
				federation = EXCLUDED.federation,
				ratings_updated_at = EXCLUDED.ratings_updated_at,
				updated_at = EXCLUDED.updated_at`

		batch.Queue(query,
			player.ID, player.FirstName, player.LastName, player.FideID,
			player.FideStandard, player.FideRapid, player.FideBlitz,
			player.FideTitle, player.Federation, player.RatingsUpdatedAt,
			player.CreatedAt, player.UpdatedAt,
		)
	}

	results := r.db.SendBatch(ctx, batch)
	defer results.Close()

	for range players {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch upsert: %w", err)
		}
	}

	return nil
}

func (r *PostgresRepository) UpsertByOesbID(ctx context.Context, player *Player) error {
	now := time.Now()
	player.UpdatedAt = now
	if player.ID == uuid.Nil {
		player.ID = uuid.New()
		player.CreatedAt = now
	}

	query := `
		INSERT INTO players (
			id, first_name, last_name, oesb_id, oesb_elo,
			federation, ratings_updated_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (oesb_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			oesb_elo = EXCLUDED.oesb_elo,
			federation = EXCLUDED.federation,
			ratings_updated_at = EXCLUDED.ratings_updated_at,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		player.ID, player.FirstName, player.LastName, player.OesbID,
		player.OesbElo, player.Federation,
		player.RatingsUpdatedAt, player.CreatedAt, player.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert player by OESB ID: %w", err)
	}

	return nil
}

func (r *PostgresRepository) UpsertBySsbID(ctx context.Context, player *Player) error {
	now := time.Now()
	player.UpdatedAt = now
	if player.ID == uuid.Nil {
		player.ID = uuid.New()
		player.CreatedAt = now
	}

	query := `
		INSERT INTO players (
			id, first_name, last_name, ssb_id,
			federation, ratings_updated_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (ssb_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			federation = EXCLUDED.federation,
			ratings_updated_at = EXCLUDED.ratings_updated_at,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		player.ID, player.FirstName, player.LastName, player.SsbID,
		player.Federation, player.RatingsUpdatedAt, player.CreatedAt, player.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert player by SSB ID: %w", err)
	}

	return nil
}

func (r *PostgresRepository) BulkUpsertByOesbID(ctx context.Context, players []*Player) error {
	if len(players) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	now := time.Now()

	for _, player := range players {
		player.UpdatedAt = now
		if player.ID == uuid.Nil {
			player.ID = uuid.New()
			player.CreatedAt = now
		}

		query := `
			INSERT INTO players (
				id, first_name, last_name, oesb_id, oesb_elo,
				federation, ratings_updated_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (oesb_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				oesb_elo = EXCLUDED.oesb_elo,
				federation = EXCLUDED.federation,
				ratings_updated_at = EXCLUDED.ratings_updated_at,
				updated_at = EXCLUDED.updated_at`

		batch.Queue(query,
			player.ID, player.FirstName, player.LastName, player.OesbID,
			player.OesbElo, player.Federation, player.RatingsUpdatedAt,
			player.CreatedAt, player.UpdatedAt,
		)
	}

	results := r.db.SendBatch(ctx, batch)
	defer results.Close()

	for range players {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch upsert: %w", err)
		}
	}

	return nil
}

func (r *PostgresRepository) BulkUpsertBySsbID(ctx context.Context, players []*Player) error {
	if len(players) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	now := time.Now()

	for _, player := range players {
		player.UpdatedAt = now
		if player.ID == uuid.Nil {
			player.ID = uuid.New()
			player.CreatedAt = now
		}

		query := `
			INSERT INTO players (
				id, first_name, last_name, ssb_id,
				federation, ratings_updated_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (ssb_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				federation = EXCLUDED.federation,
				ratings_updated_at = EXCLUDED.ratings_updated_at,
				updated_at = EXCLUDED.updated_at`

		batch.Queue(query,
			player.ID, player.FirstName, player.LastName, player.SsbID,
			player.Federation, player.RatingsUpdatedAt,
			player.CreatedAt, player.UpdatedAt,
		)
	}

	results := r.db.SendBatch(ctx, batch)
	defer results.Close()

	for range players {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch upsert: %w", err)
		}
	}

	return nil
}
