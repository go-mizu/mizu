package sqlite

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestUserStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid input", func(t *testing.T) {
		u := &store.User{
			Email:        "test@example.com",
			Name:         "Test User",
			PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$salt$hash",
			Role:         "user",
		}
		err := s.Users().Create(ctx, u)
		require.NoError(t, err)

		assertIDGenerated(t, u.ID)
		assert.False(t, u.CreatedAt.IsZero())
	})

	t.Run("duplicate email fails", func(t *testing.T) {
		email := "duplicate@example.com"

		u1 := &store.User{Email: email, Name: "User 1", PasswordHash: "hash", Role: "user"}
		u2 := &store.User{Email: email, Name: "User 2", PasswordHash: "hash", Role: "user"}

		require.NoError(t, s.Users().Create(ctx, u1))
		err := s.Users().Create(ctx, u2)
		require.Error(t, err) // Should fail with UNIQUE constraint
	})

	t.Run("admin role", func(t *testing.T) {
		u := &store.User{
			Email:        "admin@example.com",
			Name:         "Admin User",
			PasswordHash: "hash",
			Role:         "admin",
		}
		err := s.Users().Create(ctx, u)
		require.NoError(t, err)

		retrieved, _ := s.Users().GetByID(ctx, u.ID)
		assert.Equal(t, "admin", retrieved.Role)
	})
}

func TestUserStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists", func(t *testing.T) {
		u := createTestUser(t, s)

		retrieved, err := s.Users().GetByID(ctx, u.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, u.ID, retrieved.ID)
		assert.Equal(t, u.Email, retrieved.Email)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Users().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestUserStore_GetByEmail(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists", func(t *testing.T) {
		u := createTestUser(t, s)

		retrieved, err := s.Users().GetByEmail(ctx, u.Email)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, u.ID, retrieved.ID)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Users().GetByEmail(ctx, "nobody@example.com")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("case sensitive", func(t *testing.T) {
		u := &store.User{
			Email:        "CaseSensitive@Example.COM",
			Name:         "Test",
			PasswordHash: "hash",
			Role:         "user",
		}
		s.Users().Create(ctx, u)

		// Exact case should work
		retrieved, _ := s.Users().GetByEmail(ctx, "CaseSensitive@Example.COM")
		assert.NotNil(t, retrieved)
	})
}

func TestUserStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all users", func(t *testing.T) {
		createTestUser(t, s)
		createTestUser(t, s)

		users, err := s.Users().List(ctx)
		require.NoError(t, err)
		assert.Len(t, users, 2)
	})
}

func TestUserStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update name", func(t *testing.T) {
		u := createTestUser(t, s)

		u.Name = "Updated Name"
		err := s.Users().Update(ctx, u)
		require.NoError(t, err)

		retrieved, _ := s.Users().GetByID(ctx, u.ID)
		assert.Equal(t, "Updated Name", retrieved.Name)
	})

	t.Run("update password hash", func(t *testing.T) {
		u := createTestUser(t, s)

		u.PasswordHash = "new_hash_value"
		err := s.Users().Update(ctx, u)
		require.NoError(t, err)

		retrieved, _ := s.Users().GetByID(ctx, u.ID)
		assert.Equal(t, "new_hash_value", retrieved.PasswordHash)
	})

	t.Run("update role", func(t *testing.T) {
		u := createTestUser(t, s)

		u.Role = "admin"
		err := s.Users().Update(ctx, u)
		require.NoError(t, err)

		retrieved, _ := s.Users().GetByID(ctx, u.ID)
		assert.Equal(t, "admin", retrieved.Role)
	})
}

func TestUserStore_UpdateLastLogin(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("updates timestamp", func(t *testing.T) {
		u := createTestUser(t, s)
		assert.True(t, u.LastLogin.IsZero())

		err := s.Users().UpdateLastLogin(ctx, u.ID)
		require.NoError(t, err)

		retrieved, _ := s.Users().GetByID(ctx, u.ID)
		assert.False(t, retrieved.LastLogin.IsZero())
	})
}

func TestUserStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes user", func(t *testing.T) {
		u := createTestUser(t, s)

		err := s.Users().Delete(ctx, u.ID)
		require.NoError(t, err)

		retrieved, _ := s.Users().GetByID(ctx, u.ID)
		assert.Nil(t, retrieved)
	})

	t.Run("cascades to sessions", func(t *testing.T) {
		u := createTestUser(t, s)
		sess := createTestSession(t, s, u.ID)

		// Verify session exists
		retrieved, err := s.Users().GetSession(ctx, sess.Token)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Delete user
		err = s.Users().Delete(ctx, u.ID)
		require.NoError(t, err)

		// Verify session is gone
		retrieved, err = s.Users().GetSession(ctx, sess.Token)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

// Session tests

func TestSessionStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid session", func(t *testing.T) {
		u := createTestUser(t, s)

		sess := &store.Session{
			UserID:    u.ID,
			Token:     ulid.Make().String(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := s.Users().CreateSession(ctx, sess)
		require.NoError(t, err)
		assertIDGenerated(t, sess.ID)
	})

	t.Run("invalid user fails", func(t *testing.T) {
		sess := &store.Session{
			UserID:    "nonexistent",
			Token:     ulid.Make().String(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := s.Users().CreateSession(ctx, sess)
		require.Error(t, err)
	})
}

func TestSessionStore_GetSession(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid non-expired session", func(t *testing.T) {
		u := createTestUser(t, s)
		sess := createTestSession(t, s, u.ID)

		retrieved, err := s.Users().GetSession(ctx, sess.Token)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, sess.ID, retrieved.ID)
		assert.Equal(t, u.ID, retrieved.UserID)
	})

	t.Run("expired session returns nil", func(t *testing.T) {
		u := createTestUser(t, s)

		sess := &store.Session{
			UserID:    u.ID,
			Token:     ulid.Make().String(),
			ExpiresAt: time.Now().UTC().Add(-24 * time.Hour), // Clearly expired (24 hours ago in UTC)
		}
		s.Users().CreateSession(ctx, sess)

		retrieved, err := s.Users().GetSession(ctx, sess.Token)
		require.NoError(t, err)
		assert.Nil(t, retrieved) // Expired sessions should not be returned
	})

	t.Run("non-existent token returns nil", func(t *testing.T) {
		retrieved, err := s.Users().GetSession(ctx, "nonexistent_token")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestSessionStore_DeleteSession(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes session", func(t *testing.T) {
		u := createTestUser(t, s)
		sess := createTestSession(t, s, u.ID)

		err := s.Users().DeleteSession(ctx, sess.Token)
		require.NoError(t, err)

		retrieved, _ := s.Users().GetSession(ctx, sess.Token)
		assert.Nil(t, retrieved)
	})
}
