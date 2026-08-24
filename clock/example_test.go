package clock_test

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/clock"
)

// Example is the shape every use of this package has: production code reads
// the time from the context and a test decides what the context says.
func Example() {
	// Production code. It does not know or care which clock it got.
	expired := func(ctx context.Context, issued time.Time) bool {
		return clock.Since(ctx, issued) > 24*time.Hour
	}

	issued := time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)

	c := clock.Fake(issued.Add(23 * time.Hour))
	ctx := clock.With(context.Background(), c)
	fmt.Println(expired(ctx, issued))

	c.Advance(2 * time.Hour)
	fmt.Println(expired(ctx, issued))

	// Output:
	// false
	// true
}

// ExampleFake shows the case a synctest bubble cannot cover, which is code that
// cares what the date is rather than how long something took.
func ExampleFake() {
	lastDayOfMonth := func(ctx context.Context) bool {
		now := clock.Now(ctx)
		return now.AddDate(0, 0, 1).Day() == 1
	}

	// Ten to midnight on the last day of March, in Tokyo.
	c := clock.Fake(time.Date(2026, 3, 31, 23, 50, 0, 0, time.FixedZone("JST", 9*3600)))
	ctx := clock.With(context.Background(), c)
	fmt.Println(lastDayOfMonth(ctx))

	c.Advance(10 * time.Minute)
	fmt.Println(clock.Now(ctx).Format(time.RFC3339), lastDayOfMonth(ctx))

	// Output:
	// true
	// 2026-04-01T00:00:00+09:00 false
}

// ExampleFakeClock_BlockUntil is the answer to the one race a fake clock has,
// which is the test advancing before the code under test has set its timer.
func ExampleFakeClock_BlockUntil() {
	c := clock.Fake(time.Unix(0, 0))
	ctx := clock.With(context.Background(), c)

	done := make(chan error, 1)
	go func() {
		done <- clock.Sleep(ctx, time.Hour)
	}()

	c.BlockUntil(1) // The goroutine is now waiting on its timer.
	c.Advance(time.Hour)

	fmt.Println(<-done)

	// Output:
	// <nil>
}

// ExampleSleep shows the difference between this and [time.Sleep], which is
// that a caller who has given up does not have to wait for the sleep to finish.
func ExampleSleep() {
	c := clock.Fake(time.Unix(0, 0))
	ctx, cancel := context.WithCancel(clock.With(context.Background(), c))

	done := make(chan error, 1)
	go func() {
		done <- clock.Sleep(ctx, time.Hour)
	}()

	c.BlockUntil(1)
	cancel()

	fmt.Println(<-done)

	// Output:
	// context canceled
}
