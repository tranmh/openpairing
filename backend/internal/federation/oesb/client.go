package oesb

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	// ÖSB API base URL (Austrian Chess Federation)
	OESBBaseURL = "https://chess.at/api/"

	// Endpoints
	PlayerEndpoint  = "player"
	SearchEndpoint  = "search"
	ClubEndpoint    = "club"
	RatingEndpoint  = "ratings"
)

// Client handles API requests to the Austrian Chess Federation
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	logger     zerolog.Logger
}

// ClientConfig contains configuration for the ÖSB client
type ClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// NewClient creates a new ÖSB API client
func NewClient(cfg *ClientConfig, logger zerolog.Logger) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = OESBBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		logger:  logger.With().Str("component", "oesb_client").Logger(),
	}
}

// OESBPlayer represents a player from the Austrian Chess Federation
type OESBPlayer struct {
	ID         string `json:"id"`           // ÖSB player ID
	PKZ        string `json:"pkz"`          // Personenkennzahl
	Name       string `json:"name"`
	FirstName  string `json:"vorname"`
	LastName   string `json:"nachname"`
	Title      string `json:"titel"`
	Rating     int    `json:"elo"`          // ÖSB ELO
	FideID     string `json:"fide_id"`
	FideRating int    `json:"fide_elo"`
	FideTitle  string `json:"fide_titel"`
	ClubID     string `json:"verein_id"`
	ClubName   string `json:"verein_name"`
	Federation string `json:"landesverband"`
	Gender     string `json:"geschlecht"`
	BirthYear  int    `json:"geburtsjahr"`
	Status     string `json:"status"`
	LastUpdate string `json:"letzte_aenderung"`
}

// OESBClub represents a club from the Austrian Chess Federation
type OESBClub struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ShortName  string `json:"kurzname"`
	Federation string `json:"landesverband"`
	City       string `json:"ort"`
	Members    int    `json:"mitglieder"`
}

// GetPlayer retrieves a player by their ÖSB ID
func (c *Client) GetPlayer(ctx context.Context, id string) (*OESBPlayer, error) {
	params := url.Values{
		"id":     {id},
		"format": {"json"},
	}

	if c.apiKey != "" {
		params.Set("key", c.apiKey)
	}

	requestURL := fmt.Sprintf("%s%s?%s", c.baseURL, PlayerEndpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/tranmh/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrPlayerNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var player OESBPlayer
	if err := json.NewDecoder(resp.Body).Decode(&player); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &player, nil
}

// SearchPlayers searches for players by name
func (c *Client) SearchPlayers(ctx context.Context, query string, limit int) ([]*OESBPlayer, error) {
	params := url.Values{
		"q":      {query},
		"format": {"json"},
		"limit":  {strconv.Itoa(limit)},
	}

	if c.apiKey != "" {
		params.Set("key", c.apiKey)
	}

	requestURL := fmt.Sprintf("%s%s?%s", c.baseURL, SearchEndpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/tranmh/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var players []*OESBPlayer
	if err := json.NewDecoder(resp.Body).Decode(&players); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return players, nil
}

// GetRatingsList downloads the full Austrian rating list
func (c *Client) GetRatingsList(ctx context.Context) ([]*OESBPlayer, error) {
	params := url.Values{
		"format": {"csv"},
		"all":    {"1"},
	}

	if c.apiKey != "" {
		params.Set("key", c.apiKey)
	}

	requestURL := fmt.Sprintf("%s%s?%s", c.baseURL, RatingEndpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/tranmh/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return c.parsePlayersCSV(resp.Body)
}

// parsePlayersCSV parses players from CSV format
func (c *Client) parsePlayersCSV(reader io.Reader) ([]*OESBPlayer, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = ';'
	csvReader.LazyQuotes = true

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Create column index map
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(col)] = i
	}

	var players []*OESBPlayer

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to read CSV record")
			continue
		}

		player := &OESBPlayer{}

		// Extract fields by column index
		if idx, ok := colIndex["id"]; ok && idx < len(record) {
			player.ID = record[idx]
		}
		if idx, ok := colIndex["pkz"]; ok && idx < len(record) {
			player.PKZ = record[idx]
		}
		if idx, ok := colIndex["vorname"]; ok && idx < len(record) {
			player.FirstName = record[idx]
		}
		if idx, ok := colIndex["nachname"]; ok && idx < len(record) {
			player.LastName = record[idx]
		}
		if idx, ok := colIndex["titel"]; ok && idx < len(record) {
			player.Title = record[idx]
		}
		if idx, ok := colIndex["elo"]; ok && idx < len(record) {
			if rating, err := strconv.Atoi(record[idx]); err == nil {
				player.Rating = rating
			}
		}
		if idx, ok := colIndex["fide_id"]; ok && idx < len(record) {
			player.FideID = record[idx]
		}
		if idx, ok := colIndex["fide_elo"]; ok && idx < len(record) {
			if rating, err := strconv.Atoi(record[idx]); err == nil {
				player.FideRating = rating
			}
		}
		if idx, ok := colIndex["fide_titel"]; ok && idx < len(record) {
			player.FideTitle = record[idx]
		}
		if idx, ok := colIndex["verein_id"]; ok && idx < len(record) {
			player.ClubID = record[idx]
		}
		if idx, ok := colIndex["verein"]; ok && idx < len(record) {
			player.ClubName = record[idx]
		}
		if idx, ok := colIndex["landesverband"]; ok && idx < len(record) {
			player.Federation = record[idx]
		}
		if idx, ok := colIndex["geschlecht"]; ok && idx < len(record) {
			player.Gender = record[idx]
		}
		if idx, ok := colIndex["geburtsjahr"]; ok && idx < len(record) {
			if year, err := strconv.Atoi(record[idx]); err == nil {
				player.BirthYear = year
			}
		}

		if player.ID != "" || player.PKZ != "" {
			if player.FirstName != "" && player.LastName != "" {
				player.Name = fmt.Sprintf("%s, %s", player.LastName, player.FirstName)
			}
			players = append(players, player)
		}
	}

	return players, nil
}

// GetClub retrieves a club by ID
func (c *Client) GetClub(ctx context.Context, clubID string) (*OESBClub, error) {
	params := url.Values{
		"id":     {clubID},
		"format": {"json"},
	}

	if c.apiKey != "" {
		params.Set("key", c.apiKey)
	}

	requestURL := fmt.Sprintf("%s%s?%s", c.baseURL, ClubEndpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/tranmh/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrClubNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var club OESBClub
	if err := json.NewDecoder(resp.Body).Decode(&club); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &club, nil
}

// GetClubMembers retrieves all members of a club
func (c *Client) GetClubMembers(ctx context.Context, clubID string) ([]*OESBPlayer, error) {
	params := url.Values{
		"id":       {clubID},
		"members":  {"1"},
		"format":   {"json"},
	}

	if c.apiKey != "" {
		params.Set("key", c.apiKey)
	}

	requestURL := fmt.Sprintf("%s%s?%s", c.baseURL, ClubEndpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/tranmh/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var players []*OESBPlayer
	if err := json.NewDecoder(resp.Body).Decode(&players); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return players, nil
}

// Errors
var (
	ErrPlayerNotFound = fmt.Errorf("player not found")
	ErrClubNotFound   = fmt.Errorf("club not found")
)
