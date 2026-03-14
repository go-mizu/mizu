package hn2

import (
	"context"
	"fmt"
	"os"
	"time"
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

// HistoricalTaskOptions configures a historical backfill run.
type HistoricalTaskOptions struct {
	FromYear   int       // skip months before this year (0 = no limit)
	FromMonth  int       // skip months before this month within FromYear (0 = no limit)
	HFCommit   CommitFn  // required: commits files to Hugging Face
	ReadmeTmpl []byte    // required: README.md Go template
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

// Run executes the historical backfill. It calls emit (if non-nil) on each state
// transition and returns aggregate metrics when all months have been processed.
func (t *HistoricalTask) Run(ctx context.Context, emit func(*HistoricalState)) (HistoricalMetric, error) {
	cfg := t.cfg.resolved()
	started := time.Now()
	metric := HistoricalMetric{}

	existingRows, err := ReadStatsCSV(cfg.StatsCSVPath())
	if err != nil {
		return metric, fmt.Errorf("read stats.csv: %w", err)
	}
	committed := CommittedMonthSet(existingRows)

	months, err := cfg.listMonths(ctx)
	if err != nil {
		return metric, fmt.Errorf("list months: %w", err)
	}

	filtered := filterMonths(months, t.opts.FromYear, t.opts.FromMonth)
	total := len(filtered)
	var bytesDone int64

	for i, m := range filtered {
		if ctx.Err() != nil {
			return metric, ctx.Err()
		}
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
			continue
		}

		outPath := cfg.MonthPath(m.Year, m.Month)
		state.Phase = "fetch"
		if emit != nil {
			emit(state)
		}

		t0Fetch := time.Now()
		result, err := cfg.FetchMonth(ctx, m.Year, m.Month, outPath)
		if err != nil {
			return metric, fmt.Errorf("fetch %s: %w", monthStr, err)
		}
		durFetchS := int(time.Since(t0Fetch).Seconds())

		if result.Count == 0 {
			state.Phase = "skip"
			metric.MonthsSkipped++
			if emit != nil {
				emit(state)
			}
			continue
		}

		state.Rows = result.Count
		state.BytesDone = bytesDone + result.Bytes
		state.Phase = "commit"
		if emit != nil {
			emit(state)
		}

		// Snapshot stats.csv before the pre-commit write so we can roll back
		// if the HF commit fails and the month will be retried next run.
		existingRows, _ = ReadStatsCSV(cfg.StatsCSVPath())
		preCommitRows := make([]MonthRow, len(existingRows))
		copy(preCommitRows, existingRows)

		newRow := MonthRow{
			Year: m.Year, Month: m.Month,
			LowestID: result.LowestID, HighestID: result.HighestID,
			Count: result.Count, DurFetchS: durFetchS,
			SizeBytes: result.Bytes, CommittedAt: time.Now().UTC(),
		}

		// Generate README with the new row included, without a disk round-trip.
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
			{LocalPath: outPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", m.Year, m.Year, m.Month)},
			{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}
		if _, err := t.opts.HFCommit(ctx, ops, fmt.Sprintf("Add %s (%s items)", monthStr, fmtInt(result.Count))); err != nil {
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

		bytesDone += result.Bytes
		metric.MonthsWritten++
		metric.RowsWritten += result.Count
		metric.BytesWritten += result.Bytes
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
