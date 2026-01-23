// Package sync provides metadata synchronization scheduling for data sources.
package sync

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/bi/drivers"
	"github.com/go-mizu/blueprints/bi/store"
)

// Scheduler manages scheduled metadata synchronization for data sources.
type Scheduler struct {
	store    store.Store
	jobs     map[string]*Job
	mu       sync.RWMutex
	stopCh   chan struct{}
	wg       sync.WaitGroup
	logger   *log.Logger
}

// Job represents a scheduled sync job for a data source.
type Job struct {
	DataSourceID string
	Schedule     string // cron expression or interval
	LastRun      time.Time
	NextRun      time.Time
	Running      bool
	stopCh       chan struct{}
}

// SyncResult holds the result of a sync operation.
type SyncResult struct {
	DataSourceID   string    `json:"datasource_id"`
	Status         string    `json:"status"` // success, partial, failed
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
	DurationMs     int64     `json:"duration_ms"`
	SchemasSynced  int       `json:"schemas_synced"`
	TablesSynced   int       `json:"tables_synced"`
	ColumnsSynced  int       `json:"columns_synced"`
	Errors         []string  `json:"errors,omitempty"`
}

// NewScheduler creates a new sync scheduler.
func NewScheduler(store store.Store, logger *log.Logger) *Scheduler {
	if logger == nil {
		logger = log.Default()
	}
	return &Scheduler{
		store:  store,
		jobs:   make(map[string]*Job),
		stopCh: make(chan struct{}),
		logger: logger,
	}
}

// Start starts the scheduler and loads existing schedules from the database.
func (s *Scheduler) Start(ctx context.Context) error {
	// Load all data sources with auto_sync enabled
	dataSources, err := s.store.DataSources().List(ctx)
	if err != nil {
		return fmt.Errorf("load data sources: %w", err)
	}

	for _, ds := range dataSources {
		if ds.AutoSync && ds.SyncSchedule != "" {
			if err := s.Schedule(ds.ID, ds.SyncSchedule); err != nil {
				s.logger.Printf("Failed to schedule sync for %s: %v", ds.ID, err)
			}
		}
	}

	// Start the scheduler loop
	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

// Stop stops the scheduler and all running jobs.
func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()

	s.mu.Lock()
	for _, job := range s.jobs {
		if job.stopCh != nil {
			close(job.stopCh)
		}
	}
	s.jobs = make(map[string]*Job)
	s.mu.Unlock()
}

// Schedule schedules a sync job for a data source.
func (s *Scheduler) Schedule(dataSourceID, schedule string) error {
	interval, err := parseSchedule(schedule)
	if err != nil {
		return fmt.Errorf("invalid schedule %q: %w", schedule, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel existing job if any
	if existing, ok := s.jobs[dataSourceID]; ok {
		if existing.stopCh != nil {
			close(existing.stopCh)
		}
	}

	job := &Job{
		DataSourceID: dataSourceID,
		Schedule:     schedule,
		NextRun:      time.Now().Add(interval),
		stopCh:       make(chan struct{}),
	}
	s.jobs[dataSourceID] = job

	s.logger.Printf("Scheduled sync for data source %s: %s (next run: %v)", dataSourceID, schedule, job.NextRun)
	return nil
}

// Unschedule removes a scheduled sync job.
func (s *Scheduler) Unschedule(dataSourceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, ok := s.jobs[dataSourceID]; ok {
		if job.stopCh != nil {
			close(job.stopCh)
		}
		delete(s.jobs, dataSourceID)
		s.logger.Printf("Unscheduled sync for data source %s", dataSourceID)
	}
}

// RunNow triggers an immediate sync for a data source.
func (s *Scheduler) RunNow(ctx context.Context, dataSourceID string) (*SyncResult, error) {
	return s.syncDataSource(ctx, dataSourceID)
}

// GetJobs returns all scheduled jobs.
func (s *Scheduler) GetJobs() map[string]*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*Job, len(s.jobs))
	for k, v := range s.jobs {
		result[k] = &Job{
			DataSourceID: v.DataSourceID,
			Schedule:     v.Schedule,
			LastRun:      v.LastRun,
			NextRun:      v.NextRun,
			Running:      v.Running,
		}
	}
	return result
}

// run is the main scheduler loop.
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndRunJobs(ctx)
		}
	}
}

// checkAndRunJobs checks all jobs and runs those that are due.
func (s *Scheduler) checkAndRunJobs(ctx context.Context) {
	now := time.Now()

	s.mu.RLock()
	var dueJobs []*Job
	for _, job := range s.jobs {
		if !job.Running && now.After(job.NextRun) {
			dueJobs = append(dueJobs, job)
		}
	}
	s.mu.RUnlock()

	for _, job := range dueJobs {
		s.wg.Add(1)
		go func(j *Job) {
			defer s.wg.Done()
			s.runJob(ctx, j)
		}(job)
	}
}

// runJob executes a sync job.
func (s *Scheduler) runJob(ctx context.Context, job *Job) {
	s.mu.Lock()
	job.Running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		job.Running = false
		job.LastRun = time.Now()

		// Calculate next run
		interval, _ := parseSchedule(job.Schedule)
		job.NextRun = time.Now().Add(interval)
		s.mu.Unlock()
	}()

	result, err := s.syncDataSource(ctx, job.DataSourceID)
	if err != nil {
		s.logger.Printf("Sync failed for %s: %v", job.DataSourceID, err)
		return
	}

	s.logger.Printf("Sync completed for %s: %s (tables=%d, columns=%d, duration=%dms)",
		job.DataSourceID, result.Status, result.TablesSynced, result.ColumnsSynced, result.DurationMs)
}

// syncDataSource performs the actual metadata synchronization.
func (s *Scheduler) syncDataSource(ctx context.Context, dataSourceID string) (*SyncResult, error) {
	result := &SyncResult{
		DataSourceID: dataSourceID,
		StartedAt:    time.Now(),
	}

	// Get data source
	ds, err := s.store.DataSources().GetByID(ctx, dataSourceID)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}
	if ds == nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, "data source not found")
		return result, fmt.Errorf("data source not found: %s", dataSourceID)
	}

	// Create driver config
	config := drivers.Config{
		Engine:        ds.Engine,
		Host:          ds.Host,
		Port:          ds.Port,
		Database:      ds.Database,
		Username:      ds.Username,
		Password:      ds.Password,
		SSL:           ds.SSL,
		SSLMode:       ds.SSLMode,
		SSLRootCert:   ds.SSLRootCert,
		SSLClientCert: ds.SSLClientCert,
		SSLClientKey:  ds.SSLClientKey,
		Options:       ds.Options,
	}

	// Open connection
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
		s.updateSyncStatus(ctx, ds, "failed", err.Error())
		return result, err
	}
	defer driver.Close()

	// Get schemas
	schemas, err := driver.ListSchemas(ctx)
	if err != nil {
		schemas = []string{""} // Use empty schema for schema-less DBs
	}

	// Apply schema filter
	if ds.SchemaFilterType == "inclusion" && len(ds.SchemaFilterPatterns) > 0 {
		schemas = filterSchemas(schemas, ds.SchemaFilterPatterns, true)
	} else if ds.SchemaFilterType == "exclusion" && len(ds.SchemaFilterPatterns) > 0 {
		schemas = filterSchemas(schemas, ds.SchemaFilterPatterns, false)
	}

	result.SchemasSynced = len(schemas)

	// Sync tables for each schema
	for _, schema := range schemas {
		tables, err := driver.ListTables(ctx, schema)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("schema %s: %v", schema, err))
			continue
		}

		for _, t := range tables {
			table := &store.Table{
				DataSourceID: ds.ID,
				Schema:       t.Schema,
				Name:         t.Name,
				DisplayName:  t.Name,
				Description:  t.Description,
				RowCount:     t.RowCount,
			}

			if err := s.store.Tables().Create(ctx, table); err != nil {
				// Table might already exist, try to find it
				continue
			}
			result.TablesSynced++

			// Sync columns
			columns, err := driver.ListColumns(ctx, schema, t.Name)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("table %s.%s: %v", schema, t.Name, err))
				continue
			}

			for _, col := range columns {
				storeCol := &store.Column{
					TableID:     table.ID,
					Name:        col.Name,
					DisplayName: col.Name,
					Type:        col.Type,
					MappedType:  col.MappedType,
					Description: col.Description,
					Position:    col.Position,
					Nullable:    col.Nullable,
					PrimaryKey:  col.PrimaryKey,
					ForeignKey:  col.ForeignKey,
					Visibility:  "everywhere",
				}

				if err := s.store.Tables().CreateColumn(ctx, storeCol); err != nil {
					continue
				}
				result.ColumnsSynced++
			}
		}
	}

	result.CompletedAt = time.Now()
	result.DurationMs = result.CompletedAt.Sub(result.StartedAt).Milliseconds()

	if len(result.Errors) == 0 {
		result.Status = "success"
		s.updateSyncStatus(ctx, ds, "success", "")
	} else {
		result.Status = "partial"
		s.updateSyncStatus(ctx, ds, "partial", result.Errors[0])
	}

	return result, nil
}

// updateSyncStatus updates the sync status in the database.
func (s *Scheduler) updateSyncStatus(ctx context.Context, ds *store.DataSource, status, errMsg string) {
	now := time.Now()
	ds.LastSyncAt = &now
	ds.LastSyncStatus = status
	ds.LastSyncError = errMsg
	s.store.DataSources().Update(ctx, ds)
}

// parseSchedule parses a schedule string into a duration.
// Supports:
// - Intervals: "1h", "30m", "24h"
// - Named schedules: "hourly", "daily"
// - Cron-like (simplified): "0 * * * *" (hourly), "0 0 * * *" (daily)
func parseSchedule(schedule string) (time.Duration, error) {
	// Try parsing as duration first
	if d, err := time.ParseDuration(schedule); err == nil {
		if d < time.Minute {
			return 0, fmt.Errorf("minimum interval is 1 minute")
		}
		return d, nil
	}

	// Named schedules
	switch schedule {
	case "hourly", "0 * * * *":
		return time.Hour, nil
	case "daily", "0 0 * * *":
		return 24 * time.Hour, nil
	case "weekly":
		return 7 * 24 * time.Hour, nil
	}

	return 0, fmt.Errorf("unsupported schedule format: %s", schedule)
}

// filterSchemas filters schemas based on patterns.
func filterSchemas(schemas []string, patterns []string, include bool) []string {
	patternSet := make(map[string]bool)
	for _, p := range patterns {
		patternSet[p] = true
	}

	var result []string
	for _, s := range schemas {
		_, matches := patternSet[s]
		if include && matches {
			result = append(result, s)
		} else if !include && !matches {
			result = append(result, s)
		}
	}
	return result
}
