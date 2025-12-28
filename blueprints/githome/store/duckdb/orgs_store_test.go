package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/oklog/ulid/v2"
)

func createTestOrg(t *testing.T, store *OrgsStore) *orgs.Organization {
	t.Helper()
	id := ulid.Make().String()
	o := &orgs.Organization{
		ID:          id,
		Name:        "org-" + id[len(id)-8:],
		Slug:        "org-" + id[len(id)-8:],
		DisplayName: "Test Organization",
		Description: "A test organization",
		AvatarURL:   "https://example.com/avatar.png",
		Website:     "https://example.com",
		Email:       id + "@example.com",
		Location:    "San Francisco",
		IsVerified:  false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Create(context.Background(), o); err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	return o
}

// =============================================================================
// Org CRUD Tests
// =============================================================================

func TestOrgsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())

	o := &orgs.Organization{
		ID:          ulid.Make().String(),
		Name:        "acme-corp",
		Slug:        "acme-corp",
		DisplayName: "Acme Corporation",
		Description: "Building the future",
		Email:       "contact@acme.com",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := orgsStore.Create(context.Background(), o)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := orgsStore.GetByID(context.Background(), o.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected org to be created")
	}
	if got.Name != "acme-corp" {
		t.Errorf("got name %q, want %q", got.Name, "acme-corp")
	}
	if got.DisplayName != "Acme Corporation" {
		t.Errorf("got display_name %q, want %q", got.DisplayName, "Acme Corporation")
	}
}

func TestOrgsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	o := createTestOrg(t, orgsStore)

	got, err := orgsStore.GetByID(context.Background(), o.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected org")
	}
	if got.ID != o.ID {
		t.Errorf("got ID %q, want %q", got.ID, o.ID)
	}
}

func TestOrgsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())

	got, err := orgsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent org")
	}
}

func TestOrgsStore_GetBySlug(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	o := createTestOrg(t, orgsStore)

	got, err := orgsStore.GetBySlug(context.Background(), o.Slug)
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected org")
	}
	if got.Slug != o.Slug {
		t.Errorf("got slug %q, want %q", got.Slug, o.Slug)
	}
}

func TestOrgsStore_GetBySlug_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())

	got, err := orgsStore.GetBySlug(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent org slug")
	}
}

func TestOrgsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	o := createTestOrg(t, orgsStore)

	o.DisplayName = "Updated Name"
	o.Description = "Updated description"
	o.IsVerified = true

	err := orgsStore.Update(context.Background(), o)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := orgsStore.GetByID(context.Background(), o.ID)
	if got.DisplayName != "Updated Name" {
		t.Errorf("got display_name %q, want %q", got.DisplayName, "Updated Name")
	}
	if !got.IsVerified {
		t.Error("expected org to be verified")
	}
}

func TestOrgsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	o := createTestOrg(t, orgsStore)

	err := orgsStore.Delete(context.Background(), o.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := orgsStore.GetByID(context.Background(), o.ID)
	if got != nil {
		t.Error("expected org to be deleted")
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestOrgsStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestOrg(t, orgsStore)
	}

	list, err := orgsStore.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d orgs, want 5", len(list))
	}
}

func TestOrgsStore_List_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())

	for i := 0; i < 10; i++ {
		createTestOrg(t, orgsStore)
	}

	page1, _ := orgsStore.List(context.Background(), 3, 0)
	page2, _ := orgsStore.List(context.Background(), 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d orgs on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d orgs on page 2, want 3", len(page2))
	}
}

// =============================================================================
// Member Tests
// =============================================================================

func TestOrgsStore_AddMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)

	member := &orgs.Member{
		OrgID:     org.ID,
		UserID:    user.ID,
		Role:      "member",
		CreatedAt: time.Now(),
	}

	err := orgsStore.AddMember(context.Background(), member)
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	got, _ := orgsStore.GetMember(context.Background(), org.ID, user.ID)
	if got == nil {
		t.Fatal("expected member")
	}
	if got.Role != "member" {
		t.Errorf("got role %q, want %q", got.Role, "member")
	}
}

func TestOrgsStore_GetMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)

	member := &orgs.Member{
		OrgID:     org.ID,
		UserID:    user.ID,
		Role:      "admin",
		CreatedAt: time.Now(),
	}
	orgsStore.AddMember(context.Background(), member)

	got, err := orgsStore.GetMember(context.Background(), org.ID, user.ID)
	if err != nil {
		t.Fatalf("GetMember failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected member")
	}
	if got.Role != "admin" {
		t.Errorf("got role %q, want %q", got.Role, "admin")
	}
}

func TestOrgsStore_GetMember_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())

	got, err := orgsStore.GetMember(context.Background(), "org-1", "user-1")
	if err != nil {
		t.Fatalf("GetMember failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent member")
	}
}

func TestOrgsStore_UpdateMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)

	member := &orgs.Member{
		OrgID:     org.ID,
		UserID:    user.ID,
		Role:      "member",
		CreatedAt: time.Now(),
	}
	orgsStore.AddMember(context.Background(), member)

	member.Role = "admin"
	err := orgsStore.UpdateMember(context.Background(), member)
	if err != nil {
		t.Fatalf("UpdateMember failed: %v", err)
	}

	got, _ := orgsStore.GetMember(context.Background(), org.ID, user.ID)
	if got.Role != "admin" {
		t.Errorf("got role %q, want %q", got.Role, "admin")
	}
}

func TestOrgsStore_RemoveMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)

	member := &orgs.Member{
		OrgID:     org.ID,
		UserID:    user.ID,
		Role:      "member",
		CreatedAt: time.Now(),
	}
	orgsStore.AddMember(context.Background(), member)

	err := orgsStore.RemoveMember(context.Background(), org.ID, user.ID)
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	got, _ := orgsStore.GetMember(context.Background(), org.ID, user.ID)
	if got != nil {
		t.Error("expected member to be removed")
	}
}

func TestOrgsStore_ListMembers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())

	org := createTestOrg(t, orgsStore)

	for i := 0; i < 5; i++ {
		user := createTestUser(t, usersStore)
		member := &orgs.Member{
			OrgID:     org.ID,
			UserID:    user.ID,
			Role:      "member",
			CreatedAt: time.Now(),
		}
		orgsStore.AddMember(context.Background(), member)
	}

	list, err := orgsStore.ListMembers(context.Background(), org.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d members, want 5", len(list))
	}
}

func TestOrgsStore_ListByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())

	user := createTestUser(t, usersStore)

	for i := 0; i < 3; i++ {
		org := createTestOrg(t, orgsStore)
		member := &orgs.Member{
			OrgID:     org.ID,
			UserID:    user.ID,
			Role:      "member",
			CreatedAt: time.Now(),
		}
		orgsStore.AddMember(context.Background(), member)
	}

	list, err := orgsStore.ListByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d orgs, want 3", len(list))
	}
}

// Verify interface compliance
var _ orgs.Store = (*OrgsStore)(nil)
