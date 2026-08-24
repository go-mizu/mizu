package errs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
	"testing"
	"time"
)

func TestKindOf(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want Kind
	}{
		{"nil", nil, Internal},
		{"a plain error", errors.New("boom"), Internal},
		{"one of ours", NotFoundf("no such post"), NotFound},
		{"ours, wrapped by somebody else", fmt.Errorf("loading: %w", NotFoundf("no such post")), NotFound},
		{"ours, wrapping something", Wrap(errors.New("boom"), Conflict, "", ""), Conflict},
		{"the outermost of ours wins", Wrap(NotFoundf("inner"), Forbidden, "", ""), Forbidden},
		{"context canceled", context.Canceled, Canceled},
		{"context canceled, wrapped", fmt.Errorf("reading: %w", context.Canceled), Canceled},
		{"deadline exceeded", context.DeadlineExceeded, Timeout},
		{"a missing file", &fs.PathError{Op: "open", Path: "/x", Err: fs.ErrNotExist}, NotFound},
		{"a file we may not read", &fs.PathError{Op: "open", Path: "/x", Err: fs.ErrPermission}, Forbidden},
		{"a file already there", &fs.PathError{Op: "link", Path: "/x", Err: fs.ErrExist}, Exists},
		{"a truncated read", io.ErrUnexpectedEOF, Unavailable},
		{"anything that timed out", os.ErrDeadlineExceeded, Timeout},
		{"something that did not time out", notTimeout{}, Internal},
		{"joined, one of ours inside", errors.Join(errors.New("first"), NotFoundf("second")), NotFound},
	}
	for _, c := range cases {
		if got := KindOf(c.err); got != c.want {
			t.Errorf("%s: KindOf = %s, want %s", c.name, got, c.want)
		}
	}
}

type notTimeout struct{}

func (notTimeout) Error() string { return "not a timeout" }
func (notTimeout) Timeout() bool { return false }

func TestCodeOf(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"a plain error", errors.New("boom"), ""},
		{"no code", NotFoundf("no such post"), ""},
		{"a code", NotFoundf("no such post").WithCode("post.not_found"), "post.not_found"},
		{"wrapped by somebody else", fmt.Errorf("loading: %w", NotFoundf("x").WithCode("post.not_found")), "post.not_found"},
		{"the outermost code wins", Wrap(NotFoundf("x").WithCode("post.not_found"), Internal, "handler.failed", ""), "handler.failed"},
		{"the code below counts when the one above has none", Wrap(NotFoundf("x").WithCode("post.not_found"), Internal, "", ""), "post.not_found"},
	}
	for _, c := range cases {
		if got := CodeOf(c.err); got != c.want {
			t.Errorf("%s: CodeOf = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestRetryable(t *testing.T) {
	yes := []error{
		New(RateLimited, "", ""),
		New(Unavailable, "", ""),
		New(Timeout, "", ""),
		context.DeadlineExceeded,
		os.ErrDeadlineExceeded,
	}
	for _, err := range yes {
		if !Retryable(err) {
			t.Errorf("%v is not retryable and should be", err)
		}
	}
	no := []error{nil, errors.New("boom"), NotFoundf("x"), context.Canceled}
	for _, err := range no {
		if Retryable(err) {
			t.Errorf("%v is retryable and should not be", err)
		}
	}
}

func TestRetryAfter(t *testing.T) {
	err := New(RateLimited, "rate.limited", "Slow down.").WithRetry(30 * time.Second)
	if d, ok := RetryAfter(err); !ok || d != 30*time.Second {
		t.Errorf("RetryAfter = %s, %v, want 30s, true", d, ok)
	}
	if d, ok := RetryAfter(fmt.Errorf("outer: %w", err)); !ok || d != 30*time.Second {
		t.Errorf("through a wrapper: RetryAfter = %s, %v", d, ok)
	}

	for _, err := range []error{nil, errors.New("boom"), New(RateLimited, "", "")} {
		if d, ok := RetryAfter(err); ok {
			t.Errorf("%v: RetryAfter = %s, %v, want nothing", err, d, ok)
		}
	}
}

func TestFieldsOf(t *testing.T) {
	inner := New(Unprocessable, "validation.failed", "That form has errors.").
		WithField("title", "required", "Title is required.")
	outer := Wrap(inner, Unprocessable, "", "")

	if got := Fields(outer); len(got) != 1 || got[0].Name != "title" {
		t.Errorf("Fields = %+v", got)
	}
	if got := Fields(nil); got != nil {
		t.Errorf("Fields(nil) = %+v", got)
	}
	if got := Fields(errors.New("boom")); got != nil {
		t.Errorf("Fields of a plain error = %+v", got)
	}
}

func TestMetaOf(t *testing.T) {
	inner := NotFoundf("no such post").WithMeta("id", 12)
	outer := fmt.Errorf("loading: %w", inner)

	if got := Meta(outer); len(got) != 1 || got["id"] != 12 {
		t.Errorf("Meta = %v", got)
	}
	if got := Meta(errors.New("boom")); got != nil {
		t.Errorf("Meta of a plain error = %v", got)
	}
}

// The mapper tests each use an error type of their own, because a mapper stays
// registered for the rest of the run and there is no way to take one back.

type duplicateKey struct{}

func (duplicateKey) Error() string { return "duplicate key value violates unique constraint" }

type lockTimeout struct{}

func (lockTimeout) Error() string { return "lock wait timeout exceeded" }

func TestRegisterMapper(t *testing.T) {
	RegisterMapper(func(err error) (Kind, string, bool) {
		var dup duplicateKey
		if errors.As(err, &dup) {
			return Exists, "db.duplicate", true
		}
		return 0, "", false
	})

	err := duplicateKey{}
	if got := KindOf(err); got != Exists {
		t.Errorf("KindOf = %s, want exists", got)
	}
	if got := CodeOf(err); got != "db.duplicate" {
		t.Errorf("CodeOf = %q, want db.duplicate", got)
	}
	if got := KindOf(fmt.Errorf("saving: %w", err)); got != Exists {
		t.Errorf("through a wrapper: KindOf = %s", got)
	}

	// A mapper does not get a say about an error that already knows what it is.
	classified := Wrap(err, Conflict, "post.stale", "")
	if got := KindOf(classified); got != Conflict {
		t.Errorf("a mapper overrode an explicit kind: %s", got)
	}
}

func TestMapperOrder(t *testing.T) {
	RegisterMapper(func(err error) (Kind, string, bool) {
		var lock lockTimeout
		if errors.As(err, &lock) {
			return Timeout, "db.lock_timeout", true
		}
		return 0, "", false
	})
	RegisterMapper(func(err error) (Kind, string, bool) {
		var lock lockTimeout
		if errors.As(err, &lock) {
			return Conflict, "db.later", true
		}
		return 0, "", false
	})

	if got := KindOf(lockTimeout{}); got != Timeout {
		t.Errorf("KindOf = %s, want the first mapper's answer", got)
	}
}

// TestMapperAfterBuiltins checks the order the documentation promises. A
// mapper cannot change what context.Canceled means.
func TestMapperAfterBuiltins(t *testing.T) {
	RegisterMapper(func(err error) (Kind, string, bool) {
		if errors.Is(err, context.Canceled) {
			return Conflict, "no", true
		}
		return 0, "", false
	})
	if got := KindOf(context.Canceled); got != Canceled {
		t.Errorf("a mapper overrode a standard library error: %s", got)
	}
}

func TestRegisterNilMapper(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("registering nil did not panic")
		}
	}()
	RegisterMapper(nil)
}

// TestRegisterWhileReading is for the race detector. A driver registering a
// mapper at startup must not tear a slice out from under a request in flight.
func TestRegisterWhileReading(t *testing.T) {
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				KindOf(errors.New("boom"))
			}
		}()
	}
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RegisterMapper(func(error) (Kind, string, bool) { return Kind(i), "", false })
		}()
	}
	wg.Wait()
}

func TestWalkJoined(t *testing.T) {
	target := NotFoundf("the one we want").WithCode("post.not_found")
	err := errors.Join(
		errors.New("first"),
		fmt.Errorf("second: %w", errors.New("nested")),
		fmt.Errorf("third: %w", target),
	)
	if got := CodeOf(err); got != "post.not_found" {
		t.Errorf("CodeOf = %q, want post.not_found", got)
	}

	// And a walk that finds nothing has to end rather than loop.
	if got := CodeOf(errors.Join(errors.New("a"), errors.New("b"))); got != "" {
		t.Errorf("CodeOf = %q, want nothing", got)
	}
}
