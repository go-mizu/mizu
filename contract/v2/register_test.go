package contract

import (
	"context"
	"testing"
)

// Test types
type CreateInput struct {
	Title string `json:"title" required:"true"`
	Body  string `json:"body,omitempty"`
}

type GetInput struct {
	ID string `json:"id"`
}

type UpdateInput struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	Completed bool   `json:"completed,omitempty"`
}

type DeleteInput struct {
	ID string `json:"id"`
}

type Todo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Completed bool   `json:"completed"`
}

type TodoList struct {
	Items []*Todo `json:"items"`
	Total int     `json:"total"`
}

// TodoAPI is the interface to register
type TodoAPI interface {
	Create(ctx context.Context, in *CreateInput) (*Todo, error)
	List(ctx context.Context) (*TodoList, error)
	Get(ctx context.Context, in *GetInput) (*Todo, error)
	Update(ctx context.Context, in *UpdateInput) (*Todo, error)
	Delete(ctx context.Context, in *DeleteInput) error
}

// Mock implementation
type mockTodoService struct {
	todos map[string]*Todo
}

func newMockTodoService() *mockTodoService {
	return &mockTodoService{todos: make(map[string]*Todo)}
}

func (s *mockTodoService) Create(ctx context.Context, in *CreateInput) (*Todo, error) {
	todo := &Todo{
		ID:    "1",
		Title: in.Title,
		Body:  in.Body,
	}
	s.todos[todo.ID] = todo
	return todo, nil
}

func (s *mockTodoService) List(ctx context.Context) (*TodoList, error) {
	items := make([]*Todo, 0, len(s.todos))
	for _, t := range s.todos {
		items = append(items, t)
	}
	return &TodoList{Items: items, Total: len(items)}, nil
}

func (s *mockTodoService) Get(ctx context.Context, in *GetInput) (*Todo, error) {
	return s.todos[in.ID], nil
}

func (s *mockTodoService) Update(ctx context.Context, in *UpdateInput) (*Todo, error) {
	todo := s.todos[in.ID]
	if todo != nil {
		if in.Title != "" {
			todo.Title = in.Title
		}
		todo.Completed = in.Completed
	}
	return todo, nil
}

func (s *mockTodoService) Delete(ctx context.Context, in *DeleteInput) error {
	delete(s.todos, in.ID)
	return nil
}

func TestRegister(t *testing.T) {
	impl := newMockTodoService()
	svc := Register[TodoAPI](impl,
		WithName("Todo"),
		WithDescription("Todo management service"),
		WithDefaultResource("todos"),
	)

	if svc == nil {
		t.Fatal("Register returned nil")
	}

	desc := svc.Descriptor()
	if desc == nil {
		t.Fatal("Descriptor returned nil")
	}

	if desc.Name != "Todo" {
		t.Errorf("expected name 'Todo', got %q", desc.Name)
	}

	if desc.Description != "Todo management service" {
		t.Errorf("expected description 'Todo management service', got %q", desc.Description)
	}
}

func TestRegisterResources(t *testing.T) {
	impl := newMockTodoService()
	svc := Register[TodoAPI](impl,
		WithName("Todo"),
		WithDefaultResource("todos"),
	)

	desc := svc.Descriptor()

	if len(desc.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(desc.Resources))
	}

	res := desc.Resources[0]
	if res.Name != "todos" {
		t.Errorf("expected resource name 'todos', got %q", res.Name)
	}

	if len(res.Methods) != 5 {
		t.Errorf("expected 5 methods, got %d", len(res.Methods))
	}
}

func TestRegisterMethods(t *testing.T) {
	impl := newMockTodoService()
	svc := Register[TodoAPI](impl,
		WithName("Todo"),
		WithDefaultResource("todos"),
	)

	desc := svc.Descriptor()
	res := desc.Resources[0]

	// Find the create method
	var createMethod *Method
	for _, m := range res.Methods {
		if m.Name == "create" {
			createMethod = m
			break
		}
	}

	if createMethod == nil {
		t.Fatal("create method not found")
	}

	if createMethod.Input != "CreateInput" {
		t.Errorf("expected input 'CreateInput', got %q", createMethod.Input)
	}

	if createMethod.Output != "Todo" {
		t.Errorf("expected output 'Todo', got %q", createMethod.Output)
	}
}

func TestRegisterHTTPBinding(t *testing.T) {
	impl := newMockTodoService()
	svc := Register[TodoAPI](impl,
		WithName("Todo"),
		WithDefaultResource("todos"),
	)

	desc := svc.Descriptor()
	res := desc.Resources[0]

	tests := []struct {
		name       string
		httpMethod string
		path       string
	}{
		{"create", "POST", "/todos"},
		{"list", "GET", "/todos"},
		{"get", "GET", "/todos/{id}"},
		{"update", "PUT", "/todos/{id}"},
		{"delete", "DELETE", "/todos/{id}"},
	}

	for _, tc := range tests {
		var m *Method
		for _, method := range res.Methods {
			if method.Name == tc.name {
				m = method
				break
			}
		}

		if m == nil {
			t.Errorf("method %s not found", tc.name)
			continue
		}

		if m.HTTP == nil {
			t.Errorf("method %s has no HTTP binding", tc.name)
			continue
		}

		if m.HTTP.Method != tc.httpMethod {
			t.Errorf("method %s: expected HTTP method %s, got %s", tc.name, tc.httpMethod, m.HTTP.Method)
		}

		if m.HTTP.Path != tc.path {
			t.Errorf("method %s: expected path %s, got %s", tc.name, tc.path, m.HTTP.Path)
		}
	}
}

func TestRegisterTypes(t *testing.T) {
	impl := newMockTodoService()
	svc := Register[TodoAPI](impl,
		WithName("Todo"),
		WithDefaultResource("todos"),
	)

	desc := svc.Descriptor()

	// Should have types: CreateInput, GetInput, UpdateInput, DeleteInput, Todo, TodoList
	expectedTypes := map[string]bool{
		"CreateInput": true,
		"GetInput":    true,
		"UpdateInput": true,
		"DeleteInput": true,
		"Todo":        true,
		"TodoList":    true,
	}

	for _, typ := range desc.Types {
		if !expectedTypes[typ.Name] {
			t.Errorf("unexpected type %s", typ.Name)
		}
		delete(expectedTypes, typ.Name)
	}

	for name := range expectedTypes {
		t.Errorf("missing type %s", name)
	}
}

func TestRegisterCall(t *testing.T) {
	impl := newMockTodoService()
	svc := Register[TodoAPI](impl,
		WithName("Todo"),
		WithDefaultResource("todos"),
	)

	ctx := context.Background()

	// Test Create
	input := &CreateInput{Title: "Test Todo", Body: "Test Body"}
	result, err := svc.Call(ctx, "todos", "create", input)
	if err != nil {
		t.Fatalf("Call create failed: %v", err)
	}

	todo, ok := result.(*Todo)
	if !ok {
		t.Fatalf("expected *Todo, got %T", result)
	}

	if todo.Title != "Test Todo" {
		t.Errorf("expected title 'Test Todo', got %q", todo.Title)
	}

	// Test List
	result, err = svc.Call(ctx, "todos", "list", nil)
	if err != nil {
		t.Fatalf("Call list failed: %v", err)
	}

	list, ok := result.(*TodoList)
	if !ok {
		t.Fatalf("expected *TodoList, got %T", result)
	}

	if list.Total != 1 {
		t.Errorf("expected total 1, got %d", list.Total)
	}
}

func TestRegisterNewInput(t *testing.T) {
	impl := newMockTodoService()
	svc := Register[TodoAPI](impl,
		WithName("Todo"),
		WithDefaultResource("todos"),
	)

	// Test NewInput for create
	input, err := svc.NewInput("todos", "create")
	if err != nil {
		t.Fatalf("NewInput create failed: %v", err)
	}

	_, ok := input.(*CreateInput)
	if !ok {
		t.Fatalf("expected *CreateInput, got %T", input)
	}

	// Test NewInput for list (no input)
	input, err = svc.NewInput("todos", "list")
	if err != nil {
		t.Fatalf("NewInput list failed: %v", err)
	}

	if input != nil {
		t.Errorf("expected nil input for list, got %T", input)
	}
}

func TestRegisterWithHTTPOverride(t *testing.T) {
	impl := newMockTodoService()
	svc := Register[TodoAPI](impl,
		WithName("Todo"),
		WithDefaultResource("todos"),
		WithMethodHTTP("Create", "POST", "/v1/todos"),
	)

	desc := svc.Descriptor()
	res := desc.Resources[0]

	var createMethod *Method
	for _, m := range res.Methods {
		if m.Name == "create" {
			createMethod = m
			break
		}
	}

	if createMethod == nil {
		t.Fatal("create method not found")
	}

	if createMethod.HTTP.Path != "/v1/todos" {
		t.Errorf("expected path '/v1/todos', got %q", createMethod.HTTP.Path)
	}
}

