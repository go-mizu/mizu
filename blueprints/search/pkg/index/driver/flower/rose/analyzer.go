package rose

import (
	"unicode"

	"github.com/kljensen/snowball/english"
)

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

// ---------------------------------------------------------------------------
// analyze runs the full text analysis pipeline on raw text and returns the
// resulting token slice (may contain duplicates — caller counts TF).
//
// Pipeline:
//
//	Raw text
//	  → Unicode tokenise  (split on non-letter, non-digit runes)
//	  → processTok        (lowercase → length check → stopword → stem)
//	  → return []string
func analyze(text string) []string {
	if text == "" {
		return nil
	}

	var tokens []string

	runes := []rune(text)
	n := len(runes)
	inToken := false
	start := 0

	for i, r := range runes {
		isWordChar := unicode.IsLetter(r) || unicode.IsDigit(r)
		if isWordChar && !inToken {
			start = i
			inToken = true
		} else if !isWordChar && inToken {
			raw := string(runes[start:i])
			inToken = false
			if t := processTok(raw); t != "" {
				tokens = append(tokens, t)
			}
		}
	}
	// Handle token that reaches end of string.
	if inToken {
		raw := string(runes[start:n])
		if t := processTok(raw); t != "" {
			tokens = append(tokens, t)
		}
	}

	return tokens
}

// analyzeQuery is identical to analyze.  Both index-time and query-time
// analysis must produce the same token sequence for retrieval to be correct.
func analyzeQuery(text string) []string {
	return analyze(text)
}
