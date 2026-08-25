package diag

import (
	"iter"
	"slices"
	"strings"
	"unicode"
)

// Suggest returns the candidates worth offering for want, closest first, at
// most three of them.
//
// Doc 36 section 2.3 is the rule this implements: where mizu says "did you
// mean", the answer comes from a real candidate set rather than from a guess,
// and where nothing qualifies the answer is nothing. A wrong suggestion is
// worse than none, because it sends the reader down a false path with
// confidence, and the reader who follows it spends longer than the one who was
// told only that the name was not known.
//
// A candidate qualifies one of three ways.
//
// It is the same name in different case. That is the commonest mistake against
// a set of names somebody read somewhere, and it is worth answering first.
//
// It is within a small number of mistakes, counting an insertion, a deletion, a
// replacement, or two adjacent characters the wrong way round as one each. The
// number allowed is a third of the length of what was typed, and at least one,
// because one wrong letter in a long name is a typo and one wrong letter in a
// three letter name is usually a different word. Scaling it is also what lets
// database.max_conns find database.max_open_conns, which is five mistakes apart
// and unmistakably the same setting.
//
// One of the two names starts or ends with the whole of the other, and both are
// at least three characters long. That is the half-remembered name, where
// somebody writes pool and meant pool_size, and edit distance is no use because
// the two are five mistakes apart in a name too short to allow five.
//
// Sharing a prefix is not enough on its own. Every setting under database.
// shares nine characters with every other one, and answering an unknown
// database setting with three arbitrary siblings is the failure this whole
// function is written to avoid.
//
// Ordering is by number of mistakes and then alphabetical, so a typo beats a
// half-remembered name and two equally close candidates come out in the same
// order on every run.
//
// Only the closest candidates are returned. If one name is nearer than the
// rest, it is the answer on its own, and printing the runners up beside it asks
// the reader to weigh three names when one of them is right. More than one
// comes back only when they are the same distance away, which is the case where
// mizu genuinely does not know which was meant.
func Suggest(want string, candidates iter.Seq[string]) []string {
	if want == "" {
		return nil
	}
	w := fold(nil, want)
	limit := max(1, len(w)/3)

	type scored struct {
		name string
		at   int
	}
	// buf and rows are reused across the whole candidate set. A suggestion
	// measures one name against every setting a program declares, and these two
	// are the entire allocation cost of doing it.
	var buf []rune
	var rows [3][]int

	var found []scored
	for c := range candidates {
		if c == "" {
			continue
		}
		buf = fold(buf[:0], c)

		// A name that differs in length by more than the limit is further away
		// than the limit whatever its letters are, since every character of the
		// difference costs an insertion. Most candidates are ruled out here and
		// never reach the matrix.
		d := limit + 1
		if n := len(w) - len(buf); max(n, -n) <= limit {
			d = distanceRunes(w, buf, &rows)
		}

		switch {
		case d <= limit:
			found = append(found, scored{c, d})
		case shares(want, c):
			// Sorted after every real correction, and among themselves by
			// name, which is what the second key below does.
			found = append(found, scored{c, limit + 1})
		}
	}

	if len(found) == 0 {
		return nil
	}
	slices.SortFunc(found, func(a, b scored) int {
		if a.at != b.at {
			return a.at - b.at
		}
		return strings.Compare(a.name, b.name)
	})
	n := 0
	for n < len(found) && n < 3 && found[n].at == found[0].at {
		n++
	}
	found = found[:n]

	out := make([]string, len(found))
	for i, f := range found {
		out[i] = f.name
	}
	return out
}

// fold appends the characters of s to dst in lower case.
//
// It is here rather than strings.ToLower so that the result can go in a buffer
// the caller keeps, and because the comparison wants characters rather than
// bytes anyway.
func fold(dst []rune, s string) []rune {
	for _, r := range s {
		dst = append(dst, unicode.ToLower(r))
	}
	return dst
}

// minShared is how much of a name two of them have to have in common before one
// is worth offering for the other. Three, because two characters is most of the
// alphabet squared and one is not a memory of anything.
const minShared = 3

// shares reports whether either of a and b begins or ends with at least
// minShared characters of the other.
func shares(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	if len([]rune(a)) < minShared || len([]rune(b)) < minShared {
		return false
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a) ||
		strings.HasSuffix(a, b) || strings.HasSuffix(b, a)
}

// Distance is how many single character mistakes turn a into b, counting an
// insertion, a deletion, a replacement, or two adjacent characters the wrong
// way round as one each.
//
// The swap is the reason this is not plain Levenshtein. Two letters the wrong
// way round is the commonest typo there is, and counting it as one mistake
// rather than two is the difference between suggesting log for lgo and
// suggesting nothing.
//
// It counts characters rather than bytes, so a mistake in a name that is not
// ASCII costs what a mistake costs anywhere else.
func Distance(a, b string) int {
	if a == b {
		return 0
	}
	var rows [3][]int
	return distanceRunes([]rune(a), []rune(b), &rows)
}

// distanceRunes is Distance over runes the caller already has, working in rows
// the caller keeps. Both are for Suggest, which measures one name against every
// candidate and would otherwise convert and allocate for each of them.
//
// Three rows of the matrix are enough: a cell looks at the row above it, the
// cell to its left, and for a swap the row two above. Nothing is carried
// between calls except the space, since every cell a call reads is one it wrote
// first.
func distanceRunes(a, b []rune, rows *[3][]int) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	if cap(rows[0]) < len(b)+1 {
		for i := range rows {
			rows[i] = make([]int, len(b)+1)
		}
	}
	prev2 := rows[0][:len(b)+1]
	prev := rows[1][:len(b)+1]
	cur := rows[2][:len(b)+1]
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d := min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d = min(d, prev2[j-2]+1)
			}
			cur[j] = d
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(b)]
}

// Did is the sentence the suggestions go in, or empty when there are none.
//
// It is here so that an unknown setting, an unknown flag and an unknown command
// are all reported the same way, and so that the comma before the last one is
// argued about once. quote is what to wrap each name in, which is a pair of
// double quotes for a setting and the two dashes of a long flag for a flag.
func Did(names []string, quote func(string) string) string {
	if len(names) == 0 {
		return ""
	}
	if quote == nil {
		quote = func(s string) string { return s }
	}
	var b strings.Builder
	b.WriteString("did you mean ")
	for i, n := range names {
		switch {
		case i == 0:
		case i == len(names)-1:
			b.WriteString(" or ")
		default:
			b.WriteString(", ")
		}
		b.WriteString(quote(n))
	}
	b.WriteString("?")
	return b.String()
}
