package try_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-mizu/mizu/clock"
	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/try"
)

// nowait is the backoff for a test that is about what happens rather than
// about when. Everything about the waiting has its own test, on a fake clock.
var nowait = try.Backoff(func(int) time.Duration { return 0 })

var (
	down    = errs.New(errs.Unavailable, "", "the upstream is down")
	missing = errs.New(errs.NotFound, "", "no such user")
)

// counter is a function that fails a set number of times and then works. It is
// most of what these tests need.
func counter(failures int, err error) (func(context.Context) error, *int) {
	calls := 0
	return func(context.Context) error {
		calls++
		if calls <= failures {
			return err
		}
		return nil
	}, &calls
}

func TestSucceedsFirstTime(t *testing.T) {
	fn, calls := counter(0, down)

	if err := try.Do(t.Context(), fn); err != nil {
		t.Errorf("a function that worked returned %v", err)
	}
	if *calls != 1 {
		t.Errorf("it was called %d times, want 1", *calls)
	}
}

func TestRetriesUntilItWorks(t *testing.T) {
	fn, calls := counter(2, down)

	if err := try.Do(t.Context(), fn, nowait); err != nil {
		t.Errorf("it gave up with %v", err)
	}
	if *calls != 3 {
		t.Errorf("it was called %d times, want 3", *calls)
	}
}

// TestGivesUpAfterTheLastAttempt checks the count and the error together,
// because a retry that swallows the reason it failed is worse than no retry.
func TestGivesUpAfterTheLastAttempt(t *testing.T) {
	fn, calls := counter(100, down)

	err := try.Do(t.Context(), fn, nowait, try.Attempts(4))
	if !errors.Is(err, down) {
		t.Errorf("it returned %v, want the error from the last attempt", err)
	}
	if *calls != 4 {
		t.Errorf("it was called %d times, want 4", *calls)
	}
}

// TestThreeAttemptsByDefault pins the default down, since changing it silently
// would change the load every caller puts on everything it talks to.
func TestThreeAttemptsByDefault(t *testing.T) {
	fn, calls := counter(100, down)

	_ = try.Do(t.Context(), fn, nowait)
	if *calls != 3 {
		t.Errorf("it was called %d times, want the default of 3", *calls)
	}
}

// TestDoesNotRetryWhatWillNotChange is the property that makes this safe to
// wrap around anything. A 404 is a 404 however many times it is asked for.
func TestDoesNotRetryWhatWillNotChange(t *testing.T) {
	fn, calls := counter(100, missing)

	err := try.Do(t.Context(), fn, nowait)
	if !errors.Is(err, missing) {
		t.Errorf("it returned %v, want the not found", err)
	}
	if *calls != 1 {
		t.Errorf("a not found was tried %d times, want 1", *calls)
	}
}

// TestUnclassifiedErrorsAreNotRetried covers the error nobody has classified,
// which errs treats as internal. Retrying one is a guess, and guessing wrong
// means sending the same broken request three times.
func TestUnclassifiedErrorsAreNotRetried(t *testing.T) {
	plain := errors.New("something went wrong")
	fn, calls := counter(100, plain)

	if err := try.Do(t.Context(), fn, nowait); !errors.Is(err, plain) {
		t.Errorf("it returned %v, want the original", err)
	}
	if *calls != 1 {
		t.Errorf("an unclassified error was tried %d times, want 1", *calls)
	}
}

func TestRetryIf(t *testing.T) {
	fn, calls := counter(2, missing)

	err := try.Do(t.Context(), fn, nowait, try.RetryIf(func(err error) bool {
		return errs.KindOf(err) == errs.NotFound
	}))
	if err != nil {
		t.Errorf("it gave up with %v", err)
	}
	if *calls != 3 {
		t.Errorf("it was called %d times, want 3", *calls)
	}
}

func TestOnRetry(t *testing.T) {
	fn, _ := counter(2, down)

	type call struct {
		attempt int
		err     error
	}
	var seen []call
	err := try.Do(t.Context(), fn, nowait, try.OnRetry(func(attempt int, err error) {
		seen = append(seen, call{attempt, err})
	}))
	if err != nil {
		t.Fatalf("it gave up with %v", err)
	}

	// Twice, not three times: it is called before a wait, and the attempt that
	// worked was not followed by one.
	if len(seen) != 2 {
		t.Fatalf("OnRetry ran %d times, want 2", len(seen))
	}
	for i, c := range seen {
		if c.attempt != i+1 {
			t.Errorf("call %d says attempt %d", i, c.attempt)
		}
		if !errors.Is(c.err, down) {
			t.Errorf("call %d carries %v, want the failure", i, c.err)
		}
	}
}

// TestOnRetryDoesNotRunAfterTheLastAttempt is the difference between a log that
// says what happened and one that promises a retry that never comes.
func TestOnRetryDoesNotRunAfterTheLastAttempt(t *testing.T) {
	fn, _ := counter(100, down)

	n := 0
	_ = try.Do(t.Context(), fn, nowait, try.Attempts(2), try.OnRetry(func(int, error) { n++ }))
	if n != 1 {
		t.Errorf("OnRetry ran %d times for two attempts, want 1", n)
	}
}

func TestValue(t *testing.T) {
	calls := 0
	got, err := try.Value(t.Context(), func(context.Context) (string, error) {
		calls++
		if calls < 3 {
			return "", down
		}
		return "hello", nil
	}, nowait)

	if err != nil {
		t.Fatalf("it gave up with %v", err)
	}
	if got != "hello" {
		t.Errorf("it returned %q, want the value from the attempt that worked", got)
	}
}

// TestValueReturnsWhatTheLastAttemptDid matters for a function that returns a
// partial result alongside its error, such as a read that got half a body. The
// caller asked for that value and the retry should not eat it.
func TestValueReturnsWhatTheLastAttemptDid(t *testing.T) {
	n := 0
	got, err := try.Value(t.Context(), func(context.Context) (int, error) {
		n++
		return n, down
	}, nowait, try.Attempts(3))

	if !errors.Is(err, down) {
		t.Errorf("it returned %v", err)
	}
	if got != 3 {
		t.Errorf("it returned %d, want 3 from the third attempt", got)
	}
}

// TestWaitsOnTheContextClock is the whole reason the waiting goes through
// clock. Nothing here sleeps, and the test is over in microseconds.
func TestWaitsOnTheContextClock(t *testing.T) {
	c := clock.Fake(time.Unix(0, 0))
	ctx := clock.With(t.Context(), c)

	fn, calls := counter(2, down)
	done := make(chan error, 1)
	go func() {
		done <- try.Do(ctx, fn, try.Backoff(func(attempt int) time.Duration {
			return time.Duration(attempt) * time.Minute
		}))
	}()

	// One minute after the first failure, two after the second.
	for _, d := range []time.Duration{time.Minute, 2 * time.Minute} {
		c.BlockUntil(1)
		c.Advance(d - time.Nanosecond)
		select {
		case err := <-done:
			t.Fatalf("it went again early, with %v", err)
		default:
		}
		c.Advance(time.Nanosecond)
	}

	if err := <-done; err != nil {
		t.Errorf("it gave up with %v", err)
	}
	if *calls != 3 {
		t.Errorf("it was called %d times, want 3", *calls)
	}
}

// TestGivesUpWithTheCaller checks both halves of the error, since the caller
// wants to know that it was canceled and the log wants to know what failed.
func TestGivesUpWithTheCaller(t *testing.T) {
	c := clock.Fake(time.Unix(0, 0))
	ctx, cancel := context.WithCancel(clock.With(t.Context(), c))

	fn, calls := counter(100, down)
	result := make(chan error, 1)
	go func() { result <- try.Do(ctx, fn, try.Attempts(0), try.Budget(time.Hour)) }()

	c.BlockUntil(1)
	cancel()

	err := <-result
	if !errors.Is(err, context.Canceled) {
		t.Errorf("%v does not say it was canceled", err)
	}
	if !errors.Is(err, down) {
		t.Errorf("%v does not say what failed", err)
	}
	if *calls != 1 {
		t.Errorf("it was called %d times after one cancellation, want 1", *calls)
	}
}

func TestChecksTheContextBeforeTheFirstAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	fn, calls := counter(0, down)
	if err := try.Do(ctx, fn, nowait); !errors.Is(err, context.Canceled) {
		t.Errorf("it returned %v, want context.Canceled", err)
	}
	if *calls != 0 {
		t.Errorf("it called the function %d times on a canceled context, want 0", *calls)
	}
}

// TestBudgetStopsBeforeWaiting is the point of a budget. Waiting thirty seconds
// to find out that thirty seconds was the limit is a worse answer arrived at
// later.
func TestBudgetStopsBeforeWaiting(t *testing.T) {
	c := clock.Fake(time.Unix(0, 0))
	ctx := clock.With(t.Context(), c)

	fn, calls := counter(100, down)
	err := try.Do(ctx, fn, try.Attempts(0), try.Budget(10*time.Second),
		try.Backoff(func(int) time.Duration { return 30 * time.Second }))

	if !errors.Is(err, down) {
		t.Errorf("it returned %v, want the failure", err)
	}
	// A budget is not a deadline. Nothing here was canceled, so the error says
	// what failed and nothing else.
	if errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("%v mentions a deadline, and no deadline was passed", err)
	}
	if *calls != 1 {
		t.Errorf("it was called %d times, want 1, since the first wait already passed the budget", *calls)
	}
	if got := c.Now(); !got.Equal(time.Unix(0, 0)) {
		t.Errorf("the clock moved to %v, and it should not have waited at all", got)
	}
}

// TestBudgetCountsElapsedTime covers the budget running out partway through,
// which is the case an attempt count cannot express.
func TestBudgetCountsElapsedTime(t *testing.T) {
	c := clock.Fake(time.Unix(0, 0))
	ctx := clock.With(t.Context(), c)

	// Each attempt takes a second of clock time on top of its wait, so three
	// seconds of trying and two of waiting reaches the budget.
	calls := 0
	fn := func(context.Context) error {
		calls++
		c.Advance(time.Second)
		return down
	}

	done := make(chan error, 1)
	go func() {
		done <- try.Do(ctx, fn, try.Attempts(0), try.Budget(5*time.Second),
			try.Backoff(func(int) time.Duration { return time.Second }))
	}()

	for range 2 {
		c.BlockUntil(1)
		c.Advance(time.Second)
	}

	if err := <-done; !errors.Is(err, down) {
		t.Errorf("it returned %v", err)
	}
	if calls != 3 {
		t.Errorf("it was called %d times, want 3", calls)
	}
}

// TestBudgetOfZeroIsNoBudget is the default, and it is what makes an attempt
// count on its own work the way somebody who wrote only Attempts expects.
func TestBudgetOfZeroIsNoBudget(t *testing.T) {
	c := clock.Fake(time.Unix(0, 0))
	ctx := clock.With(t.Context(), c)

	fn, calls := counter(100, down)
	done := make(chan error, 1)
	go func() {
		done <- try.Do(ctx, fn, try.Attempts(3),
			try.Backoff(func(int) time.Duration { return 24 * time.Hour }))
	}()

	for range 2 {
		c.BlockUntil(1)
		c.Advance(24 * time.Hour)
	}
	<-done

	if *calls != 3 {
		t.Errorf("it was called %d times, want 3", *calls)
	}
}

// TestRetryAfterIsHonoured is what a 429 with a Retry-After header turns into
// once errs has classified it. Ignoring the header is how a client that is
// being rate limited stays rate limited.
func TestRetryAfterIsHonoured(t *testing.T) {
	c := clock.Fake(time.Unix(0, 0))
	ctx := clock.With(t.Context(), c)

	limited := errs.New(errs.RateLimited, "", "slow down").WithRetry(time.Hour)
	fn, _ := counter(1, limited)

	done := make(chan error, 1)
	go func() {
		done <- try.Do(ctx, fn, try.Backoff(func(int) time.Duration { return time.Millisecond }))
	}()

	c.BlockUntil(1)
	c.Advance(time.Hour - time.Nanosecond)
	select {
	case err := <-done:
		t.Fatalf("it went again before the hour the server asked for, with %v", err)
	default:
	}

	c.Advance(time.Nanosecond)
	if err := <-done; err != nil {
		t.Errorf("it gave up with %v", err)
	}
}

// TestRetryAfterDoesNotShortenTheBackoff is the other direction, and it is a
// safety property: a server that asks for no wait at all should not be able to
// talk a client out of backing off.
func TestRetryAfterDoesNotShortenTheBackoff(t *testing.T) {
	c := clock.Fake(time.Unix(0, 0))
	ctx := clock.With(t.Context(), c)

	limited := errs.New(errs.RateLimited, "", "slow down").WithRetry(time.Millisecond)
	fn, _ := counter(1, limited)

	done := make(chan error, 1)
	go func() {
		done <- try.Do(ctx, fn, try.Backoff(func(int) time.Duration { return time.Hour }))
	}()

	c.BlockUntil(1)
	c.Advance(time.Hour - time.Nanosecond)
	select {
	case err := <-done:
		t.Fatalf("it waited the millisecond the server asked for rather than the hour: %v", err)
	default:
	}

	c.Advance(time.Nanosecond)
	if err := <-done; err != nil {
		t.Errorf("it gave up with %v", err)
	}
}

// TestUnboundedPanics is a programming error caught at the call site rather
// than a process that retries until somebody kills it.
func TestUnboundedPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a call with nothing to stop it did not panic")
		}
	}()
	fn, _ := counter(0, down)
	_ = try.Do(t.Context(), fn, try.Attempts(0))
}

// TestUnboundedIsFineWithADeadline says the guard looks at the context too, so
// a caller who bounded the whole operation does not have to bound it twice.
func TestUnboundedIsFineWithADeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	fn, calls := counter(2, down)
	if err := try.Do(ctx, fn, nowait, try.Attempts(0)); err != nil {
		t.Errorf("it gave up with %v", err)
	}
	if *calls != 3 {
		t.Errorf("it was called %d times, want 3", *calls)
	}
}

func TestOptionsRejectNonsense(t *testing.T) {
	bad := map[string]func(){
		"a negative attempt count": func() { try.Attempts(-1) },
		"a nil RetryIf":            func() { try.RetryIf(nil) },
		"a nil OnRetry":            func() { try.OnRetry(nil) },
	}
	for name, f := range bad {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("it did not panic")
				}
			}()
			f()
		})
	}
}

// TestAttemptsOfOne is a call that opted out of retrying, which is worth
// supporting since it is how a caller turns the behaviour off at one call site
// without changing the shape of the code.
func TestAttemptsOfOne(t *testing.T) {
	fn, calls := counter(100, down)

	if err := try.Do(t.Context(), fn, try.Attempts(1)); !errors.Is(err, down) {
		t.Errorf("it returned %v", err)
	}
	if *calls != 1 {
		t.Errorf("it was called %d times, want 1", *calls)
	}
}

// TestTheDefaultBackoff pins the two things a caller who wrote no options is
// relying on: the first wait is under 100 milliseconds, and it is not always
// 100 milliseconds. A fixed schedule is how a thousand clients that failed
// together come back together and fail together again.
//
// It runs in a synctest bubble so that "has the retry happened yet" has an
// answer rather than a race. That is the composition the clock package doc
// claims: the bubble decides when every goroutine has finished what it can do,
// and the fake clock decides what time it is.
func TestTheDefaultBackoff(t *testing.T) {
	const runs = 20
	shorter := 0

	for range runs {
		synctest.Test(t, func(t *testing.T) {
			c := clock.Fake(time.Unix(0, 0))
			ctx := clock.With(context.Background(), c)

			fn, _ := counter(1, down)
			done := make(chan error, 1)
			go func() { done <- try.Do(ctx, fn) }()

			// Full jitter over the first interval, so the wait is somewhere in
			// [0, 100ms) and about half of the draws are under half of that.
			synctest.Wait()
			c.Advance(50 * time.Millisecond)
			synctest.Wait()

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("it gave up with %v", err)
				}
				shorter++
				return
			default:
			}

			// One nanosecond short of the interval covers every draw there is.
			c.Advance(50*time.Millisecond - time.Nanosecond)
			if err := <-done; err != nil {
				t.Fatalf("it waited past the first interval and returned %v", err)
			}
		})
	}

	// Twenty draws all landing in the top half is a one in a million run, and a
	// backoff that is not jittered fails this every time.
	if shorter == 0 {
		t.Errorf("none of %d retries waited less than half the interval, so the default is not jittered", runs)
	}
}
