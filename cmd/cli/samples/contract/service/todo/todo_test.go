package todo

import (
	"context"
	"testing"
)

func TestService_Create(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	t.Run("creates todo with title", func(t *testing.T) {
		todo, err := svc.Create(ctx, &CreateIn{Title: "Test todo"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if todo.Title != "Test todo" {
			t.Errorf("got title %q, want %q", todo.Title, "Test todo")
		}
		if todo.ID == "" {
			t.Error("expected non-empty ID")
		}
		if todo.Completed {
			t.Error("new todo should not be completed")
		}
		if todo.Priority != PriorityMedium {
			t.Errorf("got priority %q, want default %q", todo.Priority, PriorityMedium)
		}
	})

	t.Run("rejects empty title", func(t *testing.T) {
		_, err := svc.Create(ctx, &CreateIn{Title: ""})
		if err != ErrTitleEmpty {
			t.Errorf("got error %v, want ErrTitleEmpty", err)
		}
	})

	t.Run("creates todo with priority", func(t *testing.T) {
		todo, err := svc.Create(ctx, &CreateIn{
			Title:    "Priority todo",
			Priority: PriorityHigh,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if todo.Priority != PriorityHigh {
			t.Errorf("got priority %q, want %q", todo.Priority, PriorityHigh)
		}
	})
}

func TestService_Get(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	// Create a todo first
	created, _ := svc.Create(ctx, &CreateIn{Title: "Get test"})

	t.Run("retrieves existing todo", func(t *testing.T) {
		todo, err := svc.Get(ctx, &GetIn{ID: created.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if todo.ID != created.ID {
			t.Errorf("got ID %q, want %q", todo.ID, created.ID)
		}
	})

	t.Run("returns error for non-existent todo", func(t *testing.T) {
		_, err := svc.Get(ctx, &GetIn{ID: "nonexistent"})
		if err != ErrNotFound {
			t.Errorf("got error %v, want ErrNotFound", err)
		}
	})
}

func TestService_List(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	t.Run("returns empty list initially", func(t *testing.T) {
		list, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list.Items) != 0 {
			t.Errorf("got %d items, want 0", len(list.Items))
		}
		if list.Total != 0 {
			t.Errorf("got total %d, want 0", list.Total)
		}
	})

	t.Run("returns all todos", func(t *testing.T) {
		svc.Create(ctx, &CreateIn{Title: "Todo 1"})
		svc.Create(ctx, &CreateIn{Title: "Todo 2"})

		list, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list.Items) != 2 {
			t.Errorf("got %d items, want 2", len(list.Items))
		}
		if list.Total != 2 {
			t.Errorf("got total %d, want 2", list.Total)
		}
	})
}

func TestService_Update(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	created, _ := svc.Create(ctx, &CreateIn{Title: "Update test"})

	t.Run("updates title", func(t *testing.T) {
		updated, err := svc.Update(ctx, &UpdateIn{
			ID:    created.ID,
			Title: "Updated title",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Title != "Updated title" {
			t.Errorf("got title %q, want %q", updated.Title, "Updated title")
		}
	})

	t.Run("updates completed status", func(t *testing.T) {
		updated, err := svc.Update(ctx, &UpdateIn{
			ID:        created.ID,
			Completed: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updated.Completed {
			t.Error("expected completed to be true")
		}
	})

	t.Run("updates priority", func(t *testing.T) {
		updated, err := svc.Update(ctx, &UpdateIn{
			ID:       created.ID,
			Priority: PriorityLow,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Priority != PriorityLow {
			t.Errorf("got priority %q, want %q", updated.Priority, PriorityLow)
		}
	})

	t.Run("returns error for non-existent todo", func(t *testing.T) {
		_, err := svc.Update(ctx, &UpdateIn{ID: "nonexistent"})
		if err != ErrNotFound {
			t.Errorf("got error %v, want ErrNotFound", err)
		}
	})
}

func TestService_Delete(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	created, _ := svc.Create(ctx, &CreateIn{Title: "Delete test"})

	t.Run("deletes existing todo", func(t *testing.T) {
		err := svc.Delete(ctx, &DeleteIn{ID: created.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify deletion
		_, err = svc.Get(ctx, &GetIn{ID: created.ID})
		if err != ErrNotFound {
			t.Error("expected todo to be deleted")
		}
	})

	t.Run("returns error for non-existent todo", func(t *testing.T) {
		err := svc.Delete(ctx, &DeleteIn{ID: "nonexistent"})
		if err != ErrNotFound {
			t.Errorf("got error %v, want ErrNotFound", err)
		}
	})
}

func TestService_Health(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	err := svc.Health(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
