package markdown

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WorkerBenchResult holds throughput results for a single worker count trial.
type WorkerBenchResult struct {
	Workers      int
	Processed    int64 // Files + Errors (total throughput, excluding skipped)
	Duration     time.Duration
	FilesPerSec  float64
	ReadMBPerSec float64
	ReadBytes    int64
}

// BenchmarkConvertWorkers benchmarks the convert phase across multiple worker
// counts on an evenly-spaced sample of up to sampleSize HTML files from inputDir.
// A temporary directory of symlinks is created for each trial so only the
// sampled files are processed, not the full inputDir.
// Returns results in the same order as workerCounts.
func BenchmarkConvertWorkers(ctx context.Context, inputDir string, workerCounts []int, fast bool, sampleSize int) ([]WorkerBenchResult, error) {
	// Collect all .html files
	var all []string
	_ = filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		all = append(all, path)
		return nil
	})
	if len(all) == 0 {
		return nil, nil // no files yet — caller handles gracefully
	}

	// Take evenly-spaced sample
	sample := all
	if sampleSize > 0 && len(all) > sampleSize {
		step := len(all) / sampleSize
		sampled := make([]string, 0, sampleSize)
		for i := 0; i < len(all) && len(sampled) < sampleSize; i += step {
			sampled = append(sampled, all[i])
		}
		sample = sampled
	}

	// Build a temporary input directory of symlinks → sampled files,
	// preserving the relative sub-directory structure so RunConvertPhase
	// walks and processes only those files.
	tmpIn, err := os.MkdirTemp("", "md-bench-in-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpIn)

	for _, fpath := range sample {
		rel, err := filepath.Rel(inputDir, fpath)
		if err != nil {
			continue
		}
		dst := filepath.Join(tmpIn, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			continue
		}
		_ = os.Symlink(fpath, dst) // best-effort; skip on error
	}

	var results []WorkerBenchResult
	for _, w := range workerCounts {
		if ctx.Err() != nil {
			break
		}

		tmpOut, err := os.MkdirTemp("", "md-bench-out-*")
		if err != nil {
			return results, err
		}

		cfg := ConvertPhaseConfig{
			InputDir:  tmpIn,
			OutputDir: tmpOut,
			IndexPath: filepath.Join(tmpOut, "bench.duckdb"),
			Workers:   w,
			BatchSize: 1000,
			Fast:      fast,
			Force:     true, // always re-process (temp output dir is fresh)
		}

		start := time.Now()
		stats, err := RunConvertPhase(ctx, cfg, nil)
		elapsed := time.Since(start)
		go os.RemoveAll(tmpOut) // best-effort async cleanup

		if err != nil || stats == nil {
			continue
		}

		// Throughput = files actually touched (successful + errored), not skipped
		processed := stats.Files + stats.Errors
		fps := float64(0)
		readMBs := float64(0)
		if elapsed.Seconds() > 0 {
			fps = float64(processed) / elapsed.Seconds()
			readMBs = float64(stats.ReadBytes) / (1024 * 1024) / elapsed.Seconds()
		}
		results = append(results, WorkerBenchResult{
			Workers:      w,
			Processed:    processed,
			Duration:     elapsed,
			FilesPerSec:  fps,
			ReadMBPerSec: readMBs,
			ReadBytes:    stats.ReadBytes,
		})
	}
	return results, nil
}
