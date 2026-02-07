package cc

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Fetcher extracts pages from WARC files via byte-range requests.
type Fetcher struct {
	config Config
	client *Client
	stats  *FetchStats
	rdb    *ResultDB
}

// NewFetcher creates a new WARC page fetcher.
func NewFetcher(cfg Config, client *Client, stats *FetchStats, rdb *ResultDB) *Fetcher {
	return &Fetcher{
		config: cfg,
		client: client,
		stats:  stats,
		rdb:    rdb,
	}
}

// Run fetches all WARC records from the given pointers.
func (f *Fetcher) Run(ctx context.Context, pointers []WARCPointer, skip map[string]bool) error {
	// Filter skipped URLs
	var live []WARCPointer
	for _, p := range pointers {
		if skip != nil && skip[p.URL] {
			f.stats.RecordSkip()
			continue
		}
		live = append(live, p)
	}

	if len(live) == 0 {
		return nil
	}

	// Shuffle for load distribution across WARC files
	rand.Shuffle(len(live), func(i, j int) {
		live[i], live[j] = live[j], live[i]
	})

	// Feed pointers through a channel
	ptrCh := make(chan WARCPointer, min(len(live), 100000))
	go func() {
		defer close(ptrCh)
		for _, p := range live {
			select {
			case ptrCh <- p:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Launch workers
	workers := f.config.Workers
	if workers <= 0 {
		workers = 5000
	}
	if workers > len(live) {
		workers = len(live)
	}

	g, gctx := errgroup.WithContext(ctx)
	for i := range workers {
		workerID := i
		g.Go(func() error {
			f.worker(gctx, workerID, ptrCh)
			return nil
		})
	}

	return g.Wait()
}

// worker processes WARC pointers from the channel.
func (f *Fetcher) worker(ctx context.Context, id int, ptrCh <-chan WARCPointer) {
	maxBody := f.config.MaxBodySize
	if maxBody <= 0 {
		maxBody = 512 * 1024
	}

	for p := range ptrCh {
		if ctx.Err() != nil {
			return
		}

		start := time.Now()

		// Fetch WARC record via byte-range request
		data, err := f.client.FetchWARCRecord(ctx, id, p)
		fetchMs := time.Since(start).Milliseconds()

		if err != nil {
			f.stats.RecordFailure()
			f.rdb.Add(PageResult{
				URL:          p.URL,
				Domain:       p.Domain,
				WARCFilename: p.WARCFilename,
				FetchTimeMs:  fetchMs,
				Error:        err.Error(),
			})
			continue
		}

		// Parse WARC record
		resp, err := ParseWARCRecord(data)
		if err != nil {
			f.stats.RecordFailure()
			f.rdb.Add(PageResult{
				URL:          p.URL,
				Domain:       p.Domain,
				WARCFilename: p.WARCFilename,
				FetchTimeMs:  fetchMs,
				Error:        fmt.Sprintf("parse: %v", err),
			})
			continue
		}

		// Extract page info
		body := resp.Body
		bodyStr := ""
		if len(body) > maxBody {
			body = body[:maxBody]
		}

		contentType := p.ContentType
		if ct, ok := resp.HTTPHeaders["Content-Type"]; ok {
			contentType = ct
		}

		// Only store body for HTML content
		if isHTML(contentType) {
			bodyStr = string(body)
		}

		title, description := ExtractPageInfo(body)

		f.stats.RecordSuccess(int64(len(data)), int64(len(resp.Body)), fetchMs)

		f.rdb.Add(PageResult{
			URL:           p.URL,
			StatusCode:    resp.HTTPStatus,
			ContentType:   contentType,
			ContentLength: int64(len(resp.Body)),
			Body:          bodyStr,
			Title:         title,
			Description:   description,
			Language:      p.Language,
			Domain:        p.Domain,
			WARCFilename:  p.WARCFilename,
			FetchTimeMs:   fetchMs,
			CrawledAt:     resp.Date,
		})
	}
}

func isHTML(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
}

// RunWithDisplay runs the fetcher with a live progress display.
func RunWithDisplay(ctx context.Context, f *Fetcher, pointers []WARCPointer, skip map[string]bool, stats *FetchStats) error {
	// Start display ticker
	displayCtx, displayCancel := context.WithCancel(ctx)
	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go func() {
		defer displayWg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		var lines int
		for {
			select {
			case <-ticker.C:
				if lines > 0 {
					fmt.Printf("\033[%dA\033[J", lines)
				}
				output := stats.Render()
				fmt.Print(output)
				lines = strings.Count(output, "\n")
			case <-displayCtx.Done():
				return
			}
		}
	}()

	// Run fetcher
	err := f.Run(ctx, pointers, skip)

	// Final display
	stats.Freeze()
	displayCancel()
	displayWg.Wait()

	// Print final stats
	fmt.Print(stats.Render())
	fmt.Println()

	return err
}
