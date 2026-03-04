package dahlia

import (
	"strings"
	"sync"
	"unicode"

	"github.com/kljensen/snowball/english"
)

type token struct {
	term string
	pos  int
}

var stemCache sync.Map

func stem(s string) string {
	if v, ok := stemCache.Load(s); ok {
		return v.(string)
	}
	stemmed := english.Stem(s, false)
	stemCache.Store(s, stemmed)
	return stemmed
}

// analyzeWithPositions tokenizes text through the full pipeline:
// Unicode tokenize → lowercase → stem → (term, position).
func analyzeWithPositions(text string) []token {
	words := tokenize(text)
	tokens := make([]token, 0, len(words))
	pos := 0
	for _, w := range words {
		w = strings.ToLower(w)
		if w == "" {
			continue
		}
		stemmed := stem(w)
		if stemmed == "" {
			stemmed = w
		}
		tokens = append(tokens, token{term: stemmed, pos: pos})
		pos++
	}
	return tokens
}

// analyze returns just the stemmed terms without positions.
func analyze(text string) []string {
	tokens := analyzeWithPositions(text)
	terms := make([]string, len(tokens))
	for i, t := range tokens {
		terms[i] = t.term
	}
	return terms
}

// tokenize splits text on non-letter/digit boundaries.
func tokenize(text string) []string {
	var tokens []string
	start := -1
	for i, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = i
			}
		} else {
			if start >= 0 {
				tokens = append(tokens, text[start:i])
				start = -1
			}
		}
	}
	if start >= 0 {
		tokens = append(tokens, text[start:])
	}
	return tokens
}
