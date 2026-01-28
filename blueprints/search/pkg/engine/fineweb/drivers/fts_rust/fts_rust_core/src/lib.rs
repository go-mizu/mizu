//! FTS Rust Core - High-performance full-text search engine
//!
//! This library provides multiple search algorithm profiles optimized for different use cases:
//! - `bmw_simd`: Block-Max WAND with SIMD-accelerated posting intersection
//! - `roaring_bm25`: Roaring bitmaps with BM25 scoring
//! - `ensemble`: FST + Roaring + Block-Max WAND combined
//! - `seismic`: Learned sparse retrieval with geometry-cohesive blocks

pub mod document;
pub mod tokenizer;
pub mod result;
pub mod index;
pub mod profiles;
pub mod ffi;

pub use document::Document;
pub use result::{SearchHit, SearchResult, MemoryStats};
pub use index::FtsIndex;
pub use profiles::{SearchProfile, ProfileType};

/// Library version
pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Maximum documents per batch for optimal memory usage
pub const MAX_BATCH_SIZE: usize = 10_000;

/// Default segment size for memory-bounded indexing
pub const DEFAULT_SEGMENT_SIZE: usize = 100_000;
