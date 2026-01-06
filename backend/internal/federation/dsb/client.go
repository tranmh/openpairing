package dsb

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
	// DeWIS API base URL
	DeWISBaseURL = "https://www.schachbund.de/php/dewis/"

	// API endpoints
	PlayerEndpoint = "spieler.php"
	ClubEndpoint   = "verein.php"
	SearchEndpoint = "suche.php"
)

// Client handles API requests to the DSB DeWIS system
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	logger     zerolog.Logger
}

// ClientConfig contains configuration for the DSB client
type ClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// NewClient creates a new DSB API client
func NewClient(cfg *ClientConfig, logger zerolog.Logger) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DeWISBaseURL
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
		logger:  logger.With().Str("component", "dsb_client").Logger(),
	}
}

// DSBPlayer represents a player from the DSB API
type DSBPlayer struct {
	PKZ         string  `json:"pkz"`          // Personenkennziffer (unique ID)
	Name        string  `json:"name"`
	FirstName   string  `json:"vorname"`
	LastName    string  `json:"nachname"`
	Title       string  `json:"titel"`
	DWZ         int     `json:"dwz"`
	DWZIndex    int     `json:"dwz_index"`
	FideID      string  `json:"fide_id"`
	FideElo     int     `json:"fide_elo"`
	FideTitle   string  `json:"fide_titel"`
	ClubID      string  `json:"verein_id"`
	ClubName    string  `json:"verein_name"`
	Federation  string  `json:"verband"`
	Gender      string  `json:"geschlecht"`
	BirthYear   int     `json:"geburtsjahr"`
	Status      string  `json:"status"`
	LastUpdated string  `json:"letzte_aenderung"`
}

// DSBClub represents a club from the DSB API
type DSBClub struct {
	ID         string `json:"verein_id"`
	ZPS        string `json:"zps"`        // Vereinsnummer
	Name       string `json:"name"`
	ShortName  string `json:"kurzname"`
	Federation string `json:"verband"`
	City       string `json:"ort"`
	Members    int    `json:"mitglieder"`
}

// GetPlayer retrieves a player by their PKZ
func (c *Client) GetPlayer(ctx context.Context, pkz string) (*DSBPlayer, error) {
	params := url.Values{
		"pkz":    {pkz},
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

	var player DSBPlayer
	if err := json.NewDecoder(resp.Body).Decode(&player); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &player, nil
}

// GetPlayerCSV retrieves a player in CSV format
func (c *Client) GetPlayerCSV(ctx context.Context, pkz string) (*DSBPlayer, error) {
	params := url.Values{
		"pkz":    {pkz},
		"format": {"csv"},
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

	return c.parsePlayerCSV(resp.Body)
}

// parsePlayerCSV parses a single player from CSV response
func (c *Client) parsePlayerCSV(reader io.Reader) (*DSBPlayer, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = ';'
	csvReader.LazyQuotes = true

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Read data row
	record, err := csvReader.Read()
	if err == io.EOF {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV record: %w", err)
	}

	// Map header to record
	data := make(map[string]string)
	for i, col := range header {
		if i < len(record) {
			data[strings.ToLower(col)] = record[i]
		}
	}

	player := &DSBPlayer{
		PKZ:       data["pkz"],
		FirstName: data["vorname"],
		LastName:  data["nachname"],
		Title:     data["titel"],
		FideID:    data["fide_id"],
		FideTitle: data["fide_titel"],
		ClubID:    data["verein_id"],
		ClubName:  data["verein"],
		Federation: data["verband"],
		Gender:    data["geschlecht"],
		Status:    data["status"],
	}

	// Parse numeric fields
	if dwz, err := strconv.Atoi(data["dwz"]); err == nil {
		player.DWZ = dwz
	}
	if dwzIndex, err := strconv.Atoi(data["index"]); err == nil {
		player.DWZIndex = dwzIndex
	}
	if fideElo, err := strconv.Atoi(data["fide_elo"]); err == nil {
		player.FideElo = fideElo
	}
	if birthYear, err := strconv.Atoi(data["geburtsjahr"]); err == nil {
		player.BirthYear = birthYear
	}

	if player.FirstName != "" && player.LastName != "" {
		player.Name = fmt.Sprintf("%s, %s", player.LastName, player.FirstName)
	}

	return player, nil
}

// SearchPlayers searches for players by name or PKZ
func (c *Client) SearchPlayers(ctx context.Context, query string, limit int) ([]*DSBPlayer, error) {
	params := url.Values{
		"suche":  {query},
		"format": {"csv"},
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

	return c.parsePlayersCSV(resp.Body)
}

// parsePlayersCSV parses multiple players from CSV response
func (c *Client) parsePlayersCSV(reader io.Reader) ([]*DSBPlayer, error) {
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

	var players []*DSBPlayer

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to read CSV record")
			continue
		}

		player := &DSBPlayer{}

		// Extract fields by column index
		if idx, ok := colIndex["pkz"]; ok && idx < len(record) {
			player.PKZ = record[idx]
		}
		if idx, ok := colIndex["vorname"]; ok && idx < len(record) {
			player.FirstName = record[idx]
		}
		if idx, ok := colIndex["nachname"]; ok && idx < len(record) {
			player.LastName = record[idx]
		}
		if idx, ok := colIndex["dwz"]; ok && idx < len(record) {
			if dwz, err := strconv.Atoi(record[idx]); err == nil {
				player.DWZ = dwz
			}
		}
		if idx, ok := colIndex["index"]; ok && idx < len(record) {
			if dwzIndex, err := strconv.Atoi(record[idx]); err == nil {
				player.DWZIndex = dwzIndex
			}
		}
		if idx, ok := colIndex["verein"]; ok && idx < len(record) {
			player.ClubName = record[idx]
		}
		if idx, ok := colIndex["verband"]; ok && idx < len(record) {
			player.Federation = record[idx]
		}

		if player.PKZ != "" {
			players = append(players, player)
		}
	}

	return players, nil
}

// GetClub retrieves a club by ZPS number
func (c *Client) GetClub(ctx context.Context, zps string) (*DSBClub, error) {
	params := url.Values{
		"zps":    {zps},
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

	var club DSBClub
	if err := json.NewDecoder(resp.Body).Decode(&club); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &club, nil
}

// GetClubMembers retrieves all players in a club
func (c *Client) GetClubMembers(ctx context.Context, zps string) ([]*DSBPlayer, error) {
	params := url.Values{
		"zps":      {zps},
		"mitglied": {"1"},
		"format":   {"csv"},
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

	return c.parsePlayersCSV(resp.Body)
}

// Errors
var (
	ErrPlayerNotFound = fmt.Errorf("player not found")
	ErrClubNotFound   = fmt.Errorf("club not found")
)
