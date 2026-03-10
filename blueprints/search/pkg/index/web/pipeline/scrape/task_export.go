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
	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
	"github.com/go-mizu/mizu/blueprints/search/pkg/export"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
)

// Compile-time check.
var _ core.Task[ExportState, ExportMetric] = (*ExportTask)(nil)

// ExportState is emitted during site export.
type ExportState struct {
	Domain        string  `json:"domain"`
	Format        string  `json:"format"`
	PagesExported int64   `json:"pages_exported"`
	PagesTotal    int64   `json:"pages_total"`
	PagesPerSec   float64 `json:"pages_per_sec"`
	Progress      float64 `json:"progress"`
}

// ExportMetric is the final result of a site export.
type ExportMetric struct {
	Domain  string
	Format  string
	Pages   int64
	OutDir  string
	Elapsed time.Duration
}

// ExportTask exports a scraped domain from DuckDB shards to a browsable offline mirror.
type ExportTask struct {
	domain  string
	dataDir string
	format  string // "html", "raw", or "markdown"
}

// NewExportTask creates a task that exports a scraped domain.
func NewExportTask(domain, dataDir, format string) *ExportTask {
	if format == "" {
		format = "html"
	}
	return &ExportTask{domain: domain, dataDir: dataDir, format: format}
}

// Run reads HTML from DuckDB shards, rewrites links, writes browsable site.
func (t *ExportTask) Run(ctx context.Context, emit func(*ExportState)) (ExportMetric, error) {
	norm := dcrawler.NormalizeDomain(t.domain)
	resultDir := filepath.Join(t.dataDir, norm, "results")
	// Output to $HOME/data/export/ (shared across all sources).
	home, _ := os.UserHomeDir()
	outDir := filepath.Join(home, "data", "export")

	shards, err := filepath.Glob(filepath.Join(resultDir, "results_*.duckdb"))
	if err != nil || len(shards) == 0 {
		return ExportMetric{}, fmt.Errorf("no result shards in %s", resultDir)
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return ExportMetric{}, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	var unions []string
	for i, shard := range shards {
		alias := fmt.Sprintf("s%d", i)
		_, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS %s (READ_ONLY)", shard, alias))
		if err != nil {
			log.Printf("[export] ERROR attach shard %s: %v", shard, err)
			continue
		}
		unions = append(unions, fmt.Sprintf(
			"SELECT url, body, content_type FROM %s.pages WHERE status_code >= 200 AND status_code < 400 AND body IS NOT NULL AND octet_length(body) > 0",
			alias))
	}
	if len(unions) == 0 {
		return ExportMetric{}, fmt.Errorf("no readable shards")
	}

	// Count total HTML pages
	var total int64
	for i := range shards {
		alias := fmt.Sprintf("s%d", i)
		var n int64
		row := db.QueryRow(fmt.Sprintf(
			"SELECT count(*) FROM %s.pages WHERE status_code >= 200 AND status_code < 400 AND body IS NOT NULL AND octet_length(body) > 0 AND lower(content_type) LIKE '%%html%%'",
			alias))
		if row.Scan(&n) == nil {
			total += n
		}
	}

	// Create the appropriate exporter
	cfg := export.Config{Domain: norm, OutDir: outDir, Format: t.format}
	writer, err := newPageWriter(cfg)
	if err != nil {
		return ExportMetric{}, fmt.Errorf("create exporter: %w", err)
	}

	query := strings.Join(unions, " UNION ALL ")
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return ExportMetric{}, fmt.Errorf("query pages: %w", err)
	}
	defer rows.Close()

	start := time.Now()
	var exported int64

	for rows.Next() {
		if ctx.Err() != nil {
			return ExportMetric{}, ctx.Err()
		}

		var pageURL string
		var body []byte
		var contentType string
		if err := rows.Scan(&pageURL, &body, &contentType); err != nil {
			log.Printf("[export] ERROR scan row: %v", err)
			continue
		}

		if !strings.Contains(strings.ToLower(contentType), "html") {
			continue
		}

		html, err := zstd.Decompress(nil, body)
		if err != nil {
			log.Printf("[export] ERROR decompress %s: %v", pageURL, err)
			continue
		}

		if _, err := writer.writePage(export.Page{URL: pageURL, HTML: html}); err != nil {
			log.Printf("[export] ERROR write %s: %v", pageURL, err)
			continue
		}

		exported++

		if exported%20 == 0 {
			elapsed := time.Since(start)
			emit(&ExportState{
				Domain:        norm,
				Format:        t.format,
				PagesExported: exported,
				PagesTotal:    total,
				PagesPerSec:   float64(exported) / elapsed.Seconds(),
				Progress:      util.PhaseProgress(exported, total),
			})
		}
	}

	if err := rows.Err(); err != nil {
		return ExportMetric{}, fmt.Errorf("iterate rows: %w", err)
	}

	if err := writer.writeIndex(); err != nil {
		log.Printf("[export] ERROR write index: %v", err)
	}

	elapsed := time.Since(start)
	emit(&ExportState{
		Domain:        norm,
		Format:        t.format,
		PagesExported: exported,
		PagesTotal:    total,
		PagesPerSec:   float64(exported) / elapsed.Seconds(),
		Progress:      1.0,
	})

	siteDir := filepath.Join(outDir, t.format, norm)
	return ExportMetric{
		Domain:  norm,
		Format:  t.format,
		Pages:   exported,
		OutDir:  siteDir,
		Elapsed: elapsed,
	}, nil
}

// RemoveExport removes an existing export directory for a domain.
func RemoveExport(dataDir, domain string) error {
	norm := dcrawler.NormalizeDomain(domain)
	exportDir := filepath.Join(dataDir, norm, "export")
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(exportDir)
}

// pageWriter abstracts over HTML and Markdown exporters.
type pageWriter struct {
	htmlExp *export.Exporter
	mdExp   *export.MarkdownExporter
}

func newPageWriter(cfg export.Config) (*pageWriter, error) {
	if cfg.Format == "markdown" {
		mdExp, err := export.NewMarkdownExporter(cfg, func(html []byte, pageURL string) (string, string) {
			r := markdown.ConvertFast(html, pageURL)
			return r.Title, r.Markdown
		})
		if err != nil {
			return nil, err
		}
		return &pageWriter{mdExp: mdExp}, nil
	}
	htmlExp, err := export.New(cfg)
	if err != nil {
		return nil, err
	}
	return &pageWriter{htmlExp: htmlExp}, nil
}

func (w *pageWriter) writePage(p export.Page) (string, error) {
	if w.mdExp != nil {
		return w.mdExp.WritePage(p)
	}
	return w.htmlExp.WritePage(p)
}

func (w *pageWriter) writeIndex() error {
	if w.mdExp != nil {
		return w.mdExp.WriteIndex()
	}
	return w.htmlExp.WriteIndex()
}
