package sdkerlang

import (
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := Generate(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil service")
	}
	if !strings.Contains(err.Error(), "nil service") {
		t.Errorf("expected 'nil service' in error, got: %v", err)
	}
}

func TestGenerate_EmptyService(t *testing.T) {
	svc := &contract.Service{
		Name: "TestService",
	}
	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one file")
	}
}

func TestGenerate_BasicService(t *testing.T) {
	svc := &contract.Service{
		Name:        "TestAPI",
		Description: "A test API service",
		Client: &contract.Client{
			BaseURL: "https://api.example.com",
			Auth:    "bearer",
			Headers: map[string]string{
				"X-Custom-Header": "value",
			},
		},
		Resources: []*contract.Resource{
			{
				Name:        "users",
				Description: "User operations",
				Methods: []*contract.Method{
					{
						Name:        "create",
						Description: "Create a user",
						Input:       "CreateUserRequest",
						Output:      "User",
						HTTP: &contract.MethodHTTP{
							Method: "POST",
							Path:   "/v1/users",
						},
					},
					{
						Name:        "get",
						Description: "Get a user",
						Output:      "User",
						HTTP: &contract.MethodHTTP{
							Method: "GET",
							Path:   "/v1/users/{id}",
						},
					},
				},
			},
		},
		Types: []*contract.Type{
			{
				Name: "CreateUserRequest",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "name", Type: "string", Description: "User name"},
					{Name: "email", Type: "string", Description: "User email"},
					{Name: "age", Type: "int", Optional: true},
				},
			},
			{
				Name: "User",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "name", Type: "string"},
					{Name: "email", Type: "string"},
					{Name: "age", Type: "int", Optional: true},
					{Name: "createdAt", Type: "time.Time"},
				},
			},
		},
	}

	files, err := Generate(svc, &Config{
		AppName: "test_api",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check expected files exist
	expectedFiles := map[string]bool{
		"rebar.config":               false,
		"src/test_api.app.src":       false,
		"src/test_api.erl":           false,
		"src/test_api_client.erl":    false,
		"src/test_api_config.erl":    false,
		"src/test_api_types.erl":     false,
		"src/test_api_streaming.erl": false,
		"src/test_api_errors.erl":    false,
		"src/test_api_users.erl":     false,
		"include/test_api.hrl":       false,
	}

	for _, f := range files {
		if _, ok := expectedFiles[f.Path]; ok {
			expectedFiles[f.Path] = true
		}
	}

	for path, found := range expectedFiles {
		if !found {
			t.Errorf("expected file not found: %s", path)
		}
	}
}

func TestGenerate_WithStreaming(t *testing.T) {
	svc := &contract.Service{
		Name: "StreamingAPI",
		Client: &contract.Client{
			BaseURL: "https://api.example.com",
			Auth:    "bearer",
		},
		Resources: []*contract.Resource{
			{
				Name: "messages",
				Methods: []*contract.Method{
					{
						Name:   "create",
						Input:  "CreateMessageRequest",
						Output: "Message",
						HTTP:   &contract.MethodHTTP{Method: "POST", Path: "/v1/messages"},
					},
					{
						Name:  "stream",
						Input: "CreateMessageRequest",
						Stream: &contract.MethodStream{
							Mode: "sse",
							Item: "MessageStreamEvent",
						},
						HTTP: &contract.MethodHTTP{Method: "POST", Path: "/v1/messages"},
					},
				},
			},
		},
		Types: []*contract.Type{
			{
				Name: "CreateMessageRequest",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "model", Type: "string"},
					{Name: "messages", Type: "[]Message"},
				},
			},
			{
				Name: "Message",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "role", Type: "string"},
					{Name: "content", Type: "string"},
				},
			},
			{
				Name: "MessageStreamEvent",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string"},
					{Name: "delta", Type: "Delta", Optional: true},
				},
			},
			{
				Name: "Delta",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string"},
					{Name: "text", Type: "string", Optional: true},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check streaming is included in client
	var clientContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "_client.erl") {
			clientContent = f.Content
			break
		}
	}

	if !strings.Contains(clientContent, "stream/2") {
		t.Error("expected stream/2 export in client with SSE methods")
	}

	// Check messages resource has streaming method
	var messagesContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "_messages.erl") {
			messagesContent = f.Content
			break
		}
	}

	if !strings.Contains(messagesContent, "stream/2") {
		t.Error("expected stream/2 in messages resource")
	}
}

func TestGenerate_UnionTypes(t *testing.T) {
	svc := &contract.Service{
		Name: "UnionAPI",
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
					{Name: "type", Type: "string"},
					{Name: "text", Type: "string"},
				},
			},
			{
				Name: "ImageBlock",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string"},
					{Name: "url", Type: "string"},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check header file has union type definition
	var hrlContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".hrl") {
			hrlContent = f.Content
			break
		}
	}

	if !strings.Contains(hrlContent, "content_block()") {
		t.Error("expected content_block() type in header")
	}
	if !strings.Contains(hrlContent, "text_block") {
		t.Error("expected text_block variant in header")
	}
}

func TestGenerate_MapAndSliceTypes(t *testing.T) {
	svc := &contract.Service{
		Name: "MapSliceAPI",
		Types: []*contract.Type{
			{
				Name: "Request",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "items", Type: "[]string"},
					{Name: "metadata", Type: "map[string]any"},
					{Name: "tags", Type: "[]Tag"},
				},
			},
			{
				Name: "Tag",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "key", Type: "string"},
					{Name: "value", Type: "string"},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hrlContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".hrl") {
			hrlContent = f.Content
			break
		}
	}

	if !strings.Contains(hrlContent, "list(binary())") {
		t.Error("expected list(binary()) type spec for []string")
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
		{"HTTPServer", "httpserver"},
		{"getHTTPResponse", "get_httpresponse"},
		{"ID", "id"},
		{"userID", "user_id"},
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
		{"hello.world", "HelloWorld"},
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

func TestIsErlangReserved(t *testing.T) {
	reserved := []string{
		"after", "and", "andalso", "band", "begin", "bnot", "bor",
		"case", "catch", "cond", "div", "end", "fun", "if", "let",
		"not", "of", "or", "orelse", "receive", "rem", "try", "when",
	}

	for _, word := range reserved {
		if !isErlangReserved(word) {
			t.Errorf("expected %q to be reserved", word)
		}
	}

	nonReserved := []string{"hello", "world", "foo", "bar"}
	for _, word := range nonReserved {
		if isErlangReserved(word) {
			t.Errorf("expected %q to NOT be reserved", word)
		}
	}
}

func TestErlangBinary(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", `<<"hello">>`},
		{"hello\nworld", `<<"hello\nworld">>`},
		{`hello"world`, `<<"hello\"world">>`},
		{"hello\\world", `<<"hello\\world">>`},
	}

	for _, tt := range tests {
		result := erlangBinary(tt.input)
		if result != tt.expected {
			t.Errorf("erlangBinary(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestErlangAtom(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello_world", "hello_world"},
		{"123abc", "'123abc'"},
		{"after", "'after'"},
		{"", ""},
	}

	for _, tt := range tests {
		result := erlangAtom(tt.input)
		if result != tt.expected {
			t.Errorf("erlangAtom(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerate_EnumValues(t *testing.T) {
	svc := &contract.Service{
		Name: "EnumAPI",
		Types: []*contract.Type{
			{
				Name: "Message",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{
						Name: "role",
						Type: "string",
						Enum: []string{"user", "assistant", "system"},
					},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hrlContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".hrl") {
			hrlContent = f.Content
			break
		}
	}

	if !strings.Contains(hrlContent, "MESSAGE_ROLE_USER") {
		t.Error("expected MESSAGE_ROLE_USER macro in header")
	}
	if !strings.Contains(hrlContent, "MESSAGE_ROLE_ASSISTANT") {
		t.Error("expected MESSAGE_ROLE_ASSISTANT macro in header")
	}
}

func TestGenerate_CustomConfig(t *testing.T) {
	svc := &contract.Service{
		Name:        "MyAPI",
		Description: "My custom API",
	}

	cfg := &Config{
		AppName:     "my_custom_app",
		Version:     "2.0.0",
		Description: "Custom description",
	}

	files, err := Generate(svc, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check app name is used correctly
	var found bool
	for _, f := range files {
		if f.Path == "src/my_custom_app.erl" {
			found = true
			if !strings.Contains(f.Content, "-module(my_custom_app).") {
				t.Error("expected custom app name in module declaration")
			}
			break
		}
	}

	if !found {
		t.Error("expected file with custom app name")
	}

	// Check version in app.src
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".app.src") {
			if !strings.Contains(f.Content, `"2.0.0"`) {
				t.Error("expected version 2.0.0 in app.src")
			}
			break
		}
	}
}

func TestGenerate_AllPrimitiveTypes(t *testing.T) {
	svc := &contract.Service{
		Name: "PrimitiveAPI",
		Types: []*contract.Type{
			{
				Name: "AllTypes",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "strField", Type: "string"},
					{Name: "boolField", Type: "bool"},
					{Name: "intField", Type: "int"},
					{Name: "int8Field", Type: "int8"},
					{Name: "int16Field", Type: "int16"},
					{Name: "int32Field", Type: "int32"},
					{Name: "int64Field", Type: "int64"},
					{Name: "uintField", Type: "uint"},
					{Name: "uint8Field", Type: "uint8"},
					{Name: "float32Field", Type: "float32"},
					{Name: "float64Field", Type: "float64"},
					{Name: "timeField", Type: "time.Time"},
					{Name: "rawField", Type: "json.RawMessage"},
					{Name: "anyField", Type: "any"},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hrlContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".hrl") {
			hrlContent = f.Content
			break
		}
	}

	// Check type mappings
	expectedTypes := []string{
		"binary()",
		"boolean()",
		"integer()",
		"non_neg_integer()",
		"float()",
		"calendar:datetime()",
		"map()",
		"term()",
	}

	for _, expected := range expectedTypes {
		if !strings.Contains(hrlContent, expected) {
			t.Errorf("expected type spec %q in header", expected)
		}
	}
}
