package diag_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

// files is a source that serves what a test wrote rather than whatever happens
// to be at that path.
func files(m map[string]string) diag.Option {
	return diag.WithSource(func(name string) ([]byte, error) {
		s, ok := m[name]
		if !ok {
			return nil, errors.New("no such file")
		}
		return []byte(s), nil
	})
}

// appTOML puts pool_size on line 14, which is where doc 36 section 2.2 puts it,
// so the whole-shape test below can be that example byte for byte.
const appTOML = `[app]
name = "blog"
env = "production"

[http]
addr = ":8080"

[log]
level = "info"
format = "json"

[database]
url = "postgres://localhost/blog"
pool_size = 25
`

func render(t *testing.T, l diag.List, opts ...diag.Option) string {
	t.Helper()
	var b strings.Builder
	if err := diag.Text(&b, l, opts...); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// The shape, whole, with everything a diagnostic can carry. This is the
// example in doc 36 section 2.2, and the point of comparing it byte for byte
// is that the documented shape is the one that comes out.
func TestTextTheWholeShape(t *testing.T) {
	got := render(t, diag.List{{
		Code:    "MZ1042",
		Message: `unknown config key "database.pool_size"`,
		File:    "config/app.toml",
		Range:   diag.Span(14, 1, 9),
		Detail:  "no such field in Config.Database",
		Suggestions: []diag.Suggestion{{
			Message:    `did you mean "max_open_conns"?`,
			Confidence: diag.High,
		}},
		Fix: "mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns",
	}}, files(map[string]string{"config/app.toml": appTOML}))

	want := `error[MZ1042]: unknown config key "database.pool_size"
  --> config/app.toml:14:1
   |
14 | pool_size = 25
   | ^^^^^^^^^ no such field in Config.Database
   |
   = did you mean "max_open_conns"?
   = fix: mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns
   = explain: mizu explain MZ1042
`
	assertText(t, got, want)
}

// The gutter is as wide as the line number, so a one digit line is one column
// narrower all the way down and the arrow still points one left of the bar.
func TestTextWithNoCode(t *testing.T) {
	got := render(t, diag.List{{
		Message: "unknown setting",
		File:    "config/app.toml",
		Range:   diag.Span(2, 1, 4),
	}}, files(map[string]string{"config/app.toml": appTOML}))

	want := `error: unknown setting
 --> config/app.toml:2:1
  |
2 | name = "blog"
  | ^^^^
`
	assertText(t, got, want)
}

// A diagnostic about the project rather than about a place in it has no
// source block, and the = lines sit where there is no gutter to align under.
func TestTextWithNoPlace(t *testing.T) {
	got := render(t, diag.List{{
		Severity: diag.Warning,
		Code:     "MZ5001",
		Message:  "no APP_KEY is set",
		Fix:      "mizu key:generate",
	}})

	want := `warning[MZ5001]: no APP_KEY is set
 = fix: mizu key:generate
 = explain: mizu explain MZ5001
`
	assertText(t, got, want)
}

// A detail with no carets to sit under still has to be said, so it moves to
// the = lines rather than being dropped.
func TestTextMovesTheDetailWhenThereIsNoPlaceForIt(t *testing.T) {
	got := render(t, diag.List{{
		Message: "the module cannot be loaded",
		Detail:  "go.mod names go 1.28 and the toolchain here is 1.27",
	}})

	want := `error: the module cannot be loaded
 = go.mod names go 1.28 and the toolchain here is 1.27
`
	assertText(t, got, want)
}

// A file that cannot be read loses the quoted line and keeps everything else,
// because a diagnostic that names a place is worth printing even when the
// place has moved.
func TestTextWhenTheFileIsNotThere(t *testing.T) {
	got := render(t, diag.List{{
		Message: "unknown setting",
		File:    "config/gone.toml",
		Range:   diag.Span(3, 1, 9),
		Fix:     "delete the line",
	}}, files(nil))

	want := `error: unknown setting
--> config/gone.toml:3:1
 = fix: delete the line
`
	assertText(t, got, want)
}

// And so does a line number past the end of the file.
func TestTextWhenTheLineIsNotThere(t *testing.T) {
	got := render(t, diag.List{{
		Message: "unknown setting",
		File:    "config/app.toml",
		Range:   diag.Span(900, 1, 9),
	}}, files(map[string]string{"config/app.toml": appTOML}))

	want := `error: unknown setting
--> config/app.toml:900:1
`
	assertText(t, got, want)
}

// The gutter is as wide as the line number, so a report over a long file does
// not put every bar in a different column.
func TestTextGutterWidensWithTheLineNumber(t *testing.T) {
	src := strings.Repeat("x\n", 120) + "pool_size = 25\n"
	got := render(t, diag.List{{
		Message: "unknown setting",
		File:    "app.toml",
		Range:   diag.Span(121, 1, 9),
	}}, files(map[string]string{"app.toml": src}))

	want := `error: unknown setting
   --> app.toml:121:1
    |
121 | pool_size = 25
    | ^^^^^^^^^
`
	assertText(t, got, want)
}

// A tab in the quoted line becomes a tab in the caret line, so the carets land
// under the right characters whatever the terminal's tab stops are.
func TestTextKeepsTabsSoTheCaretsLineUp(t *testing.T) {
	got := render(t, diag.List{{
		Message: "not a number",
		File:    "app.toml",
		Range:   diag.Span(1, 8, 3),
	}}, files(map[string]string{"app.toml": "\t\tport = one\n"}))

	want := "error: not a number\n" +
		" --> app.toml:1:8\n" +
		"  |\n" +
		"1 | \t\tport = one\n" +
		"  | \t\t     ^^^\n"
	assertText(t, got, want)
}

// The columns are bytes and the carets are characters, so a line with a
// multibyte character in it underlines the right thing rather than a count of
// its bytes.
func TestTextCountsCharactersNotBytes(t *testing.T) {
	got := render(t, diag.List{{
		Message: "not a name",
		File:    "app.toml",
		// naïve is five characters and six bytes.
		Range: diag.Span(1, 8, 6),
	}}, files(map[string]string{"app.toml": `name = naïve`}))

	want := `error: not a name
 --> app.toml:1:8
  |
1 | name = naïve
  |        ^^^^^
`
	assertText(t, got, want)
}

// A range that runs past the line it started on underlines the rest of that
// line, which is the honest thing to draw without the bracket machinery.
func TestTextUnderlinesToTheEndOfAMultilineRange(t *testing.T) {
	got := render(t, diag.List{{
		Message: "unterminated string",
		File:    "app.toml",
		Range:   diag.Range{Start: diag.Position{Line: 1, Col: 8}, End: diag.Position{Line: 3, Col: 1}},
	}}, files(map[string]string{"app.toml": "name = \"blog\nmore\nend\n"}))

	want := `error: unterminated string
 --> app.toml:1:8
  |
1 | name = "blog
  |        ^^^^^
`
	assertText(t, got, want)
}

// A point range, and a column past the end of the line, both draw one caret
// rather than none. A caret line with nothing on it is a bug that looks like a
// blank line.
func TestTextAlwaysDrawsAtLeastOneCaret(t *testing.T) {
	for _, r := range []diag.Range{
		diag.At(1, 3),
		diag.Span(1, 300, 4),
		diag.Span(1, 3, 0),
	} {
		got := render(t, diag.List{{Message: "here", File: "a.toml", Range: r}},
			files(map[string]string{"a.toml": "abc\n"}))
		if !strings.Contains(got, "^") {
			t.Errorf("range %#v drew no caret:\n%s", r, got)
		}
	}
}

func TestTextSeparatesDiagnosticsWithABlankLine(t *testing.T) {
	got := render(t, diag.List{
		{Message: "first"},
		{Severity: diag.Warning, Message: "second"},
	})
	want := "error: first\n\nwarning: second\n"
	assertText(t, got, want)
}

func TestTextOfNothingIsNothing(t *testing.T) {
	if got := render(t, nil); got != "" {
		t.Errorf("an empty list rendered %q", got)
	}
}

// Two hundred lines saying the same thing is not a report. The last one shown
// says how many are behind it.
func TestTextGroupsWhatIsSaidTwice(t *testing.T) {
	var l diag.List
	for i := range 200 {
		l = append(l, diag.Diagnostic{
			Code:    "MZ1042",
			Message: "unknown setting",
			File:    "app.toml",
			Range:   diag.At(i+1, 1),
		})
	}
	got := render(t, l, files(nil))

	if n := strings.Count(got, "error[MZ1042]"); n != 3 {
		t.Errorf("printed %d of 200, want 3:\n%s", n, got)
	}
	if !strings.Contains(got, "= and 197 more like this") {
		t.Errorf("did not say how many were held back:\n%s", got)
	}
	if !strings.Contains(got, "app.toml:3:1") || strings.Contains(got, "app.toml:4:1") {
		t.Errorf("held back the wrong three:\n%s", got)
	}
}

// One held back reads as one, not as 1.
func TestTextGroupsJustOverTheLimit(t *testing.T) {
	var l diag.List
	for range 4 {
		l = append(l, diag.Diagnostic{Message: "unknown setting"})
	}
	if got := render(t, l); !strings.Contains(got, "= and 1 more like this") {
		t.Errorf("want one held back:\n%s", got)
	}
}

// Only what is the same is grouped. A different message, a different code or a
// different severity is a different thing to say.
func TestTextGroupsOnCodeMessageAndSeverity(t *testing.T) {
	l := diag.List{
		{Code: "MZ1042", Message: "unknown setting"},
		{Code: "MZ1043", Message: "unknown setting"},
		{Code: "MZ1042", Message: "unknown table"},
		{Code: "MZ1042", Message: "unknown setting", Severity: diag.Warning},
	}
	got := render(t, l)
	for _, want := range []string{"error[MZ1042]: unknown setting", "error[MZ1043]: unknown setting",
		"error[MZ1042]: unknown table", "warning[MZ1042]: unknown setting"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q was grouped away:\n%s", want, got)
		}
	}
	if strings.Contains(got, "more like this") {
		t.Errorf("grouped four different things:\n%s", got)
	}
}

// A run that wants all of them, which is what --verbose asks for.
func TestTextWithNoLimitPrintsEveryOne(t *testing.T) {
	var l diag.List
	for range 10 {
		l = append(l, diag.Diagnostic{Message: "unknown setting"})
	}
	got := render(t, l, diag.WithLimit(0))
	if n := strings.Count(got, "error:"); n != 10 {
		t.Errorf("printed %d of 10:\n%s", n, got)
	}
	if strings.Contains(got, "more like this") {
		t.Error("said something was held back when nothing was")
	}
}

func TestTextWithColor(t *testing.T) {
	got := render(t, diag.List{{
		Message: "unknown setting",
		File:    "app.toml",
		Range:   diag.Span(1, 1, 3),
		Detail:  "no such field",
	}}, files(map[string]string{"app.toml": "abc\n"}), diag.WithColor(true))

	for _, want := range []string{
		"\x1b[31merror:\x1b[0m",        // the severity in red
		"\x1b[1munknown setting",       // the message in bold
		"\x1b[2m1 |\x1b[0m",            // the gutter dim
		"\x1b[31m^^^\x1b[0m",           // the carets in red
		"\x1b[31mno such field\x1b[0m", // and the label with them
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%q", want, got)
		}
	}
}

func TestTextColorsWarningsAndNotesDifferently(t *testing.T) {
	got := render(t, diag.List{
		{Severity: diag.Warning, Message: "w"},
		{Severity: diag.Note, Message: "n"},
	}, diag.WithColor(true))

	if !strings.Contains(got, "\x1b[33mwarning:") {
		t.Errorf("a warning is not yellow:\n%q", got)
	}
	if !strings.Contains(got, "\x1b[2mnote:") {
		t.Errorf("a note is not dim:\n%q", got)
	}
}

// Colour is off by default, because this package writes to an io.Writer and
// has no way to ask whether it is a terminal.
func TestTextIsPlainByDefault(t *testing.T) {
	got := render(t, diag.List{{Message: "unknown setting"}})
	if strings.Contains(got, "\x1b[") {
		t.Errorf("coloured without being asked:\n%q", got)
	}
}

// A diagnostic with nothing in it still prints as something, because a blank
// line where an error should be is the worst possible report.
func TestTextOfAnEmptyDiagnostic(t *testing.T) {
	if got, want := render(t, diag.List{{}}), "error: (no message)\n"; got != want {
		t.Errorf("rendered %q, want %q", got, want)
	}
}

// A suggestion with edits and no message has nothing to say to a person. It
// stays in the JSON, where a program can still apply it.
func TestTextSkipsASuggestionWithNothingToRead(t *testing.T) {
	got := render(t, diag.List{{
		Message: "unknown setting",
		Suggestions: []diag.Suggestion{
			{Edits: []diag.Edit{{File: "a.toml", NewText: "x"}}},
			{Message: "or delete the line"},
		},
	}})
	want := `error: unknown setting
 = or delete the line
`
	assertText(t, got, want)
}

// The default source reads the file named in the diagnostic, which is what a
// command line tool wants and what nothing else in these tests uses.
func TestTextReadsTheFilesystemByDefault(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "app.toml")
	if err := os.WriteFile(name, []byte(appTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	got := render(t, diag.List{{Message: "unknown setting", File: name, Range: diag.Span(14, 1, 9)}})
	if !strings.Contains(got, "pool_size = 25") {
		t.Errorf("did not read the file:\n%s", got)
	}
}

// A source of nil is a source that reads nothing, rather than a nil call.
func TestTextWithNoSourceAtAll(t *testing.T) {
	got := render(t, diag.List{{Message: "unknown setting", File: "app.toml", Range: diag.At(1, 1)}},
		diag.WithSource(nil))
	if strings.Contains(got, "|") {
		t.Errorf("quoted a line from nowhere:\n%s", got)
	}
}

// Windows line endings are line endings, not part of the last word on the
// line, so a checkout with CRLF in it does not get a caret line an inch wide.
func TestTextReadsCRLFFiles(t *testing.T) {
	got := render(t, diag.List{{
		Message: "unknown setting",
		File:    "app.toml",
		Range:   diag.Span(2, 1, 3),
	}}, files(map[string]string{"app.toml": "[db]\r\nabc = 1\r\n"}))

	want := `error: unknown setting
 --> app.toml:2:1
  |
2 | abc = 1
  | ^^^
`
	assertText(t, got, want)
}

// A file is read once however many diagnostics name it.
func TestTextReadsEachFileOnce(t *testing.T) {
	reads := 0
	src := diag.WithSource(func(string) ([]byte, error) {
		reads++
		return []byte(appTOML), nil
	})
	l := diag.List{
		{Message: "one", File: "app.toml", Range: diag.At(1, 1)},
		{Message: "two", File: "app.toml", Range: diag.At(2, 1)},
		{Message: "three", File: "app.toml", Range: diag.At(3, 1)},
	}
	render(t, l, src)
	if reads != 1 {
		t.Errorf("read the file %d times, want 1", reads)
	}
}

// A file that could not be read is not read again either, since the second
// attempt fails the same way and a report over a deleted tree would otherwise
// be one failed open per diagnostic.
func TestTextDoesNotRetryAFileItCouldNotRead(t *testing.T) {
	reads := 0
	src := diag.WithSource(func(string) ([]byte, error) {
		reads++
		return nil, errors.New("no such file")
	})
	l := diag.List{
		{Message: "one", File: "gone.toml", Range: diag.At(1, 1)},
		{Message: "two", File: "gone.toml", Range: diag.At(2, 1)},
	}
	render(t, l, src)
	if reads != 1 {
		t.Errorf("tried the file %d times, want 1", reads)
	}
}

func TestTextReturnsAWriteError(t *testing.T) {
	err := diag.Text(brokenWriter{}, diag.List{{Message: "unknown setting"}})
	if !errors.Is(err, errBrokenWriter) {
		t.Errorf("Text() returned %v, want the write error", err)
	}
}

var errBrokenWriter = errors.New("broken")

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errBrokenWriter }

func assertText(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("rendered:\n%s\nwant:\n%s\ngot %q", got, want, got)
	}
}
