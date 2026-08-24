package xm

// Map turns every pair into another pair, which can change both types at once.
//
//	byID := xm.Map(byName, func(name string, u User) (int, string) {
//		return u.ID, name
//	})
//
// Two pairs that come out with the same key leave the last one, and which one
// that is depends on the order the map was ranged in, which is not fixed. A key
// function that can collide is a key function to think again about.
//
// [MapKeys] and [MapValues] are the ones for changing one side, and they are
// the ones to reach for unless both sides really are changing.
func Map[M ~map[K]V, K, K2 comparable, V, V2 any](m M, fn func(K, V) (K2, V2)) map[K2]V2 {
	out := make(map[K2]V2, len(m))
	for k, v := range m {
		k2, v2 := fn(k, v)
		out[k2] = v2
	}
	return out
}

// MapKeys rewrites the keys and leaves the values alone.
//
//	lower := xm.MapKeys(headers, func(k, v string) string {
//		return strings.ToLower(k)
//	})
//
// Two keys that rewrite to the same key leave one value, and which one depends
// on the map order, which is not fixed. The example above is exactly that risk,
// since two headers spelled differently are one header after this.
func MapKeys[M ~map[K]V, K, K2 comparable, V any](m M, fn func(K, V) K2) map[K2]V {
	out := make(map[K2]V, len(m))
	for k, v := range m {
		out[fn(k, v)] = v
	}
	return out
}

// MapValues rewrites the values and leaves the keys alone, so nothing can
// collide and the result is the same size.
//
//	counts := xm.MapValues(groups, func(k string, users []User) int {
//		return len(users)
//	})
func MapValues[M ~map[K]V, K comparable, V, V2 any](m M, fn func(K, V) V2) map[K]V2 {
	out := make(map[K]V2, len(m))
	for k, v := range m {
		out[k] = fn(k, v)
	}
	return out
}

// Filter returns the pairs that keep returns true for, in a new map.
//
//	enabled := xm.Filter(features, func(name string, on bool) bool { return on })
//
// [maps.DeleteFunc] is the one that changes the map it was given. This one
// leaves it alone, which is what you want when the map came from somewhere
// else.
func Filter[M ~map[K]V, K comparable, V any](m M, keep func(K, V) bool) M {
	out := make(M, len(m))
	for k, v := range m {
		if keep(k, v) {
			out[k] = v
		}
	}
	return out
}

// Reject returns the pairs that drop returns false for, which is [Filter] with
// the predicate the other way round. It is here for the call sites that read
// backwards otherwise.
func Reject[M ~map[K]V, K comparable, V any](m M, drop func(K, V) bool) M {
	return Filter(m, func(k K, v V) bool { return !drop(k, v) })
}

// Invert swaps the keys and the values.
//
//	nameByID := xm.Invert(idByName)
//
// A value that turns up under two keys leaves one of them, and which one
// depends on the order the map was ranged in, which is not fixed. Inverting a
// map whose values are not unique gives a different answer on different runs,
// so check that they are unique before reaching for this.
func Invert[M ~map[K]V, K, V comparable](m M) map[V]K {
	out := make(map[V]K, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}
