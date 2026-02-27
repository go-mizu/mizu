package recrawler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenFailedDB_RemovesStaleLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "failed.duckdb")

	// Create a fake .lock file with a dead PID (99999999 almost certainly doesn't exist).
	// No DB file is pre-created; DuckDB will create a fresh one after the stale lock is removed.
	lockPath := path + ".lock"
	os.WriteFile(lockPath, []byte("PID=99999999\n"), 0644)

	db, err := OpenFailedDB(path)
	if err != nil {
		t.Fatalf("OpenFailedDB should succeed after removing stale lock, got: %v", err)
	}
	db.Close()

	if _, statErr := os.Stat(lockPath); !os.IsNotExist(statErr) {
		t.Error("stale .lock file should have been removed")
	}
}

func TestOpenFailedDB_NoCrashWithNoLockFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "failed.duckdb")
	// No lock file, no db file — should create fresh
	db, err := OpenFailedDB(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	db.Close()
}

func TestOpenFailedDB_DoesNotRemoveLiveLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "failed.duckdb")

	// Write our own PID (definitely alive)
	lockPath := path + ".lock"
	os.WriteFile(lockPath, []byte("PID="+fmt.Sprintf("%d", os.Getpid())+"\n"), 0644)
	os.WriteFile(path, []byte("not a real db"), 0644)

	// Should NOT remove the lock (our process is alive), so open should fail on bad db
	db, err := OpenFailedDB(path)
	// We expect either error (bad db) or success if DuckDB handles it
	// Key: the lock file should still exist since PID is alive
	if db != nil {
		db.Close()
	}
	_ = err // may or may not error — what matters is lock file preservation
	if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
		t.Error("lock file for live PID should NOT be removed")
	}
}
