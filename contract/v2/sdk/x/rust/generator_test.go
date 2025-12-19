package sdkrust_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkrust "github.com/go-mizu/mizu/contract/v2/sdk/x/rust"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkrust.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "openai-sdk", Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"Cargo.toml":        false,
		"src/lib.rs":        false,
		"src/client.rs":     false,
		"src/types.rs":      false,
		"src/resources.rs":  false,
		"src/streaming.rs":  false,
		"src/error.rs":      false,
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

func TestGenerate_CargoToml_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{
		Crate:   "my-sdk",
		Version: "1.2.3",
		Authors: []string{"Author Name <author@example.com>"},
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var cargoToml string
	for _, f := range files {
		if f.Path == "Cargo.toml" {
			cargoToml = f.Content
			break
		}
	}

	if !strings.Contains(cargoToml, `name = "my-sdk"`) {
		t.Errorf("Cargo.toml should contain name, got:\n%s", cargoToml)
	}
	if !strings.Contains(cargoToml, `version = "1.2.3"`) {
		t.Errorf("Cargo.toml should contain version, got:\n%s", cargoToml)
	}
	if !strings.Contains(cargoToml, "reqwest") {
		t.Errorf("Cargo.toml should contain reqwest dependency, got:\n%s", cargoToml)
	}
	if !strings.Contains(cargoToml, "serde") {
		t.Errorf("Cargo.toml should contain serde dependency, got:\n%s", cargoToml)
	}
	if !strings.Contains(cargoToml, "thiserror") {
		t.Errorf("Cargo.toml should contain thiserror dependency, got:\n%s", cargoToml)
	}
}

func TestGenerate_TypesFile_ContainsStructs(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "openai-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rs") {
			typesRust = f.Content
			break
		}
	}

	if !strings.Contains(typesRust, "pub struct CreateRequest") {
		t.Errorf("types.rs should contain CreateRequest struct, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "pub struct Response") {
		t.Errorf("types.rs should contain Response struct, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]") {
		t.Errorf("types.rs should contain derive macros, got:\n%s", typesRust)
	}
}

func TestGenerate_ClientFile_ContainsClient(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "openai-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.rs") {
			clientRust = f.Content
			break
		}
	}

	if !strings.Contains(clientRust, "pub struct Client") {
		t.Errorf("client.rs should contain Client struct, got:\n%s", clientRust)
	}
	if !strings.Contains(clientRust, "pub fn responses(&self)") {
		t.Errorf("client.rs should contain responses method, got:\n%s", clientRust)
	}
	if !strings.Contains(clientRust, "pub struct ClientBuilder") {
		t.Errorf("client.rs should contain ClientBuilder struct, got:\n%s", clientRust)
	}
	if !strings.Contains(clientRust, "pub enum AuthMode") {
		t.Errorf("client.rs should contain AuthMode enum, got:\n%s", clientRust)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "openai-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.rs") {
			resourcesRust = f.Content
			break
		}
	}

	if !strings.Contains(resourcesRust, "pub struct Responses") {
		t.Errorf("resources.rs should contain Responses struct, got:\n%s", resourcesRust)
	}
	if !strings.Contains(resourcesRust, "pub async fn create(") {
		t.Errorf("resources.rs should contain create method, got:\n%s", resourcesRust)
	}
}

func TestGenerate_DefaultCrateName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkrust.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var cargoToml string
	for _, f := range files {
		if f.Path == "Cargo.toml" {
			cargoToml = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then to kebab-case
	// "My API!! v2" -> "MyAPIv2" -> "my-ap-iv2" (camelCase handling)
	if !strings.Contains(cargoToml, `name = "my-ap-iv2"`) {
		t.Errorf("expected name my-ap-iv2, got:\n%s", cargoToml)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "openai-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.rs") {
			resourcesRust = f.Content
			break
		}
	}

	// Should have streaming method returning Stream
	if !strings.Contains(resourcesRust, "pub async fn stream(") {
		t.Errorf("resources.rs should contain stream method, got:\n%s", resourcesRust)
	}
	if !strings.Contains(resourcesRust, "dyn Stream<Item = Result<ResponseEvent>>") {
		t.Errorf("resources.rs should return Stream<Item = Result<ResponseEvent>>, got:\n%s", resourcesRust)
	}
}

func TestGenerate_ErrorFile_ContainsErrorEnum(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "openai-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "error.rs") {
			errorRust = f.Content
			break
		}
	}

	if !strings.Contains(errorRust, "pub enum Error") {
		t.Errorf("error.rs should contain Error enum, got:\n%s", errorRust)
	}
	if !strings.Contains(errorRust, "Http {") {
		t.Errorf("error.rs should contain Http variant, got:\n%s", errorRust)
	}
	if !strings.Contains(errorRust, "Connection(") {
		t.Errorf("error.rs should contain Connection variant, got:\n%s", errorRust)
	}
	if !strings.Contains(errorRust, "pub fn is_retriable(") {
		t.Errorf("error.rs should contain is_retriable method, got:\n%s", errorRust)
	}
}

func TestGenerate_StreamingFile_HasSSEParser(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "openai-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "streaming.rs") {
			streamingRust = f.Content
			break
		}
	}

	if !strings.Contains(streamingRust, "struct SseParser") {
		t.Errorf("streaming.rs should contain SseParser struct, got:\n%s", streamingRust)
	}
	if !strings.Contains(streamingRust, "pub struct EventStream") {
		t.Errorf("streaming.rs should contain EventStream struct, got:\n%s", streamingRust)
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

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rs") {
			typesRust = f.Content
			break
		}
	}

	if !strings.Contains(typesRust, "pub required_field: String") {
		t.Errorf("types.rs should contain required field as String, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "pub optional_field: Option<String>") {
		t.Errorf("types.rs should contain optional field as Option<String>, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "pub nullable_field: Option<i32>") {
		t.Errorf("types.rs should contain nullable field as Option<i32>, got:\n%s", typesRust)
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

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rs") {
			typesRust = f.Content
			break
		}
	}

	if !strings.Contains(typesRust, "pub enum RequestRole") {
		t.Errorf("types.rs should contain RequestRole enum, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, `#[serde(rename = "user")]`) {
		t.Errorf("types.rs should contain #[serde(rename)] annotation, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "User,") {
		t.Errorf("types.rs should contain User variant, got:\n%s", typesRust)
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

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rs") {
			typesRust = f.Content
			break
		}
	}

	if !strings.Contains(typesRust, "pub enum ContentBlock") {
		t.Errorf("types.rs should contain ContentBlock enum, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, `#[serde(tag = "type")]`) {
		t.Errorf("types.rs should contain serde tag, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "Text(TextBlock)") {
		t.Errorf("types.rs should contain Text variant, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "Image(ImageBlock)") {
		t.Errorf("types.rs should contain Image variant, got:\n%s", typesRust)
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

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rs") {
			typesRust = f.Content
			break
		}
	}

	expectations := []string{
		"str: String",
		"num: i32",
		"bignum: i64",
		"flag: bool",
		"ratio: f64",
		"small: f32",
		"data: serde_json::Value",
		"items: Vec<String>",
		"mapping: std::collections::HashMap<String, i32>",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesRust, exp) {
			t.Errorf("types.rs should contain %q, got:\n%s", exp, typesRust)
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

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.rs") {
			resourcesRust = f.Content
			break
		}
	}

	// Check that different HTTP methods are used
	if !strings.Contains(resourcesRust, ".get(") {
		t.Errorf("resources.rs should contain .get(, got:\n%s", resourcesRust)
	}
	if !strings.Contains(resourcesRust, ".post(") {
		t.Errorf("resources.rs should contain .post(, got:\n%s", resourcesRust)
	}
	if !strings.Contains(resourcesRust, ".put(") {
		t.Errorf("resources.rs should contain .put(, got:\n%s", resourcesRust)
	}
	if !strings.Contains(resourcesRust, ".delete(") {
		t.Errorf("resources.rs should contain .delete(, got:\n%s", resourcesRust)
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

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.rs") {
			clientRust = f.Content
			break
		}
	}

	if !strings.Contains(clientRust, "pub enum AuthMode") {
		t.Errorf("client.rs should contain AuthMode enum, got:\n%s", clientRust)
	}
	if !strings.Contains(clientRust, "Bearer") {
		t.Errorf("client.rs should contain Bearer auth mode, got:\n%s", clientRust)
	}
	if !strings.Contains(clientRust, "Basic") {
		t.Errorf("client.rs should contain Basic auth mode, got:\n%s", clientRust)
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

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.rs") {
			clientRust = f.Content
			break
		}
	}

	if !strings.Contains(clientRust, `"x-custom-header"`) {
		t.Errorf("client.rs should contain x-custom-header, got:\n%s", clientRust)
	}
	if !strings.Contains(clientRust, `"custom-value"`) {
		t.Errorf("client.rs should contain custom-value, got:\n%s", clientRust)
	}
}

func TestGenerate_SliceAndMapTypes(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do", Input: "Request"}}},
		},
		Types: []*contract.Type{
			{
				Name: "StringList",
				Kind: contract.KindSlice,
				Elem: "string",
			},
			{
				Name: "IntMap",
				Kind: contract.KindMap,
				Elem: "int",
			},
			{
				Name: "Request",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "items", Type: "StringList"},
					{Name: "counts", Type: "IntMap"},
				},
			},
		},
	}

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rs") {
			typesRust = f.Content
			break
		}
	}

	if !strings.Contains(typesRust, "pub type StringList = Vec<String>") {
		t.Errorf("types.rs should contain StringList type alias, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "pub type IntMap = std::collections::HashMap<String, i32>") {
		t.Errorf("types.rs should contain IntMap type alias, got:\n%s", typesRust)
	}
}

func TestGenerate_SerdeRenameAnnotations(t *testing.T) {
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
					{Name: "snake_case_field", Type: "string"},
					{Name: "camelCaseField", Type: "int"},
				},
			},
		},
	}

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rs") {
			typesRust = f.Content
			break
		}
	}

	// Should use #[serde(rename)] for JSON field names
	if !strings.Contains(typesRust, `#[serde(rename = "snake_case_field")]`) {
		t.Errorf("types.rs should contain #[serde(rename)] for snake_case_field, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "pub snake_case_field: String") {
		t.Errorf("types.rs should use snake_case for Rust field name, got:\n%s", typesRust)
	}
}

func TestGenerate_BuilderPattern(t *testing.T) {
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
					{Name: "name", Type: "string"},
					{Name: "value", Type: "int"},
				},
			},
		},
	}

	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "test-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rs") {
			typesRust = f.Content
			break
		}
	}

	if !strings.Contains(typesRust, "pub struct RequestBuilder") {
		t.Errorf("types.rs should contain RequestBuilder struct, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "pub fn builder() -> RequestBuilder") {
		t.Errorf("types.rs should contain builder() method, got:\n%s", typesRust)
	}
	if !strings.Contains(typesRust, "pub fn build(self) -> Request") {
		t.Errorf("types.rs should contain build() method, got:\n%s", typesRust)
	}
}

func TestGenerate_LibRsExports(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkrust.Generate(svc, &sdkrust.Config{Crate: "openai-sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var libRust string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "lib.rs") {
			libRust = f.Content
			break
		}
	}

	if !strings.Contains(libRust, "pub use client::") {
		t.Errorf("lib.rs should re-export client, got:\n%s", libRust)
	}
	if !strings.Contains(libRust, "pub use error::") {
		t.Errorf("lib.rs should re-export error, got:\n%s", libRust)
	}
	if !strings.Contains(libRust, "pub use types::*") {
		t.Errorf("lib.rs should re-export types, got:\n%s", libRust)
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

func writeGeneratedRustSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkrust.Config{
		Crate:   "openai-sdk",
		Version: "0.1.0",
	}
	files, err := sdkrust.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "rust-sdk")
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
