package cli

import (
	"context"
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
)

//go:embed embed/chart_stats.py
var chartStatsPy []byte

// ccShardStats holds per-shard export statistics including pipeline timings.
type ccShardStats struct {
	CrawlID      string
	FileIdx      int
	Rows         int64
	HTMLBytes    int64
	MDBytes      int64
	ParquetBytes int64
	// Timing and metadata (zero-valued for rows loaded from old CSV format)
	CreatedAt     string // RFC3339
	DurDownloadS  int64  // seconds: raw WARC download from Common Crawl S3
	DurConvertS   int64  // seconds: HTML→Markdown conversion (pack)
	DurExportS    int64  // seconds: Parquet export
	DurPublishS   int64  // seconds: HuggingFace commit
}

// ccTotals is the aggregate across all shards for a crawl.
type ccTotals struct {
	Shards        int
	Rows          int64
	HTMLBytes     int64
	MDBytes       int64
	ParquetBytes  int64
	DurDownloadS  int64
	DurConvertS   int64
	DurExportS    int64
	DurPublishS   int64
}

func ccStatsCSVPath(repoRoot string) string {
	return filepath.Join(repoRoot, "stats.csv")
}

var ccStatsCSVHeader = []string{
	"crawl_id", "file_idx", "rows", "html_bytes", "md_bytes", "parquet_bytes",
	"created_at", "dur_download_s", "dur_convert_s", "dur_export_s", "dur_publish_s",
}

func ccReadStatsCSV(csvPath string) ([]ccShardStats, error) {
	f, err := os.Open(csvPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Read() // skip header
	var stats []ccShardStats
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) < 6 {
			continue
		}
		idx, _ := strconv.Atoi(row[1])
		rows, _ := strconv.ParseInt(row[2], 10, 64)
		htmlBytes, _ := strconv.ParseInt(row[3], 10, 64)
		mdBytes, _ := strconv.ParseInt(row[4], 10, 64)
		parquetBytes, _ := strconv.ParseInt(row[5], 10, 64)
		s := ccShardStats{
			CrawlID: row[0], FileIdx: idx,
			Rows: rows, HTMLBytes: htmlBytes, MDBytes: mdBytes, ParquetBytes: parquetBytes,
		}
		// Timing columns (backward compat: may be absent in old CSVs).
		// New format (11 cols): created_at, dur_download_s, dur_convert_s, dur_export_s, dur_publish_s
		// Old format (10 cols): created_at, dur_pack_s, dur_export_s, dur_publish_s
		if len(row) > 6 {
			s.CreatedAt = row[6]
		}
		if len(row) >= 11 {
			// New 11-column format.
			s.DurDownloadS, _ = strconv.ParseInt(row[7], 10, 64)
			s.DurConvertS, _ = strconv.ParseInt(row[8], 10, 64)
			s.DurExportS, _ = strconv.ParseInt(row[9], 10, 64)
			s.DurPublishS, _ = strconv.ParseInt(row[10], 10, 64)
		} else if len(row) >= 10 {
			// Old 10-column format: dur_pack_s mapped to DurConvertS for display.
			s.DurConvertS, _ = strconv.ParseInt(row[7], 10, 64)
			s.DurExportS, _ = strconv.ParseInt(row[8], 10, 64)
			s.DurPublishS, _ = strconv.ParseInt(row[9], 10, 64)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func ccWriteStatsCSV(csvPath string, allStats []ccShardStats) error {
	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write(ccStatsCSVHeader)
	for _, s := range allStats {
		_ = w.Write([]string{
			s.CrawlID,
			strconv.Itoa(s.FileIdx),
			strconv.FormatInt(s.Rows, 10),
			strconv.FormatInt(s.HTMLBytes, 10),
			strconv.FormatInt(s.MDBytes, 10),
			strconv.FormatInt(s.ParquetBytes, 10),
			s.CreatedAt,
			strconv.FormatInt(s.DurDownloadS, 10),
			strconv.FormatInt(s.DurConvertS, 10),
			strconv.FormatInt(s.DurExportS, 10),
			strconv.FormatInt(s.DurPublishS, 10),
		})
	}
	w.Flush()
	return w.Error()
}

// ccUpsertShardStats adds or replaces a shard's stats in the CSV file.
func ccUpsertShardStats(csvPath string, stat ccShardStats) error {
	existing, err := ccReadStatsCSV(csvPath)
	if err != nil {
		return err
	}
	var updated []ccShardStats
	for _, s := range existing {
		if s.CrawlID != stat.CrawlID || s.FileIdx != stat.FileIdx {
			updated = append(updated, s)
		}
	}
	updated = append(updated, stat)
	sort.Slice(updated, func(i, j int) bool {
		if updated[i].CrawlID != updated[j].CrawlID {
			return updated[i].CrawlID < updated[j].CrawlID
		}
		return updated[i].FileIdx < updated[j].FileIdx
	})
	return ccWriteStatsCSV(csvPath, updated)
}

// ccMergeStatsFromHF downloads stats.csv from HF and merges it into the local
// file, with local rows winning on conflict (same crawl_id+file_idx). This
// makes HF the single source of truth: all servers contribute their rows and
// every session sees the global view before each commit.
func ccMergeStatsFromHF(ctx context.Context, hf *hfClient, repoID, statsCSV string) {
	url := "https://huggingface.co/datasets/" + repoID + "/resolve/main/stats.csv"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+hf.token)
	resp, err := hf.http.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return // HF not reachable or file doesn't exist yet — skip silently
	}
	defer resp.Body.Close()

	r := csv.NewReader(resp.Body)
	r.Read() // skip header
	var remote []ccShardStats
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) < 6 {
			continue
		}
		idx, _ := strconv.Atoi(row[1])
		rows, _ := strconv.ParseInt(row[2], 10, 64)
		htmlB, _ := strconv.ParseInt(row[3], 10, 64)
		mdB, _ := strconv.ParseInt(row[4], 10, 64)
		pqB, _ := strconv.ParseInt(row[5], 10, 64)
		s := ccShardStats{CrawlID: row[0], FileIdx: idx, Rows: rows, HTMLBytes: htmlB, MDBytes: mdB, ParquetBytes: pqB}
		if len(row) > 6 {
			s.CreatedAt = row[6]
		}
		if len(row) >= 11 {
			s.DurDownloadS, _ = strconv.ParseInt(row[7], 10, 64)
			s.DurConvertS, _ = strconv.ParseInt(row[8], 10, 64)
			s.DurExportS, _ = strconv.ParseInt(row[9], 10, 64)
			s.DurPublishS, _ = strconv.ParseInt(row[10], 10, 64)
		}
		remote = append(remote, s)
	}
	if len(remote) == 0 {
		return
	}

	// Merge: start with remote as base, upsert local rows on top.
	local, _ := ccReadStatsCSV(statsCSV)
	localIdx := make(map[string]ccShardStats, len(local))
	for _, s := range local {
		localIdx[fmt.Sprintf("%s/%d", s.CrawlID, s.FileIdx)] = s
	}
	merged := make([]ccShardStats, 0, len(remote)+len(local))
	seen := make(map[string]bool)
	for _, s := range remote {
		key := fmt.Sprintf("%s/%d", s.CrawlID, s.FileIdx)
		if l, ok := localIdx[key]; ok {
			merged = append(merged, l) // local wins
		} else {
			merged = append(merged, s)
		}
		seen[key] = true
	}
	for _, s := range local {
		key := fmt.Sprintf("%s/%d", s.CrawlID, s.FileIdx)
		if !seen[key] {
			merged = append(merged, s) // local-only rows
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].CrawlID != merged[j].CrawlID {
			return merged[i].CrawlID < merged[j].CrawlID
		}
		return merged[i].FileIdx < merged[j].FileIdx
	})
	_ = ccWriteStatsCSV(statsCSV, merged)
}

// ccComputeTotals sums all shard stats for a given crawl (empty = all crawls).
func ccComputeTotals(stats []ccShardStats, crawlID string) ccTotals {
	var t ccTotals
	for _, s := range stats {
		if crawlID != "" && s.CrawlID != crawlID {
			continue
		}
		t.Shards++
		t.Rows += s.Rows
		t.HTMLBytes += s.HTMLBytes
		t.MDBytes += s.MDBytes
		t.ParquetBytes += s.ParquetBytes
		t.DurDownloadS += s.DurDownloadS
		t.DurConvertS += s.DurConvertS
		t.DurExportS += s.DurExportS
		t.DurPublishS += s.DurPublishS
	}
	return t
}

// ccTimingBar renders one row of the ASCII bar chart for the README timing section.
// label must be padded to a fixed width by the caller.
// maxS is the reference value (longest bar = full width).
func ccTimingBar(label string, totalS, avgS, maxS int64) string {
	const barWidth = 24
	filled := 0
	if maxS > 0 && totalS > 0 {
		filled = int(float64(totalS) / float64(maxS) * barWidth)
		if filled < 1 {
			filled = 1
		}
		if filled > barWidth {
			filled = barWidth
		}
	}
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return fmt.Sprintf("%s  %s  total %-12s  avg %s\n", label, bar, ccFmtDuration(totalS), ccFmtDuration(avgS))
}

// ccFmtDuration formats seconds as a human-readable duration string.
func ccFmtDuration(secs int64) string {
	if secs <= 0 {
		return "—"
	}
	d := time.Duration(secs) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ccRunCharts runs the embedded chart_stats.py via uv to generate PNG charts from stats.csv.
// Charts are written to repoRoot/charts/. Returns the paths of generated PNGs (relative to repoRoot).
// Returns nil silently if uv is not installed or chart generation fails.
func ccRunCharts(statsCSV, repoRoot, crawlID string) []string {
	chartsDir := filepath.Join(repoRoot, "charts")
	if err := os.MkdirAll(chartsDir, 0o755); err != nil {
		return nil
	}

	// Write embedded script to a stable cache path; only update if content changed.
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", "open-index")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil
	}
	scriptPath := filepath.Join(cacheDir, "chart_stats.py")
	existing, _ := os.ReadFile(scriptPath)
	if string(existing) != string(chartStatsPy) {
		if err := os.WriteFile(scriptPath, chartStatsPy, 0o755); err != nil {
			return nil
		}
	}

	// Run: uv run <script> <statsCSV> --out <chartsDir> [--crawl <crawlID>]
	args := []string{"run", scriptPath, statsCSV, "--out", chartsDir}
	if crawlID != "" {
		args = append(args, "--crawl", crawlID)
	}
	cmd := exec.Command("uv", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  [charts] skipped: %v\n%s\n", err, string(out))
		return nil
	}

	// Collect generated PNG files.
	entries, _ := os.ReadDir(chartsDir)
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			paths = append(paths, filepath.Join("charts", e.Name()))
		}
	}
	return paths
}

// ccParquetStatsRow is the minimal row struct for scanning parquet stats.
type ccParquetStatsRow struct {
	HTMLLength     int64 `parquet:"html_length"`
	MarkdownLength int64 `parquet:"markdown_length"`
}

// ccScanParquetStats reads row count and byte totals from an existing parquet shard.
func ccScanParquetStats(parquetPath string) (rows, htmlBytes, mdBytes int64, err error) {
	f, err := os.Open(parquetPath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return 0, 0, 0, err
	}
	pf, err := parquet.OpenFile(f, fi.Size())
	if err != nil {
		return 0, 0, 0, err
	}
	reader := parquet.NewGenericReader[ccParquetStatsRow](pf)
	defer reader.Close()
	batch := make([]ccParquetStatsRow, 1000)
	for {
		n, readErr := reader.Read(batch)
		for i := 0; i < n; i++ {
			rows++
			htmlBytes += batch[i].HTMLLength
			mdBytes += batch[i].MarkdownLength
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return rows, htmlBytes, mdBytes, readErr
		}
	}
	return rows, htmlBytes, mdBytes, nil
}
