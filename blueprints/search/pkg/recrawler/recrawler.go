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
	"sync/atomic"
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

	// Failed domain/URL logger (nil-safe: no-op if not set)
	failedDB *FailedDB

	// Dead domain tracking: sync.Map for lock-free reads in hot path (50K+ workers)
	// Values are reason strings: dns_nxdomain, dns_timeout, probe_unreachable, http_timeout_killed, http_refused
	deadDomains sync.Map // domain → reason (string)

	// Cached DNS: pre-resolved domain → IP for direct dialing
	dnsCache   map[string][]string
	dnsCacheMu sync.RWMutex

	// Per-domain connection limiter: prevents flooding individual servers
	// Pre-created in Run() to avoid mutex contention during fetch.
	domainSems   map[string]chan struct{}
	domainSemsMu sync.RWMutex

	// Per-domain timeout tracking: kill domains that consistently time out
	// sync.Map eliminates global mutex for 24K+ concurrent workers
	domainFailCounts sync.Map // domain → *atomic.Int32
	domainSucceeded  sync.Map // domain → true

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
		cfg.DomainFailThreshold = 2
	}
	if cfg.TransportShards < 1 {
		cfg.TransportShards = 1
	}
	if cfg.MaxConnsPerDomain == 0 {
		cfg.MaxConnsPerDomain = 8
	}

	r := &Recrawler{
		config:     cfg,
		stats:      stats,
		rdb:        rdb,
		dnsCache:   make(map[string][]string),
		domainSems: make(map[string]chan struct{}),
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

// SetFailedDB sets the FailedDB for logging failed domains and URLs.
func (r *Recrawler) SetFailedDB(fdb *FailedDB) {
	r.failedDB = fdb
}

// SetDeadDomains pre-populates dead domains with failure reasons.
// Values should be reason strings: dns_nxdomain, dns_timeout, etc.
func (r *Recrawler) SetDeadDomains(domains map[string]string) {
	for d, reason := range domains {
		r.deadDomains.Store(d, reason)
	}
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
	// Pre-populate dead domains with reasons
	for d := range dns.DeadDomains() {
		r.deadDomains.Store(d, "dns_nxdomain")
	}
	for d := range dns.TimeoutDomains() {
		r.deadDomains.Store(d, "dns_timeout")
	}
}

func (r *Recrawler) isDomainDead(domain string) bool {
	_, dead := r.deadDomains.Load(domain)
	return dead
}

// domainDeadReason returns the reason a domain was marked dead, or "".
func (r *Recrawler) domainDeadReason(domain string) string {
	v, ok := r.deadDomains.Load(domain)
	if !ok {
		return ""
	}
	return v.(string)
}

func (r *Recrawler) markDomainDead(domain, reason string) {
	r.deadDomains.Store(domain, reason)
}

// recordDomainTimeout increments the failure counter for a domain.
// If the domain has never succeeded and failures >= threshold, marks it dead.
// Uses sync.Map + atomic for lock-free operation at 50K+ workers.
func (r *Recrawler) recordDomainTimeout(domain string) {
	if _, ok := r.domainSucceeded.Load(domain); ok {
		return // domain has succeeded before, immune to timeout-kill
	}
	counter, _ := r.domainFailCounts.LoadOrStore(domain, &atomic.Int32{})
	fails := counter.(*atomic.Int32).Add(1)
	if int(fails) >= r.config.DomainFailThreshold {
		r.markDomainDead(domain, "http_timeout_killed")
		// Log to FailedDB
		ips := r.cachedIPsFor(domain)
		r.failedDB.AddDomain(FailedDomain{
			Domain: domain,
			Reason: "http_timeout_killed",
			Error:  fmt.Sprintf("%d consecutive timeouts", fails),
			IPs:    ips,
			Stage:  "http_worker",
		})
	}
}

// recordDomainSuccess marks a domain as having succeeded at least once.
// Succeeding domains are immune to timeout-based killing.
func (r *Recrawler) recordDomainSuccess(domain string) {
	r.domainSucceeded.Store(domain, true)
}

// cachedIPsFor returns comma-separated cached IPs for a domain, or "".
func (r *Recrawler) cachedIPsFor(domain string) string {
	r.dnsCacheMu.RLock()
	ips := r.dnsCache[domain]
	r.dnsCacheMu.RUnlock()
	if len(ips) == 0 {
		return ""
	}
	return strings.Join(ips, ",")
}

// HTTPDeadDomains returns domains marked dead during HTTP fetching.
// These are domains where TCP connection was refused/reset (not timeouts).
// Can be merged into DNS cache for reuse in subsequent runs.
func (r *Recrawler) HTTPDeadDomains() map[string]bool {
	result := make(map[string]bool)
	r.deadDomains.Range(func(key, _ any) bool {
		result[key.(string)] = true
		return true
	})
	return result
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

	// Pre-create domain semaphores (avoids lock contention during fetch)
	for d := range domainURLs {
		r.domainSems[d] = make(chan struct{}, r.config.MaxConnsPerDomain)
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
					r.markDomainDead(domain, "dns_nxdomain")
					r.stats.RecordDNSDead()
					for range urls {
						r.stats.RecordDomainSkip()
					}
					r.failedDB.AddDomain(FailedDomain{Domain: domain, Reason: "dns_nxdomain", URLCount: len(urls), Stage: "dns_pipeline"})
					continue
				}

				if len(ips) == 0 {
					// DNS timeout on ALL resolvers — mark dead, skip URLs
					r.markDomainDead(domain, "dns_timeout")
					r.stats.RecordDNSTimeout()
					for range urls {
						r.stats.RecordDomainSkip()
					}
					r.failedDB.AddDomain(FailedDomain{Domain: domain, Reason: "dns_timeout", URLCount: len(urls), Stage: "dns_pipeline"})
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
						r.markDomainDead(domain, "probe_unreachable")
						for range urls {
							r.stats.RecordDomainSkip()
						}
						r.failedDB.AddDomain(FailedDomain{
							Domain:   domain,
							Reason:   "probe_unreachable",
							IPs:      strings.Join(ips, ","),
							URLCount: len(urls),
							Stage:    "probe",
						})
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

// probeDomainStrict sends a HEAD request to check domain reachability.
// Conservative: returns true on timeout (server may be slow but alive).
// Only returns false on definitive connection failure (refused/reset/no route/DNS error).
// shardID distributes probes across all transport shards.
func (r *Recrawler) probeDomainStrict(ctx context.Context, probeURL string, shardID int) bool {
	probeTimeout := min(3*time.Second, r.config.Timeout)
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodHead, probeURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", r.config.UserAgent)

	client := r.clientForWorker(shardID)
	resp, err := client.Do(req)
	if err != nil {
		// Timeout → conservative: domain may be slow but alive
		if isTimeoutError(err) {
			return true
		}
		// Connection refused/reset/no route/DNS error → definitively dead
		return false
	}
	resp.Body.Close()
	return true
}

// directFeed probes domains and streams their URLs to workers immediately.
// Streaming: workers start fetching as soon as the first domain is probed alive,
// overlapping the probe phase with the fetch phase for maximum throughput.
// Probes are distributed across all transport shards (not just shard 0).
func (r *Recrawler) directFeed(ctx context.Context, domains []string, domainURLs map[string][]SeedURL, urlCh chan<- SeedURL) {
	defer close(urlCh)

	// Phase 1: bulk-skip DNS-dead domains
	var probeDomains []string
	for _, d := range domains {
		if r.isDomainDead(d) {
			r.stats.RecordDomainSkipBatch(len(domainURLs[d]))
			continue
		}
		probeDomains = append(probeDomains, d)
	}

	if len(probeDomains) == 0 {
		return
	}

	// Phase 2: streaming probe → immediate URL feed
	// Probe domains in parallel. As each domain is confirmed alive,
	// its URLs are pushed to workers immediately — no waiting for all probes.
	// 5000 probe workers (up from 2000) × distributed across transport shards.
	probeWorkers := min(len(probeDomains), 5000)
	domainCh := make(chan string, len(probeDomains))
	go func() {
		defer close(domainCh)
		for _, d := range probeDomains {
			select {
			case domainCh <- d:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect alive domains with their URLs (probe filters dead ones)
	type aliveDomain struct {
		domain string
		urls   []SeedURL
	}
	var aliveMu sync.Mutex
	var aliveList []aliveDomain

	var probeWg sync.WaitGroup
	for i := range probeWorkers {
		probeWg.Add(1)
		go func(workerID int) {
			defer probeWg.Done()
			for domain := range domainCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				urls := domainURLs[domain]
				if len(urls) == 0 {
					continue
				}

				alive := r.probeDomainStrict(ctx, urls[0].URL, workerID)
				if alive {
					r.stats.RecordProbeReachable()
					aliveMu.Lock()
					aliveList = append(aliveList, aliveDomain{domain, urls})
					aliveMu.Unlock()
				} else {
					r.stats.RecordProbeUnreachable()
					r.markDomainDead(domain, "probe_unreachable")
					r.stats.RecordDomainSkipBatch(len(urls))
					r.failedDB.AddDomain(FailedDomain{
						Domain:   domain,
						Reason:   "probe_unreachable",
						IPs:      r.cachedIPsFor(domain),
						URLCount: len(urls),
						Stage:    "probe",
					})
				}
			}
		}(i)
	}

	probeWg.Wait()

	// Interleave URLs across domains (round-robin) for even load distribution.
	// Without interleaving, all URLs for domain A are sent before domain B,
	// causing the per-domain semaphore (8 max) to serialize them.
	cursors := make([]int, len(aliveList))
	remaining := len(aliveList)
	for remaining > 0 {
		remaining = 0
		for i, ad := range aliveList {
			if cursors[i] < len(ad.urls) {
				select {
				case urlCh <- ad.urls[cursors[i]]:
					cursors[i]++
				case <-ctx.Done():
					return
				}
				if cursors[i] < len(ad.urls) {
					remaining++
				}
			}
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
			// Per-domain connection limit: prevents flooding individual servers
			sem := r.domainSem(seed.Domain)
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			r.fetchOne(ctx, client, seed)
			<-sem
		}
	}
}

// domainSem returns the per-domain semaphore channel.
// Fast path: RLock read from pre-created map (zero contention during fetch).
// Slow path: Lock + create for domains not in the initial seed set (rare).
func (r *Recrawler) domainSem(domain string) chan struct{} {
	r.domainSemsMu.RLock()
	sem := r.domainSems[domain]
	r.domainSemsMu.RUnlock()
	if sem != nil {
		return sem
	}
	r.domainSemsMu.Lock()
	sem, ok := r.domainSems[domain]
	if !ok {
		sem = make(chan struct{}, r.config.MaxConnsPerDomain)
		r.domainSems[domain] = sem
	}
	r.domainSemsMu.Unlock()
	return sem
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
		fetchMs := time.Since(start).Milliseconds()
		errStr := truncateStr(err.Error(), 200)
		r.rdb.Add(Result{
			URL:         seed.URL,
			Domain:      seed.Domain,
			FetchTimeMs: fetchMs,
			CrawledAt:   time.Now(),
			Error:       errStr,
		})
		// Log to FailedDB
		failReason := "http_error"
		if isTimeout {
			failReason = "http_timeout"
		} else if isConnectionRefused(err) {
			failReason = "http_refused"
		} else if isDNSError(err) {
			failReason = "http_dns_error"
		}
		r.failedDB.AddURL(FailedURL{
			URL:         seed.URL,
			Domain:      seed.Domain,
			Reason:      failReason,
			Error:       errStr,
			FetchTimeMs: fetchMs,
		})
		if isFatal {
			reason := "http_refused"
			if isDNSError(err) {
				reason = "http_dns_error"
			}
			r.markDomainDead(seed.Domain, reason)
			r.failedDB.AddDomain(FailedDomain{Domain: seed.Domain, Reason: reason, Error: truncateStr(err.Error(), 200), IPs: r.cachedIPsFor(seed.Domain), Stage: "http_worker"})
		} else if isTimeout {
			r.recordDomainTimeout(seed.Domain)
		}
		return
	}

	// Any HTTP response means server is alive
	r.recordDomainSuccess(seed.Domain)

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
			r.failedDB.AddURL(FailedURL{
				URL:         seed.URL,
				Domain:      seed.Domain,
				Reason:      fmt.Sprintf("http_%d", resp.StatusCode),
				StatusCode:  resp.StatusCode,
				FetchTimeMs: fetchMs,
				ContentType: resp.Header.Get("Content-Type"),
				RedirectURL: redirectURL,
			})
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
