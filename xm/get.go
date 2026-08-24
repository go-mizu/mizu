package xm

// GetOr returns the value under k, or fallback if there is none.
//
//	port := xm.GetOr(settings, "port", 8080)
//
// This is not the same as reading the map and taking the zero value, since a
// key that is present with the zero value under it keeps that zero value. A
// port explicitly set to 0 stays 0 here and would become 8080 the other way.
func GetOr[M ~map[K]V, K comparable, V any](m M, k K, fallback V) V {
	if v, found := m[k]; found {
		return v
	}
	return fallback
}

// Update replaces the value under k with what fn returns, and writes into m
// rather than returning a new map.
//
//	xm.Update(counts, word, func(n int) int { return n + 1 })
//
// fn is called with the value that is there, or with the zero value if the key
// is not in the map, which is what makes the counter above work without a
// lookup first. The key ends up in the map either way, so this puts a key there
// that was not there before.
//
// This is the one function here that changes the map it was given, which is why
// it is called Update and not something quieter.
func Update[M ~map[K]V, K comparable, V any](m M, k K, fn func(V) V) {
	m[k] = fn(m[k])
}
