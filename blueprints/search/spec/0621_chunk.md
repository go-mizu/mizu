# spec/0621 — Chunked Crawl + Body CAS Store

## Goal

Reduce peak heap from 7.1 GB → <2 GB during full HN recrawl (1.27M seeds, 414K live
domains). Add content-addressable body storage with DuckDB reference. Compare three
chunking strategies and pick the fastest as default.

---

## Root Cause Analysis (Verified via Code Audit)

| Source | Heap | Evidence |
|--------|------|----------|
| In-flight response bodies | ~2 GB | 8,192 workers × 256 KB body cap; blocked on full segCh when chan=100% |
| DuckDB CGO buffer pools | ~2 GB | 8 shards × `SET memory_limit='256MB'`; invisible to Go GC; never released |
| Seeds slice | ~190 MB | `LoadSeedURLs` reads all 1.27M seeds into `[]SeedURL` before crawl starts |
| Goroutine stacks + GC | ~1 GB | 8,192 goroutines + fragmentation overhead |

**Confirmed via:** `pkg/crawl/writer_bin.go` (chan cap=32768 fills at peak),
`pkg/archived/recrawler/resultdb.go` (memory_limit × 8 shards),
`pkg/crawl/keepalive.go` (LoadSeedURLs bulk load).

**Empty body root cause (separate issue):** `resultdb.go:236` hardcodes `""` for Body
field — intentional fix from prior session preventing DuckDB overflow-string-block data
loss (244 MB pool fills → WAL rollback → 98% data loss). This is replaced by the body
CAS store.

---

## Component 1: Body Content-Addressable Store

**Package:** `pkg/crawl/bodystore/store.go`

### Design

- **Hash algorithm:** SHA-256 (stdlib `crypto/sha256`). Self-describing ref: `"sha256:{hex64}"`.
- **File layout:**
  ```
  {bodyDir}/
    {sha256[0:2]}/
      {sha256[2:4]}/
        {sha256[4:]}.gz        # gzip-compressed raw body bytes
  ```
  Maximum 65,536 second-level dirs (256²). At 10M bodies → ~153 files/dir, well within
  ext4/btrfs inode limits.

- **DuckDB column:** `body_cid TEXT DEFAULT ''`
  - `""` = no body (non-200, non-HTML, body too large, or not stored)
  - `"sha256:3af712..."` = body exists at derived path
  - Identical bodies across different URLs share one file (true CAS deduplication)

### API

```go
// pkg/crawl/bodystore/store.go
type Store struct { dir string }

func Open(dir string) (*Store, error)
func (s *Store) Put(body []byte) (cid string, err error)   // write-once; noop if exists
func (s *Store) Get(cid string) ([]byte, error)            // decompress and return
func (s *Store) Has(cid string) bool
func (s *Store) Path(cid string) string                    // derive fs path from cid
```

### Changes to resultdb.go

- Add `body_cid TEXT DEFAULT ''` column to DDL (`IF NOT EXISTS` migration).
- `writeBatchValues`: write `r.BodyCID` instead of `""`.
- `Result` struct (`pkg/archived/recrawler/types.go`): add `BodyCID string`.

### Changes to BinSegWriter / binRecord

- Remove `Body string` from `binRecord` — reduces gob segment size ~10×.
- Add `BodyCID string` to `binRecord`.
- `writeOne`: set `jr.BodyCID = r.BodyCID`.
- `toResult`: copy `BodyCID` back.

### Changes to keepalive.go

After extracting body bytes:
```go
if s.bodyStore != nil && len(bodyBytes) > 0 {
    cid, _ := s.bodyStore.Put(bodyBytes)
    result.BodyCID = cid
}
```

---

## Component 2: Seed Cursor

**File:** `pkg/crawl/seedcursor.go`

Streaming page cursor replaces bulk `LoadSeedURLs`. Never holds more than one page in
memory (default 10,000 rows).

```go
type SeedCursor struct { ... }
func NewSeedCursor(dbPath string, pageSize int) (*SeedCursor, error)
func (c *SeedCursor) Next(ctx context.Context) ([]recrawler.SeedURL, error)
func (c *SeedCursor) Close() error
```

Uses `SELECT … LIMIT ? OFFSET ?` with a read-only DuckDB connection. Returns empty
slice at EOF.

---

## Component 3: DuckDB Shard Reopen

**File:** `pkg/archived/recrawler/resultdb.go`

```go
func (rdb *ResultDB) ReopenShards() error
```

For each shard:
1. Flush pending batch
2. `s.db.Close()` — releases CGO buffer pool immediately
3. `s.db, err = sql.Open("duckdb", path)` — fresh connection
4. Re-apply `SET memory_limit`, `SET threads`, `SET checkpoint_threshold`

Called by Mode B between domain batches to reclaim 2 GB of CGO memory.

---

## Component 4: Three Chunk Modes

### Mode A — `stream` (adaptive backpressure)

Replace `seeds []SeedURL` parameter with `SeedCursor`. Auto-size workers and channel:

```go
func AutoBinChanCap(availMB, bodyKB int) int {
    // max 5% of available RAM for channel buffer
    cap := availMB * 1024 * 1024 / 20 / (bodyKB * 1024)
    return clamp(cap, 256, 32768)
}
func AutoWorkersFull(availMB, bodyKB int) int {
    // max 20% of available RAM for in-flight bodies
    return clamp(availMB*1024/5/bodyKB, 100, 8192)
}
```

DuckDB `memory_limit` per shard: `availMB * 0.15 / shards` (was fixed 256 MB).

**Memory model:** `workers × bodyKB × 2 + chanCap × bodyKB + DuckDB` ≤ `GOMEMLIMIT × 0.85`.

**Impact:** Low — touches `autoconfig.go`, `keepalive.go`, `cli/hn.go`.

### Mode B — `batch` (domain-batch loop) ← **default**

Outer loop splits live domains into chunks:

```
for batch in chunk(liveDomains, batchDomains):
    eng.Run(ctx, batch)
    waitDrainComplete()
    rdb.ReopenShards()          // release 2 GB CGO memory
    debug.FreeOSMemory()        // return pages to OS
    updateProgress()
```

Auto-tuned batch size:

```go
func AutoBatchDomains(availMB, avgURLsPerDomain, bodyKB int) int {
    budget := availMB * 1024 / 3  // KB; 30% of RAM
    urls := budget / bodyKB
    return max(urls / avgURLsPerDomain, 500)
}
```

At 10.5 GB available, bodyKB=256, avgURLs=3 → ~4,300 domains/batch.

**Impact:** Medium — `ReopenShards()` on ResultDB; batch-split loop in `cli/hn.go`.

### Mode C — `pipeline` (staged goroutine pipeline)

```
SeedCursor → [seedPageCh cap=2] → DomainBatcher → [batchCh cap=1] →
CrawlStage (keepalive) → [segPathCh cap=4] → DrainStage → DuckDB + BodyStore
```

Each stage bounded; upstream blocks when downstream full. Drain overlaps with next
batch's crawl for higher throughput.

- `seedPageCh cap=2`: 2 pages × 10K × 150 B = 3 MB
- `batchCh cap=1`: one domain batch in flight
- `segPathCh cap=4`: 4 segments × 64 MB = 256 MB
- CrawlStage: `AutoWorkersFull(availMB × 0.40, bodyKB)` (40% RAM)

**Impact:** High — new `PipelineCrawler` type in `pkg/crawl/pipeline.go`.

---

## Component 5: pprof Integration

**File:** `cli/hn.go`

```go
if pprofPort > 0 {
    go func() { _ = http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil) }()
}
```

Auto-snapshot at peak: when `HeapAlloc > GOMEMLIMIT × 0.80`, write `heap.pprof` once.

CLI flag: `--pprof-port int` (default 0 = disabled).

---

## Component 6: Benchmark Comparison

**File:** `pkg/crawl/chunkbench.go`

Each mode writes `{dataDir}/bench_chunk_{mode}.json`:
```json
{
  "mode": "batch",
  "avg_rps": 2547,
  "peak_rps": 9363,
  "peak_heap_mb": 1842,
  "gc_cycles": 48,
  "duration_s": 50,
  "ok_count": 35031,
  "body_store_writes": 4821,
  "batch_count": 92
}
```

Makefile target:
```makefile
bench-chunk: ## Run all 3 chunk modes and compare (200K seeds, no-retry)
    @for mode in stream batch pipeline; do \
        rm -f data/hn/recrawl/results/*.duckdb; \
        $(BINARY) hn recrawl --limit 200000 --no-retry --chunk-mode $$mode; \
    done
    @$(BINARY) hn bench-chunk-compare
```

---

## Schema Change

```sql
ALTER TABLE results ADD COLUMN body_cid TEXT DEFAULT '';
```

Applied automatically in `ResultDB.Open()` via `IF NOT EXISTS` column add pattern.

---

## File Change Summary

| File | Change |
|------|--------|
| `pkg/crawl/bodystore/store.go` | **New:** CAS body store |
| `pkg/crawl/seedcursor.go` | **New:** streaming seed cursor |
| `pkg/crawl/pipeline.go` | **New:** Mode C pipeline |
| `pkg/crawl/chunkbench.go` | **New:** benchmark JSON writer |
| `pkg/crawl/autoconfig.go` | Add `AutoBinChanCap`, `AutoWorkersFull`, `AutoBatchDomains` |
| `pkg/crawl/keepalive.go` | Accept `SeedCursor`; pass `bodyStore`; call `Put()` |
| `pkg/crawl/writer_bin.go` | Remove `Body` from binRecord; add `BodyCID` |
| `pkg/archived/recrawler/resultdb.go` | Add `body_cid` column; `ReopenShards()`; store `BodyCID` |
| `pkg/archived/recrawler/types.go` | Add `BodyCID string` to `Result` |
| `cli/hn.go` | `--chunk-mode`, `--chunk-size`, `--pprof-port`; batch loop; pipeline wiring |
| `Makefile` | `bench-chunk` target |

---

## Expected Outcomes

| Metric | Before | After (Mode B) |
|--------|--------|----------------|
| Peak heap | 7.1 GB (90% lim) | <2 GB (<30% lim) |
| DuckDB CGO pool | 2 GB (never freed) | Released per batch |
| Seeds in memory | 190 MB (full slice) | 1.5 MB (1 page × 10K) |
| Body in DuckDB | "" (empty, overflow fix) | sha256 CID reference |
| Body on disk | Inline in .bseg | bodies/{sha}/{sha}.gz |

---

## Default After Benchmarking

```go
const defaultChunkMode = "batch"  // updated after bench-chunk comparison
```

Expected ranking: B ≥ C > A (batch has simplest memory model; pipeline may win on
throughput via overlapping drain/crawl phases).

---

## Status

- [ ] Implement `pkg/crawl/bodystore/store.go`
- [ ] Implement `pkg/crawl/seedcursor.go`
- [ ] Add `BodyCID` to `pkg/archived/recrawler/types.go`
- [ ] Update `pkg/archived/recrawler/resultdb.go` (body_cid column + ReopenShards)
- [ ] Update `pkg/crawl/writer_bin.go` (remove Body, add BodyCID)
- [ ] Update `pkg/crawl/keepalive.go` (bodyStore integration)
- [ ] Update `pkg/crawl/autoconfig.go` (AutoBinChanCap, AutoWorkersFull, AutoBatchDomains)
- [ ] Update `cli/hn.go` (--chunk-mode, --pprof-port, batch loop)
- [ ] Implement `pkg/crawl/pipeline.go` (Mode C)
- [ ] Implement `pkg/crawl/chunkbench.go` + Makefile target
- [ ] Run `make bench-chunk` and pick default
