package cli

import (
	"os"
	"path/filepath"
)

// DefaultDataDir returns the default data directory path.
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "data"
	}
	return filepath.Join(home, "data", "blueprint", "finewiki")
}

// LangDir returns the directory for a specific language.
func LangDir(baseDir, lang string) string {
	return filepath.Join(baseDir, lang)
}

// ParquetPath returns the parquet file path for a language.
func ParquetPath(baseDir, lang string) string {
	return filepath.Join(baseDir, lang, "data.parquet")
}

// ParquetGlob returns the parquet glob pattern for a language.
// It checks for single data.parquet or sharded data-*.parquet files.
func ParquetGlob(baseDir, lang string) string {
	// Check for single file first
	single := ParquetPath(baseDir, lang)
	if _, err := os.Stat(single); err == nil {
		return single
	}

	// Check for sharded files
	shardedPattern := filepath.Join(baseDir, lang, "data-*.parquet")
	matches, _ := filepath.Glob(shardedPattern)
	if len(matches) > 0 {
		return shardedPattern
	}

	// Default to single file path
	return single
}

// DuckDBPath returns the DuckDB file path for a language.
func DuckDBPath(baseDir, lang string) string {
	return filepath.Join(baseDir, lang, "wiki.duckdb")
}
