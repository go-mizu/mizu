//! Seismic profile: Learned sparse retrieval with geometry-cohesive blocks

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

/// Embedding dimension for learned representations
const EMBED_DIM: usize = 32;

/// Block size for geometry-cohesive partitioning
const SEISMIC_BLOCK_SIZE: usize = 256;

/// Type alias for pending document data
type PendingDoc = (String, HashMap<String, u16>, u32);

/// A geometry-cohesive block
#[derive(Debug, Clone)]
struct SeismicBlock {
    /// Document IDs in this block
    doc_ids: Vec<u32>,
    /// Term frequencies for BM25 fallback
    term_freqs: Vec<Vec<(String, u16)>>,
    /// Document lengths
    doc_lengths: Vec<u16>,
    /// Block centroid (learned embedding)
    centroid: [f32; EMBED_DIM],
    /// Maximum score potential in this block
    max_score: f32,
}

impl SeismicBlock {
    fn new() -> Self {
        Self {
            doc_ids: Vec::with_capacity(SEISMIC_BLOCK_SIZE),
            term_freqs: Vec::with_capacity(SEISMIC_BLOCK_SIZE),
            doc_lengths: Vec::with_capacity(SEISMIC_BLOCK_SIZE),
            centroid: [0.0; EMBED_DIM],
            max_score: 0.0,
        }
    }

    fn len(&self) -> usize {
        self.doc_ids.len()
    }

    fn is_full(&self) -> bool {
        self.len() >= SEISMIC_BLOCK_SIZE
    }
}

/// Seismic profile with geometry-cohesive block partitioning
pub struct SeismicProfile {
    /// Blocks organized by geometry
    blocks: RwLock<Vec<SeismicBlock>>,
    /// Term dictionary for IDF
    term_dict: RwLock<HashMap<String, u32>>,
    /// Document IDs mapping
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
    /// Current block being filled
    current_block: RwLock<SeismicBlock>,
}

impl SeismicProfile {
    pub fn new() -> Self {
        Self {
            blocks: RwLock::new(Vec::new()),
            term_dict: RwLock::new(HashMap::new()),
            doc_ids: RwLock::new(Vec::new()),
            doc_count: RwLock::new(0),
            total_doc_length: RwLock::new(0),
            bm25: Bm25Params::default(),
            tokenizer: FastTokenizer::default(),
            pending: RwLock::new(Vec::new()),
            current_block: RwLock::new(SeismicBlock::new()),
        }
    }

    /// Generate a simple embedding from term frequencies
    /// In production, this would use a learned model
    fn generate_embedding(term_freqs: &HashMap<String, u16>) -> [f32; EMBED_DIM] {
        let mut embedding = [0.0f32; EMBED_DIM];

        for (term, freq) in term_freqs {
            // Simple hash-based embedding
            let hash = Self::hash_term(term);
            let idx = (hash as usize) % EMBED_DIM;
            embedding[idx] += *freq as f32;
        }

        // L2 normalize
        let norm: f32 = embedding.iter().map(|x| x * x).sum::<f32>().sqrt();
        if norm > 0.0 {
            for x in &mut embedding {
                *x /= norm;
            }
        }

        embedding
    }

    fn hash_term(term: &str) -> u64 {
        let mut hash = 5381u64;
        for byte in term.bytes() {
            hash = hash.wrapping_mul(33).wrapping_add(byte as u64);
        }
        hash
    }

    /// Compute cosine similarity between embeddings
    #[inline]
    fn cosine_similarity(a: &[f32; EMBED_DIM], b: &[f32; EMBED_DIM]) -> f32 {
        let mut dot = 0.0f32;
        for i in 0..EMBED_DIM {
            dot += a[i] * b[i];
        }
        dot
    }

    fn build_index(&self) {
        let pending = self.pending.read();
        if pending.is_empty() {
            return;
        }

        let mut blocks = self.blocks.write();
        let mut term_dict = self.term_dict.write();
        let mut doc_ids = self.doc_ids.write();
        let mut doc_count = self.doc_count.write();
        let mut total_doc_length = self.total_doc_length.write();
        let mut current_block = self.current_block.write();

        let base_doc_id = *doc_count as u32;

        for (i, (ext_id, tfs, doc_len)) in pending.iter().enumerate() {
            let doc_id = base_doc_id + i as u32;
            doc_ids.push(ext_id.clone());
            *total_doc_length += *doc_len as u64;

            // Update term dictionary
            for term in tfs.keys() {
                *term_dict.entry(term.clone()).or_insert(0) += 1;
            }

            // Generate embedding
            let embedding = Self::generate_embedding(tfs);

            // Convert term_freqs to vec
            let tf_vec: Vec<(String, u16)> = tfs.iter().map(|(k, v)| (k.clone(), *v)).collect();

            // Add to current block
            current_block.doc_ids.push(doc_id);
            current_block.term_freqs.push(tf_vec);
            current_block.doc_lengths.push(*doc_len as u16);

            // Update centroid (running average)
            let n = current_block.len() as f32;
            for (j, &emb_val) in embedding.iter().enumerate().take(EMBED_DIM) {
                current_block.centroid[j] = current_block.centroid[j] * (n - 1.0) / n + emb_val / n;
            }

            // Finalize block if full
            if current_block.is_full() {
                let total_docs = (*doc_count + i as u64 + 1) as f32;
                let avg_doc_len = *total_doc_length as f32 / total_docs;

                // Calculate max score for block
                let mut max_score = 0.0f32;
                for (tfs, &doc_len) in current_block
                    .term_freqs
                    .iter()
                    .zip(current_block.doc_lengths.iter())
                {
                    let mut score = 0.0f32;
                    for (term, freq) in tfs {
                        let df = *term_dict.get(term).unwrap_or(&1) as f32;
                        score += self.bm25.score(
                            *freq as f32,
                            df,
                            doc_len as f32,
                            avg_doc_len,
                            total_docs,
                        );
                    }
                    max_score = max_score.max(score);
                }
                current_block.max_score = max_score;

                let completed_block = std::mem::replace(&mut *current_block, SeismicBlock::new());
                blocks.push(completed_block);
            }
        }

        *doc_count += pending.len() as u64;
    }

    fn search_seismic(
        &self,
        query_terms: &[String],
        limit: usize,
        offset: usize,
    ) -> Vec<SearchHit> {
        let blocks = self.blocks.read();
        let term_dict = self.term_dict.read();
        let doc_ids = self.doc_ids.read();
        let current_block = self.current_block.read();
        let doc_count = *self.doc_count.read();

        if doc_count == 0 || query_terms.is_empty() {
            return Vec::new();
        }

        let total_docs = doc_count as f32;
        let total_doc_length = *self.total_doc_length.read();
        let avg_doc_len = if doc_count > 0 {
            total_doc_length as f32 / doc_count as f32
        } else {
            1.0
        };

        // Generate query embedding
        let query_tfs: HashMap<String, u16> = query_terms.iter().map(|t| (t.clone(), 1)).collect();
        let query_embedding = Self::generate_embedding(&query_tfs);

        // Score blocks by centroid similarity
        let mut block_scores: Vec<(usize, f32)> = blocks
            .iter()
            .enumerate()
            .map(|(i, block)| {
                (
                    i,
                    Self::cosine_similarity(&query_embedding, &block.centroid),
                )
            })
            .collect();

        // Sort by similarity descending
        block_scores.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap());

        // Top-k heap
        let k = limit + offset;
        let mut top_k: BinaryHeap<Reverse<(OrderedFloat, u32)>> = BinaryHeap::with_capacity(k + 1);
        let mut threshold = 0.0f32;

        // Process blocks in order of centroid similarity
        for (block_idx, _block_sim) in block_scores {
            let block = &blocks[block_idx];

            // Prune block if max score can't beat threshold
            if block.max_score < threshold && top_k.len() >= k {
                continue;
            }

            // Score documents in block
            for (i, &internal_doc_id) in block.doc_ids.iter().enumerate() {
                let tfs = &block.term_freqs[i];
                let doc_len = block.doc_lengths[i] as f32;

                let mut score = 0.0f32;
                for query_term in query_terms {
                    // Find term in document
                    if let Some((_, freq)) = tfs.iter().find(|(t, _)| t == query_term) {
                        let df = *term_dict.get(query_term).unwrap_or(&1) as f32;
                        score +=
                            self.bm25
                                .score(*freq as f32, df, doc_len, avg_doc_len, total_docs);
                    }
                }

                if score > 0.0 {
                    let entry = Reverse((OrderedFloat(score), internal_doc_id));
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
            }
        }

        // Also check current (unflushed) block
        if !current_block.doc_ids.is_empty() {
            for (i, &internal_doc_id) in current_block.doc_ids.iter().enumerate() {
                let tfs = &current_block.term_freqs[i];
                let doc_len = current_block.doc_lengths[i] as f32;

                let mut score = 0.0f32;
                for query_term in query_terms {
                    if let Some((_, freq)) = tfs.iter().find(|(t, _)| t == query_term) {
                        let df = *term_dict.get(query_term).unwrap_or(&1) as f32;
                        score +=
                            self.bm25
                                .score(*freq as f32, df, doc_len, avg_doc_len, total_docs);
                    }
                }

                if score > 0.0 {
                    let entry = Reverse((OrderedFloat(score), internal_doc_id));
                    if top_k.len() < k {
                        top_k.push(entry);
                    } else if score > threshold {
                        top_k.pop();
                        top_k.push(entry);
                        threshold = top_k.peek().unwrap().0 .0 .0;
                    }
                }
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

impl Default for SeismicProfile {
    fn default() -> Self {
        Self::new()
    }
}

impl SearchProfile for SeismicProfile {
    fn name(&self) -> &'static str {
        "seismic"
    }

    fn profile_type(&self) -> ProfileType {
        ProfileType::Seismic
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
        let hits = self.search_seismic(&query_terms, limit, offset);
        let total = hits.len() as u64;

        Ok(SearchResult {
            hits,
            total,
            duration: start.elapsed(),
            profile: self.name().to_string(),
        })
    }

    fn memory_stats(&self) -> MemoryStats {
        let blocks = self.blocks.read();
        let term_dict = self.term_dict.read();
        let doc_ids = self.doc_ids.read();
        let current_block = self.current_block.read();

        let blocks_bytes: usize = blocks
            .iter()
            .map(|b| {
                b.doc_ids.len() * 4
                    + b.term_freqs.iter().map(|v| v.len() * 20).sum::<usize>()
                    + b.doc_lengths.len() * 2
                    + EMBED_DIM * 4
                    + 4
            })
            .sum();

        let current_block_bytes = current_block.doc_ids.len() * 4
            + current_block
                .term_freqs
                .iter()
                .map(|v| v.len() * 20)
                .sum::<usize>()
            + current_block.doc_lengths.len() * 2
            + EMBED_DIM * 4;

        let term_dict_bytes = term_dict.len() * 36;
        let doc_ids_bytes: usize = doc_ids.iter().map(|s| s.len()).sum();

        MemoryStats {
            index_bytes: (blocks_bytes + current_block_bytes + term_dict_bytes + doc_ids_bytes)
                as u64,
            term_dict_bytes: term_dict_bytes as u64,
            postings_bytes: (blocks_bytes + current_block_bytes) as u64,
            docs_indexed: *self.doc_count.read(),
            mmap_bytes: 0,
        }
    }

    fn save(&self, path: &Path) -> Result<(), IndexError> {
        let file = File::create(path.join("seismic.idx"))?;
        let mut writer = BufWriter::new(file);

        // Header
        writer.write_all(b"SEIS")?;
        writer.write_all(&1u32.to_le_bytes())?;

        let blocks = self.blocks.read();
        let term_dict = self.term_dict.read();
        let doc_ids = self.doc_ids.read();
        let current_block = self.current_block.read();
        let doc_count = *self.doc_count.read();
        let total_doc_length = *self.total_doc_length.read();

        // Counts
        writer.write_all(&(blocks.len() as u64).to_le_bytes())?;
        writer.write_all(&(term_dict.len() as u64).to_le_bytes())?;
        writer.write_all(&doc_count.to_le_bytes())?;
        writer.write_all(&total_doc_length.to_le_bytes())?;

        // Term dict
        for (term, &df) in term_dict.iter() {
            let term_bytes = term.as_bytes();
            writer.write_all(&(term_bytes.len() as u32).to_le_bytes())?;
            writer.write_all(term_bytes)?;
            writer.write_all(&df.to_le_bytes())?;
        }

        // Helper to write a block
        let write_block =
            |writer: &mut BufWriter<File>, block: &SeismicBlock| -> std::io::Result<()> {
                writer.write_all(&(block.doc_ids.len() as u32).to_le_bytes())?;

                for &doc_id in &block.doc_ids {
                    writer.write_all(&doc_id.to_le_bytes())?;
                }

                for tfs in &block.term_freqs {
                    writer.write_all(&(tfs.len() as u32).to_le_bytes())?;
                    for (term, freq) in tfs {
                        let term_bytes = term.as_bytes();
                        writer.write_all(&(term_bytes.len() as u32).to_le_bytes())?;
                        writer.write_all(term_bytes)?;
                        writer.write_all(&freq.to_le_bytes())?;
                    }
                }

                for &len in &block.doc_lengths {
                    writer.write_all(&len.to_le_bytes())?;
                }

                for &c in &block.centroid {
                    writer.write_all(&c.to_le_bytes())?;
                }

                writer.write_all(&block.max_score.to_le_bytes())?;
                Ok(())
            };

        // Write blocks
        for block in blocks.iter() {
            write_block(&mut writer, block)?;
        }

        // Write current block
        writer.write_all(
            &(if current_block.doc_ids.is_empty() {
                0u8
            } else {
                1u8
            })
            .to_le_bytes(),
        )?;
        if !current_block.doc_ids.is_empty() {
            write_block(&mut writer, &current_block)?;
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
        let file = File::open(path.join("seismic.idx"))?;
        let mut reader = BufReader::new(file);

        let mut magic = [0u8; 4];
        reader.read_exact(&mut magic)?;
        if &magic != b"SEIS" {
            return Err(IndexError::Corrupted("Invalid magic".into()));
        }

        let mut buf1 = [0u8; 1];
        let mut buf4 = [0u8; 4];
        let mut buf8 = [0u8; 8];

        reader.read_exact(&mut buf4)?; // version

        reader.read_exact(&mut buf8)?;
        let block_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let term_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let doc_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let total_doc_length = u64::from_le_bytes(buf8);

        // Term dict
        let mut term_dict = HashMap::with_capacity(term_count as usize);
        for _ in 0..term_count {
            reader.read_exact(&mut buf4)?;
            let term_len = u32::from_le_bytes(buf4) as usize;
            let mut term_bytes = vec![0u8; term_len];
            reader.read_exact(&mut term_bytes)?;
            let term = String::from_utf8(term_bytes)
                .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?;
            reader.read_exact(&mut buf4)?;
            let df = u32::from_le_bytes(buf4);
            term_dict.insert(term, df);
        }

        // Helper to read a block
        let read_block = |reader: &mut BufReader<File>| -> Result<SeismicBlock, IndexError> {
            let mut buf2 = [0u8; 2];
            let mut buf4 = [0u8; 4];

            reader.read_exact(&mut buf4)?;
            let block_size = u32::from_le_bytes(buf4) as usize;

            let mut doc_ids = Vec::with_capacity(block_size);
            for _ in 0..block_size {
                reader.read_exact(&mut buf4)?;
                doc_ids.push(u32::from_le_bytes(buf4));
            }

            let mut term_freqs = Vec::with_capacity(block_size);
            for _ in 0..block_size {
                reader.read_exact(&mut buf4)?;
                let tf_count = u32::from_le_bytes(buf4) as usize;
                let mut tfs = Vec::with_capacity(tf_count);
                for _ in 0..tf_count {
                    reader.read_exact(&mut buf4)?;
                    let term_len = u32::from_le_bytes(buf4) as usize;
                    let mut term_bytes = vec![0u8; term_len];
                    reader.read_exact(&mut term_bytes)?;
                    let term = String::from_utf8(term_bytes)
                        .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?;
                    reader.read_exact(&mut buf2)?;
                    let freq = u16::from_le_bytes(buf2);
                    tfs.push((term, freq));
                }
                term_freqs.push(tfs);
            }

            let mut doc_lengths = Vec::with_capacity(block_size);
            for _ in 0..block_size {
                reader.read_exact(&mut buf2)?;
                doc_lengths.push(u16::from_le_bytes(buf2));
            }

            let mut centroid = [0.0f32; EMBED_DIM];
            for c in &mut centroid {
                reader.read_exact(&mut buf4)?;
                *c = f32::from_le_bytes(buf4);
            }

            reader.read_exact(&mut buf4)?;
            let max_score = f32::from_le_bytes(buf4);

            Ok(SeismicBlock {
                doc_ids,
                term_freqs,
                doc_lengths,
                centroid,
                max_score,
            })
        };

        // Read blocks
        let mut blocks = Vec::with_capacity(block_count as usize);
        for _ in 0..block_count {
            blocks.push(read_block(&mut reader)?);
        }

        // Read current block
        reader.read_exact(&mut buf1)?;
        let current_block = if buf1[0] == 1 {
            read_block(&mut reader)?
        } else {
            SeismicBlock::new()
        };

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

        *self.blocks.write() = blocks;
        *self.term_dict.write() = term_dict;
        *self.doc_ids.write() = doc_ids;
        *self.doc_count.write() = doc_count;
        *self.total_doc_length.write() = total_doc_length;
        *self.current_block.write() = current_block;

        Ok(())
    }

    fn doc_count(&self) -> u64 {
        *self.doc_count.read()
    }

    fn clear(&mut self) {
        self.blocks.write().clear();
        self.term_dict.write().clear();
        self.doc_ids.write().clear();
        *self.doc_count.write() = 0;
        *self.total_doc_length.write() = 0;
        self.pending.write().clear();
        *self.current_block.write() = SeismicBlock::new();
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
    fn test_seismic_index_search() {
        let mut profile = SeismicProfile::new();
        let docs = vec![
            Document::new("1", "machine learning is revolutionizing technology"),
            Document::new("2", "deep learning neural networks"),
            Document::new("3", "machine learning algorithms for data science"),
        ];

        profile.index_batch(&docs).unwrap();
        profile.commit().unwrap();

        let result = profile.search("machine learning", 10, 0).unwrap();
        assert!(result.hits.len() >= 2);
    }
}
