package xs

import (
	"iter"
	"strings"
)

// CollectErr reads a sequence of values and errors into a slice, and stops at
// the first error.
//
//	rows, err := xs.CollectErr(xs.MapErr(lines, parse))
//	if err != nil {
//		return err
//	}
//
// On an error it returns nil rather than what it had so far. A half-filled
// slice looks like a whole one at every call site that forgets to check the
// error, and the same reasoning is why conc.Map does it this way.
//
// [slices.Collect] is the one for a sequence with no errors in it. There is no
// xs.Collect, because it would be that function under another name.
func CollectErr[T any](in iter.Seq2[T, error]) ([]T, error) {
	var out []T
	for v, err := range in {
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// GroupBy sorts the elements into groups by what key returns.
//
//	byStatus := xs.GroupBy(orders, Order.status)
//	for _, o := range byStatus["pending"] { ... }
//
// Each group is in the order the elements arrived in. A key with no elements is
// missing from the map rather than present and empty, so a read of a key that
// never turned up gives a nil slice, which ranges over nothing.
//
// [CountBy] is this without keeping the elements, for when the sizes are all
// you wanted.
func GroupBy[T any, K comparable](in iter.Seq[T], key func(T) K) map[K][]T {
	out := make(map[K][]T)
	for v := range in {
		k := key(v)
		out[k] = append(out[k], v)
	}
	return out
}

// KeyBy indexes the elements by what key returns, one element per key.
//
//	byID := xs.KeyBy(users, func(u User) int { return u.ID })
//
// The last element with a given key is the one kept, which is what the loop
// this replaces does. [UniqueBy] keeps the first instead, because it is
// filtering a sequence rather than building an index, and the difference is
// worth knowing when the keys are not as unique as you thought. [GroupBy] is
// the one that keeps all of them.
func KeyBy[T any, K comparable](in iter.Seq[T], key func(T) K) map[K]T {
	out := make(map[K]T)
	for v := range in {
		out[key(v)] = v
	}
	return out
}

// PartitionBy splits the elements into the ones ok accepts and the ones it does
// not.
//
//	valid, invalid := xs.PartitionBy(rows, Row.ok)
//
// Both sides keep the order the elements arrived in, and a side with nothing on
// it is nil. This reads the whole sequence, since the last element could land
// on either side.
//
// It takes a predicate rather than a key, the way [GroupBy] takes a key. Two
// groups is the case with names for both sides, and more than two is [GroupBy].
func PartitionBy[T any](in iter.Seq[T], ok func(T) bool) (yes, no []T) {
	for v := range in {
		if ok(v) {
			yes = append(yes, v)
		} else {
			no = append(no, v)
		}
	}
	return yes, no
}

// Join writes the elements one after another with sep in between.
//
//	line := xs.Join(xs.Map(cols, strconv.Itoa), ",")
//
// [strings.Join] is the one for a slice. This is the one for a sequence, and it
// is here rather than left as strings.Join over [slices.Collect] because it
// builds the string without building the slice first.
func Join(in iter.Seq[string], sep string) string {
	var b strings.Builder
	first := true
	for s := range in {
		if !first {
			b.WriteString(sep)
		}
		b.WriteString(s)
		first = false
	}
	return b.String()
}

// EachErr calls fn for every element and stops at the first error, whether the
// error came from the sequence or from fn.
//
//	err := xs.EachErr(rows, func(r Row) error { return save(ctx, r) })
//
// There is no xs.Each next to this. Calling a function for every element of a
// sequence is a range loop, and a range loop says what it does without anybody
// having to look up an argument order. This one is here because the two places
// an error can come from are worth getting right once.
func EachErr[T any](in iter.Seq2[T, error], fn func(T) error) error {
	for v, err := range in {
		if err != nil {
			return err
		}
		if err := fn(v); err != nil {
			return err
		}
	}
	return nil
}
