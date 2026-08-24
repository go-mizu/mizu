package xs_test

import (
	"iter"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

// The numbers that matter are what one stage costs per element, and what a
// pipeline costs against the loop somebody would write instead. Both are here.

func BenchmarkMap(b *testing.B) {
	for _, n := range []int{10, 1000} {
		in := slices.Values(make([]int, n))
		b.Run(strconv.Itoa(n)+" elements", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				for v := range xs.Map(in, double) {
					sink = v
				}
			}
		})
	}
}

func BenchmarkFilter(b *testing.B) {
	for _, n := range []int{10, 1000} {
		in := slices.Values(make([]int, n))
		b.Run(strconv.Itoa(n)+" elements", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				for v := range xs.Filter(in, keep) {
					sink = v
				}
			}
		})
	}
}

// BenchmarkPipeline is four stages over a thousand elements, which is the shape
// this package is for.
func BenchmarkPipeline(b *testing.B) {
	in := slices.Values(make([]int, 1000))

	b.ReportAllocs()
	for b.Loop() {
		seq := xs.Map(xs.Filter(xs.Drop(in, 10), keep), double)
		for v := range xs.Take(seq, 100) {
			sink = v
		}
	}
}

// BenchmarkPipelineByHand is the same four things in one loop, so the
// difference between the two is what the sequences cost.
func BenchmarkPipelineByHand(b *testing.B) {
	in := make([]int, 1000)

	b.ReportAllocs()
	for b.Loop() {
		taken := 0
		for i, v := range in {
			if i < 10 || !keep(v) {
				continue
			}
			sink = double(v)
			taken++
			if taken == 100 {
				break
			}
		}
	}
}

// BenchmarkTakeFromAMillion is the case the per-element cost does not describe.
// Ten elements out of a million costs ten elements of work, and the slice
// version that filters into a new slice first costs a million.
func BenchmarkTakeFromAMillion(b *testing.B) {
	in := slices.Values(make([]int, 1_000_000))

	b.ReportAllocs()
	for b.Loop() {
		for v := range xs.Take(xs.Map(in, double), 10) {
			sink = v
		}
	}
}

// BenchmarkTakeFromAMillionBySlice is what the same thing costs when every step
// builds a slice, which is what a package without sequences would have to do.
func BenchmarkTakeFromAMillionBySlice(b *testing.B) {
	in := make([]int, 1_000_000)

	b.ReportAllocs()
	for b.Loop() {
		out := make([]int, 0, len(in))
		for _, v := range in {
			out = append(out, double(v))
		}
		for _, v := range out[:10] {
			sink = v
		}
	}
}

// BenchmarkStages says what each stage adds, by running the same thousand
// elements through one, two and four of them.
func BenchmarkStages(b *testing.B) {
	in := slices.Values(make([]int, 1000))
	stages := []struct {
		name string
		of   func(iter.Seq[int]) iter.Seq[int]
	}{
		{"1 stage", func(s iter.Seq[int]) iter.Seq[int] {
			return xs.Map(s, double)
		}},
		{"2 stages", func(s iter.Seq[int]) iter.Seq[int] {
			return xs.Map(xs.Map(s, double), double)
		}},
		{"4 stages", func(s iter.Seq[int]) iter.Seq[int] {
			return xs.Map(xs.Map(xs.Map(xs.Map(s, double), double), double), double)
		}},
	}

	for _, st := range stages {
		b.Run(st.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				for v := range st.of(in) {
					sink = v
				}
			}
		})
	}
}

func double(n int) int { return n * 2 }
func keep(n int) bool  { return n%3 != 0 }

var sink int

// The reshaping functions are the ones that cost something per element rather
// than per pipeline, so each of them gets a number.

func BenchmarkEnumerate(b *testing.B) {
	in := slices.Values(make([]int, 1000))

	b.ReportAllocs()
	for b.Loop() {
		for i, v := range xs.Enumerate(in) {
			sink = i + v
		}
	}
}

// BenchmarkZip pulls one side onto a coroutine, so this is the cost of that
// against the range loop the other side gets.
func BenchmarkZip(b *testing.B) {
	a := slices.Values(make([]int, 1000))
	c := slices.Values(make([]int, 1000))

	b.ReportAllocs()
	for b.Loop() {
		for x, y := range xs.Zip(a, c) {
			sink = x + y
		}
	}
}

func BenchmarkFlatMap(b *testing.B) {
	batches := make([][]int, 100)
	for i := range batches {
		batches[i] = make([]int, 10)
	}
	in := slices.Values(batches)

	b.ReportAllocs()
	for b.Loop() {
		for v := range xs.FlatMap(in, slices.Values) {
			sink = v
		}
	}
}

// BenchmarkChunk is one allocation per batch, so the interesting number is what
// it costs per element at a batch size somebody would use.
func BenchmarkChunk(b *testing.B) {
	in := slices.Values(make([]int, 1000))

	for _, n := range []int{10, 500} {
		b.Run("batches of "+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				for batch := range xs.Chunk(in, n) {
					sink = len(batch)
				}
			}
		})
	}
}

// BenchmarkWindow is one allocation per element rather than per batch, which is
// the price of the windows overlapping.
func BenchmarkWindow(b *testing.B) {
	in := slices.Values(make([]int, 1000))

	b.ReportAllocs()
	for b.Loop() {
		for w := range xs.Window(in, 3) {
			sink = len(w)
		}
	}
}

// BenchmarkUnique is a map lookup and an insert per element, and the map is
// what the memory warning in the doc comment is about.
func BenchmarkUnique(b *testing.B) {
	in := make([]int, 1000)
	for i := range in {
		in[i] = i
	}
	seq := slices.Values(in)

	b.ReportAllocs()
	for b.Loop() {
		for v := range xs.Unique(seq) {
			sink = v
		}
	}
}

func BenchmarkInterleave(b *testing.B) {
	a := slices.Values(make([]int, 500))
	c := slices.Values(make([]int, 500))

	b.ReportAllocs()
	for b.Loop() {
		for v := range xs.Interleave(a, c) {
			sink = v
		}
	}
}

// The terminals are all one pass with nothing allocated, apart from the two
// that build a map. What is worth measuring is which of them stop early and
// what the ones that build something cost against the hand-written loop.

func BenchmarkFold(b *testing.B) {
	in := slices.Values(make([]int, 1000))

	b.ReportAllocs()
	for b.Loop() {
		sink = xs.Fold(in, 0, func(acc, v int) int { return acc + v })
	}
}

// BenchmarkSum against the range loop over the same slice is the honest
// comparison, since Sum is that loop with a function call around the sequence.
func BenchmarkSum(b *testing.B) {
	data := make([]int, 1000)
	seq := slices.Values(data)

	b.Run("xs.Sum", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = xs.Sum(seq)
		}
	})
	b.Run("range over the slice", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			total := 0
			for _, v := range data {
				total += v
			}
			sink = total
		}
	})
}

func BenchmarkMinBy(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = len(data) - i
	}
	seq := slices.Values(data)

	b.ReportAllocs()
	for b.Loop() {
		sink, _ = xs.MinBy(seq, func(n int) int { return n })
	}
}

// BenchmarkCountBy builds a map, so it is the expensive one of the terminals
// and the number moves with how many distinct keys there are.
func BenchmarkCountBy(b *testing.B) {
	in := make([]int, 1000)
	for i := range in {
		in[i] = i % 10
	}
	seq := slices.Values(in)

	b.ReportAllocs()
	for b.Loop() {
		sink = len(xs.CountBy(seq, func(n int) int { return n }))
	}
}

// BenchmarkFirst over a big sequence with a Map in front of it is the case the
// stopping early is for. It should not move with the length of the input.
func BenchmarkFirst(b *testing.B) {
	for _, n := range []int{1000, 1_000_000} {
		in := slices.Values(make([]int, n))
		b.Run(strconv.Itoa(n)+" elements", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink, _ = xs.First(xs.Map(in, double))
			}
		})
	}
}

// BenchmarkFind is First with a predicate, measured at both ends of the input
// so the cost of reaching the match is visible.
func BenchmarkFind(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}
	seq := slices.Values(data)

	for _, at := range []int{0, 999} {
		b.Run("match at "+strconv.Itoa(at), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink, _ = xs.Find(seq, func(n int) bool { return n == at })
			}
		})
	}
}

func BenchmarkAll(b *testing.B) {
	in := slices.Values(make([]int, 1000))

	b.ReportAllocs()
	for b.Loop() {
		if xs.All(in, func(n int) bool { return n == 0 }) {
			sink++
		}
	}
}

// BenchmarkCollectErr against slices.Collect over the same data shows what
// carrying the error alongside every element costs.
func BenchmarkCollectErr(b *testing.B) {
	data := make([]int, 1000)
	withErr := func(yield func(int, error) bool) {
		for _, v := range data {
			if !yield(v, nil) {
				return
			}
		}
	}
	plain := slices.Values(data)

	b.Run("xs.CollectErr", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			out, _ := xs.CollectErr(withErr)
			sink = len(out)
		}
	})
	b.Run("slices.Collect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(slices.Collect(plain))
		}
	})
}

// The terminals that build a collection cost what the collection costs, and the
// slice-shaped helpers are here against the loop they replace.

func BenchmarkGroupBy(b *testing.B) {
	in := make([]int, 1000)
	for i := range in {
		in[i] = i % 10
	}
	seq := slices.Values(in)

	b.ReportAllocs()
	for b.Loop() {
		sink = len(xs.GroupBy(seq, func(n int) int { return n }))
	}
}

func BenchmarkKeyBy(b *testing.B) {
	in := make([]int, 1000)
	for i := range in {
		in[i] = i
	}
	seq := slices.Values(in)

	b.ReportAllocs()
	for b.Loop() {
		sink = len(xs.KeyBy(seq, func(n int) int { return n }))
	}
}

func BenchmarkPartitionBy(b *testing.B) {
	in := slices.Values(make([]int, 1000))

	b.ReportAllocs()
	for b.Loop() {
		yes, _ := xs.PartitionBy(in, keep)
		sink = len(yes)
	}
}

// BenchmarkJoin against strings.Join over the collected slice is the comparison
// that says whether skipping the slice was worth a function.
func BenchmarkJoin(b *testing.B) {
	data := make([]string, 1000)
	for i := range data {
		data[i] = strconv.Itoa(i)
	}
	seq := slices.Values(data)

	b.Run("xs.Join", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(xs.Join(seq, ","))
		}
	})
	b.Run("strings.Join over slices.Collect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(strings.Join(slices.Collect(seq), ","))
		}
	})
}

func BenchmarkShuffle(b *testing.B) {
	data := make([]int, 1000)

	b.ReportAllocs()
	for b.Loop() {
		xs.Shuffle(data)
	}
}

// BenchmarkSample at two sizes out of the same slice shows the work is
// proportional to what was asked for, apart from the copy.
func BenchmarkSample(b *testing.B) {
	data := make([]int, 1000)

	for _, n := range []int{10, 1000} {
		b.Run("taking "+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink = len(xs.Sample(data, n))
			}
		})
	}
}

func BenchmarkUnion(b *testing.B) {
	a := make([]int, 500)
	c := make([]int, 500)
	for i := range a {
		a[i], c[i] = i, i+250
	}

	b.ReportAllocs()
	for b.Loop() {
		sink = len(xs.Union(a, c))
	}
}
