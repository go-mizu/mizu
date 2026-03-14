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
