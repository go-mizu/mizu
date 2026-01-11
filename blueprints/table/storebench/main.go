package storebench

import (
	"fmt"
	"os"
	"strings"
)

// Main is the entry point for the storebench CLI.
func Main(args []string) {
	cfg := DefaultConfig()

	// Parse arguments
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			printUsage()
			return
		case strings.HasPrefix(arg, "--backends="):
			cfg.Backends = strings.Split(strings.TrimPrefix(arg, "--backends="), ",")
		case strings.HasPrefix(arg, "--scenarios="):
			cfg.Scenarios = strings.Split(strings.TrimPrefix(arg, "--scenarios="), ",")
		case strings.HasPrefix(arg, "--iterations="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--iterations="), "%d", &cfg.Iterations)
		case strings.HasPrefix(arg, "--concurrency="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--concurrency="), "%d", &cfg.Concurrency)
		case strings.HasPrefix(arg, "--output="):
			cfg.OutputDir = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "--postgres-url="):
			cfg.PostgresURL = strings.TrimPrefix(arg, "--postgres-url=")
		case strings.HasPrefix(arg, "--data-dir="):
			cfg.DataDir = strings.TrimPrefix(arg, "--data-dir=")
		case arg == "--verbose" || arg == "-v":
			cfg.Verbose = true
		}
	}

	if err := Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Run executes the benchmark with the given configuration.
func Run(cfg *Config) error {
	fmt.Println("StoreBench - Storage Backend Benchmark Tool")
	fmt.Println("==========================================")
	fmt.Printf("Backends: %s\n", strings.Join(cfg.Backends, ", "))
	fmt.Printf("Scenarios: %s\n", strings.Join(cfg.Scenarios, ", "))
	fmt.Printf("Iterations: %d\n", cfg.Iterations)
	fmt.Printf("Concurrency: %d\n", cfg.Concurrency)
	fmt.Printf("Output: %s\n", cfg.OutputDir)

	runner := NewRunner(cfg)
	results, err := runner.Run()
	if err != nil {
		return err
	}

	reporter := NewReportGenerator(results, cfg.OutputDir)
	return reporter.Generate()
}

func printUsage() {
	fmt.Println(`StoreBench - Storage Backend Benchmark Tool

Usage: storebench [options]

Options:
  --backends=<list>      Backends to test (comma-separated)
                         Values: duckdb,postgres,sqlite
                         Default: duckdb,sqlite

  --scenarios=<list>     Scenarios to run (comma-separated)
                         Values: records,batch,query,fields,concurrent
                         Default: all

  --iterations=<n>       Base iteration count
                         Default: 100

  --concurrency=<n>      Max concurrency for load tests
                         Default: 50

  --output=<dir>         Output directory for reports
                         Default: ./report

  --postgres-url=<url>   PostgreSQL connection URL
                         Format: postgres://user:pass@host:port/dbname
                         Env: STOREBENCH_POSTGRES_URL

  --data-dir=<dir>       Directory for DuckDB/SQLite files
                         Default: /tmp/storebench

  --verbose, -v          Verbose output

  --help, -h             Show this help message

Examples:
  # Run all benchmarks with defaults
  storebench

  # Benchmark only DuckDB and SQLite
  storebench --backends=duckdb,sqlite

  # Run only record scenarios with more iterations
  storebench --scenarios=records --iterations=500

  # Full benchmark including PostgreSQL
  storebench --backends=duckdb,postgres,sqlite \
             --postgres-url=postgres://user:pass@localhost:5432/bench`)
}
