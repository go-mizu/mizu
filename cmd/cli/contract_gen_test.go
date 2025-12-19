package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	contract "github.com/go-mizu/mizu/contract/v2"
	sdkgo "github.com/go-mizu/mizu/contract/v2/sdk/go"
	sdkpy "github.com/go-mizu/mizu/contract/v2/sdk/py"
	sdkts "github.com/go-mizu/mizu/contract/v2/sdk/ts"
)

// Unit Tests for SDK Generation

func TestContractGen_Client_Go_Generation(t *testing.T) {
	svc := minimalServiceContract()
	files, err := sdkgo.Generate(svc, &sdkgo.Config{Package: "testapi"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if len(files) == 0 {
		t.Fatalf("expected at least 1 file, got 0")
	}

	// Verify client.go is generated
	var clientGo string
	for _, f := range files {
		if f.Path == "client.go" {
			clientGo = f.Content
			break
		}
	}

	if clientGo == "" {
		t.Fatalf("client.go not found in generated files")
	}

	// Verify code parses and typechecks
	mustParseAndTypecheck(t, "testapi", clientGo)

	// Verify key structures
	if !strings.Contains(clientGo, "type Client struct {") {
		t.Errorf("expected Client struct, got:\n%s", clientGo)
	}
	if !strings.Contains(clientGo, "Responses *ResponsesResource") {
		t.Errorf("expected Responses resource field, got:\n%s", clientGo)
	}
	if !strings.Contains(clientGo, "func NewClient(") {
		t.Errorf("expected NewClient function, got:\n%s", clientGo)
	}
}

func TestContractGen_Client_Python_Generation(t *testing.T) {
	svc := minimalServiceContract()
	files, err := sdkpy.Generate(svc, &sdkpy.Config{Package: "testapi", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"pyproject.toml":           false,
		"src/testapi/__init__.py":  false,
		"src/testapi/_client.py":   false,
		"src/testapi/_types.py":    false,
		"src/testapi/_streaming.py": false,
		"src/testapi/_resource.py": false,
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

	// Verify pyproject.toml contains package info
	var pyproject string
	for _, f := range files {
		if f.Path == "pyproject.toml" {
			pyproject = f.Content
			break
		}
	}
	if !strings.Contains(pyproject, `name = "testapi"`) {
		t.Errorf("expected package name testapi, got:\n%s", pyproject)
	}
	if !strings.Contains(pyproject, `version = "0.0.1"`) {
		t.Errorf("expected version 0.0.1, got:\n%s", pyproject)
	}
}

func TestContractGen_Client_TypeScript_Generation(t *testing.T) {
	svc := minimalServiceContract()
	files, err := sdkts.Generate(svc, &sdkts.Config{Package: "testapi", Version: "0.0.1"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expected := map[string]bool{
		"package.json":       false,
		"tsconfig.json":      false,
		"src/index.ts":       false,
		"src/_client.ts":     false,
		"src/_types.ts":      false,
		"src/_streaming.ts":  false,
		"src/_resources.ts":  false,
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

	// Verify package.json contains package info
	var pkgJSON string
	for _, f := range files {
		if f.Path == "package.json" {
			pkgJSON = f.Content
			break
		}
	}
	if !strings.Contains(pkgJSON, `"name": "testapi"`) {
		t.Errorf("expected package name testapi, got:\n%s", pkgJSON)
	}
	if !strings.Contains(pkgJSON, `"version": "0.0.1"`) {
		t.Errorf("expected version 0.0.1, got:\n%s", pkgJSON)
	}
}

func TestContractGen_NormalizeLang(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go", "go"},
		{"golang", "go"},
		{"GO", "go"},
		{"python", "python"},
		{"py", "python"},
		{"Python", "python"},
		{"typescript", "typescript"},
		{"ts", "typescript"},
		{"TypeScript", "typescript"},
		{"all", "all"},
		{"ALL", "all"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeLang(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeLang(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// E2E Tests for Go SDK

func TestContractGen_Go_E2E_HTTP_Request(t *testing.T) {
	if !goE2EEnabled() {
		t.Skip("SDKGO_E2E not enabled")
	}

	var got struct {
		Method string
		Path   string
		Auth   string
		CT     string
		Body   map[string]any
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		got.Method = r.Method
		got.Path = r.URL.Path
		got.Auth = r.Header.Get("Authorization")
		got.CT = r.Header.Get("Content-Type")

		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got.Body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"r1"}`))
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract()
	root := writeGeneratedGoSDK(t, svc)

	mainGo := `
package main

import (
	"context"
	"fmt"
	"testapi"
)

func main() {
	client := testapi.NewClient("sk-test", testapi.WithBaseURL("` + srv.URL + `"))
	resp, err := client.Responses.Create(context.Background(), &testapi.CreateRequest{
		Model: "gpt-test",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.ID)
}
`

	out := runGoProgram(t, root, mainGo)
	if strings.TrimSpace(out) != "r1" {
		t.Fatalf("expected r1, got %q", out)
	}

	if got.Method != "POST" {
		t.Fatalf("expected POST, got %s", got.Method)
	}
	if got.Path != "/v1/responses" {
		t.Fatalf("expected /v1/responses, got %s", got.Path)
	}
	if got.Auth != "Bearer sk-test" {
		t.Fatalf("expected Authorization Bearer, got %q", got.Auth)
	}
}

func TestContractGen_Go_E2E_SSE_Stream(t *testing.T) {
	if !goE2EEnabled() {
		t.Skip("SDKGO_E2E not enabled")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", 500)
			return
		}

		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text\",\"text\":\"a\"}\n\n")
		fl.Flush()
		time.Sleep(10 * time.Millisecond)

		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text\",\"text\":\"b\"}\n\n")
		fl.Flush()
		time.Sleep(10 * time.Millisecond)

		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract()
	root := writeGeneratedGoSDK(t, svc)

	mainGo := `
package main

import (
	"context"
	"fmt"
	"testapi"
)

func main() {
	client := testapi.NewClient("sk-test", testapi.WithBaseURL("` + srv.URL + `"))
	stream := client.Responses.Stream(context.Background(), &testapi.CreateRequest{Model: "gpt-test"})
	defer stream.Close()

	n := 0
	var texts []string
	for stream.Next() {
		ev := stream.Event()
		n++
		if ev.Text != nil {
			texts = append(texts, *ev.Text)
		}
	}
	if err := stream.Err(); err != nil {
		panic(err)
	}

	fmt.Println(n)
	for _, t := range texts {
		fmt.Print(t)
	}
	fmt.Println()
}
`

	out := runGoProgram(t, root, mainGo)
	lines := splitNonEmptyLines(out)
	if len(lines) < 2 {
		t.Fatalf("unexpected output:\n%s", out)
	}
	if lines[0] != "2" {
		t.Fatalf("expected 2 events, got %q\nfull:\n%s", lines[0], out)
	}
	if lines[1] != "ab" {
		t.Fatalf("expected texts ab, got %q\nfull:\n%s", lines[1], out)
	}
}

// E2E Tests for Python SDK

func TestContractGen_Python_E2E_HTTP_Request(t *testing.T) {
	if !pyE2EEnabled() {
		t.Skip("SDKPY_E2E not enabled")
	}
	requireUV(t)
	ensurePythonViaUV(t)

	var got struct {
		Method string
		Path   string
		Auth   string
		CT     string
		Body   map[string]any
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		got.Method = r.Method
		got.Path = r.URL.Path
		got.Auth = r.Header.Get("Authorization")
		got.CT = r.Header.Get("Content-Type")

		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got.Body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"r1"}`))
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract()
	root := writeGeneratedPySDK(t, svc)
	runUV(t, root, "pip", "install", "-e", ".")

	script := `
from openai import OpenAI

client = OpenAI(api_key="sk-test", base_url="` + srv.URL + `")
resp = client.responses.create({"model": "gpt-test"})
print(resp.id)
`
	scriptPath := filepath.Join(root, "test_http.py")
	mustWriteFile(t, scriptPath, []byte(script))

	out := runUV(t, root, "run", "python", scriptPath)
	if strings.TrimSpace(out) != "r1" {
		t.Fatalf("expected r1, got %q", out)
	}

	if got.Method != "POST" {
		t.Fatalf("expected POST, got %s", got.Method)
	}
	if got.Path != "/v1/responses" {
		t.Fatalf("expected /v1/responses, got %s", got.Path)
	}
	if got.Auth != "Bearer sk-test" {
		t.Fatalf("expected Authorization Bearer, got %q", got.Auth)
	}
}

func TestContractGen_Python_E2E_SSE_Stream(t *testing.T) {
	if !pyE2EEnabled() {
		t.Skip("SDKPY_E2E not enabled")
	}
	requireUV(t)
	ensurePythonViaUV(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", 500)
			return
		}

		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text\",\"text\":\"a\"}\n\n")
		fl.Flush()
		time.Sleep(10 * time.Millisecond)

		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text\",\"text\":\"b\"}\n\n")
		fl.Flush()
		time.Sleep(10 * time.Millisecond)

		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract()
	root := writeGeneratedPySDK(t, svc)
	runUV(t, root, "pip", "install", "-e", ".")

	script := `
from openai import OpenAI

client = OpenAI(api_key="sk-test", base_url="` + srv.URL + `")

n = 0
texts = []

for ev in client.responses.stream({"model":"gpt-test"}):
    n += 1
    t = None
    if hasattr(ev, "text"):
        t = ev.text
    elif isinstance(ev, dict):
        t = ev.get("text")
    if t is not None:
        texts.append(t)

print(n)
print("".join(texts))
`
	scriptPath := filepath.Join(root, "test_stream.py")
	mustWriteFile(t, scriptPath, []byte(script))

	out := runUV(t, root, "run", "python", scriptPath)
	lines := splitNonEmptyLines(out)
	if len(lines) < 2 {
		t.Fatalf("unexpected output:\n%s", out)
	}
	if lines[0] != "2" {
		t.Fatalf("expected 2 events, got %q\nfull:\n%s", lines[0], out)
	}
	if lines[1] != "ab" {
		t.Fatalf("expected texts ab, got %q\nfull:\n%s", lines[1], out)
	}
}

// E2E Tests for TypeScript SDK

func TestContractGen_TypeScript_E2E_Node_HTTP_Request(t *testing.T) {
	if !tsE2EEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireNode(t)

	var got struct {
		Method string
		Path   string
		Auth   string
		CT     string
		Body   map[string]any
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		got.Method = r.Method
		got.Path = r.URL.Path
		got.Auth = r.Header.Get("Authorization")
		got.CT = r.Header.Get("Content-Type")

		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got.Body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"r1"}`))
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract()
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';

const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });
const out = await client.responses.create({ model: 'gpt-test' });
console.log(out.id);
`
	out := runNode(t, root, script)
	if strings.TrimSpace(out) != "r1" {
		t.Fatalf("expected r1, got %q", out)
	}

	if got.Method != "POST" {
		t.Fatalf("expected POST, got %s", got.Method)
	}
	if got.Path != "/v1/responses" {
		t.Fatalf("expected /v1/responses, got %s", got.Path)
	}
	if got.Auth != "Bearer sk-test" {
		t.Fatalf("expected Authorization Bearer, got %q", got.Auth)
	}
}

func TestContractGen_TypeScript_E2E_Node_SSE_Stream(t *testing.T) {
	if !tsE2EEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireNode(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", 500)
			return
		}

		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text\",\"text\":\"a\"}\n\n")
		fl.Flush()
		time.Sleep(10 * time.Millisecond)

		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text\",\"text\":\"b\"}\n\n")
		fl.Flush()
		time.Sleep(10 * time.Millisecond)

		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract()
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';

const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });

let n = 0;
const texts: string[] = [];

for await (const ev of client.responses.stream({ model: 'gpt-test' })) {
  n++;
  if (ev.text) {
    texts.push(ev.text);
  }
}

console.log(n);
console.log(texts.join(''));
`
	out := runNode(t, root, script)
	lines := splitNonEmptyLines(out)
	if len(lines) < 2 {
		t.Fatalf("unexpected output:\n%s", out)
	}
	if lines[0] != "2" {
		t.Fatalf("expected 2 events, got %q\nfull:\n%s", lines[0], out)
	}
	if lines[1] != "ab" {
		t.Fatalf("expected texts ab, got %q\nfull:\n%s", lines[1], out)
	}
}

func TestContractGen_TypeScript_E2E_Bun_HTTP_Request(t *testing.T) {
	if !tsE2EEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireBun(t)

	var got struct {
		Method string
		Path   string
		Auth   string
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		got.Method = r.Method
		got.Path = r.URL.Path
		got.Auth = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"r1"}`))
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract()
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';

const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });
const out = await client.responses.create({ model: 'gpt-test' });
console.log(out.id);
`
	out := runBun(t, root, script)
	if strings.TrimSpace(out) != "r1" {
		t.Fatalf("expected r1, got %q", out)
	}

	if got.Method != "POST" {
		t.Fatalf("expected POST, got %s", got.Method)
	}
	if got.Path != "/v1/responses" {
		t.Fatalf("expected /v1/responses, got %s", got.Path)
	}
	if got.Auth != "Bearer sk-test" {
		t.Fatalf("expected Authorization Bearer, got %q", got.Auth)
	}
}

// Helper functions

func minimalServiceContract() *contract.Service {
	return &contract.Service{
		Name: "OpenAI",
		Client: &contract.Client{
			Auth: "bearer",
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
			// Note: Error type is auto-generated by SDK generators, so we don't include it here
		},
	}
}

func mustParseAndTypecheck(t *testing.T, pkg string, src string) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "client.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse failed: %v\nsource:\n%s", err, src)
	}

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	_, err = conf.Check(pkg, fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("typecheck failed: %v\nsource:\n%s", err, src)
	}
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

func writeGeneratedGoSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkgo.Config{Package: "testapi"}
	files, err := sdkgo.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "go-sdk")
	for _, f := range files {
		if f == nil {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(f.Path))
		mustWriteFile(t, p, []byte(f.Content))
	}

	// Create go.mod
	goMod := "module testapi\n\ngo 1.22\n"
	mustWriteFile(t, filepath.Join(root, "go.mod"), []byte(goMod))

	return root
}

func writeGeneratedPySDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkpy.Config{
		Package: "openai",
		Version: "0.0.0.dev0",
	}
	files, err := sdkpy.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "py-sdk")
	for _, f := range files {
		if f == nil {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(f.Path))
		mustWriteFile(t, p, []byte(f.Content))
	}

	// Create virtual environment
	createVenv(t, root)

	return root
}

func writeGeneratedTSSDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkts.Config{
		Package: "openai",
		Version: "0.0.0",
	}
	files, err := sdkts.Generate(svc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("Generate returned no files")
	}

	root := filepath.Join(t.TempDir(), "ts-sdk")
	for _, f := range files {
		if f == nil {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(f.Path))
		mustWriteFile(t, p, []byte(f.Content))
	}

	return root
}

func createVenv(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("uv", "venv")
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("uv venv failed: %v\nstderr:\n%s", err, stderr.String())
	}
}

func runGoProgram(t *testing.T, sdkDir string, mainGo string) string {
	t.Helper()

	// Create a cmd directory for the main program to avoid package conflicts
	cmdDir := filepath.Join(sdkDir, "cmd", "test")
	mainPath := filepath.Join(cmdDir, "main.go")
	mustWriteFile(t, mainPath, []byte(mainGo))

	// Update go.mod to use the local SDK
	goMod := `module testcmd

go 1.22

require testapi v0.0.0

replace testapi => ` + sdkDir + `
`
	mustWriteFile(t, filepath.Join(cmdDir, "go.mod"), []byte(goMod))

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = cmdDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("go run failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}
	return stdout.String()
}

func runUV(t *testing.T, dir string, args ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if isNonFatalE2E(err, stderr.String()) && !pyE2EStrict() {
			t.Skipf("uv command failed in non-strict mode: uv %s\nerr: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
		}
		t.Fatalf("uv %s failed: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
	}
	if stderr.Len() > 0 {
		t.Logf("uv %s stderr:\n%s", strings.Join(args, " "), stderr.String())
	}
	return stdout.String()
}

func runNode(t *testing.T, dir string, script string) string {
	t.Helper()

	scriptPath := filepath.Join(dir, "test_script.ts")
	mustWriteFile(t, scriptPath, []byte(script))

	cmd := exec.Command("npx", "tsx", scriptPath)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if isNonFatalE2E(err, stderr.String()) && !tsE2EStrict() {
			t.Skipf("node/tsx command failed in non-strict mode: %v\nstderr:\n%s", err, stderr.String())
		}
		t.Fatalf("node/tsx failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}
	if stderr.Len() > 0 {
		t.Logf("node/tsx stderr:\n%s", stderr.String())
	}
	return stdout.String()
}

func runBun(t *testing.T, dir string, script string) string {
	t.Helper()

	scriptPath := filepath.Join(dir, "test_script.ts")
	mustWriteFile(t, scriptPath, []byte(script))

	cmd := exec.Command("bun", "run", scriptPath)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if isNonFatalE2E(err, stderr.String()) && !tsE2EStrict() {
			t.Skipf("bun command failed in non-strict mode: %v\nstderr:\n%s", err, stderr.String())
		}
		t.Fatalf("bun failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}
	if stderr.Len() > 0 {
		t.Logf("bun stderr:\n%s", stderr.String())
	}
	return stdout.String()
}

func requireUV(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not installed")
	}
	if runtime.GOOS == "windows" && !pyE2EStrict() {
		t.Skip("windows e2e disabled unless SDKPY_E2E=strict")
	}
}

func ensurePythonViaUV(t *testing.T) {
	t.Helper()

	cmd := exec.Command("uv", "run", "python", "-c", "import sys; print(sys.version.split()[0])")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if !pyE2EStrict() {
			t.Skipf("python via uv is not available (non-strict): %v\nstderr:\n%s", err, stderr.String())
		}
		t.Fatalf("python via uv is not available: %v\nstderr:\n%s", err, stderr.String())
	}

	t.Logf("uv python version: %s", strings.TrimSpace(stdout.String()))
}

func requireNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not installed")
	}
	// Check Node version >= 18 (for native fetch)
	cmd := exec.Command("node", "--version")
	out, err := cmd.Output()
	if err != nil {
		t.Skip("cannot determine node version")
	}
	version := strings.TrimSpace(string(out))
	if !strings.HasPrefix(version, "v") {
		t.Skip("cannot parse node version")
	}
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) < 1 {
		t.Skip("cannot parse node version")
	}
	var major int
	for _, c := range parts[0] {
		if c >= '0' && c <= '9' {
			major = major*10 + int(c-'0')
		} else {
			break
		}
	}
	if major < 18 {
		t.Skipf("node version %s is less than 18, skipping (native fetch required)", version)
	}
	// Check tsx is available
	cmd = exec.Command("npx", "tsx", "--version")
	if err := cmd.Run(); err != nil {
		t.Skip("tsx not available (install with: npm install -g tsx)")
	}
	if runtime.GOOS == "windows" && !tsE2EStrict() {
		t.Skip("windows e2e disabled unless SDKTS_E2E=strict")
	}
}

func requireBun(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("bun not installed")
	}
	if runtime.GOOS == "windows" && !tsE2EStrict() {
		t.Skip("windows e2e disabled unless SDKTS_E2E=strict")
	}
}

// Environment variable helpers

func goE2EEnabled() bool {
	v := strings.TrimSpace(os.Getenv("SDKGO_E2E"))
	if v == "" {
		return false
	}
	v = strings.ToLower(v)
	if v == "0" || v == "false" || v == "no" || v == "off" {
		return false
	}
	return true
}

func pyE2EEnabled() bool {
	v := strings.TrimSpace(os.Getenv("SDKPY_E2E"))
	if v == "" {
		return false
	}
	v = strings.ToLower(v)
	if v == "0" || v == "false" || v == "no" || v == "off" {
		return false
	}
	return true
}

func tsE2EEnabled() bool {
	v := strings.TrimSpace(os.Getenv("SDKTS_E2E"))
	if v == "" {
		return false
	}
	v = strings.ToLower(v)
	if v == "0" || v == "false" || v == "no" || v == "off" {
		return false
	}
	return true
}

func pyE2EStrict() bool {
	v := strings.TrimSpace(os.Getenv("SDKPY_E2E"))
	return strings.EqualFold(v, "strict")
}

func tsE2EStrict() bool {
	v := strings.TrimSpace(os.Getenv("SDKTS_E2E"))
	return strings.EqualFold(v, "strict")
}

func isNonFatalE2E(err error, stderr string) bool {
	s := strings.ToLower(stderr)
	if strings.Contains(s, "network") && strings.Contains(s, "error") {
		return true
	}
	if strings.Contains(s, "connection") && strings.Contains(s, "refused") {
		return true
	}
	if strings.Contains(s, "timeout") {
		return true
	}
	if strings.Contains(s, "no matching distribution found") {
		return true
	}
	if strings.Contains(s, "failed to resolve") || strings.Contains(s, "resolution failed") {
		return true
	}
	if strings.Contains(s, "ssl") && strings.Contains(s, "error") {
		return true
	}
	return false
}

func splitNonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
