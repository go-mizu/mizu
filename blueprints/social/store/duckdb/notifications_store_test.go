package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-mizu/blueprints/social/feature/notifications"
)

func TestNotificationsStore_Insert(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	actor := createTestAccount(t, db, "actor")

	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, actor.ID)

	store := NewNotificationsStore(db)

	notif := &notifications.Notification{
		ID:        newTestID(),
		AccountID: account.ID,
		Type:      "like",
		ActorID:   actor.ID,
		PostID:    post.ID,
		Read:      false,
		CreatedAt: testTime(),
	}

	if err := store.Insert(ctx, notif); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify
	got, err := store.GetByID(ctx, notif.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Type != notif.Type {
		t.Errorf("Type: got %q, want %q", got.Type, notif.Type)
	}
	if got.ActorID != notif.ActorID {
		t.Errorf("ActorID: got %q, want %q", got.ActorID, notif.ActorID)
	}
	if got.PostID != notif.PostID {
		t.Errorf("PostID: got %q, want %q", got.PostID, notif.PostID)
	}
}

func TestNotificationsStore_GetByID(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	notif := &notifications.Notification{
		ID:        newTestID(),
		AccountID: account.ID,
		Type:      "follow",
		Read:      false,
		CreatedAt: testTime(),
	}
	store.Insert(ctx, notif)

	got, err := store.GetByID(ctx, notif.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != notif.ID {
		t.Errorf("ID: got %q, want %q", got.ID, notif.ID)
	}
}

func TestNotificationsStore_GetByID_NotFound(t *testing.T) {
	db := setupTestStore(t)
	store := NewNotificationsStore(db)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestNotificationsStore_List(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	// Create notifications
	for _, notifType := range []string{"like", "follow", "mention", "repost"} {
		store.Insert(ctx, &notifications.Notification{
			ID:        newTestID(),
			AccountID: account.ID,
			Type:      notifType,
			Read:      false,
			CreatedAt: testTime(),
		})
	}

	// List all
	list, err := store.List(ctx, account.ID, 10, "", "", nil, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 4 {
		t.Errorf("List count: got %d, want 4", len(list))
	}
}

func TestNotificationsStore_List_ByTypes(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	// Create notifications of different types
	for _, notifType := range []string{"like", "like", "follow", "mention"} {
		store.Insert(ctx, &notifications.Notification{
			ID:        newTestID(),
			AccountID: account.ID,
			Type:      notifType,
			Read:      false,
			CreatedAt: testTime(),
		})
	}

	// List only likes
	list, err := store.List(ctx, account.ID, 10, "", "", []string{"like"}, nil)
	if err != nil {
		t.Fatalf("List (types) failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List (types) count: got %d, want 2", len(list))
	}
}

func TestNotificationsStore_List_ExcludeTypes(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	// Create notifications
	for _, notifType := range []string{"like", "follow", "mention"} {
		store.Insert(ctx, &notifications.Notification{
			ID:        newTestID(),
			AccountID: account.ID,
			Type:      notifType,
			Read:      false,
			CreatedAt: testTime(),
		})
	}

	// Exclude likes
	list, err := store.List(ctx, account.ID, 10, "", "", nil, []string{"like"})
	if err != nil {
		t.Fatalf("List (exclude) failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List (exclude) count: got %d, want 2", len(list))
	}
}

func TestNotificationsStore_MarkRead(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	notif := &notifications.Notification{
		ID:        newTestID(),
		AccountID: account.ID,
		Type:      "follow",
		Read:      false,
		CreatedAt: testTime(),
	}
	store.Insert(ctx, notif)

	// Mark as read
	if err := store.MarkRead(ctx, notif.ID); err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	// Verify
	got, _ := store.GetByID(ctx, notif.ID)
	if !got.Read {
		t.Error("Read: got false, want true")
	}
}

func TestNotificationsStore_MarkAllRead(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	// Create unread notifications
	for i := 0; i < 5; i++ {
		store.Insert(ctx, &notifications.Notification{
			ID:        newTestID(),
			AccountID: account.ID,
			Type:      "follow",
			Read:      false,
			CreatedAt: testTime(),
		})
	}

	// Mark all as read
	if err := store.MarkAllRead(ctx, account.ID); err != nil {
		t.Fatalf("MarkAllRead failed: %v", err)
	}

	// Verify unread count is 0
	count, _ := store.UnreadCount(ctx, account.ID)
	if count != 0 {
		t.Errorf("UnreadCount after MarkAllRead: got %d, want 0", count)
	}
}

func TestNotificationsStore_Delete(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	notif := &notifications.Notification{
		ID:        newTestID(),
		AccountID: account.ID,
		Type:      "follow",
		Read:      false,
		CreatedAt: testTime(),
	}
	store.Insert(ctx, notif)

	// Delete
	if err := store.Delete(ctx, notif.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err := store.GetByID(ctx, notif.ID)
	if err != sql.ErrNoRows {
		t.Error("Notification should be deleted")
	}
}

func TestNotificationsStore_DeleteAll(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	// Create notifications
	for i := 0; i < 5; i++ {
		store.Insert(ctx, &notifications.Notification{
			ID:        newTestID(),
			AccountID: account.ID,
			Type:      "follow",
			Read:      false,
			CreatedAt: testTime(),
		})
	}

	// Delete all
	if err := store.DeleteAll(ctx, account.ID); err != nil {
		t.Fatalf("DeleteAll failed: %v", err)
	}

	// Verify
	list, _ := store.List(ctx, account.ID, 10, "", "", nil, nil)
	if len(list) != 0 {
		t.Errorf("After DeleteAll, count: got %d, want 0", len(list))
	}
}

func TestNotificationsStore_UnreadCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	// Create mix of read and unread
	for i := 0; i < 3; i++ {
		store.Insert(ctx, &notifications.Notification{
			ID:        newTestID(),
			AccountID: account.ID,
			Type:      "follow",
			Read:      false,
			CreatedAt: testTime(),
		})
	}
	for i := 0; i < 2; i++ {
		store.Insert(ctx, &notifications.Notification{
			ID:        newTestID(),
			AccountID: account.ID,
			Type:      "follow",
			Read:      true,
			CreatedAt: testTime(),
		})
	}

	// Get unread count
	count, err := store.UnreadCount(ctx, account.ID)
	if err != nil {
		t.Fatalf("UnreadCount failed: %v", err)
	}

	if count != 3 {
		t.Errorf("UnreadCount: got %d, want 3", count)
	}
}

func TestNotificationsStore_Exists(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	actor := createTestAccount(t, db, "actor")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, actor.ID)

	store := NewNotificationsStore(db)

	// Create notification
	notif := &notifications.Notification{
		ID:        newTestID(),
		AccountID: account.ID,
		Type:      "like",
		ActorID:   actor.ID,
		PostID:    post.ID,
		Read:      false,
		CreatedAt: testTime(),
	}
	store.Insert(ctx, notif)

	// Check exists
	exists, err := store.Exists(ctx, account.ID, "like", actor.ID, post.ID)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Exists: got false, want true")
	}

	// Check doesn't exist for different type
	exists2, err := store.Exists(ctx, account.ID, "follow", actor.ID, post.ID)
	if err != nil {
		t.Fatalf("Exists (different type) failed: %v", err)
	}
	if exists2 {
		t.Error("Exists (different type): got true, want false")
	}
}

func TestNotificationsStore_Insert_WithoutOptionalFields(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewNotificationsStore(db)

	// Create notification without actor or post
	notif := &notifications.Notification{
		ID:        newTestID(),
		AccountID: account.ID,
		Type:      "admin_notice",
		Read:      false,
		CreatedAt: testTime(),
	}

	if err := store.Insert(ctx, notif); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify
	got, err := store.GetByID(ctx, notif.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ActorID != "" {
		t.Errorf("ActorID: got %q, want empty", got.ActorID)
	}
	if got.PostID != "" {
		t.Errorf("PostID: got %q, want empty", got.PostID)
	}
}
