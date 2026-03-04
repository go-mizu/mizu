# Design: search bench — Wikipedia Index & Search Benchmark

**Date:** 2026-03-03
**Branch:** index-pane
**Status:** approved

---

## Goal

Add a `search bench` command family that benchmarks full-text search engines (starting
with `rose` and `tantivy`) against the standard Wikipedia corpus used by
`quickwit-oss/search-benchmark-game`. Results are output as a `results.json` compatible
with the quickwit-oss web viewer for cross-engine comparison.

Three subcommands:
- `bench download` — fetch + decompress + transform Wikipedia corpus in pure Go
- `bench index --engine ENGINE` — index the corpus using any registered engine
- `bench search --engine ENGINE` — run standardised queries and measure per-query latency

---

## Data Directory

```
$HOME/data/search/bench/
├── corpus.ndjson              ← preprocessed Wikipedia (one doc per line)
├── queries.jsonl              ← query set (embedded in binary; overridable)
├── index/{engine}/            ← per-engine index files
└── results/{timestamp}.json   ← timestamped results
```

---

## Corpus

**Source:** `https://www.dropbox.com/s/wwnfnu441w1ec9p/wiki-articles.json.bz2`
**Raw shape:** `{"url": "...", "title": "...", "body": "..."}` per line (~6 M docs, 8.5 GB bz2)

**Transform (Go equivalent of corpus_transform.py):**
- Replace `[^a-zA-Z]+` with a single space
- Lowercase entire text
- Skip docs with empty `url`

**Output: `corpus.ndjson`** — one JSON object per line:
```json
{"doc_id": "https://en.wikipedia.org/wiki/...", "text": "normalized lowercase text"}
```

---

## Queries

`queries.jsonl` embedded in the binary via `//go:embed data/queries.jsonl`.
Content copied verbatim from `quickwit-oss/search-benchmark-game/queries.txt` (225 entries).
Users may override with `--queries FILE`.

Format:
```json
{"query": "+griffith +observatory", "tags": ["intersection", "global", "intersection:num_tokens_2"]}
{"query": "\"griffith observatory\"", "tags": ["phrase", "phrase:num_tokens_2"]}
{"query": "griffith observatory", "tags": ["union", "global", "union:num_tokens_2"]}
```

---

## CLI

```
search bench download
    [--url URL]          # default: dropbox wiki-articles.json.bz2
    [--dir DIR]          # default: $HOME/data/search/bench
    [--docs N]           # stop after N docs (0 = all ~6M)

search bench index --engine ENGINE
    [--dir DIR]
    [--docs N]           # index first N docs from corpus.ndjson (0 = all)
    [--batch-size 5000]
    [--workers N]        # parallel batch workers (0 = NumCPU)
    [--addr ADDR]        # for external engines

search bench search --engine ENGINE
    [--dir DIR]
    [--queries FILE]     # default: embedded queries.jsonl
    [--commands LIST]    # comma-separated: TOP_10,COUNT,TOP_10_COUNT (default: TOP_10)
    [--iter N]           # repetitions per query per command (default: 10)
    [--warmup DURATION]  # warmup before timing (default: 30s)
    [--output FILE]      # default: {dir}/results/{timestamp}.json
    [--addr ADDR]
```

---

## Commands → Search Mapping

| Command | `Query.Limit` | Uses `Results.Total` | Notes |
|---------|--------------|----------------------|-------|
| `COUNT` | 1000 | yes → count | Fallback: len(Hits) |
| `TOP_10` | 10 | no | Standard ranked retrieval |
| `TOP_10_COUNT` | 10 | yes | Top-10 + total count |

---

## Progress & Summary

### `bench download`

Live progress (stderr, 200 ms refresh):
```
downloading  ████████░░  3.2/8.5 GB  │  12.4 MB/s  │  847,234 docs  │  1,230 docs/s  │  RSS 48 MB  │  eta 8m32s
```

Final summary:
```
── bench download complete ──────────────────────────────
  docs:          5,912,847
  corpus size:   4.8 GB
  elapsed:       11m24s
  avg dl speed:  12.7 MB/s
  avg doc rate:  8,640 docs/s
  path:          ~/data/search/bench/corpus.ndjson
```

### `bench index`

Live progress:
```
bench index [rose]  ████████░░  150,000/173,720 docs  │  1,764 docs/s  │  98.5s  │  RSS 245 MB  │  disk 191 MB
```

Final summary:
```
── bench index complete ─────────────────────────────────
  engine:        rose
  docs:          173,720
  elapsed:       1m38s
  avg rate:      1,764 docs/s
  peak RSS:      312 MB
  disk:          191 MB
  path:          ~/data/search/bench/index/rose/
```

### `bench search`

Live progress (per query):
```
bench search [rose / TOP_10]  q 42/225 "san francisco"  │  p50=1.2ms  p95=2.1ms  min=0.9ms  max=3.4ms  │  ████████░░  19%
```

Per-command summary:
```
── bench search [rose / TOP_10] ─────────────────────────
  queries:       225
  iterations:    10  (after 30s warmup)
  p50 (median):  1.4ms
  p95:           3.8ms
  p99:           8.2ms
  slowest:       "west palm beach florida" → 22.1ms
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
          "duration": [1234, 1456, 1389, 1412, 1401, 1378, 1445, 1390, 1421, 1367]
        }
      ]
    }
  }
}
```

`duration` array is sorted ascending microseconds (10 iterations). The quickwit web viewer
reads the minimum (best) time from this array for display.

---

## Package Layout

```
pkg/index/bench/
├── corpus.go      HTTP download + bzip2 stream + transform → corpus.ndjson writer
├── runner.go      BenchRunner: warmup loop, timed iterations, per-query stats
└── results.go     BenchResults / QueryResult structs; JSON marshal

cli/bench.go       "search bench" cobra command + download/index/search subcommands

data/queries.jsonl embedded with //go:embed (copied from quickwit-oss repo)
```

`bench index` reuses `pkg/index.RunPipelineFromNDJSON` (new) or streams directly via
a simple NDJSON reader → batcher → `engine.Index()`. No new indexing logic.

---

## Implementation Order

1. Copy `queries.txt` → `data/queries.jsonl`
2. `pkg/index/bench/results.go` — BenchResults struct + JSON
3. `pkg/index/bench/corpus.go` — HTTP stream + bz2 + transform + NDJSON write + progress
4. `pkg/index/bench/runner.go` — BenchRunner (warmup + timed iterations + stats)
5. `pkg/index/pack_ndjson.go` (or inline) — NDJSON corpus reader for indexing
6. `cli/bench.go` — all three subcommands with progress + summaries
7. Wire `search bench` into `cli/root.go`
8. `go test ./pkg/index/bench/...` — unit tests for corpus transform + results marshal
