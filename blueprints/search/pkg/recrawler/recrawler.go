package recrawler

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawler"
	"golang.org/x/sync/errgroup"
)

// Recrawler performs high-throughput recrawling of known URL sets.
type Recrawler struct {
	config Config
	client *http.Client
	stats  *Stats
	rdb    *ResultDB

	// Per-domain failure tracking: once a domain fails, skip remaining URLs
	deadDomainsMu sync.RWMutex
	deadDomains   map[string]bool
}

// New creates a recrawler optimized for maximum throughput.
func New(cfg Config, stats *Stats, rdb *ResultDB) *Recrawler {
	if cfg.Workers == 0 {
		cfg.Workers = 2000
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "MizuCrawler/1.0"
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 5000
	}
	if cfg.DomainFailThreshold == 0 {
		cfg.DomainFailThreshold = 1
	}

	// Tight sub-timeouts: dial and TLS are the most time-sensitive phases.
	// Keep them short so we fail fast on unreachable hosts.
	dialTimeout := min(cfg.Timeout/2, 2*time.Second)
	tlsTimeout := min(cfg.Timeout/2, 2*time.Second)

	// Cap idle conns to avoid memory bloat with very high worker counts
	maxIdle := min(cfg.Workers*2, 100000)

	// Build a high-throughput HTTP transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: false},
		MaxIdleConns:          maxIdle,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       0, // unlimited — let workers saturate
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:  tlsTimeout,
		ResponseHeaderTimeout: cfg.Timeout,
		DisableCompression:    true, // skip decompression overhead; we drain body anyway
		ForceAttemptHTTP2:     true, // multiplex on single TCP conn
		WriteBufferSize:       4 * 1024,
		ReadBufferSize:        8 * 1024,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 2 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &Recrawler{
		config:      cfg,
		client:      client,
		stats:       stats,
		rdb:         rdb,
		deadDomains: make(map[string]bool),
	}
}

// SetDeadDomains pre-populates dead domains (e.g. from DNS pre-resolution).
func (r *Recrawler) SetDeadDomains(domains map[string]bool) {
	r.deadDomainsMu.Lock()
	for d := range domains {
		r.deadDomains[d] = true
	}
	r.deadDomainsMu.Unlock()
}

// isDomainDead checks if a domain has been marked dead.
func (r *Recrawler) isDomainDead(domain string) bool {
	r.deadDomainsMu.RLock()
	dead := r.deadDomains[domain]
	r.deadDomainsMu.RUnlock()
	return dead
}

// markDomainDead marks a domain as dead after a fetch failure.
func (r *Recrawler) markDomainDead(domain string) {
	r.deadDomainsMu.Lock()
	r.deadDomains[domain] = true
	r.deadDomainsMu.Unlock()
}

// shuffleURLs randomizes URL order using Fisher-Yates shuffle.
// O(N) time, O(1) extra memory, excellent cache performance.
// Random distribution naturally staggers domain access across workers,
// avoiding thundering-herd on any single domain.
func shuffleURLs(urls []SeedURL) {
	rand.Shuffle(len(urls), func(i, j int) {
		urls[i], urls[j] = urls[j], urls[i]
	})
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

	// Pre-filter dead-domain URLs before entering the worker pipeline.
	// This avoids millions of channel sends and goroutine wakeups.
	var liveURLs []SeedURL
	for _, u := range urls {
		if r.isDomainDead(u.Domain) {
			r.stats.RecordDomainSkip()
		} else {
			liveURLs = append(liveURLs, u)
		}
	}

	if len(liveURLs) == 0 {
		return nil
	}

	// Shuffle live URLs for domain distribution — O(N) in-place
	shuffleURLs(liveURLs)

	// Feed URLs into a channel — large buffer to keep workers fed
	urlCh := make(chan SeedURL, min(len(liveURLs), r.config.Workers*4))
	go func() {
		defer close(urlCh)
		for _, u := range liveURLs {
			select {
			case urlCh <- u:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Launch workers — cap at URL count (no point having idle workers)
	nWorkers := min(r.config.Workers, len(liveURLs))
	g, ctx := errgroup.WithContext(ctx)
	for range nWorkers {
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
			// Runtime domain-death check: domains discovered dead during crawl
			if r.isDomainDead(seed.Domain) {
				r.stats.RecordDomainSkip()
				continue
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
		isFatal := isTimeout || isConnectionRefused(err) || isDNSError(err)
		r.stats.RecordFailure(0, seed.Domain, isTimeout)
		errStr := truncateStr(err.Error(), 200)
		r.rdb.Add(Result{
			URL:         seed.URL,
			Domain:      seed.Domain,
			FetchTimeMs: time.Since(start).Milliseconds(),
			CrawledAt:   time.Now(),
			Error:       errStr,
		})
		// Mark domain as dead if this is a connection-level failure
		if isFatal {
			r.markDomainDead(seed.Domain)
		}
		return
	}

	// Read/discard body to enable connection reuse
	var (
		title       string
		description string
		language    string
		bodySize    int64
	)

	ct := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")

	if !r.config.HeadOnly && resp.StatusCode == 200 && isHTML {
		// Extract basic metadata from body (limit read to 128KB for speed)
		limited := io.LimitReader(resp.Body, 128*1024)
		extracted := crawler.Extract(limited, seed.URL)
		title = extracted.Title
		description = extracted.Description
		language = extracted.Language
	}

	// Fast body drain: limit to 256KB to avoid wasting time on large responses.
	// Connection reuse only works if we drain fully, but for large responses
	// it's faster to close and open a new connection.
	n, _ := io.CopyN(io.Discard, resp.Body, 256*1024)
	resp.Body.Close()

	if resp.ContentLength > 0 {
		bodySize = resp.ContentLength
	} else if n > 0 {
		bodySize = n
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
	r.rdb.Add(Result{
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
	errStr := truncateStr(err.Error(), 200)
	r.rdb.Add(Result{
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

func isConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no route to host")
}

func isDNSError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "dial tcp: lookup")
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// RunWithDisplay runs the recrawl with live terminal display updates.
func RunWithDisplay(ctx context.Context, r *Recrawler, seeds []SeedURL, skip map[string]bool, stats *Stats) error {
	// Track display lines for ANSI cursor movement
	var displayLines int
	var displayMu sync.Mutex

	displayDone := make(chan struct{})
	go func() {
		defer close(displayDone)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				displayMu.Lock()
				if displayLines > 0 {
					fmt.Printf("\033[%dA\033[J", displayLines)
				}
				output := stats.Render()
				fmt.Print(output)
				displayLines = strings.Count(output, "\n")
				displayMu.Unlock()

				// Stop when all URLs are accounted for
				if stats.Done() >= int64(stats.TotalURLs) {
					// Freeze stats at this moment for accurate elapsed time
					stats.Freeze()
					return
				}
			}
		}
	}()

	// Run recrawl (may take longer than display due to pending flushes)
	err := r.Run(ctx, seeds, skip)

	// Wait for display to finish
	<-displayDone

	// Ensure frozen (in case Run finished before display noticed 100%)
	stats.Freeze()

	// Print final frozen stats
	displayMu.Lock()
	if displayLines > 0 {
		fmt.Printf("\033[%dA\033[J", displayLines)
	}
	fmt.Print(stats.Render())
	displayMu.Unlock()

	return err
}
