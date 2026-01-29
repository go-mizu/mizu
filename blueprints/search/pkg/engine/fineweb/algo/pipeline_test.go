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

// TestPipelinedImport tests the pipelined import approach.
func TestPipelinedImport(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create indexer
	indexer := algo.NewUltraIndexer(tmpDir, algo.UltraConfig{
		NumWorkers:  runtime.NumCPU(),
		SegmentDocs: 500000,
	})

	// Buffered channel for batches
	type batch struct {
		docIDs []uint32
		texts  []string
	}
	batchCh := make(chan batch, 64)

	// Producer goroutine
	go func() {
		defer close(batchCh)
		reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
		var docNum uint32
		var total int64

		for b, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
			if err != nil {
				t.Errorf("Read error: %v", err)
				return
			}

			docIDs := make([]uint32, len(b))
			texts := make([]string, len(b))
			for i, doc := range b {
				docIDs[i] = docNum
				texts[i] = doc.Text
				docNum++
			}

			batchCh <- batch{docIDs: docIDs, texts: texts}
			total += int64(len(b))

			if total >= 500000 {
				return
			}
		}
	}()

	// Profile CPU
	cpuFile, _ := os.Create("/tmp/pipeline_cpu.prof")
	pprof.StartCPUProfile(cpuFile)
	defer func() {
		pprof.StopCPUProfile()
		cpuFile.Close()
	}()

	// Consumer - index batches
	start := time.Now()
	var indexed int64

	for b := range batchCh {
		indexer.AddBatch(b.docIDs, b.texts)
		indexed += int64(len(b.texts))
	}

	elapsed := time.Since(start)
	docsPerSec := float64(indexed) / elapsed.Seconds()
	t.Logf("Pipelined: %d docs in %v (%.0f docs/sec)", indexed, elapsed, docsPerSec)
	t.Logf("Profile: /tmp/pipeline_cpu.prof")
}

// TestReadOnlyBenchmark measures just parquet reading speed.
func TestReadOnlyBenchmark(t *testing.T) {
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

	// Test different reader counts
	for _, numReaders := range []int{4, 8, 16, 32} {
		t.Run("readers_"+string(rune('0'+numReaders/10))+string(rune('0'+numReaders%10)), func(t *testing.T) {
			start := time.Now()
			var count int64
			var textBytes int64

			for batch, err := range reader.ReadTextsOnlyParallel(ctx, numReaders) {
				if err != nil {
					t.Fatal(err)
				}
				for _, doc := range batch {
					textBytes += int64(len(doc.Text))
				}
				count += int64(len(batch))
				if count >= 500000 {
					break
				}
			}

			elapsed := time.Since(start)
			docsPerSec := float64(count) / elapsed.Seconds()
			mbPerSec := float64(textBytes) / elapsed.Seconds() / 1024 / 1024
			t.Logf("%d readers: %d docs in %v (%.0f docs/sec, %.1f MB/sec)",
				numReaders, count, elapsed, docsPerSec, mbPerSec)
		})
	}
}

// TestIndexOnlyBenchmark measures just indexing speed (no I/O).
func TestIndexOnlyBenchmark(t *testing.T) {
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
	t.Log("Pre-loading data...")
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
		if totalDocs >= 1000000 {
			break
		}
	}
	t.Logf("Pre-loaded %d docs in %d batches", totalDocs, len(batches))

	// Benchmark pure indexing
	tmpDir := t.TempDir()
	indexer := algo.NewUltraIndexer(tmpDir, algo.UltraConfig{
		NumWorkers:  runtime.NumCPU(),
		SegmentDocs: 1000000,
	})
	runtime.GC()

	// Profile
	cpuFile, _ := os.Create("/tmp/indexonly_cpu.prof")
	pprof.StartCPUProfile(cpuFile)

	start := time.Now()
	for _, b := range batches {
		indexer.AddBatch(b.docIDs, b.texts)
	}
	elapsed := time.Since(start)

	pprof.StopCPUProfile()
	cpuFile.Close()

	docsPerSec := float64(totalDocs) / elapsed.Seconds()
	t.Logf("Index only: %d docs in %v (%.0f docs/sec)", totalDocs, elapsed, docsPerSec)
	t.Logf("Profile: /tmp/indexonly_cpu.prof")
}
