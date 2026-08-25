package mizutest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeJSONKeepsALargeNumberExact(t *testing.T) {
	// Nineteen digits, which is an id a database hands out and not an unusual
	// one. Through a float64 it comes back with zeroes on the end.
	const id = "1234567890123456789"

	v, err := decodeJSON([]byte(`{"id":` + id + `}`))
	if err != nil {
		t.Fatal(err)
	}
	got := v.(map[string]any)["id"].(json.Number)
	if got.String() != id {
		t.Errorf("the id came back as %s, want %s", got, id)
	}
}

func TestDecodeJSONReportsSomethingItCannotRead(t *testing.T) {
	if _, err := decodeJSON([]byte("{not json")); err == nil {
		t.Error("a malformed document was accepted")
	}
}

func TestNormalizeValue(t *testing.T) {
	tests := map[string]struct {
		in   any
		want string // the same value written as a document
	}{
		"a struct": {
			struct {
				Title string `json:"title"`
			}{"Hello"},
			`{"title":"Hello"}`,
		},
		"a map":         {map[string]any{"n": 1}, `{"n":1}`},
		"a slice":       {[]int{1, 2}, `[1,2]`},
		"an int":        {42, `42`},
		"a large int":   {int64(1234567890123456789), `1234567890123456789`},
		"a bool":        {true, `true`},
		"nil":           {nil, `null`},
		"a raw message": {json.RawMessage(`{"a": 1}`), `{"a":1}`},
		"bytes":         {[]byte("hi"), `"aGk="`}, // encoding/json spells these base64
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := normalizeValue(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			want, err := decodeJSON([]byte(tt.want))
			if err != nil {
				t.Fatal(err)
			}
			if !same(got, want) {
				t.Errorf("normalizeValue(%v) = %v, want %v", tt.in, got, want)
			}
		})
	}
}

// TestAStringIsAStringAndNothingElse is the decision the whole comparison rests
// on. A handler that returns an escaped document inside a field is a real
// thing, and an assertion that cannot tell the string {"a":1} from the object
// it spells passes when the field held the other one.
func TestAStringIsAStringAndNothingElse(t *testing.T) {
	asString, err := normalizeValue(`{"a":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := asString.(string); !ok {
		t.Fatalf("a string was decoded as %T, want it left alone", asString)
	}

	asDocument, err := normalizeValue(json.RawMessage(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := asDocument.(map[string]any); !ok {
		t.Fatalf("a raw message was kept as %T, want it decoded", asDocument)
	}

	if same(asString, asDocument) {
		t.Error("the string and the object it spells compare equal")
	}
}

func TestNormalizeValueReportsAValueItCannotEncode(t *testing.T) {
	if _, err := normalizeValue(make(chan int)); err == nil {
		t.Error("a channel was accepted")
	}
	if _, err := normalizeValue(json.RawMessage("{not json")); err == nil {
		t.Error("a malformed raw message was accepted")
	}
}

func TestSame(t *testing.T) {
	tests := map[string]struct {
		a, b string
		want bool
	}{
		"two objects":            {`{"a":1,"b":2}`, `{"b":2,"a":1}`, true},
		"a missing member":       {`{"a":1}`, `{"a":1,"b":2}`, false},
		"an extra member":        {`{"a":1,"b":2}`, `{"a":1}`, false},
		"a different member":     {`{"a":1}`, `{"b":1}`, false},
		"two arrays":             {`[1,2]`, `[1,2]`, true},
		"a shorter array":        {`[1]`, `[1,2]`, false},
		"a reordered array":      {`[1,2]`, `[2,1]`, false},
		"an object and an array": {`{}`, `[]`, false},
		"an array and a string":  {`[]`, `""`, false},
		"nesting":                {`{"a":[{"b":1}]}`, `{"a":[{"b":1}]}`, true},
		"nesting that differs":   {`{"a":[{"b":1}]}`, `{"a":[{"b":2}]}`, false},
		"two strings":            {`"x"`, `"x"`, true},
		"two booleans":           {`true`, `true`, true},
		"a boolean and a string": {`true`, `"true"`, false},
		"null and null":          {`null`, `null`, true},
		"null and something":     {`null`, `0`, false},
		"a number and a string":  {`1`, `"1"`, false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := sameDocs(t, tt.a, tt.b); got != tt.want {
				t.Errorf("same(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestSameNumber is where a comparison over documents earns its keep. A number
// is a value rather than a spelling, and an id longer than a float64 holds is
// still a value.
func TestSameNumber(t *testing.T) {
	tests := map[string]struct {
		a, b string
		want bool
	}{
		"the same digits":            {`1`, `1`, true},
		"an integer and a decimal":   {`1`, `1.0`, true},
		"an integer and an exponent": {`1`, `1e0`, true},
		"a decimal and an exponent":  {`0.5`, `5e-1`, true},
		"two different numbers":      {`1`, `2`, false},
		"a large id":                 {`1234567890123456789`, `1234567890123456789`, true},
		"two ids a float64 confuses": {`1234567890123456789`, `1234567890123456780`, false},
		"a negative":                 {`-1`, `-1.0`, true},
		"a huge number":              {`1e400`, `1e400`, true},
		"two huge numbers":           {`1e400`, `2e400`, false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := sameDocs(t, tt.a, tt.b); got != tt.want {
				t.Errorf("same(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := map[string]struct {
		got, want string
		ok        bool
	}{
		"a member of an object":    {`{"a":1,"b":2}`, `{"a":1}`, true},
		"every member":             {`{"a":1,"b":2}`, `{"a":1,"b":2}`, true},
		"nothing at all":           {`{"a":1}`, `{}`, true},
		"a member that is not out": {`{"a":1}`, `{"c":1}`, false},
		"a member with the wrong value": {
			`{"a":1}`, `{"a":2}`, false,
		},
		"a member of a nested object": {
			`{"data":{"id":1,"title":"x"}}`, `{"data":{"id":1}}`, true,
		},
		"an object inside an array": {
			`[{"id":1,"title":"x"}]`, `[{"id":1}]`, true,
		},
		"an array of the wrong length": {
			`[{"id":1},{"id":2}]`, `[{"id":1}]`, false,
		},
		"an array element that differs": {
			`[{"id":1},{"id":2}]`, `[{"id":1},{"id":3}]`, false,
		},
		"an object where an array was wanted": {`{}`, `[]`, false},
		"an array where an object was wanted": {`[]`, `{}`, false},
		"a plain value":                       {`1`, `1`, true},
		"a plain value that differs":          {`1`, `2`, false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, want := decode(t, tt.got), decode(t, tt.want)
			if ok := contains(got, want); ok != tt.ok {
				t.Errorf("contains(%s, %s) = %v, want %v", tt.got, tt.want, ok, tt.ok)
			}
		})
	}
}

func TestPretty(t *testing.T) {
	// Short enough to read on one line, so it stays on one.
	if got := pretty(decode(t, `{"b":2,"a":1}`)); got != `{"a":1,"b":2}` {
		t.Errorf("a short document printed as %q", got)
	}

	// Past the point where one line helps, indented and opening with a newline
	// so the first brace lines up with the rest.
	long := pretty(decode(t, `{"title":"a title long enough that this document does not fit on one line","id":1}`))
	if !strings.HasPrefix(long, "\n{\n  \"id\": 1,") {
		t.Errorf("a long document printed as %q", long)
	}

	if got := pretty(decode(t, "null")); got != "null" {
		t.Errorf("null printed as %q", got)
	}
	if got := pretty(make(chan int)); !strings.Contains(got, "will not encode") {
		t.Errorf("something that will not encode printed as %q", got)
	}
}

// TestPrettyLeavesMarkupAlone keeps a failure about an HTML body readable. The
// encoder escapes < and & by default, which turns a body a person could have
// read into one they have to decode in their head.
func TestPrettyLeavesMarkupAlone(t *testing.T) {
	got := pretty(decode(t, `{"body":"<b>hi</b> & bye"}`))
	if !strings.Contains(got, "<b>hi</b> & bye") {
		t.Errorf("markup printed as %s", got)
	}
}

// TestPrettySortsMembers is what makes a failure message the same on every run,
// which is what makes one worth reading.
func TestPrettySortsMembers(t *testing.T) {
	v := map[string]any{"c": 3, "a": map[string]any{"z": 1, "y": 2}, "b": []any{map[string]any{"n": 1, "m": 2}}}

	const want = `{"a":{"y":2,"z":1},"b":[{"m":2,"n":1}],"c":3}`
	if got := pretty(v); got != want {
		t.Errorf("pretty gave %s, want %s", got, want)
	}
}

func decode(t *testing.T, doc string) any {
	t.Helper()

	v, err := decodeJSON([]byte(doc))
	if err != nil {
		t.Fatalf("decoding %s: %v", doc, err)
	}
	return v
}

func sameDocs(t *testing.T, a, b string) bool {
	t.Helper()
	return same(decode(t, a), decode(t, b))
}
