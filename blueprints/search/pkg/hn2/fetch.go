package hn2

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// FetchResult is returned by FetchMonth and FetchSince.
type FetchResult struct {
	LowestID  int64
	HighestID int64
	Count     int64
	Bytes     int64
	Duration  time.Duration
}

// FetchMonth downloads all items for the given year/month as a parquet file to outPath.
// The file is written atomically (outPath.tmp → outPath). If Count==0 the .tmp is removed
// and the caller should skip committing.
func (c Config) FetchMonth(ctx context.Context, year, month int, outPath string) (FetchResult, error) {
	cfg := c.WithDefaults()
	start := time.Now()
	tm := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	next := tm.AddDate(0, 1, 0)
	q := fmt.Sprintf(
		`SELECT * FROM %s WHERE time >= toDateTime('%s') AND time < toDateTime('%s') ORDER BY id FORMAT Parquet`,
		cfg.fqTable(),
		tm.Format("2006-01-02 15:04:05"),
		next.Format("2006-01-02 15:04:05"),
	)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return FetchResult{}, fmt.Errorf("create month dir: %w", err)
	}
	n, err := cfg.streamToFile(ctx, q, outPath)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch month %04d-%02d: %w", year, month, err)
	}
	dur := time.Since(start)
	if n == 0 {
		return FetchResult{Duration: dur}, nil
	}
	return cfg.scanParquetResult(ctx, outPath, n, dur)
}

// FetchSince downloads all items with id > afterID and time < ceilTime as a parquet file.
// ceilTime prevents items from crossing midnight into the next day's block.
func (c Config) FetchSince(ctx context.Context, afterID int64, ceilTime time.Time, outPath string) (FetchResult, error) {
	cfg := c.WithDefaults()
	start := time.Now()
	q := fmt.Sprintf(
		`SELECT * FROM %s WHERE id > %d AND time < toDateTime('%s') ORDER BY id FORMAT Parquet`,
		cfg.fqTable(),
		afterID,
		ceilTime.UTC().Format("2006-01-02 15:04:05"),
	)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return FetchResult{}, fmt.Errorf("create today dir: %w", err)
	}
	n, err := cfg.streamToFile(ctx, q, outPath)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch since %d: %w", afterID, err)
	}
	dur := time.Since(start)
	if n == 0 {
		return FetchResult{Duration: dur}, nil
	}
	return cfg.scanParquetResult(ctx, outPath, n, dur)
}

// streamToFile executes the query and streams the response body to outPath atomically.
// Returns bytes written. If the response is empty (0 bytes), removes the .tmp file and returns 0.
func (c Config) streamToFile(ctx context.Context, q, outPath string) (int64, error) {
	cfg := c.WithDefaults()
	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)

	var lastErr error
	for attempt := 1; attempt <= 4; attempt++ {
		req, err := cfg.newRequest(ctx, q)
		if err != nil {
			return 0, err
		}
		resp, err := cfg.downloadHTTPClient().Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request attempt %d: %w", attempt, err)
			sleepWithContext(ctx, time.Duration(attempt)*500*time.Millisecond)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				return 0, lastErr
			}
			sleepWithContext(ctx, time.Duration(attempt)*500*time.Millisecond)
			continue
		}
		f, err := os.Create(tmpPath)
		if err != nil {
			resp.Body.Close()
			return 0, fmt.Errorf("create tmp file: %w", err)
		}
		n, copyErr := io.Copy(f, resp.Body)
		bodyCloseErr := resp.Body.Close()
		fileCloseErr := f.Close()
		if copyErr != nil || bodyCloseErr != nil || fileCloseErr != nil {
			_ = os.Remove(tmpPath)
			if copyErr != nil {
				lastErr = copyErr
			} else if bodyCloseErr != nil {
				lastErr = bodyCloseErr
			} else {
				lastErr = fileCloseErr
			}
			sleepWithContext(ctx, time.Duration(attempt)*500*time.Millisecond)
			continue
		}
		if n == 0 {
			_ = os.Remove(tmpPath)
			return 0, nil
		}
		if err := os.Rename(tmpPath, outPath); err != nil {
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("rename to output: %w", err)
		}
		return n, nil
	}
	_ = os.Remove(tmpPath)
	return 0, lastErr
}

// scanParquetResult reads MIN/MAX/COUNT from a parquet file via DuckDB.
func (c Config) scanParquetResult(ctx context.Context, path string, bytesWritten int64, dur time.Duration) (FetchResult, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return FetchResult{}, fmt.Errorf("open duckdb for scan: %w", err)
	}
	defer db.Close()
	q := fmt.Sprintf(`SELECT COUNT(*)::BIGINT, MIN(id)::BIGINT, MAX(id)::BIGINT FROM read_parquet('%s')`,
		escapeSQLStr(path))
	var count, minID, maxID int64
	if err := db.QueryRowContext(ctx, q).Scan(&count, &minID, &maxID); err != nil {
		return FetchResult{}, fmt.Errorf("scan parquet result: %w", err)
	}
	return FetchResult{
		LowestID:  minID,
		HighestID: maxID,
		Count:     count,
		Bytes:     bytesWritten,
		Duration:  dur,
	}, nil
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func escapeSQLStr(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
