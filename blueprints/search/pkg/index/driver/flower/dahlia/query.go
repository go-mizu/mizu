package dahlia

import "strings"

// Query types for the search evaluator.
type query interface {
	queryType() string
}

type termQuery struct {
	term string
}

func (q termQuery) queryType() string { return "term" }

type phraseQuery struct {
	terms []string
}

func (q phraseQuery) queryType() string { return "phrase" }

type booleanQuery struct {
	must    []query
	should  []query
	mustNot []query
}

func (q booleanQuery) queryType() string { return "boolean" }

// parseQuery parses a search query string into a query AST.
//
// Syntax:
//   - "+term" → must (required)
//   - "-term" → mustNot (excluded)
//   - "term"  → should (optional, boosts score)
//   - `"a b c"` → phrase query (exact sequence)
//   - "+a +b" → boolean with all must
//   - "a b"   → boolean with all should
func parseQuery(input string) query {
	input = strings.TrimSpace(input)
	if input == "" {
		return booleanQuery{}
	}

	var must, should, mustNot []query
	tokens := splitQueryTokens(input)

	for _, tok := range tokens {
		prefix := byte(0)
		if len(tok) > 1 && (tok[0] == '+' || tok[0] == '-') {
			prefix = tok[0]
			tok = tok[1:]
		}

		var queries []query
		if strings.HasPrefix(tok, "\"") && strings.HasSuffix(tok, "\"") && len(tok) > 2 {
			// Phrase query
			inner := tok[1 : len(tok)-1]
			terms := analyze(inner)
			if len(terms) == 0 {
				continue
			}
			if len(terms) == 1 {
				queries = append(queries, termQuery{term: terms[0]})
			} else {
				queries = append(queries, phraseQuery{terms: terms})
			}
		} else {
			terms := analyze(tok)
			if len(terms) == 0 {
				continue
			}
			for _, term := range terms {
				queries = append(queries, termQuery{term: term})
			}
		}

		switch prefix {
		case '+':
			must = append(must, queries...)
		case '-':
			mustNot = append(mustNot, queries...)
		default:
			should = append(should, queries...)
		}
	}

	// If everything is should and there's only one term, return it directly
	if len(must) == 0 && len(mustNot) == 0 && len(should) == 1 {
		return should[0]
	}

	return booleanQuery{must: must, should: should, mustNot: mustNot}
}

// splitQueryTokens splits a query string respecting quoted phrases.
func splitQueryTokens(input string) []string {
	var tokens []string
	i := 0
	for i < len(input) {
		// Skip whitespace
		for i < len(input) && input[i] == ' ' {
			i++
		}
		if i >= len(input) {
			break
		}

		// Check for +/- prefix before a quote
		start := i
		if i < len(input) && (input[i] == '+' || input[i] == '-') {
			if i+1 < len(input) && input[i+1] == '"' {
				// +"/- " prefix with quoted phrase
				prefix := string(input[i])
				i++ // skip + or -
				// Find closing quote
				i++ // skip opening "
				end := strings.IndexByte(input[i:], '"')
				if end >= 0 {
					tokens = append(tokens, prefix+"\""+input[i:i+end]+"\"")
					i += end + 1
				} else {
					tokens = append(tokens, prefix+"\""+input[i:]+"\"")
					i = len(input)
				}
				continue
			}
		}

		if input[i] == '"' {
			// Quoted phrase
			i++ // skip opening "
			end := strings.IndexByte(input[i:], '"')
			if end >= 0 {
				tokens = append(tokens, "\""+input[i:i+end]+"\"")
				i += end + 1
			} else {
				tokens = append(tokens, "\""+input[i:]+"\"")
				i = len(input)
			}
		} else {
			// Regular token
			end := i
			for end < len(input) && input[end] != ' ' {
				end++
			}
			tokens = append(tokens, input[start:end])
			i = end
		}
	}
	return tokens
}
