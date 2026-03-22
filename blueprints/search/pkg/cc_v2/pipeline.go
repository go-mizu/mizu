package cc_v2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
)

// PackFunc is the function signature for packing a WARC into a parquet.
// This is injected from the CLI layer since packDirectToParquet lives in cli/.
type PackFunc func(ctx context.Context, cfg warcmd.PackConfig, parquetPath string,
	progressFn warcmd.ProgressFunc) (rows, htmlBytes, mdBytes int64, stats *warcmd.PackStats, err error)

// Pipeline downloads WARC files and converts them to parquet shards.
// Uses 1-ahead prefetch: downloads next WARC while packing current one.
// This overlaps download and pack for ~2x throughput vs sequential.
type Pipeline struct {
	cfg        PipelineConfig
	store      Store
	log        *Logger
	ccClient   *cc.Client
	manifest   []string // resolved manifest paths
	warcDir    string
	parquetDir string
	packFn     PackFunc
}

// NewPipeline creates a pipeline worker.
func NewPipeline(cfg PipelineConfig, store Store, packFn PackFunc) *Pipeline {
	return &Pipeline{
		cfg:        cfg,
		store:      store,
		log:        NewLogger("pipeline", store),
		ccClient:   cc.NewClient("", 4),
		warcDir:    filepath.Join(cfg.DataDir, "warc"),
		parquetDir: filepath.Join(cfg.DataDir, "parquet"),
		packFn:     packFn,
	}
}

// prefetchResult holds the result of a background download.
type prefetchResult struct {
	idx      int
	warcPath string
	durDl    time.Duration
	err      error
}

// Run processes all configured shard indices with 1-ahead prefetch.
func (p *Pipeline) Run(ctx context.Context) error {
	// Ensure directories exist.
	for _, dir := range []string{p.warcDir, p.parquetDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Resolve manifest once.
	if err := p.resolveManifest(ctx); err != nil {
		return err
	}

	p.log.PrintBanner("Pipeline", map[string]string{
		"Crawl":   p.cfg.CrawlID,
		"Shards":  fmt.Sprintf("%d", len(p.cfg.Indices)),
		"WARC":    p.warcDir,
		"Parquet": p.parquetDir,
		"Redis":   fmt.Sprintf("%v", p.store.Available()),
	})

	// Build work list: filter out already committed/ready shards up front.
	var work []int
	for _, idx := range p.cfg.Indices {
		if p.store.IsCommitted(ctx, idx) {
			continue
		}
		shard := fmt.Sprintf("%05d", idx)
		pqPath := filepath.Join(p.parquetDir, shard+".parquet")
		if fileExists(pqPath) {
			continue
		}
		work = append(work, idx)
	}
	p.log.Info("work", "total", len(p.cfg.Indices), "todo", len(work),
		"skipped", len(p.cfg.Indices)-len(work))

	if len(work) == 0 {
		p.log.Info("pipeline complete", "shards", 0)
		return nil
	}

	// ── Prefetch loop ────────────────────────────────────────────────
	// Start downloading work[0] immediately, then for each shard:
	//   1. Wait for current download to finish
	//   2. Start downloading next shard in background
	//   3. Pack current shard (overlaps with next download)
	//   4. Mark ready

	var prefetchMu sync.Mutex
	var prefetchWg sync.WaitGroup
	var prefetchRes *prefetchResult

	// startPrefetch kicks off a download in the background.
	startPrefetch := func(idx int) {
		prefetchMu.Lock()
		prefetchRes = nil
		prefetchMu.Unlock()

		prefetchWg.Add(1)
		go func() {
			defer prefetchWg.Done()
			warcPath, durDl, err := p.download(ctx, idx)
			prefetchMu.Lock()
			prefetchRes = &prefetchResult{idx: idx, warcPath: warcPath, durDl: durDl, err: err}
			prefetchMu.Unlock()
		}()
	}

	for i, idx := range work {
		if ctx.Err() != nil {
			prefetchWg.Wait()
			return ctx.Err()
		}

		p.log.Info("start", "shard", idx, "progress", fmt.Sprintf("%d/%d", i+1, len(work)))

		// Claim the shard (distributed lock).
		if !p.store.Claim(ctx, idx) {
			p.log.Info("skip", "shard", idx, "reason", "claimed by another")
			continue
		}

		// Get the WARC (either from prefetch or direct download).
		var warcPath string
		var durDl time.Duration
		var dlErr error

		prefetchMu.Lock()
		pf := prefetchRes
		prefetchMu.Unlock()

		if pf != nil && pf.idx == idx {
			// Prefetch already completed for this shard.
			prefetchWg.Wait()
			warcPath, durDl, dlErr = pf.warcPath, pf.durDl, pf.err
		} else {
			// No prefetch available — wait for any in-flight, then download directly.
			prefetchWg.Wait()
			warcPath, durDl, dlErr = p.download(ctx, idx)
		}

		if dlErr != nil {
			p.store.Release(ctx, idx)
			if p.cfg.SkipErrors {
				p.log.Warn("download failed, skipping", "shard", idx, "err", dlErr)
				continue
			}
			return fmt.Errorf("shard %d download: %w", idx, dlErr)
		}

		// Start prefetching the NEXT shard while we pack this one.
		if i+1 < len(work) {
			nextIdx := work[i+1]
			// Only prefetch if not already committed and claimable.
			if !p.store.IsCommitted(ctx, nextIdx) {
				shard := fmt.Sprintf("%05d", nextIdx)
				pqPath := filepath.Join(p.parquetDir, shard+".parquet")
				if !fileExists(pqPath) {
					startPrefetch(nextIdx)
				}
			}
		}

		// Pack to parquet (runs while next download is in progress).
		pqPath := filepath.Join(p.parquetDir, fmt.Sprintf("%05d.parquet", idx))
		stats, err := p.pack(ctx, idx, warcPath, pqPath)
		if err != nil {
			p.store.Release(ctx, idx)
			if p.cfg.SkipErrors {
				p.log.Warn("pack failed, skipping", "shard", idx, "err", err)
				continue
			}
			prefetchWg.Wait()
			return fmt.Errorf("shard %d pack: %w", idx, err)
		}
		stats.DurDlS = int64(durDl.Seconds())

		// Mark ready — watcher will pick it up and handle cleanup.
		p.store.MarkReady(ctx, idx, pqPath, warcPath, stats)
		p.log.Info("ready", "shard", idx, "rows", stats.Rows,
			"size", FmtBytes(stats.PqBytes), "dl", durDl.Round(time.Second),
			"pack", time.Duration(stats.DurPackS)*time.Second)
	}

	prefetchWg.Wait()
	p.log.Info("pipeline complete", "shards", len(work))
	return nil
}

func (p *Pipeline) resolveManifest(ctx context.Context) error {
	paths, err := p.ccClient.DownloadManifest(ctx, p.cfg.CrawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	p.manifest = paths
	return nil
}

func (p *Pipeline) download(ctx context.Context, idx int) (string, time.Duration, error) {
	t0 := time.Now()

	// Check if WARC already on disk (from a previous crashed run).
	if existing := p.findWARC(idx); existing != "" {
		p.log.Info("warc cached", "shard", idx, "path", filepath.Base(existing))
		return existing, 0, nil
	}

	if idx < 0 || idx >= len(p.manifest) {
		return "", 0, fmt.Errorf("shard %d out of range (0–%d)", idx, len(p.manifest)-1)
	}
	remotePath := p.manifest[idx]
	localPath := filepath.Join(p.warcDir, filepath.Base(remotePath))
	tmpPath := localPath + ".dl.tmp"

	// Download to .tmp then rename (atomic).
	if err := p.ccClient.DownloadFile(ctx, remotePath, tmpPath, nil); err != nil {
		os.Remove(tmpPath)
		return "", 0, err
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		os.Remove(tmpPath)
		return "", 0, err
	}

	p.store.RecordEvent(ctx, "downloaded")
	dur := time.Since(t0)
	fi, _ := os.Stat(localPath)
	size := int64(0)
	if fi != nil {
		size = fi.Size()
	}
	p.log.Info("downloaded", "shard", idx, "size", FmtBytes(size), "dur", dur.Round(time.Second))
	return localPath, dur, nil
}

// findWARC looks for an existing raw WARC on disk for this shard index.
func (p *Pipeline) findWARC(idx int) string {
	if idx < 0 || idx >= len(p.manifest) {
		return ""
	}
	localPath := filepath.Join(p.warcDir, filepath.Base(p.manifest[idx]))
	if fileExists(localPath) {
		return localPath
	}
	return ""
}

func (p *Pipeline) pack(ctx context.Context, idx int, warcPath, pqPath string) (*ShardStats, error) {
	t0 := time.Now()
	tmpPath := pqPath + ".tmp"

	// Clean up stale tmp from previous crash.
	os.Remove(tmpPath)

	cfg := warcmd.PackConfig{
		InputFiles:   []string{warcPath},
		Workers:      0,
		Force:        true,
		LightConvert: true,
		StatusCode:   200,
		MIMEFilter:   "text/html",
		MaxBodySize:  512 * 1024,
	}

	rows, htmlBytes, mdBytes, packStats, err := p.packFn(ctx, cfg, tmpPath, nil)
	if err != nil {
		os.Remove(tmpPath)
		return nil, err
	}

	// Atomic rename.
	if err := os.Rename(tmpPath, pqPath); err != nil {
		os.Remove(tmpPath)
		return nil, err
	}

	fi, _ := os.Stat(pqPath)
	pqBytes := int64(0)
	if fi != nil {
		pqBytes = fi.Size()
	}

	durPack := int64(time.Since(t0).Seconds())
	peakRSS := int64(0)
	if packStats != nil {
		peakRSS = int64(packStats.PeakMemMB)
	}

	stats := &ShardStats{
		Rows:      rows,
		HTMLBytes: htmlBytes,
		MDBytes:   mdBytes,
		PqBytes:   pqBytes,
		DurPackS:  durPack,
		PeakRSSMB: peakRSS,
	}

	// Write .meta.json sidecar for watcher (backup if Redis is down).
	shard := fmt.Sprintf("%05d", idx)
	metaPath := filepath.Join(p.parquetDir, shard+".meta.json")
	metaJSON, _ := json.Marshal(stats)
	os.WriteFile(metaPath, metaJSON, 0o644)

	p.store.RecordEvent(ctx, "packed")
	return stats, nil
}
