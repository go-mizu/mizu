# CC Recrawl in Rust Crawler

## Goal

Implement `crawler cc recrawl` command that replicates the Go `search cc recrawl --file p:N` pipeline using the same crawler-lib engine/writer infrastructure as `crawler hn recrawl`.

## Architecture

```
CLI (cc.rs)                          crawler-lib (reused)
┌────────────────────┐               ┌──────────────────────┐
│ --file p:0         │               │ job.rs::run_job()    │
│ --crawl CC-MAIN-*  │               │ engine (reqwest/wreq)│
│ --limit, --writer  │               │ writers (bin/duckdb) │
│                    │               │ stats + TUI          │
│ 1. resolve_file()  │──seeds───────▶│ two-pass crawl       │
│ 2. download if     │               └──────────────────────┘
│    missing          │
│ 3. load_seeds_cc() │
│ 4. run_crawl_job() │
└────────────────────┘
```

## Key Differences from HN

| Aspect | HN | CC |
|--------|----|----|
| Seed source | Local .duckdb/.parquet | CC index parquet (local or download) |
| Default timeout | 1000ms | 1000ms |
| Default retry_timeout | 15000ms | 10000ms |
| Output dir | `~/data/hn/results/` | `~/data/common-crawl/{crawl_id}/recrawl/{part}/` |
| Seed query | `SELECT url, domain FROM {table}` | `SELECT url, domain FROM read_parquet() WHERE warc_filename IS NOT NULL` |
| File resolution | Direct path | Selector (p:N) → manifest → local/download |
| Expected OK rate | ~50% (many dead domains) | ~70%+ (CC pre-filtered) |
| Crawl ID | N/A | CC-MAIN-2026-08 (latest) |

## Implementation Plan

### 1. Refactor: Extract common.rs from hn.rs

Move shared functions to `crawler-cli/src/common.rs`:
- `expand_home(path) -> PathBuf`
- `parse_engine(s) -> EngineType`
- `parse_writer(s) -> WriterType`
- `create_result_writer(type, dir, shards, mem_mb, batch_size) -> Arc<dyn ResultWriter>`
- `create_failure_writer(type, path, dir, mem_mb, batch_size) -> Arc<dyn FailureWriter>`
- `run_crawl_job(RunConfig) -> Result<()>` — shared orchestration:
  1. Create live stats
  2. Build Config
  3. Create result/failure writers
  4. Spawn TUI
  5. Run job (two-pass)
  6. Close writers, stop TUI, print summary

`hn.rs` becomes thin: parse args → load seeds → call `run_crawl_job()`.

### 2. CC Seed Loading (in crawler-lib/src/seed.rs)

```rust
pub fn load_seeds_cc_parquet(path: &str, limit: usize) -> Result<Vec<SeedURL>>
```

SQL: `SELECT url, COALESCE(url_host_registered_domain, '') as domain
      FROM read_parquet('{path}')
      WHERE warc_filename IS NOT NULL
      [LIMIT {limit}]`

### 3. CC File Resolution (in cc.rs)

**Selector parsing** (`parse_cc_file_selector`):
- `N` or `p:N` or `w:N` → warc index N
- `m:N` → manifest index N
- `/path/to/file.parquet` → local path

**Resolution flow**:
1. If `--seed` provided → use directly as parquet path
2. If `--file` provided → resolve selector:
   a. Try as local file first
   b. Parse selector (kind, idx)
   c. Download manifest from CC API
   d. Filter warc subset files, pick file[idx]
   e. Check local cache at `~/data/common-crawl/{crawl_id}/index/...`
   f. If not cached → download from `https://data.commoncrawl.org/{remote_path}`

**Manifest download**:
- URL: `https://data.commoncrawl.org/crawl-data/{crawl_id}/cc-index-table.paths.gz`
- Returns gzipped text, one S3 path per line
- Filter lines containing `subset=warc/` for warc subset
- Cache manifest in `~/data/common-crawl/{crawl_id}/manifest.txt`

**Parquet download**:
- URL: `https://data.commoncrawl.org/{remote_path}`
- Save to: `~/data/common-crawl/{crawl_id}/index/{hive_path}/`
- Show progress (bytes received / total)

**Crawl ID resolution**:
- `--crawl CC-MAIN-2026-08` (explicit)
- Default: fetch latest from `https://index.commoncrawl.org/collinfo.json` (first entry)
- Cache crawl list in `~/data/common-crawl/crawls.json`

### 4. Output Directory

Per-parquet isolation (mirrors Go behavior):
```
~/data/common-crawl/{crawl_id}/recrawl/{part_name}/
  ├── results/          (sharded DuckDB or binary segments)
  ├── failed.duckdb     (failure DB for two-pass)
  └── failures/         (failure segments if binary writer)
```

Where `part_name` = e.g. `part-00000` extracted from the parquet filename.

### 5. CLI Arguments (cc.rs RecrawlArgs)

```
--seed <path>           Parquet or DuckDB file path (direct, skips manifest)
--file <selector>       CC parquet file selector: N, p:N, w:N, m:N
--crawl <id>            CC crawl ID (default: latest)
--output <dir>          Override output directory
--engine <type>         reqwest (default), hyper, wreq
--writer <type>         binary (default), duckdb, parquet, devnull
--workers <n>           0 = auto (default)
--inner-n <n>           0 = auto (default)
--timeout <ms>          Pass-1 timeout (default: 1000)
--retry-timeout <ms>    Pass-2 timeout (default: 10000)
--no-retry              Disable pass-2
--domain-dead-probe <n> Default: 3
--domain-stall-ratio <n> Default: 5
--limit <n>             0 = all seeds
--db-shards <n>         0 = auto
--db-mem-mb <n>         0 = auto
--no-tui                Disable TUI
--status-only           Filter: only 200 status seeds from CC index
--lang <code>           Filter: language (e.g. "eng")
--mime <type>           Filter: MIME type
```

### 6. Dependencies

Add to `crawler-cli/Cargo.toml`:
```toml
reqwest = { version = "0.13", default-features = false, features = ["rustls-tls", "gzip", "stream"] }
flate2 = "1"
serde_json = "1"
indicatif = "0.17"  # progress bar for downloads
```

### 7. Data on Server2

Server2 already has CC-MAIN-2026-08 data:
```
~/data/common-crawl/CC-MAIN-2026-08/index/crawl=CC-MAIN-2026-08/subset=warc/
  part-00000-ad224845-...parquet  (549 MB)
  part-00001-ad224845-...parquet  (754 MB)
  part-00100-ad224845-...parquet  (576 MB)
  part-00200-ad224845-...parquet  (559 MB)
  part-00299-ad224845-...parquet  (371 MB)
```

So `--file p:0` should resolve to the existing `part-00000` file without downloading.

### 8. Test Plan

1. `cargo check` locally
2. Build on server2 (`make build-on-server SERVER=2`)
3. Deploy (`make deploy SERVER=2`)
4. Test with existing data:
   ```
   crawler cc recrawl --file p:0 --crawl CC-MAIN-2026-08 --limit 10000 --writer devnull --no-retry
   ```
5. Verify: seeds loaded from parquet, crawl runs, summary printed
6. Test full pipeline:
   ```
   crawler cc recrawl --file p:0 --crawl CC-MAIN-2026-08 --writer binary
   ```
