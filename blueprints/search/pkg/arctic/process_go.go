package arctic

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	parquet "github.com/parquet-go/parquet-go"
	pzstd "github.com/parquet-go/parquet-go/compress/zstd"
	"github.com/tidwall/gjson"
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

// goParquetZstdCodec uses SpeedDefault (~level 3) from klauspost/compress,
// matching DuckDB's COMPRESSION_LEVEL 3. This is ~3x faster to compress than
// SpeedBestCompression (level 11) with only ~5-10% larger output.
var goParquetZstdCodec = &pzstd.Codec{Level: pzstd.SpeedDefault}

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
// Uses gjson for zero-allocation selective field extraction — only the 10-12
// needed fields are parsed, skipping the hundreds of other Reddit JSON fields.
func writeParquetFromLines[T any](ctx context.Context, lines [][]byte, shardPath string,
	parseFn func([]byte) T) (int64, error) {

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

		if len(line) == 0 || line[0] != '{' {
			badLines++
			continue
		}

		batch = append(batch, parseFn(line))
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
// with parseFn (gjson-based), writes rows to a parquet file via GenericWriter.
func writeParquetFromJSONL[T any](ctx context.Context, chunkPath, shardPath string,
	parseFn func([]byte) T, cfg Config) (int64, error) {

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
		if len(line) == 0 || line[0] != '{' {
			badLines++
			continue
		}

		// Copy line — scanner.Bytes() is reused on next Scan().
		lineCopy := append([]byte(nil), line...)
		batch = append(batch, parseFn(lineCopy))
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

// --- Row parsers (gjson-based, zero-allocation field extraction) ---
// These use gjson.GetBytes to extract only the needed fields from each JSON
// line, skipping the hundreds of other Reddit fields. This is 10-20x faster
// than encoding/json.Unmarshal into map[string]any because:
//   - No map allocation per row
//   - No interface{} boxing for each value
//   - Only the needed fields are parsed (10-12 out of 30+)
//   - String values reference the original JSON bytes (zero-copy)

func parseCommentRow(line []byte) commentRow {
	utc := gInt64(line, "created_utc")
	body := gStr(line, "body")
	var bodyLen int32
	if body != nil {
		bodyLen = int32(len(*body))
	}
	return commentRow{
		ID:              gStr(line, "id"),
		Author:          gStr(line, "author"),
		Subreddit:       gStr(line, "subreddit"),
		Body:            body,
		Score:           gInt64(line, "score"),
		CreatedUTC:      utc,
		CreatedAt:       epochToTime(utc),
		BodyLength:      bodyLen,
		LinkID:          gStr(line, "link_id"),
		ParentID:        gStr(line, "parent_id"),
		Distinguished:   gStr(line, "distinguished"),
		AuthorFlairText: gStr(line, "author_flair_text"),
	}
}

func parseSubmissionRow(line []byte) submissionRow {
	utc := gInt64(line, "created_utc")
	title := gStr(line, "title")
	var titleLen int32
	if title != nil {
		titleLen = int32(len(*title))
	}
	return submissionRow{
		ID:              gStr(line, "id"),
		Author:          gStr(line, "author"),
		Subreddit:       gStr(line, "subreddit"),
		Title:           title,
		Selftext:        gStr(line, "selftext"),
		Score:           gInt64(line, "score"),
		CreatedUTC:      utc,
		CreatedAt:       epochToTime(utc),
		TitleLength:     titleLen,
		NumComments:     gInt64(line, "num_comments"),
		URL:             gStr(line, "url"),
		Over18:          gBool(line, "over_18"),
		LinkFlairText:   gStr(line, "link_flair_text"),
		AuthorFlairText: gStr(line, "author_flair_text"),
	}
}

// --- gjson field extractors ---
// Handle the messy Reddit JSON where fields can be strings, numbers,
// booleans, null, or missing. gjson returns Result.Type == gjson.Null
// for missing fields, so we map those to nil pointers.

func gStr(line []byte, key string) *string {
	r := gjson.GetBytes(line, key)
	if !r.Exists() || r.Type == gjson.Null {
		return nil
	}
	// r.Str is zero-copy for string values. For numbers/bools, use String().
	var s string
	switch r.Type {
	case gjson.String:
		s = r.Str
	default:
		s = r.Raw // number or bool as raw JSON text
	}
	return &s
}

func gInt64(line []byte, key string) *int64 {
	r := gjson.GetBytes(line, key)
	if !r.Exists() || r.Type == gjson.Null {
		return nil
	}
	var i int64
	switch r.Type {
	case gjson.Number:
		i = int64(r.Num)
	case gjson.String:
		var err error
		i, err = strconv.ParseInt(r.Str, 10, 64)
		if err != nil {
			f, ferr := strconv.ParseFloat(r.Str, 64)
			if ferr != nil {
				return nil
			}
			i = int64(f)
		}
	default:
		return nil
	}
	return &i
}

func gBool(line []byte, key string) *bool {
	r := gjson.GetBytes(line, key)
	if !r.Exists() || r.Type == gjson.Null {
		return nil
	}
	var b bool
	switch r.Type {
	case gjson.True:
		b = true
	case gjson.False:
		b = false
	case gjson.String:
		b = r.Str == "true" || r.Str == "1"
	case gjson.Number:
		b = r.Num != 0
	default:
		return nil
	}
	return &b
}

func epochToTime(utc *int64) *time.Time {
	if utc == nil {
		return nil
	}
	t := time.Unix(*utc, 0).UTC()
	return &t
}
