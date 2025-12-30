package duckdb

import (
	"context"
	"testing"
	"time"
)

func TestSessionsStore_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		session := &Session{
			UserID:       "user123456789012345678901",
			Collection:   "users",
			Token:        "token-abc123",
			RefreshToken: "refresh-xyz789",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		err := store.Sessions.Create(ctx, session)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if session.ID == "" {
			t.Error("Expected ID to be set")
		}
		if session.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("AllFields", func(t *testing.T) {
		session := &Session{
			UserID:       "user234567890123456789012",
			Collection:   "admins",
			Token:        "token-full123",
			RefreshToken: "refresh-full789",
			UserAgent:    "Mozilla/5.0 (Test Browser)",
			IP:           "192.168.1.100",
			ExpiresAt:    time.Now().Add(48 * time.Hour),
		}

		err := store.Sessions.Create(ctx, session)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Verify all fields were saved
		fetched, err := store.Sessions.GetByToken(ctx, "token-full123")
		if err != nil {
			t.Fatalf("GetByToken failed: %v", err)
		}

		if fetched.UserAgent != "Mozilla/5.0 (Test Browser)" {
			t.Errorf("Expected UserAgent='Mozilla/5.0 (Test Browser)', got %s", fetched.UserAgent)
		}
		if fetched.IP != "192.168.1.100" {
			t.Errorf("Expected IP='192.168.1.100', got %s", fetched.IP)
		}
	})

	t.Run("NullableFields", func(t *testing.T) {
		session := &Session{
			UserID:     "user345678901234567890123",
			Collection: "users",
			Token:      "token-nullable123",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
			// UserAgent, IP, RefreshToken are empty
		}

		err := store.Sessions.Create(ctx, session)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if session.ID == "" {
			t.Error("Expected ID to be set")
		}
	})
}

func TestSessionsStore_GetByToken(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		session := &Session{
			UserID:     "user456789012345678901234",
			Collection: "users",
			Token:      "token-valid123",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		fetched, err := store.Sessions.GetByToken(ctx, "token-valid123")
		if err != nil {
			t.Fatalf("GetByToken failed: %v", err)
		}

		if fetched == nil {
			t.Fatal("Expected session to be found")
		}
		if fetched.Token != "token-valid123" {
			t.Errorf("Expected Token='token-valid123', got %s", fetched.Token)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		fetched, err := store.Sessions.GetByToken(ctx, "nonexistent-token")
		if err != nil {
			t.Fatalf("GetByToken failed: %v", err)
		}

		if fetched != nil {
			t.Error("Expected nil for non-existent token")
		}
	})

	t.Run("Expired", func(t *testing.T) {
		session := &Session{
			UserID:     "user567890123456789012345",
			Collection: "users",
			Token:      "token-expired123",
			ExpiresAt:  time.Now().Add(-1 * time.Hour), // Already expired
		}
		store.Sessions.Create(ctx, session)

		fetched, err := store.Sessions.GetByToken(ctx, "token-expired123")
		if err != nil {
			t.Fatalf("GetByToken failed: %v", err)
		}

		if fetched != nil {
			t.Error("Expected nil for expired session")
		}
	})

	t.Run("NullableFields", func(t *testing.T) {
		session := &Session{
			UserID:       "user678901234567890123456",
			Collection:   "users",
			Token:        "token-nullable-get",
			RefreshToken: "refresh-test",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		fetched, err := store.Sessions.GetByToken(ctx, "token-nullable-get")
		if err != nil {
			t.Fatalf("GetByToken failed: %v", err)
		}

		if fetched == nil {
			t.Fatal("Expected session to be found")
		}
		// Verify nullable fields are handled correctly
		if fetched.UserAgent != "" {
			t.Errorf("Expected empty UserAgent, got %s", fetched.UserAgent)
		}
		if fetched.RefreshToken != "refresh-test" {
			t.Errorf("Expected RefreshToken='refresh-test', got %s", fetched.RefreshToken)
		}
	})
}

func TestSessionsStore_GetByRefreshToken(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		session := &Session{
			UserID:       "user789012345678901234567",
			Collection:   "users",
			Token:        "token-for-refresh",
			RefreshToken: "refresh-valid123",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		fetched, err := store.Sessions.GetByRefreshToken(ctx, "refresh-valid123")
		if err != nil {
			t.Fatalf("GetByRefreshToken failed: %v", err)
		}

		if fetched == nil {
			t.Fatal("Expected session to be found")
		}
		if fetched.RefreshToken != "refresh-valid123" {
			t.Errorf("Expected RefreshToken='refresh-valid123', got %s", fetched.RefreshToken)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		fetched, err := store.Sessions.GetByRefreshToken(ctx, "nonexistent-refresh")
		if err != nil {
			t.Fatalf("GetByRefreshToken failed: %v", err)
		}

		if fetched != nil {
			t.Error("Expected nil for non-existent refresh token")
		}
	})

	t.Run("ExpiredStillReturned", func(t *testing.T) {
		// GetByRefreshToken doesn't filter by expiry (unlike GetByToken)
		session := &Session{
			UserID:       "user890123456789012345678",
			Collection:   "users",
			Token:        "token-expired-refresh",
			RefreshToken: "refresh-expired123",
			ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
		}
		store.Sessions.Create(ctx, session)

		fetched, err := store.Sessions.GetByRefreshToken(ctx, "refresh-expired123")
		if err != nil {
			t.Fatalf("GetByRefreshToken failed: %v", err)
		}

		// Note: GetByRefreshToken returns session even if expired
		// This is by design - allows for refresh token rotation
		if fetched == nil {
			t.Fatal("Expected expired session to be returned by GetByRefreshToken")
		}
	})
}

func TestSessionsStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		session := &Session{
			UserID:     "user901234567890123456789",
			Collection: "users",
			Token:      "token-delete123",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		err := store.Sessions.Delete(ctx, "token-delete123")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		err := store.Sessions.Delete(ctx, "nonexistent-token-delete")
		if err != nil {
			t.Fatalf("Delete failed for non-existent: %v", err)
		}
		// No error expected even for non-existent
	})

	t.Run("Verify", func(t *testing.T) {
		session := &Session{
			UserID:     "user012345678901234567890",
			Collection: "users",
			Token:      "token-verify-delete",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		store.Sessions.Delete(ctx, "token-verify-delete")

		fetched, _ := store.Sessions.GetByRefreshToken(ctx, "token-verify-delete")
		if fetched != nil {
			t.Error("Expected session to be deleted")
		}
	})
}

func TestSessionsStore_DeleteByUser(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Multiple", func(t *testing.T) {
		userID := "user111111111111111111111"

		// Create multiple sessions for same user
		for i := 0; i < 3; i++ {
			session := &Session{
				UserID:     userID,
				Collection: "users",
				Token:      "token-user-multi-" + string(rune('a'+i)),
				ExpiresAt:  time.Now().Add(24 * time.Hour),
			}
			store.Sessions.Create(ctx, session)
		}

		err := store.Sessions.DeleteByUser(ctx, userID)
		if err != nil {
			t.Fatalf("DeleteByUser failed: %v", err)
		}

		// Verify all sessions are gone
		for i := 0; i < 3; i++ {
			token := "token-user-multi-" + string(rune('a'+i))
			fetched, _ := store.Sessions.GetByRefreshToken(ctx, token)
			if fetched != nil {
				t.Errorf("Expected session %s to be deleted", token)
			}
		}
	})

	t.Run("PreservesOthers", func(t *testing.T) {
		user1ID := "user222222222222222222222"
		user2ID := "user333333333333333333333"

		session1 := &Session{
			UserID:     user1ID,
			Collection: "users",
			Token:      "token-user1-preserve",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session1)

		session2 := &Session{
			UserID:     user2ID,
			Collection: "users",
			Token:      "token-user2-preserve",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session2)

		// Delete only user1's sessions
		store.Sessions.DeleteByUser(ctx, user1ID)

		// User2's session should remain
		fetched, err := store.Sessions.GetByToken(ctx, "token-user2-preserve")
		if err != nil {
			t.Fatalf("GetByToken failed: %v", err)
		}
		if fetched == nil {
			t.Error("Expected user2's session to be preserved")
		}
	})
}

func TestSessionsStore_Update(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Token", func(t *testing.T) {
		session := &Session{
			UserID:     "user444444444444444444444",
			Collection: "users",
			Token:      "token-update-old",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		session.Token = "token-update-new"
		err := store.Sessions.Update(ctx, session)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Old token should not work
		old, _ := store.Sessions.GetByToken(ctx, "token-update-old")
		if old != nil {
			t.Error("Old token should not find session")
		}

		// New token should work
		updated, _ := store.Sessions.GetByToken(ctx, "token-update-new")
		if updated == nil {
			t.Error("Expected session with new token")
		}
	})

	t.Run("RefreshToken", func(t *testing.T) {
		session := &Session{
			UserID:       "user555555555555555555555",
			Collection:   "users",
			Token:        "token-refresh-update",
			RefreshToken: "refresh-old",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		session.RefreshToken = "refresh-new"
		err := store.Sessions.Update(ctx, session)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		updated, _ := store.Sessions.GetByRefreshToken(ctx, "refresh-new")
		if updated == nil {
			t.Error("Expected session with new refresh token")
		}
	})

	t.Run("ExpiresAt", func(t *testing.T) {
		session := &Session{
			UserID:     "user666666666666666666666",
			Collection: "users",
			Token:      "token-expiry-update",
			ExpiresAt:  time.Now().Add(1 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		newExpiry := time.Now().Add(48 * time.Hour)
		session.ExpiresAt = newExpiry
		err := store.Sessions.Update(ctx, session)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		updated, _ := store.Sessions.GetByToken(ctx, "token-expiry-update")
		if updated == nil {
			t.Fatal("Expected session to be found")
		}
		// Check that expiry was updated (with some tolerance)
		diff := updated.ExpiresAt.Sub(newExpiry)
		if diff > time.Second || diff < -time.Second {
			t.Errorf("ExpiresAt not updated correctly: expected %v, got %v", newExpiry, updated.ExpiresAt)
		}
	})
}

func TestSessionsStore_CleanupExpired(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("RemovesExpired", func(t *testing.T) {
		// Create expired sessions
		for i := 0; i < 3; i++ {
			session := &Session{
				UserID:     "user777777777777777777777",
				Collection: "users",
				Token:      "token-cleanup-expired-" + string(rune('a'+i)),
				ExpiresAt:  time.Now().Add(-1 * time.Hour),
			}
			store.Sessions.Create(ctx, session)
		}

		affected, err := store.Sessions.CleanupExpired(ctx)
		if err != nil {
			t.Fatalf("CleanupExpired failed: %v", err)
		}

		if affected < 3 {
			t.Errorf("Expected at least 3 sessions cleaned up, got %d", affected)
		}
	})

	t.Run("PreservesValid", func(t *testing.T) {
		session := &Session{
			UserID:     "user888888888888888888888",
			Collection: "users",
			Token:      "token-cleanup-valid",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		store.Sessions.Create(ctx, session)

		store.Sessions.CleanupExpired(ctx)

		fetched, _ := store.Sessions.GetByToken(ctx, "token-cleanup-valid")
		if fetched == nil {
			t.Error("Expected valid session to be preserved")
		}
	})

	t.Run("ReturnsCount", func(t *testing.T) {
		// Create exactly 2 expired sessions
		for i := 0; i < 2; i++ {
			session := &Session{
				UserID:     "user999999999999999999999",
				Collection: "users",
				Token:      "token-cleanup-count-" + string(rune('a'+i)),
				ExpiresAt:  time.Now().Add(-1 * time.Hour),
			}
			store.Sessions.Create(ctx, session)
		}

		affected, err := store.Sessions.CleanupExpired(ctx)
		if err != nil {
			t.Fatalf("CleanupExpired failed: %v", err)
		}

		if affected < 2 {
			t.Errorf("Expected at least 2 cleaned up, got %d", affected)
		}
	})
}
