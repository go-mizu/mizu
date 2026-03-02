package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	_ "github.com/duckdb/duckdb-go/v2"
)

// FailedDB stores failed domains and URLs in DuckDB for debugging and verification.
// Uses async batch flushers for both tables to avoid blocking the crawl pipeline.
// All public methods are nil-safe: calling on nil *FailedDB is a no-op.
// Implements crawl.FailureWriter directly.
type FailedDB struct {
	db       *sql.DB
	domainCh chan crawl.FailedDomain
	urlCh    chan crawl.FailedURL
	wg       sync.WaitGroup

	domainCount atomic.Int64
	urlCount    atomic.Int64
}

// NewFailedDB creates a new FailedDB at the given path.
// Two tables are created: failed_domains and failed_urls.
func NewFailedDB(path string) (*FailedDB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("opening failed db: %w", err)
	}
	if err := initFailedSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	fdb := &FailedDB{
		db:       db,
		domainCh: make(chan crawl.FailedDomain, 10000),
		urlCh:    make(chan crawl.FailedURL, 100000),
	}

	fdb.wg.Add(2)
	go fdb.domainFlusher()
	go fdb.urlFlusher()

	return fdb, nil
}

// OpenFailedDB is like NewFailedDB but first removes any stale DuckDB lock
// left by a dead process, preventing "conflicting lock" errors on retry.
func OpenFailedDB(path string) (*FailedDB, error) {
	removeIfStaleLocked(path)
	return NewFailedDB(path)
}

// removeIfStaleLocked checks the DuckDB .lock file alongside dbPath.
// If it exists and the recorded PID belongs to a dead process, the lock
// file is removed so the next open succeeds. The DB file is left intact.
func removeIfStaleLocked(dbPath string) {
	lockPath := dbPath + ".lock"
	data, err := os.ReadFile(lockPath)
	if errors.Is(err, os.ErrNotExist) {
		return // no lock file — nothing to do
	}
	if err != nil {
		return // unreadable — let DuckDB handle it
	}
	pid := parseLockFilePID(data)
	if pid <= 0 {
		return // can't parse PID — let DuckDB handle it
	}
	if processIsAlive(pid) {
		return // genuine live lock — don't touch it
	}
	// Dead process: remove stale lock so next open succeeds.
	// Do NOT delete the DB file — DuckDB WAL recovery handles incomplete transactions.
	os.Remove(lockPath)
}

// parseLockFilePID extracts the PID from DuckDB lock file content.
// DuckDB writes "PID=<n>\n" on Linux/macOS.
func parseLockFilePID(data []byte) int {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "PID="); ok {
			var pid int
			fmt.Sscanf(after, "%d", &pid)
			return pid
		}
	}
	// Fallback: try parsing first integer in file
	var pid int
	fmt.Sscanf(string(data), "%d", &pid)
	return pid
}

// processIsAlive returns true if the given PID is a running process.
// Sends signal 0 (no-op) to check existence without disturbing the process.
func processIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func initFailedSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS failed_domains (
			domain VARCHAR PRIMARY KEY,
			reason VARCHAR NOT NULL,
			error_msg VARCHAR DEFAULT '',
			ips VARCHAR DEFAULT '',
			url_count INTEGER DEFAULT 0,
			stage VARCHAR DEFAULT '',
			detected_at TIMESTAMP DEFAULT current_timestamp
		)
	`)
	if err != nil {
		return fmt.Errorf("creating failed_domains table: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS failed_urls (
			url VARCHAR PRIMARY KEY,
			domain VARCHAR NOT NULL,
			reason VARCHAR NOT NULL,
			error_msg VARCHAR DEFAULT '',
			status_code INTEGER DEFAULT 0,
			fetch_time_ms BIGINT DEFAULT 0,
			content_type VARCHAR DEFAULT '',
			redirect_url VARCHAR DEFAULT '',
			detected_at TIMESTAMP DEFAULT current_timestamp
		)
	`)
	if err != nil {
		return fmt.Errorf("creating failed_urls table: %w", err)
	}

	return nil
}

// AddDomain queues a failed domain for batch writing. Nil-safe.
func (f *FailedDB) AddDomain(d crawl.FailedDomain) {
	if f == nil {
		return
	}
	if d.DetectedAt.IsZero() {
		d.DetectedAt = time.Now()
	}
	select {
	case f.domainCh <- d:
	default:
	}
}

// AddURL queues a failed URL for batch writing. Nil-safe.
// Implements crawl.FailureWriter.
func (f *FailedDB) AddURL(u crawl.FailedURL) {
	if f == nil {
		return
	}
	if u.DetectedAt.IsZero() {
		u.DetectedAt = time.Now()
	}
	select {
	case f.urlCh <- u:
	default:
	}
}

// AddURLBatch queues failed URLs for all URLs of a dead domain. Nil-safe.
func (f *FailedDB) AddURLBatch(urls []crawl.SeedURL, reason string) {
	if f == nil {
		return
	}
	now := time.Now()
	for _, u := range urls {
		select {
		case f.urlCh <- crawl.FailedURL{
			URL:        u.URL,
			Domain:     u.Domain,
			Reason:     reason,
			DetectedAt: now,
		}:
		default:
		}
	}
}

// DomainCount returns the number of failed domains written. Nil-safe.
func (f *FailedDB) DomainCount() int64 {
	if f == nil {
		return 0
	}
	return f.domainCount.Load()
}

// URLCount returns the number of failed URLs written. Nil-safe.
func (f *FailedDB) URLCount() int64 {
	if f == nil {
		return 0
	}
	return f.urlCount.Load()
}

// SetMeta stores a key-value pair in a meta table. Nil-safe.
func (f *FailedDB) SetMeta(key, value string) {
	if f == nil {
		return
	}
	f.db.Exec(`CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)`)
	f.db.Exec("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)", key, value)
}

// Close flushes remaining data and closes the database.
// Implements crawl.FailureWriter.
func (f *FailedDB) Close() error {
	if f == nil {
		return nil
	}
	close(f.domainCh)
	close(f.urlCh)
	f.wg.Wait()
	return f.db.Close()
}

// ── flushers ────────────────────────────────────────────────

func (f *FailedDB) domainFlusher() {
	defer f.wg.Done()
	batch := make([]crawl.FailedDomain, 0, 1000)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := f.writeDomainBatch(batch); err != nil {
			slog.Default().Warn("failed to write domain batch", "err", err)
		}
		f.domainCount.Add(int64(len(batch)))
		batch = batch[:0]
	}

	for d := range f.domainCh {
		batch = append(batch, d)
		if len(batch) >= 1000 {
			flush()
		}
	}
	flush()
}

func (f *FailedDB) urlFlusher() {
	defer f.wg.Done()
	batch := make([]crawl.FailedURL, 0, 5000)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := f.writeURLBatch(batch); err != nil {
			slog.Default().Warn("failed to write URL batch", "err", err)
		}
		f.urlCount.Add(int64(len(batch)))
		batch = batch[:0]
	}

	for u := range f.urlCh {
		batch = append(batch, u)
		if len(batch) >= 5000 {
			flush()
		}
	}
	flush()
}

// ── batch writers ────────────────────────────────────────────

func (f *FailedDB) writeDomainBatch(batch []crawl.FailedDomain) error {
	const maxPerStmt = 500
	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT OR REPLACE INTO failed_domains (domain, reason, error_msg, ips, url_count, stage, detected_at) VALUES ")
		args := make([]any, 0, len(chunk)*7)

		for j, d := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?,?,?)")
			args = append(args, d.Domain, d.Reason, d.Error, d.IPs, d.URLCount, d.Stage, d.DetectedAt)
		}

		if _, err := f.db.Exec(b.String(), args...); err != nil {
			return err
		}
	}
	return nil
}

func (f *FailedDB) writeURLBatch(batch []crawl.FailedURL) error {
	const maxPerStmt = 500
	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT OR REPLACE INTO failed_urls (url, domain, reason, error_msg, status_code, fetch_time_ms, content_type, redirect_url, detected_at) VALUES ")
		args := make([]any, 0, len(chunk)*9)

		for j, u := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?,?,?,?,?)")
			args = append(args, u.URL, u.Domain, u.Reason, u.Error, u.StatusCode, u.FetchTimeMs, u.ContentType, u.RedirectURL, u.DetectedAt)
		}

		if _, err := f.db.Exec(b.String(), args...); err != nil {
			return err
		}
	}
	return nil
}

// ── reading / analysis ────────────────────────────────────────

// LoadFailedDomains reads all failed domains from a FailedDB file.
func LoadFailedDomains(dbPath string) ([]crawl.FailedDomain, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("opening failed db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT domain, reason, COALESCE(error_msg, ''), COALESCE(ips, ''),
		       COALESCE(url_count, 0), COALESCE(stage, ''), detected_at
		FROM failed_domains
		ORDER BY url_count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying failed domains: %w", err)
	}
	defer rows.Close()

	var results []crawl.FailedDomain
	for rows.Next() {
		var d crawl.FailedDomain
		if err := rows.Scan(&d.Domain, &d.Reason, &d.Error, &d.IPs, &d.URLCount, &d.Stage, &d.DetectedAt); err != nil {
			continue
		}
		results = append(results, d)
	}
	return results, nil
}

// FailedDomainSummary returns a breakdown of failure reasons and total count.
func FailedDomainSummary(dbPath string) (map[string]int, int, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, 0, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT reason, COUNT(*) FROM failed_domains GROUP BY reason ORDER BY COUNT(*) DESC")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	summary := make(map[string]int)
	total := 0
	for rows.Next() {
		var reason string
		var count int
		rows.Scan(&reason, &count)
		summary[reason] = count
		total += count
	}
	return summary, total, nil
}

// LoadRetryURLs reads all URLs worth retrying in pass 2:
//   - http_timeout:                server connected but timed out
//   - dns_timeout:                 DNS lookup timed out
//   - domain_http_timeout_killed:  domain hit DomainFailThreshold; URL never ran
//   - domain_deadline_exceeded:    domain 30s deadline fired; URL never ran
//
// Returns seeds ordered by domain for connection-pool efficiency.
func LoadRetryURLs(dbPath string) ([]crawl.SeedURL, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("opening failed db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT url, domain
		FROM failed_urls
		WHERE reason IN (
			'http_timeout',
			'dns_timeout',
			'domain_http_timeout_killed',
			'domain_deadline_exceeded'
		)
		ORDER BY domain, url
	`)
	if err != nil {
		return nil, fmt.Errorf("querying retry URLs: %w", err)
	}
	defer rows.Close()

	var seeds []crawl.SeedURL
	for rows.Next() {
		var s crawl.SeedURL
		if err := rows.Scan(&s.URL, &s.Domain); err != nil {
			continue
		}
		seeds = append(seeds, s)
	}
	return seeds, nil
}

// LoadRetryURLsSince is like LoadRetryURLs but only returns URLs detected on or after since.
// Use to restrict pass 2 to the current run when failedDB accumulates across multiple runs.
func LoadRetryURLsSince(dbPath string, since time.Time) ([]crawl.SeedURL, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("opening failed db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT url, domain
		FROM failed_urls
		WHERE reason IN (
			'http_timeout',
			'dns_timeout',
			'domain_http_timeout_killed',
			'domain_deadline_exceeded'
		)
		AND detected_at >= ?
		ORDER BY domain, url
	`, since)
	if err != nil {
		return nil, fmt.Errorf("querying retry URLs: %w", err)
	}
	defer rows.Close()

	var seeds []crawl.SeedURL
	for rows.Next() {
		var s crawl.SeedURL
		if err := rows.Scan(&s.URL, &s.Domain); err != nil {
			continue
		}
		seeds = append(seeds, s)
	}
	return seeds, nil
}

// FailedURLSummary returns a breakdown of failed URLs by reason and total count.
func FailedURLSummary(dbPath string) (map[string]int, int, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, 0, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT reason, COUNT(*) FROM failed_urls GROUP BY reason ORDER BY COUNT(*) DESC")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	summary := make(map[string]int)
	total := 0
	for rows.Next() {
		var reason string
		var count int
		rows.Scan(&reason, &count)
		summary[reason] = count
		total += count
	}
	return summary, total, nil
}

// FailedURLTopDomains returns the top N domains by total failure count.
// Each entry is [domain, count_string] sorted by count descending.
// Returns nil, nil when dbPath is empty or the DB has no failed URLs.
func FailedURLTopDomains(dbPath string, n int) ([][2]string, error) {
	if dbPath == "" {
		return nil, nil
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT domain, COUNT(*) AS c FROM failed_urls
		 GROUP BY domain ORDER BY c DESC LIMIT ?`, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result [][2]string
	for rows.Next() {
		var domain string
		var count int
		rows.Scan(&domain, &count)
		result = append(result, [2]string{domain, strconv.Itoa(count)})
	}
	return result, nil
}
