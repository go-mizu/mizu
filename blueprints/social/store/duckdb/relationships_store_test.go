package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/social/feature/relationships"
)

func TestRelationshipsStore_InsertFollow(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert follow
	follow := &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user1.ID,
		FollowingID: user2.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	}

	if err := store.InsertFollow(ctx, follow); err != nil {
		t.Fatalf("InsertFollow failed: %v", err)
	}

	// Verify
	exists, err := store.ExistsFollow(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("ExistsFollow failed: %v", err)
	}
	if !exists {
		t.Error("ExistsFollow: got false, want true")
	}
}

func TestRelationshipsStore_DeleteFollow(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert follow
	follow := &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user1.ID,
		FollowingID: user2.ID,
		CreatedAt:   testTime(),
	}
	store.InsertFollow(ctx, follow)

	// Delete follow
	if err := store.DeleteFollow(ctx, user1.ID, user2.ID); err != nil {
		t.Fatalf("DeleteFollow failed: %v", err)
	}

	// Verify
	exists, _ := store.ExistsFollow(ctx, user1.ID, user2.ID)
	if exists {
		t.Error("ExistsFollow after delete: got true, want false")
	}
}

func TestRelationshipsStore_GetFollow(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert follow
	follow := &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user1.ID,
		FollowingID: user2.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	}
	store.InsertFollow(ctx, follow)

	// Get follow
	got, err := store.GetFollow(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("GetFollow failed: %v", err)
	}

	if got.FollowerID != user1.ID {
		t.Errorf("FollowerID: got %q, want %q", got.FollowerID, user1.ID)
	}
	if got.FollowingID != user2.ID {
		t.Errorf("FollowingID: got %q, want %q", got.FollowingID, user2.ID)
	}
}

func TestRelationshipsStore_GetFollow_NotFound(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	store := NewRelationshipsStore(db)

	_, err := store.GetFollow(ctx, "nonexistent1", "nonexistent2")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestRelationshipsStore_SetFollowPending(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert pending follow
	follow := &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user1.ID,
		FollowingID: user2.ID,
		Pending:     true,
		CreatedAt:   testTime(),
	}
	store.InsertFollow(ctx, follow)

	// Set not pending
	if err := store.SetFollowPending(ctx, user1.ID, user2.ID, false); err != nil {
		t.Fatalf("SetFollowPending failed: %v", err)
	}

	// Verify
	got, _ := store.GetFollow(ctx, user1.ID, user2.ID)
	if got.Pending {
		t.Error("Pending: got true, want false")
	}
}

func TestRelationshipsStore_GetFollowers(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "user")
	follower1 := createTestAccount(t, db, "follower1")
	follower2 := createTestAccount(t, db, "follower2")
	follower3 := createTestAccount(t, db, "follower3")

	store := NewRelationshipsStore(db)

	// Add followers
	for _, follower := range []string{follower1.ID, follower2.ID, follower3.ID} {
		store.InsertFollow(ctx, &relationships.Follow{
			ID:          newTestID(),
			FollowerID:  follower,
			FollowingID: user.ID,
			Pending:     false,
			CreatedAt:   testTime(),
		})
	}

	// Get followers
	followers, err := store.GetFollowers(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetFollowers failed: %v", err)
	}

	if len(followers) != 3 {
		t.Errorf("GetFollowers count: got %d, want 3", len(followers))
	}
}

func TestRelationshipsStore_GetFollowing(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "user")
	following1 := createTestAccount(t, db, "following1")
	following2 := createTestAccount(t, db, "following2")

	store := NewRelationshipsStore(db)

	// Add following
	for _, following := range []string{following1.ID, following2.ID} {
		store.InsertFollow(ctx, &relationships.Follow{
			ID:          newTestID(),
			FollowerID:  user.ID,
			FollowingID: following,
			Pending:     false,
			CreatedAt:   testTime(),
		})
	}

	// Get following
	followings, err := store.GetFollowing(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetFollowing failed: %v", err)
	}

	if len(followings) != 2 {
		t.Errorf("GetFollowing count: got %d, want 2", len(followings))
	}
}

func TestRelationshipsStore_GetPendingFollowers(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "user")
	requester1 := createTestAccount(t, db, "requester1")
	requester2 := createTestAccount(t, db, "requester2")

	store := NewRelationshipsStore(db)

	// Add pending follow requests
	store.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  requester1.ID,
		FollowingID: user.ID,
		Pending:     true,
		CreatedAt:   testTime(),
	})
	store.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  requester2.ID,
		FollowingID: user.ID,
		Pending:     true,
		CreatedAt:   testTime(),
	})

	// Get pending followers
	pending, err := store.GetPendingFollowers(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetPendingFollowers failed: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("GetPendingFollowers count: got %d, want 2", len(pending))
	}
}

func TestRelationshipsStore_InsertBlock(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert block
	block := &relationships.Block{
		ID:        newTestID(),
		AccountID: user1.ID,
		TargetID:  user2.ID,
		CreatedAt: testTime(),
	}

	if err := store.InsertBlock(ctx, block); err != nil {
		t.Fatalf("InsertBlock failed: %v", err)
	}

	// Verify
	exists, err := store.ExistsBlock(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("ExistsBlock failed: %v", err)
	}
	if !exists {
		t.Error("ExistsBlock: got false, want true")
	}
}

func TestRelationshipsStore_DeleteBlock(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert block
	block := &relationships.Block{
		ID:        newTestID(),
		AccountID: user1.ID,
		TargetID:  user2.ID,
		CreatedAt: testTime(),
	}
	store.InsertBlock(ctx, block)

	// Delete block
	if err := store.DeleteBlock(ctx, user1.ID, user2.ID); err != nil {
		t.Fatalf("DeleteBlock failed: %v", err)
	}

	// Verify
	exists, _ := store.ExistsBlock(ctx, user1.ID, user2.ID)
	if exists {
		t.Error("ExistsBlock after delete: got true, want false")
	}
}

func TestRelationshipsStore_GetBlock(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert block
	block := &relationships.Block{
		ID:        newTestID(),
		AccountID: user1.ID,
		TargetID:  user2.ID,
		CreatedAt: testTime(),
	}
	store.InsertBlock(ctx, block)

	// Get block
	got, err := store.GetBlock(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("GetBlock failed: %v", err)
	}

	if got.AccountID != user1.ID {
		t.Errorf("AccountID: got %q, want %q", got.AccountID, user1.ID)
	}
	if got.TargetID != user2.ID {
		t.Errorf("TargetID: got %q, want %q", got.TargetID, user2.ID)
	}
}

func TestRelationshipsStore_GetBlocks(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "user")
	blocked1 := createTestAccount(t, db, "blocked1")
	blocked2 := createTestAccount(t, db, "blocked2")

	store := NewRelationshipsStore(db)

	// Add blocks
	for _, blocked := range []string{blocked1.ID, blocked2.ID} {
		store.InsertBlock(ctx, &relationships.Block{
			ID:        newTestID(),
			AccountID: user.ID,
			TargetID:  blocked,
			CreatedAt: testTime(),
		})
	}

	// Get blocks
	blocks, err := store.GetBlocks(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetBlocks failed: %v", err)
	}

	if len(blocks) != 2 {
		t.Errorf("GetBlocks count: got %d, want 2", len(blocks))
	}
}

func TestRelationshipsStore_ExistsBlockEither(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")
	user3 := createTestAccount(t, db, "user3")

	store := NewRelationshipsStore(db)

	// user1 blocks user2
	store.InsertBlock(ctx, &relationships.Block{
		ID:        newTestID(),
		AccountID: user1.ID,
		TargetID:  user2.ID,
		CreatedAt: testTime(),
	})

	// Check user1 -> user2 (block exists)
	exists1, err := store.ExistsBlockEither(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("ExistsBlockEither failed: %v", err)
	}
	if !exists1 {
		t.Error("ExistsBlockEither (user1, user2): got false, want true")
	}

	// Check user2 -> user1 (should also return true - reverse direction)
	exists2, err := store.ExistsBlockEither(ctx, user2.ID, user1.ID)
	if err != nil {
		t.Fatalf("ExistsBlockEither (reverse) failed: %v", err)
	}
	if !exists2 {
		t.Error("ExistsBlockEither (user2, user1): got false, want true")
	}

	// Check user1 -> user3 (no block)
	exists3, err := store.ExistsBlockEither(ctx, user1.ID, user3.ID)
	if err != nil {
		t.Fatalf("ExistsBlockEither (no block) failed: %v", err)
	}
	if exists3 {
		t.Error("ExistsBlockEither (user1, user3): got true, want false")
	}
}

func TestRelationshipsStore_InsertMute(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert mute
	mute := &relationships.Mute{
		ID:                newTestID(),
		AccountID:         user1.ID,
		TargetID:          user2.ID,
		HideNotifications: true,
		CreatedAt:         testTime(),
	}

	if err := store.InsertMute(ctx, mute); err != nil {
		t.Fatalf("InsertMute failed: %v", err)
	}

	// Verify
	exists, err := store.ExistsMute(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("ExistsMute failed: %v", err)
	}
	if !exists {
		t.Error("ExistsMute: got false, want true")
	}
}

func TestRelationshipsStore_InsertMute_WithExpiration(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	expiresAt := time.Now().Add(24 * time.Hour)

	// Insert mute with expiration
	mute := &relationships.Mute{
		ID:                newTestID(),
		AccountID:         user1.ID,
		TargetID:          user2.ID,
		HideNotifications: true,
		ExpiresAt:         &expiresAt,
		CreatedAt:         testTime(),
	}

	if err := store.InsertMute(ctx, mute); err != nil {
		t.Fatalf("InsertMute failed: %v", err)
	}

	// Verify
	got, err := store.GetMute(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("GetMute failed: %v", err)
	}

	if got.ExpiresAt == nil {
		t.Error("ExpiresAt: got nil, want non-nil")
	}
}

func TestRelationshipsStore_DeleteMute(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert mute
	mute := &relationships.Mute{
		ID:        newTestID(),
		AccountID: user1.ID,
		TargetID:  user2.ID,
		CreatedAt: testTime(),
	}
	store.InsertMute(ctx, mute)

	// Delete mute
	if err := store.DeleteMute(ctx, user1.ID, user2.ID); err != nil {
		t.Fatalf("DeleteMute failed: %v", err)
	}

	// Verify
	exists, _ := store.ExistsMute(ctx, user1.ID, user2.ID)
	if exists {
		t.Error("ExistsMute after delete: got true, want false")
	}
}

func TestRelationshipsStore_GetMutes(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "user")
	muted1 := createTestAccount(t, db, "muted1")
	muted2 := createTestAccount(t, db, "muted2")

	store := NewRelationshipsStore(db)

	// Add mutes
	for _, muted := range []string{muted1.ID, muted2.ID} {
		store.InsertMute(ctx, &relationships.Mute{
			ID:        newTestID(),
			AccountID: user.ID,
			TargetID:  muted,
			CreatedAt: testTime(),
		})
	}

	// Get mutes
	mutes, err := store.GetMutes(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetMutes failed: %v", err)
	}

	if len(mutes) != 2 {
		t.Errorf("GetMutes count: got %d, want 2", len(mutes))
	}
}

func TestRelationshipsStore_ExistsMute_Expired(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Insert expired mute
	expired := time.Now().Add(-1 * time.Hour)
	mute := &relationships.Mute{
		ID:        newTestID(),
		AccountID: user1.ID,
		TargetID:  user2.ID,
		ExpiresAt: &expired,
		CreatedAt: testTime(),
	}
	store.InsertMute(ctx, mute)

	// Expired mute should not exist
	exists, err := store.ExistsMute(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("ExistsMute failed: %v", err)
	}
	if exists {
		t.Error("ExistsMute (expired): got true, want false")
	}
}

func TestRelationshipsStore_GetRelationship(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// Initially no relationship
	rel, err := store.GetRelationship(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("GetRelationship failed: %v", err)
	}

	if rel.Following {
		t.Error("Following: got true, want false")
	}
	if rel.FollowedBy {
		t.Error("FollowedBy: got true, want false")
	}

	// user1 follows user2
	store.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user1.ID,
		FollowingID: user2.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	})

	// user2 follows user1
	store.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user2.ID,
		FollowingID: user1.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	})

	// user1 mutes user2
	store.InsertMute(ctx, &relationships.Mute{
		ID:        newTestID(),
		AccountID: user1.ID,
		TargetID:  user2.ID,
		CreatedAt: testTime(),
	})

	// Get relationship
	rel2, err := store.GetRelationship(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("GetRelationship (2) failed: %v", err)
	}

	if !rel2.Following {
		t.Error("Following: got false, want true")
	}
	if !rel2.FollowedBy {
		t.Error("FollowedBy: got false, want true")
	}
	if !rel2.Muting {
		t.Error("Muting: got false, want true")
	}
}

func TestRelationshipsStore_GetRelationship_Blocking(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// user1 blocks user2
	store.InsertBlock(ctx, &relationships.Block{
		ID:        newTestID(),
		AccountID: user1.ID,
		TargetID:  user2.ID,
		CreatedAt: testTime(),
	})

	// user2 blocks user1
	store.InsertBlock(ctx, &relationships.Block{
		ID:        newTestID(),
		AccountID: user2.ID,
		TargetID:  user1.ID,
		CreatedAt: testTime(),
	})

	// Get relationship from user1's perspective
	rel, err := store.GetRelationship(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("GetRelationship failed: %v", err)
	}

	if !rel.Blocking {
		t.Error("Blocking: got false, want true")
	}
	if !rel.BlockedBy {
		t.Error("BlockedBy: got false, want true")
	}
}

func TestRelationshipsStore_GetRelationship_Requested(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")

	store := NewRelationshipsStore(db)

	// user1 requests to follow user2 (pending)
	store.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user1.ID,
		FollowingID: user2.ID,
		Pending:     true,
		CreatedAt:   testTime(),
	})

	// Get relationship
	rel, err := store.GetRelationship(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("GetRelationship failed: %v", err)
	}

	if rel.Following {
		t.Error("Following: got true, want false")
	}
	if !rel.Requested {
		t.Error("Requested: got false, want true")
	}
}
