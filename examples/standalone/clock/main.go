// Command clock reads the time through a context, which is what makes a test
// able to hand the same code a clock that does not move.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/clock"
)

func main() {
	start := time.Unix(0, 0).UTC()
	c := clock.Fake(start)
	ctx := clock.With(context.Background(), c)

	fmt.Println(clock.Now(ctx))
	c.Advance(90 * time.Minute)
	fmt.Println(clock.Now(ctx), clock.Since(ctx, start))
}
