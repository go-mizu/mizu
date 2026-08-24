package errs

import (
	"errors"
	"log/slog"
)

// A Kind is what went wrong, from a closed set that every transport knows how
// to answer.
//
// It is the one classification in the toolkit that fans out. An HTTP status, an
// RPC code, a log level, whether trying again is worth anything, and whether
// the message may be shown to a user all come from the Kind and nothing else,
// so a new failure in a driver is one decision rather than five.
//
// A Kind is an error, so a caller matches one without this package exporting a
// mutable sentinel variable for each:
//
//	if errors.Is(err, errs.NotFound) {
//
// Returning a bare Kind as an error is allowed and means what it says, though
// an error worth returning usually has something to say about itself and is
// better made with [New].
type Kind uint8

// The kinds. The set is closed: adding one is a change to every transport that
// reads the table, so it goes through the decision register rather than through
// a pull request that needs a new status code.
const (
	// Internal is a bug, a broken invariant, or a dependency that failed in a
	// way nobody planned for. It is the zero value because a failure that
	// nobody classified is not one to show a user.
	Internal Kind = iota

	// Invalid is a request that is malformed: a body that will not decode, a
	// parameter that is not a number, a header that is missing.
	Invalid

	// Unauthenticated is a request with no identity, or one that could not be
	// established. The caller has not said who they are.
	Unauthenticated

	// Forbidden is a request from someone known who may not do this. Saying
	// who they are again will not help.
	Forbidden

	// NotFound is a resource that is not here. It is also the answer for a
	// resource the caller may not know about, since 403 on a record that
	// exists tells them it exists.
	NotFound

	// Conflict is a request that lost a race: a stale version, a row that
	// moved, a state machine that has since advanced.
	Conflict

	// Exists is a request to create something that is already there, which is
	// separate from Conflict because the caller usually wants the existing one
	// rather than a retry.
	Exists

	// Precondition is a request whose stated condition does not hold, such as
	// an If-Match that does not match.
	Precondition

	// TooLarge is a body, a file or a batch above the limit for it.
	TooLarge

	// Unprocessable is a request that parsed and is still wrong: the fields
	// are the right shape and the wrong values. Validation lands here and
	// carries [Field] values saying which.
	Unprocessable

	// RateLimited is a caller asking too often. It is retryable, and how long
	// to wait is in [Error.Retry].
	RateLimited

	// Unsupported is a route, a method or a feature that is not implemented
	// here. It is about the server and not about the request.
	Unsupported

	// Unavailable is a dependency that is down or a server that is shutting
	// down. It is retryable, and the same call may work in a moment.
	Unavailable

	// Timeout is an operation that ran out of time, whether the deadline was
	// the caller's or one further down. It is retryable.
	Timeout

	// Canceled is work that was stopped on purpose, almost always because the
	// caller went away. It is not a failure to report, and it is spelled the
	// way [context.Canceled] is.
	Canceled
)

// kinds is the whole table, indexed by Kind. Every column is here so that the
// HTTP status and the RPC code cannot drift apart in two files.
var kinds = [...]struct {
	name      string
	status    int
	rpc       RPCCode
	level     slog.Level
	retryable bool
	safe      bool
}{
	Internal:        {"internal", 500, RPCInternal, slog.LevelError, false, false},
	Invalid:         {"invalid", 400, RPCInvalidArgument, slog.LevelInfo, false, true},
	Unauthenticated: {"unauthenticated", 401, RPCUnauthenticated, slog.LevelInfo, false, true},
	Forbidden:       {"forbidden", 403, RPCPermissionDenied, slog.LevelInfo, false, true},
	NotFound:        {"not_found", 404, RPCNotFound, slog.LevelInfo, false, true},
	Conflict:        {"conflict", 409, RPCAborted, slog.LevelInfo, false, true},
	Exists:          {"exists", 409, RPCAlreadyExists, slog.LevelInfo, false, true},
	Precondition:    {"precondition", 412, RPCFailedPrecondition, slog.LevelInfo, false, true},
	TooLarge:        {"too_large", 413, RPCResourceExhausted, slog.LevelInfo, false, true},
	Unprocessable:   {"unprocessable", 422, RPCInvalidArgument, slog.LevelInfo, false, true},
	RateLimited:     {"rate_limited", 429, RPCResourceExhausted, slog.LevelWarn, true, true},
	Unsupported:     {"unsupported", 501, RPCUnimplemented, slog.LevelError, false, true},
	Unavailable:     {"unavailable", 503, RPCUnavailable, slog.LevelError, true, false},
	Timeout:         {"timeout", 504, RPCDeadlineExceeded, slog.LevelWarn, true, false},
	Canceled:        {"canceled", 499, RPCCanceled, slog.LevelDebug, false, false},
}

// row is the table entry for k. A Kind that came from somewhere other than the
// constants above, such as a number read off the wire, is treated as Internal,
// because the alternative is an index out of range in an error path.
func (k Kind) row() int {
	if int(k) >= len(kinds) {
		return int(Internal)
	}
	return int(k)
}

// String is the kind's name in lower case with underscores, which is what goes
// in a log record and in the code field of a problem document.
func (k Kind) String() string { return kinds[k.row()].name }

// Error makes a Kind usable with [errors.Is] as the target. The text is the
// same as [Kind.String].
func (k Kind) Error() string { return k.String() }

// Status is the HTTP status for this kind. Canceled is 499, which is not in
// any RFC and is what nginx and every proxy after it settled on for a client
// that hung up.
func (k Kind) Status() int { return kinds[k.row()].status }

// RPCCode is the gRPC status code for this kind, by number.
func (k Kind) RPCCode() RPCCode { return kinds[k.row()].rpc }

// Level is how loudly a failure of this kind is logged. It is what decides
// whether a stack is captured, since a stack nobody will print is work nobody
// asked for.
func (k Kind) Level() slog.Level { return kinds[k.row()].level }

// Retryable is whether the same call, unchanged, could work later. It is true
// for RateLimited, Unavailable and Timeout, and a caller that retries anything
// else is retrying a decision rather than a hiccup.
func (k Kind) Retryable() bool { return kinds[k.row()].retryable }

// Safe is whether the message may be shown to whoever made the request.
//
// It is false for Internal, Unavailable and Timeout, where the detail is about
// the inside of the system: a table name, a host, a driver. Those go to the log
// with the request id attached, and the response carries the id and nothing
// else.
func (k Kind) Safe() bool { return kinds[k.row()].safe }

// MarshalText writes the kind's name, so a Kind in a JSON document reads as
// not_found rather than 4.
func (k Kind) MarshalText() ([]byte, error) { return []byte(k.String()), nil }

// UnmarshalText reads a name written by [Kind.MarshalText]. A name that is not
// one of the kinds is an error rather than Internal, because reading rubbish
// quietly is how a taxonomy stops meaning anything.
func (k *Kind) UnmarshalText(text []byte) error {
	for i, row := range kinds {
		if row.name == string(text) {
			*k = Kind(i)
			return nil
		}
	}
	return errors.New("errs: " + string(text) + " is not a kind")
}
