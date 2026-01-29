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

// BenchmarkIndexerComparison compares all indexer variants.
func BenchmarkIndexerComparison(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		b.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Load test data (1M docs)
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	ctx := context.Background()

	var allTexts []string
	var allDocIDs []uint32
	var totalBytes int64
	targetDocs := 1000000

	b.Log("Loading test data...")
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			b.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
			allDocIDs = append(allDocIDs, uint32(len(allDocIDs)))
			totalBytes += int64(len(doc.Text))
		}
		if len(allTexts) >= targetDocs {
			break
		}
	}
	b.Logf("Loaded %d docs (%d MB), CPU count: %d", len(allTexts), totalBytes/(1024*1024), runtime.NumCPU())

	batchSize := 10000

	// Run each indexer variant
	indexers := []struct {
		name    string
		factory func(string) interface{ AddBatch([]uint32, []string) }
	}{
		{
			name: "MegaIndexer",
			factory: func(dir string) interface{ AddBatch([]uint32, []string) } {
				return algo.NewMegaIndexer(dir, algo.MegaConfig{
					NumWorkers:  runtime.NumCPU() * 4,
					SegmentDocs: 2000000,
				})
			},
		},
		{
			name: "TurboMegaIndexer",
			factory: func(dir string) interface{ AddBatch([]uint32, []string) } {
				return algo.NewTurboMegaIndexer(dir, algo.TurboMegaConfig{
					NumWorkers:  runtime.NumCPU() * 4,
					SegmentDocs: 2000000,
				})
			},
		},
		{
			name: "UltraBatchIndexer",
			factory: func(dir string) interface{ AddBatch([]uint32, []string) } {
				return algo.NewUltraBatchIndexer(dir, algo.UltraBatchConfig{
					NumWorkers: runtime.NumCPU() * 4,
				})
			},
		},
		{
			name: "FastBatchImporter",
			factory: func(dir string) interface{ AddBatch([]uint32, []string) } {
				return algo.NewFastBatchImporter()
			},
		},
	}

	for _, idx := range indexers {
		b.Run(idx.name, func(b *testing.B) {
			b.StopTimer()
			runtime.GC()

			tmpDir := b.TempDir()
			indexer := idx.factory(tmpDir)

			docIDs := make([]uint32, batchSize)
			texts := make([]string, batchSize)

			b.StartTimer()
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
			b.ReportMetric(docsPerSec, "docs/sec")
			b.Logf("%s: %.0f docs/sec in %v", idx.name, docsPerSec, elapsed)
		})
	}
}

// TestWorkerScalingTurbo tests how TurboMegaIndexer scales with workers.
func TestWorkerScalingTurbo(t *testing.T) {
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
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
		}
		if len(allTexts) >= 1000000 {
			break
		}
	}
	t.Logf("Loaded %d docs, CPU: %d", len(allTexts), runtime.NumCPU())

	batchSize := 10000
	workerCounts := []int{
		runtime.NumCPU(),
		runtime.NumCPU() * 2,
		runtime.NumCPU() * 4,
		runtime.NumCPU() * 6,
		runtime.NumCPU() * 8,
	}

	for _, numWorkers := range workerCounts {
		t.Run("workers_"+string(rune('0'+numWorkers/100))+string(rune('0'+(numWorkers/10)%10))+string(rune('0'+numWorkers%10)), func(t *testing.T) {
			runtime.GC()

			tmpDir := t.TempDir()
			indexer := algo.NewTurboMegaIndexer(tmpDir, algo.TurboMegaConfig{
				NumWorkers:  numWorkers,
				SegmentDocs: 2000000,
			})

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
			t.Logf("%d workers: %.0f docs/sec", numWorkers, docsPerSec)
		})
	}
}

// BenchmarkTokenizers compares tokenization functions.
func BenchmarkTokenizers(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		b.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Load test data
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	ctx := context.Background()

	var texts []string
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 4) {
		if err != nil {
			b.Fatal(err)
		}
		for _, doc := range batch {
			texts = append(texts, doc.Text)
		}
		if len(texts) >= 100000 {
			break
		}
	}
	b.Logf("Loaded %d docs for tokenization benchmark", len(texts))

	b.Run("TokenizeMega", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeMega(text, freqs)
			}
		}
	})

	b.Run("TokenizeMegaV2", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeMegaV2(text, freqs)
			}
		}
	})

	b.Run("TokenizeMegaV3", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeMegaV3(text, freqs)
			}
		}
	})

	b.Run("TokenizeTurboV3", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeTurboV3(text, freqs)
			}
		}
	})

	b.Run("TokenizeHyper", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeHyper(text, freqs)
			}
		}
	})
}
