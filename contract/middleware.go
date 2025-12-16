package contract

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"
)

// MethodInvoker is a function that invokes a method.
type MethodInvoker func(ctx context.Context, method *Method, in any) (any, error)

// MethodMiddleware wraps a method invoker.
type MethodMiddleware func(next MethodInvoker) MethodInvoker

// WrappedService is a service with middleware applied.
type WrappedService struct {
	*Service
	invoker MethodInvoker
}

// WithMiddleware wraps a service with middleware.
// Middleware is applied in order, with the first middleware being the outermost.
func (s *Service) WithMiddleware(mw ...MethodMiddleware) *WrappedService {
	// Base invoker calls the method directly
	invoker := MethodInvoker(func(ctx context.Context, method *Method, in any) (any, error) {
		return method.Invoker.Call(ctx, in)
	})

	// Apply middleware in reverse order so first middleware is outermost
	for i := len(mw) - 1; i >= 0; i-- {
		invoker = mw[i](invoker)
	}

	return &WrappedService{
		Service: s,
		invoker: invoker,
	}
}

// Call invokes a method through the middleware chain.
func (w *WrappedService) Call(ctx context.Context, method string, in any) (any, error) {
	m := w.Service.Method(method)
	if m == nil {
		return nil, ErrNotFound("method not found: " + method)
	}
	return w.invoker(ctx, m, in)
}

// CallMethod invokes a method directly through the middleware chain.
func (w *WrappedService) CallMethod(ctx context.Context, m *Method, in any) (any, error) {
	return w.invoker(ctx, m, in)
}

// Logger is an interface for logging middleware.
type Logger interface {
	Log(ctx context.Context, method string, duration time.Duration, err error)
}

// LoggerFunc is a function that implements Logger.
type LoggerFunc func(ctx context.Context, method string, duration time.Duration, err error)

func (f LoggerFunc) Log(ctx context.Context, method string, duration time.Duration, err error) {
	f(ctx, method, duration, err)
}

// LoggingMiddleware logs method calls.
func LoggingMiddleware(logger Logger) MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			start := time.Now()
			out, err := next(ctx, method, in)
			duration := time.Since(start)
			logger.Log(ctx, method.FullName, duration, err)
			return out, err
		}
	}
}

// StdLoggingMiddleware logs to the standard logger.
func StdLoggingMiddleware() MethodMiddleware {
	return LoggingMiddleware(LoggerFunc(func(ctx context.Context, method string, duration time.Duration, err error) {
		if err != nil {
			log.Printf("[contract] %s error=%v duration=%v", method, err, duration)
		} else {
			log.Printf("[contract] %s duration=%v", method, duration)
		}
	}))
}

// RecoveryMiddleware recovers from panics and returns an error.
func RecoveryMiddleware() MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (out any, err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					err = Errorf(ErrCodeInternal, "panic: %v\n%s", r, stack)
				}
			}()
			return next(ctx, method, in)
		}
	}
}

// TimeoutMiddleware adds a timeout to method calls.
func TimeoutMiddleware(timeout time.Duration) MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
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
				return nil, NewError(ErrCodeDeadlineExceeded, "method call timed out")
			}
		}
	}
}

// Metrics is an interface for metrics collection.
type Metrics interface {
	RecordCall(method string, duration time.Duration, err error)
}

// MetricsMiddleware collects metrics on method calls.
func MetricsMiddleware(metrics Metrics) MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			start := time.Now()
			out, err := next(ctx, method, in)
			metrics.RecordCall(method.FullName, time.Since(start), err)
			return out, err
		}
	}
}

// ValidationMiddleware validates inputs (placeholder for future validation).
func ValidationMiddleware() MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			// Future: add input validation based on schema constraints
			return next(ctx, method, in)
		}
	}
}

// ChainMiddleware combines multiple middleware into one.
func ChainMiddleware(mw ...MethodMiddleware) MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		for i := len(mw) - 1; i >= 0; i-- {
			next = mw[i](next)
		}
		return next
	}
}

// ContextKey is a type for context keys.
type ContextKey string

const (
	// ContextKeyMethod is the context key for the current method.
	ContextKeyMethod ContextKey = "contract.method"
	// ContextKeyService is the context key for the current service.
	ContextKeyService ContextKey = "contract.service"
	// ContextKeyRequestID is the context key for request ID.
	ContextKeyRequestID ContextKey = "contract.requestID"
)

// ContextMiddleware adds method info to the context.
func ContextMiddleware() MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			ctx = context.WithValue(ctx, ContextKeyMethod, method)
			ctx = context.WithValue(ctx, ContextKeyService, method.Service)
			return next(ctx, method, in)
		}
	}
}

// MethodFromContext returns the current method from context.
func MethodFromContext(ctx context.Context) *Method {
	if m, ok := ctx.Value(ContextKeyMethod).(*Method); ok {
		return m
	}
	return nil
}

// ServiceFromContext returns the current service from context.
func ServiceFromContext(ctx context.Context) *Service {
	if s, ok := ctx.Value(ContextKeyService).(*Service); ok {
		return s
	}
	return nil
}

// BeforeHook is called before method invocation.
type BeforeHook func(ctx context.Context, method *Method, in any) error

// AfterHook is called after method invocation.
type AfterHook func(ctx context.Context, method *Method, in, out any, err error)

// HooksMiddleware adds before and after hooks.
func HooksMiddleware(before BeforeHook, after AfterHook) MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			if before != nil {
				if err := before(ctx, method, in); err != nil {
					return nil, err
				}
			}

			out, err := next(ctx, method, in)

			if after != nil {
				after(ctx, method, in, out, err)
			}

			return out, err
		}
	}
}

// ConditionalMiddleware applies middleware only if the condition is true.
func ConditionalMiddleware(cond func(*Method) bool, mw MethodMiddleware) MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		wrapped := mw(next)
		return func(ctx context.Context, method *Method, in any) (any, error) {
			if cond(method) {
				return wrapped(ctx, method, in)
			}
			return next(ctx, method, in)
		}
	}
}

// RetryMiddleware retries failed method calls.
func RetryMiddleware(maxRetries int, retryable func(error) bool) MethodMiddleware {
	return func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			var lastErr error
			for i := 0; i <= maxRetries; i++ {
				out, err := next(ctx, method, in)
				if err == nil {
					return out, nil
				}
				lastErr = err
				if !retryable(err) {
					return nil, err
				}
				// Simple backoff
				if i < maxRetries {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(time.Duration(i+1) * 100 * time.Millisecond):
					}
				}
			}
			return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
		}
	}
}
