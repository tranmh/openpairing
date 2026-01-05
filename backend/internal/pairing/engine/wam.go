package engine

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// WAM implements the WAM (Württembergische Amateurmeisterschaft) pairing system
type WAM struct {
	logger zerolog.Logger
}

// NewWAM creates a new WAM pairing engine
func NewWAM(logger zerolog.Logger) *WAM {
	return &WAM{
		logger: logger.With().Str("engine", "wam").Logger(),
	}
}

// GeneratePairings implements PairingEngine interface (not used for WAM)
func (w *WAM) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	return nil, fmt.Errorf("use GenerateWAMPairings for WAM tournaments")
}

// GenerateGeneralPairings implements GeneralPairingEngine interface
func (w *WAM) GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error) {
	return nil, fmt.Errorf("use GenerateWAMPairings for WAM tournaments")
}

// Name returns the engine name
func (w *WAM) Name() string {
	return "WAM"
}

// IsAvailable always returns true as this is a built-in engine
func (w *WAM) IsAvailable(ctx context.Context) bool {
	return true
}

// SupportedTournamentTypes returns supported tournament types
func (w *WAM) SupportedTournamentTypes() []TournamentType {
	return []TournamentType{TournamentTypeWAM}
}

// AssignGroups assigns players to groups based on configuration
// Returns the created groups and any warnings
func (w *WAM) AssignGroups(players []Player, config *WAMConfig) ([]WAMGroup, []string) {
	var warnings []string
	startTime := time.Now()

	groupSize := config.GroupSize
	if groupSize < 3 {
		groupSize = 4 // Default to 4 if not set
	}

	if len(players) == 0 {
		return nil, []string{"No players to assign"}
	}

	// Sort players based on assignment method
	sorted := make([]Player, len(players))
	copy(sorted, players)

	switch config.AssignmentMethod {
	case WAMAssignmentSequential, WAMAssignmentSerpentine:
		// Sort by rating descending
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Rating > sorted[j].Rating
		})
	case WAMAssignmentRandom:
		rand.Shuffle(len(sorted), func(i, j int) {
			sorted[i], sorted[j] = sorted[j], sorted[i]
		})
	case WAMAssignmentManual:
		// No automatic sorting, return empty groups for manual assignment
		return w.createEmptyGroups(len(players), groupSize), warnings
	}

	// Calculate number of full groups
	numFullGroups := len(sorted) / groupSize
	remainder := len(sorted) % groupSize

	// Determine actual number of groups
	numGroups := numFullGroups
	if remainder > 0 {
		numGroups++
	}

	if numGroups == 0 {
		numGroups = 1
	}

	// Create groups
	groups := make([]WAMGroup, numGroups)
	for g := 0; g < numGroups; g++ {
		groups[g] = WAMGroup{
			ID:          uuid.New(),
			Name:        GroupNameFromNumber(g + 1),
			GroupNumber: g + 1,
			Players:     []WAMGroupPlayer{},
		}
	}

	// Assign players based on method
	switch config.AssignmentMethod {
	case WAMAssignmentSequential:
		w.assignSequential(sorted, groups, groupSize)
	case WAMAssignmentSerpentine:
		w.assignSerpentine(sorted, groups, groupSize)
	case WAMAssignmentRandom:
		w.assignSequential(sorted, groups, groupSize) // Random already shuffled
	}

	// Calculate rounds for each group
	for i := range groups {
		groups[i].RoundsTotal = GetTotalGroupRounds(len(groups[i].Players))
	}

	// Generate warnings
	lastGroup := &groups[len(groups)-1]
	lastGroupSize := len(lastGroup.Players)

	if lastGroupSize != groupSize {
		if lastGroupSize < groupSize {
			warnings = append(warnings,
				fmt.Sprintf("Last group (%s) has %d players instead of %d",
					lastGroup.Name, lastGroupSize, groupSize))
		} else {
			warnings = append(warnings,
				fmt.Sprintf("Last group (%s) has %d players instead of %d",
					lastGroup.Name, lastGroupSize, groupSize))
		}
	}

	// Youth age warning for groups of 6+
	if lastGroupSize >= 6 && config.YouthAgeLimit > 0 {
		warnings = append(warnings,
			fmt.Sprintf("Group %s has %d players - verify all players are under %d years old",
				lastGroup.Name, lastGroupSize, config.YouthAgeLimit))
	}

	w.logger.Info().
		Int("players", len(players)).
		Int("groups", len(groups)).
		Int("group_size", groupSize).
		Str("method", string(config.AssignmentMethod)).
		Dur("duration", time.Since(startTime)).
		Msg("Groups assigned")

	return groups, warnings
}

// assignSequential assigns players sequentially: 1-4 to A, 5-8 to B, etc.
func (w *WAM) assignSequential(players []Player, groups []WAMGroup, groupSize int) {
	for i, player := range players {
		groupIdx := i / groupSize
		if groupIdx >= len(groups) {
			groupIdx = len(groups) - 1 // Put extras in last group
		}

		groups[groupIdx].Players = append(groups[groupIdx].Players, WAMGroupPlayer{
			ID:            uuid.New(),
			GroupID:       groups[groupIdx].ID,
			Name:          player.Name,
			Rating:        player.Rating,
			SortOrder:     len(groups[groupIdx].Players) + 1,
			OriginalIndex: player.StartNo - 1, // Convert 1-indexed StartNo to 0-indexed
		})
	}
}

// assignSerpentine assigns players in snake pattern: 1→A, 2→B, 3→C, 4→D, 5→D, 6→C, 7→B, 8→A, etc.
func (w *WAM) assignSerpentine(players []Player, groups []WAMGroup, groupSize int) {
	numGroups := len(groups)
	forward := true
	groupIdx := 0

	for _, player := range players {
		// Check if group is full, skip if so
		for len(groups[groupIdx].Players) >= groupSize {
			// Move to next group or wrap
			if forward {
				groupIdx++
				if groupIdx >= numGroups {
					// All groups full, add to last group
					groupIdx = numGroups - 1
					break
				}
			} else {
				groupIdx--
				if groupIdx < 0 {
					// Wrap around and continue
					groupIdx = 0
					forward = true
				}
			}
		}

		groups[groupIdx].Players = append(groups[groupIdx].Players, WAMGroupPlayer{
			ID:            uuid.New(),
			GroupID:       groups[groupIdx].ID,
			Name:          player.Name,
			Rating:        player.Rating,
			SortOrder:     len(groups[groupIdx].Players) + 1,
			OriginalIndex: player.StartNo - 1, // Convert 1-indexed StartNo to 0-indexed
		})

		// Move to next group in serpentine pattern
		if forward {
			groupIdx++
			if groupIdx >= numGroups {
				groupIdx = numGroups - 1
				forward = false
			}
		} else {
			groupIdx--
			if groupIdx < 0 {
				groupIdx = 0
				forward = true
			}
		}
	}
}

// createEmptyGroups creates empty groups for manual assignment
func (w *WAM) createEmptyGroups(numPlayers, groupSize int) []WAMGroup {
	numGroups := numPlayers / groupSize
	if numPlayers%groupSize != 0 {
		numGroups++
	}
	if numGroups == 0 {
		numGroups = 1
	}

	groups := make([]WAMGroup, numGroups)
	for i := range groups {
		groups[i] = WAMGroup{
			ID:          uuid.New(),
			Name:        GroupNameFromNumber(i + 1),
			GroupNumber: i + 1,
			Players:     []WAMGroupPlayer{},
		}
	}
	return groups
}

// GenerateGroupPairings generates round-robin pairings for a specific group
func (w *WAM) GenerateGroupPairings(group *WAMGroup, roundNumber int) ([]Pairing, error) {
	startTime := time.Now()

	numPlayers := len(group.Players)
	if numPlayers < 2 {
		return nil, fmt.Errorf("group %s has fewer than 2 players", group.Name)
	}

	totalRounds := GetTotalGroupRounds(numPlayers)
	if roundNumber < 1 || roundNumber > totalRounds {
		return nil, fmt.Errorf("invalid round %d for group with %d rounds", roundNumber, totalRounds)
	}

	// Generate Berger table for this group
	bergerTable := GenerateBergerTable(numPlayers)
	if roundNumber > len(bergerTable) {
		return nil, fmt.Errorf("round %d exceeds Berger table size", roundNumber)
	}

	// Get pairings for this round (0-indexed)
	roundPairs := bergerTable[roundNumber-1]
	pairings := make([]Pairing, 0, len(roundPairs))

	for board, pair := range roundPairs {
		white := pair[0]
		black := pair[1]

		// Check for bye (player index > actual players means bye)
		if numPlayers%2 == 1 && (white > numPlayers || black > numPlayers) {
			// This is a bye pairing
			actualPlayer := white
			if white > numPlayers {
				actualPlayer = black
			}
			pairings = append(pairings, Pairing{
				Board:        board + 1,
				WhiteStartNo: actualPlayer,
				BlackStartNo: 0,
				IsBye:        true,
			})
		} else {
			pairings = append(pairings, Pairing{
				Board:        board + 1,
				WhiteStartNo: white,
				BlackStartNo: black,
				IsBye:        false,
			})
		}
	}

	w.logger.Info().
		Str("group", group.Name).
		Int("round", roundNumber).
		Int("pairings", len(pairings)).
		Dur("duration", time.Since(startTime)).
		Msg("Group pairings generated")

	return pairings, nil
}

// GeneratePlayoffBracket creates the playoff bracket based on group results
func (w *WAM) GeneratePlayoffBracket(groups []WAMGroup, config *WAMConfig) (*WAMPlayoffBracket, error) {
	if !config.EnablePlayoffs {
		return nil, fmt.Errorf("playoffs not enabled for this tournament")
	}

	// Collect qualifiers from each group
	var qualifiers []WAMGroupPlayer
	for _, group := range groups {
		if !group.IsComplete {
			return nil, fmt.Errorf("group %s is not complete", group.Name)
		}

		// Sort players by points
		sorted := make([]WAMGroupPlayer, len(group.Players))
		copy(sorted, group.Players)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Points > sorted[j].Points
		})

		// Take top N from each group
		for i := 0; i < config.PlayoffsTopN && i < len(sorted); i++ {
			sorted[i].IsGroupWinner = (i == 0)
			qualifiers = append(qualifiers, sorted[i])
		}
	}

	// Generate bracket based on number of qualifiers
	numQualifiers := len(qualifiers)
	if numQualifiers < 2 {
		return nil, fmt.Errorf("not enough qualifiers for playoffs: %d", numQualifiers)
	}

	bracket := &WAMPlayoffBracket{
		TournamentID: config.TournamentID,
		Matches:      []WAMPlayoffMatch{},
	}

	// Determine bracket structure
	switch {
	case numQualifiers == 2:
		// Just final
		bracket.Matches = append(bracket.Matches, WAMPlayoffMatch{
			ID:            uuid.New(),
			TournamentID:  config.TournamentID,
			RoundName:     "final",
			MatchNumber:   1,
			Player1ID:     &qualifiers[0].ParticipantID,
			Player2ID:     &qualifiers[1].ParticipantID,
			Player1Source: fmt.Sprintf("group_%s_1", groups[0].Name),
			Player2Source: fmt.Sprintf("group_%s_1", groups[1].Name),
		})

	case numQualifiers == 4:
		// Semifinals + Final (+ optional 3rd place)
		// Semi 1: Group A winner vs Group D winner (or 2nd from B)
		// Semi 2: Group B winner vs Group C winner (or 2nd from A)
		bracket.Matches = append(bracket.Matches,
			WAMPlayoffMatch{
				ID:            uuid.New(),
				TournamentID:  config.TournamentID,
				RoundName:     "semifinal",
				MatchNumber:   1,
				Player1ID:     &qualifiers[0].ParticipantID,
				Player2ID:     &qualifiers[3].ParticipantID,
				Player1Source: "qualifier_1",
				Player2Source: "qualifier_4",
			},
			WAMPlayoffMatch{
				ID:            uuid.New(),
				TournamentID:  config.TournamentID,
				RoundName:     "semifinal",
				MatchNumber:   2,
				Player1ID:     &qualifiers[1].ParticipantID,
				Player2ID:     &qualifiers[2].ParticipantID,
				Player1Source: "qualifier_2",
				Player2Source: "qualifier_3",
			},
			WAMPlayoffMatch{
				ID:           uuid.New(),
				TournamentID: config.TournamentID,
				RoundName:    "final",
				MatchNumber:  1,
				Player1Source: "semifinal_1_winner",
				Player2Source: "semifinal_2_winner",
			},
		)

		if config.ThirdPlaceMatch {
			bracket.Matches = append(bracket.Matches, WAMPlayoffMatch{
				ID:           uuid.New(),
				TournamentID: config.TournamentID,
				RoundName:    "third_place",
				MatchNumber:  1,
				Player1Source: "semifinal_1_loser",
				Player2Source: "semifinal_2_loser",
			})
		}

	default:
		// More complex bracket - use knockout engine patterns
		// For now, simple pairing of qualifiers
		numMatches := numQualifiers / 2
		for i := 0; i < numMatches; i++ {
			p1 := qualifiers[i]
			p2 := qualifiers[numQualifiers-1-i]
			bracket.Matches = append(bracket.Matches, WAMPlayoffMatch{
				ID:            uuid.New(),
				TournamentID:  config.TournamentID,
				RoundName:     "round1",
				MatchNumber:   i + 1,
				Player1ID:     &p1.ParticipantID,
				Player2ID:     &p2.ParticipantID,
				Player1Source: fmt.Sprintf("qualifier_%d", i+1),
				Player2Source: fmt.Sprintf("qualifier_%d", numQualifiers-i),
			})
		}
	}

	w.logger.Info().
		Int("qualifiers", numQualifiers).
		Int("matches", len(bracket.Matches)).
		Bool("third_place", config.ThirdPlaceMatch).
		Msg("Playoff bracket generated")

	return bracket, nil
}

// CalculateGroupStandings calculates standings within a group
// The games parameter should contain all completed games for this group
func (w *WAM) CalculateGroupStandings(group *WAMGroup, games []GameResult, tiebreaks []string) []WAMGroupStanding {
	standings := make([]WAMGroupStanding, len(group.Players))

	// Initialize standings from group players
	playerMap := make(map[int]int) // StartNo -> index
	for i, player := range group.Players {
		playerMap[player.SortOrder] = i
		standings[i] = WAMGroupStanding{
			ParticipantID: player.ParticipantID,
			Name:          player.Name,
			Rating:        player.Rating,
			ClubName:      player.ClubName,
			Points:        player.Points,
			Wins:          player.Wins,
			Draws:         player.Draws,
			Losses:        player.Losses,
		}
	}

	// Build head-to-head matrix for direct encounter tiebreak
	h2h := make(map[int]map[int]float64) // player -> opponent -> score
	for i := range group.Players {
		h2h[i] = make(map[int]float64)
	}

	// Calculate scores from games
	for _, game := range games {
		whiteIdx, whiteOk := playerMap[game.WhiteStartNo]
		blackIdx, blackOk := playerMap[game.BlackStartNo]

		if !whiteOk || !blackOk {
			continue
		}

		// Update head-to-head
		h2h[whiteIdx][blackIdx] = game.WhiteScore
		h2h[blackIdx][whiteIdx] = game.BlackScore

		// Calculate Sonneborn-Berger contribution
		// (opponent's final score * your score against them)
		// This requires a second pass after all results are in
	}

	// Calculate Sonneborn-Berger (simplified version)
	for i := range standings {
		var sb float64
		for j, score := range h2h[i] {
			sb += score * standings[j].Points
		}
		standings[i].SonnebornBerger = sb
	}

	// Sort by points and tiebreaks
	sort.Slice(standings, func(i, j int) bool {
		// Primary: Points
		if standings[i].Points != standings[j].Points {
			return standings[i].Points > standings[j].Points
		}

		// Apply configured tiebreaks
		for _, tb := range tiebreaks {
			switch tb {
			case "direct_encounter":
				// Check head-to-head
				iIdx := playerMap[group.Players[i].SortOrder]
				jIdx := playerMap[group.Players[j].SortOrder]
				if score, ok := h2h[iIdx][jIdx]; ok {
					if score > 0.5 {
						return true
					} else if score < 0.5 {
						return false
					}
				}
			case "sonneborn_berger":
				if standings[i].SonnebornBerger != standings[j].SonnebornBerger {
					return standings[i].SonnebornBerger > standings[j].SonnebornBerger
				}
			case "buchholz":
				if standings[i].Buchholz != standings[j].Buchholz {
					return standings[i].Buchholz > standings[j].Buchholz
				}
			case "wins":
				if standings[i].Wins != standings[j].Wins {
					return standings[i].Wins > standings[j].Wins
				}
			}
		}

		// Fallback to rating
		return standings[i].Rating > standings[j].Rating
	})

	// Assign ranks
	for i := range standings {
		standings[i].Rank = i + 1
	}

	return standings
}

// GameResult is used for calculating standings
type GameResult struct {
	WhiteStartNo int
	BlackStartNo int
	WhiteScore   float64
	BlackScore   float64
}

// DetectClubConflicts identifies same-club players in groups
func (w *WAM) DetectClubConflicts(groups []WAMGroup) []WAMClubConflict {
	var conflicts []WAMClubConflict

	for _, group := range groups {
		clubPlayers := make(map[uuid.UUID][]WAMGroupPlayer)

		for _, player := range group.Players {
			if player.ClubID != nil {
				clubPlayers[*player.ClubID] = append(clubPlayers[*player.ClubID], player)
			}
		}

		for clubID, players := range clubPlayers {
			if len(players) > 1 {
				clubName := ""
				if players[0].ClubName != nil {
					clubName = *players[0].ClubName
				}
				conflicts = append(conflicts, WAMClubConflict{
					GroupID:   group.ID,
					GroupName: group.Name,
					ClubID:    clubID,
					ClubName:  clubName,
					Players:   players,
				})
			}
		}
	}

	return conflicts
}

// CheckAgeWarnings checks for age-related warnings in groups
func (w *WAM) CheckAgeWarnings(groups []WAMGroup, config *WAMConfig, referenceDate time.Time) []WAMAgeWarning {
	var warnings []WAMAgeWarning

	if config.YouthAgeLimit <= 0 {
		return warnings
	}

	for _, group := range groups {
		// Only warn for groups with 6+ players
		if len(group.Players) < 6 {
			continue
		}

		for _, player := range group.Players {
			if player.BirthDate == nil {
				continue
			}

			age := calculateAge(*player.BirthDate, referenceDate)
			if age >= config.YouthAgeLimit {
				warnings = append(warnings, WAMAgeWarning{
					GroupID:   group.ID,
					GroupName: group.Name,
					GroupSize: len(group.Players),
					Player:    player.Name,
					Age:       age,
					AgeLimit:  config.YouthAgeLimit,
					Message: fmt.Sprintf("Player %s (age %d) in group %s with %d players - age limit is %d",
						player.Name, age, group.Name, len(group.Players), config.YouthAgeLimit),
				})
			}
		}
	}

	return warnings
}

// calculateAge calculates age in years
func calculateAge(birthDate, referenceDate time.Time) int {
	years := referenceDate.Year() - birthDate.Year()
	if referenceDate.YearDay() < birthDate.YearDay() {
		years--
	}
	return years
}

// IsU12Player checks if a player is under the age limit at reference date
func IsU12Player(birthDate *time.Time, referenceDate time.Time, ageLimit int) bool {
	if birthDate == nil {
		return false
	}
	return calculateAge(*birthDate, referenceDate) < ageLimit
}

// WAMGroupProposal represents a proposed group configuration
type WAMGroupProposal struct {
	Groups       []ProposedWAMGroup    `json:"groups"`
	TotalPlayers int                   `json:"totalPlayers"`
	Warnings     []string              `json:"warnings,omitempty"`
	U12Proposals []U12GroupProposal    `json:"u12Proposals,omitempty"`
}

// ProposedWAMGroup represents a single proposed group
type ProposedWAMGroup struct {
	Name       string           `json:"name"`
	TargetSize int              `json:"targetSize"`
	Players    []WAMGroupPlayer `json:"players"`
	IsU12Group bool             `json:"isU12Group"`
}

// U12GroupProposal represents a proposal for a U12-only group
type U12GroupProposal struct {
	GroupName        string           `json:"groupName"`
	Players          []WAMGroupPlayer `json:"players"`
	Reason           string           `json:"reason"`
	RequiresApproval bool             `json:"requiresApproval"`
}

// ProposeGroups generates a group proposal without saving
// This allows the organizer to review and confirm before committing
func (w *WAM) ProposeGroups(players []Player, config *WAMConfig, u12Flags map[int]bool, referenceDate time.Time) (*WAMGroupProposal, error) {
	if len(players) == 0 {
		return nil, fmt.Errorf("no players to assign")
	}

	proposal := &WAMGroupProposal{
		TotalPlayers: len(players),
	}

	// Determine group size settings
	defaultSize := config.DefaultGroupSize
	if defaultSize == 0 {
		defaultSize = config.GroupSize
	}
	if defaultSize == 0 {
		defaultSize = 4
	}

	allowedSizes := config.AllowedGroupSizes
	if len(allowedSizes) == 0 {
		allowedSizes = []int{4, 6}
	}

	// Sort players by effective rating (rating + joker)
	sorted := make([]Player, len(players))
	copy(sorted, players)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Rating > sorted[j].Rating
	})

	// Mark U12 players
	u12Indices := []int{}
	for i, player := range sorted {
		isU12 := false
		if player.BirthDate != nil && config.YouthAgeLimit > 0 {
			isU12 = IsU12Player(player.BirthDate, referenceDate, config.YouthAgeLimit)
		}
		// Override with explicit flags if provided
		if flag, ok := u12Flags[player.StartNo]; ok {
			isU12 = flag
		}
		if isU12 {
			u12Indices = append(u12Indices, i)
		}
	}

	// Calculate optimal group distribution
	numPlayers := len(sorted)
	remainder := numPlayers % defaultSize

	// Check if we should propose U12-only groups
	if remainder != 0 && contains(allowedSizes, 6) && len(u12Indices) >= 6 {
		proposal.U12Proposals = w.proposeU12Groups(sorted, u12Indices, remainder, defaultSize, config)
	}

	// Generate standard group proposal
	numFullGroups := numPlayers / defaultSize
	numGroups := numFullGroups
	if remainder > 0 {
		numGroups++
	}
	if numGroups == 0 {
		numGroups = 1
	}

	// Create proposed groups
	for g := 0; g < numGroups; g++ {
		targetSize := defaultSize
		// Last group might have different size
		if g == numGroups-1 && remainder > 0 {
			targetSize = remainder
			// Check if we can use an allowed size
			for _, size := range allowedSizes {
				if remainder <= size && size != defaultSize {
					targetSize = size
					break
				}
			}
		}

		group := ProposedWAMGroup{
			Name:       GroupNameFromNumber(g + 1),
			TargetSize: targetSize,
			Players:    []WAMGroupPlayer{},
			IsU12Group: false,
		}
		proposal.Groups = append(proposal.Groups, group)
	}

	// Assign players using serpentine pattern for balanced groups
	w.assignPlayersToProposal(sorted, proposal, defaultSize, u12Indices, referenceDate, config.YouthAgeLimit)

	// Generate warnings
	for i, group := range proposal.Groups {
		actualSize := len(group.Players)
		if actualSize != group.TargetSize && actualSize != defaultSize {
			proposal.Warnings = append(proposal.Warnings,
				fmt.Sprintf("Gruppe %s hat %d Spieler (Ziel: %d)", group.Name, actualSize, group.TargetSize))
		}
		// Calculate rounds
		proposal.Groups[i].TargetSize = actualSize // Update to actual
	}

	w.logger.Info().
		Int("players", len(players)).
		Int("groups", len(proposal.Groups)).
		Int("u12_count", len(u12Indices)).
		Int("u12_proposals", len(proposal.U12Proposals)).
		Msg("Group proposal generated")

	return proposal, nil
}

// proposeU12Groups suggests 6-player U12-only groups when player count doesn't divide evenly
func (w *WAM) proposeU12Groups(players []Player, u12Indices []int, overflow, defaultSize int, config *WAMConfig) []U12GroupProposal {
	var proposals []U12GroupProposal

	// If we have overflow and enough U12 players, propose a 6-player U12 group
	// This helps balance groups when total doesn't divide by defaultSize
	if overflow > 0 && len(u12Indices) >= 6 {
		// Take the U12 players sorted by rating (already sorted in players slice)
		var proposalPlayers []WAMGroupPlayer
		for i := 0; i < 6 && i < len(u12Indices); i++ {
			idx := u12Indices[i]
			player := players[idx]
			proposalPlayers = append(proposalPlayers, WAMGroupPlayer{
				ID:              uuid.New(),
				Name:            player.Name,
				Rating:          player.Rating,
				EffectiveRating: player.Rating, // No joker for U12 typically
				IsU12:           true,
				OriginalIndex:   player.StartNo - 1, // BUGFIX: Store original index for proper mapping
			})
		}

		proposals = append(proposals, U12GroupProposal{
			GroupName: "U12",
			Players:   proposalPlayers,
			Reason: fmt.Sprintf("Eine separate 6er-Gruppe für U12-Spieler ermöglicht eine gleichmäßige Verteilung der %d Spieler",
				len(players)),
			RequiresApproval: true,
		})
	}

	return proposals
}

// assignPlayersToProposal assigns players to proposed groups using serpentine pattern
func (w *WAM) assignPlayersToProposal(players []Player, proposal *WAMGroupProposal, groupSize int, u12Indices []int, refDate time.Time, ageLimit int) {
	numGroups := len(proposal.Groups)
	forward := true
	groupIdx := 0

	// Create set for faster U12 lookup
	u12Set := make(map[int]bool)
	for _, idx := range u12Indices {
		u12Set[idx] = true
	}

	for i, player := range players {
		// Find next group with space
		for len(proposal.Groups[groupIdx].Players) >= groupSize {
			if forward {
				groupIdx++
				if groupIdx >= numGroups {
					groupIdx = numGroups - 1
					break
				}
			} else {
				groupIdx--
				if groupIdx < 0 {
					groupIdx = 0
					forward = true
				}
			}
		}

		isU12 := u12Set[i]
		var age *int
		if player.BirthDate != nil && ageLimit > 0 {
			a := calculateAge(*player.BirthDate, refDate)
			age = &a
		}

		proposal.Groups[groupIdx].Players = append(proposal.Groups[groupIdx].Players, WAMGroupPlayer{
			ID:              uuid.New(),
			GroupID:         uuid.Nil, // Will be set when confirmed
			Name:            player.Name,
			Rating:          player.Rating,
			EffectiveRating: player.Rating,
			IsU12:           isU12,
			Age:             age,
			BirthDate:       player.BirthDate,
			SortOrder:       len(proposal.Groups[groupIdx].Players) + 1,
			OriginalIndex:   player.StartNo - 1,
		})

		// Serpentine movement
		if forward {
			groupIdx++
			if groupIdx >= numGroups {
				groupIdx = numGroups - 1
				forward = false
			}
		} else {
			groupIdx--
			if groupIdx < 0 {
				groupIdx = 0
				forward = true
			}
		}
	}
}

// contains checks if a slice contains a value
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
