package storetest

import (
	"testing"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
)

func TestUsersStore_Create(t *testing.T) {
	RunForAllDrivers(t, "Create", func(t *testing.T, factory StoreFactory) {
		db := factory.SetupDB(t)
		store := factory.NewUsersStore(db)

		now := FixedTime()
		user := &users.User{
			ID:        NewTestID(),
			Email:     "test@example.com",
			Name:      "Test User",
			Password:  "hashedpassword123",
			Avatar:    "https://example.com/avatar.png",
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := store.Create(t.Context(), user)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Verify user was created
		got, err := store.GetByID(t.Context(), user.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if got.ID != user.ID {
			t.Errorf("ID = %v, want %v", got.ID, user.ID)
		}
		if got.Email != user.Email {
			t.Errorf("Email = %v, want %v", got.Email, user.Email)
		}
		if got.Name != user.Name {
			t.Errorf("Name = %v, want %v", got.Name, user.Name)
		}
	})
}

func TestUsersStore_Create_DuplicateEmail(t *testing.T) {
	RunForAllDrivers(t, "Create_DuplicateEmail", func(t *testing.T, factory StoreFactory) {
		db := factory.SetupDB(t)
		store := factory.NewUsersStore(db)

		now := FixedTime()
		user1 := &users.User{
			ID:        NewTestID(),
			Email:     "duplicate@example.com",
			Name:      "User 1",
			Password:  "hash1",
			CreatedAt: now,
			UpdatedAt: now,
		}

		user2 := &users.User{
			ID:        NewTestID(),
			Email:     "duplicate@example.com",
			Name:      "User 2",
			Password:  "hash2",
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := store.Create(t.Context(), user1); err != nil {
			t.Fatalf("Create(user1) error = %v", err)
		}

		err := store.Create(t.Context(), user2)
		if err == nil {
			t.Error("Create(user2) expected error for duplicate email, got nil")
		}
	})
}

func TestUsersStore_GetByID_NotFound(t *testing.T) {
	RunForAllDrivers(t, "GetByID_NotFound", func(t *testing.T, factory StoreFactory) {
		db := factory.SetupDB(t)
		store := factory.NewUsersStore(db)

		_, err := store.GetByID(t.Context(), "nonexistent-id")
		if err != users.ErrUserNotFound {
			t.Errorf("GetByID() error = %v, want %v", err, users.ErrUserNotFound)
		}
	})
}

func TestUsersStore_GetByEmail(t *testing.T) {
	RunForAllDrivers(t, "GetByEmail", func(t *testing.T, factory StoreFactory) {
		db := factory.SetupDB(t)
		store := factory.NewUsersStore(db)

		now := FixedTime()
		user := &users.User{
			ID:        NewTestID(),
			Email:     "getbyemail@example.com",
			Name:      "Get By Email Test",
			Password:  "hash",
			Avatar:    "avatar.png",
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := store.Create(t.Context(), user); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := store.GetByEmail(t.Context(), user.Email)
		if err != nil {
			t.Fatalf("GetByEmail() error = %v", err)
		}

		if got.ID != user.ID {
			t.Errorf("ID = %v, want %v", got.ID, user.ID)
		}
		if got.Email != user.Email {
			t.Errorf("Email = %v, want %v", got.Email, user.Email)
		}
	})
}

func TestUsersStore_GetByEmail_NotFound(t *testing.T) {
	RunForAllDrivers(t, "GetByEmail_NotFound", func(t *testing.T, factory StoreFactory) {
		db := factory.SetupDB(t)
		store := factory.NewUsersStore(db)

		_, err := store.GetByEmail(t.Context(), "nonexistent@example.com")
		if err != users.ErrUserNotFound {
			t.Errorf("GetByEmail() error = %v, want %v", err, users.ErrUserNotFound)
		}
	})
}

func TestUsersStore_Update(t *testing.T) {
	RunForAllDrivers(t, "Update", func(t *testing.T, factory StoreFactory) {
		db := factory.SetupDB(t)
		store := factory.NewUsersStore(db)

		now := FixedTime()
		user := &users.User{
			ID:        NewTestID(),
			Email:     "update@example.com",
			Name:      "Original Name",
			Password:  "hash",
			Avatar:    "old-avatar.png",
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := store.Create(t.Context(), user); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Update user
		user.Name = "Updated Name"
		user.Avatar = "new-avatar.png"
		user.UpdatedAt = now.Add(time.Hour)

		if err := store.Update(t.Context(), user); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		// Verify update
		got, err := store.GetByID(t.Context(), user.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if got.Name != "Updated Name" {
			t.Errorf("Name = %v, want Updated Name", got.Name)
		}
		if got.Avatar != "new-avatar.png" {
			t.Errorf("Avatar = %v, want new-avatar.png", got.Avatar)
		}
	})
}
