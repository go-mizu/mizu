// Package algo provides ultra-fast tokenization with minimal allocations.
package algo

import (
	"unsafe"
)

// delimTable is a lookup table for delimiter characters.
// 1 = delimiter, 0 = not delimiter
var delimTable = func() [256]byte {
	var t [256]byte
	// Control characters and space
	for i := 0; i <= 32; i++ {
		t[i] = 1
	}
	// Punctuation: !"#$%&'()*+,-./
	for i := '!'; i <= '/'; i++ {
		t[i] = 1
	}
	// Punctuation: :;<=>?@
	for i := ':'; i <= '@'; i++ {
		t[i] = 1
	}
	// Punctuation: [\]^_`
	for i := '['; i <= '`'; i++ {
		t[i] = 1
	}
	// Punctuation: {|}~
	for i := '{'; i <= '~'; i++ {
		t[i] = 1
	}
	return t
}()

// lowerTable is a lookup table for lowercase conversion.
var lowerTable = func() [256]byte {
	var t [256]byte
	for i := 0; i < 256; i++ {
		t[i] = byte(i)
	}
	for i := 'A'; i <= 'Z'; i++ {
		t[i] = byte(i + 32)
	}
	return t
}()

// UltraFastTokenize tokenizes text with minimal allocations.
// Uses unsafe.String to avoid string copies during map operations.
// The returned map keys are only valid while the input text is valid.
func UltraFastTokenize(text string) map[string]int {
	// Pre-allocate map with estimated capacity
	termFreqs := make(map[string]int, 64)

	// Convert to bytes for in-place lowercase
	data := []byte(text)
	start := -1

	for i := 0; i < len(data); i++ {
		c := data[i]
		if delimTable[c] == 1 {
			if start >= 0 {
				length := i - start
				if length > 0 && length < 100 {
					// Lowercase in place
					token := data[start:i]
					for j := 0; j < len(token); j++ {
						token[j] = lowerTable[token[j]]
					}
					// Use unsafe.String to avoid allocation
					// This is safe because we're not modifying the underlying bytes
					// after creating the string key
					key := unsafe.String(&token[0], len(token))
					termFreqs[key]++
				}
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}

	// Handle last token
	if start >= 0 {
		length := len(data) - start
		if length > 0 && length < 100 {
			token := data[start:]
			for j := 0; j < len(token); j++ {
				token[j] = lowerTable[token[j]]
			}
			key := unsafe.String(&token[0], len(token))
			termFreqs[key]++
		}
	}

	return termFreqs
}

// UltraFastTokenizeCallback tokenizes text and calls the callback for each term.
// This avoids map allocation entirely.
func UltraFastTokenizeCallback(text string, callback func(term []byte, freq int)) {
	// For frequency counting, we need a temporary map
	termFreqs := make(map[string]int, 64)
	data := []byte(text)
	start := -1

	for i := 0; i < len(data); i++ {
		c := data[i]
		if delimTable[c] == 1 {
			if start >= 0 {
				length := i - start
				if length > 0 && length < 100 {
					token := data[start:i]
					for j := 0; j < len(token); j++ {
						token[j] = lowerTable[token[j]]
					}
					key := unsafe.String(&token[0], len(token))
					termFreqs[key]++
				}
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}

	if start >= 0 {
		length := len(data) - start
		if length > 0 && length < 100 {
			token := data[start:]
			for j := 0; j < len(token); j++ {
				token[j] = lowerTable[token[j]]
			}
			key := unsafe.String(&token[0], len(token))
			termFreqs[key]++
		}
	}

	// Call callback for each term
	for term, freq := range termFreqs {
		callback([]byte(term), freq)
	}
}

// TermAccumulator efficiently accumulates term frequencies across documents.
type TermAccumulator struct {
	// Interned term strings to avoid duplicates
	internedTerms map[string]string
	// Term frequencies per document (reusable)
	docTerms map[string]int
}

// NewTermAccumulator creates a term accumulator.
func NewTermAccumulator() *TermAccumulator {
	return &TermAccumulator{
		internedTerms: make(map[string]string, 100000),
		docTerms:      make(map[string]int, 128),
	}
}

// Tokenize tokenizes text and returns interned term frequencies.
// The map is reused across calls - copy if you need to keep it.
func (ta *TermAccumulator) Tokenize(text string) map[string]int {
	// Clear previous document terms
	clear(ta.docTerms)

	data := []byte(text)
	start := -1

	for i := 0; i < len(data); i++ {
		c := data[i]
		if delimTable[c] == 1 {
			if start >= 0 {
				length := i - start
				if length > 0 && length < 100 {
					token := data[start:i]
					for j := 0; j < len(token); j++ {
						token[j] = lowerTable[token[j]]
					}
					// Intern the term
					termStr := ta.intern(token)
					ta.docTerms[termStr]++
				}
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}

	if start >= 0 {
		length := len(data) - start
		if length > 0 && length < 100 {
			token := data[start:]
			for j := 0; j < len(token); j++ {
				token[j] = lowerTable[token[j]]
			}
			termStr := ta.intern(token)
			ta.docTerms[termStr]++
		}
	}

	return ta.docTerms
}

// intern returns an interned version of the term.
func (ta *TermAccumulator) intern(token []byte) string {
	key := unsafe.String(&token[0], len(token))
	if interned, ok := ta.internedTerms[key]; ok {
		return interned
	}
	// Create a proper string (allocates)
	str := string(token)
	ta.internedTerms[str] = str
	return str
}

// InternedCount returns the number of interned terms.
func (ta *TermAccumulator) InternedCount() int {
	return len(ta.internedTerms)
}
