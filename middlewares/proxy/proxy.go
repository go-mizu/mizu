// Package proxy provides reverse proxy middleware for Mizu.
package proxy

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the proxy middleware.
type Options struct {
	// Target is the upstream URL.
	Target *url.URL

	// Rewrite is a function to rewrite the request path.
	Rewrite func(path string) string

	// ModifyRequest allows modifying the proxy request.
	ModifyRequest func(req *http.Request)

	// ModifyResponse allows modifying the proxy response.
	ModifyResponse func(resp *http.Response) error

	// Transport is the HTTP transport to use.
	// Default: http.DefaultTransport.
	Transport http.RoundTripper

	// Timeout is the request timeout.
	// Default: 30s.
	Timeout time.Duration

	// PreserveHost preserves the original Host header.
	// Default: false.
	PreserveHost bool

	// ErrorHandler handles proxy errors.
	ErrorHandler func(c *mizu.Ctx, err error) error
}

// New creates proxy middleware with target URL string.
func New(target string) mizu.Middleware {
	u, err := url.Parse(target)
	if err != nil {
		panic("proxy: invalid target URL: " + err.Error())
	}
	return WithOptions(Options{Target: u})
}

// WithOptions creates proxy middleware with custom options.
//
//nolint:cyclop // Proxy handling requires multiple header and rewrite checks
func WithOptions(opts Options) mizu.Middleware {
	if opts.Target == nil {
		panic("proxy: target is required")
	}
	if opts.Transport == nil {
		opts.Transport = http.DefaultTransport
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	client := &http.Client{
		Transport: opts.Transport,
		Timeout:   opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()

			// Build target URL
			targetURL := *opts.Target
			path := r.URL.Path
			if opts.Rewrite != nil {
				path = opts.Rewrite(path)
			}
			targetURL.Path = singleJoiningSlash(targetURL.Path, path)
			targetURL.RawQuery = r.URL.RawQuery

			// Create proxy request
			proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
			if err != nil {
				return handleError(c, opts, err)
			}

			// Copy headers
			copyHeaders(proxyReq.Header, r.Header)

			// Set X-Forwarded headers
			if clientIP := r.RemoteAddr; clientIP != "" {
				if prior := proxyReq.Header.Get("X-Forwarded-For"); prior != "" {
					clientIP = prior + ", " + clientIP
				}
				proxyReq.Header.Set("X-Forwarded-For", clientIP)
			}

			proxyReq.Header.Set("X-Forwarded-Host", r.Host)

			if r.TLS != nil {
				proxyReq.Header.Set("X-Forwarded-Proto", "https")
			} else {
				proxyReq.Header.Set("X-Forwarded-Proto", "http")
			}

			// Handle Host header
			if opts.PreserveHost {
				proxyReq.Host = r.Host
			} else {
				proxyReq.Host = opts.Target.Host
			}

			// Allow request modification
			if opts.ModifyRequest != nil {
				opts.ModifyRequest(proxyReq)
			}

			// Send request
			resp, err := client.Do(proxyReq)
			if err != nil {
				return handleError(c, opts, err)
			}
			defer func() { _ = resp.Body.Close() }()

			// Allow response modification
			if opts.ModifyResponse != nil {
				if err := opts.ModifyResponse(resp); err != nil {
					return handleError(c, opts, err)
				}
			}

			// Copy response headers
			copyHeaders(c.Writer().Header(), resp.Header)

			// Write status code
			c.Writer().WriteHeader(resp.StatusCode)

			// Copy body
			_, err = io.Copy(c.Writer(), resp.Body)
			return err
		}
	}
}

// Balancer creates a load-balanced proxy middleware.
func Balancer(targets []string) mizu.Middleware {
	if len(targets) == 0 {
		panic("proxy: at least one target is required")
	}

	urls := make([]*url.URL, len(targets))
	for i, target := range targets {
		u, err := url.Parse(target)
		if err != nil {
			panic("proxy: invalid target URL: " + err.Error())
		}
		urls[i] = u
	}

	var counter uint64

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Round-robin selection
			idx := counter % uint64(len(urls))
			counter++

			return WithOptions(Options{Target: urls[idx]})(next)(c)
		}
	}
}

func handleError(c *mizu.Ctx, opts Options, err error) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c, err)
	}
	return c.Text(http.StatusBadGateway, "Bad Gateway")
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
