package recrawler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const dnsShardCount = 64 // must be power of 2

// dnsShard is a single shard of the DNS cache, each with its own lock.
type dnsShard struct {
	mu       sync.RWMutex
	resolved map[string][]string // domain → IPs
	dead     map[string]string   // domain → error message
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
	cached   atomic.Int64 // loaded from cache
	retrying atomic.Int64 // domains queued for retry
	duration time.Duration

	// Per-domain lookup timeout
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
func NewDNSResolver(timeout time.Duration) *DNSResolver {
	d := &DNSResolver{
		resolvers: []*net.Resolver{
			makeResolver("", timeout),           // system DNS (fast for cached, leverages OS cache)
			makeResolver("8.8.8.8:53", timeout), // Google (fallback, high-concurrency)
			makeResolver("1.1.1.1:53", timeout), // Cloudflare (tertiary)
		},
		lookupTimeout: 3 * time.Second,
	}
	for i := range d.shards {
		d.shards[i].resolved = make(map[string][]string)
		d.shards[i].dead = make(map[string]string)
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

	rows, err := db.Query("SELECT domain, ips, dead, error FROM dns")
	if err != nil {
		return 0, nil
	}
	defer rows.Close()

	loaded := 0
	for rows.Next() {
		var domain, ips, errMsg string
		var dead bool
		if err := rows.Scan(&domain, &ips, &dead, &errMsg); err != nil {
			continue
		}
		s := &d.shards[shardFor(domain)]
		s.mu.Lock()
		if dead {
			s.dead[domain] = errMsg
			d.failed.Add(1)
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
// Uses batch VALUES (1000 rows/statement) for fast bulk insert.
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
			resolved_at TIMESTAMP DEFAULT current_timestamp
		)
	`)
	if err != nil {
		return fmt.Errorf("creating dns table: %w", err)
	}

	// Collect all entries from all shards
	type entry struct {
		domain string
		ips    string
		dead   bool
		errMsg string
	}

	totalSize := 0
	for i := range d.shards {
		d.shards[i].mu.RLock()
		totalSize += len(d.shards[i].resolved) + len(d.shards[i].dead)
		d.shards[i].mu.RUnlock()
	}

	entries := make([]entry, 0, totalSize)
	for i := range d.shards {
		s := &d.shards[i]
		s.mu.RLock()
		for domain, ips := range s.resolved {
			entries = append(entries, entry{domain, strings.Join(ips, ","), false, ""})
		}
		for domain, errMsg := range s.dead {
			entries = append(entries, entry{domain, "", true, errMsg})
		}
		s.mu.RUnlock()
	}

	// Batch insert using multi-row VALUES (1000 rows per statement)
	const batchSize = 1000
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for i := 0; i < len(entries); i += batchSize {
		end := min(i+batchSize, len(entries))
		batch := entries[i:end]

		var b strings.Builder
		b.WriteString("INSERT INTO dns (domain, ips, dead, error) VALUES ")
		args := make([]any, 0, len(batch)*4)
		for j, e := range batch {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?)")
			args = append(args, e.domain, e.ips, e.dead, e.errMsg)
		}

		if _, err := tx.Exec(b.String(), args...); err != nil {
			tx.Rollback()
			return fmt.Errorf("batch insert at offset %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// DNSProgress is called periodically during DNS resolution with live stats.
type DNSProgress struct {
	Total    int     // total domains to resolve (excluding cached)
	Done     int64   // completed so far (live + dead - cached)
	Live     int64   // resolved successfully
	Dead     int64   // failed resolution
	Speed    float64 // lookups/sec (rolling)
	Elapsed  time.Duration
	Cached   int64   // loaded from cache (already done)
	Retrying int     // domains queued for retry (0 during pass 1)
}

// Resolve performs fast parallel DNS lookups optimized for throughput.
//
// Strategy: Single pass with system DNS, 1.5s timeout, 5000 workers.
//   - NXDOMAIN → mark dead (definitive, domain doesn't exist)
//   - Timeout/temp error → NOT dead (leave for HTTP pipeline to handle)
//   - Success → cache IPs for direct dialing (skip DNS during HTTP fetch)
//
// This is a speed optimization, not a filter. Only definitively dead domains
// are marked; timeouts are left for the HTTP transport's own DNS resolution.
//
// If onProgress is non-nil, it's called every 500ms with live stats.
func (d *DNSResolver) Resolve(ctx context.Context, domains []string, workers int, onProgress func(DNSProgress)) (live, dead int) {
	// Filter out already-cached domains
	var toResolve []string
	for _, domain := range domains {
		s := &d.shards[shardFor(domain)]
		s.mu.RLock()
		_, inResolved := s.resolved[domain]
		_, inDead := s.dead[domain]
		s.mu.RUnlock()
		if !inResolved && !inDead {
			toResolve = append(toResolve, domain)
		}
	}

	d.total = len(domains)
	start := time.Now()

	if len(toResolve) == 0 {
		d.duration = time.Since(start)
		return int(d.ok.Load()), int(d.failed.Load())
	}

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
					skipped := d.retrying.Load() // reuse as "skipped" count
					done := ok + fail + skipped
					dt := now.Sub(lastTime).Seconds()
					speed := float64(0)
					if dt > 0 {
						speed = float64(done-lastCount) / dt
					}
					lastCount = done
					lastTime = now
					onProgress(DNSProgress{
						Total:    len(toResolve),
						Done:     done - d.cached.Load(),
						Live:     ok,
						Dead:     fail,
						Speed:    speed,
						Elapsed:  now.Sub(start),
						Cached:   d.cached.Load(),
						Retrying: int(skipped),
					})
				case <-progressCtx.Done():
					return
				}
			}
		}()
	}

	// Single fast pass: system DNS first, then Google, then Cloudflare for NXDOMAIN verification
	maxWorkers := min(workers, 5000)
	if maxWorkers > len(toResolve) {
		maxWorkers = len(toResolve)
	}

	d.resolvePass(ctx, toResolve, maxWorkers, d.resolvers[0], d.lookupTimeout, func(domain string, err error) {
		if isDefinitelyDead(err) {
			// NXDOMAIN is definitive — mark dead
			s := &d.shards[shardFor(domain)]
			s.mu.Lock()
			s.dead[domain] = truncateErr(err)
			s.mu.Unlock()
			d.failed.Add(1)
		} else {
			// Timeout/temp error — NOT dead, just unresolved.
			// HTTP transport will do its own DNS lookup for these.
			d.retrying.Add(1) // count as "skipped" for progress
		}
	})

	// Final progress update
	progressCancel()
	if onProgress != nil {
		ok := d.ok.Load()
		fail := d.failed.Load()
		skipped := d.retrying.Load()
		onProgress(DNSProgress{
			Total:    len(toResolve),
			Done:     ok + fail + skipped - d.cached.Load(),
			Live:     ok,
			Dead:     fail,
			Speed:    0,
			Elapsed:  time.Since(start),
			Cached:   d.cached.Load(),
			Retrying: int(skipped),
		})
	}

	d.duration = time.Since(start)
	return int(d.ok.Load()), int(d.failed.Load())
}

// resolvePass runs parallel DNS lookups using a single resolver.
// onFail is called for each domain that fails (NOT recorded in shards — caller decides).
// Successful lookups are recorded directly in shards.
func (d *DNSResolver) resolvePass(ctx context.Context, domains []string, workers int, resolver *net.Resolver, timeout time.Duration, onFail func(string, error)) {
	d.runWorkers(ctx, domains, workers, resolver, timeout, onFail)
}

func (d *DNSResolver) runWorkers(ctx context.Context, domains []string, workers int, resolver *net.Resolver, timeout time.Duration, onFail func(string, error)) {
	ch := make(chan string, workers*4)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range ch {
				lookupCtx, cancel := context.WithTimeout(ctx, timeout)
				addrs, err := resolver.LookupHost(lookupCtx, domain)
				cancel()

				if err == nil && len(addrs) > 0 {
					s := &d.shards[shardFor(domain)]
					s.mu.Lock()
					s.resolved[domain] = addrs
					s.mu.Unlock()
					d.ok.Add(1)
				} else {
					onFail(domain, err)
				}
			}
		}()
	}

	for _, domain := range domains {
		select {
		case ch <- domain:
		case <-ctx.Done():
			goto drain
		}
	}
drain:
	close(ch)
	wg.Wait()
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
// Tries resolvers sequentially: Google 8.8.8.8 (primary, high-concurrency)
// → Cloudflare 1.1.1.1 → system DNS, with per-resolver timeout.
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
	s.mu.RUnlock()

	// Try each resolver sequentially with per-resolver timeout
	perTimeout := d.lookupTimeout / time.Duration(len(d.resolvers))
	if perTimeout < 500*time.Millisecond {
		perTimeout = 500 * time.Millisecond
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

	// All resolvers failed (timeout/temp) — not dead, just unreachable
	return nil, false, lastErr
}

// IsDead returns true if the domain failed DNS resolution.
func (d *DNSResolver) IsDead(domain string) bool {
	s := &d.shards[shardFor(domain)]
	s.mu.RLock()
	_, dead := s.dead[domain]
	s.mu.RUnlock()
	return dead
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
	cached := d.cached.Load()
	pct := float64(0)
	if d.total > 0 {
		pct = float64(ok) / float64(d.total) * 100
	}
	if cached > 0 {
		return fmt.Sprintf("%s domains resolved (%4.1f%%), %s dead, %s cached, took %s",
			fmtInt(int(ok)), pct, fmtInt(int(fail)), fmtInt(int(cached)), d.duration.Truncate(time.Millisecond))
	}
	return fmt.Sprintf("%s domains resolved (%4.1f%%), %s dead, took %s",
		fmtInt(int(ok)), pct, fmtInt(int(fail)), d.duration.Truncate(time.Millisecond))
}

// DeadDomains returns the set of dead domains.
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
