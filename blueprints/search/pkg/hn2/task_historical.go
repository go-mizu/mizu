package hn2

import (
	"context"
	"fmt"
	"os"
	"time"
)

// HistoricalState is emitted by HistoricalTask during execution.
type HistoricalState struct {
	Phase        string // "fetch" | "commit" | "skip"
	Month        string // "2006-10"
	MonthIndex   int
	MonthTotal   int
	Rows         int64
	BytesDone    int64
	ElapsedTotal time.Duration
	SpeedBytesPS float64
}

// HistoricalMetric is the final result of HistoricalTask.
type HistoricalMetric struct {
	MonthsWritten int
	MonthsSkipped int
	RowsWritten   int64
	BytesWritten  int64
	Elapsed       time.Duration
}

// HistoricalTaskOptions configures the historical backfill.
type HistoricalTaskOptions struct {
	FromYear   int // skip months before this year (0 = no limit)
	FromMonth  int // skip months before this month (0 = no limit)
	HFCommit   func(ctx context.Context, ops []HFOp, message string) (string, error)
	ReadmeTmpl []byte
	Analytics  *Analytics // optional; enriches README with source-level stats
}

// HFOp describes a single file operation for a Hugging Face commit.
type HFOp struct {
	LocalPath  string
	PathInRepo string
	Delete     bool
}

// HistoricalTask implements pkg/core.Task for backfilling all historical HN months.
type HistoricalTask struct {
	cfg  Config
	opts HistoricalTaskOptions
}

func NewHistoricalTask(cfg Config, opts HistoricalTaskOptions) *HistoricalTask {
	return &HistoricalTask{cfg: cfg, opts: opts}
}

func (t *HistoricalTask) Run(ctx context.Context, emit func(*HistoricalState)) (HistoricalMetric, error) {
	cfg := t.cfg.WithDefaults()
	started := time.Now()
	metric := HistoricalMetric{}

	// Load already-committed months from stats.csv.
	existingRows, err := ReadStatsCSV(cfg.StatsCSVPath())
	if err != nil {
		return metric, fmt.Errorf("read stats.csv: %w", err)
	}
	committed := CommittedMonthSet(existingRows)

	// Query remote for all available months (current month excluded by ListMonths).
	months, err := cfg.ListMonths(ctx)
	if err != nil {
		return metric, fmt.Errorf("list months: %w", err)
	}

	// Apply --from filter.
	var filtered []MonthInfo
	for _, m := range months {
		if t.opts.FromYear > 0 {
			if m.Year < t.opts.FromYear {
				continue
			}
			if m.Year == t.opts.FromYear && t.opts.FromMonth > 0 && m.Month < t.opts.FromMonth {
				continue
			}
		}
		filtered = append(filtered, m)
	}

	total := len(filtered)
	bytesDone := int64(0)

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

		// Skip already committed.
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
			// Remove any orphaned .tmp file; skip this month.
			_ = os.Remove(outPath + ".tmp")
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

		// Snapshot current stats.csv state before the pre-commit write (for rollback on failure).
		existingRows, _ = ReadStatsCSV(cfg.StatsCSVPath())
		preCommitRows := make([]MonthRow, len(existingRows))
		copy(preCommitRows, existingRows)

		newRow := MonthRow{
			Year: m.Year, Month: m.Month,
			LowestID: result.LowestID, HighestID: result.HighestID,
			Count: result.Count, DurFetchS: durFetchS,
			SizeBytes: result.Bytes, CommittedAt: time.Now().UTC(),
		}

		// Generate README in-memory with the new row included (no disk round-trip needed).
		readmeInputRows := append(append([]MonthRow{}, existingRows...), newRow)
		todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		readmeBytes, readmeErr := GenerateREADME(t.opts.ReadmeTmpl, readmeInputRows, todayRows, t.opts.Analytics)

		// Write stats.csv to disk so it can be included in the HF commit.
		if err := WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newRow, false); err != nil {
			return metric, fmt.Errorf("write stats.csv: %w", err)
		}
		if readmeErr != nil {
			fmt.Fprintf(os.Stderr, "warn: generate README for %s: %v\n", monthStr, readmeErr)
		} else if writeErr := os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "warn: write README for %s: %v\n", monthStr, writeErr)
		}

		t0Commit := time.Now()
		ops := []HFOp{
			{LocalPath: outPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", m.Year, m.Year, m.Month)},
			{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}
		msg := fmt.Sprintf("Add %s (%s items)", monthStr, fmtInt(result.Count))
		if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
			// Rollback stats.csv to pre-commit state so this month is retried next run.
			if wErr := writeStatsCSVExact(cfg.StatsCSVPath(), preCommitRows); wErr != nil {
				fmt.Fprintf(os.Stderr, "warn: rollback stats.csv after commit failure for %s: %v\n", monthStr, wErr)
			}
			return metric, fmt.Errorf("hf commit %s: %w", monthStr, err)
		}
		durCommitS := int(time.Since(t0Commit).Seconds())

		// Update commit duration in stats.csv.
		newRow.DurCommitS = durCommitS
		existingRows, _ = ReadStatsCSV(cfg.StatsCSVPath())
		if err := WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newRow, true); err != nil {
			fmt.Fprintf(os.Stderr, "warn: update stats.csv durCommit for %s: %v\n", monthStr, err)
		}

		bytesDone += result.Bytes
		metric.MonthsWritten++
		metric.RowsWritten += result.Count
		metric.BytesWritten += result.Bytes
		committed[[2]int{m.Year, m.Month}] = true

		state.Rows = result.Count
		state.BytesDone = bytesDone
		elapsed := time.Since(started)
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

func fmtInt(n int64) string {
	// Simple comma-formatted integer.
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}
