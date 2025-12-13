/*
Package expvar provides expvar endpoint middleware for Mizu.

# Overview

The expvar middleware exposes Go's expvar package variables through an
HTTP endpoint. This is useful for monitoring application metrics and
debugging runtime statistics.

# Usage

Basic usage:

	app := mizu.New()
	app.Use(expvar.New())

	// Access at /debug/vars

# Configuration

Options:

  - Path: URL path for the expvar endpoint (default: "/debug/vars")

# Helper Functions

The package provides convenience functions:

  - NewInt: Create and publish an expvar.Int
  - NewFloat: Create and publish an expvar.Float
  - NewString: Create and publish an expvar.String
  - NewMap: Create and publish an expvar.Map
  - Publish: Publish a custom Var
  - Get: Retrieve a published Var by name

# Example

Create and use custom metrics:

	requestCount := expvar.NewInt("requests_total")

	app.Use(func(next mizu.Handler) mizu.Handler {
	    return func(c *mizu.Ctx) error {
	        requestCount.Add(1)
	        return next(c)
	    }
	})

# Security

The expvar endpoint exposes internal application state. Consider:

  - Restricting access with authentication middleware
  - Using IP filtering to limit access
  - Disabling in production or using a different path

# See Also

  - Package pprof for profiling endpoints
  - Package prometheus for Prometheus metrics
  - Package metrics for custom metrics collection
*/
package expvar
