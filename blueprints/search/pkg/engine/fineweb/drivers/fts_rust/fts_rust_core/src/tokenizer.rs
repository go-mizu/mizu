//! Fast tokenization with zero-copy where possible

use std::collections::HashMap;

/// Fast byte-level tokenizer optimized for ASCII text
pub struct FastTokenizer {
    /// Minimum token length
    min_length: usize,
    /// Maximum token length
    max_length: usize,
}

impl Default for FastTokenizer {
    fn default() -> Self {
        Self {
            min_length: 2,
            max_length: 64,
        }
    }
}

impl FastTokenizer {
    pub fn new(min_length: usize, max_length: usize) -> Self {
        Self { min_length, max_length }
    }

    /// Tokenize text and count term frequencies
    /// Returns map of term -> frequency
    #[inline]
    pub fn tokenize_with_freqs(&self, text: &str) -> HashMap<String, u16> {
        let mut freqs: HashMap<String, u16> = HashMap::with_capacity(text.len() / 5);
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
                if len >= self.min_length && len <= self.max_length {
                    // Lowercase in-place
                    let token = self.normalize_token(&bytes[start..end]);
                    *freqs.entry(token).or_insert(0) += 1;
                }
                in_token = false;
            }
        }

        // Handle last token
        if in_token {
            let len = bytes.len() - start;
            if len >= self.min_length && len <= self.max_length {
                let token = self.normalize_token(&bytes[start..]);
                *freqs.entry(token).or_insert(0) += 1;
            }
        }

        freqs
    }

    /// Tokenize for query (returns unique terms)
    #[inline]
    pub fn tokenize_query(&self, query: &str) -> Vec<String> {
        let freqs = self.tokenize_with_freqs(query);
        freqs.into_keys().collect()
    }

    /// Normalize a token (lowercase ASCII)
    #[inline]
    fn normalize_token(&self, bytes: &[u8]) -> String {
        let mut result = String::with_capacity(bytes.len());
        for &b in bytes {
            result.push(b.to_ascii_lowercase() as char);
        }
        result
    }
}

/// Parallel tokenization for batch processing
pub fn tokenize_batch_parallel(
    texts: &[String],
    tokenizer: &FastTokenizer,
) -> Vec<HashMap<String, u16>> {
    use rayon::prelude::*;
    texts.par_iter()
        .map(|text| tokenizer.tokenize_with_freqs(text))
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_basic_tokenization() {
        let tokenizer = FastTokenizer::default();
        let freqs = tokenizer.tokenize_with_freqs("Hello World hello");
        assert_eq!(freqs.get("hello"), Some(&2));
        assert_eq!(freqs.get("world"), Some(&1));
    }

    #[test]
    fn test_min_length() {
        let tokenizer = FastTokenizer::new(3, 64);
        let freqs = tokenizer.tokenize_with_freqs("a ab abc");
        assert_eq!(freqs.get("a"), None);
        assert_eq!(freqs.get("ab"), None);
        assert_eq!(freqs.get("abc"), Some(&1));
    }

    #[test]
    fn test_query_tokenization() {
        let tokenizer = FastTokenizer::default();
        let terms = tokenizer.tokenize_query("Hello World");
        assert_eq!(terms.len(), 2);
        assert!(terms.contains(&"hello".to_string()));
        assert!(terms.contains(&"world".to_string()));
    }
}
