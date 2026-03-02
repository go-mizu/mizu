package hn

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const hnDomainsPagesSchemaVersion = "2"

type DomainsOptions struct {
	SourceDBPath string
	OutDBPath    string
	ForcePages   bool
	Progress     func(DomainsProgress)
}

type DomainsProgress struct {
	Stage   string
	Detail  string
	Rows    int64
	Rows2   int64
	Elapsed time.Duration
}

type DomainsResult struct {
	SourceDBPath    string
	OutDBPath       string
	SourceRows      int64
	SourceMaxID     int64
	SourceMaxTime   string
	SourceLinkItems int64
	PagesRows       int64
	DomainsRows     int64
	PagesBuilt      bool
	PagesReused     bool
	DomainsBuilt    bool
	Elapsed         time.Duration
}

func (c Config) DomainsDBPath() string {
	return filepath.Join(c.WithDefaults().BaseDir(), "hn_domains.duckdb")
}

func (c Config) BuildDomains(ctx context.Context, opts DomainsOptions) (*DomainsResult, error) {
	cfg := c.WithDefaults()
	srcPath := strings.TrimSpace(opts.SourceDBPath)
	if srcPath == "" {
		srcPath = cfg.DefaultDBPath()
	}
	outPath := strings.TrimSpace(opts.OutDBPath)
	if outPath == "" {
		outPath = cfg.DomainsDBPath()
	}
	if !fileExistsNonEmpty(srcPath) {
		return nil, fmt.Errorf("source hn duckdb not found: %s", srcPath)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("create domains db dir: %w", err)
	}

	started := time.Now()
	emit := func(stage, detail string, rows, rows2 int64) {
		if opts.Progress != nil {
			opts.Progress(DomainsProgress{
				Stage:   stage,
				Detail:  detail,
				Rows:    rows,
				Rows2:   rows2,
				Elapsed: time.Since(started),
			})
		}
	}

	emit("start", "opening source/output duckdb", 0, 0)
	srcSig, err := readDomainsSourceSignature(ctx, srcPath)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("duckdb", outPath)
	if err != nil {
		return nil, fmt.Errorf("open domains duckdb: %w", err)
	}
	defer db.Close()

	emit("attach", "attaching source hn database", 0, 0)
	if _, err := db.ExecContext(ctx, `INSTALL parquet; LOAD parquet;`); err != nil {
		// Best-effort; extension is often built-in. Ignore failures.
	}
	if _, err := db.ExecContext(ctx, `DETACH IF EXISTS src_hn`); err != nil {
		// ignore
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`ATTACH '%s' AS src_hn (READ_ONLY)`, escapeSQLString(srcPath))); err != nil {
		return nil, fmt.Errorf("attach source hn db: %w", err)
	}
	defer db.ExecContext(context.Background(), `DETACH IF EXISTS src_hn`)

	if err := ensureDomainsMetaTable(ctx, db); err != nil {
		return nil, err
	}

	reusePages := false
	if !opts.ForcePages {
		if ok, err := canReusePages(ctx, db, srcSig); err == nil && ok {
			reusePages = true
		}
	}

	res := &DomainsResult{
		SourceDBPath:    srcPath,
		OutDBPath:       outPath,
		SourceRows:      srcSig.SourceRows,
		SourceMaxID:     srcSig.SourceMaxID,
		SourceMaxTime:   srcSig.SourceMaxTime,
		SourceLinkItems: srcSig.SourceLinkItems,
	}

	if reusePages {
		emit("pages", "reusing existing pages table (source unchanged)", 0, 0)
		res.PagesReused = true
	} else {
		emit("pages", "building pages table from src_hn.items", 0, 0)
		if err := rebuildPagesTable(ctx, db); err != nil {
			return nil, err
		}
		res.PagesBuilt = true
		emit("pages", "pages table built", 0, 0)
	}

	emit("domains", "aggregating domains from pages", 0, 0)
	if err := rebuildDomainsTable(ctx, db); err != nil {
		return nil, err
	}
	res.DomainsBuilt = true

	if err := writeDomainsMeta(ctx, db, srcSig); err != nil {
		return nil, err
	}

	_ = db.QueryRowContext(ctx, `SELECT COUNT(*)::BIGINT FROM pages`).Scan(&res.PagesRows)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*)::BIGINT FROM domains`).Scan(&res.DomainsRows)
	res.Elapsed = time.Since(started)
	emit("done", "completed domains export", res.PagesRows, res.DomainsRows)
	return res, nil
}

type domainsSourceSignature struct {
	SourceRows      int64
	SourceMaxID     int64
	SourceMaxTime   string
	SourceLinkItems int64
	SourceDBSize    int64
	SourceDBMTime   int64
}

func readDomainsSourceSignature(ctx context.Context, srcPath string) (*domainsSourceSignature, error) {
	db, err := sql.Open("duckdb", srcPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open source hn duckdb: %w", err)
	}
	defer db.Close()

	var rows, maxID, linkItems sql.NullInt64
	var maxTime sql.NullString
	q := `SELECT
  COUNT(*)::BIGINT AS rows_n,
  MAX(id) AS max_id,
  CAST(MAX(time_ts) AS VARCHAR) AS max_time,
  COUNT(*) FILTER (WHERE url IS NOT NULL AND length(trim(url)) > 0)::BIGINT AS link_items
FROM items`
	if err := db.QueryRowContext(ctx, q).Scan(&rows, &maxID, &maxTime, &linkItems); err != nil {
		return nil, fmt.Errorf("read source hn db stats: %w", err)
	}
	st, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("stat source hn db: %w", err)
	}
	sig := &domainsSourceSignature{
		SourceDBSize:  st.Size(),
		SourceDBMTime: st.ModTime().Unix(),
	}
	if rows.Valid {
		sig.SourceRows = rows.Int64
	}
	if maxID.Valid {
		sig.SourceMaxID = maxID.Int64
	}
	if maxTime.Valid {
		sig.SourceMaxTime = maxTime.String
	}
	if linkItems.Valid {
		sig.SourceLinkItems = linkItems.Int64
	}
	return sig, nil
}

func ensureDomainsMetaTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS build_meta (
key VARCHAR PRIMARY KEY,
value VARCHAR
)`)
	if err != nil {
		return fmt.Errorf("create build_meta: %w", err)
	}
	return nil
}

func canReusePages(ctx context.Context, db *sql.DB, sig *domainsSourceSignature) (bool, error) {
	if !tableExists(ctx, db, "pages") {
		return false, nil
	}
	if !tableExists(ctx, db, "build_meta") {
		return false, nil
	}
	want := map[string]string{
		"pages_schema_version": hnDomainsPagesSchemaVersion,
		"source_rows":          fmt.Sprintf("%d", sig.SourceRows),
		"source_max_id":        fmt.Sprintf("%d", sig.SourceMaxID),
		"source_max_time":      sig.SourceMaxTime,
		"source_db_size":       fmt.Sprintf("%d", sig.SourceDBSize),
		"source_db_mtime":      fmt.Sprintf("%d", sig.SourceDBMTime),
		"source_link_items":    fmt.Sprintf("%d", sig.SourceLinkItems),
	}
	for k, v := range want {
		var got sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT value FROM build_meta WHERE key = ?`, k).Scan(&got); err != nil {
			return false, nil
		}
		if !got.Valid || got.String != v {
			return false, nil
		}
	}
	return true, nil
}

func writeDomainsMeta(ctx context.Context, db *sql.DB, sig *domainsSourceSignature) error {
	rows := map[string]string{
		"pages_schema_version": hnDomainsPagesSchemaVersion,
		"source_rows":          fmt.Sprintf("%d", sig.SourceRows),
		"source_max_id":        fmt.Sprintf("%d", sig.SourceMaxID),
		"source_max_time":      sig.SourceMaxTime,
		"source_db_size":       fmt.Sprintf("%d", sig.SourceDBSize),
		"source_db_mtime":      fmt.Sprintf("%d", sig.SourceDBMTime),
		"source_link_items":    fmt.Sprintf("%d", sig.SourceLinkItems),
		"built_at":             time.Now().UTC().Format(time.RFC3339),
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin meta tx: %w", err)
	}
	defer tx.Rollback()
	for k, v := range rows {
		if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO build_meta (key, value) VALUES (?, ?)`, k, v); err != nil {
			return fmt.Errorf("upsert build_meta %s: %w", k, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit meta tx: %w", err)
	}
	return nil
}

func rebuildPagesTable(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`DROP TABLE IF EXISTS pages`,
		`CREATE TABLE pages AS
WITH src AS (
  SELECT
    id AS item_id,
    CAST(type AS VARCHAR) AS item_type,
    CAST("by" AS VARCHAR) AS item_by,
    CAST(time AS BIGINT) AS item_time,
    CAST(time_ts AS TIMESTAMP) AS item_time_ts,
    CAST(title AS VARCHAR) AS item_title,
    CAST(score AS BIGINT) AS item_score,
    CAST(descendants AS BIGINT) AS item_descendants,
    CAST(parent AS BIGINT) AS item_parent,
    trim(CAST(url AS VARCHAR)) AS url
  FROM src_hn.items
  WHERE url IS NOT NULL AND length(trim(CAST(url AS VARCHAR))) > 0
),
parsed AS (
  SELECT
    *,
    lower(regexp_extract(url, '^([a-zA-Z][a-zA-Z0-9+.-]*):', 1)) AS scheme,
    lower(regexp_extract(url, '^(?:[a-zA-Z][a-zA-Z0-9+.-]*:)?//(?:[^@/?#]*@)?(\\[[^\\]]+\\]|[^:/?#]+)', 1)) AS host,
    regexp_extract(url, '^(?:[a-zA-Z][a-zA-Z0-9+.-]*:)?//[^/?#]+([^?#]*)', 1) AS path,
    regexp_extract(url, '^[^#]*\\?([^#]*)', 1) AS query
  FROM src
)
SELECT
  item_id,
  item_type,
  item_by,
  item_time,
  item_time_ts,
  item_title,
  item_score,
  item_descendants,
  item_parent,
  url,
  scheme,
  host,
  path,
  query,
  (scheme = 'https') AS is_https,
  length(url)::INTEGER AS url_len
FROM parsed`,
		`CREATE INDEX IF NOT EXISTS idx_pages_item_id ON pages(item_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pages_host ON pages(host)`,
		`CREATE INDEX IF NOT EXISTS idx_pages_item_time ON pages(item_time)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("build pages table: %w", err)
		}
	}
	return nil
}

func rebuildDomainsTable(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`DROP TABLE IF EXISTS domains`,
		`CREATE TABLE domains AS
WITH p AS (
  SELECT *,
         (coalesce(item_time, 0)::BIGINT * 1000000000::BIGINT + item_id) AS sort_key
  FROM pages
  WHERE host IS NOT NULL AND length(trim(host)) > 0
)
SELECT
  host AS domain,
  COUNT(*)::BIGINT AS item_count,
  COUNT(*) FILTER (WHERE item_type = 'story')::BIGINT AS story_count,
  COUNT(*) FILTER (WHERE item_type = 'job')::BIGINT AS job_count,
  COUNT(*) FILTER (WHERE item_type = 'comment')::BIGINT AS comment_count,
  COUNT(*) FILTER (WHERE is_https)::BIGINT AS https_count,
  MIN(item_id) AS min_item_id,
  MAX(item_id) AS max_item_id,
  CAST(MIN(item_time_ts) AS TIMESTAMP) AS first_item_time_ts,
  CAST(MAX(item_time_ts) AS TIMESTAMP) AS latest_item_time_ts,
  CAST(MAX(item_time_ts) AS DATE) AS latest_item_date,
  arg_min(item_id, sort_key) AS first_item_id,
  arg_min(url, sort_key) AS first_url,
  arg_min(item_title, sort_key) AS first_title,
  arg_max(item_id, sort_key) AS latest_item_id,
  arg_max(url, sort_key) AS latest_url,
  arg_max(item_title, sort_key) AS latest_title,
  arg_max(item_type, sort_key) AS latest_item_type
FROM p
GROUP BY 1
ORDER BY item_count DESC, domain ASC`,
		`CREATE INDEX IF NOT EXISTS idx_domains_domain ON domains(domain)`,
		`CREATE INDEX IF NOT EXISTS idx_domains_item_count ON domains(item_count)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("build domains table: %w", err)
		}
	}
	return nil
}

func tableExists(ctx context.Context, db *sql.DB, name string) bool {
	var n int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*)::BIGINT FROM information_schema.tables WHERE table_name = ?`, name).Scan(&n); err != nil {
		return false
	}
	return n > 0
}
