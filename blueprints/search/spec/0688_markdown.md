# 0688: Markdown Conversion Performance Optimization

**Date**: 2026-03-07
**Status**: Implemented
**Goal**: Optimize trafilatura-based HTML→Markdown conversion to < 10 min/shard on server 2 (6 CPU, 12 GB RAM)

## Problem

The markdown stage of the FTS pipeline was the bottleneck: 49 min for 4 shards (~25 min per pair with concurrency=2). Each CC WARC shard contains ~38K HTML documents. The index stage completes in 68 seconds, so the markdown stage dominates total pipeline time.

## CPU Profiling Results

Profiled `Convert()` with 200 real HTML samples from CC WARC on Apple M4:

| Component | Time (s) | % Total | Notes |
|-----------|----------|---------|-------|
| **trafilatura.ExtractDocument** | 20.79 | 26.4% | Main content extraction |
| → extractMetadata | 10.60 | 13.5% | Metadata extraction sub-call |
| → → **go-htmldate.FromDocument** | **8.66** | **11.0%** | Date extraction from full DOM |
| → → → go-dateparser.Parse | 6.79 | 8.6% | Regex-heavy date string parsing |
| → → → → regexp.doExecute | 6.37 | 8.1% | Core regex engine |
| chardet.matchHelper | 6.16 | 7.8% | n-gram charset detection (fallback path) |
| html.Parse | 6.16 | 7.8% | DOM construction |
| **GC (gcDrain)** | **7.99** | **10.1%** | GC pressure from allocations |
| htmltomarkdown | ~8.0 | ~10% | HTML→Markdown conversion |

### Key Finding

**40% of trafilatura time is date extraction** (go-htmldate → go-dateparser → regex). This is entirely unnecessary because:
- Common Crawl WARC records already contain `WARC-Date` headers
- We extract and preserve this date in the WARC output headers
- The htmldate extraction result is only stored in `extracted.Metadata` which we don't use for the date

## Optimizations Applied

### 1. Disable HtmlDate Extraction (~40% of trafilatura time saved)

```go
opts.HtmlDateMode = trafilatura.Disabled
```

The `trafilatura.Options` struct has an `HtmlDateMode` field with four modes:
- `Default`: Uses `Extensive` when fallback enabled (current — very slow)
- `Fast`: Skips external DateParser regex
- `Extensive`: Full date extraction with go-dateparser
- **`Disabled`**: Skip htmldate entirely — only extract date from metadata tags

Setting `Disabled` eliminates the entire go-htmldate → go-dateparser → regexp chain (~8.66s = 40% of trafilatura time in profiling).

### 2. Pooled html-to-markdown Converter (reduced GC pressure)

```go
var mdConverterPool = sync.Pool{
    New: func() any {
        return converter.NewConverter(
            converter.WithPlugins(
                base.NewBasePlugin(),
                commonmark.NewCommonmarkPlugin(),
            ),
        )
    },
}
```

Previously, every call to `htmltomarkdown.ConvertString()` created a new converter with fresh plugin registrations. Now converters are pooled via `sync.Pool`, reducing per-call allocations and GC pressure.

### 3. Direct ConvertNode (skip double string conversion)

Changed from `ConvertString(buf.String())` (which re-parses HTML from string) to `ConvertNode(reparsed)` (which operates on the already-parsed DOM). The render+reparse step is still needed because trafilatura's ContentNode is a partial fragment that html-to-markdown's collapse pass cannot handle.

**Note**: Attempted passing ContentNode directly to ConvertNode but it panics in the collapse pass (`index out of range`) because the fragment lacks the normalised document structure that collapse expects.

## Benchmark Results

Apple M4 (10 cores), 200 real CC WARC HTML samples:

| Converter | Before | After | Speedup |
|-----------|--------|-------|---------|
| **Convert** (trafilatura) | **21.9 ms/op** | **7.9 ms/op** | **2.77x** |
| ConvertFast (readability) | 9.6 ms/op | 5.7 ms/op | 1.68x |

The optimized `Convert` is now nearly as fast as the old `ConvertFast`, while maintaining trafilatura's superior extraction quality (deduplication, fallback, language detection, precision/recall tuning).

## Server 2 Results (Measured)

Server 2 specs: 6 vCPUs, 12 GB RAM, Ubuntu 24.04 Noble

| Metric | Before | After | Speedup |
|--------|--------|-------|---------|
| Per-shard time | ~12.5 min | **2m50s** | **4.4x** |
| 4-shard total (sequential CLI) | 49 min | **11m52s** | **4.1x** |
| Docs/sec | ~25/s | **119-131/s** | **5x** |

Single shard benchmark (shard 0):
- 21,653 HTML records → 21,184 markdown records (469 errors, 2.2%)
- 2.9 GB read → 87.8 MB written (33x compression)
- 131 docs/s sustained, peak 190/s
- **2m46s total** (well under 10 min target)

4-shard sequential run:
- 86,619 HTML records → 84,681 markdown records (1,938 errors, 2.2%)
- 11.5 GB read → 337.6 MB written
- 119 docs/s average
- **11m52s total** (sequential; with concurrency=2 in task_markdown.go, estimated ~6 min)

## Pipeline Configuration

```
markdownConcurrency = 2     // WARC files converted in parallel
workersPerShard = NumCPU*2  // converter goroutines per WARC file (12 on server 2)
```

## Files Changed

- `pkg/markdown/converter.go`: HtmlDateMode=Disabled, sync.Pool for converters, ConvertNode path
- `Dockerfile.linux-focal.dockerignore`: Fix `**/coverage*` glob that excluded `coverage.go` from vendor
- `.dockerignore`: Same coverage glob fix
- `spec/0688_markdown.md`: This document

## Bonus: Docker Build Fix

The `**/coverage*` glob in `.dockerignore` was matching `vendor/golang.org/x/text/language/coverage.go`, causing the Docker build to fail with `undefined: language.NewCoverage`. Fixed by replacing the broad glob with specific coverage output patterns (`coverage/`, `coverage.out`, `coverage.html`).

## Quality Impact

- **Content extraction**: Unchanged — trafilatura with FavorRecall, EnableFallback, Deduplicate
- **Date extraction**: Disabled — but WARC-Date from CC headers is preserved in output
- **Charset detection**: Unchanged — parseHTMLFast bypasses chardet for the initial parse; fallback path still uses it for edge cases
- **Markdown output**: Identical — same html-to-markdown plugins and configuration
