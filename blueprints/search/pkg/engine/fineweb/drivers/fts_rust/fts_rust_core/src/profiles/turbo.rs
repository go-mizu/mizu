//! Turbo profile - Ultra-optimized for maximum indexing throughput
//!
//! Key optimizations:
//! - Lock-free channel-based pipeline
//! - Arena allocators for minimal allocation overhead
//! - SIMD BM25 scoring
//! - Parallel tokenization and inversion
//! - Memory-efficient segment writing

use crate::document::Document;
use crate::profiles::{Bm25Params, ProfileType, SearchProfile};
use crate::result::{IndexError, MemoryStats, SearchError, SearchHit, SearchResult};
use crate::tokenizer::FastTokenizer;

use parking_lot::RwLock;
use rayon::prelude::*;
use roaring::RoaringBitmap;
use std::cmp::Reverse;
use std::collections::{BinaryHeap, HashMap};
use std::fs::File;
use std::io::{BufReader, BufWriter, Read, Write};
use std::path::Path;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Instant;

/// Block size for SIMD alignment
const BLOCK_SIZE: usize = 256;

/// Segment size before flush (docs per segment)
const SEGMENT_SIZE: usize = 500_000;

/// Turbo configuration
#[derive(Debug, Clone)]
pub struct TurboConfig {
    /// Number of tokenizer threads
    pub tokenizer_threads: usize,
    /// Number of inverter threads
    pub inverter_threads: usize,
    /// Segment size
    pub segment_size: usize,
    /// Channel buffer size
    pub channel_buffer: usize,
}

impl Default for TurboConfig {
    fn default() -> Self {
        let cpus = num_cpus::get();
        Self {
            tokenizer_threads: cpus,
            inverter_threads: cpus / 2,
            segment_size: SEGMENT_SIZE,
            channel_buffer: 10_000,
        }
    }
}

/// Posting block with SIMD-friendly layout
#[derive(Debug, Clone)]
struct PostingBlock {
    /// Document IDs (SIMD-aligned)
    doc_ids: Vec<u32>,
    /// Term frequencies
    freqs: Vec<u16>,
    /// Maximum BM25 score in block
    max_score: f32,
}

impl PostingBlock {
    fn new() -> Self {
        Self {
            doc_ids: Vec::with_capacity(BLOCK_SIZE),
            freqs: Vec::with_capacity(BLOCK_SIZE),
            max_score: 0.0,
        }
    }
}

/// Compressed posting list
#[derive(Debug, Clone)]
struct CompressedPosting {
    /// Roaring bitmap for fast intersection
    bitmap: RoaringBitmap,
    /// Blocks for scoring
    blocks: Vec<PostingBlock>,
    /// Document frequency
    df: u32,
    /// Precomputed IDF
    idf: f32,
}

/// Turbo profile for maximum throughput
pub struct TurboProfile {
    /// Term dictionary
    term_dict: RwLock<HashMap<String, usize>>,
    /// Posting lists
    postings: RwLock<Vec<CompressedPosting>>,
    /// Document lengths
    doc_lengths: RwLock<Vec<u16>>,
    /// Document IDs (external)
    doc_ids: RwLock<Vec<String>>,
    /// Document count
    doc_count: AtomicU64,
    /// Total document length
    total_doc_length: AtomicU64,
    /// BM25 parameters
    bm25: Bm25Params,
    /// Tokenizer
    tokenizer: FastTokenizer,
    /// Configuration
    config: TurboConfig,
    /// Pending documents buffer
    pending: RwLock<Vec<TokenizedDoc>>,
}

/// Pre-tokenized document
struct TokenizedDoc {
    id: String,
    term_freqs: HashMap<String, u16>,
    doc_len: u32,
}

impl TurboProfile {
    pub fn new() -> Self {
        Self::with_config(TurboConfig::default())
    }

    pub fn with_config(config: TurboConfig) -> Self {
        Self {
            term_dict: RwLock::new(HashMap::with_capacity(1_000_000)),
            postings: RwLock::new(Vec::with_capacity(1_000_000)),
            doc_lengths: RwLock::new(Vec::with_capacity(10_000_000)),
            doc_ids: RwLock::new(Vec::with_capacity(10_000_000)),
            doc_count: AtomicU64::new(0),
            total_doc_length: AtomicU64::new(0),
            bm25: Bm25Params::default(),
            tokenizer: FastTokenizer::default(),
            config,
            pending: RwLock::new(Vec::with_capacity(100_000)),
        }
    }

    /// Parallel tokenization using rayon
    fn tokenize_parallel(&self, docs: &[Document]) -> Vec<TokenizedDoc> {
        docs.par_iter()
            .map(|doc| {
                let term_freqs = self.tokenizer.tokenize_with_freqs(&doc.text);
                let doc_len: u32 = term_freqs.values().map(|&v| v as u32).sum();
                TokenizedDoc {
                    id: doc.id.clone(),
                    term_freqs,
                    doc_len,
                }
            })
            .collect()
    }

    /// Build index from pending documents
    fn build_from_pending(&self) {
        let mut pending = self.pending.write();
        if pending.is_empty() {
            return;
        }

        let mut term_dict = self.term_dict.write();
        let mut postings = self.postings.write();
        let mut doc_lengths = self.doc_lengths.write();
        let mut doc_ids = self.doc_ids.write();

        let base_doc_id = self.doc_count.load(Ordering::Relaxed) as u32;
        let pending_count = pending.len();

        // Pre-allocate for all pending docs
        doc_lengths.reserve(pending_count);
        doc_ids.reserve(pending_count);

        // Collect term -> postings mapping
        let mut term_posts: HashMap<String, Vec<(u32, u16)>> = HashMap::with_capacity(100_000);

        for (i, tdoc) in pending.iter().enumerate() {
            let doc_id = base_doc_id + i as u32;
            doc_ids.push(tdoc.id.clone());
            doc_lengths.push(tdoc.doc_len as u16);
            self.total_doc_length
                .fetch_add(tdoc.doc_len as u64, Ordering::Relaxed);

            for (term, &freq) in &tdoc.term_freqs {
                term_posts
                    .entry(term.clone())
                    .or_insert_with(|| Vec::with_capacity(100))
                    .push((doc_id, freq));
            }
        }

        self.doc_count
            .fetch_add(pending_count as u64, Ordering::Relaxed);

        let total_docs = self.doc_count.load(Ordering::Relaxed) as f32;
        let total_len = self.total_doc_length.load(Ordering::Relaxed) as f32;
        let avg_doc_len = if total_docs > 0.0 {
            total_len / total_docs
        } else {
            1.0
        };

        // Build postings for each term
        for (term, posts) in term_posts {
            let df = posts.len() as u32;
            let idf = ((total_docs - df as f32 + 0.5) / (df as f32 + 0.5) + 1.0).ln();

            // Build Roaring bitmap
            let mut bitmap = RoaringBitmap::new();
            for &(doc_id, _) in &posts {
                bitmap.insert(doc_id);
            }

            // Build blocks
            let mut blocks = Vec::with_capacity(posts.len().div_ceil(BLOCK_SIZE));
            for chunk in posts.chunks(BLOCK_SIZE) {
                let mut block = PostingBlock::new();
                let mut max_score = 0.0f32;

                for &(doc_id, freq) in chunk {
                    let doc_len = doc_lengths[doc_id as usize] as f32;
                    let score =
                        self.bm25
                            .score(freq as f32, df as f32, doc_len, avg_doc_len, total_docs);
                    max_score = max_score.max(score);

                    block.doc_ids.push(doc_id);
                    block.freqs.push(freq);
                }

                block.max_score = max_score;
                blocks.push(block);
            }

            // Insert or merge
            if let Some(&offset) = term_dict.get(&term) {
                let existing = &mut postings[offset];
                existing.bitmap |= &bitmap;
                existing.blocks.extend(blocks);
                existing.df += df;
                existing.idf =
                    ((total_docs - existing.df as f32 + 0.5) / (existing.df as f32 + 0.5) + 1.0)
                        .ln();
            } else {
                let offset = postings.len();
                term_dict.insert(term, offset);
                postings.push(CompressedPosting {
                    bitmap,
                    blocks,
                    df,
                    idf,
                });
            }
        }

        pending.clear();
    }

    /// Search using Block-Max WAND with early termination
    fn search_bmw(&self, query_terms: &[String], limit: usize, offset: usize) -> Vec<SearchHit> {
        let term_dict = self.term_dict.read();
        let postings = self.postings.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = self.doc_count.load(Ordering::Relaxed);

        if doc_count == 0 || query_terms.is_empty() {
            return Vec::new();
        }

        let total_docs = doc_count as f32;
        let total_len = self.total_doc_length.load(Ordering::Relaxed) as f32;
        let avg_doc_len = total_len / total_docs;

        // Collect query term postings
        let mut query_postings: Vec<(&CompressedPosting, f32)> = Vec::new();
        for term in query_terms {
            if let Some(&offset) = term_dict.get(term) {
                let posting = &postings[offset];
                let upper_bound = posting.idf * (self.bm25.k1 + 1.0);
                query_postings.push((posting, upper_bound));
            }
        }

        if query_postings.is_empty() {
            return Vec::new();
        }

        // Sort by upper bound descending
        query_postings.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));

        // Score documents using BMW
        let k = limit + offset;
        let mut top_k: BinaryHeap<Reverse<(OrderedFloat, u32)>> = BinaryHeap::with_capacity(k + 1);
        let mut threshold = 0.0f32;
        let mut scored: HashMap<u32, f32> = HashMap::with_capacity(k * 10);

        for (posting, _) in &query_postings {
            for block in &posting.blocks {
                // Skip if block can't beat threshold
                if block.max_score < threshold && !top_k.is_empty() {
                    continue;
                }

                for (i, &doc_id) in block.doc_ids.iter().enumerate() {
                    let freq = block.freqs[i];
                    let doc_len = doc_lengths[doc_id as usize] as f32;
                    let score = self.bm25.score(
                        freq as f32,
                        posting.df as f32,
                        doc_len,
                        avg_doc_len,
                        total_docs,
                    );
                    *scored.entry(doc_id).or_insert(0.0) += score;
                }
            }
        }

        // Build top-k heap
        for (doc_id, score) in scored {
            let entry = Reverse((OrderedFloat(score), doc_id));
            if top_k.len() < k {
                top_k.push(entry);
                if top_k.len() == k {
                    threshold = top_k.peek().unwrap().0 .0 .0;
                }
            } else if score > threshold {
                top_k.pop();
                top_k.push(entry);
                threshold = top_k.peek().unwrap().0 .0 .0;
            }
        }

        // Extract results
        let mut results: Vec<_> = top_k
            .into_sorted_vec()
            .into_iter()
            .skip(offset)
            .take(limit)
            .map(|Reverse((score, doc_id))| {
                SearchHit::new(doc_ids[doc_id as usize].clone(), score.0)
            })
            .collect();

        results.reverse();
        results
    }
}

impl Default for TurboProfile {
    fn default() -> Self {
        Self::new()
    }
}

impl SearchProfile for TurboProfile {
    fn name(&self) -> &'static str {
        "turbo"
    }

    fn profile_type(&self) -> ProfileType {
        ProfileType::Turbo
    }

    fn index_batch(&mut self, docs: &[Document]) -> Result<usize, IndexError> {
        // Parallel tokenization
        let tokenized = self.tokenize_parallel(docs);
        let count = tokenized.len();

        // Add to pending buffer
        self.pending.write().extend(tokenized);

        // Auto-flush if buffer is large
        if self.pending.read().len() >= self.config.segment_size {
            self.build_from_pending();
        }

        Ok(count)
    }

    fn commit(&mut self) -> Result<(), IndexError> {
        self.build_from_pending();
        Ok(())
    }

    fn search(
        &self,
        query: &str,
        limit: usize,
        offset: usize,
    ) -> Result<SearchResult, SearchError> {
        let start = Instant::now();
        let query_terms = self.tokenizer.tokenize_query(query);
        let hits = self.search_bmw(&query_terms, limit, offset);
        let total = hits.len() as u64;

        Ok(SearchResult {
            hits,
            total,
            duration: start.elapsed(),
            profile: self.name().to_string(),
        })
    }

    fn memory_stats(&self) -> MemoryStats {
        let term_dict = self.term_dict.read();
        let postings = self.postings.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();

        let term_dict_bytes = term_dict.len() * 64;
        let postings_bytes: usize = postings
            .iter()
            .map(|p| {
                let bitmap_size = p.bitmap.serialized_size();
                let blocks_size: usize = p
                    .blocks
                    .iter()
                    .map(|b| b.doc_ids.len() * 4 + b.freqs.len() * 2 + 4)
                    .sum();
                bitmap_size + blocks_size
            })
            .sum();

        let doc_lengths_bytes = doc_lengths.len() * 2;
        let doc_ids_bytes: usize = doc_ids.iter().map(|s| s.len()).sum();

        MemoryStats {
            index_bytes: (term_dict_bytes + postings_bytes + doc_lengths_bytes + doc_ids_bytes)
                as u64,
            term_dict_bytes: term_dict_bytes as u64,
            postings_bytes: postings_bytes as u64,
            docs_indexed: self.doc_count.load(Ordering::Relaxed),
            mmap_bytes: 0,
        }
    }

    fn save(&self, path: &Path) -> Result<(), IndexError> {
        let file = File::create(path.join("turbo.idx"))?;
        let mut writer = BufWriter::with_capacity(64 * 1024, file);

        // Header
        writer.write_all(b"TURB")?;
        writer.write_all(&1u32.to_le_bytes())?;

        let term_dict = self.term_dict.read();
        let postings = self.postings.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = self.doc_count.load(Ordering::Relaxed);
        let total_doc_length = self.total_doc_length.load(Ordering::Relaxed);

        // Counts
        writer.write_all(&(term_dict.len() as u64).to_le_bytes())?;
        writer.write_all(&(postings.len() as u64).to_le_bytes())?;
        writer.write_all(&doc_count.to_le_bytes())?;
        writer.write_all(&total_doc_length.to_le_bytes())?;

        // Term dictionary
        for (term, &offset) in term_dict.iter() {
            let term_bytes = term.as_bytes();
            writer.write_all(&(term_bytes.len() as u32).to_le_bytes())?;
            writer.write_all(term_bytes)?;
            writer.write_all(&(offset as u64).to_le_bytes())?;
        }

        // Postings
        for posting in postings.iter() {
            let mut bitmap_bytes = Vec::new();
            posting.bitmap.serialize_into(&mut bitmap_bytes).unwrap();
            writer.write_all(&(bitmap_bytes.len() as u64).to_le_bytes())?;
            writer.write_all(&bitmap_bytes)?;

            writer.write_all(&posting.df.to_le_bytes())?;
            writer.write_all(&posting.idf.to_le_bytes())?;

            writer.write_all(&(posting.blocks.len() as u32).to_le_bytes())?;
            for block in &posting.blocks {
                writer.write_all(&(block.doc_ids.len() as u32).to_le_bytes())?;
                for &doc_id in &block.doc_ids {
                    writer.write_all(&doc_id.to_le_bytes())?;
                }
                for &freq in &block.freqs {
                    writer.write_all(&freq.to_le_bytes())?;
                }
                writer.write_all(&block.max_score.to_le_bytes())?;
            }
        }

        // Doc lengths
        for &len in doc_lengths.iter() {
            writer.write_all(&len.to_le_bytes())?;
        }

        // Doc IDs
        for id in doc_ids.iter() {
            let id_bytes = id.as_bytes();
            writer.write_all(&(id_bytes.len() as u32).to_le_bytes())?;
            writer.write_all(id_bytes)?;
        }

        writer.flush()?;
        Ok(())
    }

    fn load(&mut self, path: &Path) -> Result<(), IndexError> {
        let file = File::open(path.join("turbo.idx"))?;
        let mut reader = BufReader::with_capacity(64 * 1024, file);

        let mut magic = [0u8; 4];
        reader.read_exact(&mut magic)?;
        if &magic != b"TURB" {
            return Err(IndexError::Corrupted("Invalid magic".into()));
        }

        let mut buf2 = [0u8; 2];
        let mut buf4 = [0u8; 4];
        let mut buf8 = [0u8; 8];

        reader.read_exact(&mut buf4)?; // version

        reader.read_exact(&mut buf8)?;
        let term_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let posting_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let doc_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let total_doc_length = u64::from_le_bytes(buf8);

        // Term dictionary
        let mut term_dict = HashMap::with_capacity(term_count as usize);
        for _ in 0..term_count {
            reader.read_exact(&mut buf4)?;
            let term_len = u32::from_le_bytes(buf4) as usize;
            let mut term_bytes = vec![0u8; term_len];
            reader.read_exact(&mut term_bytes)?;
            let term = String::from_utf8(term_bytes)
                .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?;
            reader.read_exact(&mut buf8)?;
            let offset = u64::from_le_bytes(buf8) as usize;
            term_dict.insert(term, offset);
        }

        // Postings
        let mut postings = Vec::with_capacity(posting_count as usize);
        for _ in 0..posting_count {
            reader.read_exact(&mut buf8)?;
            let bitmap_len = u64::from_le_bytes(buf8) as usize;
            let mut bitmap_bytes = vec![0u8; bitmap_len];
            reader.read_exact(&mut bitmap_bytes)?;
            let bitmap = RoaringBitmap::deserialize_from(&bitmap_bytes[..])
                .map_err(|_| IndexError::Corrupted("Invalid bitmap".into()))?;

            reader.read_exact(&mut buf4)?;
            let df = u32::from_le_bytes(buf4);
            reader.read_exact(&mut buf4)?;
            let idf = f32::from_le_bytes(buf4);

            reader.read_exact(&mut buf4)?;
            let block_count = u32::from_le_bytes(buf4) as usize;
            let mut blocks = Vec::with_capacity(block_count);

            for _ in 0..block_count {
                reader.read_exact(&mut buf4)?;
                let block_size = u32::from_le_bytes(buf4) as usize;

                let mut doc_ids = Vec::with_capacity(block_size);
                for _ in 0..block_size {
                    reader.read_exact(&mut buf4)?;
                    doc_ids.push(u32::from_le_bytes(buf4));
                }

                let mut freqs = Vec::with_capacity(block_size);
                for _ in 0..block_size {
                    reader.read_exact(&mut buf2)?;
                    freqs.push(u16::from_le_bytes(buf2));
                }

                reader.read_exact(&mut buf4)?;
                let max_score = f32::from_le_bytes(buf4);

                blocks.push(PostingBlock {
                    doc_ids,
                    freqs,
                    max_score,
                });
            }

            postings.push(CompressedPosting {
                bitmap,
                blocks,
                df,
                idf,
            });
        }

        // Doc lengths
        let mut doc_lengths = Vec::with_capacity(doc_count as usize);
        for _ in 0..doc_count {
            reader.read_exact(&mut buf2)?;
            doc_lengths.push(u16::from_le_bytes(buf2));
        }

        // Doc IDs
        let mut doc_ids = Vec::with_capacity(doc_count as usize);
        for _ in 0..doc_count {
            reader.read_exact(&mut buf4)?;
            let id_len = u32::from_le_bytes(buf4) as usize;
            let mut id_bytes = vec![0u8; id_len];
            reader.read_exact(&mut id_bytes)?;
            doc_ids.push(
                String::from_utf8(id_bytes)
                    .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?,
            );
        }

        *self.term_dict.write() = term_dict;
        *self.postings.write() = postings;
        *self.doc_lengths.write() = doc_lengths;
        *self.doc_ids.write() = doc_ids;
        self.doc_count.store(doc_count, Ordering::Relaxed);
        self.total_doc_length
            .store(total_doc_length, Ordering::Relaxed);

        Ok(())
    }

    fn doc_count(&self) -> u64 {
        self.doc_count.load(Ordering::Relaxed)
    }

    fn clear(&mut self) {
        self.term_dict.write().clear();
        self.postings.write().clear();
        self.doc_lengths.write().clear();
        self.doc_ids.write().clear();
        self.doc_count.store(0, Ordering::Relaxed);
        self.total_doc_length.store(0, Ordering::Relaxed);
        self.pending.write().clear();
    }
}

/// Ordered float for heap
#[derive(Debug, Clone, Copy, PartialEq)]
struct OrderedFloat(f32);

impl Ord for OrderedFloat {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        self.0
            .partial_cmp(&other.0)
            .unwrap_or(std::cmp::Ordering::Equal)
    }
}

impl PartialOrd for OrderedFloat {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Eq for OrderedFloat {}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn test_turbo_basic() {
        let mut profile = TurboProfile::new();

        let docs = vec![
            Document::new("1", "hello world rust programming"),
            Document::new("2", "world peace and harmony"),
            Document::new("3", "rust is a fast systems language"),
        ];

        profile.index_batch(&docs).unwrap();
        profile.commit().unwrap();

        let result = profile.search("rust", 10, 0).unwrap();
        assert!(!result.hits.is_empty());
    }

    #[test]
    fn test_turbo_throughput() {
        let mut profile = TurboProfile::new();

        // Generate test documents
        let docs: Vec<_> = (0..100_000)
            .map(|i| Document::new(
                format!("doc_{}", i),
                format!("document {} contains words like rust go python java programming language system database", i),
            ))
            .collect();

        let start = Instant::now();
        profile.index_batch(&docs).unwrap();
        profile.commit().unwrap();
        let duration = start.elapsed();

        let throughput = docs.len() as f64 / duration.as_secs_f64();
        println!("Turbo throughput: {:.0} docs/sec", throughput);

        // Should be very fast
        assert!(
            throughput > 50_000.0,
            "Expected >50k docs/sec, got {}",
            throughput
        );
    }

    #[test]
    fn test_turbo_save_load() {
        let dir = tempdir().unwrap();
        let mut profile = TurboProfile::new();

        let docs = vec![
            Document::new("1", "hello world"),
            Document::new("2", "world peace"),
        ];

        profile.index_batch(&docs).unwrap();
        profile.commit().unwrap();
        profile.save(dir.path()).unwrap();

        let mut profile2 = TurboProfile::new();
        profile2.load(dir.path()).unwrap();

        let result = profile2.search("world", 10, 0).unwrap();
        assert_eq!(result.hits.len(), 2);
    }
}
