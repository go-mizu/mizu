package errs

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// frameOf is what a stack captured inside a test function starts with.
func frameOf(t *testing.T) string {
	t.Helper()
	return "github.com/go-mizu/mizu/errs." + t.Name()
}

// TestStackStartsAtTheCaller is the one that catches a wrong skip count. The
// first frame is where the error was made, not anything inside this package.
func TestStackStartsAtTheCaller(t *testing.T) {
	for _, err := range []*Error{
		New(Internal, "boom", "something went wrong"),
		Newf(Internal, "boom", "something went %s", "wrong"),
		Internalf("something went wrong"),
		Wrap(errors.New("cause"), Internal, "boom", "something went wrong"),
		Wrapf(errors.New("cause"), Internal, "boom", "something went %s", "wrong"),
	} {
		frames := err.StackTrace()
		if len(frames) == 0 {
			t.Fatalf("%v has no stack", err)
		}
		if frames[0].Func != frameOf(t) {
			t.Errorf("the stack starts at %s, want %s", frames[0].Func, frameOf(t))
		}
		if !strings.HasSuffix(frames[0].File, "stack_test.go") {
			t.Errorf("the first frame is in %s", frames[0].File)
		}
		if frames[0].Line == 0 {
			t.Error("the first frame has no line")
		}
	}
}

// TestStackOnlyForErrorLevel is the capture policy. A 404 is not a bug and
// nobody is going to read a stack for one.
func TestStackOnlyForErrorLevel(t *testing.T) {
	for _, k := range []Kind{Internal, Unsupported, Unavailable} {
		if got := New(k, "", "").StackTrace(); len(got) == 0 {
			t.Errorf("%s captured no stack", k)
		}
	}
	for _, k := range []Kind{Invalid, Unauthenticated, Forbidden, NotFound, Conflict,
		Exists, Precondition, TooLarge, Unprocessable, RateLimited, Timeout, Canceled} {
		if got := New(k, "", "").StackTrace(); got != nil {
			t.Errorf("%s captured %d frames", k, len(got))
		}
	}
}

// TestDeepestStackWins is the other half of the policy. Wrapping an error that
// already knows where it happened must not overwrite that with where it was
// described.
func TestDeepestStackWins(t *testing.T) {
	inner := deepFailure()
	outer := Wrap(inner, Internal, "handler.failed", "could not answer")

	if outer.stack != nil {
		t.Error("wrapping captured a second stack")
	}
	frames := Stack(outer)
	if len(frames) == 0 {
		t.Fatal("the chain has no stack")
	}
	if !strings.HasSuffix(frames[0].Func, "deepFailure") {
		t.Errorf("the stack starts at %s, want deepFailure", frames[0].Func)
	}
}

func deepFailure() error { return Internalf("the disk went away") }

// TestStackThroughAnotherWrapper checks that a stack below somebody else's
// wrapper still counts as one, so fmt.Errorf in between does not cause a
// second capture.
func TestStackThroughAnotherWrapper(t *testing.T) {
	inner := deepFailure()
	middle := fmt.Errorf("loading the page: %w", inner)
	outer := Wrap(middle, Internal, "", "could not answer")

	if outer.stack != nil {
		t.Error("wrapping captured a second stack")
	}
	if len(Stack(outer)) == 0 {
		t.Error("the stack below the wrapper is not reachable")
	}
}

func TestNoStack(t *testing.T) {
	if got := NotFoundf("no such post").StackTrace(); got != nil {
		t.Errorf("a 404 captured %d frames", len(got))
	}
	if got := Stack(NotFoundf("no such post")); got != nil {
		t.Errorf("Stack found %d frames", len(got))
	}
	if got := Stack(nil); got != nil {
		t.Errorf("Stack(nil) found %d frames", len(got))
	}
	if got := Stack(errors.New("plain")); got != nil {
		t.Errorf("Stack of a plain error found %d frames", len(got))
	}
}

func TestFrameString(t *testing.T) {
	f := Frame{Func: "blog/post.(*Store).ByID", File: "/src/blog/post/store.go", Line: 41}
	if got, want := f.String(), "/src/blog/post/store.go:41 blog/post.(*Store).ByID"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestStackDepth checks the cap holds, since a stack is a fixed cost per error
// that captures one.
func TestStackDepth(t *testing.T) {
	err := recurse(64)
	if got := len(err.StackTrace()); got > depth {
		t.Errorf("captured %d frames, want at most %d", got, depth)
	}
}

func recurse(n int) *Error {
	if n == 0 {
		return Internalf("bottom")
	}
	return recurse(n - 1)
}
