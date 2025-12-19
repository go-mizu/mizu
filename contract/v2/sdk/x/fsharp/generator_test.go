package sdkfsharp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkfsharp "github.com/go-mizu/mizu/contract/v2/sdk/x/fsharp"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkfsharp.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "OpenAI.Sdk", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"OpenAI.Sdk.fsproj":  false,
		"src/Types.fs":      false,
		"src/Errors.fs":     false,
		"src/Http.fs":       false,
		"src/Streaming.fs":  false,
		"src/Resources.fs":  false,
		"src/Client.fs":     false,
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

func TestGenerate_FsProj_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{
		Namespace:       "Example.Sdk",
		PackageName:     "Example.Sdk",
		Version:         "1.2.3",
		TargetFramework: "net8.0",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var fsproj string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".fsproj") {
			fsproj = f.Content
			break
		}
	}

	if !strings.Contains(fsproj, "<Version>1.2.3</Version>") {
		t.Errorf("fsproj should contain version, got:\n%s", fsproj)
	}
	if !strings.Contains(fsproj, "<TargetFramework>net8.0</TargetFramework>") {
		t.Errorf("fsproj should contain target framework, got:\n%s", fsproj)
	}
	// F# requires compilation order
	if !strings.Contains(fsproj, `<Compile Include="src/Types.fs" />`) {
		t.Errorf("fsproj should list Types.fs first, got:\n%s", fsproj)
	}
}

func TestGenerate_TypesFile_ContainsRecords(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.fs") {
			typesFSharp = f.Content
			break
		}
	}

	if !strings.Contains(typesFSharp, "type CreateRequest = {") {
		t.Errorf("Types.fs should contain CreateRequest record, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "type Response = {") {
		t.Errorf("Types.fs should contain Response record, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "[<CLIMutable>]") {
		t.Errorf("Types.fs should contain CLIMutable attribute, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "[<JsonPropertyName(") {
		t.Errorf("Types.fs should contain JsonPropertyName attribute, got:\n%s", typesFSharp)
	}
}

func TestGenerate_ClientFile_ContainsClientType(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.fs") {
			clientFSharp = f.Content
			break
		}
	}

	if !strings.Contains(clientFSharp, "type OpenAIClient(options: ClientOptions)") {
		t.Errorf("Client.fs should contain OpenAIClient type, got:\n%s", clientFSharp)
	}
	if !strings.Contains(clientFSharp, "member _.Responses") {
		t.Errorf("Client.fs should contain Responses member, got:\n%s", clientFSharp)
	}
	if !strings.Contains(clientFSharp, "interface IDisposable") {
		t.Errorf("Client.fs should implement IDisposable, got:\n%s", clientFSharp)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.fs") {
			resourcesFSharp = f.Content
			break
		}
	}

	if !strings.Contains(resourcesFSharp, "type ResponsesResource") {
		t.Errorf("Resources.fs should contain ResponsesResource type, got:\n%s", resourcesFSharp)
	}
	if !strings.Contains(resourcesFSharp, "member _.CreateAsync") {
		t.Errorf("Resources.fs should contain CreateAsync method, got:\n%s", resourcesFSharp)
	}
	if !strings.Contains(resourcesFSharp, "Task<Result<Response, SdkError>>") {
		t.Errorf("Resources.fs should return Result type, got:\n%s", resourcesFSharp)
	}
}

func TestGenerate_ErrorsFile_ContainsSdkError(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Errors.fs") {
			errorsFSharp = f.Content
			break
		}
	}

	if !strings.Contains(errorsFSharp, "type SdkError =") {
		t.Errorf("Errors.fs should contain SdkError DU, got:\n%s", errorsFSharp)
	}
	if !strings.Contains(errorsFSharp, "| ConnectionError") {
		t.Errorf("Errors.fs should contain ConnectionError case, got:\n%s", errorsFSharp)
	}
	if !strings.Contains(errorsFSharp, "| ApiError") {
		t.Errorf("Errors.fs should contain ApiError case, got:\n%s", errorsFSharp)
	}
	if !strings.Contains(errorsFSharp, "| TimeoutError") {
		t.Errorf("Errors.fs should contain TimeoutError case, got:\n%s", errorsFSharp)
	}
	if !strings.Contains(errorsFSharp, "| CancelledError") {
		t.Errorf("Errors.fs should contain CancelledError case, got:\n%s", errorsFSharp)
	}
	if !strings.Contains(errorsFSharp, "module SdkError =") {
		t.Errorf("Errors.fs should contain SdkError module, got:\n%s", errorsFSharp)
	}
	if !strings.Contains(errorsFSharp, "let isRetriable") {
		t.Errorf("Errors.fs should contain isRetriable function, got:\n%s", errorsFSharp)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.fs") {
			resourcesFSharp = f.Content
			break
		}
	}

	// Should have streaming method returning IAsyncEnumerable
	if !strings.Contains(resourcesFSharp, "member _.StreamAsync") {
		t.Errorf("Resources.fs should contain StreamAsync method, got:\n%s", resourcesFSharp)
	}
	if !strings.Contains(resourcesFSharp, "IAsyncEnumerable<ResponseEvent>") {
		t.Errorf("Resources.fs should return IAsyncEnumerable, got:\n%s", resourcesFSharp)
	}
}

func TestGenerate_DefaultNamespace(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkfsharp.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.fs") {
			clientFSharp = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then PascalCase + .Sdk
	if !strings.Contains(clientFSharp, "namespace MyAPIv2.Sdk") {
		t.Errorf("expected namespace MyAPIv2.Sdk, got:\n%s", clientFSharp)
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.fs") {
			typesFSharp = f.Content
			break
		}
	}

	if !strings.Contains(typesFSharp, "RequiredField: string") {
		t.Errorf("Types.fs should contain required field as string, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "OptionalField: string option") {
		t.Errorf("Types.fs should contain optional field as string option, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "NullableField: int option") {
		t.Errorf("Types.fs should contain nullable field as int option, got:\n%s", typesFSharp)
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.fs") {
			typesFSharp = f.Content
			break
		}
	}

	if !strings.Contains(typesFSharp, "type RequestRole =") {
		t.Errorf("Types.fs should contain RequestRole DU, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "| User") {
		t.Errorf("Types.fs should contain User case, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "| Assistant") {
		t.Errorf("Types.fs should contain Assistant case, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "| System") {
		t.Errorf("Types.fs should contain System case, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "module RequestRole =") {
		t.Errorf("Types.fs should contain RequestRole module, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "let toString") {
		t.Errorf("Types.fs should contain toString function, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "let fromString") {
		t.Errorf("Types.fs should contain fromString function, got:\n%s", typesFSharp)
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.fs") {
			typesFSharp = f.Content
			break
		}
	}

	if !strings.Contains(typesFSharp, "type ContentBlock =") {
		t.Errorf("Types.fs should contain ContentBlock DU, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "| TextBlock") {
		t.Errorf("Types.fs should contain TextBlock variant, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "| ImageBlock") {
		t.Errorf("Types.fs should contain ImageBlock variant, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "ContentBlockJsonConverter") {
		t.Errorf("Types.fs should contain JSON converter, got:\n%s", typesFSharp)
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.fs") {
			typesFSharp = f.Content
			break
		}
	}

	expectations := []string{
		"Str: string",
		"Num: int",
		"Bignum: int64",
		"Flag: bool",
		"Ratio: float",
		"Small: float32",
		"Time: DateTimeOffset",
		"Data: JsonElement",
		"Items: string list",
		"Mapping: Map<string, int>",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesFSharp, exp) {
			t.Errorf("Types.fs should contain %q, got:\n%s", exp, typesFSharp)
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.fs") {
			resourcesFSharp = f.Content
			break
		}
	}

	if !strings.Contains(resourcesFSharp, "member _.ListAsync") {
		t.Errorf("Resources.fs should contain ListAsync method, got:\n%s", resourcesFSharp)
	}
	if !strings.Contains(resourcesFSharp, "member _.CreateAsync") {
		t.Errorf("Resources.fs should contain CreateAsync method, got:\n%s", resourcesFSharp)
	}
	if !strings.Contains(resourcesFSharp, "HttpMethod.Get") {
		t.Errorf("Resources.fs should contain HttpMethod.Get, got:\n%s", resourcesFSharp)
	}
	if !strings.Contains(resourcesFSharp, "HttpMethod.Delete") {
		t.Errorf("Resources.fs should contain HttpMethod.Delete, got:\n%s", resourcesFSharp)
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var httpFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Http.fs") {
			httpFSharp = f.Content
			break
		}
	}

	if !strings.Contains(httpFSharp, "type AuthMode =") {
		t.Errorf("Http.fs should contain AuthMode DU, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, "| Bearer") {
		t.Errorf("Http.fs should contain Bearer auth mode, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, "| Basic") {
		t.Errorf("Http.fs should contain Basic auth mode, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, "| None") {
		t.Errorf("Http.fs should contain None auth mode, got:\n%s", httpFSharp)
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var httpFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Http.fs") {
			httpFSharp = f.Content
			break
		}
	}

	if !strings.Contains(httpFSharp, `"X-Custom-Header"`) {
		t.Errorf("Http.fs should contain X-Custom-Header, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, `"custom-value"`) {
		t.Errorf("Http.fs should contain custom-value, got:\n%s", httpFSharp)
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.fs") {
			typesFSharp = f.Content
			break
		}
	}

	if !strings.Contains(typesFSharp, "type StringList = string list") {
		t.Errorf("Types.fs should contain StringList type alias, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "type IntMap = Map<string, int>") {
		t.Errorf("Types.fs should contain IntMap type alias, got:\n%s", typesFSharp)
	}
}

func TestGenerate_StreamingFile_HasParser(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Streaming.fs") {
			streamingFSharp = f.Content
			break
		}
	}

	if !strings.Contains(streamingFSharp, "module SseParser") {
		t.Errorf("Streaming.fs should contain SseParser module, got:\n%s", streamingFSharp)
	}
	if !strings.Contains(streamingFSharp, "parseAsync") {
		t.Errorf("Streaming.fs should contain parseAsync function, got:\n%s", streamingFSharp)
	}
	if !strings.Contains(streamingFSharp, "IAsyncEnumerable") {
		t.Errorf("Streaming.fs should return IAsyncEnumerable, got:\n%s", streamingFSharp)
	}
}

func TestGenerate_ClientOptionsModule(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "OpenAI.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var httpFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Http.fs") {
			httpFSharp = f.Content
			break
		}
	}

	if !strings.Contains(httpFSharp, "type ClientOptions = {") {
		t.Errorf("Http.fs should contain ClientOptions record, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, "module ClientOptions =") {
		t.Errorf("Http.fs should contain ClientOptions module, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, "let defaults") {
		t.Errorf("Http.fs should contain defaults, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, "let withApiKey") {
		t.Errorf("Http.fs should contain withApiKey, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, "let withBaseUrl") {
		t.Errorf("Http.fs should contain withBaseUrl, got:\n%s", httpFSharp)
	}
	if !strings.Contains(httpFSharp, "let withTimeout") {
		t.Errorf("Http.fs should contain withTimeout, got:\n%s", httpFSharp)
	}
}

func TestGenerate_JsonPropertyNameAnnotations(t *testing.T) {
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

	files, err := sdkfsharp.Generate(svc, &sdkfsharp.Config{Namespace: "Test.Sdk"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesFSharp string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.fs") {
			typesFSharp = f.Content
			break
		}
	}

	// Should use JsonPropertyName for JSON field names
	if !strings.Contains(typesFSharp, `[<JsonPropertyName("snake_case_field")>]`) {
		t.Errorf("Types.fs should contain JsonPropertyName for snake_case_field, got:\n%s", typesFSharp)
	}
	if !strings.Contains(typesFSharp, "SnakeCaseField: string") {
		t.Errorf("Types.fs should use PascalCase for F# field name, got:\n%s", typesFSharp)
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

func writeGeneratedFSharpSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkfsharp.Config{
		Namespace: "OpenAI.Sdk",
		Version:   "0.0.0",
	}
	files, err := sdkfsharp.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "fsharp-sdk")
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
