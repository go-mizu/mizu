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
)

// RunJob dispatches a job to the appropriate self-contained task in a background
// goroutine. Each per-task State is bridged to Manager.UpdateProgress via a
// non-blocking buffered channel so that slow WS broadcasts never stall the task.
func (m *Manager) RunJob(job *Job) {
	go func() {
		logInfof("job run id=%s type=%s crawl=%s files=%s engine=%s source=%s format=%s",
			job.ID, job.Config.Type, job.Config.CrawlID, job.Config.Files,
			job.Config.Engine, job.Config.Source, job.Config.Format)

		ctx, cancel := context.WithCancel(context.Background())
		m.SetRunning(job.ID, cancel)

		var err error
		switch job.Config.Type {
		case "download":
			err = m.runDownloadJob(ctx, job)
		case "markdown":
			err = m.runMarkdownJob(ctx, job)
		case "pack":
			err = m.runPackJob(ctx, job)
		case "index":
			err = m.runIndexJob(ctx, job)
		default:
			m.Fail(job.ID, fmt.Errorf("unknown job type: %s", job.Config.Type))
			return
		}

		if err != nil {
			if ctx.Err() != nil {
				logInfof("job run id=%s cancelled via context", job.ID)
				return
			}
			m.Fail(job.ID, err)
			return
		}

		m.Complete(job.ID, fmt.Sprintf("%s completed", job.Config.Type))
	}()
}

// ── Per-task adapters ─────────────────────────────────────────────────────

func (m *Manager) runDownloadJob(ctx context.Context, job *Job) error {
	paths, selected, err := m.resolveFiles(ctx, job)
	if err != nil {
		return err
	}
	crawlDir := m.resolveJobCrawlDir(job)
	task := NewDownloadTask(crawlDir, paths, selected)

	emit := nonBlockingEmit(func(s *DownloadState) {
		m.UpdateProgress(job.ID, s.Progress,
			fmt.Sprintf("[%d/%d] %s", s.FileIndex+1, s.FileTotal, s.FileName),
			s.BytesPerSec)
	})

	_, err = task.Run(ctx, emit)
	return err
}

func (m *Manager) runMarkdownJob(ctx context.Context, job *Job) error {
	paths, selected, err := m.resolveFiles(ctx, job)
	if err != nil {
		return err
	}
	crawlID, crawlDir := m.resolveJobCrawl(job)
	task := NewMarkdownTask(crawlID, crawlDir, paths, selected)

	emit := nonBlockingEmit(func(s *MarkdownState) {
		m.UpdateProgress(job.ID, s.Progress,
			fmt.Sprintf("[%d/%d] %s %s docs=%d",
				s.FileIndex+1, s.FileTotal, s.WARCIndex, s.Phase, s.DocsProcessed),
			s.WriteRate)
	})

	_, err = task.Run(ctx, emit)
	return err
}

func (m *Manager) runPackJob(ctx context.Context, job *Job) error {
	paths, selected, err := m.resolveFiles(ctx, job)
	if err != nil {
		return err
	}
	crawlDir := m.resolveJobCrawlDir(job)
	format := job.Config.Format
	task := NewPackTask(crawlDir, paths, selected, format)

	emit := nonBlockingEmit(func(s *PackState) {
		m.UpdateProgress(job.ID, s.Progress,
			fmt.Sprintf("[%d/%d] %s %s docs=%d",
				s.FileIndex+1, s.FileTotal, s.WARCIndex, s.Format, s.DocsProcessed),
			s.DocsPerSec)
	})

	_, err = task.Run(ctx, emit)
	return err
}

func (m *Manager) runIndexJob(ctx context.Context, job *Job) error {
	paths, selected, err := m.resolveFiles(ctx, job)
	if err != nil {
		return err
	}
	crawlDir := m.resolveJobCrawlDir(job)
	task := NewIndexTask(crawlDir, paths, selected, job.Config.Engine, job.Config.Source)

	emit := nonBlockingEmit(func(s *IndexState) {
		m.UpdateProgress(job.ID, s.Progress,
			fmt.Sprintf("[%d/%d] %s %s/%s docs=%d",
				s.FileIndex+1, s.FileTotal, s.WARCIndex, s.Engine, s.Source, s.DocsIndexed),
			s.DocsPerSec)
	})

	_, err = task.Run(ctx, emit)
	return err
}

// nonBlockingEmit wraps an emit function with a buffered channel so that slow
// consumers never block the task goroutine. Intermediate states are dropped
// when the channel is full — only the latest state matters for progress.
func nonBlockingEmit[S any](fn func(*S)) func(*S) {
	ch := make(chan *S, 64)
	go func() {
		for s := range ch {
			fn(s)
		}
	}()
	return func(s *S) {
		if s == nil {
			return
		}
		select {
		case ch <- s:
		default:
			// Drop intermediate state — consumer is behind.
		}
	}
}

// ── Manifest resolution ──────────────────────────────────────────────────

// resolveFiles fetches the manifest for the job's crawl and applies the file
// selector to produce the paths list and selected indices.
func (m *Manager) resolveFiles(ctx context.Context, job *Job) (paths []string, selected []int, err error) {
	crawlID := job.Config.CrawlID
	if crawlID == "" {
		crawlID = m.crawlID
	}
	paths, err = m.getManifestPaths(ctx, crawlID)
	if err != nil {
		return nil, nil, fmt.Errorf("manifest: %w", err)
	}
	selected, err = parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return nil, nil, fmt.Errorf("selector: %w", err)
	}
	return paths, selected, nil
}

// resolveJobCrawl returns the effective crawlID and crawlDir for a job.
func (m *Manager) resolveJobCrawl(job *Job) (crawlID, crawlDir string) {
	crawlID = job.Config.CrawlID
	if crawlID == "" {
		crawlID = m.crawlID
	}
	crawlDir = m.resolveCrawlDir(crawlID)
	return
}

// resolveJobCrawlDir returns only the crawlDir for a job.
func (m *Manager) resolveJobCrawlDir(job *Job) string {
	_, dir := m.resolveJobCrawl(job)
	return dir
}

func (m *Manager) getManifestPaths(ctx context.Context, crawlID string) ([]string, error) {
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

// ── Compatibility wrappers ───────────────────────────────────────────────

// execTask is a compatibility wrapper for tests and call-sites that need
// synchronous execution with no progress emission.
func (m *Manager) execTask(ctx context.Context, job *Job) error {
	switch job.Config.Type {
	case "download":
		return m.runDownloadJob(ctx, job)
	case "markdown":
		return m.runMarkdownJob(ctx, job)
	case "pack":
		return m.runPackJob(ctx, job)
	case "index":
		return m.runIndexJob(ctx, job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Config.Type)
	}
}

// Kept for existing tests.
func (m *Manager) execDownload(ctx context.Context, job *Job) error { return m.runDownloadJob(ctx, job) }
func (m *Manager) execMarkdown(ctx context.Context, job *Job) error { return m.runMarkdownJob(ctx, job) }
func (m *Manager) execPack(ctx context.Context, job *Job) error     { return m.runPackJob(ctx, job) }
func (m *Manager) execIndex(ctx context.Context, job *Job) error    { return m.runIndexJob(ctx, job) }

// ── Pure helpers ─────────────────────────────────────────────────────────

// warcFileIndex extracts the zero-padded 5-digit WARC file index from a path.
// Falls back to fmt.Sprintf("%05d", fallback) if not parseable.
func warcFileIndex(warcPath string, fallback int) string {
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

// packPath returns the expected pack file path for the given format and WARC index.
func packPath(packDir, format, warcIdx string) (string, error) {
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
// Supports: "0", "0-4", "all", "".
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
