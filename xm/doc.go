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
// # Reading left to right
//
// Free functions nest, and three of them read inside out, which is the wrong
// way round from the order the work happens in. [Of] starts a chain over the
// same map, with the operations as methods:
//
//	public := xm.Of(row).
//		Omit("password", "apiKey").
//		MapValues(redact)
//
// [M] has map[K]V as its underlying type, so it is a map in every way that
// matters. len, indexing, ranging and deleting all work on it, it marshals as a
// map, and a caller expecting a map[K]V takes one without a conversion. That is
// why there is no call to end a chain: there is nothing to convert back.
//
// It is the same cost as the free functions, the same allocations, and the same
// maps coming out, with one exception that [M.Merge] explains. Every method
// here is the free function with the arguments the other way round.
//
// A method may have type parameters of its own, so a step that changes a type
// stays in the chain. [M.MapValues] introduces a value type, [M.MapKeys]
// introduces a key type, and [M.Map] introduces both at once. The types are
// inferred from the function, so no call site writes one out.
//
// Two things stay free functions, and the reason is a language rule rather than
// a decision: a method cannot narrow the type parameters its receiver was
// declared with. [SortedKeys] wants an ordered key and [Invert] wants a
// comparable value, and neither can be asked of a receiver declared with
// comparable and any. Both take the chain as it stands:
//
//	names := xm.SortedKeys(xm.Of(users).Filter(active))
//
// The chain gives up one thing the free functions keep. They are written
// against ~map[K]V, so [Filter] of a named map type gives the named type back,
// and the chain gives back an [M]. Code that names its map types everywhere
// either writes the conversion or stays with the free functions.
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
