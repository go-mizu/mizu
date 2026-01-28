//! C FFI interface for Go integration

use crate::document::Document;
use crate::index::FtsIndex;

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int};
use std::ptr;
use std::slice;
use std::sync::Mutex;

/// Thread-local last error message
static LAST_ERROR: Mutex<Option<String>> = Mutex::new(None);
static ERROR_STRING: Mutex<Option<CString>> = Mutex::new(None);

fn set_last_error(msg: impl Into<String>) {
    *LAST_ERROR.lock().unwrap() = Some(msg.into());
}

/// Search hit for FFI
#[repr(C)]
pub struct FtsHit {
    pub id: *mut c_char,
    pub score: f32,
    pub text: *mut c_char,
}

/// Search result for FFI
#[repr(C)]
pub struct FtsSearchResult {
    pub hits: *mut FtsHit,
    pub count: u32,
    pub total: u64,
    pub duration_ns: u64,
    pub profile: *mut c_char,
}

/// Memory statistics for FFI
#[repr(C)]
pub struct FtsMemoryStats {
    pub index_bytes: u64,
    pub term_dict_bytes: u64,
    pub postings_bytes: u64,
    pub docs_indexed: u64,
    pub mmap_bytes: u64,
}

/// Progress callback type
pub type FtsProgressFn = Option<extern "C" fn(indexed: u64, total: u64)>;

/// Create a new index
///
/// # Safety
/// - `data_dir` must be a valid null-terminated C string
/// - `profile` must be a valid null-terminated C string
#[no_mangle]
pub unsafe extern "C" fn fts_index_create(
    data_dir: *const c_char,
    profile: *const c_char,
) -> *mut FtsIndex {
    if data_dir.is_null() || profile.is_null() {
        set_last_error("Null pointer passed to fts_index_create");
        return ptr::null_mut();
    }

    let data_dir = match CStr::from_ptr(data_dir).to_str() {
        Ok(s) => s,
        Err(_) => {
            set_last_error("Invalid UTF-8 in data_dir");
            return ptr::null_mut();
        }
    };

    let profile = match CStr::from_ptr(profile).to_str() {
        Ok(s) => s,
        Err(_) => {
            set_last_error("Invalid UTF-8 in profile");
            return ptr::null_mut();
        }
    };

    match FtsIndex::create(data_dir, profile) {
        Ok(index) => Box::into_raw(Box::new(index)),
        Err(e) => {
            set_last_error(e.to_string());
            ptr::null_mut()
        }
    }
}

/// Open an existing index
///
/// # Safety
/// - `data_dir` must be a valid null-terminated C string
#[no_mangle]
pub unsafe extern "C" fn fts_index_open(data_dir: *const c_char) -> *mut FtsIndex {
    if data_dir.is_null() {
        set_last_error("Null pointer passed to fts_index_open");
        return ptr::null_mut();
    }

    let data_dir = match CStr::from_ptr(data_dir).to_str() {
        Ok(s) => s,
        Err(_) => {
            set_last_error("Invalid UTF-8 in data_dir");
            return ptr::null_mut();
        }
    };

    match FtsIndex::open(data_dir) {
        Ok(index) => Box::into_raw(Box::new(index)),
        Err(e) => {
            set_last_error(e.to_string());
            ptr::null_mut()
        }
    }
}

/// Close an index
///
/// # Safety
/// - `idx` must be a valid pointer returned by `fts_index_create` or `fts_index_open`
#[no_mangle]
pub unsafe extern "C" fn fts_index_close(idx: *mut FtsIndex) {
    if !idx.is_null() {
        drop(Box::from_raw(idx));
    }
}

/// Index a batch of documents from JSON
///
/// # Safety
/// - `idx` must be a valid index pointer
/// - `docs_json` must be a valid pointer to JSON data
/// - `docs_len` must be the length of the JSON data
#[no_mangle]
pub unsafe extern "C" fn fts_index_batch(
    idx: *mut FtsIndex,
    docs_json: *const c_char,
    docs_len: usize,
    progress: FtsProgressFn,
) -> i64 {
    if idx.is_null() || docs_json.is_null() {
        set_last_error("Null pointer passed to fts_index_batch");
        return -1;
    }

    let index = &*idx;
    let json_slice = slice::from_raw_parts(docs_json as *const u8, docs_len);
    let json_str = match std::str::from_utf8(json_slice) {
        Ok(s) => s,
        Err(_) => {
            set_last_error("Invalid UTF-8 in docs_json");
            return -2;
        }
    };

    // Parse JSON array of documents
    let docs: Vec<Document> = match serde_json::from_str(json_str) {
        Ok(d) => d,
        Err(e) => {
            set_last_error(format!("JSON parse error: {}", e));
            return -2;
        }
    };

    let total = docs.len() as u64;

    // Report initial progress
    if let Some(cb) = progress {
        cb(0, total);
    }

    // Index in chunks for progress reporting
    const CHUNK_SIZE: usize = 1000;
    let mut indexed = 0u64;

    for chunk in docs.chunks(CHUNK_SIZE) {
        match index.index_batch(chunk) {
            Ok(n) => {
                indexed += n as u64;
                if let Some(cb) = progress {
                    cb(indexed, total);
                }
            }
            Err(e) => {
                set_last_error(e.to_string());
                return -3;
            }
        }
    }

    indexed as i64
}

/// Commit pending changes
///
/// # Safety
/// - `idx` must be a valid index pointer
#[no_mangle]
pub unsafe extern "C" fn fts_index_commit(idx: *mut FtsIndex) -> c_int {
    if idx.is_null() {
        set_last_error("Null pointer passed to fts_index_commit");
        return -1;
    }

    let index = &*idx;
    match index.commit() {
        Ok(()) => 0,
        Err(e) => {
            set_last_error(e.to_string());
            -1
        }
    }
}

/// Index documents from a binary format for maximum throughput
///
/// Binary format per document:
///   - id_len: u32 (little-endian)
///   - id: [u8; id_len]
///   - text_len: u32 (little-endian)
///   - text: [u8; text_len]
///
/// # Safety
/// - `idx` must be a valid index pointer
/// - `data` must be valid binary data
/// - `data_len` must be the total length
/// - `doc_count` is the number of documents
#[no_mangle]
pub unsafe extern "C" fn fts_index_batch_binary(
    idx: *mut FtsIndex,
    data: *const u8,
    data_len: usize,
    doc_count: u64,
    progress: FtsProgressFn,
) -> i64 {
    if idx.is_null() || data.is_null() {
        set_last_error("Null pointer passed to fts_index_batch_binary");
        return -1;
    }

    let index = &*idx;
    let bytes = slice::from_raw_parts(data, data_len);

    // Parse binary format into documents
    let mut docs = Vec::with_capacity(doc_count as usize);
    let mut pos = 0;

    while pos + 8 <= bytes.len() && docs.len() < doc_count as usize {
        // Read id
        if pos + 4 > bytes.len() {
            break;
        }
        let id_len =
            u32::from_le_bytes([bytes[pos], bytes[pos + 1], bytes[pos + 2], bytes[pos + 3]])
                as usize;
        pos += 4;

        if pos + id_len > bytes.len() {
            break;
        }
        let id = match std::str::from_utf8(&bytes[pos..pos + id_len]) {
            Ok(s) => s.to_string(),
            Err(_) => {
                set_last_error("Invalid UTF-8 in document ID");
                return -2;
            }
        };
        pos += id_len;

        // Read text
        if pos + 4 > bytes.len() {
            break;
        }
        let text_len =
            u32::from_le_bytes([bytes[pos], bytes[pos + 1], bytes[pos + 2], bytes[pos + 3]])
                as usize;
        pos += 4;

        if pos + text_len > bytes.len() {
            break;
        }
        let text = match std::str::from_utf8(&bytes[pos..pos + text_len]) {
            Ok(s) => s.to_string(),
            Err(_) => {
                set_last_error("Invalid UTF-8 in document text");
                return -2;
            }
        };
        pos += text_len;

        docs.push(Document::new(id, text));
    }

    let total = docs.len() as u64;

    if let Some(cb) = progress {
        cb(0, total);
    }

    // Index all documents at once for maximum throughput
    match index.index_batch(&docs) {
        Ok(n) => {
            if let Some(cb) = progress {
                cb(n as u64, total);
            }
            n as i64
        }
        Err(e) => {
            set_last_error(e.to_string());
            -3
        }
    }
}

/// Search the index
///
/// # Safety
/// - `idx` must be a valid index pointer
/// - `query` must be a valid null-terminated C string
/// - `out` must be a valid pointer to receive the result
#[no_mangle]
pub unsafe extern "C" fn fts_search(
    idx: *mut FtsIndex,
    query: *const c_char,
    limit: u32,
    offset: u32,
    out: *mut *mut FtsSearchResult,
) -> c_int {
    if idx.is_null() || query.is_null() || out.is_null() {
        set_last_error("Null pointer passed to fts_search");
        return -1;
    }

    let index = &*idx;
    let query_str = match CStr::from_ptr(query).to_str() {
        Ok(s) => s,
        Err(_) => {
            set_last_error("Invalid UTF-8 in query");
            return -2;
        }
    };

    let result = match index.search(query_str, limit as usize, offset as usize) {
        Ok(r) => r,
        Err(e) => {
            set_last_error(e.to_string());
            return -3;
        }
    };

    // Allocate hits array
    let hits: Vec<FtsHit> = result
        .hits
        .into_iter()
        .map(|hit| FtsHit {
            id: CString::new(hit.id).unwrap().into_raw(),
            score: hit.score,
            text: hit
                .text
                .map(|t| CString::new(t).unwrap().into_raw())
                .unwrap_or(ptr::null_mut()),
        })
        .collect();

    let count = hits.len() as u32;

    // Convert to raw pointer
    let hits_ptr = if hits.is_empty() {
        ptr::null_mut()
    } else {
        let boxed = hits.into_boxed_slice();
        Box::into_raw(boxed) as *mut FtsHit
    };

    let search_result = Box::new(FtsSearchResult {
        hits: hits_ptr,
        count,
        total: result.total,
        duration_ns: result.duration.as_nanos() as u64,
        profile: CString::new(result.profile).unwrap().into_raw(),
    });

    *out = Box::into_raw(search_result);
    0
}

/// Free a search result
///
/// # Safety
/// - `result` must be a valid pointer returned by `fts_search`
#[no_mangle]
pub unsafe extern "C" fn fts_result_free(result: *mut FtsSearchResult) {
    if result.is_null() {
        return;
    }

    let result = Box::from_raw(result);

    // Free hits
    if !result.hits.is_null() {
        let hits = slice::from_raw_parts_mut(result.hits, result.count as usize);
        for hit in hits {
            if !hit.id.is_null() {
                drop(CString::from_raw(hit.id));
            }
            if !hit.text.is_null() {
                drop(CString::from_raw(hit.text));
            }
        }
        drop(Box::from_raw(result.hits));
    }

    // Free profile string
    if !result.profile.is_null() {
        drop(CString::from_raw(result.profile));
    }
}

/// Get memory statistics
///
/// # Safety
/// - `idx` must be a valid index pointer
#[no_mangle]
pub unsafe extern "C" fn fts_memory_stats(idx: *mut FtsIndex) -> FtsMemoryStats {
    if idx.is_null() {
        return FtsMemoryStats {
            index_bytes: 0,
            term_dict_bytes: 0,
            postings_bytes: 0,
            docs_indexed: 0,
            mmap_bytes: 0,
        };
    }

    let index = &*idx;
    let stats = index.memory_stats();

    FtsMemoryStats {
        index_bytes: stats.index_bytes,
        term_dict_bytes: stats.term_dict_bytes,
        postings_bytes: stats.postings_bytes,
        docs_indexed: stats.docs_indexed,
        mmap_bytes: stats.mmap_bytes,
    }
}

/// Get the last error message
///
/// # Safety
/// - Returns a pointer to an internal buffer, valid until next FFI call
#[no_mangle]
pub extern "C" fn fts_last_error() -> *const c_char {
    let error = LAST_ERROR.lock().unwrap();
    let msg = error.as_deref().unwrap_or("No error");

    let cstring = CString::new(msg).unwrap_or_else(|_| CString::new("Unknown error").unwrap());
    let mut buffer = ERROR_STRING.lock().unwrap();
    *buffer = Some(cstring);
    buffer.as_ref().map(|s| s.as_ptr()).unwrap_or(ptr::null())
}

/// Get profile name
///
/// # Safety
/// - `idx` must be a valid index pointer
#[no_mangle]
pub unsafe extern "C" fn fts_profile_name(idx: *mut FtsIndex) -> *const c_char {
    if idx.is_null() {
        return ptr::null();
    }

    let index = &*idx;
    let name = index.profile_name();

    // Return static string pointer using C string literals
    match name {
        "bmw_simd" => c"bmw_simd".as_ptr(),
        "roaring_bm25" => c"roaring_bm25".as_ptr(),
        "ensemble" => c"ensemble".as_ptr(),
        "seismic" => c"seismic".as_ptr(),
        "tantivy" => c"tantivy".as_ptr(),
        "turbo" => c"turbo".as_ptr(),
        "ultra" => c"ultra".as_ptr(),
        _ => c"unknown".as_ptr(),
    }
}

/// List available profiles as JSON
///
/// # Safety
/// - Returns a pointer to a static string
#[no_mangle]
pub extern "C" fn fts_list_profiles() -> *const c_char {
    c"[\"bmw_simd\",\"roaring_bm25\",\"ensemble\",\"seismic\",\"tantivy\",\"turbo\",\"ultra\"]"
        .as_ptr()
}

/// Get document count
///
/// # Safety
/// - `idx` must be a valid index pointer
#[no_mangle]
pub unsafe extern "C" fn fts_doc_count(idx: *mut FtsIndex) -> u64 {
    if idx.is_null() {
        return 0;
    }

    let index = &*idx;
    index.doc_count()
}

/// Clear the index
///
/// # Safety
/// - `idx` must be a valid index pointer
#[no_mangle]
pub unsafe extern "C" fn fts_index_clear(idx: *mut FtsIndex) {
    if !idx.is_null() {
        let index = &*idx;
        index.clear();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ffi::CString;
    use tempfile::tempdir;

    #[test]
    fn test_ffi_lifecycle() {
        let dir = tempdir().unwrap();
        let data_dir = CString::new(dir.path().to_str().unwrap()).unwrap();
        let profile = CString::new("ensemble").unwrap();

        unsafe {
            // Create index
            let idx = fts_index_create(data_dir.as_ptr(), profile.as_ptr());
            assert!(!idx.is_null());

            // Index documents
            let docs_json = r#"[{"id":"1","text":"hello world"},{"id":"2","text":"world peace"}]"#;
            let result = fts_index_batch(
                idx,
                docs_json.as_ptr() as *const c_char,
                docs_json.len(),
                None,
            );
            assert_eq!(result, 2);

            // Commit
            assert_eq!(fts_index_commit(idx), 0);

            // Search
            let query = CString::new("hello").unwrap();
            let mut search_result: *mut FtsSearchResult = ptr::null_mut();
            let status = fts_search(idx, query.as_ptr(), 10, 0, &mut search_result);
            assert_eq!(status, 0);
            assert!(!search_result.is_null());
            assert_eq!((*search_result).count, 1);

            fts_result_free(search_result);

            // Memory stats
            let stats = fts_memory_stats(idx);
            assert_eq!(stats.docs_indexed, 2);

            // Close
            fts_index_close(idx);
        }
    }
}
