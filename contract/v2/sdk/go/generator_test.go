package sdkgo

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
)

func TestGenerateStruct(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "Todo",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "title", Type: "string"},
					{Name: "done", Type: "bool", Optional: true},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify struct is present
	src := string(code)
	if !strings.Contains(src, "type Todo struct {") {
		t.Error("missing Todo struct")
	}
	if !strings.Contains(src, "ID string") {
		t.Error("missing ID field")
	}
	if !strings.Contains(src, "Title string") {
		t.Error("missing Title field")
	}
	if !strings.Contains(src, "*bool") {
		t.Error("missing optional bool pointer")
	}
	if !strings.Contains(src, `json:"done,omitempty"`) {
		t.Error("missing omitempty for optional field")
	}

	// Verify it compiles
	assertCompiles(t, code)
}

func TestGenerateSlice(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "TodoList",
				Kind: contract.KindSlice,
				Elem: "Todo",
			},
			{
				Name: "Todo",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, "type TodoList []Todo") {
		t.Error("missing TodoList slice type")
	}

	assertCompiles(t, code)
}

func TestGenerateMap(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "Metadata",
				Kind: contract.KindMap,
				Elem: "string",
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, "type Metadata map[string]string") {
		t.Error("missing Metadata map type")
	}

	assertCompiles(t, code)
}

func TestGenerateUnion(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "ContentPart",
				Kind: contract.KindUnion,
				Tag:  "type",
				Variants: []contract.Variant{
					{Value: "text", Type: "TextPart"},
					{Value: "image", Type: "ImagePart"},
				},
			},
			{
				Name: "TextPart",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "text"},
					{Name: "text", Type: "string"},
				},
			},
			{
				Name: "ImagePart",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "image"},
					{Name: "url", Type: "string"},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, "type ContentPart struct {") {
		t.Error("missing ContentPart union struct")
	}
	if !strings.Contains(src, "TextPart *TextPart") {
		t.Error("missing TextPart variant field")
	}
	if !strings.Contains(src, "ImagePart *ImagePart") {
		t.Error("missing ImagePart variant field")
	}
	if !strings.Contains(src, "func (u *ContentPart) MarshalJSON()") {
		t.Error("missing MarshalJSON")
	}
	if !strings.Contains(src, "func (u *ContentPart) UnmarshalJSON(data []byte)") {
		t.Error("missing UnmarshalJSON")
	}

	assertCompiles(t, code)
}

func TestGenerateClient(t *testing.T) {
	svc := &contract.Service{
		Name:        "Todo",
		Description: "Todo API",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.example.com",
			Auth:    "bearer",
		},
		Resources: []*contract.Resource{
			{
				Name: "todos",
				Methods: []*contract.Method{
					{
						Name:   "create",
						Input:  "CreateInput",
						Output: "Todo",
						HTTP:   &contract.MethodHTTP{Method: "POST", Path: "/todos"},
					},
					{
						Name:   "list",
						Output: "TodoList",
						HTTP:   &contract.MethodHTTP{Method: "GET", Path: "/todos"},
					},
					{
						Name:  "delete",
						Input: "DeleteInput",
						HTTP:  &contract.MethodHTTP{Method: "DELETE", Path: "/todos/{id}"},
					},
				},
			},
		},
		Types: []*contract.Type{
			{Name: "CreateInput", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "title", Type: "string"}}},
			{Name: "DeleteInput", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}}},
			{Name: "Todo", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}, {Name: "title", Type: "string"}}},
			{Name: "TodoList", Kind: contract.KindSlice, Elem: "Todo"},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)

	// Client struct
	if !strings.Contains(src, "type Client struct {") {
		t.Error("missing Client struct")
	}
	if !strings.Contains(src, "Todos *TodosResource") {
		t.Error("missing Todos resource field")
	}

	// NewClient
	if !strings.Contains(src, "func NewClient(token string, opts ...Option) *Client") {
		t.Error("missing NewClient")
	}
	if !strings.Contains(src, `baseURL: "https://api.example.com"`) {
		t.Error("missing base URL")
	}

	// Options
	if !strings.Contains(src, "type Option func(*Client)") {
		t.Error("missing Option type")
	}
	if !strings.Contains(src, "func WithBaseURL(url string) Option") {
		t.Error("missing WithBaseURL")
	}

	// Resource
	if !strings.Contains(src, "type TodosResource struct {") {
		t.Error("missing TodosResource")
	}

	// Methods
	if !strings.Contains(src, "func (r *TodosResource) Create(ctx context.Context, in *CreateInput) (*Todo, error)") {
		t.Error("missing Create method")
	}
	if !strings.Contains(src, "func (r *TodosResource) List(ctx context.Context) (*TodoList, error)") {
		t.Error("missing List method")
	}
	if !strings.Contains(src, "func (r *TodosResource) Delete(ctx context.Context, in *DeleteInput) error") {
		t.Error("missing Delete method")
	}

	assertCompiles(t, code)
}

func TestGenerateStreaming(t *testing.T) {
	svc := &contract.Service{
		Name: "Streaming",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.example.com",
		},
		Resources: []*contract.Resource{
			{
				Name: "events",
				Methods: []*contract.Method{
					{
						Name:   "stream",
						Input:  "StreamInput",
						Output: "Event",
						Stream: &struct {
							Mode      string             `json:"mode,omitempty" yaml:"mode,omitempty"`
							Item      contract.TypeRef   `json:"item" yaml:"item"`
							Done      contract.TypeRef   `json:"done,omitempty" yaml:"done,omitempty"`
							Error     contract.TypeRef   `json:"error,omitempty" yaml:"error,omitempty"`
							InputItem contract.TypeRef   `json:"input_item,omitempty" yaml:"input_item,omitempty"`
						}{
							Mode: "sse",
							Item: "Event",
						},
						HTTP: &contract.MethodHTTP{Method: "POST", Path: "/events/stream"},
					},
				},
			},
		},
		Types: []*contract.Type{
			{Name: "StreamInput", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "topic", Type: "string"}}},
			{Name: "Event", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "type", Type: "string"}, {Name: "data", Type: "string"}}},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)

	// EventStream generic
	if !strings.Contains(src, "type EventStream[T any] struct {") {
		t.Error("missing EventStream generic type")
	}
	if !strings.Contains(src, "func (s *EventStream[T]) Next() bool") {
		t.Error("missing Next method")
	}
	if !strings.Contains(src, "func (s *EventStream[T]) Event() T") {
		t.Error("missing Event method")
	}
	if !strings.Contains(src, "func (s *EventStream[T]) Err() error") {
		t.Error("missing Err method")
	}
	if !strings.Contains(src, "func (s *EventStream[T]) Close() error") {
		t.Error("missing Close method")
	}

	// Streaming method
	if !strings.Contains(src, "func (r *EventsResource) Stream(ctx context.Context, in *StreamInput) *EventStream[Event]") {
		t.Error("missing Stream method with correct signature")
	}

	assertCompiles(t, code)
}

func TestGenerateEnumField(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "Message",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "role", Type: "string", Enum: []string{"system", "user", "assistant"}},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, "// one of: system, user, assistant") {
		t.Error("missing enum comment")
	}

	assertCompiles(t, code)
}

func TestGenerateConstField(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "TextPart",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "text"},
					{Name: "text", Type: "string"},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, `// always "text"`) {
		t.Error("missing const comment")
	}

	assertCompiles(t, code)
}

func TestGenerateNullableField(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "Response",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "error", Type: "string", Nullable: true},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	// Nullable but not optional should have pointer but no omitempty
	if !strings.Contains(src, `Error *string `+"`"+`json:"error"`+"`") {
		t.Error("nullable field should be pointer without omitempty")
	}

	assertCompiles(t, code)
}

func TestGeneratePrimitives(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "AllTypes",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "s", Type: "string"},
					{Name: "b", Type: "bool"},
					{Name: "i", Type: "int"},
					{Name: "i32", Type: "int32"},
					{Name: "i64", Type: "int64"},
					{Name: "f32", Type: "float32"},
					{Name: "f64", Type: "float64"},
					{Name: "t", Type: "time.Time"},
					{Name: "raw", Type: "json.RawMessage"},
					{Name: "any_field", Type: "any"},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, "S string") {
		t.Error("missing string field")
	}
	if !strings.Contains(src, "B bool") {
		t.Error("missing bool field")
	}
	if !strings.Contains(src, "I int") {
		t.Error("missing int field")
	}
	if !strings.Contains(src, "I32 int32") {
		t.Error("missing int32 field")
	}
	if !strings.Contains(src, "I64 int64") {
		t.Error("missing int64 field")
	}
	if !strings.Contains(src, "F32 float32") {
		t.Error("missing float32 field")
	}
	if !strings.Contains(src, "F64 float64") {
		t.Error("missing float64 field")
	}
	if !strings.Contains(src, "T time.Time") {
		t.Error("missing time.Time field")
	}
	if !strings.Contains(src, "Raw json.RawMessage") {
		t.Error("missing json.RawMessage field")
	}
	if !strings.Contains(src, "AnyField any") {
		t.Error("missing any field")
	}
	if !strings.Contains(src, `"time"`) {
		t.Error("missing time import")
	}

	assertCompiles(t, code)
}

func TestGeneratePackageName(t *testing.T) {
	svc := &contract.Service{
		Name: "MyService",
	}

	// Default package name
	code, _ := Generate(svc, nil)
	if !strings.Contains(string(code), "package myservice") {
		t.Error("default package should be lowercase service name")
	}

	// Custom package name
	code, _ = Generate(svc, &Config{Package: "custom"})
	if !strings.Contains(string(code), "package custom") {
		t.Error("should use custom package name")
	}
}

func TestGenerateOpenAILike(t *testing.T) {
	// OpenAI-like service structure (simplified from samples/openai/api.yaml)
	svc := &contract.Service{
		Name:        "OpenAI",
		Description: "OpenAI-compatible API",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.openai.com",
			Auth:    "bearer",
		},
		Resources: []*contract.Resource{
			{
				Name:        "responses",
				Description: "Create and stream model responses",
				Methods: []*contract.Method{
					{
						Name:        "create",
						Description: "Create a response",
						Input:       "ResponseCreateRequest",
						Output:      "Response",
						HTTP:        &contract.MethodHTTP{Method: "POST", Path: "/v1/responses"},
					},
					{
						Name:        "stream",
						Description: "Stream response events (SSE)",
						Input:       "ResponseCreateRequest",
						Output:      "Response",
						Stream: &struct {
							Mode      string           `json:"mode,omitempty" yaml:"mode,omitempty"`
							Item      contract.TypeRef `json:"item" yaml:"item"`
							Done      contract.TypeRef `json:"done,omitempty" yaml:"done,omitempty"`
							Error     contract.TypeRef `json:"error,omitempty" yaml:"error,omitempty"`
							InputItem contract.TypeRef `json:"input_item,omitempty" yaml:"input_item,omitempty"`
						}{
							Mode: "sse",
							Item: "ResponseEvent",
						},
						HTTP: &contract.MethodHTTP{Method: "POST", Path: "/v1/responses"},
					},
				},
			},
			{
				Name:        "models",
				Description: "List and retrieve available models",
				Methods: []*contract.Method{
					{
						Name:        "list",
						Description: "List models",
						Output:      "ModelList",
						HTTP:        &contract.MethodHTTP{Method: "GET", Path: "/v1/models"},
					},
				},
			},
		},
		Types: []*contract.Type{
			// Request
			{
				Name: "ResponseCreateRequest",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "model", Type: "string", Description: "Model ID"},
					{Name: "input", Type: "InputMessageList"},
					{Name: "stream", Type: "bool", Optional: true},
					{Name: "temperature", Type: "float64", Optional: true},
				},
			},
			{Name: "InputMessageList", Kind: contract.KindSlice, Elem: "InputMessage"},
			{
				Name: "InputMessage",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "role", Type: "string", Enum: []string{"system", "user", "assistant"}},
					{Name: "content", Type: "ContentPartList"},
				},
			},
			{Name: "ContentPartList", Kind: contract.KindSlice, Elem: "ContentPart"},
			// Union for content parts
			{
				Name: "ContentPart",
				Kind: contract.KindUnion,
				Tag:  "type",
				Variants: []contract.Variant{
					{Value: "input_text", Type: "ContentPartInputText"},
					{Value: "input_image", Type: "ContentPartInputImage"},
				},
			},
			{
				Name: "ContentPartInputText",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "input_text"},
					{Name: "text", Type: "string"},
				},
			},
			{
				Name: "ContentPartInputImage",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "input_image"},
					{Name: "image_url", Type: "string"},
				},
			},
			// Response
			{
				Name: "Response",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "object", Type: "string", Const: "response"},
					{Name: "model", Type: "string"},
					{Name: "status", Type: "string", Enum: []string{"in_progress", "completed", "failed"}},
					{Name: "output", Type: "OutputItemList", Optional: true},
				},
			},
			{Name: "OutputItemList", Kind: contract.KindSlice, Elem: "OutputItem"},
			{
				Name: "OutputItem",
				Kind: contract.KindUnion,
				Tag:  "type",
				Variants: []contract.Variant{
					{Value: "output_text", Type: "OutputTextItem"},
					{Value: "tool_call", Type: "ToolCallItem"},
				},
			},
			{
				Name: "OutputTextItem",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "output_text"},
					{Name: "text", Type: "string"},
				},
			},
			{
				Name: "ToolCallItem",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "tool_call"},
					{Name: "tool_name", Type: "string"},
					{Name: "arguments", Type: "json.RawMessage"},
				},
			},
			// Streaming events
			{
				Name: "ResponseEvent",
				Kind: contract.KindUnion,
				Tag:  "type",
				Variants: []contract.Variant{
					{Value: "response.created", Type: "ResponseCreatedEvent"},
					{Value: "response.output_text.delta", Type: "OutputTextDeltaEvent"},
					{Value: "response.completed", Type: "ResponseCompletedEvent"},
				},
			},
			{
				Name: "ResponseCreatedEvent",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "response.created"},
					{Name: "response", Type: "Response"},
				},
			},
			{
				Name: "OutputTextDeltaEvent",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "response.output_text.delta"},
					{Name: "delta", Type: "string"},
				},
			},
			{
				Name: "ResponseCompletedEvent",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string", Const: "response.completed"},
					{Name: "response", Type: "Response"},
				},
			},
			// Models
			{
				Name: "ModelList",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "data", Type: "ModelListItems"},
				},
			},
			{Name: "ModelListItems", Kind: contract.KindSlice, Elem: "Model"},
			{
				Name: "Model",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "object", Type: "string", Const: "model"},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)

	// Verify key structures
	if !strings.Contains(src, "type Client struct {") {
		t.Error("missing Client")
	}
	if !strings.Contains(src, "Responses *ResponsesResource") {
		t.Error("missing Responses resource")
	}
	if !strings.Contains(src, "Models *ModelsResource") {
		t.Error("missing Models resource")
	}
	if !strings.Contains(src, "type ResponseCreateRequest struct {") {
		t.Error("missing ResponseCreateRequest type")
	}
	if !strings.Contains(src, "type Response struct {") {
		t.Error("missing Response type")
	}
	if !strings.Contains(src, "type ContentPart struct {") {
		t.Error("missing ContentPart union")
	}
	if !strings.Contains(src, "func (u *ContentPart) MarshalJSON()") {
		t.Error("missing ContentPart MarshalJSON")
	}

	// Streaming
	if !strings.Contains(src, "type EventStream[T any] struct {") {
		t.Error("missing EventStream for streaming support")
	}
	if !strings.Contains(src, "func (r *ResponsesResource) Stream") {
		t.Error("missing Stream method")
	}

	assertCompiles(t, code)
}

func TestGenerateError(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, "type Error struct {") {
		t.Error("missing Error struct")
	}
	if !strings.Contains(src, "StatusCode int") {
		t.Error("missing StatusCode field")
	}
	if !strings.Contains(src, "func (e *Error) Error() string") {
		t.Error("missing Error() method")
	}
	if !strings.Contains(src, "func decodeError(resp *http.Response) error") {
		t.Error("missing decodeError helper")
	}

	assertCompiles(t, code)
}

func TestGenerateNilService(t *testing.T) {
	_, err := Generate(nil, nil)
	if err == nil {
		t.Error("expected error for nil service")
	}
}

func TestToGoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"id", "ID"},
		{"user_id", "UserID"},
		{"created_at", "CreatedAt"},
		{"base_url", "BaseURL"},
		{"http_client", "HTTPClient"},
		{"api_key", "APIKey"},
		{"sse_mode", "SSEMode"},
		{"json_data", "JSONData"},
		{"content-type", "ContentType"},
		{"response.created", "ResponseCreated"},
	}

	for _, tc := range tests {
		got := toGoName(tc.input)
		if got != tc.expected {
			t.Errorf("toGoName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestGenerateMultipleResources(t *testing.T) {
	svc := &contract.Service{
		Name: "Multi",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.example.com",
		},
		Resources: []*contract.Resource{
			{
				Name: "users",
				Methods: []*contract.Method{
					{Name: "list", Output: "UserList", HTTP: &contract.MethodHTTP{Method: "GET", Path: "/users"}},
				},
			},
			{
				Name: "posts",
				Methods: []*contract.Method{
					{Name: "list", Output: "PostList", HTTP: &contract.MethodHTTP{Method: "GET", Path: "/posts"}},
				},
			},
		},
		Types: []*contract.Type{
			{Name: "UserList", Kind: contract.KindSlice, Elem: "User"},
			{Name: "User", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}}},
			{Name: "PostList", Kind: contract.KindSlice, Elem: "Post"},
			{Name: "Post", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}}},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, "Users *UsersResource") {
		t.Error("missing Users resource")
	}
	if !strings.Contains(src, "Posts *PostsResource") {
		t.Error("missing Posts resource")
	}
	if !strings.Contains(src, "type UsersResource struct {") {
		t.Error("missing UsersResource struct")
	}
	if !strings.Contains(src, "type PostsResource struct {") {
		t.Error("missing PostsResource struct")
	}

	assertCompiles(t, code)
}

func assertCompiles(t *testing.T, code []byte) {
	t.Helper()
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "generated.go", code, parser.AllErrors)
	if err != nil {
		t.Errorf("generated code doesn't compile:\n%v\n\nCode:\n%s", err, string(code))
	}
}

func TestGenerateNoHTTPBinding(t *testing.T) {
	// Methods without HTTP binding should use default POST
	svc := &contract.Service{
		Name: "Test",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.example.com",
		},
		Resources: []*contract.Resource{
			{
				Name: "items",
				Methods: []*contract.Method{
					{
						Name:   "process",
						Input:  "ProcessInput",
						Output: "ProcessOutput",
						// No HTTP binding
					},
				},
			},
		},
		Types: []*contract.Type{
			{Name: "ProcessInput", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "data", Type: "string"}}},
			{Name: "ProcessOutput", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "result", Type: "string"}}},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	// Should have the method
	if !strings.Contains(src, "func (r *ItemsResource) Process") {
		t.Error("missing Process method")
	}
	// Default path should be resource name
	if !strings.Contains(src, `"/items"`) {
		t.Error("missing default path")
	}

	assertCompiles(t, code)
}

func TestGenerateDescription(t *testing.T) {
	svc := &contract.Service{
		Name:        "Test",
		Description: "Test API description",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.example.com",
		},
		Resources: []*contract.Resource{
			{
				Name:        "items",
				Description: "handles item operations",
				Methods: []*contract.Method{
					{
						Name:        "create",
						Description: "creates a new item",
						Input:       "CreateInput",
						Output:      "Item",
						HTTP:        &contract.MethodHTTP{Method: "POST", Path: "/items"},
					},
				},
			},
		},
		Types: []*contract.Type{
			{
				Name:        "CreateInput",
				Kind:        contract.KindStruct,
				Description: "input for creating an item",
				Fields: []contract.Field{
					{Name: "name", Type: "string", Description: "the item name"},
				},
			},
			{Name: "Item", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}}},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	// Type description
	if !strings.Contains(src, "// CreateInput input for creating an item") {
		t.Error("missing type description")
	}
	// Field description
	if !strings.Contains(src, "// the item name") {
		t.Error("missing field description")
	}
	// Method description
	if !strings.Contains(src, "// Create creates a new item") {
		t.Error("missing method description")
	}
	// Resource description
	if !strings.Contains(src, "// ItemsResource handles item operations") {
		t.Error("missing resource description")
	}

	assertCompiles(t, code)
}

func TestGenerateEmptyService(t *testing.T) {
	svc := &contract.Service{
		Name: "Empty",
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	// Should still have client
	if !strings.Contains(src, "type Client struct {") {
		t.Error("missing Client struct even for empty service")
	}
	if !strings.Contains(src, "func NewClient(token string, opts ...Option) *Client") {
		t.Error("missing NewClient even for empty service")
	}

	assertCompiles(t, code)
}

func TestGenerateNestedSliceType(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "Matrix",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "rows", Type: "[][]int32"},
				},
			},
		},
	}

	code, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	src := string(code)
	if !strings.Contains(src, "Rows [][]int32") {
		t.Error("missing nested slice type")
	}

	assertCompiles(t, code)
}
