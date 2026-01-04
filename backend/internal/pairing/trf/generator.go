package trf

import (
	"fmt"
	"strings"
	"time"
)

// Generator generates TRF files
type Generator struct {
	trf *TRFFile
}

// NewGenerator creates a new TRF generator
func NewGenerator() *Generator {
	return &Generator{
		trf: &TRFFile{
			XXRRecords: make(map[int]string),
		},
	}
}

// SetTournamentInfo sets tournament header information
func (g *Generator) SetTournamentInfo(name, city, federation string, rounds int) {
	g.trf.TournamentName = name
	g.trf.City = city
	g.trf.Federation = federation
	g.trf.Rounds = rounds
}

// SetDates sets tournament dates
func (g *Generator) SetDates(start, end time.Time) {
	g.trf.StartDate = start.Format("2006/01/02")
	if !end.IsZero() {
		g.trf.EndDate = end.Format("2006/01/02")
	}
}

// SetArbiters sets the arbiters
func (g *Generator) SetArbiters(chief string, deputies []string) {
	g.trf.ChiefArbiter = chief
	g.trf.DeputyArbiters = deputies
}

// SetTimeControl sets the time control string
func (g *Generator) SetTimeControl(tc string) {
	g.trf.TimeControl = tc
}

// SetTournamentType sets the tournament type
func (g *Generator) SetTournamentType(ttype string) {
	g.trf.TournamentType = ttype
}

// SetInitialColors sets the initial color configuration for bbpPairings
func (g *Generator) SetInitialColors(colors ...string) {
	g.trf.XXCRecords = colors
}

// SetRounds sets the number of rounds (XXR record)
func (g *Generator) SetRounds(rounds int) {
	g.trf.Rounds = rounds
}

// AddPlayer adds a player to the TRF file
func (g *Generator) AddPlayer(player TRFPlayer) {
	g.trf.Players = append(g.trf.Players, player)
}

// AddTeam adds a team to the TRF file
func (g *Generator) AddTeam(team TRFTeam) {
	g.trf.Teams = append(g.trf.Teams, team)
}

// Generate creates the TRF string
func (g *Generator) Generate() string {
	var lines []string

	// Header
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

	// Note: Standard TRF16 header codes (062, 072, 082, 092, 102, 112) are intentionally
	// omitted here because JaVaFo fails when it encounters unrecognized codes.
	// Both JaVaFo and bbpPairings work correctly with just the XXR record for rounds.

	// XXR record - number of rounds (required by both JaVaFo and bbpPairings)
	if g.trf.Rounds > 0 {
		lines = append(lines, fmt.Sprintf("XXR %d", g.trf.Rounds))
	}

	// XXC records - initial colors (required by bbpPairings)
	// Add before players section
	for _, xxc := range g.trf.XXCRecords {
		lines = append(lines, fmt.Sprintf("XXC %s", xxc))
	}

	// Blank line before players
	lines = append(lines, "122")

	// Players
	for _, player := range g.trf.Players {
		lines = append(lines, g.generatePlayerLine(player))
	}

	// Teams
	for _, team := range g.trf.Teams {
		lines = append(lines, g.generateTeamLine(team))
	}

	// XXR records (player-specific)
	for startNo, data := range g.trf.XXRRecords {
		lines = append(lines, fmt.Sprintf("XXR %4d %s", startNo, data))
	}

	return strings.Join(lines, "\n")
}

// generatePlayerLine generates a 001 line for a player
// Format: TRF16 (FIDE standard) - compatible with both JaVaFo and bbpPairings
// Positions (1-indexed as per TRF16 spec):
//   1-3: "001", 5-8: startno, 10: sex, 11-13: title, 15-47: name,
//   49-52: rating, 54-56: federation, 58-68: FIDE ID, 70-79: birth date,
//   81-84: points, 86-89: rank, 91+: round results
func (g *Generator) generatePlayerLine(player TRFPlayer) string {
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

// generateRoundResult generates the round result portion
// Format: TRF16 (FIDE standard) - 10 characters per round
// Positions within each round block (1-indexed):
//   1: space, 2-5: opponent number (4 digits), 6: space, 7: color, 8: space, 9: result, 10: space
func (g *Generator) generateRoundResult(result TRFRoundResult) string {
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
func (g *Generator) generateTeamLine(team TRFTeam) string {
	playerNos := make([]string, len(team.PlayerNos))
	for i, no := range team.PlayerNos {
		playerNos[i] = fmt.Sprintf("%d", no)
	}

	return fmt.Sprintf("013 %4d %s %s", team.TeamNo, team.TeamName, strings.Join(playerNos, " "))
}

// CreateFromTournament creates a TRF generator from tournament data
type TournamentData struct {
	Name           string
	City           string
	Federation     string
	StartDate      time.Time
	EndDate        time.Time
	Rounds         int
	TournamentType string
	TimeControl    string
	ChiefArbiter   string
	Players        []PlayerData
}

// PlayerData represents player data for TRF generation
type PlayerData struct {
	StartNo      int
	Sex          string
	Title        string
	FirstName    string
	LastName     string
	Rating       int
	Federation   string
	FideID       string
	BirthDate    string
	Points       float64
	Rank         int
	RoundResults []RoundResultData
}

// RoundResultData represents a round result
type RoundResultData struct {
	Round      int
	OpponentNo int
	Color      string
	Result     string
}

// CreateFromTournament creates a TRF generator from tournament data
func CreateFromTournament(data *TournamentData) *Generator {
	g := NewGenerator()

	g.SetTournamentInfo(data.Name, data.City, data.Federation, data.Rounds)
	g.SetDates(data.StartDate, data.EndDate)
	g.SetTournamentType(data.TournamentType)
	g.SetTimeControl(data.TimeControl)
	g.SetArbiters(data.ChiefArbiter, nil)

	for _, p := range data.Players {
		player := TRFPlayer{
			StartNo:    p.StartNo,
			Sex:        p.Sex,
			Title:      p.Title,
			Name:       fmt.Sprintf("%s, %s", p.LastName, p.FirstName),
			Rating:     p.Rating,
			Federation: p.Federation,
			FideID:     p.FideID,
			BirthDate:  p.BirthDate,
			Points:     p.Points,
			Rank:       p.Rank,
		}

		for _, r := range p.RoundResults {
			player.RoundResults = append(player.RoundResults, TRFRoundResult{
				Round:      r.Round,
				OpponentNo: r.OpponentNo,
				Color:      r.Color,
				Result:     r.Result,
			})
		}

		g.AddPlayer(player)
	}

	return g
}

// ValidateForFIDE validates the TRF file for FIDE submission
func (g *Generator) ValidateForFIDE() []string {
	var errors []string

	if g.trf.TournamentName == "" {
		errors = append(errors, "Tournament name is required")
	}

	if g.trf.Rounds == 0 {
		errors = append(errors, "Number of rounds is required")
	}

	if len(g.trf.Players) == 0 {
		errors = append(errors, "At least one player is required")
	}

	for _, player := range g.trf.Players {
		if player.Name == "" {
			errors = append(errors, fmt.Sprintf("Player %d: name is required", player.StartNo))
		}
		if player.FideID == "" && player.Rating >= 1000 {
			errors = append(errors, fmt.Sprintf("Player %d: FIDE ID is required for rated players", player.StartNo))
		}
	}

	return errors
}
