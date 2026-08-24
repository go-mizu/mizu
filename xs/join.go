package xs

import "iter"

// Concat returns the sequences one after another.
//
//	all := xs.Concat(pinned, recent, archived)
//
// Each one is read when the one before it has finished, so a sequence that
// nobody reaches is never read at all. With no arguments the result is empty.
func Concat[T any](seqs ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, seq := range seqs {
			for v := range seq {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Repeat returns in n times over.
//
//	laps := xs.Repeat(slices.Values(course), 3)
//
// This reads in once per pass, so in has to be a sequence that can be read more
// than once: a slice or a map is, a channel or a network connection is not, and
// handing it one of those gives an empty second pass or a panic depending on
// what is underneath. A negative or zero n yields nothing.
//
// It stops if a pass yields nothing, so a Repeat of an empty sequence returns
// rather than making the same empty pass a million times.
func Repeat[T any](in iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		for range n {
			empty := true
			for v := range in {
				empty = false
				if !yield(v) {
					return
				}
			}
			if empty {
				return
			}
		}
	}
}

// Cycle returns in over and over and never ends.
//
//	colours := xs.Take(xs.Cycle(slices.Values(palette)), len(rows))
//
// Something downstream has to stop it. [Take] and [TakeWhile] do, and so does
// breaking out of the loop. Ranging over a Cycle with nothing to end it hangs,
// and that is not a bug this can fix from the inside.
//
// The same reading twice rule as [Repeat] applies, and for the same reason.
// A Cycle of an empty sequence returns rather than spinning.
func Cycle[T any](in iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			empty := true
			for v := range in {
				empty = false
				if !yield(v) {
					return
				}
			}
			if empty {
				return
			}
		}
	}
}
