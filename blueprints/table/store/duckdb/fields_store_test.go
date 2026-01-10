package duckdb_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestFieldsStoreBehaviors(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)

	t.Run("Defaults and GetByID", func(t *testing.T) {
		options := json.RawMessage(`{"precision":2}`)
		field := &fields.Field{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Name:      "Budget",
			Type:      fields.TypeCurrency,
			Options:   options,
			CreatedBy: user.ID,
		}

		if err := store.Fields().Create(ctx, field); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Fields().GetByID(ctx, field.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if got.Width != 200 {
			t.Errorf("Expected default width 200, got %d", got.Width)
		}
		if string(got.Options) != string(options) {
			t.Errorf("Expected options %s, got %s", string(options), string(got.Options))
		}
	})

	t.Run("Reorder updates positions", func(t *testing.T) {
		f1 := createTestField(t, store, tbl, "One", fields.TypeSingleLineText, user)
		f2 := createTestField(t, store, tbl, "Two", fields.TypeSingleLineText, user)
		f3 := createTestField(t, store, tbl, "Three", fields.TypeSingleLineText, user)

		if err := store.Fields().Reorder(ctx, tbl.ID, []string{f3.ID, f1.ID, f2.ID}); err != nil {
			t.Fatalf("Reorder failed: %v", err)
		}

		list, err := store.Fields().ListByTable(ctx, tbl.ID)
		if err != nil {
			t.Fatalf("ListByTable failed: %v", err)
		}
		if len(list) < 3 {
			t.Fatalf("Expected at least 3 fields, got %d", len(list))
		}
		if list[0].ID != f3.ID || list[1].ID != f1.ID || list[2].ID != f2.ID {
			t.Errorf("Unexpected field ordering after reorder")
		}
	})

	t.Run("Select choices lifecycle", func(t *testing.T) {
		field := createTestField(t, store, tbl, "Status", fields.TypeSingleSelect, user)

		if err := store.Fields().AddSelectChoice(ctx, &fields.SelectChoice{
			FieldID: field.ID,
			Name:    "Todo",
		}); err != nil {
			t.Fatalf("AddSelectChoice failed: %v", err)
		}

		choices, err := store.Fields().ListSelectChoices(ctx, field.ID)
		if err != nil {
			t.Fatalf("ListSelectChoices failed: %v", err)
		}
		if len(choices) != 1 {
			t.Fatalf("Expected 1 choice, got %d", len(choices))
		}

		gotChoice, err := store.Fields().GetSelectChoice(ctx, choices[0].ID)
		if err != nil {
			t.Fatalf("GetSelectChoice failed: %v", err)
		}
		if gotChoice.Color == "" {
			t.Error("Expected default color for select choice")
		}

		if err := store.Fields().UpdateSelectChoice(ctx, gotChoice.ID, fields.UpdateChoiceIn{Color: "#111111"}); err != nil {
			t.Fatalf("UpdateSelectChoice failed: %v", err)
		}

		if err := store.Fields().DeleteSelectChoice(ctx, gotChoice.ID); err != nil {
			t.Fatalf("DeleteSelectChoice failed: %v", err)
		}

		if _, err := store.Fields().GetSelectChoice(ctx, gotChoice.ID); err != fields.ErrChoiceNotFound {
			t.Errorf("Expected ErrChoiceNotFound, got %v", err)
		}
	})
}
