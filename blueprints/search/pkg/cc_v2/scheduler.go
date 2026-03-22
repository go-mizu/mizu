package cc_v2

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

// Scheduler manages pipeline screen sessions across a shard range.
// It auto-heals crashed sessions, detects stalls, and scales sessions
// based on available resources.
type Scheduler struct {
	cfg   SchedulerConfig
	store Store
	log   *Logger
}

// NewScheduler creates a scheduler.
func NewScheduler(cfg SchedulerConfig, store Store) *Scheduler {
	return &Scheduler{
		cfg:   cfg,
		store: store,
		log:   NewLogger("scheduler", store),
	}
}

type schedChunk struct {
	Start, End int
}

type stallState struct {
	lastCommitted int
	rounds        int
}

// Run starts the scheduler loop with auto-restart on crash.
func (s *Scheduler) Run(ctx context.Context) error {
	for attempt := 1; ; attempt++ {
		if attempt > 1 {
			s.log.Info("scheduler restart", "attempt", attempt)
		}
		// Use fresh context for each attempt (screen session survives signals).
		attemptCtx, cancel := context.WithCancel(context.Background())
		err := s.run(attemptCtx)
		cancel()
		if err == nil {
			return nil // all chunks done
		}
		s.log.Error("scheduler crashed", "err", err)
		time.Sleep(10 * time.Second)
	}
}

func (s *Scheduler) run(ctx context.Context) error {
	searchBin := s.resolveSearchBin()
	chunks := s.buildChunks()
	if len(chunks) == 0 {
		s.log.Info("no chunks to process")
		return nil
	}

	// Hardware detection and budget.
	hw := arctic.DetectHardware(s.cfg.RepoRoot)
	budget := s.computeBudget(hw)

	// Defaults.
	if s.cfg.DonePct == 0 {
		s.cfg.DonePct = 95
	}
	if s.cfg.StallRounds == 0 {
		s.cfg.StallRounds = 40
	}

	s.log.PrintBanner("Scheduler", map[string]string{
		"Crawl":    s.cfg.CrawlID,
		"Range":    fmt.Sprintf("%d–%d", s.cfg.Start, s.cfg.End),
		"Chunks":   fmt.Sprintf("%d", len(chunks)),
		"Sessions": fmt.Sprintf("%d max (%s)", budget, s.budgetDetail(hw, budget)),
		"Binary":   searchBin,
		"Redis":    fmt.Sprintf("%v", s.store.Available()),
	})

	// Per-chunk stall tracking.
	stall := make(map[schedChunk]*stallState, len(chunks))
	for _, c := range chunks {
		stall[c] = &stallState{}
	}

	// Track previous values for deltas.
	lastSeenCommit := 0
	if ws, ok := s.store.GetWatcherStatus(ctx); ok {
		lastSeenCommit = ws.CommitNum
	}
	prevCommitted := 0
	prevDlRate, prevPackRate, prevCommitRate := 0.0, 0.0, 0.0

	round := 0
	for {
		round++
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Refresh hardware for dynamic scaling.
		hw = arctic.DetectHardware(s.cfg.RepoRoot)
		budget = s.computeBudget(hw)

		committed := s.store.CommittedSet(ctx)
		if committed == nil {
			committed = make(map[int]bool)
		}

		var running, done, todo int
		var runningNames []string

		for _, chunk := range chunks {
			nComm, total := s.chunkProgress(committed, chunk)
			isRunning := s.chunkRunning(chunk)

			if isRunning {
				ss := stall[chunk]
				if nComm > ss.lastCommitted {
					ss.lastCommitted = nComm
					ss.rounds = 0
				} else {
					ss.rounds++
				}
				if ss.rounds >= s.cfg.StallRounds {
					name := fmt.Sprintf("g%d_%d", chunk.Start, chunk.End)
					fmt.Fprintf(os.Stderr, "  KILL %s (stalled %d rounds at %d/%d)\n",
						name, ss.rounds, nComm, total)
					s.killChunk(chunk)
					ss.rounds = 0
					todo++
				} else {
					running++
					label := fmt.Sprintf("g%d_%d(%d/%d", chunk.Start, chunk.End, nComm, total)
					if ss.rounds > 0 {
						label += fmt.Sprintf(" stall=%d", ss.rounds)
					}
					label += ")"
					runningNames = append(runningNames, label)
				}
			} else {
				stall[chunk].rounds = 0
				pct := 0
				if total > 0 {
					pct = nComm * 100 / total
				}
				if pct >= s.cfg.DonePct {
					done++
				} else {
					todo++
				}
			}
		}

		totalCommitted := s.store.CommittedCount(ctx)

		// Rates from Redis.
		var dlRate, packRate, commitRate float64
		if s.store.Available() {
			dlRate = s.store.EventRate(ctx, "downloaded", 15*time.Minute)
			packRate = s.store.EventRate(ctx, "packed", 15*time.Minute)
			commitRate = s.store.EventRate(ctx, "committed", 15*time.Minute)
			if round%10 == 0 {
				s.store.TrimRates(ctx)
			}
		}

		// Watcher status.
		ws, hasWS := s.store.GetWatcherStatus(ctx)
		if hasWS && ws.CommitNum > lastSeenCommit {
			lastSeenCommit = ws.CommitNum
		}

		slots := budget - running
		if slots < 0 {
			slots = 0
		}

		// ── Compact round log ────────────────────────────────────────
		remaining := 100_000 - totalCommitted
		delta := totalCommitted - prevCommitted
		pct := float64(totalCommitted) * 100 / 100_000

		fmt.Fprintf(os.Stderr, "\n  Round %-4d  %s  sessions %d/%d  ram %.1f/%.1fGB\n",
			round, time.Now().Format("15:04:05"), running, budget, hw.RAMAvailGB, hw.RAMTotalGB)
		fmt.Fprintf(os.Stderr, "    committed  %d (%.1f%%)  remaining %d",
			totalCommitted, pct, remaining)
		if delta > 0 {
			fmt.Fprintf(os.Stderr, "  (+%d)", delta)
		}
		fmt.Fprintln(os.Stderr)

		// Rates with deltas.
		fmt.Fprintf(os.Stderr, "    rate/hr    dl %-4.0f%s  pack %-4.0f%s  commit %-4.0f%s\n",
			dlRate, rateDelta(dlRate, prevDlRate),
			packRate, rateDelta(packRate, prevPackRate),
			commitRate, rateDelta(commitRate, prevCommitRate))

		// ETA.
		if commitRate > 0 {
			etaH := float64(remaining) / commitRate
			if etaH < 48 {
				fmt.Fprintf(os.Stderr, "    ETA        %.1fh\n", etaH)
			} else {
				fmt.Fprintf(os.Stderr, "    ETA        %.0fd\n", etaH/24)
			}
		}

		// Watcher last commit.
		if hasWS && ws.CommitNum > 0 {
			ago := time.Since(ws.Timestamp).Round(time.Second)
			fmt.Fprintf(os.Stderr, "    watcher    #%d  %s  (%s ago)\n",
				ws.CommitNum, ws.Message, ago)
		}

		// Active sessions (only if > 0).
		if len(runningNames) > 0 {
			fmt.Fprintf(os.Stderr, "    active     %s\n", strings.Join(runningNames, " "))
		}

		prevCommitted = totalCommitted
		prevDlRate, prevPackRate, prevCommitRate = dlRate, packRate, commitRate

		// All done?
		if running == 0 && todo == 0 {
			fmt.Fprintf(os.Stderr, "\n  All chunks complete (%d committed)\n", totalCommitted)
			return nil
		}

		// Start new sessions (max 4 per round to avoid load spikes).
		started := 0
		for _, chunk := range chunks {
			if slots <= 0 || started >= 4 {
				break
			}
			nComm, total := s.chunkProgress(committed, chunk)
			if s.chunkRunning(chunk) {
				continue
			}
			pct := 0
			if total > 0 {
				pct = nComm * 100 / total
			}
			if pct >= s.cfg.DonePct {
				continue
			}
			s.startChunk(chunk, searchBin)
			fmt.Fprintf(os.Stderr, "    +started   g%d_%d\n", chunk.Start, chunk.End)
			slots--
			started++
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Second):
		}
	}
}

func (s *Scheduler) buildChunks() []schedChunk {
	chunkSize := s.cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 50
	}

	if len(s.cfg.GapIndices) > 0 {
		var chunks []schedChunk
		for i := 0; i < len(s.cfg.GapIndices); i += chunkSize {
			end := i + chunkSize - 1
			if end >= len(s.cfg.GapIndices) {
				end = len(s.cfg.GapIndices) - 1
			}
			chunks = append(chunks, schedChunk{
				Start: s.cfg.GapIndices[i],
				End:   s.cfg.GapIndices[end],
			})
		}
		return chunks
	}

	var chunks []schedChunk
	for i := s.cfg.Start; i <= s.cfg.End; i += chunkSize {
		e := i + chunkSize - 1
		if e > s.cfg.End {
			e = s.cfg.End
		}
		chunks = append(chunks, schedChunk{i, e})
	}
	return chunks
}

func (s *Scheduler) chunkProgress(committed map[int]bool, chunk schedChunk) (nComm, total int) {
	for i := chunk.Start; i <= chunk.End; i++ {
		total++
		if committed[i] {
			nComm++
		}
	}
	return
}

func (s *Scheduler) chunkRunning(chunk schedChunk) bool {
	pattern := fmt.Sprintf("cc_v2.*publish.*--file %d-%d$", chunk.Start, chunk.End)
	return exec.Command("pgrep", "-f", pattern).Run() == nil
}

func (s *Scheduler) killChunk(chunk schedChunk) {
	name := fmt.Sprintf("g%d_%d", chunk.Start, chunk.End)
	exec.Command("screen", "-S", name, "-X", "quit").Run()
	pattern := fmt.Sprintf("cc_v2.*publish.*--file %d-%d$", chunk.Start, chunk.End)
	exec.Command("pkill", "-9", "-f", pattern).Run()
}

func (s *Scheduler) startChunk(chunk schedChunk, searchBin string) {
	name := fmt.Sprintf("g%d_%d", chunk.Start, chunk.End)
	home, _ := os.UserHomeDir()
	pathPrefix := filepath.Join(home, "bin")

	cmdStr := fmt.Sprintf(
		"export PATH=%s:$PATH; %s cc_v2 publish --pipeline --skip-errors --file %d-%d",
		pathPrefix, searchBin, chunk.Start, chunk.End,
	)

	// Kill any lingering session with same name.
	exec.Command("screen", "-S", name, "-X", "quit").Run()
	time.Sleep(500 * time.Millisecond)
	exec.Command("screen", "-dmS", name, "bash", "-c", cmdStr+"; exec bash").Run()
}

func (s *Scheduler) resolveSearchBin() string {
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

func (s *Scheduler) computeBudget(hw arctic.HardwareProfile) int {
	if s.cfg.MaxSessions > 0 {
		return s.cfg.MaxSessions
	}
	const ramPerSession = 0.8
	const ramReserve = 2.5
	const maxCap = 12

	usableRAM := hw.RAMTotalGB - ramReserve
	if usableRAM < ramPerSession {
		usableRAM = ramPerSession
	}
	byRAM := int(usableRAM / ramPerSession)
	byCPU := hw.CPUCores
	if byCPU < 1 {
		byCPU = 1
	}

	max := byRAM
	if byCPU < max {
		max = byCPU
	}
	if max < 1 {
		max = 1
	}
	if max > maxCap {
		max = maxCap
	}
	return max
}

func (s *Scheduler) budgetDetail(hw arctic.HardwareProfile, max int) string {
	return fmt.Sprintf("ram=%.0fGB cpu=%d", hw.RAMTotalGB, hw.CPUCores)
}

// ComputeGapIndices returns uncommitted shard indices in [start, end].
func ComputeGapIndices(ctx context.Context, store Store, start, end int) []int {
	committed := store.CommittedSet(ctx)
	if committed == nil {
		committed = make(map[int]bool)
	}
	var gaps []int
	for i := start; i <= end; i++ {
		if !committed[i] {
			gaps = append(gaps, i)
		}
	}
	return gaps
}

// PrintGaps prints gap analysis for the given range.
func PrintGaps(crawlID string, start, end int, gaps []int) {
	total := end - start + 1
	committed := total - len(gaps)
	fmt.Printf("  Crawl    %s\n", crawlID)
	fmt.Printf("  Range    %d–%d (%d shards)\n", start, end, total)
	fmt.Printf("  Done     %d / %d  (%.1f%%)\n", committed, total, float64(committed)*100/float64(total))

	if len(gaps) == 0 {
		fmt.Printf("  Gaps     none — all shards committed\n")
		return
	}
	fmt.Printf("  Gaps     %d\n\n", len(gaps))

	// Collapse into ranges.
	lo, hi := gaps[0], gaps[0]
	for _, n := range gaps[1:] {
		if n == hi+1 {
			hi = n
		} else {
			printRange(lo, hi)
			lo, hi = n, n
		}
	}
	printRange(lo, hi)
	fmt.Println()

	if len(gaps) <= 200 {
		fmt.Printf("  Suggest  search cc_v2 publish --gaps --pipeline --start %d --end %d\n", start, end)
	} else {
		fmt.Printf("  Suggest  search cc_v2 publish --gaps --schedule --start %d --end %d\n", start, end)
	}
}

func printRange(lo, hi int) {
	if lo == hi {
		fmt.Printf("    %5d\n", lo)
	} else {
		fmt.Printf("    %5d – %5d  (%d)\n", lo, hi, hi-lo+1)
	}
}

// ListCommitted prints committed shards as compact ranges.
func ListCommitted(ctx context.Context, store Store, crawlID string) {
	committed := store.CommittedSet(ctx)
	if committed == nil || len(committed) == 0 {
		fmt.Printf("  Crawl    %s\n", crawlID)
		fmt.Printf("  Shards   0 committed\n")
		return
	}
	var indices []int
	for idx := range committed {
		indices = append(indices, idx)
	}
	sortInts(indices)

	fmt.Printf("  Crawl    %s\n", crawlID)
	fmt.Printf("  Shards   %d committed\n", len(indices))

	// Collapse into ranges.
	var parts []string
	lo, hi := indices[0], indices[0]
	for _, n := range indices[1:] {
		if n == hi+1 {
			hi = n
		} else {
			parts = append(parts, fmtRange(lo, hi))
			lo, hi = n, n
		}
	}
	parts = append(parts, fmtRange(lo, hi))
	fmt.Printf("  Ranges   %s\n", strings.Join(parts, ",  "))
}

func fmtRange(lo, hi int) string {
	if lo == hi {
		return strconv.Itoa(lo)
	}
	return fmt.Sprintf("%d–%d (%d)", lo, hi, hi-lo+1)
}

// rateDelta formats a rate change like "(+12)" or "(-4)" or "".
func rateDelta(cur, prev float64) string {
	d := cur - prev
	if d > 0.5 {
		return fmt.Sprintf("(+%.0f)", d)
	}
	if d < -0.5 {
		return fmt.Sprintf("(%.0f)", d)
	}
	return ""
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
