# Gemini Embedding 2 Driver Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a `gemini` embed driver (`pkg/embed/driver/gemini`) using the Gemini Embedding 2 API (`text-embedding-004` / `gemini-embedding-exp-03-07`), wire it into the `search cc fts embed` CLI, write the spec, and test end-to-end.

**Architecture:** The driver follows the exact same `embed.Driver` interface as `llamacpp` — `Open`, `Close`, `Name`, `Dimension`, `Embed`. `Open` loads `GEMINI_API_KEY` from env (with fallback to `$HOME/data/.local.env`), sends a probe request to discover dim. `Embed` calls `batchEmbedContents` (max 100 per batch). CLI imports the driver via blank import and updates docs.

**Tech Stack:** Go 1.26, `net/http`, Gemini REST API v1beta (`generativelanguage.googleapis.com`), `net/http/httptest` for unit tests.

---

## Chunk 1: Core driver + unit tests

### Task 1: Write the spec document

**Files:**
- Create: `spec/0713_gemini_embedding_2.md`

- [ ] **Step 1: Create spec file**

```markdown
# Gemini Embedding 2 — `pkg/embed/driver/gemini`

## Background

The existing `embed` package supports two drivers: `llamacpp` (HTTP to a local server)
and `onnx` (local ONNX Runtime inference). Both require local infrastructure. This spec
adds a third driver, `gemini`, that calls the Google Gemini Embedding API — zero local
infrastructure, just a `GEMINI_API_KEY`.

## Models

| Model | Dim | Notes |
|---|---|---|
| `text-embedding-004` | 768 | Stable, multilingual, free tier 1500 RPM |
| `gemini-embedding-exp-03-07` | 3072 | Matryoshka — supports `outputDimensionality` |

## API

**Endpoint:** `POST https://generativelanguage.googleapis.com/v1beta/models/{model}:batchEmbedContents?key={apiKey}`

**Request:**
```json
{
  "requests": [
    {
      "model": "models/text-embedding-004",
      "content": { "parts": [{ "text": "..." }] },
      "taskType": "RETRIEVAL_DOCUMENT"
    }
  ]
}
```

**Response:**
```json
{
  "embeddings": [
    { "values": [0.01, 0.02, ...] }
  ]
}
```

**Batch limit:** 100 requests per call.

## API Key Loading

Priority order in `Open()`:
1. `cfg.Addr` field (allows passing key directly for tests / override)
2. `os.Getenv("GEMINI_API_KEY")`
3. Parse `$HOME/data/.local.env` for `export GEMINI_API_KEY="..."` or `GEMINI_API_KEY=...`

Error message when not found:
```
gemini: GEMINI_API_KEY not set
  Set it in the environment:  export GEMINI_API_KEY=<key>
  Or add it to $HOME/data/.local.env
```

## Driver Config

Uses the standard `embed.Config`:
- `cfg.Addr` — API key override (takes priority over env)
- `cfg.Model` — model name (default: `text-embedding-004`)
- `cfg.BatchSize` — max per batch call (default/max: 100)
- `cfg.Dir` — unused (API model, no local files)

## Matryoshka Dimension

For `gemini-embedding-exp-03-07`, `outputDimensionality` in the request controls the
output size. Default is 3072. To request a smaller embedding, append `:NNN` to the
model name: `gemini-embedding-exp-03-07:768`. The driver parses this suffix.

## Output

Same as all other drivers: `vectors.bin`, `meta.jsonl`, `stats.json` written by the
existing `embedDir` pipeline in `cli/cc_fts_embed.go`. No changes to the pipeline itself.

## CLI Usage

```bash
# Default (text-embedding-004, 768-dim)
search cc fts embed run --input ./docs/ --driver gemini

# Gemini Embedding 2 full 3072-dim
search cc fts embed run --input ./docs/ --driver gemini --model gemini-embedding-exp-03-07

# Reduced Matryoshka dim (768)
search cc fts embed run --input ./docs/ --driver gemini --model gemini-embedding-exp-03-07:768

# List models
search cc fts embed models
```

## Rate Limits

`text-embedding-004` free tier: 1500 RPM. At 100 texts/batch the bottleneck is
throughput not rate. Users with heavy workloads should use `--embed-workers 1` if
hitting 429 errors (the driver surfaces the raw API error).

## Testing

- `pkg/embed/driver/gemini/gemini_test.go` — unit tests with `httptest.Server` (mock API)
- `pkg/embed/driver/gemini/integration_test.go` — live API test, build tag `integration`,
  loads key from `$HOME/data/.local.env`
- CLI smoke test: `search cc fts embed run --input /tmp/test-docs/ --driver gemini`
```

- [ ] **Step 2: Commit spec**

```bash
git add spec/0713_gemini_embedding_2.md
git commit -m "spec(embed): add 0713_gemini_embedding_2 spec"
```

---

### Task 2: Write failing unit tests for the Gemini driver

**Files:**
- Create: `pkg/embed/driver/gemini/gemini_test.go`

- [ ] **Step 1: Write the test file**

```go
package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

// mockServer spins up a fake Gemini batchEmbedContents endpoint.
func mockServer(t *testing.T, dim int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req batchEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := batchEmbedResponse{
			Embeddings: make([]embeddingValue, len(req.Requests)),
		}
		for i := range req.Requests {
			vals := make([]float32, dim)
			for j := range vals {
				vals[j] = float32(i+1) * 0.1
			}
			resp.Embeddings[i] = embeddingValue{Values: vals}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestDriverEmbed(t *testing.T) {
	dim := 4
	srv := mockServer(t, dim)
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{baseURL: srv.URL}
	if err := d.Open(ctx, embed.Config{Addr: "fake-key", BatchSize: 10}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	if d.Dimension() != dim {
		t.Fatalf("Dimension() = %d, want %d", d.Dimension(), dim)
	}

	vecs, err := d.Embed(ctx, []embed.Input{
		{Text: "hello"},
		{Text: "world"},
		{Text: "foo"},
	})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("got %d vectors, want 3", len(vecs))
	}
	for i, v := range vecs {
		if len(v.Values) != dim {
			t.Errorf("vec[%d]: got %d dims, want %d", i, len(v.Values), dim)
		}
	}
}

func TestDriverBatching(t *testing.T) {
	dim := 2
	var requestCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req batchEmbedRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp := batchEmbedResponse{Embeddings: make([]embeddingValue, len(req.Requests))}
		for i := range req.Requests {
			resp.Embeddings[i] = embeddingValue{Values: make([]float32, dim)}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{baseURL: srv.URL}
	if err := d.Open(ctx, embed.Config{Addr: "fake-key", BatchSize: 2}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	requestCount = 0 // reset after probe call

	inputs := make([]embed.Input, 5)
	for i := range inputs {
		inputs[i] = embed.Input{Text: "text"}
	}

	vecs, err := d.Embed(ctx, inputs)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 5 {
		t.Fatalf("got %d vectors, want 5", len(vecs))
	}
	// 5 inputs with batchSize=2 → 3 requests (2+2+1)
	if requestCount != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount)
	}
}

func TestDriverAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"API_KEY_INVALID"}}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{baseURL: srv.URL}
	err := d.Open(ctx, embed.Config{Addr: "bad-key"})
	if err == nil {
		d.Close()
		t.Fatal("expected Open to fail with invalid key")
	}
}

func TestDriverName(t *testing.T) {
	d := &Driver{}
	if d.Name() != "gemini" {
		t.Errorf("Name() = %q, want %q", d.Name(), "gemini")
	}
	d.model = "text-embedding-004"
	if d.Name() != "gemini/text-embedding-004" {
		t.Errorf("Name() = %q, want %q", d.Name(), "gemini/text-embedding-004")
	}
}

func TestLoadLocalEnv(t *testing.T) {
	// Write a temp env file with export syntax and plain key=value.
	tmp := t.TempDir() + "/.local.env"
	content := `# comment
export SOME_KEY="value1"
OTHER_KEY=value2
export GEMINI_API_KEY="test-key-123"
`
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got := loadKeyFromFile(tmp, "GEMINI_API_KEY")
	if got != "test-key-123" {
		t.Errorf("loadKeyFromFile = %q, want %q", got, "test-key-123")
	}
}

func TestParseModelName(t *testing.T) {
	tests := []struct {
		input   string
		model   string
		dim     int
	}{
		{"text-embedding-004", "text-embedding-004", 0},
		{"gemini-embedding-exp-03-07", "gemini-embedding-exp-03-07", 0},
		{"gemini-embedding-exp-03-07:768", "gemini-embedding-exp-03-07", 768},
		{"gemini-embedding-exp-03-07:1536", "gemini-embedding-exp-03-07", 1536},
	}
	for _, tt := range tests {
		model, dim := parseModelName(tt.input)
		if model != tt.model || dim != tt.dim {
			t.Errorf("parseModelName(%q) = (%q, %d), want (%q, %d)",
				tt.input, model, dim, tt.model, tt.dim)
		}
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail (driver not yet implemented)**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go test ./pkg/embed/driver/gemini/...
```

Expected: compile error — package does not exist yet.

---

### Task 3: Implement the Gemini driver

**Files:**
- Create: `pkg/embed/driver/gemini/gemini.go`

- [ ] **Step 1: Create the driver**

```go
// Package gemini implements an embed.Driver that calls the Google Gemini
// Embedding API (generativelanguage.googleapis.com).
//
// Supports text-embedding-004 (768-dim) and gemini-embedding-exp-03-07
// (3072-dim, Matryoshka with optional outputDimensionality suffix).
//
// API key is loaded from (in priority order):
//  1. cfg.Addr field
//  2. GEMINI_API_KEY environment variable
//  3. $HOME/data/.local.env file
package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

const (
	defaultModel     = "text-embedding-004"
	defaultBatchSize = 100
	apiBase          = "https://generativelanguage.googleapis.com/v1beta"
	localEnvPath     = ".local.env" // relative to $HOME/data/
)

// modelDims maps known model names to their default output dimensions.
var modelDims = map[string]int{
	"text-embedding-004":          768,
	"gemini-embedding-exp-03-07":  3072,
}

func init() {
	embed.Register("gemini", func() embed.Driver { return &Driver{} })
}

// Driver calls the Gemini Embedding API for embedding generation.
type Driver struct {
	apiKey    string
	model     string // resolved model name (without :NNN suffix)
	outputDim int    // Matryoshka override (0 = use model default)
	dim       int    // actual dimension discovered via probe
	batchSize int
	client    *http.Client
	baseURL   string // overridable for tests (default: apiBase)
}

func (d *Driver) Name() string {
	if d.model != "" {
		return "gemini/" + d.model
	}
	return "gemini"
}

func (d *Driver) Dimension() int { return d.dim }

// Open loads the API key, resolves the model, and probes for the actual dimension.
func (d *Driver) Open(ctx context.Context, cfg embed.Config) error {
	// API key: cfg.Addr → env → local.env
	apiKey := cfg.Addr
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		home, _ := os.UserHomeDir()
		apiKey = loadKeyFromFile(filepath.Join(home, "data", localEnvPath), "GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("gemini: GEMINI_API_KEY not set\n" +
			"  Set it in the environment:  export GEMINI_API_KEY=<key>\n" +
			"  Or add it to $HOME/data/.local.env")
	}
	d.apiKey = apiKey

	// Parse model name — supports "model:dim" for Matryoshka.
	modelInput := cfg.Model
	if modelInput == "" {
		modelInput = defaultModel
	}
	d.model, d.outputDim = parseModelName(modelInput)

	d.batchSize = cfg.BatchSize
	if d.batchSize <= 0 || d.batchSize > defaultBatchSize {
		d.batchSize = defaultBatchSize
	}

	d.client = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        16,
			MaxIdleConnsPerHost: 16,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	if d.baseURL == "" {
		d.baseURL = apiBase
	}

	// Probe: embed one text to confirm key works and discover actual dimension.
	vecs, err := d.callBatch(ctx, []string{"hello"})
	if err != nil {
		return fmt.Errorf("gemini: probe failed: %w", err)
	}
	if len(vecs) == 0 || len(vecs[0].Values) == 0 {
		return fmt.Errorf("gemini: probe returned empty embedding")
	}
	d.dim = len(vecs[0].Values)
	return nil
}

func (d *Driver) Close() error {
	if d.client != nil {
		d.client.CloseIdleConnections()
	}
	return nil
}

// Embed generates embeddings for a batch of inputs.
func (d *Driver) Embed(ctx context.Context, inputs []embed.Input) ([]embed.Vector, error) {
	texts := make([]string, len(inputs))
	for i, inp := range inputs {
		texts[i] = inp.Text
	}

	var all []embed.Vector
	for i := 0; i < len(texts); i += d.batchSize {
		end := i + d.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := d.callBatch(ctx, texts[i:end])
		if err != nil {
			return nil, err
		}
		all = append(all, vecs...)
	}
	return all, nil
}

// --- API types ---

type embedContentRequest struct {
	Model             string      `json:"model"`
	Content           embedContent `json:"content"`
	TaskType          string      `json:"taskType,omitempty"`
	OutputDimensionality int      `json:"outputDimensionality,omitempty"`
}

type embedContent struct {
	Parts []embedPart `json:"parts"`
}

type embedPart struct {
	Text string `json:"text"`
}

type batchEmbedRequest struct {
	Requests []embedContentRequest `json:"requests"`
}

type embeddingValue struct {
	Values []float32 `json:"values"`
}

type batchEmbedResponse struct {
	Embeddings []embeddingValue `json:"embeddings"`
}

// callBatch sends a batchEmbedContents request and returns vectors.
func (d *Driver) callBatch(ctx context.Context, texts []string) ([]embed.Vector, error) {
	reqs := make([]embedContentRequest, len(texts))
	for i, t := range texts {
		r := embedContentRequest{
			Model:    "models/" + d.model,
			Content:  embedContent{Parts: []embedPart{{Text: t}}},
			TaskType: "RETRIEVAL_DOCUMENT",
		}
		if d.outputDim > 0 {
			r.OutputDimensionality = d.outputDim
		}
		reqs[i] = r
	}

	body, err := json.Marshal(batchEmbedRequest{Requests: reqs})
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:batchEmbedContents?key=%s", d.baseURL, d.model, d.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var embResp batchEmbedResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("gemini: decode: %w", err)
	}
	if len(embResp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("gemini: expected %d embeddings, got %d", len(texts), len(embResp.Embeddings))
	}

	result := make([]embed.Vector, len(embResp.Embeddings))
	for i, e := range embResp.Embeddings {
		result[i] = embed.Vector{Values: e.Values}
	}
	return result, nil
}

// --- helpers ---

// parseModelName splits "model:dim" into (model, dim). dim=0 if no suffix.
func parseModelName(s string) (model string, dim int) {
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		if n, err := strconv.Atoi(s[idx+1:]); err == nil {
			return s[:idx], n
		}
	}
	return s, 0
}

// loadKeyFromFile reads key=value or export key="value" lines from a shell env file.
func loadKeyFromFile(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		// Strip leading "export ".
		line = strings.TrimPrefix(line, "export ")
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eqIdx])
		if k != key {
			continue
		}
		v := strings.TrimSpace(line[eqIdx+1:])
		// Strip surrounding quotes.
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		return v
	}
	return ""
}
```

- [ ] **Step 2: Fix the test file — add missing `os` import**

In `gemini_test.go`, add `"os"` to the imports block (the `TestLoadLocalEnv` test uses `os.WriteFile`).

- [ ] **Step 3: Run unit tests**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go test ./pkg/embed/driver/gemini/ -v -run 'TestDriverEmbed|TestDriverBatching|TestDriverAPIError|TestDriverName|TestLoadLocalEnv|TestParseModelName'
```

Expected: all 6 tests pass.

- [ ] **Step 4: Commit**

```bash
git add pkg/embed/driver/gemini/gemini.go pkg/embed/driver/gemini/gemini_test.go
git commit -m "feat(embed/gemini): add Gemini Embedding 2 driver with unit tests"
```

---

## Chunk 2: Models registry + CLI wiring

### Task 4: Add Gemini models to the registry

**Files:**
- Modify: `pkg/embed/models.go`

- [ ] **Step 1: Add Gemini entries to `Models` slice**

In `models.go`, after the `// --- ONNX models ---` block, add:

```go
// --- Gemini API models (no local files — API only) ---
{
    Name:   "text-embedding-004",
    Driver: "gemini",
    Dim:    768,
    SizeMB: 0,
    Desc:   "Gemini text-embedding-004 (768-dim, stable, multilingual, API)",
    Files:  nil,
},
{
    Name:   "gemini-embedding-exp-03-07",
    Driver: "gemini",
    Dim:    3072,
    SizeMB: 0,
    Desc:   "Gemini Embedding 2 (3072-dim Matryoshka, use :768/:1536 suffix for smaller dim)",
    Files:  nil,
},
```

- [ ] **Step 2: Update `DefaultModelName` to handle "gemini"**

In the `DefaultModelName` switch in `models.go`, add:

```go
case "gemini":
    return "text-embedding-004"
```

- [ ] **Step 3: Verify `IsModelDownloaded` returns true for API models**

`IsModelDownloaded` loops over `m.Files` — with nil/empty Files it returns true immediately. No code change needed; verify by reading the function.

- [ ] **Step 4: Run embed package tests**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go test ./pkg/embed/...
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/embed/models.go
git commit -m "feat(embed): register Gemini models in model registry"
```

---

### Task 5: Wire gemini driver into CLI

**Files:**
- Modify: `cli/cc_fts_embed.go`

- [ ] **Step 1: Add blank import for gemini driver**

In the `import` block of `cli/cc_fts_embed.go`, add alongside the existing llamacpp import:

```go
_ "github.com/go-mizu/mizu/blueprints/search/pkg/embed/driver/gemini"
```

- [ ] **Step 2: Update the `Long` description to mention gemini**

In `newCCFTSEmbed()`, update the `Long` field to add:

```
  gemini     Google Gemini Embedding API (no local server needed)
             Requires: GEMINI_API_KEY in env or $HOME/data/.local.env
             Models: text-embedding-004 (768-dim), gemini-embedding-exp-03-07 (3072-dim)
```

- [ ] **Step 3: Build and confirm gemini appears in driver list**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build -o /tmp/search-cli ./cmd/search/
/tmp/search-cli cc fts embed models
```

Expected: gemini driver models appear in list with STATUS=ready.

- [ ] **Step 4: Also verify `--driver` flag auto-populates from registry**

The `--driver` flag default already calls `embed.List()` which is populated by `init()`. After adding the blank import, "gemini" should appear in the flag description.

```bash
/tmp/search-cli cc fts embed run --help 2>&1 | grep -i gemini
```

Expected: "gemini" in driver list.

- [ ] **Step 5: Commit**

```bash
git add cli/cc_fts_embed.go
git commit -m "feat(cli): wire gemini embed driver into search cc fts embed"
```

---

## Chunk 3: Integration test + end-to-end smoke test

### Task 6: Integration test (live API)

**Files:**
- Create: `pkg/embed/driver/gemini/integration_test.go`

- [ ] **Step 1: Write integration test**

```go
//go:build integration

package gemini

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

// TestIntegrationEmbed calls the live Gemini API.
// Run with: go test -tags integration ./pkg/embed/driver/gemini/
func TestIntegrationEmbed(t *testing.T) {
	home, _ := os.UserHomeDir()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = loadKeyFromFile(filepath.Join(home, "data", ".local.env"), "GEMINI_API_KEY")
	}
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set — skipping integration test")
	}

	ctx := context.Background()

	tests := []struct {
		model string
		wantDim int
	}{
		{"text-embedding-004", 768},
		{"gemini-embedding-exp-03-07", 3072},
		{"gemini-embedding-exp-03-07:768", 768},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			d := &Driver{}
			if err := d.Open(ctx, embed.Config{
				Addr:  apiKey,
				Model: tt.model,
			}); err != nil {
				t.Fatalf("Open(%s): %v", tt.model, err)
			}
			defer d.Close()

			if d.Dimension() != tt.wantDim {
				t.Errorf("Dimension() = %d, want %d", d.Dimension(), tt.wantDim)
			}

			vecs, err := d.Embed(ctx, []embed.Input{
				{Text: "The quick brown fox jumps over the lazy dog."},
				{Text: "Go is an open-source programming language."},
			})
			if err != nil {
				t.Fatalf("Embed: %v", err)
			}
			if len(vecs) != 2 {
				t.Fatalf("got %d vectors, want 2", len(vecs))
			}
			for i, v := range vecs {
				if len(v.Values) != tt.wantDim {
					t.Errorf("vec[%d]: got %d dims, want %d", i, len(v.Values), tt.wantDim)
				}
				// Sanity: at least some non-zero values.
				nonzero := 0
				for _, f := range v.Values {
					if f != 0 {
						nonzero++
					}
				}
				if nonzero == 0 {
					t.Errorf("vec[%d]: all zeros — likely an API issue", i)
				}
			}
			t.Logf("model=%s dim=%d ok", tt.model, tt.wantDim)
		})
	}
}

// TestIntegrationBatch verifies batching works correctly with the live API.
func TestIntegrationBatch(t *testing.T) {
	home, _ := os.UserHomeDir()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = loadKeyFromFile(filepath.Join(home, "data", ".local.env"), "GEMINI_API_KEY")
	}
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set — skipping integration test")
	}

	ctx := context.Background()
	d := &Driver{}
	if err := d.Open(ctx, embed.Config{
		Addr:      apiKey,
		BatchSize: 3, // force batching with small batch size
	}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// 7 inputs with batchSize=3 → 3 API calls (3+3+1)
	inputs := make([]embed.Input, 7)
	for i := range inputs {
		inputs[i] = embed.Input{Text: fmt.Sprintf("test sentence number %d for embedding", i+1)}
	}

	vecs, err := d.Embed(ctx, inputs)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 7 {
		t.Fatalf("got %d vectors, want 7", len(vecs))
	}
	t.Logf("batched 7 inputs → %d vectors, dim=%d", len(vecs), len(vecs[0].Values))
}
```

- [ ] **Step 2: Add missing `fmt` import to integration test**

The `TestIntegrationBatch` test uses `fmt.Sprintf` — ensure `"fmt"` is in the imports.

- [ ] **Step 3: Run integration tests against live API**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go test -tags integration -v ./pkg/embed/driver/gemini/ -run TestIntegration -timeout 60s
```

Expected output (approximate):
```
--- PASS: TestIntegrationEmbed/text-embedding-004 (1.5s)
    gemini_integration_test.go:XX: model=text-embedding-004 dim=768 ok
--- PASS: TestIntegrationEmbed/gemini-embedding-exp-03-07 (2.1s)
    gemini_integration_test.go:XX: model=gemini-embedding-exp-03-07 dim=3072 ok
--- PASS: TestIntegrationEmbed/gemini-embedding-exp-03-07:768 (1.8s)
    gemini_integration_test.go:XX: model=gemini-embedding-exp-03-07:768 dim=768 ok
--- PASS: TestIntegrationBatch (2.0s)
    gemini_integration_test.go:XX: batched 7 inputs → 7 vectors, dim=768
PASS
```

- [ ] **Step 4: Commit**

```bash
git add pkg/embed/driver/gemini/integration_test.go
git commit -m "test(embed/gemini): add integration tests for live Gemini API"
```

---

### Task 7: End-to-end CLI smoke test

**Files:** none (uses built binary + temp directory)

- [ ] **Step 1: Create a small test document set**

```bash
mkdir -p /tmp/test-docs-gemini
cat > /tmp/test-docs-gemini/doc1.md << 'EOF'
# Introduction to Go

Go is a statically typed, compiled programming language designed at Google.
It is syntactically similar to C but with memory safety, garbage collection,
structural typing, and CSP-style concurrency.
EOF

cat > /tmp/test-docs-gemini/doc2.md << 'EOF'
# Vector Search

Vector search finds semantically similar documents by comparing embedding
vectors using cosine similarity or dot product. Dense retrieval outperforms
traditional BM25 on many NLP benchmarks.
EOF
```

- [ ] **Step 2: Source env and run embed**

```bash
source $HOME/data/.local.env
/tmp/search-cli cc fts embed run --input /tmp/test-docs-gemini/ --driver gemini --output /tmp/test-embed-gemini/
```

Expected output (stderr):
```
embed: driver=gemini/text-embedding-004 dim=768 batch=100
embed: /tmp/test-docs-gemini/ → /tmp/test-embed-gemini/
  found 2 markdown files, embed-workers=4
  files=2 chunks=N vectors=N errors=0 X vec/s elapsed=Xs
  output: /tmp/test-embed-gemini/vectors.bin (X bytes)
  meta:   /tmp/test-embed-gemini/meta.jsonl
```

- [ ] **Step 3: Validate output files**

```bash
# vectors.bin must be non-empty and a multiple of 768*4 bytes
ls -la /tmp/test-embed-gemini/
python3 -c "
import os, struct
size = os.path.getsize('/tmp/test-embed-gemini/vectors.bin')
dim = 768
assert size % (dim * 4) == 0, f'size {size} not divisible by {dim*4}'
n = size // (dim * 4)
print(f'vectors.bin: {n} vectors, dim={dim}, size={size}B — OK')
"

# meta.jsonl: one JSON object per line
head -2 /tmp/test-embed-gemini/meta.jsonl
cat /tmp/test-embed-gemini/stats.json
```

Expected:
- `vectors.bin` size is `N * 768 * 4` bytes
- `meta.jsonl` has N lines of valid JSON with `id`, `file`, `chunk_idx`, `text_len`, `dim`
- `stats.json` shows `driver: "gemini/text-embedding-004"`, `dim: 768`, `errors: 0`

- [ ] **Step 4: Test with gemini-embedding-exp-03-07 (3072-dim)**

```bash
source $HOME/data/.local.env
/tmp/search-cli cc fts embed run \
  --input /tmp/test-docs-gemini/ \
  --driver gemini \
  --model gemini-embedding-exp-03-07 \
  --output /tmp/test-embed-gemini-3072/
```

Validate:
```bash
python3 -c "
import os
size = os.path.getsize('/tmp/test-embed-gemini-3072/vectors.bin')
dim = 3072
assert size % (dim * 4) == 0, f'bad size {size}'
print(f'{size//(dim*4)} vectors at dim={dim} — OK')
"
```

- [ ] **Step 5: Test Matryoshka reduced dim**

```bash
source $HOME/data/.local.env
/tmp/search-cli cc fts embed run \
  --input /tmp/test-docs-gemini/ \
  --driver gemini \
  --model gemini-embedding-exp-03-07:768 \
  --output /tmp/test-embed-gemini-mat768/
```

Validate that output has 768-dim vectors (same as text-embedding-004 but from exp model).

- [ ] **Step 6: Final commit**

```bash
git add spec/0713_gemini_embedding_2.md  # ensure spec is committed
git status  # verify clean
```

If anything is uncommitted:
```bash
git add -p
git commit -m "chore: finalize gemini embedding 2 implementation"
```
