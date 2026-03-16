package arctic

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type StatsRow struct {
	Year         int
	Month        int
	Type         string
	Shards       int
	Count        int64
	SizeBytes    int64  // total Parquet size across all shards
	ZstBytes     int64  // original .zst source file size (0 = not recorded)
	DurDownloadS float64
	DurProcessS  float64
	DurCommitS   float64
	CommittedAt  time.Time
}

func (r StatsRow) Key() string {
	return fmt.Sprintf("%04d-%02d/%s", r.Year, r.Month, r.Type)
}

func ReadStatsCSV(path string) ([]StatsRow, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	if _, err := r.Read(); err != nil { // skip header
		return nil, nil
	}
	var rows []StatsRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 10 {
			continue
		}
		var row StatsRow
		row.Year, _         = strconv.Atoi(rec[0])
		row.Month, _        = strconv.Atoi(rec[1])
		row.Type             = rec[2]
		row.Shards, _       = strconv.Atoi(rec[3])
		row.Count, _        = strconv.ParseInt(rec[4], 10, 64)
		row.SizeBytes, _    = strconv.ParseInt(rec[5], 10, 64)
		// rec[6] is zst_bytes (added in v2); older files have dur_download_s here
		if len(rec) >= 11 {
			row.ZstBytes, _     = strconv.ParseInt(rec[6], 10, 64)
			row.DurDownloadS, _ = strconv.ParseFloat(rec[7], 64)
			row.DurProcessS, _  = strconv.ParseFloat(rec[8], 64)
			row.DurCommitS, _   = strconv.ParseFloat(rec[9], 64)
			row.CommittedAt, _  = time.Parse(time.RFC3339, rec[10])
		} else {
			row.DurDownloadS, _ = strconv.ParseFloat(rec[6], 64)
			row.DurProcessS, _  = strconv.ParseFloat(rec[7], 64)
			row.DurCommitS, _   = strconv.ParseFloat(rec[8], 64)
			row.CommittedAt, _  = time.Parse(time.RFC3339, rec[9])
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func WriteStatsCSV(path string, rows []StatsRow) error {
	index := make(map[string]StatsRow)
	for _, r := range rows {
		index[r.Key()] = r
	}
	merged := make([]StatsRow, 0, len(index))
	for _, r := range index {
		merged = append(merged, r)
	}
	sort.Slice(merged, func(i, j int) bool {
		a, b := merged[i], merged[j]
		if a.Year != b.Year   { return a.Year < b.Year }
		if a.Month != b.Month { return a.Month < b.Month }
		return a.Type < b.Type
	})
	return writeCSVAtomic(path, merged)
}

// CommittedSet returns keys for months that were fully committed to HuggingFace.
// A row with DurCommitS == 0 means the process was killed (SIGKILL) during or
// before the HF upload — the commit may not have landed. Such rows are excluded
// so the month is re-processed on restart rather than silently skipped.
func CommittedSet(rows []StatsRow) map[string]bool {
	m := make(map[string]bool, len(rows))
	for _, r := range rows {
		if r.DurCommitS > 0 {
			m[r.Key()] = true
		}
	}
	return m
}

func writeCSVAtomic(path string, rows []StatsRow) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".stats_*.csv")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath)
	}()

	w := csv.NewWriter(tmp)
	_ = w.Write([]string{"year","month","type","shards","count","size_bytes","zst_bytes",
		"dur_download_s","dur_process_s","dur_commit_s","committed_at"})
	for _, r := range rows {
		_ = w.Write([]string{
			strconv.Itoa(r.Year),
			strconv.Itoa(r.Month),
			r.Type,
			strconv.Itoa(r.Shards),
			strconv.FormatInt(r.Count, 10),
			strconv.FormatInt(r.SizeBytes, 10),
			strconv.FormatInt(r.ZstBytes, 10),
			strconv.FormatFloat(r.DurDownloadS, 'f', 2, 64),
			strconv.FormatFloat(r.DurProcessS, 'f', 2, 64),
			strconv.FormatFloat(r.DurCommitS, 'f', 2, 64),
			r.CommittedAt.UTC().Format(time.RFC3339),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
