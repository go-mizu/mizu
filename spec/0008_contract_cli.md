# Spec 0008: Contract CLI

## Summary

Design and implement `mizu contract` command for best-in-class developer experience when working with service contracts. The command enables service discovery, method invocation, and spec export from the command line.

## Design Principles

1. **Discoverable** - Running `mizu contract` shows help with real examples
2. **Minimal typing** - Sensible defaults, positional args for common ops
3. **Progressive disclosure** - Simple cases stay simple
4. **Machine-friendly** - `--json` flag for scripting
5. **Offline-capable** - Works with local files or remote URLs

## Command Structure

```
mizu contract                       # Show help with examples
mizu contract ls [url]              # List services and methods
mizu contract show <method> [url]   # Show method details and schema
mizu contract call <method> [url]   # Call a method
mizu contract spec [url]            # Export OpenAPI/OpenRPC spec
mizu contract types [url]           # List types and schemas
```

## URL Resolution

The URL argument is optional and resolved in order:

1. Explicit positional argument: `mizu contract ls http://api.example.com`
2. Environment variable: `MIZU_URL`
3. Default: `http://localhost:8080`

The URL can point to:
- JSON-RPC endpoint (auto-detected via POST)
- REST endpoint (auto-detected via GET)
- OpenAPI/OpenRPC spec file (local or remote)

## Subcommands

### `mizu contract ls`

List all services and methods.

```bash
# List all methods
$ mizu contract ls
todo
  todo.Create     POST /todos        Create a new todo
  todo.Get        GET  /todos/{id}   Get a todo by ID
  todo.List       GET  /todos        List all todos
  todo.Update     PUT  /todos/{id}   Update a todo
  todo.Delete     DELETE /todos/{id} Delete a todo

user
  user.Create     POST /users        Create a new user
  user.Get        GET  /users/{id}   Get a user by ID

# Filter by service
$ mizu contract ls todo
todo.Create     POST /todos        Create a new todo
todo.Get        GET  /todos/{id}   Get a todo by ID
...

# JSON output for scripting
$ mizu contract ls --json
{"services":[{"name":"todo","methods":[...]}]}
```

**Flags:**
- `--json` - Output as JSON
- `--all` - Include deprecated methods

### `mizu contract show`

Show detailed information about a method.

```bash
$ mizu contract show todo.Create
todo.Create
  POST /todos

  Create a new todo item.

Input (CreateTodoInput):
  title       string    required  The todo title
  completed   boolean             Whether the todo is done

Output (Todo):
  id          string    required  Unique identifier
  title       string    required  The todo title
  completed   boolean   required  Completion status
  createdAt   string    required  ISO 8601 timestamp

# Show JSON schema
$ mizu contract show todo.Create --schema
{
  "input": {
    "type": "object",
    "properties": {
      "title": {"type": "string"},
      "completed": {"type": "boolean"}
    },
    "required": ["title"]
  },
  ...
}
```

**Flags:**
- `--json` - Output as JSON
- `--schema` - Show full JSON schema

### `mizu contract call`

Call a method with input data.

```bash
# Call with JSON input
$ mizu contract call todo.Create '{"title":"Buy milk"}'
{
  "id": "abc123",
  "title": "Buy milk",
  "completed": false,
  "createdAt": "2024-01-15T10:30:00Z"
}

# Read input from file
$ mizu contract call todo.Create @input.json

# Read input from stdin
$ echo '{"title":"Buy milk"}' | mizu contract call todo.Create -

# Call method without input
$ mizu contract call todo.List
[
  {"id": "abc123", "title": "Buy milk", "completed": false}
]

# Call with path parameter (for REST)
$ mizu contract call todo.Get --id abc123
{
  "id": "abc123",
  "title": "Buy milk",
  ...
}
```

**Flags:**
- `--json` - Force JSON output (default for non-TTY)
- `--raw` - Output raw response without formatting
- `--id <value>` - Set path parameter for REST
- `-H, --header <key:value>` - Add request header
- `--timeout <duration>` - Request timeout (default: 30s)

### `mizu contract spec`

Export the API specification.

```bash
# Export OpenAPI 3.1 (default for REST)
$ mizu contract spec > openapi.json

# Export OpenRPC 1.3 (for JSON-RPC)
$ mizu contract spec --format openrpc > openrpc.json

# Pretty print to stdout
$ mizu contract spec --pretty

# Export specific service
$ mizu contract spec --service todo
```

**Flags:**
- `--format <openapi|openrpc|json>` - Output format (auto-detected)
- `--pretty` - Pretty print JSON
- `--service <name>` - Export specific service only
- `-o, --output <file>` - Write to file

### `mizu contract types`

List all registered types and their schemas.

```bash
$ mizu contract types
Todo                 object    4 fields
CreateTodoInput      object    2 fields
UpdateTodoInput      object    2 fields
User                 object    3 fields
Status               enum      3 values

# Show specific type
$ mizu contract types Todo
Todo (object)
  id          string    required
  title       string    required
  completed   boolean   required
  createdAt   string    format: date-time

# JSON schema output
$ mizu contract types Todo --schema
{
  "type": "object",
  "properties": {...},
  "required": ["id", "title", "completed", "createdAt"]
}
```

**Flags:**
- `--json` - Output as JSON
- `--schema` - Show full JSON schema

## Discovery Protocol

The CLI discovers services by:

1. **Try JSON-RPC discovery**: POST to URL with `{"jsonrpc":"2.0","method":"rpc.discover","id":1}`
2. **Try OpenRPC spec**: GET `{url}/openrpc.json`
3. **Try OpenAPI spec**: GET `{url}/openapi.json`
4. **Try custom endpoint**: GET `{url}/_contract`

The discovery response provides:
- List of services and methods
- Input/output schemas
- HTTP hints (method, path)

## Error Handling

Clear, actionable error messages:

```bash
$ mizu contract ls http://localhost:9999
error: cannot connect to http://localhost:9999
  connection refused

hint: is the server running? try: mizu dev

$ mizu contract call todo.Create
error: missing required input
  the todo.Create method requires input

hint: mizu contract call todo.Create '{"title":"..."}'
      mizu contract show todo.Create  # see input schema

$ mizu contract call todo.Createe
error: method not found: todo.Createe

did you mean?
  todo.Create

$ mizu contract call user.Create '{"name":123}'
error: invalid input
  name: expected string, got number

hint: mizu contract show user.Create  # see input schema
```

## Global Flags

These flags work with all subcommands:

- `--url <url>` - Override default URL
- `--json` - JSON output for scripting
- `--no-color` - Disable colored output
- `-q, --quiet` - Suppress non-essential output
- `-v, --verbose` - Show request/response details
- `-h, --help` - Show help

## Environment Variables

- `MIZU_URL` - Default server URL
- `NO_COLOR` - Disable colors (standard)

## Implementation

### File: cli/cmd_contract.go

```go
package cli

// Contract subcommands
var contractCommands = []*command{
    {name: "ls", short: "List services and methods", run: runContractLs},
    {name: "show", short: "Show method details", run: runContractShow},
    {name: "call", short: "Call a method", run: runContractCall},
    {name: "spec", short: "Export API specification", run: runContractSpec},
    {name: "types", short: "List types and schemas", run: runContractTypes},
}
```

### Discovery Types

```go
// ContractInfo represents discovered contract information.
type ContractInfo struct {
    Services []ServiceInfo `json:"services"`
    Types    []TypeInfo    `json:"types"`
}

type ServiceInfo struct {
    Name        string       `json:"name"`
    Description string       `json:"description,omitempty"`
    Version     string       `json:"version,omitempty"`
    Methods     []MethodInfo `json:"methods"`
}

type MethodInfo struct {
    Name        string    `json:"name"`
    FullName    string    `json:"fullName"`
    Description string    `json:"description,omitempty"`
    HTTPMethod  string    `json:"httpMethod,omitempty"`
    HTTPPath    string    `json:"httpPath,omitempty"`
    Deprecated  bool      `json:"deprecated,omitempty"`
    Input       *TypeRef  `json:"input,omitempty"`
    Output      *TypeRef  `json:"output,omitempty"`
}

type TypeRef struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type TypeInfo struct {
    ID     string         `json:"id"`
    Name   string         `json:"name"`
    Schema map[string]any `json:"schema"`
}
```

## Examples in Help

```
mizu contract - Work with service contracts

Usage:
  mizu contract <command> [url] [flags]

Commands:
  ls      List services and methods
  show    Show method details and schema
  call    Call a method with input
  spec    Export API specification
  types   List types and schemas

Examples:
  # List all methods
  mizu contract ls

  # Call a method
  mizu contract call todo.Create '{"title":"Buy milk"}'

  # Export OpenAPI spec
  mizu contract spec > openapi.json

  # Use a different server
  mizu contract ls http://api.example.com
  MIZU_URL=http://api.example.com mizu contract ls

Flags:
      --url string    Server URL (default "http://localhost:8080")
      --json          Output as JSON
      --no-color      Disable colored output
  -v, --verbose       Show request/response details
  -h, --help          Show help
```

## Future Enhancements

1. **Interactive mode**: `mizu contract repl` for interactive exploration
2. **Watch mode**: `mizu contract watch` to monitor method calls
3. **Completion**: Shell completion for method names
4. **History**: Remember recent calls for replay
5. **Mock server**: `mizu contract mock` to run a mock server from spec
