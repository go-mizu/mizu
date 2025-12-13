// Package recover provides panic recovery middleware for Mizu applications.
//
// The recover middleware catches panics that occur during HTTP request handling,
// preventing server crashes and enabling graceful error responses. It captures
// stack traces for debugging and integrates with structured logging.
//
// # Basic Usage
//
// The simplest way to use the recover middleware is with default options:
//
//	app := mizu.New()
//	app.Use(recover.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    panic("something went wrong") // Caught and returned as 500 error
//	})
//
// # Configuration Options
//
// The middleware supports several configuration options through the Options struct:
//
//   - StackSize: Buffer size for stack trace capture (default: 4096 bytes)
//   - DisableStackAll: Limit stack trace to current goroutine only
//   - DisablePrintStack: Disable logging of stack traces
//   - ErrorHandler: Custom function to handle recovered panics
//   - Logger: Custom slog.Logger for panic logging
//
// # Custom Error Handling
//
// You can provide a custom error handler to control how panics are processed:
//
//	app.Use(recover.WithOptions(recover.Options{
//	    ErrorHandler: func(c *mizu.Ctx, err any, stack []byte) error {
//	        // Log to external service
//	        errorTracker.Report(err, string(stack))
//
//	        return c.JSON(500, map[string]string{
//	            "error": "Internal server error",
//	        })
//	    },
//	}))
//
// # Production Configuration
//
// For production environments, you may want to disable stack trace printing
// to avoid logging sensitive information:
//
//	app.Use(recover.WithOptions(recover.Options{
//	    DisablePrintStack: true,
//	}))
//
// # Custom Logger Integration
//
// The middleware integrates with Go's log/slog package for structured logging:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//
//	app.Use(recover.WithOptions(recover.Options{
//	    Logger: logger,
//	}))
//
// # Implementation Details
//
// The middleware uses Go's defer/recover mechanism to catch panics. When a panic
// occurs:
//
//  1. The deferred recovery function catches the panic
//  2. Stack trace is captured using runtime/debug.Stack()
//  3. Stack is truncated to StackSize if necessary
//  4. Panic details are logged (unless DisablePrintStack is true)
//  5. Custom ErrorHandler is called if provided, otherwise returns 500 error
//
// # Best Practices
//
//   - Always add recover middleware as the first middleware in your chain
//   - Log panics for debugging and monitoring
//   - Don't expose stack traces to end users in production
//   - Use with request ID middleware for better error correlation
//   - Consider integrating with error tracking services in production
//
// # Performance
//
// The middleware has minimal performance impact:
//
//   - Negligible overhead when no panic occurs (only a defer statement)
//   - Stack trace capture only happens during actual panics
//   - Configurable stack size prevents unbounded memory allocation
//   - Option to disable logging for high-performance production scenarios
package recover
