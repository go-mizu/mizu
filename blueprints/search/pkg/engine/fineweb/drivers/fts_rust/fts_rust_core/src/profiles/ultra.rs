//! Ultra profile - Maximum throughput optimizations
//!
//! Target: 1M+ docs/sec indexing throughput
//!
//! Key optimizations:
//! - Massive parallel processing with rayon
//! - Lock-free data structures where possible
//! - Pre-allocated buffers
//! - Minimal memory allocations
//! - Simplified tokenization (ASCII-only)
//! - In-memory only (no disk I/O during indexing)

use crate::document::Document;
use crate::profiles::{Bm25Params, ProfileType, SearchProfile};
use crate::result::{IndexError, MemoryStats, SearchError, SearchHit, SearchResult};

use parking_lot::RwLock;
use rayon::prelude::*;
use std::cmp::Reverse;
use std::collections::{BinaryHeap, HashMap};
use std::fs::File;
use std::io::{BufReader, BufWriter, Read, Write};
use std::path::Path;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Instant;

/// Block size for efficient scoring
const BLOCK_SIZE: usize = 512;

/// Compact posting entry
#[derive(Debug, Clone, Copy)]
struct PostingEntry {
    doc_id: u32,
    freq: u16,
}

/// Posting list with block-max scores
#[derive(Debug, Clone)]
struct PostingList {
    entries: Vec<PostingEntry>,
    block_maxes: Vec<f32>,
    df: u32,
    idf: f32,
}

impl PostingList {
    fn new() -> Self {
        Self {
            entries: Vec::new(),
            block_maxes: Vec::new(),
            df: 0,
            idf: 0.0,
        }
    }

    fn compute_block_maxes(
        &mut self,
        doc_lengths: &[u16],
        avg_doc_len: f32,
        total_docs: f32,
        bm25: &Bm25Params,
    ) {
        self.block_maxes.clear();
        let num_blocks = self.entries.len().div_ceil(BLOCK_SIZE);
        self.block_maxes.reserve(num_blocks);

        for chunk in self.entries.chunks(BLOCK_SIZE) {
            let mut max_score = 0.0f32;
            for entry in chunk {
                let doc_len = doc_lengths[entry.doc_id as usize] as f32;
                let score = bm25.score(
                    entry.freq as f32,
                    self.df as f32,
                    doc_len,
                    avg_doc_len,
                    total_docs,
                );
                max_score = max_score.max(score);
            }
            self.block_maxes.push(max_score);
        }
    }
}

/// Ultra profile for maximum throughput
pub struct UltraProfile {
    /// Term dictionary: term hash -> posting index
    term_dict: RwLock<HashMap<u64, usize>>,
    /// String term dictionary for lookups
    term_strings: RwLock<HashMap<u64, String>>,
    /// Posting lists
    postings: RwLock<Vec<PostingList>>,
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
    /// Block maxes computed
    block_maxes_dirty: RwLock<bool>,
}

impl UltraProfile {
    pub fn new() -> Self {
        Self {
            term_dict: RwLock::new(HashMap::with_capacity(500_000)),
            term_strings: RwLock::new(HashMap::with_capacity(500_000)),
            postings: RwLock::new(Vec::with_capacity(500_000)),
            doc_lengths: RwLock::new(Vec::with_capacity(10_000_000)),
            doc_ids: RwLock::new(Vec::with_capacity(10_000_000)),
            doc_count: AtomicU64::new(0),
            total_doc_length: AtomicU64::new(0),
            bm25: Bm25Params::default(),
            block_maxes_dirty: RwLock::new(true),
        }
    }

    /// Fast ASCII tokenization - returns (term_hash, freq) pairs
    #[inline]
    fn tokenize_fast(text: &str) -> Vec<(u64, u16, String)> {
        let mut results = Vec::with_capacity(text.len() / 5);
        let mut freqs: HashMap<u64, (u16, String)> = HashMap::with_capacity(100);
        let bytes = text.as_bytes();
        let mut start = 0;
        let mut in_token = false;

        for (i, &b) in bytes.iter().enumerate() {
            let is_alnum = b.is_ascii_alphanumeric();

            if is_alnum {
                if !in_token {
                    start = i;
                    in_token = true;
                }
            } else if in_token {
                let end = i;
                let len = end - start;
                if (2..=32).contains(&len) {
                    // Lowercase and hash in one pass
                    let mut term = String::with_capacity(len);
                    let mut hash = 0xcbf29ce484222325u64;
                    for &c in &bytes[start..end] {
                        let lower = c.to_ascii_lowercase();
                        term.push(lower as char);
                        hash ^= lower as u64;
                        hash = hash.wrapping_mul(0x100000001b3);
                    }
                    freqs
                        .entry(hash)
                        .and_modify(|(f, _)| *f = f.saturating_add(1))
                        .or_insert((1, term));
                }
                in_token = false;
            }
        }

        // Handle last token
        if in_token {
            let len = bytes.len() - start;
            if (2..=32).contains(&len) {
                let mut term = String::with_capacity(len);
                let mut hash = 0xcbf29ce484222325u64;
                for &c in &bytes[start..] {
                    let lower = c.to_ascii_lowercase();
                    term.push(lower as char);
                    hash ^= lower as u64;
                    hash = hash.wrapping_mul(0x100000001b3);
                }
                freqs
                    .entry(hash)
                    .and_modify(|(f, _)| *f = f.saturating_add(1))
                    .or_insert((1, term));
            }
        }

        results.extend(freqs.into_iter().map(|(h, (f, t))| (h, f, t)));
        results
    }

    /// Compute block max scores for all posting lists
    fn compute_all_block_maxes(&self) {
        if !*self.block_maxes_dirty.read() {
            return;
        }

        let doc_count = self.doc_count.load(Ordering::Relaxed);
        if doc_count == 0 {
            return;
        }

        let total_docs = doc_count as f32;
        let total_len = self.total_doc_length.load(Ordering::Relaxed) as f32;
        let avg_doc_len = total_len / total_docs;

        let doc_lengths = self.doc_lengths.read();
        let mut postings = self.postings.write();

        for posting in postings.iter_mut() {
            posting.compute_block_maxes(&doc_lengths, avg_doc_len, total_docs, &self.bm25);
        }

        *self.block_maxes_dirty.write() = false;
    }

    /// Search with early termination
    fn search_fast(&self, query: &str, limit: usize, offset: usize) -> Vec<SearchHit> {
        self.compute_all_block_maxes();

        let term_dict = self.term_dict.read();
        let postings = self.postings.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = self.doc_count.load(Ordering::Relaxed);

        if doc_count == 0 {
            return Vec::new();
        }

        let total_docs = doc_count as f32;
        let total_len = self.total_doc_length.load(Ordering::Relaxed) as f32;
        let avg_doc_len = total_len / total_docs;

        // Parse query terms
        let query_terms = Self::tokenize_fast(query);
        if query_terms.is_empty() {
            return Vec::new();
        }

        // Collect matching postings
        let mut query_postings: Vec<(&PostingList, f32)> = Vec::new();
        for (hash, _, _) in &query_terms {
            if let Some(&idx) = term_dict.get(hash) {
                let posting = &postings[idx];
                let upper_bound = posting.idf * (self.bm25.k1 + 1.0);
                query_postings.push((posting, upper_bound));
            }
        }

        if query_postings.is_empty() {
            return Vec::new();
        }

        // Sort by upper bound
        query_postings.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));

        // Score documents
        let k = limit + offset;
        let mut top_k: BinaryHeap<Reverse<(OrderedFloat, u32)>> = BinaryHeap::with_capacity(k + 1);
        let mut threshold = 0.0f32;
        let mut scored: HashMap<u32, f32> = HashMap::with_capacity(k * 10);

        for (posting, _) in &query_postings {
            for (block_idx, chunk) in posting.entries.chunks(BLOCK_SIZE).enumerate() {
                // Skip if block can't beat threshold
                if block_idx < posting.block_maxes.len()
                    && posting.block_maxes[block_idx] < threshold
                    && !top_k.is_empty()
                {
                    continue;
                }

                for entry in chunk {
                    let doc_len = doc_lengths[entry.doc_id as usize] as f32;
                    let score = self.bm25.score(
                        entry.freq as f32,
                        posting.df as f32,
                        doc_len,
                        avg_doc_len,
                        total_docs,
                    );
                    *scored.entry(entry.doc_id).or_insert(0.0) += score;
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

impl Default for UltraProfile {
    fn default() -> Self {
        Self::new()
    }
}

impl SearchProfile for UltraProfile {
    fn name(&self) -> &'static str {
        "ultra"
    }

    fn profile_type(&self) -> ProfileType {
        ProfileType::Ultra
    }

    fn index_batch(&mut self, docs: &[Document]) -> Result<usize, IndexError> {
        let base_doc_id = self.doc_count.load(Ordering::Relaxed) as u32;

        // Parallel tokenization
        let tokenized: Vec<_> = docs
            .par_iter()
            .enumerate()
            .map(|(i, doc)| {
                let terms = Self::tokenize_fast(&doc.text);
                let doc_len: u32 = terms.iter().map(|(_, f, _)| *f as u32).sum();
                (
                    base_doc_id + i as u32,
                    doc.id.clone(),
                    terms,
                    doc_len as u16,
                )
            })
            .collect();

        // Update counts atomically
        self.doc_count
            .fetch_add(docs.len() as u64, Ordering::Relaxed);

        // Sequential update of shared data structures
        {
            let mut term_dict = self.term_dict.write();
            let mut term_strings = self.term_strings.write();
            let mut postings = self.postings.write();
            let mut doc_lengths = self.doc_lengths.write();
            let mut doc_ids = self.doc_ids.write();

            doc_lengths.reserve(tokenized.len());
            doc_ids.reserve(tokenized.len());

            let total_docs = self.doc_count.load(Ordering::Relaxed) as f32;

            for (doc_id, ext_id, terms, doc_len) in tokenized {
                doc_ids.push(ext_id);
                doc_lengths.push(doc_len);
                self.total_doc_length
                    .fetch_add(doc_len as u64, Ordering::Relaxed);

                for (hash, freq, term_str) in terms {
                    let idx = if let Some(&idx) = term_dict.get(&hash) {
                        idx
                    } else {
                        let idx = postings.len();
                        term_dict.insert(hash, idx);
                        term_strings.insert(hash, term_str);
                        postings.push(PostingList::new());
                        idx
                    };

                    let posting = &mut postings[idx];
                    posting.entries.push(PostingEntry { doc_id, freq });
                    posting.df += 1;
                }
            }

            // Update IDFs
            for posting in postings.iter_mut() {
                posting.idf =
                    ((total_docs - posting.df as f32 + 0.5) / (posting.df as f32 + 0.5) + 1.0).ln();
            }
        }

        *self.block_maxes_dirty.write() = true;

        Ok(docs.len())
    }

    fn commit(&mut self) -> Result<(), IndexError> {
        self.compute_all_block_maxes();
        Ok(())
    }

    fn search(
        &self,
        query: &str,
        limit: usize,
        offset: usize,
    ) -> Result<SearchResult, SearchError> {
        let start = Instant::now();
        let hits = self.search_fast(query, limit, offset);
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

        let term_dict_bytes = term_dict.len() * 16;
        let postings_bytes: usize = postings
            .iter()
            .map(|p| p.entries.len() * 6 + p.block_maxes.len() * 4 + 12)
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
        let file = File::create(path.join("ultra.idx"))?;
        let mut writer = BufWriter::with_capacity(256 * 1024, file);

        writer.write_all(b"ULTR")?;
        writer.write_all(&1u32.to_le_bytes())?;

        let term_dict = self.term_dict.read();
        let term_strings = self.term_strings.read();
        let postings = self.postings.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = self.doc_count.load(Ordering::Relaxed);
        let total_doc_length = self.total_doc_length.load(Ordering::Relaxed);

        writer.write_all(&(term_dict.len() as u64).to_le_bytes())?;
        writer.write_all(&(postings.len() as u64).to_le_bytes())?;
        writer.write_all(&doc_count.to_le_bytes())?;
        writer.write_all(&total_doc_length.to_le_bytes())?;

        // Write term dictionary
        for (&hash, &idx) in term_dict.iter() {
            writer.write_all(&hash.to_le_bytes())?;
            writer.write_all(&(idx as u64).to_le_bytes())?;
            if let Some(term) = term_strings.get(&hash) {
                let bytes = term.as_bytes();
                writer.write_all(&(bytes.len() as u32).to_le_bytes())?;
                writer.write_all(bytes)?;
            } else {
                writer.write_all(&0u32.to_le_bytes())?;
            }
        }

        // Write postings
        for posting in postings.iter() {
            writer.write_all(&posting.df.to_le_bytes())?;
            writer.write_all(&posting.idf.to_le_bytes())?;
            writer.write_all(&(posting.entries.len() as u64).to_le_bytes())?;
            for entry in &posting.entries {
                writer.write_all(&entry.doc_id.to_le_bytes())?;
                writer.write_all(&entry.freq.to_le_bytes())?;
            }
        }

        // Write doc lengths
        for &len in doc_lengths.iter() {
            writer.write_all(&len.to_le_bytes())?;
        }

        // Write doc IDs
        for id in doc_ids.iter() {
            let bytes = id.as_bytes();
            writer.write_all(&(bytes.len() as u32).to_le_bytes())?;
            writer.write_all(bytes)?;
        }

        writer.flush()?;
        Ok(())
    }

    fn load(&mut self, path: &Path) -> Result<(), IndexError> {
        let file = File::open(path.join("ultra.idx"))?;
        let mut reader = BufReader::with_capacity(256 * 1024, file);

        let mut magic = [0u8; 4];
        reader.read_exact(&mut magic)?;
        if &magic != b"ULTR" {
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

        // Read term dictionary
        let mut term_dict = HashMap::with_capacity(term_count as usize);
        let mut term_strings = HashMap::with_capacity(term_count as usize);
        for _ in 0..term_count {
            reader.read_exact(&mut buf8)?;
            let hash = u64::from_le_bytes(buf8);
            reader.read_exact(&mut buf8)?;
            let idx = u64::from_le_bytes(buf8) as usize;
            reader.read_exact(&mut buf4)?;
            let term_len = u32::from_le_bytes(buf4) as usize;
            if term_len > 0 {
                let mut term_bytes = vec![0u8; term_len];
                reader.read_exact(&mut term_bytes)?;
                let term = String::from_utf8(term_bytes)
                    .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?;
                term_strings.insert(hash, term);
            }
            term_dict.insert(hash, idx);
        }

        // Read postings
        let mut postings = Vec::with_capacity(posting_count as usize);
        for _ in 0..posting_count {
            reader.read_exact(&mut buf4)?;
            let df = u32::from_le_bytes(buf4);
            reader.read_exact(&mut buf4)?;
            let idf = f32::from_le_bytes(buf4);
            reader.read_exact(&mut buf8)?;
            let entry_count = u64::from_le_bytes(buf8) as usize;

            let mut entries = Vec::with_capacity(entry_count);
            for _ in 0..entry_count {
                reader.read_exact(&mut buf4)?;
                let doc_id = u32::from_le_bytes(buf4);
                reader.read_exact(&mut buf2)?;
                let freq = u16::from_le_bytes(buf2);
                entries.push(PostingEntry { doc_id, freq });
            }

            postings.push(PostingList {
                entries,
                block_maxes: Vec::new(),
                df,
                idf,
            });
        }

        // Read doc lengths
        let mut doc_lengths = Vec::with_capacity(doc_count as usize);
        for _ in 0..doc_count {
            reader.read_exact(&mut buf2)?;
            doc_lengths.push(u16::from_le_bytes(buf2));
        }

        // Read doc IDs
        let mut doc_ids = Vec::with_capacity(doc_count as usize);
        for _ in 0..doc_count {
            reader.read_exact(&mut buf4)?;
            let len = u32::from_le_bytes(buf4) as usize;
            let mut bytes = vec![0u8; len];
            reader.read_exact(&mut bytes)?;
            doc_ids.push(
                String::from_utf8(bytes)
                    .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?,
            );
        }

        *self.term_dict.write() = term_dict;
        *self.term_strings.write() = term_strings;
        *self.postings.write() = postings;
        *self.doc_lengths.write() = doc_lengths;
        *self.doc_ids.write() = doc_ids;
        self.doc_count.store(doc_count, Ordering::Relaxed);
        self.total_doc_length
            .store(total_doc_length, Ordering::Relaxed);
        *self.block_maxes_dirty.write() = true;

        Ok(())
    }

    fn doc_count(&self) -> u64 {
        self.doc_count.load(Ordering::Relaxed)
    }

    fn clear(&mut self) {
        self.term_dict.write().clear();
        self.term_strings.write().clear();
        self.postings.write().clear();
        self.doc_lengths.write().clear();
        self.doc_ids.write().clear();
        self.doc_count.store(0, Ordering::Relaxed);
        self.total_doc_length.store(0, Ordering::Relaxed);
        *self.block_maxes_dirty.write() = true;
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

    #[test]
    fn test_ultra_basic() {
        let mut profile = UltraProfile::new();

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
    fn test_ultra_throughput() {
        let mut profile = UltraProfile::new();

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
        println!("Ultra throughput: {:.0} docs/sec", throughput);

        assert!(
            throughput > 100_000.0,
            "Expected >100k docs/sec, got {}",
            throughput
        );
    }
}
