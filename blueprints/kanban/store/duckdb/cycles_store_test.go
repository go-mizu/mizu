package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/cycles"
	"github.com/oklog/ulid/v2"
)

func createTestCycle(t *testing.T, store *CyclesStore, teamID string, number int, status string) *cycles.Cycle {
	t.Helper()
	c := &cycles.Cycle{
		ID:        ulid.Make().String(),
		TeamID:    teamID,
		Number:    number,
		Name:      "Cycle " + ulid.Make().String()[:4],
		Status:    status,
		StartDate: time.Now(),
		EndDate:   time.Now().Add(14 * 24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Create(context.Background(), c); err != nil {
		t.Fatalf("failed to create test cycle: %v", err)
	}
	return c
}

func TestCyclesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	c := &cycles.Cycle{
		ID:        ulid.Make().String(),
		TeamID:    team.ID,
		Number:    1,
		Name:      "Cycle 1",
		Status:    cycles.StatusPlanning,
		StartDate: time.Now(),
		EndDate:   time.Now().Add(14 * 24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := cyclesStore.Create(context.Background(), c)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := cyclesStore.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected cycle to be created")
	}
	if got.Number != 1 {
		t.Errorf("got number %d, want 1", got.Number)
	}
}

func TestCyclesStore_Create_DuplicateNumber(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusPlanning)

	c2 := &cycles.Cycle{
		ID:        ulid.Make().String(),
		TeamID:    team.ID,
		Number:    1, // same number
		Name:      "Cycle Dup",
		Status:    cycles.StatusPlanning,
		StartDate: time.Now(),
		EndDate:   time.Now().Add(14 * 24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := cyclesStore.Create(context.Background(), c2)
	if err == nil {
		t.Error("expected error for duplicate number")
	}
}

func TestCyclesStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	c := createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusPlanning)

	got, err := cyclesStore.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected cycle")
	}
	if got.ID != c.ID {
		t.Errorf("got ID %q, want %q", got.ID, c.ID)
	}
}

func TestCyclesStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	cyclesStore := NewCyclesStore(store.DB())

	got, err := cyclesStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent cycle")
	}
}

func TestCyclesStore_GetByNumber(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	c := createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusPlanning)

	got, err := cyclesStore.GetByNumber(context.Background(), team.ID, 1)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected cycle")
	}
	if got.ID != c.ID {
		t.Errorf("got ID %q, want %q", got.ID, c.ID)
	}
}

func TestCyclesStore_GetByNumber_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	got, err := cyclesStore.GetByNumber(context.Background(), team.ID, 999)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent number")
	}
}

func TestCyclesStore_ListByTeam(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusCompleted)
	createTestCycle(t, cyclesStore, team.ID, 2, cycles.StatusActive)

	list, err := cyclesStore.ListByTeam(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("ListByTeam failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d cycles, want 2", len(list))
	}
	// Should be ordered by number descending
	if list[0].Number != 2 {
		t.Errorf("expected first cycle to have number 2, got %d", list[0].Number)
	}
}

func TestCyclesStore_ListByTeam_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	list, err := cyclesStore.ListByTeam(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("ListByTeam failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d cycles, want 0", len(list))
	}
}

func TestCyclesStore_GetActive(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusCompleted)
	activeCycle := createTestCycle(t, cyclesStore, team.ID, 2, cycles.StatusActive)

	got, err := cyclesStore.GetActive(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected active cycle")
	}
	if got.ID != activeCycle.ID {
		t.Errorf("got ID %q, want %q", got.ID, activeCycle.ID)
	}
}

func TestCyclesStore_GetActive_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusPlanning)

	got, err := cyclesStore.GetActive(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil when no active cycle")
	}
}

func TestCyclesStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	c := createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusPlanning)

	newName := "Updated Cycle"
	err := cyclesStore.Update(context.Background(), c.ID, &cycles.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := cyclesStore.GetByID(context.Background(), c.ID)
	if got.Name != newName {
		t.Errorf("got name %q, want %q", got.Name, newName)
	}
}

func TestCyclesStore_UpdateStatus(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	c := createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusPlanning)

	err := cyclesStore.UpdateStatus(context.Background(), c.ID, cycles.StatusActive)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, _ := cyclesStore.GetByID(context.Background(), c.ID)
	if got.Status != cycles.StatusActive {
		t.Errorf("got status %q, want %q", got.Status, cycles.StatusActive)
	}
}

func TestCyclesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	c := createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusPlanning)

	err := cyclesStore.Delete(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := cyclesStore.GetByID(context.Background(), c.ID)
	if got != nil {
		t.Error("expected cycle to be deleted")
	}
}

func TestCyclesStore_GetNextNumber(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	// First cycle should be 1
	num, err := cyclesStore.GetNextNumber(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if num != 1 {
		t.Errorf("got number %d, want 1", num)
	}

	// Create cycle 1
	createTestCycle(t, cyclesStore, team.ID, 1, cycles.StatusPlanning)

	// Next should be 2
	num, err = cyclesStore.GetNextNumber(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if num != 2 {
		t.Errorf("got number %d, want 2", num)
	}
}

func TestCyclesStore_GetNextNumber_FirstCycle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	num, err := cyclesStore.GetNextNumber(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if num != 1 {
		t.Errorf("got number %d, want 1 for first cycle", num)
	}
}
