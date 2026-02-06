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
	config  Config
	clients []*http.Client // sharded HTTP clients for reduced lock contention
	stats   *Stats
	rdb     *ResultDB

	// Per-domain failure tracking: once a domain fails, skip remaining URLs
	deadDomainsMu sync.RWMutex
	deadDomains   map[string]bool

	// Cached DNS: pre-resolved domain → IP for direct dialing
	dnsCache   map[string][]string
	dnsCacheMu sync.RWMutex
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
	if cfg.TransportShards < 1 {
		cfg.TransportShards = 1
	}

	r := &Recrawler{
		config:      cfg,
		deadDomains: make(map[string]bool),
		stats:       stats,
		rdb:         rdb,
		dnsCache:    make(map[string][]string),
	}

	// Create sharded HTTP clients — each shard has its own transport+connection pool.
	// Workers hash to a shard, spreading lock contention across N pools.
	r.clients = make([]*http.Client, cfg.TransportShards)
	for i := range cfg.TransportShards {
		r.clients[i] = r.buildClient(i)
	}

	return r
}

func (r *Recrawler) buildClient(shardID int) *http.Client {
	cfg := r.config

	dialTimeout := min(cfg.Timeout/2, 2*time.Second)
	tlsTimeout := min(cfg.Timeout/2, 2*time.Second)

	// Divide idle conns across shards
	maxIdlePerShard := min(cfg.Workers*2/max(cfg.TransportShards, 1), 100000)

	// Custom dialer that uses cached DNS IPs when available.
	// This eliminates runtime DNS lookups entirely for pre-resolved domains.
	baseDialer := &net.Dialer{
		Timeout:       dialTimeout,
		KeepAlive:     15 * time.Second,
		FallbackDelay: -1, // disable happy-eyeballs delay
	}

	dialFunc := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return baseDialer.DialContext(ctx, network, addr)
		}

		// Try cached DNS first
		r.dnsCacheMu.RLock()
		ips := r.dnsCache[host]
		r.dnsCacheMu.RUnlock()

		if len(ips) > 0 {
			// Round-robin across IPs using shard ID for distribution
			ip := ips[shardID%len(ips)]
			return baseDialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		}

		return baseDialer.DialContext(ctx, network, addr)
	}

	transport := &http.Transport{
		DialContext:           dialFunc,
		TLSClientConfig:      &tls.Config{InsecureSkipVerify: false},
		MaxIdleConns:         maxIdlePerShard,
		MaxIdleConnsPerHost:  50,
		MaxConnsPerHost:      0,
		IdleConnTimeout:      30 * time.Second,
		TLSHandshakeTimeout: tlsTimeout,
		ResponseHeaderTimeout: cfg.Timeout,
		DisableCompression:   true,
		ForceAttemptHTTP2:    false, // HTTP/1.1 is faster for many-host one-shot fetches
		WriteBufferSize:      4 * 1024,
		ReadBufferSize:       8 * 1024,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 2 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
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

// SetDNSCache populates the cached DNS map for direct-IP dialing.
func (r *Recrawler) SetDNSCache(resolved map[string][]string) {
	r.dnsCacheMu.Lock()
	for domain, ips := range resolved {
		r.dnsCache[domain] = ips
	}
	r.dnsCacheMu.Unlock()
}

func (r *Recrawler) isDomainDead(domain string) bool {
	r.deadDomainsMu.RLock()
	dead := r.deadDomains[domain]
	r.deadDomainsMu.RUnlock()
	return dead
}

func (r *Recrawler) markDomainDead(domain string) {
	r.deadDomainsMu.Lock()
	r.deadDomains[domain] = true
	r.deadDomainsMu.Unlock()
}

// shuffleURLs randomizes URL order using Fisher-Yates shuffle.
func shuffleURLs(urls []SeedURL) {
	rand.Shuffle(len(urls), func(i, j int) {
		urls[i], urls[j] = urls[j], urls[i]
	})
}

// clientForWorker returns the HTTP client for a worker ID (sharded).
func (r *Recrawler) clientForWorker(workerID int) *http.Client {
	return r.clients[workerID%len(r.clients)]
}

// Run executes the recrawl on the given URL set.
func (r *Recrawler) Run(ctx context.Context, seeds []SeedURL, skip map[string]bool) error {
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

	// Pre-filter dead-domain URLs before entering the worker pipeline
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

	shuffleURLs(liveURLs)

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

	nWorkers := min(r.config.Workers, len(liveURLs))
	g, ctx := errgroup.WithContext(ctx)
	for workerID := range nWorkers {
		client := r.clientForWorker(workerID)
		g.Go(func() error {
			return r.worker(ctx, client, urlCh)
		})
	}

	return g.Wait()
}

func (r *Recrawler) worker(ctx context.Context, client *http.Client, urls <-chan SeedURL) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case seed, ok := <-urls:
			if !ok {
				return nil
			}
			if r.isDomainDead(seed.Domain) {
				r.stats.RecordDomainSkip()
				continue
			}
			r.fetchOne(ctx, client, seed)
		}
	}
}

func (r *Recrawler) fetchOne(ctx context.Context, client *http.Client, seed SeedURL) {
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

	resp, err := client.Do(req)
	if err != nil {
		isTimeout := isTimeoutError(err)
		isFatal := isTimeout || isConnectionRefused(err) || isDNSError(err)
		r.stats.RecordFailure(0, seed.Domain, isTimeout)
		r.rdb.Add(Result{
			URL:         seed.URL,
			Domain:      seed.Domain,
			FetchTimeMs: time.Since(start).Milliseconds(),
			CrawledAt:   time.Now(),
			Error:       truncateStr(err.Error(), 200),
		})
		if isFatal {
			r.markDomainDead(seed.Domain)
		}
		return
	}

	// StatusOnly mode: close body immediately, only record status code
	if r.config.StatusOnly {
		resp.Body.Close()
		fetchMs := time.Since(start).Milliseconds()
		bodySize := resp.ContentLength

		redirectURL := ""
		if resp.Request != nil && resp.Request.URL.String() != seed.URL {
			redirectURL = resp.Request.URL.String()
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			r.stats.RecordSuccess(resp.StatusCode, seed.Domain, max(bodySize, 0), fetchMs)
		} else {
			r.stats.RecordFailure(resp.StatusCode, seed.Domain, false)
		}
		r.rdb.Add(Result{
			URL:           seed.URL,
			StatusCode:    resp.StatusCode,
			ContentType:   resp.Header.Get("Content-Type"),
			ContentLength: max(bodySize, 0),
			Domain:        seed.Domain,
			RedirectURL:   redirectURL,
			FetchTimeMs:   fetchMs,
			CrawledAt:     time.Now(),
		})
		return
	}

	// Full fetch mode: extract metadata and drain body
	var (
		title       string
		description string
		language    string
		bodySize    int64
	)

	ct := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")

	if !r.config.HeadOnly && resp.StatusCode == 200 && isHTML {
		limited := io.LimitReader(resp.Body, 128*1024)
		extracted := crawler.Extract(limited, seed.URL)
		title = extracted.Title
		description = extracted.Description
		language = extracted.Language
	}

	n, _ := io.CopyN(io.Discard, resp.Body, 256*1024)
	resp.Body.Close()

	if resp.ContentLength > 0 {
		bodySize = resp.ContentLength
	} else if n > 0 {
		bodySize = n
	}

	fetchMs := time.Since(start).Milliseconds()

	redirectURL := ""
	if resp.Request != nil && resp.Request.URL.String() != seed.URL {
		redirectURL = resp.Request.URL.String()
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		r.stats.RecordSuccess(resp.StatusCode, seed.Domain, bodySize, fetchMs)
	} else {
		r.stats.RecordFailure(resp.StatusCode, seed.Domain, false)
	}

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
	r.rdb.Add(Result{
		URL:         seed.URL,
		Domain:      seed.Domain,
		FetchTimeMs: time.Since(start).Milliseconds(),
		CrawledAt:   time.Now(),
		Error:       truncateStr(err.Error(), 200),
	})
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "timeout") ||
		strings.Contains(s, "deadline exceeded") ||
		strings.Contains(s, "context deadline")
}

func isConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "no route to host")
}

func isDNSError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "no such host") ||
		strings.Contains(s, "dial tcp: lookup")
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// RunWithDisplay runs the recrawl with live terminal display updates.
func RunWithDisplay(ctx context.Context, r *Recrawler, seeds []SeedURL, skip map[string]bool, stats *Stats) error {
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

				if stats.Done() >= int64(stats.TotalURLs) {
					stats.Freeze()
					return
				}
			}
		}
	}()

	err := r.Run(ctx, seeds, skip)

	<-displayDone
	stats.Freeze()

	displayMu.Lock()
	if displayLines > 0 {
		fmt.Printf("\033[%dA\033[J", displayLines)
	}
	fmt.Print(stats.Render())
	displayMu.Unlock()

	return err
}
