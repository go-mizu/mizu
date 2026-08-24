package xs

import "math/rand/v2"

// The rest of this package works on sequences. These seven work on slices,
// because a slice is what you have often enough that turning it into a sequence
// and back to do one of these would be the long way round.

// Shuffle reorders s in place.
//
//	xs.Shuffle(deck)
//
// This is [rand.Shuffle] with the swap written for you, and it uses the same
// source, which is seeded differently in every process and is not suitable for
// anything an attacker cares about the outcome of. Shuffling a ballot or a
// nonce wants crypto/rand instead.
func Shuffle[T any](s []T) {
	rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

// Sample returns n elements of s picked at random, without picking any position
// twice.
//
//	page := xs.Sample(candidates, 3)
//
// It leaves s alone. Asking for more than there is gives all of it in a random
// order, and asking for none or fewer gives nil. The result is in the order it
// was picked, which is a random order rather than the order in s.
//
// The same warning as [Shuffle] applies: this is not for anything where being
// able to guess the answer matters.
func Sample[T any](s []T, n int) []T {
	if n <= 0 || len(s) == 0 {
		return nil
	}
	n = min(n, len(s))

	// A partial Fisher-Yates over a copy. Only the first n positions are settled,
	// so the work is proportional to what was asked for rather than to len(s).
	pool := make([]T, len(s))
	copy(pool, s)
	for i := range n {
		j := i + rand.IntN(len(pool)-i)
		pool[i], pool[j] = pool[j], pool[i]
	}
	return pool[:n:n]
}

// Random returns one element of s at random, and reports whether there was one
// to return. The same warning as [Shuffle] applies.
func Random[T any](s []T) (T, bool) {
	if len(s) == 0 {
		var zero T
		return zero, false
	}
	return s[rand.IntN(len(s))], true
}

// Pad returns s with v appended until it is n long.
//
//	row := xs.Pad(fields, 5, "")
//
// A slice that is already n long or longer comes back as it is, with nothing
// copied and nothing to think about. A shorter one comes back as a new slice
// and the original is left alone, so the padding never writes into whatever s
// was sharing its backing array with.
func Pad[T any](s []T, n int, v T) []T {
	if len(s) >= n {
		return s
	}
	out := make([]T, n)
	copy(out, s)
	for i := len(s); i < n; i++ {
		out[i] = v
	}
	return out
}

// Diff returns the elements of a that are not in b.
//
//	gone := xs.Diff(before, after)
//
// This and [Intersect] and [Union] are set operations, so each element turns up
// at most once in the result even if it turned up twice in the input. The order
// is the order of first appearance, which keeps the answer stable and readable
// rather than whatever a map iteration would give.
func Diff[T comparable](a, b []T) []T {
	skip := setOf(b)
	var out []T
	for _, v := range a {
		if _, found := skip[v]; found {
			continue
		}
		skip[v] = struct{}{} // Also stops a repeat in a from coming out twice.
		out = append(out, v)
	}
	return out
}

// Intersect returns the elements that are in both a and b, at most once each,
// in the order they first appear in a.
func Intersect[T comparable](a, b []T) []T {
	in := setOf(b)
	var out []T
	for _, v := range a {
		if _, found := in[v]; !found {
			continue
		}
		delete(in, v) // Once it has come out, it is not in b any more.
		out = append(out, v)
	}
	return out
}

// Union returns the elements of a and b, at most once each, in the order they
// first appear.
//
//	all := xs.Union(fromCache, fromDB)
func Union[T comparable](a, b []T) []T {
	seen := make(map[T]struct{}, len(a)+len(b))
	var out []T
	for _, s := range [][]T{a, b} {
		for _, v := range s {
			if _, dup := seen[v]; dup {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// setOf is the membership test the three set operations share. The map is
// theirs to write into afterwards, which is how each of them keeps a repeat
// from coming out twice.
func setOf[T comparable](s []T) map[T]struct{} {
	out := make(map[T]struct{}, len(s))
	for _, v := range s {
		out[v] = struct{}{}
	}
	return out
}
