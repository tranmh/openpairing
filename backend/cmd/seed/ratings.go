package main

import (
	"fmt"
	"math/rand"
)

// RatingGenerator generates realistic chess ratings
type RatingGenerator struct {
	rand *rand.Rand
}

// NewRatingGenerator creates a new rating generator with the given seed
func NewRatingGenerator(seed int64) *RatingGenerator {
	return &RatingGenerator{
		rand: rand.New(rand.NewSource(seed)),
	}
}

// GenerateRating generates a rating following realistic distribution
// Beginner: 800-1200 (20%)
// Club: 1200-1600 (40%)
// Intermediate: 1600-2000 (30%)
// Advanced: 2000-2300 (8%)
// Expert: 2300-2600 (2%)
func (g *RatingGenerator) GenerateRating() int {
	r := g.rand.Float64()
	switch {
	case r < 0.20: // Beginner
		return 800 + g.rand.Intn(400)
	case r < 0.60: // Club
		return 1200 + g.rand.Intn(400)
	case r < 0.90: // Intermediate
		return 1600 + g.rand.Intn(400)
	case r < 0.98: // Advanced
		return 2000 + g.rand.Intn(300)
	default: // Expert
		return 2300 + g.rand.Intn(300)
	}
}

// GenerateDWZ generates a DWZ rating close to FIDE rating with some variance
func (g *RatingGenerator) GenerateDWZ(fideRating int) int {
	// DWZ is typically within ±50 of FIDE rating
	variance := g.rand.Intn(100) - 50
	dwz := fideRating + variance

	// Ensure minimum rating
	if dwz < 800 {
		dwz = 800
	}
	return dwz
}

// AssignTitle assigns a FIDE title based on rating
func (g *RatingGenerator) AssignTitle(rating int) *string {
	var title string

	switch {
	case rating >= 2500:
		title = "GM"
	case rating >= 2400:
		title = "IM"
	case rating >= 2300:
		title = "FM"
	case rating >= 2200:
		// 50% chance of CM at this level
		if g.rand.Float64() < 0.5 {
			title = "CM"
		}
	}

	if title == "" {
		return nil
	}
	return &title
}

// AssignWomenTitle assigns a women's FIDE title based on rating and gender
func (g *RatingGenerator) AssignWomenTitle(rating int, gender string) *string {
	if gender != "f" {
		return g.AssignTitle(rating)
	}

	var title string

	// Women can have either regular or women's titles
	// Use regular titles for higher ratings
	switch {
	case rating >= 2500:
		title = "GM"
	case rating >= 2400:
		if g.rand.Float64() < 0.3 {
			title = "IM"
		} else {
			title = "WGM"
		}
	case rating >= 2300:
		if g.rand.Float64() < 0.3 {
			title = "FM"
		} else {
			title = "WIM"
		}
	case rating >= 2200:
		title = "WIM"
	case rating >= 2100:
		title = "WFM"
	case rating >= 2000:
		if g.rand.Float64() < 0.3 {
			title = "WCM"
		}
	}

	if title == "" {
		return nil
	}
	return &title
}

// GenerateFederationID generates a federation ID for a given federation
func (g *RatingGenerator) GenerateFederationID(federation string, index int) string {
	switch federation {
	case "GER":
		// DSB PKZ format: 10000000 + index
		return formatPKZ(10000000 + index)
	case "AUT":
		// OESB ID format: A + 6 digits
		return formatOESBID(index)
	case "SUI":
		// SSB ID format: 6 digits
		return formatSSBID(index)
	default:
		return ""
	}
}

// GenerateFideID generates a FIDE ID
func (g *RatingGenerator) GenerateFideID(index int) string {
	// FIDE ID format: 8 digits starting with country code area
	// German: 46XXXXXX, Austrian: 16XXXXXX, Swiss: 14XXXXXX
	return formatFideID(index)
}

func formatPKZ(n int) string {
	// German PKZ: 8 digits
	return fmt.Sprintf("%08d", n)
}

func formatOESBID(n int) string {
	// Austrian: typically numeric
	return fmt.Sprintf("%06d", n+100000)
}

func formatSSBID(n int) string {
	// Swiss: typically numeric
	return fmt.Sprintf("%06d", n+200000)
}

func formatFideID(n int) string {
	// FIDE ID: 8 digits
	return fmt.Sprintf("%08d", 46000000+n)
}

