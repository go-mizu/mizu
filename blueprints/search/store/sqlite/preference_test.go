package sqlite

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

func TestPreferenceStore_SetPreference(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	p := &store.UserPreference{
		Domain: "example.com",
		Action: "upvote",
	}

	if err := pref.SetPreference(ctx, p); err != nil {
		t.Fatalf("SetPreference() error = %v", err)
	}

	if p.ID == "" {
		t.Error("expected ID to be set")
	}
	if p.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestPreferenceStore_SetPreference_Upsert(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	// First preference
	p1 := &store.UserPreference{
		Domain: "test.com",
		Action: "upvote",
	}
	if err := pref.SetPreference(ctx, p1); err != nil {
		t.Fatalf("first SetPreference() error = %v", err)
	}

	// Update with same domain
	p2 := &store.UserPreference{
		Domain: "test.com",
		Action: "downvote",
	}
	if err := pref.SetPreference(ctx, p2); err != nil {
		t.Fatalf("second SetPreference() error = %v", err)
	}

	// Verify only one preference exists
	prefs, err := pref.GetPreferences(ctx)
	if err != nil {
		t.Fatalf("GetPreferences() error = %v", err)
	}

	if len(prefs) != 1 {
		t.Errorf("len(prefs) = %d, want 1", len(prefs))
	}

	if prefs[0].Action != "downvote" {
		t.Errorf("Action = %q, want 'downvote'", prefs[0].Action)
	}
}

func TestPreferenceStore_GetPreferences(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	prefs := []*store.UserPreference{
		{Domain: "site1.com", Action: "upvote"},
		{Domain: "site2.com", Action: "downvote"},
		{Domain: "site3.com", Action: "block"},
	}

	for _, p := range prefs {
		if err := pref.SetPreference(ctx, p); err != nil {
			t.Fatalf("SetPreference() error = %v", err)
		}
	}

	list, err := pref.GetPreferences(ctx)
	if err != nil {
		t.Fatalf("GetPreferences() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("len(list) = %d, want 3", len(list))
	}
}

func TestPreferenceStore_DeletePreference(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	p := &store.UserPreference{
		Domain: "delete-me.com",
		Action: "upvote",
	}

	if err := pref.SetPreference(ctx, p); err != nil {
		t.Fatalf("SetPreference() error = %v", err)
	}

	if err := pref.DeletePreference(ctx, "delete-me.com"); err != nil {
		t.Fatalf("DeletePreference() error = %v", err)
	}

	// Verify deleted
	prefs, err := pref.GetPreferences(ctx)
	if err != nil {
		t.Fatalf("GetPreferences() error = %v", err)
	}

	if len(prefs) != 0 {
		t.Errorf("len(prefs) = %d, want 0", len(prefs))
	}
}

func TestPreferenceStore_DeletePreference_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	err := pref.DeletePreference(ctx, "nonexistent.com")
	if err == nil {
		t.Error("expected error for nonexistent preference")
	}
}

func TestPreferenceStore_CreateLens(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	lens := &store.SearchLens{
		Name:        "Tech Forums",
		Description: "Search technology forums",
		Domains:     []string{"reddit.com", "hackernews.com"},
		Exclude:     []string{"spam.com"},
		Keywords:    []string{"programming"},
		IsPublic:    true,
	}

	if err := pref.CreateLens(ctx, lens); err != nil {
		t.Fatalf("CreateLens() error = %v", err)
	}

	if lens.ID == "" {
		t.Error("expected ID to be set")
	}
	if lens.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestPreferenceStore_GetLens(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	lens := &store.SearchLens{
		Name:        "Academic",
		Description: "Academic sources",
		Domains:     []string{"arxiv.org", "scholar.google.com"},
		IsPublic:    true,
	}

	if err := pref.CreateLens(ctx, lens); err != nil {
		t.Fatalf("CreateLens() error = %v", err)
	}

	retrieved, err := pref.GetLens(ctx, lens.ID)
	if err != nil {
		t.Fatalf("GetLens() error = %v", err)
	}

	if retrieved.Name != "Academic" {
		t.Errorf("Name = %q, want 'Academic'", retrieved.Name)
	}
	if len(retrieved.Domains) != 2 {
		t.Errorf("len(Domains) = %d, want 2", len(retrieved.Domains))
	}
}

func TestPreferenceStore_GetLens_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	_, err := pref.GetLens(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent lens")
	}
}

func TestPreferenceStore_ListLenses(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	lenses := []*store.SearchLens{
		{Name: "Lens A", Domains: []string{"a.com"}},
		{Name: "Lens B", Domains: []string{"b.com"}, IsBuiltIn: true},
		{Name: "Lens C", Domains: []string{"c.com"}},
	}

	for _, l := range lenses {
		if err := pref.CreateLens(ctx, l); err != nil {
			t.Fatalf("CreateLens() error = %v", err)
		}
	}

	list, err := pref.ListLenses(ctx)
	if err != nil {
		t.Fatalf("ListLenses() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("len(list) = %d, want 3", len(list))
	}

	// Built-in should come first
	if !list[0].IsBuiltIn {
		t.Error("expected built-in lens first")
	}
}

func TestPreferenceStore_UpdateLens(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	lens := &store.SearchLens{
		Name:    "Updateable",
		Domains: []string{"old.com"},
	}

	if err := pref.CreateLens(ctx, lens); err != nil {
		t.Fatalf("CreateLens() error = %v", err)
	}

	lens.Name = "Updated"
	lens.Domains = []string{"new.com", "another.com"}
	lens.Description = "New description"

	if err := pref.UpdateLens(ctx, lens); err != nil {
		t.Fatalf("UpdateLens() error = %v", err)
	}

	retrieved, err := pref.GetLens(ctx, lens.ID)
	if err != nil {
		t.Fatalf("GetLens() error = %v", err)
	}

	if retrieved.Name != "Updated" {
		t.Errorf("Name = %q, want 'Updated'", retrieved.Name)
	}
	if len(retrieved.Domains) != 2 {
		t.Errorf("len(Domains) = %d, want 2", len(retrieved.Domains))
	}
}

func TestPreferenceStore_UpdateLens_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	lens := &store.SearchLens{
		ID:   "nonexistent",
		Name: "Test",
	}

	err := pref.UpdateLens(ctx, lens)
	if err == nil {
		t.Error("expected error for nonexistent lens")
	}
}

func TestPreferenceStore_DeleteLens(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	lens := &store.SearchLens{
		Name:    "Deletable",
		Domains: []string{"test.com"},
	}

	if err := pref.CreateLens(ctx, lens); err != nil {
		t.Fatalf("CreateLens() error = %v", err)
	}

	if err := pref.DeleteLens(ctx, lens.ID); err != nil {
		t.Fatalf("DeleteLens() error = %v", err)
	}

	_, err := pref.GetLens(ctx, lens.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestPreferenceStore_DeleteLens_BuiltIn(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	lens := &store.SearchLens{
		Name:      "Built-in",
		Domains:   []string{"test.com"},
		IsBuiltIn: true,
	}

	if err := pref.CreateLens(ctx, lens); err != nil {
		t.Fatalf("CreateLens() error = %v", err)
	}

	err := pref.DeleteLens(ctx, lens.ID)
	if err == nil {
		t.Error("expected error when deleting built-in lens")
	}
}

func TestPreferenceStore_GetSettings(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	settings, err := pref.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	// Check defaults
	if settings.SafeSearch != "moderate" {
		t.Errorf("SafeSearch = %q, want 'moderate'", settings.SafeSearch)
	}
	if settings.ResultsPerPage != 10 {
		t.Errorf("ResultsPerPage = %d, want 10", settings.ResultsPerPage)
	}
	if settings.Region != "us" {
		t.Errorf("Region = %q, want 'us'", settings.Region)
	}
	if settings.Language != "en" {
		t.Errorf("Language = %q, want 'en'", settings.Language)
	}
	if settings.Theme != "system" {
		t.Errorf("Theme = %q, want 'system'", settings.Theme)
	}
}

func TestPreferenceStore_UpdateSettings(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	newSettings := &store.SearchSettings{
		SafeSearch:     "strict",
		ResultsPerPage: 20,
		Region:         "uk",
		Language:       "en-GB",
		Theme:          "dark",
		OpenInNewTab:   true,
		ShowThumbnails: false,
	}

	if err := pref.UpdateSettings(ctx, newSettings); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	retrieved, err := pref.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	if retrieved.SafeSearch != "strict" {
		t.Errorf("SafeSearch = %q, want 'strict'", retrieved.SafeSearch)
	}
	if retrieved.ResultsPerPage != 20 {
		t.Errorf("ResultsPerPage = %d, want 20", retrieved.ResultsPerPage)
	}
	if retrieved.Region != "uk" {
		t.Errorf("Region = %q, want 'uk'", retrieved.Region)
	}
	if !retrieved.OpenInNewTab {
		t.Error("OpenInNewTab = false, want true")
	}
	if retrieved.ShowThumbnails {
		t.Error("ShowThumbnails = true, want false")
	}
}

func TestPreferenceStore_UpdateSettings_Upsert(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	pref := s.Preference()

	// First update
	s1 := &store.SearchSettings{
		SafeSearch:     "off",
		ResultsPerPage: 25,
	}
	if err := pref.UpdateSettings(ctx, s1); err != nil {
		t.Fatalf("first UpdateSettings() error = %v", err)
	}

	// Second update
	s2 := &store.SearchSettings{
		SafeSearch:     "moderate",
		ResultsPerPage: 15,
		Theme:          "light",
	}
	if err := pref.UpdateSettings(ctx, s2); err != nil {
		t.Fatalf("second UpdateSettings() error = %v", err)
	}

	retrieved, err := pref.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	if retrieved.SafeSearch != "moderate" {
		t.Errorf("SafeSearch = %q, want 'moderate'", retrieved.SafeSearch)
	}
	if retrieved.ResultsPerPage != 15 {
		t.Errorf("ResultsPerPage = %d, want 15", retrieved.ResultsPerPage)
	}
	if retrieved.Theme != "light" {
		t.Errorf("Theme = %q, want 'light'", retrieved.Theme)
	}
}

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Error("boolToInt(true) != 1")
	}
	if boolToInt(false) != 0 {
		t.Error("boolToInt(false) != 0")
	}
}
