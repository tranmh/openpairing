package registration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpairing/openpairing/internal/tournament"
)

var (
	ErrRegistrationNotFound    = errors.New("registration not found")
	ErrAlreadyRegistered       = errors.New("already registered for this tournament")
	ErrTournamentNotFound      = errors.New("tournament not found")
	ErrInvalidInviteToken      = errors.New("invalid or expired invite token")
	ErrRegistrationClosed      = errors.New("registration is closed")
	ErrTournamentFull          = errors.New("tournament is full")
	ErrCheckInWindowClosed     = errors.New("check-in window is closed")
	ErrGuestsNotAllowed        = errors.New("guest registration is not allowed")
	ErrPreRegistrationDisabled = errors.New("pre-registration is disabled")
)

// Repository defines the registration data access interface
type Repository interface {
	// Guest registrations
	CreateGuestRegistration(ctx context.Context, reg *GuestRegistration) error
	CreateGuestRegistrationAtomic(ctx context.Context, reg *GuestRegistration, maxPlayers *int) error
	GetGuestRegistration(ctx context.Context, id uuid.UUID) (*GuestRegistration, error)
	GetGuestRegistrationByEmail(ctx context.Context, tournamentID uuid.UUID, email string) (*GuestRegistration, error)
	UpdateGuestRegistrationStatus(ctx context.Context, id uuid.UUID, status string) error
	SetGuestApproval(ctx context.Context, id uuid.UUID, approved bool, approverID uuid.UUID, reason string) error
	SetGuestCheckIn(ctx context.Context, id uuid.UUID, method string, checkedInBy *uuid.UUID) error
	ListGuestRegistrations(ctx context.Context, tournamentID uuid.UUID, status *string) ([]GuestRegistration, error)
	DeleteGuestRegistration(ctx context.Context, id uuid.UUID) error

	// Participant check-in (existing participants)
	SetParticipantCheckIn(ctx context.Context, participantID uuid.UUID, method string, checkedInBy *uuid.UUID) error
	GetParticipantByCheckInCode(ctx context.Context, code string) (*uuid.UUID, *uuid.UUID, error) // returns participantID, tournamentID

	// Tournament queries
	GetTournamentBySlug(ctx context.Context, slug string) (*TournamentPublicInfo, *tournament.RegistrationSettings, error)
	GetTournamentByID(ctx context.Context, id uuid.UUID) (*TournamentPublicInfo, *tournament.RegistrationSettings, error)
	IsParticipantRegistered(ctx context.Context, tournamentID, playerID uuid.UUID) (bool, error)
	GetRegistrationSettings(ctx context.Context, tournamentID uuid.UUID) (*tournament.RegistrationSettings, error)
	UpdateRegistrationSettings(ctx context.Context, tournamentID uuid.UUID, settings *tournament.RegistrationSettings) error

	// Check-in codes
	GenerateCheckInCode(ctx context.Context, id uuid.UUID, isGuest bool) (string, error)
	GetRegistrationByCheckInCode(ctx context.Context, code string) (*uuid.UUID, bool, error) // returns ID, isGuest

	// Invite tokens
	CreateInviteToken(ctx context.Context, token *InviteToken, rawToken string) error
	GetInviteToken(ctx context.Context, id uuid.UUID) (*InviteToken, error)
	ValidateInviteToken(ctx context.Context, tournamentID uuid.UUID, tokenStr string) (*InviteToken, error)
	IncrementInviteTokenUses(ctx context.Context, id uuid.UUID) error
	ListInviteTokens(ctx context.Context, tournamentID uuid.UUID) ([]InviteToken, error)
	DeleteInviteToken(ctx context.Context, id uuid.UUID) error

	// Combined queries
	GetAllRegistrations(ctx context.Context, tournamentID uuid.UUID) ([]RegistrationListItem, error)
	GetCheckInStatus(ctx context.Context, tournamentID uuid.UUID) (*CheckInStatus, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new registration repository
func NewRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{db: db}
}

// CreateGuestRegistration creates a new guest registration
func (r *PostgresRepository) CreateGuestRegistration(ctx context.Context, reg *GuestRegistration) error {
	reg.ID = uuid.New()
	reg.CreatedAt = time.Now()
	reg.UpdatedAt = time.Now()
	reg.PreRegisteredAt = time.Now()

	if reg.RegistrationStatus == "" {
		reg.RegistrationStatus = StatusPreRegistered
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO tournament_guest_registrations
		(id, tournament_id, email, first_name, last_name, fide_id, dsb_id, oesb_id, ssb_id,
		 player_id, registration_status, check_in_code, notes, pre_registered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, reg.ID, reg.TournamentID, reg.Email, reg.FirstName, reg.LastName,
		reg.FideID, reg.DsbID, reg.OesbID, reg.SsbID,
		reg.PlayerID, reg.RegistrationStatus, reg.CheckInCode, reg.Notes,
		reg.PreRegisteredAt, reg.CreatedAt, reg.UpdatedAt)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return ErrAlreadyRegistered
		}
		return err
	}

	return nil
}

// CreateGuestRegistrationAtomic creates a guest registration with atomic capacity check
// This prevents race conditions for both duplicates and tournament capacity
func (r *PostgresRepository) CreateGuestRegistrationAtomic(ctx context.Context, reg *GuestRegistration, maxPlayers *int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Lock the tournament row and check capacity atomically
	if maxPlayers != nil {
		var count int
		err = tx.QueryRow(ctx, `
			SELECT participant_count FROM tournaments
			WHERE id = $1 FOR UPDATE
		`, reg.TournamentID).Scan(&count)
		if err != nil {
			return err
		}
		if count >= *maxPlayers {
			return ErrTournamentFull
		}
	}

	// Create registration (unique constraint handles duplicates)
	reg.ID = uuid.New()
	reg.CreatedAt = time.Now()
	reg.UpdatedAt = time.Now()
	reg.PreRegisteredAt = time.Now()

	if reg.RegistrationStatus == "" {
		reg.RegistrationStatus = StatusPreRegistered
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tournament_guest_registrations
		(id, tournament_id, email, first_name, last_name, fide_id, dsb_id, oesb_id, ssb_id,
		 player_id, registration_status, check_in_code, notes, pre_registered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, reg.ID, reg.TournamentID, reg.Email, reg.FirstName, reg.LastName,
		reg.FideID, reg.DsbID, reg.OesbID, reg.SsbID,
		reg.PlayerID, reg.RegistrationStatus, reg.CheckInCode, reg.Notes,
		reg.PreRegisteredAt, reg.CreatedAt, reg.UpdatedAt)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return ErrAlreadyRegistered
		}
		return err
	}

	// Increment participant count
	_, err = tx.Exec(ctx, `
		UPDATE tournaments SET participant_count = participant_count + 1
		WHERE id = $1
	`, reg.TournamentID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetGuestRegistration retrieves a guest registration by ID
func (r *PostgresRepository) GetGuestRegistration(ctx context.Context, id uuid.UUID) (*GuestRegistration, error) {
	var reg GuestRegistration
	err := r.db.QueryRow(ctx, `
		SELECT id, tournament_id, email, first_name, last_name, fide_id, dsb_id, oesb_id, ssb_id,
		       player_id, registration_status, check_in_code, checked_in_at, checked_in_by, check_in_method,
		       approved_at, approved_by, rejection_reason, notes, pre_registered_at, created_at, updated_at
		FROM tournament_guest_registrations WHERE id = $1
	`, id).Scan(
		&reg.ID, &reg.TournamentID, &reg.Email, &reg.FirstName, &reg.LastName,
		&reg.FideID, &reg.DsbID, &reg.OesbID, &reg.SsbID,
		&reg.PlayerID, &reg.RegistrationStatus, &reg.CheckInCode,
		&reg.CheckedInAt, &reg.CheckedInBy, &reg.CheckInMethod,
		&reg.ApprovedAt, &reg.ApprovedBy, &reg.RejectionReason,
		&reg.Notes, &reg.PreRegisteredAt, &reg.CreatedAt, &reg.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRegistrationNotFound
	}
	if err != nil {
		return nil, err
	}

	return &reg, nil
}

// GetGuestRegistrationByEmail retrieves a guest registration by tournament and email
func (r *PostgresRepository) GetGuestRegistrationByEmail(ctx context.Context, tournamentID uuid.UUID, email string) (*GuestRegistration, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		"SELECT id FROM tournament_guest_registrations WHERE tournament_id = $1 AND email = $2",
		tournamentID, email,
	).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRegistrationNotFound
	}
	if err != nil {
		return nil, err
	}

	return r.GetGuestRegistration(ctx, id)
}

// UpdateGuestRegistrationStatus updates the status of a guest registration
func (r *PostgresRepository) UpdateGuestRegistrationStatus(ctx context.Context, id uuid.UUID, status string) error {
	result, err := r.db.Exec(ctx,
		"UPDATE tournament_guest_registrations SET registration_status = $2, updated_at = NOW() WHERE id = $1",
		id, status,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrRegistrationNotFound
	}
	return nil
}

// SetGuestApproval sets the approval status of a guest registration
func (r *PostgresRepository) SetGuestApproval(ctx context.Context, id uuid.UUID, approved bool, approverID uuid.UUID, reason string) error {
	status := StatusApproved
	if !approved {
		status = StatusRejected
	}

	var rejectionReason *string
	if !approved && reason != "" {
		rejectionReason = &reason
	}

	result, err := r.db.Exec(ctx, `
		UPDATE tournament_guest_registrations
		SET registration_status = $2, approved_at = NOW(), approved_by = $3, rejection_reason = $4, updated_at = NOW()
		WHERE id = $1
	`, id, status, approverID, rejectionReason)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrRegistrationNotFound
	}
	return nil
}

// SetGuestCheckIn marks a guest as checked in
func (r *PostgresRepository) SetGuestCheckIn(ctx context.Context, id uuid.UUID, method string, checkedInBy *uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		UPDATE tournament_guest_registrations
		SET registration_status = $2, checked_in_at = NOW(), check_in_method = $3, checked_in_by = $4, updated_at = NOW()
		WHERE id = $1
	`, id, StatusCheckedIn, method, checkedInBy)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrRegistrationNotFound
	}
	return nil
}

// ListGuestRegistrations lists all guest registrations for a tournament
func (r *PostgresRepository) ListGuestRegistrations(ctx context.Context, tournamentID uuid.UUID, status *string) ([]GuestRegistration, error) {
	// Validate status to prevent injection
	if status != nil {
		validStatuses := map[string]bool{
			StatusPreRegistered:   true,
			StatusPendingApproval: true,
			StatusApproved:        true,
			StatusCheckedIn:       true,
			StatusConverted:       true,
			StatusRejected:        true,
			StatusExpired:         true,
			StatusNoShow:          true,
		}
		if !validStatuses[*status] {
			return nil, errors.New("invalid status filter")
		}
	}

	query := `
		SELECT id, tournament_id, email, first_name, last_name, fide_id, dsb_id, oesb_id, ssb_id,
		       player_id, registration_status, check_in_code, checked_in_at, checked_in_by, check_in_method,
		       approved_at, approved_by, rejection_reason, notes, pre_registered_at, created_at, updated_at
		FROM tournament_guest_registrations
		WHERE tournament_id = $1`

	var args []interface{}
	args = append(args, tournamentID)

	if status != nil {
		query += " AND registration_status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY pre_registered_at ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var registrations []GuestRegistration
	for rows.Next() {
		var reg GuestRegistration
		err := rows.Scan(
			&reg.ID, &reg.TournamentID, &reg.Email, &reg.FirstName, &reg.LastName,
			&reg.FideID, &reg.DsbID, &reg.OesbID, &reg.SsbID,
			&reg.PlayerID, &reg.RegistrationStatus, &reg.CheckInCode,
			&reg.CheckedInAt, &reg.CheckedInBy, &reg.CheckInMethod,
			&reg.ApprovedAt, &reg.ApprovedBy, &reg.RejectionReason,
			&reg.Notes, &reg.PreRegisteredAt, &reg.CreatedAt, &reg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, reg)
	}

	return registrations, nil
}

// DeleteGuestRegistration deletes a guest registration and decrements participant count
func (r *PostgresRepository) DeleteGuestRegistration(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get tournament ID before deletion
	var tournamentID uuid.UUID
	err = tx.QueryRow(ctx,
		"SELECT tournament_id FROM tournament_guest_registrations WHERE id = $1",
		id,
	).Scan(&tournamentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRegistrationNotFound
	}
	if err != nil {
		return err
	}

	// Delete the registration
	result, err := tx.Exec(ctx, "DELETE FROM tournament_guest_registrations WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrRegistrationNotFound
	}

	// Decrement participant count (ensure it doesn't go below 0)
	_, err = tx.Exec(ctx,
		"UPDATE tournaments SET participant_count = GREATEST(participant_count - 1, 0) WHERE id = $1",
		tournamentID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SetParticipantCheckIn marks a participant as checked in
func (r *PostgresRepository) SetParticipantCheckIn(ctx context.Context, participantID uuid.UUID, method string, checkedInBy *uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		UPDATE tournament_participants
		SET checked_in_at = NOW(), check_in_method = $2, checked_in_by = $3
		WHERE id = $1
	`, participantID, method, checkedInBy)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrRegistrationNotFound
	}
	return nil
}

// GetParticipantByCheckInCode retrieves participant by check-in code
func (r *PostgresRepository) GetParticipantByCheckInCode(ctx context.Context, code string) (*uuid.UUID, *uuid.UUID, error) {
	var participantID, tournamentID uuid.UUID
	err := r.db.QueryRow(ctx,
		"SELECT id, tournament_id FROM tournament_participants WHERE check_in_code = $1",
		code,
	).Scan(&participantID, &tournamentID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrRegistrationNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	return &participantID, &tournamentID, nil
}

// GetTournamentBySlug retrieves public tournament info by slug
func (r *PostgresRepository) GetTournamentBySlug(ctx context.Context, slug string) (*TournamentPublicInfo, *tournament.RegistrationSettings, error) {
	var info TournamentPublicInfo
	var settingsJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, slug, start_date, end_date, venue_name, venue_city, venue_country,
		       max_players, participant_count, status, registration_opens, registration_closes,
		       entry_fee_cents, currency, registration_settings
		FROM tournaments WHERE slug = $1
	`, slug).Scan(
		&info.ID, &info.Name, &info.Description, &info.Slug, &info.StartDate, &info.EndDate,
		&info.VenueName, &info.VenueCity, &info.VenueCountry,
		&info.MaxPlayers, &info.ParticipantCount, &info.Status,
		&info.RegistrationOpens, &info.RegistrationCloses,
		&info.EntryFeeCents, &info.Currency, &settingsJSON,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrTournamentNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	// Parse registration settings
	settings := tournament.DefaultRegistrationSettings()
	if len(settingsJSON) > 0 && string(settingsJSON) != "{}" {
		if err := json.Unmarshal(settingsJSON, settings); err != nil {
			// Use defaults if parsing fails
			settings = tournament.DefaultRegistrationSettings()
		}
	}

	return &info, settings, nil
}

// GetTournamentByID retrieves public tournament info by ID
func (r *PostgresRepository) GetTournamentByID(ctx context.Context, id uuid.UUID) (*TournamentPublicInfo, *tournament.RegistrationSettings, error) {
	var info TournamentPublicInfo
	var settingsJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, slug, start_date, end_date, venue_name, venue_city, venue_country,
		       max_players, participant_count, status, registration_opens, registration_closes,
		       entry_fee_cents, currency, registration_settings
		FROM tournaments WHERE id = $1
	`, id).Scan(
		&info.ID, &info.Name, &info.Description, &info.Slug, &info.StartDate, &info.EndDate,
		&info.VenueName, &info.VenueCity, &info.VenueCountry,
		&info.MaxPlayers, &info.ParticipantCount, &info.Status,
		&info.RegistrationOpens, &info.RegistrationCloses,
		&info.EntryFeeCents, &info.Currency, &settingsJSON,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrTournamentNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	// Parse registration settings
	settings := tournament.DefaultRegistrationSettings()
	if len(settingsJSON) > 0 && string(settingsJSON) != "{}" {
		if err := json.Unmarshal(settingsJSON, settings); err != nil {
			settings = tournament.DefaultRegistrationSettings()
		}
	}

	return &info, settings, nil
}

// IsParticipantRegistered checks if a player is registered
func (r *PostgresRepository) IsParticipantRegistered(ctx context.Context, tournamentID, playerID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = $1 AND player_id = $2",
		tournamentID, playerID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetRegistrationSettings retrieves registration settings for a tournament
func (r *PostgresRepository) GetRegistrationSettings(ctx context.Context, tournamentID uuid.UUID) (*tournament.RegistrationSettings, error) {
	var settingsJSON []byte
	err := r.db.QueryRow(ctx, "SELECT registration_settings FROM tournaments WHERE id = $1", tournamentID).Scan(&settingsJSON)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTournamentNotFound
	}
	if err != nil {
		return nil, err
	}

	settings := tournament.DefaultRegistrationSettings()
	if len(settingsJSON) > 0 && string(settingsJSON) != "{}" {
		if err := json.Unmarshal(settingsJSON, settings); err != nil {
			return nil, err
		}
	}

	return settings, nil
}

// UpdateRegistrationSettings updates registration settings for a tournament
func (r *PostgresRepository) UpdateRegistrationSettings(ctx context.Context, tournamentID uuid.UUID, settings *tournament.RegistrationSettings) error {
	result, err := r.db.Exec(ctx,
		"UPDATE tournaments SET registration_settings = $2, updated_at = NOW() WHERE id = $1",
		tournamentID, settings,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTournamentNotFound
	}
	return nil
}

// GenerateCheckInCode generates a unique check-in code
func (r *PostgresRepository) GenerateCheckInCode(ctx context.Context, id uuid.UUID, isGuest bool) (string, error) {
	// Generate a short, readable code
	code := generateShortCode(8)

	var err error
	if isGuest {
		_, err = r.db.Exec(ctx,
			"UPDATE tournament_guest_registrations SET check_in_code = $2, updated_at = NOW() WHERE id = $1",
			id, code,
		)
	} else {
		_, err = r.db.Exec(ctx,
			"UPDATE tournament_participants SET check_in_code = $2 WHERE id = $1",
			id, code,
		)
	}

	if err != nil {
		return "", err
	}
	return code, nil
}

// GetRegistrationByCheckInCode finds a registration by check-in code
func (r *PostgresRepository) GetRegistrationByCheckInCode(ctx context.Context, code string) (*uuid.UUID, bool, error) {
	// Try guest registrations first
	var guestID uuid.UUID
	err := r.db.QueryRow(ctx,
		"SELECT id FROM tournament_guest_registrations WHERE check_in_code = $1",
		code,
	).Scan(&guestID)

	if err == nil {
		return &guestID, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	// Try participants
	var participantID uuid.UUID
	err = r.db.QueryRow(ctx,
		"SELECT id FROM tournament_participants WHERE check_in_code = $1",
		code,
	).Scan(&participantID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, ErrRegistrationNotFound
	}
	if err != nil {
		return nil, false, err
	}

	return &participantID, false, nil
}

// CreateInviteToken creates a new invite token
func (r *PostgresRepository) CreateInviteToken(ctx context.Context, token *InviteToken, rawToken string) error {
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.TokenHash = hashToken(rawToken)

	_, err := r.db.Exec(ctx, `
		INSERT INTO tournament_invite_tokens (id, tournament_id, token_hash, name, max_uses, valid_until, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, token.ID, token.TournamentID, token.TokenHash, token.Name, token.MaxUses, token.ValidUntil, token.CreatedBy, token.CreatedAt)

	return err
}

// GetInviteToken retrieves an invite token by ID
func (r *PostgresRepository) GetInviteToken(ctx context.Context, id uuid.UUID) (*InviteToken, error) {
	var token InviteToken
	err := r.db.QueryRow(ctx, `
		SELECT id, tournament_id, token_hash, name, max_uses, uses_count, valid_until, created_by, created_at
		FROM tournament_invite_tokens WHERE id = $1
	`, id).Scan(
		&token.ID, &token.TournamentID, &token.TokenHash, &token.Name,
		&token.MaxUses, &token.UsesCount, &token.ValidUntil, &token.CreatedBy, &token.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidInviteToken
	}
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// ValidateInviteToken validates an invite token
func (r *PostgresRepository) ValidateInviteToken(ctx context.Context, tournamentID uuid.UUID, tokenStr string) (*InviteToken, error) {
	tokenHash := hashToken(tokenStr)

	var token InviteToken
	err := r.db.QueryRow(ctx, `
		SELECT id, tournament_id, token_hash, name, max_uses, uses_count, valid_until, created_by, created_at
		FROM tournament_invite_tokens
		WHERE tournament_id = $1 AND token_hash = $2
	`, tournamentID, tokenHash).Scan(
		&token.ID, &token.TournamentID, &token.TokenHash, &token.Name,
		&token.MaxUses, &token.UsesCount, &token.ValidUntil, &token.CreatedBy, &token.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidInviteToken
	}
	if err != nil {
		return nil, err
	}

	// Check if expired
	if token.ValidUntil != nil && time.Now().After(*token.ValidUntil) {
		return nil, ErrInvalidInviteToken
	}

	// Check if max uses reached
	if token.MaxUses != nil && token.UsesCount >= *token.MaxUses {
		return nil, ErrInvalidInviteToken
	}

	return &token, nil
}

// IncrementInviteTokenUses atomically increments the uses count, failing if max uses reached
func (r *PostgresRepository) IncrementInviteTokenUses(ctx context.Context, id uuid.UUID) error {
	// Atomic increment with max uses check - if no rows updated, max uses was reached
	result, err := r.db.Exec(ctx, `
		UPDATE tournament_invite_tokens
		SET uses_count = uses_count + 1
		WHERE id = $1
		AND (max_uses IS NULL OR uses_count < max_uses)
	`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInvalidInviteToken // Max uses reached or token not found
	}
	return nil
}

// ListInviteTokens lists all invite tokens for a tournament
func (r *PostgresRepository) ListInviteTokens(ctx context.Context, tournamentID uuid.UUID) ([]InviteToken, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tournament_id, token_hash, name, max_uses, uses_count, valid_until, created_by, created_at
		FROM tournament_invite_tokens WHERE tournament_id = $1 ORDER BY created_at DESC
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []InviteToken
	for rows.Next() {
		var token InviteToken
		err := rows.Scan(
			&token.ID, &token.TournamentID, &token.TokenHash, &token.Name,
			&token.MaxUses, &token.UsesCount, &token.ValidUntil, &token.CreatedBy, &token.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// DeleteInviteToken deletes an invite token
func (r *PostgresRepository) DeleteInviteToken(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, "DELETE FROM tournament_invite_tokens WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInvalidInviteToken
	}
	return nil
}

// GetAllRegistrations returns all registrations (guests and participants) for a tournament
func (r *PostgresRepository) GetAllRegistrations(ctx context.Context, tournamentID uuid.UUID) ([]RegistrationListItem, error) {
	// Query guests
	guestQuery := `
		SELECT id, 'guest' as type, email, first_name, last_name, fide_id,
		       registration_status, pre_registered_at, approved_at, checked_in_at, check_in_method, '' as registration_source
		FROM tournament_guest_registrations
		WHERE tournament_id = $1`

	// Query participants with pre-registration info
	participantQuery := `
		SELECT tp.id, 'participant' as type, '' as email,
		       COALESCE(p.first_name, '') as first_name, COALESCE(p.last_name, '') as last_name,
		       p.fide_id, tp.registration_status, tp.pre_registered_at, tp.approved_at,
		       tp.checked_in_at, tp.check_in_method, COALESCE(tp.registration_source, 'organizer') as registration_source
		FROM tournament_participants tp
		JOIN players p ON tp.player_id = p.id
		WHERE tp.tournament_id = $1`

	// Combine with UNION
	query := guestQuery + " UNION ALL " + participantQuery + " ORDER BY type, first_name, last_name"

	rows, err := r.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RegistrationListItem
	for rows.Next() {
		var item RegistrationListItem
		err := rows.Scan(
			&item.ID, &item.Type, &item.Email, &item.FirstName, &item.LastName,
			&item.FideID, &item.RegistrationStatus, &item.PreRegisteredAt,
			&item.ApprovedAt, &item.CheckedInAt, &item.CheckInMethod, &item.RegistrationSource,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// GetCheckInStatus returns check-in statistics for a tournament
func (r *PostgresRepository) GetCheckInStatus(ctx context.Context, tournamentID uuid.UUID) (*CheckInStatus, error) {
	var status CheckInStatus

	// Count from participants
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN checked_in_at IS NOT NULL THEN 1 END) as checked_in,
			COUNT(CASE WHEN checked_in_at IS NULL THEN 1 END) as not_checked_in
		FROM tournament_participants
		WHERE tournament_id = $1 AND is_active = true
	`, tournamentID).Scan(&status.TotalRegistrations, &status.CheckedIn, &status.NotCheckedIn)

	if err != nil {
		return nil, err
	}

	// Count guests pending approval
	err = r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM tournament_guest_registrations WHERE tournament_id = $1 AND registration_status IN ($2, $3)",
		tournamentID, StatusPreRegistered, StatusPendingApproval,
	).Scan(&status.PendingApproval)

	if err != nil {
		return nil, err
	}

	return &status, nil
}

// Helper functions

func generateShortCode(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Excluding confusing chars I, O, 0, 1
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GenerateInviteToken generates a new random invite token
func GenerateInviteToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
