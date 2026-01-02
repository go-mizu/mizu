package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

func TestQuestionsStoreCRUD(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	questionsStore := NewQuestionsStore(db)
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
		Title:     "How to test store?",
		Body:      "Body",
		BodyHTML:  "<p>Body</p>",
		Score:     2,
		ViewCount: 5,
		CreatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
		UpdatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
	}
	if err := questionsStore.Create(ctx, question); err != nil {
		t.Fatalf("create question: %v", err)
	}

	got, err := questionsStore.GetByID(ctx, question.ID)
	if err != nil {
		t.Fatalf("get question: %v", err)
	}
	if got.Title != question.Title || got.Body != question.Body {
		t.Fatalf("unexpected question: %#v", got)
	}

	question.Title = "Updated title"
	question.Body = "Updated body"
	question.Score = 10
	question.AnswerCount = 1
	question.CommentCount = 2
	question.FavoriteCount = 3
	question.AcceptedAnswerID = "a-1"
	question.BountyAmount = 50
	question.IsClosed = true
	question.CloseReason = "duplicate"
	question.UpdatedAt = mustTime(2024, time.January, 3, 10, 0, 0)
	if err := questionsStore.Update(ctx, question); err != nil {
		t.Fatalf("update question: %v", err)
	}

	updated, err := questionsStore.GetByID(ctx, question.ID)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.Title != "Updated title" || updated.Score != 10 || !updated.IsClosed {
		t.Fatalf("update not applied: %#v", updated)
	}

	if err := questionsStore.Delete(ctx, question.ID); err != nil {
		t.Fatalf("delete question: %v", err)
	}
	if _, err := questionsStore.GetByID(ctx, question.ID); err != questions.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestQuestionsStoreListSorting(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	questionsStore := NewQuestionsStore(db)
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

	questionsList := []*questions.Question{
		{
			ID:          "q-1",
			AuthorID:    account.ID,
			Title:       "Q1",
			Body:        "Body1",
			Score:       1,
			AnswerCount: 2,
			CreatedAt:   mustTime(2024, time.January, 1, 9, 0, 0),
			UpdatedAt:   mustTime(2024, time.January, 1, 9, 0, 0),
		},
		{
			ID:          "q-2",
			AuthorID:    account.ID,
			Title:       "Q2",
			Body:        "Body2",
			Score:       5,
			AnswerCount: 0,
			CreatedAt:   mustTime(2024, time.January, 2, 9, 0, 0),
			UpdatedAt:   mustTime(2024, time.January, 2, 9, 0, 0),
		},
		{
			ID:          "q-3",
			AuthorID:    account.ID,
			Title:       "Q3",
			Body:        "Body3",
			Score:       3,
			AnswerCount: 1,
			CreatedAt:   mustTime(2024, time.January, 3, 9, 0, 0),
			UpdatedAt:   mustTime(2024, time.January, 3, 11, 0, 0),
		},
	}

	for _, q := range questionsList {
		if err := questionsStore.Create(ctx, q); err != nil {
			t.Fatalf("create question: %v", err)
		}
	}

	newest, err := questionsStore.List(ctx, questions.ListOpts{SortBy: questions.SortNewest})
	if err != nil {
		t.Fatalf("list newest: %v", err)
	}
	if len(newest) != 3 || newest[0].ID != "q-3" {
		t.Fatalf("unexpected newest order: %#v", newest)
	}

	score, err := questionsStore.List(ctx, questions.ListOpts{SortBy: questions.SortScore})
	if err != nil {
		t.Fatalf("list score: %v", err)
	}
	if len(score) != 3 || score[0].ID != "q-2" {
		t.Fatalf("unexpected score order: %#v", score)
	}

	active, err := questionsStore.List(ctx, questions.ListOpts{SortBy: questions.SortActive})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 3 || active[0].ID != "q-3" {
		t.Fatalf("unexpected active order: %#v", active)
	}

	unanswered, err := questionsStore.List(ctx, questions.ListOpts{SortBy: questions.SortUnanswered})
	if err != nil {
		t.Fatalf("list unanswered: %v", err)
	}
	if len(unanswered) != 3 || unanswered[0].ID != "q-2" {
		t.Fatalf("unexpected unanswered order: %#v", unanswered)
	}
}

func TestQuestionsStoreQueriesAndTags(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	questionsStore := NewQuestionsStore(db)
	tagsStore := NewTagsStore(db)
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

	tagGo := &tags.Tag{ID: "t-1", Name: "go", CreatedAt: mustTime(2024, time.January, 1, 11, 0, 0)}
	tagSql := &tags.Tag{ID: "t-2", Name: "sql", CreatedAt: mustTime(2024, time.January, 1, 12, 0, 0)}
	if err := tagsStore.Create(ctx, tagGo); err != nil {
		t.Fatalf("create tag go: %v", err)
	}
	if err := tagsStore.Create(ctx, tagSql); err != nil {
		t.Fatalf("create tag sql: %v", err)
	}

	q1 := &questions.Question{
		ID:        "q-1",
		AuthorID:  account.ID,
		Title:     "Go question",
		Body:      "Body",
		Score:     3,
		CreatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
		UpdatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
	}
	q2 := &questions.Question{
		ID:        "q-2",
		AuthorID:  account.ID,
		Title:     "SQL question",
		Body:      "Body",
		Score:     1,
		CreatedAt: mustTime(2024, time.January, 3, 10, 0, 0),
		UpdatedAt: mustTime(2024, time.January, 3, 10, 0, 0),
	}
	if err := questionsStore.Create(ctx, q1); err != nil {
		t.Fatalf("create question 1: %v", err)
	}
	if err := questionsStore.Create(ctx, q2); err != nil {
		t.Fatalf("create question 2: %v", err)
	}

	if err := questionsStore.SetTags(ctx, q1.ID, []string{"go", "sql", "missing"}); err != nil {
		t.Fatalf("set tags: %v", err)
	}

	byTag, err := questionsStore.ListByTag(ctx, "go", questions.ListOpts{})
	if err != nil {
		t.Fatalf("list by tag: %v", err)
	}
	if len(byTag) != 1 || byTag[0].ID != q1.ID {
		t.Fatalf("unexpected tag results: %#v", byTag)
	}

	byAuthor, err := questionsStore.ListByAuthor(ctx, account.ID, questions.ListOpts{})
	if err != nil {
		t.Fatalf("list by author: %v", err)
	}
	if len(byAuthor) != 2 {
		t.Fatalf("unexpected author results: %#v", byAuthor)
	}

	found, err := questionsStore.Search(ctx, "go", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 1 || found[0].ID != q1.ID {
		t.Fatalf("unexpected search results: %#v", found)
	}

	tagList, err := questionsStore.GetTags(ctx, q1.ID)
	if err != nil {
		t.Fatalf("get tags: %v", err)
	}
	if len(tagList) != 2 || tagList[0].Name != "go" {
		t.Fatalf("unexpected tags: %#v", tagList)
	}
}

func TestQuestionsStoreStateUpdates(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	questionsStore := NewQuestionsStore(db)
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
		Title:     "Q1",
		Body:      "Body",
		CreatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
		UpdatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
	}
	if err := questionsStore.Create(ctx, question); err != nil {
		t.Fatalf("create question: %v", err)
	}

	if err := questionsStore.IncrementViews(ctx, question.ID); err != nil {
		t.Fatalf("increment views: %v", err)
	}

	if err := questionsStore.SetAcceptedAnswer(ctx, question.ID, "a-1"); err != nil {
		t.Fatalf("set accepted: %v", err)
	}
	if err := questionsStore.SetClosed(ctx, question.ID, true, "duplicate"); err != nil {
		t.Fatalf("set closed: %v", err)
	}
	if err := questionsStore.UpdateStats(ctx, question.ID, 2, 1, 1); err != nil {
		t.Fatalf("update stats: %v", err)
	}
	if err := questionsStore.UpdateScore(ctx, question.ID, 5); err != nil {
		t.Fatalf("update score: %v", err)
	}

	updated, err := questionsStore.GetByID(ctx, question.ID)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.ViewCount != 1 || updated.AcceptedAnswerID != "a-1" || !updated.IsClosed {
		t.Fatalf("state not updated: %#v", updated)
	}
	if updated.AnswerCount != 2 || updated.CommentCount != 1 || updated.FavoriteCount != 1 {
		t.Fatalf("counts not updated: %#v", updated)
	}
	if updated.Score != 5 {
		t.Fatalf("score not updated: %#v", updated)
	}
}
