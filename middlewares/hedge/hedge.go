// Package hedge provides hedged request middleware for Mizu.
// Hedged requests send multiple concurrent requests and return the fastest response.
package hedge

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the hedge middleware.
type Options struct {
	// Delay is how long to wait before starting hedged request.
	// Default: 100ms.
	Delay time.Duration

	// MaxHedges is the maximum number of hedged requests.
	// Default: 1 (original + 1 hedge).
	MaxHedges int

	// Timeout is the overall timeout for all requests.
	// Default: 30s.
	Timeout time.Duration

	// ShouldHedge determines if a request should be hedged.
	// Default: all requests.
	ShouldHedge func(r *http.Request) bool

	// OnHedge is called when a hedge is triggered.
	OnHedge func(hedgeNum int)

	// OnComplete is called when the fastest response returns.
	OnComplete func(hedgeNum int, duration time.Duration)
}

// contextKey is a private type for context keys.
type contextKey struct{}

// hedgeKey stores hedge info.
var hedgeKey = contextKey{}

// HedgeInfo contains information about hedging.
type HedgeInfo struct {
	HedgeNumber int
	TotalHedges int
	Winner      int
	Duration    time.Duration
}

// Stats tracks hedging statistics.
type Stats struct {
	TotalRequests   uint64
	HedgedRequests  uint64
	HedgesTriggered uint64
	WinsByOriginal  uint64
	WinsByHedge     uint64
}

// Hedger manages hedged requests.
type Hedger struct {
	opts  Options
	stats Stats
}

// New creates hedge middleware with default options.
func New() mizu.Middleware {
	return NewHedger(Options{}).Middleware()
}

// WithOptions creates hedge middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return NewHedger(opts).Middleware()
}

// NewHedger creates a new hedger.
func NewHedger(opts Options) *Hedger {
	if opts.Delay == 0 {
		opts.Delay = 100 * time.Millisecond
	}
	if opts.MaxHedges == 0 {
		opts.MaxHedges = 1
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	return &Hedger{opts: opts}
}

// Middleware returns the Mizu middleware.
//
//nolint:cyclop // Hedged request handling requires multiple timing and coordination checks
func (h *Hedger) Middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			atomic.AddUint64(&h.stats.TotalRequests, 1)

			// Check if should hedge
			if h.opts.ShouldHedge != nil && !h.opts.ShouldHedge(c.Request()) {
				return next(c)
			}

			atomic.AddUint64(&h.stats.HedgedRequests, 1)

			// Create context with timeout
			ctx, cancel := context.WithTimeout(c.Request().Context(), h.opts.Timeout)
			defer cancel()

			// Results channel
			type result struct {
				hedgeNum int
				recorder *responseRecorder
				err      error
			}

			results := make(chan result, h.opts.MaxHedges+1)
			start := time.Now()

			// Read and buffer request body
			var bodyBuf []byte
			if c.Request().Body != nil {
				bodyBuf, _ = io.ReadAll(c.Request().Body)
				_ = c.Request().Body.Close()
			}

			// Create work group
			var wg sync.WaitGroup
			var winnerSet int32

			// Runner function
			runRequest := func(hedgeNum int) {
				defer wg.Done()

				// Check if we already have a winner
				if atomic.LoadInt32(&winnerSet) == 1 {
					return
				}

				// Create recorder
				rec := &responseRecorder{
					ResponseWriter: c.Writer(),
					body:           &bytes.Buffer{},
					statusCode:     http.StatusOK,
					headers:        make(http.Header),
				}

				// Store hedge info in context
				hedgeInfo := &HedgeInfo{
					HedgeNumber: hedgeNum,
					TotalHedges: h.opts.MaxHedges + 1,
				}
				ctx := context.WithValue(c.Context(), hedgeKey, hedgeInfo)
				req := c.Request().WithContext(ctx)
				if bodyBuf != nil {
					req.Body = io.NopCloser(bytes.NewReader(bodyBuf))
				}
				*c.Request() = *req

				// Set the recorder as the writer
				origWriter := c.Writer()
				c.SetWriter(rec)

				err := next(c)

				// Restore writer
				c.SetWriter(origWriter)

				// Try to be the winner
				if atomic.CompareAndSwapInt32(&winnerSet, 0, 1) {
					results <- result{
						hedgeNum: hedgeNum,
						recorder: rec,
						err:      err,
					}
				}
			}

			// Start original request
			wg.Add(1)
			go runRequest(0)

			// Start hedged requests after delay
			for i := 1; i <= h.opts.MaxHedges; i++ {
				hedgeNum := i
				go func() {
					select {
					case <-ctx.Done():
						return
					case <-time.After(h.opts.Delay * time.Duration(hedgeNum)):
						if atomic.LoadInt32(&winnerSet) == 0 {
							atomic.AddUint64(&h.stats.HedgesTriggered, 1)
							if h.opts.OnHedge != nil {
								h.opts.OnHedge(hedgeNum)
							}
							wg.Add(1)
							go runRequest(hedgeNum)
						}
					}
				}()
			}

			// Wait for first result
			select {
			case res := <-results:
				duration := time.Since(start)

				// Track winner stats
				if res.hedgeNum == 0 {
					atomic.AddUint64(&h.stats.WinsByOriginal, 1)
				} else {
					atomic.AddUint64(&h.stats.WinsByHedge, 1)
				}

				if h.opts.OnComplete != nil {
					h.opts.OnComplete(res.hedgeNum, duration)
				}

				// Write response
				for k, v := range res.recorder.headers {
					for _, vv := range v {
						c.Header().Add(k, vv)
					}
				}
				c.Writer().WriteHeader(res.recorder.statusCode)
				_, _ = c.Writer().Write(res.recorder.body.Bytes())

				return res.err

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Stats returns hedging statistics.
func (h *Hedger) Stats() Stats {
	return Stats{
		TotalRequests:   atomic.LoadUint64(&h.stats.TotalRequests),
		HedgedRequests:  atomic.LoadUint64(&h.stats.HedgedRequests),
		HedgesTriggered: atomic.LoadUint64(&h.stats.HedgesTriggered),
		WinsByOriginal:  atomic.LoadUint64(&h.stats.WinsByOriginal),
		WinsByHedge:     atomic.LoadUint64(&h.stats.WinsByHedge),
	}
}

type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	headers    http.Header
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
}

// GetHedgeInfo returns hedge information from context.
func GetHedgeInfo(c *mizu.Ctx) *HedgeInfo {
	if info, ok := c.Context().Value(hedgeKey).(*HedgeInfo); ok {
		return info
	}
	return nil
}

// IsHedge returns true if this is a hedged request (not the original).
func IsHedge(c *mizu.Ctx) bool {
	info := GetHedgeInfo(c)
	return info != nil && info.HedgeNumber > 0
}

// Conditional creates middleware that only hedges based on a condition.
func Conditional(shouldHedge func(r *http.Request) bool) mizu.Middleware {
	return WithOptions(Options{ShouldHedge: shouldHedge})
}

// ForSlowRequests creates middleware that hedges slow requests.
func ForSlowRequests(threshold time.Duration) mizu.Middleware {
	return WithOptions(Options{
		Delay: threshold,
	})
}
