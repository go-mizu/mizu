package sdkzig

import (
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := Generate(nil, nil)
	if err == nil {
		t.Error("expected error for nil service")
	}
	if !strings.Contains(err.Error(), "nil service") {
		t.Errorf("expected 'nil service' in error, got: %v", err)
	}
}

func TestGenerate_EmptyService(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 8 {
		t.Errorf("expected 8 files, got %d", len(files))
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := &contract.Service{
		Name:        "TestAPI",
		Description: "A test API",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.example.com",
			Auth:    "bearer",
		},
		Resources: []*contract.Resource{
			{
				Name:        "messages",
				Description: "Message operations",
				Methods: []*contract.Method{
					{
						Name:        "create",
						Description: "Create a message",
						Input:       "CreateRequest",
						Output:      "Message",
						HTTP:        &contract.MethodHTTP{Method: "POST", Path: "/v1/messages"},
					},
				},
			},
		},
		Types: []*contract.Type{
			{
				Name: "CreateRequest",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "content", Type: "string", Description: "The message content"},
					{Name: "model", Type: "string", Description: "The model to use"},
				},
			},
			{
				Name: "Message",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "content", Type: "string"},
				},
			},
		},
	}

	files, err := Generate(svc, &Config{
		Package: "test_api",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFiles := map[string]bool{
		"build.zig":          false,
		"build.zig.zon":      false,
		"src/root.zig":       false,
		"src/client.zig":     false,
		"src/types.zig":      false,
		"src/resources.zig":  false,
		"src/streaming.zig":  false,
		"src/errors.zig":     false,
	}

	for _, f := range files {
		if _, ok := expectedFiles[f.Path]; !ok {
			t.Errorf("unexpected file: %s", f.Path)
		}
		expectedFiles[f.Path] = true
	}

	for path, found := range expectedFiles {
		if !found {
			t.Errorf("missing expected file: %s", path)
		}
	}
}

func TestGenerate_TypeMapping(t *testing.T) {
	svc := &contract.Service{
		Name: "TypeTest",
		Types: []*contract.Type{
			{
				Name: "AllTypes",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "str", Type: "string"},
					{Name: "b", Type: "bool"},
					{Name: "i32", Type: "int32"},
					{Name: "i64", Type: "int64"},
					{Name: "f32", Type: "float32"},
					{Name: "f64", Type: "float64"},
					{Name: "any_val", Type: "json.RawMessage"},
					{Name: "time", Type: "time.Time"},
					{Name: "opt", Type: "string", Optional: true},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var typesFile string
	for _, f := range files {
		if f.Path == "src/types.zig" {
			typesFile = f.Content
			break
		}
	}

	if typesFile == "" {
		t.Fatal("types.zig not found")
	}

	// Check type mappings
	checks := []string{
		"str: []const u8",
		"b: bool",
		"i32: i32",
		"i64: i64",
		"f32: f32",
		"f64: f64",
		"any_val: std.json.Value",
		"time: i64",
		"opt: ?[]const u8",
	}

	for _, check := range checks {
		if !strings.Contains(typesFile, check) {
			t.Errorf("expected types.zig to contain %q", check)
		}
	}
}

func TestGenerate_StreamingMethods(t *testing.T) {
	svc := &contract.Service{
		Name: "StreamTest",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.example.com",
		},
		Resources: []*contract.Resource{
			{
				Name: "messages",
				Methods: []*contract.Method{
					{
						Name:   "stream",
						Input:  "Request",
						Stream: &contract.MethodStream{Mode: "sse", Item: "Event"},
						HTTP:   &contract.MethodHTTP{Method: "POST", Path: "/stream"},
					},
				},
			},
		},
		Types: []*contract.Type{
			{Name: "Request", Kind: contract.KindStruct},
			{Name: "Event", Kind: contract.KindStruct},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resourcesFile string
	for _, f := range files {
		if f.Path == "src/resources.zig" {
			resourcesFile = f.Content
			break
		}
	}

	if !strings.Contains(resourcesFile, "streaming.EventStream") {
		t.Error("expected streaming method to use EventStream")
	}
}

func TestGenerate_UnionTypes(t *testing.T) {
	svc := &contract.Service{
		Name: "UnionTest",
		Types: []*contract.Type{
			{
				Name: "ContentBlock",
				Kind: contract.KindUnion,
				Tag:  "type",
				Variants: []contract.Variant{
					{Value: "text", Type: "TextBlock"},
					{Value: "image", Type: "ImageBlock"},
				},
			},
			{
				Name: "TextBlock",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "text", Type: "string"},
				},
			},
			{
				Name: "ImageBlock",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "url", Type: "string"},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var typesFile string
	for _, f := range files {
		if f.Path == "src/types.zig" {
			typesFile = f.Content
			break
		}
	}

	// Check union type generation
	if !strings.Contains(typesFile, "pub const ContentBlock = union(enum)") {
		t.Error("expected union type definition")
	}

	if !strings.Contains(typesFile, "text: TextBlock") {
		t.Error("expected text variant")
	}

	if !strings.Contains(typesFile, "image: ImageBlock") {
		t.Error("expected image variant")
	}

	if !strings.Contains(typesFile, "pub fn isText(") {
		t.Error("expected isText helper method")
	}

	if !strings.Contains(typesFile, "pub fn asText(") {
		t.Error("expected asText helper method")
	}
}

func TestGenerate_OptionalFields(t *testing.T) {
	svc := &contract.Service{
		Name: "OptionalTest",
		Types: []*contract.Type{
			{
				Name: "Request",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "required", Type: "string"},
					{Name: "optional", Type: "string", Optional: true},
					{Name: "nullable", Type: "int32", Nullable: true},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var typesFile string
	for _, f := range files {
		if f.Path == "src/types.zig" {
			typesFile = f.Content
			break
		}
	}

	if !strings.Contains(typesFile, "required: []const u8") {
		t.Error("expected required field without optional marker")
	}

	if !strings.Contains(typesFile, "optional: ?[]const u8") {
		t.Error("expected optional field with ? marker")
	}

	if !strings.Contains(typesFile, "nullable: ?i32") {
		t.Error("expected nullable field with ? marker")
	}
}

func TestGenerate_SliceAndMapTypes(t *testing.T) {
	svc := &contract.Service{
		Name: "CollectionTest",
		Types: []*contract.Type{
			{
				Name: "Container",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "items", Type: "[]string"},
					{Name: "nested", Type: "[]Item"},
					{Name: "meta", Type: "map[string]string"},
				},
			},
			{
				Name: "Item",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "value", Type: "string"},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var typesFile string
	for _, f := range files {
		if f.Path == "src/types.zig" {
			typesFile = f.Content
			break
		}
	}

	if !strings.Contains(typesFile, "items: []const []const u8") {
		t.Error("expected slice of strings")
	}

	if !strings.Contains(typesFile, "nested: []const Item") {
		t.Error("expected slice of Item")
	}
}

func TestGenerate_DefaultHeaders(t *testing.T) {
	svc := &contract.Service{
		Name: "HeaderTest",
		Defaults: &contract.Defaults{
			BaseURL: "https://api.example.com",
			Headers: map[string]string{
				"X-Custom-Header": "value",
				"Api-Version":     "2024-01-01",
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var clientFile string
	for _, f := range files {
		if f.Path == "src/client.zig" {
			clientFile = f.Content
			break
		}
	}

	if !strings.Contains(clientFile, "X-Custom-Header") {
		t.Error("expected custom header in client")
	}

	if !strings.Contains(clientFile, "Api-Version") {
		t.Error("expected api version header in client")
	}
}

func TestGenerate_ReservedWords(t *testing.T) {
	svc := &contract.Service{
		Name: "ReservedTest",
		Types: []*contract.Type{
			{
				Name: "TypeWithReserved",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string"},
					{Name: "error", Type: "string"},
					{Name: "async", Type: "bool"},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var typesFile string
	for _, f := range files {
		if f.Path == "src/types.zig" {
			typesFile = f.Content
			break
		}
	}

	// Reserved words should be escaped with @""
	if !strings.Contains(typesFile, "@\"type\"") {
		t.Error("expected 'type' to be escaped")
	}

	if !strings.Contains(typesFile, "@\"error\"") {
		t.Error("expected 'error' to be escaped")
	}

	if !strings.Contains(typesFile, "@\"async\"") {
		t.Error("expected 'async' to be escaped")
	}
}

func TestToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HelloWorld", "hello_world"},
		{"helloWorld", "hello_world"},
		{"hello_world", "hello_world"},
		{"hello-world", "hello_world"},
		{"HTTPServer", "http_server"},
		{"getHTTPResponse", "get_http_response"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toSnake(tt.input)
		if result != tt.expected {
			t.Errorf("toSnake(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestToPascal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "HelloWorld"},
		{"hello-world", "HelloWorld"},
		{"hello world", "HelloWorld"},
		{"helloWorld", "HelloWorld"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toPascal(tt.input)
		if result != tt.expected {
			t.Errorf("toPascal(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsZigReserved(t *testing.T) {
	reserved := []string{
		"type", "error", "async", "await", "break", "continue",
		"const", "var", "fn", "pub", "return", "struct", "enum",
		"union", "if", "else", "for", "while", "switch", "try",
	}

	for _, word := range reserved {
		if !isZigReserved(word) {
			t.Errorf("expected %q to be reserved", word)
		}
	}

	nonReserved := []string{"hello", "world", "message", "content"}
	for _, word := range nonReserved {
		if isZigReserved(word) {
			t.Errorf("expected %q to not be reserved", word)
		}
	}
}

func TestZigQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "\"hello\""},
		{"hello\"world", "\"hello\\\"world\""},
		{"hello\nworld", "\"hello\\nworld\""},
		{"hello\tworld", "\"hello\\tworld\""},
		{"hello\\world", "\"hello\\\\world\""},
	}

	for _, tt := range tests {
		result := zigQuote(tt.input)
		if result != tt.expected {
			t.Errorf("zigQuote(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
