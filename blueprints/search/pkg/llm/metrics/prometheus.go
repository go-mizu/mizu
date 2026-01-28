package metrics

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

// Prometheus metrics (stub implementation - actual Prometheus integration optional)
// To enable full Prometheus support, add prometheus/client_golang dependency.

var (
	prometheusOnce     sync.Once
	prometheusEnabled  bool
	metricsData        = &inMemoryMetrics{}
)

// inMemoryMetrics stores metrics in memory for basic reporting.
type inMemoryMetrics struct {
	mu sync.RWMutex

	// Counters
	requestsTotal     map[string]int64   // key: provider:model:status
	tokensTotal       map[string]int64   // key: provider:model:direction
	costTotal         map[string]float64 // key: provider:model
	toolCallsTotal    map[string]int64   // key: provider:tool:status

	// Histograms (simplified as lists)
	requestDurations  []float64
	timeToFirstTokens []float64
}

func registerPrometheusMetrics() {
	prometheusOnce.Do(func() {
		metricsData = &inMemoryMetrics{
			requestsTotal:  make(map[string]int64),
			tokensTotal:    make(map[string]int64),
			costTotal:      make(map[string]float64),
			toolCallsTotal: make(map[string]int64),
		}
		prometheusEnabled = true
	})
}

func recordPrometheusMetrics(m llm.RequestMetrics) {
	if !prometheusEnabled {
		return
	}

	metricsData.mu.Lock()
	defer metricsData.mu.Unlock()

	status := "success"
	if !m.Success {
		status = "error"
	}

	// Request counter
	reqKey := m.Provider + ":" + m.Model + ":" + status
	metricsData.requestsTotal[reqKey]++

	// Token counters
	inputKey := m.Provider + ":" + m.Model + ":input"
	outputKey := m.Provider + ":" + m.Model + ":output"
	metricsData.tokensTotal[inputKey] += int64(m.InputTokens)
	metricsData.tokensTotal[outputKey] += int64(m.OutputTokens)

	if m.CacheReadTokens > 0 {
		cacheReadKey := m.Provider + ":" + m.Model + ":cache_read"
		metricsData.tokensTotal[cacheReadKey] += int64(m.CacheReadTokens)
	}
	if m.CacheWriteTokens > 0 {
		cacheWriteKey := m.Provider + ":" + m.Model + ":cache_write"
		metricsData.tokensTotal[cacheWriteKey] += int64(m.CacheWriteTokens)
	}

	// Cost counter
	if m.CostUSD > 0 {
		costKey := m.Provider + ":" + m.Model
		metricsData.costTotal[costKey] += m.CostUSD
	}

	// Duration histogram
	metricsData.requestDurations = append(metricsData.requestDurations, m.TotalDuration.Seconds())
	if len(metricsData.requestDurations) > 10000 {
		metricsData.requestDurations = metricsData.requestDurations[1:]
	}

	// Time to first token histogram
	if m.TimeToFirstToken > 0 {
		metricsData.timeToFirstTokens = append(metricsData.timeToFirstTokens, m.TimeToFirstToken.Seconds())
		if len(metricsData.timeToFirstTokens) > 10000 {
			metricsData.timeToFirstTokens = metricsData.timeToFirstTokens[1:]
		}
	}

	// Tool calls
	if m.ToolCalls > 0 {
		toolKey := m.Provider + ":all:" + status
		metricsData.toolCallsTotal[toolKey] += int64(m.ToolCalls)
	}
}

// MetricsSnapshot represents a point-in-time snapshot of metrics.
type MetricsSnapshot struct {
	RequestsTotal     map[string]int64   `json:"requests_total"`
	TokensTotal       map[string]int64   `json:"tokens_total"`
	CostTotal         map[string]float64 `json:"cost_total"`
	ToolCallsTotal    map[string]int64   `json:"tool_calls_total"`
	AvgDurationSec    float64            `json:"avg_duration_sec"`
	AvgTTFTSec        float64            `json:"avg_ttft_sec"`
	P50DurationSec    float64            `json:"p50_duration_sec"`
	P95DurationSec    float64            `json:"p95_duration_sec"`
	P99DurationSec    float64            `json:"p99_duration_sec"`
}

// GetMetricsSnapshot returns current metrics snapshot.
func GetMetricsSnapshot() *MetricsSnapshot {
	metricsData.mu.RLock()
	defer metricsData.mu.RUnlock()

	snapshot := &MetricsSnapshot{
		RequestsTotal:  make(map[string]int64),
		TokensTotal:    make(map[string]int64),
		CostTotal:      make(map[string]float64),
		ToolCallsTotal: make(map[string]int64),
	}

	// Copy counters
	for k, v := range metricsData.requestsTotal {
		snapshot.RequestsTotal[k] = v
	}
	for k, v := range metricsData.tokensTotal {
		snapshot.TokensTotal[k] = v
	}
	for k, v := range metricsData.costTotal {
		snapshot.CostTotal[k] = v
	}
	for k, v := range metricsData.toolCallsTotal {
		snapshot.ToolCallsTotal[k] = v
	}

	// Calculate duration statistics
	if len(metricsData.requestDurations) > 0 {
		snapshot.AvgDurationSec = average(metricsData.requestDurations)
		snapshot.P50DurationSec = percentile(metricsData.requestDurations, 50)
		snapshot.P95DurationSec = percentile(metricsData.requestDurations, 95)
		snapshot.P99DurationSec = percentile(metricsData.requestDurations, 99)
	}

	// Calculate TTFT statistics
	if len(metricsData.timeToFirstTokens) > 0 {
		snapshot.AvgTTFTSec = average(metricsData.timeToFirstTokens)
	}

	return snapshot
}

// MetricsHandler returns an HTTP handler for serving metrics.
// This returns a simple JSON endpoint. For Prometheus format, integrate
// with prometheus/client_golang directly.
func MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := GetMetricsSnapshot()

		w.Header().Set("Content-Type", "application/json")

		// Simple JSON output
		fmt := r.URL.Query().Get("format")
		if fmt == "prometheus" {
			// Output Prometheus text format
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			writePrometheusFormat(w, snapshot)
			return
		}

		// Default JSON
		writeJSON(w, snapshot)
	}
}

func writePrometheusFormat(w http.ResponseWriter, s *MetricsSnapshot) {
	// Write metrics in Prometheus exposition format
	w.Write([]byte("# HELP llm_requests_total Total LLM requests\n"))
	w.Write([]byte("# TYPE llm_requests_total counter\n"))
	for k, v := range s.RequestsTotal {
		parts := splitKey(k)
		if len(parts) == 3 {
			line := "llm_requests_total{provider=\"" + parts[0] + "\",model=\"" + parts[1] + "\",status=\"" + parts[2] + "\"} " + itoa(v) + "\n"
			w.Write([]byte(line))
		}
	}

	w.Write([]byte("\n# HELP llm_tokens_total Total tokens processed\n"))
	w.Write([]byte("# TYPE llm_tokens_total counter\n"))
	for k, v := range s.TokensTotal {
		parts := splitKey(k)
		if len(parts) == 3 {
			line := "llm_tokens_total{provider=\"" + parts[0] + "\",model=\"" + parts[1] + "\",direction=\"" + parts[2] + "\"} " + itoa(v) + "\n"
			w.Write([]byte(line))
		}
	}

	w.Write([]byte("\n# HELP llm_cost_usd_total Total cost in USD\n"))
	w.Write([]byte("# TYPE llm_cost_usd_total counter\n"))
	for k, v := range s.CostTotal {
		parts := splitKey(k)
		if len(parts) == 2 {
			line := "llm_cost_usd_total{provider=\"" + parts[0] + "\",model=\"" + parts[1] + "\"} " + ftoa(v) + "\n"
			w.Write([]byte(line))
		}
	}

	w.Write([]byte("\n# HELP llm_request_duration_seconds Request duration in seconds\n"))
	w.Write([]byte("# TYPE llm_request_duration_seconds summary\n"))
	w.Write([]byte("llm_request_duration_seconds{quantile=\"0.5\"} " + ftoa(s.P50DurationSec) + "\n"))
	w.Write([]byte("llm_request_duration_seconds{quantile=\"0.95\"} " + ftoa(s.P95DurationSec) + "\n"))
	w.Write([]byte("llm_request_duration_seconds{quantile=\"0.99\"} " + ftoa(s.P99DurationSec) + "\n"))
}

func writeJSON(w http.ResponseWriter, s *MetricsSnapshot) {
	w.Write([]byte("{"))
	w.Write([]byte("\"requests_total\":{"))
	first := true
	for k, v := range s.RequestsTotal {
		if !first {
			w.Write([]byte(","))
		}
		w.Write([]byte("\"" + k + "\":" + itoa(v)))
		first = false
	}
	w.Write([]byte("},"))

	w.Write([]byte("\"tokens_total\":{"))
	first = true
	for k, v := range s.TokensTotal {
		if !first {
			w.Write([]byte(","))
		}
		w.Write([]byte("\"" + k + "\":" + itoa(v)))
		first = false
	}
	w.Write([]byte("},"))

	w.Write([]byte("\"cost_total\":{"))
	first = true
	for k, v := range s.CostTotal {
		if !first {
			w.Write([]byte(","))
		}
		w.Write([]byte("\"" + k + "\":" + ftoa(v)))
		first = false
	}
	w.Write([]byte("},"))

	w.Write([]byte("\"avg_duration_sec\":" + ftoa(s.AvgDurationSec) + ","))
	w.Write([]byte("\"avg_ttft_sec\":" + ftoa(s.AvgTTFTSec) + ","))
	w.Write([]byte("\"p50_duration_sec\":" + ftoa(s.P50DurationSec) + ","))
	w.Write([]byte("\"p95_duration_sec\":" + ftoa(s.P95DurationSec) + ","))
	w.Write([]byte("\"p99_duration_sec\":" + ftoa(s.P99DurationSec)))
	w.Write([]byte("}"))
}

// Helper functions

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	idx := int(float64(len(sorted)-1) * float64(p) / 100)
	return sorted[idx]
}

func splitKey(key string) []string {
	return strings.Split(key, ":")
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}

func ftoa(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}
