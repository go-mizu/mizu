package warc_md

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunFilePipeline executes two phases sequentially:
//
//	Phase 1: extract HTML records → warc_single/
//	Phase 2: convert HTML → plain .md → markdown/{warcIdx}/
//
// Temp directories (warc_single/ and markdown_raw/) are removed after
// success unless cfg.KeepTemp is set.
//
// p1Fn, p2Fn are per-phase progress callbacks (may be nil).
func RunFilePipeline(ctx context.Context, cfg Config, warcIdx string, inputFiles []string,
	p1Fn, p2Fn ProgressFunc) (*PipelineResult, error) {

	start := time.Now()

	outDir := cfg.MarkdownWarcDir(warcIdx)

	// ── Phase 1: Extract ────────────────────────────────────────────────────
	s1, err := RunExtract(ctx, ExtractConfig{
		InputFiles:  inputFiles,
		OutputDir:   cfg.WARCSingleDir(),
		Workers:     len(inputFiles),
		Force:       cfg.Force,
		StatusCode:  cfg.StatusCode,
		MIMEFilter:  cfg.MIMEFilter,
		MaxBodySize: cfg.MaxBodySize,
	}, p1Fn)
	if err != nil {
		return nil, fmt.Errorf("phase 1 extract: %w", err)
	}

	// ── Phase 2: Convert ────────────────────────────────────────────────────
	s2, err := RunConvert(ctx, ConvertConfig{
		InputDir:  cfg.WARCSingleDir(),
		OutputDir: outDir,
		Workers:   cfg.ConvertWorkers(),
		Force:     cfg.Force,
	}, p2Fn)
	if err != nil {
		return nil, fmt.Errorf("phase 2 convert: %w", err)
	}

	result := &PipelineResult{
		Extract:  s1,
		Convert:  s2,
		Duration: time.Since(start),
	}

	if !cfg.KeepTemp {
		os.RemoveAll(cfg.WARCSingleDir())
		os.RemoveAll(cfg.MarkdownDir())
	}

	return result, nil
}

// DiskUsageBytes sums the size of all regular files under path.
func DiskUsageBytes(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// DiskUsageMdGz sums only *.md.gz files under dir.
func DiskUsageMdGz(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md.gz") {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}
