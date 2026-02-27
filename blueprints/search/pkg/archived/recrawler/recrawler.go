package recrawler

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"maps"
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

// adaptiveTracker records successful response latencies and computes adaptive timeouts.
// Uses a lock-free histogram for P95 calculation at 50K+ worker concurrency.
type adaptiveTracker struct {
	// Histogram buckets: <100ms, <250ms, <500ms, <1000ms, <2000ms, <3500ms, <5000ms, >=5000ms
	buckets [8]atomic.Int64
	total   atomic.Int64
	maxMs   int64 // ceiling from config
}

var adaptiveEdges = [8]int64{100, 250, 500, 1000, 2000, 3500, 5000, 10000}

func (t *adaptiveTracker) record(ms int64) {
	t.total.Add(1)
	for i, edge := range adaptiveEdges {
		if ms < edge {
			t.buckets[i].Add(1)
			return
		}
	}
	t.buckets[len(t.buckets)-1].Add(1)
}

// timeout returns an adaptive timeout based on P95 × 2, or 0 if insufficient samples.
// Uses P95×2 (not ×3) to stay tight — alive servers rarely fluctuate beyond 2× P95.
// Kicks in after just 5 samples for fast adaptation on sparse-alive datasets.
func (t *adaptiveTracker) timeout(ceiling time.Duration) time.Duration {
	n := t.total.Load()
	if n < 5 {
		return 0 // not enough data
	}
	target := int64(float64(n) * 0.95)
	var cum int64
	for i, edge := range adaptiveEdges {
		cum += t.buckets[i].Load()
		if cum >= target {
			ms := edge * 2
			ms = max(ms, 500)                    // floor: 500ms
			ms = min(ms, ceiling.Milliseconds()) // ceiling: config timeout
			return time.Duration(ms) * time.Millisecond
		}
	}
	return ceiling
}

// p95Ms returns the current P95 latency in ms, or 0 if insufficient samples.
func (t *adaptiveTracker) p95Ms() int64 {
	n := t.total.Load()
	if n < 10 {
		return 0
	}
	target := int64(float64(n) * 0.95)
	var cum int64
	for i, edge := range adaptiveEdges {
		cum += t.buckets[i].Load()
		if cum >= target {
			return edge
		}
	}
	return adaptiveEdges[len(adaptiveEdges)-1]
}

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
	deadHosts   sync.Map // host → reason (string), used for DNS-prefetch host failures

	// Cached DNS: pre-resolved domain → IP for direct dialing
	dnsCache   map[string][]string
	dnsCacheMu sync.RWMutex

	// Per-domain connection limiter: prevents flooding individual servers
	// Pre-created in Run() to avoid mutex contention during fetch.
	domainSems   map[string]chan struct{}
	domainSemsMu sync.RWMutex

	// Warmup gate: before a domain has any successful response, only allow one
	// in-flight request. This avoids timeout storms caused by blasting unknown/
	// dead domains with high concurrency.
	domainWarmup sync.Map // domain -> chan struct{} (cap=1)

	// Domain warmup success tracking (registered-domain key).
	// Once a domain succeeds, warmup gate is bypassed for that domain.
	domainSucceeded sync.Map // domain → true

	// Host-level timeout tracking: kill only the specific hostname that times out.
	// This avoids one slow subdomain killing the whole registered domain.
	hostFailCounts sync.Map // host → *atomic.Int32
	hostSucceeded  sync.Map // host → true

	// DNS resolver for pipelined mode (resolve + fetch concurrently)
	dnsResolver *DNSResolver

	// Adaptive timeout: probe unknown domains with short timeout,
	// then adjust based on observed latencies.
	adaptive     adaptiveTracker
	probeTimeout time.Duration
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

	// Probe timeout: first request to each unknown domain.
	// Uses cfg.Timeout (default 5s) so slow-but-alive servers (responding in 3-5s)
	// are not incorrectly killed. Dead domains are still killed quickly via
	// DomainFailThreshold=2 (two consecutive timeouts → kill).
	probeTimeout := cfg.ProbeTimeout
	if probeTimeout == 0 {
		probeTimeout = cfg.Timeout
	}

	r := &Recrawler{
		config:       cfg,
		stats:        stats,
		rdb:          rdb,
		dnsCache:     make(map[string][]string),
		domainSems:   make(map[string]chan struct{}),
		probeTimeout: probeTimeout,
		adaptive:     adaptiveTracker{maxMs: cfg.Timeout.Milliseconds()},
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

	// Do not split the user's timeout budget in half for connect/TLS:
	// with low timeouts (e.g. 1000-1400ms) that creates premature failures
	// before the overall request timeout is reached.
	dialTimeout := min(cfg.Timeout, 2*time.Second)
	tlsTimeout := min(cfg.Timeout, 2*time.Second)

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
			// Prefer IPv4 on hosts without working IPv6 routes and try multiple cached
			// IPs before failing to avoid false negatives from a single bad record.
			ordered := orderDialIPs(ips, shardID)
			var lastErr error
			for _, ip := range ordered {
				conn, derr := baseDialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
				if derr == nil {
					return conn, nil
				}
				lastErr = derr
				// If IPv6 is unreachable on this host, continue trying cached IPv4s.
				if strings.Contains(strings.ToLower(derr.Error()), "network is unreachable") {
					continue
				}
			}
			if lastErr != nil {
				// If the cached set appears IPv6-only on an IPv4-only host, fall back to
				// hostname dialing so the resolver can still return an A record.
				if strings.Contains(strings.ToLower(lastErr.Error()), "network is unreachable") {
					if conn, derr := baseDialer.DialContext(ctx, network, addr); derr == nil {
						return conn, nil
					}
				}
				return nil, lastErr
			}
		}

		return baseDialer.DialContext(ctx, network, addr)
	}

	transport := &http.Transport{
		DialContext: dialFunc,
		// InsecureSkipVerify: macOS TLS cert verification uses SecTrustEvaluate (CGO).
		// At 50K concurrent TLS handshakes, Go creates 50K OS threads for CGO calls,
		// exhausting the macOS per-process thread limit (pthread_create fails).
		// Using pure Go TLS (InsecureSkipVerify=true) eliminates CGO entirely.
		// Acceptable for a content crawler: we verify content, not server identity.
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:          maxIdlePerShard,
		MaxIdleConnsPerHost:   50,
		MaxConnsPerHost:       0,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   tlsTimeout,
		ResponseHeaderTimeout: cfg.Timeout,
		DisableCompression:    true,
		ForceAttemptHTTP2:     false, // HTTP/1.1 is faster for many-host one-shot fetches
		WriteBufferSize:       4 * 1024,
		ReadBufferSize:        8 * 1024,
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

func orderDialIPs(ips []string, shardID int) []string {
	if len(ips) <= 1 {
		return ips
	}
	var v4, v6 []string
	for _, ip := range ips {
		if strings.Contains(ip, ":") {
			v6 = append(v6, ip)
		} else {
			v4 = append(v4, ip)
		}
	}
	rotate := func(in []string) []string {
		if len(in) <= 1 {
			return in
		}
		out := make([]string, 0, len(in))
		start := shardID % len(in)
		out = append(out, in[start:]...)
		out = append(out, in[:start]...)
		return out
	}
	out := make([]string, 0, len(ips))
	out = append(out, rotate(v4)...)
	out = append(out, rotate(v6)...)
	return out
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

// SetDeadHosts pre-populates dead hosts (typically from DNS batch pre-resolution).
// Keys should be URL hostnames, not registered domains.
func (r *Recrawler) SetDeadHosts(hosts map[string]string) {
	for h, reason := range hosts {
		r.deadHosts.Store(h, reason)
	}
}

// SetDNSCache populates the cached DNS map for direct-IP dialing.
func (r *Recrawler) SetDNSCache(resolved map[string][]string) {
	r.dnsCacheMu.Lock()
	maps.Copy(r.dnsCache, resolved)
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
		maps.Copy(r.dnsCache, resolved)
		r.dnsCacheMu.Unlock()
	}
	// Pre-populate dead domains with reasons
	for d := range dns.DeadDomains() {
		r.deadHosts.Store(d, "dns_nxdomain")
	}
	for d := range dns.TimeoutDomains() {
		r.deadHosts.Store(d, "dns_timeout")
	}
}

func (r *Recrawler) isDomainDead(domain string) bool {
	_, dead := r.deadDomains.Load(domain)
	return dead
}

func (r *Recrawler) isHostDead(host string) bool {
	if host == "" {
		return false
	}
	_, dead := r.deadHosts.Load(host)
	return dead
}

// hostDeadReason returns the reason a host was marked dead, or "".
func (r *Recrawler) hostDeadReason(host string) string {
	if host == "" {
		return ""
	}
	v, ok := r.deadHosts.Load(host)
	if !ok {
		return ""
	}
	return v.(string)
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

func (r *Recrawler) markHostDead(host, reason string) {
	if host == "" {
		return
	}
	r.deadHosts.Store(host, reason)
}

// markDomainDeadIfUnset marks a domain dead only if it has not already been marked.
// Returns true when the domain was newly marked in this call.
func (r *Recrawler) markDomainDeadIfUnset(domain, reason string) bool {
	_, loaded := r.deadDomains.LoadOrStore(domain, reason)
	return !loaded
}

// markHostDeadIfUnset marks a host dead only if it has not already been marked.
func (r *Recrawler) markHostDeadIfUnset(host, reason string) bool {
	if host == "" {
		return false
	}
	_, loaded := r.deadHosts.LoadOrStore(host, reason)
	return !loaded
}

// recordHostTimeout increments the failure counter for a host.
// If the host has never succeeded and failures >= threshold, marks only that host dead.
func (r *Recrawler) recordHostTimeout(host string) {
	if host == "" {
		return
	}
	if r.isHostDead(host) {
		return
	}
	if _, ok := r.hostSucceeded.Load(host); ok {
		return // host has succeeded before, immune to timeout-kill
	}
	counter, _ := r.hostFailCounts.LoadOrStore(host, &atomic.Int32{})
	fails := counter.(*atomic.Int32).Add(1)
	if int(fails) >= r.config.DomainFailThreshold {
		if r.markHostDeadIfUnset(host, "http_timeout_killed") {
			// Log once per host (stored in failed_domains for compatibility).
			ips := r.cachedIPsForHost(host)
			r.failedDB.AddDomain(FailedDomain{
				Domain: host,
				Reason: "http_timeout_killed",
				Error:  fmt.Sprintf("%d consecutive timeouts", fails),
				IPs:    ips,
				Stage:  "http_worker",
			})
		}
	}
}

// recordDomainSuccess marks a domain as having succeeded at least once.
// Succeeding domains are immune to timeout-based killing.
func (r *Recrawler) recordDomainSuccess(domain string) {
	r.domainSucceeded.Store(domain, true)
}

// recordHostSuccess marks a host as having succeeded at least once.
func (r *Recrawler) recordHostSuccess(host string) {
	if host == "" {
		return
	}
	r.hostSucceeded.Store(host, true)
}

// cachedIPsForHost returns comma-separated cached IPs for a host, or "".
func (r *Recrawler) cachedIPsForHost(host string) string {
	r.dnsCacheMu.RLock()
	ips := r.dnsCache[host]
	r.dnsCacheMu.RUnlock()
	if len(ips) == 0 {
		return ""
	}
	return strings.Join(ips, ",")
}

// cachedIPsFor is kept as a compatibility wrapper (keys are now typically hosts).
func (r *Recrawler) cachedIPsFor(key string) string {
	return r.cachedIPsForHost(key)
}

// probeOutcome classifies the result of a TCP probe attempt.
type probeOutcome int8

const (
	probeOK      probeOutcome = iota // TCP connect succeeded
	probeRefused                     // connection refused (RST) — port is definitively closed
	probeTimeout                     // timed out — host may exist but port is filtered/slow
	probeError                       // other error (no route, DNS failure, etc.)
)

// tcpDialOne attempts a single TCP connect to addr:port with the given timeout.
// Returns nil error on success.
func tcpDialOne(ctx context.Context, addr, port string, timeout time.Duration) error {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(addr, port))
	if err == nil {
		conn.Close()
	}
	return err
}

// classifyDialError maps a dial error to probeOutcome (refused/timeout/error).
func classifyDialError(err error) probeOutcome {
	if strings.Contains(err.Error(), "connection refused") {
		return probeRefused
	}
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		return probeTimeout
	}
	return probeError
}

// tcpProbeHostPort performs a TCP connect probe to host:port.
//
// Enhancement: tries ALL cached IPv4 addresses in parallel (happy eyeballs).
// Returns probeOK as soon as any IP succeeds; reports the worst outcome
// (timeout > refused > error) if all IPs fail.
//
// Why try all IPs: round-robin DNS returns multiple A records. One IP may be
// geo-blocked or dead while another is alive — only trying ips[0] misses these.
func (r *Recrawler) tcpProbeHostPort(ctx context.Context, host, port, domain string, timeout time.Duration) probeOutcome {
	if host == "" {
		return probeError
	}
	// Collect all cached IPs (prefer IPv4, include IPv6 as fallback)
	r.dnsCacheMu.RLock()
	ips := r.dnsCache[host]
	if len(ips) == 0 {
		ips = r.dnsCache[domain]
	}
	r.dnsCacheMu.RUnlock()

	// Partition into IPv4 (preferred) and IPv6 (fallback)
	var ipv4s, ipv6s []string
	for _, ip := range ips {
		if strings.Contains(ip, ":") {
			ipv6s = append(ipv6s, ip)
		} else {
			ipv4s = append(ipv4s, ip)
		}
	}
	// Build probe list: all IPv4 first, IPv6 only if no IPv4 exists
	var addrs []string
	if len(ipv4s) > 0 {
		addrs = ipv4s
	} else if len(ipv6s) > 0 {
		addrs = ipv6s
	} else {
		addrs = []string{host} // no cache: fall back to runtime DNS via hostname
	}

	// Happy eyeballs: probe all addrs in parallel within shared timeout context.
	// First success wins; all goroutines cancelled on first OK.
	subCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct{ err error }
	resultCh := make(chan result, len(addrs))
	for _, addr := range addrs {
		go func(addr string) {
			resultCh <- result{tcpDialOne(subCtx, addr, port, timeout)}
		}(addr)
	}

	// Collect results; return OK as soon as any IP succeeds.
	worstOutcome := probeError
	for range len(addrs) {
		res := <-resultCh
		if res.err == nil {
			cancel() // cancel remaining goroutines
			return probeOK
		}
		out := classifyDialError(res.err)
		// Keep worst outcome: timeout > refused > error
		if out == probeTimeout || (out == probeRefused && worstOutcome == probeError) {
			worstOutcome = out
		}
	}
	return worstOutcome
}

// freshDNSLookup resolves host live (bypassing cache), returning only IPv4 addrs.
// Used to recover domains where the cached IP has become stale.
func freshDNSLookup(ctx context.Context, host string) []string {
	resolver := &net.Resolver{}
	subCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	addrs, err := resolver.LookupIPAddr(subCtx, host)
	if err != nil {
		return nil
	}
	var ipv4s []string
	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			ipv4s = append(ipv4s, addr.IP.String())
		}
	}
	return ipv4s
}

// alternatePort returns the alternate HTTP/HTTPS port for 80/443 pairs.
// Returns "" for non-standard ports (no alternate).
func alternatePort(port string) string {
	switch port {
	case "443":
		return "80"
	case "80":
		return "443"
	}
	return ""
}

// hostPortFromURL extracts host and port from a URL string.
func hostPortFromURL(urlStr string) (host, port string) {
	rest := urlStr
	scheme := "https"
	if strings.HasPrefix(rest, "http://") {
		scheme = "http"
		rest = rest[7:]
	} else if strings.HasPrefix(rest, "https://") {
		rest = rest[8:]
	}
	if idx := strings.IndexByte(rest, '/'); idx >= 0 {
		rest = rest[:idx]
	}
	host = rest
	port = "443"
	if scheme == "http" {
		port = "80"
	}
	if idx := strings.LastIndexByte(host, ':'); idx >= 0 {
		port = host[idx+1:]
		host = host[:idx]
	}
	return host, port
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

// DeadDomainReasons returns domains marked dead and their reasons.
func (r *Recrawler) DeadDomainReasons() map[string]string {
	result := make(map[string]string)
	r.deadDomains.Range(func(key, value any) bool {
		domain, _ := key.(string)
		reason, _ := value.(string)
		result[domain] = reason
		return true
	})
	return result
}

// DeadHostReasons returns hosts marked dead and their reasons.
func (r *Recrawler) DeadHostReasons() map[string]string {
	result := make(map[string]string)
	r.deadHosts.Range(func(key, value any) bool {
		host, _ := key.(string)
		reason, _ := value.(string)
		result[host] = reason
		return true
	})
	return result
}

// SucceededDomains returns domains that had at least one successful HTTP response.
func (r *Recrawler) SucceededDomains() map[string]bool {
	result := make(map[string]bool)
	r.domainSucceeded.Range(func(key, value any) bool {
		domain, _ := key.(string)
		ok, _ := value.(bool)
		if ok {
			result[domain] = true
		}
		return true
	})
	return result
}

// SucceededHosts returns hosts that had at least one successful HTTP response.
func (r *Recrawler) SucceededHosts() map[string]bool {
	result := make(map[string]bool)
	r.hostSucceeded.Range(func(key, value any) bool {
		host, _ := key.(string)
		ok, _ := value.(bool)
		if ok {
			result[host] = true
		}
		return true
	})
	return result
}

// clientForWorker returns the HTTP client for a worker ID (sharded).
func (r *Recrawler) clientForWorker(workerID int) *http.Client {
	return r.clients[workerID%len(r.clients)]
}

type tcpProbeDomainEntry struct {
	domain string
	urls   []SeedURL
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
	feedDone := make(chan struct{})

	if r.dnsResolver != nil {
		// Pipelined: DNS workers resolve domains and feed URLs to fetch pipeline
		go func() {
			defer close(feedDone)
			r.dnsPipeline(pipeCtx, domains, domainURLs, urlCh)
		}()
	} else {
		// Direct: pre-filter dead domains, shuffle, and feed URLs
		go func() {
			defer close(feedDone)
			r.directFeed(pipeCtx, domains, domainURLs, urlCh)
		}()
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

	// Tail-timeout: cancel remaining work when progress stalls.
	// After 95% completion, if fewer than 50 new results arrive in 30s,
	// the remaining connections are stuck (slow body reads, half-open TCP).
	// Cancel the context to force-close them instead of waiting indefinitely.
	go func() {
		threshold95 := int64(float64(totalLive) * 0.95)
		stallLimit := 30 * time.Second
		checkInterval := 5 * time.Second
		lastDone := r.stats.Done()
		lastProgress := time.Now()

		for {
			select {
			case <-gCtx.Done():
				return
			case <-time.After(checkInterval):
			}

			done := r.stats.Done()
			if done < threshold95 {
				lastDone = done
				lastProgress = time.Now()
				continue
			}
			if done > lastDone {
				lastDone = done
				lastProgress = time.Now()
				continue
			}
			if time.Since(lastProgress) >= stallLimit {
				pipeCancel()
				return
			}
		}
	}()

	err := g.Wait()
	pipeCancel() // ensure DNS pipeline stops if still running
	<-feedDone
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
		wg.Go(func() {
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
						r.stats.RecordDomainSkipReason("dns_nxdomain")
					}
					r.failedDB.AddDomain(FailedDomain{Domain: domain, Reason: "dns_nxdomain", URLCount: len(urls), Stage: "dns_pipeline"})
					continue
				}

				if len(ips) == 0 {
					// DNS timeout on ALL resolvers — mark dead, skip URLs
					r.markDomainDead(domain, "dns_timeout")
					r.stats.RecordDNSTimeout()
					for range urls {
						r.stats.RecordDomainSkipReason("dns_timeout")
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
							r.stats.RecordDomainSkipReason("probe_unreachable")
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
		})
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

// probeDomainHTTPReady checks whether the origin returns any HTTP response quickly.
// Returns:
//   - ok=true if the domain should proceed to the normal worker path
//   - hardReject=true only for definitive connection failures (refused / DNS)
//   - errMsg with the probe error for diagnostics when not ok
//
// To avoid false domain-dead classifications on constrained hosts, transient
// and timeout probe failures are treated as "ok" and left to worker retries.
func (r *Recrawler) probeDomainHTTPReady(ctx context.Context, probeURL string, shardID int) (ok bool, hardReject bool, errMsg string) {
	probeTimeout := min(3*time.Second, r.config.Timeout)
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodHead, probeURL, nil)
	if err != nil {
		return false, false, truncateStr(err.Error(), 200)
	}
	req.Header.Set("User-Agent", r.config.UserAgent)

	client := r.clientForWorker(shardID)
	resp, err := client.Do(req)
	if err != nil {
		// Retry once on transient/probe handshake issues using a different transport shard.
		if shouldRetryProbeRequestError(err) {
			retryClient := r.alternateClient(client)
			if retryClient == nil {
				retryClient = client
			}
			retryCtx, retryCancel := context.WithTimeout(ctx, r.config.Timeout)
			req2, reqErr2 := http.NewRequestWithContext(retryCtx, http.MethodHead, probeURL, nil)
			if reqErr2 == nil {
				req2.Header.Set("User-Agent", r.config.UserAgent)
				if resp2, err2 := retryClient.Do(req2); err2 == nil {
					retryCancel()
					resp2.Body.Close()
					return true, false, ""
				} else {
					err = err2
				}
			}
			retryCancel()
		}
		// Only definitive connection refusal / DNS failures should hard-reject a domain.
		if isConnectionRefused(err) || isDNSError(err) {
			return false, true, truncateStr(err.Error(), 200)
		}
		// Timeout/transient/TLS oddities are allowed through; the worker path has
		// better retries and per-host timeout thresholds.
		return true, false, truncateStr(err.Error(), 200)
	}
	resp.Body.Close()
	return true, false, ""
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
			r.stats.RecordDomainSkipBatchReason(r.domainDeadReason(d), len(domainURLs[d]))
			continue
		}
		probeDomains = append(probeDomains, d)
	}

	if len(probeDomains) == 0 {
		return
	}

	// Fast path: when TwoPass probing is disabled (default for cc recrawl),
	// use a lightweight TCP connect pre-check to filter truly dead hosts.
	// TCP SYN probe is ~100× cheaper than HTTP (no TLS, no headers):
	//   Dead hosts (SYN black hole) → detected in ≤1s
	//   Refused hosts → detected instantly (RST)
	//   Alive hosts → connect in ~50-300ms
	if !r.config.TwoPass {
		// Phase 1b: Filter host-level dead from DNS batch.
		// directFeed Phase 1 only checks isDomainDead (deadDomains map),
		// but the CLI stores DNS dead/timeout in deadHosts via SetDeadHosts.
		// This batch-skips host-dead domains instead of per-URL skip in workers.
		var toProbe []tcpProbeDomainEntry
		for _, domain := range probeDomains {
			urls := domainURLs[domain]
			if len(urls) == 0 {
				continue
			}
			hostKey := strings.TrimSpace(urls[0].Host)
			if hostKey == "" {
				hostKey = strings.TrimSpace(urls[0].Domain)
			}
			if r.isHostDead(hostKey) {
				r.stats.RecordDomainSkipBatchReason(r.hostDeadReason(hostKey), len(urls))
				continue
			}
			toProbe = append(toProbe, tcpProbeDomainEntry{domain, urls})
		}

		if len(toProbe) == 0 {
			return
		}

		// Fast path: domains with pre-cached DNS IPs skip TCP probe entirely.
		// Batch DNS pre-resolution already confirmed these hosts have valid A records;
		// TCP probe would be redundant overhead (20-50s for 2K+ domains across 4 passes).
		// HTTP workers handle TCP-unreachable cases via the warmup semaphore (cap-1 per
		// domain on first request) + DomainFailThreshold kill after N consecutive timeouts.
		type dnsProbeAlive struct {
			domain string
			urls   []SeedURL
		}
		var dnsAlive []dnsProbeAlive
		var needsProbe []tcpProbeDomainEntry
		for _, entry := range toProbe {
			host := strings.TrimSpace(entry.urls[0].Host)
			if host == "" {
				host = strings.TrimSpace(entry.urls[0].Domain)
			}
			r.dnsCacheMu.RLock()
			hasDNS := len(r.dnsCache[host]) > 0 || len(r.dnsCache[entry.domain]) > 0
			r.dnsCacheMu.RUnlock()
			if hasDNS {
				r.stats.RecordProbeReachable()
				dnsAlive = append(dnsAlive, dnsProbeAlive{entry.domain, entry.urls})
			} else {
				needsProbe = append(needsProbe, entry)
			}
		}

		// Feed DNS-confirmed-alive domains immediately (interleaved round-robin).
		// Round-robin across domains prevents per-domain semaphore starvation:
		// without interleaving, all URLs for domain A occupy the 8-conn cap before
		// domain B gets any workers, serializing fetches unnecessarily.
		if len(dnsAlive) > 0 {
			cursors := make([]int, len(dnsAlive))
			remaining := len(dnsAlive)
			for remaining > 0 {
				remaining = 0
				for i, ad := range dnsAlive {
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

		// TCP probe remaining domains that lack a DNS cache entry (rare after DNS prefetch).
		const probeChunkSize = 1000
		for start := 0; start < len(needsProbe); start += probeChunkSize {
			end := min(start+probeChunkSize, len(needsProbe))
			if !r.tcpProbeAndFeedChunk(ctx, needsProbe[start:end], urlCh) {
				return
			}
		}
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
					r.stats.RecordDomainSkipBatchReason("probe_unreachable", len(urls))
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

// tcpProbeAndFeedChunk classifies a batch of domains via multi-pass TCP probing,
// records stats/dead-host metadata, and immediately feeds alive URLs to workers.
// Returns false if the context was canceled while feeding.
func (r *Recrawler) tcpProbeAndFeedChunk(ctx context.Context, toProbe []tcpProbeDomainEntry, urlCh chan<- SeedURL) bool {
	if len(toProbe) == 0 {
		return true
	}

	alive := make([]bool, len(toProbe))
	outcomes := make([]probeOutcome, len(toProbe))
	// origOutcomes tracks outcomes on the ORIGINAL port only.
	// Alt-port failures are excluded so a refused port 80 cannot condemn a domain
	// whose port 443 is merely slow (root cause of most false tcp_unreachable).
	origOutcomes := make([]probeOutcome, len(toProbe))
	httpRejectedHard := make([]bool, len(toProbe))
	httpRejectErr := make([]string, len(toProbe))

	// Pass 1: original port, 3s timeout.
	{
		pass1Timeout := 3 * time.Second
		nWorkers := min(len(toProbe), 500)
		idxCh := make(chan int, max(nWorkers*4, 1))
		go func() {
			defer close(idxCh)
			for i := range toProbe {
				select {
				case idxCh <- i:
				case <-ctx.Done():
					return
				}
			}
		}()
		var wg sync.WaitGroup
		for range nWorkers {
			wg.Go(func() {
				for idx := range idxCh {
					select {
					case <-ctx.Done():
						return
					default:
					}
					e := toProbe[idx]
					host, port := hostPortFromURL(e.urls[0].URL)
						out := r.tcpProbeHostPort(ctx, host, port, e.domain, pass1Timeout)
						if out == probeOK {
							alive[idx] = true
						} else {
							outcomes[idx] = mergeProbeOutcome(outcomes[idx], out)
							origOutcomes[idx] = mergeProbeOutcome(origOutcomes[idx], out)
						}
				}
			})
		}
		wg.Wait()
	}

	// Pass 2: alternate port + original-port retry.
	{
		type retryEntry struct {
			idx     int
			altPort string
		}
		var retryList []retryEntry
		for i := range toProbe {
			if !alive[i] {
				_, origPort := hostPortFromURL(toProbe[i].urls[0].URL)
				retryList = append(retryList, retryEntry{i, alternatePort(origPort)})
			}
		}
		if len(retryList) > 0 {
			pass2Timeout := 1 * time.Second
			nWorkers := min(len(retryList), 500)
			retryCh := make(chan retryEntry, max(nWorkers*4, 1))
			go func() {
				defer close(retryCh)
				for _, re := range retryList {
					select {
					case retryCh <- re:
					case <-ctx.Done():
						return
					}
				}
			}()
			var wg sync.WaitGroup
			for range nWorkers {
				wg.Go(func() {
					for re := range retryCh {
						select {
						case <-ctx.Done():
							return
						default:
						}
						e := toProbe[re.idx]
						host, origPort := hostPortFromURL(e.urls[0].URL)
							if re.altPort != "" {
								if out := r.tcpProbeHostPort(ctx, host, re.altPort, e.domain, pass2Timeout); out == probeOK {
									alive[re.idx] = true
									continue
								} else {
									// Alt-port failures only count toward diagnostics.
									// They must NOT update origOutcomes: a refused port 80 must not
									// condemn a domain whose port 443 merely timed out.
									outcomes[re.idx] = mergeProbeOutcome(outcomes[re.idx], out)
								}
							}
							if out := r.tcpProbeHostPort(ctx, host, origPort, e.domain, pass2Timeout); out == probeOK {
								alive[re.idx] = true
							} else {
								outcomes[re.idx] = mergeProbeOutcome(outcomes[re.idx], out)
								origOutcomes[re.idx] = mergeProbeOutcome(origOutcomes[re.idx], out)
							}
					}
				})
			}
			wg.Wait()
		}
	}

	// Pass 3: long-timeout retries on both ports.
	for round := 0; ; round++ {
		var hardFails []int
		for i := range toProbe {
			if !alive[i] {
				hardFails = append(hardFails, i)
			}
		}
		if len(hardFails) == 0 {
			break
		}
		pass3Timeout := 8 * time.Second
		nWorkers := min(len(hardFails), 200)
		idxCh := make(chan int, max(nWorkers*4, 1))
		go func() {
			defer close(idxCh)
			for _, i := range hardFails {
				select {
				case idxCh <- i:
				case <-ctx.Done():
					return
				}
			}
		}()
		var newAlive atomic.Int64
		var wg sync.WaitGroup
		for range nWorkers {
			wg.Go(func() {
				for idx := range idxCh {
					select {
					case <-ctx.Done():
						return
					default:
					}
					e := toProbe[idx]
					host, origPort := hostPortFromURL(e.urls[0].URL)
					altPort := alternatePort(origPort)
						if out := r.tcpProbeHostPort(ctx, host, origPort, e.domain, pass3Timeout); out == probeOK {
							alive[idx] = true
							newAlive.Add(1)
							continue
						} else {
							outcomes[idx] = mergeProbeOutcome(outcomes[idx], out)
							origOutcomes[idx] = mergeProbeOutcome(origOutcomes[idx], out)
						}
						if altPort != "" {
							if out := r.tcpProbeHostPort(ctx, host, altPort, e.domain, pass3Timeout); out == probeOK {
								alive[idx] = true
								newAlive.Add(1)
							} else {
								// Alt-port only: do NOT update origOutcomes here.
								outcomes[idx] = mergeProbeOutcome(outcomes[idx], out)
							}
						}
				}
			})
		}
		wg.Wait()
		if newAlive.Load() == 0 || round >= 2 {
			break
		}
	}

	// Pass 4: DNS re-resolution for stale cached IPs.
	{
		var staleFails []int
		for i := range toProbe {
			if !alive[i] {
				staleFails = append(staleFails, i)
			}
		}
		if len(staleFails) > 0 {
			pass4Timeout := 3 * time.Second
			nWorkers := min(len(staleFails), 200)
			idxCh := make(chan int, max(nWorkers*4, 1))
			go func() {
				defer close(idxCh)
				for _, i := range staleFails {
					select {
					case idxCh <- i:
					case <-ctx.Done():
						return
					}
				}
			}()
			var wg sync.WaitGroup
			for range nWorkers {
				wg.Go(func() {
					for idx := range idxCh {
						select {
						case <-ctx.Done():
							return
						default:
						}
						e := toProbe[idx]
						host, origPort := hostPortFromURL(e.urls[0].URL)
						freshIPs := freshDNSLookup(ctx, host)
						if len(freshIPs) == 0 {
							continue
						}
						r.dnsCacheMu.RLock()
						cachedIPs := r.dnsCache[host]
						r.dnsCacheMu.RUnlock()
						hasNew := false
						cachedSet := make(map[string]bool, len(cachedIPs))
						for _, ip := range cachedIPs {
							cachedSet[ip] = true
						}
						for _, ip := range freshIPs {
							if !cachedSet[ip] {
								hasNew = true
								break
							}
						}
						if !hasNew {
							continue
						}
						altPort := alternatePort(origPort)
						for _, ip := range freshIPs {
							if tcpDialOne(ctx, ip, origPort, pass4Timeout) == nil {
								alive[idx] = true
								break
							}
							if altPort != "" && tcpDialOne(ctx, ip, altPort, pass4Timeout) == nil {
								alive[idx] = true
								break
							}
						}
					}
				})
			}
			wg.Wait()
		}
	}

	// Pass 5: HTTP confirmation for original-port-refused domains.
	// Root cause #2: some servers/CDNs RST raw TCP on port 443 without a TLS ClientHello
	// (e.g. nginx ssl_reject_handshake, AWS ALB strict TLS) but serve HTTPS fine when a
	// real HTTP client sends a ClientHello. An HTTP HEAD probe confirms whether the domain
	// is truly dead before we write it off as tcp_unreachable.
	{
		var confirmList []int
		for i := range toProbe {
			if !alive[i] && !httpRejectedHard[i] && origOutcomes[i] == probeRefused {
				confirmList = append(confirmList, i)
			}
		}
		if len(confirmList) > 0 {
			nWorkers := min(len(confirmList), 200)
			idxCh := make(chan int, max(nWorkers*4, 1))
			go func() {
				defer close(idxCh)
				for _, i := range confirmList {
					select {
					case idxCh <- i:
					case <-ctx.Done():
						return
					}
				}
			}()
			var wg sync.WaitGroup
			for workerID := range nWorkers {
				id := workerID
				wg.Go(func() {
					for idx := range idxCh {
						select {
						case <-ctx.Done():
							return
						default:
						}
						ok, hardReject, errMsg := r.probeDomainHTTPReady(ctx, toProbe[idx].urls[0].URL, id)
						if ok {
							// HTTP succeeded despite raw TCP RST — TLS-gated server or CDN edge.
							alive[idx] = true
						} else if hardReject {
							// HTTP also hard-refused: definitively dead, classify as http_probe_unreachable.
							httpRejectedHard[idx] = true
							httpRejectErr[idx] = errMsg
						}
						// !ok && !hardReject (HTTP timeout): leave origOutcomes as probeRefused.
						// Domain is genuinely unreachable within time budget.
					}
				})
			}
			wg.Wait()
		}
	}

	if r.config.StatusOnly || r.config.HeadOnly {
		var candidates []int
		for i := range toProbe {
			if alive[i] {
				candidates = append(candidates, i)
			}
		}
		if len(candidates) > 0 {
			nWorkers := min(len(candidates), 500)
			idxCh := make(chan int, max(nWorkers*4, 1))
			go func() {
				defer close(idxCh)
				for _, i := range candidates {
					select {
					case idxCh <- i:
					case <-ctx.Done():
						return
					}
				}
			}()
			var wg sync.WaitGroup
			for workerID := range nWorkers {
				id := workerID
				wg.Go(func() {
					for idx := range idxCh {
						select {
						case <-ctx.Done():
							return
						default:
						}
							ok, hardReject, errMsg := r.probeDomainHTTPReady(ctx, toProbe[idx].urls[0].URL, id)
							if !ok {
								alive[idx] = false
								httpRejectedHard[idx] = hardReject
								httpRejectErr[idx] = errMsg
							}
						}
					})
			}
			wg.Wait()
		}
	}

	type aliveDomain struct {
		domain string
		urls   []SeedURL
	}
	var aliveList []aliveDomain
		for i, e := range toProbe {
		if alive[i] || (!httpRejectedHard[i] && origOutcomes[i] != probeRefused) {
			r.stats.RecordProbeReachable()
			aliveList = append(aliveList, aliveDomain{e.domain, e.urls})
			continue
		}
		hostKey := strings.TrimSpace(e.urls[0].Host)
		if hostKey == "" {
			hostKey = strings.TrimSpace(e.urls[0].Domain)
		}
		reason := "tcp_unreachable"
		stage := "tcp_probe"
		errMsg := ""
		if httpRejectedHard[i] {
			reason = "http_probe_unreachable"
			stage = "http_probe"
			errMsg = httpRejectErr[i]
		} else {
			errMsg = tcpProbeOutcomeError(outcomes[i], hostKey, e.domain)
		}
		r.markHostDeadIfUnset(hostKey, reason)
		r.markDomainDead(e.domain, reason)
		r.stats.RecordProbeUnreachable()
		r.stats.RecordDomainSkipBatchReason(reason, len(e.urls))
		r.failedDB.AddDomain(FailedDomain{
			Domain:   e.domain,
			Reason:   reason,
			Error:    truncateStr(errMsg, 200),
			IPs:      r.cachedIPsFor(e.domain),
			URLCount: len(e.urls),
			Stage:    stage,
		})
	}

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
					return false
				}
				if cursors[i] < len(ad.urls) {
					remaining++
				}
			}
		}
	}

	return true
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
			hostKey := strings.TrimSpace(seed.Host)
			if hostKey == "" {
				hostKey = strings.TrimSpace(seed.Domain)
			}
			if r.isDomainDead(seed.Domain) {
				r.stats.RecordDomainSkipReason(r.domainDeadReason(seed.Domain))
				continue
			}
			if r.isHostDead(hostKey) {
				r.stats.RecordDomainSkipReason(r.hostDeadReason(hostKey))
				continue
			}
			warmSem, needWarmup := r.domainWarmupSem(seed.Domain)
			if needWarmup {
				select {
				case warmSem <- struct{}{}:
				case <-ctx.Done():
					return ctx.Err()
				}
				// Domain may have been marked dead while waiting.
				if r.isDomainDead(seed.Domain) {
					<-warmSem
					r.stats.RecordDomainSkipReason(r.domainDeadReason(seed.Domain))
					continue
				}
				if r.isHostDead(hostKey) {
					<-warmSem
					r.stats.RecordDomainSkipReason(r.hostDeadReason(hostKey))
					continue
				}
			}
			// Per-domain connection limit: prevents flooding individual servers
			sem := r.domainSem(seed.Domain)
			select {
			case sem <- struct{}{}:
				if needWarmup {
					<-warmSem
				}
			case <-ctx.Done():
				if needWarmup {
					<-warmSem
				}
				return ctx.Err()
			}
			r.fetchOne(ctx, client, seed)
			<-sem
		}
	}
}

// domainWarmupSem returns a per-domain semaphore for domains that have not yet succeeded.
// Once a domain has any successful response, warmup is bypassed entirely.
//
// Cap=2 (not 1): allows 2 concurrent probes per new domain.
// Benefits vs cap=1:
//  1. Peak throughput: 2× more concurrent probes → 2× higher burst rate.
//  2. Faster dead-domain kill: both probes timeout simultaneously → fail=2 (threshold)
//     → domain killed in one timeout round (~3s) instead of two (~6s).
//     Workers move on to the next domain 3s sooner, improving overall efficiency.
func (r *Recrawler) domainWarmupSem(domain string) (chan struct{}, bool) {
	if _, ok := r.domainSucceeded.Load(domain); ok {
		return nil, false
	}
	v, _ := r.domainWarmup.LoadOrStore(domain, make(chan struct{}, 2))
	return v.(chan struct{}), true
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
	hostKey := strings.TrimSpace(seed.Host)
	if hostKey == "" {
		hostKey = strings.TrimSpace(seed.Domain)
	}

	// Adaptive timeout: three tiers for unknown vs known-good domains.
	//
	// Tier 1 (probe): Domain never succeeded → use probeTimeout (3s default).
	//   Single attempt, no retry. On timeout → recordHostTimeout (DomainFailThreshold
	//   consecutive timeouts = kill). On success → record latency, mark domain alive.
	//
	// Tier 2 (adaptive): Domain succeeded before, 5+ latency samples →
	//   use P95×2 of observed latencies (floor 500ms, ceiling config.Timeout).
	//   Adapts to dataset: fast servers get tight timeout, slow servers get slack.
	//
	// Tier 3 (config): Fallback to config.Timeout when not enough data.
	isProbe := false
	reqTimeout := r.config.Timeout
	if _, ok := r.domainSucceeded.Load(seed.Domain); !ok {
		isProbe = true
		reqTimeout = r.probeTimeout
		// In status/head modes many domains have only one URL, so the "probe"
		// request is the only request. Use the full timeout budget to avoid
		// over-classifying slow-but-alive hosts as timeouts.
		if r.config.StatusOnly || r.config.HeadOnly {
			reqTimeout = r.config.Timeout
		}
	} else if at := r.adaptive.timeout(r.config.Timeout); at > 0 {
		reqTimeout = at
	}
	reqCtx, reqCancel := context.WithTimeout(ctx, reqTimeout)
	defer reqCancel()

	method := http.MethodGet
	if r.config.HeadOnly {
		method = http.MethodHead
	}

	var (
		resp *http.Response
		err  error
	)
	// Probe requests: single attempt, no retry (fail fast on dead domains).
	// Non-probe: retry once on transient errors (server jitter).
	if isProbe {
		req, reqErr := http.NewRequestWithContext(reqCtx, method, seed.URL, nil)
		if reqErr != nil {
			r.recordError(seed, 0, start, reqErr)
			return
		}
		req.Header.Set("User-Agent", r.config.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")
		resp, err = client.Do(req)
		if err != nil && shouldRetryProbeRequestError(err) {
			retryClient := r.alternateClient(client)
			if retryClient == nil {
				retryClient = client
			}
			retryCtx, retryCancel := context.WithTimeout(ctx, r.config.Timeout)
			req2, reqErr2 := http.NewRequestWithContext(retryCtx, method, seed.URL, nil)
			if reqErr2 == nil {
				req2.Header.Set("User-Agent", r.config.UserAgent)
				req2.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")
				resp, err = retryClient.Do(req2)
			}
			retryCancel()
		}
	} else {
		clients := []*http.Client{client}
		if alt := r.alternateClient(client); alt != nil {
			clients = append(clients, alt)
		}
		for attempt, c := range clients {
			req, reqErr := http.NewRequestWithContext(reqCtx, method, seed.URL, nil)
			if reqErr != nil {
				r.recordError(seed, 0, start, reqErr)
				return
			}
			req.Header.Set("User-Agent", r.config.UserAgent)
			req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")

			resp, err = c.Do(req)
			if err == nil {
				break
			}
			if attempt == 0 && shouldRetryTransientRequestError(err) {
				select {
				case <-reqCtx.Done():
					goto requestDone
				case <-time.After(25 * time.Millisecond):
				}
				continue
			}
			break
		}
	}
requestDone:
	if err != nil {
		isTimeout := isTimeoutError(err)
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
			if r.markHostDeadIfUnset(hostKey, reason) {
				r.failedDB.AddDomain(FailedDomain{Domain: hostKey, Reason: reason, Error: truncateStr(err.Error(), 200), IPs: r.cachedIPsForHost(hostKey), Stage: "http_worker"})
			}
		} else if isTimeout {
			// All timeouts (probe and non-probe) go through the same threshold logic.
			// recordHostTimeout increments the per-host failure counter and kills the
			// host only after DomainFailThreshold consecutive timeouts (default 2).
			// This prevents one slow response from killing a domain with 100+ URLs.
			r.recordHostTimeout(hostKey)
		}
		return
	}

	// Any HTTP response means server is alive
	r.recordDomainSuccess(seed.Domain)
	r.recordHostSuccess(hostKey)

	// Record latency for adaptive timeout computation
	r.adaptive.record(time.Since(start).Milliseconds())

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
	bodySize := max(resp.ContentLength, int64(len(bodyBytes)))

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

func (r *Recrawler) alternateClient(current *http.Client) *http.Client {
	if len(r.clients) < 2 {
		return nil
	}
	for i, c := range r.clients {
		if c == current {
			return r.clients[(i+1)%len(r.clients)]
		}
	}
	// Fallback: a different shard than the caller likely used.
	return r.clients[1]
}

func shouldRetryTransientRequestError(err error) bool {
	if err == nil {
		return false
	}
	if isTimeoutError(err) {
		return true
	}
	// Retry a narrow set of transient transport errors.
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "temporary") ||
		strings.Contains(s, "can't assign requested address") ||
		strings.Contains(s, "server misbehaving") ||
		strings.Contains(s, "use of closed network connection")
}

func shouldRetryProbeRequestError(err error) bool {
	if err == nil {
		return false
	}
	if shouldRetryTransientRequestError(err) {
		return true
	}
	// Probe requests are the first contact with a host and account for most URLs
	// in sparse domain datasets. Retry a small set of handshake hiccups once.
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "remote error: tls: internal error") ||
		strings.Contains(s, "remote error: tls: handshake failure") ||
		strings.Contains(s, "remote error: tls: unrecognized name") ||
		strings.Contains(s, "tls handshake timeout") ||
		strings.Contains(s, "connection reset by peer") ||
		strings.Contains(s, "eof")
}

func mergeProbeOutcome(prev, next probeOutcome) probeOutcome {
	// Preserve the strongest evidence for a definitive dead classification.
	// refused > timeout > error.
	if next == probeRefused {
		return probeRefused
	}
	if prev == probeRefused {
		return prev
	}
	if next == probeTimeout {
		return probeTimeout
	}
	if prev == probeTimeout {
		return prev
	}
	if next == probeError {
		return probeError
	}
	return prev
}

func tcpProbeOutcomeError(out probeOutcome, hostKey, domain string) string {
	switch out {
	case probeRefused:
		return fmt.Sprintf("tcp probe refused (host=%s domain=%s)", hostKey, domain)
	case probeTimeout:
		return fmt.Sprintf("tcp probe timeout (host=%s domain=%s)", hostKey, domain)
	case probeError:
		return fmt.Sprintf("tcp probe error (host=%s domain=%s)", hostKey, domain)
	default:
		return ""
	}
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

// AdaptiveTimeoutInfo returns a formatted string describing current adaptive timeout state.
func (r *Recrawler) AdaptiveTimeoutInfo() string {
	samples := r.adaptive.total.Load()
	p95 := r.adaptive.p95Ms()
	at := r.adaptive.timeout(r.config.Timeout)
	if at == 0 || samples < 5 {
		return fmt.Sprintf("probe %v, adaptive pending (%d samples)", r.probeTimeout, samples)
	}
	return fmt.Sprintf("probe %v, adaptive %v (P95=%dms, %d samples)", r.probeTimeout, at, p95, samples)
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
				// Update adaptive timeout display info
				stats.adaptiveInfo.Store(r.AdaptiveTimeoutInfo())

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
