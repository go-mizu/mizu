package cc

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	"github.com/go-mizu/mizu/blueprints/search/pkg/export"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	"golang.org/x/sync/errgroup"
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
	// Output to $HOME/data/export/ (shared across all sources).
	home, _ := os.UserHomeDir()
	exportDir := filepath.Join(home, "data", "export")

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

	// Pipeline: reader → channel → worker pool.
	workers := runtime.NumCPU()
	if workers < 4 {
		workers = 4
	}
	if workers > 32 {
		workers = 32
	}

	type ccExportItem struct {
		url  string
		html string
	}
	ch := make(chan ccExportItem, workers*4)

	var exported atomic.Int64
	start := time.Now()

	// Progress reporter.
	stopProgress := make(chan struct{})
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopProgress:
				return
			case <-ticker.C:
				n := exported.Load()
				if n == 0 {
					continue
				}
				elapsed := time.Since(start)
				emit(&CCExportState{
					Domain:        domain,
					Format:        t.format,
					PagesExported: n,
					PagesTotal:    total,
					PagesPerSec:   float64(n) / elapsed.Seconds(),
					Progress:      util.PhaseProgress(n, total),
				})
			}
		}
	}()

	// Worker pool.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

	// Reader: scan rows, send to channel.
	var readErr error
	go func() {
		defer close(ch)
		for rows.Next() {
			if gctx.Err() != nil {
				return
			}
			var pageURL, body, contentType string
			if err := rows.Scan(&pageURL, &body, &contentType); err != nil {
				log.Printf("[cc-export] ERROR scan row: %v", err)
				continue
			}
			if !strings.Contains(strings.ToLower(contentType), "html") {
				continue
			}
			select {
			case ch <- ccExportItem{url: pageURL, html: body}:
			case <-gctx.Done():
				return
			}
		}
		if err := rows.Err(); err != nil {
			readErr = fmt.Errorf("iterate rows: %w", err)
		}
	}()

	// Consume items from channel with worker pool.
	for item := range ch {
		item := item
		g.Go(func() error {
			if _, err := writer.writePage(export.Page{URL: item.url, HTML: []byte(item.html)}); err != nil {
				log.Printf("[cc-export] ERROR write %s: %v", item.url, err)
			}
			exported.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return CCExportMetric{}, err
	}
	if readErr != nil {
		return CCExportMetric{}, readErr
	}

	close(stopProgress)
	<-progressDone

	if err := writer.writeIndex(); err != nil {
		log.Printf("[cc-export] ERROR write index: %v", err)
	}

	elapsed := time.Since(start)
	n := exported.Load()
	emit(&CCExportState{
		Domain:        domain,
		Format:        t.format,
		PagesExported: n,
		PagesTotal:    total,
		PagesPerSec:   float64(n) / elapsed.Seconds(),
		Progress:      1.0,
	})

	siteDir := filepath.Join(exportDir, t.format, domain)
	return CCExportMetric{
		Domain:  domain,
		Format:  t.format,
		Pages:   n,
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
