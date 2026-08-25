// Package diag is one thing wrong, said once, in two languages.
//
// A [Diagnostic] is a value: what is wrong, where, why it matters and what to
// do about it. [Text] renders it for a person and [JSON] renders it for a
// program, and both read the same value, so the two cannot drift apart. That
// is the whole reason this is a package rather than a fmt.Errorf in each tool.
//
//	d := diag.Diagnostic{
//		Code:    "MZ1042",
//		Message: `unknown config key "database.pool_size"`,
//		File:    "config/app.toml",
//		Range:   diag.Span(14, 1, 10),
//		Detail:  "no such field in Config.Database",
//		Fix:     "mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns",
//	}
//	diag.Text(os.Stderr, diag.List{d})
//
// prints
//
//	error[MZ1042]: unknown config key "database.pool_size"
//	  --> config/app.toml:14:1
//	   |
//	14 | pool_size = 25
//	   | ^^^^^^^^^ no such field in Config.Database
//	   |
//	   = fix: mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns
//	   = explain: mizu explain MZ1042
//
// The shape is the one rustc uses, because it is the most read diagnostic
// format there is and inventing a second one buys nothing.
//
// # What a diagnostic owes the reader
//
// Name the thing, say what was expected and what was found, give the fix,
// point at the explanation once, and be quiet about everything else. The type
// is built so that the first and the fourth are hard to leave out: [Diagnostic.File]
// and [Diagnostic.Range] are where the place goes, and [Diagnostic.Code] is
// what the explain and docs lines are computed from, so a diagnostic with a
// code has both without anybody writing them.
//
// The rest is a matter of what the producer puts in the fields, which is why
// the golden corpus exists.
//
// # Machine-readable, always
//
// [JSON] writes the mizu.diag/1 document, which is the format every mizu
// command emits under --json. An [Edit] in it is a byte-accurate replacement a
// program can apply without parsing anything, and [Confidence] says whether it
// should.
//
// # Did you mean
//
// [Suggest] decides what to offer for a name nobody recognises and [Did] writes
// the sentence it goes in. They are here rather than in the packages that
// report unknown names because an unknown setting, an unknown flag and an
// unknown command are the same problem, and three implementations of it drift
// apart in the threshold, in the wording and in whether anything is offered at
// all. Where nothing qualifies, nothing comes back: a wrong suggestion sends
// the reader down a false path with confidence.
//
// # Errors
//
// A [Diagnostic] is an error and a [List] turns into one with [List.Err], so a
// package that finds problems returns them the ordinary way. [Of] goes back the
// other way and pulls the diagnostics out of an error, or makes one out of an
// error that carries none, so a command with an error always has something to
// print under --json.
package diag

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

// A Severity is how much a diagnostic matters.
//
// [Error] is the zero value, so a diagnostic nobody graded is loud rather than
// quiet. The order is worst first, which is the order they should be read in
// and the order [List.Sort] puts them in.
type Severity int

const (
	Error   Severity = iota // wrong now
	Warning                 // wrong before long
	Note                    // worth knowing
)

func (s Severity) String() string {
	switch s {
	case Warning:
		return "warning"
	case Note:
		return "note"
	}
	return "error"
}

// MarshalText puts the word in the JSON rather than the number behind it,
// which is an ordering this package chose and nothing outside it should read.
func (s Severity) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

// UnmarshalText reads back what MarshalText wrote, so a mizu.diag/1 document
// round trips. An unknown word is an error rather than a silent [Error],
// because a severity nobody recognises is a document from a version this one
// does not understand.
func (s *Severity) UnmarshalText(b []byte) error {
	switch string(b) {
	case "error":
		*s = Error
	case "warning":
		*s = Warning
	case "note":
		*s = Note
	default:
		return errors.New("diag: unknown severity " + strconv.Quote(string(b)))
	}
	return nil
}

// A Confidence is how sure a [Suggestion] is.
//
// [Low] is the zero value, for the same reason [Error] is: the value nobody set
// is the one that asks a person to look. A producer that marks everything [High]
// is worse than one with no suggestions at all, because it spends the only
// signal this field carries.
type Confidence int

const (
	Low    Confidence = iota // a person should look at this
	Medium                   // probably right
	High                     // apply it
)

func (c Confidence) String() string {
	switch c {
	case Medium:
		return "medium"
	case High:
		return "high"
	}
	return "low"
}

func (c Confidence) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

func (c *Confidence) UnmarshalText(b []byte) error {
	switch string(b) {
	case "low":
		*c = Low
	case "medium":
		*c = Medium
	case "high":
		*c = High
	default:
		return errors.New("diag: unknown confidence " + strconv.Quote(string(b)))
	}
	return nil
}

// A Position is one place in a file.
//
// Line counts from one. Col counts bytes from the start of the line, also from
// one, which is what makes an [Edit] applicable: find the line, skip Col-1
// bytes. Bytes rather than runes because a program applying an edit works in
// bytes and a person reading a column number is reading it off a tool that
// counts the same way. [Text] converts when it draws the carets.
//
// The zero Position is one nothing is known about.
type Position struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

// IsValid reports whether p names a line.
func (p Position) IsValid() bool { return p.Line > 0 }

// String is line:col, or the line alone when there is no column, or empty when
// there is no line.
func (p Position) String() string {
	if !p.IsValid() {
		return ""
	}
	s := strconv.Itoa(p.Line)
	if p.Col > 0 {
		s += ":" + strconv.Itoa(p.Col)
	}
	return s
}

// compare orders positions by line and then by column.
func (p Position) compare(q Position) int {
	if c := p.Line - q.Line; c != 0 {
		return c
	}
	return p.Col - q.Col
}

// A Range is a half open span of a file: from Start up to but not including
// End. A range covering the nine bytes of pool_size on line 14 starts at 14:1
// and ends at 14:10.
//
// End may be the zero [Position], which means the span is a point at Start.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Span is the range covering n bytes from line and col.
func Span(line, col, n int) Range {
	return Range{Start: Position{line, col}, End: Position{line, col + n}}
}

// At is the range that is the single point line:col.
func At(line, col int) Range { return Range{Start: Position{line, col}} }

// IsValid reports whether r names a place.
func (r Range) IsValid() bool { return r.Start.IsValid() }

// An Edit is a replacement for the bytes in Range.
//
// It has to be applicable as it stands, without re-parsing and without
// guessing: the range is into the file as it is on disk now, and the edits
// within one [Suggestion] do not overlap. This is the field that turns a
// diagnostic into an action, and an inaccurate one is worse than none because
// what it corrupts is the file somebody was trying to fix.
type Edit struct {
	File    string `json:"file"`
	Range   Range  `json:"range"`
	NewText string `json:"new_text"`
}

// A Suggestion is one way to fix a diagnostic.
//
// The message is what a person reads and the edits are what a program applies.
// A suggestion with no edits is still worth having, since "did you mean X" is
// an answer even when nothing can apply it.
type Suggestion struct {
	Message    string     `json:"message"`
	Confidence Confidence `json:"confidence"`
	Edits      []Edit     `json:"edits,omitempty"`
}

// A Diagnostic is one thing wrong, at the place it went wrong.
//
// Message is one line, lower case, no full stop, in the shape the compiler and
// the go command use: what is wrong, concretely, naming the thing. Detail is
// the shorter label that goes under the carets, saying what was expected where
// Message said what was found.
//
// Fix is the command or the edit that puts it right, in a form somebody can
// run. Not a link and not "check your configuration".
type Diagnostic struct {
	// Code is the permanent name of this kind of diagnostic, and it is what
	// the explain and docs lines are computed from. Empty is allowed and it
	// means the reader gets no explanation to follow, which is a thing to fix
	// rather than a thing to ship.
	Code Code

	Severity Severity
	Message  string

	// File and Range are where it went wrong. A diagnostic about the project
	// rather than about a place in it leaves both empty.
	File  string
	Range Range

	// Detail is the label drawn under the carets.
	Detail string

	// Suggestions are the ways to fix it, best first.
	Suggestions []Suggestion

	// Fix is the command that fixes it.
	Fix string
}

// Error is the one line form, which is what %v and a bare return give.
//
//	config/app.toml:14:1: unknown config key "database.pool_size"
//
// Use [Text] for the form with the source line in it. This one is for the
// places that want an error and not a report: a log line, an error joined into
// another error, a test failure.
func (d Diagnostic) Error() string {
	var b strings.Builder
	if d.File != "" {
		b.WriteString(d.File)
		if pos := d.Range.Start.String(); pos != "" {
			b.WriteByte(':')
			b.WriteString(pos)
		}
		b.WriteString(": ")
	}
	if d.Message == "" {
		return b.String() + "(no message)"
	}
	b.WriteString(d.Message)
	return b.String()
}

// A List is a set of diagnostics from one run of one tool.
type List []Diagnostic

// Count is how many of l are at severity s.
func (l List) Count(s Severity) int {
	n := 0
	for _, d := range l {
		if d.Severity == s {
			n++
		}
	}
	return n
}

// Sort puts l in the order it should be read: worst first, then by file, then
// by place in the file.
//
// It is a method rather than something the renderers do, because the order a
// producer found things in is sometimes the order that makes sense and taking
// that decision away from it would mean sorting twice.
func (l List) Sort() {
	slices.SortStableFunc(l, func(a, b Diagnostic) int {
		if c := int(a.Severity) - int(b.Severity); c != 0 {
			return c
		}
		if c := strings.Compare(a.File, b.File); c != 0 {
			return c
		}
		return a.Range.Start.compare(b.Range.Start)
	})
}

// Err returns l as an error, or nil when l is empty.
//
// The nil is the point. Returning a List directly from a function declared to
// return error gives a non-nil error holding an empty list, which is the
// oldest bug in Go and one this type should not hand anybody.
func (l List) Err() error {
	if len(l) == 0 {
		return nil
	}
	return &listError{l}
}

// A listError is what [List.Err] returns.
//
// Unwrap returns the diagnostics one by one, so errors.Is and errors.As reach
// each of them and [Of] gets the list back whole.
type listError struct{ list List }

func (e *listError) Error() string {
	if len(e.list) == 1 {
		return e.list[0].Error()
	}
	var b strings.Builder
	for i, d := range e.list {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(d.Error())
	}
	return b.String()
}

func (e *listError) Unwrap() []error {
	errs := make([]error, len(e.list))
	for i, d := range e.list {
		errs[i] = d
	}
	return errs
}

// Of returns the diagnostics err carries.
//
// A [Diagnostic], a list from [List.Err], anything wrapped around either, and
// anything errors.Join put together are all found, in the order they appear.
// An error carrying none becomes one diagnostic at [Error] severity with the
// error's message and no place.
//
// That last part is what makes --json cheap to add to a command: whatever went
// wrong, there is a mizu.diag/1 document to print, and a command that grows a
// real diagnostic later starts emitting it without its caller changing.
//
// A join of ordinary errors becomes one diagnostic each rather than one
// diagnostic holding all of them. Configuration reports three wrong settings by
// joining three errors, and three problems are three entries in the list, three
// objects under --json, and three things to fix.
func Of(err error) List {
	if err == nil {
		return nil
	}
	var l List
	collect(err, &l, maxDepth)
	if len(l) == 0 {
		l = List{{Message: err.Error()}}
	}
	return l
}

// maxDepth is how far Of will follow an error chain.
//
// An error that wraps itself hangs errors.Is and errors.As too, so this is not
// a case the standard library defends against either. It is defended against
// here because the errors that reach Of come from wherever a command's work
// took it, and a command line tool that stops responding is a worse answer
// than one that reports the outer error. A hundred is far past any real chain.
const maxDepth = 100

// collect appends what err carries and reports whether it carried anything.
//
// The answer is what tells a join apart from a wrap. A branch of a join that
// held no diagnostic is still one of the problems and becomes its own
// diagnostic, because that is what a join means. A wrapped error that held none
// is left to [Of], which reports the outer message, because "boot: reading
// config/app.toml: no such file" is one problem and the wrapping is the part
// that says where.
func collect(err error, l *List, depth int) bool {
	for err != nil && depth > 0 {
		depth--
		switch x := err.(type) {
		case Diagnostic:
			*l = append(*l, x)
			return true
		case *Diagnostic:
			*l = append(*l, *x)
			return true
		case interface{ Unwrap() []error }:
			errs := x.Unwrap()
			for _, e := range errs {
				if !collect(e, l, depth) {
					*l = append(*l, Diagnostic{Message: e.Error()})
				}
			}
			return len(errs) > 0
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		default:
			return false
		}
	}
	return false
}
