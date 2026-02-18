package qq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"golang.org/x/time/rate"
)

// Crawler orchestrates the QQ News crawling pipeline.
type Crawler struct {
	config  Config
	client  *Client
	db      *DB
	filter  *bloom.BloomFilter
	limiter *rate.Limiter

	// Stats
	discovered    atomic.Int64
	fetched       atomic.Int64
	succeeded     atomic.Int64
	failed        atomic.Int64
	deleted       atomic.Int64 // 302 → babygohome
	rateLimited   atomic.Int64 // HTTP 567
	skipped       atomic.Int64
	sitemapsDone  atomic.Int64
	sitemapsTotal atomic.Int64
	startTime     time.Time
}

// New creates a new QQ News Crawler.
func New(cfg Config) (*Crawler, error) {
	db, err := OpenDB(cfg.DBPath())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	var lim *rate.Limiter
	if cfg.RateLimit > 0 {
		lim = rate.NewLimiter(rate.Limit(cfg.RateLimit), max(1, int(cfg.RateLimit)))
	}

	c := &Crawler{
		config:  cfg,
		client:  NewClient(cfg),
		db:      db,
		filter:  bloom.NewWithEstimates(BloomCapacity, BloomFPR),
		limiter: lim,
	}

	return c, nil
}

// Run executes the full crawl pipeline.
func (c *Crawler) Run(ctx context.Context) error {
	defer c.db.Close()

	// Phase 0: Restore state if resuming
	if c.config.Resume {
		if err := c.restoreState(); err != nil {
			return fmt.Errorf("restore state: %w", err)
		}
	}

	// Phase 1: Discover article URLs
	fmt.Println("Phase 1: Discovering articles...")
	articleIDs, err := c.discover(ctx)
	if err != nil {
		return fmt.Errorf("discover: %w", err)
	}

	// Filter out already-crawled articles
	var newIDs []string
	for _, id := range articleIDs {
		if !c.filter.TestString(id) {
			newIDs = append(newIDs, id)
			c.filter.AddString(id)
		} else {
			c.skipped.Add(1)
		}
	}

	c.discovered.Store(int64(len(newIDs)))
	fmt.Printf("  Discovered %d new articles (%d skipped, already crawled)\n", len(newIDs), c.skipped.Load())

	if len(newIDs) == 0 {
		fmt.Println("No new articles to fetch.")
		return c.printStats()
	}

	// Phase 2: Fetch article content
	rlStr := "unlimited"
	if c.config.RateLimit > 0 {
		rlStr = fmt.Sprintf("%.0f req/s", c.config.RateLimit)
	}
	fmt.Printf("Phase 2: Fetching %d articles with %d workers (rate: %s)...\n", len(newIDs), c.config.Workers, rlStr)
	c.startTime = time.Now()
	c.fetchArticles(ctx, newIDs)

	// Print final stats
	return c.printStats()
}

func (c *Crawler) restoreState() error {
	fmt.Println("Restoring state from previous crawl...")
	ids, err := c.db.AllArticleIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		c.filter.AddString(id)
	}
	fmt.Printf("  Loaded %d previously processed article IDs into bloom filter\n", len(ids))
	return nil
}

func (c *Crawler) discover(ctx context.Context) ([]string, error) {
	var allIDs []string
	seen := make(map[string]bool)

	addID := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			allIDs = append(allIDs, id)
		}
	}

	// 1. Sitemap discovery
	sitemapIDs, err := c.discoverFromSitemaps(ctx)
	if err != nil {
		fmt.Printf("  Warning: sitemap discovery failed: %v\n", err)
	} else {
		for _, id := range sitemapIDs {
			addID(id)
		}
		fmt.Printf("  Sitemaps: %d articles from %d sitemaps\n", len(sitemapIDs), c.sitemapsDone.Load())
	}

	// 2. Channel feed discovery (optional)
	if c.config.Channels {
		feedIDs, err := c.discoverFromFeeds(ctx)
		if err != nil {
			fmt.Printf("  Warning: feed discovery failed: %v\n", err)
		} else {
			before := len(allIDs)
			for _, id := range feedIDs {
				addID(id)
			}
			fmt.Printf("  Feeds: %d new articles from %d channels + hot ranking\n", len(allIDs)-before, len(ChannelIDs))
		}
	}

	return allIDs, nil
}

func (c *Crawler) discoverFromSitemaps(ctx context.Context) ([]string, error) {
	// Collect all sitemap URLs: from index + enumerated probes
	sitemapURLs, err := c.collectAllSitemapURLs(ctx)
	if err != nil {
		return nil, err
	}

	c.sitemapsTotal.Store(int64(len(sitemapURLs)))

	var fetchedSitemaps map[string]bool
	if c.config.Resume {
		fetchedSitemaps, err = c.db.FetchedSitemaps()
		if err != nil {
			return nil, err
		}
	}

	var allIDs []string
	var mu sync.Mutex
	var sitemapFails atomic.Int64

	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for _, sitemapURL := range sitemapURLs {
		if ctx.Err() != nil {
			break
		}

		if fetchedSitemaps != nil && fetchedSitemaps[sitemapURL] {
			c.sitemapsDone.Add(1)
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(url string) {
			defer wg.Done()
			defer func() { <-sem }()

			// Retry sitemap fetch up to 3 times with backoff (skip 404s)
			var urlSet *URLSet
			var fetchErr error
			for attempt := 0; attempt < 3; attempt++ {
				if ctx.Err() != nil {
					c.sitemapsDone.Add(1)
					sitemapFails.Add(1)
					return
				}
				if attempt > 0 {
					time.Sleep(time.Duration(attempt) * 2 * time.Second)
				}
				// Rate limit sitemap fetches to avoid WAF
				if c.limiter != nil {
					c.limiter.Wait(ctx)
				}
				urlSet, fetchErr = c.client.FetchSitemap(ctx, url)
				if fetchErr == nil {
					break
				}
				// Don't retry 404s — sitemap genuinely doesn't exist
				if strings.Contains(fetchErr.Error(), "HTTP 404") {
					break
				}
			}

			c.sitemapsDone.Add(1)

			if fetchErr != nil {
				sitemapFails.Add(1)
				done := c.sitemapsDone.Load()
				total := c.sitemapsTotal.Load()
				if done%50 == 0 || done == total {
					mu.Lock()
					n := len(allIDs)
					mu.Unlock()
					fmt.Printf("    Sitemaps: %d/%d processed (%d failed), %d articles found\n",
						done, total, sitemapFails.Load(), n)
				}
				return
			}

			var ids []string
			for _, u := range urlSet.URLs {
				if id := ExtractArticleID(u.Loc); id != "" {
					ids = append(ids, id)
				}
			}

			mu.Lock()
			allIDs = append(allIDs, ids...)
			mu.Unlock()

			c.db.MarkSitemap(url, len(ids))

			done := c.sitemapsDone.Load()
			total := c.sitemapsTotal.Load()
			if done%50 == 0 || done == total {
				fails := sitemapFails.Load()
				mu.Lock()
				n := len(allIDs)
				mu.Unlock()
				if fails > 0 {
					fmt.Printf("    Sitemaps: %d/%d processed (%d failed), %d articles found\n",
						done, total, fails, n)
				} else {
					fmt.Printf("    Sitemaps: %d/%d processed, %d articles found\n",
						done, total, n)
				}
			}
		}(sitemapURL)
	}

	wg.Wait()

	// Retry failed sitemaps: collect those not yet in DB
	fails := sitemapFails.Load()
	if fails > 0 {
		fmt.Printf("    Retrying %d failed sitemaps...\n", fails)
		sitemapFails.Store(0)
		var retryURLs []string
		dbSitemaps, _ := c.db.FetchedSitemaps()
		for _, url := range sitemapURLs {
			if dbSitemaps != nil && dbSitemaps[url] {
				continue
			}
			if fetchedSitemaps != nil && fetchedSitemaps[url] {
				continue
			}
			retryURLs = append(retryURLs, url)
		}

		// Retry with lower concurrency and longer backoff
		retrySem := make(chan struct{}, 5)
		var retryWg sync.WaitGroup

		for _, sitemapURL := range retryURLs {
			if ctx.Err() != nil {
				break
			}

			retryWg.Add(1)
			retrySem <- struct{}{}

			go func(url string) {
				defer retryWg.Done()
				defer func() { <-retrySem }()

				// Rate limit + extra delay for retry round
				if c.limiter != nil {
					c.limiter.Wait(ctx)
				}
				time.Sleep(500 * time.Millisecond)

				urlSet, err := c.client.FetchSitemap(ctx, url)
				if err != nil {
					sitemapFails.Add(1)
					return
				}

				var ids []string
				for _, u := range urlSet.URLs {
					if id := ExtractArticleID(u.Loc); id != "" {
						ids = append(ids, id)
					}
				}

				mu.Lock()
				allIDs = append(allIDs, ids...)
				mu.Unlock()

				c.db.MarkSitemap(url, len(ids))
			}(sitemapURL)
		}

		retryWg.Wait()
		retryFails := sitemapFails.Load()
		if retryFails > 0 {
			fmt.Printf("    Retry complete: %d sitemaps still failed\n", retryFails)
		} else {
			fmt.Printf("    Retry complete: all sitemaps succeeded\n")
		}
	}

	return allIDs, nil
}

// collectAllSitemapURLs builds the full list of sitemap URLs to process.
// It starts with the index.xml, then probes for sitemaps not in the index
// by enumerating the known PP/MM/DD/XXXX structure.
func (c *Crawler) collectAllSitemapURLs(ctx context.Context) ([]string, error) {
	known := make(map[string]bool)

	// 1. Fetch sitemap index
	fmt.Println("    Fetching sitemap index...")
	idx, err := c.client.FetchSitemapIndex(ctx)
	if err != nil {
		// If index fails (WAF), fall back to DB sitemaps + enumeration
		fmt.Printf("    Warning: sitemap index unavailable (%v), using cached + enumeration\n", err)
	} else {
		for _, s := range idx.Sitemaps {
			known[s.Loc] = true
		}
		fmt.Printf("    Index: %d sitemaps\n", len(idx.Sitemaps))
	}

	// Add any previously-fetched sitemaps from DB
	dbSitemaps, _ := c.db.FetchedSitemaps()
	for url := range dbSitemaps {
		known[url] = true
	}

	if !c.config.Probe {
		urls := make([]string, 0, len(known))
		for url := range known {
			urls = append(urls, url)
		}
		return urls, nil
	}

	// 2. Enumerate ALL possible sitemap buckets
	// Structure: PP/MM/DD/XXXX where:
	//   PP: 10-19 (known: 10-13, probe wider)
	//   MM: {0,1,2, 10,11,12, ..., 90,91,92} (30 values)
	//   DD: {0,3, 10,13, ..., 90,93} (20 values)
	//   XXXX: 0000-9999 suffix (must be probed)
	fmt.Println("    Probing for sitemaps not in index...")

	// Build set of already-known buckets (PP,MM,DD)
	knownBuckets := make(map[string]bool)
	for url := range known {
		id := extractSitemapNumber(url)
		if id == "" {
			continue
		}
		if len(id) >= 6 {
			bucket := id[:6] // PP+MM+DD
			knownBuckets[bucket] = true
		}
	}

	// Generate all possible buckets
	var missingBuckets []string
	for _, p1 := range sitemapP1Values {
		for _, p2 := range sitemapP2Values {
			for _, p3 := range sitemapP3Values {
				bucket := fmt.Sprintf("%02d%02d%02d", p1, p2, p3)
				if !knownBuckets[bucket] {
					missingBuckets = append(missingBuckets, bucket)
				}
			}
		}
	}

	fmt.Printf("    Known buckets: %d, missing buckets to probe: %d\n", len(knownBuckets), len(missingBuckets))

	// 3. Probe missing buckets (try suffix 0000-0999 via concurrent workers)
	if len(missingBuckets) > 0 {
		found := c.probeMissingSitemaps(ctx, missingBuckets)
		for _, url := range found {
			known[url] = true
		}
		fmt.Printf("    Probing found %d additional sitemaps\n", len(found))
	}

	// Convert to sorted list
	urls := make([]string, 0, len(known))
	for url := range known {
		urls = append(urls, url)
	}

	return urls, nil
}

// probeMissingSitemaps tries to find sitemaps for buckets not in the index.
// For each missing bucket, it probes suffixes 0001-0999 via GET requests.
// Uses a fast scan (every 50th suffix) then fine scan around hits.
// Stops immediately on WAF (501) to avoid IP ban.
func (c *Crawler) probeMissingSitemaps(ctx context.Context, buckets []string) []string {
	var found []string
	var mu sync.Mutex
	var probed atomic.Int64
	var wafHit atomic.Bool
	total := len(buckets)

	sem := make(chan struct{}, 5) // conservative concurrency for probing
	var wg sync.WaitGroup

	for _, bucket := range buckets {
		if ctx.Err() != nil || wafHit.Load() {
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(b string) {
			defer wg.Done()
			defer func() { <-sem }()

			if wafHit.Load() {
				return
			}

			// Rate limit probes
			if c.limiter != nil {
				c.limiter.Wait(ctx)
			}

			// Fast scan: try every 50th suffix (20 probes per bucket)
			for coarse := 0; coarse < 1000; coarse += 50 {
				if ctx.Err() != nil || wafHit.Load() {
					return
				}

				url := fmt.Sprintf("https://news.qq.com/sitemap/sitemap_%s%04d.xml", b, coarse)
				status := c.client.ProbeSitemapStatus(ctx, url)

				if status == 501 {
					wafHit.Store(true)
					fmt.Printf("\n    WAF detected (501), stopping probe\n")
					return
				}

				if status == 200 {
					// Found nearby! Fine-scan ±50 around the hit
					for fine := max(0, coarse-49); fine < min(coarse+50, 1000); fine++ {
						if ctx.Err() != nil || wafHit.Load() {
							return
						}
						fineURL := fmt.Sprintf("https://news.qq.com/sitemap/sitemap_%s%04d.xml", b, fine)
						if c.client.ProbeSitemapStatus(ctx, fineURL) == 200 {
							mu.Lock()
							found = append(found, fineURL)
							mu.Unlock()
							break // one sitemap per bucket
						}
					}
					probed.Add(1)
					return
				}
			}

			probed.Add(1)
			n := probed.Load()
			if n%50 == 0 {
				mu.Lock()
				nf := len(found)
				mu.Unlock()
				fmt.Printf("    Probe: %d/%d buckets checked, %d found\r", n, total, nf)
			}
		}(bucket)
	}

	wg.Wait()
	fmt.Printf("    Probe: %d/%d buckets checked, %d found\n", probed.Load(), total, len(found))
	return found
}

func extractSitemapNumber(url string) string {
	const prefix = "sitemap_"
	idx := strings.LastIndex(url, prefix)
	if idx == -1 {
		return ""
	}
	s := url[idx+len(prefix):]
	if dot := strings.Index(s, "."); dot != -1 {
		s = s[:dot]
	}
	return s
}

// Known structural values for sitemap number components.
var (
	// p1: page group prefix (known: 10-13, probe wider: 10-19)
	sitemapP1Values = []int{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}

	// p2: 30 values {X0, X1, X2} for X in 0-9
	sitemapP2Values = []int{
		0, 1, 2, 10, 11, 12, 20, 21, 22, 30, 31, 32,
		40, 41, 42, 50, 51, 52, 60, 61, 62, 70, 71, 72,
		80, 81, 82, 90, 91, 92,
	}

	// p3: 20 values {X0, X3} for X in 0-9
	sitemapP3Values = []int{
		0, 3, 10, 13, 20, 23, 30, 33, 40, 43,
		50, 53, 60, 63, 70, 73, 80, 83, 90, 93,
	}
)

func (c *Crawler) discoverFromFeeds(ctx context.Context) ([]string, error) {
	var allIDs []string

	hotItems, err := c.client.FetchHotRanking(ctx)
	if err != nil {
		fmt.Printf("    Warning: hot ranking failed: %v\n", err)
	} else {
		for _, item := range hotItems {
			allIDs = append(allIDs, item.ID)
		}
	}

	for _, chID := range ChannelIDs {
		if ctx.Err() != nil {
			break
		}
		items, err := c.client.FetchChannelFeed(ctx, chID)
		if err != nil {
			continue
		}
		for _, item := range items {
			allIDs = append(allIDs, item.ID)
		}
		time.Sleep(200 * time.Millisecond)
	}

	return allIDs, nil
}

func (c *Crawler) fetchArticles(ctx context.Context, articleIDs []string) {
	ch := make(chan string, min(len(articleIDs), 100_000))

	// Feed articles in a goroutine to avoid blocking on huge slices
	go func() {
		for _, id := range articleIDs {
			ch <- id
		}
		close(ch)
	}()

	var wg sync.WaitGroup
	var batchMu sync.Mutex
	var batch []Article

	flushBatch := func() {
		batchMu.Lock()
		if len(batch) > 0 {
			toFlush := make([]Article, len(batch))
			copy(toFlush, batch)
			batch = batch[:0]
			batchMu.Unlock()
			if err := c.db.InsertArticles(toFlush); err != nil {
				fmt.Printf("\n  Warning: batch insert failed: %v\n", err)
			}
		} else {
			batchMu.Unlock()
		}
	}

	for i := 0; i < c.config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for id := range ch {
				if ctx.Err() != nil {
					return
				}

				// Rate limit
				if c.limiter != nil {
					c.limiter.Wait(ctx)
				}

				article := c.fetchWithRetry(ctx, id)

				batchMu.Lock()
				batch = append(batch, article)
				shouldFlush := len(batch) >= 100
				batchMu.Unlock()

				if shouldFlush {
					flushBatch()
				}

				// Progress display
				fetched := c.fetched.Load()
				if fetched%50 == 0 {
					c.printProgress()
				}
			}
		}()
	}

	wg.Wait()
	flushBatch()
	fmt.Println()
}

func (c *Crawler) fetchWithRetry(ctx context.Context, articleID string) Article {
	var lastArticle Article

	for attempt := 0; attempt <= c.config.MaxRetry; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s
			backoff := time.Duration(1<<attempt) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return lastArticle
			}
		}

		article := c.fetchSingleArticle(ctx, articleID)
		lastArticle = article

		// Don't retry deleted articles or successful fetches
		if article.Error == "" || article.Error == "deleted" {
			return article
		}

		// Only retry on rate limiting (567) or server errors (5xx)
		if article.StatusCode == 567 || (article.StatusCode >= 500 && article.StatusCode < 600) {
			continue
		}

		return article // non-retryable error
	}

	return lastArticle
}

func (c *Crawler) fetchSingleArticle(ctx context.Context, articleID string) Article {
	c.fetched.Add(1)

	html, statusCode, err := c.client.FetchArticlePage(ctx, articleID)
	if err != nil {
		if errors.Is(err, ErrArticleDeleted) {
			c.deleted.Add(1)
			return Article{
				ArticleID:  articleID,
				URL:        ArticleBaseURL + articleID,
				CrawledAt:  time.Now(),
				StatusCode: statusCode,
				Error:      "deleted",
			}
		}
		c.failed.Add(1)
		return Article{
			ArticleID:  articleID,
			URL:        ArticleBaseURL + articleID,
			CrawledAt:  time.Now(),
			StatusCode: statusCode,
			Error:      err.Error(),
		}
	}

	if statusCode == 567 {
		c.rateLimited.Add(1)
		c.failed.Add(1)
		return Article{
			ArticleID:  articleID,
			URL:        ArticleBaseURL + articleID,
			CrawledAt:  time.Now(),
			StatusCode: statusCode,
			Error:      "rate limited (567)",
		}
	}

	if statusCode != 200 {
		c.failed.Add(1)
		return Article{
			ArticleID:  articleID,
			URL:        ArticleBaseURL + articleID,
			CrawledAt:  time.Now(),
			StatusCode: statusCode,
			Error:      fmt.Sprintf("HTTP %d", statusCode),
		}
	}

	article, err := ParseArticlePage(html, articleID)
	if err != nil {
		c.failed.Add(1)
		return Article{
			ArticleID:  articleID,
			URL:        ArticleBaseURL + articleID,
			CrawledAt:  time.Now(),
			StatusCode: statusCode,
			Error:      err.Error(),
		}
	}

	article.StatusCode = statusCode
	c.succeeded.Add(1)
	return *article
}

func (c *Crawler) printProgress() {
	total := c.discovered.Load()
	fetched := c.fetched.Load()
	ok := c.succeeded.Load()
	del := c.deleted.Load()
	rl := c.rateLimited.Load()
	fail := c.failed.Load()

	elapsed := time.Since(c.startTime).Seconds()
	speed := float64(0)
	if elapsed > 0 {
		speed = float64(fetched) / elapsed
	}

	fmt.Printf("\r  %d/%d  ok=%-6d deleted=%-6d ratelimit=%-5d fail=%-5d  [%.1f/s]     ",
		fetched, total, ok, del, rl, fail-rl, speed)
}

func (c *Crawler) printStats() error {
	stats, err := c.db.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("\n── QQ News Crawl Stats ──")
	fmt.Printf("  Articles in DB:    %d\n", stats.Articles)
	fmt.Printf("  With content:      %d\n", stats.WithContent)
	fmt.Printf("  Deleted:           %d\n", stats.Deleted)
	fmt.Printf("  With errors:       %d\n", stats.WithError)
	fmt.Printf("  Sitemaps tracked:  %d\n", stats.Sitemaps)
	fmt.Printf("  DB size:           %.1f MB\n", float64(stats.DBSize)/(1024*1024))

	if len(stats.Channels) > 0 {
		fmt.Println("  Top channels:")
		i := 0
		for ch, cnt := range stats.Channels {
			if i >= 10 {
				break
			}
			fmt.Printf("    %-20s %d\n", ch, cnt)
			i++
		}
	}

	fmt.Printf("  DB path:           %s\n", c.db.Path())
	return nil
}

// Stats returns current crawl statistics.
func (c *Crawler) Stats() (discovered, fetched, succeeded, failed, deleted, skipped int64) {
	return c.discovered.Load(), c.fetched.Load(), c.succeeded.Load(), c.failed.Load(), c.deleted.Load(), c.skipped.Load()
}

// DB returns the underlying database.
func (c *Crawler) DB() *DB {
	return c.db
}
