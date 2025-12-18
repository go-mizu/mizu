package sdkpy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	sdkpy "github.com/go-mizu/mizu/contract/v2/sdk/py"
)

func TestPySDK_E2E_Smoke_ImportAndInit(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKPY_E2E not enabled")
	}
	requireUV(t)
	ensurePythonViaUV(t)

	svc := minimalServiceContract(t)
	root := writeGeneratedPySDK(t, svc)

	runUV(t, root, "pip", "install", "-e", ".")

	pkg := "openai"
	script := `
from ` + pkg + ` import OpenAI
c = OpenAI(api_key="sk-test", base_url="http://example.invalid")
print("OK")
`
	out := runUV(t, root, "run", "python", "-c", script)
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestPySDK_E2E_HTTP_RequestShape_Sync(t *testing.T) {
	if !e2eEnabled() {
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

	svc := minimalServiceContract(t)
	root := writeGeneratedPySDK(t, svc)

	runUV(t, root, "pip", "install", "-e", ".")

	pkg := "openai"
	scriptPath := filepath.Join(root, "smoke_sync.py")
	mustWriteFile(t, scriptPath, []byte(`
from `+pkg+` import OpenAI

client = OpenAI(api_key="sk-test", base_url="`+srv.URL+`")
out = client.responses.create({"model":"gpt-test","input":{"x":1}})
print(out.id)
`))

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
	if !strings.HasPrefix(got.CT, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", got.CT)
	}
	if got.Body["model"] != "gpt-test" {
		t.Fatalf("expected model gpt-test, got %#v", got.Body["model"])
	}
}

func TestPySDK_E2E_ErrorDecoding_Sync(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKPY_E2E not enabled")
	}
	requireUV(t)
	ensurePythonViaUV(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"code":"bad_request","message":"nope"}`))
	}))
	t.Cleanup(srv.Close)

	svc := minimalServiceContract(t)
	root := writeGeneratedPySDK(t, svc)

	runUV(t, root, "pip", "install", "-e", ".")

	pkg := "openai"
	scriptPath := filepath.Join(root, "smoke_error.py")
	mustWriteFile(t, scriptPath, []byte(`
from `+pkg+` import OpenAI

client = OpenAI(api_key="sk-test", base_url="`+srv.URL+`")
try:
    client.responses.create({"model":"gpt-test"})
    print("NOERROR")
except Exception as e:
    s = str(e)
    ok = ("bad_request" in s) or ("nope" in s) or ("400" in s)
    print("OK" if ok else ("BAD:" + s))
`))

	out := runUV(t, root, "run", "python", scriptPath)
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("expected OK, got %q", out)
	}
}

func TestPySDK_E2E_SSE_Stream_Sync(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKPY_E2E not enabled")
	}
	requireUV(t)
	ensurePythonViaUV(t)

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
	root := writeGeneratedPySDK(t, svc)
	runUV(t, root, "pip", "install", "-e", ".")

	pkg := "openai"
	scriptPath := filepath.Join(root, "smoke_sse.py")
	mustWriteFile(t, scriptPath, []byte(`
from `+pkg+` import OpenAI

client = OpenAI(api_key="sk-test", base_url="`+srv.URL+`")

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
`))

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

func TestPySDK_E2E_AsyncClient_Basic(t *testing.T) {
	if !e2eEnabled() {
		t.Skip("SDKPY_E2E not enabled")
	}
	requireUV(t)
	ensurePythonViaUV(t)

	svc := minimalServiceContract(t)
	root := writeGeneratedPySDK(t, svc)
	runUV(t, root, "pip", "install", "-e", ".")

	pkg := "openai"
	script := `
import asyncio
from ` + pkg + ` import AsyncOpenAI

async def main():
    c = AsyncOpenAI(api_key="sk-test", base_url="http://example.invalid")
    await c.close()
    print("OK")

asyncio.run(main())
`
	out := runUV(t, root, "run", "python", "-c", script)
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("unexpected output: %q", out)
	}
}

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
						Stream: &struct {
							Mode      string           `json:"mode,omitempty" yaml:"mode,omitempty"`
							Item      contract.TypeRef `json:"item" yaml:"item"`
							Done      contract.TypeRef `json:"done,omitempty" yaml:"done,omitempty"`
							Error     contract.TypeRef `json:"error,omitempty" yaml:"error,omitempty"`
							InputItem contract.TypeRef `json:"input_item,omitempty" yaml:"input_item,omitempty"`
						}{
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

func writeGeneratedPySDK(t *testing.T, svc *contract.Service) string {
	t.Helper()

	cfg := &sdkpy.Config{
		Package: "openai",
		Version: "0.0.0-test",
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

func runUV(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("uv", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if isNonFatalE2E(err, stderr.String()) && !e2eStrict() {
			t.Skipf("uv command failed in non-strict mode: uv %s\nerr: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
		}
		t.Fatalf("uv %s failed: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
	}
	if stderr.Len() > 0 {
		t.Logf("uv %s stderr:\n%s", strings.Join(args, " "), stderr.String())
	}
	return stdout.String()
}

func requireUV(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not installed")
	}
	if runtime.GOOS == "windows" && !e2eStrict() {
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
		if !e2eStrict() {
			t.Skipf("python via uv is not available (non-strict): %v\nstderr:\n%s", err, stderr.String())
		}
		t.Fatalf("python via uv is not available: %v\nstderr:\n%s", err, stderr.String())
	}

	t.Logf("uv python version: %s", strings.TrimSpace(stdout.String()))
}

func e2eEnabled() bool {
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

func e2eStrict() bool {
	v := strings.TrimSpace(os.Getenv("SDKPY_E2E"))
	return strings.EqualFold(v, "strict")
}

func isNonFatalE2E(err error, stderr string) bool {
	s := strings.ToLower(stderr)
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if strings.Contains(s, "no matching distribution found") {
		return true
	}
	if strings.Contains(s, "failed to resolve") || strings.Contains(s, "resolution failed") {
		return true
	}
	if strings.Contains(s, "network") && strings.Contains(s, "error") {
		return true
	}
	if strings.Contains(s, "ssl") && strings.Contains(s, "error") {
		return true
	}
	if strings.Contains(s, "connection") && strings.Contains(s, "refused") {
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
