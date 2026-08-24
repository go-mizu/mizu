package xm

// M is a map with the operations on it as methods, so a pipeline reads left to
// right instead of inside out.
//
//	public := xm.Of(row).
//		Omit("password", "apiKey").
//		MapValues(redact)
//
// Its underlying type is map[K]V, so it is a map in every way that matters.
// len, indexing, ranging and deleting all work on it, it marshals as a map, and
// every free function in this package takes one because they all take
// ~map[K]V. There is no method to get the plain map back out, because a
// map[K]V is what a caller expecting one already accepts:
//
//	func serve(headers map[string]string)
//
//	serve(xm.Of(h).MapKeys(lower))   // no conversion, no terminal call
//
// The free functions that return the same map type return the chain when
// handed the chain, so [Merge] and [Filter] and the rest carry on from where a
// method left off, and the other way round.
//
// # What is not here, and why
//
// A method may have type parameters of its own, so the steps that change a type
// are methods. [M.MapValues] introduces a value type, [M.MapKeys] introduces a
// key type, and [M.Map] introduces both at once:
//
//	sizes := xm.Of(groups).
//		Filter(nonEmpty).
//		MapValues(func(k string, v []User) int { return len(v) })
//
// What a method still cannot do is narrow the type parameters it was given.
// [SortedKeys] wants an ordered key and [Invert] wants a comparable value, and
// neither can be asked of a receiver declared with comparable and any. Those
// stay free functions, and they take the chain as it is:
//
//	names := xm.SortedKeys(xm.Of(users).Filter(active))
//
// [Entries] is a method as well as a free function, since a slice of pairs is
// the end of a chain rather than another map.
type M[K comparable, V any] map[K]V

// Of starts a chain from a map.
//
// The map is not copied. Of names a type conversion, so the chain and the map
// it came from are the same map until a method that returns a new one is
// called, which is all of them except [M.Update].
//
// A map with a name of its own loses the name here, since M is a type and not
// two. [Filter] of a headers gives back a headers and the chain gives back an
// M, so a caller who wants the name back writes the conversion:
//
//	h := headers(xm.Of(h).MapKeys(lower))
//
// That is the one thing the free functions do that the chain does not, and it
// is why they are still the ones to reach for in code that names its map types
// everywhere.
func Of[K comparable, V any](m map[K]V) M[K, V] { return M[K, V](m) }

// Keys returns the keys as a slice, in no particular order. See [Keys].
func (m M[K, V]) Keys() []K { return Keys(m) }

// Values returns the values as a slice, in no particular order. See [Values].
func (m M[K, V]) Values() []V { return Values(m) }

// Entries returns the key and value pairs as a slice, in no particular order.
// It ends the chain, since a slice of pairs is not a map. See [Entries].
func (m M[K, V]) Entries() []Entry[K, V] { return Entries(m) }

// Filter returns the pairs that keep returns true for, in a new map. See
// [Filter].
func (m M[K, V]) Filter(keep func(K, V) bool) M[K, V] { return Filter(m, keep) }

// Reject returns the pairs that drop returns false for. See [Reject].
func (m M[K, V]) Reject(drop func(K, V) bool) M[K, V] { return Reject(m, drop) }

// Pick returns the pairs under the given keys, leaving out the keys that are
// not there. See [Pick].
func (m M[K, V]) Pick(keys ...K) M[K, V] { return Pick(m, keys...) }

// Omit returns everything except the pairs under the given keys. See [Omit].
//
//	safe := xm.Of(row).Omit("password", "apiKey")
func (m M[K, V]) Omit(keys ...K) M[K, V] { return Omit(m, keys...) }

// Merge returns this map and the others as one, with the later ones winning
// where they share a key. See [Merge].
//
//	settings := xm.Of(defaults).Merge(fromFile, fromEnv)
//
// The receiver is the first map, so it is the one that gets copied whole and
// the one with the least say. That is the order a configuration wants: defaults
// first, and the last argument has the last word.
//
// This is the one method here that costs something over the free function it
// calls, and it is one small allocation: [Merge] takes the maps as one variadic
// list and the receiver has to go in front of the others to be in it. Sixteen
// bytes against the fifty odd kilobytes of copying a map of a thousand keys is
// not worth a second implementation.
func (m M[K, V]) Merge(others ...M[K, V]) M[K, V] {
	return Merge(append([]M[K, V]{m}, others...)...)
}

// MergeWith is [M.Merge] with resolve deciding what happens on a shared key.
// See [MergeWith].
//
//	totals := xm.Of(january).MergeWith(add, february, march)
//
// resolve is called with the key, the value already there, and the value
// arriving on top of it, in that order.
func (m M[K, V]) MergeWith(resolve func(k K, kept, arriving V) V, others ...M[K, V]) M[K, V] {
	return MergeWith(resolve, append([]M[K, V]{m}, others...)...)
}

// GetOr returns the value under k, or fallback if there is none. See [GetOr].
func (m M[K, V]) GetOr(k K, fallback V) V { return GetOr(m, k, fallback) }

// Update replaces the value under k with what fn returns, writing into the map
// rather than returning a new one. See [Update].
//
// It returns nothing, so it does not chain. That is on purpose: everything else
// here leaves its input alone and hands back a new map, and a step that quietly
// changed the map underneath the chain would not look any different from the
// ones that do not.
func (m M[K, V]) Update(k K, fn func(V) V) { Update(m, k, fn) }

// Map turns every pair into another pair, which can change both types at once,
// and carries on as a chain of the new map type. See [Map].
//
//	byID := xm.Of(byName).Map(func(name string, u User) (int, string) {
//		return u.ID, name
//	})
//
// K2 and V2 are inferred from fn, so they are never written out. Two pairs that
// come out with the same key leave the last one, and which one that is depends
// on the order the map was ranged in, which is not fixed.
//
// [M.MapKeys] and [M.MapValues] are the ones for changing one side, and they
// are the ones to reach for unless both sides really are changing.
func (m M[K, V]) Map[K2 comparable, V2 any](fn func(K, V) (K2, V2)) M[K2, V2] {
	return Map(m, fn)
}

// MapKeys rewrites the keys and leaves the values alone. See [MapKeys].
//
//	lower := xm.Of(headers).MapKeys(func(k, v string) string {
//		return strings.ToLower(k)
//	})
//
// Two keys that rewrite to the same key leave one value, and which one depends
// on the map order. The example above is exactly that risk, since two headers
// spelled differently are one header after this.
func (m M[K, V]) MapKeys[K2 comparable](fn func(K, V) K2) M[K2, V] {
	return MapKeys(m, fn)
}

// MapValues rewrites the values and leaves the keys alone, so nothing can
// collide and the result is the same size. See [MapValues].
//
//	counts := xm.Of(groups).MapValues(func(k string, us []User) int {
//		return len(us)
//	})
func (m M[K, V]) MapValues[V2 any](fn func(K, V) V2) M[K, V2] {
	return MapValues(m, fn)
}
