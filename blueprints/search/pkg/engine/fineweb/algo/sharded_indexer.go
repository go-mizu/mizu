// Package algo provides sharded pipeline indexing for maximum throughput.
// Target: 500k+ docs/sec with <5GB memory.
package algo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// ShardedIndexer implements high-throughput sharded parallel indexing.
// Each worker maintains its own segment, avoiding lock contention entirely.
//
// Architecture:
//   [Batch Input] → [Parallel Sharded Workers] → [Local Segments] → [Merge]
//
// Each worker:
// - Receives batches of documents
// - Tokenizes and indexes locally (no locks)
// - Flushes to its own segment files
type ShardedIndexer struct {
	// Configuration
	NumShards    int           // Number of parallel shards (= workers)
	SegmentSize  int           // Docs per segment before flush
	OutputDir    string        // Segment output directory
	Tokenizer    TokenizerFunc // text → term frequencies

	// Sharded state
	shards []*indexShard

	// Global state
	segments  []*SegmentMeta
	segmentMu sync.Mutex
	segmentID atomic.Int32

	// Metrics
	docsProcessed atomic.Int64
}

// indexShard is an independent indexing unit with no shared state.
type indexShard struct {
	id        int
	indexer   *ShardedIndexer
	docCh     chan shardDoc       // Input channel for this shard
	segment   *shardSegmentBuilder
	wg        sync.WaitGroup
}

type shardDoc struct {
	docID uint32
	text  string
}

type shardSegmentBuilder struct {
	id           int
	shardID      int
	termPostings map[string][]IndexPosting
	docLens      map[uint32]int
	numDocs      int
}

// NewShardedIndexer creates a high-throughput sharded indexer.
func NewShardedIndexer(outputDir string, tokenizer TokenizerFunc) *ShardedIndexer {
	numShards := runtime.NumCPU()
	if numShards < 4 {
		numShards = 4
	}
	if numShards > 16 {
		numShards = 16
	}

	si := &ShardedIndexer{
		NumShards:   numShards,
		SegmentSize: 100000, // 100k docs per segment - match PipelineIndexer
		OutputDir:   outputDir,
		Tokenizer:   tokenizer,
		segments:    make([]*SegmentMeta, 0, 128),
	}

	// Create output directory
	os.MkdirAll(outputDir, 0755)

	// Initialize shards
	si.shards = make([]*indexShard, numShards)
	for i := 0; i < numShards; i++ {
		si.shards[i] = &indexShard{
			id:      i,
			indexer: si,
			docCh:   make(chan shardDoc, 10000), // Large buffer per shard
		}
		si.shards[i].segment = si.shards[i].newSegment()
		si.shards[i].start()
	}

	return si
}

func (s *indexShard) newSegment() *shardSegmentBuilder {
	id := int(s.indexer.segmentID.Add(1)) - 1
	return &shardSegmentBuilder{
		id:           id,
		shardID:      s.id,
		termPostings: make(map[string][]IndexPosting, 30000),
		docLens:      make(map[uint32]int, s.indexer.SegmentSize),
	}
}

func (s *indexShard) start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		for doc := range s.docCh {
			// Tokenize
			termFreqs := s.indexer.Tokenizer(doc.text)

			docLen := 0
			for _, freq := range termFreqs {
				docLen += freq
			}

			// Add to local segment
			s.segment.docLens[doc.docID] = docLen
			for term, freq := range termFreqs {
				s.segment.termPostings[term] = append(
					s.segment.termPostings[term],
					IndexPosting{DocID: doc.docID, Freq: uint16(freq)},
				)
			}
			s.segment.numDocs++

			// Flush segment when full
			if s.segment.numDocs >= s.indexer.SegmentSize {
				s.flushSegment()
			}
		}

		// Flush final partial segment
		if s.segment.numDocs > 0 {
			s.flushSegment()
		}
	}()
}

func (s *indexShard) flushSegment() {
	meta := s.writeSegment(s.segment)
	if meta != nil {
		s.indexer.segmentMu.Lock()
		s.indexer.segments = append(s.indexer.segments, meta)
		s.indexer.segmentMu.Unlock()
	}

	// Clear segment memory
	s.segment.termPostings = nil
	s.segment.docLens = nil
	runtime.GC()

	// Create new segment
	s.segment = s.newSegment()
}

func (s *indexShard) writeSegment(seg *shardSegmentBuilder) *SegmentMeta {
	if seg.numDocs == 0 {
		return nil
	}

	path := filepath.Join(s.indexer.OutputDir, fmt.Sprintf("seg_%05d_s%d.bin", seg.id, seg.shardID))

	f, err := os.Create(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 4*1024*1024) // 4MB buffer
	fw := NewFastBinaryWriter(w)

	// Write header
	fw.WriteUint32(uint32(seg.numDocs))
	fw.WriteUint32(uint32(len(seg.termPostings)))

	// Collect and sort terms
	terms := make([]string, 0, len(seg.termPostings))
	for term := range seg.termPostings {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	// Write term dictionary
	termOffsets := make(map[string]int64, len(terms))
	offset := int64(0)
	for _, term := range terms {
		postings := seg.termPostings[term]
		termOffsets[term] = offset
		offset += int64(len(postings) * 6)
	}

	for _, term := range terms {
		termBytes := []byte(term)
		fw.WriteUint16(uint16(len(termBytes)))
		fw.WriteBytes(termBytes)
		fw.WriteUint32(uint32(len(seg.termPostings[term])))
		fw.WriteInt64(termOffsets[term])
	}

	// Write posting lists using batched writer
	batchWriter := NewBatchPostingWriter(w, 64*1024)
	for _, term := range terms {
		postings := seg.termPostings[term]
		sort.Slice(postings, func(i, j int) bool {
			return postings[i].DocID < postings[j].DocID
		})
		for _, p := range postings {
			batchWriter.WritePosting(p.DocID, p.Freq)
		}
	}
	batchWriter.Flush()

	// Write doc lengths
	docIDs := make([]uint32, 0, len(seg.docLens))
	for docID := range seg.docLens {
		docIDs = append(docIDs, docID)
	}
	sort.Slice(docIDs, func(i, j int) bool { return docIDs[i] < docIDs[j] })

	fw.WriteUint32(uint32(len(docIDs)))
	for _, docID := range docIDs {
		fw.WriteUint32(docID)
		fw.WriteUint16(uint16(seg.docLens[docID]))
	}

	w.Flush()

	fi, _ := f.Stat()
	return &SegmentMeta{
		ID:       seg.id,
		Path:     path,
		NumDocs:  seg.numDocs,
		NumTerms: len(terms),
		Size:     fi.Size(),
	}
}

// Add adds a document to be indexed.
// Documents are distributed across shards in round-robin fashion.
func (si *ShardedIndexer) Add(docID uint32, text string) {
	// Round-robin distribution based on docID
	shardID := int(docID) % si.NumShards
	si.shards[shardID].docCh <- shardDoc{docID: docID, text: text}
	si.docsProcessed.Add(1)
}

// Finish waits for all shards to complete and returns segment paths.
func (si *ShardedIndexer) Finish() []string {
	// Close all shard channels
	for _, shard := range si.shards {
		close(shard.docCh)
	}

	// Wait for all shards to finish
	for _, shard := range si.shards {
		shard.wg.Wait()
	}

	// Return segment paths
	paths := make([]string, len(si.segments))
	for i, meta := range si.segments {
		paths[i] = meta.Path
	}
	return paths
}

// FinishToMmap merges all segments into a single mmap index.
func (si *ShardedIndexer) FinishToMmap(outputPath string) (*MmapIndex, error) {
	paths := si.Finish()

	if len(paths) == 0 {
		return nil, fmt.Errorf("no segments to merge")
	}

	// Use streaming merger
	merger := NewTrueStreamingMerger(outputPath, paths)
	if err := merger.Merge(); err != nil {
		return nil, err
	}

	// Clean up segment files
	for _, path := range paths {
		os.Remove(path)
	}

	return OpenMmapIndex(outputPath)
}

// DocCount returns the number of processed documents.
func (si *ShardedIndexer) DocCount() int64 {
	return si.docsProcessed.Load()
}
