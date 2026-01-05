package club

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service provides club business logic
type Service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new club service
func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("component", "club_service").Logger(),
	}
}

// Create creates a new club
func (s *Service) Create(ctx context.Context, req *CreateClubRequest) (*Club, error) {
	club := &Club{
		Name:       req.Name,
		ShortName:  req.ShortName,
		Federation: req.Federation,
		DsbClubID:  req.DsbClubID,
		OesbClubID: req.OesbClubID,
		City:       req.City,
		Country:    req.Country,
		Website:    req.Website,
		Email:      req.Email,
	}

	if err := s.repo.Create(ctx, club); err != nil {
		s.logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create club")
		return nil, err
	}

	s.logger.Info().
		Str("clubId", club.ID.String()).
		Str("name", club.Name).
		Msg("Club created")

	return club, nil
}

// GetByID retrieves a club by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Club, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByDsbID retrieves a club by DSB club ID
func (s *Service) GetByDsbID(ctx context.Context, dsbID string) (*Club, error) {
	return s.repo.GetByDsbID(ctx, dsbID)
}

// GetByOesbID retrieves a club by ÖSB club ID
func (s *Service) GetByOesbID(ctx context.Context, oesbID string) (*Club, error) {
	return s.repo.GetByOesbID(ctx, oesbID)
}

// Update updates an existing club
func (s *Service) Update(ctx context.Context, id uuid.UUID, req *UpdateClubRequest) (*Club, error) {
	club, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		club.Name = *req.Name
	}
	if req.ShortName != nil {
		club.ShortName = req.ShortName
	}
	if req.Federation != nil {
		club.Federation = req.Federation
	}
	if req.DsbClubID != nil {
		club.DsbClubID = req.DsbClubID
	}
	if req.OesbClubID != nil {
		club.OesbClubID = req.OesbClubID
	}
	if req.City != nil {
		club.City = req.City
	}
	if req.Country != nil {
		club.Country = *req.Country
	}
	if req.Website != nil {
		club.Website = req.Website
	}
	if req.Email != nil {
		club.Email = req.Email
	}

	if err := s.repo.Update(ctx, club); err != nil {
		s.logger.Error().Err(err).Str("clubId", id.String()).Msg("Failed to update club")
		return nil, err
	}

	s.logger.Info().
		Str("clubId", id.String()).
		Msg("Club updated")

	return club, nil
}

// Delete deletes a club
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error().Err(err).Str("clubId", id.String()).Msg("Failed to delete club")
		return err
	}

	s.logger.Info().
		Str("clubId", id.String()).
		Msg("Club deleted")

	return nil
}

// List lists clubs with pagination and filtering
func (s *Service) List(ctx context.Context, params ClubListParams) (*ClubListResponse, error) {
	return s.repo.List(ctx, params)
}

// GetMembers retrieves all members of a club
func (s *Service) GetMembers(ctx context.Context, clubID uuid.UUID) ([]ClubMember, error) {
	// Verify club exists
	if _, err := s.repo.GetByID(ctx, clubID); err != nil {
		return nil, err
	}

	return s.repo.GetMembers(ctx, clubID)
}

// GetStats retrieves statistics for a club
func (s *Service) GetStats(ctx context.Context, clubID uuid.UUID) (*ClubStats, error) {
	// Verify club exists
	if _, err := s.repo.GetByID(ctx, clubID); err != nil {
		return nil, err
	}

	return s.repo.GetStats(ctx, clubID)
}

// Search searches for clubs by name
func (s *Service) Search(ctx context.Context, query string, limit int) ([]Club, error) {
	return s.repo.Search(ctx, query, limit)
}
