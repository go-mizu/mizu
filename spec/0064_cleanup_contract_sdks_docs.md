# Cleanup Contract SDK Documentation

## Overview

Rewrite the contract SDK documentation pages to focus on CLI-based generation using `mizu contract gen` command, remove H1 headers (since frontmatter provides title), shorten titles, and rename the documentation group from "SDK Generation" to "SDKs".

## Changes Required

### 1. docs/docs.json

- Rename group `"SDK Generation"` to `"SDKs"`

### 2. docs/contract/sdk-overview.mdx

**Title Change:** "SDK Generation" -> "Overview"

**Content Updates:**
- Remove H1 header `# SDK Generation Overview`
- Focus on CLI-based generation using `mizu contract gen`
- Show workflow: contract YAML -> `mizu contract gen` -> SDK files
- Remove references to programmatic Go SDK generator usage
- Update examples to use CLI commands
- Remove any mentions of specific vendor names

### 3. docs/contract/sdk-go.mdx

**Title Change:** "Go SDK" -> "Go"

**Content Updates:**
- Remove H1 header `# Go SDK Generator`
- Focus on CLI-based generation: `mizu contract gen api.yaml --client --lang go`
- Remove programmatic sdkgo.Generate() examples for generation
- Keep client usage examples (how to use the generated SDK)
- Remove any mentions of specific vendor names

### 4. docs/contract/sdk-python.mdx

**Title Change:** "Python SDK" -> "Python"

**Content Updates:**
- Remove H1 header `# Python SDK Generator`
- Focus on CLI-based generation: `mizu contract gen api.yaml --client --lang python`
- Remove programmatic sdkpy.Generate() examples for generation
- Keep client usage examples (how to use the generated SDK)
- Remove any mentions of specific vendor names
- Change client class names from `OpenAI` to generic names like `Client` or `APIClient`

### 5. docs/contract/sdk-typescript.mdx

**Title Change:** "TypeScript SDK" -> "TypeScript"

**Content Updates:**
- Remove H1 header `# TypeScript SDK Generator`
- Focus on CLI-based generation: `mizu contract gen api.yaml --client --lang typescript`
- Remove programmatic sdkts.Generate() examples for generation
- Keep client usage examples (how to use the generated SDK)
- Remove any mentions of specific vendor names

## CLI Command Reference

The `mizu contract gen` command syntax:

```bash
# Generate types only
mizu contract gen api.yaml --lang go
mizu contract gen api.yaml --lang python
mizu contract gen api.yaml --lang typescript

# Generate full SDK client
mizu contract gen api.yaml --client --lang go
mizu contract gen api.yaml --client --lang python --output ./sdk
mizu contract gen api.yaml --client --lang typescript

# Generate all languages
mizu contract gen api.yaml --client --lang all --output ./sdks
```

### Flags

| Flag | Description |
|------|-------------|
| `--lang` | Target language: `go`, `python`, `py`, `typescript`, `ts`, `all` |
| `--output`, `-o` | Output directory |
| `--package` | Package/module name (default: from service name) |
| `--version` | Package version (Python/TypeScript only) |
| `--client`, `-c` | Generate full SDK client (not just types) |

## Implementation Order

1. Write plan (this file)
2. Update docs/docs.json
3. Rewrite sdk-overview.mdx
4. Rewrite sdk-go.mdx
5. Rewrite sdk-python.mdx
6. Rewrite sdk-typescript.mdx
