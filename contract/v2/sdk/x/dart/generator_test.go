package sdkdart_test

import (
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkdart "github.com/go-mizu/mizu/contract/v2/sdk/x/dart"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkdart.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "openai_sdk", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"pubspec.yaml":             false,
		"lib/openai_sdk.dart":      false,
		"lib/src/client.dart":      false,
		"lib/src/types.dart":       false,
		"lib/src/resources.dart":   false,
		"lib/src/streaming.dart":   false,
		"lib/src/errors.dart":      false,
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

func TestGenerate_Pubspec_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{
		Package:     "example_sdk",
		Version:     "1.2.3",
		Description: "Example SDK for testing",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var pubspec string
	for _, f := range files {
		if f.Path == "pubspec.yaml" {
			pubspec = f.Content
			break
		}
	}

	if !strings.Contains(pubspec, "name: example_sdk") {
		t.Errorf("pubspec.yaml should contain name, got:\n%s", pubspec)
	}
	if !strings.Contains(pubspec, "version: 1.2.3") {
		t.Errorf("pubspec.yaml should contain version, got:\n%s", pubspec)
	}
	if !strings.Contains(pubspec, "http: ^1.1.0") {
		t.Errorf("pubspec.yaml should contain http dependency, got:\n%s", pubspec)
	}
	if !strings.Contains(pubspec, "sdk: '>=3.0.0") {
		t.Errorf("pubspec.yaml should contain sdk version constraint, got:\n%s", pubspec)
	}
}

func TestGenerate_TypesFile_ContainsClasses(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "openai_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.dart") {
			typesDart = f.Content
			break
		}
	}

	if !strings.Contains(typesDart, "class CreateRequest {") {
		t.Errorf("types.dart should contain CreateRequest class, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "class Response {") {
		t.Errorf("types.dart should contain Response class, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "factory CreateRequest.fromJson(Map<String, dynamic> json)") {
		t.Errorf("types.dart should contain fromJson factory, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "Map<String, dynamic> toJson()") {
		t.Errorf("types.dart should contain toJson method, got:\n%s", typesDart)
	}
}

func TestGenerate_ClientFile_ContainsClientClass(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "openai_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.dart") {
			clientDart = f.Content
			break
		}
	}

	if !strings.Contains(clientDart, "class OpenAI {") {
		t.Errorf("client.dart should contain OpenAI class, got:\n%s", clientDart)
	}
	if !strings.Contains(clientDart, "late final ResponsesResource responses") {
		t.Errorf("client.dart should contain responses resource, got:\n%s", clientDart)
	}
	if !strings.Contains(clientDart, "class ClientOptions {") {
		t.Errorf("client.dart should contain ClientOptions class, got:\n%s", clientDart)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "openai_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.dart") {
			resourcesDart = f.Content
			break
		}
	}

	if !strings.Contains(resourcesDart, "class ResponsesResource") {
		t.Errorf("resources.dart should contain ResponsesResource class, got:\n%s", resourcesDart)
	}
	if !strings.Contains(resourcesDart, "Future<Response> create(CreateRequest request)") {
		t.Errorf("resources.dart should contain create method, got:\n%s", resourcesDart)
	}
}

func TestGenerate_DefaultPackageName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkdart.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var pubspec string
	for _, f := range files {
		if f.Path == "pubspec.yaml" {
			pubspec = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then snake_case
	// Default package should be snake_case sanitized name
	if !strings.Contains(pubspec, "name: my_apiv2") {
		t.Errorf("expected package name my_apiv2, got:\n%s", pubspec)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "openai_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.dart") {
			resourcesDart = f.Content
			break
		}
	}

	// Should have streaming method returning Stream
	if !strings.Contains(resourcesDart, "Stream<ResponseEvent> stream(CreateRequest request)") {
		t.Errorf("resources.dart should contain stream method returning Stream, got:\n%s", resourcesDart)
	}
}

func TestGenerate_ErrorsFile_ContainsSDKException(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "openai_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "errors.dart") {
			errorsDart = f.Content
			break
		}
	}

	if !strings.Contains(errorsDart, "sealed class SDKException") {
		t.Errorf("errors.dart should contain SDKException sealed class, got:\n%s", errorsDart)
	}
	if !strings.Contains(errorsDart, "final class ApiException extends SDKException") {
		t.Errorf("errors.dart should contain ApiException class, got:\n%s", errorsDart)
	}
	if !strings.Contains(errorsDart, "final class ConnectionException extends SDKException") {
		t.Errorf("errors.dart should contain ConnectionException class, got:\n%s", errorsDart)
	}
}

func TestGenerate_StreamingFile_HasParseSSE(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "openai_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "streaming.dart") {
			streamingDart = f.Content
			break
		}
	}

	if !strings.Contains(streamingDart, "parseSSEStream") {
		t.Errorf("streaming.dart should contain parseSSEStream function, got:\n%s", streamingDart)
	}
	if !strings.Contains(streamingDart, "Stream<T> parseSSEStream<T>") {
		t.Errorf("streaming.dart should contain generic parseSSEStream, got:\n%s", streamingDart)
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.dart") {
			typesDart = f.Content
			break
		}
	}

	if !strings.Contains(typesDart, "final String requiredField") {
		t.Errorf("types.dart should contain required field as String, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "final String? optionalField") {
		t.Errorf("types.dart should contain optional field as String?, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "final int? nullableField") {
		t.Errorf("types.dart should contain nullable field as int?, got:\n%s", typesDart)
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.dart") {
			typesDart = f.Content
			break
		}
	}

	if !strings.Contains(typesDart, "enum RequestRole") {
		t.Errorf("types.dart should contain RequestRole enum, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "user('user')") {
		t.Errorf("types.dart should contain user enum case, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "factory RequestRole.fromJson(String json)") {
		t.Errorf("types.dart should contain fromJson factory, got:\n%s", typesDart)
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.dart") {
			typesDart = f.Content
			break
		}
	}

	if !strings.Contains(typesDart, "sealed class ContentBlock") {
		t.Errorf("types.dart should contain ContentBlock sealed class, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "final class TextBlock extends ContentBlock") {
		t.Errorf("types.dart should contain TextBlock final class, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "final class ImageBlock extends ContentBlock") {
		t.Errorf("types.dart should contain ImageBlock final class, got:\n%s", typesDart)
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.dart") {
			typesDart = f.Content
			break
		}
	}

	expectations := []string{
		"final String str",
		"final int num",
		"final int bignum",
		"final bool flag",
		"final double ratio",
		"final double small",
		"final DateTime time",
		"final Object data",
		"final List<String> items",
		"final Map<String, int> mapping",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesDart, exp) {
			t.Errorf("types.dart should contain %q, got:\n%s", exp, typesDart)
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.dart") {
			resourcesDart = f.Content
			break
		}
	}

	// Check that different HTTP methods result in different method signatures
	if !strings.Contains(resourcesDart, "Future<ListResponse> list()") {
		t.Errorf("resources.dart should contain list method with no input, got:\n%s", resourcesDart)
	}
	if !strings.Contains(resourcesDart, "Future<Item> create(CreateRequest request)") {
		t.Errorf("resources.dart should contain create method with input, got:\n%s", resourcesDart)
	}
	if !strings.Contains(resourcesDart, "'GET'") {
		t.Errorf("resources.dart should contain GET method, got:\n%s", resourcesDart)
	}
	if !strings.Contains(resourcesDart, "'DELETE'") {
		t.Errorf("resources.dart should contain DELETE method, got:\n%s", resourcesDart)
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.dart") {
			clientDart = f.Content
			break
		}
	}

	if !strings.Contains(clientDart, "enum AuthMode") {
		t.Errorf("client.dart should contain AuthMode enum, got:\n%s", clientDart)
	}
	if !strings.Contains(clientDart, "bearer,") {
		t.Errorf("client.dart should contain bearer auth mode, got:\n%s", clientDart)
	}
	if !strings.Contains(clientDart, "basic,") {
		t.Errorf("client.dart should contain basic auth mode, got:\n%s", clientDart)
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.dart") {
			clientDart = f.Content
			break
		}
	}

	if !strings.Contains(clientDart, `'X-Custom-Header'`) {
		t.Errorf("client.dart should contain X-Custom-Header, got:\n%s", clientDart)
	}
	if !strings.Contains(clientDart, `'custom-value'`) {
		t.Errorf("client.dart should contain custom-value, got:\n%s", clientDart)
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.dart") {
			typesDart = f.Content
			break
		}
	}

	if !strings.Contains(typesDart, "typedef StringList = List<String>") {
		t.Errorf("types.dart should contain StringList typedef, got:\n%s", typesDart)
	}
	if !strings.Contains(typesDart, "typedef IntMap = Map<String, int>") {
		t.Errorf("types.dart should contain IntMap typedef, got:\n%s", typesDart)
	}
}

func TestGenerate_CopyWithMethod(t *testing.T) {
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

	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "test_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.dart") {
			typesDart = f.Content
			break
		}
	}

	if !strings.Contains(typesDart, "Request copyWith(") {
		t.Errorf("types.dart should contain copyWith method, got:\n%s", typesDart)
	}
}

func TestGenerate_MinSDKVersion(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{
		Package: "test_sdk",
		MinSDK:  "3.2.0",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var pubspec string
	for _, f := range files {
		if f.Path == "pubspec.yaml" {
			pubspec = f.Content
			break
		}
	}

	if !strings.Contains(pubspec, "sdk: '>=3.2.0") {
		t.Errorf("pubspec.yaml should contain custom SDK version, got:\n%s", pubspec)
	}
}

func TestGenerate_LibraryExports(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkdart.Generate(svc, &sdkdart.Config{Package: "openai_sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var libDart string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "openai_sdk.dart") {
			libDart = f.Content
			break
		}
	}

	if !strings.Contains(libDart, "library openai_sdk") {
		t.Errorf("library file should contain library directive, got:\n%s", libDart)
	}
	if !strings.Contains(libDart, "export 'src/client.dart'") {
		t.Errorf("library file should export client.dart, got:\n%s", libDart)
	}
	if !strings.Contains(libDart, "export 'src/types.dart'") {
		t.Errorf("library file should export types.dart, got:\n%s", libDart)
	}
	if !strings.Contains(libDart, "export 'src/errors.dart'") {
		t.Errorf("library file should export errors.dart, got:\n%s", libDart)
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
