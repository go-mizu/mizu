package hn2

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// HistoricalState is emitted by HistoricalTask on each month processed.
type HistoricalState struct {
	Phase        string // "fetch" | "commit" | "skip"
	Month        string // "YYYY-MM"
	MonthIndex   int
	MonthTotal   int
	Rows         int64
	BytesDone    int64
	ElapsedTotal time.Duration
	SpeedBytesPS float64
}

// HistoricalMetric is the aggregate result returned when HistoricalTask.Run completes.
type HistoricalMetric struct {
	MonthsWritten int
	MonthsSkipped int
	RowsWritten   int64
	BytesWritten  int64
	Elapsed       time.Duration
}

// VerifyFn checks which of the given repo paths exist in the HF dataset.
// Implementations may batch the queries however they like. Used by
// HistoricalTask to detect stats.csv/HF mismatches on startup.
type VerifyFn func(ctx context.Context, pathsInRepo []string) (map[string]bool, error)

// HistoricalTaskOptions configures a historical backfill run.
type HistoricalTaskOptions struct {
	FromYear   int        // skip months before this year (0 = no limit)
	FromMonth  int        // skip months before this month within FromYear (0 = no limit)
	Workers    int        // concurrent fetch workers (0 → 4)
	HFCommit   CommitFn   // required: commits files to Hugging Face
	HFVerify   VerifyFn   // optional: if set, verifies each committed month exists on HF at startup
	ReadmeTmpl []byte     // required: README.md Go template
	Analytics  *Analytics // optional: enriches README with source-level stats
}

// HistoricalTask backfills all historical HN months from the remote source to
// a Hugging Face dataset repository, skipping months already tracked in stats.csv.
type HistoricalTask struct {
	cfg  Config
	opts HistoricalTaskOptions
}

// NewHistoricalTask constructs a HistoricalTask ready to run.
func NewHistoricalTask(cfg Config, opts HistoricalTaskOptions) *HistoricalTask {
	return &HistoricalTask{cfg: cfg, opts: opts}
}

// pendingFetch holds the result of a parallel month fetch, ready for commit.
type pendingFetch struct {
	month     monthInfo
	outPath   string
	result    FetchResult
	durFetchS int
}

// Run executes the historical backfill. It calls emit (if non-nil) on each state
// transition and returns aggregate metrics when all months have been processed.
//
// Fetches run in parallel (up to Workers goroutines; default 4) to saturate
// network bandwidth. Commits are sequential to keep stats.csv consistent.
func (t *HistoricalTask) Run(ctx context.Context, emit func(*HistoricalState)) (HistoricalMetric, error) {
	cfg := t.cfg.resolved()
	started := time.Now()
	metric := HistoricalMetric{}

	workers := t.opts.Workers
	if workers <= 0 {
		workers = 4
	}

	existingRows, err := ReadStatsCSV(cfg.StatsCSVPath())
	if err != nil {
		return metric, fmt.Errorf("read stats.csv: %w", err)
	}
	committed := CommittedMonthSet(existingRows)

	// Verify committed months against HF and remove any that are missing there.
	// This fixes stats.csv/HF mismatches caused by interrupted commits.
	if t.opts.HFVerify != nil {
		committed = t.verifyCommitted(ctx, committed)
	}

	months, err := cfg.listMonths(ctx)
	if err != nil {
		return metric, fmt.Errorf("list months: %w", err)
	}

	filtered := filterMonths(months, t.opts.FromYear, t.opts.FromMonth)
	total := len(filtered)
	var bytesDone int64

	// pending is a buffered channel of completed fetches ordered by position.
	// Each slot is either a *pendingFetch (success) or nil (skipped/empty).
	pending := make(chan *pendingFetch, workers*2)

	// fetchErrs receives the first fetch error from the errgroup.
	var fetchErrMu sync.Mutex
	var fetchErr error

	// Producer: fetch months in parallel, preserving order via per-slot channels.
	// Each month gets an independent slot channel so the consumer reads in order.
	slots := make([]chan *pendingFetch, len(filtered))
	for i := range slots {
		slots[i] = make(chan *pendingFetch, 1)
	}

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(workers)
	_ = pending // replaced by slots pattern

	for i, m := range filtered {
		i, m := i, m
		slot := slots[i]

		if committed[[2]int{m.Year, m.Month}] {
			// Already committed; skip without a goroutine.
			slot <- nil
			continue
		}

		eg.Go(func() error {
			monthStr := fmt.Sprintf("%04d-%02d", m.Year, m.Month)
			outPath := cfg.MonthPath(m.Year, m.Month)

			if emit != nil {
				emit(&HistoricalState{
					Phase: "fetch", Month: monthStr,
					MonthIndex: i + 1, MonthTotal: total,
					ElapsedTotal: time.Since(started),
				})
			}

			t0 := time.Now()
			result, err := cfg.FetchMonth(egCtx, m.Year, m.Month, outPath)
			if err != nil {
				fetchErrMu.Lock()
				if fetchErr == nil {
					fetchErr = fmt.Errorf("fetch %s: %w", monthStr, err)
				}
				fetchErrMu.Unlock()
				slot <- nil // unblock consumer; error surfaced after drain
				return nil  // don't cancel other fetches via errgroup
			}
			if result.Count == 0 {
				slot <- nil
				return nil
			}
			slot <- &pendingFetch{
				month:     m,
				outPath:   outPath,
				result:    result,
				durFetchS: int(time.Since(t0).Seconds()),
			}
			return nil
		})
	}

	// Close errgroup after all goroutines are submitted.
	// Consumer below drains slots before we wait for producers.
	go func() { eg.Wait() }() //nolint:errcheck — error captured in fetchErr

	// Consumer: commit each slot in order.
	for i, m := range filtered {
		monthStr := fmt.Sprintf("%04d-%02d", m.Year, m.Month)
		state := &HistoricalState{
			Month:        monthStr,
			MonthIndex:   i + 1,
			MonthTotal:   total,
			ElapsedTotal: time.Since(started),
		}

		if committed[[2]int{m.Year, m.Month}] {
			state.Phase = "skip"
			metric.MonthsSkipped++
			if emit != nil {
				emit(state)
			}
			<-slots[i] // drain the nil we sent above
			continue
		}

		pf := <-slots[i]

		if pf == nil {
			// Fetch returned 0 rows, or failed (fetchErr set).
			fetchErrMu.Lock()
			ferr := fetchErr
			fetchErrMu.Unlock()
			if ferr != nil {
				return metric, ferr
			}
			state.Phase = "skip"
			metric.MonthsSkipped++
			if emit != nil {
				emit(state)
			}
			continue
		}

		state.Rows = pf.result.Count
		state.BytesDone = bytesDone + pf.result.Bytes
		state.Phase = "commit"
		if emit != nil {
			emit(state)
		}

		// Snapshot stats.csv before writing so we can roll back on HF commit failure.
		existingRows, _ = ReadStatsCSV(cfg.StatsCSVPath())
		preCommitRows := make([]MonthRow, len(existingRows))
		copy(preCommitRows, existingRows)

		newRow := MonthRow{
			Year: m.Year, Month: m.Month,
			LowestID: pf.result.LowestID, HighestID: pf.result.HighestID,
			Count: pf.result.Count, DurFetchS: pf.durFetchS,
			SizeBytes: pf.result.Bytes, CommittedAt: time.Now().UTC(),
		}

		readmeInputRows := append(append([]MonthRow{}, existingRows...), newRow)
		todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		readmeBytes, readmeErr := GenerateREADME(t.opts.ReadmeTmpl, readmeInputRows, todayRows, t.opts.Analytics)

		if err := WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newRow, false); err != nil {
			return metric, fmt.Errorf("write stats.csv: %w", err)
		}
		if readmeErr != nil {
			fmt.Fprintf(os.Stderr, "warn: generate README for %s: %v\n", monthStr, readmeErr)
		} else if err := os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "warn: write README for %s: %v\n", monthStr, err)
		}

		t0Commit := time.Now()
		ops := []HFOp{
			{LocalPath: pf.outPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", m.Year, m.Year, m.Month)},
			{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}
		if _, err := t.opts.HFCommit(ctx, ops, fmt.Sprintf("Add %s (%s items)", monthStr, fmtInt(pf.result.Count))); err != nil {
			if wErr := writeStatsCSVExact(cfg.StatsCSVPath(), preCommitRows); wErr != nil {
				fmt.Fprintf(os.Stderr, "warn: rollback stats.csv for %s: %v\n", monthStr, wErr)
			}
			return metric, fmt.Errorf("hf commit %s: %w", monthStr, err)
		}

		// Update commit duration in stats.csv.
		newRow.DurCommitS = int(time.Since(t0Commit).Seconds())
		existingRows, _ = ReadStatsCSV(cfg.StatsCSVPath())
		if err := WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newRow, true); err != nil {
			fmt.Fprintf(os.Stderr, "warn: update stats.csv dur_commit for %s: %v\n", monthStr, err)
		}

		bytesDone += pf.result.Bytes
		metric.MonthsWritten++
		metric.RowsWritten += pf.result.Count
		metric.BytesWritten += pf.result.Bytes
		committed[[2]int{m.Year, m.Month}] = true

		elapsed := time.Since(started)
		state.BytesDone = bytesDone
		state.ElapsedTotal = elapsed
		if elapsed.Seconds() > 0 {
			state.SpeedBytesPS = float64(bytesDone) / elapsed.Seconds()
		}
		if emit != nil {
			emit(state)
		}
	}

	metric.Elapsed = time.Since(started)
	return metric, nil
}

// verifyCommitted batch-checks all months in committed against HF and removes
// any that are missing. One VerifyFn call for all months (implementations batch
// internally). On error, logs a warning and keeps all months as-is.
func (t *HistoricalTask) verifyCommitted(ctx context.Context, committed map[[2]int]bool) map[[2]int]bool {
	if len(committed) == 0 {
		return committed
	}
	// Build path list preserving the month key mapping.
	paths := make([]string, 0, len(committed))
	pathToKey := make(map[string][2]int, len(committed))
	for k := range committed {
		year, month := k[0], k[1]
		path := fmt.Sprintf("data/%04d/%04d-%02d.parquet", year, year, month)
		paths = append(paths, path)
		pathToKey[path] = k
	}
	existing, err := t.opts.HFVerify(ctx, paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: HF batch verify failed (%v) — skipping repair, all committed months treated as present\n", err)
		return committed
	}
	var missing int
	for path, k := range pathToKey {
		if !existing[path] {
			fmt.Fprintf(os.Stderr, "warn: %04d-%02d in stats.csv but missing on HF — will re-commit\n", k[0], k[1])
			delete(committed, k)
			missing++
		}
	}
	if missing > 0 {
		fmt.Fprintf(os.Stderr, "info: HF verify: %d months missing on HF, queued for re-commit\n", missing)
	}
	return committed
}

// filterMonths returns only the months at or after fromYear/fromMonth.
// If fromYear is 0, all months are returned.
func filterMonths(months []monthInfo, fromYear, fromMonth int) []monthInfo {
	if fromYear == 0 {
		return months
	}
	out := months[:0:0]
	for _, m := range months {
		if m.Year < fromYear {
			continue
		}
		if m.Year == fromYear && fromMonth > 0 && m.Month < fromMonth {
			continue
		}
		out = append(out, m)
	}
	return out
}
