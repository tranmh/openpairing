package trf

import (
	"fmt"
	"strings"
)

// JaVaFoGenerator generates TRF files compatible with JaVaFo (TRFx format).
//
// JaVaFo TRFx format specifics:
//   - Supports: 012, 022, 032, 042, 052, XXR, XXC, XXA, XXP
//   - FAILS on unrecognized codes: 062, 072, 082, 092, 102, 112
//   - XXR is required for round count
//   - XXC is optional (for initial color preferences)
//   - Supports Dutch, Burstein, and Dubov pairing systems
type JaVaFoGenerator struct {
	baseGenerator
}

// NewJaVaFoGenerator creates a new JaVaFo-compatible TRF generator
func NewJaVaFoGenerator() *JaVaFoGenerator {
	return &JaVaFoGenerator{
		baseGenerator: newBaseGenerator(),
	}
}

// EngineName returns the engine name
func (g *JaVaFoGenerator) EngineName() string {
	return "JaVaFo"
}

// Generate creates the TRF string for JaVaFo.
// Important: Does NOT include header codes 062, 072, 082, 092, 102, 112
// as JaVaFo fails when encountering unrecognized codes.
func (g *JaVaFoGenerator) Generate() string {
	var lines []string

	// Header (basic codes only - JaVaFo fails on 062, 072, etc.)
	lines = append(lines, g.generateCommonHeader()...)

	// XXR record - number of rounds (REQUIRED by JaVaFo)
	if g.trf.Rounds > 0 {
		lines = append(lines, fmt.Sprintf("XXR %d", g.trf.Rounds))
	}

	// XXC records (optional for JaVaFo - initial color preferences)
	for _, xxc := range g.trf.XXCRecords {
		lines = append(lines, fmt.Sprintf("XXC %s", xxc))
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

// Validate validates the TRF data for JaVaFo compatibility
func (g *JaVaFoGenerator) Validate() []ValidationError {
	errors := g.baseValidate()

	// JaVaFo-specific validations

	// Validate player scores are reasonable (warning only)
	for _, player := range g.trf.Players {
		maxPoints := float64(len(player.RoundResults))
		if player.Points > maxPoints && len(player.RoundResults) > 0 {
			errors = append(errors, ValidationError{
				Field: fmt.Sprintf("Player[%d].Points", player.StartNo),
				Message: fmt.Sprintf("Player %d: points (%.1f) exceed maximum possible (%.1f)",
					player.StartNo, player.Points, maxPoints),
				Fatal: false, // Warning only for JaVaFo
			})
		}
	}

	return errors
}

// Verify JaVaFoGenerator implements TRFGenerator interface
var _ TRFGenerator = (*JaVaFoGenerator)(nil)
