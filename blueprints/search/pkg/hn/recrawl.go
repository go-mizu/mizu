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

type RecrawlSeedOptions struct {
	DomainsDBPath string
	OutDBPath     string
	Limit         int
	MaxPerDomain  int    // max URLs per domain (0=no limit); uses ROW_NUMBER() window for even sampling
	DomainLike    string
	Force         bool
	Progress      func(RecrawlSeedProgress)
}

type RecrawlSeedProgress struct {
	Stage   string
	Detail  string
	Rows    int64
	Elapsed time.Duration
}

type RecrawlSeedResult struct {
	DomainsDBPath string
	OutDBPath     string
	Rows          int64
	UniqueDomains int64
	Elapsed       time.Duration
}

func (c Config) RecrawlDir() string {
	return filepath.Join(c.WithDefaults().BaseDir(), "recrawl")
}

func (c Config) RecrawlSeedDBPath() string {
	return filepath.Join(c.RecrawlDir(), "hn_pages.duckdb")
}

func (c Config) BuildRecrawlSeedDB(ctx context.Context, opts RecrawlSeedOptions) (*RecrawlSeedResult, error) {
	cfg := c.WithDefaults()
	domainsDB := strings.TrimSpace(opts.DomainsDBPath)
	if domainsDB == "" {
		domainsDB = cfg.DomainsDBPath()
	}
	outDB := strings.TrimSpace(opts.OutDBPath)
	if outDB == "" {
		outDB = cfg.RecrawlSeedDBPath()
	}
	if !fileExistsNonEmpty(domainsDB) {
		return nil, fmt.Errorf("hn domains db not found: %s", domainsDB)
	}
	if err := os.MkdirAll(filepath.Dir(outDB), 0o755); err != nil {
		return nil, fmt.Errorf("create recrawl dir: %w", err)
	}
	if opts.Force {
		_ = os.Remove(outDB)
	}

	started := time.Now()
	emit := func(stage, detail string, rows int64) {
		if opts.Progress != nil {
			opts.Progress(RecrawlSeedProgress{
				Stage:   stage,
				Detail:  detail,
				Rows:    rows,
				Elapsed: time.Since(started),
			})
		}
	}

	db, err := sql.Open("duckdb", outDB)
	if err != nil {
		return nil, fmt.Errorf("open recrawl seed db: %w", err)
	}
	defer db.Close()

	emit("attach", "attaching hn domains db", 0)
	if _, err := db.ExecContext(ctx, `DETACH IF EXISTS src_hn_domains`); err != nil {
		// ignore
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`ATTACH '%s' AS src_hn_domains (READ_ONLY)`, escapeSQLString(domainsDB))); err != nil {
		return nil, fmt.Errorf("attach hn domains db: %w", err)
	}
	defer db.ExecContext(context.Background(), `DETACH IF EXISTS src_hn_domains`)

	emit("build", "materializing docs seeds from pages table", 0)
	var clauses []string
	clauses = append(clauses, "url IS NOT NULL")
	clauses = append(clauses, "length(trim(url)) > 0")
	clauses = append(clauses, "host IS NOT NULL")
	clauses = append(clauses, "length(trim(host)) > 0")
	if v := strings.TrimSpace(opts.DomainLike); v != "" {
		clauses = append(clauses, fmt.Sprintf("host ILIKE '%%%s%%'", escapeSQLString(v)))
	}
	whereSQL := strings.Join(clauses, " AND ")
	limitSQL := ""
	if opts.Limit > 0 {
		limitSQL = fmt.Sprintf("LIMIT %d", opts.Limit)
	}

	// docs schema matches recrawler.LoadSeedStats/LoadSeedURLs expectations.
	// When MaxPerDomain > 0, use ROW_NUMBER() window function for even sampling across domains.
	var createSQL string
	if opts.MaxPerDomain > 0 {
		createSQL = fmt.Sprintf(`CREATE OR REPLACE TABLE docs AS
SELECT url, domain, protocol, content_type, tld, item_id, item_time_ts
FROM (
  SELECT
    CAST(url AS VARCHAR) AS url,
    lower(CAST(host AS VARCHAR)) AS domain,
    lower(COALESCE(CAST(scheme AS VARCHAR), regexp_extract(CAST(url AS VARCHAR), '^([a-zA-Z][a-zA-Z0-9+.-]*):', 1))) AS protocol,
    CAST(NULL AS VARCHAR) AS content_type,
    lower(regexp_extract(CAST(host AS VARCHAR), '([^.]+)$', 1)) AS tld,
    CAST(item_id AS BIGINT) AS item_id,
    CAST(item_time_ts AS TIMESTAMP) AS item_time_ts,
    ROW_NUMBER() OVER (PARTITION BY lower(host) ORDER BY item_time_ts DESC NULLS LAST, item_id DESC) AS rn
  FROM src_hn_domains.pages
  WHERE %s
) sub
WHERE rn <= %d
ORDER BY domain, item_time_ts DESC NULLS LAST
%s`, whereSQL, opts.MaxPerDomain, limitSQL)
	} else {
		createSQL = fmt.Sprintf(`CREATE OR REPLACE TABLE docs AS
SELECT
  CAST(url AS VARCHAR) AS url,
  lower(CAST(host AS VARCHAR)) AS domain,
  lower(COALESCE(CAST(scheme AS VARCHAR), regexp_extract(CAST(url AS VARCHAR), '^([a-zA-Z][a-zA-Z0-9+.-]*):', 1))) AS protocol,
  CAST(NULL AS VARCHAR) AS content_type,
  lower(regexp_extract(CAST(host AS VARCHAR), '([^.]+)$', 1)) AS tld,
  CAST(item_id AS BIGINT) AS item_id,
  CAST(item_time_ts AS TIMESTAMP) AS item_time_ts
FROM src_hn_domains.pages
WHERE %s
ORDER BY item_time_ts DESC NULLS LAST, item_id DESC
%s`, whereSQL, limitSQL)
	}
	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		return nil, fmt.Errorf("build docs seeds table: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_docs_url ON docs(url)`); err != nil {
		// non-fatal
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_docs_domain ON docs(domain)`); err != nil {
		// non-fatal
	}

	var rowsN, domainsN int64
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*)::BIGINT, COUNT(DISTINCT domain)::BIGINT FROM docs`).Scan(&rowsN, &domainsN)
	emit("done", "seed docs ready", rowsN)
	return &RecrawlSeedResult{
		DomainsDBPath: domainsDB,
		OutDBPath:     outDB,
		Rows:          rowsN,
		UniqueDomains: domainsN,
		Elapsed:       time.Since(started),
	}, nil
}
