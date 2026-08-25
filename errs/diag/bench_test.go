package diag_test

import (
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

// A diagnostic is on the failure path, so the numbers that matter are the ones
// for rendering a list rather than for building one. The budget is a run that
// found a few hundred problems still printing in well under the time it took
// to find them.

func benchSource(string) ([]byte, error) {
	return []byte("[database]\nurl = \"postgres://localhost/blog\"\npool_size = 25\n"), nil
}

func benchList(n int) diag.List {
	l := make(diag.List, 0, n)
	for i := range n {
		l = append(l, diag.Diagnostic{
			Code:    "MZ1042",
			Message: fmt.Sprintf("unknown config key %q", fmt.Sprintf("database.key_%d", i)),
			File:    "config/app.toml",
			Range:   diag.Span(3, 1, 9),
			Detail:  "no such field in Config.Database",
			Suggestions: []diag.Suggestion{{
				Message:    `did you mean "max_open_conns"?`,
				Confidence: diag.High,
			}},
			Fix: "mizu fix config --rule=rename-key",
		})
	}
	return l
}

func BenchmarkText(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		l := benchList(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := diag.Text(io.Discard, l, diag.WithSource(benchSource)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// The same list where every diagnostic reads alike, which is the shape rule
// five exists for and the one that should get cheaper, not dearer.
func BenchmarkTextGrouped(b *testing.B) {
	l := make(diag.List, 200)
	for i := range l {
		l[i] = diag.Diagnostic{Code: "MZ1042", Message: "unknown config key", File: "config/app.toml", Range: diag.Span(3, 1, 9)}
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := diag.Text(io.Discard, l, diag.WithSource(benchSource)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTextWithColor(b *testing.B) {
	l := benchList(10)
	b.ReportAllocs()
	for b.Loop() {
		if err := diag.Text(io.Discard, l, diag.WithSource(benchSource), diag.WithColor(true)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSON(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		l := benchList(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := diag.JSON(io.Discard, l); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkListSort(b *testing.B) {
	src := benchList(100)
	for i := range src {
		src[i].Severity = diag.Severity(i % 3)
		src[i].Range = diag.Span(100-i, 1, 9)
	}
	l := make(diag.List, len(src))
	b.ReportAllocs()
	for b.Loop() {
		copy(l, src)
		l.Sort()
	}
}

// A registry lookup is a binary search over a slice that is kept in order, so
// it allocates nothing and it does not get slower in a way anybody will notice
// as the table grows. There is no budget row for it because there is nothing to
// promise: an operation that is a handful of comparisons is not a number worth
// arguing about, and if it ever becomes one the fix is a map.
func BenchmarkLookup(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, ok := diag.Lookup("MZ1042"); !ok {
			b.Fatal("MZ1042 is not in the registry")
		}
	}
}

// Unwrapping is what a command does once, at the top, to decide how to print
// what came back up.
func BenchmarkOf(b *testing.B) {
	err := fmt.Errorf("loading configuration: %w", benchList(10).Err())
	b.ReportAllocs()
	for b.Loop() {
		if len(diag.Of(err)) != 10 {
			b.Fatal("lost one")
		}
	}
}

// benchSettings is a candidate set the size of a real configuration: every
// setting a middling application declares, which is what Suggest walks before
// it can say anything.
var benchSettings = func() []string {
	sections := []string{"app", "database", "cache", "queue", "mail", "log", "session"}
	names := []string{
		"driver", "url", "max_open_conns", "max_idle_conns", "conn_max_lifetime",
		"connect_timeout", "read_timeout", "ssl_mode", "log_queries", "name",
	}
	out := make([]string, 0, len(sections)*len(names))
	for _, s := range sections {
		for _, n := range names {
			out = append(out, s+"."+n)
		}
	}
	return out
}()

// Suggest runs once, on the failure path, against a candidate set somebody
// typed a name into. Seventy candidates at fifteen characters each is the shape
// of the work, and the number that matters is that it is nowhere near the cost
// of the error message it goes in.
func BenchmarkSuggest(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if len(diag.Suggest("database.max_conns", slices.Values(benchSettings))) == 0 {
			b.Fatal("found nothing to suggest")
		}
	}
}

// The other half of the same path: a name close to nothing, where every
// candidate is measured and none of them qualifies.
func BenchmarkSuggestNothingClose(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if len(diag.Suggest("wildly.different", slices.Values(benchSettings))) != 0 {
			b.Fatal("it suggested something")
		}
	}
}

func BenchmarkDistance(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if diag.Distance("database.max_conns", "database.max_open_conns") != 5 {
			b.Fatal("wrong distance")
		}
	}
}
