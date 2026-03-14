package hn2

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MonthRow is one row in stats.csv (one committed historical month).
type MonthRow struct {
	Year        int
	Month       int
	LowestID    int64
	HighestID   int64
	Count       int64
	DurFetchS   int
	DurCommitS  int
	SizeBytes   int64
	CommittedAt time.Time
}

// TodayRow is one row in stats_today.csv (one committed 5-min live block).
type TodayRow struct {
	Date        string // YYYY-MM-DD
	Block       string // HH:MM
	LowestID    int64
	HighestID   int64
	Count       int64
	DurFetchS   int
	DurCommitS  int
	SizeBytes   int64
	CommittedAt time.Time
}

const statsCSVHeader = "year,month,lowest_id,highest_id,count,dur_fetch_s,dur_commit_s,size_bytes,committed_at"
const statsTodayCSVHeader = "date,block,lowest_id,highest_id,count,dur_fetch_s,dur_commit_s,size_bytes,committed_at"

// ReadStatsCSV reads stats.csv. Returns empty slice if file does not exist.
func ReadStatsCSV(path string) ([]MonthRow, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read stats csv: %w", err)
	}
	var out []MonthRow
	for i, rec := range records {
		if i == 0 {
			continue // skip header
		}
		if len(rec) < 9 {
			continue
		}
		row, err := parseMonthRow(rec)
		if err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

// WriteStatsCSV atomically rewrites stats.csv sorted by (year, month).
// If upsert is true and a row with the same (year, month) already exists, it is replaced.
func WriteStatsCSV(path string, rows []MonthRow, newRow MonthRow, upsert bool) error {
	m := make(map[[2]int]MonthRow)
	for _, r := range rows {
		m[[2]int{r.Year, r.Month}] = r
	}
	if upsert {
		m[[2]int{newRow.Year, newRow.Month}] = newRow
	} else {
		if _, exists := m[[2]int{newRow.Year, newRow.Month}]; !exists {
			m[[2]int{newRow.Year, newRow.Month}] = newRow
		}
	}
	merged := make([]MonthRow, 0, len(m))
	for _, r := range m {
		merged = append(merged, r)
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Year != merged[j].Year {
			return merged[i].Year < merged[j].Year
		}
		return merged[i].Month < merged[j].Month
	})
	return writeCSVAtomic(path, statsCSVHeader, func(w *csv.Writer) error {
		for _, r := range merged {
			w.Write([]string{
				strconv.Itoa(r.Year),
				strconv.Itoa(r.Month),
				strconv.FormatInt(r.LowestID, 10),
				strconv.FormatInt(r.HighestID, 10),
				strconv.FormatInt(r.Count, 10),
				strconv.Itoa(r.DurFetchS),
				strconv.Itoa(r.DurCommitS),
				strconv.FormatInt(r.SizeBytes, 10),
				r.CommittedAt.UTC().Format(time.RFC3339),
			})
		}
		return nil
	})
}

// ReadStatsTodayCSV reads stats_today.csv. Returns empty slice if file does not exist.
func ReadStatsTodayCSV(path string) ([]TodayRow, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read stats_today csv: %w", err)
	}
	var out []TodayRow
	for i, rec := range records {
		if i == 0 {
			continue
		}
		if len(rec) < 9 {
			continue
		}
		row, err := parseTodayRow(rec)
		if err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

// WriteStatsTodayCSV atomically rewrites stats_today.csv sorted by (date, block).
func WriteStatsTodayCSV(path string, rows []TodayRow, newRow TodayRow) error {
	rows = append(rows, newRow)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Date != rows[j].Date {
			return rows[i].Date < rows[j].Date
		}
		return rows[i].Block < rows[j].Block
	})
	return writeCSVAtomic(path, statsTodayCSVHeader, func(w *csv.Writer) error {
		for _, r := range rows {
			w.Write([]string{
				r.Date,
				r.Block,
				strconv.FormatInt(r.LowestID, 10),
				strconv.FormatInt(r.HighestID, 10),
				strconv.FormatInt(r.Count, 10),
				strconv.Itoa(r.DurFetchS),
				strconv.Itoa(r.DurCommitS),
				strconv.FormatInt(r.SizeBytes, 10),
				r.CommittedAt.UTC().Format(time.RFC3339),
			})
		}
		return nil
	})
}

// ClearStatsTodayCSV writes a header-only stats_today.csv.
func ClearStatsTodayCSV(path string) error {
	return writeCSVAtomic(path, statsTodayCSVHeader, func(w *csv.Writer) error { return nil })
}

// CommittedMonthSet returns the set of (year, month) pairs already in stats.csv.
func CommittedMonthSet(rows []MonthRow) map[[2]int]bool {
	s := make(map[[2]int]bool, len(rows))
	for _, r := range rows {
		s[[2]int{r.Year, r.Month}] = true
	}
	return s
}

// MaxHighestID returns the highest highest_id across all MonthRows.
func MaxHighestID(rows []MonthRow) int64 {
	var max int64
	for _, r := range rows {
		if r.HighestID > max {
			max = r.HighestID
		}
	}
	return max
}

// MaxTodayHighestID returns the highest highest_id across TodayRows for a given date.
func MaxTodayHighestID(rows []TodayRow, date string) int64 {
	var max int64
	for _, r := range rows {
		if r.Date == date && r.HighestID > max {
			max = r.HighestID
		}
	}
	return max
}

func writeCSVAtomic(path, header string, write func(*csv.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if _, err := fmt.Fprintln(f, header); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := write(w); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func parseMonthRow(rec []string) (MonthRow, error) {
	year, err := strconv.Atoi(rec[0])
	if err != nil {
		return MonthRow{}, err
	}
	month, err := strconv.Atoi(rec[1])
	if err != nil {
		return MonthRow{}, err
	}
	lowestID, _ := strconv.ParseInt(rec[2], 10, 64)
	highestID, _ := strconv.ParseInt(rec[3], 10, 64)
	count, _ := strconv.ParseInt(rec[4], 10, 64)
	durFetch, _ := strconv.Atoi(rec[5])
	durCommit, _ := strconv.Atoi(rec[6])
	sizeBytes, _ := strconv.ParseInt(rec[7], 10, 64)
	committedAt, _ := time.Parse(time.RFC3339, strings.TrimSpace(rec[8]))
	return MonthRow{
		Year: year, Month: month,
		LowestID: lowestID, HighestID: highestID,
		Count: count, DurFetchS: durFetch, DurCommitS: durCommit,
		SizeBytes: sizeBytes, CommittedAt: committedAt,
	}, nil
}

func parseTodayRow(rec []string) (TodayRow, error) {
	lowestID, _ := strconv.ParseInt(rec[2], 10, 64)
	highestID, _ := strconv.ParseInt(rec[3], 10, 64)
	count, _ := strconv.ParseInt(rec[4], 10, 64)
	durFetch, _ := strconv.Atoi(rec[5])
	durCommit, _ := strconv.Atoi(rec[6])
	sizeBytes, _ := strconv.ParseInt(rec[7], 10, 64)
	committedAt, _ := time.Parse(time.RFC3339, strings.TrimSpace(rec[8]))
	return TodayRow{
		Date: rec[0], Block: rec[1],
		LowestID: lowestID, HighestID: highestID,
		Count: count, DurFetchS: durFetch, DurCommitS: durCommit,
		SizeBytes: sizeBytes, CommittedAt: committedAt,
	}, nil
}
