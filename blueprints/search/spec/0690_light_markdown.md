# 0690: Light Markdown Converter — 5k+ docs/s

**Status**: Research
**Date**: 2026-03-08
**Target**: 5,000+ docs/s on server 2 (6 vCPU, 12 GB RAM, AMD EPYC)

## Problem

The `cc warc pack` pipeline converts HTML WARC records to Markdown. Current throughput on server 2:

| Engine | docs/s | Error rate | Notes |
|--------|--------|-----------|-------|
| trafilatura (Convert) | 264 | 2.2% | Best quality, slowest |
| readability (ConvertFast) | ~400 | ~5% | go-readability port |
| light (ConvertLight) | 544 | 10.3% | Prototype, quality gap |
| html.Parse alone | 2,993 | — | 6-core ceiling |
| WARC reader alone | 1,662 | — | Single-threaded I/O |

**Bottleneck breakdown** (per-doc, single core):
- trafilatura ExtractDocument: ~16 ms (dom clone, baseline rescue, language detect, metadata)
- html.Parse: ~0.33 ms
- fastMarkdown: ~0.05 ms
- WARC gzip decompress + record parse: ~0.6 ms

Target is 5k docs/s = 0.2 ms/doc budget at 1 core, or 1.2 ms/doc with 6 cores.

## Architecture

Two independent improvements combine multiplicatively:

### 1. Parallel WARC Reader via Pre-Computed Offsets

**Current state**: `doc_store.go` already stores `gzip_offset` and `gzip_size` per document in `{shard}.meta.duckdb`. `ReadDocByOffset()` already does O(1) random access to individual records. But the pack pipeline reads sequentially through one gzip stream.

**Design**: For re-processing (e.g., switching converter engine on existing .md.warc.gz), read offsets from .meta.duckdb and dispatch parallel workers that each seek+decompress+convert independently.

```
.meta.duckdb ──→ load offsets ──→ partition into N chunks
                                       │
                     ┌─────────────────┼─────────────────┐
                     ▼                 ▼                 ▼
              worker 0            worker 1           worker N-1
              seek(off₀)          seek(off₁)         seek(off₂)
              gzip.Read           gzip.Read           gzip.Read
              html.Parse          html.Parse           html.Parse
              fastMarkdown        fastMarkdown         fastMarkdown
                     │                 │                 │
                     └─────────────────┼─────────────────┘
                                       ▼
                              ordered writer (by offset)
```

Each worker opens its own file descriptor (no shared state). Gzip members are independent (concatenated-gzip format). This eliminates the sequential reader bottleneck entirely.

**For first-time pack** (no .meta.duckdb yet): use the existing sequential reader. The parallel reader is an optimization for re-pack or when converting from raw HTML .warc.gz where offsets were pre-scanned.

**Pre-scan offsets from raw .warc.gz**: Add a fast offset-scanning pass that reads only gzip member boundaries (no decompression needed — scan for 0x1f 0x8b magic bytes between members, tracking byte positions). This gives offsets without needing a full .meta.duckdb. Cost: one sequential read at disk speed (~100 MB/s on server 2 SSD, ~5s for a 500 MB WARC file).

### 2. Improved ConvertLight

The current `ConvertLight` has two quality problems vs trafilatura:
1. **Higher error rate** (10.3% vs 2.2%) — too aggressive boilerplate stripping removes real content
2. **8.3% fewer output records** — threshold (`len(md) < 50`) too strict for short-but-valid pages

**Improvements**:

a. **Smarter boilerplate detection**: Instead of stripping by tag name alone, use a scoring heuristic:
   - Compute text density = `text_bytes / total_bytes` for each subtree
   - Keep subtrees with density > 0.3 even if they match boilerplate class patterns
   - Only strip `<nav>`, `<footer>` etc. if they have low text density

b. **Lower minimum threshold**: Reduce from 50 to 20 bytes. Short pages (error messages, redirects) are already filtered by the WARC record's HTTP status code.

c. **Extract `<article>` / `<main>` first**: If the page has `<article>` or `<main>` or `role="main"`, use that subtree exclusively (trafilatura does this). Fall back to `<body>` only if no article element exists.

d. **Title from `<meta og:title>`**: Extract from Open Graph tags before falling back to `<title>`.

**Expected throughput**: html.Parse (0.33 ms) + stripBoilerplate (0.05 ms) + fastMarkdown (0.05 ms) = ~0.43 ms/doc/core. With 6 cores: ~14,000 docs/s theoretical. Accounting for I/O and GC overhead: 5,000–8,000 docs/s realistic.

## Quality Validation Results

Tested on 100 real HTML pages from two CC-MAIN-2026-08 WARC shards.

| Metric | File 0 | File 1 | Target |
|--------|--------|--------|--------|
| Success rate (both OK) | 97% | 96% | >95% |
| Median Jaccard (word overlap) | 0.725 | 0.588 | >0.55 |
| Median char ratio (light/traf) | 1.12 | 1.09 | 0.8–2.0 |
| Median link ratio | 1.80 | 1.50 | <5.0 |
| Heading ratio | 1.00 | 1.27 | 0.7–1.5 |

**Key findings**:
- ~70% of pages have Jaccard >0.5 (good overlap with trafilatura)
- ~30% of pages have Jaccard <0.3 (different content selection — trafilatura's ML scoring picks different paragraphs)
- Outliers are directory/sitemap pages where light extracts 100KB+ of link text
- For search indexing, median metrics are the right measure (outliers don't affect index quality)

**Test**: `WARC_TEST_FILE=path.warc.gz go test -run TestQualityParity -v ./pkg/markdown/`

**ConvertLight improvements over prototype**:
1. `findContentRegion`: article/main/role=main detection (like trafilatura)
2. `findBestContentBlock`: div/section scoring by text + `<p>` count (fallback, requires 3+ paragraphs)
3. Text-density-aware `stripBoilerplate`: keeps high-density nav/header/footer
4. `stripLinkHeavyBlocks`: removes divs/lists where >50-60% of text is in links
5. `<aside>` always stripped
6. OG title extraction
7. Minimum content threshold lowered from 50 to 20 bytes

## Implementation Plan

### Phase 1: Improve ConvertLight quality (pkg/markdown/)
1. Add article/main element detection to `ConvertLight`
2. Add text-density scoring to `stripBoilerplate`
3. Lower minimum content threshold to 20 bytes
4. Add `<meta og:title>` extraction
5. Write `quality_test.go` with 100-doc comparison

### Phase 2: Parallel WARC reader (pkg/warc_md/)
1. Add `ScanOffsets(warcPath) → []GzipMemberOffset` — fast sequential scan for gzip member boundaries
2. Add `PackParallel(ctx, cfg, offsets, progressFn)` — parallel reader using pre-computed offsets
3. Each worker: `os.Open` → `Seek(offset)` → `gzip.NewReader` → parse WARC record → convert → result channel
4. Ordered writer collects results sorted by original offset position
5. Wire into `cc warc pack` CLI with auto-detection: if .meta.duckdb exists, use parallel path

### Phase 3: Benchmark on server 2
1. Deploy with `make build-on-server SERVER=2`
2. Run `search cc warc pack --file 0 --light` and measure docs/s
3. Compare quality output with `--light` vs default (trafilatura)
4. Target: 5,000+ docs/s with quality parity confirmed

## Open Questions

1. **Offset scan vs full decompress**: For raw .warc.gz (not .md.warc.gz), gzip members may contain multiple WARC records. Need to verify Common Crawl format guarantees one record per gzip member.
   - **Answer**: Yes, CC .warc.gz uses one record per gzip member (concatenated-gzip format). This is how CDX indexes work.

2. **Memory pressure with parallel readers**: 6 workers × 512 KB HTML body = 3 MB peak. Negligible.

3. **Writer ordering**: Parallel conversion reorders results. For .md.warc.gz this is acceptable (records are independent, browsed by offset). No ordering constraint needed.

## Benchmark Results (Server 2, 6 vCPU, 12 GB)

### Parallel Light Pack (48 workers, --light --force)

| Metric | File 0 | File 1 | File 2 | File 3 |
|--------|--------|--------|--------|--------|
| Offsets scanned | 66,094 | 66,094 | ~66K | ~66K |
| Scan time | ~20s | ~20s | ~24s | ~20s |
| Input records | 18,878 | 20,399 | ~20K | ~20K |
| Output records | 18,373 | 19,758 | ~19K | ~19K |
| Sustained rate | 1,000–1,130 docs/s | 1,000–1,130 docs/s | similar | similar |
| Effective (incl. scan) | ~400–446 docs/s | ~400–446 docs/s | similar | similar |
| Peak memory | 800 MB–1.1 GB | similar | similar | similar |

### Bottleneck Analysis

1. **Gzip decompression**: CPU-bound. Each worker decompresses one member (~3–5 KB compressed → ~20 KB). klauspost/compress is ~2× faster than stdlib but still dominates per-record time.
2. **html.Parse**: ~0.33 ms/doc, CPU-bound. Cannot be parallelized further within a single document.
3. **Offset scan**: 20s sequential overhead per 66K-member file. Could be cached in .meta.duckdb.
4. **6 vCPU ceiling**: With 48 goroutines but only 6 cores, CPU contention limits parallelism.

### Path to 5k docs/s

The single-file ceiling is ~1,100 docs/s on 6 vCPU. To reach 5k:

| Strategy | Expected throughput | Notes |
|----------|-------------------|-------|
| Multi-file parallel (5 files) | 5,000–5,500 docs/s | Each file on separate workers, aggregate rate |
| Cache offsets in .meta.duckdb | Eliminates 20s scan overhead | Effective rate improves to ~1,100/file |
| Higher core count (12+ vCPU) | ~2,200 docs/s/file | Linear scaling with CPU |
| Skip html.Parse (regex extract) | ~3,000 docs/s | Sacrifices quality — NOT recommended |

**Recommended**: Multi-file parallel processing. The `cc warc pack --file 0-4` command already iterates files sequentially. Processing 5 files concurrently with separate worker pools would reach 5k aggregate. This matches the real-world use case (batch processing many WARC shards).

### Bug Fixes During Benchmarking

1. **ScanGzipOffsets wrong offsets**: After `gz.Reset(br)`, position is AFTER gzip header consumption. Fix: track `prevEnd` before Reset, using `cr.n - int64(br.Buffered())` after Peek confirms next member.
2. **processOneOffset custom parser**: Replaced inline `parseWARCAndHTTP` with standard `warcpkg.NewReader(gz)` + `parseHTTPResponseFast`.
3. **parseMarkdownLinkLine panic** (doc_store.go:928): `strings.Fields(linkURL)[0]` on empty result. Fix: guard with `len(fields) == 0` check.

## Risks

- **Quality gap may persist**: Some pages rely on trafilatura's ML-based scoring (baseline rescue). Mitigation: article/main detection handles >80% of structured pages; density scoring handles the rest.
- **Disk I/O contention**: 6 parallel seeks on same file may thrash HDD. Server 2 has SSD, so random access is fast (~0.1 ms seek). Not a concern.
- **Single-file 5k not achievable on 6 vCPU**: CPU-bound gzip+html.Parse limits single-file throughput to ~1,100 docs/s. Multi-file parallelism is the path to 5k aggregate.
