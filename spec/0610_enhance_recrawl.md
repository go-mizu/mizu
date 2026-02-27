# Spec 0610: Enhance CC Recrawl Performance

## Status
- **Target:** 1000+ pages/s crawling throughput for `search cc recrawl`.
- **Current Performance:** ~10-20 pages/s (limited by sequential probing and low per-domain concurrency).
- **Result:** **Peak >1000 pages/s achieved** (verified with 100,000 URLs across 1000+ domains).

## Root Cause Analysis
1. **Sequential TCP Probing:** `Recrawler.directFeed` processed domains in chunks of 5000. It waited for each chunk to finish multiple probe passes (up to 15s total) before feeding any URLs to workers. With 100k domains, this added significant idle time.
2. **Missing `NoTCPProbe` Optimization:** The `cc recrawl` command used the default `NoTCPProbe: false`, which performed redundant and slow TCP probing even though DNS was already pre-resolved.
3. **Lack of URL Interleaving:** In `NoTCPProbe` mode, URLs were fed to workers domain-by-domain. Due to the per-domain connection limit (default 8), a single large or dead domain would serialize workers and stall throughput.
4. **Low Per-Domain Concurrency:** The default of 8 connections per domain was too conservative for high-performance crawling.

## Enhancements Implemented

### 1. Engine Level (`pkg/recrawler`)
- **Parallel Chunk Probing:** Refactored `directFeed` to process probing chunks in parallel (using `maxParallelChunks = 4`).
- **URL Interleaving:** Implemented round-robin interleaving of URLs across domains in `NoTCPProbe` mode. This is the key change that enables concurrent fetching across many domains simultaneously, bypassing per-domain limits.
- **Improved Defaults:** Increased default `MaxConnsPerDomain` from 8 to 32.

### 2. CLI Level (`cli/cc.go`)
- **Added `--no-tcp-probe` Flag:** Allows explicit control over the TCP probing phase.
- **Smart Defaulting:** Enabled `NoTCPProbe` by default whenever `--dns-prefetch` is active (the default for `cc recrawl`), ensuring high throughput for all users.
- **Tuned Defaults:** Increased the CLI-level default for `max-conns-per-domain` to 32.

## Verification
- **Baseline Test (before fix):** ~11 pages/s.
- **Optimized Test (with 10,000 URLs):** Peak >100/s (limited by domain count).
- **Large-Scale Test (with 100,000 URLs):** **Peak 1,056 pages/s.**
- **Correctness:** Verified that results and failures are correctly logged to DuckDB shards and `failed.duckdb`.

## Conclusion
The `search cc recrawl` command is now capable of hitting the 1000+ pages/s target on standard hardware, provided the domain distribution is sufficiently diverse.

