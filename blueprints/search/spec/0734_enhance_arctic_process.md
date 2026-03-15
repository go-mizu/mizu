# 0734: Enhance Arctic Processing Pipeline

**Status:** Draft
**Date:** 2026-03-15
**Scope:** `pkg/arctic/process.go`, `pkg/arctic/budget.go`, `pkg/arctic/config.go`

## Problem

The arctic processing pipeline (`ProcessZst`) is bottlenecked by **interleaved decode and conversion**: the zstd decoder (2 GB RAM) sits idle while DuckDB converts each chunk to parquet. On a file with 10 chunks of 500K lines each:

```
Current timeline (single file):
[decode 500K ~30s] → [duckdb ~45s] → [decode 500K ~30s] → [duckdb ~45s] → ...
                      ^^^ idle ^^^                          ^^^ idle ^^^

Decoder utilization: ~40%
Total time: 10 × (30s + 45s) = 750s = 12.5 min
```

The zstd decoder semaphore (`zstdDecoderSem`) ensures only one 2 GB decoder exists process-wide. Because conversion blocks inside the scan loop, the semaphore is held for the entire duration (decode + all conversions), preventing the next file from starting its decode phase.

## Solution: Async Chunk Conversion (Producer-Consumer)

Decouple decode from conversion using a goroutine pool:

```
New timeline (single file):
Decoder:  [decode chunk0] [decode chunk1] [decode chunk2] ... [done, release sem]
Workers:                  [convert chunk0] [convert chunk1]   [convert chunk2] ...
                          [                convert chunk0  ]  [convert chunk1  ] ...

Decoder utilization: ~95% (only waits if convert queue is full)
Total time: max(decode_total, convert_total) ≈ 300s + 45s = 345s = 5.75 min
```

### Architecture

```
ProcessZst goroutine                     Convert worker pool
─────────────────────                    ────────────────────
                                         ┌─ worker 1 ─────┐
scan lines → write chunk → chunkCh ────→ │ convertChunk()  │ → results
             write chunk → chunkCh ────→ │ convertChunk()  │ → results
             ...                         └─────────────────┘
                                         ┌─ worker 2 ─────┐
             write chunk → chunkCh ────→ │ convertChunk()  │ → results
             ...                         └─────────────────┘

After scan done:
  close(chunkCh) → workers drain → wg.Wait() → collect results
```

### Key Design Decisions

1. **Bounded channel** — `chunkCh` has capacity = `MaxConvertWorkers`. If all workers are busy, the decoder blocks on send, providing natural backpressure. This prevents unbounded disk usage from too many unconverted chunk files.

2. **DuckDB concurrency semaphore** — A new `duckdbSem` (counting semaphore via `chan struct{}`) limits concurrent DuckDB instances. This is separate from `zstdDecoderSem` because DuckDB uses ~512 MB per instance vs 2 GB for the decoder.

3. **Chunk file lifecycle** — Each chunk file is created by the decoder, sent to a worker via the channel, and deleted by the worker after DuckDB import. No shared file access.

4. **Error propagation** — Workers send `(ShardResult, error)` pairs to a results channel. The main goroutine collects results after `wg.Wait()`. First error cancels remaining work via context.

5. **Callback ordering** — `Starting` callbacks fire immediately when a worker picks up a chunk. Completion callbacks may arrive out of order (worker 2 might finish before worker 1). The caller (pipeline) already handles this — shard upload order doesn't matter.

6. **Memory safety** — Peak RSS with 2 convert workers on server2 (11 GB):
   - OS + torrent + upload: 4 GB reserved
   - zstd decoder: 2 GB (1 at a time)
   - DuckDB workers: 2 × 512 MB = 1 GB
   - Scanner + chunk buffers: ~100 MB
   - **Total: ~7.1 GB** — safe 3.9 GB margin

## Changes

### 1. `process.go` — Async Chunk Conversion

**Before:** `closeAndConvert()` called inside scan loop blocks decoder.

**After:**
- Scan loop writes chunk to disk, sends `chunkJob{path, index, lineCount}` to `chunkCh`
- Worker pool (N goroutines) reads from `chunkCh`, calls `convertChunkToShard()`, sends result to `resultCh`
- After scan completes + decoder closed + sem released, main goroutine closes `chunkCh`, waits for workers, collects results
- `ShardCallback` for `Starting=true` fires when worker picks up job; completion callback fires when conversion done

```go
type chunkJob struct {
    path      string
    index     int
    lineCount int
}
```

The decoder goroutine flow:
1. Acquire `zstdDecoderSem`
2. Open decoder
3. Scan loop: write chunks to disk, send jobs to `chunkCh` (blocks if workers busy = backpressure)
4. Close decoder + GC + release `zstdDecoderSem`
5. Close `chunkCh` → workers drain remaining chunks
6. `wg.Wait()` → collect all results from `resultCh`

### 2. `budget.go` — Add `MaxConvertWorkers`

New field in `ResourceBudget`:

```go
MaxConvertWorkers int `json:"max_convert_workers"` // concurrent DuckDB shard conversions
```

Computation:
```go
// Each DuckDB instance uses ~512 MB. With the zstd decoder potentially active,
// budget convert workers from remaining headroom.
// usableRAM already has 4 GB reserved for OS.
// Reserve 2.5 GB for active decoder + scanner.
convertRAM := usableRAM - 2.5
MaxConvertWorkers = int(convertRAM / 0.6) // 512 MB + 100 MB overhead per worker
// Cap at CPU cores (DuckDB is CPU-bound during export).
// Cap at MaxProcess (no point having more converters than process slots).
// Minimum 1.
```

Server 2 (11 GB RAM, 6 cores):
- `usableRAM = 11 - 4 = 7 GB`
- `convertRAM = 7 - 2.5 = 4.5 GB`
- `MaxConvertWorkers = int(4.5 / 0.6) = 7` → capped by CPU (6/2=3) → capped by MaxProcess (2) → **2 workers**

Beefy server (256 GB RAM, 20 cores):
- `usableRAM = 252 GB`
- `convertRAM = 249.5 GB`
- `MaxConvertWorkers = 415` → capped by CPU (10) → capped by MaxProcess (3) → **3 workers**

### 3. `config.go` — Environment Override

```go
MIZU_ARCTIC_MAX_CONVERT=N  // override MaxConvertWorkers
```

### 4. Chunk Size Increase (Optional, Separate Concern)

Keep `ChunkLines` at 500K for now. The async conversion already eliminates the overhead of blocking. Increasing chunk size would reduce DuckDB open/close cycles but risks exceeding the 512 MB memory limit on wider schemas. Can tune later with data.

## Performance Estimate

### Server 2 (11 GB RAM, 6 cores) — Single File with 10 Chunks

**Before:**
- Decode: 10 × 30s = 300s
- Convert: 10 × 45s = 450s
- Total: 750s (12.5 min) — sequential, decoder idle during convert
- Semaphore held: 750s

**After (2 convert workers):**
- Decode: 300s (runs at full speed, slight pauses for backpressure)
- Convert: 10 × 45s / 2 workers = 225s (overlaps with decode)
- Total: max(300, 225) + last-batch drain ≈ 345s (5.75 min)
- Semaphore held: ~300s (decode only, +/- backpressure)
- **Speedup: ~2.2×**

### Pipeline Effect (Multiple Files)

With the decoder semaphore held ~300s instead of ~750s, the next file's decode can start ~450s sooner. Over a batch of 20 files:
- Before: 20 × 750s = 15,000s (4.2 hours)
- After: 300s + 19 × max(300, 225) = 300 + 5,700 = 6,000s (1.7 hours)
- **Pipeline speedup: ~2.5×**

## Safety

1. **OOM protection**: `MaxConvertWorkers` computed from RAM budget, DuckDB memory capped per-instance, convert semaphore enforces limit
2. **Disk space**: Bounded chunk files on disk — at most `MaxConvertWorkers` unconverted chunks exist simultaneously (channel backpressure)
3. **Error handling**: Context cancellation propagates to all workers; first error stops accepting new chunks
4. **Crash recovery**: No change — chunks are ephemeral work files, cleaned up on restart

## Testing

- Existing tests continue to work (single-threaded path when `MaxConvertWorkers=1`)
- Manual test on server 2 with a medium file (RC_2010-01.zst) — compare wall time before/after
- Monitor RSS via `ps aux` during processing — verify peak stays under budget
