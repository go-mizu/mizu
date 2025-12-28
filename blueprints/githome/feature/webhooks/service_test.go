package webhooks_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/feature/webhooks"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*webhooks.Service, *duckdb.Store, func()) {
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

	service := webhooks.NewService(store.Webhooks(), store.Repos(), store.Orgs(), "https://api.example.com")

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

func createTestRepo(t *testing.T, store *duckdb.Store, owner *users.User, name string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if err := store.Repos().Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestOrg(t *testing.T, store *duckdb.Store, login string) *orgs.Organization {
	t.Helper()
	// First create user with Organization type
	user := &users.User{
		Login:        login,
		Email:        login + "@example.com",
		Name:         "Test Organization",
		PasswordHash: "",
		Type:         "Organization",
	}
	if err := store.Users().Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test org user: %v", err)
	}

	org := &orgs.Organization{
		ID:    user.ID,
		Login: login,
		Email: login + "@example.com",
		Type:  "Organization",
	}
	if err := store.Orgs().Create(context.Background(), org); err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	return org
}

func createTestWebhookForRepo(t *testing.T, service *webhooks.Service, owner, repo string) *webhooks.Webhook {
	t.Helper()
	hook, err := service.CreateForRepo(context.Background(), owner, repo, &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL:         "https://example.com/webhook",
			ContentType: "json",
		},
		Events: []string{"push", "pull_request"},
	})
	if err != nil {
		t.Fatalf("failed to create test webhook: %v", err)
	}
	return hook
}

// Repository Webhook Tests

func TestService_CreateForRepo_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	hook, err := service.CreateForRepo(context.Background(), "owner", "testrepo", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL:         "https://example.com/webhook",
			ContentType: "json",
			Secret:      "mysecret",
		},
		Events: []string{"push", "pull_request"},
	})
	if err != nil {
		t.Fatalf("CreateForRepo failed: %v", err)
	}

	if hook.Name != "web" {
		t.Errorf("got name %q, want web", hook.Name)
	}
	if hook.Config.URL != "https://example.com/webhook" {
		t.Errorf("got config.url %q, want https://example.com/webhook", hook.Config.URL)
	}
	if len(hook.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(hook.Events))
	}
	if !hook.Active {
		t.Error("expected webhook to be active by default")
	}
	if hook.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if hook.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_CreateForRepo_DefaultEvents(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	hook, err := service.CreateForRepo(context.Background(), "owner", "testrepo", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL: "https://example.com/webhook",
		},
	})
	if err != nil {
		t.Fatalf("CreateForRepo failed: %v", err)
	}

	if len(hook.Events) != 1 || hook.Events[0] != "push" {
		t.Errorf("expected default event [push], got %v", hook.Events)
	}
}

func TestService_CreateForRepo_Inactive(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	active := false
	hook, err := service.CreateForRepo(context.Background(), "owner", "testrepo", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL: "https://example.com/webhook",
		},
		Active: &active,
	})
	if err != nil {
		t.Fatalf("CreateForRepo failed: %v", err)
	}

	if hook.Active {
		t.Error("expected webhook to be inactive")
	}
}

func TestService_CreateForRepo_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, store, "owner", "owner@example.com")

	_, err := service.CreateForRepo(context.Background(), "owner", "unknown", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL: "https://example.com/webhook",
		},
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetForRepo_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestWebhookForRepo(t, service, "owner", "testrepo")

	hook, err := service.GetForRepo(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("GetForRepo failed: %v", err)
	}

	if hook.ID != created.ID {
		t.Errorf("got ID %d, want %d", hook.ID, created.ID)
	}
}

func TestService_GetForRepo_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_, err := service.GetForRepo(context.Background(), "owner", "testrepo", 99999)
	if err != webhooks.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ListForRepo(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	createTestWebhookForRepo(t, service, "owner", "testrepo")
	createTestWebhookForRepo(t, service, "owner", "testrepo")

	list, err := service.ListForRepo(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListForRepo failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(list))
	}
}

func TestService_ListForRepo_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	for i := 0; i < 5; i++ {
		createTestWebhookForRepo(t, service, "owner", "testrepo")
	}

	list, err := service.ListForRepo(context.Background(), "owner", "testrepo", &webhooks.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListForRepo failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(list))
	}
}

func TestService_UpdateForRepo_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestWebhookForRepo(t, service, "owner", "testrepo")

	active := false
	updated, err := service.UpdateForRepo(context.Background(), "owner", "testrepo", created.ID, &webhooks.UpdateIn{
		Active: &active,
		Events: []string{"push"},
	})
	if err != nil {
		t.Fatalf("UpdateForRepo failed: %v", err)
	}

	if updated.Active {
		t.Error("expected webhook to be inactive after update")
	}
}

func TestService_UpdateForRepo_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	active := false
	_, err := service.UpdateForRepo(context.Background(), "owner", "testrepo", 99999, &webhooks.UpdateIn{
		Active: &active,
	})
	if err != webhooks.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_DeleteForRepo_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestWebhookForRepo(t, service, "owner", "testrepo")

	err := service.DeleteForRepo(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("DeleteForRepo failed: %v", err)
	}

	// Verify deleted
	_, err = service.GetForRepo(context.Background(), "owner", "testrepo", created.ID)
	if err != webhooks.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_DeleteForRepo_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	err := service.DeleteForRepo(context.Background(), "owner", "testrepo", 99999)
	if err != webhooks.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Organization Webhook Tests

func TestService_CreateForOrg_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	hook, err := service.CreateForOrg(context.Background(), "testorg", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL:         "https://example.com/webhook",
			ContentType: "json",
		},
		Events: []string{"push"},
	})
	if err != nil {
		t.Fatalf("CreateForOrg failed: %v", err)
	}

	if hook.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if hook.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_CreateForOrg_OrgNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.CreateForOrg(context.Background(), "unknown", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL: "https://example.com/webhook",
		},
	})
	if err != orgs.ErrNotFound {
		t.Errorf("expected orgs.ErrNotFound, got %v", err)
	}
}

func TestService_GetForOrg_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	created, _ := service.CreateForOrg(context.Background(), "testorg", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL: "https://example.com/webhook",
		},
	})

	hook, err := service.GetForOrg(context.Background(), "testorg", created.ID)
	if err != nil {
		t.Fatalf("GetForOrg failed: %v", err)
	}

	if hook.ID != created.ID {
		t.Errorf("got ID %d, want %d", hook.ID, created.ID)
	}
}

func TestService_GetForOrg_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	_, err := service.GetForOrg(context.Background(), "testorg", 99999)
	if err != webhooks.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ListForOrg(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	_, _ = service.CreateForOrg(context.Background(), "testorg", &webhooks.CreateIn{
		Config: &webhooks.Config{URL: "https://example.com/webhook1"},
	})
	_, _ = service.CreateForOrg(context.Background(), "testorg", &webhooks.CreateIn{
		Config: &webhooks.Config{URL: "https://example.com/webhook2"},
	})

	list, err := service.ListForOrg(context.Background(), "testorg", nil)
	if err != nil {
		t.Fatalf("ListForOrg failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(list))
	}
}

func TestService_UpdateForOrg_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	created, _ := service.CreateForOrg(context.Background(), "testorg", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL: "https://example.com/webhook",
		},
	})

	active := false
	updated, err := service.UpdateForOrg(context.Background(), "testorg", created.ID, &webhooks.UpdateIn{
		Active: &active,
	})
	if err != nil {
		t.Fatalf("UpdateForOrg failed: %v", err)
	}

	if updated.Active {
		t.Error("expected webhook to be inactive after update")
	}
}

func TestService_DeleteForOrg_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	created, _ := service.CreateForOrg(context.Background(), "testorg", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL: "https://example.com/webhook",
		},
	})

	err := service.DeleteForOrg(context.Background(), "testorg", created.ID)
	if err != nil {
		t.Fatalf("DeleteForOrg failed: %v", err)
	}

	// Verify deleted
	_, err = service.GetForOrg(context.Background(), "testorg", created.ID)
	if err != webhooks.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// URL Population Tests

func TestService_PopulateRepoURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	hook := createTestWebhookForRepo(t, service, "owner", "testrepo")

	if hook.URL == "" {
		t.Error("expected URL to be set")
	}
	if hook.TestURL == "" {
		t.Error("expected TestURL to be set")
	}
	if hook.PingURL == "" {
		t.Error("expected PingURL to be set")
	}
}

func TestService_PopulateOrgURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	hook, _ := service.CreateForOrg(context.Background(), "testorg", &webhooks.CreateIn{
		Config: &webhooks.Config{
			URL: "https://example.com/webhook",
		},
	})

	if hook.URL == "" {
		t.Error("expected URL to be set")
	}
	if hook.TestURL == "" {
		t.Error("expected TestURL to be set")
	}
	if hook.PingURL == "" {
		t.Error("expected PingURL to be set")
	}
}

// Integration Test - Webhooks Isolated Between Repos and Orgs

func TestService_WebhooksIsolation(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "repo1")
	createTestRepo(t, store, owner, "repo2")
	createTestOrg(t, store, "testorg")

	// Create webhooks for different owners
	repoHook1 := createTestWebhookForRepo(t, service, "owner", "repo1")
	repoHook2 := createTestWebhookForRepo(t, service, "owner", "repo2")
	orgHook, _ := service.CreateForOrg(context.Background(), "testorg", &webhooks.CreateIn{
		Config: &webhooks.Config{URL: "https://example.com/webhook"},
	})

	// Verify isolation
	repo1Hooks, _ := service.ListForRepo(context.Background(), "owner", "repo1", nil)
	repo2Hooks, _ := service.ListForRepo(context.Background(), "owner", "repo2", nil)
	orgHooks, _ := service.ListForOrg(context.Background(), "testorg", nil)

	if len(repo1Hooks) != 1 || repo1Hooks[0].ID != repoHook1.ID {
		t.Error("repo1 hooks not isolated correctly")
	}
	if len(repo2Hooks) != 1 || repo2Hooks[0].ID != repoHook2.ID {
		t.Error("repo2 hooks not isolated correctly")
	}
	if len(orgHooks) != 1 || orgHooks[0].ID != orgHook.ID {
		t.Error("org hooks not isolated correctly")
	}

	// Verify can't get repo hook from different repo
	_, err := service.GetForRepo(context.Background(), "owner", "repo2", repoHook1.ID)
	if err != webhooks.ErrNotFound {
		t.Error("expected ErrNotFound when getting hook from wrong repo")
	}

	// Verify can't get org hook from repo
	_, err = service.GetForRepo(context.Background(), "owner", "repo1", orgHook.ID)
	if err != webhooks.ErrNotFound {
		t.Error("expected ErrNotFound when getting org hook from repo")
	}
}
