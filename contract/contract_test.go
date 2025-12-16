package contract

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
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
		return nil, ErrNotFound("todo not found")
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

func TestInvoker(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	ctx := context.Background()

	// Test Create
	create := svc.Method("Create")
	out, err := create.Invoker.Call(ctx, &CreateTodoInput{Title: "Test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	todo := out.(*Todo)
	if todo.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", todo.Title)
	}

	// Test Delete (no output)
	del := svc.Method("Delete")
	_, err = del.Invoker.Call(ctx, &DeleteTodoInput{ID: "1"})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestTestClient(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	client := NewTestClient(svc)

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

	// Test MustCall
	out = client.MustCall(ctx, "List", &ListTodosInput{})
	list := out.(*ListTodosOutput)
	if len(list.Todos) != 2 {
		t.Errorf("expected 2 todos, got %d", len(list.Todos))
	}
}

func TestError(t *testing.T) {
	// Test error creation
	err := NewError(ErrCodeNotFound, "resource not found")
	if err.Code != ErrCodeNotFound {
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
	err = err.WithDetail("id", "123")
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
	if asErr.Code != ErrCodeInternal {
		t.Errorf("expected INTERNAL, got %v", asErr.Code)
	}
}

func TestSchemaGeneration(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})

	schemas := svc.Types.Schemas()
	if len(schemas) == 0 {
		t.Fatal("expected schemas")
	}

	// Find CreateTodoInput schema
	var createSchema Schema
	for _, s := range schemas {
		if strings.Contains(s.ID, "CreateTodoInput") {
			createSchema = s
			break
		}
	}

	if createSchema.ID == "" {
		t.Fatal("CreateTodoInput schema not found")
	}

	// Check schema structure
	props, ok := createSchema.JSON["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in schema")
	}

	if _, ok := props["title"]; !ok {
		t.Error("expected 'title' property")
	}
}

func TestRESTTransport(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	mux := http.NewServeMux()
	MountREST(mux, svc)

	// Test POST /todos (Create)
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"Test"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var todo Todo
	if err := json.NewDecoder(rec.Body).Decode(&todo); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if todo.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", todo.Title)
	}
}

func TestJSONRPCTransport(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	mux := http.NewServeMux()
	MountJSONRPC(mux, "/rpc", svc)

	// Test JSON-RPC call
	body := `{"jsonrpc":"2.0","id":1,"method":"Create","params":{"title":"RPC Test"}}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  *Todo  `json:"result"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result")
	}
	if resp.Result.Title != "RPC Test" {
		t.Errorf("expected title 'RPC Test', got %q", resp.Result.Title)
	}
}

func TestIntrospection(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	resp := Introspect(svc)

	if len(resp.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(resp.Services))
	}

	sd := resp.Services[0]
	if sd.Name != "todo" {
		t.Errorf("expected name 'todo', got %q", sd.Name)
	}
	if len(sd.Methods) != 5 {
		t.Errorf("expected 5 methods, got %d", len(sd.Methods))
	}

	// Check REST hints
	var createMethod *MethodDescriptor
	for i := range sd.Methods {
		if sd.Methods[i].Name == "Create" {
			createMethod = &sd.Methods[i]
			break
		}
	}
	if createMethod == nil {
		t.Fatal("Create method not found")
	}
	if createMethod.REST.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", createMethod.REST.Method)
	}
}

func TestClientDescriptor(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})
	desc := GenerateClientDescriptor("example", svc)

	if desc.Package != "example" {
		t.Errorf("expected package 'example', got %q", desc.Package)
	}
	if len(desc.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(desc.Services))
	}
	if len(desc.Errors) == 0 {
		t.Error("expected standard errors")
	}
}

func TestMiddleware(t *testing.T) {
	svc, _ := Register("todo", &TodoService{})

	var logs []string
	logMW := func(next MethodInvoker) MethodInvoker {
		return func(ctx context.Context, method *Method, in any) (any, error) {
			logs = append(logs, "before:"+method.Name)
			out, err := next(ctx, method, in)
			logs = append(logs, "after:"+method.Name)
			return out, err
		}
	}

	wrapped := svc.WithMiddleware(logMW)
	ctx := context.Background()

	_, err := wrapped.Call(ctx, "Create", &CreateTodoInput{Title: "Test"})
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

	wrapped := svc.WithMiddleware(RecoveryMiddleware())
	ctx := context.Background()

	_, err = wrapped.Call(ctx, "Create", &CreateTodoInput{Title: "Test"})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
}

func TestMockService(t *testing.T) {
	mock := NewMockService("todo")
	mock.OnReturn("Get", &Todo{ID: "mock-1", Title: "Mocked"}, nil)

	ctx := context.Background()
	out, err := mock.Call(ctx, "Get", &GetTodoInput{ID: "1"})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	todo := out.(*Todo)
	if todo.Title != "Mocked" {
		t.Errorf("expected 'Mocked', got %q", todo.Title)
	}

	// Test error return
	mock.OnError("Delete", ErrNotFound("not found"))
	_, err = mock.Call(ctx, "Delete", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestRecordingMock(t *testing.T) {
	mock := NewRecordingMock("todo")
	mock.OnRecord("Create", func(ctx context.Context, in any) (any, error) {
		input := in.(*CreateTodoInput)
		return &Todo{ID: "recorded", Title: input.Title}, nil
	})

	ctx := context.Background()
	_, _ = mock.Call(ctx, "Create", &CreateTodoInput{Title: "Recorded"})

	calls := mock.CallsFor("Create")
	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}

	if mock.CallCount("Create") != 1 {
		t.Errorf("expected call count 1, got %d", mock.CallCount("Create"))
	}

	mock.Reset()
	if len(mock.Calls()) != 0 {
		t.Error("expected 0 calls after reset")
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
	reg := newTypeRegistry()
	_, err := reg.Add(reflect.TypeOf(TaskWithEnum{}))
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	schemas := reg.Schemas()
	if len(schemas) == 0 {
		t.Fatal("expected schema")
	}
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

func TestComplexTypeSchema(t *testing.T) {
	reg := newTypeRegistry()
	_, err := reg.Add(reflect.TypeOf(ComplexInput{}))
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	schemas := reg.Schemas()
	var found bool
	for _, s := range schemas {
		if strings.Contains(s.ID, "ComplexInput") {
			found = true
			props := s.JSON["properties"].(map[string]any)

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

func TestErrorCodeMappings(t *testing.T) {
	tests := []struct {
		code       ErrorCode
		httpStatus int
		grpcCode   int
	}{
		{ErrCodeOK, 200, 0},
		{ErrCodeNotFound, 404, 5},
		{ErrCodeInvalidArgument, 400, 3},
		{ErrCodePermissionDenied, 403, 7},
		{ErrCodeUnauthenticated, 401, 16},
		{ErrCodeInternal, 500, 13},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := ErrorCodeToHTTPStatus(tt.code); got != tt.httpStatus {
				t.Errorf("HTTP: expected %d, got %d", tt.httpStatus, got)
			}
			if got := ErrorCodeToGRPC(tt.code); got != tt.grpcCode {
				t.Errorf("gRPC: expected %d, got %d", tt.grpcCode, got)
			}
		})
	}
}

func TestHTTPStatusToErrorCode(t *testing.T) {
	tests := []struct {
		status int
		code   ErrorCode
	}{
		{200, ErrCodeOK},
		{400, ErrCodeInvalidArgument},
		{401, ErrCodeUnauthenticated},
		{403, ErrCodePermissionDenied},
		{404, ErrCodeNotFound},
		{500, ErrCodeInternal},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.status)), func(t *testing.T) {
			if got := HTTPStatusToErrorCode(tt.status); got != tt.code {
				t.Errorf("expected %s, got %s", tt.code, got)
			}
		})
	}
}
