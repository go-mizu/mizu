// Package tokenizer provides text tokenization utilities for Vietnamese and other languages.
package tokenizer

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Vietnamese handles Vietnamese text processing.
// Vietnamese is syllable-based with spaces between syllables, making tokenization
// relatively straightforward compared to languages without word boundaries.
type Vietnamese struct {
	// StripAccents removes diacritical marks (tôi → toi)
	StripAccents bool
	// Lowercase converts text to lowercase
	Lowercase bool
}

// NewVietnamese creates a tokenizer with default settings.
func NewVietnamese() *Vietnamese {
	return &Vietnamese{
		StripAccents: false, // Preserve diacritics by default (important for Vietnamese)
		Lowercase:    true,
	}
}

// Token represents a parsed token from text or query.
type Token struct {
	// Term is a single word token
	Term string
	// Phrase is a quoted phrase (empty if single term)
	Phrase string
	// Position is the token position in the original text
	Position int
}

// IsPhrase returns true if this token is a phrase.
func (t Token) IsPhrase() bool {
	return t.Phrase != ""
}

// Value returns the token's text value.
func (t Token) Value() string {
	if t.Phrase != "" {
		return t.Phrase
	}
	return t.Term
}

// Tokenize splits Vietnamese text into searchable tokens.
func (v *Vietnamese) Tokenize(text string) []string {
	// 1. Normalize Unicode (NFC form)
	text = norm.NFC.String(text)

	// 2. Optionally strip diacritics
	if v.StripAccents {
		text = stripAccents(text)
	}

	// 3. Optionally lowercase
	if v.Lowercase {
		text = strings.ToLower(text)
	}

	// 4. Split on whitespace and punctuation
	tokens := splitOnBoundaries(text)

	// 5. Filter empty tokens
	return filterEmpty(tokens)
}

// TokenizeQuery processes search queries, handling quoted phrases.
// Example: "thành phố" Hồ Chí Minh
// Returns: [{Phrase: "thành phố"}, {Term: "hồ"}, {Term: "chí"}, {Term: "minh"}]
func (v *Vietnamese) TokenizeQuery(query string) []Token {
	// Normalize
	query = norm.NFC.String(query)
	if v.Lowercase {
		query = strings.ToLower(query)
	}

	var tokens []Token
	position := 0

	// Parse quoted phrases and individual terms
	i := 0
	for i < len(query) {
		// Skip whitespace
		for i < len(query) && unicode.IsSpace(rune(query[i])) {
			i++
		}
		if i >= len(query) {
			break
		}

		// Check for quoted phrase
		if query[i] == '"' {
			i++
			start := i
			for i < len(query) && query[i] != '"' {
				i++
			}
			if start < i {
				phrase := query[start:i]
				if v.StripAccents {
					phrase = stripAccents(phrase)
				}
				tokens = append(tokens, Token{Phrase: phrase, Position: position})
				position++
			}
			if i < len(query) {
				i++ // Skip closing quote
			}
			continue
		}

		// Regular term
		start := i
		for i < len(query) && !unicode.IsSpace(rune(query[i])) && query[i] != '"' {
			i++
		}
		if start < i {
			term := query[start:i]
			// Remove punctuation from term
			term = strings.TrimFunc(term, func(r rune) bool {
				return unicode.IsPunct(r)
			})
			if term != "" {
				if v.StripAccents {
					term = stripAccents(term)
				}
				tokens = append(tokens, Token{Term: term, Position: position})
				position++
			}
		}
	}

	return tokens
}

// TokenizeToTerms is a convenience method that returns just the term strings.
func (v *Vietnamese) TokenizeToTerms(query string) []string {
	tokens := v.TokenizeQuery(query)
	terms := make([]string, 0, len(tokens))
	for _, t := range tokens {
		terms = append(terms, t.Value())
	}
	return terms
}

// stripAccents removes diacritical marks from text.
// This is useful for accent-insensitive search.
func stripAccents(s string) string {
	// Use transform to remove combining marks (Mn category)
	t := transform.Chain(
		norm.NFD,                          // Decompose to base + combining chars
		runes.Remove(runes.In(unicode.Mn)), // Remove combining marks
		norm.NFC,                          // Recompose
	)
	result, _, err := transform.String(t, s)
	if err != nil {
		return s // Return original on error
	}
	return result
}

// splitOnBoundaries splits text on whitespace and punctuation.
func splitOnBoundaries(text string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsSpace(r) || isPunctuation(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// isPunctuation checks if a rune is punctuation.
// We're more permissive than unicode.IsPunct to handle Vietnamese better.
func isPunctuation(r rune) bool {
	if unicode.IsPunct(r) {
		return true
	}
	// Additional punctuation-like characters
	switch r {
	case '—', '–', '…', '·', '•':
		return true
	}
	return false
}

// filterEmpty removes empty strings from a slice.
func filterEmpty(tokens []string) []string {
	result := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

// NormalizeVietnamese normalizes Vietnamese text for consistent comparison.
func NormalizeVietnamese(text string) string {
	return strings.ToLower(norm.NFC.String(text))
}

// ContainsVietnamese checks if text contains Vietnamese characters.
func ContainsVietnamese(text string) bool {
	// Vietnamese-specific characters (not found in basic Latin)
	vietnameseChars := "àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩịòóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵđ"
	vietnameseChars += strings.ToUpper(vietnameseChars)

	for _, r := range text {
		if strings.ContainsRune(vietnameseChars, r) {
			return true
		}
	}
	return false
}
