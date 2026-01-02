package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/bookmarks"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
)

func TestBookmarksStore(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	questionsStore := NewQuestionsStore(db)
	bookmarksStore := NewBookmarksStore(db)
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

	question := &questions.Question{
		ID:        "q-1",
		AuthorID:  account.ID,
		Title:     "Question",
		Body:      "Body",
		CreatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
		UpdatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
	}
	if err := questionsStore.Create(ctx, question); err != nil {
		t.Fatalf("create question: %v", err)
	}

	bookmark := &bookmarks.Bookmark{
		ID:         "b-1",
		AccountID:  account.ID,
		QuestionID: question.ID,
		CreatedAt:  mustTime(2024, time.January, 3, 10, 0, 0),
	}
	if err := bookmarksStore.Create(ctx, bookmark); err != nil {
		t.Fatalf("create bookmark: %v", err)
	}
	if err := bookmarksStore.Create(ctx, bookmark); err != nil {
		t.Fatalf("duplicate bookmark: %v", err)
	}

	listed, err := bookmarksStore.ListByAccount(ctx, account.ID, 10)
	if err != nil {
		t.Fatalf("list bookmarks: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != bookmark.ID {
		t.Fatalf("unexpected bookmarks: %#v", listed)
	}

	if err := bookmarksStore.Delete(ctx, account.ID, question.ID); err != nil {
		t.Fatalf("delete bookmark: %v", err)
	}

	remaining, err := bookmarksStore.ListByAccount(ctx, account.ID, 10)
	if err != nil {
		t.Fatalf("list remaining: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no bookmarks, got %#v", remaining)
	}
}
