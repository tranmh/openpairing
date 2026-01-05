package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog"
)

// Scheveningen implements the Scheveningen pairing system
// In Scheveningen, two teams play each other with each player
// playing against every player from the opposing team
type Scheveningen struct {
	logger zerolog.Logger
}

// NewScheveningen creates a new Scheveningen pairing engine
func NewScheveningen(logger zerolog.Logger) *Scheveningen {
	return &Scheveningen{
		logger: logger.With().Str("engine", "scheveningen").Logger(),
	}
}

// GeneratePairings generates Scheveningen pairings from TRF data
func (s *Scheveningen) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	return nil, fmt.Errorf("Scheveningen requires team data, use GenerateGeneralPairings")
}

// GenerateGeneralPairings generates Scheveningen tournament pairings
// Teams must be provided in the request
func (s *Scheveningen) GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	s.logger.Info().
		Int("round", req.RoundNumber).
		Int("teams", len(req.Teams)).
		Msg("Generating Scheveningen pairings")

	if len(req.Teams) != 2 {
		return nil, fmt.Errorf("Scheveningen requires exactly 2 teams, got %d", len(req.Teams))
	}

	team1 := req.Teams[0]
	team2 := req.Teams[1]

	if len(team1.Players) == 0 || len(team2.Players) == 0 {
		return nil, fmt.Errorf("both teams must have at least 1 player")
	}

	// Validate round number
	totalRounds := GetScheveningenTotalRounds(len(team1.Players), len(team2.Players))
	if req.RoundNumber < 1 || req.RoundNumber > totalRounds {
		return nil, fmt.Errorf("invalid round %d, must be between 1 and %d", req.RoundNumber, totalRounds)
	}

	// Sort players within each team by board number
	t1Players := sortPlayersByBoard(team1.Players)
	t2Players := sortPlayersByBoard(team2.Players)

	// Generate pairings using rotation
	pairings := s.generateRoundPairings(t1Players, t2Players, req.RoundNumber)

	result := &PairingResult{
		Pairings:  pairings,
		TRFOutput: "",
		Logs:      fmt.Sprintf("Scheveningen round %d: %s vs %s", req.RoundNumber, team1.Name, team2.Name),
		Duration:  time.Since(startTime),
	}

	s.logger.Info().
		Int("pairings", len(pairings)).
		Dur("duration", result.Duration).
		Msg("Scheveningen pairings generated")

	return result, nil
}

// generateRoundPairings generates pairings for a specific round
// Uses rotation method to ensure each player plays all opponents
func (s *Scheveningen) generateRoundPairings(team1, team2 []Player, roundNumber int) []Pairing {
	n1 := len(team1)
	n2 := len(team2)

	// Determine which team is larger
	var largerTeam, smallerTeam []Player
	if n1 >= n2 {
		largerTeam = team1
		smallerTeam = team2
	} else {
		largerTeam = team2
		smallerTeam = team1
	}

	// Rotation offset for this round (0-indexed)
	offset := roundNumber - 1

	var pairings []Pairing
	board := 1

	// Each player from smaller team plays one player from larger team
	for i, player := range smallerTeam {
		// Calculate opponent index with rotation
		oppIdx := (i + offset) % len(largerTeam)
		opponent := largerTeam[oppIdx]

		// Determine colors - team1 gets white in odd rounds on odd boards
		var whitePlayer, blackPlayer Player
		if (roundNumber%2 == 1) == (board%2 == 1) {
			// Team1 white
			if &team1[0] == &smallerTeam[0] {
				whitePlayer = player
				blackPlayer = opponent
			} else {
				whitePlayer = opponent
				blackPlayer = player
			}
		} else {
			// Team1 black
			if &team1[0] == &smallerTeam[0] {
				whitePlayer = opponent
				blackPlayer = player
			} else {
				whitePlayer = player
				blackPlayer = opponent
			}
		}

		pairings = append(pairings, Pairing{
			Board:        board,
			WhiteStartNo: whitePlayer.StartNo,
			BlackStartNo: blackPlayer.StartNo,
			IsBye:        false,
		})
		board++
	}

	return pairings
}

// GenerateFullSchedule generates the complete Scheveningen schedule
func (s *Scheveningen) GenerateFullSchedule(team1, team2 []Player) [][][2]Player {
	n1 := len(team1)
	n2 := len(team2)
	totalRounds := n1
	if n2 > n1 {
		totalRounds = n2
	}

	schedule := make([][][2]Player, totalRounds)

	for round := 1; round <= totalRounds; round++ {
		pairings := s.generateRoundPairings(team1, team2, round)
		roundSchedule := make([][2]Player, len(pairings))

		// Map pairings back to players
		playerMap := make(map[int]Player)
		for _, p := range team1 {
			playerMap[p.StartNo] = p
		}
		for _, p := range team2 {
			playerMap[p.StartNo] = p
		}

		for i, pairing := range pairings {
			roundSchedule[i] = [2]Player{
				playerMap[pairing.WhiteStartNo],
				playerMap[pairing.BlackStartNo],
			}
		}

		schedule[round-1] = roundSchedule
	}

	return schedule
}

// sortPlayersByBoard sorts players by board number
func sortPlayersByBoard(players []Player) []Player {
	sorted := make([]Player, len(players))
	copy(sorted, players)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BoardNumber < sorted[j].BoardNumber
	})
	return sorted
}

// Name returns the engine name
func (s *Scheveningen) Name() string {
	return "Scheveningen"
}

// IsAvailable always returns true as this is a built-in engine
func (s *Scheveningen) IsAvailable(ctx context.Context) bool {
	return true
}

// SupportedTournamentTypes returns supported tournament types
func (s *Scheveningen) SupportedTournamentTypes() []TournamentType {
	return []TournamentType{TournamentTypeScheveningen}
}

// GetScheveningenTotalRounds returns total rounds for a Scheveningen match
func GetScheveningenTotalRounds(team1Size, team2Size int) int {
	if team1Size > team2Size {
		return team1Size
	}
	return team2Size
}

// ValidateScheveningen validates that teams are properly set up
func (s *Scheveningen) ValidateTeams(team1, team2 Team) []string {
	var errors []string

	if len(team1.Players) == 0 {
		errors = append(errors, "team 1 has no players")
	}
	if len(team2.Players) == 0 {
		errors = append(errors, "team 2 has no players")
	}

	// Check for duplicate start numbers
	seen := make(map[int]bool)
	for _, p := range team1.Players {
		if seen[p.StartNo] {
			errors = append(errors, fmt.Sprintf("duplicate start number %d", p.StartNo))
		}
		seen[p.StartNo] = true
	}
	for _, p := range team2.Players {
		if seen[p.StartNo] {
			errors = append(errors, fmt.Sprintf("duplicate start number %d in teams", p.StartNo))
		}
		seen[p.StartNo] = true
	}

	return errors
}
