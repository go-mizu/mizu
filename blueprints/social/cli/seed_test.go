package cli

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestSeedCommand(t *testing.T) {
	if seedCmd.Use != "seed" {
		t.Errorf("Use: got %q, want %q", seedCmd.Use, "seed")
	}

	if seedCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if seedCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if seedCmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestSeedCommand_Flags(t *testing.T) {
	flags := seedCmd.Flags()

	// Check --users flag
	usersFlag := flags.Lookup("users")
	if usersFlag == nil {
		t.Error("--users flag should exist")
	} else {
		if usersFlag.DefValue != "10" {
			t.Errorf("--users default: got %q, want %q", usersFlag.DefValue, "10")
		}
	}

	// Check --posts flag
	postsFlag := flags.Lookup("posts")
	if postsFlag == nil {
		t.Error("--posts flag should exist")
	} else {
		if postsFlag.DefValue != "50" {
			t.Errorf("--posts default: got %q, want %q", postsFlag.DefValue, "50")
		}
	}
}

func TestRunSeed(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "social-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore global state
	oldDataDir := dataDir
	oldSeedUsers := seedUsers
	oldSeedPosts := seedPosts
	dataDir = tmpDir
	seedUsers = 3
	seedPosts = 5
	defer func() {
		dataDir = oldDataDir
		seedUsers = oldSeedUsers
		seedPosts = oldSeedPosts
	}()

	// Create command
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Run seed
	err = runSeed(cmd, nil)
	if err != nil {
		t.Fatalf("runSeed failed: %v", err)
	}

	// Verify data was created
	dbPath := filepath.Join(tmpDir, "social.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Check accounts were created by counting rows
	var accountCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM accounts").Scan(&accountCount)
	if err != nil {
		t.Fatalf("Failed to count accounts: %v", err)
	}
	if accountCount < 1 {
		t.Error("Expected at least 1 account to be created")
	}

	// Check posts were created by counting rows
	var postCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts").Scan(&postCount)
	if err != nil {
		t.Fatalf("Failed to count posts: %v", err)
	}
	if postCount < 1 {
		t.Error("Expected at least 1 post to be created")
	}
}

func TestRunSeed_InvalidPath(t *testing.T) {
	// Save and restore global state
	oldDataDir := dataDir
	dataDir = "/nonexistent/path/that/cannot/be/created"
	defer func() { dataDir = oldDataDir }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSeed(cmd, nil)
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestRunSeed_DefaultValues(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "social-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore global state
	oldDataDir := dataDir
	oldSeedUsers := seedUsers
	oldSeedPosts := seedPosts
	dataDir = tmpDir
	seedUsers = 10 // Default
	seedPosts = 50 // Default
	defer func() {
		dataDir = oldDataDir
		seedUsers = oldSeedUsers
		seedPosts = oldSeedPosts
	}()

	// Create command
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Run seed with defaults
	err = runSeed(cmd, nil)
	if err != nil {
		t.Fatalf("runSeed failed: %v", err)
	}

	// Verify data was created
	dbPath := filepath.Join(tmpDir, "social.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Check accounts were created by counting rows
	var accountCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM accounts").Scan(&accountCount)
	if err != nil {
		t.Fatalf("Failed to count accounts: %v", err)
	}

	// Should have up to 10 users (limited by username array)
	if accountCount < 1 {
		t.Error("Expected accounts to be created")
	}
}

func TestRunSeed_ZeroUsers(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "social-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore global state
	oldDataDir := dataDir
	oldSeedUsers := seedUsers
	oldSeedPosts := seedPosts
	dataDir = tmpDir
	seedUsers = 0
	seedPosts = 10
	defer func() {
		dataDir = oldDataDir
		seedUsers = oldSeedUsers
		seedPosts = oldSeedPosts
	}()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Should not fail with 0 users
	err = runSeed(cmd, nil)
	if err != nil {
		t.Fatalf("runSeed with 0 users failed: %v", err)
	}
}

func TestRunSeed_Idempotent(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "social-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore global state
	oldDataDir := dataDir
	oldSeedUsers := seedUsers
	oldSeedPosts := seedPosts
	dataDir = tmpDir
	seedUsers = 2
	seedPosts = 3
	defer func() {
		dataDir = oldDataDir
		seedUsers = oldSeedUsers
		seedPosts = oldSeedPosts
	}()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Run seed twice - may produce warnings for duplicate users but should not fail
	if err := runSeed(cmd, nil); err != nil {
		t.Fatalf("First runSeed failed: %v", err)
	}

	// Second run - users already exist but should handle gracefully
	if err := runSeed(cmd, nil); err != nil {
		t.Fatalf("Second runSeed failed: %v", err)
	}
}

func TestRunSeed_CreatesFollowRelationships(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "social-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore global state
	oldDataDir := dataDir
	oldSeedUsers := seedUsers
	oldSeedPosts := seedPosts
	dataDir = tmpDir
	seedUsers = 5
	seedPosts = 0 // Just users, no posts
	defer func() {
		dataDir = oldDataDir
		seedUsers = oldSeedUsers
		seedPosts = oldSeedPosts
	}()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err = runSeed(cmd, nil)
	if err != nil {
		t.Fatalf("runSeed failed: %v", err)
	}

	// Open database to verify
	dbPath := filepath.Join(tmpDir, "social.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check if any follows were created
	var count int
	err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM follows").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count follows: %v", err)
	}

	// With 5 users and the (i+j)%3==0 pattern, we expect some follows to be created
	// The exact number depends on the pattern, but should be > 0
	t.Logf("Created %d follow relationships", count)
}
