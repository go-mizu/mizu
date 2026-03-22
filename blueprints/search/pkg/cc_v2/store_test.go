package cc_v2

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFileStoreClaim tests that file-based locking works correctly.
func TestFileStoreClaim(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir, "test-crawl")
	ctx := context.Background()

	// First claim should succeed.
	if !s.Claim(ctx, 42) {
		t.Fatal("first claim should succeed")
	}

	// Second claim should fail (lock held).
	if s.Claim(ctx, 42) {
		t.Fatal("second claim should fail")
	}

	// Release and re-claim should succeed.
	s.Release(ctx, 42)
	if !s.Claim(ctx, 42) {
		t.Fatal("claim after release should succeed")
	}
}

// TestFileStoreStalelock tests that stale lock files are cleaned up.
func TestFileStoreStalelock(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir, "test-crawl")
	ctx := context.Background()

	// Create a stale lock manually.
	lockDir := filepath.Join(dir, "locks")
	os.MkdirAll(lockDir, 0o755)
	lockPath := filepath.Join(lockDir, "00042.lock")
	os.WriteFile(lockPath, []byte("old:1234\n"), 0o644)

	// Backdate the lock file to 31 minutes ago.
	staleTime := time.Now().Add(-31 * time.Minute)
	os.Chtimes(lockPath, staleTime, staleTime)

	// Claim should succeed (stale lock auto-removed).
	if !s.Claim(ctx, 42) {
		t.Fatal("claim with stale lock should succeed")
	}
}

// TestFileStoreLifecycle tests the full shard lifecycle.
func TestFileStoreLifecycle(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir, "test-crawl")
	ctx := context.Background()

	// Initial state: not committed, not ready.
	if s.IsCommitted(ctx, 0) {
		t.Fatal("should not be committed initially")
	}
	if s.GetShardState(ctx, 0) != ShardNone {
		t.Fatal("should be ShardNone initially")
	}

	// Claim.
	if !s.Claim(ctx, 0) {
		t.Fatal("claim should succeed")
	}
	if s.GetShardState(ctx, 0) != ShardClaimed {
		t.Fatal("should be ShardClaimed after claim")
	}

	// Mark ready (creates parquet dir + warc_path sidecar).
	pqDir := filepath.Join(dir, "parquet")
	os.MkdirAll(pqDir, 0o755)
	pqPath := filepath.Join(pqDir, "00000.parquet")
	os.WriteFile(pqPath, []byte("test"), 0o644)
	s.MarkReady(ctx, 0, pqPath, "/tmp/test.warc.gz", &ShardStats{Rows: 100})

	if s.GetShardState(ctx, 0) != ShardReady {
		t.Fatal("should be ShardReady after MarkReady")
	}

	// WARC path should be stored.
	if wp := s.GetWARCPath(ctx, 0); wp != "/tmp/test.warc.gz" {
		t.Fatalf("WARC path mismatch: %q", wp)
	}

	// Lock should be released.
	lockPath := filepath.Join(dir, "locks", "00000.lock")
	if _, err := os.Stat(lockPath); err == nil {
		t.Fatal("lock should be released after MarkReady")
	}

	// Mark committed.
	s.MarkCommitted(ctx, 0)
	if !s.IsCommitted(ctx, 0) {
		t.Fatal("should be committed")
	}
	if s.GetShardState(ctx, 0) != ShardCommitted {
		t.Fatal("should be ShardCommitted")
	}

	// CommittedCount and CommittedSet.
	if n := s.CommittedCount(ctx); n != 1 {
		t.Fatalf("committed count: got %d, want 1", n)
	}
	set := s.CommittedSet(ctx)
	if !set[0] {
		t.Fatal("committed set should contain 0")
	}
}

// TestFileStoreDoubleClaim tests that two stores can't claim the same shard.
func TestFileStoreDoubleClaim(t *testing.T) {
	dir := t.TempDir()
	s1 := NewFileStore(dir, "test-crawl")
	s2 := NewFileStore(dir, "test-crawl")
	ctx := context.Background()

	if !s1.Claim(ctx, 5) {
		t.Fatal("s1 claim should succeed")
	}
	if s2.Claim(ctx, 5) {
		t.Fatal("s2 claim should fail (s1 holds lock)")
	}

	s1.Release(ctx, 5)
	if !s2.Claim(ctx, 5) {
		t.Fatal("s2 claim should succeed after s1 release")
	}
}
