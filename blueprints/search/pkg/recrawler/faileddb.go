package recrawler

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// FailedDomain records a domain classified as unreachable during recrawling.
type FailedDomain struct {
	Domain     string
	Reason     string // dns_nxdomain, dns_timeout, probe_unreachable, http_timeout_killed, http_refused, http_dns_error
	Error      string // original error message
	IPs        string // comma-separated resolved IPs (empty if DNS failed)
	URLCount   int    // number of URLs affected by this domain failure
	Stage      string // dns_batch, probe, http_worker
	DetectedAt time.Time
}

// FailedURL records a URL that was skipped or failed during recrawling.
type FailedURL struct {
	URL         string
	Domain      string
	Reason      string // domain_dns_nxdomain, domain_dns_timeout, domain_probe_dead, domain_http_timeout_killed, domain_http_refused, http_timeout, http_refused, http_reset, http_dns_error, http_error
	Error       string // error message (empty for domain-dead skips)
	StatusCode  int    // HTTP status code (0 if never fetched)
	FetchTimeMs int64  // fetch latency in ms (0 if skipped)
	ContentType string // response Content-Type header
	RedirectURL string // final URL after redirects
	DetectedAt  time.Time
}

// FailedDB stores failed domains and URLs in DuckDB for debugging and verification.
// Uses async batch flushers for both tables to avoid blocking the recrawl pipeline.
// All public methods are nil-safe: calling on nil *FailedDB is a no-op.
type FailedDB struct {
	db       *sql.DB
	domainCh chan FailedDomain
	urlCh    chan FailedURL
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
		domainCh: make(chan FailedDomain, 10000),
		urlCh:    make(chan FailedURL, 100000),
	}

	fdb.wg.Add(2)
	go fdb.domainFlusher()
	go fdb.urlFlusher()

	return fdb, nil
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
func (f *FailedDB) AddDomain(d FailedDomain) {
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
func (f *FailedDB) AddURL(u FailedURL) {
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
func (f *FailedDB) AddURLBatch(urls []SeedURL, reason string) {
	if f == nil {
		return
	}
	now := time.Now()
	for _, u := range urls {
		select {
		case f.urlCh <- FailedURL{
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
	batch := make([]FailedDomain, 0, 1000)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		f.writeDomainBatch(batch)
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
	batch := make([]FailedURL, 0, 5000)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		f.writeURLBatch(batch)
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

func (f *FailedDB) writeDomainBatch(batch []FailedDomain) {
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

		f.db.Exec(b.String(), args...)
	}
}

func (f *FailedDB) writeURLBatch(batch []FailedURL) {
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

		f.db.Exec(b.String(), args...)
	}
}

// ── reading / analysis ────────────────────────────────────────

// LoadFailedDomains reads all failed domains from a FailedDB file.
func LoadFailedDomains(dbPath string) ([]FailedDomain, error) {
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

	var results []FailedDomain
	for rows.Next() {
		var d FailedDomain
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
