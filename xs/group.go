package xs

import "iter"

// Chunk returns the elements in batches of n, with a shorter last batch if the
// count does not divide.
//
//	for batch := range xs.Chunk(ids, 500) {
//		if err := db.LoadAll(ctx, batch); err != nil {
//			return err
//		}
//	}
//
// This is what turns a sequence of any length into work of a fixed size, which
// is what a bulk insert, a batch API and a paged request all need.
//
// Every batch is a new slice that the caller owns, so keeping one is safe and
// each batch costs one allocation. [slices.Chunk] is the same idea over a slice
// and it hands back windows into the original, which is cheaper and is
// available to it because a slice is still there to point into. A sequence is
// not, so this one copies.
//
// Chunk panics if n is less than 1, matching [slices.Chunk], because a batch
// size of zero has no meaning that is better than an error.
func Chunk[T any](in iter.Seq[T], n int) iter.Seq[[]T] {
	if n < 1 {
		panic("xs: Chunk with a batch size below 1")
	}
	return func(yield func([]T) bool) {
		batch := make([]T, 0, n)
		for v := range in {
			batch = append(batch, v)
			if len(batch) < n {
				continue
			}
			if !yield(batch) {
				return
			}
			batch = make([]T, 0, n)
		}
		if len(batch) > 0 {
			yield(batch)
		}
	}
}

// Window returns every run of n elements in a row, moving along by one each
// time.
//
//	for pair := range xs.Window(prices, 2) {
//		change = append(change, pair[1]-pair[0])
//	}
//
// A sequence shorter than n yields nothing, since there is no run of n in it.
// The difference from [Chunk] is that these overlap: Chunk of 2 over four
// elements gives two batches and Window of 2 gives three.
//
// Every window is a new slice that the caller owns, so this costs one
// allocation per element rather than per batch. It is for looking at
// neighbours, not for moving bulk.
//
// Window panics if n is less than 1, for the same reason [Chunk] does.
func Window[T any](in iter.Seq[T], n int) iter.Seq[[]T] {
	if n < 1 {
		panic("xs: Window with a size below 1")
	}
	return func(yield func([]T) bool) {
		// held is the last n elements seen, oldest first. It is trimmed from the
		// front rather than kept as a ring, because a window is copied out
		// anyway and n here is small by the nature of the thing.
		held := make([]T, 0, n)
		for v := range in {
			if len(held) == n {
				held = held[:copy(held, held[1:])]
			}
			held = append(held, v)
			if len(held) < n {
				continue
			}
			if !yield(append(make([]T, 0, n), held...)) {
				return
			}
		}
	}
}
