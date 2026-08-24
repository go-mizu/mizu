package try_test

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/try"
)

func Example() {
	attempts := 0
	err := try.Do(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errs.New(errs.Unavailable, "", "the upstream is down")
		}
		return nil
	}, try.Backoff(func(int) time.Duration { return 0 })) // No waiting, for the example.

	fmt.Println(attempts, err)

	// Output:
	// 3 <nil>
}

// ExampleValue is the form for a call that returns something, which is most of
// them.
func ExampleValue() {
	type User struct{ Name string }

	attempts := 0
	user, err := try.Value(context.Background(), func(ctx context.Context) (User, error) {
		attempts++
		if attempts == 1 {
			return User{}, errs.New(errs.Timeout, "", "the request timed out")
		}
		return User{Name: "ada"}, nil
	}, try.Backoff(func(int) time.Duration { return 0 }))

	fmt.Println(user.Name, err)

	// Output:
	// ada <nil>
}

// ExampleRetryIf shows the default rather than a replacement for it, because
// the default is the part worth understanding. An error that has been
// classified already carries its own retry policy.
func ExampleRetryIf() {
	for _, err := range []error{
		errs.New(errs.Unavailable, "", "the upstream is down"),
		errs.New(errs.NotFound, "", "no such user"),
		errs.New(errs.Invalid, "", "the email is not an email"),
		errs.New(errs.RateLimited, "", "slow down"),
	} {
		attempts := 0
		_ = try.Do(context.Background(), func(ctx context.Context) error {
			attempts++
			return err
		}, try.Backoff(func(int) time.Duration { return 0 }))

		fmt.Printf("%-13s %d\n", errs.KindOf(err), attempts)
	}

	// Output:
	// unavailable   3
	// not_found     1
	// invalid       1
	// rate_limited  3
}

// ExampleBudget is the bound most people mean when they write an attempt
// count. Attempts of zero lifts the count so that the budget is the only thing
// stopping it.
func ExampleBudget() {
	err := try.Do(context.Background(), func(ctx context.Context) error {
		return errs.New(errs.Unavailable, "", "the upstream is down")
	}, try.Attempts(0), try.Budget(200*time.Millisecond))

	fmt.Println(err)

	// Output:
	// the upstream is down
}

// ExampleOnRetry is where a log line goes. It runs before the wait, so it can
// say what failed but not yet what happened next.
func ExampleOnRetry() {
	attempts := 0
	_ = try.Do(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errs.New(errs.Timeout, "", "the request timed out")
		}
		return nil
	},
		try.Backoff(func(int) time.Duration { return 0 }),
		try.OnRetry(func(attempt int, err error) {
			fmt.Printf("attempt %d failed: %v\n", attempt, err)
		}),
	)

	// Output:
	// attempt 1 failed: the request timed out
	// attempt 2 failed: the request timed out
}
