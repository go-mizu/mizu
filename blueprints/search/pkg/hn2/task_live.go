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
	Block          string // "YYYY-MM-DD HH:MM"
	NewItems       int64
	HighestID      int64
	NextFetchIn    time.Duration
	BlocksToday    int
	TotalCommitted int64
}

// LiveMetric is the aggregate result returned when LiveTask.Run exits (via context cancel).
type LiveMetric struct {
	BlocksWritten int
	RowsWritten   int64
	Rollovers     int
	Elapsed       time.Duration
}

// LiveTaskOptions configures the live polling task.
type LiveTaskOptions struct {
	Interval   time.Duration // poll interval; minimum 1m, default 5m
	HFCommit   CommitFn      // required: commits files to Hugging Face
	ReadmeTmpl []byte        // required: README.md Go template
	Analytics  *Analytics    // optional: enriches README with source-level stats
}

// LiveTask continuously polls the HN source for new items, committing each
// 5-minute batch to Hugging Face. At midnight UTC it merges today's blocks
// into the monthly Parquet via DayRolloverTask.
type LiveTask struct {
	cfg  Config
	opts LiveTaskOptions
}

// NewLiveTask constructs a LiveTask ready to run.
func NewLiveTask(cfg Config, opts LiveTaskOptions) *LiveTask {
	return &LiveTask{cfg: cfg, opts: opts}
}

// Run starts the live polling loop. It returns only when ctx is cancelled,
// at which point it returns the aggregate metrics with a nil error.
func (t *LiveTask) Run(ctx context.Context, emit func(*LiveState)) (LiveMetric, error) {
	cfg := t.cfg.resolved()
	interval := t.opts.Interval
	if interval < time.Minute {
		interval = 5 * time.Minute
	}
	started := time.Now()
	metric := LiveMetric{}

	// --- Cold-start watermark ---
	today := time.Now().UTC().Format("2006-01-02")
	lastDate := today

	todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
	lastHighestID, err := t.coldStartWatermark(ctx, cfg, today, todayRows)
	if err != nil {
		return metric, err
	}
	blocksToday := len(todayRows)
	var totalCommitted int64
	for _, r := range todayRows {
		totalCommitted += r.Count
	}

	// Roll over any orphaned blocks from a previous day (e.g. cross-midnight crash).
	todayRows = t.rolloverOrphans(ctx, cfg, today, todayRows)

	// Backfill any missing 5-min slots from 00:00 UTC to now.
	todayRows, lastHighestID = t.backfillToday(ctx, cfg, today, interval, lastHighestID, todayRows, &metric, emit)

	// --- Main live polling loop ---
	for {
		if ctx.Err() != nil {
			metric.Elapsed = time.Since(started)
			return metric, nil
		}

		now := time.Now().UTC()
		blockTime := now.Truncate(interval)
		blockDate := blockTime.Format("2006-01-02")
		blockHHMM := blockTime.Format("15:04")

		// Day rollover at midnight UTC.
		if blockDate != lastDate {
			if emit != nil {
				emit(&LiveState{Phase: "rollover", Block: lastDate})
			}
			rollover := NewDayRolloverTask(cfg, RolloverTaskOptions{
				PrevDate:   lastDate,
				HFCommit:   t.opts.HFCommit,
				ReadmeTmpl: t.opts.ReadmeTmpl,
				Analytics:  t.opts.Analytics,
			})
			if _, err := rollover.Run(ctx, nil); err != nil {
				fmt.Fprintf(os.Stderr, "warn: day rollover failed: %v\n", err)
				// Do not advance lastDate — retry on next loop iteration.
			} else {
				metric.Rollovers++
				blocksToday = 0
				totalCommitted = 0
				todayRows = nil
				lastDate = blockDate
			}
		}

		outPath := cfg.TodayBlockPath(blockDate, blockHHMM)
		if emit != nil {
			emit(&LiveState{
				Phase:          "fetch",
				Block:          blockDate + " " + blockHHMM,
				HighestID:      lastHighestID,
				BlocksToday:    blocksToday,
				TotalCommitted: totalCommitted,
			})
		}

		result, err := cfg.FetchSince(ctx, lastHighestID, now, outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: fetch since id=%d: %v\n", lastHighestID, err)
			sleepUntilNext(ctx, interval)
			continue
		}
		if result.Count == 0 {
			sleepUntilNext(ctx, interval)
			continue
		}

		lastHighestID = result.HighestID
		blocksToday++
		totalCommitted += result.Count

		newRow := TodayRow{
			Date: blockDate, Block: blockHHMM,
			LowestID: result.LowestID, HighestID: result.HighestID,
			Count: result.Count, DurFetchS: int(result.Duration.Seconds()),
			SizeBytes: result.Bytes, CommittedAt: time.Now().UTC(),
		}
		todayRows = append(todayRows, newRow)
		_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)

		if readmeBytes, err := t.generateREADME(cfg, todayRows); err == nil {
			_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
		}

		if emit != nil {
			emit(&LiveState{Phase: "commit", Block: blockDate + " " + blockHHMM, NewItems: result.Count})
		}

		ops := []HFOp{
			{LocalPath: outPath, PathInRepo: "today/" + blockFilename(blockDate, blockHHMM)},
			{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}
		msg := fmt.Sprintf("Live %s %s (+%s items)", blockDate, blockHHMM, fmtInt(result.Count))
		t0 := time.Now()
		if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
			fmt.Fprintf(os.Stderr, "warn: hf commit block: %v\n", err)
		} else {
			newRow.DurCommitS = int(time.Since(t0).Seconds())
			todayRows = updateTodayRow(todayRows, newRow)
			_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
			metric.BlocksWritten++
			metric.RowsWritten += result.Count
		}

		next := now.Truncate(interval).Add(interval)
		if emit != nil {
			emit(&LiveState{Phase: "wait", Block: blockDate + " " + blockHHMM, NextFetchIn: time.Until(next)})
		}
		sleepUntilNext(ctx, interval)
	}
}

// coldStartWatermark determines the highest committed item ID on startup.
// Priority: today's stats_today.csv → monthly stats.csv → remote source query.
func (t *LiveTask) coldStartWatermark(ctx context.Context, cfg Config, today string, todayRows []TodayRow) (int64, error) {
	if id := MaxTodayHighestID(todayRows, today); id > 0 {
		return id, nil
	}
	if monthRows, _ := ReadStatsCSV(cfg.StatsCSVPath()); len(monthRows) > 0 {
		if id := MaxHighestID(monthRows); id > 0 {
			return id, nil
		}
	}
	info, err := cfg.remoteInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("remote info for watermark: %w", err)
	}
	return info.MaxID, nil
}

// rolloverOrphans rolls over any today/ entries dated before today, which can
// happen after a cross-midnight crash. Handles one orphan date per call; the
// next cold-start will clean up additional orphans if any remain.
func (t *LiveTask) rolloverOrphans(ctx context.Context, cfg Config, today string, todayRows []TodayRow) []TodayRow {
	for _, row := range todayRows {
		if row.Date >= today {
			continue
		}
		orphanDate := row.Date
		fmt.Fprintf(os.Stderr, "warn: orphaned today/ entries for %s — rolling over\n", orphanDate)
		rollover := NewDayRolloverTask(cfg, RolloverTaskOptions{
			PrevDate:   orphanDate,
			HFCommit:   t.opts.HFCommit,
			ReadmeTmpl: t.opts.ReadmeTmpl,
			Analytics:  t.opts.Analytics,
		})
		if _, err := rollover.Run(ctx, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warn: orphan rollover for %s: %v\n", orphanDate, err)
		}
		// Re-read after rollover; stop after one orphan — next restart handles more.
		todayRows, _ = ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		break
	}
	return todayRows
}

// backfillToday fetches and batches-commits any missing 5-minute blocks from
// 00:00 UTC today up to the current truncated interval boundary.
// Each missing block is fetched sequentially (chaining the ID watermark),
// then all are committed in a single HF commit to minimise round-trips.
func (t *LiveTask) backfillToday(
	ctx context.Context,
	cfg Config,
	today string,
	interval time.Duration,
	lastHighestID int64,
	todayRows []TodayRow,
	metric *LiveMetric,
	emit func(*LiveState),
) ([]TodayRow, int64) {
	committed := make(map[string]bool, len(todayRows))
	for _, r := range todayRows {
		if r.Date == today {
			committed[r.Block] = true
			if r.HighestID > lastHighestID {
				lastHighestID = r.HighestID
			}
		}
	}

	dayStart, _ := time.Parse("2006-01-02", today)
	nowTrunc := time.Now().UTC().Truncate(interval)

	type fetched struct {
		outPath  string
		repoPath string
		row      TodayRow
	}
	var blocks []fetched
	var totalRows int64

	for bt := dayStart; bt.Before(nowTrunc); bt = bt.Add(interval) {
		if ctx.Err() != nil {
			break
		}
		hhmm := bt.Format("15:04")
		if committed[hhmm] {
			continue
		}
		date := bt.Format("2006-01-02")
		outPath := cfg.TodayBlockPath(date, hhmm)

		if emit != nil {
			emit(&LiveState{Phase: "fetch", Block: date + " " + hhmm, HighestID: lastHighestID})
		}
		result, err := cfg.FetchSince(ctx, lastHighestID, bt.Add(interval), outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: today backfill fetch %s: %v\n", hhmm, err)
			continue
		}
		if result.Count == 0 {
			continue
		}
		lastHighestID = result.HighestID
		totalRows += result.Count
		blocks = append(blocks, fetched{
			outPath:  outPath,
			repoPath: "today/" + blockFilename(date, hhmm),
			row: TodayRow{
				Date: date, Block: hhmm,
				LowestID: result.LowestID, HighestID: result.HighestID,
				Count: result.Count, DurFetchS: int(result.Duration.Seconds()),
				SizeBytes: result.Bytes, CommittedAt: time.Now().UTC(),
			},
		})
	}

	if len(blocks) == 0 || ctx.Err() != nil {
		return todayRows, lastHighestID
	}

	// Append new rows and write stats_today.csv.
	for _, b := range blocks {
		todayRows = append(todayRows, b.row)
	}
	_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)

	// Regenerate README.
	if readmeBytes, err := t.generateREADME(cfg, todayRows); err == nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	// Single batched HF commit for all backfilled blocks.
	ops := []HFOp{
		{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
		{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
	}
	for _, b := range blocks {
		ops = append(ops, HFOp{LocalPath: b.outPath, PathInRepo: b.repoPath})
	}
	msg := fmt.Sprintf("Live %s today-backfill %d blocks (+%s items)", today, len(blocks), fmtInt(totalRows))
	if emit != nil {
		emit(&LiveState{Phase: "commit", Block: today + " backfill", NewItems: totalRows})
	}
	t0 := time.Now()
	if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
		fmt.Fprintf(os.Stderr, "warn: today backfill commit: %v\n", err)
	} else {
		durS := int(time.Since(t0).Seconds())
		for i := range todayRows {
			for _, b := range blocks {
				if todayRows[i].Date == b.row.Date && todayRows[i].Block == b.row.Block {
					todayRows[i].DurCommitS = durS
				}
			}
		}
		_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
		metric.BlocksWritten += len(blocks)
		metric.RowsWritten += totalRows
	}
	return todayRows, lastHighestID
}

// generateREADME renders the README template from current stats and analytics.
func (t *LiveTask) generateREADME(cfg Config, todayRows []TodayRow) ([]byte, error) {
	monthRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	return GenerateREADME(t.opts.ReadmeTmpl, monthRows, todayRows, t.opts.Analytics)
}

// updateTodayRow replaces the row matching (date, block) in rows with newRow.
func updateTodayRow(rows []TodayRow, newRow TodayRow) []TodayRow {
	for i, r := range rows {
		if r.Date == newRow.Date && r.Block == newRow.Block {
			rows[i] = newRow
			return rows
		}
	}
	return append(rows, newRow)
}

// sleepUntilNext sleeps until the next interval boundary or ctx cancellation.
func sleepUntilNext(ctx context.Context, interval time.Duration) {
	now := time.Now().UTC()
	next := now.Truncate(interval).Add(interval)
	d := time.Until(next)
	if d < time.Second {
		d = time.Second
	}
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// removeTodayRowStr removes leading/trailing whitespace from rows for CSV compat.
// (unused — kept for clarity on the strings import)
var _ = strings.TrimSpace
