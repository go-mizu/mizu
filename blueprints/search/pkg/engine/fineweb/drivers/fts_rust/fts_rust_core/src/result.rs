//! Search result types

use serde::{Deserialize, Serialize};
use std::time::Duration;

/// A single search hit
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchHit {
    /// Document ID
    pub id: String,
    /// BM25 or relevance score
    pub score: f32,
    /// Optional text snippet (may be empty)
    pub text: Option<String>,
}

impl SearchHit {
    pub fn new(id: impl Into<String>, score: f32) -> Self {
        Self {
            id: id.into(),
            score,
            text: None,
        }
    }

    pub fn with_text(mut self, text: impl Into<String>) -> Self {
        self.text = Some(text.into());
        self
    }
}

/// Search result containing hits and metadata
#[derive(Debug, Clone)]
pub struct SearchResult {
    /// Ranked search hits
    pub hits: Vec<SearchHit>,
    /// Total matching documents (may be estimate)
    pub total: u64,
    /// Query execution time
    pub duration: Duration,
    /// Profile used for search
    pub profile: String,
}

impl SearchResult {
    pub fn empty(profile: impl Into<String>) -> Self {
        Self {
            hits: Vec::new(),
            total: 0,
            duration: Duration::ZERO,
            profile: profile.into(),
        }
    }
}

/// Memory usage statistics
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct MemoryStats {
    /// Total index memory (heap + mmap)
    pub index_bytes: u64,
    /// Term dictionary size
    pub term_dict_bytes: u64,
    /// Posting lists size
    pub postings_bytes: u64,
    /// Number of indexed documents
    pub docs_indexed: u64,
    /// Memory-mapped bytes (not in heap)
    pub mmap_bytes: u64,
}

impl MemoryStats {
    /// Heap-allocated memory only
    pub fn heap_bytes(&self) -> u64 {
        self.index_bytes.saturating_sub(self.mmap_bytes)
    }
}

/// Index errors
#[derive(Debug, thiserror::Error)]
pub enum IndexError {
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    #[error("Serialization error: {0}")]
    Serialization(String),

    #[error("Index corrupted: {0}")]
    Corrupted(String),

    #[error("Out of memory")]
    OutOfMemory,

    #[error("Unknown profile: {0}")]
    UnknownProfile(String),

    #[error("Index not found at path: {0}")]
    NotFound(String),
}

/// Search errors
#[derive(Debug, thiserror::Error)]
pub enum SearchError {
    #[error("Index not loaded")]
    IndexNotLoaded,

    #[error("Index not ready")]
    NotReady,

    #[error("Invalid query: {0}")]
    InvalidQuery(String),

    #[error("Internal error: {0}")]
    Internal(String),
}
