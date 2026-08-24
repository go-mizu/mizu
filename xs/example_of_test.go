package xs_test

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/go-mizu/mizu/xs"
)

func ExampleOf() {
	users := []user{
		{"ana", true},
		{"ben", false},
		{"cleo", true},
		{"dai", true},
	}

	names := xs.Of(users).
		Filter(func(u user) bool { return u.Active }).
		Map(user.name).
		Take(2).
		Slice()

	fmt.Println(names)
	// Output: [ana cleo]
}

func ExampleFrom() {
	// A sequence from anywhere, a query or a scanner, starts a chain the same way
	// a slice does.
	lines := slices.Values([]string{"package main", "", "func main() {}"})

	code := xs.From(lines).Reject(func(s string) bool { return s == "" }).Count()
	fmt.Println(code)
	// Output: 2
}

func ExampleSeq_Map() {
	widths := xs.Of([]string{"a", "bb", "ccc"}).Map(func(s string) int { return len(s) })

	fmt.Println(widths.Slice())
	// Output: [1 2 3]
}

func ExampleSeq_GroupBy() {
	words := []string{"go", "c", "rust", "zig", "d"}

	byLength := xs.Of(words).GroupBy(func(s string) int { return len(s) })

	for _, n := range slices.Sorted(maps.Keys(byLength)) {
		fmt.Println(n, byLength[n])
	}
	// Output:
	// 1 [c d]
	// 2 [go]
	// 3 [zig]
	// 4 [rust]
}

func ExampleSeq_Seq() {
	// Unique wants a comparable element type, which a method cannot ask of the
	// receiver it was given, so the chain hands the plain sequence to the free
	// function.
	chain := xs.Of([]string{"go", "go", "rust"})

	fmt.Println(slices.Collect(xs.Unique(chain.Seq())))
	// Output: [go rust]
}

func ExampleSeq_SortFunc() {
	words := []string{"quince", "fig", "pear"}

	// Longest first, which is not something a key can say.
	byLength := xs.Of(words).
		SortFunc(func(a, b string) int { return cmp.Compare(len(b), len(a)) }).
		Slice()

	fmt.Println(byLength)
	// Output: [quince pear fig]
}

func ExampleSeq_SortBy() {
	words := []string{"quince", "fig", "pear"}

	byLength := xs.Of(words).SortBy(func(s string) int { return len(s) }).Slice()

	fmt.Println(byLength)
	// Output: [fig pear quince]
}

func ExampleSeq_Chunk() {
	ids := xs.Of([]int{1, 2, 3, 4, 5})

	for batch := range ids.Chunk(2) {
		fmt.Println(batch)
	}
	// Output:
	// [1 2]
	// [3 4]
	// [5]
}

func ExampleSeq_Tap() {
	var log []string

	got := xs.Of([]string{"a", "b", "c"}).
		Tap(func(s string) { log = append(log, s) }).
		Take(2).
		Slice()

	fmt.Println(got, strings.Join(log, ","))
	// Output: [a b] a,b
}
