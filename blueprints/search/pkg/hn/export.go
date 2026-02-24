package hn

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

type ExportOptions struct {
	DBPath           string
	OutDir           string
	FromMonth        string // YYYY-MM
	ToMonth          string // YYYY-MM
	Force            bool
	RefreshLatest    bool
	CompressionLevel int
}

type ExportMonth struct {
	Month     string
	Rows      int64
	Path      string
	Skipped   bool
	Refreshed bool
	Size      int64
}

type ExportResult struct {
	DBPath        string
	OutDir        string
	LatestMonth   string
	MonthsScanned int
	MonthsWritten int
	MonthsSkipped int
	RowsWritten   int64
	BytesWritten  int64
	Elapsed       time.Duration
	Months        []ExportMonth
}

func (c Config) ExportMonthlyParquet(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	cfg := c.WithDefaults()
	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = cfg.DefaultDBPath()
	}
	outDir := opts.OutDir
	if strings.TrimSpace(outDir) == "" {
		outDir = filepath.Join(cfg.BaseDir(), "export", "hn", "monthly")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create export dir: %w", err)
	}
	compLevel := opts.CompressionLevel
	if compLevel <= 0 {
		compLevel = 22
	}
	fromMonth, err := parseYYYYMM(opts.FromMonth)
	if err != nil {
		return nil, fmt.Errorf("parse --from-month: %w", err)
	}
	toMonth, err := parseYYYYMM(opts.ToMonth)
	if err != nil {
		return nil, fmt.Errorf("parse --to-month: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()
	_, _ = db.ExecContext(ctx, `SET preserve_insertion_order=false`)
	_, _ = db.ExecContext(ctx, `SET threads=4`)

	type monthRow struct {
		Month string
		Rows  int64
	}
	var months []monthRow
	rs, err := db.QueryContext(ctx, `SELECT strftime(time_ts, '%Y-%m') AS ym, COUNT(*)::BIGINT AS n
FROM items
WHERE time_ts IS NOT NULL
GROUP BY 1
ORDER BY 1`)
	if err != nil {
		return nil, fmt.Errorf("list item months: %w", err)
	}
	for rs.Next() {
		var m monthRow
		if err := rs.Scan(&m.Month, &m.Rows); err != nil {
			rs.Close()
			return nil, fmt.Errorf("scan item month: %w", err)
		}
		months = append(months, m)
	}
	rs.Close()
	if len(months) == 0 {
		return &ExportResult{DBPath: dbPath, OutDir: outDir}, nil
	}
	latestMonth := months[len(months)-1].Month
	started := time.Now()
	res := &ExportResult{DBPath: dbPath, OutDir: outDir, LatestMonth: latestMonth}

	for _, m := range months {
		tm, err := time.Parse("2006-01", m.Month)
		if err != nil {
			continue
		}
		if !fromMonth.IsZero() && tm.Before(fromMonth) {
			continue
		}
		if !toMonth.IsZero() && tm.After(toMonth) {
			continue
		}
		res.MonthsScanned++
		outPath := filepath.Join(outDir, fmt.Sprintf("items_%s.parquet", strings.ReplaceAll(m.Month, "-", "_")))
		exists := fileExistsNonEmpty(outPath)
		refreshingLatest := opts.RefreshLatest && m.Month == latestMonth && exists
		if exists && !opts.Force && !refreshingLatest {
			res.MonthsSkipped++
			res.Months = append(res.Months, ExportMonth{
				Month:   m.Month,
				Rows:    m.Rows,
				Path:    outPath,
				Skipped: true,
				Size:    ternaryFileSize(outPath),
			})
			continue
		}

		nextMonth := tm.AddDate(0, 1, 0)
		tmpPath := outPath + ".tmp"
		_ = os.Remove(tmpPath)
		copySQL := fmt.Sprintf(`COPY (
  SELECT *
  FROM items
  WHERE time_ts >= TIMESTAMP '%s-01 00:00:00'
    AND time_ts < TIMESTAMP '%s-01 00:00:00'
  ORDER BY id
) TO '%s' (FORMAT PARQUET, COMPRESSION zstd, COMPRESSION_LEVEL %d)`,
			m.Month,
			nextMonth.Format("2006-01"),
			escapeSQLString(tmpPath),
			compLevel,
		)
		if _, err := db.ExecContext(ctx, copySQL); err != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("export month %s: %w", m.Month, err)
		}
		if err := os.Rename(tmpPath, outPath); err != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("rename month export %s: %w", m.Month, err)
		}
		sz, _ := fileSize(outPath)
		res.MonthsWritten++
		res.RowsWritten += m.Rows
		res.BytesWritten += sz
		res.Months = append(res.Months, ExportMonth{
			Month:     m.Month,
			Rows:      m.Rows,
			Path:      outPath,
			Refreshed: refreshingLatest,
			Size:      sz,
		})
	}

	res.Elapsed = time.Since(started)
	return res, nil
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
