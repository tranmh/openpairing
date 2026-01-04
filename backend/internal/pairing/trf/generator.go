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

	lines = append(lines, fmt.Sprintf("062 %d", g.trf.Rounds))

	if g.trf.TournamentType != "" {
		lines = append(lines, fmt.Sprintf("072 %s", g.trf.TournamentType))
	}

	if g.trf.ChiefArbiter != "" {
		lines = append(lines, fmt.Sprintf("082 %s", g.trf.ChiefArbiter))
	}

	for _, deputy := range g.trf.DeputyArbiters {
		lines = append(lines, fmt.Sprintf("092 %s", deputy))
	}

	if g.trf.TimeControl != "" {
		lines = append(lines, fmt.Sprintf("102 %s", g.trf.TimeControl))
	}

	if len(g.trf.RoundDates) > 0 {
		lines = append(lines, fmt.Sprintf("112 %s", strings.Join(g.trf.RoundDates, " ")))
	}

	// XXR record - number of rounds (required by bbpPairings)
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
// Format compatible with bbpPairings TRF(x) format:
// Positions: 0-2="001", 4-7=startno, 8-13=spaces, 14-47=name, 48-51=rating,
//            52-79=spaces, 80-83=points, 84-87=spaces, 88+=rank, then round results
func (g *Generator) generatePlayerLine(player TRFPlayer) string {
	// Start with "001" and 4-char right-justified start number
	line := fmt.Sprintf("001 %4d", player.StartNo)

	// 6 spaces (positions 8-13)
	line += "      "

	// Name (34 chars, left-padded, positions 14-47)
	name := player.Name
	if len(name) > 34 {
		name = name[:34]
	}
	line += fmt.Sprintf("%-34s", name)

	// Rating (4 digits, positions 48-51)
	if player.Rating > 0 {
		line += fmt.Sprintf("%4d", player.Rating)
	} else {
		line += "    "
	}

	// 28 spaces for federation/FIDE ID/birth date (positions 52-79)
	line += "                            "

	// Points (4 chars with 1 decimal, positions 80-83)
	line += fmt.Sprintf("%4.1f", player.Points)

	// 4 spaces (positions 84-87)
	line += "    "

	// Rank (positions 88+)
	line += fmt.Sprintf("%d", player.Rank)

	// Round results
	for _, result := range player.RoundResults {
		line += g.generateRoundResult(result)
	}

	return line
}

// generateRoundResult generates the round result portion
// Format compatible with bbpPairings: "    OO c r" (10 chars)
// - 4 leading spaces
// - opponent number (2+ digits, right-justified in remaining space)
// - space
// - color (w/b/-)
// - space
// - result (1/0/=/+/-/H/F/U/Z)
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
		// Format: 4 spaces + 2-char opponent (right-justified) + space + color + space + result
		return fmt.Sprintf("    %2d %s %s", result.OpponentNo, color, resultChar)
	} else if result.Result == "F" || result.Result == "+" {
		// Full-point bye (bbpPairings uses '+' for forfeit win / full bye)
		return fmt.Sprintf("     0 - +")
	} else if result.Result == "H" || result.Result == "Z" {
		// Half-point or zero-point bye - use '-' which bbpPairings interprets as forfeit loss / 0 pts
		return fmt.Sprintf("     0 - -")
	} else {
		// Unpaired - use '-' for no result
		return fmt.Sprintf("     0 - -")
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
