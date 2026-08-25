package console

import (
	"errors"
	"io"
	"testing"
)

func TestSection(t *testing.T) {
	s := newStreams(t, Options{})

	section := s.io.Section("Checks")
	section.Info("config loaded")
	section.Warn("no database configured")

	want := "Checks\n  config loaded\n  warning: no database configured\n"
	if got := s.err.String(); got != want {
		t.Errorf("stderr has\n%q\nwant\n%q", got, want)
	}
}

func TestSectionsNest(t *testing.T) {
	s := newStreams(t, Options{})

	checks := s.io.Section("Checks")
	checks.Info("config loaded")
	database := checks.Section("Database")
	database.Info("connecting")
	checks.Info("done")

	want := "Checks\n  config loaded\n  Database\n    connecting\n  done\n"
	if got := s.err.String(); got != want {
		t.Errorf("stderr has\n%q\nwant\n%q", got, want)
	}
}

// TestSectionLeavesDataAlone is the rule that makes sections safe to use in a
// command whose output is piped.
func TestSectionLeavesDataAlone(t *testing.T) {
	s := newStreams(t, Options{})

	section := s.io.Section("Users")
	section.Line("ada@example.com")
	section.Table([]string{"Name"}, [][]string{{"Ada"}})

	want := "ada@example.com\nName\nAda\n"
	if got := s.out.String(); got != want {
		t.Errorf("stdout has\n%q\nwant\n%q", got, want)
	}
}

func TestSectionWhenNobodyIsReading(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts Options
	}{
		{"quiet", Options{Verbosity: Quiet}},
		{"json", Options{JSON: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := newStreams(t, tt.opts)

			section := s.io.Section("Checks")
			section.Info("config loaded")

			if section != s.io {
				t.Error("Section allocated an indent for output that is not written")
			}
			if got := s.err.String(); got != "" {
				t.Errorf("stderr has %q, want nothing", got)
			}
		})
	}
}

// TestSectionKeepsBlankLinesBlank is about the trailing whitespace an indent
// would otherwise leave on an empty line, which every editor strips and every
// diff then argues about.
func TestSectionKeepsBlankLinesBlank(t *testing.T) {
	s := newStreams(t, Options{})

	section := s.io.Section("Checks")
	section.Info("")
	section.Info("done")

	want := "Checks\n\n  done\n"
	if got := s.err.String(); got != want {
		t.Errorf("stderr has %q, want %q", got, want)
	}
}

func TestSectionNarrowsTheWidth(t *testing.T) {
	s := newStreams(t, Options{Width: 40})

	section := s.io.Section("Checks")

	if section.errWidth != 38 {
		t.Errorf("a section has %d columns of stderr, want 38", section.errWidth)
	}
	if section.Width() != 40 {
		t.Errorf("a section has %d columns of stdout, want the 40 it started with", section.Width())
	}
}

// TestSectionKeepsTheBarBounded checks that a bar inside a section still writes
// lines rather than redrawing, and that the lines are indented like the rest.
func TestSectionKeepsTheBarBounded(t *testing.T) {
	s := newStreams(t, Options{})

	bar := s.io.Section("Importing").Progress(10)
	bar.Advance(1)
	bar.Advance(9)
	bar.Done()

	want := "Importing\n  10% (1/10)\n  100% (10/10)\n"
	if got := s.err.String(); got != want {
		t.Errorf("stderr has\n%q\nwant\n%q", got, want)
	}
}

// TestSectionSharesTheReader is the bug this would otherwise have: the prompt
// inside the section reads ahead into its own buffer, and the answer to the
// question after it is sitting in a buffer nobody looks at again.
func TestSectionSharesTheReader(t *testing.T) {
	s := answers(t, "Ada\ny\n", Options{})

	name, err := s.io.Section("Setup").Ask("Name", "")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if name != "Ada" {
		t.Errorf("Ask returned %q, want Ada", name)
	}

	ok, err := s.io.Confirm("Continue", false)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !ok {
		t.Error("Confirm returned false, want the y that was typed after the name")
	}
}

func TestSectionIndentsQuestions(t *testing.T) {
	s := answers(t, "Ada\n", Options{Color: ColorNever})

	if _, err := s.io.Section("Setup").Ask("Name", ""); err != nil {
		t.Fatalf("Ask: %v", err)
	}

	want := "Setup\n  Name: "
	if got := s.err.String(); got != want {
		t.Errorf("stderr has %q, want %q", got, want)
	}
}

// TestIndenterFollowsACarriageReturn is what lets a progress bar redraw inside
// a section. Without it the redraw would land at column zero and walk out from
// under the indent.
func TestIndenterFollowsACarriageReturn(t *testing.T) {
	var out fakeWriter
	w := &indenter{w: &out, prefix: indent, atLine: true}

	if _, err := w.Write([]byte("\r50%")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if want := "  \r  50%"; out.written != want {
		t.Errorf("wrote %q, want %q", out.written, want)
	}
	if out.calls != 1 {
		t.Errorf("wrote in %d calls, want 1 so that lines cannot interleave", out.calls)
	}
}

func TestIndenterCountsWhatItWasGiven(t *testing.T) {
	var out fakeWriter
	w := &indenter{w: &out, prefix: indent, atLine: true}

	p := []byte("done\n")
	n, err := w.Write(p)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(p) {
		t.Errorf("Write returned %d, want %d, the length of what it was given", n, len(p))
	}
}

func TestIndenterPassesOnAFailedWrite(t *testing.T) {
	w := &indenter{w: &fakeWriter{err: io.ErrClosedPipe}, prefix: indent, atLine: true}

	if _, err := w.Write([]byte("done\n")); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("Write returned %v, want the underlying error", err)
	}
}

// fakeWriter records what reached it, and how many calls it took.
type fakeWriter struct {
	written string
	calls   int
	err     error
}

func (w *fakeWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.err != nil {
		return 0, w.err
	}
	w.written += string(p)
	return len(p), nil
}
