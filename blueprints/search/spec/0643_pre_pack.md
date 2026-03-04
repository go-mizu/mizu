# spec/0643 — Pre-Pack Formats for FTS Import

**Status:** implemented
**Branch:** index-pane
**Commits:** `ad6e275c` (initial pack), `86c81f11` (footer + buffer pool), `a1220c6d` (`Document.Text []byte`), `8ea2c208` (DuckDB insert modes + server build), `9a6e5095` (parquet BulkLoad + writer optimisation)

---

## Motivation

Building a full-text search index over 173,720 markdown documents requires
reading every file from disk and decompressing `.md.gz` archives.  At 20 k
docs/s this takes ~8.7 s just to ingest the raw corpus.  Pre-packing the
corpus into a single binary file amortises that startup cost: once packed,
subsequent index-build runs (different FTS engines, benchmark iterations,
CI) skip file walking and decompression entirely.

Four formats were evaluated, implemented, and benchmarked against each other
and against the raw `.md.gz` baseline.

---

## Dataset

| property | value |
|----------|-------|
| crawl | CC-MAIN-2026-08 |
| documents | 173,720 |
| source files | 173,720 × `.md.gz` files |
| average text size | ~4 KB / doc |
| total uncompressed text | ~700 MB |

---

## Pack Formats

### 1. Flat Binary (`docs.bin`)

**File:** `{crawl}/fts/pack/docs.bin`
**Go:** `pkg/index/pack_flatbin.go`
**Disk:** 709.5 MB

#### Header (8 bytes)

```
"MZFTS1\n\x00"
```

#### Records (repeated, immediately after header)

```
id_len   uint16 LE   — doc ID byte length  (max 65535, truncated if longer)
id       [id_len]byte
txt_len  uint32 LE   — text byte length
text     [txt_len]byte
```

Doc IDs are the filename stem (UUID, ~36 bytes).  Text is the raw markdown
content in UTF-8.

#### Footer (30 bytes, appended after last record)

```
offset  size  type      description
------  ----  --------  ------------------------------------------
 0      8     int64 LE  record_count — number of records
 8      8     int64 LE  index_offset — byte offset to index table (0 = none)
16      4     int32 LE  index_size   — byte size of index table  (0 = none)
20      1     uint8     version      — format version, currently 1
21      1     uint8     flags        — reserved, 0
22      8     [8]byte   footer_magic — "MZFTS1F\n"
```

The reader uses `pread(2)` (`f.ReadAt`) to load the footer before touching
the sequential byte stream, so no seek is needed and the buffered reader
starts cleanly at record offset 8.  When `record_count` is known the reader
goroutine stops at exactly that boundary and never attempts to parse the
footer bytes as a record header.  Files written before the footer was added
have no footer magic; the reader falls back gracefully (`total = 0`, reads
until EOF).

#### Writer implementation

```
bufio.NewWriterSize(f, 1 MB)
→ write header
→ for each doc:  bw.Write(hdr[:2])  io.WriteString(bw, id)
                 bw.Write(hdr[2:6]) bw.Write(doc.Text)   ← []byte, no copy
→ write footer (30 bytes, stats.DocsIndexed)
→ bw.Flush()  f.Close()
```

#### Reader implementation

```
f.ReadAt(footer, size-30)         ← pread, no seek
io.ReadFull(f, magic[8])          ← validate header
bufio.NewReaderSize(f, 1 MB)
loop until count == total:
    io.ReadFull(br, hdr[2])       ← id_len
    idBuf  := make([]byte, idLen) ← small, UUID-sized
    io.ReadFull(br, idBuf)
    io.ReadFull(br, hdr[4])       ← txt_len
    textBuf := make([]byte, textLen)  ← fresh alloc, passed as Document.Text
    io.ReadFull(br, textBuf)
    docCh ← Document{string(idBuf), textBuf}  ← zero-copy for Text
```

---

### 2. NDJSON (`docs.ndjson`)

**File:** `{crawl}/fts/pack/docs.ndjson`
**Go:** `pkg/index/pack_ndjson.go`
**Disk:** 721.8 MB

One JSON object per line, no HTML escaping:

```json
{"i":"<doc_id>","t":"<text>"}\n
```

Writer uses `json.Encoder` with `SetEscapeHTML(false)` and a 1 MB
`bufio.Writer`.  Reader uses `bufio.ReadBytes('\n')` + `json.Unmarshal`; no
total row count available so progress shows docs/s but no percentage.

---

### 3. Parquet (`docs.parquet`)

**File:** `{crawl}/fts/pack/docs.parquet`
**Go:** `pkg/index/pack_parquet.go`
**Disk:** 746 MB

Schema:

```go
type packParquetDoc struct {
    DocID string `parquet:"doc_id"`
    Text  string `parquet:"text"`
}
```

Written with `parquet-go` `GenericWriter`.  Reader uses `pf.NumRows()` from
the Parquet footer metadata to obtain the total row count upfront, enabling
percentage progress display (same as flatbin after the footer was added).

---

### 4. DuckDB Raw (`docs.raw.duckdb`)

**File:** `{crawl}/fts/pack/docs.raw.duckdb`
**Go:** `pkg/index/driver/duckdb/pack_raw.go`
**Disk:** 1.1 GB

A plain DuckDB database with a single unindexed table:

```sql
CREATE TABLE docs (doc_id VARCHAR, text VARCHAR)
```

Placed in the `duckdb` driver package to keep `pkg/index` CGO-free.  Writer
inserts rows inside explicit transactions; reader counts rows with
`SELECT count(*)` before streaming to provide total for progress.

---

## Pack Creation Benchmarks

Source: 173,720 `.md.gz` files.  Single `--format all` run.

### macOS ARM64 (Apple M4, NVMe SSD)

| format  | write speed     | disk size | notes                              |
|---------|-----------------|-----------|-------------------------------------|
| ndjson  | 20,264 docs/s   | 721.8 MB  | fastest write; bottleneck = JSON enc |
| bin     | 20,123 docs/s   | **709.5 MB** | smallest; pure byte write          |
| parquet | 19,516 docs/s   | 746 MB    | columnar compression, slightly slower |
| duckdb  |  6,777 docs/s   | 1.1 GB    | **3× slower write**; WAL overhead   |

### Ubuntu 24.04 Noble (server2, x86_64 6-core, HDD)

| format  | write speed   | disk size | notes                              |
|---------|---------------|-----------|-------------------------------------|
| parquet |  9,966 docs/s | 744.9 MB  | fastest write on HDD                |
| ndjson  |  4,492 docs/s | 721.8 MB  | JSON enc overhead visible on HDD     |
| bin     |  2,463 docs/s | 710.9 MB  | **smallest**; HDD random-write bound |
| duckdb  |    570 docs/s | 1.1 GB    | WAL overhead                        |

All three text formats write at roughly the same speed (~20 k docs/s) on
NVMe; on HDD, sequential NDJSON/parquet outpace the binary format due to
alignment differences in the write pattern.  DuckDB is an outlier on both:
each batch triggers a WAL append and occasional checkpoint.

---

## Read Throughput — Devnull Engine

`devnull` discards all documents immediately; this isolates deserialisation
speed from FTS index overhead.

### macOS ARM64 — after `Document.Text []byte` optimisation

| source   | docs/s    | elapsed | peak RSS | speedup vs files |
|----------|-----------|---------|----------|-----------------|
| files    |  19,500   | 8.7 s   | 141 MB   | 1×              |
| bin      | **715,000** | 200 ms  | <1 MB †  | **37×**         |
| parquet  | 401,005   | 400 ms  | <1 MB †  | 21×             |
| duckdb   | 195,841   | 800 ms  | 141 MB   | 10×             |
| ndjson   |  48,485   | 3.6 s   | 137 MB   | 2.5×            |

### Ubuntu 24.04 Noble (server2, x86_64 6-core, HDD)

| source   | docs/s     | elapsed | speedup vs files |
|----------|------------|---------|-----------------|
| files    |  25,151    | 6.9 s   | 1×              |
| bin      | **86,736** | 2.0 s   | **3.5×**        |
| parquet  |  24,424    | 7.1 s   | ~1×             |
| duckdb   |  13,381    | 13.0 s  | 0.5×            |
| ndjson   |  14,226    | 12.2 s  | 0.6×            |

On HDD the bin parallel-reader speedup collapses from 37× to 3.5× — seek
latency dominates when multiple goroutines jump to random chunk offsets.

† Runs complete in under 500 ms; the 500 ms RSS sampling interval never
fires.  Effective working set ≈ `2 × batchSize × avgDocSize`
= 2 × 5,000 × 4 KB ≈ **40 MB**.

**Why files = 141 MB?**  Each worker holds a partially-decoded gzip stream
and the current batch.  With `NumCPU` parallel readers and batch size 5,000
the in-flight corpus is large enough to accumulate several hundred MB of
RSS before the GC reclaims it.

**Why ndjson = 137 MB?**  `json.Unmarshal` allocates intermediate string
values for every field; the decoder keeps a decode buffer per goroutine.

**Why bin / parquet = ~0 MB?**  Sequential binary reads with no intermediate
allocations beyond the 1 MB bufio buffer; GC has no opportunity to
accumulate live heap before the run finishes.

---

## `Document.Text []byte` — Before vs After

`Document.Text` was changed from `string` to `[]byte` in commit `a1220c6d`.

### What changed in the hot path

| stage | before (`string`) | after (`[]byte`) |
|-------|-------------------|-----------------|
| flatbin reader | `make([]byte, n)` + `string(buf)` copy (via sync.Pool) | `make([]byte, n)` only — passed directly |
| pipeline reader | `string(data)` copy after ReadAll | `data` passed directly; `string()` only on invalid UTF-8 |
| duckdb/sqlite engine | `doc.Text` as string arg | `string(doc.Text)` at SQL binding site |
| parquet writer | `d.Text` (string field) | `string(d.Text)` |
| ndjson writer | `doc.Text` (string field) | `string(doc.Text)` |

For real FTS engines the copy merely moves from reader to writer — net zero.
For `devnull` (pure read benchmark) the copy is eliminated entirely.

### Benchmark results

| source  | before (pool + string) | after ([]byte direct) | delta   |
|---------|------------------------|-----------------------|---------|
| bin     | 619,540 docs/s         | 715,000 docs/s        | **+15%** |
| duckdb  | 178,272 docs/s         | 195,841 docs/s        | **+10%** |
| parquet | 468,075 docs/s         | 401,005 docs/s        | ~same ¹ |
| ndjson  |  48,680 docs/s         |  48,485 docs/s        | ~same   |
| files   |  20,146 docs/s         |  19,500 docs/s        | ~same   |

¹ Parquet reader still pays `[]byte(batch[i].Text)` (schema field is
`string`); no benefit, and the GC now has more `[]byte` objects to scan.
Difference is within measurement noise.

**Root cause of flatbin +15%:** The previous approach used a `sync.Pool` to
avoid allocation but still called `string(buf)` which is an unconditional
`memcpy` of the entire text.  With `[]byte`, the `make([]byte, textLen)` call
gets a pre-zeroed OS page (virtual memory CoW) — the only write is
`io.ReadFull` itself.  Eliminating the `memcpy` saves one full pass over the
~4 KB average document body per record.

---

## Full Import Matrix — FTS Engines

173,720 documents; 4 FTS engines × 5 sources.

### macOS ARM64 (Apple M4, NVMe SSD)

#### Ingest speed (docs/s)

| source \ engine | devnull      | sqlite FTS5 | duckdb BM25 | chdb FTS    |
|-----------------|--------------|-------------|-------------|-------------|
| files           | 19,500       | 2,078       | 1,351       | —           |
| bin             | **812,000** ¹ | **3,254**  | 1,386       | **3,992**   |
| parquet         | 406,460      | 2,336       | ~1,050      | —           |
| ndjson          | 48,485       | 2,368       | ~1,100      | —           |
| duckdb raw      | 195,841      | 3,205       | **1,451**   | —           |

¹ Parallel reader (NumCPU workers, record offset index in footer).

#### Elapsed time

| source \ engine | devnull    | sqlite FTS5  | duckdb BM25   | chdb FTS    |
|-----------------|------------|--------------|---------------|-------------|
| files           | 8.7 s      | 1m 23.6 s    | 2m 8.6 s      | —           |
| bin             | **200 ms** | **53.4 s**   | 2m 5.3 s      | **43.5 s**  |
| parquet         | 400 ms     | 1m 14.4 s    | ~2m 45 s      | —           |
| ndjson          | 3.6 s      | 1m 13.4 s    | ~2m 36 s      | —           |
| duckdb raw      | 800 ms     | 54.2 s       | **1m 59.7 s** | —           |

#### Final index disk size (macOS)

| engine      | disk size |
|-------------|-----------|
| sqlite FTS5 | 1.1 GB    |
| duckdb BM25 | 1.5 GB    |
| chdb FTS    | 821 MB    |

---

### Ubuntu 24.04 Noble (server2, x86_64 6-core, HDD)

#### Ingest speed (docs/s)

Values for duckdb use naive insert mode; see §DuckDB Insert Mode Comparison for 7.5× improvement.

| source \ engine | devnull    | sqlite FTS5 | duckdb BM25 ³ | chdb FTS    |
|-----------------|------------|-------------|---------------|-------------|
| files           | 25,151     | 539         | 147           | **789**     |
| bin             | **86,736** | 580         | 157           | 751         |
| parquet         | 24,424     | **684**     | 148           | 767         |
| ndjson          | 14,226     | 664         | 148           | 739         |
| duckdb raw      | 13,381     | 675         | 150           | n/a ²       |

² chdb binary excludes the DuckDB driver (`//go:build !chdb`).
³ naive insert mode; duckdb-appender reaches 1,015 docs/s, parquet BulkLoad **1,129 docs/s**.

#### Elapsed time

| source \ engine | devnull    | sqlite FTS5   | duckdb BM25 ³  | chdb FTS      |
|-----------------|------------|---------------|----------------|---------------|
| files           | 6.9 s      | 5m 22.1 s     | 19m 43.8 s     | **3m 40.1 s** |
| bin             | **2.0 s**  | 4m 59.7 s     | 18m 27.6 s     | 3m 51.4 s     |
| parquet         | 7.1 s      | **4m 14.1 s** | 19m 31.9 s     | 3m 46.6 s     |
| ndjson          | 12.2 s     | 4m 21.5 s     | 19m 35.7 s     | 3m 55.0 s     |
| duckdb raw      | 13.0 s     | 4m 17.2 s     | **19m 16.0 s** | n/a           |

#### Final index disk size

| engine      | disk size |
|-------------|-----------|
| sqlite FTS5 | 1.1 GB    |
| duckdb BM25 | 1.5 GB    |
| chdb FTS    | ~865 MB   |

---

## DuckDB Insert Mode Comparison (server2)

DuckDB's default row-insertion path (one `INSERT OR IGNORE` per document
in a batch transaction) is dominated by WAL fsync latency on HDD.  Four
insert strategies and a native parquet bulk-load path were benchmarked.

### Results (173,720 docs, server2 HDD)

| engine           | source  | insert    | fts build | total      | docs/s    | speedup |
|-----------------|---------|-----------|-----------|------------|-----------|---------|
| duckdb (naive)   | bin     | 17m 8.9s  | 2m 12.8s  | 19m 21.9s  | 150       | 1×      |
| duckdb-prepared  | bin     | ~12m      | ~2.5m     | 14m 39.6s  | 198       | 1.3×    |
| duckdb-multirow  | bin     | 35.5s     | 2m 31.1s  | 3m 7.2s    | 928       | 6.2×    |
| duckdb-appender  | bin     | 3.1s      | 2m 37.1s  | 2m 51.2s   | 1,015     | 6.8×    |
| **duckdb (bulk)**| parquet | **6.9s**  | 2m 26.8s  | **2m 33.9s** | **1,129** | **7.5×** |

### Insert strategies

| mode      | mechanism                                           | WAL overhead       |
|-----------|-----------------------------------------------------|--------------------|
| naive     | `INSERT OR IGNORE` × 1 per doc in transaction       | per-tx fsync       |
| prepared  | prepared statement reused per batch                 | per-tx fsync       |
| multirow  | `INSERT … VALUES (?,?),…` per batch                 | per-batch fsync    |
| appender  | DuckDB Appender API (no SQL parsing)                | buffered until `Close()` |
| bulk      | `CREATE TABLE AS SELECT … FROM read_parquet(path)` | no row path at all |

### Key observations

- **FTS build dominates**: `PRAGMA create_fts_index` takes ~2.5 min regardless
  of insert strategy.  All optimisation gains are in reducing insert time.
- **Prepared mode**: minimal benefit — both naive and prepared pay a per-tx WAL
  fsync; prepared only eliminates SQL string building (~1.3× speedup).
- **Multirow**: single `INSERT … VALUES (N rows)` per batch eliminates
  per-row round-trips; 29× faster inserts vs naive, 6.2× end-to-end.
- **Appender**: bypasses SQL parser; buffers in memory; no WAL fsync until
  `Close()`.  Fastest streaming insert (5.5× faster than multirow inserts).
- **Parquet BulkLoad** (`read_parquet()`): `CREATE TABLE AS SELECT` skips the
  PRIMARY KEY uniqueness check, uses DuckDB's vectorised columnar reader, and
  processes all rows in one pass.  **Fastest overall** — insert 148× faster
  vs naive; 7.5× end-to-end improvement.

---

## Analysis

### Pack format is irrelevant for real FTS engines

For `sqlite` and `duckdb`, the bottleneck is index creation (FTS5 trigger
overhead / BM25 posting list construction), not deserialisation.  Switching
from raw files to any pack format saves at most 8 s out of 53–128 s total
on macOS NVMe — under 15% of overall build time.

On HDD (server2) the effect is similar: deserialization is ~10 s faster
via packed sources, but FTS indexing dominates at 4–20 minutes.

Source ordering on server2 (naive DuckDB):
- `parquet` marginally fastest for sqlite (684 docs/s, 4m14s).
- `bin` fastest for duckdb naive (157 docs/s, 18m28s); with appender/parquet
  BulkLoad, duckdb reaches **1,015–1,129 docs/s** (7.5× improvement).
- For chdb, `files` source wins slightly (789 docs/s, 3m40s) — gzip
  decompression is compute-bound and server2 has spare CPU while ClickHouse
  inserts serialize through a single session.

### DuckDB insert mode is the dominant performance variable

The tables above (§Full Import Matrix) show duckdb with naive inserts.
With the appender mode or parquet BulkLoad path, duckdb reaches parity
with chdb (~1,015–1,129 vs ~751–789 docs/s on server2) while maintaining
full BM25 search quality.  The FTS build (~2.5 min) is now the hard floor
regardless of insert method — further improvement requires a faster FTS
index builder.

### chdb is the fastest FTS engine for bulk import (naive inserts baseline)

chdb (ClickHouse embedded via chdb-go) indexes at **3,992 docs/s** from
`bin` source on macOS — 23% faster than sqlite FTS5 (3,254 docs/s) and
2.9× faster than duckdb BM25 naive (1,386 docs/s).

On server2 with naive inserts, chdb remains faster than duckdb (~750–790
vs ~147–157 docs/s).  With duckdb-appender or parquet BulkLoad, duckdb
**surpasses chdb** on server2 (1,015–1,129 vs 751–789 docs/s).  chdb also
produces the smallest index (~865 MB vs 1.1 GB sqlite, 1.5 GB duckdb).

chdb is 5–10× slower on server2 HDD vs macOS NVMe for the same corpus
(750 vs 3992 docs/s from bin), whereas sqlite degrades ~4–5× (580 vs 3254).
DuckDB naive degrades the most (~9×: 157 vs 1451 docs/s), suggesting BM25
posting list construction is particularly IO-sensitive.  With appender/bulk,
DuckDB HDD degradation is reduced to ~4× (1015 vs ~4000 docs/s estimated
on NVMe) since the bottleneck shifts to the FTS build, not WAL fsyncs.

The bulk-insert path uses `INSERT INTO documents FORMAT JSONEachRow` with
`json.Encoder` generating NDJSON embedded directly in the query string.
This avoids the ClickHouse SQL parser traversing a multi-megabyte VALUES
list: `json.Encoder` encodes newlines as `\n`, backslashes as `\\`, and
quotes as `\"`, keeping each line in the query body well-formed regardless
of document content.

Alternative approaches investigated:
- **VALUES clause** (original): hangs at batch=5000 because literal newlines
  in markdown text break the ClickHouse SQL parser.  Fixed by capping to
  batch=500 + escaping, but slower (3,203 docs/s).
- **INSERT FORMAT CSV**: ClickHouse's in-process C library does not support
  embedded multi-line quoted CSV fields via the `chdb_query` API — the
  parser fails on non-ASCII characters in document bodies.
- **JSONEachRow**: works at any batch size; `json.Encoder` handles all
  escaping; no artificial cap needed.

### Pack format matters enormously for devnull / pre-validation

When the goal is to verify document counts or pipeline correctness without
building an index, `bin` is 42× faster than raw files (200 ms vs 8.7 s)
with the parallel reader (NumCPU workers each reading their own chunk).

### Best pack format: flatbin

- **Smallest disk footprint** (709.5 MB — 5% smaller than NDJSON, 5% smaller
  than parquet).
- **Fastest read** (812 k docs/s with parallel index reader — 2× faster than
  parquet, 16.8× faster than NDJSON).
- **Simplest format** — 8-byte magic + length-prefixed records + 30-byte
  footer.  No schema evolution overhead, no parser dependencies, no
  compression codec.
- **Percentage progress** — footer provides exact row count upfront via
  `pread(2)`, enabling `%` display without a prior scan pass.
- **Parallel reader** — record offset index (N×uint64 LE) written after the
  last record; reader spawns `runtime.NumCPU()` goroutines each seeking to
  their chunk start using the index.
- **Backward compatible footer** — old files without footer magic fall back
  to `total=0` / read-until-EOF transparently.

### When to choose other formats

| format | use case |
|--------|----------|
| parquet | Interop with external tooling (DuckDB `read_parquet`, pandas, etc.) |
| ndjson | Human-readable inspection with `jq`; streaming from HTTP |
| duckdb raw | Existing DuckDB analytics workflow; SQL ad-hoc queries on the corpus |

---

## File Locations

```
{crawl}/fts/pack/
├── docs.bin          flatbin  (709.5 MB)
├── docs.ndjson       NDJSON   (721.8 MB)
├── docs.parquet      Parquet  (746 MB)
└── docs.raw.duckdb   DuckDB   (1.1 GB)
```

Default crawl base: `$HOME/data/common-crawl/`

---

## CLI

```sh
# Pack all formats from markdown files
search cc fts pack --crawl CC-MAIN-2026-08 --format all

# Pack a single format
search cc fts pack --crawl CC-MAIN-2026-08 --format bin

# Index from a specific pack source
search cc fts index --crawl CC-MAIN-2026-08 --engine sqlite --source bin
search cc fts index --crawl CC-MAIN-2026-08 --engine duckdb --source duckdb

# chdb requires build tag (separate binary)
make build-chdb
search cc fts index --crawl CC-MAIN-2026-08 --engine chdb --source bin

# Benchmark read speed only (devnull discards all docs)
search cc fts index --crawl CC-MAIN-2026-08 --engine devnull --source bin
```

---

## Implementation Notes

- `pkg/index/pack.go` — `PackProgressFunc`, `RunPipelineFromChannel`,
  `funcEngine` (unexported Engine wrapper backed by a single `indexFn` closure)
- `pkg/index/pack_flatbin.go` — `PackFlatBin` / `RunPipelineFromFlatBin`
- `pkg/index/pack_ndjson.go` — `PackNDJSON` / `RunPipelineFromNDJSON`
- `pkg/index/pack_parquet.go` — `PackParquet` / `RunPipelineFromParquet`
- `pkg/index/driver/duckdb/pack_raw.go` — `PackDuckDBRaw` / `RunPipelineFromDuckDBRaw`
  (in duckdb driver package to keep `pkg/index` CGO-free)
- `pkg/index/driver/chdb/chdb.go` — chdb ClickHouse engine (build tag `chdb`)
  - `INSERT … FORMAT JSONEachRow` with `json.Encoder` for safe bulk insert
  - Requires `libchdb.so` (ARM64 Mac: install name patched + re-signed)
  - Build: `make build-chdb` (sets `CGO_LDFLAGS`, `CGO_CFLAGS`, `-tags chdb`)
  - **Cannot co-load with duckdb driver** — SIGABRT on `duckdb_get_or_create_from_cache`

## chdb Setup (macOS ARM64)

```sh
# Download libchdb ARM64
curl -L https://github.com/chdb-io/chdb/releases/download/v4.0.2/macos-arm64-libchdb.tar.gz \
  | tar xz -C /tmp/libchdb/

# Install library + fix install name + re-sign (SIP strips DYLD_LIBRARY_PATH)
cp /tmp/libchdb/libchdb.so ~/lib/
install_name_tool -id "$HOME/lib/libchdb.so" ~/lib/libchdb.so
codesign --force --sign - ~/lib/libchdb.so

# Build
make build-chdb
```
