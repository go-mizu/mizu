package hn2

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
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
	HFRepo     string        // required: HF dataset repo ID, e.g. "open-index/hacker-news"
	ReadmeTmpl []byte        // required: README.md Go template
	Analytics  *Analytics    // optional: enriches README with source-level stats
}

// LiveTask continuously polls the HN Firebase API for new items, writing each 5-minute
// block directly to today/YYYY/MM/DD/HH/MM.parquet and committing to Hugging Face.
// At midnight UTC it merges today's blocks into the monthly Parquet.
type LiveTask struct {
	cfg  Config
	opts LiveTaskOptions
}

// NewLiveTask constructs a LiveTask ready to run.
func NewLiveTask(cfg Config, opts LiveTaskOptions) *LiveTask {
	return &LiveTask{cfg: cfg, opts: opts}
}

// Run starts the live polling loop. It returns only when ctx is cancelled.
func (t *LiveTask) Run(ctx context.Context, emit func(*LiveState)) (LiveMetric, error) {
	cfg := t.cfg.resolved()
	interval := t.opts.Interval
	if interval < time.Minute {
		interval = 5 * time.Minute
	}
	started := time.Now()
	metric := LiveMetric{}

	// --- Cold-start ---
	today := time.Now().UTC().Format("2006-01-02")
	lastDate := today

	todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
	// Sync stats_today.csv from HF if HF has more entries for today.
	if synced := t.syncStatsTodayFromHF(ctx, cfg, today, todayRows); synced != nil {
		todayRows = synced
	}

	// Log startup state.
	{
		now := time.Now().UTC()
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		nowTrunc := now.Truncate(interval)
		expected := int(nowTrunc.Sub(dayStart) / interval)
		committed := 0
		for _, r := range todayRows {
			if r.Date == today {
				committed++
			}
		}
		missing := expected - committed
		if missing < 0 {
			missing = 0
		}
		fmt.Fprintf(os.Stderr, "info: startup today=%s interval=%s committed=%d expected=%d missing=%d\n",
			today, interval, committed, expected, missing)
	}

	// Roll over any orphaned blocks from a previous day.
	todayRows = t.rolloverOrphans(ctx, cfg, today, todayRows)

	// Backfill missing 5-min blocks using ClickHouse (reliable for historical data).
	todayRows = t.backfillToday(ctx, cfg, today, interval, todayRows, &metric, emit)

	var totalCommitted int64
	for _, r := range todayRows {
		totalCommitted += r.Count
	}
	fmt.Fprintf(os.Stderr, "info: startup complete — %d today rows, %s total items committed\n",
		len(todayRows), fmtInt(totalCommitted))

	// --- Main live polling loop (Algolia HN API) ---
	for {
		if ctx.Err() != nil {
			metric.Elapsed = time.Since(started)
			return metric, nil
		}

		now := time.Now().UTC()
		blockDate := now.Truncate(interval).Format("2006-01-02")

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
			} else {
				metric.Rollovers++
				totalCommitted = 0
				todayRows = nil
				lastDate = blockDate
				todayRows = t.backfillToday(ctx, cfg, blockDate, interval, todayRows, &metric, emit)
			}
		}

		// Determine time to fetch from: start of the oldest missing block.
		// This lets us fill in gaps not covered by the ClickHouse backfill.
		now = time.Now().UTC()
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		currentBlock := now.Truncate(interval)
		oldestMissing, _ := oldestMissingBlock(todayRows, blockDate, dayStart, currentBlock, interval)
		since := oldestMissing
		if since.IsZero() {
			// All blocks up to current interval are committed — fetch current block items.
			since = currentBlock
		}
		// Cap lookback to 3 hours to avoid huge Algolia responses on first run.
		if cap := now.Add(-3 * time.Hour); since.Before(cap) {
			since = cap
		}

		fmt.Fprintf(os.Stderr, "info: fetching Algolia HN items since %s\n", since.Format("15:04:05"))
		if emit != nil {
			emit(&LiveState{Phase: "fetch", Block: blockDate, BlocksToday: len(todayRows), TotalCommitted: totalCommitted})
		}
		t0 := time.Now()
		items, err := FetchHNAlgoliaRecent(ctx, since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: FetchHNAlgoliaRecent: %v; retrying in 30s\n", err)
			sleepWithContext(ctx, 30*time.Second)
			continue
		}
		fmt.Fprintf(os.Stderr, "info: received %d items from Algolia in %s\n",
			len(items), time.Since(t0).Round(time.Millisecond))

		if len(items) == 0 {
			sleepUntilNext(ctx, interval)
			continue
		}

		// Group items into 5-min windows by each item's timestamp.
		windows := GroupHNItemsByWindow(items, interval)

		// Process windows in chronological order.
		windowTimes := make([]time.Time, 0, len(windows))
		for wt := range windows {
			windowTimes = append(windowTimes, wt)
		}
		sort.Slice(windowTimes, func(i, j int) bool { return windowTimes[i].Before(windowTimes[j]) })

		now = time.Now().UTC() // refresh after fetch
		for _, wt := range windowTimes {
			if ctx.Err() != nil {
				break
			}
			wDate := wt.Format("2006-01-02")
			wHHMM := wt.Format("15:04")

			if blockCommitted(todayRows, wDate, wHHMM) {
				fmt.Fprintf(os.Stderr, "info: [%s %s] already committed, skipping\n", wDate, wHHMM)
				continue
			}

			windowItems := windows[wt]
			outPath := cfg.TodayBlockPath(wDate, wHHMM)
			hfPath := cfg.TodayHFPath(wDate, wHHMM)

			fmt.Fprintf(os.Stderr, "info: [%s %s] writing %d HN items\n", wDate, wHHMM, len(windowItems))
			if emit != nil {
				emit(&LiveState{Phase: "fetch", Block: wDate + " " + wHHMM, NewItems: int64(len(windowItems))})
			}

			result, werr := WriteHNParquet(ctx, windowItems, outPath)
			if werr != nil {
				fmt.Fprintf(os.Stderr, "warn: [%s %s] WriteHNParquet: %v\n", wDate, wHHMM, werr)
				continue
			}

			fi, _ := os.Stat(outPath)
			var sizeBytes int64
			if fi != nil {
				sizeBytes = fi.Size()
			}
			newRow := TodayRow{
				Date:        wDate,
				Block:       wHHMM,
				LowestID:    result.LowestID,
				HighestID:   result.HighestID,
				Count:       result.Count,
				DurFetchS:   int(result.Duration.Seconds()),
				SizeBytes:   sizeBytes,
				CommittedAt: time.Now().UTC(),
			}
			todayRows = updateTodayRow(todayRows, newRow)
			totalCommitted += result.Count
			_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
			if readmeBytes, err := t.generateREADME(cfg, todayRows); err == nil {
				_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
			}

			msg := fmt.Sprintf("Live %s %s UTC (%s items)", wDate, wHHMM, fmtInt(result.Count))
			if emit != nil {
				emit(&LiveState{Phase: "commit", Block: wDate + " " + wHHMM, NewItems: result.Count})
			}
			ops := []HFOp{
				{LocalPath: outPath, PathInRepo: hfPath},
				{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
				{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
			}
			t0Commit := time.Now()
			if _, cerr := t.opts.HFCommit(ctx, ops, msg); cerr != nil {
				fmt.Fprintf(os.Stderr, "warn: [%s %s] hf commit: %v\n", wDate, wHHMM, cerr)
			} else {
				newRow.DurCommitS = int(time.Since(t0Commit).Seconds())
				todayRows = updateTodayRow(todayRows, newRow)
				_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
				metric.BlocksWritten++
				metric.RowsWritten += result.Count
				fmt.Fprintf(os.Stderr, "info: [%s %s] committed to HF in %ds\n",
					wDate, wHHMM, newRow.DurCommitS)
			}
		}

		// If there are still missing past blocks, loop immediately (no sleep) to
		// continue catching up. Otherwise sleep until the next interval.
		now = time.Now().UTC()
		dayStart2 := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		stillMissing, _ := oldestMissingBlock(todayRows, blockDate, dayStart2, now.Truncate(interval), interval)
		if !stillMissing.IsZero() && stillMissing.Before(now.Truncate(interval)) {
			// Still have missing past blocks — tight loop with a brief pause.
			fmt.Fprintf(os.Stderr, "info: still missing blocks starting %s, continuing\n", stillMissing.Format("15:04"))
			sleepWithContext(ctx, 10*time.Second)
			continue
		}

		next := nextIntervalTime(now, interval)
		fmt.Fprintf(os.Stderr, "info: next poll at %s (in %s)\n",
			next.UTC().Format("15:04:05"), time.Until(next).Round(time.Second))
		if emit != nil {
			emit(&LiveState{Phase: "wait", Block: blockDate, NextFetchIn: time.Until(next), TotalCommitted: totalCommitted})
		}
		sleepUntilNext(ctx, interval)
	}
}

// syncStatsTodayFromHF downloads stats_today.csv from the public HF repo and returns
// its rows if it contains more entries for today than the local version. This prevents
// a stale local CSV (e.g. written by an old binary) from causing the backfill to start
// from the wrong watermark. Returns nil if HF is not more complete or on any error.
func (t *LiveTask) syncStatsTodayFromHF(ctx context.Context, cfg Config, today string, localRows []TodayRow) []TodayRow {
	if t.opts.HFRepo == "" {
		return nil
	}
	localCount := 0
	for _, r := range localRows {
		if r.Date == today {
			localCount++
		}
	}
	url := "https://huggingface.co/datasets/" + t.opts.HFRepo + "/resolve/main/stats_today.csv"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}
	resp, err := cfg.httpClient().Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	hfRows, err := parseStatsTodayCSV(resp.Body)
	if err != nil {
		return nil
	}
	hfCount := 0
	for _, r := range hfRows {
		if r.Date == today {
			hfCount++
		}
	}
	if hfCount > localCount {
		fmt.Fprintf(os.Stderr, "info: HF stats_today.csv has %d entries for today vs %d local — syncing from HF\n", hfCount, localCount)
		_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), hfRows)
		return hfRows
	}
	return nil
}

// rolloverOrphans rolls over any today/ entries dated before today (cross-midnight crash).
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
		todayRows, _ = ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		break
	}
	return todayRows
}

// backfillToday fetches all missing 5-min blocks for today using per-block time-range queries.
// Iterates from day start up to the current interval, stopping at the first block with 0 items
// (which marks the ClickHouse lag boundary — all later blocks will also be empty).
// The main loop will pick up remaining blocks via the HN Firebase API.
func (t *LiveTask) backfillToday(
	ctx context.Context,
	cfg Config,
	today string,
	interval time.Duration,
	todayRows []TodayRow,
	metric *LiveMetric,
	emit func(*LiveState),
) []TodayRow {
	committed := make(map[string]bool, len(todayRows))
	for _, r := range todayRows {
		if r.Date == today {
			committed[r.Block] = true
		}
	}

	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	nowTrunc := now.Truncate(interval)

	var missing []time.Time
	for t0 := dayStart; t0.Before(nowTrunc); t0 = t0.Add(interval) {
		if !committed[t0.Format("15:04")] {
			missing = append(missing, t0)
		}
	}
	if len(missing) == 0 {
		fmt.Fprintf(os.Stderr, "info: backfill: no missing blocks for %s\n", today)
		return todayRows
	}
	fmt.Fprintf(os.Stderr, "info: backfill: %d missing blocks for %s (first=%s last=%s)\n",
		len(missing), today, missing[0].Format("15:04"), missing[len(missing)-1].Format("15:04"))

	type blockFetched struct {
		hfPath string
		row    TodayRow
	}
	var blocks []blockFetched

	for _, t0 := range missing {
		if ctx.Err() != nil {
			break
		}
		hhmm := t0.Format("15:04")
		blockEnd := t0.Add(interval)
		outPath := cfg.TodayBlockPath(today, hhmm)

		fmt.Fprintf(os.Stderr, "info: [%s %s] backfill: querying %s–%s from source\n",
			today, hhmm, hhmm, blockEnd.Format("15:04"))
		if emit != nil {
			emit(&LiveState{Phase: "fetch", Block: today + " " + hhmm})
		}

		result, err := cfg.FetchTimeRange(ctx, t0, blockEnd, outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: [%s %s] backfill fetch: %v\n", today, hhmm, err)
			break // likely transient; stop backfill, HN API live tail will cover the rest
		}
		if result.Count == 0 {
			// ClickHouse lag boundary — stop backfill here. The HN API live tail
			// will cover items from this point forward.
			fmt.Fprintf(os.Stderr, "info: [%s %s] backfill: 0 items (ClickHouse lag boundary), stopping\n", today, hhmm)
			break
		}

		fi, _ := os.Stat(outPath)
		var sizeBytes int64
		if fi != nil {
			sizeBytes = fi.Size()
		}
		fmt.Fprintf(os.Stderr, "info: [%s %s] backfill: got %s items (id %d–%d)\n",
			today, hhmm, fmtInt(result.Count), result.LowestID, result.HighestID)
		blocks = append(blocks, blockFetched{
			hfPath: cfg.TodayHFPath(today, hhmm),
			row: TodayRow{
				Date: today, Block: hhmm,
				LowestID: result.LowestID, HighestID: result.HighestID,
				Count: result.Count, DurFetchS: int(result.Duration.Seconds()), SizeBytes: sizeBytes,
				CommittedAt: time.Now().UTC(),
			},
		})
	}

	if len(blocks) == 0 || ctx.Err() != nil {
		return todayRows
	}

	for _, b := range blocks {
		todayRows = updateTodayRow(todayRows, b.row)
	}
	_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
	if readmeBytes, err := t.generateREADME(cfg, todayRows); err == nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	// Commit in batches of ≤50 parquet files.
	const hfBatchSize = 50
	for batchStart := 0; batchStart < len(blocks); batchStart += hfBatchSize {
		batchEnd := batchStart + hfBatchSize
		if batchEnd > len(blocks) {
			batchEnd = len(blocks)
		}
		batch := blocks[batchStart:batchEnd]

		var batchRows int64
		for _, b := range batch {
			batchRows += b.row.Count
		}
		firstBlock := batch[0].row.Block
		lastBlock := batch[len(batch)-1].row.Block
		msg := fmt.Sprintf("Live %s %s–%s UTC (%s items)", today, firstBlock, lastBlock, fmtInt(batchRows))
		fmt.Fprintf(os.Stderr, "info: backfill: committing batch %s–%s (%d blocks, %s items)\n",
			firstBlock, lastBlock, len(batch), fmtInt(batchRows))
		if emit != nil {
			emit(&LiveState{Phase: "commit", Block: today + " " + firstBlock, NewItems: batchRows})
		}

		ops := []HFOp{
			{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}
		for _, b := range batch {
			ops = append(ops, HFOp{LocalPath: cfg.TodayBlockPath(today, b.row.Block), PathInRepo: b.hfPath})
		}

		t0Commit := time.Now()
		if _, cerr := t.opts.HFCommit(ctx, ops, msg); cerr != nil {
			fmt.Fprintf(os.Stderr, "warn: backfill commit batch %s: %v\n", firstBlock, cerr)
		} else {
			durS := int(time.Since(t0Commit).Seconds())
			for i := range todayRows {
				for _, b := range batch {
					if todayRows[i].Date == b.row.Date && todayRows[i].Block == b.row.Block {
						todayRows[i].DurCommitS = durS
					}
				}
			}
			_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
			metric.BlocksWritten += len(batch)
			metric.RowsWritten += batchRows
			fmt.Fprintf(os.Stderr, "info: backfill: batch committed in %ds\n", durS)
		}

		if ctx.Err() != nil {
			break
		}
	}
	return todayRows
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

// blockCommitted reports whether a specific (date, hhmm) block is already in todayRows.
func blockCommitted(rows []TodayRow, date, hhmm string) bool {
	for _, r := range rows {
		if r.Date == date && r.Block == hhmm {
			return true
		}
	}
	return false
}

// oldestMissingBlock returns the earliest uncommitted block from dayStart through
// currentBlock (inclusive). Returns zero time and "" if all blocks are committed.
func oldestMissingBlock(rows []TodayRow, date string, dayStart, currentBlock time.Time, interval time.Duration) (time.Time, string) {
	committed := make(map[string]bool, len(rows))
	for _, r := range rows {
		if r.Date == date {
			committed[r.Block] = true
		}
	}
	for t0 := dayStart; !t0.After(currentBlock); t0 = t0.Add(interval) {
		hhmm := t0.Format("15:04")
		if !committed[hhmm] {
			return t0, hhmm
		}
	}
	return time.Time{}, ""
}

// nextIntervalTime returns the next interval boundary after now.
func nextIntervalTime(now time.Time, interval time.Duration) time.Time {
	return now.Truncate(interval).Add(interval)
}

// sleepUntilNext sleeps until the next interval boundary or ctx cancellation.
func sleepUntilNext(ctx context.Context, interval time.Duration) {
	now := time.Now().UTC()
	next := nextIntervalTime(now, interval)
	d := time.Until(next)
	if d < time.Second {
		d = time.Second
	}
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
