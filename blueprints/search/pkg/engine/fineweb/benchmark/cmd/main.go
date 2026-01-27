// Command benchmark runs the fineweb search driver benchmark suite.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/benchmark"

	// Import all drivers for registration
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/bleve"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/bluge"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/duckdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/meilisearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/porter"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/sqlite"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/zinc"
	// Note: tantivy driver requires CGO, import with -tags tantivy
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/tantivy"
)

func main() {
	var (
		all       = flag.Bool("all", false, "Run all drivers")
		driver    = flag.String("driver", "", "Run single driver")
		drivers   = flag.String("drivers", "", "Comma-separated list of drivers")
		combine   = flag.String("combine", "", "Combine JSON reports (glob pattern)")
		output    = flag.String("output", "report.md", "Output file (.md or .json)")
		dataDir   = flag.String("data", "", "Data directory for indexes")
		parquet   = flag.String("parquet", "", "Parquet file/directory path")
		list      = flag.Bool("list", false, "List available drivers")
		timeout   = flag.Duration("timeout", 4*time.Hour, "Overall timeout")
		iterations = flag.Int("iterations", 100, "Iterations per query for latency")
	)
	flag.Parse()

	// List drivers
	if *list {
		fmt.Println("Available drivers:")
		for _, name := range fineweb.List() {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	// Combine mode
	if *combine != "" {
		combineReports(*combine, *output)
		return
	}

	// Determine data directory
	if *dataDir == "" {
		*dataDir = os.Getenv("DATA_DIR")
		if *dataDir == "" {
			home, _ := os.UserHomeDir()
			*dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
		}
	}

	// Determine parquet path
	if *parquet == "" {
		*parquet = os.Getenv("PARQUET_PATH")
		if *parquet == "" {
			home, _ := os.UserHomeDir()
			*parquet = filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")
		}
	}

	// Create runner
	runner := benchmark.NewRunner(*dataDir, *parquet)
	runner.Iterations = *iterations
	runner.Logger = log.New(os.Stderr, "[benchmark] ", log.LstdFlags)

	// Determine which drivers to run
	if *all {
		runner.Drivers = fineweb.List()
	} else if *driver != "" {
		runner.Drivers = []string{*driver}
	} else if *drivers != "" {
		runner.Drivers = strings.Split(*drivers, ",")
	} else {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\nSpecify -all, -driver, or -drivers")
		os.Exit(1)
	}

	// Validate drivers
	for _, d := range runner.Drivers {
		if !fineweb.IsRegistered(d) {
			fmt.Fprintf(os.Stderr, "Unknown driver: %s\n", d)
			fmt.Fprintf(os.Stderr, "Available: %v\n", fineweb.List())
			os.Exit(1)
		}
	}

	// Run benchmark
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	log.Printf("Starting benchmark with drivers: %v", runner.Drivers)
	log.Printf("Data dir: %s", *dataDir)
	log.Printf("Parquet: %s", *parquet)

	report, err := runner.Run(ctx)
	if err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}

	// Write output
	outputPath := *output
	if outputPath == "" {
		outputPath = "report.md"
	}

	if err := writeReport(report, outputPath); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}

	log.Printf("Report written to %s", outputPath)
	fmt.Println(report.String())
}

func writeReport(report *benchmark.Report, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if strings.HasSuffix(path, ".json") {
		return report.WriteJSON(f)
	}
	return report.WriteMarkdown(f)
}

func combineReports(pattern, output string) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatalf("Invalid glob pattern: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("No files match pattern: %s", pattern)
	}

	var reports []*benchmark.Report
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Printf("Warning: cannot open %s: %v", file, err)
			continue
		}

		r, err := benchmark.LoadReport(f)
		f.Close()
		if err != nil {
			log.Printf("Warning: cannot parse %s: %v", file, err)
			continue
		}

		reports = append(reports, r)
		log.Printf("Loaded: %s (%d results)", file, len(r.Results))
	}

	if len(reports) == 0 {
		log.Fatal("No valid reports found")
	}

	combined := benchmark.CombineReports(reports...)

	if err := writeReport(combined, output); err != nil {
		log.Fatalf("Failed to write combined report: %v", err)
	}

	log.Printf("Combined %d reports into %s", len(reports), output)
}
