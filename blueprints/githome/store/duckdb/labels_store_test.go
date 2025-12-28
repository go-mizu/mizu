package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

func createTestLabel(t *testing.T, store *LabelsStore, repoID int64, name string) *labels.Label {
	t.Helper()
	l := &labels.Label{
		RepoID:      repoID,
		Name:        name,
		Color:       "f29513",
		Description: "Test label " + name,
	}
	if err := store.Create(context.Background(), l); err != nil {
		t.Fatalf("failed to create test label: %v", err)
	}
	return l
}

func TestLabelsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	labelsStore := NewLabelsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	// Create owner user first
	u := &users.User{Login: "labelowner", Name: "Label Owner", Email: "label@example.com", Type: "User", PasswordHash: "hash"}
	if err := usersStore.Create(context.Background(), u); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create repo
	r := &repos.Repository{Name: "labelrepo", FullName: "labelowner/labelrepo", OwnerID: u.ID}
	if err := reposStore.Create(context.Background(), r); err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	// Create label
	l := &labels.Label{
		RepoID:      r.ID,
		Name:        "bug",
		Color:       "d73a4a",
		Description: "Something isn't working",
	}
	err := labelsStore.Create(context.Background(), l)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if l.ID == 0 {
		t.Error("expected ID to be set")
	}
	if l.NodeID == "" {
		t.Error("expected NodeID to be set")
	}

	got, err := labelsStore.GetByID(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected label to be created")
	}
	if got.Name != l.Name {
		t.Errorf("got name %q, want %q", got.Name, l.Name)
	}
}

func TestLabelsStore_GetByName(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	labelsStore := NewLabelsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := &users.User{Login: "labelowner2", Name: "Label Owner 2", Email: "label2@example.com", Type: "User", PasswordHash: "hash"}
	usersStore.Create(context.Background(), u)
	r := &repos.Repository{Name: "labelrepo2", FullName: "labelowner2/labelrepo2", OwnerID: u.ID}
	reposStore.Create(context.Background(), r)

	l := &labels.Label{RepoID: r.ID, Name: "enhancement", Color: "a2eeef"}
	labelsStore.Create(context.Background(), l)

	got, err := labelsStore.GetByName(context.Background(), r.ID, "enhancement")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected label")
	}
	if got.Name != "enhancement" {
		t.Errorf("got name %q, want %q", got.Name, "enhancement")
	}
}

func TestLabelsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	labelsStore := NewLabelsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := &users.User{Login: "labelowner3", Name: "Label Owner 3", Email: "label3@example.com", Type: "User", PasswordHash: "hash"}
	usersStore.Create(context.Background(), u)
	r := &repos.Repository{Name: "labelrepo3", FullName: "labelowner3/labelrepo3", OwnerID: u.ID}
	reposStore.Create(context.Background(), r)

	l := &labels.Label{RepoID: r.ID, Name: "documentation", Color: "0075ca"}
	labelsStore.Create(context.Background(), l)

	newName := "docs"
	newColor := "0052cc"
	newDesc := "Documentation improvements"
	err := labelsStore.Update(context.Background(), l.ID, &labels.UpdateIn{
		NewName:     &newName,
		Color:       &newColor,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := labelsStore.GetByID(context.Background(), l.ID)
	if got.Name != newName {
		t.Errorf("got name %q, want %q", got.Name, newName)
	}
	if got.Color != newColor {
		t.Errorf("got color %q, want %q", got.Color, newColor)
	}
}

func TestLabelsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	labelsStore := NewLabelsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := &users.User{Login: "labelowner4", Name: "Label Owner 4", Email: "label4@example.com", Type: "User", PasswordHash: "hash"}
	usersStore.Create(context.Background(), u)
	r := &repos.Repository{Name: "labelrepo4", FullName: "labelowner4/labelrepo4", OwnerID: u.ID}
	reposStore.Create(context.Background(), r)

	l := &labels.Label{RepoID: r.ID, Name: "wontfix", Color: "ffffff"}
	labelsStore.Create(context.Background(), l)

	err := labelsStore.Delete(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := labelsStore.GetByID(context.Background(), l.ID)
	if got != nil {
		t.Error("expected label to be deleted")
	}
}

func TestLabelsStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	labelsStore := NewLabelsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := &users.User{Login: "labelowner5", Name: "Label Owner 5", Email: "label5@example.com", Type: "User", PasswordHash: "hash"}
	usersStore.Create(context.Background(), u)
	r := &repos.Repository{Name: "labelrepo5", FullName: "labelowner5/labelrepo5", OwnerID: u.ID}
	reposStore.Create(context.Background(), r)

	labelNames := []string{"bug", "enhancement", "documentation"}
	for _, name := range labelNames {
		labelsStore.Create(context.Background(), &labels.Label{RepoID: r.ID, Name: name, Color: "000000"})
	}

	list, err := labelsStore.List(context.Background(), r.ID, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d labels, want 3", len(list))
	}
}
