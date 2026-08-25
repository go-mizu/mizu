package diag_test

import (
	"fmt"
	"io"
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
