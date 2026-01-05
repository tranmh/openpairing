package portal64

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/player"
	"github.com/rs/zerolog"
)

// Service provides portal64 player data with caching
type Service struct {
	client     *Client
	playerRepo player.Repository
	logger     zerolog.Logger

	// Cache configuration
	playerCacheTTL time.Duration
	searchCacheTTL time.Duration

	// In-memory cache (can be replaced with Redis later)
	playerCache *cacheStore
	searchCache *cacheStore
}

// ServiceConfig contains configuration for the portal64 service
type ServiceConfig struct {
	BaseURL        string
	Timeout        time.Duration
	PlayerCacheTTL time.Duration // Default: 1 hour
	SearchCacheTTL time.Duration // Default: 15 minutes
}

// NewService creates a new portal64 service
func NewService(playerRepo player.Repository, cfg *ServiceConfig, logger zerolog.Logger) *Service {
	if cfg.PlayerCacheTTL == 0 {
		cfg.PlayerCacheTTL = 1 * time.Hour
	}
	if cfg.SearchCacheTTL == 0 {
		cfg.SearchCacheTTL = 15 * time.Minute
	}

	clientCfg := &ClientConfig{
		BaseURL: cfg.BaseURL,
		Timeout: cfg.Timeout,
	}

	return &Service{
		client:         NewClient(clientCfg, logger),
		playerRepo:     playerRepo,
		logger:         logger.With().Str("component", "portal64_service").Logger(),
		playerCacheTTL: cfg.PlayerCacheTTL,
		searchCacheTTL: cfg.SearchCacheTTL,
		playerCache:    newCacheStore(),
		searchCache:    newCacheStore(),
	}
}

// SearchPlayers searches for players via portal64 API (cached, does NOT persist to DB)
func (s *Service) SearchPlayers(ctx context.Context, query string, limit int) ([]*Player, error) {
	cacheKey := fmt.Sprintf("search:%s:%d", query, limit)

	// Check cache first
	if cached, ok := s.searchCache.get(cacheKey); ok {
		s.logger.Debug().Str("query", query).Msg("Search results from cache")
		var players []*Player
		if err := json.Unmarshal(cached, &players); err == nil {
			return players, nil
		}
	}

	// Fetch from API
	players, err := s.client.SearchPlayers(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	// Cache the results
	if data, err := json.Marshal(players); err == nil {
		s.searchCache.set(cacheKey, data, s.searchCacheTTL)
	}

	return players, nil
}

// GetPlayer retrieves a player by PKZ from cache or API (does NOT persist to DB)
func (s *Service) GetPlayer(ctx context.Context, pkz string) (*Player, error) {
	cacheKey := fmt.Sprintf("player:%s", pkz)

	// Check cache first
	if cached, ok := s.playerCache.get(cacheKey); ok {
		s.logger.Debug().Str("pkz", pkz).Msg("Player from cache")
		var p Player
		if err := json.Unmarshal(cached, &p); err == nil {
			return &p, nil
		}
	}

	// Try to find by PKZ via search
	p, err := s.client.SearchPlayersByPKZ(ctx, pkz)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(p); err == nil {
		s.playerCache.set(cacheKey, data, s.playerCacheTTL)
	}

	return p, nil
}

// GetPlayerByID retrieves a player by portal64 ID (VKZ-Spielernummer format)
func (s *Service) GetPlayerByID(ctx context.Context, id string) (*Player, error) {
	cacheKey := fmt.Sprintf("player_id:%s", id)

	// Check cache first
	if cached, ok := s.playerCache.get(cacheKey); ok {
		s.logger.Debug().Str("id", id).Msg("Player from cache")
		var p Player
		if err := json.Unmarshal(cached, &p); err == nil {
			return &p, nil
		}
	}

	// Fetch from API
	p, err := s.client.GetPlayer(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(p); err == nil {
		s.playerCache.set(cacheKey, data, s.playerCacheTTL)
	}

	return p, nil
}

// ImportPlayer fetches a player from portal64 and persists to the local database
// This should be called when a player is registered for a tournament
// id can be either VKZ-Spielernummer (e.g., "56024-173") or PKZ (e.g., "10017516")
func (s *Service) ImportPlayer(ctx context.Context, id string) (*player.Player, error) {
	s.logger.Info().Str("id", id).Msg("Importing player from portal64")

	// Determine if id is VKZ format (contains "-") or PKZ format
	isVKZFormat := strings.Contains(id, "-")

	// First check if player already exists in our database by trying both formats
	var existing *player.Player
	var err error
	if isVKZFormat {
		// VKZ format - we don't have a direct lookup by VKZ, so fetch from API first
		existing = nil
	} else {
		// PKZ format - try to find by DSB ID
		existing, err = s.playerRepo.GetByDsbID(ctx, id)
	}

	if err == nil && existing != nil {
		s.logger.Debug().Str("id", id).Msg("Player already exists in database")

		// Optionally refresh from API to get latest data
		var p64 *Player
		if isVKZFormat {
			p64, err = s.GetPlayerByID(ctx, id)
		} else {
			p64, err = s.GetPlayer(ctx, id)
		}
		if err == nil {
			// Update with fresh data
			updated := s.ConvertToPlayer(p64)
			updated.ID = existing.ID
			if err := s.playerRepo.UpsertByDsbID(ctx, updated); err != nil {
				s.logger.Warn().Err(err).Str("id", id).Msg("Failed to update player with fresh data")
			}
			// Fetch updated player by PKZ
			return s.playerRepo.GetByDsbID(ctx, p64.PKZ)
		}

		return existing, nil
	}

	// Fetch from portal64
	var p64 *Player
	if isVKZFormat {
		p64, err = s.GetPlayerByID(ctx, id)
	} else {
		p64, err = s.GetPlayer(ctx, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player from portal64: %w", err)
	}

	// Convert and save
	p := s.ConvertToPlayer(p64)
	if err := s.playerRepo.UpsertByDsbID(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to save player: %w", err)
	}

	// Fetch the saved player (to get the generated ID) using PKZ from API response
	return s.playerRepo.GetByDsbID(ctx, p64.PKZ)
}

// GetClubPlayers retrieves all players in a club (cached)
func (s *Service) GetClubPlayers(ctx context.Context, clubID string) ([]*Player, error) {
	cacheKey := fmt.Sprintf("club:%s", clubID)

	// Check cache first
	if cached, ok := s.searchCache.get(cacheKey); ok {
		s.logger.Debug().Str("clubID", clubID).Msg("Club players from cache")
		var players []*Player
		if err := json.Unmarshal(cached, &players); err == nil {
			return players, nil
		}
	}

	// Fetch from API
	players, err := s.client.GetClubPlayers(ctx, clubID, 500)
	if err != nil {
		return nil, err
	}

	// Cache the results (30 min for club lists)
	if data, err := json.Marshal(players); err == nil {
		s.searchCache.set(cacheKey, data, 30*time.Minute)
	}

	return players, nil
}

// ConvertToPlayer converts a portal64 Player to the internal player model
func (s *Service) ConvertToPlayer(p64 *Player) *player.Player {
	now := time.Now()

	p := &player.Player{
		ID:               uuid.Nil, // Will be set by repository on insert
		FirstName:        p64.Firstname,
		LastName:         p64.Name,
		DsbID:            &p64.PKZ,
		RatingsUpdatedAt: &now,
	}

	// Set federation
	federation := "GER"
	p.Federation = &federation

	// Set DWZ if present
	if p64.CurrentDWZ > 0 {
		p.DWZ = &p64.CurrentDWZ
	}
	if p64.DWZIndex > 0 {
		p.DWZIndex = &p64.DWZIndex
	}

	// Set FIDE ID if present
	if p64.FideID > 0 {
		fideID := fmt.Sprintf("%d", p64.FideID)
		p.FideID = &fideID
	}

	// Set birth date if year is present
	if p64.BirthYear != nil && *p64.BirthYear > 0 {
		birthDate := fmt.Sprintf("%d-01-01", *p64.BirthYear)
		p.BirthDate = &birthDate
	}

	// Set gender
	if p64.Gender != "" {
		gender := p64.Gender
		// Normalize to our format
		switch gender {
		case "m", "M":
			gender = "m"
		case "w", "W", "f", "F":
			gender = "f"
		default:
			gender = "other"
		}
		p.Gender = &gender
	}

	// Set club name
	if p64.Club != "" {
		p.ClubName = &p64.Club
	}

	return p
}

// HealthCheck checks if the portal64 API is available
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.client.HealthCheck(ctx)
}

// ClearCache clears all cached data
func (s *Service) ClearCache() {
	s.playerCache.clear()
	s.searchCache.clear()
	s.logger.Info().Msg("Portal64 cache cleared")
}

// GetCacheStats returns cache statistics
func (s *Service) GetCacheStats() map[string]int {
	return map[string]int{
		"players": s.playerCache.size(),
		"search":  s.searchCache.size(),
	}
}

// cacheStore is a simple in-memory cache with TTL support
type cacheStore struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

type cacheItem struct {
	data      []byte
	expiresAt time.Time
}

func newCacheStore() *cacheStore {
	cs := &cacheStore{
		items: make(map[string]*cacheItem),
	}
	// Start cleanup goroutine
	go cs.cleanup()
	return cs
}

func (cs *cacheStore) get(key string) ([]byte, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	item, ok := cs.items[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.data, true
}

func (cs *cacheStore) set(key string, data []byte, ttl time.Duration) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.items[key] = &cacheItem{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

func (cs *cacheStore) clear() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.items = make(map[string]*cacheItem)
}

func (cs *cacheStore) size() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.items)
}

func (cs *cacheStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		cs.mu.Lock()
		now := time.Now()
		for key, item := range cs.items {
			if now.After(item.expiresAt) {
				delete(cs.items, key)
			}
		}
		cs.mu.Unlock()
	}
}
