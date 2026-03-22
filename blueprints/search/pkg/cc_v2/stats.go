package cc_v2

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// StatsRow holds per-shard statistics for stats.csv.
type StatsRow struct {
	CrawlID   string
	FileIdx   int
	Rows      int64
	HTMLBytes int64
	MDBytes   int64
	PqBytes   int64
	CreatedAt string
	DurDlS    int64
	DurPackS  int64
	DurPushS  int64
	PeakRSSMB int64
}

var statsHeader = []string{
	"crawl_id", "file_idx", "rows", "html_bytes", "md_bytes",
	"parquet_bytes", "created_at", "dur_download_s", "dur_convert_s",
	"dur_publish_s", "peak_rss_mb",
}

func readStatsCSV(path string) ([]StatsRow, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // flexible column count
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var rows []StatsRow
	for i, rec := range records {
		if i == 0 {
			continue // skip header
		}
		if len(rec) < 6 {
			continue
		}
		idx, _ := strconv.Atoi(rec[1])
		nRows, _ := strconv.ParseInt(rec[2], 10, 64)
		html, _ := strconv.ParseInt(rec[3], 10, 64)
		md, _ := strconv.ParseInt(rec[4], 10, 64)
		pq, _ := strconv.ParseInt(rec[5], 10, 64)
		createdAt := ""
		if len(rec) > 6 {
			createdAt = rec[6]
		}
		var durDl, durPack, durPush, peakRSS int64
		if len(rec) > 7 {
			durDl, _ = strconv.ParseInt(rec[7], 10, 64)
		}
		if len(rec) > 8 {
			durPack, _ = strconv.ParseInt(rec[8], 10, 64)
		}
		if len(rec) > 9 {
			durPush, _ = strconv.ParseInt(rec[9], 10, 64)
		}
		if len(rec) > 10 {
			peakRSS, _ = strconv.ParseInt(rec[10], 10, 64)
		}
		rows = append(rows, StatsRow{
			CrawlID:   rec[0],
			FileIdx:   idx,
			Rows:      nRows,
			HTMLBytes: html,
			MDBytes:   md,
			PqBytes:   pq,
			CreatedAt: createdAt,
			DurDlS:    durDl,
			DurPackS:  durPack,
			DurPushS:  durPush,
			PeakRSSMB: peakRSS,
		})
	}
	return rows, nil
}

func writeStatsCSV(path string, rows []StatsRow) error {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CrawlID != rows[j].CrawlID {
			return rows[i].CrawlID < rows[j].CrawlID
		}
		return rows[i].FileIdx < rows[j].FileIdx
	})

	if err := os.MkdirAll(strings.TrimSuffix(path, "/stats.csv"), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write(statsHeader)
	for _, r := range rows {
		w.Write([]string{
			r.CrawlID,
			strconv.Itoa(r.FileIdx),
			strconv.FormatInt(r.Rows, 10),
			strconv.FormatInt(r.HTMLBytes, 10),
			strconv.FormatInt(r.MDBytes, 10),
			strconv.FormatInt(r.PqBytes, 10),
			r.CreatedAt,
			strconv.FormatInt(r.DurDlS, 10),
			strconv.FormatInt(r.DurPackS, 10),
			strconv.FormatInt(r.DurPushS, 10),
			strconv.FormatInt(r.PeakRSSMB, 10),
		})
	}
	w.Flush()
	return w.Error()
}

func upsertStats(existing []StatsRow, row StatsRow) []StatsRow {
	for i, r := range existing {
		if r.CrawlID == row.CrawlID && r.FileIdx == row.FileIdx {
			existing[i] = row // local wins
			return existing
		}
	}
	return append(existing, row)
}

func mergeStatsFromRemote(localPath string, remoteCSV []byte, crawlID string) {
	// Parse remote CSV.
	tmpPath := localPath + ".remote.tmp"
	os.WriteFile(tmpPath, remoteCSV, 0o644)
	defer os.Remove(tmpPath)

	remoteRows, err := readStatsCSV(tmpPath)
	if err != nil || len(remoteRows) == 0 {
		return
	}

	localRows, _ := readStatsCSV(localPath)
	localSet := make(map[string]bool)
	for _, r := range localRows {
		localSet[fmt.Sprintf("%s:%d", r.CrawlID, r.FileIdx)] = true
	}

	// Add remote rows that don't exist locally (remote fills gaps, local wins).
	for _, r := range remoteRows {
		key := fmt.Sprintf("%s:%d", r.CrawlID, r.FileIdx)
		if !localSet[key] {
			localRows = append(localRows, r)
		}
	}

	writeStatsCSV(localPath, localRows)
}

func generateREADME(crawlID string, rows []StatsRow) string {
	var totalRows, totalHTML, totalMD, totalPQ int64
	shards := 0
	for _, r := range rows {
		if r.CrawlID == crawlID {
			shards++
			totalRows += r.Rows
			totalHTML += r.HTMLBytes
			totalMD += r.MDBytes
			totalPQ += r.PqBytes
		}
	}

	return fmt.Sprintf(`---
license: odc-by
task_categories:
  - text-generation
language:
  - en
size_categories:
  - 100M<n<1B
---

# Open Markdown — %s

Common Crawl HTML pages converted to clean Markdown.

## Stats

| Metric | Value |
|--------|-------|
| Shards | %d |
| Documents | %s |
| HTML | %s |
| Markdown | %s |
| Parquet (zstd) | %s |

## Format

Each row in the parquet files contains:

| Column | Type | Description |
|--------|------|-------------|
| doc_id | string | Deterministic hash of URL |
| url | string | Original page URL |
| host | string | Registered domain |
| crawl_date | string | ISO date from WARC |
| warc_record_id | string | UUID |
| warc_refers_to | string | Original WARC record ID |
| html_length | int64 | Raw HTML bytes |
| markdown_length | int64 | Converted Markdown bytes |
| markdown | string | Clean Markdown text |

## Usage

`+"```python"+`
from datasets import load_dataset
ds = load_dataset("open-index/open-markdown", data_files="data/%s/*.parquet")
`+"```"+`

## License

Open Data Commons Attribution License (ODC-By).
`, crawlID, shards, fmtInt(totalRows), FmtBytes(totalHTML), FmtBytes(totalMD), FmtBytes(totalPQ), crawlID)
}

func fmtInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

const licenseText = `Open Data Commons Attribution License (ODC-By) v1.0

You are free to:
- Share: copy, distribute, and use the database.
- Create: produce works from the database.
- Adapt: modify, transform, and build upon the database.

As long as you:
- Attribute: You must attribute any public use of the database,
  or works produced from the database, in the manner specified
  in the ODC-By license.
`
