package openlibrarydump

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/mizu/blueprints/book/store/factory"
)

const (
	defaultAuthorsPattern  = "ol_dump_authors_*.txt.gz"
	defaultWorksPattern    = "ol_dump_works_*.txt.gz"
	defaultEditionsPattern = "ol_dump_editions_*.txt.gz"
	csvMaxLineSizeBytes    = 6_000_000
	csvBufferSizeBytes     = 24_000_000
	defaultImportThreads   = 2
	defaultBatchSize       = 200_000
)

type Options struct {
	Dir          string
	AuthorsPath  string
	WorksPath    string
	EditionsPath string
	LimitWorks   int
	ReplaceBooks bool
	SkipEditions bool
}

type Stats struct {
	WorksStaged    int
	AuthorsStaged  int
	EditionsStaged int
	BooksInserted  int
	Duration       time.Duration
	MemoryLimit    string
	Threads        int
}

// progress tracks import phase state for clean output.
type progress struct {
	phase int
	total int
	start time.Time
}

const dotLeaderWidth = 40

func (p *progress) exec(ctx context.Context, tx *sql.Tx, name, query string) (time.Duration, error) {
	p.phase++
	tag := fmt.Sprintf("[%2d/%d]", p.phase, p.total)
	dots := dotLeaderWidth - len(name)
	if dots < 3 {
		dots = 3
	}
	leader := " " + strings.Repeat("·", dots) + " "
	fmt.Fprintf(os.Stdout, "  %s %s%s", tag, name, leader)
	phaseStart := time.Now()

	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(phaseStart).Round(time.Second)
				fmt.Fprintf(os.Stderr, "\r  %s %s%sstill running... (%s)", tag, name, leader, elapsed)
			}
		}
	}()

	if _, err := tx.ExecContext(ctx, query); err != nil {
		fmt.Fprintln(os.Stdout, "FAILED")
		return 0, fmt.Errorf("%s: %s", name, shortErr(err))
	}
	elapsed := time.Since(phaseStart)
	return elapsed, nil
}

// execBatched runs a phase in rowid-range batches with live progress.
// setupSQL creates the target table. batchSQL returns the INSERT query for a [start,end) range.
// finalSQL runs after all batches (e.g., dedup).
func (p *progress) execBatched(ctx context.Context, tx *sql.Tx, name string,
	sourceTable string, batchSize int64,
	setupSQL string, batchSQL func(start, end int64) string,
	finalSQL []string,
) (time.Duration, error) {
	p.phase++
	tag := fmt.Sprintf("[%2d/%d]", p.phase, p.total)
	phaseStart := time.Now()

	// Print phase header.
	fmt.Fprintf(os.Stdout, "  %s %s\n", tag, name)

	// Execute setup SQL.
	if _, err := tx.ExecContext(ctx, setupSQL); err != nil {
		return 0, fmt.Errorf("%s setup: %s", name, shortErr(err))
	}

	// Get rowid range and row count.
	var minRowID, maxRowID, rowCount int64
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT COALESCE(MIN(rowid), 0), COALESCE(MAX(rowid), 0), COUNT(*) FROM %s", sourceTable),
	).Scan(&minRowID, &maxRowID, &rowCount); err != nil {
		return 0, fmt.Errorf("%s rowid range: %w", name, err)
	}
	totalRange := maxRowID - minRowID + 1
	if totalRange <= 0 {
		totalRange = 1
	}

	// Process batches.
	for start := minRowID; rowCount > 0 && start <= maxRowID; start += batchSize {
		end := start + batchSize
		if end > maxRowID+1 {
			end = maxRowID + 1
		}
		if _, err := tx.ExecContext(ctx, batchSQL(start, end)); err != nil {
			fmt.Fprintln(os.Stdout)
			return 0, fmt.Errorf("%s batch [%d, %d): %s", name, start, end, shortErr(err))
		}

		processed := end - minRowID
		pct := float64(processed) / float64(totalRange) * 100
		elapsed := time.Since(phaseStart)
		var eta string
		if pct > 0 && pct < 100 {
			remaining := time.Duration(float64(elapsed) * (100 - pct) / pct)
			eta = "ETA " + FormatDuration(remaining)
		}
		fmt.Fprintf(os.Stdout, "\r         %s / %s   %.1f%%  %s   ",
			FormatNumber(int(processed)), FormatNumber(int(totalRange)), pct, eta)
	}

	// Execute final SQL statements.
	for _, q := range finalSQL {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			fmt.Fprintln(os.Stdout)
			return 0, fmt.Errorf("%s final: %s", name, shortErr(err))
		}
	}

	elapsed := time.Since(phaseStart)

	// Clear progress line, move cursor up, reprint header with dot-leader for finish().
	if rowCount > 0 {
		fmt.Fprintf(os.Stdout, "\r%s", strings.Repeat(" ", 80))
	}
	fmt.Fprintf(os.Stdout, "\r\033[1A\r")
	dots := dotLeaderWidth - len(name)
	if dots < 3 {
		dots = 3
	}
	leader := " " + strings.Repeat("·", dots) + " "
	fmt.Fprintf(os.Stdout, "  %s %s%s", tag, name, leader)

	return elapsed, nil
}

func (p *progress) finish(count int, label string, elapsed time.Duration) {
	if count > 0 {
		fmt.Fprintf(os.Stdout, "%-24s %s\n", FormatNumber(count)+" "+label, FormatDuration(elapsed))
	} else {
		fmt.Fprintf(os.Stdout, "%24s %s\n", "", FormatDuration(elapsed))
	}
}

func ImportToDuckDB(ctx context.Context, dbPath string, opts Options) (*Stats, error) {
	importStart := time.Now()

	// Auto-skip editions when using --limit: scanning the full editions dump
	// (12GB+) for a small number of works is extremely slow and wasteful.
	if opts.LimitWorks > 0 && !opts.SkipEditions {
		opts.SkipEditions = true
	}

	// Ensure schema exists with default backend wiring.
	st, err := factory.Open(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	if err := st.Close(); err != nil {
		return nil, fmt.Errorf("close store: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Configure DuckDB memory and threading.
	importThreads := envInt("BOOK_OL_IMPORT_THREADS", defaultImportThreads)
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA threads=%d", importThreads)); err != nil {
		return nil, fmt.Errorf("set threads pragma: %w", err)
	}
	memoryLimit := envString("BOOK_OL_IMPORT_MEMORY_LIMIT", autoMemoryLimit())
	memoryLimitSQL := strings.ReplaceAll(memoryLimit, "'", "''")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET memory_limit = '%s'", memoryLimitSQL)); err != nil {
		fmt.Fprintf(os.Stderr, "  [warn] could not set memory_limit=%s: %v\n", memoryLimit, err)
	}
	if _, err := tx.ExecContext(ctx, "SET preserve_insertion_order = false"); err != nil {
		fmt.Fprintf(os.Stderr, "  [warn] could not set preserve_insertion_order=false: %v\n", err)
	}
	tempDir := envString("BOOK_OL_IMPORT_TEMP_DIR", filepath.Join(opts.Dir, "duckdb_tmp"))
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return nil, fmt.Errorf("create duckdb temp dir: %w", err)
	}
	tempDirSQL := strings.ReplaceAll(tempDir, "'", "''")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET temp_directory = '%s'", tempDirSQL)); err != nil {
		fmt.Fprintf(os.Stderr, "  [warn] could not set temp_directory: %v\n", err)
	}
	if maxTempSize := envString("BOOK_OL_IMPORT_MAX_TEMP_SIZE", ""); maxTempSize != "" {
		maxTempSizeSQL := strings.ReplaceAll(maxTempSize, "'", "''")
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET max_temp_directory_size = '%s'", maxTempSizeSQL)); err != nil {
			fmt.Fprintf(os.Stderr, "  [warn] could not set max_temp_directory_size=%s: %v\n", maxTempSize, err)
		}
	}

	// Pre-flight disk space check for full imports.
	if opts.LimitWorks == 0 {
		if avail := availableDiskGB(tempDir); avail > 0 && avail < 20 {
			fmt.Fprintf(os.Stderr, "\n  [warn] Only %.1f GB free on %s\n", avail, tempDir)
			fmt.Fprintf(os.Stderr, "  [warn] Full import needs ~20-50 GB. Use --limit=N or free disk space.\n")
		}
	}

	// Compute total phases.
	totalPhases := 8
	if opts.ReplaceBooks {
		totalPhases = 9
	}
	prog := &progress{total: totalPhases, start: importStart}
	batchSize := int64(envInt("BOOK_OL_IMPORT_BATCH_SIZE", defaultBatchSize))

	worksLimit := ""
	if opts.LimitWorks > 0 {
		worksLimit = fmt.Sprintf(" LIMIT %d", opts.LimitWorks)
	}

	// Phase 1: Stage works
	worksSQL := fmt.Sprintf(`
CREATE OR REPLACE TEMP TABLE ol_works_stage AS
SELECT
  t.col2 AS ol_key,
  NULLIF(TRIM(json_extract_string(t.raw_json, '$.title')), '') AS title,
  COALESCE(
    NULLIF(TRIM(json_extract_string(t.raw_json, '$.description.value')), ''),
    NULLIF(TRIM(json_extract_string(t.raw_json, '$.description')), ''),
    ''
  ) AS description,
  COALESCE(CAST(json_extract(t.raw_json, '$.subjects') AS VARCHAR), '[]') AS subjects_json,
  TRY_CAST(json_extract_string(t.raw_json, '$.covers[0]') AS INTEGER) AS cover_id,
  COALESCE(NULLIF(TRIM(json_extract_string(t.raw_json, '$.first_publish_date')), ''), '') AS first_publish_date,
  TRY_CAST(regexp_extract(json_extract_string(t.raw_json, '$.first_publish_date'), '(\d{4})', 1) AS INTEGER) AS publish_year,
  COALESCE(TRY_CAST(json_extract_string(t.raw_json, '$.ratings_average') AS REAL), 0) AS average_rating,
  COALESCE(TRY_CAST(json_extract_string(t.raw_json, '$.ratings_count') AS INTEGER), 0) AS ratings_count,
  CAST(COALESCE(json_extract(t.raw_json, '$.authors'), '[]') AS VARCHAR) AS authors_json
FROM (
  SELECT
    column1 AS col1,
    column2 AS col2,
    column5 AS raw_json
  FROM read_csv('%s',
      delim='\t',
      header=false,
      compression='gzip',
      max_line_size=%d,
      buffer_size=%d,
      columns={'column1':'VARCHAR','column2':'VARCHAR','column3':'VARCHAR','column4':'VARCHAR','column5':'VARCHAR'})
  WHERE column1 = '/type/work'
%s
) t
WHERE t.col2 IS NOT NULL;
`, sqlString(opts.WorksPath), csvMaxLineSizeBytes, csvBufferSizeBytes, worksLimit)
	if d, err := prog.exec(ctx, tx, "Stage works", worksSQL); err != nil {
		return nil, err
	} else if n, err := queryCount(ctx, tx, "SELECT COUNT(*) FROM ol_works_stage"); err == nil {
		prog.finish(n, "works", d)
	}

	// Phase 2: Extract work-author pairs (batched, no DISTINCT — dedup at the end).
	// Intermediate ol_work_author_pairs avoids running regex twice (reused in Phase 4).
	// pos column preserves original author order from JSON for string_agg ORDER BY.
	if d, err := prog.execBatched(ctx, tx, "Extract author pairs",
		"ol_works_stage", batchSize,
		`CREATE TEMP TABLE ol_work_author_pairs (ol_key VARCHAR, author_key VARCHAR, pos INTEGER)`,
		func(start, end int64) string {
			return fmt.Sprintf(`INSERT INTO ol_work_author_pairs
SELECT ol_key,
  unnest(keys) AS author_key,
  unnest(range(1, len(keys) + 1)) AS pos
FROM (
  SELECT ol_key, regexp_extract_all(authors_json, '"key"\s*:\s*"(/authors/[^"]+)"', 1) AS keys
  FROM ol_works_stage
  WHERE rowid >= %d AND rowid < %d
    AND authors_json IS NOT NULL AND authors_json != '[]'
) t`, start, end)
		},
		[]string{
			// Derive distinct author refs for Phase 3 (simple DISTINCT on existing table).
			`CREATE TEMP TABLE ol_author_refs AS SELECT DISTINCT author_key FROM ol_work_author_pairs`,
			// Free authors_json early — no longer needed since pairs are extracted.
			`ALTER TABLE ol_works_stage DROP COLUMN authors_json`,
		},
	); err != nil {
		return nil, err
	} else if n, err := queryCount(ctx, tx, "SELECT COUNT(*) FROM ol_author_refs"); err == nil {
		prog.finish(n, "refs", d)
	}

	// Phase 3: Stage authors
	authorsSQL := fmt.Sprintf(`
CREATE OR REPLACE TEMP TABLE ol_authors_stage AS
SELECT
  r.column2 AS ol_key,
  NULLIF(TRIM(json_extract_string(r.column5, '$.name')), '') AS name,
  COALESCE(
    NULLIF(TRIM(json_extract_string(r.column5, '$.bio.value')), ''),
    NULLIF(TRIM(json_extract_string(r.column5, '$.bio')), ''),
    ''
  ) AS bio,
  COALESCE(NULLIF(TRIM(json_extract_string(r.column5, '$.birth_date')), ''), '') AS birth_date,
  COALESCE(NULLIF(TRIM(json_extract_string(r.column5, '$.death_date')), ''), '') AS death_date,
  COALESCE(TRY_CAST(json_extract_string(r.column5, '$.work_count') AS INTEGER), 0) AS works_count
FROM read_csv('%s',
    delim='\t',
    header=false,
    compression='gzip',
    max_line_size=%d,
    buffer_size=%d,
    columns={'column1':'VARCHAR','column2':'VARCHAR','column3':'VARCHAR','column4':'VARCHAR','column5':'VARCHAR'}) r
JOIN ol_author_refs refs ON refs.author_key = r.column2
WHERE r.column1 = '/type/author'
  AND NULLIF(TRIM(json_extract_string(r.column5, '$.name')), '') IS NOT NULL;
`, sqlString(opts.AuthorsPath), csvMaxLineSizeBytes, csvBufferSizeBytes)
	if d, err := prog.exec(ctx, tx, "Stage authors", authorsSQL); err != nil {
		return nil, err
	} else if n, err := queryCount(ctx, tx, "SELECT COUNT(*) FROM ol_authors_stage"); err == nil {
		prog.finish(n, "authors", d)
	}

	// Phase 4: Build work-author names (batched join on pre-extracted pairs — no regex/unnest).
	if d, err := prog.execBatched(ctx, tx, "Build work-author names",
		"ol_works_stage", batchSize,
		`CREATE TEMP TABLE ol_work_author_names (ol_key VARCHAR, author_names VARCHAR)`,
		func(start, end int64) string {
			return fmt.Sprintf(`INSERT INTO ol_work_author_names
SELECT w.ol_key, COALESCE(string_agg(a.name, ', ' ORDER BY p.pos), '') AS author_names
FROM (SELECT ol_key FROM ol_works_stage WHERE rowid >= %d AND rowid < %d) w
LEFT JOIN ol_work_author_pairs p ON p.ol_key = w.ol_key
LEFT JOIN ol_authors_stage a ON a.ol_key = p.author_key
GROUP BY w.ol_key`, start, end)
		},
		[]string{
			`DROP TABLE IF EXISTS ol_work_author_pairs`,
			`DROP TABLE IF EXISTS ol_author_refs`,
		},
	); err != nil {
		return nil, err
	} else {
		prog.finish(0, "", d)
	}

	// Phase 5: Stage or skip editions
	if opts.SkipEditions {
		if d, err := prog.exec(ctx, tx, "Skip editions (empty table)", `
CREATE OR REPLACE TEMP TABLE ol_editions_stage (
  ol_key VARCHAR,
  isbn13 VARCHAR,
  isbn10 VARCHAR,
  publisher VARCHAR,
  publish_date VARCHAR,
  publish_year INTEGER,
  page_count INTEGER,
  language VARCHAR
);
`); err != nil {
			return nil, err
		} else {
			prog.finish(0, "", d)
		}
	} else {
		editionsSQL := fmt.Sprintf(`
CREATE OR REPLACE TEMP TABLE ol_editions_stage AS
SELECT
  ol_key,
  first(isbn13) FILTER (WHERE isbn13 IS NOT NULL) AS isbn13,
  first(isbn10) FILTER (WHERE isbn10 IS NOT NULL) AS isbn10,
  first(publisher) FILTER (WHERE publisher IS NOT NULL) AS publisher,
  first(publish_date) FILTER (WHERE publish_date IS NOT NULL) AS publish_date,
  first(publish_year) FILTER (WHERE publish_year IS NOT NULL) AS publish_year,
  first(page_count) FILTER (WHERE page_count > 0) AS page_count,
  first(language) FILTER (WHERE language IS NOT NULL) AS language
FROM (
  SELECT
    json_extract_string(w.value, '$.key') AS ol_key,
    NULLIF(regexp_replace(json_extract_string(r.column5, '$.isbn_13[0]'), '[^0-9Xx]', '', 'g'), '') AS isbn13,
    NULLIF(regexp_replace(json_extract_string(r.column5, '$.isbn_10[0]'), '[^0-9Xx]', '', 'g'), '') AS isbn10,
    NULLIF(TRIM(json_extract_string(r.column5, '$.publishers[0]')), '') AS publisher,
    NULLIF(TRIM(json_extract_string(r.column5, '$.publish_date')), '') AS publish_date,
    TRY_CAST(regexp_extract(json_extract_string(r.column5, '$.publish_date'), '(\d{4})', 1) AS INTEGER) AS publish_year,
    COALESCE(TRY_CAST(json_extract_string(r.column5, '$.number_of_pages') AS INTEGER), 0) AS page_count,
    CASE
      WHEN split_part(json_extract_string(r.column5, '$.languages[0].key'), '/', 3) = 'eng' THEN 'en'
      WHEN split_part(json_extract_string(r.column5, '$.languages[0].key'), '/', 3) = 'fre' THEN 'fr'
      WHEN split_part(json_extract_string(r.column5, '$.languages[0].key'), '/', 3) = 'ger' THEN 'de'
      WHEN split_part(json_extract_string(r.column5, '$.languages[0].key'), '/', 3) = 'spa' THEN 'es'
      ELSE COALESCE(NULLIF(split_part(json_extract_string(r.column5, '$.languages[0].key'), '/', 3), ''), 'en')
    END AS language
  FROM read_csv('%s',
      delim='\t',
      header=false,
      compression='gzip',
      max_line_size=%d,
      buffer_size=%d,
      columns={'column1':'VARCHAR','column2':'VARCHAR','column3':'VARCHAR','column4':'VARCHAR','column5':'VARCHAR'}) r,
    json_each(COALESCE(json_extract(r.column5, '$.works'), '[]')) w
  WHERE r.column1 = '/type/edition'
    AND json_extract_string(w.value, '$.key') IN (SELECT ol_key FROM ol_works_stage)
)
GROUP BY ol_key;
`, sqlString(opts.EditionsPath), csvMaxLineSizeBytes, csvBufferSizeBytes)
		if d, err := prog.exec(ctx, tx, "Stage editions", editionsSQL); err != nil {
			return nil, err
		} else if n, err := queryCount(ctx, tx, "SELECT COUNT(*) FROM ol_editions_stage"); err == nil {
			prog.finish(n, "editions", d)
		}
	}

	// Phase 6 (conditional): Delete existing books
	if opts.ReplaceBooks {
		if d, err := prog.exec(ctx, tx, "Delete existing books", "DELETE FROM books WHERE ol_key IN (SELECT ol_key FROM ol_works_stage)"); err != nil {
			return nil, err
		} else {
			prog.finish(0, "", d)
		}
	}

	// Phase 6/7: Delete existing authors
	if d, err := prog.exec(ctx, tx, "Delete existing authors", "DELETE FROM authors WHERE ol_key IN (SELECT ol_key FROM ol_authors_stage)"); err != nil {
		return nil, err
	} else {
		prog.finish(0, "", d)
	}

	// Phase 7/8: Insert authors
	if d, err := prog.exec(ctx, tx, "Insert authors", `
INSERT INTO authors (ol_key, name, bio, birth_date, death_date, works_count)
SELECT ol_key, name, bio, birth_date, death_date, works_count
FROM ol_authors_stage
`); err != nil {
		return nil, err
	} else if n, err := queryCount(ctx, tx, "SELECT COUNT(*) FROM authors WHERE ol_key IN (SELECT ol_key FROM ol_authors_stage)"); err == nil {
		prog.finish(n, "inserted", d)
	}

	// Phase 8/9: Insert books
	if d, err := prog.exec(ctx, tx, "Insert books", `
INSERT INTO books (
  ol_key, title, description, author_names, cover_url, cover_id,
  isbn10, isbn13, publisher, publish_date, publish_year, page_count, language, subjects_json,
  average_rating, ratings_count
)
SELECT
  w.ol_key,
  w.title,
  w.description,
  COALESCE(NULLIF(wa.author_names, ''), 'Unknown'),
  CASE WHEN COALESCE(w.cover_id, 0) > 0 THEN
    'https://covers.openlibrary.org/b/id/' || CAST(w.cover_id AS VARCHAR) || '-M.jpg'
  ELSE '' END AS cover_url,
  COALESCE(w.cover_id, 0),
  COALESCE(e.isbn10, ''),
  COALESCE(e.isbn13, ''),
  COALESCE(e.publisher, ''),
  COALESCE(e.publish_date, w.first_publish_date, ''),
  COALESCE(e.publish_year, w.publish_year, 0),
  COALESCE(e.page_count, 0),
  COALESCE(e.language, 'en'),
  COALESCE(w.subjects_json, '[]'),
  COALESCE(w.average_rating, 0),
  COALESCE(w.ratings_count, 0)
FROM ol_works_stage w
LEFT JOIN ol_work_author_names wa ON wa.ol_key = w.ol_key
LEFT JOIN ol_editions_stage e ON e.ol_key = w.ol_key
WHERE w.title IS NOT NULL
  AND TRIM(w.title) <> ''
  AND NOT EXISTS (
    SELECT 1 FROM books b WHERE b.ol_key = w.ol_key
  );
`); err != nil {
		return nil, err
	} else if n, err := queryCount(ctx, tx, "SELECT COUNT(*) FROM books WHERE ol_key IN (SELECT ol_key FROM ol_works_stage)"); err == nil {
		prog.finish(n, "inserted", d)
	}

	// Collect final counts.
	stats := &Stats{MemoryLimit: memoryLimit, Threads: importThreads}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM ol_works_stage").Scan(&stats.WorksStaged); err != nil {
		return nil, fmt.Errorf("count works stage: %w", err)
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM ol_authors_stage").Scan(&stats.AuthorsStaged); err != nil {
		return nil, fmt.Errorf("count authors stage: %w", err)
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM ol_editions_stage").Scan(&stats.EditionsStaged); err != nil {
		return nil, fmt.Errorf("count editions stage: %w", err)
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM books WHERE ol_key IN (SELECT ol_key FROM ol_works_stage)").Scan(&stats.BooksInserted); err != nil {
		return nil, fmt.Errorf("count imported books: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	stats.Duration = time.Since(importStart)

	// Best-effort cleanup of DuckDB temp directory.
	_ = os.RemoveAll(tempDir)
	return stats, nil
}

func queryCount(ctx context.Context, tx *sql.Tx, query string) (int, error) {
	var n int
	if err := tx.QueryRowContext(ctx, query).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func shortErr(err error) string {
	const maxLen = 1200
	msg := err.Error()
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "... [truncated]"
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envString(name, fallback string) string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	return raw
}

// autoMemoryLimit returns a DuckDB memory_limit based on system RAM.
// Detects total physical memory and uses 50% (clamped 2-16 GB).
func autoMemoryLimit() string {
	totalGB := detectSystemMemoryGB()
	halfGB := totalGB / 2
	if halfGB < 2 {
		halfGB = 2
	}
	if halfGB > 16 {
		halfGB = 16
	}
	return fmt.Sprintf("%dGB", halfGB)
}

func detectSystemMemoryGB() int64 {
	// macOS: sysctl hw.memsize
	out, err := execOutput("sysctl", "-n", "hw.memsize")
	if err == nil {
		if bytes, e := strconv.ParseInt(strings.TrimSpace(out), 10, 64); e == nil && bytes > 0 {
			return bytes / (1024 * 1024 * 1024)
		}
	}
	// Linux: /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, e := strconv.ParseInt(fields[1], 10, 64); e == nil {
						return kb / (1024 * 1024)
					}
				}
			}
		}
	}
	return 8 // safe default
}

func execOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

func availableDiskGB(path string) float64 {
	out, err := execOutput("df", "-k", path)
	if err != nil {
		return -1
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return -1
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 4 {
		return -1
	}
	kb, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return -1
	}
	return float64(kb) / (1024 * 1024)
}

func ResolvePaths(opts Options) (Options, error) {
	out := opts
	if out.Dir == "" {
		out.Dir = filepath.Join(os.Getenv("HOME"), "data", "openlibrary")
	}
	if out.AuthorsPath == "" {
		p, err := latestMatch(out.Dir, defaultAuthorsPattern)
		if err != nil {
			return out, fmt.Errorf("resolve authors dump: %w", err)
		}
		out.AuthorsPath = p
	}
	if out.WorksPath == "" {
		p, err := latestMatch(out.Dir, defaultWorksPattern)
		if err != nil {
			return out, fmt.Errorf("resolve works dump: %w", err)
		}
		out.WorksPath = p
	}
	if out.EditionsPath == "" {
		p, err := latestMatch(out.Dir, defaultEditionsPattern)
		if err != nil {
			return out, fmt.Errorf("resolve editions dump: %w", err)
		}
		out.EditionsPath = p
	}
	return out, nil
}

func latestMatch(dir, pattern string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no files match %s", filepath.Join(dir, pattern))
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func sqlString(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}

// FormatNumber formats an integer with comma separators (e.g. 1234567 → "1,234,567").
func FormatNumber(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	offset := len(s) % 3
	if offset > 0 {
		b.WriteString(s[:offset])
	}
	for i := offset; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// fileSizeStr returns a human-readable file size string, or empty if the file can't be stat'd.
func fileSizeStr(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("(%s)", FormatBytes(info.Size()))
}

// FormatDuration formats a duration for display.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

// ExportStats holds row counts from a parquet export.
type ExportStats struct {
	BooksExported   int
	AuthorsExported int
}

// ExportParquet writes imported Open Library records into parquet files.
func ExportParquet(ctx context.Context, dbPath, outDir string) ([]string, *ExportStats, error) {
	if outDir == "" {
		outDir = filepath.Join(filepath.Dir(dbPath), "parquet")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create parquet dir: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	booksPath := filepath.Join(outDir, "openlibrary_books.parquet")
	authorsPath := filepath.Join(outDir, "openlibrary_authors.parquet")

	booksSQL := fmt.Sprintf(`
COPY (
  SELECT
    id, ol_key, title, description, author_names, cover_url, cover_id,
    isbn10, isbn13, publisher, publish_date, publish_year, page_count,
    language, subjects_json, average_rating, ratings_count,
    created_at, updated_at
  FROM books
  WHERE ol_key LIKE '/works/%%'
) TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD);
`, sqlString(booksPath))
	if _, err := db.ExecContext(ctx, booksSQL); err != nil {
		return nil, nil, fmt.Errorf("export books parquet: %w", err)
	}

	authorsSQL := fmt.Sprintf(`
COPY (
  SELECT
    id, ol_key, name, bio, birth_date, death_date, works_count, created_at
  FROM authors
  WHERE ol_key LIKE '/authors/%%'
) TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD);
`, sqlString(authorsPath))
	if _, err := db.ExecContext(ctx, authorsSQL); err != nil {
		return nil, nil, fmt.Errorf("export authors parquet: %w", err)
	}

	stats := &ExportStats{}
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM read_parquet(?)", booksPath).Scan(&stats.BooksExported)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM read_parquet(?)", authorsPath).Scan(&stats.AuthorsExported)

	return []string{booksPath, authorsPath}, stats, nil
}

// VerifyStats holds data quality metrics from parquet verification.
type VerifyStats struct {
	BookRows       int
	WithTitle      int
	WithISBN       int
	WithCover      int
	WithRating     int
	AvgRating      float64
	BookFileSize   int64
	AuthorRows     int
	WithName       int
	WithBio        int
	AuthorFileSize int64
}

// VerifyParquet reads exported parquet files and computes data quality stats.
func VerifyParquet(ctx context.Context, booksPath, authorsPath string) (*VerifyStats, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open in-memory duckdb: %w", err)
	}
	defer db.Close()

	vs := &VerifyStats{}

	// Books stats
	err = db.QueryRowContext(ctx, `
SELECT
  COUNT(*) AS total,
  COUNT(*) FILTER (WHERE title IS NOT NULL AND title != '') AS with_title,
  COUNT(*) FILTER (WHERE isbn13 IS NOT NULL AND isbn13 != '') AS with_isbn,
  COUNT(*) FILTER (WHERE cover_url IS NOT NULL AND cover_url != '') AS with_cover,
  COUNT(*) FILTER (WHERE average_rating > 0) AS with_rating,
  COALESCE(AVG(average_rating) FILTER (WHERE average_rating > 0), 0) AS avg_rating
FROM read_parquet(?)
`, booksPath).Scan(&vs.BookRows, &vs.WithTitle, &vs.WithISBN, &vs.WithCover, &vs.WithRating, &vs.AvgRating)
	if err != nil {
		return nil, fmt.Errorf("verify books parquet: %w", err)
	}
	if info, err := os.Stat(booksPath); err == nil {
		vs.BookFileSize = info.Size()
	}

	// Authors stats
	err = db.QueryRowContext(ctx, `
SELECT
  COUNT(*) AS total,
  COUNT(*) FILTER (WHERE name IS NOT NULL AND name != '') AS with_name,
  COUNT(*) FILTER (WHERE bio IS NOT NULL AND bio != '') AS with_bio
FROM read_parquet(?)
`, authorsPath).Scan(&vs.AuthorRows, &vs.WithName, &vs.WithBio)
	if err != nil {
		return nil, fmt.Errorf("verify authors parquet: %w", err)
	}
	if info, err := os.Stat(authorsPath); err == nil {
		vs.AuthorFileSize = info.Size()
	}

	return vs, nil
}

// DeleteSourceFiles removes source dump files after successful import/export.
func DeleteSourceFiles(paths ...string) error {
	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}
	return nil
}
