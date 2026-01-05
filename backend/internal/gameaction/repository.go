package gameaction

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for game actions
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new game action repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetGame retrieves a chess game by ID
func (r *Repository) GetGame(ctx context.Context, gameID uuid.UUID) (*ChessGame, error) {
	query := `
		SELECT id, white_user_id, black_user_id, fen, moves, move_count,
		       white_clock_ms, black_clock_ms, initial_time_ms, increment_ms,
		       status, result, termination, started_at, last_move_at,
		       completed_at, created_at
		FROM chess_games
		WHERE id = $1`

	var g ChessGame
	err := r.pool.QueryRow(ctx, query, gameID).Scan(
		&g.ID, &g.WhiteUserID, &g.BlackUserID, &g.FEN, &g.Moves, &g.MoveCount,
		&g.WhiteClockMs, &g.BlackClockMs, &g.InitialTimeMs, &g.IncrementMs,
		&g.Status, &g.Result, &g.Termination, &g.StartedAt, &g.LastMoveAt,
		&g.CompletedAt, &g.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, err
	}
	return &g, nil
}

// GetGameWithPlayers retrieves a game with player information
func (r *Repository) GetGameWithPlayers(ctx context.Context, gameID uuid.UUID) (*GameWithPlayers, error) {
	query := `
		SELECT g.id, g.white_user_id, g.black_user_id, g.fen, g.moves, g.move_count,
		       g.white_clock_ms, g.black_clock_ms, g.initial_time_ms, g.increment_ms,
		       g.status, g.result, g.termination, g.started_at, g.last_move_at,
		       g.completed_at, g.created_at,
		       COALESCE(wp.first_name || ' ' || wp.last_name, wu.email) as white_player_name,
		       COALESCE(bp.first_name || ' ' || bp.last_name, bu.email) as black_player_name,
		       wp.fide_standard as white_rating,
		       bp.fide_standard as black_rating
		FROM chess_games g
		LEFT JOIN users wu ON wu.id = g.white_user_id
		LEFT JOIN players wp ON wp.user_id = g.white_user_id
		LEFT JOIN users bu ON bu.id = g.black_user_id
		LEFT JOIN players bp ON bp.user_id = g.black_user_id
		WHERE g.id = $1`

	var g GameWithPlayers
	err := r.pool.QueryRow(ctx, query, gameID).Scan(
		&g.ID, &g.WhiteUserID, &g.BlackUserID, &g.FEN, &g.Moves, &g.MoveCount,
		&g.WhiteClockMs, &g.BlackClockMs, &g.InitialTimeMs, &g.IncrementMs,
		&g.Status, &g.Result, &g.Termination, &g.StartedAt, &g.LastMoveAt,
		&g.CompletedAt, &g.CreatedAt,
		&g.WhitePlayerName, &g.BlackPlayerName, &g.WhiteRating, &g.BlackRating,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, err
	}
	return &g, nil
}

// UpdateGameResult updates the result of a game
func (r *Repository) UpdateGameResult(ctx context.Context, gameID uuid.UUID, result string, termination Termination) error {
	query := `
		UPDATE chess_games
		SET status = 'completed', result = $1, termination = $2, completed_at = NOW()
		WHERE id = $3`

	res, err := r.pool.Exec(ctx, query, result, termination, gameID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("game not found")
	}
	return nil
}

// AbortGame marks a game as aborted
func (r *Repository) AbortGame(ctx context.Context, gameID uuid.UUID) error {
	query := `
		UPDATE chess_games
		SET status = 'aborted', termination = 'aborted', completed_at = NOW()
		WHERE id = $1`

	res, err := r.pool.Exec(ctx, query, gameID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("game not found")
	}
	return nil
}

// IsPlayerInGame checks if a user is a player in the game
func (r *Repository) IsPlayerInGame(ctx context.Context, gameID, userID uuid.UUID) (bool, string, error) {
	query := `
		SELECT
			CASE
				WHEN white_user_id = $2 THEN 'white'
				WHEN black_user_id = $2 THEN 'black'
				ELSE ''
			END as color
		FROM chess_games
		WHERE id = $1`

	var color string
	err := r.pool.QueryRow(ctx, query, gameID, userID).Scan(&color)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, "", fmt.Errorf("game not found")
		}
		return false, "", err
	}
	return color != "", color, nil
}

// CreateArbiterRequest creates a new arbiter request
func (r *Repository) CreateArbiterRequest(ctx context.Context, gameID, requestedBy uuid.UUID, reason string) (*ArbiterRequest, error) {
	query := `
		INSERT INTO arbiter_requests (game_id, requested_by, reason, status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING id, game_id, requested_by, reason, status, created_at`

	var req ArbiterRequest
	err := r.pool.QueryRow(ctx, query, gameID, requestedBy, reason).Scan(
		&req.ID, &req.GameID, &req.RequestedBy, &req.Reason, &req.Status, &req.CreatedAt,
	)
	if err != nil {
		// Table might not exist, create it inline for now
		_, createErr := r.pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS arbiter_requests (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				game_id UUID NOT NULL,
				requested_by UUID NOT NULL REFERENCES users(id),
				reason TEXT NOT NULL,
				status VARCHAR(50) NOT NULL DEFAULT 'pending',
				resolved_by UUID REFERENCES users(id),
				resolution TEXT,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				resolved_at TIMESTAMPTZ
			)`)
		if createErr != nil {
			return nil, err
		}
		// Retry
		err = r.pool.QueryRow(ctx, query, gameID, requestedBy, reason).Scan(
			&req.ID, &req.GameID, &req.RequestedBy, &req.Reason, &req.Status, &req.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
	}
	return &req, nil
}

// GetActiveArbiterRequest gets any pending arbiter request for a game
func (r *Repository) GetActiveArbiterRequest(ctx context.Context, gameID uuid.UUID) (*ArbiterRequest, error) {
	query := `
		SELECT id, game_id, requested_by, reason, status, resolved_by, resolution, created_at, resolved_at
		FROM arbiter_requests
		WHERE game_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1`

	var req ArbiterRequest
	err := r.pool.QueryRow(ctx, query, gameID).Scan(
		&req.ID, &req.GameID, &req.RequestedBy, &req.Reason, &req.Status,
		&req.ResolvedBy, &req.Resolution, &req.CreatedAt, &req.ResolvedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No active request
		}
		return nil, err
	}
	return &req, nil
}

// GetUserActiveGames gets active games for a user
func (r *Repository) GetUserActiveGames(ctx context.Context, userID uuid.UUID) ([]ChessGame, error) {
	query := `
		SELECT id, white_user_id, black_user_id, fen, moves, move_count,
		       white_clock_ms, black_clock_ms, initial_time_ms, increment_ms,
		       status, result, termination, started_at, last_move_at,
		       completed_at, created_at
		FROM chess_games
		WHERE (white_user_id = $1 OR black_user_id = $1)
		  AND status IN ('waiting', 'ongoing')
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []ChessGame
	for rows.Next() {
		var g ChessGame
		if err := rows.Scan(
			&g.ID, &g.WhiteUserID, &g.BlackUserID, &g.FEN, &g.Moves, &g.MoveCount,
			&g.WhiteClockMs, &g.BlackClockMs, &g.InitialTimeMs, &g.IncrementMs,
			&g.Status, &g.Result, &g.Termination, &g.StartedAt, &g.LastMoveAt,
			&g.CompletedAt, &g.CreatedAt,
		); err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, nil
}

// draw offer storage (in-memory for now, could be Redis in production)
var drawOffers = make(map[uuid.UUID]*DrawOffer)

// SetDrawOffer stores a draw offer
func (r *Repository) SetDrawOffer(gameID, offeredBy uuid.UUID) *DrawOffer {
	offer := &DrawOffer{
		GameID:    gameID,
		OfferedBy: offeredBy,
		OfferedAt: time.Now(),
		ExpiresAt: time.Now().Add(60 * time.Second), // 60 second expiry
	}
	drawOffers[gameID] = offer
	return offer
}

// GetDrawOffer retrieves a draw offer for a game
func (r *Repository) GetDrawOffer(gameID uuid.UUID) *DrawOffer {
	offer, exists := drawOffers[gameID]
	if !exists {
		return nil
	}
	// Check if expired
	if time.Now().After(offer.ExpiresAt) {
		delete(drawOffers, gameID)
		return nil
	}
	return offer
}

// ClearDrawOffer removes a draw offer
func (r *Repository) ClearDrawOffer(gameID uuid.UUID) {
	delete(drawOffers, gameID)
}
