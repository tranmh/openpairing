package export

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTournament() *TournamentExport {
	endDate := time.Date(2025, 7, 5, 0, 0, 0, 0, time.UTC)

	return &TournamentExport{
		ID:              uuid.New(),
		Name:            "City Championship 2025",
		Site:            "Chess Club Berlin",
		StartDate:       time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:         &endDate,
		Rounds:          7,
		Type:            "Swiss",
		TimeControl:     "90+30",
		TimeControlType: "standard",
		PairingSystem:   "Dutch",
		Chief:           "IA John Smith",
		Deputy:          "FA Jane Doe",
		Federation:      "GER",
		Participants: []ParticipantExport{
			{
				ID:              uuid.New(),
				StartRank:       1,
				Name:            "Carlsen, Magnus",
				FideID:          strPtr("1503014"),
				Rating:          2850,
				Title:           strPtr("GM"),
				Federation:      strPtr("NOR"),
				Club:            strPtr("OSG Baden-Baden"),
				Points:          6.0,
				Wins:            5,
				Draws:           2,
				Losses:          0,
				Buchholz:        35.5,
				BuchholzCut1:    30.0,
				SonnebornBerger: 28.5,
				Rank:            1,
				Results: []RoundResult{
					{RoundNumber: 1, OpponentRank: intPtr(8), Color: "w", Score: "1"},
					{RoundNumber: 2, OpponentRank: intPtr(5), Color: "b", Score: "1"},
					{RoundNumber: 3, OpponentRank: intPtr(3), Color: "w", Score: "="},
					{RoundNumber: 4, OpponentRank: intPtr(2), Color: "b", Score: "1"},
					{RoundNumber: 5, OpponentRank: intPtr(4), Color: "w", Score: "1"},
					{RoundNumber: 6, OpponentRank: intPtr(6), Color: "b", Score: "1"},
					{RoundNumber: 7, OpponentRank: intPtr(7), Color: "w", Score: "="},
				},
			},
			{
				ID:              uuid.New(),
				StartRank:       2,
				Name:            "Nakamura, Hikaru",
				FideID:          strPtr("2016192"),
				Rating:          2780,
				Title:           strPtr("GM"),
				Federation:      strPtr("USA"),
				Club:            strPtr("Saint Louis"),
				Points:          5.5,
				Wins:            4,
				Draws:           3,
				Losses:          0,
				Buchholz:        34.0,
				BuchholzCut1:    29.0,
				SonnebornBerger: 26.0,
				Rank:            2,
				Results: []RoundResult{
					{RoundNumber: 1, OpponentRank: intPtr(7), Color: "b", Score: "1"},
					{RoundNumber: 2, OpponentRank: intPtr(6), Color: "w", Score: "="},
					{RoundNumber: 3, OpponentRank: intPtr(4), Color: "b", Score: "1"},
					{RoundNumber: 4, OpponentRank: intPtr(1), Color: "w", Score: "0"},
					{RoundNumber: 5, OpponentRank: intPtr(3), Color: "b", Score: "1"},
					{RoundNumber: 6, OpponentRank: intPtr(5), Color: "w", Score: "1"},
					{RoundNumber: 7, OpponentRank: intPtr(8), Color: "b", Score: "1"},
				},
			},
			{
				ID:              uuid.New(),
				StartRank:       3,
				Name:            "Caruana, Fabiano",
				FideID:          strPtr("2020009"),
				Rating:          2770,
				Title:           strPtr("GM"),
				Federation:      strPtr("USA"),
				Club:            strPtr("Obiettivo Risarcimento"),
				Points:          5.0,
				Wins:            4,
				Draws:           2,
				Losses:          1,
				Buchholz:        33.5,
				BuchholzCut1:    28.5,
				SonnebornBerger: 24.0,
				Rank:            3,
				Results: []RoundResult{
					{RoundNumber: 1, OpponentRank: intPtr(6), Color: "w", Score: "1"},
					{RoundNumber: 2, OpponentRank: intPtr(4), Color: "b", Score: "="},
					{RoundNumber: 3, OpponentRank: intPtr(1), Color: "b", Score: "="},
					{RoundNumber: 4, OpponentRank: intPtr(5), Color: "w", Score: "1"},
					{RoundNumber: 5, OpponentRank: intPtr(2), Color: "w", Score: "0"},
					{RoundNumber: 6, OpponentRank: intPtr(8), Color: "b", Score: "1"},
					{RoundNumber: 7, OpponentRank: intPtr(4), Color: "w", Score: "1"},
				},
			},
			{
				ID:              uuid.New(),
				StartRank:       4,
				Name:            "Ding, Liren",
				FideID:          strPtr("8603677"),
				Rating:          2760,
				Title:           strPtr("GM"),
				Federation:      strPtr("CHN"),
				Points:          4.5,
				Wins:            3,
				Draws:           3,
				Losses:          1,
				Buchholz:        33.0,
				BuchholzCut1:    28.0,
				SonnebornBerger: 22.0,
				Rank:            4,
				Results: []RoundResult{
					{RoundNumber: 1, OpponentRank: intPtr(5), Color: "b", Score: "="},
					{RoundNumber: 2, OpponentRank: intPtr(3), Color: "w", Score: "="},
					{RoundNumber: 3, OpponentRank: intPtr(2), Color: "w", Score: "0"},
					{RoundNumber: 4, OpponentRank: intPtr(8), Color: "b", Score: "1"},
					{RoundNumber: 5, OpponentRank: intPtr(1), Color: "b", Score: "0"},
					{RoundNumber: 6, OpponentRank: intPtr(7), Color: "w", Score: "1"},
					{RoundNumber: 7, OpponentRank: intPtr(3), Color: "b", Score: "0"},
				},
			},
		},
	}
}

// strPtr and intPtr are defined in pgn_test.go

func TestPDFGenerator_GenerateStandingsPDF(t *testing.T) {
	g := NewPDFGenerator()
	tournament := createTestTournament()

	pdfBytes, err := g.GenerateStandingsPDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Check PDF header (magic bytes)
	assert.True(t, len(pdfBytes) > 4)
	assert.Equal(t, "%PDF", string(pdfBytes[:4]))
}

func TestPDFGenerator_GenerateCrosstablePDF(t *testing.T) {
	g := NewPDFGenerator()
	tournament := createTestTournament()

	pdfBytes, err := g.GenerateCrosstablePDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Check PDF header (magic bytes)
	assert.True(t, len(pdfBytes) > 4)
	assert.Equal(t, "%PDF", string(pdfBytes[:4]))
}

func TestPDFGenerator_EmptyTournament(t *testing.T) {
	g := NewPDFGenerator()
	tournament := &TournamentExport{
		ID:           uuid.New(),
		Name:         "Empty Tournament",
		Site:         "Test Site",
		StartDate:    time.Now(),
		Rounds:       5,
		Type:         "Swiss",
		TimeControl:  "90+30",
		Participants: []ParticipantExport{},
	}

	// Standings PDF
	pdfBytes, err := g.GenerateStandingsPDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Crosstable PDF
	pdfBytes, err = g.GenerateCrosstablePDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}

func TestPDFGenerator_LongPlayerName(t *testing.T) {
	g := NewPDFGenerator()
	tournament := &TournamentExport{
		ID:          uuid.New(),
		Name:        "Test Tournament",
		Site:        "Test Site",
		StartDate:   time.Now(),
		Rounds:      3,
		Type:        "Swiss",
		TimeControl: "90+30",
		Participants: []ParticipantExport{
			{
				ID:        uuid.New(),
				StartRank: 1,
				Name:      "Very Long Player Name That Should Be Truncated In The PDF Output",
				Rating:    2000,
				Points:    2.0,
				Wins:      2,
				Draws:     0,
				Losses:    1,
				Rank:      1,
				Results: []RoundResult{
					{RoundNumber: 1, OpponentRank: intPtr(2), Color: "w", Score: "1"},
					{RoundNumber: 2, OpponentRank: intPtr(3), Color: "b", Score: "1"},
					{RoundNumber: 3, OpponentRank: intPtr(4), Color: "w", Score: "0"},
				},
			},
		},
	}

	pdfBytes, err := g.GenerateStandingsPDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	pdfBytes, err = g.GenerateCrosstablePDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}

func TestPDFGenerator_ManyRounds(t *testing.T) {
	g := NewPDFGenerator()
	tournament := &TournamentExport{
		ID:          uuid.New(),
		Name:        "Long Tournament",
		Site:        "Test Site",
		StartDate:   time.Now(),
		Rounds:      11,
		Type:        "Swiss",
		TimeControl: "90+30",
		Participants: []ParticipantExport{
			{
				ID:        uuid.New(),
				StartRank: 1,
				Name:      "Player One",
				Rating:    2000,
				Points:    8.0,
				Wins:      7,
				Draws:     2,
				Losses:    2,
				Rank:      1,
			},
		},
	}

	// With many rounds, should use landscape orientation
	pdfBytes, err := g.GenerateStandingsPDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}

func TestPDFGenerator_ManyParticipants(t *testing.T) {
	g := NewPDFGenerator()
	tournament := &TournamentExport{
		ID:          uuid.New(),
		Name:        "Large Tournament",
		Site:        "Test Site",
		StartDate:   time.Now(),
		Rounds:      5,
		Type:        "Swiss",
		TimeControl: "90+30",
	}

	// Create 50 participants
	for i := 1; i <= 50; i++ {
		tournament.Participants = append(tournament.Participants, ParticipantExport{
			ID:        uuid.New(),
			StartRank: i,
			Name:      "Player " + string(rune('A'+i-1)),
			Rating:    2000 - i*10,
			Points:    float64(50-i) / 10.0,
			Wins:      (50 - i) / 5,
			Draws:     i % 3,
			Losses:    i / 5,
			Rank:      i,
		})
	}

	// Should handle pagination
	pdfBytes, err := g.GenerateStandingsPDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	pdfBytes, err = g.GenerateCrosstablePDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}

func TestFormatPoints(t *testing.T) {
	tests := []struct {
		points   float64
		expected string
	}{
		{6.0, "6"},
		{5.5, "5.5"},
		{4.0, "4"},
		{3.5, "3.5"},
		{0.0, "0"},
		{0.5, "0.5"},
	}

	for _, tt := range tests {
		result := formatPoints(tt.points)
		assert.Equal(t, tt.expected, result, "formatPoints(%v)", tt.points)
	}
}

func TestColorCode(t *testing.T) {
	tests := []struct {
		color    string
		expected string
	}{
		{"w", "w"},
		{"b", "s"},
		{"-", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := colorCode(tt.color)
		assert.Equal(t, tt.expected, result, "colorCode(%v)", tt.color)
	}
}

func TestPDFGenerator_GenerateParticipantListPDF(t *testing.T) {
	g := NewPDFGenerator()
	tournament := createTestTournament()

	pdfBytes, err := g.GenerateParticipantListPDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Check PDF header (magic bytes)
	assert.True(t, len(pdfBytes) > 4)
	assert.Equal(t, "%PDF", string(pdfBytes[:4]))
}

func TestPDFGenerator_GenerateParticipantListPDF_Empty(t *testing.T) {
	g := NewPDFGenerator()
	tournament := &TournamentExport{
		ID:           uuid.New(),
		Name:         "Empty Tournament",
		Site:         "Test Site",
		StartDate:    time.Now(),
		Rounds:       5,
		Type:         "Swiss",
		Participants: []ParticipantExport{},
	}

	pdfBytes, err := g.GenerateParticipantListPDF(tournament)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}

func TestPDFGenerator_GeneratePairingsPDF(t *testing.T) {
	g := NewPDFGenerator()
	pairings := &PairingsExport{
		TournamentName: "Test Tournament",
		Site:           "Test Site",
		RoundNumber:    1,
		Date:           time.Now(),
		Games: []PairingGame{
			{BoardNumber: 1, WhiteName: "Player A", WhiteRating: 2000, BlackName: "Player B", BlackRating: 1900, Result: ""},
			{BoardNumber: 2, WhiteName: "Player C", WhiteRating: 1850, BlackName: "Player D", BlackRating: 1800, Result: "1-0"},
			{BoardNumber: 3, WhiteName: "Player E", WhiteRating: 1750, BlackName: "Player F", BlackRating: 1700, Result: "0-1"},
			{BoardNumber: 4, WhiteName: "Player G", WhiteRating: 1650, BlackName: "Player H", BlackRating: 1600, Result: "1/2-1/2"},
		},
	}

	pdfBytes, err := g.GeneratePairingsPDF(pairings)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Check PDF header (magic bytes)
	assert.True(t, len(pdfBytes) > 4)
	assert.Equal(t, "%PDF", string(pdfBytes[:4]))
}

func TestPDFGenerator_GeneratePairingsPDF_Empty(t *testing.T) {
	g := NewPDFGenerator()
	pairings := &PairingsExport{
		TournamentName: "Test Tournament",
		RoundNumber:    1,
		Date:           time.Now(),
		Games:          []PairingGame{},
	}

	pdfBytes, err := g.GeneratePairingsPDF(pairings)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}

func TestPDFGenerator_GeneratePairingsPDF_WithBye(t *testing.T) {
	g := NewPDFGenerator()
	pairings := &PairingsExport{
		TournamentName: "Test Tournament",
		Site:           "Test Site",
		RoundNumber:    1,
		Date:           time.Now(),
		Games: []PairingGame{
			{BoardNumber: 1, WhiteName: "Player A", WhiteRating: 2000, BlackName: "Player B", BlackRating: 1900, Result: "1-0"},
			{BoardNumber: 2, WhiteName: "Player C", WhiteRating: 1850, BlackName: "", BlackRating: 0, Result: "BYE"},
		},
	}

	pdfBytes, err := g.GeneratePairingsPDF(pairings)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}
