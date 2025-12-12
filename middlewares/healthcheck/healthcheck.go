// Package healthcheck provides health check endpoint middleware for Mizu.
package healthcheck

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Check defines a health check.
type Check struct {
	Name    string
	Check   func(ctx context.Context) error
	Timeout time.Duration
}

// Status is the health check response.
type Status struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Options configures the health check endpoints.
type Options struct {
	// LivenessPath is the path for liveness probe.
	// Default: "/healthz".
	LivenessPath string

	// ReadinessPath is the path for readiness probe.
	// Default: "/readyz".
	ReadinessPath string

	// Checks is a list of health checks for readiness.
	Checks []Check
}

// New creates a health check handler.
func New(opts Options) mizu.Handler {
	if opts.LivenessPath == "" {
		opts.LivenessPath = "/healthz"
	}
	if opts.ReadinessPath == "" {
		opts.ReadinessPath = "/readyz"
	}

	return func(c *mizu.Ctx) error {
		path := c.Request().URL.Path
		if path == opts.LivenessPath {
			return liveness(c)
		}
		if path == opts.ReadinessPath {
			return readiness(c, opts.Checks)
		}
		return c.Text(http.StatusNotFound, "Not Found")
	}
}

// Liveness creates a simple liveness probe handler.
func Liveness() mizu.Handler {
	return liveness
}

func liveness(c *mizu.Ctx) error {
	return c.Text(http.StatusOK, "ok")
}

// Readiness creates a readiness probe with checks.
func Readiness(checks ...Check) mizu.Handler {
	return func(c *mizu.Ctx) error {
		return readiness(c, checks)
	}
}

func readiness(c *mizu.Ctx, checks []Check) error {
	if len(checks) == 0 {
		return c.Text(http.StatusOK, "ok")
	}

	ctx := c.Context()
	results := make(map[string]string)
	healthy := true

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, check := range checks {
		wg.Add(1)
		go func(ch Check) {
			defer wg.Done()

			timeout := ch.Timeout
			if timeout == 0 {
				timeout = 5 * time.Second
			}

			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			err := ch.Check(checkCtx)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results[ch.Name] = err.Error()
				healthy = false
			} else {
				results[ch.Name] = "ok"
			}
		}(check)
	}

	wg.Wait()

	status := Status{
		Status: "ok",
		Checks: results,
	}

	statusCode := http.StatusOK
	if !healthy {
		status.Status = "error"
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, status)
}

// Register registers health check routes on a router.
func Register(r *mizu.Router, opts Options) {
	if opts.LivenessPath == "" {
		opts.LivenessPath = "/healthz"
	}
	if opts.ReadinessPath == "" {
		opts.ReadinessPath = "/readyz"
	}

	r.Get(opts.LivenessPath, Liveness())
	r.Get(opts.ReadinessPath, Readiness(opts.Checks...))
}

// DBCheck creates a database ping check.
func DBCheck(name string, pingFunc func(ctx context.Context) error) Check {
	return Check{
		Name:    name,
		Check:   pingFunc,
		Timeout: 5 * time.Second,
	}
}

// HTTPCheck creates an HTTP endpoint check.
func HTTPCheck(name, url string) Check {
	return Check{
		Name: name,
		Check: func(ctx context.Context) error {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				return http.ErrNotSupported
			}
			return nil
		},
		Timeout: 10 * time.Second,
	}
}
