package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestScheduler() *Scheduler {
	logger := zerolog.Nop()
	return NewScheduler(logger)
}

func TestNewScheduler(t *testing.T) {
	s := newTestScheduler()
	assert.NotNil(t, s)
	assert.NotNil(t, s.jobs)
	assert.NotNil(t, s.results)
}

func TestScheduler_RegisterJob(t *testing.T) {
	s := newTestScheduler()

	jobID := s.RegisterJob(
		"test_job",
		"A test job",
		time.Hour,
		func(ctx context.Context) error { return nil },
	)

	assert.NotEmpty(t, jobID)

	jobs := s.GetJobs()
	assert.Len(t, jobs, 1)
	assert.Equal(t, "test_job", jobs[0].Name)
	assert.Equal(t, "A test job", jobs[0].Description)
	assert.Equal(t, time.Hour, jobs[0].Interval)
	assert.True(t, jobs[0].Enabled)
}

func TestScheduler_DisableEnableJob(t *testing.T) {
	s := newTestScheduler()

	jobID := s.RegisterJob(
		"test_job",
		"A test job",
		time.Hour,
		func(ctx context.Context) error { return nil },
	)

	// Disable
	err := s.DisableJob(jobID)
	require.NoError(t, err)

	jobs := s.GetJobs()
	assert.False(t, jobs[0].Enabled)

	// Enable
	err = s.EnableJob(jobID)
	require.NoError(t, err)

	jobs = s.GetJobs()
	assert.True(t, jobs[0].Enabled)
}

func TestScheduler_DisableJob_NotFound(t *testing.T) {
	s := newTestScheduler()

	err := s.DisableJob("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestScheduler_RunJobNow(t *testing.T) {
	s := newTestScheduler()

	var executed int32 = 0

	jobID := s.RegisterJob(
		"test_job",
		"A test job",
		time.Hour, // Long interval so it won't auto-run
		func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		},
	)

	ctx := context.Background()
	err := s.RunJobNow(ctx, jobID)
	require.NoError(t, err)

	// Wait for job to complete
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&executed))
}

func TestScheduler_RunJobNow_NotFound(t *testing.T) {
	s := newTestScheduler()
	ctx := context.Background()

	err := s.RunJobNow(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestScheduler_StartStop(t *testing.T) {
	s := newTestScheduler()

	var executed int32 = 0

	s.RegisterJob(
		"fast_job",
		"A fast job",
		50*time.Millisecond,
		func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		},
	)

	ctx := context.Background()
	s.Start(ctx)

	// Wait for scheduler tick (1 second) plus job execution
	time.Sleep(1500 * time.Millisecond)

	s.Stop()

	count := atomic.LoadInt32(&executed)
	assert.GreaterOrEqual(t, count, int32(1), "Job should have executed at least once")
}

func TestScheduler_JobResults(t *testing.T) {
	s := newTestScheduler()

	jobID := s.RegisterJob(
		"test_job",
		"A test job",
		time.Hour,
		func(ctx context.Context) error { return nil },
	)

	ctx := context.Background()
	err := s.RunJobNow(ctx, jobID)
	require.NoError(t, err)

	// Wait for job to complete
	time.Sleep(100 * time.Millisecond)

	results := s.GetRecentResults(10)
	assert.Len(t, results, 1)
	assert.Equal(t, "test_job", results[0].JobName)
	assert.True(t, results[0].Success)
	assert.Empty(t, results[0].Error)
}

func TestScheduler_JobResultsWithError(t *testing.T) {
	s := newTestScheduler()

	jobID := s.RegisterJob(
		"failing_job",
		"A job that fails",
		time.Hour,
		func(ctx context.Context) error {
			return assert.AnError
		},
	)

	ctx := context.Background()
	err := s.RunJobNow(ctx, jobID)
	require.NoError(t, err)

	// Wait for job to complete
	time.Sleep(100 * time.Millisecond)

	results := s.GetRecentResults(10)
	assert.Len(t, results, 1)
	assert.Equal(t, "failing_job", results[0].JobName)
	assert.False(t, results[0].Success)
	assert.NotEmpty(t, results[0].Error)
}

func TestScheduler_MaxResults(t *testing.T) {
	s := newTestScheduler()
	s.maxResults = 5

	jobID := s.RegisterJob(
		"test_job",
		"A test job",
		time.Hour,
		func(ctx context.Context) error { return nil },
	)

	ctx := context.Background()

	// Run job multiple times
	for i := 0; i < 10; i++ {
		s.RunJobNow(ctx, jobID)
		time.Sleep(50 * time.Millisecond) // Wait for completion
	}

	// Should only keep last 5 results
	results := s.GetRecentResults(100)
	assert.LessOrEqual(t, len(results), 5)
}

func TestScheduler_ConcurrentJobExecution(t *testing.T) {
	s := newTestScheduler()

	var running int32 = 0
	var maxRunning int32 = 0

	// Create a job that takes some time
	s.RegisterJob(
		"slow_job",
		"A slow job",
		500*time.Millisecond,
		func(ctx context.Context) error {
			current := atomic.AddInt32(&running, 1)
			if current > atomic.LoadInt32(&maxRunning) {
				atomic.StoreInt32(&maxRunning, current)
			}
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(&running, -1)
			return nil
		},
	)

	ctx := context.Background()
	s.Start(ctx)

	// Wait for scheduler tick (1 second) plus some executions
	time.Sleep(2500 * time.Millisecond)

	s.Stop()

	// Job should not run concurrently with itself
	// (max running should be 1)
	assert.Equal(t, int32(1), atomic.LoadInt32(&maxRunning))
}

func TestScheduler_ContextCancellation(t *testing.T) {
	s := newTestScheduler()

	var started int32 = 0
	var cancelled int32 = 0

	s.RegisterJob(
		"cancellable_job",
		"A job that respects context",
		100*time.Millisecond,
		func(ctx context.Context) error {
			atomic.AddInt32(&started, 1)
			select {
			case <-ctx.Done():
				atomic.AddInt32(&cancelled, 1)
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Wait for scheduler tick (1 second) and job to start
	time.Sleep(1500 * time.Millisecond)

	// Cancel context
	cancel()

	s.Stop()

	assert.GreaterOrEqual(t, atomic.LoadInt32(&started), int32(1))
}

func TestScheduler_MultipleJobs(t *testing.T) {
	s := newTestScheduler()

	var job1Count int32 = 0
	var job2Count int32 = 0

	s.RegisterJob(
		"job1",
		"First job",
		500*time.Millisecond,
		func(ctx context.Context) error {
			atomic.AddInt32(&job1Count, 1)
			return nil
		},
	)

	s.RegisterJob(
		"job2",
		"Second job",
		500*time.Millisecond,
		func(ctx context.Context) error {
			atomic.AddInt32(&job2Count, 1)
			return nil
		},
	)

	ctx := context.Background()
	s.Start(ctx)

	// Wait for scheduler tick (1 second) plus executions
	time.Sleep(1500 * time.Millisecond)

	s.Stop()

	// Both jobs should have executed
	assert.GreaterOrEqual(t, atomic.LoadInt32(&job1Count), int32(1))
	assert.GreaterOrEqual(t, atomic.LoadInt32(&job2Count), int32(1))
}
