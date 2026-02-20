package bench_s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Runner orchestrates S3 benchmark execution.
type Runner struct {
	config  *Config
	results []*Metrics
	mu      sync.Mutex
	log     func(format string, args ...any)

	payloads   map[int][]byte
	payloadsMu sync.Mutex
	keyCounter uint64
}

// Metrics holds results for a single benchmark operation.
type Metrics struct {
	Driver     string        `json:"driver"`
	Operation  string        `json:"operation"`
	ObjectSize int           `json:"object_size"`
	Iterations int           `json:"iterations"`
	TotalTime  time.Duration `json:"total_time"`
	Errors     int           `json:"errors"`
	LastError  string        `json:"last_error,omitempty"`

	AvgLatency time.Duration `json:"avg_latency"`
	MinLatency time.Duration `json:"min_latency"`
	MaxLatency time.Duration `json:"max_latency"`
	P50Latency time.Duration `json:"p50_latency"`
	P95Latency time.Duration `json:"p95_latency"`
	P99Latency time.Duration `json:"p99_latency"`

	ThroughputMBps float64 `json:"throughput_mbps"`
	OpsPerSec      float64 `json:"ops_per_sec"`
}

// Report holds all benchmark results.
type Report struct {
	Timestamp time.Time  `json:"timestamp"`
	Config    *Config    `json:"config"`
	Results   []*Metrics `json:"results"`
}

// NewRunner creates a new benchmark runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		config:   cfg,
		results:  make([]*Metrics, 0),
		log:      func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
		payloads: make(map[int][]byte),
	}
}

// SetLogger sets a custom logger.
func (r *Runner) SetLogger(fn func(format string, args ...any)) {
	r.log = fn
}

// Run executes benchmarks in the configured mode.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	switch r.config.Mode {
	case "local":
		return r.runLocal(ctx)
	case "docker":
		return r.runDocker(ctx)
	default:
		return nil, fmt.Errorf("unknown mode: %s (use 'local' or 'docker')", r.config.Mode)
	}
}

// runLocal resolves binaries, starts local servers one at a time, benchmarks, then cleans up.
func (r *Runner) runLocal(ctx context.Context) (*Report, error) {
	r.log("=== S3 Client Benchmark (Local Mode) ===")
	r.log("BenchTime: %v, Warmup: %d", r.config.BenchTime, r.config.WarmupIters)
	r.log("")

	// Resolve all binaries and create temp dirs
	allServers := LocalServerConfigs()

	// Filter by driver names if specified
	if len(r.config.Drivers) > 0 {
		nameSet := make(map[string]bool)
		for _, n := range r.config.Drivers {
			nameSet[n] = true
		}
		var filtered []LocalServer
		for _, s := range allServers {
			if nameSet[s.Endpoint.Name] {
				filtered = append(filtered, s)
			}
		}
		allServers = filtered
	}

	r.log("--- Resolving binaries ---")
	servers, cleanup := ResolveBinaries(allServers, r.log)
	defer func() {
		r.log("")
		r.log("--- Final cleanup ---")
		cleanup()
	}()

	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers available for local mode")
	}
	r.log("")

	// Start, benchmark, stop each server sequentially
	for i, srv := range servers {
		if ctx.Err() != nil {
			break
		}

		r.log("=== [%d/%d] %s ===", i+1, len(servers), srv.Endpoint.Name)

		proc, err := StartServer(ctx, srv, r.log)
		if err != nil {
			r.log("  Failed to start %s: %v", srv.Endpoint.Name, err)
			continue
		}

		// Run benchmarks
		client := NewS3Client(srv.Endpoint)
		if err := EnsureBucket(ctx, client, r.config.Bucket); err != nil {
			r.log("  Failed to create bucket: %v", err)
			proc.Stop(r.log)
			continue
		}

		r.benchmarkEndpoint(ctx, client, srv.Endpoint.Name)

		// Cleanup S3 objects before stopping server
		r.cleanupObjects(ctx, client, srv.Endpoint.Name)

		// Stop server process
		r.log("  Stopping %s...", srv.Endpoint.Name)
		proc.Stop(r.log)
		r.log("")
	}

	return &Report{
		Timestamp: time.Now(),
		Config:    r.config,
		Results:   r.results,
	}, nil
}

// runDocker benchmarks against already-running docker containers.
func (r *Runner) runDocker(ctx context.Context) (*Report, error) {
	endpoints := FilterEndpoints(DockerEndpoints(), r.config.Drivers)
	r.log("=== S3 Client Benchmark (Docker Mode) ===")
	r.log("BenchTime: %v, Warmup: %d, Endpoints: %d", r.config.BenchTime, r.config.WarmupIters, len(endpoints))
	r.log("")

	// Detect available endpoints
	var available []Endpoint
	for _, ep := range endpoints {
		if CheckEndpoint(ctx, ep) {
			r.log("  %s: available", ep.Name)
			available = append(available, ep)
		} else {
			r.log("  %s: not available", ep.Name)
		}
	}
	r.log("")

	if len(available) == 0 {
		return nil, fmt.Errorf("no S3 endpoints available (start docker services first)")
	}

	for i, ep := range available {
		if ctx.Err() != nil {
			break
		}
		r.log("=== [%d/%d] %s ===", i+1, len(available), ep.Name)

		client := NewS3Client(ep)
		if err := EnsureBucket(ctx, client, r.config.Bucket); err != nil {
			r.log("  Failed to create bucket: %v", err)
			continue
		}

		r.benchmarkEndpoint(ctx, client, ep.Name)
		r.cleanupObjects(ctx, client, ep.Name)
		r.log("")
	}

	return &Report{
		Timestamp: time.Now(),
		Config:    r.config,
		Results:   r.results,
	}, nil
}

// benchmarkEndpoint runs all benchmark operations against one endpoint.
func (r *Runner) benchmarkEndpoint(ctx context.Context, client *s3.Client, driver string) {
	sizes := []int{SizeSmall, SizeMedium, SizeLarge, SizeXLarge}

	// PutObject
	for _, size := range sizes {
		if ctx.Err() != nil {
			return
		}
		if !r.matchFilter("PutObject") {
			continue
		}
		r.benchPutObject(ctx, client, driver, size)
	}

	// GetObject
	for _, size := range sizes {
		if ctx.Err() != nil {
			return
		}
		if !r.matchFilter("GetObject") {
			continue
		}
		r.benchGetObject(ctx, client, driver, size)
	}

	// HeadObject
	if ctx.Err() == nil && r.matchFilter("HeadObject") {
		r.benchHeadObject(ctx, client, driver)
	}

	// DeleteObject
	if ctx.Err() == nil && r.matchFilter("DeleteObject") {
		r.benchDeleteObject(ctx, client, driver)
	}

	// ListObjectsV2
	if ctx.Err() == nil && r.matchFilter("ListObjects") {
		r.benchListObjects(ctx, client, driver)
	}

	// Multipart
	if ctx.Err() == nil && r.matchFilter("Multipart") {
		r.benchMultipart(ctx, client, driver)
	}

	// Mixed workloads
	if ctx.Err() == nil && r.matchFilter("Mixed") {
		r.benchMixed(ctx, client, driver)
	}

	// Concurrency scaling
	if ctx.Err() == nil && r.matchFilter("Concurrency") {
		r.benchConcurrencyScaling(ctx, client, driver)
	}
}

func (r *Runner) matchFilter(name string) bool {
	if r.config.Filter == "" {
		return true
	}
	return strings.Contains(name, r.config.Filter)
}

func (r *Runner) payload(size int) []byte {
	r.payloadsMu.Lock()
	defer r.payloadsMu.Unlock()
	if p, ok := r.payloads[size]; ok {
		return p
	}
	p := make([]byte, size)
	rand.Read(p)
	r.payloads[size] = p
	return p
}

func (r *Runner) uniqueKey(prefix string) string {
	n := atomic.AddUint64(&r.keyCounter, 1)
	return fmt.Sprintf("bench/%s/%d", prefix, n)
}

func (r *Runner) addResult(m *Metrics) {
	r.mu.Lock()
	r.results = append(r.results, m)
	r.mu.Unlock()
}

// ============================================================================
// LATENCY COLLECTOR
// ============================================================================

type latencyCollector struct {
	mu        sync.Mutex
	latencies []time.Duration
	errors    int
	lastError string
}

func newCollector() *latencyCollector {
	return &latencyCollector{
		latencies: make([]time.Duration, 0, 1024),
	}
}

func (c *latencyCollector) record(d time.Duration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.errors++
		c.lastError = err.Error()
		return
	}
	c.latencies = append(c.latencies, d)
}

func (c *latencyCollector) metrics(op, driver string, size int) *Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	m := &Metrics{
		Driver:     driver,
		Operation:  op,
		ObjectSize: size,
		Iterations: len(c.latencies),
		Errors:     c.errors,
		LastError:  c.lastError,
	}

	if len(c.latencies) == 0 {
		return m
	}

	sorted := make([]time.Duration, len(c.latencies))
	copy(sorted, c.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	var total time.Duration
	for _, d := range sorted {
		total += d
	}

	m.TotalTime = total
	m.AvgLatency = total / time.Duration(len(sorted))
	m.MinLatency = sorted[0]
	m.MaxLatency = sorted[len(sorted)-1]
	m.P50Latency = percentile(sorted, 50)
	m.P95Latency = percentile(sorted, 95)
	m.P99Latency = percentile(sorted, 99)

	secs := total.Seconds()
	if secs > 0 {
		m.OpsPerSec = float64(len(sorted)) / secs
		if size > 0 {
			m.ThroughputMBps = float64(int64(len(sorted))*int64(size)) / secs / (1024 * 1024)
		}
	}

	return m
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// ============================================================================
// ADAPTIVE BENCHMARK
// ============================================================================

type adaptiveBench struct {
	target    time.Duration
	minIters  int
	maxIters  int
	totalN    int
	totalTime time.Duration
	ctx       context.Context
}

func newAdaptiveBench(ctx context.Context, target time.Duration, minIters, maxIters int) *adaptiveBench {
	return &adaptiveBench{
		target:   target,
		minIters: minIters,
		maxIters: maxIters,
		ctx:      ctx,
	}
}

func (ab *adaptiveBench) shouldContinue() bool {
	if ab.ctx.Err() != nil {
		return false
	}
	if ab.totalN < ab.minIters {
		return true
	}
	return ab.totalTime < ab.target
}

func (ab *adaptiveBench) nextN() int {
	if ab.totalN == 0 {
		return 1
	}
	if ab.totalTime <= 0 {
		return ab.totalN * 2
	}
	remaining := ab.target - ab.totalTime
	if remaining <= 0 {
		return 1
	}
	rate := float64(ab.totalN) / ab.totalTime.Seconds()
	n := int(rate * remaining.Seconds())
	if n < 1 {
		n = 1
	}
	if n > ab.totalN*2 {
		n = ab.totalN * 2
	}
	if ab.totalN+n > ab.maxIters {
		n = ab.maxIters - ab.totalN
		if n < 1 {
			n = 1
		}
	}
	return n
}

func (ab *adaptiveBench) recordRun(n int, elapsed time.Duration) {
	ab.totalN += n
	ab.totalTime += elapsed
}

// ============================================================================
// BENCHMARK OPERATIONS
// ============================================================================

func (r *Runner) benchPutObject(ctx context.Context, client *s3.Client, driver string, size int) {
	op := fmt.Sprintf("PutObject/%s", sizeLabel(size))
	data := r.payload(size)
	warmup := r.config.WarmupForSize(size)

	for i := 0; i < warmup; i++ {
		key := r.uniqueKey("warmup")
		client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        aws.String(r.config.Bucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(data),
			ContentLength: aws.Int64(int64(size)),
		})
	}

	coll := newCollector()
	benchTime := r.config.BenchTimeForSize(size)
	ab := newAdaptiveBench(ctx, benchTime, r.config.MinIterations, r.config.MaxIterationsForSize(size))

	for ab.shouldContinue() {
		n := ab.nextN()
		start := time.Now()
		for i := 0; i < n; i++ {
			if i > 0 && time.Since(start) > 3*benchTime {
				break
			}
			key := r.uniqueKey("put")
			t0 := time.Now()
			_, err := client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:        aws.String(r.config.Bucket),
				Key:           aws.String(key),
				Body:          bytes.NewReader(data),
				ContentLength: aws.Int64(int64(size)),
			})
			coll.record(time.Since(t0), err)
		}
		ab.recordRun(n, time.Since(start))
	}

	m := coll.metrics(op, driver, size)
	r.addResult(m)
	r.log("  %s: %d iters, avg=%v, p50=%v, p99=%v, %.1f MB/s, %d errors",
		op, m.Iterations, m.AvgLatency.Round(time.Microsecond), m.P50Latency.Round(time.Microsecond),
		m.P99Latency.Round(time.Microsecond), m.ThroughputMBps, m.Errors)
}

func (r *Runner) benchGetObject(ctx context.Context, client *s3.Client, driver string, size int) {
	op := fmt.Sprintf("GetObject/%s", sizeLabel(size))
	data := r.payload(size)

	key := r.uniqueKey("get-seed")
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.config.Bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(size)),
	})
	if err != nil {
		r.log("  %s: seed failed: %v", op, err)
		return
	}

	warmup := r.config.WarmupForSize(size)
	for i := 0; i < warmup; i++ {
		resp, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(r.config.Bucket),
			Key:    aws.String(key),
		})
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}

	coll := newCollector()
	benchTime := r.config.BenchTimeForSize(size)
	ab := newAdaptiveBench(ctx, benchTime, r.config.MinIterations, r.config.MaxIterationsForSize(size))

	for ab.shouldContinue() {
		n := ab.nextN()
		start := time.Now()
		for i := 0; i < n; i++ {
			if i > 0 && time.Since(start) > 3*benchTime {
				break
			}
			t0 := time.Now()
			resp, err := client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(r.config.Bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				coll.record(0, err)
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			coll.record(time.Since(t0), nil)
		}
		ab.recordRun(n, time.Since(start))
	}

	m := coll.metrics(op, driver, size)
	r.addResult(m)
	r.log("  %s: %d iters, avg=%v, p50=%v, p99=%v, %.1f MB/s, %d errors",
		op, m.Iterations, m.AvgLatency.Round(time.Microsecond), m.P50Latency.Round(time.Microsecond),
		m.P99Latency.Round(time.Microsecond), m.ThroughputMBps, m.Errors)
}

func (r *Runner) benchHeadObject(ctx context.Context, client *s3.Client, driver string) {
	op := "HeadObject"
	data := r.payload(SizeSmall)

	key := r.uniqueKey("head-seed")
	client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.config.Bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(SizeSmall)),
	})

	for i := 0; i < r.config.WarmupIters; i++ {
		client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(r.config.Bucket),
			Key:    aws.String(key),
		})
	}

	coll := newCollector()
	ab := newAdaptiveBench(ctx, r.config.BenchTime, r.config.MinIterations, 1_000_000)

	for ab.shouldContinue() {
		n := ab.nextN()
		start := time.Now()
		for i := 0; i < n; i++ {
			t0 := time.Now()
			_, err := client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(r.config.Bucket),
				Key:    aws.String(key),
			})
			coll.record(time.Since(t0), err)
		}
		ab.recordRun(n, time.Since(start))
	}

	m := coll.metrics(op, driver, 0)
	r.addResult(m)
	r.log("  %s: %d iters, avg=%v, p50=%v, p99=%v, %.0f ops/s, %d errors",
		op, m.Iterations, m.AvgLatency.Round(time.Microsecond), m.P50Latency.Round(time.Microsecond),
		m.P99Latency.Round(time.Microsecond), m.OpsPerSec, m.Errors)
}

func (r *Runner) benchDeleteObject(ctx context.Context, client *s3.Client, driver string) {
	op := "DeleteObject"
	data := r.payload(SizeSmall)

	coll := newCollector()
	ab := newAdaptiveBench(ctx, r.config.BenchTime, r.config.MinIterations, 1_000_000)

	for ab.shouldContinue() {
		n := ab.nextN()

		keys := make([]string, n)
		for i := 0; i < n; i++ {
			keys[i] = r.uniqueKey("del")
			client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:        aws.String(r.config.Bucket),
				Key:           aws.String(keys[i]),
				Body:          bytes.NewReader(data),
				ContentLength: aws.Int64(int64(SizeSmall)),
			})
		}

		start := time.Now()
		for _, key := range keys {
			t0 := time.Now()
			_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(r.config.Bucket),
				Key:    aws.String(key),
			})
			coll.record(time.Since(t0), err)
		}
		ab.recordRun(n, time.Since(start))
	}

	m := coll.metrics(op, driver, 0)
	r.addResult(m)
	r.log("  %s: %d iters, avg=%v, p50=%v, p99=%v, %.0f ops/s, %d errors",
		op, m.Iterations, m.AvgLatency.Round(time.Microsecond), m.P50Latency.Round(time.Microsecond),
		m.P99Latency.Round(time.Microsecond), m.OpsPerSec, m.Errors)
}

func (r *Runner) benchListObjects(ctx context.Context, client *s3.Client, driver string) {
	op := "ListObjects"
	data := r.payload(SizeSmall)

	prefix := r.uniqueKey("list-seed")
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("%s/%04d", prefix, i)
		client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        aws.String(r.config.Bucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(data),
			ContentLength: aws.Int64(int64(SizeSmall)),
		})
	}

	for i := 0; i < r.config.WarmupIters; i++ {
		client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(r.config.Bucket),
			Prefix: aws.String(prefix + "/"),
		})
	}

	coll := newCollector()
	ab := newAdaptiveBench(ctx, r.config.BenchTime, r.config.MinIterations, 100_000)

	for ab.shouldContinue() {
		n := ab.nextN()
		start := time.Now()
		for i := 0; i < n; i++ {
			t0 := time.Now()
			_, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
				Bucket: aws.String(r.config.Bucket),
				Prefix: aws.String(prefix + "/"),
			})
			coll.record(time.Since(t0), err)
		}
		ab.recordRun(n, time.Since(start))
	}

	m := coll.metrics(op, driver, 0)
	r.addResult(m)
	r.log("  %s: %d iters, avg=%v, p50=%v, p99=%v, %.0f ops/s, %d errors",
		op, m.Iterations, m.AvgLatency.Round(time.Microsecond), m.P50Latency.Round(time.Microsecond),
		m.P99Latency.Round(time.Microsecond), m.OpsPerSec, m.Errors)
}

func (r *Runner) benchMultipart(ctx context.Context, client *s3.Client, driver string) {
	op := "Multipart/20MB"
	partSize := 5 * 1024 * 1024
	numParts := 4
	partData := r.payload(partSize)

	coll := newCollector()
	ab := newAdaptiveBench(ctx, r.config.BenchTime, r.config.MinIterations, 100)

	for ab.shouldContinue() {
		n := ab.nextN()
		start := time.Now()
		for i := 0; i < n; i++ {
			if i > 0 && time.Since(start) > 3*r.config.BenchTime {
				break
			}
			key := r.uniqueKey("multipart")
			t0 := time.Now()

			createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
				Bucket: aws.String(r.config.Bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				coll.record(0, err)
				continue
			}

			var completedParts []types.CompletedPart
			failed := false
			for p := 1; p <= numParts; p++ {
				partResp, err := client.UploadPart(ctx, &s3.UploadPartInput{
					Bucket:     aws.String(r.config.Bucket),
					Key:        aws.String(key),
					UploadId:   createResp.UploadId,
					PartNumber: aws.Int32(int32(p)),
					Body:       bytes.NewReader(partData),
				})
				if err != nil {
					coll.record(0, err)
					client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   aws.String(r.config.Bucket),
						Key:      aws.String(key),
						UploadId: createResp.UploadId,
					})
					failed = true
					break
				}
				completedParts = append(completedParts, types.CompletedPart{
					ETag:       partResp.ETag,
					PartNumber: aws.Int32(int32(p)),
				})
			}
			if failed {
				continue
			}

			_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
				Bucket:   aws.String(r.config.Bucket),
				Key:      aws.String(key),
				UploadId: createResp.UploadId,
				MultipartUpload: &types.CompletedMultipartUpload{
					Parts: completedParts,
				},
			})
			coll.record(time.Since(t0), err)
		}
		ab.recordRun(n, time.Since(start))
	}

	totalSize := partSize * numParts
	m := coll.metrics(op, driver, totalSize)
	r.addResult(m)
	r.log("  %s: %d iters, avg=%v, p50=%v, p99=%v, %.1f MB/s, %d errors",
		op, m.Iterations, m.AvgLatency.Round(time.Microsecond), m.P50Latency.Round(time.Microsecond),
		m.P99Latency.Round(time.Microsecond), m.ThroughputMBps, m.Errors)
}

func (r *Runner) benchMixed(ctx context.Context, client *s3.Client, driver string) {
	type ratio struct {
		name     string
		readPct  int
	}
	ratios := []ratio{
		{"Mixed/90r10w", 90},
		{"Mixed/50r50w", 50},
		{"Mixed/10r90w", 10},
	}

	size := SizeMedium
	data := r.payload(size)
	concurrency := 50

	for _, rat := range ratios {
		if ctx.Err() != nil {
			break
		}

		readKeys := make([]string, 100)
		for i := range readKeys {
			readKeys[i] = r.uniqueKey("mixed-seed")
			client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:        aws.String(r.config.Bucket),
				Key:           aws.String(readKeys[i]),
				Body:          bytes.NewReader(data),
				ContentLength: aws.Int64(int64(size)),
			})
		}

		coll := newCollector()
		benchTime := r.config.BenchTime

		var wg sync.WaitGroup
		opCtx, cancel := context.WithTimeout(ctx, benchTime+5*time.Second)
		deadline := time.Now().Add(benchTime)

		for w := 0; w < concurrency; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				counter := 0
				for time.Now().Before(deadline) {
					if opCtx.Err() != nil {
						return
					}
					counter++
					isRead := (counter % 100) < rat.readPct

					t0 := time.Now()
					if isRead {
						key := readKeys[counter%len(readKeys)]
						resp, err := client.GetObject(opCtx, &s3.GetObjectInput{
							Bucket: aws.String(r.config.Bucket),
							Key:    aws.String(key),
						})
						if err != nil {
							coll.record(0, err)
							continue
						}
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
						coll.record(time.Since(t0), nil)
					} else {
						key := r.uniqueKey("mixed-w")
						_, err := client.PutObject(opCtx, &s3.PutObjectInput{
							Bucket:        aws.String(r.config.Bucket),
							Key:           aws.String(key),
							Body:          bytes.NewReader(data),
							ContentLength: aws.Int64(int64(size)),
						})
						coll.record(time.Since(t0), err)
					}
				}
			}(w)
		}

		wg.Wait()
		cancel()

		m := coll.metrics(rat.name, driver, size)
		r.addResult(m)
		r.log("  %s (C%d): %d iters, avg=%v, p50=%v, p99=%v, %.1f MB/s, %d errors",
			rat.name, concurrency, m.Iterations, m.AvgLatency.Round(time.Microsecond),
			m.P50Latency.Round(time.Microsecond), m.P99Latency.Round(time.Microsecond),
			m.ThroughputMBps, m.Errors)
	}
}

func (r *Runner) benchConcurrencyScaling(ctx context.Context, client *s3.Client, driver string) {
	size := SizeMedium
	data := r.payload(size)
	concLevels := []int{1, 10, 50, 100, 200}

	for _, conc := range concLevels {
		if ctx.Err() != nil {
			break
		}

		op := fmt.Sprintf("Concurrency/C%d/PutObject/64KB", conc)
		coll := newCollector()
		benchTime := r.config.BenchTime

		var wg sync.WaitGroup
		opCtx, cancel := context.WithTimeout(ctx, benchTime+5*time.Second)
		deadline := time.Now().Add(benchTime)

		for w := 0; w < conc; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for time.Now().Before(deadline) {
					if opCtx.Err() != nil {
						return
					}
					key := r.uniqueKey("conc")
					t0 := time.Now()
					_, err := client.PutObject(opCtx, &s3.PutObjectInput{
						Bucket:        aws.String(r.config.Bucket),
						Key:           aws.String(key),
						Body:          bytes.NewReader(data),
						ContentLength: aws.Int64(int64(size)),
					})
					coll.record(time.Since(t0), err)
				}
			}()
		}

		wg.Wait()
		cancel()

		m := coll.metrics(op, driver, size)
		r.addResult(m)
		r.log("  %s: %d iters, avg=%v, %.1f MB/s, %.0f ops/s, %d errors",
			op, m.Iterations, m.AvgLatency.Round(time.Microsecond), m.ThroughputMBps, m.OpsPerSec, m.Errors)
	}
}

func (r *Runner) cleanupObjects(ctx context.Context, client *s3.Client, driver string) {
	r.log("  Cleaning up %s objects...", driver)

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(r.config.Bucket),
		Prefix: aws.String("bench/"),
	})

	deleted := 0
	for paginator.HasMorePages() {
		if ctx.Err() != nil {
			break
		}
		page, err := paginator.NextPage(ctx)
		if err != nil {
			break
		}
		for _, obj := range page.Contents {
			client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(r.config.Bucket),
				Key:    obj.Key,
			})
			deleted++
		}
	}
	r.log("  Deleted %d objects", deleted)
}

func sizeLabel(size int) string {
	switch {
	case size >= 1024*1024:
		return fmt.Sprintf("%dMB", size/(1024*1024))
	case size >= 1024:
		return fmt.Sprintf("%dKB", size/1024)
	default:
		return fmt.Sprintf("%dB", size)
	}
}
