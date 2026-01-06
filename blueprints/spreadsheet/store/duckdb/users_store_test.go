package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
)

func TestUsersStore_Create(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

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

	err := store.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify user was created
	got, err := store.GetByID(ctx, user.ID)
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
	if got.Password != user.Password {
		t.Errorf("Password = %v, want %v", got.Password, user.Password)
	}
}

func TestUsersStore_Create_DuplicateEmail(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

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
		Email:     "duplicate@example.com", // Same email
		Name:      "User 2",
		Password:  "hash2",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(ctx, user1); err != nil {
		t.Fatalf("Create(user1) error = %v", err)
	}

	err := store.Create(ctx, user2)
	if err == nil {
		t.Error("Create(user2) expected error for duplicate email, got nil")
	}
}

func TestUsersStore_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create a user first
	now := FixedTime()
	user := &users.User{
		ID:        NewTestID(),
		Email:     "getbyid@example.com",
		Name:      "Get By ID Test",
		Password:  "hash",
		Avatar:    "avatar.png",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %v, want %v", got.ID, user.ID)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %v, want %v", got.Email, user.Email)
	}
}

func TestUsersStore_GetByID_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent-id")
	if err != users.ErrUserNotFound {
		t.Errorf("GetByID() error = %v, want %v", err, users.ErrUserNotFound)
	}
}

func TestUsersStore_GetByEmail(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

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
	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %v, want %v", got.ID, user.ID)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %v, want %v", got.Email, user.Email)
	}
	if got.Avatar != user.Avatar {
		t.Errorf("Avatar = %v, want %v", got.Avatar, user.Avatar)
	}
}

func TestUsersStore_GetByEmail_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	_, err := store.GetByEmail(ctx, "nonexistent@example.com")
	if err != users.ErrUserNotFound {
		t.Errorf("GetByEmail() error = %v, want %v", err, users.ErrUserNotFound)
	}
}

func TestUsersStore_GetByEmail_NullAvatar(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	now := FixedTime()
	user := &users.User{
		ID:        NewTestID(),
		Email:     "nullavatar@example.com",
		Name:      "Null Avatar Test",
		Password:  "hash",
		Avatar:    "", // Empty avatar
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	if got.Avatar != "" {
		t.Errorf("Avatar = %v, want empty string", got.Avatar)
	}
}

func TestUsersStore_Update(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

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
	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update user
	user.Name = "Updated Name"
	user.Avatar = "new-avatar.png"
	user.UpdatedAt = now.Add(time.Hour)

	if err := store.Update(ctx, user); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, err := store.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", got.Name)
	}
	if got.Avatar != "new-avatar.png" {
		t.Errorf("Avatar = %v, want new-avatar.png", got.Avatar)
	}
}

func TestUsersStore_Update_PartialFields(t *testing.T) {
	db := SetupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	now := FixedTime()
	user := &users.User{
		ID:        NewTestID(),
		Email:     "partial@example.com",
		Name:      "Original Name",
		Password:  "original-hash",
		Avatar:    "avatar.png",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update only name, keep avatar
	user.Name = "New Name"
	user.UpdatedAt = now.Add(time.Hour)

	if err := store.Update(ctx, user); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := store.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Verify name changed
	if got.Name != "New Name" {
		t.Errorf("Name = %v, want New Name", got.Name)
	}

	// Verify other fields unchanged
	if got.Email != "partial@example.com" {
		t.Errorf("Email = %v, want partial@example.com (should be unchanged)", got.Email)
	}
	if got.Password != "original-hash" {
		t.Errorf("Password should be unchanged")
	}
}
