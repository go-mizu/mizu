// Package bench provides comprehensive benchmarks for the storage package.
//
// Run with: go test -bench=. -benchmem -benchtime=3s ./...
// Generate report: go test -bench=. -benchmem -json ./... > results.json
package bench

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/exp/s3"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/local"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/memory"
)

// DriverConfig holds configuration for a storage driver.
type DriverConfig struct {
	Name     string
	DSN      string
	Bucket   string
	Skip     bool
	SkipMsg  string
	Features map[string]bool
}

// BenchResult holds benchmark results for report generation.
type BenchResult struct {
	Driver     string        `json:"driver"`
	Benchmark  string        `json:"benchmark"`
	Iterations int           `json:"iterations"`
	NsPerOp    float64       `json:"ns_per_op"`
	MBPerSec   float64       `json:"mb_per_sec,omitempty"`
	BytesPerOp int64         `json:"bytes_per_op"`
	AllocsOp   int64         `json:"allocs_per_op"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// benchResults collects results for report generation.
var (
	benchResults   []BenchResult
	benchResultsMu sync.Mutex
)

// recordResult adds a benchmark result.
func recordResult(r BenchResult) {
	benchResultsMu.Lock()
	benchResults = append(benchResults, r)
	benchResultsMu.Unlock()
}

// getDriverConfigs returns configurations for all available drivers.
func getDriverConfigs(t testing.TB) []DriverConfig {
	configs := []DriverConfig{}

	// Memory driver (always available, baseline)
	// TEMPORARILY DISABLED for S3-only benchmarks
	configs = append(configs, DriverConfig{
		Name:    "memory",
		Skip:    true,
		SkipMsg: "Memory driver temporarily disabled for S3-only benchmarks",
	})

	// Local filesystem driver
	// TEMPORARILY DISABLED for S3-only benchmarks
	configs = append(configs, DriverConfig{
		Name:    "local",
		Skip:    true,
		SkipMsg: "Local driver temporarily disabled for S3-only benchmarks",
	})

	// MinIO (port 9000)
	if checkS3Endpoint("localhost:9000", "minioadmin", "minioadmin") {
		configs = append(configs, DriverConfig{
			Name:   "minio",
			DSN:    "s3://minioadmin:minioadmin@localhost:9000/test-bucket?insecure=true&force_path_style=true",
			Bucket: "test-bucket",
		})
	} else {
		configs = append(configs, DriverConfig{
			Name:    "minio",
			Skip:    true,
			SkipMsg: "MinIO not available at localhost:9000",
		})
	}

	// RustFS (port 9100)
	if checkS3Endpoint("localhost:9100", "rustfsadmin", "rustfsadmin") {
		configs = append(configs, DriverConfig{
			Name:   "rustfs",
			DSN:    "s3://rustfsadmin:rustfsadmin@localhost:9100/test-bucket?insecure=true&force_path_style=true",
			Bucket: "test-bucket",
		})
	} else {
		configs = append(configs, DriverConfig{
			Name:    "rustfs",
			Skip:    true,
			SkipMsg: "RustFS not available at localhost:9100",
		})
	}

	// SeaweedFS (port 8333)
	if checkS3Endpoint("localhost:8333", "admin", "adminpassword") {
		configs = append(configs, DriverConfig{
			Name:   "seaweedfs",
			DSN:    "s3://admin:adminpassword@localhost:8333/test-bucket?insecure=true&force_path_style=true",
			Bucket: "test-bucket",
		})
	} else {
		configs = append(configs, DriverConfig{
			Name:    "seaweedfs",
			Skip:    true,
			SkipMsg: "SeaweedFS not available at localhost:8333",
		})
	}

	// LocalStack (port 4566)
	if checkS3Endpoint("localhost:4566", "test", "test") {
		configs = append(configs, DriverConfig{
			Name:   "localstack",
			DSN:    "s3://test:test@localhost:4566/test-bucket?insecure=true&force_path_style=true",
			Bucket: "test-bucket",
		})
	} else {
		configs = append(configs, DriverConfig{
			Name:    "localstack",
			Skip:    true,
			SkipMsg: "LocalStack not available at localhost:4566",
		})
	}

	return configs
}

// checkS3Endpoint performs a quick S3 connection check with credentials.
func checkS3Endpoint(addr, accessKey, secretKey string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Try to open storage to verify connection
	dsn := fmt.Sprintf("s3://%s:%s@%s/test-bucket?insecure=true&force_path_style=true", accessKey, secretKey, addr)
	st, err := storage.Open(ctx, dsn)
	if err != nil {
		return false
	}
	defer st.Close()

	// Try to list buckets to verify the connection actually works
	_, err = st.Buckets(ctx, 1, 0, nil)
	return err == nil
}

// openStorage opens a storage connection from config.
func openStorage(ctx context.Context, cfg DriverConfig) (storage.Storage, error) {
	return storage.Open(ctx, cfg.DSN)
}

// generateData generates random data of the specified size.
func generateData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// generatePatternData generates deterministic pattern data.
func generatePatternData(size int) []byte {
	pattern := []byte("benchmark_test_data_pattern_0123456789_")
	data := make([]byte, size)
	for i := range data {
		data[i] = pattern[i%len(pattern)]
	}
	return data
}

// uniqueKey generates a unique object key.
var keyCounter uint64

func uniqueKey(prefix string) string {
	n := atomic.AddUint64(&keyCounter, 1)
	return fmt.Sprintf("%s/%d/%d", prefix, time.Now().UnixNano(), n)
}

// setupBucket ensures a bucket exists and is clean.
func setupBucket(ctx context.Context, st storage.Storage, name string) (storage.Bucket, error) {
	// Try to create, ignore if exists
	st.CreateBucket(ctx, name, nil)
	return st.Bucket(name), nil
}

// cleanupBucket removes all objects from a bucket.
func cleanupBucket(ctx context.Context, bucket storage.Bucket) {
	iter, err := bucket.List(ctx, "", 1000, 0, nil)
	if err != nil {
		return
	}
	defer iter.Close()

	for {
		obj, err := iter.Next()
		if err != nil || obj == nil {
			break
		}
		bucket.Delete(ctx, obj.Key, nil)
	}
}

// ============================================================================
// WRITE BENCHMARKS
// ============================================================================

func BenchmarkWrite(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Tiny_1B", 1},
		{"Small_1KB", 1024},
		{"Medium_64KB", 64 * 1024},
		{"Standard_1MB", 1024 * 1024},
	}

	// Include larger sizes only if explicitly requested
	if os.Getenv("BENCH_LARGE") == "1" {
		sizes = append(sizes,
			struct{ name string; size int }{"Large_10MB", 10 * 1024 * 1024},
			struct{ name string; size int }{"XLarge_100MB", 100 * 1024 * 1024},
		)
	}

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			for _, sz := range sizes {
				data := generateData(sz.size)

				b.Run(sz.name, func(b *testing.B) {
					b.SetBytes(int64(sz.size))
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						key := uniqueKey("write")
						_, err := bucket.Write(ctx, key, bytes.NewReader(data), int64(sz.size), "application/octet-stream", nil)
						if err != nil {
							b.Fatalf("write: %v", err)
						}
					}
				})
			}
		})
	}
}

// ============================================================================
// READ BENCHMARKS
// ============================================================================

func BenchmarkRead(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Small_1KB", 1024},
		{"Medium_64KB", 64 * 1024},
		{"Standard_1MB", 1024 * 1024},
	}

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			for _, sz := range sizes {
				data := generateData(sz.size)

				// Pre-create object for reading
				key := uniqueKey("read")
				_, err := bucket.Write(ctx, key, bytes.NewReader(data), int64(sz.size), "application/octet-stream", nil)
				if err != nil {
					b.Fatalf("setup write: %v", err)
				}

				b.Run(sz.name, func(b *testing.B) {
					b.SetBytes(int64(sz.size))
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						rc, _, err := bucket.Open(ctx, key, 0, 0, nil)
						if err != nil {
							b.Fatalf("open: %v", err)
						}
						io.Copy(io.Discard, rc)
						rc.Close()
					}
				})
			}
		})
	}
}

// ============================================================================
// RANGE READ BENCHMARKS
// ============================================================================

func BenchmarkRangeRead(b *testing.B) {
	const totalSize = 1024 * 1024 // 1MB object (reduced for speed)

	ranges := []struct {
		name   string
		offset int64
		length int64
	}{
		{"Start_256KB", 0, 256 * 1024},
		{"Middle_256KB", 512 * 1024, 256 * 1024},
		{"End_256KB", 768 * 1024, 256 * 1024},
		{"Tiny_4KB", 512 * 1024, 4096},
	}

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			// Pre-create large object
			data := generateData(totalSize)
			key := uniqueKey("range")
			_, err = bucket.Write(ctx, key, bytes.NewReader(data), int64(totalSize), "application/octet-stream", nil)
			if err != nil {
				b.Fatalf("setup write: %v", err)
			}

			for _, rng := range ranges {
				b.Run(rng.name, func(b *testing.B) {
					b.SetBytes(rng.length)
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						rc, _, err := bucket.Open(ctx, key, rng.offset, rng.length, nil)
						if err != nil {
							b.Fatalf("range open: %v", err)
						}
						io.Copy(io.Discard, rc)
						rc.Close()
					}
				})
			}
		})
	}
}

// ============================================================================
// STAT BENCHMARKS
// ============================================================================

func BenchmarkStat(b *testing.B) {
	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			// Create test object
			key := uniqueKey("stat")
			data := generateData(1024)
			_, err = bucket.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
			if err != nil {
				b.Fatalf("setup write: %v", err)
			}

			b.Run("Exists", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := bucket.Stat(ctx, key, nil)
					if err != nil {
						b.Fatalf("stat: %v", err)
					}
				}
			})

			b.Run("NotExists", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					bucket.Stat(ctx, "nonexistent-key-12345", nil)
				}
			})
		})
	}
}

// ============================================================================
// DELETE BENCHMARKS
// ============================================================================

func BenchmarkDelete(b *testing.B) {
	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			b.Run("Single", func(b *testing.B) {
				// Pre-create objects
				keys := make([]string, b.N)
				data := generateData(1024)
				for i := 0; i < b.N; i++ {
					keys[i] = uniqueKey("delete")
					bucket.Write(ctx, keys[i], bytes.NewReader(data), 1024, "application/octet-stream", nil)
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					bucket.Delete(ctx, keys[i], nil)
				}
			})

			b.Run("NonExistent", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					bucket.Delete(ctx, fmt.Sprintf("nonexistent-%d", i), nil)
				}
			})
		})
	}
}

// ============================================================================
// COPY BENCHMARKS
// ============================================================================

func BenchmarkCopy(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Small_1KB", 1024},
		{"Standard_1MB", 1024 * 1024},
	}

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			for _, sz := range sizes {
				data := generateData(sz.size)

				// Create source object
				srcKey := uniqueKey("copy-src")
				_, err := bucket.Write(ctx, srcKey, bytes.NewReader(data), int64(sz.size), "application/octet-stream", nil)
				if err != nil {
					b.Fatalf("setup write: %v", err)
				}

				b.Run("SameBucket_"+sz.name, func(b *testing.B) {
					b.SetBytes(int64(sz.size))
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						dstKey := uniqueKey("copy-dst")
						_, err := bucket.Copy(ctx, dstKey, cfg.Bucket, srcKey, nil)
						if err != nil {
							b.Fatalf("copy: %v", err)
						}
					}
				})
			}
		})
	}
}

// ============================================================================
// LIST BENCHMARKS
// ============================================================================

func BenchmarkList(b *testing.B) {
	counts := []int{10, 50, 100} // Reduced for speed

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			for _, count := range counts {
				// Create objects for listing
				prefix := uniqueKey("list")
				data := generateData(100)
				for i := 0; i < count; i++ {
					key := fmt.Sprintf("%s/obj-%05d", prefix, i)
					bucket.Write(ctx, key, bytes.NewReader(data), 100, "text/plain", nil)
				}

				b.Run(fmt.Sprintf("%d_Objects", count), func(b *testing.B) {
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						iter, err := bucket.List(ctx, prefix, 0, 0, nil)
						if err != nil {
							b.Fatalf("list: %v", err)
						}
						n := 0
						for {
							obj, err := iter.Next()
							if err != nil || obj == nil {
								break
							}
							n++
						}
						iter.Close()
						if n < count {
							b.Fatalf("expected %d objects, got %d", count, n)
						}
					}
				})
			}

			// Test prefix filtering (reduced for speed)
			prefix := uniqueKey("listprefix")
			data := generateData(100)
			for i := 0; i < 200; i++ {
				var key string
				if i < 50 {
					key = fmt.Sprintf("%s/match/obj-%05d", prefix, i)
				} else {
					key = fmt.Sprintf("%s/other/obj-%05d", prefix, i)
				}
				bucket.Write(ctx, key, bytes.NewReader(data), 100, "text/plain", nil)
			}

			b.Run("Prefix_50_of_200", func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					iter, err := bucket.List(ctx, prefix+"/match", 0, 0, nil)
					if err != nil {
						b.Fatalf("list: %v", err)
					}
					n := 0
					for {
						obj, err := iter.Next()
						if err != nil || obj == nil {
							break
						}
						n++
					}
					iter.Close()
				}
			})
		})
	}
}

// ============================================================================
// PARALLEL WRITE BENCHMARKS
// ============================================================================

func BenchmarkParallelWrite(b *testing.B) {
	concurrencies := []int{1, 10, 25}
	objectSize := 64 * 1024 // 64KB (reduced for speed)

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			for _, conc := range concurrencies {
				b.Run(fmt.Sprintf("C%d", conc), func(b *testing.B) {
					data := generateData(objectSize)
					b.SetBytes(int64(objectSize))
					b.SetParallelism(conc)
					b.ResetTimer()

					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							key := uniqueKey("parallel-write")
							_, err := bucket.Write(ctx, key, bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
							if err != nil {
								b.Errorf("write: %v", err)
							}
						}
					})
				})
			}
		})
	}
}

// ============================================================================
// PARALLEL READ BENCHMARKS
// ============================================================================

func BenchmarkParallelRead(b *testing.B) {
	concurrencies := []int{1, 10, 25}
	objectSize := 64 * 1024 // 64KB (reduced for speed)
	numObjects := 20        // Pre-create this many objects to read from

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			// Pre-create objects
			data := generateData(objectSize)
			keys := make([]string, numObjects)
			for i := 0; i < numObjects; i++ {
				keys[i] = uniqueKey("parallel-read")
				_, err := bucket.Write(ctx, keys[i], bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
				if err != nil {
					b.Fatalf("setup write: %v", err)
				}
			}

			var keyIdx uint64

			for _, conc := range concurrencies {
				b.Run(fmt.Sprintf("C%d", conc), func(b *testing.B) {
					b.SetBytes(int64(objectSize))
					b.SetParallelism(conc)
					b.ResetTimer()

					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							idx := atomic.AddUint64(&keyIdx, 1) % uint64(numObjects)
							rc, _, err := bucket.Open(ctx, keys[idx], 0, 0, nil)
							if err != nil {
								b.Errorf("open: %v", err)
								continue
							}
							io.Copy(io.Discard, rc)
							rc.Close()
						}
					})
				})
			}
		})
	}
}

// ============================================================================
// MIXED WORKLOAD BENCHMARKS
// ============================================================================

func BenchmarkMixedWorkload(b *testing.B) {
	workloads := []struct {
		name       string
		readRatio  int // percentage of reads
		writeRatio int // percentage of writes
	}{
		{"ReadHeavy_90_10", 90, 10},
		{"WriteHeavy_10_90", 10, 90},
		{"Balanced_50_50", 50, 50},
	}

	objectSize := 16 * 1024 // 16KB (reduced for speed)
	concurrency := 10       // Reduced for speed

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			// Pre-create objects for reading
			data := generateData(objectSize)
			keys := make([]string, 100)
			for i := range keys {
				keys[i] = uniqueKey("mixed")
				bucket.Write(ctx, keys[i], bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
			}

			var keyIdx uint64
			var opCounter uint64

			for _, wl := range workloads {
				b.Run(wl.name, func(b *testing.B) {
					b.SetBytes(int64(objectSize))
					b.SetParallelism(concurrency)
					b.ResetTimer()

					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							op := atomic.AddUint64(&opCounter, 1) % 100
							if int(op) < wl.readRatio {
								// Read operation
								idx := atomic.AddUint64(&keyIdx, 1) % uint64(len(keys))
								rc, _, err := bucket.Open(ctx, keys[idx], 0, 0, nil)
								if err == nil {
									io.Copy(io.Discard, rc)
									rc.Close()
								}
							} else {
								// Write operation
								key := uniqueKey("mixed-write")
								bucket.Write(ctx, key, bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
							}
						}
					})
				})
			}
		})
	}
}

// ============================================================================
// MULTIPART UPLOAD BENCHMARKS
// ============================================================================

func BenchmarkMultipart(b *testing.B) {
	uploads := []struct {
		name      string
		totalSize int
		partSize  int
		partCount int
	}{
		{"5MB_5Parts", 5 * 1024 * 1024, 1024 * 1024, 5},
		{"10MB_5Parts", 10 * 1024 * 1024, 2 * 1024 * 1024, 5},
	}

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			// Check if multipart is supported
			mp, ok := bucket.(storage.HasMultipart)
			if !ok {
				b.Skip("multipart not supported")
				return
			}

			for _, up := range uploads {
				partData := generateData(up.partSize)

				b.Run(up.name, func(b *testing.B) {
					b.SetBytes(int64(up.totalSize))
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						key := uniqueKey("multipart")

						// Init
						mu, err := mp.InitMultipart(ctx, key, "application/octet-stream", nil)
						if err != nil {
							b.Fatalf("init multipart: %v", err)
						}

						// Upload parts
						parts := make([]*storage.PartInfo, up.partCount)
						for p := 0; p < up.partCount; p++ {
							part, err := mp.UploadPart(ctx, mu, p+1, bytes.NewReader(partData), int64(up.partSize), nil)
							if err != nil {
								mp.AbortMultipart(ctx, mu, nil)
								b.Fatalf("upload part %d: %v", p+1, err)
							}
							parts[p] = part
						}

						// Complete
						_, err = mp.CompleteMultipart(ctx, mu, parts, nil)
						if err != nil {
							b.Fatalf("complete multipart: %v", err)
						}
					}
				})
			}
		})
	}
}

// ============================================================================
// EDGE CASE BENCHMARKS
// ============================================================================

func BenchmarkEdgeCases(b *testing.B) {
	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			// Empty object
			b.Run("Empty_Write", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					key := uniqueKey("empty")
					bucket.Write(ctx, key, bytes.NewReader(nil), 0, "application/octet-stream", nil)
				}
			})

			// Long key names
			b.Run("LongKey_256", func(b *testing.B) {
				data := generateData(100)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					key := fmt.Sprintf("prefix/%s/%d", strings.Repeat("a", 200), i)
					bucket.Write(ctx, key, bytes.NewReader(data), 100, "text/plain", nil)
				}
			})

			// Unicode keys
			b.Run("UnicodeKey", func(b *testing.B) {
				data := generateData(100)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					key := fmt.Sprintf("unicode/\u4e2d\u6587/\u65e5\u672c\u8a9e/%d", i)
					bucket.Write(ctx, key, bytes.NewReader(data), 100, "text/plain", nil)
				}
			})

			// Deeply nested paths
			b.Run("DeepNested", func(b *testing.B) {
				data := generateData(100)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					key := fmt.Sprintf("a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/%d", i)
					bucket.Write(ctx, key, bytes.NewReader(data), 100, "text/plain", nil)
				}
			})
		})
	}
}

// ============================================================================
// METADATA BENCHMARKS
// ============================================================================

func BenchmarkMetadata(b *testing.B) {
	metadataCounts := []struct {
		name  string
		count int
	}{
		{"None", 0},
		{"Small_5", 5},
		{"Large_20", 20},
	}

	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			bucket, err := setupBucket(ctx, st, cfg.Bucket)
			if err != nil {
				b.Fatalf("setup bucket: %v", err)
			}
			defer cleanupBucket(ctx, bucket)

			data := generateData(1024)

			for _, mc := range metadataCounts {
				metadata := make(map[string]string)
				for i := 0; i < mc.count; i++ {
					metadata[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d-with-some-extra-content", i)
				}

				opts := storage.Options{}
				if mc.count > 0 {
					opts["metadata"] = metadata
				}

				b.Run(mc.name, func(b *testing.B) {
					b.SetBytes(1024)
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						key := uniqueKey("metadata")
						_, err := bucket.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", opts)
						if err != nil {
							b.Fatalf("write: %v", err)
						}
					}
				})
			}
		})
	}
}

// ============================================================================
// BUCKET OPERATIONS BENCHMARKS
// ============================================================================

func BenchmarkBucketOps(b *testing.B) {
	configs := getDriverConfigs(b)

	for _, cfg := range configs {
		if cfg.Skip {
			b.Run(cfg.Name, func(b *testing.B) {
				b.Skip(cfg.SkipMsg)
			})
			continue
		}

		b.Run(cfg.Name, func(b *testing.B) {
			ctx := context.Background()
			st, err := openStorage(ctx, cfg)
			if err != nil {
				b.Fatalf("open storage: %v", err)
			}
			defer st.Close()

			b.Run("CreateDelete", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					name := fmt.Sprintf("bench-bucket-%d-%d", time.Now().UnixNano(), i)
					_, err := st.CreateBucket(ctx, name, nil)
					if err != nil {
						b.Fatalf("create bucket: %v", err)
					}
					err = st.DeleteBucket(ctx, name, nil)
					if err != nil {
						b.Fatalf("delete bucket: %v", err)
					}
				}
			})

			b.Run("BucketInfo", func(b *testing.B) {
				bucket := st.Bucket(cfg.Bucket)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := bucket.Info(ctx)
					if err != nil {
						b.Fatalf("bucket info: %v", err)
					}
				}
			})
		})
	}
}

// ============================================================================
// REPORT GENERATION
// ============================================================================

func TestGenerateReport(t *testing.T) {
	if os.Getenv("BENCH_REPORT") != "1" {
		t.Skip("Set BENCH_REPORT=1 to generate report")
	}

	// This test should be run after benchmarks complete
	// It reads benchmark output and generates a markdown report

	reportPath := filepath.Join("..", "report", "benchmark_report.md")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0755); err != nil {
		t.Fatalf("create report dir: %v", err)
	}

	// Generate report from collected results
	report := generateMarkdownReport(benchResults)

	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	t.Logf("Report written to %s", reportPath)
}

func generateMarkdownReport(results []BenchResult) string {
	var sb strings.Builder

	sb.WriteString("# Storage Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Go Version: %s\n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("Platform: %s/%s\n\n", runtime.GOOS, runtime.GOARCH))

	sb.WriteString("## Summary\n\n")
	sb.WriteString("This report contains benchmark results for the storage package across multiple drivers.\n\n")

	// Group results by benchmark category
	categories := make(map[string][]BenchResult)
	for _, r := range results {
		parts := strings.Split(r.Benchmark, "/")
		if len(parts) > 0 {
			cat := parts[0]
			categories[cat] = append(categories[cat], r)
		}
	}

	// Sort categories
	var catNames []string
	for name := range categories {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	for _, cat := range catNames {
		sb.WriteString(fmt.Sprintf("## %s\n\n", cat))
		sb.WriteString("| Driver | Benchmark | ops/sec | MB/s | Allocs/op |\n")
		sb.WriteString("|--------|-----------|---------|------|----------|\n")

		for _, r := range categories[cat] {
			opsPerSec := 1e9 / r.NsPerOp
			sb.WriteString(fmt.Sprintf("| %s | %s | %.2f | %.2f | %d |\n",
				r.Driver, r.Benchmark, opsPerSec, r.MBPerSec, r.AllocsOp))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ParseBenchmarkOutput parses go test -bench output into results.
func ParseBenchmarkOutput(output string) []BenchResult {
	var results []BenchResult
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		// Parse: BenchmarkWrite/memory/Small_1KB-8    100000    10234 ns/op    1024 B/op    5 allocs/op
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		name := parts[0]
		// Extract driver from name
		nameParts := strings.Split(strings.TrimPrefix(name, "Benchmark"), "/")
		if len(nameParts) < 2 {
			continue
		}

		r := BenchResult{
			Benchmark: strings.Join(nameParts, "/"),
			Driver:    nameParts[0],
		}

		// Parse numeric fields
		for i := 1; i < len(parts); i++ {
			if parts[i] == "ns/op" && i > 0 {
				fmt.Sscanf(parts[i-1], "%f", &r.NsPerOp)
			}
			if parts[i] == "B/op" && i > 0 {
				fmt.Sscanf(parts[i-1], "%d", &r.BytesPerOp)
			}
			if parts[i] == "allocs/op" && i > 0 {
				fmt.Sscanf(parts[i-1], "%d", &r.AllocsOp)
			}
			if strings.HasSuffix(parts[i], "MB/s") {
				fmt.Sscanf(parts[i], "%f", &r.MBPerSec)
			}
		}

		results = append(results, r)
	}

	return results
}

// ============================================================================
// JSON OUTPUT FOR REPORT GENERATION
// ============================================================================

func TestOutputJSON(t *testing.T) {
	if os.Getenv("BENCH_JSON") != "1" {
		t.Skip("Set BENCH_JSON=1 to output JSON")
	}

	data, err := json.MarshalIndent(benchResults, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	jsonPath := filepath.Join("..", "report", "benchmark_results.json")
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err != nil {
		t.Fatalf("create report dir: %v", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("write json: %v", err)
	}

	t.Logf("JSON written to %s", jsonPath)
}
