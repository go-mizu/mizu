package conc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-mizu/mizu/errs"
)

// key is what a Group is stored under in a context.
type key struct{}

// A Group runs goroutines and collects what happens to them.
//
// It is created together with the context its goroutines run under, and that
// context is cancelled as soon as one of them fails, so the rest find out
// without anybody having to tell them.
//
//	g, ctx := conc.NewGroup(ctx, conc.Limit(8))
//	for _, u := range urls {
//		g.Go(func(ctx context.Context) error { return fetch(ctx, u) })
//	}
//	if err := g.Wait(); err != nil {
//		return err
//	}
//
// The zero Group is not usable. Use [NewGroup].
type Group struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	// sem is the concurrency limit, and is nil when there is none.
	sem chan struct{}

	wg sync.WaitGroup

	// once guards err, so the error that comes out is the first one in rather
	// than whichever goroutine finished last.
	once sync.Once
	err  error

	// waited is set once Wait has returned, which is when starting anything
	// else is a leak rather than a fan-out.
	waited atomic.Bool
}

// A GroupOption changes how a [Group] is built. The set is [Limit].
type GroupOption interface {
	apply(*Group)
}

// groupOption is a GroupOption written as a function.
type groupOption func(*Group)

func (o groupOption) apply(g *Group) { o(g) }

// Limit is how many goroutines the group runs at once.
//
// Over the limit, [Group.Go] blocks until a slot frees up, so a loop over ten
// thousand rows with a limit of eight holds eight connections rather than ten
// thousand. Without it a group runs everything it is given at once.
//
// The count must be at least one. Zero would be a group that never runs
// anything, which is a typo rather than a policy, so it panics.
func Limit(n int) GroupOption {
	if n < 1 {
		panic("conc: Limit below one")
	}
	return groupOption(func(g *Group) { g.sem = make(chan struct{}, n) })
}

// NewGroup returns a group and the context its goroutines run under.
//
// The context is a child of ctx that is cancelled when the first goroutine
// fails, when [Group.Wait] returns, or when ctx itself is cancelled.
// [context.Cause] on it names the failure, so a goroutine that stopped early
// can say what it was that actually went wrong rather than reporting the
// cancellation it noticed.
//
// It also carries the group, which is what makes the package level [Go] and
// [Wait] work further down the call stack.
func NewGroup(ctx context.Context, opts ...GroupOption) (*Group, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	g := &Group{cancel: cancel}
	for _, o := range opts {
		o.apply(g)
	}
	g.ctx = context.WithValue(ctx, key{}, g)
	return g, g.ctx
}

// Go runs fn in a new goroutine, blocking first if the group is at its limit.
//
// fn is given the group's context. A panic inside it is recovered and becomes
// the group's error, so a bug in one goroutine takes down the group rather than
// the process.
//
// Calling this after [Group.Wait] has returned panics, since there would be
// nothing left to wait for it.
func (g *Group) Go(fn func(context.Context) error) { g.run(g.ctx, fn) }

// run is Go with the context to hand fn spelled out, because the package level
// Go hands over the caller's context rather than the group's.
func (g *Group) run(ctx context.Context, fn func(context.Context) error) {
	if g.waited.Load() {
		panic("conc: Go after Wait returned, so nothing would wait for this goroutine")
	}
	// Nothing new starts once the group is over. Handing fn a context that is
	// already cancelled would mean every fn in the codebase has to check for
	// that itself, and most of them would not.
	if g.ctx.Err() != nil {
		g.fail(context.Cause(g.ctx))
		return
	}
	if g.sem != nil {
		select {
		case g.sem <- struct{}{}:
		case <-g.ctx.Done():
			g.fail(context.Cause(g.ctx))
			return
		}
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if g.sem != nil {
			defer func() { <-g.sem }()
		}
		defer func() {
			if v := recover(); v != nil {
				g.fail(recovered(v))
			}
		}()
		if err := fn(ctx); err != nil {
			g.fail(err)
		}
	}()
}

// Wait blocks until every goroutine has finished and returns the first error
// any of them produced.
//
// The group's context is cancelled by the time this returns, whether anything
// failed or not, so it is not the context to carry on with afterwards.
func (g *Group) Wait() error {
	g.wg.Wait()
	g.waited.Store(true)
	g.cancel(g.err)
	return g.err
}

// fail records the first error and cancels the group with it.
func (g *Group) fail(err error) {
	g.once.Do(func() {
		g.err = err
		g.cancel(err)
	})
}

// recovered turns a recovered panic value into an error.
//
// The stack is captured here, inside the deferred function, which is while the
// panicking frames are still on the stack. So the trace on the error points at
// the line that panicked and not at this one.
func recovered(v any) error {
	err, ok := v.(error)
	if !ok {
		err = fmt.Errorf("%v", v)
	}
	return errs.Wrap(err, errs.Internal, "panic", "a goroutine panicked")
}

// Go runs fn in the group carried by ctx.
//
// This is the supported way to start a goroutine, and the reason framework code
// contains no bare go statement. Somebody is waiting for the result, a panic
// comes back as an error, and a failure cancels the work alongside it.
//
//	conc.Go(ctx, func(ctx context.Context) error {
//		return audit.Record(ctx, event)
//	})
//
// fn is given ctx rather than the group's own context, so values added since
// the group was created are still there.
//
// A context with no group in it panics. A goroutine with nothing waiting for it
// and nowhere to report is the bug this package exists to stop, and starting it
// anyway would hide it until something in production went quiet.
func Go(ctx context.Context, fn func(context.Context) error) {
	g, ok := ctx.Value(key{}).(*Group)
	if !ok {
		panic("conc: Go without a group in the context, so nothing would wait for this goroutine")
	}
	g.run(ctx, fn)
}

// Wait blocks until the goroutines started on ctx have finished and returns the
// first error any of them produced.
//
// A context with no group in it returns nil, since there is nothing to wait
// for. That is not the mistake [Go] guards against: code that never started a
// goroutine has nothing to leak, and a middleware that ends every request with
// this should not care whether the handler used one.
func Wait(ctx context.Context) error {
	g, ok := ctx.Value(key{}).(*Group)
	if !ok {
		return nil
	}
	return g.Wait()
}
