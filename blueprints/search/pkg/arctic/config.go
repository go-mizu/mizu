//go:build !windows

package arctic

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type Config struct {
	RepoRoot   string
	HFRepo     string
	RawDir     string
	WorkDir    string
	MinFreeGB  int
	ChunkLines int
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	root := envOr("MIZU_ARCTIC_REPO_ROOT", filepath.Join(home, "data", "arctic", "repo"))
	raw  := envOr("MIZU_ARCTIC_RAW_DIR",   filepath.Join(home, "data", "arctic", "raw"))
	work := envOr("MIZU_ARCTIC_WORK_DIR",  filepath.Join(home, "data", "arctic", "work"))
	minFree := envIntOr("MIZU_ARCTIC_MIN_FREE_GB", 30)
	chunkLines := envIntOr("MIZU_ARCTIC_CHUNK_LINES", 2_000_000)
	return Config{
		RepoRoot:   root,
		HFRepo:     "open-index/arctic",
		RawDir:     raw,
		WorkDir:    work,
		MinFreeGB:  minFree,
		ChunkLines: chunkLines,
	}
}

func (c Config) WithDefaults() Config {
	def := DefaultConfig()
	if c.RepoRoot   == "" { c.RepoRoot   = def.RepoRoot }
	if c.HFRepo     == "" { c.HFRepo     = def.HFRepo }
	if c.RawDir     == "" { c.RawDir     = def.RawDir }
	if c.WorkDir    == "" { c.WorkDir    = def.WorkDir }
	if c.MinFreeGB  == 0  { c.MinFreeGB  = def.MinFreeGB }
	if c.ChunkLines == 0  { c.ChunkLines = def.ChunkLines }
	return c
}

func (c Config) StatsCSVPath() string { return filepath.Join(c.RepoRoot, "stats.csv") }
func (c Config) READMEPath() string   { return filepath.Join(c.RepoRoot, "README.md") }

// ZstPath returns the local path where the torrent client saves a .zst file.
// The bundle torrent has a root folder "reddit/", so files land at:
//   RawDir/reddit/comments/RC_YYYY-MM.zst
//   RawDir/reddit/submissions/RS_YYYY-MM.zst
func (c Config) ZstPath(prefix, ym string) string {
	subdir := "comments"
	if prefix == "RS" {
		subdir = "submissions"
	}
	return filepath.Join(c.RawDir, "reddit", subdir, fmt.Sprintf("%s_%s.zst", prefix, ym))
}

// ChunkPath: temp JSONL chunk file
func (c Config) ChunkPath(n int) string {
	return filepath.Join(c.WorkDir, fmt.Sprintf("chunk_%04d.jsonl", n))
}

// ShardLocalDir: WorkDir/comments/2025/03
func (c Config) ShardLocalDir(typ, year, mm string) string {
	return filepath.Join(c.WorkDir, typ, year, mm)
}

// ShardLocalPath: WorkDir/comments/2025/03/000.parquet
func (c Config) ShardLocalPath(typ, year, mm string, n int) string {
	return filepath.Join(c.ShardLocalDir(typ, year, mm), fmt.Sprintf("%03d.parquet", n))
}

// ShardHFPath: "data/comments/2025/03/000.parquet"
func (c Config) ShardHFPath(typ, year, mm string, n int) string {
	return fmt.Sprintf("data/%s/%s/%s/%03d.parquet", typ, year, mm, n)
}

func (c Config) EnsureDirs() error {
	for _, dir := range []string{c.RepoRoot, c.RawDir, c.WorkDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return nil
}

// FreeDiskGB returns free disk GB for the partition containing WorkDir.
func (c Config) FreeDiskGB() (float64, error) {
	var st syscall.Statfs_t
	dir := c.WorkDir
	if dir == "" {
		dir = c.RawDir
	}
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, fmt.Errorf("statfs %s: %w", dir, err)
	}
	freeBytes := st.Bavail * uint64(st.Bsize)
	return float64(freeBytes) / (1024 * 1024 * 1024), nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}
