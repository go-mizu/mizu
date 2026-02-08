package reddit

import (
	"os"
	"path/filepath"
)

const ArcticBaseURL = "https://arctic-shift.photon-reddit.com"

// ArcticDir returns the base directory for Arctic Shift downloads.
func ArcticDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "reddit", "arctic")
}

// SubredditDir returns the data directory for a subreddit.
func SubredditDir(name string) string {
	return filepath.Join(ArcticDir(), "subreddit", name)
}

// UserDir returns the data directory for a user.
func UserDir(name string) string {
	return filepath.Join(ArcticDir(), "user", name)
}

// ArcticTarget identifies what to download.
type ArcticTarget struct {
	Kind string // "subreddit" or "user"
	Name string // subreddit name or username (without prefix)
}

// Dir returns the data directory for this target.
func (t ArcticTarget) Dir() string {
	if t.Kind == "user" {
		return UserDir(t.Name)
	}
	return SubredditDir(t.Name)
}

// JSONLPath returns the path to the JSONL file for a given file kind.
func (t ArcticTarget) JSONLPath(kind FileKind) string {
	return filepath.Join(t.Dir(), string(kind)+".jsonl")
}

// DBPath returns the path to the DuckDB database.
func (t ArcticTarget) DBPath() string {
	return filepath.Join(t.Dir(), "data.duckdb")
}

// ParquetPath returns the path to the parquet file for a given file kind.
func (t ArcticTarget) ParquetPath(kind FileKind) string {
	return filepath.Join(t.Dir(), string(kind)+".parquet")
}

// PartitionDir returns the directory for partitioned JSONL files for a kind.
func (t ArcticTarget) PartitionDir(kind FileKind) string {
	return filepath.Join(t.Dir(), string(kind))
}

// ProgressPath returns the path to the progress file.
func (t ArcticTarget) ProgressPath() string {
	return filepath.Join(t.Dir(), ".progress")
}
