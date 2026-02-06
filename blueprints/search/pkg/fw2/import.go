package fw2

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ImportParquetToDuckDB creates a DuckDB file from a parquet file with derived columns.
// The resulting DuckDB has a 'docs' table with all original parquet columns plus:
// text_len, word_count, host, domain, year, month, hour, day_of_week,
// url_len, protocol, has_query, url_depth, tld.
// Returns the number of rows imported and duration.
func ImportParquetToDuckDB(parquetPath, dbPath string) (int64, time.Duration, error) {
	start := time.Now()

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return 0, 0, fmt.Errorf("opening duckdb: %w", err)
	}
	defer db.Close()

	// Drop existing table for clean import
	db.Exec("DROP TABLE IF EXISTS docs")

	// Create table with derived columns from parquet
	// Uses generic TLD extraction that works for any language
	query := fmt.Sprintf(`
		CREATE TABLE docs AS
		SELECT *,
			LENGTH(text) AS text_len,
			LENGTH(text) - LENGTH(REPLACE(text, ' ', '')) + 1 AS word_count,
			REGEXP_EXTRACT(url, '://([^/]+)', 1) AS host,
			REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\.', '') AS domain,
			CAST(STRFTIME(TRY_CAST(date AS TIMESTAMP), '%%Y') AS VARCHAR) AS year,
			STRFTIME(TRY_CAST(date AS TIMESTAMP), '%%Y-%%m') AS month,
			STRFTIME(TRY_CAST(date AS TIMESTAMP), '%%H') AS hour,
			DAYNAME(TRY_CAST(date AS TIMESTAMP)) AS day_of_week,
			LENGTH(url) AS url_len,
			CASE WHEN url LIKE 'https%%' THEN 'HTTPS' ELSE 'HTTP' END AS protocol,
			CASE WHEN url LIKE '%%?%%' THEN 1 ELSE 0 END AS has_query,
			LENGTH(REGEXP_EXTRACT(url, '://[^/]+(.*)', 1))
				- LENGTH(REPLACE(REGEXP_EXTRACT(url, '://[^/]+(.*)', 1), '/', '')) AS url_depth,
			'.' || REGEXP_EXTRACT(REGEXP_EXTRACT(url, '://([^/]+)', 1), '\.([a-z]+)$', 1) AS tld,
			CASE
				WHEN url ILIKE '%%news%%' OR url ILIKE '%%article%%' THEN 'News'
				WHEN url ILIKE '%%forum%%' OR url ILIKE '%%thread%%' THEN 'Forum'
				WHEN url ILIKE '%%blog%%' THEN 'Blog'
				WHEN url ILIKE '%%shop%%' OR url ILIKE '%%product%%' THEN 'E-commerce'
				WHEN url ILIKE '%%wiki%%' THEN 'Wiki/Reference'
				WHEN url ILIKE '%%video%%' THEN 'Video/Media'
				ELSE 'Other'
			END AS content_type
		FROM read_parquet('%s')
	`, escapeSQLString(parquetPath))

	_, err = db.Exec(query)
	if err != nil {
		return 0, 0, fmt.Errorf("importing parquet: %w", err)
	}

	// Get row count
	var count int64
	err = db.QueryRow("SELECT COUNT(*) FROM docs").Scan(&count)
	if err != nil {
		return 0, 0, fmt.Errorf("counting rows: %w", err)
	}

	// Create indexes for common query patterns
	for _, col := range []string{"domain", "tld"} {
		db.Exec(fmt.Sprintf("CREATE INDEX idx_%s ON docs(%s)", col, col))
	}

	return count, time.Since(start), nil
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
