package mizutest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParsePath(t *testing.T) {
	tests := map[string]struct {
		in   string
		want string // the steps spelled back out
	}{
		"the whole document":     {"$", ""},
		"nothing at all":         {"", ""},
		"a member":               {"$.data", ".data"},
		"a member with no $":     {"data", ".data"},
		"two members":            {"$.data.title", ".data.title"},
		"an index":               {"$.data[0]", ".data[0]"},
		"an index with no $":     {"data[0]", ".data[0]"},
		"a negative index":       {"$.data[-1]", ".data[-1]"},
		"an index of an index":   {"$[0][1]", "[0][1]"},
		"a quoted name":          {`$["content-type"]`, `["content-type"]`},
		"a name in single quote": {"$['content-type']", `["content-type"]`},
		"a quoted name with a dot in it": {
			`$["a.b"].c`, `["a.b"].c`,
		},
		"everything at once": {
			`$.data[0]["content-type"].charset`,
			`.data[0]["content-type"].charset`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			steps, err := parsePath(tt.in)
			if err != nil {
				t.Fatalf("parsePath(%q): %v", tt.in, err)
			}
			var b strings.Builder
			for _, s := range steps {
				b.WriteString(s.String())
			}
			if got := b.String(); got != tt.want {
				t.Errorf("parsePath(%q) spells back as %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestParsePathIsStrict is the point of the parser having errors at all. A path
// with a typo in it that quietly means something else is a test that passes for
// the wrong reason, which is the one thing worse than a test that fails.
func TestParsePathIsStrict(t *testing.T) {
	tests := map[string]struct{ in, want string }{
		"a dot with nothing after it": {"$.", "a dot with no name after it"},
		"two dots":                    {"$..data", "a dot with no name after it"},
		"an unclosed bracket":         {"$.data[0", "a [ with no ] after it"},
		"a wildcard":                  {"$.data[*]", "neither an index nor a quoted name"},
		"a slice":                     {"$.data[0:2]", "neither an index nor a quoted name"},
		"a missing dot":               {"$.data[0]title", "expected . or [ before"},
		"an unquoted name in brackets": {
			"$.data[title]", "neither an index nor a quoted name",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parsePath(tt.in)
			if err == nil {
				t.Fatalf("parsePath(%q) was accepted, want an error", tt.in)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("parsePath(%q) said %v, want it to mention %q", tt.in, err, tt.want)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	const doc = `{
		"data": [{"id": 1, "tags": ["go", "web"]}],
		"meta": {"content-type": "application/json"}
	}`

	v, err := decodeJSON([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		path string
		want string
	}{
		"a member":                 {"$.meta", `{"content-type":"application/json"}`},
		"an element":               {"$.data[0].id", "1"},
		"the last element":         {"$.data[0].tags[-1]", `"web"`},
		"a bracketed name":         {`$.meta["content-type"]`, `"application/json"`},
		"the whole document":       {"$", doc},
		"the whole document again": {"", doc},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := evaluate(v, tt.path)
			if err != nil {
				t.Fatalf("evaluate(%q): %v", tt.path, err)
			}
			want, err := decodeJSON([]byte(tt.want))
			if err != nil {
				t.Fatal(err)
			}
			if !same(got, want) {
				t.Errorf("evaluate(%q) = %v, want %v", tt.path, got, want)
			}
		})
	}
}

// TestDescribe is what every path failure is built from, and a message that
// says "not an object" without saying what it is instead is half an answer.
func TestDescribe(t *testing.T) {
	tests := map[string]struct {
		in   string
		want string
	}{
		"null":            {"null", "null"},
		"a boolean":       {"true", "the boolean true"},
		"a string":        {`"x"`, `the string "x"`},
		"a number":        {"1.5", "the number 1.5"},
		"a large number":  {"1234567890123456789", "the number 1234567890123456789"},
		"an empty object": {"{}", "an empty object"},
		"an object":       {`{"a":1,"b":2}`, "an object members a, b"},
		"one member":      {`{"a":1}`, "an object the member a"},
		"an array":        {"[1,2,3]", "an array of 3"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			v, err := decodeJSON([]byte(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if got := describe(v); got != tt.want {
				t.Errorf("describe(%s) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	if got := describe(float64(2)); got != "the number 2" {
		t.Errorf("describe of a float64 gave %q", got)
	}
	if got := describe(struct{}{}); !strings.Contains(got, "struct") {
		t.Errorf("describe of something unexpected gave %q", got)
	}
}

// TestMembersStopsListing keeps a failure about a wide object readable.
func TestMembersStopsListing(t *testing.T) {
	m := map[string]any{}
	for _, k := range "abcdefghijklmnopqrst" {
		m[string(k)] = 1
	}

	got := members(m)
	if !strings.Contains(got, "and 8 more") {
		t.Errorf("members of a wide object gave %q, want it cut short", got)
	}
	if strings.Contains(got, ", t") {
		t.Errorf("members listed the whole object: %q", got)
	}
	if got := members(nil); got != "no members" {
		t.Errorf("members of nothing gave %q", got)
	}
}

// TestPlainNameDecidesTheSpelling keeps an error message quoting a name only
// when a reader could not have written it after a dot.
func TestPlainNameDecidesTheSpelling(t *testing.T) {
	tests := map[string]bool{
		"data": true, "data_2": true, "ID": true, "x1": true,
		"content-type": false, "a.b": false, "": false, "a b": false,
	}
	for in, want := range tests {
		if got := plainName(in); got != want {
			t.Errorf("plainName(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestEvaluateReportsABadPath(t *testing.T) {
	v, err := decodeJSON([]byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := evaluate(v, "$.["); err == nil {
		t.Error("a path that will not parse was accepted")
	}
}

// TestEvaluateOnRawMessage keeps the walk working over a document a test built
// rather than one a handler sent.
func TestEvaluateOnRawMessage(t *testing.T) {
	v, err := normalizeValue(json.RawMessage(`{"a":{"b":[1,2]}}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := evaluate(v, "$.a.b[1]")
	if err != nil {
		t.Fatal(err)
	}
	if got.(json.Number).String() != "2" {
		t.Errorf("evaluate gave %v, want 2", got)
	}
}
