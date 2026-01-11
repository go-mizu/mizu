package storebench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ReportGenerator generates benchmark reports.
type ReportGenerator struct {
	results   *BenchmarkResults
	outputDir string
}

// NewReportGenerator creates a new report generator.
func NewReportGenerator(results *BenchmarkResults, outputDir string) *ReportGenerator {
	return &ReportGenerator{
		results:   results,
		outputDir: outputDir,
	}
}

// Generate generates all report files.
func (g *ReportGenerator) Generate() error {
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate main markdown report
	reportPath := filepath.Join(g.outputDir, fmt.Sprintf("benchmark_report_%s.md", time.Now().Format("20060102_150405")))
	if err := g.generateMarkdownReport(reportPath); err != nil {
		return fmt.Errorf("failed to generate markdown report: %w", err)
	}

	// Generate JSON data
	rawDataDir := filepath.Join(g.outputDir, "raw_data")
	if err := os.MkdirAll(rawDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create raw data directory: %w", err)
	}

	if err := g.generateJSONData(rawDataDir); err != nil {
		return fmt.Errorf("failed to generate JSON data: %w", err)
	}

	fmt.Printf("\nReport generated: %s\n", reportPath)
	return nil
}

func (g *ReportGenerator) generateMarkdownReport(path string) error {
	var sb strings.Builder

	g.writeHeader(&sb)
	g.writeExecutiveSummary(&sb)
	g.writeEnvironment(&sb)
	g.writeResultsByCategory(&sb)
	g.writeConcurrencyAnalysis(&sb)
	g.writeRecommendations(&sb)

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func (g *ReportGenerator) writeHeader(sb *strings.Builder) {
	sb.WriteString("# Storage Backend Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", g.results.EndTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n\n", g.results.EndTime.Sub(g.results.StartTime).Round(time.Second)))
	sb.WriteString("---\n\n")
}

func (g *ReportGenerator) writeExecutiveSummary(sb *strings.Builder) {
	sb.WriteString("## Executive Summary\n\n")

	// Group results by scenario category
	categories := g.categorizeResults()

	// Find winners per category
	winners := make(map[string]string)
	for category, results := range categories {
		if len(results) == 0 {
			continue
		}

		// Find the backend with best average latency
		bestBackend := ""
		var bestAvg time.Duration
		backendAvgs := make(map[string][]time.Duration)

		for _, r := range results {
			backendAvgs[r.Backend] = append(backendAvgs[r.Backend], r.Stats.Avg)
		}

		for backend, avgs := range backendAvgs {
			var sum time.Duration
			for _, a := range avgs {
				sum += a
			}
			avg := sum / time.Duration(len(avgs))
			if bestBackend == "" || avg < bestAvg {
				bestBackend = backend
				bestAvg = avg
			}
		}
		winners[category] = bestBackend
	}

	sb.WriteString("### Winners by Category\n\n")
	sb.WriteString("| Category | Best Backend | Notes |\n")
	sb.WriteString("|----------|--------------|-------|\n")

	categoryOrder := []string{"Single Record", "Batch Operations", "Queries", "Field Operations", "Concurrent"}
	for _, cat := range categoryOrder {
		if winner, ok := winners[cat]; ok {
			notes := g.getCategoryNotes(cat, winner)
			sb.WriteString(fmt.Sprintf("| %s | **%s** | %s |\n", cat, winner, notes))
		}
	}
	sb.WriteString("\n")

	// Key findings
	sb.WriteString("### Key Findings\n\n")
	g.writeKeyFindings(sb, winners)
	sb.WriteString("\n")
}

func (g *ReportGenerator) categorizeResults() map[string][]Result {
	categories := make(map[string][]Result)

	for _, r := range g.results.Results {
		var category string
		switch {
		case strings.HasPrefix(r.Scenario, "record_"):
			category = "Single Record"
		case strings.HasPrefix(r.Scenario, "batch_"):
			category = "Batch Operations"
		case strings.HasPrefix(r.Scenario, "list_"):
			category = "Queries"
		case strings.HasPrefix(r.Scenario, "field_") || strings.HasPrefix(r.Scenario, "select_"):
			category = "Field Operations"
		case strings.HasPrefix(r.Scenario, "concurrent_"):
			category = "Concurrent"
		default:
			category = "Other"
		}
		categories[category] = append(categories[category], r)
	}

	return categories
}

func (g *ReportGenerator) getCategoryNotes(category, winner string) string {
	switch category {
	case "Single Record":
		return "CRUD operations on individual records"
	case "Batch Operations":
		return "Bulk insert/delete operations"
	case "Queries":
		return "List and filter operations"
	case "Field Operations":
		return "Schema operations"
	case "Concurrent":
		return "Parallel workloads"
	default:
		return ""
	}
}

func (g *ReportGenerator) writeKeyFindings(sb *strings.Builder, winners map[string]string) {
	// Count wins
	winCounts := make(map[string]int)
	for _, w := range winners {
		winCounts[w]++
	}

	// Sort backends by wins
	type kv struct {
		backend string
		wins    int
	}
	var sorted []kv
	for b, w := range winCounts {
		sorted = append(sorted, kv{b, w})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].wins > sorted[j].wins })

	if len(sorted) > 0 {
		sb.WriteString(fmt.Sprintf("1. **%s** leads in %d out of %d categories\n", sorted[0].backend, sorted[0].wins, len(winners)))
	}

	// Backend-specific findings
	for _, backend := range []string{"duckdb", "postgres", "sqlite"} {
		findings := g.getBackendFindings(backend)
		if findings != "" {
			sb.WriteString(fmt.Sprintf("2. **%s**: %s\n", backend, findings))
		}
	}
}

func (g *ReportGenerator) getBackendFindings(backend string) string {
	results := g.results.GetResultsByBackend()[backend]
	if len(results) == 0 {
		return ""
	}

	// Calculate average error rate
	var totalErrors int
	var totalOps int
	for _, r := range results {
		totalErrors += r.Stats.Errors
		totalOps += r.Stats.Count
	}

	if totalOps == 0 {
		return "No operations recorded"
	}

	errorRate := float64(totalErrors) / float64(totalOps) * 100

	switch backend {
	case "duckdb":
		if errorRate < 1 {
			return "Strong performance for analytical workloads"
		}
		return fmt.Sprintf("%.2f%% error rate", errorRate)
	case "postgres":
		if errorRate < 1 {
			return "Excellent for concurrent operations"
		}
		return fmt.Sprintf("%.2f%% error rate", errorRate)
	case "sqlite":
		if errorRate < 1 {
			return "Best for single-writer scenarios"
		}
		return fmt.Sprintf("%.2f%% error rate - consider for read-heavy workloads", errorRate)
	}
	return ""
}

func (g *ReportGenerator) writeEnvironment(sb *strings.Builder) {
	sb.WriteString("## Environment\n\n")
	sb.WriteString("| Property | Value |\n")
	sb.WriteString("|----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Go Version | %s |\n", g.results.Environment.GoVersion))
	sb.WriteString(fmt.Sprintf("| OS | %s |\n", g.results.Environment.OS))
	sb.WriteString(fmt.Sprintf("| Architecture | %s |\n", g.results.Environment.Arch))
	sb.WriteString(fmt.Sprintf("| CPUs | %d |\n", g.results.Environment.NumCPU))
	sb.WriteString(fmt.Sprintf("| Hostname | %s |\n", g.results.Environment.Hostname))
	sb.WriteString(fmt.Sprintf("| PostgreSQL | %s |\n", g.results.Environment.PostgresURL))
	sb.WriteString(fmt.Sprintf("| Data Dir | %s |\n", g.results.Environment.DataDir))
	sb.WriteString("\n")

	sb.WriteString("### Configuration\n\n")
	sb.WriteString("| Setting | Value |\n")
	sb.WriteString("|---------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Backends | %s |\n", strings.Join(g.results.Config.Backends, ", ")))
	sb.WriteString(fmt.Sprintf("| Scenarios | %s |\n", strings.Join(g.results.Config.Scenarios, ", ")))
	sb.WriteString(fmt.Sprintf("| Iterations | %d |\n", g.results.Config.Iterations))
	sb.WriteString(fmt.Sprintf("| Concurrency | %d |\n", g.results.Config.Concurrency))
	sb.WriteString(fmt.Sprintf("| Warmup | %d |\n", g.results.Config.WarmupIters))
	sb.WriteString("\n")
}

func (g *ReportGenerator) writeResultsByCategory(sb *strings.Builder) {
	sb.WriteString("## Results by Category\n\n")

	categories := g.categorizeResults()
	categoryOrder := []string{"Single Record", "Batch Operations", "Queries", "Field Operations", "Concurrent"}

	for _, category := range categoryOrder {
		results, ok := categories[category]
		if !ok || len(results) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s\n\n", category))

		// Group by scenario
		scenarios := make(map[string][]Result)
		for _, r := range results {
			scenarios[r.Scenario] = append(scenarios[r.Scenario], r)
		}

		// Get unique scenario names in order
		var scenarioNames []string
		seen := make(map[string]bool)
		for _, r := range results {
			if !seen[r.Scenario] {
				seen[r.Scenario] = true
				scenarioNames = append(scenarioNames, r.Scenario)
			}
		}

		for _, scenario := range scenarioNames {
			scenarioResults := scenarios[scenario]
			sb.WriteString(fmt.Sprintf("#### %s\n\n", scenario))

			sb.WriteString("| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |\n")
			sb.WriteString("|---------|-----|-----|-----|-----|-------|--------|\n")

			for _, r := range scenarioResults {
				sb.WriteString(fmt.Sprintf("| %s | %v | %v | %v | %v | %.2f | %d |\n",
					r.Backend,
					r.Stats.Avg.Round(time.Microsecond),
					r.Stats.P50.Round(time.Microsecond),
					r.Stats.P95.Round(time.Microsecond),
					r.Stats.P99.Round(time.Microsecond),
					r.Stats.OpsPerSec,
					r.Stats.Errors,
				))
			}
			sb.WriteString("\n")

			// Add comparison if multiple backends
			if len(scenarioResults) > 1 {
				g.writeComparison(sb, scenarioResults)
			}
		}
	}
}

func (g *ReportGenerator) writeComparison(sb *strings.Builder, results []Result) {
	if len(results) < 2 {
		return
	}

	// Find the fastest
	fastest := results[0]
	for _, r := range results[1:] {
		if r.Stats.Avg < fastest.Stats.Avg {
			fastest = r
		}
	}

	sb.WriteString("**Comparison:**\n")
	for _, r := range results {
		if r.Backend == fastest.Backend {
			sb.WriteString(fmt.Sprintf("- %s: baseline (fastest)\n", r.Backend))
		} else {
			pctSlower := float64(r.Stats.Avg-fastest.Stats.Avg) / float64(fastest.Stats.Avg) * 100
			sb.WriteString(fmt.Sprintf("- %s: +%.1f%% slower\n", r.Backend, pctSlower))
		}
	}
	sb.WriteString("\n")
}

func (g *ReportGenerator) writeConcurrencyAnalysis(sb *strings.Builder) {
	sb.WriteString("## Concurrency Analysis\n\n")

	// Group concurrent results by backend and type
	grouped := make(map[concurrencyKey][]Result)

	for _, r := range g.results.Results {
		if !strings.HasPrefix(r.Scenario, "concurrent_") {
			continue
		}

		// Extract operation type (reads, writes, mixed)
		parts := strings.Split(r.Scenario, "_")
		if len(parts) < 2 {
			continue
		}
		opType := parts[1]

		k := concurrencyKey{r.Backend, opType}
		grouped[k] = append(grouped[k], r)
	}

	if len(grouped) == 0 {
		sb.WriteString("No concurrent benchmark data available.\n\n")
		return
	}

	for opType := range map[string]bool{"reads": true, "writes": true, "mixed": true} {
		sb.WriteString(fmt.Sprintf("### Concurrent %s\n\n", capitalize(opType)))
		sb.WriteString("| Backend | Concurrency | Avg | P99 | Ops/s | Errors |\n")
		sb.WriteString("|---------|-------------|-----|-----|-------|--------|\n")

		for _, backend := range []string{"duckdb", "postgres", "sqlite"} {
			k := concurrencyKey{backend, opType}
			results := grouped[k]
			if len(results) == 0 {
				continue
			}

			// Sort by concurrency level
			sort.Slice(results, func(i, j int) bool {
				return extractConcurrency(results[i].Scenario) < extractConcurrency(results[j].Scenario)
			})

			for _, r := range results {
				conc := extractConcurrency(r.Scenario)
				sb.WriteString(fmt.Sprintf("| %s | %d | %v | %v | %.2f | %d |\n",
					r.Backend,
					conc,
					r.Stats.Avg.Round(time.Microsecond),
					r.Stats.P99.Round(time.Microsecond),
					r.Stats.OpsPerSec,
					r.Stats.Errors,
				))
			}
		}
		sb.WriteString("\n")
	}

	// Scaling analysis
	sb.WriteString("### Scaling Observations\n\n")
	g.writeScalingObservations(sb, grouped)
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func extractConcurrency(scenario string) int {
	parts := strings.Split(scenario, "_")
	if len(parts) < 3 {
		return 0
	}
	var conc int
	fmt.Sscanf(parts[len(parts)-1], "%d", &conc)
	return conc
}

type concurrencyKey struct {
	backend string
	opType  string
}

func (g *ReportGenerator) writeScalingObservations(sb *strings.Builder, grouped map[concurrencyKey][]Result) {
	for _, backend := range []string{"duckdb", "postgres", "sqlite"} {
		observations := []string{}

		// Check writes scaling
		writesKey := concurrencyKey{backend, "writes"}
		if results := grouped[writesKey]; len(results) >= 2 {
			// Sort by concurrency
			sort.Slice(results, func(i, j int) bool {
				return extractConcurrency(results[i].Scenario) < extractConcurrency(results[j].Scenario)
			})

			first := results[0]
			last := results[len(results)-1]

			if last.Stats.Errors > first.Stats.Errors*2 {
				observations = append(observations, "write contention increases significantly at higher concurrency")
			} else if last.Stats.OpsPerSec > first.Stats.OpsPerSec*1.5 {
				observations = append(observations, "scales well for concurrent writes")
			}
		}

		// Check reads scaling
		readsKey := concurrencyKey{backend, "reads"}
		if results := grouped[readsKey]; len(results) >= 2 {
			sort.Slice(results, func(i, j int) bool {
				return extractConcurrency(results[i].Scenario) < extractConcurrency(results[j].Scenario)
			})

			first := results[0]
			last := results[len(results)-1]

			if last.Stats.OpsPerSec > first.Stats.OpsPerSec*2 {
				observations = append(observations, "excellent read scaling")
			}
		}

		if len(observations) > 0 {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", backend, strings.Join(observations, "; ")))
		}
	}
	sb.WriteString("\n")
}

func (g *ReportGenerator) writeRecommendations(sb *strings.Builder) {
	sb.WriteString("## Recommendations\n\n")

	// Analyze results and provide recommendations
	categories := g.categorizeResults()

	sb.WriteString("### Use Case Recommendations\n\n")

	// Single-user / embedded use case
	sb.WriteString("#### Embedded / Single-User Applications\n\n")
	sqliteWins := 0
	duckdbWins := 0
	for _, results := range categories {
		for _, r := range results {
			if !strings.Contains(r.Scenario, "concurrent") {
				// For non-concurrent, compare SQLite and DuckDB
				// (simplified logic)
				if r.Backend == "sqlite" {
					sqliteWins++
				} else if r.Backend == "duckdb" {
					duckdbWins++
				}
			}
		}
	}

	if sqliteWins > duckdbWins {
		sb.WriteString("**Recommended: SQLite**\n\n")
		sb.WriteString("- Mature, well-tested embedded database\n")
		sb.WriteString("- Excellent single-writer performance\n")
		sb.WriteString("- Zero configuration required\n\n")
	} else {
		sb.WriteString("**Recommended: DuckDB**\n\n")
		sb.WriteString("- Better performance for analytical queries\n")
		sb.WriteString("- Good batch operation support\n")
		sb.WriteString("- Modern embedded database\n\n")
	}

	// Multi-user / server use case
	sb.WriteString("#### Multi-User / Server Applications\n\n")
	sb.WriteString("**Recommended: PostgreSQL**\n\n")
	sb.WriteString("- Best concurrent write handling\n")
	sb.WriteString("- Mature connection pooling\n")
	sb.WriteString("- MVCC for high concurrency\n")
	sb.WriteString("- Rich querying capabilities\n\n")

	// Analytical use case
	sb.WriteString("#### Analytical / Reporting Workloads\n\n")
	sb.WriteString("**Recommended: DuckDB**\n\n")
	sb.WriteString("- Columnar storage for analytics\n")
	sb.WriteString("- Efficient batch operations\n")
	sb.WriteString("- Good for read-heavy workloads\n\n")

	sb.WriteString("### Configuration Tips\n\n")
	sb.WriteString("1. **SQLite**: Use WAL mode (already configured) for better concurrency\n")
	sb.WriteString("2. **PostgreSQL**: Tune connection pool size based on workload\n")
	sb.WriteString("3. **DuckDB**: Consider memory settings for large datasets\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("*Report generated by StoreBench*\n")
}

func (g *ReportGenerator) generateJSONData(dir string) error {
	// Write full results
	allData, err := json.MarshalIndent(g.results, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "all_results.json"), allData, 0644); err != nil {
		return err
	}

	// Write per-backend files
	byBackend := g.results.GetResultsByBackend()
	for backend, results := range byBackend {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s_results.json", backend)), data, 0644); err != nil {
			return err
		}
	}

	return nil
}
