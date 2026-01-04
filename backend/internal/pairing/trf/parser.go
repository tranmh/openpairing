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
			playerNo, xxrData := p.parseXXRLine(line)
			if playerNo > 0 {
				trf.XXRRecords[playerNo] = xxrData
			}
		case "XXC":
			trf.XXCRecords = append(trf.XXCRecords, content)
		}
	}

	return trf, scanner.Err()
}

// parsePlayerLine parses a 001 player line
// Format: 001 SSSS  T  NAME                            RTNG FED   FIDE   BDAY  PTS  RANK RND1 RND2 ...
//         001    1  GM Carlsen, Magnus                 2830 NOR   1503014 1990  7.5     1  w    2  b   15
func (p *Parser) parsePlayerLine(line string) (TRFPlayer, error) {
	player := TRFPlayer{}

	if len(line) < 80 {
		// Pad line to minimum length
		line = line + strings.Repeat(" ", 80-len(line))
	}

	// Start number (columns 5-8)
	startNo, err := strconv.Atoi(strings.TrimSpace(line[4:8]))
	if err != nil {
		return player, fmt.Errorf("invalid start number: %w", err)
	}
	player.StartNo = startNo

	// Sex (column 10)
	if len(line) > 9 {
		player.Sex = strings.TrimSpace(string(line[9]))
	}

	// Title (columns 11-13)
	if len(line) > 13 {
		player.Title = strings.TrimSpace(line[10:14])
	}

	// Name (columns 15-47)
	if len(line) > 47 {
		player.Name = strings.TrimSpace(line[14:47])
	}

	// Rating (columns 49-52)
	if len(line) > 52 {
		rating, _ := strconv.Atoi(strings.TrimSpace(line[48:52]))
		player.Rating = rating
	}

	// Federation (columns 54-56)
	if len(line) > 56 {
		player.Federation = strings.TrimSpace(line[53:56])
	}

	// FIDE ID (columns 58-68)
	if len(line) > 68 {
		player.FideID = strings.TrimSpace(line[57:68])
	}

	// Birth date (columns 70-79)
	if len(line) > 79 {
		player.BirthDate = strings.TrimSpace(line[69:79])
	}

	// Points (columns 81-84)
	if len(line) > 84 {
		pointsStr := strings.TrimSpace(line[80:84])
		points, _ := strconv.ParseFloat(pointsStr, 64)
		player.Points = points
	}

	// Rank (columns 86-89)
	if len(line) > 89 {
		rank, _ := strconv.Atoi(strings.TrimSpace(line[85:89]))
		player.Rank = rank
	}

	// Round results start at column 92 (91 in 0-indexed)
	// Each round result is 8 characters: "  c  opp" where c=color, opp=opponent
	roundStart := 91
	roundWidth := 8
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
// Format: "  c  opp" or " res opp" where res includes color
func (p *Parser) parseRoundResult(round int, data string) TRFRoundResult {
	result := TRFRoundResult{Round: round}

	if len(data) < 8 {
		data = data + strings.Repeat(" ", 8-len(data))
	}

	// Color is at position 2 (0-indexed)
	color := strings.TrimSpace(string(data[2]))
	if color == "w" || color == "b" || color == "-" {
		result.Color = color
	}

	// Result is at position 3-4 or combined with color
	resultStr := strings.TrimSpace(data[2:5])
	if len(resultStr) > 1 {
		// Parse result character
		switch string(resultStr[len(resultStr)-1]) {
		case "1", "+", "W":
			result.Result = "1"
		case "0", "-", "L":
			result.Result = "0"
		case "=", "D":
			result.Result = "="
		case "H": // Half-point bye
			result.Result = "H"
		case "F": // Forfeit win
			result.Result = "F"
		case "U": // Unpaired
			result.Result = "U"
		case "Z": // Zero-point bye
			result.Result = "Z"
		}
	}

	// Opponent number is at positions 5-7 (0-indexed)
	oppStr := strings.TrimSpace(data[4:8])
	if oppStr != "" && oppStr != "0000" {
		opp, _ := strconv.Atoi(oppStr)
		result.OpponentNo = opp
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
