package recrawler

import (
	"context"
	"database/sql"
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
// Results can be persisted to a DuckDB cache for instant reuse across runs.
type DNSResolver struct {
	resolver *net.Resolver
	shards   [dnsShardCount]dnsShard

	// Stats
	total    int
	ok       atomic.Int64
	failed   atomic.Int64
	cached   atomic.Int64 // loaded from cache
	duration time.Duration

	// Per-domain lookup timeout
	lookupTimeout time.Duration
}

// NewDNSResolver creates a DNS resolver with a custom timeout.
func NewDNSResolver(timeout time.Duration) *DNSResolver {
	d := &DNSResolver{
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: timeout}
				return d.DialContext(ctx, "udp", address)
			},
		},
		lookupTimeout: 5 * time.Second, // generous per-domain timeout
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

// Resolve performs parallel DNS lookups for all unique domains.
// Domains already loaded from cache are skipped.
// Uses moderate concurrency (caps at 2000 workers) to avoid overwhelming
// the system DNS resolver, which causes false "dead" timeouts.
func (d *DNSResolver) Resolve(ctx context.Context, domains []string, workers int) (live, dead int) {
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

	// Cap workers to avoid overwhelming system DNS.
	// 10K workers on macOS causes massive DNS timeout storms.
	maxWorkers := min(workers, 2000)
	if maxWorkers > len(toResolve) {
		maxWorkers = max(len(toResolve), 1)
	}

	if len(toResolve) > 0 {
		ch := make(chan string, maxWorkers*4)
		var wg sync.WaitGroup

		for range maxWorkers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for domain := range ch {
					d.resolveOne(ctx, domain)
				}
			}()
		}

		for _, domain := range toResolve {
			select {
			case ch <- domain:
			case <-ctx.Done():
				goto done
			}
		}
	done:
		close(ch)
		wg.Wait()
	}

	d.duration = time.Since(start)
	return int(d.ok.Load()), int(d.failed.Load())
}

func (d *DNSResolver) resolveOne(ctx context.Context, domain string) {
	lookupCtx, cancel := context.WithTimeout(ctx, d.lookupTimeout)
	defer cancel()

	addrs, err := d.resolver.LookupHost(lookupCtx, domain)

	s := &d.shards[shardFor(domain)]
	s.mu.Lock()
	if err != nil || len(addrs) == 0 {
		errMsg := "no addresses"
		if err != nil {
			errMsg = truncateStr(err.Error(), 100)
		}
		s.dead[domain] = errMsg
		d.failed.Add(1)
	} else {
		s.resolved[domain] = addrs
		d.ok.Add(1)
	}
	s.mu.Unlock()
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
