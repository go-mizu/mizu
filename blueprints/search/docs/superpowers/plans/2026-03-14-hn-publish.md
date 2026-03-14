# HN Publish Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `search hn publish` — a command that streams the full Hacker News dataset to `open-index/hacker-news` on Hugging Face as monthly parquet files, with 5-minute live blocks for today.

**Architecture:** A new `pkg/hn2` package handles all data fetching and task logic (no "ClickHouse" in names — one comment explains the backing source). Three `pkg/core.Task` implementations (Historical, Live, DayRollover) are wired together in `cli/hn_publish.go`. Two CSV files (`stats.csv`, `stats_today.csv`) are the source of truth for what has been committed, enabling safe resume. The `hf_commit.py` Python helper is extended to support delete operations for the day-rollover commit.

**Tech Stack:** Go 1.22+, `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`, `github.com/duckdb/duckdb-go/v2`, `huggingface_hub` (Python via `uv`), `html/template`

**Spec:** `spec/0727_hn_publish.md`

---

## Chunk 1: pkg/hn2 — Config, Client, Fetch

### Task 1: pkg/hn2 Config and Client

**Files:**
- Create: `pkg/hn2/config.go`
- Create: `pkg/hn2/client.go`

- [ ] **Step 1: Create `pkg/hn2/config.go`**

```go
// Package hn2 publishes the Hacker News dataset to Hugging Face.
// Data is fetched from the ClickHouse public SQL playground (sql.clickhouse.com).
package hn2

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultEndpointURL = "https://sql-clickhouse.clickhouse.com"
	defaultUser        = "demo"
	defaultDatabase    = "hackernews"
	defaultTable       = "hackernews"
)

// Config controls the remote data source and local repo root for HN publishing.
type Config struct {
	RepoRoot    string
	EndpointURL string
	Database    string
	Table       string
	User        string
	DNSServer   string
	HTTPClient  *http.Client
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		RepoRoot:    filepath.Join(home, "data", "hn", "repo"),
		EndpointURL: defaultEndpointURL,
		User:        defaultUser,
		Database:    defaultDatabase,
		Table:       defaultTable,
		HTTPClient:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c Config) WithDefaults() Config {
	d := DefaultConfig()
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_ENDPOINT")); v != "" {
		d.EndpointURL = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_USER")); v != "" {
		d.User = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_DATABASE")); v != "" {
		d.Database = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_TABLE")); v != "" {
		d.Table = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_DNS_SERVER")); v != "" {
		d.DNSServer = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_REPO_ROOT")); v != "" {
		d.RepoRoot = v
	}
	if strings.TrimSpace(c.RepoRoot) != "" {
		d.RepoRoot = c.RepoRoot
	}
	if strings.TrimSpace(c.EndpointURL) != "" {
		d.EndpointURL = c.EndpointURL
	}
	if strings.TrimSpace(c.User) != "" {
		d.User = c.User
	}
	if strings.TrimSpace(c.Database) != "" {
		d.Database = c.Database
	}
	if strings.TrimSpace(c.Table) != "" {
		d.Table = c.Table
	}
	if strings.TrimSpace(c.DNSServer) != "" {
		d.DNSServer = c.DNSServer
	}
	if c.HTTPClient != nil {
		d.HTTPClient = c.HTTPClient
	}
	return d
}

// Dir helpers

func (c Config) DataDir() string {
	return filepath.Join(c.WithDefaults().RepoRoot, "data")
}

func (c Config) TodayDir() string {
	return filepath.Join(c.WithDefaults().RepoRoot, "today")
}

func (c Config) MonthDir(year int) string {
	return filepath.Join(c.DataDir(), fmt.Sprintf("%04d", year))
}

func (c Config) MonthPath(year, month int) string {
	return filepath.Join(c.MonthDir(year), fmt.Sprintf("%04d-%02d.parquet", year, month))
}

func (c Config) TodayBlockPath(date, blockHHMM string) string {
	// date = "2026-03-14", blockHHMM = "00:05" → today/2026-03-14_00_05.parquet
	block := strings.ReplaceAll(blockHHMM, ":", "_")
	return filepath.Join(c.TodayDir(), date+"_"+block+".parquet")
}

func (c Config) StatsCSVPath() string {
	return filepath.Join(c.WithDefaults().RepoRoot, "stats.csv")
}

func (c Config) StatsTodayCSVPath() string {
	return filepath.Join(c.WithDefaults().RepoRoot, "stats_today.csv")
}

func (c Config) READMEPath() string {
	return filepath.Join(c.WithDefaults().RepoRoot, "README.md")
}

func (c Config) EnsureDirs() error {
	cfg := c.WithDefaults()
	for _, d := range []string{cfg.DataDir(), cfg.TodayDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) httpClient() *http.Client {
	cfg := c.WithDefaults()
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c Config) fqTable() string {
	cfg := c.WithDefaults()
	return quoteIdent(cfg.Database) + "." + quoteIdent(cfg.Table)
}

func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}
```

(imports already include `"fmt"` in the block above)

- [ ] **Step 2: Create `pkg/hn2/client.go`**

```go
package hn2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RemoteInfo describes the current state of the remote HN data source.
type RemoteInfo struct {
	Count     int64
	MaxID     int64
	MaxTime   string
	CheckedAt time.Time
}

// MonthInfo describes a month available in the remote source.
type MonthInfo struct {
	Year  int
	Month int
	Count int64
}

// RemoteInfo queries the remote source for current item count and max ID.
func (c Config) RemoteInfo(ctx context.Context) (*RemoteInfo, error) {
	cfg := c.WithDefaults()
	q := fmt.Sprintf(
		`SELECT toInt64(count()) AS c, toInt64(max(id)) AS max_id, toString(max(time)) AS max_time FROM %s FORMAT JSONEachRow`,
		cfg.fqTable(),
	)
	body, err := cfg.query(ctx, q)
	if err != nil {
		return nil, err
	}
	var row struct {
		Count   any    `json:"c"`
		MaxID   any    `json:"max_id"`
		MaxTime string `json:"max_time"`
	}
	if err := json.Unmarshal(body, &row); err != nil {
		return nil, fmt.Errorf("decode remote info: %w", err)
	}
	count, err := parseIntAny(row.Count)
	if err != nil {
		return nil, fmt.Errorf("parse remote count: %w", err)
	}
	maxID, err := parseIntAny(row.MaxID)
	if err != nil {
		return nil, fmt.Errorf("parse remote max_id: %w", err)
	}
	return &RemoteInfo{
		Count:     count,
		MaxID:     maxID,
		MaxTime:   row.MaxTime,
		CheckedAt: time.Now().UTC(),
	}, nil
}

// ListMonths returns all months with data available in the remote source.
// The current calendar month is excluded (it is incomplete).
func (c Config) ListMonths(ctx context.Context) ([]MonthInfo, error) {
	cfg := c.WithDefaults()
	q := fmt.Sprintf(
		`SELECT toYear(time) AS y, toMonth(time) AS m, toInt64(count()) AS n FROM %s WHERE time IS NOT NULL GROUP BY y, m ORDER BY y, m FORMAT JSONEachRow`,
		cfg.fqTable(),
	)
	body, err := cfg.query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list months: %w", err)
	}
	now := time.Now().UTC()
	curYear, curMonth := now.Year(), int(now.Month())
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	out := make([]MonthInfo, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Y any `json:"y"`
			M any `json:"m"`
			N any `json:"n"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("decode month row: %w", err)
		}
		y, _ := parseIntAny(row.Y)
		m, _ := parseIntAny(row.M)
		n, _ := parseIntAny(row.N)
		// Exclude the current in-progress month.
		if int(y) == curYear && int(m) == curMonth {
			continue
		}
		out = append(out, MonthInfo{Year: int(y), Month: int(m), Count: n})
	}
	return out, nil
}

func (c Config) query(ctx context.Context, q string) ([]byte, error) {
	req, err := c.newRequest(ctx, q)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote query: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read remote response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("remote query HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (c Config) newRequest(ctx context.Context, q string) (*http.Request, error) {
	cfg := c.WithDefaults()
	u, err := url.Parse(cfg.EndpointURL)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint URL: %w", err)
	}
	qp := u.Query()
	if cfg.User != "" {
		qp.Set("user", cfg.User)
	}
	if cfg.Database != "" {
		qp.Set("database", cfg.Database)
	}
	qp.Set("max_result_rows", "0")
	qp.Set("max_result_bytes", "0")
	qp.Set("result_overflow_mode", "throw")
	u.RawQuery = qp.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(q))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	return req, nil
}

// downloadHTTPClient returns an HTTP client with no timeout for streaming large parquet downloads.
func (c Config) downloadHTTPClient() *http.Client {
	cfg := c.WithDefaults()
	base := cfg.httpClient()
	clone := *base
	clone.Timeout = 0
	if cfg.DNSServer == "" {
		return &clone
	}
	var tr *http.Transport
	if bt, ok := base.Transport.(*http.Transport); ok && bt != nil {
		tr = bt.Clone()
	} else {
		tr = http.DefaultTransport.(*http.Transport).Clone()
	}
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := &net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", cfg.DNSServer)
		},
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  resolver,
	}
	tr.DialContext = dialer.DialContext
	clone.Transport = tr
	return &clone
}

func parseIntAny(v any) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	case json.Number:
		return x.Int64()
	case string:
		return strconv.ParseInt(strings.TrimSpace(x), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}
```

Add `"strconv"` to imports.

- [ ] **Step 3: Build to check for compile errors**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/hn2/...
```

Expected: may fail with missing files (fetch.go) — that's fine. Fix any syntax errors from the above.

- [ ] **Step 4: Commit**

```bash
git add pkg/hn2/config.go pkg/hn2/client.go
git commit -m "feat(hn2): add config, client, remote info and month listing"
```

---

### Task 2: pkg/hn2 Fetch

**Files:**
- Create: `pkg/hn2/fetch.go`

- [ ] **Step 1: Create `pkg/hn2/fetch.go`**

```go
package hn2

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// FetchResult is returned by FetchMonth and FetchSince.
type FetchResult struct {
	LowestID  int64
	HighestID int64
	Count     int64
	Bytes     int64
	Duration  time.Duration
}

// FetchMonth downloads all items for the given year/month as a parquet file to outPath.
// The file is written atomically (outPath.tmp → outPath). If Count==0 the .tmp is removed
// and the caller should skip committing.
func (c Config) FetchMonth(ctx context.Context, year, month int, outPath string) (FetchResult, error) {
	cfg := c.WithDefaults()
	start := time.Now()
	tm := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	next := tm.AddDate(0, 1, 0)
	q := fmt.Sprintf(
		`SELECT * FROM %s WHERE time >= toDateTime('%s') AND time < toDateTime('%s') ORDER BY id FORMAT Parquet`,
		cfg.fqTable(),
		tm.Format("2006-01-02 15:04:05"),
		next.Format("2006-01-02 15:04:05"),
	)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return FetchResult{}, fmt.Errorf("create month dir: %w", err)
	}
	n, err := cfg.streamToFile(ctx, q, outPath)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch month %04d-%02d: %w", year, month, err)
	}
	dur := time.Since(start)
	if n == 0 {
		return FetchResult{Duration: dur}, nil
	}
	return cfg.scanParquetResult(ctx, outPath, n, dur)
}

// FetchSince downloads all items with id > afterID and time < ceilTime as a parquet file.
// ceilTime prevents items from crossing midnight into the next day's block.
func (c Config) FetchSince(ctx context.Context, afterID int64, ceilTime time.Time, outPath string) (FetchResult, error) {
	cfg := c.WithDefaults()
	start := time.Now()
	q := fmt.Sprintf(
		`SELECT * FROM %s WHERE id > %d AND time < toDateTime('%s') ORDER BY id FORMAT Parquet`,
		cfg.fqTable(),
		afterID,
		ceilTime.UTC().Format("2006-01-02 15:04:05"),
	)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return FetchResult{}, fmt.Errorf("create today dir: %w", err)
	}
	n, err := cfg.streamToFile(ctx, q, outPath)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch since %d: %w", afterID, err)
	}
	dur := time.Since(start)
	if n == 0 {
		return FetchResult{Duration: dur}, nil
	}
	return cfg.scanParquetResult(ctx, outPath, n, dur)
}

// streamToFile executes the query and streams the response body to outPath atomically.
// Returns bytes written. If the response is empty (0 bytes), removes the .tmp file and returns 0.
func (c Config) streamToFile(ctx context.Context, q, outPath string) (int64, error) {
	cfg := c.WithDefaults()
	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)

	var lastErr error
	for attempt := 1; attempt <= 4; attempt++ {
		req, err := cfg.newRequest(ctx, q)
		if err != nil {
			return 0, err
		}
		resp, err := cfg.downloadHTTPClient().Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request attempt %d: %w", attempt, err)
			sleepWithContext(ctx, time.Duration(attempt)*500*time.Millisecond)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				return 0, lastErr
			}
			sleepWithContext(ctx, time.Duration(attempt)*500*time.Millisecond)
			continue
		}
		f, err := os.Create(tmpPath)
		if err != nil {
			resp.Body.Close()
			return 0, fmt.Errorf("create tmp file: %w", err)
		}
		n, copyErr := io.Copy(f, resp.Body)
		bodyCloseErr := resp.Body.Close()
		fileCloseErr := f.Close()
		if copyErr != nil || bodyCloseErr != nil || fileCloseErr != nil {
			_ = os.Remove(tmpPath)
			if copyErr != nil {
				lastErr = copyErr
			} else if bodyCloseErr != nil {
				lastErr = bodyCloseErr
			} else {
				lastErr = fileCloseErr
			}
			sleepWithContext(ctx, time.Duration(attempt)*500*time.Millisecond)
			continue
		}
		if n == 0 {
			_ = os.Remove(tmpPath)
			return 0, nil
		}
		if err := os.Rename(tmpPath, outPath); err != nil {
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("rename to output: %w", err)
		}
		return n, nil
	}
	_ = os.Remove(tmpPath)
	return 0, lastErr
}

// scanParquetResult reads MIN/MAX/COUNT from a parquet file via DuckDB.
func (c Config) scanParquetResult(ctx context.Context, path string, bytesWritten int64, dur time.Duration) (FetchResult, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return FetchResult{}, fmt.Errorf("open duckdb for scan: %w", err)
	}
	defer db.Close()
	q := fmt.Sprintf(`SELECT COUNT(*)::BIGINT, MIN(id)::BIGINT, MAX(id)::BIGINT FROM read_parquet('%s')`,
		escapeSQLStr(path))
	var count, minID, maxID int64
	if err := db.QueryRowContext(ctx, q).Scan(&count, &minID, &maxID); err != nil {
		return FetchResult{}, fmt.Errorf("scan parquet result: %w", err)
	}
	return FetchResult{
		LowestID:  minID,
		HighestID: maxID,
		Count:     count,
		Bytes:     bytesWritten,
		Duration:  dur,
	}, nil
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func escapeSQLStr(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
```

- [ ] **Step 2: Build to check compile**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/hn2/...
```

Expected: success (or only missing-file errors for stats.go / task files).

- [ ] **Step 3: Commit**

```bash
git add pkg/hn2/fetch.go
git commit -m "feat(hn2): add FetchMonth and FetchSince with atomic write and retry"
```

---

### Task 3: pkg/hn2 Stats

**Files:**
- Create: `pkg/hn2/stats.go`

- [ ] **Step 1: Create `pkg/hn2/stats.go`**

```go
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

// statsCSVHeader and statsTodayCSVHeader are the exact header lines.
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
	// Build map for dedup/upsert
	m := make(map[[2]int]MonthRow)
	for _, r := range rows {
		m[[2]int{r.Year, r.Month}] = r
	}
	if upsert {
		m[[2]int{newRow.Year, newRow.Month}] = newRow
	} else {
		// append only if not present
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

// CommittedMonthSet returns the set of (year*100+month) keys already in stats.csv.
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

// --- internal helpers ---

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
	// write header manually to avoid quoting
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
```

(imports already include `"path/filepath"` in the block above)

- [ ] **Step 2: Build**

```bash
go build ./pkg/hn2/...
```

- [ ] **Step 3: Commit**

```bash
git add pkg/hn2/stats.go
git commit -m "feat(hn2): add stats.csv and stats_today.csv read/write helpers"
```

---

## Chunk 2: pkg/hn2 — Tasks and README

### Task 4: pkg/hn2 README Generator

**Files:**
- Create: `cli/embed/hn_readme.md.tmpl`
- Create: `pkg/hn2/readme.go`

- [ ] **Step 1: Create `cli/embed/hn_readme.md.tmpl`**

```markdown
---
license: odc-by
task_categories:
- text-generation
- feature-extraction
language:
- en
pretty_name: Hacker News Open Index
size_categories:
- 10M<n<100M
tags:
- hacker-news
- forum
- text
- parquet
configs:
- config_name: default
  data_files:
  - split: train
    path: data/*/*.parquet
- config_name: today
  data_files:
  - split: train
    path: today/*.parquet
---

# Hacker News Open Index

> Every Hacker News item since 2006, updated every 5 minutes — ready for training and retrieval

## What is it?

This dataset contains the full Hacker News archive: **{{.TotalItems}} items** spanning {{.FirstMonth}} to {{.LastUpdated}}, published as monthly Parquet files with 5-minute live blocks for today.

Data includes stories, comments, Ask HN, Show HN, jobs, polls, and poll options — all fields preserved exactly as posted.

## Dataset Stats

| Metric | Value |
|--------|-------|
| Total items | {{.TotalItems}} |
| Historical months | {{.TotalMonths}} |
| First month | {{.FirstMonth}} |
| Last committed month | {{.LastMonth}} |
| Total size | {{.TotalSizeMB}} MB |
| Last updated | {{.LastUpdated}} |

{{- if .TodayBlocks}}

### Live Today ({{.TodayDate}})

| Metric | Value |
|--------|-------|
| Blocks committed | {{.TodayBlocks}} |
| Items today | {{.TodayItems}} |
| Last block | {{.TodayLastBlock}} |
| Today size | {{.TodaySizeKB}} KB |
{{- end}}

## File Layout

```
data/
  2006/2006-10.parquet    ← first HN month
  ...
  2026/2026-02.parquet
today/
  2026-03-14_00_00.parquet   ← 5-min live blocks
  2026-03-14_00_05.parquet
  ...
stats.csv                    ← one row per committed month
stats_today.csv              ← one row per committed 5-min block
```

## How to Use

### Python (datasets)

```python
from datasets import load_dataset

# Stream the full history
ds = load_dataset("open-index/hacker-news", split="train", streaming=True)
for item in ds:
    print(item["id"], item["type"], item["title"])

# Load a single month
ds = load_dataset(
    "open-index/hacker-news",
    data_files="data/2024/2024-01.parquet",
    split="train",
)
```

### DuckDB

```sql
-- All stories from 2024
SELECT id, by, title, score, url
FROM read_parquet('hf://datasets/open-index/hacker-news/data/2024/*.parquet')
WHERE type = 'story'
ORDER BY score DESC
LIMIT 20;

-- Live blocks for today
SELECT id, by, title, time
FROM read_parquet('hf://datasets/open-index/hacker-news/today/*.parquet')
ORDER BY id DESC
LIMIT 50;
```

### huggingface_hub

```python
from huggingface_hub import snapshot_download

folder = snapshot_download(
    "open-index/hacker-news",
    repo_type="dataset",
    local_dir="./hn/",
    allow_patterns="data/2024/*",
)
```

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `id` | int64 | Item ID (monotonically increasing) |
| `deleted` | bool | Soft-deleted flag |
| `type` | string | story, comment, ask, show, job, poll, pollopt |
| `by` | string | Username of author |
| `time` | DateTime | Post timestamp (UTC) |
| `text` | string | HTML body (comments, Ask HN, jobs) |
| `dead` | bool | Flagged/killed by moderators |
| `parent` | int64 | Parent item ID (for comments) |
| `poll` | int64 | Poll item ID (for pollopts) |
| `kids` | Array(int64) | Direct child item IDs |
| `url` | string | External URL (stories) |
| `score` | int64 | Points |
| `title` | string | Story/Ask/Show/Job title |
| `parts` | Array(int64) | Poll option IDs |
| `descendants` | int64 | Total descendant comment count |

## License

Released under the **Open Data Commons Attribution License (ODC-By) v1.0**.
Original content is subject to the rights of its respective authors.
Hacker News data is provided by Y Combinator.
```

- [ ] **Step 2: Create `pkg/hn2/readme.go`**

```go
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
	TotalItems    int64
	TotalSizeBytes int64
	TotalSizeMB   string
	TodaySizeKB   string
	LastUpdated   string
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
```

- [ ] **Step 3: Build**

```bash
go build ./pkg/hn2/...
```

- [ ] **Step 4: Commit**

```bash
git add cli/embed/hn_readme.md.tmpl pkg/hn2/readme.go
git commit -m "feat(hn2): add README template and generator"
```

---

### Task 5: pkg/hn2 Historical Task

**Files:**
- Create: `pkg/hn2/task_historical.go`

- [ ] **Step 1: Create `pkg/hn2/task_historical.go`**

```go
package hn2

import (
	"context"
	"fmt"
	"os"
	"time"
)

// HistoricalState is emitted by HistoricalTask during execution.
type HistoricalState struct {
	Phase        string // "fetch" | "commit" | "skip"
	Month        string // "2006-10"
	MonthIndex   int
	MonthTotal   int
	Rows         int64
	BytesDone    int64
	ElapsedTotal time.Duration
	SpeedBytesPS float64
}

// HistoricalMetric is the final result of HistoricalTask.
type HistoricalMetric struct {
	MonthsWritten int
	MonthsSkipped int
	RowsWritten   int64
	BytesWritten  int64
	Elapsed       time.Duration
}

// HistoricalTaskOptions configures the historical backfill.
type HistoricalTaskOptions struct {
	FromYear  int // skip months before this year (0 = no limit)
	FromMonth int // skip months before this month (0 = no limit)
	HFCommit  func(ctx context.Context, ops []HFOp, message string) (string, error)
	ReadmeTmpl []byte
}

// HFOp describes a single file operation for a Hugging Face commit.
type HFOp struct {
	LocalPath  string
	PathInRepo string
	Delete     bool
}

// HistoricalTask implements pkg/core.Task for backfilling all historical HN months.
type HistoricalTask struct {
	cfg  Config
	opts HistoricalTaskOptions
}

func NewHistoricalTask(cfg Config, opts HistoricalTaskOptions) *HistoricalTask {
	return &HistoricalTask{cfg: cfg, opts: opts}
}

func (t *HistoricalTask) Run(ctx context.Context, emit func(*HistoricalState)) (HistoricalMetric, error) {
	cfg := t.cfg.WithDefaults()
	started := time.Now()
	metric := HistoricalMetric{}

	// Load already-committed months from stats.csv.
	existingRows, err := ReadStatsCSV(cfg.StatsCSVPath())
	if err != nil {
		return metric, fmt.Errorf("read stats.csv: %w", err)
	}
	committed := CommittedMonthSet(existingRows)

	// Query remote for all available months (current month excluded by ListMonths).
	months, err := cfg.ListMonths(ctx)
	if err != nil {
		return metric, fmt.Errorf("list months: %w", err)
	}

	// Apply --from filter.
	var filtered []MonthInfo
	for _, m := range months {
		if t.opts.FromYear > 0 {
			if m.Year < t.opts.FromYear {
				continue
			}
			if m.Year == t.opts.FromYear && t.opts.FromMonth > 0 && m.Month < t.opts.FromMonth {
				continue
			}
		}
		filtered = append(filtered, m)
	}

	total := len(filtered)
	bytesDone := int64(0)

	for i, m := range filtered {
		if ctx.Err() != nil {
			return metric, ctx.Err()
		}
		monthStr := fmt.Sprintf("%04d-%02d", m.Year, m.Month)
		state := &HistoricalState{
			Month:        monthStr,
			MonthIndex:   i + 1,
			MonthTotal:   total,
			ElapsedTotal: time.Since(started),
		}

		// Skip already committed.
		if committed[[2]int{m.Year, m.Month}] {
			state.Phase = "skip"
			metric.MonthsSkipped++
			if emit != nil {
				emit(state)
			}
			continue
		}

		outPath := cfg.MonthPath(m.Year, m.Month)
		state.Phase = "fetch"
		if emit != nil {
			emit(state)
		}

		t0Fetch := time.Now()
		result, err := cfg.FetchMonth(ctx, m.Year, m.Month, outPath)
		if err != nil {
			return metric, fmt.Errorf("fetch %s: %w", monthStr, err)
		}
		durFetchS := int(time.Since(t0Fetch).Seconds())

		if result.Count == 0 {
			// Remove any orphaned .tmp file; skip this month.
			_ = os.Remove(outPath + ".tmp")
			state.Phase = "skip"
			metric.MonthsSkipped++
			if emit != nil {
				emit(state)
			}
			continue
		}

		state.Rows = result.Count
		state.BytesDone = bytesDone + result.Bytes
		state.Phase = "commit"
		if emit != nil {
			emit(state)
		}

		// Write stats.csv first so README includes the new month's numbers.
		existingRows, _ = ReadStatsCSV(cfg.StatsCSVPath())
		newRow := MonthRow{
			Year: m.Year, Month: m.Month,
			LowestID: result.LowestID, HighestID: result.HighestID,
			Count: result.Count, DurFetchS: durFetchS,
			SizeBytes: result.Bytes, CommittedAt: time.Now().UTC(),
		}
		if err := WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newRow, false); err != nil {
			return metric, fmt.Errorf("write stats.csv: %w", err)
		}

		// Regenerate README from updated stats.csv (now includes newRow).
		updatedRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
		todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		readmeBytes, _ := GenerateREADME(t.opts.ReadmeTmpl, updatedRows, todayRows)
		if readmeBytes != nil {
			_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
		}

		t0Commit := time.Now()
		ops := []HFOp{
			{LocalPath: outPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", m.Year, m.Year, m.Month)},
			{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}
		msg := fmt.Sprintf("Add %s (%s items)", monthStr, fmtInt(result.Count))
		if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
			return metric, fmt.Errorf("hf commit %s: %w", monthStr, err)
		}
		durCommitS := int(time.Since(t0Commit).Seconds())

		// Update commit duration in stats.csv.
		newRow.DurCommitS = durCommitS
		existingRows, _ = ReadStatsCSV(cfg.StatsCSVPath())
		_ = WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newRow, true)

		bytesDone += result.Bytes
		metric.MonthsWritten++
		metric.RowsWritten += result.Count
		metric.BytesWritten += result.Bytes
		committed[[2]int{m.Year, m.Month}] = true

		state.Rows = result.Count
		state.BytesDone = bytesDone
		elapsed := time.Since(started)
		state.ElapsedTotal = elapsed
		if elapsed.Seconds() > 0 {
			state.SpeedBytesPS = float64(bytesDone) / elapsed.Seconds()
		}
		if emit != nil {
			emit(state)
		}
	}

	metric.Elapsed = time.Since(started)
	return metric, nil
}

func fmtInt(n int64) string {
	// Simple comma-formatted integer.
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}
```

- [ ] **Step 2: Build**

```bash
go build ./pkg/hn2/...
```

- [ ] **Step 3: Commit**

```bash
git add pkg/hn2/task_historical.go
git commit -m "feat(hn2): add HistoricalTask for backfilling monthly parquet files"
```

---

### Task 6: pkg/hn2 Live Task and Day Rollover Task

**Files:**
- Create: `pkg/hn2/task_live.go`
- Create: `pkg/hn2/task_rollover.go`

- [ ] **Step 1: Create `pkg/hn2/task_rollover.go`**

```go
package hn2

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// RolloverState is emitted by DayRolloverTask.
type RolloverState struct {
	Phase      string // "merge" | "commit"
	PrevDate   string
	FilesFound int
	RowsMerged int64
}

// RolloverMetric is the final result of DayRolloverTask.
type RolloverMetric struct {
	PrevDate    string
	MonthPath   string
	RowsMerged  int64
	FilesPruned int
	CommitURL   string
}

// RolloverTaskOptions configures the day rollover.
type RolloverTaskOptions struct {
	PrevDate   string // YYYY-MM-DD
	HFCommit   func(ctx context.Context, ops []HFOp, message string) (string, error)
	ReadmeTmpl []byte
}

// DayRolloverTask merges today's live blocks into the monthly parquet and commits to HF.
type DayRolloverTask struct {
	cfg  Config
	opts RolloverTaskOptions
}

func NewDayRolloverTask(cfg Config, opts RolloverTaskOptions) *DayRolloverTask {
	return &DayRolloverTask{cfg: cfg, opts: opts}
}

func (t *DayRolloverTask) Run(ctx context.Context, emit func(*RolloverState)) (RolloverMetric, error) {
	cfg := t.cfg.WithDefaults()
	prevDate := t.opts.PrevDate
	metric := RolloverMetric{PrevDate: prevDate}

	// Determine year/month from prevDate.
	prevTime, err := time.Parse("2006-01-02", prevDate)
	if err != nil {
		return metric, fmt.Errorf("parse prev date: %w", err)
	}
	year, month := prevTime.Year(), int(prevTime.Month())
	monthPath := cfg.MonthPath(year, month)
	metric.MonthPath = monthPath

	// Collect today/ files for prevDate.
	pattern := filepath.Join(cfg.TodayDir(), prevDate+"_*.parquet")
	todayFiles, _ := filepath.Glob(pattern)

	state := &RolloverState{Phase: "merge", PrevDate: prevDate, FilesFound: len(todayFiles)}
	if emit != nil {
		emit(state)
	}

	// Re-entrant safety: if no today files but monthPath exists, skip merge.
	if len(todayFiles) == 0 {
		if fileExistsNE(monthPath) {
			// Already merged; re-run steps 4–7 (scan, upsert stats, commit).
			goto scanAndCommit
		}
		return metric, nil // nothing to do
	}

	{
		// Build parquet source list: existing monthly (if any) + today files.
		var sources []string
		if fileExistsNE(monthPath) {
			sources = append(sources, monthPath)
		}
		sources = append(sources, todayFiles...)

		tmpPath := monthPath + ".tmp"
		_ = os.Remove(tmpPath)
		if err := os.MkdirAll(filepath.Dir(monthPath), 0o755); err != nil {
			return metric, fmt.Errorf("create month dir: %w", err)
		}

		// Build DuckDB read_parquet list.
		listSQL := buildParquetList(sources)
		mergeQ := fmt.Sprintf(
			`COPY (SELECT * FROM read_parquet(%s) ORDER BY id) TO '%s' (FORMAT PARQUET, COMPRESSION zstd, COMPRESSION_LEVEL 22)`,
			listSQL, escapeSQLStr(tmpPath),
		)
		db, err := sql.Open("duckdb", "")
		if err != nil {
			return metric, fmt.Errorf("open duckdb for merge: %w", err)
		}
		_, mergeErr := db.ExecContext(ctx, mergeQ)
		db.Close()
		if mergeErr != nil {
			_ = os.Remove(tmpPath)
			return metric, fmt.Errorf("merge parquet: %w", mergeErr)
		}
		if err := os.Rename(tmpPath, monthPath); err != nil {
			_ = os.Remove(tmpPath)
			return metric, fmt.Errorf("rename merged parquet: %w", err)
		}
	}

scanAndCommit:
	// Scan merged file.
	db2, err := sql.Open("duckdb", "")
	if err != nil {
		return metric, fmt.Errorf("open duckdb for scan: %w", err)
	}
	var count, minID, maxID int64
	scanQ := fmt.Sprintf(`SELECT COUNT(*)::BIGINT, MIN(id)::BIGINT, MAX(id)::BIGINT FROM read_parquet('%s')`, escapeSQLStr(monthPath))
	_ = db2.QueryRowContext(ctx, scanQ).Scan(&count, &minID, &maxID)
	db2.Close()
	metric.RowsMerged = count

	state.RowsMerged = count
	state.Phase = "commit"
	if emit != nil {
		emit(state)
	}

	// Upsert stats.csv with the merged month row.
	fi, _ := os.Stat(monthPath)
	var sizeBytes int64
	if fi != nil {
		sizeBytes = fi.Size()
	}
	existingRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	newMonthRow := MonthRow{
		Year: year, Month: month,
		LowestID: minID, HighestID: maxID,
		Count: count, SizeBytes: sizeBytes,
		CommittedAt: time.Now().UTC(),
	}
	_ = WriteStatsCSV(cfg.StatsCSVPath(), existingRows, newMonthRow, true /* upsert */)

	// Clear stats_today.csv.
	_ = ClearStatsTodayCSV(cfg.StatsTodayCSVPath())

	// Regenerate README.
	updatedMonths, _ := ReadStatsCSV(cfg.StatsCSVPath())
	readmeBytes, _ := GenerateREADME(t.opts.ReadmeTmpl, updatedMonths, nil)
	if readmeBytes != nil {
		_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
	}

	// Build HF commit: delete today files + add monthly + metadata.
	var ops []HFOp
	for _, f := range todayFiles {
		base := filepath.Base(f)
		ops = append(ops, HFOp{PathInRepo: "today/" + base, Delete: true})
	}
	ops = append(ops,
		HFOp{LocalPath: monthPath, PathInRepo: fmt.Sprintf("data/%04d/%04d-%02d.parquet", year, year, month)},
		HFOp{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
		HFOp{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
		HFOp{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
	)
	msg := fmt.Sprintf("Merge %s → data/%04d/%04d-%02d.parquet (%s items)", prevDate, year, year, month, fmtInt(count))
	commitURL, err := t.opts.HFCommit(ctx, ops, msg)
	if err != nil {
		return metric, fmt.Errorf("hf rollover commit: %w", err)
	}
	metric.CommitURL = commitURL

	// Delete local today files after confirmed successful commit.
	for _, f := range todayFiles {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			// Log but don't fail — files are inert at this point.
			fmt.Fprintf(os.Stderr, "warn: remove local today file %s: %v\n", f, err)
		} else {
			metric.FilesPruned++
		}
	}
	return metric, nil
}

func buildParquetList(paths []string) string {
	if len(paths) == 1 {
		return "'" + escapeSQLStr(paths[0]) + "'"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, p := range paths {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("'")
		sb.WriteString(escapeSQLStr(p))
		sb.WriteString("'")
	}
	sb.WriteString("]")
	return sb.String()
}

func fileExistsNE(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir() && fi.Size() > 0
}
```

(imports already include `"strings"` in the block above)

- [ ] **Step 2: Create `pkg/hn2/task_live.go`**

```go
package hn2

import (
	"context"
	"fmt"
	"os"
	"time"
)

// LiveState is emitted by LiveTask on each poll cycle.
type LiveState struct {
	Phase          string // "fetch" | "commit" | "wait" | "rollover"
	Block          string // "2026-03-14 00:05"
	NewItems       int64
	HighestID      int64
	NextFetchIn    time.Duration
	BlocksToday    int
	TotalCommitted int64
}

// LiveMetric is the final result of LiveTask (only returned on context cancel).
type LiveMetric struct {
	BlocksWritten int
	RowsWritten   int64
	Rollovers     int
	Elapsed       time.Duration
}

// LiveTaskOptions configures the live polling task.
type LiveTaskOptions struct {
	Interval   time.Duration // poll interval, default 5m
	HFCommit   func(ctx context.Context, ops []HFOp, message string) (string, error)
	ReadmeTmpl []byte
}

// LiveTask implements pkg/core.Task for continuous 5-min live publishing.
type LiveTask struct {
	cfg  Config
	opts LiveTaskOptions
}

func NewLiveTask(cfg Config, opts LiveTaskOptions) *LiveTask {
	return &LiveTask{cfg: cfg, opts: opts}
}

func (t *LiveTask) Run(ctx context.Context, emit func(*LiveState)) (LiveMetric, error) {
	cfg := t.cfg.WithDefaults()
	interval := t.opts.Interval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	started := time.Now()
	metric := LiveMetric{}

	// --- Cold-start watermark ---
	today := time.Now().UTC().Format("2006-01-02")
	lastDate := today
	var lastHighestID int64

	todayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
	if maxID := MaxTodayHighestID(todayRows, today); maxID > 0 {
		lastHighestID = maxID
	} else {
		monthRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
		if maxID := MaxHighestID(monthRows); maxID > 0 {
			lastHighestID = maxID
		} else {
			info, err := cfg.RemoteInfo(ctx)
			if err != nil {
				return metric, fmt.Errorf("remote info for watermark: %w", err)
			}
			lastHighestID = info.MaxID
		}
	}

	blocksToday := len(todayRows)
	totalCommitted := int64(0)
	for _, r := range todayRows {
		totalCommitted += r.Count
	}

	for {
		if ctx.Err() != nil {
			metric.Elapsed = time.Since(started)
			return metric, nil
		}

		// Compute 5-min aligned block time.
		now := time.Now().UTC()
		blockTime := now.Truncate(interval)
		blockDate := blockTime.Format("2006-01-02")
		blockHHMM := blockTime.Format("15:04")

		// Check for day rollover.
		if blockDate != lastDate {
			state := &LiveState{Phase: "rollover", Block: lastDate}
			if emit != nil {
				emit(state)
			}
			rollover := NewDayRolloverTask(cfg, RolloverTaskOptions{
				PrevDate:   lastDate,
				HFCommit:   t.opts.HFCommit,
				ReadmeTmpl: t.opts.ReadmeTmpl,
			})
			if _, err := rollover.Run(ctx, nil); err != nil {
				fmt.Fprintf(os.Stderr, "warn: day rollover failed: %v\n", err)
			} else {
				metric.Rollovers++
				blocksToday = 0
				totalCommitted = 0
				todayRows = nil
			}
			lastDate = blockDate
		}

		outPath := cfg.TodayBlockPath(blockDate, blockHHMM)
		state := &LiveState{
			Phase:          "fetch",
			Block:          blockDate + " " + blockHHMM,
			HighestID:      lastHighestID,
			BlocksToday:    blocksToday,
			TotalCommitted: totalCommitted,
		}
		if emit != nil {
			emit(state)
		}

		ceilTime := now // bound query to avoid leaking tomorrow's items
		result, err := cfg.FetchSince(ctx, lastHighestID, ceilTime, outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: fetch since %d: %v\n", lastHighestID, err)
			sleepUntilNext(ctx, interval)
			continue
		}
		if result.Count == 0 {
			_ = os.Remove(outPath + ".tmp")
			sleepUntilNext(ctx, interval)
			continue
		}

		lastHighestID = result.HighestID
		blocksToday++
		totalCommitted += result.Count

		// Append row to stats_today.csv.
		// Build block filename with "_" instead of ":" for filesystem compatibility.
		blockFilename := blockDate + "_" + strings.ReplaceAll(blockHHMM, ":", "_") + ".parquet"
		blockPathInRepo := "today/" + blockFilename

		t0Commit := time.Now()
		todayRows, _ = ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		newTodayRow := TodayRow{
			Date: blockDate, Block: blockHHMM,
			LowestID: result.LowestID, HighestID: result.HighestID,
			Count: result.Count, DurFetchS: int(result.Duration.Seconds()),
			SizeBytes: result.Bytes, CommittedAt: time.Now().UTC(),
		}
		_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), todayRows, newTodayRow)

		// Regenerate README from both CSVs.
		monthRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
		allTodayRows, _ := ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
		readmeBytes, _ := GenerateREADME(t.opts.ReadmeTmpl, monthRows, allTodayRows)
		if readmeBytes != nil {
			_ = os.WriteFile(cfg.READMEPath(), readmeBytes, 0o644)
		}

		state.Phase = "commit"
		state.NewItems = result.Count
		if emit != nil {
			emit(state)
		}

		ops := []HFOp{
			{LocalPath: outPath, PathInRepo: blockPathInRepo},
			{LocalPath: cfg.StatsTodayCSVPath(), PathInRepo: "stats_today.csv"},
			{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		}

		msg := fmt.Sprintf("Live %s %s (+%s items)", blockDate, blockHHMM, fmtInt(result.Count))
		if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
			fmt.Fprintf(os.Stderr, "warn: hf commit block: %v\n", err)
		} else {
			durCommitS := int(time.Since(t0Commit).Seconds())
			newTodayRow.DurCommitS = durCommitS
			// Update the row with the actual commit duration (safe re-read and re-write).
			allTodayRows, _ = ReadStatsTodayCSV(cfg.StatsTodayCSVPath())
			filtered := make([]TodayRow, 0, len(allTodayRows))
			for _, r := range allTodayRows {
				if r.Date == blockDate && r.Block == blockHHMM {
					continue
				}
				filtered = append(filtered, r)
			}
			_ = WriteStatsTodayCSV(cfg.StatsTodayCSVPath(), filtered, newTodayRow)

			metric.BlocksWritten++
			metric.RowsWritten += result.Count
		}

		sleepUntilNext(ctx, interval)
	}
}

func sleepUntilNext(ctx context.Context, interval time.Duration) {
	now := time.Now().UTC()
	next := now.Truncate(interval).Add(interval)
	d := next.Sub(now)
	if d < time.Second {
		d = time.Second
	}
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
```

(imports already include `"strings"` in the block above)

- [ ] **Step 3: Build**

```bash
go build ./pkg/hn2/...
```

Fix any compile errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/hn2/task_live.go pkg/hn2/task_rollover.go
git commit -m "feat(hn2): add LiveTask and DayRolloverTask"
```

---

## Chunk 3: HF Client Update + CLI Command + Wire-up

### Task 7: Update hfOperation and hf_commit.py for Delete Support

**Files:**
- Modify: `cli/cc_publish_hf.go`
- Modify: `cli/embed/hf_commit.py`

- [ ] **Step 1: Add `Delete bool` to `hfOperation` and update `opJSON` in `cc_publish_hf.go`**

In `cli/cc_publish_hf.go`, change:
```go
// hfOperation is a file to include in a commit.
type hfOperation struct {
	LocalPath  string
	PathInRepo string
}
```
to:
```go
// hfOperation describes a file add or delete for a HuggingFace commit.
// Set Delete=true for CommitOperationDelete (LocalPath is ignored).
type hfOperation struct {
	LocalPath  string
	PathInRepo string
	Delete     bool
}
```

And change the `opJSON` struct in `createCommitPython`:
```go
type opJSON struct {
	LocalPath  string `json:"local_path,omitempty"`
	PathInRepo string `json:"path_in_repo"`
	Delete     bool   `json:"delete,omitempty"`
}
```

Update the ops mapping:
```go
out[i] = opJSON{LocalPath: op.LocalPath, PathInRepo: op.PathInRepo, Delete: op.Delete}
```

- [ ] **Step 2: Update `cli/embed/hf_commit.py` to support delete ops**

Replace the import line and operations loop:
```python
from huggingface_hub import HfApi, CommitOperationAdd, CommitOperationDelete
```

Replace the operations loop:
```python
    operations = []
    for op in ops_raw:
        if op.get("delete", False):
            operations.append(CommitOperationDelete(path_in_repo=op["path_in_repo"]))
            continue
        local = op["local_path"]
        repo_path = op["path_in_repo"]
        if not os.path.isfile(local):
            print(f"[hf_commit.py] WARNING: file not found: {local}", file=sys.stderr)
            continue
        operations.append(CommitOperationAdd(path_in_repo=repo_path, path_or_fileobj=local))
```

- [ ] **Step 3: Build**

```bash
go build ./cli/...
```

- [ ] **Step 4: Commit**

```bash
git add cli/cc_publish_hf.go cli/embed/hf_commit.py
git commit -m "feat(hf): add Delete support to hfOperation and hf_commit.py"
```

---

### Task 8: CLI Command — hn_publish.go

**Files:**
- Create: `cli/hn_publish.go`
- Modify: `cli/hn.go` (add `cmd.AddCommand(newHNPublish())`)

- [ ] **Step 1: Create `cli/hn_publish.go`**

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/hn2"
	"github.com/spf13/cobra"
)

//go:embed embed/hn_readme.md.tmpl
var hnReadmeTmpl []byte

func newHNPublish() *cobra.Command {
	var (
		repoRoot string
		repoID   string
		live     bool
		interval time.Duration
		fromStr  string
		private  bool
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish Hacker News dataset to Hugging Face",
		Long: `Publish the full Hacker News dataset to a Hugging Face dataset repo.

Historical mode (default): backfills all missing months from the first HN post
to last month. Already-committed months (tracked in stats.csv) are skipped —
safe to resume after interruption.

Live mode (--live): after historical backfill, polls every 5 minutes for new
items and commits them as today/YYYY-MM-DD_HH_MM.parquet blocks. At midnight,
today's blocks are merged into the monthly parquet and committed atomically.

Run both as separate screen sessions:
  screen -S hn-history    search hn publish
  screen -S hn-live       search hn publish --live`,
		Example: `  search hn publish
  search hn publish --live
  search hn publish --live --interval 5m
  search hn publish --from 2024-01`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHNPublish(cmd.Context(), repoRoot, repoID, live, interval, fromStr, private)
		},
	}

	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "Local root directory (default: $HOME/data/hn/repo)")
	cmd.Flags().StringVar(&repoID, "repo", "open-index/hacker-news", "Hugging Face dataset repo ID")
	cmd.Flags().BoolVar(&live, "live", false, "Enable continuous 5-min live polling after backfill")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Minute, "Live poll interval (minimum 1m)")
	cmd.Flags().StringVar(&fromStr, "from", "", "Start month YYYY-MM (skip older months in historical backfill)")
	cmd.Flags().BoolVar(&private, "private", false, "Create HF repo as private if it does not exist")
	return cmd
}

func runHNPublish(ctx context.Context, repoRoot, repoID string, live bool, interval time.Duration, fromStr string, private bool) error {
	token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
	if token == "" {
		return fmt.Errorf("HF_TOKEN environment variable is not set")
	}
	if interval < time.Minute {
		interval = time.Minute
	}

	cfg := hn2.Config{RepoRoot: repoRoot}
	cfg = cfg.WithDefaults()

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	hf := newHFClient(token)
	if err := hf.createDatasetRepo(ctx, repoID, private); err != nil {
		fmt.Printf("  note: create repo: %v\n", err)
	}

	// hfCommitFn bridges pkg/hn2.HFOp to cli.hfOperation.
	hfCommitFn := func(ctx context.Context, ops []hn2.HFOp, message string) (string, error) {
		var hfOps []hfOperation
		for _, op := range ops {
			hfOps = append(hfOps, hfOperation{
				LocalPath:  op.LocalPath,
				PathInRepo: op.PathInRepo,
				Delete:     op.Delete,
			})
		}
		return hf.createCommit(ctx, repoID, "main", message, hfOps)
	}

	// Parse --from flag.
	var fromYear, fromMonth int
	if fromStr != "" {
		t, err := time.Parse("2006-01", fromStr)
		if err != nil {
			return fmt.Errorf("--from: expected YYYY-MM, got %q", fromStr)
		}
		fromYear, fromMonth = t.Year(), int(t.Month())
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("HN Publish → " + repoID))
	fmt.Println()
	fmt.Printf("  Repo root  %s\n", labelStyle.Render(cfg.RepoRoot))
	fmt.Printf("  HF repo    %s\n", infoStyle.Render(repoID))
	if fromStr != "" {
		fmt.Printf("  From       %s\n", labelStyle.Render(fromStr))
	}
	if live {
		fmt.Printf("  Live mode  every %s\n", infoStyle.Render(interval.String()))
	}
	fmt.Println()

	// --- Historical backfill ---
	histTask := hn2.NewHistoricalTask(cfg, hn2.HistoricalTaskOptions{
		FromYear:   fromYear,
		FromMonth:  fromMonth,
		HFCommit:   hfCommitFn,
		ReadmeTmpl: hnReadmeTmpl,
	})

	metric, err := histTask.Run(ctx, func(s *hn2.HistoricalState) {
		switch s.Phase {
		case "skip":
			fmt.Printf("  [%s] %s\n", labelStyle.Render(s.Month), labelStyle.Render("skipped (already committed)"))
		case "fetch":
			fmt.Printf("  [%s] %s  [%d/%d]\n",
				labelStyle.Render(s.Month), infoStyle.Render("fetching…"), s.MonthIndex, s.MonthTotal)
		case "commit":
			fmt.Printf("  [%s] %s  %s rows\n",
				labelStyle.Render(s.Month), successStyle.Render("committing"), ccFmtInt64(s.Rows))
		}
	})
	if err != nil {
		return fmt.Errorf("historical backfill: %w", err)
	}

	fmt.Println()
	fmt.Printf("  Historical  %s months written, %s skipped\n",
		infoStyle.Render(fmt.Sprintf("%d", metric.MonthsWritten)),
		labelStyle.Render(fmt.Sprintf("%d", metric.MonthsSkipped)))
	fmt.Printf("  Rows        %s\n", infoStyle.Render(ccFmtInt64(metric.RowsWritten)))
	fmt.Printf("  Elapsed     %s\n", labelStyle.Render(metric.Elapsed.Round(time.Second).String()))
	fmt.Println()

	if !live {
		return nil
	}

	// --- Live mode ---
	fmt.Println(subtitleStyle.Render("Live mode — polling every " + interval.String()))
	fmt.Println()

	liveTask := hn2.NewLiveTask(cfg, hn2.LiveTaskOptions{
		Interval:   interval,
		HFCommit:   hfCommitFn,
		ReadmeTmpl: hnReadmeTmpl,
	})

	_, err = liveTask.Run(ctx, func(s *hn2.LiveState) {
		switch s.Phase {
		case "fetch":
			fmt.Printf("  [%s] fetching since id=%d…\n",
				labelStyle.Render(s.Block), s.HighestID)
		case "commit":
			fmt.Printf("  [%s] +%s items  committed\n",
				labelStyle.Render(s.Block), infoStyle.Render(ccFmtInt64(s.NewItems)))
		case "wait":
			fmt.Printf("  [%s] next fetch in %s\n",
				labelStyle.Render(s.Block), labelStyle.Render(s.NextFetchIn.Round(time.Second).String()))
		case "rollover":
			fmt.Printf("  %s day rollover for %s…\n",
				warningStyle.Render("↻"), labelStyle.Render(s.Block))
		}
	})
	return err
}

// Use ccFmtInt64 (already defined in cc.go) for all integer formatting.
```

Note: use `ccFmtInt64` (defined in `cli/cc.go`) for integer formatting — do not introduce new helpers. `dimStyle` does not exist in the cli package; `labelStyle` is the correct substitute (already applied in the code above).

- [ ] **Step 2: Wire `newHNPublish()` into `cli/hn.go`**

In `cli/hn.go`, after the existing `cmd.AddCommand(newHNExport())` line, add:
```go
cmd.AddCommand(newHNPublish())
```

- [ ] **Step 3: Build**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./...
```

Fix any compile errors (missing style variables, import issues, etc.).

- [ ] **Step 4: Smoke test — check help**

```bash
go run ./cmd/search/... hn publish --help
```

Expected output shows `Usage: publish`, flags `--repo`, `--live`, `--interval`, `--from`, `--private`.

- [ ] **Step 5: Commit**

```bash
git add cli/hn_publish.go cli/hn.go
git commit -m "feat(cli): add search hn publish command"
```

---

### Task 9: Build Binary and Quick Validation

- [ ] **Step 1: Build the binary**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
make install
```

Expected: binary written to `$HOME/bin/search` (or `$HOME/bin/mizu`).

- [ ] **Step 2: Verify command is registered**

```bash
search hn --help
```

Expected: `publish` appears in the list of subcommands.

- [ ] **Step 3: Verify flags and dry-run output**

```bash
search hn publish --help
```

Expected: shows all flags including `--live`, `--from`, `--interval`.

- [ ] **Step 4: Run a quick info sanity check (no HF_TOKEN needed)**

The remote info query should work without authentication:
```bash
# This tests the remote query path (exits immediately since no HF_TOKEN)
HF_TOKEN="" search hn publish 2>&1 | head -5
```

Expected: prints `HF_TOKEN environment variable is not set`.

- [ ] **Step 5: Full build commit**

```bash
git add -u
git commit -m "build: verify hn publish compiles and runs"
```

---

## Chunk 4: Deploy to Server 2

### Task 10: Deploy to Server 2

- [ ] **Step 1: Push current branch to remote**

```bash
git push origin workbase
```

- [ ] **Step 2: SSH to server 2 and pull**

```bash
ssh server2
cd /path/to/mizu/blueprints/search   # adjust to actual server path
git fetch origin
git checkout workbase
git pull
```

- [ ] **Step 3: Build binary on server 2**

```bash
make install
```

Expected: `$HOME/bin/search` updated.

- [ ] **Step 4: Set HF_TOKEN on server**

```bash
echo 'export HF_TOKEN=hf_...' >> ~/.bashrc   # or ~/.zshrc
source ~/.bashrc
```

- [ ] **Step 5: Start historical backfill screen session**

```bash
screen -dmS hn-history bash -c 'search hn publish 2>&1 | tee $HOME/logs/hn-history.log'
screen -r hn-history   # verify it started
# Ctrl+A D to detach
```

- [ ] **Step 6: Start live mode screen session**

```bash
screen -dmS hn-live bash -c 'search hn publish --live 2>&1 | tee $HOME/logs/hn-live.log'
screen -r hn-live   # verify it started
# Ctrl+A D to detach
```

- [ ] **Step 7: Verify both sessions are running without conflicts**

> **Safety note:** Both sessions share `$HOME/data/hn/repo/stats.csv`. This is safe because: HistoricalTask only writes stats.csv for *completed past months*; LiveTask's DayRolloverTask writes stats.csv once per midnight (upsert). The two writes never target the same `(year, month)` key simultaneously — historical always processes earlier months while live processes only today's/current-month data. Both writes use atomic `file.tmp → rename` so no partial-write corruption is possible. If you want extra safety, start live mode only after historical has committed all months up to last month.

```bash
screen -ls
# Expected: hn-history and hn-live both listed

tail -f $HOME/logs/hn-history.log &
tail -f $HOME/logs/hn-live.log &
```

Historical should show months being fetched and committed. Live should show "fetching since id=…" every 5 minutes.

- [ ] **Step 8: Verify HF repo**

Check `https://huggingface.co/datasets/open-index/hacker-news` to confirm:
- `data/2006/2006-10.parquet` (or whichever first month appears) is present
- `stats.csv` is present
- `README.md` shows correct item counts

---

## Notes for Implementer

### Style variables in cli package

Check `cli/banner.go` or `cli/styles.go` for available Lipgloss styles. The ones used in `hn_publish.go` are:
- `labelStyle` — cyan/dim label text
- `infoStyle` — bright info text
- `successStyle` — green success text
- `warningStyle` — yellow warning text
- `subtitleStyle` — subtitle heading
- `dimStyle` — if missing, replace with `labelStyle`

### Missing `fmt` import in config.go

`pkg/hn2/config.go` uses `fmt.Sprintf` in `MonthDir()` and `MonthPath()` — ensure `"fmt"` is in the import block.

### DuckDB import

Both `fetch.go` and `task_rollover.go` import `_ "github.com/duckdb/duckdb-go/v2"` as a blank import. This is required to register the DuckDB driver with `database/sql`. Confirm this import exists in `go.mod` (it does — used by `pkg/hn/export.go`).

### The `goto` in task_rollover.go

The `goto scanAndCommit` pattern requires that no new variable declarations appear between the `goto` and the label. If the Go compiler rejects this, refactor by extracting `scanAndCommit` logic into a helper function `runRolloverScanAndCommit(ctx, cfg, monthPath, todayFiles, ...)`.

### TodayBlockPath filename

`TodayBlockPath` produces `today/2026-03-14_00_05.parquet`. The `LiveTask` path construction should match exactly — double-check the `":" → "_"` replacement for the HH:MM block string.

### stats_today.csv path in TodayBlockPath

`TodayBlockPath` returns a path with `HH:MM` → `HH_MM` in the filename. The `PathInRepo` for the live block commit must also use `HH_MM` (underscores), not `HH:MM`. This is handled in `task_live.go` via `strings.ReplaceAll(blockHHMM, ":", "_")`.
