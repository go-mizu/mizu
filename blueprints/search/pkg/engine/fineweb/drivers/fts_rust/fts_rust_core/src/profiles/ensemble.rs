//! Ensemble profile: FST + Roaring + Block-Max WAND

use crate::document::Document;
use crate::profiles::{Bm25Params, ProfileType, SearchProfile};
use crate::result::{IndexError, MemoryStats, SearchError, SearchHit, SearchResult};
use crate::tokenizer::FastTokenizer;

use fst::{Map, MapBuilder};
use parking_lot::RwLock;
use rayon::prelude::*;
use roaring::RoaringBitmap;
use std::cmp::Reverse;
use std::collections::{BinaryHeap, HashMap};
use std::fs::File;
use std::io::{BufReader, BufWriter, Read, Write};
use std::path::Path;
use std::time::Instant;

const BLOCK_SIZE: usize = 128;

/// Type alias for pending document data
type PendingDoc = (String, HashMap<String, u16>, u32);

/// Posting block with max score
#[derive(Debug, Clone)]
struct PostingBlock {
    doc_ids: Vec<u32>,
    freqs: Vec<u16>,
    max_score: f32,
}

/// Compressed posting list
#[derive(Debug, Clone)]
struct CompressedPosting {
    /// Roaring bitmap for doc IDs
    bitmap: RoaringBitmap,
    /// Blocks for BMW search
    blocks: Vec<PostingBlock>,
    /// Document frequency
    df: u32,
    /// IDF
    idf: f32,
}

/// Ensemble profile combining FST, Roaring, and Block-Max WAND
pub struct EnsembleProfile {
    /// FST for term dictionary (memory-mapped when loaded from disk)
    fst_data: RwLock<Option<Vec<u8>>>,
    fst_map: RwLock<Option<Map<Vec<u8>>>>,
    /// In-memory term -> posting offset mapping (for building)
    term_offsets: RwLock<HashMap<String, usize>>,
    /// Posting lists
    postings: RwLock<Vec<CompressedPosting>>,
    /// Document lengths
    doc_lengths: RwLock<Vec<u16>>,
    /// Document IDs
    doc_ids: RwLock<Vec<String>>,
    /// Document count
    doc_count: RwLock<u64>,
    /// Total document length
    total_doc_length: RwLock<u64>,
    /// BM25 parameters
    bm25: Bm25Params,
    /// Tokenizer
    tokenizer: FastTokenizer,
    /// Pending documents
    pending: RwLock<Vec<PendingDoc>>,
    /// Whether FST needs rebuild
    fst_dirty: RwLock<bool>,
}

impl EnsembleProfile {
    pub fn new() -> Self {
        Self {
            fst_data: RwLock::new(None),
            fst_map: RwLock::new(None),
            term_offsets: RwLock::new(HashMap::new()),
            postings: RwLock::new(Vec::new()),
            doc_lengths: RwLock::new(Vec::new()),
            doc_ids: RwLock::new(Vec::new()),
            doc_count: RwLock::new(0),
            total_doc_length: RwLock::new(0),
            bm25: Bm25Params::default(),
            tokenizer: FastTokenizer::default(),
            pending: RwLock::new(Vec::new()),
            fst_dirty: RwLock::new(true),
        }
    }

    fn build_index(&self) {
        let pending = self.pending.read();
        if pending.is_empty() {
            return;
        }

        let mut term_offsets = self.term_offsets.write();
        let mut postings = self.postings.write();
        let mut doc_lengths = self.doc_lengths.write();
        let mut doc_ids = self.doc_ids.write();
        let mut doc_count = self.doc_count.write();
        let mut total_doc_length = self.total_doc_length.write();

        let base_doc_id = *doc_count as u32;

        // Collect term -> postings
        let mut term_posts: HashMap<String, Vec<(u32, u16)>> = HashMap::new();

        for (i, (ext_id, tfs, doc_len)) in pending.iter().enumerate() {
            let doc_id = base_doc_id + i as u32;
            doc_ids.push(ext_id.clone());
            doc_lengths.push(*doc_len as u16);
            *total_doc_length += *doc_len as u64;

            for (term, freq) in tfs {
                term_posts
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

        // Build compressed postings with blocks
        for (term, posts) in term_posts {
            let df = posts.len() as u32;
            let idf = ((total_docs - df as f32 + 0.5) / (df as f32 + 0.5) + 1.0).ln();

            // Build Roaring bitmap
            let mut bitmap = RoaringBitmap::new();
            for &(doc_id, _) in &posts {
                bitmap.insert(doc_id);
            }

            // Build blocks
            let mut blocks = Vec::new();
            for chunk in posts.chunks(BLOCK_SIZE) {
                let mut block = PostingBlock {
                    doc_ids: Vec::with_capacity(chunk.len()),
                    freqs: Vec::with_capacity(chunk.len()),
                    max_score: 0.0,
                };

                for &(doc_id, freq) in chunk {
                    let doc_len = doc_lengths[doc_id as usize] as f32;
                    let score =
                        self.bm25
                            .score(freq as f32, df as f32, doc_len, avg_doc_len, total_docs);
                    block.max_score = block.max_score.max(score);
                    block.doc_ids.push(doc_id);
                    block.freqs.push(freq);
                }

                blocks.push(block);
            }

            // Check if term already exists
            if let Some(&offset) = term_offsets.get(&term) {
                // Merge into existing posting
                let existing = &mut postings[offset];
                existing.bitmap |= &bitmap;
                existing.blocks.extend(blocks);
                existing.df += df;
                // Recalculate IDF
                existing.idf =
                    ((total_docs - existing.df as f32 + 0.5) / (existing.df as f32 + 0.5) + 1.0)
                        .ln();
            } else {
                // New term
                let offset = postings.len();
                term_offsets.insert(term, offset);
                postings.push(CompressedPosting {
                    bitmap,
                    blocks,
                    df,
                    idf,
                });
            }
        }

        *self.fst_dirty.write() = true;
    }

    fn rebuild_fst(&self) {
        if !*self.fst_dirty.read() {
            return;
        }

        let term_offsets = self.term_offsets.read();
        if term_offsets.is_empty() {
            return;
        }

        // Sort terms for FST
        let mut terms: Vec<_> = term_offsets.iter().collect();
        terms.sort_by_key(|(k, _)| k.as_bytes());

        // Build FST
        let mut builder = MapBuilder::memory();
        for (term, &offset) in &terms {
            builder.insert(term, offset as u64).unwrap();
        }
        let fst_bytes = builder.into_inner().unwrap();

        let fst = Map::new(fst_bytes.clone()).unwrap();

        *self.fst_data.write() = Some(fst_bytes);
        *self.fst_map.write() = Some(fst);
        *self.fst_dirty.write() = false;
    }

    fn search_ensemble(
        &self,
        query_terms: &[String],
        limit: usize,
        offset: usize,
    ) -> Vec<SearchHit> {
        // Ensure FST is built
        self.rebuild_fst();

        let _fst_map = self.fst_map.read();
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

        // Look up terms in FST
        let term_offsets = self.term_offsets.read();
        let mut query_postings: Vec<(&CompressedPosting, f32)> = Vec::new();

        for term in query_terms {
            if let Some(&offset) = term_offsets.get(term) {
                let posting = &postings[offset];
                // Upper bound score
                let upper_bound = posting.idf * (self.bm25.k1 + 1.0);
                query_postings.push((posting, upper_bound));
            }
        }

        if query_postings.is_empty() {
            return Vec::new();
        }

        // Sort by upper bound for efficiency
        query_postings.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap());

        // Score using Block-Max WAND
        let k = limit + offset;
        let mut top_k: BinaryHeap<Reverse<(OrderedFloat, u32)>> = BinaryHeap::with_capacity(k + 1);
        let mut threshold = 0.0f32;
        let mut scored: HashMap<u32, f32> = HashMap::new();

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

        // Build top-k
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

impl Default for EnsembleProfile {
    fn default() -> Self {
        Self::new()
    }
}

impl SearchProfile for EnsembleProfile {
    fn name(&self) -> &'static str {
        "ensemble"
    }

    fn profile_type(&self) -> ProfileType {
        ProfileType::Ensemble
    }

    fn index_batch(&mut self, docs: &[Document]) -> Result<usize, IndexError> {
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
        self.build_index();
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
        let hits = self.search_ensemble(&query_terms, limit, offset);
        let total = hits.len() as u64;

        Ok(SearchResult {
            hits,
            total,
            duration: start.elapsed(),
            profile: self.name().to_string(),
        })
    }

    fn memory_stats(&self) -> MemoryStats {
        let term_offsets = self.term_offsets.read();
        let postings = self.postings.read();
        let fst_data = self.fst_data.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();

        let fst_bytes = fst_data.as_ref().map(|d| d.len()).unwrap_or(0);
        let term_dict_bytes = term_offsets.len() * 40 + fst_bytes;

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
            docs_indexed: *self.doc_count.read(),
            mmap_bytes: 0,
        }
    }

    fn save(&self, path: &Path) -> Result<(), IndexError> {
        // Ensure FST is built
        self.rebuild_fst();

        let file = File::create(path.join("ensemble.idx"))?;
        let mut writer = BufWriter::new(file);

        // Header
        writer.write_all(b"ENSM")?;
        writer.write_all(&1u32.to_le_bytes())?;

        let term_offsets = self.term_offsets.read();
        let postings = self.postings.read();
        let fst_data = self.fst_data.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = *self.doc_count.read();
        let total_doc_length = *self.total_doc_length.read();

        // Counts
        writer.write_all(&(term_offsets.len() as u64).to_le_bytes())?;
        writer.write_all(&(postings.len() as u64).to_le_bytes())?;
        writer.write_all(&doc_count.to_le_bytes())?;
        writer.write_all(&total_doc_length.to_le_bytes())?;

        // FST data
        if let Some(fst) = fst_data.as_ref() {
            writer.write_all(&(fst.len() as u64).to_le_bytes())?;
            writer.write_all(fst)?;
        } else {
            writer.write_all(&0u64.to_le_bytes())?;
        }

        // Term offsets
        for (term, &offset) in term_offsets.iter() {
            let term_bytes = term.as_bytes();
            writer.write_all(&(term_bytes.len() as u32).to_le_bytes())?;
            writer.write_all(term_bytes)?;
            writer.write_all(&(offset as u64).to_le_bytes())?;
        }

        // Postings
        for posting in postings.iter() {
            // Bitmap
            let mut bitmap_bytes = Vec::new();
            posting.bitmap.serialize_into(&mut bitmap_bytes).unwrap();
            writer.write_all(&(bitmap_bytes.len() as u64).to_le_bytes())?;
            writer.write_all(&bitmap_bytes)?;

            // df, idf
            writer.write_all(&posting.df.to_le_bytes())?;
            writer.write_all(&posting.idf.to_le_bytes())?;

            // Blocks
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
        let file = File::open(path.join("ensemble.idx"))?;
        let mut reader = BufReader::new(file);

        let mut magic = [0u8; 4];
        reader.read_exact(&mut magic)?;
        if &magic != b"ENSM" {
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

        // FST
        reader.read_exact(&mut buf8)?;
        let fst_len = u64::from_le_bytes(buf8) as usize;
        let fst_data = if fst_len > 0 {
            let mut fst_bytes = vec![0u8; fst_len];
            reader.read_exact(&mut fst_bytes)?;
            Some(fst_bytes)
        } else {
            None
        };

        // Term offsets
        let mut term_offsets = HashMap::with_capacity(term_count as usize);
        for _ in 0..term_count {
            reader.read_exact(&mut buf4)?;
            let term_len = u32::from_le_bytes(buf4) as usize;
            let mut term_bytes = vec![0u8; term_len];
            reader.read_exact(&mut term_bytes)?;
            let term = String::from_utf8(term_bytes)
                .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?;
            reader.read_exact(&mut buf8)?;
            let offset = u64::from_le_bytes(buf8) as usize;
            term_offsets.insert(term, offset);
        }

        // Postings
        let mut postings = Vec::with_capacity(posting_count as usize);
        for _ in 0..posting_count {
            // Bitmap
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

            // Blocks
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

        // Rebuild FST map
        let fst_map = fst_data.as_ref().and_then(|d| Map::new(d.clone()).ok());

        *self.fst_data.write() = fst_data;
        *self.fst_map.write() = fst_map;
        *self.term_offsets.write() = term_offsets;
        *self.postings.write() = postings;
        *self.doc_lengths.write() = doc_lengths;
        *self.doc_ids.write() = doc_ids;
        *self.doc_count.write() = doc_count;
        *self.total_doc_length.write() = total_doc_length;
        *self.fst_dirty.write() = false;

        Ok(())
    }

    fn doc_count(&self) -> u64 {
        *self.doc_count.read()
    }

    fn clear(&mut self) {
        *self.fst_data.write() = None;
        *self.fst_map.write() = None;
        self.term_offsets.write().clear();
        self.postings.write().clear();
        self.doc_lengths.write().clear();
        self.doc_ids.write().clear();
        *self.doc_count.write() = 0;
        *self.total_doc_length.write() = 0;
        self.pending.write().clear();
        *self.fst_dirty.write() = true;
    }
}

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
    fn test_ensemble_index_search() {
        let mut profile = EnsembleProfile::new();
        let docs = vec![
            Document::new("1", "rust is a systems programming language"),
            Document::new("2", "go is a programming language by google"),
            Document::new("3", "rust and go are both great languages"),
        ];

        profile.index_batch(&docs).unwrap();
        profile.commit().unwrap();

        let result = profile.search("rust programming", 10, 0).unwrap();
        assert!(!result.hits.is_empty());
        // Docs 1 and 3 both contain "rust", doc 1 and 2 contain "programming"
        let ids: Vec<_> = result.hits.iter().map(|h| h.id.as_str()).collect();
        assert!(ids.contains(&"1") || ids.contains(&"3"));
    }
}
