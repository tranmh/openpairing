package export

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPGNGenerator_GenerateGame(t *testing.T) {
	gen := NewPGNGenerator()

	tests := []struct {
		name     string
		game     *GameExport
		wantTags []string
	}{
		{
			name: "basic game with all tags",
			game: &GameExport{
				ID: uuid.New(),
				Header: PGNHeader{
					Event:       "Test Tournament",
					Site:        "Berlin, GER",
					Date:        "2024.01.15",
					Round:       "3",
					White:       "Fischer, Bobby",
					Black:       "Spassky, Boris",
					Result:      "1-0",
					WhiteElo:    intPtr(2785),
					BlackElo:    intPtr(2690),
					WhiteTitle:  strPtr("GM"),
					BlackTitle:  strPtr("GM"),
					TimeControl: "90+30",
				},
				Moves: "1. e4 e5 2. Nf3 Nc6",
			},
			wantTags: []string{
				`[Event "Test Tournament"]`,
				`[Site "Berlin, GER"]`,
				`[Date "2024.01.15"]`,
				`[Round "3"]`,
				`[White "Fischer, Bobby"]`,
				`[Black "Spassky, Boris"]`,
				`[Result "1-0"]`,
				`[WhiteElo "2785"]`,
				`[BlackElo "2690"]`,
				`[WhiteTitle "GM"]`,
				`[BlackTitle "GM"]`,
			},
		},
		{
			name: "game without optional tags",
			game: &GameExport{
				ID: uuid.New(),
				Header: PGNHeader{
					Event:  "Simple Game",
					Site:   "Online",
					Date:   "2024.01.15",
					Round:  "1",
					White:  "Player1",
					Black:  "Player2",
					Result: "1/2-1/2",
				},
				Moves: "",
			},
			wantTags: []string{
				`[Event "Simple Game"]`,
				`[White "Player1"]`,
				`[Black "Player2"]`,
				`[Result "1/2-1/2"]`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgn := gen.GenerateGame(tt.game)

			for _, tag := range tt.wantTags {
				if !strings.Contains(pgn, tag) {
					t.Errorf("PGN missing tag %q", tag)
				}
			}

			// Result should appear at the end
			if !strings.HasSuffix(strings.TrimSpace(pgn), tt.game.Header.Result) {
				t.Errorf("PGN should end with result %q", tt.game.Header.Result)
			}
		})
	}
}

func TestPGNGenerator_GenerateTournament(t *testing.T) {
	gen := NewPGNGenerator()

	tournament := &TournamentExport{
		ID:               uuid.New(),
		Name:             "Summer Open 2024",
		Site:             "Munich, GER",
		StartDate:        time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		Rounds:           7,
		ParticipantCount: 50,
		Games: []GameExport{
			{
				ID: uuid.New(),
				Header: PGNHeader{
					Event:  "Summer Open 2024",
					Site:   "Munich, GER",
					Date:   "2024.07.01",
					Round:  "1",
					White:  "Player A",
					Black:  "Player B",
					Result: "1-0",
				},
			},
			{
				ID: uuid.New(),
				Header: PGNHeader{
					Event:  "Summer Open 2024",
					Site:   "Munich, GER",
					Date:   "2024.07.01",
					Round:  "1",
					White:  "Player C",
					Black:  "Player D",
					Result: "0-1",
				},
			},
		},
	}

	pgn := gen.GenerateTournament(tournament)

	// Should contain tournament comment
	if !strings.Contains(pgn, "; Tournament: Summer Open 2024") {
		t.Error("PGN missing tournament name in comment")
	}

	// Should contain both games
	if !strings.Contains(pgn, `[White "Player A"]`) {
		t.Error("PGN missing first game")
	}
	if !strings.Contains(pgn, `[White "Player C"]`) {
		t.Error("PGN missing second game")
	}
}

func TestResultToPGN(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1-0", "1-0"},
		{"0-1", "0-1"},
		{"1/2-1/2", "1/2-1/2"},
		{"1-0F", "1-0"},
		{"0-1F", "0-1"},
		{"0-0F", "0-0"},
		{"unknown", "*"},
		{"", "*"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ResultToPGN(tt.input)
			if result != tt.expected {
				t.Errorf("ResultToPGN(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTerminationString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"checkmate", "Normal"},
		{"resignation", "Resigned"},
		{"timeout", "Time forfeit"},
		{"draw_agreement", "Draw agreed"},
		{"stalemate", "Stalemate"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TerminationString(tt.input)
			if result != tt.expected {
				t.Errorf("TerminationString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExportFormat_GetMimeType(t *testing.T) {
	tests := []struct {
		format   ExportFormat
		expected string
	}{
		{FormatPGN, "application/x-chess-pgn"},
		{FormatJSON, "application/json"},
		{FormatCSV, "text/csv"},
		{FormatTXT, "text/plain"},
		{ExportFormat("unknown"), "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := tt.format.GetMimeType(); got != tt.expected {
				t.Errorf("GetMimeType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExportFormat_GetFileExtension(t *testing.T) {
	tests := []struct {
		format   ExportFormat
		expected string
	}{
		{FormatPGN, ".pgn"},
		{FormatJSON, ".json"},
		{FormatCSV, ".csv"},
		{FormatTXT, ".txt"},
		{ExportFormat("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := tt.format.GetFileExtension(); got != tt.expected {
				t.Errorf("GetFileExtension() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
