package crawl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

)

// benchURLsPerRun is the number of URLs processed per benchmark iteration.
const benchURLsPerRun = 200

// benchDomains is the number of distinct fake domains to spread seeds across,
// allowing domain-affine engines (KeepAlive) to exercise parallel domain workers.
const benchDomains = 10

// makeBenchSeeds creates n seed URLs spread across domains fake domains,
// all pointing to srv (so there is no real network latency).
func makeBenchSeeds(srv *httptest.Server, n, domains int) []SeedURL {
	seeds := make([]SeedURL, n)
	for i := range seeds {
		dom := fmt.Sprintf("d%d.bench.localhost", i%domains)
		seeds[i] = SeedURL{
			URL:    fmt.Sprintf("%s/%s/%d", srv.URL, dom, i),
			Domain: dom,
			Host:   "localhost",
		}
	}
	return seeds
}

// benchmarkEngine is a shared helper that drives any Engine through b.N iterations.
// Each iteration processes benchURLsPerRun URLs; throughput is reported as urls/s.
func benchmarkEngine(b *testing.B, eng Engine) {
	b.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	seeds := makeBenchSeeds(srv, benchURLsPerRun, benchDomains)

	cfg := DefaultConfig()
	cfg.Workers = 50
	cfg.Timeout = 5 * time.Second
	cfg.InsecureTLS = false
	cfg.StatusOnly = true

	b.ResetTimer()
	b.ReportAllocs()

	start := time.Now()
	var totalURLs int64

	for range b.N {
		stats, err := eng.Run(context.Background(), seeds, &NoopDNS{}, cfg,
			&noopResultWriter{}, &noopFailureWriter{})
		if err != nil {
			b.Fatalf("Run failed: %v", err)
		}
		totalURLs += stats.Total
	}

	elapsed := time.Since(start)
	if elapsed.Seconds() > 0 {
		b.ReportMetric(float64(totalURLs)/elapsed.Seconds(), "urls/s")
	}
}

func BenchmarkEngineKeepAlive(b *testing.B) {
	benchmarkEngine(b, &KeepAliveEngine{})
}

func BenchmarkEngineEpoll(b *testing.B) {
	benchmarkEngine(b, &EpollEngine{})
}

func BenchmarkEngineRawHTTP(b *testing.B) {
	benchmarkEngine(b, &RawHTTPEngine{})
}
