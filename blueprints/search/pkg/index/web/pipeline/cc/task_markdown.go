package cc

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
)

// Compile-time check.
var _ core.Task[MarkdownState, MarkdownMetric] = (*MarkdownTask)(nil)

const markdownConcurrency = 2

// MarkdownTask converts downloaded WARC files to .md.warc.gz (markdown WARC).
type MarkdownTask struct {
	crawlID  string
	crawlDir string
	paths    []string
	selected []int
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
	return &MarkdownTask{crawlID: crawlID, crawlDir: crawlDir, paths: paths, selected: selected}
}

func (t *MarkdownTask) Run(ctx context.Context, emit func(*MarkdownState)) (MarkdownMetric, error) {
	start := time.Now()
	total := len(t.selected)
	if total == 0 {
		return MarkdownMetric{Elapsed: time.Since(start)}, nil
	}

	warcDir := filepath.Join(t.crawlDir, "warc")
	warcMdDir := filepath.Join(t.crawlDir, "warc_md")
	var totalDocs atomic.Int64

	workersPerShard := runtime.NumCPU() * 2

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(markdownConcurrency)

	for i, idx := range t.selected {
		i, idx := i, idx
		g.Go(func() error {
			warcPath := t.paths[idx]
			warcIdx := util.WARCFileIndex(warcPath, idx)
			localPath := filepath.Join(warcDir, filepath.Base(warcPath))
			if !util.FileExists(localPath) {
				return fmt.Errorf("warc file not found: %s (run download first)", localPath)
			}

			outputPath := filepath.Join(warcMdDir, warcIdx+".md.warc.gz")

			progressCb := func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, _ float64) {
				totalDocs.Store(done)
				phase := "extract"
				if total > 0 && done >= total {
					phase = "convert"
				}
				emitMarkdownProgress(emit, i, len(t.selected), warcIdx, phase,
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

func emitMarkdownProgress(emit func(*MarkdownState), fileIdx, fileTotal int, warcIdx, phase string,
	done, total, errors, readBytes, writeBytes int64, elapsed time.Duration) {
	if emit == nil {
		return
	}
	localPct := util.PhaseProgress(done, total)
	var overall float64
	switch phase {
	case "extract":
		overall = util.FileProgress(fileIdx, fileTotal, 0.5*localPct)
	case "convert":
		overall = util.FileProgress(fileIdx, fileTotal, 0.5+0.5*localPct)
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
		ReadRate:      util.MBPerSec(readBytes, elapsed),
		WriteRate:     util.MBPerSec(writeBytes, elapsed),
		Progress:      overall,
	})
}
