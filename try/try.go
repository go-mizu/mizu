package try

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/clock"
	"github.com/go-mizu/mizu/errs"
)

// An Option changes how a call retries. The set is [Attempts], [Budget],
// [RetryIf], [OnRetry] and any [Backoff], which is an Option itself.
//
// It is an interface with an unexported method rather than a function type, so
// that a Backoff can be passed where an Option is wanted without a second name
// for the same idea.
type Option interface {
	apply(*policy)
}

// option is an Option written as a function, which is every one of them except
// Backoff.
type option func(*policy)

func (o option) apply(p *policy) { o(p) }

// defaultBackoff is built once rather than per call, since the two closures
// behind it are the same two closures every time.
var defaultBackoff = Jitter(Exponential(100*time.Millisecond, 30*time.Second))

// policy is the settled form of the options, built once per call.
type policy struct {
	attempts int // Zero is no limit.
	budget   time.Duration
	backoff  Backoff
	retryIf  func(error) bool
	onRetry  func(attempt int, err error)
}

// Attempts is how many times to call the function, counting the first.
//
// Three is the default. Zero is no limit, which needs a [Budget] or a deadline
// on the context to stop it. A negative count panics.
func Attempts(n int) Option {
	if n < 0 {
		panic("try: Attempts with a negative count")
	}
	return option(func(p *policy) { p.attempts = n })
}

// Budget is how long the whole call gets, waiting included.
//
// Once the elapsed time plus the next wait would pass it, the call gives up and
// returns the last error rather than waiting for a deadline it already knows it
// will miss. Zero, the default, is no budget.
//
// This bounds the waiting and the decision to try again. It does not bound an
// individual attempt, since the attempt has the context and [context.WithTimeout]
// is what bounds that.
func Budget(d time.Duration) Option {
	return option(func(p *policy) { p.budget = d })
}

// RetryIf decides whether an error is worth another attempt.
//
// The default is [errs.Retryable], so an error that has been classified already
// carries its own retry policy. Replacing it is for the cases classification
// cannot reach, such as a driver error nobody has mapped yet.
//
//	try.RetryIf(func(err error) bool {
//		return errs.Retryable(err) || errors.Is(err, io.ErrUnexpectedEOF)
//	})
func RetryIf(f func(error) bool) Option {
	if f == nil {
		panic("try: RetryIf with a nil function")
	}
	return option(func(p *policy) { p.retryIf = f })
}

// OnRetry is called before each wait, with the number of the attempt that has
// failed and the error it failed with. It is for a log line, and it runs on the
// calling goroutine, so a slow one is a slow retry.
func OnRetry(f func(attempt int, err error)) Option {
	if f == nil {
		panic("try: OnRetry with a nil function")
	}
	return option(func(p *policy) { p.onRetry = f })
}

// Do calls fn until it returns nil, gives up, or the context ends.
//
// The error is the last one fn returned. When the context ended first it is
// that error joined with the context's, so [errors.Is] finds either one and a
// log line still says what actually failed.
func Do(ctx context.Context, fn func(context.Context) error, opts ...Option) error {
	_, err := Value(ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	}, opts...)
	return err
}

// Value is [Do] for a function that returns something.
//
// On success it returns what the successful attempt returned. On failure it
// returns what the last attempt returned, which is usually the zero value but
// is whatever fn gave back, since a function that returns a partial result with
// an error knows something the caller may want.
func Value[T any](ctx context.Context, fn func(context.Context) (T, error), opts ...Option) (T, error) {
	p := policy{
		attempts: 3,
		backoff:  defaultBackoff,
		retryIf:  errs.Retryable,
	}
	for _, o := range opts {
		o.apply(&p)
	}

	_, hasDeadline := ctx.Deadline()
	if p.attempts == 0 && p.budget == 0 && !hasDeadline {
		panic("try: no attempt limit, no budget and no deadline, so nothing would stop this")
	}

	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	start := clock.Now(ctx)
	for attempt := 1; ; attempt++ {
		v, err := fn(ctx)
		if err == nil {
			return v, nil
		}
		if attempt == p.attempts || !p.retryIf(err) {
			return v, err
		}

		wait := p.backoff(attempt)
		// An error that says how long to wait outranks the backoff when it asks
		// for longer. Asking for less does not shorten the backoff, or a server
		// under load could talk a client into hammering it.
		if d, ok := errs.RetryAfter(err); ok && d > wait {
			wait = d
		}
		if p.budget > 0 && clock.Since(ctx, start)+wait > p.budget {
			return v, err
		}

		if p.onRetry != nil {
			p.onRetry(attempt, err)
		}
		if serr := clock.Sleep(ctx, wait); serr != nil {
			return v, errors.Join(err, serr)
		}
	}
}
