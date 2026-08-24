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
