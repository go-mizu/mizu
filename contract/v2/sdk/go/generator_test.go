package sdkgo

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	contract "github.com/go-mizu/mizu/contract/v2"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ValidGo_ParsesAndTypechecks_Minimal(t *testing.T) {
	svc := &contract.Service{
		Name: "OpenAI",
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
		},
	}

	files, err := Generate(svc, &Config{Package: "openai", Filename: "client.go"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	src := files[0].Content

	mustParseAndTypecheck(t, "openai", src)

	// DX checks: resources exposed as fields
	if !strings.Contains(src, "type Client struct {") ||
		!strings.Contains(src, "Responses *ResponsesResource") {
		t.Fatalf("expected Client to expose Responses resource field, src:\n%s", src)
	}

	// Method signature should be typed and discoverable
	if !strings.Contains(src, "func (r *ResponsesResource) Create(ctx context.Context, in *CreateRequest) (*Response, error)") {
		t.Fatalf("expected typed Create method signature, src:\n%s", src)
	}
}

func TestGenerate_DefaultPackageName_SanitizedLowercased(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src := files[0].Content

	// sanitizeIdent keeps letters/digits/underscore, then lowercased
	// "My API!! v2" -> "MyAPIv2" -> "myapiv2"
	if !strings.Contains(src, "package myapiv2") {
		t.Fatalf("expected package myapiv2, src:\n%s", src)
	}

	mustParseAndTypecheck(t, "myapiv2", src)
}

func TestGenerate_Defaults_BaseURL_Headers_AuthApplied(t *testing.T) {
	svc := &contract.Service{
		Name: "Svc",
		Defaults: &contract.Defaults{
			BaseURL: "https://example.com/",
			Auth:    "bearer",
			Headers: map[string]string{
				"x-b": "2",
				"x-a": "1",
			},
		},
		Resources: []*contract.Resource{
			{Name: "r", Methods: []*contract.Method{{Name: "m"}}},
		},
	}

	files, err := Generate(svc, &Config{Package: "svc"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src := files[0].Content
	mustParseAndTypecheck(t, "svc", src)

	// BaseURL trimmed
	if !strings.Contains(src, `baseURL: "https://example.com",`) {
		t.Fatalf("expected baseURL trimmed, src:\n%s", src)
	}

	// Auth defaulted from Defaults.Auth
	if !strings.Contains(src, `auth:    "bearer",`) {
		t.Fatalf("expected auth set from defaults, src:\n%s", src)
	}

	// Headers deterministic order: x-a then x-b
	iA := strings.Index(src, `c.headers["x-a"] = "1"`)
	iB := strings.Index(src, `c.headers["x-b"] = "2"`)
	if iA < 0 || iB < 0 || iA > iB {
		t.Fatalf("expected sorted header insertion (x-a before x-b), src:\n%s", src)
	}
}

func TestGenerate_Imports_TimeOnlyWhenUsed(t *testing.T) {
	// No time usage
	svc1 := &contract.Service{
		Name: "Svc",
		Resources: []*contract.Resource{
			{Name: "r", Methods: []*contract.Method{{Name: "m"}}},
		},
		Types: []*contract.Type{
			{Name: "X", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}}},
		},
	}
	files1, err := Generate(svc1, &Config{Package: "svc"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src1 := files1[0].Content
	mustParseAndTypecheck(t, "svc", src1)
	if strings.Contains(src1, `"time"`) {
		t.Fatalf("did not expect time import when unused, src:\n%s", src1)
	}

	// With time usage
	svc2 := &contract.Service{
		Name: "Svc",
		Resources: []*contract.Resource{
			{Name: "r", Methods: []*contract.Method{{Name: "m"}}},
		},
		Types: []*contract.Type{
			{Name: "X", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "created_at", Type: "time.Time"}}},
		},
	}
	files2, err := Generate(svc2, &Config{Package: "svc"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src2 := files2[0].Content
	mustParseAndTypecheck(t, "svc", src2)
	if !strings.Contains(src2, `"time"`) {
		t.Fatalf("expected time import when used, src:\n%s", src2)
	}
}

func TestGenerate_Streaming_SSE_EmitsEventStreamAndBufioImport(t *testing.T) {
	svc := &contract.Service{
		Name: "Svc",
		Resources: []*contract.Resource{
			{
				Name: "responses",
				Methods: []*contract.Method{
					{
						Name: "stream",
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
			{Name: "ResponseEvent", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "type", Type: "string"}}},
		},
	}

	files, err := Generate(svc, &Config{Package: "svc"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src := files[0].Content

	mustParseAndTypecheck(t, "svc", src)

	if !strings.Contains(src, "type EventStream[T any] struct") {
		t.Fatalf("expected EventStream[T] generated, src:\n%s", src)
	}
	if !strings.Contains(src, `"bufio"`) {
		t.Fatalf("expected bufio import when streaming exists, src:\n%s", src)
	}
	if !strings.Contains(src, `req.Header.Set("Accept", "text/event-stream")`) {
		t.Fatalf("expected SSE Accept header, src:\n%s", src)
	}
}

func TestGenerate_Streaming_NonSSE_GeneratesStub(t *testing.T) {
	svc := &contract.Service{
		Name: "Svc",
		Resources: []*contract.Resource{
			{
				Name: "events",
				Methods: []*contract.Method{
					{
						Name: "subscribe",
						Stream: &contract.MethodStream{
							Mode: "ws",
							Item: "Event",
						},
					},
				},
			},
		},
		Types: []*contract.Type{
			{Name: "Event", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "id", Type: "string"}}},
		},
	}

	files, err := Generate(svc, &Config{Package: "svc"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src := files[0].Content

	mustParseAndTypecheck(t, "svc", src)

	// Expect stub that compiles and is honest about unsupported mode
	if !strings.Contains(src, `stream mode "ws" is not supported`) {
		t.Fatalf("expected non-SSE stream stub, src:\n%s", src)
	}
}

func TestGenerate_Types_StructTags_OptionalPointer_UnknownFallback(t *testing.T) {
	svc := &contract.Service{
		Name: "Svc",
		Resources: []*contract.Resource{
			{Name: "r", Methods: []*contract.Method{{Name: "m"}}},
		},
		Types: []*contract.Type{
			{
				Name: "T",
				Kind: contract.KindStruct,
				Fields: []contract.Field{
					{Name: "name", Type: "string"},
					{Name: "note", Type: "string", Optional: true},   // *string, omitempty
					{Name: "meta", Type: "UnknownType", Optional: true}, // json.RawMessage (pointerized? should not pointerize RawMessage)
				},
			},
		},
	}

	files, err := Generate(svc, &Config{Package: "svc"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src := files[0].Content

	mustParseAndTypecheck(t, "svc", src)

	// JSON tags correct
	if !strings.Contains(src, "`json:\"name\"`") {
		t.Fatalf("expected json tag for name, src:\n%s", src)
	}
	if !strings.Contains(src, "`json:\"note,omitempty\"`") {
		t.Fatalf("expected omitempty for optional note, src:\n%s", src)
	}

	// Optional scalar pointer
	if !strings.Contains(src, "Note *string") {
		t.Fatalf("expected optional scalar to be pointer type, src:\n%s", src)
	}

	// Unknown fallback to json.RawMessage and should not be pointerized
	if !strings.Contains(src, "Meta json.RawMessage") {
		t.Fatalf("expected unknown type fallback to json.RawMessage without pointer, src:\n%s", src)
	}
}

func TestGenerate_Types_SliceAndMapKinds(t *testing.T) {
	svc := &contract.Service{
		Name: "Svc",
		Resources: []*contract.Resource{
			{Name: "r", Methods: []*contract.Method{{Name: "m"}}},
		},
		Types: []*contract.Type{
			{Name: "IDs", Kind: contract.KindSlice, Elem: "string"},
			{Name: "Meta", Kind: contract.KindMap, Elem: "string"},
		},
	}

	files, err := Generate(svc, &Config{Package: "svc"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src := files[0].Content
	mustParseAndTypecheck(t, "svc", src)

	if !strings.Contains(src, "type IDs []string") {
		t.Fatalf("expected slice kind type alias, src:\n%s", src)
	}
	if !strings.Contains(src, "type Meta map[string]string") {
		t.Fatalf("expected map kind type alias, src:\n%s", src)
	}
}

func TestGenerate_Types_UnionDiscriminatorTagCompiles(t *testing.T) {
	svc := &contract.Service{
		Name: "Svc",
		Resources: []*contract.Resource{
			{Name: "r", Methods: []*contract.Method{{Name: "m"}}},
		},
		Types: []*contract.Type{
			{
				Name: "EventUnion",
				Kind: contract.KindUnion,
				Tag:  "type",
				Variants: []contract.Variant{
					{Value: "a", Type: "EventA"},
					{Value: "b", Type: "EventB"},
				},
			},
			{Name: "EventA", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "type", Type: "string", Const: "a"}}},
			{Name: "EventB", Kind: contract.KindStruct, Fields: []contract.Field{{Name: "type", Type: "string", Const: "b"}}},
		},
	}

	files, err := Generate(svc, &Config{Package: "svc"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	src := files[0].Content

	mustParseAndTypecheck(t, "svc", src)

	// Ensure discriminator reader struct uses correct json tag syntax
	if !strings.Contains(src, "`json:\"type\"`") {
		t.Fatalf("expected correct discriminator json tag, src:\n%s", src)
	}
}

// mustParseAndTypecheck parses and type-checks generated source against stdlib.
func mustParseAndTypecheck(t *testing.T, pkg string, src string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "client.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse failed: %v\nsrc:\n%s", err, src)
	}

	// Typecheck
	conf := types.Config{
		Importer: importer.Default(),
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	_, err = conf.Check(pkg, fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("typecheck failed: %v\nsrc:\n%s", err, src)
	}
}
