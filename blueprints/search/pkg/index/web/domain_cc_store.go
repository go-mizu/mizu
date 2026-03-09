package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
)

var ErrCCDomainNotFound = errors.New("cc domain not found in cache")

// CCDomainStore caches Common Crawl CDX URL lists in a dedicated DuckDB file.
// This is intentionally separate from domains.duckdb (parquet-derived cache).
type CCDomainStore struct {
	crawlDir string
	mu       sync.Mutex
	db       *sql.DB
}

func NewCCDomainStore(crawlDir string) *CCDomainStore {
	return &CCDomainStore{crawlDir: crawlDir}
}

func (cs *CCDomainStore) dbPath() string {
	return filepath.Join(cs.crawlDir, "domains_cc.duckdb")
}

func (cs *CCDomainStore) open(ctx context.Context) (*sql.DB, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.db != nil {
		return cs.db, nil
	}
	db, err := sql.Open("duckdb", cs.dbPath())
	if err != nil {
		return nil, fmt.Errorf("cc_domain_store: open: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, ccDomainSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cc_domain_store: schema: %w", err)
	}
	cs.db = db
	return db, nil
}

const ccDomainSchema = `
CREATE TABLE IF NOT EXISTS cc_domain_fetches (
	domain      TEXT NOT NULL,
	crawl_id    TEXT NOT NULL,
	fetched_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	total_urls  BIGINT NOT NULL DEFAULT 0,
	truncated   BOOLEAN NOT NULL DEFAULT FALSE,
	PRIMARY KEY (domain, crawl_id)
);
CREATE TABLE IF NOT EXISTS cc_domain_urls (
	domain       TEXT NOT NULL,
	crawl_id     TEXT NOT NULL,
	url          TEXT NOT NULL,
	fetch_status INTEGER NOT NULL DEFAULT 0,
	mime         TEXT NOT NULL DEFAULT '',
	ts           TEXT NOT NULL DEFAULT '',
	digest       TEXT NOT NULL DEFAULT '',
	rec_length   BIGINT NOT NULL DEFAULT 0,
	rec_offset   BIGINT NOT NULL DEFAULT 0,
	filename     TEXT NOT NULL DEFAULT '',
	language     TEXT NOT NULL DEFAULT '',
	encoding     TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (domain, crawl_id, url, ts)
);
CREATE INDEX IF NOT EXISTS idx_cc_domain_urls_domain_crawl ON cc_domain_urls(domain, crawl_id);
CREATE INDEX IF NOT EXISTS idx_cc_domain_urls_status ON cc_domain_urls(fetch_status);
`

type CCDomainFetchResponse struct {
	Domain      string `json:"domain"`
	CrawlID     string `json:"crawl_id"`
	TotalURLs   int64  `json:"total_urls"`
	Truncated   bool   `json:"truncated"`
	FetchedAt   string `json:"fetched_at"`
	TotalPages  int    `json:"total_pages"`
	FetchedPage int    `json:"fetched_pages"`
}

type CCDomainURLRow struct {
	URL          string `json:"url"`
	FetchStatus  int    `json:"fetch_status,omitempty"`
	Mime         string `json:"mime,omitempty"`
	Timestamp    string `json:"timestamp,omitempty"`
	Filename     string `json:"filename,omitempty"`
	RecordLength int64  `json:"record_length,omitempty"`
	RecordOffset int64  `json:"record_offset,omitempty"`
	Digest       string `json:"digest,omitempty"`
	Language     string `json:"language,omitempty"`
	Encoding     string `json:"encoding,omitempty"`
}

type CCDomainDetailResponse struct {
	Domain      string           `json:"domain"`
	CrawlID     string           `json:"crawl_id"`
	Total       int64            `json:"total"`
	Page        int              `json:"page"`
	PageSize    int              `json:"page_size"`
	Query       string           `json:"query,omitempty"`
	Sort        string           `json:"sort,omitempty"`
	StatusGroup string           `json:"status_group,omitempty"`
	CachedAt    string           `json:"cached_at,omitempty"`
	Truncated   bool             `json:"truncated,omitempty"`
	Stats       CCDomainStats    `json:"stats"`
	Docs        []CCDomainURLRow `json:"docs"`
}

type CCDomainStats struct {
	Total       int64            `json:"total"`
	StatusCodes []CCStatusBucket `json:"status_codes"`
	MimeTypes   []CCStringBucket `json:"mime_types"`
}

type CCStatusBucket struct {
	Code  int   `json:"code"`
	Count int64 `json:"count"`
}

type CCStringBucket struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// FetchAndCache queries Common Crawl CDX API for a domain and stores results in domains_cc.duckdb.
// crawlID may be empty to auto-resolve the latest crawl from collinfo.json.
func (cs *CCDomainStore) FetchAndCache(ctx context.Context, domain, crawlID string, maxURLs int) (CCDomainFetchResponse, error) {
	db, err := cs.open(ctx)
	if err != nil {
		return CCDomainFetchResponse{}, err
	}
	if crawlID == "" {
		latest, err := cc.ResolveLatestCrawlID(ctx, cc.Config{
			DataDir: filepath.Dir(cs.crawlDir),
		})
		if err != nil {
			return CCDomainFetchResponse{}, err
		}
		crawlID = latest.ID
	}
	if maxURLs <= 0 {
		maxURLs = 20000
	}

	totalPages, err := cc.CDXJPageCount(ctx, crawlID, domain)
	if err != nil {
		return CCDomainFetchResponse{}, err
	}

	var (
		entries      []cc.CDXJEntry
		fetchedPages int
		truncated    bool
	)
	for page := 0; page < totalPages; page++ {
		pageEntries, err := cc.LookupDomainPage(ctx, crawlID, domain, page)
		if err != nil {
			return CCDomainFetchResponse{}, err
		}
		fetchedPages++
		entries = append(entries, pageEntries...)
		if len(entries) >= maxURLs {
			entries = entries[:maxURLs]
			truncated = true
			break
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return CCDomainFetchResponse{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM cc_domain_urls WHERE domain = ? AND crawl_id = ?`, domain, crawlID); err != nil {
		return CCDomainFetchResponse{}, err
	}

	const batchSize = 400
	for i := 0; i < len(entries); i += batchSize {
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[i:end]
		placeholders := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*12)
		for _, e := range batch {
			placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args,
				domain, crawlID, e.URL,
				atoiSafe(e.Status),
				e.Mime,
				e.Timestamp,
				e.Digest,
				atollSafe(e.Length),
				atollSafe(e.Offset),
				e.Filename,
				e.Languages,
				e.Encoding,
			)
		}
		q := `INSERT OR REPLACE INTO cc_domain_urls
			(domain,crawl_id,url,fetch_status,mime,ts,digest,rec_length,rec_offset,filename,language,encoding)
			VALUES ` + strings.Join(placeholders, ",")
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return CCDomainFetchResponse{}, err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO cc_domain_fetches (domain, crawl_id, fetched_at, total_urls, truncated)
		VALUES (?, ?, CURRENT_TIMESTAMP, ?, ?)
	`, domain, crawlID, len(entries), truncated); err != nil {
		return CCDomainFetchResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return CCDomainFetchResponse{}, err
	}

	var fetchedAt time.Time
	_ = db.QueryRowContext(ctx,
		`SELECT fetched_at FROM cc_domain_fetches WHERE domain = ? AND crawl_id = ?`,
		domain, crawlID,
	).Scan(&fetchedAt)

	return CCDomainFetchResponse{
		Domain:      domain,
		CrawlID:     crawlID,
		TotalURLs:   int64(len(entries)),
		Truncated:   truncated,
		FetchedAt:   fetchedAt.UTC().Format(time.RFC3339),
		TotalPages:  totalPages,
		FetchedPage: fetchedPages,
	}, nil
}

func (cs *CCDomainStore) GetDomainURLs(ctx context.Context, domain, crawlID, sortBy, statusGroup, q string, page, pageSize int) (CCDomainDetailResponse, error) {
	db, err := cs.open(ctx)
	if err != nil {
		return CCDomainDetailResponse{}, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}
	if crawlID == "" {
		if err := db.QueryRowContext(ctx, `
			SELECT crawl_id
			FROM cc_domain_fetches
			WHERE domain = ?
			ORDER BY fetched_at DESC
			LIMIT 1
		`, domain).Scan(&crawlID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CCDomainDetailResponse{}, ErrCCDomainNotFound
			}
			return CCDomainDetailResponse{}, err
		}
	}

	orderClause := "ORDER BY url ASC"
	switch sortBy {
	case "status":
		orderClause = "ORDER BY fetch_status ASC, url ASC"
	case "newest":
		orderClause = "ORDER BY ts DESC, url ASC"
	}

	whereBase := "WHERE domain = ? AND crawl_id = ?"
	args := []any{domain, crawlID}
	switch strings.TrimSpace(strings.ToLower(statusGroup)) {
	case "2xx":
		statusGroup = "2xx"
	case "3xx":
		statusGroup = "3xx"
	case "4xx":
		statusGroup = "4xx"
	case "5xx":
		statusGroup = "5xx"
	case "other":
		statusGroup = "other"
	default:
		statusGroup = ""
	}
	if strings.TrimSpace(q) != "" {
		whereBase += " AND url ILIKE ?"
		args = append(args, "%"+q+"%")
	}
	where := whereBase
	switch statusGroup {
	case "2xx":
		where += " AND fetch_status BETWEEN 200 AND 299"
	case "3xx":
		where += " AND fetch_status BETWEEN 300 AND 399"
	case "4xx":
		where += " AND fetch_status BETWEEN 400 AND 499"
	case "5xx":
		where += " AND fetch_status BETWEEN 500 AND 599"
	case "other":
		where += " AND (fetch_status < 200 OR fetch_status >= 600)"
	}

	var total int64
	countSQL := `SELECT COUNT(*) FROM cc_domain_urls ` + where
	if err := db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return CCDomainDetailResponse{}, err
	}
	if total == 0 {
		var exists int
		if err := db.QueryRowContext(ctx, `SELECT 1 FROM cc_domain_fetches WHERE domain=? AND crawl_id=? LIMIT 1`, domain, crawlID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CCDomainDetailResponse{}, ErrCCDomainNotFound
			}
			return CCDomainDetailResponse{}, err
		}
	}

	var (
		cachedAt  time.Time
		truncated bool
	)
	_ = db.QueryRowContext(ctx, `
		SELECT fetched_at, truncated
		FROM cc_domain_fetches
		WHERE domain = ? AND crawl_id = ?
	`, domain, crawlID).Scan(&cachedAt, &truncated)

	offset := (page - 1) * pageSize
	pageSQL := `
		SELECT url, fetch_status, mime, ts, filename, rec_length, rec_offset, digest, language, encoding
		FROM cc_domain_urls
		` + where + `
		` + orderClause + `
		LIMIT ? OFFSET ?
	`
	pageArgs := append(args, pageSize, offset)
	rows, err := db.QueryContext(ctx, pageSQL, pageArgs...)
	if err != nil {
		return CCDomainDetailResponse{}, err
	}
	defer rows.Close()

	docs := make([]CCDomainURLRow, 0, pageSize)
	for rows.Next() {
		var r CCDomainURLRow
		_ = rows.Scan(
			&r.URL,
			&r.FetchStatus,
			&r.Mime,
			&r.Timestamp,
			&r.Filename,
			&r.RecordLength,
			&r.RecordOffset,
			&r.Digest,
			&r.Language,
			&r.Encoding,
		)
		docs = append(docs, r)
	}

	var statsTotal int64
	countAllSQL := `SELECT COUNT(*) FROM cc_domain_urls ` + whereBase
	if err := db.QueryRowContext(ctx, countAllSQL, args...).Scan(&statsTotal); err != nil {
		return CCDomainDetailResponse{}, err
	}
	stats := CCDomainStats{Total: statsTotal}
	statusRows, err := db.QueryContext(ctx, `
		SELECT fetch_status, COUNT(*) AS cnt
		FROM cc_domain_urls
		`+whereBase+`
		GROUP BY fetch_status
		ORDER BY cnt DESC, fetch_status ASC
	`, args...)
	if err != nil {
		return CCDomainDetailResponse{}, err
	}
	for statusRows.Next() {
		var b CCStatusBucket
		_ = statusRows.Scan(&b.Code, &b.Count)
		stats.StatusCodes = append(stats.StatusCodes, b)
	}
	_ = statusRows.Close()

	mimeRows, err := db.QueryContext(ctx, `
		SELECT mime, COUNT(*) AS cnt
		FROM cc_domain_urls
		`+whereBase+`
		AND mime != ''
		GROUP BY mime
		ORDER BY cnt DESC, mime ASC
		LIMIT 12
	`, args...)
	if err != nil {
		return CCDomainDetailResponse{}, err
	}
	for mimeRows.Next() {
		var b CCStringBucket
		_ = mimeRows.Scan(&b.Key, &b.Count)
		stats.MimeTypes = append(stats.MimeTypes, b)
	}
	_ = mimeRows.Close()

	return CCDomainDetailResponse{
		Domain:      domain,
		CrawlID:     crawlID,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		Query:       q,
		Sort:        sortBy,
		StatusGroup: statusGroup,
		CachedAt:    cachedAt.UTC().Format(time.RFC3339),
		Truncated:   truncated,
		Stats:       stats,
		Docs:        docs,
	}, nil
}

func (cs *CCDomainStore) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.db == nil {
		return nil
	}
	err := cs.db.Close()
	cs.db = nil
	return err
}

func atoiSafe(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func atollSafe(s string) int64 {
	v, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return v
}
