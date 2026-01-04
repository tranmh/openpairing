package reports

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DWZReport generates reports in DWZ format for German Chess Federation submission
type DWZReport struct {
	Tournament     DWZTournamentInfo
	Players        []DWZPlayerRecord
	Games          []DWZGameRecord
	GeneratedAt    time.Time
}

// DWZTournamentInfo contains tournament metadata for DWZ report
type DWZTournamentInfo struct {
	TurnierCode     string    // Tournament code assigned by federation
	Name            string
	StartDate       time.Time
	EndDate         time.Time
	City            string
	Organizer       string
	ChiefArbiter    string
	TimeControl     string
	Rounds          int
	PairingSystem   string // Swiss, Round Robin, etc.
	RatingType      string // Standard, Rapid, Blitz
}

// DWZPlayerRecord contains player data for DWZ report
type DWZPlayerRecord struct {
	StartNumber   int
	PKZ           string // Personenkennziffer (DSB ID)
	Name          string
	FirstName     string
	LastName      string
	ClubID        string // ZPS number
	ClubName      string
	DWZ           int
	DWZIndex      int
	FideID        string
	FideRating    int
	BirthYear     int
	Gender        string
	Score         float64
	Rank          int
	Rounds        []DWZRoundResult
}

// DWZRoundResult contains a single round result
type DWZRoundResult struct {
	Round        int
	OpponentNum  int    // 0 for bye/forfeit
	Color        string // w, b, or empty
	Result       string // 1, 0, =, +, -, etc.
}

// DWZGameRecord represents a single game for DWZ submission
type DWZGameRecord struct {
	Round       int
	WhitePKZ    string
	WhiteName   string
	WhiteDWZ    int
	BlackPKZ    string
	BlackName   string
	BlackDWZ    int
	Result      string // 1-0, 0-1, 1/2-1/2, +/-, -/+, etc.
}

// DWZGenerator generates DWZ format reports
type DWZGenerator struct {
	report *DWZReport
}

// NewDWZGenerator creates a new DWZ report generator
func NewDWZGenerator() *DWZGenerator {
	return &DWZGenerator{
		report: &DWZReport{
			GeneratedAt: time.Now(),
		},
	}
}

// SetTournamentInfo sets the tournament metadata
func (g *DWZGenerator) SetTournamentInfo(info DWZTournamentInfo) {
	g.report.Tournament = info
}

// AddPlayer adds a player record to the report
func (g *DWZGenerator) AddPlayer(player DWZPlayerRecord) {
	g.report.Players = append(g.report.Players, player)
}

// AddGame adds a game record to the report
func (g *DWZGenerator) AddGame(game DWZGameRecord) {
	g.report.Games = append(g.report.Games, game)
}

// GenerateCSV generates the DWZ report in CSV format
func (g *DWZGenerator) GenerateCSV() ([]byte, error) {
	var buf bytes.Buffer

	// Header
	buf.WriteString("# DWZ-Auswertungsdatei\n")
	buf.WriteString(fmt.Sprintf("# Generiert: %s\n", g.report.GeneratedAt.Format("2006-01-02 15:04:05")))
	buf.WriteString("#\n")

	// Tournament info section
	buf.WriteString("[Turnier]\n")
	buf.WriteString(fmt.Sprintf("Name;%s\n", g.report.Tournament.Name))
	buf.WriteString(fmt.Sprintf("Ort;%s\n", g.report.Tournament.City))
	buf.WriteString(fmt.Sprintf("Beginn;%s\n", g.report.Tournament.StartDate.Format("02.01.2006")))
	buf.WriteString(fmt.Sprintf("Ende;%s\n", g.report.Tournament.EndDate.Format("02.01.2006")))
	buf.WriteString(fmt.Sprintf("Runden;%d\n", g.report.Tournament.Rounds))
	buf.WriteString(fmt.Sprintf("System;%s\n", g.report.Tournament.PairingSystem))
	buf.WriteString(fmt.Sprintf("Hauptschiedsrichter;%s\n", g.report.Tournament.ChiefArbiter))
	buf.WriteString(fmt.Sprintf("Bedenkzeit;%s\n", g.report.Tournament.TimeControl))
	if g.report.Tournament.TurnierCode != "" {
		buf.WriteString(fmt.Sprintf("Turniercode;%s\n", g.report.Tournament.TurnierCode))
	}
	buf.WriteString("\n")

	// Players section
	buf.WriteString("[Teilnehmer]\n")
	buf.WriteString("Nr;PKZ;Name;Vorname;Verein-ZPS;DWZ;DWZ-Index;FIDE-ID;ELO;Geburtsjahr;Geschlecht;Punkte;Rang\n")
	for _, p := range g.report.Players {
		buf.WriteString(fmt.Sprintf("%d;%s;%s;%s;%s;%d;%d;%s;%d;%d;%s;%.1f;%d\n",
			p.StartNumber, p.PKZ, p.LastName, p.FirstName, p.ClubID,
			p.DWZ, p.DWZIndex, p.FideID, p.FideRating, p.BirthYear,
			p.Gender, p.Score, p.Rank))
	}
	buf.WriteString("\n")

	// Results section
	buf.WriteString("[Ergebnisse]\n")
	buf.WriteString("Nr;Runde;Gegner;Farbe;Ergebnis\n")
	for _, p := range g.report.Players {
		for _, r := range p.Rounds {
			result := g.convertResult(r.Result)
			buf.WriteString(fmt.Sprintf("%d;%d;%d;%s;%s\n",
				p.StartNumber, r.Round, r.OpponentNum, r.Color, result))
		}
	}
	buf.WriteString("\n")

	// Games section (detailed)
	buf.WriteString("[Partien]\n")
	buf.WriteString("Runde;Weiss-PKZ;Weiss-Name;Weiss-DWZ;Schwarz-PKZ;Schwarz-Name;Schwarz-DWZ;Ergebnis\n")
	for _, game := range g.report.Games {
		buf.WriteString(fmt.Sprintf("%d;%s;%s;%d;%s;%s;%d;%s\n",
			game.Round, game.WhitePKZ, game.WhiteName, game.WhiteDWZ,
			game.BlackPKZ, game.BlackName, game.BlackDWZ, game.Result))
	}

	return buf.Bytes(), nil
}

// GenerateXML generates the DWZ report in XML format (DeWIS compatible)
func (g *DWZGenerator) GenerateXML() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString("<dwz-auswertung>\n")

	// Tournament info
	buf.WriteString("  <turnier>\n")
	buf.WriteString(fmt.Sprintf("    <name>%s</name>\n", escapeXML(g.report.Tournament.Name)))
	buf.WriteString(fmt.Sprintf("    <ort>%s</ort>\n", escapeXML(g.report.Tournament.City)))
	buf.WriteString(fmt.Sprintf("    <beginn>%s</beginn>\n", g.report.Tournament.StartDate.Format("2006-01-02")))
	buf.WriteString(fmt.Sprintf("    <ende>%s</ende>\n", g.report.Tournament.EndDate.Format("2006-01-02")))
	buf.WriteString(fmt.Sprintf("    <runden>%d</runden>\n", g.report.Tournament.Rounds))
	buf.WriteString(fmt.Sprintf("    <system>%s</system>\n", escapeXML(g.report.Tournament.PairingSystem)))
	buf.WriteString(fmt.Sprintf("    <schiedsrichter>%s</schiedsrichter>\n", escapeXML(g.report.Tournament.ChiefArbiter)))
	buf.WriteString(fmt.Sprintf("    <bedenkzeit>%s</bedenkzeit>\n", escapeXML(g.report.Tournament.TimeControl)))
	if g.report.Tournament.TurnierCode != "" {
		buf.WriteString(fmt.Sprintf("    <turniercode>%s</turniercode>\n", escapeXML(g.report.Tournament.TurnierCode)))
	}
	buf.WriteString("  </turnier>\n")

	// Players
	buf.WriteString("  <teilnehmer>\n")
	for _, p := range g.report.Players {
		buf.WriteString("    <spieler>\n")
		buf.WriteString(fmt.Sprintf("      <nr>%d</nr>\n", p.StartNumber))
		buf.WriteString(fmt.Sprintf("      <pkz>%s</pkz>\n", escapeXML(p.PKZ)))
		buf.WriteString(fmt.Sprintf("      <name>%s</name>\n", escapeXML(p.LastName)))
		buf.WriteString(fmt.Sprintf("      <vorname>%s</vorname>\n", escapeXML(p.FirstName)))
		buf.WriteString(fmt.Sprintf("      <verein>%s</verein>\n", escapeXML(p.ClubID)))
		buf.WriteString(fmt.Sprintf("      <dwz>%d</dwz>\n", p.DWZ))
		buf.WriteString(fmt.Sprintf("      <dwz-index>%d</dwz-index>\n", p.DWZIndex))
		if p.FideID != "" {
			buf.WriteString(fmt.Sprintf("      <fide-id>%s</fide-id>\n", escapeXML(p.FideID)))
			buf.WriteString(fmt.Sprintf("      <elo>%d</elo>\n", p.FideRating))
		}
		buf.WriteString(fmt.Sprintf("      <geburtsjahr>%d</geburtsjahr>\n", p.BirthYear))
		buf.WriteString(fmt.Sprintf("      <geschlecht>%s</geschlecht>\n", p.Gender))
		buf.WriteString(fmt.Sprintf("      <punkte>%.1f</punkte>\n", p.Score))
		buf.WriteString(fmt.Sprintf("      <rang>%d</rang>\n", p.Rank))

		// Round results
		buf.WriteString("      <runden>\n")
		for _, r := range p.Rounds {
			buf.WriteString("        <runde>\n")
			buf.WriteString(fmt.Sprintf("          <nr>%d</nr>\n", r.Round))
			buf.WriteString(fmt.Sprintf("          <gegner>%d</gegner>\n", r.OpponentNum))
			buf.WriteString(fmt.Sprintf("          <farbe>%s</farbe>\n", r.Color))
			buf.WriteString(fmt.Sprintf("          <ergebnis>%s</ergebnis>\n", g.convertResult(r.Result)))
			buf.WriteString("        </runde>\n")
		}
		buf.WriteString("      </runden>\n")

		buf.WriteString("    </spieler>\n")
	}
	buf.WriteString("  </teilnehmer>\n")

	buf.WriteString("</dwz-auswertung>\n")

	return buf.Bytes(), nil
}

// convertResult converts result notation
func (g *DWZGenerator) convertResult(result string) string {
	switch result {
	case "1":
		return "1"
	case "0":
		return "0"
	case "=", "0.5":
		return "0.5"
	case "+":
		return "+"
	case "-":
		return "-"
	default:
		return result
	}
}

// Validate checks if the report is valid for submission
func (g *DWZGenerator) Validate() []string {
	var errors []string

	if g.report.Tournament.Name == "" {
		errors = append(errors, "Tournament name is required")
	}
	if g.report.Tournament.StartDate.IsZero() {
		errors = append(errors, "Start date is required")
	}
	if g.report.Tournament.EndDate.IsZero() {
		errors = append(errors, "End date is required")
	}
	if g.report.Tournament.Rounds == 0 {
		errors = append(errors, "Number of rounds is required")
	}
	if len(g.report.Players) == 0 {
		errors = append(errors, "At least one player is required")
	}

	// Check players have PKZ
	for _, p := range g.report.Players {
		if p.PKZ == "" {
			errors = append(errors, fmt.Sprintf("Player %d (%s %s) missing PKZ", p.StartNumber, p.FirstName, p.LastName))
		}
	}

	return errors
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// DWZReportService generates DWZ reports from tournament data
type DWZReportService struct {
	tournamentRepo TournamentDataProvider
}

// TournamentDataProvider interface for fetching tournament data
type TournamentDataProvider interface {
	GetTournament(ctx context.Context, id uuid.UUID) (*TournamentData, error)
	GetParticipants(ctx context.Context, tournamentID uuid.UUID) ([]ParticipantData, error)
	GetGames(ctx context.Context, tournamentID uuid.UUID) ([]GameData, error)
}

// TournamentData contains tournament info for reports
type TournamentData struct {
	ID            uuid.UUID
	Name          string
	StartDate     time.Time
	EndDate       time.Time
	City          string
	Organizer     string
	ChiefArbiter  string
	TimeControl   string
	Rounds        int
	PairingSystem string
}

// ParticipantData contains participant info for reports
type ParticipantData struct {
	StartNumber int
	PlayerID    uuid.UUID
	FirstName   string
	LastName    string
	DsbID       string
	ClubID      string
	ClubName    string
	DWZ         int
	DWZIndex    int
	FideID      string
	FideRating  int
	BirthYear   int
	Gender      string
	Score       float64
	Rank        int
}

// GameData contains game info for reports
type GameData struct {
	Round       int
	WhiteID     uuid.UUID
	BlackID     uuid.UUID
	WhiteScore  float64
	BlackScore  float64
}

// NewDWZReportService creates a new DWZ report service
func NewDWZReportService(repo TournamentDataProvider) *DWZReportService {
	return &DWZReportService{
		tournamentRepo: repo,
	}
}

// GenerateReport generates a DWZ report for a tournament
func (s *DWZReportService) GenerateReport(ctx context.Context, tournamentID uuid.UUID) (*DWZReport, error) {
	tournament, err := s.tournamentRepo.GetTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament: %w", err)
	}

	participants, err := s.tournamentRepo.GetParticipants(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	games, err := s.tournamentRepo.GetGames(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}

	gen := NewDWZGenerator()
	gen.SetTournamentInfo(DWZTournamentInfo{
		Name:          tournament.Name,
		StartDate:     tournament.StartDate,
		EndDate:       tournament.EndDate,
		City:          tournament.City,
		Organizer:     tournament.Organizer,
		ChiefArbiter:  tournament.ChiefArbiter,
		TimeControl:   tournament.TimeControl,
		Rounds:        tournament.Rounds,
		PairingSystem: tournament.PairingSystem,
	})

	// Build participant lookup
	participantByID := make(map[uuid.UUID]*ParticipantData)
	participantByNum := make(map[int]*ParticipantData)
	for i := range participants {
		p := &participants[i]
		participantByID[p.PlayerID] = p
		participantByNum[p.StartNumber] = p
	}

	// Build round results for each player
	playerRounds := make(map[int][]DWZRoundResult)
	for _, game := range games {
		white := participantByID[game.WhiteID]
		black := participantByID[game.BlackID]

		if white != nil {
			playerRounds[white.StartNumber] = append(playerRounds[white.StartNumber], DWZRoundResult{
				Round:       game.Round,
				OpponentNum: black.StartNumber,
				Color:       "w",
				Result:      scoreToResult(game.WhiteScore),
			})
		}
		if black != nil {
			playerRounds[black.StartNumber] = append(playerRounds[black.StartNumber], DWZRoundResult{
				Round:       game.Round,
				OpponentNum: white.StartNumber,
				Color:       "s",
				Result:      scoreToResult(game.BlackScore),
			})
		}

		// Add game record
		if white != nil && black != nil {
			gen.AddGame(DWZGameRecord{
				Round:     game.Round,
				WhitePKZ:  white.DsbID,
				WhiteName: white.LastName + ", " + white.FirstName,
				WhiteDWZ:  white.DWZ,
				BlackPKZ:  black.DsbID,
				BlackName: black.LastName + ", " + black.FirstName,
				BlackDWZ:  black.DWZ,
				Result:    gameResultString(game.WhiteScore, game.BlackScore),
			})
		}
	}

	// Add players with their rounds
	for _, p := range participants {
		gen.AddPlayer(DWZPlayerRecord{
			StartNumber: p.StartNumber,
			PKZ:         p.DsbID,
			Name:        p.LastName + ", " + p.FirstName,
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			ClubID:      p.ClubID,
			ClubName:    p.ClubName,
			DWZ:         p.DWZ,
			DWZIndex:    p.DWZIndex,
			FideID:      p.FideID,
			FideRating:  p.FideRating,
			BirthYear:   p.BirthYear,
			Gender:      p.Gender,
			Score:       p.Score,
			Rank:        p.Rank,
			Rounds:      playerRounds[p.StartNumber],
		})
	}

	return gen.report, nil
}

func scoreToResult(score float64) string {
	if score == 1.0 {
		return "1"
	} else if score == 0.5 {
		return "="
	} else {
		return "0"
	}
}

func gameResultString(whiteScore, blackScore float64) string {
	if whiteScore == 1.0 {
		return "1-0"
	} else if blackScore == 1.0 {
		return "0-1"
	} else {
		return "1/2-1/2"
	}
}
