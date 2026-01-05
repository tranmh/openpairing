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

// CertificateData contains data for generating a certificate
type CertificateData struct {
	TournamentName   string
	TournamentDate   string
	PlayerName       string
	PlayerFideID     string
	FinalRank        int
	TotalParticipants int
	Points           float64
	TotalRounds      int
	ArbiterName      string
	OrganizerName    string
}

// GenerateCertificatePDF generates a participation/achievement certificate
func (g *PDFGenerator) GenerateCertificatePDF(data *CertificateData) ([]byte, error) {
	g.pdf = fpdf.New("L", "mm", "A4", "") // Landscape for certificate
	g.pageWidth, g.pageHeight = g.pdf.GetPageSize()

	g.pdf.SetMargins(20, 20, 20)
	g.pdf.AddPage()

	// Decorative border
	g.addCertificateBorder()

	// Certificate title
	g.pdf.SetY(35)
	g.pdf.SetFont("Arial", "B", 28)
	g.pdf.SetTextColor(30, 58, 138) // Dark blue
	g.pdf.CellFormat(g.pageWidth-40, 15, "CERTIFICATE", "", 1, "C", false, 0, "")

	// Subtitle
	g.pdf.SetFont("Arial", "", 14)
	g.pdf.SetTextColor(75, 85, 99) // Gray
	certificateType := "of Participation"
	if data.FinalRank <= 3 {
		certificateType = "of Achievement"
	}
	g.pdf.CellFormat(g.pageWidth-40, 10, certificateType, "", 1, "C", false, 0, "")

	// Player name
	g.pdf.Ln(15)
	g.pdf.SetFont("Arial", "B", 24)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(g.pageWidth-40, 12, data.PlayerName, "", 1, "C", false, 0, "")

	// Achievement description
	g.pdf.Ln(8)
	g.pdf.SetFont("Arial", "", 12)
	g.pdf.SetTextColor(55, 65, 81)

	var achievementText string
	if data.FinalRank == 1 {
		achievementText = "achieved FIRST PLACE"
	} else if data.FinalRank == 2 {
		achievementText = "achieved SECOND PLACE"
	} else if data.FinalRank == 3 {
		achievementText = "achieved THIRD PLACE"
	} else {
		achievementText = fmt.Sprintf("achieved rank %d of %d participants", data.FinalRank, data.TotalParticipants)
	}

	g.pdf.CellFormat(g.pageWidth-40, 8, achievementText, "", 1, "C", false, 0, "")

	// Points information
	g.pdf.SetFont("Arial", "", 11)
	pointsText := fmt.Sprintf("with %.1f points out of %d rounds", data.Points, data.TotalRounds)
	g.pdf.CellFormat(g.pageWidth-40, 7, pointsText, "", 1, "C", false, 0, "")

	// Tournament info
	g.pdf.Ln(5)
	g.pdf.SetFont("Arial", "", 11)
	g.pdf.CellFormat(g.pageWidth-40, 7, "in the", "", 1, "C", false, 0, "")

	g.pdf.SetFont("Arial", "B", 14)
	g.pdf.CellFormat(g.pageWidth-40, 8, data.TournamentName, "", 1, "C", false, 0, "")

	g.pdf.SetFont("Arial", "", 11)
	g.pdf.CellFormat(g.pageWidth-40, 7, data.TournamentDate, "", 1, "C", false, 0, "")

	// Medal icon for top 3
	if data.FinalRank <= 3 {
		g.addMedalIcon(data.FinalRank)
	}

	// Signature lines
	g.addSignatureLines(data)

	// Footer
	g.pdf.SetY(g.pageHeight - 25)
	g.pdf.SetFont("Arial", "I", 8)
	g.pdf.SetTextColor(128, 128, 128)
	g.pdf.CellFormat(g.pageWidth-40, 5, "Generated by OpenPairing - Chess Tournament Management", "", 1, "C", false, 0, "")

	// Output to bytes
	var buf bytes.Buffer
	err := g.pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *PDFGenerator) addCertificateBorder() {
	// Outer border
	g.pdf.SetDrawColor(30, 58, 138) // Dark blue
	g.pdf.SetLineWidth(1)
	g.pdf.Rect(10, 10, g.pageWidth-20, g.pageHeight-20, "D")

	// Inner border
	g.pdf.SetLineWidth(0.5)
	g.pdf.Rect(15, 15, g.pageWidth-30, g.pageHeight-30, "D")

	// Decorative corners
	g.pdf.SetLineWidth(0.3)
	cornerSize := 15.0

	// Top-left
	g.pdf.Line(15, 15+cornerSize, 15+cornerSize, 15)
	// Top-right
	g.pdf.Line(g.pageWidth-15-cornerSize, 15, g.pageWidth-15, 15+cornerSize)
	// Bottom-left
	g.pdf.Line(15, g.pageHeight-15-cornerSize, 15+cornerSize, g.pageHeight-15)
	// Bottom-right
	g.pdf.Line(g.pageWidth-15-cornerSize, g.pageHeight-15, g.pageWidth-15, g.pageHeight-15-cornerSize)
}

func (g *PDFGenerator) addMedalIcon(rank int) {
	// Position medal icon
	medalX := g.pageWidth/2 - 10
	medalY := float64(125)

	// Medal colors based on rank
	var r, gColor, b int
	switch rank {
	case 1: // Gold
		r, gColor, b = 255, 215, 0
	case 2: // Silver
		r, gColor, b = 192, 192, 192
	case 3: // Bronze
		r, gColor, b = 205, 127, 50
	}

	// Draw medal circle
	g.pdf.SetFillColor(r, gColor, b)
	g.pdf.Circle(medalX+10, medalY+10, 10, "F")

	// Draw medal outline
	g.pdf.SetDrawColor(100, 100, 100)
	g.pdf.SetLineWidth(0.5)
	g.pdf.Circle(medalX+10, medalY+10, 10, "D")

	// Medal number
	g.pdf.SetFont("Arial", "B", 14)
	g.pdf.SetTextColor(50, 50, 50)
	g.pdf.SetXY(medalX+6, medalY+5)
	g.pdf.CellFormat(8, 10, fmt.Sprintf("%d", rank), "", 0, "C", false, 0, "")
}

func (g *PDFGenerator) addSignatureLines(data *CertificateData) {
	lineY := g.pageHeight - 50
	lineWidth := 60.0
	spacing := 40.0

	g.pdf.SetDrawColor(100, 100, 100)
	g.pdf.SetLineWidth(0.3)

	// Left signature (Arbiter)
	leftX := (g.pageWidth - 2*lineWidth - spacing) / 2
	g.pdf.Line(leftX, lineY, leftX+lineWidth, lineY)

	g.pdf.SetFont("Arial", "", 9)
	g.pdf.SetTextColor(100, 100, 100)
	g.pdf.SetXY(leftX, lineY+2)
	label := "Chief Arbiter"
	if data.ArbiterName != "" {
		label = data.ArbiterName
	}
	g.pdf.CellFormat(lineWidth, 5, label, "", 0, "C", false, 0, "")

	// Right signature (Organizer)
	rightX := leftX + lineWidth + spacing
	g.pdf.Line(rightX, lineY, rightX+lineWidth, lineY)

	g.pdf.SetXY(rightX, lineY+2)
	orgLabel := "Tournament Organizer"
	if data.OrganizerName != "" {
		orgLabel = data.OrganizerName
	}
	g.pdf.CellFormat(lineWidth, 5, orgLabel, "", 0, "C", false, 0, "")
}

// GeneratePairingsPDF generates a PDF of round pairings
func (g *PDFGenerator) GeneratePairingsPDF(data *PairingsExport) ([]byte, error) {
	g.pdf = fpdf.New("P", "mm", "A4", "")
	g.pageWidth, g.pageHeight = g.pdf.GetPageSize()

	g.pdf.SetMargins(g.margin, g.margin, g.margin)
	g.pdf.SetAutoPageBreak(true, g.margin)

	g.pdf.AddPage()

	// Header
	g.addHeader(data.TournamentName, fmt.Sprintf("Paarungen - Runde %d", data.RoundNumber))

	// Round info
	g.pdf.SetFont("Arial", "", 9)
	g.pdf.CellFormat(0, 5, fmt.Sprintf("Datum: %s", data.Date.Format("02.01.2006")), "", 1, "L", false, 0, "")
	if data.Site != "" {
		g.pdf.CellFormat(0, 5, fmt.Sprintf("Ort: %s", data.Site), "", 1, "L", false, 0, "")
	}
	g.pdf.Ln(5)

	// Pairings table
	g.addPairingsTable(data.Games)

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

func (g *PDFGenerator) addPairingsTable(games []PairingGame) {
	if len(games) == 0 {
		g.pdf.SetFont("Arial", "I", 10)
		g.pdf.CellFormat(0, 10, "Keine Paarungen vorhanden", "", 1, "C", false, 0, "")
		return
	}

	availWidth := g.pageWidth - 2*g.margin

	// Column definitions
	colBoard := 12.0
	colWhiteRating := 15.0
	colResult := 18.0
	colBlackRating := 15.0

	nameWidth := (availWidth - colBoard - colWhiteRating - colResult - colBlackRating) / 2
	colWhiteName := nameWidth
	colBlackName := nameWidth

	// Table header
	g.pdf.SetFont("Arial", "B", 8)
	g.pdf.SetFillColor(220, 220, 220)

	g.pdf.CellFormat(colBoard, 7, "Brett", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colWhiteName, 7, "Weiß", "1", 0, "L", true, 0, "")
	g.pdf.CellFormat(colWhiteRating, 7, "Elo", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colResult, 7, "Ergebnis", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colBlackRating, 7, "Elo", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colBlackName, 7, "Schwarz", "1", 1, "R", true, 0, "")

	// Table rows
	g.pdf.SetFont("Arial", "", 8)

	for i, game := range games {
		if i%2 == 0 {
			g.pdf.SetFillColor(245, 245, 245)
		} else {
			g.pdf.SetFillColor(255, 255, 255)
		}

		whiteName := game.WhiteName
		if len(whiteName) > 25 {
			whiteName = whiteName[:22] + "..."
		}
		blackName := game.BlackName
		if blackName == "" {
			blackName = "spielfrei"
		} else if len(blackName) > 25 {
			blackName = blackName[:22] + "..."
		}

		result := game.Result
		if result == "" {
			result = "-"
		}

		whiteRating := fmt.Sprintf("%d", game.WhiteRating)
		if game.WhiteRating == 0 {
			whiteRating = "-"
		}
		blackRating := fmt.Sprintf("%d", game.BlackRating)
		if game.BlackRating == 0 {
			blackRating = "-"
		}

		g.pdf.CellFormat(colBoard, 6, fmt.Sprintf("%d", game.BoardNumber), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colWhiteName, 6, whiteName, "1", 0, "L", true, 0, "")
		g.pdf.CellFormat(colWhiteRating, 6, whiteRating, "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colResult, 6, result, "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colBlackRating, 6, blackRating, "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colBlackName, 6, blackName, "1", 1, "R", true, 0, "")
	}

	g.pdf.Ln(5)
	g.pdf.SetFont("Arial", "I", 8)
	g.pdf.CellFormat(0, 5, fmt.Sprintf("Gesamt: %d Partien", len(games)), "", 1, "L", false, 0, "")
}

// GenerateParticipantListPDF generates a PDF of tournament participant list
func (g *PDFGenerator) GenerateParticipantListPDF(tournament *TournamentExport) ([]byte, error) {
	g.pdf = fpdf.New("P", "mm", "A4", "")
	g.pageWidth, g.pageHeight = g.pdf.GetPageSize()

	g.pdf.SetMargins(g.margin, g.margin, g.margin)
	g.pdf.SetAutoPageBreak(true, g.margin)

	g.pdf.AddPage()

	// Header
	g.addHeader(tournament.Name, "Teilnehmerliste")

	// Tournament info
	g.addTournamentInfo(tournament)

	// Participant list table
	g.addParticipantListTable(tournament)

	// Footer
	g.addFooter()

	var buf bytes.Buffer
	err := g.pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *PDFGenerator) addParticipantListTable(tournament *TournamentExport) {
	if len(tournament.Participants) == 0 {
		g.pdf.SetFont("Arial", "I", 10)
		g.pdf.CellFormat(0, 10, "Keine Teilnehmer vorhanden", "", 1, "C", false, 0, "")
		return
	}

	availWidth := g.pageWidth - 2*g.margin

	// Column definitions
	colRank := 10.0
	colRating := 15.0
	colFideID := 22.0
	colTitle := 12.0
	colFederation := 15.0
	colClub := 35.0

	// Calculate name column width
	usedWidth := colRank + colRating + colFideID + colTitle + colFederation + colClub
	colName := availWidth - usedWidth
	if colName < 40 {
		colName = 40
		colClub = 0 // Hide club if not enough space
	}

	// Table header
	g.pdf.SetFont("Arial", "B", 8)
	g.pdf.SetFillColor(220, 220, 220)

	g.pdf.CellFormat(colRank, 7, "Nr.", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colName, 7, "Name", "1", 0, "L", true, 0, "")
	g.pdf.CellFormat(colRating, 7, "Elo", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colFideID, 7, "FIDE-ID", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colTitle, 7, "Titel", "1", 0, "C", true, 0, "")
	g.pdf.CellFormat(colFederation, 7, "Fed", "1", 0, "C", true, 0, "")
	if colClub > 0 {
		g.pdf.CellFormat(colClub, 7, "Verein", "1", 0, "L", true, 0, "")
	}
	g.pdf.Ln(-1)

	// Table rows - sorted by start rank
	g.pdf.SetFont("Arial", "", 8)

	for i, p := range tournament.Participants {
		if i%2 == 0 {
			g.pdf.SetFillColor(245, 245, 245)
		} else {
			g.pdf.SetFillColor(255, 255, 255)
		}

		fideID := ""
		if p.FideID != nil {
			fideID = *p.FideID
		}
		title := ""
		if p.Title != nil {
			title = *p.Title
		}
		federation := ""
		if p.Federation != nil {
			federation = *p.Federation
		}
		club := ""
		if p.Club != nil {
			club = *p.Club
		}

		name := p.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		g.pdf.CellFormat(colRank, 6, fmt.Sprintf("%d", p.StartRank), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colName, 6, name, "1", 0, "L", true, 0, "")
		g.pdf.CellFormat(colRating, 6, fmt.Sprintf("%d", p.Rating), "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colFideID, 6, fideID, "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colTitle, 6, title, "1", 0, "C", true, 0, "")
		g.pdf.CellFormat(colFederation, 6, federation, "1", 0, "C", true, 0, "")
		if colClub > 0 {
			if len(club) > 20 {
				club = club[:17] + "..."
			}
			g.pdf.CellFormat(colClub, 6, club, "1", 0, "L", true, 0, "")
		}
		g.pdf.Ln(-1)
	}

	g.pdf.Ln(5)
	g.pdf.SetFont("Arial", "I", 8)
	g.pdf.CellFormat(0, 5, fmt.Sprintf("Gesamt: %d Teilnehmer", len(tournament.Participants)), "", 1, "L", false, 0, "")
}
