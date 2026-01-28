// Command benchmark runs the fineweb search driver benchmark suite.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
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

	// New search engine drivers
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/elasticsearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/lnx"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/manticore"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/opensearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/postgres"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/quickwit"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/sonic"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/typesense"

	// PostgreSQL FTS extension drivers
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/postgres_pgroonga"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/postgres_pgsearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/postgres_textsearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/postgres_trgm"

	// Optimized FTS profile drivers
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_balanced"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_compact"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_lowmem"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_production"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_speed"

	// High-throughput driver
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_highthroughput"

	// Rust FFI driver (requires CGO and pre-built Rust library)
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_rust"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_rust_tantivy"
)

// External service endpoints
const (
	MeiliSearchURL    = "http://localhost:7700"
	ZincURL           = "http://localhost:4080"
	OpenSearchURL     = "http://localhost:9200"
	ElasticsearchURL  = "http://localhost:9201"
	PostgresURL       = "localhost:5432"
	TypesenseURL      = "http://localhost:8108"
	ManticoreURL      = "http://localhost:9308"
	QuickWitURL       = "http://localhost:7280"
	LnxURL            = "http://localhost:8000"
	SonicURL          = "localhost:1491"

	// PostgreSQL FTS extension service endpoints
	ParadeDBURL         = "localhost:5433"
	PGroongaURL         = "localhost:5434"
	PostgresTrgmURL     = "localhost:5435"
	PostgresNativeURL   = "localhost:5436"
	PostgresTextsearchURL = "localhost:5437"
)

func main() {
	var (
		// Driver selection
		all     = flag.Bool("all", false, "Run all drivers")
		driver  = flag.String("driver", "", "Run single driver")
		drivers = flag.String("drivers", "", "Comma-separated list of drivers")
		embedded = flag.Bool("embedded", false, "Run only embedded drivers (duckdb, sqlite, bleve, bluge, porter)")
		external = flag.Bool("external", false, "Run only external drivers (meilisearch, zinc)")

		// Data paths
		dataDir = flag.String("data", "", "Data directory for indexes")
		parquet = flag.String("parquet", "", "Parquet file/directory path")

		// Output options
		output    = flag.String("output", "", "Output file (.md, .json, or .csv)")
		reportDir = flag.String("report-dir", "", "Directory for multiple report formats")

		// Benchmark parameters
		timeout            = flag.Duration("timeout", 4*time.Hour, "Overall timeout")
		iterations         = flag.Int("iterations", 100, "Iterations per query for latency")
		fast               = flag.Bool("fast", false, "Fast mode with good coverage")
		throughputDuration = flag.Duration("throughput-duration", 10*time.Second, "Duration for each throughput test")
		freshIndex         = flag.Bool("fresh", false, "Force fresh indexing (delete existing)")
		testIncremental    = flag.Bool("incremental", false, "Test incremental indexing")
		incrementalDocs    = flag.Int64("incremental-docs", 1000, "Number of docs for incremental test")

		// Service management
		startDocker = flag.Bool("start-docker", false, "Start Docker services before benchmark")
		stopDocker  = flag.Bool("stop-docker", false, "Stop Docker services after benchmark")
		checkDocker = flag.Bool("check-docker", false, "Check Docker services availability")
		dockerDir   = flag.String("docker-dir", "", "Docker compose directory")

		// Utility commands
		list    = flag.Bool("list", false, "List available drivers")
		combine = flag.String("combine", "", "Combine JSON reports (glob pattern)")
		info    = flag.Bool("info", false, "Show driver information")
	)
	flag.Parse()

	// List drivers
	if *list {
		listDrivers()
		return
	}

	// Show driver info
	if *info {
		showDriverInfo()
		return
	}

	// Check Docker services
	if *checkDocker {
		checkDockerServices()
		return
	}

	// Combine mode
	if *combine != "" {
		combineReports(*combine, *output)
		return
	}

	// Resolve docker directory
	if *dockerDir == "" {
		// Try to find docker directory relative to current directory
		candidates := []string{
			"docker",
			"../docker",
			"../../docker",
			"/Users/apple/github/go-mizu/mizu/blueprints/search/docker",
		}
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(c, "docker-compose.search.yml")); err == nil {
				*dockerDir = c
				break
			}
		}
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
			*parquet = filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")
		}
	}

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Start Docker services if requested
	if *startDocker && *dockerDir != "" {
		log.Println("Starting Docker services...")
		if err := startDockerServices(*dockerDir); err != nil {
			log.Fatalf("Failed to start Docker services: %v", err)
		}
		// Wait for services to be healthy
		waitForServices()
	}

	// Create runner
	runner := benchmark.NewRunner(*dataDir, *parquet)
	runner.Iterations = *iterations
	runner.Logger = log.New(os.Stderr, "[benchmark] ", log.LstdFlags)

	// Mode adjustments
	if *fast {
		// Fast mode with comprehensive coverage
		runner.Iterations = 3
		runner.Concurrency = []int{10, 20, 40, 80}
		runner.ThroughputDuration = 2 * time.Second
		runner.SkipColdStart = true  // Cold start is slow (opens full index from disk)
		runner.SkipPerQuery = true   // Per-query stats redundant with latency
	}

	// Override throughput duration if explicitly set
	if *throughputDuration != 10*time.Second {
		runner.ThroughputDuration = *throughputDuration
	}

	// Apply new options
	runner.FreshIndex = *freshIndex
	runner.TestIncremental = *testIncremental
	runner.IncrementalDocs = *incrementalDocs

	// Determine which drivers to run
	if *all {
		runner.Drivers = fineweb.List()
	} else if *embedded {
		runner.Drivers = []string{"duckdb", "sqlite", "bleve", "bluge", "porter"}
	} else if *external {
		runner.Drivers = []string{"meilisearch", "zinc"}
	} else if *driver != "" {
		runner.Drivers = []string{*driver}
	} else if *drivers != "" {
		runner.Drivers = strings.Split(*drivers, ",")
	} else {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\nSpecify -all, -embedded, -external, -driver, or -drivers")
		os.Exit(1)
	}

	// Filter out unavailable external drivers
	runner.Drivers = filterAvailableDrivers(runner.Drivers)

	if len(runner.Drivers) == 0 {
		log.Fatal("No drivers available to benchmark")
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
	log.Printf("Iterations: %d", runner.Iterations)

	report, err := runner.Run(ctx)
	if err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}

	// Write output
	if *reportDir != "" {
		// Write multiple formats to report directory
		if err := writeReportDir(report, *reportDir); err != nil {
			log.Fatalf("Failed to write reports: %v", err)
		}
		log.Printf("Reports written to %s", *reportDir)
	} else if *output != "" {
		if err := writeReport(report, *output); err != nil {
			log.Fatalf("Failed to write report: %v", err)
		}
		log.Printf("Report written to %s", *output)
	}

	// Print summary
	fmt.Println(report.String())

	// Stop Docker services if requested
	if *stopDocker && *dockerDir != "" {
		log.Println("Stopping Docker services...")
		if err := stopDockerServices(*dockerDir); err != nil {
			log.Printf("Warning: Failed to stop Docker services: %v", err)
		}
	}
}

func listDrivers() {
	fmt.Println("Available drivers:")
	fmt.Println()

	// Group by type
	embedded := []string{"duckdb", "sqlite", "bleve", "bluge", "porter"}
	external := []string{"meilisearch", "zinc", "opensearch", "elasticsearch", "postgres", "typesense", "manticore", "quickwit", "lnx", "sonic"}
	cgo := []string{"tantivy"}

	fmt.Println("Embedded (no external dependencies):")
	for _, name := range embedded {
		status := "✓"
		if !fineweb.IsRegistered(name) {
			status = "✗"
		}
		fmt.Printf("  %s %s\n", status, name)
	}

	fmt.Println()
	fmt.Println("External (require Docker/services):")
	for _, name := range external {
		status := "✓"
		if !fineweb.IsRegistered(name) {
			status = "✗"
		}
		available := checkServiceAvailable(name)
		if available {
			fmt.Printf("  %s %s (service available)\n", status, name)
		} else {
			fmt.Printf("  %s %s (service not running)\n", status, name)
		}
	}

	fmt.Println()
	fmt.Println("CGO Required (special build):")
	for _, name := range cgo {
		status := "✗"
		if fineweb.IsRegistered(name) {
			status = "✓"
		}
		fmt.Printf("  %s %s\n", status, name)
	}
}

func showDriverInfo() {
	fmt.Println("Driver Information:")
	fmt.Println()

	tmpDir, _ := os.MkdirTemp("", "driver-info")
	defer os.RemoveAll(tmpDir)

	for _, name := range fineweb.List() {
		driver, err := fineweb.Open(name, fineweb.DriverConfig{DataDir: tmpDir})
		if err != nil {
			fmt.Printf("%s: ERROR - %v\n", name, err)
			continue
		}

		info := fineweb.GetDriverInfo(driver)
		if info != nil {
			fmt.Printf("%s:\n", name)
			fmt.Printf("  Description: %s\n", info.Description)
			fmt.Printf("  Features: %v\n", info.Features)
			fmt.Printf("  External: %v\n", info.External)
		} else {
			fmt.Printf("%s: (no info available)\n", name)
		}
		driver.Close()
		fmt.Println()
	}
}

func checkDockerServices() {
	fmt.Println("Docker Services Status:")
	fmt.Println()

	services := map[string]string{
		"meilisearch":   MeiliSearchURL + "/health",
		"zinc":          ZincURL + "/healthz",
		"opensearch":    OpenSearchURL + "/_cluster/health",
		"elasticsearch": ElasticsearchURL + "/_cluster/health",
		"typesense":     TypesenseURL + "/health",
		"manticore":     ManticoreURL,
		"quickwit":      QuickWitURL + "/health/livez",
		"lnx":           LnxURL,
	}

	for name, url := range services {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("  ✗ %s: not available (%v)\n", name, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == 200 || resp.StatusCode == 401 {
			fmt.Printf("  ✓ %s: healthy\n", name)
		} else {
			fmt.Printf("  ✗ %s: unhealthy (status %d)\n", name, resp.StatusCode)
		}
	}

	// Special checks for non-HTTP services
	fmt.Println()
	fmt.Println("  postgres: use 'pg_isready -h localhost -p 5432' to check")
	fmt.Println("  sonic: use 'nc -z localhost 1491' to check")
}

func checkServiceAvailable(name string) bool {
	var url string
	switch name {
	case "meilisearch":
		url = MeiliSearchURL + "/health"
	case "zinc":
		url = ZincURL + "/healthz"
	case "opensearch":
		url = OpenSearchURL + "/_cluster/health"
	case "elasticsearch":
		url = ElasticsearchURL + "/_cluster/health"
	case "typesense":
		url = TypesenseURL + "/health"
	case "manticore":
		url = ManticoreURL
	case "quickwit":
		url = QuickWitURL + "/health/livez"
	case "lnx":
		url = LnxURL
	case "postgres":
		// Check PostgreSQL via TCP connection
		return checkTCPService(PostgresURL)
	case "sonic":
		// Check Sonic via TCP connection
		return checkTCPService(SonicURL)
	case "postgres_pgsearch":
		// Check ParadeDB via TCP connection
		return checkTCPService(ParadeDBURL)
	case "postgres_pgroonga":
		// Check PGroonga via TCP connection
		return checkTCPService(PGroongaURL)
	case "postgres_trgm":
		// Check PostgreSQL with pg_trgm via TCP connection
		return checkTCPService(PostgresTrgmURL)
	case "postgres_textsearch":
		// Check PostgreSQL with pg_textsearch via TCP connection
		return checkTCPService(PostgresTextsearchURL)
	default:
		return true // Embedded drivers are always available
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200 || resp.StatusCode == 401
}

func checkTCPService(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func filterAvailableDrivers(drivers []string) []string {
	externalDrivers := map[string]bool{
		"meilisearch":         true,
		"zinc":                true,
		"opensearch":          true,
		"elasticsearch":       true,
		"postgres":            true,
		"typesense":           true,
		"manticore":           true,
		"quickwit":            true,
		"lnx":                 true,
		"sonic":               true,
		"postgres_pgsearch":   true,
		"postgres_pgroonga":   true,
		"postgres_trgm":       true,
		"postgres_textsearch": true,
	}

	var available []string
	for _, d := range drivers {
		if externalDrivers[d] {
			if !checkServiceAvailable(d) {
				log.Printf("Skipping %s (service not available)", d)
				continue
			}
		}
		available = append(available, d)
	}
	return available
}

func startDockerServices(dockerDir string) error {
	composePath := filepath.Join(dockerDir, "docker-compose.search.yml")
	cmd := exec.Command("docker-compose", "-f", composePath, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopDockerServices(dockerDir string) error {
	composePath := filepath.Join(dockerDir, "docker-compose.search.yml")
	cmd := exec.Command("docker-compose", "-f", composePath, "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForServices() {
	log.Println("Waiting for services to be healthy...")

	services := map[string]string{
		"meilisearch":   MeiliSearchURL + "/health",
		"zinc":          ZincURL + "/healthz",
		"opensearch":    OpenSearchURL + "/_cluster/health",
		"elasticsearch": ElasticsearchURL + "/_cluster/health",
		"typesense":     TypesenseURL + "/health",
		"manticore":     ManticoreURL,
		"quickwit":      QuickWitURL + "/health/livez",
		"lnx":           LnxURL,
	}

	deadline := time.Now().Add(3 * time.Minute) // Longer timeout for more services
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		allHealthy := true
		for name, url := range services {
			resp, err := client.Get(url)
			if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 401) {
				allHealthy = false
				log.Printf("  Waiting for %s...", name)
				if resp != nil {
					resp.Body.Close()
				}
				break
			}
			resp.Body.Close()
		}

		// Check TCP-based services
		if allHealthy {
			if !checkTCPService(PostgresURL) {
				allHealthy = false
				log.Printf("  Waiting for postgres...")
			}
		}
		if allHealthy {
			if !checkTCPService(SonicURL) {
				allHealthy = false
				log.Printf("  Waiting for sonic...")
			}
		}

		if allHealthy {
			log.Println("All services healthy")
			return
		}
		time.Sleep(5 * time.Second)
	}

	log.Println("Warning: Not all services became healthy")
}

func writeReport(report *benchmark.Report, path string) error {
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch {
	case strings.HasSuffix(path, ".json"):
		return report.WriteJSON(f)
	case strings.HasSuffix(path, ".csv"):
		return writeCSV(report, f)
	default:
		return report.WriteMarkdown(f)
	}
}

func writeReportDir(report *benchmark.Report, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")

	// Write all formats
	formats := map[string]func() error{
		filepath.Join(dir, fmt.Sprintf("benchmark_%s.md", timestamp)): func() error {
			return writeReport(report, filepath.Join(dir, fmt.Sprintf("benchmark_%s.md", timestamp)))
		},
		filepath.Join(dir, fmt.Sprintf("benchmark_%s.json", timestamp)): func() error {
			return writeReport(report, filepath.Join(dir, fmt.Sprintf("benchmark_%s.json", timestamp)))
		},
		filepath.Join(dir, fmt.Sprintf("benchmark_%s.csv", timestamp)): func() error {
			return writeReport(report, filepath.Join(dir, fmt.Sprintf("benchmark_%s.csv", timestamp)))
		},
		filepath.Join(dir, "latest.md"): func() error {
			return writeReport(report, filepath.Join(dir, "latest.md"))
		},
		filepath.Join(dir, "latest.json"): func() error {
			return writeReport(report, filepath.Join(dir, "latest.json"))
		},
	}

	for path, writeFunc := range formats {
		if err := writeFunc(); err != nil {
			log.Printf("Warning: Failed to write %s: %v", path, err)
		} else {
			log.Printf("Wrote %s", path)
		}
	}

	return nil
}

func writeCSV(report *benchmark.Report, w *os.File) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	header := []string{
		"Driver", "Status", "Index Time (s)", "Index Size (MB)",
		"p50 (ms)", "p95 (ms)", "p99 (ms)", "Avg (ms)",
		"QPS (1)", "QPS (10)", "QPS (50)", "QPS (100)",
		"Cold Start (ms)", "Peak Memory (MB)",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Data rows
	for _, result := range report.Results {
		row := make([]string, len(header))
		row[0] = result.Name

		if result.Error != "" {
			row[1] = "ERROR: " + result.Error
			writer.Write(row)
			continue
		}
		row[1] = "OK"

		if result.Indexing != nil {
			row[2] = fmt.Sprintf("%.2f", result.Indexing.Duration.Seconds())
		}
		if result.IndexSize > 0 {
			row[3] = fmt.Sprintf("%.2f", float64(result.IndexSize)/(1024*1024))
		}
		if result.Latency != nil {
			row[4] = fmt.Sprintf("%.3f", float64(result.Latency.P50.Microseconds())/1000)
			row[5] = fmt.Sprintf("%.3f", float64(result.Latency.P95.Microseconds())/1000)
			row[6] = fmt.Sprintf("%.3f", float64(result.Latency.P99.Microseconds())/1000)
			row[7] = fmt.Sprintf("%.3f", float64(result.Latency.Avg.Microseconds())/1000)
		}
		if result.Throughput != nil {
			row[8] = fmt.Sprintf("%.0f", result.Throughput.QPS)
		}
		if t, ok := result.Concurrency[10]; ok {
			row[9] = fmt.Sprintf("%.0f", t.QPS)
		}
		if t, ok := result.Concurrency[50]; ok {
			row[10] = fmt.Sprintf("%.0f", t.QPS)
		}
		if t, ok := result.Concurrency[100]; ok {
			row[11] = fmt.Sprintf("%.0f", t.QPS)
		}
		if result.ColdStart > 0 {
			row[12] = fmt.Sprintf("%.2f", float64(result.ColdStart.Milliseconds()))
		}
		if result.Memory != nil {
			row[13] = fmt.Sprintf("%.2f", float64(result.Memory.IndexingPeak)/(1024*1024))
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
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

	if output == "" {
		output = "combined_report.md"
	}

	if err := writeReport(combined, output); err != nil {
		log.Fatalf("Failed to write combined report: %v", err)
	}

	log.Printf("Combined %d reports into %s", len(reports), output)
}
