package web

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
)

// MarkdownTask converts downloaded WARC files to markdown.
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

	cfg := markdownConfig(t.CrawlID, filepath.Dir(t.CrawlDir))
	warcDir := filepath.Join(t.CrawlDir, "warc")
	var totalDocs int64

	for i, idx := range t.Selected {
		if ctx.Err() != nil {
			return MarkdownMetric{}, ctx.Err()
		}
		warcPath := t.Paths[idx]
		warcIdx := warcFileIndex(warcPath, idx)
		localPath := filepath.Join(warcDir, filepath.Base(warcPath))
		if !fileExists(localPath) {
			return MarkdownMetric{}, fmt.Errorf("warc file not found: %s (run download first)", localPath)
		}

		extractCb := func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, _ float64) {
			totalDocs = done
			emitMarkdownProgress(emit, i, len(t.Selected), warcIdx, "extract",
				done, total, errors, readBytes, writeBytes, elapsed)
		}
		convertCb := func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, _ float64) {
			totalDocs = done
			emitMarkdownProgress(emit, i, len(t.Selected), warcIdx, "convert",
				done, total, errors, readBytes, writeBytes, elapsed)
		}

		if _, err := warcmd.RunFilePipeline(ctx, cfg, warcIdx, []string{localPath}, extractCb, convertCb); err != nil {
			return MarkdownMetric{}, fmt.Errorf("markdown %s: %w", warcIdx, err)
		}
	}

	return MarkdownMetric{
		Files:   total,
		Docs:    totalDocs,
		Elapsed: time.Since(start),
	}, nil
}

// markdownConfig builds a warc_md pipeline configuration.
func markdownConfig(crawlID, dataDir string) warcmd.Config {
	cfg := warcmd.DefaultConfig(crawlID)
	cfg.DataDir = dataDir
	cfg.Workers = runtime.NumCPU()
	cfg.Force = false
	cfg.KeepTemp = false
	return cfg
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
