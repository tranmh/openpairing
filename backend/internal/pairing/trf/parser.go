package trf

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// TRFFile represents a parsed TRF file
type TRFFile struct {
	// Header fields
	TournamentName string
	City           string
	Federation     string
	StartDate      string
	EndDate        string
	Rounds         int
	TournamentType string
	ChiefArbiter   string
	DeputyArbiters []string
	TimeControl    string
	RoundDates     []string

	// Players
	Players []TRFPlayer

	// Teams (for team tournaments)
	Teams []TRFTeam

	// XXR records (additional player info)
	XXRRecords map[int]string

	// XXC records (pairing acceleration)
	XXCRecords []string
}

// TRFPlayer represents a player in TRF format
type TRFPlayer struct {
	StartNo    int
	Sex        string
	Title      string
	Name       string
	Rating     int
	Federation string
	FideID     string
	BirthDate  string
	Points     float64
	Rank       int
	RoundResults []TRFRoundResult
}

// TRFRoundResult represents a player's result in a round
type TRFRoundResult struct {
	Round       int
	OpponentNo  int // 0 = bye, -1 = not paired
	Color       string // "w", "b", or "-"
	Result      string // "1", "0", "=", "+", "-", "H", "F", "U", "Z"
}

// TRFTeam represents a team in TRF format
type TRFTeam struct {
	TeamNo     int
	TeamName   string
	PlayerNos  []int
}

// LineType identifies the type of TRF line
type LineType int

const (
	LineUnknown LineType = iota
	LineTournamentName    // 012
	LineCity              // 022
	LineFederation        // 032
	LineStartDate         // 042
	LineEndDate           // 052
	LineRounds            // 062
	LineTournamentType    // 072
	LineChiefArbiter      // 082
	LineDeputyArbiter     // 092
	LineTimeControl       // 102
	LineRoundDates        // 112
	LinePlayer            // 001
	LineTeam              // 013
	LineXXR               // XXR
	LineXXC               // XXC
	LineBlank             // 122
)

// Parser parses TRF files
type Parser struct {
}

// NewParser creates a new TRF parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a TRF file from a reader
func (p *Parser) Parse(reader io.Reader) (*TRFFile, error) {
	trf := &TRFFile{
		XXRRecords: make(map[int]string),
	}

	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if len(line) < 3 {
			continue
		}

		lineType := line[:3]
		content := ""
		if len(line) > 4 {
			content = line[4:]
		}

		switch lineType {
		case "012":
			trf.TournamentName = strings.TrimSpace(content)
		case "022":
			trf.City = strings.TrimSpace(content)
		case "032":
			trf.Federation = strings.TrimSpace(content)
		case "042":
			trf.StartDate = strings.TrimSpace(content)
		case "052":
			trf.EndDate = strings.TrimSpace(content)
		case "062":
			rounds, _ := strconv.Atoi(strings.TrimSpace(content))
			trf.Rounds = rounds
		case "072":
			trf.TournamentType = strings.TrimSpace(content)
		case "082":
			trf.ChiefArbiter = strings.TrimSpace(content)
		case "092":
			trf.DeputyArbiters = append(trf.DeputyArbiters, strings.TrimSpace(content))
		case "102":
			trf.TimeControl = strings.TrimSpace(content)
		case "112":
			trf.RoundDates = strings.Fields(content)
		case "001":
			player, err := p.parsePlayerLine(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			trf.Players = append(trf.Players, player)
		case "013":
			team, err := p.parseTeamLine(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			trf.Teams = append(trf.Teams, team)
		case "XXR":
			// XXR can be either:
			// - Global round count: "XXR 5" (number of rounds)
			// - Player-specific: "XXR    1 data" (player number + data)
			if len(content) > 0 {
				// Check if it's a simple number (round count)
				if rounds, err := strconv.Atoi(strings.TrimSpace(content)); err == nil && rounds > 0 && len(content) < 5 {
					trf.Rounds = rounds
				} else {
					// Try parsing as player-specific XXR
					playerNo, xxrData := p.parseXXRLine(line)
					if playerNo > 0 {
						trf.XXRRecords[playerNo] = xxrData
					}
				}
			}
		case "XXC":
			trf.XXCRecords = append(trf.XXCRecords, content)
		}
	}

	return trf, scanner.Err()
}

// parsePlayerLine parses a 001 player line
// Format: TRF16 (FIDE standard)
// Positions (1-indexed per TRF16, converted to 0-indexed for Go):
//   1-3 (0-2): "001", 5-8 (4-7): start number, 10 (9): sex
//   11-13 (10-12): title, 15-47 (14-46): name (33 chars)
//   49-52 (48-51): rating, 54-56 (53-55): federation
//   58-68 (57-67): FIDE ID (11 chars), 70-79 (69-78): birth date (10 chars)
//   81-84 (80-83): points, 86-89 (85-88): rank
//   91+ (90+): round results (10 chars each)
func (p *Parser) parsePlayerLine(line string) (TRFPlayer, error) {
	player := TRFPlayer{}

	if len(line) < 48 {
		// Pad line to minimum length for basic parsing
		line = line + strings.Repeat(" ", 48-len(line))
	}

	// Start number (TRF16 positions 5-8, 0-indexed 4-7)
	startNo, err := strconv.Atoi(strings.TrimSpace(line[4:8]))
	if err != nil {
		return player, fmt.Errorf("invalid start number: %w", err)
	}
	player.StartNo = startNo

	// Sex (TRF16 position 10, 0-indexed 9)
	if len(line) > 9 {
		player.Sex = strings.TrimSpace(string(line[9]))
	}

	// Title (TRF16 positions 11-13, 0-indexed 10-12)
	if len(line) > 12 {
		player.Title = strings.TrimSpace(line[10:13])
	}

	// Name (TRF16 positions 15-47, 0-indexed 14-46, 33 chars)
	if len(line) > 46 {
		player.Name = strings.TrimSpace(line[14:47])
	}

	// Rating (TRF16 positions 49-52, 0-indexed 48-51)
	if len(line) > 51 {
		rating, _ := strconv.Atoi(strings.TrimSpace(line[48:52]))
		player.Rating = rating
	}

	// Federation (TRF16 positions 54-56, 0-indexed 53-55)
	if len(line) > 55 {
		player.Federation = strings.TrimSpace(line[53:56])
	}

	// FIDE ID (TRF16 positions 58-68, 0-indexed 57-67, 11 chars)
	if len(line) > 67 {
		player.FideID = strings.TrimSpace(line[57:68])
	}

	// Birth date (TRF16 positions 70-79, 0-indexed 69-78, 10 chars)
	if len(line) > 78 {
		player.BirthDate = strings.TrimSpace(line[69:79])
	}

	// Points (TRF16 positions 81-84, 0-indexed 80-83, 4 chars)
	if len(line) > 83 {
		pointsStr := strings.TrimSpace(line[80:84])
		points, _ := strconv.ParseFloat(pointsStr, 64)
		player.Points = points
	}

	// Rank (TRF16 positions 86-89, 0-indexed 85-88, 4 chars)
	if len(line) > 88 {
		rank, _ := strconv.Atoi(strings.TrimSpace(line[85:89]))
		player.Rank = rank
	}

	// Round results start at TRF16 position 92 (0-indexed 91)
	// Each round result is 10 characters per TRF16
	roundStart := 91
	roundWidth := 10
	roundNum := 1

	for i := roundStart; i+roundWidth <= len(line); i += roundWidth {
		roundData := line[i : i+roundWidth]
		result := p.parseRoundResult(roundNum, roundData)
		player.RoundResults = append(player.RoundResults, result)
		roundNum++
	}

	return player, nil
}

// parseRoundResult parses a single round result
// TRF16 format: 10 characters per round: " OOOO c r "
// Position 1 (0): space
// Positions 2-5 (1-4): opponent number (4 digits, right-justified)
// Position 6 (5): space
// Position 7 (6): color (w/b/-)
// Position 8 (7): space
// Position 9 (8): result (1/0/=/+/-/H/F/U/Z/W/D/L)
// Position 10 (9): space
func (p *Parser) parseRoundResult(round int, data string) TRFRoundResult {
	result := TRFRoundResult{Round: round}

	if len(data) < 10 {
		data = data + strings.Repeat(" ", 10-len(data))
	}

	// Opponent number at positions 2-5 (0-indexed 1-4), 4 digits right-justified
	if len(data) > 4 {
		oppStr := strings.TrimSpace(data[1:5])
		if oppStr != "" && oppStr != "0" && oppStr != "0000" {
			opp, _ := strconv.Atoi(oppStr)
			result.OpponentNo = opp
		}
	}

	// Color at position 7 (0-indexed 6)
	if len(data) > 6 {
		color := string(data[6])
		if color == "w" || color == "b" || color == "-" {
			result.Color = color
		}
	}

	// Result at position 9 (0-indexed 8)
	if len(data) > 8 {
		resultChar := string(data[8])
		switch resultChar {
		case "1", "+", "W":
			result.Result = "1"
		case "0", "L":
			result.Result = "0"
		case "-":
			// Forfeit loss or unpaired
			result.Result = "0"
		case "=", "D":
			result.Result = "="
		case "H": // Half-point bye
			result.Result = "H"
		case "F": // Forfeit win / full-point bye
			result.Result = "F"
		case "U": // Pairing-allocated bye (unpaired by system)
			result.Result = "U"
		case "Z": // Zero-point bye (known absence)
			result.Result = "Z"
		}
	}

	return result
}

// parseTeamLine parses a 013 team line
func (p *Parser) parseTeamLine(line string) (TRFTeam, error) {
	team := TRFTeam{}

	parts := strings.Fields(line)
	if len(parts) < 3 {
		return team, fmt.Errorf("invalid team line")
	}

	// First field after "013" is team number
	teamNo, err := strconv.Atoi(parts[1])
	if err != nil {
		return team, fmt.Errorf("invalid team number: %w", err)
	}
	team.TeamNo = teamNo

	// Find team name (usually in quotes or next field)
	// Parse player numbers
	for i := 2; i < len(parts); i++ {
		if playerNo, err := strconv.Atoi(parts[i]); err == nil {
			team.PlayerNos = append(team.PlayerNos, playerNo)
		} else {
			// Probably team name
			if team.TeamName == "" {
				team.TeamName = parts[i]
			} else {
				team.TeamName += " " + parts[i]
			}
		}
	}

	return team, nil
}

// parseXXRLine parses an XXR record
func (p *Parser) parseXXRLine(line string) (int, string) {
	if len(line) < 8 {
		return 0, ""
	}

	// Format: XXR ssss data
	// ssss = start number
	playerNo, err := strconv.Atoi(strings.TrimSpace(line[4:8]))
	if err != nil {
		return 0, ""
	}

	data := ""
	if len(line) > 9 {
		data = line[9:]
	}

	return playerNo, data
}

// GetPairings extracts pairings for a specific round
func (trf *TRFFile) GetPairings(round int) []Pairing {
	var pairings []Pairing
	paired := make(map[int]bool)

	for _, player := range trf.Players {
		if paired[player.StartNo] {
			continue
		}

		if round-1 < len(player.RoundResults) {
			result := player.RoundResults[round-1]
			if result.OpponentNo > 0 {
				white := player.StartNo
				black := result.OpponentNo

				// Determine actual colors
				if result.Color == "b" {
					white, black = black, white
				}

				pairings = append(pairings, Pairing{
					Board: len(pairings) + 1,
					White: white,
					Black: black,
				})

				paired[player.StartNo] = true
				paired[result.OpponentNo] = true
			} else if result.Result == "H" || result.Result == "Z" || result.Result == "U" {
				// Bye
				pairings = append(pairings, Pairing{
					Board: len(pairings) + 1,
					White: player.StartNo,
					Black: 0,
					IsBye: true,
				})
				paired[player.StartNo] = true
			}
		}
	}

	return pairings
}

// Pairing represents a single pairing
type Pairing struct {
	Board int
	White int // Start number
	Black int // Start number, 0 for bye
	IsBye bool
}
