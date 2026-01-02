package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/votes"
)

func TestVotesStoreUpsert(t *testing.T) {
	db := newTestDB(t)
	store := NewVotesStore(db)
	ctx := context.Background()

	missing, err := store.Get(ctx, "voter", votes.TargetQuestion, "q-1")
	if err != nil {
		t.Fatalf("get missing: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil, got %#v", missing)
	}

	vote := &votes.Vote{
		ID:         "v-1",
		VoterID:    "voter",
		TargetType: votes.TargetQuestion,
		TargetID:   "q-1",
		Value:      1,
		CreatedAt:  mustTime(2024, time.January, 1, 10, 0, 0),
		UpdatedAt:  mustTime(2024, time.January, 1, 10, 0, 0),
	}
	stored, err := store.Upsert(ctx, vote)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if stored.Value != 1 {
		t.Fatalf("unexpected stored vote: %#v", stored)
	}

	vote.Value = -1
	vote.UpdatedAt = mustTime(2024, time.January, 1, 11, 0, 0)
	stored, err = store.Upsert(ctx, vote)
	if err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if stored.Value != -1 {
		t.Fatalf("expected updated vote, got %#v", stored)
	}
}
