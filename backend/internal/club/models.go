package club

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrClubNotFound      = errors.New("club not found")
	ErrDuplicateClub     = errors.New("club with this identifier already exists")
	ErrInvalidClubData   = errors.New("invalid club data")
)

// Club represents a chess club
type Club struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	ShortName  *string   `json:"shortName,omitempty" db:"short_name"`
	Federation *string   `json:"federation,omitempty" db:"federation"`
	DsbClubID  *string   `json:"dsbClubId,omitempty" db:"dsb_club_id"`
	OesbClubID *string   `json:"oesbClubId,omitempty" db:"oesb_club_id"`
	City       *string   `json:"city,omitempty" db:"city"`
	Country    string    `json:"country" db:"country"`
	Website    *string   `json:"website,omitempty" db:"website"`
	Email      *string   `json:"email,omitempty" db:"email"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt" db:"updated_at"`
}

// CreateClubRequest represents a request to create a club
type CreateClubRequest struct {
	Name       string  `json:"name" binding:"required,min=2,max=255"`
	ShortName  *string `json:"shortName,omitempty" binding:"omitempty,max=50"`
	Federation *string `json:"federation,omitempty" binding:"omitempty,len=3"`
	DsbClubID  *string `json:"dsbClubId,omitempty" binding:"omitempty,max=20"`
	OesbClubID *string `json:"oesbClubId,omitempty" binding:"omitempty,max=20"`
	City       *string `json:"city,omitempty" binding:"omitempty,max=100"`
	Country    string  `json:"country" binding:"required,len=3"`
	Website    *string `json:"website,omitempty" binding:"omitempty,url,max=500"`
	Email      *string `json:"email,omitempty" binding:"omitempty,email,max=255"`
}

// UpdateClubRequest represents a request to update a club
type UpdateClubRequest struct {
	Name       *string `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	ShortName  *string `json:"shortName,omitempty" binding:"omitempty,max=50"`
	Federation *string `json:"federation,omitempty" binding:"omitempty,len=3"`
	DsbClubID  *string `json:"dsbClubId,omitempty" binding:"omitempty,max=20"`
	OesbClubID *string `json:"oesbClubId,omitempty" binding:"omitempty,max=20"`
	City       *string `json:"city,omitempty" binding:"omitempty,max=100"`
	Country    *string `json:"country,omitempty" binding:"omitempty,len=3"`
	Website    *string `json:"website,omitempty" binding:"omitempty,url,max=500"`
	Email      *string `json:"email,omitempty" binding:"omitempty,email,max=255"`
}

// ClubListParams contains parameters for listing clubs
type ClubListParams struct {
	Search     string `form:"search"`
	Federation string `form:"federation"`
	City       string `form:"city"`
	Country    string `form:"country"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"pageSize,default=20"`
}

// ClubListResponse contains a paginated list of clubs
type ClubListResponse struct {
	Clubs      []Club `json:"clubs"`
	TotalCount int    `json:"totalCount"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	TotalPages int    `json:"totalPages"`
}

// ClubMember represents a player's membership in a club
type ClubMember struct {
	PlayerID    uuid.UUID  `json:"playerId" db:"player_id"`
	PlayerName  string     `json:"playerName"`
	Rating      int        `json:"rating"`
	Title       *string    `json:"title,omitempty"`
	JoinedAt    *time.Time `json:"joinedAt,omitempty" db:"joined_at"`
}

// ClubStats contains statistics about a club
type ClubStats struct {
	MemberCount     int     `json:"memberCount"`
	AverageRating   float64 `json:"averageRating"`
	HighestRating   int     `json:"highestRating"`
	TitledPlayers   int     `json:"titledPlayers"`
	ActiveTeams     int     `json:"activeTeams"`
}
