package algo_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

// TestCompareTokenizers compares different tokenization implementations.
func TestCompareTokenizers(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Load test data
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
		if len(allTexts) >= 200000 {
			break
		}
	}
	t.Logf("Loaded %d texts (%d MB)", len(allTexts), totalBytes/(1024*1024))

	runtime.GC()

	// Test original tokenizer
	t.Run("Original", func(t *testing.T) {
		freqs := make(map[uint64]uint16, 256)
		var totalTokens int64

		start := time.Now()
		for _, text := range allTexts {
			n := algo.TokenizeToHashReuse(text, freqs)
			totalTokens += int64(n)
		}
		elapsed := time.Since(start)

		docsPerSec := float64(len(allTexts)) / elapsed.Seconds()
		mbPerSec := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)
		t.Logf("Original: %.0f docs/sec, %.1f MB/sec, %d tokens", docsPerSec, mbPerSec, totalTokens)
	})

	// Test mega tokenizer
	t.Run("Mega", func(t *testing.T) {
		freqs := make(map[uint64]uint16, 256)
		var totalTokens int64

		start := time.Now()
		for _, text := range allTexts {
			n := algo.TokenizeMega(text, freqs)
			totalTokens += int64(n)
		}
		elapsed := time.Since(start)

		docsPerSec := float64(len(allTexts)) / elapsed.Seconds()
		mbPerSec := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)
		t.Logf("Mega: %.0f docs/sec, %.1f MB/sec, %d tokens", docsPerSec, mbPerSec, totalTokens)
	})

	// Test mega v2 tokenizer (no branching)
	t.Run("MegaV2", func(t *testing.T) {
		freqs := make(map[uint64]uint16, 256)
		var totalTokens int64

		start := time.Now()
		for _, text := range allTexts {
			n := algo.TokenizeMegaV2(text, freqs)
			totalTokens += int64(n)
		}
		elapsed := time.Since(start)

		docsPerSec := float64(len(allTexts)) / elapsed.Seconds()
		mbPerSec := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)
		t.Logf("MegaV2: %.0f docs/sec, %.1f MB/sec, %d tokens", docsPerSec, mbPerSec, totalTokens)
	})

	// Test mega v3 tokenizer
	t.Run("MegaV3", func(t *testing.T) {
		freqs := make(map[uint64]uint16, 256)
		var totalTokens int64

		start := time.Now()
		for _, text := range allTexts {
			n := algo.TokenizeMegaV3(text, freqs)
			totalTokens += int64(n)
		}
		elapsed := time.Since(start)

		docsPerSec := float64(len(allTexts)) / elapsed.Seconds()
		mbPerSec := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)
		t.Logf("MegaV3: %.0f docs/sec, %.1f MB/sec, %d tokens", docsPerSec, mbPerSec, totalTokens)
	})
}

// TestCompareIndexers compares different indexer implementations.
func TestCompareIndexers(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Load test data
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
		if len(allTexts) >= 1000000 {
			break
		}
	}
	t.Logf("Loaded %d docs (%d MB)", len(allTexts), totalBytes/(1024*1024))

	numWorkers := runtime.NumCPU() * 2
	batchSize := 10000

	runIndexer := func(name string, addBatch func([]uint32, []string)) time.Duration {
		runtime.GC()

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

			addBatch(docIDs[:n], texts[:n])
		}
		elapsed := time.Since(start)

		docsPerSec := float64(len(allTexts)) / elapsed.Seconds()
		mbPerSec := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)
		t.Logf("%s: %.0f docs/sec, %.1f MB/sec", name, docsPerSec, mbPerSec)
		return elapsed
	}

	// Test UltraIndexer (current best)
	t.Run("UltraIndexer", func(t *testing.T) {
		tmpDir := t.TempDir()
		indexer := algo.NewUltraIndexer(tmpDir, algo.UltraConfig{
			NumWorkers:  numWorkers,
			SegmentDocs: 2000000,
		})
		runIndexer("UltraIndexer", indexer.AddBatch)
	})

	// Test RocketIndexer (lock-free workers)
	t.Run("RocketIndexer", func(t *testing.T) {
		tmpDir := t.TempDir()
		indexer := algo.NewRocketIndexer(tmpDir, algo.RocketConfig{
			NumWorkers:  numWorkers,
			SegmentDocs: 2000000,
		})
		runIndexer("RocketIndexer", indexer.AddBatch)
	})

	// Test Rocket256Indexer (256 shards)
	t.Run("Rocket256Indexer", func(t *testing.T) {
		tmpDir := t.TempDir()
		indexer := algo.NewRocket256Indexer(tmpDir, algo.RocketConfig{
			NumWorkers:  numWorkers,
			SegmentDocs: 2000000,
		})
		runIndexer("Rocket256Indexer", indexer.AddBatch)
	})

	// Test MegaIndexer
	t.Run("MegaIndexer", func(t *testing.T) {
		tmpDir := t.TempDir()
		indexer := algo.NewMegaIndexer(tmpDir, algo.MegaConfig{
			NumWorkers:  numWorkers,
			SegmentDocs: 2000000,
		})
		runIndexer("MegaIndexer", indexer.AddBatch)
	})

	// Test Mega256Indexer (256 shards)
	t.Run("Mega256Indexer", func(t *testing.T) {
		tmpDir := t.TempDir()
		indexer := algo.NewMega256Indexer(tmpDir, algo.MegaConfig{
			NumWorkers:  numWorkers,
			SegmentDocs: 2000000,
		})
		runIndexer("Mega256Indexer", indexer.AddBatch)
	})
}

// TestMegaIndexer tests the full MegaIndexer pipeline.
func TestMegaIndexer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Load test data
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
		if len(allTexts) >= 1000000 {
			break
		}
	}
	t.Logf("Loaded %d docs (%d MB)", len(allTexts), totalBytes/(1024*1024))

	runtime.GC()

	// Test MegaIndexer
	tmpDir := t.TempDir()
	numWorkers := runtime.NumCPU() * 2
	indexer := algo.NewMegaIndexer(tmpDir, algo.MegaConfig{
		NumWorkers:  numWorkers,
		SegmentDocs: 2000000,
	})

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
	elapsed := time.Since(start)

	docsPerSec := float64(len(allTexts)) / elapsed.Seconds()
	mbPerSec := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)
	t.Logf("MegaIndexer: %d docs in %v (%.0f docs/sec, %.1f MB/sec)", len(allTexts), elapsed, docsPerSec, mbPerSec)
	t.Logf("Workers: %d, Target: 1M docs/sec", numWorkers)
	t.Logf("Gap to target: %.1fx improvement needed", 1000000/docsPerSec)
}
