package xs

import "iter"

// Enumerate pairs every element with its position, counting from zero.
//
//	for i, line := range xs.Enumerate(lines) {
//		fmt.Printf("%4d  %s\n", i+1, line)
//	}
//
// [slices.All] does this for a slice. This is the one for a sequence, where
// there is no index to read and the count has to be kept.
func Enumerate[T any](in iter.Seq[T]) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		i := 0
		for v := range in {
			if !yield(i, v) {
				return
			}
			i++
		}
	}
}

// Zip pairs the elements of a and b, and ends when the shorter one does.
//
//	for name, score := range xs.Zip(names, scores) { ... }
//
// The leftovers of the longer sequence are not read. If one of them is
// infinite, [Cycle] for instance, then the other one decides where this ends,
// which is the usual reason to reach for it.
func Zip[A, B any](a iter.Seq[A], b iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		// One side has to be pulled, because two sequences cannot both be in a
		// range loop and take turns. Pull runs b on a coroutine, so stop has to
		// run whatever happens or that coroutine is left parked forever.
		next, stop := iter.Pull(b)
		defer stop()

		for va := range a {
			vb, ok := next()
			if !ok || !yield(va, vb) {
				return
			}
		}
	}
}

// Unzip splits a sequence of pairs into two slices.
//
//	names, scores := xs.Unzip(pairs)
//
// This is the one function here that collects, and it has to. Two sequences
// read at different speeds cannot come from one, since handing out the tenth
// name means having read the first ten scores and having somewhere to keep
// them. Reading the whole thing once and giving back two slices is the honest
// version of that, so the memory is visible in the signature.
func Unzip[A, B any](in iter.Seq2[A, B]) ([]A, []B) {
	var as []A
	var bs []B
	for a, b := range in {
		as = append(as, a)
		bs = append(bs, b)
	}
	return as, bs
}
