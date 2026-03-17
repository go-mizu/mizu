package soundcloud

import (
	"context"
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
				for _, u := range inFlight {
					urls = append(urls, u)
				}
				mu.Unlock()
				elapsed := time.Since(start).Seconds()
				rps := 0.0
				if elapsed > 0 {
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
		if err != nil || len(items) == 0 {
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
				runEntityTask(ctx, item, t.Client, t.DB, t.StateDB)
				mu.Lock()
				delete(inFlight, wid)
				mu.Unlock()
				done.Add(1)
			}()
		}
	}

	wg.Wait()
	_, _, _, failed := t.StateDB.QueueStats()
	return CrawlMetric{Done: done.Load(), Failed: failed, Duration: time.Since(start)}, nil
}

func runEntityTask(ctx context.Context, item QueueItem, client *Client, db *DB, stateDB *State) {
	switch item.EntityType {
	case EntityTrack:
		_, _ = (&TrackTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}).Run(ctx, func(*TrackState) {})
	case EntityUser:
		_, _ = (&UserTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}).Run(ctx, func(*UserState) {})
	case EntityPlaylist:
		_, _ = (&PlaylistTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}).Run(ctx, func(*PlaylistState) {})
	default:
		_ = stateDB.Done(item.URL, 0, item.EntityType)
	}
}
