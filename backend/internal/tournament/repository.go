package tournament

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTournamentNotFound   = errors.New("tournament not found")
	ErrParticipantNotFound  = errors.New("participant not found")
	ErrRoundNotFound        = errors.New("round not found")
	ErrGameNotFound         = errors.New("game not found")
	ErrDuplicateRegistration = errors.New("player already registered")
	ErrTournamentFull       = errors.New("tournament is full")
	ErrRegistrationClosed   = errors.New("registration is closed")
	ErrInvalidStatus        = errors.New("invalid tournament status for this operation")
)

type Repository interface {
	// Tournament CRUD
	Create(ctx context.Context, t *Tournament) error
	GetByID(ctx context.Context, id uuid.UUID) (*Tournament, error)
	GetBySlug(ctx context.Context, slug string) (*Tournament, error)
	Update(ctx context.Context, t *Tournament) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filters *ListFilters) ([]Tournament, int64, error)

	// Participants
	AddParticipant(ctx context.Context, p *Participant) error
	GetParticipant(ctx context.Context, id uuid.UUID) (*Participant, error)
	GetParticipantByPlayerID(ctx context.Context, tournamentID, playerID uuid.UUID) (*Participant, error)
	UpdateParticipant(ctx context.Context, p *Participant) error
	RemoveParticipant(ctx context.Context, id uuid.UUID) error
	ListParticipants(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error)
	GetStandings(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error)

	// Rounds
	CreateRound(ctx context.Context, r *Round) error
	GetRound(ctx context.Context, id uuid.UUID) (*Round, error)
	GetRoundByNumber(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*Round, error)
	UpdateRound(ctx context.Context, r *Round) error
	ListRounds(ctx context.Context, tournamentID uuid.UUID) ([]Round, error)

	// Games
	CreateGame(ctx context.Context, g *Game) error
	CreateGames(ctx context.Context, games []Game) error
	GetGame(ctx context.Context, id uuid.UUID) (*Game, error)
	UpdateGame(ctx context.Context, g *Game) error
	ListGamesByRound(ctx context.Context, roundID uuid.UUID) ([]Game, error)
	ListGamesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Game, error)
	GetPlayerGames(ctx context.Context, tournamentID, participantID uuid.UUID) ([]Game, error)

	// Forbidden pairs
	AddForbiddenPair(ctx context.Context, fp *ForbiddenPair) error
	ListForbiddenPairs(ctx context.Context, tournamentID uuid.UUID) ([]ForbiddenPair, error)
	RemoveForbiddenPair(ctx context.Context, id uuid.UUID) error
}

type ListFilters struct {
	Status         *TournamentStatus
	Type           *TournamentType
	OrganizationID *uuid.UUID
	StartDateFrom  *string
	StartDateTo    *string
	Page           int
	Limit          int
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{db: db}
}

// Tournament operations

func (r *PostgresRepository) Create(ctx context.Context, t *Tournament) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	if t.Status == "" {
		t.Status = StatusDraft
	}
	if t.MinPlayers == 0 {
		t.MinPlayers = 4
	}
	if t.Currency == "" {
		t.Currency = "EUR"
	}
	if t.PairingEngine == "" {
		t.PairingEngine = EngineJaVaFo
	}
	if t.PairingSystem == "" {
		t.PairingSystem = PairingDutch
	}
	if len(t.Tiebreaks) == 0 {
		t.Tiebreaks = DefaultTiebreaks()
	}

	// Generate slug from name
	t.Slug = generateSlug(t.Name)

	tiebreaksJSON, _ := json.Marshal(t.Tiebreaks)
	settingsJSON, _ := json.Marshal(t.Settings)

	query := `
		INSERT INTO tournaments (
			id, organization_id, name, description, slug,
			type, mode, time_control, time_control_type,
			pairing_engine, pairing_system, accelerated_pairing,
			start_date, end_date, registration_opens, registration_closes,
			max_players, min_players, rounds,
			venue_name, venue_address, venue_city, venue_country,
			fide_rated, dwz_rated, oesb_rated,
			entry_fee_cents, currency, status,
			tiebreaks, settings, current_round, participant_count,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19,
			$20, $21, $22, $23,
			$24, $25, $26,
			$27, $28, $29,
			$30, $31, $32, $33,
			$34, $35
		)`

	_, err := r.db.Exec(ctx, query,
		t.ID, t.OrganizationID, t.Name, t.Description, t.Slug,
		t.Type, t.Mode, t.TimeControl, t.TimeControlType,
		t.PairingEngine, t.PairingSystem, t.AcceleratedPairing,
		t.StartDate, t.EndDate, t.RegistrationOpens, t.RegistrationCloses,
		t.MaxPlayers, t.MinPlayers, t.Rounds,
		t.VenueName, t.VenueAddress, t.VenueCity, t.VenueCountry,
		t.FideRated, t.DwzRated, t.OesbRated,
		t.EntryFeeCents, t.Currency, t.Status,
		tiebreaksJSON, settingsJSON, t.CurrentRound, t.ParticipantCount,
		t.CreatedAt, t.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create tournament: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	query := `
		SELECT
			id, organization_id, name, description, slug,
			type, mode, time_control, time_control_type,
			pairing_engine, pairing_system, accelerated_pairing,
			start_date, end_date, registration_opens, registration_closes,
			max_players, min_players, rounds,
			venue_name, venue_address, venue_city, venue_country,
			fide_rated, dwz_rated, oesb_rated,
			entry_fee_cents, currency, status,
			tiebreaks, settings, current_round, participant_count,
			created_at, updated_at
		FROM tournaments WHERE id = $1`

	var t Tournament
	var tiebreaksJSON, settingsJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.OrganizationID, &t.Name, &t.Description, &t.Slug,
		&t.Type, &t.Mode, &t.TimeControl, &t.TimeControlType,
		&t.PairingEngine, &t.PairingSystem, &t.AcceleratedPairing,
		&t.StartDate, &t.EndDate, &t.RegistrationOpens, &t.RegistrationCloses,
		&t.MaxPlayers, &t.MinPlayers, &t.Rounds,
		&t.VenueName, &t.VenueAddress, &t.VenueCity, &t.VenueCountry,
		&t.FideRated, &t.DwzRated, &t.OesbRated,
		&t.EntryFeeCents, &t.Currency, &t.Status,
		&tiebreaksJSON, &settingsJSON, &t.CurrentRound, &t.ParticipantCount,
		&t.CreatedAt, &t.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTournamentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament: %w", err)
	}

	json.Unmarshal(tiebreaksJSON, &t.Tiebreaks)
	json.Unmarshal(settingsJSON, &t.Settings)

	return &t, nil
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (*Tournament, error) {
	query := `SELECT id FROM tournaments WHERE slug = $1`
	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTournamentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament by slug: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresRepository) Update(ctx context.Context, t *Tournament) error {
	t.UpdatedAt = time.Now()

	tiebreaksJSON, _ := json.Marshal(t.Tiebreaks)
	settingsJSON, _ := json.Marshal(t.Settings)

	query := `
		UPDATE tournaments SET
			name = $2, description = $3,
			time_control = $4, time_control_type = $5,
			pairing_engine = $6, accelerated_pairing = $7,
			start_date = $8, end_date = $9,
			registration_opens = $10, registration_closes = $11,
			max_players = $12, rounds = $13,
			venue_name = $14, venue_address = $15, venue_city = $16,
			fide_rated = $17, dwz_rated = $18, oesb_rated = $19,
			entry_fee_cents = $20, status = $21,
			tiebreaks = $22, settings = $23,
			current_round = $24, participant_count = $25,
			updated_at = $26
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query,
		t.ID, t.Name, t.Description,
		t.TimeControl, t.TimeControlType,
		t.PairingEngine, t.AcceleratedPairing,
		t.StartDate, t.EndDate,
		t.RegistrationOpens, t.RegistrationCloses,
		t.MaxPlayers, t.Rounds,
		t.VenueName, t.VenueAddress, t.VenueCity,
		t.FideRated, t.DwzRated, t.OesbRated,
		t.EntryFeeCents, t.Status,
		tiebreaksJSON, settingsJSON,
		t.CurrentRound, t.ParticipantCount,
		t.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update tournament: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTournamentNotFound
	}

	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, "DELETE FROM tournaments WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete tournament: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTournamentNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, filters *ListFilters) ([]Tournament, int64, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filters.Status)
		argIndex++
	}

	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *filters.Type)
		argIndex++
	}

	if filters.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argIndex))
		args = append(args, *filters.OrganizationID)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) FROM tournaments" + whereClause
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tournaments: %w", err)
	}

	// Get page
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.Limit <= 0 {
		filters.Limit = 20
	}

	offset := (filters.Page - 1) * filters.Limit
	args = append(args, filters.Limit, offset)

	query := fmt.Sprintf(`
		SELECT
			id, organization_id, name, description, slug,
			type, mode, time_control, time_control_type,
			pairing_engine, pairing_system, accelerated_pairing,
			start_date, end_date, registration_opens, registration_closes,
			max_players, min_players, rounds,
			venue_name, venue_address, venue_city, venue_country,
			fide_rated, dwz_rated, oesb_rated,
			entry_fee_cents, currency, status,
			tiebreaks, settings, current_round, participant_count,
			created_at, updated_at
		FROM tournaments
		%s
		ORDER BY start_date DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tournaments: %w", err)
	}
	defer rows.Close()

	var tournaments []Tournament
	for rows.Next() {
		var t Tournament
		var tiebreaksJSON, settingsJSON []byte

		err := rows.Scan(
			&t.ID, &t.OrganizationID, &t.Name, &t.Description, &t.Slug,
			&t.Type, &t.Mode, &t.TimeControl, &t.TimeControlType,
			&t.PairingEngine, &t.PairingSystem, &t.AcceleratedPairing,
			&t.StartDate, &t.EndDate, &t.RegistrationOpens, &t.RegistrationCloses,
			&t.MaxPlayers, &t.MinPlayers, &t.Rounds,
			&t.VenueName, &t.VenueAddress, &t.VenueCity, &t.VenueCountry,
			&t.FideRated, &t.DwzRated, &t.OesbRated,
			&t.EntryFeeCents, &t.Currency, &t.Status,
			&tiebreaksJSON, &settingsJSON, &t.CurrentRound, &t.ParticipantCount,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan tournament: %w", err)
		}

		json.Unmarshal(tiebreaksJSON, &t.Tiebreaks)
		json.Unmarshal(settingsJSON, &t.Settings)

		tournaments = append(tournaments, t)
	}

	return tournaments, total, nil
}

// Participant operations

func (r *PostgresRepository) AddParticipant(ctx context.Context, p *Participant) error {
	p.ID = uuid.New()
	p.RegisteredAt = time.Now()
	if p.RegistrationStatus == "" {
		p.RegistrationStatus = "registered"
	}
	if p.PaymentStatus == "" {
		p.PaymentStatus = "pending"
	}
	p.IsActive = true

	byeRequestsJSON, _ := json.Marshal(p.ByeRequests)

	query := `
		INSERT INTO tournament_participants (
			id, tournament_id, section_id, player_id,
			registration_status, registered_at,
			start_rank, starting_rating,
			points, buchholz, buchholz_cut1, sonneborn_berger,
			wins, draws, losses, is_active, bye_requests,
			payment_status, payment_reference
		) VALUES (
			$1, $2, $3, $4,
			$5, $6,
			$7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16, $17,
			$18, $19
		)`

	_, err := r.db.Exec(ctx, query,
		p.ID, p.TournamentID, p.SectionID, p.PlayerID,
		p.RegistrationStatus, p.RegisteredAt,
		p.StartRank, p.StartingRating,
		p.Points, p.Buchholz, p.BuchholzCut1, p.SonnebornBerger,
		p.Wins, p.Draws, p.Losses, p.IsActive, byeRequestsJSON,
		p.PaymentStatus, p.PaymentReference,
	)

	if err != nil {
		if strings.Contains(err.Error(), "tournament_participants_tournament_id_player_id_key") {
			return ErrDuplicateRegistration
		}
		return fmt.Errorf("failed to add participant: %w", err)
	}

	// Update participant count
	_, err = r.db.Exec(ctx, `
		UPDATE tournaments SET participant_count = participant_count + 1 WHERE id = $1
	`, p.TournamentID)

	return err
}

func (r *PostgresRepository) GetParticipant(ctx context.Context, id uuid.UUID) (*Participant, error) {
	query := `
		SELECT
			tp.id, tp.tournament_id, tp.section_id, tp.player_id,
			tp.registration_status, tp.registered_at,
			tp.start_rank, tp.starting_rating,
			tp.points, tp.buchholz, tp.buchholz_cut1, tp.sonneborn_berger,
			tp.wins, tp.draws, tp.losses, tp.is_active, tp.bye_requests,
			tp.payment_status, tp.payment_reference,
			p.first_name, p.last_name, p.fide_id
		FROM tournament_participants tp
		JOIN players p ON tp.player_id = p.id
		WHERE tp.id = $1`

	var participant Participant
	var byeRequestsJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&participant.ID, &participant.TournamentID, &participant.SectionID, &participant.PlayerID,
		&participant.RegistrationStatus, &participant.RegisteredAt,
		&participant.StartRank, &participant.StartingRating,
		&participant.Points, &participant.Buchholz, &participant.BuchholzCut1, &participant.SonnebornBerger,
		&participant.Wins, &participant.Draws, &participant.Losses, &participant.IsActive, &byeRequestsJSON,
		&participant.PaymentStatus, &participant.PaymentReference,
		&participant.PlayerFirstName, &participant.PlayerLastName, &participant.PlayerFideID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrParticipantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get participant: %w", err)
	}

	json.Unmarshal(byeRequestsJSON, &participant.ByeRequests)

	return &participant, nil
}

func (r *PostgresRepository) GetParticipantByPlayerID(ctx context.Context, tournamentID, playerID uuid.UUID) (*Participant, error) {
	query := `SELECT id FROM tournament_participants WHERE tournament_id = $1 AND player_id = $2`
	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, tournamentID, playerID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrParticipantNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetParticipant(ctx, id)
}

func (r *PostgresRepository) UpdateParticipant(ctx context.Context, p *Participant) error {
	byeRequestsJSON, _ := json.Marshal(p.ByeRequests)

	query := `
		UPDATE tournament_participants SET
			registration_status = $2,
			start_rank = $3, starting_rating = $4,
			points = $5, buchholz = $6, buchholz_cut1 = $7, sonneborn_berger = $8,
			wins = $9, draws = $10, losses = $11, is_active = $12, bye_requests = $13,
			payment_status = $14, payment_reference = $15
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query,
		p.ID, p.RegistrationStatus,
		p.StartRank, p.StartingRating,
		p.Points, p.Buchholz, p.BuchholzCut1, p.SonnebornBerger,
		p.Wins, p.Draws, p.Losses, p.IsActive, byeRequestsJSON,
		p.PaymentStatus, p.PaymentReference,
	)

	if err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	return nil
}

func (r *PostgresRepository) RemoveParticipant(ctx context.Context, id uuid.UUID) error {
	// Get tournament ID first
	var tournamentID uuid.UUID
	err := r.db.QueryRow(ctx, "SELECT tournament_id FROM tournament_participants WHERE id = $1", id).Scan(&tournamentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrParticipantNotFound
	}
	if err != nil {
		return err
	}

	result, err := r.db.Exec(ctx, "DELETE FROM tournament_participants WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	// Update count
	_, err = r.db.Exec(ctx, `
		UPDATE tournaments SET participant_count = participant_count - 1 WHERE id = $1
	`, tournamentID)

	return err
}

func (r *PostgresRepository) ListParticipants(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error) {
	query := `
		SELECT
			tp.id, tp.tournament_id, tp.section_id, tp.player_id,
			tp.registration_status, tp.registered_at,
			tp.start_rank, tp.starting_rating,
			tp.points, tp.buchholz, tp.buchholz_cut1, tp.sonneborn_berger,
			tp.wins, tp.draws, tp.losses, tp.is_active, tp.bye_requests,
			tp.payment_status, tp.payment_reference,
			p.first_name, p.last_name, p.fide_id,
			c.name as club_name
		FROM tournament_participants tp
		JOIN players p ON tp.player_id = p.id
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE tp.tournament_id = $1
		ORDER BY tp.start_rank ASC`

	rows, err := r.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}
	defer rows.Close()

	var participants []Participant
	for rows.Next() {
		var p Participant
		var byeRequestsJSON []byte

		err := rows.Scan(
			&p.ID, &p.TournamentID, &p.SectionID, &p.PlayerID,
			&p.RegistrationStatus, &p.RegisteredAt,
			&p.StartRank, &p.StartingRating,
			&p.Points, &p.Buchholz, &p.BuchholzCut1, &p.SonnebornBerger,
			&p.Wins, &p.Draws, &p.Losses, &p.IsActive, &byeRequestsJSON,
			&p.PaymentStatus, &p.PaymentReference,
			&p.PlayerFirstName, &p.PlayerLastName, &p.PlayerFideID,
			&p.PlayerClubName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}

		json.Unmarshal(byeRequestsJSON, &p.ByeRequests)
		participants = append(participants, p)
	}

	return participants, nil
}

func (r *PostgresRepository) GetStandings(ctx context.Context, tournamentID uuid.UUID) ([]Participant, error) {
	query := `
		SELECT
			tp.id, tp.tournament_id, tp.section_id, tp.player_id,
			tp.registration_status, tp.registered_at,
			tp.start_rank, tp.starting_rating,
			tp.points, tp.buchholz, tp.buchholz_cut1, tp.sonneborn_berger,
			tp.wins, tp.draws, tp.losses, tp.is_active, tp.bye_requests,
			tp.payment_status, tp.payment_reference,
			p.first_name, p.last_name, p.fide_id,
			c.name as club_name
		FROM tournament_participants tp
		JOIN players p ON tp.player_id = p.id
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE tp.tournament_id = $1 AND tp.is_active = true
		ORDER BY tp.points DESC, tp.buchholz_cut1 DESC, tp.buchholz DESC, tp.sonneborn_berger DESC`

	rows, err := r.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}
	defer rows.Close()

	var standings []Participant
	for rows.Next() {
		var p Participant
		var byeRequestsJSON []byte

		err := rows.Scan(
			&p.ID, &p.TournamentID, &p.SectionID, &p.PlayerID,
			&p.RegistrationStatus, &p.RegisteredAt,
			&p.StartRank, &p.StartingRating,
			&p.Points, &p.Buchholz, &p.BuchholzCut1, &p.SonnebornBerger,
			&p.Wins, &p.Draws, &p.Losses, &p.IsActive, &byeRequestsJSON,
			&p.PaymentStatus, &p.PaymentReference,
			&p.PlayerFirstName, &p.PlayerLastName, &p.PlayerFideID,
			&p.PlayerClubName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan standing: %w", err)
		}

		json.Unmarshal(byeRequestsJSON, &p.ByeRequests)
		standings = append(standings, p)
	}

	return standings, nil
}

// Round operations

func (r *PostgresRepository) CreateRound(ctx context.Context, round *Round) error {
	round.ID = uuid.New()
	if round.Status == "" {
		round.Status = RoundStatusPending
	}

	query := `
		INSERT INTO tournament_rounds (
			id, tournament_id, round_number,
			scheduled_start, actual_start, completed_at,
			status, paired_at, paired_by, pairing_engine_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(ctx, query,
		round.ID, round.TournamentID, round.RoundNumber,
		round.ScheduledStart, round.ActualStart, round.CompletedAt,
		round.Status, round.PairedAt, round.PairedBy, round.PairingEngineUsed,
	)

	if err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetRound(ctx context.Context, id uuid.UUID) (*Round, error) {
	query := `
		SELECT
			r.id, r.tournament_id, r.round_number,
			r.scheduled_start, r.actual_start, r.completed_at,
			r.status, r.paired_at, r.paired_by, r.pairing_engine_used,
			COUNT(g.id) as game_count,
			COUNT(CASE WHEN g.status = 'completed' THEN 1 END) as completed_count
		FROM tournament_rounds r
		LEFT JOIN games g ON r.id = g.round_id
		WHERE r.id = $1
		GROUP BY r.id`

	var round Round
	err := r.db.QueryRow(ctx, query, id).Scan(
		&round.ID, &round.TournamentID, &round.RoundNumber,
		&round.ScheduledStart, &round.ActualStart, &round.CompletedAt,
		&round.Status, &round.PairedAt, &round.PairedBy, &round.PairingEngineUsed,
		&round.GameCount, &round.CompletedCount,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoundNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}

	return &round, nil
}

func (r *PostgresRepository) GetRoundByNumber(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*Round, error) {
	query := `SELECT id FROM tournament_rounds WHERE tournament_id = $1 AND round_number = $2`
	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, tournamentID, roundNumber).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoundNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetRound(ctx, id)
}

func (r *PostgresRepository) UpdateRound(ctx context.Context, round *Round) error {
	query := `
		UPDATE tournament_rounds SET
			scheduled_start = $2, actual_start = $3, completed_at = $4,
			status = $5, paired_at = $6, paired_by = $7, pairing_engine_used = $8
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query,
		round.ID,
		round.ScheduledStart, round.ActualStart, round.CompletedAt,
		round.Status, round.PairedAt, round.PairedBy, round.PairingEngineUsed,
	)

	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRoundNotFound
	}

	return nil
}

func (r *PostgresRepository) ListRounds(ctx context.Context, tournamentID uuid.UUID) ([]Round, error) {
	query := `
		SELECT
			r.id, r.tournament_id, r.round_number,
			r.scheduled_start, r.actual_start, r.completed_at,
			r.status, r.paired_at, r.paired_by, r.pairing_engine_used,
			COUNT(g.id) as game_count,
			COUNT(CASE WHEN g.status = 'completed' THEN 1 END) as completed_count
		FROM tournament_rounds r
		LEFT JOIN games g ON r.id = g.round_id
		WHERE r.tournament_id = $1
		GROUP BY r.id
		ORDER BY r.round_number`

	rows, err := r.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list rounds: %w", err)
	}
	defer rows.Close()

	var rounds []Round
	for rows.Next() {
		var round Round
		err := rows.Scan(
			&round.ID, &round.TournamentID, &round.RoundNumber,
			&round.ScheduledStart, &round.ActualStart, &round.CompletedAt,
			&round.Status, &round.PairedAt, &round.PairedBy, &round.PairingEngineUsed,
			&round.GameCount, &round.CompletedCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan round: %w", err)
		}
		rounds = append(rounds, round)
	}

	return rounds, nil
}

// Game operations

func (r *PostgresRepository) CreateGame(ctx context.Context, g *Game) error {
	g.ID = uuid.New()
	g.CreatedAt = time.Now()
	if g.Status == "" {
		g.Status = GameStatusScheduled
	}

	query := `
		INSERT INTO games (
			id, tournament_id, round_id, section_id,
			board_number, white_participant_id, black_participant_id,
			result, white_score, black_score,
			online_game_id, status,
			result_entered_by, result_entered_at, pgn,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err := r.db.Exec(ctx, query,
		g.ID, g.TournamentID, g.RoundID, g.SectionID,
		g.BoardNumber, g.WhiteParticipantID, g.BlackParticipantID,
		g.Result, g.WhiteScore, g.BlackScore,
		g.OnlineGameID, g.Status,
		g.ResultEnteredBy, g.ResultEnteredAt, g.PGN,
		g.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	return nil
}

func (r *PostgresRepository) CreateGames(ctx context.Context, games []Game) error {
	batch := &pgx.Batch{}

	for _, g := range games {
		if g.ID == uuid.Nil {
			g.ID = uuid.New()
		}
		if g.CreatedAt.IsZero() {
			g.CreatedAt = time.Now()
		}
		if g.Status == "" {
			g.Status = GameStatusScheduled
		}

		query := `
			INSERT INTO games (
				id, tournament_id, round_id, section_id,
				board_number, white_participant_id, black_participant_id,
				result, white_score, black_score,
				online_game_id, status,
				result_entered_by, result_entered_at, pgn,
				created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

		batch.Queue(query,
			g.ID, g.TournamentID, g.RoundID, g.SectionID,
			g.BoardNumber, g.WhiteParticipantID, g.BlackParticipantID,
			g.Result, g.WhiteScore, g.BlackScore,
			g.OnlineGameID, g.Status,
			g.ResultEnteredBy, g.ResultEnteredAt, g.PGN,
			g.CreatedAt,
		)
	}

	results := r.db.SendBatch(ctx, batch)
	defer results.Close()

	for range games {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("failed to create games batch: %w", err)
		}
	}

	return nil
}

func (r *PostgresRepository) GetGame(ctx context.Context, id uuid.UUID) (*Game, error) {
	query := `
		SELECT
			g.id, g.tournament_id, g.round_id, g.section_id,
			g.board_number, g.white_participant_id, g.black_participant_id,
			g.result, g.white_score, g.black_score,
			g.online_game_id, g.status,
			g.result_entered_by, g.result_entered_at, g.pgn,
			g.created_at
		FROM games g
		WHERE g.id = $1`

	var game Game
	err := r.db.QueryRow(ctx, query, id).Scan(
		&game.ID, &game.TournamentID, &game.RoundID, &game.SectionID,
		&game.BoardNumber, &game.WhiteParticipantID, &game.BlackParticipantID,
		&game.Result, &game.WhiteScore, &game.BlackScore,
		&game.OnlineGameID, &game.Status,
		&game.ResultEnteredBy, &game.ResultEnteredAt, &game.PGN,
		&game.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGameNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	return &game, nil
}

func (r *PostgresRepository) UpdateGame(ctx context.Context, g *Game) error {
	query := `
		UPDATE games SET
			result = $2, white_score = $3, black_score = $4,
			status = $5, result_entered_by = $6, result_entered_at = $7, pgn = $8
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query,
		g.ID, g.Result, g.WhiteScore, g.BlackScore,
		g.Status, g.ResultEnteredBy, g.ResultEnteredAt, g.PGN,
	)

	if err != nil {
		return fmt.Errorf("failed to update game: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGameNotFound
	}

	return nil
}

func (r *PostgresRepository) ListGamesByRound(ctx context.Context, roundID uuid.UUID) ([]Game, error) {
	query := `
		SELECT
			g.id, g.tournament_id, g.round_id, g.section_id,
			g.board_number, g.white_participant_id, g.black_participant_id,
			g.result, g.white_score, g.black_score,
			g.online_game_id, g.status,
			g.result_entered_by, g.result_entered_at, g.pgn,
			g.created_at,
			CASE WHEN wp.id IS NOT NULL THEN
				COALESCE(wpplayer.last_name, '') || CASE WHEN wpplayer.first_name IS NOT NULL THEN ', ' || wpplayer.first_name ELSE '' END
			END as white_player_name,
			CASE WHEN bp.id IS NOT NULL THEN
				COALESCE(bpplayer.last_name, '') || CASE WHEN bpplayer.first_name IS NOT NULL THEN ', ' || bpplayer.first_name ELSE '' END
			END as black_player_name,
			wp.starting_rating as white_rating,
			bp.starting_rating as black_rating
		FROM games g
		LEFT JOIN tournament_participants wp ON g.white_participant_id = wp.id
		LEFT JOIN tournament_participants bp ON g.black_participant_id = bp.id
		LEFT JOIN players wpplayer ON wp.player_id = wpplayer.id
		LEFT JOIN players bpplayer ON bp.player_id = bpplayer.id
		WHERE g.round_id = $1
		ORDER BY g.board_number`

	rows, err := r.db.Query(ctx, query, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to list games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		err := rows.Scan(
			&g.ID, &g.TournamentID, &g.RoundID, &g.SectionID,
			&g.BoardNumber, &g.WhiteParticipantID, &g.BlackParticipantID,
			&g.Result, &g.WhiteScore, &g.BlackScore,
			&g.OnlineGameID, &g.Status,
			&g.ResultEnteredBy, &g.ResultEnteredAt, &g.PGN,
			&g.CreatedAt,
			&g.WhitePlayerName, &g.BlackPlayerName,
			&g.WhiteRating, &g.BlackRating,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}
		games = append(games, g)
	}

	return games, nil
}

func (r *PostgresRepository) ListGamesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Game, error) {
	query := `
		SELECT
			g.id, g.tournament_id, g.round_id, g.section_id,
			g.board_number, g.white_participant_id, g.black_participant_id,
			g.result, g.white_score, g.black_score,
			g.online_game_id, g.status,
			g.result_entered_by, g.result_entered_at, g.pgn,
			g.created_at
		FROM games g
		JOIN tournament_rounds r ON g.round_id = r.id
		WHERE g.tournament_id = $1
		ORDER BY r.round_number, g.board_number`

	rows, err := r.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		err := rows.Scan(
			&g.ID, &g.TournamentID, &g.RoundID, &g.SectionID,
			&g.BoardNumber, &g.WhiteParticipantID, &g.BlackParticipantID,
			&g.Result, &g.WhiteScore, &g.BlackScore,
			&g.OnlineGameID, &g.Status,
			&g.ResultEnteredBy, &g.ResultEnteredAt, &g.PGN,
			&g.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}
		games = append(games, g)
	}

	return games, nil
}

func (r *PostgresRepository) GetPlayerGames(ctx context.Context, tournamentID, participantID uuid.UUID) ([]Game, error) {
	query := `
		SELECT
			g.id, g.tournament_id, g.round_id, g.section_id,
			g.board_number, g.white_participant_id, g.black_participant_id,
			g.result, g.white_score, g.black_score,
			g.online_game_id, g.status,
			g.result_entered_by, g.result_entered_at, g.pgn,
			g.created_at
		FROM games g
		JOIN tournament_rounds r ON g.round_id = r.id
		WHERE g.tournament_id = $1
		  AND (g.white_participant_id = $2 OR g.black_participant_id = $2)
		ORDER BY r.round_number`

	rows, err := r.db.Query(ctx, query, tournamentID, participantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		err := rows.Scan(
			&g.ID, &g.TournamentID, &g.RoundID, &g.SectionID,
			&g.BoardNumber, &g.WhiteParticipantID, &g.BlackParticipantID,
			&g.Result, &g.WhiteScore, &g.BlackScore,
			&g.OnlineGameID, &g.Status,
			&g.ResultEnteredBy, &g.ResultEnteredAt, &g.PGN,
			&g.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}
		games = append(games, g)
	}

	return games, nil
}

// Forbidden pairs

func (r *PostgresRepository) AddForbiddenPair(ctx context.Context, fp *ForbiddenPair) error {
	fp.ID = uuid.New()
	fp.CreatedAt = time.Now()

	query := `
		INSERT INTO forbidden_pairs (id, tournament_id, player1_id, player2_id, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.Exec(ctx, query, fp.ID, fp.TournamentID, fp.Player1ID, fp.Player2ID, fp.Reason, fp.CreatedAt)
	return err
}

func (r *PostgresRepository) ListForbiddenPairs(ctx context.Context, tournamentID uuid.UUID) ([]ForbiddenPair, error) {
	query := `SELECT id, tournament_id, player1_id, player2_id, reason, created_at FROM forbidden_pairs WHERE tournament_id = $1`

	rows, err := r.db.Query(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []ForbiddenPair
	for rows.Next() {
		var fp ForbiddenPair
		if err := rows.Scan(&fp.ID, &fp.TournamentID, &fp.Player1ID, &fp.Player2ID, &fp.Reason, &fp.CreatedAt); err != nil {
			return nil, err
		}
		pairs = append(pairs, fp)
	}

	return pairs, nil
}

func (r *PostgresRepository) RemoveForbiddenPair(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM forbidden_pairs WHERE id = $1", id)
	return err
}

// Helper functions

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "ä", "ae")
	slug = strings.ReplaceAll(slug, "ö", "oe")
	slug = strings.ReplaceAll(slug, "ü", "ue")
	slug = strings.ReplaceAll(slug, "ß", "ss")

	// Add timestamp for uniqueness
	slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix())

	return slug
}
