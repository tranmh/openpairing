package export

import (
	"fmt"
	"strings"
)

// PGNGenerator generates PGN format output
type PGNGenerator struct{}

// NewPGNGenerator creates a new PGN generator
func NewPGNGenerator() *PGNGenerator {
	return &PGNGenerator{}
}

// GenerateGame generates PGN for a single game
func (g *PGNGenerator) GenerateGame(game *GameExport) string {
	var sb strings.Builder

	// Write headers
	g.writeHeader(&sb, "Event", game.Header.Event)
	g.writeHeader(&sb, "Site", game.Header.Site)
	g.writeHeader(&sb, "Date", game.Header.Date)
	g.writeHeader(&sb, "Round", game.Header.Round)
	g.writeHeader(&sb, "White", game.Header.White)
	g.writeHeader(&sb, "Black", game.Header.Black)
	g.writeHeader(&sb, "Result", game.Header.Result)

	// Optional headers
	if game.Header.WhiteElo != nil {
		g.writeHeader(&sb, "WhiteElo", fmt.Sprintf("%d", *game.Header.WhiteElo))
	}
	if game.Header.BlackElo != nil {
		g.writeHeader(&sb, "BlackElo", fmt.Sprintf("%d", *game.Header.BlackElo))
	}
	if game.Header.WhiteTitle != nil && *game.Header.WhiteTitle != "" {
		g.writeHeader(&sb, "WhiteTitle", *game.Header.WhiteTitle)
	}
	if game.Header.BlackTitle != nil && *game.Header.BlackTitle != "" {
		g.writeHeader(&sb, "BlackTitle", *game.Header.BlackTitle)
	}
	if game.Header.WhiteFideID != nil && *game.Header.WhiteFideID != "" {
		g.writeHeader(&sb, "WhiteFideId", *game.Header.WhiteFideID)
	}
	if game.Header.BlackFideID != nil && *game.Header.BlackFideID != "" {
		g.writeHeader(&sb, "BlackFideId", *game.Header.BlackFideID)
	}
	if game.Header.ECO != nil && *game.Header.ECO != "" {
		g.writeHeader(&sb, "ECO", *game.Header.ECO)
	}
	if game.Header.Opening != nil && *game.Header.Opening != "" {
		g.writeHeader(&sb, "Opening", *game.Header.Opening)
	}
	if game.Header.TimeControl != "" {
		g.writeHeader(&sb, "TimeControl", game.Header.TimeControl)
	}
	if game.Header.Termination != "" {
		g.writeHeader(&sb, "Termination", game.Header.Termination)
	}
	if game.FEN != "" && game.FEN != "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" {
		g.writeHeader(&sb, "FEN", game.FEN)
		g.writeHeader(&sb, "SetUp", "1")
	}

	// Empty line before moves
	sb.WriteString("\n")

	// Write moves with line wrapping at 80 chars
	if game.Moves != "" {
		sb.WriteString(g.formatMoves(game.Moves))
	}

	// Result at the end
	sb.WriteString(" ")
	sb.WriteString(game.Header.Result)
	sb.WriteString("\n")

	return sb.String()
}

// GenerateTournament generates PGN for all games in a tournament
func (g *PGNGenerator) GenerateTournament(tournament *TournamentExport) string {
	var sb strings.Builder

	// Write tournament header comment
	sb.WriteString(fmt.Sprintf("; Tournament: %s\n", tournament.Name))
	sb.WriteString(fmt.Sprintf("; Site: %s\n", tournament.Site))
	sb.WriteString(fmt.Sprintf("; Date: %s\n", tournament.StartDate.Format("2006.01.02")))
	sb.WriteString(fmt.Sprintf("; Rounds: %d\n", tournament.Rounds))
	sb.WriteString(fmt.Sprintf("; Players: %d\n", tournament.ParticipantCount))
	sb.WriteString("\n")

	// Generate each game
	for i, game := range tournament.Games {
		sb.WriteString(g.GenerateGame(&game))
		if i < len(tournament.Games)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// GenerateGames generates PGN for multiple games
func (g *PGNGenerator) GenerateGames(games []GameExport) string {
	var sb strings.Builder

	for i, game := range games {
		sb.WriteString(g.GenerateGame(&game))
		if i < len(games)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (g *PGNGenerator) writeHeader(sb *strings.Builder, tag, value string) {
	// Escape special characters in value
	escapedValue := strings.ReplaceAll(value, "\\", "\\\\")
	escapedValue = strings.ReplaceAll(escapedValue, "\"", "\\\"")
	sb.WriteString(fmt.Sprintf("[%s \"%s\"]\n", tag, escapedValue))
}

func (g *PGNGenerator) formatMoves(moves string) string {
	// If moves are already in SAN format with move numbers, return as-is
	if strings.Contains(moves, ".") {
		return g.wrapLines(moves, 80)
	}

	// If moves are in UCI format (e2e4 e7e5), we can't convert them here
	// Return as-is with a note
	return moves
}

func (g *PGNGenerator) wrapLines(text string, maxLen int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var sb strings.Builder
	lineLen := 0

	for i, word := range words {
		if i > 0 {
			if lineLen+len(word)+1 > maxLen {
				sb.WriteString("\n")
				lineLen = 0
			} else {
				sb.WriteString(" ")
				lineLen++
			}
		}
		sb.WriteString(word)
		lineLen += len(word)
	}

	return sb.String()
}

// ResultToPGN converts a game result to PGN format
func ResultToPGN(result string) string {
	switch result {
	case "1-0", "1-0F":
		return "1-0"
	case "0-1", "0-1F":
		return "0-1"
	case "1/2-1/2", "=":
		return "1/2-1/2"
	case "0-0F":
		return "0-0" // Double forfeit - rare but valid
	default:
		return "*"
	}
}

// TerminationString converts a termination code to a human-readable string
func TerminationString(termination string) string {
	terminationMap := map[string]string{
		"checkmate":             "Normal",
		"resignation":           "Resigned",
		"timeout":               "Time forfeit",
		"draw_agreement":        "Draw agreed",
		"stalemate":             "Stalemate",
		"insufficient_material": "Insufficient material",
		"threefold_repetition":  "Threefold repetition",
		"fifty_move_rule":       "50 move rule",
		"aborted":               "Abandoned",
		"forfeit":               "Forfeit",
	}

	if term, ok := terminationMap[termination]; ok {
		return term
	}
	return termination
}
