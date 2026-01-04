package player

import (
	"time"

	"github.com/google/uuid"
)

// Player represents a chess player in the system
type Player struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    *uuid.UUID `json:"userId,omitempty" db:"user_id"`

	// Personal Info
	FirstName string  `json:"firstName" db:"first_name"`
	LastName  string  `json:"lastName" db:"last_name"`
	BirthDate *string `json:"birthDate,omitempty" db:"birth_date"`
	Gender    *string `json:"gender,omitempty" db:"gender"`

	// Federation IDs
	FideID *string `json:"fideId,omitempty" db:"fide_id"`
	DsbID  *string `json:"dsbId,omitempty" db:"dsb_id"`
	OesbID *string `json:"oesbId,omitempty" db:"oesb_id"`
	SsbID  *string `json:"ssbId,omitempty" db:"ssb_id"`

	// Ratings
	FideStandard *int `json:"fideStandard,omitempty" db:"fide_standard"`
	FideRapid    *int `json:"fideRapid,omitempty" db:"fide_rapid"`
	FideBlitz    *int `json:"fideBlitz,omitempty" db:"fide_blitz"`
	DWZ          *int `json:"dwz,omitempty" db:"dwz"`
	DWZIndex     *int `json:"dwzIndex,omitempty" db:"dwz_index"`
	OesbElo      *int `json:"oesbElo,omitempty" db:"oesb_elo"`

	// Titles
	FideTitle     *string `json:"fideTitle,omitempty" db:"fide_title"`
	NationalTitle *string `json:"nationalTitle,omitempty" db:"national_title"`

	// Club & Federation
	ClubID     *uuid.UUID `json:"clubId,omitempty" db:"club_id"`
	ClubName   *string    `json:"clubName,omitempty" db:"club_name"`
	Federation *string    `json:"federation,omitempty" db:"federation"`

	// Timestamps
	RatingsUpdatedAt *time.Time `json:"ratingsUpdatedAt,omitempty" db:"ratings_updated_at"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time  `json:"updatedAt" db:"updated_at"`
}

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
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
}

// RatingHistory tracks rating changes over time
type RatingHistory struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	PlayerID     uuid.UUID  `json:"playerId" db:"player_id"`
	RatingType   string     `json:"ratingType" db:"rating_type"`
	OldRating    *int       `json:"oldRating,omitempty" db:"old_rating"`
	NewRating    int        `json:"newRating" db:"new_rating"`
	TournamentID *uuid.UUID `json:"tournamentId,omitempty" db:"tournament_id"`
	ChangeDate   string     `json:"changeDate" db:"change_date"`
	Source       string     `json:"source" db:"source"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
}

// CreatePlayerRequest is the request body for creating a player
type CreatePlayerRequest struct {
	FirstName    string  `json:"firstName" binding:"required,min=1,max=100"`
	LastName     string  `json:"lastName" binding:"required,min=1,max=100"`
	BirthDate    *string `json:"birthDate,omitempty"`
	Gender       *string `json:"gender,omitempty" binding:"omitempty,oneof=m f other"`
	FideID       *string `json:"fideId,omitempty"`
	DsbID        *string `json:"dsbId,omitempty"`
	OesbID       *string `json:"oesbId,omitempty"`
	FideStandard *int    `json:"fideStandard,omitempty"`
	FideRapid    *int    `json:"fideRapid,omitempty"`
	FideBlitz    *int    `json:"fideBlitz,omitempty"`
	DWZ          *int    `json:"dwz,omitempty"`
	OesbElo      *int    `json:"oesbElo,omitempty"`
	FideTitle    *string `json:"fideTitle,omitempty" binding:"omitempty,oneof=GM IM FM CM WGM WIM WFM WCM"`
	ClubID       *string `json:"clubId,omitempty"`
	Federation   *string `json:"federation,omitempty"`
}

// UpdatePlayerRequest is the request body for updating a player
type UpdatePlayerRequest struct {
	FirstName    *string `json:"firstName,omitempty" binding:"omitempty,min=1,max=100"`
	LastName     *string `json:"lastName,omitempty" binding:"omitempty,min=1,max=100"`
	BirthDate    *string `json:"birthDate,omitempty"`
	Gender       *string `json:"gender,omitempty" binding:"omitempty,oneof=m f other"`
	FideID       *string `json:"fideId,omitempty"`
	DsbID        *string `json:"dsbId,omitempty"`
	OesbID       *string `json:"oesbId,omitempty"`
	FideStandard *int    `json:"fideStandard,omitempty"`
	FideRapid    *int    `json:"fideRapid,omitempty"`
	FideBlitz    *int    `json:"fideBlitz,omitempty"`
	DWZ          *int    `json:"dwz,omitempty"`
	OesbElo      *int    `json:"oesbElo,omitempty"`
	FideTitle    *string `json:"fideTitle,omitempty" binding:"omitempty,oneof=GM IM FM CM WGM WIM WFM WCM"`
	ClubID       *string `json:"clubId,omitempty"`
	Federation   *string `json:"federation,omitempty"`
}

// SearchPlayersRequest is the request for searching players
type SearchPlayersRequest struct {
	Query      string  `form:"q"`
	Federation *string `form:"federation" binding:"omitempty,oneof=fide dsb oesb"`
	MinRating  *int    `form:"minRating" binding:"omitempty,min=0,max=3500"`
	MaxRating  *int    `form:"maxRating" binding:"omitempty,min=0,max=3500"`
	ClubID     *string `form:"clubId"`
	Page       int     `form:"page,default=1" binding:"min=1"`
	Limit      int     `form:"limit,default=20" binding:"min=1,max=100"`
}

// PlayerListResponse is the response for listing players
type PlayerListResponse struct {
	Players []Player `json:"players"`
	Total   int64    `json:"total"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
}

// PlayerSearchResult is a player with search relevance score
type PlayerSearchResult struct {
	Player
	Rank float64 `json:"rank,omitempty" db:"rank"`
}

// ValidFideTitles is a list of valid FIDE titles
var ValidFideTitles = []string{"GM", "IM", "FM", "CM", "WGM", "WIM", "WFM", "WCM"}

// ValidGenders is a list of valid gender values
var ValidGenders = []string{"m", "f", "other"}

// GetDisplayName returns the player's full name
func (p *Player) GetDisplayName() string {
	return p.FirstName + " " + p.LastName
}

// GetHighestRating returns the player's highest rating across all rating systems
func (p *Player) GetHighestRating() int {
	var highest int
	if p.FideStandard != nil && *p.FideStandard > highest {
		highest = *p.FideStandard
	}
	if p.DWZ != nil && *p.DWZ > highest {
		highest = *p.DWZ
	}
	if p.OesbElo != nil && *p.OesbElo > highest {
		highest = *p.OesbElo
	}
	return highest
}

// HasFederationID checks if the player has any federation ID
func (p *Player) HasFederationID() bool {
	return (p.FideID != nil && *p.FideID != "") ||
		(p.DsbID != nil && *p.DsbID != "") ||
		(p.OesbID != nil && *p.OesbID != "")
}
