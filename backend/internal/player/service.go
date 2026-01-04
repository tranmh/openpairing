package player

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Service struct {
	repo   Repository
	logger zerolog.Logger
}

func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("service", "player").Logger(),
	}
}

// CreatePlayer creates a new player in the system
func (s *Service) CreatePlayer(ctx context.Context, req *CreatePlayerRequest) (*Player, error) {
	player := &Player{
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		BirthDate:    req.BirthDate,
		Gender:       req.Gender,
		FideID:       req.FideID,
		DsbID:        req.DsbID,
		OesbID:       req.OesbID,
		FideStandard: req.FideStandard,
		FideRapid:    req.FideRapid,
		FideBlitz:    req.FideBlitz,
		DWZ:          req.DWZ,
		OesbElo:      req.OesbElo,
		FideTitle:    req.FideTitle,
		Federation:   req.Federation,
	}

	// Parse club ID if provided
	if req.ClubID != nil && *req.ClubID != "" {
		clubUUID, err := uuid.Parse(*req.ClubID)
		if err != nil {
			return nil, errors.New("invalid club ID format")
		}
		player.ClubID = &clubUUID
	}

	if err := s.repo.Create(ctx, player); err != nil {
		s.logger.Error().Err(err).Str("name", player.GetDisplayName()).Msg("Failed to create player")
		return nil, err
	}

	s.logger.Info().
		Str("id", player.ID.String()).
		Str("name", player.GetDisplayName()).
		Msg("Player created")

	return player, nil
}

// GetPlayer retrieves a player by ID
func (s *Service) GetPlayer(ctx context.Context, id uuid.UUID) (*Player, error) {
	player, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrPlayerNotFound) {
			return nil, err
		}
		s.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to get player")
		return nil, err
	}
	return player, nil
}

// GetPlayerByFideID retrieves a player by their FIDE ID
func (s *Service) GetPlayerByFideID(ctx context.Context, fideID string) (*Player, error) {
	player, err := s.repo.GetByFideID(ctx, fideID)
	if err != nil {
		if errors.Is(err, ErrPlayerNotFound) {
			return nil, err
		}
		s.logger.Error().Err(err).Str("fideId", fideID).Msg("Failed to get player by FIDE ID")
		return nil, err
	}
	return player, nil
}

// GetPlayerByDsbID retrieves a player by their DSB PKZ
func (s *Service) GetPlayerByDsbID(ctx context.Context, dsbID string) (*Player, error) {
	player, err := s.repo.GetByDsbID(ctx, dsbID)
	if err != nil {
		if errors.Is(err, ErrPlayerNotFound) {
			return nil, err
		}
		s.logger.Error().Err(err).Str("dsbId", dsbID).Msg("Failed to get player by DSB ID")
		return nil, err
	}
	return player, nil
}

// GetPlayerByOesbID retrieves a player by their ÖSB ID
func (s *Service) GetPlayerByOesbID(ctx context.Context, oesbID string) (*Player, error) {
	player, err := s.repo.GetByOesbID(ctx, oesbID)
	if err != nil {
		if errors.Is(err, ErrPlayerNotFound) {
			return nil, err
		}
		s.logger.Error().Err(err).Str("oesbId", oesbID).Msg("Failed to get player by ÖSB ID")
		return nil, err
	}
	return player, nil
}

// GetPlayerByUserID retrieves the player profile linked to a user account
func (s *Service) GetPlayerByUserID(ctx context.Context, userID uuid.UUID) (*Player, error) {
	player, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrPlayerNotFound) {
			return nil, err
		}
		s.logger.Error().Err(err).Str("userId", userID.String()).Msg("Failed to get player by user ID")
		return nil, err
	}
	return player, nil
}

// UpdatePlayer updates an existing player
func (s *Service) UpdatePlayer(ctx context.Context, id uuid.UUID, req *UpdatePlayerRequest) (*Player, error) {
	player, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.FirstName != nil {
		player.FirstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		player.LastName = strings.TrimSpace(*req.LastName)
	}
	if req.BirthDate != nil {
		player.BirthDate = req.BirthDate
	}
	if req.Gender != nil {
		player.Gender = req.Gender
	}
	if req.FideID != nil {
		player.FideID = req.FideID
	}
	if req.DsbID != nil {
		player.DsbID = req.DsbID
	}
	if req.OesbID != nil {
		player.OesbID = req.OesbID
	}
	if req.FideStandard != nil {
		player.FideStandard = req.FideStandard
	}
	if req.FideRapid != nil {
		player.FideRapid = req.FideRapid
	}
	if req.FideBlitz != nil {
		player.FideBlitz = req.FideBlitz
	}
	if req.DWZ != nil {
		player.DWZ = req.DWZ
	}
	if req.OesbElo != nil {
		player.OesbElo = req.OesbElo
	}
	if req.FideTitle != nil {
		player.FideTitle = req.FideTitle
	}
	if req.Federation != nil {
		player.Federation = req.Federation
	}
	if req.ClubID != nil {
		if *req.ClubID == "" {
			player.ClubID = nil
		} else {
			clubUUID, err := uuid.Parse(*req.ClubID)
			if err != nil {
				return nil, errors.New("invalid club ID format")
			}
			player.ClubID = &clubUUID
		}
	}

	if err := s.repo.Update(ctx, player); err != nil {
		s.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to update player")
		return nil, err
	}

	s.logger.Info().
		Str("id", player.ID.String()).
		Str("name", player.GetDisplayName()).
		Msg("Player updated")

	// Fetch updated player with club name
	return s.repo.GetByID(ctx, id)
}

// DeletePlayer removes a player from the system
func (s *Service) DeletePlayer(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrPlayerNotFound) {
			return err
		}
		s.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to delete player")
		return err
	}

	s.logger.Info().Str("id", id.String()).Msg("Player deleted")
	return nil
}

// SearchPlayers searches for players based on query and filters
func (s *Service) SearchPlayers(ctx context.Context, req *SearchPlayersRequest) (*PlayerListResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	results, total, err := s.repo.Search(ctx, req)
	if err != nil {
		s.logger.Error().Err(err).Str("query", req.Query).Msg("Failed to search players")
		return nil, err
	}

	// Convert PlayerSearchResult to Player
	players := make([]Player, len(results))
	for i, r := range results {
		players[i] = r.Player
	}

	return &PlayerListResponse{
		Players: players,
		Total:   total,
		Page:    req.Page,
		Limit:   req.Limit,
	}, nil
}

// ListPlayers returns a paginated list of all players
func (s *Service) ListPlayers(ctx context.Context, page, limit int) (*PlayerListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	players, total, err := s.repo.List(ctx, page, limit)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list players")
		return nil, err
	}

	return &PlayerListResponse{
		Players: players,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

// GetRatingHistory retrieves the rating history for a player
func (s *Service) GetRatingHistory(ctx context.Context, playerID uuid.UUID, ratingType string, limit int) ([]RatingHistory, error) {
	if limit <= 0 {
		limit = 50
	}

	history, err := s.repo.GetRatingHistory(ctx, playerID, ratingType, limit)
	if err != nil {
		s.logger.Error().Err(err).Str("playerId", playerID.String()).Msg("Failed to get rating history")
		return nil, err
	}

	return history, nil
}

// LinkPlayerToUser links a player profile to a user account
func (s *Service) LinkPlayerToUser(ctx context.Context, playerID, userID uuid.UUID) error {
	player, err := s.repo.GetByID(ctx, playerID)
	if err != nil {
		return err
	}

	player.UserID = &userID
	if err := s.repo.Update(ctx, player); err != nil {
		s.logger.Error().Err(err).
			Str("playerId", playerID.String()).
			Str("userId", userID.String()).
			Msg("Failed to link player to user")
		return err
	}

	s.logger.Info().
		Str("playerId", playerID.String()).
		Str("userId", userID.String()).
		Msg("Player linked to user")

	return nil
}

// Club operations

// CreateClub creates a new chess club
func (s *Service) CreateClub(ctx context.Context, club *Club) error {
	if err := s.repo.CreateClub(ctx, club); err != nil {
		s.logger.Error().Err(err).Str("name", club.Name).Msg("Failed to create club")
		return err
	}
	s.logger.Info().Str("id", club.ID.String()).Str("name", club.Name).Msg("Club created")
	return nil
}

// GetClub retrieves a club by ID
func (s *Service) GetClub(ctx context.Context, id uuid.UUID) (*Club, error) {
	return s.repo.GetClubByID(ctx, id)
}

// ListClubs returns a paginated list of clubs
func (s *Service) ListClubs(ctx context.Context, page, limit int) ([]Club, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListClubs(ctx, page, limit)
}
