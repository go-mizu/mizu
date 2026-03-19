package hn2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RolloverState is emitted by DayRolloverTask during execution.
type RolloverState struct {
	Phase       string // "fetch" | "commit"
	PrevDate    string
	RowsFetched int64
}

// RolloverMetric is the result returned when DayRolloverTask.Run completes.
type RolloverMetric struct {
	PrevDate    string
	MonthPath   string
	RowsFetched int64
	FilesPruned int
	CommitURL   string
}

// RolloverTaskOptions configures a day rollover run.
type RolloverTaskOptions struct {
	PrevDate   string     // YYYY-MM-DD: the day to roll over (its today/ blocks will be merged + deleted)
	HFCommit   CommitFn   // required: commits files to Hugging Face
	ReadmeTmpl []byte     // required: README.md Go template
	Analytics  *Analytics // optional: enriches README with source-level stats
}

// DayRolloverTask produces a complete month parquet by:
//  1. Re-fetching the month from ClickHouse (authoritative for what it has), and
//  2. Merging the local today/ block files for prevDate (Algolia live data that
//     covers the ClickHouse lag period).
//
// The merged parquet is committed to Hugging Face along with deleted today/ blocks.
// Because local block files are preserved until rollover (not removed after each
// live HF commit), the merge is always available and the month parquet is complete
// regardless of how far ClickHouse has caught up.
type DayRolloverTask struct {
	cfg  Config
	opts RolloverTaskOptions
}

// NewDayRolloverTask constructs a DayRolloverTask ready to run.
func NewDayRolloverTask(cfg Config, opts RolloverTaskOptions) *DayRolloverTask {
	return &DayRolloverTask{cfg: cfg, opts: opts}
}

// Run executes the day rollover. It emits state transitions via emit (if non-nil)
// and returns aggregate metrics on completion.
func (t *DayRolloverTask) Run(ctx context.Context, emit func(*RolloverState)) (RolloverMetric, error) {
	cfg := t.cfg.resolved()
	prevDate := t.opts.PrevDate
	metric := RolloverMetric{PrevDate: prevDate}

	prevTime, err := time.Parse("2006-01-02", prevDate)
	if err != nil {
		return metric, fmt.Errorf("parse prev date: %w", err)
	}
	year, month := prevTime.Year(), int(prevTime.Month())
	monthPath := cfg.MonthPath(year, month)
	metric.MonthPath = monthPath

	// midnight = start of the day AFTER prevDate (upper bound for ClickHouse fetch).
	midnight := prevTime.AddDate(0, 0, 1)

	if emit != nil {
		emit(&RolloverState{Phase: "fetch", PrevDate: prevDate})
	}

	// 1. Fetch month from ClickHouse to a temp path.
	//    Writing to a temp preserves any Algolia gap-fill data already in monthPath
	//    (written by a prior gapFillMonth run) so it survives this rollover.
	chTmpPath := monthPath + ".ch.tmp"
	defer os.Remove(chTmpPath)

	fmt.Fprintf(os.Stderr, "info: rollover %s — fetching %04d-%02d from ClickHouse\n", prevDate, year, month)
	chResult, err := cfg.FetchMonthUntil(ctx, year, month, midnight, chTmpPath)
	if err != nil {
		return metric, fmt.Errorf("rollover fetch %04d-%02d: %w", year, month, err)
	}
	fmt.Fprintf(os.Stderr, "info: rollover %s — ClickHouse: %s items (id %d–%d)\n",
		prevDate, fmtInt(chResult.Count), chResult.LowestID, chResult.HighestID)

	// 2. Find local today/ block files for prevDate (kept on disk until rollover).
	prevParts := strings.SplitN(prevDate, "-", 3) // ["2026", "03", "19"]
	todayPattern := filepath.Join(cfg.TodayDir(), prevParts[0], prevParts[1], prevParts[2], "*", "*.parquet")
	localTodayFiles, _ := filepath.Glob(todayPattern)

	// 3. Merge: CH (authoritative) > existing monthPath (may have Algolia gap data) > local today blocks.
	//    MergeParquets deduplicates by id, first srcPath wins on ties.
	srcPaths := make([]string, 0, 2+len(localTodayFiles))
	srcPaths = append(srcPaths, chTmpPath, monthPath)
	srcPaths = append(srcPaths, localTodayFiles...)
	if len(localTodayFiles) > 0 {
		fmt.Fprintf(os.Stderr, "info: rollover %s — merging CH + existing + %d local blocks\n",
			prevDate, len(localTodayFiles))
	}
	merged, mergeErr := MergeParquets(ctx, monthPath, srcPaths)
	result := chResult
	if mergeErr != nil {
		// Non-fatal: log and continue. monthPath is intact (MergeParquets is atomic).
		// If monthPath is missing and CH had data, promote the CH temp file.
		fmt.Fprintf(os.Stderr, "warn: rollover %s — merge failed: %v; using ClickHouse-only data\n",
			prevDate, mergeErr)
		if _, statErr := os.Stat(monthPath); statErr != nil && chResult.Count > 0 {
			if renErr := os.Rename(chTmpPath, monthPath); renErr != nil {
				return metric, fmt.Errorf("rollover fallback rename: %w", renErr)
			}
		}
	} else {
		added := merged.Count - chResult.Count
		if added > 0 {
			fmt.Fprintf(os.Stderr, "info: rollover %s — merged: %s items total (+%s, id %d–%d)\n",
				prevDate, fmtInt(merged.Count), fmtInt(added), merged.LowestID, merged.HighestID)
		}
		result = merged
	}
	metric.RowsFetched = result.Count

	if emit != nil {
		emit(&RolloverState{Phase: "commit", PrevDate: prevDate, RowsFetched: result.Count})
	}

	// 3. Read today rows BEFORE clearing (needed to build the HF delete list + rollback).
	prevTodayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())

	// 4. Snapshot stats.csv before writing so we can roll back on HF commit failure.
	existingRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	preCommitRows := make([]MonthRow, len(existingRows))
	copy(preCommitRows, existingRows)

	// 5. Update stats files.
	fi, _ := os.Stat(monthPath)
	var sizeBytes int64
	if fi != nil {
		sizeBytes = fi.Size()
	}
	newMonthRow := MonthRow{
		Year: year, Month: month,
		LowestID: result.LowestID, HighestID: result.HighestID,
		Count: result.Count, DurFetchS: int(chResult.Duration.Seconds()), SizeBytes: sizeBytes,
		CommittedAt: time.Now().UTC(),
	}
	_ = WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newMonthRow, true)
	// Write empty stats_today.csv so the HF commit clears it on remote too.
	// Rolled back on commit failure so a retried rollover re-reads the correct rows.
	_ = ClearStatsTodayCSV(cfg.StatsTodayCSVPath())

	// 6. Regenerate README with updated months (no today rows — just rolled over).
	updatedMonths, _ := ReadStatsCSV(cfg.StatsCSVPath())
	if readmeBytes, err := GenerateREADME(t.opts.ReadmeTmpl, updatedMonths, nil, t.opts.Analytics); err == nil && readmeBytes != nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	// 7. Build HF commit: delete confirmed today/ blocks, upsert month parquet + metadata.
	//    Only delete blocks where DurCommitS > 0 — those were confirmed committed to HF
	//    by the live task. Blocks with DurCommitS == 0 were never committed to HF
	//    (live HF commit failed); their data is now in the merged month parquet anyway.
	ops := make([]HFOp, 0, len(prevTodayRows)+4)
	hfPathsSeen := make(map[string]bool)
	for _, r := range prevTodayRows {
		if r.Date != prevDate || r.DurCommitS == 0 {
			continue
		}
		hfPath := cfg.TodayHFPath(r.Date, r.Block)
		if !hfPathsSeen[hfPath] {
			ops = append(ops, HFOp{PathInRepo: hfPath, Delete: true})
			hfPathsSeen[hfPath] = true
		}
	}
	nDeletes := len(ops)
	ops = append(ops,
		HFOp{LocalPath: monthPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", year, year, month)},
		HFOp{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
		HFOp{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
		HFOp{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
	)

	msg := fmt.Sprintf("Rollover %s → data/%04d/%04d-%02d.parquet (%s items)", prevDate, year, year, month, fmtInt(result.Count))
	fmt.Fprintf(os.Stderr, "info: rollover %s — committing to HF (%d deletes + 4 upserts)\n", prevDate, nDeletes)
	commitURL, err := t.opts.HFCommit(ctx, ops, msg)
	if err != nil {
		// Rollback local state so a retried rollover can re-read the correct today rows.
		if wErr := writeStatsCSVExact(cfg.StatsCSVPath(), preCommitRows); wErr != nil {
			fmt.Fprintf(os.Stderr, "warn: rollback stats.csv for rollover %s: %v\n", prevDate, wErr)
		}
		if wErr := WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), prevTodayRows); wErr != nil {
			fmt.Fprintf(os.Stderr, "warn: rollback stats_today.csv for rollover %s: %v\n", prevDate, wErr)
		}
		return metric, fmt.Errorf("hf rollover commit: %w", err)
	}
	metric.CommitURL = commitURL

	// 8. Remove local today/ files after confirmed commit.
	//    Parent directories (HH/ and DD/) are pruned if now empty.
	for _, f := range localTodayFiles {
		if err := os.Remove(f); err == nil {
			metric.FilesPruned++
		}
		_ = os.Remove(filepath.Dir(f))
		_ = os.Remove(filepath.Dir(filepath.Dir(f)))
	}
	fmt.Fprintf(os.Stderr, "info: rollover %s — done (%d local files pruned)\n", prevDate, metric.FilesPruned)
	return metric, nil
}
