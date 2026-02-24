package hn

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type CompactOptions struct {
	FromID           int64
	ToID             int64
	ChunkIDSpan      int64
	CompressionLevel int
	PruneAPI         bool
}

type CompactChunkResult struct {
	ChunkStart int64
	ChunkEnd   int64
	Path       string
	Rows       int64
}

type CompactResult struct {
	Dir             string
	ChunkIDSpan     int64
	FromID          int64
	ToID            int64
	APIRows         int64
	ChunksTouched   int
	ChunksWritten   int
	ChunksSkipped   int
	FilesPruned     int
	APIChunksPruned int
	Elapsed         time.Duration
	Chunks          []CompactChunkResult
}

func (c Config) CompactDeltaToClickHouseParquet(ctx context.Context, opts CompactOptions) (*CompactResult, error) {
	cfg := c.WithDefaults()
	if err := cfg.EnsureRawDirs(); err != nil {
		return nil, fmt.Errorf("prepare directories: %w", err)
	}

	apiChunks, err := listLocalAPIChunks(cfg.APIChunksDir())
	if err != nil {
		return nil, fmt.Errorf("list api chunks: %w", err)
	}
	if len(apiChunks) == 0 {
		return &CompactResult{Dir: cfg.ClickHouseParquetDir()}, nil
	}

	span := opts.ChunkIDSpan
	if span <= 0 {
		if s, ok := cfg.DetectLocalClickHouseChunkSpan(); ok {
			span = s
		}
	}
	if span <= 0 {
		span = 500_000
	}
	compressionLevel := opts.CompressionLevel
	if compressionLevel <= 0 {
		compressionLevel = 22
	}

	fromID, toID := opts.FromID, opts.ToID
	if fromID <= 0 || toID <= 0 {
		if st, err := cfg.ReadDownloadState(); err == nil && st != nil && st.API != nil {
			if fromID <= 0 {
				fromID = st.API.StartID
			}
			if toID <= 0 {
				toID = st.API.EndID
			}
		}
	}
	if fromID <= 0 {
		fromID = apiChunks[0].StartID
	}
	if toID <= 0 {
		toID = apiChunks[len(apiChunks)-1].EndID
	}
	if toID < fromID {
		// Most recent delta run may be a no-op and store start>end. Fall back to local API chunk files.
		fromID = apiChunks[0].StartID
		toID = apiChunks[len(apiChunks)-1].EndID
	}
	if toID < fromID {
		return nil, fmt.Errorf("invalid compact range: from=%d to=%d", fromID, toID)
	}

	started := time.Now()
	res := &CompactResult{
		Dir:         cfg.ClickHouseParquetDir(),
		ChunkIDSpan: span,
		FromID:      fromID,
		ToID:        toID,
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open duckdb (in-memory): %w", err)
	}
	defer db.Close()
	_, _ = db.ExecContext(ctx, `SET preserve_insertion_order=false`)
	_, _ = db.ExecContext(ctx, `SET threads=4`)

	apiPattern := filepath.Join(cfg.APIChunksDir(), "*.jsonl")
	apiDeltaRawSQL := buildAPIRawClickHouseLikeSelect(apiPattern)
	apiDeltaRawSQL = fmt.Sprintf(`SELECT * FROM (%s) AS __api_raw WHERE id BETWEEN %d AND %d`, apiDeltaRawSQL, fromID, toID)
	if _, err := db.ExecContext(ctx, `CREATE TEMP TABLE hn_api_delta_raw AS `+apiDeltaRawSQL); err != nil {
		return nil, fmt.Errorf("create api delta raw temp table: %w", err)
	}
	defer func() { _, _ = db.ExecContext(context.Background(), `DROP TABLE IF EXISTS hn_api_delta_raw`) }()

	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hn_api_delta_raw`).Scan(&res.APIRows); err != nil {
		return nil, fmt.Errorf("count api delta rows: %w", err)
	}
	if res.APIRows == 0 {
		res.Elapsed = time.Since(started)
		return res, nil
	}

	type touchedChunk struct {
		ChunkStart int64
		APIMaxID   int64
		APIRows    int64
	}
	var touched []touchedChunk
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`SELECT (((id - 1) // %d) * %d) + 1 AS chunk_start, MAX(id) AS api_max_id, COUNT(*) AS n
FROM hn_api_delta_raw
GROUP BY 1
ORDER BY 1`, span, span))
	if err != nil {
		return nil, fmt.Errorf("list touched chunks: %w", err)
	}
	for rows.Next() {
		var tc touchedChunk
		if err := rows.Scan(&tc.ChunkStart, &tc.APIMaxID, &tc.APIRows); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan touched chunk: %w", err)
		}
		touched = append(touched, tc)
	}
	rows.Close()
	res.ChunksTouched = len(touched)
	if len(touched) == 0 {
		res.Elapsed = time.Since(started)
		return res, nil
	}

	localCH, _ := listLocalCHChunks(cfg.ClickHouseParquetDir())
	byStart := map[int64][]localChunkFile{}
	for _, cf := range localCH {
		byStart[cf.StartID] = append(byStart[cf.StartID], cf)
	}
	for _, files := range byStart {
		sort.Slice(files, func(i, j int) bool { return files[i].EndID < files[j].EndID })
	}

	for _, tc := range touched {
		nominalEnd := tc.ChunkStart + span - 1
		targetEnd := nominalEnd
		files := byStart[tc.ChunkStart]
		var existingEnd int64
		for _, cf := range files {
			if cf.EndID > existingEnd {
				existingEnd = cf.EndID
			}
		}
		if existingEnd > 0 && existingEnd < targetEnd {
			targetEnd = existingEnd
		}
		if tc.APIMaxID > targetEnd && tc.APIMaxID < nominalEnd {
			targetEnd = tc.APIMaxID
		}
		if existingEnd == 0 && tc.APIMaxID < targetEnd {
			targetEnd = tc.APIMaxID
		}

		var apiRowsInChunk int64
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hn_api_delta_raw WHERE id BETWEEN ? AND ?`, tc.ChunkStart, targetEnd).Scan(&apiRowsInChunk); err != nil {
			return nil, fmt.Errorf("count api rows in chunk %d-%d: %w", tc.ChunkStart, targetEnd, err)
		}
		if apiRowsInChunk == 0 {
			res.ChunksSkipped++
			continue
		}

		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS hn_chunk_base_raw`)
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS hn_chunk_merged_raw`)

		if len(files) > 0 {
			// Read all files that share the same chunk start and filter by range.
			glob := filepath.Join(cfg.ClickHouseParquetDir(), fmt.Sprintf("id_%09d_*.parquet", tc.ChunkStart))
			baseSQL := fmt.Sprintf(`CREATE TEMP TABLE hn_chunk_base_raw AS
SELECT
  try_cast(id AS BIGINT) AS id,
  try_cast(deleted AS BIGINT) AS deleted,
  try_cast(type AS BIGINT) AS type,
  CAST("by" AS VARCHAR) AS "by",
  try_cast(time AS BIGINT) AS time,
  CAST(text AS VARCHAR) AS text,
  try_cast(dead AS BIGINT) AS dead,
  try_cast(parent AS BIGINT) AS parent,
  try_cast(poll AS BIGINT) AS poll,
  try_cast(kids AS BIGINT[]) AS kids,
  CAST(url AS VARCHAR) AS url,
  try_cast(score AS BIGINT) AS score,
  CAST(title AS VARCHAR) AS title,
  try_cast(parts AS BIGINT[]) AS parts,
  try_cast(descendants AS BIGINT) AS descendants
FROM read_parquet('%s')
WHERE try_cast(id AS BIGINT) BETWEEN %d AND %d`, escapeSQLString(glob), tc.ChunkStart, targetEnd)
			if _, err := db.ExecContext(ctx, baseSQL); err != nil {
				return nil, fmt.Errorf("load base raw chunk %d: %w", tc.ChunkStart, err)
			}
		} else {
			if _, err := db.ExecContext(ctx, `CREATE TEMP TABLE hn_chunk_base_raw AS SELECT * FROM hn_api_delta_raw WHERE 1=0`); err != nil {
				return nil, fmt.Errorf("create empty base raw chunk table: %w", err)
			}
		}

		mergeSQL := fmt.Sprintf(`CREATE TEMP TABLE hn_chunk_merged_raw AS
SELECT * EXCLUDE (__rn, source_priority) FROM (
  SELECT *,
         row_number() OVER (PARTITION BY id ORDER BY source_priority ASC) AS __rn
  FROM (
    SELECT
      id, deleted, type, "by", time, text, dead, parent, poll, kids, url, score, title, parts, descendants,
      1 AS source_priority
    FROM hn_chunk_base_raw
    WHERE id BETWEEN %d AND %d
    UNION ALL
    SELECT
      id, deleted, type, "by", time, text, dead, parent, poll, kids, url, score, title, parts, descendants,
      0 AS source_priority
    FROM hn_api_delta_raw
    WHERE id BETWEEN %d AND %d
  ) AS __raw_union
) AS __raw_ranked
WHERE __rn = 1`, tc.ChunkStart, targetEnd, tc.ChunkStart, targetEnd)
		if _, err := db.ExecContext(ctx, mergeSQL); err != nil {
			return nil, fmt.Errorf("merge raw chunk %d-%d: %w", tc.ChunkStart, targetEnd, err)
		}
		var mergedRows int64
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hn_chunk_merged_raw`).Scan(&mergedRows); err != nil {
			return nil, fmt.Errorf("count merged raw chunk rows %d-%d: %w", tc.ChunkStart, targetEnd, err)
		}

		finalPath := filepath.Join(cfg.ClickHouseParquetDir(), fmt.Sprintf("id_%09d_%09d.parquet", tc.ChunkStart, targetEnd))
		tmpPath := finalPath + ".tmp"
		_ = os.Remove(tmpPath)
		copySQL := fmt.Sprintf(`COPY (
  SELECT id, deleted, type, "by", time, text, dead, parent, poll, kids, url, score, title, parts, descendants
  FROM hn_chunk_merged_raw
  ORDER BY id
) TO '%s' (FORMAT PARQUET, COMPRESSION zstd, COMPRESSION_LEVEL %d)`, escapeSQLString(tmpPath), compressionLevel)
		if _, err := db.ExecContext(ctx, copySQL); err != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("copy compacted parquet chunk %d-%d: %w", tc.ChunkStart, targetEnd, err)
		}

		for _, old := range files {
			if err := os.Remove(old.Path); err == nil || os.IsNotExist(err) {
				res.FilesPruned++
			}
		}
		if err := os.Rename(tmpPath, finalPath); err != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("rename compacted parquet chunk %d-%d: %w", tc.ChunkStart, targetEnd, err)
		}

		res.ChunksWritten++
		res.Chunks = append(res.Chunks, CompactChunkResult{
			ChunkStart: tc.ChunkStart,
			ChunkEnd:   targetEnd,
			Path:       finalPath,
			Rows:       mergedRows,
		})
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS hn_chunk_base_raw`)
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS hn_chunk_merged_raw`)
	}

	if opts.PruneAPI {
		for _, cf := range apiChunks {
			if cf.StartID >= fromID && cf.EndID <= toID {
				if err := os.Remove(cf.Path); err == nil || os.IsNotExist(err) {
					res.APIChunksPruned++
				}
			}
		}
	}

	res.Elapsed = time.Since(started)
	return res, nil
}

func buildAPIRawClickHouseLikeSelect(apiJSONLPattern string) string {
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
  CASE lower(trim(CAST(src.type AS VARCHAR)))
    WHEN 'story' THEN 1
    WHEN 'comment' THEN 2
    WHEN 'poll' THEN 3
    WHEN 'pollopt' THEN 4
    WHEN 'job' THEN 5
    ELSE try_cast(src.type AS BIGINT)
  END AS type,
  CAST(src."by" AS VARCHAR) AS "by",
  try_cast(src.time AS BIGINT) AS time,
  CAST(src.text AS VARCHAR) AS text,
  COALESCE(CASE WHEN src.dead THEN 1 ELSE 0 END, 0) AS dead,
  try_cast(src.parent AS BIGINT) AS parent,
  try_cast(src.poll AS BIGINT) AS poll,
  try_cast(src.kids AS BIGINT[]) AS kids,
  CAST(src.url AS VARCHAR) AS url,
  try_cast(src.score AS BIGINT) AS score,
  CAST(src.title AS VARCHAR) AS title,
  try_cast(src.parts AS BIGINT[]) AS parts,
  try_cast(src.descendants AS BIGINT) AS descendants
FROM __hn_api_tmp AS src
WHERE try_cast(src.id AS BIGINT) IS NOT NULL`, escaped)
}
