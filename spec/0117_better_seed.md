# 0117: Better Seed - Immutable Parquet, Mutable Database

## Overview

This spec defines a better seeding strategy for finewiki where:
- **Parquet files are immutable**: Once downloaded, they are the source of truth
- **Database is mutable**: Can be deleted and recreated from parquet at any time
- **Incremental refresh**: Database is re-seeded when parquet count differs from DB count

## Current State

### Import (cli/import.go)
- Downloads parquet files from HuggingFace
- Always re-downloads even if file exists
- No verification of existing files

### Seeding (store/duckdb/seed.sql)
- Only seeds if tables are empty
- Uses `INSERT OR IGNORE` with existence checks
- No support for incremental updates when parquet changes

### Schema (store/duckdb/schema.sql)
- `titles`: Search index table
- `pages`: Full content table
- `meta`: Simple key-value store (only tracks `seeded_at`)

## Proposed Changes

### 1. Import - Skip Existing Files

**Logic:**
```
for each parquet file from HuggingFace:
    dst = local path for this file
    if file exists AND size matches:
        print "Skipping {filename} (already exists)"
        continue
    else:
        download file
```

**Benefits:**
- Faster subsequent imports (no re-download)
- Resume interrupted imports
- Bandwidth efficient

**File:** `cli/import.go`

### 2. Enhanced Metadata Tracking

**New `meta` table entries:**

| Key | Value | Description |
|-----|-------|-------------|
| `seeded_at` | ISO timestamp | When DB was last fully seeded |
| `parquet_count` | integer | Row count from parquet at last seed |
| `parquet_glob` | string | Glob pattern used for seeding |

**Updated schema.sql:**
```sql
CREATE TABLE IF NOT EXISTS meta (
  k VARCHAR PRIMARY KEY,
  v VARCHAR NOT NULL
);
```

(No schema change needed - same structure, just more keys)

### 3. Seed Count Comparison

**Logic in `store.Ensure()`:**
```
1. Get current DB count: SELECT count(*) FROM pages
2. Get parquet count: SELECT count(*) FROM read_parquet(glob)
3. Get stored parquet_count from meta
4. If DB count differs from parquet count:
     - Delete all data (TRUNCATE titles; TRUNCATE pages)
     - Re-seed from parquet
     - Update meta with new parquet_count
5. Else if tables are empty:
     - Initial seed from parquet
     - Store parquet_count in meta
6. Else:
     - No action needed (counts match)
```

**Why delete + re-seed instead of incremental?**
- Parquet is append-only from HuggingFace
- Updates/deletes in parquet are rare
- Full re-seed is simpler and guarantees consistency
- DuckDB's `read_parquet` is very fast

### 4. Idempotent seed.sql

**New seed.sql:**
```sql
-- Clear existing data for fresh seed
DELETE FROM titles;
DELETE FROM pages;

-- Seed titles table from parquet
INSERT INTO titles
SELECT
  id,
  wikiname,
  in_language,
  title,
  lower(title) AS title_lc
FROM read_parquet('__PARQUET_GLOB__');

-- Seed pages table from parquet
INSERT INTO pages
SELECT
  id,
  wikiname,
  page_id,
  title,
  lower(title) AS title_lc,
  url,
  COALESCE(date_modified, ''),
  in_language,
  COALESCE(text, ''),
  COALESCE(wikidata_id, ''),
  COALESCE(bytes_html, 0),
  COALESCE(has_math, false),
  COALESCE(wikitext, ''),
  COALESCE(version, ''),
  COALESCE(infoboxes::VARCHAR, '[]')
FROM read_parquet('__PARQUET_GLOB__');

-- Update metadata
DELETE FROM meta WHERE k IN ('seeded_at', 'parquet_count', 'parquet_glob');

INSERT INTO meta (k, v) VALUES
  ('seeded_at', cast(now() AS VARCHAR)),
  ('parquet_count', (SELECT cast(count(*) AS VARCHAR) FROM pages)),
  ('parquet_glob', '__PARQUET_GLOB__');
```

**Key changes:**
- DELETE before INSERT (idempotent, no duplicates)
- Tracks parquet_count in meta for future comparison
- Tracks parquet_glob used for seeding

### 5. Updated Ensure() Flow

```go
func (s *Store) Ensure(ctx context.Context, cfg Config, opts EnsureOptions) error {
    // 1. Create schema (IF NOT EXISTS - idempotent)
    s.db.ExecContext(ctx, schemaDDL)

    // 2. Check if re-seed needed
    if opts.SeedIfEmpty && cfg.ParquetGlob != "" {
        needsSeed, reason := s.checkNeedsSeed(ctx, cfg.ParquetGlob)
        if needsSeed {
            log.Printf("Seeding database: %s", reason)
            seed := strings.ReplaceAll(seedSQL, "__PARQUET_GLOB__", cfg.ParquetGlob)
            s.db.ExecContext(ctx, seed)
        }
    }

    // 3. Build indexes (IF NOT EXISTS - idempotent)
    if opts.BuildIndex { ... }

    // 4. Build FTS (idempotent with overwrite=1)
    if opts.BuildFTS { ... }
}

func (s *Store) checkNeedsSeed(ctx context.Context, glob string) (bool, string) {
    // Check if tables are empty
    var dbCount int64
    s.db.QueryRowContext(ctx, "SELECT count(*) FROM pages").Scan(&dbCount)

    if dbCount == 0 {
        return true, "tables are empty"
    }

    // Get parquet count
    var parquetCount int64
    query := fmt.Sprintf("SELECT count(*) FROM read_parquet('%s')", glob)
    s.db.QueryRowContext(ctx, query).Scan(&parquetCount)

    if dbCount != parquetCount {
        return true, fmt.Sprintf("count mismatch (db=%d, parquet=%d)", dbCount, parquetCount)
    }

    return false, ""
}
```

## File Changes Summary

| File | Change |
|------|--------|
| `cli/import.go` | Skip download if file exists with matching size |
| `store/duckdb/schema.sql` | No change (same structure) |
| `store/duckdb/seed.sql` | DELETE+INSERT pattern, track parquet_count |
| `store/duckdb/store.go` | Add `checkNeedsSeed()`, update `Ensure()` |

## Usage Scenarios

### Scenario 1: Fresh Install
```
$ finewiki import vi     # Downloads parquet
$ finewiki serve vi      # Creates DB, seeds from parquet
```

### Scenario 2: Re-import (no change)
```
$ finewiki import vi     # Skips - file exists
$ finewiki serve vi      # No re-seed - counts match
```

### Scenario 3: Parquet Updated
```
$ rm ~/data/.../vi/data.parquet
$ finewiki import vi     # Downloads new parquet
$ finewiki serve vi      # Detects count mismatch, re-seeds
```

### Scenario 4: Database Corrupted
```
$ rm ~/data/.../vi/wiki.duckdb
$ finewiki serve vi      # Fresh DB, seeds from parquet
```

## Implementation Notes

1. **No backward compatibility needed** - user will delete existing duckdb
2. **Parquet has duplicates** - Use `DISTINCT ON (id)` to deduplicate during seed
3. **Count comparison uses stored count** - Compare DB count with stored `parquet_count` (post-dedup), not raw parquet count
4. **DELETE is safe** - indexes are rebuilt anyway with IF NOT EXISTS
5. **FTS uses overwrite=1** - already idempotent
6. **Parquet glob change detection** - Also reseed if parquet source path changes

## Actual Implementation Details

### Key Insight: Parquet Has Duplicates

The HuggingFace FineWiki parquet files contain duplicate IDs. For example:
- Raw parquet rows: 1,279,087
- After deduplication: 1,275,846

This means we can't compare raw parquet count with DB count directly. Instead:
1. Store the **deduplicated** count in meta after seeding
2. Compare that stored count with current DB count

### checkNeedsSeed Logic

```go
func checkNeedsSeed(ctx, glob):
    // 1. Empty tables → seed needed
    if dbCount == 0:
        return true, "tables are empty"

    // 2. Parquet source changed → reseed
    if storedGlob != glob:
        return true, "parquet source changed"

    // 3. Count mismatch with stored count → reseed
    if dbCount != storedCount:
        return true, "count mismatch"

    return false, ""
```
