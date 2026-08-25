package mizutest

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Everything here works on documents rather than on Go values. A response body
// is bytes, the expected value in a test is a struct or a map, and comparing
// them directly means asking whether an int is a float64, which is a question
// about Go and not about the API. So both sides go through the same decoding
// first and the comparison happens after.

// decodeJSON reads a document with numbers kept as text.
//
// UseNumber is the whole point: through a float64, an id of nineteen digits
// comes back with zeroes on the end, and an id is exactly what a test asserts.
func decodeJSON(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// normalizeValue puts a Go value through the same decoding, so that what a test
// wrote down and what the handler sent are comparable.
//
// A json.RawMessage is a document already written and is decoded as one. A
// string is a JSON string and nothing else, even when it is full of braces.
// Guessing there would be a mistake: a handler that returns an escaped document
// inside a field is a real thing, and an assertion that cannot tell the string
// {"a":1} from the object it spells is one that passes when the field was
// meant to hold the other.
//
// So a test that wants to write its expected document as text says which it
// means:
//
//	res.AssertJSON(json.RawMessage(`{"a":1}`))  // the object
//	res.AssertJSONPath("$.raw", `{"a":1}`)      // the string
func normalizeValue(v any) (any, error) {
	switch b := v.(type) {
	case json.RawMessage:
		return decodeJSON(b)
	case string:
		return b, nil
	}

	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return decodeJSON(encoded)
}

// same reports whether two decoded documents are equal.
func same(a, b any) bool {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, xv := range x {
			yv, ok := y[k]
			if !ok || !same(xv, yv) {
				return false
			}
		}
		return true

	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !same(x[i], y[i]) {
				return false
			}
		}
		return true

	case json.Number:
		y, ok := b.(json.Number)
		return ok && sameNumber(x, y)

	default:
		return a == b
	}
}

// sameNumber compares two numbers by value rather than by spelling, so that 1,
// 1.0 and 1e0 are one number. Integers are compared as integers first, because
// an id of nineteen digits does not survive a float64 and a test that says two
// different ids are equal is worse than one that says nothing.
func sameNumber(a, b json.Number) bool {
	if a == b {
		return true
	}
	if x, err := strconv.ParseInt(a.String(), 10, 64); err == nil {
		if y, err := strconv.ParseInt(b.String(), 10, 64); err == nil {
			return x == y
		}
	}
	x, err1 := a.Float64()
	y, err2 := b.Float64()
	return err1 == nil && err2 == nil && x == y
}

// contains reports whether got holds want.
//
// An object matches when every member of want is there and matches, so a
// response with more in it than the test named is fine. An array has to be the
// same length and match element by element, because an assertion that a list
// contains something somewhere goes on passing after the list is wrong.
func contains(got, want any) bool {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for k, wv := range w {
			gv, ok := g[k]
			if !ok || !contains(gv, wv) {
				return false
			}
		}
		return true

	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for i := range w {
			if !contains(g[i], w[i]) {
				return false
			}
		}
		return true

	default:
		return same(got, want)
	}
}

// pretty writes a decoded document back out for a failure message. It is one
// line when it fits and indented when it does not, since a short value inline
// reads better and a long one does not.
func pretty(v any) string {
	compact, err := marshal(v, "")
	if err != nil {
		return "(will not encode: " + err.Error() + ")"
	}
	if len(compact) <= 72 && !strings.Contains(compact, "\n") {
		return compact
	}
	indented, err := marshal(v, "  ")
	if err != nil {
		return compact
	}
	return "\n" + indented
}

// marshal encodes with members sorted and HTML left alone, so that a failure
// message is stable between runs and readable when it holds markup.
func marshal(v any, indent string) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(sorted(v)); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// sorted rebuilds a document with its objects ordered, which encoding/json does
// for a map already and does not for anything else it might be handed.
func sorted(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for _, k := range slices.Sorted(maps.Keys(x)) {
			out[k] = sorted(x[k])
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = sorted(x[i])
		}
		return out
	default:
		return v
	}
}
