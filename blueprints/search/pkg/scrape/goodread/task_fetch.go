package goodread

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// HTMLFetcher is the minimal interface satisfied by Client, RodClient, and WorkerClient.
type HTMLFetcher interface {
	FetchHTML(ctx context.Context, url string) (*goquery.Document, int, error)
}

// FetchState is the observable state for a FetchTask.
type FetchState struct {
	Fetched  int64
	Failed   int64
	Fetched2 int64 // already-fetched (skipped) — non-zero during import pipeline
	InFlight []string
	RPS      float64
}

// FetchMetric is the final result of a FetchTask.
type FetchMetric struct {
	Fetched  int64
	Failed   int64
	Duration time.Duration
}

// FetchTask runs Phase 1 of the two-phase pipeline: pops pending URLs, downloads
// HTML to disk as .html.gz, and marks them 'fetched' in the state DB.
// It does NOT write to the main data DB — only disk + state.
type FetchTask struct {
	Config  Config
	Fetcher HTMLFetcher
	StateDB *State
}

var _ core.Task[FetchState, FetchMetric] = (*FetchTask)(nil)

func (t *FetchTask) Run(ctx context.Context, emit func(*FetchState)) (FetchMetric, error) {
	start := time.Now()

	var (
		fetched  atomic.Int64
		failed   atomic.Int64
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
				mu.Lock()
				urls := make([]string, 0, len(inFlight))
				for _, u := range inFlight {
					urls = append(urls, u)
				}
				mu.Unlock()

				elapsed := time.Since(start).Seconds()
				rps := 0.0
				if elapsed > 0 {
					rps = float64(fetched.Load()) / elapsed
				}
				emit(&FetchState{
					Fetched:  fetched.Load(),
					Failed:   failed.Load(),
					InFlight: urls,
					RPS:      rps,
				})
			}
		}
	}()

	popBatch := workers * 4
	if popBatch < 16 {
		popBatch = 16
	}

	workerID := 0
	for {
		if ctx.Err() != nil {
			break
		}

		items, err := t.StateDB.Pop(popBatch)
		if err != nil {
			fmt.Printf("queue pop error: %v\n", err)
			break
		}
		if len(items) == 0 {
			wg.Wait()
			items, err = t.StateDB.Pop(popBatch)
			if err != nil || len(items) == 0 {
				break
			}
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

				if err := t.fetchAndSave(ctx, item); err != nil {
					failed.Add(1)
				} else {
					fetched.Add(1)
				}

				mu.Lock()
				delete(inFlight, wid)
				mu.Unlock()
			}()
		}
	}

	wg.Wait()

	return FetchMetric{
		Fetched:  fetched.Load(),
		Failed:   failed.Load(),
		Duration: time.Since(start),
	}, nil
}

func (t *FetchTask) fetchAndSave(ctx context.Context, item QueueItem) error {
	doc, code, err := t.Fetcher.FetchHTML(ctx, item.URL)
	if err != nil {
		t.StateDB.Fail(item.URL, err.Error())
		return err
	}
	if code == 404 {
		t.StateDB.Done(item.URL, 404, item.EntityType)
		return nil // not an error — permanent 404
	}
	if doc == nil {
		msg := fmt.Sprintf("HTTP %d", code)
		t.StateDB.Fail(item.URL, msg)
		return fmt.Errorf("%s", msg)
	}

	html, err := doc.Html()
	if err != nil {
		t.StateDB.Fail(item.URL, err.Error())
		return err
	}

	id := entityIDFromURL(item.URL)
	if id == "" {
		id = fmt.Sprintf("item_%d", item.ID)
	}

	htmlPath := HTMLCachePath(t.Config.DataDir, item.EntityType, id)
	if err := SaveHTML(htmlPath, html); err != nil {
		t.StateDB.Fail(item.URL, err.Error())
		return err
	}

	return t.StateDB.MarkFetched(item.URL, htmlPath)
}
