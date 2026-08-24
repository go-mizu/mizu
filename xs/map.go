package xs

import "iter"

// Map returns a sequence of fn applied to every element of in.
//
//	names := xs.Map(users, func(u User) string { return u.Name })
//
// fn runs when the result is read and once per element, so a Map that nobody
// ranges over calls fn no times and a Map read twice calls it twice.
//
// A method expression works where the method takes nothing and returns one
// thing, which covers most of what a Map is for:
//
//	names := xs.Map(users, User.Name)
func Map[T, R any](in iter.Seq[T], fn func(T) R) iter.Seq[R] {
	return func(yield func(R) bool) {
		for v := range in {
			if !yield(fn(v)) {
				return
			}
		}
	}
}

// MapErr is [Map] for a sequence that carries an error alongside each element,
// which is the shape a query, a scanner or a streaming decoder produces.
//
//	users := xs.MapErr(rows, func(r Row) (User, error) { return r.User() })
//
// An element that arrived with an error is passed straight through and fn does
// not see it, so a failure stays attached to the element it belongs to rather
// than ending the sequence. An error from fn is yielded with the zero value of
// R and the sequence carries on to the next element. The caller decides whether
// to break, and does so in the loop at the end where the decision belongs.
func MapErr[T, R any](in iter.Seq2[T, error], fn func(T) (R, error)) iter.Seq2[R, error] {
	return func(yield func(R, error) bool) {
		var zero R
		for v, err := range in {
			if err != nil {
				if !yield(zero, err) {
					return
				}
				continue
			}
			if !yield(fn(v)) {
				return
			}
		}
	}
}

// Filter returns the elements of in that keep returns true for.
//
//	active := xs.Filter(users, func(u User) bool { return u.Active })
func Filter[T any](in iter.Seq[T], keep func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range in {
			if keep(v) && !yield(v) {
				return
			}
		}
	}
}

// Reject returns the elements of in that drop returns false for, which is
// [Filter] with the condition the other way round.
//
//	kept := xs.Reject(users, User.Banned)
//
// It exists because the negation is where the reading goes wrong. A filter that
// keeps what is not expired and a reject of what is expired describe the same
// set, and only one of them has a "not" in it.
func Reject[T any](in iter.Seq[T], drop func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range in {
			if !drop(v) && !yield(v) {
				return
			}
		}
	}
}

// Tap calls fn for every element on its way past and yields it unchanged.
//
//	seq = xs.Tap(seq, func(u User) { log.Debug("saw", "user", u.ID) })
//
// It is for looking at a pipeline without taking it apart: counting, logging,
// a breakpoint. fn only runs for elements that are read, so a Tap in front of a
// [Take] of ten sees ten.
func Tap[T any](in iter.Seq[T], fn func(T)) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range in {
			fn(v)
			if !yield(v) {
				return
			}
		}
	}
}
