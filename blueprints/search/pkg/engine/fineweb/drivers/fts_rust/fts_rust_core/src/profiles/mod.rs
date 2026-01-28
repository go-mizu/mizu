//! Search profile trait and implementations

mod bmw_simd;
mod roaring_bm25;
mod ensemble;
mod seismic;
mod tantivy_profile;
mod turbo;
mod ultra;

pub use bmw_simd::BmwSimdProfile;
pub use roaring_bm25::RoaringBm25Profile;
pub use ensemble::EnsembleProfile;
pub use seismic::SeismicProfile;
pub use tantivy_profile::TantivyProfile;
pub use turbo::TurboProfile;
pub use ultra::UltraProfile;

use crate::document::Document;
use crate::result::{IndexError, MemoryStats, SearchError, SearchResult};
use std::path::Path;

/// Available profile types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProfileType {
    BmwSimd,
    RoaringBm25,
    Ensemble,
    Seismic,
    Tantivy,
    Turbo,
    Ultra,
}

impl ProfileType {
    pub fn from_str(s: &str) -> Option<Self> {
        match s.to_lowercase().as_str() {
            "bmw_simd" | "bmwsimd" => Some(Self::BmwSimd),
            "roaring_bm25" | "roaringbm25" | "roaring" => Some(Self::RoaringBm25),
            "ensemble" | "default" => Some(Self::Ensemble),
            "seismic" => Some(Self::Seismic),
            "tantivy" => Some(Self::Tantivy),
            "turbo" => Some(Self::Turbo),
            "ultra" => Some(Self::Ultra),
            _ => None,
        }
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Self::BmwSimd => "bmw_simd",
            Self::RoaringBm25 => "roaring_bm25",
            Self::Ensemble => "ensemble",
            Self::Seismic => "seismic",
            Self::Tantivy => "tantivy",
            Self::Turbo => "turbo",
            Self::Ultra => "ultra",
        }
    }

    /// List all available profiles
    pub fn all() -> &'static [ProfileType] {
        &[
            Self::BmwSimd,
            Self::RoaringBm25,
            Self::Ensemble,
            Self::Seismic,
            Self::Tantivy,
            Self::Turbo,
            Self::Ultra,
        ]
    }
}

/// Core trait for search profiles
pub trait SearchProfile: Send + Sync {
    /// Profile name
    fn name(&self) -> &'static str;

    /// Profile type
    fn profile_type(&self) -> ProfileType;

    /// Initialize the profile with a data directory
    /// Called when creating a new index
    fn init(&mut self, data_dir: &Path) -> Result<(), IndexError> {
        // Default implementation does nothing
        let _ = data_dir;
        Ok(())
    }

    /// Index a batch of documents
    fn index_batch(&mut self, docs: &[Document]) -> Result<usize, IndexError>;

    /// Commit pending changes to disk
    fn commit(&mut self) -> Result<(), IndexError>;

    /// Search the index
    fn search(
        &self,
        query: &str,
        limit: usize,
        offset: usize,
    ) -> Result<SearchResult, SearchError>;

    /// Get memory statistics
    fn memory_stats(&self) -> MemoryStats;

    /// Save index to disk
    fn save(&self, path: &Path) -> Result<(), IndexError>;

    /// Load index from disk
    fn load(&mut self, path: &Path) -> Result<(), IndexError>;

    /// Number of indexed documents
    fn doc_count(&self) -> u64;

    /// Clear the index
    fn clear(&mut self);
}

/// Create a profile by type
pub fn create_profile(profile_type: ProfileType) -> Box<dyn SearchProfile> {
    match profile_type {
        ProfileType::BmwSimd => Box::new(BmwSimdProfile::new()),
        ProfileType::RoaringBm25 => Box::new(RoaringBm25Profile::new()),
        ProfileType::Ensemble => Box::new(EnsembleProfile::new()),
        ProfileType::Seismic => Box::new(SeismicProfile::new()),
        ProfileType::Tantivy => Box::new(TantivyProfile::new()),
        ProfileType::Turbo => Box::new(TurboProfile::new()),
        ProfileType::Ultra => Box::new(UltraProfile::new()),
    }
}

/// BM25 scoring parameters
#[derive(Debug, Clone, Copy)]
pub struct Bm25Params {
    pub k1: f32,
    pub b: f32,
}

impl Default for Bm25Params {
    fn default() -> Self {
        Self { k1: 1.2, b: 0.75 }
    }
}

impl Bm25Params {
    /// Calculate BM25 score for a term
    #[inline]
    pub fn score(&self, tf: f32, df: f32, doc_len: f32, avg_doc_len: f32, total_docs: f32) -> f32 {
        let idf = ((total_docs - df + 0.5) / (df + 0.5) + 1.0).ln();
        let tf_component = (tf * (self.k1 + 1.0))
            / (tf + self.k1 * (1.0 - self.b + self.b * doc_len / avg_doc_len));
        idf * tf_component
    }
}
