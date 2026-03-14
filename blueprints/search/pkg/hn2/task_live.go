package hn2

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// LiveState is emitted by LiveTask on each poll cycle.
type LiveState struct {
	Phase          string // "fetch" | "commit" | "wait" | "rollover"
	Block          string // "2026-03-14 00:05"
	NewItems       int64
	HighestID      int64
	NextFetchIn    time.Duration
	BlocksToday    int
	TotalCommitted int64
}

// LiveMetric is the final result of LiveTask (only returned on context cancel).
type LiveMetric struct {
	BlocksWritten int
	RowsWritten   int64
	Rollovers     int
	Elapsed       time.Duration
}

// LiveTaskOptions configures the live polling task.
type LiveTaskOptions struct {
	Interval   time.Duration // poll interval, default 5m
	HFCommit   func(ctx context.Context, ops []HFOp, message string) (string, error)
	ReadmeTmpl []byte
	Analytics  *Analytics // optional; enriches README with source-level stats
}

// LiveTask implements pkg/core.Task for continuous 5-min live publishing.
type LiveTask struct {
	cfg  Config
	opts LiveTaskOptions
}

func NewLiveTask(cfg Config, opts LiveTaskOptions) *LiveTask {
	return &LiveTask{cfg: cfg, opts: opts}
}

func (t *LiveTask) Run(ctx context.Context, emit func(*LiveState)) (LiveMetric, error) {
	cfg := t.cfg.WithDefaults()
	interval := t.opts.Interval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	started := time.Now()
	metric := LiveMetric{}

	// --- Cold-start watermark ---
	today := time.Now().UTC().Format("2006-01-02")
	lastDate := today
	var lastHighestID int64

	todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
	if maxID := MaxTodayHighestID(todayRows, today); maxID > 0 {
		lastHighestID = maxID
	} else {
		monthRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
		if maxID := MaxHighestID(monthRows); maxID > 0 {
			lastHighestID = maxID
		} else {
			info, err := cfg.RemoteInfo(ctx)
			if err != nil {
				return metric, fmt.Errorf("remote info for watermark: %w", err)
			}
			lastHighestID = info.MaxID
		}
	}

	blocksToday := len(todayRows)
	totalCommitted := int64(0)
	for _, r := range todayRows {
		totalCommitted += r.Count
	}

	// Detect and roll over orphaned today/ files from before today (e.g. after cross-midnight crash).
	for _, row := range todayRows {
		if row.Date < today {
			// Found rows from a previous day — trigger rollover for that date.
			orphanDate := row.Date
			fmt.Fprintf(os.Stderr, "warn: found orphaned today/ entries for %s, rolling over\n", orphanDate)
			rollover := NewDayRolloverTask(cfg, RolloverTaskOptions{
				PrevDate:   orphanDate,
				HFCommit:   t.opts.HFCommit,
				ReadmeTmpl: t.opts.ReadmeTmpl,
				Analytics:  t.opts.Analytics,
			})
			if _, err := rollover.Run(ctx, nil); err != nil {
				fmt.Fprintf(os.Stderr, "warn: orphan rollover for %s failed: %v\n", orphanDate, err)
			}
			// Re-read today rows after rollover.
			todayRows, _ = ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
			break // only handle one orphan at a time; next cold-start handles more
		}
	}

	for {
		if ctx.Err() != nil {
			metric.Elapsed = time.Since(started)
			return metric, nil
		}

		// Compute 5-min aligned block time.
		now := time.Now().UTC()
		blockTime := now.Truncate(interval)
		blockDate := blockTime.Format("2006-01-02")
		blockHHMM := blockTime.Format("15:04")

		// Check for day rollover.
		if blockDate != lastDate {
			state := &LiveState{Phase: "rollover", Block: lastDate}
			if emit != nil {
				emit(state)
			}
			rollover := NewDayRolloverTask(cfg, RolloverTaskOptions{
				PrevDate:   lastDate,
				HFCommit:   t.opts.HFCommit,
				ReadmeTmpl: t.opts.ReadmeTmpl,
				Analytics:  t.opts.Analytics,
			})
			if _, err := rollover.Run(ctx, nil); err != nil {
				fmt.Fprintf(os.Stderr, "warn: day rollover failed: %v\n", err)
				// Do NOT advance lastDate — retry rollover on next loop iteration.
			} else {
				metric.Rollovers++
				blocksToday = 0
				totalCommitted = 0
				todayRows = nil
				lastDate = blockDate
			}
		}

		outPath := cfg.TodayBlockPath(blockDate, blockHHMM)
		state := &LiveState{
			Phase:          "fetch",
			Block:          blockDate + " " + blockHHMM,
			HighestID:      lastHighestID,
			BlocksToday:    blocksToday,
			TotalCommitted: totalCommitted,
		}
		if emit != nil {
			emit(state)
		}

		ceilTime := now // bound query to avoid leaking tomorrow's items
		result, err := cfg.FetchSince(ctx, lastHighestID, ceilTime, outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: fetch since %d: %v\n", lastHighestID, err)
			sleepUntilNext(ctx, interval)
			continue
		}
		if result.Count == 0 {
			_ = os.Remove(outPath + ".tmp")
			sleepUntilNext(ctx, interval)
			continue
		}

		lastHighestID = result.HighestID
		blocksToday++
		totalCommitted += result.Count

		// Append row to stats_today.csv.
		// Build block filename with "_" instead of ":" for filesystem compatibility.
		blockFilename := blockDate + "_" + strings.ReplaceAll(blockHHMM, ":", "_") + ".parquet"
		blockPathInRepo := "today/" + blockFilename

		t0Commit := time.Now()
		todayRows, _ = ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		newTodayRow := TodayRow{
			Date: blockDate, Block: blockHHMM,
			LowestID: result.LowestID, HighestID: result.HighestID,
			Count: result.Count, DurFetchS: int(result.Duration.Seconds()),
			SizeBytes: result.Bytes, CommittedAt: time.Now().UTC(),
		}
		_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows, newTodayRow)

		// Regenerate README from both CSVs.
		monthRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
		allTodayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		readmeBytes, _ := GenerateREADME(t.opts.ReadmeTmpl, monthRows, allTodayRows, t.opts.Analytics)
		if readmeBytes != nil {
			_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
		}

		state.Phase = "commit"
		state.NewItems = result.Count
		if emit != nil {
			emit(state)
		}

		ops := []HFOp{
			{LocalPath: outPath, PathInRepo: blockPathInRepo},
			{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}

		msg := fmt.Sprintf("Live %s %s (+%s items)", blockDate, blockHHMM, fmtInt(result.Count))
		if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
			fmt.Fprintf(os.Stderr, "warn: hf commit block: %v\n", err)
		} else {
			durCommitS := int(time.Since(t0Commit).Seconds())
			newTodayRow.DurCommitS = durCommitS
			// Update the row with the actual commit duration (safe re-read and re-write).
			allTodayRows, _ = ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
			filtered := make([]TodayRow, 0, len(allTodayRows))
			for _, r := range allTodayRows {
				if r.Date == blockDate && r.Block == blockHHMM {
					continue
				}
				filtered = append(filtered, r)
			}
			_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), filtered, newTodayRow)

			metric.BlocksWritten++
			metric.RowsWritten += result.Count
		}

		sleepUntilNext(ctx, interval)
	}
}

func sleepUntilNext(ctx context.Context, interval time.Duration) {
	now := time.Now().UTC()
	next := now.Truncate(interval).Add(interval)
	d := next.Sub(now)
	if d < time.Second {
		d = time.Second
	}
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
