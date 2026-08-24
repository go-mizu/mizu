// Package xm works on maps, the way [xs] works on sequences.
//
//	names := xm.SortedKeys(handlers)
//	public := xm.Omit(config, "password", "apiKey")
//	upper := xm.MapValues(headers, func(k, v string) string {
//		return strings.ToUpper(v)
//	})
//
// Every function that returns a map returns a new one and leaves the input
// alone, apart from [Update], which says so in its name and its doc comment.
// A returned map is never nil, so it can be written into without checking.
//
// # Order
//
// A map has no order, and ranging over one gives the keys in a different order
// every time on purpose. Anything here that returns a slice of keys or entries
// therefore returns them in no particular order, and the doc comment says so.
//
// [SortedKeys] is the exception and the reason it exists. Generated code, a
// configuration dump and a test fixture all need the same output twice running,
// and writing slices.Sorted(maps.Keys(m)) in forty places is worse than one
// function with a name that says what it is for.
//
// # What the standard library already does
//
// [maps] covers more than people expect, and anything it covers is not here:
//
//	maps.Clone(m)          // a copy
//	maps.Equal(a, b)       // compare
//	maps.Copy(dst, src)    // merge into an existing map
//	maps.DeleteFunc(m, f)  // filter in place
//	maps.All(m)            // key and value as a sequence
//	maps.Collect(seq)      // a sequence of pairs as a map
//
// [Keys] and [Values] here return slices, which is what callers want most of
// the time and is a different thing from what [maps.Keys] and [maps.Values]
// return. [Filter] is [maps.DeleteFunc] without the mutation, and [Merge] is
// [maps.Copy] without one of the two maps having to be the answer.
//
// There is no xm.Group. Grouping a slice into a map is xs.GroupBy over
// [slices.Values], and having it twice under two names is worse than having it
// once.
//
// # Cost
//
// Everything here is a pass over the map or over the keys it was given, and the
// maps that come back are sized in advance where the size is known. Nothing
// holds a reference to the input after it returns.
//
// A map operation is a hash and a lookup, which moves with the key type rather
// than with anything this package does. On a map of a thousand string keys,
// [Keys] takes about 7 microseconds in one allocation, against about 8 in
// eleven for slices.Collect(maps.Keys(m)), because the size is known up front
// here and the growing slice there has to be copied as it goes.
//
// [SortedKeys] adds a sort and is the expensive one, about 72 microseconds on
// that same map, so it is worth keeping the result rather than calling it again
// inside a loop.
//
// [Merge] copies the first map whole and writes the rest into it a key at a
// time, rather than treating all of them the same way. The runtime has a bulk
// path for copying a map, and a configuration merge is nearly always one large
// map of defaults followed by small overrides, so this is about five times
// faster than the obvious loop on that shape. The cost is that the result has
// to grow when the later maps bring keys the first one did not have, which for
// two maps of a thousand keys each with nothing in common is 164 kilobytes
// against the 109 the obvious loop takes.
//
// [Pick] walks the keys it was given rather than the map, so it costs the
// length of the key list and not the size of the map. [Omit] has to walk the
// map, since it is keeping everything else.
//
// Timings came from an Apple M4 with other work running on it, so read them as
// ceilings. The allocation counts do not move.
package xm
