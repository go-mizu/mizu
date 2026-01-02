package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
)

func TestAccountsStoreCRUD(t *testing.T) {
	db := newTestDB(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	createdAt := mustTime(2024, time.January, 10, 12, 0, 0)
	updatedAt := mustTime(2024, time.January, 11, 12, 0, 0)

	account := &accounts.Account{
		ID:           "acc-1",
		Username:     "UserOne",
		Email:        "user1@example.com",
		PasswordHash: "hash",
		DisplayName:  "User One",
		Bio:          "bio",
		AvatarURL:    "https://example.com/avatar.png",
		Location:     "Earth",
		WebsiteURL:   "https://example.com",
		Reputation:   10,
		IsModerator:  true,
		IsAdmin:      false,
		IsSuspended:  false,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}

	if err := store.Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	got, err := store.GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if got.Username != account.Username || got.Email != account.Email {
		t.Fatalf("unexpected account: %#v", got)
	}
	if !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected timestamps: %#v", got)
	}

	byUsername, err := store.GetByUsername(ctx, "userone")
	if err != nil {
		t.Fatalf("get by username: %v", err)
	}
	if byUsername.ID != account.ID {
		t.Fatalf("unexpected username match: %#v", byUsername)
	}

	byEmail, err := store.GetByEmail(ctx, "USER1@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if byEmail.ID != account.ID {
		t.Fatalf("unexpected email match: %#v", byEmail)
	}

	ids, err := store.GetByIDs(ctx, []string{account.ID, "missing"})
	if err != nil {
		t.Fatalf("get by ids: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 account, got %d", len(ids))
	}

	account.DisplayName = "Updated"
	account.Reputation = 42
	account.IsAdmin = true
	account.UpdatedAt = mustTime(2024, time.January, 12, 12, 0, 0)
	if err := store.Update(ctx, account); err != nil {
		t.Fatalf("update account: %v", err)
	}

	updated, err := store.GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.DisplayName != "Updated" || updated.Reputation != 42 || !updated.IsAdmin {
		t.Fatalf("update not applied: %#v", updated)
	}

	if err := store.Delete(ctx, account.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := store.GetByID(ctx, account.ID); err != accounts.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestAccountsStoreListAndSearch(t *testing.T) {
	db := newTestDB(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	accountsList := []*accounts.Account{
		{
			ID:           "acc-1",
			Username:     "alpha",
			Email:        "a@example.com",
			PasswordHash: "hash",
			DisplayName:  "Alpha",
			Reputation:   5,
			CreatedAt:    mustTime(2024, time.January, 1, 10, 0, 0),
			UpdatedAt:    mustTime(2024, time.January, 1, 10, 0, 0),
		},
		{
			ID:           "acc-2",
			Username:     "beta",
			Email:        "b@example.com",
			PasswordHash: "hash",
			DisplayName:  "Beta User",
			Reputation:   15,
			CreatedAt:    mustTime(2024, time.January, 2, 10, 0, 0),
			UpdatedAt:    mustTime(2024, time.January, 2, 10, 0, 0),
		},
	}

	for _, account := range accountsList {
		if err := store.Create(ctx, account); err != nil {
			t.Fatalf("create account: %v", err)
		}
	}

	listed, err := store.List(ctx, accounts.ListOpts{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 2 || listed[0].ID != "acc-2" {
		t.Fatalf("unexpected list order: %#v", listed)
	}

	byReputation, err := store.List(ctx, accounts.ListOpts{OrderBy: "reputation"})
	if err != nil {
		t.Fatalf("list by reputation: %v", err)
	}
	if len(byReputation) != 2 || byReputation[0].ID != "acc-2" {
		t.Fatalf("unexpected reputation order: %#v", byReputation)
	}

	found, err := store.Search(ctx, "beta", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 1 || found[0].ID != "acc-2" {
		t.Fatalf("unexpected search result: %#v", found)
	}
}

func TestAccountsStoreSessions(t *testing.T) {
	db := newTestDB(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           "acc-1",
		Username:     "alpha",
		Email:        "a@example.com",
		PasswordHash: "hash",
		CreatedAt:    mustTime(2024, time.January, 1, 10, 0, 0),
		UpdatedAt:    mustTime(2024, time.January, 1, 10, 0, 0),
	}
	if err := store.Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	activeSession := &accounts.Session{
		ID:        "sess-1",
		AccountID: account.ID,
		Token:     "token-1",
		UserAgent: "ua",
		IP:        "127.0.0.1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := store.CreateSession(ctx, activeSession); err != nil {
		t.Fatalf("create session: %v", err)
	}

	got, err := store.GetSessionByToken(ctx, activeSession.Token)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.AccountID != account.ID {
		t.Fatalf("unexpected session: %#v", got)
	}

	if err := store.DeleteSession(ctx, activeSession.Token); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := store.GetSessionByToken(ctx, activeSession.Token); err != accounts.ErrSessionExpired {
		t.Fatalf("expected expired, got %v", err)
	}

	sessionOne := &accounts.Session{
		ID:        "sess-2",
		AccountID: account.ID,
		Token:     "token-2",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	sessionTwo := &accounts.Session{
		ID:        "sess-3",
		AccountID: account.ID,
		Token:     "token-3",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := store.CreateSession(ctx, sessionOne); err != nil {
		t.Fatalf("create session one: %v", err)
	}
	if err := store.CreateSession(ctx, sessionTwo); err != nil {
		t.Fatalf("create session two: %v", err)
	}
	if err := store.DeleteSessionsByAccount(ctx, account.ID); err != nil {
		t.Fatalf("delete sessions by account: %v", err)
	}
	if _, err := store.GetSessionByToken(ctx, sessionOne.Token); err != accounts.ErrSessionExpired {
		t.Fatalf("expected expired, got %v", err)
	}

	expired := &accounts.Session{
		ID:        "sess-4",
		AccountID: account.ID,
		Token:     "token-4",
		ExpiresAt: mustTime(2000, time.January, 1, 0, 0, 0),
		CreatedAt: mustTime(2000, time.January, 1, 0, 0, 0),
	}
	active := &accounts.Session{
		ID:        "sess-5",
		AccountID: account.ID,
		Token:     "token-5",
		ExpiresAt: mustTime(2099, time.January, 1, 0, 0, 0),
		CreatedAt: mustTime(2099, time.January, 1, 0, 0, 0),
	}
	if err := store.CreateSession(ctx, expired); err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	if err := store.CreateSession(ctx, active); err != nil {
		t.Fatalf("create active session: %v", err)
	}
	if err := store.CleanExpiredSessions(ctx); err != nil {
		t.Fatalf("clean expired: %v", err)
	}
	if _, err := store.GetSessionByToken(ctx, expired.Token); err != accounts.ErrSessionExpired {
		t.Fatalf("expected expired, got %v", err)
	}
	if _, err := store.GetSessionByToken(ctx, active.Token); err != nil {
		t.Fatalf("expected active session, got %v", err)
	}
}
