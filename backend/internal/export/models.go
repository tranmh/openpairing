package export

import (
	"time"

	"github.com/google/uuid"
)

// ExportFormat represents the output format for exports
type ExportFormat string

const (
	FormatPGN  ExportFormat = "pgn"
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
	FormatTXT  ExportFormat = "txt"
	FormatPDF  ExportFormat = "pdf"
)

// PGNHeader contains standard PGN seven tag roster headers
type PGNHeader struct {
	Event      string
	Site       string
	Date       string
	Round      string
	White      string
	Black      string
	Result     string
	WhiteElo   *int
	BlackElo   *int
	WhiteTitle *string
	BlackTitle *string
	WhiteFideID *string
	BlackFideID *string
	ECO        *string
	Opening    *string
	TimeControl string
	Termination string
}

// GameExport represents a single game for export
type GameExport struct {
	ID           uuid.UUID
	TournamentID *uuid.UUID
	RoundNumber  *int
	BoardNumber  *int
	Header       PGNHeader
	Moves        string // Standard Algebraic Notation (SAN) moves
	FEN          string // Starting FEN if not standard
	CreatedAt    time.Time
}

// TournamentExport contains all data needed to export a tournament
type TournamentExport struct {
	ID               uuid.UUID
	Name             string
	Site             string
	StartDate        time.Time
	EndDate          *time.Time
	Rounds           int
	Type             string
	TimeControl      string
	TimeControlType  string
	PairingSystem    string
	Chief            string
	Deputy           string
	Federation       string
	Games            []GameExport
	Participants     []ParticipantExport
	ParticipantCount int
}

// ParticipantExport represents a participant for export
type ParticipantExport struct {
	ID             uuid.UUID
	StartRank      int
	Name           string
	FideID         *string
	Rating         int
	Title          *string
	Federation     *string
	Club           *string
	Points         float64
	Wins           int
	Draws          int
	Losses         int
	Buchholz       float64
	BuchholzCut1   float64
	SonnebornBerger float64
	Rank           int
	Results        []RoundResult
}

// RoundResult represents a single round result for a participant
type RoundResult struct {
	RoundNumber  int
	OpponentRank *int
	Color        string // "w" or "b" or "-" for bye
	Score        string // "1", "0", "=", "+" (forfeit win), "-" (forfeit loss)
}

// PairingsExport contains data for pairings PDF export
type PairingsExport struct {
	TournamentName string
	Site           string
	RoundNumber    int
	Date           time.Time
	Games          []PairingGame
}

// PairingGame represents a single pairing for export
type PairingGame struct {
	BoardNumber int
	WhiteName   string
	WhiteRating int
	BlackName   string
	BlackRating int
	Result      string
}

// ExportRequest represents a request to export data
type ExportRequest struct {
	TournamentID *uuid.UUID  `json:"tournamentId,omitempty"`
	GameID       *uuid.UUID  `json:"gameId,omitempty"`
	RoundNumber  *int        `json:"roundNumber,omitempty"`
	Format       ExportFormat `json:"format" binding:"required,oneof=pgn json csv txt"`
	IncludeMoves bool        `json:"includeMoves"`
}

// ExportResponse contains the exported data
type ExportResponse struct {
	Format   ExportFormat `json:"format"`
	Filename string       `json:"filename"`
	Content  string       `json:"content"`
	MimeType string       `json:"mimeType"`
}

// GetMimeType returns the MIME type for an export format
func (f ExportFormat) GetMimeType() string {
	switch f {
	case FormatPGN:
		return "application/x-chess-pgn"
	case FormatJSON:
		return "application/json"
	case FormatCSV:
		return "text/csv"
	case FormatTXT:
		return "text/plain"
	case FormatPDF:
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// GetFileExtension returns the file extension for an export format
func (f ExportFormat) GetFileExtension() string {
	switch f {
	case FormatPGN:
		return ".pgn"
	case FormatJSON:
		return ".json"
	case FormatCSV:
		return ".csv"
	case FormatTXT:
		return ".txt"
	case FormatPDF:
		return ".pdf"
	default:
		return ""
	}
}
