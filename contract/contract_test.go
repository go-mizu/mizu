package contract

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---- Test Service Definitions ----

type TodoService struct{}

type CreateTodoInput struct {
	Title string `json:"title" contract:"required,minLength=1"`
}

type Todo struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"createdAt"`
	DoneAt    *time.Time `json:"doneAt,omitempty"`
}

type GetTodoInput struct {
	ID string `json:"id"`
}

type ListTodosInput struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type ListTodosOutput struct {
	Todos      []*Todo `json:"todos"`
	NextCursor string  `json:"nextCursor,omitempty"`
}

type UpdateTodoInput struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
	Done  *bool  `json:"done,omitempty"`
}

type DeleteTodoInput struct {
	ID string `json:"id"`
}

func (s *TodoService) Create(ctx context.Context, in *CreateTodoInput) (*Todo, error) {
	return &Todo{
		ID:        "todo-1",
		Title:     in.Title,
		CreatedAt: time.Now(),
	}, nil
}

func (s *TodoService) Get(ctx context.Context, in *GetTodoInput) (*Todo, error) {
	if in.ID == "" {
		return nil, NewError(NotFound, "todo not found")
	}
	return &Todo{
		ID:        in.ID,
		Title:     "Test Todo",
		CreatedAt: time.Now(),
	}, nil
}

func (s *TodoService) List(ctx context.Context, in *ListTodosInput) (*ListTodosOutput, error) {
	return &ListTodosOutput{
		Todos: []*Todo{
			{ID: "1", Title: "Todo 1"},
			{ID: "2", Title: "Todo 2"},
		},
	}, nil
}

func (s *TodoService) Update(ctx context.Context, in *UpdateTodoInput) (*Todo, error) {
	return &Todo{
		ID:    in.ID,
		Title: in.Title,
	}, nil
}

func (s *TodoService) Delete(ctx context.Context, in *DeleteTodoInput) error {
	return nil
}

// ServiceMeta implementation
func (*TodoService) ContractServiceMeta() ServiceOptions {
	return ServiceOptions{
		Description: "Manages todo items",
		Version:     "1.0.0",
		Tags:        []string{"todos"},
	}
}

// MethodMeta implementation
func (*TodoService) ContractMeta() map[string]MethodOptions {
	return map[string]MethodOptions{
		"Create": {Description: "Creates a new todo", Summary: "Create todo"},
		"Get":    {Description: "Gets a todo by ID", Summary: "Get todo"},
		"List":   {Description: "Lists all todos", Summary: "List todos"},
		"Update": {Description: "Updates a todo", Summary: "Update todo"},
		"Delete": {Description: "Deletes a todo", Summary: "Delete todo"},
	}
}

// ---- Tests ----

func TestRegister(t *testing.T) {
	svc, err := Register("todo", &TodoService{})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if svc.Name != "todo" {
		t.Errorf("expected name 'todo', got %q", svc.Name)
	}

	if len(svc.Methods) != 5 {
		t.Errorf("expected 5 methods, got %d", len(svc.Methods))
	}

	// Check metadata
	if svc.Description != "Manages todo items" {
		t.Errorf("expected description, got %q", svc.Description)
	}
	if svc.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", svc.Version)
	}

	// Check method metadata
	create := svc.Method("Create")
	if create == nil {
		t.Fatal("Create method not found")
	}
	if create.Description != "Creates a new todo" {
		t.Errorf("expected description, got %q", create.Description)
	}
}

func TestRegisterErrors(t *testing.T) {
	tests := []struct {
		name    string
		svcName string
		svc     any
		wantErr string
	}{
		{"empty name", "", &TodoService{}, "empty name"},
		{"nil service", "test", nil, "nil service"},
		{"not struct", "test", "string", "expected struct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Register(tt.svcName, tt.svc)
			if err == nil {
				t.Error("expected error")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestMethodCall(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	ctx := context.Background()

	// Test Create
	create := svc.Method("Create")
	out, err := create.Call(ctx, &CreateTodoInput{Title: "Test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	todo := out.(*Todo)
	if todo.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", todo.Title)
	}

	// Test Delete (no output)
	del := svc.Method("Delete")
	_, err = del.Call(ctx, &DeleteTodoInput{ID: "1"})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestClient(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	client := NewClient(svc)

	ctx := context.Background()

	// Test Call
	out, err := client.Call(ctx, "Create", &CreateTodoInput{Title: "Test"})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	todo := out.(*Todo)
	if todo.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", todo.Title)
	}

	// Test CallJSON
	jsonOut, err := client.CallJSON(ctx, "Create", []byte(`{"title":"JSON Test"}`))
	if err != nil {
		t.Fatalf("CallJSON failed: %v", err)
	}
	var jsonTodo Todo
	if err := json.Unmarshal(jsonOut, &jsonTodo); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if jsonTodo.Title != "JSON Test" {
		t.Errorf("expected title 'JSON Test', got %q", jsonTodo.Title)
	}

	// Test method not found
	_, err = client.Call(ctx, "Unknown", nil)
	if err == nil {
		t.Error("expected error for unknown method")
	}
}

func TestError(t *testing.T) {
	// Test error creation
	err := NewError(NotFound, "resource not found")
	if err.Code != NotFound {
		t.Errorf("expected NOT_FOUND, got %v", err.Code)
	}
	if err.Error() != "resource not found" {
		t.Errorf("expected 'resource not found', got %q", err.Error())
	}

	// Test HTTP status mapping
	if err.HTTPStatus() != http.StatusNotFound {
		t.Errorf("expected 404, got %d", err.HTTPStatus())
	}

	// Test with details
	err = err.WithDetails(map[string]any{"id": "123"})
	if err.Details["id"] != "123" {
		t.Error("expected detail 'id' = '123'")
	}

	// Test error wrapping
	cause := errors.New("underlying error")
	err = err.WithCause(cause)
	if !errors.Is(err, cause) {
		t.Error("expected errors.Is to return true")
	}

	// Test AsError
	asErr := AsError(errors.New("plain error"))
	if asErr.Code != Internal {
		t.Errorf("expected INTERNAL, got %v", asErr.Code)
	}
}

func TestSchemaGeneration(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})

	types := svc.Types.All()
	if len(types) == 0 {
		t.Fatal("expected types")
	}

	// Find CreateTodoInput schema
	var foundID string
	for _, typ := range types {
		if strings.Contains(typ.ID, "CreateTodoInput") {
			foundID = typ.ID
			break
		}
	}

	if foundID == "" {
		t.Fatal("CreateTodoInput schema not found")
	}

	schema := svc.Types.Schema(foundID)
	if schema == nil {
		t.Fatal("schema not found for ID")
	}

	// Check schema structure
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in schema")
	}

	if _, ok := props["title"]; !ok {
		t.Error("expected 'title' property")
	}
}

func TestDescribe(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	desc := Describe(svc)

	if len(desc.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(desc.Services))
	}

	sd := desc.Services[0]
	if sd.Name != "todo" {
		t.Errorf("expected name 'todo', got %q", sd.Name)
	}
	if len(sd.Methods) != 5 {
		t.Errorf("expected 5 methods, got %d", len(sd.Methods))
	}

	// Check HTTP hints
	var createMethod *MethodDesc
	for i := range sd.Methods {
		if sd.Methods[i].Name == "Create" {
			createMethod = &sd.Methods[i]
			break
		}
	}
	if createMethod == nil {
		t.Fatal("Create method not found")
	}
	if createMethod.HTTP.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", createMethod.HTTP.Method)
	}
}

func TestMiddleware(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})

	var logs []string
	logMW := func(next Invoker) Invoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			logs = append(logs, "before:"+method.Name)
			out, err := next(ctx, method, in)
			logs = append(logs, "after:"+method.Name)
			return out, err
		}
	}

	// Apply middleware
	invoker := logMW(func(ctx context.Context, method *Method, in any) (any, error) {
		return method.Call(ctx, in)
	})

	ctx := context.Background()
	create := svc.Method("Create")
	_, err := invoker(ctx, create, &CreateTodoInput{Title: "Test"})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("expected 2 log entries, got %d", len(logs))
	}
	if logs[0] != "before:Create" {
		t.Errorf("expected 'before:Create', got %q", logs[0])
	}
	if logs[1] != "after:Create" {
		t.Errorf("expected 'after:Create', got %q", logs[1])
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	svc, err := Register("todo", &TodoService{})
	if err != nil {
		t.Fatal(err)
	}

	recovery := Recovery()
	invoker := recovery(func(ctx context.Context, method *Method, in any) (any, error) {
		return method.Call(ctx, in)
	})

	ctx := context.Background()
	create := svc.Method("Create")
	_, err = invoker(ctx, create, &CreateTodoInput{Title: "Test"})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
}

func TestChainMiddleware(t *testing.T) {
	var calls []string

	mw1 := func(next Invoker) Invoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			calls = append(calls, "mw1-before")
			out, err := next(ctx, method, in)
			calls = append(calls, "mw1-after")
			return out, err
		}
	}

	mw2 := func(next Invoker) Invoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			calls = append(calls, "mw2-before")
			out, err := next(ctx, method, in)
			calls = append(calls, "mw2-after")
			return out, err
		}
	}

	chained := Chain(mw1, mw2)
	invoker := chained(func(ctx context.Context, method *Method, in any) (any, error) {
		calls = append(calls, "handler")
		return nil, nil
	})

	svc, _ := Register("todo", &TodoService{})
	create := svc.Method("Create")
	_, _ = invoker(context.Background(), create, &CreateTodoInput{Title: "Test"})

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}
	for i, e := range expected {
		if calls[i] != e {
			t.Errorf("call %d: expected %q, got %q", i, e, calls[i])
		}
	}
}

// ---- Enum Type Test ----

type Status string

const (
	StatusPending Status = "pending"
	StatusActive  Status = "active"
	StatusDone    Status = "done"
)

func (Status) ContractEnum() []any {
	return []any{"pending", "active", "done"}
}

type TaskWithEnum struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
}

func TestEnumSchema(t *testing.T) {
	svc, err := Register("task", &taskService{})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	types := svc.Types.All()
	if len(types) == 0 {
		t.Fatal("expected types")
	}
}

type taskService struct{}

func (s *taskService) Get(ctx context.Context, in *GetTodoInput) (*TaskWithEnum, error) {
	return &TaskWithEnum{ID: in.ID, Status: StatusPending}, nil
}

// ---- Array and Map Types Test ----

type ComplexInput struct {
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
	Nested   *NestedType       `json:"nested,omitempty"`
}

type NestedType struct {
	Value int `json:"value"`
}

type complexService struct{}

func (s *complexService) Process(ctx context.Context, in *ComplexInput) error {
	return nil
}

func TestComplexTypeSchema(t *testing.T) {
	svc, err := Register("complex", &complexService{})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	types := svc.Types.All()
	var found bool
	for _, typ := range types {
		if strings.Contains(typ.ID, "ComplexInput") {
			found = true
			schema := svc.Types.Schema(typ.ID)
			if schema == nil {
				t.Fatal("schema not found")
			}
			props := schema["properties"].(map[string]any)

			// Check tags is array
			tagsSchema := props["tags"].(map[string]any)
			if tagsSchema["type"] != "array" {
				t.Error("expected tags to be array")
			}

			// Check metadata is object with additionalProperties
			metaSchema := props["metadata"].(map[string]any)
			if metaSchema["type"] != "object" {
				t.Error("expected metadata to be object")
			}
		}
	}
	if !found {
		t.Error("ComplexInput schema not found")
	}
}

func TestCodeMappings(t *testing.T) {
	tests := []struct {
		code       Code
		httpStatus int
	}{
		{OK, 200},
		{NotFound, 404},
		{InvalidArgument, 400},
		{PermissionDenied, 403},
		{Unauthenticated, 401},
		{Internal, 500},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := CodeToHTTP(tt.code); got != tt.httpStatus {
				t.Errorf("HTTP: expected %d, got %d", tt.httpStatus, got)
			}
		})
	}
}

func TestHTTPToCode(t *testing.T) {
	tests := []struct {
		status int
		code   Code
	}{
		{200, OK},
		{400, InvalidArgument},
		{401, Unauthenticated},
		{403, PermissionDenied},
		{404, NotFound},
		{500, Internal},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.status)), func(t *testing.T) {
			if got := HTTPToCode(tt.status); got != tt.code {
				t.Errorf("expected %s, got %s", tt.code, got)
			}
		})
	}
}
