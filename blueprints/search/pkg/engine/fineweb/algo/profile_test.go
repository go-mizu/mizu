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

// TestProfileUltraIndexer profiles the UltraIndexer to find bottlenecks.
func TestProfileUltraIndexer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping profiling test in short mode")
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Create CPU profile
	cpuFile, err := os.Create("/tmp/ultra_cpu.prof")
	if err != nil {
		t.Fatal(err)
	}
	defer cpuFile.Close()

	tmpDir := t.TempDir()
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	ctx := context.Background()

	indexer := algo.NewUltraIndexer(tmpDir, algo.UltraConfig{
		NumWorkers:  runtime.NumCPU() * 2,
		SegmentDocs: 500000,
	})

	runtime.GC()
	pprof.StartCPUProfile(cpuFile)

	start := time.Now()
	var docCount int64

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}

		docIDs := make([]uint32, len(batch))
		texts := make([]string, len(batch))
		for i, doc := range batch {
			docIDs[i] = uint32(docCount + int64(i))
			texts[i] = doc.Text
		}

		indexer.AddBatch(docIDs, texts)
		docCount += int64(len(batch))

		if docCount >= 500000 {
			break
		}
	}

	pprof.StopCPUProfile()
	elapsed := time.Since(start)

	docsPerSec := float64(docCount) / elapsed.Seconds()
	t.Logf("UltraIndexer: %d docs in %v (%.0f docs/sec)", docCount, elapsed, docsPerSec)
	t.Logf("CPU profile: /tmp/ultra_cpu.prof")
	t.Logf("Run: go tool pprof -http=:8080 /tmp/ultra_cpu.prof")
}

// BenchmarkTokenization benchmarks different tokenization approaches.
func BenchmarkTokenization(b *testing.B) {
	texts := []struct {
		name string
		text string
	}{
		{"short", "The quick brown fox jumps over the lazy dog."},
		{"medium", generateText(500)},
		{"long", generateText(2000)},
	}

	for _, tc := range texts {
		b.Run(tc.name, func(b *testing.B) {
			freqs := make(map[uint64]uint16, 256)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				algo.TokenizeToHashReuse(tc.text, freqs)
			}
		})
	}
}

func generateText(words int) string {
	wordList := []string{
		"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"hello", "world", "testing", "performance", "optimization", "golang",
		"search", "index", "document", "text", "word", "token", "hash", "map",
	}
	result := ""
	for i := 0; i < words; i++ {
		if i > 0 {
			result += " "
		}
		result += wordList[i%len(wordList)]
	}
	return result
}

// TestBottleneckAnalysis runs targeted profiling on specific components.
func TestBottleneckAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	ctx := context.Background()

	// Collect sample texts
	var texts []string
	var count int
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			texts = append(texts, doc.Text)
			count++
			if count >= 100000 {
				break
			}
		}
		if count >= 100000 {
			break
		}
	}
	t.Logf("Collected %d texts for analysis", len(texts))

	// Benchmark pure tokenization
	t.Run("tokenization", func(t *testing.T) {
		start := time.Now()
		freqs := make(map[uint64]uint16, 512)
		var totalTokens int
		for _, text := range texts {
			n := algo.TokenizeToHashReuse(text, freqs)
			totalTokens += n
		}
		elapsed := time.Since(start)
		t.Logf("Tokenization: %d texts, %d tokens in %v (%.0f texts/sec)",
			len(texts), totalTokens, elapsed, float64(len(texts))/elapsed.Seconds())
	})

	// Benchmark parquet reading only
	t.Run("parquet_read", func(t *testing.T) {
		reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
		start := time.Now()
		var readCount int64
		for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
			if err != nil {
				t.Fatal(err)
			}
			readCount += int64(len(batch))
			if readCount >= 500000 {
				break
			}
		}
		elapsed := time.Since(start)
		t.Logf("Parquet read: %d docs in %v (%.0f docs/sec)",
			readCount, elapsed, float64(readCount)/elapsed.Seconds())
	})
}
