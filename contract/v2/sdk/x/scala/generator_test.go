package sdkscala_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkscala "github.com/go-mizu/mizu/contract/v2/sdk/x/scala"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkscala.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.openai", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"build.sbt":                                          false,
		"project/build.properties":                           false,
		"src/main/scala/com/openai/Client.scala":             false,
		"src/main/scala/com/openai/Types.scala":              false,
		"src/main/scala/com/openai/Resources.scala":          false,
		"src/main/scala/com/openai/Streaming.scala":          false,
		"src/main/scala/com/openai/Errors.scala":             false,
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

func TestGenerate_BuildSbt_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkscala.Generate(svc, &sdkscala.Config{
		Package:      "com.example.sdk",
		Organization: "com.example",
		Version:      "1.2.3",
		ArtifactId:   "example-sdk",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var buildSbt string
	for _, f := range files {
		if f.Path == "build.sbt" {
			buildSbt = f.Content
			break
		}
	}

	if !strings.Contains(buildSbt, `organization := "com.example"`) {
		t.Errorf("build.sbt should contain organization, got:\n%s", buildSbt)
	}
	if !strings.Contains(buildSbt, `version      := "1.2.3"`) {
		t.Errorf("build.sbt should contain version, got:\n%s", buildSbt)
	}
	if !strings.Contains(buildSbt, "circe-core") {
		t.Errorf("build.sbt should contain circe dependency, got:\n%s", buildSbt)
	}
	if !strings.Contains(buildSbt, "sttp.client3") {
		t.Errorf("build.sbt should contain sttp dependency, got:\n%s", buildSbt)
	}
}

func TestGenerate_TypesFile_ContainsCaseClasses(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.scala") {
			typesScala = f.Content
			break
		}
	}

	if !strings.Contains(typesScala, "final case class CreateRequest(") {
		t.Errorf("Types.scala should contain CreateRequest case class, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "final case class Response(") {
		t.Errorf("Types.scala should contain Response case class, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "implicit val encoder: Encoder[") {
		t.Errorf("Types.scala should contain encoder, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "implicit val decoder: Decoder[") {
		t.Errorf("Types.scala should contain decoder, got:\n%s", typesScala)
	}
}

func TestGenerate_ClientFile_ContainsClientClass(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.scala") {
			clientScala = f.Content
			break
		}
	}

	if !strings.Contains(clientScala, "final class OpenAI(") {
		t.Errorf("Client.scala should contain OpenAI class, got:\n%s", clientScala)
	}
	if !strings.Contains(clientScala, "lazy val responses: ResponsesResource") {
		t.Errorf("Client.scala should contain responses resource, got:\n%s", clientScala)
	}
	if !strings.Contains(clientScala, "final case class ClientConfig(") {
		t.Errorf("Client.scala should contain ClientConfig case class, got:\n%s", clientScala)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.scala") {
			resourcesScala = f.Content
			break
		}
	}

	if !strings.Contains(resourcesScala, "final class ResponsesResource") {
		t.Errorf("Resources.scala should contain ResponsesResource class, got:\n%s", resourcesScala)
	}
	if !strings.Contains(resourcesScala, "def create(request: CreateRequest): Future[SDKResult[Response]]") {
		t.Errorf("Resources.scala should contain create method, got:\n%s", resourcesScala)
	}
}

func TestGenerate_DefaultPackageName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkscala.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.scala") {
			clientScala = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then lowercase
	// Default package should be com.example.{sanitized_name}
	if !strings.Contains(clientScala, "package com.example.myapiv2") {
		t.Errorf("expected package com.example.myapiv2, got:\n%s", clientScala)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.scala") {
			resourcesScala = f.Content
			break
		}
	}

	// Should have streaming method returning Stream
	if !strings.Contains(resourcesScala, "def stream(request: CreateRequest): Stream[Future, SDKResult[ResponseEvent]]") {
		t.Errorf("Resources.scala should contain stream method returning Stream, got:\n%s", resourcesScala)
	}
}

func TestGenerate_ErrorsFile_ContainsSDKError(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Errors.scala") {
			errorsScala = f.Content
			break
		}
	}

	if !strings.Contains(errorsScala, "sealed trait SDKError") {
		t.Errorf("Errors.scala should contain SDKError sealed trait, got:\n%s", errorsScala)
	}
	if !strings.Contains(errorsScala, "final case class ApiError(") {
		t.Errorf("Errors.scala should contain ApiError case class, got:\n%s", errorsScala)
	}
	if !strings.Contains(errorsScala, "final case class ConnectionError(") {
		t.Errorf("Errors.scala should contain ConnectionError case class, got:\n%s", errorsScala)
	}
}

func TestGenerate_StreamingFile_HasSSESupport(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Streaming.scala") {
			streamingScala = f.Content
			break
		}
	}

	if !strings.Contains(streamingScala, "parseSSE") {
		t.Errorf("Streaming.scala should contain parseSSE function, got:\n%s", streamingScala)
	}
	if !strings.Contains(streamingScala, "SSEEvent") {
		t.Errorf("Streaming.scala should contain SSEEvent case class, got:\n%s", streamingScala)
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

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.scala") {
			typesScala = f.Content
			break
		}
	}

	if !strings.Contains(typesScala, "requiredField: String") {
		t.Errorf("Types.scala should contain required field as String, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "optionalField: Option[String] = None") {
		t.Errorf("Types.scala should contain optional field as Option[String] = None, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "nullableField: Option[Int] = None") {
		t.Errorf("Types.scala should contain nullable field as Option[Int] = None, got:\n%s", typesScala)
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

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.scala") {
			typesScala = f.Content
			break
		}
	}

	if !strings.Contains(typesScala, "sealed trait Role") {
		t.Errorf("Types.scala should contain Role sealed trait, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, `case object User extends Role`) {
		t.Errorf("Types.scala should contain User case object, got:\n%s", typesScala)
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

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.scala") {
			typesScala = f.Content
			break
		}
	}

	if !strings.Contains(typesScala, "sealed trait ContentBlock") {
		t.Errorf("Types.scala should contain ContentBlock sealed trait, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "final case class TextBlock(") {
		t.Errorf("Types.scala should contain TextBlock case class, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "final case class ImageBlock(") {
		t.Errorf("Types.scala should contain ImageBlock case class, got:\n%s", typesScala)
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

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.scala") {
			typesScala = f.Content
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
		"time: java.time.Instant",
		"data: io.circe.Json",
		"items: List[String]",
		"mapping: Map[String, Int]",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesScala, exp) {
			t.Errorf("Types.scala should contain %q, got:\n%s", exp, typesScala)
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

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.scala") {
			resourcesScala = f.Content
			break
		}
	}

	// Check that different HTTP methods result in different method signatures
	if !strings.Contains(resourcesScala, "def list(): Future[SDKResult[ListResponse]]") {
		t.Errorf("Resources.scala should contain list method with no input, got:\n%s", resourcesScala)
	}
	if !strings.Contains(resourcesScala, "def create(request: CreateRequest): Future[SDKResult[Item]]") {
		t.Errorf("Resources.scala should contain create method with input, got:\n%s", resourcesScala)
	}
	if !strings.Contains(resourcesScala, "Method.GET") {
		t.Errorf("Resources.scala should contain Method.GET, got:\n%s", resourcesScala)
	}
	if !strings.Contains(resourcesScala, "Method.DELETE") {
		t.Errorf("Resources.scala should contain Method.DELETE, got:\n%s", resourcesScala)
	}
}

func TestGenerate_AuthModes(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Defaults: &contract.Defaults{
			Auth:    "bearer",
			BaseURL: "https://api.example.com",
		},
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.scala") {
			clientScala = f.Content
			break
		}
	}

	if !strings.Contains(clientScala, "sealed trait AuthMode") {
		t.Errorf("Client.scala should contain AuthMode sealed trait, got:\n%s", clientScala)
	}
	if !strings.Contains(clientScala, "case object Bearer") {
		t.Errorf("Client.scala should contain Bearer auth mode, got:\n%s", clientScala)
	}
	if !strings.Contains(clientScala, "case object Basic") {
		t.Errorf("Client.scala should contain Basic auth mode, got:\n%s", clientScala)
	}
}

func TestGenerate_DefaultHeaders(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Defaults: &contract.Defaults{
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
				"User-Agent":      "TestSDK/1.0",
			},
		},
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.scala") {
			clientScala = f.Content
			break
		}
	}

	if !strings.Contains(clientScala, `"X-Custom-Header"`) {
		t.Errorf("Client.scala should contain X-Custom-Header, got:\n%s", clientScala)
	}
	if !strings.Contains(clientScala, `"custom-value"`) {
		t.Errorf("Client.scala should contain custom-value, got:\n%s", clientScala)
	}
}

func TestGenerate_NoSSE_StreamingFileMinimal(t *testing.T) {
	svc := &contract.Service{
		Name: "TestAPI",
		Resources: []*contract.Resource{
			{Name: "test", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Streaming.scala") {
			streamingScala = f.Content
			break
		}
	}

	// When no SSE, the streaming file should have minimal content
	if strings.Contains(streamingScala, "final case class SSEEvent") {
		t.Errorf("Streaming.scala should NOT contain SSEEvent case class when no SSE, got:\n%s", streamingScala)
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

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.scala") {
			typesScala = f.Content
			break
		}
	}

	if !strings.Contains(typesScala, "type StringList = List[String]") {
		t.Errorf("Types.scala should contain StringList type alias, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "type IntMap = Map[String, Int]") {
		t.Errorf("Types.scala should contain IntMap type alias, got:\n%s", typesScala)
	}
}

func TestGenerate_ScalaReservedWords(t *testing.T) {
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
					{Name: "type", Type: "string"},
					{Name: "class", Type: "string"},
					{Name: "object", Type: "string"},
				},
			},
		},
	}

	files, err := sdkscala.Generate(svc, &sdkscala.Config{Package: "com.test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesScala string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.scala") {
			typesScala = f.Content
			break
		}
	}

	// Reserved words should be escaped with backticks
	if !strings.Contains(typesScala, "`type`: String") {
		t.Errorf("Types.scala should escape 'type' with backticks, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "`class`: String") {
		t.Errorf("Types.scala should escape 'class' with backticks, got:\n%s", typesScala)
	}
	if !strings.Contains(typesScala, "`object`: String") {
		t.Errorf("Types.scala should escape 'object' with backticks, got:\n%s", typesScala)
	}
}

// Helper functions

func minimalServiceContract(t *testing.T) *contract.Service {
	t.Helper()
	return &contract.Service{
		Name: "OpenAI",
		Defaults: &contract.Defaults{
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

func writeGeneratedScalaSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkscala.Config{
		Package: "com.openai",
		Version: "0.0.0",
	}
	files, err := sdkscala.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "scala-sdk")
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
