# Spec 0652: Dahlia — Pure-Go Tantivy Clone

## Summary

Dahlia is a pure-Go full-text search engine that reproduces tantivy's segment-based
architecture with BP128 compression, FST term dictionaries, Block-Max WAND scoring,
and position-aware phrase queries.

## On-Disk Format

### Index Metadata: `dahlia.meta` (JSON)
```json
{
  "version": 1,
  "doc_count": 1000000,
  "avg_doc_len": 245.3,
  "segments": ["seg_00000001", "seg_00000002"],
  "next_seg_seq": 3
}
```

### Per-Segment Files: `seg_{NNNNNNNN}/`

| File | Description | Format |
|------|-------------|--------|
| `segment.tdi` | Term dictionary (FST) | Vellum FST: term → packed u64 |
| `segment.doc` | Doc ID posting lists | BP128 blocks + VInt tail + skip + trailer |
| `segment.freq` | Term frequencies | BP128 blocks + VInt tail |
| `segment.pos` | Position data | VInt delta-encoded per doc |
| `segment.store` | Stored fields | Zstd 16KB blocks + skip index + footer |
| `segment.fnm` | Field norms | Raw uint8[docCount] |
| `segment.meta` | Segment metadata | JSON: {docCount, avgDocLen} |

### TermInfo Packing (uint64)
- Bits [0:30] — docFreq (max ~1 billion)
- Bit [30] — hasPositions flag
- Bits [32:63] — postings offset

### Skip Entry (21 bytes)
```
lastDoc:      uint32  — highest docID in block
docOff:       uint32  — .doc file byte offset
freqOff:      uint32  — .freq file byte offset
posOff:       uint32  — .pos file byte offset
blockMaxTF:   uint32  — max TF in block (WAND upper bound)
blockMaxNorm: uint8   — shortest doc norm (WAND upper bound)
```

### Posting List Layout (.doc + .freq)
Per term:
1. Full BP128 blocks (128 docs each): delta-encoded docIDs in .doc, raw freqs in .freq
2. VInt tail (< 128 remaining docs)
3. Skip entries array
4. Trailer: `[numFullBlocks:u32][tailCount:u32]`

### Position Data Layout (.pos)
Per term per doc: `[numPositions:VInt][delta0:VInt][delta1:VInt]...`

### Store Layout (.store)
1. Compressed blocks: docs serialized as `[idLen:4][id][textLen:4][text]`, grouped into 16KB blocks, zstd compressed
2. Skip index: `[lastDocID:4][blockOffset:8]` entries
3. Footer: 8-byte skip index offset

## Component Interfaces

### Engine (implements index.Engine + index.Finalizer)
```go
func (e *Engine) Name() string
func (e *Engine) Open(ctx context.Context, dir string) error
func (e *Engine) Close() error
func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error)
func (e *Engine) Index(ctx context.Context, docs []index.Document) error
func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error)
func (e *Engine) Finalize(ctx context.Context) error
```

### Memory-Bounded Indexing
- Buffer docs in memory up to 64MB threshold
- Flush segment when threshold exceeded or on explicit Finalize
- Prevents unbounded memory growth during batch indexing

## Performance Targets

| Metric | Tantivy CGO | Dahlia Target |
|--------|------------|---------------|
| Index rate | 11,808 docs/s | ≥ 10,000 docs/s |
| Search p50 | 2.7 ms | ≤ 5 ms |
| Search p95 | 3.4 ms | ≤ 8 ms |
| Peak RSS | 278 MB | ≤ 600 MB |
| Disk size | 7.3 GB | ≤ 8 GB |

## Benchmark Results

_To be filled after implementation and benchmarking._
