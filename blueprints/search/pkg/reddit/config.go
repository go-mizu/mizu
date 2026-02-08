// Package reddit handles downloading and processing Reddit archive data
// from the Pushshift torrent (comments + submissions, 2005-2025).
package reddit

import (
	"os"
	"path/filepath"
)

const (
	// InfoHash is the Academic Torrents info hash for the Reddit archive.
	InfoHash = "3d426c47c767d40f82c7ef0f47c3acacedd2bf44"
)

// Trackers for the Reddit archive torrent.
var Trackers = []string{
	"https://academictorrents.com/announce.php",
	"udp://tracker.coppersurfer.tk:6969",
	"udp://tracker.opentrackr.org:1337/announce",
}

// FileKind represents the type of Reddit data file.
type FileKind string

const (
	Comments    FileKind = "comments"
	Submissions FileKind = "submissions"
)

// DataFile represents a single Reddit archive file with all its paths.
type DataFile struct {
	Kind      FileKind // comments or submissions
	Name      string   // e.g. "RC_2005-12" or "RS_2005-06"
	YearMonth string   // e.g. "2005-12"
	ZstPath   string   // Full path to raw .zst file
	DBPath    string   // Full path to .duckdb file
	PQPath    string   // Full path to .parquet file
	Size      int64    // Size in bytes (from torrent metadata)
}

// DataDir returns the base reddit data directory.
func DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "reddit")
}

// RawDir returns the directory for raw downloaded .zst files.
func RawDir() string {
	return filepath.Join(DataDir(), "raw")
}

// DatabaseDir returns the directory for imported DuckDB files.
func DatabaseDir() string {
	return filepath.Join(DataDir(), "database")
}

// ParquetDir returns the directory for exported parquet files.
func ParquetDir() string {
	return filepath.Join(DataDir(), "parquet")
}

// RawPath returns the full path for a raw .zst file.
func RawPath(kind FileKind, name string) string {
	return filepath.Join(RawDir(), string(kind), name+".zst")
}

// DBPath returns the full path for a DuckDB file.
func DBPath(kind FileKind, name string) string {
	return filepath.Join(DatabaseDir(), string(kind), name+".duckdb")
}

// PQPath returns the full path for a parquet file.
func PQPath(kind FileKind, name string) string {
	return filepath.Join(ParquetDir(), string(kind), name+".parquet")
}

// ParseTorrentPath parses a torrent file path like "comments/RC_2005-12.zst"
// into a DataFile.
func ParseTorrentPath(path string, size int64) (DataFile, bool) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	var kind FileKind
	var prefix string
	switch dir {
	case "comments":
		kind = Comments
		prefix = "RC_"
	case "submissions":
		kind = Submissions
		prefix = "RS_"
	default:
		return DataFile{}, false
	}

	// Strip .zst extension
	name := base
	if len(name) > 4 && name[len(name)-4:] == ".zst" {
		name = name[:len(name)-4]
	}

	// Extract year-month
	yearMonth := ""
	if len(name) > len(prefix) {
		yearMonth = name[len(prefix):]
	}

	return DataFile{
		Kind:      kind,
		Name:      name,
		YearMonth: yearMonth,
		ZstPath:   RawPath(kind, name),
		DBPath:    DBPath(kind, name),
		PQPath:    PQPath(kind, name),
		Size:      size,
	}, true
}
