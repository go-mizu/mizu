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

// FetchMonth downloads all items for the given year/month as a Parquet file to outPath.
// The file is written atomically (unique temp → outPath). Returns Count==0 when the
// remote has no data for the month.
func (c Config) FetchMonth(ctx context.Context, year, month int, outPath string) (FetchResult, error) {
	cfg := c.resolved()
	start := time.Now()
	tm := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	next := tm.AddDate(0, 1, 0)
	q := fmt.Sprintf(
		`SELECT * FROM %s WHERE time >= toDateTime('%s') AND time < toDateTime('%s') ORDER BY id FORMAT Parquet`,
		cfg.fqTable(),
		tm.Format("2006-01-02 15:04:05"),
		next.Format("2006-01-02 15:04:05"),
	)
	if err := ensureParentDir(outPath); err != nil {
		return FetchResult{}, fmt.Errorf("create month dir: %w", err)
	}
	n, err := cfg.streamToFile(ctx, q, outPath)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch month %04d-%02d: %w", year, month, err)
	}
	if n == 0 {
		return FetchResult{Duration: time.Since(start)}, nil
	}
	return cfg.scanParquetResult(ctx, outPath, n, time.Since(start))
}

// FetchSince downloads all items with id > afterID and time < ceilTime as a Parquet file.
// ceilTime bounds the query to prevent items from crossing midnight into the next day's block.
func (c Config) FetchSince(ctx context.Context, afterID int64, ceilTime time.Time, outPath string) (FetchResult, error) {
	cfg := c.resolved()
	start := time.Now()
	q := fmt.Sprintf(
		`SELECT * FROM %s WHERE id > %d AND time < toDateTime('%s') ORDER BY id FORMAT Parquet`,
		cfg.fqTable(),
		afterID,
		ceilTime.UTC().Format("2006-01-02 15:04:05"),
	)
	if err := ensureParentDir(outPath); err != nil {
		return FetchResult{}, fmt.Errorf("create today dir: %w", err)
	}
	n, err := cfg.streamToFile(ctx, q, outPath)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch since id=%d: %w", afterID, err)
	}
	if n == 0 {
		return FetchResult{Duration: time.Since(start)}, nil
	}
	return cfg.scanParquetResult(ctx, outPath, n, time.Since(start))
}

// streamToFile executes q against the ClickHouse endpoint and writes the response
// body to outPath atomically (unique temp file → rename). Returns bytes written.
//
// Uses os.CreateTemp so concurrent processes fetching the same outPath (e.g.
// hn-publish and hn-publish-live both running backfill) do not race on a shared
// ".tmp" filename.
func (c Config) streamToFile(ctx context.Context, q, outPath string) (int64, error) {
	cfg := c.resolved()
	var lastErr error
	for attempt := 1; attempt <= 4; attempt++ {
		if attempt > 1 {
			fmt.Fprintf(os.Stderr, "info: streamToFile retry %d/4 for %s\n", attempt, filepath.Base(outPath))
			sleepWithContext(ctx, time.Duration(attempt)*500*time.Millisecond)
		}
		req, err := cfg.newRequest(ctx, q)
		if err != nil {
			return 0, err
		}
		resp, err := cfg.downloadHTTPClient().Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt, err)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				return 0, lastErr // non-retryable
			}
			continue
		}
		n, err := writeResponseToFile(resp.Body, outPath)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		return n, nil
	}
	return 0, lastErr
}

// writeResponseToFile streams body into a unique temp file in the same directory
// as outPath, then atomically renames it to outPath. Returns bytes written.
// Returns 0 (no error) if the body is empty.
func writeResponseToFile(body io.Reader, outPath string) (int64, error) {
	f, err := os.CreateTemp(filepath.Dir(outPath), ".hn-fetch-*.tmp")
	if err != nil {
		return 0, fmt.Errorf("create tmp file: %w", err)
	}
	tmp := f.Name()
	n, copyErr := io.Copy(f, body)
	closeErr := f.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return 0, copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return 0, closeErr
	}
	if n == 0 {
		os.Remove(tmp)
		return 0, nil
	}
	if err := os.Rename(tmp, outPath); err != nil {
		os.Remove(tmp)
		return 0, fmt.Errorf("rename to output: %w", err)
	}
	return n, nil
}

// scanParquetResult reads COUNT/MIN(id)/MAX(id) from a Parquet file via DuckDB.
func (c Config) scanParquetResult(ctx context.Context, path string, bytesWritten int64, dur time.Duration) (FetchResult, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return FetchResult{}, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()
	q := fmt.Sprintf(`SELECT COUNT(*)::BIGINT, MIN(id)::BIGINT, MAX(id)::BIGINT FROM read_parquet('%s')`,
		escapeSQLStr(path))
	var count, minID, maxID int64
	if err := db.QueryRowContext(ctx, q).Scan(&count, &minID, &maxID); err != nil {
		return FetchResult{}, fmt.Errorf("scan parquet: %w", err)
	}
	return FetchResult{
		LowestID:  minID,
		HighestID: maxID,
		Count:     count,
		Bytes:     bytesWritten,
		Duration:  dur,
	}, nil
}

// ensureParentDir creates the parent directory of path if it does not exist.
func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

// escapeSQLStr escapes single quotes for embedding a string in a SQL literal.
func escapeSQLStr(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// sleepWithContext sleeps for d or until ctx is cancelled.
func sleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
