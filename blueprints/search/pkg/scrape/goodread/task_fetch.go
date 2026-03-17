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
//
// If Fetcher also implements BatchHTMLFetcher, FetchTask uses batch mode:
// each worker goroutine pops BatchSize URLs and sends them all in one HTTP call,
// achieving much higher throughput (e.g. 20 workers × 50 URLs/batch = ~100 rps).
type FetchTask struct {
	Config    Config
	Fetcher   HTMLFetcher
	StateDB   *State
	BatchSize int // URLs per batch when Fetcher implements BatchHTMLFetcher (default 50)
}

var _ core.Task[FetchState, FetchMetric] = (*FetchTask)(nil)

func (t *FetchTask) Run(ctx context.Context, emit func(*FetchState)) (FetchMetric, error) {
	if bf, ok := t.Fetcher.(BatchHTMLFetcher); ok {
		return t.runBatch(ctx, emit, bf)
	}
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

// runBatch is the high-throughput path used when Fetcher implements BatchHTMLFetcher.
// Each worker goroutine pops BatchSize URLs and sends them all in one batch call.
func (t *FetchTask) runBatch(ctx context.Context, emit func(*FetchState), bf BatchHTMLFetcher) (FetchMetric, error) {
	start := time.Now()

	var (
		fetched atomic.Int64
		failed  atomic.Int64
		mu      sync.Mutex
		inFlight = make(map[int]string)
	)

	workers := t.Config.Workers
	if workers <= 0 {
		workers = DefaultWorkers
	}
	batchSize := t.BatchSize
	if batchSize <= 0 {
		batchSize = 50
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
				emit(&FetchState{Fetched: fetched.Load(), Failed: failed.Load(), InFlight: urls, RPS: rps})
			}
		}
	}()

	workerID := 0
	for {
		if ctx.Err() != nil {
			break
		}

		items, err := t.StateDB.Pop(workers * batchSize)
		if err != nil {
			fmt.Printf("queue pop error: %v\n", err)
			break
		}
		if len(items) == 0 {
			wg.Wait()
			items, err = t.StateDB.Pop(workers * batchSize)
			if err != nil || len(items) == 0 {
				break
			}
		}

		// Split items into batches of batchSize, one goroutine per batch.
		for i := 0; i < len(items); i += batchSize {
			end := i + batchSize
			if end > len(items) {
				end = len(items)
			}
			batch := items[i:end]
			wid := workerID
			workerID++

			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				urls := make([]string, len(batch))
				for j, it := range batch {
					urls[j] = it.URL
					mu.Lock()
					inFlight[wid*batchSize+j] = it.URL
					mu.Unlock()
				}

				results, err := bf.FetchHTMLBatch(ctx, urls)

				// Clean inFlight for this batch.
				mu.Lock()
				for j := range batch {
					delete(inFlight, wid*batchSize+j)
				}
				mu.Unlock()

				if err != nil {
					fmt.Printf("\n[batch error] %v\n", err)
					for _, it := range batch {
						t.StateDB.Fail(it.URL, err.Error())
						failed.Add(1)
					}
					return
				}

				// Map results back by URL.
				byURL := make(map[string]BatchHTMLResult, len(results))
				for _, r := range results {
					byURL[r.URL] = r
				}

				for _, it := range batch {
					r, ok := byURL[it.URL]
					if !ok {
						fmt.Printf("\n[batch miss] url=%s not in response (got %d)\n", it.URL, len(results))
						t.StateDB.Fail(it.URL, "missing from batch response")
						failed.Add(1)
						continue
					}
					if err := t.saveBatchResult(it, r); err != nil {
						fmt.Printf("\n[save error] url=%s err=%v\n", it.URL, err)
						failed.Add(1)
					} else {
						fetched.Add(1)
					}
				}
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

func (t *FetchTask) saveBatchResult(item QueueItem, r BatchHTMLResult) error {
	if r.Err != nil {
		t.StateDB.Fail(item.URL, r.Err.Error())
		return r.Err
	}
	if r.StatusCode == 404 {
		t.StateDB.Done(item.URL, 404, item.EntityType)
		return nil
	}
	if r.Doc == nil {
		msg := fmt.Sprintf("HTTP %d", r.StatusCode)
		t.StateDB.Fail(item.URL, msg)
		return fmt.Errorf("%s", msg)
	}

	html, err := r.Doc.Html()
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
