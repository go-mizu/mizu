package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/notifications"
)

func TestNotificationsStore_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")
	actor := createTestAccount(t, store, "actor")

	notification := &notifications.Notification{
		ID:        newTestID(),
		AccountID: recipient.ID,
		Type:      notifications.NotifyReply,
		ActorID:   actor.ID,
		ThreadID:  newTestID(),
		CommentID: newTestID(),
		Message:   "Someone replied to your comment",
		Read:      false,
		CreatedAt: testTime(),
	}

	if err := store.Notifications().Create(ctx, notification); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify
	got, err := store.Notifications().GetByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Type != notification.Type {
		t.Errorf("Type: got %q, want %q", got.Type, notification.Type)
	}
	if got.Message != notification.Message {
		t.Errorf("Message: got %q, want %q", got.Message, notification.Message)
	}
	if got.Read {
		t.Error("Read: got true, want false")
	}
}

func TestNotificationsStore_GetByID(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")

	notification := &notifications.Notification{
		ID:        newTestID(),
		AccountID: recipient.ID,
		Type:      notifications.NotifyMention,
		Message:   "You were mentioned",
		Read:      false,
		CreatedAt: testTime(),
	}

	if err := store.Notifications().Create(ctx, notification); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Notifications().GetByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != notification.ID {
		t.Errorf("ID: got %q, want %q", got.ID, notification.ID)
	}
}

func TestNotificationsStore_GetByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Notifications().GetByID(ctx, "nonexistent")
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestNotificationsStore_List(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")

	// Create multiple notifications
	for i := 0; i < 5; i++ {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: recipient.ID,
			Type:      notifications.NotifyReply,
			Message:   "Notification " + string(rune('A'+i)),
			Read:      i >= 3, // Last 2 are read
			CreatedAt: testTime().Add(time.Duration(i) * time.Hour),
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create notification %d failed: %v", i, err)
		}
	}

	// List all
	list, err := store.Notifications().List(ctx, recipient.ID, notifications.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 5 {
		t.Errorf("List count: got %d, want 5", len(list))
	}

	// Should be ordered by created_at DESC
	if list[0].CreatedAt.Before(list[len(list)-1].CreatedAt) {
		t.Error("List should be ordered by created_at DESC")
	}
}

func TestNotificationsStore_List_Unread(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")

	// Create mix of read and unread
	for i := 0; i < 5; i++ {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: recipient.ID,
			Type:      notifications.NotifyReply,
			Message:   "Notification " + string(rune('A'+i)),
			Read:      i >= 3, // Last 2 are read
			CreatedAt: testTime(),
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create notification %d failed: %v", i, err)
		}
	}

	// List unread only
	unread, err := store.Notifications().List(ctx, recipient.ID, notifications.ListOpts{
		Limit:  10,
		Unread: true,
	})
	if err != nil {
		t.Fatalf("List unread failed: %v", err)
	}

	if len(unread) != 3 {
		t.Errorf("Unread count: got %d, want 3", len(unread))
	}

	// All should be unread
	for _, n := range unread {
		if n.Read {
			t.Error("Expected all to be unread")
		}
	}
}

func TestNotificationsStore_MarkAllRead(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")

	// Create unread notifications
	for i := 0; i < 5; i++ {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: recipient.ID,
			Type:      notifications.NotifyReply,
			Message:   "Notification " + string(rune('A'+i)),
			Read:      false,
			CreatedAt: testTime(),
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create notification %d failed: %v", i, err)
		}
	}

	// Mark all as read
	if err := store.Notifications().MarkAllRead(ctx, recipient.ID); err != nil {
		t.Fatalf("MarkAllRead failed: %v", err)
	}

	// Verify all are read
	unread, err := store.Notifications().List(ctx, recipient.ID, notifications.ListOpts{
		Limit:  10,
		Unread: true,
	})
	if err != nil {
		t.Fatalf("List unread failed: %v", err)
	}

	if len(unread) != 0 {
		t.Errorf("Unread count after MarkAllRead: got %d, want 0", len(unread))
	}
}

func TestNotificationsStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")

	notification := &notifications.Notification{
		ID:        newTestID(),
		AccountID: recipient.ID,
		Type:      notifications.NotifyReply,
		Message:   "Test notification",
		Read:      false,
		CreatedAt: testTime(),
	}

	if err := store.Notifications().Create(ctx, notification); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.Notifications().Delete(ctx, notification.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Notifications().GetByID(ctx, notification.ID)
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestNotificationsStore_DeleteBefore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")

	now := time.Now()

	// Create old notifications
	for i := 0; i < 3; i++ {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: recipient.ID,
			Type:      notifications.NotifyReply,
			Message:   "Old notification " + string(rune('A'+i)),
			Read:      false,
			CreatedAt: now.Add(-48 * time.Hour), // 2 days ago
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create old notification %d failed: %v", i, err)
		}
	}

	// Create recent notifications
	for i := 0; i < 2; i++ {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: recipient.ID,
			Type:      notifications.NotifyReply,
			Message:   "Recent notification " + string(rune('A'+i)),
			Read:      false,
			CreatedAt: now,
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create recent notification %d failed: %v", i, err)
		}
	}

	// Delete notifications older than 1 day
	if err := store.Notifications().DeleteBefore(ctx, now.Add(-24*time.Hour)); err != nil {
		t.Fatalf("DeleteBefore failed: %v", err)
	}

	// Should only have recent ones left
	list, err := store.Notifications().List(ctx, recipient.ID, notifications.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List count after DeleteBefore: got %d, want 2", len(list))
	}
}

func TestNotificationsStore_CountUnread(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")

	// Create mix of read and unread
	for i := 0; i < 5; i++ {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: recipient.ID,
			Type:      notifications.NotifyReply,
			Message:   "Notification " + string(rune('A'+i)),
			Read:      i >= 3, // Last 2 are read
			CreatedAt: testTime(),
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create notification %d failed: %v", i, err)
		}
	}

	count, err := store.Notifications().CountUnread(ctx, recipient.ID)
	if err != nil {
		t.Fatalf("CountUnread failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Unread count: got %d, want 3", count)
	}
}

func TestNotificationsStore_DifferentTypes(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")

	types := []notifications.NotificationType{
		notifications.NotifyReply,
		notifications.NotifyMention,
		notifications.NotifyThreadVote,
		notifications.NotifyCommentVote,
		notifications.NotifyFollow,
		notifications.NotifyMod,
	}

	for _, nt := range types {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: recipient.ID,
			Type:      nt,
			Message:   "Notification: " + string(nt),
			Read:      false,
			CreatedAt: testTime(),
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create %s notification failed: %v", nt, err)
		}

		got, err := store.Notifications().GetByID(ctx, notification.ID)
		if err != nil {
			t.Fatalf("GetByID %s failed: %v", nt, err)
		}

		if got.Type != nt {
			t.Errorf("Type: got %q, want %q", got.Type, nt)
		}
	}
}

func TestNotificationsStore_WithRelatedIDs(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	recipient := createTestAccount(t, store, "recipient")
	actor := createTestAccount(t, store, "actor")

	threadID := newTestID()
	commentID := newTestID()
	boardID := newTestID()

	notification := &notifications.Notification{
		ID:        newTestID(),
		AccountID: recipient.ID,
		Type:      notifications.NotifyReply,
		ActorID:   actor.ID,
		BoardID:   boardID,
		ThreadID:  threadID,
		CommentID: commentID,
		Message:   "Full notification",
		Read:      false,
		CreatedAt: testTime(),
	}

	if err := store.Notifications().Create(ctx, notification); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Notifications().GetByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ActorID != actor.ID {
		t.Errorf("ActorID: got %q, want %q", got.ActorID, actor.ID)
	}
	if got.BoardID != boardID {
		t.Errorf("BoardID: got %q, want %q", got.BoardID, boardID)
	}
	if got.ThreadID != threadID {
		t.Errorf("ThreadID: got %q, want %q", got.ThreadID, threadID)
	}
	if got.CommentID != commentID {
		t.Errorf("CommentID: got %q, want %q", got.CommentID, commentID)
	}
}

func TestNotificationsStore_DifferentUsers(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, store, "user1")
	user2 := createTestAccount(t, store, "user2")

	// Create notifications for user1
	for i := 0; i < 3; i++ {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: user1.ID,
			Type:      notifications.NotifyReply,
			Message:   "User1 notification " + string(rune('A'+i)),
			Read:      false,
			CreatedAt: testTime(),
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create user1 notification %d failed: %v", i, err)
		}
	}

	// Create notifications for user2
	for i := 0; i < 2; i++ {
		notification := &notifications.Notification{
			ID:        newTestID(),
			AccountID: user2.ID,
			Type:      notifications.NotifyReply,
			Message:   "User2 notification " + string(rune('A'+i)),
			Read:      false,
			CreatedAt: testTime(),
		}

		if err := store.Notifications().Create(ctx, notification); err != nil {
			t.Fatalf("Create user2 notification %d failed: %v", i, err)
		}
	}

	// User1's notifications
	list1, err := store.Notifications().List(ctx, user1.ID, notifications.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List user1 failed: %v", err)
	}

	if len(list1) != 3 {
		t.Errorf("User1 notification count: got %d, want 3", len(list1))
	}

	// User2's notifications
	list2, err := store.Notifications().List(ctx, user2.ID, notifications.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List user2 failed: %v", err)
	}

	if len(list2) != 2 {
		t.Errorf("User2 notification count: got %d, want 2", len(list2))
	}
}
