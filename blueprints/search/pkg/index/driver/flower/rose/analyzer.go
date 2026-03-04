package rose

import (
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/kljensen/snowball/english"
)

// tokenPool provides reusable scratch slices for analyze() to avoid per-call
// []string allocations.  The pool holds *[]string so the slice header itself
// is heap-stable across GC cycles.
var tokenPool = sync.Pool{New: func() any { s := make([]string, 0, 64); return &s }}

// stemCache maps lowercase token → stemmed form ("" = rejected by length/stopword filter).
// english.Stem is deterministic, so caching across calls is always correct.
var stemCache sync.Map

// ---------------------------------------------------------------------------
// Stopword list
// ---------------------------------------------------------------------------

// stopwordList is the canonical 127-word English stopword set, identical to
// Lucene's EnglishAnalyzer default set.  It is applied after lowercasing and
// before stemming so the stemmer never processes high-frequency stopwords.
var stopwordList = []string{
	"a", "an", "the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with",
	"by", "from", "as", "is", "was", "are", "were", "be", "been", "being", "have",
	"has", "had", "do", "does", "did", "will", "would", "could", "should", "may",
	"might", "shall", "can", "need", "dare", "ought", "used", "am", "it", "its",
	"he", "she", "they", "we", "you", "i", "me", "him", "her", "them", "us", "my",
	"your", "his", "their", "our", "this", "that", "these", "those", "what", "which",
	"who", "whom", "whose", "when", "where", "why", "how", "all", "both", "each",
	"few", "more", "most", "other", "some", "such", "no", "nor", "not", "only", "own",
	"same", "so", "than", "too", "very", "just", "because", "if", "then", "else",
	"while", "about", "against", "between", "into", "through", "during", "before",
	"after", "above", "below", "up", "down", "out", "off", "over", "under", "again",
	"further", "once", "here", "there", "any", "also", "etc", "ie", "eg", "via",
	"per", "vs", "re",
}

// stopwords is a O(1) lookup set built from stopwordList at init time.
var stopwords map[string]struct{}

func init() {
	stopwords = make(map[string]struct{}, len(stopwordList))
	for _, w := range stopwordList {
		stopwords[w] = struct{}{}
	}
}

// ---------------------------------------------------------------------------
// Token bounds
// ---------------------------------------------------------------------------

const (
	minTokLen = 2  // bytes, after lowercasing, before stemming
	maxTokLen = 64 // bytes, discard URL/hash tokens
)

// ---------------------------------------------------------------------------
// processTok applies the normalisation pipeline to a single pre-split token:
//
//  1. Unicode lowercase every rune.
//  2. Length check: discard if < minTokLen or > maxTokLen bytes.
//  3. Stopword filter: discard if the lowercased form is a stopword.
//  4. Snowball English stem.
//
// Returns "" when the token should be discarded.
// Used by snippetFor(); analyze() uses an inlined fast path instead.
func processTok(tok string) string {
	// Step 1: lowercase
	runes := []rune(tok)
	for i, r := range runes {
		runes[i] = unicode.ToLower(r)
	}
	lower := string(runes)

	// Step 2: length bounds (byte length of lowercased form)
	l := len(lower)
	if l < minTokLen || l > maxTokLen {
		return ""
	}

	// Step 3: stopword filter
	if _, ok := stopwords[lower]; ok {
		return ""
	}

	// Step 4: Snowball English stem (false = don't lowercase again)
	return english.Stem(lower, false)
}

// processLower applies steps 3–4 of the normalisation pipeline to an already-
// lowercased token of known valid length (minTokLen ≤ len(lower) ≤ maxTokLen).
// Checks stemCache before calling english.Stem; caches the result on miss.
// Returns "" when the token is a stopword or otherwise rejected.
func processLower(lower string) string {
	if v, ok := stemCache.Load(lower); ok {
		return v.(string)
	}
	var result string
	if _, ok := stopwords[lower]; !ok {
		result = english.Stem(lower, false)
	}
	stemCache.Store(lower, result)
	return result
}

// ---------------------------------------------------------------------------
// analyze runs the full text analysis pipeline on raw text and returns the
// resulting token slice (may contain duplicates — caller counts TF).
//
// Pipeline:
//
//	Raw text
//	  → Unicode tokenise  (split on non-letter, non-digit runes; range over string)
//	  → inline lowercase  (stack [64]byte buffer, avoids []rune + string allocs)
//	  → processLower      (stopword filter → stemCache → english.Stem)
//	  → return []string
//
// Opts applied: F (tokenPool), G (inline lowercase with stack buffer).
func analyze(text string) []string {
	if text == "" {
		return nil
	}

	// Borrow scratch builder from pool (Opt F).
	sp := tokenPool.Get().(*[]string)
	builder := (*sp)[:0]

	// Stack buffer for lowercasing the current token (Opt G).
	// maxTokLen+utf8.UTFMax ensures a single rune always fits even at the limit.
	var lowBuf [maxTokLen + utf8.UTFMax]byte
	inToken := false
	n := 0        // write cursor into lowBuf
	overflow := false // token exceeded maxTokLen

	for _, r := range text {
		isWordChar := unicode.IsLetter(r) || unicode.IsDigit(r)
		if isWordChar {
			if !inToken {
				inToken = true
				n = 0
				overflow = false
			}
			if !overflow {
				lc := unicode.ToLower(r)
				sz := utf8.RuneLen(lc)
				if sz < 0 {
					sz = utf8.RuneLen(utf8.RuneError)
				}
				if n+sz > maxTokLen {
					overflow = true
				} else {
					utf8.EncodeRune(lowBuf[n:], lc)
					n += sz
				}
			}
		} else if inToken {
			if !overflow && n >= minTokLen {
				if t := processLower(string(lowBuf[:n])); t != "" {
					builder = append(builder, t)
				}
			}
			inToken = false
		}
	}
	// Handle token that reaches end of string.
	if inToken && !overflow && n >= minTokLen {
		if t := processLower(string(lowBuf[:n])); t != "" {
			builder = append(builder, t)
		}
	}

	// Copy result to a fresh slice so we can return the builder to the pool.
	var tokens []string
	if len(builder) > 0 {
		tokens = make([]string, len(builder))
		copy(tokens, builder)
	}
	*sp = builder
	tokenPool.Put(sp)
	return tokens
}

// analyzeQuery is identical to analyze.  Both index-time and query-time
// analysis must produce the same token sequence for retrieval to be correct.
func analyzeQuery(text string) []string {
	return analyze(text)
}
