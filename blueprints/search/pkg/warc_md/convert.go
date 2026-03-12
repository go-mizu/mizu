package warc_md

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	mdpkg "github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	"golang.org/x/sync/errgroup"
)

// ConvertConfig configures Phase 2: warc_single/**/*.warc → markdown/**/*.md
type ConvertConfig struct {
	InputDir  string // warc_single/ base directory
	OutputDir string // markdown/ base directory
	Workers   int    // parallel workers (0 = NumCPU)
	Force     bool   // re-convert existing files
}

// RunConvert reads each .warc file (which contains raw HTML bytes),
// converts to Markdown, and writes .md files.
//
// Workers process files in parallel.
func RunConvert(ctx context.Context, cfg ConvertConfig, progressFn ProgressFunc) (*PhaseStats, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	// Collect all .warc files from input dir
	var files []string
	_ = filepath.WalkDir(cfg.InputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".warc") {
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

			recordID := recordIDFromWARCSinglePath(fpath)
			outPath := MarkdownFilePath(cfg.OutputDir, recordID)

			if !cfg.Force {
				if _, err := os.Stat(outPath); err == nil {
					atomic.AddInt64(&stats.Skipped, 1)
					return nil
				}
			}

			// Read HTML bytes (the .warc file contains raw HTML)
			htmlBytes, err := os.ReadFile(fpath)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return nil
			}
			atomic.AddInt64(&stats.ReadBytes, int64(len(htmlBytes)))

			// Convert HTML → Markdown
			result := mdpkg.Convert(htmlBytes, "")

			// Write .md file (only if content was extracted)
			if result.HasContent && result.Markdown != "" {
				if err := writeRawFile(outPath, []byte(result.Markdown)); err != nil {
					atomic.AddInt64(&stats.Errors, 1)
					return nil
				}
				atomic.AddInt64(&stats.WriteBytes, int64(result.MarkdownSize))
			}

			if result.Error != "" {
				atomic.AddInt64(&stats.Errors, 1)
			} else {
				atomic.AddInt64(&stats.Files, 1)
			}
			return nil
		})
	}

	werr := g.Wait()
	if stopProgress != nil {
		close(stopProgress)
	}
	close(stopMem)
	stats.Duration = time.Since(start)
	stats.PeakMemMB = getPeakMB()
	return &stats, werr
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
