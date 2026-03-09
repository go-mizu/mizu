package dcrawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DataDog/zstd"
	"github.com/cespare/xxhash/v2"
	"golang.org/x/sync/errgroup"
)

const (
	defaultWorkerURL      = "https://crawler.go-mizu.workers.dev"
	defaultWorkerBatch    = 10
	defaultWorkerParallel = 20
)

// workerRequest is the JSON body sent to POST /crawl.
type workerRequest struct {
	URLs    []string `json:"urls"`
	Browser bool     `json:"browser,omitempty"`
	Timeout int      `json:"timeout,omitempty"`
}

// workerResult is one element of the JSON response from POST /crawl.
type workerResult struct {
	URL           string  `json:"url"`
	Status        int     `json:"status"`
	HTML          *string `json:"html"`
	Markdown      *string `json:"markdown"`
	Title         *string `json:"title"`
	ContentType   *string `json:"content_type"`
	ContentLength int     `json:"content_length"`
	RedirectURL   *string `json:"redirect_url"`
	FetchTimeMs   int64   `json:"fetch_time_ms"`
	Error         *string `json:"error"`
}

// WorkerClient sends batch crawl requests to the CF Worker.
type WorkerClient struct {
	url     string
	token   string
	browser bool
	timeout int // per-URL timeout in ms
	client  *http.Client
}

// NewWorkerClient creates a worker client from config.
func NewWorkerClient(cfg Config) *WorkerClient {
	workerURL := cfg.WorkerURL
	if workerURL == "" {
		workerURL = defaultWorkerURL
	}
	return &WorkerClient{
		url:     workerURL,
		token:   cfg.WorkerToken,
		browser: cfg.WorkerBrowser,
		timeout: int(cfg.Timeout.Milliseconds()),
		client: &http.Client{
			Timeout: 60 * time.Second, // overall HTTP timeout for batch request
		},
	}
}

// FetchBatch sends a batch of URLs to the worker and returns results.
func (wc *WorkerClient) FetchBatch(ctx context.Context, urls []string) ([]workerResult, error) {
	reqBody := workerRequest{
		URLs:    urls,
		Browser: wc.browser,
		Timeout: wc.timeout,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", wc.url+"/crawl", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+wc.token)

	resp, err := wc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("worker request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("worker returned %d: %s", resp.StatusCode, string(respBody))
	}

	var results []workerResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return results, nil
}

// runWorkerMode runs the crawler in worker-proxied mode.
// Uses errgroup with concurrency limit for clean resource management.
func (c *Crawler) runWorkerMode(ctx context.Context) error {
	wc := NewWorkerClient(c.config)

	batchSize := c.config.WorkerBatch
	if batchSize <= 0 {
		batchSize = defaultWorkerBatch
	}
	parallel := c.config.WorkerParallel
	if parallel <= 0 {
		parallel = defaultWorkerParallel
	}

	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(parallel)

	done := false
	for !done {
		// Check max pages (before collecting batch)
		if c.config.MaxPages > 0 && c.stats.success.Load() >= int64(c.config.MaxPages) {
			break
		}

		if ctx.Err() != nil {
			break
		}

		// Collect a batch from frontier
		batch := c.collectBatch(ctx, batchSize)
		if len(batch) == 0 {
			// Frontier empty and no in-flight work — done
			if c.stats.inFlight.Load() == 0 {
				// Double-check after a short sleep to avoid race
				// with goroutines adding links back to frontier
				time.Sleep(200 * time.Millisecond)
				if c.frontier.Len() == 0 && c.stats.inFlight.Load() == 0 {
					done = true
				}
			} else {
				time.Sleep(100 * time.Millisecond)
			}
			continue
		}

		c.stats.inFlight.Add(int64(len(batch)))

		g.Go(func() error {
			defer c.stats.inFlight.Add(-int64(len(batch)))

			urls := make([]string, len(batch))
			for i, it := range batch {
				urls[i] = it.URL
			}

			// Retry on 503 (CF Worker overload / error 1102) with backoff.
			var results []workerResult
			var err error
			for attempt := 0; attempt < 5; attempt++ {
				if ctx.Err() != nil {
					return nil
				}
				if attempt > 0 {
					time.Sleep(time.Duration(1<<attempt) * time.Second) // 2s, 4s, 8s, 16s
				}
				fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 60*time.Second)
				results, err = wc.FetchBatch(fetchCtx, urls)
				fetchCancel()
				if err == nil {
					break
				}
				// Only retry on 503 (worker overload); other errors are permanent.
				if !strings.Contains(err.Error(), "503") {
					break
				}
			}
			if err != nil {
				// If the parent context was canceled (crawl stopped),
				// silently drop — don't record as errors.
				if ctx.Err() != nil {
					return nil
				}
				// Record genuine worker errors
				for _, it := range batch {
					c.stats.RecordFailure(0, true)
					c.resultDB.AddPage(Result{
						URL: it.URL, URLHash: xxhash.Sum64String(it.URL),
						Depth: it.Depth, CrawledAt: time.Now(),
						Error: fmt.Sprintf("worker: %v", err),
					})
				}
				return nil
			}

			// Build URL→item map for depth lookup
			itemMap := make(map[string]CrawlItem, len(batch))
			for _, it := range batch {
				itemMap[it.URL] = it
			}

			for _, wr := range results {
				item := itemMap[wr.URL]
				c.processWorkerResult(item, wr)
			}
			return nil
		})
	}

	// Wait for all in-flight batches to complete
	return g.Wait()
}

// collectBatch drains up to n items from the frontier channel.
func (c *Crawler) collectBatch(ctx context.Context, n int) []CrawlItem {
	var batch []CrawlItem

	// Block on first item
	select {
	case <-ctx.Done():
		return nil
	case item := <-c.frontier.ch:
		batch = append(batch, item)
	case <-time.After(500 * time.Millisecond):
		return nil
	}

	// Drain up to n-1 more without blocking
	for len(batch) < n {
		select {
		case item := <-c.frontier.ch:
			batch = append(batch, item)
		default:
			return batch
		}
	}
	return batch
}

// processWorkerResult handles a single result from the worker.
func (c *Crawler) processWorkerResult(item CrawlItem, wr workerResult) {
	if wr.Error != nil && *wr.Error != "" {
		c.stats.RecordFailure(0, true)
		c.resultDB.AddPage(Result{
			URL: item.URL, URLHash: xxhash.Sum64String(item.URL),
			Depth: item.Depth, CrawledAt: time.Now(),
			Error: *wr.Error,
		})
		return
	}

	result := Result{
		URL:           item.URL,
		URLHash:       xxhash.Sum64String(item.URL),
		Depth:         item.Depth,
		StatusCode:    wr.Status,
		ContentLength: int64(wr.ContentLength),
		FetchTimeMs:   wr.FetchTimeMs,
		CrawledAt:     time.Now(),
	}
	if wr.ContentType != nil {
		result.ContentType = *wr.ContentType
	}
	if wr.RedirectURL != nil {
		result.RedirectURL = *wr.RedirectURL
	}
	if wr.Title != nil {
		result.Title = *wr.Title
	}

	// Store HTML (zstd compressed) and markdown
	if wr.HTML != nil && *wr.HTML != "" {
		htmlBytes := []byte(*wr.HTML)
		result.BodyHash = xxhash.Sum64(htmlBytes)

		if compressed, err := zstd.Compress(nil, htmlBytes); err == nil {
			result.HTML = compressed
		}

		// Extract links and metadata from HTML (Go-side)
		if wr.Status >= 200 && wr.Status < 400 {
			baseURL, _ := url.Parse(item.URL)
			if wr.RedirectURL != nil && *wr.RedirectURL != "" {
				if ru, err := url.Parse(*wr.RedirectURL); err == nil {
					baseURL = ru
				}
			}
			meta := ExtractLinksAndMeta(htmlBytes, baseURL, c.config.Domain, c.config.ExtractImages)
			if result.Title == "" {
				result.Title = meta.Title
			}
			result.Description = meta.Description
			result.Language = meta.Language
			result.Canonical = meta.Canonical
			result.LinkCount = len(meta.Links)
			c.stats.RecordLinks(len(meta.Links))

			// Feed internal links back to frontier
			if c.config.MaxDepth == 0 || item.Depth < c.config.MaxDepth {
				for _, link := range meta.Links {
					if link.IsInternal {
						c.frontier.TryAdd(link.TargetURL, item.Depth+1)
					}
				}
			}
			if c.config.StoreLinks && len(meta.Links) > 0 {
				c.resultDB.AddLinks(result.URLHash, meta.Links)
			}
		}
	}

	if wr.Markdown != nil && *wr.Markdown != "" {
		result.Markdown = *wr.Markdown
	}

	c.resultDB.AddPage(result)
	if wr.Status >= 200 && wr.Status < 400 {
		bodyLen := int64(0)
		if wr.HTML != nil {
			bodyLen = int64(len(*wr.HTML))
		}
		c.stats.RecordSuccess(wr.Status, bodyLen, wr.FetchTimeMs)
	} else {
		c.stats.RecordFailure(wr.Status, false)
	}
	c.stats.RecordDepth(item.Depth)
}
