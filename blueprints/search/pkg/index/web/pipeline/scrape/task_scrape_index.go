package scrape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	indexpack "github.com/go-mizu/mizu/blueprints/search/pkg/index/pack"
)

// Compile-time check.
var _ core.Task[ScrapeIndexState, ScrapeIndexMetric] = (*ScrapeIndexTask)(nil)

// ScrapeIndexState is emitted during indexing progress.
type ScrapeIndexState struct {
	Domain      string  `json:"domain"`
	DocsIndexed int64   `json:"docs_indexed"`
	DocsTotal   int64   `json:"docs_total"`
	DocsPerSec  float64 `json:"docs_per_sec"`
	Progress    float64 `json:"progress"`
}

// ScrapeIndexMetric is the final result.
type ScrapeIndexMetric struct {
	Domain  string
	Docs    int64
	Elapsed time.Duration
}

// ScrapeIndexTask indexes markdown files into a dahlia FTS engine.
type ScrapeIndexTask struct {
	domain  string
	dataDir string
	engine  string // default "dahlia"
}

// NewScrapeIndexTask creates a new scrape index task.
func NewScrapeIndexTask(domain, dataDir, engine string) *ScrapeIndexTask {
	if engine == "" {
		engine = "dahlia"
	}
	return &ScrapeIndexTask{domain: domain, dataDir: dataDir, engine: engine}
}

func (t *ScrapeIndexTask) Run(ctx context.Context, emit func(*ScrapeIndexState)) (ScrapeIndexMetric, error) {
	start := time.Now()
	norm := dcrawler.NormalizeDomain(t.domain)
	mdDir := filepath.Join(t.dataDir, norm, "markdown")
	outputDir := filepath.Join(t.dataDir, norm, "fts", t.engine)

	// List all .md files.
	entries, err := os.ReadDir(mdDir)
	if err != nil {
		return ScrapeIndexMetric{}, fmt.Errorf("read markdown dir %s: %w", mdDir, err)
	}
	var mdFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, filepath.Join(mdDir, e.Name()))
		}
	}
	total := int64(len(mdFiles))
	if total == 0 {
		return ScrapeIndexMetric{Domain: norm, Docs: 0, Elapsed: time.Since(start)}, nil
	}

	// Open engine.
	eng, err := index.NewEngine(t.engine)
	if err != nil {
		return ScrapeIndexMetric{}, fmt.Errorf("create engine %s: %w", t.engine, err)
	}
	if err := eng.Open(ctx, outputDir); err != nil {
		return ScrapeIndexMetric{}, fmt.Errorf("open engine %s at %s: %w", t.engine, outputDir, err)
	}

	// Create document channel and producer goroutine.
	docCh := make(chan indexpack.Document, 256)
	go func() {
		defer close(docCh)
		for _, path := range mdFiles {
			if ctx.Err() != nil {
				return
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				continue
			}
			docID := strings.TrimSuffix(filepath.Base(path), ".md")
			select {
			case docCh <- indexpack.Document{DocID: docID, Text: data}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Index documents via pipeline.
	progress := func(done, docTotal int64, elapsed time.Duration) {
		if emit == nil {
			return
		}
		var pct float64
		if total > 0 {
			pct = float64(done) / float64(total)
		}
		var dps float64
		if elapsed > 0 && done > 0 {
			dps = float64(done) / elapsed.Seconds()
		}
		emit(&ScrapeIndexState{
			Domain:      norm,
			DocsIndexed: done,
			DocsTotal:   total,
			DocsPerSec:  dps,
			Progress:    pct,
		})
	}

	stats, err := indexpack.RunPipelineFromChannel(ctx, eng, docCh, total, 5000, progress)
	if err != nil {
		eng.Close()
		return ScrapeIndexMetric{}, err
	}

	// Finalize if supported.
	if fin, ok := eng.(index.Finalizer); ok {
		if err := fin.Finalize(ctx); err != nil {
			eng.Close()
			return ScrapeIndexMetric{}, fmt.Errorf("finalize: %w", err)
		}
	}
	eng.Close()

	return ScrapeIndexMetric{
		Domain:  norm,
		Docs:    stats.DocsIndexed.Load(),
		Elapsed: time.Since(start),
	}, nil
}
