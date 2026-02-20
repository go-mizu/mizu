# Owl Storage Driver -- Research Notes

## Paper Architecture (XStore -- OSDI 2020)

XStore is a high-performance RDMA-based key-value store that replaces
traditional B+-tree index traversal with a machine-learned index model.
The core insight is that an index is fundamentally a function from keys to
positions, and a simple regression model can approximate this function with
bounded error, enabling O(1) model prediction followed by a short bounded
binary search instead of O(log N) tree traversal.

### Two-Level Recursive Model Index (RMI)

The learned index uses a two-level Recursive Model Index (RMI) structure:

1. **Top-level model** -- A single linear model that takes a key hash and
   predicts which sub-model (segment) is responsible for that key range.
   This partitions the key space into contiguous ranges.

2. **Sub-models (segments)** -- Each sub-model is a simple piecewise linear
   regression defined by (slope, intercept). Given a key hash, the sub-model
   predicts the approximate position of that key within the sorted data array.
   Each segment also stores a `maxError` bound so the system knows the
   maximum distance the prediction can be off by.

The two-level design means lookup is: hash the key, top model selects
segment, segment predicts position, then binary search within
[predicted - maxError, predicted + maxError].

### Server-Side Sorted Array

All key-value pairs are stored in a single sorted array on the server.
This is the actual data store. The sorted order enables the learned model
to work because the mapping from key to position is monotonically
increasing, which simple linear models approximate well.

### Client-Side Learned Cache

In XStore's RDMA context, the learned model lives on the client side.
The client predicts positions locally, then issues one-sided RDMA reads
to fetch data at the predicted position. This avoids server CPU involvement
during reads. The model is small (a few KB for thousands of segments) so
it fits easily in client memory.

### Training

Each sub-model is a simple linear regression: `position = slope * keyHash + intercept`.
Training one sub-model takes approximately 8 microseconds because it is just
computing the least-squares fit over the keys in that segment's range. The
greedy segmentation algorithm walks the sorted keys and starts a new segment
whenever the prediction error would exceed the allowed bound.

### Performance

XStore achieves 80M+ read operations per second on RDMA hardware, which is
5.9x faster than the best RDMA B+-tree alternative (Cell). The learned
model reduces RDMA round trips from O(log N) tree traversals to 1-2 reads.

### Translation Table

XStore uses a translation table for logical-to-physical address mapping,
enabling the sorted array to be reorganized (compacted) without
invalidating cached model predictions on clients.

## Our Implementation Plan

We adapt the XStore learned-index architecture for a local single-process
storage driver. Instead of RDMA, we use in-process memory access. The key
data structures remain faithful to the paper.

### Data File (data.dat)

A single flat file storing all key-value pairs in sorted key order. Each
entry has the binary format:

    keyLen(2B) | key | ctLen(2B) | contentType | valLen(8B) | value | created(8B) | updated(8B) | deleted(1B)

The file is rewritten during compaction (merge of write buffer into sorted
order). Between compactions, reads use the in-memory sorted index built
at load time.

### Learned Index Model (model.dat)

The model is an array of piecewise linear segments persisted to disk:

    Segment: minKeyHash(8B) | slope(8B float64) | intercept(8B float64) | maxError(4B)

A top-level lookup finds the segment whose minKeyHash range contains the
query key hash (binary search on segment boundaries). The segment then
predicts position: `pos = slope * float64(keyHash) + intercept`. The
actual key is found by binary searching within [pos - maxError, pos + maxError].

### Write Buffer

New writes go into an in-memory map keyed by composite key (bucket + "\x00" + objectKey).
Each entry holds the value bytes, content type, timestamps, and a deleted flag.
When the buffer size exceeds the configured `buffer_size` threshold, a
compaction is triggered: the buffer entries are merge-sorted with the
existing data.dat, a new data.dat is written, and the model is retrained.

### Metadata File (meta.json)

A small JSON file tracking entry count, segment count, and model parameters.
Written atomically on each compaction.

### Write Path

1. Insert/update/delete into write buffer (in-memory map, RWMutex-protected).
2. When buffer byte size exceeds threshold, trigger compaction:
   a. Merge-sort buffer entries with existing sorted data.
   b. Write new data.dat atomically (write to temp, rename).
   c. Retrain learned model from new sorted keys.
   d. Write model.dat and meta.json.
   e. Rebuild in-memory sorted index and key array.
   f. Clear write buffer.

### Read Path

1. Check write buffer first (has latest writes).
2. If not in buffer, use learned model to predict position in sorted array.
3. Binary search within error bound to find exact key.
4. Return value bytes from data file (or memory-mapped data).

### Delete Path

Mark entry as deleted in the write buffer. On next compaction, deleted
entries are excluded from the new data.dat.

### Key Design Decisions

- **Composite key**: `bucket + "\x00" + key` ensures bucket-level isolation
  while keeping all data in a single sorted structure.
- **FNV-1a hash**: Used to convert variable-length keys to uint64 for the
  learned model. Fast and good distribution.
- **Greedy segmentation**: The training algorithm walks sorted key hashes
  and starts a new segment when the linear fit error exceeds the allowed
  bound. This is the same approach used in the original Learned Index
  paper (Kraska et al., 2018).
- **In-memory operation**: Between compactions, all reads come from the
  in-memory sorted index. The learned model provides O(1) prediction
  for lookups instead of binary search over the full sorted array.
- **Configurable segments**: The `segments` DSN parameter controls the
  target number of segments. More segments = tighter error bounds =
  faster lookups but slightly more memory.
