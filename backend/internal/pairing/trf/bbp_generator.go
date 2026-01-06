package trf

import (
	"fmt"
	"math"
	"strings"
)

// BBPInitialColorConfig represents initial color configuration for bbpPairings
type BBPInitialColorConfig string

const (
	// BBPColorWhite1 - player 1 gets white in round 1 (default)
	BBPColorWhite1 BBPInitialColorConfig = "white1"
	// BBPColorBlack1 - player 1 gets black in round 1
	BBPColorBlack1 BBPInitialColorConfig = "black1"
	// BBPColorRank - color based on ranking
	BBPColorRank BBPInitialColorConfig = "rank"
)

// BBPGenerator generates TRF files compatible with bbpPairings (TRFbx format).
//
// bbpPairings TRFbx format specifics:
//   - REQUIRES XXC record for initial color configuration
//   - Supports additional point system records: BBW, BBD, BBL, BBZ, BBF, BBU
//   - Validates player scores strictly against tournament results
//   - Only supports Dutch and Burstein systems (NO Dubov)
type BBPGenerator struct {
	baseGenerator
	initialColorConfig BBPInitialColorConfig
}

// NewBBPGenerator creates a new bbpPairings-compatible TRF generator
func NewBBPGenerator() *BBPGenerator {
	return &BBPGenerator{
		baseGenerator:      newBaseGenerator(),
		initialColorConfig: BBPColorWhite1, // Default
	}
}

// EngineName returns the engine name
func (g *BBPGenerator) EngineName() string {
	return "bbpPairings"
}

// SetInitialColorConfig sets the initial color configuration.
// This is REQUIRED for bbpPairings. Default is BBPColorWhite1.
func (g *BBPGenerator) SetInitialColorConfig(config BBPInitialColorConfig) {
	g.initialColorConfig = config
}

// Generate creates the TRF string for bbpPairings.
// Important: Always includes XXC record (required by bbpPairings).
func (g *BBPGenerator) Generate() string {
	var lines []string

	// Header (basic codes only)
	lines = append(lines, g.generateCommonHeader()...)

	// XXC record - REQUIRED for bbpPairings
	// If no XXC records provided, use the configured default
	if len(g.trf.XXCRecords) == 0 {
		lines = append(lines, fmt.Sprintf("XXC %s", g.initialColorConfig))
	} else {
		for _, xxc := range g.trf.XXCRecords {
			lines = append(lines, fmt.Sprintf("XXC %s", xxc))
		}
	}

	// XXR record - number of rounds (required)
	if g.trf.Rounds > 0 {
		lines = append(lines, fmt.Sprintf("XXR %d", g.trf.Rounds))
	}

	// Blank line before players (122 record)
	lines = append(lines, "122")

	// Players
	lines = append(lines, g.generatePlayersSection()...)

	// Teams
	lines = append(lines, g.generateTeamsSection()...)

	// Player-specific XXR records
	lines = append(lines, g.generatePlayerXXRRecords()...)

	return strings.Join(lines, "\n")
}

// Validate validates the TRF data for bbpPairings compatibility.
// bbpPairings is stricter than JaVaFo - declared points must match calculated points.
func (g *BBPGenerator) Validate() []ValidationError {
	errors := g.baseValidate()

	// bbpPairings-specific validations

	// Validate player scores match their results (bbpPairings is strict about this)
	for _, player := range g.trf.Players {
		if len(player.RoundResults) > 0 {
			calculatedPoints := g.calculatePoints(player.RoundResults)
			// Use epsilon comparison to handle floating point precision issues
			if math.Abs(player.Points-calculatedPoints) > 0.001 {
				errors = append(errors, ValidationError{
					Field: fmt.Sprintf("Player[%d].Points", player.StartNo),
					Message: fmt.Sprintf("Player %d: declared points (%.1f) don't match calculated points (%.1f) from results",
						player.StartNo, player.Points, calculatedPoints),
					Fatal: true, // bbpPairings will reject this
				})
			}
		}
	}

	// Validate all players have consistent round results
	if len(g.trf.Players) > 0 {
		expectedRounds := len(g.trf.Players[0].RoundResults)
		for _, player := range g.trf.Players[1:] {
			if len(player.RoundResults) != expectedRounds {
				errors = append(errors, ValidationError{
					Field: fmt.Sprintf("Player[%d].RoundResults", player.StartNo),
					Message: fmt.Sprintf("Player %d has %d round results, expected %d",
						player.StartNo, len(player.RoundResults), expectedRounds),
					Fatal: false, // Warning only
				})
			}
		}
	}

	return errors
}

// calculatePoints calculates total points from round results
func (g *BBPGenerator) calculatePoints(results []TRFRoundResult) float64 {
	var points float64
	for _, r := range results {
		switch r.Result {
		case "1", "+", "W", "F":
			points += 1.0
		case "=", "D", "H":
			points += 0.5
		// "0", "-", "L", "Z", "U" = 0 points
		}
	}
	return points
}

// Verify BBPGenerator implements TRFGenerator interface
var _ TRFGenerator = (*BBPGenerator)(nil)
