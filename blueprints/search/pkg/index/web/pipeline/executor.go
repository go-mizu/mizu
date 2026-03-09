package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ccpkg "github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/scrape"
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
			err = runDownloadJob(ctx, m, job)
		case "markdown":
			err = runMarkdownJob(ctx, m, job)
		case "pack":
			err = runPackJob(ctx, m, job)
		case "index":
			err = runIndexJob(ctx, m, job)
		case "parquet_download":
			err = runParquetDownloadJob(ctx, m, job)
		case "scrape":
			err = runScrapeJob(ctx, m, job)
		case "scrape_markdown":
			err = runScrapeMarkdownJob(ctx, m, job)
		case "scrape_index":
			err = runScrapeIndexJob(ctx, m, job)
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

// ── Per-task adapters ─────────────────────────────────────────────────────────

func runDownloadJob(ctx context.Context, m *Manager, job *Job) error {
	paths, selected, err := resolveFiles(ctx, m, job)
	if err != nil {
		return err
	}
	crawlDir := resolveJobCrawlDir(m, job)
	task := cc.NewDownloadTask(crawlDir, paths, selected)

	emit := NonBlockingEmit(func(s *cc.DownloadState) {
		m.UpdateProgress(job.ID, s.Progress,
			fmt.Sprintf("[%d/%d] %s", s.FileIndex+1, s.FileTotal, s.FileName),
			s.BytesPerSec)
	})
	_, err = task.Run(ctx, emit)
	return err
}

func runMarkdownJob(ctx context.Context, m *Manager, job *Job) error {
	paths, selected, err := resolveFiles(ctx, m, job)
	if err != nil {
		return err
	}
	crawlID, crawlDir := resolveJobCrawl(m, job)
	task := cc.NewMarkdownTask(crawlID, crawlDir, paths, selected)

	emit := NonBlockingEmit(func(s *cc.MarkdownState) {
		m.UpdateProgress(job.ID, s.Progress,
			fmt.Sprintf("[%d/%d] %s %s docs=%d",
				s.FileIndex+1, s.FileTotal, s.WARCIndex, s.Phase, s.DocsProcessed),
			s.WriteRate)
	})
	_, err = task.Run(ctx, emit)
	return err
}

func runPackJob(ctx context.Context, m *Manager, job *Job) error {
	paths, selected, err := resolveFiles(ctx, m, job)
	if err != nil {
		return err
	}
	crawlDir := resolveJobCrawlDir(m, job)
	task := cc.NewPackTask(crawlDir, paths, selected, job.Config.Format)

	emit := NonBlockingEmit(func(s *cc.PackState) {
		m.UpdateProgress(job.ID, s.Progress,
			fmt.Sprintf("[%d/%d] %s %s docs=%d",
				s.FileIndex+1, s.FileTotal, s.WARCIndex, s.Format, s.DocsProcessed),
			s.DocsPerSec)
	})
	_, err = task.Run(ctx, emit)
	return err
}

func runIndexJob(ctx context.Context, m *Manager, job *Job) error {
	paths, selected, err := resolveFiles(ctx, m, job)
	if err != nil {
		return err
	}
	crawlDir := resolveJobCrawlDir(m, job)
	task := cc.NewIndexTask(crawlDir, paths, selected, job.Config.Engine, job.Config.Source)

	emit := NonBlockingEmit(func(s *cc.IndexState) {
		m.UpdateProgress(job.ID, s.Progress,
			fmt.Sprintf("[%d/%d] %s %s/%s docs=%d",
				s.FileIndex+1, s.FileTotal, s.WARCIndex, s.Engine, s.Source, s.DocsIndexed),
			s.DocsPerSec)
	})
	_, err = task.Run(ctx, emit)
	return err
}

func runScrapeJob(ctx context.Context, m *Manager, job *Job) error {
	domain := job.Config.Domain
	if domain == "" {
		return fmt.Errorf("scrape job missing domain")
	}
	dataDir := scrapeDataDir(m.baseDir)
	params := scrape.ParseStartParams(job.Config.Source)
	cfg := buildDCrawlerConfig(domain, dataDir, params)

	task := scrape.NewScrapeTask(cfg)
	emit := NonBlockingEmit(func(s *scrape.ScrapeState) {
		b, _ := json.Marshal(s)
		m.UpdateProgress(job.ID, s.Progress, string(b), s.PagesPerSec)
	})
	_, err := task.Run(ctx, emit)
	m.InvalidateScrapeCache()
	return err
}

func runScrapeMarkdownJob(ctx context.Context, m *Manager, job *Job) error {
	domain := job.Config.Domain
	if domain == "" {
		return fmt.Errorf("scrape_markdown job missing domain")
	}
	dataDir := scrapeDataDir(m.baseDir)
	task := scrape.NewScrapeMarkdownTask(domain, dataDir)

	emit := NonBlockingEmit(func(s *scrape.ScrapeMarkdownState) {
		msg := fmt.Sprintf("%s docs=%d/%.0f speed=%.0f/s",
			domain, s.DocsProcessed, float64(s.DocsTotal), s.DocsPerSec)
		m.UpdateProgress(job.ID, s.Progress, msg, s.DocsPerSec)
	})
	metric, err := task.Run(ctx, emit)
	if err == nil {
		m.Complete(job.ID, fmt.Sprintf("converted %d pages to markdown", metric.Docs))
	}
	m.InvalidateScrapeCache()
	return err
}

func runScrapeIndexJob(ctx context.Context, m *Manager, job *Job) error {
	domain := job.Config.Domain
	if domain == "" {
		return fmt.Errorf("scrape_index job missing domain")
	}
	dataDir := scrapeDataDir(m.baseDir)
	engine := job.Config.Engine
	if engine == "" {
		engine = "dahlia"
	}
	task := scrape.NewScrapeIndexTask(domain, dataDir, engine)

	emit := NonBlockingEmit(func(s *scrape.ScrapeIndexState) {
		msg := fmt.Sprintf("%s indexed=%d/%d speed=%.0f/s",
			domain, s.DocsIndexed, s.DocsTotal, s.DocsPerSec)
		m.UpdateProgress(job.ID, s.Progress, msg, s.DocsPerSec)
	})
	metric, err := task.Run(ctx, emit)
	if err == nil {
		m.Complete(job.ID, fmt.Sprintf("indexed %d docs with %s", metric.Docs, engine))
	}
	m.InvalidateScrapeCache()
	return err
}

func runParquetDownloadJob(ctx context.Context, m *Manager, job *Job) error {
	cfg := ccpkg.DefaultConfig()
	cfg.CrawlID = m.crawlID
	cfg.DataDir = filepath.Dir(m.baseDir)

	client := ccpkg.NewClient(cfg.BaseURL, cfg.TransportShards)

	var idxSet map[int]bool
	if job.Config.Files != "" && job.Config.Files != "all" {
		parts := strings.Split(job.Config.Files, ",")
		idxSet = make(map[int]bool, len(parts))
		for _, p := range parts {
			if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				idxSet[n] = true
			}
		}
	}

	opts := ccpkg.ParquetListOptions{}
	if job.Config.Source != "" && job.Config.Source != "all" {
		opts.Subset = job.Config.Source
	}

	files, err := ccpkg.ListParquetFiles(ctx, client, cfg, opts)
	if err != nil {
		return fmt.Errorf("parquet manifest: %w", err)
	}

	if idxSet != nil {
		filtered := files[:0]
		for _, f := range files {
			if idxSet[f.ManifestIndex] {
				filtered = append(filtered, f)
			}
		}
		files = filtered
	}

	toDownload := files[:0]
	for _, f := range files {
		localPath := ccpkg.LocalParquetPathForRemote(cfg, f.RemotePath)
		if !ccpkg.IsValidParquetFile(localPath) {
			toDownload = append(toDownload, f)
		}
	}

	if len(toDownload) == 0 {
		return nil
	}

	emit := NonBlockingEmit(func(p *ccpkg.DownloadProgress) {
		if p.TotalFiles <= 0 {
			return
		}
		pct := float64(p.FileIndex) / float64(p.TotalFiles)
		if p.Done || p.Skipped {
			pct = float64(p.FileIndex+1) / float64(p.TotalFiles)
		}
		msg := fmt.Sprintf("[%d/%d] %s", p.FileIndex+1, p.TotalFiles, p.File)
		m.UpdateProgress(job.ID, pct, msg, 0)
	})

	return ccpkg.DownloadParquetFiles(ctx, client, cfg, toDownload, cfg.IndexWorkers, func(p ccpkg.DownloadProgress) {
		emit(&p)
	})
}

// ── Manifest resolution ───────────────────────────────────────────────────────

func resolveFiles(ctx context.Context, m *Manager, job *Job) (paths []string, selected []int, err error) {
	crawlID := job.Config.CrawlID
	if crawlID == "" {
		crawlID = m.crawlID
	}
	paths, err = getManifestPaths(ctx, m, crawlID)
	if err != nil {
		return nil, nil, fmt.Errorf("manifest: %w", err)
	}
	selected, err = ParseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return nil, nil, fmt.Errorf("selector: %w", err)
	}
	return paths, selected, nil
}

func resolveJobCrawl(m *Manager, job *Job) (crawlID, crawlDir string) {
	crawlID = job.Config.CrawlID
	if crawlID == "" {
		crawlID = m.crawlID
	}
	crawlDir = jobCrawlDir(m, crawlID)
	return
}

func resolveJobCrawlDir(m *Manager, job *Job) string {
	_, dir := resolveJobCrawl(m, job)
	return dir
}

func jobCrawlDir(m *Manager, crawlID string) string {
	if crawlID == m.crawlID {
		return m.baseDir
	}
	return filepath.Join(filepath.Dir(m.baseDir), crawlID)
}

func getManifestPaths(ctx context.Context, m *Manager, crawlID string) ([]string, error) {
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
		client := ccpkg.NewClient("", 4)
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

// scrapeDataDir derives the crawler data dir from the dashboard base dir.
// Dashboard base dir is ~/data/common-crawl/{crawlID}; crawler data is ~/data/crawler.
func scrapeDataDir(baseDir string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(baseDir)), "crawler")
}

func buildDCrawlerConfig(domain, dataDir string, params scrape.StartParams) dcrawler.Config {
	cfg := dcrawler.DefaultConfig()
	cfg.Domain = domain
	cfg.DataDir = dataDir

	if params.Mode == "browser" {
		cfg.UseRod = true
		cfg.RodHeadless = true
		cfg.RodBlockResources = true
		// Flush more frequently so dashboard pages/stats reflect progress during active crawls.
		cfg.BatchSize = 50
		// Prefer throughput in dashboard browser crawls; most modern sites (including
		// Next.js/SSG) expose useful links before full visual stabilization.
		cfg.RodNoRenderWait = true
	}
	if params.MaxPages > 0 {
		cfg.MaxPages = params.MaxPages
	}
	if params.MaxDepth > 0 {
		cfg.MaxDepth = params.MaxDepth
	}
	if params.Workers > 0 {
		// Browser mode uses RodWorkers (tab count), HTTP mode uses Workers.
		if cfg.UseRod || cfg.UseLightpanda {
			cfg.RodWorkers = params.Workers
		} else {
			cfg.Workers = params.Workers
		}
	}
	if params.TimeoutS > 0 {
		cfg.Timeout = time.Duration(params.TimeoutS) * time.Second
	} else if cfg.UseRod || cfg.UseLightpanda {
		// Browser mode needs a longer default deadline for challenge pages and JS-heavy apps.
		cfg.Timeout = 30 * time.Second
	}
	cfg.StoreBody = true
	cfg.Resume = params.Resume

	// Advanced options — match CLI crawl-domain flags.
	if params.NoRobots {
		cfg.RespectRobots = false
	}
	if params.NoSitemap {
		cfg.FollowSitemap = false
	}
	cfg.IncludeSubdomain = params.IncludeSubdomain
	if params.ScrollCount > 0 {
		cfg.ScrollCount = params.ScrollCount
	}
	cfg.Continuous = params.Continuous
	if params.StaleHours > 0 {
		cfg.StaleHours = params.StaleHours
	}
	if params.SeedURL != "" {
		cfg.SeedURLs = []string{params.SeedURL}
	}

	// Worker mode — proxy fetches through CF Worker.
	if params.Mode == "worker" {
		cfg.UseWorker = true
		cfg.WorkerToken = params.WorkerToken
		if cfg.WorkerToken == "" {
			cfg.WorkerToken = os.Getenv("CRAWLER_WORKER_TOKEN")
		}
		if params.WorkerURL != "" {
			cfg.WorkerURL = params.WorkerURL
		}
		cfg.WorkerBrowser = params.WorkerBrowser
	}
	return cfg
}
