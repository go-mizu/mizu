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

	// Summary comparison (at the top for quick overview)
	writeSummary(&sb, rpt.Results)

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

// writeResultTable writes a detail table with winner highlighting per operation group.
// Within each operation (e.g. PutObject/1KB), rows are sorted by avg latency (fastest first),
// and the fastest driver is bold. A "vs Best" column shows relative slowdown.
func writeResultTable(sb *strings.Builder, results []*Metrics) {
	sb.WriteString("| | Driver | Operation | Iters | Avg | P50 | P99 | Throughput | Ops/s | vs Best |\n")
	sb.WriteString("|---|--------|-----------|------:|----:|----:|----:|-----------:|------:|--------:|\n")

	// Group by operation
	groups := map[string][]*Metrics{}
	var opOrder []string
	for _, m := range results {
		if _, seen := groups[m.Operation]; !seen {
			opOrder = append(opOrder, m.Operation)
		}
		groups[m.Operation] = append(groups[m.Operation], m)
	}
	sort.Strings(opOrder)

	for _, op := range opOrder {
		ms := groups[op]
		// Sort by avg latency ascending (fastest first)
		sort.Slice(ms, func(i, j int) bool {
			return ms[i].AvgLatency < ms[j].AvgLatency
		})

		bestLatency := ms[0].AvgLatency
		worstLatency := ms[len(ms)-1].AvgLatency

		for rank, m := range ms {
			tp := "-"
			if m.ThroughputMBps > 0 {
				tp = fmt.Sprintf("%.1f MB/s", m.ThroughputMBps)
			}

			// Relative to best
			vsBest := ""
			if rank == 0 && bestLatency > 0 && worstLatency > bestLatency {
				// Winner: show how much faster than the slowest
				ratio := float64(worstLatency) / float64(bestLatency)
				vsBest = fmt.Sprintf("**%.0fx**", ratio)
			} else if bestLatency > 0 {
				ratio := float64(m.AvgLatency) / float64(bestLatency)
				vsBest = fmt.Sprintf("%.1fx", ratio)
			}

			// Rank indicator
			medal := "  "
			switch rank {
			case 0:
				medal = "1"
			case 1:
				medal = "2"
			case 2:
				medal = "3"
			}

			driver := m.Driver
			avg := m.AvgLatency.Round(time.Microsecond).String()
			p50 := m.P50Latency.Round(time.Microsecond).String()
			p99 := m.P99Latency.Round(time.Microsecond).String()

			if rank == 0 {
				sb.WriteString(fmt.Sprintf("| %s | **%s** | %s | %d | **%s** | %s | %s | %s | %.0f | %s |\n",
					medal, driver, op, m.Iterations,
					avg, p50, p99, tp, m.OpsPerSec, vsBest))
			} else {
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s | %s | %s | %.0f | %s |\n",
					medal, driver, op, m.Iterations,
					avg, p50, p99, tp, m.OpsPerSec, vsBest))
			}
		}
	}
}

// writeConcurrencyTable writes a matrix of driver x concurrency level with winner highlighting.
func writeConcurrencyTable(sb *strings.Builder, results []*Metrics) {
	type driverResults struct {
		name   string
		byConc map[int]*Metrics
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

	var levels []int
	for c := range concLevels {
		levels = append(levels, c)
	}
	sort.Ints(levels)

	// Find winner per concurrency level
	winners := map[int]string{}
	for _, c := range levels {
		var bestTP float64
		for name, dr := range drivers {
			if m, ok := dr.byConc[c]; ok && m.ThroughputMBps > bestTP {
				bestTP = m.ThroughputMBps
				winners[c] = name
			}
		}
	}

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

	// Sort drivers by C1 throughput descending
	driverNames := make([]string, 0, len(drivers))
	for name := range drivers {
		driverNames = append(driverNames, name)
	}
	sort.Slice(driverNames, func(i, j int) bool {
		ti, tj := 0.0, 0.0
		if m, ok := drivers[driverNames[i]].byConc[levels[0]]; ok {
			ti = m.ThroughputMBps
		}
		if m, ok := drivers[driverNames[j]].byConc[levels[0]]; ok {
			tj = m.ThroughputMBps
		}
		return ti > tj
	})

	for _, name := range driverNames {
		dr := drivers[name]
		sb.WriteString(fmt.Sprintf("| %s |", name))
		for _, c := range levels {
			if m, ok := dr.byConc[c]; ok {
				val := ""
				if m.ThroughputMBps > 0 {
					val = fmt.Sprintf("%.1f MB/s", m.ThroughputMBps)
				} else {
					val = fmt.Sprintf("%.0f ops/s", m.OpsPerSec)
				}
				if winners[c] == name {
					sb.WriteString(fmt.Sprintf(" **%s** |", val))
				} else {
					sb.WriteString(fmt.Sprintf(" %s |", val))
				}
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString("\n")
	}
}

func writeSummary(sb *strings.Builder, results []*Metrics) {
	ops := map[string][]*Metrics{}
	for _, m := range results {
		if m.Iterations == 0 || m.Errors > 0 {
			continue
		}
		if strings.HasPrefix(m.Operation, "Concurrency") {
			continue
		}
		ops[m.Operation] = append(ops[m.Operation], m)
	}

	if len(ops) == 0 {
		return
	}

	opNames := make([]string, 0, len(ops))
	for op := range ops {
		opNames = append(opNames, op)
	}
	sort.Strings(opNames)

	// For each operation, rank by avg latency
	type ranking struct {
		winner   *Metrics
		runnerUp *Metrics
	}
	rankings := make(map[string]*ranking, len(opNames))
	for _, op := range opNames {
		ms := ops[op]
		sort.Slice(ms, func(i, j int) bool {
			return ms[i].AvgLatency < ms[j].AvgLatency
		})
		r := &ranking{winner: ms[0]}
		if len(ms) > 1 {
			r.runnerUp = ms[1]
		}
		rankings[op] = r
	}

	// Count wins per driver
	winCounts := map[string]int{}
	for _, r := range rankings {
		winCounts[r.winner.Driver]++
	}

	sb.WriteString("## Summary\n\n")

	// Win count leaderboard
	type driverWin struct {
		name string
		wins int
	}
	var dws []driverWin
	for d, c := range winCounts {
		dws = append(dws, driverWin{d, c})
	}
	sort.Slice(dws, func(i, j int) bool {
		if dws[i].wins != dws[j].wins {
			return dws[i].wins > dws[j].wins
		}
		return dws[i].name < dws[j].name
	})

	sb.WriteString("### Leaderboard\n\n")
	sb.WriteString("| Rank | Driver | Wins | Share |\n")
	sb.WriteString("|-----:|--------|-----:|------:|\n")
	for i, dw := range dws {
		pct := float64(dw.wins) / float64(len(opNames)) * 100
		if i == 0 {
			sb.WriteString(fmt.Sprintf("| %d | **%s** | **%d** | **%.0f%%** |\n", i+1, dw.name, dw.wins, pct))
		} else {
			sb.WriteString(fmt.Sprintf("| %d | %s | %d | %.0f%% |\n", i+1, dw.name, dw.wins, pct))
		}
	}
	sb.WriteString("\n")

	// Per-operation winner table
	sb.WriteString("### Winner per Operation\n\n")
	sb.WriteString("| Operation | Winner | Avg | Runner-up | Avg | Speedup |\n")
	sb.WriteString("|-----------|--------|----:|-----------|----:|--------:|\n")

	for _, op := range opNames {
		r := rankings[op]
		w := r.winner
		if r.runnerUp != nil {
			ru := r.runnerUp
			speedup := float64(ru.AvgLatency) / float64(w.AvgLatency)
			sb.WriteString(fmt.Sprintf("| %s | **%s** | %v | %s | %v | %.1fx |\n",
				op, w.Driver, w.AvgLatency.Round(time.Microsecond),
				ru.Driver, ru.AvgLatency.Round(time.Microsecond),
				speedup))
		} else {
			sb.WriteString(fmt.Sprintf("| %s | **%s** | %v | - | - | - |\n",
				op, w.Driver, w.AvgLatency.Round(time.Microsecond)))
		}
	}

	// Throughput highlights
	sb.WriteString("\n### Throughput Highlights\n\n")
	sb.WriteString("| Operation | Winner | Throughput | Runner-up | Throughput | Speedup |\n")
	sb.WriteString("|-----------|--------|----------:|-----------|----------:|--------:|\n")

	for _, op := range opNames {
		ms := ops[op]
		// Sort by throughput descending
		sort.Slice(ms, func(i, j int) bool {
			return ms[i].ThroughputMBps > ms[j].ThroughputMBps
		})
		if ms[0].ThroughputMBps <= 0 {
			sort.Slice(ms, func(i, j int) bool {
				return ms[i].OpsPerSec > ms[j].OpsPerSec
			})
			if len(ms) > 1 && ms[0].OpsPerSec > 0 {
				speedup := ms[0].OpsPerSec / ms[1].OpsPerSec
				sb.WriteString(fmt.Sprintf("| %s | **%s** | %.0f ops/s | %s | %.0f ops/s | %.1fx |\n",
					op, ms[0].Driver, ms[0].OpsPerSec,
					ms[1].Driver, ms[1].OpsPerSec, speedup))
			}
			continue
		}
		if len(ms) > 1 {
			speedup := ms[0].ThroughputMBps / ms[1].ThroughputMBps
			sb.WriteString(fmt.Sprintf("| %s | **%s** | %.1f MB/s | %s | %.1f MB/s | %.1fx |\n",
				op, ms[0].Driver, ms[0].ThroughputMBps,
				ms[1].Driver, ms[1].ThroughputMBps, speedup))
		}
	}

	sb.WriteString("\n---\n\n")
}

func parseConcurrency(op string) int {
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
