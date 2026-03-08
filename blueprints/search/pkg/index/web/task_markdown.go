package web

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
)

// markdownConcurrency is the number of WARC files converted in parallel.
const markdownConcurrency = 2

// MarkdownTask converts downloaded WARC files to .md.warc.gz (markdown WARC).
// Uses warcmd.RunPack which goes directly from .warc.gz → .md.warc.gz,
// preserving WARC-Target-URI, WARC-Date, and other headers.
// It is a self-contained core.Task with no dependency on Manager.
type MarkdownTask struct {
	CrawlID  string   `json:"crawl_id"`
	CrawlDir string   `json:"crawl_dir"`
	Paths    []string `json:"paths"`    // manifest paths
	Selected []int    `json:"selected"` // indices into Paths
}

// MarkdownState is emitted during markdown conversion with per-phase detail.
type MarkdownState struct {
	FileIndex     int     `json:"file_index"`
	FileTotal     int     `json:"file_total"`
	WARCIndex     string  `json:"warc_index"`
	Phase         string  `json:"phase"` // "extract" or "convert"
	DocsProcessed int64   `json:"docs_processed"`
	DocsTotal     int64   `json:"docs_total"`
	DocsErrors    int64   `json:"docs_errors"`
	ReadBytes     int64   `json:"read_bytes"`
	WriteBytes    int64   `json:"write_bytes"`
	ReadRate      float64 `json:"read_rate_mbps"`
	WriteRate     float64 `json:"write_rate_mbps"`
	Progress      float64 `json:"progress"`
}

// MarkdownMetric is the final result after all files are converted.
type MarkdownMetric struct {
	Files   int           `json:"files"`
	Docs    int64         `json:"docs"`
	Elapsed time.Duration `json:"elapsed_ns"`
}

// NewMarkdownTask creates a markdown conversion task.
func NewMarkdownTask(crawlID, crawlDir string, paths []string, selected []int) *MarkdownTask {
	return &MarkdownTask{CrawlID: crawlID, CrawlDir: crawlDir, Paths: paths, Selected: selected}
}

func (t *MarkdownTask) Run(ctx context.Context, emit func(*MarkdownState)) (MarkdownMetric, error) {
	start := time.Now()
	total := len(t.Selected)
	if total == 0 {
		return MarkdownMetric{Elapsed: time.Since(start)}, nil
	}

	warcDir := filepath.Join(t.CrawlDir, "warc")
	warcMdDir := filepath.Join(t.CrawlDir, "warc_md")
	var totalDocs atomic.Int64

	// Oversubscribe: NumCPU*2 workers per shard (goroutines are cheap;
	// oversubscription hides GC pauses and keeps all cores busy).
	workersPerShard := runtime.NumCPU() * 2

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(markdownConcurrency)

	for i, idx := range t.Selected {
		i, idx := i, idx
		g.Go(func() error {
			warcPath := t.Paths[idx]
			warcIdx := warcFileIndex(warcPath, idx)
			localPath := filepath.Join(warcDir, filepath.Base(warcPath))
			if !fileExists(localPath) {
				return fmt.Errorf("warc file not found: %s (run download first)", localPath)
			}

			outputPath := filepath.Join(warcMdDir, warcIdx+".md.warc.gz")

			progressCb := func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, _ float64) {
				totalDocs.Store(done)
				phase := "extract"
				if total > 0 && done >= total {
					phase = "convert"
				}
				emitMarkdownProgress(emit, i, len(t.Selected), warcIdx, phase,
					done, total, errors, readBytes, writeBytes, elapsed)
			}

			cfg := warcmd.PackConfig{
				InputFiles:  []string{localPath},
				OutputPath:  outputPath,
				Workers:     workersPerShard,
				Force:       true,
				StatusCode:  200,
				MIMEFilter:  "text/html",
				MaxBodySize: 512 * 1024,
			}

			stats, err := warcmd.RunPack(gctx, cfg, progressCb)
			if err != nil {
				return fmt.Errorf("markdown %s: %w", warcIdx, err)
			}
			if stats != nil {
				totalDocs.Store(stats.OutputRecords)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return MarkdownMetric{}, err
	}

	return MarkdownMetric{
		Files:   total,
		Docs:    totalDocs.Load(),
		Elapsed: time.Since(start),
	}, nil
}

// emitMarkdownProgress emits a detailed markdown conversion state.
func emitMarkdownProgress(emit func(*MarkdownState), fileIdx, fileTotal int, warcIdx, phase string,
	done, total, errors, readBytes, writeBytes int64, elapsed time.Duration) {
	if emit == nil {
		return
	}
	localPct := phaseProgress(done, total)
	var overall float64
	switch phase {
	case "extract":
		overall = fileProgress(fileIdx, fileTotal, 0.5*localPct)
	case "convert":
		overall = fileProgress(fileIdx, fileTotal, 0.5+0.5*localPct)
	}
	emit(&MarkdownState{
		FileIndex:     fileIdx,
		FileTotal:     fileTotal,
		WARCIndex:     warcIdx,
		Phase:         phase,
		DocsProcessed: done,
		DocsTotal:     total,
		DocsErrors:    errors,
		ReadBytes:     readBytes,
		WriteBytes:    writeBytes,
		ReadRate:      mbPerSec(readBytes, elapsed),
		WriteRate:     mbPerSec(writeBytes, elapsed),
		Progress:      overall,
	})
}
