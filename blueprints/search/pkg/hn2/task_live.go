package hn2

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
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

// LiveTask continuously polls the HN source for new items, writing each 5-minute
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

	// --- Cold-start watermark ---
	today := time.Now().UTC().Format("2006-01-02")
	lastDate := today

	todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
	// Sync stats_today.csv from HF if HF has more entries for today (prevents stale local CSV corruption).
	if synced := t.syncStatsTodayFromHF(ctx, cfg, today, todayRows); synced != nil {
		todayRows = synced
	}
	liveWatermark, err := t.coldStartWatermark(ctx, cfg, today, todayRows)
	if err != nil {
		return metric, err
	}

	// Roll over any orphaned blocks from a previous day.
	todayRows = t.rolloverOrphans(ctx, cfg, today, todayRows)

	// Backfill any missing 5-min blocks for today using a single bulk ClickHouse
	// query + DuckDB split (2 quota units regardless of how many blocks to backfill).
	backfillWatermark := liveWatermark
	if id, berr := cfg.maxIDBeforeDate(ctx, today); berr == nil && id > 0 {
		backfillWatermark = id
	} else if berr != nil {
		fmt.Fprintf(os.Stderr, "warn: max id before today: %v — backfill starts from live watermark\n", berr)
	}
	lastHighestID := backfillWatermark
	todayRows, lastHighestID = t.backfillToday(ctx, cfg, today, interval, backfillWatermark, todayRows, &metric, emit)
	if lastHighestID < liveWatermark {
		lastHighestID = liveWatermark
	}

	var totalCommitted int64
	for _, r := range todayRows {
		totalCommitted += r.Count
	}

	// --- Main live polling loop ---
	for {
		if ctx.Err() != nil {
			metric.Elapsed = time.Since(started)
			return metric, nil
		}

		now := time.Now().UTC()
		blockTime := now.Truncate(interval)
		blockDate := blockTime.Format("2006-01-02")
		blockHH := blockTime.Format("15:04")

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
				totalCommitted = 0
				todayRows = nil
				lastDate = blockDate
				// Recalculate backfill watermark for the new day.
				if id, berr := cfg.maxIDBeforeDate(ctx, blockDate); berr == nil && id > 0 {
					backfillWatermark = id
				} else {
					backfillWatermark = lastHighestID
				}
				todayRows, lastHighestID = t.backfillToday(ctx, cfg, blockDate, interval, backfillWatermark, todayRows, &metric, emit)
			}
		}

		// Skip block if already committed.
		if blockCommitted(todayRows, blockDate, blockHH) {
			sleepUntilNext(ctx, interval)
			continue
		}

		outPath := cfg.TodayBlockPath(blockDate, blockHH)
		hfPath := cfg.TodayHFPath(blockDate, blockHH)

		if emit != nil {
			emit(&LiveState{
				Phase:          "fetch",
				Block:          blockDate + " " + blockHH,
				HighestID:      lastHighestID,
				BlocksToday:    len(todayRows),
				TotalCommitted: totalCommitted,
			})
		}

		t0 := time.Now()
		result, err := cfg.FetchSince(ctx, lastHighestID, now, outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: fetch since id=%d: %v\n", lastHighestID, err)
			next := nextIntervalTime(now, interval)
			if emit != nil {
				emit(&LiveState{Phase: "wait", NextFetchIn: time.Until(next)})
			}
			sleepUntilNext(ctx, interval)
			continue
		}
		if result.Count == 0 {
			next := nextIntervalTime(now, interval)
			fmt.Fprintf(os.Stderr, "info: [%s %s] 0 new items (source up to id=%d), next fetch in %s\n",
				blockDate, blockHH, lastHighestID, time.Until(next).Round(time.Second))
			if emit != nil {
				emit(&LiveState{Phase: "wait", NextFetchIn: time.Until(next)})
			}
			sleepUntilNext(ctx, interval)
			continue
		}
		durFetchS := int(time.Since(t0).Seconds())
		lastHighestID = result.HighestID
		totalCommitted += result.Count

		fi, _ := os.Stat(outPath)
		var sizeBytes int64
		if fi != nil {
			sizeBytes = fi.Size()
		}

		newRow := TodayRow{
			Date: blockDate, Block: blockHH,
			LowestID: result.LowestID, HighestID: result.HighestID,
			Count: result.Count, DurFetchS: durFetchS, SizeBytes: sizeBytes,
			CommittedAt: time.Now().UTC(),
		}
		todayRows = updateTodayRow(todayRows, newRow)
		_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
		if readmeBytes, err := t.generateREADME(cfg, todayRows); err == nil {
			_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
		}

		msg := fmt.Sprintf("Live %s %s UTC (%s items)", blockDate, blockHH, fmtInt(result.Count))
		if emit != nil {
			emit(&LiveState{Phase: "commit", Block: blockDate + " " + blockHH, NewItems: result.Count})
		}
		ops := []HFOp{
			{LocalPath: outPath, PathInRepo: hfPath},
			{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}
		t0Commit := time.Now()
		if _, cerr := t.opts.HFCommit(ctx, ops, msg); cerr != nil {
			fmt.Fprintf(os.Stderr, "warn: hf commit block %s %s: %v\n", blockDate, blockHH, cerr)
		} else {
			newRow.DurCommitS = int(time.Since(t0Commit).Seconds())
			todayRows = updateTodayRow(todayRows, newRow)
			_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
			metric.BlocksWritten++
			metric.RowsWritten += result.Count
		}

		next := blockTime.Add(interval)
		if emit != nil {
			emit(&LiveState{Phase: "wait", Block: blockDate + " " + blockHH, NextFetchIn: time.Until(next)})
		}
		sleepUntilNext(ctx, interval)
	}
}

// coldStartWatermark determines the highest committed item ID on startup.
// Priority: today's stats_today.csv → remote source query.
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

func (t *LiveTask) coldStartWatermark(ctx context.Context, cfg Config, today string, todayRows []TodayRow) (int64, error) {
	if id := MaxTodayHighestID(todayRows, today); id > 0 {
		return id, nil
	}
	backoff := time.Minute
	for {
		info, err := cfg.remoteInfo(ctx)
		if err == nil {
			return info.MaxID, nil
		}
		if backoff > 30*time.Minute {
			return 0, fmt.Errorf("remote info for watermark: %w", err)
		}
		fmt.Fprintf(os.Stderr, "warn: remote info failed (%v), retrying in %s\n", err, backoff)
		sleepWithContext(ctx, backoff)
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		backoff *= 2
	}
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

// backfillToday fetches all today's missing 5-min blocks in ONE ClickHouse query,
// then splits the result into per-block Parquet files using DuckDB.
// Only blocks before the current wall-clock interval are backfilled.
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
	// Build set of already-committed blocks.
	committed := make(map[string]bool, len(todayRows))
	for _, r := range todayRows {
		if r.Date == today {
			committed[r.Block] = true
			if r.HighestID > lastHighestID {
				lastHighestID = r.HighestID
			}
		}
	}

	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	nowTrunc := now.Truncate(interval)

	// Enumerate all expected 5-min blocks up to (but not including) current block.
	var missing []time.Time
	for t0 := dayStart; t0.Before(nowTrunc); t0 = t0.Add(interval) {
		hhmm := t0.Format("15:04")
		if !committed[hhmm] {
			missing = append(missing, t0)
		}
	}
	if len(missing) == 0 {
		return todayRows, lastHighestID
	}

	if emit != nil {
		emit(&LiveState{Phase: "fetch", Block: today + " 00:00", HighestID: lastHighestID})
	}

	// ONE ClickHouse query: fetch all items from lastHighestID up to current block start.
	tmpPath := filepath.Join(cfg.TodayDir(), ".today-backfill.tmp.parquet")
	defer os.Remove(tmpPath)

	result, err := cfg.FetchSince(ctx, lastHighestID, nowTrunc, tmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: today backfill bulk fetch: %v\n", err)
		return todayRows, lastHighestID
	}
	if result.Count == 0 {
		return todayRows, lastHighestID
	}

	// Split bulk parquet into per-block files via DuckDB.
	db, err := sql.Open("duckdb", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: open duckdb for backfill split: %v\n", err)
		return todayRows, lastHighestID
	}
	defer db.Close()

	type blockFetched struct {
		hfPath string
		row    TodayRow
	}
	var blocks []blockFetched
	var totalRows int64

	for _, t0 := range missing {
		if ctx.Err() != nil {
			break
		}
		hhmm := t0.Format("15:04")
		t1 := fmt.Sprintf("%s %s:00", today, hhmm)
		t2end := t0.Add(interval)
		t2 := fmt.Sprintf("%s %02d:%02d:00", today, t2end.Hour(), t2end.Minute())

		outPath := cfg.TodayBlockPath(today, hhmm)
		if err := ensureParentDir(outPath); err != nil {
			continue
		}

		var count int64
		var minID, maxID sql.NullInt64
		statsQ := fmt.Sprintf(
			`SELECT COUNT(*)::BIGINT, MIN(id)::BIGINT, MAX(id)::BIGINT FROM read_parquet('%s') WHERE time >= '%s' AND time < '%s'`,
			escapeSQLStr(tmpPath), t1, t2,
		)
		if err := db.QueryRowContext(ctx, statsQ).Scan(&count, &minID, &maxID); err != nil || count == 0 {
			continue
		}

		copyQ := fmt.Sprintf(
			`COPY (SELECT * FROM read_parquet('%s') WHERE time >= '%s' AND time < '%s' ORDER BY id) TO '%s' (FORMAT Parquet)`,
			escapeSQLStr(tmpPath), t1, t2, escapeSQLStr(outPath),
		)
		if _, err := db.ExecContext(ctx, copyQ); err != nil {
			fmt.Fprintf(os.Stderr, "warn: backfill split %s %s: %v\n", today, hhmm, err)
			continue
		}

		fi, _ := os.Stat(outPath)
		var sizeBytes int64
		if fi != nil {
			sizeBytes = fi.Size()
		}
		if maxID.Int64 > lastHighestID {
			lastHighestID = maxID.Int64
		}
		totalRows += count
		blocks = append(blocks, blockFetched{
			hfPath: cfg.TodayHFPath(today, hhmm),
			row: TodayRow{
				Date: today, Block: hhmm,
				LowestID: minID.Int64, HighestID: maxID.Int64,
				Count: count, SizeBytes: sizeBytes,
				CommittedAt: time.Now().UTC(),
			},
		})
	}

	if len(blocks) == 0 || ctx.Err() != nil {
		return todayRows, lastHighestID
	}

	for _, b := range blocks {
		todayRows = updateTodayRow(todayRows, b.row)
	}
	_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows)
	if readmeBytes, err := t.generateREADME(cfg, todayRows); err == nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	// Commit in batches of ≤50 parquet files to stay under the HF per-commit limit.
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
		msg := fmt.Sprintf("Live %s %s UTC (%s items)", today, firstBlock, fmtInt(batchRows))
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
			fmt.Fprintf(os.Stderr, "warn: today backfill commit batch starting %s: %v\n", firstBlock, cerr)
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
		}

		if ctx.Err() != nil {
			break
		}
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

// blockCommitted reports whether a specific (date, hhmm) block is already in todayRows.
func blockCommitted(rows []TodayRow, date, hhmm string) bool {
	for _, r := range rows {
		if r.Date == date && r.Block == hhmm {
			return true
		}
	}
	return false
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

// Keep strings imported (used transitively via fmt in this file).
var _ = strings.TrimSpace
