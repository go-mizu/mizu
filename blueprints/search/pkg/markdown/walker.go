package markdown

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
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

// WalkConfig configures the walker that batch-converts a bodystore to markdown.
type WalkConfig struct {
	InputDir  string // bodystore root (e.g. ~/data/common-crawl/bodies)
	OutputDir string // markdown output root (e.g. ~/data/common-crawl/markdown)
	IndexPath string // DuckDB index path (e.g. ~/data/common-crawl/markdown/index.duckdb)
	Workers   int    // parallel workers (default: NumCPU)
	Force     bool   // re-convert existing files
	BatchSize int    // DB write batch size (default: 1000)
	Fast      bool   // use go-readability instead of trafilatura (3-8x faster, slightly lower quality)
}

// WalkStats holds final summary statistics.
type WalkStats struct {
	Converted  int64
	Skipped    int64
	Errors     int64
	TotalFiles int64

	TotalHTMLBytes int64
	TotalMDBytes   int64
	Duration       time.Duration
}

// ProgressFunc is called periodically with current stats.
type ProgressFunc func(converted, skipped, errors, total int64, htmlBytes, mdBytes int64, elapsed time.Duration)

// Walk scans InputDir for .gz files, converts each to markdown, writes .md.gz
// to OutputDir, and records metadata in the IndexDB. It calls progressFn
// periodically (may be nil).
func Walk(ctx context.Context, cfg WalkConfig, progressFn ProgressFunc) (*WalkStats, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}

	// Collect all .gz files
	var files []string
	err := filepath.WalkDir(cfg.InputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if !d.IsDir() && strings.HasSuffix(path, ".gz") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk input dir: %w", err)
	}

	if len(files) == 0 {
		return &WalkStats{}, nil
	}

	// Open index DB
	idx, err := OpenIndex(cfg.IndexPath, cfg.BatchSize)
	if err != nil {
		return nil, err
	}
	defer idx.Close()

	// Ensure output dir exists
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir output: %w", err)
	}

	var stats WalkStats
	totalFiles := int64(len(files))
	stats.TotalFiles = totalFiles
	start := time.Now()

	// Progress ticker
	var stopProgress chan struct{}
	if progressFn != nil {
		stopProgress = make(chan struct{})
		go func() {
			tick := time.NewTicker(500 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					progressFn(
						atomic.LoadInt64(&stats.Converted),
						atomic.LoadInt64(&stats.Skipped),
						atomic.LoadInt64(&stats.Errors),
						totalFiles,
						atomic.LoadInt64(&stats.TotalHTMLBytes),
						atomic.LoadInt64(&stats.TotalMDBytes),
						time.Since(start),
					)
				case <-stopProgress:
					return
				}
			}
		}()
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Workers)

	// Reusable gzip reader pool
	var gzPool sync.Pool

	for _, fpath := range files {
		fpath := fpath
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			// Determine relative path and CID
			relPath, err := filepath.Rel(cfg.InputDir, fpath)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return nil
			}

			// CID: reconstruct from path  ab/cd/ef...89.gz → sha256:abcdef...89
			cid := cidFromPath(relPath)

			// Output path: same relative path but .gz → .md.gz
			outRel := strings.TrimSuffix(relPath, ".gz") + ".md.gz"
			outPath := filepath.Join(cfg.OutputDir, outRel)

			// Skip if output exists and not forced
			if !cfg.Force {
				if _, err := os.Stat(outPath); err == nil {
					atomic.AddInt64(&stats.Skipped, 1)
					return nil
				}
			}

			// Read and decompress
			htmlBytes, err := readGzFile(fpath, &gzPool)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				_ = idx.Add(IndexRecord{CID: cid, Error: "read: " + err.Error()})
				return nil
			}

			atomic.AddInt64(&stats.TotalHTMLBytes, int64(len(htmlBytes)))

			// Convert (no URL available from bodystore — bodies are content-addressed by SHA)
			var result Result
			if cfg.Fast {
				result = ConvertFast(htmlBytes, "")
			} else {
				result = Convert(htmlBytes, "")
			}

			// Write output
			if result.HasContent && result.Markdown != "" {
				if err := writeGzFile(outPath, []byte(result.Markdown)); err != nil {
					atomic.AddInt64(&stats.Errors, 1)
					_ = idx.Add(IndexRecord{CID: cid, Error: "write: " + err.Error()})
					return nil
				}
				atomic.AddInt64(&stats.TotalMDBytes, int64(result.MarkdownSize))
			}

			// Record in index
			ratio := float64(0)
			if result.HTMLSize > 0 {
				ratio = float64(result.MarkdownSize) / float64(result.HTMLSize)
			}
			_ = idx.Add(IndexRecord{
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
				atomic.AddInt64(&stats.Converted, 1)
			}
			return nil
		})
	}

	err = g.Wait()
	if stopProgress != nil {
		close(stopProgress)
	}
	stats.Duration = time.Since(start)

	return &stats, err
}

// cidFromPath reconstructs CID from bodystore relative path.
// ab/cd/ef0123456789...rest.gz → sha256:abcdef0123456789...rest
func cidFromPath(relPath string) string {
	// Normalize separators
	relPath = filepath.ToSlash(relPath)
	parts := strings.Split(relPath, "/")
	if len(parts) != 3 {
		return "unknown:" + relPath
	}
	name := strings.TrimSuffix(parts[2], ".gz")
	return "sha256:" + parts[0] + parts[1] + name
}

func readGzFile(path string, pool *sync.Pool) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var gz *gzip.Reader
	if v := pool.Get(); v != nil {
		gz = v.(*gzip.Reader)
		if err := gz.Reset(f); err != nil {
			// Discard corrupted reader — do not return to pool
			return nil, err
		}
	} else {
		gz, err = gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
	}

	data, err := io.ReadAll(gz)
	gz.Close()
	pool.Put(gz)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeGzFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	gz, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
