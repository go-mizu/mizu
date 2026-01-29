package algo_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

// TestProfileFullPipeline profiles the full indexing pipeline
func TestProfileFullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Create CPU profile
	cpuFile, err := os.Create("/tmp/cpu_profile.pprof")
	if err != nil {
		t.Fatal(err)
	}
	defer cpuFile.Close()

	ctx := context.Background()
	numWorkers := runtime.NumCPU() * 5
	targetDocs := 500000

	t.Logf("CPU cores: %d, Workers: %d, Target: %d docs", runtime.NumCPU(), numWorkers, targetDocs)

	// Start CPU profiling
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		t.Fatal(err)
	}

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	indexer := algo.NewStreamingOptimizedIndexer(numWorkers)

	start := time.Now()
	var docCount int

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			indexer.Add(doc.Text)
			docCount++
		}
		if docCount >= targetDocs {
			break
		}
	}
	indexer.Flush()
	elapsed := time.Since(start)

	// Stop CPU profiling
	pprof.StopCPUProfile()

	rate := float64(docCount) / elapsed.Seconds()
	t.Logf("Indexed %d docs in %v = %.0f docs/sec", docCount, elapsed, rate)
	t.Logf("CPU profile saved to /tmp/cpu_profile.pprof")
	t.Log("Run: go tool pprof -http=:8080 /tmp/cpu_profile.pprof")
}

// TestProfileTokenizationOnly profiles just the tokenization phase
func TestProfileTokenizationOnly(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()

	// Load data first (outside profiling)
	var allTexts []string
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
		}
	}
	t.Logf("Loaded %d docs", len(allTexts))

	// Create CPU profile
	cpuFile, err := os.Create("/tmp/cpu_tokenize.pprof")
	if err != nil {
		t.Fatal(err)
	}
	defer cpuFile.Close()

	numWorkers := runtime.NumCPU()
	runtime.GC()

	// Start CPU profiling
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		t.Fatal(err)
	}

	start := time.Now()

	// Run tokenization multiple times for better profiling
	for iter := 0; iter < 10; iter++ {
		batchSize := (len(allTexts) + numWorkers - 1) / numWorkers
		done := make(chan struct{}, numWorkers)

		for w := 0; w < numWorkers; w++ {
			startIdx := w * batchSize
			endIdx := startIdx + batchSize
			if endIdx > len(allTexts) {
				endIdx = len(allTexts)
			}
			if startIdx >= endIdx {
				done <- struct{}{}
				continue
			}

			go func(s, e int) {
				table := algo.NewFixedHashTable(4096)
				for i := s; i < e; i++ {
					algo.FixedTokenize(allTexts[i], table)
				}
				done <- struct{}{}
			}(startIdx, endIdx)
		}

		for w := 0; w < numWorkers; w++ {
			<-done
		}
	}

	elapsed := time.Since(start)
	pprof.StopCPUProfile()

	totalDocs := len(allTexts) * 10
	rate := float64(totalDocs) / elapsed.Seconds()
	t.Logf("Tokenized %d docs in %v = %.0f docs/sec", totalDocs, elapsed, rate)
	t.Logf("CPU profile saved to /tmp/cpu_tokenize.pprof")
	t.Log("Run: go tool pprof -http=:8080 /tmp/cpu_tokenize.pprof")
}

// TestProfileParquetReading profiles just the parquet reading
func TestProfileParquetReading(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Create CPU profile
	cpuFile, err := os.Create("/tmp/cpu_parquet.pprof")
	if err != nil {
		t.Fatal(err)
	}
	defer cpuFile.Close()

	ctx := context.Background()
	targetDocs := 500000

	t.Logf("Target: %d docs", targetDocs)

	// Start CPU profiling
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		t.Fatal(err)
	}

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	start := time.Now()
	var docCount int
	var totalBytes int64

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			totalBytes += int64(len(doc.Text))
			docCount++
		}
		if docCount >= targetDocs {
			break
		}
	}

	elapsed := time.Since(start)
	pprof.StopCPUProfile()

	rate := float64(docCount) / elapsed.Seconds()
	mbps := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)
	t.Logf("Read %d docs (%d MB) in %v = %.0f docs/sec (%.1f MB/s)",
		docCount, totalBytes/(1024*1024), elapsed, rate, mbps)
	t.Logf("CPU profile saved to /tmp/cpu_parquet.pprof")
	t.Log("Run: go tool pprof -http=:8080 /tmp/cpu_parquet.pprof")
}
