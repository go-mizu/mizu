package markdown

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// ExtractConfig configures Phase 1: decompress bodies/*.gz → html/*.html.
type ExtractConfig struct {
	InputDir  string // bodystore root (e.g. ~/data/common-crawl/bodies)
	OutputDir string // html output root (e.g. ~/data/common-crawl/html)
	Workers   int    // parallel workers (0 = NumCPU)
	Force     bool   // re-extract existing files
}

// RunExtract decompresses every .gz file in InputDir and writes the raw bytes
// as a .html file to OutputDir, preserving the sub-directory structure.
func RunExtract(ctx context.Context, cfg ExtractConfig, progressFn PhaseProgressFunc) (*PhaseStats, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	// Collect .gz files
	var files []string
	_ = filepath.WalkDir(cfg.InputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".gz") {
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

	var gzPool sync.Pool

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

			outRel := strings.TrimSuffix(relPath, ".gz") + ".html"
			outPath := filepath.Join(cfg.OutputDir, outRel)

			if !cfg.Force {
				if _, err := os.Stat(outPath); err == nil {
					atomic.AddInt64(&stats.Skipped, 1)
					return nil
				}
			}

			// Decompress via shared pool (readGzFile defined in walker.go)
			data, err := readGzFile(fpath, &gzPool)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return nil
			}
			atomic.AddInt64(&stats.ReadBytes, int64(len(data)))

			// Write raw HTML
			if err := writeRawFile(outPath, data); err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return nil
			}
			atomic.AddInt64(&stats.WriteBytes, int64(len(data)))
			atomic.AddInt64(&stats.Files, 1)
			return nil
		})
	}

	err := g.Wait()
	if stopProgress != nil {
		close(stopProgress)
	}
	close(stopMem)
	stats.Duration = time.Since(start)
	stats.PeakMemMB = getPeakMB()
	return &stats, err
}

// writeRawFile writes data to path atomically (via tmp file + rename).
func writeRawFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
