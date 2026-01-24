package sqlite

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

func TestKnowledgeStore_CreateEntity(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		Name:        "Go",
		Type:        "programming_language",
		Description: "A statically typed programming language",
		Image:       "https://go.dev/logo.png",
		Facts: map[string]any{
			"Created by": "Google",
			"Year":       2009,
		},
		Links: []store.Link{
			{Title: "Official Site", URL: "https://go.dev"},
		},
	}

	if err := knowledge.CreateEntity(ctx, entity); err != nil {
		t.Fatalf("CreateEntity() error = %v", err)
	}

	if entity.ID == "" {
		t.Error("expected ID to be set")
	}
	if entity.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestKnowledgeStore_GetEntity_ExactMatch(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		Name:        "Python",
		Type:        "programming_language",
		Description: "A high-level programming language",
	}

	if err := knowledge.CreateEntity(ctx, entity); err != nil {
		t.Fatalf("CreateEntity() error = %v", err)
	}

	// Exact match query
	panel, err := knowledge.GetEntity(ctx, "Python")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}

	if panel == nil {
		t.Fatal("expected panel, got nil")
	}
	if panel.Title != "Python" {
		t.Errorf("Title = %q, want 'Python'", panel.Title)
	}
	if panel.Subtitle != "programming_language" {
		t.Errorf("Subtitle = %q, want 'programming_language'", panel.Subtitle)
	}
}

func TestKnowledgeStore_GetEntity_CaseInsensitive(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		Name:        "JavaScript",
		Type:        "programming_language",
		Description: "A dynamic programming language",
	}

	if err := knowledge.CreateEntity(ctx, entity); err != nil {
		t.Fatalf("CreateEntity() error = %v", err)
	}

	// Different case query
	panel, err := knowledge.GetEntity(ctx, "javascript")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}

	if panel == nil {
		t.Fatal("expected case-insensitive match")
	}
	if panel.Title != "JavaScript" {
		t.Errorf("Title = %q, want 'JavaScript'", panel.Title)
	}
}

func TestKnowledgeStore_GetEntity_FTSFallback(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		Name:        "PostgreSQL",
		Type:        "database",
		Description: "An open source relational database management system",
	}

	if err := knowledge.CreateEntity(ctx, entity); err != nil {
		t.Fatalf("CreateEntity() error = %v", err)
	}

	// Partial match via FTS
	panel, err := knowledge.GetEntity(ctx, "Postgres")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}

	if panel == nil {
		t.Fatal("expected FTS match")
	}
}

func TestKnowledgeStore_GetEntity_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	panel, err := knowledge.GetEntity(ctx, "NonexistentEntity")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}

	if panel != nil {
		t.Errorf("expected nil for nonexistent entity, got %+v", panel)
	}
}

func TestKnowledgeStore_GetEntity_WithFacts(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		Name:        "Rust",
		Type:        "programming_language",
		Description: "A systems programming language",
		Facts: map[string]any{
			"Designed by": "Mozilla",
			"First appeared": 2010,
		},
	}

	if err := knowledge.CreateEntity(ctx, entity); err != nil {
		t.Fatalf("CreateEntity() error = %v", err)
	}

	panel, err := knowledge.GetEntity(ctx, "Rust")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}

	if len(panel.Facts) != 2 {
		t.Errorf("len(Facts) = %d, want 2", len(panel.Facts))
	}
}

func TestKnowledgeStore_GetEntity_WithLinks(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		Name:        "Docker",
		Type:        "software",
		Description: "Container platform",
		Links: []store.Link{
			{Title: "Website", URL: "https://docker.com"},
			{Title: "Hub", URL: "https://hub.docker.com"},
		},
	}

	if err := knowledge.CreateEntity(ctx, entity); err != nil {
		t.Fatalf("CreateEntity() error = %v", err)
	}

	panel, err := knowledge.GetEntity(ctx, "Docker")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}

	if len(panel.Links) != 2 {
		t.Errorf("len(Links) = %d, want 2", len(panel.Links))
	}
}

func TestKnowledgeStore_UpdateEntity(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		Name:        "Kubernetes",
		Type:        "software",
		Description: "Container orchestration",
	}

	if err := knowledge.CreateEntity(ctx, entity); err != nil {
		t.Fatalf("CreateEntity() error = %v", err)
	}

	// Update
	entity.Description = "Production-grade container orchestration"
	entity.Image = "https://kubernetes.io/logo.png"
	entity.Facts = map[string]any{"Also known as": "K8s"}

	if err := knowledge.UpdateEntity(ctx, entity); err != nil {
		t.Fatalf("UpdateEntity() error = %v", err)
	}

	// Verify
	panel, err := knowledge.GetEntity(ctx, "Kubernetes")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}

	if panel.Description != "Production-grade container orchestration" {
		t.Errorf("Description = %q, want updated value", panel.Description)
	}
}

func TestKnowledgeStore_UpdateEntity_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		ID:   "nonexistent",
		Name: "Test",
	}

	err := knowledge.UpdateEntity(ctx, entity)
	if err == nil {
		t.Error("expected error for nonexistent entity")
	}
}

func TestKnowledgeStore_DeleteEntity(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entity := &store.Entity{
		Name:        "ToDelete",
		Type:        "test",
		Description: "Will be deleted",
	}

	if err := knowledge.CreateEntity(ctx, entity); err != nil {
		t.Fatalf("CreateEntity() error = %v", err)
	}

	if err := knowledge.DeleteEntity(ctx, entity.ID); err != nil {
		t.Fatalf("DeleteEntity() error = %v", err)
	}

	// Verify deleted
	panel, err := knowledge.GetEntity(ctx, "ToDelete")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}

	if panel != nil {
		t.Error("expected nil after deletion")
	}
}

func TestKnowledgeStore_DeleteEntity_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	err := knowledge.DeleteEntity(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent entity")
	}
}

func TestKnowledgeStore_ListEntities(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entities := []*store.Entity{
		{Name: "Entity A", Type: "type1", Description: "First"},
		{Name: "Entity B", Type: "type2", Description: "Second"},
		{Name: "Entity C", Type: "type1", Description: "Third"},
	}

	for _, e := range entities {
		if err := knowledge.CreateEntity(ctx, e); err != nil {
			t.Fatalf("CreateEntity() error = %v", err)
		}
	}

	// List all
	list, err := knowledge.ListEntities(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("ListEntities() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("len(list) = %d, want 3", len(list))
	}
}

func TestKnowledgeStore_ListEntities_ByType(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	entities := []*store.Entity{
		{Name: "Lang A", Type: "language", Description: "First"},
		{Name: "Lang B", Type: "language", Description: "Second"},
		{Name: "DB A", Type: "database", Description: "Third"},
	}

	for _, e := range entities {
		if err := knowledge.CreateEntity(ctx, e); err != nil {
			t.Fatalf("CreateEntity() error = %v", err)
		}
	}

	// List by type
	list, err := knowledge.ListEntities(ctx, "language", 10, 0)
	if err != nil {
		t.Fatalf("ListEntities() error = %v", err)
	}

	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}

	for _, e := range list {
		if e.Type != "language" {
			t.Errorf("Type = %q, want 'language'", e.Type)
		}
	}
}

func TestKnowledgeStore_ListEntities_Pagination(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	knowledge := s.Knowledge()

	for i := 0; i < 10; i++ {
		entity := &store.Entity{
			Name:        "Entity " + string(rune('A'+i)),
			Type:        "test",
			Description: "Test entity",
		}
		if err := knowledge.CreateEntity(ctx, entity); err != nil {
			t.Fatalf("CreateEntity() error = %v", err)
		}
	}

	// First page
	list1, err := knowledge.ListEntities(ctx, "", 5, 0)
	if err != nil {
		t.Fatalf("ListEntities() error = %v", err)
	}

	if len(list1) != 5 {
		t.Errorf("page 1: len(list) = %d, want 5", len(list1))
	}

	// Second page
	list2, err := knowledge.ListEntities(ctx, "", 5, 5)
	if err != nil {
		t.Fatalf("ListEntities() error = %v", err)
	}

	if len(list2) != 5 {
		t.Errorf("page 2: len(list) = %d, want 5", len(list2))
	}
}
