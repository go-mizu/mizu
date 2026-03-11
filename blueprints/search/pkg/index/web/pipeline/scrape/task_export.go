package scrape

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

	"github.com/DataDog/zstd"
	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	crawler "github.com/go-mizu/mizu/blueprints/search/pkg/scrape"
	"github.com/go-mizu/mizu/blueprints/search/pkg/export"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	"golang.org/x/sync/errgroup"
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
	Phase         string  `json:"phase,omitempty"`          // "pages" or "assets"
	AssetsTotal   int     `json:"assets_total,omitempty"`   // total asset URLs discovered
	AssetsDown    int64   `json:"assets_down,omitempty"`    // downloaded so far
	AssetsFailed  int64   `json:"assets_failed,omitempty"`  // failed downloads
	AssetsBytes   int64   `json:"assets_bytes,omitempty"`   // total bytes downloaded
}

// ExportMetric is the final result of a site export.
type ExportMetric struct {
	Domain  string
	Format  string
	Pages   int64
	Assets  int64
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

// exportItem is sent from the reader goroutine to the worker pool.
type exportItem struct {
	url  string
	body []byte // zstd-compressed; workers decompress in parallel
}

// Run reads HTML from DuckDB shards, decompresses and converts in parallel.
func (t *ExportTask) Run(ctx context.Context, emit func(*ExportState)) (ExportMetric, error) {
	norm := crawler.NormalizeDomain(t.domain)
	resultDir := filepath.Join(t.dataDir, norm, "results")
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
		if _, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS %s (READ_ONLY)", shard, alias)); err != nil {
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

	// Count total HTML pages.
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

	// Create thread-safe exporter.
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

	// Pipeline: reader → channel → worker pool.
	workers := runtime.NumCPU()
	if workers < 4 {
		workers = 4
	}
	if workers > 32 {
		workers = 32
	}
	ch := make(chan exportItem, workers*4)

	var exported atomic.Int64
	start := time.Now()

	// Progress reporter.
	stopProgress := make(chan struct{})
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(1 * time.Second)
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
				emit(&ExportState{
					Domain:        norm,
					Format:        t.format,
					PagesExported: n,
					PagesTotal:    total,
					PagesPerSec:   float64(n) / elapsed.Seconds(),
					Progress:      util.PhaseProgress(n, total),
				})
			}
		}
	}()

	// Worker pool: decompress + convert + write in parallel.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

	// Reader: scan rows, send compressed body to channel.
	var readErr error
	go func() {
		defer close(ch)
		for rows.Next() {
			if gctx.Err() != nil {
				return
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
			// Copy body — DuckDB row buffer is reused.
			raw := make([]byte, len(body))
			copy(raw, body)
			select {
			case ch <- exportItem{url: pageURL, body: raw}:
			case <-gctx.Done():
				return
			}
		}
		if err := rows.Err(); err != nil {
			readErr = fmt.Errorf("iterate rows: %w", err)
		}
	}()

	// Consume items from channel with worker pool.
	// Each worker decompresses + converts + writes in parallel.
	for item := range ch {
		item := item
		g.Go(func() error {
			html, err := zstd.Decompress(nil, item.body)
			if err != nil {
				log.Printf("[export] ERROR decompress %s: %v", item.url, err)
				return nil
			}
			if _, err := writer.writePage(export.Page{URL: item.url, HTML: html}); err != nil {
				log.Printf("[export] ERROR write %s: %v", item.url, err)
			}
			exported.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return ExportMetric{}, err
	}
	if readErr != nil {
		return ExportMetric{}, readErr
	}

	close(stopProgress)
	<-progressDone

	if err := writer.writeIndex(); err != nil {
		log.Printf("[export] ERROR write index: %v", err)
	}

	n := exported.Load()
	emit(&ExportState{
		Domain:        norm,
		Format:        t.format,
		Phase:         "pages",
		PagesExported: n,
		PagesTotal:    total,
		PagesPerSec:   float64(n) / time.Since(start).Seconds(),
		Progress:      1.0,
	})

	// Download assets (CSS, images) for offline viewing.
	var assetCount int64
	if ac := writer.assets(); ac != nil && ac.Count() > 0 {
		sd := writer.siteDir()
		assetWorkers := workers
		if assetWorkers > 16 {
			assetWorkers = 16
		}
		if err := export.DownloadAssets(ctx, ac, sd, assetWorkers, func(stats export.AssetDownloadStats) {
			emit(&ExportState{
				Domain:        norm,
				Format:        t.format,
				Phase:         "assets",
				PagesExported: n,
				PagesTotal:    total,
				AssetsTotal:   stats.Total,
				AssetsDown:    stats.Downloaded,
				AssetsFailed:  stats.Failed,
				AssetsBytes:   stats.Bytes,
			})
		}); err != nil {
			log.Printf("[export] ERROR download assets: %v", err)
		}

		// Rewrite CSS files to use local paths for nested url() references.
		for absURL, localPath := range ac.URLs() {
			if isCSS := strings.HasSuffix(strings.ToLower(localPath), ".css"); isCSS {
				export.RewriteDownloadedCSS(filepath.Join(sd, localPath), absURL, sd, ac)
			}
		}
		assetCount = int64(ac.Count())
	}

	elapsed := time.Since(start)
	siteDir := filepath.Join(outDir, t.format, norm)
	return ExportMetric{
		Domain:  norm,
		Format:  t.format,
		Pages:   n,
		Assets:  assetCount,
		OutDir:  siteDir,
		Elapsed: elapsed,
	}, nil
}

// RemoveExport removes an existing export directory for a domain.
func RemoveExport(dataDir, domain string) error {
	norm := crawler.NormalizeDomain(domain)
	exportDir := filepath.Join(dataDir, norm, "export")
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(exportDir)
}

// pageWriter abstracts over HTML and Markdown exporters. Thread-safe.
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

func (w *pageWriter) assets() *export.AssetCollector {
	if w.htmlExp != nil {
		return w.htmlExp.Assets
	}
	return nil
}

func (w *pageWriter) siteDir() string {
	if w.htmlExp != nil {
		return w.htmlExp.SiteDir()
	}
	return ""
}
