package cc

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	"github.com/go-mizu/mizu/blueprints/search/pkg/export"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
)

// Compile-time check.
var _ core.Task[CCExportState, CCExportMetric] = (*CCExportTask)(nil)

// CCExportState is emitted during CC site export.
type CCExportState struct {
	Domain        string  `json:"domain"`
	Format        string  `json:"format"`
	PagesExported int64   `json:"pages_exported"`
	PagesTotal    int64   `json:"pages_total"`
	PagesPerSec   float64 `json:"pages_per_sec"`
	Progress      float64 `json:"progress"`
}

// CCExportMetric is the final result of a CC site export.
type CCExportMetric struct {
	Domain  string
	Format  string
	Pages   int64
	OutDir  string
	Elapsed time.Duration
}

// CCExportTask exports a domain from CC result DuckDB to a browsable offline mirror.
type CCExportTask struct {
	domain   string
	crawlDir string
	format   string // "html", "raw", or "markdown"
}

// NewCCExportTask creates a task that exports a CC domain.
func NewCCExportTask(domain, crawlDir, format string) *CCExportTask {
	if format == "" {
		format = "html"
	}
	return &CCExportTask{domain: domain, crawlDir: crawlDir, format: format}
}

// Run reads HTML from CC result DuckDB shards, rewrites links, writes browsable site.
func (t *CCExportTask) Run(ctx context.Context, emit func(*CCExportState)) (CCExportMetric, error) {
	recrawlDir := filepath.Join(t.crawlDir, "recrawl")
	// crawlDir = ~/data/common-crawl/CC-MAIN-2026-04/ → exportDir = ~/data/common-crawl/export/
	exportDir := filepath.Join(filepath.Dir(filepath.Dir(t.crawlDir)), "export")

	// Find result shards
	shards, err := filepath.Glob(filepath.Join(recrawlDir, "results_*.duckdb"))
	if err != nil || len(shards) == 0 {
		return CCExportMetric{}, fmt.Errorf("no result shards in %s", recrawlDir)
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return CCExportMetric{}, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	domain := strings.ToLower(strings.TrimPrefix(t.domain, "www."))

	var unions []string
	for i, shard := range shards {
		alias := fmt.Sprintf("s%d", i)
		_, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS %s (READ_ONLY)", shard, alias))
		if err != nil {
			log.Printf("[cc-export] ERROR attach shard %s: %v", shard, err)
			continue
		}
		unions = append(unions, fmt.Sprintf(
			"SELECT url, body, content_type FROM %s.results WHERE status_code >= 200 AND status_code < 400 AND body IS NOT NULL AND length(body) > 0 AND domain = '%s'",
			alias, escapeSingleQuote(domain)))
	}
	if len(unions) == 0 {
		return CCExportMetric{}, fmt.Errorf("no readable shards")
	}

	// Count total HTML pages for this domain
	var total int64
	for i := range shards {
		alias := fmt.Sprintf("s%d", i)
		var n int64
		row := db.QueryRow(fmt.Sprintf(
			"SELECT count(*) FROM %s.results WHERE status_code >= 200 AND status_code < 400 AND body IS NOT NULL AND length(body) > 0 AND domain = '%s' AND lower(content_type) LIKE '%%html%%'",
			alias, escapeSingleQuote(domain)))
		if row.Scan(&n) == nil {
			total += n
		}
	}

	if total == 0 {
		return CCExportMetric{}, fmt.Errorf("no pages found for domain %s", domain)
	}

	// Create the appropriate exporter
	cfg := export.Config{Domain: domain, OutDir: exportDir, Format: t.format}
	writer, err := newCCPageWriter(cfg)
	if err != nil {
		return CCExportMetric{}, fmt.Errorf("create exporter: %w", err)
	}

	query := strings.Join(unions, " UNION ALL ")
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return CCExportMetric{}, fmt.Errorf("query results: %w", err)
	}
	defer rows.Close()

	start := time.Now()
	var exported int64

	for rows.Next() {
		if ctx.Err() != nil {
			return CCExportMetric{}, ctx.Err()
		}

		var pageURL, body, contentType string
		if err := rows.Scan(&pageURL, &body, &contentType); err != nil {
			log.Printf("[cc-export] ERROR scan row: %v", err)
			continue
		}

		if !strings.Contains(strings.ToLower(contentType), "html") {
			continue
		}

		if _, err := writer.writePage(export.Page{URL: pageURL, HTML: []byte(body)}); err != nil {
			log.Printf("[cc-export] ERROR write %s: %v", pageURL, err)
			continue
		}

		exported++

		if exported%20 == 0 {
			elapsed := time.Since(start)
			emit(&CCExportState{
				Domain:        domain,
				Format:        t.format,
				PagesExported: exported,
				PagesTotal:    total,
				PagesPerSec:   float64(exported) / elapsed.Seconds(),
				Progress:      util.PhaseProgress(exported, total),
			})
		}
	}

	if err := rows.Err(); err != nil {
		return CCExportMetric{}, fmt.Errorf("iterate rows: %w", err)
	}

	if err := writer.writeIndex(); err != nil {
		log.Printf("[cc-export] ERROR write index: %v", err)
	}

	elapsed := time.Since(start)
	emit(&CCExportState{
		Domain:        domain,
		Format:        t.format,
		PagesExported: exported,
		PagesTotal:    total,
		PagesPerSec:   float64(exported) / elapsed.Seconds(),
		Progress:      1.0,
	})

	siteDir := filepath.Join(exportDir, t.format, domain)
	if t.format == "markdown" {
		siteDir = filepath.Join(exportDir, "markdown", domain)
	}
	return CCExportMetric{
		Domain:  domain,
		Format:  t.format,
		Pages:   exported,
		OutDir:  siteDir,
		Elapsed: elapsed,
	}, nil
}

// RemoveCCExport removes an existing CC export directory.
func RemoveCCExport(crawlDir string) error {
	exportDir := filepath.Join(filepath.Dir(filepath.Dir(crawlDir)), "export")
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(exportDir)
}

func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// ccPageWriter abstracts over HTML and Markdown exporters for CC data.
type ccPageWriter struct {
	htmlExp *export.Exporter
	mdExp   *export.MarkdownExporter
}

func newCCPageWriter(cfg export.Config) (*ccPageWriter, error) {
	if cfg.Format == "markdown" {
		mdExp, err := export.NewMarkdownExporter(cfg, func(html []byte, pageURL string) (string, string) {
			r := markdown.ConvertFast(html, pageURL)
			return r.Title, r.Markdown
		})
		if err != nil {
			return nil, err
		}
		return &ccPageWriter{mdExp: mdExp}, nil
	}
	htmlExp, err := export.New(cfg)
	if err != nil {
		return nil, err
	}
	return &ccPageWriter{htmlExp: htmlExp}, nil
}

func (w *ccPageWriter) writePage(p export.Page) (string, error) {
	if w.mdExp != nil {
		return w.mdExp.WritePage(p)
	}
	return w.htmlExp.WritePage(p)
}

func (w *ccPageWriter) writeIndex() error {
	if w.mdExp != nil {
		return w.mdExp.WriteIndex()
	}
	return w.htmlExp.WriteIndex()
}
