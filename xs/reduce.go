package xs

import (
	"cmp"
	"iter"
)

// Number is what [Sum] and [Product] add up and multiply. Complex numbers are
// not in it, because nothing in this toolkit has ever wanted a sequence of
// them and the constraint is easier to widen later than to narrow.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Reduce combines the elements with fn, starting from the first one.
//
//	longest, ok := xs.Reduce(names, func(a, b string) string {
//		if len(b) > len(a) {
//			return b
//		}
//		return a
//	})
//
// The second result reports whether there was anything to reduce. An empty
// sequence has no first element to start from, so there is no answer to give
// and the zero value is not one.
//
// Use [Fold] when the running total is a different type from the elements,
// which is most of the time.
func Reduce[T any](in iter.Seq[T], fn func(a, b T) T) (T, bool) {
	var acc T
	started := false
	for v := range in {
		if !started {
			acc, started = v, true
			continue
		}
		acc = fn(acc, v)
	}
	return acc, started
}

// Fold combines the elements with fn, starting from init.
//
//	total := xs.Fold(items, 0, func(sum int, it Item) int {
//		return sum + it.Price*it.Quantity
//	})
//
// The running total is its own type, which is what makes this the general one:
// a sequence of orders folds into a report, a sequence of events folds into a
// state. An empty sequence gives back init, so there is no second result to
// check.
func Fold[T, A any](in iter.Seq[T], init A, fn func(A, T) A) A {
	acc := init
	for v := range in {
		acc = fn(acc, v)
	}
	return acc
}

// Sum adds the elements up. An empty sequence sums to zero.
//
// Floating-point addition is not associative, so a Sum of float64 depends on
// the order the elements arrive in and is not the answer you would get from a
// pairwise or compensated sum. That is true of every loop that adds floats and
// is worth knowing rather than worth working around here.
func Sum[T Number](in iter.Seq[T]) T {
	var total T
	for v := range in {
		total += v
	}
	return total
}

// Product multiplies the elements together. An empty sequence gives 1, which is
// the answer that keeps Product over two halves equal to Product over the
// whole.
func Product[T Number](in iter.Seq[T]) T {
	total := T(1)
	for v := range in {
		total *= v
	}
	return total
}

// Min returns the smallest element.
//
// The second result reports whether there was one. [slices.Min] panics on an
// empty slice instead, which it can do because the caller could have checked
// the length first. A sequence has no length to check, so this reports it.
//
// NaN propagates, the same as [slices.Min] and the min builtin: one NaN in the
// sequence makes the answer NaN.
func Min[T cmp.Ordered](in iter.Seq[T]) (T, bool) {
	return Reduce(in, func(a, b T) T { return min(a, b) })
}

// Max returns the largest element, and reports whether there was one. NaN
// propagates, the same as in [Min].
func Max[T cmp.Ordered](in iter.Seq[T]) (T, bool) {
	return Reduce(in, func(a, b T) T { return max(a, b) })
}

// MinBy returns the element with the smallest key.
//
//	cheapest, ok := xs.MinBy(rooms, Room.price)
//
// The first element with a given key wins, so a tie leaves the earlier one.
// [slices.MinFunc] is the version that takes a comparison rather than a key,
// for an order that is not the natural one.
//
// A NaN key never compares smaller than anything, so it cannot win unless it
// arrives first. Take NaN out before you get here if the difference matters.
func MinBy[T any, K cmp.Ordered](in iter.Seq[T], key func(T) K) (T, bool) {
	return bestBy(in, key, true)
}

// MaxBy returns the element with the largest key, and reports whether there was
// one. The first element with a given key wins, the same as in [MinBy].
func MaxBy[T any, K cmp.Ordered](in iter.Seq[T], key func(T) K) (T, bool) {
	return bestBy(in, key, false)
}

// bestBy is [MinBy] and [MaxBy], which differ only in which way the comparison
// goes. key runs once per element rather than twice per comparison, since it is
// the part the caller wrote and the part that might be expensive.
func bestBy[T any, K cmp.Ordered](in iter.Seq[T], key func(T) K, smallest bool) (T, bool) {
	var best T
	var bestKey K
	started := false

	for v := range in {
		k := key(v)
		if !started {
			best, bestKey, started = v, k, true
			continue
		}
		better := k < bestKey
		if !smallest {
			better = k > bestKey
		}
		if better {
			best, bestKey = v, k
		}
	}
	return best, started
}

// Count returns how many elements there are, which means reading all of them.
//
//	n := xs.Count(xs.Filter(users, User.active))
//
// There is no cheaper answer available. A sequence knows its length only by
// producing it, so counting a sequence backed by a query runs the query.
func Count[T any](in iter.Seq[T]) int {
	n := 0
	for range in {
		n++
	}
	return n
}

// CountBy groups the elements by key and counts each group.
//
//	perStatus := xs.CountBy(orders, Order.status)
//
// This is the histogram, not the total. [Count] is the total.
func CountBy[T any, K comparable](in iter.Seq[T], key func(T) K) map[K]int {
	out := make(map[K]int)
	for v := range in {
		out[key(v)]++
	}
	return out
}
