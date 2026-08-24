package xm

import "maps"

// Merge returns the maps as one, with the later ones winning where they share a
// key.
//
//	settings := xm.Merge(defaults, fromFile, fromEnv)
//
// The order is the order of the arguments, so the last one has the last word,
// which is the layering everybody wants out of a configuration. All the inputs
// are left alone. [maps.Copy] is the one that writes into an existing map.
//
// [MergeWith] is the one for when a shared key should combine the two values
// rather than one of them replacing the other.
func Merge[M ~map[K]V, K comparable, V any](ms ...M) M {
	if len(ms) == 0 {
		return make(M)
	}

	// The first map is copied in one go rather than key by key, because the
	// runtime has a bulk path for that. It is worth going out of the way for:
	// the usual call has a large map of defaults first and small overrides
	// after it, and inserting the large one a key at a time costs several
	// times more than copying it whole.
	out := maps.Clone(ms[0])
	if out == nil {
		out = make(M)
	}

	for _, m := range ms[1:] {
		maps.Copy(out, m)
	}
	return out
}

// MergeWith is [Merge] with resolve deciding what happens on a shared key.
//
//	totals := xm.MergeWith(func(k string, a, b int) int { return a + b },
//		januarySales, februarySales)
//
// resolve is called with the key, the value that is already there, and the
// value arriving on top of it, in that order. It runs only where there is a
// clash, so a key in one map and not the others goes straight through.
//
// The function comes first because the maps are variadic and something has to.
func MergeWith[M ~map[K]V, K comparable, V any](resolve func(k K, kept, arriving V) V, ms ...M) M {
	out := make(M)
	for _, m := range ms {
		for k, v := range m {
			if old, clash := out[k]; clash {
				v = resolve(k, old, v)
			}
			out[k] = v
		}
	}
	return out
}

// Pick returns the pairs under the given keys, leaving out the keys that are
// not in the map.
//
//	summary := xm.Pick(row, "id", "name", "email")
//
// The result is at most as long as the list of keys, so this is the one to use
// when the keys are the short list. [Filter] is the one for a rule rather than
// a list.
func Pick[M ~map[K]V, K comparable, V any](m M, keys ...K) M {
	out := make(M, len(keys))
	for _, k := range keys {
		if v, found := m[k]; found {
			out[k] = v
		}
	}
	return out
}

// Omit returns everything except the pairs under the given keys.
//
//	safe := xm.Omit(row, "password", "apiKey")
//
// A key that is not in the map is not an error and not missed. The result is a
// new map, so the original still has the secrets in it and whatever else is
// holding it still sees them.
func Omit[M ~map[K]V, K comparable, V any](m M, keys ...K) M {
	drop := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		drop[k] = struct{}{}
	}

	out := make(M, len(m))
	for k, v := range m {
		if _, skip := drop[k]; skip {
			continue
		}
		out[k] = v
	}
	return out
}
