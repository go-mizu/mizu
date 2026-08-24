package errs

import (
	"fmt"
	"maps"
	"slices"
	"time"
)

// An Error is a failure with everything a transport needs to answer for it.
//
// Build one with [New], [Wrap] or one of the shorthands, and decorate it with
// the With methods, each of which returns a copy. Every field is exported so
// that a renderer can read it, and none of them has to be set: the zero Error
// is an internal failure with nothing to say, which is what an unclassified
// failure is.
type Error struct {
	// Kind is what went wrong, and decides the status, the RPC code, the log
	// level and whether Msg may be shown to the caller.
	Kind Kind

	// Code is a stable machine name for this failure, such as
	// auth.token_expired. It is what a client switches on, what the docs site
	// has a page for, and what a test asserts. It is a string rather than a
	// constant so that application code can add its own.
	Code string

	// Msg is the message for whoever made the request, already translated. It
	// is not shown when the kind is not safe, so writing a table name here is
	// a leak waiting for the day somebody changes the kind.
	Msg string

	// Fields are the individual things wrong with the request, which is what
	// validation produces and what a form redisplays.
	Fields []Field

	// Meta is detail for a machine: an id that was not found, a limit that was
	// exceeded. It is included in the response only when the kind is safe.
	Meta map[string]any

	// Retry is how long to wait before trying again, and is set for
	// RateLimited and Unavailable. Zero means no advice rather than no wait.
	Retry time.Duration

	// Err is the cause, and is what [errors.Unwrap] returns.
	Err error

	// stack is where this was built, captured only for a kind that logs at
	// error level and only when nothing below already has one.
	stack []uintptr
}

// A Field is one thing wrong with one part of the request.
//
// It is constructed positionally on purpose, since a validation error produces
// thousands of these and errs.Field{Name: ..., Code: ..., Msg: ...} three times
// per form is a lot of typing for no information.
type Field struct {
	// Name is the field it is about, in the spelling the client sent, so a
	// nested one is items.0.quantity.
	Name string

	// Code is the rule that failed, such as required or min.
	Code string

	// Msg is the message to show next to the field.
	Msg string
}

// New is a failure of this kind, with this code and this message.
//
// The message is for the caller, so write it as the caller reads it. A kind
// that logs at error level captures a stack here.
func New(k Kind, code, msg string) *Error {
	return &Error{Kind: k, Code: code, Msg: msg, stack: capture(k, nil, 0)}
}

// Newf is [New] with a formatted message. It is the general form of the
// shorthands below, for a kind that has none.
func Newf(k Kind, code, format string, a ...any) *Error {
	return &Error{Kind: k, Code: code, Msg: sprintf(format, a), stack: capture(k, nil, 0)}
}

// Wrap puts a kind, a code and a message on an existing error, keeping the
// original as the cause so that [errors.Is] and [errors.As] still reach it.
//
// It does not return nil for a nil cause. A *Error in an error variable is
// never nil, so a version that returned nil here would hand back an error that
// is not nil and has nothing wrong with it, which is the worst bug this package
// could ship. Check the error first, the way you were going to anyway.
func Wrap(err error, k Kind, code, msg string) *Error {
	return &Error{Kind: k, Code: code, Msg: msg, Err: err, stack: capture(k, err, 0)}
}

// Wrapf is [Wrap] with a formatted message.
func Wrapf(err error, k Kind, code, format string, a ...any) *Error {
	return &Error{Kind: k, Code: code, Msg: sprintf(format, a), Err: err, stack: capture(k, err, 0)}
}

// The shorthands are the kinds an application writes over and over. Each takes
// a message and no code, and [Error.WithCode] adds one where a client needs to
// tell two of them apart.

// NotFoundf is a resource that is not here.
func NotFoundf(format string, a ...any) *Error {
	return &Error{Kind: NotFound, Msg: sprintf(format, a)}
}

// Invalidf is a request that is malformed.
func Invalidf(format string, a ...any) *Error {
	return &Error{Kind: Invalid, Msg: sprintf(format, a)}
}

// Forbiddenf is a request from someone who may not do this.
func Forbiddenf(format string, a ...any) *Error {
	return &Error{Kind: Forbidden, Msg: sprintf(format, a)}
}

// Conflictf is a request that lost a race.
func Conflictf(format string, a ...any) *Error {
	return &Error{Kind: Conflict, Msg: sprintf(format, a)}
}

// Internalf is a bug or a broken invariant, and captures a stack.
func Internalf(format string, a ...any) *Error {
	return &Error{Kind: Internal, Msg: sprintf(format, a), stack: capture(Internal, nil, 0)}
}

// sprintf formats only when there is something to format, so that a message
// with no arguments costs no allocation and no scan of the string.
func sprintf(format string, a []any) string {
	if len(a) == 0 {
		return format
	}
	return fmt.Sprintf(format, a...)
}

// Error is the message and the cause, joined by a colon, which is the shape Go
// errors have. The code and the kind are not in it: they go in the log record
// and in the response as their own fields, and repeating them in the text makes
// every line longer without telling a reader anything new.
func (e *Error) Error() string {
	switch {
	case e.Msg != "" && e.Err != nil:
		return e.Msg + ": " + e.Err.Error()
	case e.Msg != "":
		return e.Msg
	case e.Err != nil:
		return e.Err.Error()
	case e.Code != "":
		return e.Code
	}
	return e.Kind.String()
}

// Unwrap is the cause, so the whole chain stays reachable.
func (e *Error) Unwrap() error { return e.Err }

// Is matches a [Kind] target by kind, and a *Error target by code when the
// target has one and by kind when it does not.
//
//	errors.Is(err, errs.NotFound)                       // any missing thing
//	errors.Is(err, errs.New(errs.NotFound, "post.not_found", ""))  // that one
func (e *Error) Is(target error) bool {
	switch t := target.(type) {
	case Kind:
		return e.Kind == t
	case *Error:
		if t.Code != "" {
			return e.Code == t.Code
		}
		return e.Kind == t.Kind
	}
	return false
}

// WithCode is a copy carrying this code, for a shorthand that needs one.
func (e *Error) WithCode(code string) *Error {
	out := e.clone()
	out.Code = code
	return out
}

// WithField is a copy with one more field problem on it. Validation collects
// several, so a form comes back with everything wrong with it at once.
func (e *Error) WithField(name, code, msg string) *Error {
	out := e.clone()
	out.Fields = append(slices.Clip(out.Fields), Field{name, code, msg})
	return out
}

// WithFields is a copy with these field problems on it, which is what a
// validator hands over in one go.
func (e *Error) WithFields(fields ...Field) *Error {
	out := e.clone()
	out.Fields = append(slices.Clip(out.Fields), fields...)
	return out
}

// WithMeta is a copy with one more piece of structured detail.
//
// The map is copied, so a call costs an allocation for every key already there.
// Setting five of them in a row is five copies, which is fine for an error and
// would not be for anything on a hot path.
func (e *Error) WithMeta(key string, value any) *Error {
	out := e.clone()
	out.Meta = maps.Clone(out.Meta)
	if out.Meta == nil {
		out.Meta = make(map[string]any, 1)
	}
	out.Meta[key] = value
	return out
}

// WithRetry is a copy that says how long to wait before trying again.
func (e *Error) WithRetry(d time.Duration) *Error {
	out := e.clone()
	out.Retry = d
	return out
}

// clone is the copy every With method decorates. The stack comes with it,
// since it is where the failure happened and not where it was described.
func (e *Error) clone() *Error {
	out := new(Error)
	*out = *e
	return out
}
