package golden

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// context is how many matching lines to print on each side of a difference.
const context = 3

// maxLines caps the report, because a golden file that moved by a thousand
// lines is not read a thousand lines at a time and the terminal scrollback is
// not the place to store it.
const maxLines = 40

// diff describes what changed between want and got, in a form a person reads
// in a test failure rather than one a patch tool applies.
//
// It is not a real diff. It finds the run of lines that match at the top, the
// run that matches at the bottom, and prints what is left in between with the
// line numbers it starts at. A change in the middle of a file therefore reads
// exactly like a diff, and a change that shifts every line reads as one large
// block, which is the honest description of what happened.
func diff(path string, want, got []byte) string {
	if isBinary(want) || isBinary(got) {
		return fmt.Sprintf("  %s holds %d bytes and the test produced %d, and neither looks like text.\n"+
			"  compare them with a tool that knows the format.", path, len(want), len(got))
	}

	wantLines, gotLines := lines(want), lines(got)

	head := 0
	for head < len(wantLines) && head < len(gotLines) && wantLines[head] == gotLines[head] {
		head++
	}

	tail := 0
	for tail < len(wantLines)-head && tail < len(gotLines)-head &&
		wantLines[len(wantLines)-1-tail] == gotLines[len(gotLines)-1-tail] {
		tail++
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  --- %s\n  +++ what the test produced\n", path)

	from := max(head-context, 0)
	fmt.Fprintf(&b, "  @@ line %d @@\n", from+1)

	printed := 0
	write := func(sign string, s string) bool {
		if printed == maxLines {
			b.WriteString("  ... and more, which is what -update and git diff are for\n")
			return false
		}
		printed++
		fmt.Fprintf(&b, "  %s %s\n", sign, escape(s))
		return true
	}

	// The leading context is at most context lines and the cap is far above
	// that, so this is the one run that cannot reach it.
	for _, l := range wantLines[from:head] {
		write(" ", l)
	}
	for _, l := range wantLines[head : len(wantLines)-tail] {
		if !write("-", l) {
			return b.String()
		}
	}
	for _, l := range gotLines[head : len(gotLines)-tail] {
		if !write("+", l) {
			return b.String()
		}
	}
	end := min(len(wantLines)-tail+context, len(wantLines))
	for _, l := range wantLines[len(wantLines)-tail : end] {
		if !write(" ", l) {
			return b.String()
		}
	}
	return b.String()
}

// lines splits on newlines and drops the empty piece a trailing newline leaves,
// so a file ending in a newline does not report a difference against one that
// does not until there really is one.
func lines(b []byte) []string {
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// escape makes a line printable. Trailing whitespace is the difference nobody
// can see, so it is spelled out rather than shown.
func escape(s string) string {
	trimmed := strings.TrimRight(s, " \t")
	if trimmed != s {
		return strconv.Quote(s)
	}
	if strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 && r != '\t' }) {
		return strconv.Quote(s)
	}
	return s
}

// isBinary reports whether b is something a line-based diff would mangle. A NUL
// byte or invalid UTF-8 in the first few kilobytes is the usual test, and it is
// the one git uses.
func isBinary(b []byte) bool {
	const sniff = 8000
	if len(b) > sniff {
		b = b[:sniff]
		// Cutting at a fixed offset can land in the middle of a rune, which
		// would look like invalid UTF-8 in a file that is nothing of the sort.
		for len(b) > 0 && !utf8.Valid(b) && len(b) > sniff-utf8.UTFMax {
			b = b[:len(b)-1]
		}
	}
	return bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b)
}
