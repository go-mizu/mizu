package sdkkotlin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkkotlin "github.com/go-mizu/mizu/contract/v2/sdk/x/kotlin"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkkotlin.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.openai", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"build.gradle.kts":                           false,
		"src/main/kotlin/com/openai/Client.kt":      false,
		"src/main/kotlin/com/openai/Types.kt":       false,
		"src/main/kotlin/com/openai/Resources.kt":   false,
		"src/main/kotlin/com/openai/Streaming.kt":   false,
		"src/main/kotlin/com/openai/Errors.kt":      false,
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

func TestGenerate_BuildGradle_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{
		Package: "com.example.sdk",
		GroupId: "com.example",
		Version: "1.2.3",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var buildGradle string
	for _, f := range files {
		if f.Path == "build.gradle.kts" {
			buildGradle = f.Content
			break
		}
	}

	if !strings.Contains(buildGradle, `group = "com.example"`) {
		t.Errorf("build.gradle.kts should contain group, got:\n%s", buildGradle)
	}
	if !strings.Contains(buildGradle, `version = "1.2.3"`) {
		t.Errorf("build.gradle.kts should contain version, got:\n%s", buildGradle)
	}
	if !strings.Contains(buildGradle, "kotlinx-serialization-json") {
		t.Errorf("build.gradle.kts should contain serialization dependency, got:\n%s", buildGradle)
	}
	if !strings.Contains(buildGradle, "ktor-client-core") {
		t.Errorf("build.gradle.kts should contain ktor dependency, got:\n%s", buildGradle)
	}
}

func TestGenerate_TypesFile_ContainsDataClasses(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.kt") {
			typesKotlin = f.Content
			break
		}
	}

	if !strings.Contains(typesKotlin, "data class CreateRequest(") {
		t.Errorf("Types.kt should contain CreateRequest data class, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "data class Response(") {
		t.Errorf("Types.kt should contain Response data class, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "@Serializable") {
		t.Errorf("Types.kt should contain @Serializable annotation, got:\n%s", typesKotlin)
	}
}

func TestGenerate_ClientFile_ContainsClientClass(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.kt") {
			clientKotlin = f.Content
			break
		}
	}

	if !strings.Contains(clientKotlin, "class OpenAI(") {
		t.Errorf("Client.kt should contain OpenAI class, got:\n%s", clientKotlin)
	}
	if !strings.Contains(clientKotlin, "val responses: ResponsesResource") {
		t.Errorf("Client.kt should contain responses resource, got:\n%s", clientKotlin)
	}
	if !strings.Contains(clientKotlin, "data class ClientOptions(") {
		t.Errorf("Client.kt should contain ClientOptions data class, got:\n%s", clientKotlin)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.kt") {
			resourcesKotlin = f.Content
			break
		}
	}

	if !strings.Contains(resourcesKotlin, "class ResponsesResource") {
		t.Errorf("Resources.kt should contain ResponsesResource class, got:\n%s", resourcesKotlin)
	}
	if !strings.Contains(resourcesKotlin, "suspend fun create(request: CreateRequest): Response") {
		t.Errorf("Resources.kt should contain create method, got:\n%s", resourcesKotlin)
	}
}

func TestGenerate_DefaultPackageName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkkotlin.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.kt") {
			clientKotlin = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then lowercase
	// Default package should be com.example.{sanitized_name}
	if !strings.Contains(clientKotlin, "package com.example.myapiv2") {
		t.Errorf("expected package com.example.myapiv2, got:\n%s", clientKotlin)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.kt") {
			resourcesKotlin = f.Content
			break
		}
	}

	// Should have streaming method returning Flow
	if !strings.Contains(resourcesKotlin, "fun stream(request: CreateRequest): Flow<ResponseEvent>") {
		t.Errorf("Resources.kt should contain stream method returning Flow, got:\n%s", resourcesKotlin)
	}
}

func TestGenerate_ErrorsFile_ContainsSDKException(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Errors.kt") {
			errorsKotlin = f.Content
			break
		}
	}

	if !strings.Contains(errorsKotlin, "sealed class SDKException") {
		t.Errorf("Errors.kt should contain SDKException sealed class, got:\n%s", errorsKotlin)
	}
	if !strings.Contains(errorsKotlin, "data class ApiError(") {
		t.Errorf("Errors.kt should contain ApiError data class, got:\n%s", errorsKotlin)
	}
	if !strings.Contains(errorsKotlin, "data class ConnectionError(") {
		t.Errorf("Errors.kt should contain ConnectionError data class, got:\n%s", errorsKotlin)
	}
}

func TestGenerate_StreamingFile_HasExtensions(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Streaming.kt") {
			streamingKotlin = f.Content
			break
		}
	}

	if !strings.Contains(streamingKotlin, "parseSSEStream") {
		t.Errorf("Streaming.kt should contain parseSSEStream function, got:\n%s", streamingKotlin)
	}
	if !strings.Contains(streamingKotlin, "fun <T> Flow<T>.toList()") {
		t.Errorf("Streaming.kt should contain toList extension, got:\n%s", streamingKotlin)
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.kt") {
			typesKotlin = f.Content
			break
		}
	}

	if !strings.Contains(typesKotlin, "val requiredField: String") {
		t.Errorf("Types.kt should contain required field as String, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "val optionalField: String? = null") {
		t.Errorf("Types.kt should contain optional field as String? = null, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "val nullableField: Int? = null") {
		t.Errorf("Types.kt should contain nullable field as Int? = null, got:\n%s", typesKotlin)
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.kt") {
			typesKotlin = f.Content
			break
		}
	}

	if !strings.Contains(typesKotlin, "enum class RequestRole") {
		t.Errorf("Types.kt should contain RequestRole enum, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, `@SerialName("user")`) {
		t.Errorf("Types.kt should contain @SerialName annotation, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "USER") {
		t.Errorf("Types.kt should contain USER case, got:\n%s", typesKotlin)
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.kt") {
			typesKotlin = f.Content
			break
		}
	}

	if !strings.Contains(typesKotlin, "sealed class ContentBlock") {
		t.Errorf("Types.kt should contain ContentBlock sealed class, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "data class TextBlock(") {
		t.Errorf("Types.kt should contain TextBlock data class, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "data class ImageBlock(") {
		t.Errorf("Types.kt should contain ImageBlock data class, got:\n%s", typesKotlin)
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.kt") {
			typesKotlin = f.Content
			break
		}
	}

	expectations := []string{
		"str: String",
		"num: Int",
		"bignum: Long",
		"flag: Boolean",
		"ratio: Double",
		"small: Float",
		"time: Instant",
		"data: JsonElement",
		"items: List<String>",
		"mapping: Map<String, Int>",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesKotlin, exp) {
			t.Errorf("Types.kt should contain %q, got:\n%s", exp, typesKotlin)
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.kt") {
			resourcesKotlin = f.Content
			break
		}
	}

	// Check that different HTTP methods result in different method signatures
	if !strings.Contains(resourcesKotlin, "suspend fun list(): ListResponse") {
		t.Errorf("Resources.kt should contain list method with no input, got:\n%s", resourcesKotlin)
	}
	if !strings.Contains(resourcesKotlin, "suspend fun create(request: CreateRequest): Item") {
		t.Errorf("Resources.kt should contain create method with input, got:\n%s", resourcesKotlin)
	}
	if !strings.Contains(resourcesKotlin, "HttpMethod.Get") {
		t.Errorf("Resources.kt should contain HttpMethod.Get, got:\n%s", resourcesKotlin)
	}
	if !strings.Contains(resourcesKotlin, "HttpMethod.Delete") {
		t.Errorf("Resources.kt should contain HttpMethod.Delete, got:\n%s", resourcesKotlin)
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.kt") {
			clientKotlin = f.Content
			break
		}
	}

	if !strings.Contains(clientKotlin, "enum class AuthMode") {
		t.Errorf("Client.kt should contain AuthMode enum, got:\n%s", clientKotlin)
	}
	if !strings.Contains(clientKotlin, "BEARER") {
		t.Errorf("Client.kt should contain BEARER auth mode, got:\n%s", clientKotlin)
	}
	if !strings.Contains(clientKotlin, "BASIC") {
		t.Errorf("Client.kt should contain BASIC auth mode, got:\n%s", clientKotlin)
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.kt") {
			clientKotlin = f.Content
			break
		}
	}

	if !strings.Contains(clientKotlin, `"X-Custom-Header"`) {
		t.Errorf("Client.kt should contain X-Custom-Header, got:\n%s", clientKotlin)
	}
	if !strings.Contains(clientKotlin, `"custom-value"`) {
		t.Errorf("Client.kt should contain custom-value, got:\n%s", clientKotlin)
	}
}

func TestGenerate_NoSSE_StreamingFileMinimal(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Streaming.kt") {
			streamingKotlin = f.Content
			break
		}
	}

	// When no SSE, the streaming file should have minimal content
	if strings.Contains(streamingKotlin, "parseSSEStream") {
		t.Errorf("Streaming.kt should NOT contain parseSSEStream when no SSE, got:\n%s", streamingKotlin)
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.kt") {
			typesKotlin = f.Content
			break
		}
	}

	if !strings.Contains(typesKotlin, "typealias StringList = List<String>") {
		t.Errorf("Types.kt should contain StringList typealias, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "typealias IntMap = Map<String, Int>") {
		t.Errorf("Types.kt should contain IntMap typealias, got:\n%s", typesKotlin)
	}
}

func TestGenerate_SerialNameAnnotations(t *testing.T) {
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

	files, err := sdkkotlin.Generate(svc, &sdkkotlin.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesKotlin string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.kt") {
			typesKotlin = f.Content
			break
		}
	}

	// Should use @SerialName for JSON field names
	if !strings.Contains(typesKotlin, `@SerialName("snake_case_field")`) {
		t.Errorf("Types.kt should contain @SerialName for snake_case_field, got:\n%s", typesKotlin)
	}
	if !strings.Contains(typesKotlin, "val snakeCaseField: String") {
		t.Errorf("Types.kt should use camelCase for Kotlin field name, got:\n%s", typesKotlin)
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

func writeGeneratedKotlinSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkkotlin.Config{
		Package: "com.openai",
		Version: "0.0.0",
	}
	files, err := sdkkotlin.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "kotlin-sdk")
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
