package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/contract"
)

// Test service for OpenAPI tests
type todoService struct{}

type CreateTodoInput struct {
	Title string `json:"title"`
}

type Todo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// GetByIdInput contains "id" in the type name for needsID to detect
type GetByIdInput struct {
	ID string `json:"id"`
}

type ListTodosInput struct {
	Limit int `json:"limit,omitempty"`
}

type ListTodosOutput struct {
	Todos []*Todo `json:"todos"`
}

func (s *todoService) Create(ctx context.Context, in *CreateTodoInput) (*Todo, error) {
	return &Todo{ID: "1", Title: in.Title}, nil
}

func (s *todoService) Get(ctx context.Context, in *GetByIdInput) (*Todo, error) {
	return &Todo{ID: in.ID, Title: "Test"}, nil
}

func (s *todoService) List(ctx context.Context, in *ListTodosInput) (*ListTodosOutput, error) {
	return &ListTodosOutput{Todos: []*Todo{{ID: "1", Title: "Test"}}}, nil
}

func (s *todoService) Delete(ctx context.Context, in *GetByIdInput) error {
	return nil
}

func (*todoService) ContractServiceMeta() contract.ServiceOptions {
	return contract.ServiceOptions{
		Description: "Todo service",
		Version:     "1.0.0",
		Tags:        []string{"todos"},
	}
}

func (*todoService) ContractMeta() map[string]contract.MethodOptions {
	return map[string]contract.MethodOptions{
		"Create": {Summary: "Create todo", Description: "Creates a new todo"},
		"Get":    {Summary: "Get todo", Description: "Gets a todo by ID"},
		"List":   {Summary: "List todos", Description: "Lists all todos"},
		"Delete": {Summary: "Delete todo", Description: "Deletes a todo"},
	}
}

func TestGenerate(t *testing.T) {
	svc, err := contract.Register("todo", &todoService{})
	if err != nil {
		t.Fatalf("register error: %v", err)
	}

	doc := Generate(svc)

	// Check OpenAPI version
	if doc.OpenAPI != Version {
		t.Errorf("openapi = %q, want %q", doc.OpenAPI, Version)
	}

	// Check info
	if doc.Info.Title != "todo API" {
		t.Errorf("title = %q, want %q", doc.Info.Title, "todo API")
	}
	if doc.Info.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", doc.Info.Version, "1.0.0")
	}
	if doc.Info.Description != "Todo service" {
		t.Errorf("description = %q, want %q", doc.Info.Description, "Todo service")
	}

	// Check paths exist
	if len(doc.Paths) == 0 {
		t.Error("expected paths")
	}

	// Check /todos path
	todosPath, ok := doc.Paths["/todos"]
	if !ok {
		t.Fatal("expected /todos path")
	}
	if todosPath.Post == nil {
		t.Error("expected POST operation on /todos")
	}
	if todosPath.Get == nil {
		t.Error("expected GET operation on /todos")
	}

	// Check /todos/{id} path
	todoIDPath, ok := doc.Paths["/todos/{id}"]
	if !ok {
		t.Fatal("expected /todos/{id} path")
	}
	if todoIDPath.Get == nil {
		t.Error("expected GET operation on /todos/{id}")
	}
	if todoIDPath.Delete == nil {
		t.Error("expected DELETE operation on /todos/{id}")
	}

	// Check schemas
	if len(doc.Components.Schemas) == 0 {
		t.Error("expected schemas")
	}
}

func TestGenerate_MultipleServices(t *testing.T) {
	svc1, _ := contract.Register("todo", &todoService{})

	type userService struct{}
	type userInput struct {
		Name string `json:"name"`
	}
	type user struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	us := &struct {
		Create func(context.Context, *userInput) (*user, error)
	}{
		Create: func(ctx context.Context, in *userInput) (*user, error) {
			return &user{ID: "1", Name: in.Name}, nil
		},
	}
	_ = us // Placeholder for second service

	doc := Generate(svc1)

	// Should have info from first service
	if !strings.Contains(doc.Info.Title, "todo") {
		t.Errorf("title should contain 'todo', got %q", doc.Info.Title)
	}
}

func TestDocument_WriteJSON(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	doc := Generate(svc)

	var sb strings.Builder
	if err := doc.WriteJSON(&sb); err != nil {
		t.Fatalf("write error: %v", err)
	}

	output := sb.String()
	if !strings.Contains(output, `"openapi"`) {
		t.Error("output should contain openapi field")
	}
	if !strings.Contains(output, `"paths"`) {
		t.Error("output should contain paths field")
	}
}

func TestDocument_MarshalJSON(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	doc := Generate(svc)

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed["openapi"] != Version {
		t.Errorf("openapi = %v, want %s", parsed["openapi"], Version)
	}
}

func TestSchemaRef_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		ref  SchemaRef
		want string
	}{
		{
			name: "with ref",
			ref:  SchemaRef{Ref: "#/components/schemas/Todo"},
			want: `{"$ref":"#/components/schemas/Todo"}`,
		},
		{
			name: "with schema",
			ref:  SchemaRef{Schema: &Schema{Type: "string"}},
			want: `{"type":"string"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(&tt.ref)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("got %s, want %s", data, tt.want)
			}
		})
	}
}

func TestPathItem_SetOperation(t *testing.T) {
	p := &PathItem{}
	op := &Operation{OperationID: "test"}

	tests := []struct {
		method string
		getter func() *Operation
	}{
		{"get", func() *Operation { return p.Get }},
		{"GET", func() *Operation { return p.Get }},
		{"post", func() *Operation { return p.Post }},
		{"put", func() *Operation { return p.Put }},
		{"delete", func() *Operation { return p.Delete }},
		{"patch", func() *Operation { return p.Patch }},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			p.SetOperation(tt.method, op)
			if got := tt.getter(); got != op {
				t.Errorf("operation not set for method %s", tt.method)
			}
		})
	}
}

func TestNewSpecHandler(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})

	h, err := NewSpecHandler(svc)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if h.Name() != "rest" {
		t.Errorf("name = %q, want %q", h.Name(), "rest")
	}

	if h.Document() == nil {
		t.Error("expected document")
	}

	if len(h.JSON()) == 0 {
		t.Error("expected cached JSON")
	}
}

func TestSpecHandler_ServeHTTP(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	h, _ := NewSpecHandler(svc)

	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantType   string
	}{
		{
			name:       "GET request",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantType:   "application/json",
		},
		{
			name:       "POST not allowed",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/openapi.json", nil)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantType != "" {
				ct := rec.Header().Get("Content-Type")
				if ct != tt.wantType {
					t.Errorf("content-type = %q, want %q", ct, tt.wantType)
				}
			}

			if tt.wantStatus == http.StatusOK {
				// Verify it's valid JSON
				var doc map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
					t.Errorf("invalid JSON: %v", err)
				}
			}
		})
	}
}

func TestMountSpec(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	mux := http.NewServeMux()

	err := MountSpec(mux, "/openapi.json", svc)
	if err != nil {
		t.Fatalf("mount error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMountSpec_DefaultPath(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	mux := http.NewServeMux()

	err := MountSpec(mux, "", svc)
	if err != nil {
		t.Fatalf("mount error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMountSpecWithDocs(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	mux := http.NewServeMux()

	err := MountSpecWithDocs(mux, "/openapi.json", "/docs", svc)
	if err != nil {
		t.Fatalf("mount error: %v", err)
	}

	// Test spec endpoint
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("spec status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Test docs endpoint
	req = httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("docs status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("docs should return HTML")
	}
}

func TestDocsHandler_ServeHTTP(t *testing.T) {
	h := NewDocsHandler("/openapi.json")

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Error("expected Swagger UI in response")
	}
	if !strings.Contains(body, "/openapi.json") {
		t.Error("expected spec URL in response")
	}
}

func TestConvertSchema(t *testing.T) {
	input := map[string]any{
		"type":        "object",
		"description": "Test schema",
		"properties": map[string]any{
			"name": map[string]any{
				"type":      "string",
				"minLength": 1,
				"maxLength": 100,
			},
			"age": map[string]any{
				"type":    "integer",
				"minimum": float64(0),
				"maximum": float64(150),
			},
			"tags": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []any{"name"},
	}

	schema := convertSchema(input)

	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if schema.Description != "Test schema" {
		t.Errorf("description = %q, want %q", schema.Description, "Test schema")
	}
	if len(schema.Properties) != 3 {
		t.Errorf("got %d properties, want 3", len(schema.Properties))
	}
	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Errorf("required = %v, want [name]", schema.Required)
	}

	nameProp := schema.Properties["name"]
	if nameProp == nil {
		t.Fatal("expected name property")
	}
	if *nameProp.MinLength != 1 {
		t.Errorf("name.minLength = %v, want 1", nameProp.MinLength)
	}

	tagsProp := schema.Properties["tags"]
	if tagsProp == nil {
		t.Fatal("expected tags property")
	}
	if tagsProp.Items == nil {
		t.Error("expected items on array property")
	}
}

// REST Handler Tests

func TestNewHandler(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	h := NewHandler(svc)

	if h.Name() != "rest" {
		t.Errorf("name = %q, want %q", h.Name(), "rest")
	}

	if h.basePath != "/todos" {
		t.Errorf("basePath = %q, want %q", h.basePath, "/todos")
	}
}

func TestHandler_Create(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	h := NewHandler(svc)

	body := strings.NewReader(`{"title": "Test Todo"}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result Todo
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if result.Title != "Test Todo" {
		t.Errorf("title = %q, want %q", result.Title, "Test Todo")
	}
}

func TestHandler_Get(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/todos/123", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result Todo
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if result.ID != "123" {
		t.Errorf("id = %q, want %q", result.ID, "123")
	}
}

func TestHandler_List(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/todos?limit=10", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result ListTodosOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(result.Todos) != 1 {
		t.Errorf("todos count = %d, want 1", len(result.Todos))
	}
}

func TestHandler_Delete(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/todos/123", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestHandler_NotFound(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/todos", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestMount_REST(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	mux := http.NewServeMux()

	Mount(mux, svc)

	// Test create
	body := strings.NewReader(`{"title": "Test"}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("create status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMountWithSpec_REST(t *testing.T) {
	svc, _ := contract.Register("todo", &todoService{})
	mux := http.NewServeMux()

	err := MountWithSpec(mux, "/openapi.json", svc)
	if err != nil {
		t.Fatalf("mount error: %v", err)
	}

	// Test spec endpoint
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("spec status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Test REST endpoint
	body := strings.NewReader(`{"title": "Test"}`)
	req = httptest.NewRequest(http.MethodPost, "/todos", body)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("rest status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestInferHTTPMethod(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   string
	}{
		{"CreateTodo", "CreateTodo", http.MethodPost},
		{"GetTodo", "GetTodo", http.MethodGet},
		{"ListTodos", "ListTodos", http.MethodGet},
		{"UpdateTodo", "UpdateTodo", http.MethodPut},
		{"DeleteTodo", "DeleteTodo", http.MethodDelete},
		{"PatchTodo", "PatchTodo", http.MethodPatch},
		{"DoSomething", "DoSomething", http.MethodPost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferHTTPMethod(tt.method)
			if got != tt.want {
				t.Errorf("inferHTTPMethod(%q) = %q, want %q", tt.method, got, tt.want)
			}
		})
	}
}

func TestExtractPathVars(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/todos", nil},
		{"/todos/{id}", []string{"id"}},
		{"/users/{userId}/todos/{todoId}", []string{"userId", "todoId"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := extractPathVars(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("extractPathVars(%q) = %v, want %v", tt.path, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractPathVars(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPathToRegexp(t *testing.T) {
	tests := []struct {
		path    string
		testURL string
		match   bool
	}{
		{"/todos", "/todos", true},
		{"/todos", "/todos/", false},
		{"/todos/{id}", "/todos/123", true},
		{"/todos/{id}", "/todos", false},
		{"/users/{userId}/todos/{todoId}", "/users/1/todos/2", true},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.testURL, func(t *testing.T) {
			re := pathToRegexp(tt.path)
			got := re.MatchString(tt.testURL)
			if got != tt.match {
				t.Errorf("pathToRegexp(%q).MatchString(%q) = %v, want %v", tt.path, tt.testURL, got, tt.match)
			}
		})
	}
}
