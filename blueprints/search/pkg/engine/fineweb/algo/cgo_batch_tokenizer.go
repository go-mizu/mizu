package algo

/*
#cgo CFLAGS: -O3 -march=native
#include <stdint.h>
#include <string.h>
#include <stdlib.h>

// Lookup table for character classification and lowercasing
static const uint8_t char_lut[256] = {
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 0-15
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 16-31
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 32-47 (space, punctuation)
    '0','1','2','3','4','5','6','7','8','9',0,0,0,0,0,0, // 48-63 (0-9)
    0,'a','b','c','d','e','f','g','h','i','j','k','l','m','n','o', // 64-79 (@,A-O)
    'p','q','r','s','t','u','v','w','x','y','z',0,0,0,0,0, // 80-95 (P-Z)
    0,'a','b','c','d','e','f','g','h','i','j','k','l','m','n','o', // 96-111 (`,a-o)
    'p','q','r','s','t','u','v','w','x','y','z',0,0,0,0,0, // 112-127 (p-z)
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 128-143
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 144-159
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 160-175
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 176-191
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 192-207
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 208-223
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 224-239
    0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0, // 240-255
};

#define FNV_OFFSET 14695981039346656037ULL
#define FNV_PRIME 1099511628211ULL

// Tokenize a single document and return hashes
// Returns the number of unique tokens (up to max_tokens)
static inline int tokenize_doc_c(
    const char* text,
    int text_len,
    uint64_t* hashes,
    uint16_t* counts,
    int max_tokens
) {
    if (text_len == 0) return 0;

    int num_unique = 0;
    int i = 0;

    while (i < text_len && num_unique < max_tokens) {
        // Skip non-alphanumeric
        while (i < text_len && char_lut[(uint8_t)text[i]] == 0) {
            i++;
        }
        if (i >= text_len) break;

        // Hash the token
        uint64_t hash = FNV_OFFSET;
        int token_start = i;

        while (i < text_len) {
            uint8_t c = char_lut[(uint8_t)text[i]];
            if (c == 0) break;
            hash = (hash ^ c) * FNV_PRIME;
            i++;
        }

        int token_len = i - token_start;
        if (token_len >= 2 && token_len <= 32) {
            // Linear probe hash table
            int slot = (int)(hash % (uint64_t)max_tokens);
            int start_slot = slot;

            while (1) {
                if (counts[slot] == 0) {
                    // Empty slot
                    hashes[slot] = hash;
                    counts[slot] = 1;
                    num_unique++;
                    break;
                } else if (hashes[slot] == hash) {
                    // Existing entry
                    if (counts[slot] < 65535) counts[slot]++;
                    break;
                }
                slot = (slot + 1) % max_tokens;
                if (slot == start_slot) break; // Table full
            }
        }
    }

    return num_unique;
}

// Batch tokenize multiple documents
// Each document is separated by null bytes in the concatenated buffer
void batch_tokenize_c(
    const char* concat_texts,      // Concatenated texts with null separators
    const int* offsets,            // Start offset of each document
    const int* lengths,            // Length of each document
    int num_docs,
    uint64_t* all_hashes,          // Output: num_docs * TABLE_SIZE hashes
    uint16_t* all_counts,          // Output: num_docs * TABLE_SIZE counts
    int* num_tokens_per_doc,       // Output: number of unique tokens per doc
    int table_size                 // Hash table size per document
) {
    for (int d = 0; d < num_docs; d++) {
        const char* text = concat_texts + offsets[d];
        int text_len = lengths[d];
        uint64_t* hashes = all_hashes + (d * table_size);
        uint16_t* counts = all_counts + (d * table_size);

        // Clear the hash table for this document
        memset(counts, 0, table_size * sizeof(uint16_t));

        num_tokens_per_doc[d] = tokenize_doc_c(text, text_len, hashes, counts, table_size);
    }
}
*/
import "C"
import (
	"runtime"
	"sync"
	"unsafe"
)

const cgoBatchTableSize = 4096

// CGOBatchTokenizer processes batches of documents using CGO
type CGOBatchTokenizer struct {
	batchSize int
}

// NewCGOBatchTokenizer creates a new batch tokenizer
func NewCGOBatchTokenizer(batchSize int) *CGOBatchTokenizer {
	if batchSize <= 0 {
		batchSize = 1000
	}
	return &CGOBatchTokenizer{batchSize: batchSize}
}

// CGOTokenResult holds tokenization results for a document
type CGOTokenResult struct {
	Hashes     []uint64
	Counts     []uint16
	NumUnique  int
}

// TokenizeBatch tokenizes a batch of documents using CGO
func (t *CGOBatchTokenizer) TokenizeBatch(texts []string) []CGOTokenResult {
	if len(texts) == 0 {
		return nil
	}

	numDocs := len(texts)

	// Calculate total size and build offsets
	totalSize := 0
	offsets := make([]C.int, numDocs)
	lengths := make([]C.int, numDocs)

	for i, text := range texts {
		offsets[i] = C.int(totalSize)
		lengths[i] = C.int(len(text))
		totalSize += len(text)
	}

	// Concatenate all texts
	concat := make([]byte, totalSize)
	pos := 0
	for _, text := range texts {
		copy(concat[pos:], text)
		pos += len(text)
	}

	// Allocate output buffers
	allHashes := make([]uint64, numDocs*cgoBatchTableSize)
	allCounts := make([]uint16, numDocs*cgoBatchTableSize)
	numTokens := make([]C.int, numDocs)

	// Pin memory for CGO
	var pinner runtime.Pinner
	pinner.Pin(&concat[0])
	pinner.Pin(&offsets[0])
	pinner.Pin(&lengths[0])
	pinner.Pin(&allHashes[0])
	pinner.Pin(&allCounts[0])
	pinner.Pin(&numTokens[0])
	defer pinner.Unpin()

	// Call CGO batch tokenizer
	C.batch_tokenize_c(
		(*C.char)(unsafe.Pointer(&concat[0])),
		(*C.int)(unsafe.Pointer(&offsets[0])),
		(*C.int)(unsafe.Pointer(&lengths[0])),
		C.int(numDocs),
		(*C.uint64_t)(unsafe.Pointer(&allHashes[0])),
		(*C.uint16_t)(unsafe.Pointer(&allCounts[0])),
		(*C.int)(unsafe.Pointer(&numTokens[0])),
		C.int(cgoBatchTableSize),
	)

	// Build results
	results := make([]CGOTokenResult, numDocs)
	for i := 0; i < numDocs; i++ {
		start := i * cgoBatchTableSize
		end := start + cgoBatchTableSize
		results[i] = CGOTokenResult{
			Hashes:    allHashes[start:end],
			Counts:    allCounts[start:end],
			NumUnique: int(numTokens[i]),
		}
	}

	return results
}

// CGOBatchIndexer uses CGO batch tokenization for high throughput
type CGOBatchIndexer struct {
	tokenizer   *CGOBatchTokenizer
	shards      [256]*optimizedShard
	numWorkers  int
	batchSize   int
}

// NewCGOBatchIndexer creates a new CGO batch indexer
func NewCGOBatchIndexer(numWorkers, batchSize int) *CGOBatchIndexer {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if batchSize <= 0 {
		batchSize = 1000
	}

	idx := &CGOBatchIndexer{
		tokenizer:  NewCGOBatchTokenizer(batchSize),
		numWorkers: numWorkers,
		batchSize:  batchSize,
	}

	for i := 0; i < 256; i++ {
		idx.shards[i] = &optimizedShard{
			terms: make(map[uint64]*optimizedPostings, 10000),
		}
	}

	return idx
}

// IndexBatch indexes documents using CGO batch tokenization
func (idx *CGOBatchIndexer) IndexBatch(texts []string, startDocID int) {
	if len(texts) == 0 {
		return
	}

	// Process in worker batches
	numDocs := len(texts)
	numWorkers := idx.numWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	batchSize := (numDocs + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup

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
		go func(workerBatch []string, workerStartDocID int) {
			defer wg.Done()

			// Tokenize batch using CGO
			results := idx.tokenizer.TokenizeBatch(workerBatch)

			// Local shard buffers
			localShards := make([][]optimizedPosting, 256)
			for s := 0; s < 256; s++ {
				localShards[s] = make([]optimizedPosting, 0, 128)
			}

			// Collect results into local shards
			for docIdx, result := range results {
				docID := uint32(workerStartDocID + docIdx)
				for slot := 0; slot < cgoBatchTableSize; slot++ {
					if result.Counts[slot] > 0 {
						hash := result.Hashes[slot]
						shardID := int(hash & 0xFF)
						localShards[shardID] = append(localShards[shardID],
							optimizedPosting{hash, docID, result.Counts[slot]})
					}
				}
			}

			// Merge local shards into global shards
			for shardID := 0; shardID < 256; shardID++ {
				if len(localShards[shardID]) == 0 {
					continue
				}
				shard := idx.shards[shardID]
				shard.mu.Lock()
				for _, p := range localShards[shardID] {
					pl, exists := shard.terms[p.hash]
					if !exists {
						pl = &optimizedPostings{
							docIDs: make([]uint32, 0, 64),
							freqs:  make([]uint16, 0, 64),
						}
						shard.terms[p.hash] = pl
					}
					pl.docIDs = append(pl.docIDs, p.docID)
					pl.freqs = append(pl.freqs, p.freq)
				}
				shard.mu.Unlock()
			}
		}(texts[start:end], startDocID+start)
	}

	wg.Wait()
}
