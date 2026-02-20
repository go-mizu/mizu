# Kangaroo Storage Driver -- Tiered Flash Cache (SOSP 2021, Best Paper)

## Paper Architecture (Kangaroo -- Tiered Flash Cache)

**Reference**: "Kangaroo: Caching Billions of Tiny Objects on Flash" (SOSP 2021, Best Paper)

### Core Insight

Flash-based caches face a fundamental tension between write amplification and miss
ratio. Log-structured caches (like RippleDB) have low write amplification but poor
miss ratios due to FIFO eviction. Set-associative caches (like CacheLib's small object
cache) allow flexible eviction within sets but suffer from high write amplification
because inserting a single object rewrites an entire flash page.

Kangaroo resolves this tension by combining both designs into a tiered architecture:
a small log-structured tier (KLog) absorbs writes with minimal amplification, and a
large set-associative tier (KSet) stores the majority of objects with near-optimal
miss ratios. A threshold admission policy prevents one-hit wonders from polluting KSet.

### Architecture Overview

```
                    +---------------------+
                    |     DRAM LRU        |
                    |   (small, fast,     |
                    |    bounded size)    |
                    +----------+----------+
                               |
                         eviction (LRU)
                               |
                    +----------v----------+
                    |     KLog (~5%)      |
                    |  circular append-   |
                    |  only log on flash  |
                    |  + in-memory index  |
                    +----------+----------+
                               |
                 threshold admission (hitCount >= T)
                               |
                    +----------v----------+
                    |     KSet (~95%)     |
                    |  set-associative    |
                    |  pages on flash     |
                    |  + per-page bloom   |
                    |  filters            |
                    +---------------------+
```

### Key Components

1. **DRAM LRU** (small in-memory front tier)
   - Bounded number of entries held entirely in memory.
   - Provides sub-microsecond access for hot objects.
   - On eviction (least-recently-used), the entry moves down to KLog.
   - Typical size: 10K--100K entries depending on available DRAM.

2. **KLog** (~5% of total flash capacity)
   - Circular, append-only log file on flash storage.
   - Divided into fixed-size segments; when the log wraps, the oldest segment
     is scanned and surviving objects are promoted to KSet.
   - In-memory index maps each key to its {offset, size} in the log.
   - Sequential writes minimize flash write amplification (1x ideal).
   - Absorbs write bursts: many objects are evicted from DRAM before being
     accessed again, so they never need to reach KSet.

3. **KSet** (~95% of total flash capacity)
   - Fixed-size pages (typically 4KB, matching flash page size).
   - Set-associative: each key maps to exactly one page via `hash(key) % numPages`.
   - Each page stores multiple small objects packed together.
   - **Per-page bloom filters** (stored at the end of each page, typically 256 bytes):
     - Fast negative lookups: if the bloom filter says "not present," skip the
       page read entirely.
     - ~1% false positive rate with 256 bytes and typical object counts per page.
   - No in-memory per-object index needed -- just hash to find the page.
   - When a page is full and a new object must be inserted, the oldest object in
     that page is evicted.

4. **Threshold Admission** (KLog to KSet promotion filter)
   - Maintains an in-memory access counter per key (or key hash).
   - An object is promoted from KLog to KSet only when its access count reaches
     a threshold T (default T=2).
   - Prevents one-hit wonders (objects accessed only once) from consuming flash
     writes and polluting the set-associative tier.
   - The paper shows T=2 is optimal for most workloads, filtering ~60% of objects
     that would otherwise be written to KSet unnecessarily.

### Object Flow

```
Write --> DRAM LRU
                |
          LRU eviction --> KLog (append-only, sequential)
                              |
                        log wraps --> scan oldest segment
                              |
                  hitCount >= T ? --> YES --> KSet (set-associative page)
                              |
                              NO --> discard (one-hit wonder)
```

### Read Path

```
Read(key) -->
  1. Check DRAM LRU (hash map, O(1))
  2. Check KLog (in-memory index, O(1))
  3. Check KSet:
     a. Compute page = hash(key) % numPages
     b. Check bloom filter at end of page
     c. If bloom says "maybe present": scan page entries
     d. If found: return object
  4. Not found
```

### Performance Results (from paper)

- **29% fewer misses** than set-associative alone (CacheLib).
- **56% fewer misses** than log-structured alone (Segcache).
- **40% fewer flash writes** than alternatives at the same miss ratio.
- Handles billions of tiny objects (median size ~100 bytes at Facebook).
- Threshold admission filters ~60% of unnecessary KSet writes.
- KLog absorbs 95% of writes, reducing KSet write amplification to near 1x.

---

## Our Implementation Plan

### Mapping to the Storage Interface

We adapt the Kangaroo tiered cache architecture to implement the `storage.Storage` /
`storage.Bucket` interfaces. The three tiers (DRAM LRU, KLog, KSet) form the
on-disk and in-memory persistence layer.

### Data Structures

1. **L1 -- DRAM LRU** (in-memory)
   - Doubly-linked list + hash map for O(1) access and eviction.
   - Each entry: key, value []byte, contentType, created timestamp, updated timestamp, size.
   - Max entries: `l1_size` (default 10,000).
   - On eviction (LRU tail): entry moves to KLog.

2. **L2 -- KLog File** (`klog.dat`)
   - Circular append-only log file.
   - File size: `klog_mb` MB (default 64MB), pre-allocated.
   - Entry format (variable-length):
     ```
     keyLen(2B) | key | ctLen(2B) | contentType | valLen(8B) | value | created(8B) | updated(8B)
     ```
   - In-memory index: `map[compositeKey] -> {offset, size}` for O(1) lookup.
   - Write pointer wraps circularly: `writePos = (writePos + entrySize) % klogSize`.
   - When the write pointer overtakes the read frontier, the overwritten region is
     scanned and surviving entries (those still in the index) are promoted to KSet
     if they meet the threshold criterion.

3. **L3 -- KSet File** (`kset.dat`)
   - Fixed-size pages organized as a set-associative cache.
   - File size: `kset_mb` MB (default 512MB), pre-allocated.
   - Page size: `kset_page` bytes (default 4KB).
   - Number of pages: `kset_mb * 1024 * 1024 / kset_page`.
   - Page assignment: `FNV-1a(compositeKey) % numPages`.
   - Page format:
     ```
     [usedBytes(2B)] [entry0] [entry1] ... [padding] [bloomFilter(256B at page end)]
     ```
   - Entry within page:
     ```
     keyLen(2B) | key | ctLen(2B) | contentType | valLen(8B) | value | created(8B) | updated(8B)
     ```
   - Bloom filter: 256-byte bit array at the end of each page.
     - Hash functions: FNV-1a primary + bit-shifted secondary.
     - ~1% false positive rate for typical object counts per page.
   - If page is full when inserting, the oldest entry in that page is evicted
     to make room.

4. **Access Counter** (in-memory threshold admission)
   - `map[uint32] -> uint8`: maps FNV-1a(compositeKey) to hit count.
   - Promotion from KLog to KSet occurs only when `hitCount >= admissionThreshold`
     (default threshold = 2).
   - Counters are cleared when an entry is promoted or evicted from KLog.

### File Layout

```
{root}/
  klog.dat     -- circular log (L2)
  kset.dat     -- set-associative pages (L3)
  meta.json    -- write pointers, entry counts, configuration snapshot
```

### Write Path

```
Write(key, value) -->
  1. Compute compositeKey = bucket + "\x00" + key
  2. Insert into L1 DRAM LRU (front of list)
  3. If L1 is full (entries > l1_size):
     a. Evict LRU tail entry from L1
     b. Serialize evicted entry to KLog at writePos
     c. Update KLog in-memory index
     d. Advance writePos; if wrap-around:
        - Scan overwritten region for entries still in KLog index
        - For each surviving entry with hitCount >= threshold: promote to KSet
        - Remove overwritten entries from KLog index
  4. Update access counter
```

### Read Path

```
Open(key) -->
  1. Compute compositeKey = bucket + "\x00" + key
  2. Check L1 DRAM LRU (hash map lookup, O(1))
     - If found: move to front (LRU touch), return value
  3. Check KLog (in-memory index lookup, O(1))
     - If found: read from klog.dat at offset, increment access counter, return
  4. Check KSet:
     a. Compute page = FNV-1a(compositeKey) % numPages
     b. Read bloom filter (last 256 bytes of page)
     c. If bloom says "definitely not present": return not found
     d. Read full page, scan entries for matching key
     e. If found: increment access counter, return value
  5. Return not found
```

### Delete Path

```
Delete(key) -->
  1. Compute compositeKey
  2. Remove from L1 DRAM LRU (if present)
  3. Remove from KLog in-memory index (if present)
  4. Mark as deleted in KSet page (if present):
     - Read page, find entry, set valLen=0 as tombstone, rewrite page
     - Update bloom filter
  5. Remove from access counter
```

### DSN Format

```
kangaroo:///path/to/data?sync=none&l1_size=10000&klog_mb=64&kset_mb=512&kset_page=4096
```

Parameters:
- `sync`: "none" (default, no fsync), "batch", "full"
- `l1_size`: max entries in DRAM LRU (default 10,000)
- `klog_mb`: KLog file size in MB (default 64)
- `kset_mb`: KSet file size in MB (default 512)
- `kset_page`: KSet page size in bytes (default 4096)
- `admission`: threshold hit count for KLog-to-KSet promotion (default 2)

### Interface Coverage

- `storage.Storage`: Bucket(), Buckets(), CreateBucket(), DeleteBucket(), Features(), Close()
- `storage.Bucket`: Name(), Info(), Features(), Write(), Open(), Stat(), Delete(), Copy(), Move(), List(), SignedURL()
- `storage.HasDirectories`: Directory() returning storage.Directory
- `storage.HasMultipart`: InitMultipart(), UploadPart(), CopyPart(), ListParts(), CompleteMultipart(), AbortMultipart()
- Iterators: `storage.BucketIter`, `storage.ObjectIter`
- `storage.Directory`: Bucket(), Path(), Info(), List(), Delete(), Move()
