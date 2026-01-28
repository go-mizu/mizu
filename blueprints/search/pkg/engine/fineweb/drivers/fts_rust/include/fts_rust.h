#ifndef FTS_RUST_H
#define FTS_RUST_H

#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

/**
 * Maximum documents per batch for optimal memory usage
 */
#define MAX_BATCH_SIZE 10000

/**
 * Default segment size for memory-bounded indexing
 */
#define DEFAULT_SEGMENT_SIZE 100000

/**
 * Main FTS index
 */
typedef struct FtsIndex FtsIndex;

/**
 * Progress callback type
 */
typedef void (*FtsProgressFn)(uint64_t indexed, uint64_t total);

/**
 * Search hit for FFI
 */
typedef struct FtsHit {
  char *id;
  float score;
  char *text;
} FtsHit;

/**
 * Search result for FFI
 */
typedef struct FtsSearchResult {
  struct FtsHit *hits;
  uint32_t count;
  uint64_t total;
  uint64_t duration_ns;
  char *profile;
} FtsSearchResult;

/**
 * Memory statistics for FFI
 */
typedef struct FtsMemoryStats {
  uint64_t index_bytes;
  uint64_t term_dict_bytes;
  uint64_t postings_bytes;
  uint64_t docs_indexed;
  uint64_t mmap_bytes;
} FtsMemoryStats;

/**
 * Create a new index
 *
 * # Safety
 * - `data_dir` must be a valid null-terminated C string
 * - `profile` must be a valid null-terminated C string
 */
struct FtsIndex *fts_index_create(const char *data_dir, const char *profile);

/**
 * Open an existing index
 *
 * # Safety
 * - `data_dir` must be a valid null-terminated C string
 */
struct FtsIndex *fts_index_open(const char *data_dir);

/**
 * Close an index
 *
 * # Safety
 * - `idx` must be a valid pointer returned by `fts_index_create` or `fts_index_open`
 */
void fts_index_close(struct FtsIndex *idx);

/**
 * Index a batch of documents from JSON
 *
 * # Safety
 * - `idx` must be a valid index pointer
 * - `docs_json` must be a valid pointer to JSON data
 * - `docs_len` must be the length of the JSON data
 */
int64_t fts_index_batch(struct FtsIndex *idx,
                        const char *docs_json,
                        uintptr_t docs_len,
                        FtsProgressFn progress);

/**
 * Commit pending changes
 *
 * # Safety
 * - `idx` must be a valid index pointer
 */
int fts_index_commit(struct FtsIndex *idx);

/**
 * Index documents from a binary format for maximum throughput
 *
 * Binary format per document:
 *   - id_len: u32 (little-endian)
 *   - id: [u8; id_len]
 *   - text_len: u32 (little-endian)
 *   - text: [u8; text_len]
 *
 * # Safety
 * - `idx` must be a valid index pointer
 * - `data` must be valid binary data
 * - `data_len` must be the total length
 * - `doc_count` is the number of documents
 */
int64_t fts_index_batch_binary(struct FtsIndex *idx,
                               const uint8_t *data,
                               uintptr_t data_len,
                               uint64_t doc_count,
                               FtsProgressFn progress);

/**
 * Search the index
 *
 * # Safety
 * - `idx` must be a valid index pointer
 * - `query` must be a valid null-terminated C string
 * - `out` must be a valid pointer to receive the result
 */
int fts_search(struct FtsIndex *idx,
               const char *query,
               uint32_t limit,
               uint32_t offset,
               struct FtsSearchResult **out);

/**
 * Free a search result
 *
 * # Safety
 * - `result` must be a valid pointer returned by `fts_search`
 */
void fts_result_free(struct FtsSearchResult *result);

/**
 * Get memory statistics
 *
 * # Safety
 * - `idx` must be a valid index pointer
 */
struct FtsMemoryStats fts_memory_stats(struct FtsIndex *idx);

/**
 * Get the last error message
 *
 * # Safety
 * - Returns a pointer to an internal buffer, valid until next FFI call
 */
const char *fts_last_error(void);

/**
 * Get profile name
 *
 * # Safety
 * - `idx` must be a valid index pointer
 */
const char *fts_profile_name(struct FtsIndex *idx);

/**
 * List available profiles as JSON
 *
 * # Safety
 * - Returns a pointer to a static string
 */
const char *fts_list_profiles(void);

/**
 * Get document count
 *
 * # Safety
 * - `idx` must be a valid index pointer
 */
uint64_t fts_doc_count(struct FtsIndex *idx);

/**
 * Clear the index
 *
 * # Safety
 * - `idx` must be a valid index pointer
 */
void fts_index_clear(struct FtsIndex *idx);

#endif  /* FTS_RUST_H */
