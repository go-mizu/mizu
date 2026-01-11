package duckdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/table/feature/operations"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestOperationsStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)

	t.Run("Create and GetByID", func(t *testing.T) {
		op := &operations.Operation{
			ID:       ulid.New(),
			TableID:  tbl.ID,
			OpType:   operations.OpCreateRecord,
			UserID:   user.ID,
			NewValue: []byte(`{"id":"rec1"}`),
		}

		if err := store.Operations().Create(ctx, op); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Operations().GetByID(ctx, op.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if got.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set")
		}
		if string(got.NewValue) != string(op.NewValue) {
			t.Errorf("Expected new_value %s, got %s", string(op.NewValue), string(got.NewValue))
		}
	})

	t.Run("List filters and limits", func(t *testing.T) {
		now := time.Now()
		op1 := &operations.Operation{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			RecordID:  "rec1",
			OpType:    operations.OpUpdateRecord,
			UserID:    user.ID,
			Timestamp: now.Add(-2 * time.Minute),
		}
		op2 := &operations.Operation{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			RecordID:  "rec2",
			OpType:    operations.OpUpdateRecord,
			UserID:    user.ID,
			Timestamp: now.Add(-1 * time.Minute),
		}
		op3 := &operations.Operation{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			RecordID:  "rec3",
			OpType:    operations.OpUpdateRecord,
			UserID:    user.ID,
			Timestamp: now,
		}

		if err := store.Operations().CreateBatch(ctx, []*operations.Operation{op1, op2, op3}); err != nil {
			t.Fatalf("CreateBatch failed: %v", err)
		}

		list, err := store.Operations().ListByTable(ctx, tbl.ID, operations.ListOpts{
			Since: now.Add(-90 * time.Second),
			Until: now.Add(10 * time.Second),
			Limit: 2,
		})
		if err != nil {
			t.Fatalf("ListByTable failed: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("Expected 2 ops, got %d", len(list))
		}

		byRecord, err := store.Operations().ListByRecord(ctx, "rec1", operations.ListOpts{Limit: 10})
		if err != nil {
			t.Fatalf("ListByRecord failed: %v", err)
		}
		if len(byRecord) != 1 {
			t.Errorf("Expected 1 record op, got %d", len(byRecord))
		}

		byUser, err := store.Operations().ListByUser(ctx, user.ID, operations.ListOpts{Limit: 10})
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}
		if len(byUser) < 3 {
			t.Errorf("Expected at least 3 user ops, got %d", len(byUser))
		}
	})
}
