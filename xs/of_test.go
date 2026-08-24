package xs_test

import (
	"cmp"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestOfAndSlice(t *testing.T) {
	got := xs.Of([]int{1, 2, 3}).Slice()

	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("the chain gave %v, want %v", got, want)
	}
}

func TestOfNothing(t *testing.T) {
	if got := xs.Of([]int(nil)).Slice(); got != nil {
		t.Errorf("a chain over nothing gave %v, want nil", got)
	}
}

func TestFrom(t *testing.T) {
	got := xs.From(slices.Values([]string{"a", "b"})).Slice()

	if want := []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("From gave %q, want %q", got, want)
	}
}

// TestSeqGoesBackToTheFreeFunctions is the promise that the two forms are the
// same thing, so nothing is lost by starting with one and finishing with the
// other.
func TestSeqGoesBackToTheFreeFunctions(t *testing.T) {
	chain := xs.Of([]string{"go", "go", "rust"}).Filter(func(s string) bool { return len(s) == 2 })

	got := slices.Collect(xs.Unique(chain.Seq()))
	if want := []string{"go"}; !slices.Equal(got, want) {
		t.Errorf("the free function over the chain gave %q, want %q", got, want)
	}
}

// TestTheChainIsLazy is the whole point. A chain that is built and never read
// runs nothing, and one that is read partly reads its input partly.
func TestTheChainIsLazy(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4, 5})

	chain := xs.From(in).Filter(func(n int) bool { return n%2 == 1 }).Take(2)
	if *read != 0 {
		t.Fatalf("building the chain read %d elements, want 0", *read)
	}

	got := chain.Slice()
	if want := []int{1, 3}; !slices.Equal(got, want) {
		t.Errorf("the chain gave %v, want %v", got, want)
	}
	if *read != 3 {
		t.Errorf("it read %d elements to find two odd ones, want 3", *read)
	}
}

func TestChainStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3})

	for range xs.From(in).Filter(func(n int) bool { return true }).Seq() {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements before the break, want 1", *read)
	}
}

func TestTheLazyMethods(t *testing.T) {
	odd := func(n int) bool { return n%2 == 1 }
	in := []int{1, 2, 3, 4, 5}

	cases := []struct {
		name string
		got  []int
		want []int
	}{
		{"Filter", xs.Of(in).Filter(odd).Slice(), []int{1, 3, 5}},
		{"Reject", xs.Of(in).Reject(odd).Slice(), []int{2, 4}},
		{"Take", xs.Of(in).Take(2).Slice(), []int{1, 2}},
		{"TakeWhile", xs.Of(in).TakeWhile(func(n int) bool { return n < 3 }).Slice(), []int{1, 2}},
		{"Drop", xs.Of(in).Drop(3).Slice(), []int{4, 5}},
		{"DropWhile", xs.Of(in).DropWhile(func(n int) bool { return n < 4 }).Slice(), []int{4, 5}},
		{"Repeat", xs.Of([]int{1, 2}).Repeat(2).Slice(), []int{1, 2, 1, 2}},
		{"Cycle", xs.Of([]int{1, 2}).Cycle().Take(5).Slice(), []int{1, 2, 1, 2, 1}},
		{"Concat", xs.Of([]int{1}).Concat(xs.Of([]int{2}), xs.Of([]int{3})).Slice(), []int{1, 2, 3}},
	}

	for _, c := range cases {
		if !slices.Equal(c.got, c.want) {
			t.Errorf("%s gave %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestConcatWithNothingAfterIt(t *testing.T) {
	got := xs.Of([]int{1, 2}).Concat().Slice()

	if want := []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("Concat with nothing after it gave %v, want %v", got, want)
	}
}

func TestChainTap(t *testing.T) {
	var seen []int

	got := xs.Of([]int{1, 2, 3}).Tap(func(n int) { seen = append(seen, n) }).Take(2).Slice()
	if want := []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("the chain gave %v, want %v", got, want)
	}
	if !slices.Equal(seen, []int{1, 2}) {
		t.Errorf("Tap saw %v, want only the two that were read", seen)
	}
}

func TestChunkAndWindowGiveBackPlainSequences(t *testing.T) {
	in := xs.Of([]int{1, 2, 3, 4})

	batches := slices.Collect(in.Chunk(2))
	if !equalBatches(batches, [][]int{{1, 2}, {3, 4}}) {
		t.Errorf("Chunk gave %v, want [[1 2] [3 4]]", batches)
	}

	windows := slices.Collect(in.Window(3))
	if !equalBatches(windows, [][]int{{1, 2, 3}, {2, 3, 4}}) {
		t.Errorf("Window gave %v, want [[1 2 3] [2 3 4]]", windows)
	}

	// The point of returning a plain sequence is that From starts a chain again.
	if got := xs.From(in.Chunk(2)).Count(); got != 2 {
		t.Errorf("a chain over the batches counted %d, want 2", got)
	}
}

func TestChainEnumerate(t *testing.T) {
	var at []int
	for i := range xs.Of([]string{"a", "b", "c"}).Enumerate() {
		at = append(at, i)
	}

	if want := []int{0, 1, 2}; !slices.Equal(at, want) {
		t.Errorf("Enumerate counted %v, want %v", at, want)
	}
}

func TestSortFunc(t *testing.T) {
	in := []string{"pear", "fig", "quince"}

	got := xs.Of(in).SortFunc(func(a, b string) int { return cmp.Compare(len(a), len(b)) }).Slice()
	if want := []string{"fig", "pear", "quince"}; !slices.Equal(got, want) {
		t.Errorf("SortFunc gave %q, want %q", got, want)
	}
}

// TestSortFuncReadsEverythingEvenWithATakeAfterIt is the cost the doc comment
// warns about, and the reason SortFunc is the one method that is not lazy.
func TestSortFuncReadsEverythingEvenWithATakeAfterIt(t *testing.T) {
	in, read := counted([]int{5, 1, 4, 2, 3})

	got := xs.From(in).SortFunc(cmp.Compare).Take(2).Slice()
	if want := []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("the chain gave %v, want %v", got, want)
	}
	if *read != 5 {
		t.Errorf("it read %d elements to sort five, want all 5", *read)
	}
}

func TestSortFuncStopsWhenTheCallerDoes(t *testing.T) {
	n := 0
	for range xs.Of([]int{3, 1, 2}).SortFunc(cmp.Compare).Seq() {
		n++
		break
	}
	if n != 1 {
		t.Errorf("the loop ran %d times after a break, want 1", n)
	}
}

func TestSortFuncSortsAgainOnEveryRead(t *testing.T) {
	in, read := counted([]int{2, 1})
	chain := xs.From(in).SortFunc(cmp.Compare)

	chain.Slice()
	chain.Slice()
	if *read != 4 {
		t.Errorf("two reads of the chain read %d elements, want 4", *read)
	}
}

func TestTheTerminalMethods(t *testing.T) {
	in := []int{1, 2, 3, 4}
	chain := xs.Of(in)
	even := func(n int) bool { return n%2 == 0 }

	if got := chain.Count(); got != 4 {
		t.Errorf("Count gave %d, want 4", got)
	}
	if got, ok := chain.First(); !ok || got != 1 {
		t.Errorf("First gave %d, want 1", got)
	}
	if got, ok := chain.Last(); !ok || got != 4 {
		t.Errorf("Last gave %d, want 4", got)
	}
	if got, ok := chain.Find(even); !ok || got != 2 {
		t.Errorf("Find gave %d, want 2", got)
	}
	if got := chain.Index(even); got != 1 {
		t.Errorf("Index gave %d, want 1", got)
	}
	if !chain.Any(even) {
		t.Error("Any said there is no even number in 1, 2, 3, 4")
	}
	if chain.All(even) {
		t.Error("All said 1, 2, 3, 4 are all even")
	}
	if chain.None(even) {
		t.Error("None said there is no even number in 1, 2, 3, 4")
	}
	if got, ok := chain.Reduce(func(a, b int) int { return a + b }); !ok || got != 10 {
		t.Errorf("Reduce gave %d, want 10", got)
	}

	yes, no := chain.PartitionBy(even)
	if !slices.Equal(yes, []int{2, 4}) || !slices.Equal(no, []int{1, 3}) {
		t.Errorf("PartitionBy gave %v and %v, want [2 4] and [1 3]", yes, no)
	}
}

func TestMapTo(t *testing.T) {
	users := []user{{"ana", true}, {"ben", false}, {"cleo", true}}

	got := xs.MapTo(xs.Of(users).Filter(func(u user) bool { return u.Active }), user.name).
		Take(1).
		Slice()

	if want := []string{"ana"}; !slices.Equal(got, want) {
		t.Errorf("the chain gave %q, want %q", got, want)
	}
}

func TestMapToIsLazyToo(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	calls := 0
	got := xs.MapTo(xs.From(in), func(n int) string {
		calls++
		return strings.Repeat("x", n)
	}).Take(2).Slice()

	if want := []string{"x", "xx"}; !slices.Equal(got, want) {
		t.Errorf("MapTo gave %q, want %q", got, want)
	}
	if calls != 2 || *read != 2 {
		t.Errorf("it called fn %d times over %d elements, want 2 and 2", calls, *read)
	}
}

// TestTheChainAndTheFreeFunctionsAgree keeps the two forms honest about being
// the same code, since the methods are only worth having if they are.
func TestTheChainAndTheFreeFunctionsAgree(t *testing.T) {
	in := []int{5, 1, 4, 2, 3, 1}
	odd := func(n int) bool { return n%2 == 1 }

	chained := xs.Of(in).Filter(odd).Drop(1).Take(2).Slice()
	free := slices.Collect(xs.Take(xs.Drop(xs.Filter(slices.Values(in), odd), 1), 2))

	if !slices.Equal(chained, free) {
		t.Errorf("the chain gave %v and the free functions gave %v", chained, free)
	}
}
