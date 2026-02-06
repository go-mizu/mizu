package crawler

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// RecrawlConfig holds configuration for high-throughput recrawling.
type RecrawlConfig struct {
	Workers   int           // Concurrent workers (default: 500)
	Timeout   time.Duration // Per-request timeout (default: 10s)
	UserAgent string        // User-Agent header
	HeadOnly  bool          // Only fetch headers, skip body
	BatchSize int           // DB write batch size (default: 1000)
	Resume    bool          // Skip already-crawled URLs
}

// DefaultRecrawlConfig returns optimal defaults for high throughput.
func DefaultRecrawlConfig() RecrawlConfig {
	return RecrawlConfig{
		Workers:   500,
		Timeout:   10 * time.Second,
		UserAgent: "MizuCrawler/1.0",
		HeadOnly:  false,
		BatchSize: 1000,
	}
}

// Recrawler performs high-throughput recrawling of known URL sets.
type Recrawler struct {
	config RecrawlConfig
	client *http.Client
	stats  *RecrawlStats
	rdb    *ResultDB
}

// NewRecrawler creates a recrawler optimized for maximum throughput.
func NewRecrawler(cfg RecrawlConfig, stats *RecrawlStats, rdb *ResultDB) *Recrawler {
	if cfg.Workers == 0 {
		cfg.Workers = 500
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "MizuCrawler/1.0"
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1000
	}

	// Build a high-throughput HTTP transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
		MaxIdleConns:         cfg.Workers * 2,
		MaxIdleConnsPerHost:  10,
		MaxConnsPerHost:      20,
		IdleConnTimeout:      90 * time.Second,
		TLSHandshakeTimeout:  5 * time.Second,
		ResponseHeaderTimeout: cfg.Timeout,
		DisableCompression:   true, // avoid gzip overhead for speed
		ForceAttemptHTTP2:    false, // HTTP/1.1 is faster for many small requests
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &Recrawler{
		config: cfg,
		client: client,
		stats:  stats,
		rdb:    rdb,
	}
}

// Run executes the recrawl on the given URL set.
func (r *Recrawler) Run(ctx context.Context, seeds []SeedURL, skip map[string]bool) error {
	// Filter out already-crawled URLs
	var urls []SeedURL
	for _, s := range seeds {
		if skip != nil && skip[s.URL] {
			r.stats.RecordSkip()
			continue
		}
		urls = append(urls, s)
	}

	if len(urls) == 0 {
		return nil
	}

	// Feed URLs into a channel
	urlCh := make(chan SeedURL, r.config.Workers*2)
	go func() {
		defer close(urlCh)
		for _, u := range urls {
			select {
			case urlCh <- u:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Launch workers
	g, ctx := errgroup.WithContext(ctx)
	for range r.config.Workers {
		g.Go(func() error {
			return r.worker(ctx, urlCh)
		})
	}

	return g.Wait()
}

func (r *Recrawler) worker(ctx context.Context, urls <-chan SeedURL) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case seed, ok := <-urls:
			if !ok {
				return nil
			}
			r.fetchOne(ctx, seed)
		}
	}
}

func (r *Recrawler) fetchOne(ctx context.Context, seed SeedURL) {
	start := time.Now()

	method := http.MethodGet
	if r.config.HeadOnly {
		method = http.MethodHead
	}

	req, err := http.NewRequestWithContext(ctx, method, seed.URL, nil)
	if err != nil {
		r.recordError(seed, 0, start, err)
		return
	}
	req.Header.Set("User-Agent", r.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")

	resp, err := r.client.Do(req)
	if err != nil {
		isTimeout := isTimeoutError(err)
		r.stats.RecordFailure(0, seed.Domain, isTimeout)
		errStr := err.Error()
		// Truncate long error messages
		if len(errStr) > 200 {
			errStr = errStr[:200]
		}
		r.rdb.Add(RecrawlResult{
			URL:         seed.URL,
			Domain:      seed.Domain,
			FetchTimeMs: time.Since(start).Milliseconds(),
			CrawledAt:   time.Now(),
			Error:       errStr,
		})
		return
	}

	// Read/discard body to enable connection reuse
	var (
		title       string
		description string
		language    string
		bodySize    int64
	)

	if !r.config.HeadOnly && resp.StatusCode == 200 &&
		(strings.Contains(resp.Header.Get("Content-Type"), "text/html") ||
			strings.Contains(resp.Header.Get("Content-Type"), "application/xhtml")) {
		// Extract basic metadata from body
		limited := io.LimitReader(resp.Body, 512*1024) // 512KB max for metadata
		extracted := Extract(limited, seed.URL)
		title = extracted.Title
		description = extracted.Description
		language = extracted.Language
		// Drain remaining body for connection reuse
		io.Copy(io.Discard, resp.Body)
	} else {
		io.Copy(io.Discard, resp.Body)
	}
	resp.Body.Close()

	if resp.ContentLength > 0 {
		bodySize = resp.ContentLength
	}

	fetchMs := time.Since(start).Milliseconds()

	// Determine redirect
	redirectURL := ""
	if resp.Request != nil && resp.Request.URL.String() != seed.URL {
		redirectURL = resp.Request.URL.String()
	}

	// Record stats
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		r.stats.RecordSuccess(resp.StatusCode, seed.Domain, bodySize, fetchMs)
	} else {
		r.stats.RecordFailure(resp.StatusCode, seed.Domain, false)
	}

	// Store result
	r.rdb.Add(RecrawlResult{
		URL:           seed.URL,
		StatusCode:    resp.StatusCode,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: bodySize,
		Title:         title,
		Description:   description,
		Language:      language,
		Domain:        seed.Domain,
		RedirectURL:   redirectURL,
		FetchTimeMs:   fetchMs,
		CrawledAt:     time.Now(),
	})
}

func (r *Recrawler) recordError(seed SeedURL, statusCode int, start time.Time, err error) {
	isTimeout := isTimeoutError(err)
	r.stats.RecordFailure(statusCode, seed.Domain, isTimeout)
	errStr := err.Error()
	if len(errStr) > 200 {
		errStr = errStr[:200]
	}
	r.rdb.Add(RecrawlResult{
		URL:         seed.URL,
		Domain:      seed.Domain,
		FetchTimeMs: time.Since(start).Milliseconds(),
		CrawledAt:   time.Now(),
		Error:       errStr,
	})
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline")
}

// RunWithDisplay runs the recrawl with live terminal display updates.
func RunWithDisplay(ctx context.Context, r *Recrawler, seeds []SeedURL, skip map[string]bool, stats *RecrawlStats) error {
	// Start display goroutine
	displayDone := make(chan struct{})
	var displayOnce sync.Once
	go func() {
		defer close(displayDone)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		lines := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Clear previous output
				if lines > 0 {
					fmt.Printf("\033[%dA\033[J", lines)
				}
				output := stats.Render()
				fmt.Print(output)
				lines = strings.Count(output, "\n")

				// Stop if all done
				if stats.Done() >= int64(stats.totalURLs) {
					return
				}
			}
		}
	}()

	// Run recrawl
	err := r.Run(ctx, seeds, skip)

	// Wait a moment for final display
	displayOnce.Do(func() {
		time.Sleep(600 * time.Millisecond)
	})

	// Final display
	fmt.Printf("\033[%dA\033[J", strings.Count(stats.Render(), "\n"))
	fmt.Print(stats.Render())

	return err
}
