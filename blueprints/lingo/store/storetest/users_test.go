package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestUserStore_Create(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		users := s.Users()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		user := &store.User{
			ID:                uuid.New(),
			Email:             "test@example.com",
			Username:          "testuser",
			DisplayName:       "Test User",
			EncryptedPassword: string(hashedPassword),
			XPTotal:           0,
			Gems:              500,
			Hearts:            5,
			StreakDays:        0,
			DailyGoalMinutes:  10,
			CreatedAt:         time.Now(),
		}

		err := users.Create(ctx, user)
		assertNoError(t, err, "create user")

		// Verify user was created
		retrieved, err := users.GetByID(ctx, user.ID)
		assertNoError(t, err, "get user by id")
		assertEqual(t, user.Email, retrieved.Email, "email")
		assertEqual(t, user.Username, retrieved.Username, "username")
		assertEqual(t, 500, retrieved.Gems, "gems")
		assertEqual(t, 5, retrieved.Hearts, "hearts")
	})
}

func TestUserStore_GetByEmail(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		users := s.Users()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		user := &store.User{
			ID:                uuid.New(),
			Email:             "email@example.com",
			Username:          "emailuser",
			DisplayName:       "Email User",
			EncryptedPassword: string(hashedPassword),
			Gems:              500,
			Hearts:            5,
			DailyGoalMinutes:  10,
			CreatedAt:         time.Now(),
		}

		_ = users.Create(ctx, user)

		// Test get by email
		retrieved, err := users.GetByEmail(ctx, "email@example.com")
		assertNoError(t, err, "get user by email")
		assertEqual(t, user.ID, retrieved.ID, "user id")

		// Test non-existent email
		_, err = users.GetByEmail(ctx, "nonexistent@example.com")
		assertError(t, err, "get non-existent email")
	})
}

func TestUserStore_GetByUsername(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		users := s.Users()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		user := &store.User{
			ID:                uuid.New(),
			Email:             "username@example.com",
			Username:          "uniqueusername",
			DisplayName:       "Unique User",
			EncryptedPassword: string(hashedPassword),
			Gems:              500,
			Hearts:            5,
			DailyGoalMinutes:  10,
			CreatedAt:         time.Now(),
		}

		_ = users.Create(ctx, user)

		// Test get by username
		retrieved, err := users.GetByUsername(ctx, "uniqueusername")
		assertNoError(t, err, "get user by username")
		assertEqual(t, user.ID, retrieved.ID, "user id")

		// Test non-existent username
		_, err = users.GetByUsername(ctx, "nonexistent")
		assertError(t, err, "get non-existent username")
	})
}

func TestUserStore_Update(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		users := s.Users()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		user := &store.User{
			ID:                uuid.New(),
			Email:             "update@example.com",
			Username:          "updateuser",
			DisplayName:       "Update User",
			EncryptedPassword: string(hashedPassword),
			Gems:              500,
			Hearts:            5,
			DailyGoalMinutes:  10,
			CreatedAt:         time.Now(),
		}

		_ = users.Create(ctx, user)

		// Update display name
		user.DisplayName = "Updated Display Name"
		user.Bio = "New bio"
		err := users.Update(ctx, user)
		assertNoError(t, err, "update user")

		// Verify update
		retrieved, _ := users.GetByID(ctx, user.ID)
		assertEqual(t, "Updated Display Name", retrieved.DisplayName, "display name")
		assertEqual(t, "New bio", retrieved.Bio, "bio")
	})
}

func TestUserStore_UpdateXP(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		users := s.Users()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		user := &store.User{
			ID:                uuid.New(),
			Email:             "xp@example.com",
			Username:          "xpuser",
			DisplayName:       "XP User",
			EncryptedPassword: string(hashedPassword),
			XPTotal:           100,
			Gems:              500,
			Hearts:            5,
			DailyGoalMinutes:  10,
			CreatedAt:         time.Now(),
		}

		_ = users.Create(ctx, user)

		// Add XP
		err := users.UpdateXP(ctx, user.ID, 50)
		assertNoError(t, err, "update xp")

		// Verify XP
		retrieved, _ := users.GetByID(ctx, user.ID)
		assertEqual(t, int64(150), retrieved.XPTotal, "xp total")

		// Add more XP
		_ = users.UpdateXP(ctx, user.ID, 25)
		retrieved, _ = users.GetByID(ctx, user.ID)
		assertEqual(t, int64(175), retrieved.XPTotal, "xp total after second add")
	})
}

func TestUserStore_UpdateHearts(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		users := s.Users()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		user := &store.User{
			ID:                uuid.New(),
			Email:             "hearts@example.com",
			Username:          "heartsuser",
			DisplayName:       "Hearts User",
			EncryptedPassword: string(hashedPassword),
			Gems:              500,
			Hearts:            5,
			DailyGoalMinutes:  10,
			CreatedAt:         time.Now(),
		}

		_ = users.Create(ctx, user)

		// Reduce hearts
		err := users.UpdateHearts(ctx, user.ID, 3)
		assertNoError(t, err, "update hearts")

		// Verify hearts
		retrieved, _ := users.GetByID(ctx, user.ID)
		assertEqual(t, 3, retrieved.Hearts, "hearts")

		// Refill hearts
		_ = users.UpdateHearts(ctx, user.ID, 5)
		retrieved, _ = users.GetByID(ctx, user.ID)
		assertEqual(t, 5, retrieved.Hearts, "hearts after refill")
	})
}

func TestUserStore_UpdateGems(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		users := s.Users()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		user := &store.User{
			ID:                uuid.New(),
			Email:             "gems@example.com",
			Username:          "gemsuser",
			DisplayName:       "Gems User",
			EncryptedPassword: string(hashedPassword),
			Gems:              500,
			Hearts:            5,
			DailyGoalMinutes:  10,
			CreatedAt:         time.Now(),
		}

		_ = users.Create(ctx, user)

		// Spend gems
		err := users.UpdateGems(ctx, user.ID, 350)
		assertNoError(t, err, "update gems")

		// Verify gems
		retrieved, _ := users.GetByID(ctx, user.ID)
		assertEqual(t, 350, retrieved.Gems, "gems")

		// Add gems
		_ = users.UpdateGems(ctx, user.ID, 500)
		retrieved, _ = users.GetByID(ctx, user.ID)
		assertEqual(t, 500, retrieved.Gems, "gems after add")
	})
}
