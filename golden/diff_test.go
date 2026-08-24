package golden

import (
	"strings"
	"testing"
)

func TestDiffShowsTheChangedLinesOnly(t *testing.T) {
	want := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"
	got := "a\nb\nc\nd\nE\nf\ng\nh\ni\nj\n"

	out := diff("testdata/x.golden", []byte(want), []byte(got))

	if !strings.Contains(out, "- e") || !strings.Contains(out, "+ E") {
		t.Errorf("the change is not in the report:\n%s", out)
	}
	if strings.Contains(out, "  a\n") {
		t.Errorf("line a is four lines away and should not be shown:\n%s", out)
	}
	if !strings.Contains(out, "@@ line 2 @@") {
		t.Errorf("the report does not say where it starts:\n%s", out)
	}
}

func TestDiffNamesTheFile(t *testing.T) {
	out := diff("testdata/TestThing.golden", []byte("a\n"), []byte("b\n"))

	if !strings.Contains(out, "--- testdata/TestThing.golden") {
		t.Errorf("the report does not name the file:\n%s", out)
	}
	if !strings.Contains(out, "+++ what the test produced") {
		t.Errorf("the report does not say which side is which:\n%s", out)
	}
}

func TestDiffHandlesAnAddedLine(t *testing.T) {
	out := diff("x", []byte("a\nb\n"), []byte("a\nb\nc\n"))

	if !strings.Contains(out, "+ c") {
		t.Errorf("the added line is not in the report:\n%s", out)
	}
	if strings.Contains(out, "\n  - ") {
		t.Errorf("nothing was removed and the report says otherwise:\n%s", out)
	}
}

func TestDiffHandlesARemovedLine(t *testing.T) {
	out := diff("x", []byte("a\nb\nc\n"), []byte("a\nb\n"))

	if !strings.Contains(out, "- c") {
		t.Errorf("the removed line is not in the report:\n%s", out)
	}
	if strings.Contains(out, "\n  + ") {
		t.Errorf("nothing was added and the report says otherwise:\n%s", out)
	}
}

func TestDiffHandlesAnEmptySide(t *testing.T) {
	if out := diff("x", nil, []byte("a\n")); !strings.Contains(out, "+ a") {
		t.Errorf("a file that was empty and is not:\n%s", out)
	}
	if out := diff("x", []byte("a\n"), nil); !strings.Contains(out, "- a") {
		t.Errorf("a file that had a line and now does not:\n%s", out)
	}
}

// TestDiffStops keeps a failure readable when everything moved. A thousand line
// diff in a terminal is not read, it is scrolled past.
func TestDiffStops(t *testing.T) {
	var want, got strings.Builder
	for i := range 500 {
		want.WriteString("old line\n")
		got.WriteString("new line\n")
		_ = i
	}

	out := diff("x", []byte(want.String()), []byte(got.String()))

	if n := strings.Count(out, "\n"); n > maxLines+6 {
		t.Errorf("the report is %d lines, want it capped near %d", n, maxLines)
	}
	if !strings.Contains(out, "and more") {
		t.Errorf("the report does not say it was cut short:\n%s", out)
	}
}

// TestDiffSpellsOutInvisibleChanges is the one a reader cannot solve alone. Two
// lines that look identical differ in a trailing space, and quoting is the only
// way to show that in a terminal.
func TestDiffSpellsOutInvisibleChanges(t *testing.T) {
	out := diff("x", []byte("value\n"), []byte("value \n"))

	if !strings.Contains(out, `"value "`) {
		t.Errorf("a trailing space is not visible in the report:\n%s", out)
	}
}

func TestDiffDoesNotTryToDiffBinary(t *testing.T) {
	out := diff("x", []byte{0x00, 0x01, 0x02}, []byte{0x00, 0x03})

	if !strings.Contains(out, "neither looks like text") {
		t.Errorf("a binary file was diffed line by line:\n%s", out)
	}
	if !strings.Contains(out, "3 bytes") || !strings.Contains(out, "2") {
		t.Errorf("the report does not give the sizes:\n%s", out)
	}
}

func TestIsBinary(t *testing.T) {
	tests := map[string]struct {
		in   []byte
		want bool
	}{
		"text":              {[]byte("hello\nworld\n"), false},
		"nothing":           {nil, false},
		"a nul byte":        {[]byte("hello\x00world"), true},
		"invalid utf-8":     {[]byte{0xff, 0xfe, 0xfd}, true},
		"valid utf-8":       {[]byte("日本語"), false},
		"a long text file":  {[]byte(strings.Repeat("abc\n", 5000)), false},
		"a long utf-8 file": {[]byte(strings.Repeat("日本語\n", 5000)), false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := isBinary(tt.in); got != tt.want {
				t.Errorf("isBinary = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLinesIgnoresATrailingNewline(t *testing.T) {
	with := lines([]byte("a\nb\n"))
	without := lines([]byte("a\nb"))

	if len(with) != 2 || len(without) != 2 {
		t.Fatalf("lines gave %v and %v, want two each", with, without)
	}
	if with[1] != without[1] {
		t.Errorf("lines gave %q and %q, want the same", with[1], without[1])
	}
	if got := lines(nil); got != nil {
		t.Errorf("lines of nothing gave %v, want nothing", got)
	}
}

// TestEscape covers what a line looks like in a report. A line with nothing
// odd about it is printed as it is, and anything invisible is quoted, since a
// difference the reader cannot see is a failure they cannot act on.
func TestEscape(t *testing.T) {
	tests := map[string]struct{ in, want string }{
		"ordinary":          {"hello", "hello"},
		"a tab inside":      {"a\tb", "a\tb"},
		"trailing space":    {"value ", `"value "`},
		"trailing tab":      {"value\t", `"value\t"`},
		"a control byte":    {"a\x01b", `"a\x01b"`},
		"a carriage return": {"a\rb", `"a\rb"`},
		"nothing":           {"", ""},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := escape(tt.in); got != tt.want {
				t.Errorf("escape(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

// TestIsBinaryTrimsAPartialRune pins the reason the sniff loop is there. The
// cut at 8000 bytes lands inside a three-byte rune, and without the trim the
// file reads as binary because of where the sniff stopped rather than because
// of anything in it.
func TestIsBinaryTrimsAPartialRune(t *testing.T) {
	var b []byte
	for len(b) < 8002 {
		b = append(b, "日"...) // three bytes, so 8000 is not a boundary
	}

	if isBinary(b) {
		t.Error("a valid utf-8 file reads as binary because the sniff cut a rune")
	}
}

// TestDiffStopsInEveryDirection walks the cap through each of the four runs the
// report is built from, since each one returns early on its own.
func TestDiffStopsInEveryDirection(t *testing.T) {
	long := func(s string, n int) string { return strings.Repeat(s+"\n", n) }

	tests := map[string]struct{ want, got string }{
		"the leading context is long": {
			want: long("same", 100) + long("old", 100),
			got:  long("same", 100) + long("new", 100),
		},
		"only the golden file is long": {
			want: long("old", 100),
			got:  "new\n",
		},
		"only what the test produced is long": {
			want: "old\n",
			got:  long("new", 100),
		},
		"the trailing context is long": {
			want: "old\n" + long("same", 100),
			got:  "new\n" + long("same", 100),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			out := diff("x", []byte(tt.want), []byte(tt.got))
			if n := strings.Count(out, "\n"); n > maxLines+6 {
				t.Errorf("the report is %d lines, want it capped near %d", n, maxLines)
			}
		})
	}
}
