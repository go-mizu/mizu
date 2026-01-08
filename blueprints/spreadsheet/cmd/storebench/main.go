// Package main implements storebench - a comprehensive benchmark tool for spreadsheet storage.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
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

	fmt.Println("=== Spreadsheet Storage Benchmark ===")
	fmt.Printf("Drivers: %v\n", driverList)
	fmt.Printf("Categories: %v\n", categoryList)
	fmt.Printf("Output: %s\n", *outputFile)
	fmt.Println()

	// Run benchmarks
	runner := NewBenchmarkRunner(config)
	results, err := runner.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running benchmarks: %v\n", err)
		os.Exit(1)
	}

	// Generate report
	report := GenerateMarkdownReport(results, config)

	// Write report
	if err := os.WriteFile(*outputFile, []byte(report), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nReport written to: %s\n", *outputFile)
	printSummary(results)
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

func printSummary(results *BenchResults) {
	fmt.Println("\n=== Summary ===")
	fmt.Printf("Total benchmarks: %d\n", len(results.Results))
	fmt.Printf("Total duration: %v\n", results.TotalDuration.Round(time.Millisecond))

	// Find fastest driver per category
	categoryWins := make(map[string]map[string]int)
	for _, r := range results.Results {
		if categoryWins[r.Category] == nil {
			categoryWins[r.Category] = make(map[string]int)
		}
	}

	// Group by benchmark name and find fastest
	byName := make(map[string][]BenchResult)
	for _, r := range results.Results {
		key := r.Category + "/" + r.Name
		byName[key] = append(byName[key], r)
	}

	for _, rs := range byName {
		if len(rs) < 2 {
			continue
		}
		fastest := rs[0]
		for _, r := range rs[1:] {
			if r.NsPerOp < fastest.NsPerOp {
				fastest = r
			}
		}
		categoryWins[fastest.Category][fastest.Driver]++
	}

	fmt.Println("\nWins by category:")
	for cat, wins := range categoryWins {
		fmt.Printf("  %s: ", cat)
		for driver, count := range wins {
			fmt.Printf("%s=%d ", driver, count)
		}
		fmt.Println()
	}
}
