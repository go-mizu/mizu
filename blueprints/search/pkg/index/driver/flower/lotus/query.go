package lotus

import "strings"

type query interface{ isQuery() }

type termQuery struct{ term string }
type phraseQuery struct{ terms []string }
type booleanQuery struct {
	must    []query
	should  []query
	mustNot []query
}

func (termQuery) isQuery()     {}
func (phraseQuery) isQuery()   {}
func (*booleanQuery) isQuery() {}

// parseQuery parses benchmark-compatible query strings:
//
//	"+a +b"    → BooleanQuery{must: [a, b]}
//	"a b"      → BooleanQuery{should: [a, b]}
//	"+a -b"    → BooleanQuery{must: [a], mustNot: [b]}
//	`"a b c"`  → PhraseQuery{terms: [a, b, c]}
func parseQuery(text string) query {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`) && len(text) > 2 {
		inner := text[1 : len(text)-1]
		terms := analyze(inner)
		if len(terms) == 0 {
			return &booleanQuery{}
		}
		return phraseQuery{terms: terms}
	}
	parts := strings.Fields(text)
	hasBoolOp := false
	for _, p := range parts {
		if strings.HasPrefix(p, "+") || strings.HasPrefix(p, "-") {
			hasBoolOp = true
			break
		}
	}
	bq := &booleanQuery{}
	for _, p := range parts {
		switch {
		case strings.HasPrefix(p, "+"):
			terms := analyze(p[1:])
			for _, t := range terms {
				bq.must = append(bq.must, termQuery{term: t})
			}
		case strings.HasPrefix(p, "-"):
			terms := analyze(p[1:])
			for _, t := range terms {
				bq.mustNot = append(bq.mustNot, termQuery{term: t})
			}
		default:
			terms := analyze(p)
			if hasBoolOp {
				for _, t := range terms {
					bq.must = append(bq.must, termQuery{term: t})
				}
			} else {
				for _, t := range terms {
					bq.should = append(bq.should, termQuery{term: t})
				}
			}
		}
	}
	return bq
}
