package sdkcsharp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkcsharp "github.com/go-mizu/mizu/contract/v2/sdk/x/csharp"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkcsharp.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "OpenAI.Sdk", PackageName: "OpenAI.Sdk", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"OpenAI.Sdk.csproj":              false,
		"src/OpenAIClient.cs":            false,
		"src/Models/Types.cs":            false,
		"src/Resources/Resources.cs":     false,
		"src/Streaming.cs":               false,
		"src/Exceptions.cs":              false,
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

func TestGenerate_Csproj_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{
		Namespace:       "Example.Sdk",
		PackageName:     "Example.Sdk",
		Version:         "1.2.3",
		TargetFramework: "net9.0",
		Nullable:        true,
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var csproj string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".csproj") {
			csproj = f.Content
			break
		}
	}

	if !strings.Contains(csproj, "<PackageId>Example.Sdk</PackageId>") {
		t.Errorf("csproj should contain PackageId, got:\n%s", csproj)
	}
	if !strings.Contains(csproj, "<Version>1.2.3</Version>") {
		t.Errorf("csproj should contain Version, got:\n%s", csproj)
	}
	if !strings.Contains(csproj, "<TargetFramework>net9.0</TargetFramework>") {
		t.Errorf("csproj should contain TargetFramework, got:\n%s", csproj)
	}
	if !strings.Contains(csproj, "<Nullable>enable</Nullable>") {
		t.Errorf("csproj should contain Nullable enable, got:\n%s", csproj)
	}
}

func TestGenerate_TypesFile_ContainsRecords(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.cs") {
			typesCs = f.Content
			break
		}
	}

	if !strings.Contains(typesCs, "public sealed record CreateRequest") {
		t.Errorf("Types.cs should contain CreateRequest record, got:\n%s", typesCs)
	}
	if !strings.Contains(typesCs, "public sealed record Response") {
		t.Errorf("Types.cs should contain Response record, got:\n%s", typesCs)
	}
	if !strings.Contains(typesCs, "[JsonPropertyName(") {
		t.Errorf("Types.cs should contain JsonPropertyName attribute, got:\n%s", typesCs)
	}
	if !strings.Contains(typesCs, "public required") {
		t.Errorf("Types.cs should contain required properties, got:\n%s", typesCs)
	}
}

func TestGenerate_ClientFile_ContainsClientClass(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.cs") {
			clientCs = f.Content
			break
		}
	}

	if !strings.Contains(clientCs, "public sealed class OpenAIClient : IDisposable") {
		t.Errorf("Client.cs should contain OpenAIClient class, got:\n%s", clientCs)
	}
	if !strings.Contains(clientCs, "public ResponsesResource Responses { get; }") {
		t.Errorf("Client.cs should contain Responses resource, got:\n%s", clientCs)
	}
	if !strings.Contains(clientCs, "public sealed record ClientOptions") {
		t.Errorf("Client.cs should contain ClientOptions record, got:\n%s", clientCs)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.cs") {
			resourcesCs = f.Content
			break
		}
	}

	if !strings.Contains(resourcesCs, "public sealed class ResponsesResource") {
		t.Errorf("Resources.cs should contain ResponsesResource class, got:\n%s", resourcesCs)
	}
	if !strings.Contains(resourcesCs, "public async Task<Response> CreateAsync(") {
		t.Errorf("Resources.cs should contain CreateAsync method, got:\n%s", resourcesCs)
	}
}

func TestGenerate_DefaultNamespace(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkcsharp.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.cs") {
			clientCs = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then PascalCase
	// Default namespace should be PascalCase sanitized name + ".Sdk"
	if !strings.Contains(clientCs, "namespace MyAPIv2.Sdk") {
		t.Errorf("expected namespace MyAPIv2.Sdk, got:\n%s", clientCs)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.cs") {
			resourcesCs = f.Content
			break
		}
	}

	// Should have streaming method returning IAsyncEnumerable
	if !strings.Contains(resourcesCs, "public async IAsyncEnumerable<ResponseEvent> StreamAsync(") {
		t.Errorf("Resources.cs should contain StreamAsync method returning IAsyncEnumerable, got:\n%s", resourcesCs)
	}
}

func TestGenerate_ExceptionsFile_ContainsSdkException(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var exceptionsCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Exceptions.cs") {
			exceptionsCs = f.Content
			break
		}
	}

	if !strings.Contains(exceptionsCs, "public abstract class SdkException : Exception") {
		t.Errorf("Exceptions.cs should contain SdkException class, got:\n%s", exceptionsCs)
	}
	if !strings.Contains(exceptionsCs, "public sealed class ApiException : SdkException") {
		t.Errorf("Exceptions.cs should contain ApiException class, got:\n%s", exceptionsCs)
	}
	if !strings.Contains(exceptionsCs, "public sealed class ConnectionException : SdkException") {
		t.Errorf("Exceptions.cs should contain ConnectionException class, got:\n%s", exceptionsCs)
	}
}

func TestGenerate_StreamingFile_HasSseParser(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Streaming.cs") {
			streamingCs = f.Content
			break
		}
	}

	if !strings.Contains(streamingCs, "internal static class SseParser") {
		t.Errorf("Streaming.cs should contain SseParser class, got:\n%s", streamingCs)
	}
	if !strings.Contains(streamingCs, "ParseAsync<T>") {
		t.Errorf("Streaming.cs should contain generic ParseAsync, got:\n%s", streamingCs)
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

	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.cs") {
			typesCs = f.Content
			break
		}
	}

	if !strings.Contains(typesCs, "public required string RequiredField") {
		t.Errorf("Types.cs should contain required field as string, got:\n%s", typesCs)
	}
	if !strings.Contains(typesCs, "public string? OptionalField") {
		t.Errorf("Types.cs should contain optional field as string?, got:\n%s", typesCs)
	}
	if !strings.Contains(typesCs, "public int? NullableField") {
		t.Errorf("Types.cs should contain nullable field as int?, got:\n%s", typesCs)
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

	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.cs") {
			typesCs = f.Content
			break
		}
	}

	if !strings.Contains(typesCs, "[JsonPolymorphic(TypeDiscriminatorPropertyName =") {
		t.Errorf("Types.cs should contain JsonPolymorphic attribute, got:\n%s", typesCs)
	}
	if !strings.Contains(typesCs, "[JsonDerivedType(typeof(TextBlock),") {
		t.Errorf("Types.cs should contain JsonDerivedType for TextBlock, got:\n%s", typesCs)
	}
	if !strings.Contains(typesCs, "public abstract record ContentBlock") {
		t.Errorf("Types.cs should contain ContentBlock abstract record, got:\n%s", typesCs)
	}
	if !strings.Contains(typesCs, "public sealed record TextBlock : ContentBlock") {
		t.Errorf("Types.cs should contain TextBlock record, got:\n%s", typesCs)
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

	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.cs") {
			typesCs = f.Content
			break
		}
	}

	expectations := []string{
		"string Str",
		"int Num",
		"long Bignum",
		"bool Flag",
		"double Ratio",
		"float Small",
		"DateTimeOffset Time",
		"JsonElement Data",
		"IReadOnlyList<string> Items",
		"IReadOnlyDictionary<string, int> Mapping",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesCs, exp) {
			t.Errorf("Types.cs should contain %q, got:\n%s", exp, typesCs)
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

	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.cs") {
			resourcesCs = f.Content
			break
		}
	}

	// Check that different HTTP methods result in different method signatures
	if !strings.Contains(resourcesCs, "public async Task<ListResponse> ListAsync(") {
		t.Errorf("Resources.cs should contain ListAsync method, got:\n%s", resourcesCs)
	}
	if !strings.Contains(resourcesCs, "public async Task<Item> CreateAsync(") {
		t.Errorf("Resources.cs should contain CreateAsync method, got:\n%s", resourcesCs)
	}
	if !strings.Contains(resourcesCs, "HttpMethod.Get") {
		t.Errorf("Resources.cs should contain HttpMethod.Get, got:\n%s", resourcesCs)
	}
	if !strings.Contains(resourcesCs, "HttpMethod.Delete") {
		t.Errorf("Resources.cs should contain HttpMethod.Delete, got:\n%s", resourcesCs)
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

	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.cs") {
			clientCs = f.Content
			break
		}
	}

	if !strings.Contains(clientCs, "public enum AuthMode") {
		t.Errorf("Client.cs should contain AuthMode enum, got:\n%s", clientCs)
	}
	if !strings.Contains(clientCs, "Bearer,") {
		t.Errorf("Client.cs should contain Bearer auth mode, got:\n%s", clientCs)
	}
	if !strings.Contains(clientCs, "Basic,") {
		t.Errorf("Client.cs should contain Basic auth mode, got:\n%s", clientCs)
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

	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientCs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.cs") {
			clientCs = f.Content
			break
		}
	}

	if !strings.Contains(clientCs, `"X-Custom-Header"`) {
		t.Errorf("Client.cs should contain X-Custom-Header, got:\n%s", clientCs)
	}
	if !strings.Contains(clientCs, `"custom-value"`) {
		t.Errorf("Client.cs should contain custom-value, got:\n%s", clientCs)
	}
}

func TestGenerate_NullableDisabled(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkcsharp.Generate(svc, &sdkcsharp.Config{
		Namespace: "Test.Sdk",
		Nullable:  false,
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var csproj string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".csproj") {
			csproj = f.Content
			break
		}
	}

	if !strings.Contains(csproj, "<Nullable>disable</Nullable>") {
		t.Errorf("csproj should contain Nullable disable, got:\n%s", csproj)
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

func writeGeneratedCSharpSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkcsharp.Config{
		Namespace:   "OpenAI.Sdk",
		PackageName: "OpenAI.Sdk",
		Version:     "0.0.0",
	}
	files, err := sdkcsharp.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "csharp-sdk")
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
