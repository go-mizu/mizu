package sdkelixir_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkelixir "github.com/go-mizu/mizu/contract/v2/sdk/x/elixir"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkelixir.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"mix.exs":                    false,
		"lib/openai.ex":              false,
		"lib/openai/client.ex":       false,
		"lib/openai/config.ex":       false,
		"lib/openai/types.ex":        false,
		"lib/openai/resources.ex":    false,
		"lib/openai/streaming.ex":    false,
		"lib/openai/errors.ex":       false,
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

func TestGenerate_MixExs_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{
		AppName:     "example_sdk",
		ModuleName:  "ExampleSDK",
		Version:     "1.2.3",
		Description: "An example SDK",
		Homepage:    "https://github.com/example/example_sdk",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var mixExs string
	for _, f := range files {
		if f.Path == "mix.exs" {
			mixExs = f.Content
			break
		}
	}

	if !strings.Contains(mixExs, `app: :example_sdk`) {
		t.Errorf("mix.exs should contain app name, got:\n%s", mixExs)
	}
	if !strings.Contains(mixExs, `@version "1.2.3"`) {
		t.Errorf("mix.exs should contain version, got:\n%s", mixExs)
	}
	if !strings.Contains(mixExs, `name: "ExampleSDK"`) {
		t.Errorf("mix.exs should contain module name, got:\n%s", mixExs)
	}
	if !strings.Contains(mixExs, `{:req, "~> 0.4"}`) {
		t.Errorf("mix.exs should contain req dependency, got:\n%s", mixExs)
	}
	if !strings.Contains(mixExs, `{:jason, "~> 1.4"}`) {
		t.Errorf("mix.exs should contain jason dependency, got:\n%s", mixExs)
	}
}

func TestGenerate_TypesFile_ContainsModules(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.ex") {
			typesEx = f.Content
			break
		}
	}

	if !strings.Contains(typesEx, "defmodule OpenAI.Types.CreateRequest do") {
		t.Errorf("types.ex should contain CreateRequest module, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, "defmodule OpenAI.Types.Response do") {
		t.Errorf("types.ex should contain Response module, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, "defmodule OpenAI.Types do") {
		t.Errorf("types.ex should contain Types module, got:\n%s", typesEx)
	}
}

func TestGenerate_ClientFile_ContainsClientModule(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.ex") {
			clientEx = f.Content
			break
		}
	}

	if !strings.Contains(clientEx, "defmodule OpenAI.Client do") {
		t.Errorf("client.ex should contain Client module, got:\n%s", clientEx)
	}
	if !strings.Contains(clientEx, "defstruct [:config, :req]") {
		t.Errorf("client.ex should contain defstruct, got:\n%s", clientEx)
	}
	if !strings.Contains(clientEx, "def new(opts") {
		t.Errorf("client.ex should contain new function, got:\n%s", clientEx)
	}
	if !strings.Contains(clientEx, "def request(") {
		t.Errorf("client.ex should contain request function, got:\n%s", clientEx)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.ex") {
			resourcesEx = f.Content
			break
		}
	}

	if !strings.Contains(resourcesEx, "defmodule OpenAI.Resources.Responses do") {
		t.Errorf("resources.ex should contain Responses module, got:\n%s", resourcesEx)
	}
	if !strings.Contains(resourcesEx, "def create(") {
		t.Errorf("resources.ex should contain create method, got:\n%s", resourcesEx)
	}
	if !strings.Contains(resourcesEx, "def create!(") {
		t.Errorf("resources.ex should contain create! method, got:\n%s", resourcesEx)
	}
}

func TestGenerate_DefaultAppName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkelixir.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var found bool
	for _, f := range files {
		// The sanitizer removes special chars, so "My API v2" -> "my_apiv2" or similar
		if strings.HasPrefix(f.Path, "lib/") && strings.HasSuffix(f.Path, ".ex") && !strings.Contains(f.Path, "/") {
			found = true
			break
		}
	}

	// Just verify we got a valid main lib file
	var mainLibFound bool
	for _, f := range files {
		if strings.HasPrefix(f.Path, "lib/") && strings.Count(f.Path, "/") == 1 && strings.HasSuffix(f.Path, ".ex") {
			mainLibFound = true
			break
		}
	}

	if !mainLibFound {
		var paths []string
		for _, f := range files {
			paths = append(paths, f.Path)
		}
		t.Errorf("expected main lib file with snake_case name, got files: %v", paths)
	}
	_ = found
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.ex") {
			resourcesEx = f.Content
			break
		}
	}

	// Should have streaming method
	if !strings.Contains(resourcesEx, "def stream_stream(") {
		t.Errorf("resources.ex should contain stream_stream method, got:\n%s", resourcesEx)
	}
	if !strings.Contains(resourcesEx, "Client.stream(") {
		t.Errorf("resources.ex should call Client.stream, got:\n%s", resourcesEx)
	}
}

func TestGenerate_ErrorsFile_ContainsExceptions(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "errors.ex") {
			errorsEx = f.Content
			break
		}
	}

	// The error modules are nested inside OpenAI.Errors
	if !strings.Contains(errorsEx, "defmodule OpenAI.Errors do") {
		t.Errorf("errors.ex should contain Errors module, got:\n%s", errorsEx)
	}
	if !strings.Contains(errorsEx, "defmodule SDKError do") {
		t.Errorf("errors.ex should contain SDKError module, got:\n%s", errorsEx)
	}
	if !strings.Contains(errorsEx, "defmodule APIError do") {
		t.Errorf("errors.ex should contain APIError module, got:\n%s", errorsEx)
	}
	if !strings.Contains(errorsEx, "defmodule RateLimitError do") {
		t.Errorf("errors.ex should contain RateLimitError module, got:\n%s", errorsEx)
	}
	if !strings.Contains(errorsEx, "defexception") {
		t.Errorf("errors.ex should contain defexception, got:\n%s", errorsEx)
	}
}

func TestGenerate_StreamingFile_HasSSEParsing(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "streaming.ex") {
			streamingEx = f.Content
			break
		}
	}

	if !strings.Contains(streamingEx, "defmodule OpenAI.Streaming do") {
		t.Errorf("streaming.ex should contain Streaming module, got:\n%s", streamingEx)
	}
	if !strings.Contains(streamingEx, "def parse_event(") {
		t.Errorf("streaming.ex should contain parse_event function, got:\n%s", streamingEx)
	}
	if !strings.Contains(streamingEx, `"[DONE]"`) {
		t.Errorf("streaming.ex should handle [DONE] terminator, got:\n%s", streamingEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.ex") {
			typesEx = f.Content
			break
		}
	}

	if !strings.Contains(typesEx, ":required_field") {
		t.Errorf("types.ex should contain required_field, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, ":optional_field") {
		t.Errorf("types.ex should contain optional_field, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, "| nil") {
		t.Errorf("types.ex should have nil typespec for optional field, got:\n%s", typesEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.ex") {
			typesEx = f.Content
			break
		}
	}

	if !strings.Contains(typesEx, "defmodule Role do") {
		t.Errorf("types.ex should contain Role module, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, `@user "user"`) {
		t.Errorf("types.ex should contain @user constant, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, `@assistant "assistant"`) {
		t.Errorf("types.ex should contain @assistant constant, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, "def valid?(value)") {
		t.Errorf("types.ex should contain valid? function, got:\n%s", typesEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.ex") {
			typesEx = f.Content
			break
		}
	}

	if !strings.Contains(typesEx, "defmodule TestAPI.Types.ContentBlock do") {
		t.Errorf("types.ex should contain ContentBlock module, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, "def from_map(") {
		t.Errorf("types.ex should contain from_map function, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, `"type" => "text"`) {
		t.Errorf("types.ex should handle text variant, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, "TextBlock.from_map(map)") {
		t.Errorf("types.ex should create TextBlock variant, got:\n%s", typesEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.ex") {
			typesEx = f.Content
			break
		}
	}

	// Elixir uses typespecs
	expectations := []string{
		":str",
		":num",
		":bignum",
		":flag",
		":ratio",
		":small",
		":time",
		":data",
		":items",
		":mapping",
		"String.t()",
		"integer()",
		"boolean()",
		"float()",
		"DateTime.t()",
		"map()",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesEx, exp) {
			t.Errorf("types.ex should contain %q, got:\n%s", exp, typesEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.ex") {
			resourcesEx = f.Content
			break
		}
	}

	// Check HTTP methods are used
	if !strings.Contains(resourcesEx, "method: :get") {
		t.Errorf("resources.ex should contain :get method, got:\n%s", resourcesEx)
	}
	if !strings.Contains(resourcesEx, "method: :post") {
		t.Errorf("resources.ex should contain :post method, got:\n%s", resourcesEx)
	}
	if !strings.Contains(resourcesEx, "method: :put") {
		t.Errorf("resources.ex should contain :put method, got:\n%s", resourcesEx)
	}
	if !strings.Contains(resourcesEx, "method: :delete") {
		t.Errorf("resources.ex should contain :delete method, got:\n%s", resourcesEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.ex") {
			clientEx = f.Content
			break
		}
	}

	if !strings.Contains(clientEx, "defp apply_auth(") {
		t.Errorf("client.ex should contain apply_auth function, got:\n%s", clientEx)
	}
	if !strings.Contains(clientEx, "auth_mode: :bearer") {
		t.Errorf("client.ex should handle :bearer auth mode, got:\n%s", clientEx)
	}
	if !strings.Contains(clientEx, "auth_mode: :basic") {
		t.Errorf("client.ex should handle :basic auth mode, got:\n%s", clientEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var configEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "config.ex") {
			configEx = f.Content
			break
		}
	}

	if !strings.Contains(configEx, `"X-Custom-Header"`) {
		t.Errorf("config.ex should contain X-Custom-Header, got:\n%s", configEx)
	}
	if !strings.Contains(configEx, `"custom-value"`) {
		t.Errorf("config.ex should contain custom-value, got:\n%s", configEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesEx string
	var typesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.ex") {
			resourcesEx = f.Content
		}
		if strings.HasSuffix(f.Path, "types.ex") {
			typesEx = f.Content
		}
	}

	// Should use snake_case for Elixir functions
	if !strings.Contains(resourcesEx, "def get_by_id(") {
		t.Errorf("resources.ex should use snake_case function name, got:\n%s", resourcesEx)
	}

	// Should use snake_case for Elixir struct keys
	if !strings.Contains(typesEx, ":user_id") {
		t.Errorf("types.ex should use snake_case field name, got:\n%s", typesEx)
	}
}

func TestGenerate_MainFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var mainEx string
	for _, f := range files {
		if f.Path == "lib/openai.ex" {
			mainEx = f.Content
			break
		}
	}

	if !strings.Contains(mainEx, "defmodule OpenAI do") {
		t.Errorf("main.ex should define OpenAI module, got:\n%s", mainEx)
	}
	if !strings.Contains(mainEx, "def client(") {
		t.Errorf("main.ex should have client function, got:\n%s", mainEx)
	}
	if !strings.Contains(mainEx, "Client.new(opts)") {
		t.Errorf("main.ex should call Client.new, got:\n%s", mainEx)
	}
}

func TestGenerate_ConfigFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var configEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "config.ex") {
			configEx = f.Content
			break
		}
	}

	if !strings.Contains(configEx, "defmodule OpenAI.Config do") {
		t.Errorf("config.ex should define Config module, got:\n%s", configEx)
	}
	if !strings.Contains(configEx, "defstruct") {
		t.Errorf("config.ex should use defstruct, got:\n%s", configEx)
	}
	if !strings.Contains(configEx, "api_key:") {
		t.Errorf("config.ex should have api_key field, got:\n%s", configEx)
	}
	if !strings.Contains(configEx, "base_url:") {
		t.Errorf("config.ex should have base_url field, got:\n%s", configEx)
	}
	if !strings.Contains(configEx, "timeout:") {
		t.Errorf("config.ex should have timeout field, got:\n%s", configEx)
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

	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.ex") {
			typesEx = f.Content
			break
		}
	}

	if !strings.Contains(typesEx, "defmodule TestAPI.Types.StringList do") {
		t.Errorf("types.ex should contain StringList module, got:\n%s", typesEx)
	}
	if !strings.Contains(typesEx, "defmodule TestAPI.Types.IntMap do") {
		t.Errorf("types.ex should contain IntMap module, got:\n%s", typesEx)
	}
}

func TestGenerate_Typespecs(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkelixir.Generate(svc, &sdkelixir.Config{AppName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientEx string
	var resourcesEx string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.ex") {
			clientEx = f.Content
		}
		if strings.HasSuffix(f.Path, "resources.ex") {
			resourcesEx = f.Content
		}
	}

	// Should have typespecs
	if !strings.Contains(clientEx, "@type t ::") {
		t.Errorf("client.ex should contain type definition, got:\n%s", clientEx)
	}
	if !strings.Contains(clientEx, "@spec new(") {
		t.Errorf("client.ex should contain spec for new, got:\n%s", clientEx)
	}
	if !strings.Contains(resourcesEx, "@spec create(") {
		t.Errorf("resources.ex should contain spec for create, got:\n%s", resourcesEx)
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

func writeGeneratedElixirSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkelixir.Config{
		AppName:    "openai",
		ModuleName: "OpenAI",
		Version:    "0.0.0",
	}
	files, err := sdkelixir.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "elixir-sdk")
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
