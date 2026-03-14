package hn2

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

// ReadmeData holds all template variables for the HN README.
type ReadmeData struct {
	TotalHistoricalItems int64
	TotalMonths          int
	FirstMonth           string
	LastMonth            string
	HistoricalSizeBytes  int64
	AvgFetchSec          float64
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
}

// BuildReadmeData aggregates stats from both CSV files into ReadmeData.
func BuildReadmeData(months []MonthRow, today []TodayRow) ReadmeData {
	d := ReadmeData{}
	for _, r := range months {
		d.TotalHistoricalItems += r.Count
		d.TotalMonths++
		d.HistoricalSizeBytes += r.SizeBytes
		d.AvgFetchSec += float64(r.DurFetchS)
		ym := fmt.Sprintf("%04d-%02d", r.Year, r.Month)
		if d.FirstMonth == "" || ym < d.FirstMonth {
			d.FirstMonth = ym
		}
		if ym > d.LastMonth {
			d.LastMonth = ym
		}
	}
	if d.TotalMonths > 0 {
		d.AvgFetchSec /= float64(d.TotalMonths)
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
	return d
}

// GenerateREADME renders the embedded template with data from both CSV files.
func GenerateREADME(tmplBytes []byte, months []MonthRow, today []TodayRow) ([]byte, error) {
	t, err := template.New("readme").Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("parse readme template: %w", err)
	}
	data := BuildReadmeData(months, today)
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render readme template: %w", err)
	}
	return buf.Bytes(), nil
}
