package cli

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/parquet-go/parquet-go"
)

// ccShardStats holds per-shard export statistics.
type ccShardStats struct {
	CrawlID      string
	FileIdx      int
	Rows         int64
	HTMLBytes    int64
	MDBytes      int64
	ParquetBytes int64
}

// ccTotals is the aggregate across all shards for a crawl.
type ccTotals struct {
	Shards       int
	Rows         int64
	HTMLBytes    int64
	MDBytes      int64
	ParquetBytes int64
}

func ccStatsCSVPath(repoRoot string) string {
	return filepath.Join(repoRoot, "stats.csv")
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
		stats = append(stats, ccShardStats{
			CrawlID: row[0], FileIdx: idx,
			Rows: rows, HTMLBytes: htmlBytes, MDBytes: mdBytes, ParquetBytes: parquetBytes,
		})
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
	_ = w.Write([]string{"crawl_id", "file_idx", "rows", "html_bytes", "md_bytes", "parquet_bytes"})
	for _, s := range allStats {
		_ = w.Write([]string{
			s.CrawlID,
			strconv.Itoa(s.FileIdx),
			strconv.FormatInt(s.Rows, 10),
			strconv.FormatInt(s.HTMLBytes, 10),
			strconv.FormatInt(s.MDBytes, 10),
			strconv.FormatInt(s.ParquetBytes, 10),
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
	}
	return t
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
