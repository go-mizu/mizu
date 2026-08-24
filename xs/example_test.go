package xs_test

import (
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/xs"
)

type user struct {
	Name   string
	Active bool
}

func (u user) name() string { return u.Name }

func Example() {
	users := []user{
		{"ana", true},
		{"ben", false},
		{"cleo", true},
		{"dai", true},
	}

	active := xs.Filter(slices.Values(users), func(u user) bool { return u.Active })
	names := slices.Collect(xs.Take(xs.Map(active, user.name), 2))

	fmt.Println(names)

	// Output:
	// [ana cleo]
}

func ExampleMap() {
	in := slices.Values([]int{1, 2, 3})
	fmt.Println(slices.Collect(xs.Map(in, strconv.Itoa)))

	// Output:
	// [1 2 3]
}

// ExampleMap_lazy shows that building a pipeline runs none of it. fn is called
// when something reads the result, once per element read.
func ExampleMap_lazy() {
	seq := xs.Map(slices.Values([]int{1, 2, 3}), func(n int) int {
		fmt.Println("doubling", n)
		return n * 2
	})

	fmt.Println("nothing has happened yet")
	fmt.Println(slices.Collect(seq))

	// Output:
	// nothing has happened yet
	// doubling 1
	// doubling 2
	// doubling 3
	// [2 4 6]
}

// ExampleMapErr is the shape a query or a streaming decoder produces. The
// element that arrived with an error keeps it and fn never sees that one.
func ExampleMapErr() {
	rows := func(yield func(string, error) bool) {
		yield("12", nil)
		yield("", fmt.Errorf("the connection dropped"))
		yield("34", nil)
	}

	for n, err := range xs.MapErr(rows, strconv.Atoi) {
		fmt.Println(n, err)
	}

	// Output:
	// 12 <nil>
	// 0 the connection dropped
	// 34 <nil>
}

func ExampleFilter() {
	in := slices.Values([]int{1, 2, 3, 4, 5, 6})
	even := xs.Filter(in, func(n int) bool { return n%2 == 0 })

	fmt.Println(slices.Collect(even))

	// Output:
	// [2 4 6]
}

// ExampleReject is [xs.Filter] with the condition the other way round, for when
// naming what to drop reads better than naming what to keep.
func ExampleReject() {
	in := slices.Values([]string{"ana", "", "cleo", ""})
	named := xs.Reject(in, func(s string) bool { return s == "" })

	fmt.Println(slices.Collect(named))

	// Output:
	// [ana cleo]
}

// ExampleTap looks at a pipeline without taking it apart. It only sees the
// elements that are read, which here is two of five.
func ExampleTap() {
	in := slices.Values([]int{1, 2, 3, 4, 5})
	seen := xs.Tap(in, func(n int) { fmt.Println("saw", n) })

	fmt.Println(slices.Collect(xs.Take(seen, 2)))

	// Output:
	// saw 1
	// saw 2
	// [1 2]
}

func ExampleTake() {
	in := slices.Values([]int{1, 2, 3, 4, 5})
	fmt.Println(slices.Collect(xs.Take(in, 3)))

	// Output:
	// [1 2 3]
}

// ExampleTakeWhile ends at the first element that fails and does not start
// again, which is what you want over something already in order.
func ExampleTakeWhile() {
	in := slices.Values([]int{1, 2, 9, 3})
	fmt.Println(slices.Collect(xs.TakeWhile(in, func(n int) bool { return n < 5 })))

	// Output:
	// [1 2]
}

func ExampleDrop() {
	in := slices.Values([]string{"# a comment", "# another", "the body"})
	fmt.Println(slices.Collect(xs.Drop(in, 2)))

	// Output:
	// [the body]
}

// ExampleDropWhile skips the leading elements that match and keeps everything
// from the first one that does not, including later ones that would have
// matched.
func ExampleDropWhile() {
	lines := slices.Values([]string{"# title", "# author", "body", "# not a header"})
	body := xs.DropWhile(lines, func(s string) bool { return strings.HasPrefix(s, "#") })

	fmt.Println(slices.Collect(body))

	// Output:
	// [body # not a header]
}

func ExampleConcat() {
	pinned := slices.Values([]string{"welcome"})
	recent := slices.Values([]string{"today", "yesterday"})

	fmt.Println(slices.Collect(xs.Concat(pinned, recent)))

	// Output:
	// [welcome today yesterday]
}

// ExampleRepeat reads its input once for each pass, so the input has to be
// something that can be read more than once.
func ExampleRepeat() {
	in := slices.Values([]string{"tick", "tock"})
	fmt.Println(slices.Collect(xs.Repeat(in, 2)))

	// Output:
	// [tick tock tick tock]
}

// ExampleCycle never ends on its own, so something downstream has to stop it.
// Here that is [xs.Take].
func ExampleCycle() {
	palette := slices.Values([]string{"red", "green", "blue"})

	var colours iter.Seq[string] = xs.Cycle(palette)
	fmt.Println(slices.Collect(xs.Take(colours, 5)))

	// Output:
	// [red green blue red green]
}

func ExampleEnumerate() {
	lines := slices.Values([]string{"package main", "import \"fmt\"", "func main() {}"})

	for i, line := range xs.Enumerate(lines) {
		fmt.Printf("%d  %s\n", i+1, line)
	}

	// Output:
	// 1  package main
	// 2  import "fmt"
	// 3  func main() {}
}

// ExampleZip ends with the shorter of the two, so the fourth name is not
// reached and the fourth score is never read.
func ExampleZip() {
	names := slices.Values([]string{"ana", "ben", "cleo"})
	scores := slices.Values([]int{10, 20, 30, 40})

	for name, score := range xs.Zip(names, scores) {
		fmt.Println(name, score)
	}

	// Output:
	// ana 10
	// ben 20
	// cleo 30
}

// ExampleZip_endless is the usual reason to reach for it. One side decides how
// long the result is and the other one supplies whatever it needs.
func ExampleZip_endless() {
	rows := slices.Values([]string{"ana", "ben", "cleo"})
	stripes := xs.Cycle(slices.Values([]string{"light", "dark"}))

	for name, stripe := range xs.Zip(rows, stripes) {
		fmt.Println(name, stripe)
	}

	// Output:
	// ana light
	// ben dark
	// cleo light
}

// ExampleUnzip is the one function here that collects, because two sequences
// read at different speeds cannot come out of one without somewhere to keep the
// difference.
func ExampleUnzip() {
	pairs := func(yield func(string, int) bool) {
		yield("ana", 10)
		yield("ben", 20)
	}

	names, scores := xs.Unzip(pairs)
	fmt.Println(names, scores)

	// Output:
	// [ana ben] [10 20]
}

func ExampleFlatten() {
	pages := slices.Values([]iter.Seq[string]{
		slices.Values([]string{"a", "b"}),
		slices.Values([]string{"c"}),
	})

	fmt.Println(slices.Collect(xs.Flatten(pages)))

	// Output:
	// [a b c]
}

// ExampleFlatMap over a sequence of slices is what [xs.Flatten] would be if it
// took slices, and slices.Values is the whole of the difference.
func ExampleFlatMap() {
	posts := slices.Values([][]string{{"go", "http"}, {"go"}})
	fmt.Println(slices.Collect(xs.FlatMap(posts, slices.Values)))

	// Output:
	// [go http go]
}

// ExampleFlatMap_dropping shows the filter half. An element that turns into an
// empty sequence is not in the result at all.
func ExampleFlatMap_dropping() {
	in := slices.Values([]string{"12", "not a number", "34"})

	numbers := xs.FlatMap(in, func(s string) iter.Seq[int] {
		n, err := strconv.Atoi(s)
		if err != nil {
			return slices.Values([]int(nil))
		}
		return slices.Values([]int{n})
	})

	fmt.Println(slices.Collect(numbers))

	// Output:
	// [12 34]
}

// ExampleChunk turns a sequence of any length into work of a fixed size, which
// is what a bulk insert or a batch API wants.
func ExampleChunk() {
	ids := slices.Values([]int{1, 2, 3, 4, 5})

	for batch := range xs.Chunk(ids, 2) {
		fmt.Println("loading", batch)
	}

	// Output:
	// loading [1 2]
	// loading [3 4]
	// loading [5]
}

// ExampleWindow is for looking at neighbours. Batches from [xs.Chunk] do not
// overlap and these do, which is what makes a difference between one element
// and the next possible to write.
func ExampleWindow() {
	prices := slices.Values([]int{100, 104, 99, 99})

	for pair := range xs.Window(prices, 2) {
		fmt.Printf("%+d\n", pair[1]-pair[0])
	}

	// Output:
	// +4
	// -5
	// +0
}

// ExampleUnique keeps the first of each and the order they arrived in.
// slices.Compact is the one that only looks at neighbours.
func ExampleUnique() {
	tags := slices.Values([]string{"go", "http", "go", "db", "http"})
	fmt.Println(slices.Collect(xs.Unique(tags)))

	// Output:
	// [go http db]
}

func ExampleUniqueBy() {
	names := slices.Values([]string{"Ana", "BEN", "ana", "Cleo"})
	fmt.Println(slices.Collect(xs.UniqueBy(names, strings.ToLower)))

	// Output:
	// [Ana BEN Cleo]
}

// ExampleInterleave takes one from each in turn, and a sequence that runs out
// drops away while the rest carry on.
func ExampleInterleave() {
	europe := slices.Values([]string{"eu-1", "eu-2", "eu-3"})
	asia := slices.Values([]string{"ap-1"})
	americas := slices.Values([]string{"us-1", "us-2"})

	fmt.Println(slices.Collect(xs.Interleave(europe, asia, americas)))

	// Output:
	// [eu-1 ap-1 us-1 eu-2 us-2 eu-3]
}
