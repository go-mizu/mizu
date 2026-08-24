package conc

import (
	"context"
	"errors"
	"iter"
)

// errWon is the cause the losers of a [Race] see, and errNoMore the one a
// [MapSeq] worker sees once the caller has stopped reading. Neither is returned
// to anybody, so neither is exported. They exist so that a goroutine asking
// [context.Cause] why it was cancelled gets a sentence rather than "context
// canceled".
var (
	errWon    = errors.New("conc: something else finished first")
	errNoMore = errors.New("conc: the caller stopped reading")
)

// call runs fn and turns a panic into an error, so that whoever is waiting for
// a result gets one either way.
//
// A [Group] would recover the panic too, but into the group's error, and a
// goroutine that dies that way never sends the result somebody is blocked on.
func call[T any](ctx context.Context, fn func(context.Context) (T, error)) (v T, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = recovered(p)
		}
	}()
	return fn(ctx)
}

// Map applies fn to every element of in, at most n at a time, and returns the
// results in the order of the input.
//
//	prices, err := conc.Map(ctx, ids, 8, func(ctx context.Context, id int) (Price, error) {
//		return client.Price(ctx, id)
//	})
//
// The first error cancels the rest and comes back on its own, with no results.
// A half filled slice is a thing nobody can use safely, and the caller who
// wants the successes anyway has [Each] and a slice of their own.
//
// n is how many run at once and must be at least one. An empty input does
// nothing at all, so it does not mind what n is, which means passing len(in) is
// a way to ask for no limit that does not blow up on an empty slice.
func Map[T, R any](ctx context.Context, in []T, n int, fn func(context.Context, T) (R, error)) ([]R, error) {
	out := make([]R, len(in))
	if len(in) == 0 {
		return out, nil
	}
	if n < 1 {
		panic("conc: Map with a limit below one")
	}

	g, _ := NewGroup(ctx, Limit(n))
	for i, v := range in {
		// Each goroutine owns one slot, so the results keep the order of the
		// input without a lock and without a sort afterwards.
		g.Go(func(ctx context.Context) error {
			r, err := fn(ctx, v)
			if err != nil {
				return err
			}
			out[i] = r
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// Each applies fn to every element of in, at most n at a time, and returns the
// first error.
//
//	err := conc.Each(ctx, users, 4, func(ctx context.Context, u User) error {
//		return mail.Send(ctx, u, welcome)
//	})
//
// It is [Map] for work that has an effect rather than a result. The first error
// cancels the rest, so some of the work will not have been done when one comes
// back.
//
// n is how many run at once and must be at least one, except on an empty input,
// which does nothing.
func Each[T any](ctx context.Context, in []T, n int, fn func(context.Context, T) error) error {
	if len(in) == 0 {
		return nil
	}
	if n < 1 {
		panic("conc: Each with a limit below one")
	}

	g, _ := NewGroup(ctx, Limit(n))
	for _, v := range in {
		g.Go(func(ctx context.Context) error { return fn(ctx, v) })
	}
	return g.Wait()
}

// result is one finished piece of work on its way back to the caller.
type result[R any] struct {
	v   R
	err error
}

// MapSeq is [Map] over a sequence, yielding each result as it is ready instead
// of collecting them all first.
//
//	for row, err := range conc.MapSeq(ctx, rows, 8, enrich) {
//		if err != nil {
//			return err
//		}
//		w.Write(row)
//	}
//
// Use it when the input is long enough that holding every result in memory is
// the problem, or when it does not end. n elements are in flight at a time and
// nothing is read from in ahead of that, so a sequence of ten million rows
// costs the same as a sequence of ten.
//
// Results come out in the order they went in, the same as [Map]. That costs
// something: an element that takes a long time holds up the ones behind it,
// which are finished and waiting. The alternative is a sequence whose order has
// nothing to do with the input, and a caller who wants that can stop caring
// about the order instead.
//
// An error is yielded against the element it belongs to and does not end the
// sequence. Break to make it end. This is the [iter.Seq2] contract rather than
// this package being relaxed about failure: a sequence hands the decision to
// the caller, one element at a time, and stopping early is how the caller says
// no.
//
// Breaking out, or returning from inside the loop, cancels everything in flight
// and waits for it before the loop is over. Nothing this started outlives it.
//
// If in itself panics, the elements it had already handed over come out first
// and that panic is the last error the sequence yields. The same goes for the
// context ending: what is in flight finishes, and then the reason arrives.
func MapSeq[T, R any](ctx context.Context, in iter.Seq[T], n int, fn func(context.Context, T) (R, error)) iter.Seq2[R, error] {
	if n < 1 {
		panic("conc: MapSeq with a limit below one")
	}
	return func(yield func(R, error) bool) {
		var zero R

		g, gctx := NewGroup(ctx)

		// slots is the pipeline, one channel per element in flight and in the
		// order the elements arrived, which is what keeps the output ordered.
		//
		// The capacity is n-1 rather than n because the element being waited on
		// has already been taken out of the channel and is still running. With
		// n-1 queued behind it that comes to n.
		slots := make(chan chan result[R], n-1)

		feeder := func(ctx context.Context) error {
			defer close(slots)
			for v := range in {
				ch := make(chan result[R], 1)
				select {
				case slots <- ch:
				default:
					// Room in the pipeline is taken without asking whether the
					// context is still alive, because this element has already
					// been pulled out of in and giving up on it here would drop
					// it without the caller ever hearing that it existed. The
					// cancellation only matters when there is no room, which is
					// where this would otherwise wait.
					select {
					case slots <- ch:
					case <-ctx.Done():
						return nil
					}
				}

				started := g.run(ctx, func(ctx context.Context) error {
					r, err := call(ctx, func(ctx context.Context) (R, error) {
						return fn(ctx, v)
					})
					ch <- result[R]{r, err}
					return nil
				})
				if !started {
					// The group ended between this slot being taken and its
					// goroutine starting, so nothing else is going to write
					// here and the loop below would wait for it forever.
					ch <- result[R]{err: context.Cause(gctx)}
				}
			}
			return nil
		}

		if !g.run(gctx, feeder) {
			// The context was over before any of this began, so slots is never
			// closed and there is nothing to read.
			yield(zero, context.Cause(gctx))
			g.stop(errNoMore)
			return
		}

		// Every channel that reaches this loop is written exactly once, which
		// is what lets it be an ordinary receive rather than a race between a
		// result and a cancellation.
		//
		// stopped records that yield has already returned false, since calling
		// it again after that is not allowed.
		stopped := false
		for ch := range slots {
			r := <-ch
			if !yield(r.v, r.err) {
				stopped = true
				break
			}
		}

		// Whatever happened, nothing this started is left running.
		//
		// The reason it ended comes from stop rather than from the group's
		// context because stop waits. An input that panics closes the pipeline
		// on its way out, which can reach the loop above before the panic has
		// become an error, and reading the context here would be a race with
		// that.
		err := g.stop(errNoMore)
		if stopped {
			return
		}
		if err == nil && ctx.Err() != nil {
			err = context.Cause(ctx)
		}
		if err != nil {
			yield(zero, err)
		}
	}
}

// All runs every function at once and returns their results in order.
//
//	pages, err := conc.All(ctx,
//		func(ctx context.Context) (Page, error) { return header(ctx) },
//		func(ctx context.Context) (Page, error) { return body(ctx) },
//	)
//
// It is [Map] for a fixed list of different jobs rather than one job over a
// list, so there is no limit to pass: the limit is how many arguments were
// written. The first error cancels the rest and comes back on its own.
//
// No functions is not an error. Everything in an empty list succeeded.
func All[T any](ctx context.Context, fns ...func(context.Context) (T, error)) ([]T, error) {
	out := make([]T, len(fns))
	if len(fns) == 0 {
		return out, nil
	}

	g, _ := NewGroup(ctx)
	for i, fn := range fns {
		g.Go(func(ctx context.Context) error {
			v, err := fn(ctx)
			if err != nil {
				return err
			}
			out[i] = v
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// Race runs every function at once and returns the first result that is not an
// error, cancelling the rest.
//
//	body, err := conc.Race(ctx, fromCache, fromOrigin)
//
// This is for the cases where two ways of getting the same answer are both
// worth trying, such as a replica and its primary, or a mirror in each region.
// A loser asking [context.Cause] why it was cancelled is told that something
// else finished first, so it can tell that apart from the request being
// abandoned.
//
// Race returns when a function succeeds, but not before every other one has
// stopped. A background goroutine still writing somewhere after the caller has
// moved on is the thing this package is for.
//
// If they all fail the errors come back joined, in the order the functions were
// written, since the one that lost the race is often the one that explains it.
//
// No functions panics. There is no first result of nothing, and returning a
// zero value with no error would be a wrong answer rather than a slow one.
func Race[T any](ctx context.Context, fns ...func(context.Context) (T, error)) (T, error) {
	var zero T
	if len(fns) == 0 {
		panic("conc: Race with no functions")
	}

	// Buffered for the whole field, so a loser that finishes after the race is
	// over still has somewhere to put its result and does not block on the way
	// out.
	ch := make(chan result[T], len(fns))

	g, gctx := NewGroup(ctx)
	for _, fn := range fns {
		started := g.run(gctx, func(ctx context.Context) error {
			v, err := call(ctx, fn)
			ch <- result[T]{v, err}
			// The group's error is not the race's error, so a failure here is
			// reported through the channel and not by failing the group.
			return nil
		})
		if !started {
			// The context was over before this one got going, so it counts as a
			// runner that did not finish rather than one nobody hears from.
			ch <- result[T]{err: context.Cause(gctx)}
		}
	}

	// Every function puts exactly one result in the channel, whether it ran or
	// not, so this reads a fixed number of them and cannot be left waiting.
	failures := make([]error, 0, len(fns))
	for range fns {
		r := <-ch
		if r.err == nil {
			g.stop(errWon)
			return r.v, nil
		}
		failures = append(failures, r.err)
	}

	g.stop(errWon)
	return zero, errors.Join(failures...)
}
