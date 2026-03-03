// Package dahlia implements a pure-Go full-text search engine modeled after
// tantivy's segment-based architecture with BP128 compression, FST term
// dictionaries, Block-Max WAND scoring, and position-aware phrase queries.
package dahlia

const (
	blockSize          = 128            // BP128 block size
	storeBlockSize     = 16 * 1024      // 16KB store blocks
	skipEntrySize      = 21             // bytes per skip entry
	noMoreDocs         = ^uint32(0)     // sentinel: no more documents
	metaFile           = "dahlia.meta"  // index-level metadata
	segMetaFile        = "segment.meta" // per-segment metadata
	segTermDictFile    = "segment.tdi"  // FST term dictionary
	segDocFile         = "segment.doc"  // doc ID posting lists
	segFreqFile        = "segment.freq" // term frequency lists
	segPosFile         = "segment.pos"  // position data
	segStoreFile       = "segment.store"
	segFieldNormFile   = "segment.fnm"
	memoryFlushBytes   = 8 * 1024 * 1024  // 8MB flush threshold (low RSS)
	indexMaxTextBytes  = 512              // max text bytes for indexing
	storeMaxTextBytes  = 256              // max text bytes for storage (snippets)
	maxSegBeforeMerge  = 10
	maxMergeSegments   = 10
	segDirFmt          = "seg_%08d"
)
