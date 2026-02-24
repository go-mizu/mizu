package hn

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type ImportSource string

const (
	ImportSourceAuto       ImportSource = "auto"
	ImportSourceParquet    ImportSource = "parquet"
	ImportSourceClickHouse ImportSource = "clickhouse"
	ImportSourceHybrid     ImportSource = "hybrid"
	ImportSourceAPI        ImportSource = "api"
)

// ImportOptions controls how local HN data is imported into DuckDB.
type ImportOptions struct {
	Source  ImportSource
	DBPath  string
	Rebuild bool
}

// ImportResult summarizes a DuckDB import.
type ImportResult struct {
	DBPath       string
	SourceUsed   ImportSource
	SourcePath   string
	Rows         int64
	RowsBefore   int64
	RowsDelta    int64
	Mode         string
	ImportFromID int64
	ImportedAt   time.Time
	IndexesMade  int
}

func (c Config) Import(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	cfg := c.WithDefaults()
	source, sourcePath, err := cfg.resolveLocalImportSource(opts.Source)
	if err != nil {
		return nil, err
	}
	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = cfg.DefaultDBPath()
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()
	_, _ = db.ExecContext(ctx, `SET preserve_insertion_order=false`)
	_, _ = db.ExecContext(ctx, `SET threads=4`)

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)`); err != nil {
		return nil, fmt.Errorf("create meta table: %w", err)
	}

	itemsExists, err := tableExists(ctx, db, "items")
	if err != nil {
		return nil, fmt.Errorf("check items table: %w", err)
	}
	var rowsBefore int64
	if itemsExists {
		rowsBefore, _ = countRows(ctx, db, "items")
	}

	mode := "full"
	importFromID := int64(0)
	if itemsExists && !opts.Rebuild && source != ImportSourceParquet && itemsSupportsIncremental(ctx, db) {
		importFromID, _ = cfg.suggestIncrementalImportFromID(ctx, db, source)
		if importFromID > 0 {
			if err := importIncremental(ctx, db, cfg, source, importFromID); err != nil {
				return nil, fmt.Errorf("incremental import from %s (id >= %d): %w", source, importFromID, err)
			}
			mode = "incremental"
		}
	}

	if mode == "full" {
		if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS items`); err != nil {
			return nil, fmt.Errorf("drop items table: %w", err)
		}
		var createSQL string
		switch source {
		case ImportSourceParquet:
			escaped := escapeSQLString(sourcePath)
			createSQL = fmt.Sprintf(`CREATE TABLE items AS
SELECT *,
       CASE WHEN try_cast(time AS BIGINT) IS NOT NULL THEN epoch_ms(try_cast(time AS BIGINT) * 1000) ELSE NULL END AS time_ts
FROM read_parquet('%s')`, escaped)
		case ImportSourceClickHouse:
			createSQL = buildCreateItemsFromNormalizedSelectSQL(buildNormalizedClickHouseSelect(sourcePath))
		case ImportSourceHybrid:
			if err := importHybrid(ctx, db, cfg); err != nil {
				return nil, fmt.Errorf("create items table from %s: %w", source, err)
			}
		case ImportSourceAPI:
			createSQL = buildCreateItemsFromNormalizedSelectSQL(buildNormalizedAPISelect(sourcePath))
		default:
			return nil, fmt.Errorf("unsupported import source %q", source)
		}

		if source != ImportSourceHybrid {
			if _, err := db.ExecContext(ctx, createSQL); err != nil {
				return nil, fmt.Errorf("create items table from %s: %w", source, err)
			}
		}
	}

	var rows int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items`).Scan(&rows); err != nil {
		return nil, fmt.Errorf("count imported rows: %w", err)
	}

	indexesMade := ensureItemIndexes(ctx, db)

	importedAt := time.Now().UTC()
	metaPairs := map[string]string{
		"source_kind":     string(source),
		"source_path":     sourcePath,
		"imported_at":     importedAt.Format(time.RFC3339),
		"row_count":       fmt.Sprintf("%d", rows),
		"import_mode":     mode,
		"import_from_id":  fmt.Sprintf("%d", importFromID),
		"row_count_prev":  fmt.Sprintf("%d", rowsBefore),
		"row_count_delta": fmt.Sprintf("%d", rows-rowsBefore),
	}
	for k, v := range metaPairs {
		if _, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO meta(key, value) VALUES (?, ?)`, k, v); err != nil {
			// Non-fatal; keep import result usable.
		}
	}

	_ = cfg.WriteImportState(&ImportState{
		CompletedAt:  importedAt,
		DBPath:       dbPath,
		SourceUsed:   string(source),
		Mode:         mode,
		RowsBefore:   rowsBefore,
		RowsAfter:    rows,
		RowsDelta:    rows - rowsBefore,
		ImportFromID: importFromID,
	})

	return &ImportResult{
		DBPath:       dbPath,
		SourceUsed:   source,
		SourcePath:   sourcePath,
		Rows:         rows,
		RowsBefore:   rowsBefore,
		RowsDelta:    rows - rowsBefore,
		Mode:         mode,
		ImportFromID: importFromID,
		ImportedAt:   importedAt,
		IndexesMade:  indexesMade,
	}, nil
}

func (c Config) resolveLocalImportSource(requested ImportSource) (ImportSource, string, error) {
	cfg := c.WithDefaults()
	if requested == "" {
		requested = ImportSourceAuto
	}

	hasParquet := fileExistsNonEmpty(cfg.RawParquetPath())
	chParquetPattern := filepath.Join(cfg.ClickHouseParquetDir(), "*.parquet")
	chParquetFiles, _ := sortedGlob(chParquetPattern)
	hasClickHouseParquet := len(chParquetFiles) > 0
	apiPattern := filepath.Join(cfg.APIChunksDir(), "*.jsonl")
	apiChunks, _ := sortedGlob(apiPattern)
	hasAPI := len(apiChunks) > 0

	switch requested {
	case ImportSourceAuto:
		if hasClickHouseParquet && hasAPI {
			return ImportSourceHybrid, chParquetPattern + " + " + apiPattern, nil
		}
		if hasClickHouseParquet {
			return ImportSourceClickHouse, chParquetPattern, nil
		}
		if hasParquet {
			return ImportSourceParquet, cfg.RawParquetPath(), nil
		}
		if hasAPI {
			return ImportSourceAPI, apiPattern, nil
		}
		return "", "", fmt.Errorf("no local HN data found (expected %s or %s)", cfg.RawParquetPath(), apiPattern)
	case ImportSourceParquet:
		if !hasParquet {
			return "", "", fmt.Errorf("parquet file not found: %s", cfg.RawParquetPath())
		}
		return ImportSourceParquet, cfg.RawParquetPath(), nil
	case ImportSourceClickHouse:
		if !hasClickHouseParquet {
			return "", "", fmt.Errorf("no clickhouse parquet chunk files found: %s", chParquetPattern)
		}
		return ImportSourceClickHouse, chParquetPattern, nil
	case ImportSourceHybrid:
		if !hasClickHouseParquet {
			return "", "", fmt.Errorf("hybrid import requires clickhouse parquet chunks: %s", chParquetPattern)
		}
		if !hasAPI {
			return "", "", fmt.Errorf("hybrid import requires API chunk files: %s", apiPattern)
		}
		return ImportSourceHybrid, chParquetPattern + " + " + apiPattern, nil
	case ImportSourceAPI:
		if !hasAPI {
			return "", "", fmt.Errorf("no API chunk files found: %s", apiPattern)
		}
		return ImportSourceAPI, apiPattern, nil
	default:
		return "", "", fmt.Errorf("invalid import source %q", requested)
	}
}

func tableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var n int64
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, name).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func countRows(ctx context.Context, db *sql.DB, table string) (int64, error) {
	var n int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func maxItemID(ctx context.Context, db *sql.DB) int64 {
	var v sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT MAX(id) FROM items`).Scan(&v); err != nil || !v.Valid {
		return 0
	}
	return v.Int64
}

func itemsSupportsIncremental(ctx context.Context, db *sql.DB) bool {
	rows, err := db.QueryContext(ctx, `SELECT column_name FROM information_schema.columns WHERE table_name='items'`)
	if err != nil {
		return false
	}
	defer rows.Close()
	have := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			have[name] = true
		}
	}
	required := []string{
		"id", "deleted", "type", "by", "time", "time_ts", "text", "dead",
		"parent", "poll", "kids", "url", "score", "title", "parts", "descendants",
	}
	for _, c := range required {
		if !have[c] {
			return false
		}
	}
	return true
}

func ensureItemIndexes(ctx context.Context, db *sql.DB) int {
	indexSQL := []string{
		`CREATE INDEX IF NOT EXISTS idx_hn_items_id ON items(id)`,
		`CREATE INDEX IF NOT EXISTS idx_hn_items_type ON items(type)`,
		`CREATE INDEX IF NOT EXISTS idx_hn_items_time ON items(time)`,
		`CREATE INDEX IF NOT EXISTS idx_hn_items_by ON items("by")`,
		`CREATE INDEX IF NOT EXISTS idx_hn_items_parent ON items(parent)`,
	}
	indexesMade := 0
	for _, stmt := range indexSQL {
		if _, err := db.ExecContext(ctx, stmt); err == nil {
			indexesMade++
		}
	}
	return indexesMade
}

func (c Config) suggestIncrementalImportFromID(ctx context.Context, db *sql.DB, source ImportSource) (int64, string) {
	cfg := c.WithDefaults()
	dbMaxID := maxItemID(ctx, db)

	fromState := int64(0)
	if st, err := cfg.ReadDownloadState(); err == nil {
		fromState = st.IncrementalFromIDFor(source)
	}

	fromClickHouseTail := int64(0)
	if source == ImportSourceClickHouse || source == ImportSourceHybrid {
		if chFiles, err := listLocalCHChunks(cfg.ClickHouseParquetDir()); err == nil {
			if starts := newestChunkStarts(chFiles, 2); len(starts) > 0 {
				fromClickHouseTail = starts[0]
			}
		}
	}

	fromAPITail := int64(0)
	if source == ImportSourceAPI || source == ImportSourceHybrid {
		if apiFiles, err := listLocalAPIChunks(cfg.APIChunksDir()); err == nil {
			if starts := newestChunkStarts(apiFiles, 2); len(starts) > 0 {
				fromAPITail = starts[0]
			}
		}
	}

	from := int64(0)
	reason := ""
	if fromState > 0 {
		from = fromState
		reason = "download_state"
	} else {
		from = minPositiveInt64(fromClickHouseTail, fromAPITail)
		switch from {
		case fromClickHouseTail:
			if from > 0 {
				reason = "clickhouse_tail"
			}
		case fromAPITail:
			if from > 0 {
				reason = "api_tail"
			}
		}
	}
	if from == 0 && dbMaxID > 0 {
		from = dbMaxID + 1
		reason = "db_max_plus_one"
	}
	return from, reason
}

func importIncremental(ctx context.Context, db *sql.DB, cfg Config, source ImportSource, fromID int64) error {
	if fromID <= 0 {
		return fmt.Errorf("invalid incremental from id %d", fromID)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS hn_delta`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS hn_api_delta`); err != nil {
		return err
	}
	if err := createTempDeltaTable(ctx, db, cfg, source, fromID); err != nil {
		return err
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DROP TABLE IF EXISTS hn_api_delta`)
		_, _ = db.ExecContext(context.Background(), `DROP TABLE IF EXISTS hn_delta`)
	}()
	if _, err := db.ExecContext(ctx, `DELETE FROM items WHERE id IN (SELECT id FROM hn_delta WHERE id IS NOT NULL)`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO items BY NAME SELECT * FROM hn_delta`); err != nil {
		return err
	}
	return nil
}

func createTempDeltaTable(ctx context.Context, db *sql.DB, cfg Config, source ImportSource, fromID int64) error {
	chPattern := filepath.Join(cfg.ClickHouseParquetDir(), "*.parquet")
	apiPattern := filepath.Join(cfg.APIChunksDir(), "*.jsonl")
	switch source {
	case ImportSourceClickHouse:
		_, err := db.ExecContext(ctx, `CREATE TEMP TABLE hn_delta AS `+buildSelectItemsFromNormalizedSelectSQL(buildFilteredNormalizedSelect(buildNormalizedClickHouseSelect(chPattern), fromID)))
		return err
	case ImportSourceAPI:
		_, err := db.ExecContext(ctx, `CREATE TEMP TABLE hn_delta AS `+buildSelectItemsFromNormalizedSelectSQL(buildFilteredNormalizedSelect(buildNormalizedAPISelect(apiPattern), fromID)))
		return err
	case ImportSourceHybrid:
		if _, err := db.ExecContext(ctx, `CREATE TEMP TABLE hn_delta AS `+buildSelectItemsFromNormalizedSelectSQL(buildFilteredNormalizedSelect(buildNormalizedClickHouseSelect(chPattern), fromID))); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `CREATE TEMP TABLE hn_api_delta AS `+buildSelectItemsFromNormalizedSelectSQL(buildFilteredNormalizedSelect(buildNormalizedAPISelect(apiPattern), fromID))); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM hn_delta WHERE id IN (SELECT id FROM hn_api_delta WHERE id IS NOT NULL)`); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO hn_delta BY NAME SELECT * FROM hn_api_delta`); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("incremental import not supported for source %q", source)
	}
}

func importHybrid(ctx context.Context, db *sql.DB, cfg Config) error {
	chPattern := filepath.Join(cfg.ClickHouseParquetDir(), "*.parquet")
	apiPattern := filepath.Join(cfg.APIChunksDir(), "*.jsonl")
	if _, err := db.ExecContext(ctx, buildCreateItemsFromNormalizedSelectSQL(buildNormalizedClickHouseSelect(chPattern))); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE TEMP TABLE hn_api_delta AS `+buildSelectItemsFromNormalizedSelectSQL(buildNormalizedAPISelect(apiPattern))); err != nil {
		return err
	}
	// Overlay API rows onto the base snapshot. This is cheap because delta is typically small.
	if _, err := db.ExecContext(ctx, `DELETE FROM items WHERE id IN (SELECT id FROM hn_api_delta WHERE id IS NOT NULL)`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO items BY NAME SELECT * FROM hn_api_delta`); err != nil {
		return err
	}
	_, _ = db.ExecContext(ctx, `DROP TABLE hn_api_delta`)
	return nil
}

func buildCreateItemsFromNormalizedSelectSQL(inner string) string {
	return `CREATE TABLE items AS ` + buildSelectItemsFromNormalizedSelectSQL(inner)
}

func buildSelectItemsFromNormalizedSelectSQL(inner string) string {
	return fmt.Sprintf(`SELECT * EXCLUDE (source_priority, __hn_rn)
FROM (
  SELECT *,
         row_number() OVER (
           PARTITION BY id
           ORDER BY source_priority ASC, COALESCE(time, -1) DESC
         ) AS __hn_rn
  FROM (%s)
) AS __hn_dedup
WHERE __hn_rn = 1`, inner)
}

func buildFilteredNormalizedSelect(inner string, fromID int64) string {
	if fromID <= 0 {
		return inner
	}
	return fmt.Sprintf(`SELECT * FROM (%s) AS __hn_filtered WHERE id >= %d`, inner, fromID)
}

func buildNormalizedClickHouseSelect(parquetPattern string) string {
	escaped := escapeSQLString(parquetPattern)
	return fmt.Sprintf(`WITH __hn_ch_tmp AS (
  SELECT * FROM read_parquet('%s')
)
SELECT
  try_cast(src.id AS BIGINT) AS id,
  COALESCE(try_cast(src.deleted AS BIGINT), 0) AS deleted,
  CASE COALESCE(try_cast(src.type AS BIGINT), -1)
    WHEN 1 THEN 'story'
    WHEN 2 THEN 'comment'
    WHEN 3 THEN 'poll'
    WHEN 4 THEN 'pollopt'
    WHEN 5 THEN 'job'
    ELSE CAST(src.type AS VARCHAR)
  END AS type,
  CAST(src."by" AS VARCHAR) AS "by",
  CASE
    WHEN try_cast(src.time AS TIMESTAMP) IS NOT NULL THEN CAST(epoch(try_cast(src.time AS TIMESTAMP)) AS BIGINT)
    ELSE try_cast(src.time AS BIGINT)
  END AS time,
  COALESCE(
    try_cast(src.time AS TIMESTAMP),
    CASE WHEN try_cast(src.time AS BIGINT) IS NOT NULL THEN epoch_ms(try_cast(src.time AS BIGINT) * 1000) ELSE NULL END
  ) AS time_ts,
  CAST(src.text AS VARCHAR) AS text,
  COALESCE(try_cast(src.dead AS BIGINT), 0) AS dead,
  try_cast(src.parent AS BIGINT) AS parent,
  try_cast(src.poll AS BIGINT) AS poll,
  try_cast(src.kids AS BIGINT[]) AS kids,
  CAST(src.url AS VARCHAR) AS url,
  try_cast(src.score AS BIGINT) AS score,
  CAST(src.title AS VARCHAR) AS title,
  try_cast(src.parts AS BIGINT[]) AS parts,
  try_cast(src.descendants AS BIGINT) AS descendants,
  1 AS source_priority
FROM __hn_ch_tmp AS src`, escaped)
}

func buildNormalizedAPISelect(apiJSONLPattern string) string {
	escaped := escapeSQLString(apiJSONLPattern)
	return fmt.Sprintf(`WITH __hn_api_tmp AS (
  SELECT * FROM read_json_auto(
    '%s',
    format='newline_delimited',
    union_by_name=true,
    ignore_errors=true,
    columns={
      id:'BIGINT',
      deleted:'BOOLEAN',
      type:'VARCHAR',
      "by":'VARCHAR',
      time:'BIGINT',
      text:'VARCHAR',
      dead:'BOOLEAN',
      parent:'BIGINT',
      poll:'BIGINT',
      kids:'BIGINT[]',
      url:'VARCHAR',
      score:'BIGINT',
      title:'VARCHAR',
      parts:'BIGINT[]',
      descendants:'BIGINT'
    }
  )
)
SELECT
  try_cast(src.id AS BIGINT) AS id,
  COALESCE(CASE WHEN src.deleted THEN 1 ELSE 0 END, 0) AS deleted,
  CAST(src.type AS VARCHAR) AS type,
  CAST(src."by" AS VARCHAR) AS "by",
  try_cast(src.time AS BIGINT) AS time,
  CASE WHEN try_cast(src.time AS BIGINT) IS NOT NULL THEN epoch_ms(try_cast(src.time AS BIGINT) * 1000) ELSE NULL END AS time_ts,
  CAST(src.text AS VARCHAR) AS text,
  COALESCE(CASE WHEN src.dead THEN 1 ELSE 0 END, 0) AS dead,
  try_cast(src.parent AS BIGINT) AS parent,
  try_cast(src.poll AS BIGINT) AS poll,
  try_cast(src.kids AS BIGINT[]) AS kids,
  CAST(src.url AS VARCHAR) AS url,
  try_cast(src.score AS BIGINT) AS score,
  CAST(src.title AS VARCHAR) AS title,
  try_cast(src.parts AS BIGINT[]) AS parts,
  try_cast(src.descendants AS BIGINT) AS descendants,
  0 AS source_priority
FROM __hn_api_tmp AS src`, escaped)
}
