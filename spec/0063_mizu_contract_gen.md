# Spec 0063: Mizu Contract SDK Generation CLI

## Overview

Enhance the `mizu contract gen` command to generate fully-typed SDK clients for Go, Python, and TypeScript. The command integrates existing SDK generators from `contract/v2/sdk/{go,py,ts}` into the CLI, providing a unified interface for SDK generation.

**Package**: `cmd/cli` (file: `contract_v2.go`)

## Design Goals

1. **Simple CLI interface** - Single command with `--client` flag to generate SDK clients
2. **Multi-language support** - Go, Python, and TypeScript SDKs from same contract
3. **Consistent output** - All SDKs follow OpenAI-style DX patterns
4. **Tested connectivity** - E2E tests verify generated SDKs can connect to servers
5. **Minimal changes** - Reuse existing SDK generators, only wire CLI integration

## Current State

The existing `mizu contract gen` command generates types-only code:

```bash
# Current behavior (types only)
mizu contract gen api.yaml --lang typescript  # generates api.gen.ts (types)
mizu contract gen api.yaml --lang go          # generates api.gen.go (types)
```

The SDK generators already exist but are not wired to CLI:
- `contract/v2/sdk/go` - Full Go SDK client generator
- `contract/v2/sdk/py` - Full Python SDK client generator
- `contract/v2/sdk/ts` - Full TypeScript SDK client generator

## Enhanced CLI Interface

### Command Syntax

```bash
# Generate SDK client (new behavior with --client)
mizu contract gen api.yaml --client --lang go
mizu contract gen api.yaml --client --lang python
mizu contract gen api.yaml --client --lang typescript

# Short forms
mizu contract gen api.yaml -c --lang ts
mizu contract gen api.yaml -c --lang py

# With output directory
mizu contract gen api.yaml --client --lang go --output ./sdk/go

# With package name
mizu contract gen api.yaml --client --lang go --package myapi
mizu contract gen api.yaml --client --lang python --package myapi
mizu contract gen api.yaml --client --lang typescript --package myapi

# Generate all languages at once
mizu contract gen api.yaml --client --lang all --output ./sdks
```

### Command Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--client` | `-c` | bool | false | Generate SDK client (not just types) |
| `--lang` | | string | typescript | Target language (go, python, py, typescript, ts, all) |
| `--output` | `-o` | string | (same as input dir) | Output directory |
| `--package` | | string | (from service name) | Package/module name |
| `--version` | | string | 0.0.0 | Version for package.json/pyproject.toml |

### Language Mapping

| Input | Normalized | Generator |
|-------|------------|-----------|
| `go`, `golang` | `go` | `sdkgo.Generate()` |
| `python`, `py` | `python` | `sdkpy.Generate()` |
| `typescript`, `ts` | `typescript` | `sdkts.Generate()` |
| `all` | all | All three generators |

## Output Structure

### Go SDK Output

```
{output}/
└── client.go       # Single file with types, client, resources, streaming
```

### Python SDK Output

```
{output}/
├── pyproject.toml
└── src/
    └── {package}/
        ├── __init__.py
        ├── _client.py
        ├── _types.py
        ├── _streaming.py
        └── _resource.py
```

### TypeScript SDK Output

```
{output}/
├── package.json
├── tsconfig.json
└── src/
    ├── index.ts
    ├── _client.ts
    ├── _types.ts
    ├── _streaming.ts
    └── _resources.ts
```

### Multi-language Output (--lang all)

```
{output}/
├── go/
│   └── client.go
├── python/
│   ├── pyproject.toml
│   └── src/{package}/...
└── typescript/
    ├── package.json
    └── src/...
```

## Implementation

### CLI Changes (`contract_v2.go`)

```go
import (
    sdkgo "github.com/go-mizu/mizu/contract/v2/sdk/go"
    sdkpy "github.com/go-mizu/mizu/contract/v2/sdk/py"
    sdkts "github.com/go-mizu/mizu/contract/v2/sdk/ts"
)

// Add to contractV2Flags struct
var contractV2Flags struct {
    // ... existing flags ...
    client  bool   // NEW: Generate SDK client
    version string // NEW: Package version
}

// Add flag registration in init()
func init() {
    // ... existing code ...
    contractGenCmd.Flags().BoolVarP(&contractV2Flags.client, "client", "c", false, "Generate SDK client")
    contractGenCmd.Flags().StringVar(&contractV2Flags.version, "version", "0.0.0", "Package version (Python/TypeScript)")
}

// Update runContractGenCmd
func runContractGenCmd(cmd *cobra.Command, args []string) error {
    // ... existing validation code ...

    if contractV2Flags.client {
        return runContractGenClientCmd(svc)
    }

    // ... existing types-only generation ...
}

// New function for SDK client generation
func runContractGenClientCmd(svc *contract.Service) error {
    out := NewOutput()

    lang := normalizeLang(contractV2Flags.lang)
    outputDir := contractV2Flags.output
    if outputDir == "" {
        outputDir = "."
    }

    languages := []string{lang}
    if lang == "all" {
        languages = []string{"go", "python", "typescript"}
    }

    for _, l := range languages {
        subDir := outputDir
        if lang == "all" {
            subDir = filepath.Join(outputDir, l)
        }

        files, err := generateSDK(svc, l, subDir)
        if err != nil {
            out.PrintError("generate %s SDK failed: %v", l, err)
            return err
        }

        // Write files
        for _, f := range files {
            path := filepath.Join(subDir, f.Path)
            if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
                return err
            }
            if err := os.WriteFile(path, []byte(f.Content), 0644); err != nil {
                return err
            }
            out.Print("%s %s\n", out.Success("Created"), path)
        }
    }

    return nil
}

func generateSDK(svc *contract.Service, lang, outputDir string) ([]*sdk.File, error) {
    pkg := contractV2Flags.pkg
    version := contractV2Flags.version

    switch lang {
    case "go":
        return sdkgo.Generate(svc, &sdkgo.Config{
            Package: pkg,
        })
    case "python":
        return sdkpy.Generate(svc, &sdkpy.Config{
            Package: pkg,
            Version: version,
        })
    case "typescript":
        return sdkts.Generate(svc, &sdkts.Config{
            Package: pkg,
            Version: version,
        })
    default:
        return nil, fmt.Errorf("unsupported language: %s", lang)
    }
}

func normalizeLang(lang string) string {
    switch strings.ToLower(lang) {
    case "go", "golang":
        return "go"
    case "python", "py":
        return "python"
    case "typescript", "ts":
        return "typescript"
    case "all":
        return "all"
    default:
        return lang
    }
}
```

## Testing Strategy

### Test Location

Tests are added to a new file: `cmd/cli/contract_gen_test.go`

### Test Categories

1. **Unit Tests** - CLI flag parsing and validation
2. **Integration Tests** - Full generation pipeline
3. **E2E Connectivity Tests** - Generated SDKs connect to test server

### Unit Tests

```go
func TestContractGen_ClientFlag_GoGeneration(t *testing.T) {
    // Create temp contract file
    contractYAML := `
name: TestAPI
resources:
  - name: items
    methods:
      - name: list
        output: ItemList
        http:
          method: GET
          path: /items
types:
  - name: ItemList
    kind: slice
    elem: Item
  - name: Item
    kind: struct
    fields:
      - name: id
        type: string
`
    // Write and generate
    // Verify Go client.go is created
    // Verify code parses and typechecks
}

func TestContractGen_ClientFlag_PythonGeneration(t *testing.T) {
    // Verify pyproject.toml and src/ structure
}

func TestContractGen_ClientFlag_TypeScriptGeneration(t *testing.T) {
    // Verify package.json and src/ structure
}

func TestContractGen_LangAll_GeneratesAllLanguages(t *testing.T) {
    // Verify go/, python/, typescript/ subdirs created
}
```

### E2E Connectivity Tests

Environment variable enables E2E tests:
- `SDKGO_E2E=1` - Enable Go SDK E2E tests
- `SDKPY_E2E=1` - Enable Python SDK E2E tests
- `SDKTS_E2E=1` - Enable TypeScript SDK E2E tests

#### Go SDK E2E Test

```go
func TestContractGen_Go_E2E_Connectivity(t *testing.T) {
    if os.Getenv("SDKGO_E2E") == "" {
        t.Skip("SDKGO_E2E not set")
    }

    // 1. Start test HTTP server
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.Method != "GET" || r.URL.Path != "/items" {
            t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
        }
        if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
            t.Errorf("unexpected auth: %s", auth)
        }
        // Return response
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode([]map[string]string{{"id": "1"}})
    }))
    defer srv.Close()

    // 2. Generate SDK
    root := t.TempDir()
    // ... generate SDK to root ...

    // 3. Create test program
    testProg := fmt.Sprintf(`
package main

import (
    "context"
    "fmt"
    "testapi"
)

func main() {
    client := testapi.NewClient("test-token", testapi.WithBaseURL(%q))
    items, err := client.Items.List(context.Background())
    if err != nil {
        panic(err)
    }
    fmt.Printf("count=%%d\n", len(items))
}
`, srv.URL)

    // 4. Write test program
    // 5. Run: go run .
    // 6. Verify output contains "count=1"
}
```

#### Python SDK E2E Test

```go
func TestContractGen_Python_E2E_Connectivity(t *testing.T) {
    if os.Getenv("SDKPY_E2E") == "" {
        t.Skip("SDKPY_E2E not set")
    }
    requireUV(t)

    // 1. Start test HTTP server
    srv := httptest.NewServer(...)

    // 2. Generate SDK
    root := t.TempDir()
    // ... generate SDK to root ...

    // 3. Create virtual env and install
    runUV(t, root, "venv", ".venv")
    runUV(t, root, "pip", "install", "-e", ".")

    // 4. Create test script
    testScript := fmt.Sprintf(`
from testapi import TestAPI
client = TestAPI(api_key="test-token", base_url=%q)
items = client.items.list()
print(f"count={len(items)}")
`, srv.URL)

    // 5. Run: uv run python test.py
    // 6. Verify output contains "count=1"
}

func requireUV(t *testing.T) {
    if _, err := exec.LookPath("uv"); err != nil {
        t.Skip("uv not installed")
    }
}
```

#### TypeScript SDK E2E Test

```go
func TestContractGen_TypeScript_E2E_Connectivity_Node(t *testing.T) {
    if os.Getenv("SDKTS_E2E") == "" {
        t.Skip("SDKTS_E2E not set")
    }
    requireNode(t)

    // 1. Start test HTTP server
    srv := httptest.NewServer(...)

    // 2. Generate SDK
    root := t.TempDir()
    // ... generate SDK to root ...

    // 3. Create test script
    testScript := fmt.Sprintf(`
import { TestAPI } from './src/index.js';
const client = new TestAPI({ apiKey: 'test-token', baseURL: %q });
const items = await client.items.list();
console.log('count=' + items.length);
`, srv.URL)

    // 4. Run: npx tsx test.ts
    // 5. Verify output contains "count=1"
}

func TestContractGen_TypeScript_E2E_Connectivity_Bun(t *testing.T) {
    if os.Getenv("SDKTS_E2E") == "" {
        t.Skip("SDKTS_E2E not set")
    }
    requireBun(t)
    // Similar to Node but use: bun run test.ts
}

func TestContractGen_TypeScript_E2E_Connectivity_Deno(t *testing.T) {
    if os.Getenv("SDKTS_E2E") == "" {
        t.Skip("SDKTS_E2E not set")
    }
    requireDeno(t)
    // Similar but use: deno run --allow-net test.ts
}
```

### Streaming Tests

```go
func TestContractGen_Go_E2E_SSE_Streaming(t *testing.T) {
    if os.Getenv("SDKGO_E2E") == "" {
        t.Skip("SDKGO_E2E not set")
    }

    // Server sends SSE events
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/event-stream")
        flusher := w.(http.Flusher)

        io.WriteString(w, "data: {\"text\":\"a\"}\n\n")
        flusher.Flush()
        io.WriteString(w, "data: {\"text\":\"b\"}\n\n")
        flusher.Flush()
        io.WriteString(w, "data: [DONE]\n\n")
        flusher.Flush()
    }))
    defer srv.Close()

    // Generate SDK with streaming method
    // Run test program that collects events
    // Verify: events=["a","b"]
}
```

## Test Execution

```bash
# Run unit tests (no external dependencies)
go test ./cmd/cli/... -run TestContractGen

# Run with Go E2E tests
SDKGO_E2E=1 go test ./cmd/cli/... -run TestContractGen_Go_E2E

# Run with Python E2E tests (requires uv)
SDKPY_E2E=1 go test ./cmd/cli/... -run TestContractGen_Python_E2E

# Run with TypeScript E2E tests (requires node/bun/deno)
SDKTS_E2E=1 go test ./cmd/cli/... -run TestContractGen_TypeScript_E2E

# Run all E2E tests
SDKGO_E2E=1 SDKPY_E2E=1 SDKTS_E2E=1 go test ./cmd/cli/... -run TestContractGen
```

## Example Usage

### Basic SDK Generation

```bash
# Create contract
mizu contract init myapi

# Generate Go SDK client
mizu contract gen api.yaml --client --lang go --output ./sdk/go

# Generate Python SDK client
mizu contract gen api.yaml --client --lang python --output ./sdk/python

# Generate TypeScript SDK client
mizu contract gen api.yaml --client --lang typescript --output ./sdk/typescript

# Generate all SDKs
mizu contract gen api.yaml --client --lang all --output ./sdks
```

### Using Generated Go SDK

```go
package main

import (
    "context"
    "fmt"
    "myapi"
)

func main() {
    client := myapi.NewClient("your-api-key")

    items, err := client.Items.List(context.Background())
    if err != nil {
        panic(err)
    }

    for _, item := range items {
        fmt.Println(item.ID, item.Name)
    }
}
```

### Using Generated Python SDK

```python
from myapi import MyAPI

client = MyAPI(api_key="your-api-key")

items = client.items.list()
for item in items:
    print(item.id, item.name)

# Async
import asyncio
from myapi import AsyncMyAPI

async def main():
    client = AsyncMyAPI(api_key="your-api-key")
    items = await client.items.list()
    print(items)

asyncio.run(main())
```

### Using Generated TypeScript SDK

```typescript
import { MyAPI } from 'myapi';

const client = new MyAPI({ apiKey: 'your-api-key' });

const items = await client.items.list();
items.forEach(item => console.log(item.id, item.name));

// Streaming
for await (const event of client.items.stream({ query: 'hello' })) {
    console.log(event);
}
```

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `cmd/cli/contract_v2.go` | Modify | Add `--client` flag and SDK generation logic |
| `cmd/cli/contract_gen_test.go` | Create | Unit and E2E tests for SDK generation |
| `cmd/go.mod` | Modify | Add SDK generator imports |

## Dependencies

CLI module needs to import SDK generators:

```go
// cmd/go.mod additions
require (
    github.com/go-mizu/mizu/contract/v2/sdk/go
    github.com/go-mizu/mizu/contract/v2/sdk/py
    github.com/go-mizu/mizu/contract/v2/sdk/ts
)
```

## Success Criteria

- [ ] `mizu contract gen --client --lang go` generates valid Go SDK
- [ ] `mizu contract gen --client --lang python` generates valid Python SDK
- [ ] `mizu contract gen --client --lang typescript` generates valid TypeScript SDK
- [ ] `mizu contract gen --client --lang all` generates all three SDKs
- [ ] Generated Go SDK typechecks with `go build`
- [ ] Generated Python SDK installs with `uv pip install -e .`
- [ ] Generated TypeScript SDK typechecks with `tsc --noEmit`
- [ ] E2E tests verify HTTP connectivity for all languages
- [ ] E2E tests verify SSE streaming for all languages
- [ ] Error messages are clear and actionable
- [ ] `--output` flag correctly controls output directory
- [ ] `--package` flag correctly names the package/module

## Future Extensions

Not in scope for this spec:

- **Interactive mode** - Wizard for contract generation
- **Watch mode** - Regenerate on contract file changes
- **Validation** - Validate contract before generation
- **Diff mode** - Show what would change before writing
