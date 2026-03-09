package scrape

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
)

// Store reads per-domain crawl metadata from dcrawler result databases.
type Store interface {
	ListDomains() (*ListResponse, error)
	GetDomainStats(domain string) (*Domain, error)
	GetPages(domain string, page, pageSize int, q, sortBy, statusFilter string) (*PagesResponse, error)
	GetDomainSummary(domain string) *DomainSummary
	InvalidateCache()
	Close()
}

// Compile-time check.
var _ Store = (*store)(nil)

// store is the concrete implementation of Store.
// Stats are materialized in a DuckDB meta file and served from memory.
type store struct {
	dataDir   string
	mu        sync.RWMutex
	domains   []Domain     // in-memory cache, populated by background refresh
	ready     bool         // true after first load/refresh
	triggerCh chan struct{} // signal immediate refresh
	stopCh    chan struct{} // signal goroutine to stop
}

// NewStore creates a Store and starts background stat refresh.
func NewStore(dataDir string) Store {
	s := &store{
		dataDir:   dataDir,
		triggerCh: make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
	}
	// Load persisted stats from meta DB for instant startup.
	loadFromMeta(s)
	// Start background goroutine to refresh stats every 60s.
	startBackground(s)
	return s
}

// Close stops the background goroutine.
func (s *store) Close() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
}

// InvalidateCache triggers an immediate background refresh.
func (s *store) InvalidateCache() {
	select {
	case s.triggerCh <- struct{}{}:
	default:
	}
}

// Domain holds summary stats for a scraped domain.
type Domain struct {
	Domain     string    `json:"domain"`
	Pages      int64     `json:"pages"`
	Success    int64     `json:"success"`
	Failed     int64     `json:"failed"`
	Links      int64     `json:"links"`
	HtmlBytes  int64     `json:"html_bytes"`
	MdBytes    int64     `json:"md_bytes"`
	IndexBytes int64     `json:"index_bytes"`
	LastCrawl  time.Time `json:"last_crawl"`
	HasMD      bool      `json:"has_markdown"`
	HasIndex   bool      `json:"has_index"`
}

// Page represents a single scraped page for the dashboard.
type Page struct {
	URL           string `json:"url"`
	URLHash       int64  `json:"url_hash"`
	StatusCode    int    `json:"status_code"`
	ContentType   string `json:"content_type"`
	ContentLength int64  `json:"content_length"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Language      string `json:"language"`
	FetchTimeMs   int64  `json:"fetch_time_ms"`
	CrawledAt     string `json:"crawled_at"`
	Error         string `json:"error,omitempty"`
	MdSize        int64  `json:"md_size"`
}

// DomainSummary holds per-status-group counts and size averages for a domain.
type DomainSummary struct {
	Status2xx int64 `json:"status_2xx"`
	Status3xx int64 `json:"status_3xx"`
	Status4xx int64 `json:"status_4xx"`
	Status5xx int64 `json:"status_5xx"`
	StatusErr int64 `json:"status_error"`
	AvgSize   int64 `json:"avg_size"`
	AvgMdSize int64 `json:"avg_md_size"`
}

// ListResponse is the response body for listing scraped domains.
type ListResponse struct {
	Domains []Domain `json:"domains"`
	Total   int      `json:"total"`
}

// PagesResponse is the response body for listing pages of a domain.
type PagesResponse struct {
	Domain   string `json:"domain"`
	Pages    []Page `json:"pages"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// DomainStatus is the response body for a domain status query.
type DomainStatus struct {
	Domain    string         `json:"domain"`
	Stats     *Domain        `json:"stats,omitempty"`
	Summary   *DomainSummary `json:"summary,omitempty"`
	ActiveJob *JobInfo       `json:"active_job,omitempty"`
	HasData   bool           `json:"has_data"`
}

// JobInfo is an embedded active-job summary in DomainStatus.
type JobInfo struct {
	ID       string  `json:"id"`
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
	Rate     float64 `json:"rate"`
}

// StartParams holds per-request scrape configuration packed into JobConfig.Source as JSON.
type StartParams struct {
	Mode             string `json:"mode"`
	MaxPages         int    `json:"max_pages"`
	MaxDepth         int    `json:"max_depth"`
	Workers          int    `json:"workers"`
	TimeoutS         int    `json:"timeout_s"`
	StoreBody        bool   `json:"store_body"`
	Resume           bool   `json:"resume"`
	NoRobots         bool   `json:"no_robots"`
	NoSitemap        bool   `json:"no_sitemap"`
	IncludeSubdomain bool   `json:"include_subdomain"`
	ScrollCount      int    `json:"scroll_count"`
	Continuous       bool   `json:"continuous"`
	StaleHours       int    `json:"stale_hours"`
	SeedURL          string `json:"seed_url"`
	// Worker mode — proxy fetches through CF Worker.
	WorkerToken   string `json:"worker_token,omitempty"`
	WorkerURL     string `json:"worker_url,omitempty"`
	WorkerBrowser bool   `json:"worker_browser,omitempty"`
}

// ParseStartParams decodes a JSON-encoded StartParams from JobConfig.Source.
func ParseStartParams(source string) StartParams {
	var p StartParams
	if source != "" {
		_ = json.Unmarshal([]byte(source), &p)
	}
	return p
}

// ListDomains returns cached domain stats from memory (instant).
func (s *store) ListDomains() (*ListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready {
		return &ListResponse{Domains: []Domain{}}, nil
	}
	domains := s.domains
	if domains == nil {
		domains = []Domain{}
	}
	return &ListResponse{Domains: domains, Total: len(domains)}, nil
}

// GetDomainStats returns aggregate stats for a domain.
// Uses the in-memory cache when available, falls back to shard queries.
func (s *store) GetDomainStats(domain string) (*Domain, error) {
	norm := dcrawler.NormalizeDomain(domain)

	// Try in-memory cache first.
	s.mu.RLock()
	for i := range s.domains {
		if s.domains[i].Domain == norm {
			d := s.domains[i]
			s.mu.RUnlock()
			return &d, nil
		}
	}
	s.mu.RUnlock()

	// Fallback: query shards directly.
	domainDir := filepath.Join(s.dataDir, norm)
	resultDir := filepath.Join(domainDir, "results")

	shards, _ := filepath.Glob(filepath.Join(resultDir, "results_*.duckdb"))
	if len(shards) == 0 {
		return nil, fmt.Errorf("no data for %s", domain)
	}

	d := computeDomainStats(domainDir, norm)
	return &d, nil
}

// GetPages returns paginated pages for a domain.
// statusFilter: "2xx", "3xx", "4xx", "5xx", "error", or "" for all.
func (s *store) GetPages(domain string, page, pageSize int, q, sortBy, statusFilter string) (*PagesResponse, error) {
	domainDir := filepath.Join(s.dataDir, dcrawler.NormalizeDomain(domain))
	resultDir := filepath.Join(domainDir, "results")

	shards, _ := filepath.Glob(filepath.Join(resultDir, "results_*.duckdb"))
	if len(shards) == 0 {
		return nil, fmt.Errorf("no result shards for %s", domain)
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var unions []string
	for i, shard := range shards {
		alias := fmt.Sprintf("s%d", i)
		if _, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS %s (READ_ONLY)", shard, alias)); err != nil {
			continue
		}
		mdExpr := "0"
		if shardHasColumn(db, alias, "markdown") {
			mdExpr = "COALESCE(length(markdown), 0)"
		}
		unions = append(unions, fmt.Sprintf(
			"SELECT url, url_hash, status_code, COALESCE(content_type, '') AS content_type, COALESCE(content_length, 0) AS content_length, COALESCE(title, '') AS title, COALESCE(description, '') AS description, COALESCE(language, '') AS language, COALESCE(fetch_time_ms, 0) AS fetch_time_ms, crawled_at, COALESCE(error, '') AS error, %s AS md_size FROM %s.pages",
			mdExpr, alias,
		))
	}
	if len(unions) == 0 {
		return nil, fmt.Errorf("no readable shards for %s", domain)
	}

	view := joinUnions(unions)
	statusCond := statusCondition(statusFilter)

	var whereParts []string
	var args []any
	if statusCond != "" {
		whereParts = append(whereParts, statusCond)
	}
	if q != "" {
		whereParts = append(whereParts, fmt.Sprintf("(url ILIKE '%%' || $%d || '%%' OR title ILIKE '%%' || $%d || '%%')", len(args)+1, len(args)+1))
		args = append(args, q)
	}
	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = " WHERE " + strings.Join(whereParts, " AND ")
	}

	var total int64
	db.QueryRow(fmt.Sprintf("SELECT count(*) FROM (%s)%s", view, whereClause), args...).Scan(&total)

	orderCol, orderDir := sortColumn(sortBy)
	offset := (page - 1) * pageSize

	dataRows, err := db.Query(fmt.Sprintf(
		"SELECT url, url_hash, status_code, content_type, content_length, title, description, language, fetch_time_ms, crawled_at::VARCHAR, error, md_size FROM (%s)%s ORDER BY %s %s LIMIT %d OFFSET %d",
		view, whereClause, orderCol, orderDir, pageSize, offset), args...)
	if err != nil {
		return nil, fmt.Errorf("query pages: %w", err)
	}
	defer dataRows.Close()

	var pages []Page
	for dataRows.Next() {
		var p Page
		if err := dataRows.Scan(&p.URL, &p.URLHash, &p.StatusCode, &p.ContentType, &p.ContentLength,
			&p.Title, &p.Description, &p.Language, &p.FetchTimeMs, &p.CrawledAt, &p.Error, &p.MdSize); err != nil {
			continue
		}
		pages = append(pages, p)
	}
	if pages == nil {
		pages = []Page{}
	}

	return &PagesResponse{
		Domain:   domain,
		Pages:    pages,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetDomainSummary returns per-status-group counts and size averages for a domain.
// Queries shards directly (not cached) — call only for domain detail views.
func (s *store) GetDomainSummary(domain string) *DomainSummary {
	domainDir := filepath.Join(s.dataDir, dcrawler.NormalizeDomain(domain))
	resultDir := filepath.Join(domainDir, "results")
	shards, _ := filepath.Glob(filepath.Join(resultDir, "results_*.duckdb"))
	if len(shards) == 0 {
		return nil
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil
	}
	defer db.Close()

	var unions []string
	hasMD := false
	for i, shard := range shards {
		alias := fmt.Sprintf("s%d", i)
		if _, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS %s (READ_ONLY)", shard, alias)); err != nil {
			// Retry without read_only for active crawl shards.
			if _, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS %s", shard, alias)); err != nil {
				continue
			}
		}
		if shardHasColumn(db, alias, "markdown") {
			hasMD = true
		}
		unions = append(unions, fmt.Sprintf(
			"SELECT status_code, COALESCE(content_length, 0) AS content_length, COALESCE(error, '') AS error FROM %s.pages", alias))
	}
	if len(unions) == 0 {
		return nil
	}

	view := joinUnions(unions)
	var sum DomainSummary
	db.QueryRow(fmt.Sprintf(`SELECT
		count(*) FILTER (WHERE status_code >= 200 AND status_code < 300),
		count(*) FILTER (WHERE status_code >= 300 AND status_code < 400),
		count(*) FILTER (WHERE status_code >= 400 AND status_code < 500),
		count(*) FILTER (WHERE status_code >= 500 AND status_code < 600),
		count(*) FILTER (WHERE status_code = 0 OR error != ''),
		COALESCE(AVG(content_length) FILTER (WHERE content_length > 0), 0)::BIGINT
	FROM (%s)`, view)).Scan(&sum.Status2xx, &sum.Status3xx, &sum.Status4xx, &sum.Status5xx, &sum.StatusErr, &sum.AvgSize)

	// Query avg markdown size if any shard has the column.
	if hasMD {
		var mdUnions []string
		for i := range shards {
			alias := fmt.Sprintf("s%d", i)
			if shardHasColumn(db, alias, "markdown") {
				mdUnions = append(mdUnions, fmt.Sprintf(
					"SELECT length(markdown) AS md_len FROM %s.pages WHERE markdown IS NOT NULL AND length(markdown) > 0", alias))
			}
		}
		if len(mdUnions) > 0 {
			mdView := joinUnions(mdUnions)
			db.QueryRow(fmt.Sprintf("SELECT COALESCE(AVG(md_len), 0)::BIGINT FROM (%s)", mdView)).Scan(&sum.AvgMdSize)
		}
	}

	return &sum
}

// ── private helpers ───────────────────────────────────────────────────────────

// shardHasColumn checks if a column exists in the pages table of an attached shard.
func shardHasColumn(db *sql.DB, alias, column string) bool {
	var cnt int
	if db.QueryRow(
		"SELECT count(*) FROM duckdb_columns() WHERE database_name=$1 AND table_name='pages' AND column_name=$2",
		alias, column).Scan(&cnt) == nil {
		return cnt > 0
	}
	return false
}

func joinUnions(parts []string) string {
	s := ""
	for i, p := range parts {
		if i > 0 {
			s += " UNION ALL "
		}
		s += p
	}
	return s
}

func sortColumn(sortBy string) (col, dir string) {
	switch sortBy {
	case "url":
		return "url", "ASC"
	case "status":
		return "status_code", "ASC"
	case "fetch_time":
		return "fetch_time_ms", "DESC"
	case "size":
		return "content_length", "DESC"
	default:
		return "crawled_at", "DESC"
	}
}

func statusCondition(filter string) string {
	switch filter {
	case "2xx":
		return "status_code >= 200 AND status_code < 300"
	case "3xx":
		return "status_code >= 300 AND status_code < 400"
	case "4xx":
		return "status_code >= 400 AND status_code < 500"
	case "5xx":
		return "status_code >= 500 AND status_code < 600"
	case "error":
		return "(status_code = 0 OR error != '')"
	default:
		return ""
	}
}
