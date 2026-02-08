package reddit

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/klauspost/compress/zstd"
)

// ImportProgress reports import progress.
type ImportProgress struct {
	Phase   string        // "decompress", "import", "parquet", "index"
	Rows    int64         // Rows processed so far
	Bytes   int64         // Bytes processed (decompress phase)
	Elapsed time.Duration // Time since start
	Done    bool          // Phase complete
	Detail  string        // Extra info (file path, etc.)
}

// ImportCallback is called with import progress updates.
type ImportCallback func(ImportProgress)

// Import reads a zst-compressed ndjson file, decompresses it, imports into
// DuckDB, and exports to parquet. The Reddit archive uses a 2GB zstd window
// which DuckDB's built-in decompressor can't handle, so we decompress first.
func Import(ctx context.Context, file DataFile, cb ImportCallback) error {
	// Verify source file exists
	if _, err := os.Stat(file.ZstPath); err != nil {
		return fmt.Errorf("source file not found: %s", file.ZstPath)
	}

	// Create output directories
	if err := os.MkdirAll(filepath.Dir(file.DBPath), 0o755); err != nil {
		return fmt.Errorf("create database dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(file.PQPath), 0o755); err != nil {
		return fmt.Errorf("create parquet dir: %w", err)
	}

	// Remove existing files to start fresh
	os.Remove(file.DBPath)
	os.Remove(file.PQPath)

	start := time.Now()

	// Phase 1: Decompress zst → ndjson temp file
	// Reddit archive uses 2GB zstd window which DuckDB can't handle natively.
	if cb != nil {
		cb(ImportProgress{Phase: "decompress", Detail: file.ZstPath, Elapsed: time.Since(start)})
	}

	ndjsonPath, bytesWritten, err := decompressZst(ctx, file.ZstPath, cb, start)
	if err != nil {
		return fmt.Errorf("decompress: %w", err)
	}
	defer os.Remove(ndjsonPath)

	if cb != nil {
		cb(ImportProgress{Phase: "decompress", Bytes: bytesWritten, Done: true, Elapsed: time.Since(start)})
	}

	// Phase 2: Import ndjson into DuckDB
	if cb != nil {
		cb(ImportProgress{Phase: "import", Detail: ndjsonPath, Elapsed: time.Since(start)})
	}

	db, err := sql.Open("duckdb", file.DBPath)
	if err != nil {
		return fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	tableName := "comments"
	if file.Kind == Submissions {
		tableName = "submissions"
	}

	importQuery := buildImportQuery(tableName, file.Kind, ndjsonPath)
	if _, err := db.ExecContext(ctx, importQuery); err != nil {
		return fmt.Errorf("import ndjson: %w", err)
	}

	// Get row count
	var rowCount int64
	if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount); err != nil {
		return fmt.Errorf("count rows: %w", err)
	}

	if cb != nil {
		cb(ImportProgress{Phase: "import", Rows: rowCount, Done: true, Elapsed: time.Since(start)})
	}

	// Phase 3: Export to parquet
	if cb != nil {
		cb(ImportProgress{Phase: "parquet", Rows: rowCount, Detail: file.PQPath, Elapsed: time.Since(start)})
	}

	escapedPQ := strings.ReplaceAll(file.PQPath, "'", "''")
	exportQuery := fmt.Sprintf(
		"COPY %s TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD)",
		tableName, escapedPQ,
	)
	if _, err := db.ExecContext(ctx, exportQuery); err != nil {
		return fmt.Errorf("export parquet: %w", err)
	}

	if cb != nil {
		cb(ImportProgress{Phase: "parquet", Rows: rowCount, Done: true, Elapsed: time.Since(start)})
	}

	// Phase 4: Create indexes
	if cb != nil {
		cb(ImportProgress{Phase: "index", Rows: rowCount, Elapsed: time.Since(start)})
	}

	indexCols := []string{"subreddit", "author"}
	for _, col := range indexCols {
		idx := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s ON %s(%s)", col, tableName, col)
		if _, err := db.ExecContext(ctx, idx); err != nil {
			continue
		}
	}

	if cb != nil {
		cb(ImportProgress{Phase: "index", Rows: rowCount, Done: true, Elapsed: time.Since(start)})
	}

	return nil
}

// decompressZst decompresses a .zst file to a temp .ndjson file.
// Uses klauspost/compress with large window support (2GB).
func decompressZst(ctx context.Context, zstPath string, cb ImportCallback, start time.Time) (string, int64, error) {
	in, err := os.Open(zstPath)
	if err != nil {
		return "", 0, fmt.Errorf("open zst: %w", err)
	}
	defer in.Close()

	// klauspost/compress zstd decoder with max window size (2GB)
	dec, err := zstd.NewReader(in, zstd.WithDecoderMaxWindow(1<<31))
	if err != nil {
		return "", 0, fmt.Errorf("create zstd decoder: %w", err)
	}
	defer dec.Close()

	// Create temp file for decompressed ndjson
	tmpFile, err := os.CreateTemp("", "reddit-*.ndjson")
	if err != nil {
		return "", 0, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Copy with progress tracking
	var written int64
	buf := make([]byte, 4*1024*1024) // 4MB buffer
	lastReport := time.Now()

	for {
		select {
		case <-ctx.Done():
			tmpFile.Close()
			os.Remove(tmpPath)
			return "", 0, ctx.Err()
		default:
		}

		n, readErr := dec.Read(buf)
		if n > 0 {
			if _, err := tmpFile.Write(buf[:n]); err != nil {
				tmpFile.Close()
				os.Remove(tmpPath)
				return "", 0, fmt.Errorf("write temp: %w", err)
			}
			written += int64(n)

			if cb != nil && time.Since(lastReport) > 500*time.Millisecond {
				lastReport = time.Now()
				cb(ImportProgress{
					Phase:   "decompress",
					Bytes:   written,
					Elapsed: time.Since(start),
					Detail:  fmt.Sprintf("%s decompressed", formatSize(written)),
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return "", 0, fmt.Errorf("read zst: %w", readErr)
		}
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", 0, fmt.Errorf("close temp: %w", err)
	}

	return tmpPath, written, nil
}

func formatSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// buildImportQuery constructs the CREATE TABLE AS SELECT query for importing
// Reddit ndjson data from a decompressed (plain) ndjson file.
func buildImportQuery(tableName string, kind FileKind, ndjsonPath string) string {
	escaped := strings.ReplaceAll(ndjsonPath, "'", "''")

	var derivedCols string
	if kind == Comments {
		derivedCols = `,
        CASE WHEN created_utc IS NOT NULL THEN epoch_ms(CAST(created_utc AS BIGINT) * 1000) ELSE NULL END AS created_at,
        CASE WHEN body IS NOT NULL THEN LENGTH(CAST(body AS VARCHAR)) ELSE 0 END AS body_length`
	} else {
		derivedCols = `,
        CASE WHEN created_utc IS NOT NULL THEN epoch_ms(CAST(created_utc AS BIGINT) * 1000) ELSE NULL END AS created_at,
        CASE WHEN title IS NOT NULL THEN LENGTH(CAST(title AS VARCHAR)) ELSE 0 END AS title_length`
	}

	return fmt.Sprintf(`CREATE TABLE %s AS
SELECT *%s
FROM read_json_auto('%s',
    format='newline_delimited',
    maximum_object_size=10485760,
    ignore_errors=true,
    union_by_name=true
)`, tableName, derivedCols, escaped)
}

// Info returns statistics about an imported DuckDB file.
type Info struct {
	Rows          int64
	Columns       int
	ColumnNames   []string
	ZstSize       int64 // bytes
	DBSize        int64 // bytes
	PQSize        int64 // bytes
	TopSubreddits []SubredditCount
	TopAuthors    []AuthorCount
	DateRange     [2]string // min, max created_at
}

// SubredditCount is a subreddit with its count.
type SubredditCount struct {
	Name  string
	Count int64
}

// AuthorCount is an author with their count.
type AuthorCount struct {
	Name  string
	Count int64
}

// GetInfo reads stats from an imported DuckDB file.
func GetInfo(file DataFile) (*Info, error) {
	if _, err := os.Stat(file.DBPath); err != nil {
		return nil, fmt.Errorf("database not found: %s", file.DBPath)
	}

	db, err := sql.Open("duckdb", file.DBPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	tableName := "comments"
	if file.Kind == Submissions {
		tableName = "submissions"
	}

	info := &Info{}

	// Row count
	db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&info.Rows)

	// Column info via information_schema
	rows, err := db.Query(fmt.Sprintf(
		"SELECT column_name FROM information_schema.columns WHERE table_name = '%s' ORDER BY ordinal_position",
		tableName,
	))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			if rows.Scan(&name) == nil {
				info.ColumnNames = append(info.ColumnNames, name)
			}
		}
		info.Columns = len(info.ColumnNames)
	}

	// File sizes
	if st, err := os.Stat(file.ZstPath); err == nil {
		info.ZstSize = st.Size()
	}
	if st, err := os.Stat(file.DBPath); err == nil {
		info.DBSize = st.Size()
	}
	if st, err := os.Stat(file.PQPath); err == nil {
		info.PQSize = st.Size()
	}

	// Top subreddits
	subRows, err := db.Query(fmt.Sprintf(
		"SELECT subreddit, COUNT(*) as cnt FROM %s WHERE subreddit IS NOT NULL GROUP BY subreddit ORDER BY cnt DESC LIMIT 10",
		tableName,
	))
	if err == nil {
		defer subRows.Close()
		for subRows.Next() {
			var sc SubredditCount
			if subRows.Scan(&sc.Name, &sc.Count) == nil {
				info.TopSubreddits = append(info.TopSubreddits, sc)
			}
		}
	}

	// Top authors
	authRows, err := db.Query(fmt.Sprintf(
		"SELECT author, COUNT(*) as cnt FROM %s WHERE author IS NOT NULL AND author != '[deleted]' GROUP BY author ORDER BY cnt DESC LIMIT 10",
		tableName,
	))
	if err == nil {
		defer authRows.Close()
		for authRows.Next() {
			var ac AuthorCount
			if authRows.Scan(&ac.Name, &ac.Count) == nil {
				info.TopAuthors = append(info.TopAuthors, ac)
			}
		}
	}

	// Date range
	var minDate, maxDate sql.NullString
	db.QueryRow(fmt.Sprintf(
		"SELECT MIN(created_at)::VARCHAR, MAX(created_at)::VARCHAR FROM %s WHERE created_at IS NOT NULL",
		tableName,
	)).Scan(&minDate, &maxDate)
	if minDate.Valid {
		info.DateRange[0] = minDate.String
	}
	if maxDate.Valid {
		info.DateRange[1] = maxDate.String
	}

	return info, nil
}
