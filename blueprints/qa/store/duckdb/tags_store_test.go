package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

func TestTagsStore(t *testing.T) {
	db := newTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tagOne := &tags.Tag{ID: "t-1", Name: "go", QuestionCount: 2, CreatedAt: mustTime(2024, time.January, 1, 10, 0, 0)}
	tagTwo := &tags.Tag{ID: "t-2", Name: "duckdb", QuestionCount: 5, CreatedAt: mustTime(2024, time.January, 2, 10, 0, 0)}
	if err := store.Create(ctx, tagOne); err != nil {
		t.Fatalf("create tag one: %v", err)
	}
	if err := store.Create(ctx, tagTwo); err != nil {
		t.Fatalf("create tag two: %v", err)
	}

	got, err := store.GetByName(ctx, "GO")
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if got.ID != tagOne.ID {
		t.Fatalf("unexpected tag: %#v", got)
	}

	listed, err := store.List(ctx, tags.ListOpts{})
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(listed) != 2 || listed[0].ID != tagTwo.ID {
		t.Fatalf("unexpected list: %#v", listed)
	}

	filtered, err := store.List(ctx, tags.ListOpts{Query: "go"})
	if err != nil {
		t.Fatalf("list tags query: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != tagOne.ID {
		t.Fatalf("unexpected query result: %#v", filtered)
	}

	if err := store.IncrementQuestionCount(ctx, "go", 3); err != nil {
		t.Fatalf("increment count: %v", err)
	}
	updated, err := store.GetByName(ctx, "go")
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.QuestionCount != 5 {
		t.Fatalf("expected updated count, got %#v", updated)
	}
}
