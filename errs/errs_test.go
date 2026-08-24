package errs

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

var cause = errors.New("dial tcp 10.0.0.4:5432: connection refused")

func TestErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		err  *Error
		want string
	}{
		{"message and cause", Wrap(cause, Unavailable, "db.down", "the database is unavailable"),
			"the database is unavailable: dial tcp 10.0.0.4:5432: connection refused"},
		{"message only", New(NotFound, "post.not_found", "No such post."), "No such post."},
		{"cause only", Wrap(cause, Unavailable, "db.down", ""), cause.Error()},
		{"code only", &Error{Kind: Invalid, Code: "body.malformed"}, "body.malformed"},
		{"nothing at all", &Error{}, "internal"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("%s: Error() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestUnwrap(t *testing.T) {
	err := Wrap(cause, Unavailable, "db.down", "the database is unavailable")
	if !errors.Is(err, cause) {
		t.Error("the cause is not reachable through errors.Is")
	}
	if got := errors.Unwrap(err); got != cause {
		t.Errorf("Unwrap() = %v, want the cause", got)
	}
	if got := errors.Unwrap(New(Invalid, "", "bad")); got != nil {
		t.Errorf("Unwrap() of an error with no cause = %v", got)
	}
}

func TestIs(t *testing.T) {
	err := New(NotFound, "post.not_found", "No such post.")

	if !errors.Is(err, NotFound) {
		t.Error("does not match its own kind")
	}
	if !errors.Is(err, New(NotFound, "post.not_found", "")) {
		t.Error("does not match the same code")
	}
	if errors.Is(err, New(NotFound, "user.not_found", "")) {
		t.Error("matches a different code with the same kind")
	}
	if !errors.Is(err, &Error{Kind: NotFound}) {
		t.Error("does not match a target with no code")
	}
	if errors.Is(err, cause) {
		t.Error("matches an unrelated error")
	}

	// A code matches across kinds, since a code is the more specific of the
	// two and a caller that wrote one meant it.
	moved := New(Conflict, "post.not_found", "")
	if !errors.Is(moved, New(NotFound, "post.not_found", "")) {
		t.Error("a code does not match when the kinds differ")
	}
}

func TestShorthands(t *testing.T) {
	cases := []struct {
		err  *Error
		kind Kind
	}{
		{NotFoundf("no post with id %d", 12), NotFound},
		{Invalidf("page must be a number, got %q", "x"), Invalid},
		{Forbiddenf("you may not edit post %d", 12), Forbidden},
		{Conflictf("post %d has moved on", 12), Conflict},
		{Internalf("the cache is %s", "on fire"), Internal},
	}
	for _, c := range cases {
		if c.err.Kind != c.kind {
			t.Errorf("%v is a %s, want %s", c.err, c.err.Kind, c.kind)
		}
		if c.err.Code != "" {
			t.Errorf("%v arrived with the code %q", c.err, c.err.Code)
		}
	}
	if got := NotFoundf("no post with id %d", 12).Error(); got != "no post with id 12" {
		t.Errorf("Error() = %q", got)
	}
}

// TestMessageWithNoArguments is why sprintf checks the argument count. A
// message that happens to contain a percent sign is a message, not a format.
func TestMessageWithNoArguments(t *testing.T) {
	err := Invalidf("discount must be under 100%")
	if got, want := err.Error(), "discount must be under 100%"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestNewf(t *testing.T) {
	err := Newf(TooLarge, "upload.too_large", "that file is %d MB, and the limit is %d", 12, 8)
	if err.Kind != TooLarge || err.Code != "upload.too_large" {
		t.Errorf("Newf produced %+v", err)
	}
	if got, want := err.Error(), "that file is 12 MB, and the limit is 8"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestWrapNilCause is the trap this package refuses to set. Wrap of nil must
// not hand back a *Error that is nil, because it would be an error that is not
// nil with nothing wrong.
func TestWrapNilCause(t *testing.T) {
	err := Wrap(nil, Internal, "boom", "something went wrong")
	if err == nil {
		t.Fatal("Wrap returned nil, which is a typed nil waiting to happen")
	}
	var as error = err
	if as == nil {
		t.Fatal("the error is nil as an interface")
	}
	if got := err.Error(); got != "something went wrong" {
		t.Errorf("Error() = %q", got)
	}
}

func TestWithCode(t *testing.T) {
	base := NotFoundf("no such post")
	coded := base.WithCode("post.not_found")

	if base.Code != "" {
		t.Errorf("the original grew a code: %q", base.Code)
	}
	if coded.Code != "post.not_found" {
		t.Errorf("the copy has the code %q", coded.Code)
	}
	if coded.Msg != base.Msg || coded.Kind != base.Kind {
		t.Error("the copy lost something the original had")
	}
}

func TestWithField(t *testing.T) {
	base := New(Unprocessable, "validation.failed", "That form has errors.")
	one := base.WithField("title", "required", "Title is required.")
	two := one.WithField("body", "min", "Body must be at least 20 characters.")

	if len(base.Fields) != 0 {
		t.Errorf("the original grew %d fields", len(base.Fields))
	}
	if len(one.Fields) != 1 {
		t.Errorf("the first copy has %d fields, want 1", len(one.Fields))
	}
	if len(two.Fields) != 2 {
		t.Fatalf("the second copy has %d fields, want 2", len(two.Fields))
	}
	if two.Fields[0] != (Field{"title", "required", "Title is required."}) {
		t.Errorf("the first field is %+v", two.Fields[0])
	}

	// The copies must not share the array underneath, or adding a second
	// field to one branch would overwrite the second field of the other.
	other := one.WithField("slug", "taken", "That slug is in use.")
	if two.Fields[1].Name != "body" {
		t.Errorf("adding a field to one copy changed another: %+v", two.Fields[1])
	}
	if other.Fields[1].Name != "slug" {
		t.Errorf("the other copy has %+v", other.Fields[1])
	}
}

func TestWithFields(t *testing.T) {
	base := New(Unprocessable, "validation.failed", "That form has errors.")
	err := base.WithFields(
		Field{"title", "required", "Title is required."},
		Field{"body", "min", "Body must be at least 20 characters."},
	)
	if len(base.Fields) != 0 {
		t.Errorf("the original grew %d fields", len(base.Fields))
	}
	if len(err.Fields) != 2 {
		t.Fatalf("the copy has %d fields, want 2", len(err.Fields))
	}
	if err.Fields[1].Code != "min" {
		t.Errorf("the second field is %+v", err.Fields[1])
	}
}

func TestWithMeta(t *testing.T) {
	base := NotFoundf("no such post")
	one := base.WithMeta("id", 12)
	two := one.WithMeta("table", "posts")

	if base.Meta != nil {
		t.Errorf("the original grew a map: %v", base.Meta)
	}
	if len(one.Meta) != 1 {
		t.Errorf("the first copy has %v", one.Meta)
	}
	if len(two.Meta) != 2 || two.Meta["id"] != 12 || two.Meta["table"] != "posts" {
		t.Errorf("the second copy has %v", two.Meta)
	}

	other := one.WithMeta("id", 99)
	if two.Meta["id"] != 12 {
		t.Errorf("writing to one copy changed another: %v", two.Meta)
	}
	if other.Meta["id"] != 99 {
		t.Errorf("the other copy has %v", other.Meta)
	}
}

func TestWithRetry(t *testing.T) {
	base := New(RateLimited, "rate.limited", "Slow down.")
	err := base.WithRetry(30 * time.Second)

	if base.Retry != 0 {
		t.Errorf("the original grew a retry of %s", base.Retry)
	}
	if err.Retry != 30*time.Second {
		t.Errorf("Retry is %s, want 30s", err.Retry)
	}
}

// TestDecorationKeepsTheStack checks that describing an error does not lose
// where it happened.
func TestDecorationKeepsTheStack(t *testing.T) {
	err := Internalf("the cache is on fire").WithCode("cache.fire").WithMeta("shard", 3)
	if len(err.StackTrace()) == 0 {
		t.Fatal("the copy has no stack")
	}
	if err.StackTrace()[0].Func != frameOf(t) {
		t.Errorf("the stack starts at %s", err.StackTrace()[0].Func)
	}
}

// TestFormatting checks that an *Error prints usefully through fmt, which is
// where most of them end up.
func TestFormatting(t *testing.T) {
	err := Wrap(cause, Unavailable, "db.down", "the database is unavailable")
	if got, want := fmt.Sprintf("%v", err), err.Error(); got != want {
		t.Errorf("%%v = %q, want %q", got, want)
	}
	if got, want := fmt.Sprintf("%s", error(err)), err.Error(); got != want {
		t.Errorf("%%s = %q, want %q", got, want)
	}
}
