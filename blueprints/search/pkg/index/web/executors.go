package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	indexpack "github.com/go-mizu/mizu/blueprints/search/pkg/index/pack"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
)

// RunJob dispatches a job to the appropriate executor in a background goroutine.
func (m *JobManager) RunJob(job *Job) {
	go func() {
		logInfof("job run id=%s type=%s crawl=%s files=%s engine=%s source=%s format=%s",
			job.ID, job.Config.Type, job.Config.CrawlID, job.Config.Files, job.Config.Engine, job.Config.Source, job.Config.Format)
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
				logInfof("job run id=%s cancelled via context", job.ID)
				return
			}
			m.Fail(job.ID, err)
		} else {
			logInfof("job run id=%s completed successfully", job.ID)
			m.Complete(job.ID, fmt.Sprintf("%s completed", job.Config.Type))
		}
	}()
}

// ── Download Executor ────────────────────────────────────────────────────

func (m *JobManager) execDownload(ctx context.Context, job *Job) error {
	crawlID, crawlDir := m.resolveJobCrawl(job)
	logInfof("pipeline download start job=%s crawl=%s dir=%s", job.ID, crawlID, crawlDir)

	m.UpdateProgress(job.ID, 0, "fetching crawl manifest...", 0)
	paths, err := m.getManifestPaths(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return fmt.Errorf("files: %w", err)
	}

	warcDir := filepath.Join(crawlDir, "warc")
	if err := os.MkdirAll(warcDir, 0o755); err != nil {
		return fmt.Errorf("creating warc dir: %w", err)
	}
	client := cc.NewClient("", 4)

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		remotePath := paths[idx]
		localPath := filepath.Join(warcDir, filepath.Base(remotePath))
		logInfof("pipeline download file-start job=%s crawl=%s idx=%d remote=%s local=%s", job.ID, crawlID, idx, remotePath, localPath)

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
		logInfof("pipeline download file-done job=%s crawl=%s idx=%d", job.ID, crawlID, idx)
	}
	logInfof("pipeline download done job=%s crawl=%s selected=%d", job.ID, crawlID, len(selected))

	return nil
}

// ── Markdown Executor ────────────────────────────────────────────────────

func (m *JobManager) execMarkdown(ctx context.Context, job *Job) error {
	crawlID, crawlDir := m.resolveJobCrawl(job)
	logInfof("pipeline markdown start job=%s crawl=%s dir=%s", job.ID, crawlID, crawlDir)

	m.UpdateProgress(job.ID, 0, "fetching crawl manifest...", 0)
	paths, err := m.getManifestPaths(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return fmt.Errorf("files: %w", err)
	}
	if len(selected) == 0 {
		return nil
	}

	cfg := warcmd.DefaultConfig(crawlID)
	cfg.DataDir = filepath.Dir(crawlDir)
	cfg.Workers = runtime.NumCPU()
	cfg.Force = false
	cfg.KeepTemp = false

	warcDir := filepath.Join(crawlDir, "warc")
	totalFiles := float64(len(selected))

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		warcPath := paths[idx]
		warcIdx := warcIndexFromPath(warcPath, idx)
		localPath := filepath.Join(warcDir, filepath.Base(warcPath))
		logInfof("pipeline markdown file-start job=%s crawl=%s warc_idx=%s local=%s", job.ID, crawlID, warcIdx, localPath)
		if _, err := os.Stat(localPath); err != nil {
			return fmt.Errorf("warc file not found: %s (run download step first)", localPath)
		}

		basePct := float64(i) / totalFiles
		fileWeight := 1.0 / totalFiles
		m.UpdateProgress(job.ID, basePct, fmt.Sprintf("markdown [%s] file %d/%d: preparing", warcIdx, i+1, len(selected)), 0)

		p1Fn := func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, _ float64) {
			localPct := phaseProgress(done, total)
			overall := basePct + fileWeight*(0.5*localPct)
			rate := phaseRate(done, elapsed)
			msg := fmt.Sprintf(
				"markdown [%s] extract %d/%d docs (err=%d, r=%.1fMB/s, w=%.1fMB/s)",
				warcIdx, done, total, errors, mbPerSec(readBytes, elapsed), mbPerSec(writeBytes, elapsed),
			)
			m.UpdateProgress(job.ID, overall, msg, rate)
		}
		p2Fn := func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, _ float64) {
			localPct := phaseProgress(done, total)
			overall := basePct + fileWeight*(0.5+0.5*localPct)
			rate := phaseRate(done, elapsed)
			msg := fmt.Sprintf(
				"markdown [%s] convert %d/%d docs (err=%d, r=%.1fMB/s, w=%.1fMB/s)",
				warcIdx, done, total, errors, mbPerSec(readBytes, elapsed), mbPerSec(writeBytes, elapsed),
			)
			m.UpdateProgress(job.ID, overall, msg, rate)
		}

		if _, err := warcmd.RunFilePipeline(ctx, cfg, warcIdx, []string{localPath}, p1Fn, p2Fn); err != nil {
			return fmt.Errorf("markdown file %s: %w", warcIdx, err)
		}
		logInfof("pipeline markdown file-done job=%s crawl=%s warc_idx=%s", job.ID, crawlID, warcIdx)

		donePct := float64(i+1) / totalFiles
		m.UpdateProgress(job.ID, donePct, fmt.Sprintf("markdown [%s] complete (%d/%d)", warcIdx, i+1, len(selected)), 0)
	}
	logInfof("pipeline markdown done job=%s crawl=%s selected=%d", job.ID, crawlID, len(selected))

	return nil
}

// ── Pack Executor ────────────────────────────────────────────────────────

func (m *JobManager) execPack(ctx context.Context, job *Job) error {
	crawlID, crawlDir := m.resolveJobCrawl(job)

	format := job.Config.Format
	if format == "" {
		format = "parquet"
	}
	logInfof("pipeline pack start job=%s crawl=%s dir=%s format=%s", job.ID, crawlID, crawlDir, format)

	m.UpdateProgress(job.ID, 0, "fetching crawl manifest...", 0)
	paths, err := m.getManifestPaths(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return fmt.Errorf("files: %w", err)
	}

	packDir := filepath.Join(crawlDir, "pack")

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		warcIdx := warcIndexFromPath(paths[idx], idx)
		markdownDir := filepath.Join(crawlDir, "markdown", warcIdx)
		logInfof("pipeline pack file-start job=%s crawl=%s warc_idx=%s format=%s", job.ID, crawlID, warcIdx, format)
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

		progress := func(stats *indexpack.PipelineStats) {
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
			_, packErr = indexpack.PackParquet(ctx, markdownDir, packFile, 0, 5000, progress)
		case "bin":
			_, packErr = indexpack.PackFlatBin(ctx, markdownDir, packFile, 0, 5000, progress)
		case "duckdb":
			_, packErr = packDuckDBRaw(ctx, markdownDir, packFile, 0, 5000, progress)
		case "markdown":
			_, packErr = indexpack.PackFlatBinGz(ctx, markdownDir, packFile, 0, 5000, progress)
		default:
			return fmt.Errorf("unknown format %q (valid: parquet, bin, duckdb, markdown)", format)
		}
		if packErr != nil {
			return fmt.Errorf("pack %s: %w", format, packErr)
		}
		logInfof("pipeline pack file-done job=%s crawl=%s warc_idx=%s format=%s", job.ID, crawlID, warcIdx, format)
	}
	logInfof("pipeline pack done job=%s crawl=%s selected=%d format=%s", job.ID, crawlID, len(selected), format)

	return nil
}

// ── Index Executor ───────────────────────────────────────────────────────

func (m *JobManager) execIndex(ctx context.Context, job *Job) error {
	crawlID, crawlDir := m.resolveJobCrawl(job)

	engineName := job.Config.Engine
	if engineName == "" {
		engineName = "rose"
	}

	source := job.Config.Source
	if source == "" {
		source = "files"
	}
	logInfof("pipeline index start job=%s crawl=%s dir=%s engine=%s source=%s", job.ID, crawlID, crawlDir, engineName, source)

	m.UpdateProgress(job.ID, 0, "fetching crawl manifest...", 0)
	paths, err := m.getManifestPaths(ctx, crawlID)
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
		outputDir := filepath.Join(crawlDir, "fts", engineName, warcIdx)
		logInfof("pipeline index file-start job=%s crawl=%s warc_idx=%s engine=%s source=%s", job.ID, crawlID, warcIdx, engineName, source)

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
			sourceDir := filepath.Join(crawlDir, "markdown", warcIdx)
			if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
				eng.Close()
				return fmt.Errorf("markdown dir not found: %s", sourceDir)
			}

			cfg := indexpack.PipelineConfig{
				SourceDir: sourceDir,
				BatchSize: 5000,
			}
			progress := func(stats *indexpack.PipelineStats) {
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
			_, pipeErr = indexpack.RunPipeline(ctx, eng, cfg, progress)
		} else {
			packDir := filepath.Join(crawlDir, "pack")
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
				_, pipeErr = indexpack.RunPipelineFromParquet(ctx, eng, packFile, 5000, progress)
			case "bin":
				_, pipeErr = indexpack.RunPipelineFromFlatBin(ctx, eng, packFile, 5000, progress)
			case "duckdb":
				_, pipeErr = runPipelineFromDuckDBRaw(ctx, eng, packFile, 5000, progress)
			case "markdown":
				_, pipeErr = indexpack.RunPipelineFromFlatBinGz(ctx, eng, packFile, 5000, progress)
			default:
				eng.Close()
				return fmt.Errorf("unknown source %q (valid: files, parquet, bin, duckdb, markdown)", source)
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
		logInfof("pipeline index file-done job=%s crawl=%s warc_idx=%s engine=%s source=%s", job.ID, crawlID, warcIdx, engineName, source)
	}
	logInfof("pipeline index done job=%s crawl=%s selected=%d engine=%s source=%s", job.ID, crawlID, len(selected), engineName, source)

	return nil
}

// resolveJobCrawl returns the effective crawlID and crawlDir for a job,
// falling back to the JobManager's configured crawl when the job has none.
func (m *JobManager) resolveJobCrawl(job *Job) (crawlID, crawlDir string) {
	crawlID = job.Config.CrawlID
	if crawlID == "" {
		crawlID = m.crawlID
	}
	crawlDir = m.resolveCrawlDir(crawlID)
	return
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
	case "duckdb":
		return filepath.Join(packDir, "duckdb", warcIdx+".duckdb"), nil
	case "markdown":
		return filepath.Join(packDir, "markdown", warcIdx+".bin.gz"), nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: parquet, bin, duckdb, markdown)", format)
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

func (m *JobManager) getManifestPaths(ctx context.Context, crawlID string) ([]string, error) {
	const manifestTTL = 10 * time.Minute

	now := time.Now()
	m.manifestMu.Lock()
	if entry, ok := m.manifestCache[crawlID]; ok && now.Sub(entry.fetchedAt) < manifestTTL && len(entry.paths) > 0 {
		cached := append([]string(nil), entry.paths...)
		m.manifestMu.Unlock()
		logInfof("manifest cache hit crawl=%s entries=%d age=%s", crawlID, len(cached), now.Sub(entry.fetchedAt).Round(time.Second))
		return cached, nil
	}
	m.manifestMu.Unlock()

	m.mu.RLock()
	fetchFn := m.manifestFetch
	m.mu.RUnlock()
	if fetchFn == nil {
		client := cc.NewClient("", 4)
		fetchFn = func(ctx context.Context, crawlID string) ([]string, error) {
			return client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
		}
	}

	logInfof("manifest cache miss crawl=%s fetching remote manifest", crawlID)
	paths, err := fetchFn(ctx, crawlID)
	if err != nil {
		logErrorf("manifest fetch failed crawl=%s err=%v", crawlID, err)
		return nil, err
	}

	m.manifestMu.Lock()
	m.manifestCache[crawlID] = manifestCacheEntry{
		paths:     append([]string(nil), paths...),
		fetchedAt: now,
	}
	m.manifestMu.Unlock()
	logInfof("manifest fetched crawl=%s entries=%d", crawlID, len(paths))

	return paths, nil
}

func phaseProgress(done, total int64) float64 {
	if total <= 0 {
		if done > 0 {
			return 0.95
		}
		return 0
	}
	p := float64(done) / float64(total)
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

func phaseRate(done int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(done) / elapsed.Seconds()
}

func mbPerSec(bytes int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(bytes) / (1024 * 1024) / elapsed.Seconds()
}
