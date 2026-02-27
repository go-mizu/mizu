package recrawler

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/phuslu/fastdns"
)

const dnsShardCount = 64 // must be power of 2

const (
	fastDNSServerCount = 2
	// fastDNSConnsPerServer sets the UDP connection pool per server.
	// 4096 per server × 2 servers = 8192 pool → 6144 recommended worker cap.
	// This allows 4000+ workers for large batches (80K+ domains) while keeping
	// each server well under its rate limit (~50K QPS for Cloudflare/Google).
	fastDNSConnsPerServer   = 4096
	fastDNSWorkerStableFrac = 3 // use ~75% of pool capacity for lower timeout noise
)

// dnsShard is a single shard of the DNS cache, each with its own lock.
type dnsShard struct {
	mu       sync.RWMutex
	resolved map[string][]string // domain → IPs
	dead     map[string]string   // domain → error message (NXDOMAIN)
	timeout  map[string]string   // domain → error message (timeout/temp)
}

// DNSResolver performs parallel DNS pre-resolution for a set of domains.
// Uses sharded maps (64 shards) to eliminate global mutex contention.
// Multi-resolver strategy: tries system DNS first, then Google/Cloudflare as fallback.
// Only marks domain dead on definitive NXDOMAIN; retries on timeout/temporary errors.
// Results can be persisted to a DuckDB cache for instant reuse across runs.
type DNSResolver struct {
	resolvers []*net.Resolver // system, Google 8.8.8.8, Cloudflare 1.1.1.1
	shards    [dnsShardCount]dnsShard

	// Stats
	total    int
	ok       atomic.Int64
	failed   atomic.Int64
	timedOut atomic.Int64 // domains that timed out (all resolvers)
	cached   atomic.Int64 // loaded from cache

	duration time.Duration

	// Per-domain lookup timeout (configurable)
	lookupTimeout time.Duration
}

// makeResolver creates a net.Resolver that dials the given DNS server.
// If addr is empty, uses the system default resolver.
func makeResolver(addr string, timeout time.Duration) *net.Resolver {
	if addr == "" {
		return &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: timeout}
				return d.DialContext(ctx, "udp", address)
			},
		}
	}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "udp", addr)
		},
	}
}

// NewDNSResolver creates a DNS resolver with multi-server fallback.
// The timeout parameter controls both the dial timeout for each resolver
// and the per-domain lookup timeout.
func NewDNSResolver(timeout time.Duration) *DNSResolver {
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	d := &DNSResolver{
		resolvers: []*net.Resolver{
			makeResolver("", timeout),           // system DNS (fast for cached, leverages OS cache)
			makeResolver("8.8.8.8:53", timeout), // Google (fallback, high-concurrency)
			makeResolver("1.1.1.1:53", timeout), // Cloudflare (tertiary)
		},
		lookupTimeout: timeout,
	}
	for i := range d.shards {
		d.shards[i].resolved = make(map[string][]string)
		d.shards[i].dead = make(map[string]string)
		d.shards[i].timeout = make(map[string]string)
	}
	return d
}

// shardFor returns the shard index for a domain using FNV-1a hash.
func shardFor(domain string) int {
	h := uint32(2166136261)
	for i := 0; i < len(domain); i++ {
		h ^= uint32(domain[i])
		h *= 16777619
	}
	return int(h & (dnsShardCount - 1))
}

// LoadCache loads previously resolved DNS data from a DuckDB file.
// Timeout domains are loaded as dead-for-this-run (prevents re-resolving).
func (d *DNSResolver) LoadCache(dbPath string) (int, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, nil
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'dns'").Scan(&count)
	if err != nil || count == 0 {
		return 0, nil
	}

	// Check if timeout column exists (schema migration)
	var hasTimeout bool
	var colCount int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'dns' AND column_name = 'timeout'").Scan(&colCount)
	hasTimeout = err == nil && colCount > 0

	var query string
	if hasTimeout {
		query = "SELECT domain, COALESCE(ips, '') as ips, dead, COALESCE(error, '') as error, timeout FROM dns"
	} else {
		query = "SELECT domain, COALESCE(ips, '') as ips, dead, COALESCE(error, '') as error FROM dns"
	}

	rows, err := db.Query(query)
	if err != nil {
		return 0, nil
	}
	defer rows.Close()

	loaded := 0
	for rows.Next() {
		var domain, ips, errMsg string
		var dead, isTimeout bool
		var scanErr error
		if hasTimeout {
			scanErr = rows.Scan(&domain, &ips, &dead, &errMsg, &isTimeout)
		} else {
			scanErr = rows.Scan(&domain, &ips, &dead, &errMsg)
		}
		if scanErr != nil {
			continue
		}
		s := &d.shards[shardFor(domain)]
		s.mu.Lock()
		if dead {
			if errMsg == "http_dead" {
				s.mu.Unlock()
				continue // Re-resolve; HTTP failure != DNS dead
			}
			s.dead[domain] = errMsg
			d.failed.Add(1)
		} else if isTimeout {
			// Timeout domains treated as dead for this run
			s.timeout[domain] = errMsg
			d.timedOut.Add(1)
		} else {
			s.resolved[domain] = strings.Split(ips, ",")
			d.ok.Add(1)
		}
		s.mu.Unlock()
		loaded++
	}
	d.cached.Store(int64(loaded))
	return loaded, nil
}

// SaveCache persists DNS resolution results to a DuckDB file.
// Saves resolved, dead (NXDOMAIN), and timeout domains.
// Uses a temporary CSV file for bulk loading (10-100x faster than parameterized inserts).
func (d *DNSResolver) SaveCache(dbPath string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("opening dns cache db: %w", err)
	}
	defer db.Close()

	db.Exec("DROP TABLE IF EXISTS dns")
	_, err = db.Exec(`
		CREATE TABLE dns (
			domain VARCHAR,
			ips VARCHAR,
			dead BOOLEAN DEFAULT false,
			error VARCHAR DEFAULT '',
			timeout BOOLEAN DEFAULT false,
			resolved_at TIMESTAMP DEFAULT current_timestamp
		)
	`)
	if err != nil {
		return fmt.Errorf("creating dns table: %w", err)
	}

	// Write to temp CSV, then bulk load via DuckDB's COPY
	tmpFile := dbPath + ".tmp.csv"
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("creating temp csv: %w", err)
	}

	w := bufio.NewWriterSize(f, 1024*1024) // 1MB buffer
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		for domain, ips := range s.resolved {
			fmt.Fprintf(w, "%s\t%s\tfalse\t\tfalse\n", csvEscape(domain), csvEscape(strings.Join(ips, ",")))
		}
		for domain, errMsg := range s.dead {
			fmt.Fprintf(w, "%s\t\ttrue\t%s\tfalse\n", csvEscape(domain), csvEscape(errMsg))
		}
		for domain, errMsg := range s.timeout {
			fmt.Fprintf(w, "%s\t\tfalse\t%s\ttrue\n", csvEscape(domain), csvEscape(errMsg))
		}
		s.mu.RUnlock()
	}
	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("flush dns cache temp csv: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("close dns cache temp csv: %w", err)
	}

	// Bulk load via explicit-schema CSV reader to avoid brittle auto-detection/type inference.
	_, err = db.Exec(fmt.Sprintf(`INSERT INTO dns(domain, ips, dead, error, timeout)
SELECT
  NULLIF(domain, '') AS domain,
  NULLIF(ips, '') AS ips,
  lower(trim(dead)) = 'true' AS dead,
  COALESCE(error, '') AS error,
  lower(trim(timeout)) = 'true' AS timeout
FROM read_csv('%s',
  delim='\t',
  header=false,
  columns={'domain':'VARCHAR','ips':'VARCHAR','dead':'VARCHAR','error':'VARCHAR','timeout':'VARCHAR'},
  nullstr='')`, tmpFile))
	_ = os.Remove(tmpFile)
	if err != nil {
		return fmt.Errorf("bulk loading dns cache: %w", err)
	}

	return nil
}

// csvEscape replaces tabs and newlines in error messages for TSV format.
func csvEscape(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// DNSProgress is called periodically during DNS resolution with live stats.
type DNSProgress struct {
	Total    int     // total domains to resolve (excluding cached)
	Done     int64   // completed so far
	Live     int64   // newly resolved successfully (excludes cached)
	Dead     int64   // newly failed resolution (NXDOMAIN, excludes cached)
	Timeout  int64   // newly timed out (all resolvers, excludes cached)
	Speed    float64 // current lookups/sec
	AvgSpeed float64 // average lookups/sec over the batch so far
	Elapsed  time.Duration
	Cached   int64 // loaded from cache (already done)
}

// makeFastDNSClients creates fastdns clients for direct UDP DNS resolution.
// Uses Cloudflare + Google DNS with connection pooling for high throughput
// without overwhelming the system DNS resolver (mDNSResponder).
func makeFastDNSClients(timeout time.Duration, connsPerServer int) []*fastdns.Client {
	servers := []string{"1.1.1.1:53", "8.8.8.8:53"}
	clients := make([]*fastdns.Client, len(servers))
	for i, addr := range servers {
		udpAddr, _ := net.ResolveUDPAddr("udp", addr)
		clients[i] = &fastdns.Client{
			Addr:    addr,
			Timeout: timeout,
			Dialer: &fastdns.UDPDialer{
				Addr:     udpAddr,
				MaxConns: uint16(connsPerServer),
			},
		}
	}
	return clients
}

// DNSBatchPoolCapacity returns the total fastdns UDP connection pool size used by ResolveBatch.
func DNSBatchPoolCapacity() int {
	return fastDNSServerCount * fastDNSConnsPerServer
}

// DNSBatchRecommendedWorkerCap returns a stable worker ceiling for batch DNS resolution.
// It is intentionally below raw pool capacity to avoid timeout spikes from oversubscription.
func DNSBatchRecommendedWorkerCap() int {
	return (DNSBatchPoolCapacity() * fastDNSWorkerStableFrac) / 4
}

// ResolveBatch performs fast parallel DNS lookups optimized for maximum throughput.
//
// Uses phuslu/fastdns with direct UDP to Cloudflare (1.1.1.1) and Google (8.8.8.8),
// completely bypassing the system DNS resolver (mDNSResponder) which can't handle
// high concurrency. Connection-pooled UDP sockets achieve 97% accuracy at 1500+ QPS.
//
//   - No IPs returned → mark dead (NXDOMAIN — fastdns returns nil error for NXDOMAIN)
//   - Timeout → mark timeout (saved to cache, treated as dead for this run)
//   - Success → cache IPs for direct dialing (skip DNS during HTTP fetch)
func (d *DNSResolver) ResolveBatch(ctx context.Context, domains []string, workers int, batchTimeout time.Duration, onProgress func(DNSProgress)) (live, dead, timedout int) {
	// Filter out already-cached domains (resolved, dead, or timed-out)
	var toResolve []string
	for _, domain := range domains {
		s := &d.shards[shardFor(domain)]
		s.mu.RLock()
		_, inResolved := s.resolved[domain]
		_, inDead := s.dead[domain]
		_, inTimeout := s.timeout[domain]
		s.mu.RUnlock()
		if !inResolved && !inDead && !inTimeout {
			toResolve = append(toResolve, domain)
		}
	}

	d.total = len(domains)
	start := time.Now()

	if len(toResolve) == 0 {
		d.duration = time.Since(start)
		return int(d.ok.Load()), int(d.failed.Load()), int(d.timedOut.Load())
	}

	if batchTimeout == 0 {
		batchTimeout = 2 * time.Second
	}

	// Create fastdns clients: 2 servers (Cloudflare + Google), pooled UDP connections each
	connsPerServer := fastDNSConnsPerServer
	clients := makeFastDNSClients(batchTimeout, connsPerServer)
	fastDNSPoolCap := len(clients) * connsPerServer

	// Baselines so progress metrics report uncached work only.
	baseOK := d.ok.Load()
	baseFail := d.failed.Load()
	baseTout := d.timedOut.Load()

	// Start progress goroutine
	progressCtx, progressCancel := context.WithCancel(ctx)
	defer progressCancel()
	if onProgress != nil {
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			var lastCount int64
			lastTime := start
			for {
				select {
				case <-ticker.C:
					now := time.Now()
					ok := d.ok.Load()
					fail := d.failed.Load()
					tout := d.timedOut.Load()
					okNew := ok - baseOK
					failNew := fail - baseFail
					toutNew := tout - baseTout
					doneNew := okNew + failNew + toutNew
					dt := now.Sub(lastTime).Seconds()
					speed := float64(0)
					if dt > 0 {
						speed = float64(doneNew-lastCount) / dt
					}
					lastCount = doneNew
					lastTime = now
					avgSpeed := float64(0)
					elapsed := now.Sub(start)
					if elapsed > 0 {
						avgSpeed = float64(doneNew) / elapsed.Seconds()
					}
					onProgress(DNSProgress{
						Total:    len(toResolve),
						Done:     doneNew,
						Live:     okNew,
						Dead:     failNew,
						Timeout:  toutNew,
						Speed:    speed,
						AvgSpeed: avgSpeed,
						Elapsed:  elapsed,
						Cached:   d.cached.Load(),
					})
				case <-progressCtx.Done():
					return
				}
			}
		}()
	}

	// Workers are capped to the fastdns UDP connection pool capacity.
	// More goroutines than pooled UDP connections mostly adds blocking and timeout noise.
	maxWorkers := min(workers, len(toResolve))
	if fastDNSPoolCap > 0 {
		maxWorkers = min(maxWorkers, fastDNSPoolCap)
	}
	ch := make(chan string, maxWorkers*4)
	var wg sync.WaitGroup

	// Standard-library fallback resolver can become a bottleneck under high concurrency.
	// Bound fallback concurrency separately so it doesn't collapse throughput.
	// Cap at workers/4 but raised from 256 to 1024 so retry throughput scales with
	// large worker counts (4K workers → 1K retries @ 1s = 1K/s retry throughput).
	//
	// stdlib is kept in the first pass even for large batches: macOS mDNSResponder
	// serves cached DNS hits in <50ms, recovering most live domains from the previous
	// run's DNS cache. Only the RETRY is skipped for large batches (see resolveOneBatch).
	fallbackTimeout := batchTimeout
	if maxWorkers >= 256 && fallbackTimeout > time.Second {
		fallbackTimeout = time.Second
	}
	stdResolver := makeResolver("", fallbackTimeout)
	// noRetry: for very large batches, skip the 2s retry that serializes workers.
	// The retry recovers ~30% of timed-out domains but adds 2s × workers blocking,
	// tripling per-domain cost (1s fastdns + 2s retry = 3s) and capping throughput
	// at ~1,365/s instead of 4,096/s. Missed slow domains are recovered via the
	// HTTP dialer's inline DNS fallback at fetch time.
	noRetry := len(toResolve) >= 50000
	fallbackLimit := min(max(maxWorkers/4, 32), 1024)
	if len(toResolve) < fallbackLimit {
		fallbackLimit = len(toResolve)
	}
	if fallbackLimit < 1 {
		fallbackLimit = 1
	}
	fallbackSem := make(chan struct{}, fallbackLimit)

	for range maxWorkers {
		wg.Go(func() {
			for domain := range ch {
				d.resolveOneBatch(ctx, domain, clients, stdResolver, fallbackSem, batchTimeout, noRetry)
			}
		})
	}

	for _, domain := range toResolve {
		select {
		case ch <- domain:
		case <-ctx.Done():
			goto drain
		}
	}
drain:
	close(ch)
	wg.Wait()

	// Final progress update
	progressCancel()
	if onProgress != nil {
		ok := d.ok.Load()
		fail := d.failed.Load()
		tout := d.timedOut.Load()
		okNew := ok - baseOK
		failNew := fail - baseFail
		toutNew := tout - baseTout
		doneNew := okNew + failNew + toutNew
		avgSpeed := float64(0)
		if elapsed := time.Since(start); elapsed > 0 {
			avgSpeed = float64(doneNew) / elapsed.Seconds()
		}
		onProgress(DNSProgress{
			Total:    len(toResolve),
			Done:     doneNew,
			Live:     okNew,
			Dead:     failNew,
			Timeout:  toutNew,
			Speed:    0,
			AvgSpeed: avgSpeed,
			Elapsed:  time.Since(start),
			Cached:   d.cached.Load(),
		})
	}

	d.duration = time.Since(start)
	return int(d.ok.Load()), int(d.failed.Load()), int(d.timedOut.Load())
}

// resolveOneBatch resolves a single domain using concurrent multi-server lookups.
// All DNS servers (fastdns + stdlib) are queried simultaneously; first success wins.
// noRetry skips the 2s stdlib retry for large batches to eliminate per-worker serial blocking.
func (d *DNSResolver) resolveOneBatch(ctx context.Context, domain string, clients []*fastdns.Client, stdResolver *net.Resolver, fallbackSem chan struct{}, batchTimeout time.Duration, noRetry bool) {
	totalResolvers := len(clients)
	if stdResolver != nil {
		totalResolvers++
	}
	resolveCtx, resolveCancel := context.WithTimeout(ctx, batchTimeout)
	defer resolveCancel()

	type dnsRes struct {
		addrs          []string
		err            error
		definitelyDead bool
		emptyAnswer    bool
	}
	resCh := make(chan dnsRes, totalResolvers)

	for _, client := range clients {
		go func(c *fastdns.Client) {
			// Query both IPv4 and IPv6. Using only ip4 can misclassify IPv6-only domains as dead.
			ips, lookupErr := c.LookupNetIP(resolveCtx, "ip", domain)
			var addrs []string
			emptyAnswer := false
			if lookupErr == nil {
				addrs = make([]string, len(ips))
				for j, ip := range ips {
					addrs[j] = ip.String()
				}
				emptyAnswer = len(addrs) == 0
			}
			resCh <- dnsRes{addrs: addrs, err: lookupErr, emptyAnswer: emptyAnswer}
		}(client)
	}

	// stdlib fallback (concurrent, bounded by semaphore). Optional in batch mode.
	if stdResolver != nil {
		go func() {
			select {
			case fallbackSem <- struct{}{}:
			case <-resolveCtx.Done():
				resCh <- dnsRes{err: resolveCtx.Err()}
				return
			}
			addrs, lookupErr := stdResolver.LookupHost(resolveCtx, domain)
			<-fallbackSem
			resCh <- dnsRes{addrs: addrs, err: lookupErr, definitelyDead: isDefinitelyDead(lookupErr)}
		}()
	}

	// Collect results — first success wins
	var resolved bool
	var lastErr error
	var sawDefinitelyDead bool
	var sawEmptyAnswer bool
	received := 0
	for received < totalResolvers {
		select {
		case res := <-resCh:
			received++
			if res.err == nil && len(res.addrs) > 0 {
				s := &d.shards[shardFor(domain)]
				s.mu.Lock()
				s.resolved[domain] = res.addrs
				s.mu.Unlock()
				d.ok.Add(1)
				resolved = true
				resolveCancel() // cancel remaining lookups
				break
			}
			if res.definitelyDead {
				sawDefinitelyDead = true
			}
			if res.emptyAnswer {
				sawEmptyAnswer = true
			}
			if res.err != nil {
				lastErr = res.err
			}
		case <-resolveCtx.Done():
			// Some resolver goroutine (typically stdlib fallback) can ignore context and hang.
			// Don't block the worker waiting for all backends; classify based on the timeout.
			if lastErr == nil {
				lastErr = resolveCtx.Err()
			}
			received = totalResolvers
		}
		if resolved {
			break
		}
	}
	if resolved {
		return
	}

	// All servers failed on concurrent attempt.
	// For timeout failures: retry once with doubled timeout via stdlib only.
	// DNS timeouts are often transient (packet loss), not permanent.
	// This recovers ~30-50% of timeout domains with minimal overhead.
	// noRetry skips this for large batches where the 2s wait serializes workers
	// and triples per-domain cost (1s first-pass + 2s retry = 3s per dead domain).
	if !noRetry && stdResolver != nil && lastErr != nil && isTimeoutErr(lastErr) {
		retryTimeout := min(batchTimeout*2, 4*time.Second)
		retryCtx, retryCancel := context.WithTimeout(ctx, retryTimeout)
		select {
		case fallbackSem <- struct{}{}:
		case <-retryCtx.Done():
			retryCancel()
			goto classify
		}
		retryAddrs, retryErr := stdResolver.LookupHost(retryCtx, domain)
		<-fallbackSem
		retryCancel()
		if retryErr == nil && len(retryAddrs) > 0 {
			s := &d.shards[shardFor(domain)]
			s.mu.Lock()
			s.resolved[domain] = retryAddrs
			s.mu.Unlock()
			d.ok.Add(1)
			return
		}
		if retryErr != nil {
			lastErr = retryErr
		}
	}

classify:
	if sawDefinitelyDead {
		errMsg := "NXDOMAIN"
		if lastErr != nil && isDefinitelyDead(lastErr) {
			errMsg = truncateErr(lastErr)
		}
		s := &d.shards[shardFor(domain)]
		s.mu.Lock()
		s.dead[domain] = errMsg
		s.mu.Unlock()
		d.failed.Add(1)
		return
	}
	// Empty answers (especially from fastdns) are ambiguous in batch mode and can include
	// non-NXDOMAIN cases; treat them as timeout/temporary to avoid false "dead" caching.
	if sawEmptyAnswer && lastErr == nil {
		lastErr = fmt.Errorf("no dns answer")
	}
	if lastErr != nil && isTimeoutErr(lastErr) {
		s := &d.shards[shardFor(domain)]
		s.mu.Lock()
		s.timeout[domain] = truncateErr(lastErr)
		s.mu.Unlock()
		d.timedOut.Add(1)
	} else {
		// Non-timeout, non-definitive errors are treated as timeout/temporary to avoid false deads.
		errMsg := "temporary_dns_error"
		if lastErr != nil {
			errMsg = truncateErr(lastErr)
		}
		s := &d.shards[shardFor(domain)]
		s.mu.Lock()
		s.timeout[domain] = errMsg
		s.mu.Unlock()
		d.timedOut.Add(1)
	}
}

// isTimeoutErr returns true if the error indicates a DNS timeout.
func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "timeout") || strings.Contains(s, "deadline")
}

// Resolve performs fast parallel DNS lookups (legacy, use ResolveBatch for new code).
func (d *DNSResolver) Resolve(ctx context.Context, domains []string, workers int, onProgress func(DNSProgress)) (live, dead int) {
	l, de, _ := d.ResolveBatch(ctx, domains, workers, d.lookupTimeout, onProgress)
	return l, de
}

// isDefinitelyDead returns true only for errors that prove the domain doesn't exist.
// Timeouts and temporary errors are NOT definitive — they just mean the resolver was busy.
func isDefinitelyDead(err error) bool {
	if err == nil {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return true
		}
		if !dnsErr.IsTimeout && !dnsErr.IsTemporary &&
			strings.Contains(dnsErr.Error(), "no such host") {
			return true
		}
	}
	return false
}

// truncateErr returns a short error message for storage.
func truncateErr(err error) string {
	if err == nil {
		return "unknown"
	}
	return truncateStr(err.Error(), 100)
}

// ResolveOne resolves a single domain, checking cache first.
// Tries resolvers sequentially: system DNS → Google 8.8.8.8 → Cloudflare 1.1.1.1,
// with per-resolver timeout.
//
// Returns (ips, false, nil) on cache hit or successful resolution.
// Returns (nil, true, nil/err) on NXDOMAIN (definitive dead).
// Returns (nil, false, err) if ALL resolvers timeout (not dead, just unreachable).
func (d *DNSResolver) ResolveOne(ctx context.Context, domain string) (ips []string, dead bool, err error) {
	s := &d.shards[shardFor(domain)]

	// Check cache
	s.mu.RLock()
	if cached, ok := s.resolved[domain]; ok {
		s.mu.RUnlock()
		return cached, false, nil
	}
	if _, isDead := s.dead[domain]; isDead {
		s.mu.RUnlock()
		return nil, true, nil
	}
	if _, isTimeout := s.timeout[domain]; isTimeout {
		s.mu.RUnlock()
		return nil, false, fmt.Errorf("previously timed out")
	}
	s.mu.RUnlock()

	// Try each resolver sequentially with per-resolver timeout
	perTimeout := max(d.lookupTimeout/time.Duration(len(d.resolvers)), 300*time.Millisecond)

	var lastErr error
	for _, resolver := range d.resolvers {
		lookupCtx, cancel := context.WithTimeout(ctx, perTimeout)
		addrs, lookupErr := resolver.LookupHost(lookupCtx, domain)
		cancel()

		if lookupErr == nil && len(addrs) > 0 {
			// Success — cache and return
			s.mu.Lock()
			s.resolved[domain] = addrs
			s.mu.Unlock()
			d.ok.Add(1)
			return addrs, false, nil
		}

		if isDefinitelyDead(lookupErr) {
			// NXDOMAIN — definitive, no need to try other resolvers
			s.mu.Lock()
			s.dead[domain] = truncateErr(lookupErr)
			s.mu.Unlock()
			d.failed.Add(1)
			return nil, true, lookupErr
		}

		// Timeout/temp error — try next resolver
		lastErr = lookupErr
	}

	// All resolvers failed (timeout/temp) — mark as timeout for caching
	s.mu.Lock()
	s.timeout[domain] = truncateErr(lastErr)
	s.mu.Unlock()
	d.timedOut.Add(1)

	return nil, false, lastErr
}

// IsDead returns true if the domain is definitively dead (NXDOMAIN).
func (d *DNSResolver) IsDead(domain string) bool {
	s := &d.shards[shardFor(domain)]
	s.mu.RLock()
	_, dead := s.dead[domain]
	s.mu.RUnlock()
	return dead
}

// IsTimeout returns true if the domain timed out during resolution.
func (d *DNSResolver) IsTimeout(domain string) bool {
	s := &d.shards[shardFor(domain)]
	s.mu.RLock()
	_, tout := s.timeout[domain]
	s.mu.RUnlock()
	return tout
}

// IsDeadOrTimeout returns true if the domain should be skipped (dead or timed out).
func (d *DNSResolver) IsDeadOrTimeout(domain string) bool {
	s := &d.shards[shardFor(domain)]
	s.mu.RLock()
	_, dead := s.dead[domain]
	_, tout := s.timeout[domain]
	s.mu.RUnlock()
	return dead || tout
}

// IsResolved returns true if the domain passed DNS resolution.
func (d *DNSResolver) IsResolved(domain string) bool {
	s := &d.shards[shardFor(domain)]
	s.mu.RLock()
	_, ok := s.resolved[domain]
	s.mu.RUnlock()
	return ok
}

// Stats returns a formatted stats string.
func (d *DNSResolver) Stats() string {
	ok := d.ok.Load()
	fail := d.failed.Load()
	tout := d.timedOut.Load()
	cached := d.cached.Load()
	pct := float64(0)
	if d.total > 0 {
		pct = float64(ok) / float64(d.total) * 100
	}
	if cached > 0 {
		return fmt.Sprintf("%s live (%4.1f%%), %s dead, %s timeout, %s cached, took %s",
			fmtInt(int(ok)), pct, fmtInt(int(fail)), fmtInt(int(tout)), fmtInt(int(cached)), d.duration.Truncate(time.Millisecond))
	}
	return fmt.Sprintf("%s live (%4.1f%%), %s dead, %s timeout, took %s",
		fmtInt(int(ok)), pct, fmtInt(int(fail)), fmtInt(int(tout)), d.duration.Truncate(time.Millisecond))
}

// DeadDomains returns the set of dead domains (NXDOMAIN only).
func (d *DNSResolver) DeadDomains() map[string]bool {
	result := make(map[string]bool)
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		for domain := range s.dead {
			result[domain] = true
		}
		s.mu.RUnlock()
	}
	return result
}

// DeadOrTimeoutDomains returns domains that are dead OR timed out.
func (d *DNSResolver) DeadOrTimeoutDomains() map[string]bool {
	result := make(map[string]bool)
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		for domain := range s.dead {
			result[domain] = true
		}
		for domain := range s.timeout {
			result[domain] = true
		}
		s.mu.RUnlock()
	}
	return result
}

// DeadOrTimeoutDomainsWithReasons returns domains with their failure reason.
// Values: "dns_nxdomain" for dead, "dns_timeout" for timed out.
func (d *DNSResolver) DeadOrTimeoutDomainsWithReasons() map[string]string {
	result := make(map[string]string)
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		for domain := range s.dead {
			result[domain] = "dns_nxdomain"
		}
		for domain := range s.timeout {
			result[domain] = "dns_timeout"
		}
		s.mu.RUnlock()
	}
	return result
}

// DeadDomainsWithErrors returns dead domains with their error messages.
func (d *DNSResolver) DeadDomainsWithErrors() map[string]string {
	result := make(map[string]string)
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		maps.Copy(result, s.dead)
		s.mu.RUnlock()
	}
	return result
}

// TimeoutDomainsWithErrors returns timed-out domains with their error messages.
func (d *DNSResolver) TimeoutDomainsWithErrors() map[string]string {
	result := make(map[string]string)
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		maps.Copy(result, s.timeout)
		s.mu.RUnlock()
	}
	return result
}

// TimeoutDomains returns the set of timed-out domains.
func (d *DNSResolver) TimeoutDomains() map[string]bool {
	result := make(map[string]bool)
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		for domain := range s.timeout {
			result[domain] = true
		}
		s.mu.RUnlock()
	}
	return result
}

// ResolvedIPs returns the map of domain to IPs.
func (d *DNSResolver) ResolvedIPs() map[string][]string {
	result := make(map[string][]string)
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		maps.Copy(result, s.resolved)
		s.mu.RUnlock()
	}
	return result
}

// CachedCount returns how many entries were loaded from cache.
func (d *DNSResolver) CachedCount() int64 {
	return d.cached.Load()
}

// Duration returns how long the DNS resolution took.
func (d *DNSResolver) Duration() time.Duration {
	return d.duration
}

// LiveCount returns the number of successfully resolved domains.
func (d *DNSResolver) LiveCount() int64 {
	return d.ok.Load()
}

// DeadCount returns the number of NXDOMAIN domains.
func (d *DNSResolver) DeadCount() int64 {
	return d.failed.Load()
}

// TimeoutCount returns the number of timed-out domains.
func (d *DNSResolver) TimeoutCount() int64 {
	return d.timedOut.Load()
}

// MergeHTTPDead is a no-op. HTTP failures should NOT contaminate the DNS cache.
// The directFeed probe phase already handles HTTP reachability separately.
// Kept as a method stub so callers don't break.
func (d *DNSResolver) MergeHTTPDead(httpDead map[string]bool) int {
	return 0
}
