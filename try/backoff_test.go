package try_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/go-mizu/mizu/try"
)

func TestExponential(t *testing.T) {
	b := try.Exponential(100*time.Millisecond, time.Second)

	want := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		time.Second, // 1.6s would pass the cap.
		time.Second,
	}
	for i, w := range want {
		if got := b(i + 1); got != w {
			t.Errorf("attempt %d waits %v, want %v", i+1, got, w)
		}
	}
}

// TestExponentialDoesNotOverflow is the case that turns a backoff into a
// negative duration and a retry loop into a hot one. A shift of 63 or more is
// undefined in Go, and a duration is an int64 of nanoseconds, so an attempt
// number in the thousands is not a hypothetical for a call with no attempt
// limit.
func TestExponentialDoesNotOverflow(t *testing.T) {
	b := try.Exponential(time.Second, time.Minute)

	for _, attempt := range []int{60, 61, 62, 63, 64, 1000, math.MaxInt32, math.MaxInt} {
		if got := b(attempt); got != time.Minute {
			t.Errorf("attempt %d waits %v, want the cap of 1m", attempt, got)
		}
	}
}

// TestExponentialBelowOne covers a Backoff called by hand, since nothing in the
// package passes an attempt number below one.
func TestExponentialBelowOne(t *testing.T) {
	b := try.Exponential(time.Second, time.Minute)

	for _, attempt := range []int{0, -1} {
		if got := b(attempt); got != time.Second {
			t.Errorf("attempt %d waits %v, want the base of 1s", attempt, got)
		}
	}
}

// TestExponentialWithoutAGap is base equal to max, which is a constant backoff
// and is allowed, unlike a max below the base.
func TestExponentialWithoutAGap(t *testing.T) {
	b := try.Exponential(time.Second, time.Second)

	for attempt := 1; attempt < 5; attempt++ {
		if got := b(attempt); got != time.Second {
			t.Errorf("attempt %d waits %v, want 1s", attempt, got)
		}
	}
}

func TestExponentialRejectsNonsense(t *testing.T) {
	bad := map[string][2]time.Duration{
		"a base of zero":     {0, time.Second},
		"a negative base":    {-time.Second, time.Second},
		"a max below base":   {time.Second, time.Millisecond},
		"a max of zero":      {time.Second, 0},
		"a negative max":     {time.Second, -time.Second},
		"both non positive":  {0, 0},
		"a negative on both": {-time.Second, -time.Minute},
	}
	for name, d := range bad {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Exponential(%v, %v) did not panic", d[0], d[1])
				}
			}()
			try.Exponential(d[0], d[1])
		})
	}
}

// TestJitter checks the range rather than the value, since the value is random
// by design. Full jitter is the whole interval below what it wraps.
func TestJitter(t *testing.T) {
	const d = time.Second
	b := try.Jitter(func(int) time.Duration { return d })

	var total time.Duration
	const runs = 2000
	low, high := 0, 0
	for range runs {
		got := b(1)
		if got < 0 || got >= d {
			t.Fatalf("waited %v, want something in [0, %v)", got, d)
		}
		total += got
		if got < d/2 {
			low++
		} else {
			high++
		}
	}

	// The mean of a uniform draw over [0, 1s) is 500ms. A run this long lands
	// inside 50ms of that unless the distribution is wrong, and this is here to
	// catch a jitter that quietly stopped being one.
	if mean := total / runs; mean < 450*time.Millisecond || mean > 550*time.Millisecond {
		t.Errorf("the mean wait is %v, want about 500ms", mean)
	}
	if low == 0 || high == 0 {
		t.Errorf("%d waits below half the interval and %d above, want both", low, high)
	}
}

// TestJitterOfNothing is the backoff a test sets to skip the waiting, and
// rand.N panics on a bound that is not positive, so this is the guard on that.
func TestJitterOfNothing(t *testing.T) {
	b := try.Jitter(func(attempt int) time.Duration {
		return time.Duration(attempt-2) * time.Second // -1s, 0, 1s
	})

	for _, attempt := range []int{1, 2} {
		if got := b(attempt); got != 0 {
			t.Errorf("attempt %d waits %v, want 0", attempt, got)
		}
	}
	if got := b(3); got < 0 || got >= time.Second {
		t.Errorf("attempt 3 waits %v, want something in [0, 1s)", got)
	}
}

func TestJitterRejectsNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Jitter of nil did not panic")
		}
	}()
	try.Jitter(nil)
}

func TestNilBackoffAsAnOption(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a nil Backoff as an option did not panic")
		}
	}()
	_ = try.Do(t.Context(), func(context.Context) error { return nil }, try.Backoff(nil))
}
