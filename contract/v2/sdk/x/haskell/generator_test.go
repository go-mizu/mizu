package sdkhaskell_test

import (
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkhaskell "github.com/go-mizu/mizu/contract/v2/sdk/x/haskell"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkhaskell.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"openai.cabal":             false,
		"src/OpenAI.hs":            false,
		"src/OpenAI/Client.hs":     false,
		"src/OpenAI/Config.hs":     false,
		"src/OpenAI/Types.hs":      false,
		"src/OpenAI/Resources.hs":  false,
		"src/OpenAI/Streaming.hs":  false,
		"src/OpenAI/Errors.hs":     false,
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

func TestGenerate_CabalFile_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{
		PackageName: "example-sdk",
		ModuleName:  "ExampleSDK",
		Version:     "1.2.3",
		Synopsis:    "An example SDK",
		Author:      "Test Author",
		License:     "MIT",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var cabalFile string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".cabal") {
			cabalFile = f.Content
			break
		}
	}

	if !strings.Contains(cabalFile, `name:               example-sdk`) {
		t.Errorf(".cabal should contain package name, got:\n%s", cabalFile)
	}
	if !strings.Contains(cabalFile, `version:            1.2.3`) {
		t.Errorf(".cabal should contain version, got:\n%s", cabalFile)
	}
	if !strings.Contains(cabalFile, `synopsis:           An example SDK`) {
		t.Errorf(".cabal should contain synopsis, got:\n%s", cabalFile)
	}
	if !strings.Contains(cabalFile, `aeson`) {
		t.Errorf(".cabal should contain aeson dependency, got:\n%s", cabalFile)
	}
	if !strings.Contains(cabalFile, `http-conduit`) {
		t.Errorf(".cabal should contain http-conduit dependency, got:\n%s", cabalFile)
	}
	if !strings.Contains(cabalFile, `conduit`) {
		t.Errorf(".cabal should contain conduit dependency, got:\n%s", cabalFile)
	}
}

func TestGenerate_TypesFile_ContainsDataTypes(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.hs") {
			typesHs = f.Content
			break
		}
	}

	if !strings.Contains(typesHs, "data CreateRequest = CreateRequest") {
		t.Errorf("Types.hs should contain CreateRequest data type, got:\n%s", typesHs)
	}
	if !strings.Contains(typesHs, "data Response = Response") {
		t.Errorf("Types.hs should contain Response data type, got:\n%s", typesHs)
	}
	if !strings.Contains(typesHs, "instance FromJSON CreateRequest") {
		t.Errorf("Types.hs should contain FromJSON instance for CreateRequest, got:\n%s", typesHs)
	}
	if !strings.Contains(typesHs, "instance ToJSON CreateRequest") {
		t.Errorf("Types.hs should contain ToJSON instance for CreateRequest, got:\n%s", typesHs)
	}
}

func TestGenerate_ClientFile_ContainsClientModule(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.hs") {
			clientHs = f.Content
			break
		}
	}

	if !strings.Contains(clientHs, "module OpenAI.Client") {
		t.Errorf("Client.hs should contain Client module declaration, got:\n%s", clientHs)
	}
	if !strings.Contains(clientHs, "data Client = Client") {
		t.Errorf("Client.hs should contain Client data type, got:\n%s", clientHs)
	}
	if !strings.Contains(clientHs, "newClient ::") {
		t.Errorf("Client.hs should contain newClient function, got:\n%s", clientHs)
	}
	if !strings.Contains(clientHs, "newClientWith ::") {
		t.Errorf("Client.hs should contain newClientWith function, got:\n%s", clientHs)
	}
	if !strings.Contains(clientHs, "request") {
		t.Errorf("Client.hs should contain request function, got:\n%s", clientHs)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.hs") {
			resourcesHs = f.Content
			break
		}
	}

	if !strings.Contains(resourcesHs, "module OpenAI.Resources") {
		t.Errorf("Resources.hs should contain Resources module, got:\n%s", resourcesHs)
	}
	if !strings.Contains(resourcesHs, "create") {
		t.Errorf("Resources.hs should contain create method, got:\n%s", resourcesHs)
	}
	if !strings.Contains(resourcesHs, "create_") {
		t.Errorf("Resources.hs should contain create_ method (throw version), got:\n%s", resourcesHs)
	}
}

func TestGenerate_DefaultPackageName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkhaskell.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Should generate kebab-case package name
	var cabalFound bool
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".cabal") {
			cabalFound = true
			break
		}
	}

	if !cabalFound {
		var paths []string
		for _, f := range files {
			paths = append(paths, f.Path)
		}
		t.Errorf("expected .cabal file, got files: %v", paths)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.hs") {
			resourcesHs = f.Content
			break
		}
	}

	// Should have streaming method
	if !strings.Contains(resourcesHs, "streamStream") || !strings.Contains(resourcesHs, "Stream") {
		t.Errorf("Resources.hs should contain streaming method, got:\n%s", resourcesHs)
	}
}

func TestGenerate_ErrorsFile_ContainsTypes(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Errors.hs") {
			errorsHs = f.Content
			break
		}
	}

	if !strings.Contains(errorsHs, "module OpenAI.Errors") {
		t.Errorf("Errors.hs should contain Errors module, got:\n%s", errorsHs)
	}
	if !strings.Contains(errorsHs, "data SDKError") {
		t.Errorf("Errors.hs should contain SDKError data type, got:\n%s", errorsHs)
	}
	if !strings.Contains(errorsHs, "data APIError") {
		t.Errorf("Errors.hs should contain APIError data type, got:\n%s", errorsHs)
	}
	if !strings.Contains(errorsHs, "RateLimitError") {
		t.Errorf("Errors.hs should contain RateLimitError constructor, got:\n%s", errorsHs)
	}
	if !strings.Contains(errorsHs, "isRetryable") {
		t.Errorf("Errors.hs should contain isRetryable function, got:\n%s", errorsHs)
	}
}

func TestGenerate_StreamingFile_HasSSEParsing(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Streaming.hs") {
			streamingHs = f.Content
			break
		}
	}

	if !strings.Contains(streamingHs, "module OpenAI.Streaming") {
		t.Errorf("Streaming.hs should contain Streaming module, got:\n%s", streamingHs)
	}
	if !strings.Contains(streamingHs, "data Event") {
		t.Errorf("Streaming.hs should contain Event data type, got:\n%s", streamingHs)
	}
	if !strings.Contains(streamingHs, "parseSSE") {
		t.Errorf("Streaming.hs should contain parseSSE function, got:\n%s", streamingHs)
	}
	if !strings.Contains(streamingHs, `"[DONE]"`) {
		t.Errorf("Streaming.hs should handle [DONE] terminator, got:\n%s", streamingHs)
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

	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "test-api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.hs") {
			typesHs = f.Content
			break
		}
	}

	if !strings.Contains(typesHs, "!Text") {
		t.Errorf("Types.hs should contain strict Text type for required field, got:\n%s", typesHs)
	}
	if !strings.Contains(typesHs, "Maybe") {
		t.Errorf("Types.hs should use Maybe for optional/nullable fields, got:\n%s", typesHs)
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

	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "test-api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.hs") {
			typesHs = f.Content
			break
		}
	}

	// Should generate enum type
	if !strings.Contains(typesHs, "data RequestRole") {
		t.Errorf("Types.hs should contain RequestRole enum type, got:\n%s", typesHs)
	}
	// Enum values should be PascalCase prefixed with type
	if !strings.Contains(typesHs, "RequestUser") {
		t.Errorf("Types.hs should contain RequestUser enum value, got:\n%s", typesHs)
	}
	if !strings.Contains(typesHs, "RequestAssistant") {
		t.Errorf("Types.hs should contain RequestAssistant enum value, got:\n%s", typesHs)
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

	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "test-api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.hs") {
			typesHs = f.Content
			break
		}
	}

	if !strings.Contains(typesHs, "data ContentBlock") {
		t.Errorf("Types.hs should contain ContentBlock union type, got:\n%s", typesHs)
	}
	// Check for variant constructors
	if !strings.Contains(typesHs, "ContentBlockTextBlock") {
		t.Errorf("Types.hs should contain ContentBlockTextBlock variant, got:\n%s", typesHs)
	}
	if !strings.Contains(typesHs, "ContentBlockImageBlock") {
		t.Errorf("Types.hs should contain ContentBlockImageBlock variant, got:\n%s", typesHs)
	}
	// Check for tag-based parsing
	if !strings.Contains(typesHs, `"type"`) {
		t.Errorf("Types.hs should parse by type tag, got:\n%s", typesHs)
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

	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "test-api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.hs") {
			typesHs = f.Content
			break
		}
	}

	// Haskell types
	expectations := []string{
		"Text",      // string
		"Int",       // int
		"Int64",     // int64
		"Bool",      // bool
		"Double",    // float64
		"Float",     // float32
		"UTCTime",   // time.Time
		"Value",     // json.RawMessage
		"[Text]",    // []string
		"Map Text",  // map[string]int
	}

	for _, exp := range expectations {
		if !strings.Contains(typesHs, exp) {
			t.Errorf("Types.hs should contain %q, got:\n%s", exp, typesHs)
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

	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "test-api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.hs") {
			resourcesHs = f.Content
			break
		}
	}

	// Check HTTP methods are used
	if !strings.Contains(resourcesHs, "methodGet") {
		t.Errorf("Resources.hs should contain methodGet, got:\n%s", resourcesHs)
	}
	if !strings.Contains(resourcesHs, "methodPost") {
		t.Errorf("Resources.hs should contain methodPost, got:\n%s", resourcesHs)
	}
	if !strings.Contains(resourcesHs, "methodPut") {
		t.Errorf("Resources.hs should contain methodPut, got:\n%s", resourcesHs)
	}
	if !strings.Contains(resourcesHs, "methodDelete") {
		t.Errorf("Resources.hs should contain methodDelete, got:\n%s", resourcesHs)
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

	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "test-api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Client.hs") {
			clientHs = f.Content
			break
		}
	}

	if !strings.Contains(clientHs, "buildAuthHeaders") {
		t.Errorf("Client.hs should contain buildAuthHeaders function, got:\n%s", clientHs)
	}
	if !strings.Contains(clientHs, "BearerAuth") {
		t.Errorf("Client.hs should handle BearerAuth, got:\n%s", clientHs)
	}
	if !strings.Contains(clientHs, "BasicAuth") {
		t.Errorf("Client.hs should handle BasicAuth, got:\n%s", clientHs)
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

	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "test-api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var configHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Config.hs") {
			configHs = f.Content
			break
		}
	}

	if !strings.Contains(configHs, `"X-Custom-Header"`) {
		t.Errorf("Config.hs should contain X-Custom-Header, got:\n%s", configHs)
	}
	if !strings.Contains(configHs, `"custom-value"`) {
		t.Errorf("Config.hs should contain custom-value, got:\n%s", configHs)
	}
}

func TestGenerate_CamelCaseNaming(t *testing.T) {
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

	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "test-api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesHs string
	var typesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Resources.hs") {
			resourcesHs = f.Content
		}
		if strings.HasSuffix(f.Path, "Types.hs") {
			typesHs = f.Content
		}
	}

	// Should use camelCase for Haskell functions
	if !strings.Contains(resourcesHs, "getById") {
		t.Errorf("Resources.hs should use camelCase function name, got:\n%s", resourcesHs)
	}

	// Should use prefixed camelCase for Haskell record fields
	if !strings.Contains(typesHs, "requestUserId") || !strings.Contains(typesHs, "requestProfileType") {
		t.Errorf("Types.hs should use prefixed camelCase field names, got:\n%s", typesHs)
	}
}

func TestGenerate_MainFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var mainHs string
	for _, f := range files {
		if f.Path == "src/OpenAI.hs" {
			mainHs = f.Content
			break
		}
	}

	if !strings.Contains(mainHs, "module OpenAI") {
		t.Errorf("main.hs should define OpenAI module, got:\n%s", mainHs)
	}
	if !strings.Contains(mainHs, "newClient") {
		t.Errorf("main.hs should export newClient, got:\n%s", mainHs)
	}
	if !strings.Contains(mainHs, "newClientWith") {
		t.Errorf("main.hs should export newClientWith, got:\n%s", mainHs)
	}
	if !strings.Contains(mainHs, "module OpenAI.Types") {
		t.Errorf("main.hs should re-export Types, got:\n%s", mainHs)
	}
}

func TestGenerate_ConfigFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var configHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Config.hs") {
			configHs = f.Content
			break
		}
	}

	if !strings.Contains(configHs, "module OpenAI.Config") {
		t.Errorf("Config.hs should define Config module, got:\n%s", configHs)
	}
	if !strings.Contains(configHs, "data Config = Config") {
		t.Errorf("Config.hs should define Config data type, got:\n%s", configHs)
	}
	if !strings.Contains(configHs, "apiKey") {
		t.Errorf("Config.hs should have apiKey field, got:\n%s", configHs)
	}
	if !strings.Contains(configHs, "baseUrl") {
		t.Errorf("Config.hs should have baseUrl field, got:\n%s", configHs)
	}
	if !strings.Contains(configHs, "timeout") {
		t.Errorf("Config.hs should have timeout field, got:\n%s", configHs)
	}
	if !strings.Contains(configHs, "defaultConfig") {
		t.Errorf("Config.hs should have defaultConfig, got:\n%s", configHs)
	}
	if !strings.Contains(configHs, "configFromEnv") {
		t.Errorf("Config.hs should have configFromEnv, got:\n%s", configHs)
	}
}

func TestGenerate_AesonInstances(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Types.hs") {
			typesHs = f.Content
			break
		}
	}

	// Should import Aeson
	if !strings.Contains(typesHs, "Data.Aeson") {
		t.Errorf("Types.hs should import Data.Aeson, got:\n%s", typesHs)
	}
	// Should have FromJSON and ToJSON instances
	if !strings.Contains(typesHs, "instance FromJSON") {
		t.Errorf("Types.hs should have FromJSON instances, got:\n%s", typesHs)
	}
	if !strings.Contains(typesHs, "instance ToJSON") {
		t.Errorf("Types.hs should have ToJSON instances, got:\n%s", typesHs)
	}
}

func TestGenerate_EnvironmentVariables(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkhaskell.Generate(svc, &sdkhaskell.Config{PackageName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var configHs string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "Config.hs") {
			configHs = f.Content
			break
		}
	}

	// Should have env var names based on service name (screaming snake case converts "OpenAI" -> "OPEN_AI")
	if !strings.Contains(configHs, "OPEN_AI_API_KEY") {
		t.Errorf("Config.hs should reference OPEN_AI_API_KEY env var, got:\n%s", configHs)
	}
	if !strings.Contains(configHs, "lookupEnv") {
		t.Errorf("Config.hs should use lookupEnv, got:\n%s", configHs)
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
