package algo_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/DataDog/zstd"
	klauspost "github.com/klauspost/compress/zstd"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

// TestZstdDecompressionComparison compares pure Go vs CGO zstd decompression
func TestZstdDecompressionComparison(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// Create test data - simulate typical compressed parquet column data
	// Real parquet uses zstd for column compression
	testSizes := []int{10_000, 100_000, 1_000_000}

	for _, size := range testSizes {
		t.Run("", func(t *testing.T) {
			// Generate compressible text data (similar to FineWeb text)
			original := generateCompressibleData(size)

			// Compress with CGO zstd (reference)
			compressed, err := zstd.Compress(nil, original)
			if err != nil {
				t.Fatal(err)
			}

			compressionRatio := float64(len(compressed)) / float64(len(original))
			t.Logf("Size: %d bytes, Compressed: %d bytes (%.2f%% ratio)",
				len(original), len(compressed), compressionRatio*100)

			iterations := 1000
			if size >= 1_000_000 {
				iterations = 100
			}

			// Benchmark CGO zstd (DataDog/zstd)
			runtime.GC()
			start := time.Now()
			for i := 0; i < iterations; i++ {
				_, err := zstd.Decompress(nil, compressed)
				if err != nil {
					t.Fatal(err)
				}
			}
			cgoTime := time.Since(start)
			cgoMBps := float64(len(original)*iterations) / cgoTime.Seconds() / (1024 * 1024)

			// Benchmark Pure Go zstd (klauspost/compress)
			decoder, err := klauspost.NewReader(nil)
			if err != nil {
				t.Fatal(err)
			}
			defer decoder.Close()

			runtime.GC()
			start = time.Now()
			for i := 0; i < iterations; i++ {
				err := decoder.Reset(bytes.NewReader(compressed))
				if err != nil {
					t.Fatal(err)
				}
				_, err = io.ReadAll(decoder)
				if err != nil {
					t.Fatal(err)
				}
			}
			pureGoTime := time.Since(start)
			pureGoMBps := float64(len(original)*iterations) / pureGoTime.Seconds() / (1024 * 1024)

			speedup := cgoMBps / pureGoMBps
			t.Logf("  CGO zstd:     %8.2f MB/sec", cgoMBps)
			t.Logf("  Pure Go zstd: %8.2f MB/sec", pureGoMBps)
			t.Logf("  Speedup:      %.2fx", speedup)
		})
	}
}

// TestParquetDecompressionOverhead measures actual parquet decompression
func TestParquetDecompressionOverhead(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	numWorkers := 8

	t.Logf("CPU cores: %d, Workers: %d", runtime.NumCPU(), numWorkers)

	// Read data and measure total time
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	var docCount int
	var totalBytes int64

	runtime.GC()
	start := time.Now()

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, numWorkers) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			docCount++
			totalBytes += int64(len(doc.Text))
		}
	}
	elapsed := time.Since(start)
	rate := float64(docCount) / elapsed.Seconds()
	throughput := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)

	t.Logf("Read %d docs (%d MB) in %v", docCount, totalBytes/(1024*1024), elapsed)
	t.Logf("Rate: %.0f docs/sec, %.2f MB/sec", rate, throughput)
	t.Logf("Gap to 1M docs/sec: %.1fx needed", 1000000.0/rate)
}

// TestParallelZstdDecompression benchmarks parallel decompression
func TestParallelZstdDecompression(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// Simulate 8 parallel decompression workers (like parquet row groups)
	numWorkers := 8
	dataSize := 1_000_000 // 1MB per chunk (typical row group text column)
	numChunks := 100

	t.Logf("Workers: %d, Chunks: %d, Size per chunk: %d bytes", numWorkers, numChunks, dataSize)

	// Prepare compressed chunks
	original := generateCompressibleData(dataSize)
	compressedChunks := make([][]byte, numChunks)
	for i := 0; i < numChunks; i++ {
		compressed, _ := zstd.Compress(nil, original)
		compressedChunks[i] = compressed
	}

	// CGO zstd parallel decompression
	runtime.GC()
	start := time.Now()

	var wg sync.WaitGroup
	chunksPerWorker := (numChunks + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		startIdx := w * chunksPerWorker
		endIdx := startIdx + chunksPerWorker
		if endIdx > numChunks {
			endIdx = numChunks
		}
		if startIdx >= endIdx {
			break
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for i := s; i < e; i++ {
				zstd.Decompress(nil, compressedChunks[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	cgoTime := time.Since(start)
	cgoMBps := float64(dataSize*numChunks) / cgoTime.Seconds() / (1024 * 1024)

	// Pure Go zstd parallel decompression
	runtime.GC()
	start = time.Now()

	// Pre-create decoders per worker
	decoders := make([]*klauspost.Decoder, numWorkers)
	for i := 0; i < numWorkers; i++ {
		decoders[i], _ = klauspost.NewReader(nil)
	}

	for w := 0; w < numWorkers; w++ {
		startIdx := w * chunksPerWorker
		endIdx := startIdx + chunksPerWorker
		if endIdx > numChunks {
			endIdx = numChunks
		}
		if startIdx >= endIdx {
			break
		}

		wg.Add(1)
		go func(workerID, s, e int) {
			defer wg.Done()
			dec := decoders[workerID]
			for i := s; i < e; i++ {
				dec.Reset(bytes.NewReader(compressedChunks[i]))
				io.ReadAll(dec)
			}
		}(w, startIdx, endIdx)
	}
	wg.Wait()
	pureGoTime := time.Since(start)
	pureGoMBps := float64(dataSize*numChunks) / pureGoTime.Seconds() / (1024 * 1024)

	for _, d := range decoders {
		d.Close()
	}

	speedup := cgoMBps / pureGoMBps
	t.Logf("CGO zstd parallel:     %.2f MB/sec", cgoMBps)
	t.Logf("Pure Go zstd parallel: %.2f MB/sec", pureGoMBps)
	t.Logf("Speedup: %.2fx", speedup)

	// Estimate docs per second improvement
	// If decompression is 78% of I/O time and CGO is 1.5x faster:
	// New I/O time = 0.22 + 0.78/1.5 = 0.22 + 0.52 = 0.74 of original
	// Improvement = 1/0.74 = 1.35x
	if speedup > 1.0 {
		ioSpeedup := 1.0 / (0.22 + 0.78/speedup)
		t.Logf("\nEstimated I/O improvement: %.2fx", ioSpeedup)
		t.Logf("Current rate: ~150k docs/sec")
		t.Logf("Projected rate: ~%.0f docs/sec", 150000.0*ioSpeedup)
	}
}

// generateCompressibleData creates text-like data similar to FineWeb documents
func generateCompressibleData(size int) []byte {
	// Simulate typical Vietnamese/English web text patterns
	patterns := []string{
		"Đây là một văn bản tiếng Việt mẫu. ",
		"This is sample English text for testing. ",
		"Lorem ipsum dolor sit amet consectetur. ",
		"Các từ phổ biến trong tiếng Việt. ",
		"Common words in English text documents. ",
		"The quick brown fox jumps over lazy dog. ",
		"Nội dung web thường có nhiều từ lặp lại. ",
	}

	var result bytes.Buffer
	patternIdx := 0
	for result.Len() < size {
		result.WriteString(patterns[patternIdx])
		patternIdx = (patternIdx + 1) % len(patterns)
	}
	return result.Bytes()[:size]
}
