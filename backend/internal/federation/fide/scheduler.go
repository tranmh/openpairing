package fide

import (
	"context"
	"sync"
	"time"

	"github.com/tranmh/openpairing/internal/player"
	"github.com/rs/zerolog"
)

// Scheduler manages periodic FIDE sync jobs
type Scheduler struct {
	importer    *Importer
	logger      zerolog.Logger
	interval    time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
	lastSync    time.Time
	lastResult  *ImportResult
	mu          sync.RWMutex
	running     bool
}

// SchedulerConfig contains scheduler configuration
type SchedulerConfig struct {
	CacheDir   string
	BatchSize  int
	Workers    int
	FilterDACH bool
	MinRating  int
	Interval   time.Duration // Default: 24 hours
}

// NewScheduler creates a new FIDE sync scheduler from importer
func NewScheduler(importer *Importer, interval time.Duration, logger zerolog.Logger) *Scheduler {
	if interval <= 0 {
		interval = 24 * time.Hour // Daily sync by default
	}

	return &Scheduler{
		importer: importer,
		logger:   logger.With().Str("component", "fide_scheduler").Logger(),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// NewSchedulerWithConfig creates a new FIDE sync scheduler with full config
func NewSchedulerWithConfig(repo player.Repository, cfg *SchedulerConfig, logger zerolog.Logger) *Scheduler {
	if cfg.Interval <= 0 {
		cfg.Interval = 24 * time.Hour // Daily sync by default
	}

	importerCfg := &ImporterConfig{
		CacheDir:   cfg.CacheDir,
		BatchSize:  cfg.BatchSize,
		Workers:    cfg.Workers,
		FilterDACH: cfg.FilterDACH,
		MinRating:  cfg.MinRating,
	}

	return &Scheduler{
		importer: NewImporter(repo, importerCfg, logger),
		logger:   logger.With().Str("component", "fide_scheduler").Logger(),
		interval: cfg.Interval,
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
		Msg("FIDE scheduler started")
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

	s.logger.Info().Msg("FIDE scheduler stopped")
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

// runSync performs a single sync operation
func (s *Scheduler) runSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	s.logger.Info().Msg("Starting FIDE sync")
	startTime := time.Now()

	result, err := s.importer.ImportLatest(ctx)

	s.mu.Lock()
	s.lastSync = startTime
	s.lastResult = result
	s.mu.Unlock()

	if err != nil {
		s.logger.Error().Err(err).Msg("FIDE sync failed")
		return
	}

	s.logger.Info().
		Int("total", result.TotalPlayers).
		Int("new", result.NewPlayers).
		Int("updated", result.UpdatedPlayers).
		Dur("duration", result.ImportTime).
		Msg("FIDE sync completed")
}

// TriggerSync triggers an immediate sync
func (s *Scheduler) TriggerSync(ctx context.Context) (*ImportResult, error) {
	s.logger.Info().Msg("Manual FIDE sync triggered")
	return s.importer.ImportLatest(ctx)
}

// GetStatus returns the current scheduler status
func (s *Scheduler) GetStatus() *SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &SchedulerStatus{
		Running:  s.running,
		Interval: s.interval,
		LastSync: s.lastSync,
		NextSync: s.lastSync.Add(s.interval),
	}

	if s.lastResult != nil {
		status.LastResult = &SyncResultSummary{
			TotalPlayers:   s.lastResult.TotalPlayers,
			NewPlayers:     s.lastResult.NewPlayers,
			UpdatedPlayers: s.lastResult.UpdatedPlayers,
			Duration:       s.lastResult.ImportTime,
		}
	}

	return status
}

// SchedulerStatus represents the current state of the scheduler
type SchedulerStatus struct {
	Running    bool
	Interval   time.Duration
	LastSync   time.Time
	NextSync   time.Time
	LastResult *SyncResultSummary
}

// SyncResultSummary is a summary of the last sync result
type SyncResultSummary struct {
	TotalPlayers   int
	NewPlayers     int
	UpdatedPlayers int
	Duration       time.Duration
}

// IsHealthy returns true if the scheduler is healthy
func (s *Scheduler) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return false
	}

	// Consider unhealthy if last sync was more than 2 intervals ago
	if time.Since(s.lastSync) > 2*s.interval {
		return false
	}

	return true
}

// GetLastSyncStatus returns the last sync time and result
func (s *Scheduler) GetLastSyncStatus() (time.Time, *ImportResult) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync, s.lastResult
}
