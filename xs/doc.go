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
// # Changing the shape
//
// Not every step keeps one element as one element. [Chunk] gathers them into
// batches of a fixed size, which is what a bulk insert and a batch API both
// want, and [Window] gathers them into overlapping runs, which is how you look
// at an element and its neighbour:
//
//	for batch := range xs.Chunk(ids, 500) { load(ctx, batch) }
//	for pair := range xs.Window(prices, 2) { change(pair[0], pair[1]) }
//
// [FlatMap] goes the other way, turning one element into several or into none,
// which makes it a [Filter] and a [Map] at once. [Flatten] is the same thing
// for a sequence that already holds sequences. [Unique] and [UniqueBy] leave
// out the repeats, wherever in the sequence they turned up, which is what
// [slices.Compact] does not do.
//
// [Enumerate] pairs each element with its position, and [Zip] pairs two
// sequences together and ends with the shorter one. [Interleave] takes one
// element from each of several sequences in turn and carries on without the
// ones that run out. [Unzip] is the only function here that collects, and its
// doc comment explains why it has no choice.
//
// # Ending a pipeline
//
// A pipeline ends in a range loop, in [slices.Collect], or in one of the
// functions that reads the sequence and returns an answer instead of another
// sequence:
//
//	total := xs.Sum(xs.Map(items, Item.total))
//	admin, found := xs.Find(users, User.isAdmin)
//	if xs.Any(rows, Row.invalid) { ... }
//
// [Reduce] combines the elements with each other and starts from the first one,
// so it reports whether there was anything to start from. [Fold] starts from a
// value you pass and builds something of another type, which is the one to
// reach for most of the time. [Sum], [Product], [Min], [Max], [MinBy], [MaxBy],
// [Count] and [CountBy] are the everyday folds under their own names.
//
// [First], [Find], [Index], [Any], [All] and [None] stop as soon as they know
// the answer, and over a lazy pipeline that means the work behind the elements
// they never reach is never done. [Last] and [Count] have to read everything,
// since there is no way to know which element was the last one until the
// sequence ends.
//
// The ones that can come back empty return a second bool rather than a zero
// value on its own, because a sum of zero and an empty sequence are different
// answers and only the caller knows which one matters. [slices.Min] panics
// instead, which it can afford to do because the caller could have checked the
// length first. A sequence has no length to check.
//
// [GroupBy], [KeyBy] and [PartitionBy] build a collection out of the elements
// rather than a single answer. GroupBy keeps every element under its key, KeyBy
// keeps one, and PartitionBy splits into the two sides of a predicate. [Join]
// writes the elements of a sequence of strings into one string without building
// the slice in between.
//
// [CollectErr] and [EachErr] are the terminals for the [iter.Seq2] shape from
// the errors section above. Both stop at the first error, and EachErr stops on
// an error from either side, the sequence or the function it was given.
//
// # Slices
//
// Seven functions here take a slice instead of a sequence, because a slice is
// what you have often enough that going through a sequence to do one of these
// would be the long way round.
//
//	xs.Shuffle(deck)                  // in place
//	page := xs.Sample(rows, 3)        // three of them, no position twice
//	row := xs.Pad(fields, 5, "")      // padded to five with the zero value
//	gone := xs.Diff(before, after)    // in the first and not the second
//
// [Shuffle], [Sample] and [Random] use [math/rand/v2], which is seeded
// differently in every process and is not suitable for anything an attacker
// cares about the outcome of. A shuffled ballot or a token wants crypto/rand.
//
// [Diff], [Intersect] and [Union] are set operations, so each element turns up
// at most once in the result even when it turned up twice in the input. The
// order is the order of first appearance rather than whatever a map iteration
// would give, which keeps the answer stable enough to test and to print.
//
// # Reading left to right
//
// Free functions nest, and a pipeline of four of them reads inside out, which
// is the wrong way round from the order the elements move in. [Of] and [From]
// start a chain over the same [iter.Seq], with the operations as methods:
//
//	recent := xs.Of(posts).
//		Filter(Post.published).
//		Drop(page * 20).
//		Take(20).
//		Slice()
//
// It is the same laziness and the same cost. A chain allocates nothing that the
// free functions do not, and on a pipeline of two stages over a thousand
// elements the two are the same speed to within the noise of a shared machine.
// A method here is the free function with the arguments the other way round,
// and the compiler is not fooled by that.
//
// A method may have type parameters of its own, so a step that changes the
// element type stays in the chain:
//
//	names := xs.Of(users).
//		Filter(User.active).
//		Map(User.name).
//		SortBy(strings.ToLower).
//		Slice()
//
// The result type is inferred from the function, so it is never written out.
// [Seq.Map], [Seq.FlatMap], [Seq.Fold] and [Seq.Zip] introduce a result type
// this way, and [Seq.GroupBy], [Seq.KeyBy], [Seq.CountBy], [Seq.UniqueBy],
// [Seq.MinBy], [Seq.MaxBy] and [Seq.SortBy] introduce a key.
//
// The chain still cannot cover everything, and the reason is a language rule
// rather than a decision: a method cannot narrow the type parameter its
// receiver was declared with. [Sum] and [Product] want a number, [Min] and
// [Max] want an ordered type, [Unique] wants a comparable one and [Join] wants
// a string, and none of those can be asked of a receiver declared with any.
// Those stay free functions, and [Seq.Seq] hands them the plain sequence:
//
//	total := xs.Sum(xs.Of(items).Map(Item.total).Seq())
//
// [Seq.Chunk] and [Seq.Window] are the other hole, for a different reason. Both
// return a sequence of slices of the element type, and a method returning
// Seq[[]T] on a Seq[T] receiver is an instantiation cycle, so both hand back a
// plain [iter.Seq] and [From] starts a new chain over it.
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
//	slices.MinFunc(s, f)  // smallest under a comparison
//
// There is no xs.Collect for that reason. [Chunk] here is the one for
// sequences, which is a different function that happens to share a name with
// [slices.Chunk], and the same goes for [All], which is a question here and a
// sequence of index and value pairs there.
//
// There is no xs.Each either. Calling a function for every element is a range
// loop, and a range loop says what it does without anybody having to remember
// which argument comes first. [EachErr] is here because the error shape has two
// places an error can come from and getting that right once is worth a
// function. [Tap] is the one for the middle of a pipeline, where there is no
// loop to put the call in.
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
// Four of these cost more than a function call per element, and the doc comment
// on each one says so. [Zip] and [Interleave] read through [iter.Pull], which
// is a coroutine switch per element and about 80 nanoseconds. [Chunk] and
// [Window] hand out a slice the caller owns, which is one allocation per batch
// for Chunk and one per element for Window. [Unique] and [UniqueBy] are a map
// insert per element, about 70 nanoseconds, and hold every distinct element
// they have seen.
//
// The functions that end a pipeline allocate a small fixed amount, around 90
// bytes, whenever the compiler cannot inline the whole thing. A range over a
// function turns the loop body into a closure, and a closure over a running
// total puts that total on the heap when it cannot see where the sequence takes
// it. The amount is the same over ten elements and over a hundred thousand,
// which is what to check if you are wondering whether it matters. The ones that
// build a collection are the exception and grow with what they build:
// [CountBy], [GroupBy] and [KeyBy] build a map, [PartitionBy] builds two slices
// and [Sample] copies the input before it picks.
//
// [Join] does about half the allocation of [strings.Join] over [slices.Collect]
// of the same sequence, a thousand short strings costing about 12 kilobytes
// against about 39, because the slice of strings is never built.
//
// Timings were taken on a machine with other work on it, so read them as
// ceilings. The allocation counts do not move.
package xs
