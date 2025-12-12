// Package metrics provides simple metrics middleware for Mizu.
package metrics

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu"
)

// Metrics holds request metrics.
type Metrics struct {
	RequestCount   int64
	ErrorCount     int64
	TotalDuration  int64 // nanoseconds
	ActiveRequests int64

	statusCodes map[int]*int64
	pathCounts  map[string]*int64
	mu          sync.RWMutex
}

// NewMetrics creates a new metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		statusCodes: make(map[int]*int64),
		pathCounts:  make(map[string]*int64),
	}
}

// Middleware returns a metrics middleware.
func (m *Metrics) Middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			start := time.Now()

			atomic.AddInt64(&m.ActiveRequests, 1)
			defer atomic.AddInt64(&m.ActiveRequests, -1)

			atomic.AddInt64(&m.RequestCount, 1)

			// Track path
			m.incrementPath(c.Request().URL.Path)

			// Capture status code
			capture := &statusCapture{ResponseWriter: c.Writer(), statusCode: http.StatusOK}
			c.SetWriter(capture)

			err := next(c)

			// Restore writer
			c.SetWriter(capture.ResponseWriter)

			// Track duration
			duration := time.Since(start).Nanoseconds()
			atomic.AddInt64(&m.TotalDuration, duration)

			// Track status code
			m.incrementStatus(capture.statusCode)

			// Track errors
			if err != nil || capture.statusCode >= 400 {
				atomic.AddInt64(&m.ErrorCount, 1)
			}

			return err
		}
	}
}

type statusCapture struct {
	http.ResponseWriter
	statusCode int
}

func (s *statusCapture) WriteHeader(code int) {
	s.statusCode = code
	s.ResponseWriter.WriteHeader(code)
}

func (m *Metrics) incrementStatus(status int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.statusCodes[status] == nil {
		m.statusCodes[status] = new(int64)
	}
	atomic.AddInt64(m.statusCodes[status], 1)
}

func (m *Metrics) incrementPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pathCounts[path] == nil {
		m.pathCounts[path] = new(int64)
	}
	atomic.AddInt64(m.pathCounts[path], 1)
}

// Stats returns current statistics.
func (m *Metrics) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statusCodes := make(map[int]int64)
	for code, count := range m.statusCodes {
		statusCodes[code] = atomic.LoadInt64(count)
	}

	pathCounts := make(map[string]int64)
	for path, count := range m.pathCounts {
		pathCounts[path] = atomic.LoadInt64(count)
	}

	reqCount := atomic.LoadInt64(&m.RequestCount)
	totalDur := atomic.LoadInt64(&m.TotalDuration)

	var avgDuration float64
	if reqCount > 0 {
		avgDuration = float64(totalDur) / float64(reqCount) / 1e6 // milliseconds
	}

	return Stats{
		RequestCount:      reqCount,
		ErrorCount:        atomic.LoadInt64(&m.ErrorCount),
		ActiveRequests:    atomic.LoadInt64(&m.ActiveRequests),
		AverageDurationMs: avgDuration,
		StatusCodes:       statusCodes,
		PathCounts:        pathCounts,
	}
}

// Stats contains metric statistics.
type Stats struct {
	RequestCount      int64            `json:"request_count"`
	ErrorCount        int64            `json:"error_count"`
	ActiveRequests    int64            `json:"active_requests"`
	AverageDurationMs float64          `json:"average_duration_ms"`
	StatusCodes       map[int]int64    `json:"status_codes"`
	PathCounts        map[string]int64 `json:"path_counts"`
}

// Handler returns a handler that exposes metrics as JSON.
func (m *Metrics) Handler() mizu.Handler {
	return func(c *mizu.Ctx) error {
		stats := m.Stats()
		return c.JSON(http.StatusOK, stats)
	}
}

// Reset resets all metrics.
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	atomic.StoreInt64(&m.RequestCount, 0)
	atomic.StoreInt64(&m.ErrorCount, 0)
	atomic.StoreInt64(&m.TotalDuration, 0)
	m.statusCodes = make(map[int]*int64)
	m.pathCounts = make(map[string]*int64)
}

// New creates metrics middleware with a new metrics instance.
func New() (*Metrics, mizu.Middleware) {
	m := NewMetrics()
	return m, m.Middleware()
}

// Prometheus returns metrics in Prometheus format.
func (m *Metrics) Prometheus() mizu.Handler {
	return func(c *mizu.Ctx) error {
		stats := m.Stats()

		var output string
		output += "# HELP http_requests_total Total HTTP requests\n"
		output += "# TYPE http_requests_total counter\n"
		output += "http_requests_total " + itoa(stats.RequestCount) + "\n"

		output += "# HELP http_errors_total Total HTTP errors\n"
		output += "# TYPE http_errors_total counter\n"
		output += "http_errors_total " + itoa(stats.ErrorCount) + "\n"

		output += "# HELP http_active_requests Current active requests\n"
		output += "# TYPE http_active_requests gauge\n"
		output += "http_active_requests " + itoa(stats.ActiveRequests) + "\n"

		c.Header().Set("Content-Type", "text/plain; version=0.0.4")
		return c.Text(http.StatusOK, output)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// JSON returns metrics as JSON bytes.
func (m *Metrics) JSON() ([]byte, error) {
	return json.Marshal(m.Stats())
}
