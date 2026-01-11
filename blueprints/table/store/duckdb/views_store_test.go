package duckdb_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestViewsStoreBehaviors(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)

	t.Run("Default type and config fields", func(t *testing.T) {
		view := &views.View{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Name:      "Default View",
			CreatedBy: user.ID,
		}

		if err := store.Views().Create(ctx, view); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Views().GetByID(ctx, view.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if got.Type != views.TypeGrid {
			t.Errorf("Expected default view type grid, got %s", got.Type)
		}
	})

	t.Run("Update full configuration", func(t *testing.T) {
		view := createTestView(t, store, tbl, user, "Config View")
		view.Config = json.RawMessage(`{"density":"compact"}`)
		view.Filters = []views.Filter{{FieldID: "fld", Operator: "contains", Value: "test"}}
		view.Sorts = []views.SortSpec{{FieldID: "fld", Direction: "desc"}}
		view.Groups = []views.GroupSpec{{FieldID: "fld", Direction: "asc", Collapsed: true}}
		view.FieldConfig = []views.FieldViewConfig{{FieldID: "fld", Visible: true, Width: 220, Position: 1}}

		if err := store.Views().Update(ctx, view); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, err := store.Views().GetByID(ctx, view.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if string(got.Config) != string(view.Config) {
			t.Errorf("Expected config %s, got %s", string(view.Config), string(got.Config))
		}
		if len(got.Filters) != 1 || len(got.Sorts) != 1 || len(got.Groups) != 1 || len(got.FieldConfig) != 1 {
			t.Errorf("Expected all config arrays to be set")
		}
	})
}
