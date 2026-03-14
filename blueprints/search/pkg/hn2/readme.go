package hn2

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// ReadmeData holds all template variables for the HN README.
type ReadmeData struct {
	// Core metrics (from stats.csv)
	TotalHistoricalItems int64
	TotalMonths          int
	FirstMonth           string
	LastMonth            string
	HistoricalSizeBytes  int64
	TodayDate            string
	TodayBlocks          int
	TodayItems           int64
	TodayLastBlock       string
	TodaySizeBytes       int64

	// Computed combined
	TotalItems     int64
	TotalSizeBytes int64
	TotalSizeMB    string
	TodaySizeKB    string
	LastUpdated    string

	// Yearly growth (computed from stats.csv)
	GrowthChart string // pre-rendered bar chart

	// Full dataset expected totals (from ClickHouse source, optional)
	HasAnalytics       bool
	ExpectedTotalItems string // total items in source
	TypeTable          string // pre-rendered type breakdown
	ScoreSummary       string // pre-rendered score stats
	TopAuthorsTable    string // pre-rendered top authors
	TopDomainsTable    string // pre-rendered top domains
	UniqueAuthors      string
	StoriesWithURLPct  string
	AvgDescendants     string
	MaxDescendants     string
	TotalStories       string
	TotalComments      string
}

// BuildReadmeData aggregates stats from CSV files and optional analytics into ReadmeData.
func BuildReadmeData(months []MonthRow, today []TodayRow, analytics *Analytics) ReadmeData {
	d := ReadmeData{}
	for _, r := range months {
		d.TotalHistoricalItems += r.Count
		d.TotalMonths++
		d.HistoricalSizeBytes += r.SizeBytes
		ym := fmt.Sprintf("%04d-%02d", r.Year, r.Month)
		// Skip 1970 (Unix epoch bucket) when computing first/last month display range.
		if r.Year != 1970 {
			if d.FirstMonth == "" || ym < d.FirstMonth {
				d.FirstMonth = ym
			}
			if ym > d.LastMonth {
				d.LastMonth = ym
			}
		}
	}
	// Fallback if only 1970 data exists.
	if d.FirstMonth == "" && d.TotalMonths > 0 {
		d.FirstMonth = fmt.Sprintf("%04d-%02d", months[0].Year, months[0].Month)
		d.LastMonth = fmt.Sprintf("%04d-%02d", months[len(months)-1].Year, months[len(months)-1].Month)
	}
	var latestCommit time.Time
	for _, r := range months {
		if r.CommittedAt.After(latestCommit) {
			latestCommit = r.CommittedAt
		}
	}
	for _, r := range today {
		d.TodayItems += r.Count
		d.TodayBlocks++
		d.TodaySizeBytes += r.SizeBytes
		if d.TodayDate == "" {
			d.TodayDate = r.Date
		}
		if r.Block > d.TodayLastBlock {
			d.TodayLastBlock = r.Block
		}
		if r.CommittedAt.After(latestCommit) {
			latestCommit = r.CommittedAt
		}
	}
	d.TotalItems = d.TotalHistoricalItems + d.TodayItems
	d.TotalSizeBytes = d.HistoricalSizeBytes + d.TodaySizeBytes
	d.TotalSizeMB = fmt.Sprintf("%.1f", float64(d.TotalSizeBytes)/1024/1024)
	d.TodaySizeKB = fmt.Sprintf("%.1f", float64(d.TodaySizeBytes)/1024)
	if !latestCommit.IsZero() {
		d.LastUpdated = latestCommit.UTC().Format("2006-01-02 15:04 UTC")
	} else {
		d.LastUpdated = "—"
	}

	// Build yearly growth chart from stats.csv data.
	d.GrowthChart = buildGrowthChart(months)

	// Integrate analytics if available.
	if analytics != nil {
		d.HasAnalytics = true
		expectedTotal := analytics.Stories + analytics.Comments + analytics.Jobs + analytics.Polls + analytics.PollOpts
		d.ExpectedTotalItems = fmtInt(expectedTotal)
		d.UniqueAuthors = fmtInt(analytics.UniqueAuthors)
		d.TotalStories = fmtInt(analytics.Stories)
		d.TotalComments = fmtInt(analytics.Comments)
		d.StoriesWithURLPct = fmt.Sprintf("%.1f", analytics.StoriesWithURLPct)
		d.AvgDescendants = fmt.Sprintf("%.1f", analytics.AvgDescendants)
		d.MaxDescendants = fmtInt(analytics.MaxDescendants)
		d.TypeTable = buildTypeTable(analytics)
		d.ScoreSummary = buildScoreSummary(analytics)
		d.TopAuthorsTable = buildTopAuthorsTable(analytics.TopAuthors)
		d.TopDomainsTable = buildTopDomainsTable(analytics.TopDomains)
	}

	return d
}

// GenerateREADME renders the embedded template with data from CSV files and optional analytics.
func GenerateREADME(tmplBytes []byte, months []MonthRow, today []TodayRow, analytics *Analytics) ([]byte, error) {
	t, err := template.New("readme").Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("parse readme template: %w", err)
	}
	data := BuildReadmeData(months, today, analytics)
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render readme template: %w", err)
	}
	return buf.Bytes(), nil
}

// buildGrowthChart creates a text bar chart of items per year from stats.csv data.
func buildGrowthChart(months []MonthRow) string {
	// Aggregate items per year.
	yearly := make(map[int]int64)
	for _, r := range months {
		yearly[r.Year] += r.Count
	}
	if len(yearly) == 0 {
		return "  (no data yet)"
	}

	// Find year range and max count.
	minYear, maxYear := 9999, 0
	var maxCount int64
	for y, c := range yearly {
		if y < minYear {
			minYear = y
		}
		if y > maxYear {
			maxYear = y
		}
		if c > maxCount {
			maxCount = c
		}
	}

	// Skip 1970 (unix epoch bucket) in chart display.
	const barWidth = 30
	var sb strings.Builder
	for y := minYear; y <= maxYear; y++ {
		if y == 1970 {
			continue
		}
		c := yearly[y]
		if c == 0 {
			continue
		}
		// Compute proportional bar width.
		width := int(float64(c) / float64(maxCount) * barWidth)
		if width == 0 && c > 0 {
			width = 1
		}
		bar := strings.Repeat("█", width) + strings.Repeat("░", barWidth-width)
		sb.WriteString(fmt.Sprintf("  %d  %s  %s\n", y, bar, fmtCount(c)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// buildTypeTable creates a formatted type breakdown table.
func buildTypeTable(a *Analytics) string {
	total := a.Stories + a.Comments + a.Jobs + a.Polls + a.PollOpts
	if total == 0 {
		return ""
	}
	types := []struct {
		name  string
		count int64
	}{
		{"comment", a.Comments},
		{"story", a.Stories},
		{"job", a.Jobs},
		{"poll", a.Polls},
		{"pollopt", a.PollOpts},
	}
	var sb strings.Builder
	sb.WriteString("| Type | Count | Share |\n")
	sb.WriteString("|------|------:|------:|\n")
	for _, t := range types {
		pct := float64(t.count) / float64(total) * 100
		sb.WriteString(fmt.Sprintf("| %s | %s | %.1f%% |\n", t.name, fmtInt(t.count), pct))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// buildScoreSummary creates a formatted score statistics summary.
func buildScoreSummary(a *Analytics) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	sb.WriteString(fmt.Sprintf("|--------|------:|\n"))
	sb.WriteString(fmt.Sprintf("| Average score | %.1f |\n", a.AvgScore))
	sb.WriteString(fmt.Sprintf("| Median score | %s |\n", fmtInt(a.MedianScore)))
	sb.WriteString(fmt.Sprintf("| Highest score ever | %s |\n", fmtInt(a.MaxScore)))
	sb.WriteString(fmt.Sprintf("| Stories with 100+ points | %s |\n", fmtInt(a.StoriesOver100)))
	sb.WriteString(fmt.Sprintf("| Stories with 1,000+ points | %s |\n", fmtInt(a.StoriesOver1000)))
	return strings.TrimRight(sb.String(), "\n")
}

// buildTopAuthorsTable creates a formatted top authors table.
func buildTopAuthorsTable(authors []NameCount) string {
	if len(authors) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("| # | User | Stories |\n")
	sb.WriteString("|--:|------|--------:|\n")
	for i, a := range authors {
		sb.WriteString(fmt.Sprintf("| %d | %s | %s |\n", i+1, a.Name, fmtInt(a.Count)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// buildTopDomainsTable creates a formatted top domains table.
func buildTopDomainsTable(domains []NameCount) string {
	if len(domains) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("| # | Domain | Stories |\n")
	sb.WriteString("|--:|--------|--------:|\n")
	for i, d := range domains {
		sb.WriteString(fmt.Sprintf("| %d | %s | %s |\n", i+1, d.Name, fmtInt(d.Count)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// fmtCount formats a count as a human-readable string (e.g. 1.2M, 320K).
func fmtCount(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
