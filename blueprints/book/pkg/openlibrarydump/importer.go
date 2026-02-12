package openlibrarydump

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/mizu/blueprints/book/store/factory"
)

const (
	defaultAuthorsPattern  = "ol_dump_authors_*.txt.gz"
	defaultWorksPattern    = "ol_dump_works_*.txt.gz"
	defaultEditionsPattern = "ol_dump_editions_*.txt.gz"
)

type Options struct {
	Dir          string
	AuthorsPath  string
	WorksPath    string
	EditionsPath string
	LimitWorks   int
	ReplaceBooks bool
}

type Stats struct {
	WorksStaged    int
	AuthorsStaged  int
	EditionsStaged int
	BooksInserted  int
}

func ImportToDuckDB(ctx context.Context, dbPath string, opts Options) (*Stats, error) {
	resolved, err := ResolvePaths(opts)
	if err != nil {
		return nil, err
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

	if _, err := tx.ExecContext(ctx, "PRAGMA threads=4"); err != nil {
		return nil, fmt.Errorf("set threads pragma: %w", err)
	}

	worksLimit := ""
	if resolved.LimitWorks > 0 {
		worksLimit = fmt.Sprintf(" LIMIT %d", resolved.LimitWorks)
	}

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
  TRY_CAST(regexp_extract(json_extract_string(t.raw_json, '$.first_publish_date'), '(\\d{4})', 1) AS INTEGER) AS publish_year,
  COALESCE(json_extract(t.raw_json, '$.authors'), '[]') AS authors_json
FROM (
  SELECT
    column1 AS col1,
    column2 AS col2,
    column5 AS raw_json
  FROM read_csv('%s',
      delim='\t',
      header=false,
      compression='gzip',
      columns={'column1':'VARCHAR','column2':'VARCHAR','column3':'VARCHAR','column4':'VARCHAR','column5':'VARCHAR'})
  WHERE column1 = '/type/work'
%s
) t
WHERE t.col2 IS NOT NULL;
`, sqlString(resolved.WorksPath), worksLimit)
	if _, err := tx.ExecContext(ctx, worksSQL); err != nil {
		return nil, fmt.Errorf("stage works: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
CREATE OR REPLACE TEMP TABLE ol_author_refs AS
SELECT DISTINCT json_extract_string(a.value, '$.author.key') AS author_key
FROM ol_works_stage w, json_each(w.authors_json) a
WHERE json_extract_string(a.value, '$.author.key') IS NOT NULL;
`); err != nil {
		return nil, fmt.Errorf("stage author refs: %w", err)
	}

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
    columns={'column1':'VARCHAR','column2':'VARCHAR','column3':'VARCHAR','column4':'VARCHAR','column5':'VARCHAR'}) r
JOIN ol_author_refs refs ON refs.author_key = r.column2
WHERE r.column1 = '/type/author'
  AND NULLIF(TRIM(json_extract_string(r.column5, '$.name')), '') IS NOT NULL;
`, sqlString(resolved.AuthorsPath))
	if _, err := tx.ExecContext(ctx, authorsSQL); err != nil {
		return nil, fmt.Errorf("stage authors: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
CREATE OR REPLACE TEMP TABLE ol_work_author_names AS
SELECT
  w.ol_key,
  COALESCE(string_agg(a.name, ', ' ORDER BY TRY_CAST(j.key AS INTEGER)), '') AS author_names
FROM ol_works_stage w
LEFT JOIN json_each(w.authors_json) j ON true
LEFT JOIN ol_authors_stage a ON a.ol_key = json_extract_string(j.value, '$.author.key')
GROUP BY w.ol_key;
`); err != nil {
		return nil, fmt.Errorf("build work author names: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
CREATE OR REPLACE TEMP TABLE ol_work_refs AS
SELECT ol_key FROM ol_works_stage;
`); err != nil {
		return nil, fmt.Errorf("stage work refs: %w", err)
	}

	editionsSQL := fmt.Sprintf(`
CREATE OR REPLACE TEMP TABLE ol_editions_flat AS
WITH src AS (
  SELECT column5 AS raw_json
  FROM read_csv('%s',
      delim='\t',
      header=false,
      compression='gzip',
      columns={'column1':'VARCHAR','column2':'VARCHAR','column3':'VARCHAR','column4':'VARCHAR','column5':'VARCHAR'})
  WHERE column1 = '/type/edition'
)
SELECT
  json_extract_string(w.value, '$.key') AS ol_key,
  NULLIF(regexp_replace(json_extract_string(src.raw_json, '$.isbn_13[0]'), '[^0-9Xx]', '', 'g'), '') AS isbn13,
  NULLIF(regexp_replace(json_extract_string(src.raw_json, '$.isbn_10[0]'), '[^0-9Xx]', '', 'g'), '') AS isbn10,
  NULLIF(TRIM(json_extract_string(src.raw_json, '$.publishers[0]')), '') AS publisher,
  NULLIF(TRIM(json_extract_string(src.raw_json, '$.publish_date')), '') AS publish_date,
  TRY_CAST(regexp_extract(json_extract_string(src.raw_json, '$.publish_date'), '(\\d{4})', 1) AS INTEGER) AS publish_year,
  COALESCE(TRY_CAST(json_extract_string(src.raw_json, '$.number_of_pages') AS INTEGER), 0) AS page_count,
  CASE
    WHEN split_part(json_extract_string(src.raw_json, '$.languages[0].key'), '/', 3) = 'eng' THEN 'en'
    WHEN split_part(json_extract_string(src.raw_json, '$.languages[0].key'), '/', 3) = 'fre' THEN 'fr'
    WHEN split_part(json_extract_string(src.raw_json, '$.languages[0].key'), '/', 3) = 'ger' THEN 'de'
    WHEN split_part(json_extract_string(src.raw_json, '$.languages[0].key'), '/', 3) = 'spa' THEN 'es'
    ELSE COALESCE(NULLIF(split_part(json_extract_string(src.raw_json, '$.languages[0].key'), '/', 3), ''), 'en')
  END AS language
FROM src, json_each(COALESCE(json_extract(src.raw_json, '$.works'), '[]')) w
JOIN ol_work_refs refs ON refs.ol_key = json_extract_string(w.value, '$.key');
`, sqlString(resolved.EditionsPath))
	if _, err := tx.ExecContext(ctx, editionsSQL); err != nil {
		return nil, fmt.Errorf("stage editions: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
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
FROM ol_editions_flat
GROUP BY ol_key;
`); err != nil {
		return nil, fmt.Errorf("aggregate editions: %w", err)
	}

	if resolved.ReplaceBooks {
		if _, err := tx.ExecContext(ctx, "DELETE FROM books WHERE ol_key IN (SELECT ol_key FROM ol_works_stage)"); err != nil {
			return nil, fmt.Errorf("delete existing books by ol_key: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM authors WHERE ol_key IN (SELECT ol_key FROM ol_authors_stage)"); err != nil {
		return nil, fmt.Errorf("delete existing authors by ol_key: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO authors (ol_key, name, bio, birth_date, death_date, works_count)
SELECT ol_key, name, bio, birth_date, death_date, works_count
FROM ol_authors_stage
`); err != nil {
		return nil, fmt.Errorf("insert authors: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO books (
  ol_key, title, description, author_names, cover_url, cover_id,
  isbn10, isbn13, publisher, publish_date, publish_year, page_count, language, subjects_json
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
  COALESCE(w.subjects_json, '[]')
FROM ol_works_stage w
LEFT JOIN ol_work_author_names wa ON wa.ol_key = w.ol_key
LEFT JOIN ol_editions_stage e ON e.ol_key = w.ol_key
WHERE w.title IS NOT NULL
  AND TRIM(w.title) <> ''
  AND NOT EXISTS (
    SELECT 1 FROM books b WHERE b.ol_key = w.ol_key
  );
`); err != nil {
		return nil, fmt.Errorf("insert books: %w", err)
	}

	stats := &Stats{}
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
	return stats, nil
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

// ExportParquet writes imported Open Library records into parquet files.
func ExportParquet(ctx context.Context, dbPath, outDir string) ([]string, error) {
	if outDir == "" {
		outDir = filepath.Join(filepath.Dir(dbPath), "parquet")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create parquet dir: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	booksPath := filepath.Join(outDir, "openlibrary_books.parquet")
	authorsPath := filepath.Join(outDir, "openlibrary_authors.parquet")

	booksSQL := fmt.Sprintf(`
COPY (
  SELECT
    id, ol_key, title, description, author_names, cover_url, cover_id,
    isbn10, isbn13, publisher, publish_date, publish_year, page_count,
    language, subjects_json, created_at, updated_at
  FROM books
  WHERE ol_key LIKE '/works/%%'
) TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD);
`, sqlString(booksPath))
	if _, err := db.ExecContext(ctx, booksSQL); err != nil {
		return nil, fmt.Errorf("export books parquet: %w", err)
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
		return nil, fmt.Errorf("export authors parquet: %w", err)
	}

	return []string{booksPath, authorsPath}, nil
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
