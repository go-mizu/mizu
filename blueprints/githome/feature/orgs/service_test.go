package orgs_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*orgs.Service, *duckdb.Store, func()) {
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

	orgsStore := duckdb.NewOrgsStore(db)
	service := orgs.NewService(orgsStore, store.Users(), "https://api.example.com")

	cleanup := func() {
		store.Close()
	}

	return service, store, cleanup
}

func createTestUser(t *testing.T, store *duckdb.Store, login, email string) *users.User {
	t.Helper()
	user := &users.User{
		Login:        login,
		Email:        email,
		Name:         "Test User",
		PasswordHash: "hash",
		Type:         "User",
	}
	if err := store.Users().Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestOrg(t *testing.T, service *orgs.Service, creatorID int64, login string) *orgs.Organization {
	t.Helper()
	org, err := service.Create(context.Background(), creatorID, &orgs.CreateIn{
		Login:       login,
		Email:       login + "@example.com",
		Description: "Test organization",
	})
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	return org
}

// Organization Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	org, err := service.Create(context.Background(), creator.ID, &orgs.CreateIn{
		Login:       "testorg",
		Email:       "org@example.com",
		Name:        "Test Organization",
		Description: "A test organization",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if org.Login != "testorg" {
		t.Errorf("got login %q, want testorg", org.Login)
	}
	if org.Email != "org@example.com" {
		t.Errorf("got email %q, want org@example.com", org.Email)
	}
	if org.Name != "Test Organization" {
		t.Errorf("got name %q, want Test Organization", org.Name)
	}
	if org.Description != "A test organization" {
		t.Errorf("got description %q, want A test organization", org.Description)
	}
	if org.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if org.Type != "Organization" {
		t.Errorf("got type %q, want Organization", org.Type)
	}
	if org.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Create_DuplicateLogin(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	createTestOrg(t, service, creator.ID, "testorg")

	_, err := service.Create(context.Background(), creator.ID, &orgs.CreateIn{
		Login: "testorg",
		Email: "other@example.com",
	})
	if err != orgs.ErrOrgExists {
		t.Errorf("expected ErrOrgExists, got %v", err)
	}
}

// Organization Retrieval Tests

func TestService_Get_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	created := createTestOrg(t, service, creator.ID, "testorg")

	org, err := service.Get(context.Background(), "testorg")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if org.ID != created.ID {
		t.Errorf("got ID %d, want %d", org.ID, created.ID)
	}
	if org.Login != "testorg" {
		t.Errorf("got login %q, want testorg", org.Login)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Get(context.Background(), "nonexistent")
	if err != orgs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetByID_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	created := createTestOrg(t, service, creator.ID, "testorg")

	org, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if org.ID != created.ID {
		t.Errorf("got ID %d, want %d", org.ID, created.ID)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetByID(context.Background(), 99999)
	if err != orgs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_List(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	createTestOrg(t, service, creator.ID, "org1")
	createTestOrg(t, service, creator.ID, "org2")
	createTestOrg(t, service, creator.ID, "org3")

	list, err := service.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 orgs, got %d", len(list))
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	for i := 0; i < 5; i++ {
		createTestOrg(t, service, creator.ID, "org"+string(rune('a'+i)))
	}

	list, err := service.List(context.Background(), &orgs.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 orgs, got %d", len(list))
	}
}

// Organization Update Tests

func TestService_Update_Description(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	created := createTestOrg(t, service, creator.ID, "testorg")

	newDesc := "Updated description"
	updated, err := service.Update(context.Background(), created.Login, &orgs.UpdateIn{
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("got description %q, want Updated description", updated.Description)
	}
}

func TestService_Update_Name(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	created := createTestOrg(t, service, creator.ID, "testorg")

	newName := "New Org Name"
	updated, err := service.Update(context.Background(), created.Login, &orgs.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "New Org Name" {
		t.Errorf("got name %q, want New Org Name", updated.Name)
	}
}

// Organization Delete Tests

func TestService_Delete_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	created := createTestOrg(t, service, creator.ID, "testorg")

	err := service.Delete(context.Background(), created.Login)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = service.Get(context.Background(), "testorg")
	if err != orgs.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// Membership Tests - skipped until membership methods are implemented

func TestService_AddMember_Success(t *testing.T) {
	t.Skip("AddMember not yet implemented")
}

func TestService_AddMember_OrgNotFound(t *testing.T) {
	t.Skip("AddMember not yet implemented")
}

func TestService_GetMembership_NotMember(t *testing.T) {
	t.Skip("membership methods not yet implemented")
}

func TestService_RemoveMember_Success(t *testing.T) {
	t.Skip("membership methods not yet implemented")
}

func TestService_ListMembers(t *testing.T) {
	t.Skip("membership methods not yet implemented")
}

func TestService_ListMembers_Pagination(t *testing.T) {
	t.Skip("membership methods not yet implemented")
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	creator := createTestUser(t, store, "creator", "creator@example.com")
	org := createTestOrg(t, service, creator.ID, "testorg")

	if org.URL != "https://api.example.com/api/v3/orgs/testorg" {
		t.Errorf("unexpected URL: %s", org.URL)
	}
	if org.HTMLURL != "https://api.example.com/testorg" {
		t.Errorf("unexpected HTMLURL: %s", org.HTMLURL)
	}
	if org.ReposURL != "https://api.example.com/api/v3/orgs/testorg/repos" {
		t.Errorf("unexpected ReposURL: %s", org.ReposURL)
	}
	if org.MembersURL == "" {
		t.Error("expected MembersURL to be set")
	}
	if org.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

// List User's Orgs Tests

func TestService_ListForUser(t *testing.T) {
	t.Skip("membership methods not yet implemented")
}
