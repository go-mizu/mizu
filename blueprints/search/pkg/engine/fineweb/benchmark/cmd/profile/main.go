// Command profile runs profiling on the fts_highthroughput indexer.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"

	// Import the driver
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_highthroughput"
)

func main() {
	parquetDir := flag.String("parquet", "", "Path to parquet directory")
	cpuProfile := flag.String("cpuprofile", "cpu.prof", "CPU profile output")
	memProfile := flag.String("memprofile", "mem.prof", "Memory profile output")
	blockProfile := flag.String("blockprofile", "block.prof", "Block profile output")
	mutexProfile := flag.String("mutexprofile", "mutex.prof", "Mutex profile output")
	limit := flag.Int("limit", 500000, "Number of documents to index")
	flag.Parse()

	if *parquetDir == "" {
		log.Fatal("Must specify -parquet directory")
	}

	// Enable block and mutex profiling
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	// Start CPU profiling
	cpuFile, err := os.Create(*cpuProfile)
	if err != nil {
		log.Fatal("Could not create CPU profile: ", err)
	}
	defer cpuFile.Close()
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		log.Fatal("Could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	// Create driver
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "blueprints", "search", "fineweb-profile")
	os.MkdirAll(dataDir, 0755)

	// Clean existing index
	os.RemoveAll(filepath.Join(dataDir, "profile.fts_highthroughput"))

	driver, err := fineweb.Open("fts_highthroughput", fineweb.DriverConfig{
		DataDir:  dataDir,
		Language: "profile",
	})
	if err != nil {
		log.Fatal("Could not open driver: ", err)
	}
	defer driver.Close()

	indexer, ok := driver.(fineweb.Indexer)
	if !ok {
		log.Fatal("Driver does not support indexing")
	}

	// Open parquet reader
	reader := fineweb.NewParquetReader(*parquetDir)

	ctx := context.Background()
	start := time.Now()
	var indexed int64

	// Create document iterator with limit
	docs := func(yield func(fineweb.Document, error) bool) {
		for doc, err := range reader.ReadN(ctx, *limit) {
			if err != nil {
				yield(fineweb.Document{}, err)
				return
			}
			if !yield(doc, nil) {
				return
			}
			indexed++
			if indexed%50000 == 0 {
				elapsed := time.Since(start)
				rate := float64(indexed) / elapsed.Seconds()
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				fmt.Printf("Progress: %d docs, %.0f docs/sec, %.2f GB heap\n",
					indexed, rate, float64(m.HeapAlloc)/(1024*1024*1024))
			}
		}
	}

	// Run indexing
	fmt.Printf("Starting indexing of %d documents...\n", *limit)
	err = indexer.Import(ctx, docs, nil)
	if err != nil {
		log.Fatal("Indexing failed: ", err)
	}

	elapsed := time.Since(start)
	rate := float64(indexed) / elapsed.Seconds()
	fmt.Printf("\nCompleted: %d docs in %v (%.0f docs/sec)\n", indexed, elapsed, rate)

	// Force GC and write memory profile
	runtime.GC()
	memFile, err := os.Create(*memProfile)
	if err != nil {
		log.Fatal("Could not create memory profile: ", err)
	}
	defer memFile.Close()
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		log.Fatal("Could not write memory profile: ", err)
	}

	// Write block profile
	blockFile, err := os.Create(*blockProfile)
	if err != nil {
		log.Fatal("Could not create block profile: ", err)
	}
	defer blockFile.Close()
	if err := pprof.Lookup("block").WriteTo(blockFile, 0); err != nil {
		log.Fatal("Could not write block profile: ", err)
	}

	// Write mutex profile
	mutexFile, err := os.Create(*mutexProfile)
	if err != nil {
		log.Fatal("Could not create mutex profile: ", err)
	}
	defer mutexFile.Close()
	if err := pprof.Lookup("mutex").WriteTo(mutexFile, 0); err != nil {
		log.Fatal("Could not write mutex profile: ", err)
	}

	fmt.Printf("\nProfiles written:\n")
	fmt.Printf("  CPU:   %s\n", *cpuProfile)
	fmt.Printf("  Heap:  %s\n", *memProfile)
	fmt.Printf("  Block: %s\n", *blockProfile)
	fmt.Printf("  Mutex: %s\n", *mutexProfile)
	fmt.Printf("\nAnalyze with:\n")
	fmt.Printf("  go tool pprof -http=:8080 %s\n", *cpuProfile)
}
