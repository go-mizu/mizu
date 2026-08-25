package diag_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

func TestSeverityWords(t *testing.T) {
	for _, tt := range []struct {
		s    diag.Severity
		want string
	}{
		{diag.Error, "error"},
		{diag.Warning, "warning"},
		{diag.Note, "note"},
		{diag.Severity(99), "error"},
	} {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

// The zero Severity is error, so a diagnostic nobody graded is the loud one.
func TestTheUngradedSeverityIsError(t *testing.T) {
	var d diag.Diagnostic
	if d.Severity != diag.Error {
		t.Errorf("the zero Severity is %v, want error", d.Severity)
	}
}

// And the zero Confidence is low, for the same argument the other way round.
func TestTheUngradedConfidenceIsLow(t *testing.T) {
	var s diag.Suggestion
	if s.Confidence != diag.Low {
		t.Errorf("the zero Confidence is %v, want low", s.Confidence)
	}
}

func TestConfidenceWords(t *testing.T) {
	for _, tt := range []struct {
		c    diag.Confidence
		want string
	}{
		{diag.Low, "low"},
		{diag.Medium, "medium"},
		{diag.High, "high"},
		{diag.Confidence(99), "low"},
	} {
		if got := tt.c.String(); got != tt.want {
			t.Errorf("Confidence(%d).String() = %q, want %q", tt.c, got, tt.want)
		}
	}
}

func TestSeverityRoundTrips(t *testing.T) {
	for _, want := range []diag.Severity{diag.Error, diag.Warning, diag.Note} {
		b, err := want.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		var got diag.Severity
		if err := got.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%v round tripped to %v", want, got)
		}
	}
}

func TestConfidenceRoundTrips(t *testing.T) {
	for _, want := range []diag.Confidence{diag.Low, diag.Medium, diag.High} {
		b, err := want.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		var got diag.Confidence
		if err := got.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%v round tripped to %v", want, got)
		}
	}
}

// A word this package does not know is a document from a version it does not
// know, and reading it as an error rather than as the zero value is the
// difference between saying so and quietly downgrading somebody's error.
func TestAnUnknownSeverityIsAnError(t *testing.T) {
	var s diag.Severity
	if err := s.UnmarshalText([]byte("catastrophe")); err == nil {
		t.Fatal("UnmarshalText accepted a severity that does not exist")
	}
	var c diag.Confidence
	if err := c.UnmarshalText([]byte("certain")); err == nil {
		t.Fatal("UnmarshalText accepted a confidence that does not exist")
	}
}

func TestPositionString(t *testing.T) {
	for _, tt := range []struct {
		p    diag.Position
		want string
	}{
		{diag.Position{Line: 14, Col: 1}, "14:1"},
		{diag.Position{Line: 14}, "14"},
		{diag.Position{}, ""},
		{diag.Position{Col: 3}, ""},
	} {
		if got := tt.p.String(); got != tt.want {
			t.Errorf("%#v.String() = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestSpanAndAt(t *testing.T) {
	span := diag.Span(14, 1, 9)
	want := diag.Range{Start: diag.Position{Line: 14, Col: 1}, End: diag.Position{Line: 14, Col: 10}}
	if span != want {
		t.Errorf("Span(14, 1, 9) = %#v, want %#v", span, want)
	}
	if !span.IsValid() {
		t.Error("Span is not valid")
	}

	at := diag.At(3, 7)
	if at.Start != (diag.Position{Line: 3, Col: 7}) || at.End.IsValid() {
		t.Errorf("At(3, 7) = %#v", at)
	}
	if (diag.Range{}).IsValid() {
		t.Error("the zero Range says it names a place")
	}
}

func TestDiagnosticError(t *testing.T) {
	for _, tt := range []struct {
		name string
		d    diag.Diagnostic
		want string
	}{
		{
			"with a place",
			diag.Diagnostic{Message: "unknown setting", File: "config/app.toml", Range: diag.At(14, 1)},
			"config/app.toml:14:1: unknown setting",
		},
		{
			"with a line and no column",
			diag.Diagnostic{Message: "unknown setting", File: ".env", Range: diag.Range{Start: diag.Position{Line: 9}}},
			".env:9: unknown setting",
		},
		{
			"with a file and no line",
			diag.Diagnostic{Message: "cannot be read", File: "config/app.toml"},
			"config/app.toml: cannot be read",
		},
		{
			"with no place at all",
			diag.Diagnostic{Message: "no APP_KEY is set"},
			"no APP_KEY is set",
		},
		{
			"with nothing in it",
			diag.Diagnostic{},
			"(no message)",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListCount(t *testing.T) {
	l := diag.List{
		{Severity: diag.Error},
		{Severity: diag.Warning},
		{Severity: diag.Error},
		{Severity: diag.Note},
	}
	for _, tt := range []struct {
		s    diag.Severity
		want int
	}{{diag.Error, 2}, {diag.Warning, 1}, {diag.Note, 1}} {
		if got := l.Count(tt.s); got != tt.want {
			t.Errorf("Count(%v) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestListSortPutsTheWorstFirst(t *testing.T) {
	l := diag.List{
		{Severity: diag.Note, Message: "note"},
		{Severity: diag.Error, File: "b.toml", Range: diag.At(1, 1), Message: "b1"},
		{Severity: diag.Warning, Message: "warning"},
		{Severity: diag.Error, File: "a.toml", Range: diag.At(9, 1), Message: "a9"},
		{Severity: diag.Error, File: "a.toml", Range: diag.At(2, 4), Message: "a2"},
	}
	l.Sort()

	var got []string
	for _, d := range l {
		got = append(got, d.Message)
	}
	want := []string{"a2", "a9", "b1", "warning", "note"}
	if !slices.Equal(got, want) {
		t.Errorf("Sort() gave %v, want %v", got, want)
	}
}

// Sorting is stable, so two diagnostics in the same place stay in the order
// the producer found them, which is usually the order they happened in.
func TestListSortIsStable(t *testing.T) {
	l := diag.List{
		{File: "a.toml", Range: diag.At(1, 1), Message: "first"},
		{File: "a.toml", Range: diag.At(1, 1), Message: "second"},
		{File: "a.toml", Range: diag.At(1, 1), Message: "third"},
	}
	l.Sort()
	if l[0].Message != "first" || l[2].Message != "third" {
		t.Errorf("Sort() reordered diagnostics in the same place: %v", l)
	}
}

// An empty list is not an error, and it is a nil error rather than a non-nil
// error holding an empty list, which is the trap this method exists to avoid.
func TestEmptyListIsNoError(t *testing.T) {
	var l diag.List
	if err := l.Err(); err != nil {
		t.Errorf("an empty List is the error %v", err)
	}
	if err := (diag.List{}).Err(); err != nil {
		t.Errorf("an empty List is the error %v", err)
	}
}

func TestListErrReadsAsOneOrAsMany(t *testing.T) {
	one := diag.List{{Message: "unknown setting", File: "app.toml", Range: diag.At(1, 1)}}
	if got, want := one.Err().Error(), "app.toml:1:1: unknown setting"; got != want {
		t.Errorf("one diagnostic reads %q, want %q", got, want)
	}

	two := diag.List{{Message: "first"}, {Message: "second"}}
	if got, want := two.Err().Error(), "first\nsecond"; got != want {
		t.Errorf("two diagnostics read %q, want %q", got, want)
	}
}

// errors.As has to reach a diagnostic inside the error a list turned into,
// because that is how a caller asks what sort of failure it was without
// knowing this package returned it.
func TestErrorsAsReachesADiagnostic(t *testing.T) {
	err := fmt.Errorf("loading configuration: %w", diag.List{
		{Message: "first"},
		{Code: "MZ1042", Message: "second"},
	}.Err())

	var d diag.Diagnostic
	if !errors.As(err, &d) {
		t.Fatal("errors.As did not reach a diagnostic")
	}
	if d.Message != "first" {
		t.Errorf("errors.As found %q, want the first one", d.Message)
	}
}

func TestOf(t *testing.T) {
	one := diag.Diagnostic{Code: "MZ1042", Message: "unknown setting"}
	two := diag.Diagnostic{Severity: diag.Warning, Message: "deprecated setting"}

	for _, tt := range []struct {
		name string
		err  error
		want diag.List
	}{
		{"nil", nil, nil},
		{"a diagnostic", one, diag.List{one}},
		{"a pointer to one", &one, diag.List{one}},
		{"a list", diag.List{one, two}.Err(), diag.List{one, two}},
		{"a wrapped list", fmt.Errorf("loading: %w", diag.List{one, two}.Err()), diag.List{one, two}},
		{"a wrapped diagnostic", fmt.Errorf("loading: %w", one), diag.List{one}},
		{"joined", errors.Join(one, two), diag.List{one, two}},
		{"joined with something else", errors.Join(one, errors.New("no such file")), diag.List{one}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := diag.Of(tt.err)
			if len(got) != len(tt.want) {
				t.Fatalf("Of() gave %d diagnostics, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i].Message != tt.want[i].Message || got[i].Code != tt.want[i].Code {
					t.Errorf("Of()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// An error carrying no diagnostic becomes one, so a command under --json
// always has a document to print rather than a choice to make.
func TestOfAnOrdinaryError(t *testing.T) {
	got := diag.Of(fmt.Errorf("open config/app.toml: %w", errors.New("no such file or directory")))
	if len(got) != 1 {
		t.Fatalf("Of() gave %d diagnostics, want 1", len(got))
	}
	if want := "open config/app.toml: no such file or directory"; got[0].Message != want {
		t.Errorf("message is %q, want %q", got[0].Message, want)
	}
	if got[0].Severity != diag.Error {
		t.Errorf("severity is %v, want error", got[0].Severity)
	}
}

// An error that wraps itself hangs errors.Is and errors.As too, but a command
// line tool that stops responding is a worse answer than one that reports the
// outer error, so this walk gives up rather than going round.
func TestOfStopsOnAnErrorThatWrapsItself(t *testing.T) {
	c := &cyclic{}
	c.inner = c
	if got := diag.Of(c); len(got) != 1 || got[0].Message != "round and round" {
		t.Errorf("Of() gave %v, want the outer error once", got)
	}
}

type cyclic struct{ inner error }

func (c *cyclic) Error() string { return "round and round" }
func (c *cyclic) Unwrap() error { return c.inner }

func TestListSummary(t *testing.T) {
	for _, tt := range []struct {
		name string
		l    diag.List
		want string
	}{
		{"nothing", nil, ""},
		{"one error", diag.List{{}}, "1 error"},
		{"two errors", diag.List{{}, {}}, "2 errors"},
		{
			"one of each",
			diag.List{{}, {Severity: diag.Warning}, {Severity: diag.Note}},
			"1 error, 1 warning, 1 note",
		},
		{
			"warnings and notes only",
			diag.List{{Severity: diag.Warning}, {Severity: diag.Note}, {Severity: diag.Note}},
			"1 warning, 2 notes",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.Summary(); got != tt.want {
				t.Errorf("Summary() = %q, want %q", got, tt.want)
			}
		})
	}
}
