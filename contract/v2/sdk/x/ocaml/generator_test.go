package sdkocaml_test

import (
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkocaml "github.com/go-mizu/mizu/contract/v2/sdk/x/ocaml"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkocaml.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"dune-project":          false,
		"openai.opam":           false,
		"lib/dune":              false,
		"lib/openai.ml":         false,
		"lib/openai.mli":        false,
		"lib/config.ml":         false,
		"lib/config.mli":        false,
		"lib/client.ml":         false,
		"lib/client.mli":        false,
		"lib/types.ml":          false,
		"lib/types.mli":         false,
		"lib/resources.ml":      false,
		"lib/resources.mli":     false,
		"lib/streaming.ml":      false,
		"lib/streaming.mli":     false,
		"lib/errors.ml":         false,
		"lib/errors.mli":        false,
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

func TestGenerate_DuneProject_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{
		PackageName: "example_sdk",
		ModuleName:  "ExampleSDK",
		Version:     "1.2.3",
		Synopsis:    "An example SDK",
		Author:      "Test Author",
		License:     "MIT",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var duneProject string
	for _, f := range files {
		if f.Path == "dune-project" {
			duneProject = f.Content
			break
		}
	}

	if !strings.Contains(duneProject, "(name example_sdk)") {
		t.Errorf("dune-project should contain package name, got:\n%s", duneProject)
	}
	if !strings.Contains(duneProject, "(version 1.2.3)") {
		t.Errorf("dune-project should contain version, got:\n%s", duneProject)
	}
	if !strings.Contains(duneProject, "yojson") {
		t.Errorf("dune-project should contain yojson dependency, got:\n%s", duneProject)
	}
	if !strings.Contains(duneProject, "cohttp-lwt-unix") {
		t.Errorf("dune-project should contain cohttp-lwt-unix dependency, got:\n%s", duneProject)
	}
	if !strings.Contains(duneProject, "lwt") {
		t.Errorf("dune-project should contain lwt dependency, got:\n%s", duneProject)
	}
}

func TestGenerate_TypesFile_ContainsTypes(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesMl string
	for _, f := range files {
		if f.Path == "lib/types.ml" {
			typesMl = f.Content
			break
		}
	}

	if !strings.Contains(typesMl, "type create_request") {
		t.Errorf("types.ml should contain create_request type, got:\n%s", typesMl)
	}
	if !strings.Contains(typesMl, "type response") {
		t.Errorf("types.ml should contain response type, got:\n%s", typesMl)
	}
	if !strings.Contains(typesMl, "create_request_of_yojson") {
		t.Errorf("types.ml should contain of_yojson function, got:\n%s", typesMl)
	}
	if !strings.Contains(typesMl, "yojson_of_create_request") {
		t.Errorf("types.ml should contain yojson_of function, got:\n%s", typesMl)
	}
}

func TestGenerate_ClientFile_ContainsClient(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientMl string
	for _, f := range files {
		if f.Path == "lib/client.ml" {
			clientMl = f.Content
			break
		}
	}

	if !strings.Contains(clientMl, "type t") {
		t.Errorf("client.ml should contain client type, got:\n%s", clientMl)
	}
	if !strings.Contains(clientMl, "let create") {
		t.Errorf("client.ml should contain create function, got:\n%s", clientMl)
	}
	if !strings.Contains(clientMl, "let create_with") {
		t.Errorf("client.ml should contain create_with function, got:\n%s", clientMl)
	}
	if !strings.Contains(clientMl, "let request") {
		t.Errorf("client.ml should contain request function, got:\n%s", clientMl)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesMl string
	for _, f := range files {
		if f.Path == "lib/resources.ml" {
			resourcesMl = f.Content
			break
		}
	}

	if !strings.Contains(resourcesMl, "module Responses") {
		t.Errorf("resources.ml should contain Responses module, got:\n%s", resourcesMl)
	}
	if !strings.Contains(resourcesMl, "let create") {
		t.Errorf("resources.ml should contain create method, got:\n%s", resourcesMl)
	}
	if !strings.Contains(resourcesMl, "let create_exn") {
		t.Errorf("resources.ml should contain create_exn method, got:\n%s", resourcesMl)
	}
}

func TestGenerate_DefaultPackageName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkocaml.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Should generate snake_case package name
	var opamFound bool
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".opam") {
			opamFound = true
			break
		}
	}

	if !opamFound {
		var paths []string
		for _, f := range files {
			paths = append(paths, f.Path)
		}
		t.Errorf("expected .opam file, got files: %v", paths)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesMl string
	for _, f := range files {
		if f.Path == "lib/resources.ml" {
			resourcesMl = f.Content
			break
		}
	}

	// Should have streaming method
	if !strings.Contains(resourcesMl, "stream") {
		t.Errorf("resources.ml should contain streaming method, got:\n%s", resourcesMl)
	}
}

func TestGenerate_ErrorsFile_ContainsTypes(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsMl string
	for _, f := range files {
		if f.Path == "lib/errors.ml" {
			errorsMl = f.Content
			break
		}
	}

	if !strings.Contains(errorsMl, "type t") {
		t.Errorf("errors.ml should contain error type, got:\n%s", errorsMl)
	}
	if !strings.Contains(errorsMl, "Api_error") {
		t.Errorf("errors.ml should contain Api_error, got:\n%s", errorsMl)
	}
	if !strings.Contains(errorsMl, "Rate_limit_error") {
		t.Errorf("errors.ml should contain Rate_limit_error, got:\n%s", errorsMl)
	}
	if !strings.Contains(errorsMl, "is_retryable") {
		t.Errorf("errors.ml should contain is_retryable function, got:\n%s", errorsMl)
	}
}

func TestGenerate_StreamingFile_HasSSEParsing(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingMl string
	for _, f := range files {
		if f.Path == "lib/streaming.ml" {
			streamingMl = f.Content
			break
		}
	}

	if !strings.Contains(streamingMl, "type 'a event") {
		t.Errorf("streaming.ml should contain event type, got:\n%s", streamingMl)
	}
	if !strings.Contains(streamingMl, "Data") {
		t.Errorf("streaming.ml should contain Data constructor, got:\n%s", streamingMl)
	}
	if !strings.Contains(streamingMl, "Done") {
		t.Errorf("streaming.ml should contain Done constructor, got:\n%s", streamingMl)
	}
	if !strings.Contains(streamingMl, `"[DONE]"`) {
		t.Errorf("streaming.ml should handle [DONE] terminator, got:\n%s", streamingMl)
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

	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesMl string
	for _, f := range files {
		if f.Path == "lib/types.ml" {
			typesMl = f.Content
			break
		}
	}

	if !strings.Contains(typesMl, "string") {
		t.Errorf("types.ml should contain string type, got:\n%s", typesMl)
	}
	if !strings.Contains(typesMl, "option") {
		t.Errorf("types.ml should use option for optional/nullable fields, got:\n%s", typesMl)
	}
}

func TestGenerate_EnumFields(t *testing.T) {
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
					{Name: "role", Type: "string", Enum: []string{"user", "assistant", "system"}},
				},
			},
		},
	}

	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesMl string
	for _, f := range files {
		if f.Path == "lib/types.ml" {
			typesMl = f.Content
			break
		}
	}

	// Should generate enum type
	if !strings.Contains(typesMl, "type request_role") {
		t.Errorf("types.ml should contain request_role enum type, got:\n%s", typesMl)
	}
	// Enum values should be PascalCase
	if !strings.Contains(typesMl, "User") {
		t.Errorf("types.ml should contain User enum value, got:\n%s", typesMl)
	}
	if !strings.Contains(typesMl, "Assistant") {
		t.Errorf("types.ml should contain Assistant enum value, got:\n%s", typesMl)
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

	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesMl string
	for _, f := range files {
		if f.Path == "lib/types.ml" {
			typesMl = f.Content
			break
		}
	}

	if !strings.Contains(typesMl, "type content_block") {
		t.Errorf("types.ml should contain content_block union type, got:\n%s", typesMl)
	}
	// Check for variant constructors (polymorphic variants use backtick)
	if !strings.Contains(typesMl, "`TextBlock") {
		t.Errorf("types.ml should contain `TextBlock variant, got:\n%s", typesMl)
	}
	if !strings.Contains(typesMl, "`ImageBlock") {
		t.Errorf("types.ml should contain `ImageBlock variant, got:\n%s", typesMl)
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
					{Name: "data", Type: "json.RawMessage"},
					{Name: "items", Type: "[]string"},
					{Name: "mapping", Type: "map[string]int"},
				},
			},
		},
	}

	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesMl string
	for _, f := range files {
		if f.Path == "lib/types.ml" {
			typesMl = f.Content
			break
		}
	}

	// OCaml types
	expectations := []string{
		"string",         // string
		"int",            // int
		"int64",          // int64
		"bool",           // bool
		"float",          // float64 and float32
		"Yojson.Safe.t",  // json.RawMessage
		"string list",    // []string
	}

	for _, exp := range expectations {
		if !strings.Contains(typesMl, exp) {
			t.Errorf("types.ml should contain %q, got:\n%s", exp, typesMl)
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
			{Name: "ListResponse", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "items", Type: "[]Item"}}},
			{Name: "CreateRequest", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "name", Type: "string"}}},
			{Name: "UpdateRequest", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "name", Type: "string"}}},
			{Name: "Item", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}, {Name: "name", Type: "string"}}},
		},
	}

	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesMl string
	for _, f := range files {
		if f.Path == "lib/resources.ml" {
			resourcesMl = f.Content
			break
		}
	}

	// Check HTTP methods are used
	if !strings.Contains(resourcesMl, "`GET") {
		t.Errorf("resources.ml should contain GET method, got:\n%s", resourcesMl)
	}
	if !strings.Contains(resourcesMl, "`POST") {
		t.Errorf("resources.ml should contain POST method, got:\n%s", resourcesMl)
	}
	if !strings.Contains(resourcesMl, "`PUT") {
		t.Errorf("resources.ml should contain PUT method, got:\n%s", resourcesMl)
	}
	if !strings.Contains(resourcesMl, "`DELETE") {
		t.Errorf("resources.ml should contain DELETE method, got:\n%s", resourcesMl)
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

	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientMl string
	for _, f := range files {
		if f.Path == "lib/client.ml" {
			clientMl = f.Content
			break
		}
	}

	if !strings.Contains(clientMl, "build_auth_headers") {
		t.Errorf("client.ml should contain build_auth_headers function, got:\n%s", clientMl)
	}
	if !strings.Contains(clientMl, "Bearer") {
		t.Errorf("client.ml should handle Bearer auth, got:\n%s", clientMl)
	}
	if !strings.Contains(clientMl, "Basic") {
		t.Errorf("client.ml should handle Basic auth, got:\n%s", clientMl)
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

	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var configMl string
	for _, f := range files {
		if f.Path == "lib/config.ml" {
			configMl = f.Content
			break
		}
	}

	if !strings.Contains(configMl, `"X-Custom-Header"`) {
		t.Errorf("config.ml should contain X-Custom-Header, got:\n%s", configMl)
	}
	if !strings.Contains(configMl, `"custom-value"`) {
		t.Errorf("config.ml should contain custom-value, got:\n%s", configMl)
	}
}

func TestGenerate_SnakeCaseNaming(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "userProfiles", Methods: []*contract.Method{{Name: "getById", Input: "Request"}}},
		},
		Types: []*contract.Type{
			{
				Name: "Request",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "userId", Type: "string"},
					{Name: "profileType", Type: "string"},
				},
			},
		},
	}

	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesMl string
	var typesMl string
	for _, f := range files {
		if f.Path == "lib/resources.ml" {
			resourcesMl = f.Content
		}
		if f.Path == "lib/types.ml" {
			typesMl = f.Content
		}
	}

	// Should use snake_case for OCaml functions
	if !strings.Contains(resourcesMl, "get_by_id") {
		t.Errorf("resources.ml should use snake_case function name, got:\n%s", resourcesMl)
	}

	// Should use snake_case for OCaml record fields
	if !strings.Contains(typesMl, "user_id") || !strings.Contains(typesMl, "profile_type") {
		t.Errorf("types.ml should use snake_case field names, got:\n%s", typesMl)
	}
}

func TestGenerate_MainFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var mainMl string
	for _, f := range files {
		if f.Path == "lib/openai.ml" {
			mainMl = f.Content
			break
		}
	}

	if !strings.Contains(mainMl, "module Config") {
		t.Errorf("main.ml should export Config module, got:\n%s", mainMl)
	}
	if !strings.Contains(mainMl, "module Client") {
		t.Errorf("main.ml should export Client module, got:\n%s", mainMl)
	}
	if !strings.Contains(mainMl, "module Types") {
		t.Errorf("main.ml should export Types module, got:\n%s", mainMl)
	}
	if !strings.Contains(mainMl, "module Resources") {
		t.Errorf("main.ml should export Resources module, got:\n%s", mainMl)
	}
}

func TestGenerate_ConfigFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var configMl string
	for _, f := range files {
		if f.Path == "lib/config.ml" {
			configMl = f.Content
			break
		}
	}

	if !strings.Contains(configMl, "type t") {
		t.Errorf("config.ml should define t type, got:\n%s", configMl)
	}
	if !strings.Contains(configMl, "api_key") {
		t.Errorf("config.ml should have api_key field, got:\n%s", configMl)
	}
	if !strings.Contains(configMl, "base_url") {
		t.Errorf("config.ml should have base_url field, got:\n%s", configMl)
	}
	if !strings.Contains(configMl, "timeout") {
		t.Errorf("config.ml should have timeout field, got:\n%s", configMl)
	}
	if !strings.Contains(configMl, "let default") {
		t.Errorf("config.ml should have default, got:\n%s", configMl)
	}
	if !strings.Contains(configMl, "let from_env") {
		t.Errorf("config.ml should have from_env, got:\n%s", configMl)
	}
}

func TestGenerate_EnvironmentVariables(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkocaml.Generate(svc, &sdkocaml.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var configMl string
	for _, f := range files {
		if f.Path == "lib/config.ml" {
			configMl = f.Content
			break
		}
	}

	// Should have env var names based on service name
	if !strings.Contains(configMl, "OPEN_AI_API_KEY") {
		t.Errorf("config.ml should reference OPEN_AI_API_KEY env var, got:\n%s", configMl)
	}
	if !strings.Contains(configMl, "Sys.getenv_opt") {
		t.Errorf("config.ml should use Sys.getenv_opt, got:\n%s", configMl)
	}
}

// Helper functions

func minimalServiceContract(t *testing.T) *contract.Service {
	t.Helper()
	return &contract.Service{
		Name: "OpenAI",
		Client: &contract.Client{
			Auth:    "bearer",
			BaseURL: "https://api.openai.com",
		},
		Resources: []*contract.Resource{
			{
				Name: "responses",
				Methods: []*contract.Method{
					{
						Name:   "create",
						Input:  "CreateRequest",
						Output: "Response",
						HTTP:   &contract.MethodHTTP{Method: "POST", Path: "/v1/responses"},
					},
					{
						Name:  "stream",
						Input: "CreateRequest",
						Stream: &contract.MethodStream{
							Mode: "sse",
							Item: "ResponseEvent",
						},
						HTTP: &contract.MethodHTTP{Method: "POST", Path: "/v1/responses"},
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
					{Name: "input", Type: "json.RawMessage", Optional: true},
				},
			},
			{
				Name: "Response",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "id", Type: "string"},
				},
			},
			{
				Name: "ResponseEvent",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "type", Type: "string"},
					{Name: "text", Type: "string", Optional: true},
				},
			},
			{
				Name: "Error",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "code", Type: "string", Optional: true},
					{Name: "message", Type: "string"},
				},
			},
		},
	}
}

var _ = sdk.File{}
