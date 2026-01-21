package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Report contains all benchmark results.
type Report struct {
	Generated time.Time
	Config    *Config
	Results   []*Result
	ByDriver  map[string][]*Result
	ByOp      map[string][]*Result
	BySize    map[string][]*Result
}

// NewReport creates a report from results.
func NewReport(cfg *Config, results []*Result) *Report {
	r := &Report{
		Generated: time.Now(),
		Config:    cfg,
		Results:   results,
		ByDriver:  make(map[string][]*Result),
		ByOp:      make(map[string][]*Result),
		BySize:    make(map[string][]*Result),
	}

	for _, res := range results {
		if res.Skipped && res.Operation == "" {
			continue // Skip driver-level skip markers
		}
		r.ByDriver[res.Driver] = append(r.ByDriver[res.Driver], res)
		r.ByOp[res.Operation] = append(r.ByOp[res.Operation], res)
		if res.ObjectSize != "" {
			r.BySize[res.ObjectSize] = append(r.BySize[res.ObjectSize], res)
		}
	}

	return r
}

// SaveMarkdown writes the report to a markdown file.
func (r *Report) SaveMarkdown(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(outputDir, "warp_report.md")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Header
	fmt.Fprintf(f, "# Warp S3 Benchmark Report\n\n")
	fmt.Fprintf(f, "**Generated**: %s\n\n", r.Generated.Format(time.RFC3339))
	fmt.Fprintf(f, "## Configuration\n\n")
	fmt.Fprintf(f, "| Parameter | Value |\n")
	fmt.Fprintf(f, "|-----------|-------|\n")
	fmt.Fprintf(f, "| Duration per test | %v |\n", r.Config.Duration)
	fmt.Fprintf(f, "| Concurrent clients | %d |\n", r.Config.Concurrent)
	fmt.Fprintf(f, "| Objects | %d |\n", r.Config.Objects)
	fmt.Fprintf(f, "| Object sizes | %s |\n", strings.Join(r.Config.ObjectSizes, ", "))
	fmt.Fprintf(f, "| Operations | %s |\n", strings.Join(r.Config.Operations, ", "))
	fmt.Fprintf(f, "| List objects | %d |\n", r.Config.ListObjects)
	fmt.Fprintf(f, "| List max keys | %d |\n", r.Config.ListMaxKeys)
	fmt.Fprintf(f, "| Docker cleanup | %t |\n", r.Config.DockerClean)
	fmt.Fprintf(f, "| Compose dir | %s |\n", r.Config.ComposeDir)
	fmt.Fprintf(f, "| Output dir | %s |\n", r.Config.OutputDir)
	fmt.Fprintf(f, "| No clear | %t |\n", r.Config.NoClear)
	fmt.Fprintf(f, "| Prefix | %s |\n", r.Config.Prefix)
	fmt.Fprintf(f, "| Lookup style | %s |\n", r.Config.Lookup)
	fmt.Fprintf(f, "| Disable SHA256 | %t |\n", r.Config.DisableSHA256)
	fmt.Fprintf(f, "| Autoterm | %t |\n", r.Config.AutoTerm)
	fmt.Fprintf(f, "| Autoterm duration | %v |\n", r.Config.AutoTermDur)
	fmt.Fprintf(f, "| Autoterm pct | %.2f |\n", r.Config.AutoTermPct)
	fmt.Fprintf(f, "| PTY wrapper | %t |\n", r.Config.UsePTY)
	fmt.Fprintf(f, "| Progress interval | %v |\n", r.Config.ProgressEvery)
	fmt.Fprintf(f, "\n")

	// Environment
	fmt.Fprintf(f, "## Environment\n\n")
	fmt.Fprintf(f, "| Item | Value |\n")
	fmt.Fprintf(f, "|------|-------|\n")
	fmt.Fprintf(f, "| Go version | %s |\n", runtime.Version())
	fmt.Fprintf(f, "| OS/Arch | %s/%s |\n", runtime.GOOS, runtime.GOARCH)
	if r.Config.WarpVersion != "" {
		fmt.Fprintf(f, "| Warp version | %s |\n", r.Config.WarpVersion)
	}
	if r.Config.WarpPath != "" {
		fmt.Fprintf(f, "| Warp path | %s |\n", r.Config.WarpPath)
	}
	if r.Config.RunDir != "" {
		fmt.Fprintf(f, "| Warp work dir | %s |\n", r.Config.RunDir)
	}
	fmt.Fprintf(f, "| Keep work dir | %t |\n", r.Config.KeepWorkDir)
	fmt.Fprintf(f, "\n")

	// Drivers
	fmt.Fprintf(f, "## Drivers\n\n")
	r.writeDriverTable(f)

	// Summary table
	fmt.Fprintf(f, "## Summary\n\n")
	r.writeSummaryTable(f)

	// Winners
	fmt.Fprintf(f, "## Winners by Operation (Avg Throughput)\n\n")
	r.writeWinnersTable(f)

	// Detailed results by operation
	fmt.Fprintf(f, "## Detailed Results\n\n")
	for _, op := range r.Config.Operations {
		results := r.ByOp[op]
		if len(results) == 0 {
			continue
		}
		fmt.Fprintf(f, "### %s Operations\n\n", strings.ToUpper(op))
		r.writeOperationTable(f, op, results)
		fmt.Fprintf(f, "\n")
	}

	// Skipped drivers
	r.writeSkippedSection(f)

	return nil
}

// writeDriverTable writes driver details for comparison.
func (r *Report) writeDriverTable(f *os.File) {
	drivers := r.uniqueDrivers()
	if len(drivers) == 0 {
		fmt.Fprintf(f, "No drivers found.\n\n")
		return
	}

	driverInfo := make(map[string]*DriverConfig)
	for _, d := range DefaultDrivers() {
		driverInfo[d.Name] = d
	}

	fmt.Fprintf(f, "| Driver | Endpoint | Bucket | Status | Notes |\n")
	fmt.Fprintf(f, "|--------|----------|--------|--------|-------|\n")
	for _, name := range drivers {
		info := driverInfo[name]
		endpoint := "-"
		bucket := "-"
		if info != nil {
			endpoint = info.Endpoint
			bucket = info.Bucket
		}

		status := "benchmarked"
		notes := ""
		if skipped, reason := r.driverSkipReason(name); skipped {
			status = "skipped"
			notes = reason
		}

		fmt.Fprintf(f, "| %s | %s | %s | %s | %s |\n", name, endpoint, bucket, status, notes)
	}
	fmt.Fprintf(f, "\n")
}

func (r *Report) uniqueDrivers() []string {
	driverSet := make(map[string]bool)
	for _, res := range r.Results {
		driverSet[res.Driver] = true
	}
	var drivers []string
	for d := range driverSet {
		drivers = append(drivers, d)
	}
	sort.Strings(drivers)
	return drivers
}

func (r *Report) driverSkipReason(name string) (bool, string) {
	for _, res := range r.Results {
		if res.Driver == name && res.Skipped && res.Operation == "" {
			return true, res.SkipReason
		}
	}
	return false, ""
}

// writeSummaryTable writes a summary comparison table.
func (r *Report) writeSummaryTable(f *os.File) {
	// Get unique drivers (sorted)
	drivers := r.uniqueDrivers()

	if len(drivers) == 0 {
		fmt.Fprintf(f, "No results available.\n\n")
		return
	}

	// Calculate averages per driver per operation
	driverOps := make(map[string]map[string]*opResult)
	for _, d := range drivers {
		driverOps[d] = make(map[string]*opResult)
	}

	for _, res := range r.Results {
		if res.Skipped {
			continue
		}
		if driverOps[res.Driver][res.Operation] == nil {
			driverOps[res.Driver][res.Operation] = &opResult{}
		}
		or := driverOps[res.Driver][res.Operation]
		or.throughput += res.ThroughputMBps
		or.ops += res.OpsPerSec
		or.count++
	}

	bestByOp := make(map[string]float64)
	for _, op := range r.Config.Operations {
		bestByOp[op] = 0
		for _, driver := range drivers {
			or := driverOps[driver][op]
			if or == nil || or.count == 0 {
				continue
			}
			avg := or.throughput / float64(or.count)
			if avg > bestByOp[op] {
				bestByOp[op] = avg
			}
		}
	}

	// Build summary table
	fmt.Fprintf(f, "| Driver |")
	for _, op := range r.Config.Operations {
		fmt.Fprintf(f, " %s (MB/s) |", strings.ToUpper(op))
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "|--------|")
	for range r.Config.Operations {
		fmt.Fprintf(f, "------------|")
	}
	fmt.Fprintf(f, "\n")

	for _, driver := range drivers {
		fmt.Fprintf(f, "| %s |", driver)
		for _, op := range r.Config.Operations {
			or := driverOps[driver][op]
			if or != nil && or.count > 0 {
				avg := or.throughput / float64(or.count)
				if nearlyEqual(avg, bestByOp[op]) {
					fmt.Fprintf(f, " **%.2f** |", avg)
				} else {
					fmt.Fprintf(f, " %.2f |", avg)
				}
			} else {
				fmt.Fprintf(f, " - |")
			}
		}
		fmt.Fprintf(f, "\n")
	}
	fmt.Fprintf(f, "\n")
}

// writeWinnersTable writes a winners table by operation.
func (r *Report) writeWinnersTable(f *os.File) {
	type winner struct {
		driver string
		avg    float64
		second float64
	}

	drivers := r.uniqueDrivers()
	if len(drivers) == 0 {
		fmt.Fprintf(f, "No results available.\n\n")
		return
	}

	driverOps := make(map[string]map[string]*opResult)
	for _, d := range drivers {
		driverOps[d] = make(map[string]*opResult)
	}
	for _, res := range r.Results {
		if res.Skipped {
			continue
		}
		if driverOps[res.Driver][res.Operation] == nil {
			driverOps[res.Driver][res.Operation] = &opResult{}
		}
		or := driverOps[res.Driver][res.Operation]
		or.throughput += res.ThroughputMBps
		or.count++
	}

	fmt.Fprintf(f, "| Operation | Winner | Avg MB/s | Margin vs #2 |\n")
	fmt.Fprintf(f, "|-----------|--------|----------|--------------|\n")
	for _, op := range r.Config.Operations {
		var best winner
		best.avg = -1
		var second float64
		for _, d := range drivers {
			or := driverOps[d][op]
			if or == nil || or.count == 0 {
				continue
			}
			avg := or.throughput / float64(or.count)
			if avg > best.avg {
				second = best.avg
				best = winner{driver: d, avg: avg, second: second}
			} else if avg > second {
				second = avg
			}
		}
		if best.avg < 0 {
			fmt.Fprintf(f, "| %s | - | - | - |\n", strings.ToUpper(op))
			continue
		}
		margin := "-"
		if second > 0 {
			margin = fmt.Sprintf("+%.1f%%", ((best.avg-second)/second)*100)
		}
		fmt.Fprintf(f, "| %s | %s | %.2f | %s |\n", strings.ToUpper(op), best.driver, best.avg, margin)
	}
	fmt.Fprintf(f, "\n")
}

// writeOperationTable writes detailed results for an operation.
func (r *Report) writeOperationTable(f *os.File, op string, results []*Result) {
	// Group by size
	bySize := make(map[string][]*Result)
	for _, res := range results {
		size := res.ObjectSize
		if size == "" {
			size = "N/A"
		}
		bySize[size] = append(bySize[size], res)
	}

	// Get unique sizes (sorted)
	var sizes []string
	for s := range bySize {
		sizes = append(sizes, s)
	}
	sort.Slice(sizes, func(i, j int) bool {
		return parseSizeOrder(sizes[i]) < parseSizeOrder(sizes[j])
	})

	for _, size := range sizes {
		sizeResults := bySize[size]
		if size != "N/A" {
			fmt.Fprintf(f, "#### Object Size: %s\n\n", size)
		}

		fmt.Fprintf(f, "| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |\n")
		fmt.Fprintf(f, "|--------|-------------------|-----------|-------|----------|----------|----------|--------|\n")

		// Sort by throughput descending
		sort.Slice(sizeResults, func(i, j int) bool {
			return sizeResults[i].ThroughputMBps > sizeResults[j].ThroughputMBps
		})

		bestThroughput := 0.0
		for _, res := range sizeResults {
			if !res.Skipped && res.ThroughputMBps > bestThroughput {
				bestThroughput = res.ThroughputMBps
			}
		}

		for _, res := range sizeResults {
			if res.Skipped {
				fmt.Fprintf(f, "| %s | - | - | - | - | - | - | skipped |\n", res.Driver)
				continue
			}
			delta := "-"
			if bestThroughput > 0 {
				delta = fmt.Sprintf("%.1f%%", ((res.ThroughputMBps-bestThroughput)/bestThroughput)*100)
			}
			driver := res.Driver
			if nearlyEqual(res.ThroughputMBps, bestThroughput) {
				driver = "**" + driver + "**"
			}
			fmt.Fprintf(f, "| %s | %.2f | %s | %.2f | %.2f | %.2f | %.2f | %d |\n",
				driver,
				res.ThroughputMBps,
				delta,
				res.OpsPerSec,
				res.LatencyAvgMs,
				res.LatencyP50Ms,
				res.LatencyP99Ms,
				res.Errors,
			)
		}
		fmt.Fprintf(f, "\n")
	}
}

// writeSkippedSection writes information about skipped benchmarks.
func (r *Report) writeSkippedSection(f *os.File) {
	var skipped []*Result
	for _, res := range r.Results {
		if res.Skipped {
			skipped = append(skipped, res)
		}
	}

	if len(skipped) == 0 {
		return
	}

	fmt.Fprintf(f, "## Skipped Benchmarks\n\n")
	fmt.Fprintf(f, "| Driver | Operation | Size | Reason |\n")
	fmt.Fprintf(f, "|--------|-----------|------|--------|\n")

	for _, res := range skipped {
		op := res.Operation
		if op == "" {
			op = "(all)"
		}
		size := res.ObjectSize
		if size == "" {
			size = "N/A"
		}
		fmt.Fprintf(f, "| %s | %s | %s | %s |\n", res.Driver, op, size, res.SkipReason)
	}
	fmt.Fprintf(f, "\n")
}

// SaveJSON writes results to a JSON file.
func (r *Report) SaveJSON(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(outputDir, "warp_results.json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	data := struct {
		Generated string    `json:"generated"`
		Config    *Config   `json:"config"`
		Results   []*Result `json:"results"`
	}{
		Generated: r.Generated.Format(time.RFC3339),
		Config:    r.Config,
		Results:   r.Results,
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// parseSizeOrder returns a numeric order for size strings.
func parseSizeOrder(size string) int {
	if size == "N/A" {
		return 0
	}
	size = strings.ToLower(size)
	var multiplier int
	switch {
	case strings.HasSuffix(size, "kib") || strings.HasSuffix(size, "kb"):
		multiplier = 1
	case strings.HasSuffix(size, "mib") || strings.HasSuffix(size, "mb"):
		multiplier = 1024
	case strings.HasSuffix(size, "gib") || strings.HasSuffix(size, "gb"):
		multiplier = 1024 * 1024
	default:
		return 0
	}

	numStr := strings.TrimRight(size, "kibmgabKIBMGAB")
	num, _ := strconv.Atoi(numStr)
	return num * multiplier
}

func nearlyEqual(a, b float64) bool {
	const epsilon = 0.0001
	if a > b {
		return a-b < epsilon
	}
	return b-a < epsilon
}

type opResult struct {
	throughput float64
	ops        float64
	count      int
}
