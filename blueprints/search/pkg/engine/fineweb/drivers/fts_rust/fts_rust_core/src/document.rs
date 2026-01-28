//! Document types for indexing

use serde::{Deserialize, Serialize};

/// A document to be indexed
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Document {
    /// Unique document identifier
    pub id: String,
    /// Text content to be indexed
    pub text: String,
    /// Optional URL
    #[serde(default)]
    pub url: Option<String>,
    /// Optional metadata
    #[serde(default)]
    pub metadata: Option<DocumentMetadata>,
}

impl Document {
    pub fn new(id: impl Into<String>, text: impl Into<String>) -> Self {
        Self {
            id: id.into(),
            text: text.into(),
            url: None,
            metadata: None,
        }
    }

    pub fn with_url(mut self, url: impl Into<String>) -> Self {
        self.url = Some(url.into());
        self
    }
}

/// Optional document metadata
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DocumentMetadata {
    pub language: Option<String>,
    pub date: Option<String>,
    pub source: Option<String>,
}

/// Internal representation after tokenization
#[derive(Debug, Clone)]
pub struct IndexedDocument {
    pub doc_id: u32,
    pub external_id: String,
    pub term_freqs: Vec<(String, u16)>,
    pub doc_length: u32,
}
