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

	kgzip "github.com/klauspost/compress/gzip"
	"golang.org/x/sync/errgroup"
)

// CompressConfig configures Phase 3: markdown/**/*.md → markdown_gz/**/*.md.gz
type CompressConfig struct {
	InputDir  string // markdown/ base directory
	OutputDir string // markdown_gz/ base directory
	Workers   int    // parallel workers (0 = NumCPU)
	Force     bool   // re-compress existing files
}

// RunCompress reads each .md file, compresses it with gzip (BestSpeed),
// and writes the result as a .md.gz file to OutputDir.
func RunCompress(ctx context.Context, cfg CompressConfig, progressFn ProgressFunc) (*PhaseStats, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	// Collect all .md files
	var files []string
	_ = filepath.WalkDir(cfg.InputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
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

			relPath, err := filepath.Rel(cfg.InputDir, fpath)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return nil
			}

			outPath := filepath.Join(cfg.OutputDir, relPath+".gz")

			if !cfg.Force {
				if _, err := os.Stat(outPath); err == nil {
					atomic.AddInt64(&stats.Skipped, 1)
					return nil
				}
			}

			// Read .md
			data, err := os.ReadFile(fpath)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return nil
			}
			atomic.AddInt64(&stats.ReadBytes, int64(len(data)))

			// Compress to .md.gz
			if err := compressToGz(outPath, data); err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return nil
			}

			if fi, err := os.Stat(outPath); err == nil {
				atomic.AddInt64(&stats.WriteBytes, fi.Size())
			}
			atomic.AddInt64(&stats.Files, 1)
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

// compressToGz writes data as a gzip-compressed file (BestSpeed), atomically.
func compressToGz(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	gz, err := kgzip.NewWriterLevel(f, kgzip.BestSpeed)
	if err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if _, err := gz.Write(data); err != nil {
		gz.Close()
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := gz.Close(); err != nil {
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
