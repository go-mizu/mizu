# 0685: API Fast Response (<50ms)

## Goal

All browse/metadata API endpoints return in <50ms through heavy caching
and background refresh jobs.

## Architecture

### In-memory cache (already exists)

`DocStore.shardCache` holds all DocRecords per shard in memory. Read
operations (`ListDocs`, `GetDoc`, `GetShardMeta`) serve from cache and
never touch DuckDB. Cache is populated on first access via `warmShard`
(reads entire DuckDB file once).

### Cache warm on startup

`NewDashboard` → `DocStore` → warm all existing `.meta.duckdb` files
into memory on first access. Subsequent requests are pure in-memory
map lookups + sorting = O(N) for listing, O(1) for single doc.

### Background stale refresh

When `handleBrowseDocs` detects stale metadata (>1 hour old), it
triggers a background goroutine to re-scan the `.md.warc.gz` file.
The scan writes to DuckDB and invalidates the cache. The next request
re-warms from DuckDB into memory.

### WebSocket push

After any scan completes, `broadcastShardScan` sends a WebSocket event
so the frontend knows to refresh.

## Endpoint Performance Analysis

| Endpoint | Source | Expected |
|----------|--------|----------|
| `GET /api/browse` (shard list) | MetaManager cache + DocStore cache | <5ms |
| `GET /api/browse?shard=X` (docs) | DocStore in-memory cache (map → sort → slice) | <10ms |
| `GET /api/browse/stats?shard=X` | DuckDB read-only query | <30ms |
| `GET /api/doc/{shard}/{docid}` | DocStore cache (meta) + offset seek + gzip decompress | <20ms |
| `GET /api/search?q=X` | FTS engine query + DocStore enrichment (parallel) | ~50-200ms (engine-bound) |
| `GET /api/overview` | All in-memory caches | <5ms |
| `GET /api/jobs` | In-memory job list | <1ms |
| `GET /api/warc` | MetaManager cached WARCs | <10ms |

### Doc retrieval: offset-based O(1)

Before: `readDocFromWARCMd` scanned the entire .md.warc.gz sequentially
looking for a matching record ID. With 40K+ records per shard, this
could take seconds.

After: `ReadDocByOffset(path, offset, size)` seeks to the exact byte
offset, opens one gzip member, reads one WARC record. Constant time.

## Changes Made

- `doc_store.go`: Added `gzip_offset`, `gzip_size`, `host` columns;
  countingReader + bufio for offset tracking; `ReadDocByOffset` for O(1) reads
- `server.go`: `handleDoc` uses offset-based read; removed markdown/ fallback
- DuckDB stats query uses pre-extracted `host` column (no regex)
- Title extraction reads 2KB (was 256 bytes) for better hit rate
- `browse.js`: Added Host column to doc table

## Tests (no mocks)

All tests in `doc_store_test.go` use real DuckDB + real .md.warc.gz files:
- `TestDocStoreScanAndList`: scan, list, verify metadata fields
- `TestDocStoreOffsetRead`: verify offset-based read returns correct doc body
- `TestDocStoreScanAll`: multi-shard scan
- `TestDocStoreShardStats`: verify stats with top domains, size buckets, date histogram
- `TestExtractHost`: host extraction unit tests
- `TestExtractDocTitle`: title extraction unit tests
