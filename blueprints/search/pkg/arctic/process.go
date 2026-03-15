package arctic

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/sync/errgroup"
)

// zstdDecoderSem limits concurrent zstd decoders to 1.  Each decoder with a
// 2 GB window (WithDecoderMaxWindow(1<<31)) allocates ~2 GB in its internal
// startStreamDecoder goroutine.  The pipeline may run multiple ProcessZst
// workers concurrently; without this gate two decoders would coexist (4+ GB)
// and OOM a server with ≤12 GB RAM.  The semaphore only guards the scan
// phase — after scanning, the decoder is closed and another worker can start,
// while the first worker continues with DuckDB shard conversion.
var zstdDecoderSem sync.Mutex

type ShardResult struct {
	Index     int
	LocalPath string       // empty when Starting=true
	Rows      int64        // line count (estimate) when Starting=true; actual when done
	SizeBytes int64
	Duration  time.Duration // wall time for this shard (set when Starting=false)
	Starting  bool          // true = shard just started (DuckDB not yet done)
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

	dec, err := zstd.NewReader(f, zstd.WithDecoderMaxWindow(1<<27))
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

// chunkJob describes a completed chunk ready for parquet conversion.
// For Go engine: lines holds in-memory data (path is empty).
// For DuckDB engine: path holds the disk chunk file (lines is nil).
type chunkJob struct {
	path      string   // disk path (DuckDB mode)
	lines     [][]byte // in-memory lines (Go mode)
	index     int
	lineCount int
}

// ProcessZst streams the .zst file at zstPath, writing lines directly to
// temporary JSONL chunk files on disk (never buffering all lines in memory),
// then converts chunks to parquet shards via a pool of DuckDB workers.
//
// The decode and conversion phases are decoupled: the zstd decoder writes
// chunks at full speed while a separate goroutine pool converts them to
// parquet concurrently.  This keeps the 2 GB decoder busy and releases the
// decoder semaphore much sooner, allowing the next file to start decoding.
//
// Concurrency is controlled by cfg.MaxConvertWorkers (default 1 = sequential).
func ProcessZst(ctx context.Context, cfg Config, zstPath, typ, year, mm string,
	cb ShardCallback) (ProcessResult, error) {

	start := time.Now()
	workers := cfg.MaxConvertWorkers
	if workers <= 0 {
		workers = 1
	}

	// Optional CPU profiling: set MIZU_ARCTIC_CPUPROF=path to write a profile.
	if profPath := os.Getenv("MIZU_ARCTIC_CPUPROF"); profPath != "" {
		pf, err := os.Create(profPath)
		if err == nil {
			if err := pprof.StartCPUProfile(pf); err == nil {
				logf("CPU profiling → %s", profPath)
				defer func() {
					pprof.StopCPUProfile()
					pf.Close()
					logf("CPU profile written to %s", profPath)
				}()
			} else {
				pf.Close()
			}
		}
	}

	// Optional memory profiling: set MIZU_ARCTIC_MEMPROF=path to write a heap profile on completion.
	if memProfPath := os.Getenv("MIZU_ARCTIC_MEMPROF"); memProfPath != "" {
		defer func() {
			runtime.GC()
			mf, err := os.Create(memProfPath)
			if err == nil {
				if err := pprof.WriteHeapProfile(mf); err == nil {
					logf("heap profile written to %s", memProfPath)
				}
				mf.Close()
			}
		}()
	}

	// Acquire the decoder semaphore BEFORE opening the file + decoder.
	// This ensures at most one 2 GB zstd window exists process-wide.
	logf("[%s-%s] %s waiting for zstd decoder semaphore…", year, mm, typ)
	zstdDecoderSem.Lock()
	engine := "go"
	if !cfg.UseGoParquet() {
		engine = "duckdb"
	}
	logf("[%s-%s] %s acquired zstd decoder semaphore (%d convert workers, engine=%s)", year, mm, typ, workers, engine)

	f, err := os.Open(zstPath)
	if err != nil {
		zstdDecoderSem.Unlock()
		return ProcessResult{}, fmt.Errorf("open zst: %w", err)
	}
	// Not deferred — f and dec are closed explicitly after scanning to allow
	// immediate GC of the decoder's 2 GB window buffer before returning.

	dec, err := zstd.NewReader(f, zstd.WithDecoderMaxWindow(1<<31))
	if err != nil {
		f.Close()
		zstdDecoderSem.Unlock()
		return ProcessResult{}, fmt.Errorf("zstd reader: %w", err)
	}

	shardDir := cfg.ShardLocalDir(typ, year, mm)
	if err := os.MkdirAll(shardDir, 0o755); err != nil {
		dec.Close()
		f.Close()
		zstdDecoderSem.Unlock()
		return ProcessResult{}, fmt.Errorf("mkdir shards: %w", err)
	}

	// --- Convert worker pool (errgroup with limit) ---
	// errgroup.SetLimit provides natural backpressure: g.Go() blocks when
	// all worker slots are busy, preventing the scanner from racing ahead.
	// No channels or manual draining needed — eliminates deadlock risk.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

	var result ProcessResult
	var resultMu sync.Mutex

	// --- Decode phase: scan JSONL, dispatch to workers ---
	scanner := bufio.NewScanner(dec)
	scanner.Buffer(make([]byte, 16*1024*1024), 16*1024*1024)

	chunkIdx := 0
	lineCount := 0
	totalChunks := 0
	useGoMem := cfg.UseGoParquet() // in-memory chunks for Go engine

	// State for disk-based chunks (DuckDB engine).
	var chunkPath string
	var chunkFile *os.File
	var chunkWriter *bufio.Writer

	// State for in-memory chunks (Go engine).
	var memBuf [][]byte

	if useGoMem {
		memBuf = make([][]byte, 0, cfg.ChunkLines)
	} else {
		chunkPath = cfg.ChunkPath(chunkIdx)
		_ = os.MkdirAll(filepath.Dir(chunkPath), 0o755)
		var createErr error
		chunkFile, createErr = os.Create(chunkPath)
		if createErr != nil {
			dec.Close()
			f.Close()
			zstdDecoderSem.Unlock()
			_ = g.Wait()
			return ProcessResult{}, fmt.Errorf("create chunk: %w", createErr)
		}
		chunkWriter = bufio.NewWriterSize(chunkFile, 8*1024*1024)
	}

	// dispatchChunk sends the current chunk to the errgroup convert pool.
	dispatchChunk := func() error {
		if lineCount == 0 {
			if !useGoMem && chunkFile != nil {
				chunkFile.Close()
				os.Remove(chunkPath)
			}
			return nil
		}

		var job chunkJob
		if useGoMem {
			job = chunkJob{lines: memBuf, index: chunkIdx, lineCount: lineCount}
			memBuf = make([][]byte, 0, cfg.ChunkLines) // fresh buffer for next chunk
		} else {
			if err := chunkWriter.Flush(); err != nil {
				chunkFile.Close()
				os.Remove(chunkPath)
				return fmt.Errorf("flush chunk: %w", err)
			}
			chunkFile.Close()
			job = chunkJob{path: chunkPath, index: chunkIdx, lineCount: lineCount}
		}

		// g.Go blocks when all worker slots are busy (backpressure).
		g.Go(func() error {
			if cb != nil {
				cb(ShardResult{Index: job.index, Rows: int64(job.lineCount), Starting: true})
			}
			shardStart := time.Now()
			var sr ShardResult
			var err error
			if job.lines != nil {
				sr, err = convertChunkToShardGoMem(gctx, cfg, job.lines, typ, year, mm, job.index)
				job.lines = nil // release memory immediately
			} else if cfg.UseGoParquet() {
				sr, err = convertChunkToShardGo(gctx, cfg, job.path, typ, year, mm, job.index)
			} else {
				sr, err = convertChunkToShard(gctx, cfg, job.path, typ, year, mm, job.index)
			}
			if err != nil {
				return fmt.Errorf("shard %d: %w", job.index, err)
			}
			sr.Duration = time.Since(shardStart)
			if cb != nil {
				cb(sr)
			}
			resultMu.Lock()
			result.Shards = append(result.Shards, sr)
			result.TotalRows += sr.Rows
			result.TotalSize += sr.SizeBytes
			resultMu.Unlock()
			return nil
		})

		totalChunks++
		chunkIdx++
		lineCount = 0
		return nil
	}

	openNextChunk := func() error {
		if useGoMem {
			return nil // memBuf already reset in dispatchChunk
		}
		chunkPath = cfg.ChunkPath(chunkIdx)
		var err error
		chunkFile, err = os.Create(chunkPath)
		if err != nil {
			return fmt.Errorf("create chunk: %w", err)
		}
		chunkWriter = bufio.NewWriterSize(chunkFile, 8*1024*1024)
		return nil
	}

	var scanErr error
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			scanErr = ctx.Err()
			goto scanDone
		default:
		}
		if useGoMem {
			// Copy the line — scanner.Bytes() is reused.
			memBuf = append(memBuf, append([]byte(nil), scanner.Bytes()...))
		} else {
			chunkWriter.Write(scanner.Bytes())
			chunkWriter.WriteByte('\n')
		}
		lineCount++
		if lineCount >= cfg.ChunkLines {
			if err := dispatchChunk(); err != nil {
				scanErr = err
				goto scanDone
			}
			if err := openNextChunk(); err != nil {
				scanErr = err
				goto scanDone
			}
		}
	}
	if err := scanner.Err(); err != nil {
		scanErr = fmt.Errorf("scan jsonl: %w", err)
	}

scanDone:
	// Close decoder + file to free the 2 GB zstd window buffer.
	dec.Close()
	f.Close()
	runtime.GC()
	debug.FreeOSMemory()

	// Release the semaphore — another ProcessZst can now start decoding.
	// Convert workers continue running in the background.
	logf("[%s-%s] %s released zstd decoder semaphore (scan done, %d chunks dispatched)",
		year, mm, typ, totalChunks)
	zstdDecoderSem.Unlock()

	if scanErr != nil {
		if !useGoMem && chunkFile != nil {
			chunkFile.Close()
			os.Remove(chunkPath)
		}
		// Wait for in-flight workers; gctx is already cancelled via parent ctx or will finish.
		_ = g.Wait()
		return ProcessResult{}, scanErr
	}

	// Dispatch final partial chunk.
	logf("[%s-%s] %s scan done, dispatching final chunk (%d lines)", year, mm, typ, lineCount)
	if err := dispatchChunk(); err != nil {
		_ = g.Wait()
		return ProcessResult{}, err
	}

	// Wait for all convert workers to finish.
	if err := g.Wait(); err != nil {
		return ProcessResult{}, err
	}

	// Sort shards by index — async workers may complete out of order.
	sort.Slice(result.Shards, func(i, j int) bool {
		return result.Shards[i].Index < result.Shards[j].Index
	})

	result.Duration = time.Since(start)
	logf("[%s-%s] %s processing complete: %d shards, %d rows in %s",
		year, mm, typ, len(result.Shards), result.TotalRows, result.Duration.Round(time.Second))
	return result, nil
}

// convertChunkToShard imports a JSONL chunk file into an in-memory DuckDB
// instance and exports it as a zstd-compressed parquet shard.  With ChunkLines
// reduced to 500K, each chunk is ~250 MB of text — within DuckDB's 512 MB
// memory limit — so no disk spilling is needed.  In-memory avoids the mmap
// overhead that disk-backed DuckDB adds to RSS.
func convertChunkToShard(ctx context.Context, cfg Config, chunkPath,
	typ, year, mm string, idx int) (ShardResult, error) {

	shardPath := cfg.ShardLocalPath(typ, year, mm, idx)
	if err := os.MkdirAll(filepath.Dir(shardPath), 0o755); err != nil {
		return ShardResult{}, err
	}

	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }

	// In-memory DuckDB — no file mmap, memory bounded by SET memory_limit.
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return ShardResult{}, fmt.Errorf("duckdb open: %w", err)
	}
	// mallocTrim must be deferred BEFORE db.Close() so it runs AFTER (LIFO order).
	// After db.Close() releases DuckDB's C-side heap, glibc retains those pages
	// as RSS. malloc_trim(0) forces glibc to return them to the OS, keeping RSS
	// bounded across sequential chunk processing.
	defer mallocTrim()
	defer db.Close()

	// Cap DuckDB memory to keep RSS within server limits (500K-line chunks
	// decompress to ~250 MB, so 512 MB headroom is sufficient).
	db.ExecContext(ctx, fmt.Sprintf("SET memory_limit='%s'", cfg.DuckDBMemory()))

	selectCols := commentsSelect
	readCols := commentsReadColumns
	if typ == "submissions" {
		selectCols = submissionsSelect
		readCols = submissionsReadColumns
	}

	importSQL := fmt.Sprintf(`CREATE TABLE data AS
SELECT %s
FROM read_json('%s',
    format='newline_delimited',
    columns=%s,
    maximum_object_size=10485760,
    ignore_errors=true
)`, selectCols, esc(chunkPath), readCols)

	if _, err := db.ExecContext(ctx, importSQL); err != nil {
		return ShardResult{}, fmt.Errorf("duckdb import: %w", err)
	}

	// Delete chunk file immediately after import to free disk space.
	os.Remove(chunkPath)

	var rowCount int64
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM data").Scan(&rowCount)

	exportSQL := fmt.Sprintf(
		"COPY data TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD, COMPRESSION_LEVEL 3, ROW_GROUP_SIZE 131072)",
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

// commentsReadColumns tells DuckDB read_json the exact schema, avoiding the
// expensive sampling pass that read_json_auto performs on every chunk.
// All fields are VARCHAR to tolerate type changes across Reddit archive months.
const commentsReadColumns = `{
    id: 'VARCHAR', author: 'VARCHAR', subreddit: 'VARCHAR',
    body: 'VARCHAR', score: 'VARCHAR', created_utc: 'VARCHAR',
    link_id: 'VARCHAR', parent_id: 'VARCHAR',
    distinguished: 'VARCHAR', author_flair_text: 'VARCHAR'
}`

// submissionsReadColumns — explicit schema for submissions JSON.
const submissionsReadColumns = `{
    id: 'VARCHAR', author: 'VARCHAR', subreddit: 'VARCHAR',
    title: 'VARCHAR', selftext: 'VARCHAR', score: 'VARCHAR',
    created_utc: 'VARCHAR', num_comments: 'VARCHAR',
    url: 'VARCHAR', over_18: 'VARCHAR',
    link_flair_text: 'VARCHAR', author_flair_text: 'VARCHAR'
}`

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
