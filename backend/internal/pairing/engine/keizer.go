package engine

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/rs/zerolog"
)

// Keizer implements the Keizer pairing system
// Popular in Dutch club championships with these features:
// - Dynamic ranking based on performance
// - Points depend on opponent's rank (more points for beating higher-ranked)
// - Attendance/participation matters
// - No fixed number of rounds
type Keizer struct {
	logger        zerolog.Logger
	keizerWeights []float64 // Points awarded based on opponent rank
}

// KeizerConfig contains configuration for Keizer system
type KeizerConfig struct {
	NumPlayers int // Used to calculate default weights
}

// KeizerPlayer extends Player with Keizer-specific data
type KeizerPlayer struct {
	Player
	KeizerRank   int     // Current Keizer ranking (1 = best)
	KeizerPoints float64 // Accumulated Keizer points
	GamesPlayed  int
	Attendance   int // Number of rounds attended
}

// NewKeizer creates a new Keizer pairing engine
func NewKeizer(logger zerolog.Logger) *Keizer {
	return &Keizer{
		logger: logger.With().Str("engine", "keizer").Logger(),
	}
}

// GeneratePairings generates Keizer pairings
func (k *Keizer) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	players, err := k.parseTRFPlayers(req.TRFData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRF: %w", err)
	}

	genReq := &GeneralPairingRequest{
		TournamentType: TournamentTypeSwiss,
		Players:        players,
		RoundNumber:    req.RoundNumber,
		PairingSystem:  "keizer",
	}

	return k.GenerateGeneralPairings(ctx, genReq)
}

// GenerateGeneralPairings generates Keizer pairings
func (k *Keizer) GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	k.logger.Info().
		Int("round", req.RoundNumber).
		Int("players", len(req.Players)).
		Msg("Generating Keizer pairings")

	if len(req.Players) < 2 {
		return nil, fmt.Errorf("need at least 2 players")
	}

	// Initialize Keizer weights if not set
	k.initWeights(len(req.Players))

	// Convert to Keizer players
	keizerPlayers := k.convertToKeizerPlayers(req.Players)

	// Get available players for this round
	availablePlayers := k.getAvailablePlayers(keizerPlayers)
	if len(availablePlayers) < 2 {
		return nil, fmt.Errorf("need at least 2 available players")
	}

	// Calculate current Keizer ranking
	k.calculateRanking(keizerPlayers)

	// Generate pairings
	pairings, err := k.generatePairings(availablePlayers)
	if err != nil {
		return nil, err
	}

	result := &PairingResult{
		Pairings:  pairings,
		TRFOutput: "",
		Logs:      fmt.Sprintf("Keizer round %d generated with %d pairings", req.RoundNumber, len(pairings)),
		Duration:  time.Since(startTime),
	}

	k.logger.Info().
		Int("pairings", len(pairings)).
		Dur("duration", result.Duration).
		Msg("Keizer pairings generated")

	return result, nil
}

// initWeights initializes Keizer point weights
// Standard: N points for rank 1, N-1 for rank 2, etc.
// With bonus for wins against higher ranked
func (k *Keizer) initWeights(numPlayers int) {
	k.keizerWeights = make([]float64, numPlayers+1)
	for i := 1; i <= numPlayers; i++ {
		k.keizerWeights[i] = float64(numPlayers - i + 1)
	}
}

// convertToKeizerPlayers converts regular players to Keizer players
func (k *Keizer) convertToKeizerPlayers(players []Player) []KeizerPlayer {
	kPlayers := make([]KeizerPlayer, len(players))
	for i, p := range players {
		kPlayers[i] = KeizerPlayer{
			Player:       p,
			KeizerPoints: p.Points, // Initially use tournament points
			GamesPlayed:  len(p.Opponents),
		}
	}
	return kPlayers
}

// calculateRanking calculates current Keizer ranking
func (k *Keizer) calculateRanking(players []KeizerPlayer) {
	// Sort by Keizer points (desc), then by number of games (desc), then by rating
	sort.Slice(players, func(i, j int) bool {
		if players[i].KeizerPoints != players[j].KeizerPoints {
			return players[i].KeizerPoints > players[j].KeizerPoints
		}
		if players[i].GamesPlayed != players[j].GamesPlayed {
			return players[i].GamesPlayed > players[j].GamesPlayed
		}
		return players[i].Rating > players[j].Rating
	})

	// Assign ranks
	for i := range players {
		players[i].KeizerRank = i + 1
	}
}

// getAvailablePlayers returns players available for this round
func (k *Keizer) getAvailablePlayers(players []KeizerPlayer) []KeizerPlayer {
	var available []KeizerPlayer
	for _, p := range players {
		if p.IsActive {
			available = append(available, p)
		}
	}
	return available
}

// generatePairings generates Keizer pairings
func (k *Keizer) generatePairings(players []KeizerPlayer) ([]Pairing, error) {
	n := len(players)
	paired := make(map[int]bool)
	var pairings []Pairing
	board := 1

	// Sort by Keizer rank
	sort.Slice(players, func(i, j int) bool {
		return players[i].KeizerRank < players[j].KeizerRank
	})

	// Pair using maximum weight matching approach
	// For simplicity, use greedy pairing with rank consideration
	for i := 0; i < n; i++ {
		p1 := players[i]
		if paired[p1.StartNo] {
			continue
		}

		// Find best opponent based on Keizer criteria
		bestOpponent := -1
		bestScore := float64(-1)

		for j := i + 1; j < n; j++ {
			p2 := players[j]
			if paired[p2.StartNo] {
				continue
			}

			// Skip if already played
			if k.havePlayed(p1.Player, p2.Player) {
				continue
			}

			// Calculate pairing score
			// Prefer opponents close in rank but not too close
			rankDiff := math.Abs(float64(p1.KeizerRank - p2.KeizerRank))
			score := k.pairingScore(p1, p2, rankDiff)

			if score > bestScore {
				bestScore = score
				bestOpponent = j
			}
		}

		if bestOpponent >= 0 {
			p2 := players[bestOpponent]

			// Determine colors
			white, black := k.determineColors(p1.Player, p2.Player)

			pairings = append(pairings, Pairing{
				Board:        board,
				WhiteStartNo: white.StartNo,
				BlackStartNo: black.StartNo,
				IsBye:        false,
			})
			board++

			paired[p1.StartNo] = true
			paired[p2.StartNo] = true
		}
	}

	// Handle unpaired players (odd number or all played each other)
	for _, p := range players {
		if !paired[p.StartNo] {
			// Try to find any unpaired opponent (allow rematch if necessary)
			for _, p2 := range players {
				if p2.StartNo != p.StartNo && !paired[p2.StartNo] {
					white, black := k.determineColors(p.Player, p2.Player)
					pairings = append(pairings, Pairing{
						Board:        board,
						WhiteStartNo: white.StartNo,
						BlackStartNo: black.StartNo,
						IsBye:        false,
					})
					board++
					paired[p.StartNo] = true
					paired[p2.StartNo] = true
					break
				}
			}

			// If still unpaired, give bye
			if !paired[p.StartNo] {
				pairings = append(pairings, Pairing{
					Board:        board,
					WhiteStartNo: p.StartNo,
					BlackStartNo: 0,
					IsBye:        true,
				})
				board++
				paired[p.StartNo] = true
			}
		}
	}

	return pairings, nil
}

// pairingScore calculates a score for potential pairing
func (k *Keizer) pairingScore(p1, p2 KeizerPlayer, rankDiff float64) float64 {
	// Prefer:
	// 1. Opponents who haven't played many games together
	// 2. Reasonable rank difference (not too high, not too low)
	// 3. Players with similar games played

	score := 100.0

	// Penalty for very close ranks (want variety)
	if rankDiff < 2 {
		score -= 10
	}

	// Penalty for very far ranks
	if rankDiff > float64(len(k.keizerWeights)/2) {
		score -= rankDiff * 2
	}

	// Bonus for similar number of games played
	gamesDiff := math.Abs(float64(p1.GamesPlayed - p2.GamesPlayed))
	score -= gamesDiff * 5

	return score
}

// havePlayed checks if two players have played each other
func (k *Keizer) havePlayed(p1, p2 Player) bool {
	for _, opp := range p1.Opponents {
		if opp == p2.StartNo {
			return true
		}
	}
	return false
}

// determineColors determines who plays white
func (k *Keizer) determineColors(p1, p2 Player) (Player, Player) {
	// Count colors
	p1Whites := k.countColor(p1, ColorWhite)
	p1Blacks := k.countColor(p1, ColorBlack)
	p2Whites := k.countColor(p2, ColorWhite)
	p2Blacks := k.countColor(p2, ColorBlack)

	p1Diff := p1Whites - p1Blacks
	p2Diff := p2Whites - p2Blacks

	if p1Diff < p2Diff {
		return p1, p2
	}
	if p2Diff < p1Diff {
		return p2, p1
	}

	// Check last color
	p1Last := k.lastColor(p1)
	if p1Last == ColorBlack {
		return p1, p2
	}
	if p1Last == ColorWhite {
		return p2, p1
	}

	// Higher ranked gets white
	if p1.Rating > p2.Rating {
		return p1, p2
	}
	return p2, p1
}

// countColor counts occurrences of a color
func (k *Keizer) countColor(p Player, c Color) int {
	count := 0
	for _, color := range p.ColorHistory {
		if color == c {
			count++
		}
	}
	return count
}

// lastColor returns the last color played
func (k *Keizer) lastColor(p Player) Color {
	if len(p.ColorHistory) == 0 {
		return ColorNone
	}
	return p.ColorHistory[len(p.ColorHistory)-1]
}

// CalculateKeizerPoints calculates points earned for a result
// win = 1.0, draw = 0.5, loss = 0.0
func (k *Keizer) CalculateKeizerPoints(opponentRank int, result float64) float64 {
	if opponentRank < 1 || opponentRank >= len(k.keizerWeights) {
		return 0
	}

	basePoints := k.keizerWeights[opponentRank]

	// Win: full points
	// Draw: half points
	// Loss: zero points (but could give small participation bonus)
	return basePoints * result
}

// parseTRFPlayers parses player data from TRF format
func (k *Keizer) parseTRFPlayers(trfData string) ([]Player, error) {
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

		if len(line) >= 84 {
			fmt.Sscanf(line[80:84], "%f", &player.Points)
		}

		// Parse round results
		player.Opponents, player.ColorHistory = k.parseRoundResults(line)

		if player.StartNo > 0 {
			players = append(players, player)
		}
	}

	if len(players) == 0 {
		return nil, fmt.Errorf("no players found in TRF data")
	}

	return players, nil
}

// parseRoundResults extracts opponents and colors from TRF line
func (k *Keizer) parseRoundResults(line string) ([]int, []Color) {
	var opponents []int
	var colors []Color

	pos := 91
	for pos+10 <= len(line) {
		roundData := line[pos : pos+10]
		if len(trimSpace(roundData)) == 0 {
			break
		}

		var oppNo int
		fmt.Sscanf(roundData[1:5], "%d", &oppNo)
		if oppNo > 0 {
			opponents = append(opponents, oppNo)
		}

		if len(roundData) > 5 {
			switch roundData[5] {
			case 'w', 'W':
				colors = append(colors, ColorWhite)
			case 'b', 'B':
				colors = append(colors, ColorBlack)
			default:
				colors = append(colors, ColorNone)
			}
		}

		pos += 10
	}

	return opponents, colors
}

// Name returns the engine name
func (k *Keizer) Name() string {
	return "Keizer"
}

// IsAvailable always returns true as this is a built-in engine
func (k *Keizer) IsAvailable(ctx context.Context) bool {
	return true
}

// SupportedTournamentTypes returns supported tournament types
func (k *Keizer) SupportedTournamentTypes() []TournamentType {
	return []TournamentType{TournamentTypeSwiss}
}

// GetKeizerStandings returns current standings with Keizer points
func (k *Keizer) GetKeizerStandings(players []KeizerPlayer) []KeizerPlayer {
	standings := make([]KeizerPlayer, len(players))
	copy(standings, players)

	sort.Slice(standings, func(i, j int) bool {
		if standings[i].KeizerPoints != standings[j].KeizerPoints {
			return standings[i].KeizerPoints > standings[j].KeizerPoints
		}
		if standings[i].GamesPlayed != standings[j].GamesPlayed {
			return standings[i].GamesPlayed > standings[j].GamesPlayed
		}
		return standings[i].Rating > standings[j].Rating
	})

	return standings
}
