package xs

import (
	"cmp"
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
// A method may have type parameters of its own, so the steps that introduce a
// new type are methods: [Seq.Map] introduces a result type, [Seq.GroupBy] and
// the rest of the By methods introduce a key, and the chain carries on across
// the change.
//
//	names := xs.Of(users).
//		Filter(User.active).
//		Map(User.name).
//		Take(10).
//		Slice()
//
// What a method still cannot do is narrow the type parameter it was given. Sum
// wants a number, Min and Max want an ordered type, Unique wants a comparable
// one and Join wants a string, and none of those can be said about a receiver
// declared with any. Those stay free functions, and [Seq.Seq] hands the
// sequence over to them.
//
//	total := xs.Sum(xs.Of(items).Map(Item.total).Seq())
//
// [Seq.Chunk] and [Seq.Window] are the other hole. Both return a sequence of
// slices of the element type, and a method returning Seq[[]T] on a Seq[T]
// receiver is an instantiation cycle, so both hand back a plain [iter.Seq] and
// [From] starts a new chain if the batches want one.
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
// This and [Seq.SortBy] are the two methods that are not lazy, and they cannot
// be: the smallest element could be the last one to arrive. The sort happens
// when the chain is read rather than when it is built, and it happens again on
// every read. A [Take] after it still saves the work further down the chain,
// and none of the work above it.
//
// [Seq.SortBy] is this with the comparison written for you, and is the one to
// reach for when the order is a field. This one is for the orders that are not:
// two fields, a reversal, or anything [cmp.Compare] does not say.
func (s Seq[T]) SortFunc(compare func(a, b T) int) Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range slices.SortedFunc(s.Seq(), compare) {
			if !yield(v) {
				return
			}
		}
	}
}

// SortBy sorts the elements by what key returns for each of them.
//
//	youngest := xs.Of(users).SortBy(User.age).Take(3).Slice()
//
// key is called once per comparison rather than once per element, so a key that
// costs something wants [Seq.SortFunc] over a slice of pairs instead. For a
// field read, which is what this is for, the call is free and the sort is the
// cost.
//
// It is not lazy, for the reason [Seq.SortFunc] gives.
func (s Seq[T]) SortBy[K cmp.Ordered](key func(T) K) Seq[T] {
	return s.SortFunc(func(a, b T) int { return cmp.Compare(key(a), key(b)) })
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

// Map turns every element into something else and carries on as a chain of the
// new type. See [Map].
//
//	names := xs.Of(users).Map(User.name).Take(10).Slice()
//
// R is inferred from fn, so it is never written out. Writing it out is how you
// take a method value: xs.Of(users).Map[string] is a func(func(User) string)
// Seq[string], which is worth knowing and almost never worth using.
func (s Seq[T]) Map[R any](fn func(T) R) Seq[R] {
	return Seq[R](Map(s.Seq(), fn))
}

// FlatMap turns every element into a sequence and runs them one after another,
// so an element can become several or none. See [FlatMap].
//
//	tags := xs.Of(posts).FlatMap(Post.tags).Slice()
func (s Seq[T]) FlatMap[R any](fn func(T) iter.Seq[R]) Seq[R] {
	return Seq[R](FlatMap(s.Seq(), fn))
}

// Zip pairs the elements with the elements of other and ends with whichever
// runs out first. See [Zip].
//
// It ends the chain, since a sequence of pairs is an [iter.Seq2] and this type
// holds one element.
func (s Seq[T]) Zip[U any](other iter.Seq[U]) iter.Seq2[T, U] {
	return Zip(s.Seq(), other)
}

// Fold combines the elements into a single value of another type, starting from
// init. See [Fold].
//
//	widest := xs.Of(words).Fold(0, func(n int, s string) int { return max(n, len(s)) })
//
// [Seq.Reduce] is the one for combining the elements with each other, where the
// answer has the element's own type and there is nothing to start from.
func (s Seq[T]) Fold[A any](init A, fn func(A, T) A) A {
	return Fold(s.Seq(), init, fn)
}

// UniqueBy leaves out the elements whose key has been seen already, keeping the
// first of each. See [UniqueBy].
func (s Seq[T]) UniqueBy[K comparable](key func(T) K) Seq[T] {
	return Seq[T](UniqueBy(s.Seq(), key))
}

// GroupBy collects the elements into a map from key to every element with that
// key, in the order they arrived. See [GroupBy].
//
//	byAuthor := xs.Of(posts).GroupBy(Post.author)
func (s Seq[T]) GroupBy[K comparable](key func(T) K) map[K][]T {
	return GroupBy(s.Seq(), key)
}

// KeyBy collects the elements into a map from key to element, keeping the last
// of each. See [KeyBy].
func (s Seq[T]) KeyBy[K comparable](key func(T) K) map[K]T {
	return KeyBy(s.Seq(), key)
}

// CountBy counts how many elements share each key. See [CountBy].
func (s Seq[T]) CountBy[K comparable](key func(T) K) map[K]int {
	return CountBy(s.Seq(), key)
}

// MinBy returns the element with the smallest key, and reports whether there
// was one. See [MinBy].
func (s Seq[T]) MinBy[K cmp.Ordered](key func(T) K) (T, bool) {
	return MinBy(s.Seq(), key)
}

// MaxBy returns the element with the largest key, and reports whether there was
// one. See [MaxBy].
func (s Seq[T]) MaxBy[K cmp.Ordered](key func(T) K) (T, bool) {
	return MaxBy(s.Seq(), key)
}
