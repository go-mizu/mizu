package mizu

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// The tests in this file are the contract json_v1.go and json_v2.go are both
// held to. They run against whichever one was compiled in, so running the
// package twice, once with GOEXPERIMENT=nojsonv2, is what checks that the two
// agree. The CI workflow does that.

type jsonPayload struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func TestJSONDecodeStrict(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want jsonPayload
		ok   bool
	}{
		{"a value", `{"a":1,"b":"x"}`, jsonPayload{1, "x"}, true},
		{"trailing space", `{"a":1,"b":"x"}` + "\n\t ", jsonPayload{1, "x"}, true},
		{"a missing member", `{"a":1}`, jsonPayload{A: 1}, true},
		{"an empty object", `{}`, jsonPayload{}, true},
		{"a member with no field", `{"a":1,"c":2}`, jsonPayload{}, false},
		{"a second value", `{"a":1}{"a":2}`, jsonPayload{}, false},
		{"a second value on its own line", `{"a":1}` + "\n" + `{"a":2}`, jsonPayload{}, false},
		{"a second value that is not json", `{"a":1} nonsense`, jsonPayload{}, false},
		{"a fragment", `{"a":`, jsonPayload{}, false},
		{"nothing at all", ``, jsonPayload{}, false},
		{"not json", `hello`, jsonPayload{}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got jsonPayload
			err := jsonDecodeStrict(strings.NewReader(c.in), &got)

			if c.ok {
				if err != nil {
					t.Fatalf("jsonDecodeStrict(%q) = %v", c.in, err)
				}
				if got != c.want {
					t.Errorf("jsonDecodeStrict(%q) gave %+v, want %+v", c.in, got, c.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("jsonDecodeStrict(%q) returned nothing, and gave %+v", c.in, got)
			}
		})
	}
}

// TestJSONDecodeStrictReadsTheWholeValue checks that a value long enough to
// cross the decoder's read buffer arrives in one piece, which a table of short
// inputs would not catch.
func TestJSONDecodeStrictReadsTheWholeValue(t *testing.T) {
	want := jsonPayload{A: 7, B: strings.Repeat("a long string ", 2000)}

	in, err := jsonMarshal(want)
	if err != nil {
		t.Fatalf("jsonMarshal: %v", err)
	}

	var got jsonPayload
	if err := jsonDecodeStrict(bytes.NewReader(in), &got); err != nil {
		t.Fatalf("jsonDecodeStrict: %v", err)
	}
	if got != want {
		t.Errorf("a %d byte value came back as %d bytes", len(want.B), len(got.B))
	}
}

func TestJSONWrite(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"an object", map[string]int{"a": 1}, `{"a":1}` + "\n"},
		{"a struct", jsonPayload{1, "x"}, `{"a":1,"b":"x"}` + "\n"},
		{"a string", "hi", `"hi"` + "\n"},
		{"null", nil, "null\n"},
		{"an empty slice", []int{}, "[]\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := jsonWrite(&buf, c.in); err != nil {
				t.Fatalf("jsonWrite: %v", err)
			}
			if got := buf.String(); got != c.want {
				t.Errorf("jsonWrite(%#v) wrote %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestJSONLeavesHTMLAlone is the reason both files turn escaping off. A
// response with a JSON content type is not going into a script tag, and a
// client comparing the bytes against something another language produced should
// not find < in them.
func TestJSONLeavesHTMLAlone(t *testing.T) {
	const in = `<b>&</b>`
	const want = `{"h":"<b>&</b>"}`

	var buf bytes.Buffer
	if err := jsonWrite(&buf, map[string]string{"h": in}); err != nil {
		t.Fatalf("jsonWrite: %v", err)
	}
	if got := strings.TrimSuffix(buf.String(), "\n"); got != want {
		t.Errorf("jsonWrite gave %s, want %s", got, want)
	}

	b, err := jsonMarshal(map[string]string{"h": in})
	if err != nil {
		t.Fatalf("jsonMarshal: %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("jsonMarshal gave %s, want %s", got, want)
	}
}

// TestJSONSortsMapKeys is the other reason both files pass options rather than
// taking the defaults. The v2 encoder writes a Go map in whatever order the
// runtime hands it over, so without asking, the same handler would send a
// different body on every request and nothing comparing bytes would work.
//
// A hundred rounds is enough that a map of six keys in runtime order would not
// come out sorted every time.
func TestJSONSortsMapKeys(t *testing.T) {
	in := map[string]int{"f": 6, "d": 4, "a": 1, "e": 5, "c": 3, "b": 2}
	const want = `{"a":1,"b":2,"c":3,"d":4,"e":5,"f":6}`

	for range 100 {
		b, err := jsonMarshal(in)
		if err != nil {
			t.Fatalf("jsonMarshal: %v", err)
		}
		if got := string(b); got != want {
			t.Fatalf("jsonMarshal gave %s, want %s", got, want)
		}

		var buf bytes.Buffer
		if err := jsonWrite(&buf, in); err != nil {
			t.Fatalf("jsonWrite: %v", err)
		}
		if got := strings.TrimSuffix(buf.String(), "\n"); got != want {
			t.Fatalf("jsonWrite gave %s, want %s", got, want)
		}
	}
}

// TestJSONMarshalHasNoNewline is what the SSE loop depends on. An event is
// "data: " and the payload and a blank line, so a newline arriving inside the
// payload would end the event early.
func TestJSONMarshalHasNoNewline(t *testing.T) {
	for _, v := range []any{map[string]int{"a": 1}, jsonPayload{1, "x"}, "hi", nil, []int{}} {
		b, err := jsonMarshal(v)
		if err != nil {
			t.Fatalf("jsonMarshal(%#v): %v", v, err)
		}
		if bytes.ContainsAny(b, "\n") {
			t.Errorf("jsonMarshal(%#v) gave %q, which has a newline in it", v, b)
		}
	}
}

// TestJSONMarshalAndWriteAgree keeps the two output paths from drifting, since
// one is used for a response body and the other for an SSE payload and a reader
// should not be able to tell which produced what.
func TestJSONMarshalAndWriteAgree(t *testing.T) {
	values := []any{
		map[string]any{"a": 1, "b": "x", "c": nil},
		jsonPayload{2, "<y>"},
		[]any{1, "two", true, nil},
		"a string",
		3.5,
	}

	for _, v := range values {
		b, err := jsonMarshal(v)
		if err != nil {
			t.Fatalf("jsonMarshal(%#v): %v", v, err)
		}

		var buf bytes.Buffer
		if err := jsonWrite(&buf, v); err != nil {
			t.Fatalf("jsonWrite(%#v): %v", v, err)
		}

		if got, want := strings.TrimSuffix(buf.String(), "\n"), string(b); got != want {
			t.Errorf("jsonWrite(%#v) gave %s and jsonMarshal gave %s", v, got, want)
		}
	}
}

// TestJSONWriteReportsAWriteError checks the newline is not the one place an
// error goes missing, since the v2 side writes it in a second call.
func TestJSONWriteReportsAWriteError(t *testing.T) {
	for _, limit := range []int{0, 3} {
		w := &shortWriter{limit: limit}
		if err := jsonWrite(w, map[string]int{"a": 1}); err == nil {
			t.Errorf("a writer that accepts %d bytes gave no error", limit)
		}
	}
}

func BenchmarkJSONDecodeStrict(b *testing.B) {
	const in = `{"a":1,"b":"a reasonable sort of value"}`
	r := strings.NewReader(in)

	b.ReportAllocs()
	for b.Loop() {
		r.Reset(in)
		var p jsonPayload
		if err := jsonDecodeStrict(r, &p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONWrite(b *testing.B) {
	cases := map[string]any{
		"struct": jsonPayload{1, "a reasonable sort of value"},
		"map":    map[string]int{"f": 6, "d": 4, "a": 1, "e": 5, "c": 3, "b": 2},
	}

	for name, v := range cases {
		b.Run(name, func(b *testing.B) {
			var buf bytes.Buffer
			b.ReportAllocs()
			for b.Loop() {
				buf.Reset()
				if err := jsonWrite(&buf, v); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkJSONMarshal(b *testing.B) {
	v := jsonPayload{1, "a reasonable sort of value"}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := jsonMarshal(v); err != nil {
			b.Fatal(err)
		}
	}
}

// shortWriter accepts limit bytes and fails after that.
type shortWriter struct {
	limit int
	n     int
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if w.n+len(p) > w.limit {
		n := max(w.limit-w.n, 0)
		w.n += n
		return n, io.ErrShortWrite
	}
	w.n += len(p)
	return len(p), nil
}
