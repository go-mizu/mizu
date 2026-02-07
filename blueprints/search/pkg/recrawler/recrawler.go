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

	// DNS resolver for pipelined mode (resolve + fetch concurrently)
	dnsResolver *DNSResolver
}

// New creates a recrawler optimized for maximum throughput.
func New(cfg Config, stats *Stats, rdb *ResultDB) *Recrawler {
	if cfg.Workers == 0 {
		cfg.Workers = 2000
	}
	if cfg.DNSWorkers == 0 {
		cfg.DNSWorkers = 2000
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
	tlsTimeout := min(cfg.Timeout/2, 3*time.Second)

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

// SetDNSResolver enables pipelined mode: DNS resolution and HTTP fetching
// happen concurrently, partitioned by domain. As each domain resolves,
// its URLs immediately enter the fetch pipeline.
func (r *Recrawler) SetDNSResolver(dns *DNSResolver) {
	r.dnsResolver = dns
	// Pre-populate dnsCache with already-cached entries from the resolver
	resolved := dns.ResolvedIPs()
	if len(resolved) > 0 {
		r.dnsCacheMu.Lock()
		for domain, ips := range resolved {
			r.dnsCache[domain] = ips
		}
		r.dnsCacheMu.Unlock()
	}
	// Pre-populate dead domains from cached dead + timeout entries
	deadOrTimeout := dns.DeadOrTimeoutDomains()
	if len(deadOrTimeout) > 0 {
		r.deadDomainsMu.Lock()
		for d := range deadOrTimeout {
			r.deadDomains[d] = true
		}
		r.deadDomainsMu.Unlock()
	}
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

// HTTPDeadDomains returns domains marked dead during HTTP fetching.
// These are domains where TCP connection was refused/reset (not timeouts).
// Can be merged into DNS cache for reuse in subsequent runs.
func (r *Recrawler) HTTPDeadDomains() map[string]bool {
	r.deadDomainsMu.RLock()
	result := make(map[string]bool, len(r.deadDomains))
	for d := range r.deadDomains {
		result[d] = true
	}
	r.deadDomainsMu.RUnlock()
	return result
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
// If a DNS resolver is set (via SetDNSResolver), uses a domain-partitioned pipeline
// where DNS resolution and HTTP fetching happen concurrently.
func (r *Recrawler) Run(ctx context.Context, seeds []SeedURL, skip map[string]bool) error {
	// Group URLs by domain, filtering skipped URLs
	domainURLs := make(map[string][]SeedURL, len(seeds)/2)
	totalLive := 0
	for _, s := range seeds {
		if skip != nil && skip[s.URL] {
			r.stats.RecordSkip()
			continue
		}
		domainURLs[s.Domain] = append(domainURLs[s.Domain], s)
		totalLive++
	}

	if totalLive == 0 {
		return nil
	}

	// Shuffle domains for load distribution (Fisher-Yates)
	domains := make([]string, 0, len(domainURLs))
	for d := range domainURLs {
		domains = append(domains, d)
	}
	rand.Shuffle(len(domains), func(i, j int) {
		domains[i], domains[j] = domains[j], domains[i]
	})

	// Create URL channel for fetch workers
	urlCh := make(chan SeedURL, min(totalLive, r.config.Workers*4))

	// Create a cancelable context so we can stop the pipeline on return
	pipeCtx, pipeCancel := context.WithCancel(ctx)

	if r.dnsResolver != nil {
		// Pipelined: DNS workers resolve domains and feed URLs to fetch pipeline
		go r.dnsPipeline(pipeCtx, domains, domainURLs, urlCh)
	} else {
		// Direct: pre-filter dead domains, shuffle, and feed URLs
		go r.directFeed(pipeCtx, domains, domainURLs, urlCh)
	}

	// Launch HTTP fetch workers
	nWorkers := min(r.config.Workers, totalLive)
	g, gCtx := errgroup.WithContext(pipeCtx)
	for workerID := range nWorkers {
		client := r.clientForWorker(workerID)
		g.Go(func() error {
			return r.worker(gCtx, client, urlCh)
		})
	}

	err := g.Wait()
	pipeCancel() // ensure DNS pipeline stops if still running
	return err
}

// dnsPipeline resolves domains and pushes their URLs to the fetch channel.
// Runs concurrently with HTTP fetch workers for maximum throughput.
// NXDOMAIN and timeout domains are filtered out.
//
// In TwoPass mode, DNS-live domains get an additional HTTP HEAD probe:
// only domains that respond to the probe have their URLs pushed to fetch.
func (r *Recrawler) dnsPipeline(ctx context.Context, domains []string, domainURLs map[string][]SeedURL, urlCh chan<- SeedURL) {
	defer close(urlCh)

	domainCh := make(chan string, min(len(domains), 10000))

	// Feed domains
	go func() {
		defer close(domainCh)
		for _, d := range domains {
			select {
			case domainCh <- d:
			case <-ctx.Done():
				return
			}
		}
	}()

	// DNS workers: resolve each domain, push its URLs
	dnsWorkers := min(r.config.DNSWorkers, len(domains))
	var wg sync.WaitGroup
	for range dnsWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range domainCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				urls := domainURLs[domain]

				ips, dead, _ := r.dnsResolver.ResolveOne(ctx, domain)

				if dead {
					r.markDomainDead(domain)
					r.stats.RecordDNSDead()
					for range urls {
						r.stats.RecordDomainSkip()
					}
					continue
				}

				if len(ips) == 0 {
					// DNS timeout on ALL resolvers — mark dead, skip URLs
					r.markDomainDead(domain)
					r.stats.RecordDNSTimeout()
					for range urls {
						r.stats.RecordDomainSkip()
					}
					continue
				}

				// DNS resolved — cache IPs
				r.stats.RecordDNSLive()
				r.dnsCacheMu.Lock()
				r.dnsCache[domain] = ips
				r.dnsCacheMu.Unlock()

				// Two-pass mode: probe domain before pushing URLs
				if r.config.TwoPass {
					if !r.probeDomain(ctx, domain, urls[0].URL) {
						r.stats.RecordProbeUnreachable()
						r.markDomainDead(domain)
						for range urls {
							r.stats.RecordDomainSkip()
						}
						continue
					}
					r.stats.RecordProbeReachable()
				}

				for _, u := range urls {
					select {
					case urlCh <- u:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	wg.Wait()
}

// probeDomain sends a lightweight HEAD request to one URL on the domain
// to check if the server is reachable. Returns true if the domain should
// be fetched (server responded or timed out — conservative), false only
// if the connection was definitively refused/reset.
func (r *Recrawler) probeDomain(ctx context.Context, _, probeURL string) bool {
	probeTimeout := 500 * time.Millisecond
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodHead, probeURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", r.config.UserAgent)

	client := r.clientForWorker(0)
	resp, err := client.Do(req)
	if err != nil {
		// Timeout → conservative: domain might be slow but alive, still fetch
		if isTimeoutError(err) {
			return true
		}
		// Connection refused/reset/no route → definitively unreachable
		return false
	}
	resp.Body.Close()
	// Any HTTP response (1xx-5xx) means the server is alive
	return true
}

// directFeed pushes URLs to the fetch channel without DNS resolution.
// Pre-filters dead-domain URLs and shuffles the rest.
func (r *Recrawler) directFeed(ctx context.Context, domains []string, domainURLs map[string][]SeedURL, urlCh chan<- SeedURL) {
	defer close(urlCh)

	var liveURLs []SeedURL
	for _, d := range domains {
		if r.isDomainDead(d) {
			for range domainURLs[d] {
				r.stats.RecordDomainSkip()
			}
			continue
		}
		liveURLs = append(liveURLs, domainURLs[d]...)
	}

	if len(liveURLs) == 0 {
		return
	}

	shuffleURLs(liveURLs)

	for _, u := range liveURLs {
		select {
		case urlCh <- u:
		case <-ctx.Done():
			return
		}
	}
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
		// Mark domain dead on definitive connection failures only.
		// Timeouts are NOT fatal — server may be slow but alive.
		isFatal := isConnectionRefused(err) || isDNSError(err)
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

	// Full fetch mode: read body, extract metadata
	ct := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")

	// Read body (up to 512KB for full content capture)
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	resp.Body.Close()
	bodySize := int64(len(bodyBytes))
	if resp.ContentLength > bodySize {
		bodySize = resp.ContentLength
	}

	var title, description, language, body string
	if resp.StatusCode == 200 && isHTML && len(bodyBytes) > 0 {
		body = string(bodyBytes)
		extracted := crawler.Extract(strings.NewReader(body), seed.URL)
		title = extracted.Title
		description = extracted.Description
		language = extracted.Language
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
		ContentType:   ct,
		ContentLength: bodySize,
		Body:          body,
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
