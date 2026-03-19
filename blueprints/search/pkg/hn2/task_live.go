package hn2

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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

// ListFn lists all file paths recursively under a given HF path prefix.
// Returns nil paths (no error) if the prefix does not exist on HF.
type ListFn func(ctx context.Context, pathPrefix string) ([]string, error)

// LiveTaskOptions configures the live polling task.
type LiveTaskOptions struct {
	Interval   time.Duration // poll interval; minimum 1m, default 5m
	HFCommit   CommitFn      // required: commits files to Hugging Face
	HFRepo     string        // required: HF dataset repo ID, e.g. "open-index/hacker-news"
	HFListDir  ListFn        // optional: lists HF files; used to detect and clean HF-only orphan today/ blocks
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

	// Remove stale today/ files that don't match the current YYYY/MM/DD/HH/MM format.
	cleanOrphanTodayFiles(cfg)

	// Roll over any orphaned blocks from a previous day.
	todayRows = t.rolloverOrphans(ctx, cfg, today, todayRows)

	// Fill any gap in the current month parquet (ClickHouse lag + deleted today/ blocks).
	// This is a no-op when the month was recently committed; only runs if a gap is detected.
	t.gapFillMonth(ctx, cfg, today)

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
				durS := int(time.Since(t0Commit).Seconds())
				if durS == 0 {
					durS = 1 // ensure DurCommitS > 0 marks this block as confirmed on HF
				}
				newRow.DurCommitS = durS
				todayRows = updateTodayRow(todayRows, newRow)
				_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
				metric.BlocksWritten++
				metric.RowsWritten += result.Count
				fmt.Fprintf(os.Stderr, "info: [%s %s] committed to HF in %ds\n",
					wDate, wHHMM, newRow.DurCommitS)
				// Local block file is kept on disk until rollover merges it into the
				// month parquet. DurCommitS > 0 signals the rollover that this block
				// is on HF and should be deleted after the merge.
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
// It collects orphaned dates from three sources:
//  1. stats_today.csv rows with Date < today
//  2. Local today/ subdirectories with Date < today
//  3. HF today/ date directories (via HFListDir) with Date < today that have no local presence
//
// Sources 1 & 2 trigger a full day rollover (re-fetch month + delete HF blocks).
// Source 3 (HF-only orphans) triggers a targeted delete-only commit to remove the stale files.
// All dates are processed in chronological order; processing stops on context cancel.
func (t *LiveTask) rolloverOrphans(ctx context.Context, cfg Config, today string, todayRows []TodayRow) []TodayRow {
	orphanSet := make(map[string]bool)

	// Source 1: stats_today.csv.
	for _, row := range todayRows {
		if row.Date < today {
			orphanSet[row.Date] = true
		}
	}

	// Source 2: local today/ directory (catches dates not in stats_today.csv, e.g. after
	// repeated rollover failures that cleared stats_today.csv without rollback).
	// Structure: today/YYYY/MM/DD/HH/MM.parquet → date dir is at depth 3.
	todayDir := cfg.TodayDir()
	if yearEntries, err := os.ReadDir(todayDir); err == nil {
		for _, ye := range yearEntries {
			if !ye.IsDir() {
				continue
			}
			monthEntries, _ := os.ReadDir(filepath.Join(todayDir, ye.Name()))
			for _, me := range monthEntries {
				if !me.IsDir() {
					continue
				}
				dayEntries, _ := os.ReadDir(filepath.Join(todayDir, ye.Name(), me.Name()))
				for _, de := range dayEntries {
					if !de.IsDir() {
						continue
					}
					date := ye.Name() + "-" + me.Name() + "-" + de.Name()
					if date < today {
						orphanSet[date] = true
					}
				}
			}
		}
	}

	// Source 3: HF remote today/ directories (catches dates where local files were
	// successfully committed + removed, but rollover never ran to clean HF).
	// Keyed by date → list of HF file paths to delete.
	hfOnlyOrphans := make(map[string][]string)
	if t.opts.HFListDir != nil && ctx.Err() == nil {
		// Scan the current month and prior month (covers all realistic missed-rollover windows).
		now := time.Now().UTC()
		monthsToScan := []string{
			now.Format("2006/01"),
			now.AddDate(0, -1, 0).Format("2006/01"),
		}
		for _, ym := range monthsToScan {
			if ctx.Err() != nil {
				break
			}
			hfPrefix := "today/" + ym
			files, err := t.opts.HFListDir(ctx, hfPrefix)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warn: HFListDir %s: %v\n", hfPrefix, err)
				continue
			}
			// Group files by date: today/YYYY/MM/DD/HH/MM.parquet → YYYY-MM-DD
			for _, f := range files {
				// f = "today/2026/03/15/13/15.parquet"
				parts := strings.SplitN(f, "/", 6)
				if len(parts) < 4 {
					continue
				}
				date := parts[1] + "-" + parts[2] + "-" + parts[3]
				if date >= today {
					continue // keep current day's data
				}
				if !orphanSet[date] {
					// Only collect as HF-only if not already covered by local rollover.
					hfOnlyOrphans[date] = append(hfOnlyOrphans[date], f)
				}
			}
		}
	}

	// Run full day rollover for dates detected locally (sources 1 & 2).
	if len(orphanSet) > 0 {
		dates := make([]string, 0, len(orphanSet))
		for d := range orphanSet {
			dates = append(dates, d)
		}
		sort.Strings(dates)

		for _, orphanDate := range dates {
			if ctx.Err() != nil {
				break
			}
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
		}

		todayRows, _ = ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
	}

	// Delete HF-only orphans (source 3): files that exist on HF but have no local
	// counterpart and no stats_today.csv record. Issue a targeted delete commit.
	if len(hfOnlyOrphans) > 0 {
		hfDates := make([]string, 0, len(hfOnlyOrphans))
		for d := range hfOnlyOrphans {
			hfDates = append(hfDates, d)
		}
		sort.Strings(hfDates)

		for _, orphanDate := range hfDates {
			if ctx.Err() != nil {
				break
			}
			hfFiles := hfOnlyOrphans[orphanDate]
			fmt.Fprintf(os.Stderr, "warn: HF-only orphan today/%s (%d files) — issuing delete commit\n",
				strings.ReplaceAll(orphanDate, "-", "/"), len(hfFiles))
			ops := make([]HFOp, len(hfFiles))
			for i, f := range hfFiles {
				ops[i] = HFOp{PathInRepo: f, Delete: true}
			}
			msg := fmt.Sprintf("Cleanup stale today/%s blocks (%d files)", orphanDate, len(ops))
			if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
				fmt.Fprintf(os.Stderr, "warn: HF orphan cleanup for %s: %v\n", orphanDate, err)
			} else {
				fmt.Fprintf(os.Stderr, "info: cleaned up HF today/%s (%d files deleted)\n",
					strings.ReplaceAll(orphanDate, "-", "/"), len(ops))
			}
		}
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
			if durS == 0 {
				durS = 1 // ensure DurCommitS > 0 marks blocks as confirmed on HF
			}
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
			// Local block files are kept on disk until rollover merges them into
			// the month parquet. DurCommitS > 0 signals confirmed HF commits.
		}

		if ctx.Err() != nil {
			break
		}
	}
	return todayRows
}

// gapFillMonth checks whether the current month's parquet has a gap (items missing between
// the last committed data and today's midnight) and fills it using ClickHouse + Algolia.
//
// It runs once at cold start. If stats.csv shows the month was committed within the last
// 20 hours it returns immediately (recently rolled over → no gap expected). Otherwise it:
//  1. Re-fetches the month from ClickHouse to a temp file (may have caught up since last run).
//  2. Finds any remaining gap between the ClickHouse max time and today midnight.
//  3. Fetches items in that gap window from the Algolia search API.
//  4. Merges CH + existing month + gap data into the month parquet (CH wins on duplicates).
//  5. Updates stats.csv and commits the updated parquet to Hugging Face.
func (t *LiveTask) gapFillMonth(ctx context.Context, cfg Config, today string) {
	todayTime, err := time.Parse("2006-01-02", today)
	if err != nil {
		return
	}
	year, month := todayTime.Year(), int(todayTime.Month())
	todayMidnight := todayTime.UTC() // YYYY-MM-DD → 00:00:00 UTC

	// Check when this month was last committed.
	monthRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	var monthRow *MonthRow
	for i := range monthRows {
		if monthRows[i].Year == year && monthRows[i].Month == month {
			monthRow = &monthRows[i]
			break
		}
	}
	if monthRow == nil {
		return // no stats entry yet; historical task handles initial population
	}
	if time.Since(monthRow.CommittedAt) < 20*time.Hour {
		return // committed recently — assume up-to-date
	}

	monthPath := cfg.MonthPath(year, month)
	if _, statErr := os.Stat(monthPath); statErr != nil {
		return // month parquet missing; skip
	}

	fmt.Fprintf(os.Stderr, "info: gapFill: %04d-%02d last committed %s ago — checking for gaps\n",
		year, month, time.Since(monthRow.CommittedAt).Round(time.Hour))

	// Step 1: re-fetch ClickHouse for the whole month up to today midnight.
	chTmpPath := monthPath + ".gf-ch.tmp"
	defer os.Remove(chTmpPath)

	chResult, err := cfg.FetchMonthUntil(ctx, year, month, todayMidnight, chTmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: gapFill: ClickHouse fetch failed: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "info: gapFill: ClickHouse has %s items (id %d–%d)\n",
		fmtInt(chResult.Count), chResult.LowestID, chResult.HighestID)

	// Step 2: find the gap — max of (CH max time, existing month max time) → today midnight.
	chMaxTime := ScanParquetMaxTime(ctx, chTmpPath)
	existingMaxTime := ScanParquetMaxTime(ctx, monthPath)
	coverUpTo := chMaxTime
	if existingMaxTime.After(coverUpTo) {
		coverUpTo = existingMaxTime
	}
	// Round up to the next 5-min boundary to avoid re-fetching the tail minute.
	gapStart := coverUpTo.Truncate(5 * time.Minute).Add(5 * time.Minute)

	// Step 3: fetch the gap from Algolia if one exists.
	var gapPaths []string
	if !gapStart.IsZero() && gapStart.Before(todayMidnight) {
		fmt.Fprintf(os.Stderr, "info: gapFill: gap detected %s – %s, fetching from Algolia\n",
			gapStart.Format("2006-01-02 15:04"), todayMidnight.Format("2006-01-02 15:04"))
		gapItems, algErr := FetchHNAlgoliaRange(ctx, gapStart, todayMidnight)
		if algErr != nil {
			fmt.Fprintf(os.Stderr, "warn: gapFill: Algolia fetch failed: %v\n", algErr)
			// Continue without gap data — CH alone may have caught up.
		} else if len(gapItems) > 0 {
			gapPath := monthPath + ".gf-gap.tmp"
			defer os.Remove(gapPath)
			if _, werr := WriteHNParquet(ctx, gapItems, gapPath); werr == nil {
				gapPaths = append(gapPaths, gapPath)
				fmt.Fprintf(os.Stderr, "info: gapFill: Algolia gap: %d items\n", len(gapItems))
			} else {
				fmt.Fprintf(os.Stderr, "warn: gapFill: write gap parquet: %v\n", werr)
			}
		} else {
			fmt.Fprintf(os.Stderr, "info: gapFill: Algolia gap: 0 items (CH may have caught up)\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "info: gapFill: no gap (coverUpTo=%s)\n", coverUpTo.Format("2006-01-02 15:04"))
	}

	// Step 4: merge CH > existing month > gap data.
	srcPaths := make([]string, 0, 3+len(gapPaths))
	srcPaths = append(srcPaths, chTmpPath, monthPath)
	srcPaths = append(srcPaths, gapPaths...)

	merged, mergeErr := MergeParquets(ctx, monthPath, srcPaths)
	if mergeErr != nil {
		fmt.Fprintf(os.Stderr, "warn: gapFill: merge failed: %v\n", mergeErr)
		return
	}
	if merged.Count == 0 {
		fmt.Fprintf(os.Stderr, "warn: gapFill: merge produced 0 rows — skipping commit\n")
		return
	}
	added := merged.Count - int64(monthRow.Count)
	fmt.Fprintf(os.Stderr, "info: gapFill: merged %s items (+%s), id %d–%d\n",
		fmtInt(merged.Count), fmtInt(added), merged.LowestID, merged.HighestID)

	// Step 5: update stats.csv and commit to HF.
	fi, _ := os.Stat(monthPath)
	var sizeBytes int64
	if fi != nil {
		sizeBytes = fi.Size()
	}
	preCommitRows := make([]MonthRow, len(monthRows))
	copy(preCommitRows, monthRows)
	newMonthRow := MonthRow{
		Year: year, Month: month,
		LowestID: merged.LowestID, HighestID: merged.HighestID,
		Count: merged.Count, DurFetchS: int(chResult.Duration.Seconds()), SizeBytes: sizeBytes,
		CommittedAt: time.Now().UTC(),
	}
	_ = WriteStatsCSV(cfg.StatsCSVPath(), monthRows, newMonthRow, true)

	todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
	updatedMonths, _ := ReadStatsCSV(cfg.StatsCSVPath())
	if readmeBytes, genErr := GenerateREADME(t.opts.ReadmeTmpl, updatedMonths, todayRows, t.opts.Analytics); genErr == nil && readmeBytes != nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	msg := fmt.Sprintf("Gap fill %04d-%02d (%s items, +%s from Algolia)", year, month, fmtInt(merged.Count), fmtInt(added))
	ops := []HFOp{
		{LocalPath: monthPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", year, year, month)},
		{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
		{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
	}
	if _, commitErr := t.opts.HFCommit(ctx, ops, msg); commitErr != nil {
		fmt.Fprintf(os.Stderr, "warn: gapFill: HF commit failed: %v — rolling back stats.csv\n", commitErr)
		_ = writeStatsCSVExact(cfg.StatsCSVPath(), preCommitRows)
	} else {
		fmt.Fprintf(os.Stderr, "info: gapFill: committed %04d-%02d to HF (%s items)\n", year, month, fmtInt(merged.Count))
	}
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

// cleanOrphanTodayFiles removes any parquet files in the today/ root directory
// that don't match the current YYYY/MM/DD/HH/MM.parquet layout (e.g. stale
// flat-format files written by an older binary version).
func cleanOrphanTodayFiles(cfg Config) {
	pattern := filepath.Join(cfg.TodayDir(), "*.parquet")
	matches, _ := filepath.Glob(pattern)
	for _, f := range matches {
		if err := os.Remove(f); err == nil {
			fmt.Fprintf(os.Stderr, "info: removed orphan today/ file: %s\n", filepath.Base(f))
		}
	}
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
