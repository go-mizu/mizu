package dcrawler

import (
	"bufio"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DataDog/zstd"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cespare/xxhash/v2"
	_ "github.com/duckdb/duckdb-go/v2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// Crawler is a high-throughput single-domain web crawler.
type Crawler struct {
	config        Config
	clients       []*http.Client
	frontier      *Frontier
	resultDB      *ResultDB
	stateDB       *StateDB
	stats         *Stats
	robots        *RobotsChecker
	limiter       *rate.Limiter
	retryQ        *retryQueue
	retryAttempts sync.Map // URL -> int: per-URL retry attempt counter
	rodPool       *rodPool // browser page pool (nil if not in rod mode)
	urlClasses    sync.Map // string → *urlClassStats: adaptive block rate per URL class
}

// urlClassStats tracks blocked vs total counts for a URL "class"
// (e.g., all URLs with the same path prefix + type character).
type urlClassStats struct {
	blocked atomic.Int64
	total   atomic.Int64
}

// urlClass extracts a classification key from a URL path for adaptive filtering.
// Groups URLs by their structural pattern: path prefix + the first letter after
// a date-like digit run (8+ consecutive digits). This captures patterns like
// QQ's `/rain/a/YYYYMMDDV...` (video, usually blocked) vs `/rain/a/YYYYMMDDA...`
// (article, usually works).
//
// Returns empty string if no classifiable pattern is found.
func urlClass(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	path := u.Path
	// Find runs of 8+ consecutive digits followed by a letter
	digits := 0
	for i := 0; i < len(path); i++ {
		if path[i] >= '0' && path[i] <= '9' {
			digits++
		} else {
			if digits >= 8 && path[i] >= 'A' && path[i] <= 'Z' {
				// Found: e.g., "/rain/a/20260218V..." → key = "/rain/a/V"
				return path[:i-digits] + string(path[i])
			}
			digits = 0
		}
	}
	return ""
}

// isURLClassBlocked checks if a URL belongs to a class with high block rate.
// Returns true if the class has >85% block rate with >20 samples.
func (c *Crawler) isURLClassBlocked(rawURL string) bool {
	key := urlClass(rawURL)
	if key == "" {
		return false
	}
	if v, ok := c.urlClasses.Load(key); ok {
		stats := v.(*urlClassStats)
		total := stats.total.Load()
		blocked := stats.blocked.Load()
		if total >= 20 && float64(blocked)/float64(total) > 0.85 {
			return true
		}
	}
	return false
}

// recordURLClass updates the URL class stats for adaptive filtering.
func (c *Crawler) recordURLClass(rawURL string, wasBlocked bool) {
	key := urlClass(rawURL)
	if key == "" {
		return
	}
	v, _ := c.urlClasses.LoadOrStore(key, &urlClassStats{})
	stats := v.(*urlClassStats)
	stats.total.Add(1)
	if wasBlocked {
		stats.blocked.Add(1)
	}
}

// retryItem tracks a URL that needs to be retried after a transient error.
type retryItem struct {
	item     CrawlItem
	attempts int
	nextAt   time.Time
}

// retryQueue holds URLs waiting for retry with exponential backoff.
type retryQueue struct {
	mu    sync.Mutex
	items []retryItem
}

func newRetryQueue() *retryQueue {
	return &retryQueue{}
}

func (rq *retryQueue) add(item CrawlItem, attempts int, delay time.Duration) {
	rq.mu.Lock()
	rq.items = append(rq.items, retryItem{
		item:     item,
		attempts: attempts,
		nextAt:   time.Now().Add(delay),
	})
	rq.mu.Unlock()
}

func (rq *retryQueue) len() int {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	return len(rq.items)
}

// drain returns all ready items and removes them from the queue.
func (rq *retryQueue) drain() []retryItem {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	now := time.Now()
	var ready []retryItem
	var remaining []retryItem
	for _, ri := range rq.items {
		if now.After(ri.nextAt) {
			ready = append(ready, ri)
		} else {
			remaining = append(remaining, ri)
		}
	}
	rq.items = remaining
	return ready
}

const maxRetryAttempts = 5

// New creates a new Crawler with the given config.
func New(cfg Config) (*Crawler, error) {
	cfg.Domain = NormalizeDomain(cfg.Domain)
	if cfg.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	d := DefaultConfig()
	if cfg.Workers <= 0 {
		cfg.Workers = d.Workers
	}
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = d.MaxConns
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = d.MaxIdleConns
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = d.Timeout
	}
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = d.MaxBodySize
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = d.UserAgent
	}
	if cfg.DataDir == "" {
		cfg.DataDir = d.DataDir
	}
	if cfg.ShardCount <= 0 {
		cfg.ShardCount = d.ShardCount
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = d.BatchSize
	}
	if cfg.FrontierSize <= 0 {
		cfg.FrontierSize = d.FrontierSize
	}
	if cfg.BloomCapacity <= 0 {
		cfg.BloomCapacity = d.BloomCapacity
	}
	if cfg.BloomFPR <= 0 {
		cfg.BloomFPR = d.BloomFPR
	}
	if cfg.TransportShards <= 0 {
		cfg.TransportShards = d.TransportShards
	}
	if len(cfg.SeedURLs) == 0 {
		cfg.SeedURLs = []string{fmt.Sprintf("https://%s/", cfg.Domain)}
	}

	c := &Crawler{config: cfg}
	c.setupTransport()
	c.frontier = NewFrontier(cfg.Domain, cfg.FrontierSize, cfg.BloomCapacity, cfg.BloomFPR, cfg.IncludeSubdomain, cfg.DomainAliases)
	c.stats = NewStats(cfg.Domain, cfg.MaxPages, cfg.Continuous)
	c.stats.SetFrontierFuncs(c.frontier.Len, c.frontier.BloomCount)
	c.stats.SetUseRod(cfg.UseRod || cfg.UseLightpanda)
	c.retryQ = newRetryQueue()
	c.stats.SetRetryQLen(c.retryQ.len)

	if cfg.RateLimit > 0 {
		c.limiter = rate.NewLimiter(rate.Limit(cfg.RateLimit), max(cfg.RateLimit/10, 1))
	}
	return c, nil
}

func (c *Crawler) setupTransport() {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}

	var cachedIPs []string
	for _, host := range []string{c.config.Domain, "www." + c.config.Domain} {
		if ips, err := net.LookupHost(host); err == nil && len(ips) > 0 {
			// Filter to IPv4 only: IPv6 connectivity is unreliable for crawling,
			// and Go's Happy Eyeballs (RFC 6555) doesn't apply when using cached IPs.
			for _, ip := range ips {
				if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() != nil {
					cachedIPs = append(cachedIPs, ip)
				}
			}
			if len(cachedIPs) > 0 {
				break
			}
			// Fall back to all IPs if no IPv4 found
			cachedIPs = ips
			break
		}
	}
	var ipIdx atomic.Uint64

	shards := c.config.TransportShards
	connsPerShard := max(c.config.MaxConns/shards, 1)
	idlePerShard := max(c.config.MaxIdleConns/shards, 1)

	c.clients = make([]*http.Client, shards)
	for i := range shards {
		t := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if len(cachedIPs) > 0 {
					_, port, _ := net.SplitHostPort(addr)
					ip := cachedIPs[ipIdx.Add(1)%uint64(len(cachedIPs))]
					return dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
				}
				return dialer.DialContext(ctx, network, addr)
			},
			ForceAttemptHTTP2:     !c.config.ForceHTTP1,
			MaxIdleConnsPerHost:   idlePerShard,
			MaxConnsPerHost:       connsPerShard,
			MaxIdleConns:          idlePerShard,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: c.config.Timeout,
			WriteBufferSize:       4096,
			ReadBufferSize:        32768,
			DisableCompression:    true,
		}
		c.clients[i] = &http.Client{
			Transport: t,
			Timeout:   c.config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}
	}
}

func (c *Crawler) clientForWorker(workerID int) *http.Client {
	return c.clients[workerID%len(c.clients)]
}

// Run executes the crawl. Blocks until frontier drains or MaxPages reached.
func (c *Crawler) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// robots.txt: always fetch for Sitemap directives (URL discovery).
	// In browser mode, skip path-blocking rules (Disallow) but still use Sitemap: directives.
	// Many sites (e.g., news.qq.com) have sitemaps at non-standard paths only listed in robots.txt.
	{
		rctx, rc := context.WithTimeout(ctx, 10*time.Second)
		if r, _ := FetchRobots(rctx, c.clients[0], c.config.Domain); r != nil {
			c.robots = r
			// Only enforce path-blocking rules for non-browser mode
			if c.config.RespectRobots && !c.config.UseRod && !c.config.UseLightpanda {
				c.frontier.SetRobots(r)
			}
		}
		rc()
	}

	// State DB
	sdb, err := OpenStateDB(c.config.DomainDir())
	if err != nil {
		return fmt.Errorf("state db: %w", err)
	}
	c.stateDB = sdb
	defer sdb.Close()

	// Resume: restore bloom from already-crawled URLs and re-feed pending links
	if c.config.Resume {
		c.restoreState()
	}

	// Result DB
	rdb, err := NewResultDB(c.config.ResultDir(), c.config.ShardCount, c.config.BatchSize)
	if err != nil {
		return fmt.Errorf("result db: %w", err)
	}
	c.resultDB = rdb
	defer rdb.Close()

	sdb.SetMeta("domain", c.config.Domain)
	sdb.SetMeta("status", "running")
	sdb.SetMeta("start_time", time.Now().UTC().Format(time.RFC3339))
	rdb.SetMeta("domain", c.config.Domain)

	// Seed loading (priority order):
	// 1. --seed-file: load URLs from text file
	// 2. State DB frontier: auto-load saved frontier entries
	// 3. Fallback: config SeedURLs (domain root)
	c.loadSeeds()

	// Sitemap discovery: use sitemap.xml as additional seed source.
	// Uses robots.txt Sitemap: directives (fetched above) for non-standard paths.
	if c.config.FollowSitemap {
		var robotsSitemaps []string
		if c.robots != nil {
			robotsSitemaps = c.robots.Sitemaps()
		}
		// 2 minutes timeout: large sitemap indexes (1000+ child sitemaps) need time
		// for concurrent fetching of individual sitemaps.
		sctx, sc := context.WithTimeout(ctx, 2*time.Minute)
		sitemapURLs, _ := DiscoverSitemapURLs(sctx, c.clients[0], c.config.Domain, robotsSitemaps, 100_000)
		sc()
		if len(sitemapURLs) > 0 {
			added := 0
			for _, u := range sitemapURLs {
				if c.frontier.TryAdd(u, 1) {
					added++
				}
			}
			if added > 0 {
				c.logInit("Sitemap: %s URLs added to frontier", fmtInt(added))
			}
		}
	}

	c.logInit("Frontier: %s seed URLs", fmtInt(c.frontier.Len()))

	// errgroup: workers + coordinator
	g, gctx := errgroup.WithContext(ctx)

	if c.config.UseLightpanda {
		lp, err := newLightpandaPool(c.config)
		if err != nil {
			return fmt.Errorf("lightpanda: %w", err)
		}
		defer lp.close()

		workers := c.config.RodWorkers
		if workers <= 0 {
			workers = 8
		}
		c.stats.SetRodTotalWorkers(workers)
		for i := range workers {
			workerID := i
			g.Go(func() error {
				c.lightpandaWorker(gctx, lp, workerID)
				return nil
			})
		}
	} else if c.config.UseRod {
		rp, err := newRodPool(c.config)
		if err != nil {
			return fmt.Errorf("rod: %w", err)
		}
		defer rp.close()
		c.rodPool = rp

		workers := c.config.RodWorkers
		if workers <= 0 {
			workers = 40
		}
		c.stats.SetRodTotalWorkers(workers)
		for i := range workers {
			workerID := i
			g.Go(func() error {
				c.rodWorker(gctx, rp, workerID)
				return nil
			})
		}
	} else {
		for i := range c.config.Workers {
			client := c.clientForWorker(i)
			g.Go(func() error {
				c.worker(gctx, client)
				return nil
			})
		}
	}

	g.Go(func() error {
		c.coordinator(gctx, cancel)
		return nil
	})

	// Retry feeder: re-feeds timed-out items back to frontier
	g.Go(func() error {
		c.retryFeeder(gctx)
		return nil
	})

	g.Wait()

	c.saveState()
	rdb.SetMeta("end_time", time.Now().UTC().Format(time.RFC3339))
	rdb.SetMeta("total_pages", fmt.Sprintf("%d", c.stats.Done()))
	return nil
}

func (c *Crawler) restoreState() {
	dir := c.config.ResultDir()

	// Phase 1: Mark successfully-crawled URLs as seen in bloom (skip errors)
	crawled := c.restoreFromShards(dir,
		"SELECT url FROM pages WHERE status_code >= 200 AND status_code < 400",
		c.frontier.MarkSeen)
	if crawled > 0 {
		c.logInit("Resume: %s crawled URLs in bloom", fmtInt(crawled))
	}

	// Phase 2: Re-feed discovered-but-uncrawled internal links into frontier.
	// These are links extracted from crawled pages that were never fetched
	// (either due to frontier overflow, shutdown, or channel-full drops).
	var pendingAdded int
	c.restoreFromShards(dir,
		"SELECT DISTINCT target_url FROM links WHERE is_internal = true AND target_url NOT IN (SELECT url FROM pages)",
		func(u string) {
			if c.frontier.TryAdd(u, 1) {
				pendingAdded++
			}
		},
	)
	if pendingAdded > 0 {
		c.logInit("Resume: %s pending links re-fed to frontier", fmtInt(pendingAdded))
	}

	// Phase 3: Re-attempt ALL previously failed URLs.
	// Browser errors (rod pool, page info, get html, deadline exceeded) are almost always
	// transient — Chrome tab state, timing, resource pressure. Retry them all.
	// Permanent errors (DNS, SSL) will fail again quickly and won't waste much time.
	var retryAdded int
	c.restoreFromShards(dir,
		"SELECT url FROM pages WHERE error != ''",
		func(u string) {
			if c.frontier.TryAdd(u, 1) {
				retryAdded++
			}
		},
	)
	if retryAdded > 0 {
		c.logInit("Resume: %s failed URLs queued for retry", fmtInt(retryAdded))
	}

	// Phase 4: Delete error-only rows for URLs being retried.
	// This prevents stale error entries from accumulating across resume runs.
	// When the URL is re-fetched, INSERT OR REPLACE will write a fresh result.
	if retryAdded > 0 {
		deleted := c.deleteErrorRows(dir)
		if deleted > 0 {
			c.logInit("Resume: %s stale error rows cleaned", fmtInt(deleted))
		}
	}

	// Phase 5: Re-crawl stale pages (incremental crawling).
	// If StaleHours > 0, pages older than N hours are re-fed to the frontier.
	if c.config.StaleHours > 0 {
		cutoff := time.Now().Add(-time.Duration(c.config.StaleHours) * time.Hour).UTC().Format(time.RFC3339)
		var staleAdded int
		c.restoreFromShards(dir,
			fmt.Sprintf("SELECT url FROM pages WHERE status_code >= 200 AND status_code < 400 AND crawled_at < '%s'", cutoff),
			func(u string) {
				if c.frontier.TryAdd(u, 1) {
					staleAdded++
				}
			},
		)
		if staleAdded > 0 {
			c.logInit("Resume: %s stale pages queued for re-crawl (>%dh old)", fmtInt(staleAdded), c.config.StaleHours)
		}
	}
}

// deleteErrorRows removes error-only rows from result shards so they don't accumulate.
// Only deletes rows that have error != '' — successful rows are preserved.
func (c *Crawler) deleteErrorRows(dir string) int {
	count := 0
	for i := range c.config.ShardCount {
		path := fmt.Sprintf("%s/results_%03d.duckdb", dir, i)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		db, err := sql.Open("duckdb", path)
		if err != nil {
			continue
		}
		result, err := db.Exec("DELETE FROM pages WHERE error != ''")
		if err == nil {
			if n, _ := result.RowsAffected(); n > 0 {
				count += int(n)
			}
		}
		db.Close()
	}
	return count
}

// restoreFromShards opens each result shard read-only and runs a query,
// calling fn for each row's first VARCHAR column. Returns total count.
func (c *Crawler) restoreFromShards(dir, query string, fn func(string)) int {
	count := 0
	for i := range c.config.ShardCount {
		path := fmt.Sprintf("%s/results_%03d.duckdb", dir, i)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
		if err != nil {
			continue
		}
		rows, err := db.Query(query)
		if err != nil {
			db.Close()
			continue
		}
		for rows.Next() {
			var u string
			if rows.Scan(&u) == nil {
				fn(u)
				count++
			}
		}
		rows.Close()
		db.Close()
	}
	return count
}

// loadSeeds populates the frontier with seed URLs in priority order:
// 1. Seed file (--seed-file)
// 2. State DB frontier (auto-load saved entries)
// 3. Fallback: config SeedURLs (domain root)
func (c *Crawler) loadSeeds() {
	// Priority 1: seed file
	if c.config.SeedFile != "" {
		n := c.loadSeedFile(c.config.SeedFile)
		if n > 0 {
			c.logInit("Seeds: %s URLs from file %s", fmtInt(n), c.config.SeedFile)
			return
		}
	}

	// Priority 2: state DB frontier (independent of --resume flag)
	if c.stateDB != nil {
		items, _ := c.stateDB.LoadFrontier()
		if len(items) > 0 {
			n := 0
			for _, item := range items {
				if c.frontier.PushDirect(item) {
					n++
				}
			}
			if n > 0 {
				c.logInit("Seeds: %s URLs from state DB frontier", fmtInt(n))
				return
			}
		}
	}

	// Priority 3: fallback to config seed URLs
	for _, u := range c.config.SeedURLs {
		c.frontier.TryAdd(u, 0)
	}
}

func (c *Crawler) loadSeedFile(path string) int {
	f, err := os.Open(path)
	if err != nil {
		c.logInit("Warning: cannot open seed file: %v", err)
		return 0
	}
	defer f.Close()

	n := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		c.frontier.TryAdd(line, 0)
		n++
	}
	return n
}

func (c *Crawler) saveState() {
	if c.stateDB == nil {
		return
	}
	items := c.frontier.Drain()
	if len(items) > 0 {
		c.stateDB.SaveFrontier(items)
	}
	c.stateDB.SetMeta("status", "stopped")
	c.stateDB.SetMeta("end_time", time.Now().UTC().Format(time.RFC3339))
	c.stateDB.SetMeta("pages_crawled", fmt.Sprintf("%d", c.stats.Done()))
	c.stateDB.SetMeta("pages_ok", fmt.Sprintf("%d", c.stats.success.Load()))
	c.stateDB.SetMeta("total_bytes", fmt.Sprintf("%d", c.stats.bytes.Load()))
}

// coordinator watches for crawl completion: max-pages or frontier drained.
// Calls cancel() to signal all workers to stop, then returns.
//
// Browser watchdog: detects throughput stalls (0 pages/s for >60s with
// in-flight workers) and force-restarts Chrome. Also restarts Chrome
// every 2000 pages to prevent memory bloat.
func (c *Crawler) coordinator(ctx context.Context, cancel context.CancelFunc) {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	empty := 0
	var lastReseed time.Time
	reseedInterval := c.config.ReseedInterval
	if reseedInterval <= 0 {
		reseedInterval = 30 * time.Second
	}
	var geoBlockWarned bool // only warn once about potential geo-blocking

	// Browser watchdog state
	lastDone := c.stats.Done()
	lastDoneAt := time.Now()
	var lastRestartDone int64 // pages at last periodic restart
	const stallTimeout = 60 * time.Second   // restart after 60s of zero progress
	const restartEvery = int64(2000)         // restart every 2000 pages for memory

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		if c.config.MaxPages > 0 && c.stats.success.Load() >= int64(c.config.MaxPages) {
			cancel()
			return
		}

		// Browser watchdog: detect stalls and periodic restart
		done := c.stats.Done()
		if c.rodPool != nil {
			// Throughput stall detection: no pages processed for >60s with workers in-flight.
			// This catches Chrome hangs, memory exhaustion, and stuck tabs.
			if done != lastDone {
				lastDone = done
				lastDoneAt = time.Now()
			} else if c.stats.inFlight.Load() > 0 && time.Since(lastDoneAt) > stallTimeout {
				c.rodPool.tryRestart()
				c.stats.rodRestarts.Add(1)
				lastDone = done
				lastDoneAt = time.Now()
			}

			// Periodic restart: every 2000 pages to clear Chrome memory.
			// Chrome accumulates memory from page history, JS heaps, and DOM.
			if done-lastRestartDone >= restartEvery {
				c.rodPool.tryRestart()
				c.stats.rodRestarts.Add(1)
				lastRestartDone = done
				lastDone = done
				lastDoneAt = time.Now()
			}
		}

		// Geo-blocking / total WAF detection: if ALL pages so far are blocked
		// and no links were found, the site is likely geo-blocked from this IP.
		// Warn the user early instead of silently exiting with 0 ok pages.
		if !geoBlockWarned && done > 0 && c.stats.success.Load() == 0 && c.stats.linksFound.Load() == 0 {
			blk := c.stats.Blocked()
			if blk > 0 && blk == done {
				geoBlockWarned = true
			}
		}

		if c.frontier.Len() == 0 && c.stats.inFlight.Load() == 0 && c.retryQ.len() == 0 {
			empty++
			if c.config.Continuous && empty >= 15 {
				// Re-seed: fetch new URLs from sitemap + homepage
				if time.Since(lastReseed) >= reseedInterval {
					n := c.reseed(ctx)
					lastReseed = time.Now()
					if n > 0 {
						c.stats.reseeds.Add(1)
						empty = 0
						continue
					}
				}
				// No new URLs found, keep waiting (only stop on Ctrl+C)
				empty = 15 // stay at threshold, re-check next tick
				continue
			}
			if !c.config.Continuous && empty >= 15 { // 3s sustained
				cancel()
				return
			}
		} else {
			empty = 0
		}
	}
}

// reseed discovers new URLs from sitemap and homepage. Returns count of new URLs added.
func (c *Crawler) reseed(ctx context.Context) int {
	added := 0

	// Browser mode: re-insert homepage into frontier for browser-based re-crawl.
	// This triggers fresh XHR API calls that discover newly published articles.
	// The bloom filter is bypassed for the homepage since it was already crawled.
	if c.config.UseRod || c.config.UseLightpanda {
		homeURL := fmt.Sprintf("https://%s/", NormalizeDomain(c.config.Domain))
		if c.frontier.PushDirect(CrawlItem{URL: homeURL, Depth: 0}) {
			added++
		}
		return added
	}

	// Re-fetch sitemap for new URLs
	if c.config.FollowSitemap {
		var robotsSitemaps []string
		if c.robots != nil {
			robotsSitemaps = c.robots.Sitemaps()
		}
		sctx, sc := context.WithTimeout(ctx, 30*time.Second)
		urls, _ := DiscoverSitemapURLs(sctx, c.clients[0], c.config.Domain, robotsSitemaps, 1_000_000)
		sc()
		for _, u := range urls {
			if c.frontier.TryAdd(u, 1) {
				added++
			}
		}
	}

	// Re-add homepage (may have new links)
	homeURL := fmt.Sprintf("https://%s/", c.config.Domain)
	// Fetch homepage and extract links directly (bypass bloom for the homepage itself)
	hctx, hc := context.WithTimeout(ctx, 10*time.Second)
	defer hc()
	links := c.fetchLinksFrom(hctx, homeURL)
	for _, link := range links {
		if c.frontier.TryAdd(link, 1) {
			added++
		}
	}

	return added
}

// fetchLinksFrom fetches a page and returns discovered internal URLs.
func (c *Crawler) fetchLinksFrom(ctx context.Context, pageURL string) []string {
	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := c.clients[0].Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		if gr, e := gzip.NewReader(resp.Body); e == nil {
			reader = gr
			defer gr.Close()
		}
	}
	body, _ := io.ReadAll(io.LimitReader(reader, c.config.MaxBodySize))
	if len(body) == 0 {
		return nil
	}

	baseURL := resp.Request.URL
	if baseURL == nil {
		baseURL, _ = url.Parse(pageURL)
	}
	meta := ExtractLinksAndMeta(body, baseURL, c.config.Domain, c.config.ExtractImages)

	var urls []string
	for _, link := range meta.Links {
		if link.IsInternal {
			urls = append(urls, link.TargetURL)
		}
	}
	return urls
}

// worker pulls from the frontier until ctx is cancelled.
func (c *Crawler) worker(ctx context.Context, client *http.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-c.frontier.ch:
			if c.limiter != nil {
				if err := c.limiter.Wait(ctx); err != nil {
					return
				}
			}
			c.fetchAndProcess(ctx, client, item)
		}
	}
}

func (c *Crawler) fetchAndProcess(ctx context.Context, client *http.Client, item CrawlItem) {
	if c.config.MaxPages > 0 && c.stats.success.Load() >= int64(c.config.MaxPages) {
		return
	}
	// Adaptive filter: skip URLs from classes with >85% block rate
	if c.isURLClassBlocked(item.URL) {
		c.stats.RecordSkipped()
		return
	}
	c.stats.inFlight.Add(1)
	defer c.stats.inFlight.Add(-1)

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", item.URL, nil)
	if err != nil {
		c.recordError(item, err, 0)
		return
	}
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	fetchMs := time.Since(start).Milliseconds()
	if err != nil {
		if ctx.Err() != nil {
			return // context cancelled, don't record
		}
		c.recordError(item, err, fetchMs)
		return
	}
	defer resp.Body.Close()

	// Rate-limited or server error: enqueue for retry
	if isRetryableStatus(resp.StatusCode) {
		io.Copy(io.Discard, resp.Body)
		c.stats.RecordFailure(resp.StatusCode, false)
		c.stats.RecordDepth(item.Depth)
		c.stats.fetchMs.Add(fetchMs)
		c.enqueueRetry(item, resp.StatusCode)
		return
	}

	// HTTP 403 from CDN/WAF (CloudFront, Akamai) — record as blocked, not error.
	// Check for WAF signatures in the response to distinguish from app-level 403.
	if resp.StatusCode == 403 {
		server := strings.ToLower(resp.Header.Get("Server"))
		if server == "cloudfront" || server == "akamaighost" || strings.Contains(server, "awselb") {
			io.Copy(io.Discard, resp.Body)
			c.recordBlocked(item, fmt.Sprintf("HTTP 403 from %s (WAF/geo-block)", resp.Header.Get("Server")), fetchMs)
			return
		}
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		if gr, e := gzip.NewReader(resp.Body); e == nil {
			reader = gr
			defer gr.Close()
		}
	}
	body, _ := io.ReadAll(io.LimitReader(reader, c.config.MaxBodySize))

	result := Result{
		URL: item.URL, URLHash: xxhash.Sum64String(item.URL),
		Depth: item.Depth, StatusCode: resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
		BodyHash: xxhash.Sum64(body),
		ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified"),
		Server: resp.Header.Get("Server"), FetchTimeMs: fetchMs,
		CrawledAt: time.Now(),
	}
	if resp.Request != nil && resp.Request.URL.String() != item.URL {
		result.RedirectURL = resp.Request.URL.String()
	}

	if isHTML(result.ContentType) && resp.StatusCode >= 200 && resp.StatusCode < 400 && len(body) > 0 {
		baseURL := resp.Request.URL
		if baseURL == nil {
			baseURL, _ = url.Parse(item.URL)
		}
		meta := ExtractLinksAndMeta(body, baseURL, c.config.Domain, c.config.ExtractImages)
		result.Title = meta.Title
		result.Description = meta.Description
		result.Language = meta.Language
		result.Canonical = meta.Canonical

		// Detect soft 404 / anti-bot pages before extracting links.
		if blocked, reason := isBlockedPage(body, meta.Title, item.URL); blocked {
			c.recordBlocked(item, reason, fetchMs)
			return
		}

		result.LinkCount = len(meta.Links)
		c.stats.RecordLinks(len(meta.Links))

		if c.config.StoreBody {
			if compressed, err := zstd.Compress(nil, body); err == nil {
				result.BodyCompressed = compressed
			}
		}
		hasAliases := len(c.config.DomainAliases) > 0
		if c.config.MaxDepth == 0 || item.Depth < c.config.MaxDepth {
			for _, link := range meta.Links {
				if link.IsInternal || hasAliases {
					c.frontier.TryAdd(link.TargetURL, item.Depth+1)
				}
			}
		}
		if c.config.StoreLinks && len(meta.Links) > 0 {
			c.resultDB.AddLinks(result.URLHash, meta.Links)
		}
	}

	c.resultDB.AddPage(result)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		c.stats.RecordSuccess(result.StatusCode, int64(len(body)), fetchMs)
		c.recordURLClass(item.URL, false)
	} else {
		c.stats.RecordFailure(result.StatusCode, false)
	}
	c.stats.RecordDepth(item.Depth)
}

func (c *Crawler) recordBlocked(item CrawlItem, reason string, fetchMs int64) {
	c.stats.RecordBlocked()
	c.stats.RecordDepth(item.Depth)
	c.recordURLClass(item.URL, true)
	c.stats.fetchMs.Add(fetchMs)
	c.resultDB.AddPage(Result{
		URL: item.URL, URLHash: xxhash.Sum64String(item.URL),
		Depth: item.Depth, FetchTimeMs: fetchMs,
		CrawledAt: time.Now(), Error: "blocked: " + reason,
	})
}

func (c *Crawler) recordError(item CrawlItem, err error, fetchMs int64) {
	isTimeout := isTimeoutError(err)
	c.stats.RecordFailure(0, isTimeout)
	c.stats.RecordDepth(item.Depth)
	c.stats.fetchMs.Add(fetchMs)
	// Retry transient errors (timeouts, browser glitches, render deadline).
	// Permanent errors (DNS, SSL, connection refused) are NOT retried.
	if isTimeout || isRetryableError(err) {
		c.enqueueRetry(item, 0)
	}
	c.resultDB.AddPage(Result{
		URL: item.URL, URLHash: xxhash.Sum64String(item.URL),
		Depth: item.Depth, FetchTimeMs: fetchMs,
		CrawledAt: time.Now(), Error: err.Error(),
	})
}

// isRetryableError returns true for transient browser/network errors that may
// succeed on retry. Permanent errors (DNS, SSL, connection refused) return false.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()

	// Permanent navigation errors — never retry these
	permanentPrefixes := []string{
		"net::ERR_NAME_NOT_RESOLVED",       // DNS failure
		"net::ERR_CONNECTION_REFUSED",      // server not listening
		"net::ERR_CERT_",                   // any SSL cert error
		"net::ERR_SSL_",                    // SSL protocol errors
		"net::ERR_BLOCKED_BY_RESPONSE",     // CORS/CSP blocked
		"net::ERR_ABORTED",                 // intentionally cancelled
		"net::ERR_INVALID_URL",             // malformed URL
		"net::ERR_TOO_MANY_REDIRECTS",      // redirect loop
	}
	for _, prefix := range permanentPrefixes {
		if strings.Contains(s, prefix) {
			return false
		}
	}

	// Transient errors — retry these
	return strings.Contains(s, "rod pool:") ||
		strings.Contains(s, "page info:") ||
		strings.Contains(s, "get html:") ||
		strings.Contains(s, "deadline exceeded") ||
		strings.Contains(s, "timeout waiting for DOM") ||
		strings.Contains(s, "navigate:") || // net::ERR_CONNECTION_RESET, etc.
		strings.Contains(s, "context deadline exceeded") ||
		strings.Contains(s, "connection reset")
}

// isRetryableStatus returns true for HTTP status codes that are transient.
func isRetryableStatus(code int) bool {
	return code == 429 || code == 500 || code == 502 || code == 503 || code == 504
}

// enqueueRetry adds a failed URL to the retry queue with exponential backoff.
// Uses a per-URL attempt counter (sync.Map) to prevent infinite retry loops.
func (c *Crawler) enqueueRetry(item CrawlItem, statusCode int) {
	// Increment per-URL attempt counter
	val, _ := c.retryAttempts.LoadOrStore(item.URL, 0)
	attempts := val.(int) + 1
	c.retryAttempts.Store(item.URL, attempts)
	if attempts >= maxRetryAttempts {
		c.stats.retryExhausted.Add(1)
		return // give up after max attempts
	}
	delay := time.Duration(attempts) * 5 * time.Second // 5s, 10s, 15s, 20s
	if statusCode == 429 {
		delay = time.Duration(attempts) * 30 * time.Second // 30s, 60s, 90s, 120s
	}
	c.retryQ.add(item, attempts, delay)
}

// retryFeeder periodically checks the retry queue and re-feeds items to frontier.
func (c *Crawler) retryFeeder(ctx context.Context) {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			ready := c.retryQ.drain()
			for _, ri := range ready {
				// Push directly to frontier, bypassing bloom (URL is already in bloom from first attempt)
				select {
				case c.frontier.ch <- ri.item:
					c.stats.retries.Add(1)
				default:
					// Frontier full, re-enqueue (per-URL attempts tracked in retryAttempts map)
					c.retryQ.add(ri.item, ri.attempts, 15*time.Second)
				}
			}
		}
	}
}

func (c *Crawler) Stats() *Stats      { return c.stats }
func (c *Crawler) ResultDB() *ResultDB { return c.resultDB }
func (c *Crawler) DataDir() string     { return c.config.DomainDir() }

// logInit sends an initialization message to the TUI (if running) or prints to stdout.
func (c *Crawler) logInit(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if c.stats.initLog != nil {
		select {
		case c.stats.initLog <- msg:
		default:
		}
	} else {
		fmt.Printf("  %s\n", msg)
	}
}

// RunWithDisplay runs the crawler with a bubbletea TUI dashboard.
func RunWithDisplay(ctx context.Context, c *Crawler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	m := newTUIModel(c.stats, c.config, cancel)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))

	// Capture init messages from Run() via channel
	c.stats.initLog = make(chan string, 32)

	var crawlErr error
	go func() {
		crawlErr = c.Run(ctx)
		p.Send(phaseMsg("done"))
	}()

	if _, err := p.Run(); err != nil {
		cancel()
		return err
	}
	cancel()

	return crawlErr
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok {
		return ne.Timeout()
	}
	s := err.Error()
	return strings.Contains(s, "timeout") || strings.Contains(s, "deadline")
}
