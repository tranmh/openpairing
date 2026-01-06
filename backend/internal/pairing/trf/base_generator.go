package trf

import (
	"fmt"
	"time"
)

// baseGenerator contains shared TRF generation logic used by all engine-specific generators.
// This struct is embedded by JaVaFoGenerator and BBPGenerator.
type baseGenerator struct {
	trf *TRFFile
}

func newBaseGenerator() baseGenerator {
	return baseGenerator{
		trf: &TRFFile{
			XXRRecords: make(map[int]string),
		},
	}
}

// SetTournamentInfo sets tournament header information
func (g *baseGenerator) SetTournamentInfo(name, city, federation string, rounds int) {
	g.trf.TournamentName = name
	g.trf.City = city
	g.trf.Federation = federation
	g.trf.Rounds = rounds
}

// SetDates sets tournament dates
func (g *baseGenerator) SetDates(start, end time.Time) {
	g.trf.StartDate = start.Format("2006/01/02")
	if !end.IsZero() {
		g.trf.EndDate = end.Format("2006/01/02")
	}
}

// SetArbiters sets the arbiters
func (g *baseGenerator) SetArbiters(chief string, deputies []string) {
	g.trf.ChiefArbiter = chief
	g.trf.DeputyArbiters = deputies
}

// SetTimeControl sets the time control string
func (g *baseGenerator) SetTimeControl(tc string) {
	g.trf.TimeControl = tc
}

// SetTournamentType sets the tournament type
func (g *baseGenerator) SetTournamentType(ttype string) {
	g.trf.TournamentType = ttype
}

// SetInitialColors sets initial color preferences (XXC records)
func (g *baseGenerator) SetInitialColors(colors ...string) {
	g.trf.XXCRecords = colors
}

// AddPlayer adds a player to the TRF file
func (g *baseGenerator) AddPlayer(player TRFPlayer) {
	g.trf.Players = append(g.trf.Players, player)
}

// AddTeam adds a team to the TRF file
func (g *baseGenerator) AddTeam(team TRFTeam) {
	g.trf.Teams = append(g.trf.Teams, team)
}

// generateCommonHeader generates header lines common to all engines (012, 022, 032, 042, 052)
func (g *baseGenerator) generateCommonHeader() []string {
	var lines []string

	lines = append(lines, fmt.Sprintf("012 %s", g.trf.TournamentName))

	if g.trf.City != "" {
		lines = append(lines, fmt.Sprintf("022 %s", g.trf.City))
	}

	if g.trf.Federation != "" {
		lines = append(lines, fmt.Sprintf("032 %s", g.trf.Federation))
	}

	if g.trf.StartDate != "" {
		lines = append(lines, fmt.Sprintf("042 %s", g.trf.StartDate))
	}

	if g.trf.EndDate != "" {
		lines = append(lines, fmt.Sprintf("052 %s", g.trf.EndDate))
	}

	return lines
}

// generatePlayerLine generates a 001 line for a player.
// Format: TRF16 (FIDE standard) - compatible with both JaVaFo and bbpPairings
// Positions (1-indexed as per TRF16 spec):
//
//	1-3: "001", 5-8: startno, 10: sex, 11-13: title, 15-47: name,
//	49-52: rating, 54-56: federation, 58-68: FIDE ID, 70-79: birth date,
//	81-84: points, 86-89: rank, 91+: round results
func (g *baseGenerator) generatePlayerLine(player TRFPlayer) string {
	// Build line with fixed-width fields per TRF16 spec
	// Using 0-indexed positions (TRF16 pos - 1)

	// Positions 1-3: "001" (0-2 in 0-indexed)
	line := "001"

	// Position 4: space
	line += " "

	// Positions 5-8: Starting rank number (4 digits, right-justified)
	line += fmt.Sprintf("%4d", player.StartNo)

	// Position 9: space
	line += " "

	// Position 10: Sex (m/w/blank)
	if player.Sex != "" {
		line += string(player.Sex[0])
	} else {
		line += " "
	}

	// Positions 11-13: Title (3 chars, left-justified)
	title := player.Title
	if len(title) > 3 {
		title = title[:3]
	}
	line += fmt.Sprintf("%-3s", title)

	// Position 14: space
	line += " "

	// Positions 15-47: Name (33 chars, left-justified)
	name := player.Name
	if len(name) > 33 {
		name = name[:33]
	}
	line += fmt.Sprintf("%-33s", name)

	// Position 48: space
	line += " "

	// Positions 49-52: FIDE Rating (4 digits, right-justified)
	if player.Rating > 0 {
		line += fmt.Sprintf("%4d", player.Rating)
	} else {
		line += "    "
	}

	// Position 53: space
	line += " "

	// Positions 54-56: FIDE Federation (3 chars)
	fed := player.Federation
	if len(fed) > 3 {
		fed = fed[:3]
	}
	line += fmt.Sprintf("%-3s", fed)

	// Position 57: space
	line += " "

	// Positions 58-68: FIDE Number (11 chars, right-justified)
	fideID := player.FideID
	if len(fideID) > 11 {
		fideID = fideID[:11]
	}
	line += fmt.Sprintf("%11s", fideID)

	// Position 69: space
	line += " "

	// Positions 70-79: Birth Date (10 chars, YYYY/MM/DD)
	birthDate := player.BirthDate
	if len(birthDate) > 10 {
		birthDate = birthDate[:10]
	}
	line += fmt.Sprintf("%-10s", birthDate)

	// Position 80: space
	line += " "

	// Positions 81-84: Points (4 chars with 1 decimal, right-justified)
	line += fmt.Sprintf("%4.1f", player.Points)

	// Position 85: space
	line += " "

	// Positions 86-89: Rank (4 digits, right-justified)
	line += fmt.Sprintf("%4d", player.Rank)

	// Positions 90-91: gap before rounds (2 spaces)
	line += "  "

	// Round results start at position 92 (index 91)
	// Each round is 10 characters per TRF16
	for _, result := range player.RoundResults {
		line += g.generateRoundResult(result)
	}

	return line
}

// generateRoundResult generates the round result portion.
// Format: TRF16 (FIDE standard) - 10 characters per round
// Positions within each round block (1-indexed):
//
//	1: space, 2-5: opponent number (4 digits), 6: space, 7: color, 8: space, 9: result, 10: space
func (g *baseGenerator) generateRoundResult(result TRFRoundResult) string {
	color := result.Color
	if color == "" {
		color = "-"
	}

	resultChar := result.Result
	if resultChar == "" {
		resultChar = "-"
	}

	if result.OpponentNo > 0 {
		// Regular game: " OOOO c r " (10 chars)
		return fmt.Sprintf(" %4d %s %s ", result.OpponentNo, color, resultChar)
	} else if result.Result == "F" || result.Result == "+" {
		// Full-point bye / forfeit win
		return fmt.Sprintf(" %4d - + ", 0)
	} else if result.Result == "H" {
		// Half-point bye
		return fmt.Sprintf(" %4d - H ", 0)
	} else if result.Result == "Z" {
		// Zero-point bye (known absence)
		return fmt.Sprintf(" %4d - Z ", 0)
	} else if result.Result == "U" {
		// Pairing-allocated bye (unpaired by system)
		return fmt.Sprintf(" %4d - U ", 0)
	} else {
		// Default: zero-point bye
		return fmt.Sprintf(" %4d - Z ", 0)
	}
}

// generateTeamLine generates a 013 line for a team
func (g *baseGenerator) generateTeamLine(team TRFTeam) string {
	playerNos := make([]string, len(team.PlayerNos))
	for i, no := range team.PlayerNos {
		playerNos[i] = fmt.Sprintf("%d", no)
	}

	return fmt.Sprintf("013 %4d %s %s", team.TeamNo, team.TeamName, joinStrings(playerNos, " "))
}

// generatePlayersSection generates all player lines
func (g *baseGenerator) generatePlayersSection() []string {
	var lines []string
	for _, player := range g.trf.Players {
		lines = append(lines, g.generatePlayerLine(player))
	}
	return lines
}

// generateTeamsSection generates all team lines
func (g *baseGenerator) generateTeamsSection() []string {
	var lines []string
	for _, team := range g.trf.Teams {
		lines = append(lines, g.generateTeamLine(team))
	}
	return lines
}

// generatePlayerXXRRecords generates player-specific XXR records
func (g *baseGenerator) generatePlayerXXRRecords() []string {
	var lines []string
	for startNo, data := range g.trf.XXRRecords {
		lines = append(lines, fmt.Sprintf("XXR %4d %s", startNo, data))
	}
	return lines
}

// baseValidate performs validation common to all engines
func (g *baseGenerator) baseValidate() []ValidationError {
	var errors []ValidationError

	if g.trf.TournamentName == "" {
		errors = append(errors, ValidationError{
			Field:   "TournamentName",
			Message: "Tournament name is required",
			Fatal:   true,
		})
	}

	if g.trf.Rounds == 0 {
		errors = append(errors, ValidationError{
			Field:   "Rounds",
			Message: "Number of rounds is required",
			Fatal:   true,
		})
	}

	if len(g.trf.Players) == 0 {
		errors = append(errors, ValidationError{
			Field:   "Players",
			Message: "At least one player is required",
			Fatal:   true,
		})
	}

	for _, player := range g.trf.Players {
		if player.Name == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("Player[%d].Name", player.StartNo),
				Message: fmt.Sprintf("Player %d: name is required", player.StartNo),
				Fatal:   true,
			})
		}
	}

	return errors
}

// joinStrings joins strings with a separator (to avoid importing strings package)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
