package sdkc_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkc "github.com/go-mizu/mizu/contract/v2/sdk/x/c"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkc.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "anthropic", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"CMakeLists.txt":                   false,
		"Makefile":                         false,
		"include/anthropic/anthropic.h":   false,
		"include/anthropic/errors.h":      false,
		"include/anthropic/client.h":      false,
		"include/anthropic/types.h":       false,
		"include/anthropic/resources.h":   false,
		"include/anthropic/streaming.h":   false,
		"src/errors.c":                    false,
		"src/client.c":                    false,
		"src/types.c":                     false,
		"src/resources.c":                 false,
		"src/streaming.c":                 false,
		"src/internal.h":                  false,
	}

	for _, f := range files {
		if _, ok := expected[f.Path]; ok {
			expected[f.Path] = true
		}
	}

	for path, found := range expected {
		if !found {
			t.Errorf("expected file %s not found in output", path)
		}
	}
}

func TestGenerate_CMakeLists_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{
		Package: "myapi",
		Version: "1.2.3",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var cmake string
	for _, f := range files {
		if f.Path == "CMakeLists.txt" {
			cmake = f.Content
			break
		}
	}

	if !strings.Contains(cmake, "project(myapi VERSION 1.2.3 LANGUAGES C)") {
		t.Errorf("CMakeLists.txt should contain project with version, got:\n%s", cmake)
	}
	if !strings.Contains(cmake, "MYAPI_SOURCES") {
		t.Errorf("CMakeLists.txt should contain MYAPI_SOURCES, got:\n%s", cmake)
	}
}

func TestGenerate_TypesHeader_ContainsStructs(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "anthropic"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.h") {
			typesH = f.Content
			break
		}
	}

	if !strings.Contains(typesH, "typedef struct anthropic_create_request anthropic_create_request_t") {
		t.Errorf("types.h should contain create_request typedef, got:\n%s", typesH)
	}
	if !strings.Contains(typesH, "anthropic_create_request_builder_t *anthropic_create_request_builder_create") {
		t.Errorf("types.h should contain builder create function, got:\n%s", typesH)
	}
	if !strings.Contains(typesH, "anthropic_create_request_builder_set_model") {
		t.Errorf("types.h should contain builder set function, got:\n%s", typesH)
	}
	if !strings.Contains(typesH, "anthropic_create_request_get_model") {
		t.Errorf("types.h should contain getter function, got:\n%s", typesH)
	}
	if !strings.Contains(typesH, "anthropic_create_request_destroy") {
		t.Errorf("types.h should contain destroy function, got:\n%s", typesH)
	}
}

func TestGenerate_ClientHeader_ContainsClientHandle(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "anthropic"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.h") {
			clientH = f.Content
			break
		}
	}

	if !strings.Contains(clientH, "typedef struct anthropic_client anthropic_client_t") {
		t.Errorf("client.h should contain client typedef, got:\n%s", clientH)
	}
	if !strings.Contains(clientH, "typedef struct") {
		t.Errorf("client.h should contain config struct, got:\n%s", clientH)
	}
	if !strings.Contains(clientH, "anthropic_client_create") {
		t.Errorf("client.h should contain client_create function, got:\n%s", clientH)
	}
	if !strings.Contains(clientH, "anthropic_client_destroy") {
		t.Errorf("client.h should contain client_destroy function, got:\n%s", clientH)
	}
	if !strings.Contains(clientH, "ANTHROPIC_AUTH_BEARER") {
		t.Errorf("client.h should contain auth mode enum, got:\n%s", clientH)
	}
}

func TestGenerate_ResourcesHeader_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "anthropic"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.h") {
			resourcesH = f.Content
			break
		}
	}

	if !strings.Contains(resourcesH, "anthropic_messages_create") {
		t.Errorf("resources.h should contain messages_create function, got:\n%s", resourcesH)
	}
}

func TestGenerate_ErrorsHeader_ContainsErrorCodes(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "anthropic"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "errors.h") {
			errorsH = f.Content
			break
		}
	}

	if !strings.Contains(errorsH, "ANTHROPIC_OK = 0") {
		t.Errorf("errors.h should contain OK error code, got:\n%s", errorsH)
	}
	if !strings.Contains(errorsH, "ANTHROPIC_ERR_NULL_ARG") {
		t.Errorf("errors.h should contain ERR_NULL_ARG, got:\n%s", errorsH)
	}
	if !strings.Contains(errorsH, "ANTHROPIC_ERR_HTTP") {
		t.Errorf("errors.h should contain ERR_HTTP, got:\n%s", errorsH)
	}
	if !strings.Contains(errorsH, "anthropic_error_info_t") {
		t.Errorf("errors.h should contain error_info_t, got:\n%s", errorsH)
	}
	if !strings.Contains(errorsH, "anthropic_error_string") {
		t.Errorf("errors.h should contain error_string function, got:\n%s", errorsH)
	}
}

func TestGenerate_DefaultPackage(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkc.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.h") {
			clientH = f.Content
			break
		}
	}

	// Default package should be snake_case sanitized name
	if !strings.Contains(clientH, "my_api_v2_client_t") {
		t.Errorf("expected my_api_v2 package name, got:\n%s", clientH)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "anthropic"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.h") {
			resourcesH = f.Content
			break
		}
	}

	// Should have streaming method with callback
	if !strings.Contains(resourcesH, "_callback_t") {
		t.Errorf("resources.h should contain callback typedef, got:\n%s", resourcesH)
	}
	if !strings.Contains(resourcesH, "_stream") {
		t.Errorf("resources.h should contain stream function, got:\n%s", resourcesH)
	}
}

func TestGenerate_StreamingHeader_HasSseParser(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "anthropic"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "streaming.h") {
			streamingH = f.Content
			break
		}
	}

	if !strings.Contains(streamingH, "anthropic_sse_parser_t") {
		t.Errorf("streaming.h should contain sse_parser typedef, got:\n%s", streamingH)
	}
	if !strings.Contains(streamingH, "anthropic_sse_parser_create") {
		t.Errorf("streaming.h should contain sse_parser_create, got:\n%s", streamingH)
	}
	if !strings.Contains(streamingH, "anthropic_sse_parser_feed") {
		t.Errorf("streaming.h should contain sse_parser_feed, got:\n%s", streamingH)
	}
	if !strings.Contains(streamingH, "anthropic_sse_event_t") {
		t.Errorf("streaming.h should contain sse_event_t, got:\n%s", streamingH)
	}
}

func TestGenerate_OptionalFields(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do", Input: "Request"}}},
		},
		Types: []*contract.Type{
			{
				Name: "Request",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "required_field", Type: "string"},
					{Name: "optional_field", Type: "string", Optional: true},
					{Name: "nullable_field", Type: "int", Nullable: true},
				},
			},
		},
	}

	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.h") {
			typesH = f.Content
			break
		}
	}

	// Should have builder set functions for all fields
	if !strings.Contains(typesH, "builder_set_required_field") {
		t.Errorf("types.h should contain set_required_field, got:\n%s", typesH)
	}
	if !strings.Contains(typesH, "builder_set_optional_field") {
		t.Errorf("types.h should contain set_optional_field, got:\n%s", typesH)
	}
}

func TestGenerate_UnionTypes(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do", Input: "ContentBlock"}}},
		},
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

	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.h") {
			typesH = f.Content
			break
		}
	}

	if !strings.Contains(typesH, "test_content_block_kind_t") {
		t.Errorf("types.h should contain union kind enum, got:\n%s", typesH)
	}
	if !strings.Contains(typesH, "TEST_CONTENT_BLOCK_TEXT_BLOCK") {
		t.Errorf("types.h should contain TEXT_BLOCK variant, got:\n%s", typesH)
	}
	if !strings.Contains(typesH, "test_content_block_as_text_block") {
		t.Errorf("types.h should contain as_text_block function, got:\n%s", typesH)
	}
	if !strings.Contains(typesH, "test_content_block_from_text_block") {
		t.Errorf("types.h should contain from_text_block function, got:\n%s", typesH)
	}
}

func TestGenerate_TypeMapping(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do", Input: "Request"}}},
		},
		Types: []*contract.Type{
			{
				Name: "Request",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "str", Type: "string"},
					{Name: "num", Type: "int"},
					{Name: "bignum", Type: "int64"},
					{Name: "flag", Type: "bool"},
					{Name: "ratio", Type: "float64"},
					{Name: "small", Type: "float32"},
					{Name: "time", Type: "time.Time"},
					{Name: "data", Type: "json.RawMessage"},
				},
			},
		},
	}

	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.h") {
			typesH = f.Content
			break
		}
	}

	// Check that types are properly converted
	expectations := []string{
		"const char *",  // string
		"int32_t",       // int
		"int64_t",       // int64
		"bool",          // bool
		"double",        // float64
		"float",         // float32
	}

	for _, exp := range expectations {
		if !strings.Contains(typesH, exp) {
			t.Errorf("types.h should contain %q, got:\n%s", exp, typesH)
		}
	}
}

func TestGenerate_HTTPMethods(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{
				Name: "items",
				Methods: []*contract.Method{
					{Name: "list", Output: "ListResponse", HTTP: &contract.MethodHTTP{Method: "GET", Path: "/items"}},
					{Name: "create", Input: "CreateRequest", Output: "Item", HTTP: &contract.MethodHTTP{Method: "POST", Path: "/items"}},
					{Name: "update", Input: "UpdateRequest", Output: "Item", HTTP: &contract.MethodHTTP{Method: "PUT", Path: "/items/{id}"}},
					{Name: "delete", HTTP: &contract.MethodHTTP{Method: "DELETE", Path: "/items/{id}"}},
				},
			},
		},
		Types: []*contract.Type{
			{Name: "ListResponse", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "items", Type: "string"}}},
			{Name: "CreateRequest", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "name", Type: "string"}}},
			{Name: "UpdateRequest", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "name", Type: "string"}}},
			{Name: "Item", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}, {Name: "name", Type: "string"}}},
		},
	}

	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.h") {
			resourcesH = f.Content
			break
		}
	}

	// Check that different methods are generated
	if !strings.Contains(resourcesH, "test_items_list") {
		t.Errorf("resources.h should contain items_list function, got:\n%s", resourcesH)
	}
	if !strings.Contains(resourcesH, "test_items_create") {
		t.Errorf("resources.h should contain items_create function, got:\n%s", resourcesH)
	}
	if !strings.Contains(resourcesH, "test_items_update") {
		t.Errorf("resources.h should contain items_update function, got:\n%s", resourcesH)
	}
	if !strings.Contains(resourcesH, "test_items_delete") {
		t.Errorf("resources.h should contain items_delete function, got:\n%s", resourcesH)
	}
}

func TestGenerate_AuthModes(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Client: &contract.Client{
			Auth:    "bearer",
			BaseURL: "https://api.example.com",
		},
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.h") {
			clientH = f.Content
			break
		}
	}

	if !strings.Contains(clientH, "TEST_AUTH_NONE") {
		t.Errorf("client.h should contain AUTH_NONE, got:\n%s", clientH)
	}
	if !strings.Contains(clientH, "TEST_AUTH_BEARER") {
		t.Errorf("client.h should contain AUTH_BEARER, got:\n%s", clientH)
	}
	if !strings.Contains(clientH, "TEST_AUTH_BASIC") {
		t.Errorf("client.h should contain AUTH_BASIC, got:\n%s", clientH)
	}
}

func TestGenerate_DefaultHeaders(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Client: &contract.Client{
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
				"User-Agent":      "TestSDK/1.0",
			},
		},
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkc.Generate(svc, &sdkc.Config{Package: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientC string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.c") {
			clientC = f.Content
			break
		}
	}

	if !strings.Contains(clientC, "X-Custom-Header") {
		t.Errorf("client.c should contain X-Custom-Header, got:\n%s", clientC)
	}
	if !strings.Contains(clientC, "custom-value") {
		t.Errorf("client.c should contain custom-value, got:\n%s", clientC)
	}
}

func TestGenerate_HeaderGuards(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkc.Generate(svc, &sdkc.Config{
		Package:           "anthropic",
		HeaderGuardPrefix: "ANTHROPIC_SDK",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientH string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.h") {
			clientH = f.Content
			break
		}
	}

	if !strings.Contains(clientH, "#ifndef ANTHROPIC_SDK_CLIENT_H") {
		t.Errorf("client.h should use custom header guard prefix, got:\n%s", clientH)
	}
}

// Helper functions

func minimalServiceContract(t *testing.T) *contract.Service {
	t.Helper()
	return &contract.Service{
		Name: "Anthropic",
		Client: &contract.Client{
			Auth:    "bearer",
			BaseURL: "https://api.anthropic.com",
			Headers: map[string]string{
				"anthropic-version": "2024-01-01",
			},
		},
		Resources: []*contract.Resource{
			{
				Name: "messages",
				Methods: []*contract.Method{
					{
						Name:   "create",
						Input:  "CreateRequest",
						Output: "Message",
						HTTP:   &contract.MethodHTTP{Method: "POST", Path: "/v1/messages"},
					},
					{
						Name:  "stream",
						Input: "CreateRequest",
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
				Name: "CreateRequest",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "model", Type: "string"},
					{Name: "messages", Type: "json.RawMessage"},
					{Name: "max_tokens", Type: "int"},
					{Name: "stream", Type: "bool", Optional: true},
				},
			},
			{
				Name: "Message",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
					{Name: "type", Type: "string"},
					{Name: "role", Type: "string"},
					{Name: "content", Type: "json.RawMessage"},
				},
			},
			{
				Name: "MessageStreamEvent",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string"},
					{Name: "delta", Type: "json.RawMessage", Optional: true},
				},
			},
		},
	}
}

func writeGeneratedCSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkc.Config{
		Package: "anthropic",
		Version: "0.0.0",
	}
	files, err := sdkc.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "c-sdk")
	for _, f := range files {
		if f == nil {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(f.Path))
		mustWriteFile(t, p, []byte(f.Content))
	}

	return root
}

func mustWriteFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

var _ = sdk.File{}
