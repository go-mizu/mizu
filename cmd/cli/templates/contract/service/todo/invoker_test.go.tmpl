package todo

import (
	"context"
	"fmt"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"gopkg.in/yaml.v3"
)

func TestInvoker_Call(t *testing.T) {
	svc := &Service{}

	// Minimal contract for testing
	contractYAML := `
name: Todo
resources:
  - name: todos
    methods:
      - name: list
      - name: create
      - name: get
      - name: update
      - name: delete
  - name: health
    methods:
      - name: check
`
	var c contract.Service
	if err := yaml.Unmarshal([]byte(contractYAML), &c); err != nil {
		t.Fatalf("failed to parse contract: %v", err)
	}

	invoker := NewInvoker(svc, &c)
	ctx := context.Background()

	t.Run("todos.list returns TodoList", func(t *testing.T) {
		result, err := invoker.Call(ctx, "todos", "list", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		list, ok := result.(*TodoList)
		if !ok {
			t.Fatalf("expected *TodoList, got %T", result)
		}
		if list.Items == nil {
			t.Error("expected non-nil Items slice")
		}
	})

	t.Run("todos.create creates todo", func(t *testing.T) {
		result, err := invoker.Call(ctx, "todos", "create", map[string]any{
			"title": "Test from invoker",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		todo, ok := result.(*Todo)
		if !ok {
			t.Fatalf("expected *Todo, got %T", result)
		}
		if todo.Title != "Test from invoker" {
			t.Errorf("got title %q, want %q", todo.Title, "Test from invoker")
		}
		if todo.ID == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("todos.get retrieves todo", func(t *testing.T) {
		// First create a todo
		created, _ := invoker.Call(ctx, "todos", "create", map[string]any{
			"title": "Get test",
		})
		todo := created.(*Todo)

		result, err := invoker.Call(ctx, "todos", "get", map[string]any{
			"id": todo.ID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := result.(*Todo)
		if !ok {
			t.Fatalf("expected *Todo, got %T", result)
		}
		if got.ID != todo.ID {
			t.Errorf("got ID %q, want %q", got.ID, todo.ID)
		}
	})

	t.Run("todos.update modifies todo", func(t *testing.T) {
		created, _ := invoker.Call(ctx, "todos", "create", map[string]any{
			"title": "Update test",
		})
		todo := created.(*Todo)

		result, err := invoker.Call(ctx, "todos", "update", map[string]any{
			"id":        todo.ID,
			"completed": true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		updated, ok := result.(*Todo)
		if !ok {
			t.Fatalf("expected *Todo, got %T", result)
		}
		if !updated.Completed {
			t.Error("expected completed to be true")
		}
	})

	t.Run("todos.delete removes todo", func(t *testing.T) {
		created, _ := invoker.Call(ctx, "todos", "create", map[string]any{
			"title": "Delete test",
		})
		todo := created.(*Todo)

		_, err := invoker.Call(ctx, "todos", "delete", map[string]any{
			"id": todo.ID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify deletion
		_, err = invoker.Call(ctx, "todos", "get", map[string]any{
			"id": todo.ID,
		})
		if err != ErrNotFound {
			t.Error("expected todo to be deleted")
		}
	})

	t.Run("health.check returns ok", func(t *testing.T) {
		result, err := invoker.Call(ctx, "health", "check", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		status, ok := result.(*HealthStatus)
		if !ok {
			t.Fatalf("expected *HealthStatus, got %T", result)
		}
		if status.Status != "ok" {
			t.Errorf("got status %q, want %q", status.Status, "ok")
		}
	})

	t.Run("unknown method returns error", func(t *testing.T) {
		_, err := invoker.Call(ctx, "unknown", "method", nil)
		if err == nil {
			t.Error("expected error for unknown method")
		}
	})
}

func TestInvoker_NewInput(t *testing.T) {
	svc := &Service{}
	var c contract.Service
	invoker := NewInvoker(svc, &c)

	tests := []struct {
		resource string
		method   string
		wantType any
	}{
		{"todos", "create", &CreateIn{}},
		{"todos", "get", &GetIn{}},
		{"todos", "update", &UpdateIn{}},
		{"todos", "delete", &DeleteIn{}},
		{"todos", "list", nil},
		{"health", "check", nil},
	}

	for _, tt := range tests {
		t.Run(tt.resource+"."+tt.method, func(t *testing.T) {
			got, err := invoker.NewInput(tt.resource, tt.method)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantType == nil {
				if got != nil {
					t.Errorf("expected nil, got %T", got)
				}
				return
			}
			// Check type matches
			wantTypeName := getTypeName(tt.wantType)
			gotTypeName := getTypeName(got)
			if gotTypeName != wantTypeName {
				t.Errorf("got type %s, want %s", gotTypeName, wantTypeName)
			}
		})
	}
}

func getTypeName(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", v)
}

func TestInvoker_Stream(t *testing.T) {
	svc := &Service{}
	var c contract.Service
	invoker := NewInvoker(svc, &c)

	_, err := invoker.Stream(context.Background(), "todos", "list", nil)
	if err != contract.ErrUnsupported {
		t.Errorf("expected ErrUnsupported, got %v", err)
	}
}

func TestInvoker_Descriptor(t *testing.T) {
	svc := &Service{}
	c := &contract.Service{Name: "TestService"}
	invoker := NewInvoker(svc, c)

	desc := invoker.Descriptor()
	if desc != c {
		t.Error("Descriptor should return the contract")
	}
	if desc.Name != "TestService" {
		t.Errorf("got name %q, want %q", desc.Name, "TestService")
	}
}
