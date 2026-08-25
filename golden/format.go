package golden

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode"
)

// AssertJSON compares v against the golden file as JSON rather than as bytes.
//
//	golden.AssertJSON(t, res.Body())
//
// v is either a value to marshal or the JSON itself as a []byte, string or
// [json.RawMessage], which is what a response body usually is.
//
// Both sides are re-marshalled before comparing, so object members come out
// sorted and the indentation is this package's rather than whatever produced
// it. That is what stops a change of encoder, or a member moving in a struct
// definition, from showing up as a diff in every golden file at once.
//
// Numbers keep the digits they were written with. Round-tripping through
// float64 would turn a 19 digit ID into something ending in 000, and an ID is
// exactly the kind of thing a golden file is checking.
func AssertJSON(tb testing.TB, v any, opts ...Option) {
	tb.Helper()

	o := settle(tb, opts)
	o.normalize = normalizeJSON

	assert(tb, rawJSON(tb, v), o)
}

// rawJSON turns v into JSON bytes, leaving it alone if it already is some.
func rawJSON(tb testing.TB, v any) []byte {
	tb.Helper()

	switch v := v.(type) {
	case json.RawMessage:
		return v
	case []byte:
		return v
	case string:
		return []byte(v)
	}

	b, err := json.Marshal(v)
	if err != nil {
		tb.Fatalf("golden: cannot marshal %T: %v", v, err)
		return nil
	}
	return b
}

// normalizeJSON re-marshals b with sorted members and two-space indentation.
//
// Invalid JSON is handed back untouched rather than reported here, so the
// failure the reader gets is the diff against the golden file, which shows what
// the invalid thing was. Reporting a syntax error instead would say a document
// is broken without showing it.
func normalizeJSON(tb testing.TB, b []byte) []byte {
	tb.Helper()

	if len(bytes.TrimSpace(b)) == 0 {
		return b
	}

	dec := json.NewDecoder(bytes.NewReader(b))

	// UseNumber is what keeps a large integer intact. Without it every number
	// becomes a float64 on the way in and comes back out in whatever form that
	// round trip leaves it in.
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return b
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	// A golden file is read as text, and a response body is not written into a
	// script tag, so there is no reason for the encoder to spell < as <.
	enc.SetEscapeHTML(false)

	if err := enc.Encode(v); err != nil {
		return b
	}
	return buf.Bytes()
}

// AssertSQL compares query against the golden file with its whitespace
// normalised, so reindenting a query builder does not rewrite every golden file
// in the package.
//
//	golden.AssertSQL(t, q.String())
//
// A run of whitespace between two tokens becomes one space and the whole thing
// is trimmed. Nothing else changes: keywords keep their case, because a builder
// that starts emitting lowercase select is a change worth seeing, and a string
// literal keeps its spacing, because the spaces inside it are data.
func AssertSQL(tb testing.TB, query string, opts ...Option) {
	tb.Helper()

	o := settle(tb, opts)
	o.normalize = normalizeSQL

	assert(tb, []byte(query), o)
}

// normalizeSQL collapses the whitespace between tokens and leaves the
// whitespace inside quotes alone.
//
// It tracks quoting rather than running a regexp over the whole string, because
// a space inside 'a  b' is part of the value and collapsing it would change
// what the statement means. Doubling is how both SQL string literals and
// quoted identifiers escape their own quote, and it needs no special case: the
// closing quote ends one literal and the next one opens another.
func normalizeSQL(_ testing.TB, b []byte) []byte {
	var out strings.Builder
	out.Grow(len(b))

	var quote byte
	pendingSpace := false

	for i := range len(b) {
		c := b[i]

		if quote != 0 {
			out.WriteByte(c)
			if c == quote {
				quote = 0
			}
			continue
		}

		if unicode.IsSpace(rune(c)) {
			pendingSpace = out.Len() > 0
			continue
		}
		if pendingSpace {
			out.WriteByte(' ')
			pendingSpace = false
		}

		switch c {
		case '\'', '"', '`':
			quote = c
		}
		out.WriteByte(c)
	}
	return []byte(out.String())
}
