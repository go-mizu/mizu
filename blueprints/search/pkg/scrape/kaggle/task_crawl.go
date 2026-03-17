package kaggle

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type CrawlState struct {
	Done     int64
	Pending  int64
	Failed   int64
	InFlight []string
	RPS      float64
}

type CrawlMetric struct {
	Done     int64
	Failed   int64
	Duration time.Duration
}

type CrawlTask struct {
	Config  Config
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[CrawlState, CrawlMetric] = (*CrawlTask)(nil)

func (t *CrawlTask) Run(ctx context.Context, emit func(*CrawlState)) (CrawlMetric, error) {
	start := time.Now()
	var (
		done     atomic.Int64
		mu       sync.Mutex
		inFlight = make(map[int]string)
	)

	workers := t.Config.Workers
	if workers <= 0 {
		workers = DefaultWorkers
	}

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pending, _, _, failed := t.StateDB.QueueStats()
				mu.Lock()
				urls := make([]string, 0, len(inFlight))
				for _, rawURL := range inFlight {
					urls = append(urls, rawURL)
				}
				mu.Unlock()
				elapsed := time.Since(start).Seconds()
				rps := 0.0
				if elapsed > 0 {
					rps = float64(done.Load()) / elapsed
				}
				emit(&CrawlState{
					Done:     done.Load(),
					Pending:  pending,
					Failed:   failed,
					InFlight: urls,
					RPS:      rps,
				})
			}
		}
	}()

	workerID := 0
	for {
		if ctx.Err() != nil {
			break
		}
		pending, _, _, _ := t.StateDB.QueueStats()
		if pending == 0 {
			wg.Wait()
			pending, _, _, _ = t.StateDB.QueueStats()
			if pending == 0 {
				break
			}
		}
		items, err := t.StateDB.Pop(workers)
		if err != nil {
			fmt.Printf("queue pop error: %v\n", err)
			break
		}
		if len(items) == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, item := range items {
			item := item
			wid := workerID
			workerID++
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				mu.Lock()
				inFlight[wid] = item.URL
				mu.Unlock()
				t.runEntityTask(ctx, item)
				mu.Lock()
				delete(inFlight, wid)
				mu.Unlock()
				done.Add(1)
			}()
		}
	}

	wg.Wait()
	_, _, _, failed := t.StateDB.QueueStats()
	return CrawlMetric{
		Done:     done.Load(),
		Failed:   failed,
		Duration: time.Since(start),
	}, nil
}

func (t *CrawlTask) runEntityTask(ctx context.Context, item QueueItem) {
	switch item.EntityType {
	case EntityDataset:
		task := &DatasetTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}
		_, _ = task.Run(ctx, func(*DatasetState) {})
	case EntityModel:
		task := &ModelTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}
		_, _ = task.Run(ctx, func(*ModelState) {})
	case EntityCompetition:
		task := &CompetitionTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}
		_, _ = task.Run(ctx, func(*CompetitionState) {})
	case EntityNotebook:
		task := &NotebookTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}
		_, _ = task.Run(ctx, func(*NotebookState) {})
	case EntityProfile:
		task := &ProfileTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}
		_, _ = task.Run(ctx, func(*ProfileState) {})
	default:
		_ = t.StateDB.Done(item.URL, item.EntityType, 0)
	}
}
