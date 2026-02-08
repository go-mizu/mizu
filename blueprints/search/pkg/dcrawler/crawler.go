package dcrawler

import (
	"bufio"
	"compress/gzip"
	"context"
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
	"github.com/cespare/xxhash/v2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// Crawler is a high-throughput single-domain web crawler.
type Crawler struct {
	config   Config
	clients  []*http.Client
	frontier *Frontier
	resultDB *ResultDB
	stateDB  *StateDB
	stats    *Stats
	robots   *RobotsChecker
	limiter  *rate.Limiter
}

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
	c.frontier = NewFrontier(cfg.Domain, cfg.FrontierSize, cfg.BloomCapacity, cfg.BloomFPR, cfg.IncludeSubdomain)
	c.stats = NewStats(cfg.Domain, cfg.MaxPages, cfg.Continuous)
	c.stats.SetFrontierFuncs(c.frontier.Len, c.frontier.BloomCount)

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

	// robots.txt
	if c.config.RespectRobots {
		rctx, rc := context.WithTimeout(ctx, 10*time.Second)
		if r, _ := FetchRobots(rctx, c.clients[0], c.config.Domain); r != nil {
			c.robots = r
			c.frontier.SetRobots(r)
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

	// Resume: restore bloom/seen URLs only
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
	fmt.Printf("  Frontier: %s seed URLs\n\n", fmtInt(c.frontier.Len()))

	// errgroup: workers + coordinator
	g, gctx := errgroup.WithContext(ctx)

	for i := range c.config.Workers {
		client := c.clientForWorker(i)
		g.Go(func() error {
			c.worker(gctx, client)
			return nil
		})
	}

	g.Go(func() error {
		c.coordinator(gctx, cancel)
		return nil
	})

	g.Wait()

	c.saveState()
	rdb.SetMeta("end_time", time.Now().UTC().Format(time.RFC3339))
	rdb.SetMeta("total_pages", fmt.Sprintf("%d", c.stats.Done()))
	return nil
}

func (c *Crawler) restoreState() {
	rdb, err := NewResultDB(c.config.ResultDir(), c.config.ShardCount, c.config.BatchSize)
	if err != nil {
		fmt.Printf("  Resume: failed to open result DB: %v\n", err)
		return
	}
	defer rdb.Close()

	// Phase 1: Mark all already-crawled URLs as seen in bloom
	crawled, _ := rdb.LoadExistingURLs(c.frontier.MarkSeen)
	if crawled > 0 {
		fmt.Printf("  Resume: %s crawled URLs in bloom\n", fmtInt(crawled))
	}

	// Phase 2: Re-feed discovered-but-uncrawled internal links into frontier.
	// These are links extracted from crawled pages that were never fetched
	// (either due to frontier overflow, shutdown, or channel-full drops).
	pending, _ := rdb.LoadPendingLinks(c.frontier.TryAdd)
	if pending > 0 {
		fmt.Printf("  Resume: %s pending links re-fed to frontier\n", fmtInt(pending))
	}
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
			fmt.Printf("  Seeds: %s URLs from file %s\n", fmtInt(n), c.config.SeedFile)
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
				fmt.Printf("  Seeds: %s URLs from state DB frontier\n", fmtInt(n))
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
		fmt.Printf("  Warning: cannot open seed file: %v\n", err)
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
		fmt.Printf("\n  State: saved %s frontier URLs for restart\n", fmtInt(len(items)))
	}
	c.stateDB.SetMeta("status", "stopped")
	c.stateDB.SetMeta("end_time", time.Now().UTC().Format(time.RFC3339))
	c.stateDB.SetMeta("pages_crawled", fmt.Sprintf("%d", c.stats.Done()))
	c.stateDB.SetMeta("pages_ok", fmt.Sprintf("%d", c.stats.success.Load()))
	c.stateDB.SetMeta("total_bytes", fmt.Sprintf("%d", c.stats.bytes.Load()))
}

// coordinator watches for crawl completion: max-pages or frontier drained.
// Calls cancel() to signal all workers to stop, then returns.
func (c *Crawler) coordinator(ctx context.Context, cancel context.CancelFunc) {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	empty := 0
	var lastReseed time.Time
	reseedInterval := c.config.ReseedInterval
	if reseedInterval <= 0 {
		reseedInterval = 30 * time.Second
	}
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
		if c.frontier.Len() == 0 && c.stats.inFlight.Load() == 0 {
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
			if c.frontier.TryAdd(u, 0) {
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
	meta := ExtractLinksAndMeta(body, baseURL, c.config.Domain)

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
	c.stats.inFlight.Add(1)
	defer c.stats.inFlight.Add(-1)

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", item.URL, nil)
	if err != nil {
		c.recordError(item, err, 0)
		return
	}
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	fetchMs := time.Since(start).Milliseconds()
	if err != nil {
		c.recordError(item, err, fetchMs)
		return
	}
	defer resp.Body.Close()

	// Rate-limited or server error
	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		io.Copy(io.Discard, resp.Body)
		c.stats.RecordFailure(resp.StatusCode, false)
		c.stats.RecordDepth(item.Depth)
		c.resultDB.AddPage(Result{
			URL: item.URL, URLHash: xxhash.Sum64String(item.URL),
			Depth: item.Depth, StatusCode: resp.StatusCode,
			FetchTimeMs: fetchMs, CrawledAt: time.Now(),
			Error: fmt.Sprintf("HTTP %d", resp.StatusCode),
		})
		return
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
		meta := ExtractLinksAndMeta(body, baseURL, c.config.Domain)
		result.Title = meta.Title
		result.Description = meta.Description
		result.Language = meta.Language
		result.Canonical = meta.Canonical
		result.LinkCount = len(meta.Links)
		c.stats.RecordLinks(len(meta.Links))

		if c.config.StoreBody {
			if compressed, err := zstd.Compress(nil, body); err == nil {
				result.BodyCompressed = compressed
			}
		}
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

	c.resultDB.AddPage(result)
	c.stats.RecordSuccess(result.StatusCode, int64(len(body)), fetchMs)
	c.stats.RecordDepth(item.Depth)
}

func (c *Crawler) recordError(item CrawlItem, err error, fetchMs int64) {
	c.stats.RecordFailure(0, isTimeoutError(err))
	c.stats.RecordDepth(item.Depth)
	c.resultDB.AddPage(Result{
		URL: item.URL, URLHash: xxhash.Sum64String(item.URL),
		Depth: item.Depth, FetchTimeMs: fetchMs,
		CrawledAt: time.Now(), Error: err.Error(),
	})
}

func (c *Crawler) Stats() *Stats      { return c.stats }
func (c *Crawler) ResultDB() *ResultDB { return c.resultDB }
func (c *Crawler) DataDir() string     { return c.config.DomainDir() }

// RunWithDisplay runs the crawler with live terminal progress display.
func RunWithDisplay(ctx context.Context, c *Crawler) error {
	var lines int
	var mu sync.Mutex

	dctx, dcancel := context.WithCancel(ctx)
	defer dcancel()

	go func() {
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-dctx.Done():
				return
			case <-tick.C:
				mu.Lock()
				if lines > 0 {
					fmt.Printf("\033[%dA\033[J", lines)
				}
				out := c.stats.Render()
				fmt.Print(out)
				lines = strings.Count(out, "\n")
				mu.Unlock()
			}
		}
	}()

	err := c.Run(ctx)
	dcancel()
	time.Sleep(50 * time.Millisecond)

	c.stats.Freeze()
	mu.Lock()
	if lines > 0 {
		fmt.Printf("\033[%dA\033[J", lines)
	}
	fmt.Print(c.stats.Render())
	mu.Unlock()
	return err
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
