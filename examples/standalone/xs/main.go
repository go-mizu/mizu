// Command xs is the loop over a slice written as what it is doing.
package main

import (
	"fmt"
	"slices"

	"github.com/go-mizu/mizu/xs"
)

func main() {
	n := []int{3, 1, 4, 1, 5, 9, 2, 6}

	odd := xs.Of(n).Filter(func(v int) bool { return v%2 == 1 }).SortBy(func(v int) int { return v })
	fmt.Println(odd.Slice())
	fmt.Println(xs.Sum(slices.Values(n)))
	fmt.Println(xs.GroupBy(slices.Values(n), func(v int) bool { return v > 3 }))
}
