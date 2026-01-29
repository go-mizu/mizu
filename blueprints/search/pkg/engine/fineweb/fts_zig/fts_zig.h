/* fts_zig.h - C header for fts_zig FFI */
#ifndef FTS_ZIG_H
#define FTS_ZIG_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Opaque handle to an index */
typedef void* fts_handle_t;

/* Search result */
typedef struct {
    uint32_t doc_id;
    float score;
} fts_search_result_t;

/* Index statistics */
typedef struct {
    uint32_t doc_count;
    uint32_t term_count;
    uint64_t memory_bytes;
} fts_stats_t;

/* Error codes */
typedef enum {
    FTS_OK = 0,
    FTS_ERR_INVALID_HANDLE = -1,
    FTS_ERR_ALLOCATION_FAILED = -2,
    FTS_ERR_IO_ERROR = -3,
    FTS_ERR_INVALID_ARGUMENT = -4,
    FTS_ERR_NOT_FOUND = -5
} fts_error_t;

/* ============================================================================
 * Speed Profile (raw arrays, no compression, <1ms p99)
 * ============================================================================ */

/* Create a new speed index builder */
fts_handle_t fts_speed_builder_create(void);

/* Add a document to the speed index builder */
int fts_speed_builder_add(fts_handle_t handle, const char* text, size_t text_len);

/* Build the speed index from builder */
fts_handle_t fts_speed_builder_build(fts_handle_t handle);

/* Destroy a speed index builder */
void fts_speed_builder_destroy(fts_handle_t handle);

/* Search the speed index
 * Returns: number of results written to results array */
int fts_speed_search(fts_handle_t handle, const char* query, size_t query_len,
                     fts_search_result_t* results, size_t max_results);

/* Get speed index statistics */
void fts_speed_stats(fts_handle_t handle, fts_stats_t* stats);

/* Destroy a speed index */
void fts_speed_destroy(fts_handle_t handle);

/* ============================================================================
 * Balanced Profile (Block-Max WAND + VByte, 1-10ms p99)
 * ============================================================================ */

/* Create a new balanced index builder */
fts_handle_t fts_balanced_builder_create(void);

/* Add a document to the balanced index builder */
int fts_balanced_builder_add(fts_handle_t handle, const char* text, size_t text_len);

/* Build the balanced index from builder */
fts_handle_t fts_balanced_builder_build(fts_handle_t handle);

/* Destroy a balanced index builder */
void fts_balanced_builder_destroy(fts_handle_t handle);

/* Search the balanced index */
int fts_balanced_search(fts_handle_t handle, const char* query, size_t query_len,
                        fts_search_result_t* results, size_t max_results);

/* Destroy a balanced index */
void fts_balanced_destroy(fts_handle_t handle);

/* ============================================================================
 * Compact Profile (Elias-Fano, 10-50ms p99)
 * ============================================================================ */

/* Create a new compact index builder */
fts_handle_t fts_compact_builder_create(void);

/* Add a document to the compact index builder */
int fts_compact_builder_add(fts_handle_t handle, const char* text, size_t text_len);

/* Build the compact index from builder */
fts_handle_t fts_compact_builder_build(fts_handle_t handle);

/* Destroy a compact index builder */
void fts_compact_builder_destroy(fts_handle_t handle);

/* Search the compact index */
int fts_compact_search(fts_handle_t handle, const char* query, size_t query_len,
                       fts_search_result_t* results, size_t max_results);

/* Destroy a compact index */
void fts_compact_destroy(fts_handle_t handle);

/* ============================================================================
 * Utility Functions
 * ============================================================================ */

/* Get library version string */
const char* fts_version(void);

/* Hash a string (for debugging) */
uint64_t fts_hash(const char* text, size_t text_len);

#ifdef __cplusplus
}
#endif

#endif /* FTS_ZIG_H */
