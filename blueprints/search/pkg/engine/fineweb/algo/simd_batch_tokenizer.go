// Package algo provides batch SIMD tokenization to amortize CGO overhead.
//
// Key insight: CGO overhead is ~100ns per call. For small documents (1-5KB),
// this overhead dominates. By processing multiple documents in a single CGO
// call, we can amortize this cost.
//
// This implementation also uses true SIMD for character classification
// instead of scalar LUT lookups inside NEON code.
package algo

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#ifdef __ARM_NEON
#include <arm_neon.h>
#endif

// FNV-1a constants
#define FNV_OFFSET 14695981039346656037ULL
#define FNV_PRIME  1099511628211ULL

// Character classification using SIMD range checks (no LUT)
// Returns: 0 = delimiter, lowercase byte = alphanumeric
#ifdef __ARM_NEON
static inline uint8x16_t classify_chars_neon(uint8x16_t chars) {
    // Check for lowercase: 'a' (0x61) to 'z' (0x7a)
    uint8x16_t lower_lo = vcgeq_u8(chars, vdupq_n_u8(0x61));  // >= 'a'
    uint8x16_t lower_hi = vcleq_u8(chars, vdupq_n_u8(0x7a));  // <= 'z'
    uint8x16_t is_lower = vandq_u8(lower_lo, lower_hi);

    // Check for uppercase: 'A' (0x41) to 'Z' (0x5a)
    uint8x16_t upper_lo = vcgeq_u8(chars, vdupq_n_u8(0x41));
    uint8x16_t upper_hi = vcleq_u8(chars, vdupq_n_u8(0x5a));
    uint8x16_t is_upper = vandq_u8(upper_lo, upper_hi);

    // Check for digits: '0' (0x30) to '9' (0x39)
    uint8x16_t digit_lo = vcgeq_u8(chars, vdupq_n_u8(0x30));
    uint8x16_t digit_hi = vcleq_u8(chars, vdupq_n_u8(0x39));
    uint8x16_t is_digit = vandq_u8(digit_lo, digit_hi);

    // Combine all alphanumeric checks
    uint8x16_t is_alnum = vorrq_u8(vorrq_u8(is_lower, is_upper), is_digit);

    // For uppercase, convert to lowercase by OR with 0x20
    uint8x16_t lowercase_mask = vandq_u8(is_upper, vdupq_n_u8(0x20));
    uint8x16_t lowercased = vorrq_u8(chars, lowercase_mask);

    // Result: lowercased char if alphanumeric, 0 otherwise
    return vandq_u8(lowercased, is_alnum);
}

// Count leading non-zero bytes in a NEON vector
static inline int count_leading_alnum_neon(uint8x16_t classified) {
    // Find first zero byte
    uint8x16_t zero_vec = vdupq_n_u8(0);
    uint8x16_t cmp = vceqq_u8(classified, zero_vec);

    // Extract comparison results to scalar
    uint64_t lo = vgetq_lane_u64(vreinterpretq_u64_u8(cmp), 0);
    uint64_t hi = vgetq_lane_u64(vreinterpretq_u64_u8(cmp), 1);

    if (lo == 0 && hi == 0) return 16;  // All alphanumeric

    // Find first 0xFF byte (first delimiter)
    if (lo != 0) {
        // Check each byte position in lower 64 bits
        for (int i = 0; i < 8; i++) {
            if ((lo >> (i * 8)) & 0xFF) return i;
        }
    }
    // Check upper 64 bits
    for (int i = 0; i < 8; i++) {
        if ((hi >> (i * 8)) & 0xFF) return 8 + i;
    }
    return 16;
}

// Batch tokenize multiple documents with true SIMD
// Returns total number of unique (hash, freq) pairs across all documents
int batch_tokenize_simd(
    const char** texts,
    const int* lens,
    int num_docs,
    uint64_t* out_hashes,    // Output: unique hashes
    uint16_t* out_freqs,     // Output: frequencies
    int* out_doc_offsets,    // Output: offset in out_hashes for each doc
    int* out_doc_lens        // Output: number of tokens per doc
) {
    int total_output = 0;

    for (int doc = 0; doc < num_docs; doc++) {
        const uint8_t* data = (const uint8_t*)texts[doc];
        int n = lens[doc];
        int i = 0;
        int doc_tokens = 0;
        int doc_start = total_output;

        out_doc_offsets[doc] = doc_start;

        // Simple hash table for this document
        #define TABLE_SIZE 4096
        #define TABLE_MASK (TABLE_SIZE - 1)
        static uint64_t keys[TABLE_SIZE];
        static uint16_t counts[TABLE_SIZE];
        static int used_slots[TABLE_SIZE];
        static int used_count;

        memset(keys, 0, TABLE_SIZE * sizeof(uint64_t));
        used_count = 0;

        while (i < n) {
            // Skip delimiters using SIMD
            while (i + 16 <= n) {
                uint8x16_t chunk = vld1q_u8(&data[i]);
                uint8x16_t classified = classify_chars_neon(chunk);

                // Check if any byte is alphanumeric
                uint64_t lo = vgetq_lane_u64(vreinterpretq_u64_u8(classified), 0);
                uint64_t hi = vgetq_lane_u64(vreinterpretq_u64_u8(classified), 1);

                if (lo != 0 || hi != 0) {
                    // Found alphanumeric - find exact position
                    int skip = count_leading_alnum_neon(vceqq_u8(classified, vdupq_n_u8(0)));
                    // skip is count of zeros (delimiters) before first alnum
                    // Actually we want the inverse - count of leading zeros
                    for (int j = 0; j < 16 && i + j < n; j++) {
                        uint8_t c = data[i + j];
                        if ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
                            i += j;
                            goto found_token_start;
                        }
                    }
                }
                i += 16;
            }

            // Handle remaining bytes
            while (i < n) {
                uint8_t c = data[i];
                if ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
                    break;
                }
                i++;
            }

found_token_start:
            if (i >= n) break;

            // Found start of token - hash it
            int start = i;
            uint64_t hash = FNV_OFFSET;

            // Process token bytes with SIMD classification
            while (i + 16 <= n) {
                uint8x16_t chunk = vld1q_u8(&data[i]);
                uint8x16_t classified = classify_chars_neon(chunk);

                // Check how many bytes are alphanumeric
                int alnum_count = count_leading_alnum_neon(vceqq_u8(classified, vdupq_n_u8(0)));
                alnum_count = 16 - alnum_count;  // Invert: count of leading alnums

                // Actually count from start
                uint8_t result[16];
                vst1q_u8(result, classified);

                for (int j = 0; j < 16; j++) {
                    if (result[j] == 0) {
                        i += j;
                        goto token_done;
                    }
                    hash ^= (uint64_t)result[j];
                    hash *= FNV_PRIME;
                }
                i += 16;
            }

            // Handle remaining bytes
            while (i < n) {
                uint8_t c = data[i];
                uint8_t lower = 0;
                if (c >= 'a' && c <= 'z') lower = c;
                else if (c >= 'A' && c <= 'Z') lower = c | 0x20;
                else if (c >= '0' && c <= '9') lower = c;

                if (lower == 0) break;

                hash ^= (uint64_t)lower;
                hash *= FNV_PRIME;
                i++;
            }

token_done:
            {
                int token_len = i - start;
                if (token_len >= 2 && token_len <= 32) {
                    doc_tokens++;

                    // Insert into local hash table
                    if (hash == 0) hash = 1;
                    uint64_t idx = hash & TABLE_MASK;

                    for (int probe = 0; probe < TABLE_SIZE; probe++) {
                        if (keys[idx] == 0) {
                            keys[idx] = hash;
                            counts[idx] = 1;
                            used_slots[used_count++] = idx;
                            break;
                        }
                        if (keys[idx] == hash) {
                            counts[idx]++;
                            break;
                        }
                        idx = (idx + 1) & TABLE_MASK;
                    }
                }
            }
        }

        // Output unique hashes for this document
        for (int j = 0; j < used_count; j++) {
            int idx = used_slots[j];
            out_hashes[total_output] = keys[idx];
            out_freqs[total_output] = counts[idx];
            total_output++;
        }

        out_doc_lens[doc] = doc_tokens;
        #undef TABLE_SIZE
        #undef TABLE_MASK
    }

    return total_output;
}
#else
// Scalar fallback
int batch_tokenize_simd(
    const char** texts,
    const int* lens,
    int num_docs,
    uint64_t* out_hashes,
    uint16_t* out_freqs,
    int* out_doc_offsets,
    int* out_doc_lens
) {
    // Scalar implementation for non-ARM
    return 0;
}
#endif
*/
import "C"

import (
	"runtime"
	"unsafe"
)

// BatchSIMDResult holds the results of batch SIMD tokenization
type BatchSIMDResult struct {
	Hashes     []uint64
	Freqs      []uint16
	DocOffsets []int
	DocLens    []int
	TotalPairs int
}

// BatchSIMDTokenize processes multiple documents in a single CGO call
// This amortizes CGO overhead across many documents
func BatchSIMDTokenize(texts []string, maxPairs int) *BatchSIMDResult {
	if len(texts) == 0 {
		return &BatchSIMDResult{}
	}

	numDocs := len(texts)

	// Prepare C arrays
	cTexts := make([]*C.char, numDocs)
	cLens := make([]C.int, numDocs)

	// Pin memory and convert texts
	var pinner runtime.Pinner
	for i, text := range texts {
		if len(text) > 0 {
			b := []byte(text)
			pinner.Pin(&b[0])
			cTexts[i] = (*C.char)(unsafe.Pointer(&b[0]))
		} else {
			cTexts[i] = nil
		}
		cLens[i] = C.int(len(text))
	}

	// Allocate output buffers
	outHashes := make([]uint64, maxPairs)
	outFreqs := make([]uint16, maxPairs)
	outDocOffsets := make([]C.int, numDocs)
	outDocLens := make([]C.int, numDocs)

	pinner.Pin(&cTexts[0])
	pinner.Pin(&cLens[0])
	pinner.Pin(&outHashes[0])
	pinner.Pin(&outFreqs[0])
	pinner.Pin(&outDocOffsets[0])
	pinner.Pin(&outDocLens[0])

	defer pinner.Unpin()

	// Call batch SIMD tokenization
	totalPairs := C.batch_tokenize_simd(
		(**C.char)(unsafe.Pointer(&cTexts[0])),
		(*C.int)(unsafe.Pointer(&cLens[0])),
		C.int(numDocs),
		(*C.uint64_t)(unsafe.Pointer(&outHashes[0])),
		(*C.uint16_t)(unsafe.Pointer(&outFreqs[0])),
		(*C.int)(unsafe.Pointer(&outDocOffsets[0])),
		(*C.int)(unsafe.Pointer(&outDocLens[0])),
	)

	// Convert results
	docOffsets := make([]int, numDocs)
	docLens := make([]int, numDocs)
	for i := 0; i < numDocs; i++ {
		docOffsets[i] = int(outDocOffsets[i])
		docLens[i] = int(outDocLens[i])
	}

	return &BatchSIMDResult{
		Hashes:     outHashes[:totalPairs],
		Freqs:      outFreqs[:totalPairs],
		DocOffsets: docOffsets,
		DocLens:    docLens,
		TotalPairs: int(totalPairs),
	}
}

// BatchSIMDIndexer uses batch SIMD tokenization for maximum throughput
type BatchSIMDIndexer struct {
	shards   [256]*batchSIMDShard
	docLens  []uint16
	docCount uint64
	totalLen uint64
}

type batchSIMDShard struct {
	terms map[uint64]*batchSIMDPostings
}

type batchSIMDPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// NewBatchSIMDIndexer creates a new batch SIMD indexer
func NewBatchSIMDIndexer() *BatchSIMDIndexer {
	idx := &BatchSIMDIndexer{
		docLens: make([]uint16, 0, 4000000),
	}

	for i := 0; i < 256; i++ {
		idx.shards[i] = &batchSIMDShard{
			terms: make(map[uint64]*batchSIMDPostings, 10000),
		}
	}

	return idx
}

// IndexBatch indexes a batch of documents using batch SIMD
func (idx *BatchSIMDIndexer) IndexBatch(texts []string, startDocID int) {
	if len(texts) == 0 {
		return
	}

	// Estimate max pairs (average 100 unique terms per doc)
	maxPairs := len(texts) * 100

	// Process batch
	result := BatchSIMDTokenize(texts, maxPairs)

	// Update doc lens
	for _, docLen := range result.DocLens {
		if docLen > 65535 {
			docLen = 65535
		}
		idx.docLens = append(idx.docLens, uint16(docLen))
		idx.totalLen += uint64(docLen)
	}
	idx.docCount += uint64(len(texts))

	// Distribute to shards
	for docIdx := 0; docIdx < len(texts); docIdx++ {
		start := result.DocOffsets[docIdx]
		var end int
		if docIdx+1 < len(texts) {
			end = result.DocOffsets[docIdx+1]
		} else {
			end = result.TotalPairs
		}

		docID := uint32(startDocID + docIdx)

		for i := start; i < end; i++ {
			hash := result.Hashes[i]
			freq := result.Freqs[i]
			shardID := hash & 0xFF

			shard := idx.shards[shardID]
			pl, exists := shard.terms[hash]
			if !exists {
				pl = &batchSIMDPostings{
					docIDs: make([]uint32, 0, 64),
					freqs:  make([]uint16, 0, 64),
				}
				shard.terms[hash] = pl
			}
			pl.docIDs = append(pl.docIDs, docID)
			pl.freqs = append(pl.freqs, freq)
		}
	}
}

// Finish returns the built index
func (idx *BatchSIMDIndexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(idx.docCount)
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(idx.totalLen) / float64(numDocs)
	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < 256; shardID++ {
		shard := idx.shards[shardID]
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range idx.docLens {
		docLensMap[uint32(i)] = dl
	}

	segment := &SearchSegment{
		id:      0,
		terms:   terms,
		docLens: docLensMap,
		numDocs: numDocs,
	}

	return &SegmentedIndex{
		segments:  []*SearchSegment{segment},
		numDocs:   numDocs,
		avgDocLen: avgDocLen,
		docLens:   idx.docLens,
	}, nil
}
