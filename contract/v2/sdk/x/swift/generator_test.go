package sdkswift_test

import (
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkswift "github.com/go-mizu/mizu/contract/v2/sdk/x/swift"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkswift.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "OpenAI", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"Package.swift":                false,
		"Sources/OpenAI/Client.swift":    false,
		"Sources/OpenAI/Types.swift":     false,
		"Sources/OpenAI/Resources.swift": false,
		"Sources/OpenAI/Streaming.swift": false,
		"Sources/OpenAI/Errors.swift":    false,
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

func TestGenerate_PackageSwift_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "MySDK", Version: "1.2.3"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var pkgSwift string
	for _, f := range files {
		if f.Path == "Package.swift" {
			pkgSwift = f.Content
			break
		}
	}

	if !strings.Contains(pkgSwift, `name: "MySDK"`) {
		t.Errorf("Package.swift should contain package name, got:\n%s", pkgSwift)
	}
	if !strings.Contains(pkgSwift, `targets: ["MySDK"]`) {
		t.Errorf("Package.swift should contain target name, got:\n%s", pkgSwift)
	}
}

func TestGenerate_TypesFile_ContainsStructs(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesSwift string
	for _, f := range files {
		if f.Path == "Sources/OpenAI/Types.swift" {
			typesSwift = f.Content
			break
		}
	}

	if !strings.Contains(typesSwift, "public struct CreateRequest:") {
		t.Errorf("Types.swift should contain CreateRequest struct, got:\n%s", typesSwift)
	}
	if !strings.Contains(typesSwift, "public struct Response:") {
		t.Errorf("Types.swift should contain Response struct, got:\n%s", typesSwift)
	}
	if !strings.Contains(typesSwift, "Codable") {
		t.Errorf("Types.swift structs should conform to Codable, got:\n%s", typesSwift)
	}
	if !strings.Contains(typesSwift, "Sendable") {
		t.Errorf("Types.swift structs should conform to Sendable, got:\n%s", typesSwift)
	}
}

func TestGenerate_ClientFile_ContainsClientClass(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientSwift string
	for _, f := range files {
		if f.Path == "Sources/OpenAI/Client.swift" {
			clientSwift = f.Content
			break
		}
	}

	if !strings.Contains(clientSwift, "public final class OpenAI:") {
		t.Errorf("Client.swift should contain OpenAI class, got:\n%s", clientSwift)
	}
	if !strings.Contains(clientSwift, "public let responses:") {
		t.Errorf("Client.swift should contain responses resource, got:\n%s", clientSwift)
	}
	if !strings.Contains(clientSwift, "Sendable") {
		t.Errorf("Client.swift should conform to Sendable, got:\n%s", clientSwift)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesSwift string
	for _, f := range files {
		if f.Path == "Sources/OpenAI/Resources.swift" {
			resourcesSwift = f.Content
			break
		}
	}

	if !strings.Contains(resourcesSwift, "public struct ResponsesResource:") {
		t.Errorf("Resources.swift should contain ResponsesResource struct, got:\n%s", resourcesSwift)
	}
	if !strings.Contains(resourcesSwift, "func create(request: CreateRequest) async throws -> Response") {
		t.Errorf("Resources.swift should contain create method, got:\n%s", resourcesSwift)
	}
}

func TestGenerate_DefaultPackageName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkswift.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var pkgSwift string
	for _, f := range files {
		if f.Path == "Package.swift" {
			pkgSwift = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then PascalCase
	// "My API!! v2" -> "MyAPIv2"
	if !strings.Contains(pkgSwift, `name: "MyAPIv2"`) {
		t.Errorf("expected package name MyAPIv2, got:\n%s", pkgSwift)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesSwift string
	for _, f := range files {
		if f.Path == "Sources/OpenAI/Resources.swift" {
			resourcesSwift = f.Content
			break
		}
	}

	// Should have streaming method returning AsyncThrowingStream
	if !strings.Contains(resourcesSwift, "func stream(request: CreateRequest) -> AsyncThrowingStream<ResponseEvent, Error>") {
		t.Errorf("Resources.swift should contain stream method, got:\n%s", resourcesSwift)
	}
}

func TestGenerate_ErrorsFile_ContainsSDKError(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsSwift string
	for _, f := range files {
		if f.Path == "Sources/OpenAI/Errors.swift" {
			errorsSwift = f.Content
			break
		}
	}

	if !strings.Contains(errorsSwift, "public enum SDKError: Error") {
		t.Errorf("Errors.swift should contain SDKError enum, got:\n%s", errorsSwift)
	}
	if !strings.Contains(errorsSwift, "public struct APIError: Error") {
		t.Errorf("Errors.swift should contain APIError struct, got:\n%s", errorsSwift)
	}
}

func TestGenerate_StreamingFile_HasExtensions(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingSwift string
	for _, f := range files {
		if f.Path == "Sources/OpenAI/Streaming.swift" {
			streamingSwift = f.Content
			break
		}
	}

	if !strings.Contains(streamingSwift, "extension AsyncThrowingStream") {
		t.Errorf("Streaming.swift should contain AsyncThrowingStream extension, got:\n%s", streamingSwift)
	}
	if !strings.Contains(streamingSwift, "func collect()") {
		t.Errorf("Streaming.swift should contain collect method, got:\n%s", streamingSwift)
	}
}

func TestGenerate_PlatformVersions(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkswift.Generate(svc, &sdkswift.Config{
		Package: "OpenAI",
		Platforms: sdkswift.Platforms{
			IOS:     "16.0",
			MacOS:   "13.0",
			WatchOS: "9.0",
			TvOS:    "16.0",
		},
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var pkgSwift string
	for _, f := range files {
		if f.Path == "Package.swift" {
			pkgSwift = f.Content
			break
		}
	}

	if !strings.Contains(pkgSwift, ".iOS(.v16.0)") {
		t.Errorf("Package.swift should contain iOS v16.0, got:\n%s", pkgSwift)
	}
	if !strings.Contains(pkgSwift, ".macOS(.v13.0)") {
		t.Errorf("Package.swift should contain macOS v13.0, got:\n%s", pkgSwift)
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

	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesSwift string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.swift") {
			typesSwift = f.Content
			break
		}
	}

	if !strings.Contains(typesSwift, "requiredField: String") {
		t.Errorf("Types.swift should contain required field as String, got:\n%s", typesSwift)
	}
	if !strings.Contains(typesSwift, "optionalField: String?") {
		t.Errorf("Types.swift should contain optional field as String?, got:\n%s", typesSwift)
	}
	if !strings.Contains(typesSwift, "nullableField: Int?") {
		t.Errorf("Types.swift should contain nullable field as Int?, got:\n%s", typesSwift)
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

	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesSwift string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.swift") {
			typesSwift = f.Content
			break
		}
	}

	if !strings.Contains(typesSwift, "public enum RequestRole:") {
		t.Errorf("Types.swift should contain RequestRole enum, got:\n%s", typesSwift)
	}
	if !strings.Contains(typesSwift, `case user = "user"`) {
		t.Errorf("Types.swift should contain user case, got:\n%s", typesSwift)
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

	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesSwift string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.swift") {
			typesSwift = f.Content
			break
		}
	}

	if !strings.Contains(typesSwift, "public enum ContentBlock:") {
		t.Errorf("Types.swift should contain ContentBlock enum, got:\n%s", typesSwift)
	}
	if !strings.Contains(typesSwift, "case textBlock(TextBlock)") {
		t.Errorf("Types.swift should contain textBlock case, got:\n%s", typesSwift)
	}
	if !strings.Contains(typesSwift, "case imageBlock(ImageBlock)") {
		t.Errorf("Types.swift should contain imageBlock case, got:\n%s", typesSwift)
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
					{Name: "items", Type: "[]string"},
					{Name: "mapping", Type: "map[string]int"},
				},
			},
		},
	}

	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesSwift string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.swift") {
			typesSwift = f.Content
			break
		}
	}

	expectations := []string{
		"str: String",
		"num: Int",
		"bignum: Int64",
		"flag: Bool",
		"ratio: Double",
		"small: Float",
		"time: Date",
		"data: AnyCodable",
		"items: [String]",
		"mapping: [String: Int]",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesSwift, exp) {
			t.Errorf("Types.swift should contain %q, got:\n%s", exp, typesSwift)
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

	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesSwift string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.swift") {
			resourcesSwift = f.Content
			break
		}
	}

	// Check that different HTTP methods result in different method signatures
	if !strings.Contains(resourcesSwift, "func list() async throws -> ListResponse") {
		t.Errorf("Resources.swift should contain list method with no input, got:\n%s", resourcesSwift)
	}
	if !strings.Contains(resourcesSwift, "func create(request: CreateRequest) async throws -> Item") {
		t.Errorf("Resources.swift should contain create method with input, got:\n%s", resourcesSwift)
	}
	if !strings.Contains(resourcesSwift, `method: "GET"`) {
		t.Errorf("Resources.swift should contain GET method, got:\n%s", resourcesSwift)
	}
	if !strings.Contains(resourcesSwift, `method: "DELETE"`) {
		t.Errorf("Resources.swift should contain DELETE method, got:\n%s", resourcesSwift)
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

	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientSwift string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.swift") {
			clientSwift = f.Content
			break
		}
	}

	if !strings.Contains(clientSwift, "AuthMode") {
		t.Errorf("Client.swift should contain AuthMode, got:\n%s", clientSwift)
	}
	if !strings.Contains(clientSwift, "case bearer") {
		t.Errorf("Client.swift should contain bearer auth mode, got:\n%s", clientSwift)
	}
	if !strings.Contains(clientSwift, "case basic") {
		t.Errorf("Client.swift should contain basic auth mode, got:\n%s", clientSwift)
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

	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientSwift string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.swift") {
			clientSwift = f.Content
			break
		}
	}

	if !strings.Contains(clientSwift, `"X-Custom-Header"`) {
		t.Errorf("Client.swift should contain X-Custom-Header, got:\n%s", clientSwift)
	}
	if !strings.Contains(clientSwift, `"custom-value"`) {
		t.Errorf("Client.swift should contain custom-value, got:\n%s", clientSwift)
	}
}

func TestGenerate_NoSSE_StreamingFileMinimal(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkswift.Generate(svc, &sdkswift.Config{Package: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingSwift string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Streaming.swift") {
			streamingSwift = f.Content
			break
		}
	}

	// When no SSE, the streaming file should have minimal content
	if strings.Contains(streamingSwift, "extension AsyncThrowingStream") {
		t.Errorf("Streaming.swift should NOT contain AsyncThrowingStream extension when no SSE, got:\n%s", streamingSwift)
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
