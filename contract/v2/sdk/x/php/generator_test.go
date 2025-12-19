package sdkphp_test

import (
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkphp "github.com/go-mizu/mizu/contract/v2/sdk/x/php"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkphp.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "OpenAI", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expectedPrefixes := []string{
		"composer.json",
		"src/Client.php",
		"src/ClientOptions.php",
		"src/AuthMode.php",
		"src/Exceptions/SDKException.php",
		"src/Types/",
		"src/Resources/",
	}

	for _, prefix := range expectedPrefixes {
		found := false
		for _, f := range files {
			if strings.HasPrefix(f.Path, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected file with prefix %s not found in output", prefix)
		}
	}
}

func TestGenerate_ComposerJSON_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkphp.Generate(svc, &sdkphp.Config{
		Namespace:   "Acme\\SDK",
		PackageName: "acme/sdk",
		Version:     "1.2.3",
		License:     "MIT",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var composerJSON string
	for _, f := range files {
		if f.Path == "composer.json" {
			composerJSON = f.Content
			break
		}
	}

	if !strings.Contains(composerJSON, `"name": "acme/sdk"`) {
		t.Errorf("composer.json should contain name, got:\n%s", composerJSON)
	}
	if !strings.Contains(composerJSON, `"guzzlehttp/guzzle"`) {
		t.Errorf("composer.json should contain guzzle dependency, got:\n%s", composerJSON)
	}
	if !strings.Contains(composerJSON, `"php": "^8.1"`) {
		t.Errorf("composer.json should require PHP 8.1+, got:\n%s", composerJSON)
	}
}

func TestGenerate_ClientFile_ContainsClientClass(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.php") && !strings.Contains(f.Path, "Options") {
			clientPHP = f.Content
			break
		}
	}

	if !strings.Contains(clientPHP, "final class Client") {
		t.Errorf("Client.php should contain Client class, got:\n%s", clientPHP)
	}
	if !strings.Contains(clientPHP, "namespace OpenAI;") {
		t.Errorf("Client.php should have correct namespace, got:\n%s", clientPHP)
	}
	if !strings.Contains(clientPHP, "public function __construct(") {
		t.Errorf("Client.php should contain constructor, got:\n%s", clientPHP)
	}
	if !strings.Contains(clientPHP, "public function request(") {
		t.Errorf("Client.php should contain request method, got:\n%s", clientPHP)
	}
}

func TestGenerate_TypesFile_ContainsReadonlyClasses(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var createRequestPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "CreateRequest.php") {
			createRequestPHP = f.Content
			break
		}
	}

	if !strings.Contains(createRequestPHP, "final readonly class CreateRequest") {
		t.Errorf("Type file should contain readonly class, got:\n%s", createRequestPHP)
	}
	if !strings.Contains(createRequestPHP, "public static function fromArray(array $data): self") {
		t.Errorf("Type file should contain fromArray method, got:\n%s", createRequestPHP)
	}
	if !strings.Contains(createRequestPHP, "public function toArray(): array") {
		t.Errorf("Type file should contain toArray method, got:\n%s", createRequestPHP)
	}
	if !strings.Contains(createRequestPHP, "implements \\JsonSerializable") {
		t.Errorf("Type file should implement JsonSerializable, got:\n%s", createRequestPHP)
	}
}

func TestGenerate_ResourceFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcePHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "ResponsesResource.php") {
			resourcePHP = f.Content
			break
		}
	}

	if !strings.Contains(resourcePHP, "final readonly class ResponsesResource") {
		t.Errorf("Resource file should contain ResponsesResource class, got:\n%s", resourcePHP)
	}
	if !strings.Contains(resourcePHP, "public function create(") {
		t.Errorf("Resource file should contain create method, got:\n%s", resourcePHP)
	}
}

func TestGenerate_DefaultNamespace(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkphp.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.php") && !strings.Contains(f.Path, "Options") {
			clientPHP = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then PascalCase
	if !strings.Contains(clientPHP, "namespace MyAPIv2;") {
		t.Errorf("expected namespace MyAPIv2, got:\n%s", clientPHP)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcePHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "ResponsesResource.php") {
			resourcePHP = f.Content
			break
		}
	}

	// Should have streaming method returning Generator
	if !strings.Contains(resourcePHP, "public function stream(") {
		t.Errorf("Resource file should contain stream method, got:\n%s", resourcePHP)
	}
	if !strings.Contains(resourcePHP, "\\Generator") {
		t.Errorf("Resource file should return Generator for streaming, got:\n%s", resourcePHP)
	}
}

func TestGenerate_ExceptionsFile_ContainsExceptions(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var exceptionsPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "SDKException.php") {
			exceptionsPHP = f.Content
			break
		}
	}

	if !strings.Contains(exceptionsPHP, "class SDKException extends \\Exception") {
		t.Errorf("Exceptions file should contain SDKException class, got:\n%s", exceptionsPHP)
	}
	if !strings.Contains(exceptionsPHP, "class ApiException extends SDKException") {
		t.Errorf("Exceptions file should contain ApiException class, got:\n%s", exceptionsPHP)
	}
	if !strings.Contains(exceptionsPHP, "class RateLimitException extends ApiException") {
		t.Errorf("Exceptions file should contain RateLimitException class, got:\n%s", exceptionsPHP)
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

	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "Test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Request.php") {
			typesPHP = f.Content
			break
		}
	}

	if !strings.Contains(typesPHP, "public string $requiredField") {
		t.Errorf("Type file should contain required field as string, got:\n%s", typesPHP)
	}
	if !strings.Contains(typesPHP, "public ?string $optionalField = null") {
		t.Errorf("Type file should contain optional field as ?string = null, got:\n%s", typesPHP)
	}
	if !strings.Contains(typesPHP, "public ?int $nullableField = null") {
		t.Errorf("Type file should contain nullable field as ?int = null, got:\n%s", typesPHP)
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

	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "Test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var unionPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "ContentBlock.php") {
			unionPHP = f.Content
			break
		}
	}

	if !strings.Contains(unionPHP, "abstract readonly class ContentBlock") {
		t.Errorf("Union type file should contain abstract class, got:\n%s", unionPHP)
	}
	if !strings.Contains(unionPHP, "public static function fromArray(array $data): self") {
		t.Errorf("Union type file should contain fromArray factory, got:\n%s", unionPHP)
	}
	if !strings.Contains(unionPHP, "'text' => TextBlock::fromArray($data)") {
		t.Errorf("Union type file should have TextBlock variant, got:\n%s", unionPHP)
	}
	if !strings.Contains(unionPHP, "'image' => ImageBlock::fromArray($data)") {
		t.Errorf("Union type file should have ImageBlock variant, got:\n%s", unionPHP)
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
					{Name: "items", Type: "[]string"},
					{Name: "mapping", Type: "map[string]int"},
				},
			},
		},
	}

	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "Test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Request.php") {
			typesPHP = f.Content
			break
		}
	}

	expectations := []string{
		"public string $str",
		"public int $num",
		"public int $bignum",
		"public bool $flag",
		"public float $ratio",
		"public array $items",
		"public array $mapping",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesPHP, exp) {
			t.Errorf("Type file should contain %q, got:\n%s", exp, typesPHP)
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

	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "Test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "ItemsResource.php") {
			resourcesPHP = f.Content
			break
		}
	}

	if !strings.Contains(resourcesPHP, "public function list(): ListResponse") {
		t.Errorf("Resource file should contain list method with no input, got:\n%s", resourcesPHP)
	}
	if !strings.Contains(resourcesPHP, "public function create(CreateRequest $request): Item") {
		t.Errorf("Resource file should contain create method with input, got:\n%s", resourcesPHP)
	}
	if !strings.Contains(resourcesPHP, "'GET'") {
		t.Errorf("Resource file should contain GET method, got:\n%s", resourcesPHP)
	}
	if !strings.Contains(resourcesPHP, "'DELETE'") {
		t.Errorf("Resource file should contain DELETE method, got:\n%s", resourcesPHP)
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

	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "Test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var authModePHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "AuthMode.php") {
			authModePHP = f.Content
			break
		}
	}

	if !strings.Contains(authModePHP, "enum AuthMode: string") {
		t.Errorf("AuthMode file should contain enum, got:\n%s", authModePHP)
	}
	if !strings.Contains(authModePHP, "case Bearer = 'bearer'") {
		t.Errorf("AuthMode file should contain Bearer case, got:\n%s", authModePHP)
	}
	if !strings.Contains(authModePHP, "case Basic = 'basic'") {
		t.Errorf("AuthMode file should contain Basic case, got:\n%s", authModePHP)
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

	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "Test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var optionsPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "ClientOptions.php") {
			optionsPHP = f.Content
			break
		}
	}

	if !strings.Contains(optionsPHP, "'X-Custom-Header'") {
		t.Errorf("ClientOptions.php should contain X-Custom-Header, got:\n%s", optionsPHP)
	}
	if !strings.Contains(optionsPHP, "'custom-value'") {
		t.Errorf("ClientOptions.php should contain custom-value, got:\n%s", optionsPHP)
	}
}

func TestGenerate_ClientOptionsFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var optionsPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "ClientOptions.php") {
			optionsPHP = f.Content
			break
		}
	}

	if !strings.Contains(optionsPHP, "final readonly class ClientOptions") {
		t.Errorf("ClientOptions.php should contain readonly class, got:\n%s", optionsPHP)
	}
	if !strings.Contains(optionsPHP, "public function withApiKey(?string $apiKey): self") {
		t.Errorf("ClientOptions.php should contain withApiKey method, got:\n%s", optionsPHP)
	}
	if !strings.Contains(optionsPHP, "public function withBaseUrl(string $baseUrl): self") {
		t.Errorf("ClientOptions.php should contain withBaseUrl method, got:\n%s", optionsPHP)
	}
	if !strings.Contains(optionsPHP, "public function withTimeout(float $timeout): self") {
		t.Errorf("ClientOptions.php should contain withTimeout method, got:\n%s", optionsPHP)
	}
}

func TestGenerate_NoSSE_NoStreamMethod(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do", Output: "Response"}}},
		},
		Types: []*contract.Type{
			{Name: "Response", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}}},
		},
	}

	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "Test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.php") && !strings.Contains(f.Path, "Options") {
			clientPHP = f.Content
			break
		}
	}

	// When no SSE, the Client should not contain stream method
	if strings.Contains(clientPHP, "public function stream(") {
		t.Errorf("Client.php should NOT contain stream method when no SSE, got:\n%s", clientPHP)
	}
}

func TestGenerate_PHPDocTypes(t *testing.T) {
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
					{Name: "items", Type: "[]string"},
					{Name: "mapping", Type: "map[string]int"},
				},
			},
		},
	}

	files, err := sdkphp.Generate(svc, &sdkphp.Config{Namespace: "Test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesPHP string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Request.php") {
			typesPHP = f.Content
			break
		}
	}

	// PHPDoc should have array<T> style type hints
	if !strings.Contains(typesPHP, "array<string>") {
		t.Errorf("Type file should contain array<string> PHPDoc type, got:\n%s", typesPHP)
	}
	if !strings.Contains(typesPHP, "array<string, int>") {
		t.Errorf("Type file should contain array<string, int> PHPDoc type, got:\n%s", typesPHP)
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
