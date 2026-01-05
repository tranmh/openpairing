package engine

import (
	"context"
	"time"
)

// TournamentType represents the type of tournament for pairing
type TournamentType string

const (
	TournamentTypeSwiss       TournamentType = "swiss"
	TournamentTypeRoundRobin  TournamentType = "round_robin"
	TournamentTypeKnockout    TournamentType = "knockout"
	TournamentTypeTeamSwiss   TournamentType = "team_swiss"
	TournamentTypeScheveningen TournamentType = "scheveningen"
	TournamentTypeArena       TournamentType = "arena"
)

// Player represents a player for pairing purposes
type Player struct {
	StartNo        int
	Name           string
	Rating         int
	Title          string
	FideID         string
	Federation     string
	Points         float64
	ColorHistory   []Color // W, B for each round played
	Opponents      []int   // StartNo of opponents faced
	IsActive       bool
	ByeCount       int
	FloatStatus    FloatStatus
	TeamID         int // For team tournaments
	BoardNumber    int // For team tournaments
}

// Color represents piece color
type Color string

const (
	ColorWhite Color = "W"
	ColorBlack Color = "B"
	ColorNone  Color = ""
)

// FloatStatus indicates player's float status
type FloatStatus string

const (
	FloatNone FloatStatus = ""
	FloatUp   FloatStatus = "U"
	FloatDown FloatStatus = "D"
)

// Team represents a team in team tournaments
type Team struct {
	ID      int
	Name    string
	Players []Player
	Points  float64
}

// GeneralPairingRequest is a more flexible pairing request
type GeneralPairingRequest struct {
	TournamentType  TournamentType
	Players         []Player
	Teams           []Team // For team tournaments
	RoundNumber     int
	TotalRounds     int
	PairingSystem   string // dutch, burstein, monrad, keizer, etc.

	// Swiss-specific
	TRFData         string
	Accelerated     bool

	// Round Robin specific
	DoubleRoundRobin bool

	// Knockout specific
	KnockoutFormat  KnockoutFormat

	// Arena specific
	AvailablePlayers []int // StartNo of players currently available
}

// KnockoutFormat specifies the knockout tournament format
type KnockoutFormat string

const (
	KnockoutSingleElimination KnockoutFormat = "single"
	KnockoutDoubleElimination KnockoutFormat = "double"
)

// GeneralPairingEngine is an extended interface for all pairing types
type GeneralPairingEngine interface {
	PairingEngine
	GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error)
	SupportedTournamentTypes() []TournamentType
}

// MatchResult represents the result of a match (for knockout)
type MatchResult struct {
	WhiteStartNo int
	BlackStartNo int
	WhiteScore   float64
	BlackScore   float64
	WinnerStartNo int
	LoserStartNo  int
}

// BracketPosition represents a position in a knockout bracket
type BracketPosition struct {
	Round    int
	Position int
	PlayerStartNo int
	IsBye    bool
}

// generateBergerTable generates a Berger table for n players
// Returns a slice of rounds, each containing pairs of player positions (1-indexed)
func GenerateBergerTable(n int) [][][2]int {
	// If odd number of players, add a dummy player for byes
	numPlayers := n
	if n%2 == 1 {
		numPlayers = n + 1
	}

	numRounds := numPlayers - 1
	rounds := make([][][2]int, numRounds)

	// Create initial arrangement
	// Fixed player at position 0, others rotate
	players := make([]int, numPlayers)
	for i := 0; i < numPlayers; i++ {
		players[i] = i + 1 // 1-indexed
	}

	for round := 0; round < numRounds; round++ {
		numPairs := numPlayers / 2
		pairs := make([][2]int, numPairs)

		for i := 0; i < numPairs; i++ {
			home := players[i]
			away := players[numPlayers-1-i]

			// Alternate home/away for fairness
			// Player 1 (fixed position) alternates each round
			if i == 0 && round%2 == 1 {
				home, away = away, home
			}

			pairs[i] = [2]int{home, away}
		}

		rounds[round] = pairs

		// Rotate players (keep first player fixed)
		// Move last player to position 1, shift others right
		if numPlayers > 2 {
			last := players[numPlayers-1]
			for i := numPlayers - 1; i > 1; i-- {
				players[i] = players[i-1]
			}
			players[1] = last
		}
	}

	return rounds
}

// calculateByes calculates number of byes needed for knockout
func CalculateByesForKnockout(numPlayers int) int {
	// Find next power of 2
	nextPow2 := 1
	for nextPow2 < numPlayers {
		nextPow2 *= 2
	}
	return nextPow2 - numPlayers
}

// isPowerOfTwo checks if n is a power of 2
func IsPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// BaseEngine provides common functionality for internal pairing engines
type BaseEngine struct {
	logger interface {
		Info() interface{ Msg(string) }
	}
}

// TimedResult wraps result generation with timing
func TimedResult(start time.Time, pairings []Pairing, logs string) *PairingResult {
	return &PairingResult{
		Pairings:  pairings,
		TRFOutput: "",
		Logs:      logs,
		Duration:  time.Since(start),
	}
}
