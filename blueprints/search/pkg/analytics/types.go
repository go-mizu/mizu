package analytics

import (
	"fmt"
	"strings"
	"time"
)

// LabelCount is a generic key-value pair from SQL query results.
type LabelCount struct {
	Label string
	Count float64
	Extra map[string]any // for multi-column results
}

// Report holds all computed analytics data.
type Report struct {
	Summary SummaryStats

	// Text Statistics (Charts 1-12)
	TextLengthDist    []LabelCount
	WordCountDist     []LabelCount
	SentenceCountDist []LabelCount
	LineCountDist     []LabelCount
	TextPercentiles   []LabelCount
	ShortDocDist      []LabelCount
	TopWords          []LabelCount
	TopBigrams        []LabelCount
	CharTypeDist      []LabelCount

	// Temporal (Charts 13-23)
	DocsPerYear   []LabelCount
	MonthlyTrend  []LabelCount
	TopDumps      []LabelCount
	HourDist      []LabelCount
	DOWDist       []LabelCount
	QuarterlyDist []LabelCount
	DateRange     []LabelCount
	DumpTimeline  []LabelCount

	// Domain (Charts 24-35)
	TopDomains           []LabelCount
	TLDDist              []LabelCount
	ProtocolDist         []LabelCount
	PathDepthDist        []LabelCount
	URLLengthDist        []LabelCount
	SubdomainDist        []LabelCount
	VNDomains            []LabelCount
	DomainConcentration  []LabelCount
	DomainAvgTextLen     []LabelCount
	QueryParamDist       []LabelCount
	NewDomainsPerYear    []LabelCount
	DomainDiversityByYear []LabelCount

	// Quality (Charts 36-45)
	LangScoreDist        []LabelCount
	LangScorePercentiles []LabelCount
	ClusterSizeDist      []LabelCount
	ClusterCategories    []LabelCount
	QualityBands         []LabelCount
	ScoreVsTextLen       []LabelCount
	ScoreVsCluster       []LabelCount
	TopLangsField        []LabelCount
	AvgClusterByDump     []LabelCount

	// Vietnamese Content (Charts 46-55)
	ToneDist           []LabelCount
	DiacriticFreq      []LabelCount
	VowelFreq          []LabelCount
	StopWordFreq       []LabelCount
	PunctuationDist    []LabelCount
	ContentTypeDist    []LabelCount
	BoilerplateDist    []LabelCount
	NumericDensityDist []LabelCount
	VNCharDensity      []LabelCount
	AvgComplexityByDump []LabelCount

	Duration time.Duration
}

// SummaryStats holds computed summary statistics for the report header.
type SummaryStats struct {
	TotalDocs      int64
	TotalChars     int64
	TotalWords     int64
	UniqueDomains  int64
	DateRange      string
	AvgLangScore   float64
	AvgTextLength  float64
	MedianTextLen  float64
	AvgClusterSize float64
}

// PieSlice represents a single slice in a pie chart.
type PieSlice struct {
	Label string
	Value float64
}

// BarItem represents a single bar in a bar chart.
type BarItem struct {
	Label string
	Value float64
}

func formatBound(v float64) string {
	if v >= 1000000 {
		return fmtNum(v/1000000) + "M"
	}
	if v >= 1000 {
		return fmtNum(v/1000) + "K"
	}
	return fmtNum(v)
}

func fmtNum(v float64) string {
	if v == float64(int64(v)) {
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", v), "0"), ".")
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

// FormatInt formats an integer with comma separators.
func FormatInt(n int64) string {
	if n < 0 {
		return "-" + FormatInt(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
