package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// JobFunc represents a function that can be executed as a job
type JobFunc func(ctx context.Context) error

// Job represents a scheduled job
type Job struct {
	ID          string
	Name        string
	Description string
	Execute     JobFunc
	Interval    time.Duration
	LastRun     *time.Time
	NextRun     *time.Time
	Running     bool
	Enabled     bool
	mu          sync.Mutex
}

// JobResult represents the result of a job execution
type JobResult struct {
	JobID     string
	JobName   string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Success   bool
	Error     string
}

// Scheduler manages job scheduling
type Scheduler struct {
	jobs       map[string]*Job
	jobsMu     sync.RWMutex
	results    []JobResult
	resultsMu  sync.RWMutex
	maxResults int
	logger     zerolog.Logger
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewScheduler creates a new job scheduler
func NewScheduler(logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		jobs:       make(map[string]*Job),
		results:    make([]JobResult, 0),
		maxResults: 100, // Keep last 100 results
		logger:     logger.With().Str("component", "scheduler").Logger(),
		stopChan:   make(chan struct{}),
	}
}

// RegisterJob registers a new job with the scheduler
func (s *Scheduler) RegisterJob(name, description string, interval time.Duration, execute JobFunc) string {
	job := &Job{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Execute:     execute,
		Interval:    interval,
		Enabled:     true,
	}

	s.jobsMu.Lock()
	s.jobs[job.ID] = job
	s.jobsMu.Unlock()

	s.logger.Info().
		Str("jobId", job.ID).
		Str("name", name).
		Dur("interval", interval).
		Msg("Job registered")

	return job.ID
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info().Msg("Starting scheduler")

	// Schedule initial next run for all jobs to run immediately
	s.jobsMu.Lock()
	now := time.Now()
	for _, job := range s.jobs {
		// Set NextRun to now so jobs execute on first tick
		nextRun := now
		job.NextRun = &nextRun
	}
	s.jobsMu.Unlock()

	// Start scheduler loop
	s.wg.Add(1)
	go s.run(ctx)
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	s.logger.Info().Msg("Stopping scheduler")
	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info().Msg("Scheduler stopped")
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndRunJobs(ctx)
		}
	}
}

func (s *Scheduler) checkAndRunJobs(ctx context.Context) {
	s.jobsMu.RLock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	s.jobsMu.RUnlock()

	now := time.Now()

	for _, job := range jobs {
		job.mu.Lock()
		shouldRun := job.Enabled && !job.Running && job.NextRun != nil && now.After(*job.NextRun)
		if shouldRun {
			job.Running = true
		}
		job.mu.Unlock()

		if shouldRun {
			s.wg.Add(1)
			go s.executeJob(ctx, job)
		}
	}
}

func (s *Scheduler) executeJob(ctx context.Context, job *Job) {
	defer s.wg.Done()

	startTime := time.Now()
	s.logger.Info().Str("jobId", job.ID).Str("name", job.Name).Msg("Executing job")

	var errStr string
	err := job.Execute(ctx)
	endTime := time.Now()

	if err != nil {
		errStr = err.Error()
		s.logger.Error().
			Err(err).
			Str("jobId", job.ID).
			Str("name", job.Name).
			Dur("duration", endTime.Sub(startTime)).
			Msg("Job failed")
	} else {
		s.logger.Info().
			Str("jobId", job.ID).
			Str("name", job.Name).
			Dur("duration", endTime.Sub(startTime)).
			Msg("Job completed successfully")
	}

	// Record result
	result := JobResult{
		JobID:     job.ID,
		JobName:   job.Name,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Success:   err == nil,
		Error:     errStr,
	}

	s.resultsMu.Lock()
	s.results = append(s.results, result)
	if len(s.results) > s.maxResults {
		s.results = s.results[1:]
	}
	s.resultsMu.Unlock()

	// Update job state
	job.mu.Lock()
	job.LastRun = &startTime
	nextRun := time.Now().Add(job.Interval)
	job.NextRun = &nextRun
	job.Running = false
	job.mu.Unlock()
}

// RunJobNow runs a job immediately
func (s *Scheduler) RunJobNow(ctx context.Context, jobID string) error {
	s.jobsMu.RLock()
	job, ok := s.jobs[jobID]
	s.jobsMu.RUnlock()

	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.mu.Lock()
	if job.Running {
		job.mu.Unlock()
		return fmt.Errorf("job is already running")
	}
	job.Running = true
	job.mu.Unlock()

	s.wg.Add(1)
	go s.executeJob(ctx, job)

	return nil
}

// EnableJob enables a job
func (s *Scheduler) EnableJob(jobID string) error {
	s.jobsMu.RLock()
	job, ok := s.jobs[jobID]
	s.jobsMu.RUnlock()

	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.mu.Lock()
	job.Enabled = true
	job.mu.Unlock()

	return nil
}

// DisableJob disables a job
func (s *Scheduler) DisableJob(jobID string) error {
	s.jobsMu.RLock()
	job, ok := s.jobs[jobID]
	s.jobsMu.RUnlock()

	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.mu.Lock()
	job.Enabled = false
	job.mu.Unlock()

	return nil
}

// GetJobs returns all registered jobs
func (s *Scheduler) GetJobs() []JobInfo {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	infos := make([]JobInfo, 0, len(s.jobs))
	for _, job := range s.jobs {
		job.mu.Lock()
		info := JobInfo{
			ID:          job.ID,
			Name:        job.Name,
			Description: job.Description,
			Interval:    job.Interval,
			LastRun:     job.LastRun,
			NextRun:     job.NextRun,
			Running:     job.Running,
			Enabled:     job.Enabled,
		}
		job.mu.Unlock()
		infos = append(infos, info)
	}

	return infos
}

// JobInfo represents job information for display
type JobInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Interval    time.Duration `json:"interval"`
	LastRun     *time.Time    `json:"lastRun"`
	NextRun     *time.Time    `json:"nextRun"`
	Running     bool          `json:"running"`
	Enabled     bool          `json:"enabled"`
}

// GetRecentResults returns recent job execution results
func (s *Scheduler) GetRecentResults(limit int) []JobResult {
	s.resultsMu.RLock()
	defer s.resultsMu.RUnlock()

	if limit <= 0 || limit > len(s.results) {
		limit = len(s.results)
	}

	// Return most recent first
	results := make([]JobResult, limit)
	for i := 0; i < limit; i++ {
		results[i] = s.results[len(s.results)-1-i]
	}

	return results
}
