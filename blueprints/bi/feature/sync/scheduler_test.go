package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name        string
		schedule    string
		expected    time.Duration
		expectError bool
	}{
		// Duration formats
		{
			name:     "1 hour duration",
			schedule: "1h",
			expected: time.Hour,
		},
		{
			name:     "30 minutes duration",
			schedule: "30m",
			expected: 30 * time.Minute,
		},
		{
			name:     "24 hours duration",
			schedule: "24h",
			expected: 24 * time.Hour,
		},
		{
			name:     "2 hours 30 minutes",
			schedule: "2h30m",
			expected: 2*time.Hour + 30*time.Minute,
		},

		// Named schedules
		{
			name:     "hourly named",
			schedule: "hourly",
			expected: time.Hour,
		},
		{
			name:     "daily named",
			schedule: "daily",
			expected: 24 * time.Hour,
		},
		{
			name:     "weekly named",
			schedule: "weekly",
			expected: 7 * 24 * time.Hour,
		},

		// Cron-like
		{
			name:     "hourly cron",
			schedule: "0 * * * *",
			expected: time.Hour,
		},
		{
			name:     "daily cron",
			schedule: "0 0 * * *",
			expected: 24 * time.Hour,
		},

		// Errors
		{
			name:        "too short interval",
			schedule:    "30s",
			expectError: true,
		},
		{
			name:        "invalid format",
			schedule:    "invalid",
			expectError: true,
		},
		{
			name:        "unsupported cron",
			schedule:    "*/5 * * * *",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := parseSchedule(tt.schedule)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, duration)
			}
		})
	}
}

func TestFilterSchemas(t *testing.T) {
	schemas := []string{"public", "analytics", "internal", "pg_catalog", "information_schema"}

	tests := []struct {
		name     string
		patterns []string
		include  bool
		expected []string
	}{
		{
			name:     "include specific schemas",
			patterns: []string{"public", "analytics"},
			include:  true,
			expected: []string{"public", "analytics"},
		},
		{
			name:     "exclude system schemas",
			patterns: []string{"pg_catalog", "information_schema"},
			include:  false,
			expected: []string{"public", "analytics", "internal"},
		},
		{
			name:     "include non-existent",
			patterns: []string{"nonexistent"},
			include:  true,
			expected: nil,
		},
		{
			name:     "exclude all",
			patterns: []string{"public", "analytics", "internal", "pg_catalog", "information_schema"},
			include:  false,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterSchemas(schemas, tt.patterns, tt.include)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSyncResult(t *testing.T) {
	now := time.Now()
	result := SyncResult{
		DataSourceID:   "ds-123",
		Status:         "success",
		StartedAt:      now,
		CompletedAt:    now.Add(5 * time.Second),
		DurationMs:     5000,
		SchemasSynced:  3,
		TablesSynced:   45,
		ColumnsSynced:  380,
		Errors:         nil,
	}

	assert.Equal(t, "ds-123", result.DataSourceID)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, int64(5000), result.DurationMs)
	assert.Equal(t, 3, result.SchemasSynced)
	assert.Equal(t, 45, result.TablesSynced)
	assert.Equal(t, 380, result.ColumnsSynced)
	assert.Empty(t, result.Errors)
}

func TestJob(t *testing.T) {
	now := time.Now()
	job := Job{
		DataSourceID: "ds-123",
		Schedule:     "1h",
		LastRun:      now.Add(-1 * time.Hour),
		NextRun:      now,
		Running:      false,
	}

	assert.Equal(t, "ds-123", job.DataSourceID)
	assert.Equal(t, "1h", job.Schedule)
	assert.False(t, job.Running)
	assert.True(t, time.Now().After(job.NextRun) || time.Now().Equal(job.NextRun))
}

func TestNewScheduler(t *testing.T) {
	// This test verifies the scheduler can be created
	// Full integration tests would require a mock store
	scheduler := NewScheduler(nil, nil)
	require.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.jobs)
	assert.NotNil(t, scheduler.stopCh)
}

func TestSchedulerScheduleInvalidCron(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	err := scheduler.Schedule("ds-123", "invalid-cron")
	assert.Error(t, err)
}

func TestSchedulerScheduleValid(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	err := scheduler.Schedule("ds-123", "1h")
	require.NoError(t, err)

	jobs := scheduler.GetJobs()
	assert.Len(t, jobs, 1)
	assert.Contains(t, jobs, "ds-123")
	assert.Equal(t, "1h", jobs["ds-123"].Schedule)
}

func TestSchedulerUnschedule(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	// Schedule a job
	err := scheduler.Schedule("ds-123", "1h")
	require.NoError(t, err)
	assert.Len(t, scheduler.GetJobs(), 1)

	// Unschedule it
	scheduler.Unschedule("ds-123")
	assert.Len(t, scheduler.GetJobs(), 0)
}

func TestSchedulerUnscheduleNonExistent(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	// Should not panic
	scheduler.Unschedule("nonexistent")
	assert.Len(t, scheduler.GetJobs(), 0)
}

func TestSchedulerReschedule(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	// Schedule with 1h
	err := scheduler.Schedule("ds-123", "1h")
	require.NoError(t, err)

	// Reschedule with 2h
	err = scheduler.Schedule("ds-123", "2h")
	require.NoError(t, err)

	jobs := scheduler.GetJobs()
	assert.Len(t, jobs, 1)
	assert.Equal(t, "2h", jobs["ds-123"].Schedule)
}

func TestSchedulerStop(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	// Schedule a job
	scheduler.Schedule("ds-123", "1h")

	// Stop should not panic
	scheduler.Stop()

	// Jobs should be cleared
	assert.Len(t, scheduler.GetJobs(), 0)
}
