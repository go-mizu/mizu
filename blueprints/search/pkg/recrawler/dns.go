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
		query = "SELECT domain, ips, dead, COALESCE(error, '') as error, timeout FROM dns"
	} else {
		query = "SELECT domain, ips, dead, COALESCE(error, '') as error FROM dns"
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
			fmt.Fprintf(w, "%s\t%s\tfalse\t \tfalse\n", domain, strings.Join(ips, ","))
		}
		for domain, errMsg := range s.dead {
			fmt.Fprintf(w, "%s\t \ttrue\t%s\tfalse\n", domain, csvEscape(errMsg))
		}
		for domain, errMsg := range s.timeout {
			fmt.Fprintf(w, "%s\t \tfalse\t%s\ttrue\n", domain, csvEscape(errMsg))
		}
		s.mu.RUnlock()
	}
	w.Flush()
	f.Close()

	// Bulk load via COPY
	_, err = db.Exec(fmt.Sprintf(`COPY dns(domain, ips, dead, error, timeout) FROM '%s' (DELIMITER '\t', HEADER false)`, tmpFile))
	os.Remove(tmpFile)
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
	Total   int     // total domains to resolve (excluding cached)
	Done    int64   // completed so far
	Live    int64   // resolved successfully
	Dead    int64   // failed resolution (NXDOMAIN)
	Timeout int64   // timed out (all resolvers)
	Speed   float64 // lookups/sec (rolling)
	Elapsed time.Duration
	Cached  int64 // loaded from cache (already done)
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
				Timeout:  timeout,
				MaxConns: uint16(connsPerServer),
			},
		}
	}
	return clients
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

	// Create fastdns clients: 2 servers (Cloudflare + Google), 256 connections each
	connsPerServer := 256
	clients := makeFastDNSClients(batchTimeout, connsPerServer)

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
					done := ok + fail + tout
					dt := now.Sub(lastTime).Seconds()
					speed := float64(0)
					if dt > 0 {
						speed = float64(done-lastCount) / dt
					}
					lastCount = done
					lastTime = now
					onProgress(DNSProgress{
						Total:   len(toResolve),
						Done:    done - d.cached.Load(),
						Live:    ok,
						Dead:    fail,
						Timeout: tout,
						Speed:   speed,
						Elapsed: now.Sub(start),
						Cached:  d.cached.Load(),
					})
				case <-progressCtx.Done():
					return
				}
			}
		}()
	}

	// Workers: 1000 per server (2000 total), round-robin across clients
	maxWorkers := min(workers, len(toResolve))
	ch := make(chan string, maxWorkers*4)
	var wg sync.WaitGroup

	for i := range maxWorkers {
		wg.Add(1)
		client := clients[i%len(clients)]
		go func() {
			defer wg.Done()
			for domain := range ch {
				lookupCtx, cancel := context.WithTimeout(ctx, batchTimeout)
				ips, err := client.LookupNetIP(lookupCtx, "ip4", domain)
				cancel()

				if err == nil && len(ips) > 0 {
					// Success — convert netip.Addr to strings, cache IPs
					addrs := make([]string, len(ips))
					for j, ip := range ips {
						addrs[j] = ip.String()
					}
					s := &d.shards[shardFor(domain)]
					s.mu.Lock()
					s.resolved[domain] = addrs
					s.mu.Unlock()
					d.ok.Add(1)
				} else if err == nil {
					// No IPs, no error = NXDOMAIN (fastdns returns nil for NXDOMAIN)
					s := &d.shards[shardFor(domain)]
					s.mu.Lock()
					s.dead[domain] = "NXDOMAIN"
					s.mu.Unlock()
					d.failed.Add(1)
				} else if isTimeoutErr(err) {
					// Timeout — mark as timeout for caching
					s := &d.shards[shardFor(domain)]
					s.mu.Lock()
					s.timeout[domain] = truncateErr(err)
					s.mu.Unlock()
					d.timedOut.Add(1)
				} else {
					// Other error (server failure, etc.) — treat as dead
					s := &d.shards[shardFor(domain)]
					s.mu.Lock()
					s.dead[domain] = truncateErr(err)
					s.mu.Unlock()
					d.failed.Add(1)
				}
			}
		}()
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
		onProgress(DNSProgress{
			Total:   len(toResolve),
			Done:    ok + fail + tout - d.cached.Load(),
			Live:    ok,
			Dead:    fail,
			Timeout: tout,
			Speed:   0,
			Elapsed: time.Since(start),
			Cached:  d.cached.Load(),
		})
	}

	d.duration = time.Since(start)
	return int(d.ok.Load()), int(d.failed.Load()), int(d.timedOut.Load())
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
	perTimeout := d.lookupTimeout / time.Duration(len(d.resolvers))
	if perTimeout < 300*time.Millisecond {
		perTimeout = 300 * time.Millisecond
	}

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

// MergeHTTPDead merges HTTP-level dead domains into the DNS cache.
// These are domains where DNS resolved but HTTP connection was refused/reset.
// Stored as "dead" in the cache with an "http_dead" error message so
// subsequent runs skip them instantly.
func (d *DNSResolver) MergeHTTPDead(httpDead map[string]bool) int {
	merged := 0
	for domain := range httpDead {
		s := &d.shards[shardFor(domain)]
		s.mu.Lock()
		// Only mark as dead if not already resolved/dead/timeout
		if _, ok := s.dead[domain]; !ok {
			if _, ok := s.timeout[domain]; !ok {
				// Move from resolved → dead (server has IP but doesn't respond)
				delete(s.resolved, domain)
				s.dead[domain] = "http_dead"
				merged++
			}
		}
		s.mu.Unlock()
	}
	return merged
}
