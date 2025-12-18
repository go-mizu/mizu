package sdkts_test

import (
	"bytes"
	"encoding/json"
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
	"github.com/go-mizu/mizu/contract/v2/sdk"
	sdkts "github.com/go-mizu/mizu/contract/v2/sdk/ts"
)

func TestGenerate_NilService(t *testing.T) {
	_, err := sdkts.Generate(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil service")
	}
}

func TestGenerate_ProducesExpectedFiles(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkts.Generate(svc, &sdkts.Config{Package: "openai", Version: "0.0.1"})
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
}

func TestGenerate_PackageJSON_ContainsConfig(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkts.Generate(svc, &sdkts.Config{Package: "my-sdk", Version: "1.2.3"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var pkgJSON string
	for _, f := range files {
		if f.Path == "package.json" {
			pkgJSON = f.Content
			break
		}
	}

	if !strings.Contains(pkgJSON, `"name": "my-sdk"`) {
		t.Errorf("package.json should contain package name, got:\n%s", pkgJSON)
	}
	if !strings.Contains(pkgJSON, `"version": "1.2.3"`) {
		t.Errorf("package.json should contain version, got:\n%s", pkgJSON)
	}
}

func TestGenerate_TypesFile_ContainsInterfaces(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkts.Generate(svc, &sdkts.Config{Package: "openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var typesTS string
	for _, f := range files {
		if f.Path == "src/_types.ts" {
			typesTS = f.Content
			break
		}
	}

	if !strings.Contains(typesTS, "export interface CreateRequest") {
		t.Errorf("_types.ts should contain CreateRequest interface, got:\n%s", typesTS)
	}
	if !strings.Contains(typesTS, "export interface Response") {
		t.Errorf("_types.ts should contain Response interface, got:\n%s", typesTS)
	}
}

func TestGenerate_ClientFile_ContainsClientClass(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkts.Generate(svc, &sdkts.Config{Package: "openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var clientTS string
	for _, f := range files {
		if f.Path == "src/_client.ts" {
			clientTS = f.Content
			break
		}
	}

	if !strings.Contains(clientTS, "export class OpenAI") {
		t.Errorf("_client.ts should contain OpenAI class, got:\n%s", clientTS)
	}
	if !strings.Contains(clientTS, "readonly responses:") {
		t.Errorf("_client.ts should contain responses resource, got:\n%s", clientTS)
	}
}

func TestGenerate_ResourcesFile_ContainsMethods(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkts.Generate(svc, &sdkts.Config{Package: "openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesTS string
	for _, f := range files {
		if f.Path == "src/_resources.ts" {
			resourcesTS = f.Content
			break
		}
	}

	if !strings.Contains(resourcesTS, "export class ResponsesResource") {
		t.Errorf("_resources.ts should contain ResponsesResource class, got:\n%s", resourcesTS)
	}
	if !strings.Contains(resourcesTS, "async create(request: CreateRequest): Promise<Response>") {
		t.Errorf("_resources.ts should contain create method, got:\n%s", resourcesTS)
	}
}

func TestGenerate_DefaultPackageName(t *testing.T) {
	svc := &contract.Service{
		Name: "My API!! v2",
		Resources: []*contract.Resource{
			{Name: "ping", Methods: []*contract.Method{{Name: "do"}}},
		},
	}

	files, err := sdkts.Generate(svc, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var pkgJSON string
	for _, f := range files {
		if f.Path == "package.json" {
			pkgJSON = f.Content
			break
		}
	}

	// sanitizeIdent keeps letters/digits/underscore, then lowercased
	// "My API!! v2" -> "MyAPIv2" -> "myapiv2"
	if !strings.Contains(pkgJSON, `"name": "myapiv2"`) {
		t.Errorf("expected package name myapiv2, got:\n%s", pkgJSON)
	}
}

func TestGenerate_StreamingMethod(t *testing.T) {
	svc := minimalServiceContract(t)
	files, err := sdkts.Generate(svc, &sdkts.Config{Package: "openai"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var resourcesTS string
	for _, f := range files {
		if f.Path == "src/_resources.ts" {
			resourcesTS = f.Content
			break
		}
	}

	// Should have streaming method
	if !strings.Contains(resourcesTS, "stream(request: CreateRequest): Stream<ResponseEvent>") {
		t.Errorf("_resources.ts should contain stream method, got:\n%s", resourcesTS)
	}
}

// E2E Tests - require SDKTS_E2E env var

func TestTSSDK_E2E_Node_ImportAndInit(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireNode(t)

	svc := minimalServiceContract(t)
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';
const c = new OpenAI({ apiKey: 'sk-test', baseURL: 'http://example.invalid' });
console.log('OK');
`
	out := runNode(t, root, script)
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTSSDK_E2E_Node_HTTP_RequestShape(t *testing.T) {
	if !e2eEnabled() {
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

	svc := minimalServiceContract(t)
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';

const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });
const out = await client.responses.create({ model: 'gpt-test', input: { x: 1 } });
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
	if !strings.HasPrefix(got.CT, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", got.CT)
	}
	if got.Body["model"] != "gpt-test" {
		t.Fatalf("expected model gpt-test, got %#v", got.Body["model"])
	}
}

func TestTSSDK_E2E_Node_ErrorDecoding(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireNode(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"code":"bad_request","message":"nope"}`))
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract(t)
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI, APIStatusError } from './src/index.ts';

const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });
try {
  await client.responses.create({ model: 'gpt-test' });
  console.log('NOERROR');
} catch (e) {
  const s = String(e);
  const ok = s.includes('nope') || s.includes('400') || s.includes('bad_request');
  console.log(ok ? 'OK' : ('BAD:' + s));
}
`
	out := runNode(t, root, script)
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("expected OK, got %q", out)
	}
}

func TestTSSDK_E2E_Node_SSE_Stream(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireNode(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			http.NotFound(w, r)
			return
		}
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

	svc := minimalServiceContract(t)
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

func TestTSSDK_E2E_Bun_ImportAndInit(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireBun(t)

	svc := minimalServiceContract(t)
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';
const c = new OpenAI({ apiKey: 'sk-test', baseURL: 'http://example.invalid' });
console.log('OK');
`
	out := runBun(t, root, script)
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTSSDK_E2E_Bun_HTTP_RequestShape(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireBun(t)

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

	svc := minimalServiceContract(t)
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';

const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });
const out = await client.responses.create({ model: 'gpt-test', input: { x: 1 } });
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

func TestTSSDK_E2E_Bun_SSE_Stream(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireBun(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			http.NotFound(w, r)
			return
		}
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

	svc := minimalServiceContract(t)
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
	out := runBun(t, root, script)
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

func TestTSSDK_E2E_Deno_ImportAndInit(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireDeno(t)

	svc := minimalServiceContract(t)
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';
const c = new OpenAI({ apiKey: 'sk-test', baseURL: 'http://example.invalid' });
console.log('OK');
`
	out := runDeno(t, root, script)
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTSSDK_E2E_Deno_HTTP_RequestShape(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireDeno(t)

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

	svc := minimalServiceContract(t)
	root := writeGeneratedTSSDK(t, svc)

	script := `
import { OpenAI } from './src/index.ts';

const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });
const out = await client.responses.create({ model: 'gpt-test', input: { x: 1 } });
console.log(out.id);
`
	out := runDeno(t, root, script)
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

func TestTSSDK_E2E_Deno_SSE_Stream(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKTS_E2E not enabled")
	}
	requireDeno(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			http.NotFound(w, r)
			return
		}
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

	svc := minimalServiceContract(t)
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
	out := runDeno(t, root, script)
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

// Helper functions

func minimalServiceContract(t *testing.T) *contract.Service {
	t.Helper()
	return &contract.Service{
		Name: "OpenAI",
		Defaults: &contract.Defaults{
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

func mustWriteFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func runNode(t *testing.T, dir string, script string) string {
	t.Helper()

	scriptPath := filepath.Join(dir, "test_script.ts")
	mustWriteFile(t, scriptPath, []byte(script))

	// Use npx tsx to run TypeScript directly
	cmd := exec.Command("npx", "tsx", scriptPath)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if isNonFatalE2E(err, stderr.String()) && !e2eStrict() {
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
		if isNonFatalE2E(err, stderr.String()) && !e2eStrict() {
			t.Skipf("bun command failed in non-strict mode: %v\nstderr:\n%s", err, stderr.String())
		}
		t.Fatalf("bun failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}
	if stderr.Len() > 0 {
		t.Logf("bun stderr:\n%s", stderr.String())
	}
	return stdout.String()
}

func runDeno(t *testing.T, dir string, script string) string {
	t.Helper()

	scriptPath := filepath.Join(dir, "test_script.ts")
	mustWriteFile(t, scriptPath, []byte(script))

	cmd := exec.Command("deno", "run", "--allow-net", scriptPath)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if isNonFatalE2E(err, stderr.String()) && !e2eStrict() {
			t.Skipf("deno command failed in non-strict mode: %v\nstderr:\n%s", err, stderr.String())
		}
		t.Fatalf("deno failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}
	if stderr.Len() > 0 {
		t.Logf("deno stderr:\n%s", stderr.String())
	}
	return stdout.String()
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
	// Parse major version
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
	if runtime.GOOS == "windows" && !e2eStrict() {
		t.Skip("windows e2e disabled unless SDKTS_E2E=strict")
	}
}

func requireBun(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("bun not installed")
	}
	if runtime.GOOS == "windows" && !e2eStrict() {
		t.Skip("windows e2e disabled unless SDKTS_E2E=strict")
	}
}

func requireDeno(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("deno"); err != nil {
		t.Skip("deno not installed")
	}
	if runtime.GOOS == "windows" && !e2eStrict() {
		t.Skip("windows e2e disabled unless SDKTS_E2E=strict")
	}
}

func e2eEnabled() bool {
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

func e2eStrict() bool {
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

var _ = sdk.File{}
