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

// Interleave takes one element from each sequence in turn.
//
//	mixed := xs.Interleave(fromEurope, fromAsia, fromAmericas)
//
// A sequence that runs out drops out and the rest carry on without it, so this
// ends when the last one does rather than when the first one does. That is the
// difference from [Zip], which stops at the shorter one because it has to
// produce pairs.
//
// It is for spreading a few sources out evenly: results from three regions, or
// a page that should not show six posts from the same author in a row. Every
// sequence past the first is read through [iter.Pull], which means a goroutine
// each, so this is for a handful of sequences and not for thousands.
func Interleave[T any](seqs ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		nexts := make([]func() (T, bool), 0, len(seqs))
		for _, seq := range seqs {
			next, stop := iter.Pull(seq)
			defer stop()
			nexts = append(nexts, next)
		}

		for len(nexts) > 0 {
			// live is written over the front of nexts as the round goes on. The
			// write index never gets ahead of the read index, so an entry is
			// always read before anything is written over it.
			live := nexts[:0]
			for _, next := range nexts {
				v, ok := next()
				if !ok {
					continue
				}
				if !yield(v) {
					return
				}
				live = append(live, next)
			}
			nexts = live
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
