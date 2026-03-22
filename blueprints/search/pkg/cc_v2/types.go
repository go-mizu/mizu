// Package cc_v2 implements a simplified Common Crawl pipeline for downloading,
// converting, and publishing WARC files as parquet datasets to HuggingFace.
//
// Architecture: three cooperating processes (pipeline, watcher, scheduler) with
// Redis as the single source of truth and file-based locking as fallback.
package cc_v2

import "time"

// ShardState represents the lifecycle state of a shard.
type ShardState string

const (
	ShardNone      ShardState = ""          // not in store = not started
	ShardClaimed   ShardState = "claimed"   // pipeline is downloading/packing
	ShardReady     ShardState = "ready"     // parquet written, waiting for watcher
	ShardCommitted ShardState = "committed" // pushed to HuggingFace
)

// ShardStats holds per-shard pipeline statistics.
type ShardStats struct {
	Rows      int64 `json:"rows"`
	HTMLBytes int64 `json:"html_bytes"`
	MDBytes   int64 `json:"md_bytes"`
	PqBytes   int64 `json:"pq_bytes"`
	DurDlS    int64 `json:"dur_dl_s"`
	DurPackS  int64 `json:"dur_pack_s"`
	DurPushS  int64 `json:"dur_push_s"`
	PeakRSSMB int64 `json:"peak_rss_mb"`
}

// ShardInfo is the full state of a shard from the store.
type ShardInfo struct {
	State       ShardState `json:"state"`
	ClaimedBy   string     `json:"claimed_by,omitempty"`
	ClaimedAt   time.Time  `json:"claimed_at,omitempty"`
	ReadyAt     time.Time  `json:"ready_at,omitempty"`
	CommittedAt time.Time  `json:"committed_at,omitempty"`
	WARCPath    string     `json:"warc_path,omitempty"`
	PqPath      string     `json:"pq_path,omitempty"`
	Stats       ShardStats `json:"stats"`
}

// WatcherStatus is the latest watcher commit information.
type WatcherStatus struct {
	CommitNum      int       `json:"commit_num"`
	Message        string    `json:"message"`
	CommitURL      string    `json:"commit_url"`
	ShardsInCommit int       `json:"shards_in_commit"`
	TotalCommitted int       `json:"total_committed"`
	Timestamp      time.Time `json:"timestamp"`
}

// Config holds configuration for all cc_v2 components.
type Config struct {
	CrawlID  string
	RepoID   string // HuggingFace repo ID (e.g., "open-index/open-markdown")
	DataDir  string // $HOME/data/cc_v2/{crawl}
	RepoRoot string // $HOME/data/cc_v2/{crawl}/repo
	Private  bool   // create HF repo as private
}

// DefaultDataDir returns the default data directory for a crawl.
func DefaultDataDir(crawlID string) string {
	return homeDir() + "/data/cc_v2/" + crawlID
}

// DefaultRepoRoot returns the default repo root for a crawl.
func DefaultRepoRoot(crawlID string) string {
	return DefaultDataDir(crawlID) + "/repo"
}

// PipelineConfig holds configuration for the pipeline worker.
type PipelineConfig struct {
	Config
	SkipErrors bool
	Indices    []int
}

// WatcherConfig holds configuration for the watcher.
type WatcherConfig struct {
	Config
	PollInterval    time.Duration // default 10s
	CommitInterval  time.Duration // min gap between HF commits, default 90s
	ChartsEvery     time.Duration // regenerate charts, default 60min
	MaxBatch        int           // max parquets per commit, default 30
}

// SchedulerConfig holds configuration for the scheduler.
type SchedulerConfig struct {
	Config
	Start       int
	End         int
	MaxSessions int     // 0 = auto-detect
	ChunkSize   int     // shards per screen session, default 50
	DonePct     int     // % committed = chunk done, default 95
	StallRounds int     // rounds until kill, default 40
	GapIndices  []int   // non-nil = gap backfill mode
}

// ParquetFile represents a parquet file found on disk.
type ParquetFile struct {
	Idx      int
	Shard    string // "00042"
	Path     string // absolute path
	MetaPath string // .meta.json path
	Size     int64
}
