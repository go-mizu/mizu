package xm_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/xm"
)

func ExampleOf() {
	row := map[string]string{
		"id":       "42",
		"name":     "ana",
		"password": "hunter2",
		"apiKey":   "sk-live-1",
	}

	public := xm.Of(row).Omit("password", "apiKey")

	for _, k := range xm.SortedKeys(public) {
		fmt.Printf("%s=%s\n", k, public[k])
	}
	// Output:
	// id=42
	// name=ana
}

// The chain is a map, so anything that takes one takes it without a conversion
// and without a call to end the chain.
func ExampleM() {
	headers := map[string]string{"Content-Type": "text/html", "ACCEPT": "*/*"}

	send(xm.Of(headers).MapKeys(func(k, v string) string {
		return strings.ToLower(k)
	}))
	// Output:
	// accept: */*
	// content-type: text/html
}

func send(h map[string]string) {
	for _, k := range xm.SortedKeys(h) {
		fmt.Printf("%s: %s\n", k, h[k])
	}
}

func ExampleM_MapValues() {
	groups := map[string][]string{
		"admin":  {"ana", "ben"},
		"editor": {"cal"},
		"reader": nil,
	}

	sizes := xm.Of(groups).MapValues(func(name string, users []string) int {
		return len(users)
	})

	for _, k := range xm.SortedKeys(sizes) {
		fmt.Printf("%s has %d\n", k, sizes[k])
	}
	// Output:
	// admin has 2
	// editor has 1
	// reader has 0
}

func ExampleM_Map() {
	byName := map[string]int{"ana": 1, "ben": 2}

	byID := xm.Of(byName).Map(func(name string, id int) (int, string) {
		return id, name
	})

	fmt.Println(byID[1], byID[2])
	// Output: ana ben
}

func ExampleM_Merge() {
	defaults := map[string]string{"host": "localhost", "port": "80", "tls": "off"}
	fromFile := map[string]string{"port": "8080"}
	fromEnv := map[string]string{"tls": "on"}

	settings := xm.Of(defaults).Merge(fromFile, fromEnv)

	for _, k := range xm.SortedKeys(settings) {
		fmt.Printf("%s=%s\n", k, settings[k])
	}
	// Output:
	// host=localhost
	// port=8080
	// tls=on
}

// A chain of several steps that each change a type is what generic methods buy.
// Written with the free functions this needs a name for every map in between.
func ExampleM_MapKeys() {
	raw := map[string]string{"a": "1", "b": "", "cc": "3"}

	widths := xm.Of(raw).
		Reject(func(k, v string) bool { return v == "" }).
		MapValues(func(k, v string) int { n, _ := strconv.Atoi(v); return n }).
		MapKeys(func(k string, v int) int { return len(k) })

	fmt.Println(widths[1], widths[2])
	// Output: 1 3
}
