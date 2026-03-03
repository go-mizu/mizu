# spec/0649 — search bench: Wikipedia Index & Search Benchmark

**Status:** complete
**Branch:** index-pane

---

## Goal

Add `search bench` subcommands that benchmark full-text search engines against the
standard English Wikipedia corpus used by `quickwit-oss/search-benchmark-game`.
Results are output as a `results.json` compatible with the quickwit-oss web viewer,
enabling direct comparison with published Tantivy/Lucene numbers.

Three subcommands:
- `bench download` — stream-download Wikipedia corpus, bzip2-decompress, normalize, write `corpus.ndjson`
- `bench index --engine ENGINE` — index the corpus using any registered `pkg/index.Engine`
- `bench search --engine ENGINE` — run 962 standardized queries, measure per-query latency

Primary engines: `rose` (pure Go, embedded) and `tantivy` (CGO, build-tagged).

---

## Data Directory

```
$HOME/data/search/bench/
├── corpus.ndjson              ← normalized Wikipedia docs (one per line)
├── queries.jsonl              ← embedded in binary (data/queries.jsonl)
├── index/{engine}/            ← per-engine index files
└── results/{timestamp}.json   ← timestamped results
```

---

## Corpus

**Source:** `https://www.dropbox.com/s/wwnfnu441w1ec9p/wiki-articles.json.bz2`

Raw line format:
```json
{"url": "https://en.wikipedia.org/wiki/...", "title": "...", "body": "..."}
```

**Transform** (Go, replaces `corpus_transform.py`):
1. Skip docs with empty `url`
2. Replace `[^a-zA-Z]+` with a single space
3. Lowercase the full text

**Output `corpus.ndjson`** — one JSON object per line:
```json
{"doc_id":"https://en.wikipedia.org/wiki/Griffith_Observatory","text":"griffith observatory is a facility..."}
```

The fields use the long names `doc_id` / `text` to be directly readable by
`pkg/index.RunPipelineFromNDJSON` (which reads `"i"` and `"t"` short keys).
Wait — we match the internal NDJSON format with keys `"i"`/`"t"` for pipeline
compatibility, but the bench corpus uses `"doc_id"`/`"text"` for readability.
**Resolution:** bench corpus uses standard `{"doc_id":"...","text":"..."}` keys;
a dedicated `benchCorpusReader` in `corpus.go` reads those keys directly without
going through RunPipelineFromNDJSON (which uses `"i"`/`"t"` short keys).

---

## Queries

`data/queries.jsonl` embedded via `//go:embed data/queries.jsonl`.
Content copied verbatim from `quickwit-oss/search-benchmark-game/queries.txt` (962 lines).

Format:
```json
{"query": "+griffith +observatory", "tags": ["intersection", "global", "intersection:num_tokens_2"]}
{"query": "\"griffith observatory\"", "tags": ["phrase", "phrase:num_tokens_2"]}
{"query": "griffith observatory", "tags": ["union", "global", "union:num_tokens_2"]}
```

Users may override the embedded file with `--queries FILE`.

---

## CLI

```
search bench download
    [--url URL]          # default: Dropbox wiki-articles.json.bz2
    [--dir DIR]          # default: $HOME/data/search/bench
    [--docs N]           # stop after N docs (0 = all, ~6 M)
    [--force]            # overwrite existing corpus.ndjson

search bench index --engine ENGINE
    [--dir DIR]          # default: $HOME/data/search/bench
    [--docs N]           # index first N docs (0 = all)
    [--batch-size 5000]
    [--workers N]        # parallel batch workers (0 = NumCPU)
    [--addr ADDR]        # for external engines

search bench search --engine ENGINE
    [--dir DIR]
    [--queries FILE]     # default: embedded queries.jsonl
    [--commands LIST]    # comma-separated: TOP_10,COUNT,TOP_10_COUNT (default: TOP_10)
    [--iter N]           # iterations per query per command (default: 10)
    [--warmup DURATION]  # warmup before timing (default: 30s)
    [--output FILE]      # default: {dir}/results/{timestamp}.json
    [--addr ADDR]
```

---

## Commands → Search Mapping

| Command | `Query.Limit` | Result field | Notes |
|---------|--------------|--------------|-------|
| `COUNT` | 1000 | `Results.Total` (fallback `len(Hits)`) | Count only, no ranking needed |
| `TOP_10` | 10 | — | Standard ranked retrieval |
| `TOP_10_COUNT` | 10 | `Results.Total` (fallback `len(Hits)`) | Top-10 plus total count |

---

## Progress & Summaries

### `bench download`

Live (stderr, 200 ms refresh):
```
downloading  ████████░░  3.24/8.50 GB  │  12.4 MB/s  │  847,234 docs  │  1,230 docs/s  │  RSS 48 MB  │  eta 8m32s
```

Summary:
```
── bench download complete ──────────────────────────────
  docs:          5,912,847
  corpus size:   4.82 GB
  elapsed:       11m24s
  avg dl speed:  12.7 MB/s
  avg doc rate:  8,640 docs/s
  path:          ~/data/search/bench/corpus.ndjson
```

### `bench index`

Live:
```
bench index [rose]  ████████░░  150,000/173,720 docs  │  1,764 docs/s  │  98.5s  │  RSS 245 MB  │  disk 191 MB
```

Summary:
```
── bench index complete ─────────────────────────────────
  engine:        rose
  docs:          173,720
  elapsed:       1m38.5s
  avg rate:      1,764 docs/s
  peak RSS:      312 MB
  disk:          191 MB
  path:          ~/data/search/bench/index/rose/
```

### `bench search`

Live (per query):
```
bench search [rose / TOP_10]  q 42/962 "+san +francisco"  │  p50=1.2ms  p95=2.1ms  min=0.9ms  max=3.4ms  │  19%
```

Summary per command:
```
── bench search [rose / TOP_10] ─────────────────────────
  queries:       962
  iterations:    10  (after 30s warmup)
  p50 (median):  1.4ms
  p95:           3.8ms
  p99:           8.2ms
  slowest:       "+west +palm +beach +florida" → 22.1ms
  fastest:       "the" → 0.2ms
  results:       ~/data/search/bench/results/2026-03-03T14:22:00.json
```

---

## Output: `results/{timestamp}.json`

Compatible with `quickwit-oss/search-benchmark-game` web viewer:

```json
{
  "details": {
    "rose": [{"docs": 173720, "index_time_s": 98.5, "disk_mb": 191}]
  },
  "results": {
    "TOP_10": {
      "rose": [
        {
          "query": "+griffith +observatory",
          "tags": ["intersection", "global"],
          "count": 3,
          "duration": [1234, 1367, 1378, 1389, 1390, 1401, 1412, 1421, 1445, 1456]
        }
      ]
    }
  }
}
```

`duration` is sorted ascending microseconds. The quickwit viewer uses the minimum.

---

## Package Layout

```
blueprints/search/
├── data/
│   └── queries.jsonl              ← //go:embed target (copied from reference repo)
├── pkg/index/bench/
│   ├── corpus.go                  ← Download(): HTTP+bzip2+transform+write; DownloadStats
│   ├── runner.go                  ← BenchRunner: warmup + timed runs + per-query stats
│   └── results.go                 ← BenchResults, QueryResult, EngineDetails; JSON marshal
└── cli/
    └── bench.go                   ← newBench(), newBenchDownload(), newBenchIndex(), newBenchSearch()
```

`cli/root.go` gains one line: `root.AddCommand(NewBench())`.

---

## Key Types

### `pkg/index/bench/corpus.go`

```go
type DownloadConfig struct {
    URL     string // default wiki-articles.json.bz2 dropbox URL
    OutPath string // absolute path for corpus.ndjson
    MaxDocs int64  // 0 = unlimited
    Force   bool   // overwrite if exists
}

type DownloadStats struct {
    BytesDownloaded atomic.Int64 // compressed bytes read from network
    BytesWritten    atomic.Int64 // uncompressed bytes written to corpus.ndjson
    DocsWritten     atomic.Int64
    StartTime       time.Time
    TotalBytes      int64 // from Content-Length header (0 = unknown)
}

// Download streams, decompresses, transforms, writes corpus.ndjson.
// Calls progress every 200ms. Returns stats when done or on error.
func Download(ctx context.Context, cfg DownloadConfig, progress func(*DownloadStats)) (*DownloadStats, error)
```

### `pkg/index/bench/results.go`

```go
type EngineDetails struct {
    Docs         int64   `json:"docs"`
    IndexTimeSec float64 `json:"index_time_s"`
    DiskMB       int64   `json:"disk_mb"`
}

type QueryResult struct {
    Query    string   `json:"query"`
    Tags     []string `json:"tags"`
    Count    int      `json:"count"`
    Duration []int    `json:"duration"` // sorted ascending microseconds
}

type BenchResults struct {
    Details map[string][]EngineDetails              `json:"details"`
    Results map[string]map[string][]QueryResult     `json:"results"`
}
```

### `pkg/index/bench/runner.go`

```go
type BenchConfig struct {
    Command      string        // "TOP_10" | "COUNT" | "TOP_10_COUNT"
    Queries      []BenchQuery  // parsed from queries.jsonl
    Iter         int           // timing iterations (default 10)
    Warmup       time.Duration // warmup duration (default 30s)
}

type BenchQuery struct {
    Query string   `json:"query"`
    Tags  []string `json:"tags"`
}

type IterStats struct {
    P50, P95, Min, Max time.Duration
}

// Run executes the benchmark for one command.
// Calls progress(queryIdx, total, query, stats) after each query's iterations complete.
func Run(ctx context.Context, eng index.Engine, cfg BenchConfig,
    progress func(idx, total int, q string, s IterStats)) ([]QueryResult, error)
```

---

## Implementation Order

1. Copy `queries.txt` → `data/queries.jsonl` (embed source)
2. `pkg/index/bench/results.go` — types + JSON + `LoadResults`/`SaveResults` + test
3. `pkg/index/bench/corpus.go` — `Download()` with streaming pipeline + test for transform
4. `pkg/index/bench/runner.go` — `Run()` with warmup + timed loop + stats + test
5. `cli/bench.go` — three subcommands, progress, summaries
6. Wire `NewBench()` into `cli/root.go`
7. `go build ./...` — verify compilation
8. Manual smoke test: `bench download --docs 1000`, `bench index --engine devnull --docs 1000`, `bench search --engine devnull --docs 1000 --warmup 0s --iter 1`

---

## Benchmark Results

Corpus: English Wikipedia (5,032,104 docs full).
Machine: Apple M-series (darwin/arm64), local NVMe, `go build -tags tantivy`.
Engines: **sqlite** (FTS5), **duckdb** (FTS extension via Appender API), **rose** (pure Go), **tantivy** (CGO/Rust).

### Index Performance

| Engine | Docs | Index time | Rate (docs/s) | Disk |
|--------|-----:|-----------|--------------|-----:|
| sqlite | 100 | < 1ms | 13,936 | 220 KB |
| duckdb | 100 | < 1ms | 4,103 | 586 KB |
| rose | 100 | < 1ms | 20,077 | 0 B |
| tantivy | 100 | 100ms | 693 | 124 KB |
| sqlite | 1,000 | 100ms | 13,426 | 1.9 MB |
| duckdb | 1,000 | 100ms | 11,840 | 5.7 MB |
| rose | 1,000 | < 1ms | 28,469 | 256 KB |
| tantivy | 1,000 | 200ms | 5,922 | 1.2 MB |
| sqlite | 10,000 | 900ms | 11,742 | 16.7 MB |
| duckdb | 10,000 | 700ms | 14,586 | 37.0 MB |
| rose | 10,000 | 300ms | 39,224 | 3.5 MB |
| tantivy | 10,000 | 400ms | 22,879 | 10.7 MB |

Full-dataset results (rose + tantivy only):

| Engine | Docs | Index time | Rate (docs/s) | Disk | Peak RSS |
|--------|-----:|-----------|--------------|-----:|--------:|
| **rose** | **5,032,104** | **6m40s** | **12,563** | **3.2 GB** | **22.4 GB** |
| **tantivy** | **5,032,104** | **7m6s** | **11,808** | **7.3 GB** | **278 MB** |

Notes:
- **Rose** is fastest at indexing across all sizes (20k–39k docs/s), thanks to in-memory architecture with no per-commit overhead.
- **Tantivy** has ~200ms fixed startup cost (CGO + Rust runtime init); at 10k+ docs, throughput reaches 23k docs/s.
- **SQLite** is steady at ~12k docs/s with low disk overhead (FTS5 triggers update the virtual table incrementally).
- **DuckDB** index rate varies; disk usage is 2–3× larger (FTS extension creates auxiliary BM25 tables via `PRAGMA create_fts_index`).
- Rose full-scale peak RSS = 22.4 GB (entire index in RAM); tantivy = 278 MB (memory-mapped).

### Search Performance (TOP_10, 962 queries)

All engines: 10 iterations after warmup. DuckDB: 3 iterations, no warmup (O(n) full-scan per query makes it too slow for more).

| Engine | Docs | p50 | p95 | p99 | Slowest query |
|--------|-----:|----:|----:|----:|--------------|
| **sqlite** | **100** | **33 µs** | **36 µs** | **36 µs** | "the garden of eden" → 1.1 ms |
| duckdb | 100 | 5.5 ms | 5.8 ms | 5.8 ms | "+business +consultants" → 15.8 ms |
| rose | 100 | 67 µs | 75 µs | 75 µs | "customer +service phone number" → 353 µs |
| tantivy | 100 | 31 µs | 38 µs | 38 µs | long-phrase query → 988 µs |
| **sqlite** | **1,000** | **36 µs** | **42 µs** | **42 µs** | long-phrase query → 3.6 ms |
| duckdb | 1,000 | 7.6 ms | 8.0 ms | 8.0 ms | "+ellen +degeneres +show" → 13.7 ms |
| rose | 1,000 | 247 µs | 262 µs | 262 µs | `"music videos"` → 513 µs |
| tantivy | 1,000 | 36 µs | 49 µs | 49 µs | "+interest +only" → 1.7 ms |
| **sqlite** | **10,000** | **105 µs** | **118 µs** | **118 µs** | long-phrase query → 22.2 ms |
| duckdb | 10,000 | 13.8 ms | 14.6 ms | 14.6 ms | long-phrase query → 25.5 ms |
| rose | 10,000 | 297 µs | 315 µs | 315 µs | long-phrase query → 826 µs |
| tantivy | 10,000 | 291 µs | 335 µs | 335 µs | "+care +a +lot" → 3.4 ms |

Full-dataset search (TOP_10, `--warmup 30s --iter 10`):

| Engine | Docs | p50 | p95 | p99 | Slowest query |
|--------|-----:|----:|----:|----:|--------------|
| **rose** | **5,032,104** | **14.4 ms** | **16.1 ms** | **16.1 ms** | long-phrase query → 6.8 s |
| **tantivy** | **5,032,104** | **2.7 ms** | **3.4 ms** | **3.4 ms** | `+"the who" +uk` → 227.7 ms |

Notes:
- **SQLite FTS5** is the fastest search engine up to 10k docs — 33–105 µs p50 (inverted index with minimal overhead, pure SQL).
- **Tantivy** matches SQLite at small scale (31–36 µs) and is **5× faster** than rose at full 5M scale (2.7 ms vs 14.4 ms).
- **Rose** scales linearly from 67 µs (100 docs) to 297 µs (10k docs) — consistent but slower than SQLite/tantivy due to pure-Go overhead.
- **DuckDB FTS** is 100–200× slower than SQLite/tantivy. Its `match_bm25()` function does an O(n) full-table scan per query (not an inverted index lookup). At 10k docs: 13.8 ms vs. 105 µs for SQLite.
- At 10k docs, sqlite (105 µs) < tantivy (291 µs) < rose (297 µs) ≪ duckdb (13.8 ms).
- Long-phrase queries are the universal worst case across all engines.
