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
	LastMonthYear        string // year portion of LastMonth, e.g. "2026"
	HistoricalSizeBytes  int64
	TodayDate            string
	TodayBlocks          int
	TodayHours           int    // distinct hours with committed blocks today
	TodayItems           int64
	TodayLastBlock       string
	TodayLastBlockPath   string // TodayLastBlock in HH/MM form for use in paths
	TodayDatePath        string // TodayDate in YYYY/MM/DD form for use in paths, e.g. "2026/03/14"
	TodaySizeBytes       int64

	// Current partial month (ongoing, not yet complete)
	CurrentMonth      string // e.g. "2026-03"
	CurrentMonthYear  string // e.g. "2026"
	CurrentMonthUntil string // last date with data, e.g. "2026-03-13"

	// Combined totals
	TotalItems     int64
	TotalItemsFmt  string // comma-formatted, e.g. "27,694,247"
	TotalSizeBytes int64
	TotalSizeMB    string
	TodaySizeKB    string
	TodayItemsFmt  string // comma-formatted today item count
	LastUpdated    string

	// LatestTime is the most recent data point we have committed, used in "spans to".
	// Derived from TodayDate+TodayLastBlock if available, else SourceMaxTime, else LastMonth.
	LatestTime string

	// Yearly growth bar chart (pre-rendered)
	GrowthChart string
	// Today's hourly bar chart (pre-rendered)
	TodayChart string

	// Analytics (optional — from ClickHouse source)
	HasAnalytics        bool
	SourceMaxTime       string // latest item time in the ClickHouse source, e.g. "2026-03-14 15:30:00 UTC"
	ExpectedTotalItems  string
	TypeTable           string
	ScoreSummary        string
	TopAuthorsTable     string
	TopDomainsTable     string
	UniqueAuthors       string
	StoriesWithURLPct   string
	AvgDescendants      string
	MaxDescendants      string
	TotalStories        string
	TotalComments       string
}

// GenerateREADME renders the embedded template with data derived from the
// committed stats and optional ClickHouse analytics.
func GenerateREADME(tmplBytes []byte, months []MonthRow, today []TodayRow, analytics *Analytics) ([]byte, error) {
	t, err := template.New("readme").Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("parse readme template: %w", err)
	}
	data := buildReadmeData(months, today, analytics)
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render readme template: %w", err)
	}
	return buf.Bytes(), nil
}

// buildReadmeData aggregates stats from CSV rows and optional analytics into ReadmeData.
func buildReadmeData(months []MonthRow, today []TodayRow, analytics *Analytics) ReadmeData {
	d := ReadmeData{}

	var latestCommit time.Time

	for _, r := range months {
		d.TotalHistoricalItems += r.Count
		d.TotalMonths++
		d.HistoricalSizeBytes += r.SizeBytes
		// Skip year 1970 (Unix epoch bucket) when computing the displayed date range.
		if r.Year != 1970 {
			ym := fmt.Sprintf("%04d-%02d", r.Year, r.Month)
			if d.FirstMonth == "" || ym < d.FirstMonth {
				d.FirstMonth = ym
			}
			if ym > d.LastMonth {
				d.LastMonth = ym
			}
		}
		if r.CommittedAt.After(latestCommit) {
			latestCommit = r.CommittedAt
		}
	}
	// Fallback if only year-1970 data exists.
	if d.FirstMonth == "" && d.TotalMonths > 0 {
		d.FirstMonth = fmt.Sprintf("%04d-%02d", months[0].Year, months[0].Month)
		d.LastMonth = fmt.Sprintf("%04d-%02d", months[len(months)-1].Year, months[len(months)-1].Month)
	}

	todayHoursSeen := make(map[int]bool)
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
		if len(r.Block) >= 2 {
			var h int
			fmt.Sscanf(r.Block[:2], "%d", &h)
			todayHoursSeen[h] = true
		}
	}
	d.TodayHours = len(todayHoursSeen)
	d.TodayLastBlockPath = strings.ReplaceAll(d.TodayLastBlock, ":", "/")
	d.TodayDatePath = strings.ReplaceAll(d.TodayDate, "-", "/")

	d.TotalItems = d.TotalHistoricalItems + d.TodayItems
	d.TotalItemsFmt = fmtInt(d.TotalItems)
	d.TotalSizeBytes = d.HistoricalSizeBytes + d.TodaySizeBytes
	d.TotalSizeMB = fmt.Sprintf("%.1f", float64(d.TotalSizeBytes)/1024/1024)
	d.TodaySizeKB = fmt.Sprintf("%.1f", float64(d.TodaySizeBytes)/1024)
	d.TodayItemsFmt = fmtInt(d.TodayItems)
	if !latestCommit.IsZero() {
		d.LastUpdated = latestCommit.UTC().Format("2006-01-02 15:04 UTC")
	} else {
		d.LastUpdated = "—"
	}

	if len(d.LastMonth) >= 4 {
		d.LastMonthYear = d.LastMonth[:4]
	}
	if d.TodayDate != "" {
		parts := strings.SplitN(d.TodayDate, "-", 3)
		if len(parts) == 3 {
			d.CurrentMonthYear = parts[0]
			d.CurrentMonth = parts[0] + "-" + parts[1]
			if t, err := time.Parse("2006-01-02", d.TodayDate); err == nil {
				d.CurrentMonthUntil = t.AddDate(0, 0, -1).Format("2006-01-02")
			}
		}
	}
	d.GrowthChart = buildGrowthChart(months)
	d.TodayChart = buildTodayChart(today)

	if analytics != nil {
		d.HasAnalytics = true
		total := analytics.Stories + analytics.Comments + analytics.Jobs + analytics.Polls + analytics.PollOpts
		d.ExpectedTotalItems = fmtInt(total)
		if analytics.SourceMaxTime != "" {
			// Format as "YYYY-MM-DD HH:MM UTC" (strip seconds)
			parts := strings.Fields(analytics.SourceMaxTime) // ["2026-03-14", "17:10:00"]
			if len(parts) >= 2 {
				hhmm := strings.Join(strings.SplitN(parts[1], ":", 3)[:2], ":") // "17:10"
				d.SourceMaxTime = parts[0] + " " + hhmm + " UTC"
			} else {
				d.SourceMaxTime = analytics.SourceMaxTime + " UTC"
			}
		}
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

	// LatestTime: prefer actual committed data; fall back to analytics SourceMaxTime, then LastMonth.
	// This ensures we never show a stale cached timestamp from the analytics query.
	if d.TodayDate != "" && d.TodayLastBlock != "" {
		d.LatestTime = d.TodayDate + " " + d.TodayLastBlock + " UTC"
	} else if d.SourceMaxTime != "" {
		d.LatestTime = d.SourceMaxTime
	} else {
		d.LatestTime = d.LastMonth
	}

	return d
}

// buildGrowthChart renders a Unicode bar chart of items per year.
func buildGrowthChart(months []MonthRow) string {
	yearly := make(map[int]int64)
	for _, r := range months {
		yearly[r.Year] += r.Count
	}
	if len(yearly) == 0 {
		return "  (no data yet)"
	}
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
	const barWidth = 30
	var sb strings.Builder
	for y := minYear; y <= maxYear; y++ {
		if y == 1970 {
			continue // skip Unix epoch bucket
		}
		c := yearly[y]
		if c == 0 {
			continue
		}
		width := int(float64(c) / float64(maxCount) * barWidth)
		if width == 0 {
			width = 1
		}
		bar := strings.Repeat("█", width) + strings.Repeat("░", barWidth-width)
		sb.WriteString(fmt.Sprintf("  %d  %s  %s\n", y, bar, fmtCount(c)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// buildTypeTable renders a Markdown table of item type counts and percentages.
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

// buildScoreSummary renders a Markdown table of score statistics.
func buildScoreSummary(a *Analytics) string {
	var sb strings.Builder
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|------:|\n")
	sb.WriteString(fmt.Sprintf("| Average score | %.1f |\n", a.AvgScore))
	sb.WriteString(fmt.Sprintf("| Median score | %s |\n", fmtInt(a.MedianScore)))
	sb.WriteString(fmt.Sprintf("| Highest score ever | %s |\n", fmtInt(a.MaxScore)))
	sb.WriteString(fmt.Sprintf("| Stories with 100+ points | %s |\n", fmtInt(a.StoriesOver100)))
	sb.WriteString(fmt.Sprintf("| Stories with 1,000+ points | %s |\n", fmtInt(a.StoriesOver1000)))
	return strings.TrimRight(sb.String(), "\n")
}

// buildTopAuthorsTable renders a Markdown ranked table of top story submitters.
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

// buildTodayChart renders a Unicode bar chart of items per committed hour today.
func buildTodayChart(today []TodayRow) string {
	if len(today) == 0 {
		return "  (no data committed today yet)"
	}
	hourly := make(map[int]int64)
	var maxCount int64
	for _, r := range today {
		h := 0
		if len(r.Block) >= 2 {
			fmt.Sscanf(r.Block[:2], "%d", &h)
		}
		hourly[h] += r.Count
		if hourly[h] > maxCount {
			maxCount = hourly[h]
		}
	}
	if maxCount == 0 {
		return "  (no data committed today yet)"
	}
	const barWidth = 30
	var sb strings.Builder
	for h := 0; h < 24; h++ {
		c := hourly[h]
		if c == 0 {
			continue
		}
		width := int(float64(c) / float64(maxCount) * barWidth)
		if width == 0 {
			width = 1
		}
		bar := strings.Repeat("█", width) + strings.Repeat("░", barWidth-width)
		sb.WriteString(fmt.Sprintf("  %02d:00  %s  %s\n", h, bar, fmtCount(c)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// buildTopDomainsTable renders a Markdown ranked table of most-linked domains.
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
