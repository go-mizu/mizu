package hn

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type ExportOptions struct {
	OutDir        string
	FromMonth     string // YYYY-MM
	ToMonth       string // YYYY-MM
	Force         bool
	RefreshLatest bool
	Progress      func(ExportProgress)
}

type ExportMonth struct {
	Month      string
	Rows       int64
	Path       string
	Skipped    bool
	Refreshed  bool
	SkipReason string
	Size       int64
}

type ExportResult struct {
	OutDir        string
	SourceUsed    string
	SourceDetail  string
	LatestMonth   string
	MonthsScanned int
	MonthsWritten int
	MonthsSkipped int
	RowsWritten   int64
	BytesWritten  int64
	Elapsed       time.Duration
	Months        []ExportMonth
}

type ExportProgress struct {
	Stage        string
	SourceUsed   string
	SourceDetail string
	OutDir       string
	Month        string
	MonthIndex   int
	MonthTotal   int
	Rows         int64
	Path         string
	Skipped      bool
	SkipReason   string
	Done         bool
}

type exportSourceSpec struct {
	kind       string
	detail     string
	months     []ExportMonth
	writeMonth func(context.Context, string, string) error
}

func (c Config) ExportMonthlyParquet(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	cfg := c.WithDefaults()
	outDir := strings.TrimSpace(opts.OutDir)
	if outDir == "" {
		outDir = filepath.Join(cfg.BaseDir(), "export", "hn", "monthly")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create export dir: %w", err)
	}
	fromMonth, err := parseYYYYMM(opts.FromMonth)
	if err != nil {
		return nil, fmt.Errorf("parse from month: %w", err)
	}
	toMonth, err := parseYYYYMM(opts.ToMonth)
	if err != nil {
		return nil, fmt.Errorf("parse to month: %w", err)
	}

	src, err := cfg.resolveExportSource(ctx)
	if err != nil {
		return nil, err
	}
	res := &ExportResult{
		OutDir:       outDir,
		SourceUsed:   src.kind,
		SourceDetail: src.detail,
	}
	if opts.Progress != nil {
		opts.Progress(ExportProgress{
			Stage:        "start",
			SourceUsed:   src.kind,
			SourceDetail: src.detail,
			OutDir:       outDir,
		})
	}
	if len(src.months) == 0 {
		if opts.Progress != nil {
			opts.Progress(ExportProgress{
				Stage:        "done",
				SourceUsed:   src.kind,
				SourceDetail: src.detail,
				OutDir:       outDir,
				Done:         true,
			})
		}
		return res, nil
	}
	res.LatestMonth = src.months[len(src.months)-1].Month
	started := time.Now()
	selectedTotal := 0
	for _, m := range src.months {
		tm, _ := time.Parse("2006-01", m.Month)
		if !fromMonth.IsZero() && tm.Before(fromMonth) {
			continue
		}
		if !toMonth.IsZero() && tm.After(toMonth) {
			continue
		}
		selectedTotal++
	}
	monthIdx := 0

	for _, m := range src.months {
		tm, _ := time.Parse("2006-01", m.Month)
		if !fromMonth.IsZero() && tm.Before(fromMonth) {
			continue
		}
		if !toMonth.IsZero() && tm.After(toMonth) {
			continue
		}
		res.MonthsScanned++
		monthIdx++
		outPath := filepath.Join(outDir, fmt.Sprintf("items_%s.parquet", strings.ReplaceAll(m.Month, "-", "_")))
		exists := fileExistsNonEmpty(outPath)
		isLatest := m.Month == res.LatestMonth
		refreshLatest := opts.RefreshLatest && exists && isLatest
		latestUnchanged := false
		if exists && !opts.Force && isLatest {
			if exportedRows, ok := countParquetRows(ctx, outPath); ok && exportedRows == m.Rows {
				latestUnchanged = true
			}
		}
		if opts.Progress != nil {
			opts.Progress(ExportProgress{
				Stage:        "month_start",
				SourceUsed:   src.kind,
				SourceDetail: src.detail,
				OutDir:       outDir,
				Month:        m.Month,
				MonthIndex:   monthIdx,
				MonthTotal:   selectedTotal,
				Rows:         m.Rows,
				Path:         outPath,
			})
		}
		if exists && !opts.Force && (!refreshLatest || latestUnchanged) {
			reason := "exists"
			if isLatest && latestUnchanged {
				reason = "latest_unchanged"
			}
			res.MonthsSkipped++
			em := ExportMonth{Month: m.Month, Rows: m.Rows, Path: outPath, Skipped: true, SkipReason: reason, Size: ternaryFileSize(outPath)}
			res.Months = append(res.Months, em)
			if opts.Progress != nil {
				opts.Progress(ExportProgress{
					Stage:        "month_done",
					SourceUsed:   src.kind,
					SourceDetail: src.detail,
					OutDir:       outDir,
					Month:        m.Month,
					MonthIndex:   monthIdx,
					MonthTotal:   selectedTotal,
					Rows:         m.Rows,
					Path:         outPath,
					Skipped:      true,
					SkipReason:   reason,
				})
			}
			continue
		}
		if err := src.writeMonth(ctx, m.Month, outPath); err != nil {
			return nil, err
		}
		sz, _ := fileSize(outPath)
		res.MonthsWritten++
		res.RowsWritten += m.Rows
		res.BytesWritten += sz
		em := ExportMonth{Month: m.Month, Rows: m.Rows, Path: outPath, Refreshed: refreshLatest, Size: sz}
		res.Months = append(res.Months, em)
		if opts.Progress != nil {
			opts.Progress(ExportProgress{
				Stage:        "month_done",
				SourceUsed:   src.kind,
				SourceDetail: src.detail,
				OutDir:       outDir,
				Month:        m.Month,
				MonthIndex:   monthIdx,
				MonthTotal:   selectedTotal,
				Rows:         m.Rows,
				Path:         outPath,
			})
		}
	}
	res.Elapsed = time.Since(started)
	if opts.Progress != nil {
		opts.Progress(ExportProgress{
			Stage:        "done",
			SourceUsed:   src.kind,
			SourceDetail: src.detail,
			OutDir:       outDir,
			Done:         true,
		})
	}
	return res, nil
}

func (c Config) resolveExportSource(ctx context.Context) (*exportSourceSpec, error) {
	cfg := c.WithDefaults()
	dbPath := cfg.DefaultDBPath()
	if fileExistsNonEmpty(dbPath) {
		if months, err := cfg.localDuckDBMonths(ctx, dbPath); err == nil && len(months) > 0 {
			return &exportSourceSpec{
				kind:   "duckdb",
				detail: dbPath,
				months: months,
				writeMonth: func(ctx context.Context, month, outPath string) error {
					return cfg.exportMonthFromDuckDB(ctx, dbPath, month, outPath)
				},
			}, nil
		}
	}
	months, err := cfg.clickHouseMonths(ctx)
	if err != nil {
		return nil, err
	}
	return &exportSourceSpec{
		kind:   "clickhouse",
		detail: cfg.ClickHouseBaseURL,
		months: months,
		writeMonth: func(ctx context.Context, month, outPath string) error {
			return cfg.downloadClickHouseMonthParquet(ctx, month, outPath)
		},
	}, nil
}

type clickHouseMonthRow struct {
	Month string `json:"ym"`
	Rows  any    `json:"n"`
}

func (c Config) clickHouseMonths(ctx context.Context) ([]ExportMonth, error) {
	cfg := c.WithDefaults()
	q := fmt.Sprintf(`SELECT formatDateTime(toStartOfMonth(time), '%%Y-%%m') AS ym, toInt64(count()) AS n
FROM %s
WHERE time IS NOT NULL
GROUP BY ym
ORDER BY ym
FORMAT JSONEachRow`, cfg.clickHouseFQTable())
	body, err := cfg.clickHouseQuery(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("clickhouse list months: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	out := make([]ExportMonth, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var r clickHouseMonthRow
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("decode clickhouse month row: %w", err)
		}
		n, err := parseInt64Any(r.Rows)
		if err != nil {
			return nil, fmt.Errorf("parse clickhouse month row count: %w", err)
		}
		out = append(out, ExportMonth{Month: r.Month, Rows: n})
	}
	return out, nil
}

func (c Config) localDuckDBMonths(ctx context.Context, dbPath string) ([]ExportMonth, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open duckdb for export source discovery: %w", err)
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT strftime(time_ts, '%Y-%m') AS ym, COUNT(*)::BIGINT AS n
FROM items
WHERE time_ts IS NOT NULL
GROUP BY 1
ORDER BY 1`)
	if err != nil {
		return nil, fmt.Errorf("list months from duckdb: %w", err)
	}
	defer rows.Close()
	var out []ExportMonth
	for rows.Next() {
		var m string
		var n int64
		if err := rows.Scan(&m, &n); err != nil {
			return nil, fmt.Errorf("scan duckdb month row: %w", err)
		}
		out = append(out, ExportMonth{Month: m, Rows: n})
	}
	return out, rows.Err()
}

func (c Config) downloadClickHouseMonthParquet(ctx context.Context, month string, outPath string) error {
	cfg := c.WithDefaults()
	tm, err := time.Parse("2006-01", month)
	if err != nil {
		return fmt.Errorf("parse month %q: %w", month, err)
	}
	next := tm.AddDate(0, 1, 0)
	query := fmt.Sprintf(`SELECT * FROM %s
WHERE time >= toDateTime('%s-01 00:00:00')
  AND time < toDateTime('%s-01 00:00:00')
ORDER BY id
FORMAT Parquet`, cfg.clickHouseFQTable(), tm.Format("2006-01"), next.Format("2006-01"))

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create export month dir: %w", err)
	}
	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)
	req, err := cfg.newClickHouseRequest(ctx, query)
	if err != nil {
		return err
	}
	resp, err := cfg.clickHouseDownloadHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("clickhouse month export %s request: %w", month, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("clickhouse month export %s returned %d: %s", month, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create month parquet tmp file: %w", err)
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		if copyErr != nil {
			return fmt.Errorf("write month parquet %s: %w", month, copyErr)
		}
		return fmt.Errorf("close month parquet %s: %w", month, closeErr)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename month parquet %s: %w", month, err)
	}
	return nil
}

func (c Config) exportMonthFromDuckDB(ctx context.Context, dbPath, month, outPath string) error {
	tm, err := time.Parse("2006-01", month)
	if err != nil {
		return fmt.Errorf("parse month %q: %w", month, err)
	}
	next := tm.AddDate(0, 1, 0)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create export month dir: %w", err)
	}
	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open duckdb for export: %w", err)
	}
	defer db.Close()
	query := fmt.Sprintf(`COPY (
SELECT * FROM items
WHERE time_ts >= TIMESTAMP '%s'
  AND time_ts < TIMESTAMP '%s'
ORDER BY id
) TO '%s' (FORMAT PARQUET, COMPRESSION zstd, COMPRESSION_LEVEL 22)`,
		tm.Format("2006-01-02 15:04:05"),
		next.Format("2006-01-02 15:04:05"),
		escapeSQLString(tmpPath),
	)
	if _, err := db.ExecContext(ctx, query); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("duckdb export month %s: %w", month, err)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename month parquet %s: %w", month, err)
	}
	return nil
}

func countParquetRows(ctx context.Context, path string) (int64, bool) {
	if !fileExistsNonEmpty(path) {
		return 0, false
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return 0, false
	}
	defer db.Close()
	var n int64
	q := fmt.Sprintf(`SELECT COUNT(*)::BIGINT FROM read_parquet('%s')`, escapeSQLString(path))
	if err := db.QueryRowContext(ctx, q).Scan(&n); err != nil {
		return 0, false
	}
	return n, true
}

func parseYYYYMM(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01", s)
}

func ternaryFileSize(path string) int64 {
	sz, _ := fileSize(path)
	return sz
}
