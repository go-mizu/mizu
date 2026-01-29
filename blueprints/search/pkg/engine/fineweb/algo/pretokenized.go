package algo

import (
	"encoding/binary"
	"io"
	"os"
	"sync"
)

// PreTokenizedDoc represents a pre-tokenized document
type PreTokenizedDoc struct {
	DocID   uint32
	Tokens  []uint64 // FNV hashes
	Freqs   []uint16 // Frequencies
	DocLen  uint16   // Number of tokens
}

// PreTokenizedFormat stores documents in a binary format optimized for fast reading
// Format:
//   Header: [4 bytes: num_docs] [4 bytes: total_tokens]
//   Per document: [4 bytes: doc_id] [2 bytes: num_tokens] [2 bytes: doc_len]
//                 [8*num_tokens bytes: hashes] [2*num_tokens bytes: freqs]
type PreTokenizedFormat struct {
	NumDocs     uint32
	TotalTokens uint64
	Docs        []PreTokenizedDoc
}

// WritePreTokenized writes pre-tokenized documents to a file
func WritePreTokenized(filename string, docs []PreTokenizedDoc) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Calculate total tokens
	var totalTokens uint64
	for _, doc := range docs {
		totalTokens += uint64(len(doc.Tokens))
	}

	// Write header
	header := make([]byte, 12)
	binary.LittleEndian.PutUint32(header[0:4], uint32(len(docs)))
	binary.LittleEndian.PutUint64(header[4:12], totalTokens)
	if _, err := f.Write(header); err != nil {
		return err
	}

	// Write documents
	for _, doc := range docs {
		// Doc header: doc_id (4) + num_tokens (2) + doc_len (2) = 8 bytes
		docHeader := make([]byte, 8)
		binary.LittleEndian.PutUint32(docHeader[0:4], doc.DocID)
		binary.LittleEndian.PutUint16(docHeader[4:6], uint16(len(doc.Tokens)))
		binary.LittleEndian.PutUint16(docHeader[6:8], doc.DocLen)
		if _, err := f.Write(docHeader); err != nil {
			return err
		}

		// Write hashes
		hashBuf := make([]byte, 8*len(doc.Tokens))
		for i, h := range doc.Tokens {
			binary.LittleEndian.PutUint64(hashBuf[i*8:], h)
		}
		if _, err := f.Write(hashBuf); err != nil {
			return err
		}

		// Write freqs
		freqBuf := make([]byte, 2*len(doc.Tokens))
		for i, freq := range doc.Freqs {
			binary.LittleEndian.PutUint16(freqBuf[i*2:], freq)
		}
		if _, err := f.Write(freqBuf); err != nil {
			return err
		}
	}

	return nil
}

// ReadPreTokenized reads pre-tokenized documents from a file
func ReadPreTokenized(filename string) (*PreTokenizedFormat, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read header
	header := make([]byte, 12)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}

	numDocs := binary.LittleEndian.Uint32(header[0:4])
	totalTokens := binary.LittleEndian.Uint64(header[4:12])

	result := &PreTokenizedFormat{
		NumDocs:     numDocs,
		TotalTokens: totalTokens,
		Docs:        make([]PreTokenizedDoc, numDocs),
	}

	// Read documents
	docHeader := make([]byte, 8)
	for i := uint32(0); i < numDocs; i++ {
		if _, err := io.ReadFull(f, docHeader); err != nil {
			return nil, err
		}

		docID := binary.LittleEndian.Uint32(docHeader[0:4])
		numTokens := binary.LittleEndian.Uint16(docHeader[4:6])
		docLen := binary.LittleEndian.Uint16(docHeader[6:8])

		// Read hashes
		hashBuf := make([]byte, 8*int(numTokens))
		if _, err := io.ReadFull(f, hashBuf); err != nil {
			return nil, err
		}
		hashes := make([]uint64, numTokens)
		for j := 0; j < int(numTokens); j++ {
			hashes[j] = binary.LittleEndian.Uint64(hashBuf[j*8:])
		}

		// Read freqs
		freqBuf := make([]byte, 2*int(numTokens))
		if _, err := io.ReadFull(f, freqBuf); err != nil {
			return nil, err
		}
		freqs := make([]uint16, numTokens)
		for j := 0; j < int(numTokens); j++ {
			freqs[j] = binary.LittleEndian.Uint16(freqBuf[j*2:])
		}

		result.Docs[i] = PreTokenizedDoc{
			DocID:  docID,
			Tokens: hashes,
			Freqs:  freqs,
			DocLen: docLen,
		}
	}

	return result, nil
}

// PreTokenizedIndexer builds index from pre-tokenized data
type PreTokenizedIndexer struct {
	shards     [256]*optimizedShard
	docLens    []uint16
	numWorkers int
	mu         sync.Mutex
}

// NewPreTokenizedIndexer creates a new indexer for pre-tokenized data
func NewPreTokenizedIndexer(numWorkers int) *PreTokenizedIndexer {
	idx := &PreTokenizedIndexer{
		docLens:    make([]uint16, 0, 4000000),
		numWorkers: numWorkers,
	}

	for i := 0; i < 256; i++ {
		idx.shards[i] = &optimizedShard{
			terms: make(map[uint64]*optimizedPostings, 10000),
		}
	}

	return idx
}

// IndexBatch indexes a batch of pre-tokenized documents
// Optimized for large scale with reduced allocations and better batching
func (idx *PreTokenizedIndexer) IndexBatch(docs []PreTokenizedDoc) {
	if len(docs) == 0 {
		return
	}

	numDocs := len(docs)
	numWorkers := idx.numWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Process in chunks to reduce memory pressure
	chunkSize := 100000
	if chunkSize > numDocs {
		chunkSize = numDocs
	}

	for chunkStart := 0; chunkStart < numDocs; chunkStart += chunkSize {
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > numDocs {
			chunkEnd = numDocs
		}
		idx.indexChunk(docs[chunkStart:chunkEnd], numWorkers)
	}
}

func (idx *PreTokenizedIndexer) indexChunk(docs []PreTokenizedDoc, numWorkers int) {
	numDocs := len(docs)
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Local shard buffers per worker - pre-allocate with estimated size
	avgTokensPerDoc := 100
	workerShards := make([][256][]optimizedPosting, numWorkers)
	for w := 0; w < numWorkers; w++ {
		docsPerWorker := (numDocs + numWorkers - 1) / numWorkers
		postingsPerShard := (docsPerWorker * avgTokensPerDoc) / 256
		if postingsPerShard < 64 {
			postingsPerShard = 64
		}
		for s := 0; s < 256; s++ {
			workerShards[w][s] = make([]optimizedPosting, 0, postingsPerShard)
		}
	}

	docLensLocal := make([]uint16, numDocs)
	batchSize := (numDocs + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup

	// Phase 1: Distribute to local shards (no locks needed)
	for w := 0; w < numWorkers; w++ {
		start := w * batchSize
		end := start + batchSize
		if end > numDocs {
			end = numDocs
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(workerID, start, end int) {
			defer wg.Done()
			myShards := &workerShards[workerID]

			for i := start; i < end; i++ {
				doc := &docs[i]
				docLensLocal[i] = doc.DocLen

				for j, hash := range doc.Tokens {
					shardID := int(hash & 0xFF)
					(*myShards)[shardID] = append((*myShards)[shardID],
						optimizedPosting{hash, doc.DocID, doc.Freqs[j]})
				}
			}
		}(w, start, end)
	}
	wg.Wait()

	// Collect doc lengths
	idx.mu.Lock()
	idx.docLens = append(idx.docLens, docLensLocal...)
	idx.mu.Unlock()

	// Phase 2: Merge into global shards - one worker per shard for no contention
	// Process shards in parallel batches
	const shardsPerBatch = 32
	for shardBatchStart := 0; shardBatchStart < 256; shardBatchStart += shardsPerBatch {
		shardBatchEnd := shardBatchStart + shardsPerBatch
		if shardBatchEnd > 256 {
			shardBatchEnd = 256
		}

		for shardID := shardBatchStart; shardID < shardBatchEnd; shardID++ {
			wg.Add(1)
			go func(shardID int) {
				defer wg.Done()
				shard := idx.shards[shardID]

				// Count total postings for this shard
				var totalPostings int
				for w := 0; w < numWorkers; w++ {
					totalPostings += len(workerShards[w][shardID])
				}
				if totalPostings == 0 {
					return
				}

				shard.mu.Lock()
				// Pre-grow maps if needed
				if len(shard.terms) == 0 {
					shard.terms = make(map[uint64]*optimizedPostings, totalPostings/2)
				}

				for w := 0; w < numWorkers; w++ {
					for _, p := range workerShards[w][shardID] {
						pl, exists := shard.terms[p.hash]
						if !exists {
							pl = &optimizedPostings{
								docIDs: make([]uint32, 0, 32),
								freqs:  make([]uint16, 0, 32),
							}
							shard.terms[p.hash] = pl
						}
						pl.docIDs = append(pl.docIDs, p.docID)
						pl.freqs = append(pl.freqs, p.freq)
					}
				}
				shard.mu.Unlock()
			}(shardID)
		}
		wg.Wait()
	}
}
