// Package prometheus provides Prometheus metrics middleware for Mizu.
package prometheus

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the prometheus middleware.
type Options struct {
	// Namespace is the metric namespace prefix.
	Namespace string

	// Subsystem is the metric subsystem.
	Subsystem string

	// MetricsPath is the path for metrics endpoint.
	// Default: "/metrics".
	MetricsPath string

	// SkipPaths are paths to skip from metrics.
	SkipPaths []string

	// Buckets are histogram buckets for request duration.
	// Default: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10].
	Buckets []float64

	// RequestCountName is the name for request counter metric.
	// Default: "http_requests_total".
	RequestCountName string

	// RequestDurationName is the name for request duration histogram.
	// Default: "http_request_duration_seconds".
	RequestDurationName string

	// RequestSizeName is the name for request size histogram.
	// Default: "http_request_size_bytes".
	RequestSizeName string

	// ResponseSizeName is the name for response size histogram.
	// Default: "http_response_size_bytes".
	ResponseSizeName string
}

// Metrics holds all Prometheus metrics.
type Metrics struct {
	opts Options

	mu sync.RWMutex

	// Request counters by method, path, status
	requestCount map[string]*uint64

	// Request durations
	requestDurations map[string]*histogram

	// Request sizes
	requestSizes map[string]*histogram

	// Response sizes
	responseSizes map[string]*histogram

	// Active requests gauge
	activeRequests int64

	// Total requests
	totalRequests uint64
}

type histogram struct {
	buckets []float64
	counts  []uint64
	sum     float64
	count   uint64
	mu      sync.Mutex
}

func newHistogram(buckets []float64) *histogram {
	return &histogram{
		buckets: buckets,
		counts:  make([]uint64, len(buckets)+1),
	}
}

func (h *histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sum += v
	h.count++

	for i, b := range h.buckets {
		if v <= b {
			h.counts[i]++
			return
		}
	}
	h.counts[len(h.buckets)]++
}

var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// NewMetrics creates a new metrics instance.
func NewMetrics(opts Options) *Metrics {
	if opts.MetricsPath == "" {
		opts.MetricsPath = "/metrics"
	}
	if opts.RequestCountName == "" {
		opts.RequestCountName = "http_requests_total"
	}
	if opts.RequestDurationName == "" {
		opts.RequestDurationName = "http_request_duration_seconds"
	}
	if opts.RequestSizeName == "" {
		opts.RequestSizeName = "http_request_size_bytes"
	}
	if opts.ResponseSizeName == "" {
		opts.ResponseSizeName = "http_response_size_bytes"
	}
	if len(opts.Buckets) == 0 {
		opts.Buckets = defaultBuckets
	}

	return &Metrics{
		opts:             opts,
		requestCount:     make(map[string]*uint64),
		requestDurations: make(map[string]*histogram),
		requestSizes:     make(map[string]*histogram),
		responseSizes:    make(map[string]*histogram),
	}
}

// New creates prometheus middleware with default options.
func New() mizu.Middleware {
	return NewMetrics(Options{}).Middleware()
}

// WithOptions creates prometheus middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return NewMetrics(opts).Middleware()
}

// Middleware returns the Mizu middleware.
func (m *Metrics) Middleware() mizu.Middleware {
	skipPaths := make(map[string]bool)
	for _, p := range m.opts.SkipPaths {
		skipPaths[p] = true
	}
	skipPaths[m.opts.MetricsPath] = true

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			// Skip metrics path and configured skip paths
			if skipPaths[path] {
				return next(c)
			}

			start := time.Now()
			atomic.AddInt64(&m.activeRequests, 1)
			atomic.AddUint64(&m.totalRequests, 1)

			// Wrap response writer to capture status and size
			rw := &responseWriter{
				ResponseWriter: c.Writer(),
				statusCode:     http.StatusOK,
			}
			c.SetWriter(rw)

			err := next(c)

			atomic.AddInt64(&m.activeRequests, -1)

			duration := time.Since(start).Seconds()
			method := c.Request().Method
			status := strconv.Itoa(rw.statusCode)

			// Record metrics
			m.recordRequest(method, path, status)
			m.recordDuration(method, path, status, duration)
			m.recordRequestSize(method, path, float64(c.Request().ContentLength))
			m.recordResponseSize(method, path, float64(rw.size))

			return err
		}
	}
}

// Handler returns a handler for the metrics endpoint.
func (m *Metrics) Handler() mizu.Handler {
	return func(c *mizu.Ctx) error {
		c.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.Writer().WriteHeader(http.StatusOK)

		output := m.Export()
		c.Writer().Write([]byte(output))
		return nil
	}
}

// Export exports metrics in Prometheus format.
func (m *Metrics) Export() string {
	var sb strings.Builder

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Full metric name
	countName := m.fullName(m.opts.RequestCountName)
	durationName := m.fullName(m.opts.RequestDurationName)
	requestSizeName := m.fullName(m.opts.RequestSizeName)
	responseSizeName := m.fullName(m.opts.ResponseSizeName)

	// Request count
	sb.WriteString(fmt.Sprintf("# HELP %s Total number of HTTP requests.\n", countName))
	sb.WriteString(fmt.Sprintf("# TYPE %s counter\n", countName))

	keys := make([]string, 0, len(m.requestCount))
	for k := range m.requestCount {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		count := atomic.LoadUint64(m.requestCount[k])
		sb.WriteString(fmt.Sprintf("%s%s %d\n", countName, k, count))
	}

	// Request duration histogram
	sb.WriteString(fmt.Sprintf("\n# HELP %s HTTP request duration in seconds.\n", durationName))
	sb.WriteString(fmt.Sprintf("# TYPE %s histogram\n", durationName))

	for labels, h := range m.requestDurations {
		m.writeHistogram(&sb, durationName, labels, h)
	}

	// Request size histogram
	sb.WriteString(fmt.Sprintf("\n# HELP %s HTTP request size in bytes.\n", requestSizeName))
	sb.WriteString(fmt.Sprintf("# TYPE %s histogram\n", requestSizeName))

	for labels, h := range m.requestSizes {
		m.writeHistogram(&sb, requestSizeName, labels, h)
	}

	// Response size histogram
	sb.WriteString(fmt.Sprintf("\n# HELP %s HTTP response size in bytes.\n", responseSizeName))
	sb.WriteString(fmt.Sprintf("# TYPE %s histogram\n", responseSizeName))

	for labels, h := range m.responseSizes {
		m.writeHistogram(&sb, responseSizeName, labels, h)
	}

	// Active requests gauge
	activeGaugeName := m.fullName("http_requests_active")
	sb.WriteString(fmt.Sprintf("\n# HELP %s Current number of active HTTP requests.\n", activeGaugeName))
	sb.WriteString(fmt.Sprintf("# TYPE %s gauge\n", activeGaugeName))
	sb.WriteString(fmt.Sprintf("%s %d\n", activeGaugeName, atomic.LoadInt64(&m.activeRequests)))

	return sb.String()
}

func (m *Metrics) fullName(name string) string {
	if m.opts.Namespace != "" && m.opts.Subsystem != "" {
		return fmt.Sprintf("%s_%s_%s", m.opts.Namespace, m.opts.Subsystem, name)
	}
	if m.opts.Namespace != "" {
		return fmt.Sprintf("%s_%s", m.opts.Namespace, name)
	}
	if m.opts.Subsystem != "" {
		return fmt.Sprintf("%s_%s", m.opts.Subsystem, name)
	}
	return name
}

func (m *Metrics) writeHistogram(sb *strings.Builder, name, labels string, h *histogram) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var cumulative uint64
	baseLabels := strings.TrimSuffix(strings.TrimPrefix(labels, "{"), "}")

	for i, b := range h.buckets {
		cumulative += h.counts[i]
		le := strconv.FormatFloat(b, 'f', -1, 64)
		if baseLabels != "" {
			sb.WriteString(fmt.Sprintf("%s_bucket{%s,le=\"%s\"} %d\n", name, baseLabels, le, cumulative))
		} else {
			sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%s\"} %d\n", name, le, cumulative))
		}
	}

	cumulative += h.counts[len(h.buckets)]
	if baseLabels != "" {
		sb.WriteString(fmt.Sprintf("%s_bucket{%s,le=\"+Inf\"} %d\n", name, baseLabels, cumulative))
		sb.WriteString(fmt.Sprintf("%s_sum{%s} %f\n", name, baseLabels, h.sum))
		sb.WriteString(fmt.Sprintf("%s_count{%s} %d\n", name, baseLabels, h.count))
	} else {
		sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", name, cumulative))
		sb.WriteString(fmt.Sprintf("%s_sum %f\n", name, h.sum))
		sb.WriteString(fmt.Sprintf("%s_count %d\n", name, h.count))
	}
}

func (m *Metrics) recordRequest(method, path, status string) {
	labels := fmt.Sprintf(`{method="%s",path="%s",status="%s"}`, method, path, status)

	m.mu.Lock()
	if m.requestCount[labels] == nil {
		var zero uint64
		m.requestCount[labels] = &zero
	}
	m.mu.Unlock()

	atomic.AddUint64(m.requestCount[labels], 1)
}

func (m *Metrics) recordDuration(method, path, status string, duration float64) {
	labels := fmt.Sprintf(`{method="%s",path="%s",status="%s"}`, method, path, status)

	m.mu.Lock()
	if m.requestDurations[labels] == nil {
		m.requestDurations[labels] = newHistogram(m.opts.Buckets)
	}
	h := m.requestDurations[labels]
	m.mu.Unlock()

	h.Observe(duration)
}

func (m *Metrics) recordRequestSize(method, path string, size float64) {
	if size <= 0 {
		return
	}

	labels := fmt.Sprintf(`{method="%s",path="%s"}`, method, path)

	m.mu.Lock()
	if m.requestSizes[labels] == nil {
		m.requestSizes[labels] = newHistogram([]float64{100, 1000, 10000, 100000, 1000000})
	}
	h := m.requestSizes[labels]
	m.mu.Unlock()

	h.Observe(size)
}

func (m *Metrics) recordResponseSize(method, path string, size float64) {
	if size <= 0 {
		return
	}

	labels := fmt.Sprintf(`{method="%s",path="%s"}`, method, path)

	m.mu.Lock()
	if m.responseSizes[labels] == nil {
		m.responseSizes[labels] = newHistogram([]float64{100, 1000, 10000, 100000, 1000000})
	}
	h := m.responseSizes[labels]
	m.mu.Unlock()

	h.Observe(size)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// TotalRequests returns the total number of requests.
func (m *Metrics) TotalRequests() uint64 {
	return atomic.LoadUint64(&m.totalRequests)
}

// ActiveRequests returns the current number of active requests.
func (m *Metrics) ActiveRequests() int64 {
	return atomic.LoadInt64(&m.activeRequests)
}

// RegisterEndpoint registers the metrics endpoint on the router.
func (m *Metrics) RegisterEndpoint(r *mizu.Router) {
	r.Get(m.opts.MetricsPath, m.Handler())
}
