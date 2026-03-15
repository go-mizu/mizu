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
	Phase      string // "fetch" | "commit"
	PrevDate   string
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
	PrevDate   string     // YYYY-MM-DD: the day to roll over (its today/ blocks will be deleted)
	HFCommit   CommitFn   // required: commits files to Hugging Face
	ReadmeTmpl []byte     // required: README.md Go template
	Analytics  *Analytics // optional: enriches README with source-level stats
}

// DayRolloverTask refetches the current month from the source and commits it to
// Hugging Face, removing the individual today/ blocks for the rolled-over day.
// Refetching is more reliable than merging local blocks: it always produces an
// authoritative, deduplicated result directly from the source.
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

	// midnight = start of the day AFTER prevDate (= upper bound of data to fetch).
	midnight := prevTime.AddDate(0, 0, 1)

	if emit != nil {
		emit(&RolloverState{Phase: "fetch", PrevDate: prevDate})
	}

	// Refetch the entire month from the source up to (but not including) midnight.
	// This is more stable than merging local today/ blocks — always authoritative.
	fmt.Fprintf(os.Stderr, "info: rollover %s — refetching %04d-%02d from source\n", prevDate, year, month)
	result, err := cfg.FetchMonthUntil(ctx, year, month, midnight, monthPath)
	if err != nil {
		return metric, fmt.Errorf("rollover refetch %04d-%02d: %w", year, month, err)
	}
	metric.RowsFetched = result.Count
	fmt.Fprintf(os.Stderr, "info: rollover %s — fetched %s items (id %d–%d)\n",
		prevDate, fmtInt(result.Count), result.LowestID, result.HighestID)

	if emit != nil {
		emit(&RolloverState{Phase: "commit", PrevDate: prevDate, RowsFetched: result.Count})
	}

	// Read today rows BEFORE clearing (needed to build the HF delete list).
	prevTodayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())

	// Update stats files.
	fi, _ := os.Stat(monthPath)
	var sizeBytes int64
	if fi != nil {
		sizeBytes = fi.Size()
	}
	existingRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	newMonthRow := MonthRow{
		Year: year, Month: month,
		LowestID: result.LowestID, HighestID: result.HighestID,
		Count: result.Count, DurFetchS: int(result.Duration.Seconds()), SizeBytes: sizeBytes,
		CommittedAt: time.Now().UTC(),
	}
	_ = WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newMonthRow, true)
	_ = ClearStatsTodayCSV(cfg.StatsTodayCSVPath())

	// Regenerate README.
	updatedMonths, _ := ReadStatsCSV(cfg.StatsCSVPath())
	if readmeBytes, err := GenerateREADME(t.opts.ReadmeTmpl, updatedMonths, nil, t.opts.Analytics); err == nil && readmeBytes != nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	// Collect today/ HF paths to delete from prevTodayRows (captured before clear)
	// and from local today/ directory as fallback.
	prevParts := strings.SplitN(prevDate, "-", 3) // ["2026", "03", "14"]
	todayPattern := filepath.Join(cfg.TodayDir(), prevParts[0], prevParts[1], prevParts[2], "*", "*.parquet")
	localTodayFiles, _ := filepath.Glob(todayPattern)

	// Build HF commit: delete today/ blocks, upsert monthly parquet and metadata.
	ops := make([]HFOp, 0, len(prevTodayRows)+len(localTodayFiles)+4)

	// Delete blocks tracked in stats_today.csv (pre-clear snapshot).
	hfPathsSeen := make(map[string]bool)
	for _, r := range prevTodayRows {
		if r.Date != prevDate {
			continue
		}
		hfPath := cfg.TodayHFPath(r.Date, r.Block)
		if !hfPathsSeen[hfPath] {
			ops = append(ops, HFOp{PathInRepo: hfPath, Delete: true})
			hfPathsSeen[hfPath] = true
		}
	}

	// Also delete any local today/ files not captured in stats_today.csv.
	todayDirSlash := cfg.TodayDir() + string(filepath.Separator)
	for _, f := range localTodayFiles {
		rel := strings.TrimPrefix(filepath.ToSlash(f), filepath.ToSlash(todayDirSlash))
		hfPath := "today/" + rel
		if !hfPathsSeen[hfPath] {
			ops = append(ops, HFOp{PathInRepo: hfPath, Delete: true})
			hfPathsSeen[hfPath] = true
		}
	}

	ops = append(ops,
		HFOp{LocalPath: monthPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", year, year, month)},
		HFOp{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
		HFOp{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
		HFOp{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
	)
	msg := fmt.Sprintf("Rollover %s → data/%04d/%04d-%02d.parquet (%s items)", prevDate, year, year, month, fmtInt(result.Count))
	fmt.Fprintf(os.Stderr, "info: rollover %s — committing to HF (%d ops)\n", prevDate, len(ops))
	commitURL, err := t.opts.HFCommit(ctx, ops, msg)
	if err != nil {
		return metric, fmt.Errorf("hf rollover commit: %w", err)
	}
	metric.CommitURL = commitURL

	// Remove local today/ files after confirmed commit.
	for _, f := range localTodayFiles {
		if err := os.Remove(f); err == nil {
			metric.FilesPruned++
		}
		// Remove empty parent directories (HH/ and DD/ levels).
		_ = os.Remove(filepath.Dir(f))
		_ = os.Remove(filepath.Dir(filepath.Dir(f)))
	}
	fmt.Fprintf(os.Stderr, "info: rollover %s — done (%d local files pruned)\n", prevDate, metric.FilesPruned)
	return metric, nil
}

// fileExists reports whether path exists, is a regular file, and is non-empty.
func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir() && fi.Size() > 0
}
