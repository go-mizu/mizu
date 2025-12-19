package sdkclojure

import (
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
)

func TestGenerate(t *testing.T) {
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
		Namespace:     "anthropic.sdk",
		GroupId:       "com.anthropic",
		Version:       "1.0.0",
		GenerateSpecs: true,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify expected files
	// Namespace "anthropic.sdk" -> path "anthropic/sdk"
	expectedFiles := map[string]bool{
		"deps.edn":                       false,
		"src/anthropic/sdk/core.clj":      false,
		"src/anthropic/sdk/types.clj":     false,
		"src/anthropic/sdk/resources.clj": false,
		"src/anthropic/sdk/streaming.clj": false,
		"src/anthropic/sdk/errors.clj":    false,
		"src/anthropic/sdk/spec.clj":      false,
	}

	for _, f := range files {
		t.Logf("Generated file: %s", f.Path)
		if _, ok := expectedFiles[f.Path]; ok {
			expectedFiles[f.Path] = true
		}
	}

	for path, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file not generated: %s", path)
		}
	}

	// Verify content
	for _, f := range files {
		if f.Content == "" {
			t.Errorf("Empty content for file: %s", f.Path)
		}

		// Verify namespace in Clojure files
		if strings.HasSuffix(f.Path, ".clj") {
			if !strings.Contains(f.Content, "(ns anthropic.sdk.") {
				t.Errorf("File %s missing namespace declaration", f.Path)
			}
		}

		// Verify deps.edn content
		if f.Path == "deps.edn" {
			if !strings.Contains(f.Content, "clj-http") {
				t.Errorf("deps.edn missing clj-http dependency")
			}
			if !strings.Contains(f.Content, "cheshire") {
				t.Errorf("deps.edn missing cheshire dependency")
			}
			if !strings.Contains(f.Content, "core.async") {
				t.Errorf("deps.edn missing core.async dependency")
			}
		}

		// Verify core.clj has client creation
		if strings.HasSuffix(f.Path, "core.clj") {
			if !strings.Contains(f.Content, "defn create-client") {
				t.Errorf("core.clj missing create-client function")
			}
			if !strings.Contains(f.Content, "messages-create") {
				t.Errorf("core.clj missing messages-create function")
			}
			if !strings.Contains(f.Content, "messages-stream") {
				t.Errorf("core.clj missing messages-stream function")
			}
		}

		// Verify types.clj has record definitions
		if strings.HasSuffix(f.Path, "types.clj") {
			if !strings.Contains(f.Content, "defrecord Message") {
				t.Errorf("types.clj missing Message record")
			}
			if !strings.Contains(f.Content, "defn ->message") {
				t.Errorf("types.clj missing ->message coercion function")
			}
		}

		// Verify streaming.clj has SSE support
		if strings.HasSuffix(f.Path, "streaming.clj") {
			if !strings.Contains(f.Content, "defn stream-request") {
				t.Errorf("streaming.clj missing stream-request function")
			}
			if !strings.Contains(f.Content, "parse-sse-line") {
				t.Errorf("streaming.clj missing SSE parsing")
			}
		}

		// Verify errors.clj has error types
		if strings.HasSuffix(f.Path, "errors.clj") {
			if !strings.Contains(f.Content, "defn api-error") {
				t.Errorf("errors.clj missing api-error function")
			}
			if !strings.Contains(f.Content, "defn retryable?") {
				t.Errorf("errors.clj missing retryable? function")
			}
		}
	}
}

func TestGenerateNilService(t *testing.T) {
	_, err := Generate(nil, nil)
	if err == nil {
		t.Error("Expected error for nil service")
	}
}

func TestGenerateMinimal(t *testing.T) {
	svc := &contract.Service{
		Name: "SimpleAPI",
		Resources: []*contract.Resource{
			{
				Name: "users",
				Methods: []*contract.Method{
					{
						Name:   "list",
						Output: "UserList",
						HTTP:   &contract.MethodHTTP{Method: "GET", Path: "/users"},
					},
				},
			},
		},
		Types: []*contract.Type{
			{
				Name: "UserList",
				Kind: contract.KindSlice,
				Elem: "User",
			},
			{
				Name: "User",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "name", Type: "string"},
				},
			},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(files) < 6 {
		t.Errorf("Expected at least 6 files, got %d", len(files))
	}

	// Verify namespace defaults to kebab-case service name
	// "SimpleAPI" -> "simple-api" (camelCase to kebab-case)
	for _, f := range files {
		if strings.HasSuffix(f.Path, "core.clj") {
			if !strings.Contains(f.Content, "(ns simple-api.core") {
				t.Logf("Content: %s", f.Content[:min(500, len(f.Content))])
				t.Errorf("Unexpected namespace in core.clj, expected 'simple-api.core'")
			}
		}
	}
}

func TestNamingConventions(t *testing.T) {
	tests := []struct {
		input    string
		kebab    string
		pascal   string
		camel    string
		keyword  string
	}{
		{"userId", "user-id", "UserId", "userId", ":user-id"},
		{"user_name", "user-name", "UserName", "userName", ":user-name"},
		{"ID", "id", "ID", "iD", ":id"},                           // All-caps stays lowercase in kebab
		{"HTTPMethod", "httpmethod", "HTTPMethod", "hTTPMethod", ":httpmethod"}, // All-caps run treated as single word
		{"simple", "simple", "Simple", "simple", ":simple"},
	}

	for _, tt := range tests {
		if got := toKebab(tt.input); got != tt.kebab {
			t.Errorf("toKebab(%q) = %q, want %q", tt.input, got, tt.kebab)
		}
		if got := toPascal(tt.input); got != tt.pascal {
			t.Errorf("toPascal(%q) = %q, want %q", tt.input, got, tt.pascal)
		}
		if got := toCamel(tt.input); got != tt.camel {
			t.Errorf("toCamel(%q) = %q, want %q", tt.input, got, tt.camel)
		}
		if got := toCljKeyword(tt.input); got != tt.keyword {
			t.Errorf("toCljKeyword(%q) = %q, want %q", tt.input, got, tt.keyword)
		}
	}
}

func TestReservedWords(t *testing.T) {
	reserved := []string{"def", "if", "fn", "let", "map", "filter", "type", "name"}

	for _, word := range reserved {
		if !isCljReserved(word) {
			t.Errorf("Expected %q to be reserved", word)
		}
	}

	notReserved := []string{"user", "message", "create", "handle"}
	for _, word := range notReserved {
		if isCljReserved(word) {
			t.Errorf("Expected %q to not be reserved", word)
		}
	}
}

func TestNsToPath(t *testing.T) {
	tests := []struct {
		ns   string
		path string
	}{
		{"my-api", "my_api"},
		{"my-api.core", "my_api/core"},
		{"com.example.my-sdk", "com/example/my_sdk"},
	}

	for _, tt := range tests {
		if got := nsToPath(tt.ns); got != tt.path {
			t.Errorf("nsToPath(%q) = %q, want %q", tt.ns, got, tt.path)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
