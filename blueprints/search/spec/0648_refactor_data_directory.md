# spec/0648 — Refactor Data Directory Layout

**Status:** done
**Branch:** index-pane

---

## Motivation

The current layout stores all markdown and FTS artefacts in flat, crawl-wide directories.
This makes it impossible to:
- process one WARC file independently and incrementally add it to an index,
- keep pack files and FTS indices co-located with the data they were built from,
- parallelize pack/index across WARC files.

This spec introduces a **per-WARC-index** directory layout for every artefact
downstream of the raw `.warc.gz` download.

---

## New Directory Layout

```
$HOME/data/common-crawl/CC-MAIN-2026-08/
│
├── warc/                                          # unchanged
│   ├── CC-MAIN-20260206181458-20260206211458-00000.warc.gz
│   └── CC-MAIN-20260206181458-20260206211458-00001.warc.gz
│
├── markdown/                                      # per-WARC, uncompressed plain .md
│   ├── 00000/
│   │   └── 5d/0e/22/{uuid}.md                    # sharding: first 6 hex chars of UUID
│   └── 00001/
│       └── ...
│
├── pack/                                          # per-WARC packed bundles
│   ├── bin/
│   │   ├── 00000.bin                              # flatbin with footer+index
│   │   └── 00001.bin
│   ├── parquet/
│   │   └── 00000.parquet
│   ├── duckdb/
│   │   └── 00000.duckdb
│   └── markdown/
│       ├── 00000.bin.gz                           # concat gzip members of flatbin records
│       └── 00000.bin.gz.idx                       # member offset index (lazy-built)
│
└── fts/                                           # per-WARC FTS indices
    ├── rose/
    │   ├── 00000/  (*.seg, rose.docs)
    │   └── 00001/
    ├── bleve/
    │   └── 00000/
    ├── tantivy/
    │   └── 00000/
    ├── duckdb/
    │   └── 00000.duckdb
    └── sqlite/
        └── 00000.sqlite
```

**WARC index** is the zero-padded 5-digit suffix extracted from the filename:
`CC-MAIN-20260206181458-20260206211458-00000.warc.gz` → `"00000"`.
Falls back to `fmt.Sprintf("%05d", manifestPosition)` if not parseable.

---

## pack/markdown/00000.bin.gz Format

Concatenated gzip members. Each member is independently decompressible.
Standard tools (`zcat`, `gunzip`) work transparently.

```
[gzip member 0]   BestCompression → flatbin records 0..999
[gzip member 1]   BestCompression → flatbin records 1000..1999
...
[gzip member K]   last chunk (≤ 1000 records)
```

**Inside each gzip member** (raw flatbin, no per-member header/footer):
```
uint16 id_len  (LE)  — doc ID byte length
[]byte id
uint32 txt_len (LE)  — markdown text byte length
[]byte text
(repeated for all records in the member)
```

**Index file** `00000.bin.gz.idx`:
```
per member (12 bytes × M):
  uint64 LE  byte_offset   — start of this gzip member in the .bin.gz file
  uint32 LE  doc_count     — number of docs in this member
footer (4 bytes):
  uint32 LE  member_count
```

Written alongside the `.bin.gz` during `fts pack --format markdown`.
If the `.idx` is missing, it is rebuilt by scanning for gzip magic `0x1f 0x8b`.

**Parallel read**: N workers each open their own file handle, seek to assigned
member offsets, create a `gzip.NewReader` per member (reads exactly one member),
parse raw flatbin records until member EOF.

---

## Command Changes

### `cc warc markdown`

| Before | After |
|--------|-------|
| 3 phases: extract → convert → compress | 2 phases: extract → convert |
| Output: `markdown/{uuid}.md.gz` (compressed) | Output: `markdown/{warcIdx}/{sharded}/{uuid}.md` (plain) |
| `--mem` flag (streaming in-memory) | **removed** |
| All WARC files share one `markdown/` dir | Each WARC gets `markdown/{warcIdx}/` |

### `cc fts pack`

| Before | After |
|--------|-------|
| `fts/pack/docs.bin` | `pack/bin/{warcIdx}.bin` |
| `fts/pack/docs.parquet` | `pack/parquet/{warcIdx}.parquet` |
| `fts/pack/docs.raw.duckdb` | `pack/duckdb/{warcIdx}.duckdb` |
| `fts/pack/docs.ndjson` (removed) | — |
| — | `pack/markdown/{warcIdx}.bin.gz` (new) |
| Reads from global `markdown/` | Reads from `markdown/{warcIdx}/` |

Formats: `bin`, `parquet`, `duckdb`, `markdown`, `all`.

### `cc fts index`

| Before | After |
|--------|-------|
| Opens `fts/{engine}/` | Opens `fts/{engine}/{warcIdx}/` |
| Source `files`: reads global `markdown/` | Source `files`: reads `markdown/{warcIdx}/` |
| Source `bin`: reads `fts/pack/docs.bin` | Source `bin`: reads `pack/bin/{warcIdx}.bin` |
| Sources `ndjson` removed | — |
| — | Source `markdown`: reads `pack/markdown/{warcIdx}.bin.gz` |
| Requires `--crawl` | Also requires `--file N` |

### `cc fts search`

| Before | After |
|--------|-------|
| Opens `fts/{engine}/` | Default: fan-out across all `fts/{engine}/{warcIdx}*/` |
| — | `--file N`: search only `fts/{engine}/{warcIdx}/` |
| Single result set | Parallel search + top-K merge |

### `cc fts decompress`

Removed. Markdown is always written as uncompressed `.md` now.

---

## Code Changes Summary

| File | Change |
|------|--------|
| `pkg/cc/config.go` | Add `MarkdownWarcDir`, `PackFile`, `FTSEngineDir`, `FTSEngineFile` |
| `pkg/warc_md/config.go` | Add `MarkdownWarcDir(idx)`; remove `MarkdownGzDir`, `CompressWorkers` |
| `pkg/warc_md/pipeline.go` | Remove phase 3; `RunFilePipeline` writes to `MarkdownWarcDir`; remove `RunInMemoryPipeline` |
| `pkg/warc_md/compress.go` | Deleted |
| `pkg/index/pack_bingz.go` | New: `PackFlatBinGz`, `RunPipelineFromFlatBinGz`, `BuildBinGzIndex` |
| `cli/cc_warc_markdown.go` | Extract warcIdx; 2-phase only; remove `--mem`; remove compress phase |
| `cli/cc_fts.go` | New paths; add `markdown` source/format; fan-out search; remove decompress |

---

## Benchmark Results

_Measured on server2 (Ubuntu 24.04 Noble, 12 GB RAM, 21,184 docs from WARC 00000)._

### `cc warc markdown --file 0`

| Phase | Metric | Result |
|-------|--------|--------|
| Phase 1 Extract | docs/s | 360 docs/s |
| Phase 1 Extract | MB/s read | 48.3 MB/s |
| Phase 1 Extract | elapsed | 1m0.6s |
| Phase 2 Convert | docs/s | 25 docs/s |
| Phase 2 Convert | MB/s read | 3.3 MB/s |
| Phase 2 Convert | elapsed | 14m23s |
| Total | elapsed | 15m34s |
| Total | docs converted | 21,184 (604 errors) |
| Total | peak RSS | ~290 MB |
| Total | `markdown/00000/` disk | 297 MB |

### `cc fts pack --file 0 --format all`

| Format | docs/s | elapsed | file size | notes |
|--------|--------|---------|-----------|-------|
| parquet | 8,739 | 2.4s | 31.4 MB | |
| bin | 35,456 | 0.6s | 88.8 MB | |
| duckdb | 871 | 24.3s | 138.5 MB | |
| markdown (.bin.gz) | 3,431 | 6.2s | 29.3 MB | 33% of bin size (67% compression) |

Peak RSS (total run, all formats sequentially): ~455 MB.
`.bin.gz.idx` index file: 4.0 KB.

### `cc fts index --file 0`

| Engine | Source | docs/s | elapsed | peak RSS | index disk |
|--------|--------|--------|---------|----------|------------|
| rose | files | 1,118 | 19s | ~913 MB | 51 MB |
| rose | markdown | 1,155 | 18.3s | ~889 MB | 51 MB |
| sqlite | files | 662 | 32s | ~294 MB | 143 MB |
| bleve | files | 342 | 1m2s | ~1,878 MB | 394 MB |
| duckdb | files | 151 | 2m20s | ~1,377 MB | 189 MB |

Rose `source files` vs `source markdown`: throughput essentially identical (~1,100 docs/s).
SQLite lowest peak RSS (294 MB). Rose smallest index (51 MB).

### `cc fts search "machine learning"` (fan-out 1 WARC, process startup included)

| Engine | Latency (3 runs avg) |
|--------|---------------------|
| rose | ~940 ms |
| bleve | ~400 ms |
| sqlite | ~430 ms |
| duckdb | ~1,130 ms |

Fan-out 3 WARCs: not yet measured (only 1 WARC downloaded).
