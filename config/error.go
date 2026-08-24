package config

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/toml"
)

// An Error is something wrong with a configuration source, at the place it
// went wrong.
//
// A configuration problem is nearly always one line of one file, and an error
// that names the line is the difference between fixing it and looking for it.
// Errors from the TOML parser arrive as this type too, so a caller has one
// thing to match on.
type Error struct {
	File string
	Line int
	Col  int // zero when the source has no columns, as .env files do not
	Msg  string
	Err  error // the underlying error, when there is one
}

func (e *Error) Error() string {
	var b strings.Builder
	if e.File != "" {
		b.WriteString(e.File)
		if e.Line > 0 {
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(e.Line))
			if e.Col > 0 {
				b.WriteByte(':')
				b.WriteString(strconv.Itoa(e.Col))
			}
		}
		b.WriteString(": ")
	}
	b.WriteString(e.Msg)
	return b.String()
}

func (e *Error) Unwrap() error { return e.Err }

// wrap turns a TOML error into a config error, so that reading configuration
// does not hand callers an error type from a package they never named.
func wrap(err error) error {
	var terr *toml.Error
	if errors.As(err, &terr) {
		return &Error{
			File: terr.Pos.File,
			Line: terr.Pos.Line,
			Col:  terr.Pos.Col,
			Msg:  terr.Msg,
			Err:  err,
		}
	}
	return err
}

// An Unknown is a setting that was written down but that nothing asked for.
//
// It is almost always a typo, and the whole point of reporting it is that a
// misspelled setting otherwise does nothing at all and says nothing about it.
type Unknown struct {
	Path string // the dotted path, as written
	From Source // the file and line, or the flag
	Near string // the closest setting that does exist, or empty
}

func (u Unknown) Error() string {
	var b strings.Builder
	b.WriteString(u.From.String())
	b.WriteString(": unknown setting ")
	b.WriteString(strconv.Quote(u.Path))
	if u.Near != "" {
		b.WriteString(", did you mean ")
		b.WriteString(strconv.Quote(u.Near))
		b.WriteString("?")
	}
	return b.String()
}

// nearest returns the candidate closest to want, or empty when none of them is
// close enough to be worth suggesting. The limit grows with the length of the
// name, because one wrong letter in a long name is a typo and one wrong letter
// in a three letter name is a different word.
func nearest(want string, candidates []string) string {
	limit := 1 + len(want)/4
	best, bestDist := "", limit+1
	for _, c := range candidates {
		// It takes at least one mistake per character of difference in
		// length, so a candidate that is much longer or shorter cannot win
		// and does not have to be measured.
		if len(c)-len(want) > limit || len(want)-len(c) > limit {
			continue
		}
		if d := distance(want, c); d < bestDist {
			best, bestDist = c, d
		}
	}
	if bestDist > limit {
		return ""
	}
	return best
}

// distance is how many mistakes turn one string into the other, counting an
// insertion, a deletion, a replacement, or two letters the wrong way round as
// one each. Configuration keys are names people type, so bytes and runes come
// to the same thing in every case that matters here.
func distance(a, b string) int {
	if a == b {
		return 0
	}
	// Three rows of the matrix are enough: a cell looks at the row above it,
	// the cell to its left, and for a swap the row two above.
	prev2 := make([]int, len(b)+1)
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
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
			// Two letters the wrong way round is the commonest typo there is,
			// and counting it as one mistake rather than two is what makes lgo
			// suggest log.
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d = min(d, prev2[j-2]+1)
			}
			cur[j] = d
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(b)]
}
