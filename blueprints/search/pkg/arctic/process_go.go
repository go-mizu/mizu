package arctic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	parquet "github.com/parquet-go/parquet-go"
	pzstd "github.com/parquet-go/parquet-go/compress/zstd"
)

// commentRow matches the parquet schema produced by convertChunkToShard's DuckDB
// query (commentsSelect). Field order and types must match for output parity.
type commentRow struct {
	ID              *string    `parquet:"id,optional"`
	Author          *string    `parquet:"author,optional"`
	Subreddit       *string    `parquet:"subreddit,optional"`
	Body            *string    `parquet:"body,optional"`
	Score           *int64     `parquet:"score,optional"`
	CreatedUTC      *int64     `parquet:"created_utc,optional"`
	CreatedAt       *time.Time `parquet:"created_at,optional,timestamp(millisecond)"`
	BodyLength      int32      `parquet:"body_length"`
	LinkID          *string    `parquet:"link_id,optional"`
	ParentID        *string    `parquet:"parent_id,optional"`
	Distinguished   *string    `parquet:"distinguished,optional"`
	AuthorFlairText *string    `parquet:"author_flair_text,optional"`
}

// submissionRow matches the parquet schema from submissionsSelect.
type submissionRow struct {
	ID              *string    `parquet:"id,optional"`
	Author          *string    `parquet:"author,optional"`
	Subreddit       *string    `parquet:"subreddit,optional"`
	Title           *string    `parquet:"title,optional"`
	Selftext        *string    `parquet:"selftext,optional"`
	Score           *int64     `parquet:"score,optional"`
	CreatedUTC      *int64     `parquet:"created_utc,optional"`
	CreatedAt       *time.Time `parquet:"created_at,optional,timestamp(millisecond)"`
	TitleLength     int32      `parquet:"title_length"`
	NumComments     *int64     `parquet:"num_comments,optional"`
	URL             *string    `parquet:"url,optional"`
	Over18          *bool      `parquet:"over_18,optional"`
	LinkFlairText   *string    `parquet:"link_flair_text,optional"`
	AuthorFlairText *string    `parquet:"author_flair_text,optional"`
}

// goParquetZstdCodec uses SpeedBestCompression (~level 11) from klauspost/compress.
// This is ~5x faster than DuckDB's ZSTD level 22 with only ~5% larger output.
// Pure Go — no CGo overhead per data page compression call.
var goParquetZstdCodec = &pzstd.Codec{Level: pzstd.SpeedBestCompression}

// convertChunkToShardGoMem converts in-memory JSONL lines directly to a parquet
// shard — no intermediate disk file. This is the fast path for the Go engine
// where lines are accumulated during the scan phase.
func convertChunkToShardGoMem(ctx context.Context, cfg Config, lines [][]byte,
	typ, year, mm string, idx int) (ShardResult, error) {

	shardPath := cfg.ShardLocalPath(typ, year, mm, idx)
	if err := os.MkdirAll(filepath.Dir(shardPath), 0o755); err != nil {
		return ShardResult{}, err
	}

	var rowCount int64
	var parseErr error

	if typ == "submissions" {
		rowCount, parseErr = writeParquetFromLines[submissionRow](
			ctx, lines, shardPath, parseSubmissionRow)
	} else {
		rowCount, parseErr = writeParquetFromLines[commentRow](
			ctx, lines, shardPath, parseCommentRow)
	}

	if parseErr != nil {
		os.Remove(shardPath)
		return ShardResult{}, fmt.Errorf("go parquet mem convert: %w", parseErr)
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

// writeParquetFromLines converts in-memory JSONL lines to a parquet file.
func writeParquetFromLines[T any](ctx context.Context, lines [][]byte, shardPath string,
	parseFn func(map[string]any) T) (int64, error) {

	sf, err := os.Create(shardPath)
	if err != nil {
		return 0, fmt.Errorf("create shard: %w", err)
	}
	defer sf.Close()

	w := parquet.NewGenericWriter[T](sf,
		parquet.Compression(goParquetZstdCodec),
		parquet.MaxRowsPerRowGroup(131072),
	)

	const batchSize = 4096
	batch := make([]T, 0, batchSize)
	var rowCount int64
	var badLines int64

	for _, line := range lines {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		if len(line) == 0 {
			continue
		}

		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			badLines++
			continue
		}

		batch = append(batch, parseFn(m))
		if len(batch) >= batchSize {
			if _, err := w.Write(batch); err != nil {
				return 0, fmt.Errorf("write batch: %w", err)
			}
			rowCount += int64(len(batch))
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if _, err := w.Write(batch); err != nil {
			return 0, fmt.Errorf("write final batch: %w", err)
		}
		rowCount += int64(len(batch))
	}

	if err := w.Close(); err != nil {
		return 0, fmt.Errorf("close parquet writer: %w", err)
	}

	if badLines > 0 {
		logf("go parquet: skipped %d malformed JSON lines (mem)", badLines)
	}

	return rowCount, nil
}

// convertChunkToShardGo reads a JSONL chunk file and writes a parquet shard
// using Go's parquet-go library. This replaces DuckDB for the conversion step.
//
// Compared to the DuckDB path:
//   - No DuckDB process startup/teardown overhead
//   - No schema inference (columns are compiled in)
//   - No C heap fragmentation (pure Go, GC-managed)
//   - ZSTD level ~11 vs DuckDB's level 22 (~5% larger output, ~3x faster compress)
func convertChunkToShardGo(ctx context.Context, cfg Config, chunkPath,
	typ, year, mm string, idx int) (ShardResult, error) {

	shardPath := cfg.ShardLocalPath(typ, year, mm, idx)
	if err := os.MkdirAll(filepath.Dir(shardPath), 0o755); err != nil {
		return ShardResult{}, err
	}

	var rowCount int64
	var parseErr error

	if typ == "submissions" {
		rowCount, parseErr = writeParquetFromJSONL[submissionRow](
			ctx, chunkPath, shardPath, parseSubmissionRow, cfg)
	} else {
		rowCount, parseErr = writeParquetFromJSONL[commentRow](
			ctx, chunkPath, shardPath, parseCommentRow, cfg)
	}

	if parseErr != nil {
		os.Remove(shardPath)
		return ShardResult{}, fmt.Errorf("go parquet convert: %w", parseErr)
	}

	// Delete chunk file after successful conversion.
	os.Remove(chunkPath)

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

// writeParquetFromJSONL is the generic core: reads JSONL, parses each line
// with parseFn, writes rows to a parquet file via GenericWriter.
func writeParquetFromJSONL[T any](ctx context.Context, chunkPath, shardPath string,
	parseFn func(map[string]any) T, cfg Config) (int64, error) {

	// Open chunk file for reading.
	cf, err := os.Open(chunkPath)
	if err != nil {
		return 0, fmt.Errorf("open chunk: %w", err)
	}
	defer cf.Close()

	// Create output parquet file.
	sf, err := os.Create(shardPath)
	if err != nil {
		return 0, fmt.Errorf("create shard: %w", err)
	}
	defer sf.Close()

	w := parquet.NewGenericWriter[T](sf,
		parquet.Compression(goParquetZstdCodec),
		parquet.MaxRowsPerRowGroup(131072),
	)

	scanner := bufio.NewScanner(cf)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	// Batch rows to reduce Write() call overhead.
	const batchSize = 4096
	batch := make([]T, 0, batchSize)
	var rowCount int64
	var badLines int64

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			badLines++
			continue // ignore_errors=true equivalent
		}

		batch = append(batch, parseFn(m))
		if len(batch) >= batchSize {
			if _, err := w.Write(batch); err != nil {
				return 0, fmt.Errorf("write batch: %w", err)
			}
			rowCount += int64(len(batch))
			batch = batch[:0]
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan chunk: %w", err)
	}

	// Flush remaining rows.
	if len(batch) > 0 {
		if _, err := w.Write(batch); err != nil {
			return 0, fmt.Errorf("write final batch: %w", err)
		}
		rowCount += int64(len(batch))
	}

	if err := w.Close(); err != nil {
		return 0, fmt.Errorf("close parquet writer: %w", err)
	}

	if badLines > 0 {
		logf("go parquet: skipped %d malformed JSON lines in %s", badLines, filepath.Base(chunkPath))
	}

	return rowCount, nil
}

// --- Row parsers ---

func parseCommentRow(m map[string]any) commentRow {
	utc := jsonInt64(m, "created_utc")
	body := jsonStr(m, "body")
	var bodyLen int32
	if body != nil {
		bodyLen = int32(len(*body))
	}
	return commentRow{
		ID:              jsonStr(m, "id"),
		Author:          jsonStr(m, "author"),
		Subreddit:       jsonStr(m, "subreddit"),
		Body:            body,
		Score:           jsonInt64(m, "score"),
		CreatedUTC:      utc,
		CreatedAt:       epochToTime(utc),
		BodyLength:      bodyLen,
		LinkID:          jsonStr(m, "link_id"),
		ParentID:        jsonStr(m, "parent_id"),
		Distinguished:   jsonStr(m, "distinguished"),
		AuthorFlairText: jsonStr(m, "author_flair_text"),
	}
}

func parseSubmissionRow(m map[string]any) submissionRow {
	utc := jsonInt64(m, "created_utc")
	title := jsonStr(m, "title")
	var titleLen int32
	if title != nil {
		titleLen = int32(len(*title))
	}
	return submissionRow{
		ID:              jsonStr(m, "id"),
		Author:          jsonStr(m, "author"),
		Subreddit:       jsonStr(m, "subreddit"),
		Title:           title,
		Selftext:        jsonStr(m, "selftext"),
		Score:           jsonInt64(m, "score"),
		CreatedUTC:      utc,
		CreatedAt:       epochToTime(utc),
		TitleLength:     titleLen,
		NumComments:     jsonInt64(m, "num_comments"),
		URL:             jsonStr(m, "url"),
		Over18:          jsonBool(m, "over_18"),
		LinkFlairText:   jsonStr(m, "link_flair_text"),
		AuthorFlairText: jsonStr(m, "author_flair_text"),
	}
}

// --- JSON field extractors ---
// These handle the messy Reddit JSON where fields can be strings, numbers,
// booleans, null, or missing.

func jsonStr(m map[string]any, key string) *string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch s := v.(type) {
	case string:
		return &s
	case float64:
		str := strconv.FormatFloat(s, 'f', -1, 64)
		return &str
	case bool:
		str := strconv.FormatBool(s)
		return &str
	default:
		return nil
	}
}

func jsonInt64(m map[string]any, key string) *int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		i := int64(n)
		return &i
	case string:
		// Try parsing as int first, then float.
		if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return &i
		}
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			i := int64(f)
			return &i
		}
		return nil
	default:
		return nil
	}
}

func jsonBool(m map[string]any, key string) *bool {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch b := v.(type) {
	case bool:
		return &b
	case string:
		if b == "true" || b == "1" {
			t := true
			return &t
		}
		if b == "false" || b == "0" {
			f := false
			return &f
		}
		return nil
	case float64:
		t := b != 0
		return &t
	default:
		return nil
	}
}

func epochToTime(utc *int64) *time.Time {
	if utc == nil {
		return nil
	}
	t := time.Unix(*utc, 0).UTC()
	return &t
}
