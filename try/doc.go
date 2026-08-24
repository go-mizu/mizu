// Package try runs something again when it is worth running again.
//
//	err := try.Do(ctx, func(ctx context.Context) error {
//		return charge(ctx, invoice)
//	})
//
// With no options that is three attempts, exponential backoff with full
// jitter starting at 100 milliseconds and capped at 30 seconds, and a decision
// to retry taken by [github.com/go-mizu/mizu/errs.Retryable]. Options change
// any of those:
//
//	err := try.Do(ctx, fetch,
//		try.Attempts(5),
//		try.Jitter(try.Exponential(50*time.Millisecond, 5*time.Second)),
//		try.Budget(30*time.Second),
//	)
//
// A [Backoff] is an [Option] as well as a type, which is why the strategy goes
// in the list without a wrapper around it.
//
// [Value] is the same thing for a function that returns something.
//
//	user, err := try.Value(ctx, func(ctx context.Context) (User, error) {
//		return client.User(ctx, id)
//	})
//
// # What is retried
//
// A retry is only correct for an error that could go away on its own. Retrying
// a 404 is a slower 404, and retrying a 400 five times is four extra chances to
// be wrong in the same way.
//
// So the default is [github.com/go-mizu/mizu/errs.Retryable], which is true for
// the rate limited, unavailable and timeout kinds and false for the other
// twelve. Retry policy and error classification are then the same decision
// written down once, in the place that already had to make it. Code that
// classifies its errors gets a working retry policy for free, and code that
// does not gets no retries rather than wrong ones.
//
//	try.RetryIf(func(err error) bool { return errs.KindOf(err) == errs.Unavailable })
//
// An error carrying a wait, from [github.com/go-mizu/mizu/errs.Error.WithRetry],
// is honoured when it asks for longer than the backoff would have waited. That
// is where a Retry-After header ends up, and ignoring it is how a client that is
// being rate limited stays rate limited.
//
// # Budgets
//
// An attempt count on its own is a poor bound, because attempts times the
// maximum backoff is almost never the number anybody meant. [Budget] is the
// wall clock bound:
//
//	try.Do(ctx, fn, try.Attempts(0), try.Budget(10*time.Second))
//
// [Attempts] of zero is no attempt limit, so that reads as "keep trying for ten
// seconds". The budget covers the waiting and the decision to go again, not the
// individual attempt: a call that hangs forever hangs forever, and bounding
// that is [context.WithTimeout] on the context handed in, which the attempt
// gets and the budget cannot do for it.
//
// Something has to stop the loop. A call with no attempt limit, no budget and
// no deadline on its context panics rather than retrying until the process is
// killed.
//
// # Testing
//
// The waiting goes through [github.com/go-mizu/mizu/clock], so a test drives it
// rather than living through it:
//
//	c := clock.Fake(time.Now())
//	ctx := clock.With(t.Context(), c)
//
//	go func() { done <- try.Do(ctx, flaky) }()
//	c.BlockUntil(1)          // the first attempt failed and it is waiting
//	c.Advance(time.Minute)   // and now it is not
//
// A test that does not care about the timing sets the backoff to nothing:
//
//	try.Do(ctx, fn, try.Backoff(func(int) time.Duration { return 0 }))
//
// # Cost
//
// A call that succeeds first time is one pass over the options, one call to fn
// and one comparison. On an M4 that is about 45 nanoseconds and one allocation
// of 48 bytes, which is the settled options escaping to the heap.
//
// Every option in the list is a value built at the call site, so a call with
// three of them is three allocations rather than one. Nothing worth worrying
// about next to whatever is being retried, but a hot loop can build the option
// list once and pass the same slice every time.
//
// The default backoff is built once for the package rather than once per call,
// since it is the same two closures every time.
package try
