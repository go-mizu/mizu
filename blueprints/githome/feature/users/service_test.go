package users_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*users.Service, *duckdb.Store, func()) {
	t.Helper()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}

	store, err := duckdb.New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		store.Close()
		t.Fatalf("failed to ensure schema: %v", err)
	}

	usersStore := duckdb.NewUsersStore(db)
	service := users.NewService(usersStore, "https://api.example.com")

	cleanup := func() {
		store.Close()
	}

	return service, store, cleanup
}

func createTestUser(t *testing.T, service *users.Service, login, email string) *users.User {
	t.Helper()
	user, err := service.Create(context.Background(), &users.CreateIn{
		Login:    login,
		Email:    email,
		Password: "password123",
		Name:     "Test User",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// User Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	user, err := service.Create(context.Background(), &users.CreateIn{
		Login:    "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if user.Login != "testuser" {
		t.Errorf("got login %q, want testuser", user.Login)
	}
	if user.Email != "test@example.com" {
		t.Errorf("got email %q, want test@example.com", user.Email)
	}
	if user.Name != "Test User" {
		t.Errorf("got name %q, want Test User", user.Name)
	}
	if user.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if user.Type != "User" {
		t.Errorf("got type %q, want User", user.Type)
	}
	if user.URL == "" {
		t.Error("expected URL to be populated")
	}
	if user.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

func TestService_Create_DuplicateLogin(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, service, "testuser", "test1@example.com")

	_, err := service.Create(context.Background(), &users.CreateIn{
		Login:    "testuser",
		Email:    "test2@example.com",
		Password: "password123",
	})
	if err != users.ErrUserExists {
		t.Errorf("expected ErrUserExists, got %v", err)
	}
}

func TestService_Create_DuplicateEmail(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, service, "testuser1", "test@example.com")

	_, err := service.Create(context.Background(), &users.CreateIn{
		Login:    "testuser2",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != users.ErrEmailExists {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestService_Create_PasswordHashed(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	user, err := service.Create(context.Background(), &users.CreateIn{
		Login:    "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Password hash should not equal plaintext
	if user.PasswordHash == "password123" {
		t.Error("password should be hashed")
	}
	if user.PasswordHash == "" {
		t.Error("password hash should be set")
	}
}

// Authentication Tests

func TestService_Authenticate_ByLogin(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, service, "testuser", "test@example.com")

	user, err := service.Authenticate(context.Background(), "testuser", "password123")
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}

	if user.Login != "testuser" {
		t.Errorf("got login %q, want testuser", user.Login)
	}
}

func TestService_Authenticate_ByEmail(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, service, "testuser", "test@example.com")

	user, err := service.Authenticate(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}

	if user.Login != "testuser" {
		t.Errorf("got login %q, want testuser", user.Login)
	}
}

func TestService_Authenticate_WrongPassword(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, service, "testuser", "test@example.com")

	_, err := service.Authenticate(context.Background(), "testuser", "wrongpassword")
	if err != users.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_Authenticate_NonexistentUser(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Authenticate(context.Background(), "nonexistent", "password123")
	if err != users.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

// User Retrieval Tests

func TestService_GetByID_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	created := createTestUser(t, service, "testuser", "test@example.com")

	user, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if user.ID != created.ID {
		t.Errorf("got ID %d, want %d", user.ID, created.ID)
	}
	if user.Login != "testuser" {
		t.Errorf("got login %q, want testuser", user.Login)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetByID(context.Background(), 99999)
	if err != users.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetByLogin_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, service, "testuser", "test@example.com")

	user, err := service.GetByLogin(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("GetByLogin failed: %v", err)
	}

	if user.Login != "testuser" {
		t.Errorf("got login %q, want testuser", user.Login)
	}
}

func TestService_GetByLogin_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetByLogin(context.Background(), "nonexistent")
	if err != users.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetByEmail_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, service, "testuser", "test@example.com")

	user, err := service.GetByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("got email %q, want test@example.com", user.Email)
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	// Create multiple users
	for i := 0; i < 5; i++ {
		createTestUser(t, service, "user"+string(rune('a'+i)), "user"+string(rune('a'+i))+"@example.com")
	}

	// List with pagination
	list, err := service.List(context.Background(), &users.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 users, got %d", len(list))
	}
}

func TestService_List_MaxPerPage(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	// Request more than max
	_, err := service.List(context.Background(), &users.ListOpts{
		PerPage: 200, // Should be capped at 100
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

// User Update Tests

func TestService_Update_Name(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	created := createTestUser(t, service, "testuser", "test@example.com")

	newName := "Updated Name"
	updated, err := service.Update(context.Background(), created.ID, &users.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("got name %q, want Updated Name", updated.Name)
	}
}

func TestService_Update_Bio(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	created := createTestUser(t, service, "testuser", "test@example.com")

	newBio := "This is my bio"
	updated, err := service.Update(context.Background(), created.ID, &users.UpdateIn{
		Bio: &newBio,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Bio != "This is my bio" {
		t.Errorf("got bio %q, want This is my bio", updated.Bio)
	}
}

func TestService_Delete_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	created := createTestUser(t, service, "testuser", "test@example.com")

	err := service.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = service.GetByID(context.Background(), created.ID)
	if err != users.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// Password Management Tests

func TestService_UpdatePassword_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	created := createTestUser(t, service, "testuser", "test@example.com")

	err := service.UpdatePassword(context.Background(), created.ID, "password123", "newpassword456")
	if err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	// Verify new password works
	_, err = service.Authenticate(context.Background(), "testuser", "newpassword456")
	if err != nil {
		t.Errorf("authentication with new password failed: %v", err)
	}

	// Verify old password doesn't work
	_, err = service.Authenticate(context.Background(), "testuser", "password123")
	if err != users.ErrInvalidCredentials {
		t.Error("old password should not work")
	}
}

func TestService_UpdatePassword_WrongOld(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	created := createTestUser(t, service, "testuser", "test@example.com")

	err := service.UpdatePassword(context.Background(), created.ID, "wrongpassword", "newpassword456")
	if err != users.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

// Follow System Tests

func TestService_Follow_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	follower := createTestUser(t, service, "follower", "follower@example.com")
	target := createTestUser(t, service, "target", "target@example.com")

	err := service.Follow(context.Background(), follower.ID, "target")
	if err != nil {
		t.Fatalf("Follow failed: %v", err)
	}

	// Verify relationship exists
	isFollowing, err := service.IsFollowing(context.Background(), "follower", "target")
	if err != nil {
		t.Fatalf("IsFollowing failed: %v", err)
	}
	if !isFollowing {
		t.Error("expected to be following")
	}

	// Verify counters updated
	updatedFollower, _ := service.GetByID(context.Background(), follower.ID)
	if updatedFollower.Following != 1 {
		t.Errorf("expected following count 1, got %d", updatedFollower.Following)
	}

	updatedTarget, _ := service.GetByID(context.Background(), target.ID)
	if updatedTarget.Followers != 1 {
		t.Errorf("expected followers count 1, got %d", updatedTarget.Followers)
	}
}

func TestService_Follow_AlreadyFollowing(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	follower := createTestUser(t, service, "follower", "follower@example.com")
	createTestUser(t, service, "target", "target@example.com")

	// First follow
	err := service.Follow(context.Background(), follower.ID, "target")
	if err != nil {
		t.Fatalf("First follow failed: %v", err)
	}

	// Second follow should be idempotent
	err = service.Follow(context.Background(), follower.ID, "target")
	if err != nil {
		t.Fatalf("Second follow should succeed (idempotent): %v", err)
	}

	// Counter should still be 1
	updatedFollower, _ := service.GetByID(context.Background(), follower.ID)
	if updatedFollower.Following != 1 {
		t.Errorf("expected following count 1, got %d", updatedFollower.Following)
	}
}

func TestService_Follow_TargetNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	follower := createTestUser(t, service, "follower", "follower@example.com")

	err := service.Follow(context.Background(), follower.ID, "nonexistent")
	if err != users.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Unfollow_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	follower := createTestUser(t, service, "follower", "follower@example.com")
	target := createTestUser(t, service, "target", "target@example.com")

	// First follow
	_ = service.Follow(context.Background(), follower.ID, "target")

	// Then unfollow
	err := service.Unfollow(context.Background(), follower.ID, "target")
	if err != nil {
		t.Fatalf("Unfollow failed: %v", err)
	}

	// Verify relationship removed
	isFollowing, _ := service.IsFollowing(context.Background(), "follower", "target")
	if isFollowing {
		t.Error("should not be following after unfollow")
	}

	// Verify counters decremented
	updatedFollower, _ := service.GetByID(context.Background(), follower.ID)
	if updatedFollower.Following != 0 {
		t.Errorf("expected following count 0, got %d", updatedFollower.Following)
	}

	updatedTarget, _ := service.GetByID(context.Background(), target.ID)
	if updatedTarget.Followers != 0 {
		t.Errorf("expected followers count 0, got %d", updatedTarget.Followers)
	}
}

func TestService_Unfollow_NotFollowing(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	follower := createTestUser(t, service, "follower", "follower@example.com")
	createTestUser(t, service, "target", "target@example.com")

	// Unfollow without following should be idempotent
	err := service.Unfollow(context.Background(), follower.ID, "target")
	if err != nil {
		t.Fatalf("Unfollow should succeed (idempotent): %v", err)
	}
}

func TestService_IsFollowing_True(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	follower := createTestUser(t, service, "follower", "follower@example.com")
	createTestUser(t, service, "target", "target@example.com")

	_ = service.Follow(context.Background(), follower.ID, "target")

	isFollowing, err := service.IsFollowing(context.Background(), "follower", "target")
	if err != nil {
		t.Fatalf("IsFollowing failed: %v", err)
	}
	if !isFollowing {
		t.Error("expected to be following")
	}
}

func TestService_IsFollowing_False(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, service, "follower", "follower@example.com")
	createTestUser(t, service, "target", "target@example.com")

	isFollowing, err := service.IsFollowing(context.Background(), "follower", "target")
	if err != nil {
		t.Fatalf("IsFollowing failed: %v", err)
	}
	if isFollowing {
		t.Error("should not be following")
	}
}

func TestService_ListFollowers(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	target := createTestUser(t, service, "target", "target@example.com")
	follower1 := createTestUser(t, service, "follower1", "follower1@example.com")
	follower2 := createTestUser(t, service, "follower2", "follower2@example.com")

	_ = service.Follow(context.Background(), follower1.ID, "target")
	_ = service.Follow(context.Background(), follower2.ID, "target")

	followers, err := service.ListFollowers(context.Background(), "target", nil)
	if err != nil {
		t.Fatalf("ListFollowers failed: %v", err)
	}

	if len(followers) != 2 {
		t.Errorf("expected 2 followers, got %d", len(followers))
	}

	// Verify target user
	updatedTarget, _ := service.GetByID(context.Background(), target.ID)
	if updatedTarget.Followers != 2 {
		t.Errorf("expected followers count 2, got %d", updatedTarget.Followers)
	}
}

func TestService_ListFollowing(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	follower := createTestUser(t, service, "follower", "follower@example.com")
	createTestUser(t, service, "target1", "target1@example.com")
	createTestUser(t, service, "target2", "target2@example.com")

	_ = service.Follow(context.Background(), follower.ID, "target1")
	_ = service.Follow(context.Background(), follower.ID, "target2")

	following, err := service.ListFollowing(context.Background(), "follower", nil)
	if err != nil {
		t.Fatalf("ListFollowing failed: %v", err)
	}

	if len(following) != 2 {
		t.Errorf("expected following 2, got %d", len(following))
	}

	// Verify follower user
	updatedFollower, _ := service.GetByID(context.Background(), follower.ID)
	if updatedFollower.Following != 2 {
		t.Errorf("expected following count 2, got %d", updatedFollower.Following)
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, service, "testuser", "test@example.com")

	if user.URL != "https://api.example.com/api/v3/users/testuser" {
		t.Errorf("unexpected URL: %s", user.URL)
	}
	if user.HTMLURL != "https://api.example.com/testuser" {
		t.Errorf("unexpected HTMLURL: %s", user.HTMLURL)
	}
	if user.FollowersURL != "https://api.example.com/api/v3/users/testuser/followers" {
		t.Errorf("unexpected FollowersURL: %s", user.FollowersURL)
	}
	if user.ReposURL != "https://api.example.com/api/v3/users/testuser/repos" {
		t.Errorf("unexpected ReposURL: %s", user.ReposURL)
	}
}
