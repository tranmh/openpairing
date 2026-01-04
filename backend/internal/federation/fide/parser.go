package fide

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// FIDEPlayer represents a player from the FIDE XML rating list
type FIDEPlayer struct {
	FideID      string  `xml:"fideid"`
	Name        string  `xml:"name"`
	Country     string  `xml:"country"`
	Sex         string  `xml:"sex"`
	Title       string  `xml:"title"`
	WTitle      string  `xml:"w_title"`
	OTitle      string  `xml:"o_title"`
	FOATitle    string  `xml:"foa_title"`
	Rating      int     `xml:"rating"`
	Games       int     `xml:"games"`
	K           int     `xml:"k"`
	RapidRating int     `xml:"rapid_rating"`
	RapidGames  int     `xml:"rapid_games"`
	RapidK      int     `xml:"rapid_k"`
	BlitzRating int     `xml:"blitz_rating"`
	BlitzGames  int     `xml:"blitz_games"`
	BlitzK      int     `xml:"blitz_k"`
	Birthday    string  `xml:"birthday"`
	Flag        string  `xml:"flag"`
}

// ParsedName contains parsed first and last name
type ParsedName struct {
	FirstName string
	LastName  string
}

// ParseName parses a FIDE name (usually "LASTNAME, Firstname") into components
func ParseName(name string) ParsedName {
	name = strings.TrimSpace(name)

	// FIDE format: "LASTNAME, Firstname" or "LASTNAME Firstname"
	if idx := strings.Index(name, ","); idx > 0 {
		lastName := strings.TrimSpace(name[:idx])
		firstName := strings.TrimSpace(name[idx+1:])

		// Convert LASTNAME to Lastname
		lastName = capitalizeWords(lastName)
		firstName = capitalizeWords(firstName)

		return ParsedName{FirstName: firstName, LastName: lastName}
	}

	// No comma, try to split on space
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		// First part(s) might be last name in UPPERCASE
		var lastNameParts []string
		var firstNameParts []string

		for i, part := range parts {
			if strings.ToUpper(part) == part && len(part) > 1 {
				// All uppercase = likely last name
				lastNameParts = append(lastNameParts, part)
			} else {
				// Mixed case = first name starts here
				firstNameParts = append(firstNameParts, parts[i:]...)
				break
			}
		}

		if len(lastNameParts) > 0 && len(firstNameParts) > 0 {
			return ParsedName{
				FirstName: capitalizeWords(strings.Join(firstNameParts, " ")),
				LastName:  capitalizeWords(strings.Join(lastNameParts, " ")),
			}
		}
	}

	// Fallback: first word is last name
	if len(parts) > 0 {
		lastName := capitalizeWords(parts[0])
		firstName := ""
		if len(parts) > 1 {
			firstName = capitalizeWords(strings.Join(parts[1:], " "))
		}
		return ParsedName{FirstName: firstName, LastName: lastName}
	}

	return ParsedName{FirstName: name, LastName: ""}
}

// capitalizeWords converts "WORD" to "Word"
func capitalizeWords(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			// Handle special cases like "von", "van", "de"
			lower := strings.ToLower(word)
			if lower == "von" || lower == "van" || lower == "de" || lower == "der" || lower == "den" {
				words[i] = lower
			} else {
				words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
			}
		}
	}
	return strings.Join(words, " ")
}

// GetTitle returns the player's highest title
func (p *FIDEPlayer) GetTitle() string {
	// Priority: GM/IM/FM/CM > WGM/WIM/WFM/WCM > other titles
	if p.Title != "" {
		return p.Title
	}
	if p.WTitle != "" {
		return p.WTitle
	}
	if p.OTitle != "" {
		return p.OTitle
	}
	return p.FOATitle
}

// GetGender returns normalized gender
func (p *FIDEPlayer) GetGender() string {
	switch strings.ToUpper(p.Sex) {
	case "M":
		return "m"
	case "F":
		return "f"
	default:
		return ""
	}
}

// GetBirthDate parses and returns the birth date
func (p *FIDEPlayer) GetBirthDate() *string {
	if p.Birthday == "" {
		return nil
	}
	// FIDE uses YYYY or YYYY-MM-DD format
	birthday := strings.TrimSpace(p.Birthday)

	// Try full date first
	if _, err := time.Parse("2006-01-02", birthday); err == nil {
		return &birthday
	}

	// Try year only
	if len(birthday) == 4 {
		if _, err := strconv.Atoi(birthday); err == nil {
			date := birthday + "-01-01"
			return &date
		}
	}

	return nil
}

// Parser handles streaming parsing of FIDE XML files
type Parser struct {
	logger zerolog.Logger
}

// NewParser creates a new FIDE XML parser
func NewParser(logger zerolog.Logger) *Parser {
	return &Parser{
		logger: logger.With().Str("component", "fide_parser").Logger(),
	}
}

// ParseResult contains parsing statistics
type ParseResult struct {
	TotalPlayers    int
	ParsedPlayers   int
	SkippedPlayers  int
	ErrorCount      int
	ProcessingTime  time.Duration
}

// PlayerHandler is called for each parsed player
type PlayerHandler func(player *FIDEPlayer) error

// ParseXML parses a FIDE XML file using streaming to handle large files
func (p *Parser) ParseXML(reader io.Reader, handler PlayerHandler) (*ParseResult, error) {
	startTime := time.Now()
	result := &ParseResult{}

	decoder := xml.NewDecoder(reader)

	// Find the start of the player list
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read XML token: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "player" {
				var player FIDEPlayer
				if err := decoder.DecodeElement(&player, &se); err != nil {
					p.logger.Warn().Err(err).Msg("Failed to decode player")
					result.ErrorCount++
					continue
				}

				result.TotalPlayers++

				// Skip players without FIDE ID
				if player.FideID == "" {
					result.SkippedPlayers++
					continue
				}

				// Call handler
				if err := handler(&player); err != nil {
					p.logger.Warn().Err(err).Str("fideId", player.FideID).Msg("Handler error")
					result.ErrorCount++
					continue
				}

				result.ParsedPlayers++

				// Log progress every 10000 players
				if result.ParsedPlayers%10000 == 0 {
					p.logger.Info().
						Int("parsed", result.ParsedPlayers).
						Int("total", result.TotalPlayers).
						Msg("Parsing progress")
				}
			}
		}
	}

	result.ProcessingTime = time.Since(startTime)

	p.logger.Info().
		Int("total", result.TotalPlayers).
		Int("parsed", result.ParsedPlayers).
		Int("skipped", result.SkippedPlayers).
		Int("errors", result.ErrorCount).
		Dur("duration", result.ProcessingTime).
		Msg("Parsing complete")

	return result, nil
}

// ParseXMLFromFile parses a FIDE XML file from disk
func (p *Parser) ParseXMLFromFile(path string, handler PlayerHandler) (*ParseResult, error) {
	reader, err := NewStreamReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	return p.ParseXML(reader, handler)
}

// FilterDACH returns a handler that only processes DACH region players
func FilterDACH(handler PlayerHandler) PlayerHandler {
	dachCountries := map[string]bool{
		"GER": true,
		"AUT": true,
		"SUI": true,
	}

	return func(player *FIDEPlayer) error {
		if !dachCountries[player.Country] {
			return nil // Skip non-DACH players
		}
		return handler(player)
	}
}

// FilterMinRating returns a handler that only processes players with minimum rating
func FilterMinRating(minRating int, handler PlayerHandler) PlayerHandler {
	return func(player *FIDEPlayer) error {
		if player.Rating < minRating && player.RapidRating < minRating && player.BlitzRating < minRating {
			return nil // Skip low-rated players
		}
		return handler(player)
	}
}
