package errs

import (
	"errors"
	"log/slog"
	"testing"
)

// TestKindTable is the whole taxonomy written out a second time, by hand, so
// that a change to the table in kind.go has to be made twice on purpose. It is
// the table doc 38 publishes and every transport reads.
func TestKindTable(t *testing.T) {
	cases := []struct {
		kind      Kind
		name      string
		status    int
		rpc       RPCCode
		level     slog.Level
		retryable bool
		safe      bool
	}{
		{Internal, "internal", 500, RPCInternal, slog.LevelError, false, false},
		{Invalid, "invalid", 400, RPCInvalidArgument, slog.LevelInfo, false, true},
		{Unauthenticated, "unauthenticated", 401, RPCUnauthenticated, slog.LevelInfo, false, true},
		{Forbidden, "forbidden", 403, RPCPermissionDenied, slog.LevelInfo, false, true},
		{NotFound, "not_found", 404, RPCNotFound, slog.LevelInfo, false, true},
		{Conflict, "conflict", 409, RPCAborted, slog.LevelInfo, false, true},
		{Exists, "exists", 409, RPCAlreadyExists, slog.LevelInfo, false, true},
		{Precondition, "precondition", 412, RPCFailedPrecondition, slog.LevelInfo, false, true},
		{TooLarge, "too_large", 413, RPCResourceExhausted, slog.LevelInfo, false, true},
		{Unprocessable, "unprocessable", 422, RPCInvalidArgument, slog.LevelInfo, false, true},
		{RateLimited, "rate_limited", 429, RPCResourceExhausted, slog.LevelWarn, true, true},
		{Unsupported, "unsupported", 501, RPCUnimplemented, slog.LevelError, false, true},
		{Unavailable, "unavailable", 503, RPCUnavailable, slog.LevelError, true, false},
		{Timeout, "timeout", 504, RPCDeadlineExceeded, slog.LevelWarn, true, false},
		{Canceled, "canceled", 499, RPCCanceled, slog.LevelDebug, false, false},
	}
	if len(cases) != len(kinds) {
		t.Fatalf("the table has %d kinds and this test knows about %d", len(kinds), len(cases))
	}
	for _, c := range cases {
		if got := c.kind.String(); got != c.name {
			t.Errorf("%d.String() = %q, want %q", c.kind, got, c.name)
		}
		if got := c.kind.Status(); got != c.status {
			t.Errorf("%s.Status() = %d, want %d", c.name, got, c.status)
		}
		if got := c.kind.RPCCode(); got != c.rpc {
			t.Errorf("%s.RPCCode() = %s, want %s", c.name, got, c.rpc)
		}
		if got := c.kind.Level(); got != c.level {
			t.Errorf("%s.Level() = %s, want %s", c.name, got, c.level)
		}
		if got := c.kind.Retryable(); got != c.retryable {
			t.Errorf("%s.Retryable() = %v, want %v", c.name, got, c.retryable)
		}
		if got := c.kind.Safe(); got != c.safe {
			t.Errorf("%s.Safe() = %v, want %v", c.name, got, c.safe)
		}
	}
}

// TestKindZeroIsInternal is the reason Internal is first. A struct literal
// that says nothing about the kind must not produce a 400.
func TestKindZeroIsInternal(t *testing.T) {
	var k Kind
	if k != Internal {
		t.Errorf("the zero kind is %s, want internal", k)
	}
	if got := (&Error{}).Kind.Status(); got != 500 {
		t.Errorf("the zero error is a %d, want 500", got)
	}
}

// TestKindOutOfRange covers a kind that came from somewhere other than the
// constants, such as a number decoded off the wire.
func TestKindOutOfRange(t *testing.T) {
	k := Kind(200)
	if got := k.String(); got != "internal" {
		t.Errorf("String() = %q, want internal", got)
	}
	if got := k.Status(); got != 500 {
		t.Errorf("Status() = %d, want 500", got)
	}
	if got := k.RPCCode(); got != RPCInternal {
		t.Errorf("RPCCode() = %s, want Internal", got)
	}
	if got := k.Level(); got != slog.LevelError {
		t.Errorf("Level() = %s, want ERROR", got)
	}
	if k.Retryable() {
		t.Error("Retryable() is true")
	}
	if k.Safe() {
		t.Error("Safe() is true")
	}
}

// TestKindIsError is the reason there are no sentinel variables.
func TestKindIsError(t *testing.T) {
	err := New(NotFound, "post.not_found", "No such post.")
	if !errors.Is(err, NotFound) {
		t.Error("a not_found error does not match errs.NotFound")
	}
	if errors.Is(err, Conflict) {
		t.Error("a not_found error matches errs.Conflict")
	}
	if got := NotFound.Error(); got != "not_found" {
		t.Errorf("NotFound.Error() = %q", got)
	}

	// Through a wrap by another package, which is the case that matters.
	wrapped := errors.Join(errors.New("first"), err)
	if !errors.Is(wrapped, NotFound) {
		t.Error("a joined not_found error does not match errs.NotFound")
	}
}

func TestKindText(t *testing.T) {
	for i := range kinds {
		k := Kind(i)
		text, err := k.MarshalText()
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		var back Kind
		if err := back.UnmarshalText(text); err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if back != k {
			t.Errorf("%s came back as %s", k, back)
		}
	}

	var k Kind
	err := k.UnmarshalText([]byte("teapot"))
	if err == nil {
		t.Fatal("teapot read as a kind")
	}
	if got, want := err.Error(), "errs: teapot is not a kind"; got != want {
		t.Errorf("error is %q, want %q", got, want)
	}
}

func TestRPCCodeString(t *testing.T) {
	cases := map[RPCCode]string{
		RPCOK:                 "OK",
		RPCCanceled:           "Canceled",
		RPCUnknown:            "Unknown",
		RPCInvalidArgument:    "InvalidArgument",
		RPCDeadlineExceeded:   "DeadlineExceeded",
		RPCNotFound:           "NotFound",
		RPCAlreadyExists:      "AlreadyExists",
		RPCPermissionDenied:   "PermissionDenied",
		RPCResourceExhausted:  "ResourceExhausted",
		RPCFailedPrecondition: "FailedPrecondition",
		RPCAborted:            "Aborted",
		RPCOutOfRange:         "OutOfRange",
		RPCUnimplemented:      "Unimplemented",
		RPCInternal:           "Internal",
		RPCUnavailable:        "Unavailable",
		RPCDataLoss:           "DataLoss",
		RPCUnauthenticated:    "Unauthenticated",
		RPCCode(99):           "Code(99)",
	}
	for code, want := range cases {
		if got := code.String(); got != want {
			t.Errorf("RPCCode(%d).String() = %q, want %q", uint32(code), got, want)
		}
	}
}
