package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/forum/feature/votes"
)

func TestVotesStore_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")

	vote := &votes.Vote{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: votes.TargetThread,
		TargetID:   newTestID(),
		Value:      1,
		CreatedAt:  testTime(),
		UpdatedAt:  testTime(),
	}

	if err := store.Votes().Create(ctx, vote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify
	got, err := store.Votes().GetByTarget(ctx, vote.AccountID, vote.TargetType, vote.TargetID)
	if err != nil {
		t.Fatalf("GetByTarget failed: %v", err)
	}

	if got.Value != vote.Value {
		t.Errorf("Value: got %d, want %d", got.Value, vote.Value)
	}
}

func TestVotesStore_GetByTarget(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	targetID := newTestID()

	vote := &votes.Vote{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: votes.TargetThread,
		TargetID:   targetID,
		Value:      1,
		CreatedAt:  testTime(),
		UpdatedAt:  testTime(),
	}

	if err := store.Votes().Create(ctx, vote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Votes().GetByTarget(ctx, author.ID, votes.TargetThread, targetID)
	if err != nil {
		t.Fatalf("GetByTarget failed: %v", err)
	}

	if got.ID != vote.ID {
		t.Errorf("ID: got %q, want %q", got.ID, vote.ID)
	}
}

func TestVotesStore_GetByTarget_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Votes().GetByTarget(ctx, "account", "thread", "nonexistent")
	if err != votes.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestVotesStore_Update(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	targetID := newTestID()

	vote := &votes.Vote{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: votes.TargetThread,
		TargetID:   targetID,
		Value:      1, // Upvote
		CreatedAt:  testTime(),
		UpdatedAt:  testTime(),
	}

	if err := store.Votes().Create(ctx, vote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Change to downvote
	vote.Value = -1
	vote.UpdatedAt = testTime()

	if err := store.Votes().Update(ctx, vote); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, err := store.Votes().GetByTarget(ctx, author.ID, votes.TargetThread, targetID)
	if err != nil {
		t.Fatalf("GetByTarget failed: %v", err)
	}

	if got.Value != -1 {
		t.Errorf("Value: got %d, want -1", got.Value)
	}
}

func TestVotesStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	targetID := newTestID()

	vote := &votes.Vote{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: votes.TargetThread,
		TargetID:   targetID,
		Value:      1,
		CreatedAt:  testTime(),
		UpdatedAt:  testTime(),
	}

	if err := store.Votes().Create(ctx, vote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.Votes().Delete(ctx, author.ID, votes.TargetThread, targetID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Votes().GetByTarget(ctx, author.ID, votes.TargetThread, targetID)
	if err != votes.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestVotesStore_CountByTarget(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	targetID := newTestID()

	// Create multiple votes
	for i := 0; i < 5; i++ {
		author := createTestAccount(t, store, "voter"+string(rune('a'+i)))
		value := 1
		if i >= 3 { // Last 2 are downvotes
			value = -1
		}

		vote := &votes.Vote{
			ID:         newTestID(),
			AccountID:  author.ID,
			TargetType: votes.TargetThread,
			TargetID:   targetID,
			Value:      value,
			CreatedAt:  testTime(),
			UpdatedAt:  testTime(),
		}

		if err := store.Votes().Create(ctx, vote); err != nil {
			t.Fatalf("Create vote %d failed: %v", i, err)
		}
	}

	up, down, err := store.Votes().CountByTarget(ctx, votes.TargetThread, targetID)
	if err != nil {
		t.Fatalf("CountByTarget failed: %v", err)
	}

	if up != 3 {
		t.Errorf("Up: got %d, want 3", up)
	}
	if down != 2 {
		t.Errorf("Down: got %d, want 2", down)
	}
}

func TestVotesStore_VoteOnComment(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	commentID := newTestID()

	vote := &votes.Vote{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: votes.TargetComment,
		TargetID:   commentID,
		Value:      1,
		CreatedAt:  testTime(),
		UpdatedAt:  testTime(),
	}

	if err := store.Votes().Create(ctx, vote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Votes().GetByTarget(ctx, author.ID, votes.TargetComment, commentID)
	if err != nil {
		t.Fatalf("GetByTarget failed: %v", err)
	}

	if got.TargetType != votes.TargetComment {
		t.Errorf("TargetType: got %q, want %q", got.TargetType, votes.TargetComment)
	}
}

func TestVotesStore_ChangeVote(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	targetID := newTestID()

	// Initial upvote
	vote := &votes.Vote{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: votes.TargetThread,
		TargetID:   targetID,
		Value:      1,
		CreatedAt:  testTime(),
		UpdatedAt:  testTime(),
	}

	if err := store.Votes().Create(ctx, vote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Check initial count
	up, down, _ := store.Votes().CountByTarget(ctx, votes.TargetThread, targetID)
	if up != 1 || down != 0 {
		t.Errorf("Initial: up=%d down=%d, want up=1 down=0", up, down)
	}

	// Change to downvote
	vote.Value = -1
	if err := store.Votes().Update(ctx, vote); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Check new count
	up, down, _ = store.Votes().CountByTarget(ctx, votes.TargetThread, targetID)
	if up != 0 || down != 1 {
		t.Errorf("After change: up=%d down=%d, want up=0 down=1", up, down)
	}
}

func TestVotesStore_MultipleTargets(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")

	// Vote on multiple threads
	targetIDs := []string{newTestID(), newTestID(), newTestID()}
	for i, targetID := range targetIDs {
		vote := &votes.Vote{
			ID:         newTestID(),
			AccountID:  author.ID,
			TargetType: votes.TargetThread,
			TargetID:   targetID,
			Value:      1,
			CreatedAt:  testTime(),
			UpdatedAt:  testTime(),
		}

		if err := store.Votes().Create(ctx, vote); err != nil {
			t.Fatalf("Create vote %d failed: %v", i, err)
		}
	}

	// Each target should have 1 upvote
	for i, targetID := range targetIDs {
		up, down, err := store.Votes().CountByTarget(ctx, votes.TargetThread, targetID)
		if err != nil {
			t.Fatalf("CountByTarget %d failed: %v", i, err)
		}
		if up != 1 || down != 0 {
			t.Errorf("Target %d: up=%d down=%d, want up=1 down=0", i, up, down)
		}
	}
}
