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
// EnsureFresh is idempotent: it re-syncs only parquet files whose mtime or
// size has changed since the last sync.
type DomainStore struct {
	crawlDir string // ~/data/common-crawl/{crawlID}
	mu       sync.Mutex
	db       *sql.DB // lazily opened domains.duckdb
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

// EnsureFresh syncs stale or new parquet files into domains.duckdb.
// Removed parquet files are cleaned up. Safe to call concurrently.
func (ds *DomainStore) EnsureFresh(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Lazy-open DB.
	if ds.db == nil {
		if err := os.MkdirAll(ds.crawlDir, 0o755); err != nil {
			return fmt.Errorf("domain_store: mkdir: %w", err)
		}
		db, err := ds.openDB()
		if err != nil {
			return fmt.Errorf("domain_store: open: %w", err)
		}
		if err := ds.initSchema(ctx, db); err != nil {
			db.Close()
			return fmt.Errorf("domain_store: schema: %w", err)
		}
		ds.db = db
	}

	// Find all local parquet files.
	presentFiles := make(map[string]os.FileInfo) // path → stat
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

	// Load known versions.
	rows, err := ds.db.QueryContext(ctx, `SELECT parquet_path, file_mtime, file_size FROM parquet_file_versions`)
	if err != nil {
		return nil // non-fatal
	}
	type versionRow struct{ mtime string; size int64 }
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
			ds.db.ExecContext(ctx, `DELETE FROM domain_counts WHERE parquet_path = ?`, path)
			ds.db.ExecContext(ctx, `DELETE FROM parquet_file_versions WHERE parquet_path = ?`, path)
		}
	}

	// Sync new / changed parquet files.
	for path, info := range presentFiles {
		mtime := info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
		size := info.Size()
		if v, ok := knownVersions[path]; ok && v.mtime == mtime && v.size == size {
			continue // up to date
		}
		_ = ds.syncParquetFile(ctx, path, mtime, size)
	}
	return nil
}

// syncParquetFile aggregates domain counts from one parquet file into domains.duckdb.
func (ds *DomainStore) syncParquetFile(ctx context.Context, parquetPath, mtime string, size int64) error {
	// Extract shard name from hive partition or filename.
	shard := parquetShardName(parquetPath)

	// Open a throwaway DuckDB to read the parquet file.
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
	defer rows.Close()

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
	ds.db.ExecContext(ctx, `DELETE FROM domain_counts WHERE parquet_path = ?`, parquetPath)

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
		ds.db.ExecContext(ctx, q, args...)
	}

	ds.db.ExecContext(ctx,
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
	Domain string `json:"domain"`
	Count  int64  `json:"count"`
}

// DomainsResponse is returned by GET /api/domains.
type DomainsResponse struct {
	Domains  []DomainRow `json:"domains"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
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

	var total int64
	countSQL := fmt.Sprintf(`SELECT COUNT(DISTINCT domain) FROM domain_counts %s`, whereClause)
	if err := ds.db.QueryRowContext(ctx, countSQL, filterArgs...).Scan(&total); err != nil {
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

	rows, err := ds.db.QueryContext(ctx, listSQL, listArgs...)
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
	return DomainsResponse{Domains: domains, Total: total, Page: page, PageSize: pageSize}, nil
}

// DomainDocRow is one URL entry in the domain detail page.
type DomainDocRow struct {
	URL         string `json:"url"`
	Shard       string `json:"shard"`
	FetchStatus int    `json:"fetch_status,omitempty"`
	CrawlDate   string `json:"crawl_date,omitempty"`
}

// DomainDetailResponse is returned by GET /api/domains/{domain}.
type DomainDetailResponse struct {
	Domain   string         `json:"domain"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Docs     []DomainDocRow `json:"docs"`
}

// ListDomainURLs queries parquet files directly for all URLs under a domain.
// sortBy: "url" (default, asc) | "status".
func (ds *DomainStore) ListDomainURLs(ctx context.Context, domain, sortBy string, page, pageSize int) (DomainDetailResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}

	// Find which parquet files contain this domain.
	rows, err := ds.db.QueryContext(ctx,
		`SELECT parquet_path, shard FROM domain_counts WHERE domain = ? AND count > 0`,
		domain,
	)
	if err != nil {
		return DomainDetailResponse{}, err
	}
	type parquetRef struct{ path, shard string }
	var refs []parquetRef
	for rows.Next() {
		var r parquetRef
		rows.Scan(&r.path, &r.shard)
		refs = append(refs, r)
	}
	rows.Close()

	if len(refs) == 0 {
		return DomainDetailResponse{Domain: domain, Page: page, PageSize: pageSize, Docs: []DomainDocRow{}}, nil
	}

	// Query all parquet files for this domain in a single DuckDB session.
	tmpDB, err := sql.Open("duckdb", "")
	if err != nil {
		return DomainDetailResponse{}, err
	}
	defer tmpDB.Close()
	tmpDB.SetMaxOpenConns(1)

	// Build quoted list.
	quoted := make([]string, len(refs))
	for i, r := range refs {
		quoted[i] = duckQuotePath(r.path)
	}
	fileList := strings.Join(quoted, ", ")

	orderClause := "ORDER BY url ASC"
	if sortBy == "status" {
		orderClause = "ORDER BY fetch_status ASC, url ASC"
	}

	// Collect all matching docs (for a single domain the count is manageable).
	// We query with LIMIT/OFFSET pushed down when possible.
	type rawRow struct {
		url         string
		fetchStatus int
		crawlDate   string
		shard       string
	}

	// Build a shard lookup by path.
	shardByPath := make(map[string]string, len(refs))
	for _, r := range refs {
		shardByPath[r.path] = r.shard
	}

	// Use hive_partitioning to get subset column automatically if available,
	// otherwise fall back to the shard we already know.
	selectSQL := fmt.Sprintf(`
		SELECT url, COALESCE(fetch_status, 0) AS fetch_status, COALESCE(crawl_date, '') AS crawl_date
		FROM read_parquet([%s], union_by_name=true, hive_partitioning=true)
		WHERE url_host_registered_domain = ?
		%s
	`, fileList, orderClause)

	qrows, err := tmpDB.QueryContext(ctx, selectSQL, domain)
	if err != nil {
		return DomainDetailResponse{}, fmt.Errorf("domain_store: query parquet: %w", err)
	}
	defer qrows.Close()

	var all []rawRow
	for qrows.Next() {
		var r rawRow
		qrows.Scan(&r.url, &r.fetchStatus, &r.crawlDate)
		all = append(all, r)
	}
	qrows.Close()

	total := int64(len(all))
	offset := (page - 1) * pageSize

	// Apply pagination.
	if offset >= len(all) {
		return DomainDetailResponse{Domain: domain, Total: total, Page: page, PageSize: pageSize, Docs: []DomainDocRow{}}, nil
	}
	end := offset + pageSize
	if end > len(all) {
		end = len(all)
	}
	slice := all[offset:end]

	// For each URL, determine shard by matching against parquet refs.
	// Since we query multiple files with union, use the first shard as fallback.
	defaultShard := ""
	if len(refs) > 0 {
		defaultShard = refs[0].shard
	}

	docs := make([]DomainDocRow, len(slice))
	for i, r := range slice {
		docs[i] = DomainDocRow{
			URL:         r.url,
			Shard:       defaultShard,
			FetchStatus: r.fetchStatus,
			CrawlDate:   r.crawlDate,
		}
	}

	// Sort shard list for stable shard labelling when refs > 1.
	if len(refs) > 1 {
		// Assign shard based on URL prefix match — best effort.
		// Build a sorted list of (urlPrefix, shard) from each parquet file.
		// For simplicity, label shard as comma-joined unique shard names.
		shardNames := make([]string, 0, len(refs))
		seen := make(map[string]bool)
		for _, r := range refs {
			if !seen[r.shard] {
				seen[r.shard] = true
				shardNames = append(shardNames, r.shard)
			}
		}
		sort.Strings(shardNames)
		combined := strings.Join(shardNames, ",")
		for i := range docs {
			docs[i].Shard = combined
		}
	}

	return DomainDetailResponse{
		Domain:   domain,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Docs:     docs,
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
