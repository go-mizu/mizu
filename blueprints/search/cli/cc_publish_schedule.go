package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ccScheduleConfig holds configuration for the CC pipeline scheduler.
type ccScheduleConfig struct {
	CrawlID     string
	RepoRoot    string
	Start       int
	End         int
	MaxSessions int
	ChunkSize   int
	DonePct     int
	StallRounds int
	SearchBin   string
}

type ccSchedChunk struct {
	Start, End int
}

// runCCScheduleLoop wraps runCCSchedule with an auto-restart loop so the
// scheduler heals itself if it crashes (network errors, transient failures,
// etc.). Returns only on context cancellation or when all chunks are done.
func runCCScheduleLoop(ctx context.Context, cfg ccScheduleConfig) error {
	restartDelay := 10 * time.Second
	attempt := 0
	for {
		attempt++
		if attempt > 1 {
			fmt.Printf("  [schedule] restart attempt %d\n", attempt)
		}
		err := runCCSchedule(ctx, cfg)
		if err == nil {
			return nil // all chunks done
		}
		if ctx.Err() != nil {
			return ctx.Err() // context cancelled
		}
		fmt.Printf("  [schedule] crashed: %v — restarting in %s\n", err, restartDelay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(restartDelay):
		}
	}
}

// runCCSchedule drives CC pipeline screen sessions to cover [start, end].
// It reads stats.csv every 2 minutes, detects stalled sessions, restarts
// them, and fills free slots with new sessions until all chunks are done.
func runCCSchedule(ctx context.Context, cfg ccScheduleConfig) error {
	searchBin := cfg.resolveSearchBin()

	// Build chunk list
	chunks := cfg.buildChunks()

	// Open append-only log file
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

	statsCSV := ccStatsCSVPath(cfg.RepoRoot)

	logLine("=== CC Schedule starting ===")
	logLine(fmt.Sprintf("  Crawl:       %s", cfg.CrawlID))
	logLine(fmt.Sprintf("  Range:       %d\u2013%d", cfg.Start, cfg.End))
	logLine(fmt.Sprintf("  Chunks:      %d  (size=%d)", len(chunks), cfg.ChunkSize))
	logLine(fmt.Sprintf("  Sessions:    %d max", cfg.MaxSessions))
	logLine(fmt.Sprintf("  Done pct:    %d%%", cfg.DonePct))
	logLine(fmt.Sprintf("  Stall kill:  after %d rounds (~%dm) with no new commits", cfg.StallRounds, cfg.StallRounds*2))
	logLine(fmt.Sprintf("  Binary:      %s", searchBin))
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

	round := 0
	for {
		round++

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read committed set (safe: returns empty map on any error)
		committed := ccSchedReadCommitted(statsCSV, cfg.CrawlID)

		var runningNames, todoKeys []string
		nRunning, nDone := 0, 0

		for _, chunk := range chunks {
			total := chunk.End - chunk.Start + 1
			nComm := ccSchedCountRange(committed, chunk.Start, chunk.End)
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
		nTodo := len(todoKeys)
		slots := cfg.MaxSessions - nRunning

		logLine(fmt.Sprintf("Round %d | committed=%d | done=%d/%d chunks | running=%d | todo=%d | slots=%d",
			round, totalCommitted, nDone, len(chunks), nRunning, nTodo, slots))
		if len(runningNames) > 0 {
			logLine("  running: " + strings.Join(runningNames, " "))
		}

		if nRunning == 0 && nTodo == 0 {
			logLine("")
			logLine(fmt.Sprintf("=== All chunks complete for range %d\u2013%d ===", cfg.Start, cfg.End))
			logLine(fmt.Sprintf("Total committed: %d", totalCommitted))
			logLine(fmt.Sprintf("Run: search cc publish --list"))
			return nil
		}

		started := 0
		for _, key := range todoKeys {
			if slots <= 0 {
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
	time.Sleep(200 * time.Millisecond)
	_ = exec.Command("screen", "-dmS", name, "bash", "-c", cmdStr+"; exec bash").Run()
}
