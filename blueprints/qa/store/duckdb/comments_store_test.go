package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/comments"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
)

func TestCommentsStore(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	questionsStore := NewQuestionsStore(db)
	commentsStore := NewCommentsStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           "acc-1",
		Username:     "author",
		Email:        "author@example.com",
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

	commentOne := &comments.Comment{
		ID:         "c-1",
		TargetType: comments.TargetQuestion,
		TargetID:   question.ID,
		AuthorID:   account.ID,
		Body:       "First",
		Score:      1,
		CreatedAt:  mustTime(2024, time.January, 3, 10, 0, 0),
		UpdatedAt:  mustTime(2024, time.January, 3, 10, 0, 0),
	}
	commentTwo := &comments.Comment{
		ID:         "c-2",
		TargetType: comments.TargetQuestion,
		TargetID:   question.ID,
		AuthorID:   account.ID,
		Body:       "Second",
		Score:      2,
		CreatedAt:  mustTime(2024, time.January, 3, 11, 0, 0),
		UpdatedAt:  mustTime(2024, time.January, 3, 11, 0, 0),
	}

	if err := commentsStore.Create(ctx, commentTwo); err != nil {
		t.Fatalf("create comment two: %v", err)
	}
	if err := commentsStore.Create(ctx, commentOne); err != nil {
		t.Fatalf("create comment one: %v", err)
	}

	listed, err := commentsStore.ListByTarget(ctx, comments.TargetQuestion, question.ID, comments.ListOpts{})
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(listed) != 2 || listed[0].ID != "c-1" {
		t.Fatalf("unexpected order: %#v", listed)
	}

	if err := commentsStore.UpdateScore(ctx, "c-1", 2); err != nil {
		t.Fatalf("update score: %v", err)
	}

	updated, err := commentsStore.ListByTarget(ctx, comments.TargetQuestion, question.ID, comments.ListOpts{})
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if updated[0].Score != 3 {
		t.Fatalf("score not updated: %#v", updated[0])
	}

	if err := commentsStore.Delete(ctx, "c-2"); err != nil {
		t.Fatalf("delete comment: %v", err)
	}

	remaining, err := commentsStore.ListByTarget(ctx, comments.TargetQuestion, question.ID, comments.ListOpts{})
	if err != nil {
		t.Fatalf("list remaining: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != "c-1" {
		t.Fatalf("unexpected remaining comments: %#v", remaining)
	}
}
