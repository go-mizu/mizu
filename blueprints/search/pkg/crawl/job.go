package crawl

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

// JobConfig configures a two-pass recrawl job.
type JobConfig struct {
	Engine            string        // "keepalive" | "epoll" | "swarm" | "rawhttp"; empty = "keepalive"
	Workers           int           // -1 or 0 = auto from SysInfo
	MaxConnsPerDomain int           // -1 or 0 = auto from SysInfo
	Timeout           time.Duration // pass-1 per-request timeout
	RetryTimeout      time.Duration // pass-2 timeout; 0 or NoRetry=true skips pass 2
	NoRetry           bool

	StatusOnly          bool
	InsecureTLS         bool
	DomainFailThreshold int
	DomainTimeout       time.Duration // 0=disabled, <0=adaptive per-domain
	AdaptiveTimeoutMax  time.Duration // upper bound for adaptive DomainTimeout (0=default 10min); lower values free workers faster from slow domains
	DomainDeadProbe     int           // abandon dead-HTTP domains after N timeouts with 0 successes (0=disabled)
	DomainStallRatio    int           // abandon stalling domains when timeouts ≥ successes×ratio (0=disabled, e.g. 20 = >95% timeout rate)
	BatchSize           int

	SysInfo *SysInfo // nil = auto-gather via LoadOrGatherSysInfo

	// Storage — injected by caller. nil = DevNull (results/failures discarded).
	// OpenResultWriter is called once; the writer is shared across pass 1 and pass 2.
	// OpenFailureWriter is called once for pass 1, then again for pass 2 (appends).
	OpenResultWriter  func() (ResultWriter, error)
	OpenFailureWriter func() (FailureWriter, error)

	// LoadRetrySeeds is called after pass 1 to fetch URLs for pass 2.
	// If nil or returns empty → pass 2 is skipped.
	LoadRetrySeeds func(ctx context.Context, since time.Time) ([]SeedURL, error)

	// Progress hooks for CLI display.
	Notifier DomainNotifier

	// ChunkMode controls seed delivery: "stream" (default) | "batch" | "pipeline"
	ChunkMode string
	ChunkSize int    // domains per batch; 0 = auto
	SeedPath  string // for "pipeline" mode (cursor reads DB directly)

	// Pass2Workers overrides worker count for pass 2; 0 = same as pass 1
	Pass2Workers int

	// Logger is used for internal soft errors (nil = slog.Default())
	Logger *slog.Logger
}

// JobResult holds combined statistics from both passes.
type JobResult struct {
	Pass1   *Stats
	Pass2   *Stats    // nil if pass 2 was not run
	Total   *Stats    // merged Pass1 + Pass2
	Start   time.Time
	End     time.Time
	Workers int       // resolved worker count (after auto-config)
}

// RunJob executes a two-pass recrawl job.
//
// Pass 1: run engine with cfg.Timeout.
// Pass 2: if RetryTimeout > 0 and !NoRetry, close FailureWriter, call LoadRetrySeeds,
//
//	re-run with DomainFailThreshold=0 and cfg.RetryTimeout.
//
// If OpenResultWriter/OpenFailureWriter are nil, DevNull writers are used.
// Hardware auto-config (GOMEMLIMIT, workers) is applied when Workers <= 0.
func RunJob(ctx context.Context, seeds []SeedURL, dns DNSCache, cfg JobConfig) (*JobResult, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// ── Hardware auto-config ──────────────────────────────────────────────
	si := cfg.SysInfo
	if si == nil {
		gathered := LoadOrGatherSysInfo("", 0)
		si = &gathered
	}

	if si.MemTotalMB > 0 {
		// GOMEMLIMIT = 75% of total RAM. Using total (not available) avoids
		// stale-cache issues: available fluctuates when sysinfo is measured
		// while a prior run is still in memory. Total is stable and seeds
		// (~1 GB for 7M URLs) need headroom above the startup heap peak.
		if limit := int64(si.MemTotalMB) * 1024 * 1024 * 75 / 100; limit > 0 {
			debug.SetMemoryLimit(limit)
		}
	}

	workers := cfg.Workers
	maxConns := cfg.MaxConnsPerDomain
	if workers <= 0 {
		autoCfg, _ := AutoConfigKeepAlive(*si, !cfg.StatusOnly)
		workers = autoCfg.Workers
		if maxConns <= 0 {
			maxConns = autoCfg.MaxConnsPerDomain
		}
	} else if maxConns <= 0 {
		maxConns = clamp(si.CPUCount*2, 4, 16)
	}

	engCfg := DefaultConfig()
	engCfg.Workers = workers
	engCfg.MaxConnsPerDomain = maxConns
	engCfg.Timeout = cfg.Timeout
	engCfg.StatusOnly = cfg.StatusOnly
	engCfg.InsecureTLS = cfg.InsecureTLS
	if cfg.DomainFailThreshold >= 0 {
		engCfg.DomainFailThreshold = cfg.DomainFailThreshold
	}
	if cfg.DomainTimeout != 0 {
		engCfg.DomainTimeout = cfg.DomainTimeout // <0 = adaptive sentinel
	}
	if cfg.AdaptiveTimeoutMax > 0 {
		engCfg.AdaptiveTimeoutMax = cfg.AdaptiveTimeoutMax
	}
	if cfg.DomainDeadProbe > 0 {
		engCfg.DomainDeadProbe = cfg.DomainDeadProbe
	}
	if cfg.DomainStallRatio > 0 {
		engCfg.DomainStallRatio = cfg.DomainStallRatio
	}
	if cfg.BatchSize > 0 {
		engCfg.BatchSize = cfg.BatchSize
	}
	engCfg.Notifier = cfg.Notifier

	if dns == nil {
		dns = &NoopDNS{}
	}

	// ── Open result writer (shared across both passes) ───────────────────
	var resultWriter ResultWriter
	if cfg.OpenResultWriter != nil {
		var err error
		resultWriter, err = cfg.OpenResultWriter()
		if err != nil {
			return nil, err
		}
		defer resultWriter.Close()
	} else {
		resultWriter = &DevNullResultWriter{}
	}

	// ── Open failure writer for pass 1 ────────────────────────────────────
	failureWriter1, err := openFailureWriter(cfg)
	if err != nil {
		return nil, err
	}

	eng, err := New(cfg.Engine)
	if err != nil {
		return nil, err
	}
	if eng == nil {
		eng, _ = New("keepalive")
	}

	start := time.Now()

	// ── Pass 1 ────────────────────────────────────────────────────────────
	pass1Stats, runErr := runWithChunkMode(ctx, eng, seeds, dns, engCfg, cfg, resultWriter, failureWriter1)
	if pass1Stats != nil {
		pass1Stats.Workers = workers
	}
	failureWriter1.Close() // release DuckDB lock before LoadRetrySeeds opens same file

	result := &JobResult{
		Pass1:   pass1Stats,
		Start:   start,
		Workers: workers,
	}

	if runErr != nil {
		result.End = time.Now()
		result.Total = pass1Stats
		return result, runErr
	}

	// ── Pass 2 ────────────────────────────────────────────────────────────
	doRetry := !cfg.NoRetry &&
		cfg.RetryTimeout > 0 &&
		cfg.LoadRetrySeeds != nil &&
		ctx.Err() == nil

	if doRetry {
		retrySeeds, rErr := cfg.LoadRetrySeeds(ctx, start)
		if rErr != nil {
			logger.Warn("RunJob: LoadRetrySeeds failed", "err", rErr)
		} else if len(retrySeeds) > 0 {
			failureWriter2, _ := openFailureWriter(cfg)

			retryCfg := engCfg
			retryCfg.Timeout = cfg.RetryTimeout
			retryCfg.DomainFailThreshold = 0
			retryCfg.DomainTimeout = cfg.RetryTimeout * 3
			// Disable adaptive timeout for pass 2: fast domains skew the adaptive P95
			// down, cutting off slow-but-rescuable URLs that need the full retry timeout.
			retryCfg.DisableAdaptiveTimeout = true
			// DomainDeadProbe=2: after 2 timeouts with 0 pass-2 successes, abandon domain.
			// Eliminates false negatives from unfiltered retry set — alive-but-slow domains
			// continue, dead domains fail fast (one 15s batch) and are skipped.
			retryCfg.DomainDeadProbe = 2
			if cfg.Pass2Workers > 0 {
				retryCfg.Workers = cfg.Pass2Workers
			}

			eng2, _ := New(cfg.Engine)
			if eng2 == nil {
				eng2, _ = New("keepalive")
			}
			pass2Stats, _ := eng2.Run(ctx, retrySeeds, dns, retryCfg, resultWriter, failureWriter2)
			failureWriter2.Close()
			result.Pass2 = pass2Stats
		}
	}

	result.End = time.Now()
	result.Total = mergeStats(result.Pass1, result.Pass2)
	return result, nil
}

// openFailureWriter opens a FailureWriter or returns a DevNull.
func openFailureWriter(cfg JobConfig) (FailureWriter, error) {
	if cfg.OpenFailureWriter != nil {
		fw, err := cfg.OpenFailureWriter()
		if err != nil {
			return nil, err
		}
		return fw, nil
	}
	return &DevNullFailureWriter{}, nil
}

// runWithChunkMode dispatches seeds to the engine using the configured chunk mode.
func runWithChunkMode(ctx context.Context, eng Engine, seeds []SeedURL, dns DNSCache,
	engCfg Config, jobCfg JobConfig, rw ResultWriter, fw FailureWriter) (*Stats, error) {

	mode := jobCfg.ChunkMode
	if mode == "" {
		mode = "stream"
	}

	switch mode {
	case "batch":
		return runBatchMode(ctx, eng, seeds, dns, engCfg, jobCfg, rw, fw)
	case "pipeline":
		if jobCfg.SeedPath == "" {
			return eng.Run(ctx, seeds, dns, engCfg, rw, fw)
		}
		si := LoadOrGatherSysInfo("", 0)
		batchSize := jobCfg.ChunkSize
		if batchSize <= 0 {
			batchSize = AutoBatchDomains(int(si.MemAvailableMB), 3, 256)
		}
		rdb, _ := rw.(ShardReopener) // nil-safe: rdb will be nil if rw doesn't implement it
		pStats, pErr := RunPipeline(ctx, PipelineConfig{
			Cfg:       engCfg,
			DNS:       dns,
			Results:   rw,
			Failures:  fw,
			RDB:       rdb,
			SeedPath:  jobCfg.SeedPath,
			BatchSize: batchSize,
			AvailMB:   int(si.MemAvailableMB),
		})
		return pStats, pErr
	default: // "stream"
		return eng.Run(ctx, seeds, dns, engCfg, rw, fw)
	}
}

// runBatchMode groups seeds by domain and processes them in batches.
func runBatchMode(ctx context.Context, eng Engine, seeds []SeedURL, dns DNSCache,
	engCfg Config, jobCfg JobConfig, rw ResultWriter, fw FailureWriter) (*Stats, error) {

	si := LoadOrGatherSysInfo("", 0)
	batchDomains := jobCfg.ChunkSize
	if batchDomains <= 0 {
		batchDomains = AutoBatchDomains(int(si.MemAvailableMB), 3, 256)
	}

	domainMap := make(map[string][]SeedURL)
	for _, s := range seeds {
		domainMap[s.Domain] = append(domainMap[s.Domain], s)
	}
	keys := make([]string, 0, len(domainMap))
	for d := range domainMap {
		keys = append(keys, d)
	}

	var combined *Stats
	for i := 0; i < len(keys); i += batchDomains {
		if ctx.Err() != nil {
			break
		}
		end := min(i+batchDomains, len(keys))
		var batch []SeedURL
		for _, d := range keys[i:end] {
			batch = append(batch, domainMap[d]...)
		}
		batchStats, err := eng.Run(ctx, batch, dns, engCfg, rw, fw)
		if err != nil && ctx.Err() == nil {
			return combined, err
		}
		combined = mergeStats(combined, batchStats)
	}
	return combined, nil
}

// mergeStats combines pass1 and pass2 stats into a Total.
func mergeStats(pass1, pass2 *Stats) *Stats {
	if pass1 == nil && pass2 == nil {
		return &Stats{}
	}
	if pass2 == nil {
		t := *pass1
		return &t
	}
	if pass1 == nil {
		t := *pass2
		return &t
	}
	peakRPS := pass1.PeakRPS
	if pass2.PeakRPS > peakRPS {
		peakRPS = pass2.PeakRPS
	}
	return &Stats{
		Total:    pass1.Total + pass2.Total,
		OK:       pass1.OK + pass2.OK,
		Failed:   pass1.Failed + pass2.Failed,
		Timeout:  pass1.Timeout + pass2.Timeout,
		Skipped:  pass1.Skipped + pass2.Skipped,
		Bytes:    pass1.Bytes + pass2.Bytes,
		Duration: pass1.Duration + pass2.Duration,
		PeakRPS:  peakRPS,
	}
}
