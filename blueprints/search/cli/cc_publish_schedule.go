package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/arctic"
)

// ccScheduleConfig holds configuration for the CC pipeline scheduler.
type ccScheduleConfig struct {
	CrawlID     string
	RepoRoot    string
	Start       int
	End         int
	MaxSessions int // 0 = auto-detect from hardware
	ChunkSize   int
	DonePct     int
	StallRounds int
	SearchBin   string
	// GapIndices, when non-nil, enables gap mode: only these specific shard
	// indices are targeted. Chunks are built from clusters of gap indices and
	// done-pct is evaluated against gap targets only (not the full range).
	GapIndices []int
}

type ccSchedChunk struct {
	Start, End int
}

// ccBudget holds the computed resource budget for CC pipeline sessions.
type ccBudget struct {
	MaxSessions   int     // max concurrent screen sessions
	RAMPerSession float64 // estimated GB per session (0.9)
	ByRAM         int     // limit from RAM
	ByCPU         int     // limit from CPU
	Auto          bool    // true if computed from hardware
}

func (b ccBudget) String() string {
	if !b.Auto {
		return fmt.Sprintf("%d max (manual override)", b.MaxSessions)
	}
	return fmt.Sprintf("%d max (auto: ram=%d cpu=%d → %d)", b.MaxSessions, b.ByRAM, b.ByCPU, b.MaxSessions)
}

// computeCCBudget derives the session budget from hardware profile.
//
// Each pipeline session (download → pack → export) peaks at ~2–5 GB observed
// during WARC parallel offset scanning. Budget 2.5 GB per session.
// Use available RAM (not total) since background services (lnx, quickwit,
// llama-server, chrome, docker) consume significant memory.
// Hard cap at 6 sessions (diminishing returns from disk I/O contention).
func computeCCBudget(hw arctic.HardwareProfile) ccBudget {
	const (
		perSessionGB = 2.5
		maxCap       = 6
	)

	b := ccBudget{
		RAMPerSession: perSessionGB,
		Auto:          true,
	}

	// By RAM: use available RAM (accounts for background services dynamically).
	// Available RAM already excludes OS caches / other processes.
	usableRAM := hw.RAMAvailGB
	if usableRAM < perSessionGB {
		usableRAM = perSessionGB
	}
	b.ByRAM = int(usableRAM / perSessionGB)

	// By CPU: each session is a mix of I/O-bound and CPU-bound work.
	// Allow 1.5 sessions per core (pack is I/O heavy, export is CPU heavy).
	b.ByCPU = int(float64(hw.CPUCores) * 1.5)
	if b.ByCPU < 1 {
		b.ByCPU = 1
	}

	// Take the minimum, clamp to [1, maxCap].
	b.MaxSessions = b.ByRAM
	if b.ByCPU < b.MaxSessions {
		b.MaxSessions = b.ByCPU
	}
	if b.MaxSessions < 1 {
		b.MaxSessions = 1
	}
	if b.MaxSessions > maxCap {
		b.MaxSessions = maxCap
	}

	// Safety: on machines with < 16 GB total, never exceed 3 sessions regardless
	// of what available RAM says (background services can release cache temporarily,
	// creating a false sense of headroom that vanishes under load).
	if hw.RAMTotalGB < 16 && b.MaxSessions > 3 {
		b.MaxSessions = 3
	}

	// Environment override.
	if v := os.Getenv("MIZU_CC_MAX_SESSIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			b.MaxSessions = n
			b.Auto = false
		}
	}

	return b
}

// ccResourceTracker tracks resource usage and throughput across scheduler rounds
// to identify bottlenecks and make smarter scaling decisions.
type ccResourceTracker struct {
	// Sliding window of (round, committedCount) for throughput detection.
	history      []ccRoundSnapshot
	maxHistory   int
	prevDiskFree float64 // disk free GB from previous round (detect disk fill rate)
}

type ccRoundSnapshot struct {
	round     int
	committed int
	running   int
	ramAvail  float64
	loadAvg   float64
	diskFree  float64
}

// throughputPerSession returns the average commits per round per running session
// over the last N rounds. Returns 0 if not enough data.
func (t *ccResourceTracker) throughputPerSession() float64 {
	if len(t.history) < 3 {
		return 0
	}
	oldest := t.history[0]
	newest := t.history[len(t.history)-1]
	rounds := newest.round - oldest.round
	if rounds < 2 {
		return 0
	}
	totalCommits := float64(newest.committed - oldest.committed)
	// Average running sessions over the window.
	avgRunning := 0.0
	for _, h := range t.history {
		avgRunning += float64(h.running)
	}
	avgRunning /= float64(len(t.history))
	if avgRunning < 0.5 {
		return 0
	}
	return totalCommits / float64(rounds) / avgRunning
}

// isBottlenecked returns true if adding sessions isn't helping throughput.
// Compares recent throughput-per-session to earlier throughput-per-session.
func (t *ccResourceTracker) isBottlenecked() bool {
	if len(t.history) < 10 {
		return false
	}
	// Compare first half vs second half of history.
	mid := len(t.history) / 2
	firstHalf := t.history[:mid]
	secondHalf := t.history[mid:]

	// Throughput per session in each half.
	tps := func(slice []ccRoundSnapshot) float64 {
		if len(slice) < 2 {
			return 0
		}
		commits := float64(slice[len(slice)-1].committed - slice[0].committed)
		rounds := float64(slice[len(slice)-1].round - slice[0].round)
		avgRun := 0.0
		for _, s := range slice {
			avgRun += float64(s.running)
		}
		avgRun /= float64(len(slice))
		if rounds < 1 || avgRun < 0.5 {
			return 0
		}
		return commits / rounds / avgRun
	}

	firstTPS := tps(firstHalf)
	secondTPS := tps(secondHalf)

	// If second half has more sessions but same/less throughput per session,
	// we're bottlenecked (likely disk I/O or network).
	avgRunFirst := 0.0
	for _, s := range firstHalf {
		avgRunFirst += float64(s.running)
	}
	avgRunFirst /= float64(len(firstHalf))
	avgRunSecond := 0.0
	for _, s := range secondHalf {
		avgRunSecond += float64(s.running)
	}
	avgRunSecond /= float64(len(secondHalf))

	// More sessions but throughput per session dropped > 30%.
	if avgRunSecond > avgRunFirst+0.5 && firstTPS > 0 && secondTPS < firstTPS*0.7 {
		return true
	}
	return false
}

// diskFillRate returns GB/round being consumed (positive = disk filling, negative = freeing).
func (t *ccResourceTracker) diskFillRate() float64 {
	if len(t.history) < 3 {
		return 0
	}
	oldest := t.history[0]
	newest := t.history[len(t.history)-1]
	rounds := float64(newest.round - oldest.round)
	if rounds < 1 {
		return 0
	}
	return (oldest.diskFree - newest.diskFree) / rounds
}

func (t *ccResourceTracker) record(snap ccRoundSnapshot) {
	t.history = append(t.history, snap)
	if t.maxHistory == 0 {
		t.maxHistory = 30 // ~1 hour of 2-min rounds
	}
	if len(t.history) > t.maxHistory {
		t.history = t.history[len(t.history)-t.maxHistory:]
	}
}

// dynamicMaxSessions adjusts the effective max sessions based on live resource usage.
// It accounts for ALL processes on the system (not just CC pipeline sessions),
// tracks throughput trends, and identifies bottlenecks.
// Returns the effective max and a reason string for logging.
func dynamicMaxSessions(hw arctic.HardwareProfile, initialMax, nRunning int, tracker *ccResourceTracker) (int, string) {
	effective := initialMax
	var reasons []string

	// --- Critical: immediate danger ---

	// RAM critically low: other processes (k8s, arctic, hn) may be competing.
	if hw.RAMAvailGB < 0.3 {
		// Emergency: try to get to nRunning-2 (kill at least 2 slots worth of headroom).
		effective = nRunning - 2
		if effective < 0 {
			effective = 0
		}
		reasons = append(reasons, fmt.Sprintf("CRITICAL ram=%.0fMB: max→%d", hw.RAMAvailGB*1024, effective))
		return effective, strings.Join(reasons, ", ")
	}

	// --- Pressure: approaching limits ---

	// RAM under pressure (< 500 MB available).
	if hw.RAMAvailGB < 0.5 {
		effective = nRunning - 1
		if effective < 1 {
			effective = 1
		}
		reasons = append(reasons, fmt.Sprintf("ram_low=%.0fMB", hw.RAMAvailGB*1024))
	}

	// RAM moderately low (< 1 GB available). Don't grow.
	if hw.RAMAvailGB < 1.0 && effective > nRunning {
		effective = nRunning
		reasons = append(reasons, fmt.Sprintf("ram_tight=%.1fGB: hold", hw.RAMAvailGB))
	}

	// Load average too high: other processes are contending.
	loadAvg := readLoadAvg1()
	if loadAvg > 0 {
		overloadThreshold := float64(hw.CPUCores) * 3
		highThreshold := float64(hw.CPUCores) * 2

		if loadAvg > overloadThreshold {
			// Severely overloaded — shed load proportionally.
			reduction := int((loadAvg - overloadThreshold) / float64(hw.CPUCores))
			if reduction < 1 {
				reduction = 1
			}
			newMax := nRunning - reduction
			if newMax < 1 {
				newMax = 1
			}
			if newMax < effective {
				effective = newMax
			}
			reasons = append(reasons, fmt.Sprintf("load=%.1f>%d×3: -%d", loadAvg, hw.CPUCores, reduction))
		} else if loadAvg > highThreshold && effective > nRunning {
			// High load — don't grow.
			effective = nRunning
			reasons = append(reasons, fmt.Sprintf("load=%.1f: hold", loadAvg))
		}
	}

	// Disk critically low.
	if hw.DiskFreeGB < 20 {
		effective = 0
		reasons = append(reasons, fmt.Sprintf("disk_critical=%.0fGB: pause", hw.DiskFreeGB))
	} else if hw.DiskFreeGB < 50 {
		if effective > nRunning-1 {
			effective = nRunning - 1
			if effective < 1 {
				effective = 1
			}
		}
		reasons = append(reasons, fmt.Sprintf("disk_low=%.0fGB", hw.DiskFreeGB))
	}

	// Disk filling fast: if we'll hit 20GB within 10 rounds (~20 min), throttle.
	if fillRate := tracker.diskFillRate(); fillRate > 0 && hw.DiskFreeGB > 20 {
		roundsUntilFull := (hw.DiskFreeGB - 20) / fillRate
		if roundsUntilFull < 10 {
			if effective > nRunning-1 {
				effective = nRunning - 1
				if effective < 1 {
					effective = 1
				}
			}
			reasons = append(reasons, fmt.Sprintf("disk_fill=%.1fGB/round: ~%.0f rounds to 20GB", fillRate, roundsUntilFull))
		}
	}

	// --- Bottleneck detection: adding sessions doesn't help ---
	if tracker.isBottlenecked() && effective > nRunning {
		effective = nRunning
		reasons = append(reasons, "bottleneck: more sessions not helping, hold")
	}

	// --- Relaxed: can grow ---
	if len(reasons) == 0 && effective < initialMax {
		// All clear but we were previously throttled — allow one step up.
		effective = nRunning + 1
		if effective > initialMax {
			effective = initialMax
		}
		reasons = append(reasons, "relaxed: +1")
	}

	if len(reasons) == 0 {
		return effective, "ok"
	}
	return effective, strings.Join(reasons, ", ")
}

// readLoadAvg1 reads the 1-minute load average from /proc/loadavg (Linux).
// Returns 0 on non-Linux or any error.
func readLoadAvg1() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return v
}

// ccCleanupLeftovers removes files that are no longer needed: committed WARCs,
// orphaned .tmp files, and stale sidecars. Returns bytes freed and file count.
func ccCleanupLeftovers(crawlID string, committed map[int]struct{}, logFn func(string)) (freed int64, count int) {
	home, _ := os.UserHomeDir()
	warcDir := filepath.Join(home, "data", "common-crawl", crawlID, "warc")
	warcMdDir := filepath.Join(home, "data", "common-crawl", crawlID, "warc_md")
	repoRoot := ccDefaultExportRepoRoot(crawlID)
	dataDir := filepath.Join(repoRoot, "data", crawlID)

	rm := func(path string) {
		fi, err := os.Stat(path)
		if err != nil {
			return
		}
		sz := fi.Size()
		if err := os.Remove(path); err != nil {
			return
		}
		freed += sz
		count++
	}

	// 1. Orphaned .tmp files — always safe to remove (crashed sessions).
	if entries, err := os.ReadDir(warcMdDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".tmp") {
				rm(filepath.Join(warcMdDir, e.Name()))
			}
		}
	}

	// 2. For committed shards: remove raw WARC, packed WARC, sidecars.
	for idx := range committed {
		shard := fmt.Sprintf("%05d", idx)

		// Raw WARC: might have various names, use the sidecar to find it.
		sidecarPath := filepath.Join(warcMdDir, shard+".warc.path")
		if data, err := os.ReadFile(sidecarPath); err == nil {
			rawPath := strings.TrimSpace(string(data))
			if rawPath != "" {
				rm(rawPath)
			}
		}
		// Also try legacy glob pattern.
		pattern := filepath.Join(warcDir, fmt.Sprintf("*-%05d.warc.gz", idx))
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			for _, m := range matches {
				rm(m)
			}
		}

		// Packed md.warc.gz.
		rm(filepath.Join(warcMdDir, shard+".md.warc.gz"))
		// Sidecars.
		rm(filepath.Join(warcMdDir, shard+".warc.path"))
		// Meta files.
		rm(filepath.Join(dataDir, shard+".meta"))

		// Parquet files that are committed AND older than 10 minutes
		// (safety: watcher may still be reading them for upload).
		pqPath := filepath.Join(dataDir, shard+".parquet")
		if fi, err := os.Stat(pqPath); err == nil {
			if time.Since(fi.ModTime()) > 10*time.Minute {
				rm(pqPath)
			}
		}
	}

	if count > 0 && logFn != nil {
		logFn(fmt.Sprintf("  cleanup: freed %s (%d files)", ccFmtBytes(freed), count))
	}

	return freed, count
}

// runCCScheduleLoop wraps runCCSchedule with an auto-restart loop so the
// scheduler heals itself if it crashes (network errors, transient failures,
// OOM kills of child processes, or even signal-induced context cancellation).
//
// The scheduler is a long-running daemon in a detached screen session — it
// must survive everything short of SIGKILL to itself. SIGTERM from OOM or
// stray signals should not cause permanent death.
//
// Returns only when all chunks are done (err == nil).
func runCCScheduleLoop(ctx context.Context, cfg ccScheduleConfig) error {
	restartDelay := 10 * time.Second
	attempt := 0
	for {
		attempt++
		if attempt > 1 {
			fmt.Printf("  [schedule] restart attempt %d\n", attempt)
		}

		// Use a fresh, independent context for each attempt. The scheduler
		// runs in a detached screen — there is no interactive user. Parent
		// context cancellation (from SIGTERM) must not propagate because
		// once canceled, ctx.Done() stays closed and would kill every
		// subsequent attempt immediately.
		attemptCtx, attemptCancel := context.WithCancel(context.Background())
		err := runCCSchedule(attemptCtx, cfg)
		attemptCancel()

		if err == nil {
			return nil // all chunks done
		}

		fmt.Printf("  [schedule] crashed: %v — restarting in %s\n", err, restartDelay)

		// Re-register signal handling: the previous signal consumed the
		// notification, so we need to be ready for the next one.
		// Sleep uses a simple timer (not the parent context) so we
		// always wake up and retry.
		time.Sleep(restartDelay)
	}
}

// runCCSchedule drives CC pipeline screen sessions to cover [start, end].
// It reads stats.csv every 2 minutes, detects stalled sessions, restarts
// them, and fills free slots with new sessions until all chunks are done.
//
// When MaxSessions == 0 (auto), hardware is detected at startup and the budget
// is computed dynamically each round based on live RAM, CPU load, and disk usage.
func runCCSchedule(ctx context.Context, cfg ccScheduleConfig) error {
	searchBin := cfg.resolveSearchBin()
	chunks := cfg.buildChunks()

	// Open append-only log file.
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, "log")
	_ = os.MkdirAll(logDir, 0o755)
	logPath := filepath.Join(logDir, fmt.Sprintf("cc_schedule_%d_%d.log", cfg.Start, cfg.End))
	logF, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if logF != nil {
		defer logF.Close()
	}

	logLine := func(msg string) {
		line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
		fmt.Print(line)
		if logF != nil {
			_, _ = logF.WriteString(line)
		}
	}

	// ── Hardware detection and budget ────────────────────────────────────────
	autoMode := cfg.MaxSessions == 0
	var budget ccBudget
	var hw arctic.HardwareProfile

	if autoMode {
		hw = arctic.DetectHardware(cfg.RepoRoot)
		budget = computeCCBudget(hw)
		cfg.MaxSessions = budget.MaxSessions
	} else {
		budget = ccBudget{MaxSessions: cfg.MaxSessions, Auto: false}
	}
	initialMax := cfg.MaxSessions

	statsCSV := ccStatsCSVPath(cfg.RepoRoot)

	// Build gap set for gap-aware progress tracking.
	var gapSet map[int]bool
	if len(cfg.GapIndices) > 0 {
		gapSet = make(map[int]bool, len(cfg.GapIndices))
		for _, idx := range cfg.GapIndices {
			gapSet[idx] = true
		}
	}

	logLine("=== CC Schedule starting ===")
	logLine(fmt.Sprintf("  Crawl:       %s", cfg.CrawlID))
	if gapSet != nil {
		logLine(fmt.Sprintf("  Mode:        gap backfill (%d uncommitted shards in %d\u2013%d)",
			len(cfg.GapIndices), cfg.Start, cfg.End))
	} else {
		logLine(fmt.Sprintf("  Range:       %d\u2013%d", cfg.Start, cfg.End))
	}
	logLine(fmt.Sprintf("  Chunks:      %d  (size=%d)", len(chunks), cfg.ChunkSize))
	if autoMode {
		logLine(fmt.Sprintf("  Hardware:    %s", hw))
		logLine(fmt.Sprintf("  Budget:      %s", budget))
	} else {
		logLine(fmt.Sprintf("  Sessions:    %d max (manual)", cfg.MaxSessions))
	}
	logLine(fmt.Sprintf("  Done pct:    %d%%", cfg.DonePct))
	logLine(fmt.Sprintf("  Stall kill:  after %d rounds (~%dm) with no new commits", cfg.StallRounds, cfg.StallRounds*2))
	logLine(fmt.Sprintf("  Binary:      %s", searchBin))

	// ── Initial cleanup ──────────────────────────────────────────────────────
	committed := ccSchedReadCommitted(statsCSV, cfg.CrawlID)
	ccCleanupLeftovers(cfg.CrawlID, committed, logLine)
	logLine("")

	// Per-chunk stall tracking: last committed count + consecutive stall rounds.
	type stallState struct {
		lastCommitted int
		rounds        int
	}
	stall := make(map[ccSchedChunk]*stallState, len(chunks))
	for _, c := range chunks {
		stall[c] = &stallState{}
	}

	// Resource tracker for adaptive throughput/bottleneck detection.
	tracker := &ccResourceTracker{}

	round := 0
	for {
		round++

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Re-detect hardware for dynamic scaling (available RAM, disk change over time).
		if autoMode {
			hw = arctic.DetectHardware(cfg.RepoRoot)
		}

		// Read committed set.
		committed = ccSchedReadCommitted(statsCSV, cfg.CrawlID)

		// Periodic cleanup (every 5 rounds = ~10 min).
		if round%5 == 0 {
			ccCleanupLeftovers(cfg.CrawlID, committed, logLine)
		}

		var runningNames, todoKeys []string
		var runningChunks []ccSchedChunk
		nRunning, nDone := 0, 0

		for _, chunk := range chunks {
			nComm, total := ccSchedChunkProgress(committed, gapSet, chunk.Start, chunk.End)
			running := ccSchedChunkRunning(chunk.Start, chunk.End)

			if running {
				ss := stall[chunk]
				if nComm > ss.lastCommitted {
					ss.lastCommitted = nComm
					ss.rounds = 0
				} else {
					ss.rounds++
				}
				if ss.rounds >= cfg.StallRounds {
					logLine(fmt.Sprintf("  STALL: g%d_%d stuck at %d/%d for %d rounds — killing and restarting",
						chunk.Start, chunk.End, nComm, total, ss.rounds))
					ccSchedKillChunk(chunk.Start, chunk.End)
					ss.rounds = 0
					todoKeys = append(todoKeys, fmt.Sprintf("%d:%d", chunk.Start, chunk.End))
				} else {
					nRunning++
					runningChunks = append(runningChunks, chunk)
					label := fmt.Sprintf("g%d_%d(%d/%d", chunk.Start, chunk.End, nComm, total)
					if ss.rounds > 0 {
						label += fmt.Sprintf(" stall=%d", ss.rounds)
					}
					label += ")"
					runningNames = append(runningNames, label)
				}
			} else {
				stall[chunk].rounds = 0
				pct := nComm * 100 / total
				if pct >= cfg.DonePct {
					nDone++
				} else {
					todoKeys = append(todoKeys, fmt.Sprintf("%d:%d", chunk.Start, chunk.End))
				}
			}
		}

		totalCommitted := len(committed)

		// Record snapshot for adaptive tracking.
		loadAvg := readLoadAvg1()
		tracker.record(ccRoundSnapshot{
			round:     round,
			committed: totalCommitted,
			running:   nRunning,
			ramAvail:  hw.RAMAvailGB,
			loadAvg:   loadAvg,
			diskFree:  hw.DiskFreeGB,
		})

		// Dynamic scaling: adjust effective max based on live resources + throughput trends.
		effectiveMax := cfg.MaxSessions
		reason := "ok"
		if autoMode {
			effectiveMax, reason = dynamicMaxSessions(hw, initialMax, nRunning, tracker)
		}

		nTodo := len(todoKeys)
		slots := effectiveMax - nRunning

		// Log resource state with throughput metrics.
		if autoMode {
			tps := tracker.throughputPerSession()
			fillRate := tracker.diskFillRate()
			logLine(fmt.Sprintf("Round %d | hw: %.1fGB RAM (%.1f avail), %.0fGB disk (%.0f free), load %.1f/%d cores",
				round, hw.RAMTotalGB, hw.RAMAvailGB, hw.DiskTotalGB, hw.DiskFreeGB, loadAvg, hw.CPUCores))
			throughputInfo := fmt.Sprintf("tps=%.2f commits/round/sess", tps)
			if fillRate > 0.01 {
				throughputInfo += fmt.Sprintf(", disk_fill=%.1fGB/round", fillRate)
			}
			logLine(fmt.Sprintf("         | budget: %s, effective: %d (%s)",
				budget, effectiveMax, reason))
			logLine(fmt.Sprintf("         | %s", throughputInfo))
		}

		logLine(fmt.Sprintf("Round %d | committed=%d | done=%d/%d chunks | running=%d | todo=%d | slots=%d",
			round, totalCommitted, nDone, len(chunks), nRunning, nTodo, slots))
		if len(runningNames) > 0 {
			logLine("  running: " + strings.Join(runningNames, " "))
		}

		// If we need to shed sessions (effective max < running), kill the most stalled.
		if effectiveMax < nRunning && effectiveMax >= 0 {
			toKill := nRunning - effectiveMax
			// Sort running chunks by stall count descending — kill the most stalled first.
			type stallEntry struct {
				chunk ccSchedChunk
				stall int
			}
			var candidates []stallEntry
			for _, c := range runningChunks {
				candidates = append(candidates, stallEntry{c, stall[c].rounds})
			}
			// Simple selection sort for small N.
			for i := 0; i < len(candidates)-1; i++ {
				for j := i + 1; j < len(candidates); j++ {
					if candidates[j].stall > candidates[i].stall {
						candidates[i], candidates[j] = candidates[j], candidates[i]
					}
				}
			}
			killed := 0
			for _, c := range candidates {
				if killed >= toKill {
					break
				}
				logLine(fmt.Sprintf("  SHED: killing g%d_%d (stall=%d) to free resources",
					c.chunk.Start, c.chunk.End, c.stall))
				ccSchedKillChunk(c.chunk.Start, c.chunk.End)
				stall[c.chunk].rounds = 0
				killed++
			}
		}

		if nRunning == 0 && nTodo == 0 {
			logLine("")
			if gapSet != nil {
				logLine(fmt.Sprintf("=== Gap backfill complete: %d shards filled in %d\u2013%d ===",
					len(cfg.GapIndices), cfg.Start, cfg.End))
			} else {
				logLine(fmt.Sprintf("=== All chunks complete for range %d\u2013%d ===", cfg.Start, cfg.End))
			}
			logLine(fmt.Sprintf("Total committed: %d", totalCommitted))
			logLine(fmt.Sprintf("Run: search cc publish --list"))
			// Final cleanup.
			ccCleanupLeftovers(cfg.CrawlID, committed, logLine)
			return nil
		}

		// Ramp up gradually: max 2 new sessions per round to avoid
		// spiking load/memory when many slots open at once.
		const maxStartPerRound = 2
		started := 0
		for _, key := range todoKeys {
			if slots <= 0 || started >= maxStartPerRound {
				break
			}
			var s, e int
			fmt.Sscanf(key, "%d:%d", &s, &e)
			ccSchedStartChunk(s, e, searchBin)
			logLine(fmt.Sprintf("  started g%d_%d  (files %d\u2013%d)", s, e, s, e))
			slots--
			started++
		}
		if started > 0 {
			logLine(fmt.Sprintf("  launched %d new session(s)", started))
		}
		logLine("")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Minute):
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (cfg ccScheduleConfig) resolveSearchBin() string {
	if cfg.SearchBin != "" {
		return cfg.SearchBin
	}
	home, _ := os.UserHomeDir()
	for _, c := range []string{
		filepath.Join(home, "bin", "search"),
		"/usr/local/bin/search",
	} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "search"
}

func (cfg ccScheduleConfig) buildChunks() []ccSchedChunk {
	if len(cfg.GapIndices) > 0 {
		return ccBuildGapChunks(cfg.GapIndices, cfg.ChunkSize)
	}
	var chunks []ccSchedChunk
	for s := cfg.Start; s <= cfg.End; s += cfg.ChunkSize {
		e := s + cfg.ChunkSize - 1
		if e > cfg.End {
			e = cfg.End
		}
		chunks = append(chunks, ccSchedChunk{s, e})
	}
	return chunks
}

// ccBuildGapChunks groups sparse gap indices into [lo, hi] range chunks.
// Each chunk covers at most chunkSize gap indices; the range spans [first_gap, last_gap].
// Pipelines running over these ranges will skip committed shards naturally.
func ccBuildGapChunks(gaps []int, chunkSize int) []ccSchedChunk {
	if len(gaps) == 0 {
		return nil
	}
	var chunks []ccSchedChunk
	for i := 0; i < len(gaps); i += chunkSize {
		end := i + chunkSize - 1
		if end >= len(gaps) {
			end = len(gaps) - 1
		}
		chunks = append(chunks, ccSchedChunk{Start: gaps[i], End: gaps[end]})
	}
	return chunks
}

// ccSchedChunkProgress returns (nCommitted, nTotal) for done/stall tracking.
// In gap mode (gapSet non-nil), counts only gap-target indices in [start, end].
// In normal mode, counts all indices in the range.
func ccSchedChunkProgress(committed map[int]struct{}, gapSet map[int]bool, start, end int) (nComm, total int) {
	if gapSet != nil {
		for i := start; i <= end; i++ {
			if gapSet[i] {
				total++
				if _, ok := committed[i]; ok {
					nComm++
				}
			}
		}
		return
	}
	total = end - start + 1
	nComm = ccSchedCountRange(committed, start, end)
	return
}

// ccSchedReadCommitted returns a set of committed file indices for the crawl.
// Never returns an error — callers get an empty set on any failure.
func ccSchedReadCommitted(statsCSV, crawlID string) map[int]struct{} {
	stats, _ := ccReadStatsCSV(statsCSV)
	out := make(map[int]struct{}, len(stats))
	for _, s := range stats {
		if s.CrawlID == crawlID {
			out[s.FileIdx] = struct{}{}
		}
	}
	return out
}

// ccSchedCountRange counts committed indices in [start, end].
func ccSchedCountRange(committed map[int]struct{}, start, end int) int {
	n := 0
	for i := start; i <= end; i++ {
		if _, ok := committed[i]; ok {
			n++
		}
	}
	return n
}

// ccSchedChunkRunning returns true if a pipeline process for this chunk is running.
func ccSchedChunkRunning(start, end int) bool {
	pattern := fmt.Sprintf("publish.*--file %d-%d$", start, end)
	return exec.Command("pgrep", "-f", pattern).Run() == nil
}

// ccSchedKillChunk kills all pipeline processes for a chunk.
func ccSchedKillChunk(start, end int) {
	pattern := fmt.Sprintf("publish.*--file %d-%d$", start, end)
	_ = exec.Command("pkill", "-9", "-f", pattern).Run()
}

// ccSchedStartChunk launches a new screen session for a pipeline chunk.
// Logs to stderr if screen fails to start (common when screen is not installed
// or the session name conflicts).
func ccSchedStartChunk(start, end int, searchBin string) {
	name := fmt.Sprintf("g%d_%d", start, end)
	home, _ := os.UserHomeDir()
	pathPrefix := filepath.Join(home, "bin")
	cmdStr := fmt.Sprintf(
		"export PATH=%s:$PATH; %s cc publish --pipeline --cleanup --skip-errors --file %d-%d",
		pathPrefix, searchBin, start, end,
	)
	// Kill any lingering session with same name before creating a new one.
	_ = exec.Command("screen", "-S", name, "-X", "quit").Run()
	time.Sleep(500 * time.Millisecond)
	if err := exec.Command("screen", "-dmS", name, "bash", "-c", cmdStr+"; exec bash").Run(); err != nil {
		fmt.Printf("  [schedule] ERROR: failed to start screen session %s: %v\n", name, err)
	}
}
