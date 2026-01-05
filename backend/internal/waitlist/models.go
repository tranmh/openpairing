package waitlist

import (
	"time"

	"github.com/google/uuid"
)

// WaitlistEntry represents a player on the tournament waitlist
type WaitlistEntry struct {
	ID             uuid.UUID  `json:"id"`
	TournamentID   uuid.UUID  `json:"tournamentId"`
	PlayerID       uuid.UUID  `json:"playerId"`
	Position       int        `json:"position"`
	StartingRating int        `json:"startingRating,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	NotifiedAt     *time.Time `json:"notifiedAt,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`

	// Joined fields from players table
	PlayerFirstName string  `json:"playerFirstName,omitempty"`
	PlayerLastName  string  `json:"playerLastName,omitempty"`
	PlayerFideID    *string `json:"playerFideId,omitempty"`
	PlayerClubName  *string `json:"playerClubName,omitempty"`
}

// AddToWaitlistRequest is the request body for adding a player to the waitlist
type AddToWaitlistRequest struct {
	PlayerID       uuid.UUID `json:"playerId" binding:"required"`
	StartingRating int       `json:"startingRating,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
}

// WaitlistListResponse wraps a list of waitlist entries
type WaitlistListResponse struct {
	Entries    []WaitlistEntry `json:"entries"`
	Total      int             `json:"total"`
	Tournament *TournamentInfo `json:"tournament,omitempty"`
}

// TournamentInfo contains basic tournament information
type TournamentInfo struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	MaxPlayers *int      `json:"maxPlayers,omitempty"`
	Capacity   int       `json:"capacity"`
}

// PromoteFromWaitlistRequest is the request for promoting a player from waitlist
type PromoteFromWaitlistRequest struct {
	NotifyPlayer  bool `json:"notifyPlayer"`
	ExpiryHours   int  `json:"expiryHours"` // Hours to confirm before spot goes to next person
	AutoPromote   bool `json:"autoPromote"` // Immediately add to participants
}
