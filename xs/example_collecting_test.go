package xs_test

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/go-mizu/mizu/xs"
)

func ExampleGroupBy() {
	words := slices.Values([]string{"go", "rust", "gc", "ruby", "git"})

	byLetter := xs.GroupBy(words, func(s string) string { return s[:1] })
	for _, letter := range slices.Sorted(maps.Keys(byLetter)) {
		fmt.Println(letter, byLetter[letter])
	}
	// Output:
	// g [go gc git]
	// r [rust ruby]
}

func ExampleKeyBy() {
	users := slices.Values([]user{{"ana", true}, {"ben", false}})

	byName := xs.KeyBy(users, func(u user) string { return u.Name })
	fmt.Println(byName["ana"].Active, byName["ben"].Active)
	// Output: true false
}

func ExamplePartitionBy() {
	numbers := slices.Values([]int{1, 2, 3, 4, 5})

	even, odd := xs.PartitionBy(numbers, func(n int) bool { return n%2 == 0 })
	fmt.Println(even, odd)
	// Output: [2 4] [1 3 5]
}

func ExampleJoin() {
	columns := slices.Values([]int{1, 2, 3})

	fmt.Println(xs.Join(xs.Map(columns, strconv.Itoa), ","))
	// Output: 1,2,3
}

func ExampleEachErr() {
	rows := errorFree([]string{"12", "34", "x"})

	err := xs.EachErr(rows, func(s string) error {
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		fmt.Println(n)
		return nil
	})
	fmt.Println(err)
	// Output:
	// 12
	// 34
	// strconv.Atoi: parsing "x": invalid syntax
}

func ExampleShuffle() {
	deck := []int{1, 2, 3, 4, 5}
	xs.Shuffle(deck)

	// The order is different every run, so sorting is the only thing there is to
	// print. Everything that went in is still there.
	slices.Sort(deck)
	fmt.Println(deck)
	// Output: [1 2 3 4 5]
}

func ExampleSample() {
	candidates := []string{"ana", "ben", "cleo", "dai"}

	picked := xs.Sample(candidates, 2)
	fmt.Println(len(picked))
	// Output: 2
}

func ExampleRandom() {
	one, ok := xs.Random([]int{1, 2, 3})
	fmt.Println(ok, one >= 1 && one <= 3)

	_, ok = xs.Random([]int(nil))
	fmt.Println(ok)
	// Output:
	// true true
	// false
}

func ExamplePad() {
	fields := []string{"name", "email"}

	fmt.Printf("%q\n", xs.Pad(fields, 4, ""))
	fmt.Printf("%q\n", xs.Pad(fields, 1, ""))
	// Output:
	// ["name" "email" "" ""]
	// ["name" "email"]
}

func ExampleDiff() {
	before := []string{"a", "b", "c"}
	after := []string{"b", "c", "d"}

	fmt.Println(xs.Diff(before, after), xs.Diff(after, before))
	// Output: [a] [d]
}

func ExampleIntersect() {
	fmt.Println(xs.Intersect([]int{1, 2, 3}, []int{3, 1, 5}))
	// Output: [1 3]
}

func ExampleUnion() {
	fromCache := []string{"a", "b"}
	fromDB := []string{"b", "c"}

	fmt.Println(xs.Union(fromCache, fromDB))
	// Output: [a b c]
}
