package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// Crawler is a concurrent web crawler with robots.txt compliance,
// per-domain politeness, and resumable state.
type Crawler struct {
	config   Config
	client   *http.Client
	frontier *Frontier
	robots   *RobotsCache
	startURL string

	// Callbacks
	onResult   ResultFn
	onProgress ProgressFn

	// Stats
	pagesSuccess atomic.Int64
	pagesFailed  atomic.Int64
	pagesSkipped atomic.Int64
	bytesTotal   atomic.Int64
	startTime    time.Time
}

// New creates a new Crawler with the given config.
func New(config Config) (*Crawler, error) {
	config = config.merge()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	client := &http.Client{
		Timeout: config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &Crawler{
		config:   config,
		client:   client,
		frontier: NewFrontier(config.Delay),
		robots:   NewRobotsCache(client, config.UserAgent),
	}, nil
}

// OnResult sets the callback for each crawled page.
func (c *Crawler) OnResult(fn ResultFn) {
	c.onResult = fn
}

// OnProgress sets the callback for progress updates.
func (c *Crawler) OnProgress(fn ProgressFn) {
	c.onProgress = fn
}

// Crawl starts crawling from a URL and returns aggregate stats.
func (c *Crawler) Crawl(ctx context.Context, startURL string) (CrawlStats, error) {
	normalized, err := NormalizeURL(startURL)
	if err != nil {
		return CrawlStats{}, fmt.Errorf("invalid start URL: %w", err)
	}
	c.startURL = normalized
	c.startTime = time.Now()

	// Restore from state file if resume
	if c.config.StateFile != "" {
		if state, err := LoadState(c.config.StateFile); err == nil && state != nil {
			c.restoreState(state)
		}
	}

	// Seed the frontier
	c.frontier.Push(URLEntry{URL: normalized, Depth: 0, Priority: 0})

	return c.run(ctx)
}

// CrawlSitemap crawls URLs from a sitemap.
func (c *Crawler) CrawlSitemap(ctx context.Context, sitemapURL string) (CrawlStats, error) {
	c.startURL = sitemapURL
	c.startTime = time.Now()

	urls, err := FetchSitemap(c.client, sitemapURL, c.config.MaxPages)
	if err != nil {
		return CrawlStats{}, fmt.Errorf("fetching sitemap: %w", err)
	}

	for i, u := range urls {
		c.frontier.Push(URLEntry{
			URL:      u.URL,
			Depth:    0,
			Priority: i, // Maintain sitemap order
		})
	}

	return c.run(ctx)
}

// Stats returns the current crawl statistics.
func (c *Crawler) Stats() CrawlStats {
	success := int(c.pagesSuccess.Load())
	failed := int(c.pagesFailed.Load())
	skipped := int(c.pagesSkipped.Load())
	elapsed := time.Since(c.startTime)
	pps := 0.0
	if elapsed > 0 {
		pps = float64(success+failed) / elapsed.Seconds()
	}
	return CrawlStats{
		PagesTotal:     success + failed + skipped,
		PagesSuccess:   success,
		PagesFailed:    failed,
		PagesSkipped:   skipped,
		BytesTotal:     c.bytesTotal.Load(),
		Duration:       elapsed,
		PagesPerSecond: pps,
	}
}

func (c *Crawler) run(ctx context.Context) (CrawlStats, error) {
	g, ctx := errgroup.WithContext(ctx)

	// Limit concurrency
	workers := c.config.Workers
	if workers < 1 {
		workers = 1
	}

	// Track active workers for clean shutdown
	var active atomic.Int64

	for range workers {
		g.Go(func() error {
			for {
				// Check max pages
				total := int(c.pagesSuccess.Load()) + int(c.pagesFailed.Load())
				if total >= c.config.MaxPages {
					return nil
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				entry, ok := c.frontier.TryPop()
				if !ok {
					// No work available — check if other workers are active
					if active.Load() == 0 {
						return nil
					}
					// Brief sleep before retry
					time.Sleep(50 * time.Millisecond)
					continue
				}

				active.Add(1)
				c.processURL(ctx, entry)
				active.Add(-1)

				// Save state periodically
				if c.config.StateFile != "" {
					total := int(c.pagesSuccess.Load()) + int(c.pagesFailed.Load())
					if total%10 == 0 {
						c.saveState()
					}
				}
			}
		})
	}

	err := g.Wait()
	c.frontier.Close()

	// Final state save
	if c.config.StateFile != "" {
		c.saveState()
	}

	stats := c.Stats()

	if err != nil && ctx.Err() != nil {
		return stats, ctx.Err()
	}
	return stats, nil
}

func (c *Crawler) processURL(ctx context.Context, entry URLEntry) {
	// Check depth
	if entry.Depth > c.config.MaxDepth {
		c.pagesSkipped.Add(1)
		return
	}

	// Check scope
	if c.startURL != "" && !IsSameScope(c.startURL, entry.URL, c.config.Scope) {
		c.pagesSkipped.Add(1)
		return
	}

	// Check include/exclude globs
	if len(c.config.IncludeGlobs) > 0 && !MatchesGlobs(entry.URL, c.config.IncludeGlobs) {
		c.pagesSkipped.Add(1)
		return
	}
	if len(c.config.ExcludeGlobs) > 0 && MatchesGlobs(entry.URL, c.config.ExcludeGlobs) {
		c.pagesSkipped.Add(1)
		return
	}

	// Check robots.txt
	if c.config.RespectRobots && !c.robots.IsAllowed(entry.URL) {
		c.pagesSkipped.Add(1)
		return
	}

	// Per-domain rate limiting
	domain := DomainOf(entry.URL)
	c.frontier.WaitForDomain(domain)

	// Honor robots.txt crawl-delay
	if c.config.RespectRobots {
		if delay := c.robots.GetCrawlDelay(entry.URL); delay > c.config.Delay {
			c.frontier.SetDomainDelay(domain, delay)
		}
	}

	// Fetch
	start := time.Now()
	result := c.fetch(ctx, entry)
	result.FetchTimeMs = time.Since(start).Milliseconds()
	result.CrawledAt = time.Now()
	result.Depth = entry.Depth

	if result.Error != nil {
		c.pagesFailed.Add(1)
		return
	}

	c.pagesSuccess.Add(1)

	// Enqueue child links
	for _, link := range result.Links {
		if !IsValidCrawlURL(link) {
			continue
		}
		c.frontier.Push(URLEntry{
			URL:      link,
			Depth:    entry.Depth + 1,
			Priority: entry.Depth + 1,
		})
	}

	// Call result callback
	if c.onResult != nil {
		c.onResult(result)
	}

	// Call progress callback
	if c.onProgress != nil {
		c.onProgress(c.Stats())
	}
}

func (c *Crawler) fetch(ctx context.Context, entry URLEntry) CrawlResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.URL, nil)
	if err != nil {
		return CrawlResult{URL: entry.URL, Error: err}
	}
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := c.client.Do(req)
	if err != nil {
		return CrawlResult{URL: entry.URL, Error: err}
	}
	defer resp.Body.Close()

	result := CrawlResult{
		URL:        entry.URL,
		Domain:     DomainOf(entry.URL),
		StatusCode: resp.StatusCode,
	}

	if resp.StatusCode != 200 {
		result.Error = fmt.Errorf("HTTP %d", resp.StatusCode)
		return result
	}

	contentType := resp.Header.Get("Content-Type")
	result.ContentType = contentType
	if !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml") {
		result.Error = fmt.Errorf("non-HTML content type: %s", contentType)
		return result
	}

	// Limit body size to 10MB
	body := io.LimitReader(resp.Body, 10*1024*1024)

	// Extract content using streaming tokenizer
	extracted := Extract(body, entry.URL)

	result.Title = extracted.Title
	result.Description = extracted.Description
	result.Content = extracted.Content
	result.Language = extracted.Language
	result.Links = extracted.Links
	result.Metadata = extracted.Metadata

	if resp.ContentLength > 0 {
		c.bytesTotal.Add(resp.ContentLength)
	}

	return result
}

func (c *Crawler) saveState() {
	state := &CrawlState{
		StartURL:  c.startURL,
		StartedAt: c.startTime,
		Stats:     c.Stats(),
		Visited:   c.frontier.VisitedURLs(),
		Pending:   c.frontier.PendingEntries(),
	}
	SaveState(c.config.StateFile, state)
}

func (c *Crawler) restoreState(state *CrawlState) {
	// Mark visited URLs
	for _, u := range state.Visited {
		c.frontier.Push(URLEntry{URL: u})
		c.frontier.TryPop() // Pop to mark as visited without adding to queue
	}

	// Actually: we need to mark as visited without enqueuing
	// Push already marks as visited, so just push. The TryPop above removes from queue.

	// Re-enqueue pending
	for _, entry := range state.Pending {
		// Reset visited for pending so they can be re-pushed
		c.frontier.mu.Lock()
		normalized, _ := NormalizeURL(entry.URL)
		delete(c.frontier.visited, normalized)
		c.frontier.mu.Unlock()
		c.frontier.Push(entry)
	}

	// Restore stats
	c.pagesSuccess.Store(int64(state.Stats.PagesSuccess))
	c.pagesFailed.Store(int64(state.Stats.PagesFailed))
	c.pagesSkipped.Store(int64(state.Stats.PagesSkipped))
	c.bytesTotal.Store(state.Stats.BytesTotal)
}

// GetStateInfo returns current state info for the status command.
func GetStateInfo(stateFile string) (*CrawlState, error) {
	return LoadState(stateFile)
}

// VisitedSet is a simple thread-safe set of visited URLs (exported for testing).
type VisitedSet struct {
	mu      sync.RWMutex
	visited map[string]bool
}

// NewVisitedSet creates a new VisitedSet.
func NewVisitedSet() *VisitedSet {
	return &VisitedSet{visited: make(map[string]bool)}
}

// Add marks a URL as visited. Returns true if it was new.
func (v *VisitedSet) Add(rawURL string) bool {
	normalized, err := NormalizeURL(rawURL)
	if err != nil {
		return false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.visited[normalized] {
		return false
	}
	v.visited[normalized] = true
	return true
}

// Contains checks if a URL has been visited.
func (v *VisitedSet) Contains(rawURL string) bool {
	normalized, err := NormalizeURL(rawURL)
	if err != nil {
		return false
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.visited[normalized]
}

// Len returns the number of visited URLs.
func (v *VisitedSet) Len() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.visited)
}
