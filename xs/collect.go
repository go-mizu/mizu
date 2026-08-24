package xs

import "iter"

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
