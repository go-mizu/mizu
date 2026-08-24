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
