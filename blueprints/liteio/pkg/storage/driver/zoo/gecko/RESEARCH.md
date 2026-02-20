# Gecko Storage Driver -- KDSep-Inspired Key-Delta Separation

## Paper Architecture (KDSep -- Key-Delta Separation)

**Reference**: "KDSep: Key-Delta Separation for Write-Efficient Key-Value Stores" (ICDE 2024)

### Core Insight

Traditional LSM-tree key-value stores write the full value on every update, even when
only a small portion of the value changes. For read-modify-write (RMW) workloads -- common
in databases (counter increments, field updates, append operations) -- this amplifies
write I/O proportionally to value size.

KDSep extends the WiscKey key-value separation idea: instead of separating keys from
values, it separates **keys from deltas** (the changes applied to values).

### Architecture Overview

```
                      +---------------------+
                      |    Write Buffer     |
                      | (in-memory, grouped |
                      |   by bucket ID)     |
                      +----------+----------+
                                 |
                    flush when threshold exceeded
                                 |
              +------------------+------------------+
              |                                     |
   +----------v-----------+            +-----------v-----------+
   |     Base Store        |            |     Delta Store        |
   |  (LSM-tree / sorted   |            |  (hash-bucketed delta  |
   |   key-value file)     |            |   files on disk)       |
   |                       |            |                        |
   |  Full key-value pairs |            |  delta_000.dat         |
   |  (authoritative data) |            |  delta_001.dat         |
   |                       |            |  ...                   |
   |                       |            |  delta_063.dat         |
   +-----------+-----------+            +-----------+------------+
               |                                    |
               +----------------+-------------------+
                                |
                          READ: merge
                       base + deltas
```

### Key Components

1. **Base Store** (LSM-tree for full key-value pairs)
   - Stores the authoritative, fully-merged value for each key.
   - Structured as a sorted key-value file.
   - Updated during GC/compaction when accumulated deltas are folded back in.

2. **Delta Store** (hash-bucketed delta files on disk)
   - Deltas (changes) are grouped by key hash into N "delta buckets."
   - Each bucket is a small append-only file on disk.
   - Bucket assignment: `hash(key) % N` where N is typically 64--256.
   - Delta entry format: key, new value, timestamp, operation type (put/delete).

3. **Write Buffer** (in-memory grouping of deltas by bucket ID)
   - Incoming writes are buffered in memory, grouped by delta bucket ID.
   - When the buffer exceeds a threshold (e.g., 1MB total), all pending deltas
     are flushed in parallel to their respective delta bucket files.
   - This batching amortizes I/O: one `write()` + `fsync()` per bucket file
     instead of per individual delta.

4. **Read Path** (merge base with pending deltas)
   - Check write buffer for pending deltas for the requested key.
   - Check the delta bucket file (determined by `hash(key) % N`).
   - Read the base value from the base store.
   - Merge: apply deltas on top of the base value to produce current state.
   - For simple put operations, the latest delta IS the current value.

5. **Delta-Based GC / Compaction**
   - Periodically, when a delta bucket accumulates too many entries, GC runs:
     1. Read all deltas from the bucket file.
     2. For each key with deltas, read the base value.
     3. Apply deltas to produce the new base value.
     4. Write updated base entries back to the base store.
     5. Truncate or replace the delta bucket file.
   - This keeps delta files small and read latency bounded.

### Performance Characteristics

- **Write amplification reduced by 27--41%** on RMW workloads (paper results).
- Small deltas avoid rewriting the full value each time.
- Batched delta flushes amortize fsync overhead.
- Read latency bounded by keeping delta files small via GC.
- Hash-bucketed delta files enable parallel flush and independent GC per bucket.

---

## Our Implementation Plan

### Mapping to the Storage Interface

We adapt the KDSep architecture to implement the `storage.Storage` / `storage.Bucket`
interfaces, using the key-delta separation model for the on-disk persistence layer.

### Data Structures

1. **Base Store File** (`base.dat`)
   - Sorted key-value pairs, append-only with periodic compaction.
   - Entry format:
     ```
     keyLen(2B) | key | ctLen(2B) | contentType | valLen(8B) | value | created(8B) | updated(8B)
     ```
   - In-memory index: `map[compositeKey] -> file offset` for O(1) lookup.
   - Composite key: `bucket + "\x00" + objectKey`

2. **Delta Bucket Files** (`delta_000.dat` ... `delta_063.dat`)
   - Default 64 buckets (configurable via DSN `delta_buckets` parameter).
   - Bucket assignment: `FNV-1a(compositeKey) % delta_buckets`
   - Delta entry format:
     ```
     keyLen(2B) | key | valLen(8B) | value | ts(8B) | op(1B: 0=put, 1=delete)
     ```
   - Append-only within each bucket file.
   - Each bucket file has its own mutex for concurrent-safe append.

3. **Write Buffer** (in-memory `map[bucketID] -> []deltaEntry`)
   - Pending deltas grouped by delta bucket ID.
   - Flushed to delta files when total buffer size exceeds 1MB.
   - Also flushed on explicit `Close()` or GC trigger.

4. **In-Memory Index** (`map[compositeKey] -> indexEntry`)
   - Maps each composite key to `{baseOffset, latestDeltaValue, contentType, size, timestamps}`.
   - Rebuilt from base + delta files on recovery (Open).

5. **GC / Compaction**
   - Triggered when a delta bucket exceeds `gc_threshold` entries (default 1000).
   - Merges deltas into base store, rebuilds the base file and index.

### File Layout

```
{root}/
  base.dat          -- base store (sorted key-value pairs)
  delta_000.dat     -- delta bucket 0
  delta_001.dat     -- delta bucket 1
  ...
  delta_063.dat     -- delta bucket 63
```

### Write Path

```
Write(key, value) ->
  1. Compute compositeKey = bucket + "\x00" + key
  2. Compute bucketID = FNV-1a(compositeKey) % delta_buckets
  3. Create delta entry {key, value, timestamp, op=put}
  4. Append to write buffer[bucketID]
  5. Update in-memory index with latest value metadata
  6. If buffer size > threshold: flush all pending deltas to disk
```

### Read Path

```
Open(key) ->
  1. Compute compositeKey = bucket + "\x00" + key
  2. Look up in-memory index -> get latest value info
  3. If value is in write buffer (pending delta): return from memory
  4. If value is in delta file: read from delta file at recorded offset
  5. If value is in base file: read from base file at recorded offset
  6. Return value
```

### Delete Path

```
Delete(key) ->
  1. Compute compositeKey
  2. Write delete-delta to buffer (op=delete)
  3. Remove from in-memory index
```

### DSN Format

```
gecko:///path/to/data?sync=none&delta_buckets=64&gc_threshold=1000
```

Parameters:
- `sync`: "none" (default, no fsync), "batch", "full"
- `delta_buckets`: number of hash buckets for delta store (default 64)
- `gc_threshold`: entries per delta bucket before GC triggers (default 1000)

### Interface Coverage

- `storage.Storage`: Bucket(), Buckets(), CreateBucket(), DeleteBucket(), Features(), Close()
- `storage.Bucket`: Name(), Info(), Features(), Write(), Open(), Stat(), Delete(), Copy(), Move(), List(), SignedURL()
- `storage.HasDirectories`: Directory() returning storage.Directory
- `storage.HasMultipart`: InitMultipart(), UploadPart(), CopyPart(), ListParts(), CompleteMultipart(), AbortMultipart()
- Iterators: `storage.BucketIter`, `storage.ObjectIter`
- `storage.Directory`: Bucket(), Path(), Info(), List(), Delete(), Move()
