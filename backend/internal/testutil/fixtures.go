package testutil

import (
	"time"

	"github.com/google/uuid"
)

// TestPlayer represents a minimal player for testing.
// This avoids importing the player package to prevent circular dependencies.
type TestPlayer struct {
	ID           uuid.UUID
	FirstName    string
	LastName     string
	FideID       *string
	FideStandard *int
	FideRapid    *int
	FideBlitz    *int
	DWZ          *int
	FideTitle    *string
	Federation   *string
	BirthYear    *int
	Gender       *string
}

// TestPlayerOption is a functional option for configuring TestPlayer.
type TestPlayerOption func(*TestPlayer)

// NewTestPlayer creates a new test player with sensible defaults.
func NewTestPlayer(opts ...TestPlayerOption) TestPlayer {
	p := TestPlayer{
		ID:        uuid.New(),
		FirstName: "Test",
		LastName:  "Player",
	}
	for _, opt := range opts {
		opt(&p)
	}
	return p
}

// WithPlayerID sets the player ID.
func WithPlayerID(id uuid.UUID) TestPlayerOption {
	return func(p *TestPlayer) {
		p.ID = id
	}
}

// WithPlayerName sets the player's name.
func WithPlayerName(first, last string) TestPlayerOption {
	return func(p *TestPlayer) {
		p.FirstName = first
		p.LastName = last
	}
}

// WithFideRating sets the player's FIDE standard rating.
func WithFideRating(rating int) TestPlayerOption {
	return func(p *TestPlayer) {
		p.FideStandard = &rating
	}
}

// WithDWZ sets the player's DWZ rating.
func WithDWZ(rating int) TestPlayerOption {
	return func(p *TestPlayer) {
		p.DWZ = &rating
	}
}

// WithFideTitle sets the player's FIDE title.
func WithFideTitle(title string) TestPlayerOption {
	return func(p *TestPlayer) {
		p.FideTitle = &title
	}
}

// WithFederation sets the player's federation.
func WithFederation(fed string) TestPlayerOption {
	return func(p *TestPlayer) {
		p.Federation = &fed
	}
}

// WithBirthYear sets the player's birth year.
func WithBirthYear(year int) TestPlayerOption {
	return func(p *TestPlayer) {
		p.BirthYear = &year
	}
}

// WithGender sets the player's gender.
func WithGender(gender string) TestPlayerOption {
	return func(p *TestPlayer) {
		p.Gender = &gender
	}
}

// Common test players representing typical chess scenarios.
var (
	// GrandmasterPlayer is a typical GM-level player.
	GrandmasterPlayer = NewTestPlayer(
		WithPlayerID(PlayerID1),
		WithPlayerName("Magnus", "Carlsen"),
		WithFideRating(2830),
		WithFideTitle("GM"),
		WithFederation("NOR"),
		WithBirthYear(1990),
		WithGender("m"),
	)

	// InternationalMasterPlayer is a typical IM-level player.
	InternationalMasterPlayer = NewTestPlayer(
		WithPlayerID(PlayerID2),
		WithPlayerName("Test", "IM"),
		WithFideRating(2450),
		WithFideTitle("IM"),
		WithFederation("GER"),
		WithBirthYear(1985),
		WithGender("m"),
	)

	// ClubPlayer is a typical club-level player.
	ClubPlayer = NewTestPlayer(
		WithPlayerID(PlayerID3),
		WithPlayerName("Club", "Player"),
		WithFideRating(1800),
		WithDWZ(1850),
		WithFederation("GER"),
		WithBirthYear(1975),
		WithGender("m"),
	)

	// JuniorPlayer is a typical junior player.
	JuniorPlayer = NewTestPlayer(
		WithPlayerID(PlayerID4),
		WithPlayerName("Junior", "Player"),
		WithFideRating(1500),
		WithFederation("GER"),
		WithBirthYear(2010),
		WithGender("m"),
	)

	// UnratedPlayer is a player without any rating.
	UnratedPlayer = NewTestPlayer(
		WithPlayerID(PlayerID5),
		WithPlayerName("Unrated", "Player"),
		WithFederation("GER"),
		WithBirthYear(2000),
		WithGender("f"),
	)
)

// Create8PlayerTournamentField returns a slice of 8 test players
// with varying ratings, suitable for testing pairing algorithms.
func Create8PlayerTournamentField() []TestPlayer {
	return []TestPlayer{
		NewTestPlayer(WithPlayerID(PlayerID1), WithPlayerName("Player", "One"), WithFideRating(2200)),
		NewTestPlayer(WithPlayerID(PlayerID2), WithPlayerName("Player", "Two"), WithFideRating(2150)),
		NewTestPlayer(WithPlayerID(PlayerID3), WithPlayerName("Player", "Three"), WithFideRating(2100)),
		NewTestPlayer(WithPlayerID(PlayerID4), WithPlayerName("Player", "Four"), WithFideRating(2050)),
		NewTestPlayer(WithPlayerID(PlayerID5), WithPlayerName("Player", "Five"), WithFideRating(2000)),
		NewTestPlayer(WithPlayerID(PlayerID6), WithPlayerName("Player", "Six"), WithFideRating(1950)),
		NewTestPlayer(WithPlayerID(PlayerID7), WithPlayerName("Player", "Seven"), WithFideRating(1900)),
		NewTestPlayer(WithPlayerID(PlayerID8), WithPlayerName("Player", "Eight"), WithFideRating(1850)),
	}
}

// TestTime provides fixed timestamps for consistent test output.
var (
	TestTimeNow   = time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	TestTimeStart = time.Date(2024, 6, 15, 9, 0, 0, 0, time.UTC)
	TestTimeEnd   = time.Date(2024, 6, 15, 18, 0, 0, 0, time.UTC)
)
