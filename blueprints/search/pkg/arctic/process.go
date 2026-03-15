package arctic

import (
	"bufio"
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

// QuickValidateZst performs a fast sanity check on a .zst file:
//   - Verifies the zstd magic bytes at the start.
//   - Checks the last 16 bytes are not all zeros (catches mmap boundary-piece
//     corruption where anacrolix/torrent left the tail of the file unwritten).
//   - Samples 4 KB at 25%, 50%, 75% offsets — catches zero-filled mid-sections
//     left by incomplete mmap pages.
//
// This runs in microseconds and saves wasting time streaming a bad file.
func QuickValidateZst(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	size := fi.Size()

	// zstd regular-frame magic: 0xFD2FB528 little-endian = [0x28 0xB5 0x2F 0xFD]
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return fmt.Errorf("read magic: %w", err)
	}
	if magic[0] != 0x28 || magic[1] != 0xb5 || magic[2] != 0x2f || magic[3] != 0xfd {
		return fmt.Errorf("invalid zstd magic: %02x%02x%02x%02x", magic[0], magic[1], magic[2], magic[3])
	}

	// Check last 16 bytes are not all zeros.
	// A valid zstd stream always ends with non-zero data (checksum / frame header).
	// All-zero tail = mmap was pre-allocated but the boundary piece was never written.
	if _, err := f.Seek(-16, io.SeekEnd); err == nil {
		var tail [16]byte
		if _, err := io.ReadFull(f, tail[:]); err == nil {
			if isAllZero(tail[:]) {
				return fmt.Errorf("zero-padded tail: boundary piece was not downloaded (mmap incomplete)")
			}
		}
	}

	// Sample 4 KB at 25%, 50%, 75% — catches large zero-filled holes from
	// incomplete mmap pages or missed torrent pieces in the middle of the file.
	const sampleSize = 4096
	if size > sampleSize*4 {
		var sample [sampleSize]byte
		for _, pct := range []int64{25, 50, 75} {
			offset := size * pct / 100
			if _, err := f.Seek(offset, io.SeekStart); err != nil {
				continue
			}
			n, err := io.ReadFull(f, sample[:])
			if err != nil || n < sampleSize {
				continue
			}
			if isAllZero(sample[:]) {
				return fmt.Errorf("zero-filled region at %d%% offset (%d): likely incomplete download", pct, offset)
			}
		}
	}

	return nil
}

// DeepValidateZst performs a full streaming decode of the .zst file into
// io.Discard. This catches mid-file corruption that QuickValidateZst misses
// (bit flips, truncated zstd frames, etc.). It costs ~1-3s per GB on modern
// hardware but guarantees the entire stream is decodable before we spend
// time processing it.
func DeepValidateZst(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	dec, err := zstd.NewReader(f, zstd.WithDecoderMaxWindow(1<<31))
	if err != nil {
		return fmt.Errorf("zstd reader: %w", err)
	}
	defer dec.Close()

	n, err := io.Copy(io.Discard, dec)
	if err != nil {
		return fmt.Errorf("zstd decode failed after %d bytes: %w", n, err)
	}
	return nil
}

func isAllZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// ValidateParquet performs a basic sanity check on a parquet shard:
// verifies the file is non-empty and has the PAR1 magic at the start and end.
func ValidateParquet(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if fi.Size() < 12 { // min parquet: 4 (magic) + 4 (footer len) + 4 (magic)
		return fmt.Errorf("file too small (%d bytes)", fi.Size())
	}

	// Check leading PAR1 magic
	var head [4]byte
	if _, err := io.ReadFull(f, head[:]); err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	if string(head[:]) != "PAR1" {
		return fmt.Errorf("invalid header magic: %q", head)
	}

	// Check trailing PAR1 magic
	if _, err := f.Seek(-4, io.SeekEnd); err != nil {
		return fmt.Errorf("seek tail: %w", err)
	}
	var tail [4]byte
	if _, err := io.ReadFull(f, tail[:]); err != nil {
		return fmt.Errorf("read tail: %w", err)
	}
	if string(tail[:]) != "PAR1" {
		return fmt.Errorf("invalid tail magic: %q (truncated?)", tail)
	}

	return nil
}

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

	if err := ValidateParquet(shardPath); err != nil {
		os.Remove(shardPath)
		return ShardResult{}, fmt.Errorf("validate shard %d: %w", idx, err)
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
