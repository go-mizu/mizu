package diag

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Text writes l for a person to read.
//
//	error[MZ1042]: unknown config key "database.pool_size"
//	  --> config/app.toml:14:1
//	   |
//	14 | pool_size = 25
//	   | ^^^^^^^^^ no such field in Config.Database
//	   |
//	   = did you mean "max_open_conns"?
//	   = fix: mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns
//	   = explain: mizu explain MZ1042
//
// The quoted line comes from the file named in the diagnostic, read through
// [WithSource]. A file that cannot be read, or a line that is not in it, drops
// the block with the source in it and keeps everything else, because a
// diagnostic that names a place is worth printing even when the place has
// moved.
//
// Diagnostics print in the order they are in. [List.Sort] is the order they
// should usually be read in.
//
// Diagnostics that share a code and a message print at most [WithLimit] times,
// and the last of them says how many were held back. Two hundred lines saying
// the same thing is not a report, it is the reason people stop reading reports.
// [JSON] holds all of them, since a program is not the one being spared.
func Text(w io.Writer, l List, opts ...Option) error {
	o := newOptions(opts)
	src := &source{read: o.source, lines: map[string][][]byte{}}
	bw := bufio.NewWriter(w)

	// A group is one code and one message. Counting first is what lets the
	// last one printed say how many are behind it.
	total := map[string]int{}
	if o.limit > 0 {
		for _, d := range l {
			total[group(d)]++
		}
	}

	shown := map[string]int{}
	first := true
	for _, d := range l {
		g := group(d)
		if o.limit > 0 {
			if shown[g] >= o.limit {
				continue
			}
			shown[g]++
		}
		if !first {
			bw.WriteByte('\n')
		}
		first = false

		held := 0
		if o.limit > 0 && shown[g] == o.limit {
			held = total[g] - o.limit
		}
		writeOne(bw, d, src, o, held)
	}
	return bw.Flush()
}

// group is what makes two diagnostics the same thing said twice.
func group(d Diagnostic) string {
	return strconv.Itoa(int(d.Severity)) + "\x00" + string(d.Code) + "\x00" + d.Message
}

func writeOne(w *bufio.Writer, d Diagnostic, src *source, o options, held int) {
	color := severityStyle(d.Severity)

	// error[MZ1042]: unknown config key "database.pool_size"
	head := d.Severity.String()
	if d.Code.Valid() {
		head += "[" + string(d.Code) + "]"
	}
	w.WriteString(color.wrap(head+":", o.color))
	w.WriteByte(' ')
	w.WriteString(styleBold.wrap(messageOf(d), o.color))
	w.WriteByte('\n')

	line := src.line(d.File, d.Range.Start.Line)
	gutter := 0
	if line != nil {
		gutter = len(strconv.Itoa(d.Range.Start.Line))
	}
	pad := strings.Repeat(" ", gutter)
	bar := styleDim.wrap(pad+" |", o.color)

	//   --> config/app.toml:14:1
	if d.File != "" {
		w.WriteString(styleDim.wrap(pad+"--> ", o.color))
		w.WriteString(d.File)
		if pos := d.Range.Start.String(); pos != "" {
			w.WriteByte(':')
			w.WriteString(pos)
		}
		w.WriteByte('\n')
	}

	// 14 | pool_size = 25
	//    | ^^^^^^^^^ no such field in Config.Database
	if line != nil {
		w.WriteString(bar)
		w.WriteByte('\n')

		w.WriteString(styleDim.wrap(strconv.Itoa(d.Range.Start.Line)+" |", o.color))
		w.WriteByte(' ')
		w.Write(line)
		w.WriteByte('\n')

		w.WriteString(bar)
		w.WriteByte(' ')
		w.WriteString(caretPad(line, d.Range.Start.Col))
		w.WriteString(color.wrap(strings.Repeat("^", caretLen(line, d.Range)), o.color))
		if d.Detail != "" {
			w.WriteByte(' ')
			w.WriteString(color.wrap(d.Detail, o.color))
		}
		w.WriteByte('\n')
	}

	notes := noteLines(d, held)
	if len(notes) == 0 {
		return
	}
	if line != nil {
		w.WriteString(bar)
		w.WriteByte('\n')
	}
	for _, n := range notes {
		w.WriteString(styleDim.wrap(pad+" =", o.color))
		w.WriteByte(' ')
		w.WriteString(n)
		w.WriteByte('\n')
	}
}

// noteLines are the = lines under a diagnostic, in the order a reader wants
// them: what to try, then how to do it, then where the reason is.
func noteLines(d Diagnostic, held int) []string {
	var notes []string
	for _, s := range d.Suggestions {
		if s.Message == "" {
			continue
		}
		notes = append(notes, s.Message)
	}
	if d.Detail != "" && !hasSource(d) {
		// Nowhere to draw the label, so it goes here rather than nowhere.
		notes = append(notes, d.Detail)
	}
	if d.Fix != "" {
		notes = append(notes, "fix: "+d.Fix)
	}
	if e := d.Code.Explain(); e != "" {
		notes = append(notes, "explain: "+e)
	}
	if held > 0 {
		notes = append(notes, "and "+plural(held, "more like this"))
	}
	return notes
}

// hasSource reports whether writeOne drew a quoted line for d. It re-reads
// nothing: a detail with no place to hang under is one whose diagnostic has no
// range, which is what this asks.
func hasSource(d Diagnostic) bool { return d.File != "" && d.Range.IsValid() }

func messageOf(d Diagnostic) string {
	if d.Message == "" {
		return "(no message)"
	}
	return d.Message
}

func plural(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return strconv.Itoa(n) + " " + what
}

// caretPad is the run of blanks that puts the first caret under column col.
//
// A tab in the line becomes a tab here rather than some number of spaces, so
// the carets land right whatever the terminal's tab stops are set to.
func caretPad(line []byte, col int) string {
	end := min(max(col-1, 0), len(line))
	var b strings.Builder
	for _, r := range string(line[:end]) {
		if r == '\t' {
			b.WriteByte('\t')
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// caretLen is how many carets to draw.
//
// It counts runes rather than bytes, since a caret is drawn per character and
// the columns are bytes. A range ending on a later line, or ending before it
// starts, underlines the rest of the line: the bracket drawing rustc does for a
// span over several lines is a lot of machinery for a case no producer here
// has yet, and the first line plus the position is already the answer.
func caretLen(line []byte, r Range) int {
	start := min(max(r.Start.Col-1, 0), len(line))
	end := len(line)
	if r.End.Line == r.Start.Line && r.End.Col > r.Start.Col {
		end = min(r.End.Col-1, len(line))
	}
	return max(utf8.RuneCount(line[start:end]), 1)
}

// A source is the files [Text] quotes from, read once each.
type source struct {
	read  func(string) ([]byte, error)
	lines map[string][][]byte
}

// line is line n of file, or nil when there is no such line to quote.
func (s *source) line(file string, n int) []byte {
	if file == "" || n <= 0 {
		return nil
	}
	lines, ok := s.lines[file]
	if !ok {
		b, err := s.read(file)
		if err == nil {
			lines = bytes.Split(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")), []byte("\n"))
		}
		s.lines[file] = lines
	}
	if n > len(lines) {
		return nil
	}
	return lines[n-1]
}

// An Option changes how a list is rendered.
type Option func(*options)

type options struct {
	source func(string) ([]byte, error)
	color  bool
	limit  int
	dur    int64 // milliseconds, for the JSON summary
}

const defaultLimit = 3

func newOptions(opts []Option) options {
	o := options{source: os.ReadFile, limit: defaultLimit}
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// WithSource is where [Text] reads the lines it quotes.
//
// The default reads the named file from the filesystem, which is what a command
// line tool wants. Pass this for diagnostics about something that is not on
// disk, or in a test, where reading whatever happens to be at that path is not
// what the test meant.
func WithSource(read func(file string) ([]byte, error)) Option {
	return func(o *options) {
		if read == nil {
			read = func(string) ([]byte, error) { return nil, os.ErrNotExist }
		}
		o.source = read
	}
}

// WithColor turns the escapes on. The default is off, because this package
// writes to an io.Writer and has no way to ask whether it is a terminal. The
// caller does.
func WithColor(on bool) Option { return func(o *options) { o.color = on } }

// WithLimit is how many times one code and message may print before the rest
// are counted instead. Zero or less prints all of them, which is what a
// --verbose run wants.
func WithLimit(n int) Option { return func(o *options) { o.limit = n } }

// WithDuration is how long the run took, for the summary in [JSON]. It is
// ignored by [Text], where the number would be noise.
func WithDuration(ms int64) Option { return func(o *options) { o.dur = ms } }

// A style is one SGR parameter.
type style string

const (
	styleNone   style = ""
	styleBold   style = "1"
	styleDim    style = "2"
	styleRed    style = "31"
	styleYellow style = "33"
)

func (s style) wrap(text string, on bool) string {
	if !on || s == styleNone {
		return text
	}
	return "\x1b[" + string(s) + "m" + text + "\x1b[0m"
}

func severityStyle(s Severity) style {
	switch s {
	case Warning:
		return styleYellow
	case Note:
		return styleDim
	}
	return styleRed
}

// Summary is the count line a command prints under a report.
//
//	2 errors, 1 warning
//
// It is empty when there is nothing to count, so a caller can print it without
// asking first.
func (l List) Summary() string {
	var parts []string
	for _, s := range []Severity{Error, Warning, Note} {
		n := l.Count(s)
		switch {
		case n == 1:
			parts = append(parts, "1 "+s.String())
		case n > 1:
			parts = append(parts, strconv.Itoa(n)+" "+s.String()+"s")
		}
	}
	return strings.Join(parts, ", ")
}
