// Command tool is here because a main package has no export data, and a rule
// over a whole module runs into one sooner or later.
package main

import (
	"fmt"

	"mizu.test/api/store"
)

func main() {
	b, err := store.Encode(store.Record{ID: "1"})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))
}
