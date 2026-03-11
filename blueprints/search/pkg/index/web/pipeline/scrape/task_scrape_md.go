package scrape

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DataDog/zstd"
	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	crawler "github.com/go-mizu/mizu/blueprints/search/pkg/scrape"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
)

// Compile-time check.
var _ core.Task[ScrapeMarkdownState, ScrapeMarkdownMetric] = (*ScrapeMarkdownTask)(nil)

// ScrapeMarkdownState is emitted during DuckDB→markdown conversion.
type ScrapeMarkdownState struct {
	Domain        string  `json:"domain"`
	DocsProcessed int64   `json:"docs_processed"`
	DocsTotal     int64   `json:"docs_total"`
	DocsPerSec    float64 `json:"docs_per_sec"`
	Progress      float64 `json:"progress"`
}

// ScrapeMarkdownMetric is the final result of DuckDB→markdown conversion.
type ScrapeMarkdownMetric struct {
	Domain  string
	Docs    int64
	Elapsed time.Duration
}

// ScrapeMarkdownTask converts HTML pages from dcrawler DuckDB shards to markdown files.
type ScrapeMarkdownTask struct {
	domain  string
	dataDir string
}

// NewScrapeMarkdownTask creates a task that converts scraped HTML to markdown.
func NewScrapeMarkdownTask(domain, dataDir string) *ScrapeMarkdownTask {
	return &ScrapeMarkdownTask{domain: domain, dataDir: dataDir}
}

// Run reads HTML from DuckDB shards, converts to markdown, writes files.
func (t *ScrapeMarkdownTask) Run(ctx context.Context, emit func(*ScrapeMarkdownState)) (ScrapeMarkdownMetric, error) {
	domainDir := filepath.Join(t.dataDir, crawler.NormalizeDomain(t.domain))
	resultDir := filepath.Join(domainDir, "results")
	mdDir := filepath.Join(domainDir, "markdown")

	if err := os.MkdirAll(mdDir, 0o755); err != nil {
		return ScrapeMarkdownMetric{}, fmt.Errorf("create markdown dir: %w", err)
	}

	shards, err := filepath.Glob(filepath.Join(resultDir, "results_*.duckdb"))
	if err != nil || len(shards) == 0 {
		return ScrapeMarkdownMetric{}, fmt.Errorf("no result shards in %s", resultDir)
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return ScrapeMarkdownMetric{}, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	var unions []string
	for i, shard := range shards {
		alias := fmt.Sprintf("s%d", i)
		_, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS %s (READ_ONLY)", shard, alias))
		if err != nil {
			log.Printf("[scrape] ERROR attach shard %s: %v", shard, err)
			continue
		}
		unions = append(unions, fmt.Sprintf("SELECT url, url_hash, body, content_type FROM %s.pages WHERE status_code >= 200 AND status_code < 400 AND body IS NOT NULL AND octet_length(body) > 0", alias))
	}
	if len(unions) == 0 {
		return ScrapeMarkdownMetric{}, fmt.Errorf("no readable shards")
	}

	var total int64
	for i := range shards {
		alias := fmt.Sprintf("s%d", i)
		var n int64
		row := db.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s.pages WHERE status_code >= 200 AND status_code < 400 AND body IS NOT NULL AND octet_length(body) > 0", alias))
		if row.Scan(&n) == nil {
			total += n
		}
	}

	query := strings.Join(unions, " UNION ALL ")
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return ScrapeMarkdownMetric{}, fmt.Errorf("query pages: %w", err)
	}
	defer rows.Close()

	start := time.Now()
	var processed int64

	for rows.Next() {
		if ctx.Err() != nil {
			return ScrapeMarkdownMetric{}, ctx.Err()
		}

		var pageURL string
		var urlHash int64
		var body []byte
		var contentType string
		if err := rows.Scan(&pageURL, &urlHash, &body, &contentType); err != nil {
			log.Printf("[scrape] ERROR scan row: %v", err)
			continue
		}

		if !strings.Contains(strings.ToLower(contentType), "html") {
			continue
		}

		mdPath := filepath.Join(mdDir, fmt.Sprintf("%d.md", uint64(urlHash)))
		if util.FileExists(mdPath) {
			processed++
			continue
		}

		html, err := zstd.Decompress(nil, body)
		if err != nil {
			log.Printf("[scrape] ERROR decompress %s: %v", pageURL, err)
			continue
		}

		result := markdown.ConvertFast(html, pageURL)
		if result.Markdown == "" {
			processed++
			continue
		}

		content := fmt.Sprintf("---\nurl: %s\ntitle: %s\n---\n\n%s\n",
			pageURL, escapeFrontmatter(result.Title), result.Markdown)
		if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
			log.Printf("[scrape] ERROR write %s: %v", mdPath, err)
			continue
		}

		processed++

		if processed%50 == 0 {
			elapsed := time.Since(start)
			emit(&ScrapeMarkdownState{
				Domain:        t.domain,
				DocsProcessed: processed,
				DocsTotal:     total,
				DocsPerSec:    float64(processed) / elapsed.Seconds(),
				Progress:      util.PhaseProgress(processed, total),
			})
		}
	}

	elapsed := time.Since(start)
	emit(&ScrapeMarkdownState{
		Domain:        t.domain,
		DocsProcessed: processed,
		DocsTotal:     total,
		DocsPerSec:    float64(processed) / elapsed.Seconds(),
		Progress:      1.0,
	})

	return ScrapeMarkdownMetric{
		Domain:  t.domain,
		Docs:    processed,
		Elapsed: elapsed,
	}, nil
}

func escapeFrontmatter(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if strings.ContainsAny(s, ":\"'{}[]|>&*!%#`@,") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}
