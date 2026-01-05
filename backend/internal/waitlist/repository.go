package waitlist

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEntryNotFound     = errors.New("waitlist entry not found")
	ErrAlreadyOnWaitlist = errors.New("player is already on the waitlist")
	ErrAlreadyRegistered = errors.New("player is already registered for this tournament")
)

// Repository defines the waitlist data access interface
type Repository interface {
	Add(ctx context.Context, entry *WaitlistEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*WaitlistEntry, error)
	GetByTournamentAndPlayer(ctx context.Context, tournamentID, playerID uuid.UUID) (*WaitlistEntry, error)
	List(ctx context.Context, tournamentID uuid.UUID) ([]WaitlistEntry, error)
	Remove(ctx context.Context, id uuid.UUID) error
	UpdatePosition(ctx context.Context, id uuid.UUID, position int) error
	SetNotified(ctx context.Context, id uuid.UUID, expiresAt time.Time) error
	GetNextInQueue(ctx context.Context, tournamentID uuid.UUID) (*WaitlistEntry, error)
	GetExpiredEntries(ctx context.Context) ([]WaitlistEntry, error)
	GetTournamentInfo(ctx context.Context, tournamentID uuid.UUID) (*TournamentInfo, error)
	IsPlayerRegistered(ctx context.Context, tournamentID, playerID uuid.UUID) (bool, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new waitlist repository
func NewRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{db: db}
}

// Add adds a new waitlist entry
func (r *PostgresRepository) Add(ctx context.Context, entry *WaitlistEntry) error {
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()

	// Get the next position
	var maxPosition int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) FROM tournament_waitlist WHERE tournament_id = $1
	`, entry.TournamentID).Scan(&maxPosition)
	if err != nil {
		return err
	}
	entry.Position = maxPosition + 1

	_, err = r.db.Exec(ctx, `
		INSERT INTO tournament_waitlist (id, tournament_id, player_id, position, starting_rating, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, entry.ID, entry.TournamentID, entry.PlayerID, entry.Position, entry.StartingRating, entry.Notes, entry.CreatedAt)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return ErrAlreadyOnWaitlist
		}
		return err
	}

	return nil
}

// GetByID retrieves a waitlist entry by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*WaitlistEntry, error) {
	query := `
		SELECT w.id, w.tournament_id, w.player_id, w.position, w.starting_rating, w.notes,
		       w.notified_at, w.expires_at, w.created_at,
		       p.first_name, p.last_name, p.fide_id, c.name as club_name
		FROM tournament_waitlist w
		JOIN players p ON w.player_id = p.id
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE w.id = $1`

	var entry WaitlistEntry
	err := r.db.QueryRow(ctx, query, id).Scan(
		&entry.ID, &entry.TournamentID, &entry.PlayerID, &entry.Position, &entry.StartingRating, &entry.Notes,
		&entry.NotifiedAt, &entry.ExpiresAt, &entry.CreatedAt,
		&entry.PlayerFirstName, &entry.PlayerLastName, &entry.PlayerFideID, &entry.PlayerClubName,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEntryNotFound
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetByTournamentAndPlayer retrieves a waitlist entry by tournament and player
func (r *PostgresRepository) GetByTournamentAndPlayer(ctx context.Context, tournamentID, playerID uuid.UUID) (*WaitlistEntry, error) {
	query := `SELECT id FROM tournament_waitlist WHERE tournament_id = $1 AND player_id = $2`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, tournamentID, playerID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEntryNotFound
	}
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// List retrieves all waitlist entries for a tournament
func (r *PostgresRepository) List(ctx context.Context, tournamentID uuid.UUID) ([]WaitlistEntry, error) {
	query := `
		SELECT w.id, w.tournament_id, w.player_id, w.position, w.starting_rating, w.notes,
		       w.notified_at, w.expires_at, w.created_at,
		       p.first_name, p.last_name, p.fide_id, c.name as club_name
		FROM tournament_waitlist w
		JOIN players p ON w.player_id = p.id
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE w.tournament_id = $1
		ORDER BY w.position ASC`

	rows, err := r.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []WaitlistEntry
	for rows.Next() {
		var entry WaitlistEntry
		err := rows.Scan(
			&entry.ID, &entry.TournamentID, &entry.PlayerID, &entry.Position, &entry.StartingRating, &entry.Notes,
			&entry.NotifiedAt, &entry.ExpiresAt, &entry.CreatedAt,
			&entry.PlayerFirstName, &entry.PlayerLastName, &entry.PlayerFideID, &entry.PlayerClubName,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Remove removes a waitlist entry and reorders positions
func (r *PostgresRepository) Remove(ctx context.Context, id uuid.UUID) error {
	// Get entry to know position and tournament
	entry, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete the entry
	_, err = tx.Exec(ctx, "DELETE FROM tournament_waitlist WHERE id = $1", id)
	if err != nil {
		return err
	}

	// Reorder positions
	_, err = tx.Exec(ctx, `
		UPDATE tournament_waitlist
		SET position = position - 1
		WHERE tournament_id = $1 AND position > $2
	`, entry.TournamentID, entry.Position)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdatePosition updates the position of a waitlist entry
func (r *PostgresRepository) UpdatePosition(ctx context.Context, id uuid.UUID, position int) error {
	result, err := r.db.Exec(ctx, "UPDATE tournament_waitlist SET position = $2 WHERE id = $1", id, position)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrEntryNotFound
	}
	return nil
}

// SetNotified marks an entry as notified with an expiration time
func (r *PostgresRepository) SetNotified(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE tournament_waitlist SET notified_at = NOW(), expires_at = $2 WHERE id = $1
	`, id, expiresAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrEntryNotFound
	}
	return nil
}

// GetNextInQueue gets the next unnotified entry in the queue
func (r *PostgresRepository) GetNextInQueue(ctx context.Context, tournamentID uuid.UUID) (*WaitlistEntry, error) {
	query := `
		SELECT id FROM tournament_waitlist
		WHERE tournament_id = $1 AND notified_at IS NULL
		ORDER BY position ASC
		LIMIT 1`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, tournamentID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No entries in queue
	}
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// GetExpiredEntries gets all entries that have expired their confirmation window
func (r *PostgresRepository) GetExpiredEntries(ctx context.Context) ([]WaitlistEntry, error) {
	query := `
		SELECT w.id, w.tournament_id, w.player_id, w.position, w.starting_rating, w.notes,
		       w.notified_at, w.expires_at, w.created_at,
		       p.first_name, p.last_name, p.fide_id, c.name as club_name
		FROM tournament_waitlist w
		JOIN players p ON w.player_id = p.id
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE w.expires_at IS NOT NULL AND w.expires_at < NOW()
		ORDER BY w.tournament_id, w.position`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []WaitlistEntry
	for rows.Next() {
		var entry WaitlistEntry
		err := rows.Scan(
			&entry.ID, &entry.TournamentID, &entry.PlayerID, &entry.Position, &entry.StartingRating, &entry.Notes,
			&entry.NotifiedAt, &entry.ExpiresAt, &entry.CreatedAt,
			&entry.PlayerFirstName, &entry.PlayerLastName, &entry.PlayerFideID, &entry.PlayerClubName,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetTournamentInfo retrieves basic tournament information
func (r *PostgresRepository) GetTournamentInfo(ctx context.Context, tournamentID uuid.UUID) (*TournamentInfo, error) {
	query := `
		SELECT t.id, t.name, t.max_players, COALESCE(t.participant_count, 0) as capacity
		FROM tournaments t
		WHERE t.id = $1`

	var info TournamentInfo
	err := r.db.QueryRow(ctx, query, tournamentID).Scan(
		&info.ID, &info.Name, &info.MaxPlayers, &info.Capacity,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("tournament not found")
	}
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// IsPlayerRegistered checks if a player is already registered for a tournament
func (r *PostgresRepository) IsPlayerRegistered(ctx context.Context, tournamentID, playerID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = $1 AND player_id = $2
	`, tournamentID, playerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
