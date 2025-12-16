package contract

import (
	"context"
	"runtime/debug"
	"time"
)

// Invoker calls a method with input and returns output.
type Invoker func(ctx context.Context, method *Method, in any) (any, error)

// Middleware wraps an Invoker.
type Middleware func(next Invoker) Invoker

// Chain combines multiple middleware into one.
// Middleware is applied in order, with the first being outermost.
func Chain(mw ...Middleware) Middleware {
	return func(next Invoker) Invoker {
		for i := len(mw) - 1; i >= 0; i-- {
			next = mw[i](next)
		}
		return next
	}
}

// Recovery returns middleware that recovers from panics.
func Recovery() Middleware {
	return func(next Invoker) Invoker {
		return func(ctx context.Context, method *Method, in any) (out any, err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					err = Errorf(Internal, "panic: %v\n%s", r, stack)
				}
			}()
			return next(ctx, method, in)
		}
	}
}

// Timeout returns middleware that adds a timeout to method calls.
func Timeout(d time.Duration) Middleware {
	return func(next Invoker) Invoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			ctx, cancel := context.WithTimeout(ctx, d)
			defer cancel()

			done := make(chan struct{})
			var out any
			var err error

			go func() {
				out, err = next(ctx, method, in)
				close(done)
			}()

			select {
			case <-done:
				return out, err
			case <-ctx.Done():
				return nil, NewError(DeadlineExceeded, "method call timed out")
			}
		}
	}
}

// Logging returns middleware that logs method calls.
func Logging(log func(method string, duration time.Duration, err error)) Middleware {
	return func(next Invoker) Invoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			start := time.Now()
			out, err := next(ctx, method, in)
			log(method.FullName, time.Since(start), err)
			return out, err
		}
	}
}
