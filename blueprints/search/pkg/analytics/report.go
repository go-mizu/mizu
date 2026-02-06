package analytics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteReport generates a complete markdown report and writes it to a file.
func WriteReport(r *Report, split, lang, outputPath, parquetPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	report := generateReport(r, split, lang, parquetPath)

	if err := os.WriteFile(outputPath, []byte(report), 0o644); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	return nil
}

func generateReport(r *Report, split, lang, parquetPath string) string {
	var sb strings.Builder
	s := r.Summary

	// Header
	sb.WriteString(fmt.Sprintf("# FineWeb-2 Analytics: %s — %s\n\n", lang, split))
	sb.WriteString(fmt.Sprintf("> Generated: %s | Records: %s | File: `%s`\n\n",
		time.Now().Format("2006-01-02 15:04:05"),
		FormatInt(s.TotalDocs),
		parquetPath,
	))

	// Table of Contents
	sb.WriteString("## Table of Contents\n\n")
	sb.WriteString("1. [Overview](#overview)\n")
	sb.WriteString("2. [Text Statistics](#text-statistics) (Charts 1-12)\n")
	sb.WriteString("3. [Temporal Analysis](#temporal-analysis) (Charts 13-23)\n")
	sb.WriteString("4. [URL & Domain Analysis](#url--domain-analysis) (Charts 24-35)\n")
	sb.WriteString("5. [Quality & Deduplication](#quality--deduplication) (Charts 36-45)\n")
	sb.WriteString("6. [Vietnamese Content Analysis](#vietnamese-content-analysis) (Charts 46-55)\n\n")

	// Overview
	sb.WriteString("---\n\n## Overview\n\n### Dataset Summary\n\n")
	sb.WriteString(mdTable([]string{"Metric", "Value"}, [][]string{
		{"Total Documents", FormatInt(s.TotalDocs)},
		{"Total Characters", FormatInt(s.TotalChars)},
		{"Total Words", FormatInt(s.TotalWords)},
		{"Unique Domains", FormatInt(s.UniqueDomains)},
		{"Date Range", s.DateRange},
		{"Average Language Score", fmt.Sprintf("%.6f", s.AvgLangScore)},
		{"Average Text Length", fmt.Sprintf("%.0f chars", s.AvgTextLength)},
		{"Median Text Length", fmt.Sprintf("%.0f chars", s.MedianTextLen)},
		{"Average Cluster Size", fmt.Sprintf("%.1f", s.AvgClusterSize)},
	}))

	// ── Text Statistics ────────────────────────────────────────
	sb.WriteString("\n---\n\n## Text Statistics\n\n")
	writeXYBar(&sb, "1", "Document Length Distribution", "Length", "Count", r.TextLengthDist)
	writeXYBar(&sb, "2", "Word Count Distribution", "Words", "Documents", r.WordCountDist)
	writeXYBar(&sb, "3", "Sentence Count Distribution", "Sentences", "Documents", r.SentenceCountDist)
	writeXYBar(&sb, "4", "Line Count Distribution", "Lines", "Documents", r.LineCountDist)

	// Percentiles
	sb.WriteString("### 5. Text Length Percentiles\n\n")
	if len(r.TextPercentiles) > 0 {
		e := r.TextPercentiles[0].Extra
		sb.WriteString(mdTable([]string{"Percentile", "Value"}, [][]string{
			{"P1", fmtVal(e["p1"])}, {"P5", fmtVal(e["p5"])}, {"P10", fmtVal(e["p10"])},
			{"P25", fmtVal(e["p25"])}, {"P50 (Median)", fmtVal(e["p50"])}, {"P75", fmtVal(e["p75"])},
			{"P90", fmtVal(e["p90"])}, {"P95", fmtVal(e["p95"])}, {"P99", fmtVal(e["p99"])},
			{"Mean", fmtVal(e["mean"])}, {"Std Dev", fmtVal(e["stddev"])},
			{"Min", fmtVal(e["min_val"])}, {"Max", fmtVal(e["max_val"])},
		}))
	}
	sb.WriteString("\n")

	writePie(&sb, "6", "Short Document Analysis", r.ShortDocDist)
	writePie(&sb, "7", "Character Type Distribution", r.CharTypeDist)
	writeASCIIBar(&sb, "8", "Top 30 Most Frequent Words", r.TopWords)
	writeASCIIBar(&sb, "9", "Top 30 Most Frequent Bigrams", r.TopBigrams)

	// ── Temporal Analysis ──────────────────────────────────────
	sb.WriteString("\n---\n\n## Temporal Analysis\n\n")
	writeXYBar(&sb, "10", "Documents per Year", "Year", "Documents", r.DocsPerYear)

	// Monthly trend - use line chart, sample if too many
	monthly := r.MonthlyTrend
	if len(monthly) > 40 {
		sampled := make([]LabelCount, 0, len(monthly)/3+1)
		for i, m := range monthly {
			if i%3 == 0 {
				sampled = append(sampled, m)
			}
		}
		monthly = sampled
	}
	writeXYLine(&sb, "11", "Monthly Document Trend", "Month", "Documents", monthly)
	writeASCIIBar(&sb, "12", "Top 30 Common Crawl Dumps", r.TopDumps)

	// Date range summary
	sb.WriteString("### 13. Crawl Date Summary\n\n")
	if len(r.DateRange) > 0 {
		e := r.DateRange[0].Extra
		sb.WriteString(mdTable([]string{"Metric", "Value"}, [][]string{
			{"Earliest Date", fmtVal(e["earliest"])},
			{"Latest Date", fmtVal(e["latest"])},
			{"Unique Years", fmtVal(e["unique_years"])},
			{"Unique Months", fmtVal(e["unique_months"])},
			{"Unique Dumps", fmtVal(e["unique_dumps"])},
		}))
	}
	sb.WriteString("\n")

	writePie(&sb, "14", "Day-of-Week Distribution", r.DOWDist)
	writeXYBar(&sb, "15", "Documents by Hour (UTC)", "Hour", "Documents", r.HourDist)

	// Year-over-year growth
	if len(r.DocsPerYear) > 1 {
		sb.WriteString("### 16. Year-over-Year Growth\n\n")
		growthCats := make([]string, len(r.DocsPerYear)-1)
		growthVals := make([]float64, len(r.DocsPerYear)-1)
		for i := 1; i < len(r.DocsPerYear); i++ {
			growthCats[i-1] = r.DocsPerYear[i].Label
			prev := r.DocsPerYear[i-1].Count
			if prev > 0 {
				growthVals[i-1] = (r.DocsPerYear[i].Count - prev) / prev * 100
			}
		}
		sb.WriteString(MermaidXYBar("Year-over-Year Growth (%)", "Year", "Growth %", growthCats, growthVals))
		sb.WriteString("\n")
	}

	// Dump timeline
	timeline := r.DumpTimeline
	if len(timeline) > 40 {
		timeline = timeline[len(timeline)-40:]
	}
	writeASCIIBar(&sb, "17", "Dump Timeline (Chronological)", timeline)
	writeXYBar(&sb, "18", "Quarterly Document Volume", "Quarter", "Documents", r.QuarterlyDist)

	// ── Domain Analysis ────────────────────────────────────────
	sb.WriteString("\n---\n\n## URL & Domain Analysis\n\n")
	writeASCIIBar(&sb, "19", "Top 30 Domains", r.TopDomains)
	writePie(&sb, "20", "TLD Distribution", r.TLDDist)
	writeXYLine(&sb, "21", "Domain Diversity Over Time", "Year", "Unique Domains", r.DomainDiversityByYear)
	writeXYBar(&sb, "22", "URL Path Depth", "Depth", "Documents", r.PathDepthDist)
	writePie(&sb, "23", "Protocol Distribution", r.ProtocolDist)
	writeXYBar(&sb, "24", "URL Length Distribution", "Length", "Documents", r.URLLengthDist)
	writePie(&sb, "25", "Subdomain Analysis", r.SubdomainDist)
	writeASCIIBar(&sb, "26", "Top 20 Vietnamese Domains (.vn)", r.VNDomains)

	// Domain concentration
	sb.WriteString("### 27. Domain Concentration\n\n")
	if len(r.DomainConcentration) > 0 {
		e := r.DomainConcentration[0].Extra
		sb.WriteString(fmt.Sprintf("- Total unique domains: **%s**\n", fmtVal(e["unique_domains"])))
		sb.WriteString(fmt.Sprintf("- Top 10 domains cover: **%s%%** of all documents\n", fmtVal(e["top10_pct"])))
		sb.WriteString(fmt.Sprintf("- Top 100 domains cover: **%s%%** of all documents\n", fmtVal(e["top100_pct"])))
	}
	sb.WriteString("\n")

	writeASCIIBar(&sb, "28", "Top Domains by Average Text Length", r.DomainAvgTextLen)
	writeXYBar(&sb, "29", "New Domains per Year", "Year", "New Domains", r.NewDomainsPerYear)
	writePie(&sb, "30", "Query Parameter Prevalence", r.QueryParamDist)

	// ── Quality & Deduplication ────────────────────────────────
	sb.WriteString("\n---\n\n## Quality & Deduplication\n\n")
	writeXYBar(&sb, "31", "Language Score Distribution", "Score", "Documents", r.LangScoreDist)

	// Lang score percentiles
	sb.WriteString("### 32. Language Score Percentiles\n\n")
	if len(r.LangScorePercentiles) > 0 {
		e := r.LangScorePercentiles[0].Extra
		sb.WriteString(mdTable([]string{"Percentile", "Value"}, [][]string{
			{"P1", fmtVal(e["p1"])}, {"P5", fmtVal(e["p5"])}, {"P10", fmtVal(e["p10"])},
			{"P25", fmtVal(e["p25"])}, {"P50 (Median)", fmtVal(e["p50"])}, {"P75", fmtVal(e["p75"])},
			{"P90", fmtVal(e["p90"])}, {"P95", fmtVal(e["p95"])}, {"P99", fmtVal(e["p99"])},
			{"Mean", fmtVal(e["mean"])}, {"Std Dev", fmtVal(e["stddev"])},
			{"Min", fmtVal(e["min_val"])}, {"Max", fmtVal(e["max_val"])},
		}))
	}
	sb.WriteString("\n")

	writeXYBar(&sb, "33", "MinHash Cluster Size Distribution", "Size", "Documents", r.ClusterSizeDist)
	writePie(&sb, "34", "Cluster Size Categories", r.ClusterCategories)

	// Score vs text len correlation
	sb.WriteString("### 35. Language Score vs Text Length\n\n")
	if len(r.ScoreVsTextLen) > 0 {
		rows := make([][]string, len(r.ScoreVsTextLen))
		for i, lc := range r.ScoreVsTextLen {
			rows[i] = []string{
				fmtVal(lc.Extra["band"]),
				FormatInt(int64(lc.Count)),
				fmtVal(lc.Extra["avg_len"]),
				fmtVal(lc.Extra["median_len"]),
			}
		}
		sb.WriteString(mdTable([]string{"Score Band", "Documents", "Mean Text Len", "Median Text Len"}, rows))
	}
	sb.WriteString("\n")

	// Score vs cluster summary
	sb.WriteString("### 36. Score & Cluster Summary\n\n")
	if len(r.ScoreVsCluster) > 0 {
		e := r.ScoreVsCluster[0].Extra
		sb.WriteString(mdTable([]string{"Metric", "Language Score", "Cluster Size"}, [][]string{
			{"Mean", fmtVal(e["avg_score"]), fmtVal(e["avg_cluster"])},
			{"Std Dev", fmtVal(e["std_score"]), fmtVal(e["std_cluster"])},
			{"Median", fmtVal(e["med_score"]), fmtVal(e["med_cluster"])},
		}))
	}
	sb.WriteString("\n")

	writePie(&sb, "37", "Quality Score Bands", r.QualityBands)
	writePie(&sb, "38", "top_langs Field Completeness", r.TopLangsField)
	writeASCIIBar(&sb, "39", "Avg Cluster Size by Top Dumps", r.AvgClusterByDump)

	// ── Vietnamese Content Analysis ────────────────────────────
	sb.WriteString("\n---\n\n## Vietnamese Content Analysis\n\n")
	writePie(&sb, "40", "Vietnamese Tone Distribution", r.ToneDist)
	writeASCIIBar(&sb, "41", "Vietnamese Diacritic Frequency", r.DiacriticFreq)
	writeASCIIBar(&sb, "42", "Vietnamese Vowel Frequency", r.VowelFreq)
	writePie(&sb, "43", "Vietnamese Character Density", r.VNCharDensity)
	writeASCIIBar(&sb, "44", "Vietnamese Stop Words", r.StopWordFreq)
	writePie(&sb, "45", "Sentence-Ending Punctuation", r.PunctuationDist)
	writePie(&sb, "46", "Numeric Content Density", r.NumericDensityDist)
	writePie(&sb, "47", "Content Cleanliness", r.BoilerplateDist)
	writePie(&sb, "48", "Content Type Classification", r.ContentTypeDist)

	// Complexity by dump
	sb.WriteString("### 49. Vietnamese Complexity by Dump\n\n")
	if len(r.AvgComplexityByDump) > 0 {
		rows := make([][]string, len(r.AvgComplexityByDump))
		for i, lc := range r.AvgComplexityByDump {
			rows[i] = []string{lc.Label, fmtVal(lc.Extra["avg_ratio"]), fmtVal(lc.Extra["cnt"])}
		}
		sb.WriteString(mdTable([]string{"Dump", "Avg Diacritic Ratio", "Documents"}, rows))
	}
	sb.WriteString("\n")

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Report generated in %s using DuckDB analytics*\n", r.Duration.Round(time.Millisecond)))

	return sb.String()
}

// ── Chart Helpers ──────────────────────────────────────────────

func writeXYBar(sb *strings.Builder, num, title, xLabel, yLabel string, data []LabelCount) {
	sb.WriteString(fmt.Sprintf("### %s. %s\n\n", num, title))
	if len(data) == 0 {
		sb.WriteString("*No data*\n\n")
		return
	}
	cats := make([]string, len(data))
	vals := make([]float64, len(data))
	for i, d := range data {
		cats[i] = d.Label
		vals[i] = d.Count
	}
	sb.WriteString(MermaidXYBar(title, xLabel, yLabel, cats, vals))
	sb.WriteString("\n")
}

func writeXYLine(sb *strings.Builder, num, title, xLabel, yLabel string, data []LabelCount) {
	sb.WriteString(fmt.Sprintf("### %s. %s\n\n", num, title))
	if len(data) == 0 {
		sb.WriteString("*No data*\n\n")
		return
	}
	cats := make([]string, len(data))
	vals := make([]float64, len(data))
	for i, d := range data {
		cats[i] = d.Label
		vals[i] = d.Count
	}
	sb.WriteString(MermaidXYLine(title, xLabel, yLabel, cats, vals))
	sb.WriteString("\n")
}

func writePie(sb *strings.Builder, num, title string, data []LabelCount) {
	sb.WriteString(fmt.Sprintf("### %s. %s\n\n", num, title))
	if len(data) == 0 {
		sb.WriteString("*No data*\n\n")
		return
	}
	slices := make([]PieSlice, len(data))
	for i, d := range data {
		slices[i] = PieSlice{Label: d.Label, Value: d.Count}
	}
	sb.WriteString(MermaidPie(title, slices))
	sb.WriteString("\n")
}

func writeASCIIBar(sb *strings.Builder, num, title string, data []LabelCount) {
	sb.WriteString(fmt.Sprintf("### %s. %s\n\n", num, title))
	if len(data) == 0 {
		sb.WriteString("*No data*\n\n")
		return
	}
	bars := make([]BarItem, len(data))
	for i, d := range data {
		bars[i] = BarItem{Label: d.Label, Value: d.Count}
	}
	sb.WriteString(ASCIIBar(title, bars, 40))
	sb.WriteString("\n")
}

func mdTable(headers []string, rows [][]string) string {
	return MarkdownTable(headers, rows)
}

func fmtVal(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
