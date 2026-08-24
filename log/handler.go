package log

import (
	"errors"
	"log/slog"
	"strings"
	"sync"

	"github.com/go-mizu/mizu/errs"
)

// Mask is what a redacted value is written as.
//
// It is a fixed string rather than a run of asterisks the length of the value,
// because the length of a secret is information about the secret.
const Mask = "[redacted]"

// DefaultRedact is the attribute keys whose values are masked when the options
// say nothing about it.
//
// Redaction by key is the backstop, not the design. A value that should never
// be printed is better as a type with a LogValue method that masks itself,
// since that travels with the value instead of relying on every handler being
// configured. This list catches the ones that arrive as a plain string from
// somewhere else, which in practice is where leaks come from.
func DefaultRedact() []string {
	return []string{
		"password",
		"passwd",
		"secret",
		"token",
		"api_key",
		"apikey",
		"authorization",
		"cookie",
		"set-cookie",
		"session",
		"private_key",
		"credit_card",
	}
}

// redactor decides whether an attribute's value is written or masked.
//
// It is a slice rather than a map because it holds a dozen keys at most and a
// map lookup would need the key folded to lower case first, which allocates on
// a path that runs for every attribute of every record.
type redactor []string

// newRedactor is the list to use, where a nil list means [DefaultRedact] and an
// empty one means nothing is masked.
func newRedactor(keys []string) redactor {
	if keys == nil {
		return DefaultRedact()
	}
	return keys
}

// hides is whether this key's value should be masked, ignoring case, so that
// Authorization and authorization are the same key.
func (r redactor) hides(key string) bool {
	for _, k := range r {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}

// kindOf is the kind of an error and whether anything actually classified it,
// so that a plain error from a package that has never heard of errs does not
// arrive in a record labelled internal.
//
// The chain is walked by hand rather than with [errors.As], which puts its
// target on the heap and would cost an allocation on every error that gets
// logged. The walk follows single unwrapping only. An error that was joined
// with [errors.Join] and holds an errs error somewhere in the tree still gets
// its kind from [errs.KindOf] below, and is only missed when that kind is
// Internal, which is what the record would have said anyway.
func kindOf(err error) (errs.Kind, bool) {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if own, ok := e.(*errs.Error); ok {
			return own.Kind, true
		}
	}
	k := errs.KindOf(err)
	return k, k != errs.Internal
}

// levelTag is the three letters a level prints as. It is a comparison rather
// than a lookup so that a custom level, such as slog.LevelWarn+1, still says
// which of the four it is closest to.
func levelTag(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERR"
	case l >= slog.LevelWarn:
		return "WRN"
	case l >= slog.LevelInfo:
		return "INF"
	default:
		return "DBG"
	}
}

// bufPool holds the byte slices records are formatted into. A record is
// written with one call to the writer, so a line from one goroutine never
// arrives inside a line from another.
var bufPool = sync.Pool{New: func() any {
	b := make([]byte, 0, 1024)
	return &b
}}

func newBuffer() *[]byte { return bufPool.Get().(*[]byte) }

// freeBuffer returns a buffer to the pool, dropping one that grew large enough
// to be worth collecting rather than keeping alive forever.
func freeBuffer(b *[]byte) {
	const maxKept = 16 << 10
	if cap(*b) <= maxKept {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}
