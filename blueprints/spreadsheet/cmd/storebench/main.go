// Package main implements storebench - a comprehensive benchmark tool for spreadsheet storage.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	outputFile     = flag.String("output", "benchmark_report.md", "Output file for benchmark report")
	driversFlag    = flag.String("drivers", "duckdb,sqlite,swandb", "Comma-separated list of drivers to test (duckdb,postgres,sqlite,swandb)")
	categoriesFlag = flag.String("categories", "all", "Benchmark categories: all,cells,rows,merge,format,import,query")
	usecasesFlag   = flag.String("usecases", "all", "Use cases: all,financial,import,collaboration,report,sparse,bulk")
	loadTests      = flag.Bool("load", false, "Run load tests")
	quick          = flag.Bool("quick", false, "Run quick subset of benchmarks")
	verbose        = flag.Bool("v", false, "Verbose output")
	warmup         = flag.Int("warmup", 3, "Number of warmup iterations")
	iterations     = flag.Int("iter", 5, "Number of benchmark iterations")
	postgresDSN    = flag.String("postgres-dsn", "", "PostgreSQL DSN (default: from POSTGRES_TEST_DSN env)")
)

func main() {
	flag.Parse()

	// Parse drivers
	driverList := parseList(*driversFlag)
	categoryList := parseList(*categoriesFlag)
	usecaseList := parseList(*usecasesFlag)

	// Adjust iterations for quick mode
	warmupCount := *warmup
	iterCount := *iterations
	if *quick {
		if warmupCount > 1 {
			warmupCount = 1
		}
		if iterCount > 2 {
			iterCount = 2
		}
	}

	config := &BenchConfig{
		Drivers:    driverList,
		Categories: categoryList,
		Usecases:   usecaseList,
		RunLoad:    *loadTests,
		Quick:      *quick,
		Verbose:    *verbose,
		Warmup:     warmupCount,
		Iterations: iterCount,
	}

	// Setup PostgreSQL DSN
	if *postgresDSN != "" {
		os.Setenv("POSTGRES_TEST_DSN", *postgresDSN)
	}

	// Create progress display
	progress := NewProgressDisplay(os.Stdout)
	progress.PrintHeader(config)

	// Run benchmarks
	runner := NewBenchmarkRunner(config, progress)
	results, err := runner.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError running benchmarks: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	progress.PrintSummary(results)

	// Ensure output directory exists
	outputDir := filepath.Dir(*outputFile)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate report
	report := GenerateMarkdownReport(results, config)

	// Write report
	if err := os.WriteFile(*outputFile, []byte(report), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nReport written to: %s\n", *outputFile)
}

func parseList(s string) []string {
	if s == "all" {
		return []string{"all"}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

