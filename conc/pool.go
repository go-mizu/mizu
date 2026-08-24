package conc

import "sync"

// A TypedPool is a [sync.Pool] with the type kept. Use [Pool] to make one.
//
// The zero value is not usable, because a pool with nothing to build has
// nothing to hand out.
type TypedPool[T any] struct {
	p sync.Pool
}

// Pool returns a pool of values that fn builds.
//
//	buffers := conc.Pool(func() *bytes.Buffer { return new(bytes.Buffer) })
//
//	b := buffers.Get()
//	defer func() { b.Reset(); buffers.Put(b) }()
//
// This is [sync.Pool] without the type assertion on every Get and without the
// nil check that goes with it. The saving is one line at each call site and one
// class of bug, since a pool holding two kinds of value compiles as a
// sync.Pool and does not compile as this.
//
// Give it a pointer type. [TypedPool.Put] takes an interface underneath, and a
// pointer goes in without allocating while a struct or a slice does not, which
// would mean allocating in order to avoid allocating.
//
// A pool is for reuse under churn and is not a cache. Anything in it can
// disappear at the next collection, so nothing that matters should be the only
// copy of itself. Under the race detector a Put is dropped on purpose now and
// then, which is worth knowing before writing a test that expects the value
// back. Values come back in whatever state they were put back in, so resetting
// is the caller's job and belongs next to the Put that follows it.
func Pool[T any](fn func() T) *TypedPool[T] {
	if fn == nil {
		panic("conc: Pool without a function to build with")
	}
	p := &TypedPool[T]{}
	p.p.New = func() any { return fn() }
	return p
}

// Get returns a value from the pool, building one if there is none to reuse.
func (p *TypedPool[T]) Get() T { return p.p.Get().(T) }

// Put returns a value to the pool. What happens to it after that is the pool's
// business, so the caller keeps no reference to it.
func (p *TypedPool[T]) Put(v T) { p.p.Put(v) }
