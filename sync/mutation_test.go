package sync

import (
	"context"
	"testing"
)

func TestMutation_Basic(t *testing.T) {
	m := Mutation{
		Name:      "todo/create",
		Scope:     "user:123",
		Args:      map[string]any{"title": "Test"},
		ClientID:  "client-1",
		ClientSeq: 1,
	}

	if m.Name != "todo/create" {
		t.Errorf("expected name todo/create, got %s", m.Name)
	}
	if m.Scope != "user:123" {
		t.Errorf("expected scope user:123, got %s", m.Scope)
	}
	if m.Args["title"] != "Test" {
		t.Errorf("expected args.title Test, got %v", m.Args["title"])
	}
}

func TestMutatorFunc(t *testing.T) {
	var calledWith Mutation
	var calledStore Store

	fn := MutatorFunc(func(ctx context.Context, store Store, m Mutation) ([]Change, error) {
		calledWith = m
		calledStore = store
		return []Change{{
			Scope:  m.Scope,
			Entity: "todo",
			ID:     "1",
			Op:     OpCreate,
		}}, nil
	})

	store := NewMemoryStore()
	mut := Mutation{Name: "test", Scope: "test-scope"}

	changes, err := fn.Apply(context.Background(), store, mut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calledWith.Name != "test" {
		t.Errorf("expected mutation name test, got %s", calledWith.Name)
	}
	if calledStore != store {
		t.Error("expected store to be passed through")
	}
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}
}

func TestMutatorMap_Register(t *testing.T) {
	mm := NewMutatorMap()

	handler1Called := false
	handler2Called := false

	mm.Register("action1", func(ctx context.Context, store Store, m Mutation) ([]Change, error) {
		handler1Called = true
		return nil, nil
	})
	mm.Register("action2", func(ctx context.Context, store Store, m Mutation) ([]Change, error) {
		handler2Called = true
		return nil, nil
	})

	store := NewMemoryStore()

	_, err := mm.Apply(context.Background(), store, Mutation{Name: "action1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handler1Called {
		t.Error("handler1 should have been called")
	}
	if handler2Called {
		t.Error("handler2 should not have been called")
	}
}

func TestMutatorMap_UnknownMutation(t *testing.T) {
	mm := NewMutatorMap()
	store := NewMemoryStore()

	_, err := mm.Apply(context.Background(), store, Mutation{Name: "unknown"})
	if err != ErrUnknownMutation {
		t.Errorf("expected ErrUnknownMutation, got %v", err)
	}
}

func TestMutationResult(t *testing.T) {
	result := MutationResult{
		OK:     true,
		Cursor: 42,
		Changes: []Change{{
			Cursor: 42,
			Entity: "todo",
			ID:     "1",
			Op:     OpCreate,
		}},
	}

	if !result.OK {
		t.Error("expected OK to be true")
	}
	if result.Cursor != 42 {
		t.Errorf("expected cursor 42, got %d", result.Cursor)
	}
	if len(result.Changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(result.Changes))
	}
}

func TestChangeOp(t *testing.T) {
	if OpCreate != "create" {
		t.Errorf("expected OpCreate to be 'create', got %s", OpCreate)
	}
	if OpUpdate != "update" {
		t.Errorf("expected OpUpdate to be 'update', got %s", OpUpdate)
	}
	if OpDelete != "delete" {
		t.Errorf("expected OpDelete to be 'delete', got %s", OpDelete)
	}
}
