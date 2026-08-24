package ctxdata

import (
	"context"
	"iter"
	"slices"
)

// A Key names one piece of request data and says what type it holds.
//
// Declare one per datum, at package level, with [NewKey]. Two keys with the
// same name are two different keys, since a key is identified by the variable
// and not by its name, so a package cannot reach another package's data by
// guessing what it called it.
//
// The zero Key was not made by NewKey and every function here panics on one.
type Key[T any] struct{ datum *datum }

// datum is the identity of a key and everything decided about it at
// declaration. It is behind a pointer so that a Key stays comparable and small
// enough to copy, and so that identity is the address rather than the contents.
type datum struct {
	name       string
	logged     bool
	propagated bool
	redacted   bool
}

// A KeyOption says what should happen to a datum beyond being readable, and is
// passed to [NewKey].
type KeyOption func(*datum)

// Logged puts the datum in every log record made from a context that has it.
func Logged() KeyOption { return func(d *datum) { d.logged = true } }

// Propagated carries the datum out of the process: into a queued job, into an
// outbound request to a host the caller trusts, and onto a span.
//
// Nothing propagates anything yet. The flag is read by the packages that carry
// work elsewhere, and those arrive in later milestones.
func Propagated() KeyOption { return func(d *datum) { d.propagated = true } }

// Redacted logs the datum with [Mask] in place of the value, for something that
// belongs in a record by name but not by value.
//
// It implies [Logged], since a value nobody logs has nothing to hide.
func Redacted() KeyOption {
	return func(d *datum) {
		d.redacted = true
		d.logged = true
	}
}

// NewKey declares a key. The name is what a log record and a propagation header
// call it, so write it the way the record should read: tenant_id, not TenantID.
//
//	var TenantID = ctxdata.NewKey[string]("tenant_id", ctxdata.Logged(), ctxdata.Propagated())
//
// It panics on an empty name, which is a mistake that would otherwise show up
// as a log record with an empty key months later.
func NewKey[T any](name string, opts ...KeyOption) Key[T] {
	if name == "" {
		panic("ctxdata: a key needs a name")
	}
	d := &datum{name: name}
	for _, opt := range opts {
		opt(d)
	}
	return Key[T]{datum: d}
}

// Name is what the key is called in a record.
func (k Key[T]) Name() string { return k.id().name }

// String is the name, so a key prints as itself in an error message.
func (k Key[T]) String() string { return k.id().name }

// id is the key's identity, and the one place that catches a Key nobody made.
func (k Key[T]) id() *datum {
	if k.datum == nil {
		panic("ctxdata: this Key did not come from NewKey")
	}
	return k.datum
}

// bagKey is the single context slot every datum lives in, so reading them all
// back costs one [context.Context.Value] lookup no matter how many there are.
type bagKey struct{}

// An entry is one datum with the value it was given.
type entry struct {
	datum *datum
	value any
}

// With is ctx with v stored under k.
//
// The context that comes back has the datum and the one passed in does not, the
// way [context.WithValue] works. Storing the same key twice replaces the value
// rather than shadowing it, so a chain of contexts does not grow a copy per
// call.
func With[T any](ctx context.Context, k Key[T], v T) context.Context {
	d := k.id()
	old := bag(ctx)
	for i, e := range old {
		if e.datum == d {
			next := slices.Clone(old)
			next[i].value = v
			return context.WithValue(ctx, bagKey{}, next)
		}
	}
	next := make([]entry, len(old), len(old)+1)
	copy(next, old)
	next = append(next, entry{datum: d, value: v})
	return context.WithValue(ctx, bagKey{}, next)
}

// Get is the value stored under k, and whether there was one.
func Get[T any](ctx context.Context, k Key[T]) (T, bool) {
	d := k.id()
	for _, e := range bag(ctx) {
		if e.datum == d {
			v, ok := e.value.(T)
			return v, ok
		}
	}
	var zero T
	return zero, false
}

// MustGet is the value stored under k, and panics when there is none.
//
// It is for a value that middleware puts on every request, where a handler
// reaching the missing case means the middleware is not installed and no
// sensible answer exists. Anywhere else, use [Get].
func MustGet[T any](ctx context.Context, k Key[T]) T {
	v, ok := Get(ctx, k)
	if !ok {
		panic("ctxdata: no value for " + k.id().name)
	}
	return v
}

// An Entry is one datum in a context, with what was decided about it when it
// was declared.
//
// It is what a carrier reads: the log handler takes the ones that are Logged,
// and a queue takes the ones that are Propagated.
type Entry struct {
	// Name is what the key is called.
	Name string

	// Value is what was stored, exactly as it was stored. It is not masked
	// here even when Redacted is set, since masking is a decision for whoever
	// writes it down.
	Value any

	// Logged is whether this belongs in every log record from the context.
	Logged bool

	// Propagated is whether this follows the work out of the process.
	Propagated bool

	// Redacted is whether the value should be written as [Mask].
	Redacted bool
}

// All is every datum in ctx, oldest first, which is the order they were stored
// in.
//
// Ranging over it allocates nothing, so a log handler can filter and append
// into a buffer of its own rather than taking the slice [Attrs] builds.
func All(ctx context.Context) iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		for _, e := range bag(ctx) {
			ok := yield(Entry{
				Name:       e.datum.name,
				Value:      e.value,
				Logged:     e.datum.logged,
				Propagated: e.datum.propagated,
				Redacted:   e.datum.redacted,
			})
			if !ok {
				return
			}
		}
	}
}

// bag is the entries in ctx, and nil for a context that has none.
func bag(ctx context.Context) []entry {
	entries, _ := ctx.Value(bagKey{}).([]entry)
	return entries
}
