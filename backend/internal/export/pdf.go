package export

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

// PDFGenerator generates PDF exports for tournament data
type PDFGenerator struct {
	pdf        *fpdf.Fpdf
	pageWidth  float64
	pageHeight float64
	margin     float64
}

// NewPDFGenerator creates a new PDF generator
func NewPDFGenerator() *PDFGenerator {
	return &PDFGenerator{
		margin: 15,
	}
}

// PDFConfig contains configuration for PDF generation
type PDFConfig struct {
	Title       string
	Subtitle    string
	Orientation string // "P" for portrait, "L" for landscape
	PageSize    string // "A4", "Letter", etc.
}

// GenerateStandingsPDF generates a PDF of tournament standings
func (g *PDFGenerator) GenerateStandingsPDF(tournament *TournamentExport) ([]byte, error) {
	// Determine orientation based on number of columns needed
	orientation := "P"
	if len(tournament.Participants) > 0 && tournament.Rounds > 5 {
		orientation = "L"
	}

	g.pdf = fpdf.New(orientation, "mm", "A4", "")
	g.pageWidth, g.pageHeight = g.pdf.GetPageSize()

	g.pdf.SetMargins(g.margin, g.margin, g.margin)
	g.pdf.SetAutoPageBreak(true, g.margin)

	g.pdf.AddPage()

	// Header
	g.addHeader(tournament.Name, "Tournament Standings")

	// Tournament info
	g.addTournamentInfo(tournament)

	// Standings table
	g.addStandingsTable(tournament)

	// Footer
	g.addFooter()

	// Output to bytes
	var buf bytes.Buffer
	err := g.pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateCrosstablePDF generates a PDF of tournament crosstable
func (g *PDFGenerator) GenerateCrosstablePDF(tournament *TournamentExport) ([]byte, error) {
	// Crosstable typically needs landscape for more players
	orientation := "L"
	if len(tournament.Participants) <= 10 {
		orientation = "P"
	}

	g.pdf = fpdf.New(orientation, "mm", "A4", "")
	g.pageWidth, g.pageHeight = g.pdf.GetPageSize()

	g.pdf.SetMargins(g.margin, g.margin, g.margin)
	g.pdf.SetAutoPageBreak(true, g.margin)

	g.pdf.AddPage()

	// Header
	g.addHeader(tournament.Name, "Crosstable")

	// Tournament info
	g.addTournamentInfo(tournament)

	// Crosstable
	g.addCrosstable(tournament)

	// Footer
	g.addFooter()

	// Output to bytes
	var buf bytes.Buffer
	err := g.pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *PDFGenerator) addHeader(title, subtitle string) {
	// Title
	g.pdf.SetFont("Arial", "B", 16)
	g.pdf.CellFormat(g.pageWidth-2*g.margin, 10, title, "", 1, "C", false, 0, "")

	// Subtitle
	g.pdf.SetFont("Arial", "", 12)
	g.pdf.CellFormat(g.pageWidth-2*g.margin, 8, subtitle, "", 1, "C", false, 0, "")

	g.pdf.Ln(5)
}

func (g *PDFGenerator) addTournamentInfo(tournament *TournamentExport) {
	g.pdf.SetFont("Arial", "", 9)

	// First row: Date and Site
	infoWidth := (g.pageWidth - 2*g.margin) / 2

	dateStr := tournament.StartDate.Format("02.01.2006")
	if tournament.EndDate != nil {
		dateStr += " - " + tournament.EndDate.Format("02.01.2006")
	}

	g.pdf.CellFormat(infoWidth, 5, fmt.Sprintf("Date: %s", dateStr), "", 0, "L", false, 0, "")
	g.pdf.CellFormat(infoWidth, 5, fmt.Sprintf("Site: %s", tournament.Site), "", 1, "R", false, 0, "")

	// Second row: Rounds and Type
	g.pdf.CellFormat(infoWidth, 5, fmt.Sprintf("Rounds: %d", tournament.Rounds), "", 0, "L", false, 0, "")
	g.pdf.CellFormat(infoWidth, 5, fmt.Sprintf("Type: %s", tournament.Type), "", 1, "R", false, 0, "")

	// Third row: Time Control and Pairing System
	g.pdf.CellFormat(infoWidth, 5, fmt.Sprintf("Time Control: %s", tournament.TimeControl), "", 0, "L", false, 0, "")
	g.pdf.CellFormat(infoWidth, 5, fmt.Sprintf("Pairing: %s", tournament.PairingSystem), "", 1, "R", false, 0, "")

	// Fourth row: Arbiter info
	if tournament.Chief != "" {
		g.pdf.CellFormat(infoWidth, 5, fmt.Sprintf("Chief Arbiter: %s", tournament.Chief), "", 0, "L", false, 0, "")
		if tournament.Deputy != "" {
			g.pdf.CellFormat(infoWidth, 5, fmt.Sprintf("Deputy: %s", tournament.Deputy), "", 1, "R", false, 0, "")
		} else {
			g.pdf.Ln(5)
		}
	}

	g.pdf.Ln(8)
}

func (g *PDFGenerator) addStandingsTable(tournament *TournamentExport) {
	// Calculate column widths based on available space
	availWidth := g.pageWidth - 2*g.margin

	// Column definitions
	colRank := 10.0
	colName := 50.0
	colRating := 15.0
	colFideID := 22.0
	colPoints := 15.0
	colWins := 10.0
	colDraws := 10.0
	colLosses := 10.0
	colBuchholz := 18.0
	colSB := 18.0

	// Adjust name column to fit
	usedWidth := colRank + colRating + colFideID + colPoints + colWins + colDraws + colLosses + colBuchholz + colSB
	colName = availWidth - usedWidth
	if colName < 30 {
		colName = 30
	}

	// Table header
	g.pdf.SetFont("Arial", "B", 8)
	g.pdf.SetFillColor(220, 220, 220)

	g.pdf.CellFormat(colRank, 7, "Rk", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colName, 7, "Name", "1", 0, "L", true, 0, "")
	g.pdf.CellFormat(colRating, 7, "Rtg", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colFideID, 7, "FIDE ID", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colPoints, 7, "Pts", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colWins, 7, "W", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colDraws, 7, "D", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colLosses, 7, "L", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colBuchholz, 7, "BH", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colSB, 7, "SB", "1", 1, "C", true, 0, "")

	// Table rows
	g.pdf.SetFont("Arial", "", 8)

	for i, p := range tournament.Participants {
		// Alternate row colors
		if i%2 == 0 {
			g.pdf.SetFillColor(245, 245, 245)
		} else {
			g.pdf.SetFillColor(255, 255, 255)
		}

		fideID := ""
		if p.FideID != nil {
			fideID = *p.FideID
		}

		// Truncate name if too long
		name := p.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		g.pdf.CellFormat(colRank, 6, fmt.Sprintf("%d", p.Rank), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colName, 6, name, "1", 0, "L", true, 0, "")
		g.pdf.CellFormat(colRating, 6, fmt.Sprintf("%d", p.Rating), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colFideID, 6, fideID, "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colPoints, 6, formatPoints(p.Points), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colWins, 6, fmt.Sprintf("%d", p.Wins), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colDraws, 6, fmt.Sprintf("%d", p.Draws), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colLosses, 6, fmt.Sprintf("%d", p.Losses), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colBuchholz, 6, fmt.Sprintf("%.1f", p.Buchholz), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colSB, 6, fmt.Sprintf("%.1f", p.SonnebornBerger), "1", 1, "C", true, 0, "")
	}

	g.pdf.Ln(5)

	// Summary
	g.pdf.SetFont("Arial", "I", 8)
	g.pdf.CellFormat(0, 5, fmt.Sprintf("Total Participants: %d", len(tournament.Participants)), "", 1, "L", false, 0, "")
}

func (g *PDFGenerator) addCrosstable(tournament *TournamentExport) {
	numParticipants := len(tournament.Participants)
	if numParticipants == 0 {
		return
	}

	// Calculate column widths
	availWidth := g.pageWidth - 2*g.margin

	// Fixed columns
	colRank := 8.0
	colName := 35.0
	colRating := 12.0
	colPoints := 12.0

	// Calculate round column width
	fixedWidth := colRank + colName + colRating + colPoints
	remainingWidth := availWidth - fixedWidth
	colRound := remainingWidth / float64(tournament.Rounds)
	if colRound < 8 {
		colRound = 8
	}
	if colRound > 15 {
		colRound = 15
	}

	// Table header
	g.pdf.SetFont("Arial", "B", 7)
	g.pdf.SetFillColor(220, 220, 220)

	g.pdf.CellFormat(colRank, 6, "Rk", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colName, 6, "Name", "1", 0, "L", true, 0, "")
	g.pdf.CellFormat(colRating, 6, "Rtg", "1", 0, "C", true, 0, "")

	// Round headers
	for i := 1; i <= tournament.Rounds; i++ {
		g.pdf.CellFormat(colRound, 6, fmt.Sprintf("R%d", i), "1", 0, "C", true, 0, "")
	}

	g.pdf.CellFormat(colPoints, 6, "Pts", "1", 1, "C", true, 0, "")

	// Table rows
	g.pdf.SetFont("Arial", "", 7)

	for i, p := range tournament.Participants {
		// Alternate row colors
		if i%2 == 0 {
			g.pdf.SetFillColor(245, 245, 245)
		} else {
			g.pdf.SetFillColor(255, 255, 255)
		}

		// Truncate name if too long
		name := p.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}

		g.pdf.CellFormat(colRank, 5, fmt.Sprintf("%d", p.Rank), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colName, 5, name, "1", 0, "L", true, 0, "")
		g.pdf.CellFormat(colRating, 5, fmt.Sprintf("%d", p.Rating), "1", 0, "C", true, 0, "")

		// Round results
		for round := 1; round <= tournament.Rounds; round++ {
			result := "-"
			for _, r := range p.Results {
				if r.RoundNumber == round {
					if r.OpponentRank != nil {
						result = fmt.Sprintf("%s%d", colorCode(r.Color), *r.OpponentRank)
						if r.Score == "1" {
							result += "+"
						} else if r.Score == "0" {
							result += "-"
						} else if r.Score == "=" {
							result += "="
						}
					} else {
						// Bye or forfeit
						result = r.Score
						if r.Color == "-" {
							result = "bye"
						}
					}
					break
				}
			}
			g.pdf.CellFormat(colRound, 5, result, "1", 0, "C", true, 0, "")
		}

		g.pdf.CellFormat(colPoints, 5, formatPoints(p.Points), "1", 1, "C", true, 0, "")
	}
}

func (g *PDFGenerator) addFooter() {
	// Move to bottom of page
	g.pdf.SetY(-20)
	g.pdf.SetFont("Arial", "I", 8)
	g.pdf.SetTextColor(128, 128, 128)

	// Generated timestamp
	timestamp := time.Now().Format("02.01.2006 15:04")
	g.pdf.CellFormat(0, 5, fmt.Sprintf("Generated: %s | OpenPairing", timestamp), "", 1, "C", false, 0, "")

	// Reset text color
	g.pdf.SetTextColor(0, 0, 0)
}

// Helper functions

func formatPoints(points float64) string {
	if points == float64(int(points)) {
		return fmt.Sprintf("%.0f", points)
	}
	return fmt.Sprintf("%.1f", points)
}

func colorCode(color string) string {
	switch color {
	case "w":
		return "w"
	case "b":
		return "s" // "s" for "schwarz" (German for black)
	default:
		return ""
	}
}
