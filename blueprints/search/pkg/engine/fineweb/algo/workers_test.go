package algo_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

// TestWorkerScaling tests how performance scales with worker count.
func TestWorkerScaling(t *testing.T) {
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

	// Pre-load data
	var batches []struct {
		docIDs []uint32
		texts  []string
	}
	var totalDocs int64
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		docIDs := make([]uint32, len(batch))
		texts := make([]string, len(batch))
		for i, doc := range batch {
			docIDs[i] = uint32(totalDocs + int64(i))
			texts[i] = doc.Text
		}
		batches = append(batches, struct {
			docIDs []uint32
			texts  []string
		}{docIDs, texts})
		totalDocs += int64(len(batch))
		if totalDocs >= 500000 {
			break
		}
	}
	t.Logf("Pre-loaded %d docs in %d batches", totalDocs, len(batches))

	// Test different worker counts
	for _, numWorkers := range []int{4, 8, 12, 16, 24, 32} {
		t.Run("workers_"+string(rune('0'+numWorkers/10))+string(rune('0'+numWorkers%10)), func(t *testing.T) {
			tmpDir := t.TempDir()
			indexer := algo.NewUltraIndexer(tmpDir, algo.UltraConfig{
				NumWorkers:  numWorkers,
				SegmentDocs: 500000,
			})

			start := time.Now()
			for _, b := range batches {
				indexer.AddBatch(b.docIDs, b.texts)
			}
			elapsed := time.Since(start)

			docsPerSec := float64(totalDocs) / elapsed.Seconds()
			t.Logf("%d workers: %d docs in %v (%.0f docs/sec)", numWorkers, totalDocs, elapsed, docsPerSec)
		})
	}
}
