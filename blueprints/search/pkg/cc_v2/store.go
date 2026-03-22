package cc_v2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store is the state backend for cc_v2.
type Store interface {
	// Shard lifecycle
	IsCommitted(ctx context.Context, idx int) bool
	IsReady(ctx context.Context, idx int) bool
	Claim(ctx context.Context, idx int) bool   // SETNX with 30-min TTL
	Release(ctx context.Context, idx int)       // DEL lock
	MarkReady(ctx context.Context, idx int, pqPath, warcPath string, stats *ShardStats)
	MarkCommitted(ctx context.Context, idx int)

	// Queries
	CommittedCount(ctx context.Context) int
	CommittedSet(ctx context.Context) map[int]bool
	GetWARCPath(ctx context.Context, idx int) string
	GetShardState(ctx context.Context, idx int) ShardState

	// Telemetry
	RecordEvent(ctx context.Context, event string) // downloaded|packed|committed
	EventRate(ctx context.Context, event string, window time.Duration) float64
	TrimRates(ctx context.Context)

	// Watcher status
	SetWatcherStatus(ctx context.Context, status WatcherStatus)
	GetWatcherStatus(ctx context.Context) (WatcherStatus, bool)

	// Logging
	Log(ctx context.Context, source, level, msg string)

	// Lifecycle
	Available() bool
	Close() error
}

// ── Redis Store ─────────────────────────────────────────────────────────────

const lockTTL = 30 * time.Minute

// RedisStore implements Store using Redis.
type RedisStore struct {
	rdb   *redis.Client
	crawl string
	host  string // hostname:pid for lock ownership
}

// NewRedisStore creates a Redis-backed store. Returns nil if REDIS_PASSWORD is unset.
func NewRedisStore(crawlID string) *RedisStore {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	if password == "" {
		return nil
	}
	hostname, _ := os.Hostname()
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})
	return &RedisStore{
		rdb:   rdb,
		crawl: crawlID,
		host:  fmt.Sprintf("%s:%d", hostname, os.Getpid()),
	}
}

func (s *RedisStore) key(parts ...string) string {
	k := "cc:v2:" + s.crawl
	for _, p := range parts {
		k += ":" + p
	}
	return k
}

func (s *RedisStore) Available() bool {
	if s == nil || s.rdb == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.rdb.Ping(ctx).Err() == nil
}

func (s *RedisStore) Close() error {
	if s == nil || s.rdb == nil {
		return nil
	}
	return s.rdb.Close()
}

// ── Shard Lifecycle ────────────────────────────────────────────────────────

func (s *RedisStore) IsCommitted(ctx context.Context, idx int) bool {
	ok, err := s.rdb.SIsMember(ctx, s.key("committed"), idx).Result()
	return err == nil && ok
}

func (s *RedisStore) IsReady(ctx context.Context, idx int) bool {
	val, err := s.rdb.HGet(ctx, s.key("shard", strconv.Itoa(idx)), "state").Result()
	return err == nil && val == string(ShardReady)
}

func (s *RedisStore) GetShardState(ctx context.Context, idx int) ShardState {
	if s.IsCommitted(ctx, idx) {
		return ShardCommitted
	}
	val, err := s.rdb.HGet(ctx, s.key("shard", strconv.Itoa(idx)), "state").Result()
	if err != nil {
		return ShardNone
	}
	return ShardState(val)
}

func (s *RedisStore) Claim(ctx context.Context, idx int) bool {
	lockKey := s.key("lock", strconv.Itoa(idx))
	ok, err := s.rdb.SetNX(ctx, lockKey, s.host, lockTTL).Result()
	if err != nil || !ok {
		return false
	}
	s.rdb.HSet(ctx, s.key("shard", strconv.Itoa(idx)), map[string]interface{}{
		"state":      string(ShardClaimed),
		"claimed_by": s.host,
		"claimed_at": time.Now().UTC().Format(time.RFC3339),
	})
	return true
}

func (s *RedisStore) Release(ctx context.Context, idx int) {
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, s.key("lock", strconv.Itoa(idx)))
	pipe.Del(ctx, s.key("shard", strconv.Itoa(idx)))
	pipe.Exec(ctx)
}

func (s *RedisStore) MarkReady(ctx context.Context, idx int, pqPath, warcPath string, stats *ShardStats) {
	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, s.key("shard", strconv.Itoa(idx)), map[string]interface{}{
		"state":       string(ShardReady),
		"ready_at":    time.Now().UTC().Format(time.RFC3339),
		"pq_path":     pqPath,
		"warc_path":   warcPath,
		"rows":        stats.Rows,
		"html_bytes":  stats.HTMLBytes,
		"md_bytes":    stats.MDBytes,
		"pq_bytes":    stats.PqBytes,
		"dur_dl_s":    stats.DurDlS,
		"dur_pack_s":  stats.DurPackS,
		"peak_rss_mb": stats.PeakRSSMB,
	})
	// Release the lock — pipeline is done with this shard
	pipe.Del(ctx, s.key("lock", strconv.Itoa(idx)))
	pipe.Exec(ctx)
}

func (s *RedisStore) MarkCommitted(ctx context.Context, idx int) {
	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, s.key("shard", strconv.Itoa(idx)),
		"state", string(ShardCommitted),
		"committed_at", time.Now().UTC().Format(time.RFC3339))
	pipe.SAdd(ctx, s.key("committed"), idx)
	pipe.Exec(ctx)
}

// ── Queries ───────────────────────────────────────────────────────────────

func (s *RedisStore) CommittedCount(ctx context.Context) int {
	n, err := s.rdb.SCard(ctx, s.key("committed")).Result()
	if err != nil {
		return 0
	}
	return int(n)
}

func (s *RedisStore) CommittedSet(ctx context.Context) map[int]bool {
	members, err := s.rdb.SMembers(ctx, s.key("committed")).Result()
	if err != nil {
		return nil
	}
	set := make(map[int]bool, len(members))
	for _, m := range members {
		if idx, err := strconv.Atoi(m); err == nil {
			set[idx] = true
		}
	}
	return set
}

func (s *RedisStore) GetWARCPath(ctx context.Context, idx int) string {
	val, err := s.rdb.HGet(ctx, s.key("shard", strconv.Itoa(idx)), "warc_path").Result()
	if err != nil {
		return ""
	}
	return val
}

// ── Telemetry ─────────────────────────────────────────────────────────────

func (s *RedisStore) RecordEvent(ctx context.Context, event string) {
	now := float64(time.Now().UnixMilli()) / 1000.0
	counter, err := s.rdb.Incr(ctx, s.key("counter", event)).Result()
	if err != nil {
		return
	}
	s.rdb.ZAdd(ctx, s.key("rate", event), redis.Z{
		Score:  now,
		Member: fmt.Sprintf("%d", counter),
	})
}

func (s *RedisStore) EventRate(ctx context.Context, event string, window time.Duration) float64 {
	now := time.Now()
	minScore := fmt.Sprintf("%f", float64(now.Add(-window).UnixMilli())/1000.0)
	maxScore := fmt.Sprintf("%f", float64(now.UnixMilli())/1000.0)
	count, err := s.rdb.ZCount(ctx, s.key("rate", event), minScore, maxScore).Result()
	if err != nil || count == 0 {
		return 0
	}
	return float64(count) / window.Hours()
}

func (s *RedisStore) TrimRates(ctx context.Context) {
	cutoff := fmt.Sprintf("%f", float64(time.Now().Add(-time.Hour).UnixMilli())/1000.0)
	for _, event := range []string{"downloaded", "packed", "committed"} {
		s.rdb.ZRemRangeByScore(ctx, s.key("rate", event), "-inf", cutoff)
	}
}

// ── Watcher Status ────────────────────────────────────────────────────────

func (s *RedisStore) SetWatcherStatus(ctx context.Context, status WatcherStatus) {
	s.rdb.HSet(ctx, s.key("watcher"), map[string]interface{}{
		"commit_num":       status.CommitNum,
		"message":          status.Message,
		"commit_url":       status.CommitURL,
		"shards_in_commit": status.ShardsInCommit,
		"total_committed":  status.TotalCommitted,
		"timestamp":        status.Timestamp.UTC().Format(time.RFC3339),
	})
}

func (s *RedisStore) GetWatcherStatus(ctx context.Context) (WatcherStatus, bool) {
	vals, err := s.rdb.HGetAll(ctx, s.key("watcher")).Result()
	if err != nil || len(vals) == 0 {
		return WatcherStatus{}, false
	}
	cn, _ := strconv.Atoi(vals["commit_num"])
	sc, _ := strconv.Atoi(vals["shards_in_commit"])
	tc, _ := strconv.Atoi(vals["total_committed"])
	ts, _ := time.Parse(time.RFC3339, vals["timestamp"])
	return WatcherStatus{
		CommitNum:      cn,
		Message:        vals["message"],
		CommitURL:      vals["commit_url"],
		ShardsInCommit: sc,
		TotalCommitted: tc,
		Timestamp:      ts,
	}, true
}

// ── Logging ───────────────────────────────────────────────────────────────

func (s *RedisStore) Log(ctx context.Context, source, level, msg string) {
	s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: s.key("log"),
		MaxLen: 2000,
		Approx: true,
		Values: map[string]interface{}{
			"source":  source,
			"level":   level,
			"message": msg,
			"time":    time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ── File-Based Store (fallback) ───────────────────────────────────────────

// FileStore implements Store using the filesystem for when Redis is unavailable.
type FileStore struct {
	dataDir string
	crawl   string
	host    string
}

// NewFileStore creates a file-based store.
func NewFileStore(dataDir, crawlID string) *FileStore {
	hostname, _ := os.Hostname()
	return &FileStore{
		dataDir: dataDir,
		crawl:   crawlID,
		host:    fmt.Sprintf("%s:%d", hostname, os.Getpid()),
	}
}

func (s *FileStore) lockDir() string {
	return filepath.Join(s.dataDir, "locks")
}

func (s *FileStore) lockPath(idx int) string {
	return filepath.Join(s.lockDir(), fmt.Sprintf("%05d.lock", idx))
}

func (s *FileStore) committedDir() string {
	return filepath.Join(s.dataDir, "committed")
}

func (s *FileStore) committedPath(idx int) string {
	return filepath.Join(s.committedDir(), fmt.Sprintf("%05d", idx))
}

func (s *FileStore) Available() bool { return true }
func (s *FileStore) Close() error    { return nil }

func (s *FileStore) IsCommitted(_ context.Context, idx int) bool {
	_, err := os.Stat(s.committedPath(idx))
	return err == nil
}

func (s *FileStore) IsReady(_ context.Context, idx int) bool {
	pqPath := filepath.Join(s.dataDir, "parquet", fmt.Sprintf("%05d.parquet", idx))
	_, err := os.Stat(pqPath)
	return err == nil
}

func (s *FileStore) GetShardState(ctx context.Context, idx int) ShardState {
	if s.IsCommitted(ctx, idx) {
		return ShardCommitted
	}
	if s.IsReady(ctx, idx) {
		return ShardReady
	}
	if _, err := os.Stat(s.lockPath(idx)); err == nil {
		return ShardClaimed
	}
	return ShardNone
}

func (s *FileStore) Claim(_ context.Context, idx int) bool {
	_ = os.MkdirAll(s.lockDir(), 0o755)
	lp := s.lockPath(idx)
	f, err := os.OpenFile(lp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		// Check if stale (>30 min old)
		if fi, serr := os.Stat(lp); serr == nil && time.Since(fi.ModTime()) > lockTTL {
			os.Remove(lp)
			f2, err2 := os.OpenFile(lp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
			if err2 != nil {
				return false
			}
			fmt.Fprintf(f2, "%s\n", s.host)
			f2.Close()
			return true
		}
		return false
	}
	fmt.Fprintf(f, "%s\n", s.host)
	f.Close()
	return true
}

func (s *FileStore) Release(_ context.Context, idx int) {
	os.Remove(s.lockPath(idx))
}

func (s *FileStore) MarkReady(_ context.Context, idx int, pqPath, warcPath string, stats *ShardStats) {
	// Release lock
	os.Remove(s.lockPath(idx))
	// Store warc path in a sidecar
	sidecar := filepath.Join(s.dataDir, "parquet", fmt.Sprintf("%05d.warc_path", idx))
	os.WriteFile(sidecar, []byte(warcPath), 0o644)
}

func (s *FileStore) MarkCommitted(_ context.Context, idx int) {
	_ = os.MkdirAll(s.committedDir(), 0o755)
	os.WriteFile(s.committedPath(idx), []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)
}

func (s *FileStore) CommittedCount(_ context.Context) int {
	entries, err := os.ReadDir(s.committedDir())
	if err != nil {
		return 0
	}
	return len(entries)
}

func (s *FileStore) CommittedSet(_ context.Context) map[int]bool {
	entries, err := os.ReadDir(s.committedDir())
	if err != nil {
		return nil
	}
	set := make(map[int]bool, len(entries))
	for _, e := range entries {
		if idx, err := strconv.Atoi(strings.TrimLeft(e.Name(), "0")); err == nil {
			set[idx] = true
		} else if e.Name() == "00000" {
			set[0] = true
		}
	}
	return set
}

func (s *FileStore) GetWARCPath(_ context.Context, idx int) string {
	sidecar := filepath.Join(s.dataDir, "parquet", fmt.Sprintf("%05d.warc_path", idx))
	data, err := os.ReadFile(sidecar)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Telemetry no-ops for FileStore (no rate tracking without Redis).
func (s *FileStore) RecordEvent(_ context.Context, _ string)                                {}
func (s *FileStore) EventRate(_ context.Context, _ string, _ time.Duration) float64          { return 0 }
func (s *FileStore) TrimRates(_ context.Context)                                             {}
func (s *FileStore) SetWatcherStatus(_ context.Context, _ WatcherStatus)                     {}
func (s *FileStore) GetWatcherStatus(_ context.Context) (WatcherStatus, bool)                { return WatcherStatus{}, false }
func (s *FileStore) Log(_ context.Context, _, _, _ string)                                   {}

// NewStore creates the best available store: Redis if configured, else file-based.
func NewStore(dataDir, crawlID string) Store {
	if rs := NewRedisStore(crawlID); rs != nil && rs.Available() {
		return rs
	}
	return NewFileStore(dataDir, crawlID)
}
