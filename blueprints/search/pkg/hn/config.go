package hn

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// ClickHouse-hosted historical snapshot parquet (37M rows through 2023-08-18).
	// Kept as a fallback static snapshot source.
	DefaultParquetURL = "https://datasets-documentation.s3.eu-west-3.amazonaws.com/hackernews/2023-08-18.parquet"
	DefaultAPIBaseURL = "https://hacker-news.firebaseio.com/v0"

	DefaultClickHouseBaseURL  = "https://sql-clickhouse.clickhouse.com"
	DefaultClickHouseUser     = "demo"
	DefaultClickHouseDatabase = "hackernews"
	DefaultClickHouseTable    = "hackernews"
)

// Config controls paths and endpoints for Hacker News data ingestion.
type Config struct {
	DataDir    string
	ParquetURL string // static snapshot parquet fallback
	APIBaseURL string

	ClickHouseBaseURL   string
	ClickHouseUser      string
	ClickHouseDatabase  string
	ClickHouseTable     string
	ClickHouseDNSServer string

	HTTPClient *http.Client
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DataDir:            filepath.Join(home, "data", "hn"),
		ParquetURL:         DefaultParquetURL,
		APIBaseURL:         DefaultAPIBaseURL,
		ClickHouseBaseURL:  DefaultClickHouseBaseURL,
		ClickHouseUser:     DefaultClickHouseUser,
		ClickHouseDatabase: DefaultClickHouseDatabase,
		ClickHouseTable:    DefaultClickHouseTable,
		HTTPClient:         &http.Client{Timeout: 60 * time.Second},
	}
}

func (c Config) WithDefaults() Config {
	d := DefaultConfig()
	if v := strings.TrimSpace(os.Getenv("MIZU_HN_DATA_DIR")); v != "" {
		d.DataDir = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN_PARQUET_URL")); v != "" {
		d.ParquetURL = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN_API_BASE_URL")); v != "" {
		d.APIBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN_CLICKHOUSE_BASE_URL")); v != "" {
		d.ClickHouseBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN_CLICKHOUSE_USER")); v != "" {
		d.ClickHouseUser = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN_CLICKHOUSE_DATABASE")); v != "" {
		d.ClickHouseDatabase = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN_CLICKHOUSE_TABLE")); v != "" {
		d.ClickHouseTable = v
	}
	if v := strings.TrimSpace(os.Getenv("MIZU_HN_CLICKHOUSE_DNS_SERVER")); v != "" {
		d.ClickHouseDNSServer = v
	}

	if strings.TrimSpace(c.DataDir) != "" {
		d.DataDir = c.DataDir
	}
	if strings.TrimSpace(c.ParquetURL) != "" {
		d.ParquetURL = c.ParquetURL
	}
	if strings.TrimSpace(c.APIBaseURL) != "" {
		d.APIBaseURL = c.APIBaseURL
	}
	if strings.TrimSpace(c.ClickHouseBaseURL) != "" {
		d.ClickHouseBaseURL = c.ClickHouseBaseURL
	}
	if strings.TrimSpace(c.ClickHouseUser) != "" {
		d.ClickHouseUser = c.ClickHouseUser
	}
	if strings.TrimSpace(c.ClickHouseDatabase) != "" {
		d.ClickHouseDatabase = c.ClickHouseDatabase
	}
	if strings.TrimSpace(c.ClickHouseTable) != "" {
		d.ClickHouseTable = c.ClickHouseTable
	}
	if strings.TrimSpace(c.ClickHouseDNSServer) != "" {
		d.ClickHouseDNSServer = c.ClickHouseDNSServer
	}
	if c.HTTPClient != nil {
		d.HTTPClient = c.HTTPClient
	}
	return d
}

func (c Config) httpClient() *http.Client {
	cfg := c.WithDefaults()
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c Config) BaseDir() string {
	return c.WithDefaults().DataDir
}

func (c Config) RawDir() string {
	return filepath.Join(c.BaseDir(), "raw")
}

func (c Config) RawParquetPath() string {
	return filepath.Join(c.RawDir(), "items.parquet")
}

func (c Config) ClickHouseParquetDir() string {
	return filepath.Join(c.RawDir(), "clickhouse")
}

func (c Config) ClickHouseDeltaParquetDir() string {
	return filepath.Join(c.RawDir(), "clickhouse_delta")
}

func (c Config) APIDir() string {
	return filepath.Join(c.RawDir(), "api")
}

func (c Config) APIChunksDir() string {
	return filepath.Join(c.APIDir(), "chunks")
}

func (c Config) StateDir() string {
	return filepath.Join(c.BaseDir(), "state")
}

func (c Config) ParquetHeadCachePath() string {
	return filepath.Join(c.StateDir(), "parquet_head.json")
}

func (c Config) DownloadStatePath() string {
	return filepath.Join(c.StateDir(), "download_state.json")
}

func (c Config) ImportStatePath() string {
	return filepath.Join(c.StateDir(), "import_state.json")
}

func (c Config) DefaultDBPath() string {
	return filepath.Join(c.BaseDir(), "hn.duckdb")
}

func (c Config) ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (c Config) EnsureRawDirs() error {
	if err := c.ensureDir(c.RawDir()); err != nil {
		return err
	}
	if err := c.ensureDir(c.APIChunksDir()); err != nil {
		return err
	}
	if err := c.ensureDir(c.ClickHouseParquetDir()); err != nil {
		return err
	}
	if err := c.ensureDir(c.ClickHouseDeltaParquetDir()); err != nil {
		return err
	}
	return c.ensureDir(c.StateDir())
}
