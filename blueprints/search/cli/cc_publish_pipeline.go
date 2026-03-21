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

// chartTimeout is the maximum duration for chart generation (prevents kaleido/uv hangs).
const chartTimeout = 5 * time.Minute

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
	PeakRSSMB     int64  // peak RSS in MB during pack (0 = not measured)
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
	// Throughput (computed from CreatedAt timestamps in stats.csv)
	ShardsPerHour float64   // shards/hour over last 24h window
	FirstShard    time.Time // earliest CreatedAt
	LastShard     time.Time // latest CreatedAt
	// Memory profiling (from peak_rss_mb in stats.csv)
	AvgRSSMB int64 // average peak RSS across measured shards
	MaxRSSMB int64 // highest peak RSS seen
}

func ccStatsCSVPath(repoRoot string) string {
	return filepath.Join(repoRoot, "stats.csv")
}

func ccSkippedCSVPath(repoRoot string) string {
	return filepath.Join(repoRoot, "skipped.csv")
}

// ccRecordSkip appends a skip entry to skipped.csv so permanently-failed shards
// are visible without needing to search ephemeral screen session output.
// Fields: crawl_id, file_idx, stage (pack/export/rename), error, skipped_at.
// Appends rather than rewrites to avoid read-modify-write races between sessions.
func ccRecordSkip(csvPath, crawlID string, fileIdx int, stage string, err error) {
	needHeader := !fileExists(csvPath)
	f, openErr := os.OpenFile(csvPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr != nil {
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if needHeader {
		_ = w.Write([]string{"crawl_id", "file_idx", "stage", "error", "skipped_at"})
	}
	errStr := ""
	if err != nil {
		// Truncate very long errors (stack traces etc.) to keep CSV readable.
		errStr = err.Error()
		if len(errStr) > 300 {
			errStr = errStr[:300] + "…"
		}
	}
	_ = w.Write([]string{
		crawlID,
		strconv.Itoa(fileIdx),
		stage,
		errStr,
		time.Now().UTC().Format(time.RFC3339),
	})
	w.Flush()
}

var ccStatsCSVHeader = []string{
	"crawl_id", "file_idx", "rows", "html_bytes", "md_bytes", "parquet_bytes",
	"created_at", "dur_download_s", "dur_convert_s", "dur_export_s", "dur_publish_s",
	"peak_rss_mb",
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
			// New 11+ column format.
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
		if len(row) >= 12 {
			s.PeakRSSMB, _ = strconv.ParseInt(row[11], 10, 64)
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
			strconv.FormatInt(s.PeakRSSMB, 10),
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
// Uses a 2-minute timeout to prevent blocking the watcher on slow HF responses.
func ccMergeStatsFromHF(ctx context.Context, hf *hfClient, repoID, statsCSV string) {
	mergeCtx, mergeCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer mergeCancel()

	url := "https://huggingface.co/datasets/" + repoID + "/resolve/main/stats.csv"
	req, err := http.NewRequestWithContext(mergeCtx, http.MethodGet, url, nil)
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
		if len(row) >= 12 {
			s.PeakRSSMB, _ = strconv.ParseInt(row[11], 10, 64)
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
	var rssSum, rssCount int64
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

		if s.PeakRSSMB > 0 {
			rssSum += s.PeakRSSMB
			rssCount++
			if s.PeakRSSMB > t.MaxRSSMB {
				t.MaxRSSMB = s.PeakRSSMB
			}
		}

		if s.CreatedAt != "" {
			if ts, err := time.Parse(time.RFC3339, s.CreatedAt); err == nil {
				if t.FirstShard.IsZero() || ts.Before(t.FirstShard) {
					t.FirstShard = ts
				}
				if ts.After(t.LastShard) {
					t.LastShard = ts
				}
			}
		}
	}

	if rssCount > 0 {
		t.AvgRSSMB = rssSum / rssCount
	}

	// Compute shards/hour from the time span between first and last shard.
	if t.Shards >= 2 && !t.FirstShard.IsZero() && !t.LastShard.IsZero() {
		span := t.LastShard.Sub(t.FirstShard)
		if span.Hours() > 0.1 {
			t.ShardsPerHour = float64(t.Shards) / span.Hours()
		}
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
// Returns nil silently if uv is not installed, chart generation fails, or the 5-minute timeout expires.
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

	uvBin := resolveUV()
	if uvBin == "" {
		fmt.Printf("  [charts] skipped: uv not found\n")
		return nil
	}

	// Snapshot mtime of existing PNGs before running so we can detect which were
	// actually written (script may only generate a subset of historical chart files).
	before := map[string]time.Time{}
	if entries, err := os.ReadDir(chartsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
				if fi, err := e.Info(); err == nil {
					before[e.Name()] = fi.ModTime()
				}
			}
		}
	}

	// Run: uv run <script> <statsCSV> --out <chartsDir> [--crawl <crawlID>]
	// 5-minute timeout prevents kaleido/uv hangs from blocking the watcher indefinitely.
	args := []string{"run", scriptPath, statsCSV, "--out", chartsDir}
	if crawlID != "" {
		args = append(args, "--crawl", crawlID)
	}
	chartCtx, chartCancel := context.WithTimeout(context.Background(), chartTimeout)
	defer chartCancel()
	cmd := exec.CommandContext(chartCtx, uvBin, args...)
	out, err := cmd.CombinedOutput()
	if chartCtx.Err() == context.DeadlineExceeded {
		fmt.Printf("  [charts] skipped: timed out after %s\n", chartTimeout)
		return nil
	}
	if err != nil {
		outStr := string(out)
		// Kaleido requires browser deps (libnss3 etc.) that may be missing on servers.
		if strings.Contains(outStr, "BrowserDepsError") || strings.Contains(outStr, "chromium") ||
			strings.Contains(outStr, "kaleido") || strings.Contains(outStr, "libnss") {
			fmt.Printf("  [charts] skipped: kaleido browser deps missing (install libnss3 libatk-bridge2.0-0 libcups2 libgbm1 etc.)\n")
		} else {
			fmt.Printf("  [charts] skipped: %v\n%s\n", err, outStr)
		}
		return nil
	}

	// Collect only PNGs that were written or updated in this run (mtime changed).
	entries, _ := os.ReadDir(chartsDir)
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if prev, existed := before[e.Name()]; !existed || fi.ModTime().After(prev) {
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
