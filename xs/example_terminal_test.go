package xs_test

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/xs"
)

func ExampleReduce() {
	words := slices.Values([]string{"go", "gopher", "http"})

	longest, ok := xs.Reduce(words, func(a, b string) string {
		if len(b) > len(a) {
			return b
		}
		return a
	})
	fmt.Println(longest, ok)

	_, ok = xs.Reduce(slices.Values([]string(nil)), func(a, b string) string { return a })
	fmt.Println(ok)
	// Output:
	// gopher true
	// false
}

func ExampleFold() {
	type item struct {
		Price    int
		Quantity int
	}
	items := slices.Values([]item{{500, 2}, {150, 1}, {99, 3}})

	total := xs.Fold(items, 0, func(sum int, it item) int {
		return sum + it.Price*it.Quantity
	})
	fmt.Println(total)
	// Output: 1447
}

func ExampleSum() {
	fmt.Println(xs.Sum(slices.Values([]int{1, 2, 3, 4})))
	fmt.Println(xs.Sum(slices.Values([]int(nil))))
	// Output:
	// 10
	// 0
}

func ExampleProduct() {
	fmt.Println(xs.Product(slices.Values([]int{2, 3, 4})))
	fmt.Println(xs.Product(slices.Values([]int(nil))))
	// Output:
	// 24
	// 1
}

func ExampleMin() {
	temperatures := slices.Values([]float64{18.5, 12.0, 21.25})

	low, ok := xs.Min(temperatures)
	fmt.Println(low, ok)
	// Output: 12 true
}

func ExampleMax() {
	high, ok := xs.Max(slices.Values([]int{3, 1, 4, 1, 5}))
	fmt.Println(high, ok)

	_, ok = xs.Max(slices.Values([]int(nil)))
	fmt.Println(ok)
	// Output:
	// 5 true
	// false
}

func ExampleMinBy() {
	users := slices.Values([]user{{"cleopatra", true}, {"bo", false}, {"ana", true}})

	shortest, _ := xs.MinBy(users, func(u user) int { return len(u.Name) })
	fmt.Println(shortest.Name)
	// Output: bo
}

func ExampleMaxBy() {
	users := slices.Values([]user{{"cleopatra", true}, {"bo", false}, {"ana", true}})

	longest, _ := xs.MaxBy(users, func(u user) int { return len(u.Name) })
	fmt.Println(longest.Name)
	// Output: cleopatra
}

func ExampleCount() {
	users := slices.Values([]user{{"ana", true}, {"ben", false}, {"cleo", true}})

	active := xs.Filter(users, func(u user) bool { return u.Active })
	fmt.Println(xs.Count(active))
	// Output: 2
}

func ExampleCountBy() {
	words := slices.Values([]string{"go", "rust", "gc", "ruby", "git"})

	perLetter := xs.CountBy(words, func(s string) string { return s[:1] })
	for _, letter := range slices.Sorted(maps.Keys(perLetter)) {
		fmt.Printf("%s %d\n", letter, perLetter[letter])
	}
	// Output:
	// g 3
	// r 2
}

func ExampleFirst() {
	posts := slices.Values([]string{"newest", "older", "oldest"})

	newest, ok := xs.First(posts)
	fmt.Println(newest, ok)
	// Output: newest true
}

func ExampleLast() {
	oldest, ok := xs.Last(slices.Values([]string{"newest", "older", "oldest"}))
	fmt.Println(oldest, ok)
	// Output: oldest true
}

func ExampleFind() {
	files := slices.Values([]string{"go.mod", "main.go", "go.sum"})

	source, found := xs.Find(files, func(s string) bool { return strings.HasSuffix(s, ".go") })
	fmt.Println(source, found)
	// Output: main.go true
}

func ExampleIndex() {
	lines := slices.Values([]string{"package main", "", "func main() {}"})

	at := xs.Index(lines, func(s string) bool { return strings.HasPrefix(s, "func ") })
	fmt.Println(at)

	fmt.Println(xs.Index(lines, func(s string) bool { return s == "type" }))
	// Output:
	// 2
	// -1
}

func ExampleAny() {
	numbers := slices.Values([]int{1, 3, 4, 5})

	fmt.Println(xs.Any(numbers, func(n int) bool { return n%2 == 0 }))
	fmt.Println(xs.Any(numbers, func(n int) bool { return n > 10 }))
	// Output:
	// true
	// false
}

func ExampleAll() {
	fmt.Println(xs.All(slices.Values([]int{2, 4, 6}), func(n int) bool { return n%2 == 0 }))
	fmt.Println(xs.All(slices.Values([]int(nil)), func(n int) bool { return false }))
	// Output:
	// true
	// true
}

func ExampleNone() {
	numbers := slices.Values([]int{1, 3, 5})

	fmt.Println(xs.None(numbers, func(n int) bool { return n%2 == 0 }))
	// Output: true
}

func ExampleCollectErr() {
	lines := []string{"12", "34", "56"}

	numbers, err := xs.CollectErr(xs.MapErr(errorFree(lines), strconv.Atoi))
	fmt.Println(numbers, err)

	_, err = xs.CollectErr(xs.MapErr(errorFree([]string{"12", "x", "56"}), strconv.Atoi))
	fmt.Println(err)
	// Output:
	// [12 34 56] <nil>
	// strconv.Atoi: parsing "x": invalid syntax
}
