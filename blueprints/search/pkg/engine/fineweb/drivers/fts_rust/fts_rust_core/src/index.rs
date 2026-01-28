//! FtsIndex - Main index wrapper

use crate::document::Document;
use crate::profiles::{create_profile, ProfileType, SearchProfile};
use crate::result::{IndexError, MemoryStats, SearchError, SearchResult};

use parking_lot::RwLock;
use std::path::{Path, PathBuf};
use std::sync::Arc;

/// Main FTS index
pub struct FtsIndex {
    /// Data directory
    data_dir: PathBuf,
    /// Current profile
    profile: RwLock<Box<dyn SearchProfile>>,
    /// Profile type
    profile_type: ProfileType,
}

impl FtsIndex {
    /// Create a new index with the specified profile
    pub fn create(data_dir: impl AsRef<Path>, profile_name: &str) -> Result<Self, IndexError> {
        let data_dir = data_dir.as_ref().to_path_buf();
        std::fs::create_dir_all(&data_dir)?;

        let profile_type = ProfileType::from_str(profile_name)
            .ok_or_else(|| IndexError::UnknownProfile(profile_name.to_string()))?;

        let mut profile = create_profile(profile_type);

        // Initialize profile with data directory
        // This is needed for profiles like Tantivy that write to disk immediately
        profile.init(&data_dir)?;

        Ok(Self {
            data_dir,
            profile: RwLock::new(profile),
            profile_type,
        })
    }

    /// Open an existing index
    pub fn open(data_dir: impl AsRef<Path>) -> Result<Self, IndexError> {
        let data_dir = data_dir.as_ref().to_path_buf();
        if !data_dir.exists() {
            return Err(IndexError::NotFound(data_dir.display().to_string()));
        }

        // Try to detect profile from existing files
        let profile_type = Self::detect_profile(&data_dir)?;
        let mut profile = create_profile(profile_type);
        profile.load(&data_dir)?;

        Ok(Self {
            data_dir,
            profile: RwLock::new(profile),
            profile_type,
        })
    }

    /// Detect profile type from index files
    fn detect_profile(data_dir: &Path) -> Result<ProfileType, IndexError> {
        if data_dir.join("bmw_simd.idx").exists() {
            Ok(ProfileType::BmwSimd)
        } else if data_dir.join("roaring_bm25.idx").exists() {
            Ok(ProfileType::RoaringBm25)
        } else if data_dir.join("ensemble.idx").exists() {
            Ok(ProfileType::Ensemble)
        } else if data_dir.join("seismic.idx").exists() {
            Ok(ProfileType::Seismic)
        } else if data_dir.join("tantivy_index").exists() || data_dir.join("tantivy.idx").exists() {
            Ok(ProfileType::Tantivy)
        } else if data_dir.join("turbo.idx").exists() {
            Ok(ProfileType::Turbo)
        } else if data_dir.join("ultra.idx").exists() {
            Ok(ProfileType::Ultra)
        } else {
            // Default to ultra for best throughput
            Ok(ProfileType::Ultra)
        }
    }

    /// Get profile name
    pub fn profile_name(&self) -> &'static str {
        self.profile_type.as_str()
    }

    /// Get profile type
    pub fn profile_type(&self) -> ProfileType {
        self.profile_type
    }

    /// Index a batch of documents
    pub fn index_batch(&self, docs: &[Document]) -> Result<usize, IndexError> {
        self.profile.write().index_batch(docs)
    }

    /// Commit pending changes
    pub fn commit(&self) -> Result<(), IndexError> {
        let mut profile = self.profile.write();
        profile.commit()?;
        profile.save(&self.data_dir)?;
        Ok(())
    }

    /// Search the index
    pub fn search(&self, query: &str, limit: usize, offset: usize) -> Result<SearchResult, SearchError> {
        self.profile.read().search(query, limit, offset)
    }

    /// Get memory statistics
    pub fn memory_stats(&self) -> MemoryStats {
        self.profile.read().memory_stats()
    }

    /// Get document count
    pub fn doc_count(&self) -> u64 {
        self.profile.read().doc_count()
    }

    /// Clear the index
    pub fn clear(&self) {
        self.profile.write().clear();
    }

    /// Get data directory
    pub fn data_dir(&self) -> &Path {
        &self.data_dir
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn test_index_lifecycle() {
        let dir = tempdir().unwrap();

        // Create index
        let index = FtsIndex::create(dir.path(), "ensemble").unwrap();
        let docs = vec![
            Document::new("1", "hello world"),
            Document::new("2", "world peace"),
        ];

        index.index_batch(&docs).unwrap();
        index.commit().unwrap();

        // Search
        let result = index.search("hello", 10, 0).unwrap();
        assert_eq!(result.hits.len(), 1);

        // Reopen
        drop(index);
        let index = FtsIndex::open(dir.path()).unwrap();
        let result = index.search("world", 10, 0).unwrap();
        assert_eq!(result.hits.len(), 2);
    }

    #[test]
    fn test_all_profiles() {
        for profile in ProfileType::all() {
            let dir = tempdir().unwrap();
            let index = FtsIndex::create(dir.path(), profile.as_str()).unwrap();

            let docs = vec![
                Document::new("1", "test document one"),
                Document::new("2", "test document two"),
            ];

            index.index_batch(&docs).unwrap();
            index.commit().unwrap();

            let result = index.search("test", 10, 0).unwrap();
            assert!(result.hits.len() >= 1, "Profile {} failed", profile.as_str());
        }
    }
}
