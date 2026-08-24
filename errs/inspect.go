package errs

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"sync"
	"sync/atomic"
	"time"
)

// KindOf is what kind of failure an error is.
//
// It looks for a *Error first, then for the standard library errors that mean
// something definite, then asks the registered mappers in the order they were
// registered. An error none of those recognise is Internal, since a failure
// nobody classified is not one to show a user. So is nil, which is not a
// failure at all and is not worth asking about.
func KindOf(err error) Kind {
	k, _ := classify(err)
	return k
}

// CodeOf is the machine code of the first error in the chain that has one, or
// an empty string. A code is optional, so an empty answer means nobody set one
// rather than that something went wrong.
func CodeOf(err error) string {
	var code string
	walk(err, func(e error) bool {
		if x, ok := e.(*Error); ok && x.Code != "" {
			code = x.Code
			return true
		}
		return false
	})
	if code != "" {
		return code
	}
	_, code = classify(err)
	return code
}

// Retryable is whether the same call, unchanged, could work later.
func Retryable(err error) bool { return KindOf(err).Retryable() }

// RetryAfter is how long to wait before trying again, from the first error in
// the chain that says. The boolean is whether anything said, since zero is
// also a real answer meaning try now.
func RetryAfter(err error) (time.Duration, bool) {
	var d time.Duration
	found := walk(err, func(e error) bool {
		x, ok := e.(*Error)
		if ok && x.Retry > 0 {
			d = x.Retry
			return true
		}
		return false
	})
	return d, found
}

// Fields are the individual problems with the request, from the first error in
// the chain that has any. They are what a form redisplays and what a client
// puts next to its inputs.
func Fields(err error) []Field {
	var fields []Field
	walk(err, func(e error) bool {
		if x, ok := e.(*Error); ok && len(x.Fields) > 0 {
			fields = x.Fields
			return true
		}
		return false
	})
	return fields
}

// Meta is the structured detail of the first error in the chain that has any.
func Meta(err error) map[string]any {
	var meta map[string]any
	walk(err, func(e error) bool {
		if x, ok := e.(*Error); ok && len(x.Meta) > 0 {
			meta = x.Meta
			return true
		}
		return false
	})
	return meta
}

// A Mapper turns an error this package has never heard of into a kind and a
// code. The boolean is whether it recognised the error at all.
//
// It is how a database driver says that its unique-violation error is Exists
// without this package importing the driver, or knowing that databases exist.
type Mapper func(error) (Kind, string, bool)

// mappers is read on every unrecognised error and written once per driver at
// startup, so the readers take an atomic load and the writers take a lock and
// swap a new slice in. A mapper registered while a request is in flight is
// seen by the next one.
var (
	mappers  atomic.Pointer[[]Mapper]
	mapperMu sync.Mutex
)

// RegisterMapper adds a mapper, which is consulted after the standard library
// errors and after any mapper registered before it.
//
// It is meant to be called from a driver package's init, which is the one
// sanctioned use of init in the toolkit: it is additive, it is idempotent in
// effect, and there is nowhere else to put it that every user of the driver
// would remember to call.
func RegisterMapper(m Mapper) {
	if m == nil {
		panic("errs: RegisterMapper needs a mapper")
	}
	mapperMu.Lock()
	defer mapperMu.Unlock()
	old := mappers.Load()
	next := make([]Mapper, 0, len(deref(old))+1)
	next = append(next, deref(old)...)
	next = append(next, m)
	mappers.Store(&next)
}

// deref is the registered mappers, and nil before anything registers one.
func deref(p *[]Mapper) []Mapper {
	if p == nil {
		return nil
	}
	return *p
}

// classify is the whole of the lookup: this package's own errors, then the
// standard library, then whatever a driver registered.
func classify(err error) (Kind, string) {
	if err == nil {
		return Internal, ""
	}

	if own, ok := find[*Error](err); ok {
		return own.Kind, own.Code
	}
	if k, ok := builtin(err); ok {
		return k, ""
	}
	for _, m := range deref(mappers.Load()) {
		if k, code, ok := m(err); ok {
			return k, code
		}
	}
	return Internal, ""
}

// builtin recognises the standard library errors that mean one definite thing.
//
// The list is short on purpose. database/sql, net/http and encoding/json all
// have errors worth mapping, and mapping them here would put those packages in
// every binary that returns an error. They are registered by the mizu package
// that owns the transport, through [RegisterMapper], which is what the harness
// is for.
func builtin(err error) (Kind, bool) {
	switch {
	case errors.Is(err, context.Canceled):
		return Canceled, true
	case errors.Is(err, context.DeadlineExceeded):
		return Timeout, true
	case errors.Is(err, fs.ErrNotExist):
		return NotFound, true
	case errors.Is(err, fs.ErrPermission):
		return Forbidden, true
	case errors.Is(err, fs.ErrExist):
		return Exists, true
	case errors.Is(err, io.ErrUnexpectedEOF):
		return Unavailable, true
	}

	// net.Error is an interface and net is a large package to import for one
	// method, so the method is asked for directly. os.ErrDeadlineExceeded and
	// every network timeout arrive here.
	if t, ok := find[interface{ Timeout() bool }](err); ok && t.Timeout() {
		return Timeout, true
	}
	return Internal, false
}

// find is the first error in the chain that is a T.
//
// It is [errors.As] without the allocation. errors.As takes its target as an
// any, which puts the target on the heap, and [KindOf] runs on every error the
// toolkit reports. The cost of doing it here is that an error with an As method
// of its own is not consulted, which nothing in the toolkit has.
func find[T any](err error) (T, bool) {
	for err != nil {
		if x, ok := any(err).(T); ok {
			return x, true
		}
		switch u := err.(type) {
		case interface{ Unwrap() error }:
			err = u.Unwrap()
		case interface{ Unwrap() []error }:
			for _, e := range u.Unwrap() {
				if x, ok := find[T](e); ok {
					return x, true
				}
			}
			err = nil
		default:
			err = nil
		}
	}
	var zero T
	return zero, false
}

// walk visits err and everything it wraps, deepest last, and stops at the
// first visit that says it found what it wanted.
//
// It is [find] for the questions that are about more than the type, such as
// the first error carrying a code. Those run once per failure rather than on
// every call, so the closure is worth the shorter code.
//
// Both handle either shape of Unwrap, so an error joined with [errors.Join] is
// searched through rather than stopped at.
func walk(err error, visit func(error) bool) bool {
	for err != nil {
		if visit(err) {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		case interface{ Unwrap() []error }:
			for _, e := range x.Unwrap() {
				if walk(e, visit) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}
