package ssb

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tranmh/openpairing/internal/player"
	"github.com/rs/zerolog"
)

// Importer handles importing SSB data into the player database
type Importer struct {
	client     *Client
	playerRepo player.Repository
	logger     zerolog.Logger

	// Configuration
	batchSize int
	workers   int
}

// ImporterConfig contains configuration for the importer
type ImporterConfig struct {
	BatchSize int
	Workers   int
	APIKey    string
	Timeout   time.Duration
}

// NewImporter creates a new SSB importer
func NewImporter(repo player.Repository, cfg *ImporterConfig, logger zerolog.Logger) *Importer {
	l := logger.With().Str("component", "ssb_importer").Logger()

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}

	clientCfg := &ClientConfig{
		APIKey:  cfg.APIKey,
		Timeout: cfg.Timeout,
	}

	return &Importer{
		client:     NewClient(clientCfg, l),
		playerRepo: repo,
		logger:     l,
		batchSize:  cfg.BatchSize,
		workers:    cfg.Workers,
	}
}

// ImportResult contains statistics from the import
type ImportResult struct {
	TotalPlayers   int
	NewPlayers     int
	UpdatedPlayers int
	SkippedPlayers int
	Errors         int
	Duration       time.Duration
	mu             sync.Mutex // protects concurrent counter updates
}

// ImportPlayer imports a single player by SSB ID
func (i *Importer) ImportPlayer(ctx context.Context, ssbID string) (*player.Player, error) {
	ssbPlayer, err := i.client.GetPlayer(ctx, ssbID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	p := i.convertToPlayer(ssbPlayer)
	if err := i.playerRepo.UpsertBySsbID(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to upsert player: %w", err)
	}

	return p, nil
}

// SearchAndImport searches for players and imports them
func (i *Importer) SearchAndImport(ctx context.Context, query string, limit int) (*ImportResult, error) {
	result := &ImportResult{}
	startTime := time.Now()

	players, err := i.client.SearchPlayers(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search players: %w", err)
	}

	result.TotalPlayers = len(players)

	for _, ssbPlayer := range players {
		p := i.convertToPlayer(ssbPlayer)
		if err := i.playerRepo.UpsertBySsbID(ctx, p); err != nil {
			i.logger.Warn().Err(err).Str("id", ssbPlayer.ID).Msg("Failed to upsert player")
			result.Errors++
			continue
		}
		result.UpdatedPlayers++
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// ImportClubMembers imports all members of a club
func (i *Importer) ImportClubMembers(ctx context.Context, clubID string) (*ImportResult, error) {
	result := &ImportResult{}
	startTime := time.Now()

	players, err := i.client.GetClubMembers(ctx, clubID)
	if err != nil {
		return nil, fmt.Errorf("failed to get club members: %w", err)
	}

	result.TotalPlayers = len(players)

	// Process in batches
	var batch []*player.Player
	for _, ssbPlayer := range players {
		p := i.convertToPlayer(ssbPlayer)
		batch = append(batch, p)

		if len(batch) >= i.batchSize {
			if err := i.processBatch(ctx, batch, result); err != nil {
				i.logger.Warn().Err(err).Msg("Batch processing error")
			}
			batch = batch[:0]
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		if err := i.processBatch(ctx, batch, result); err != nil {
			i.logger.Warn().Err(err).Msg("Final batch processing error")
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// ImportRatingsList imports the full Swiss rating list
func (i *Importer) ImportRatingsList(ctx context.Context) (*ImportResult, error) {
	result := &ImportResult{}
	startTime := time.Now()

	i.logger.Info().Msg("Downloading SSB ratings list")

	ssbPlayers, err := i.client.GetRatingsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ratings list: %w", err)
	}

	result.TotalPlayers = len(ssbPlayers)
	i.logger.Info().Int("players", result.TotalPlayers).Msg("Downloaded SSB ratings")

	// Create player channel for concurrent processing
	playerChan := make(chan *SSBPlayer, i.batchSize)
	var wg sync.WaitGroup

	// Start worker goroutines
	for w := 0; w < i.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			i.importWorker(ctx, playerChan, result)
		}()
	}

	// Send players to workers
	for _, p := range ssbPlayers {
		select {
		case <-ctx.Done():
			close(playerChan)
			return nil, ctx.Err()
		case playerChan <- p:
		}
	}

	close(playerChan)
	wg.Wait()

	result.Duration = time.Since(startTime)

	i.logger.Info().
		Int("total", result.TotalPlayers).
		Int("updated", result.UpdatedPlayers).
		Int("errors", result.Errors).
		Dur("duration", result.Duration).
		Msg("SSB import complete")

	return result, nil
}

// importWorker processes players from the channel
func (i *Importer) importWorker(ctx context.Context, players <-chan *SSBPlayer, result *ImportResult) {
	var batch []*player.Player

	for ssbPlayer := range players {
		p := i.convertToPlayer(ssbPlayer)
		batch = append(batch, p)

		if len(batch) >= i.batchSize {
			if err := i.processBatch(ctx, batch, result); err != nil {
				i.logger.Warn().Err(err).Msg("Batch processing error")
				result.mu.Lock()
				result.Errors++
				result.mu.Unlock()
			}
			batch = batch[:0]
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		if err := i.processBatch(ctx, batch, result); err != nil {
			i.logger.Warn().Err(err).Msg("Final batch processing error")
			result.mu.Lock()
			result.Errors++
			result.mu.Unlock()
		}
	}
}

// processBatch processes a batch of players
func (i *Importer) processBatch(ctx context.Context, batch []*player.Player, result *ImportResult) error {
	if err := i.playerRepo.BulkUpsertBySsbID(ctx, batch); err != nil {
		return fmt.Errorf("failed to upsert batch: %w", err)
	}

	result.mu.Lock()
	result.UpdatedPlayers += len(batch)
	result.mu.Unlock()

	return nil
}

// convertToPlayer converts an SSB player to our player model
func (i *Importer) convertToPlayer(sp *SSBPlayer) *player.Player {
	now := time.Now()

	p := &player.Player{
		ID:               uuid.Nil, // Will be set by repository
		FirstName:        sp.FirstName,
		LastName:         sp.LastName,
		RatingsUpdatedAt: &now,
	}

	// Set SSB ID
	if sp.ID != "" {
		p.SsbID = &sp.ID
	} else if sp.MemberNumber != "" {
		p.SsbID = &sp.MemberNumber
	}

	// Set federation
	federation := "SUI"
	p.Federation = &federation

	// Set SSB ELO - stored in a generic national rating field
	// Since there's no specific field for SSB rating in the player model,
	// we can use NationalTitle to indicate the source and store in a comment
	// In a production system, you'd add a specific SSBElo field

	// Set FIDE data if present
	if sp.FideID != "" {
		p.FideID = &sp.FideID
	}
	if sp.FideRating > 0 {
		p.FideStandard = &sp.FideRating
	}
	if sp.FideTitle != "" {
		p.FideTitle = &sp.FideTitle
	}

	// Set national title
	if sp.Title != "" {
		p.NationalTitle = &sp.Title
	}

	// Set gender
	if gender := normalizeGender(sp.Gender); gender != "" {
		p.Gender = &gender
	}

	// Set birth year as date string
	if sp.BirthYear > 0 {
		birthDate := fmt.Sprintf("%d-01-01", sp.BirthYear)
		p.BirthDate = &birthDate
	}

	// Set club info
	if sp.ClubID != "" {
		clubID, err := uuid.Parse(sp.ClubID)
		if err == nil {
			p.ClubID = &clubID
		}
	}
	if sp.ClubName != "" {
		p.ClubName = &sp.ClubName
	}

	return p
}

// normalizeGender normalizes gender strings to M/F
func normalizeGender(gender string) string {
	switch strings.ToUpper(strings.TrimSpace(gender)) {
	case "M", "MALE", "MASCULIN", "MÄNNLICH", "H":
		return "M"
	case "F", "FEMALE", "FEMININ", "WEIBLICH", "W", "D":
		return "F"
	default:
		return ""
	}
}

// Scheduler handles periodic syncing of SSB data
type Scheduler struct {
	importer *Importer
	logger   zerolog.Logger
	interval time.Duration

	stopCh     chan struct{}
	lastSync   time.Time
	lastResult *ImportResult
	mu         sync.RWMutex
}

// NewScheduler creates a new SSB sync scheduler
func NewScheduler(importer *Importer, interval time.Duration, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		importer: importer,
		logger:   logger.With().Str("component", "ssb_scheduler").Logger(),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic sync
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info().Dur("interval", s.interval).Msg("Starting SSB scheduler")

	// Initial sync
	go s.sync(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.sync(ctx)
		}
	}
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// sync performs a single sync operation
func (s *Scheduler) sync(ctx context.Context) {
	s.logger.Info().Msg("Starting SSB sync")

	result, err := s.importer.ImportRatingsList(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("SSB sync failed")
		return
	}

	s.mu.Lock()
	s.lastSync = time.Now()
	s.lastResult = result
	s.mu.Unlock()

	s.logger.Info().
		Int("updated", result.UpdatedPlayers).
		Dur("duration", result.Duration).
		Msg("SSB sync completed")
}

// GetStatus returns the current scheduler status
func (s *Scheduler) GetStatus() (time.Time, *ImportResult) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync, s.lastResult
}

// TriggerSync manually triggers a sync
func (s *Scheduler) TriggerSync(ctx context.Context) (*ImportResult, error) {
	result, err := s.importer.ImportRatingsList(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.lastSync = time.Now()
	s.lastResult = result
	s.mu.Unlock()

	return result, nil
}
