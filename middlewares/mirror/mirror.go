// Package mirror provides request mirroring middleware for Mizu.
package mirror

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

// Target represents a mirror target.
type Target struct {
	// URL is the target URL to mirror requests to.
	URL string

	// Percentage is the percentage of requests to mirror.
	// Default: 100.
	Percentage int
}

// Options configures the mirror middleware.
type Options struct {
	// Targets is the list of mirror targets.
	Targets []Target

	// Timeout is the timeout for mirrored requests.
	// Default: 5s.
	Timeout time.Duration

	// Async runs mirror requests asynchronously.
	// Default: true.
	Async bool

	// CopyBody copies request body for mirroring.
	// Default: true.
	CopyBody bool

	// OnError is called when a mirror request fails.
	OnError func(target string, err error)

	// OnSuccess is called when a mirror request succeeds.
	OnSuccess func(target string, resp *http.Response)
}

// New creates mirror middleware with target URLs.
func New(targets ...string) mizu.Middleware {
	t := make([]Target, len(targets))
	for i, url := range targets {
		t[i] = Target{URL: url, Percentage: 100}
	}
	return WithOptions(Options{Targets: t})
}

// WithOptions creates mirror middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}

	client := &http.Client{
		Timeout: opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var counter uint64

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()

			// Read and store body if needed
			var bodyBytes []byte
			if opts.CopyBody && r.Body != nil && r.ContentLength > 0 {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			// Mirror to targets
			for _, target := range opts.Targets {
				// Check percentage
				if target.Percentage < 100 {
					counter++
					if int(counter%100) >= target.Percentage {
						continue
					}
				}

				if opts.Async {
					go mirrorRequest(client, target.URL, r, bodyBytes, opts)
				} else {
					mirrorRequest(client, target.URL, r, bodyBytes, opts)
				}
			}

			return next(c)
		}
	}
}

func mirrorRequest(client *http.Client, target string, r *http.Request, body []byte, opts Options) {
	targetURL := target + r.URL.RequestURI()

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(r.Method, targetURL, bodyReader)
	if err != nil {
		if opts.OnError != nil {
			opts.OnError(target, err)
		}
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add mirroring header
	req.Header.Set("X-Mirrored-From", r.Host)

	resp, err := client.Do(req)
	if err != nil {
		if opts.OnError != nil {
			opts.OnError(target, err)
		}
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if opts.OnSuccess != nil {
		opts.OnSuccess(target, resp)
	}
}

// Percentage creates a target with specified percentage.
func Percentage(url string, pct int) Target {
	return Target{URL: url, Percentage: pct}
}
