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

// TestMegaProfile profiles all components to find remaining bottlenecks.
func TestMegaProfile(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Pre-load data to isolate indexing performance
	t.Log("Loading data...")
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	ctx := context.Background()

	var allTexts []string
	var totalBytes int64
	loadStart := time.Now()
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
			totalBytes += int64(len(doc.Text))
		}
		if len(allTexts) >= 1000000 { // 1M docs for profiling
			break
		}
	}
	t.Logf("Loaded %d docs (%d MB) in %v", len(allTexts), totalBytes/(1024*1024), time.Since(loadStart))

	// Force GC before profiling
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Start CPU profile
	cpuFile, _ := os.Create("/tmp/mega_cpu.prof")
	pprof.StartCPUProfile(cpuFile)
	defer func() {
		pprof.StopCPUProfile()
		cpuFile.Close()
		t.Log("CPU profile written to /tmp/mega_cpu.prof")
		t.Log("Run: go tool pprof -http=:8080 /tmp/mega_cpu.prof")
	}()

	// Create indexer - using MegaIndexer (fastest)
	tmpDir := t.TempDir()
	numWorkers := runtime.NumCPU() * 2
	indexer := algo.NewMegaIndexer(tmpDir, algo.MegaConfig{
		NumWorkers:  numWorkers,
		SegmentDocs: 2000000,
	})

	// Index in batches
	batchSize := 10000
	docIDs := make([]uint32, batchSize)
	texts := make([]string, batchSize)

	start := time.Now()
	for i := 0; i < len(allTexts); i += batchSize {
		end := i + batchSize
		if end > len(allTexts) {
			end = len(allTexts)
		}

		n := end - i
		for j := 0; j < n; j++ {
			docIDs[j] = uint32(i + j)
			texts[j] = allTexts[i+j]
		}

		indexer.AddBatch(docIDs[:n], texts[:n])
	}
	indexElapsed := time.Since(start)

	docsPerSec := float64(len(allTexts)) / indexElapsed.Seconds()
	mbPerSec := float64(totalBytes) / indexElapsed.Seconds() / (1024 * 1024)
	t.Logf("Indexed %d docs in %v (%.0f docs/sec, %.1f MB/sec)", len(allTexts), indexElapsed, docsPerSec, mbPerSec)
	t.Logf("Workers: %d, Target: 1M docs/sec", numWorkers)
	t.Logf("Gap to target: %.1fx improvement needed", 1000000/docsPerSec)
}

// TestTokenizationProfile profiles just tokenization.
func TestTokenizationProfile(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Load fewer docs but focus on tokenization
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	ctx := context.Background()

	var allTexts []string
	var totalBytes int64
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
			totalBytes += int64(len(doc.Text))
		}
		if len(allTexts) >= 500000 {
			break
		}
	}
	t.Logf("Loaded %d docs (%d MB)", len(allTexts), totalBytes/(1024*1024))

	runtime.GC()

	// Profile just tokenization
	cpuFile, _ := os.Create("/tmp/tokenize_cpu.prof")
	pprof.StartCPUProfile(cpuFile)
	defer func() {
		pprof.StopCPUProfile()
		cpuFile.Close()
		t.Log("CPU profile written to /tmp/tokenize_cpu.prof")
	}()

	freqs := make(map[uint64]uint16, 256)
	var totalTokens int64
	start := time.Now()

	for _, text := range allTexts {
		n := algo.TokenizeToHashReuse(text, freqs)
		totalTokens += int64(n)
	}

	elapsed := time.Since(start)
	docsPerSec := float64(len(allTexts)) / elapsed.Seconds()
	tokensPerSec := float64(totalTokens) / elapsed.Seconds()
	bytesPerSec := float64(totalBytes) / elapsed.Seconds()

	t.Logf("Tokenized %d docs in %v", len(allTexts), elapsed)
	t.Logf("  %.0f docs/sec", docsPerSec)
	t.Logf("  %.0f tokens/sec", tokensPerSec)
	t.Logf("  %.0f MB/sec", bytesPerSec/(1024*1024))
}
