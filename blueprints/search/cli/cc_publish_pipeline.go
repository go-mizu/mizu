package cli

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
)

// ccShardStats holds per-shard export statistics including pipeline timings.
type ccShardStats struct {
	CrawlID      string
	FileIdx      int
	Rows         int64
	HTMLBytes    int64
	MDBytes      int64
	ParquetBytes int64
	// Timing and metadata (zero-valued for rows loaded from old CSV format)
	CreatedAt    string // RFC3339
	DurPackS     int64  // seconds: download + HTML→Markdown conversion
	DurExportS   int64  // seconds: Parquet export
	DurPublishS  int64  // seconds: HuggingFace commit
}

// ccTotals is the aggregate across all shards for a crawl.
type ccTotals struct {
	Shards       int
	Rows         int64
	HTMLBytes    int64
	MDBytes      int64
	ParquetBytes int64
	DurPackS     int64
	DurExportS   int64
	DurPublishS  int64
}

func ccStatsCSVPath(repoRoot string) string {
	return filepath.Join(repoRoot, "stats.csv")
}

var ccStatsCSVHeader = []string{
	"crawl_id", "file_idx", "rows", "html_bytes", "md_bytes", "parquet_bytes",
	"created_at", "dur_pack_s", "dur_export_s", "dur_publish_s",
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
		// New timing columns (backward compat: may be absent in old CSVs)
		if len(row) > 6 {
			s.CreatedAt = row[6]
		}
		if len(row) > 7 {
			s.DurPackS, _ = strconv.ParseInt(row[7], 10, 64)
		}
		if len(row) > 8 {
			s.DurExportS, _ = strconv.ParseInt(row[8], 10, 64)
		}
		if len(row) > 9 {
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
			strconv.FormatInt(s.DurPackS, 10),
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
		t.DurPackS += s.DurPackS
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

// ccRunCharts runs chart_stats.py (if available) to generate PNG charts from stats.csv.
// Charts are written to repoRoot/charts/. Returns the list of generated PNG paths (relative to repoRoot).
// Returns nil silently if the script or uv is not found.
func ccRunCharts(statsCSV, repoRoot, crawlID string) []string {
	// Locate the chart script: check well-known repo location.
	home, _ := os.UserHomeDir()
	scriptPath := filepath.Join(home, "github", "go-mizu", "mizu",
		"blueprints", "search", "tools", "open-index", "chart_stats.py")
	if _, err := os.Stat(scriptPath); err != nil {
		return nil // script not found, skip silently
	}

	chartsDir := filepath.Join(repoRoot, "charts")
	if err := os.MkdirAll(chartsDir, 0o755); err != nil {
		return nil
	}

	// Run: uv run <script> <statsCSV> --out <chartsDir> --crawl <crawlID>
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
