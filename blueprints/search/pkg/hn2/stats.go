package hn2

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MonthRow is one row in stats.csv — one committed historical month.
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

// TodayRow is one row in stats_today.csv — one committed 5-minute live block.
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

const (
	statsCSVHeader      = "year,month,lowest_id,highest_id,count,dur_fetch_s,dur_commit_s,size_bytes,committed_at"
	statsTodayCSVHeader = "date,block,lowest_id,highest_id,count,dur_fetch_s,dur_commit_s,size_bytes,committed_at"
)

// ReadStatsCSV reads stats.csv and returns all rows.
// Returns an empty slice if the file does not exist.
func ReadStatsCSV(path string) ([]MonthRow, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read stats.csv: %w", err)
	}
	out := make([]MonthRow, 0, len(records))
	for i, rec := range records {
		if i == 0 {
			continue // skip header
		}
		if len(rec) < 9 {
			continue
		}
		if row, err := parseMonthRow(rec); err == nil {
			out = append(out, row)
		}
	}
	return out, nil
}

// WriteStatsCSV atomically rewrites stats.csv, sorted by (year, month).
// newRow is merged into rows: if upsert is true an existing row with the same
// (year, month) is replaced; otherwise it is a no-op if already present.
func WriteStatsCSV(path string, rows []MonthRow, newRow MonthRow, upsert bool) error {
	m := make(map[[2]int]MonthRow, len(rows)+1)
	for _, r := range rows {
		m[[2]int{r.Year, r.Month}] = r
	}
	key := [2]int{newRow.Year, newRow.Month}
	if upsert {
		m[key] = newRow
	} else if _, exists := m[key]; !exists {
		m[key] = newRow
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
			_ = w.Write([]string{
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

// writeStatsCSVExact atomically rewrites stats.csv with exactly the given rows.
// Used to roll back a pre-commit write when an HF commit fails.
func writeStatsCSVExact(path string, rows []MonthRow) error {
	sorted := make([]MonthRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Year != sorted[j].Year {
			return sorted[i].Year < sorted[j].Year
		}
		return sorted[i].Month < sorted[j].Month
	})
	return writeCSVAtomic(path, statsCSVHeader, func(w *csv.Writer) error {
		for _, r := range sorted {
			_ = w.Write([]string{
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

// ReadStatsTodayCSV reads stats_today.csv and returns all rows.
// Returns an empty slice if the file does not exist.
func ReadStatsTodayCSV(path string) ([]TodayRow, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseStatsTodayCSV(f)
}

// parseStatsTodayCSV parses stats_today.csv rows from any io.Reader.
func parseStatsTodayCSV(r io.Reader) ([]TodayRow, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read stats_today.csv: %w", err)
	}
	out := make([]TodayRow, 0, len(records))
	for i, rec := range records {
		if i == 0 {
			continue
		}
		if len(rec) < 9 {
			continue
		}
		if row, err := parseTodayRow(rec); err == nil {
			out = append(out, row)
		}
	}
	return out, nil
}

// WriteStatsTodayCSV atomically rewrites stats_today.csv with the given rows,
// sorted by (date, block). The caller is responsible for appending or modifying
// rows before passing them in.
func WriteStatsTodayCSV(path string, rows []TodayRow) error {
	sorted := make([]TodayRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Date != sorted[j].Date {
			return sorted[i].Date < sorted[j].Date
		}
		return sorted[i].Block < sorted[j].Block
	})
	return writeCSVAtomic(path, statsTodayCSVHeader, func(w *csv.Writer) error {
		for _, r := range sorted {
			_ = w.Write([]string{
				r.Date, r.Block,
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

// ClearStatsTodayCSV writes a header-only stats_today.csv, effectively resetting
// the live block log after a day rollover.
func ClearStatsTodayCSV(path string) error {
	return WriteStatsTodayCSV(path, nil)
}

// CommittedMonthSet returns the set of (year, month) pairs present in rows.
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

// writeCSVAtomic writes a CSV file atomically using a unique temp file and rename.
// Using os.CreateTemp avoids the fixed-name ".tmp" race when multiple processes
// write to the same path concurrently.
func writeCSVAtomic(path, header string, write func(*csv.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".csv-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
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
