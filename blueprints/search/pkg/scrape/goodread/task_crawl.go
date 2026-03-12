package goodread

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// CrawlState is the observable state for a CrawlTask.
type CrawlState struct {
	Done     int64
	Pending  int64
	Failed   int64
	InFlight []string
	RPS      float64
}

// CrawlMetric is the final result of a CrawlTask.
type CrawlMetric struct {
	Done     int64
	Failed   int64
	Duration time.Duration
}

// CrawlTask runs the frontier crawl loop: pops URLs from the queue,
// dispatches entity tasks, and tracks progress.
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
		done    atomic.Int64
		failed  atomic.Int64
		mu      sync.Mutex
		inFlight = make(map[int]string) // workerID → URL
	)

	workers := t.Config.Workers
	if workers <= 0 {
		workers = DefaultWorkers
	}

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	// Emit progress periodically
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pending, _, _, f := t.StateDB.QueueStats()
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
				emit(&CrawlState{
					Done:     done.Load(),
					Pending:  pending,
					Failed:   f,
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

		// Check if there's work to do
		pending, _, _, _ := t.StateDB.QueueStats()
		if pending == 0 {
			// Wait briefly for in-flight workers to enqueue new URLs
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

				runEntityTask(ctx, item, t.Client, t.DB, t.StateDB)

				mu.Lock()
				delete(inFlight, wid)
				mu.Unlock()

				done.Add(1)
			}()
		}
	}

	wg.Wait()

	return CrawlMetric{
		Done:     done.Load(),
		Failed:   failed.Load(),
		Duration: time.Since(start),
	}, nil
}

// runEntityTask dispatches a QueueItem to the appropriate entity task.
func runEntityTask(ctx context.Context, item QueueItem, client *Client, db *DB, stateDB *State) {
	switch item.EntityType {
	case "book":
		task := &BookTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}
		task.Run(ctx, func(*BookState) {})
	case "author":
		task := &AuthorTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}
		task.Run(ctx, func(*AuthorState) {})
	case "series":
		task := &SeriesTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}
		task.Run(ctx, func(*SeriesState) {})
	case "list":
		task := &ListTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}
		task.Run(ctx, func(*ListState) {})
	case "quote":
		task := &QuoteTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}
		task.Run(ctx, func(*QuoteState) {})
	case "user":
		task := &UserTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}
		task.Run(ctx, func(*UserState) {})
	case "genre":
		task := &GenreTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}
		task.Run(ctx, func(*GenreState) {})
	case "shelf":
		task := &ShelfTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB}
		task.Run(ctx, func(*ShelfState) {})
	default:
		// Unknown entity type; mark as done to prevent loops
		stateDB.Done(item.URL, 0, item.EntityType)
	}
}
