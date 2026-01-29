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

// TestWorkerScalingMega tests how MegaIndexer scales with different worker counts.
func TestWorkerScalingMega(t *testing.T) {
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
	t.Logf("CPU count: %d", runtime.NumCPU())

	batchSize := 10000

	// Test different worker counts
	workerCounts := []int{
		runtime.NumCPU(),
		runtime.NumCPU() * 2,
		runtime.NumCPU() * 3,
		runtime.NumCPU() * 4,
		32, 48, 64,
	}

	for _, numWorkers := range workerCounts {
		t.Run("workers_"+string(rune('0'+numWorkers/10))+string(rune('0'+numWorkers%10)), func(t *testing.T) {
			runtime.GC()

			tmpDir := t.TempDir()
			indexer := algo.NewMegaIndexer(tmpDir, algo.MegaConfig{
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
