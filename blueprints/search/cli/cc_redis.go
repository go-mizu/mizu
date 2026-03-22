package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// ccRedis wraps redis.Client with CC pipeline-specific helpers.
// All methods are no-ops when rdb is nil (graceful degradation to file-based state).
type ccRedis struct {
	rdb   *redis.Client
	crawl string
}

// newCCRedis returns a Redis wrapper. If REDIS_PASSWORD is not set,
// rdb will be nil and all methods become no-ops.
func newCCRedis(crawlID string) *ccRedis {
	return &ccRedis{rdb: ccRedisClient(), crawl: crawlID}
}

// ccRedisClient returns a Redis client if REDIS_PASSWORD is set.
// Returns nil if Redis is not configured.
func ccRedisClient() *redis.Client {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	if password == "" {
		return nil
	}
	return redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})
}

// Available checks if Redis is connected and responsive.
func (r *ccRedis) Available(ctx context.Context) bool {
	if r.rdb == nil {
		return false
	}
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.rdb.Ping(ctx2).Err() == nil
}

// Close closes the underlying Redis connection.
func (r *ccRedis) Close() error {
	if r.rdb == nil {
		return nil
	}
	return r.rdb.Close()
}

// key builds a namespaced Redis key: "cc:{crawl}:parts..."
func (r *ccRedis) key(parts ...string) string {
	k := "cc:" + r.crawl
	for _, p := range parts {
		k += ":" + p
	}
	return k
}

// ── Committed Set ──────────────────────────────────────────────────────────────

// AddCommitted marks a file index as committed (pushed to HF).
func (r *ccRedis) AddCommitted(ctx context.Context, fileIdx int) error {
	if r.rdb == nil {
		return nil
	}
	return r.rdb.SAdd(ctx, r.key("committed"), fileIdx).Err()
}

// AddCommittedBatch marks multiple file indices as committed.
func (r *ccRedis) AddCommittedBatch(ctx context.Context, indices []int) error {
	if r.rdb == nil || len(indices) == 0 {
		return nil
	}
	members := make([]interface{}, len(indices))
	for i, idx := range indices {
		members[i] = idx
	}
	return r.rdb.SAdd(ctx, r.key("committed"), members...).Err()
}

// IsCommitted checks if a file index is in the committed set.
func (r *ccRedis) IsCommitted(ctx context.Context, fileIdx int) bool {
	if r.rdb == nil {
		return false
	}
	ok, err := r.rdb.SIsMember(ctx, r.key("committed"), fileIdx).Result()
	return err == nil && ok
}

// CommittedCount returns the total number of committed shards.
func (r *ccRedis) CommittedCount(ctx context.Context) int {
	if r.rdb == nil {
		return 0
	}
	n, err := r.rdb.SCard(ctx, r.key("committed")).Result()
	if err != nil {
		return 0
	}
	return int(n)
}

// CommittedSet returns all committed file indices as a set.
func (r *ccRedis) CommittedSet(ctx context.Context) map[int]bool {
	if r.rdb == nil {
		return nil
	}
	members, err := r.rdb.SMembers(ctx, r.key("committed")).Result()
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

// ── Shard Stats ────────────────────────────────────────────────────────────────

// SetShardStats stores per-shard statistics in a Redis hash.
func (r *ccRedis) SetShardStats(ctx context.Context, fileIdx int, stats ccShardStats) error {
	if r.rdb == nil {
		return nil
	}
	key := r.key("stats", strconv.Itoa(fileIdx))
	return r.rdb.HSet(ctx, key, map[string]interface{}{
		"crawl_id":       stats.CrawlID,
		"file_idx":       stats.FileIdx,
		"rows":           stats.Rows,
		"html_bytes":     stats.HTMLBytes,
		"md_bytes":       stats.MDBytes,
		"parquet_bytes":  stats.ParquetBytes,
		"created_at":     stats.CreatedAt,
		"dur_download_s": stats.DurDownloadS,
		"dur_convert_s":  stats.DurConvertS,
		"dur_export_s":   stats.DurExportS,
		"dur_publish_s":  stats.DurPublishS,
		"peak_rss_mb":    stats.PeakRSSMB,
	}).Err()
}

// ── Pipeline Session State ─────────────────────────────────────────────────────

// RegisterPipeline registers a pipeline session with a 5-minute TTL.
func (r *ccRedis) RegisterPipeline(ctx context.Context, sessionID string) error {
	if r.rdb == nil {
		return nil
	}
	pipe := r.rdb.Pipeline()
	pipe.SAdd(ctx, r.key("pipelines"), sessionID)
	pipe.HSet(ctx, r.key("pipeline", sessionID), map[string]interface{}{
		"status":         "starting",
		"started_at":     time.Now().UTC().Format(time.RFC3339),
		"last_heartbeat": time.Now().UTC().Format(time.RFC3339),
	})
	pipe.Expire(ctx, r.key("pipeline", sessionID), 5*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

// UpdatePipeline updates fields on a pipeline session hash and refreshes TTL.
func (r *ccRedis) UpdatePipeline(ctx context.Context, sessionID string, fields map[string]interface{}) error {
	if r.rdb == nil {
		return nil
	}
	fields["last_heartbeat"] = time.Now().UTC().Format(time.RFC3339)
	pipe := r.rdb.Pipeline()
	pipe.HSet(ctx, r.key("pipeline", sessionID), fields)
	pipe.Expire(ctx, r.key("pipeline", sessionID), 5*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

// HeartbeatPipeline refreshes the session TTL and heartbeat timestamp.
func (r *ccRedis) HeartbeatPipeline(ctx context.Context, sessionID string) error {
	if r.rdb == nil {
		return nil
	}
	pipe := r.rdb.Pipeline()
	pipe.HSet(ctx, r.key("pipeline", sessionID), "last_heartbeat", time.Now().UTC().Format(time.RFC3339))
	pipe.Expire(ctx, r.key("pipeline", sessionID), 5*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

// UnregisterPipeline removes a pipeline session.
func (r *ccRedis) UnregisterPipeline(ctx context.Context, sessionID string) error {
	if r.rdb == nil {
		return nil
	}
	pipe := r.rdb.Pipeline()
	pipe.SRem(ctx, r.key("pipelines"), sessionID)
	pipe.Del(ctx, r.key("pipeline", sessionID))
	_, err := pipe.Exec(ctx)
	return err
}

// ActivePipelines returns all registered pipeline session IDs.
func (r *ccRedis) ActivePipelines(ctx context.Context) []string {
	if r.rdb == nil {
		return nil
	}
	members, err := r.rdb.SMembers(ctx, r.key("pipelines")).Result()
	if err != nil {
		return nil
	}
	// Filter out sessions whose hash has expired (TTL expired = dead session).
	var active []string
	for _, id := range members {
		exists, err := r.rdb.Exists(ctx, r.key("pipeline", id)).Result()
		if err == nil && exists > 0 {
			active = append(active, id)
		} else {
			// Clean up stale membership.
			r.rdb.SRem(ctx, r.key("pipelines"), id)
		}
	}
	return active
}

// PipelineState returns the current state of a pipeline session.
func (r *ccRedis) PipelineState(ctx context.Context, sessionID string) map[string]string {
	if r.rdb == nil {
		return nil
	}
	result, err := r.rdb.HGetAll(ctx, r.key("pipeline", sessionID)).Result()
	if err != nil {
		return nil
	}
	return result
}

// ── Watcher Status ─────────────────────────────────────────────────────────────

// SetWatcherStatus writes the latest watcher state to Redis.
func (r *ccRedis) SetWatcherStatus(ctx context.Context, status ccWatcherStatus) error {
	if r.rdb == nil {
		return nil
	}
	return r.rdb.HSet(ctx, r.key("watcher"), map[string]interface{}{
		"commit_number":    status.CommitNumber,
		"message":          status.Message,
		"commit_url":       status.CommitURL,
		"shards_in_commit": status.ShardsInCommit,
		"total_committed":  status.TotalCommitted,
		"timestamp":        status.Timestamp.UTC().Format(time.RFC3339),
	}).Err()
}

// GetWatcherStatus reads the latest watcher state from Redis.
func (r *ccRedis) GetWatcherStatus(ctx context.Context) (ccWatcherStatus, bool) {
	if r.rdb == nil {
		return ccWatcherStatus{}, false
	}
	vals, err := r.rdb.HGetAll(ctx, r.key("watcher")).Result()
	if err != nil || len(vals) == 0 {
		return ccWatcherStatus{}, false
	}
	cn, _ := strconv.Atoi(vals["commit_number"])
	sc, _ := strconv.Atoi(vals["shards_in_commit"])
	tc, _ := strconv.Atoi(vals["total_committed"])
	ts, _ := time.Parse(time.RFC3339, vals["timestamp"])
	return ccWatcherStatus{
		CommitNumber:   cn,
		Message:        vals["message"],
		CommitURL:      vals["commit_url"],
		ShardsInCommit: sc,
		TotalCommitted: tc,
		Timestamp:      ts,
	}, true
}

// ── Pending Queue (pipeline → watcher) ─────────────────────────────────────────

// PushPendingParquet adds a parquet file path to the watcher's pending queue.
func (r *ccRedis) PushPendingParquet(ctx context.Context, path string) error {
	if r.rdb == nil {
		return nil
	}
	return r.rdb.RPush(ctx, r.key("watcher", "pending"), path).Err()
}

// PendingCount returns the number of parquets waiting for watcher to commit.
func (r *ccRedis) PendingCount(ctx context.Context) int {
	if r.rdb == nil {
		return 0
	}
	n, err := r.rdb.LLen(ctx, r.key("watcher", "pending")).Result()
	if err != nil {
		return 0
	}
	return int(n)
}

// ── Rate Tracking (sorted sets with unix timestamp scores) ─────────────────────

// RecordEvent adds a timestamped event to a rate sorted set.
// The score is the unix timestamp; the member is a unique counter value.
func (r *ccRedis) recordEvent(ctx context.Context, eventKey string) error {
	if r.rdb == nil {
		return nil
	}
	now := float64(time.Now().UnixMilli()) / 1000.0
	// Use ZADD with unique member (timestamp + counter from INCR).
	counter, err := r.rdb.Incr(ctx, r.key("counter", eventKey)).Result()
	if err != nil {
		return err
	}
	member := fmt.Sprintf("%d", counter)
	return r.rdb.ZAdd(ctx, r.key("rate", eventKey), redis.Z{
		Score:  now,
		Member: member,
	}).Err()
}

// eventRate returns events/hour over the given window from a rate sorted set.
func (r *ccRedis) eventRate(ctx context.Context, eventKey string, window time.Duration) float64 {
	if r.rdb == nil {
		return 0
	}
	now := time.Now()
	minScore := fmt.Sprintf("%f", float64(now.Add(-window).UnixMilli())/1000.0)
	maxScore := fmt.Sprintf("%f", float64(now.UnixMilli())/1000.0)
	count, err := r.rdb.ZCount(ctx, r.key("rate", eventKey), minScore, maxScore).Result()
	if err != nil || count == 0 {
		return 0
	}
	return float64(count) / window.Hours()
}

// trimRateSet removes entries older than 1 hour to bound memory.
func (r *ccRedis) trimRateSet(ctx context.Context, eventKey string) {
	if r.rdb == nil {
		return
	}
	cutoff := fmt.Sprintf("%f", float64(time.Now().Add(-time.Hour).UnixMilli())/1000.0)
	r.rdb.ZRemRangeByScore(ctx, r.key("rate", eventKey), "-inf", cutoff)
}

// RecordPacked records a new parquet creation event.
func (r *ccRedis) RecordPacked(ctx context.Context) error {
	return r.recordEvent(ctx, "packed")
}

// RecordCommitted records an HF commit event (n shards).
func (r *ccRedis) RecordCommitted(ctx context.Context, n int) error {
	for i := 0; i < n; i++ {
		if err := r.recordEvent(ctx, "committed"); err != nil {
			return err
		}
	}
	return nil
}

// RecordDownloaded records a download completion event.
func (r *ccRedis) RecordDownloaded(ctx context.Context) error {
	return r.recordEvent(ctx, "downloaded")
}

// PackRate returns shards packed per hour over the given window.
func (r *ccRedis) PackRate(ctx context.Context, window time.Duration) float64 {
	return r.eventRate(ctx, "packed", window)
}

// CommitRate returns shards committed per hour over the given window.
func (r *ccRedis) CommitRate(ctx context.Context, window time.Duration) float64 {
	return r.eventRate(ctx, "committed", window)
}

// DownloadRate returns downloads per hour over the given window.
func (r *ccRedis) DownloadRate(ctx context.Context, window time.Duration) float64 {
	return r.eventRate(ctx, "downloaded", window)
}

// TrimRates removes old entries from all rate sorted sets.
func (r *ccRedis) TrimRates(ctx context.Context) {
	r.trimRateSet(ctx, "packed")
	r.trimRateSet(ctx, "committed")
	r.trimRateSet(ctx, "downloaded")
}

// ── Counters ───────────────────────────────────────────────────────────────────

// IncrCounter atomically increments a named counter and returns the new value.
func (r *ccRedis) IncrCounter(ctx context.Context, name string, delta int64) int64 {
	if r.rdb == nil {
		return 0
	}
	v, err := r.rdb.HIncrBy(ctx, r.key("counters"), name, delta).Result()
	if err != nil {
		return 0
	}
	return v
}

// GetCounter returns the current value of a named counter.
func (r *ccRedis) GetCounter(ctx context.Context, name string) int64 {
	if r.rdb == nil {
		return 0
	}
	v, err := r.rdb.HGet(ctx, r.key("counters"), name).Result()
	if err != nil {
		return 0
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

// ── Log Stream ─────────────────────────────────────────────────────────────────

// ccLogEntry represents a structured log entry from Redis Streams.
type ccLogEntry struct {
	ID      string
	Source  string // pipeline, watcher, scheduler
	Level   string // info, warn, error
	Message string
	Time    time.Time
	Fields  map[string]string
}

// Log adds a structured log entry to the Redis stream.
func (r *ccRedis) Log(ctx context.Context, source, level, msg string) error {
	if r.rdb == nil {
		return nil
	}
	err := r.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: r.key("log"),
		MaxLen: 1000,
		Approx: true,
		Values: map[string]interface{}{
			"source":  source,
			"level":   level,
			"message": msg,
			"time":    time.Now().UTC().Format(time.RFC3339),
		},
	}).Err()
	return err
}

// ── Hardware State ─────────────────────────────────────────────────────────────

// SetHardware stores the latest hardware snapshot in Redis.
func (r *ccRedis) SetHardware(ctx context.Context, fields map[string]interface{}) error {
	if r.rdb == nil {
		return nil
	}
	return r.rdb.HSet(ctx, r.key("hw"), fields).Err()
}

// SetSessionRSS updates the RSS measurement for a pipeline session.
func (r *ccRedis) SetSessionRSS(ctx context.Context, sessionID string, rssMB float64) error {
	if r.rdb == nil {
		return nil
	}
	return r.rdb.ZAdd(ctx, r.key("sessions", "rss"), redis.Z{
		Score:  rssMB,
		Member: sessionID,
	}).Err()
}

// ── Seed from CSV ──────────────────────────────────────────────────────────────

// SeedFromCSV populates Redis committed set and stats from an existing stats.csv.
// Only runs if the Redis committed set is empty (first run after Redis install).
func (r *ccRedis) SeedFromCSV(ctx context.Context, statsCSV, crawlID string) error {
	if r.rdb == nil {
		return nil
	}
	// Don't overwrite if Redis already has data.
	if r.CommittedCount(ctx) > 0 {
		return nil
	}
	stats, err := ccReadStatsCSV(statsCSV)
	if err != nil || len(stats) == 0 {
		return err
	}
	var indices []int
	for _, s := range stats {
		if s.CrawlID == crawlID {
			indices = append(indices, s.FileIdx)
			r.SetShardStats(ctx, s.FileIdx, s)
		}
	}
	if len(indices) > 0 {
		r.AddCommittedBatch(ctx, indices)
	}
	return nil
}
