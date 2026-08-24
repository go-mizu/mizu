package xs

import (
	"iter"
	"slices"
)

// Seq is a sequence with the operations on it as methods, so a pipeline reads
// left to right instead of inside out.
//
//	names := xs.Of(users).
//		Filter(User.active).
//		Take(10).
//		Slice()
//
// It is the same [iter.Seq] underneath and it is as lazy as the free
// functions, so nothing runs until the last call in the chain reads it.
// [Seq.Seq] hands the plain
// [iter.Seq] back for the free functions and for anything else that takes one.
//
// # What is not here, and why
//
// A method cannot have type parameters of its own and cannot narrow the type
// parameters of its receiver. Everything in this package that introduces a new
// type or a new constraint is therefore missing from this type: Map and FlatMap
// introduce a result type, Unique needs comparable, Sum needs a number, and
// GroupBy, KeyBy, CountBy, MinBy, MaxBy and UniqueBy all introduce a key.
//
// [MapTo] is the way out for the common one. It is a free function that takes a
// Seq and returns a Seq, so the chain carries on after it:
//
//	names := xs.MapTo(xs.Of(users).Filter(User.active), User.name).
//		Take(10).
//		Slice()
//
// For the rest, call [Seq.Seq] and use the free functions. Nothing is lost by
// doing that, since the methods are those functions with a shorter spelling.
type Seq[T any] func(yield func(T) bool)

// Of starts a chain from a slice.
func Of[T any](s []T) Seq[T] {
	return Seq[T](slices.Values(s))
}

// From starts a chain from a sequence, which is what a database query, a
// scanner or another pipeline gives you.
//
// There is one function per input rather than one that takes both, because Go
// has no overloading and a single function taking an interface would give up
// the element type.
func From[T any](seq iter.Seq[T]) Seq[T] {
	return Seq[T](seq)
}

// Seq returns the sequence as a plain [iter.Seq], which is what every free
// function in this package and every range loop takes.
func (s Seq[T]) Seq() iter.Seq[T] { return iter.Seq[T](s) }

// Slice reads the sequence into a slice. It is [slices.Collect] under a name
// that fits on the end of a chain.
func (s Seq[T]) Slice() []T { return slices.Collect(s.Seq()) }

// Filter keeps the elements that keep returns true for. See [Filter].
func (s Seq[T]) Filter(keep func(T) bool) Seq[T] {
	return Seq[T](Filter(s.Seq(), keep))
}

// Reject leaves out the elements that drop returns true for. See [Reject].
func (s Seq[T]) Reject(drop func(T) bool) Seq[T] {
	return Seq[T](Reject(s.Seq(), drop))
}

// Take stops after n elements. See [Take].
func (s Seq[T]) Take(n int) Seq[T] {
	return Seq[T](Take(s.Seq(), n))
}

// TakeWhile stops at the first element ok returns false for. See [TakeWhile].
func (s Seq[T]) TakeWhile(ok func(T) bool) Seq[T] {
	return Seq[T](TakeWhile(s.Seq(), ok))
}

// Drop skips the first n elements. See [Drop].
func (s Seq[T]) Drop(n int) Seq[T] {
	return Seq[T](Drop(s.Seq(), n))
}

// DropWhile skips elements until skip returns false, and keeps the rest. See
// [DropWhile].
func (s Seq[T]) DropWhile(skip func(T) bool) Seq[T] {
	return Seq[T](DropWhile(s.Seq(), skip))
}

// Tap calls fn for every element that is read and passes it along unchanged.
// See [Tap].
func (s Seq[T]) Tap(fn func(T)) Seq[T] {
	return Seq[T](Tap(s.Seq(), fn))
}

// Concat runs this sequence and then each of the others in turn. See [Concat].
func (s Seq[T]) Concat(next ...Seq[T]) Seq[T] {
	all := make([]iter.Seq[T], 0, len(next)+1)
	all = append(all, s.Seq())
	for _, n := range next {
		all = append(all, n.Seq())
	}
	return Seq[T](Concat(all...))
}

// Repeat runs the sequence n times over. See [Repeat].
func (s Seq[T]) Repeat(n int) Seq[T] {
	return Seq[T](Repeat(s.Seq(), n))
}

// Cycle runs the sequence over and over and never ends, so something further
// along has to stop it. See [Cycle].
func (s Seq[T]) Cycle() Seq[T] {
	return Seq[T](Cycle(s.Seq()))
}

// Chunk gathers the elements into batches of n. See [Chunk].
//
// It returns a plain [iter.Seq] of batches rather than another chain, because a
// method that returns Seq of a type built out of its own receiver is an
// instantiation cycle and the compiler says so. [From] starts a new chain from
// the result if the batches want one.
func (s Seq[T]) Chunk(n int) iter.Seq[[]T] {
	return Chunk(s.Seq(), n)
}

// Window gathers the elements into overlapping runs of n, and returns a plain
// [iter.Seq] for the same reason [Seq.Chunk] does. See [Window].
func (s Seq[T]) Window(n int) iter.Seq[[]T] {
	return Window(s.Seq(), n)
}

// Enumerate pairs every element with its position. It ends the chain, since a
// sequence of pairs is an [iter.Seq2] and this type holds one element. See
// [Enumerate].
func (s Seq[T]) Enumerate() iter.Seq2[int, T] {
	return Enumerate(s.Seq())
}

// SortFunc reads the whole sequence, sorts it, and hands the elements back in
// order.
//
//	oldest := xs.Of(users).
//		SortFunc(func(a, b User) int { return cmp.Compare(b.Age, a.Age) }).
//		Take(3).
//		Slice()
//
// This is the one method that is not lazy, and it cannot be: the smallest
// element could be the last one to arrive. The sort happens when the chain is
// read rather than when it is built, and it happens again on every read. A
// [Take] after it still saves the work further down the chain, and none of the
// work above it.
//
// It takes a comparison rather than a key, because a key would be a type
// parameter and a method cannot have one. [slices.SortedFunc] over [Seq.Seq] is
// the same thing spelled out.
func (s Seq[T]) SortFunc(compare func(a, b T) int) Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range slices.SortedFunc(s.Seq(), compare) {
			if !yield(v) {
				return
			}
		}
	}
}

// Count returns how many elements there are. See [Count].
func (s Seq[T]) Count() int { return Count(s.Seq()) }

// First returns the first element, and reports whether there was one. See
// [First].
func (s Seq[T]) First() (T, bool) { return First(s.Seq()) }

// Last returns the last element, and reports whether there was one. See [Last].
func (s Seq[T]) Last() (T, bool) { return Last(s.Seq()) }

// Find returns the first element ok accepts, and reports whether there was one.
// See [Find].
func (s Seq[T]) Find(ok func(T) bool) (T, bool) { return Find(s.Seq(), ok) }

// Index returns the position of the first element ok accepts, or -1. See
// [Index].
func (s Seq[T]) Index(ok func(T) bool) int { return Index(s.Seq(), ok) }

// Any reports whether ok accepts at least one element. See [Any].
func (s Seq[T]) Any(ok func(T) bool) bool { return Any(s.Seq(), ok) }

// All reports whether ok accepts every element. See [All].
func (s Seq[T]) All(ok func(T) bool) bool { return All(s.Seq(), ok) }

// None reports whether ok accepts no element at all. See [None].
func (s Seq[T]) None(ok func(T) bool) bool { return None(s.Seq(), ok) }

// Reduce combines the elements with each other, starting from the first. See
// [Reduce].
func (s Seq[T]) Reduce(fn func(a, b T) T) (T, bool) { return Reduce(s.Seq(), fn) }

// PartitionBy splits the elements into the ones ok accepts and the ones it does
// not. See [PartitionBy].
func (s Seq[T]) PartitionBy(ok func(T) bool) (yes, no []T) { return PartitionBy(s.Seq(), ok) }

// MapTo turns every element into something else and gives back a chain of the
// new type.
//
//	names := xs.MapTo(xs.Of(users), User.name).Take(10).Slice()
//
// It is a function rather than a method on [Seq] because the result type is a
// type parameter and a method cannot have one. That restriction is the reason
// this exists at all, and it is worth knowing before wondering why the chain
// has a hole in the middle of it.
func MapTo[T, R any](s Seq[T], fn func(T) R) Seq[R] {
	return Seq[R](Map(s.Seq(), fn))
}
