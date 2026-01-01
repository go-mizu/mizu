package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupTestStore creates a new in-memory DuckDB store for testing.
func setupTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	store, err := New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}
	if err := store.Ensure(context.Background()); err != nil {
		store.Close()
		t.Fatalf("failed to ensure schema: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// Test helper functions
func newTestUser(id, email string) *User {
	now := time.Now().Truncate(time.Microsecond)
	return &User{
		ID:            id,
		Email:         email,
		Name:          "Test User",
		PasswordHash:  "hash123",
		StorageQuota:  10737418240,
		StorageUsed:   0,
		IsAdmin:       false,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func newTestFile(id, userID, name string) *File {
	now := time.Now().Truncate(time.Microsecond)
	return &File{
		ID:         id,
		UserID:     userID,
		Name:       name,
		MimeType:   "text/plain",
		Size:       1024,
		StorageKey: "storage/" + id,
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func newTestFolder(id, userID, name string) *Folder {
	now := time.Now().Truncate(time.Microsecond)
	return &Folder{
		ID:        id,
		UserID:    userID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newTestShare(id, ownerID, resourceType, resourceID string) *Share {
	now := time.Now().Truncate(time.Microsecond)
	return &Share{
		ID:           id,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OwnerID:      ownerID,
		Permission:   "viewer",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func newTestActivity(id, userID, action, resourceType, resourceID string) *Activity {
	return &Activity{
		ID:           id,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
}

func newTestComment(id, fileID, userID, content string) *Comment {
	now := time.Now().Truncate(time.Microsecond)
	return &Comment{
		ID:        id,
		FileID:    fileID,
		UserID:    userID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newTestSession(id, userID, tokenHash string) *Session {
	now := time.Now().Truncate(time.Microsecond)
	return &Session{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}
}

func newTestSettings(userID string) *Settings {
	return &Settings{
		UserID:               userID,
		Theme:                "system",
		Language:             "en",
		Timezone:             "UTC",
		ListView:             "list",
		SortBy:               "name",
		SortOrder:            "asc",
		NotificationsEnabled: true,
		EmailNotifications:   true,
		UpdatedAt:            time.Now().Truncate(time.Microsecond),
	}
}

// ============================================================
// Store Initialization Tests
// ============================================================

func TestNew_NilDB(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("expected error for nil db, got nil")
	}
}

func TestNew_ValidDB(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if store == nil {
		t.Error("expected store, got nil")
	}
}

func TestEnsure_CreatesSchema(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := store.Ensure(ctx); err != nil {
		t.Fatalf("ensure failed: %v", err)
	}

	// Verify tables exist by querying them
	tables := []string{"users", "sessions", "files", "folders", "shares", "activities", "comments", "settings", "file_versions"}
	for _, table := range tables {
		_, err := db.ExecContext(ctx, "SELECT 1 FROM "+table+" LIMIT 1")
		if err != nil {
			t.Errorf("table %s not created: %v", table, err)
		}
	}
}

func TestEnsure_Idempotent(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := store.Ensure(ctx); err != nil {
		t.Fatalf("first ensure failed: %v", err)
	}
	if err := store.Ensure(ctx); err != nil {
		t.Fatalf("second ensure failed: %v", err)
	}
}

// ============================================================
// Store Utility Tests
// ============================================================

func TestDB_ReturnsConnection(t *testing.T) {
	store := setupTestStore(t)
	if store.DB() == nil {
		t.Error("expected db connection, got nil")
	}
}

func TestStats_EmptyDB(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	for table, count := range stats {
		if c, ok := count.(int64); ok && c != 0 {
			t.Errorf("expected 0 for %s, got %d", table, c)
		}
	}
}

func TestStats_WithData(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create some test data
	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	file := newTestFile("file1", "user1", "test.txt")
	if err := store.CreateFile(ctx, file); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	if c, ok := stats["users"].(int64); !ok || c != 1 {
		t.Errorf("expected users=1, got %v", stats["users"])
	}
	if c, ok := stats["files"].(int64); !ok || c != 1 {
		t.Errorf("expected files=1, got %v", stats["files"])
	}
}

func TestClose_ClosesConnection(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// Verify connection is closed by trying to use it
	if err := db.Ping(); err == nil {
		t.Error("expected error after close, got nil")
	}
}

func TestExec_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	result, err := store.Exec(ctx, "UPDATE users SET name = ? WHERE id = ?", "New Name", "user1")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row affected, got %d", rows)
	}
}

func TestQuery_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	rows, err := store.Query(ctx, "SELECT id, email FROM users")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var id, email string
		if err := rows.Scan(&id, &email); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestQueryRow_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	var id string
	row := store.QueryRow(ctx, "SELECT id FROM users WHERE email = ?", "test@example.com")
	if err := row.Scan(&id); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if id != "user1" {
		t.Errorf("expected user1, got %s", id)
	}
}
