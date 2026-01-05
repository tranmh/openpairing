package export

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Service provides export functionality
type Service struct {
	repo         *Repository
	pgnGenerator *PGNGenerator
	pdfGenerator *PDFGenerator
}

// NewService creates a new export service
func NewService(repo *Repository) *Service {
	return &Service{
		repo:         repo,
		pgnGenerator: NewPGNGenerator(),
		pdfGenerator: NewPDFGenerator(),
	}
}

// ExportTournamentPGN exports all games from a tournament in PGN format
func (s *Service) ExportTournamentPGN(ctx context.Context, tournamentID uuid.UUID) (*ExportResponse, error) {
	tournament, err := s.repo.GetTournamentExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament data: %w", err)
	}

	pgn := s.pgnGenerator.GenerateTournament(tournament)

	// Create safe filename
	safeName := sanitizeFilename(tournament.Name)
	filename := fmt.Sprintf("%s.pgn", safeName)

	return &ExportResponse{
		Format:   FormatPGN,
		Filename: filename,
		Content:  pgn,
		MimeType: FormatPGN.GetMimeType(),
	}, nil
}

// ExportRoundPGN exports all games from a specific round in PGN format
func (s *Service) ExportRoundPGN(ctx context.Context, tournamentID uuid.UUID, roundNumber int) (*ExportResponse, error) {
	games, err := s.repo.GetGamesExportData(ctx, tournamentID, &roundNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}

	if len(games) == 0 {
		return nil, fmt.Errorf("no games found for round %d", roundNumber)
	}

	pgn := s.pgnGenerator.GenerateGames(games)

	filename := fmt.Sprintf("round_%d.pgn", roundNumber)

	return &ExportResponse{
		Format:   FormatPGN,
		Filename: filename,
		Content:  pgn,
		MimeType: FormatPGN.GetMimeType(),
	}, nil
}

// ExportGamePGN exports a single game in PGN format
func (s *Service) ExportGamePGN(ctx context.Context, gameID uuid.UUID) (*ExportResponse, error) {
	game, err := s.repo.GetSingleGameExportData(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	pgn := s.pgnGenerator.GenerateGame(game)

	// Create filename from player names and round
	whiteName := sanitizeFilename(game.Header.White)
	blackName := sanitizeFilename(game.Header.Black)
	filename := fmt.Sprintf("%s_vs_%s.pgn", whiteName, blackName)

	return &ExportResponse{
		Format:   FormatPGN,
		Filename: filename,
		Content:  pgn,
		MimeType: FormatPGN.GetMimeType(),
	}, nil
}

// ExportTournamentJSON exports tournament data in JSON format
func (s *Service) ExportTournamentJSON(ctx context.Context, tournamentID uuid.UUID) (*ExportResponse, error) {
	tournament, err := s.repo.GetTournamentExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament data: %w", err)
	}

	data, err := json.MarshalIndent(tournament, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	safeName := sanitizeFilename(tournament.Name)
	filename := fmt.Sprintf("%s.json", safeName)

	return &ExportResponse{
		Format:   FormatJSON,
		Filename: filename,
		Content:  string(data),
		MimeType: FormatJSON.GetMimeType(),
	}, nil
}

// ExportStandingsCSV exports tournament standings in CSV format
func (s *Service) ExportStandingsCSV(ctx context.Context, tournamentID uuid.UUID) (*ExportResponse, error) {
	tournament, err := s.repo.GetTournamentExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament data: %w", err)
	}

	var sb strings.Builder

	// Header
	sb.WriteString("Rank,Name,Rating,FIDE ID,Points,Wins,Draws,Losses,Buchholz,Buchholz Cut 1,Sonneborn-Berger\n")

	// Data rows
	for _, p := range tournament.Participants {
		fideID := ""
		if p.FideID != nil {
			fideID = *p.FideID
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%d,%s,%.1f,%d,%d,%d,%.2f,%.2f,%.2f\n",
			p.Rank,
			escapeCSV(p.Name),
			p.Rating,
			fideID,
			p.Points,
			p.Wins,
			p.Draws,
			p.Losses,
			p.Buchholz,
			p.BuchholzCut1,
			p.SonnebornBerger,
		))
	}

	safeName := sanitizeFilename(tournament.Name)
	filename := fmt.Sprintf("%s_standings.csv", safeName)

	return &ExportResponse{
		Format:   FormatCSV,
		Filename: filename,
		Content:  sb.String(),
		MimeType: FormatCSV.GetMimeType(),
	}, nil
}

// ExportCrosstableCSV exports tournament crosstable in CSV format
func (s *Service) ExportCrosstableCSV(ctx context.Context, tournamentID uuid.UUID) (*ExportResponse, error) {
	tournament, err := s.repo.GetTournamentExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament data: %w", err)
	}

	var sb strings.Builder

	// Header row
	sb.WriteString("Rank,Name,Rating")
	for i := 1; i <= tournament.Rounds; i++ {
		sb.WriteString(fmt.Sprintf(",Round %d", i))
	}
	sb.WriteString(",Points\n")

	// Data rows
	for _, p := range tournament.Participants {
		sb.WriteString(fmt.Sprintf("%d,%s,%d", p.Rank, escapeCSV(p.Name), p.Rating))

		// Round results
		for i := 1; i <= tournament.Rounds; i++ {
			result := "-"
			for _, r := range p.Results {
				if r.RoundNumber == i {
					if r.OpponentRank != nil {
						result = fmt.Sprintf("%s%d %s", r.Color, *r.OpponentRank, r.Score)
					} else {
						result = fmt.Sprintf("%s %s", r.Color, r.Score)
					}
					break
				}
			}
			sb.WriteString("," + result)
		}

		sb.WriteString(fmt.Sprintf(",%.1f\n", p.Points))
	}

	safeName := sanitizeFilename(tournament.Name)
	filename := fmt.Sprintf("%s_crosstable.csv", safeName)

	return &ExportResponse{
		Format:   FormatCSV,
		Filename: filename,
		Content:  sb.String(),
		MimeType: FormatCSV.GetMimeType(),
	}, nil
}

// Helper functions

func sanitizeFilename(name string) string {
	// Replace spaces with underscores
	result := strings.ReplaceAll(name, " ", "_")

	// Remove or replace invalid characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", ","}
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "")
	}

	// Truncate if too long
	if len(result) > 100 {
		result = result[:100]
	}

	return result
}

func escapeCSV(s string) string {
	// If string contains comma, quote, or newline, wrap in quotes and escape quotes
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// ExportStandingsPDF exports tournament standings in PDF format
func (s *Service) ExportStandingsPDF(ctx context.Context, tournamentID uuid.UUID) (*PDFExportResponse, error) {
	tournament, err := s.repo.GetTournamentExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament data: %w", err)
	}

	pdfBytes, err := s.pdfGenerator.GenerateStandingsPDF(tournament)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	safeName := sanitizeFilename(tournament.Name)
	filename := fmt.Sprintf("%s_standings.pdf", safeName)

	return &PDFExportResponse{
		Format:   FormatPDF,
		Filename: filename,
		Content:  pdfBytes,
		MimeType: FormatPDF.GetMimeType(),
	}, nil
}

// ExportCrosstablePDF exports tournament crosstable in PDF format
func (s *Service) ExportCrosstablePDF(ctx context.Context, tournamentID uuid.UUID) (*PDFExportResponse, error) {
	tournament, err := s.repo.GetTournamentExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament data: %w", err)
	}

	pdfBytes, err := s.pdfGenerator.GenerateCrosstablePDF(tournament)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	safeName := sanitizeFilename(tournament.Name)
	filename := fmt.Sprintf("%s_crosstable.pdf", safeName)

	return &PDFExportResponse{
		Format:   FormatPDF,
		Filename: filename,
		Content:  pdfBytes,
		MimeType: FormatPDF.GetMimeType(),
	}, nil
}

// PDFExportResponse contains the exported PDF data (binary)
type PDFExportResponse struct {
	Format   ExportFormat `json:"format"`
	Filename string       `json:"filename"`
	Content  []byte       `json:"-"` // Binary content, not included in JSON
	MimeType string       `json:"mimeType"`
}

// GenerateCertificate generates a certificate PDF for a participant
func (s *Service) GenerateCertificate(ctx context.Context, tournamentID, participantID uuid.UUID) (*PDFExportResponse, error) {
	// Get tournament and participant data
	tournament, err := s.repo.GetTournamentExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament data: %w", err)
	}

	// Find participant
	var participant *ParticipantExport
	for _, p := range tournament.Participants {
		if p.ID == participantID {
			participant = &p
			break
		}
	}

	if participant == nil {
		return nil, fmt.Errorf("participant not found")
	}

	// Format date
	dateStr := tournament.StartDate.Format("02 January 2006")
	if tournament.EndDate != nil {
		dateStr += " - " + tournament.EndDate.Format("02 January 2006")
	}

	certData := &CertificateData{
		TournamentName:    tournament.Name,
		TournamentDate:    dateStr,
		PlayerName:        participant.Name,
		PlayerFideID:      "",
		FinalRank:         participant.Rank,
		TotalParticipants: len(tournament.Participants),
		Points:            participant.Points,
		TotalRounds:       tournament.Rounds,
		ArbiterName:       tournament.Chief,
		OrganizerName:     "",
	}

	if participant.FideID != nil {
		certData.PlayerFideID = *participant.FideID
	}

	pdfBytes, err := s.pdfGenerator.GenerateCertificatePDF(certData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w", err)
	}

	safeName := sanitizeFilename(participant.Name)
	safeTournament := sanitizeFilename(tournament.Name)
	filename := fmt.Sprintf("%s_%s_certificate.pdf", safeTournament, safeName)

	return &PDFExportResponse{
		Format:   FormatPDF,
		Filename: filename,
		Content:  pdfBytes,
		MimeType: FormatPDF.GetMimeType(),
	}, nil
}

// GenerateAllCertificates generates certificates for all participants or top N
func (s *Service) GenerateAllCertificates(ctx context.Context, tournamentID uuid.UUID, topN int) ([]PDFExportResponse, error) {
	tournament, err := s.repo.GetTournamentExportData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament data: %w", err)
	}

	// Format date
	dateStr := tournament.StartDate.Format("02 January 2006")
	if tournament.EndDate != nil {
		dateStr += " - " + tournament.EndDate.Format("02 January 2006")
	}

	var certificates []PDFExportResponse
	count := len(tournament.Participants)
	if topN > 0 && topN < count {
		count = topN
	}

	for i := 0; i < count; i++ {
		p := tournament.Participants[i]

		certData := &CertificateData{
			TournamentName:    tournament.Name,
			TournamentDate:    dateStr,
			PlayerName:        p.Name,
			FinalRank:         p.Rank,
			TotalParticipants: len(tournament.Participants),
			Points:            p.Points,
			TotalRounds:       tournament.Rounds,
			ArbiterName:       tournament.Chief,
		}

		if p.FideID != nil {
			certData.PlayerFideID = *p.FideID
		}

		pdfBytes, err := s.pdfGenerator.GenerateCertificatePDF(certData)
		if err != nil {
			return nil, fmt.Errorf("failed to generate certificate for %s: %w", p.Name, err)
		}

		safeName := sanitizeFilename(p.Name)
		safeTournament := sanitizeFilename(tournament.Name)
		filename := fmt.Sprintf("%s_%s_certificate.pdf", safeTournament, safeName)

		certificates = append(certificates, PDFExportResponse{
			Format:   FormatPDF,
			Filename: filename,
			Content:  pdfBytes,
			MimeType: FormatPDF.GetMimeType(),
		})
	}

	return certificates, nil
}
