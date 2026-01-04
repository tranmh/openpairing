package oesb

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/player"
	"github.com/rs/zerolog"
)

// Importer handles importing ÖSB data into the player database
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

// NewImporter creates a new ÖSB importer
func NewImporter(repo player.Repository, cfg *ImporterConfig, logger zerolog.Logger) *Importer {
	l := logger.With().Str("component", "oesb_importer").Logger()

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
}

// ImportPlayer imports a single player by ÖSB ID
func (i *Importer) ImportPlayer(ctx context.Context, oesbID string) (*player.Player, error) {
	oesbPlayer, err := i.client.GetPlayer(ctx, oesbID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	p := i.convertToPlayer(oesbPlayer)
	if err := i.playerRepo.UpsertByOesbID(ctx, p); err != nil {
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

	for _, oesbPlayer := range players {
		p := i.convertToPlayer(oesbPlayer)
		if err := i.playerRepo.UpsertByOesbID(ctx, p); err != nil {
			i.logger.Warn().Err(err).Str("id", oesbPlayer.ID).Msg("Failed to upsert player")
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
	for _, oesbPlayer := range players {
		p := i.convertToPlayer(oesbPlayer)
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

// ImportRatingsList imports the full Austrian rating list
func (i *Importer) ImportRatingsList(ctx context.Context) (*ImportResult, error) {
	result := &ImportResult{}
	startTime := time.Now()

	i.logger.Info().Msg("Downloading ÖSB ratings list")

	oesbPlayers, err := i.client.GetRatingsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ratings list: %w", err)
	}

	result.TotalPlayers = len(oesbPlayers)
	i.logger.Info().Int("players", result.TotalPlayers).Msg("Downloaded ÖSB ratings")

	// Create player channel for concurrent processing
	playerChan := make(chan *OESBPlayer, i.batchSize)
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
	for _, p := range oesbPlayers {
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
		Msg("ÖSB import complete")

	return result, nil
}

// importWorker processes players from the channel
func (i *Importer) importWorker(ctx context.Context, players <-chan *OESBPlayer, result *ImportResult) {
	var mu sync.Mutex
	var batch []*player.Player

	for oesbPlayer := range players {
		p := i.convertToPlayer(oesbPlayer)
		batch = append(batch, p)

		if len(batch) >= i.batchSize {
			if err := i.processBatch(ctx, batch, result); err != nil {
				i.logger.Warn().Err(err).Msg("Batch processing error")
				mu.Lock()
				result.Errors++
				mu.Unlock()
			}
			batch = batch[:0]
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		if err := i.processBatch(ctx, batch, result); err != nil {
			i.logger.Warn().Err(err).Msg("Final batch processing error")
			mu.Lock()
			result.Errors++
			mu.Unlock()
		}
	}
}

// processBatch processes a batch of players
func (i *Importer) processBatch(ctx context.Context, batch []*player.Player, result *ImportResult) error {
	if err := i.playerRepo.BulkUpsertByOesbID(ctx, batch); err != nil {
		return fmt.Errorf("failed to upsert batch: %w", err)
	}

	// Thread-safe update
	var mu sync.Mutex
	mu.Lock()
	result.UpdatedPlayers += len(batch)
	mu.Unlock()

	return nil
}

// convertToPlayer converts an ÖSB player to our player model
func (i *Importer) convertToPlayer(op *OESBPlayer) *player.Player {
	now := time.Now()

	p := &player.Player{
		ID:               uuid.Nil, // Will be set by repository
		FirstName:        op.FirstName,
		LastName:         op.LastName,
		RatingsUpdatedAt: &now,
	}

	// Set ÖSB ID
	if op.ID != "" {
		p.OesbID = &op.ID
	} else if op.PKZ != "" {
		p.OesbID = &op.PKZ
	}

	// Set federation
	federation := "AUT"
	p.Federation = &federation

	// Set ÖSB ELO
	if op.Rating > 0 {
		p.OesbElo = &op.Rating
	}

	// Set FIDE data if present
	if op.FideID != "" {
		p.FideID = &op.FideID
	}
	if op.FideRating > 0 {
		p.FideStandard = &op.FideRating
	}
	if op.FideTitle != "" {
		p.FideTitle = &op.FideTitle
	}

	// Set title
	if op.Title != "" {
		p.NationalTitle = &op.Title
	}

	// Set gender
	if gender := normalizeGender(op.Gender); gender != "" {
		p.Gender = &gender
	}

	// Set birth year as date string
	if op.BirthYear > 0 {
		birthDate := fmt.Sprintf("%d-01-01", op.BirthYear)
		p.BirthDate = &birthDate
	}

	// Set club info
	if op.ClubID != "" {
		clubID, err := uuid.Parse(op.ClubID)
		if err == nil {
			p.ClubID = &clubID
		}
	}
	if op.ClubName != "" {
		p.ClubName = &op.ClubName
	}

	return p
}

// normalizeGender normalizes gender strings to M/F
func normalizeGender(gender string) string {
	switch strings.ToUpper(strings.TrimSpace(gender)) {
	case "M", "MALE", "MÄNNLICH", "H":
		return "M"
	case "F", "FEMALE", "WEIBLICH", "W", "D":
		return "F"
	default:
		return ""
	}
}

// Scheduler handles periodic syncing of ÖSB data
type Scheduler struct {
	importer *Importer
	logger   zerolog.Logger
	interval time.Duration

	stopCh     chan struct{}
	lastSync   time.Time
	lastResult *ImportResult
	mu         sync.RWMutex
}

// NewScheduler creates a new ÖSB sync scheduler
func NewScheduler(importer *Importer, interval time.Duration, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		importer: importer,
		logger:   logger.With().Str("component", "oesb_scheduler").Logger(),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic sync
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info().Dur("interval", s.interval).Msg("Starting ÖSB scheduler")

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
	s.logger.Info().Msg("Starting ÖSB sync")

	result, err := s.importer.ImportRatingsList(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("ÖSB sync failed")
		return
	}

	s.mu.Lock()
	s.lastSync = time.Now()
	s.lastResult = result
	s.mu.Unlock()

	s.logger.Info().
		Int("updated", result.UpdatedPlayers).
		Dur("duration", result.Duration).
		Msg("ÖSB sync completed")
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

