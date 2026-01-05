package engine

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Arena implements continuous/arena pairing for online rapid play
// Features:
// - Players join/leave dynamically
// - Instant pairing when players become available
// - Performance-based rating matching
// - Streak bonuses (berserk, win streaks)
type Arena struct {
	logger         zerolog.Logger
	mu             sync.RWMutex
	waitingPlayers map[int]*ArenaPlayer
	activePairings map[int]int // playerStartNo -> opponentStartNo
	config         ArenaConfig
}

// ArenaConfig contains configuration for arena pairing
type ArenaConfig struct {
	MinRatingDiff       int           // Minimum acceptable rating difference
	MaxRatingDiff       int           // Maximum rating difference to allow
	MaxWaitTime         time.Duration // Max time before expanding search
	RatingDiffExpansion int           // How much to expand range after wait
	PreventRematches    int           // Number of recent games to check for rematches
	StreakBonusEnabled  bool          // Enable win streak bonuses
}

// ArenaPlayer extends Player with arena-specific data
type ArenaPlayer struct {
	Player
	JoinedAt        time.Time
	LastGameEndedAt time.Time
	WinStreak       int
	LossStreak      int
	ArenaScore      float64 // Arena-specific scoring
	RecentOpponents []int   // Last N opponents
	IsSearching     bool
	SearchStartedAt time.Time
}

// NewArena creates a new Arena pairing engine
func NewArena(logger zerolog.Logger, config *ArenaConfig) *Arena {
	if config == nil {
		config = &ArenaConfig{
			MinRatingDiff:       0,
			MaxRatingDiff:       300,
			MaxWaitTime:         30 * time.Second,
			RatingDiffExpansion: 100,
			PreventRematches:    3,
			StreakBonusEnabled:  true,
		}
	}

	return &Arena{
		logger:         logger.With().Str("engine", "arena").Logger(),
		waitingPlayers: make(map[int]*ArenaPlayer),
		activePairings: make(map[int]int),
		config:         *config,
	}
}

// GeneratePairings generates arena pairings for all waiting players
func (a *Arena) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	players, err := a.parseTRFPlayers(req.TRFData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRF: %w", err)
	}

	genReq := &GeneralPairingRequest{
		TournamentType:   TournamentTypeArena,
		Players:          players,
		RoundNumber:      req.RoundNumber,
		AvailablePlayers: a.getAvailablePlayerNos(),
	}

	return a.GenerateGeneralPairings(ctx, genReq)
}

// GenerateGeneralPairings generates pairings for available players
func (a *Arena) GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	a.logger.Info().
		Int("players", len(req.Players)).
		Int("available", len(req.AvailablePlayers)).
		Msg("Generating arena pairings")

	// Convert to arena players
	arenaPlayers := make([]ArenaPlayer, 0, len(req.Players))
	for _, p := range req.Players {
		if p.IsActive {
			arenaPlayers = append(arenaPlayers, ArenaPlayer{
				Player:      p,
				IsSearching: true,
				ArenaScore:  p.Points,
			})
		}
	}

	if len(arenaPlayers) < 2 {
		return &PairingResult{
			Pairings: nil,
			Logs:     "Not enough players for pairing",
			Duration: time.Since(startTime),
		}, nil
	}

	// Generate pairings
	pairings := a.matchPlayers(arenaPlayers)

	result := &PairingResult{
		Pairings:  pairings,
		TRFOutput: "",
		Logs:      fmt.Sprintf("Arena pairing generated %d matches", len(pairings)),
		Duration:  time.Since(startTime),
	}

	a.logger.Info().
		Int("pairings", len(pairings)).
		Dur("duration", result.Duration).
		Msg("Arena pairings generated")

	return result, nil
}

// matchPlayers creates optimal pairings from available players
func (a *Arena) matchPlayers(players []ArenaPlayer) []Pairing {
	if len(players) < 2 {
		return nil
	}

	// Sort by arena score for matching
	sort.Slice(players, func(i, j int) bool {
		return players[i].ArenaScore > players[j].ArenaScore
	})

	var pairings []Pairing
	paired := make(map[int]bool)
	board := 1

	for i := 0; i < len(players); i++ {
		p1 := players[i]
		if paired[p1.StartNo] {
			continue
		}

		// Find best opponent
		bestMatch := -1
		bestScore := float64(-1e9)

		for j := i + 1; j < len(players); j++ {
			p2 := players[j]
			if paired[p2.StartNo] {
				continue
			}

			// Calculate match quality
			score := a.calculateMatchScore(p1, p2)
			if score > bestScore {
				bestScore = score
				bestMatch = j
			}
		}

		if bestMatch >= 0 {
			p2 := players[bestMatch]

			// Determine colors
			white, black := a.determineColors(p1.Player, p2.Player)

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

	return pairings
}

// calculateMatchScore calculates how good a potential match is
func (a *Arena) calculateMatchScore(p1, p2 ArenaPlayer) float64 {
	// Check for recent rematch
	if a.isRecentRematch(p1, p2) {
		return -1e9
	}

	score := 1000.0

	// Rating difference penalty
	ratingDiff := math.Abs(float64(p1.Rating - p2.Rating))
	if ratingDiff > float64(a.config.MaxRatingDiff) {
		score -= (ratingDiff - float64(a.config.MaxRatingDiff)) * 10
	}

	// Prefer similar arena scores
	scoreDiff := math.Abs(p1.ArenaScore - p2.ArenaScore)
	score -= scoreDiff * 5

	// Wait time bonus (players waiting longer get priority)
	waitTime1 := time.Since(p1.SearchStartedAt).Seconds()
	waitTime2 := time.Since(p2.SearchStartedAt).Seconds()
	score += (waitTime1 + waitTime2) * 2

	// Streak considerations
	if a.config.StreakBonusEnabled {
		// Prefer matching players with similar streaks
		streakDiff := math.Abs(float64(p1.WinStreak - p2.WinStreak))
		score -= streakDiff * 3
	}

	return score
}

// isRecentRematch checks if players recently played each other
func (a *Arena) isRecentRematch(p1, p2 ArenaPlayer) bool {
	// Check p1's recent opponents
	for i := 0; i < len(p1.RecentOpponents) && i < a.config.PreventRematches; i++ {
		if p1.RecentOpponents[i] == p2.StartNo {
			return true
		}
	}
	return false
}

// determineColors determines who plays white
func (a *Arena) determineColors(p1, p2 Player) (Player, Player) {
	// Count colors
	p1Whites := a.countColor(p1, ColorWhite)
	p1Blacks := a.countColor(p1, ColorBlack)
	p2Whites := a.countColor(p2, ColorWhite)
	p2Blacks := a.countColor(p2, ColorBlack)

	p1Diff := p1Whites - p1Blacks
	p2Diff := p2Whites - p2Blacks

	if p1Diff < p2Diff {
		return p1, p2
	}
	if p2Diff < p1Diff {
		return p2, p1
	}

	// Check last color
	if a.lastColor(p1) == ColorBlack {
		return p1, p2
	}
	if a.lastColor(p1) == ColorWhite {
		return p2, p1
	}

	// Random-ish: use rating difference
	if p1.Rating > p2.Rating {
		return p1, p2
	}
	return p2, p1
}

// countColor counts occurrences of a color
func (a *Arena) countColor(p Player, c Color) int {
	count := 0
	for _, color := range p.ColorHistory {
		if color == c {
			count++
		}
	}
	return count
}

// lastColor returns the last color played
func (a *Arena) lastColor(p Player) Color {
	if len(p.ColorHistory) == 0 {
		return ColorNone
	}
	return p.ColorHistory[len(p.ColorHistory)-1]
}

// AddPlayer adds a player to the waiting pool
func (a *Arena) AddPlayer(player Player) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.waitingPlayers[player.StartNo] = &ArenaPlayer{
		Player:          player,
		JoinedAt:        time.Now(),
		IsSearching:     true,
		SearchStartedAt: time.Now(),
		ArenaScore:      player.Points,
	}

	a.logger.Debug().
		Int("player", player.StartNo).
		Msg("Player added to arena pool")
}

// RemovePlayer removes a player from the waiting pool
func (a *Arena) RemovePlayer(startNo int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.waitingPlayers, startNo)

	a.logger.Debug().
		Int("player", startNo).
		Msg("Player removed from arena pool")
}

// GetWaitingCount returns the number of waiting players
func (a *Arena) GetWaitingCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.waitingPlayers)
}

// getAvailablePlayerNos returns start numbers of available players
func (a *Arena) getAvailablePlayerNos() []int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	nos := make([]int, 0, len(a.waitingPlayers))
	for no := range a.waitingPlayers {
		nos = append(nos, no)
	}
	return nos
}

// PairSinglePlayer attempts to find a match for a single player
// Returns the opponent's StartNo or 0 if no match found
func (a *Arena) PairSinglePlayer(player ArenaPlayer) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if already paired
	if _, exists := a.activePairings[player.StartNo]; exists {
		return 0, fmt.Errorf("player already in active game")
	}

	// Find best match from waiting players
	var bestMatch *ArenaPlayer
	bestScore := float64(-1e9)

	for _, candidate := range a.waitingPlayers {
		if candidate.StartNo == player.StartNo {
			continue
		}
		if !candidate.IsSearching {
			continue
		}

		score := a.calculateMatchScore(player, *candidate)
		if score > bestScore {
			bestScore = score
			bestMatch = candidate
		}
	}

	// Check if match is acceptable
	if bestMatch != nil && bestScore > 0 {
		// Mark both as paired
		a.activePairings[player.StartNo] = bestMatch.StartNo
		a.activePairings[bestMatch.StartNo] = player.StartNo

		// Remove from waiting
		delete(a.waitingPlayers, player.StartNo)
		delete(a.waitingPlayers, bestMatch.StartNo)

		return bestMatch.StartNo, nil
	}

	// No match found, add to waiting pool
	a.waitingPlayers[player.StartNo] = &player

	return 0, nil
}

// GameCompleted handles game completion and re-adds players to pool
func (a *Arena) GameCompleted(player1StartNo, player2StartNo int, winner int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Remove from active pairings
	delete(a.activePairings, player1StartNo)
	delete(a.activePairings, player2StartNo)

	a.logger.Debug().
		Int("player1", player1StartNo).
		Int("player2", player2StartNo).
		Int("winner", winner).
		Msg("Arena game completed")
}

// CalculateArenaScore calculates arena score with bonuses
func (a *Arena) CalculateArenaScore(result float64, winStreak int, berserk bool) float64 {
	score := result * 2.0 // Base: 2 for win, 1 for draw, 0 for loss

	// Win streak bonus
	if result == 1.0 && winStreak >= 2 {
		streakBonus := math.Min(float64(winStreak-1)*0.5, 2.0)
		score += streakBonus
	}

	// Berserk bonus (playing with less time)
	if berserk && result >= 0.5 {
		score += 1.0
	}

	return score
}

// parseTRFPlayers parses player data from TRF format
func (a *Arena) parseTRFPlayers(trfData string) ([]Player, error) {
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
func (a *Arena) Name() string {
	return "Arena"
}

// IsAvailable always returns true as this is a built-in engine
func (a *Arena) IsAvailable(ctx context.Context) bool {
	return true
}

// SupportedTournamentTypes returns supported tournament types
func (a *Arena) SupportedTournamentTypes() []TournamentType {
	return []TournamentType{TournamentTypeArena}
}
