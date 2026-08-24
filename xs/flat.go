package xs

import "iter"

// Flatten reads a sequence of sequences as one sequence.
//
//	all := xs.Flatten(pages)
//
// For a sequence of slices, which is the more common thing to have, hand
// [slices.Values] to [FlatMap]:
//
//	all := xs.FlatMap(batches, slices.Values)
//
// It goes one level deep. A sequence of sequences of sequences comes out as a
// sequence of sequences, and the second call is yours to write, because a
// version that kept going could not say what type it returned.
func Flatten[T any](in iter.Seq[iter.Seq[T]]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for seq := range in {
			for v := range seq {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// FlatMap applies fn to every element and reads the results as one sequence.
//
//	tags := xs.FlatMap(posts, func(p Post) iter.Seq[string] {
//		return slices.Values(p.Tags)
//	})
//
// This is [Map] followed by [Flatten], and it is the one to reach for when one
// element turns into several, or into none. Returning an empty sequence drops
// the element, which makes this a [Filter] and a [Map] at once when that is
// what the work needs.
func FlatMap[T, R any](in iter.Seq[T], fn func(T) iter.Seq[R]) iter.Seq[R] {
	return func(yield func(R) bool) {
		for v := range in {
			for r := range fn(v) {
				if !yield(r) {
					return
				}
			}
		}
	}
}
