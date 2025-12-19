package sdkjava

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
}

func TestGenerate_Basic(t *testing.T) {
	svc := &contract.Service{
		Name:        "Anthropic",
		Description: "Claude API client",
		Client: &contract.Client{
			BaseURL: "https://api.anthropic.com",
			Auth:    "bearer",
			Headers: map[string]string{
				"anthropic-version": "2024-01-01",
			},
		},
		Resources: []*contract.Resource{
			{
				Name:        "messages",
				Description: "Message operations",
				Methods: []*contract.Method{
					{
						Name:        "create",
						Description: "Create a message",
						Input:       "CreateMessageRequest",
						Output:      "Message",
						HTTP:        &contract.MethodHTTP{Method: "POST", Path: "/v1/messages"},
					},
					{
						Name:        "stream",
						Description: "Stream a message",
						Input:       "CreateMessageRequest",
						Stream:      &contract.MethodStream{Mode: "sse", Item: "MessageStreamEvent"},
						HTTP:        &contract.MethodHTTP{Method: "POST", Path: "/v1/messages"},
					},
				},
			},
		},
		Types: []*contract.Type{
			{
				Name: "CreateMessageRequest",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "model", Type: "string", Description: "The model to use"},
					{Name: "messages", Type: "[]Message", Description: "The messages"},
					{Name: "max_tokens", Type: "int", Description: "Max tokens to generate"},
					{Name: "stream", Type: "bool", Optional: true},
				},
			},
			{
				Name: "Message",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "type", Type: "string"},
					{Name: "role", Type: "string", Enum: []string{"user", "assistant"}},
					{Name: "content", Type: "[]ContentBlock"},
				},
			},
			{
				Name: "ContentBlock",
				Kind: contract.KindUnion,
				Tag:  "type",
				Variants: []contract.Variant{
					{Value: "text", Type: "TextBlock"},
					{Value: "tool_use", Type: "ToolUseBlock"},
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
				Name: "ToolUseBlock",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string"},
					{Name: "id", Type: "string"},
					{Name: "name", Type: "string"},
					{Name: "input", Type: "json.RawMessage"},
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

	files, err := Generate(svc, &Config{
		Package:    "com.anthropic.sdk",
		GroupId:    "com.anthropic",
		ArtifactId: "anthropic-sdk",
		Version:    "1.0.0",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check expected files
	expectedFiles := []string{
		"pom.xml",
		"AnthropicClient.java",
		"ClientOptions.java",
		"AuthMode.java",
		"Types.java",
		"Resources.java",
		"HttpClientWrapper.java",
		"SSEReader.java",
		"Exceptions.java",
	}

	fileMap := make(map[string]string)
	for _, f := range files {
		fileMap[f.Path] = f.Content
	}

	for _, expected := range expectedFiles {
		found := false
		for path := range fileMap {
			if strings.HasSuffix(path, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected file %s not found in output", expected)
		}
	}

	// Print files for inspection
	for _, f := range files {
		t.Logf("=== %s ===\n%s\n", f.Path, f.Content)
	}
}

func TestGenerate_TypeMapping(t *testing.T) {
	svc := &contract.Service{
		Name: "Test",
		Types: []*contract.Type{
			{
				Name: "AllTypes",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "str", Type: "string"},
					{Name: "b", Type: "bool"},
					{Name: "i", Type: "int"},
					{Name: "i8", Type: "int8"},
					{Name: "i16", Type: "int16"},
					{Name: "i32", Type: "int32"},
					{Name: "i64", Type: "int64"},
					{Name: "f32", Type: "float32"},
					{Name: "f64", Type: "float64"},
					{Name: "dt", Type: "time.Time"},
					{Name: "raw", Type: "json.RawMessage"},
					{Name: "any_val", Type: "any"},
					{Name: "list", Type: "[]string"},
					{Name: "map", Type: "map[string]int"},
					{Name: "optional_str", Type: "string", Optional: true},
					{Name: "nullable_int", Type: "int", Nullable: true},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Find Types.java
	var typesContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.java") {
			typesContent = f.Content
			break
		}
	}

	if typesContent == "" {
		t.Fatal("Types.java not found")
	}

	// Check type mappings
	expectedMappings := []string{
		"String str",
		"boolean b",
		"int i",
		"byte i8",
		"short i16",
		"int i32",
		"long i64",
		"float f32",
		"double f64",
		"Instant dt",
		"JsonNode raw",
		"JsonNode anyVal",
		"List<String> list",
		"Map<String, Integer> map",
		"String optionalStr",
		"Integer nullableInt",
	}

	for _, expected := range expectedMappings {
		if !strings.Contains(typesContent, expected) {
			t.Errorf("expected type mapping %q not found in Types.java", expected)
		}
	}
}
