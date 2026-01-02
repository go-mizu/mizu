package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/answers"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
)

func TestAnswersStoreCRUDAndList(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	questionsStore := NewQuestionsStore(db)
	answersStore := NewAnswersStore(db)
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

	answersList := []*answers.Answer{
		{
			ID:         "a-1",
			QuestionID: question.ID,
			AuthorID:   account.ID,
			Body:       "First",
			Score:      2,
			IsAccepted: false,
			CreatedAt:  mustTime(2024, time.January, 3, 10, 0, 0),
			UpdatedAt:  mustTime(2024, time.January, 3, 10, 0, 0),
		},
		{
			ID:         "a-2",
			QuestionID: question.ID,
			AuthorID:   account.ID,
			Body:       "Second",
			Score:      5,
			IsAccepted: true,
			CreatedAt:  mustTime(2024, time.January, 3, 9, 0, 0),
			UpdatedAt:  mustTime(2024, time.January, 3, 9, 0, 0),
		},
		{
			ID:         "a-3",
			QuestionID: question.ID,
			AuthorID:   account.ID,
			Body:       "Third",
			Score:      1,
			IsAccepted: false,
			CreatedAt:  mustTime(2024, time.January, 3, 8, 0, 0),
			UpdatedAt:  mustTime(2024, time.January, 3, 8, 0, 0),
		},
	}

	for _, answer := range answersList {
		if err := answersStore.Create(ctx, answer); err != nil {
			t.Fatalf("create answer: %v", err)
		}
	}

	got, err := answersStore.GetByID(ctx, "a-1")
	if err != nil {
		t.Fatalf("get answer: %v", err)
	}
	if got.Body != "First" {
		t.Fatalf("unexpected answer: %#v", got)
	}

	listed, err := answersStore.ListByQuestion(ctx, question.ID, answers.ListOpts{})
	if err != nil {
		t.Fatalf("list answers: %v", err)
	}
	if len(listed) != 3 || listed[0].ID != "a-2" {
		t.Fatalf("unexpected answer order: %#v", listed)
	}

	update := &answers.Answer{
		ID:         "a-1",
		QuestionID: question.ID,
		AuthorID:   account.ID,
		Body:       "Updated",
		BodyHTML:   "<p>Updated</p>",
		Score:      4,
		IsAccepted: true,
		UpdatedAt:  mustTime(2024, time.January, 4, 10, 0, 0),
	}
	if err := answersStore.Update(ctx, update); err != nil {
		t.Fatalf("update answer: %v", err)
	}

	updated, err := answersStore.GetByID(ctx, "a-1")
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.Body != "Updated" || !updated.IsAccepted {
		t.Fatalf("update not applied: %#v", updated)
	}

	if err := answersStore.SetAccepted(ctx, "a-3", true); err != nil {
		t.Fatalf("set accepted: %v", err)
	}
	if err := answersStore.UpdateScore(ctx, "a-3", 2); err != nil {
		t.Fatalf("update score: %v", err)
	}

	accepted, err := answersStore.GetByID(ctx, "a-3")
	if err != nil {
		t.Fatalf("get accepted: %v", err)
	}
	if !accepted.IsAccepted || accepted.Score != 3 {
		t.Fatalf("expected accepted score update: %#v", accepted)
	}

	if err := answersStore.Delete(ctx, "a-2"); err != nil {
		t.Fatalf("delete answer: %v", err)
	}
	if _, err := answersStore.GetByID(ctx, "a-2"); err != answers.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}
