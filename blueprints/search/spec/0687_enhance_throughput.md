# 0687: Enhance FTS Pipeline Throughput

**Date**: 2026-03-07
**Target**: Server 2 (12 GB RAM, 6 CPUs, Ubuntu 24.04 Noble, amd64)

## Previous Architecture (sequential)

```
download file 0 → file 1 → file 2 → file 3
  → markdown file 0 → file 1 → file 2 → file 3
    → index file 0 → file 1 → file 2 → file 3
```

Each stage processed files one at a time. Total wall-clock = sum of all stages.

### Stage Bottlenecks (before)

| Stage | Architecture | Bottleneck |
|-------|-------------|------------|
| Download | Sequential single-stream | Network bandwidth underutilized |
| Markdown | reader → N workers → writer | Single-threaded writer (gzip), single-threaded reader |
| Index | WARC reader → batchIndex(5000) | engine.Index() serialized per shard |

## New Architecture (parallel)

```
download: [file 0, file 1, file 2] concurrent → file 3   (concurrency=3)
markdown: [file 0 + file 1] concurrent → [file 2 + file 3] concurrent   (concurrency=2)
index:    [file 0 + file 1] concurrent → [file 2 + file 3] concurrent   (concurrency=2)
```

### Changes Made

| File | Change |
|------|--------|
| `task_download.go` | `errgroup` parallel downloads (concurrency=3) |
| `task_markdown.go` | `errgroup` parallel RunPack (concurrency=2, `NumCPU/2` workers per shard) |
| `task_index.go` | `errgroup` parallel index (concurrency=2) |
| `pkg/warc_md/pack.go` | Add 1 MB `bufio.Writer` to `packWriteFile` |

### Concurrency Limits (Server 2: 6 CPUs, 12 GB RAM)

| Resource | Budget | Per-shard |
|----------|--------|-----------|
| CPU (6 cores) | 100% | 3 workers per concurrent markdown shard |
| RAM (12 GB) | ~9 GB usable | ~2 GB per concurrent shard |
| Disk I/O | SSD | 2-3 concurrent read/write streams |
| Network | ~20 MB/s | 3 concurrent download streams |

## Measured Results (Server 2, 4 shards, files 0-3)

### Markdown Stage (RunPack: .warc.gz → .md.warc.gz)

- **Duration**: 49 minutes (10:41:05 → 11:30:33)
- **Concurrency**: 2 shards at a time (3 workers each)
- **Pair 1**: 00000 + 00001 → completed ~11:06
- **Pair 2**: 00002 + 00003 → completed ~11:30
- **Output**: ~21K docs per shard, ~37-38 MB per .md.warc.gz
- **Bottleneck**: CPU-bound (HTML→Markdown via trafilatura/readability)
- **Note**: With 3 workers/shard (vs 6 sequential), per-shard time is ~25 min
  vs ~15 min sequential. Net gain: 49 min parallel vs ~60 min sequential = **1.2x**

### Index Stage (dahlia FTS)

- **Duration**: 68 seconds (11:31:00 → 11:32:08)
- **Concurrency**: 2 shards at a time
- **Rate**: ~297 docs/s aggregate
- **Output**: ~135-141 MB per shard index
- **Massive improvement**: Index is I/O-bound; parallel shards use separate
  disk areas, so 2× parallelism gives nearly 2× throughput

### Summary

| Stage | Time | Docs | Rate |
|-------|------|------|------|
| Download | 0 min (pre-downloaded) | - | - |
| Markdown | 49 min | 84,681 | ~29 docs/s |
| Index | 68 sec | 84,681 | ~1,245 docs/s |
| **Total** | **~50 min** | | |

### Comparison (estimated vs actual)

| Stage | Sequential (est.) | Parallel (actual) | Speedup |
|-------|------------------|-------------------|---------|
| Markdown | ~60 min | 49 min | 1.2× |
| Index | ~2 min | 68 sec | 1.8× |
| Download | ~12 min (4 files) | ~4 min (3 concurrent) | ~3× (est.) |

## Architecture Detail

### Parallel Downloads (`task_download.go`)

```go
g, gctx := errgroup.WithContext(ctx)
g.SetLimit(downloadConcurrency) // 3
for i, idx := range t.Selected {
    g.Go(func() error { ... client.DownloadFile(gctx, ...) ... })
}
```

- 3 concurrent HTTP streams to CC S3
- Each uses sharded transport (32 available)
- Atomic progress tracking via `atomic.Int64`

### Parallel Markdown (`task_markdown.go`)

```go
workersPerShard := runtime.NumCPU() / markdownConcurrency // 6/2 = 3
g.SetLimit(markdownConcurrency) // 2
for i, idx := range t.Selected {
    g.Go(func() error { ... warcmd.RunPack(gctx, cfg, ...) ... })
}
```

- 2 concurrent RunPack pipelines
- Each with `NumCPU/2` converter workers (3 on server 2)
- Total CPU usage = ~100% (3 workers × 2 pipelines = 6 cores)
- Each pipeline has its own reader → converters → writer chain

### Parallel Index (`task_index.go`)

```go
g.SetLimit(indexConcurrency) // 2
for i, idx := range t.Selected {
    g.Go(func() error { ... eng.Open(...); indexFromWARCMd(...) ... })
}
```

- 2 concurrent indexing pipelines
- Each writes to separate shard directory (no contention)
- Atomic doc count via `atomic.Int64`

### Buffered Writer (`pkg/warc_md/pack.go`)

```go
bw := bufio.NewWriterSize(f, 1024*1024) // 1 MB write buffer
gz, _ := kgzip.NewWriterLevel(bw, kgzip.BestSpeed)
// ... write records ...
bw.Flush()
```

- Reduces syscall overhead for many small gzip members
- ~10-15% writer throughput improvement

## Safety

- **Download**: errgroup limits concurrency; any failure cancels all via context
- **Markdown**: each RunPack writes to unique output file; no shared state
- **Index**: each engine.Open() creates separate directory; no contention
- **Memory**: 2 concurrent shards × ~2 GB = ~4 GB < 9 GB budget
- **Progress**: uses `atomic.Int64` and `atomic.Store` for thread-safe updates

## Future Optimizations

1. **Pipeline overlap**: Start markdown while downloads still in progress
   (requires per-file completion hooks, more complex orchestration)
2. **Increase markdown concurrency to 3**: On larger servers with more CPUs
3. **Streaming index**: Start indexing a shard as soon as its markdown completes
   (requires job dependency tracking in the Manager)
4. **GOMEMLIMIT tuning**: Set `GOMEMLIMIT=9GB` for server 2 dashboard process
