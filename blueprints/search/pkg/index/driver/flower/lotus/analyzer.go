package lotus

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/kljensen/snowball/english"
)

type token struct {
	term string
	pos  uint32 // 0-indexed absolute position in document
}

var stemCache sync.Map

// analyzeWithPositions tokenizes and stems text, returning (term, position) pairs.
// Position is the 0-indexed offset of the original token (before stopword removal).
func analyzeWithPositions(text string) []token {
	var tokens []token
	pos := uint32(0)
	var lowBuf [68]byte // maxTokLen(64) + utf8.UTFMax
	start := -1
	n := 0
	overflow := false

	flush := func() {
		defer func() {
			start = -1
			n = 0
			overflow = false
			pos++
		}()
		if start < 0 || overflow || n < 2 || n > 64 {
			return
		}
		low := string(lowBuf[:n])

		if cached, ok := stemCache.Load(low); ok {
			if s := cached.(string); s != "" {
				tokens = append(tokens, token{term: s, pos: pos})
			}
			return
		}

		if isStopword(low) {
			stemCache.Store(low, "")
			return
		}

		stemmed := english.Stem(low, false)
		if len(stemmed) < 2 {
			stemmed = low
		}
		stemCache.Store(low, stemmed)
		tokens = append(tokens, token{term: stemmed, pos: pos})
	}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = 1 // mark as started
				n = 0
				overflow = false
			}
			lr := unicode.ToLower(r)
			size := utf8.EncodeRune(lowBuf[n:n+utf8.UTFMax], lr)
			if n+size > 64 {
				overflow = true
			} else {
				n += size
			}
		} else {
			if start >= 0 {
				flush()
			}
		}
	}
	if start >= 0 {
		flush()
	}
	return tokens
}

// analyze returns just the stemmed terms (no positions).
func analyze(text string) []string {
	toks := analyzeWithPositions(text)
	result := make([]string, len(toks))
	for i, t := range toks {
		result[i] = t.term
	}
	return result
}

var stopwords = func() map[string]struct{} {
	words := strings.Fields(`a about above after again against all am an and any are
		as at be because been before being below between both but by can could did do
		does doing down during each few for from further get got had has have having
		he her here hers herself him himself his how i if in into is it its itself
		let me more most my myself no nor not of off on once only or other our ours
		ourselves out over own same she should so some such than that the their theirs
		them themselves then there these they this those through to too under until up
		very was we were what when where which while who whom why will with would you
		your yours yourself yourselves`)
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}()

func isStopword(s string) bool {
	_, ok := stopwords[s]
	return ok
}
