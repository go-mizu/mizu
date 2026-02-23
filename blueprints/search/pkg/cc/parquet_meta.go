package cc

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ParquetRowCountRequest requests a row-count query for one parquet file.
// Key is returned unchanged in the result map so callers can map counts back
// to manifest remote paths while Path may be a local filesystem path or remote URL/S3 path.
type ParquetRowCountRequest struct {
	Key  string
	Path string
}

// QueryParquetRowCounts returns row counts per parquet file using DuckDB read_parquet().
// This can query local files or remote parquet objects (http/s3) and is intended to be
// called in small chunks so callers can show progress and cache partial results.
func QueryParquetRowCounts(ctx context.Context, reqs []ParquetRowCountRequest) (map[string]int64, error) {
	out := make(map[string]int64, len(reqs))
	if len(reqs) == 0 {
		return out, nil
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("opening duckdb: %w", err)
	}
	defer db.Close()

	needsHTTPFS := false
	for _, r := range reqs {
		if strings.HasPrefix(r.Path, "s3://") || strings.HasPrefix(r.Path, "http://") || strings.HasPrefix(r.Path, "https://") {
			needsHTTPFS = true
			break
		}
	}
	if needsHTTPFS {
		if _, err := db.ExecContext(ctx, "LOAD httpfs"); err != nil {
			if _, instErr := db.ExecContext(ctx, "INSTALL httpfs"); instErr == nil {
				_, _ = db.ExecContext(ctx, "LOAD httpfs")
			}
		}
		_, _ = db.ExecContext(ctx, "SET s3_region='us-east-1'")
	}

	var b strings.Builder
	for i, r := range reqs {
		if i > 0 {
			b.WriteString(" UNION ALL ")
		}
		b.WriteString("SELECT ")
		b.WriteString(duckSQLString(r.Key))
		b.WriteString(" AS cache_key, COALESCE(SUM(row_group_num_rows), 0) AS row_count FROM parquet_metadata(")
		b.WriteString(duckSQLString(r.Path))
		b.WriteString(") WHERE column_id = 0")
	}

	rows, err := db.QueryContext(ctx, b.String())
	if err != nil {
		return nil, fmt.Errorf("querying parquet row counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scanning parquet row counts: %w", err)
		}
		out[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating parquet row counts: %w", err)
	}
	return out, nil
}

func duckSQLString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
