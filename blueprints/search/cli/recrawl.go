package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

// recrawlJobArgs holds all arguments for runRecrawlJob.
type recrawlJobArgs struct {
	Seeds        []crawl.SeedURL
	DNSCache     crawl.DNSCache
	JobCfg       crawl.JobConfig
	ResultDir    string
	FailedDBPath string
	WriterMode   string // "duckdb" | "bin" | "devnull"
	SlowDomainMs int64
	SegSizeMB    int
	BodyStoreDir string
	DBShards     int
	DBMemMB      int
	SysInfo      crawl.SysInfo
}

// runRecrawlJob is the shared two-pass recrawl runner used by both HN and CC pipelines.
// It handles writer setup, progress display, and calls crawl.RunJob.
func runRecrawlJob(ctx context.Context, args recrawlJobArgs) error {
	si := args.SysInfo

	ls := &v3LiveStats{slowDomainMs: args.SlowDomainMs}
	args.JobCfg.Notifier = ls
	args.JobCfg.SysInfo = &si

	writerMode := strings.TrimSpace(strings.ToLower(args.WriterMode))
	if writerMode == "" {
		writerMode = "duckdb"
	}

	hwmon := crawl.NewHWMonitor(2 * time.Second)
	defer hwmon.Stop()
	ls.hwmon = hwmon

	var rdb *store.ResultDB
	var binWriter *crawl.BinSegWriter

	if writerMode != "devnull" {
		if err := os.MkdirAll(args.ResultDir, 0o755); err != nil {
			return fmt.Errorf("create result dir: %w", err)
		}
		var err error
		rdb, err = store.NewResultDB(args.ResultDir, args.DBShards, args.JobCfg.BatchSize, args.DBMemMB)
		if err != nil {
			return fmt.Errorf("opening result db: %w", err)
		}
		// rdb lifecycle: for duckdb mode, RunJob closes via resultWriter.Close().
		// For bin mode, rdb is closed explicitly below after RunJob returns.
	}

	switch writerMode {
	case "bin":
		segDir := filepath.Join(filepath.Dir(args.ResultDir), "segments")
		if n, err := crawl.DrainLeftovers(segDir, rdb); err != nil {
			fmt.Fprintf(os.Stderr, "  [warn] drain leftovers: %v\n", err)
		} else if n > 0 {
			fmt.Printf("  Recovered %s records from leftover segments\n",
				labelStyle.Render(formatInt64Exact(n)))
		}
		var bwErr error
		binWriter, bwErr = crawl.NewBinSegWriter(segDir, args.SegSizeMB, int(si.MemAvailableMB), rdb)
		if bwErr != nil {
			rdb.Close() // rdb is not owned by binWriter; close before returning
			return fmt.Errorf("creating bin writer: %w", bwErr)
		}
		// binWriter lifecycle: RunJob closes via resultWriter.Close().
		// rdb is closed explicitly below after RunJob returns (binWriter.Close flushes to rdb first).
		ls.binWriter = binWriter
	}

	if args.BodyStoreDir != "" {
		// body store setup if needed
		_ = args.BodyStoreDir
	}

	// aliveCount is populated by v3ProgressWriter during pass 1.
	// It tracks per-domain success counts (domain → *atomic.Int64).
	// LoadRetrySeeds filters the pass-2 retry list using a ratio-based check:
	// only retry URLs from domains where pass-1 successes >= retry-seed count for that domain.
	// This skips dead domains (0 successes) AND barely-alive domains (high timeout rate).
	var aliveCount sync.Map

	// Inject storage constructors
	if writerMode != "devnull" {
		args.JobCfg.OpenResultWriter = func() (crawl.ResultWriter, error) {
			if writerMode == "bin" {
				return &v3ProgressWriter{inner: binWriter, ls: ls, aliveCount: &aliveCount}, nil
			}
			return &v3ProgressWriter{inner: rdb, ls: ls, aliveCount: &aliveCount}, nil
		}
		args.JobCfg.OpenFailureWriter = func() (crawl.FailureWriter, error) {
			fdb, err := store.OpenFailedDB(args.FailedDBPath)
			if err != nil {
				return nil, fmt.Errorf("opening failed db: %w", err)
			}
			return &v3ProgressFailureWriter{inner: fdb, ls: ls}, nil
		}
		args.JobCfg.LoadRetrySeeds = func(ctx context.Context, since time.Time) ([]crawl.SeedURL, error) {
			seeds, err := store.LoadRetryURLsSince(args.FailedDBPath, since)
			if err != nil || len(seeds) == 0 {
				return seeds, err
			}
			// Count how many retry seeds exist per domain (= pass-1 timeout count).
			timeoutCounts := make(map[string]int64, len(seeds)/5)
			for _, s := range seeds {
				timeoutCounts[s.Domain]++
			}
			// Filter: only retry domains where pass-1 successes >= timeout count.
			// Ratio ≥1 means the domain was at least 50% OK in pass 1 — worth retrying.
			// Domains with 0 successes or high timeout rates are skipped entirely.
			total := len(seeds)
			filtered := seeds[:0]
			for _, s := range seeds {
				if v, ok := aliveCount.Load(s.Domain); ok {
					successes := v.(*atomic.Int64).Load()
					if successes >= timeoutCounts[s.Domain] {
						filtered = append(filtered, s)
					}
				}
			}
			skipped := total - len(filtered)
			if skipped > 0 {
				fmt.Printf("  Pass-2 filter  %s → %s retry seeds (%s skipped: dead or low-success-rate domains)\n",
					labelStyle.Render(formatInt64Exact(int64(total))),
					labelStyle.Render(formatInt64Exact(int64(len(filtered)))),
					labelStyle.Render(formatInt64Exact(int64(skipped))),
				)
			}
			return filtered, nil
		}
	}

	// Progress display
	stdoutStat, statErr := os.Stdout.Stat()
	isTTY := statErr == nil && stdoutStat.Mode()&os.ModeCharDevice != 0
	progressInterval := 500 * time.Millisecond
	if !isTTY {
		progressInterval = 2 * time.Second
	}

	engineName := args.JobCfg.Engine
	if engineName == "" {
		engineName = "keepalive"
	}
	seedTotal := int64(len(args.Seeds))
	start := time.Now()

	// Build a display config from JobConfig fields
	displayCfg := crawl.DefaultConfig()
	displayCfg.Timeout = args.JobCfg.Timeout
	if args.JobCfg.StatusOnly {
		displayCfg.StatusOnly = true
	}

	progressCtx, cancelProgress := context.WithCancel(ctx)
	defer cancelProgress()
	progressDone := make(chan struct{})

	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(progressInterval)
		defer ticker.Stop()
		var displayLines int
		for {
			select {
			case <-progressCtx.Done():
				return
			case t := <-ticker.C:
				ls.updateSpeed(t)
				output := v3RenderProgress(ls, displayCfg, engineName, seedTotal, start, isTTY)
				if isTTY {
					if displayLines > 0 {
						fmt.Printf("\033[%dA\033[J", displayLines)
					}
					fmt.Print(output)
					displayLines = strings.Count(output, "\n")
				} else {
					fmt.Print(output)
				}
			}
		}
	}()

	jobResult, err := crawl.RunJob(ctx, args.Seeds, args.DNSCache, args.JobCfg)

	// bin mode: RunJob closed binWriter; now close the backing rdb.
	if writerMode == "bin" && rdb != nil {
		rdb.Close()
	}

	cancelProgress()
	<-progressDone
	if isTTY {
		fmt.Println()
	}

	// Print summary
	if jobResult != nil && jobResult.Pass1 != nil {
		s := jobResult.Pass1
		skipped := ls.skipped.Load()
		skippedNote := ""
		if skipped > 0 {
			skippedNote = fmt.Sprintf("  skipped %s domain-killed", ccFmtInt64(skipped))
		}
		passLabel := ""
		if !args.JobCfg.NoRetry && args.JobCfg.RetryTimeout > 0 {
			passLabel = " (pass 1)"
		}
		fmt.Println(successStyle.Render(fmt.Sprintf(
			"Engine %s done%s: %s ok / %s total | avg %.0f rps | peak %.0f rps | %s%s",
			engineName, passLabel,
			ccFmtInt64(s.OK), ccFmtInt64(s.Total),
			s.AvgRPS, s.PeakRPS,
			s.Duration.Truncate(time.Second),
			skippedNote,
		)))
	}

	if jobResult != nil && jobResult.Pass2 != nil {
		s := jobResult.Pass2
		fmt.Println(successStyle.Render(fmt.Sprintf(
			"Pass 2 done: %s rescued / %s retried | avg %.0f rps | %s",
			ccFmtInt64(s.OK), ccFmtInt64(s.Total),
			s.AvgRPS, s.Duration.Truncate(time.Second),
		)))
	}

	if args.FailedDBPath != "" {
		printErrorBreakdown(args.FailedDBPath)
	}

	return err
}

// printErrorBreakdown prints a reason breakdown and top failing domains from the failed URL DB.
// It prints even when queries partially fail (graceful degradation).
func printErrorBreakdown(dbPath string) {
	summary, total, err := store.FailedURLSummary(dbPath)
	if err == nil && len(summary) > 0 {
		fmt.Println("\nError breakdown (failed URLs):")
		// Sort reasons by count descending
		type kv struct {
			reason string
			count  int
		}
		pairs := make([]kv, 0, len(summary))
		for r, c := range summary {
			pairs = append(pairs, kv{r, c})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })
		for _, p := range pairs {
			pct := float64(0)
			if total > 0 {
				pct = float64(p.count) / float64(total) * 100
			}
			fmt.Printf("  %-38s %s  (%5.1f%%)\n", p.reason, ccFmtInt64(int64(p.count)), pct)
		}
	}

	top, err := store.FailedURLTopDomains(dbPath, 10)
	if err == nil && len(top) > 0 {
		fmt.Println("\nTop failing domains:")
		for _, entry := range top {
			fmt.Printf("  %-40s %s\n", entry[0], entry[1])
		}
		fmt.Printf("  (top %d)\n", len(top))
	}
}

// ── Display types ──────────────────────────────────────────────────────────────

// v3SpeedTick records a point-in-time measurement for rolling speed calculation.
type v3SpeedTick struct {
	t     time.Time
	total int64
	bytes int64
}

// v3LiveStats tracks live statistics for the v3 progress display.
// All fields are updated atomically from the result/failure writers.
type v3LiveStats struct {
	total   atomic.Int64 // all results received (ok + fail + timeout)
	ok      atomic.Int64 // successful fetches (no error)
	failed  atomic.Int64 // hard failures (non-timeout errors)
	timeout atomic.Int64 // timeout failures
	skipped atomic.Int64 // domain-killed (domain_http_timeout_killed)
	bytes   atomic.Int64 // total body bytes received
	fetchMs atomic.Int64 // sum of FetchTimeMs for successful fetches (for avg)

	statusCodes sync.Map // int → *atomic.Int64

	// Latency histogram for adaptive timeout display (mirrors engine's tracker)
	latBuckets [8]atomic.Int64
	latTotal   atomic.Int64

	// Per-domain tracking (implements crawl.DomainNotifier)
	activeDomains sync.Map     // domain → *v3DomainInfo
	totalDomains  atomic.Int64 // domains entered via StartDomain
	doneDomains   atomic.Int64 // domains exited via EndDomain
	slowDomainMs  int64        // show domain as "slow" if active for > this ms (0=disabled)

	// Rolling speed (10-second window), protected by speedMu
	speedMu    sync.Mutex
	speedTicks []v3SpeedTick
	peakRPS    float64
	rollingRPS float64
	rollingBW  float64

	// Optional binary writer for segment stats display (nil when not in use).
	binWriter *crawl.BinSegWriter

	// Optional hardware monitor for disk/net throughput display (nil when not in use).
	hwmon *crawl.HWMonitor
}

// v3DomainInfo holds per-domain state for the slow-domain display.
type v3DomainInfo struct {
	start time.Time
	total int
}

// StartDomain implements crawl.DomainNotifier.
func (ls *v3LiveStats) StartDomain(domain string, urlCount int) {
	ls.totalDomains.Add(1)
	ls.activeDomains.Store(domain, &v3DomainInfo{start: time.Now(), total: urlCount})
}

// EndDomain implements crawl.DomainNotifier.
func (ls *v3LiveStats) EndDomain(domain string) {
	ls.activeDomains.Delete(domain)
	ls.doneDomains.Add(1)
}

func (ls *v3LiveStats) recordResult(r crawl.Result) {
	ls.total.Add(1)
	ls.bytes.Add(r.ContentLength)
	if r.StatusCode > 0 {
		v, _ := ls.statusCodes.LoadOrStore(r.StatusCode, &atomic.Int64{})
		v.(*atomic.Int64).Add(1)
	}
	switch {
	case r.Error == "":
		ls.ok.Add(1)
		ls.fetchMs.Add(r.FetchTimeMs)
		// Track latency for adaptive timeout display
		ms := r.FetchTimeMs
		ls.latTotal.Add(1)
		for i, edge := range v3LatEdges {
			if ms < edge {
				ls.latBuckets[i].Add(1)
				return
			}
		}
		ls.latBuckets[len(ls.latBuckets)-1].Add(1)
	case strings.Contains(r.Error, "timeout") || strings.Contains(r.Error, "deadline"):
		ls.timeout.Add(1)
	default:
		ls.failed.Add(1)
	}
}

func (ls *v3LiveStats) recordSkip() {
	ls.skipped.Add(1)
}

// updateSpeed refreshes rolling RPS and bandwidth (call every tick).
func (ls *v3LiveStats) updateSpeed(now time.Time) {
	tot := ls.total.Load()
	b := ls.bytes.Load()

	ls.speedMu.Lock()
	defer ls.speedMu.Unlock()

	ls.speedTicks = append(ls.speedTicks, v3SpeedTick{t: now, total: tot, bytes: b})
	cutoff := now.Add(-10 * time.Second)
	for len(ls.speedTicks) > 1 && ls.speedTicks[0].t.Before(cutoff) {
		ls.speedTicks = ls.speedTicks[1:]
	}

	var rps, bw float64
	if len(ls.speedTicks) >= 2 {
		first := ls.speedTicks[0]
		last := ls.speedTicks[len(ls.speedTicks)-1]
		dt := last.t.Sub(first.t).Seconds()
		if dt > 0 {
			rps = float64(last.total-first.total) / dt
			bw = float64(last.bytes-first.bytes) / dt
		}
	}
	ls.rollingRPS = rps
	ls.rollingBW = bw
	if rps > ls.peakRPS {
		ls.peakRPS = rps
	}
}

func (ls *v3LiveStats) p95Ms() int64 {
	n := ls.latTotal.Load()
	if n < 10 {
		return 0
	}
	target := int64(float64(n) * 0.95)
	var cum int64
	for i, edge := range v3LatEdges {
		cum += ls.latBuckets[i].Load()
		if cum >= target {
			return edge
		}
	}
	return v3LatEdges[len(v3LatEdges)-1]
}

var v3LatEdges = [8]int64{100, 250, 500, 1000, 2000, 3500, 5000, 10000}

// v3ProgressWriter wraps ResultWriter and tracks live statistics.
// When aliveCount is non-nil, every successful result increments a per-domain
// counter — used to filter pass-2 retry seeds by pass-1 success rate.
type v3ProgressWriter struct {
	inner      crawl.ResultWriter
	ls         *v3LiveStats
	aliveCount *sync.Map // optional: domain → *atomic.Int64 (pass-1 success count)
}

func (p *v3ProgressWriter) Add(r crawl.Result) {
	p.inner.Add(r)
	p.ls.recordResult(r)
	if p.aliveCount != nil && r.Error == "" {
		v, _ := p.aliveCount.LoadOrStore(r.Domain, new(atomic.Int64))
		v.(*atomic.Int64).Add(1)
	}
}
func (p *v3ProgressWriter) Flush(ctx context.Context) error { return p.inner.Flush(ctx) }
func (p *v3ProgressWriter) Close() error                    { return p.inner.Close() }

// v3ProgressFailureWriter wraps FailureWriter and counts domain-killed skips.
type v3ProgressFailureWriter struct {
	inner crawl.FailureWriter
	ls    *v3LiveStats
}

func (f *v3ProgressFailureWriter) AddURL(u crawl.FailedURL) {
	f.inner.AddURL(u)
	if u.Reason == "domain_http_timeout_killed" {
		f.ls.recordSkip()
	}
}
func (f *v3ProgressFailureWriter) Close() error { return f.inner.Close() }

// v3RenderProgress returns a formatted multi-line progress string.
func v3RenderProgress(ls *v3LiveStats, cfg crawl.Config, engineName string, seedTotal int64, start time.Time, isTTY bool) string {
	ls.speedMu.Lock()
	rollingRPS := ls.rollingRPS
	rollingBW := ls.rollingBW
	peakRPS := ls.peakRPS
	ls.speedMu.Unlock()

	tot := ls.total.Load()
	ok := ls.ok.Load()
	fail := ls.failed.Load()
	tout := ls.timeout.Load()
	skip := ls.skipped.Load()
	b := ls.bytes.Load()
	elapsed := time.Since(start)

	pct := float64(0)
	if seedTotal > 0 {
		pct = float64(tot) / float64(seedTotal) * 100
	}

	// ETA based on rolling speed
	eta := "---"
	if elapsed.Seconds() > 2 && tot > 0 {
		speed := rollingRPS
		if speed <= 0 {
			speed = float64(tot) / elapsed.Seconds()
		}
		if speed > 0 {
			remaining := seedTotal - tot
			if remaining > 0 {
				etaDur := time.Duration(float64(remaining)/speed) * time.Second
				eta = v3FmtDur(etaDur)
			} else {
				eta = "0s"
			}
		}
	}

	// Progress bar (40 chars)
	barWidth := 40
	filled := min(int(pct/100*float64(barWidth)), barWidth)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Avg fetch time
	avgMs := int64(0)
	if okN := ls.ok.Load(); okN > 0 {
		avgMs = ls.fetchMs.Load() / okN
	}

	// Avg bandwidth
	avgBW := float64(0)
	if elapsed.Seconds() > 0 {
		avgBW = float64(b) / elapsed.Seconds()
	}

	// Adaptive timeout display
	p95 := ls.p95Ms()
	adaptiveStr := ""
	if p95 > 0 {
		adapted := p95 * 2
		if adapted < 500 {
			adapted = 500
		}
		if ceil := cfg.Timeout.Milliseconds(); adapted > ceil {
			adapted = ceil
		}
		adaptiveStr = fmt.Sprintf("  Adaptive  P95=%dms  →  timeout=%dms  (ceiling %v)\n", p95, adapted, cfg.Timeout)
	}

	// HTTP status codes
	statusStr := v3StatusLine(&ls.statusCodes)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %5.1f%%  %s/%s\n",
		bar, pct, ccFmtInt64(tot), ccFmtInt64(seedTotal)))
	sb.WriteString(fmt.Sprintf("  Speed   %s/s  │  Peak %s/s  │  %s/s  │  Total %s\n",
		ccFmtInt64(int64(rollingRPS)), ccFmtInt64(int64(peakRPS)),
		v3FmtBytes(int64(rollingBW)), v3FmtBytes(b)))
	sb.WriteString(fmt.Sprintf("  ETA     %s  │  Elapsed %s  │  Avg %dms/req  │  Avg %s/s\n",
		eta, v3FmtDur(elapsed), avgMs, v3FmtBytes(int64(avgBW))))
	sb.WriteString("\n")
	done := tot + skip
	sb.WriteString(fmt.Sprintf("  ✓ %s ok (%4.1f%%)  ✗ %s fail (%4.1f%%)  ⏱ %s timeout (%4.1f%%)\n",
		ccFmtInt64(ok), v3SafePct(ok, done),
		ccFmtInt64(fail), v3SafePct(fail, done),
		ccFmtInt64(tout), v3SafePct(tout, done)))
	if skip > 0 {
		sb.WriteString(fmt.Sprintf("  ⌛ %s domain-killed (%4.1f%%)\n",
			ccFmtInt64(skip), v3SafePct(skip, done)))
	}
	// Domain progress (only shown when DomainNotifier is active)
	if totDom := ls.totalDomains.Load(); totDom > 0 {
		doneDom := ls.doneDomains.Load()
		activeDom := totDom - doneDom
		sb.WriteString(fmt.Sprintf("  Domains   total=%s  done=%s  active=%s\n",
			ccFmtInt64(totDom), ccFmtInt64(doneDom), ccFmtInt64(activeDom)))
		if ls.slowDomainMs > 0 {
			now := time.Now()
			threshold := time.Duration(ls.slowDomainMs) * time.Millisecond
			type slowEntry struct {
				domain  string
				elapsed time.Duration
				total   int
			}
			var slow []slowEntry
			ls.activeDomains.Range(func(key, val any) bool {
				if info, ok := val.(*v3DomainInfo); ok {
					if el := now.Sub(info.start); el >= threshold {
						slow = append(slow, slowEntry{key.(string), el, info.total})
					}
				}
				return true
			})
			sort.Slice(slow, func(i, j int) bool { return slow[i].elapsed > slow[j].elapsed })
			if len(slow) > 3 {
				slow = slow[:3]
			}
			if len(slow) > 0 {
				var parts []string
				for _, s := range slow {
					parts = append(parts, fmt.Sprintf("%s (%s, %d urls)", s.domain, v3FmtDur(s.elapsed), s.total))
				}
				sb.WriteString(fmt.Sprintf("  Slow      %s\n", strings.Join(parts, "  │  ")))
			}
		}
	}
	if statusStr != "" {
		sb.WriteString(fmt.Sprintf("  HTTP  %s\n", statusStr))
	}
	if adaptiveStr != "" {
		sb.WriteString(adaptiveStr)
	}
	// Memory + writer telemetry
	sb.WriteString(v3MemLine(ls.binWriter))
	// Disk + network + channel fill
	sb.WriteString(v3HWLine(ls.hwmon, ls.binWriter))
	return sb.String()
}

// v3MemLine returns a status line showing heap usage, GOMEMLIMIT, GC cycles,
// and (when w != nil) BinSegWriter segment/drain stats.
func v3MemLine(w *crawl.BinSegWriter) string {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// GOMEMLIMIT: pass -1 to read current limit without changing it.
	limitBytes := debug.SetMemoryLimit(-1)

	heapGB := float64(ms.HeapInuse) / (1 << 30)
	limitGB := float64(limitBytes) / (1 << 30)
	pct := 0.0
	if limitBytes > 0 {
		pct = float64(ms.HeapInuse) / float64(limitBytes) * 100
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  Mem   heap=%.1f GB / lim=%.1f GB (%.0f%%)  │  GC %d×",
		heapGB, limitGB, pct, ms.NumGC))
	if w != nil {
		sb.WriteString(fmt.Sprintf("  │  Writer seg=%d pend=%d drain=%s",
			w.SegCount(), w.PendingSegs(), ccFmtInt64(w.Drained())))
	}
	sb.WriteByte('\n')
	return sb.String()
}

// v3HWLine returns a status line showing disk and network throughput (MB/s)
// and the BinSegWriter channel fill level.  Returns "" when hwmon is nil.
func v3HWLine(m *crawl.HWMonitor, w *crawl.BinSegWriter) string {
	if m == nil {
		return ""
	}
	s := m.Stats()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  HW    disk rd=%.1f wr=%.1f MB/s  │  net rx=%.1f tx=%.1f MB/s",
		s.DiskReadMBps, s.DiskWriteMBps, s.NetRxMBps, s.NetTxMBps))
	if w != nil {
		sb.WriteString(fmt.Sprintf("  │  chan %.0f%%", w.ChanFill()*100))
	}
	sb.WriteByte('\n')
	return sb.String()
}

func v3StatusLine(m *sync.Map) string {
	type kv struct {
		code  int
		count int64
	}
	var pairs []kv
	m.Range(func(key, value any) bool {
		if code, ok1 := key.(int); ok1 {
			if cnt, ok2 := value.(*atomic.Int64); ok2 {
				if n := cnt.Load(); n > 0 {
					pairs = append(pairs, kv{code, n})
				}
			}
		}
		return true
	})
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })
	var parts []string
	for i, p := range pairs {
		if i >= 8 {
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%s", p.code, ccFmtInt64(p.count)))
	}
	return strings.Join(parts, "  ")
}

func v3SafePct(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

func v3FmtBytes(b int64) string {
	if b < 0 {
		return "0 B"
	}
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	case b < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(b)/(1024*1024*1024))
	}
}

func v3FmtDur(d time.Duration) string {
	d = d.Truncate(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
