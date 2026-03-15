package arctic

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/klauspost/compress/zstd"
)

type ShardResult struct {
	Index     int
	LocalPath string
	Rows      int64
	SizeBytes int64
}

type ProcessResult struct {
	Shards    []ShardResult
	TotalRows int64
	TotalSize int64
	Duration  time.Duration
}

type ShardCallback func(ShardResult)

// ProcessZst streams the .zst file at zstPath, reads cfg.ChunkLines lines per chunk,
// writes each chunk to a temp file, imports to DuckDB :memory:, exports parquet shard,
// deletes the chunk file, and repeats. typ must be "comments" or "submissions".
// year is "YYYY", mm is "MM" (zero-padded).
func ProcessZst(ctx context.Context, cfg Config, zstPath, typ, year, mm string,
	cb ShardCallback) (ProcessResult, error) {

	start := time.Now()

	f, err := os.Open(zstPath)
	if err != nil {
		return ProcessResult{}, fmt.Errorf("open zst: %w", err)
	}
	defer f.Close()

	dec, err := zstd.NewReader(f, zstd.WithDecoderMaxWindow(1<<31))
	if err != nil {
		return ProcessResult{}, fmt.Errorf("zstd reader: %w", err)
	}
	defer dec.Close()

	shardDir := cfg.ShardLocalDir(typ, year, mm)
	if err := os.MkdirAll(shardDir, 0o755); err != nil {
		return ProcessResult{}, fmt.Errorf("mkdir shards: %w", err)
	}

	var result ProcessResult
	scanner := bufio.NewScanner(dec)
	scanner.Buffer(make([]byte, 16*1024*1024), 16*1024*1024)

	chunkIdx := 0
	lines := make([]string, 0, cfg.ChunkLines)

	flush := func() error {
		if len(lines) == 0 {
			return nil
		}
		sr, err := writeChunkToShard(ctx, cfg, lines, typ, year, mm, chunkIdx)
		if err != nil {
			return fmt.Errorf("shard %d: %w", chunkIdx, err)
		}
		result.Shards = append(result.Shards, sr)
		result.TotalRows += sr.Rows
		result.TotalSize += sr.SizeBytes
		if cb != nil {
			cb(sr)
		}
		lines = lines[:0]
		chunkIdx++
		return nil
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ProcessResult{}, ctx.Err()
		default:
		}
		lines = append(lines, scanner.Text())
		if len(lines) >= cfg.ChunkLines {
			if err := flush(); err != nil {
				return ProcessResult{}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ProcessResult{}, fmt.Errorf("scan jsonl: %w", err)
	}
	if err := flush(); err != nil {
		return ProcessResult{}, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

func writeChunkToShard(ctx context.Context, cfg Config, lines []string,
	typ, year, mm string, idx int) (ShardResult, error) {

	chunkPath := cfg.ChunkPath(idx)
	if err := os.MkdirAll(filepath.Dir(chunkPath), 0o755); err != nil {
		return ShardResult{}, err
	}

	cf, err := os.Create(chunkPath)
	if err != nil {
		return ShardResult{}, fmt.Errorf("create chunk: %w", err)
	}
	w := bufio.NewWriterSize(cf, 8*1024*1024)
	for _, l := range lines {
		w.WriteString(l)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		cf.Close()
		os.Remove(chunkPath)
		return ShardResult{}, fmt.Errorf("flush chunk: %w", err)
	}
	cf.Close()
	defer os.Remove(chunkPath)

	shardPath := cfg.ShardLocalPath(typ, year, mm, idx)

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return ShardResult{}, fmt.Errorf("duckdb open: %w", err)
	}
	defer db.Close()

	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }

	selectCols := commentsSelect
	if typ == "submissions" {
		selectCols = submissionsSelect
	}

	importSQL := fmt.Sprintf(`CREATE TABLE data AS
SELECT %s
FROM read_json_auto('%s',
    format='newline_delimited',
    maximum_object_size=10485760,
    ignore_errors=true,
    union_by_name=true
)`, selectCols, esc(chunkPath))

	if _, err := db.ExecContext(ctx, importSQL); err != nil {
		return ShardResult{}, fmt.Errorf("duckdb import: %w", err)
	}

	var rowCount int64
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM data").Scan(&rowCount)

	if err := os.MkdirAll(filepath.Dir(shardPath), 0o755); err != nil {
		return ShardResult{}, err
	}
	exportSQL := fmt.Sprintf(
		"COPY data TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD, ROW_GROUP_SIZE 131072)",
		esc(shardPath))
	if _, err := db.ExecContext(ctx, exportSQL); err != nil {
		os.Remove(shardPath)
		return ShardResult{}, fmt.Errorf("duckdb export: %w", err)
	}

	fi, err := os.Stat(shardPath)
	if err != nil {
		return ShardResult{}, fmt.Errorf("stat shard: %w", err)
	}

	return ShardResult{
		Index:     idx,
		LocalPath: shardPath,
		Rows:      rowCount,
		SizeBytes: fi.Size(),
	}, nil
}

const commentsSelect = `
    TRY_CAST(id AS VARCHAR)                                         AS id,
    TRY_CAST(author AS VARCHAR)                                     AS author,
    TRY_CAST(subreddit AS VARCHAR)                                  AS subreddit,
    TRY_CAST(body AS VARCHAR)                                       AS body,
    TRY_CAST(score AS BIGINT)                                       AS score,
    TRY_CAST(created_utc AS BIGINT)                                 AS created_utc,
    CASE WHEN created_utc IS NOT NULL
         THEN epoch_ms(CAST(created_utc AS BIGINT) * 1000)
         ELSE NULL END                                              AS created_at,
    CASE WHEN body IS NOT NULL
         THEN LENGTH(CAST(body AS VARCHAR))
         ELSE 0 END                                                 AS body_length,
    TRY_CAST(link_id AS VARCHAR)                                    AS link_id,
    TRY_CAST(parent_id AS VARCHAR)                                  AS parent_id,
    TRY_CAST(distinguished AS VARCHAR)                              AS distinguished,
    TRY_CAST(author_flair_text AS VARCHAR)                          AS author_flair_text`

const submissionsSelect = `
    TRY_CAST(id AS VARCHAR)                                         AS id,
    TRY_CAST(author AS VARCHAR)                                     AS author,
    TRY_CAST(subreddit AS VARCHAR)                                  AS subreddit,
    TRY_CAST(title AS VARCHAR)                                      AS title,
    TRY_CAST(selftext AS VARCHAR)                                   AS selftext,
    TRY_CAST(score AS BIGINT)                                       AS score,
    TRY_CAST(created_utc AS BIGINT)                                 AS created_utc,
    CASE WHEN created_utc IS NOT NULL
         THEN epoch_ms(CAST(created_utc AS BIGINT) * 1000)
         ELSE NULL END                                              AS created_at,
    CASE WHEN title IS NOT NULL
         THEN LENGTH(CAST(title AS VARCHAR))
         ELSE 0 END                                                 AS title_length,
    TRY_CAST(num_comments AS BIGINT)                                AS num_comments,
    TRY_CAST(url AS VARCHAR)                                        AS url,
    TRY_CAST(over_18 AS BOOLEAN)                                    AS over_18,
    TRY_CAST(link_flair_text AS VARCHAR)                            AS link_flair_text,
    TRY_CAST(author_flair_text AS VARCHAR)                          AS author_flair_text`
