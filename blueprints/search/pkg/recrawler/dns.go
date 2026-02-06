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

// DNSResolver performs parallel DNS pre-resolution for a set of domains.
// Domains that fail DNS lookup are marked dead, allowing their URLs to be skipped.
// Results can be persisted to a DuckDB cache for instant reuse across runs.
type DNSResolver struct {
	resolver *net.Resolver

	mu       sync.RWMutex
	resolved map[string][]string // domain → IPs
	dead     map[string]string   // domain → error message

	// Stats
	total    int
	ok       atomic.Int64
	failed   atomic.Int64
	cached   atomic.Int64 // loaded from cache
	duration time.Duration
}

// NewDNSResolver creates a DNS resolver with a custom timeout.
func NewDNSResolver(timeout time.Duration) *DNSResolver {
	return &DNSResolver{
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: timeout}
				return d.DialContext(ctx, "udp", address)
			},
		},
		resolved: make(map[string][]string),
		dead:     make(map[string]string),
	}
}

// LoadCache loads previously resolved DNS data from a DuckDB file.
// Returns (loaded, error). Domains already in cache are skipped during Resolve().
func (d *DNSResolver) LoadCache(dbPath string) (int, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, nil // cache doesn't exist yet
	}
	defer db.Close()

	// Check if table exists
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
	d.mu.Lock()
	for rows.Next() {
		var domain, ips, errMsg string
		var dead bool
		if err := rows.Scan(&domain, &ips, &dead, &errMsg); err != nil {
			continue
		}
		if dead {
			d.dead[domain] = errMsg
			d.failed.Add(1)
		} else {
			ipList := strings.Split(ips, ",")
			d.resolved[domain] = ipList
			d.ok.Add(1)
		}
		loaded++
	}
	d.mu.Unlock()
	d.cached.Store(int64(loaded))
	return loaded, nil
}

// SaveCache persists DNS resolution results to a DuckDB file.
// Uses a single transaction for maximum speed.
func (d *DNSResolver) SaveCache(dbPath string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("opening dns cache db: %w", err)
	}
	defer db.Close()

	// Drop and recreate without PRIMARY KEY for faster bulk insert
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

	d.mu.RLock()
	// Collect all entries
	type entry struct {
		domain string
		ips    string
		dead   bool
		errMsg string
	}
	entries := make([]entry, 0, len(d.resolved)+len(d.dead))
	for domain, ips := range d.resolved {
		entries = append(entries, entry{domain, strings.Join(ips, ","), false, ""})
	}
	for domain, errMsg := range d.dead {
		entries = append(entries, entry{domain, "", true, errMsg})
	}
	d.mu.RUnlock()

	// Single transaction, prepared statement, all rows
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO dns (domain, ips, dead, error) VALUES (?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, e := range entries {
		if _, err := stmt.Exec(e.domain, e.ips, e.dead, e.errMsg); err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
	}
	stmt.Close()

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Resolve performs parallel DNS lookups for all unique domains.
// Domains already loaded from cache are skipped.
// Returns the number of live and dead domains.
func (d *DNSResolver) Resolve(ctx context.Context, domains []string, workers int) (live, dead int) {
	// Filter out already-cached domains
	var toResolve []string
	d.mu.RLock()
	for _, domain := range domains {
		if _, ok := d.resolved[domain]; ok {
			continue
		}
		if _, ok := d.dead[domain]; ok {
			continue
		}
		toResolve = append(toResolve, domain)
	}
	d.mu.RUnlock()

	d.total = len(domains)
	start := time.Now()

	if len(toResolve) > 0 {
		ch := make(chan string, workers*4)
		var wg sync.WaitGroup

		// Launch workers
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for domain := range ch {
					d.resolveOne(ctx, domain)
				}
			}()
		}

		// Feed domains
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
	lookupCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	addrs, err := d.resolver.LookupHost(lookupCtx, domain)

	d.mu.Lock()
	if err != nil || len(addrs) == 0 {
		errMsg := "no addresses"
		if err != nil {
			errMsg = truncateStr(err.Error(), 100)
		}
		d.dead[domain] = errMsg
		d.failed.Add(1)
	} else {
		d.resolved[domain] = addrs
		d.ok.Add(1)
	}
	d.mu.Unlock()
}

// IsDead returns true if the domain failed DNS resolution.
func (d *DNSResolver) IsDead(domain string) bool {
	d.mu.RLock()
	_, dead := d.dead[domain]
	d.mu.RUnlock()
	return dead
}

// IsResolved returns true if the domain passed DNS resolution.
func (d *DNSResolver) IsResolved(domain string) bool {
	d.mu.RLock()
	_, ok := d.resolved[domain]
	d.mu.RUnlock()
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
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string]bool, len(d.dead))
	for domain := range d.dead {
		result[domain] = true
	}
	return result
}

// ResolvedIPs returns the map of domain to IPs.
func (d *DNSResolver) ResolvedIPs() map[string][]string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string][]string, len(d.resolved))
	maps.Copy(result, d.resolved)
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
