package xs

import "iter"

// Take returns at most the first n elements of in.
//
//	first10 := xs.Take(seq, 10)
//
// It stops reading in once it has them, which is the point: ten elements out of
// a sequence of ten million does ten elements of work. A negative or zero n
// yields nothing and reads nothing.
func Take[T any](in iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		if n <= 0 {
			return
		}
		left := n
		for v := range in {
			if !yield(v) {
				return
			}
			left--
			if left == 0 {
				return
			}
		}
	}
}

// TakeWhile returns the elements of in up to the first one that ok returns
// false for, and stops there.
//
//	recent := xs.TakeWhile(events, func(e Event) bool { return e.At.After(cutoff) })
//
// The element that failed is not yielded and nothing after it is read, so this
// is the one to reach for over a sorted sequence where the rest cannot match
// either. Over an unsorted one it is [Filter] you want.
func TakeWhile[T any](in iter.Seq[T], ok func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range in {
			if !ok(v) || !yield(v) {
				return
			}
		}
	}
}

// Drop returns in without its first n elements.
//
//	rest := xs.Drop(seq, 10)
//
// The skipped elements are still read, because a sequence has no way to be
// asked for its eleventh without producing the ten in front of it. A negative
// or zero n drops nothing.
func Drop[T any](in iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		left := n
		for v := range in {
			if left > 0 {
				left--
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}

// DropWhile returns in from the first element that skip returns false for
// onwards.
//
//	body := xs.DropWhile(lines, func(s string) bool { return strings.HasPrefix(s, "#") })
//
// skip stops being called once it has returned false once, so an element in the
// middle that looks like one of the leading ones is kept. That is the
// difference between this and [Reject], and it is the reason both exist.
func DropWhile[T any](in iter.Seq[T], skip func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		dropping := true
		for v := range in {
			if dropping {
				if skip(v) {
					continue
				}
				dropping = false
			}
			if !yield(v) {
				return
			}
		}
	}
}
