package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// RunJob dispatches a job to the appropriate executor in a background goroutine.
func (m *JobManager) RunJob(job *Job) {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.SetRunning(job.ID, cancel)
		var err error
		switch job.Config.Type {
		case "download":
			err = m.execDownload(ctx, job)
		case "markdown":
			err = m.execMarkdown(ctx, job)
		case "pack":
			err = m.execPack(ctx, job)
		case "index":
			err = m.execIndex(ctx, job)
		default:
			err = fmt.Errorf("unknown job type: %s", job.Config.Type)
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			m.Fail(job.ID, err)
		} else {
			m.Complete(job.ID, fmt.Sprintf("%s completed", job.Config.Type))
		}
	}()
}

// ── Download Executor ────────────────────────────────────────────────────

func (m *JobManager) execDownload(ctx context.Context, job *Job) error {
	crawlID := job.Config.CrawlID
	if crawlID == "" {
		crawlID = m.crawlID
	}

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return fmt.Errorf("files: %w", err)
	}

	warcDir := filepath.Join(m.baseDir, "warc")
	if err := os.MkdirAll(warcDir, 0o755); err != nil {
		return fmt.Errorf("creating warc dir: %w", err)
	}

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		remotePath := paths[idx]
		localPath := filepath.Join(warcDir, filepath.Base(remotePath))

		m.UpdateProgress(job.ID,
			float64(i)/float64(len(selected)),
			fmt.Sprintf("downloading file %d of %d: %s", i+1, len(selected), filepath.Base(remotePath)),
			0,
		)

		err := client.DownloadFile(ctx, remotePath, localPath, func(received, total int64) {
			filePct := float64(0)
			if total > 0 {
				filePct = float64(received) / float64(total)
			}
			overallPct := (float64(i) + filePct) / float64(len(selected))
			m.UpdateProgress(job.ID,
				overallPct,
				fmt.Sprintf("downloading file %d of %d: %s (%.0f%%)", i+1, len(selected), filepath.Base(remotePath), filePct*100),
				0,
			)
		})
		if err != nil {
			return fmt.Errorf("download %s: %w", filepath.Base(remotePath), err)
		}
	}

	return nil
}

// ── Markdown Executor ────────────────────────────────────────────────────

func (m *JobManager) execMarkdown(ctx context.Context, job *Job) error {
	return fmt.Errorf("not yet implemented via dashboard -- use CLI")
}

// ── Pack Executor ────────────────────────────────────────────────────────

func (m *JobManager) execPack(ctx context.Context, job *Job) error {
	crawlID := job.Config.CrawlID
	if crawlID == "" {
		crawlID = m.crawlID
	}

	format := job.Config.Format
	if format == "" {
		format = "parquet"
	}

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return fmt.Errorf("files: %w", err)
	}

	packDir := filepath.Join(m.baseDir, "pack")

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		warcIdx := warcIndexFromPath(paths[idx], idx)
		markdownDir := filepath.Join(m.baseDir, "markdown", warcIdx)
		if _, err := os.Stat(markdownDir); os.IsNotExist(err) {
			return fmt.Errorf("markdown dir not found: %s", markdownDir)
		}

		packFile, err := packFilePath(packDir, format, warcIdx)
		if err != nil {
			return err
		}

		m.UpdateProgress(job.ID,
			float64(i)/float64(len(selected)),
			fmt.Sprintf("packing [%s] warc %s (%d/%d)", format, warcIdx, i+1, len(selected)),
			0,
		)

		progress := func(stats *index.PipelineStats) {
			total := stats.TotalFiles.Load()
			done := stats.DocsIndexed.Load()
			pct := float64(0)
			if total > 0 {
				pct = float64(done) / float64(total)
			}
			elapsed := time.Since(stats.StartTime).Seconds()
			rate := float64(0)
			if elapsed > 0 {
				rate = float64(done) / elapsed
			}
			overallPct := (float64(i) + pct) / float64(len(selected))
			m.UpdateProgress(job.ID, overallPct,
				fmt.Sprintf("packing [%s] warc %s: %d/%d docs", format, warcIdx, done, total),
				rate,
			)
		}

		var packErr error
		switch format {
		case "parquet":
			_, packErr = index.PackParquet(ctx, markdownDir, packFile, 0, 5000, progress)
		case "bin":
			_, packErr = index.PackFlatBin(ctx, markdownDir, packFile, 0, 5000, progress)
		case "markdown":
			_, packErr = index.PackFlatBinGz(ctx, markdownDir, packFile, 0, 5000, progress)
		default:
			return fmt.Errorf("unknown format %q (valid: parquet, bin, markdown)", format)
		}
		if packErr != nil {
			return fmt.Errorf("pack %s: %w", format, packErr)
		}
	}

	return nil
}

// ── Index Executor ───────────────────────────────────────────────────────

func (m *JobManager) execIndex(ctx context.Context, job *Job) error {
	crawlID := job.Config.CrawlID
	if crawlID == "" {
		crawlID = m.crawlID
	}

	engineName := job.Config.Engine
	if engineName == "" {
		engineName = "duckdb"
	}

	source := job.Config.Source
	if source == "" {
		source = "files"
	}

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return fmt.Errorf("files: %w", err)
	}

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		warcIdx := warcIndexFromPath(paths[idx], idx)
		outputDir := filepath.Join(m.baseDir, "fts", engineName, warcIdx)

		eng, err := index.NewEngine(engineName)
		if err != nil {
			return err
		}
		if err := eng.Open(ctx, outputDir); err != nil {
			return fmt.Errorf("open engine: %w", err)
		}

		m.UpdateProgress(job.ID,
			float64(i)/float64(len(selected)),
			fmt.Sprintf("indexing [%s] warc %s (%d/%d)", engineName, warcIdx, i+1, len(selected)),
			0,
		)

		var pipeErr error
		if source == "files" {
			sourceDir := filepath.Join(m.baseDir, "markdown", warcIdx)
			if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
				eng.Close()
				return fmt.Errorf("markdown dir not found: %s", sourceDir)
			}

			cfg := index.PipelineConfig{
				SourceDir: sourceDir,
				BatchSize: 5000,
			}
			progress := func(stats *index.PipelineStats) {
				total := stats.TotalFiles.Load()
				done := stats.DocsIndexed.Load()
				pct := float64(0)
				if total > 0 {
					pct = float64(done) / float64(total)
				}
				elapsed := time.Since(stats.StartTime).Seconds()
				rate := float64(0)
				if elapsed > 0 {
					rate = float64(done) / elapsed
				}
				overallPct := (float64(i) + pct) / float64(len(selected))
				m.UpdateProgress(job.ID, overallPct,
					fmt.Sprintf("indexing [%s] warc %s: %d/%d docs", engineName, warcIdx, done, total),
					rate,
				)
			}
			_, pipeErr = index.RunPipeline(ctx, eng, cfg, progress)
		} else {
			packDir := filepath.Join(m.baseDir, "pack")
			packFile, perr := packFilePath(packDir, source, warcIdx)
			if perr != nil {
				eng.Close()
				return perr
			}
			if _, err := os.Stat(packFile); os.IsNotExist(err) {
				eng.Close()
				return fmt.Errorf("pack file not found: %s", packFile)
			}

			progress := func(done, total int64, elapsed time.Duration) {
				pct := float64(0)
				if total > 0 {
					pct = float64(done) / float64(total)
				}
				rate := float64(0)
				if secs := elapsed.Seconds(); secs > 0 {
					rate = float64(done) / secs
				}
				overallPct := (float64(i) + pct) / float64(len(selected))
				m.UpdateProgress(job.ID, overallPct,
					fmt.Sprintf("indexing [%s<-%s] warc %s: %d docs", engineName, source, warcIdx, done),
					rate,
				)
			}

			switch source {
			case "parquet":
				_, pipeErr = index.RunPipelineFromParquet(ctx, eng, packFile, 5000, progress)
			case "bin":
				_, pipeErr = index.RunPipelineFromFlatBin(ctx, eng, packFile, 5000, progress)
			case "markdown":
				_, pipeErr = index.RunPipelineFromFlatBinGz(ctx, eng, packFile, 5000, progress)
			default:
				eng.Close()
				return fmt.Errorf("unknown source %q (valid: files, parquet, bin, markdown)", source)
			}
		}

		if pipeErr != nil {
			eng.Close()
			return pipeErr
		}

		// Finalize if the engine supports it (e.g., DuckDB BM25 index creation).
		if fin, ok := eng.(index.Finalizer); ok {
			if err := fin.Finalize(ctx); err != nil {
				eng.Close()
				return fmt.Errorf("finalize: %w", err)
			}
		}

		eng.Close()
	}

	return nil
}

// ── Helper Functions ─────────────────────────────────────────────────────
// Duplicated from cli/ to avoid circular import (cli imports web).

// warcIndexFromPath extracts the zero-padded 5-digit WARC file index from a WARC
// filename. Falls back to fmt.Sprintf("%05d", fallback) if not parseable.
//
//	"CC-MAIN-20260206181458-20260206211458-00000.warc.gz" -> "00000"
func warcIndexFromPath(warcPath string, fallback int) string {
	base := filepath.Base(warcPath)
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".warc")
	parts := strings.Split(name, "-")
	if last := parts[len(parts)-1]; len(last) == 5 {
		if _, err := strconv.Atoi(last); err == nil {
			return last
		}
	}
	return fmt.Sprintf("%05d", fallback)
}

// packFilePath returns the expected pack file path for the given format and WARC index.
func packFilePath(packDir, format, warcIdx string) (string, error) {
	switch format {
	case "parquet":
		return filepath.Join(packDir, "parquet", warcIdx+".parquet"), nil
	case "bin":
		return filepath.Join(packDir, "bin", warcIdx+".bin"), nil
	case "markdown":
		return filepath.Join(packDir, "markdown", warcIdx+".bin.gz"), nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: parquet, bin, markdown)", format)
	}
}

// parseFileSelector parses a file selector string into a list of indices.
// Supports: "0", "0-4", "all".
func parseFileSelector(s string, total int) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "all" {
		idx := make([]int, total)
		for i := range idx {
			idx[i] = i
		}
		return idx, nil
	}

	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		lo, err1 := strconv.Atoi(parts[0])
		hi, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("invalid range %q", s)
		}
		if lo < 0 || hi >= total || lo > hi {
			return nil, fmt.Errorf("range %d-%d out of bounds (total: %d)", lo, hi, total)
		}
		idx := make([]int, hi-lo+1)
		for i := range idx {
			idx[i] = lo + i
		}
		return idx, nil
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("invalid file index %q", s)
	}
	if n < 0 || n >= total {
		return nil, fmt.Errorf("file index %d out of bounds (total: %d)", n, total)
	}
	return []int{n}, nil
}
