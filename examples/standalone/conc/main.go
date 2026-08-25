// Command conc runs the same work over a slice with a limit on how much of it
// happens at once, and gets the results back in the order they went in.
package main

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/conc"
)

func main() {
	pages := []string{"index", "about", "pricing", "contact"}

	sizes, err := conc.Map(context.Background(), pages, 2, func(ctx context.Context, p string) (int, error) {
		return len(p), nil
	})
	fmt.Println(sizes, err)
}
