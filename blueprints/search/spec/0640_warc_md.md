# spec/0640 — WARC → Markdown Pipeline (`search cc warc markdown`)

## Overview

A 3-phase pipeline that converts Common Crawl `.warc.gz` files into clean,
compressed Markdown files. Two modes:

- **File mode** (default): each phase writes intermediate files to disk;
  useful for inspection/debugging and resuming after interruption.
- **In-memory mode** (`--mem`): phases are connected by Go channels;
  zero temp files, lower disk I/O, suitable for production bulk runs.

Both modes end with the same output: `markdown_gz/**/*.md.gz`.
After completion, temp files (`warc_single/` and `markdown/`) are removed
unless `--keep-temp` is set.

---

## CLI Command

```
search cc warc markdown --file 0 [flags]
```

### Flags

| Flag            | Default         | Description                                         |
|-----------------|-----------------|-----------------------------------------------------|
| `--file`        | `0`             | File index, range `0-9`, or `all`                   |
| `--crawl`       | latest          | Crawl ID (e.g. `CC-MAIN-2026-08`)                  |
| `--workers`     | `NumCPU`        | Parallel workers for convert and compress phases    |
| `--fast`        | false           | go-readability instead of trafilatura (3–8× faster) |
| `--force`       | false           | Re-process existing files                           |
| `--mem`         | false           | Streaming pipeline; no temp files                   |
| `--keep-temp`   | false           | Keep `warc_single/` and `markdown/` after pipeline  |
| `--status`      | `200`           | HTTP status filter                                  |
| `--mime`        | `text/html`     | MIME type filter                                    |
| `--max-body`    | `524288`        | Max HTML body bytes per record                      |

---

## Data Layout

```
$HOME/data/common-crawl/CC-MAIN-2026-08/
  warc/
    CC-MAIN-20260206181458-20260206211458-00000.warc.gz   ← input
  warc_single/                                             ← Phase 1 output (file mode only)
    5d/0e/22/5d0e2270-349c-4861-bf28-1234567890ab.warc
    ...
  markdown/                                                ← Phase 2 output (file mode only)
    5d/0e/22/5d0e2270-349c-4861-bf28-1234567890ab.md
    ...
  markdown_gz/                                             ← Phase 3 output (final, kept)
    5d/0e/22/5d0e2270-349c-4861-bf28-1234567890ab.md.gz
    ...
```

### Record ID → Path mapping

WARC-Record-ID: `<urn:uuid:5d0e2270-349c-4861-bf28-1234567890ab>`

Stripping `<urn:uuid:>` yields UUID: `5d0e2270-349c-4861-bf28-1234567890ab`

Path: `{UUID[0:2]}/{UUID[2:4]}/{UUID[4:6]}/{UUID}.warc`
→ `5d/0e/22/5d0e2270-349c-4861-bf28-1234567890ab.warc`

The first 6 hex characters give 3 levels of 2-char directory sharding,
limiting each directory to ~256 entries at the leaf level.

---

## Phase 1 — Extract (`warc.gz → warc_single`)

**Input:** `warc/CC-MAIN-*.warc.gz` (selected by `--file`)
**Output:** `warc_single/**/*.warc` (one file per HTML record)
**Format:** raw HTML body bytes (NOT full WARC format — the `.warc` extension
            is a convention for the directory structure)

### Algorithm

1. For each selected `.warc.gz` file (each file processed by a worker):
2. Stream WARC records via `pkg/warc.Reader` (handles gzip + plain)
3. Filter: `WARC-Type: response`, HTTP status = 200, MIME = `text/html`
4. Parse HTTP response block: extract status, MIME, body bytes
5. Strip HTTP headers; keep raw HTML body
6. Write HTML body to `warc_single/{path}.warc` (atomic tmp+rename)
7. Track stats: files written, bytes read/written, errors

### Progress display (every 500ms)
```
  Extracting: 12,450/? (?)  1,250 docs/s  R:45.2 MB/s  W:32.1 MB/s  Mem:142 MB
```
Total is unknown upfront (streaming); `?` is shown.

---

## Phase 2 — Convert (`warc_single → markdown`)

**Input:** `warc_single/**/*.warc` (plain HTML bytes)
**Output:** `markdown/**/*.md` (clean Markdown text)

### Algorithm

1. Walk `warc_single/` recursively, collect all `.warc` files
2. Parallel workers (N = `--workers`):
   a. Read file as HTML bytes
   b. Call `markdown.Convert(html, "")` (or `ConvertFast` if `--fast`)
   c. If `HasContent && Markdown != ""`: write to `markdown/{path}.md`
   d. Record in index DuckDB at `markdown_gz/index.duckdb`
3. Files with no extractable content: counted as errors (no output written)

### Worker auto-tune (file mode only)

Before Phase 2, benchmarks 8/16/32/64/128/256 workers on a 200-file sample
and selects the fastest count for this machine. Override with `--workers N`.

### Progress display
```
  Converting: 8,200/12,450 (65.9%)  820 docs/s  R:28.1 MB/s  W:2.4 MB/s  Mem:318 MB
```

---

## Phase 3 — Compress (`markdown → markdown_gz`)

**Input:** `markdown/**/*.md`
**Output:** `markdown_gz/**/*.md.gz` (gzip BestSpeed)

### Algorithm

1. Walk `markdown/` recursively, collect all `.md` files
2. Parallel workers:
   a. Read `.md` file
   b. Compress with klauspost/gzip at `BestSpeed`
   c. Write `.md.gz` atomically

### Progress display
```
  Compressing: 9,100/9,300 (97.8%)  3,800 docs/s  R:2.4 MB/s  W:1.1 MB/s  Mem:175 MB
```

---

## In-Memory Mode (`--mem`)

Three stages connected by buffered channels. No temp files.

```
  .warc.gz ──► producer ──► warcCh(500) ──► N converters ──► mdCh(500) ──► M writers
                                                                              │
                                                                              ▼
                                                                     markdown_gz/**/*.md.gz
```

- **Producer** (1 goroutine): streams `.warc.gz` → sends `WARCItem{recordID, htmlBody}` to `warcCh`
- **Converters** (N goroutines = workers): read `warcCh` → `markdown.Convert` → send `MarkdownItem` to `mdCh`
- **Writers** (N goroutines): read `mdCh` → gzip compress → write `.md.gz`

### Memory estimate
- `warcCh` cap=500 × ~200KB avg = ~100MB HTML buffer
- `mdCh` cap=500 × ~20KB avg = ~10MB markdown buffer
- Peak: ~110MB pipeline + GC overhead

### Comparison to file mode

| Metric        | File mode               | In-memory mode         |
|---------------|-------------------------|------------------------|
| Temp disk I/O | ~2× input size extra    | Zero                   |
| Peak RAM      | ~200–400MB              | ~150–300MB             |
| Resumable     | Yes (skip existing)     | No (full re-run)       |
| Debuggable    | Yes (inspect `.warc`)   | No                     |
| Throughput    | Similar (disk-bounded)  | 5–15% faster typically |

---

## Progress & Summary Format

### Per-phase progress (live, 500ms)
```
  {Verb}: {done}/{total} ({pct}%)  {docs/s} docs/s  R:{MB/s} MB/s  W:{MB/s} MB/s  Mem:{MB} MB  [err:{n}]
```

### Per-phase summary
```
  ✓ Extract done
    Files   12,450 processed  (0 skipped)
    Rate    1,250 docs/s  ·  45.2 MB/s read  ·  32.1 MB/s write
    Time    9.960s
    Disk    warc_single/  →  583.2 MB
    RAM     before 45 MB  →  after 142 MB  (peak 158 MB)
```

### Final summary table
```
  ────────────────────────────────────────────────────────────────────────────────
  Phase        Files     Read     Write   Disk out    Rate    RAM pk  Time
  ────────────────────────────────────────────────────────────────────────────────
  Extract     12,450   562 MB   583 MB    583 MB  1,250/s   158 MB   9.960s
  Convert      9,300   583 MB    47 MB     51 MB    930/s   348 MB  10.010s
  Compress     9,300    47 MB    18 MB     18 MB  3,800/s   175 MB   2.450s
  ────────────────────────────────────────────────────────────────────────────────
  Total       12,450  1,192 MB   18 MB     18 MB    558/s   348 MB  22.420s
  ────────────────────────────────────────────────────────────────────────────────
```

### Disk layout (after cleanup)
```
  warc_single/  DELETED (was 583 MB)
  markdown/     DELETED (was 51 MB)
  markdown_gz/  18 MB  (-61.7% vs markdown/)
```

---

## Implementation

### Package: `pkg/warc_md`

```
pkg/warc_md/
  config.go    — Config, directory helpers
  types.go     — PhaseStats, ProgressFunc, WARCItem, MarkdownItem
  path.go      — WARC-Record-ID → sharded file path
  mem.go       — trackPeakMem helper
  extract.go   — Phase 1: RunExtract
  convert.go   — Phase 2: RunConvert
  compress.go  — Phase 3: RunCompress
  pipeline.go  — RunFilePipeline + RunInMemoryPipeline
```

### CLI: `cli/cc_warc_markdown.go`

New subcommand registered in `cli/cc_warc.go` under `cc warc`:
```
search cc warc markdown --file 0
```

---

## Cleanup

After all phases succeed (and `--keep-temp` is NOT set):

1. `rm -rf warc_single/` (Phase 1 temp)
2. `rm -rf markdown/` (Phase 2 temp)

Only `markdown_gz/` is kept as the final output.

---

## Download Trigger

If the selected `.warc.gz` file is not present on disk, the command
automatically downloads it before starting Phase 1 (same as `cc warc download`).

---

## Actual Benchmarks (Apple M4 Pro, local SSD)

**Test:** `CC-MAIN-20260206181458-20260206211458-00000.warc.gz` (file 0)
- Input: 1.1 GB `.warc.gz` → 21,788 HTML records extracted
- Flags: `--fast --workers 8 --force`
- Errors: 1,073 records with no extractable content

### File mode

| Phase    | Files  | Time   | Rate     | Read     | Write    | Disk out | RAM pk |
|----------|--------|--------|----------|----------|----------|----------|--------|
| Extract  | 21,788 | 14.2s  | 1,533/s  | 2.9 GB   | 2.8 GB   | 2.8 GB   | 95 MB  |
| Convert  | 20,715 | 39.9s  | 545/s    | 2.8 GB   | 88 MB    | 88 MB    | 212 MB |
| Compress | 20,714 | 3.7s   | 5,542/s  | 88 MB    | 33 MB    | 38 MB    | 212 MB |
| **Total**| 21,788 | **61.4s** | **355/s** | 5.7 GB | 33 MB  | **38 MB**| 212 MB |

CPU: 317s user at 556% (5.6 cores avg)

### In-memory mode (`--mem`)

| Metric      | Value                            |
|-------------|----------------------------------|
| Total time  | **46.2s** (1.33× faster)         |
| Rate        | 448 docs/s                       |
| Read        | 2.9 GB (WARC only, no re-read)   |
| Write       | 0.7 MB/s final                   |
| RAM peak    | **459 MB** (2.2× more than file) |

CPU: 338s user at 728% (7.3 cores avg)

### Comparison

| Metric          | File mode | `--mem` mode  |
|-----------------|-----------|---------------|
| Total time      | 61.4s     | **46.2s**     |
| Speedup         | 1×        | **1.33×**     |
| RAM peak        | 212 MB    | 459 MB        |
| Disk temp I/O   | ~5.7 GB   | **0**         |
| Resumable       | Yes       | No            |
| Debuggable      | Yes       | No            |

**Rule of thumb:** use `--mem` for batch/production; use file mode for debugging or when RAM is limited.

### Expected Performance Range

| Phase      | Throughput        | Notes                                 |
|------------|-------------------|---------------------------------------|
| Extract    | 1,500–2,500 rec/s | gzip-bounded; sequential file read   |
| Convert    | 400–700 rec/s     | trafilatura CPU-bound; scales workers |
| Convert    | 400–600 rec/s     | go-readability `--fast` (trafilatura faster on M4 Pro) |
| Compress   | 4,000–6,000 rec/s | fast gzip (BestSpeed); disk-bound     |

A typical WARC file (`.warc.gz`, ~1 GB) contains ~20,000–90,000 HTML records.
Full pipeline (file mode, `--fast`):  ~60s.
Full pipeline (`--mem`, `--fast`):    ~46s.
