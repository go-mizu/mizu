//! Roaring Bitmaps with BM25 scoring profile

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
use std::time::Instant;

/// Type alias for pending document data
type PendingDoc = (String, HashMap<String, u16>, u32);

/// Term metadata for Roaring profile
#[derive(Debug, Clone)]
struct TermMeta {
    /// Document frequency
    df: u32,
    /// Precomputed IDF
    idf: f32,
}

/// Roaring Bitmaps + BM25 profile
pub struct RoaringBm25Profile {
    /// Term dictionary
    term_dict: RwLock<HashMap<String, TermMeta>>,
    /// Posting lists using Roaring bitmaps
    postings: RwLock<HashMap<String, RoaringBitmap>>,
    /// Term frequencies: term -> [freq per doc]
    term_freqs: RwLock<HashMap<String, Vec<(u32, u16)>>>,
    /// Document lengths
    doc_lengths: RwLock<Vec<u16>>,
    /// External document IDs
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
}

impl RoaringBm25Profile {
    pub fn new() -> Self {
        Self {
            term_dict: RwLock::new(HashMap::new()),
            postings: RwLock::new(HashMap::new()),
            term_freqs: RwLock::new(HashMap::new()),
            doc_lengths: RwLock::new(Vec::new()),
            doc_ids: RwLock::new(Vec::new()),
            doc_count: RwLock::new(0),
            total_doc_length: RwLock::new(0),
            bm25: Bm25Params::default(),
            tokenizer: FastTokenizer::default(),
            pending: RwLock::new(Vec::new()),
        }
    }

    fn build_index(&self) {
        let pending = self.pending.read();
        if pending.is_empty() {
            return;
        }

        let mut term_dict = self.term_dict.write();
        let mut postings = self.postings.write();
        let mut term_freqs = self.term_freqs.write();
        let mut doc_lengths = self.doc_lengths.write();
        let mut doc_ids = self.doc_ids.write();
        let mut doc_count = self.doc_count.write();
        let mut total_doc_length = self.total_doc_length.write();

        let base_doc_id = *doc_count as u32;

        // Collect postings
        let mut new_postings: HashMap<String, Vec<(u32, u16)>> = HashMap::new();

        for (i, (ext_id, tfs, doc_len)) in pending.iter().enumerate() {
            let doc_id = base_doc_id + i as u32;
            doc_ids.push(ext_id.clone());
            doc_lengths.push(*doc_len as u16);
            *total_doc_length += *doc_len as u64;

            for (term, freq) in tfs {
                new_postings
                    .entry(term.clone())
                    .or_default()
                    .push((doc_id, *freq));
            }
        }

        *doc_count += pending.len() as u64;
        let total_docs = *doc_count as f32;

        // Merge into main index
        for (term, posts) in new_postings {
            let bitmap = postings.entry(term.clone()).or_default();
            let freqs = term_freqs.entry(term.clone()).or_default();

            for (doc_id, freq) in posts {
                bitmap.insert(doc_id);
                freqs.push((doc_id, freq));
            }

            // Update term metadata
            let df = bitmap.len() as u32;
            let idf = ((total_docs - df as f32 + 0.5) / (df as f32 + 0.5) + 1.0).ln();
            term_dict.insert(term, TermMeta { df, idf });
        }
    }

    fn search_roaring(
        &self,
        query_terms: &[String],
        limit: usize,
        offset: usize,
    ) -> Vec<SearchHit> {
        let term_dict = self.term_dict.read();
        let postings = self.postings.read();
        let term_freqs = self.term_freqs.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = *self.doc_count.read();

        if doc_count == 0 || query_terms.is_empty() {
            return Vec::new();
        }

        let total_docs = doc_count as f32;
        let total_doc_length = *self.total_doc_length.read();
        let avg_doc_len = total_doc_length as f32 / doc_count as f32;

        // Collect bitmaps for query terms
        let mut query_bitmaps: Vec<(&RoaringBitmap, &str, &TermMeta)> = Vec::new();
        for term in query_terms {
            if let (Some(bitmap), Some(meta)) = (postings.get(term), term_dict.get(term)) {
                query_bitmaps.push((bitmap, term, meta));
            }
        }

        if query_bitmaps.is_empty() {
            return Vec::new();
        }

        // Intersect bitmaps (OR for multiple query terms)
        let mut result_bitmap = query_bitmaps[0].0.clone();
        for (bitmap, _, _) in query_bitmaps.iter().skip(1) {
            result_bitmap |= *bitmap;
        }

        // Score documents
        let k = limit + offset;
        let mut top_k: BinaryHeap<Reverse<(OrderedFloat, u32)>> = BinaryHeap::with_capacity(k + 1);

        // Build freq lookup for efficiency
        let freq_lookups: HashMap<&str, HashMap<u32, u16>> = query_bitmaps
            .iter()
            .map(|(_, term, _)| {
                let lookup: HashMap<u32, u16> = term_freqs
                    .get(*term)
                    .map(|v| v.iter().cloned().collect())
                    .unwrap_or_default();
                (*term, lookup)
            })
            .collect();

        for doc_id in result_bitmap.iter() {
            let doc_len = doc_lengths[doc_id as usize] as f32;
            let mut score = 0.0f32;

            for (_, term, meta) in &query_bitmaps {
                if let Some(freq_map) = freq_lookups.get(term) {
                    if let Some(&freq) = freq_map.get(&doc_id) {
                        score += self.bm25.score(
                            freq as f32,
                            meta.df as f32,
                            doc_len,
                            avg_doc_len,
                            total_docs,
                        );
                    }
                }
            }

            let entry = Reverse((OrderedFloat(score), doc_id));
            if top_k.len() < k {
                top_k.push(entry);
            } else if let Some(Reverse((min_score, _))) = top_k.peek() {
                if score > min_score.0 {
                    top_k.pop();
                    top_k.push(entry);
                }
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

impl Default for RoaringBm25Profile {
    fn default() -> Self {
        Self::new()
    }
}

impl SearchProfile for RoaringBm25Profile {
    fn name(&self) -> &'static str {
        "roaring_bm25"
    }

    fn profile_type(&self) -> ProfileType {
        ProfileType::RoaringBm25
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
        let hits = self.search_roaring(&query_terms, limit, offset);
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
        let term_freqs = self.term_freqs.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();

        let term_dict_bytes = term_dict.len() * (32 + std::mem::size_of::<TermMeta>());
        let postings_bytes: usize = postings.values().map(|b| b.serialized_size()).sum();
        let term_freqs_bytes: usize = term_freqs.values().map(|v| v.len() * 6).sum();
        let doc_lengths_bytes = doc_lengths.len() * 2;
        let doc_ids_bytes: usize = doc_ids.iter().map(|s| s.len()).sum();

        MemoryStats {
            index_bytes: (term_dict_bytes
                + postings_bytes
                + term_freqs_bytes
                + doc_lengths_bytes
                + doc_ids_bytes) as u64,
            term_dict_bytes: term_dict_bytes as u64,
            postings_bytes: (postings_bytes + term_freqs_bytes) as u64,
            docs_indexed: *self.doc_count.read(),
            mmap_bytes: 0,
        }
    }

    fn save(&self, path: &Path) -> Result<(), IndexError> {
        let file = File::create(path.join("roaring_bm25.idx"))?;
        let mut writer = BufWriter::new(file);

        // Write header
        writer.write_all(b"ROAR")?;
        writer.write_all(&1u32.to_le_bytes())?;

        let term_dict = self.term_dict.read();
        let postings = self.postings.read();
        let term_freqs = self.term_freqs.read();
        let doc_lengths = self.doc_lengths.read();
        let doc_ids = self.doc_ids.read();
        let doc_count = *self.doc_count.read();
        let total_doc_length = *self.total_doc_length.read();

        // Write counts
        writer.write_all(&(term_dict.len() as u64).to_le_bytes())?;
        writer.write_all(&doc_count.to_le_bytes())?;
        writer.write_all(&total_doc_length.to_le_bytes())?;

        // Write term dict + postings + freqs
        for (term, meta) in term_dict.iter() {
            let term_bytes = term.as_bytes();
            writer.write_all(&(term_bytes.len() as u32).to_le_bytes())?;
            writer.write_all(term_bytes)?;
            writer.write_all(&meta.df.to_le_bytes())?;
            writer.write_all(&meta.idf.to_le_bytes())?;

            // Write bitmap
            if let Some(bitmap) = postings.get(term) {
                let mut bitmap_bytes = Vec::new();
                bitmap.serialize_into(&mut bitmap_bytes).unwrap();
                writer.write_all(&(bitmap_bytes.len() as u64).to_le_bytes())?;
                writer.write_all(&bitmap_bytes)?;
            } else {
                writer.write_all(&0u64.to_le_bytes())?;
            }

            // Write freqs
            if let Some(freqs) = term_freqs.get(term) {
                writer.write_all(&(freqs.len() as u64).to_le_bytes())?;
                for (doc_id, freq) in freqs {
                    writer.write_all(&doc_id.to_le_bytes())?;
                    writer.write_all(&freq.to_le_bytes())?;
                }
            } else {
                writer.write_all(&0u64.to_le_bytes())?;
            }
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
        let file = File::open(path.join("roaring_bm25.idx"))?;
        let mut reader = BufReader::new(file);

        let mut magic = [0u8; 4];
        reader.read_exact(&mut magic)?;
        if &magic != b"ROAR" {
            return Err(IndexError::Corrupted("Invalid magic".into()));
        }

        let mut buf4 = [0u8; 4];
        let mut buf8 = [0u8; 8];
        let mut buf2 = [0u8; 2];

        reader.read_exact(&mut buf4)?; // version

        reader.read_exact(&mut buf8)?;
        let term_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let doc_count = u64::from_le_bytes(buf8);
        reader.read_exact(&mut buf8)?;
        let total_doc_length = u64::from_le_bytes(buf8);

        let mut term_dict = HashMap::with_capacity(term_count as usize);
        let mut postings = HashMap::with_capacity(term_count as usize);
        let mut term_freqs = HashMap::with_capacity(term_count as usize);

        for _ in 0..term_count {
            reader.read_exact(&mut buf4)?;
            let term_len = u32::from_le_bytes(buf4) as usize;
            let mut term_bytes = vec![0u8; term_len];
            reader.read_exact(&mut term_bytes)?;
            let term = String::from_utf8(term_bytes)
                .map_err(|_| IndexError::Corrupted("Invalid UTF-8".into()))?;

            reader.read_exact(&mut buf4)?;
            let df = u32::from_le_bytes(buf4);
            reader.read_exact(&mut buf4)?;
            let idf = f32::from_le_bytes(buf4);

            term_dict.insert(term.clone(), TermMeta { df, idf });

            // Read bitmap
            reader.read_exact(&mut buf8)?;
            let bitmap_len = u64::from_le_bytes(buf8) as usize;
            if bitmap_len > 0 {
                let mut bitmap_bytes = vec![0u8; bitmap_len];
                reader.read_exact(&mut bitmap_bytes)?;
                let bitmap = RoaringBitmap::deserialize_from(&bitmap_bytes[..])
                    .map_err(|_| IndexError::Corrupted("Invalid bitmap".into()))?;
                postings.insert(term.clone(), bitmap);
            }

            // Read freqs
            reader.read_exact(&mut buf8)?;
            let freq_count = u64::from_le_bytes(buf8) as usize;
            let mut freqs = Vec::with_capacity(freq_count);
            for _ in 0..freq_count {
                reader.read_exact(&mut buf4)?;
                let doc_id = u32::from_le_bytes(buf4);
                reader.read_exact(&mut buf2)?;
                let freq = u16::from_le_bytes(buf2);
                freqs.push((doc_id, freq));
            }
            term_freqs.insert(term, freqs);
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
        *self.term_freqs.write() = term_freqs;
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
        self.term_freqs.write().clear();
        self.doc_lengths.write().clear();
        self.doc_ids.write().clear();
        *self.doc_count.write() = 0;
        *self.total_doc_length.write() = 0;
        self.pending.write().clear();
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
    fn test_roaring_index_search() {
        let mut profile = RoaringBm25Profile::new();
        let docs = vec![
            Document::new("1", "the quick brown fox"),
            Document::new("2", "the lazy dog"),
            Document::new("3", "quick brown dog"),
        ];

        profile.index_batch(&docs).unwrap();
        profile.commit().unwrap();

        let result = profile.search("quick dog", 10, 0).unwrap();
        assert!(result.hits.len() >= 2);
    }
}
