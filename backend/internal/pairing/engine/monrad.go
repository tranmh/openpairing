package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog"
)

// Monrad implements the Monrad pairing system (Scandinavian Swiss variant)
// Key differences from Dutch:
// - Simpler rules, no sophisticated floating
// - In score groups, pairs adjacent players after sorting
// - Color allocation is simpler
type Monrad struct {
	logger zerolog.Logger
}

// NewMonrad creates a new Monrad pairing engine
func NewMonrad(logger zerolog.Logger) *Monrad {
	return &Monrad{
		logger: logger.With().Str("engine", "monrad").Logger(),
	}
}

// GeneratePairings generates Monrad pairings
func (m *Monrad) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	players, err := m.parseTRFPlayers(req.TRFData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRF: %w", err)
	}

	genReq := &GeneralPairingRequest{
		TournamentType: TournamentTypeSwiss,
		Players:        players,
		RoundNumber:    req.RoundNumber,
		PairingSystem:  "monrad",
	}

	return m.GenerateGeneralPairings(ctx, genReq)
}

// GenerateGeneralPairings generates Monrad pairings
func (m *Monrad) GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	m.logger.Info().
		Int("round", req.RoundNumber).
		Int("players", len(req.Players)).
		Msg("Generating Monrad pairings")

	if len(req.Players) < 2 {
		return nil, fmt.Errorf("need at least 2 players")
	}

	// Get active players
	activePlayers := m.getActivePlayers(req.Players)
	if len(activePlayers) < 2 {
		return nil, fmt.Errorf("need at least 2 active players")
	}

	// Sort players by points (desc), then rating (desc), then start number (asc)
	sort.Slice(activePlayers, func(i, j int) bool {
		if activePlayers[i].Points != activePlayers[j].Points {
			return activePlayers[i].Points > activePlayers[j].Points
		}
		if activePlayers[i].Rating != activePlayers[j].Rating {
			return activePlayers[i].Rating > activePlayers[j].Rating
		}
		return activePlayers[i].StartNo < activePlayers[j].StartNo
	})

	// Generate pairings using Monrad method
	pairings, err := m.generatePairings(activePlayers)
	if err != nil {
		return nil, err
	}

	result := &PairingResult{
		Pairings:  pairings,
		TRFOutput: "",
		Logs:      fmt.Sprintf("Monrad round %d generated with %d pairings", req.RoundNumber, len(pairings)),
		Duration:  time.Since(startTime),
	}

	m.logger.Info().
		Int("pairings", len(pairings)).
		Dur("duration", result.Duration).
		Msg("Monrad pairings generated")

	return result, nil
}

// generatePairings generates pairings using Monrad method
func (m *Monrad) generatePairings(players []Player) ([]Pairing, error) {
	paired := make(map[int]bool)
	var pairings []Pairing
	board := 1

	// Group players by score
	scoreGroups := m.groupByScore(players)

	// Process each score group
	var floaters []Player // Players who couldn't be paired in their group

	for _, group := range scoreGroups {
		// Add floaters from previous group
		groupPlayers := append(floaters, group...)
		floaters = nil

		// Sort within group by rating
		sort.Slice(groupPlayers, func(i, j int) bool {
			if groupPlayers[i].Rating != groupPlayers[j].Rating {
				return groupPlayers[i].Rating > groupPlayers[j].Rating
			}
			return groupPlayers[i].StartNo < groupPlayers[j].StartNo
		})

		// Pair adjacent players, avoiding rematches
		for i := 0; i < len(groupPlayers); i++ {
			p1 := groupPlayers[i]
			if paired[p1.StartNo] {
				continue
			}

			// Find best opponent (next unpaired player, avoiding rematch)
			var opponent *Player
			for j := i + 1; j < len(groupPlayers); j++ {
				p2 := groupPlayers[j]
				if paired[p2.StartNo] {
					continue
				}
				if !m.havePlayed(p1, p2) {
					opponent = &p2
					break
				}
			}

			// If no valid opponent found, float to next group instead of forcing rematch
			if opponent == nil {
				floaters = append(floaters, p1)
				continue
			}

			if opponent != nil {
				// Determine colors
				white, black := m.determineColors(p1, *opponent)

				pairings = append(pairings, Pairing{
					Board:        board,
					WhiteStartNo: white.StartNo,
					BlackStartNo: black.StartNo,
					IsBye:        false,
				})
				board++

				paired[p1.StartNo] = true
				paired[opponent.StartNo] = true
			} else {
				// Float to next group
				floaters = append(floaters, p1)
			}
		}
	}

	// Handle remaining floaters (odd player gets bye)
	for _, p := range floaters {
		if !paired[p.StartNo] {
			// Check if player already had a bye
			if p.ByeCount > 0 {
				m.logger.Warn().
					Int("player", p.StartNo).
					Msg("Player receiving second bye")
			}

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

	return pairings, nil
}

// groupByScore groups players by their score
func (m *Monrad) groupByScore(players []Player) [][]Player {
	groups := make(map[float64][]Player)
	var scores []float64

	for _, p := range players {
		if _, exists := groups[p.Points]; !exists {
			scores = append(scores, p.Points)
		}
		groups[p.Points] = append(groups[p.Points], p)
	}

	// Sort scores descending
	sort.Float64s(scores)
	for i, j := 0, len(scores)-1; i < j; i, j = i+1, j-1 {
		scores[i], scores[j] = scores[j], scores[i]
	}

	result := make([][]Player, len(scores))
	for i, s := range scores {
		result[i] = groups[s]
	}

	return result
}

// havePlayed checks if two players have played each other
func (m *Monrad) havePlayed(p1, p2 Player) bool {
	for _, opp := range p1.Opponents {
		if opp == p2.StartNo {
			return true
		}
	}
	return false
}

// determineColors determines who plays white based on color history
func (m *Monrad) determineColors(p1, p2 Player) (Player, Player) {
	// Count colors
	p1Whites := m.countColor(p1, ColorWhite)
	p1Blacks := m.countColor(p1, ColorBlack)
	p2Whites := m.countColor(p2, ColorWhite)
	p2Blacks := m.countColor(p2, ColorBlack)

	// Player with fewer whites gets white
	p1Diff := p1Whites - p1Blacks
	p2Diff := p2Whites - p2Blacks

	if p1Diff < p2Diff {
		return p1, p2
	}
	if p2Diff < p1Diff {
		return p2, p1
	}

	// Equal color balance - check last color
	p1Last := m.lastColor(p1)
	p2Last := m.lastColor(p2)

	if p1Last == ColorBlack && p2Last != ColorBlack {
		return p1, p2
	}
	if p2Last == ColorBlack && p1Last != ColorBlack {
		return p2, p1
	}

	// Still tied - higher rated gets white
	if p1.Rating > p2.Rating {
		return p1, p2
	}
	if p2.Rating > p1.Rating {
		return p2, p1
	}

	// Lower start number gets white
	if p1.StartNo < p2.StartNo {
		return p1, p2
	}
	return p2, p1
}

// countColor counts occurrences of a color in history
func (m *Monrad) countColor(p Player, c Color) int {
	count := 0
	for _, color := range p.ColorHistory {
		if color == c {
			count++
		}
	}
	return count
}

// lastColor returns the last color played
func (m *Monrad) lastColor(p Player) Color {
	if len(p.ColorHistory) == 0 {
		return ColorNone
	}
	return p.ColorHistory[len(p.ColorHistory)-1]
}

// getActivePlayers returns only active players
func (m *Monrad) getActivePlayers(players []Player) []Player {
	var active []Player
	for _, p := range players {
		if p.IsActive {
			active = append(active, p)
		}
	}
	return active
}

// parseTRFPlayers parses player data from TRF format
func (m *Monrad) parseTRFPlayers(trfData string) ([]Player, error) {
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

		// Parse points (columns 81-84)
		if len(line) >= 84 {
			fmt.Sscanf(line[80:84], "%f", &player.Points)
		}

		// Parse round results to get opponents and colors
		player.Opponents, player.ColorHistory = m.parseRoundResults(line)

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
func (m *Monrad) parseRoundResults(line string) ([]int, []Color) {
	var opponents []int
	var colors []Color

	// Round results start at column 92, each round takes 10 chars
	// Format: " XXXXC RRR" where XXXX=opponent, C=color, RRR=result
	pos := 91
	for pos+10 <= len(line) {
		roundData := line[pos : pos+10]
		if len(trimSpace(roundData)) == 0 {
			break
		}

		// Parse opponent number (columns 1-4 of round data)
		var oppNo int
		fmt.Sscanf(roundData[1:5], "%d", &oppNo)
		if oppNo > 0 {
			opponents = append(opponents, oppNo)
		}

		// Parse color (column 5)
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
func (m *Monrad) Name() string {
	return "Monrad"
}

// IsAvailable always returns true as this is a built-in engine
func (m *Monrad) IsAvailable(ctx context.Context) bool {
	return true
}

// SupportedTournamentTypes returns supported tournament types
func (m *Monrad) SupportedTournamentTypes() []TournamentType {
	return []TournamentType{TournamentTypeSwiss}
}
