package pack

import (
	"context"
	"time"
)

// PackProgressFunc reports progress during pack read operations.
// done: docs processed so far, total: expected total (0 = unknown), elapsed: wall time.
type PackProgressFunc func(done, total int64, elapsed time.Duration)

// RunPipelineFromChannel drains docCh into engine via batchIndex.
// Starts a progress goroutine (200ms tick) if progress != nil.
// Drains docCh after batchIndex returns to unblock any sender goroutine.
func RunPipelineFromChannel(ctx context.Context, engine Engine, docCh <-chan Document, total int64, batchSize int, progress PackProgressFunc) (*PipelineStats, error) {
	if batchSize <= 0 {
		batchSize = 5000
	}
	stats := &PipelineStats{StartTime: time.Now()}

	// Track peak memory
	memStop := make(chan struct{})
	go trackPeakMem(stats, memStop)
	defer close(memStop)

	var stopProgress chan struct{}
	if progress != nil {
		stopProgress = make(chan struct{})
		go func() {
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					progress(stats.DocsIndexed.Load(), total, time.Since(stats.StartTime))
				case <-stopProgress:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	err := batchIndex(ctx, engine, docCh, batchSize, stats)

	if stopProgress != nil {
		close(stopProgress)
	}
	// Drain docCh to unblock any sender goroutine still running after batchIndex returned.
	for range docCh {
	}

	return stats, err
}

// funcEngine is an Engine adapter backed by an Index function.
// Open, Close, Stats, and Search are no-ops.
type funcEngine struct {
	name    string
	indexFn func(context.Context, []Document) error
}

func (e *funcEngine) Name() string                                       { return e.name }
func (e *funcEngine) Open(_ context.Context, _ string) error             { return nil }
func (e *funcEngine) Close() error                                       { return nil }
func (e *funcEngine) Stats(_ context.Context) (EngineStats, error)       { return EngineStats{}, nil }
func (e *funcEngine) Index(ctx context.Context, docs []Document) error   { return e.indexFn(ctx, docs) }
func (e *funcEngine) Search(_ context.Context, _ Query) (Results, error) { return Results{}, nil }
