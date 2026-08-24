package errs

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// The budget in doc 06 is one allocation and 80 nanoseconds for a 404, and
// three allocations and 1.5 microseconds for an internal error with a stack.
// The difference between them is the whole reason the capture policy exists.

func BenchmarkNotFoundf(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = NotFoundf("no such post")
	}
}

func BenchmarkNotFoundfArgs(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = NotFoundf("no post with id %d", 12)
	}
}

func BenchmarkInternalf(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = Internalf("the cache is on fire")
	}
}

func BenchmarkWrap(b *testing.B) {
	err := errors.New("dial tcp: connection refused")
	b.ReportAllocs()
	for b.Loop() {
		sink = Wrap(err, Unavailable, "db.down", "the database is unavailable")
	}
}

// BenchmarkWrapClassified is wrapping something that already has a stack,
// which is the common case in a handler and costs no second capture.
func BenchmarkWrapClassified(b *testing.B) {
	err := Internalf("the disk went away")
	b.ReportAllocs()
	for b.Loop() {
		sink = Wrap(err, Internal, "handler.failed", "could not answer")
	}
}

func BenchmarkStackTrace(b *testing.B) {
	err := Internalf("the cache is on fire")
	b.ReportAllocs()
	for b.Loop() {
		frames = err.StackTrace()
	}
}

// BenchmarkKindOf is the lookup every error rendering starts with.
func BenchmarkKindOf(b *testing.B) {
	cases := []struct {
		name string
		err  error
	}{
		{"ours", NotFoundf("no such post")},
		{"wrapped", fmt.Errorf("loading: %w", fmt.Errorf("reading: %w", NotFoundf("no such post")))},
		{"stdlib", context.Canceled},
		{"unclassified", errors.New("boom")},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				kind = KindOf(c.err)
			}
		})
	}
}

func BenchmarkKindTable(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		status = NotFound.Status() + int(NotFound.RPCCode()) + int(NotFound.Level())
	}
}

func BenchmarkWithMeta(b *testing.B) {
	err := NotFoundf("no such post")
	b.ReportAllocs()
	for b.Loop() {
		sink = err.WithMeta("id", 12)
	}
}

var (
	sink   *Error
	kind   Kind
	status int
	frames []Frame
)
