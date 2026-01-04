package dsb

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/player"
	"github.com/rs/zerolog"
)

// Importer handles importing DSB data into the player database
type Importer struct {
	client     *Client
	playerRepo player.Repository
	logger     zerolog.Logger
	batchSize  int
}

// ImporterConfig contains configuration for the importer
type ImporterConfig struct {
	APIKey    string
	BaseURL   string
	BatchSize int
}

// NewImporter creates a new DSB importer
func NewImporter(repo player.Repository, cfg *ImporterConfig, logger zerolog.Logger) *Importer {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}

	clientCfg := &ClientConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Timeout: 30 * time.Second,
	}

	return &Importer{
		client:     NewClient(clientCfg, logger),
		playerRepo: repo,
		logger:     logger.With().Str("component", "dsb_importer").Logger(),
		batchSize:  cfg.BatchSize,
	}
}

// ImportResult contains statistics from the import
type ImportResult struct {
	ImportTime     time.Duration
	Duration       time.Duration
	TotalPlayers   int
	NewPlayers     int
	UpdatedPlayers int
	SkippedPlayers int
	Errors         int
}

// ImportPlayer imports or updates a single player by PKZ
func (i *Importer) ImportPlayer(ctx context.Context, pkz string) (*player.Player, error) {
	i.logger.Debug().Str("pkz", pkz).Msg("Importing player from DSB")

	// Fetch from DSB
	dsbPlayer, err := i.client.GetPlayer(ctx, pkz)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player from DSB: %w", err)
	}

	// Convert to player model
	p := i.convertToPlayer(dsbPlayer)

	// Upsert
	if err := i.playerRepo.UpsertByDsbID(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to upsert player: %w", err)
	}

	// Fetch the updated player
	return i.playerRepo.GetByDsbID(ctx, pkz)
}

// ImportClubMembers imports all players from a club (alias for ImportClub)
func (i *Importer) ImportClubMembers(ctx context.Context, zps string) (*ImportResult, error) {
	return i.ImportClub(ctx, zps)
}

// ImportClub imports all players from a club
func (i *Importer) ImportClub(ctx context.Context, zps string) (*ImportResult, error) {
	startTime := time.Now()
	result := &ImportResult{}

	i.logger.Info().Str("zps", zps).Msg("Importing club members from DSB")

	// Fetch club members
	members, err := i.client.GetClubMembers(ctx, zps)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch club members: %w", err)
	}

	result.TotalPlayers = len(members)

	// Process in batches
	var batch []*player.Player
	for _, dsbPlayer := range members {
		p := i.convertToPlayer(dsbPlayer)
		batch = append(batch, p)

		if len(batch) >= i.batchSize {
			if err := i.processBatch(ctx, batch, result); err != nil {
				i.logger.Warn().Err(err).Msg("Batch processing error")
				result.Errors++
			}
			batch = batch[:0]
		}
	}

	// Process remaining
	if len(batch) > 0 {
		if err := i.processBatch(ctx, batch, result); err != nil {
			i.logger.Warn().Err(err).Msg("Batch processing error")
			result.Errors++
		}
	}

	result.ImportTime = time.Since(startTime)

	i.logger.Info().
		Str("zps", zps).
		Int("total", result.TotalPlayers).
		Int("updated", result.UpdatedPlayers).
		Int("errors", result.Errors).
		Dur("duration", result.ImportTime).
		Msg("Club import complete")

	return result, nil
}

// processBatch processes a batch of players
func (i *Importer) processBatch(ctx context.Context, batch []*player.Player, result *ImportResult) error {
	for _, p := range batch {
		if err := i.playerRepo.UpsertByDsbID(ctx, p); err != nil {
			i.logger.Warn().Err(err).Str("dsbId", *p.DsbID).Msg("Failed to upsert player")
			result.Errors++
			continue
		}
		result.UpdatedPlayers++
	}
	return nil
}

// convertToPlayer converts a DSB player to our player model
func (i *Importer) convertToPlayer(dp *DSBPlayer) *player.Player {
	now := time.Now()

	p := &player.Player{
		ID:               uuid.Nil, // Will be set by repository
		FirstName:        dp.FirstName,
		LastName:         dp.LastName,
		DsbID:            &dp.PKZ,
		RatingsUpdatedAt: &now,
	}

	// Set federation
	federation := "GER"
	p.Federation = &federation

	// Set DWZ if present
	if dp.DWZ > 0 {
		p.DWZ = &dp.DWZ
	}
	if dp.DWZIndex > 0 {
		p.DWZIndex = &dp.DWZIndex
	}

	// Set FIDE data if present
	if dp.FideID != "" {
		p.FideID = &dp.FideID
	}
	if dp.FideElo > 0 {
		p.FideStandard = &dp.FideElo
	}
	if dp.FideTitle != "" {
		p.FideTitle = &dp.FideTitle
	}

	// Set birth date if year is present
	if dp.BirthYear > 0 {
		birthDate := fmt.Sprintf("%d-01-01", dp.BirthYear)
		p.BirthDate = &birthDate
	}

	// Set gender
	switch dp.Gender {
	case "M", "m":
		gender := "m"
		p.Gender = &gender
	case "W", "w", "F", "f":
		gender := "f"
		p.Gender = &gender
	}

	return p
}

// SyncPlayer synchronizes a single player from DSB
func (i *Importer) SyncPlayer(ctx context.Context, pkz string) error {
	_, err := i.ImportPlayer(ctx, pkz)
	return err
}

// SearchAndImport searches for players and imports them
func (i *Importer) SearchAndImport(ctx context.Context, query string, limit int) (*ImportResult, error) {
	startTime := time.Now()
	result := &ImportResult{}

	if limit <= 0 {
		limit = 20
	}

	dsbPlayers, err := i.client.SearchPlayers(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search DSB: %w", err)
	}

	result.TotalPlayers = len(dsbPlayers)

	for _, dp := range dsbPlayers {
		p := i.convertToPlayer(dp)
		if err := i.playerRepo.UpsertByDsbID(ctx, p); err != nil {
			i.logger.Warn().Err(err).Str("pkz", dp.PKZ).Msg("Failed to upsert player")
			result.Errors++
			continue
		}
		result.UpdatedPlayers++
	}

	result.Duration = time.Since(startTime)
	result.ImportTime = result.Duration

	return result, nil
}

// Scheduler manages periodic DSB sync jobs
type Scheduler struct {
	importer   *Importer
	logger     zerolog.Logger
	interval   time.Duration
	clubZPS    []string // List of clubs to sync
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
	lastSync   time.Time
	lastResult *ImportResult
}

// SchedulerConfig contains scheduler configuration
type SchedulerConfig struct {
	APIKey    string
	BaseURL   string
	BatchSize int
	Interval  time.Duration
	ClubZPS   []string // List of clubs to sync
}

// NewScheduler creates a new DSB sync scheduler from importer
func NewScheduler(importer *Importer, interval time.Duration, logger zerolog.Logger) *Scheduler {
	if interval <= 0 {
		interval = 7 * 24 * time.Hour // Weekly sync by default
	}

	return &Scheduler{
		importer: importer,
		logger:   logger.With().Str("component", "dsb_scheduler").Logger(),
		interval: interval,
		clubZPS:  []string{},
		stopCh:   make(chan struct{}),
	}
}

// NewSchedulerWithConfig creates a new DSB sync scheduler with full config
func NewSchedulerWithConfig(repo player.Repository, cfg *SchedulerConfig, logger zerolog.Logger) *Scheduler {
	if cfg.Interval <= 0 {
		cfg.Interval = 7 * 24 * time.Hour // Weekly sync by default
	}

	importerCfg := &ImporterConfig{
		APIKey:    cfg.APIKey,
		BaseURL:   cfg.BaseURL,
		BatchSize: cfg.BatchSize,
	}

	return &Scheduler{
		importer: NewImporter(repo, importerCfg, logger),
		logger:   logger.With().Str("component", "dsb_scheduler").Logger(),
		interval: cfg.Interval,
		clubZPS:  cfg.ClubZPS,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic sync schedule
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.runWithContext(ctx)

	s.logger.Info().
		Dur("interval", s.interval).
		Int("clubs", len(s.clubZPS)).
		Msg("DSB scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	s.wg.Wait()

	s.logger.Info().Msg("DSB scheduler stopped")
}

// runWithContext is the main scheduler loop with context
func (s *Scheduler) runWithContext(ctx context.Context) {
	defer s.wg.Done()

	// Run immediately on start
	s.runSync()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runSync()
		}
	}
}

// runSync performs a sync operation for all configured clubs
func (s *Scheduler) runSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	s.logger.Info().Msg("Starting DSB sync")
	startTime := time.Now()

	var totalUpdated, totalErrors int

	for _, zps := range s.clubZPS {
		result, err := s.importer.ImportClub(ctx, zps)
		if err != nil {
			s.logger.Error().Err(err).Str("zps", zps).Msg("Failed to sync club")
			totalErrors++
			continue
		}
		totalUpdated += result.UpdatedPlayers
		totalErrors += result.Errors
	}

	s.mu.Lock()
	s.lastSync = startTime
	s.mu.Unlock()

	s.logger.Info().
		Int("updated", totalUpdated).
		Int("errors", totalErrors).
		Dur("duration", time.Since(startTime)).
		Msg("DSB sync completed")
}

// TriggerSync triggers an immediate sync
func (s *Scheduler) TriggerSync(ctx context.Context) (*ImportResult, error) {
	s.logger.Info().Msg("Manual DSB sync triggered")

	result := &ImportResult{}
	startTime := time.Now()

	for _, zps := range s.clubZPS {
		clubResult, err := s.importer.ImportClub(ctx, zps)
		if err != nil {
			return nil, err
		}
		result.UpdatedPlayers += clubResult.UpdatedPlayers
		result.TotalPlayers += clubResult.TotalPlayers
		result.Errors += clubResult.Errors
	}

	result.Duration = time.Since(startTime)
	s.mu.Lock()
	s.lastSync = time.Now()
	s.lastResult = result
	s.mu.Unlock()

	return result, nil
}

// GetStatus returns the scheduler status
func (s *Scheduler) GetStatus() (time.Time, *ImportResult) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync, s.lastResult
}

// AddClub adds a club to the sync list
func (s *Scheduler) AddClub(zps string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists
	for _, existing := range s.clubZPS {
		if existing == zps {
			return
		}
	}

	s.clubZPS = append(s.clubZPS, zps)
	s.logger.Info().Str("zps", zps).Msg("Added club to DSB sync")
}

// RemoveClub removes a club from the sync list
func (s *Scheduler) RemoveClub(zps string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.clubZPS {
		if existing == zps {
			s.clubZPS = append(s.clubZPS[:i], s.clubZPS[i+1:]...)
			s.logger.Info().Str("zps", zps).Msg("Removed club from DSB sync")
			return
		}
	}
}

// helper to convert string to int
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
