package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog"
)

// Knockout implements knockout/elimination pairing
type Knockout struct {
	logger zerolog.Logger
}

// NewKnockout creates a new Knockout pairing engine
func NewKnockout(logger zerolog.Logger) *Knockout {
	return &Knockout{
		logger: logger.With().Str("engine", "knockout").Logger(),
	}
}

// BracketEntry represents an entry in the knockout bracket
type BracketEntry struct {
	Position      int
	Round         int
	PlayerStartNo int
	IsBye         bool
	WinnerOf      [2]int // Positions of matches whose winners feed into this
}

// GeneratePairings generates knockout pairings from TRF data
func (k *Knockout) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	players, err := k.parseTRFPlayers(req.TRFData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRF: %w", err)
	}

	genReq := &GeneralPairingRequest{
		TournamentType: TournamentTypeKnockout,
		Players:        players,
		RoundNumber:    req.RoundNumber,
		KnockoutFormat: KnockoutSingleElimination,
	}

	return k.GenerateGeneralPairings(ctx, genReq)
}

// GenerateGeneralPairings generates knockout tournament pairings
func (k *Knockout) GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	k.logger.Info().
		Int("round", req.RoundNumber).
		Int("players", len(req.Players)).
		Str("format", string(req.KnockoutFormat)).
		Msg("Generating knockout pairings")

	if len(req.Players) < 2 {
		return nil, fmt.Errorf("need at least 2 players for knockout")
	}

	// Get only active players
	activePlayers := k.getActivePlayers(req.Players)
	if len(activePlayers) < 2 {
		return nil, fmt.Errorf("need at least 2 active players")
	}

	var pairings []Pairing
	var err error

	if req.RoundNumber == 1 {
		// First round - generate initial bracket
		pairings, err = k.generateFirstRoundPairings(activePlayers)
	} else {
		// Subsequent rounds - pair winners
		pairings, err = k.generateSubsequentRoundPairings(activePlayers, req.RoundNumber)
	}

	if err != nil {
		return nil, err
	}

	result := &PairingResult{
		Pairings:  pairings,
		TRFOutput: "",
		Logs:      fmt.Sprintf("Knockout round %d generated with %d pairings", req.RoundNumber, len(pairings)),
		Duration:  time.Since(startTime),
	}

	k.logger.Info().
		Int("pairings", len(pairings)).
		Dur("duration", result.Duration).
		Msg("Knockout pairings generated")

	return result, nil
}

// generateFirstRoundPairings generates pairings for the first round
// Uses seeding: 1 vs N, 2 vs N-1, etc. with byes for non-power-of-2
func (k *Knockout) generateFirstRoundPairings(players []Player) ([]Pairing, error) {
	n := len(players)

	// Sort by rating (seed) descending
	sorted := make([]Player, len(players))
	copy(sorted, players)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Rating > sorted[j].Rating
	})

	// Calculate bracket size (next power of 2)
	bracketSize := 1
	for bracketSize < n {
		bracketSize *= 2
	}

	// numByes would be: bracketSize - n (used for bye assignment below)

	// Create seeded bracket positions
	// Standard seeding: 1v16, 8v9, 5v12, 4v13, 3v14, 6v11, 7v10, 2v15 for 16-player bracket
	seeds := k.generateSeededPositions(bracketSize)

	var pairings []Pairing
	board := 1

	// Assign players to seed positions
	seedToPlayer := make(map[int]*Player)
	for i, player := range sorted {
		p := player
		seedToPlayer[i+1] = &p
	}

	// Generate pairings from seeded positions
	for i := 0; i < len(seeds); i += 2 {
		seed1 := seeds[i]
		seed2 := seeds[i+1]

		player1 := seedToPlayer[seed1]
		player2 := seedToPlayer[seed2]

		// Handle byes - higher seeds get byes
		if player1 == nil && player2 == nil {
			continue // Both bye positions, skip
		}

		if player2 == nil {
			// Player 1 gets a bye
			pairings = append(pairings, Pairing{
				Board:        board,
				WhiteStartNo: player1.StartNo,
				BlackStartNo: 0,
				IsBye:        true,
			})
		} else if player1 == nil {
			// Player 2 gets a bye
			pairings = append(pairings, Pairing{
				Board:        board,
				WhiteStartNo: player2.StartNo,
				BlackStartNo: 0,
				IsBye:        true,
			})
		} else {
			// Regular pairing - higher seed (lower number) gets white
			if seed1 < seed2 {
				pairings = append(pairings, Pairing{
					Board:        board,
					WhiteStartNo: player1.StartNo,
					BlackStartNo: player2.StartNo,
					IsBye:        false,
				})
			} else {
				pairings = append(pairings, Pairing{
					Board:        board,
					WhiteStartNo: player2.StartNo,
					BlackStartNo: player1.StartNo,
					IsBye:        false,
				})
			}
		}
		board++
	}

	return pairings, nil
}

// generateSubsequentRoundPairings generates pairings for rounds after the first
// Active players are assumed to be winners from previous round
func (k *Knockout) generateSubsequentRoundPairings(players []Player, round int) ([]Pairing, error) {
	n := len(players)

	if n < 2 {
		return nil, fmt.Errorf("need at least 2 players for round %d", round)
	}

	// Sort by original seeding (rating)
	sorted := make([]Player, len(players))
	copy(sorted, players)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Rating > sorted[j].Rating
	})

	var pairings []Pairing

	// Pair in order: best seed vs worst seed remaining
	for i := 0; i < n/2; i++ {
		p1 := sorted[i]
		p2 := sorted[n-1-i]

		// Alternate colors based on board
		if i%2 == 0 {
			pairings = append(pairings, Pairing{
				Board:        i + 1,
				WhiteStartNo: p1.StartNo,
				BlackStartNo: p2.StartNo,
				IsBye:        false,
			})
		} else {
			pairings = append(pairings, Pairing{
				Board:        i + 1,
				WhiteStartNo: p2.StartNo,
				BlackStartNo: p1.StartNo,
				IsBye:        false,
			})
		}
	}

	// Handle odd player (shouldn't happen in knockout after round 1)
	if n%2 == 1 {
		middle := sorted[n/2]
		pairings = append(pairings, Pairing{
			Board:        n/2 + 1,
			WhiteStartNo: middle.StartNo,
			BlackStartNo: 0,
			IsBye:        true,
		})
	}

	return pairings, nil
}

// generateSeededPositions generates standard bracket seeding
// For a 16-player bracket: 1,16,8,9,4,13,5,12,2,15,7,10,3,14,6,11
func (k *Knockout) generateSeededPositions(size int) []int {
	if size == 2 {
		return []int{1, 2}
	}

	// Recursively build bracket
	half := k.generateSeededPositions(size / 2)
	result := make([]int, size)

	for i, seed := range half {
		result[i*2] = seed
		result[i*2+1] = size + 1 - seed
	}

	return result
}

// getActivePlayers returns only active players
func (k *Knockout) getActivePlayers(players []Player) []Player {
	var active []Player
	for _, p := range players {
		if p.IsActive {
			active = append(active, p)
		}
	}
	return active
}

// parseTRFPlayers parses player data from TRF format
func (k *Knockout) parseTRFPlayers(trfData string) ([]Player, error) {
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

		if len(line) >= 8 {
			fmt.Sscanf(line[4:8], "%d", &player.StartNo)
		}

		if len(line) >= 47 {
			player.Name = trimSpace(line[14:47])
		}

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
func (k *Knockout) Name() string {
	return "Knockout"
}

// IsAvailable always returns true as this is a built-in engine
func (k *Knockout) IsAvailable(ctx context.Context) bool {
	return true
}

// SupportedTournamentTypes returns supported tournament types
func (k *Knockout) SupportedTournamentTypes() []TournamentType {
	return []TournamentType{TournamentTypeKnockout}
}

// GetKnockoutTotalRounds returns the total number of rounds for a knockout
func GetKnockoutTotalRounds(numPlayers int) int {
	if numPlayers <= 1 {
		return 0
	}
	rounds := 0
	n := numPlayers
	for n > 1 {
		n = (n + 1) / 2 // Round up for odd numbers
		rounds++
	}
	return rounds
}

// GenerateBracket generates the full bracket structure for display
func (k *Knockout) GenerateBracket(players []Player) [][]BracketEntry {
	n := len(players)
	totalRounds := GetKnockoutTotalRounds(n)

	bracket := make([][]BracketEntry, totalRounds)

	// Sort players by seed
	sorted := make([]Player, len(players))
	copy(sorted, players)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Rating > sorted[j].Rating
	})

	// First round
	bracketSize := 1
	for bracketSize < n {
		bracketSize *= 2
	}

	seeds := k.generateSeededPositions(bracketSize)
	firstRound := make([]BracketEntry, bracketSize)

	for i, seed := range seeds {
		if seed <= n {
			firstRound[i] = BracketEntry{
				Position:      i + 1,
				Round:         1,
				PlayerStartNo: sorted[seed-1].StartNo,
				IsBye:         false,
			}
		} else {
			firstRound[i] = BracketEntry{
				Position:      i + 1,
				Round:         1,
				PlayerStartNo: 0,
				IsBye:         true,
			}
		}
	}
	bracket[0] = firstRound

	// Subsequent rounds (empty, to be filled as matches complete)
	matchesInRound := bracketSize / 2
	for r := 1; r < totalRounds; r++ {
		matchesInRound /= 2
		if matchesInRound < 1 {
			matchesInRound = 1
		}
		bracket[r] = make([]BracketEntry, matchesInRound*2)
		for i := range bracket[r] {
			bracket[r][i] = BracketEntry{
				Position: i + 1,
				Round:    r + 1,
			}
		}
	}

	return bracket
}
