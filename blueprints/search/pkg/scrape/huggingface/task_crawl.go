package huggingface

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
		inFlight = map[int]string{}
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
				for _, raw := range inFlight {
					urls = append(urls, raw)
				}
				mu.Unlock()
				rps := 0.0
				if elapsed := time.Since(start).Seconds(); elapsed > 0 {
					rps = float64(done.Load()) / elapsed
				}
				emit(&CrawlState{Done: done.Load(), Pending: pending, Failed: failed, InFlight: urls, RPS: rps})
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
			return CrawlMetric{}, fmt.Errorf("queue pop: %w", err)
		}
		if len(items) == 0 {
			time.Sleep(300 * time.Millisecond)
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
	return CrawlMetric{Done: done.Load(), Duration: time.Since(start)}, nil
}

func (t *CrawlTask) runEntityTask(ctx context.Context, item QueueItem) {
	switch item.EntityType {
	case EntityModel:
		(&ModelTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}).Run(ctx, func(*ModelState) {})
	case EntityDataset:
		(&DatasetTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}).Run(ctx, func(*DatasetState) {})
	case EntitySpace:
		(&SpaceTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}).Run(ctx, func(*SpaceState) {})
	case EntityCollection:
		(&CollectionTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}).Run(ctx, func(*CollectionState) {})
	case EntityPaper:
		(&PaperTask{URL: item.URL, Client: t.Client, DB: t.DB, StateDB: t.StateDB}).Run(ctx, func(*PaperState) {})
	default:
		_ = t.StateDB.Done(item.URL, 0, item.EntityType)
	}
}
