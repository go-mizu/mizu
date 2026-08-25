// Command xm is the same for a map, which is where a missing key and an
// unordered range are the two things worth not writing again.
package main

import (
	"fmt"

	"github.com/go-mizu/mizu/xm"
)

func main() {
	stock := map[string]int{"apples": 3, "pears": 0, "plums": 7}

	fmt.Println(xm.Filter(stock, func(_ string, n int) bool { return n > 0 }))
	fmt.Println(xm.SortedKeys(stock))
	fmt.Println(xm.GetOr(stock, "figs", 0))
}
