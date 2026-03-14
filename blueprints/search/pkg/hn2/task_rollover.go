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

// RolloverState is emitted by DayRolloverTask during execution.
type RolloverState struct {
	Phase      string // "merge" | "commit"
	PrevDate   string
	FilesFound int
	RowsMerged int64
}

// RolloverMetric is the result returned when DayRolloverTask.Run completes.
type RolloverMetric struct {
	PrevDate    string
	MonthPath   string
	RowsMerged  int64
	FilesPruned int
	CommitURL   string
}

// RolloverTaskOptions configures a day rollover run.
type RolloverTaskOptions struct {
	PrevDate   string     // YYYY-MM-DD: the day whose today/ blocks should be merged
	HFCommit  CommitFn   // required: commits files to Hugging Face
	ReadmeTmpl []byte    // required: README.md Go template
	Analytics  *Analytics // optional: enriches README with source-level stats
}

// DayRolloverTask merges a day's live 5-minute blocks into the monthly Parquet
// file and commits the result to Hugging Face, removing the individual blocks.
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

	todayFiles, _ := filepath.Glob(filepath.Join(cfg.TodayDir(), prevDate+"_*.parquet"))

	if emit != nil {
		emit(&RolloverState{Phase: "merge", PrevDate: prevDate, FilesFound: len(todayFiles)})
	}

	if len(todayFiles) == 0 && !fileExists(monthPath) {
		return metric, nil // nothing to do
	}

	// Open a single DuckDB connection for both merge and scan.
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return metric, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	if len(todayFiles) > 0 {
		if err := mergeParquetFiles(ctx, db, monthPath, todayFiles); err != nil {
			return metric, err
		}
	}

	count, minID, maxID, err := scanParquet(ctx, db, monthPath)
	if err != nil {
		return metric, fmt.Errorf("scan merged parquet: %w", err)
	}
	metric.RowsMerged = count

	if emit != nil {
		emit(&RolloverState{Phase: "commit", PrevDate: prevDate, FilesFound: len(todayFiles), RowsMerged: count})
	}

	// Update stats files.
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
	_ = WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newMonthRow, true)
	_ = ClearStatsTodayCSV(cfg.StatsTodayCSVPath())

	// Regenerate README.
	updatedMonths, _ := ReadStatsCSV(cfg.StatsCSVPath())
	if readmeBytes, err := GenerateREADME(t.opts.ReadmeTmpl, updatedMonths, nil, t.opts.Analytics); err == nil && readmeBytes != nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	// Build HF commit: delete today/ blocks, upsert monthly parquet and metadata.
	ops := make([]HFOp, 0, len(todayFiles)+4)
	for _, f := range todayFiles {
		ops = append(ops, HFOp{PathInRepo: "today/" + filepath.Base(f), Delete: true})
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

	// Remove local today/ files after confirmed commit.
	// If removal fails, the files are inert — the repo no longer references them.
	for _, f := range todayFiles {
		if err := os.Remove(f); err == nil {
			metric.FilesPruned++
		}
	}
	return metric, nil
}

// mergeParquetFiles merges existing monthly parquet (if any) with todayFiles
// into monthPath, sorted by id, using Zstandard compression level 22.
func mergeParquetFiles(ctx context.Context, db *sql.DB, monthPath string, todayFiles []string) error {
	var sources []string
	if fileExists(monthPath) {
		sources = append(sources, monthPath)
	}
	sources = append(sources, todayFiles...)

	if err := ensureParentDir(monthPath); err != nil {
		return fmt.Errorf("create month dir: %w", err)
	}
	tmp := monthPath + ".merge.tmp"
	defer os.Remove(tmp) // clean up on any error path

	mergeQ := fmt.Sprintf(
		`COPY (SELECT * FROM read_parquet(%s) ORDER BY id) TO '%s' (FORMAT PARQUET, COMPRESSION zstd, COMPRESSION_LEVEL 22)`,
		parquetList(sources), escapeSQLStr(tmp),
	)
	if _, err := db.ExecContext(ctx, mergeQ); err != nil {
		return fmt.Errorf("merge parquet: %w", err)
	}
	if err := os.Rename(tmp, monthPath); err != nil {
		return fmt.Errorf("rename merged parquet: %w", err)
	}
	return nil
}

// scanParquet returns COUNT(*), MIN(id), MAX(id) for the given Parquet file.
// Uses NullInt64 for min/max because an empty Parquet file returns NULL for aggregates.
func scanParquet(ctx context.Context, db *sql.DB, path string) (count, minID, maxID int64, err error) {
	q := fmt.Sprintf(`SELECT COUNT(*)::BIGINT, MIN(id)::BIGINT, MAX(id)::BIGINT FROM read_parquet('%s')`, escapeSQLStr(path))
	var nMin, nMax sql.NullInt64
	err = db.QueryRowContext(ctx, q).Scan(&count, &nMin, &nMax)
	minID, maxID = nMin.Int64, nMax.Int64
	return
}

// parquetList formats a slice of file paths as a DuckDB read_parquet argument:
// a single-quoted string for one file, or a bracket list for multiple.
func parquetList(paths []string) string {
	if len(paths) == 1 {
		return "'" + escapeSQLStr(paths[0]) + "'"
	}
	var sb strings.Builder
	sb.WriteByte('[')
	for i, p := range paths {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteByte('\'')
		sb.WriteString(escapeSQLStr(p))
		sb.WriteByte('\'')
	}
	sb.WriteByte(']')
	return sb.String()
}

// fileExists reports whether path exists, is a regular file, and is non-empty.
func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir() && fi.Size() > 0
}
