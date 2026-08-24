package xs

import "iter"

// Unique returns the elements with the repeats left out, keeping the first of
// each.
//
//	tags := xs.Unique(xs.FlatMap(posts, Post.tags))
//
// The order of what is left is the order it arrived in. [slices.Compact] is the
// one that only removes neighbours, which is what you want after a sort and not
// what you want here.
//
// This holds every distinct element it has seen, so the memory grows with the
// number of distinct elements rather than with the length of the sequence.
// Over something unbounded with unbounded variety it grows without stopping,
// which is a reason to put a [Take] in front of it or to not use it at all.
func Unique[T comparable](in iter.Seq[T]) iter.Seq[T] {
	return UniqueBy(in, func(v T) T { return v })
}

// UniqueBy is [Unique] over what key returns rather than over the element, for
// elements that are not comparable or that are the same thing twice under two
// spellings.
//
//	oneEach := xs.UniqueBy(users, func(u User) string { return u.Email })
//
// The first element with a given key is the one that is kept, so the rest of
// its fields are the first version's rather than the last's.
func UniqueBy[T any, K comparable](in iter.Seq[T], key func(T) K) iter.Seq[T] {
	return func(yield func(T) bool) {
		seen := make(map[K]struct{})
		for v := range in {
			k := key(v)
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			if !yield(v) {
				return
			}
		}
	}
}
