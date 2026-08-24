package golden

import (
	"strings"
	"testing"
)

func TestNormalizeJSONSortsMembers(t *testing.T) {
	got := string(normalizeJSON(t, []byte(`{"c":3,"a":1,"b":2}`)))
	want := "{\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3\n}\n"

	if got != want {
		t.Errorf("normalizeJSON gave\n%s\nwant\n%s", got, want)
	}
}

// TestNormalizeJSONKeepsLargeNumbers is the reason the decoder is told to use
// json.Number. A snowflake ID through a float64 comes back with zeroes on the
// end, and an ID is exactly what a golden file is checking.
func TestNormalizeJSONKeepsLargeNumbers(t *testing.T) {
	const id = "1234567890123456789"

	got := string(normalizeJSON(t, []byte(`{"id":`+id+`}`)))
	if !strings.Contains(got, id) {
		t.Errorf("normalizeJSON gave %s, want the id intact", got)
	}
}

// TestNormalizeJSONLeavesHTMLAlone matters because a golden file is read as
// text and an escaped angle bracket is unreadable.
func TestNormalizeJSONLeavesHTMLAlone(t *testing.T) {
	got := string(normalizeJSON(t, []byte(`{"html":"<b>&</b>"}`)))

	if !strings.Contains(got, "<b>&</b>") {
		t.Errorf("normalizeJSON gave %s, want the tags unescaped", got)
	}
}

// TestNormalizeJSONHandsBackWhatItCannotParse keeps the failure useful. A
// syntax error reported here would say a document is broken without showing it,
// and the diff shows it.
func TestNormalizeJSONHandsBackWhatItCannotParse(t *testing.T) {
	const broken = `{"a":`

	if got := string(normalizeJSON(t, []byte(broken))); got != broken {
		t.Errorf("normalizeJSON gave %q, want it handed back unchanged", got)
	}
	if got := string(normalizeJSON(t, nil)); got != "" {
		t.Errorf("normalizeJSON of nothing gave %q, want nothing", got)
	}
}

func TestAssertJSONMatchesAcrossMemberOrder(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { AssertJSON(r, []byte(`{"a":1,"b":[2,3]}`), dir) })
	AssertJSON(r, []byte(`{"b":[2,3],"a":1}`), dir)

	if r.failed {
		t.Errorf("the same document in a different member order did not match: %s", r.msg)
	}
}

// TestAssertJSONTakesAValueOrBytes is what makes it usable both from a test
// holding a struct and from one holding a response body.
func TestAssertJSONTakesAValueOrBytes(t *testing.T) {
	type payload struct {
		A int    `json:"a"`
		B string `json:"b"`
	}
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { AssertJSON(r, payload{1, "x"}, dir) })

	AssertJSON(r, []byte(`{"b":"x","a":1}`), dir)
	if r.failed {
		t.Errorf("bytes did not match the struct they encode: %s", r.msg)
	}

	AssertJSON(r, `{"a":1,"b":"x"}`, dir)
	if r.failed {
		t.Errorf("a string did not match: %s", r.msg)
	}
}

func TestAssertJSONReportsADifference(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { AssertJSON(r, []byte(`{"a":1,"b":2}`), dir) })
	AssertJSON(r, []byte(`{"a":1,"b":3}`), dir)

	r.says(t, `-   "b": 2`, `+   "b": 3`)
}

func TestNormalizeSQL(t *testing.T) {
	tests := map[string]struct{ in, want string }{
		"newlines and indentation": {
			in:   "select id,\n       name\n  from users\n where id = ?",
			want: "select id, name from users where id = ?",
		},
		"leading and trailing space": {
			in:   "   select 1   \n",
			want: "select 1",
		},
		"tabs": {
			in:   "select\t1",
			want: "select 1",
		},
		"a string literal keeps its spaces": {
			in:   "select 'a  b' from t",
			want: "select 'a  b' from t",
		},
		"a newline inside a literal survives": {
			in:   "select 'a\nb'",
			want: "select 'a\nb'",
		},
		"a quoted identifier keeps its spaces": {
			in:   `select "two  words" from t`,
			want: `select "two  words" from t`,
		},
		"a backtick identifier keeps its spaces": {
			in:   "select `two  words` from t",
			want: "select `two  words` from t",
		},
		"a doubled quote closes and reopens": {
			in:   "select 'it''s  here'  from t",
			want: "select 'it''s  here' from t",
		},
		"case is left alone": {
			in:   "SELECT   1",
			want: "SELECT 1",
		},
		"nothing": {in: "", want: ""},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := string(normalizeSQL(t, []byte(tt.in))); got != tt.want {
				t.Errorf("normalizeSQL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAssertSQLMatchesAcrossReindenting(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { AssertSQL(r, "select id from users where id = ?", dir) })
	AssertSQL(r, "select id\n  from users\n where id = ?", dir)

	if r.failed {
		t.Errorf("the same query reindented did not match: %s", r.msg)
	}
}

func TestAssertSQLStillSeesARealChange(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { AssertSQL(r, "select id from users", dir) })
	AssertSQL(r, "select id, name from users", dir)

	r.says(t, "select id from users", "select id, name from users")
}
