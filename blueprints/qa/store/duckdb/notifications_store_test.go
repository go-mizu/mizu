package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/notifications"
)

func TestNotificationsStore(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	notificationsStore := NewNotificationsStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           "acc-1",
		Username:     "user",
		Email:        "user@example.com",
		PasswordHash: "hash",
		CreatedAt:    mustTime(2024, time.January, 1, 10, 0, 0),
		UpdatedAt:    mustTime(2024, time.January, 1, 10, 0, 0),
	}
	if err := accountsStore.Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	n1 := &notifications.Notification{
		ID:        "n-1",
		AccountID: account.ID,
		Type:      "answer",
		Title:     "New answer",
		IsRead:    false,
		CreatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
	}
	n2 := &notifications.Notification{
		ID:        "n-2",
		AccountID: account.ID,
		Type:      "comment",
		Title:     "New comment",
		IsRead:    false,
		CreatedAt: mustTime(2024, time.January, 3, 10, 0, 0),
	}
	if err := notificationsStore.Create(ctx, n1); err != nil {
		t.Fatalf("create notification 1: %v", err)
	}
	if err := notificationsStore.Create(ctx, n2); err != nil {
		t.Fatalf("create notification 2: %v", err)
	}

	listed, err := notificationsStore.ListByAccount(ctx, account.ID, 10)
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(listed) != 2 || listed[0].ID != n2.ID {
		t.Fatalf("unexpected notification order: %#v", listed)
	}

	count, err := notificationsStore.GetUnreadCount(ctx, account.ID)
	if err != nil {
		t.Fatalf("get unread count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 unread, got %d", count)
	}

	if err := notificationsStore.MarkRead(ctx, n1.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	count, err = notificationsStore.GetUnreadCount(ctx, account.ID)
	if err != nil {
		t.Fatalf("get unread count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 unread, got %d", count)
	}

	if err := notificationsStore.MarkAllRead(ctx, account.ID); err != nil {
		t.Fatalf("mark all read: %v", err)
	}

	count, err = notificationsStore.GetUnreadCount(ctx, account.ID)
	if err != nil {
		t.Fatalf("get unread count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 unread, got %d", count)
	}
}
