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
func (g *Generator) generatePlayerLine(player TRFPlayer) string {
	// Fixed width fields according to TRF16 format
	// 001 SSSS  S TT NAME                            RTNG FED FIDE_ID    BDATE      PTS  RANK
	// Positions: 4-7=startno, 9=sex, 10-13=title, 14-46=name, 48-51=rating, 53-55=fed, 57-67=fideid, 69-78=bdate, 80-83=pts, 85-88=rank

	line := fmt.Sprintf("001 %4d ", player.StartNo)

	// Sex
	sex := player.Sex
	if sex == "" {
		sex = " "
	}
	line += sex + " "

	// Title (3 chars, right-padded)
	title := player.Title
	if len(title) > 3 {
		title = title[:3]
	}
	line += fmt.Sprintf("%-3s", title) + " "

	// Name (33 chars, left-padded)
	name := player.Name
	if len(name) > 33 {
		name = name[:33]
	}
	line += fmt.Sprintf("%-33s", name)

	// Rating (4 digits)
	if player.Rating > 0 {
		line += fmt.Sprintf("%4d", player.Rating)
	} else {
		line += "    "
	}
	line += " "

	// Federation (3 chars)
	fed := player.Federation
	if len(fed) > 3 {
		fed = fed[:3]
	}
	line += fmt.Sprintf("%-3s", fed)
	line += " "

	// FIDE ID (11 chars)
	fideID := player.FideID
	if len(fideID) > 11 {
		fideID = fideID[:11]
	}
	line += fmt.Sprintf("%-11s", fideID)

	// Birth date (10 chars)
	bdate := player.BirthDate
	if len(bdate) > 10 {
		bdate = bdate[:10]
	}
	line += fmt.Sprintf("%-10s", bdate)

	// Points (4 chars with 1 decimal)
	line += fmt.Sprintf("%4.1f", player.Points)
	line += " "

	// Rank (4 digits)
	line += fmt.Sprintf("%4d", player.Rank)

	// Round results
	for _, result := range player.RoundResults {
		line += g.generateRoundResult(result)
	}

	return line
}

// generateRoundResult generates the round result portion
func (g *Generator) generateRoundResult(result TRFRoundResult) string {
	// Format: "  c r ooo" (8 chars total)
	// Position 0-1: padding spaces
	// Position 2: color (w/b/-)
	// Position 3: space
	// Position 4: result (1/0/=/+/-/H/F/U/Z)
	// Position 5-7: opponent number (3 digits, right-justified)

	color := result.Color
	if color == "" {
		color = "-"
	}

	resultChar := result.Result
	if resultChar == "" {
		resultChar = "-"
	}

	// TRF format: 4 chars opponent (right-justified) + space + color + space + result
	// Total: 8 chars per round result
	if result.OpponentNo > 0 {
		return fmt.Sprintf("  %4d %s %s", result.OpponentNo, color, resultChar)
	} else if result.Result == "H" || result.Result == "F" || result.Result == "Z" {
		// Bye (H=half-point, F=full-point, Z=zero-point)
		return fmt.Sprintf("       - %s", resultChar)
	} else {
		// Unpaired
		return fmt.Sprintf("       - U")
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
