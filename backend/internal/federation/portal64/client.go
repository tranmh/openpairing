package portal64

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

const (
	// DefaultBaseURL is the default portal64 API base URL
	DefaultBaseURL = "https://test.svw.info:8080"

	// API endpoints
	PlayersEndpoint = "/api/v1/players"
	ClubsEndpoint   = "/api/v1/clubs"
)

// Client handles API requests to the portal64 API
type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     zerolog.Logger
}

// ClientConfig contains configuration for the portal64 client
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// NewClient creates a new portal64 API client
func NewClient(cfg *ClientConfig, logger zerolog.Logger) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL: cfg.BaseURL,
		logger:  logger.With().Str("component", "portal64_client").Logger(),
	}
}

// Player represents a player from the portal64 API
type Player struct {
	ID         string `json:"id"`          // VKZ-Spielernummer format (e.g., "C0101-1014")
	PKZ        string `json:"pkz"`         // Personenkennziffer (unique DSB ID)
	Name       string `json:"name"`        // Last name
	Firstname  string `json:"firstname"`   // First name
	Club       string `json:"club"`        // Club name
	ClubID     string `json:"club_id"`     // Club ID (VKZ format)
	BirthYear  *int   `json:"birth_year"`  // Birth year only (GDPR compliant)
	Gender     string `json:"gender"`      // "m", "w", or "d"
	Nation     string `json:"nation"`      // Country code
	FideID     uint   `json:"fide_id"`     // FIDE ID
	CurrentDWZ int    `json:"current_dwz"` // Current DWZ rating
	DWZIndex   int    `json:"dwz_index"`   // DWZ index
	Status     string `json:"status"`      // "active" or "inactive"
}

// APIResponse wraps all API responses
type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error,omitempty"`
}

// SearchResponse represents the paginated search response
type SearchResponse struct {
	Data []*Player   `json:"data"`
	Meta SearchMeta  `json:"meta"`
}

// SearchMeta contains pagination metadata
type SearchMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
}

// GetPlayer retrieves a player by their portal64 ID (VKZ-Spielernummer format)
func (c *Client) GetPlayer(ctx context.Context, id string) (*Player, error) {
	requestURL := fmt.Sprintf("%s%s/%s", c.baseURL, PlayersEndpoint, url.PathEscape(id))

	c.logger.Debug().Str("id", id).Str("url", requestURL).Msg("Fetching player from portal64")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/openpairing/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrPlayerNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrInvalidResponse, resp.StatusCode)
	}

	// Parse the wrapped API response
	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("%w: %s", ErrInvalidResponse, apiResp.Error)
	}

	var player Player
	if err := json.Unmarshal(apiResp.Data, &player); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	return &player, nil
}

// SearchPlayers searches for players by name
func (c *Client) SearchPlayers(ctx context.Context, query string, limit int) ([]*Player, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	params := url.Values{
		"query":  {query},
		"limit":  {strconv.Itoa(limit)},
		"active": {"true"},
	}

	requestURL := fmt.Sprintf("%s%s?%s", c.baseURL, PlayersEndpoint, params.Encode())

	c.logger.Debug().Str("query", query).Int("limit", limit).Msg("Searching players in portal64")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/openpairing/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrInvalidResponse, resp.StatusCode)
	}

	// Parse the wrapped API response
	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("%w: %s", ErrInvalidResponse, apiResp.Error)
	}

	// Parse the nested search response
	var searchResp SearchResponse
	if err := json.Unmarshal(apiResp.Data, &searchResp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	c.logger.Debug().
		Int("results", len(searchResp.Data)).
		Int("total", searchResp.Meta.Total).
		Msg("Search completed")

	return searchResp.Data, nil
}

// GetClubPlayers retrieves all players in a club by club ID (VKZ format)
func (c *Client) GetClubPlayers(ctx context.Context, clubID string, limit int) ([]*Player, error) {
	if limit <= 0 {
		limit = 500
	}

	params := url.Values{
		"limit":  {strconv.Itoa(limit)},
		"active": {"true"},
	}

	requestURL := fmt.Sprintf("%s%s/%s/players?%s", c.baseURL, ClubsEndpoint, url.PathEscape(clubID), params.Encode())

	c.logger.Debug().Str("clubID", clubID).Msg("Fetching club players from portal64")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/openpairing/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrClubNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrInvalidResponse, resp.StatusCode)
	}

	// Parse the wrapped API response
	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("%w: %s", ErrInvalidResponse, apiResp.Error)
	}

	// Parse the nested search response
	var searchResp SearchResponse
	if err := json.Unmarshal(apiResp.Data, &searchResp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	return searchResp.Data, nil
}

// SearchPlayersByPKZ searches for a player by PKZ
// This uses the general search and filters by PKZ
func (c *Client) SearchPlayersByPKZ(ctx context.Context, pkz string) (*Player, error) {
	players, err := c.SearchPlayers(ctx, pkz, 10)
	if err != nil {
		return nil, err
	}

	// Find exact PKZ match
	for _, p := range players {
		if p.PKZ == pkz {
			return p, nil
		}
	}

	return nil, ErrPlayerNotFound
}

// HealthCheck checks if the portal64 API is available
func (c *Client) HealthCheck(ctx context.Context) error {
	requestURL := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAPIUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: health check returned %d", ErrAPIUnavailable, resp.StatusCode)
	}

	return nil
}
