package bench_s3

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SaveAll saves the report in all configured formats.
func (rpt *Report) SaveAll(outputDir string, formats []string) error {
	os.MkdirAll(outputDir, 0o755)

	for _, fmt := range formats {
		switch fmt {
		case "markdown":
			if err := rpt.saveMarkdown(outputDir); err != nil {
				return err
			}
		case "json":
			if err := rpt.saveJSON(outputDir); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rpt *Report) saveJSON(dir string) error {
	data, err := json.MarshalIndent(rpt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "s3_bench.json"), data, 0o644)
}

func (rpt *Report) saveMarkdown(dir string) error {
	var sb strings.Builder

	sb.WriteString("# S3 Client Benchmark Results\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", rpt.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**BenchTime:** %v | **Warmup:** %d\n\n", rpt.Config.BenchTime, rpt.Config.WarmupIters))

	// Group results by operation category
	categories := map[string][]*Metrics{}
	for _, m := range rpt.Results {
		cat := operationCategory(m.Operation)
		categories[cat] = append(categories[cat], m)
	}

	// Ordered categories
	catOrder := []string{"PutObject", "GetObject", "HeadObject", "DeleteObject", "ListObjects", "Multipart", "Mixed", "Concurrency"}
	for _, cat := range catOrder {
		results, ok := categories[cat]
		if !ok || len(results) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s\n\n", cat))

		if cat == "Concurrency" {
			writeConcurrencyTable(&sb, results)
		} else {
			writeResultTable(&sb, results)
		}
		sb.WriteString("\n")
	}

	return os.WriteFile(filepath.Join(dir, "s3_bench.md"), []byte(sb.String()), 0o644)
}

func operationCategory(op string) string {
	switch {
	case strings.HasPrefix(op, "PutObject"):
		return "PutObject"
	case strings.HasPrefix(op, "GetObject"):
		return "GetObject"
	case strings.HasPrefix(op, "HeadObject"):
		return "HeadObject"
	case strings.HasPrefix(op, "DeleteObject"):
		return "DeleteObject"
	case strings.HasPrefix(op, "ListObjects"):
		return "ListObjects"
	case strings.HasPrefix(op, "Multipart"):
		return "Multipart"
	case strings.HasPrefix(op, "Mixed"):
		return "Mixed"
	case strings.HasPrefix(op, "Concurrency"):
		return "Concurrency"
	default:
		return "Other"
	}
}

func writeResultTable(sb *strings.Builder, results []*Metrics) {
	sb.WriteString("| Driver | Operation | Iters | Avg | P50 | P95 | P99 | Throughput | Ops/s | Errors |\n")
	sb.WriteString("|--------|-----------|------:|----:|----:|----:|----:|-----------:|------:|-------:|\n")

	// Sort by driver then operation
	sort.Slice(results, func(i, j int) bool {
		if results[i].Operation != results[j].Operation {
			return results[i].Operation < results[j].Operation
		}
		return results[i].Driver < results[j].Driver
	})

	for _, m := range results {
		tp := "-"
		if m.ThroughputMBps > 0 {
			tp = fmt.Sprintf("%.1f MB/s", m.ThroughputMBps)
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %v | %v | %v | %v | %s | %.0f | %d |\n",
			m.Driver, m.Operation, m.Iterations,
			m.AvgLatency.Round(time.Microsecond),
			m.P50Latency.Round(time.Microsecond),
			m.P95Latency.Round(time.Microsecond),
			m.P99Latency.Round(time.Microsecond),
			tp, m.OpsPerSec, m.Errors))
	}
}

func writeConcurrencyTable(sb *strings.Builder, results []*Metrics) {
	// Group by driver, extract concurrency levels
	type driverResults struct {
		name    string
		byConc  map[int]*Metrics
	}

	drivers := map[string]*driverResults{}
	concLevels := map[int]bool{}

	for _, m := range results {
		conc := parseConcurrency(m.Operation)
		concLevels[conc] = true
		dr, ok := drivers[m.Driver]
		if !ok {
			dr = &driverResults{name: m.Driver, byConc: map[int]*Metrics{}}
			drivers[m.Driver] = dr
		}
		dr.byConc[conc] = m
	}

	// Sort concurrency levels
	var levels []int
	for c := range concLevels {
		levels = append(levels, c)
	}
	sort.Ints(levels)

	// Header
	sb.WriteString("| Driver |")
	for _, c := range levels {
		sb.WriteString(fmt.Sprintf(" C%d |", c))
	}
	sb.WriteString("\n|--------|")
	for range levels {
		sb.WriteString("----:|")
	}
	sb.WriteString("\n")

	// Sort drivers
	driverNames := make([]string, 0, len(drivers))
	for name := range drivers {
		driverNames = append(driverNames, name)
	}
	sort.Strings(driverNames)

	for _, name := range driverNames {
		dr := drivers[name]
		sb.WriteString(fmt.Sprintf("| %s |", name))
		for _, c := range levels {
			if m, ok := dr.byConc[c]; ok {
				if m.ThroughputMBps > 0 {
					sb.WriteString(fmt.Sprintf(" %.1f MB/s |", m.ThroughputMBps))
				} else {
					sb.WriteString(fmt.Sprintf(" %.0f ops/s |", m.OpsPerSec))
				}
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString("\n")
	}
}

func parseConcurrency(op string) int {
	// Parse "Concurrency/C10/..." -> 10
	parts := strings.Split(op, "/")
	for _, p := range parts {
		if len(p) > 1 && p[0] == 'C' {
			var n int
			fmt.Sscanf(p[1:], "%d", &n)
			if n > 0 {
				return n
			}
		}
	}
	return 1
}
