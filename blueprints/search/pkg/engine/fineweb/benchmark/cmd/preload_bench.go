//go:build ignore

// Preload benchmark - measures indexing speed without I/O overhead
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"

	// Import drivers
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_balanced"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_compact"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_production"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_speed"
)

func main() {
	parquetPath := flag.String("parquet", "", "Parquet file/directory path")
	dataDir := flag.String("data", "", "Data directory for indexes")
	drivers := flag.String("drivers", "fts_speed,fts_balanced,fts_compact,fts_production", "Comma-separated drivers")
	flag.Parse()

	if *parquetPath == "" {
		home, _ := os.UserHomeDir()
		*parquetPath = filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")
	}
	if *dataDir == "" {
		home, _ := os.UserHomeDir()
		*dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	ctx := context.Background()

	// Phase 1: Pre-load all documents into memory
	fmt.Println("=== Phase 1: Pre-loading documents into memory ===")
	loadStart := time.Now()

	var allDocs []fineweb.Document
	reader := fineweb.NewParquetReader(*parquetPath)

	for doc, err := range reader.ReadAll(ctx) {
		if err != nil {
			fmt.Printf("Error reading document: %v\n", err)
			break
		}
		allDocs = append(allDocs, doc)
	}

	loadDuration := time.Since(loadStart)
	fmt.Printf("Loaded %d documents in %v (%.0f docs/sec)\n\n",
		len(allDocs), loadDuration, float64(len(allDocs))/loadDuration.Seconds())

	// Force GC before benchmarking
	runtime.GC()

	// Phase 2: Benchmark each driver with pre-loaded data
	fmt.Println("=== Phase 2: Benchmarking indexing (no I/O) ===")
	fmt.Println()

	driverList := []string{"fts_speed", "fts_balanced", "fts_compact", "fts_production"}
	if *drivers != "" {
		driverList = splitDrivers(*drivers)
	}

	results := make(map[string]float64)

	for _, driverName := range driverList {
		// Clean existing index
		indexDir := filepath.Join(*dataDir, "vie_Latn."+driverName)
		os.RemoveAll(indexDir)

		// Open driver
		driver, err := fineweb.Open(driverName, fineweb.DriverConfig{
			DataDir:  *dataDir,
			Language: "vie_Latn",
		})
		if err != nil {
			fmt.Printf("%s: Error opening: %v\n", driverName, err)
			continue
		}

		indexer, ok := driver.(fineweb.Indexer)
		if !ok {
			fmt.Printf("%s: Not an indexer\n", driverName)
			driver.Close()
			continue
		}

		// Create iterator from pre-loaded docs
		docIter := func(yield func(fineweb.Document, error) bool) {
			for _, doc := range allDocs {
				if !yield(doc, nil) {
					return
				}
			}
		}

		// Benchmark indexing only
		runtime.GC()
		indexStart := time.Now()

		err = indexer.Import(ctx, docIter, nil)
		if err != nil {
			fmt.Printf("%s: Error indexing: %v\n", driverName, err)
			driver.Close()
			continue
		}

		indexDuration := time.Since(indexStart)
		docsPerSec := float64(len(allDocs)) / indexDuration.Seconds()
		results[driverName] = docsPerSec

		// Get index size
		var indexSize int64
		filepath.Walk(indexDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				indexSize += info.Size()
			}
			return nil
		})

		fmt.Printf("%s:\n", driverName)
		fmt.Printf("  Indexing: %d docs in %v (%.0f docs/sec)\n",
			len(allDocs), indexDuration, docsPerSec)
		fmt.Printf("  Index size: %.2f MB\n", float64(indexSize)/(1024*1024))

		if docsPerSec >= 50000 {
			fmt.Printf("  Status: PASS (>= 50k docs/sec)\n")
		} else {
			fmt.Printf("  Status: %.1f%% of target\n", docsPerSec/50000*100)
		}
		fmt.Println()

		driver.Close()
	}

	// Summary
	fmt.Println("=== Summary (Indexing only, no I/O) ===")
	fmt.Println()
	fmt.Printf("| Driver | docs/sec | %% of 50k target |\n")
	fmt.Printf("|--------|----------|----------------|\n")
	for _, name := range driverList {
		if speed, ok := results[name]; ok {
			status := "PASS"
			if speed < 50000 {
				status = fmt.Sprintf("%.1f%%", speed/50000*100)
			}
			fmt.Printf("| %s | %.0f | %s |\n", name, speed, status)
		}
	}
}

func splitDrivers(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	return result
}
