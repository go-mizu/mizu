package reddit

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ArcticImport imports downloaded JSONL files into DuckDB and exports to parquet.
func ArcticImport(ctx context.Context, target ArcticTarget, kinds []FileKind, cb ImportCallback) error {
	dir := target.Dir()

	dbPath := target.DBPath()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	// Remove existing DB to start fresh
	os.Remove(dbPath)

	start := time.Now()

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	for _, kind := range kinds {
		jsonlPath := target.JSONLPath(kind)
		if _, err := os.Stat(jsonlPath); err != nil {
			continue // Skip if JSONL doesn't exist
		}

		tableName := string(kind)

		// Phase: import
		if cb != nil {
			cb(ImportProgress{Phase: "import", Detail: jsonlPath, Elapsed: time.Since(start)})
		}

		escapedPath := strings.ReplaceAll(jsonlPath, "'", "''")

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

		importQuery := fmt.Sprintf(`CREATE TABLE %s AS
SELECT *%s
FROM read_json_auto('%s',
    format='newline_delimited',
    maximum_object_size=10485760,
    ignore_errors=true,
    union_by_name=true
)`, tableName, derivedCols, escapedPath)

		if _, err := db.ExecContext(ctx, importQuery); err != nil {
			return fmt.Errorf("import %s: %w", kind, err)
		}

		var rowCount int64
		db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount)

		if cb != nil {
			cb(ImportProgress{Phase: "import", Rows: rowCount, Done: true, Detail: string(kind), Elapsed: time.Since(start)})
		}

		// Phase: parquet export
		pqPath := target.ParquetPath(kind)
		if cb != nil {
			cb(ImportProgress{Phase: "parquet", Rows: rowCount, Detail: pqPath, Elapsed: time.Since(start)})
		}

		escapedPQ := strings.ReplaceAll(pqPath, "'", "''")
		exportQuery := fmt.Sprintf("COPY %s TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD)", tableName, escapedPQ)
		if _, err := db.ExecContext(ctx, exportQuery); err != nil {
			return fmt.Errorf("export %s parquet: %w", kind, err)
		}

		if cb != nil {
			cb(ImportProgress{Phase: "parquet", Rows: rowCount, Done: true, Detail: string(kind), Elapsed: time.Since(start)})
		}

		// Phase: create indexes
		if cb != nil {
			cb(ImportProgress{Phase: "index", Rows: rowCount, Detail: string(kind), Elapsed: time.Since(start)})
		}

		db.ExecContext(ctx, fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_author ON %s(author)", kind, tableName))
		if kind == Comments {
			db.ExecContext(ctx, fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_subreddit ON %s(subreddit)", kind, tableName))
		}

		if cb != nil {
			cb(ImportProgress{Phase: "index", Rows: rowCount, Done: true, Detail: string(kind), Elapsed: time.Since(start)})
		}
	}

	return nil
}

// ArcticInfo returns statistics about an imported Arctic Shift database.
type ArcticInfo struct {
	Target          ArcticTarget
	CommentsRows    int64
	SubmissionsRows int64
	DBSize          int64
	CommentsPQSize  int64
	SubmissionsPQSize int64
	CommentsJSONLSize int64
	SubmissionsJSONLSize int64
	DateRange       [2]string // min, max created_at across both tables
	TopAuthors      []AuthorCount
	TopSubreddits   []SubredditCount
}

// GetArcticInfo reads stats from an imported Arctic Shift DuckDB file.
func GetArcticInfo(target ArcticTarget) (*ArcticInfo, error) {
	dbPath := target.DBPath()
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database not found: %s", dbPath)
	}

	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	info := &ArcticInfo{Target: target}

	// File sizes
	if st, err := os.Stat(dbPath); err == nil {
		info.DBSize = st.Size()
	}
	if st, err := os.Stat(target.ParquetPath(Comments)); err == nil {
		info.CommentsPQSize = st.Size()
	}
	if st, err := os.Stat(target.ParquetPath(Submissions)); err == nil {
		info.SubmissionsPQSize = st.Size()
	}
	if st, err := os.Stat(target.JSONLPath(Comments)); err == nil {
		info.CommentsJSONLSize = st.Size()
	}
	if st, err := os.Stat(target.JSONLPath(Submissions)); err == nil {
		info.SubmissionsJSONLSize = st.Size()
	}

	// Comments stats
	if err := db.QueryRow("SELECT COUNT(*) FROM comments").Scan(&info.CommentsRows); err != nil {
		// Table may not exist
		info.CommentsRows = 0
	}

	// Submissions stats
	if err := db.QueryRow("SELECT COUNT(*) FROM submissions").Scan(&info.SubmissionsRows); err != nil {
		info.SubmissionsRows = 0
	}

	// Date range (across both tables)
	var minDate, maxDate sql.NullString
	dateQueries := []string{}
	if info.CommentsRows > 0 {
		dateQueries = append(dateQueries,
			"SELECT MIN(created_at)::VARCHAR, MAX(created_at)::VARCHAR FROM comments WHERE created_at IS NOT NULL")
	}
	if info.SubmissionsRows > 0 {
		dateQueries = append(dateQueries,
			"SELECT MIN(created_at)::VARCHAR, MAX(created_at)::VARCHAR FROM submissions WHERE created_at IS NOT NULL")
	}

	for _, q := range dateQueries {
		var mn, mx sql.NullString
		if db.QueryRow(q).Scan(&mn, &mx) == nil {
			if mn.Valid && (!minDate.Valid || mn.String < minDate.String) {
				minDate = mn
			}
			if mx.Valid && (!maxDate.Valid || mx.String > maxDate.String) {
				maxDate = mx
			}
		}
	}
	if minDate.Valid {
		info.DateRange[0] = minDate.String
	}
	if maxDate.Valid {
		info.DateRange[1] = maxDate.String
	}

	// Top authors (from whichever table has data)
	topQuery := ""
	if info.CommentsRows > 0 && info.SubmissionsRows > 0 {
		topQuery = `SELECT author, SUM(cnt) as total FROM (
			SELECT author, COUNT(*) as cnt FROM comments WHERE author IS NOT NULL AND author != '[deleted]' GROUP BY author
			UNION ALL
			SELECT author, COUNT(*) as cnt FROM submissions WHERE author IS NOT NULL AND author != '[deleted]' GROUP BY author
		) GROUP BY author ORDER BY total DESC LIMIT 10`
	} else if info.CommentsRows > 0 {
		topQuery = "SELECT author, COUNT(*) as cnt FROM comments WHERE author IS NOT NULL AND author != '[deleted]' GROUP BY author ORDER BY cnt DESC LIMIT 10"
	} else if info.SubmissionsRows > 0 {
		topQuery = "SELECT author, COUNT(*) as cnt FROM submissions WHERE author IS NOT NULL AND author != '[deleted]' GROUP BY author ORDER BY cnt DESC LIMIT 10"
	}

	if topQuery != "" {
		rows, err := db.Query(topQuery)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var ac AuthorCount
				if rows.Scan(&ac.Name, &ac.Count) == nil {
					info.TopAuthors = append(info.TopAuthors, ac)
				}
			}
		}
	}

	// Top subreddits (only relevant for user downloads)
	if target.Kind == "user" {
		subQuery := ""
		if info.CommentsRows > 0 && info.SubmissionsRows > 0 {
			subQuery = `SELECT subreddit, SUM(cnt) as total FROM (
				SELECT subreddit, COUNT(*) as cnt FROM comments WHERE subreddit IS NOT NULL GROUP BY subreddit
				UNION ALL
				SELECT subreddit, COUNT(*) as cnt FROM submissions WHERE subreddit IS NOT NULL GROUP BY subreddit
			) GROUP BY subreddit ORDER BY total DESC LIMIT 10`
		} else if info.CommentsRows > 0 {
			subQuery = "SELECT subreddit, COUNT(*) as cnt FROM comments WHERE subreddit IS NOT NULL GROUP BY subreddit ORDER BY cnt DESC LIMIT 10"
		} else if info.SubmissionsRows > 0 {
			subQuery = "SELECT subreddit, COUNT(*) as cnt FROM submissions WHERE subreddit IS NOT NULL GROUP BY subreddit ORDER BY cnt DESC LIMIT 10"
		}

		if subQuery != "" {
			rows, err := db.Query(subQuery)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var sc SubredditCount
					if rows.Scan(&sc.Name, &sc.Count) == nil {
						info.TopSubreddits = append(info.TopSubreddits, sc)
					}
				}
			}
		}
	}

	return info, nil
}
