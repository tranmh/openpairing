package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog"
)

// RoundRobin implements round robin pairing using Berger tables
type RoundRobin struct {
	logger zerolog.Logger
}

// NewRoundRobin creates a new Round Robin pairing engine
func NewRoundRobin(logger zerolog.Logger) *RoundRobin {
	return &RoundRobin{
		logger: logger.With().Str("engine", "roundrobin").Logger(),
	}
}

// GeneratePairings generates round robin pairings from TRF data
// This is a compatibility wrapper that parses TRF and delegates to GenerateGeneralPairings
func (r *RoundRobin) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	// Parse player count from TRF data
	players, err := r.parseTRFPlayers(req.TRFData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRF: %w", err)
	}

	genReq := &GeneralPairingRequest{
		TournamentType: TournamentTypeRoundRobin,
		Players:        players,
		RoundNumber:    req.RoundNumber,
		TotalRounds:    len(players) - 1,
	}

	return r.GenerateGeneralPairings(ctx, genReq)
}

// GenerateGeneralPairings generates round robin pairings
func (r *RoundRobin) GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	r.logger.Info().
		Int("round", req.RoundNumber).
		Int("players", len(req.Players)).
		Bool("double", req.DoubleRoundRobin).
		Msg("Generating round robin pairings")

	if len(req.Players) < 2 {
		return nil, fmt.Errorf("need at least 2 players for round robin")
	}

	// Sort players by start number
	players := make([]Player, len(req.Players))
	copy(players, req.Players)
	sort.Slice(players, func(i, j int) bool {
		return players[i].StartNo < players[j].StartNo
	})

	// Generate Berger table
	numPlayers := len(players)
	bergerTable := GenerateBergerTable(numPlayers)

	// Calculate which round to use
	roundIdx := req.RoundNumber - 1

	// For double round robin, second half mirrors first half with colors swapped
	isSecondHalf := false
	if req.DoubleRoundRobin {
		halfRounds := len(bergerTable)
		if roundIdx >= halfRounds {
			roundIdx = roundIdx - halfRounds
			isSecondHalf = true
		}
	}

	if roundIdx < 0 || roundIdx >= len(bergerTable) {
		return nil, fmt.Errorf("invalid round number %d for %d players (max rounds: %d)",
			req.RoundNumber, numPlayers, len(bergerTable))
	}

	roundPairings := bergerTable[roundIdx]
	var pairings []Pairing

	for board, pair := range roundPairings {
		whitePos := pair[0] // 1-indexed position
		blackPos := pair[1]

		// Swap colors for second half of double round robin
		if isSecondHalf {
			whitePos, blackPos = blackPos, whitePos
		}

		// Handle bye (position > numPlayers means bye due to odd players)
		if whitePos > numPlayers {
			// Black player gets bye
			blackPlayer := r.getPlayerByPosition(players, blackPos)
			pairings = append(pairings, Pairing{
				Board:        board + 1,
				WhiteStartNo: blackPlayer.StartNo,
				BlackStartNo: 0,
				IsBye:        true,
			})
			continue
		}
		if blackPos > numPlayers {
			// White player gets bye
			whitePlayer := r.getPlayerByPosition(players, whitePos)
			pairings = append(pairings, Pairing{
				Board:        board + 1,
				WhiteStartNo: whitePlayer.StartNo,
				BlackStartNo: 0,
				IsBye:        true,
			})
			continue
		}

		whitePlayer := r.getPlayerByPosition(players, whitePos)
		blackPlayer := r.getPlayerByPosition(players, blackPos)

		pairings = append(pairings, Pairing{
			Board:        board + 1,
			WhiteStartNo: whitePlayer.StartNo,
			BlackStartNo: blackPlayer.StartNo,
			IsBye:        false,
		})
	}

	result := &PairingResult{
		Pairings:  pairings,
		TRFOutput: "",
		Logs:      fmt.Sprintf("Round robin round %d generated with %d pairings", req.RoundNumber, len(pairings)),
		Duration:  time.Since(startTime),
	}

	r.logger.Info().
		Int("pairings", len(pairings)).
		Dur("duration", result.Duration).
		Msg("Round robin pairings generated")

	return result, nil
}

// getPlayerByPosition returns the player at the given 1-indexed position
func (r *RoundRobin) getPlayerByPosition(players []Player, position int) Player {
	if position <= 0 || position > len(players) {
		return Player{}
	}
	return players[position-1]
}

// parseTRFPlayers parses player data from TRF format
func (r *RoundRobin) parseTRFPlayers(trfData string) ([]Player, error) {
	var players []Player
	lines := splitLines(trfData)

	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		if line[:3] != "001" {
			continue
		}

		player := Player{
			IsActive: true,
		}

		// Parse start number (columns 5-8)
		if len(line) >= 8 {
			fmt.Sscanf(line[4:8], "%d", &player.StartNo)
		}

		// Parse name (columns 15-47)
		if len(line) >= 47 {
			player.Name = trimSpace(line[14:47])
		}

		// Parse rating (columns 49-52)
		if len(line) >= 52 {
			fmt.Sscanf(line[48:52], "%d", &player.Rating)
		}

		if player.StartNo > 0 {
			players = append(players, player)
		}
	}

	if len(players) == 0 {
		return nil, fmt.Errorf("no players found in TRF data")
	}

	return players, nil
}

// Name returns the engine name
func (r *RoundRobin) Name() string {
	return "Round Robin"
}

// IsAvailable always returns true as this is a built-in engine
func (r *RoundRobin) IsAvailable(ctx context.Context) bool {
	return true
}

// SupportedTournamentTypes returns supported tournament types
func (r *RoundRobin) SupportedTournamentTypes() []TournamentType {
	return []TournamentType{TournamentTypeRoundRobin}
}

// GetTotalRounds returns the total number of rounds for a round robin
func GetRoundRobinTotalRounds(numPlayers int, doubleRoundRobin bool) int {
	rounds := numPlayers - 1
	if numPlayers%2 == 1 {
		rounds = numPlayers
	}
	if doubleRoundRobin {
		rounds *= 2
	}
	return rounds
}

// ValidateRoundRobinPairings validates that pairings follow round robin rules
func (r *RoundRobin) ValidatePairings(players []Player, pairings []Pairing, roundNumber int) []string {
	var errors []string

	// Check no player appears twice
	seen := make(map[int]bool)
	for _, p := range pairings {
		if p.IsBye {
			if seen[p.WhiteStartNo] {
				errors = append(errors, fmt.Sprintf("player %d appears multiple times", p.WhiteStartNo))
			}
			seen[p.WhiteStartNo] = true
			continue
		}

		if seen[p.WhiteStartNo] {
			errors = append(errors, fmt.Sprintf("player %d appears multiple times", p.WhiteStartNo))
		}
		if seen[p.BlackStartNo] {
			errors = append(errors, fmt.Sprintf("player %d appears multiple times", p.BlackStartNo))
		}
		seen[p.WhiteStartNo] = true
		seen[p.BlackStartNo] = true
	}

	return errors
}
