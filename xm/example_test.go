package xm_test

import (
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/xm"
)

func ExampleKeys() {
	m := map[string]int{"a": 1, "b": 2}

	// Keys comes back in map order, so sort it before printing anything a
	// reader is going to compare against.
	fmt.Println(len(xm.Keys(m)))
	// Output: 2
}

func ExampleValues() {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	total := 0
	for _, v := range xm.Values(m) {
		total += v
	}
	fmt.Println(total)
	// Output: 6
}

func ExampleSortedKeys() {
	settings := map[string]string{"port": "8080", "host": "localhost", "tls": "on"}

	for _, k := range xm.SortedKeys(settings) {
		fmt.Printf("%s=%s\n", k, settings[k])
	}
	// Output:
	// host=localhost
	// port=8080
	// tls=on
}

func ExampleEntries() {
	m := map[string]int{"only": 1}

	for _, e := range xm.Entries(m) {
		fmt.Println(e.Key, e.Value)
	}
	// Output: only 1
}

func ExampleFromEntries() {
	es := []xm.Entry[string, int]{
		{Key: "a", Value: 1},
		{Key: "b", Value: 2},
	}

	m := xm.FromEntries(es)
	fmt.Println(m["a"], m["b"])
	// Output: 1 2
}

func ExampleMap() {
	byName := map[string]int{"ana": 7}

	byID := xm.Map(byName, func(name string, id int) (int, string) {
		return id, name
	})
	fmt.Println(byID[7])
	// Output: ana
}

func ExampleMapKeys() {
	headers := map[string]string{"Content-Type": "application/json"}

	lower := xm.MapKeys(headers, func(k, v string) string {
		return strings.ToLower(k)
	})
	fmt.Println(lower["content-type"])
	// Output: application/json
}

func ExampleMapValues() {
	groups := map[string][]string{"admins": {"ana", "ben"}, "guests": {"cara"}}

	counts := xm.MapValues(groups, func(name string, members []string) int {
		return len(members)
	})
	fmt.Println(counts["admins"], counts["guests"])
	// Output: 2 1
}

func ExampleFilter() {
	features := map[string]bool{"logging": true, "tracing": false}

	enabled := xm.Filter(features, func(name string, on bool) bool { return on })
	fmt.Println(xm.SortedKeys(enabled))
	// Output: [logging]
}

func ExampleReject() {
	scores := map[string]int{"ana": 90, "ben": 40, "cara": 75}

	passed := xm.Reject(scores, func(name string, score int) bool { return score < 50 })
	fmt.Println(xm.SortedKeys(passed))
	// Output: [ana cara]
}

func ExampleInvert() {
	idByName := map[string]int{"ana": 1}

	nameByID := xm.Invert(idByName)
	fmt.Println(nameByID[1])
	// Output: ana
}

func ExampleMerge() {
	defaults := map[string]string{"host": "localhost", "port": "8080"}
	fromEnv := map[string]string{"port": "9090"}

	settings := xm.Merge(defaults, fromEnv)
	for _, k := range xm.SortedKeys(settings) {
		fmt.Printf("%s=%s\n", k, settings[k])
	}
	// Output:
	// host=localhost
	// port=9090
}

func ExampleMergeWith() {
	january := map[string]int{"ana": 3, "ben": 1}
	february := map[string]int{"ana": 4}

	totals := xm.MergeWith(func(name string, a, b int) int { return a + b }, january, february)
	fmt.Println(totals["ana"], totals["ben"])
	// Output: 7 1
}

func ExamplePick() {
	row := map[string]string{"id": "7", "name": "ana", "password": "hunter2"}

	summary := xm.Pick(row, "id", "name")
	fmt.Println(xm.SortedKeys(summary))
	// Output: [id name]
}

func ExampleOmit() {
	row := map[string]string{"id": "7", "name": "ana", "password": "hunter2"}

	safe := xm.Omit(row, "password")
	fmt.Println(xm.SortedKeys(safe))
	// Output: [id name]
}

func ExampleGetOr() {
	settings := map[string]int{"retries": 0}

	// The 0 is really in the map, so it stays 0 rather than falling back.
	fmt.Println(xm.GetOr(settings, "retries", 3))
	fmt.Println(xm.GetOr(settings, "timeout", 30))
	// Output:
	// 0
	// 30
}

func ExampleUpdate() {
	counts := map[string]int{}

	for _, word := range strings.Fields("go go mizu") {
		xm.Update(counts, word, func(n int) int { return n + 1 })
	}
	fmt.Println(counts["go"], counts["mizu"])
	// Output: 2 1
}
