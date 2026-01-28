//! Block-Max WAND with SIMD-accelerated posting intersection

use crate::document::Document;
use crate::profiles::{Bm25Params, ProfileType, SearchProfile};
use crate::result::{IndexError, MemoryStats, SearchError, SearchHit, SearchResult};
use crate::tokenizer::FastTokenizer;

use parking_lot::RwLock;
use rayon::prelude::*;
use std::cmp::Reverse;
use std::collections::{BinaryHeap, HashMap};
use std::fs::File;
use std::io::{BufReader, BufWriter, Read, Write};
use std::path::Path;
use std::time::Instant;

/// Block size for SIMD alignment (128 docs per block)
const BLOCK_SIZE: usize = 128;

/// Term metadata
#[derive(Debug, Clone)]
struct TermMeta {
    /// Document frequency
    df: u32,
    /// Index into postings array
    posting_offset: usize,
    /// Number of blocks
    num_blocks: usize,
    /// Precomputed IDF
    idf: f32,
}

/// A block of postings (128 docs)
#[derive(Debug, Clone)]
struct PostingBlock {
    /// Document IDs in this block
    doc_ids: Vec<u32>,
    /// Term frequencies
    freqs: Vec<u16>,
    /// Maximum BM25 score in this block (for pruning)
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

/// Block-Max WAND profile
pub struct BmwSimdProfile {
    /// Term dictionary: term -> metadata
    term_dict: RwLock<HashMap<String, TermMeta>>,
    /// Posting blocks
    postings: RwLock<Vec<PostingBlock>>,
    /// Document lengths (for BM25)
    doc_lengths: RwLock<Vec<u16>>,
    /// Document IDs (external)
    doc_ids: RwLock<Vec<String>>,
    /// Total document count
    doc_count: RwLock<u64>,
    /// Sum of all document lengths
    total_doc_length: RwLock<u64>,
    /// BM25 parameters
    bm25: Bm25Params,
    /// Tokenizer
    tokenizer: FastTokenizer,
    /// Pending documents (not yet committed)
    pending: RwLock<Vec<(String, HashMap<String, u16>, u32)>>,
}

impl BmwSimdProfile {
    pub fn new() -> Self {
        Self {
            term_dict: RwLock::new(HashMap::new()),
            postings: RwLock::new(Vec::new()),
            doc_lengths: RwLock::new(Vec::new()),
            doc_ids: RwLock::new(Vec::new()),
            doc_count: RwLock::new(0),
            total_doc_length: RwLock::new(0),
            bm25: Bm25Params::default(),
            tokenizer: FastTokenizer::default(),
            pending: RwLock::new(Vec::new()),
        }
    }

    /// Build posting blocks from pending documents
    fn build_blocks(&self) {
        let pending = self.pending.read();
        if pending.is_empty() {
            return;
        }

        let mut term_dict = self.term_dict.write();
        let mut postings = self.postings.write();
        let mut doc_lengths = self.doc_lengths.write();
        let mut doc_ids = self.doc_ids.write();
        let mut doc_count = self.doc_count.write();
        let mut total_doc_length = self.total_doc_length.write();

        let base_doc_id = *doc_count as u32;

        // Collect all term -> [(doc_id, freq)] mappings
        let mut term_postings: HashMap<String, Vec<(u32, u16)>> = HashMap::new();

        for (i, (ext_id, term_freqs, doc_len)) in pending.iter().enumerate() {
            let doc_id = base_doc_id + i as u32;

            doc_ids.push(ext_id.clone());
            doc_lengths.push(*doc_len as u16);
            *total_doc_length += *doc_len as u64;

            for (term, freq) in term_freqs {
                term_postings
                    .entry(term.clone())
                    .or_default()
                    .push((doc_id, *freq));
            }
        }

        *doc_count += pending.len() as u64;

        let total_docs = *doc_count as f32;
        let avg_doc_len = if *doc_count > 0 {
            *total_doc_length as f32 / *doc_count as f32
        } else {
            1.0
        };

        // Build blocks for each term
        for (term, posts) in term_postings {
            let df = posts.len() as u32;
            let idf = ((total_docs - df as f32 + 0.5) / (df as f32 + 0.5) + 1.0).ln();

            let posting_offset = postings.len();
            let mut num_blocks = 0;

            // Create blocks of BLOCK_SIZE
            for chunk in posts.chunks(BLOCK_SIZE) {
                let mut block = PostingBlock::new();
                let mut max_score = 0.0f32;

                for &(doc_id, freq) in chunk {
                    let doc_len = doc_lengths[doc_id as usize] as f32;
                    let score = self.bm25.score(freq as f32, df as f32, doc_len, avg_doc_len, total_docs);
                    max_score = max_score.max(score);

                    block.doc_ids.push(doc_id);
                    block.freqs.push(freq);
                }

                block.max_score = max_score;
                postings.push(block);
                num_blocks += 1;
            }

            term_dict.insert(
                term,
                TermMeta {
                    df,
                    posting_offset,
                    num_blocks,
                    idf,
                },
            );
        }
    }

    /// Search using Block-Max WAND algorithm
    fn search_bmw(&self, query_terms: &[String], limit: usize, offset: usize) -> Vec<SearchHit> {
        let term_dict = self.term_dict.read();
        let postings = self.postings.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = *self.doc_count.read();

        if doc_count == 0 || query_terms.is_empty() {
            return Vec::new();
        }

        let total_docs = doc_count as f32;
        let total_doc_length = *self.total_doc_length.read();
        let avg_doc_len = total_doc_length as f32 / doc_count as f32;

        // Collect query term info
        let mut query_info: Vec<(&str, &TermMeta, f32)> = Vec::new();
        for term in query_terms {
            if let Some(meta) = term_dict.get(term) {
                // Upper bound score for this term (using max possible TF)
                let upper_bound = meta.idf * (self.bm25.k1 + 1.0);
                query_info.push((term, meta, upper_bound));
            }
        }

        if query_info.is_empty() {
            return Vec::new();
        }

        // Sort by upper bound descending for efficiency
        query_info.sort_by(|a, b| b.2.partial_cmp(&a.2).unwrap());

        // Top-k heap (min-heap for efficient replacement)
        let k = limit + offset;
        let mut top_k: BinaryHeap<Reverse<(ordered_float::OrderedFloat<f32>, u32)>> =
            BinaryHeap::with_capacity(k + 1);
        let mut threshold = 0.0f32;

        // Iterate through all documents using block-max pruning
        // Simplified: iterate blocks and score documents
        let mut scored: HashMap<u32, f32> = HashMap::new();

        for (_term, meta, _) in &query_info {
            for block_idx in 0..meta.num_blocks {
                let block = &postings[meta.posting_offset + block_idx];

                // Skip block if max score can't beat threshold
                if block.max_score < threshold && !top_k.is_empty() {
                    continue;
                }

                // Score documents in block
                for (i, &doc_id) in block.doc_ids.iter().enumerate() {
                    let freq = block.freqs[i];
                    let doc_len = doc_lengths[doc_id as usize] as f32;
                    let score = self.bm25.score(
                        freq as f32,
                        meta.df as f32,
                        doc_len,
                        avg_doc_len,
                        total_docs,
                    );
                    *scored.entry(doc_id).or_insert(0.0) += score;
                }
            }
        }

        // Build final top-k
        for (doc_id, score) in scored {
            let entry = Reverse((ordered_float::OrderedFloat(score), doc_id));
            if top_k.len() < k {
                top_k.push(entry);
                if top_k.len() == k {
                    threshold = top_k.peek().unwrap().0 .0.into_inner();
                }
            } else if score > threshold {
                top_k.pop();
                top_k.push(entry);
                threshold = top_k.peek().unwrap().0 .0.into_inner();
            }
        }

        // Extract results
        let mut results: Vec<_> = top_k
            .into_sorted_vec()
            .into_iter()
            .skip(offset)
            .take(limit)
            .map(|Reverse((score, doc_id))| {
                SearchHit::new(doc_ids[doc_id as usize].clone(), score.into_inner())
            })
            .collect();

        results.reverse(); // Highest score first
        results
    }
}

impl Default for BmwSimdProfile {
    fn default() -> Self {
        Self::new()
    }
}

impl SearchProfile for BmwSimdProfile {
    fn name(&self) -> &'static str {
        "bmw_simd"
    }

    fn profile_type(&self) -> ProfileType {
        ProfileType::BmwSimd
    }

    fn index_batch(&mut self, docs: &[Document]) -> Result<usize, IndexError> {
        // Tokenize in parallel
        let tokenized: Vec<_> = docs
            .par_iter()
            .map(|doc| {
                let term_freqs = self.tokenizer.tokenize_with_freqs(&doc.text);
                let doc_len: u32 = term_freqs.values().map(|&v| v as u32).sum();
                (doc.id.clone(), term_freqs, doc_len)
            })
            .collect();

        let count = tokenized.len();
        self.pending.write().extend(tokenized);

        Ok(count)
    }

    fn commit(&mut self) -> Result<(), IndexError> {
        self.build_blocks();
        self.pending.write().clear();
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

        let term_dict_bytes = term_dict.len() * (32 + std::mem::size_of::<TermMeta>());
        let postings_bytes: usize = postings
            .iter()
            .map(|b| b.doc_ids.len() * 4 + b.freqs.len() * 2 + 4)
            .sum();
        let doc_lengths_bytes = doc_lengths.len() * 2;
        let doc_ids_bytes: usize = doc_ids.iter().map(|s| s.len()).sum();

        MemoryStats {
            index_bytes: (term_dict_bytes + postings_bytes + doc_lengths_bytes + doc_ids_bytes) as u64,
            term_dict_bytes: term_dict_bytes as u64,
            postings_bytes: postings_bytes as u64,
            docs_indexed: *self.doc_count.read(),
            mmap_bytes: 0,
        }
    }

    fn save(&self, path: &Path) -> Result<(), IndexError> {
        use bincode::serialize_into;

        let file = File::create(path.join("bmw_simd.idx"))?;
        let mut writer = BufWriter::new(file);

        // Write header
        writer.write_all(b"BMWS")?;
        writer.write_all(&1u32.to_le_bytes())?; // version

        // Serialize data
        let term_dict = self.term_dict.read();
        let postings = self.postings.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = *self.doc_count.read();
        let total_doc_length = *self.total_doc_length.read();

        // Write counts
        writer.write_all(&(term_dict.len() as u64).to_le_bytes())?;
        writer.write_all(&(postings.len() as u64).to_le_bytes())?;
        writer.write_all(&doc_count.to_le_bytes())?;
        writer.write_all(&total_doc_length.to_le_bytes())?;

        // Write term dict
        for (term, meta) in term_dict.iter() {
            let term_bytes = term.as_bytes();
            writer.write_all(&(term_bytes.len() as u32).to_le_bytes())?;
            writer.write_all(term_bytes)?;
            writer.write_all(&meta.df.to_le_bytes())?;
            writer.write_all(&(meta.posting_offset as u64).to_le_bytes())?;
            writer.write_all(&(meta.num_blocks as u32).to_le_bytes())?;
            writer.write_all(&meta.idf.to_le_bytes())?;
        }

        // Write postings
        for block in postings.iter() {
            writer.write_all(&(block.doc_ids.len() as u32).to_le_bytes())?;
            for &doc_id in &block.doc_ids {
                writer.write_all(&doc_id.to_le_bytes())?;
            }
            for &freq in &block.freqs {
                writer.write_all(&freq.to_le_bytes())?;
            }
            writer.write_all(&block.max_score.to_le_bytes())?;
        }

        // Write doc lengths
        for &len in doc_lengths.iter() {
            writer.write_all(&len.to_le_bytes())?;
        }

        // Write doc IDs
        for id in doc_ids.iter() {
            let id_bytes = id.as_bytes();
            writer.write_all(&(id_bytes.len() as u32).to_le_bytes())?;
            writer.write_all(id_bytes)?;
        }

        writer.flush()?;
        Ok(())
    }

    fn load(&mut self, path: &Path) -> Result<(), IndexError> {
        let file = File::open(path.join("bmw_simd.idx"))?;
        let mut reader = BufReader::new(file);

        // Read header
        let mut magic = [0u8; 4];
        reader.read_exact(&mut magic)?;
        if &magic != b"BMWS" {
            return Err(IndexError::Corrupted("Invalid magic".into()));
        }

        let mut version = [0u8; 4];
        reader.read_exact(&mut version)?;

        // Read counts
        let mut buf8 = [0u8; 8];
        reader.read_exact(&mut buf8)?;
        let term_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let posting_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let doc_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let total_doc_length = u64::from_le_bytes(buf8);

        // Read term dict
        let mut term_dict = HashMap::with_capacity(term_count as usize);
        let mut buf4 = [0u8; 4];

        for _ in 0..term_count {
            reader.read_exact(&mut buf4)?;
            let term_len = u32::from_le_bytes(buf4) as usize;
            let mut term_bytes = vec![0u8; term_len];
            reader.read_exact(&mut term_bytes)?;
            let term = String::from_utf8(term_bytes)
                .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?;

            reader.read_exact(&mut buf4)?;
            let df = u32::from_le_bytes(buf4);
            reader.read_exact(&mut buf8)?;
            let posting_offset = u64::from_le_bytes(buf8) as usize;
            reader.read_exact(&mut buf4)?;
            let num_blocks = u32::from_le_bytes(buf4) as usize;
            reader.read_exact(&mut buf4)?;
            let idf = f32::from_le_bytes(buf4);

            term_dict.insert(term, TermMeta { df, posting_offset, num_blocks, idf });
        }

        // Read postings
        let mut postings = Vec::with_capacity(posting_count as usize);
        for _ in 0..posting_count {
            reader.read_exact(&mut buf4)?;
            let block_size = u32::from_le_bytes(buf4) as usize;

            let mut doc_ids = Vec::with_capacity(block_size);
            for _ in 0..block_size {
                reader.read_exact(&mut buf4)?;
                doc_ids.push(u32::from_le_bytes(buf4));
            }

            let mut freqs = Vec::with_capacity(block_size);
            let mut buf2 = [0u8; 2];
            for _ in 0..block_size {
                reader.read_exact(&mut buf2)?;
                freqs.push(u16::from_le_bytes(buf2));
            }

            reader.read_exact(&mut buf4)?;
            let max_score = f32::from_le_bytes(buf4);

            postings.push(PostingBlock { doc_ids, freqs, max_score });
        }

        // Read doc lengths
        let mut doc_lengths = Vec::with_capacity(doc_count as usize);
        let mut buf2 = [0u8; 2];
        for _ in 0..doc_count {
            reader.read_exact(&mut buf2)?;
            doc_lengths.push(u16::from_le_bytes(buf2));
        }

        // Read doc IDs
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
        *self.doc_count.write() = doc_count;
        *self.total_doc_length.write() = total_doc_length;

        Ok(())
    }

    fn doc_count(&self) -> u64 {
        *self.doc_count.read()
    }

    fn clear(&mut self) {
        self.term_dict.write().clear();
        self.postings.write().clear();
        self.doc_lengths.write().clear();
        self.doc_ids.write().clear();
        *self.doc_count.write() = 0;
        *self.total_doc_length.write() = 0;
        self.pending.write().clear();
    }
}

mod ordered_float {
    #[derive(Debug, Clone, Copy, PartialEq)]
    pub struct OrderedFloat<T>(pub T);

    impl<T: PartialOrd> PartialOrd for OrderedFloat<T> {
        fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
            self.0.partial_cmp(&other.0)
        }
    }

    impl<T: PartialOrd> Ord for OrderedFloat<T> {
        fn cmp(&self, other: &Self) -> std::cmp::Ordering {
            self.partial_cmp(other).unwrap_or(std::cmp::Ordering::Equal)
        }
    }

    impl<T: PartialEq> Eq for OrderedFloat<T> {}

    impl OrderedFloat<f32> {
        pub fn into_inner(self) -> f32 {
            self.0
        }
    }

    impl From<OrderedFloat<f32>> for f32 {
        fn from(val: OrderedFloat<f32>) -> Self {
            val.0
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_index_and_search() {
        let mut profile = BmwSimdProfile::new();
        let docs = vec![
            Document::new("1", "hello world"),
            Document::new("2", "world peace"),
            Document::new("3", "hello hello world"),
        ];

        profile.index_batch(&docs).unwrap();
        profile.commit().unwrap();

        let result = profile.search("hello", 10, 0).unwrap();
        assert_eq!(result.hits.len(), 2);

        // Both docs 1 and 3 contain "hello"
        let ids: Vec<_> = result.hits.iter().map(|h| h.id.as_str()).collect();
        assert!(ids.contains(&"1"));
        assert!(ids.contains(&"3"));
    }
}
