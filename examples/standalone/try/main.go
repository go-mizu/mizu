// Command try calls something that fails twice and then works, waiting a
// little longer after each failure.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/try"
)

func main() {
	calls := 0
	err := try.Do(context.Background(), func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("connection refused")
		}
		return nil
		// Without this only an error the errs package marks retryable is tried
		// again, and a plain error from the standard library is not one.
	}, try.Attempts(5), try.RetryIf(func(error) bool { return true }),
		try.Exponential(time.Millisecond, 20*time.Millisecond))

	fmt.Println(calls, err)
}
