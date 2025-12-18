# Contract CLI v2 Migration Specification

**Status:** Implementation Ready
**Depends on:** 0049_contract_v2.md

## Overview

This spec defines the migration of the Mizu CLI `contract` command to support contract/v2's definition-first approach while maintaining backward compatibility with v1 services.

## Current CLI Commands

| Command | Purpose |
|---------|---------|
| `mizu contract ls [url]` | List services and methods |
| `mizu contract show <method> [url]` | Show method details |
| `mizu contract call <method> [input] [url]` | Call a method |
| `mizu contract spec [url]` | Export API specification |
| `mizu contract types [type] [url]` | List types and schemas |

## New Commands for v2

### `mizu contract init`

Create a new contract definition file:

```bash
mizu contract init [name]
mizu contract init --template openai
mizu contract init --from http://localhost:8080/openapi.json
```

**Flags:**
- `--template <name>` - Use a template (openai, github, minimal)
- `--from <url>` - Import from existing OpenAPI/OpenRPC spec
- `--output <file>` - Output file (default: api.yaml)

### `mizu contract validate`

Validate a contract definition:

```bash
mizu contract validate [file]
mizu contract validate api.yaml
```

**Flags:**
- `--strict` - Fail on warnings

### `mizu contract gen`

Generate code from contract definition:

```bash
mizu contract gen [file]
mizu contract gen api.yaml --lang typescript
mizu contract gen api.yaml --lang go --package api
```

**Flags:**
- `--lang <language>` - Target language (typescript, go)
- `--output <dir>` - Output directory
- `--package <name>` - Package name (Go)
- `--client` - Generate client code
- `--server` - Generate server code
- `--types` - Generate types only

### Enhanced `mizu contract spec`

Export specification from contract definition:

```bash
mizu contract spec api.yaml --format openapi
mizu contract spec api.yaml --format openrpc
mizu contract spec api.yaml --format asyncapi
```

**Flags:**
- `--format <format>` - Output format (openapi, openrpc, asyncapi)
- `--pretty` - Pretty print
- `--output <file>` - Output file

## Implementation Plan

### Phase 1: YAML Parsing

Add YAML support to load v2 contract definitions:

```go
// contract_v2.go

func loadContractV2(path string) (*contract.Service, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var svc contract.Service
    if err := yaml.Unmarshal(data, &svc); err != nil {
        return nil, err
    }

    return &svc, nil
}
```

### Phase 2: New Commands

#### `init` command

```go
var contractInitCmd = &cobra.Command{
    Use:   "init [name]",
    Short: "Create a new contract definition",
    RunE:  wrapRunE(runContractInitCmd),
}

func runContractInitCmd(cmd *cobra.Command, args []string) error {
    // 1. Determine name
    // 2. Load template if specified
    // 3. Import from URL if --from specified
    // 4. Write api.yaml
}
```

#### `validate` command

```go
var contractValidateCmd = &cobra.Command{
    Use:   "validate [file]",
    Short: "Validate a contract definition",
    RunE:  wrapRunE(runContractValidateCmd),
}

func runContractValidateCmd(cmd *cobra.Command, args []string) error {
    // 1. Load YAML
    // 2. Parse into contract.Service
    // 3. Validate:
    //    - All type refs resolve
    //    - HTTP paths are valid
    //    - Union variants exist
    //    - Required fields present
    // 4. Report errors/warnings
}
```

#### `gen` command

```go
var contractGenCmd = &cobra.Command{
    Use:   "gen [file]",
    Short: "Generate code from contract",
    RunE:  wrapRunE(runContractGenCmd),
}

func runContractGenCmd(cmd *cobra.Command, args []string) error {
    // 1. Load contract
    // 2. Select generator based on --lang
    // 3. Generate code
    // 4. Write to output
}
```

### Phase 3: Code Generators

#### TypeScript Generator

```go
// codegen/typescript.go

type TypeScriptGenerator struct {
    svc *contract.Service
}

func (g *TypeScriptGenerator) GenerateTypes() string {
    var sb strings.Builder

    for _, t := range g.svc.Types {
        switch t.Kind {
        case contract.KindStruct:
            g.writeStruct(&sb, t)
        case contract.KindSlice:
            g.writeSlice(&sb, t)
        case contract.KindUnion:
            g.writeUnion(&sb, t)
        case contract.KindMap:
            g.writeMap(&sb, t)
        }
    }

    return sb.String()
}

func (g *TypeScriptGenerator) writeStruct(sb *strings.Builder, t *contract.Type) {
    fmt.Fprintf(sb, "export type %s = {\n", t.Name)
    for _, f := range t.Fields {
        optional := ""
        if f.Optional {
            optional = "?"
        }
        tsType := g.typeToTS(f.Type, f.Nullable, f.Enum)
        fmt.Fprintf(sb, "  %s%s: %s\n", f.Name, optional, tsType)
    }
    fmt.Fprintf(sb, "}\n\n")
}

func (g *TypeScriptGenerator) typeToTS(ref contract.TypeRef, nullable bool, enum []string) string {
    if len(enum) > 0 {
        quoted := make([]string, len(enum))
        for i, v := range enum {
            quoted[i] = fmt.Sprintf("%q", v)
        }
        return strings.Join(quoted, " | ")
    }

    base := g.primitiveToTS(string(ref))
    if nullable {
        return base + " | null"
    }
    return base
}

func (g *TypeScriptGenerator) primitiveToTS(t string) string {
    switch t {
    case "string":
        return "string"
    case "int", "int32", "int64", "float32", "float64":
        return "number"
    case "bool":
        return "boolean"
    case "time.Time":
        return "string"
    case "json.RawMessage":
        return "unknown"
    default:
        return t // Declared type reference
    }
}
```

#### Go Generator

```go
// codegen/golang.go

type GoGenerator struct {
    svc     *contract.Service
    Package string
}

func (g *GoGenerator) GenerateTypes() string {
    var sb strings.Builder

    fmt.Fprintf(&sb, "package %s\n\n", g.Package)

    for _, t := range g.svc.Types {
        switch t.Kind {
        case contract.KindStruct:
            g.writeStruct(&sb, t)
        case contract.KindSlice:
            g.writeSlice(&sb, t)
        // ...
        }
    }

    return sb.String()
}
```

### Phase 4: Spec Export Enhancement

Update the `spec` command to work with v2 contracts:

```go
func runContractSpecCmd(cmd *cobra.Command, args []string) error {
    // Check if first arg is a file
    if len(args) > 0 && isLocalFile(args[0]) {
        return runContractSpecV2(cmd, args)
    }

    // Fall back to URL discovery (v1 behavior)
    return runContractSpecV1(cmd, args)
}

func runContractSpecV2(cmd *cobra.Command, args []string) error {
    svc, err := loadContractV2(args[0])
    if err != nil {
        return err
    }

    var specData []byte
    switch contractFlags.format {
    case "openapi", "":
        specData, err = rest.OpenAPIDocument(svc)
    case "openrpc":
        specData, err = jsonrpc.OpenRPCDocument(svc)
    case "asyncapi":
        specData, err = async.AsyncAPIDocument(svc)
    }

    // Output handling...
}
```

### Phase 5: Enhanced Discovery

Update discovery to recognize v2 contracts:

```go
func discoverContractNew(url string, out *Output) (*ContractInfo, error) {
    // Existing v1 discovery chain...

    // Add v2 YAML discovery
    info, err := discoverContractV2YAML(url)
    if err == nil {
        return info, nil
    }
}

// Convert v2 Service to ContractInfo for display
func serviceToContractInfo(svc *contract.Service) *ContractInfo {
    info := &ContractInfo{
        Types: make([]TypeInfo, 0),
    }

    for _, res := range svc.Resources {
        svcInfo := ServiceInfo{
            Name:        res.Name,
            Description: res.Description,
            Methods:     make([]MethodInfo, 0),
        }

        for _, m := range res.Methods {
            method := MethodInfo{
                Name:        m.Name,
                FullName:    res.Name + "." + m.Name,
                Description: m.Description,
            }
            if m.HTTP != nil {
                method.HTTPMethod = m.HTTP.Method
                method.HTTPPath = m.HTTP.Path
            }
            if m.Input != "" {
                method.Input = &TypeRef{Name: string(m.Input)}
            }
            if m.Output != "" {
                method.Output = &TypeRef{Name: string(m.Output)}
            }
            svcInfo.Methods = append(svcInfo.Methods, method)
        }

        info.Services = append(info.Services, svcInfo)
    }

    // Convert types
    for _, t := range svc.Types {
        info.Types = append(info.Types, typeToTypeInfo(t))
    }

    return info
}
```

## Command Summary

### Updated Commands

| Command | v1 Support | v2 Support |
|---------|------------|------------|
| `contract ls` | Yes (URL) | Yes (file or URL) |
| `contract show` | Yes | Yes |
| `contract call` | Yes | Yes |
| `contract spec` | Yes | Enhanced (multiple formats) |
| `contract types` | Yes | Yes |

### New Commands

| Command | Purpose |
|---------|---------|
| `contract init` | Create new contract definition |
| `contract validate` | Validate contract syntax and semantics |
| `contract gen` | Generate code from contract |

## File Detection Logic

```go
func isContractFile(arg string) bool {
    if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
        return false
    }
    ext := filepath.Ext(arg)
    return ext == ".yaml" || ext == ".yml" || ext == ".json"
}
```

## Templates

### Minimal Template

```yaml
name: MyAPI
description: My API description
defaults:
  base_url: http://localhost:8080

resources:
  - name: items
    methods:
      - name: list
        output: ItemList
        http:
          method: GET
          path: /items

      - name: create
        input: CreateItemRequest
        output: Item
        http:
          method: POST
          path: /items

types:
  - name: Item
    kind: struct
    fields:
      - name: id
        type: int64
      - name: name
        type: string

  - name: ItemList
    kind: slice
    elem: Item

  - name: CreateItemRequest
    kind: struct
    fields:
      - name: name
        type: string
```

## Error Messages

```
Error: invalid contract definition
  api.yaml:15: unknown type reference "InvalidType" in method responses.create
  api.yaml:28: union variant "text" missing discriminator const field

Hint: run 'mizu contract validate api.yaml' for detailed validation
```

## Migration Path

1. Users with v1 services continue to use existing `contract ls/call/spec` with URLs
2. Users can create new v2 contracts with `contract init`
3. Both can coexist - URL discovery uses v1 chain, file paths use v2 parser
4. Code generation is v2-only feature

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/cli/contract.go` | Existing contract commands |
| `cmd/cli/contract_v2.go` | New v2 commands (init, validate, gen) |
| `cmd/cli/contract_types.go` | Existing type definitions |
| `cmd/cli/codegen/typescript.go` | TypeScript code generator |
| `cmd/cli/codegen/golang.go` | Go code generator |
| `cmd/cli/templates/contract_*.yaml` | Contract templates |

## Testing

```bash
# Unit tests
go test ./cmd/cli/... -run TestContract

# Integration tests
mizu contract init myapi
mizu contract validate myapi/api.yaml
mizu contract gen myapi/api.yaml --lang typescript
mizu contract spec myapi/api.yaml --format openapi
```
