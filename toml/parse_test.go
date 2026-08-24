package toml

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func parse(t *testing.T, doc string) *Table {
	t.Helper()
	tab, err := Parse("test.toml", []byte(doc))
	if err != nil {
		t.Fatalf("%v\n\nin:\n%s", err, doc)
	}
	return tab
}

// check reads one value out of a document and compares it with what the test
// asked for. The type of want picks the kind, except for the four date and
// time kinds, which share a Go type and are given as a kind and a time.
func check(t *testing.T, tab *Table, path []string, want any) {
	t.Helper()
	v := tab.Lookup(path...)
	if v == nil {
		t.Fatalf("%s is not in the document", strings.Join(path, "."))
	}
	name := strings.Join(path, ".")
	switch want := want.(type) {
	case string:
		if v.Kind != KindString || v.Str != want {
			t.Errorf("%s is a %s %v, want the string %q", name, v.Kind, value(v), want)
		}
	case int64:
		if v.Kind != KindInt || v.Int != want {
			t.Errorf("%s is a %s %v, want the integer %d", name, v.Kind, value(v), want)
		}
	case float64:
		ok := v.Kind == KindFloat && (v.Float == want || math.IsNaN(want) && math.IsNaN(v.Float))
		if !ok {
			t.Errorf("%s is a %s %v, want the float %v", name, v.Kind, value(v), want)
		}
	case bool:
		if v.Kind != KindBool || v.Bool != want {
			t.Errorf("%s is a %s %v, want the boolean %v", name, v.Kind, value(v), want)
		}
	case time.Time:
		if !v.Time.Equal(want) {
			t.Errorf("%s is %s, want %s", name, v.Time.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
		}
	default:
		t.Fatalf("the test does not know how to check a %T", want)
	}
}

// value is the value a Value holds, for a failure message.
func value(v *Value) any {
	switch v.Kind {
	case KindString:
		return v.Str
	case KindInt:
		return v.Int
	case KindFloat:
		return v.Float
	case KindBool:
		return v.Bool
	case KindArray:
		return v.Array
	case KindTable:
		return v.Table.Keys()
	}
	return v.Time
}

func date(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseEmpty(t *testing.T) {
	for _, doc := range []string{"", "\n", "  \n\t\n", "# nothing here\n", "# no newline at the end"} {
		tab := parse(t, doc)
		if tab.Len() != 0 {
			t.Errorf("%q parsed to %v, want an empty table", doc, tab.Keys())
		}
	}
}

func TestParseKeys(t *testing.T) {
	tests := []struct {
		doc  string
		path []string
	}{
		{`key = "v"`, []string{"key"}},
		{`bare_key = "v"`, []string{"bare_key"}},
		{`bare-key = "v"`, []string{"bare-key"}},
		{`1234 = "v"`, []string{"1234"}},
		{`"127.0.0.1" = "v"`, []string{"127.0.0.1"}},
		{`"ʎǝʞ" = "v"`, []string{"ʎǝʞ"}},
		{`'quoted "value"' = "v"`, []string{`quoted "value"`}},
		{`"" = "v"`, []string{""}},
		{`'' = "v"`, []string{""}},
		{`a.b.c = "v"`, []string{"a", "b", "c"}},
		{`a . b . c = "v"`, []string{"a", "b", "c"}},
		{`site."google.com" = "v"`, []string{"site", "google.com"}},
		{"3.14159 = \"v\"", []string{"3", "14159"}},
	}
	for _, tt := range tests {
		check(t, parse(t, tt.doc), tt.path, "v")
	}
}

func TestParseArrays(t *testing.T) {
	tab := parse(t, `
empty = []
ints = [1, 2, 3]
trailing = [1, 2, 3,]
mixed = ["a", 1, true]
nested = [[1, 2], ["a"]]
spread = [
	1,
	# a comment in the middle
	2,
]
tables = [{a = 1}, {a = 2}]
`)
	if v := tab.Get("empty"); v.Kind != KindArray || len(v.Array) != 0 {
		t.Errorf("empty is %v, want an empty array", value(v))
	}
	for _, name := range []string{"ints", "trailing"} {
		v := tab.Get(name)
		if v.Kind != KindArray || len(v.Array) != 3 {
			t.Fatalf("%s is a %s with %d elements, want an array of 3", name, v.Kind, len(v.Array))
		}
		for i, el := range v.Array {
			if el.Kind != KindInt || el.Int != int64(i+1) {
				t.Errorf("%s[%d] is %v, want %d", name, i, value(el), i+1)
			}
		}
	}
	if v := tab.Get("mixed"); len(v.Array) != 3 || v.Array[0].Kind != KindString || v.Array[1].Kind != KindInt || v.Array[2].Kind != KindBool {
		t.Errorf("mixed did not keep the kinds of its elements")
	}
	if v := tab.Get("nested"); len(v.Array) != 2 || len(v.Array[0].Array) != 2 || len(v.Array[1].Array) != 1 {
		t.Errorf("nested is %v, want [[1 2] [a]]", value(v))
	}
	if v := tab.Get("spread"); len(v.Array) != 2 {
		t.Errorf("spread has %d elements, want 2", len(v.Array))
	}
	if v := tab.Get("tables"); len(v.Array) != 2 || v.Array[1].Table.Get("a").Int != 2 {
		t.Errorf("tables is %v, want two inline tables", value(v))
	}
}

func TestParseInlineTables(t *testing.T) {
	tab := parse(t, `
empty = {}
point = {x = 1, y = 2}
nested = {a = {b = "c"}}
dotted = {a.b = "c"}
`)
	if v := tab.Get("empty"); v.Kind != KindTable || v.Table.Len() != 0 {
		t.Errorf("empty is %v, want an empty table", value(v))
	}
	check(t, tab, []string{"point", "x"}, int64(1))
	check(t, tab, []string{"point", "y"}, int64(2))
	check(t, tab, []string{"nested", "a", "b"}, "c")
	check(t, tab, []string{"dotted", "a", "b"}, "c")
}

func TestParseTables(t *testing.T) {
	tab := parse(t, `
[server]
host = "localhost"
port = 8080

[server.tls]
cert = "cert.pem"

[log]
level = "info"

# A header for a table that already exists implicitly is fine.
[a.b.c]
here = true
[a]
also = true
`)
	check(t, tab, []string{"server", "host"}, "localhost")
	check(t, tab, []string{"server", "port"}, int64(8080))
	check(t, tab, []string{"server", "tls", "cert"}, "cert.pem")
	check(t, tab, []string{"log", "level"}, "info")
	check(t, tab, []string{"a", "b", "c", "here"}, true)
	check(t, tab, []string{"a", "also"}, true)

	if got, want := tab.Keys(), []string{"server", "log", "a"}; !slices.Equal(got, want) {
		t.Errorf("the top level keys are %v, want %v", got, want)
	}
}

func TestParseArraysOfTables(t *testing.T) {
	tab := parse(t, `
[[fruit]]
name = "apple"

[fruit.physical]
colour = "red"

[[fruit.variety]]
name = "red delicious"

[[fruit.variety]]
name = "granny smith"

[[fruit]]
name = "banana"

[[fruit.variety]]
name = "plantain"
`)
	v := tab.Get("fruit")
	if v.Kind != KindArray || len(v.Array) != 2 {
		t.Fatalf("fruit is %v, want an array of 2", value(v))
	}
	check(t, v.Array[0].Table, []string{"name"}, "apple")
	check(t, v.Array[0].Table, []string{"physical", "colour"}, "red")
	check(t, v.Array[1].Table, []string{"name"}, "banana")

	varieties := v.Array[0].Table.Get("variety")
	if len(varieties.Array) != 2 {
		t.Fatalf("the first fruit has %d varieties, want 2", len(varieties.Array))
	}
	check(t, varieties.Array[0].Table, []string{"name"}, "red delicious")
	check(t, varieties.Array[1].Table, []string{"name"}, "granny smith")

	if got := v.Array[1].Table.Get("variety"); len(got.Array) != 1 {
		t.Errorf("the second fruit has %d varieties, want 1", len(got.Array))
	}
}

func TestParseKeepsDocumentOrder(t *testing.T) {
	tab := parse(t, "z = 1\na = 2\nm = 3\n[b]\nq = 1\nc = 2\n")
	if got, want := tab.Keys(), []string{"z", "a", "m", "b"}; !slices.Equal(got, want) {
		t.Errorf("the keys are %v, want %v", got, want)
	}
	if got, want := tab.Get("b").Table.Keys(), []string{"q", "c"}; !slices.Equal(got, want) {
		t.Errorf("the keys of b are %v, want %v", got, want)
	}

	var seen []string
	for k := range tab.All() {
		seen = append(seen, k)
	}
	if got, want := seen, []string{"z", "a", "m", "b"}; !slices.Equal(got, want) {
		t.Errorf("All gave %v, want %v", got, want)
	}
}

func TestParsePositions(t *testing.T) {
	tab := parse(t, "a = 1\nb = 2\n\n[t]\n  c = 3\n")
	tests := []struct {
		path      []string
		line, col int
	}{
		{[]string{"a"}, 1, 5},
		{[]string{"b"}, 2, 5},
		{[]string{"t"}, 4, 1},
		{[]string{"t", "c"}, 5, 7},
	}
	for _, tt := range tests {
		v := tab.Lookup(tt.path...)
		if v.Pos.Line != tt.line || v.Pos.Col != tt.col {
			t.Errorf("%s is at %s, want test.toml:%d:%d", strings.Join(tt.path, "."), v.Pos, tt.line, tt.col)
		}
		if v.Pos.File != "test.toml" {
			t.Errorf("%s says it is in %q", strings.Join(tt.path, "."), v.Pos.File)
		}
	}
}

func TestParseComments(t *testing.T) {
	tab := parse(t, `
# The whole file is about this.
a = 1 # and this one is about a
# and this is about nothing at all

[t] # even here
b = 2
`)
	check(t, tab, []string{"a"}, int64(1))
	check(t, tab, []string{"t", "b"}, int64(2))
}

func TestParseCRLF(t *testing.T) {
	tab := parse(t, "a = 1\r\n[t]\r\nb = 2\r\n")
	check(t, tab, []string{"a"}, int64(1))
	check(t, tab, []string{"t", "b"}, int64(2))
	if v := tab.Lookup("t", "b"); v.Pos.Line != 3 {
		t.Errorf("b is on line %d, want 3", v.Pos.Line)
	}
}

func TestParseByteOrderMark(t *testing.T) {
	tab := parse(t, "\ufeffa = 1\n")
	check(t, tab, []string{"a"}, int64(1))
}

// The example from the front page of toml.io, which is the closest thing the
// format has to a conformance test everybody has seen.
func TestParseTheExample(t *testing.T) {
	tab := parse(t, `
# This is a TOML document

title = "TOML Example"

[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00

[database]
enabled = true
ports = [ 8000, 8001, 8002 ]
data = [ ["delta", "phi"], [3.14] ]
temp_targets = { cpu = 79.5, case = 72.0 }

[servers]

[servers.alpha]
ip = "10.0.0.1"
role = "frontend"

[servers.beta]
ip = "10.0.0.2"
role = "backend"
`)
	check(t, tab, []string{"title"}, "TOML Example")
	check(t, tab, []string{"owner", "name"}, "Tom Preston-Werner")
	check(t, tab, []string{"owner", "dob"}, date("1979-05-27T07:32:00-08:00"))
	check(t, tab, []string{"database", "enabled"}, true)
	check(t, tab, []string{"database", "temp_targets", "cpu"}, 79.5)
	check(t, tab, []string{"servers", "alpha", "ip"}, "10.0.0.1")
	check(t, tab, []string{"servers", "beta", "role"}, "backend")

	if v := tab.Lookup("database", "ports"); len(v.Array) != 3 || v.Array[2].Int != 8002 {
		t.Errorf("ports is %v", value(v))
	}
	if v := tab.Lookup("database", "data"); len(v.Array) != 2 || v.Array[1].Array[0].Float != 3.14 {
		t.Errorf("data is %v", value(v))
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name, doc, want string
	}{
		// Keys and values.
		{"no value", "a =", "want a value"},
		{"no equals", "a 1", "want ="},
		{"no key", "= 1", "want a key"},
		{"two values", "a = 1 2", "want the end of the line"},
		{"key twice", "a = 1\na = 2", "defined twice"},
		{"key twice dotted", "a.b = 1\na.b = 2", "defined twice"},
		{"multi-line key", "\"\"\"a\"\"\" = 1", "a key cannot be a multi-line string"},
		{"key alone", "a", "want ="},
		{"header as a value", "a = [", "missing its closing ]"},

		// Strings.
		{"unclosed basic", `a = "hello`, "missing its closing quote"},
		{"unclosed literal", `a = 'hello`, "missing its closing quote"},
		{"newline in a string", "a = \"one\ntwo\"", "missing its closing quote"},
		{"unclosed multi-line", `a = """hello`, `missing its closing """`},
		{"bad escape", `a = "\q"`, "is not an escape"},
		{"short escape", `a = "\u00zz"`, "hexadecimal digits"},
		{"surrogate escape", `a = "\uD800"`, "is not a character"},
		{"control in a string", "a = \"one\x00two\"", "cannot contain"},
		{"three quotes inside", `a = """""""""`, "in a row"},

		// Numbers.
		{"leading zero", "a = 07", "cannot start with a zero"},
		{"leading zero float", "a = 07.5", "cannot start with a zero"},
		{"trailing underscore", "a = 1_", "between two digits"},
		{"leading underscore", "a = _1", "between two digits"},
		{"double underscore", "a = 1__0", "between two digits"},
		{"no fraction", "a = 1.", "no digits"},
		{"no whole part", "a = .5", "no digits"},
		{"no exponent", "a = 1e", "no digits"},
		{"signed hex", "a = -0x1", "cannot have a sign"},
		{"bad hex digit", "a = 0xg", "not a digit in base 16"},
		{"bad octal digit", "a = 0o8", "not a digit in base 8"},
		{"bad binary digit", "a = 0b2", "not a digit in base 2"},
		{"too big", "a = 9223372036854775808", "does not fit in a 64 bit integer"},
		{"too small", "a = -9223372036854775809", "does not fit in a 64 bit integer"},
		{"not a value", "a = ?", "want a value"},
		{"almost true", "a = truthy", "want a value"},

		// Dates and times.
		{"bad month", "a = 1979-13-27", "not a valid local date"},
		{"bad day", "a = 1979-02-30", "not a valid local date"},
		{"bad hour", "a = 25:00:00", "not a valid local time"},
		{"no seconds", "a = 07:32", "not a valid local time"},
		{"no separator", "a = 1979-05-27x07:32:00", "needs a T or a space"},

		// Arrays.
		{"unclosed array", "a = [1, 2", "missing its closing ]"},
		{"no comma", "a = [1 2]", "want a comma or ]"},
		{"double comma", "a = [1,, 2]", "want a value"},

		// Inline tables.
		{"unclosed inline table", "a = {b = 1", "missing its closing }"},
		{"newline in an inline table", "a = {b = 1,\nc = 2}", "on one line"},
		{"trailing comma", "a = {b = 1,}", "cannot end with a comma"},
		{"no comma in an inline table", "a = {b = 1 c = 2}", "want a comma or }"},
		{"add to an inline table", "a = {b = 1}\na.c = 2", "written as an inline table"},
		{"header into an inline table", "a = {b = 1}\n[a.c]", "written as an inline table"},

		// Headers.
		{"unclosed header", "[a", "want ] to close"},
		{"unclosed array header", "[[a]", "want ]] to close"},
		{"empty header", "[]", "want a key"},
		{"table twice", "[a]\n[a]", "defined twice"},
		{"table over a value", "a = 1\n[a]", "is a integer"},
		{"value over a table", "[a]\nb = 1\n[a.b.c]", "cannot hold a table"},
		{"dotted then header", "[t]\na.b = 1\n[t.a]", "made by a dotted key"},
		{"dotted then deeper header", "[t]\na.b.c = 1\n[t.a.b]", "made by a dotted key"},
		{"append to a static array", "a = []\n[[a]]", "array of values"},
		{"append to a table", "[a]\n[[a]]", "is a table"},
		{"table over an array of tables", "[[a]]\n[a]", "is a array"},
		{"junk after a header", "[a] x", "want the end of the line"},

		// Whole documents.
		{"bare carriage return", "a = 1\rb = 2", "carriage return"},
		{"bare carriage return in an array", "a = [1,\r2]", "carriage return"},
		{"control in a comment", "# one\x00two", "cannot contain"},

		// The rest of the ways a value can be in the way of a key.
		{"value under a value", "a = 1\na.b = 2", "cannot go inside it"},
		{"sign alone", "a = +", "is not a number"},
		{"escape at the end of the file", `a = "\u0`, "ends in the middle"},
		{"backslash at the end of the file", `a = "\`, "ends in the middle"},
		{"control in a literal string", "a = 'one\x00two'", "no escapes"},
		{"control in a multi-line string", "a = \"\"\"one\x00two\"\"\"", "cannot contain"},
		{"carriage return in a multi-line string", "a = \"\"\"one\rtwo\"\"\"", "carriage return"},
		{"unclosed multi-line literal", "a = '''hello", "missing its closing '''"},
		{"empty key part", "[a..b]", "want a key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tab, err := Parse("test.toml", []byte(tt.doc))
			if err == nil {
				t.Fatalf("parsed without an error, and gave %v", tab.Keys())
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("the error is %q, want it to mention %q", err, tt.want)
			}
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("the error is a %T, want a *toml.Error", err)
			}
			if e.Pos.File != "test.toml" || e.Pos.Line < 1 || e.Pos.Col < 1 {
				t.Errorf("the error is at %s, which is not a place in the file", e.Pos)
			}
		})
	}
}

func TestErrorPositions(t *testing.T) {
	tests := []struct {
		doc       string
		line, col int
	}{
		{"a = 1\nb = ?", 2, 5},
		{"a = 1\n\n\nb = 07", 4, 5},
		{"[a]\nb = 1\n[a]", 3, 1},
		{"a = [\n1,\n2,\n?]", 4, 1},
		{"a = \"\"\"\none\ntwo\"\"\"\nb = ?", 4, 5},
		{"a = 1 # ok\nb = ?", 2, 5},
	}
	for _, tt := range tests {
		_, err := Parse("test.toml", []byte(tt.doc))
		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("%q gave %v, want a *toml.Error", tt.doc, err)
		}
		if e.Pos.Line != tt.line || e.Pos.Col != tt.col {
			t.Errorf("%q failed at %s, want test.toml:%d:%d", tt.doc, e.Pos, tt.line, tt.col)
		}
	}
}

func TestParseInvalidUTF8(t *testing.T) {
	_, err := Parse("test.toml", []byte("a = 1\nb = \"\xff\"\n"))
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("got %v, want a *toml.Error", err)
	}
	if !strings.Contains(e.Msg, "UTF-8") {
		t.Errorf("the error is %q, want it to mention UTF-8", e)
	}
	if e.Pos.Line != 2 || e.Pos.Col != 6 {
		t.Errorf("the error is at %s, want test.toml:2:6", e.Pos)
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[db]\ndsn = \"postgres://\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tab, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	check(t, tab, []string{"db", "dsn"}, "postgres://")

	if v := tab.Lookup("db", "dsn"); v.Pos.File != path {
		t.Errorf("the value says it is in %q, want %q", v.Pos.File, path)
	}
}

// FuzzParse looks for the two things a parser is not allowed to do: panic, or
// come back with a value that says it was written somewhere impossible.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"a = 1",
		"a = \"one\"",
		"a = '''one'''",
		"a = \"\"\"one \\\n two\"\"\"",
		"a = [1, [2], {b = 3}]",
		"[a]\nb.c = 1",
		"[[a]]\n[a.b]\nc = 1",
		"a = 1979-05-27T07:32:00Z",
		"a = 0x1_2 # comment",
		"a = inf\nb = -nan",
		"\ufeffa = 1\r\n",
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		tab, err := Parse("fuzz.toml", data)
		if err != nil {
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("the error is a %T, want a *toml.Error", err)
			}
			if e.Pos.Line < 1 || e.Pos.Col < 1 {
				t.Fatalf("the error is at %s, which is not a place in a file", e.Pos)
			}
			return
		}
		walk(t, tab)
	})
}

func walk(t *testing.T, tab *Table) {
	t.Helper()
	for key, v := range tab.All() {
		if v == nil {
			t.Fatalf("%q is in the table with no value", key)
		}
		if v.Pos.Line < 1 || v.Pos.Col < 1 {
			t.Fatalf("%q says it is at %s", key, v.Pos)
		}
		switch v.Kind {
		case KindInvalid:
			t.Fatalf("%q came back with no kind", key)
		case KindTable:
			walk(t, v.Table)
		case KindArray:
			for _, el := range v.Array {
				if el.Kind == KindTable {
					walk(t, el.Table)
				}
			}
		}
	}
}

func TestParseFileMissing(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "gone.toml"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got %v, want it to be os.ErrNotExist", err)
	}
}

func TestParseFileNamesTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.toml")
	if err := os.WriteFile(path, []byte("a = = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ParseFile(path)
	if err == nil || !strings.HasPrefix(err.Error(), path+":1:") {
		t.Errorf("got %v, want an error starting with %s:1:", err, path)
	}
}
