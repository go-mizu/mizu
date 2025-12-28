package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

func createTestUser(t *testing.T, store *UsersStore, login string) *users.User {
	t.Helper()
	u := &users.User{
		Login:        login,
		Name:         "Test " + login,
		Email:        login + "@example.com",
		Type:         "User",
		PasswordHash: "hashed",
	}
	if err := store.Create(context.Background(), u); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u
}

func TestUsersStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	u := &users.User{
		Login:        "testuser",
		Name:         "Test User",
		Email:        "test@example.com",
		Type:         "User",
		PasswordHash: "hashed",
	}

	err := usersStore.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if u.ID == 0 {
		t.Error("expected ID to be set")
	}
	if u.NodeID == "" {
		t.Error("expected NodeID to be set")
	}

	got, err := usersStore.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user to be created")
	}
	if got.Login != u.Login {
		t.Errorf("got login %q, want %q", got.Login, u.Login)
	}
}

func TestUsersStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore, "getbyid")

	got, err := usersStore.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user")
	}
	if got.ID != u.ID {
		t.Errorf("got ID %d, want %d", got.ID, u.ID)
	}
}

func TestUsersStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	got, err := usersStore.GetByID(context.Background(), 999999)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent user")
	}
}

func TestUsersStore_GetByLogin(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore, "getbylogin")

	got, err := usersStore.GetByLogin(context.Background(), u.Login)
	if err != nil {
		t.Fatalf("GetByLogin failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user")
	}
	if got.Login != u.Login {
		t.Errorf("got login %q, want %q", got.Login, u.Login)
	}
}

func TestUsersStore_GetByEmail(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore, "getbyemail")

	got, err := usersStore.GetByEmail(context.Background(), u.Email)
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user")
	}
	if got.Email != u.Email {
		t.Errorf("got email %q, want %q", got.Email, u.Email)
	}
}

func TestUsersStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore, "updateuser")

	newName := "Updated Name"
	newBio := "Updated bio"
	err := usersStore.Update(context.Background(), u.ID, &users.UpdateIn{
		Name: &newName,
		Bio:  &newBio,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got.Name != newName {
		t.Errorf("got name %q, want %q", got.Name, newName)
	}
	if got.Bio != newBio {
		t.Errorf("got bio %q, want %q", got.Bio, newBio)
	}
}

func TestUsersStore_UpdatePassword(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore, "updatepwd")

	newHash := "newhash"
	err := usersStore.UpdatePassword(context.Background(), u.ID, newHash)
	if err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got.PasswordHash != newHash {
		t.Errorf("got hash %q, want %q", got.PasswordHash, newHash)
	}
}

func TestUsersStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore, "deleteuser")

	err := usersStore.Delete(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got != nil {
		t.Error("expected user to be deleted")
	}
}

func TestUsersStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	createTestUser(t, usersStore, "list1")
	createTestUser(t, usersStore, "list2")
	createTestUser(t, usersStore, "list3")

	list, err := usersStore.List(context.Background(), &users.ListOpts{PerPage: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d users, want 3", len(list))
	}
}

func TestUsersStore_List_WithSince(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := createTestUser(t, usersStore, "since1")
	createTestUser(t, usersStore, "since2")
	createTestUser(t, usersStore, "since3")

	list, err := usersStore.List(context.Background(), &users.ListOpts{Since: u1.ID})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d users, want 2", len(list))
	}
}

func TestUsersStore_CreateFollow(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := createTestUser(t, usersStore, "follower")
	u2 := createTestUser(t, usersStore, "followed")

	err := usersStore.CreateFollow(context.Background(), u1.ID, u2.ID)
	if err != nil {
		t.Fatalf("CreateFollow failed: %v", err)
	}

	isFollowing, err := usersStore.IsFollowing(context.Background(), u1.ID, u2.ID)
	if err != nil {
		t.Fatalf("IsFollowing failed: %v", err)
	}
	if !isFollowing {
		t.Error("expected user to be following")
	}
}

func TestUsersStore_DeleteFollow(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := createTestUser(t, usersStore, "unfollower")
	u2 := createTestUser(t, usersStore, "unfollowed")

	usersStore.CreateFollow(context.Background(), u1.ID, u2.ID)

	err := usersStore.DeleteFollow(context.Background(), u1.ID, u2.ID)
	if err != nil {
		t.Fatalf("DeleteFollow failed: %v", err)
	}

	isFollowing, _ := usersStore.IsFollowing(context.Background(), u1.ID, u2.ID)
	if isFollowing {
		t.Error("expected user to not be following")
	}
}

func TestUsersStore_ListFollowers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := createTestUser(t, usersStore, "target")
	u2 := createTestUser(t, usersStore, "follower1")
	u3 := createTestUser(t, usersStore, "follower2")

	usersStore.CreateFollow(context.Background(), u2.ID, u1.ID)
	usersStore.CreateFollow(context.Background(), u3.ID, u1.ID)

	list, err := usersStore.ListFollowers(context.Background(), u1.ID, nil)
	if err != nil {
		t.Fatalf("ListFollowers failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d followers, want 2", len(list))
	}
}

func TestUsersStore_ListFollowing(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := createTestUser(t, usersStore, "follower")
	u2 := createTestUser(t, usersStore, "target1")
	u3 := createTestUser(t, usersStore, "target2")

	usersStore.CreateFollow(context.Background(), u1.ID, u2.ID)
	usersStore.CreateFollow(context.Background(), u1.ID, u3.ID)

	list, err := usersStore.ListFollowing(context.Background(), u1.ID, nil)
	if err != nil {
		t.Fatalf("ListFollowing failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d following, want 2", len(list))
	}
}

func TestUsersStore_IncrementFollowers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore, "incfollowers")

	err := usersStore.IncrementFollowers(context.Background(), u.ID, 5)
	if err != nil {
		t.Fatalf("IncrementFollowers failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got.Followers != 5 {
		t.Errorf("got followers %d, want 5", got.Followers)
	}
}

func TestUsersStore_IncrementFollowing(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore, "incfollowing")

	err := usersStore.IncrementFollowing(context.Background(), u.ID, 3)
	if err != nil {
		t.Fatalf("IncrementFollowing failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got.Following != 3 {
		t.Errorf("got following %d, want 3", got.Following)
	}
}
