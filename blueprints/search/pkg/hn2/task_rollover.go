package hn2

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// RolloverState is emitted by DayRolloverTask.
type RolloverState struct {
	Phase      string // "merge" | "commit"
	PrevDate   string
	FilesFound int
	RowsMerged int64
}

// RolloverMetric is the final result of DayRolloverTask.
type RolloverMetric struct {
	PrevDate    string
	MonthPath   string
	RowsMerged  int64
	FilesPruned int
	CommitURL   string
}

// RolloverTaskOptions configures the day rollover.
type RolloverTaskOptions struct {
	PrevDate   string // YYYY-MM-DD
	HFCommit   func(ctx context.Context, ops []HFOp, message string) (string, error)
	ReadmeTmpl []byte
}

// DayRolloverTask merges today's live blocks into the monthly parquet and commits to HF.
type DayRolloverTask struct {
	cfg  Config
	opts RolloverTaskOptions
}

func NewDayRolloverTask(cfg Config, opts RolloverTaskOptions) *DayRolloverTask {
	return &DayRolloverTask{cfg: cfg, opts: opts}
}

func (t *DayRolloverTask) Run(ctx context.Context, emit func(*RolloverState)) (RolloverMetric, error) {
	cfg := t.cfg.WithDefaults()
	prevDate := t.opts.PrevDate
	metric := RolloverMetric{PrevDate: prevDate}

	// Determine year/month from prevDate.
	prevTime, err := time.Parse("2006-01-02", prevDate)
	if err != nil {
		return metric, fmt.Errorf("parse prev date: %w", err)
	}
	year, month := prevTime.Year(), int(prevTime.Month())
	monthPath := cfg.MonthPath(year, month)
	metric.MonthPath = monthPath

	// Collect today/ files for prevDate.
	pattern := filepath.Join(cfg.TodayDir(), prevDate+"_*.parquet")
	todayFiles, _ := filepath.Glob(pattern)

	state := &RolloverState{Phase: "merge", PrevDate: prevDate, FilesFound: len(todayFiles)}
	if emit != nil {
		emit(state)
	}

	// Early return: nothing to do if no today files and no month file yet.
	if len(todayFiles) == 0 && !fileExistsNE(monthPath) {
		return metric, nil
	}

	// Merge block: only if we have today files to merge in.
	if len(todayFiles) > 0 {
		// Build parquet source list: existing monthly (if any) + today files.
		var sources []string
		if fileExistsNE(monthPath) {
			sources = append(sources, monthPath)
		}
		sources = append(sources, todayFiles...)

		tmpPath := monthPath + ".tmp"
		_ = os.Remove(tmpPath)
		if err := os.MkdirAll(filepath.Dir(monthPath), 0o755); err != nil {
			return metric, fmt.Errorf("create month dir: %w", err)
		}

		// Build DuckDB read_parquet list.
		listSQL := buildParquetList(sources)
		mergeQ := fmt.Sprintf(
			`COPY (SELECT * FROM read_parquet(%s) ORDER BY id) TO '%s' (FORMAT PARQUET, COMPRESSION zstd, COMPRESSION_LEVEL 22)`,
			listSQL, escapeSQLStr(tmpPath),
		)
		db, err := sql.Open("duckdb", "")
		if err != nil {
			return metric, fmt.Errorf("open duckdb for merge: %w", err)
		}
		_, mergeErr := db.ExecContext(ctx, mergeQ)
		db.Close()
		if mergeErr != nil {
			_ = os.Remove(tmpPath)
			return metric, fmt.Errorf("merge parquet: %w", mergeErr)
		}
		if err := os.Rename(tmpPath, monthPath); err != nil {
			_ = os.Remove(tmpPath)
			return metric, fmt.Errorf("rename merged parquet: %w", err)
		}
	}

	// Scan merged file.
	db2, err := sql.Open("duckdb", "")
	if err != nil {
		return metric, fmt.Errorf("open duckdb for scan: %w", err)
	}
	var count, minID, maxID int64
	scanQ := fmt.Sprintf(`SELECT COUNT(*)::BIGINT, MIN(id)::BIGINT, MAX(id)::BIGINT FROM read_parquet('%s')`, escapeSQLStr(monthPath))
	_ = db2.QueryRowContext(ctx, scanQ).Scan(&count, &minID, &maxID)
	db2.Close()
	metric.RowsMerged = count

	state.RowsMerged = count
	state.Phase = "commit"
	if emit != nil {
		emit(state)
	}

	// Upsert stats.csv with the merged month row.
	fi, _ := os.Stat(monthPath)
	var sizeBytes int64
	if fi != nil {
		sizeBytes = fi.Size()
	}
	existingRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	newMonthRow := MonthRow{
		Year: year, Month: month,
		LowestID: minID, HighestID: maxID,
		Count: count, SizeBytes: sizeBytes,
		CommittedAt: time.Now().UTC(),
	}
	_ = WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newMonthRow, true /* upsert */)

	// Clear stats_today.csv.
	_ = ClearStatsTodayCSV(cfg.StatsTodayCSVPath())

	// Regenerate README.
	updatedMonths, _ := ReadStatsCSV(cfg.StatsCSVPath())
	readmeBytes, _ := GenerateREADME(t.opts.ReadmeTmpl, updatedMonths, nil)
	if readmeBytes != nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	// Build HF commit: delete today files + add monthly + metadata.
	var ops []HFOp
	for _, f := range todayFiles {
		base := filepath.Base(f)
		ops = append(ops, HFOp{PathInRepo: "today/" + base, Delete: true})
	}
	ops = append(ops,
		HFOp{LocalPath: monthPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", year, year, month)},
		HFOp{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
		HFOp{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
		HFOp{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
	)
	msg := fmt.Sprintf("Merge %s → data/%04d/%04d-%02d.parquet (%s items)", prevDate, year, year, month, fmtInt(count))
	commitURL, err := t.opts.HFCommit(ctx, ops, msg)
	if err != nil {
		return metric, fmt.Errorf("hf rollover commit: %w", err)
	}
	metric.CommitURL = commitURL

	// Delete local today files after confirmed successful commit.
	for _, f := range todayFiles {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			// Log but don't fail — files are inert at this point.
			fmt.Fprintf(os.Stderr, "warn: remove local today file %s: %v\n", f, err)
		} else {
			metric.FilesPruned++
		}
	}
	return metric, nil
}

func buildParquetList(paths []string) string {
	if len(paths) == 1 {
		return "'" + escapeSQLStr(paths[0]) + "'"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, p := range paths {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("'")
		sb.WriteString(escapeSQLStr(p))
		sb.WriteString("'")
	}
	sb.WriteString("]")
	return sb.String()
}

func fileExistsNE(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir() && fi.Size() > 0
}
