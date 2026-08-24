// Package xs works on sequences, one element at a time and without collecting
// them.
//
//	active := xs.Filter(slices.Values(users), func(u User) bool {
//		return u.Active
//	})
//	names := slices.Collect(xs.Take(xs.Map(active, User.Name), 10))
//
// Everything here takes an [iter.Seq] and most of it returns one, so the whole
// pipeline is one pass and the only slice is the one at the end, if there is
// one at all. [slices.Values] turns a slice into a sequence and
// [slices.Collect] turns one back, so a pipeline can start and end wherever the
// data already is.
//
// # Nothing runs until something reads
//
// An [iter.Seq] is a function, so building a pipeline runs nothing:
//
//	seq := xs.Map(rows, expensive)   // expensive has not been called.
//	for v := range seq {             // Now it has, once per row.
//		use(v)
//	}
//
// This is what makes [Take] worth having. Ten elements out of a sequence of ten
// million reads ten million times if the sequence is a slice being filtered
// into a new slice at every step, and eleven times here. Breaking out of the
// loop stops the whole pipeline, all the way back to whatever is producing the
// elements, because that is how a range over a function ends.
//
// A sequence is a function and not a container, so reading it twice runs it
// twice. Whether that works at all is the sequence's business: one over a slice
// can be read as often as you like, and one over a network connection or a
// channel cannot be read twice at all. [Repeat] and [Cycle] read theirs more
// than once and say so.
//
// # Errors
//
// A database query, a scanner and a streaming decoder all produce an
// [iter.Seq2] of a value and an error rather than an [iter.Seq]. [MapErr] is
// [Map] over that shape:
//
//	rows := db.Query(ctx, "select ...")     // iter.Seq2[Row, error]
//	users := xs.MapErr(rows, rowToUser)     // iter.Seq2[User, error]
//
// An element that arrived with an error is passed along without fn seeing it,
// so a failure halfway down a pipeline stays attached to the element it belongs
// to instead of ending the sequence. What to do about it is the caller's
// decision, taken in the loop at the end, which is where the caller is.
//
// # What the standard library already does
//
// [slices] and [maps] cover more of this than people expect, and anything they
// cover is not here:
//
//	slices.Values(s)      // a slice as a sequence
//	slices.Collect(seq)   // a sequence as a slice
//	slices.Sorted(seq)    // a sequence as a sorted slice
//	slices.All(s)         // index and value
//	slices.Chunk(s, n)    // a slice in batches
//
// There is no xs.Collect for that reason. [Chunk] here is the one for
// sequences, which is a different function that happens to share a name with
// [slices.Chunk].
//
// # Cost
//
// Nothing here allocates per element. Each function allocates at most once, for
// the closure it returns, and often not even that, since a pipeline that is
// built and ranged over in the same function keeps its closures on the stack.
//
// A stage costs a function call per element that the compiler cannot inline
// through, which is about 7 nanoseconds for a [Map] and about 4 for a [Filter].
// Four stages over a thousand elements runs about six times as long as the one
// hand-written loop that does all four things at once, and that loop is what
// you write when the profile says to.
//
// The saving is not in the per-element cost, and reading the paragraph above on
// its own gets this backwards. It is that [Take] and [Filter] stop early and
// never build the slice in between, so taking ten elements out of a million
// costs about 43 nanoseconds and nothing, against about 3 milliseconds and 8
// megabytes for the version that maps into a new slice and then takes ten of
// it. That is the difference the shape buys, and it grows with the input while
// the per-stage cost does not.
//
// Timings were taken on a machine with other work on it, so read them as
// ceilings. The allocation counts do not move.
package xs
