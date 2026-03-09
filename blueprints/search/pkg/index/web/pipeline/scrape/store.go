package scrape

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
)

// Store reads per-domain crawl metadata from dcrawler result databases.
type Store struct {
	dataDir string
}

// NewStore creates a Store that reads from the crawl data directory.
func NewStore(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

// Domain holds summary stats for a scraped domain.
type Domain struct {
	Domain    string    `json:"domain"`
	Pages     int64     `json:"pages"`
	Success   int64     `json:"success"`
	Failed    int64     `json:"failed"`
	Links     int64     `json:"links"`
	LastCrawl time.Time `json:"last_crawl"`
	HasMD     bool      `json:"has_markdown"`
	HasIndex  bool      `json:"has_index"`
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
	Domain    string   `json:"domain"`
	Stats     *Domain  `json:"stats,omitempty"`
	ActiveJob *JobInfo `json:"active_job,omitempty"`
	HasData   bool     `json:"has_data"`
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
	Mode      string `json:"mode"`
	MaxPages  int    `json:"max_pages"`
	MaxDepth  int    `json:"max_depth"`
	Workers   int    `json:"workers"`
	TimeoutS  int    `json:"timeout_s"`
	StoreBody bool   `json:"store_body"`
	Resume    bool   `json:"resume"`
}

// ParseStartParams decodes a JSON-encoded StartParams from JobConfig.Source.
func ParseStartParams(source string) StartParams {
	var p StartParams
	if source != "" {
		_ = json.Unmarshal([]byte(source), &p)
	}
	return p
}

// ListDomains discovers scraped domains by scanning the data directory.
func (s *Store) ListDomains() (*ListResponse, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &ListResponse{Domains: []Domain{}}, nil
		}
		return nil, err
	}

	var domains []Domain
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		domainDir := filepath.Join(s.dataDir, e.Name())
		resultDir := filepath.Join(domainDir, "results")

		shards, _ := filepath.Glob(filepath.Join(resultDir, "results_*.duckdb"))
		if len(shards) == 0 {
			continue
		}

		d := Domain{Domain: e.Name()}
		stats := quickStats(shards)
		d.Pages = stats.pages
		d.Success = stats.success
		d.Failed = stats.failed
		d.Links = stats.links
		d.LastCrawl = stats.lastCrawl

		if ents, _ := os.ReadDir(filepath.Join(domainDir, "markdown")); len(ents) > 0 {
			d.HasMD = true
		}
		if ents, _ := os.ReadDir(filepath.Join(domainDir, "fts")); len(ents) > 0 {
			d.HasIndex = true
		}

		domains = append(domains, d)
	}

	if domains == nil {
		domains = []Domain{}
	}
	return &ListResponse{Domains: domains, Total: len(domains)}, nil
}

// GetDomainStats returns aggregate stats for a domain.
func (s *Store) GetDomainStats(domain string) (*Domain, error) {
	domainDir := filepath.Join(s.dataDir, dcrawler.NormalizeDomain(domain))
	resultDir := filepath.Join(domainDir, "results")

	shards, _ := filepath.Glob(filepath.Join(resultDir, "results_*.duckdb"))
	if len(shards) == 0 {
		return nil, fmt.Errorf("no data for %s", domain)
	}

	stats := quickStats(shards)
	d := &Domain{
		Domain:    domain,
		Pages:     stats.pages,
		Success:   stats.success,
		Failed:    stats.failed,
		Links:     stats.links,
		LastCrawl: stats.lastCrawl,
	}
	if ents, _ := os.ReadDir(filepath.Join(domainDir, "markdown")); len(ents) > 0 {
		d.HasMD = true
	}
	if ents, _ := os.ReadDir(filepath.Join(domainDir, "fts")); len(ents) > 0 {
		d.HasIndex = true
	}
	return d, nil
}

// GetPages returns paginated pages for a domain.
func (s *Store) GetPages(domain string, page, pageSize int, q, sortBy string) (*PagesResponse, error) {
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
		unions = append(unions, fmt.Sprintf(
			"SELECT url, url_hash, status_code, COALESCE(content_type, '') AS content_type, COALESCE(content_length, 0) AS content_length, COALESCE(title, '') AS title, COALESCE(description, '') AS description, COALESCE(language, '') AS language, COALESCE(fetch_time_ms, 0) AS fetch_time_ms, crawled_at, COALESCE(error, '') AS error FROM %s.pages",
			alias,
		))
	}
	if len(unions) == 0 {
		return nil, fmt.Errorf("no readable shards for %s", domain)
	}

	view := joinUnions(unions)

	var total int64
	if q != "" {
		db.QueryRow(fmt.Sprintf("SELECT count(*) FROM (%s) WHERE url ILIKE '%%' || $1 || '%%' OR title ILIKE '%%' || $1 || '%%'", view), q).Scan(&total)
	} else {
		db.QueryRow(fmt.Sprintf("SELECT count(*) FROM (%s)", view)).Scan(&total)
	}

	orderCol, orderDir := sortColumn(sortBy)
	offset := (page - 1) * pageSize

	var dataRows *sql.Rows
	if q != "" {
		dataRows, err = db.Query(fmt.Sprintf(
			"SELECT url, url_hash, status_code, content_type, content_length, title, description, language, fetch_time_ms, crawled_at::VARCHAR, error FROM (%s) WHERE url ILIKE '%%' || $1 || '%%' OR title ILIKE '%%' || $1 || '%%' ORDER BY %s %s LIMIT %d OFFSET %d",
			view, orderCol, orderDir, pageSize, offset), q)
	} else {
		dataRows, err = db.Query(fmt.Sprintf(
			"SELECT url, url_hash, status_code, content_type, content_length, title, description, language, fetch_time_ms, crawled_at::VARCHAR, error FROM (%s) ORDER BY %s %s LIMIT %d OFFSET %d",
			view, orderCol, orderDir, pageSize, offset))
	}
	if err != nil {
		return nil, fmt.Errorf("query pages: %w", err)
	}
	defer dataRows.Close()

	var pages []Page
	for dataRows.Next() {
		var p Page
		if err := dataRows.Scan(&p.URL, &p.URLHash, &p.StatusCode, &p.ContentType, &p.ContentLength,
			&p.Title, &p.Description, &p.Language, &p.FetchTimeMs, &p.CrawledAt, &p.Error); err != nil {
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

// ── private helpers ───────────────────────────────────────────────────────────

type quickStatsResult struct {
	pages     int64
	success   int64
	failed    int64
	links     int64
	lastCrawl time.Time
}

func quickStats(shards []string) quickStatsResult {
	var r quickStatsResult
	for _, shard := range shards {
		db, err := sql.Open("duckdb", shard+"?access_mode=read_only")
		if err != nil {
			continue
		}
		var total, success, failed int64
		var lastCrawl sql.NullString
		row := db.QueryRow(`SELECT
			count(*),
			count(*) FILTER (WHERE status_code >= 200 AND status_code < 400),
			count(*) FILTER (WHERE status_code >= 400 OR error != ''),
			max(crawled_at)::VARCHAR
		FROM pages`)
		if row.Scan(&total, &success, &failed, &lastCrawl) == nil {
			r.pages += total
			r.success += success
			r.failed += failed
			if lastCrawl.Valid && len(lastCrawl.String) >= 19 {
				if t, err := time.Parse("2006-01-02 15:04:05", lastCrawl.String[:19]); err == nil {
					if t.After(r.lastCrawl) {
						r.lastCrawl = t
					}
				}
			}
		}
		var links int64
		if db.QueryRow(`SELECT count(*) FROM links`).Scan(&links) == nil {
			r.links += links
		}
		db.Close()
	}
	return r
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
