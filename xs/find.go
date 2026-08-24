package xs

import "iter"

// First returns the first element, and reports whether there was one.
//
//	newest, ok := xs.First(posts)
//
// Only one element is read. That is the point of it: a sequence that runs a
// query or reads a file does the smallest amount of work it can and then stops.
func First[T any](in iter.Seq[T]) (T, bool) {
	for v := range in {
		return v, true
	}
	var zero T
	return zero, false
}

// Last returns the last element, and reports whether there was one. Every
// element is read, since there is no way to know which one was last until the
// sequence ends.
func Last[T any](in iter.Seq[T]) (T, bool) {
	var last T
	found := false
	for v := range in {
		last, found = v, true
	}
	return last, found
}

// Find returns the first element that ok returns true for.
//
//	admin, found := xs.Find(users, User.isAdmin)
//
// Nothing after the match is read. Over a sequence that is already filtered
// this is the same as [First], and the shorter one to write is the one to
// write.
func Find[T any](in iter.Seq[T], ok func(T) bool) (T, bool) {
	for v := range in {
		if ok(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// Index returns the position of the first element that ok returns true for, or
// -1 if there is none.
//
//	at := xs.Index(lines, func(s string) bool { return strings.HasPrefix(s, "func ") })
//
// This is the [slices.IndexFunc] shape rather than the [slices.Index] shape.
// One predicate covers both, and searching a sequence for a value is a
// comparison per element either way, so there is nothing cheaper to offer.
func Index[T any](in iter.Seq[T], ok func(T) bool) int {
	i := 0
	for v := range in {
		if ok(v) {
			return i
		}
		i++
	}
	return -1
}

// Any reports whether ok returns true for at least one element, and stops at the
// first one that does. Over an empty sequence it is false.
func Any[T any](in iter.Seq[T], ok func(T) bool) bool {
	for v := range in {
		if ok(v) {
			return true
		}
	}
	return false
}

// All reports whether ok returns true for every element, and stops at the first
// one it does not.
//
//	if xs.All(rows, Row.valid) { ... }
//
// Over an empty sequence it is true, which is the answer that keeps All over
// two halves equal to All over the whole. There is nothing in an empty sequence
// to break the rule.
//
// [slices.All] is a different thing under the same name: it turns a slice into
// a sequence of index and value pairs. This one asks a question and returns a
// bool.
func All[T any](in iter.Seq[T], ok func(T) bool) bool {
	for v := range in {
		if !ok(v) {
			return false
		}
	}
	return true
}

// None reports whether ok returns false for every element, and stops at the
// first one it does not. Over an empty sequence it is true.
//
// This is Any with the answer flipped, and it is here because the flipped
// version reads backwards at the call site often enough to be worth a name.
func None[T any](in iter.Seq[T], ok func(T) bool) bool {
	return !Any(in, ok)
}
