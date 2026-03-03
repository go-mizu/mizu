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
// Unicode tokenize → lowercase → stopword filter → stem → (term, position).
func analyzeWithPositions(text string) []token {
	words := tokenize(text)
	tokens := make([]token, 0, len(words)/2+1)
	pos := 0
	for _, w := range words {
		w = strings.ToLower(w)
		if len(w) < 2 || len(w) > 64 {
			continue
		}
		if stopwords[w] {
			pos++
			continue
		}
		stemmed := stem(w)
		if len(stemmed) < 2 {
			continue
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

var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "but": true, "by": true, "for": true, "if": true, "in": true,
	"into": true, "is": true, "it": true, "no": true, "not": true, "of": true,
	"on": true, "or": true, "such": true, "that": true, "the": true, "their": true,
	"then": true, "there": true, "these": true, "they": true, "this": true,
	"to": true, "was": true, "will": true, "with": true, "we": true, "he": true,
	"she": true, "his": true, "her": true, "its": true, "my": true, "your": true,
	"our": true, "you": true, "do": true, "does": true, "did": true, "has": true,
	"have": true, "had": true, "been": true, "would": true, "could": true,
	"should": true, "may": true, "might": true, "must": true, "shall": true,
	"can": true, "need": true, "dare": true, "ought": true, "used": true,
	"so": true, "than": true, "very": true, "just": true, "about": true,
	"above": true, "after": true, "again": true, "all": true, "also": true,
	"am": true, "any": true, "because": true, "before": true, "being": true,
	"below": true, "between": true, "both": true, "each": true, "few": true,
	"from": true, "further": true, "get": true, "got": true, "here": true,
	"him": true, "how": true, "i": true, "me": true, "more": true, "most": true,
	"much": true, "nor": true, "now": true, "only": true, "other": true,
	"over": true, "own": true, "re": true, "same": true, "some": true,
	"still": true, "too": true, "under": true, "until": true, "up": true,
	"us": true, "what": true, "when": true, "where": true, "which": true,
	"while": true, "who": true, "whom": true, "why": true, "down": true,
	"out": true, "off": true, "were": true, "those": true, "through": true,
	"during": true, "don": true, "doesn": true, "didn": true, "won": true,
	"ll": true, "ve": true,
}
