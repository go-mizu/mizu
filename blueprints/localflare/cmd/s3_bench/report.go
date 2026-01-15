package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Report holds all benchmark data for export.
type Report struct {
	Timestamp time.Time
	Config    *Config
	Results   []*BenchmarkResult
}

// NewReport creates a new report from benchmark results.
func NewReport(cfg *Config, results []*BenchmarkResult) *Report {
	return &Report{
		Timestamp: time.Now(),
		Config:    cfg,
		Results:   results,
	}
}

// SaveAll saves all report formats to the output directory.
func (r *Report) SaveAll(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	if err := r.SaveCSV(filepath.Join(outputDir, "benchmark_results.csv")); err != nil {
		return fmt.Errorf("save CSV: %w", err)
	}

	if err := r.SaveJSON(filepath.Join(outputDir, "benchmark_results.json")); err != nil {
		return fmt.Errorf("save JSON: %w", err)
	}

	if err := r.SaveMarkdown(filepath.Join(outputDir, "benchmark_report.md")); err != nil {
		return fmt.Errorf("save markdown: %w", err)
	}

	return nil
}

// SaveCSV saves results to a CSV file.
func (r *Report) SaveCSV(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	headers := []string{
		"driver", "object_size_bytes", "threads", "throughput_mbps",
		"ttfb_avg_ms", "ttfb_min_ms", "ttfb_p25_ms", "ttfb_p50_ms",
		"ttfb_p75_ms", "ttfb_p90_ms", "ttfb_p99_ms", "ttfb_max_ms",
		"ttlb_avg_ms", "ttlb_min_ms", "ttlb_p25_ms", "ttlb_p50_ms",
		"ttlb_p75_ms", "ttlb_p90_ms", "ttlb_p99_ms", "ttlb_max_ms",
		"samples", "errors", "duration_ms",
	}
	if err := w.Write(headers); err != nil {
		return err
	}

	// Data rows
	for _, res := range r.Results {
		row := []string{
			res.Driver,
			fmt.Sprintf("%d", res.ObjectSize),
			fmt.Sprintf("%d", res.Threads),
			fmt.Sprintf("%.2f", res.Throughput),
			fmt.Sprintf("%d", res.TTFB.Avg.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.Min.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P25.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P50.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P75.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P90.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P99.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.Max.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.Avg.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.Min.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P25.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P50.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P75.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P90.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P99.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.Max.Milliseconds()),
			fmt.Sprintf("%d", res.Samples),
			fmt.Sprintf("%d", res.Errors),
			fmt.Sprintf("%d", res.Duration.Milliseconds()),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// JSONReport is the JSON export format.
type JSONReport struct {
	Timestamp string       `json:"timestamp"`
	Config    JSONConfig   `json:"config"`
	Results   []JSONResult `json:"results"`
	Summary   JSONSummary  `json:"summary"`
}

// JSONConfig is the JSON config export.
type JSONConfig struct {
	ThreadsMin  int `json:"threads_min"`
	ThreadsMax  int `json:"threads_max"`
	PayloadsMin int `json:"payloads_min"`
	PayloadsMax int `json:"payloads_max"`
	Samples     int `json:"samples"`
}

// JSONResult is a single result in JSON format.
type JSONResult struct {
	Driver     string      `json:"driver"`
	ObjectSize int         `json:"object_size_bytes"`
	Threads    int         `json:"threads"`
	Throughput float64     `json:"throughput_mbps"`
	TTFB       JSONLatency `json:"ttfb"`
	TTLB       JSONLatency `json:"ttlb"`
	Samples    int         `json:"samples"`
	Errors     int         `json:"errors"`
	DurationMs int64       `json:"duration_ms"`
}

// JSONLatency is latency stats in JSON format.
type JSONLatency struct {
	AvgMs int64 `json:"avg_ms"`
	MinMs int64 `json:"min_ms"`
	P25Ms int64 `json:"p25_ms"`
	P50Ms int64 `json:"p50_ms"`
	P75Ms int64 `json:"p75_ms"`
	P90Ms int64 `json:"p90_ms"`
	P99Ms int64 `json:"p99_ms"`
	MaxMs int64 `json:"max_ms"`
}

// JSONSummary is the summary section.
type JSONSummary struct {
	OverallWinner  string           `json:"overall_winner"`
	BestThroughput JSONBestResult   `json:"best_throughput"`
	BestTTFB       JSONBestResult   `json:"best_ttfb"`
	BestTTLB       JSONBestResult   `json:"best_ttlb"`
	Rankings       []JSONDriverRank `json:"rankings"`
}

// JSONBestResult identifies the best performer.
type JSONBestResult struct {
	Driver     string  `json:"driver"`
	ObjectSize int     `json:"object_size_bytes"`
	Threads    int     `json:"threads"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
}

// JSONDriverRank shows driver ranking.
type JSONDriverRank struct {
	Rank       int     `json:"rank"`
	Driver     string  `json:"driver"`
	Score      float64 `json:"score"`
	Throughput float64 `json:"avg_throughput_mbps"`
	LatencyP50 float64 `json:"avg_ttfb_p50_ms"`
}

// SaveJSON saves results to a JSON file.
func (r *Report) SaveJSON(path string) error {
	analysis := r.analyzeResults()

	jr := JSONReport{
		Timestamp: r.Timestamp.Format(time.RFC3339),
		Config: JSONConfig{
			ThreadsMin:  r.Config.ThreadsMin,
			ThreadsMax:  r.Config.ThreadsMax,
			PayloadsMin: r.Config.PayloadsMin,
			PayloadsMax: r.Config.PayloadsMax,
			Samples:     r.Config.Samples,
		},
		Results: make([]JSONResult, 0, len(r.Results)),
	}

	var bestThroughput *BenchmarkResult
	var bestTTFB *BenchmarkResult
	var bestTTLB *BenchmarkResult

	for _, res := range r.Results {
		jr.Results = append(jr.Results, JSONResult{
			Driver:     res.Driver,
			ObjectSize: res.ObjectSize,
			Threads:    res.Threads,
			Throughput: res.Throughput,
			TTFB: JSONLatency{
				AvgMs: res.TTFB.Avg.Milliseconds(),
				MinMs: res.TTFB.Min.Milliseconds(),
				P25Ms: res.TTFB.P25.Milliseconds(),
				P50Ms: res.TTFB.P50.Milliseconds(),
				P75Ms: res.TTFB.P75.Milliseconds(),
				P90Ms: res.TTFB.P90.Milliseconds(),
				P99Ms: res.TTFB.P99.Milliseconds(),
				MaxMs: res.TTFB.Max.Milliseconds(),
			},
			TTLB: JSONLatency{
				AvgMs: res.TTLB.Avg.Milliseconds(),
				MinMs: res.TTLB.Min.Milliseconds(),
				P25Ms: res.TTLB.P25.Milliseconds(),
				P50Ms: res.TTLB.P50.Milliseconds(),
				P75Ms: res.TTLB.P75.Milliseconds(),
				P90Ms: res.TTLB.P90.Milliseconds(),
				P99Ms: res.TTLB.P99.Milliseconds(),
				MaxMs: res.TTLB.Max.Milliseconds(),
			},
			Samples:    res.Samples,
			Errors:     res.Errors,
			DurationMs: res.Duration.Milliseconds(),
		})

		// Track best results
		if bestThroughput == nil || res.Throughput > bestThroughput.Throughput {
			bestThroughput = res
		}
		if bestTTFB == nil || (res.TTFB.Avg > 0 && res.TTFB.Avg < bestTTFB.TTFB.Avg) {
			bestTTFB = res
		}
		if bestTTLB == nil || (res.TTLB.Avg > 0 && res.TTLB.Avg < bestTTLB.TTLB.Avg) {
			bestTTLB = res
		}
	}

	jr.Summary.OverallWinner = analysis.OverallWinner
	if bestThroughput != nil {
		jr.Summary.BestThroughput = JSONBestResult{
			Driver:     bestThroughput.Driver,
			ObjectSize: bestThroughput.ObjectSize,
			Threads:    bestThroughput.Threads,
			Value:      bestThroughput.Throughput,
			Unit:       "MB/s",
		}
	}
	if bestTTFB != nil {
		jr.Summary.BestTTFB = JSONBestResult{
			Driver:     bestTTFB.Driver,
			ObjectSize: bestTTFB.ObjectSize,
			Threads:    bestTTFB.Threads,
			Value:      float64(bestTTFB.TTFB.Avg.Milliseconds()),
			Unit:       "ms",
		}
	}
	if bestTTLB != nil {
		jr.Summary.BestTTLB = JSONBestResult{
			Driver:     bestTTLB.Driver,
			ObjectSize: bestTTLB.ObjectSize,
			Threads:    bestTTLB.Threads,
			Value:      float64(bestTTLB.TTLB.Avg.Milliseconds()),
			Unit:       "ms",
		}
	}

	// Add rankings
	for i, score := range analysis.DriverScores {
		jr.Summary.Rankings = append(jr.Summary.Rankings, JSONDriverRank{
			Rank:       i + 1,
			Driver:     score.Driver,
			Score:      score.TotalScore,
			Throughput: score.AvgThroughput,
			LatencyP50: float64(score.AvgTTFBP50.Milliseconds()),
		})
	}

	data, err := json.MarshalIndent(jr, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DriverScore holds the calculated score for a driver.
type DriverScore struct {
	Driver        string
	TotalScore    float64
	AvgThroughput float64
	AvgTTFBP50    time.Duration
	AvgTTFBP99    time.Duration
	Consistency   float64 // Standard deviation of throughput
	SampleCount   int
	ErrorCount    int
}

// ResultAnalysis contains the complete analysis of benchmark results.
type ResultAnalysis struct {
	OverallWinner   string
	DriverScores    []DriverScore
	CategoryWinners map[string]CategoryWinner
	Recommendations []Recommendation
}

// CategoryWinner identifies the winner in a specific category.
type CategoryWinner struct {
	Category  string
	Winner    string
	Value     float64
	Unit      string
	RunnerUp  string
	RunnerVal float64
	Margin    float64 // Percentage difference
}

// Recommendation is an actionable recommendation.
type Recommendation struct {
	UseCase     string
	Recommended string
	Reason      string
}

// analyzeResults performs comprehensive analysis of benchmark results.
func (r *Report) analyzeResults() *ResultAnalysis {
	analysis := &ResultAnalysis{
		CategoryWinners: make(map[string]CategoryWinner),
		Recommendations: make([]Recommendation, 0),
	}

	// Group results by driver
	driverResults := make(map[string][]*BenchmarkResult)
	for _, res := range r.Results {
		driverResults[res.Driver] = append(driverResults[res.Driver], res)
	}

	// Calculate scores for each driver
	for driver, results := range driverResults {
		score := calculateDriverScore(driver, results)
		analysis.DriverScores = append(analysis.DriverScores, score)
	}

	// Sort by score descending
	sort.Slice(analysis.DriverScores, func(i, j int) bool {
		return analysis.DriverScores[i].TotalScore > analysis.DriverScores[j].TotalScore
	})

	// Set overall winner
	if len(analysis.DriverScores) > 0 {
		analysis.OverallWinner = analysis.DriverScores[0].Driver
	}

	// Calculate category winners
	analysis.calculateCategoryWinners(r.Results)

	// Generate recommendations
	analysis.generateRecommendations()

	return analysis
}

// calculateDriverScore calculates the composite score for a driver.
func calculateDriverScore(driver string, results []*BenchmarkResult) DriverScore {
	if len(results) == 0 {
		return DriverScore{Driver: driver}
	}

	var totalThroughput float64
	var totalTTFBP50, totalTTFBP99 time.Duration
	var throughputs []float64
	var errors int

	for _, res := range results {
		totalThroughput += res.Throughput
		totalTTFBP50 += res.TTFB.P50
		totalTTFBP99 += res.TTFB.P99
		throughputs = append(throughputs, res.Throughput)
		errors += res.Errors
	}

	n := float64(len(results))
	avgThroughput := totalThroughput / n
	avgTTFBP50 := time.Duration(float64(totalTTFBP50) / n)
	avgTTFBP99 := time.Duration(float64(totalTTFBP99) / n)

	// Calculate standard deviation for consistency
	var variance float64
	for _, t := range throughputs {
		diff := t - avgThroughput
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / n)

	// Consistency score: lower std dev = better consistency
	// Normalize: 100 - (stdDev / avgThroughput * 100)
	consistencyScore := 100.0
	if avgThroughput > 0 {
		cv := (stdDev / avgThroughput) * 100
		consistencyScore = math.Max(0, 100-cv*2) // Penalty for variance
	}

	// Normalize throughput to 0-100 scale (will be adjusted relative to others later)
	// For now, use raw throughput as the base
	throughputScore := avgThroughput

	// Normalize latency to 0-100 scale (lower is better)
	// Assume 100ms is baseline "worst", 1ms is baseline "best"
	latencyScore := 100.0
	if avgTTFBP50 > 0 {
		latencyMs := float64(avgTTFBP50.Milliseconds())
		latencyScore = math.Max(0, 100-latencyMs)
	}

	// Composite score: 50% throughput + 30% latency + 20% consistency
	// Throughput needs normalization across all drivers (done later)
	totalScore := throughputScore*0.5 + latencyScore*0.3 + consistencyScore*0.2

	return DriverScore{
		Driver:        driver,
		TotalScore:    totalScore,
		AvgThroughput: avgThroughput,
		AvgTTFBP50:    avgTTFBP50,
		AvgTTFBP99:    avgTTFBP99,
		Consistency:   stdDev,
		SampleCount:   len(results),
		ErrorCount:    errors,
	}
}

// calculateCategoryWinners finds winners in each category.
func (a *ResultAnalysis) calculateCategoryWinners(results []*BenchmarkResult) {
	// Best throughput
	var bestThroughput *BenchmarkResult
	var secondThroughput *BenchmarkResult
	for _, res := range results {
		if bestThroughput == nil || res.Throughput > bestThroughput.Throughput {
			secondThroughput = bestThroughput
			bestThroughput = res
		} else if secondThroughput == nil || res.Throughput > secondThroughput.Throughput {
			secondThroughput = res
		}
	}

	if bestThroughput != nil {
		margin := 0.0
		runnerUp := ""
		runnerVal := 0.0
		if secondThroughput != nil && secondThroughput.Throughput > 0 {
			margin = (bestThroughput.Throughput - secondThroughput.Throughput) / secondThroughput.Throughput * 100
			runnerUp = secondThroughput.Driver
			runnerVal = secondThroughput.Throughput
		}
		a.CategoryWinners["throughput"] = CategoryWinner{
			Category:  "Best Throughput",
			Winner:    bestThroughput.Driver,
			Value:     bestThroughput.Throughput,
			Unit:      "MB/s",
			RunnerUp:  runnerUp,
			RunnerVal: runnerVal,
			Margin:    margin,
		}
	}

	// Best TTFB p50
	var bestTTFB *BenchmarkResult
	var secondTTFB *BenchmarkResult
	for _, res := range results {
		if res.TTFB.P50 > 0 {
			if bestTTFB == nil || res.TTFB.P50 < bestTTFB.TTFB.P50 {
				secondTTFB = bestTTFB
				bestTTFB = res
			} else if secondTTFB == nil || res.TTFB.P50 < secondTTFB.TTFB.P50 {
				secondTTFB = res
			}
		}
	}

	if bestTTFB != nil {
		margin := 0.0
		runnerUp := ""
		runnerVal := 0.0
		if secondTTFB != nil && secondTTFB.TTFB.P50 > 0 {
			margin = float64(secondTTFB.TTFB.P50-bestTTFB.TTFB.P50) / float64(secondTTFB.TTFB.P50) * 100
			runnerUp = secondTTFB.Driver
			runnerVal = float64(secondTTFB.TTFB.P50.Milliseconds())
		}
		a.CategoryWinners["ttfb_p50"] = CategoryWinner{
			Category:  "Lowest TTFB p50",
			Winner:    bestTTFB.Driver,
			Value:     float64(bestTTFB.TTFB.P50.Milliseconds()),
			Unit:      "ms",
			RunnerUp:  runnerUp,
			RunnerVal: runnerVal,
			Margin:    margin,
		}
	}

	// Winners by object size
	sizeResults := make(map[int][]*BenchmarkResult)
	for _, res := range results {
		sizeResults[res.ObjectSize] = append(sizeResults[res.ObjectSize], res)
	}

	for size, sizeRes := range sizeResults {
		var best *BenchmarkResult
		for _, res := range sizeRes {
			if best == nil || res.Throughput > best.Throughput {
				best = res
			}
		}
		if best != nil {
			key := fmt.Sprintf("size_%d", size)
			a.CategoryWinners[key] = CategoryWinner{
				Category: fmt.Sprintf("%s Objects", FormatSize(size)),
				Winner:   best.Driver,
				Value:    best.Throughput,
				Unit:     "MB/s",
			}
		}
	}
}

// generateRecommendations creates actionable recommendations.
func (a *ResultAnalysis) generateRecommendations() {
	if len(a.DriverScores) == 0 {
		return
	}

	// Find best for throughput
	var bestThroughput *DriverScore
	for i := range a.DriverScores {
		if bestThroughput == nil || a.DriverScores[i].AvgThroughput > bestThroughput.AvgThroughput {
			bestThroughput = &a.DriverScores[i]
		}
	}

	// Find best for latency
	var bestLatency *DriverScore
	for i := range a.DriverScores {
		if a.DriverScores[i].AvgTTFBP50 > 0 {
			if bestLatency == nil || a.DriverScores[i].AvgTTFBP50 < bestLatency.AvgTTFBP50 {
				bestLatency = &a.DriverScores[i]
			}
		}
	}

	// Find most consistent
	var mostConsistent *DriverScore
	for i := range a.DriverScores {
		if mostConsistent == nil || a.DriverScores[i].Consistency < mostConsistent.Consistency {
			mostConsistent = &a.DriverScores[i]
		}
	}

	if bestThroughput != nil {
		a.Recommendations = append(a.Recommendations, Recommendation{
			UseCase:     "High-Throughput Workloads",
			Recommended: bestThroughput.Driver,
			Reason:      fmt.Sprintf("Delivers highest average throughput at %.1f MB/s", bestThroughput.AvgThroughput),
		})
	}

	if bestLatency != nil {
		a.Recommendations = append(a.Recommendations, Recommendation{
			UseCase:     "Latency-Sensitive Workloads",
			Recommended: bestLatency.Driver,
			Reason:      fmt.Sprintf("Lowest median latency at %d ms (p50)", bestLatency.AvgTTFBP50.Milliseconds()),
		})
	}

	if mostConsistent != nil && mostConsistent.Driver != bestThroughput.Driver {
		a.Recommendations = append(a.Recommendations, Recommendation{
			UseCase:     "Consistent Performance",
			Recommended: mostConsistent.Driver,
			Reason:      fmt.Sprintf("Most consistent throughput with lowest variance"),
		})
	}
}

// SaveMarkdown saves results to a Markdown file with comprehensive analysis.
func (r *Report) SaveMarkdown(path string) error {
	analysis := r.analyzeResults()

	var sb strings.Builder

	sb.WriteString("# S3 Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n", r.Timestamp.Format("2006-01-02 15:04:05 UTC")))

	// Calculate total duration
	var totalDuration time.Duration
	for _, res := range r.Results {
		totalDuration += res.Duration
	}
	sb.WriteString(fmt.Sprintf("**Total Samples:** %d\n\n", len(r.Results)*r.Config.Samples))

	sb.WriteString("---\n\n")

	// Executive Summary
	sb.WriteString("## Executive Summary\n\n")

	if analysis.OverallWinner != "" {
		sb.WriteString(fmt.Sprintf("### Overall Winner: **%s**\n\n", analysis.OverallWinner))
		sb.WriteString("Based on weighted scoring across all object sizes and thread configurations:\n\n")

		// Rankings table
		sb.WriteString("| Rank | Driver | Score | Throughput | TTFB p50 | Consistency |\n")
		sb.WriteString("|------|--------|-------|------------|----------|-------------|\n")

		medals := []string{"1st", "2nd", "3rd"}
		for i, score := range analysis.DriverScores {
			rank := fmt.Sprintf("%d", i+1)
			if i < len(medals) {
				rank = medals[i]
			}
			consistency := fmt.Sprintf("%.1f%%", 100-score.Consistency/score.AvgThroughput*100)
			sb.WriteString(fmt.Sprintf("| %s | %s | %.1f | %.1f MB/s | %d ms | %s |\n",
				rank, score.Driver, score.TotalScore, score.AvgThroughput,
				score.AvgTTFBP50.Milliseconds(), consistency))
		}
		sb.WriteString("\n")
	}

	// Key Findings
	sb.WriteString("### Key Findings\n\n")

	if cw, ok := analysis.CategoryWinners["throughput"]; ok {
		if cw.Margin > 0 {
			sb.WriteString(fmt.Sprintf("- **Best Throughput:** %s achieves %.0f%% higher throughput than runner-up\n", cw.Winner, cw.Margin))
		} else {
			sb.WriteString(fmt.Sprintf("- **Best Throughput:** %s at %.1f MB/s\n", cw.Winner, cw.Value))
		}
	}

	if cw, ok := analysis.CategoryWinners["ttfb_p50"]; ok {
		if cw.Margin > 0 {
			sb.WriteString(fmt.Sprintf("- **Lowest Latency:** %s has %.0f%% lower TTFB p50 than runner-up\n", cw.Winner, cw.Margin))
		} else {
			sb.WriteString(fmt.Sprintf("- **Lowest Latency:** %s at %.0f ms (p50)\n", cw.Winner, cw.Value))
		}
	}

	if len(analysis.DriverScores) > 0 {
		best := analysis.DriverScores[0]
		sb.WriteString(fmt.Sprintf("- **Most Consistent:** %s shows lowest variance in throughput\n", best.Driver))
	}

	sb.WriteString("\n---\n\n")

	// Category Winners
	sb.WriteString("## Category Winners\n\n")

	// By Object Size
	sb.WriteString("### By Object Size\n\n")
	sb.WriteString("| Object Size | Winner | Throughput | Runner-up |\n")
	sb.WriteString("|------------|--------|------------|----------|\n")

	sizes := make([]int, 0)
	for key := range analysis.CategoryWinners {
		if strings.HasPrefix(key, "size_") {
			var size int
			fmt.Sscanf(key, "size_%d", &size)
			sizes = append(sizes, size)
		}
	}
	sort.Ints(sizes)

	for _, size := range sizes {
		key := fmt.Sprintf("size_%d", size)
		if cw, ok := analysis.CategoryWinners[key]; ok {
			runnerUp := "-"
			if cw.RunnerUp != "" {
				runnerUp = fmt.Sprintf("%s (%.1f MB/s)", cw.RunnerUp, cw.RunnerVal)
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %.1f MB/s | %s |\n",
				FormatSize(size), cw.Winner, cw.Value, runnerUp))
		}
	}
	sb.WriteString("\n")

	// By Metric
	sb.WriteString("### By Metric\n\n")
	sb.WriteString("| Metric | Winner | Value | vs Average |\n")
	sb.WriteString("|--------|--------|-------|------------|\n")

	if cw, ok := analysis.CategoryWinners["throughput"]; ok {
		avgThroughput := 0.0
		for _, score := range analysis.DriverScores {
			avgThroughput += score.AvgThroughput
		}
		if len(analysis.DriverScores) > 0 {
			avgThroughput /= float64(len(analysis.DriverScores))
		}
		diff := (cw.Value - avgThroughput) / avgThroughput * 100
		sb.WriteString(fmt.Sprintf("| Throughput | %s | %.1f MB/s | +%.0f%% |\n", cw.Winner, cw.Value, diff))
	}

	if cw, ok := analysis.CategoryWinners["ttfb_p50"]; ok {
		avgLatency := 0.0
		for _, score := range analysis.DriverScores {
			avgLatency += float64(score.AvgTTFBP50.Milliseconds())
		}
		if len(analysis.DriverScores) > 0 {
			avgLatency /= float64(len(analysis.DriverScores))
		}
		diff := (avgLatency - cw.Value) / avgLatency * 100
		sb.WriteString(fmt.Sprintf("| TTFB p50 | %s | %.0f ms | -%.0f%% |\n", cw.Winner, cw.Value, diff))
	}

	sb.WriteString("\n---\n\n")

	// Recommendations
	sb.WriteString("## Recommendations\n\n")

	for _, rec := range analysis.Recommendations {
		sb.WriteString(fmt.Sprintf("### %s\n", rec.UseCase))
		sb.WriteString(fmt.Sprintf("**Use %s** - %s\n\n", rec.Recommended, rec.Reason))
	}

	// Trade-offs
	if len(analysis.DriverScores) > 1 {
		sb.WriteString("### Trade-offs\n\n")
		sb.WriteString("| Driver | Strengths | Considerations |\n")
		sb.WriteString("|--------|-----------|----------------|\n")

		for _, score := range analysis.DriverScores {
			strengths := []string{}
			considerations := []string{}

			// Determine strengths/considerations relative to others
			maxThroughput := analysis.DriverScores[0].AvgThroughput
			minLatency := analysis.DriverScores[0].AvgTTFBP50
			for _, s := range analysis.DriverScores {
				if s.AvgThroughput > maxThroughput {
					maxThroughput = s.AvgThroughput
				}
				if s.AvgTTFBP50 < minLatency && s.AvgTTFBP50 > 0 {
					minLatency = s.AvgTTFBP50
				}
			}

			if score.AvgThroughput >= maxThroughput*0.95 {
				strengths = append(strengths, "High throughput")
			} else {
				considerations = append(considerations, "Lower throughput")
			}

			if float64(score.AvgTTFBP50) <= float64(minLatency)*1.1 {
				strengths = append(strengths, "Low latency")
			} else {
				considerations = append(considerations, "Higher latency")
			}

			if score.Consistency < score.AvgThroughput*0.1 {
				strengths = append(strengths, "Consistent")
			}

			if len(strengths) == 0 {
				strengths = append(strengths, "Balanced")
			}
			if len(considerations) == 0 {
				considerations = append(considerations, "None observed")
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				score.Driver,
				strings.Join(strengths, ", "),
				strings.Join(considerations, ", ")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")

	// Detailed Results
	sb.WriteString("## Detailed Results\n\n")

	// Results by object size
	sizesMap := make(map[int]bool)
	for _, res := range r.Results {
		sizesMap[res.ObjectSize] = true
	}
	sizeList := make([]int, 0, len(sizesMap))
	for s := range sizesMap {
		sizeList = append(sizeList, s)
	}
	sort.Ints(sizeList)

	for _, size := range sizeList {
		sb.WriteString(fmt.Sprintf("### %s Objects\n\n", FormatSize(size)))

		// Table header
		sb.WriteString("| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |\n")
		sb.WriteString("|--------|---------|------------|----------|----------|----------|----------|\n")

		// Filter and sort results for this size
		var sizeResults []*BenchmarkResult
		for _, res := range r.Results {
			if res.ObjectSize == size {
				sizeResults = append(sizeResults, res)
			}
		}
		sort.Slice(sizeResults, func(i, j int) bool {
			if sizeResults[i].Driver != sizeResults[j].Driver {
				return sizeResults[i].Driver < sizeResults[j].Driver
			}
			return sizeResults[i].Threads < sizeResults[j].Threads
		})

		for _, res := range sizeResults {
			sb.WriteString(fmt.Sprintf("| %s | %d | %.1f MB/s | %d ms | %d ms | %d ms | %d ms |\n",
				res.Driver, res.Threads, res.Throughput,
				res.TTFB.P50.Milliseconds(), res.TTFB.P99.Milliseconds(),
				res.TTLB.P50.Milliseconds(), res.TTLB.P99.Milliseconds()))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")

	// Methodology
	sb.WriteString("## Methodology\n\n")
	sb.WriteString(fmt.Sprintf("- **Samples per Configuration:** %d\n", r.Config.Samples))
	sb.WriteString("- **Metrics:** TTFB (Time to First Byte), TTLB (Time to Last Byte)\n")
	sb.WriteString("- **Scoring:** Weighted composite: 50% throughput + 30% latency + 20% consistency\n\n")

	// Configuration
	sb.WriteString("## Configuration\n\n")
	sb.WriteString("| Parameter | Value |\n")
	sb.WriteString("|-----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Thread Range | %d - %d |\n", r.Config.ThreadsMin, r.Config.ThreadsMax))
	sb.WriteString(fmt.Sprintf("| Object Sizes | %s - %s |\n",
		FormatSize(1<<(r.Config.PayloadsMin+10)),
		FormatSize(1<<(r.Config.PayloadsMax+10))))
	sb.WriteString(fmt.Sprintf("| Samples/Config | %d |\n", r.Config.Samples))

	// Drivers
	drivers := make(map[string]bool)
	for _, res := range r.Results {
		drivers[res.Driver] = true
	}
	driverList := make([]string, 0, len(drivers))
	for d := range drivers {
		driverList = append(driverList, d)
	}
	sort.Strings(driverList)
	sb.WriteString(fmt.Sprintf("| Drivers | %s |\n", strings.Join(driverList, ", ")))

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
