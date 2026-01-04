package fide

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/player"
	"github.com/rs/zerolog"
)

// Importer handles importing FIDE data into the player database
type Importer struct {
	client     *Client
	parser     *Parser
	playerRepo player.Repository
	logger     zerolog.Logger

	// Configuration
	batchSize    int
	workers      int
	filterDACH   bool
	minRating    int
}

// ImporterConfig contains configuration for the importer
type ImporterConfig struct {
	CacheDir   string
	BatchSize  int
	Workers    int
	FilterDACH bool
	MinRating  int
}

// NewImporter creates a new FIDE importer
func NewImporter(repo player.Repository, cfg *ImporterConfig, logger zerolog.Logger) *Importer {
	l := logger.With().Str("component", "fide_importer").Logger()

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = "/tmp/fide_cache"
	}

	return &Importer{
		client:     NewClient(l, cfg.CacheDir),
		parser:     NewParser(l),
		playerRepo: repo,
		logger:     l,
		batchSize:  cfg.BatchSize,
		workers:    cfg.Workers,
		filterDACH: cfg.FilterDACH,
		minRating:  cfg.MinRating,
	}
}

// ImportResult contains statistics from the import
type ImportResult struct {
	DownloadTime   time.Duration
	ParseTime      time.Duration
	ImportTime     time.Duration
	TotalPlayers   int
	NewPlayers     int
	UpdatedPlayers int
	SkippedPlayers int
	Errors         int
}

// ImportLatest downloads and imports the latest FIDE rating list
func (i *Importer) ImportLatest(ctx context.Context) (*ImportResult, error) {
	result := &ImportResult{}
	startTime := time.Now()

	// Download latest ratings
	i.logger.Info().Msg("Downloading FIDE rating list")
	downloadStart := time.Now()

	download, err := i.client.DownloadLatestRatings("all")
	if err != nil {
		return nil, fmt.Errorf("failed to download FIDE ratings: %w", err)
	}
	result.DownloadTime = time.Since(downloadStart)

	// Extract if needed
	xmlPath := download.FilePath
	if download.FilePath[len(download.FilePath)-4:] == ".zip" {
		xmlPath, err = i.client.ExtractZip(download.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to extract zip: %w", err)
		}
	}

	// Parse and import
	i.logger.Info().Str("file", xmlPath).Msg("Parsing FIDE XML")
	parseStart := time.Now()

	// Create player channel
	playerChan := make(chan *FIDEPlayer, i.batchSize)
	errorChan := make(chan error, 100)
	var wg sync.WaitGroup

	// Start worker goroutines
	for w := 0; w < i.workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			i.importWorker(ctx, workerID, playerChan, result, errorChan)
		}(w)
	}

	// Create handler
	handler := func(fidePlayer *FIDEPlayer) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case playerChan <- fidePlayer:
			return nil
		}
	}

	// Apply filters
	if i.filterDACH {
		handler = FilterDACH(handler)
	}
	if i.minRating > 0 {
		handler = FilterMinRating(i.minRating, handler)
	}

	// Parse file
	parseResult, err := i.parser.ParseXMLFromFile(xmlPath, handler)
	if err != nil {
		close(playerChan)
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Close channel and wait for workers
	close(playerChan)
	wg.Wait()
	close(errorChan)

	result.ParseTime = time.Since(parseStart)
	result.TotalPlayers = parseResult.TotalPlayers
	result.ImportTime = time.Since(startTime)

	// Check for errors
	for err := range errorChan {
		i.logger.Warn().Err(err).Msg("Import error")
		result.Errors++
	}

	i.logger.Info().
		Int("total", result.TotalPlayers).
		Int("new", result.NewPlayers).
		Int("updated", result.UpdatedPlayers).
		Int("skipped", result.SkippedPlayers).
		Int("errors", result.Errors).
		Dur("duration", result.ImportTime).
		Msg("Import complete")

	return result, nil
}

// importWorker processes players from the channel
func (i *Importer) importWorker(ctx context.Context, workerID int, players <-chan *FIDEPlayer, result *ImportResult, errors chan<- error) {
	var batch []*player.Player
	var mu sync.Mutex

	for fidePlayer := range players {
		p := i.convertToPlayer(fidePlayer)
		batch = append(batch, p)

		if len(batch) >= i.batchSize {
			if err := i.processBatch(ctx, batch, result, &mu); err != nil {
				select {
				case errors <- err:
				default:
				}
			}
			batch = batch[:0]
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		if err := i.processBatch(ctx, batch, result, &mu); err != nil {
			select {
			case errors <- err:
			default:
			}
		}
	}
}

// processBatch processes a batch of players
func (i *Importer) processBatch(ctx context.Context, batch []*player.Player, result *ImportResult, mu *sync.Mutex) error {
	if err := i.playerRepo.BulkUpsertByFideID(ctx, batch); err != nil {
		return fmt.Errorf("failed to upsert batch: %w", err)
	}

	mu.Lock()
	result.UpdatedPlayers += len(batch)
	mu.Unlock()

	return nil
}

// convertToPlayer converts a FIDE player to our player model
func (i *Importer) convertToPlayer(fp *FIDEPlayer) *player.Player {
	name := ParseName(fp.Name)
	now := time.Now()

	p := &player.Player{
		ID:               uuid.Nil, // Will be set by repository
		FirstName:        name.FirstName,
		LastName:         name.LastName,
		FideID:           &fp.FideID,
		Federation:       &fp.Country,
		RatingsUpdatedAt: &now,
	}

	// Set ratings if present
	if fp.Rating > 0 {
		p.FideStandard = &fp.Rating
	}
	if fp.RapidRating > 0 {
		p.FideRapid = &fp.RapidRating
	}
	if fp.BlitzRating > 0 {
		p.FideBlitz = &fp.BlitzRating
	}

	// Set title
	if title := fp.GetTitle(); title != "" {
		p.FideTitle = &title
	}

	// Set gender
	if gender := fp.GetGender(); gender != "" {
		p.Gender = &gender
	}

	// Set birth date
	p.BirthDate = fp.GetBirthDate()

	return p
}

// SyncLog records a sync operation
type SyncLog struct {
	ID            uuid.UUID
	Federation    string
	SyncType      string
	StartedAt     time.Time
	CompletedAt   *time.Time
	PlayersUpdated int
	Status        string
	ErrorMessage  *string
}

// StartSync begins a sync operation and returns a log entry
func (i *Importer) StartSync(syncType string) *SyncLog {
	return &SyncLog{
		ID:         uuid.New(),
		Federation: "fide",
		SyncType:   syncType,
		StartedAt:  time.Now(),
		Status:     "running",
	}
}

// CompleteSync marks a sync as complete
func (i *Importer) CompleteSync(log *SyncLog, result *ImportResult, err error) {
	now := time.Now()
	log.CompletedAt = &now
	log.PlayersUpdated = result.NewPlayers + result.UpdatedPlayers

	if err != nil {
		log.Status = "failed"
		errMsg := err.Error()
		log.ErrorMessage = &errMsg
	} else {
		log.Status = "completed"
	}
}
