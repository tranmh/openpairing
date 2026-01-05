package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog"
)

// TeamSwiss implements team Swiss pairing (like Chess Olympiad)
type TeamSwiss struct {
	logger zerolog.Logger
}

// TeamSwissConfig contains configuration for team Swiss pairing
type TeamSwissConfig struct {
	BoardsPerMatch   int  // Number of boards per team match
	GamePointScoring bool // Use game points for tiebreaks (vs match points only)
}

// NewTeamSwiss creates a new Team Swiss pairing engine
func NewTeamSwiss(logger zerolog.Logger) *TeamSwiss {
	return &TeamSwiss{
		logger: logger.With().Str("engine", "teamswiss").Logger(),
	}
}

// TeamPairing represents a team vs team pairing
type TeamPairing struct {
	Board       int
	HomeTeamID  int
	AwayTeamID  int
	BoardPairs  []Pairing // Individual board pairings
	IsBye       bool
}

// GeneratePairings generates team Swiss pairings from TRF data
func (t *TeamSwiss) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	return nil, fmt.Errorf("Team Swiss requires team data, use GenerateGeneralPairings")
}

// GenerateGeneralPairings generates team Swiss pairings
func (t *TeamSwiss) GenerateGeneralPairings(ctx context.Context, req *GeneralPairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	t.logger.Info().
		Int("round", req.RoundNumber).
		Int("teams", len(req.Teams)).
		Msg("Generating team Swiss pairings")

	if len(req.Teams) < 2 {
		return nil, fmt.Errorf("need at least 2 teams for team Swiss")
	}

	// Sort teams by points, then by rating
	teams := make([]Team, len(req.Teams))
	copy(teams, req.Teams)
	sort.Slice(teams, func(i, j int) bool {
		if teams[i].Points != teams[j].Points {
			return teams[i].Points > teams[j].Points
		}
		return t.teamAvgRating(teams[i]) > t.teamAvgRating(teams[j])
	})

	// Generate team pairings using basic Swiss rules
	teamPairings, err := t.generateTeamPairings(teams, req.RoundNumber)
	if err != nil {
		return nil, err
	}

	// Convert team pairings to individual board pairings
	var allPairings []Pairing
	boardNum := 1

	for _, tp := range teamPairings {
		if tp.IsBye {
			// Bye for the team - could generate individual byes or handle differently
			allPairings = append(allPairings, Pairing{
				Board:        boardNum,
				WhiteStartNo: tp.HomeTeamID, // Use team ID as marker
				BlackStartNo: 0,
				IsBye:        true,
			})
			boardNum++
			continue
		}

		// Generate board pairings for this team match
		homeTeam := t.findTeam(teams, tp.HomeTeamID)
		awayTeam := t.findTeam(teams, tp.AwayTeamID)

		if homeTeam == nil || awayTeam == nil {
			continue
		}

		boardPairings := t.generateBoardPairings(*homeTeam, *awayTeam, req.RoundNumber)
		for _, bp := range boardPairings {
			bp.Board = boardNum
			allPairings = append(allPairings, bp)
			boardNum++
		}
	}

	result := &PairingResult{
		Pairings:  allPairings,
		TRFOutput: "",
		Logs:      fmt.Sprintf("Team Swiss round %d: %d team matches, %d board pairings", req.RoundNumber, len(teamPairings), len(allPairings)),
		Duration:  time.Since(startTime),
	}

	t.logger.Info().
		Int("teamMatches", len(teamPairings)).
		Int("boardPairings", len(allPairings)).
		Dur("duration", result.Duration).
		Msg("Team Swiss pairings generated")

	return result, nil
}

// generateTeamPairings generates team vs team pairings using Swiss method
func (t *TeamSwiss) generateTeamPairings(teams []Team, round int) ([]TeamPairing, error) {
	var pairings []TeamPairing

	// Track which teams have been paired
	paired := make(map[int]bool)

	// Pair teams within score groups
	scoreGroups := t.groupByPoints(teams)

	matchNum := 1
	var floaters []Team // Teams that couldn't find opponents in their group

	for _, group := range scoreGroups {
		// Add floaters from previous score group
		groupWithFloaters := append(floaters, group...)
		floaters = nil

		// Within each score group, pair teams
		unpaired := t.getUnpairedTeams(groupWithFloaters, paired)

		for len(unpaired) >= 2 {
			// Try to pair first unpaired with best available opponent
			team1 := unpaired[0]
			var team2 *Team

			for i := len(unpaired) - 1; i >= 1; i-- {
				candidate := unpaired[i]
				if !t.havePlayed(team1, candidate) {
					team2 = &candidate
					break
				}
			}

			if team2 == nil && len(unpaired) > 1 {
				// No valid opponent found, force pairing with someone
				team2 = &unpaired[len(unpaired)-1]
			}

			if team2 != nil {
				// Determine home/away (alternate based on previous colors)
				homeTeam, awayTeam := t.determineColors(team1, *team2, round)

				pairings = append(pairings, TeamPairing{
					Board:      matchNum,
					HomeTeamID: homeTeam.ID,
					AwayTeamID: awayTeam.ID,
					IsBye:      false,
				})
				matchNum++

				paired[team1.ID] = true
				paired[team2.ID] = true
			} else {
				// Float team1 to next score group
				floaters = append(floaters, team1)
			}

			unpaired = t.getUnpairedTeams(groupWithFloaters, paired)
		}

		// Any remaining unpaired teams in this group float down
		for _, team := range unpaired {
			if !paired[team.ID] {
				floaters = append(floaters, team)
			}
		}
	}

	// Handle odd team (bye) - use remaining floaters or any unpaired team
	for _, team := range floaters {
		if !paired[team.ID] {
			pairings = append(pairings, TeamPairing{
				Board:      matchNum,
				HomeTeamID: team.ID,
				AwayTeamID: 0,
				IsBye:      true,
			})
			paired[team.ID] = true
			matchNum++
			break
		}
	}
	// Also check original teams list for any stragglers
	for _, team := range teams {
		if !paired[team.ID] {
			pairings = append(pairings, TeamPairing{
				Board:      matchNum,
				HomeTeamID: team.ID,
				AwayTeamID: 0,
				IsBye:      true,
			})
			break
		}
	}

	return pairings, nil
}

// generateBoardPairings generates individual board pairings for a team match
func (t *TeamSwiss) generateBoardPairings(home, away Team, round int) []Pairing {
	var pairings []Pairing

	// Sort players by board number
	homePlayers := sortPlayersByBoard(home.Players)
	awayPlayers := sortPlayersByBoard(away.Players)

	// Match players by board order
	maxBoards := len(homePlayers)
	if len(awayPlayers) > maxBoards {
		maxBoards = len(awayPlayers)
	}

	for i := 0; i < maxBoards; i++ {
		var homePlayer, awayPlayer *Player

		if i < len(homePlayers) {
			homePlayer = &homePlayers[i]
		}
		if i < len(awayPlayers) {
			awayPlayer = &awayPlayers[i]
		}

		if homePlayer == nil && awayPlayer == nil {
			continue
		}

		if homePlayer == nil {
			// Away player gets bye
			pairings = append(pairings, Pairing{
				Board:        i + 1,
				WhiteStartNo: awayPlayer.StartNo,
				BlackStartNo: 0,
				IsBye:        true,
			})
			continue
		}

		if awayPlayer == nil {
			// Home player gets bye
			pairings = append(pairings, Pairing{
				Board:        i + 1,
				WhiteStartNo: homePlayer.StartNo,
				BlackStartNo: 0,
				IsBye:        true,
			})
			continue
		}

		// Alternate colors by board: odd boards home=white, even boards home=black
		if (i+1)%2 == 1 {
			pairings = append(pairings, Pairing{
				Board:        i + 1,
				WhiteStartNo: homePlayer.StartNo,
				BlackStartNo: awayPlayer.StartNo,
				IsBye:        false,
			})
		} else {
			pairings = append(pairings, Pairing{
				Board:        i + 1,
				WhiteStartNo: awayPlayer.StartNo,
				BlackStartNo: homePlayer.StartNo,
				IsBye:        false,
			})
		}
	}

	return pairings
}

// groupByPoints groups teams by their points
func (t *TeamSwiss) groupByPoints(teams []Team) [][]Team {
	groups := make(map[float64][]Team)
	var points []float64

	for _, team := range teams {
		if _, exists := groups[team.Points]; !exists {
			points = append(points, team.Points)
		}
		groups[team.Points] = append(groups[team.Points], team)
	}

	// Sort points descending
	sort.Float64s(points)
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}

	result := make([][]Team, len(points))
	for i, p := range points {
		result[i] = groups[p]
	}

	return result
}

// getUnpairedTeams returns teams not yet paired
func (t *TeamSwiss) getUnpairedTeams(teams []Team, paired map[int]bool) []Team {
	var unpaired []Team
	for _, team := range teams {
		if !paired[team.ID] {
			unpaired = append(unpaired, team)
		}
	}
	return unpaired
}

// havePlayed checks if two teams have already played
func (t *TeamSwiss) havePlayed(team1, team2 Team) bool {
	// This would need to check historical results
	// For now, we assume teams track their opponents
	// In practice, this should be passed in the request
	return false
}

// determineColors determines home/away based on previous colors and round
func (t *TeamSwiss) determineColors(team1, team2 Team, round int) (Team, Team) {
	// Simple alternation based on rating
	if t.teamAvgRating(team1) > t.teamAvgRating(team2) {
		if round%2 == 1 {
			return team1, team2
		}
		return team2, team1
	}
	if round%2 == 1 {
		return team2, team1
	}
	return team1, team2
}

// teamAvgRating calculates average rating of a team
func (t *TeamSwiss) teamAvgRating(team Team) float64 {
	if len(team.Players) == 0 {
		return 0
	}
	total := 0
	for _, p := range team.Players {
		total += p.Rating
	}
	return float64(total) / float64(len(team.Players))
}

// findTeam finds a team by ID
func (t *TeamSwiss) findTeam(teams []Team, id int) *Team {
	for i := range teams {
		if teams[i].ID == id {
			return &teams[i]
		}
	}
	return nil
}

// Name returns the engine name
func (t *TeamSwiss) Name() string {
	return "Team Swiss"
}

// IsAvailable always returns true as this is a built-in engine
func (t *TeamSwiss) IsAvailable(ctx context.Context) bool {
	return true
}

// SupportedTournamentTypes returns supported tournament types
func (t *TeamSwiss) SupportedTournamentTypes() []TournamentType {
	return []TournamentType{TournamentTypeTeamSwiss}
}

// ValidateTeams validates team configuration for team Swiss
func (t *TeamSwiss) ValidateTeams(teams []Team) []string {
	var errors []string

	if len(teams) < 2 {
		errors = append(errors, "need at least 2 teams")
	}

	// Check all teams have same number of boards
	var boardCount int
	for i, team := range teams {
		if i == 0 {
			boardCount = len(team.Players)
		} else if len(team.Players) != boardCount {
			errors = append(errors, fmt.Sprintf("team %s has %d players, expected %d",
				team.Name, len(team.Players), boardCount))
		}

		// Check players have board numbers
		for j, p := range team.Players {
			if p.BoardNumber == 0 {
				errors = append(errors, fmt.Sprintf("player %s in team %s has no board number",
					p.Name, team.Name))
			}
			if p.BoardNumber > boardCount {
				errors = append(errors, fmt.Sprintf("player %s has board %d but team only has %d boards",
					p.Name, p.BoardNumber, boardCount))
			}
			// Check for duplicate board numbers
			for k := j + 1; k < len(team.Players); k++ {
				if team.Players[k].BoardNumber == p.BoardNumber {
					errors = append(errors, fmt.Sprintf("duplicate board %d in team %s",
						p.BoardNumber, team.Name))
				}
			}
		}
	}

	return errors
}
