package markdown

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// ConvertPhaseConfig configures Phase 2: convert html/*.html → md/*.md.
type ConvertPhaseConfig struct {
	InputDir  string // html root (e.g. ~/data/common-crawl/html)
	OutputDir string // md output root (e.g. ~/data/common-crawl/md)
	IndexPath string // DuckDB index path (e.g. ~/data/common-crawl/md/index.duckdb)
	Workers   int    // parallel workers (0 = NumCPU)
	Force     bool   // re-convert existing files
	BatchSize int    // DuckDB write batch size (0 = 1000)
	Fast      bool   // use go-readability instead of trafilatura
}

// RunConvertPhase reads .html files, converts to markdown, writes .md files,
// and records metadata in DuckDB.
func RunConvertPhase(ctx context.Context, cfg ConvertPhaseConfig, progressFn PhaseProgressFunc) (*PhaseStats, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}

	// Collect .html files
	var files []string
	_ = filepath.WalkDir(cfg.InputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			files = append(files, path)
		}
		return nil
	})
	if len(files) == 0 {
		return &PhaseStats{}, nil
	}

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, err
	}

	// Open DuckDB index
	idx, err := OpenIndex(cfg.IndexPath, cfg.BatchSize)
	if err != nil {
		return nil, err
	}
	defer idx.Close()

	var stats PhaseStats
	totalFiles := int64(len(files))
	start := time.Now()

	stopMem := make(chan struct{})
	getPeakMB := trackPeakMem(stopMem)

	var stopProgress chan struct{}
	if progressFn != nil {
		stopProgress = make(chan struct{})
		go func() {
			tick := time.NewTicker(500 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					done := atomic.LoadInt64(&stats.Files) + atomic.LoadInt64(&stats.Skipped) + atomic.LoadInt64(&stats.Errors)
					progressFn(done, totalFiles, atomic.LoadInt64(&stats.Errors),
						atomic.LoadInt64(&stats.ReadBytes), atomic.LoadInt64(&stats.WriteBytes),
						time.Since(start), getPeakMB())
				case <-stopProgress:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Workers)

	for _, fpath := range files {
		if gctx.Err() != nil {
			break
		}
		fpath := fpath
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			relPath, err := filepath.Rel(cfg.InputDir, fpath)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return nil
			}

			cid := cidFromRelPath(relPath)

			outRel := strings.TrimSuffix(relPath, ".html") + ".md"
			outPath := filepath.Join(cfg.OutputDir, outRel)

			if !cfg.Force {
				if _, err := os.Stat(outPath); err == nil {
					atomic.AddInt64(&stats.Skipped, 1)
					return nil
				}
			}

			// Read HTML
			htmlBytes, err := os.ReadFile(fpath)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				idx.Add(IndexRecord{CID: cid, Error: "read: " + err.Error()})
				return nil
			}
			atomic.AddInt64(&stats.ReadBytes, int64(len(htmlBytes)))

			// Convert
			var result Result
			if cfg.Fast {
				result = ConvertFast(htmlBytes, "")
			} else {
				result = Convert(htmlBytes, "")
			}

			// Write .md file
			if result.HasContent && result.Markdown != "" {
				if err := writeRawFile(outPath, []byte(result.Markdown)); err != nil {
					atomic.AddInt64(&stats.Errors, 1)
					idx.Add(IndexRecord{CID: cid, Error: "write: " + err.Error()})
					return nil
				}
				atomic.AddInt64(&stats.WriteBytes, int64(result.MarkdownSize))
			}

			// Record in index
			ratio := float64(0)
			if result.HTMLSize > 0 {
				ratio = float64(result.MarkdownSize) / float64(result.HTMLSize)
			}
			idx.Add(IndexRecord{
				CID:              cid,
				HTMLSize:         result.HTMLSize,
				MarkdownSize:     result.MarkdownSize,
				HTMLTokens:       result.HTMLTokens,
				MarkdownTokens:   result.MarkdownTokens,
				CompressionRatio: ratio,
				Title:            truncate(result.Title, 500),
				Language:         result.Language,
				HasContent:       result.HasContent,
				ConvertMs:        result.ConvertMs,
				Error:            result.Error,
			})

			if result.Error != "" {
				atomic.AddInt64(&stats.Errors, 1)
			} else {
				atomic.AddInt64(&stats.Files, 1)
			}
			return nil
		})
	}

	err = g.Wait()
	if stopProgress != nil {
		close(stopProgress)
	}
	close(stopMem)
	stats.Duration = time.Since(start)
	stats.PeakMemMB = getPeakMB()
	return &stats, err
}
