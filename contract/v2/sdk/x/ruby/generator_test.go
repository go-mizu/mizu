package sdkruby_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkruby "github.com/go-mizu/mizu/contract/v2/sdk/x/ruby"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkruby.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "openai", ModuleName: "OpenAI", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"openai.gemspec":            false,
		"Gemfile":                   false,
		"lib/openai.rb":             false,
		"lib/openai/version.rb":    false,
		"lib/openai/client.rb":     false,
		"lib/openai/types.rb":      false,
		"lib/openai/resources.rb":  false,
		"lib/openai/streaming.rb":  false,
		"lib/openai/errors.rb":     false,
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

func TestGenerate_Gemspec_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{
		GemName:    "example_sdk",
		ModuleName: "ExampleSDK",
		Version:    "1.2.3",
		Authors:    []string{"Test Author"},
		Homepage:   "https://github.com/example/example_sdk",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var gemspec string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".gemspec") {
			gemspec = f.Content
			break
		}
	}

	if !strings.Contains(gemspec, `spec.name = "example_sdk"`) {
		t.Errorf("gemspec should contain gem name, got:\n%s", gemspec)
	}
	if !strings.Contains(gemspec, `spec.version = ExampleSDK::VERSION`) {
		t.Errorf("gemspec should contain version reference, got:\n%s", gemspec)
	}
	if !strings.Contains(gemspec, `"Test Author"`) {
		t.Errorf("gemspec should contain author, got:\n%s", gemspec)
	}
	if !strings.Contains(gemspec, `faraday`) {
		t.Errorf("gemspec should contain faraday dependency, got:\n%s", gemspec)
	}
}

func TestGenerate_TypesFile_ContainsClasses(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rb") {
			typesRuby = f.Content
			break
		}
	}

	if !strings.Contains(typesRuby, "class CreateRequest < Base") {
		t.Errorf("types.rb should contain CreateRequest class, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, "class Response < Base") {
		t.Errorf("types.rb should contain Response class, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, "module Types") {
		t.Errorf("types.rb should contain Types module, got:\n%s", typesRuby)
	}
}

func TestGenerate_ClientFile_ContainsClientClass(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.rb") {
			clientRuby = f.Content
			break
		}
	}

	if !strings.Contains(clientRuby, "class Client") {
		t.Errorf("client.rb should contain Client class, got:\n%s", clientRuby)
	}
	if !strings.Contains(clientRuby, "class Configuration") {
		t.Errorf("client.rb should contain Configuration class, got:\n%s", clientRuby)
	}
	if !strings.Contains(clientRuby, "@responses = ResponsesResource.new(self)") {
		t.Errorf("client.rb should initialize responses resource, got:\n%s", clientRuby)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.rb") {
			resourcesRuby = f.Content
			break
		}
	}

	if !strings.Contains(resourcesRuby, "class ResponsesResource") {
		t.Errorf("resources.rb should contain ResponsesResource class, got:\n%s", resourcesRuby)
	}
	if !strings.Contains(resourcesRuby, "def create(") {
		t.Errorf("resources.rb should contain create method, got:\n%s", resourcesRuby)
	}
}

func TestGenerate_DefaultGemName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkruby.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var found bool
	for _, f := range files {
		if strings.HasSuffix(f.Path, "my_api_v2.gemspec") {
			found = true
			break
		}
	}

	if !found {
		var paths []string
		for _, f := range files {
			paths = append(paths, f.Path)
		}
		t.Errorf("expected gemspec with snake_case name, got files: %v", paths)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.rb") {
			resourcesRuby = f.Content
			break
		}
	}

	// Should have streaming method with block support
	if !strings.Contains(resourcesRuby, "def stream(") {
		t.Errorf("resources.rb should contain stream method, got:\n%s", resourcesRuby)
	}
	if !strings.Contains(resourcesRuby, "Enumerator.new") {
		t.Errorf("resources.rb should return Enumerator for streaming, got:\n%s", resourcesRuby)
	}
}

func TestGenerate_ErrorsFile_ContainsExceptions(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var errorsRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "errors.rb") {
			errorsRuby = f.Content
			break
		}
	}

	if !strings.Contains(errorsRuby, "class Error < StandardError") {
		t.Errorf("errors.rb should contain Error base class, got:\n%s", errorsRuby)
	}
	if !strings.Contains(errorsRuby, "class APIError < Error") {
		t.Errorf("errors.rb should contain APIError class, got:\n%s", errorsRuby)
	}
	if !strings.Contains(errorsRuby, "class RateLimitError < APIError") {
		t.Errorf("errors.rb should contain RateLimitError class, got:\n%s", errorsRuby)
	}
	if !strings.Contains(errorsRuby, "attr_reader :status") {
		t.Errorf("errors.rb should contain status reader, got:\n%s", errorsRuby)
	}
}

func TestGenerate_StreamingFile_HasSSEParsing(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var streamingRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "streaming.rb") {
			streamingRuby = f.Content
			break
		}
	}

	if !strings.Contains(streamingRuby, "module Streaming") {
		t.Errorf("streaming.rb should contain Streaming module, got:\n%s", streamingRuby)
	}
	if !strings.Contains(streamingRuby, "parse_sse_event") {
		t.Errorf("streaming.rb should contain parse_sse_event method, got:\n%s", streamingRuby)
	}
	if !strings.Contains(streamingRuby, `"[DONE]"`) {
		t.Errorf("streaming.rb should handle [DONE] terminator, got:\n%s", streamingRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rb") {
			typesRuby = f.Content
			break
		}
	}

	if !strings.Contains(typesRuby, "attr_accessor :required_field") {
		t.Errorf("types.rb should contain required_field, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, "attr_accessor :optional_field") {
		t.Errorf("types.rb should contain optional_field, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, "optional_field: nil") {
		t.Errorf("types.rb should have nil default for optional field, got:\n%s", typesRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rb") {
			typesRuby = f.Content
			break
		}
	}

	if !strings.Contains(typesRuby, "module Role") {
		t.Errorf("types.rb should contain Role module, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, `USER = "user"`) {
		t.Errorf("types.rb should contain USER constant, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, `ASSISTANT = "assistant"`) {
		t.Errorf("types.rb should contain ASSISTANT constant, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, "def self.valid?(value)") {
		t.Errorf("types.rb should contain valid? method, got:\n%s", typesRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rb") {
			typesRuby = f.Content
			break
		}
	}

	if !strings.Contains(typesRuby, "module ContentBlock") {
		t.Errorf("types.rb should contain ContentBlock module, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, "def self.from_hash(hash)") {
		t.Errorf("types.rb should contain from_hash factory method, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, `when "text"`) {
		t.Errorf("types.rb should handle text variant, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, "TextBlock.from_hash(hash)") {
		t.Errorf("types.rb should create TextBlock variant, got:\n%s", typesRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rb") {
			typesRuby = f.Content
			break
		}
	}

	// Ruby uses YARD documentation for types
	expectations := []string{
		"attr_accessor :str",
		"attr_accessor :num",
		"attr_accessor :bignum",
		"attr_accessor :flag",
		"attr_accessor :ratio",
		"attr_accessor :small",
		"attr_accessor :time",
		"attr_accessor :data",
		"attr_accessor :items",
		"attr_accessor :mapping",
	}

	for _, exp := range expectations {
		if !strings.Contains(typesRuby, exp) {
			t.Errorf("types.rb should contain %q, got:\n%s", exp, typesRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.rb") {
			resourcesRuby = f.Content
			break
		}
	}

	// Check HTTP methods are used
	if !strings.Contains(resourcesRuby, "method: :get") {
		t.Errorf("resources.rb should contain :get method, got:\n%s", resourcesRuby)
	}
	if !strings.Contains(resourcesRuby, "method: :post") {
		t.Errorf("resources.rb should contain :post method, got:\n%s", resourcesRuby)
	}
	if !strings.Contains(resourcesRuby, "method: :put") {
		t.Errorf("resources.rb should contain :put method, got:\n%s", resourcesRuby)
	}
	if !strings.Contains(resourcesRuby, "method: :delete") {
		t.Errorf("resources.rb should contain :delete method, got:\n%s", resourcesRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.rb") {
			clientRuby = f.Content
			break
		}
	}

	if !strings.Contains(clientRuby, "attr_accessor :auth_mode") {
		t.Errorf("client.rb should contain auth_mode accessor, got:\n%s", clientRuby)
	}
	if !strings.Contains(clientRuby, "when :bearer") {
		t.Errorf("client.rb should handle :bearer auth mode, got:\n%s", clientRuby)
	}
	if !strings.Contains(clientRuby, "when :basic") {
		t.Errorf("client.rb should handle :basic auth mode, got:\n%s", clientRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.rb") {
			clientRuby = f.Content
			break
		}
	}

	if !strings.Contains(clientRuby, `"X-Custom-Header"`) {
		t.Errorf("client.rb should contain X-Custom-Header, got:\n%s", clientRuby)
	}
	if !strings.Contains(clientRuby, `"custom-value"`) {
		t.Errorf("client.rb should contain custom-value, got:\n%s", clientRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesRuby string
	var typesRuby string
	var clientRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "resources.rb") {
			resourcesRuby = f.Content
		}
		if strings.HasSuffix(f.Path, "types.rb") {
			typesRuby = f.Content
		}
		if strings.HasSuffix(f.Path, "client.rb") {
			clientRuby = f.Content
		}
	}

	// Should use snake_case for Ruby methods
	if !strings.Contains(resourcesRuby, "def get_by_id(") {
		t.Errorf("resources.rb should use snake_case method name, got:\n%s", resourcesRuby)
	}
	// Resource attr_reader is in client.rb, not resources.rb
	if !strings.Contains(clientRuby, "attr_reader :user_profiles") {
		t.Errorf("client.rb should use snake_case resource name, got:\n%s", clientRuby)
	}

	// Should use snake_case for Ruby attributes
	if !strings.Contains(typesRuby, "attr_accessor :user_id") {
		t.Errorf("types.rb should use snake_case attribute name, got:\n%s", typesRuby)
	}
}

func TestGenerate_VersionFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{
		GemName:    "openai",
		ModuleName: "OpenAI",
		Version:    "2.3.4",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var versionRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "version.rb") {
			versionRuby = f.Content
			break
		}
	}

	if !strings.Contains(versionRuby, `VERSION = "2.3.4"`) {
		t.Errorf("version.rb should contain version constant, got:\n%s", versionRuby)
	}
	if !strings.Contains(versionRuby, "module OpenAI") {
		t.Errorf("version.rb should be in module, got:\n%s", versionRuby)
	}
}

func TestGenerate_MainLibFile(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "openai", ModuleName: "OpenAI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var libRuby string
	for _, f := range files {
		if f.Path == "lib/openai.rb" {
			libRuby = f.Content
			break
		}
	}

	if !strings.Contains(libRuby, `require_relative "openai/version"`) {
		t.Errorf("lib.rb should require version, got:\n%s", libRuby)
	}
	if !strings.Contains(libRuby, `require_relative "openai/client"`) {
		t.Errorf("lib.rb should require client, got:\n%s", libRuby)
	}
	if !strings.Contains(libRuby, "module OpenAI") {
		t.Errorf("lib.rb should define module, got:\n%s", libRuby)
	}
	if !strings.Contains(libRuby, "def self.configure") {
		t.Errorf("lib.rb should have configure method, got:\n%s", libRuby)
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

	files, err := sdkruby.Generate(svc, &sdkruby.Config{GemName: "test_api", ModuleName: "TestAPI"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesRuby string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.rb") {
			typesRuby = f.Content
			break
		}
	}

	if !strings.Contains(typesRuby, "StringList = Array") {
		t.Errorf("types.rb should contain StringList alias, got:\n%s", typesRuby)
	}
	if !strings.Contains(typesRuby, "IntMap = Hash") {
		t.Errorf("types.rb should contain IntMap alias, got:\n%s", typesRuby)
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

func writeGeneratedRubySDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkruby.Config{
		GemName:    "openai",
		ModuleName: "OpenAI",
		Version:    "0.0.0",
	}
	files, err := sdkruby.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "ruby-sdk")
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
