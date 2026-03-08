package web

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DomainStore maintains a lightweight domains.duckdb that caches
// (domain, parquet_path, count, shard) aggregate rows.
//
// Domain detail queries are served by querying the parquet files directly
// via DuckDB read_parquet() — no per-parquet .meta.duckdb files needed.
//
// EnsureFresh is non-blocking: the first call starts a background goroutine
// and returns immediately. Callers get stale-but-fast results while sync runs.
type DomainStore struct {
	crawlDir string // ~/data/common-crawl/{crawlID}
	mu       sync.Mutex
	db       *sql.DB // lazily opened domains.duckdb
	syncing  bool    // background sync in progress
	synced   bool    // at least one sync completed
}

// NewDomainStore creates a DomainStore rooted at crawlDir.
func NewDomainStore(crawlDir string) *DomainStore {
	return &DomainStore{crawlDir: crawlDir}
}

func (ds *DomainStore) dbPath() string {
	return filepath.Join(ds.crawlDir, "domains.duckdb")
}

func (ds *DomainStore) indexDir() string {
	return filepath.Join(ds.crawlDir, "index")
}

func (ds *DomainStore) openDB() (*sql.DB, error) {
	db, err := sql.Open("duckdb", ds.dbPath())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

const domainStoreSchema = `
CREATE TABLE IF NOT EXISTS parquet_file_versions (
	parquet_path TEXT PRIMARY KEY,
	file_mtime   TEXT NOT NULL DEFAULT '',
	file_size    BIGINT DEFAULT 0
);
CREATE TABLE IF NOT EXISTS domain_counts (
	domain       TEXT NOT NULL DEFAULT '',
	parquet_path TEXT NOT NULL DEFAULT '',
	shard        TEXT NOT NULL DEFAULT '',
	count        INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (domain, parquet_path)
);
CREATE INDEX IF NOT EXISTS idx_domain_counts_domain ON domain_counts(domain);
`

func (ds *DomainStore) initSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, domainStoreSchema)
	return err
}

// EnsureFresh checks whether a sync is needed and, if so, starts one in the
// background. Returns immediately — callers always get fast access to whatever
// cached data exists. The first call also initialises the DB.
func (ds *DomainStore) EnsureFresh(ctx context.Context) error {
	ds.mu.Lock()

	// Lazy-open DB on first call.
	if ds.db == nil {
		if err := os.MkdirAll(ds.crawlDir, 0o755); err != nil {
			ds.mu.Unlock()
			return fmt.Errorf("domain_store: mkdir: %w", err)
		}
		db, err := ds.openDB()
		if err != nil {
			ds.mu.Unlock()
			return fmt.Errorf("domain_store: open: %w", err)
		}
		if err := ds.initSchema(ctx, db); err != nil {
			db.Close()
			ds.mu.Unlock()
			return fmt.Errorf("domain_store: schema: %w", err)
		}
		ds.db = db
	}

	// If a sync is already running, return immediately — don't pile up goroutines.
	if ds.syncing {
		ds.mu.Unlock()
		return nil
	}

	ds.syncing = true
	ds.mu.Unlock()

	// Run sync in background so API calls are never blocked.
	go func() {
		ds.runSync()
		ds.mu.Lock()
		ds.syncing = false
		ds.synced = true
		ds.mu.Unlock()
	}()

	return nil
}

// IsSyncing reports whether a background sync is in progress.
func (ds *DomainStore) IsSyncing() bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.syncing
}

// runSync performs the full idempotent sync — called from a background goroutine.
func (ds *DomainStore) runSync() {
	ctx := context.Background()

	// Find all local parquet files.
	presentFiles := make(map[string]os.FileInfo)
	_ = filepath.WalkDir(ds.indexDir(), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".parquet") {
			if info, e := d.Info(); e == nil {
				presentFiles[path] = info
			}
		}
		return nil
	})

	ds.mu.Lock()
	db := ds.db
	ds.mu.Unlock()
	if db == nil {
		return
	}

	// Load known versions.
	rows, err := db.QueryContext(ctx, `SELECT parquet_path, file_mtime, file_size FROM parquet_file_versions`)
	if err != nil {
		return
	}
	type versionRow struct {
		mtime string
		size  int64
	}
	knownVersions := make(map[string]versionRow)
	for rows.Next() {
		var path, mtime string
		var size int64
		rows.Scan(&path, &mtime, &size)
		knownVersions[path] = versionRow{mtime, size}
	}
	rows.Close()

	// Remove deleted parquet files from cache.
	for path := range knownVersions {
		if _, ok := presentFiles[path]; !ok {
			db.ExecContext(ctx, `DELETE FROM domain_counts WHERE parquet_path = ?`, path)
			db.ExecContext(ctx, `DELETE FROM parquet_file_versions WHERE parquet_path = ?`, path)
		}
	}

	// Sync new / changed parquet files.
	for path, info := range presentFiles {
		mtime := info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
		size := info.Size()
		if v, ok := knownVersions[path]; ok && v.mtime == mtime && v.size == size {
			continue
		}
		_ = ds.syncParquetFile(ctx, db, path, mtime, size)
	}
}

// syncParquetFile aggregates domain counts from one parquet file into domains.duckdb.
func (ds *DomainStore) syncParquetFile(ctx context.Context, cacheDB *sql.DB, parquetPath, mtime string, size int64) error {
	shard := parquetShardName(parquetPath)

	// Open a throwaway in-memory DuckDB to read the parquet file.
	tmpDB, err := sql.Open("duckdb", "")
	if err != nil {
		return err
	}
	defer tmpDB.Close()
	tmpDB.SetMaxOpenConns(1)

	quoted := duckQuotePath(parquetPath)
	query := fmt.Sprintf(`
		SELECT url_host_registered_domain, COUNT(*) AS cnt
		FROM read_parquet(%s, hive_partitioning=true)
		WHERE url_host_registered_domain IS NOT NULL
		  AND url_host_registered_domain != ''
		GROUP BY url_host_registered_domain
	`, quoted)

	rows, err := tmpDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("domain_store: aggregate %s: %w", filepath.Base(parquetPath), err)
	}

	type aggRow struct {
		domain string
		count  int64
	}
	var agg []aggRow
	for rows.Next() {
		var r aggRow
		rows.Scan(&r.domain, &r.count)
		if r.domain != "" {
			agg = append(agg, r)
		}
	}
	rows.Close()

	// Replace old data for this parquet file.
	cacheDB.ExecContext(ctx, `DELETE FROM domain_counts WHERE parquet_path = ?`, parquetPath)

	const batchSize = 500
	for i := 0; i < len(agg); i += batchSize {
		end := i + batchSize
		if end > len(agg) {
			end = len(agg)
		}
		batch := agg[i:end]
		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)*4)
		for j, r := range batch {
			placeholders[j] = "(?,?,?,?)"
			args = append(args, r.domain, parquetPath, shard, r.count)
		}
		q := `INSERT OR REPLACE INTO domain_counts (domain, parquet_path, shard, count) VALUES ` +
			strings.Join(placeholders, ",")
		cacheDB.ExecContext(ctx, q, args...)
	}

	cacheDB.ExecContext(ctx,
		`INSERT OR REPLACE INTO parquet_file_versions (parquet_path, file_mtime, file_size) VALUES (?,?,?)`,
		parquetPath, mtime, size,
	)
	return nil
}

// parquetShardName extracts a short shard label from a parquet file path.
// Prefers the subset= hive partition value, falls back to the filename stem.
func parquetShardName(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range parts {
		if strings.HasPrefix(p, "subset=") {
			return strings.TrimPrefix(p, "subset=")
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ── Query methods ─────────────────────────────────────────────────────────────

// DomainRow is one entry in the domain list.
type DomainRow struct {
	Domain  string `json:"domain"`
	Count   int64  `json:"count"`
	Syncing bool   `json:"syncing,omitempty"`
}

// DomainsResponse is returned by GET /api/domains.
type DomainsResponse struct {
	Domains  []DomainRow `json:"domains"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Syncing  bool        `json:"syncing,omitempty"`
}

// ListDomains returns a paginated list of domains with total URL counts.
// sortBy: "count" (default, desc) | "alpha" (domain A→Z).
// q: optional substring filter on domain name.
func (ds *DomainStore) ListDomains(ctx context.Context, sortBy, q string, page, pageSize int) (DomainsResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	orderClause := "ORDER BY total DESC, domain ASC"
	if sortBy == "alpha" {
		orderClause = "ORDER BY domain ASC"
	}

	var whereClause string
	var filterArgs []any
	if q != "" {
		whereClause = "WHERE domain ILIKE ?"
		filterArgs = append(filterArgs, "%"+q+"%")
	}

	ds.mu.Lock()
	db := ds.db
	syncing := ds.syncing
	ds.mu.Unlock()

	if db == nil {
		return DomainsResponse{Syncing: syncing, Domains: []DomainRow{}}, nil
	}

	var total int64
	countSQL := fmt.Sprintf(`SELECT COUNT(DISTINCT domain) FROM domain_counts %s`, whereClause)
	if err := db.QueryRowContext(ctx, countSQL, filterArgs...).Scan(&total); err != nil {
		return DomainsResponse{}, err
	}

	listSQL := fmt.Sprintf(`
		SELECT domain, SUM(count) AS total
		FROM domain_counts
		%s
		GROUP BY domain
		%s
		LIMIT ? OFFSET ?
	`, whereClause, orderClause)
	listArgs := append(filterArgs, pageSize, offset)

	rows, err := db.QueryContext(ctx, listSQL, listArgs...)
	if err != nil {
		return DomainsResponse{}, err
	}
	defer rows.Close()

	domains := make([]DomainRow, 0)
	for rows.Next() {
		var d DomainRow
		rows.Scan(&d.Domain, &d.Count)
		domains = append(domains, d)
	}
	return DomainsResponse{Domains: domains, Total: total, Page: page, PageSize: pageSize, Syncing: syncing}, nil
}

// DomainDocRow is one URL entry in the domain detail page.
type DomainDocRow struct {
	URL         string `json:"url"`
	Shard       string `json:"shard"`
	FetchStatus int    `json:"fetch_status,omitempty"`
}

// DomainDetailResponse is returned by GET /api/domains/{domain}.
type DomainDetailResponse struct {
	Domain   string         `json:"domain"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Docs     []DomainDocRow `json:"docs"`
	Syncing  bool           `json:"syncing,omitempty"`
}

// ListDomainURLs queries parquet files directly for all URLs under a domain.
// LIMIT/OFFSET are pushed to DuckDB for efficiency — no full scan in Go memory.
// sortBy: "url" (default, asc) | "status".
func (ds *DomainStore) ListDomainURLs(ctx context.Context, domain, sortBy string, page, pageSize int) (DomainDetailResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	ds.mu.Lock()
	db := ds.db
	syncing := ds.syncing
	ds.mu.Unlock()

	if db == nil {
		return DomainDetailResponse{Domain: domain, Page: page, PageSize: pageSize, Syncing: syncing, Docs: []DomainDocRow{}}, nil
	}

	// Find which parquet files contain this domain.
	prows, err := db.QueryContext(ctx,
		`SELECT parquet_path, shard FROM domain_counts WHERE domain = ? AND count > 0`,
		domain,
	)
	if err != nil {
		return DomainDetailResponse{}, err
	}
	type parquetRef struct{ path, shard string }
	var refs []parquetRef
	for prows.Next() {
		var r parquetRef
		prows.Scan(&r.path, &r.shard)
		refs = append(refs, r)
	}
	prows.Close()

	if len(refs) == 0 {
		return DomainDetailResponse{Domain: domain, Page: page, PageSize: pageSize, Syncing: syncing, Docs: []DomainDocRow{}}, nil
	}

	// Query parquet files directly using an in-memory DuckDB session.
	tmpDB, err := sql.Open("duckdb", "")
	if err != nil {
		return DomainDetailResponse{}, err
	}
	defer tmpDB.Close()
	tmpDB.SetMaxOpenConns(1)

	quoted := make([]string, len(refs))
	for i, r := range refs {
		quoted[i] = duckQuotePath(r.path)
	}
	fileList := strings.Join(quoted, ", ")

	orderClause := "ORDER BY url ASC"
	if sortBy == "status" {
		orderClause = "ORDER BY fetch_status ASC, url ASC"
	}

	// Count total matching rows.
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM read_parquet([%s], union_by_name=true, hive_partitioning=true)
		WHERE url_host_registered_domain = ?
	`, fileList)
	var total int64
	tmpDB.QueryRowContext(ctx, countSQL, domain).Scan(&total)

	// Fetch the page — push LIMIT/OFFSET to DuckDB.
	// Note: crawl_date does not exist in CC parquet; omit it to avoid binder errors.
	pageSQL := fmt.Sprintf(`
		SELECT url, fetch_status
		FROM read_parquet([%s], union_by_name=true, hive_partitioning=true)
		WHERE url_host_registered_domain = ?
		%s
		LIMIT ? OFFSET ?
	`, fileList, orderClause)

	qrows, err := tmpDB.QueryContext(ctx, pageSQL, domain, pageSize, offset)
	if err != nil {
		return DomainDetailResponse{}, fmt.Errorf("domain_store: query parquet: %w", err)
	}
	defer qrows.Close()

	// Determine shard label (combine unique shards when multiple files involved).
	shardLabel := refs[0].shard
	if len(refs) > 1 {
		seen := make(map[string]bool)
		var names []string
		for _, r := range refs {
			if !seen[r.shard] {
				seen[r.shard] = true
				names = append(names, r.shard)
			}
		}
		sort.Strings(names)
		shardLabel = strings.Join(names, ",")
	}

	docs := make([]DomainDocRow, 0, pageSize)
	for qrows.Next() {
		var url string
		var status sql.NullInt64
		qrows.Scan(&url, &status)
		docs = append(docs, DomainDocRow{
			URL:         url,
			Shard:       shardLabel,
			FetchStatus: int(status.Int64),
		})
	}

	return DomainDetailResponse{
		Domain:   domain,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Docs:     docs,
		Syncing:  syncing,
	}, nil
}

// Close releases the underlying DuckDB connection.
func (ds *DomainStore) Close() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.db != nil {
		err := ds.db.Close()
		ds.db = nil
		return err
	}
	return nil
}
