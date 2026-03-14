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

// Config controls the remote data source and local repository root for HN publishing.
// The zero value is not useful; use DefaultConfig or set at least RepoRoot.
type Config struct {
	RepoRoot    string
	EndpointURL string
	Database    string
	Table       string
	User        string
	DNSServer   string
	HTTPClient  *http.Client
}

// DefaultConfig returns a Config populated with production defaults and
// any overrides from MIZU_HN2_* environment variables.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	c := Config{
		RepoRoot:    filepath.Join(home, "data", "hn", "repo"),
		EndpointURL: defaultEndpointURL,
		User:        defaultUser,
		Database:    defaultDatabase,
		Table:       defaultTable,
		HTTPClient:  &http.Client{Timeout: 60 * time.Second},
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_ENDPOINT")); v != "" {
		c.EndpointURL = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_USER")); v != "" {
		c.User = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_DATABASE")); v != "" {
		c.Database = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_TABLE")); v != "" {
		c.Table = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_DNS_SERVER")); v != "" {
		c.DNSServer = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN2_REPO_ROOT")); v != "" {
		c.RepoRoot = v
	}
	return c
}

// WithDefaults returns a copy of c with any zero fields filled from DefaultConfig.
// This is the public entry point used by CLI callers that construct a partial Config.
func (c Config) WithDefaults() Config {
	return c.resolved()
}

// resolved returns a Config with all fields filled. It is the single internal
// resolver; every method that needs a complete Config calls c.resolved().
func (c Config) resolved() Config {
	d := DefaultConfig()
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

// Path helpers — all return absolute paths derived from RepoRoot.

func (c Config) DataDir() string        { return filepath.Join(c.resolved().RepoRoot, "data") }
func (c Config) TodayDir() string       { return filepath.Join(c.resolved().RepoRoot, "today") }
func (c Config) StatsCSVPath() string   { return filepath.Join(c.resolved().RepoRoot, "stats.csv") }
func (c Config) StatsTodayCSVPath() string {
	return filepath.Join(c.resolved().RepoRoot, "stats_today.csv")
}
func (c Config) READMEPath() string { return filepath.Join(c.resolved().RepoRoot, "README.md") }

func (c Config) MonthDir(year int) string {
	return filepath.Join(c.DataDir(), fmt.Sprintf("%04d", year))
}

func (c Config) MonthPath(year, month int) string {
	return filepath.Join(c.MonthDir(year), fmt.Sprintf("%04d-%02d.parquet", year, month))
}

// TodayBlockPath returns the local absolute path for a live 5-min block parquet file.
// Layout: {RepoRoot}/today/YYYY/MM/DD/HH/MM.parquet
// date = "2026-03-14", hhmm = "15:04" (colon-separated)
func (c Config) TodayBlockPath(date, hhmm string) string {
	parts := strings.SplitN(date, "-", 3) // ["2026", "03", "14"]
	hh, mm := hhmm[:2], hhmm[3:5]
	return filepath.Join(c.TodayDir(), parts[0], parts[1], parts[2], hh, mm+".parquet")
}

// TodayHFPath returns the Hugging Face repo-relative path for a live block.
// e.g. "today/2026/03/14/15/04.parquet"
func (c Config) TodayHFPath(date, hhmm string) string {
	parts := strings.SplitN(date, "-", 3)
	hh, mm := hhmm[:2], hhmm[3:5]
	return "today/" + parts[0] + "/" + parts[1] + "/" + parts[2] + "/" + hh + "/" + mm + ".parquet"
}

// EnsureDirs creates the data and today directories if they do not exist.
// Per-year subdirectories are created on demand by FetchMonth.
func (c Config) EnsureDirs() error {
	cfg := c.resolved()
	for _, d := range []string{cfg.DataDir(), cfg.TodayDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// httpClient returns the configured HTTP client, falling back to a 60-second default.
func (c Config) httpClient() *http.Client {
	cfg := c.resolved()
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// fqTable returns the fully-qualified ClickHouse table name (e.g. `hackernews`.`hackernews`).
func (c Config) fqTable() string {
	cfg := c.resolved()
	return quoteIdent(cfg.Database) + "." + quoteIdent(cfg.Table)
}

func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}
